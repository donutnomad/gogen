package gormparse

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sync"

	"github.com/donutnomad/gogen/internal/structparse"
)

// ParseContext gormparse 解析上下文，提供文件级 AST 缓存和结构体解析缓存
type ParseContext struct {
	structCtx *structparse.ParseContext

	mu       sync.Mutex
	astCache map[string]*cachedAST // key: filePath
}

type cachedAST struct {
	fset *token.FileSet
	file *ast.File
}

// NewParseContext 创建 gormparse 解析上下文
func NewParseContext() *ParseContext {
	return &ParseContext{
		structCtx: structparse.NewParseContext(),
		astCache:  make(map[string]*cachedAST),
	}
}

// getOrParseFile 获取或解析文件 AST（带缓存）
func (c *ParseContext) getOrParseFile(filePath string) (*ast.File, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.astCache[filePath]; ok {
		return cached.file, nil
	}

	// 优先从 structparse 缓存获取
	if f := c.structCtx.GetFileAST(filePath); f != nil {
		c.astCache[filePath] = &cachedAST{file: f}
		return f, nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	c.astCache[filePath] = &cachedAST{fset: fset, file: file}
	return file, nil
}

// ParseStruct 带缓存的结构体解析（委托给 structparse.ParseContext）
func (c *ParseContext) ParseStruct(filePath, structName string) (*structparse.StructInfo, error) {
	return c.structCtx.ParseStruct(filePath, structName)
}

// ParseGormModel 带缓存的 GORM 模型解析
func (c *ParseContext) ParseGormModel(filePath, structName string) (*GormModelInfo, error) {
	structInfo, err := c.structCtx.ParseStruct(filePath, structName)
	if err != nil {
		return nil, err
	}
	return c.parseGormModelFromStructInfo(structInfo)
}

// parseGormModelFromStructInfo 从 StructInfo 构建 GormModelInfo，使用缓存的 AST
func (c *ParseContext) parseGormModelFromStructInfo(structInfo *structparse.StructInfo) (*GormModelInfo, error) {
	// 推导表名 —— 使用缓存的 AST
	tableName, err := c.inferTableName(structInfo.FilePath, structInfo.Name)
	if err != nil {
		return nil, err
	}

	// 尝试从 MysqlCreateTable() 方法解析 DDL 获取列类型
	ddlTypes, _ := c.extractSQLTypeFromDDL(structInfo.FilePath, structInfo.Name)

	gormModel := &GormModelInfo{
		Name:        structInfo.Name,
		PackageName: structInfo.PackageName,
		TableName:   tableName,
		Imports:     structInfo.Imports,
	}

	for _, field := range structInfo.Fields {
		gormField := GormFieldInfo{
			Name:           field.Name,
			Type:           field.Type,
			PkgPath:        field.PkgPath,
			PkgAlias:       field.PkgAlias,
			SourceType:     field.SourceType,
			SourceField:    field.SourceField,
			Tag:            field.Tag,
			EmbeddedPrefix: field.EmbeddedPrefix,
		}

		gormField.ColumnName = ExtractColumnNameWithPrefix(field.Name, field.Tag, field.EmbeddedPrefix)

		gormField.SQLType = ExtractSQLType(field.Tag)
		if gormField.SQLType == "" && ddlTypes != nil {
			if sqlType, exists := ddlTypes[gormField.ColumnName]; exists {
				gormField.SQLType = sqlType
			}
		}

		gormField.GormDataType = InferGormDataType(field.Type, field.Tag)

		gormModel.Fields = append(gormModel.Fields, gormField)
	}

	return gormModel, nil
}

// inferTableName 使用缓存 AST 推导表名
func (c *ParseContext) inferTableName(filePath, structName string) (string, error) {
	node, err := c.getOrParseFile(filePath)
	if err != nil {
		// 回退到无缓存方式
		return InferTableName(filePath, structName)
	}

	tableName := extractTableNameFromNode(node, structName)
	if tableName != "" {
		return tableName, nil
	}
	return toSnakeCase(structName) + "s", nil
}

// extractSQLTypeFromDDL 使用缓存 AST 提取 DDL 列类型
func (c *ParseContext) extractSQLTypeFromDDL(filePath, structName string) (map[string]string, error) {
	node, err := c.getOrParseFile(filePath)
	if err != nil {
		return ExtractSQLTypeFromDDL(filePath, structName)
	}

	return extractDDLFromNode(node, structName), nil
}

// extractTableNameFromNode 从 AST 节点提取 TableName
func extractTableNameFromNode(node *ast.File, structName string) string {
	for n := range ast.Preorder(node) {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != "TableName" || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}
		if getRecvTypeName(funcDecl) != structName || funcDecl.Body == nil {
			continue
		}
		if s := extractFirstReturnString(funcDecl); s != "" {
			return s
		}
	}
	return ""
}

// extractDDLFromNode 从 AST 节点提取 MysqlCreateTable DDL
func extractDDLFromNode(node *ast.File, structName string) map[string]string {
	for n := range ast.Preorder(node) {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != "MysqlCreateTable" || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}
		if getRecvTypeName(funcDecl) != structName || funcDecl.Body == nil {
			continue
		}
		if s := extractFirstReturnString(funcDecl); s != "" {
			return ParseDDLColumnTypes(s)
		}
	}
	return nil
}

// getRecvTypeName 获取方法接收器类型名（去除指针）
func getRecvTypeName(funcDecl *ast.FuncDecl) string {
	switch t := funcDecl.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// extractFirstReturnString 提取方法体中第一个 return 的字符串字面量
func extractFirstReturnString(funcDecl *ast.FuncDecl) string {
	for _, stmt := range funcDecl.Body.List {
		retStmt, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(retStmt.Results) == 0 {
			continue
		}
		lit, ok := retStmt.Results[0].(*ast.BasicLit)
		if !ok {
			continue
		}
		return trimQuotes(lit.Value)
	}
	return ""
}

// trimQuotes 去除字符串字面量的引号
func trimQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '`' && s[len(s)-1] == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

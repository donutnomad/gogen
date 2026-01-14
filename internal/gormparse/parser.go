package gormparse

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"github.com/donutnomad/gogen/internal/structparse"
	"github.com/donutnomad/gogen/internal/utils"
)

// GormFieldInfo GORM字段信息
type GormFieldInfo struct {
	Name           string // 字段名
	Type           string // 字段类型
	PkgPath        string // 类型所在包路径
	PkgAlias       string // 包在源文件中的别名（如果有）
	ColumnName     string // 数据库列名
	IsEmbedded     bool   // 是否为嵌入字段
	SourceType     string // 字段来源类型,为空表示来自结构体本身,否则表示来自嵌入的结构体
	Tag            string // 字段标签
	EmbeddedPrefix string // gorm embedded 字段的 prefix
}

// GormModelInfo GORM模型信息
type GormModelInfo struct {
	Name        string          // 结构体名称
	PackageName string          // 包名
	TableName   string          // 表名
	Prefix      string          // 生成的结构体前缀
	Fields      []GormFieldInfo // 字段列表
	Imports     []string        // 导入的包
}

// ExtractColumnName 提取列名(从gorm标签或使用默认规则)
func ExtractColumnName(fieldName, fieldTag string) string {
	return ExtractColumnNameWithPrefix(fieldName, fieldTag, "")
}

// ExtractColumnNameWithPrefix 提取列名，支持 embeddedPrefix
func ExtractColumnNameWithPrefix(fieldName, fieldTag, embeddedPrefix string) string {
	var columnName string

	if fieldTag == "" {
		columnName = toSnakeCase(fieldName)
	} else {
		// 解析GORM标签
		gormTags := parseGormTag(fieldTag)
		if col, exists := gormTags["column"]; exists {
			columnName = col
		} else {
			// 没有找到column标签,使用默认规则
			columnName = toSnakeCase(fieldName)
		}
	}

	// 应用 embeddedPrefix
	if embeddedPrefix != "" {
		columnName = embeddedPrefix + columnName
	}

	return columnName
}

// ParseGormModel 解析GORM模型
func ParseGormModel(structInfo *structparse.StructInfo) (*GormModelInfo, error) {
	// 推导表名
	tableName, err := InferTableName(structInfo.FilePath, structInfo.Name)
	if err != nil {
		return nil, err
	}

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
			PkgPath:        field.PkgPath,        // 复制包路径
			PkgAlias:       field.PkgAlias,       // 复制包别名
			SourceType:     field.SourceType,     // 复制来源信息
			Tag:            field.Tag,            // 保存标签信息
			EmbeddedPrefix: field.EmbeddedPrefix, // 复制 embeddedPrefix
		}

		// 解析列名（使用 embeddedPrefix）
		gormField.ColumnName = ExtractColumnNameWithPrefix(field.Name, field.Tag, field.EmbeddedPrefix)

		gormModel.Fields = append(gormModel.Fields, gormField)
	}

	return gormModel, nil
}

func toSnakeCase(name string) string {
	return utils.ToSnakeCase(name)
}

// parseGormTag 解析GORM标签
func parseGormTag(tag string) map[string]string {
	result := make(map[string]string)

	// 提取gorm标签内容
	re := regexp.MustCompile(`gorm:"([^"]*)"`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) < 2 {
		return result
	}

	gormTag := matches[1]

	// 解析标签内的各个部分
	for part := range strings.SplitSeq(gormTag, ";") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 {
				result[kv[0]] = kv[1]
			}
		} else {
			result[part] = ""
		}
	}

	return result
}

// InferTableName 推导表名
// 首先尝试从 TableName() 方法中提取表名
// 如果没有找到，使用默认规则: 结构体名的蛇形命名 + "s"
func InferTableName(filename, structName string) (string, error) {
	// 首先尝试查找TableName方法
	tableName, err := ExtractTableNameFromMethod(filename, structName)
	if err == nil && tableName != "" {
		return tableName, nil
	}

	// 如果没有TableName方法,使用默认规则: 结构体名的复数形式 + 蛇形命名
	return toSnakeCase(structName) + "s", nil
}

// ExtractTableNameFromMethod 从 TableName() 方法中提取表名
// 解析 AST 查找指定结构体的 TableName 方法，并提取其返回的字符串字面量
func ExtractTableNameFromMethod(filename, structName string) (string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return "", err
	}

	var tableName string
OUT:
	for n := range ast.Preorder(node) {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if funcDecl.Name.Name != "TableName" || funcDecl.Recv == nil {
			continue
		}
		if len(funcDecl.Recv.List) == 0 {
			continue
		}

		recvType := ""
		switch t := funcDecl.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				recvType = ident.Name
			}
		case *ast.Ident:
			recvType = t.Name
		}
		if recvType != structName {
			continue
		}
		if funcDecl.Body == nil {
			continue
		}

		// 提取返回值
		for _, stmt := range funcDecl.Body.List {
			retStmt, ok := stmt.(*ast.ReturnStmt)
			if !ok {
				continue
			}
			if len(retStmt.Results) == 0 {
				continue
			}
			lit, ok := retStmt.Results[0].(*ast.BasicLit)
			if !ok {
				continue
			}
			tableName = strings.Trim(lit.Value, `"`)
			break OUT
		}
	}

	return tableName, nil
}

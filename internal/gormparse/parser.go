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
	SQLType        string // SQL类型（如 datetime, date, time），从 gorm 标签的 type:xxx 解析
	GormDataType   string // GORM 数据类型，从类型的 GormDataType() 方法返回值解析（如 json）
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

// ExtractSQLType 从 gorm 标签中提取 SQL 类型
// 例如: gorm:"type:datetime(3)" -> "datetime"
// 例如: gorm:"type:date" -> "date"
// 例如: gorm:"type:time" -> "time"
func ExtractSQLType(fieldTag string) string {
	if fieldTag == "" {
		return ""
	}

	gormTags := parseGormTag(fieldTag)
	sqlType, exists := gormTags["type"]
	if !exists || sqlType == "" {
		return ""
	}

	// 标准化 SQL 类型：提取基本类型名（去除括号和参数）
	// 例如: "datetime(3)" -> "datetime", "varchar(255)" -> "varchar"
	sqlType = strings.ToLower(sqlType)
	if idx := strings.Index(sqlType, "("); idx != -1 {
		sqlType = sqlType[:idx]
	}

	return sqlType
}

// InferGormDataType 根据字段类型推断 GormDataType
// 通过检查类型的 GormDataType() 方法返回值来判断
// 目前支持的类型：
//   - datatypes.JSON, datatypes.JSONType[T], datatypes.JSONSlice[T] -> "json"
//   - serializer:json 标签 -> "json"
func InferGormDataType(fieldType, fieldTag string) string {
	// 首先检查 gorm 标签中的 serializer:json
	if fieldTag != "" {
		gormTags := parseGormTag(fieldTag)
		if serializer, exists := gormTags["serializer"]; exists && serializer == "json" {
			return "json"
		}
	}

	// 移除指针前缀
	cleanType := strings.TrimPrefix(fieldType, "*")

	// 检查已知的 GORM JSON 类型
	// 这些类型的 GormDataType() 方法返回 "json"
	// 注意: 泛型类型如 datatypes.JSONType[T] 中的 T 可能是本地类型名
	jsonTypePatterns := []string{
		"datatypes.JSON",
		"datatypes.JSONType[",
		"datatypes.JSONSlice[",
		"datatypes.JSONMap[",
	}

	for _, pattern := range jsonTypePatterns {
		if cleanType == pattern || strings.HasPrefix(cleanType, pattern) {
			return "json"
		}
	}

	return ""
}

// ParseGormModel 解析GORM模型
func ParseGormModel(structInfo *structparse.StructInfo) (*GormModelInfo, error) {
	// 推导表名
	tableName, err := InferTableName(structInfo.FilePath, structInfo.Name)
	if err != nil {
		return nil, err
	}

	// 尝试从 MysqlCreateTable() 方法解析 DDL 获取列类型
	ddlTypes, _ := ExtractSQLTypeFromDDL(structInfo.FilePath, structInfo.Name)

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

		// 解析 SQL 类型：优先从 gorm 标签，其次从 DDL
		gormField.SQLType = ExtractSQLType(field.Tag)
		if gormField.SQLType == "" && ddlTypes != nil {
			// 从 DDL 中查找该列的类型
			if sqlType, exists := ddlTypes[gormField.ColumnName]; exists {
				gormField.SQLType = sqlType
			}
		}

		// 推断 GormDataType（用于检测 JSON 等特殊类型）
		gormField.GormDataType = InferGormDataType(field.Type, field.Tag)

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

// ExtractSQLTypeFromDDL 从 MysqlCreateTable() 方法提取列的 SQL 类型
// 解析 AST 查找指定结构体的 MysqlCreateTable 方法，提取返回的 DDL 字符串
// 然后解析 CREATE TABLE 语句，返回 map[columnName]sqlType
func ExtractSQLTypeFromDDL(filename, structName string) (map[string]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var ddl string
OUT:
	for n := range ast.Preorder(node) {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if funcDecl.Name.Name != "MysqlCreateTable" || funcDecl.Recv == nil {
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

		// 提取返回值（字符串字面量或反引号字符串）
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
			// 去除引号（双引号或反引号）
			ddl = strings.Trim(lit.Value, "`\"")
			break OUT
		}
	}

	if ddl == "" {
		return nil, nil
	}

	return ParseDDLColumnTypes(ddl), nil
}

// ParseDDLColumnTypes 从 CREATE TABLE DDL 语句中解析列类型
// 返回 map[columnName]sqlType
// 例如: `id` BIGINT UNSIGNED -> map["id"] = "bigint"
// 例如: `created_at` DATETIME(3) -> map["created_at"] = "datetime"
func ParseDDLColumnTypes(ddl string) map[string]string {
	result := make(map[string]string)

	// 正则匹配列定义：`column_name` TYPE(...)
	// 支持反引号或单引号包围的列名
	columnDefPattern := regexp.MustCompile("(?i)[`']?(\\w+)[`']?\\s+(\\w+)")

	// 需要跳过的关键字（不区分大小写）
	// 这些关键字后面必须跟空格或直接是行尾，确保是完整单词匹配
	skipKeywords := []string{
		"CREATE", "PRIMARY", "INDEX", "UNIQUE", "KEY",
		"CONSTRAINT", "FOREIGN", "ENGINE", "CHARSET", "COLLATE",
	}

	// 先按换行分割，再按逗号分割（支持单行 DDL）
	var parts []string
	lines := strings.Split(ddl, "\n")
	for _, line := range lines {
		// 按逗号分割每行
		subParts := strings.Split(line, ",")
		parts = append(parts, subParts...)
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(part, ")") {
			continue
		}

		// 检查是否以跳过关键字开头（必须是完整单词）
		upperPart := strings.ToUpper(part)
		skip := false
		for _, keyword := range skipKeywords {
			// 检查是否以关键字开头，且关键字后面是空格或到达结尾
			if strings.HasPrefix(upperPart, keyword) {
				// 检查关键字后面是否是空格（完整单词）
				if len(upperPart) == len(keyword) || upperPart[len(keyword)] == ' ' {
					skip = true
					break
				}
			}
		}
		if skip {
			continue
		}

		matches := columnDefPattern.FindStringSubmatch(part)
		if len(matches) >= 3 {
			columnName := strings.ToLower(matches[1])
			sqlType := strings.ToLower(matches[2])
			result[columnName] = sqlType
		}
	}

	return result
}

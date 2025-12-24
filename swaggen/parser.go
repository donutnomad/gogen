package swaggen

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"github.com/donutnomad/gogen/internal/xast"
	parsers "github.com/donutnomad/gogen/swaggen/parser"
	"github.com/samber/lo"
)

// ============================================================================
// 类型解析器
// ============================================================================

// NewReturnTypeParser 创建返回类型解析器
func NewReturnTypeParser(imports xast.ImportInfoSlice) *ReturnTypeParser {
	return &ReturnTypeParser{
		imports: imports,
	}
}

// ParseReturnType 解析返回类型
func (p *ReturnTypeParser) ParseReturnType(expr ast.Expr) TypeInfo {
	return p.parseType(expr)
}

// ParseParameterType 解析参数类型
func (p *ReturnTypeParser) ParseParameterType(expr ast.Expr) TypeInfo {
	return p.parseType(expr)
}

// parseType 解析类型表达式
func (p *ReturnTypeParser) parseType(expr ast.Expr) TypeInfo {
	typeStr := xast.GetFieldType(expr, nil)
	return p.parseTypeString(typeStr, expr)
}

// parseTypeString 将类型字符串解析为 TypeInfo
func (p *ReturnTypeParser) parseTypeString(typeStr string, expr ast.Expr) TypeInfo {
	info := TypeInfo{
		FullName: typeStr,
		TypeName: typeStr,
	}

	if strings.HasPrefix(typeStr, "*") {
		info.IsPointer = true
	}

	if strings.HasPrefix(typeStr, "[]") || strings.HasPrefix(typeStr, "*[]") {
		info.IsSlice = true
	}

	if bracketIdx := strings.Index(typeStr, "["); bracketIdx != -1 && !strings.HasPrefix(typeStr, "[]") && !strings.HasPrefix(typeStr, "map[") {
		info.IsGeneric = true
		info.GenericArgs = p.parseGenericArgs(expr)
	}

	p.extractPackageInfo(&info, expr)

	return info
}

// extractPackageInfo 从 AST 表达式中提取包信息
func (p *ReturnTypeParser) extractPackageInfo(info *TypeInfo, expr ast.Expr) {
	switch t := expr.(type) {
	case *ast.Ident:
		info.TypeName = t.Name
	case *ast.SelectorExpr:
		info.TypeName = t.Sel.Name
		if ident, ok := t.X.(*ast.Ident); ok {
			info.Alias = ident.Name
			if imp := p.imports.Find(ident.Name); imp != nil {
				info.Package = imp.GetPath()
			}
		}
	case *ast.StarExpr:
		p.extractPackageInfo(info, t.X)
	case *ast.ArrayType:
		p.extractPackageInfo(info, t.Elt)
	case *ast.IndexExpr:
		p.extractPackageInfo(info, t.X)
	case *ast.IndexListExpr:
		p.extractPackageInfo(info, t.X)
	case *ast.MapType:
		// map 类型不需要特殊处理包信息
	}
}

// parseGenericArgs 解析泛型参数
func (p *ReturnTypeParser) parseGenericArgs(expr ast.Expr) []TypeInfo {
	var args []TypeInfo

	switch t := expr.(type) {
	case *ast.IndexExpr:
		args = append(args, p.parseType(t.Index))
	case *ast.IndexListExpr:
		for _, index := range t.Indices {
			args = append(args, p.parseType(index))
		}
	case *ast.StarExpr:
		return p.parseGenericArgs(t.X)
	case *ast.ArrayType:
		return p.parseGenericArgs(t.Elt)
	}

	return args
}

// GetSwaggerType 获取 Swagger 类型字符串
func (info *TypeInfo) GetSwaggerType() string {
	if info.IsSlice {
		return "array"
	}

	typeName := info.TypeName
	if info.IsPointer {
		typeName = strings.TrimPrefix(typeName, "*")
	}

	switch typeName {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "int64":
		return "integer"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "object"
	}
}

// GetSwaggerFormat 获取 Swagger 格式字符串
func (info *TypeInfo) GetSwaggerFormat() string {
	typeName := info.TypeName
	if info.IsPointer {
		typeName = strings.TrimPrefix(typeName, "*")
	}

	switch typeName {
	case "int32":
		return "integer"
	case "int64":
		return "integer"
	case "float32":
		return "number"
	case "float64":
		return "number"
	default:
		return ""
	}
}

// ============================================================================
// 注释解析器
// ============================================================================

// NewAnnotationParser 创建注释解析器
func NewAnnotationParser(fileSet *token.FileSet) *AnnotationParser {
	tagsParser, err := newTagParserSafe()
	if err != nil {
		panic(err)
	}
	return &AnnotationParser{
		fileSet:    fileSet,
		tagsParser: tagsParser,
	}
}

// ParseMethodAnnotations 解析方法注释
func (p *AnnotationParser) ParseMethodAnnotations(method *ast.FuncDecl) (*SwaggerMethod, error) {
	if method.Doc == nil {
		return nil, nil
	}

	swaggerMethod := &SwaggerMethod{
		Name: method.Name.Name,
	}

	var summaryLines []string
	var descriptionLines []string
	var isDescription = false

	for _, comment := range method.Doc.List {
		line := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))

		if strings.HasPrefix(line, "@") {
			isDescription = false
			parse, err := p.tagsParser.Parse(line)
			if err != nil {
				return nil, NewParseError("method comment parsing failed",
					fmt.Sprintf("failed to parse comment '%s' in method %s", line, swaggerMethod.Name), err)
			}
			swaggerMethod.Def = append(swaggerMethod.Def, parse.(parsers.Definition))
		} else if line != "" {
			if len(summaryLines) == 0 {
				summaryLines = append(summaryLines, line)
				isDescription = true
			} else {
				if isDescription {
					descriptionLines = append(descriptionLines, line)
				}
			}
		}
	}

	if len(summaryLines) > 0 {
		swaggerMethod.Summary = strings.TrimSpace(strings.TrimPrefix(strings.Join(summaryLines, " "), swaggerMethod.Name))
	}
	if len(descriptionLines) > 0 {
		swaggerMethod.Description = strings.Join(descriptionLines, "\n")
	}

	if len(swaggerMethod.GetPaths()) == 0 {
		return nil, nil
	}

	return swaggerMethod, nil
}

// ParseParameterAnnotations 解析参数注释
func (p *AnnotationParser) ParseParameterAnnotations(paramName string, tag string) Parameter {
	param := Parameter{
		Name:     paramName,
		Required: true,
	}
	line := tag

	switch {
	case strings.HasPrefix(line, "@PARAM"):
		param.Source = "path"
		if aliasRegex := regexp.MustCompile(`@PARAM\s*\(([^)]+)\)`); aliasRegex.MatchString(line) {
			matches := aliasRegex.FindStringSubmatch(line)
			if len(matches) == 2 {
				param.Alias = matches[1]
			}
		}
	case strings.HasPrefix(line, "@HEADER"):
		param.Source = "header"
	}

	return param
}

// extractPathParameters 从路径中提取参数
func (p *AnnotationParser) extractPathParameters(path string) []Parameter {
	var parameters []Parameter

	pathParamRegex := regexp.MustCompile(`\{([^}]+)\}`)
	matches := pathParamRegex.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		if len(match) == 2 {
			paramName := match[1]
			param := Parameter{
				Name:     paramName,
				Source:   "path",
				Required: true,
			}
			parameters = append(parameters, param)
		}
	}

	return parameters
}

// ParseCommonAnnotation 解析接口级别的通用注释
func (p *AnnotationParser) ParseCommonAnnotation(line string) *CommonAnnotation {
	re := regexp.MustCompile(`\(([^)]+)\)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil
	}

	content := matches[1]
	parts := strings.SplitN(content, ";", 2)
	value := strings.TrimSpace(parts[0])

	var excludes []string
	if len(parts) > 1 {
		excludePart := strings.TrimSpace(parts[1])
		if strings.HasPrefix(excludePart, "exclude=") {
			excludeStr := strings.TrimPrefix(excludePart, "exclude=")
			excludeStr = strings.Trim(excludeStr, `"`)
			for _, item := range strings.Split(excludeStr, ",") {
				excludes = append(excludes, strings.TrimSpace(item))
			}
		}
	}

	return &CommonAnnotation{
		Value:   value,
		Exclude: excludes,
	}
}

// ============================================================================
// 接口解析器
// ============================================================================

// getContent 从文件的指定位置提取内容
func getContent(fileContent []byte, start, end token.Position) string {
	reader := bufio.NewScanner(bytes.NewReader(fileContent))
	var lineNum int
	var sb strings.Builder

	for reader.Scan() {
		lineNum++
		line := reader.Text()

		if lineNum == start.Line {
			var endColumn = len(line)
			if lineNum == end.Line {
				endColumn = end.Column
			}
			sb.WriteString(line[start.Column-1 : endColumn])
		}

		if start.Line != end.Line && lineNum > start.Line && lineNum <= end.Line {
			if lineNum == end.Line {
				sb.WriteString(line[:end.Column])
			} else {
				sb.WriteString(line)
			}
		}

		if lineNum > end.Line {
			break
		}
	}

	return sb.String()
}

// FilterInterfacesByName 按名称过滤接口
func (collection *InterfaceCollection) FilterInterfacesByName(names []string) *InterfaceCollection {
	if len(names) == 0 {
		return collection
	}

	filtered := lo.Filter(collection.Interfaces, func(iface SwaggerInterface, _ int) bool {
		return lo.Contains(names, iface.Name)
	})

	return &InterfaceCollection{
		Interfaces: filtered,
	}
}

// GetAllMethods 获取所有方法
func (collection *InterfaceCollection) GetAllMethods() []SwaggerMethod {
	var allMethods []SwaggerMethod

	for _, iface := range collection.Interfaces {
		allMethods = append(allMethods, iface.Methods...)
	}

	return allMethods
}

package swaggen

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/donutnomad/gogen/internal/utils"
	parsers "github.com/donutnomad/gogen/swaggen/parser"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

// ============================================================================
// 标签解析器
// ============================================================================

// newTagParserSafe 创建标签解析器
func newTagParserSafe() (*parsers.Parser, error) {
	parser := parsers.NewParser()
	err := parser.Register(
		parsers.Tag{},
		parsers.GET{},
		parsers.POST{},
		parsers.PUT{},
		parsers.PATCH{},
		parsers.DELETE{},

		parsers.Security{},
		parsers.Header{},
		parsers.MiddleWare{},

		parsers.JsonReq{},
		parsers.FormReq{},
		parsers.MimeReq{},

		parsers.JSON{},
		parsers.MIME{},

		parsers.FORM{},
		parsers.BODY{},
		parsers.PARAM{},
		parsers.QUERY{},

		parsers.Removed{},
		parsers.ExcludeFromBindAll{},
		parsers.Raw{},
		parsers.Prefix{},
	)

	return parser, err
}

// ============================================================================
// Swagger 生成器
// ============================================================================

// GenerateSwaggerComments 生成 Swagger 注释
func (g *SwaggerGenerator) GenerateSwaggerComments() map[string]string {
	var out = make(map[string]string)
	for _, iface := range g.collection.Interfaces {
		for _, method := range iface.Methods {
			if method.Def.IsRemoved() {
				continue
			}
			methodComments := g.generateMethodComments(method, iface)
			methodKey := fmt.Sprintf("%s.%s", iface.Name, method.Name)
			out[methodKey] = strings.Join(methodComments, "\n")
		}
	}
	return out
}

func mergeDefs[T any](ifaceDefs, methodDefs []parsers.Definition, f func(item parsers.Definition) (T, bool), post func([]T)) {
	var methodTags []T
	for _, item := range methodDefs {
		if v, ok := f(item); ok {
			methodTags = append(methodTags, v)
		}
	}
	if len(methodTags) == 0 {
		for _, item := range ifaceDefs {
			if v, ok := f(item); ok {
				methodTags = append(methodTags, v)
			}
		}
	}
	post(methodTags)
}

// generateMethodComments 生成单个方法的 Swagger 注释
func (g *SwaggerGenerator) generateMethodComments(method SwaggerMethod, iface SwaggerInterface) []string {
	var lines []string

	lines = append(lines, fmt.Sprintf("// %s", method.Name))

	if method.Summary != "" {
		lines = append(lines, fmt.Sprintf("// @Summary %s", method.Summary))
	} else {
		lines = append(lines, fmt.Sprintf("// @Summary %s", method.Name))
	}

	if method.Description != "" {
		for _, desc := range strings.Split(method.Description, "\n") {
			lines = append(lines, fmt.Sprintf("// @Description %s", desc))
		}
	}

	for _, md := range CollectDef[*parsers.MiddleWare](method.Def) {
		for _, name := range md.Value {
			if idx := strings.Index(name, "_"); idx > 0 {
				prefix := name[:idx]
				if idx != len(name)-1 {
					lines = append(lines, fmt.Sprintf("// @Description %s: %s", prefix, name[idx+1:]))
				}
			}
		}
	}

	mergeDefs[string](iface.CommonDef, method.Def, func(item parsers.Definition) (string, bool) {
		v, ok := item.(*parsers.Tag)
		if !ok {
			return "", false
		}
		return v.Value, true
	}, func(i []string) {
		if len(i) > 0 {
			lines = append(lines, fmt.Sprintf("// @Tags %s", strings.Join(i, ",")))
		}
	})

	if method.GetHTTPMethod() != "GET" {
		mergeDefs[string](iface.CommonDef, method.Def, func(item parsers.Definition) (string, bool) {
			return DefSlice{item}.GetAcceptType()
		}, func(i []string) {
			var ret = "json"
			if len(i) > 0 {
				ret = i[0]
			}
			lines = append(lines, fmt.Sprintf("// @Accept %s", ret))
		})
	}

	mergeDefs[string](iface.CommonDef, method.Def, func(item parsers.Definition) (string, bool) {
		return DefSlice{item}.GetContentType()
	}, func(i []string) {
		var ret = "json"
		if len(i) > 0 {
			ret = i[0]
		}
		lines = append(lines, fmt.Sprintf("// @Produce %s", ret))
	})

	mergeDefs[string](iface.CommonDef, method.Def, func(item parsers.Definition) (string, bool) {
		v, ok := item.(*parsers.Security)
		if !ok {
			return "", false
		}
		ok = false
		if len(v.Include) > 0 {
			if lo.Contains(v.Include, method.Name) {
				ok = true
			}
		} else if len(v.Exclude) > 0 {
			if !lo.Contains(v.Include, method.Name) {
				ok = true
			}
		} else {
			ok = true
		}
		return v.Value, ok
	}, func(i []string) {
		if len(i) > 0 {
			lines = append(lines, fmt.Sprintf("// @Security %s", strings.Join(i, ",")))
		}
	})

	paramLines := g.generateParameterComments(method, method.Parameters, iface.CommonDef, method.Def)
	lines = append(lines, paramLines...)

	for _, md := range CollectDef[*parsers.Raw](method.Def) {
		lines = append(lines, fmt.Sprintf("// %s", md.Value))
	}

	successLine := g.generateSuccessComment(method.ResponseType)
	lines = append(lines, successLine)

	prefix := iface.CommonDef.GetPrefix()

	for _, pathRouter := range method.GetPaths() {
		lines = append(lines, fmt.Sprintf("// @Router %s [%s]", prefix+pathRouter, strings.ToLower(method.GetHTTPMethod())))
	}

	return lines
}

// generateParameterComments 生成参数注释
func (g *SwaggerGenerator) generateParameterComments(method SwaggerMethod, parameters []Parameter, ifaceDef, def DefSlice) []string {
	var lines []string

	for i, param := range parameters {
		if param.Type.FullName == GinContextType || param.Type.TypeName == "Context" {
			continue
		}
		if param.Source == "path" {
		} else if param.Source == "header" {
		} else if i == len(parameters)-1 {
			if method.GetHTTPMethod() == "GET" {
				param.Source = "query"
			} else if v, _ := slices.Concat(def, ifaceDef).GetAcceptType(); v == "json" {
				param.Source = "body"
			} else {
				param.Source = "formData"
			}
		}

		paramLine := g.generateParameterComment(param)
		lines = append(lines, paramLine)
	}

	var allSlice = slices.Concat(ifaceDef, def)
	var headerMap = make(map[string]*parsers.Header)
	var headerNames []string
	for _, param := range allSlice {
		if v, ok := param.(*parsers.Header); ok {
			_, exists := headerMap[v.Value]
			headerMap[v.Value] = v
			if !exists {
				headerNames = append(headerNames, v.Value)
			}
		}
	}
	for _, key := range headerNames {
		value := headerMap[key]
		headerLine := fmt.Sprintf("// @Param %s header string %s \"%s\"", key, lo.Ternary(value.Required, "true", "false"), lo.Ternary(len(value.Description) > 0, value.Description, key))
		lines = append(lines, headerLine)
	}

	return lines
}

// generateParameterComment 生成单个参数注释
func (g *SwaggerGenerator) generateParameterComment(param Parameter) string {
	paramType := g.getParameterType(param)
	required := lo.Ternary(param.Required, "true", "false")
	description := lo.Ternary(param.Comment == "", param.Name, param.Comment)

	if param.Source == "body" {
		return fmt.Sprintf("// @Param %s body %s %s \"%s\"", param.Name, param.Type.FullName, required, description)
	}

	n := lo.Ternary(len(param.PathName) > 0, param.PathName, param.Name)
	return fmt.Sprintf("// @Param %s %s %s %s \"%s\"", n, param.Source, paramType, required, description)
}

// generateSuccessComment 生成成功响应注释
func (g *SwaggerGenerator) generateSuccessComment(responseType TypeInfo) string {
	if responseType.FullName == "" {
		return "// @Success 200 {string} string \"success\""
	}

	responseTypeStr := responseType.FullName
	if responseTypeStr == "" {
		responseTypeStr = "string"
	}
	responseTypeStr = strings.ReplaceAll(responseTypeStr, "*", "")

	return fmt.Sprintf("// @Success 200 {object} %s", responseTypeStr)
}

// getParameterType 获取参数类型字符串
func (g *SwaggerGenerator) getParameterType(param Parameter) string {
	typeInfo := param.Type

	switch typeInfo.GetSwaggerType() {
	case "string":
		return "string"
	case "integer":
		format := typeInfo.GetSwaggerFormat()
		if format != "" {
			return format
		}
		return "integer"
	case "number":
		format := typeInfo.GetSwaggerFormat()
		if format != "" {
			return format
		}
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "array"
	default:
		if typeInfo.FullName != "" {
			return typeInfo.FullName
		}
		return "string"
	}
}

// GenerateFileHeader 生成文件头部
func (g *SwaggerGenerator) GenerateFileHeader(packageName string) string {
	var lines []string

	lines = append(lines, "// Code generated by swagGen. DO NOT EDIT.")
	lines = append(lines, "//")
	lines = append(lines, "// This file contains Swagger documentation and Gin binding code.")
	lines = append(lines, "// Generated from interface definitions with Swagger annotations.")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("package %s", packageName))
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// GenerateImports 生成导入声明
func (g *SwaggerGenerator) GenerateImports() string {
	var imports []string
	imports = append(imports, `	"strings"`)
	imports = append(imports, "")
	imports = append(imports, `	"github.com/gin-gonic/gin"`)

	if g.needsCastImport() {
		imports = append(imports, `	"github.com/spf13/cast"`)
	}

	return "import (\n" + strings.Join(imports, "\n") + "\n)"
}

// needsCastImport 检查是否需要 cast 导入
func (g *SwaggerGenerator) needsCastImport() bool {
	for _, iface := range g.collection.Interfaces {
		for _, method := range iface.Methods {
			for _, param := range method.Parameters {
				if param.Source == "path" || param.Source == "query" {
					typeName := param.Type.TypeName
					switch typeName {
					case "int", "int8", "int16", "int32", "int64",
						"uint", "uint8", "uint16", "uint32", "uint64",
						"float32", "float64", "bool":
						return true
					}
				}
			}
		}
	}
	return false
}

// GenerateTypeReferences 生成类型引用 - 已禁用
func (g *SwaggerGenerator) GenerateTypeReferences() string {
	return ""
}

// ============================================================================
// Gin 生成器
// ============================================================================

// convertPathToGinFormat 将 Swagger 路径格式 {param} 转换为 Gin 路径格式 :param
func convertPathToGinFormat(path string) string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	return re.ReplaceAllString(path, ":$1")
}

// NewGinGenerator 创建 Gin 生成器
func NewGinGenerator(collection *InterfaceCollection) *GinGenerator {
	return &GinGenerator{
		collection: collection,
	}
}

// GenerateGinCode 生成 Gin 绑定代码
func (g *GinGenerator) GenerateGinCode(comments map[string]string) (constructCode, code string) {
	var parts []string
	var constructorParts []string

	var handlerInterface []string

	for _, iface := range g.collection.Interfaces {
		var middlewareCount int
		var middlewareMap = make(map[string][]*parsers.MiddleWare)
		var handlerItfName = fmt.Sprintf("%sHandler", iface.Name)

		for _, method := range iface.Methods {
			middlewareMap[method.Name] = CollectDef[*parsers.MiddleWare](iface.CommonDef, method.Def)
			middlewareCount += len(middlewareMap[method.Name])
		}

		constructor, wrapperCode := g.generateWrapperStruct(iface, handlerItfName)
		parts = append(parts, wrapperCode)

		bindMethodCode := g.generateBindMethod(iface)
		parts = append(parts, bindMethodCode)
		parts = append(parts, "")

		for _, method := range iface.Methods {
			methodKey := fmt.Sprintf("%s.%s", iface.Name, method.Name)
			if v, ok := comments[methodKey]; ok {
				parts = append(parts, v)
			}
			handlerCode := g.generateHandlerMethod(iface, method)
			parts = append(parts, handlerCode)
			parts = append(parts, "")
		}

		for _, method := range iface.Methods {
			methodCode := g.generateMethodBinding(iface, method, middlewareMap[method.Name])
			parts = append(parts, methodCode)
			parts = append(parts, "")
		}

		template := fmt.Sprintf("func (a *%s) BindAll(router gin.IRoutes, preHandlers ...gin.HandlerFunc) {", iface.GetWrapperName())
		parts = append(parts, template)
		for _, method := range iface.Methods {
			if method.Def.IsRemoved() || method.Def.IsExcludeFromBindAll() {
				continue
			}
			parts = append(parts, fmt.Sprintf("	a.%s(router, preHandlers...)", fmt.Sprintf("Bind%s", method.Name)))
		}
		parts = append(parts, "}")
		parts = append(parts, "")

		if len(middlewareMap) > 0 {
			var items = lo.Uniq(lo.Flatten(lo.Map(lo.Flatten(maps.Values(middlewareMap)), func(item *parsers.MiddleWare, index int) []string {
				return item.Value
			})))
			sort.Strings(items)

			handlerInterface = append(handlerInterface, fmt.Sprintf("type %s interface {", handlerItfName))
			handlerInterface = append(handlerInterface, fmt.Sprintf("%s() []gin.HandlerFunc", "PreHandlers"))
			for _, key := range items {
				handlerInterface = append(handlerInterface, fmt.Sprintf("%s() []gin.HandlerFunc", key))
			}
			handlerInterface = append(handlerInterface, "}")
			handlerInterface = append(handlerInterface, "\n")
		}

		constructorParts = append(constructorParts, constructor)
	}

	return strings.Join(constructorParts, "\n\n"), strings.Join(slices.Concat(handlerInterface, parts), "\n")
}

// generateWrapperStruct 生成包装结构体
func (g *GinGenerator) generateWrapperStruct(iface SwaggerInterface, handlerItfName string) (string, string) {
	wrapperName := iface.GetWrapperName()
	constructorName := fmt.Sprintf("New%s", wrapperName)

	if len(handlerItfName) == 0 {
		template1 := `
func {{.ConstructorName}}(inner {{.InterfaceName}}) *{{.WrapperName}} {
    return &{{.WrapperName}}{
        inner: inner,
    }
}
`

		data1 := map[string]any{
			"ConstructorName": constructorName,
			"WrapperName":     wrapperName,
			"InterfaceName":   iface.Name,
		}
		constructorResult := utils.MustExecuteTemplate(data1, template1)

		template := `
type {{.WrapperName}} struct {
    inner {{.InterfaceName}}
}
`
		data := map[string]any{
			"WrapperName":   wrapperName,
			"InterfaceName": iface.Name,
		}

		result := utils.MustExecuteTemplate(data, template)
		return strings.TrimSpace(constructorResult), strings.TrimSpace(result)
	}

	template1 := `
func {{.ConstructorName}}(inner {{.InterfaceName}}, handler {{.HandlerName}}) *{{.WrapperName}} {
    return &{{.WrapperName}}{
        inner: inner,
        handler: handler,
    }
}
`

	data1 := map[string]any{
		"ConstructorName": constructorName,
		"WrapperName":     wrapperName,
		"InterfaceName":   iface.Name,
		"HandlerName":     handlerItfName,
	}

	constructorResult := utils.MustExecuteTemplate(data1, template1)

	template := `
type {{.WrapperName}} struct {
    inner {{.InterfaceName}}
    handler {{.HandlerName}}
}
`
	data := map[string]any{
		"WrapperName":   wrapperName,
		"InterfaceName": iface.Name,
		"HandlerName":   handlerItfName,
	}

	result := utils.MustExecuteTemplate(data, template)
	return strings.TrimSpace(constructorResult), strings.TrimSpace(result)
}

// generateBindMethod 生成通用的 bind 方法
func (g *GinGenerator) generateBindMethod(iface SwaggerInterface) string {
	wrapperName := iface.GetWrapperName()

	template := `
func (a *{{.WrapperName}}) bind(router gin.IRoutes, method, path string, preHandlers, innerHandlers []gin.HandlerFunc, f gin.HandlerFunc) {
    var basePath string
    if v, ok := router.(interface {
        BasePath() string
    }); ok {
        basePath = v.BasePath()
    }
    handlers := make([]gin.HandlerFunc, 0, len(preHandlers)+len(innerHandlers)+1)
    handlers = append(handlers, preHandlers...)
    handlers = append(handlers, innerHandlers...)
    handlers = append(handlers, f)
    router.Handle(method, strings.TrimPrefix(path, basePath), handlers...)
}
`

	data := map[string]any{
		"WrapperName": wrapperName,
	}

	result := utils.MustExecuteTemplate(data, template)
	return strings.TrimSpace(result)
}

// generateHandlerMethod 生成处理器方法
func (g *GinGenerator) generateHandlerMethod(iface SwaggerInterface, method SwaggerMethod) string {
	wrapperName := iface.GetWrapperName()
	handlerMethodName := method.Name

	paramBindingCode := g.generateParameterBinding(iface, method)
	methodCallCode := g.generateMethodCall(method)

	var template string
	if paramBindingCode == "" {
		template = `
func (a *{{.WrapperName}}) {{.HandlerMethodName}}(ctx *gin.Context) {
{{.MethodCall}}
}
`
	} else {
		template = `
func (a *{{.WrapperName}}) {{.HandlerMethodName}}(ctx *gin.Context) {
{{.ParameterBinding}}
{{.MethodCall}}
}
`
	}
	data := map[string]any{
		"WrapperName":       wrapperName,
		"HandlerMethodName": handlerMethodName,
		"ParameterBinding":  paramBindingCode,
		"MethodCall":        methodCallCode,
	}
	return strings.TrimSpace(utils.MustExecuteTemplate(data, template))
}

// generateMethodBinding 生成方法绑定
func (g *GinGenerator) generateMethodBinding(iface SwaggerInterface, method SwaggerMethod, middlewares []*parsers.MiddleWare) string {
	wrapperName := iface.GetWrapperName()
	bindMethodName := fmt.Sprintf("Bind%s", method.Name)
	handlerMethodName := method.Name
	prefix := iface.CommonDef.GetPrefix()

	ginPaths := lo.Map(method.GetPaths(), func(item string, index int) string {
		return prefix + convertPathToGinFormat(item)
	})

	template := `
func (a *{{.WrapperName}}) {{.BindMethodName}}(router gin.IRoutes, preHandlers ...gin.HandlerFunc) { {{- range .GinPath}}
	var handlers []gin.HandlerFunc
	if a.handler != nil {
		handlers = append(handlers, a.handler.PreHandlers()...)
		{{range $.Handlers}}handlers = append(handlers, a.handler.{{.}}()...)
		{{end -}}
	}
	a.bind(router, "{{$.HTTPMethod}}", "{{.}}", preHandlers, handlers, a.{{$.HandlerMethodName}}){{end}}
}
`

	data := map[string]any{
		"WrapperName":    wrapperName,
		"BindMethodName": bindMethodName,
		"Handlers": lo.Uniq(lo.Flatten(lo.Map(middlewares, func(item *parsers.MiddleWare, index int) []string {
			return item.Value
		}))),
		"HTTPMethod":        method.GetHTTPMethod(),
		"GinPath":           ginPaths,
		"HandlerMethodName": handlerMethodName,
	}
	return strings.TrimSpace(utils.MustExecuteTemplate(data, template))
}

// generateParameterBinding 生成参数绑定代码
func (g *GinGenerator) generateParameterBinding(iface SwaggerInterface, method SwaggerMethod) string {
	var lines []string

	for i, param := range method.Parameters {
		if param.Type.FullName == GinContextType ||
			param.Type.TypeName == "Context" ||
			strings.Contains(param.Type.FullName, "context.Context") {
			continue
		}
		if param.Source == "path" {
			lines = append(lines, g.generatePathParamBinding(param))
			continue
		} else if param.Source == "header" {
			lines = append(lines, g.generateHeaderParamBinding(param))
			continue
		}

		if i == len(method.Parameters)-1 {
			if method.GetHTTPMethod() == "GET" {
				lines = append(lines, g.generateQueryParamBinding(param))
			} else if v, _ := slices.Concat(method.Def, iface.CommonDef).GetAcceptType(); v == "json" {
				lines = append(lines, g.generateBodyParamBinding(param))
			} else {
				lines = append(lines, g.generateFormParamBinding(param))
			}
		}
	}

	for i, line := range lines {
		if line != "" && !strings.HasPrefix(line, "        ") {
			lines[i] = "        " + line
		}
	}

	return strings.Join(lines, "\n")
}

// generateTypedParamBinding 生成带类型转换的参数绑定
func (g *GinGenerator) generateTypedParamBinding(param Parameter, paramValue string) string {
	typeName := param.Type.TypeName

	switch typeName {
	case "int":
		return fmt.Sprintf(`%s := cast.ToInt(%s)`, param.Name, paramValue)
	case "int8":
		return fmt.Sprintf(`%s := cast.ToInt8(%s)`, param.Name, paramValue)
	case "int16":
		return fmt.Sprintf(`%s := cast.ToInt16(%s)`, param.Name, paramValue)
	case "int32":
		return fmt.Sprintf(`%s := cast.ToInt32(%s)`, param.Name, paramValue)
	case "int64":
		return fmt.Sprintf(`%s := cast.ToInt64(%s)`, param.Name, paramValue)
	case "uint":
		return fmt.Sprintf(`%s := cast.ToUint(%s)`, param.Name, paramValue)
	case "uint8":
		return fmt.Sprintf(`%s := cast.ToUint8(%s)`, param.Name, paramValue)
	case "uint16":
		return fmt.Sprintf(`%s := cast.ToUint16(%s)`, param.Name, paramValue)
	case "uint32":
		return fmt.Sprintf(`%s := cast.ToUint32(%s)`, param.Name, paramValue)
	case "uint64":
		return fmt.Sprintf(`%s := cast.ToUint64(%s)`, param.Name, paramValue)
	case "float32":
		return fmt.Sprintf(`%s := cast.ToFloat32(%s)`, param.Name, paramValue)
	case "float64":
		return fmt.Sprintf(`%s := cast.ToFloat64(%s)`, param.Name, paramValue)
	case "bool":
		return fmt.Sprintf(`%s := cast.ToBool(%s)`, param.Name, paramValue)
	case "string":
		return fmt.Sprintf(`%s := %s`, param.Name, paramValue)
	default:
		return fmt.Sprintf(`%s := %s`, param.Name, paramValue)
	}
}

// generatePathParamBinding 生成路径参数绑定
func (g *GinGenerator) generatePathParamBinding(param Parameter) string {
	paramNameInPath := param.Name
	if param.Alias != "" {
		paramNameInPath = param.Alias
	}
	paramValue := fmt.Sprintf(`ctx.Param("%s")`, paramNameInPath)
	return g.generateTypedParamBinding(param, paramValue)
}

// generateQueryParamBinding 生成query参数绑定
func (g *GinGenerator) generateQueryParamBinding(param Parameter) string {
	varName := param.Name
	typeName := param.Type.FullName

	s := fmt.Sprintf(`var %s %s
        if !onGinBind(ctx, &%s, "QUERY") {
			return
		}`, varName, typeName, varName)
	return s
}

// generateFormParamBinding 生成表单参数绑定
func (g *GinGenerator) generateFormParamBinding(param Parameter) string {
	varName := param.Name
	typeName := param.Type.FullName

	s := fmt.Sprintf(`var %s %s
        if !onGinBind(ctx, &%s, "FORM") {
			return
		}`, varName, typeName, varName)
	return s
}

// generateBodyParamBinding 生成 body 参数绑定
func (g *GinGenerator) generateBodyParamBinding(param Parameter) string {
	varName := param.Name
	typeName := param.Type.FullName

	s := fmt.Sprintf(`var %s %s
        if !onGinBind(ctx, &%s, "JSON") {
			return
		}`, varName, typeName, varName)
	return s
}

// generateHeaderParamBinding 生成头部参数绑定
func (g *GinGenerator) generateHeaderParamBinding(param Parameter) string {
	return fmt.Sprintf(`%s := ctx.GetHeader("%s")`, param.Name, param.Name)
}

// generateMethodCall 生成方法调用代码
func (g *GinGenerator) generateMethodCall(method SwaggerMethod) string {
	var args []string

	needsContext := g.methodNeedsContext(method)
	if needsContext {
		args = append(args, "ctx")
	}

	for _, param := range method.Parameters {
		if param.Type.FullName == GinContextType ||
			param.Type.TypeName == "Context" ||
			strings.Contains(param.Type.FullName, "context.Context") {
			continue
		}
		args = append(args, param.Name)
	}

	methodCall := fmt.Sprintf("a.inner.%s(%s)", method.Name, strings.Join(args, ", "))

	responseCode := g.generateResponseHandling(method, methodCall, "ctx")

	return "        " + responseCode
}

// methodNeedsContext 检查方法是否需要 context 参数
func (g *GinGenerator) methodNeedsContext(method SwaggerMethod) bool {
	for _, param := range method.Parameters {
		if param.Type.FullName == GinContextType ||
			param.Type.TypeName == "Context" ||
			strings.Contains(param.Type.FullName, "context.Context") {
			return true
		}
	}
	return false
}

// generateResponseHandling 生成响应处理代码
func (g *GinGenerator) generateResponseHandling(method SwaggerMethod, methodCall, receiverName string) string {
	if method.ResponseType.FullName == "" {
		return fmt.Sprintf(`%s
        onGinResponse[string](%s, "", nil)`, methodCall, receiverName)
	}

	if g.isErrorType(method.ResponseType) {
		return fmt.Sprintf(`err := %s
        onGinResponse[string](%s, "", err)`, methodCall, receiverName)
	}

	if *version < 2 {
		return fmt.Sprintf(`var result %s = %s
        onGinResponse(%s, result)`, method.ResponseType.FullName, methodCall, receiverName)
	}
	return fmt.Sprintf(`result, err := %s
        onGinResponse[%s](%s, result, err)`, methodCall, method.ResponseType.FullName, receiverName)
}

// isErrorType 检查是否是错误类型
func (g *GinGenerator) isErrorType(typeInfo TypeInfo) bool {
	return typeInfo.TypeName == "error" ||
		strings.Contains(typeInfo.FullName, "error") ||
		strings.HasSuffix(typeInfo.TypeName, "Error")
}

// GenerateComplete 生成完整的 Gin 绑定代码
func (g *GinGenerator) GenerateComplete(comments map[string]string) string {
	var parts []string

	constructorCode, ginCode := g.GenerateGinCode(comments)
	if constructorCode != "" {
		parts = append(parts, constructorCode)
	}
	if ginCode != "" {
		parts = append(parts, ginCode)
	}

	helperFunctions := g.generateHelperFunctions()
	if helperFunctions != "" {
		helperFunctions = strings.Join(lo.Map(strings.Split(helperFunctions, "\n"), func(item string, _ int) string {
			return "//" + item
		}), "\n")
		parts = append(parts, helperFunctions)
	}

	return strings.Join(parts, "\n\n")
}

// generateHelperFunctions 生成辅助函数
func (g *GinGenerator) generateHelperFunctions() string {
	return `
func onGinBind(c *gin.Context, val any, typ string) bool {
    switch typ {
    case "JSON":
        if err := c.ShouldBindJSON(val); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return false
        }
    case "FORM":
        if err := c.ShouldBind(val); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return false
        }
    case "QUERY":
        if err := c.ShouldBindQuery(val); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return false
        }
    default:
        if err := c.ShouldBind(val); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return false
        }
    }
    return true
}

func onGinResponse[T any](c *gin.Context, data any, err error) {
    c.JSON(200, data)
}

func onGinBindErr(c *gin.Context, err error) {
    c.JSON(500, gin.H{"error": err.Error()})
}`
}

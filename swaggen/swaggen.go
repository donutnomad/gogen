package swaggen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"

	"github.com/donutnomad/gogen/internal/xast"
	"github.com/donutnomad/gogen/plugin"
	parsers "github.com/donutnomad/gogen/swaggen/parser"
)

const generatorName = "swaggen"

// SwagParams 定义 Swag 生成器的参数
type SwagParams struct {
	Output string `param:"name=output,required=false,default=,description=输出文件路径"`
}

// SwagGenerator 实现 plugin.Generator 接口
type SwagGenerator struct {
	plugin.BaseGenerator
}

// NewSwagGenerator 创建 Swag 生成器
func NewSwagGenerator() *SwagGenerator {
	gen := &SwagGenerator{
		BaseGenerator: *plugin.NewBaseGeneratorWithParamsStruct(
			generatorName,
			// HTTP 方法注解作为触发器
			[]string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			[]plugin.TargetKind{plugin.TargetInterface},
			SwagParams{},
		),
	}
	gen.SetPriority(50) // Swag 优先级
	return gen
}

// AnnotationFormats 返回触发注解的显示格式
func (g *SwagGenerator) AnnotationFormats() []string {
	return []string{
		"GET(path)",
		"POST(path)",
		"PUT(path)",
		"PATCH(path)",
		"DELETE(path)",
	}
}

// NoDefaultParams 返回 true，表示不显示默认的 output 参数和示例
func (g *SwagGenerator) NoDefaultParams() bool {
	return true
}

// ExtraHelp 返回辅助注解的帮助信息
func (g *SwagGenerator) ExtraHelp() string {
	return `    辅助注解 (接口级别):
      @TAG(name)              - Swagger 标签分组
      @SECURITY(name)         - 安全认证，支持 exclude/include 参数
      @HEADER(name,required,desc) - 公共请求头
      @PREFIX(path)           - 路由前缀
    辅助注解 (方法级别):
      @JSON                   - 响应类型为 JSON
      @MIME(type)             - 自定义响应 MIME 类型
      @JSON-REQ               - 请求类型为 JSON
      @FORM-REQ               - 请求类型为表单
      @MIME-REQ(type)         - 自定义请求 MIME 类型
      @MID(name1 name2)       - 中间件，多个用空格分隔
      @Removed                - 从生成中移除此方法
      @ExcludeFromBindAll     - 从 BindAll 中排除
      @Raw(text)              - 原始 Swagger 注释
    辅助注解 (参数级别):
      @PARAM                  - 路径参数，可指定别名 @PARAM(alias)
      @QUERY                  - 查询参数
      @BODY                   - 请求体参数
      @FORM                   - 表单参数
      @HEADER                 - 请求头参数
    示例:
      // @TAG(用户管理)
      // @SECURITY(Bearer)
      type IUserAPI interface {
          // 获取用户
          // @GET(/api/v1/user/{id})
          // @JSON
          GetUser(ctx context.Context, id int64) (Response, error)
      }
`
}

// Generate 执行代码生成
func (g *SwagGenerator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	result := plugin.NewGenerateResult()

	if len(ctx.Targets) == 0 {
		return result, nil
	}

	// 按输出文件分组处理
	// key: 输出路径, value: 待处理的接口列表
	fileTargets := make(map[string][]*swagTargetInfo)

	// 用于去重的 map，key: 文件路径+接口名
	processedInterfaces := make(map[string]bool)

	for _, at := range ctx.Targets {
		// 确保是接口类型
		if at.Target.Kind != plugin.TargetInterface {
			continue
		}

		// 去重：同一个文件中的同一个接口只处理一次
		interfaceKey := at.Target.FilePath + ":" + at.Target.Name
		if processedInterfaces[interfaceKey] {
			continue
		}
		processedInterfaces[interfaceKey] = true

		// 获取输出路径
		fileConfig := ctx.GetFileConfig(at.Target.FilePath)
		ann := getFirstAnnotation(at.Annotations, "GET", "POST", "PUT", "PATCH", "DELETE")
		outputPath := plugin.GetOutputPath(at.Target, ann, "$FILE_swagger.go", fileConfig, g.Name(), ctx.DefaultOutput)

		// 解析接口
		swaggerInterface, err := g.parseInterface(at)
		if err != nil {
			result.AddError(fmt.Errorf("解析接口 %s 失败: %w", at.Target.Name, err))
			continue
		}

		if swaggerInterface == nil || len(swaggerInterface.Methods) == 0 {
			continue
		}

		fileTargets[outputPath] = append(fileTargets[outputPath], &swagTargetInfo{
			iface:  swaggerInterface,
			target: at,
		})

		if ctx.Verbose {
			fmt.Printf("[swaggen] 处理接口 %s -> %s (%d 个方法)\n",
				at.Target.Name, outputPath, len(swaggerInterface.Methods))
		}
	}

	// 为每个输出文件生成代码
	// 按输出路径排序，确保生成顺序一致
	outputPaths := make([]string, 0, len(fileTargets))
	for outputPath := range fileTargets {
		outputPaths = append(outputPaths, outputPath)
	}
	slices.Sort(outputPaths)

	for _, outputPath := range outputPaths {
		targets := fileTargets[outputPath]
		// 按接口名称排序，确保同一文件中不同接口的顺序一致
		slices.SortFunc(targets, func(a, b *swagTargetInfo) int {
			return strings.Compare(a.target.Target.Name, b.target.Target.Name)
		})

		code, err := g.generateCode(targets)
		if err != nil {
			result.AddError(fmt.Errorf("生成 %s 失败: %w", outputPath, err))
			continue
		}
		result.AddRawOutput(outputPath, []byte(code))
	}

	return result, nil
}

// swagTargetInfo 存储单个接口的处理信息
type swagTargetInfo struct {
	iface  *SwaggerInterface
	target *plugin.AnnotatedTarget
}

// parseInterface 解析接口定义
func (g *SwagGenerator) parseInterface(at *plugin.AnnotatedTarget) (*SwaggerInterface, error) {
	// 获取 AST 节点，验证是否为接口类型
	typeSpec, ok := at.Target.Node.(*ast.TypeSpec)
	if !ok {
		return nil, fmt.Errorf("节点不是 TypeSpec")
	}

	if _, ok := typeSpec.Type.(*ast.InterfaceType); !ok {
		return nil, fmt.Errorf("类型不是接口")
	}

	// 解析文件获取导入信息
	imports, err := parseFileImports(at.Target.FilePath)
	if err != nil {
		return nil, fmt.Errorf("解析导入信息失败: %w", err)
	}

	// 创建 tag 解析器
	tagsParser, err := newTagParserSafe()
	if err != nil {
		return nil, fmt.Errorf("创建标签解析器失败: %w", err)
	}

	// 读取文件内容用于解析参数
	fileBs, err := readFile(at.Target.FilePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 创建返回类型解析器
	typeParser := NewReturnTypeParser(imports)

	// 重新解析文件获取正确的位置信息
	fset := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fset, at.Target.FilePath, fileBs, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("重新解析文件失败: %w", err)
	}

	// 查找接口类型
	var targetInterfaceType *ast.InterfaceType
	for _, decl := range parsedFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != at.Target.Name {
				continue
			}
			if it, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				targetInterfaceType = it
				break
			}
		}
	}

	if targetInterfaceType == nil {
		return nil, fmt.Errorf("无法找到接口 %s", at.Target.Name)
	}

	// 解析接口
	swaggerInterface := &SwaggerInterface{
		Name:        at.Target.Name,
		PackagePath: at.Target.PackageName,
		Imports:     imports,
		Methods:     []SwaggerMethod{},
	}

	// 解析接口级注释
	if genDecl := findGenDecl(at.Target.FilePath, at.Target.Name); genDecl != nil && genDecl.Doc != nil {
		for _, comment := range genDecl.Doc.List {
			line := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if strings.HasPrefix(line, "@") {
				parse, err := tagsParser.Parse(line)
				if err == nil {
					swaggerInterface.CommonDef = append(swaggerInterface.CommonDef, parse.(parsers.Definition))
				}
			}
		}
	}

	// 创建注释解析器（使用之前创建的 fset）
	annotationParser := NewAnnotationParser(fset)

	// 解析接口方法（使用重新解析的接口类型）
	for _, field := range targetInterfaceType.Methods.List {
		funcType, ok := field.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		if len(field.Names) == 0 {
			continue
		}

		// 创建虚拟的 FuncDecl 用于解析
		virtualFunc := &ast.FuncDecl{
			Name: field.Names[0],
			Type: funcType,
			Doc:  field.Doc,
		}

		swaggerMethod, err := annotationParser.ParseMethodAnnotations(virtualFunc)
		if err != nil {
			return nil, fmt.Errorf("解析方法 %s 失败: %w", field.Names[0].Name, err)
		}

		if swaggerMethod == nil {
			continue
		}

		// 解析参数
		if funcType.Params != nil {
			paramAnnotations, _ := parsers.ParseParameters(getParamsContent(fileBs, fset, funcType))
			allParams := extractBaseParameters(funcType.Params.List, paramAnnotations, typeParser, annotationParser)
			mapPathParameters(swaggerMethod, allParams)
			swaggerMethod.Parameters = allParams
		}

		// 解析返回类型
		if funcType.Results != nil && len(funcType.Results.List) > 0 {
			firstResult := funcType.Results.List[0]
			swaggerMethod.ResponseType = typeParser.ParseReturnType(firstResult.Type)
		}

		swaggerInterface.Methods = append(swaggerInterface.Methods, *swaggerMethod)
	}

	return swaggerInterface, nil
}

// generateCode 生成完整代码
func (g *SwagGenerator) generateCode(targets []*swagTargetInfo) (string, error) {
	if len(targets) == 0 {
		return "", fmt.Errorf("没有目标需要生成")
	}

	// 收集所有接口
	var interfaces []SwaggerInterface
	for _, t := range targets {
		interfaces = append(interfaces, *t.iface)
	}

	// 获取包名
	packageName := targets[0].target.Target.PackageName

	// 创建接口集合
	collection := &InterfaceCollection{
		Interfaces: interfaces,
	}

	// 创建生成器
	swaggerGen := NewSwaggerGenerator2(collection)
	ginGen := NewGinGenerator(collection)

	// 生成代码部分
	var parts []string

	// 文件头
	header := swaggerGen.GenerateFileHeader(packageName)
	parts = append(parts, header)

	// 导入声明
	imports := swaggerGen.GenerateImports()
	if imports != "" {
		parts = append(parts, imports, "")
	}

	// Swagger 注释
	swaggerComments := swaggerGen.GenerateSwaggerComments()

	// Gin 绑定代码
	ginCode := ginGen.GenerateComplete(swaggerComments)
	if ginCode != "" {
		parts = append(parts, ginCode)
	}

	return strings.Join(parts, "\n"), nil
}

// 辅助函数

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func findGenDecl(filePath string, typeName string) *ast.GenDecl {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if typeSpec.Name.Name == typeName {
				return genDecl
			}
		}
	}

	return nil
}

func getParamsContent(fileBs []byte, fset *token.FileSet, funcType *ast.FuncType) string {
	if funcType.Params == nil {
		return ""
	}
	start := fset.Position(funcType.Params.Opening)
	end := fset.Position(funcType.Params.Closing)
	return getContent(fileBs, start, end)
}

func extractBaseParameters(fields []*ast.Field, paramAnnotations []parsers.Parameter, typeParser *ReturnTypeParser, annotationParser *AnnotationParser) []Parameter {
	var allParams []Parameter

	var expandedFields []*ast.Field
	for _, field := range fields {
		for _, name := range field.Names {
			expandedFields = append(expandedFields, &ast.Field{
				Doc:     field.Doc,
				Names:   []*ast.Ident{name},
				Type:    field.Type,
				Tag:     field.Tag,
				Comment: field.Comment,
			})
		}
	}

	for i, field := range expandedFields {
		paramType := typeParser.ParseParameterType(field.Type)

		if i < len(paramAnnotations) {
			annotation := paramAnnotations[i]
			parameter := annotationParser.ParseParameterAnnotations(annotation.Name, annotation.Tag)
			parameter.Type = paramType
			allParams = append(allParams, parameter)
		}
	}

	return allParams
}

func mapPathParameters(swaggerMethod *SwaggerMethod, allParams []Parameter) {
	for _, routerPath := range swaggerMethod.GetPaths() {
		pathParams := extractPathParameters(routerPath)
		processPathParams(routerPath, pathParams, allParams)
	}
}

func extractPathParameters(path string) []Parameter {
	var params []Parameter

	start := 0
	for {
		openIdx := strings.Index(path[start:], "{")
		if openIdx == -1 {
			break
		}
		openIdx += start

		closeIdx := strings.Index(path[openIdx:], "}")
		if closeIdx == -1 {
			break
		}
		closeIdx += openIdx

		paramName := path[openIdx+1 : closeIdx]
		params = append(params, Parameter{
			Name:   paramName,
			Source: ParamSourcePath,
		})

		start = closeIdx + 1
	}

	return params
}

func processPathParams(routerPath string, pathParams []Parameter, allParams []Parameter) {
	for _, pathParam := range pathParams {
		paramIndex := findMatchingParameter(pathParam, allParams)

		if paramIndex != -1 {
			allParams[paramIndex].PathName = pathParam.Name
			allParams[paramIndex].Source = ParamSourcePath
		}
	}
}

func findMatchingParameter(pathParam Parameter, allParams []Parameter) int {
	for i, param := range allParams {
		if param.Name == pathParam.Name {
			return i
		}

		if param.Alias != "" && param.Alias == pathParam.Name {
			return i
		}

		if parsers.NewCamelString(pathParam.Name).Equal(param.Name) {
			allParams[i].Alias = pathParam.Name
			allParams[i].Source = ParamSourcePath
			return i
		}
	}

	return -1
}

// NewSwaggerGenerator2 创建 Swagger 生成器（为了避免与老代码命名冲突）
func NewSwaggerGenerator2(collection *InterfaceCollection) *SwaggerGenerator2 {
	parser, err := newTagParserSafe()
	if err != nil {
		panic(err)
	}
	return &SwaggerGenerator2{
		collection: collection,
		tagsParser: parser,
	}
}

// SwaggerGenerator2 Swagger 生成器（内部使用）
type SwaggerGenerator2 struct {
	collection *InterfaceCollection
	tagsParser *parsers.Parser
}

// GenerateSwaggerComments 生成 Swagger 注释
func (g *SwaggerGenerator2) GenerateSwaggerComments() map[string]string {
	sg := &SwaggerGenerator{collection: g.collection, tagsParser: g.tagsParser}
	return sg.GenerateSwaggerComments()
}

// GenerateFileHeader 生成文件头部
func (g *SwaggerGenerator2) GenerateFileHeader(packageName string) string {
	sg := &SwaggerGenerator{collection: g.collection, tagsParser: g.tagsParser}
	return sg.GenerateFileHeader(packageName)
}

// GenerateImports 生成导入声明
func (g *SwaggerGenerator2) GenerateImports() string {
	sg := &SwaggerGenerator{collection: g.collection, tagsParser: g.tagsParser}
	return sg.GenerateImports()
}

// getFirstAnnotation 获取第一个匹配的注解
func getFirstAnnotation(annotations []*plugin.Annotation, names ...string) *plugin.Annotation {
	for _, ann := range annotations {
		for _, name := range names {
			if ann.Name == name {
				return ann
			}
		}
	}
	return nil
}

// parseFileImports 解析文件导入信息
func parseFileImports(filePath string) (xast.ImportInfoSlice, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	return new(xast.ImportInfoSlice).From(file.Imports), nil
}

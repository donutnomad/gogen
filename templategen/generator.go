package templategen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/donutnomad/gogen/plugin"
)

const generatorName = "templategen"

// DefineParams @Define 注解参数（动态解析，不使用固定结构体）
type DefineParams struct {
	Name   string            // name 参数
	Values map[string]string // 其他 k=v 参数
}

// ImportParams @Import 注解参数
type ImportParams struct {
	Alias string `param:"name=alias,required=true,description=包别名"`
	Path  string `param:"name=path,required=true,description=完整包路径"`
}

// TemplateGenerator 实现 plugin.Generator 接口
type TemplateGenerator struct {
	plugin.BaseGenerator
}

// NewTemplateGenerator 创建 templategen 生成器
func NewTemplateGenerator() *TemplateGenerator {
	gen := &TemplateGenerator{
		BaseGenerator: *plugin.NewBaseGenerator(
			generatorName,
			[]string{"Define", "Import"},
			[]plugin.TargetKind{
				plugin.TargetStruct,
				plugin.TargetInterface,
				plugin.TargetMethod,
				plugin.TargetFunc,
			},
		),
	}
	gen.SetPriority(50)
	return gen
}

// Generate 执行代码生成
func (g *TemplateGenerator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	result := plugin.NewGenerateResult()

	if len(ctx.Targets) == 0 {
		return result, nil
	}

	// 按文件分组目标
	fileTargets := make(map[string][]*plugin.AnnotatedTarget)
	for _, target := range ctx.Targets {
		fileTargets[target.Target.FilePath] = append(fileTargets[target.Target.FilePath], target)
	}

	// 处理每个文件
	for filePath, targets := range fileTargets {
		// 解析文件的 go:gogen: plugin:templategen 配置
		configs, err := parseTemplateConfigs(filePath)
		if err != nil {
			result.AddError(fmt.Errorf("解析 %s 的模板配置失败: %w", filePath, err))
			continue
		}

		if len(configs) == 0 {
			// 该文件没有 templategen 配置，跳过
			continue
		}

		// 收集文件中的元数据
		data, err := g.collectTemplateData(filePath, targets)
		if err != nil {
			result.AddError(fmt.Errorf("收集 %s 的模板数据失败: %w", filePath, err))
			continue
		}

		// 为每个模板配置生成代码
		for _, cfg := range configs {
			outputPath := cfg.Output
			if outputPath == "" {
				outputPath = "$FILE_gen.go"
			}
			outputPath = resolveOutputPath(filePath, outputPath)

			// 加载并执行模板
			content, err := g.executeTemplate(cfg, data, filePath)
			if err != nil {
				result.AddError(fmt.Errorf("执行模板 %s 失败: %w", cfg.Template, err))
				continue
			}

			// 转换为 gg.Generator
			gen, err := plugin.ParseSourceToGG(content)
			if err != nil {
				if ctx.Verbose {
					fmt.Printf("[templategen] 生成的原始内容:\n%s\n", content)
				}
				result.AddError(fmt.Errorf("解析生成的代码失败: %w", err))
				continue
			}

			result.AddDefinition(outputPath, gen)

			if ctx.Verbose {
				fmt.Printf("[templategen] %s -> %s (模板: %s)\n", filePath, outputPath, cfg.Template)
			}
		}
	}

	return result, nil
}

// TemplateConfig 单个模板配置
type TemplateConfig struct {
	Template string   // 模板文件路径
	Output   string   // 输出文件路径
	Include  []string // 额外包含的模板文件
}

// templateConfigRegex 匹配 plugin:templategen 配置
var templateConfigRegex = regexp.MustCompile(`plugin:templategen\s+(.+?)(?:\s+plugin:|$)`)

// parseTemplateConfigs 解析文件中的 templategen 配置
func parseTemplateConfigs(filePath string) ([]TemplateConfig, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var configs []TemplateConfig

	// 遍历所有注释
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			text := c.Text
			// 移除注释前缀
			text = strings.TrimPrefix(text, "//")
			text = strings.TrimPrefix(text, "/*")
			text = strings.TrimSuffix(text, "*/")
			text = strings.TrimSpace(text)

			// 检查是否是 go:gogen: 指令
			if !strings.HasPrefix(text, "go:gogen:") {
				continue
			}
			text = strings.TrimPrefix(text, "go:gogen:")
			text = strings.TrimSpace(text)

			// 查找 plugin:templategen 配置
			matches := templateConfigRegex.FindAllStringSubmatch(text, -1)
			for _, match := range matches {
				if len(match) < 2 {
					continue
				}
				cfg := parseTemplateArgs(match[1])
				if cfg.Template != "" {
					configs = append(configs, cfg)
				}
			}
		}
	}

	return configs, nil
}

// parseTemplateArgs 解析 templategen 参数
func parseTemplateArgs(args string) TemplateConfig {
	cfg := TemplateConfig{}
	parts := splitArgs(args)

	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "-template":
			if i+1 < len(parts) {
				cfg.Template = unquote(parts[i+1])
				i++
			}
		case "-output":
			if i+1 < len(parts) {
				cfg.Output = unquote(parts[i+1])
				i++
			}
		case "-include":
			if i+1 < len(parts) {
				cfg.Include = append(cfg.Include, unquote(parts[i+1]))
				i++
			}
		}
	}

	return cfg
}

// splitArgs 分割参数，处理引号
func splitArgs(s string) []string {
	var result []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, c := range s {
		switch {
		case (c == '"' || c == '\'' || c == '`') && !inQuote:
			inQuote = true
			quoteChar = c
		case c == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
		case c == ' ' && !inQuote:
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(c)
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// unquote 移除引号
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') ||
			(s[0] == '`' && s[len(s)-1] == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// resolveOutputPath 解析输出路径
func resolveOutputPath(srcPath, outputPattern string) string {
	dir := filepath.Dir(srcPath)
	baseName := filepath.Base(srcPath)
	nameWithoutExt := strings.TrimSuffix(baseName, ".go")

	result := outputPattern
	result = strings.ReplaceAll(result, "$FILE", nameWithoutExt)
	result = strings.ReplaceAll(result, "$DIR", dir)

	if !filepath.IsAbs(result) && !strings.HasPrefix(result, ".") {
		result = filepath.Join(dir, result)
	}

	if !strings.HasSuffix(result, ".go") {
		result += ".go"
	}

	return result
}

// collectTemplateData 收集模板数据
func (g *TemplateGenerator) collectTemplateData(filePath string, targets []*plugin.AnnotatedTarget) (*TemplateData, error) {
	data := &TemplateData{
		File: FileInfo{
			Path:        filePath,
			Dir:         filepath.Dir(filePath),
			Name:        strings.TrimSuffix(filepath.Base(filePath), ".go"),
			PackageName: "",
		},
		Structs:       []StructData{},
		Interfaces:    []InterfaceData{},
		Functions:     []FunctionData{},
		ImportAliases: make(map[string]string),
		Imports:       NewImportManager(),
	}

	// 获取包名
	if len(targets) > 0 {
		data.File.PackageName = targets[0].Target.PackageName
	}

	// 创建 import 解析器
	resolver, err := NewImportResolver(filePath)
	if err != nil {
		return nil, err
	}

	// 按类型分组
	structMap := make(map[string]*StructData)
	methodsByReceiver := make(map[string][]*plugin.AnnotatedTarget)

	for _, target := range targets {
		// 处理 @Import 注解
		for _, ann := range target.Annotations {
			if ann.Name == "Import" {
				alias := ann.Params["alias"]
				path := ann.Params["path"]
				if alias != "" && path != "" {
					data.ImportAliases[alias] = path
					resolver.AddAlias(alias, path)
				}
			}
		}

		switch target.Target.Kind {
		case plugin.TargetStruct:
			defines := g.parseDefines(target.Annotations, resolver)
			if len(defines) > 0 {
				// 检查是否已存在，如果存在则合并 Defines
				if existing, ok := structMap[target.Target.Name]; ok {
					// 合并 defines
					for name, values := range defines {
						if existing.Defines[name] == nil {
							existing.Defines[name] = make(map[string]TypeRef)
						}
						for k, v := range values {
							existing.Defines[name][k] = v
						}
					}
				} else {
					sd := &StructData{
						Name:    target.Target.Name,
						Fields:  extractFields(target.Target.Node),
						Defines: defines,
						Methods: []MethodData{},
					}
					structMap[target.Target.Name] = sd
				}
			}

		case plugin.TargetInterface:
			defines := g.parseDefines(target.Annotations, resolver)
			if len(defines) > 0 {
				// 检查是否已存在
				found := false
				for i := range data.Interfaces {
					if data.Interfaces[i].Name == target.Target.Name {
						// 合并 defines
						for name, values := range defines {
							if data.Interfaces[i].Defines[name] == nil {
								data.Interfaces[i].Defines[name] = make(map[string]TypeRef)
							}
							for k, v := range values {
								data.Interfaces[i].Defines[name][k] = v
							}
						}
						found = true
						break
					}
				}
				if !found {
					data.Interfaces = append(data.Interfaces, InterfaceData{
						Name:    target.Target.Name,
						Methods: extractInterfaceMethods(target.Target.Node),
						Defines: defines,
					})
				}
			}

		case plugin.TargetMethod:
			methodsByReceiver[target.Target.ReceiverType] = append(
				methodsByReceiver[target.Target.ReceiverType],
				target,
			)

		case plugin.TargetFunc:
			defines := g.parseDefines(target.Annotations, resolver)
			if len(defines) > 0 {
				// 检查是否已存在
				found := false
				for i := range data.Functions {
					if data.Functions[i].Name == target.Target.Name {
						// 合并 defines
						for name, values := range defines {
							if data.Functions[i].Defines[name] == nil {
								data.Functions[i].Defines[name] = make(map[string]TypeRef)
							}
							for k, v := range values {
								data.Functions[i].Defines[name][k] = v
							}
						}
						found = true
						break
					}
				}
				if !found {
					params, returns := extractFuncSignature(target.Target.Node)
					data.Functions = append(data.Functions, FunctionData{
						Name:    target.Target.Name,
						Params:  params,
						Returns: returns,
						Defines: defines,
					})
				}
			}
		}
	}

	// 将 structMap 中的结构体添加到 data.Structs
	for _, sd := range structMap {
		data.Structs = append(data.Structs, *sd)
	}

	// 将方法关联到结构体
	for receiverType, methods := range methodsByReceiver {
		// 移除指针前缀
		cleanType := strings.TrimPrefix(receiverType, "*")
		sd, exists := structMap[cleanType]
		if !exists {
			// 如果结构体不在 map 中，但有带注解的方法，创建一个新的结构体条目
			sd = &StructData{
				Name:    cleanType,
				Fields:  []FieldData{},
				Defines: DefineGroup{},
				Methods: []MethodData{},
			}
			structMap[cleanType] = sd
			data.Structs = append(data.Structs, *sd)
		}

		// 去重方法（同一方法可能因为多个注解被分派多次）
		seenMethods := make(map[string]bool)
		for _, m := range methods {
			if seenMethods[m.Target.Name] {
				continue
			}
			seenMethods[m.Target.Name] = true

			defines := g.parseDefines(m.Annotations, resolver)
			if len(defines) > 0 {
				params, returns := extractFuncSignature(m.Target.Node)
				md := MethodData{
					Name:         m.Target.Name,
					ReceiverName: m.Target.ReceiverName,
					ReceiverType: cleanType,
					IsPointer:    strings.HasPrefix(receiverType, "*"),
					Params:       params,
					Returns:      returns,
					Defines:      defines,
				}
				sd.Methods = append(sd.Methods, md)
			}
		}
	}

	// 更新 data.Structs 以包含方法（需要重新同步因为 structMap 中的指针和 data.Structs 中的副本）
	for i := range data.Structs {
		if sd, exists := structMap[data.Structs[i].Name]; exists {
			data.Structs[i] = *sd
		}
	}

	// 排序以保证输出稳定性
	slices.SortFunc(data.Structs, func(a, b StructData) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(data.Interfaces, func(a, b InterfaceData) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(data.Functions, func(a, b FunctionData) int {
		return strings.Compare(a.Name, b.Name)
	})

	return data, nil
}

// parseDefines 解析 @Define 注解
func (g *TemplateGenerator) parseDefines(annotations []*plugin.Annotation, resolver *ImportResolver) DefineGroup {
	defines := make(DefineGroup)

	for _, ann := range annotations {
		if ann.Name != "Define" {
			continue
		}

		name := ann.Params["name"]
		if name == "" {
			continue
		}

		if defines[name] == nil {
			defines[name] = make(map[string]TypeRef)
		}

		for k, v := range ann.Params {
			if k == "name" {
				continue
			}
			defines[name][k] = resolver.ResolveTypeRef(v)
		}
	}

	return defines
}

// executeTemplate 执行模板
func (g *TemplateGenerator) executeTemplate(cfg TemplateConfig, data *TemplateData, srcFilePath string) ([]byte, error) {
	// 解析模板路径
	templatePath := resolveTemplatePath(cfg.Template, srcFilePath)

	// 创建模板，添加 Sprig 函数和自定义函数
	tmpl := template.New(filepath.Base(templatePath)).
		Funcs(sprig.FuncMap()).
		Funcs(customFuncs(data))

	// 加载基础模板（_*.tmpl）
	dir := filepath.Dir(templatePath)
	baseFiles, _ := filepath.Glob(filepath.Join(dir, "_*.tmpl"))

	// 加载 -include 指定的文件
	var allFiles []string
	for _, inc := range cfg.Include {
		incPath := resolveTemplatePath(inc, srcFilePath)
		allFiles = append(allFiles, incPath)
	}
	allFiles = append(allFiles, baseFiles...)
	allFiles = append(allFiles, templatePath)

	// 解析所有模板文件
	tmpl, err := tmpl.ParseFiles(allFiles...)
	if err != nil {
		return nil, fmt.Errorf("解析模板失败: %w", err)
	}

	// 执行模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("执行模板失败: %w", err)
	}

	// 添加 package 声明和 imports
	return g.wrapGeneratedCode(data, buf.Bytes())
}

// wrapGeneratedCode 包装生成的代码，添加 package 和 imports
func (g *TemplateGenerator) wrapGeneratedCode(data *TemplateData, body []byte) ([]byte, error) {
	var buf bytes.Buffer

	// 写入 package
	fmt.Fprintf(&buf, "package %s\n\n", data.File.PackageName)

	// 收集所有需要的 imports
	allImports := make(map[string]string)
	for path, alias := range data.Imports.All() {
		allImports[path] = alias
	}

	// 写入 imports
	if len(allImports) > 0 {
		buf.WriteString("import (\n")
		for path, alias := range allImports {
			if alias != "" {
				fmt.Fprintf(&buf, "\t%s %q\n", alias, path)
			} else {
				fmt.Fprintf(&buf, "\t%q\n", path)
			}
		}
		buf.WriteString(")\n\n")
	}

	// 写入主体
	buf.Write(body)

	return buf.Bytes(), nil
}

// resolveTemplatePath 解析模板路径
func resolveTemplatePath(templatePath, srcFilePath string) string {
	if filepath.IsAbs(templatePath) {
		return templatePath
	}

	// 相对于源文件目录
	srcDir := filepath.Dir(srcFilePath)
	relPath := filepath.Join(srcDir, templatePath)
	if fileExists(relPath) {
		return relPath
	}

	// 相对于项目根目录（查找 go.mod）
	projectRoot := findProjectRoot(srcDir)
	if projectRoot != "" {
		rootPath := filepath.Join(projectRoot, templatePath)
		if fileExists(rootPath) {
			return rootPath
		}
	}

	// 返回原始路径
	return templatePath
}

// findProjectRoot 查找项目根目录
func findProjectRoot(dir string) string {
	for {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	// 简单检查
	return true
}

// extractFields 从 AST 节点提取字段
func extractFields(node ast.Node) []FieldData {
	var fields []FieldData

	typeSpec, ok := node.(*ast.TypeSpec)
	if !ok {
		return fields
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok || structType.Fields == nil {
		return fields
	}

	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			fd := FieldData{
				Name: name.Name,
				Type: exprToString(field.Type),
			}
			if field.Tag != nil {
				fd.Tag = field.Tag.Value
			}
			if field.Comment != nil {
				fd.Comment = field.Comment.Text()
			}
			fields = append(fields, fd)
		}
	}

	return fields
}

// extractInterfaceMethods 从 AST 节点提取接口方法
func extractInterfaceMethods(node ast.Node) []MethodSig {
	var methods []MethodSig

	typeSpec, ok := node.(*ast.TypeSpec)
	if !ok {
		return methods
	}

	interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok || interfaceType.Methods == nil {
		return methods
	}

	for _, method := range interfaceType.Methods.List {
		if len(method.Names) == 0 {
			continue
		}
		funcType, ok := method.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		params, returns := extractParamsAndReturns(funcType)
		methods = append(methods, MethodSig{
			Name:    method.Names[0].Name,
			Params:  params,
			Returns: returns,
		})
	}

	return methods
}

// extractFuncSignature 提取函数签名
func extractFuncSignature(node ast.Node) ([]ParamData, []ReturnData) {
	switch n := node.(type) {
	case *ast.FuncDecl:
		return extractParamsAndReturns(n.Type)
	}
	return nil, nil
}

// extractParamsAndReturns 从 FuncType 提取参数和返回值
func extractParamsAndReturns(funcType *ast.FuncType) ([]ParamData, []ReturnData) {
	var params []ParamData
	var returns []ReturnData

	if funcType.Params != nil {
		for _, param := range funcType.Params.List {
			typeStr := exprToString(param.Type)
			if len(param.Names) == 0 {
				params = append(params, ParamData{Type: typeStr})
			} else {
				for _, name := range param.Names {
					params = append(params, ParamData{Name: name.Name, Type: typeStr})
				}
			}
		}
	}

	if funcType.Results != nil {
		for _, result := range funcType.Results.List {
			typeStr := exprToString(result.Type)
			if len(result.Names) == 0 {
				returns = append(returns, ReturnData{Type: typeStr})
			} else {
				for _, name := range result.Names {
					returns = append(returns, ReturnData{Name: name.Name, Type: typeStr})
				}
			}
		}
	}

	return params, returns
}

// exprToString 将 AST 表达式转换为字符串
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + exprToString(e.Elt)
		}
		return "[" + exprToString(e.Len) + "]" + exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + exprToString(e.Value)
	case *ast.Ellipsis:
		return "..." + exprToString(e.Elt)
	case *ast.BasicLit:
		return e.Value
	default:
		return "any"
	}
}

// customFuncs 返回自定义模板函数
func customFuncs(data *TemplateData) template.FuncMap {
	return template.FuncMap{
		// 类型相关
		"typeName": func(t TypeRef) string { return t.TypeName },
		"fullType": func(t TypeRef) string { return t.FullType },
		"isString": func(t TypeRef) bool { return t.IsString },
		"stringVal": func(t TypeRef) string {
			if t.IsString {
				return t.StringVal
			}
			return ""
		},

		// Import 管理
		"import": func(path string) string {
			return data.Imports.Add(path)
		},
		"importAlias": func(path, alias string) string {
			return data.Imports.AddAlias(path, alias)
		},

		// 代码生成辅助
		"receiver": func(name string) string {
			if len(name) == 0 {
				return "r"
			}
			return strings.ToLower(name[:1])
		},
		"exported": func(name string) string {
			if len(name) == 0 {
				return ""
			}
			return strings.ToUpper(name[:1]) + name[1:]
		},
		"unexported": func(name string) string {
			if len(name) == 0 {
				return ""
			}
			return strings.ToLower(name[:1]) + name[1:]
		},

		// 返回类型格式化
		"formatReturns": func(returns []ReturnData) string {
			if len(returns) == 0 {
				return ""
			}
			if len(returns) == 1 {
				return returns[0].Type
			}
			// 多个返回值需要括号
			var parts []string
			for _, r := range returns {
				if r.Name != "" {
					parts = append(parts, r.Name+" "+r.Type)
				} else {
					parts = append(parts, r.Type)
				}
			}
			return "(" + strings.Join(parts, ", ") + ")"
		},

		// 参数格式化
		"formatParams": func(params []ParamData) string {
			if len(params) == 0 {
				return ""
			}
			var parts []string
			for _, p := range params {
				if p.Name != "" {
					parts = append(parts, p.Name+" "+p.Type)
				} else {
					parts = append(parts, p.Type)
				}
			}
			return strings.Join(parts, ", ")
		},
	}
}

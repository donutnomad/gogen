package settergen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/automap"
	"github.com/donutnomad/gogen/internal/gormparse"
	"github.com/donutnomad/gogen/internal/structparse"
	"github.com/donutnomad/gogen/plugin"
)

const generatorName = "settergen"

// SetterParams 定义 Setter 注解支持的参数
type SetterParams struct {
	Patch       string `param:"name=patch,required=false,default=none,description=Patch 模式: none|v2|full，支持组合如 v2|full"`
	PatchMapper string `param:"name=patch_mapper,required=false,default=ToPO,description=Patch mapper 方法名"`
	Setter      string `param:"name=setter,required=false,default=true,description=是否生成 setter 方法: true|false"`
}

// SetterGenerator 实现 plugin.Generator 接口
type SetterGenerator struct {
	plugin.BaseGenerator
}

func NewSetterGenerator() *SetterGenerator {
	gen := &SetterGenerator{
		BaseGenerator: *plugin.NewBaseGeneratorWithParamsStruct(
			generatorName,
			[]string{"Setter"},
			[]plugin.TargetKind{plugin.TargetStruct},
			SetterParams{}, // 传入参数结构体的零值实例
		),
	}
	gen.SetPriority(20) // Setter 优先级第二
	return gen
}

// Generate 执行代码生成
func (g *SetterGenerator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	result := plugin.NewGenerateResult()

	if len(ctx.Targets) == 0 {
		return result, nil
	}

	// 按输出文件分组处理
	// key: 输出路径, value: 待处理的目标列表
	fileTargets := make(map[string][]*targetInfo)

	for _, at := range ctx.Targets {
		ann := plugin.GetAnnotation(at.Annotations, "Setter")
		if ann == nil {
			continue
		}

		// 从 ParsedParams 获取解析好的参数
		var params SetterParams
		if at.ParsedParams != nil {
			var ok bool
			params, ok = at.ParsedParams.(SetterParams)
			if !ok {
				result.AddError(fmt.Errorf("ParsedParams 类型断言失败: %T", at.ParsedParams))
				continue
			}
		}

		// 跳过 patch 和 setter 都为空/none/false 的情况
		patchMode := strings.ToLower(strings.TrimSpace(params.Patch))
		setterEnabled := parseBoolParam(params.Setter)
		if (patchMode == "none" || patchMode == "") && !setterEnabled {
			if ctx.Verbose {
				fmt.Printf("[settergen] 跳过结构体 %s (patch=none, setter=false)\n", at.Target.Name)
			}
			continue
		}

		// 解析结构体
		structInfo, err := structparse.ParseStruct(at.Target.FilePath, at.Target.Name)
		if err != nil {
			result.AddError(fmt.Errorf("解析结构体 %s 失败: %w", at.Target.Name, err))
			continue
		}

		// 转换为 GORM 模型（用于获取字段信息）
		gormModel, err := gormparse.ParseGormModel(structInfo)
		if err != nil {
			result.AddError(fmt.Errorf("解析模型失败: %w", err))
			continue
		}

		// 计算输出路径
		// 优先使用注解指定的 output，否则使用包级默认文件 setter_gen.go
		fileConfig := ctx.GetFileConfig(at.Target.FilePath)
		outputPath := plugin.GetOutputPath(at.Target, ann, "$FILE_setter.go", fileConfig, g.Name(), ctx.DefaultOutput)

		// 收集 mapper 方法信息
		var mapperMethod *[2]string
		patchModeForMapper := strings.ToLower(params.Patch)
		if patchModeForMapper == "v2" {
			mapperMethod = g.processPatchMapper(filepath.Dir(at.Target.FilePath), at.Target.Name, &params)
		}

		fileTargets[outputPath] = append(fileTargets[outputPath], &targetInfo{
			model:        gormModel,
			params:       &SetterParams{Patch: params.Patch, PatchMapper: params.PatchMapper, Setter: params.Setter},
			mapperMethod: mapperMethod,
		})

		if ctx.Verbose {
			fmt.Printf("[settergen] 处理结构体 %s -> %s\n", at.Target.Name, outputPath)
		}
	}

	// 为每个输出文件生成 gg 定义
	// 按输出路径排序，确保生成顺序一致
	outputPaths := make([]string, 0, len(fileTargets))
	for outputPath := range fileTargets {
		outputPaths = append(outputPaths, outputPath)
	}
	slices.Sort(outputPaths)

	for _, outputPath := range outputPaths {
		targets := fileTargets[outputPath]
		// 按结构体名称排序，确保同一文件中不同结构体的顺序一致
		slices.SortFunc(targets, func(a, b *targetInfo) int {
			return strings.Compare(a.model.Name, b.model.Name)
		})

		if ctx.Verbose {
			for _, item := range targets {
				fmt.Printf("[settergen] %s", spew.Sdump(item.params))
			}
		}
		gen, err := g.generateDefinition(targets)
		if err != nil {
			result.AddError(fmt.Errorf("生成 %s 失败: %w", outputPath, err))
			continue
		}
		result.AddDefinition(outputPath, gen)
	}

	return result, nil
}

// targetInfo 存储单个目标的处理信息
type targetInfo struct {
	model        *gormparse.GormModelInfo
	params       *SetterParams
	mapperMethod *[2]string
}

// generateDefinition 为一组目标生成 gg 定义
func (g *SetterGenerator) generateDefinition(targets []*targetInfo) (*gg.Generator, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("没有目标需要生成")
	}

	gen := gg.New()
	gen.SetPackage(targets[0].model.PackageName)

	// 收集所有 imports（带别名支持）
	for _, t := range targets {
		for _, imp := range getSetterImports(t.model) {
			if imp.Alias != "" {
				gen.PAlias(imp.Path, imp.Alias)
			} else {
				gen.P(imp.Path)
			}
		}
	}

	// 生成代码
	for i, t := range targets {
		if i > 0 {
			gen.Body().AddLine()
		}

		// 处理 setter 参数
		if parseBoolParam(t.params.Setter) {
			// 生成 Patch 结构体和 setter 方法
			generateSetterV1(gen, t.model)
		}

		// 处理 patch 模式（支持 v2|full 多值输入）
		patchModes := strings.Split(strings.ToLower(t.params.Patch), "|")
		for _, patchMode := range patchModes {
			patchMode = strings.TrimSpace(patchMode)
			switch patchMode {
			case "v2":
				// 使用 automap 生成 ToPatch 方法
				if t.mapperMethod != nil {
					_, code, imports, err := automap.Generate2WithOptions((*t.mapperMethod)[0], "ToPatch", automap.WithFileContext((*t.mapperMethod)[1]))
					if err != nil {
						return nil, fmt.Errorf("生成 ToPatch 代码失败: %w", err)
					}
					// 添加 imports
					for _, imp := range imports {
						gen.P(imp)
					}
					gen.Body().AddString(code)
				} else {
					fmt.Printf("[settergen] 警告: 结构体 %s 的 patch=v2 模式未找到 mapper 方法 %s\n", t.model.Name, t.params.PatchMapper)
				}
			case "full":
				// 生成 ToMap 方法
				generateToMapMethod(gen, t.model)
			case "", "none":
				// 不生成
			default:
				fmt.Printf("[settergen] 警告: 结构体 %s 的 patch=%s 不支持，可选值: none|v2|full\n", t.model.Name, patchMode)
			}
		}
	}

	return gen, nil
}

// processPatchMapper 处理 patch_mapper 参数
// structName: 当前处理的结构体名称，用于查找该结构体的 mapper 方法
func (g *SetterGenerator) processPatchMapper(fileDir string, structName string, params *SetterParams) *[2]string {
	patchMapper := params.PatchMapper

	// 如果只是方法名（不含"."），在目录中查找该结构体的方法
	if !strings.Contains(patchMapper, ".") {
		method, found := findMethodInDirectory(fileDir, structName, patchMapper)
		if !found {
			fmt.Printf("[settergen] 警告: 结构体 %s 在目录 %s 中未找到方法 %s\n", structName, fileDir, patchMapper)
			return nil
		}
		return &[2]string{
			fmt.Sprintf("%s.%s", trimPtr(method.ReceiverType), method.Name),
			method.FilePath,
		}
	}

	// 解析 patch_mapper 参数 (Type.Method 格式)
	parts := strings.Split(patchMapper, ".")
	if len(parts) != 2 {
		fmt.Printf("[settergen] 警告: patch_mapper 格式错误: %s\n", patchMapper)
		return nil
	}
	targetStructName, methodName := parts[0], parts[1]

	// 在目录中查找指定结构体的方法
	method, found := findMethodInDirectory(fileDir, targetStructName, methodName)
	if !found {
		fmt.Printf("[settergen] 警告: 未找到方法 %s.%s\n", targetStructName, methodName)
		return nil
	}

	return &[2]string{
		fmt.Sprintf("%s.%s", trimPtr(method.ReceiverType), method.Name),
		method.FilePath,
	}
}

// methodInfo 存储方法信息
type methodInfo struct {
	Name         string
	ReceiverType string
	FilePath     string
}

// findMethodInDirectory 在目录中查找指定结构体的方法
func findMethodInDirectory(dir, structName, methodName string) (methodInfo, bool) {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return methodInfo{}, false
	}

	for _, filePath := range files {
		// 跳过测试文件
		if strings.HasSuffix(filePath, "_test.go") {
			continue
		}

		methods := parseAllMethodsFromFile(filePath)
		for _, method := range methods {
			if method.Name == methodName && trimPtr(method.ReceiverType) == structName {
				return method, true
			}
		}
	}

	return methodInfo{}, false
}

// parseAllMethodsFromFile 从文件中解析所有方法
func parseAllMethodsFromFile(filename string) []methodInfo {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		absPath = filename
	}

	var methods []methodInfo

	ast.Inspect(node, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// 检查是否是方法（有接收器）
		if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			return true
		}

		recv := funcDecl.Recv.List[0]
		var recvType string

		// 处理接收器类型
		if starExpr, ok := recv.Type.(*ast.StarExpr); ok {
			if ident, ok := starExpr.X.(*ast.Ident); ok {
				recvType = "*" + ident.Name
			}
		} else if ident, ok := recv.Type.(*ast.Ident); ok {
			recvType = ident.Name
		}

		if recvType != "" {
			methods = append(methods, methodInfo{
				Name:         funcDecl.Name.Name,
				ReceiverType: recvType,
				FilePath:     absPath,
			})
		}

		return true
	})

	return methods
}

// trimPtr 移除类型前的指针符号
func trimPtr(s string) string {
	return strings.TrimPrefix(s, "*")
}

// parseBoolParam 解析布尔参数，支持 true/false/1/0/t/f
func parseBoolParam(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "1", "t", "yes", "y":
		return true
	default:
		return false
	}
}

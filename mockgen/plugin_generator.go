package mockgen

import (
	"fmt"
	"path/filepath"

	"github.com/donutnomad/gogen/plugin"
)

const pluginGeneratorName = "mockgen"

// MockParams 定义 Mock 注解支持的参数
type MockParams struct {
	Output   string `param:"name=output,required=false,default=,description=输出文件路径"`
	Package  string `param:"name=package,required=false,default=,description=生成代码的包名"`
	Typed    bool   `param:"name=typed,required=false,default=true,description=是否生成类型安全的方法"`
	MockName string `param:"name=mock_name,required=false,default=,description=Mock类型名称，默认为 Mock+接口名"`
}

// MockGenerator 实现 plugin.Generator 接口
type MockGenerator struct {
	plugin.BaseGenerator
}

// NewMockGenerator 创建新的 MockGenerator
func NewMockGenerator() *MockGenerator {
	gen := &MockGenerator{
		BaseGenerator: *plugin.NewBaseGeneratorWithParamsStruct(
			pluginGeneratorName,
			[]string{"Mock"},
			[]plugin.TargetKind{plugin.TargetInterface},
			MockParams{},
		),
	}
	gen.SetPriority(40) // Mock 优先级第四
	return gen
}

// Generate 执行代码生成
func (g *MockGenerator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	result := plugin.NewGenerateResult()

	if len(ctx.Targets) == 0 {
		return result, nil
	}

	// 按输出文件分组处理
	// key: 输出路径, value: 待处理的目标列表
	fileTargets := make(map[string][]*mockTargetInfo)

	for _, at := range ctx.Targets {
		ann := plugin.GetAnnotation(at.Annotations, "Mock")
		if ann == nil {
			continue
		}

		// 从 ParsedParams 获取解析好的参数
		var params MockParams
		if at.ParsedParams != nil {
			var ok bool
			params, ok = at.ParsedParams.(MockParams)
			if !ok {
				result.AddError(fmt.Errorf("ParsedParams 类型断言失败: %T", at.ParsedParams))
				continue
			}
		}

		// 计算输出路径
		fileConfig := ctx.GetFileConfig(at.Target.FilePath)
		outputPath := plugin.GetOutputPath(at.Target, ann, "$FILE_mock.go", fileConfig, g.Name(), ctx.DefaultOutput)

		fileTargets[outputPath] = append(fileTargets[outputPath], &mockTargetInfo{
			target: at,
			params: &params,
		})

		if ctx.Verbose {
			fmt.Printf("[mockgen] 处理接口 %s -> %s\n", at.Target.Name, outputPath)
		}
	}

	// 为每个输出文件生成代码
	for outputPath, targets := range fileTargets {
		output, err := g.generateMockCode(outputPath, targets)
		if err != nil {
			result.AddError(fmt.Errorf("生成 %s 失败: %w", outputPath, err))
			continue
		}
		if len(output) == 0 {
			result.AddError(fmt.Errorf("生成 %s 失败: 输出为空", outputPath))
			continue
		}
		result.AddRawOutput(outputPath, output)
	}

	return result, nil
}

// mockTargetInfo 存储单个目标的处理信息
type mockTargetInfo struct {
	target *plugin.AnnotatedTarget
	params *MockParams
}

// generateMockCode 为一组目标生成 mock 代码
func (g *MockGenerator) generateMockCode(outputPath string, targets []*mockTargetInfo) ([]byte, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("没有目标需要生成")
	}

	// 收集所有接口名称
	var interfaceNames []string
	for _, t := range targets {
		interfaceNames = append(interfaceNames, t.target.Target.Name)
	}

	// 获取第一个目标的信息用于配置
	firstTarget := targets[0]
	sourceFile := firstTarget.target.Target.FilePath
	sourcePkgName := firstTarget.target.Target.PackageName

	// 确定输出包名
	outputPkgName := firstTarget.params.Package
	if outputPkgName == "" {
		// 检查输出目录是否与源文件目录相同
		sourceDir := filepath.Dir(sourceFile)
		outputDir := filepath.Dir(outputPath)
		if sourceDir == outputDir {
			// 同一目录下，使用源包名
			outputPkgName = sourcePkgName
		} else {
			// 不同目录，使用 mock_ + 源包名
			outputPkgName = "mock_" + sanitize(sourcePkgName)
		}
	}

	// 确定 mock 名称映射
	var mockNamesStr string
	for _, t := range targets {
		if t.params.MockName != "" {
			if mockNamesStr != "" {
				mockNamesStr += ","
			}
			mockNamesStr += t.target.Target.Name + "=" + t.params.MockName
		}
	}

	// 使用 sourceModeWithOptions 解析源文件
	opts := SourceModeOptions{
		Source:            sourceFile,
		IncludeInterfaces: interfaceNames,
	}

	pkg, err := sourceModeWithOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("解析源文件失败: %w", err)
	}

	// 确定输出包路径
	outputPackagePath := ""
	if outputPath != "" {
		dstPath, err := filepath.Abs(filepath.Dir(outputPath))
		if err == nil {
			pkgPath, err := parsePackageImport(dstPath)
			if err == nil {
				outputPackagePath = pkgPath
			}
		}
	}

	// 使用 generator 生成代码
	gen := &generator{
		writePkgComment:    true,
		writeSourceComment: true,
		typed:              firstTarget.params.Typed,
		filename:           sourceFile,
		destination:        outputPath,
	}

	if mockNamesStr != "" {
		gen.mockNames = parseMockNames(mockNamesStr)
	}

	if err := gen.Generate(pkg, outputPkgName, outputPackagePath); err != nil {
		return nil, fmt.Errorf("生成 mock 代码失败: %w", err)
	}

	return gen.Output(), nil
}

package mockgen

import (
	"cmp"
	"fmt"
	"go/token"
	"slices"
	"strings"
	"time"

	"github.com/donutnomad/gogen/plugin"
)

const pluginGeneratorName = "mockgen"

// MockParams 定义 Mock 注解支持的参数
type MockParams struct {
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

// mockTargetInfo 存储单个目标的处理信息
type mockTargetInfo struct {
	interface_ *InterfaceInfo
	params     *MockParams
	filePath   string    // 源文件路径
	position   token.Pos // 源文件中的位置
}

// Generate 执行代码生成
func (g *MockGenerator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	totalStart := time.Now()
	result := plugin.NewGenerateResult()

	if len(ctx.Targets) == 0 {
		return result, nil
	}

	// 按输出文件分组处理
	fileTargets := make(map[string][]*mockTargetInfo)

	var parseTotal time.Duration
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

		// 解析接口信息
		parseStart := time.Now()
		interfaceInfo, err := ParseInterface(at.Target.FilePath, at.Target.Name)
		parseTotal += time.Since(parseStart)
		if err != nil {
			result.AddError(fmt.Errorf("解析接口 %s 失败: %w", at.Target.Name, err))
			continue
		}

		// 计算输出路径
		fileConfig := ctx.GetFileConfig(at.Target.FilePath)
		outputPath := plugin.GetOutputPath(at.Target, ann, "$FILE_mock.go", fileConfig, g.Name(), ctx.DefaultOutput)

		fileTargets[outputPath] = append(fileTargets[outputPath], &mockTargetInfo{
			interface_: interfaceInfo,
			params:     &params,
			filePath:   at.Target.FilePath,
			position:   at.Target.Position,
		})

		if ctx.Verbose {
			fmt.Printf("[mockgen] 处理接口 %s -> %s\n", at.Target.Name, outputPath)
		}
	}

	// 为每个输出文件生成代码
	// 按输出路径排序，确保生成顺序一致
	outputPaths := make([]string, 0, len(fileTargets))
	for outputPath := range fileTargets {
		outputPaths = append(outputPaths, outputPath)
	}
	slices.Sort(outputPaths)

	var generateTotal time.Duration
	for _, outputPath := range outputPaths {
		targets := fileTargets[outputPath]
		// 按源文件路径和位置排序，保持原始顺序
		slices.SortFunc(targets, func(a, b *mockTargetInfo) int {
			// 先按文件路径排序
			if c := strings.Compare(a.filePath, b.filePath); c != 0 {
				return c
			}
			// 同一文件内按位置排序
			return cmp.Compare(a.position, b.position)
		})

		genStart := time.Now()
		gen, err := g.generateDefinition(targets)
		generateTotal += time.Since(genStart)
		if err != nil {
			result.AddError(fmt.Errorf("生成 %s 失败: %w", outputPath, err))
			continue
		}
		result.AddDefinition(outputPath, gen)
	}

	if ctx.Verbose {
		totalDur := time.Since(totalStart)
		fmt.Printf("[mockgen] 耗时统计:\n")
		fmt.Printf("  - 接口解析总耗时: %v\n", parseTotal)
		fmt.Printf("  - 代码生成总耗时: %v\n", generateTotal)
		fmt.Printf("  - 总耗时: %v\n", totalDur)
	}

	return result, nil
}

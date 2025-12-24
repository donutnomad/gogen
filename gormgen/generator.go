package gormgen

import (
	"fmt"
	"time"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/internal/gormparse"
	"github.com/donutnomad/gogen/internal/structparse"
	"github.com/donutnomad/gogen/plugin"
)

const generatorName = "gormgen"

// GsqlParams 定义 Gsql 注解支持的参数
type GsqlParams struct {
	Prefix string `param:"name=prefix,required=false,default=,description=生成的 Schema 结构体前缀"`
}

// GsqlGenerator 实现 plugin.Generator 接口
type GsqlGenerator struct {
	plugin.BaseGenerator
}

func NewGsqlGenerator() *GsqlGenerator {
	gen := &GsqlGenerator{
		BaseGenerator: *plugin.NewBaseGeneratorWithParamsStruct(
			generatorName,
			[]string{"Gsql"},
			[]plugin.TargetKind{plugin.TargetStruct},
			GsqlParams{}, // 传入参数结构体的零值实例
		),
	}
	gen.SetPriority(10) // Gsql 优先级最高
	return gen
}

// Generate 执行代码生成
func (g *GsqlGenerator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	totalStart := time.Now()
	result := plugin.NewGenerateResult()

	if len(ctx.Targets) == 0 {
		return result, nil
	}

	// 按输出文件分组处理
	// key: 输出路径, value: 待处理的目标列表
	fileTargets := make(map[string][]*targetInfo)

	var parseStructTotal, parseGormTotal time.Duration

	for _, at := range ctx.Targets {
		ann := plugin.GetAnnotation(at.Annotations, "Gsql")
		if ann == nil {
			continue
		}

		// 从 ParsedParams 获取解析好的参数
		var params GsqlParams
		if at.ParsedParams != nil {
			var ok bool
			params, ok = at.ParsedParams.(GsqlParams)
			if !ok {
				result.AddError(fmt.Errorf("ParsedParams 类型断言失败: %T", at.ParsedParams))
				continue
			}
		}

		// 解析结构体
		parseStructStart := time.Now()
		structInfo, err := structparse.ParseStruct(at.Target.FilePath, at.Target.Name)
		parseStructDur := time.Since(parseStructStart)
		parseStructTotal += parseStructDur
		if err != nil {
			result.AddError(fmt.Errorf("解析结构体 %s 失败: %w", at.Target.Name, err))
			continue
		}

		// 转换为 GORM 模型（内部会推导表名）
		parseGormStart := time.Now()
		gormModel, err := gormparse.ParseGormModel(structInfo)
		parseGormDur := time.Since(parseGormStart)
		parseGormTotal += parseGormDur
		if err != nil {
			result.AddError(fmt.Errorf("解析 GORM 模型失败: %w", err))
			continue
		}
		gormModel.Prefix = params.Prefix

		// 计算输出路径
		// 优先使用注解指定的 output，否则使用包级默认文件 generate.go
		fileConfig := ctx.GetFileConfig(at.Target.FilePath)
		outputPath := plugin.GetOutputPath(at.Target, ann, "$FILE_query.go", fileConfig, g.Name(), ctx.DefaultOutput)

		fileTargets[outputPath] = append(fileTargets[outputPath], &targetInfo{
			model:  gormModel,
			params: &params,
		})

		if ctx.Verbose {
			fmt.Printf("[gormgen] 处理结构体 %s -> %s (结构体解析: %v, GORM解析: %v)\n",
				at.Target.Name, outputPath, parseStructDur, parseGormDur)
		}
	}

	// 为每个输出文件生成 gg 定义
	var generateTotal time.Duration
	for outputPath, targets := range fileTargets {
		genStart := time.Now()
		gen, err := g.generateDefinition(targets)
		genDur := time.Since(genStart)
		generateTotal += genDur
		if err != nil {
			result.AddError(fmt.Errorf("生成 %s 失败: %w", outputPath, err))
			continue
		}
		result.AddDefinition(outputPath, gen)

		if ctx.Verbose {
			fmt.Printf("[gormgen] 生成定义 %s (耗时: %v)\n", outputPath, genDur)
		}
	}

	if ctx.Verbose {
		totalDur := time.Since(totalStart)
		fmt.Printf("[gormgen] 耗时统计:\n")
		fmt.Printf("  - 结构体解析总耗时: %v\n", parseStructTotal)
		fmt.Printf("  - GORM模型解析总耗时: %v\n", parseGormTotal)
		fmt.Printf("  - 代码生成总耗时: %v\n", generateTotal)
		fmt.Printf("  - 总耗时: %v\n", totalDur)
	}

	return result, nil
}

// targetInfo 存储单个目标的处理信息
type targetInfo struct {
	model  *gormparse.GormModelInfo
	params *GsqlParams
}

// generateDefinition 为一组目标生成 gg 定义
func (g *GsqlGenerator) generateDefinition(targets []*targetInfo) (*gg.Generator, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("没有目标需要生成")
	}

	gen := gg.New()
	gen.SetPackage(targets[0].model.PackageName)

	// 获取所需的包引用
	gsql := gen.P("github.com/donutnomad/gsql")
	field := gen.P("github.com/donutnomad/gsql/field")

	// 收集所有 imports（带别名支持）
	for _, t := range targets {
		for _, imp := range getGormQueryImports(t.model) {
			if imp.Alias != "" {
				gen.PAlias(imp.Path, imp.Alias)
			} else {
				gen.P(imp.Path)
			}
		}
	}

	// 生成 Query 代码
	for i, t := range targets {
		if i > 0 {
			gen.Body().AddLine()
		}
		generateModelCode(gen, t.model, gsql, field)
	}

	return gen, nil
}

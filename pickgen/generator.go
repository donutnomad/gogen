package pickgen

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/internal/structparse"
	"github.com/donutnomad/gogen/plugin"
)

const generatorName = "pickgen"

// SelectionMode 字段选择模式
type SelectionMode int

const (
	ModePick SelectionMode = iota // 选择指定字段
	ModeOmit                      // 排除指定字段
)

// PickParams Pick 注解参数
type PickParams struct {
	Name   string `param:"name=name,required=true,description=生成的新结构体名称"`
	Fields string `param:"name=fields,required=true,description=选择的字段列表，格式: [A,B,C]"`
	Source string `param:"name=source,required=false,description=源结构体（用于第三方包），格式: pkg.Type"`
}

// OmitParams Omit 注解参数
type OmitParams struct {
	Name   string `param:"name=name,required=true,description=生成的新结构体名称"`
	Fields string `param:"name=fields,required=true,description=排除的字段列表，格式: [X,Y]"`
	Source string `param:"name=source,required=false,description=源结构体（用于第三方包），格式: pkg.Type"`
}

// PickGenerator 实现 plugin.Generator 接口
// 同时支持 @Pick 和 @Omit 两个注解
type PickGenerator struct {
	plugin.BaseGenerator
}

// NewPickGenerator 创建 Pick 生成器
func NewPickGenerator() *PickGenerator {
	gen := &PickGenerator{
		BaseGenerator: *plugin.NewBaseGeneratorWithParamsStruct(
			generatorName,
			[]string{"Pick"},
			[]plugin.TargetKind{plugin.TargetStruct, plugin.TargetComment},
			PickParams{},
		),
	}
	gen.SetPriority(40)
	return gen
}

// OmitGenerator Omit 生成器
type OmitGenerator struct {
	plugin.BaseGenerator
}

// NewOmitGenerator 创建 Omit 生成器
func NewOmitGenerator() *OmitGenerator {
	gen := &OmitGenerator{
		BaseGenerator: *plugin.NewBaseGeneratorWithParamsStruct(
			"omitgen",
			[]string{"Omit"},
			[]plugin.TargetKind{plugin.TargetStruct, plugin.TargetComment},
			OmitParams{},
		),
	}
	gen.SetPriority(40)
	return gen
}

// targetInfo 存储单个目标的处理信息
type targetInfo struct {
	filePath       string
	packageName    string
	sourceName     string   // 源结构体名
	targetName     string   // 目标结构体名
	fields         []string // 字段列表
	mode           SelectionMode
	sourceType     string // 完整源类型（如 pkg.Type）
	sourceImport   string // 源类型的导入路径
	sourceAlias    string // 源类型的包别名
	isExternalType bool   // 是否是外部类型
}

// Generate 执行代码生成
func (g *PickGenerator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	return generatePick(ctx, ModePick)
}

// Generate 执行代码生成
func (g *OmitGenerator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	return generatePick(ctx, ModeOmit)
}

// generatePick 通用生成逻辑
func generatePick(ctx *plugin.GenerateContext, mode SelectionMode) (*plugin.GenerateResult, error) {
	result := plugin.NewGenerateResult()

	if len(ctx.Targets) == 0 {
		return result, nil
	}

	// 按输出文件分组处理
	fileTargets := make(map[string][]*targetInfo)

	annName := "Pick"
	if mode == ModeOmit {
		annName = "Omit"
	}

	for _, at := range ctx.Targets {
		ann := plugin.GetAnnotation(at.Annotations, annName)
		if ann == nil {
			continue
		}

		// 解析参数
		var targetName, fieldsStr, sourceStr string
		if mode == ModePick {
			params, ok := at.ParsedParams.(PickParams)
			if !ok {
				result.AddError(fmt.Errorf("ParsedParams 类型断言失败: %T", at.ParsedParams))
				continue
			}
			targetName = params.Name
			fieldsStr = params.Fields
			sourceStr = params.Source
		} else {
			params, ok := at.ParsedParams.(OmitParams)
			if !ok {
				result.AddError(fmt.Errorf("ParsedParams 类型断言失败: %T", at.ParsedParams))
				continue
			}
			targetName = params.Name
			fieldsStr = params.Fields
			sourceStr = params.Source
		}

		// 验证必填参数
		if targetName == "" {
			result.AddError(fmt.Errorf("[%s] 结构体 %s: name 参数是必填的", annName, at.Target.Name))
			continue
		}
		if fieldsStr == "" {
			result.AddError(fmt.Errorf("[%s] 结构体 %s: fields 参数是必填的", annName, at.Target.Name))
			continue
		}

		// 对于独立注释 (//go:gen:)，source 参数是必填的
		if at.Target.Kind == plugin.TargetComment && sourceStr == "" {
			result.AddError(fmt.Errorf("[%s] //go:gen: 注解: source 参数是必填的，用于指定源结构体", annName))
			continue
		}

		fields := parseArrayParam(fieldsStr)

		// 解析源类型
		sourceName := at.Target.Name
		sourceType := at.Target.Name
		sourceImport := ""
		sourceAlias := ""
		isExternalType := false

		if sourceStr != "" {
			pkgPath, typeName, alias, err := parseSourceParam(sourceStr, at.Target.FilePath)
			if err != nil {
				result.AddError(fmt.Errorf("[%s] 结构体 %s: 解析 source 参数失败: %w", annName, at.Target.Name, err))
				continue
			}
			sourceName = typeName
			sourceImport = pkgPath
			sourceAlias = alias
			isExternalType = pkgPath != ""
			if isExternalType {
				if alias != "" {
					sourceType = alias + "." + typeName
				} else {
					sourceType = filepath.Base(pkgPath) + "." + typeName
				}
			} else {
				sourceType = typeName
			}
		}

		// 计算输出路径
		fileConfig := ctx.GetFileConfig(at.Target.FilePath)
		outputPath := plugin.GetOutputPath(at.Target, ann, "$FILE_pick.go", fileConfig, generatorName, ctx.DefaultOutput)

		fileTargets[outputPath] = append(fileTargets[outputPath], &targetInfo{
			filePath:       at.Target.FilePath,
			packageName:    at.Target.PackageName,
			sourceName:     sourceName,
			targetName:     targetName,
			fields:         fields,
			mode:           mode,
			sourceType:     sourceType,
			sourceImport:   sourceImport,
			sourceAlias:    sourceAlias,
			isExternalType: isExternalType,
		})

		if ctx.Verbose {
			fmt.Printf("[%s] 处理结构体 %s -> %s (%s)\n", annName, at.Target.Name, targetName, outputPath)
		}
	}

	// 为每个输出文件生成 gg 定义
	outputPaths := make([]string, 0, len(fileTargets))
	for outputPath := range fileTargets {
		outputPaths = append(outputPaths, outputPath)
	}
	slices.Sort(outputPaths)

	for _, outputPath := range outputPaths {
		targets := fileTargets[outputPath]
		// 按目标结构体名称排序
		slices.SortFunc(targets, func(a, b *targetInfo) int {
			return strings.Compare(a.targetName, b.targetName)
		})

		gen, err := generateDefinition(targets)
		if err != nil {
			result.AddError(fmt.Errorf("生成 %s 失败: %w", outputPath, err))
			continue
		}
		result.AddDefinition(outputPath, gen)
	}

	return result, nil
}

// generateDefinition 为一组目标生成 gg 定义
func generateDefinition(targets []*targetInfo) (*gg.Generator, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("没有目标需要生成")
	}

	gen := gg.New()
	gen.SetPackage(targets[0].packageName)

	// 收集所有需要的导入
	imports := make(map[string]string) // path -> alias

	for _, t := range targets {
		// 解析源结构体
		var structInfo *structparse.StructInfo
		var err error

		if t.isExternalType {
			// 外部类型：需要找到包路径并解析
			structInfo, err = resolveExternalStruct(t.sourceImport, t.sourceName, t.filePath)
			if err != nil {
				return nil, fmt.Errorf("解析外部结构体 %s 失败: %w", t.sourceType, err)
			}
			// 添加外部包导入
			if t.sourceImport != "" {
				imports[t.sourceImport] = t.sourceAlias
			}
		} else {
			// 本地类型
			structInfo, err = structparse.ParseStruct(t.filePath, t.sourceName)
			if err != nil {
				return nil, fmt.Errorf("解析结构体 %s 失败: %w", t.sourceName, err)
			}
		}

		// 过滤字段
		selectedFields, err := filterFields(structInfo.Fields, t.fields, t.mode)
		if err != nil {
			return nil, err
		}

		// 收集字段类型的导入
		for _, field := range selectedFields {
			if field.PkgPath != "" {
				imports[field.PkgPath] = field.PkgAlias
			}
		}

		// 生成结构体定义
		buildStruct(gen, t.targetName, t.sourceName, t.mode, selectedFields)

		// 生成 From 方法
		buildFromMethod(gen, t.targetName, t.sourceType, selectedFields)

		// 生成构造函数
		buildNewFunction(gen, t.targetName, t.sourceType, selectedFields)
	}

	// 添加导入
	for path, alias := range imports {
		if alias != "" {
			gen.PAlias(path, alias)
		} else {
			gen.P(path)
		}
	}

	return gen, nil
}

// parseArrayParam 解析数组格式的参数 [a,b,c] -> []string
func parseArrayParam(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// 移除方括号
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")

	// 按逗号分割
	parts := strings.Split(s, ",")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

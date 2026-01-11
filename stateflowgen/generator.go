package stateflowgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/plugin"
)

const generatorName = "stateflow"

// StateFlowGenerator 状态流转代码生成器
type StateFlowGenerator struct {
	plugin.BaseGenerator
}

// NewStateFlowGenerator 创建状态流转生成器
func NewStateFlowGenerator() *StateFlowGenerator {
	return &StateFlowGenerator{
		BaseGenerator: *plugin.NewBaseGenerator(
			generatorName,
			[]string{"StateFlow"},
			[]plugin.TargetKind{plugin.TargetConst}, // 使用 const 作为注解载体
		),
	}
}

// Generate 执行代码生成
func (g *StateFlowGenerator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	result := plugin.NewGenerateResult()

	if len(ctx.Targets) == 0 {
		return result, nil
	}

	// 按文件分组处理
	// key: 文件路径, value: 注解目标列表
	fileTargets := make(map[string][]*plugin.AnnotatedTarget)

	for _, at := range ctx.Targets {
		ann := plugin.GetAnnotation(at.Annotations, "StateFlow")
		if ann == nil {
			continue
		}
		fileTargets[at.Target.FilePath] = append(fileTargets[at.Target.FilePath], at)
	}

	// 按文件路径排序，确保生成顺序一致
	filePaths := make([]string, 0, len(fileTargets))
	for fp := range fileTargets {
		filePaths = append(filePaths, fp)
	}
	slices.Sort(filePaths)

	for _, filePath := range filePaths {
		targets := fileTargets[filePath]

		// 解析文件中的所有 StateFlow 定义
		models, err := g.parseStateFlowsFromFile(filePath, targets)
		if err != nil {
			result.AddError(fmt.Errorf("解析 %s 失败: %w", filePath, err))
			continue
		}

		for _, modelInfo := range models {
			// 计算输出路径
			fileConfig := ctx.GetFileConfig(filePath)
			outputPath := plugin.GetOutputPath(modelInfo.target.Target, modelInfo.ann, "$FILE_stateflow.go", fileConfig, g.Name(), ctx.DefaultOutput)

			// 生成代码
			gen, err := g.generateCode(modelInfo.model, modelInfo.packageName)
			if err != nil {
				result.AddError(fmt.Errorf("生成 %s 代码失败: %w", modelInfo.model.Name, err))
				continue
			}

			result.AddDefinition(outputPath, gen)

			if ctx.Verbose {
				fmt.Printf("[stateflow] 处理 %s -> %s\n", modelInfo.model.Name, outputPath)
			}
		}
	}

	return result, nil
}

// modelInfo 存储解析结果
type modelInfo struct {
	model       *StateModel
	target      *plugin.AnnotatedTarget
	ann         *plugin.Annotation
	packageName string
}

// parseStateFlowsFromFile 从文件中解析所有 StateFlow 定义
func (g *StateFlowGenerator) parseStateFlowsFromFile(filePath string, targets []*plugin.AnnotatedTarget) ([]*modelInfo, error) {
	// 重新解析文件以获取完整的注释
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var models []*modelInfo

	for _, at := range targets {
		ann := plugin.GetAnnotation(at.Annotations, "StateFlow")
		if ann == nil {
			continue
		}

		// 查找包含完整注解的注释组
		commentText := g.findFullComment(file, at.Target.Position, fset)
		if commentText == "" {
			return nil, fmt.Errorf("无法找到 %s 的注释", at.Target.Name)
		}

		// 解析 StateFlow 配置和规则
		config, rules, err := ParseFlowAnnotations(commentText)
		if err != nil {
			return nil, fmt.Errorf("解析 StateFlow 注解失败: %w", err)
		}

		if config == nil {
			return nil, fmt.Errorf("未找到 @StateFlow 配置")
		}

		// 如果没有指定 name，保留为空字符串
		// 这样生成的类型名称将是 Phase, State, Stage 等，没有前缀

		// 构建模型
		model, err := BuildModel(config, rules)
		if err != nil {
			return nil, fmt.Errorf("构建状态模型失败: %w", err)
		}

		models = append(models, &modelInfo{
			model:       model,
			target:      at,
			ann:         ann,
			packageName: file.Name.Name,
		})
	}

	return models, nil
}

// findFullComment 查找目标位置的完整注释
func (g *StateFlowGenerator) findFullComment(file *ast.File, pos token.Pos, fset *token.FileSet) string {
	targetLine := fset.Position(pos).Line

	// 查找最近的注释组
	var bestComment *ast.CommentGroup
	for _, cg := range file.Comments {
		endLine := fset.Position(cg.End()).Line
		// 注释组必须在目标行之前
		if endLine < targetLine && endLine >= targetLine-20 {
			// 选择最近的注释组
			if bestComment == nil || fset.Position(cg.End()).Line > fset.Position(bestComment.End()).Line {
				bestComment = cg
			}
		}
	}

	// 也检查声明上方的 Doc
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			if fset.Position(genDecl.Pos()).Line == targetLine || fset.Position(genDecl.Pos()).Line == targetLine+1 {
				if genDecl.Doc != nil {
					return genDecl.Doc.Text()
				}
			}
		}
	}

	if bestComment != nil {
		return bestComment.Text()
	}

	return ""
}

// generateCode 生成代码
func (g *StateFlowGenerator) generateCode(model *StateModel, packageName string) (*gg.Generator, error) {
	cg := NewCodeGenerator(model, packageName)
	return cg.Generate()
}

// GetOutputPath 计算输出路径
func GetOutputPath(target *plugin.Target, ann *plugin.Annotation, defaultPattern string, fileConfig *plugin.PackageConfig, pluginName string, cmdDefault string) string {
	// 优先使用注解参数
	if ann != nil {
		if output := ann.GetParam("output"); output != "" {
			return expandOutputPath(output, target.FilePath)
		}
	}

	// 其次使用包级配置
	if fileConfig != nil {
		if output := fileConfig.GetPluginOutput(pluginName); output != "" {
			return expandOutputPath(output, target.FilePath)
		}
	}

	// 最后使用命令行默认值或默认模式
	if cmdDefault != "" {
		return expandOutputPath(cmdDefault, target.FilePath)
	}

	return expandOutputPath(defaultPattern, target.FilePath)
}

// expandOutputPath 展开输出路径中的变量
func expandOutputPath(pattern, sourcePath string) string {
	dir := filepath.Dir(sourcePath)
	base := filepath.Base(sourcePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	result := pattern
	result = strings.ReplaceAll(result, "$FILE", filepath.Join(dir, name))
	result = strings.ReplaceAll(result, "$DIR", dir)
	result = strings.ReplaceAll(result, "$NAME", name)

	return result
}

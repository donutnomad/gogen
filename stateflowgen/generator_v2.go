package stateflowgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/plugin"
)

const generatorV2Name = "stateflowv2"

type StateFlowV2Generator struct {
	plugin.BaseGenerator
}

func NewStateFlowV2Generator() *StateFlowV2Generator {
	return &StateFlowV2Generator{
		BaseGenerator: *plugin.NewBaseGenerator(
			generatorV2Name,
			[]string{"StateFlowV2"},
			[]plugin.TargetKind{plugin.TargetConst},
		),
	}
}

func (g *StateFlowV2Generator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	result := plugin.NewGenerateResult()
	if len(ctx.Targets) == 0 {
		return result, nil
	}

	fileTargets := make(map[string][]*plugin.AnnotatedTarget)
	for _, at := range ctx.Targets {
		ann := plugin.GetAnnotation(at.Annotations, "StateFlowV2")
		if ann == nil {
			continue
		}
		fileTargets[at.Target.FilePath] = append(fileTargets[at.Target.FilePath], at)
	}

	filePaths := make([]string, 0, len(fileTargets))
	for filePath := range fileTargets {
		filePaths = append(filePaths, filePath)
	}
	slices.Sort(filePaths)

	for _, filePath := range filePaths {
		models, err := g.parseStateFlowV2FromFile(filePath, fileTargets[filePath])
		if err != nil {
			result.AddError(fmt.Errorf("parse %s failed: %w", filePath, err))
			continue
		}

		for _, modelInfo := range models {
			fileConfig := ctx.GetFileConfig(filePath)
			outputPath := plugin.GetOutputPath(modelInfo.target.Target, modelInfo.ann, "$FILE_stateflow_v2.go", fileConfig, g.Name(), ctx.DefaultOutput)

			gen, err := g.generateCode(modelInfo.model, modelInfo.packageName)
			if err != nil {
				result.AddError(fmt.Errorf("generate %s failed: %w", modelInfo.model.Name, err))
				continue
			}
			result.AddDefinition(outputPath, gen)
		}
	}

	return result, nil
}

type modelV2Info struct {
	model       *StateFlowV2Model
	target      *plugin.AnnotatedTarget
	ann         *plugin.Annotation
	packageName string
}

func (g *StateFlowV2Generator) parseStateFlowV2FromFile(filePath string, targets []*plugin.AnnotatedTarget) ([]*modelV2Info, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var models []*modelV2Info
	for _, at := range targets {
		ann := plugin.GetAnnotation(at.Annotations, "StateFlowV2")
		if ann == nil {
			continue
		}

		commentText := findFullCommentV2(file, at.Target.Position, fset)
		if commentText == "" {
			return nil, fmt.Errorf("comment not found for %s", at.Target.Name)
		}

		config, rules, err := ParseFlowV2Annotations(commentText)
		if err != nil {
			return nil, err
		}
		if config == nil {
			return nil, fmt.Errorf("@StateFlowV2 config not found")
		}

		model, err := BuildStateFlowV2Model(config, rules)
		if err != nil {
			return nil, err
		}

		models = append(models, &modelV2Info{
			model:       model,
			target:      at,
			ann:         ann,
			packageName: file.Name.Name,
		})
	}

	return models, nil
}

func (g *StateFlowV2Generator) generateCode(model *StateFlowV2Model, packageName string) (*gg.Generator, error) {
	return NewCodeGeneratorV2(model, packageName).Generate()
}

func findFullCommentV2(file *ast.File, pos token.Pos, fset *token.FileSet) string {
	targetLine := fset.Position(pos).Line

	var bestComment *ast.CommentGroup
	for _, cg := range file.Comments {
		endLine := fset.Position(cg.End()).Line
		if endLine < targetLine && endLine >= targetLine-30 {
			if bestComment == nil || fset.Position(cg.End()).Line > fset.Position(bestComment.End()).Line {
				bestComment = cg
			}
		}
	}

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

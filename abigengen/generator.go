package abigengen

import (
	"fmt"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/donutnomad/eths/abigen/gogenapi"
	"github.com/donutnomad/gogen/plugin"
)

const generatorName = "abigengen"

type AbigenParams struct {
	ABI    string `param:"name=abi,required=true,description=ABI or artifact json path"`
	Output string `param:"name=output,required=false,default=,description=output Go file name"`
	Name   string `param:"name=name,required=false,default=,description=generated contract type name"`
	Pkg    string `param:"name=pkg,required=false,default=,description=generated package name"`
	Alias  string `param:"name=alias,required=false,default=,description=comma separated ABI aliases"`
}

type AbigenGenerator struct {
	plugin.BaseGenerator
}

func NewAbigenGenerator() *AbigenGenerator {
	gen := &AbigenGenerator{
		BaseGenerator: *plugin.NewBaseGeneratorWithParamsStruct(
			generatorName,
			[]string{"Abigen"},
			[]plugin.TargetKind{
				plugin.TargetStruct,
				plugin.TargetInterface,
				plugin.TargetFunc,
				plugin.TargetMethod,
				plugin.TargetVar,
				plugin.TargetConst,
				plugin.TargetComment,
			},
			AbigenParams{},
		),
	}
	gen.SetPriority(60)
	return gen
}

type targetInfo struct {
	target     *plugin.AnnotatedTarget
	params     AbigenParams
	abiPath    string
	outputPath string
}

func (g *AbigenGenerator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	result := plugin.NewGenerateResult()
	if len(ctx.Targets) == 0 {
		return result, nil
	}

	targets := make([]targetInfo, 0, len(ctx.Targets))
	for _, at := range ctx.Targets {
		ann := plugin.GetAnnotation(at.Annotations, "Abigen")
		if ann == nil {
			continue
		}
		params, ok := at.ParsedParams.(AbigenParams)
		if !ok {
			result.AddError(fmt.Errorf("[Abigen] ParsedParams type mismatch: %T", at.ParsedParams))
			continue
		}
		if params.ABI == "" {
			result.AddError(fmt.Errorf("[Abigen] %s: abi is required", at.Target.FilePath))
			continue
		}

		abiPath := resolvePath(filepath.Dir(at.Target.FilePath), params.ABI)
		input, err := gogenapi.LoadInput(abiPath)
		if err != nil {
			result.AddError(fmt.Errorf("[Abigen] load %s: %w", abiPath, err))
			continue
		}
		outputPath := resolveOutputPath(filepath.Dir(at.Target.FilePath), params.Output, input.Path)
		targets = append(targets, targetInfo{
			target:     at,
			params:     params,
			abiPath:    abiPath,
			outputPath: outputPath,
		})
	}

	slices.SortFunc(targets, func(a, b targetInfo) int {
		if a.outputPath != b.outputPath {
			return strings.Compare(a.outputPath, b.outputPath)
		}
		if a.target.Target.FilePath != b.target.Target.FilePath {
			return strings.Compare(a.target.Target.FilePath, b.target.Target.FilePath)
		}
		return int(a.target.Target.Position - b.target.Target.Position)
	})

	seenOutputs := make(map[string]bool)
	for _, t := range targets {
		if seenOutputs[t.outputPath] {
			result.AddError(fmt.Errorf("[Abigen] duplicate output path: %s", t.outputPath))
			continue
		}
		seenOutputs[t.outputPath] = true

		pkgName := t.params.Pkg
		if pkgName == "" {
			pkgName = t.target.Target.PackageName
		}
		if pkgName == "" {
			pkgName = packageNameFromDir(filepath.Dir(t.target.Target.FilePath))
		}

		code, err := gogenapi.Generate(gogenapi.GenerateOptions{
			ABIPath:     t.abiPath,
			PackageName: pkgName,
			TypeName:    t.params.Name,
			Aliases:     parseAliases(t.params.Alias),
		})
		if err != nil {
			result.AddError(fmt.Errorf("[Abigen] generate %s: %w", t.abiPath, err))
			continue
		}
		result.AddRawOutput(t.outputPath, code)
		if ctx.Verbose {
			fmt.Printf("[abigengen] %s -> %s\n", t.abiPath, t.outputPath)
		}
	}

	return result, nil
}

func resolvePath(baseDir, path string) string {
	path = strings.TrimPrefix(path, "@")
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(baseDir, path))
}

func resolveOutputPath(baseDir, output, abiPath string) string {
	if output == "" {
		output = toSnakeCase(strings.TrimSuffix(filepath.Base(abiPath), filepath.Ext(abiPath)))
	}
	if !strings.HasSuffix(output, ".go") {
		output += ".go"
	}
	if filepath.IsAbs(output) {
		return filepath.Clean(output)
	}
	return filepath.Clean(filepath.Join(baseDir, output))
}

func parseAliases(raw string) map[string]string {
	aliases := make(map[string]string)
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		var parts []string
		if strings.Contains(item, "=") {
			parts = strings.SplitN(item, "=", 2)
		} else if strings.Contains(item, ":") {
			parts = strings.SplitN(item, ":", 2)
		}
		if len(parts) != 2 {
			continue
		}
		from := strings.TrimSpace(parts[0])
		to := strings.TrimSpace(parts[1])
		if from != "" && to != "" {
			aliases[from] = to
		}
	}
	return aliases
}

func toSnakeCase(input string) string {
	var out []rune
	var prev rune
	for i, r := range input {
		if r == '-' || r == ' ' || r == '.' {
			r = '_'
		}
		if unicode.IsUpper(r) {
			if i > 0 && prev != '_' && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
				out = append(out, '_')
			}
			r = unicode.ToLower(r)
		}
		out = append(out, r)
		prev = r
	}
	return strings.Trim(strings.Join(strings.FieldsFunc(string(out), func(r rune) bool { return r == '_' }), "_"), "_")
}

func packageNameFromDir(dir string) string {
	name := filepath.Base(dir)
	var out []rune
	for i, r := range name {
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			if i == 0 && unicode.IsDigit(r) {
				out = append(out, '_')
			}
			out = append(out, r)
		}
	}
	if len(out) == 0 || !token.IsIdentifier(string(out)) {
		return "main"
	}
	return string(out)
}

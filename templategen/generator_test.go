package templategen_test

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"testing"

	"github.com/donutnomad/gogen/plugin"
)

var annotationRegex = regexp.MustCompile(`@(\w+)(?:\(([^)]*)\))?`)

func ParseAnnotationsTest(comment string) []string {
	var annotations []string
	lines := strings.Split(comment, "\n")
	for _, line := range lines {
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimSpace(line)
		matches := annotationRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			annotations = append(annotations, match[1])
		}
	}
	return annotations
}

func TestParseFile(t *testing.T) {
	filePath := "example/service.go"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	t.Logf("Package: %s", file.Name.Name)

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			t.Logf("GenDecl Tok: %v", d.Tok)
			if d.Doc != nil {
				annotations := ParseAnnotationsTest(d.Doc.Text())
				t.Logf("  Annotations: %v", annotations)
			}
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					t.Logf("  TypeSpec: %s", s.Name.Name)
				}
			}
		case *ast.FuncDecl:
			if d.Recv != nil {
				t.Logf("FuncDecl (Method): %s", d.Name.Name)
			} else {
				t.Logf("FuncDecl (Func): %s", d.Name.Name)
			}
			if d.Doc != nil {
				annotations := ParseAnnotationsTest(d.Doc.Text())
				t.Logf("  Annotations: %v", annotations)
			}
		}
	}
}

func TestQuickMatch(t *testing.T) {
	scanner := plugin.NewScanner()

	matched, err := scanner.QuickMatchFile("example/service.go")
	if err != nil {
		t.Fatalf("QuickMatchFile error: %v", err)
	}

	t.Logf("Quick match result: %v", matched)
	if !matched {
		t.Error("Expected file to match")
	}
}

func TestScan(t *testing.T) {
	scanner := plugin.NewScanner(
		plugin.WithScannerVerbose(true),
	)

	result, err := scanner.Scan(context.Background(), "./example/...")
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	t.Logf("All targets: %d", len(result.All()))
	t.Logf("Structs: %d", len(result.Structs))
	t.Logf("Methods: %d", len(result.Methods))
	t.Logf("Funcs: %d", len(result.Funcs))
	t.Logf("PackageConfigs: %d", len(result.PackageConfigs))

	for path, cfg := range result.PackageConfigs {
		t.Logf("  PackageConfig: %s", path)
		t.Logf("    DefaultOutput: %s", cfg.DefaultOutput)
		t.Logf("    PluginOutputs: %v", cfg.PluginOutputs)
	}

	for _, target := range result.All() {
		t.Logf("Target: %s (%s)", target.Target.Name, target.Target.Kind)
		for _, ann := range target.Annotations {
			t.Logf("  Annotation: %s %v", ann.Name, ann.Params)
		}
	}
}

func TestScanWithFilter(t *testing.T) {
	scanner := plugin.NewScanner(
		plugin.WithAnnotationFilter("Define", "Import"),
		plugin.WithScannerVerbose(true),
	)

	result, err := scanner.Scan(context.Background(), "./example/...")
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	t.Logf("All targets: %d", len(result.All()))
	fmt.Printf("result.All() = %+v\n", result.All())
}

func TestParseGoGenConfig(t *testing.T) {
	filePath := "example/service.go"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	t.Logf("Total comments: %d", len(file.Comments))

	goGenRegex := regexp.MustCompile(`go:gogen:\s*(.*)`)

	for i, cg := range file.Comments {
		t.Logf("CommentGroup %d:", i)
		for _, c := range cg.List {
			t.Logf("  Comment: %q", c.Text)

			// 模拟 parsePackageConfig 的逻辑
			text := strings.TrimPrefix(c.Text, "//")
			text = strings.TrimPrefix(text, "/*")
			text = strings.TrimSuffix(text, "*/")
			text = strings.TrimSpace(text)

			t.Logf("  Trimmed: %q", text)

			if matches := goGenRegex.FindStringSubmatch(text); len(matches) > 1 {
				t.Logf("  Matched! Config: %q", matches[1])
			}
		}
	}
}

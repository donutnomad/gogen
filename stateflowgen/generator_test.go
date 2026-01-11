package stateflowgen

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/donutnomad/gogen/plugin"
)

func TestParseServerTestFile(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "examples/server.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	t.Logf("File parsed successfully")
	t.Logf("Comments groups: %d", len(file.Comments))
	for i, cg := range file.Comments {
		t.Logf("Comment group %d:\n%s---", i, cg.Text())
	}

	t.Log("\nDeclarations:")
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			t.Logf("GenDecl at %v, Tok: %v", fset.Position(genDecl.Pos()), genDecl.Tok)
			if genDecl.Doc != nil {
				t.Logf("  Doc:\n%s---", genDecl.Doc.Text())
			} else {
				t.Log("  Doc: nil")
			}
		}
	}
}

func TestAnnotationParsing(t *testing.T) {
	comment := `@StateFlow(name="Server")
@Flow: Init           => [ Provisioning ]
@Flow: Provisioning   => [ Ready(Enabled), Failed ]`

	annotations := plugin.ParseAnnotations(comment)
	t.Logf("Found %d annotations", len(annotations))
	for i, ann := range annotations {
		t.Logf("  [%d] %s: %v", i, ann.Name, ann.Params)
	}

	// 检查 StateFlow 是否被正确解析
	if len(annotations) == 0 {
		t.Error("Expected at least one annotation")
		return
	}

	found := false
	for _, ann := range annotations {
		if ann.Name == "StateFlow" {
			found = true
			if ann.Params["name"] != "Server" {
				t.Errorf("Expected name=Server, got %s", ann.Params["name"])
			}
		}
	}
	if !found {
		t.Error("StateFlow annotation not found")
	}
}

func TestScannerIntegration(t *testing.T) {
	// 获取当前工作目录
	wd, _ := os.Getwd()
	t.Logf("Working directory: %s", wd)

	// 测试扫描器能否正确识别 StateFlow 注解
	scanner := plugin.NewScanner(
		plugin.WithAnnotationFilter("StateFlow"),
		plugin.WithScannerVerbose(true),
	)

	result, err := scanner.Scan(context.Background(), "./examples")
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	t.Logf("Scan result:")
	t.Logf("  Structs: %d", len(result.Structs))
	t.Logf("  Interfaces: %d", len(result.Interfaces))
	t.Logf("  Funcs: %d", len(result.Funcs))
	t.Logf("  Methods: %d", len(result.Methods))
	t.Logf("  Vars: %d", len(result.Vars))
	t.Logf("  Consts: %d", len(result.Consts))
	t.Logf("  All: %d", len(result.All()))

	for _, target := range result.Consts {
		t.Logf("  Const Target: %s (kind: %v)", target.Target.Name, target.Target.Kind)
		for _, ann := range target.Annotations {
			t.Logf("    Annotation: %s %v", ann.Name, ann.Params)
		}
	}

	if len(result.Consts) == 0 {
		t.Error("Expected at least one const target with StateFlow annotation")
	}
}

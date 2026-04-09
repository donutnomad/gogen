package stateflowgen

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
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

// TestSelfTransitionDiagramSize 测试自环转换生成的图不会指数爆炸
func TestSelfTransitionDiagramSize(t *testing.T) {
	text := `
@StateFlow(output=types_state.go)
@Flow: none => [ active? via creating else rejected ]
@Flow: active => [ inactive? via updating, (=)? via updating ]
@Flow: inactive => [ active? via updating, (=)? via updating ]
`
	config, rules, err := ParseFlowAnnotations(text)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	model, err := BuildModel(config, rules)
	if err != nil {
		t.Fatalf("Build model error: %v", err)
	}

	t.Logf("Transitions: %d", len(model.Transitions))
	for i, tr := range model.Transitions {
		t.Logf("  [%d] %s -> %s (via=%s, fallback=%s, optional=%v)", i, tr.From, tr.To, tr.Via, tr.Fallback, tr.ApprovalOptional)
	}

	cg := NewCodeGenerator(model, "automation")
	gen, err := cg.Generate()
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	output := gen.String()
	lines := strings.Split(output, "\n")

	// 统计 Flowchart 部分的行数
	chartLines := 0
	inChart := false
	for _, line := range lines {
		if strings.Contains(line, "/* Flowchart:") {
			inChart = true
		}
		if inChart {
			chartLines++
			if strings.Contains(line, "*/") {
				break
			}
		}
	}

	t.Logf("Flowchart: %d lines, total output: %d lines", chartLines, len(lines))

	// 打印 Flowchart 部分
	inChart = false
	for _, line := range lines {
		if strings.Contains(line, "/* Flowchart:") {
			inChart = true
		}
		if inChart {
			t.Log(line)
			if strings.Contains(line, "*/") {
				break
			}
		}
	}

	// 修复前是 1400+ 行，修复后应该在 100 行以内
	if chartLines > 100 {
		t.Errorf("Flowchart too large: %d lines (expected < 100), likely infinite recursion", chartLines)
	}
}

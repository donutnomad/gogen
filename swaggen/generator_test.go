package swaggen

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donutnomad/gogen/plugin"
	parsers "github.com/donutnomad/gogen/swaggen/parser"
)

func TestParseMethodAnnotationsPreservesRawLineComments(t *testing.T) {
	src := `package test

import "context"

type AutoSweepAPI interface {
	// GetAutoSweepList Get auto sweep list
	// @MID(Auth_AutoSweep_View)
	// @Meta(资源处理即可，按需处理非常的棒)
	// @GET(/autosweeps)
	GetAutoSweepList(ctx context.Context, req GetAutoSweepListReq) (GetAutoSweepListResp, error)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "api.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	iface := file.Decls[1].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.InterfaceType)
	field := iface.Methods.List[0]
	method, err := NewAnnotationParser(fset).ParseMethodAnnotations(&ast.FuncDecl{
		Name: field.Names[0],
		Type: field.Type.(*ast.FuncType),
		Doc:  field.Doc,
	})
	if err != nil {
		t.Fatalf("ParseMethodAnnotations failed: %v", err)
	}

	expected := []string{
		"// GetAutoSweepList Get auto sweep list",
		"// @MID(Auth_AutoSweep_View)",
		"// @Meta(资源处理即可，按需处理非常的棒)",
		"// @GET(/autosweeps)",
	}
	if strings.Join(method.RawComments, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("RawComments mismatch:\nwant:\n%s\ngot:\n%s", strings.Join(expected, "\n"), strings.Join(method.RawComments, "\n"))
	}
	if len(method.Def) != 2 {
		t.Fatalf("method Def count mismatch: want 2, got %d", len(method.Def))
	}
}

func TestParseMethodAnnotationsLogsUnregisteredTags(t *testing.T) {
	src := `package test

type AutoSweepAPI interface {
	// GetAutoSweepList Get auto sweep list
	// @Meta(资源处理即可，按需处理非常的棒)
	// @GET(/autosweeps)
	GetAutoSweepList()
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "api.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	iface := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.InterfaceType)
	field := iface.Methods.List[0]
	output := captureStdout(t, func() {
		_, err = NewAnnotationParser(fset).ParseMethodAnnotations(&ast.FuncDecl{
			Name: field.Names[0],
			Type: field.Type.(*ast.FuncType),
			Doc:  field.Doc,
		})
	})
	if err != nil {
		t.Fatalf("ParseMethodAnnotations failed: %v", err)
	}

	if !strings.Contains(output, "警告: 未注册注解: Meta") ||
		!strings.Contains(output, "方法: GetAutoSweepList") ||
		!strings.Contains(output, "@Meta(资源处理即可，按需处理非常的棒)") {
		t.Fatalf("missing unregistered tag log:\n%s", output)
	}
}

func TestParseInterfacePreservesRawLineComments(t *testing.T) {
	src := `package test

import "context"

// AutoSweepAPI Auto sweep management API
// @TAG(Customer-AutoSweep)
// @MID(OnlyAdmin)
// @PREFIX(/api/v1/xxx/aaa)
type AutoSweepAPI interface {
	// GetAutoSweepList Get auto sweep list
	// @GET(/autosweeps)
	GetAutoSweepList(ctx context.Context, req GetAutoSweepListReq) (GetAutoSweepListResp, error)
}
`

	dir := t.TempDir()
	filePath := filepath.Join(dir, "api.go")
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	typeSpec := file.Decls[1].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	iface, err := NewSwagGenerator().parseInterface(&plugin.AnnotatedTarget{
		Target: &plugin.Target{
			Kind:        plugin.TargetInterface,
			Name:        "AutoSweepAPI",
			PackageName: "test",
			FilePath:    filePath,
			Node:        typeSpec,
		},
	})
	if err != nil {
		t.Fatalf("parseInterface failed: %v", err)
	}

	expected := []string{
		"// AutoSweepAPI Auto sweep management API",
		"// @TAG(Customer-AutoSweep)",
		"// @MID(OnlyAdmin)",
		"// @PREFIX(/api/v1/xxx/aaa)",
	}
	if strings.Join(iface.RawComments, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("RawComments mismatch:\nwant:\n%s\ngot:\n%s", strings.Join(expected, "\n"), strings.Join(iface.RawComments, "\n"))
	}
	if len(iface.CommonDef) != 3 {
		t.Fatalf("interface CommonDef count mismatch: want 3, got %d", len(iface.CommonDef))
	}
}

func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	f()

	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}
	return buf.String()
}

func TestGenerateMethodBindingSetsGormgenComments(t *testing.T) {
	method := SwaggerMethod{
		Name: "GetAutoSweepList",
		Def: DefSlice{
			&parsers.MiddleWare{Value: []string{"Auth_AutoSweep_View"}},
			&parsers.GET{Value: "/autosweeps"},
		},
		RawComments: []string{
			"// GetAutoSweepList Get auto sweep list",
			"// @MID(Auth_AutoSweep_View)",
			"// @Meta(资源处理即可，按需处理非常的棒)",
			"// @GET(/autosweeps)",
		},
	}

	code := NewGinGenerator(&InterfaceCollection{}).generateMethodBinding(SwaggerInterface{
		Name: "AutoSweepAPI",
		RawComments: []string{
			"// AutoSweepAPI Auto sweep management API",
			"// @TAG(Customer-AutoSweep)",
			"// @MID(OnlyAdmin)",
			"// @PREFIX(/api/v1/xxx/aaa)",
		},
	}, method, CollectDef[*parsers.MiddleWare](method.Def))

	wantMethodComment := `c.Set("gormgen:methodcomment", "// GetAutoSweepList Get auto sweep list\n// @MID(Auth_AutoSweep_View)\n// @Meta(资源处理即可，按需处理非常的棒)\n// @GET(/autosweeps)")`
	wantInterfaceComment := `c.Set("gormgen:interfacecomment", "// AutoSweepAPI Auto sweep management API\n// @TAG(Customer-AutoSweep)\n// @MID(OnlyAdmin)\n// @PREFIX(/api/v1/xxx/aaa)")`
	if !strings.Contains(code, "var handlers = []gin.HandlerFunc{") ||
		!strings.Contains(code, "func(c *gin.Context)") ||
		!strings.Contains(code, wantMethodComment) ||
		!strings.Contains(code, wantInterfaceComment) ||
		strings.Contains(code, "gormgen:comment") {
		t.Fatalf("generated binding missing comment handler:\n%s", code)
	}
}

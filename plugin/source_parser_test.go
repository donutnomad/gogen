package plugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donutnomad/gg"
)

func TestParseSourceToGG(t *testing.T) {
	source := []byte(`package test

import (
	"context"
	"fmt"
)

type User struct {
	ID   int64
	Name string
}

func GetUser(ctx context.Context, id int64) (*User, error) {
	fmt.Println("getting user")
	return nil, nil
}
`)

	gen, err := ParseSourceToGG(source)
	if err != nil {
		t.Fatalf("ParseSourceToGG failed: %v", err)
	}

	if gen.PackageName() != "test" {
		t.Errorf("expected package name 'test', got '%s'", gen.PackageName())
	}

	output := string(gen.Bytes())

	// 验证包含 imports
	if !strings.Contains(output, `"context"`) {
		t.Error("expected output to contain context import")
	}
	if !strings.Contains(output, `"fmt"`) {
		t.Error("expected output to contain fmt import")
	}

	// 验证包含类型定义
	if !strings.Contains(output, "type User struct") {
		t.Error("expected output to contain User struct")
	}

	// 验证包含函数定义
	if !strings.Contains(output, "func GetUser") {
		t.Error("expected output to contain GetUser function")
	}
}

func TestParseSourceToGGWithAlias(t *testing.T) {
	source := []byte(`package test

import (
	ctx "context"
	. "fmt"
)

func Hello() {
	Println("hello")
}
`)

	gen, err := ParseSourceToGG(source)
	if err != nil {
		t.Fatalf("ParseSourceToGG failed: %v", err)
	}

	output := string(gen.Bytes())

	// 验证别名 import 被保留
	if !strings.Contains(output, `ctx "context"`) {
		t.Error("expected output to contain aliased context import")
	}

	// dot import 应该被跳过（暂不支持）
	if strings.Contains(output, `. "fmt"`) {
		t.Error("dot import should be skipped")
	}
}

// rawOutputGenerator 测试用生成器，返回原始字节输出
type rawOutputGenerator struct {
	BaseGenerator
	outputSource []byte
}

func (g *rawOutputGenerator) Generate(ctx *GenerateContext) (*GenerateResult, error) {
	result := NewGenerateResult()
	for _, target := range ctx.Targets {
		dir := filepath.Dir(target.Target.FilePath)
		outputPath := filepath.Join(dir, "generated.go")
		result.AddRawOutput(outputPath, g.outputSource)
	}
	return result, nil
}

// ggOutputGenerator 测试用生成器，返回 gg 定义
type ggOutputGenerator struct {
	BaseGenerator
	importPath  string
	importAlias string
}

func (g *ggOutputGenerator) Generate(ctx *GenerateContext) (*GenerateResult, error) {
	result := NewGenerateResult()
	for _, target := range ctx.Targets {
		gen := gg.New()
		gen.SetPackage(target.Target.PackageName)

		// 获取 PackageRef 并使用它生成调用
		var pkgRef *gg.PackageRef
		if g.importAlias != "" {
			pkgRef = gen.PAlias(g.importPath, g.importAlias)
		} else {
			pkgRef = gen.P(g.importPath)
		}

		gen.Body().NewFunction("Helper"+target.Target.Name).
			AddResult("", "string").
			AddBody(gg.Return(pkgRef.Call("Helper")))

		dir := filepath.Dir(target.Target.FilePath)
		outputPath := filepath.Join(dir, "generated.go")
		result.AddDefinition(outputPath, gen)
	}
	return result, nil
}

func TestMergeWithSamePackageNameDifferentPath(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "model.go")
	content := `package test

// @RawGen
// @GGGen
type User struct {
	ID int64
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 创建注册表
	registry := NewRegistry()

	// 生成器1: 使用原始输出，引用 github.com/foo/utils
	rawGen := &rawOutputGenerator{
		BaseGenerator: *NewBaseGenerator("rawgen", []string{"RawGen"}, []TargetKind{TargetStruct}),
		outputSource: []byte(`package test

import (
	"github.com/foo/utils"
)

func FromFoo() string {
	return utils.Helper()
}
`),
	}

	// 生成器2: 使用 gg 输出，引用 github.com/bar/utils
	ggGen := &ggOutputGenerator{
		BaseGenerator: *NewBaseGenerator("gggen", []string{"GGGen"}, []TargetKind{TargetStruct}),
		importPath:    "github.com/bar/utils",
		importAlias:   "",
	}

	if err := registry.Register(rawGen); err != nil {
		t.Fatalf("failed to register rawgen: %v", err)
	}
	if err := registry.Register(ggGen); err != nil {
		t.Fatalf("failed to register gggen: %v", err)
	}

	// 运行生成
	err := Run(context.Background(), registry, tmpDir)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// 读取生成的文件
	generatedFile := filepath.Join(tmpDir, "generated.go")
	generatedContent, err := os.ReadFile(generatedFile)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	output := string(generatedContent)
	t.Logf("Generated output:\n%s", output)

	// 验证两个包都被引用
	if !strings.Contains(output, "github.com/foo/utils") {
		t.Error("expected output to contain github.com/foo/utils import")
	}
	if !strings.Contains(output, "github.com/bar/utils") {
		t.Error("expected output to contain github.com/bar/utils import")
	}

	// 验证两个函数都存在
	if !strings.Contains(output, "FromFoo") {
		t.Error("expected output to contain FromFoo function")
	}
	if !strings.Contains(output, "HelperUser") {
		t.Error("expected output to contain HelperUser function")
	}

	// 验证包名相同但路径不同的包是否有别名
	// gg 库应该自动为同名包添加别名
	fooCount := strings.Count(output, "github.com/foo/utils")
	barCount := strings.Count(output, "github.com/bar/utils")
	if fooCount != 1 || barCount != 1 {
		t.Errorf("expected each import to appear exactly once, foo=%d, bar=%d", fooCount, barCount)
	}
}

func TestMergeMultipleRawOutputs(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "model.go")
	content := `package test

// @Gen1
// @Gen2
type User struct {
	ID int64
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 创建注册表
	registry := NewRegistry()

	// 生成器1: 引用 context 和 fmt
	gen1 := &rawOutputGenerator{
		BaseGenerator: *NewBaseGenerator("gen1", []string{"Gen1"}, []TargetKind{TargetStruct}),
		outputSource: []byte(`package test

import (
	"context"
	"fmt"
)

func Func1(ctx context.Context) {
	fmt.Println("func1")
}
`),
	}

	// 生成器2: 也引用 context 和 errors
	gen2 := &rawOutputGenerator{
		BaseGenerator: *NewBaseGenerator("gen2", []string{"Gen2"}, []TargetKind{TargetStruct}),
		outputSource: []byte(`package test

import (
	"context"
	"errors"
)

func Func2(ctx context.Context) error {
	return errors.New("func2")
}
`),
	}

	if err := registry.Register(gen1); err != nil {
		t.Fatalf("failed to register gen1: %v", err)
	}
	if err := registry.Register(gen2); err != nil {
		t.Fatalf("failed to register gen2: %v", err)
	}

	// 运行生成
	err := Run(context.Background(), registry, tmpDir)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// 读取生成的文件
	generatedFile := filepath.Join(tmpDir, "generated.go")
	generatedContent, err := os.ReadFile(generatedFile)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	output := string(generatedContent)
	t.Logf("Generated output:\n%s", output)

	// 验证所有 imports 都存在且去重
	contextCount := strings.Count(output, `"context"`)
	if contextCount != 1 {
		t.Errorf("expected context import to appear exactly once, got %d", contextCount)
	}

	if !strings.Contains(output, `"fmt"`) {
		t.Error("expected output to contain fmt import")
	}
	if !strings.Contains(output, `"errors"`) {
		t.Error("expected output to contain errors import")
	}

	// 验证两个函数都存在
	if !strings.Contains(output, "Func1") {
		t.Error("expected output to contain Func1 function")
	}
	if !strings.Contains(output, "Func2") {
		t.Error("expected output to contain Func2 function")
	}
}

func TestParseSourceToGGPreservesComments(t *testing.T) {
	// 测试大块注释（被注释掉的代码）是否被保留
	source := []byte(`package test

import "fmt"

func Hello() {
	fmt.Println("Hello")
}

//
//func onGinBind(c *gin.Context, val any, typ string) bool {
//    switch typ {
//    case "JSON":
//        if err := c.ShouldBindJSON(val); err != nil {
//            c.JSON(400, gin.H{"error": err.Error()})
//            return false
//        }
//    case "FORM":
//        if err := c.ShouldBind(val); err != nil {
//            c.JSON(400, gin.H{"error": err.Error()})
//            return false
//        }
//    default:
//        if err := c.ShouldBind(val); err != nil {
//            c.JSON(400, gin.H{"error": err.Error()})
//            return false
//        }
//    }
//    return true
//}
//
//func onGinResponse[T any](c *gin.Context, data any, err error) {
//    c.JSON(200, data)
//}
//
`)

	gen, err := ParseSourceToGG(source)
	if err != nil {
		t.Fatalf("ParseSourceToGG failed: %v", err)
	}

	output := string(gen.Bytes())

	// 验证被注释掉的函数被保留
	if !strings.Contains(output, "onGinBind") {
		t.Error("expected output to contain commented-out onGinBind function")
	}
	if !strings.Contains(output, "onGinResponse") {
		t.Error("expected output to contain commented-out onGinResponse function")
	}
	if !strings.Contains(output, "ShouldBindJSON") {
		t.Error("expected output to contain ShouldBindJSON in comments")
	}
	if !strings.Contains(output, "ShouldBind") {
		t.Error("expected output to contain ShouldBind in comments")
	}
}

func TestParseSourceToGGPreservesInterDeclarationComments(t *testing.T) {
	// 测试声明之间的注释是否被保留
	source := []byte(`package test

import "fmt"

type User struct {
	ID   int64
	Name string
}

// 下面是一些被注释掉的旧代码
// type OldUser struct {
//     ID int
// }

// 这是一个辅助函数的文档注释
func Helper() {
	fmt.Println("helper")
}

//
// 文件末尾的注释块
// 包含多行内容
//
`)

	gen, err := ParseSourceToGG(source)
	if err != nil {
		t.Fatalf("ParseSourceToGG failed: %v", err)
	}

	output := string(gen.Bytes())

	// 验证声明之间的注释被保留
	if !strings.Contains(output, "下面是一些被注释掉的旧代码") {
		t.Error("expected output to contain inter-declaration comments")
	}
	if !strings.Contains(output, "OldUser") {
		t.Error("expected output to contain commented-out OldUser struct")
	}
	if !strings.Contains(output, "文件末尾的注释块") {
		t.Error("expected output to contain trailing comments")
	}
}

func TestParseSourceToGGWithNoImports(t *testing.T) {
	// 测试没有 import 的文件
	source := []byte(`package test

type Simple struct {
	Value int
}

// 这是一个注释
// 多行注释
`)

	gen, err := ParseSourceToGG(source)
	if err != nil {
		t.Fatalf("ParseSourceToGG failed: %v", err)
	}

	output := string(gen.Bytes())

	if !strings.Contains(output, "type Simple struct") {
		t.Error("expected output to contain Simple struct")
	}
	if !strings.Contains(output, "这是一个注释") {
		t.Error("expected output to contain trailing comments")
	}
}

package plugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donutnomad/gg"
)

func TestParseAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "simple annotation",
			input:    "// @Gsql",
			expected: 1,
		},
		{
			name:     "annotation with params",
			input:    "// @Gsql(prefix=`L`, patch=`v2`)",
			expected: 1,
		},
		{
			name:     "multiple annotations",
			input:    "// @Gsql @Mapper",
			expected: 2,
		},
		{
			name:     "multiline annotations",
			input:    "// @Gsql(prefix=`L`)\n// @Mapper(to=`UserDTO`)",
			expected: 2,
		},
		{
			name:     "no annotation",
			input:    "// This is a comment",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := ParseAnnotations(tt.input)
			if len(annotations) != tt.expected {
				t.Errorf("expected %d annotations, got %d", tt.expected, len(annotations))
			}
		})
	}
}

func TestAnnotationParams(t *testing.T) {
	input := "// @Gsql(prefix=`L`, patch=`v2`, patch_mapper=`User.ToPO`)"
	annotations := ParseAnnotations(input)

	if len(annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(annotations))
	}

	ann := annotations[0]
	if ann.Name != "Gsql" {
		t.Errorf("expected name 'Gsql', got '%s'", ann.Name)
	}

	if ann.GetParam("prefix") != "L" {
		t.Errorf("expected prefix 'L', got '%s'", ann.GetParam("prefix"))
	}

	if ann.GetParam("patch") != "v2" {
		t.Errorf("expected patch 'v2', got '%s'", ann.GetParam("patch"))
	}

	if ann.GetParam("patch_mapper") != "User.ToPO" {
		t.Errorf("expected patch_mapper 'User.ToPO', got '%s'", ann.GetParam("patch_mapper"))
	}
}

func TestAnnotationParamsWithoutQuotes(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedParams map[string]string
	}{
		{
			name:  "普通格式多参数（逗号分隔）",
			input: "// @Setter(setter=true, patch=full)",
			expectedParams: map[string]string{
				"setter": "true",
				"patch":  "full",
			},
		},
		{
			name:  "普通格式无空格",
			input: "// @Setter(setter=true,patch=full)",
			expectedParams: map[string]string{
				"setter": "true",
				"patch":  "full",
			},
		},
		{
			name:  "混合格式（反引号和普通）",
			input: "// @Setter(setter=`true`, patch=full)",
			expectedParams: map[string]string{
				"setter": "true",
				"patch":  "full",
			},
		},
		{
			name:  "三个参数普通格式",
			input: "// @Setter(setter=true, patch=v2|full, mapper=ToPO)",
			expectedParams: map[string]string{
				"setter": "true",
				"patch":  "v2|full",
				"mapper": "ToPO",
			},
		},
		{
			name:  "布尔值1",
			input: "// @Setter(enabled=1, disabled=0)",
			expectedParams: map[string]string{
				"enabled":  "1",
				"disabled": "0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := ParseAnnotations(tt.input)
			if len(annotations) != 1 {
				t.Fatalf("expected 1 annotation, got %d", len(annotations))
			}

			ann := annotations[0]
			for key, expected := range tt.expectedParams {
				actual := ann.GetParam(key)
				if actual != expected {
					t.Errorf("param %s: expected '%s', got '%s'", key, expected, actual)
				}
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// 创建测试生成器
	gen1 := &testGenerator{
		BaseGenerator: *NewBaseGenerator("gen1", []string{"Gsql"}, []TargetKind{TargetStruct}),
	}
	gen2 := &testGenerator{
		BaseGenerator: *NewBaseGenerator("gen2", []string{"Mapper"}, []TargetKind{TargetStruct, TargetMethod}),
	}

	// 注册
	if err := registry.Register(gen1); err != nil {
		t.Fatalf("failed to register gen1: %v", err)
	}
	if err := registry.Register(gen2); err != nil {
		t.Fatalf("failed to register gen2: %v", err)
	}

	// 检查注册
	if !registry.IsRegistered("Gsql") {
		t.Error("Gsql should be registered")
	}
	if !registry.IsRegistered("Mapper") {
		t.Error("Mapper should be registered")
	}

	// 测试重复注册
	gen3 := &testGenerator{
		BaseGenerator: *NewBaseGenerator("gen3", []string{"Gsql"}, []TargetKind{TargetStruct}),
	}
	if err := registry.Register(gen3); err == nil {
		t.Error("should fail when registering duplicate annotation")
	}

	// 测试获取
	if gen, ok := registry.GetByAnnotation("Gsql"); !ok || gen.Name() != "gen1" {
		t.Error("should get gen1 by annotation Gsql")
	}
}

func TestScanner(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package test

// @Gsql(prefix=` + "`L`" + `)
type User struct {
	ID   uint
	Name string
}

// @Mapper(to=` + "`UserDTO`" + `)
type UserService interface {
	GetUser(id uint) *User
}

// @Cache(ttl=` + "`60s`" + `)
func GetUserByID(id uint) *User {
	return nil
}

// @Log
func (u *User) Save() error {
	return nil
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 扫描
	scanner := NewScanner()
	result, err := scanner.Scan(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// 验证结果
	if len(result.Structs) != 1 {
		t.Errorf("expected 1 struct, got %d", len(result.Structs))
	}
	if len(result.Interfaces) != 1 {
		t.Errorf("expected 1 interface, got %d", len(result.Interfaces))
	}
	if len(result.Funcs) != 1 {
		t.Errorf("expected 1 func, got %d", len(result.Funcs))
	}
	if len(result.Methods) != 1 {
		t.Errorf("expected 1 method, got %d", len(result.Methods))
	}

	// 验证结构体注解
	if len(result.Structs) > 0 {
		s := result.Structs[0]
		if s.Target.Name != "User" {
			t.Errorf("expected struct name 'User', got '%s'", s.Target.Name)
		}
		if s.Target.Kind != TargetStruct {
			t.Errorf("expected kind TargetStruct, got %v", s.Target.Kind)
		}
		ann := GetAnnotation(s.Annotations, "Gsql")
		if ann == nil {
			t.Error("expected Gsql annotation")
		} else if ann.GetParam("prefix") != "L" {
			t.Errorf("expected prefix 'L', got '%s'", ann.GetParam("prefix"))
		}
	}

	// 验证方法
	if len(result.Methods) > 0 {
		m := result.Methods[0]
		if m.Target.Name != "Save" {
			t.Errorf("expected method name 'Save', got '%s'", m.Target.Name)
		}
		if m.Target.ReceiverType != "*User" {
			t.Errorf("expected receiver type '*User', got '%s'", m.Target.ReceiverType)
		}
	}
}

func TestScannerWithFilter(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package test

// @Gsql
type User struct {}

// @Mapper
type Order struct {}

// @Cache
func GetUser() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 带过滤器扫描
	scanner := NewScanner(WithAnnotationFilter("Gsql"))
	result, err := scanner.Scan(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// 应该只有 Gsql 注解的结构体
	if len(result.Structs) != 1 {
		t.Errorf("expected 1 struct with Gsql annotation, got %d", len(result.Structs))
	}
	if len(result.Funcs) != 0 {
		t.Errorf("expected 0 funcs (Cache filtered out), got %d", len(result.Funcs))
	}
}

func TestScannerRecursive(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// 创建根目录文件
	rootFile := filepath.Join(tmpDir, "root.go")
	rootContent := `package root
// @Gsql
type RootModel struct {}
`
	if err := os.WriteFile(rootFile, []byte(rootContent), 0644); err != nil {
		t.Fatalf("failed to write root file: %v", err)
	}

	// 创建子目录文件
	subFile := filepath.Join(subDir, "sub.go")
	subContent := `package sub
// @Gsql
type SubModel struct {}
`
	if err := os.WriteFile(subFile, []byte(subContent), 0644); err != nil {
		t.Fatalf("failed to write sub file: %v", err)
	}

	// 使用 ./... 语法递归扫描
	scanner := NewScanner()
	result, err := scanner.Scan(context.Background(), tmpDir+"/...")
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// 应该找到两个结构体
	if len(result.Structs) != 2 {
		t.Errorf("expected 2 structs, got %d", len(result.Structs))
	}
}

// testGenerator 测试用生成器
type testGenerator struct {
	BaseGenerator
}

func (g *testGenerator) Generate(ctx *GenerateContext) (*GenerateResult, error) {
	return NewGenerateResult(), nil
}

// ggTestGenerator 测试 gg 定义返回的生成器
type ggTestGenerator struct {
	BaseGenerator
}

func (g *ggTestGenerator) Generate(ctx *GenerateContext) (*GenerateResult, error) {
	result := NewGenerateResult()

	for _, target := range ctx.Targets {
		// 为每个目标创建一个 gg 定义
		gen := gg.New()
		gen.SetPackage(target.Target.PackageName)

		// 生成一个简单的查询函数
		gen.Body().NewFunction("Query"+target.Target.Name).
			AddResult("", "string").
			AddBody(gg.Return(gg.Lit("querying " + target.Target.Name)))

		// 输出到同一目录下的 _query.go 文件
		dir := filepath.Dir(target.Target.FilePath)
		outputPath := filepath.Join(dir, strings.ToLower(target.Target.Name)+"_query.go")
		result.AddDefinition(outputPath, gen)
	}

	return result, nil
}

func TestGeneratorWithGGDefinition(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "model.go")
	content := `package test

// @TestGen
type User struct {
	ID   uint
	Name string
}

// @TestGen
type Order struct {
	ID     uint
	Amount float64
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 创建注册表并注册生成器
	registry := NewRegistry()
	gen := &ggTestGenerator{
		BaseGenerator: *NewBaseGenerator("testgen", []string{"TestGen"}, []TargetKind{TargetStruct}),
	}
	if err := registry.Register(gen); err != nil {
		t.Fatalf("failed to register generator: %v", err)
	}

	// 运行生成
	err := Run(context.Background(), registry, "", tmpDir)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// 验证生成的文件
	userQueryFile := filepath.Join(tmpDir, "user_query.go")
	if _, err := os.Stat(userQueryFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", userQueryFile)
	} else {
		content, _ := os.ReadFile(userQueryFile)
		if !strings.Contains(string(content), "QueryUser") {
			t.Errorf("expected QueryUser function in generated file")
		}
		if !strings.Contains(string(content), "Code generated by gogen") {
			t.Errorf("expected header comment in generated file")
		}
	}

	orderQueryFile := filepath.Join(tmpDir, "order_query.go")
	if _, err := os.Stat(orderQueryFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", orderQueryFile)
	} else {
		content, _ := os.ReadFile(orderQueryFile)
		if !strings.Contains(string(content), "QueryOrder") {
			t.Errorf("expected QueryOrder function in generated file")
		}
	}
}

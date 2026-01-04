package codegen_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donutnomad/gogen/codegen"
	"github.com/donutnomad/gogen/plugin"
)

func TestCodegenBasic(t *testing.T) {
	// 测试基本功能
	testCodegen(t, "testdata/basic.go", true, 0)
}

func TestCodegenEdge(t *testing.T) {
	// 测试边缘情况（所有 gRPC 和 HTTP 状态码）
	testCodegen(t, "testdata/edge.go", true, 0)
}

func TestCodegenInvalid(t *testing.T) {
	// 测试无效场景，应该有错误
	testCodegen(t, "testdata/invalid.go", false, 6)
}

func TestCodegenDuplicate(t *testing.T) {
	// 测试重复检测，应该有错误
	testCodegen(t, "testdata/duplicate.go", false, 2)
}

func TestCodegenReuse(t *testing.T) {
	// 测试 reuse=true 功能，允许重复使用相同的 code 值
	testCodegen(t, "testdata/reuse.go", true, 0)
}

func TestCodegenCrossPackage(t *testing.T) {
	// 测试不同包可以使用相同的 code 值
	ctx := context.Background()

	gen := codegen.NewCodeGenerator()
	scanner := plugin.NewScanner(plugin.WithAnnotationFilter("Code"))

	// 扫描 pkg1
	absPath1, _ := filepath.Abs("testdata/pkg1")
	result1, err := scanner.Scan(ctx, absPath1)
	if err != nil {
		t.Fatalf("扫描 pkg1 失败: %v", err)
	}

	// 扫描 pkg2
	absPath2, _ := filepath.Abs("testdata/pkg2")
	result2, err := scanner.Scan(ctx, absPath2)
	if err != nil {
		t.Fatalf("扫描 pkg2 失败: %v", err)
	}

	// 合并目标
	allTargets := append(result1.All(), result2.All()...)

	// 解析参数
	parseParams(t, gen, allTargets)

	// 生成代码
	genCtx := &plugin.GenerateContext{
		Targets:        allTargets,
		PackageConfigs: make(map[string]*plugin.PackageConfig),
		DefaultOutput:  "",
		Verbose:        testing.Verbose(),
	}

	genResult, err := gen.Generate(genCtx)
	if err != nil {
		t.Fatalf("生成代码失败: %v", err)
	}

	// 不应该有错误（不同包可以使用相同的 code）
	if len(genResult.Errors) > 0 {
		t.Errorf("跨包使用相同 code 值应该允许，但得到错误: %v", genResult.Errors)
	}

	// 应该生成2个文件（每个包一个）
	if len(genResult.Definitions) != 2 {
		t.Errorf("应该生成2个文件，实际: %d", len(genResult.Definitions))
	}
}

func TestCodegenMixed(t *testing.T) {
	// 测试混合场景（error 和 const，包括分组声明）
	testCodegen(t, "testdata/mixed.go", true, 0)
}

func TestCodegenGroupedVarConst(t *testing.T) {
	// 测试分组 var/const 声明中每个变量的注解
	ctx := context.Background()
	gen := codegen.NewCodeGenerator()
	scanner := plugin.NewScanner(plugin.WithAnnotationFilter("Code"))

	absPath, err := filepath.Abs("testdata/grouped.go")
	if err != nil {
		t.Fatalf("获取绝对路径失败: %v", err)
	}

	result, err := scanner.Scan(ctx, absPath)
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}

	all := result.All()
	// 应该识别出 5 个 var + 2 个 const = 7 个带注解的目标
	// (RecordNotFound 和 CodeNoContent 没有注解，不应该被包含)
	if len(all) != 7 {
		t.Errorf("期望 7 个带注解的目标，实际: %d", len(all))
		for _, at := range all {
			t.Logf("  - %s (%s)", at.Target.Name, at.Target.Kind)
		}
	}

	// 检查是否包含预期的变量
	expected := map[string]bool{
		"ErrNotFound":          false,
		"ErrBadRequest":        false,
		"ErrOperateNotAllowed": false,
		"ErrorForbidden":       false,
		"ErrServer":            false,
		"CodeSuccess":          false,
		"CodeCreated":          false,
	}

	for _, at := range all {
		if _, ok := expected[at.Target.Name]; ok {
			expected[at.Target.Name] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("缺少预期的目标: %s", name)
		}
	}

	// 不应该包含没有注解的变量
	for _, at := range all {
		if at.Target.Name == "RecordNotFound" || at.Target.Name == "CodeNoContent" {
			t.Errorf("不应该包含没有注解的目标: %s", at.Target.Name)
		}
	}

	// 解析参数并生成代码
	parseParams(t, gen, all)

	genCtx := &plugin.GenerateContext{
		Targets:        all,
		PackageConfigs: result.PackageConfigs,
		DefaultOutput:  "",
		Verbose:        testing.Verbose(),
	}

	genResult, err := gen.Generate(genCtx)
	if err != nil {
		t.Fatalf("生成代码失败: %v", err)
	}

	if len(genResult.Errors) > 0 {
		t.Errorf("生成过程中有错误: %v", genResult.Errors)
	}

	if len(genResult.Definitions) == 0 {
		t.Error("期望生成定义，但没有")
	}

	// 验证生成的代码包含所有目标
	for path, def := range genResult.Definitions {
		t.Logf("生成文件: %s", path)
		code := def.String()

		for name := range expected {
			if !strings.Contains(code, name) {
				t.Errorf("生成的代码缺少 %s", name)
			}
		}
	}
}

func TestCodegenGroupedMergeAndSkip(t *testing.T) {
	// 测试分组声明中只使用每个变量自己的注解
	// var ( 上方的注解不起作用
	ctx := context.Background()
	scanner := plugin.NewScanner(plugin.WithAnnotationFilter("Code", "GlobalTag"))

	absPath, err := filepath.Abs("testdata/grouped_merge.go")
	if err != nil {
		t.Fatalf("获取绝对路径失败: %v", err)
	}

	result, err := scanner.Scan(ctx, absPath)
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}

	all := result.All()

	// 应该识别出 3 个目标：
	// 1. ErrWithSpec - 分组声明中有 spec 注解（上方注释）
	// 2. ErrLineComment - 分组声明中有行尾注释注解
	// 3. ErrSingleLine - 单行声明使用 decl 注解
	// 不应该包含：
	// - ErrNoAnnotation - 分组声明中无注解
	// - ErrNoSpecAnnotation - 分组声明上方的注解不起作用
	if len(all) != 3 {
		t.Errorf("期望 3 个带注解的目标，实际: %d", len(all))
		for _, at := range all {
			t.Logf("  - %s (%s)", at.Target.Name, at.Target.Kind)
		}
	}

	// 检查具体目标
	foundWithSpec := false
	foundLineComment := false
	foundSingleLine := false

	for _, at := range all {
		switch at.Target.Name {
		case "ErrWithSpec":
			foundWithSpec = true
			// 应该只有 Code 注解（GlobalTag 在 var ( 上方，不起作用）
			if len(at.Annotations) != 1 || at.Annotations[0].Name != "Code" {
				t.Errorf("ErrWithSpec 应该只有 Code 注解, 实际: %v", at.Annotations)
			}
		case "ErrLineComment":
			foundLineComment = true
			// 应该有 Code 注解（来自行尾注释）
			if len(at.Annotations) != 1 || at.Annotations[0].Name != "Code" {
				t.Errorf("ErrLineComment 应该有 Code 注解, 实际: %v", at.Annotations)
			}
		case "ErrSingleLine":
			foundSingleLine = true
			// 应该有 Code 注解
			if len(at.Annotations) != 1 || at.Annotations[0].Name != "Code" {
				t.Errorf("ErrSingleLine 应该有 Code 注解, 实际: %v", at.Annotations)
			}
		case "ErrNoAnnotation":
			t.Errorf("ErrNoAnnotation 不应该被包含（分组声明中无注解）")
		case "ErrNoSpecAnnotation":
			t.Errorf("ErrNoSpecAnnotation 不应该被包含（分组声明上方的注解不起作用）")
		}
	}

	if !foundWithSpec {
		t.Error("缺少 ErrWithSpec")
	}
	if !foundLineComment {
		t.Error("缺少 ErrLineComment（行尾注释注解未被识别）")
	}
	if !foundSingleLine {
		t.Error("缺少 ErrSingleLine")
	}
}

func testCodegen(t *testing.T, file string, expectSuccess bool, expectedErrors int) {
	t.Helper()

	ctx := context.Background()
	gen := codegen.NewCodeGenerator()
	scanner := plugin.NewScanner(plugin.WithAnnotationFilter("Code"))

	absPath, err := filepath.Abs(file)
	if err != nil {
		t.Fatalf("获取绝对路径失败: %v", err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skipf("测试文件不存在: %s", absPath)
		return
	}

	result, err := scanner.Scan(ctx, absPath)
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}

	if len(result.All()) == 0 {
		t.Skipf("没有找到带注解的目标: %s", file)
		return
	}

	t.Logf("找到 %d 个带注解的目标", len(result.All()))

	// 解析参数
	parseParams(t, gen, result.All())

	// 生成代码
	genCtx := &plugin.GenerateContext{
		Targets:        result.All(),
		PackageConfigs: result.PackageConfigs,
		DefaultOutput:  "",
		Verbose:        testing.Verbose(),
	}

	genResult, err := gen.Generate(genCtx)
	if err != nil {
		if expectSuccess {
			t.Fatalf("生成代码失败: %v", err)
		} else {
			t.Logf("生成代码失败 (预期): %v", err)
			return
		}
	}

	// 检查错误数量
	if len(genResult.Errors) != expectedErrors {
		t.Errorf("期望 %d 个错误，实际: %d", expectedErrors, len(genResult.Errors))
		for _, e := range genResult.Errors {
			t.Logf("  错误: %v", e)
		}
	}

	// 检查是否成功
	if expectSuccess {
		if len(genResult.Errors) > 0 {
			t.Errorf("期望成功，但有错误: %v", genResult.Errors)
		}
		if len(genResult.Definitions) == 0 {
			t.Error("期望生成定义，但没有")
		}
	}

	// 验证生成的代码
	if len(genResult.Definitions) > 0 {
		for path, def := range genResult.Definitions {
			t.Logf("生成文件: %s", path)
			code := def.String()

			// 基本验证
			if !strings.Contains(code, "GetHttpCode") {
				t.Error("生成的代码缺少 GetHttpCode 方法")
			}
			if !strings.Contains(code, "GetGrpcCode") {
				t.Error("生成的代码缺少 GetGrpcCode 方法")
			}
			if !strings.Contains(code, "GetCode") {
				t.Error("生成的代码缺少 GetCode 方法")
			}
			if !strings.Contains(code, "GetName") {
				t.Error("生成的代码缺少 GetName 方法")
			}
			if !strings.Contains(code, "_codegen_getInfo") {
				t.Error("生成的代码缺少 _codegen_getInfo 方法")
			}
			if !strings.Contains(code, "_codegen_equal") {
				t.Error("生成的代码缺少 _codegen_equal 方法")
			}
		}
	}
}

func parseParams(t *testing.T, gen *codegen.CodeGenerator, targets []*plugin.AnnotatedTarget) {
	t.Helper()

	paramDefs := plugin.ParseParamsFromStruct(codegen.CodeParams{})

	for _, at := range targets {
		ann := plugin.GetAnnotation(at.Annotations, "Code")
		if ann != nil {
			paramsProto := codegen.CodeParams{}
			if err := plugin.ParseAnnotationParams(ann, &paramsProto, paramDefs); err != nil {
				t.Logf("解析参数失败 (可能预期): %v", err)
				continue
			}
			at.ParsedParams = paramsProto
		}
	}
}

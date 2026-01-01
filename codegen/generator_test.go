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

package pickgen

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/donutnomad/gogen/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDispatchMultipleAnnotations 测试框架对多个同名注解的分发
func TestDispatchMultipleAnnotations(t *testing.T) {
	// 创建一个新的 registry 并注册生成器
	registry := plugin.NewRegistry()
	registry.Register(NewPickGenerator())
	registry.Register(NewOmitGenerator())

	// 打印已注册的生成器
	fmt.Println("已注册的注解:")
	for _, ann := range registry.Annotations() {
		fmt.Printf("  - %s\n", ann)
	}

	// 创建一个模拟的扫描结果，User 有两个 @Pick 注解
	result := &plugin.ScanResult{
		Structs: []*plugin.AnnotatedTarget{
			{
				Target: &plugin.Target{
					Kind:        plugin.TargetStruct,
					Name:        "User",
					PackageName: "testpkg",
					FilePath:    "/test/user.go",
				},
				Annotations: []*plugin.Annotation{
					{
						Name: "Pick",
						Params: map[string]string{
							"name":   "UserBasic",
							"fields": "[ID,Name]",
						},
					},
					{
						Name: "Pick",
						Params: map[string]string{
							"name":   "UserProfile",
							"fields": "[ID,Name,Email]",
						},
					},
				},
			},
		},
	}

	// 分发目标
	dispatch := registry.DispatchTargets(result)

	// 打印分发结果
	fmt.Println("分发结果:")
	for genName, targets := range dispatch {
		fmt.Printf("  - %s: %d 个 targets\n", genName, len(targets))
	}

	// pickgen 应该收到 2 个 targets（每个 @Pick 注解一个）
	pickgenTargets := dispatch["pickgen"]
	require.Len(t, pickgenTargets, 2, "pickgen 应该收到 2 个 targets")

	// 验证每个 target 只有一个注解
	for i, target := range pickgenTargets {
		require.Len(t, target.Annotations, 1, "target %d 应该只有一个注解", i)
	}

	// 验证两个 targets 的注解参数不同
	names := make(map[string]bool)
	for _, target := range pickgenTargets {
		name := target.Annotations[0].GetParam("name")
		names[name] = true
	}
	assert.Len(t, names, 2, "应该有两个不同的 name 参数")
	assert.True(t, names["UserBasic"], "应该包含 UserBasic")
	assert.True(t, names["UserProfile"], "应该包含 UserProfile")
}

// TestParseParams 测试参数解析
func TestParseParams(t *testing.T) {
	// 创建一个新的 registry 并注册生成器
	registry := plugin.NewRegistry()
	pickGen := NewPickGenerator()
	registry.Register(pickGen)

	// 创建一个模拟的扫描结果，User 有两个 @Pick 注解
	result := &plugin.ScanResult{
		Structs: []*plugin.AnnotatedTarget{
			{
				Target: &plugin.Target{
					Kind:        plugin.TargetStruct,
					Name:        "User",
					PackageName: "testpkg",
					FilePath:    "/test/user.go",
				},
				Annotations: []*plugin.Annotation{
					{
						Name: "Pick",
						Params: map[string]string{
							"name":   "UserBasic",
							"fields": "[ID,Name]",
						},
					},
					{
						Name: "Pick",
						Params: map[string]string{
							"name":   "UserProfile",
							"fields": "[ID,Name,Email]",
						},
					},
				},
			},
		},
	}

	// 分发目标
	dispatch := registry.DispatchTargets(result)
	targets := dispatch["pickgen"]

	// 手动解析参数（模拟 run.go 的逻辑）
	paramDefs := pickGen.ParamDefs()
	for _, target := range targets {
		paramsProto := pickGen.NewParams()
		if paramsProto == nil {
			continue
		}

		// 找到目标的注解
		var targetAnn *plugin.Annotation
		for _, ann := range target.Annotations {
			for _, supportedAnn := range pickGen.Annotations() {
				if ann.Name == supportedAnn {
					targetAnn = ann
					break
				}
			}
			if targetAnn != nil {
				break
			}
		}

		if targetAnn != nil {
			err := plugin.ParseAnnotationParams(targetAnn, paramsProto, paramDefs)
			require.NoError(t, err)

			val := reflect.ValueOf(paramsProto)
			if val.Kind() == reflect.Ptr {
				target.ParsedParams = val.Elem().Interface()
			}
		}
	}

	// 验证解析后的参数
	fmt.Println("解析后的参数:")
	names := make(map[string]bool)
	for _, target := range targets {
		params, ok := target.ParsedParams.(PickParams)
		require.True(t, ok, "ParsedParams 应该是 PickParams 类型")
		fmt.Printf("  - Name: %s, Fields: %s\n", params.Name, params.Fields)
		names[params.Name] = true
	}

	assert.Len(t, names, 2, "应该有两个不同的 name")
	assert.True(t, names["UserBasic"], "应该包含 UserBasic")
	assert.True(t, names["UserProfile"], "应该包含 UserProfile")
}

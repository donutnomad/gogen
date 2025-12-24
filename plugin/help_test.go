package plugin

import (
	"strings"
	"testing"
)

// mockGenerator 用于测试的 mock 生成器
type mockGenerator struct {
	BaseGenerator
}

func (m *mockGenerator) Generate(ctx *GenerateContext) (*GenerateResult, error) {
	return NewGenerateResult(), nil
}

func newMockGenerator(name string, annotations []string, targets []TargetKind, params []ParamDef) *mockGenerator {
	return &mockGenerator{
		BaseGenerator: *NewBaseGeneratorWithParams(name, annotations, targets, params),
	}
}

func TestFormatHelpText(t *testing.T) {
	// 创建一个测试注册表
	registry := NewRegistry()

	// 创建测试生成器
	gen := newMockGenerator(
		"test-generator",
		[]string{"TestAnnotation"},
		[]TargetKind{TargetStruct},
		[]ParamDef{
			{Name: "param1", Required: true, Default: "", Description: "Required parameter"},
			{Name: "param2", Required: false, Default: "default_value", Description: "Optional parameter"},
		},
	)

	// 注册生成器
	if err := registry.Register(gen); err != nil {
		t.Fatalf("Failed to register generator: %v", err)
	}

	// 生成帮助文本
	helpText := FormatHelpText(registry)

	// 验证帮助文本包含预期内容
	expectedContents := []string{
		"@TestAnnotation",
		"test-generator",
		"output",
		"param1 (必填)",
		"param2",
		"[默认: default_value]",
		"Required parameter",
		"Optional parameter",
		"示例:",
		"@TestAnnotation",
		"output=$FILE_query.go",
		"output=$PACKAGE_query.go",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(helpText, expected) {
			t.Errorf("Help text should contain '%s', got:\n%s", expected, helpText)
		}
	}
}

func TestFormatHelpText_MultipleGenerators(t *testing.T) {
	registry := NewRegistry()

	// 注册多个生成器
	gen1 := newMockGenerator(
		"generator1",
		[]string{"Ann1"},
		[]TargetKind{TargetStruct},
		[]ParamDef{
			{Name: "param1", Required: false, Default: "", Description: "Param 1"},
		},
	)

	gen2 := newMockGenerator(
		"generator2",
		[]string{"Ann2"},
		[]TargetKind{TargetInterface},
		[]ParamDef{
			{Name: "param2", Required: true, Default: "", Description: "Param 2"},
		},
	)

	registry.Register(gen1)
	registry.Register(gen2)

	helpText := FormatHelpText(registry)

	// 验证两个生成器都在帮助文本中
	if !strings.Contains(helpText, "@Ann1") {
		t.Error("Help text should contain @Ann1")
	}
	if !strings.Contains(helpText, "@Ann2") {
		t.Error("Help text should contain @Ann2")
	}
	if !strings.Contains(helpText, "generator1") {
		t.Error("Help text should contain generator1")
	}
	if !strings.Contains(helpText, "generator2") {
		t.Error("Help text should contain generator2")
	}
}

func TestFormatHelpText_EmptyRegistry(t *testing.T) {
	registry := NewRegistry()
	helpText := FormatHelpText(registry)

	expected := "(暂无已注册的生成器)"
	if !strings.Contains(helpText, expected) {
		t.Errorf("Expected '%s', got: %s", expected, helpText)
	}
}

func TestFormatParamDef(t *testing.T) {
	tests := []struct {
		name     string
		param    ParamDef
		expected []string // 预期包含的字符串片段
	}{
		{
			name: "required param",
			param: ParamDef{
				Name:        "test",
				Required:    true,
				Default:     "",
				Description: "Test parameter",
			},
			expected: []string{"test", "required", "Test parameter"},
		},
		{
			name: "optional param with default",
			param: ParamDef{
				Name:        "opt",
				Required:    false,
				Default:     "default",
				Description: "Optional param",
			},
			expected: []string{"opt", "optional", "default=default", "Optional param"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatParamDef(tt.param)
			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("FormatParamDef() should contain '%s', got: %s", exp, result)
				}
			}
		})
	}
}

package plugin

import (
	"testing"
)

func TestParseParamsFromStruct(t *testing.T) {
	type TestParams struct {
		Field1 string `param:"name=field1,required=true,default=,description=Field 1 description"`
		Field2 string `param:"name=field2,required=false,default=default_value,description=Field 2 description"`
		Field3 string `param:"name=field3,required=false,default=,description=Field 3 with comma"`
		Field4 string // 没有tag,应该被忽略
	}

	params := ParseParamsFromStruct(TestParams{})

	if len(params) != 3 {
		t.Errorf("Expected 3 params, got %d", len(params))
	}

	// 验证第一个参数
	if params[0].Name != "field1" {
		t.Errorf("Expected name 'field1', got '%s'", params[0].Name)
	}
	if !params[0].Required {
		t.Error("Expected field1 to be required")
	}
	if params[0].Description != "Field 1 description" {
		t.Errorf("Expected description 'Field 1 description', got '%s'", params[0].Description)
	}

	// 验证第二个参数
	if params[1].Name != "field2" {
		t.Errorf("Expected name 'field2', got '%s'", params[1].Name)
	}
	if params[1].Required {
		t.Error("Expected field2 to not be required")
	}
	if params[1].Default != "default_value" {
		t.Errorf("Expected default 'default_value', got '%s'", params[1].Default)
	}

	// 验证第三个参数(包含逗号)
	if params[2].Name != "field3" {
		t.Errorf("Expected name 'field3', got '%s'", params[2].Name)
	}
	if params[2].Description != "Field 3 with comma" {
		t.Errorf("Expected description 'Field 3 with comma', got '%s'", params[2].Description)
	}
}

func TestParseParamsFromStruct_EmptyStruct(t *testing.T) {
	type EmptyParams struct{}

	params := ParseParamsFromStruct(EmptyParams{})

	if len(params) != 0 {
		t.Errorf("Expected 0 params, got %d", len(params))
	}
}

func TestParseParamsFromStruct_Pointer(t *testing.T) {
	type TestParams struct {
		Field1 string `param:"name=field1,required=true,default=,description=Test field"`
	}

	params := ParseParamsFromStruct(&TestParams{})

	if len(params) != 1 {
		t.Errorf("Expected 1 param, got %d", len(params))
	}

	if params[0].Name != "field1" {
		t.Errorf("Expected name 'field1', got '%s'", params[0].Name)
	}
}

func TestParseAnnotationParams(t *testing.T) {
	type TestParams struct {
		Mode   string `param:"name=mode,required=false,default=none,description=生成模式"`
		Patch  string `param:"name=patch,required=false,default=none,description=Patch模式"`
		Count  int    `param:"name=count,required=false,default=10,description=数量"`
		Enable bool   `param:"name=enable,required=false,default=false,description=启用"`
	}

	tests := []struct {
		name       string
		comment    string
		wantMode   string
		wantPatch  string
		wantCount  int
		wantEnable bool
	}{
		{
			name:       "反引号格式",
			comment:    "// @Test(mode=`v1`)",
			wantMode:   "v1",
			wantPatch:  "none",
			wantCount:  10,
			wantEnable: false,
		},
		{
			name:       "双引号格式",
			comment:    `// @Test(mode="v2")`,
			wantMode:   "v2",
			wantPatch:  "none",
			wantCount:  10,
			wantEnable: false,
		},
		{
			name:       "普通格式",
			comment:    "// @Test(mode=v3)",
			wantMode:   "v3",
			wantPatch:  "none",
			wantCount:  10,
			wantEnable: false,
		},
		{
			name:       "多个参数",
			comment:    "// @Test(mode=`v1`, patch=`v2`, count=`20`, enable=`true`)",
			wantMode:   "v1",
			wantPatch:  "v2",
			wantCount:  20,
			wantEnable: true,
		},
		{
			name:       "无参数使用默认值",
			comment:    "// @Test()",
			wantMode:   "none",
			wantPatch:  "none",
			wantCount:  10,
			wantEnable: false,
		},
	}

	paramDefs := ParseParamsFromStruct(TestParams{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := ParseAnnotations(tt.comment)
			if len(annotations) == 0 {
				t.Fatal("未解析到注解")
			}

			var params TestParams
			err := ParseAnnotationParams(annotations[0], &params, paramDefs)
			if err != nil {
				t.Fatalf("解析参数失败: %v", err)
			}

			if params.Mode != tt.wantMode {
				t.Errorf("Mode = %q, want %q", params.Mode, tt.wantMode)
			}
			if params.Patch != tt.wantPatch {
				t.Errorf("Patch = %q, want %q", params.Patch, tt.wantPatch)
			}
			if params.Count != tt.wantCount {
				t.Errorf("Count = %d, want %d", params.Count, tt.wantCount)
			}
			if params.Enable != tt.wantEnable {
				t.Errorf("Enable = %v, want %v", params.Enable, tt.wantEnable)
			}
		})
	}
}

func TestBaseGenerator_NewParams(t *testing.T) {
	type TestParams struct {
		Mode  string `param:"name=mode,required=false,default=none,description=生成模式"`
		Count int    `param:"name=count,required=false,default=10,description=数量"`
	}

	gen := NewBaseGeneratorWithParamsStruct(
		"test",
		[]string{"Test"},
		[]TargetKind{TargetStruct},
		TestParams{},
	)

	// 测试 NewParams 返回新实例
	p1 := gen.NewParams()
	p2 := gen.NewParams()

	if p1 == nil {
		t.Fatal("NewParams 返回 nil")
	}
	if p2 == nil {
		t.Fatal("NewParams 返回 nil")
	}

	// 验证返回的是指针类型
	params1, ok := p1.(*TestParams)
	if !ok {
		t.Fatalf("NewParams 返回类型错误: %T", p1)
	}
	params2, ok := p2.(*TestParams)
	if !ok {
		t.Fatalf("NewParams 返回类型错误: %T", p2)
	}

	// 验证是不同的实例
	if params1 == params2 {
		t.Error("NewParams 应该返回不同的实例")
	}

	// 验证可以修改实例
	params1.Mode = "v1"
	params1.Count = 20
	if params2.Mode != "" || params2.Count != 0 {
		t.Error("修改一个实例不应该影响另一个实例")
	}
}

func TestParseAnnotationParams_WithNewParams(t *testing.T) {
	type TestParams struct {
		Mode  string `param:"name=mode,required=false,default=none,description=生成模式"`
		Patch string `param:"name=patch,required=false,default=none,description=Patch模式"`
	}

	gen := NewBaseGeneratorWithParamsStruct(
		"test",
		[]string{"Test"},
		[]TargetKind{TargetStruct},
		TestParams{},
	)

	// 模拟 run.go 中的使用方式
	paramsProto := gen.NewParams()
	if paramsProto == nil {
		t.Fatal("NewParams 返回 nil")
	}

	comment := "// @Test(mode=`v1`, patch=`v2`)"
	annotations := ParseAnnotations(comment)
	if len(annotations) == 0 {
		t.Fatal("未解析到注解")
	}

	// 直接传入 paramsProto（已经是指针）
	err := ParseAnnotationParams(annotations[0], paramsProto, gen.ParamDefs())
	if err != nil {
		t.Fatalf("解析参数失败: %v", err)
	}

	// 验证参数已被正确设置
	params, ok := paramsProto.(*TestParams)
	if !ok {
		t.Fatalf("类型断言失败: %T", paramsProto)
	}

	if params.Mode != "v1" {
		t.Errorf("Mode = %q, want %q", params.Mode, "v1")
	}
	if params.Patch != "v2" {
		t.Errorf("Patch = %q, want %q", params.Patch, "v2")
	}
}

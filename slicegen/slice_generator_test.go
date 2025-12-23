package slicegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/donutnomad/gogen/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSliceGenerator(t *testing.T) {
	g := NewSliceGenerator()
	assert.NotNil(t, g)
	assert.Equal(t, "slicegen", g.Name())
	assert.Equal(t, []string{"Slice"}, g.Annotations())
	assert.Equal(t, []plugin.TargetKind{plugin.TargetStruct}, g.SupportedTargets())
}

func TestParseArrayParam(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]bool{},
		},
		{
			name:     "single value",
			input:    "[a]",
			expected: map[string]bool{"a": true},
		},
		{
			name:     "multiple values",
			input:    "[a,b,c]",
			expected: map[string]bool{"a": true, "b": true, "c": true},
		},
		{
			name:     "values with spaces",
			input:    "[ a , b , c ]",
			expected: map[string]bool{"a": true, "b": true, "c": true},
		},
		{
			name:     "without brackets",
			input:    "a,b,c",
			expected: map[string]bool{"a": true, "b": true, "c": true},
		},
		{
			name:     "field names",
			input:    "[ID,Name,Email]",
			expected: map[string]bool{"ID": true, "Name": true, "Email": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseArrayParam(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseArrayParamToSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "[filter]",
			expected: []string{"filter"},
		},
		{
			name:     "multiple values",
			input:    "[filter,map,sort]",
			expected: []string{"filter", "map", "sort"},
		},
		{
			name:     "values with spaces",
			input:    "[ filter , map , sort ]",
			expected: []string{"filter", "map", "sort"},
		},
		{
			name:     "all methods",
			input:    "[filter,map,reduce,sort,groupby]",
			expected: []string{"filter", "map", "reduce", "sort", "groupby"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseArrayParamToSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseBoolParam(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue bool
		expected     bool
	}{
		{"true", "true", false, true},
		{"false", "false", true, false},
		{"1", "1", false, true},
		{"0", "0", true, false},
		{"yes", "yes", false, true},
		{"no", "no", true, false},
		{"empty with default true", "", true, true},
		{"empty with default false", "", false, false},
		{"TRUE uppercase", "TRUE", false, true},
		{"FALSE uppercase", "FALSE", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBoolParam(tt.input, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMethodImpl(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		hasImpl bool
	}{
		{"filter", "filter", true},
		{"map", "map", true},
		{"reduce", "reduce", true},
		{"sort", "sort", true},
		{"groupby", "groupby", true},
		{"unknown", "unknown", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			impl := getMethodImpl(tt.method)
			if tt.hasImpl {
				assert.NotNil(t, impl, "expected implementation for method %s", tt.method)
			} else {
				assert.Nil(t, impl, "expected no implementation for method %s", tt.method)
			}
		})
	}
}

func TestSliceGeneratorGenerate(t *testing.T) {
	g := NewSliceGenerator()

	testdataDir, err := filepath.Abs("testdata")
	require.NoError(t, err)

	tests := []struct {
		name           string
		structName     string
		params         SliceParams
		expectContains []string
		expectMissing  []string
	}{
		{
			name:       "basic user slice",
			structName: "User",
			params:     SliceParams{Ptr: "true"},
			expectContains: []string{
				"type UserSlice []*User",
				"func (s UserSlice) ID()",
				"func (s UserSlice) Name()",
				"func (s UserSlice) Email()",
			},
		},
		{
			name:       "with exclude fields",
			structName: "Product",
			params:     SliceParams{Exclude: "[CreatedAt,UpdatedAt]", Ptr: "true"},
			expectContains: []string{
				"type ProductSlice []*Product",
				"func (s ProductSlice) ID()",
				"func (s ProductSlice) Name()",
			},
			expectMissing: []string{
				"func (s ProductSlice) CreatedAt()",
				"func (s ProductSlice) UpdatedAt()",
			},
		},
		{
			name:       "with include fields",
			structName: "Customer",
			params:     SliceParams{Include: "[ID,Name,Email]", Ptr: "true"},
			expectContains: []string{
				"type CustomerSlice []*Customer",
				"func (s CustomerSlice) ID()",
				"func (s CustomerSlice) Name()",
				"func (s CustomerSlice) Email()",
			},
			expectMissing: []string{
				"func (s CustomerSlice) Phone()",
				"func (s CustomerSlice) Address()",
			},
		},
		{
			name:       "non-pointer type",
			structName: "Order",
			params:     SliceParams{Ptr: "false"},
			expectContains: []string{
				"type OrderSlice []Order",
			},
			expectMissing: []string{
				"type OrderSlice []*Order",
			},
		},
		{
			name:       "with extra methods",
			structName: "Article",
			params:     SliceParams{Ptr: "true", Methods: "[filter,map,sort]"},
			expectContains: []string{
				"type ArticleSlice []*Article",
				"func (s ArticleSlice) Filter(",
				"func (s ArticleSlice) Map(",
				"func (s ArticleSlice) Sort(",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 构建目标
			target := &plugin.AnnotatedTarget{
				Target: &plugin.Target{
					Kind:        plugin.TargetStruct,
					Name:        tt.structName,
					PackageName: "testdata",
					FilePath:    filepath.Join(testdataDir, "models.go"),
				},
				Annotations: []*plugin.Annotation{
					{Name: "Slice", Params: map[string]string{
						"exclude": tt.params.Exclude,
						"include": tt.params.Include,
						"ptr":     tt.params.Ptr,
						"methods": tt.params.Methods,
					}},
				},
				ParsedParams: tt.params,
			}

			ctx := &plugin.GenerateContext{
				Targets: []*plugin.AnnotatedTarget{target},
				Verbose: false,
			}

			result, err := g.Generate(ctx)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Empty(t, result.Errors, "expected no errors, got: %v", result.Errors)

			// 检查生成的定义
			require.Len(t, result.Definitions, 1, "expected 1 definition")

			// 获取生成的代码
			var generatedCode string
			for _, def := range result.Definitions {
				generatedCode = string(def.Bytes())
			}

			// 检查期望包含的内容
			for _, expected := range tt.expectContains {
				assert.True(t, strings.Contains(generatedCode, expected),
					"expected generated code to contain %q, but it doesn't.\nGenerated code:\n%s", expected, generatedCode)
			}

			// 检查期望不包含的内容
			for _, missing := range tt.expectMissing {
				assert.False(t, strings.Contains(generatedCode, missing),
					"expected generated code NOT to contain %q, but it does.\nGenerated code:\n%s", missing, generatedCode)
			}
		})
	}
}

func TestSliceGeneratorEmptyTargets(t *testing.T) {
	g := NewSliceGenerator()

	ctx := &plugin.GenerateContext{
		Targets: []*plugin.AnnotatedTarget{},
		Verbose: false,
	}

	result, err := g.Generate(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Definitions)
	assert.Empty(t, result.Errors)
}

func TestSliceParams(t *testing.T) {
	g := NewSliceGenerator()

	// 测试参数定义
	paramDefs := g.ParamDefs()
	require.Len(t, paramDefs, 4, "expected 4 param definitions")

	paramNames := make(map[string]bool)
	for _, p := range paramDefs {
		paramNames[p.Name] = true
	}

	assert.True(t, paramNames["exclude"], "expected 'exclude' param")
	assert.True(t, paramNames["include"], "expected 'include' param")
	assert.True(t, paramNames["ptr"], "expected 'ptr' param")
	assert.True(t, paramNames["methods"], "expected 'methods' param")
}

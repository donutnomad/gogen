package pickgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/donutnomad/gogen/internal/structparse"
	"github.com/donutnomad/gogen/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试字段过滤逻辑

func TestFilterFields_Pick(t *testing.T) {
	fields := []structparse.FieldInfo{
		{Name: "ID", Type: "int64", Tag: "`json:\"id\"`"},
		{Name: "Name", Type: "string", Tag: "`json:\"name\"`"},
		{Name: "Email", Type: "string", Tag: "`json:\"email\"`"},
		{Name: "Password", Type: "string", Tag: "`json:\"-\"`"},
	}

	result, err := filterFields(fields, []string{"ID", "Name"}, ModePick)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "ID", result[0].Name)
	assert.Equal(t, "Name", result[1].Name)
}

func TestFilterFields_Omit(t *testing.T) {
	fields := []structparse.FieldInfo{
		{Name: "ID", Type: "int64"},
		{Name: "Name", Type: "string"},
		{Name: "Email", Type: "string"},
		{Name: "Password", Type: "string"},
	}

	result, err := filterFields(fields, []string{"Password"}, ModeOmit)
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "ID", result[0].Name)
	assert.Equal(t, "Name", result[1].Name)
	assert.Equal(t, "Email", result[2].Name)
}

func TestFilterFields_InvalidField(t *testing.T) {
	fields := []structparse.FieldInfo{
		{Name: "ID", Type: "int64"},
		{Name: "Name", Type: "string"},
	}

	_, err := filterFields(fields, []string{"NotExist"}, ModePick)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "字段不存在")
	assert.Contains(t, err.Error(), "NotExist")
}

// 测试参数解析

func TestParseArrayParam(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"[ID,Name,Email]", []string{"ID", "Name", "Email"}},
		{"[ID, Name, Email]", []string{"ID", "Name", "Email"}},
		{"[ID]", []string{"ID"}},
		{"[]", nil},
		{"", nil},
		{"ID,Name", []string{"ID", "Name"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseArrayParam(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// 测试 source 参数解析

func TestParseSourceParam_FullPath(t *testing.T) {
	pkgPath, typeName, alias, err := parseSourceParam("github.com/user/repo/pkg.User", "")
	require.NoError(t, err)
	assert.Equal(t, "github.com/user/repo/pkg", pkgPath)
	assert.Equal(t, "User", typeName)
	assert.Equal(t, "pkg", alias)
}

func TestParseSourceParam_LocalType(t *testing.T) {
	pkgPath, typeName, alias, err := parseSourceParam("User", "")
	require.NoError(t, err)
	assert.Equal(t, "", pkgPath)
	assert.Equal(t, "User", typeName)
	assert.Equal(t, "", alias)
}

func TestParseSourceParam_Empty(t *testing.T) {
	_, _, _, err := parseSourceParam("", "")
	require.Error(t, err)
}

// 测试代码生成

func TestGenerateDefinition(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "user.go")

	err := os.WriteFile(testFile, []byte(`package testpkg

type User struct {
	ID       int64  `+"`json:\"id\"`"+`
	Name     string `+"`json:\"name\"`"+`
	Email    string `+"`json:\"email\"`"+`
	Password string `+"`json:\"-\"`"+`
}
`), 0644)
	require.NoError(t, err)

	targets := []*targetInfo{
		{
			filePath:       testFile,
			packageName:    "testpkg",
			sourceName:     "User",
			targetName:     "UserBasic",
			fields:         []string{"ID", "Name"},
			mode:           ModePick,
			sourceType:     "User",
			isExternalType: false,
		},
	}

	gen, err := generateDefinition(targets)
	require.NoError(t, err)
	require.NotNil(t, gen)

	// 生成代码
	code := gen.Bytes()

	codeStr := string(code)

	// 验证生成的代码包含预期内容
	assert.Contains(t, codeStr, "type UserBasic struct")
	assert.Contains(t, codeStr, "ID")
	assert.Contains(t, codeStr, "Name")
	assert.NotContains(t, codeStr, "Email")
	assert.NotContains(t, codeStr, "Password")

	// 验证生成了 From 方法
	assert.Contains(t, codeStr, "func (t *UserBasic)From(src *User)")

	// 验证生成了构造函数
	assert.Contains(t, codeStr, "func NewUserBasic(src *User)")
}

// 测试完整生成器流程

func TestPickGenerator_Generate(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "model.go")

	err := os.WriteFile(testFile, []byte(`package testpkg

// @Pick(name=UserBasic, fields=[ID,Name])
type User struct {
	ID       int64
	Name     string
	Email    string
	Password string
}
`), 0644)
	require.NoError(t, err)

	// 创建生成器
	gen := NewPickGenerator()

	// 构造上下文
	ctx := &plugin.GenerateContext{
		Targets: []*plugin.AnnotatedTarget{
			{
				Target: &plugin.Target{
					Kind:        plugin.TargetStruct,
					Name:        "User",
					PackageName: "testpkg",
					FilePath:    testFile,
				},
				Annotations: []*plugin.Annotation{
					{
						Name: "Pick",
						Params: map[string]string{
							"name":   "UserBasic",
							"fields": "[ID,Name]",
						},
					},
				},
				ParsedParams: PickParams{
					Name:   "UserBasic",
					Fields: "[ID,Name]",
				},
			},
		},
	}

	// 执行生成
	result, err := gen.Generate(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Errors)
	assert.Len(t, result.Definitions, 1)
}

func TestOmitGenerator_Generate(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "model.go")

	err := os.WriteFile(testFile, []byte(`package testpkg

// @Omit(name=UserPublic, fields=[Password])
type User struct {
	ID       int64
	Name     string
	Email    string
	Password string
}
`), 0644)
	require.NoError(t, err)

	// 创建生成器
	gen := NewOmitGenerator()

	// 构造上下文
	ctx := &plugin.GenerateContext{
		Targets: []*plugin.AnnotatedTarget{
			{
				Target: &plugin.Target{
					Kind:        plugin.TargetStruct,
					Name:        "User",
					PackageName: "testpkg",
					FilePath:    testFile,
				},
				Annotations: []*plugin.Annotation{
					{
						Name: "Omit",
						Params: map[string]string{
							"name":   "UserPublic",
							"fields": "[Password]",
						},
					},
				},
				ParsedParams: OmitParams{
					Name:   "UserPublic",
					Fields: "[Password]",
				},
			},
		},
	}

	// 执行生成
	result, err := gen.Generate(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Errors)
}

// 测试多个同名注解的支持
func TestMultipleAnnotations(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "model.go")

	err := os.WriteFile(testFile, []byte(`package testpkg

type User struct {
	ID       int64
	Name     string
	Email    string
	Password string
}
`), 0644)
	require.NoError(t, err)

	// 创建生成器
	gen := NewPickGenerator()

	// 模拟框架为每个注解创建独立的 AnnotatedTarget
	ctx := &plugin.GenerateContext{
		Targets: []*plugin.AnnotatedTarget{
			{
				Target: &plugin.Target{
					Kind:        plugin.TargetStruct,
					Name:        "User",
					PackageName: "testpkg",
					FilePath:    testFile,
				},
				Annotations: []*plugin.Annotation{
					{
						Name: "Pick",
						Params: map[string]string{
							"name":   "UserBasic",
							"fields": "[ID,Name]",
						},
					},
				},
				ParsedParams: PickParams{
					Name:   "UserBasic",
					Fields: "[ID,Name]",
				},
			},
			{
				Target: &plugin.Target{
					Kind:        plugin.TargetStruct,
					Name:        "User",
					PackageName: "testpkg",
					FilePath:    testFile,
				},
				Annotations: []*plugin.Annotation{
					{
						Name: "Pick",
						Params: map[string]string{
							"name":   "UserProfile",
							"fields": "[ID,Name,Email]",
						},
					},
				},
				ParsedParams: PickParams{
					Name:   "UserProfile",
					Fields: "[ID,Name,Email]",
				},
			},
		},
	}

	// 执行生成
	result, err := gen.Generate(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Errors)
	assert.Len(t, result.Definitions, 1)

	// 验证生成的代码包含两个不同的结构体
	for _, def := range result.Definitions {
		code := string(def.Bytes())
		assert.Contains(t, code, "type UserBasic struct")
		assert.Contains(t, code, "type UserProfile struct")
	}
}

package pickgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/donutnomad/gogen/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// sanitizeAlias 测试
// ============================================================

func TestSanitizeAlias(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "普通包名",
			input:    "models",
			expected: "models",
		},
		{
			name:     "带连字符的包名",
			input:    "special-pkg",
			expected: "specialpkg",
		},
		{
			name:     "带多个连字符",
			input:    "my-cool-package",
			expected: "mycoolpackage",
		},
		{
			name:     "版本化包名",
			input:    "v2-api",
			expected: "v2api",
		},
		{
			name:     "数字开头",
			input:    "123pkg",
			expected: "_123pkg",
		},
		{
			name:     "带点的名称",
			input:    "gorm.io",
			expected: "gormio",
		},
		{
			name:     "带下划线",
			input:    "my_package",
			expected: "my_package",
		},
		{
			name:     "大写字母",
			input:    "MyPackage",
			expected: "MyPackage",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
		{
			name:     "纯数字",
			input:    "123",
			expected: "_123",
		},
		{
			name:     "特殊字符混合",
			input:    "pkg-v2.0-beta",
			expected: "pkgv20beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeAlias(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================
// parseSourceParam 测试
// ============================================================

func TestParseSourceParam_FullPathVariants(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		wantPkgPath  string
		wantTypeName string
		wantAlias    string
		wantErr      bool
	}{
		{
			name:         "GitHub 完整路径",
			source:       "github.com/user/repo/pkg.User",
			wantPkgPath:  "github.com/user/repo/pkg",
			wantTypeName: "User",
			wantAlias:    "pkg",
		},
		{
			name:         "带连字符的路径",
			source:       "github.com/user/my-repo/special-pkg.Config",
			wantPkgPath:  "github.com/user/my-repo/special-pkg",
			wantTypeName: "Config",
			wantAlias:    "specialpkg",
		},
		{
			name:         "版本化路径",
			source:       "github.com/user/repo/v2-api.Request",
			wantPkgPath:  "github.com/user/repo/v2-api",
			wantTypeName: "Request",
			wantAlias:    "v2api",
		},
		{
			name:         "gorm.io 路径",
			source:       "gorm.io/gorm.Model",
			wantPkgPath:  "gorm.io/gorm",
			wantTypeName: "Model",
			wantAlias:    "gorm",
		},
		{
			name:         "深层嵌套路径",
			source:       "github.com/org/repo/internal/pkg/models.Entity",
			wantPkgPath:  "github.com/org/repo/internal/pkg/models",
			wantTypeName: "Entity",
			wantAlias:    "models",
		},
		{
			name:         "本地类型（无包）",
			source:       "User",
			wantPkgPath:  "",
			wantTypeName: "User",
			wantAlias:    "",
		},
		{
			name:    "空字符串",
			source:  "",
			wantErr: true,
		},
		{
			name:    "只有点号",
			source:  ".",
			wantErr: true,
		},
		{
			name:    "类型名为空",
			source:  "github.com/user/repo.",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgPath, typeName, alias, err := parseSourceParam(tt.source, "")

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPkgPath, pkgPath)
			assert.Equal(t, tt.wantTypeName, typeName)
			assert.Equal(t, tt.wantAlias, alias)
		})
	}
}

func TestParseSourceParam_WithImports(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	err := os.WriteFile(testFile, []byte(`package test

import (
	"github.com/user/repo/models"
	customAlias "github.com/user/repo/entities"
	"gorm.io/gorm"
)

type Local struct {}
`), 0644)
	require.NoError(t, err)

	tests := []struct {
		name         string
		source       string
		wantPkgPath  string
		wantTypeName string
		wantAlias    string
		wantErr      bool
	}{
		{
			name:         "使用默认别名",
			source:       "models.User",
			wantPkgPath:  "github.com/user/repo/models",
			wantTypeName: "User",
			wantAlias:    "models",
		},
		{
			name:         "使用自定义别名",
			source:       "customAlias.Entity",
			wantPkgPath:  "github.com/user/repo/entities",
			wantTypeName: "Entity",
			wantAlias:    "customAlias",
		},
		{
			name:         "使用 gorm 包",
			source:       "gorm.Model",
			wantPkgPath:  "gorm.io/gorm",
			wantTypeName: "Model",
			wantAlias:    "gorm",
		},
		{
			name:    "未导入的包",
			source:  "notimported.Type",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgPath, typeName, alias, err := parseSourceParam(tt.source, testFile)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPkgPath, pkgPath)
			assert.Equal(t, tt.wantTypeName, typeName)
			assert.Equal(t, tt.wantAlias, alias)
		})
	}
}

// ============================================================
// extractFileImports 测试
// ============================================================

func TestExtractFileImports(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	err := os.WriteFile(testFile, []byte(`package test

import (
	"fmt"
	"github.com/user/repo/models"
	myAlias "github.com/user/repo/entities"
	. "github.com/user/repo/dot"
	_ "github.com/user/repo/blank"
)
`), 0644)
	require.NoError(t, err)

	imports, err := extractFileImports(testFile)
	require.NoError(t, err)

	// 验证各种导入类型
	assert.Contains(t, imports, "fmt")
	assert.Equal(t, "fmt", imports["fmt"].ImportPath)

	assert.Contains(t, imports, "models")
	assert.Equal(t, "github.com/user/repo/models", imports["models"].ImportPath)

	assert.Contains(t, imports, "myAlias")
	assert.Equal(t, "github.com/user/repo/entities", imports["myAlias"].ImportPath)

	// 点导入使用 "." 作为别名
	assert.Contains(t, imports, ".")
	assert.Equal(t, "github.com/user/repo/dot", imports["."].ImportPath)

	// 空白导入使用 "_" 作为别名
	assert.Contains(t, imports, "_")
	assert.Equal(t, "github.com/user/repo/blank", imports["_"].ImportPath)
}

func TestExtractFileImports_InvalidFile(t *testing.T) {
	_, err := extractFileImports("/nonexistent/file.go")
	require.Error(t, err)
}

func TestExtractFileImports_InvalidSyntax(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "invalid.go")

	err := os.WriteFile(testFile, []byte(`this is not valid go code`), 0644)
	require.NoError(t, err)

	_, err = extractFileImports(testFile)
	require.Error(t, err)
}

// ============================================================
// findProjectRootFromFile 测试
// ============================================================

func TestFindProjectRootFromFile(t *testing.T) {
	// 使用当前项目作为测试
	currentDir, err := os.Getwd()
	require.NoError(t, err)

	// 创建测试文件路径（在当前目录下）
	testFile := filepath.Join(currentDir, "doc.go")

	root, err := findProjectRootFromFile(testFile)
	require.NoError(t, err)

	// 验证找到的根目录包含 go.mod
	goModPath := filepath.Join(root, "go.mod")
	_, err = os.Stat(goModPath)
	require.NoError(t, err)
}

func TestFindProjectRootFromFile_NotFound(t *testing.T) {
	// 在根目录测试，应该找不到 go.mod
	_, err := findProjectRootFromFile("/tmp/nonexistent.go")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "go.mod")
}

// ============================================================
// getModuleNameFromRoot 测试
// ============================================================

func TestGetModuleNameFromRoot(t *testing.T) {
	tempDir := t.TempDir()

	// 创建有效的 go.mod
	goModPath := filepath.Join(tempDir, "go.mod")
	err := os.WriteFile(goModPath, []byte(`module github.com/test/mymodule

go 1.21

require (
	github.com/stretchr/testify v1.8.0
)
`), 0644)
	require.NoError(t, err)

	moduleName, err := getModuleNameFromRoot(tempDir)
	require.NoError(t, err)
	assert.Equal(t, "github.com/test/mymodule", moduleName)
}

func TestGetModuleNameFromRoot_NoModuleDirective(t *testing.T) {
	tempDir := t.TempDir()

	// 创建没有 module 指令的 go.mod
	goModPath := filepath.Join(tempDir, "go.mod")
	err := os.WriteFile(goModPath, []byte(`go 1.21

require (
	github.com/stretchr/testify v1.8.0
)
`), 0644)
	require.NoError(t, err)

	_, err = getModuleNameFromRoot(tempDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "模块名称")
}

func TestGetModuleNameFromRoot_NoGoMod(t *testing.T) {
	tempDir := t.TempDir()
	_, err := getModuleNameFromRoot(tempDir)
	require.Error(t, err)
}

// ============================================================
// TargetComment 生成器测试
// ============================================================

func TestPickGenerator_TargetComment_RequiresSource(t *testing.T) {
	gen := NewPickGenerator()

	// 构造 TargetComment 类型的目标，但没有 source 参数
	ctx := &plugin.GenerateContext{
		Targets: []*plugin.AnnotatedTarget{
			{
				Target: &plugin.Target{
					Kind:        plugin.TargetComment,
					Name:        "Pick",
					PackageName: "testpkg",
					FilePath:    "/test/file.go",
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
					Source: "", // 没有 source
				},
			},
		},
	}

	result, err := gen.Generate(ctx)
	require.NoError(t, err)

	// 应该有错误，因为 TargetComment 必须提供 source
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0].Error(), "source")
}

func TestOmitGenerator_TargetComment_RequiresSource(t *testing.T) {
	gen := NewOmitGenerator()

	ctx := &plugin.GenerateContext{
		Targets: []*plugin.AnnotatedTarget{
			{
				Target: &plugin.Target{
					Kind:        plugin.TargetComment,
					Name:        "Omit",
					PackageName: "testpkg",
					FilePath:    "/test/file.go",
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
					Source: "", // 没有 source
				},
			},
		},
	}

	result, err := gen.Generate(ctx)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0].Error(), "source")
}

func TestPickGenerator_TargetComment_WithSource(t *testing.T) {
	// 创建临时测试目录模拟项目结构
	tempDir := t.TempDir()

	// 创建 go.mod
	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(`module testmodule

go 1.21
`), 0644)
	require.NoError(t, err)

	// 创建源结构体目录
	modelsDir := filepath.Join(tempDir, "models")
	err = os.Mkdir(modelsDir, 0755)
	require.NoError(t, err)

	// 创建源结构体文件
	modelsFile := filepath.Join(modelsDir, "user.go")
	err = os.WriteFile(modelsFile, []byte(`package models

type User struct {
	ID       int64  `+"`json:\"id\"`"+`
	Name     string `+"`json:\"name\"`"+`
	Email    string `+"`json:\"email\"`"+`
	Password string `+"`json:\"-\"`"+`
}
`), 0644)
	require.NoError(t, err)

	// 创建使用 go:gen 的文件
	apiDir := filepath.Join(tempDir, "api")
	err = os.Mkdir(apiDir, 0755)
	require.NoError(t, err)

	apiFile := filepath.Join(apiDir, "types.go")
	err = os.WriteFile(apiFile, []byte(`package api

//go:gen: @Pick(name=UserBasic, source=`+"`testmodule/models.User`"+`, fields=`+"`[ID,Name]`"+`)
`), 0644)
	require.NoError(t, err)

	gen := NewPickGenerator()

	ctx := &plugin.GenerateContext{
		Targets: []*plugin.AnnotatedTarget{
			{
				Target: &plugin.Target{
					Kind:        plugin.TargetComment,
					Name:        "Pick",
					PackageName: "api",
					FilePath:    apiFile,
				},
				Annotations: []*plugin.Annotation{
					{
						Name: "Pick",
						Params: map[string]string{
							"name":   "UserBasic",
							"fields": "[ID,Name]",
							"source": "testmodule/models.User",
						},
					},
				},
				ParsedParams: PickParams{
					Name:   "UserBasic",
					Fields: "[ID,Name]",
					Source: "testmodule/models.User",
				},
			},
		},
	}

	result, err := gen.Generate(ctx)
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Len(t, result.Definitions, 1)

	// 验证生成的代码
	for _, def := range result.Definitions {
		code := string(def.Bytes())
		assert.Contains(t, code, "type UserBasic struct")
		assert.Contains(t, code, "From(src *models.User)")
		assert.Contains(t, code, "NewUserBasic(src *models.User)")
	}
}

// ============================================================
// 生成器支持的目标类型测试
// ============================================================

func TestPickGenerator_SupportedTargetKinds(t *testing.T) {
	gen := NewPickGenerator()
	kinds := gen.SupportedTargets()

	assert.Contains(t, kinds, plugin.TargetStruct)
	assert.Contains(t, kinds, plugin.TargetComment)
}

func TestOmitGenerator_SupportedTargetKinds(t *testing.T) {
	gen := NewOmitGenerator()
	kinds := gen.SupportedTargets()

	assert.Contains(t, kinds, plugin.TargetStruct)
	assert.Contains(t, kinds, plugin.TargetComment)
}

// ============================================================
// 边界情况测试
// ============================================================

func TestParseSourceParam_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantErr   bool
		errMsg    string
		wantPkg   string
		wantType  string
		wantAlias string
	}{
		{
			name:    "域名格式但没有斜杠且无文件",
			source:  "gorm.io.Model",
			wantErr: true,
			errMsg:  "解析导入失败", // 空文件路径会导致解析导入失败
		},
		{
			name:      "多个连续点号被当作完整路径",
			source:    "github.com/user/repo..Type",
			wantErr:   false, // 函数只做解析，不验证路径有效性
			wantPkg:   "github.com/user/repo.",
			wantType:  "Type",
			wantAlias: "repo", // 对 "repo." 调用 sanitizeAlias 结果是 "repo"
		},
		{
			name:      "空格被正确处理",
			source:    "  github.com/user/repo/pkg.User  ",
			wantErr:   false, // 应该被 trim 处理
			wantPkg:   "github.com/user/repo/pkg",
			wantType:  "User",
			wantAlias: "pkg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgPath, typeName, alias, err := parseSourceParam(tt.source, "")
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantPkg, pkgPath)
				assert.Equal(t, tt.wantType, typeName)
				assert.Equal(t, tt.wantAlias, alias)
			}
		})
	}
}

func TestSanitizeAlias_Unicode(t *testing.T) {
	// Go 标识符支持 Unicode 字母，但我们的实现只保留 ASCII
	tests := []struct {
		input    string
		expected string
	}{
		{"包名", ""},          // 中文字符被忽略
		{"pkg名称", "pkg"},    // 混合
		{"αβγ", ""},         // 希腊字母被忽略
		{"pkg_日本語", "pkg_"}, // 下划线保留
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeAlias(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

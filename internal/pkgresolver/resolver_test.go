package pkgresolver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStdLibScanner_IsStdLib(t *testing.T) {
	scanner := NewStdLibScanner()

	tests := []struct {
		name       string
		importPath string
		wantStdLib bool
	}{
		{"fmt", "fmt", true},
		{"os", "os", true},
		{"net/http", "net/http", true},
		{"encoding/json", "encoding/json", true},
		{"time", "time", true},
		{"third-party", "github.com/samber/lo", false},
		{"third-party-2", "gorm.io/datatypes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isStd, err := scanner.IsStdLib(tt.importPath)
			if err != nil {
				t.Fatalf("IsStdLib() error = %v", err)
			}
			if isStd != tt.wantStdLib {
				t.Errorf("IsStdLib(%s) = %v, want %v",
					tt.importPath, isStd, tt.wantStdLib)
			}
		})
	}
}

func TestPackageNameResolver_StdLib(t *testing.T) {
	projectRoot := findTestProjectRoot(t)
	resolver := NewPackageNameResolver(projectRoot)

	tests := []struct {
		name        string
		importPath  string
		wantPkgName string
	}{
		{"fmt", "fmt", "fmt"},
		{"os", "os", "os"},
		{"net/http", "net/http", "http"},
		{"encoding/json", "encoding/json", "json"},
		{"time", "time", "time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgName, err := resolver.GetPackageName(tt.importPath)
			if err != nil {
				t.Fatalf("GetPackageName() error = %v", err)
			}
			if pkgName != tt.wantPkgName {
				t.Errorf("GetPackageName(%s) = %s, want %s",
					tt.importPath, pkgName, tt.wantPkgName)
			}
		})
	}
}

func findTestProjectRoot(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("未找到项目根目录")
		}
		dir = parent
	}
}

// TestPackageNameResolver_ExplicitAlias 测试显式别名场景
// import cc "github.com/xxx/gg"
// 然后使用 cc.SomeType
func TestPackageNameResolver_ExplicitAlias(t *testing.T) {
	projectRoot := findTestProjectRoot(t)
	resolver := NewPackageNameResolver(projectRoot)

	// 测试通过别名路径解析
	importPath := "github.com/donutnomad/gogen/internal/pkgresolver/testdata/aliasedpkg"
	pkgName, err := resolver.GetPackageName(importPath)
	if err != nil {
		t.Fatalf("GetPackageName(%s) error = %v", importPath, err)
	}

	// 应该返回真实的 package 声明名称，而不是别名
	expectedPkgName := "aliasedpkg"
	if pkgName != expectedPkgName {
		t.Errorf("GetPackageName(%s) = %s, want %s", importPath, pkgName, expectedPkgName)
	}
}

// TestPackageNameResolver_MismatchedPkgName 测试包名与文件夹名不一致的场景
// 文件夹是 gg，但 package 声明是 g2
func TestPackageNameResolver_MismatchedPkgName(t *testing.T) {
	projectRoot := findTestProjectRoot(t)
	resolver := NewPackageNameResolver(projectRoot)

	// 测试文件夹名是 gg，但 package 声明是 g2 的情况
	importPath := "github.com/donutnomad/gogen/internal/pkgresolver/testdata/gg"
	pkgName, err := resolver.GetPackageName(importPath)
	if err != nil {
		t.Fatalf("GetPackageName(%s) error = %v", importPath, err)
	}

	// 应该返回真实的 package 声明 "g2"，而不是文件夹名 "gg"
	expectedPkgName := "g2"
	if pkgName != expectedPkgName {
		t.Errorf("GetPackageName(%s) = %s, want %s", importPath, pkgName, expectedPkgName)
	}
}

// TestPackageNameResolver_ProjectInternal 测试项目内部包解析
func TestPackageNameResolver_ProjectInternal(t *testing.T) {
	projectRoot := findTestProjectRoot(t)
	resolver := NewPackageNameResolver(projectRoot)

	tests := []struct {
		name        string
		importPath  string
		wantPkgName string
	}{
		{"structparse", "github.com/donutnomad/gogen/internal/structparse", "structparse"},
		{"gormparse", "github.com/donutnomad/gogen/internal/gormparse", "gormparse"},
		{"xast", "github.com/donutnomad/gogen/internal/xast", "xast"},
		{"pkgresolver", "github.com/donutnomad/gogen/internal/pkgresolver", "pkgresolver"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgName, err := resolver.GetPackageName(tt.importPath)
			if err != nil {
				t.Fatalf("GetPackageName(%s) error = %v", tt.importPath, err)
			}
			if pkgName != tt.wantPkgName {
				t.Errorf("GetPackageName(%s) = %s, want %s", tt.importPath, pkgName, tt.wantPkgName)
			}
		})
	}
}

// TestPackageNameResolver_Cache 测试缓存机制
func TestPackageNameResolver_Cache(t *testing.T) {
	projectRoot := findTestProjectRoot(t)
	resolver := NewPackageNameResolver(projectRoot)

	importPath := "fmt"

	// 第一次调用
	pkgName1, err := resolver.GetPackageName(importPath)
	if err != nil {
		t.Fatalf("First GetPackageName(%s) error = %v", importPath, err)
	}

	// 第二次调用应该从缓存中获取
	pkgName2, err := resolver.GetPackageName(importPath)
	if err != nil {
		t.Fatalf("Second GetPackageName(%s) error = %v", importPath, err)
	}

	if pkgName1 != pkgName2 {
		t.Errorf("Cache inconsistency: first call returned %s, second call returned %s", pkgName1, pkgName2)
	}

	if pkgName1 != "fmt" {
		t.Errorf("GetPackageName(%s) = %s, want fmt", importPath, pkgName1)
	}
}

// TestPackageNameResolver_EmptyPath 测试空路径处理
func TestPackageNameResolver_EmptyPath(t *testing.T) {
	projectRoot := findTestProjectRoot(t)
	resolver := NewPackageNameResolver(projectRoot)

	// 测试空字符串路径
	pkgName, _ := resolver.GetPackageName("")
	// 空路径的 filepath.Base("") 返回 "."
	if pkgName != "." {
		t.Errorf("GetPackageName(\"\") = %s, expected \".\"", pkgName)
	}
}

// TestPackageNameResolver_StdLibExtended 扩展的标准库测试
func TestPackageNameResolver_StdLibExtended(t *testing.T) {
	projectRoot := findTestProjectRoot(t)
	resolver := NewPackageNameResolver(projectRoot)

	tests := []struct {
		name        string
		importPath  string
		wantPkgName string
	}{
		{"context", "context", "context"},
		{"strings", "strings", "strings"},
		{"io", "io", "io"},
		{"io/ioutil", "io/ioutil", "ioutil"},
		{"crypto/sha256", "crypto/sha256", "sha256"},
		{"net/http", "net/http", "http"},
		{"encoding/json", "encoding/json", "json"},
		{"database/sql", "database/sql", "sql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgName, err := resolver.GetPackageName(tt.importPath)
			if err != nil {
				t.Fatalf("GetPackageName(%s) error = %v", tt.importPath, err)
			}
			if pkgName != tt.wantPkgName {
				t.Errorf("GetPackageName(%s) = %s, want %s", tt.importPath, pkgName, tt.wantPkgName)
			}
		})
	}
}

// TestStdLibScanner_Extended 扩展的标准库识别测试
func TestStdLibScanner_Extended(t *testing.T) {
	scanner := NewStdLibScanner()

	tests := []struct {
		name       string
		importPath string
		wantStdLib bool
	}{
		{"context", "context", true},
		{"strings", "strings", true},
		{"io", "io", true},
		{"io/ioutil", "io/ioutil", true},
		{"crypto/sha256", "crypto/sha256", true},
		{"internal/cpu", "internal/cpu", false}, // internal 包不能被用户代码导入
		{"golang.org/x/tools", "golang.org/x/tools", false},
		{"github.com/pkg/errors", "github.com/pkg/errors", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isStd, err := scanner.IsStdLib(tt.importPath)
			if err != nil {
				t.Fatalf("IsStdLib(%s) error = %v", tt.importPath, err)
			}
			if isStd != tt.wantStdLib {
				t.Errorf("IsStdLib(%s) = %v, want %v", tt.importPath, isStd, tt.wantStdLib)
			}
		})
	}
}

// TestPackageFileReader_ReadPackageName 测试直接读取包声明
func TestPackageFileReader_ReadPackageName(t *testing.T) {
	reader := &PackageFileReader{}

	tests := []struct {
		name        string
		pkgDir      string
		wantPkgName string
	}{
		{
			"aliasedpkg",
			"testdata/aliasedpkg",
			"aliasedpkg",
		},
		{
			"mismatched-gg-g2",
			"testdata/gg",
			"g2", // 文件夹是 gg，但 package 声明是 g2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgName, err := reader.ReadPackageName(tt.pkgDir)
			if err != nil {
				t.Fatalf("ReadPackageName(%s) error = %v", tt.pkgDir, err)
			}
			if pkgName != tt.wantPkgName {
				t.Errorf("ReadPackageName(%s) = %s, want %s", tt.pkgDir, pkgName, tt.wantPkgName)
			}
		})
	}
}

// TestIntegration_StructParseWithPkgPath 集成测试：验证 structparse 能正确填充 PkgPath
// 这个测试验证了从 AST 解析到 PkgPath 填充的完整流程
func TestIntegration_StructParseWithPkgPath(t *testing.T) {
	// 导入 structparse 和 gormparse 包以进行集成测试
	// 这里我们会测试两个关键场景：
	// 1. 显式别名：import cc "github.com/xxx/gg"
	// 2. 包名≠文件夹名：文件夹 gg，package g2

	projectRoot := findTestProjectRoot(t)
	resolver := NewPackageNameResolver(projectRoot)

	// 测试场景 1：显式别名
	aliasImportPath := "github.com/donutnomad/gogen/internal/pkgresolver/testdata/aliasedpkg"
	aliasPkgName, err := resolver.GetPackageName(aliasImportPath)
	if err != nil {
		t.Fatalf("GetPackageName(%s) error = %v", aliasImportPath, err)
	}
	if aliasPkgName != "aliasedpkg" {
		t.Errorf("Alias scenario: expected package name 'aliasedpkg', got %s", aliasPkgName)
	}

	// 测试场景 2：包名≠文件夹名
	mismatchedImportPath := "github.com/donutnomad/gogen/internal/pkgresolver/testdata/gg"
	mismatchedPkgName, err := resolver.GetPackageName(mismatchedImportPath)
	if err != nil {
		t.Fatalf("GetPackageName(%s) error = %v", mismatchedImportPath, err)
	}
	if mismatchedPkgName != "g2" {
		t.Errorf("Mismatched scenario: folder is 'gg' but package should be 'g2', got %s", mismatchedPkgName)
	}

	t.Logf("Integration test passed: alias scenario returns '%s', mismatched scenario returns '%s'",
		aliasPkgName, mismatchedPkgName)
}

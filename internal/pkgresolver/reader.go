package pkgresolver

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// PackageFileReader 从磁盘路径读取真实包名
type PackageFileReader struct{}

// ReadPackageName 读取指定目录的 package 声明
// 返回：包名, 错误
func (r *PackageFileReader) ReadPackageName(pkgDir string) (string, error) {
	// 查找目录中的所有 .go 文件（排除测试文件）
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return "", fmt.Errorf("读取目录失败 %s: %w", pkgDir, err)
	}

	var goFiles []string
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() &&
			strings.HasSuffix(name, ".go") &&
			!strings.HasSuffix(name, "_test.go") {
			goFiles = append(goFiles, name)
		}
	}

	if len(goFiles) == 0 {
		return "", fmt.Errorf("目录 %s 中没有找到 Go 源文件", pkgDir)
	}

	// 读取第一个文件的 package 声明
	firstFile := filepath.Join(pkgDir, goFiles[0])
	pkgName, err := r.parsePackageNameFromFile(firstFile)
	if err != nil {
		return "", err
	}

	return pkgName, nil
}

// parsePackageNameFromFile 从单个文件解析包名
func (r *PackageFileReader) parsePackageNameFromFile(filename string) (string, error) {
	fset := token.NewFileSet()

	// 只解析包声明，不需要完整解析
	f, err := parser.ParseFile(fset, filename, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("解析文件 %s 失败: %w", filename, err)
	}

	if f.Name == nil {
		return "", fmt.Errorf("文件 %s 中没有 package 声明", filename)
	}

	return f.Name.Name, nil
}

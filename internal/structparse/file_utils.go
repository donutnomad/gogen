package structparse

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindGoFiles 查找目录中的所有Go文件（不包含测试文件）
func FindGoFiles(dir string) ([]string, error) {
	return findGoFiles(dir)
}

// findGoFiles 查找目录中的所有Go文件（内部使用）
func findGoFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ContainsStruct 检查文件是否包含指定的结构体
func ContainsStruct(filename, structName string) bool {
	return containsStruct(filename, structName)
}

// containsStruct 检查文件是否包含指定的结构体（内部使用）
func containsStruct(filename, structName string) bool {
	content, err := os.ReadFile(filename)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), fmt.Sprintf("type %s struct", structName))
}

package pkgresolver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindGoWorkFile 从 go.mod 所在目录继续向上查找 go.work 文件，返回 go.work 所在目录
func FindGoWorkFile(projectRoot string) string {
	dir := filepath.Dir(projectRoot)
	for {
		goWorkPath := filepath.Join(dir, "go.work")
		if _, err := os.Stat(goWorkPath); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// FindPackageInWorkspace 解析 go.work 中的 use 指令，查找 importPath 所属的 workspace 模块
func FindPackageInWorkspace(goWorkDir, importPath string) (string, error) {
	goWorkPath := filepath.Join(goWorkDir, "go.work")
	content, err := os.ReadFile(goWorkPath)
	if err != nil {
		return "", fmt.Errorf("无法读取 go.work: %v", err)
	}

	useDirs := ParseGoWorkUseDirs(string(content))

	for _, useDir := range useDirs {
		absDir := useDir
		if !filepath.IsAbs(useDir) {
			absDir = filepath.Join(goWorkDir, useDir)
		}

		moduleName, err := getModuleName(absDir)
		if err != nil {
			continue
		}

		if importPath == moduleName || strings.HasPrefix(importPath, moduleName+"/") {
			relativePath := strings.TrimPrefix(importPath, moduleName)
			relativePath = strings.TrimPrefix(relativePath, "/")
			packagePath := filepath.Join(absDir, relativePath)

			if _, err := os.Stat(packagePath); err == nil {
				return packagePath, nil
			}
		}
	}

	return "", fmt.Errorf("未在 go.work workspace 中找到包 %s", importPath)
}

// ParseGoWorkUseDirs 解析 go.work 文件内容，提取所有 use 指令中的目录路径
func ParseGoWorkUseDirs(content string) []string {
	var dirs []string
	lines := strings.Split(content, "\n")
	inUseBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, "use") && strings.Contains(line, "(") {
			inUseBlock = true
			continue
		}

		if inUseBlock {
			if strings.Contains(line, ")") {
				inUseBlock = false
				continue
			}
			dir := strings.TrimSpace(line)
			if dir != "" {
				dirs = append(dirs, dir)
			}
			continue
		}

		if strings.HasPrefix(line, "use ") {
			dir := strings.TrimSpace(strings.TrimPrefix(line, "use"))
			if dir != "" && dir != "(" {
				dirs = append(dirs, dir)
			}
		}
	}

	return dirs
}

package structparse

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// findStructInPackageWithImportsAndBaseDir 在指定包中查找结构体定义，使用导入信息和基础目录
func (c *ParseContext) findStructInPackageWithImportsAndBaseDir(packageName, structName string, imports map[string]*ImportInfo, baseDir string) (string, error) {
	// 从imports中获取完整的导入路径
	importInfo, exists := imports[packageName]
	if !exists {
		return "", fmt.Errorf("未找到包 %s 的导入信息", packageName)
	}
	fullImportPath := importInfo.ImportPath

	// 从基础目录开始查找项目根目录
	projectRoot, err := findProjectRootFromDir(baseDir)
	if err != nil {
		return "", err
	}

	// 根据完整导入路径查找包路径
	packagePath, err := findPackagePathByImport(projectRoot, fullImportPath)
	if err != nil {
		return "", err
	}

	// 在包路径中查找包含结构体的文件
	files, err := findGoFiles(packagePath)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if containsStruct(file, structName) {
			return file, nil
		}
	}

	return "", fmt.Errorf("未在包 %s 中找到结构体 %s", packageName, structName)
}

// findPackagePathByImport 根据完整导入路径查找包路径
func findPackagePathByImport(projectRoot, importPath string) (string, error) {
	// 读取go.mod获取module名称
	moduleName, err := getModuleName(projectRoot)
	if err != nil {
		return "", err
	}

	// 如果导入路径以当前模块名开头，则是项目内部包
	if strings.HasPrefix(importPath, moduleName) {
		relativePath := strings.TrimPrefix(importPath, moduleName)
		relativePath = strings.TrimPrefix(relativePath, "/")
		packagePath := filepath.Join(projectRoot, relativePath)

		if _, err := os.Stat(packagePath); err == nil {
			return packagePath, nil
		}
	}

	// 处理标准库导入：标准库包不包含域名（不含"."）
	// 例如：fmt, os, net/http, encoding/json, crypto/sha256 等
	if !strings.Contains(importPath, ".") {
		return "", fmt.Errorf("标准库包 %s 不支持结构体解析", importPath)
	}

	// 处理第三方包：尝试从Go模块缓存中查找
	return FindThirdPartyPackage(importPath)
}

// FindThirdPartyPackage 查找第三方包的路径（导出供其他包使用）
func FindThirdPartyPackage(importPath string) (string, error) {
	// 获取GOPATH和GOMODCACHE
	goPath := os.Getenv("GOPATH")
	goModCache := os.Getenv("GOMODCACHE")

	// 如果GOMODCACHE为空，使用默认路径
	if goModCache == "" {
		if goPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("无法获取用户主目录: %v", err)
			}
			goPath = filepath.Join(homeDir, "go")
		}
		goModCache = filepath.Join(goPath, "pkg", "mod")
	}

	// 尝试查找模块根路径
	// 对于 github.com/user/repo/pkg/sub，需要尝试：
	// 1. github.com/user/repo/pkg/sub@*
	// 2. github.com/user/repo/pkg@*
	// 3. github.com/user/repo@*
	parts := strings.Split(importPath, "/")

	for i := len(parts); i >= 1; i-- {
		// 构建可能的模块根路径
		modulePath := strings.Join(parts[:i], "/")
		subPath := ""
		if i < len(parts) {
			subPath = strings.Join(parts[i:], "/")
		}

		// Go模块缓存中大写字母的编码规则：
		// 大写字母会被编码为 "!小写字母"
		encodedModulePath := encodeModulePath(modulePath)

		// 尝试在GOMODCACHE中查找包
		packageCachePattern := filepath.Join(goModCache, encodedModulePath+"@*")
		matches, err := filepath.Glob(packageCachePattern)
		if err != nil {
			continue
		}

		// 如果找到匹配的模块
		if len(matches) > 0 {
			// 选择最新的版本（按字典序排序，最后一个通常版本号较高）
			latestMatch := matches[len(matches)-1]

			// 如果有子路径，拼接上
			finalPath := latestMatch
			if subPath != "" {
				finalPath = filepath.Join(latestMatch, subPath)
			}

			// 验证路径是否存在
			if _, err := os.Stat(finalPath); err == nil {
				return finalPath, nil
			}
		}
	}

	// 如果在模块缓存中没找到，尝试在GOPATH/src中查找（旧版本Go的方式）
	if goPath != "" {
		goPathSrc := filepath.Join(goPath, "src", importPath)
		if _, err := os.Stat(goPathSrc); err == nil {
			return goPathSrc, nil
		}
	}

	return "", fmt.Errorf("未找到第三方包 %s，请确保该包已正确安装", importPath)
}

// encodeModulePath 将模块路径编码为 Go 模块缓存使用的格式
// Go 模块缓存规则：大写字母前添加 ! 并转为小写
// 例如：github.com/Xuanwo/gg -> github.com/!xuanwo/gg
func encodeModulePath(path string) string {
	var result strings.Builder
	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			result.WriteRune('!')
			result.WriteRune(r + 32) // 转为小写 (A-Z -> a-z)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// getModuleName 从go.mod文件获取模块名称
func getModuleName(projectRoot string) (string, error) {
	goModPath := filepath.Join(projectRoot, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	return "", fmt.Errorf("未在 go.mod 中找到模块名称")
}

// findProjectRoot 查找项目根目录（包含go.mod的目录）- 向后兼容
func findProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findProjectRootFromDir(currentDir)
}

// findProjectRootFromDir 从指定目录开始查找项目根目录（包含go.mod的目录）
func findProjectRootFromDir(startDir string) (string, error) {
	// 获取绝对路径
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// 已经到了根目录
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("未找到项目根目录（go.mod文件）从 %s 开始", startDir)
}

// findPackagePath 根据包名查找包路径
func findPackagePath(projectRoot, packageName string) (string, error) {
	// 如果是相对导入（如 "taas-backend/pkg/orm"）
	// 则在项目根目录下查找对应路径

	// 读取当前文件的import信息来解析完整的import路径
	currentFile := ""
	files, err := findGoFiles(".")
	if err == nil && len(files) > 0 {
		currentFile = files[0]
	}

	if currentFile != "" {
		fullImportPath, err := findImportPath(currentFile, packageName)
		if err == nil {
			// 解析相对于项目根目录的路径
			if strings.Contains(fullImportPath, "/") {
				parts := strings.Split(fullImportPath, "/")
				if len(parts) > 1 {
					// 从项目根目录构建路径
					relativePath := strings.Join(parts[1:], "/")
					packagePath := filepath.Join(projectRoot, relativePath)
					if _, err := os.Stat(packagePath); err == nil {
						return packagePath, nil
					}
				}
			}
		}
	}

	// 如果无法从import解析，尝试常见的包结构
	commonPaths := []string{
		filepath.Join(projectRoot, "pkg", packageName),
		filepath.Join(projectRoot, "internal", packageName),
		filepath.Join(projectRoot, packageName),
		filepath.Join(projectRoot, "src", packageName),
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("未找到包 %s 的路径", packageName)
}

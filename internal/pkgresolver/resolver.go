package pkgresolver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PackageNameResolver 包名解析器（统一入口）
type PackageNameResolver struct {
	cache       *PackageNameCache
	stdLib      *StdLibScanner
	reader      *PackageFileReader
	projectRoot string // 项目根目录（包含 go.mod）
}

// NewPackageNameResolver 创建解析器
func NewPackageNameResolver(projectRoot string) *PackageNameResolver {
	return &PackageNameResolver{
		cache:       NewPackageNameCache(),
		stdLib:      NewStdLibScanner(),
		reader:      &PackageFileReader{},
		projectRoot: projectRoot,
	}
}

// GetPackageName 获取导入路径对应的真实包名
//
// 示例：
//
//	"fmt" → "fmt"
//	"net/http" → "http"
//	"github.com/samber/lo" → "lo"
//	"github.com/Xuanwo/gg" → "g2" (如果 package 声明是 g2)
func (r *PackageNameResolver) GetPackageName(importPath string) (string, error) {
	// 检查缓存
	if name, ok := r.cache.GetByImportPath(importPath); ok {
		return name, nil
	}

	// 判断包类型并获取磁盘路径
	diskPath, err := r.resolveDiskPath(importPath)
	if err != nil {
		// 降级：返回路径最后一部分
		return filepath.Base(importPath), nil
	}

	// 从磁盘读取包名
	pkgName, err := r.reader.ReadPackageName(diskPath)
	if err != nil {
		// 降级：返回路径最后一部分
		return filepath.Base(importPath), nil
	}

	// 缓存结果
	r.cache.SetByImportPath(importPath, pkgName)
	r.cache.SetByDiskPath(diskPath, pkgName)

	return pkgName, nil
}

// resolveDiskPath 将导入路径解析为磁盘路径
func (r *PackageNameResolver) resolveDiskPath(importPath string) (string, error) {
	// 判断是否是标准库
	isStd, err := r.stdLib.IsStdLib(importPath)
	if err == nil && isStd {
		return r.stdLib.GetStdLibPath(importPath)
	}

	// 判断是否是项目内部包
	if r.projectRoot != "" {
		moduleName, err := getModuleName(r.projectRoot)
		if err == nil && strings.HasPrefix(importPath, moduleName) {
			// 项目内部包
			relativePath := strings.TrimPrefix(importPath, moduleName)
			relativePath = strings.TrimPrefix(relativePath, "/")
			return filepath.Join(r.projectRoot, relativePath), nil
		}
	}

	// 第三方包：查找 GOMODCACHE
	return findThirdPartyPackage(importPath)
}

// IsStdLib 判断是否是标准库（便捷方法）
func (r *PackageNameResolver) IsStdLib(importPath string) (bool, error) {
	return r.stdLib.IsStdLib(importPath)
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

// findThirdPartyPackage 查找第三方包的路径
func findThirdPartyPackage(importPath string) (string, error) {
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
	parts := strings.Split(importPath, "/")

	for i := len(parts); i >= 1; i-- {
		// 构建可能的模块根路径
		modulePath := strings.Join(parts[:i], "/")
		subPath := ""
		if i < len(parts) {
			subPath = strings.Join(parts[i:], "/")
		}

		// Go模块缓存中大写字母的编码规则
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

	// 如果在模块缓存中没找到，尝试在GOPATH/src中查找
	if goPath != "" {
		goPathSrc := filepath.Join(goPath, "src", importPath)
		if _, err := os.Stat(goPathSrc); err == nil {
			return goPathSrc, nil
		}
	}

	return "", fmt.Errorf("未找到第三方包 %s", importPath)
}

// encodeModulePath 将模块路径编码为 Go 模块缓存使用的格式
// Go 模块缓存规则：大写字母前添加 ! 并转为小写
// 例如：github.com/Xuanwo/gg -> github.com/!xuanwo/gg
func encodeModulePath(path string) string {
	var result strings.Builder
	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			result.WriteRune('!')
			result.WriteRune(r + 32) // 转为小写
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

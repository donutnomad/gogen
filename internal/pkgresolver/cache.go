package pkgresolver

import "sync"

// PackageNameCache 包名缓存
type PackageNameCache struct {
	mu sync.RWMutex
	// 导入路径 → 包名
	importCache map[string]string
	// 磁盘路径 → 包名
	diskCache map[string]string
}

// NewPackageNameCache 创建缓存
func NewPackageNameCache() *PackageNameCache {
	return &PackageNameCache{
		importCache: make(map[string]string),
		diskCache:   make(map[string]string),
	}
}

// GetByImportPath 根据导入路径获取缓存的包名
func (c *PackageNameCache) GetByImportPath(importPath string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	name, ok := c.importCache[importPath]
	return name, ok
}

// GetByDiskPath 根据磁盘路径获取缓存的包名
func (c *PackageNameCache) GetByDiskPath(diskPath string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	name, ok := c.diskCache[diskPath]
	return name, ok
}

// SetByImportPath 缓存导入路径 → 包名
func (c *PackageNameCache) SetByImportPath(importPath, pkgName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.importCache[importPath] = pkgName
}

// SetByDiskPath 缓存磁盘路径 → 包名
func (c *PackageNameCache) SetByDiskPath(diskPath, pkgName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.diskCache[diskPath] = pkgName
}

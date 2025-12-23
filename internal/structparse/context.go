package structparse

import (
	"sync"

	"github.com/donutnomad/gogen/internal/pkgresolver"
)

// PackageResolver 包名解析器接口
type PackageResolver interface {
	GetPackageName(importPath string) (string, error)
}

// ParseContext 解析上下文，替代全局单例
type ParseContext struct {
	resolver     PackageResolver
	projectRoot  string
	resolverOnce sync.Once
}

// NewParseContext 创建解析上下文（使用默认工作目录）
func NewParseContext() *ParseContext {
	root, _ := findProjectRoot()
	return &ParseContext{projectRoot: root}
}

// NewParseContextWithRoot 创建解析上下文（指定项目根目录）
func NewParseContextWithRoot(projectRoot string) *ParseContext {
	return &ParseContext{projectRoot: projectRoot}
}

// NewParseContextWithResolver 创建解析上下文（指定PackageResolver，用于测试）
func NewParseContextWithResolver(resolver PackageResolver) *ParseContext {
	return &ParseContext{resolver: resolver}
}

// GetResolver 获取包解析器（延迟初始化）
func (c *ParseContext) GetResolver() PackageResolver {
	if c.resolver != nil {
		return c.resolver
	}

	if c.projectRoot == "" {
		return nil
	}

	c.resolverOnce.Do(func() {
		c.resolver = &defaultPackageResolver{
			resolver: pkgresolver.NewPackageNameResolver(c.projectRoot),
		}
	})

	return c.resolver
}

// defaultPackageResolver 默认包解析器实现
type defaultPackageResolver struct {
	resolver *pkgresolver.PackageNameResolver
}

func (r *defaultPackageResolver) GetPackageName(importPath string) (string, error) {
	if r.resolver == nil {
		return "", nil
	}
	return r.resolver.GetPackageName(importPath)
}

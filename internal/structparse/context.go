package structparse

import (
	"go/ast"
	"go/parser"
	"go/token"
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

	// 文件级缓存
	fileCacheMu sync.Mutex
	fileCache   map[string]*cachedFile   // key: filePath -> parsed AST
	importCache map[string]cachedImports // key: filePath -> imports
	methodCache map[string]cachedMethods // key: "dir:structName" -> methods
	dirFilesMu  sync.Mutex
	dirFiles    map[string][]string // key: dir -> go files list
}

// cachedFile 缓存的文件 AST
type cachedFile struct {
	fset *token.FileSet
	file *ast.File
}

// cachedImports 缓存的 import 信息
type cachedImports struct {
	imports map[string]*ImportInfo
	err     error
}

// cachedMethods 缓存的方法信息
type cachedMethods struct {
	methods []MethodInfo
	err     error
}

// NewParseContext 创建解析上下文（使用默认工作目录）
func NewParseContext() *ParseContext {
	root, _ := findProjectRoot()
	return &ParseContext{
		projectRoot: root,
		fileCache:   make(map[string]*cachedFile),
		importCache: make(map[string]cachedImports),
		methodCache: make(map[string]cachedMethods),
		dirFiles:    make(map[string][]string),
	}
}

// NewParseContextWithRoot 创建解析上下文（指定项目根目录）
func NewParseContextWithRoot(projectRoot string) *ParseContext {
	return &ParseContext{
		projectRoot: projectRoot,
		fileCache:   make(map[string]*cachedFile),
		importCache: make(map[string]cachedImports),
		methodCache: make(map[string]cachedMethods),
		dirFiles:    make(map[string][]string),
	}
}

// NewParseContextWithResolver 创建解析上下文（指定PackageResolver，用于测试）
func NewParseContextWithResolver(resolver PackageResolver) *ParseContext {
	return &ParseContext{
		resolver:    resolver,
		fileCache:   make(map[string]*cachedFile),
		importCache: make(map[string]cachedImports),
		methodCache: make(map[string]cachedMethods),
		dirFiles:    make(map[string][]string),
	}
}

// getOrParseFile 获取或解析文件 AST（带缓存）
func (c *ParseContext) getOrParseFile(filename string) (*ast.File, *token.FileSet, error) {
	c.fileCacheMu.Lock()
	defer c.fileCacheMu.Unlock()

	if cached, ok := c.fileCache[filename]; ok {
		return cached.file, cached.fset, nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	c.fileCache[filename] = &cachedFile{fset: fset, file: file}
	return file, fset, nil
}

// getDirGoFiles 获取目录中的 Go 文件列表（带缓存）
func (c *ParseContext) getDirGoFiles(dir string) ([]string, error) {
	c.dirFilesMu.Lock()
	defer c.dirFilesMu.Unlock()

	if files, ok := c.dirFiles[dir]; ok {
		return files, nil
	}

	files, err := findGoFiles(dir)
	if err != nil {
		return nil, err
	}

	c.dirFiles[dir] = files
	return files, nil
}

// GetFileAST 获取缓存中的文件 AST（如果已解析过）
func (c *ParseContext) GetFileAST(filename string) *ast.File {
	c.fileCacheMu.Lock()
	defer c.fileCacheMu.Unlock()

	if cached, ok := c.fileCache[filename]; ok {
		return cached.file
	}
	return nil
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

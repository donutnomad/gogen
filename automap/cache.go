package automap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sync"

	"github.com/donutnomad/gogen/internal/structparse"
)

// FileASTCache 文件 AST 缓存，避免同一文件被 parser.ParseFile 反复解析
type FileASTCache struct {
	mu    sync.Mutex
	fset  *token.FileSet
	files map[string]*ast.File // key: filePath
}

// NewFileASTCache 创建新的文件 AST 缓存
func NewFileASTCache() *FileASTCache {
	return &FileASTCache{
		fset:  token.NewFileSet(),
		files: make(map[string]*ast.File),
	}
}

// GetOrParse 获取或解析文件 AST
func (c *FileASTCache) GetOrParse(filePath string) (*ast.File, *token.FileSet, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if file, ok := c.files[filePath]; ok {
		return file, c.fset, nil
	}

	file, err := parser.ParseFile(c.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	c.files[filePath] = file
	return file, c.fset, nil
}

// FileSet 返回共享的 FileSet
func (c *FileASTCache) FileSet() *token.FileSet {
	return c.fset
}

// ParseContext2 扩展的解析上下文，包含 AST 缓存和 structparse 缓存
type ParseContext2 struct {
	ASTCache *FileASTCache

	mu              sync.Mutex
	structParseCtx  *structparse.ParseContext
	structCache     map[string]*structparse.StructInfo // key: "filePath:structName"
	structCacheErrs map[string]error                   // key: "filePath:structName"
}

// NewParseContext2 创建新的解析上下文
func NewParseContext2() *ParseContext2 {
	return &ParseContext2{
		ASTCache:        NewFileASTCache(),
		structParseCtx:  structparse.NewParseContext(),
		structCache:     make(map[string]*structparse.StructInfo),
		structCacheErrs: make(map[string]error),
	}
}

// ParseStruct 带缓存的结构体解析
func (c *ParseContext2) ParseStruct(filePath, structName string) (*structparse.StructInfo, error) {
	key := filePath + ":" + structName
	c.mu.Lock()
	if info, ok := c.structCache[key]; ok {
		err := c.structCacheErrs[key]
		c.mu.Unlock()
		return info, err
	}
	c.mu.Unlock()

	info, err := c.structParseCtx.ParseStruct(filePath, structName)

	c.mu.Lock()
	c.structCache[key] = info
	c.structCacheErrs[key] = err
	c.mu.Unlock()

	return info, err
}

package templategen

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// ImportResolver 解析类型引用的 import 路径
type ImportResolver struct {
	// 当前文件的 import 映射: alias/pkgName -> full path
	fileImports map[string]string
	// @Import 注解定义的别名
	annotationAliases map[string]string
}

// NewImportResolver 创建新的 ImportResolver
func NewImportResolver(filePath string) (*ImportResolver, error) {
	resolver := &ImportResolver{
		fileImports:       make(map[string]string),
		annotationAliases: make(map[string]string),
	}

	// 解析文件的 imports
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")

		var alias string
		if imp.Name != nil {
			// 有显式别名
			alias = imp.Name.Name
		} else {
			// 使用路径最后一部分作为包名
			parts := strings.Split(importPath, "/")
			alias = parts[len(parts)-1]
		}

		resolver.fileImports[alias] = importPath
	}

	return resolver, nil
}

// AddAlias 添加 @Import 注解定义的别名
func (r *ImportResolver) AddAlias(alias, path string) {
	r.annotationAliases[alias] = path
}

// ResolveTypeRef 解析类型引用
// 由于注解解析器已经去除了引号，这里采用启发式方法判断：
// - 如果值包含 . 且包前缀可解析为包路径，则为类型引用
// - 如果值是 Go 内置类型，则为类型引用
// - 其他情况视为字符串值
func (r *ImportResolver) ResolveTypeRef(value string) TypeRef {
	ref := TypeRef{Raw: value}

	// 检查是否有包前缀 (如 io.Reader, myutil.Helper)
	dotIdx := strings.Index(value, ".")
	if dotIdx > 0 {
		pkgPrefix := value[:dotIdx]
		typeName := value[dotIdx+1:]

		// 检查是否是全限定类型路径 (如 github.com/pkg/errors.StackTracer)
		if strings.Contains(value, "/") {
			lastDot := strings.LastIndex(value, ".")
			if lastDot > 0 {
				ref.IsString = false
				ref.PkgPath = value[:lastDot]
				ref.TypeName = value[lastDot+1:]
				ref.PkgAlias = filepath.Base(ref.PkgPath)
				ref.FullType = ref.PkgAlias + "." + ref.TypeName
				return ref
			}
		}

		// 优先级 1: 当前文件的 import
		if path, ok := r.fileImports[pkgPrefix]; ok {
			ref.IsString = false
			ref.PkgPath = path
			ref.PkgAlias = pkgPrefix
			ref.TypeName = typeName
			ref.FullType = pkgPrefix + "." + typeName
			return ref
		}

		// 优先级 2: @Import 注解定义
		if path, ok := r.annotationAliases[pkgPrefix]; ok {
			ref.IsString = false
			ref.PkgPath = path
			ref.PkgAlias = pkgPrefix
			ref.TypeName = typeName
			ref.FullType = pkgPrefix + "." + typeName
			return ref
		}

		// 优先级 3: 标准库白名单
		if path, ok := stdLibPackages[pkgPrefix]; ok {
			ref.IsString = false
			ref.PkgPath = path
			ref.PkgAlias = pkgPrefix
			ref.TypeName = typeName
			ref.FullType = pkgPrefix + "." + typeName
			return ref
		}

		// 包前缀无法解析，视为字符串值（可能是包含点的字符串如 "v1.0.0"）
		ref.IsString = true
		ref.StringVal = value
		ref.FullType = value
		return ref
	}

	// 没有包前缀，检查是否是 Go 内置类型
	if isBuiltinType(value) {
		ref.IsString = false
		ref.TypeName = value
		ref.FullType = value
		return ref
	}

	// 不是内置类型，视为字符串值
	ref.IsString = true
	ref.StringVal = value
	ref.FullType = value
	return ref
}

// isBuiltinType 检查是否是 Go 内置类型
func isBuiltinType(name string) bool {
	builtins := map[string]bool{
		// 基本类型
		"bool":       true,
		"string":     true,
		"int":        true,
		"int8":       true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"uint":       true,
		"uint8":      true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"uintptr":    true,
		"byte":       true,
		"rune":       true,
		"float32":    true,
		"float64":    true,
		"complex64":  true,
		"complex128": true,
		// 特殊类型
		"any":        true,
		"error":      true,
		"comparable": true,
	}
	return builtins[name]
}

// stdLibPackages 标准库包名到路径的映射
var stdLibPackages = map[string]string{
	// 常用包
	"fmt":      "fmt",
	"io":       "io",
	"os":       "os",
	"time":     "time",
	"context":  "context",
	"errors":   "errors",
	"strings":  "strings",
	"bytes":    "bytes",
	"strconv":  "strconv",
	"sync":     "sync",
	"math":     "math",
	"sort":     "sort",
	"regexp":   "regexp",
	"reflect":  "reflect",
	"runtime":  "runtime",
	"testing":  "testing",
	"log":      "log",
	"flag":     "flag",
	"path":     "path",
	"filepath": "path/filepath",
	"bufio":    "bufio",
	"unicode":  "unicode",

	// encoding
	"json":     "encoding/json",
	"xml":      "encoding/xml",
	"base64":   "encoding/base64",
	"hex":      "encoding/hex",
	"binary":   "encoding/binary",
	"gob":      "encoding/gob",
	"csv":      "encoding/csv",
	"encoding": "encoding",

	// net
	"http":      "net/http",
	"url":       "net/url",
	"net":       "net",
	"rpc":       "net/rpc",
	"smtp":      "net/smtp",
	"mail":      "net/mail",
	"textproto": "net/textproto",

	// crypto
	"crypto": "crypto",
	"md5":    "crypto/md5",
	"sha1":   "crypto/sha1",
	"sha256": "crypto/sha256",
	"sha512": "crypto/sha512",
	"aes":    "crypto/aes",
	"cipher": "crypto/cipher",
	"rand":   "crypto/rand",
	"rsa":    "crypto/rsa",
	"tls":    "crypto/tls",
	"x509":   "crypto/x509",
	"hmac":   "crypto/hmac",

	// database
	"sql":    "database/sql",
	"driver": "database/sql/driver",

	// container
	"list": "container/list",
	"heap": "container/heap",
	"ring": "container/ring",

	// compress
	"gzip":  "compress/gzip",
	"zlib":  "compress/zlib",
	"flate": "compress/flate",
	"bzip2": "compress/bzip2",
	"lzw":   "compress/lzw",

	// archive
	"tar": "archive/tar",
	"zip": "archive/zip",

	// text
	"template":  "text/template",
	"scanner":   "text/scanner",
	"tabwriter": "text/tabwriter",

	// html
	"html": "html",

	// image
	"image": "image",
	"color": "image/color",
	"draw":  "image/draw",
	"png":   "image/png",
	"jpeg":  "image/jpeg",
	"gif":   "image/gif",

	// debug
	"dwarf":    "debug/dwarf",
	"elf":      "debug/elf",
	"gosym":    "debug/gosym",
	"macho":    "debug/macho",
	"pe":       "debug/pe",
	"plan9obj": "debug/plan9obj",

	// go
	"ast":      "go/ast",
	"build":    "go/build",
	"doc":      "go/doc",
	"format":   "go/format",
	"importer": "go/importer",
	"parser":   "go/parser",
	"printer":  "go/printer",
	"token":    "go/token",
	"types":    "go/types",

	// embed
	"embed": "embed",

	// slices & maps (Go 1.21+)
	"slices": "slices",
	"maps":   "maps",
	"cmp":    "cmp",

	// slog (Go 1.21+)
	"slog": "log/slog",

	// iter (Go 1.23+)
	"iter": "iter",
}

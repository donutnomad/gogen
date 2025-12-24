package plugin

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"

	"github.com/donutnomad/gg"
)

// ParseSourceToGG 将 Go 源代码解析并转换为 gg.Generator
// 这使得不使用 gg 库的生成器也能与 gg 框架集成
// 支持提取 imports 并与其他生成器的输出合并
func ParseSourceToGG(source []byte) (*gg.Generator, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析源代码失败: %w", err)
	}

	gen := gg.New()

	// 设置包名
	gen.SetPackage(file.Name.Name)

	// 提取并添加 imports
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if imp.Name != nil && imp.Name.Name != "" && imp.Name.Name != "_" {
			// 有别名的 import
			if imp.Name.Name == "." {
				// dot import - 暂不支持，跳过
				continue
			}
			gen.PAlias(importPath, imp.Name.Name)
		} else {
			gen.P(importPath)
		}
	}

	// 提取代码体（除了 package 和 import 之外的所有内容）
	body, err := extractBody(fset, file, source)
	if err != nil {
		return nil, fmt.Errorf("提取代码体失败: %w", err)
	}

	if body != "" {
		gen.Body().Append(gg.String("%s", body))
	}

	return gen, nil
}

// extractBody 提取代码体（声明部分）
func extractBody(fset *token.FileSet, file *ast.File, source []byte) (string, error) {
	var parts []string

	for _, decl := range file.Decls {
		// 跳过 import 声明
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			continue
		}

		// 获取声明的源代码
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, decl); err != nil {
			return "", err
		}
		parts = append(parts, buf.String())
	}

	return strings.Join(parts, "\n\n"), nil
}

// ParseSourceToGGWithHeader 与 ParseSourceToGG 相同，但可以设置文件头注释
func ParseSourceToGGWithHeader(source []byte, headerFormat string, args ...any) (*gg.Generator, error) {
	gen, err := ParseSourceToGG(source)
	if err != nil {
		return nil, err
	}

	if headerFormat != "" {
		gen.SetHeader(headerFormat, args...)
	}

	return gen, nil
}

// MustParseSourceToGG 是 ParseSourceToGG 的 panic 版本
func MustParseSourceToGG(source []byte) *gg.Generator {
	gen, err := ParseSourceToGG(source)
	if err != nil {
		panic(err)
	}
	return gen
}

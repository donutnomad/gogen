package plugin

import (
	"fmt"
	"go/ast"
	"go/parser"
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

// extractBody 提取代码体（import 之后的所有内容，包括注释）
func extractBody(fset *token.FileSet, file *ast.File, source []byte) (string, error) {
	// 找到 body 的起始位置（最后一个 import 之后，或 package 声明之后）
	var bodyStart token.Pos

	// 首先找到最后一个 import 声明的结束位置
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			if genDecl.End() > bodyStart {
				bodyStart = genDecl.End()
			}
		}
	}

	// 如果没有 import，从 package 声明之后开始
	if bodyStart == 0 {
		bodyStart = file.Name.End()
	}

	// 转换为字节偏移量
	startOffset := fset.Position(bodyStart).Offset

	// 确保不越界
	if startOffset >= len(source) {
		return "", nil
	}

	// 提取 body 部分
	body := string(source[startOffset:])

	// 去除开头的空白行，但保留后续内容的格式
	body = strings.TrimLeft(body, " \t\n\r")

	return body, nil
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

package utils

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/imports"
)

func WriteFormat(fileName string, src []byte) error {
	// 先移除未使用的 import（不自动添加缺失的）
	src = removeUnusedImports(src)

	bs, err := imports.Process(fileName, src, &imports.Options{
		Fragment:   true,
		AllErrors:  true,
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
		FormatOnly: false,
	})
	if err != nil {
		fmt.Println("format file failed:")
		lines := strings.Split(string(src), "\n")
		for i, line := range lines {
			fmt.Printf("%d: %s\n", i+1, line)
		}
		return err
	}
	// 输出到文件中
	return os.WriteFile(fileName, bs, 0644)
}

// removeUnusedImports 移除未使用的 import，不自动添加缺失的 import
func removeUnusedImports(src []byte) []byte {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return src
	}

	// 收集所有 import 的包名/别名
	type importEntry struct {
		spec    *ast.ImportSpec
		pkgName string // 实际使用的包名（别名或路径最后一段）
	}

	var entries []importEntry
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)

		// 跳过 blank import（ _ "pkg"）和 dot import（. "pkg"）
		if imp.Name != nil {
			name := imp.Name.Name
			if name == "_" || name == "." {
				continue
			}
			entries = append(entries, importEntry{spec: imp, pkgName: name})
		} else {
			parts := strings.Split(path, "/")
			entries = append(entries, importEntry{spec: imp, pkgName: parts[len(parts)-1]})
		}
	}

	if len(entries) == 0 {
		return src
	}

	// 收集 AST 中所有被引用的包名（SelectorExpr 的 X 部分）
	usedPkgs := make(map[string]bool)
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		usedPkgs[ident.Name] = true
		return true
	})

	// 找出未使用的 import
	var unused []*ast.ImportSpec
	for _, e := range entries {
		if !usedPkgs[e.pkgName] {
			unused = append(unused, e.spec)
		}
	}

	if len(unused) == 0 {
		return src
	}

	// 从源码中删除未使用的 import 行
	lines := strings.Split(string(src), "\n")
	removeLines := make(map[int]bool)
	for _, spec := range unused {
		line := fset.Position(spec.Pos()).Line
		removeLines[line] = true
	}

	var result []string
	for i, line := range lines {
		if !removeLines[i+1] { // 行号从 1 开始
			result = append(result, line)
		}
	}

	return []byte(strings.Join(result, "\n"))
}

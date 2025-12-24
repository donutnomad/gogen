package structparse

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// ParseStruct 解析指定文件中的结构体（包级便捷函数，保持向后兼容）
func ParseStruct(filename, structName string) (*StructInfo, error) {
	ctx := NewParseContext()
	return ctx.ParseStruct(filename, structName)
}

// ParseStruct 解析指定文件中的结构体（ParseContext 方法）
func (c *ParseContext) ParseStruct(filename, structName string) (*StructInfo, error) {
	// 解析当前文件的导入信息
	imports, err := c.extractImports(filename)
	if err != nil {
		return nil, err
	}
	return c.parseStructWithStackAndImports(filename, structName, make(map[string]bool), imports)
}

// parseStructWithStackAndImports 带栈和导入信息的结构体解析（向后兼容）
func (c *ParseContext) parseStructWithStackAndImports(filename, structName string, stack map[string]bool, imports map[string]*ImportInfo) (*StructInfo, error) {
	return c.parseStructWithStackAndImportsAndBaseDir(filename, structName, stack, imports, filepath.Dir(filename))
}

// parseStructWithStackAndImportsAndBaseDir 带栈、导入信息和基础目录的结构体解析
func (c *ParseContext) parseStructWithStackAndImportsAndBaseDir(filename, structName string, stack map[string]bool, imports map[string]*ImportInfo, baseDir string) (*StructInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析文件失败: %w", err)
	}

	structInfo := &StructInfo{
		Name:        structName,
		PackageName: node.Name.Name,
		FilePath:    filename,
	}

	// 收集导入信息
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		structInfo.Imports = append(structInfo.Imports, importPath)
	}

	// 查找目标结构体
	var targetStruct *ast.StructType
	ast.Inspect(node, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok {
			if genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name.Name == structName {
							if structType, ok := typeSpec.Type.(*ast.StructType); ok {
								targetStruct = structType
								return false
							}
						}
					}
				}
			}
		}
		return true
	})

	if targetStruct == nil {
		return nil, fmt.Errorf("未找到结构体 %s", structName)
	}

	// 提取当前文件的 imports 用于解析字段类型的 PkgPath
	// 这是关键修复：嵌入的外部包结构体（如 gorm.Model）的字段类型需要使用该文件自己的 imports
	currentFileImports, err := c.extractImports(filename)
	if err != nil {
		return nil, err
	}

	// 解析字段（使用当前文件的 imports 解析 PkgPath，使用传入的 imports 解析嵌入类型）
	fields, err := c.parseStructFieldsWithStackAndImportsAndBaseDir(targetStruct.Fields.List, stack, currentFileImports, baseDir)
	if err != nil {
		return nil, err
	}
	structInfo.Fields = fields

	// 解析方法信息 - 需要搜索整个包中的所有文件
	methods, err := parseMethodsFromPackage(filename, structName)
	if err != nil {
		return nil, err
	}
	structInfo.Methods = methods

	return structInfo, nil
}

// parseStructWithStack 带栈的结构体解析（保留向后兼容）
func (c *ParseContext) parseStructWithStack(filename, structName string, stack map[string]bool) (*StructInfo, error) {
	// 提取当前文件的导入信息
	imports, err := c.extractImports(filename)
	if err != nil {
		return nil, err
	}
	return c.parseStructWithStackAndImports(filename, structName, stack, imports)
}

package structparse

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// extractImports 提取文件中的导入信息（ParseContext 方法）
func (c *ParseContext) extractImports(filename string) (map[string]*ImportInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	imports := make(map[string]*ImportInfo)
	resolver := c.GetResolver()

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")

		var alias string
		if imp.Name != nil {
			// 有显式别名
			alias = imp.Name.Name
		} else {
			// 没有显式别名，使用路径最后一部分作为临时 key
			parts := strings.Split(importPath, "/")
			alias = parts[len(parts)-1]
		}

		// 获取真实包名
		var packageName string
		if resolver != nil {
			packageName, _ = resolver.GetPackageName(importPath)
		}
		if packageName == "" {
			// 降级：使用路径最后一部分
			packageName = filepath.Base(importPath)
		}

		imports[alias] = &ImportInfo{
			Alias:       alias,
			PackageName: packageName,
			ImportPath:  importPath,
		}

		// 如果真实包名不等于别名，也添加一个真实包名的映射
		if packageName != alias {
			imports[packageName] = &ImportInfo{
				Alias:       "",
				PackageName: packageName,
				ImportPath:  importPath,
			}
		}
	}

	return imports, nil
}

// extractPkgPath 从字段类型提取包路径
func extractPkgPath(fieldType string, imports map[string]*ImportInfo) string {
	// 移除修饰符（指针、切片等）
	cleanType := strings.TrimPrefix(fieldType, "*")
	cleanType = strings.TrimPrefix(cleanType, "[]")
	cleanType = strings.TrimPrefix(cleanType, "map[")

	// 检查是否有包前缀
	dotIdx := strings.Index(cleanType, ".")
	if dotIdx <= 0 {
		// 没有包前缀，是本包类型或内置类型
		return ""
	}

	// 提取包前缀
	pkgPrefix := cleanType[:dotIdx]

	// 从 imports 中查找对应的导入路径
	if info, exists := imports[pkgPrefix]; exists {
		return info.ImportPath
	}

	// 未找到，返回空
	return ""
}

// parseTypePackageAndName 解析类型的包名和结构体名
// 输入: "orm.Model" 返回: "orm", "Model"
// 输入: "User" 返回: "", "User"
func parseTypePackageAndName(typeName string) (packageName, structName string) {
	parts := strings.Split(typeName, ".")
	if len(parts) == 1 {
		return "", parts[0]
	}
	return parts[0], parts[1]
}

// findImportPath 从文件中查找指定包的完整import路径
func findImportPath(filename, packageAlias string) (string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return "", err
	}

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")

		// 检查是否有别名
		if imp.Name != nil {
			if imp.Name.Name == packageAlias {
				return importPath, nil
			}
		} else {
			// 没有别名，使用路径最后一部分作为包名
			parts := strings.Split(importPath, "/")
			if len(parts) > 0 && parts[len(parts)-1] == packageAlias {
				return importPath, nil
			}
		}
	}

	return "", nil
}

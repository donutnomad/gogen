package pickgen

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/donutnomad/gogen/internal/structparse"
)

// parseSourceParam 解析 source 参数
// 支持格式:
//   - "pkg.Type"：当前文件导入的包
//   - "github.com/user/repo/pkg.Type"：完整路径
//
// 返回: pkgPath, typeName, alias, error
func parseSourceParam(source, currentFilePath string) (string, string, string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", "", "", fmt.Errorf("source 参数不能为空")
	}

	// 查找最后一个 "." 来分隔包和类型
	lastDot := strings.LastIndex(source, ".")
	if lastDot == -1 {
		// 没有 "."，可能是当前包内的类型
		return "", source, "", nil
	}

	pkgPart := source[:lastDot]
	typeName := source[lastDot+1:]

	if typeName == "" {
		return "", "", "", fmt.Errorf("类型名不能为空: %s", source)
	}

	// 判断是否是完整路径（包含 "/" 表示完整路径）
	if strings.Contains(pkgPart, "/") {
		// 完整路径，如 "github.com/user/repo/pkg"
		// 提取 alias（路径最后一段）并清理为有效的 Go 标识符
		alias := sanitizeAlias(filepath.Base(pkgPart))
		return pkgPart, typeName, alias, nil
	}

	// 短路径，如 "pkg" 或 "gorm"，需要从当前文件的导入中查找
	// 这种情况下，pkgPart 就是包别名
	imports, err := extractFileImports(currentFilePath)
	if err != nil {
		return "", "", "", fmt.Errorf("解析导入失败: %w", err)
	}

	// 查找匹配的导入
	if importInfo, ok := imports[pkgPart]; ok {
		return importInfo.ImportPath, typeName, importInfo.Alias, nil
	}

	// 没找到导入，可能是域名格式但没有 "/"，如 "gorm.io"
	// 尝试将整个 source 解析为 "域名.Type" 格式
	if strings.Contains(pkgPart, ".") {
		// 看起来像是域名格式，但缺少完整路径
		return "", "", "", fmt.Errorf("无法解析 source 参数 %q，如果是第三方包请使用完整路径（如 github.com/xxx/pkg.Type）", source)
	}

	return "", "", "", fmt.Errorf("未找到包 %q 的导入，请使用完整路径或确保已导入该包", pkgPart)
}

// importInfo 导入信息
type importInfo struct {
	Alias      string
	ImportPath string
}

// extractFileImports 提取文件中的导入信息
func extractFileImports(filename string) (map[string]*importInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	imports := make(map[string]*importInfo)

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")

		var alias string
		if imp.Name != nil {
			// 有显式别名
			alias = imp.Name.Name
		} else {
			// 没有显式别名，使用路径最后一部分
			alias = filepath.Base(importPath)
		}

		imports[alias] = &importInfo{
			Alias:      alias,
			ImportPath: importPath,
		}
	}

	return imports, nil
}

// resolveExternalStruct 解析外部包的结构体
func resolveExternalStruct(pkgPath, typeName, currentFilePath string) (*structparse.StructInfo, error) {
	// 首先尝试解析为本地模块包
	diskPath, err := resolvePackagePath(pkgPath, currentFilePath)
	if err != nil {
		return nil, fmt.Errorf("查找包 %s 失败: %w", pkgPath, err)
	}

	// 在包目录中查找包含目标结构体的文件
	structFile, err := findStructFile(diskPath, typeName)
	if err != nil {
		return nil, fmt.Errorf("在包 %s 中查找结构体 %s 失败: %w", pkgPath, typeName, err)
	}

	// 解析结构体
	return structparse.ParseStruct(structFile, typeName)
}

// resolvePackagePath 解析包路径，支持本地模块包和第三方包
func resolvePackagePath(pkgPath, currentFilePath string) (string, error) {
	// 首先尝试找到项目根目录
	projectRoot, err := findProjectRootFromFile(currentFilePath)
	if err == nil {
		// 读取go.mod获取模块名称
		moduleName, err := getModuleNameFromRoot(projectRoot)
		if err == nil && strings.HasPrefix(pkgPath, moduleName) {
			// 这是本地模块包
			relativePath := strings.TrimPrefix(pkgPath, moduleName)
			relativePath = strings.TrimPrefix(relativePath, "/")
			packagePath := filepath.Join(projectRoot, relativePath)

			if info, err := os.Stat(packagePath); err == nil && info.IsDir() {
				return packagePath, nil
			}
		}
	}

	// 不是本地包，尝试作为第三方包查找
	return structparse.FindThirdPartyPackage(pkgPath)
}

// findProjectRootFromFile 从文件路径查找项目根目录
func findProjectRootFromFile(filePath string) (string, error) {
	dir := filepath.Dir(filePath)
	if !filepath.IsAbs(dir) {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return "", err
		}
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("未找到项目根目录（go.mod文件）")
}

// getModuleNameFromRoot 从项目根目录获取模块名
func getModuleNameFromRoot(projectRoot string) (string, error) {
	goModPath := filepath.Join(projectRoot, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	return "", fmt.Errorf("未在 go.mod 中找到模块名称")
}

// findStructFile 在目录中查找包含指定结构体的文件
func findStructFile(dir, structName string) (string, error) {
	files, err := structparse.FindGoFiles(dir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if structparse.ContainsStruct(file, structName) {
			return file, nil
		}
	}

	return "", fmt.Errorf("未找到结构体 %s", structName)
}

// sanitizeAlias 将包名转换为有效的 Go 标识符
// Go 标识符只能包含字母、数字和下划线，不能以数字开头
// 按照 Go 包命名惯例，移除连字符（而非转换为下划线）
func sanitizeAlias(name string) string {
	var result strings.Builder
	for i, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' {
			result.WriteRune(r)
		} else if r >= '0' && r <= '9' {
			if i == 0 {
				// 不能以数字开头，添加下划线前缀
				result.WriteRune('_')
			}
			result.WriteRune(r)
		}
		// 连字符和其他非法字符直接忽略（符合 Go 包命名惯例）
	}
	return result.String()
}

package mockgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/donutnomad/gogen/internal/pkgresolver"
	"github.com/donutnomad/gogen/internal/structparse"
)

// InterfaceInfo 接口信息
type InterfaceInfo struct {
	Name        string
	PackageName string
	FilePath    string
	Methods     []*MethodInfo
	TypeParams  []*TypeParamInfo  // 泛型参数
	Imports     map[string]string // 导入路径 -> 包名
}

// MethodInfo 方法信息
type MethodInfo struct {
	Name     string
	Params   []*ParamInfo
	Results  []*ParamInfo
	Variadic *ParamInfo // 可变参数
}

// ParamInfo 参数/返回值信息
type ParamInfo struct {
	Name string
	Type string
}

// TypeParamInfo 泛型参数信息
type TypeParamInfo struct {
	Name       string
	Constraint string
}

// ImportInfo 导入信息
type ImportInfo struct {
	Alias      string // 显式别名
	ImportPath string // 完整导入路径
}

// interfaceParser 接口解析器
type interfaceParser struct {
	fset        *token.FileSet
	file        *ast.File
	filePath    string
	baseDir     string                        // 原始文件所在目录
	projectRoot string                        // 项目根目录
	imports     map[string]*ImportInfo        // 别名/包名 -> 导入信息
	interfaces  map[string]*ast.InterfaceType // 接口名 -> 接口类型
	typeSpecs   map[string]*ast.TypeSpec      // 类型名 -> 类型定义
	parsed      map[string]bool               // 已解析的接口（防止循环），key 格式: "pkgPath:interfaceName"
	stdLib      *pkgresolver.StdLibScanner    // 标准库扫描器
	pkgPath     string                        // 当前包路径（用于循环检测）
}

// 全局标准库扫描器（延迟初始化）
var (
	globalStdLib     *pkgresolver.StdLibScanner
	globalStdLibOnce sync.Once
)

func getStdLibScanner() *pkgresolver.StdLibScanner {
	globalStdLibOnce.Do(func() {
		globalStdLib = pkgresolver.NewStdLibScanner()
	})
	return globalStdLib
}

// ParseInterface 解析指定文件中的接口
func ParseInterface(filePath, interfaceName string) (*InterfaceInfo, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析文件失败: %w", err)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}
	baseDir := filepath.Dir(absPath)

	// 查找项目根目录
	projectRoot, err := findProjectRootFromDir(baseDir)
	if err != nil {
		// 如果找不到 go.mod，使用当前目录
		projectRoot = baseDir
	}

	p := &interfaceParser{
		fset:        fset,
		file:        file,
		filePath:    filePath,
		baseDir:     baseDir,
		projectRoot: projectRoot,
		imports:     make(map[string]*ImportInfo),
		interfaces:  make(map[string]*ast.InterfaceType),
		typeSpecs:   make(map[string]*ast.TypeSpec),
		parsed:      make(map[string]bool),
		stdLib:      getStdLibScanner(),
		pkgPath:     "local", // 本地包的标识
	}

	// 收集导入
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		var alias string
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			// 从路径提取包名
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}
		p.imports[alias] = &ImportInfo{
			Alias:      alias,
			ImportPath: path,
		}
	}

	// 收集文件中所有接口定义
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			p.typeSpecs[typeSpec.Name.Name] = typeSpec
			if iface, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				p.interfaces[typeSpec.Name.Name] = iface
			}
		}
	}

	// 查找目标接口
	targetSpec, ok := p.typeSpecs[interfaceName]
	if !ok {
		return nil, fmt.Errorf("未找到接口 %s", interfaceName)
	}

	targetInterface, ok := p.interfaces[interfaceName]
	if !ok {
		return nil, fmt.Errorf("%s 不是接口类型", interfaceName)
	}

	// 转换导入信息
	importsMap := make(map[string]string)
	for alias, info := range p.imports {
		importsMap[info.ImportPath] = alias
	}

	info := &InterfaceInfo{
		Name:        interfaceName,
		PackageName: file.Name.Name,
		FilePath:    filePath,
		Imports:     importsMap,
	}

	// 解析泛型参数
	if targetSpec.TypeParams != nil {
		info.TypeParams = parseTypeParams(targetSpec.TypeParams)
	}

	// 解析方法（包括嵌入接口的方法）
	methods, err := p.parseInterfaceMethods(targetInterface, interfaceName)
	if err != nil {
		return nil, err
	}
	info.Methods = methods

	return info, nil
}

// parseInterfaceMethods 解析接口的所有方法（包括嵌入接口）
func (p *interfaceParser) parseInterfaceMethods(iface *ast.InterfaceType, ifaceName string) ([]*MethodInfo, error) {
	if iface.Methods == nil {
		return nil, nil
	}

	// 使用 "pkgPath:interfaceName" 格式作为 key，防止跨包循环引用
	parseKey := p.pkgPath + ":" + ifaceName
	if p.parsed[parseKey] {
		// 已解析过，防止循环引用
		return nil, nil
	}
	p.parsed[parseKey] = true

	var methods []*MethodInfo

	for _, field := range iface.Methods.List {
		if len(field.Names) == 0 {
			// 嵌入接口
			embeddedMethods, err := p.parseEmbeddedInterface(field.Type)
			if err != nil {
				return nil, err
			}
			methods = append(methods, embeddedMethods...)
		} else {
			// 直接定义的方法
			funcType, ok := field.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			method := p.parseMethod(field.Names[0].Name, funcType)
			methods = append(methods, method)
		}
	}

	return methods, nil
}

// parseEmbeddedInterface 解析嵌入的接口
func (p *interfaceParser) parseEmbeddedInterface(expr ast.Expr) ([]*MethodInfo, error) {
	switch e := expr.(type) {
	case *ast.Ident:
		// 同一包内的接口，如 Reader
		return p.parseLocalInterface(e.Name)

	case *ast.SelectorExpr:
		// 外部包的接口，如 io.Reader
		pkgIdent, ok := e.X.(*ast.Ident)
		if !ok {
			return nil, nil
		}
		return p.parseExternalInterface(pkgIdent.Name, e.Sel.Name)

	case *ast.IndexExpr:
		// 泛型接口，如 Container[T]
		return p.parseEmbeddedInterface(e.X)

	case *ast.IndexListExpr:
		// 多参数泛型接口，如 Map[K, V]
		return p.parseEmbeddedInterface(e.X)

	default:
		return nil, nil
	}
}

// parseLocalInterface 解析同一文件中的接口
func (p *interfaceParser) parseLocalInterface(name string) ([]*MethodInfo, error) {
	// 使用 "pkgPath:interfaceName" 格式检查是否已解析（防止循环引用）
	parseKey := p.pkgPath + ":" + name
	if p.parsed[parseKey] {
		return nil, nil
	}

	iface, ok := p.interfaces[name]
	if !ok {
		// 本地接口未找到，可能是外部定义的接口或类型别名
		return nil, nil
	}

	return p.parseInterfaceMethods(iface, name)
}

// parseExternalInterface 解析外部包的接口
func (p *interfaceParser) parseExternalInterface(pkgAlias, ifaceName string) ([]*MethodInfo, error) {
	// 查找包的导入路径
	importInfo, ok := p.imports[pkgAlias]
	if !ok {
		// 导入信息未找到，可能是内置类型或未导入的包
		return nil, nil
	}
	importPath := importInfo.ImportPath

	// 定位包目录
	pkgDir, err := p.findPackagePath(importPath)
	if err != nil {
		// 无法定位包，可能是不支持的包类型（如 cgo 生成的包）
		return nil, nil
	}

	// 解析包目录中的文件，查找接口
	return p.parseInterfaceFromDir(pkgDir, ifaceName, importPath)
}

// findPackagePath 根据导入路径查找包目录
func (p *interfaceParser) findPackagePath(importPath string) (string, error) {
	// 1. 检查是否是标准库
	isStd, err := p.stdLib.IsStdLib(importPath)
	if err == nil && isStd {
		return p.stdLib.GetStdLibPath(importPath)
	}

	// 2. 检查是否是项目内部包
	moduleName, err := getModuleName(p.projectRoot)
	if err == nil && strings.HasPrefix(importPath, moduleName) {
		relativePath := strings.TrimPrefix(importPath, moduleName)
		relativePath = strings.TrimPrefix(relativePath, "/")
		packagePath := filepath.Join(p.projectRoot, relativePath)

		if _, err := os.Stat(packagePath); err == nil {
			return packagePath, nil
		}
	}

	// 3. 第三方包：从 Go 模块缓存中查找
	return structparse.FindThirdPartyPackage(importPath)
}

// parseInterfaceFromDir 从目录中解析接口
// pkgPath 是导入路径，用于循环引用检测
func (p *interfaceParser) parseInterfaceFromDir(pkgDir, ifaceName, pkgPath string) ([]*MethodInfo, error) {
	fset := token.NewFileSet()

	// 读取目录中的 Go 文件
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		// 目录读取失败，返回错误而不是静默忽略
		return nil, fmt.Errorf("读取目录 %s 失败: %w", pkgDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, name)
		file, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			// 解析单个文件失败，继续尝试其他文件
			continue
		}

		// 查找接口定义
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != ifaceName {
					continue
				}

				iface, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}

				// 创建临时解析器来解析外部包的接口
				// 关键修复：共享 parsed map 以正确检测跨包循环引用
				tempParser := &interfaceParser{
					fset:        fset,
					file:        file,
					filePath:    filePath,
					baseDir:     pkgDir,
					projectRoot: p.projectRoot,
					imports:     make(map[string]*ImportInfo),
					interfaces:  make(map[string]*ast.InterfaceType),
					typeSpecs:   make(map[string]*ast.TypeSpec),
					parsed:      p.parsed, // 共享 parsed map
					stdLib:      p.stdLib,
					pkgPath:     pkgPath, // 使用导入路径作为包标识
				}

				// 收集该文件的导入
				for _, imp := range file.Imports {
					path := strings.Trim(imp.Path.Value, `"`)
					var alias string
					if imp.Name != nil {
						alias = imp.Name.Name
					} else {
						parts := strings.Split(path, "/")
						alias = parts[len(parts)-1]
					}
					tempParser.imports[alias] = &ImportInfo{
						Alias:      alias,
						ImportPath: path,
					}
				}

				// 收集文件中的接口
				for _, d := range file.Decls {
					gd, ok := d.(*ast.GenDecl)
					if !ok || gd.Tok != token.TYPE {
						continue
					}
					for _, s := range gd.Specs {
						ts, ok := s.(*ast.TypeSpec)
						if !ok {
							continue
						}
						tempParser.typeSpecs[ts.Name.Name] = ts
						if i, ok := ts.Type.(*ast.InterfaceType); ok {
							tempParser.interfaces[ts.Name.Name] = i
						}
					}
				}

				return tempParser.parseInterfaceMethods(iface, ifaceName)
			}
		}
	}

	// 接口未在包中找到，返回 nil（这是预期的情况，如接口在其他文件中）
	return nil, nil
}

// parseMethod 解析单个方法
func (p *interfaceParser) parseMethod(name string, funcType *ast.FuncType) *MethodInfo {
	method := &MethodInfo{
		Name: name,
	}

	// 解析参数
	if funcType.Params != nil && len(funcType.Params.List) > 0 {
		// 检查最后一个参数是否是可变参数
		lastField := funcType.Params.List[len(funcType.Params.List)-1]
		if ellipsis, ok := lastField.Type.(*ast.Ellipsis); ok {
			// 有可变参数
			regularParams := funcType.Params.List[:len(funcType.Params.List)-1]
			method.Params = parseParams(regularParams)

			// 解析可变参数
			variadicType := exprToString(ellipsis.Elt)
			var variadicName string
			if len(lastField.Names) > 0 {
				variadicName = lastField.Names[0].Name
			}
			method.Variadic = &ParamInfo{
				Name: variadicName,
				Type: variadicType,
			}
		} else {
			method.Params = parseParams(funcType.Params.List)
		}
	}

	// 解析返回值
	if funcType.Results != nil {
		method.Results = parseParams(funcType.Results.List)
	}

	return method
}

// parseTypeParams 解析类型参数列表
func parseTypeParams(fieldList *ast.FieldList) []*TypeParamInfo {
	if fieldList == nil {
		return nil
	}

	var params []*TypeParamInfo
	for _, field := range fieldList.List {
		constraint := exprToString(field.Type)
		for _, name := range field.Names {
			params = append(params, &TypeParamInfo{
				Name:       name.Name,
				Constraint: constraint,
			})
		}
	}
	return params
}

// parseParams 解析参数列表
func parseParams(fields []*ast.Field) []*ParamInfo {
	var params []*ParamInfo

	for _, field := range fields {
		typeStr := exprToString(field.Type)

		if len(field.Names) == 0 {
			// 匿名参数
			params = append(params, &ParamInfo{
				Type: typeStr,
			})
		} else {
			// 命名参数
			for _, name := range field.Names {
				params = append(params, &ParamInfo{
					Name: name.Name,
					Type: typeStr,
				})
			}
		}
	}

	return params
}

// exprToString 将 AST 表达式转换为字符串
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + exprToString(e.Elt)
		}
		return "[" + exprToString(e.Len) + "]" + exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)
	case *ast.ChanType:
		switch e.Dir {
		case ast.SEND:
			return "chan<- " + exprToString(e.Value)
		case ast.RECV:
			return "<-chan " + exprToString(e.Value)
		default:
			return "chan " + exprToString(e.Value)
		}
	case *ast.FuncType:
		return funcTypeToString(e)
	case *ast.InterfaceType:
		if e.Methods == nil || len(e.Methods.List) == 0 {
			return "any"
		}
		return "interface{}"
	case *ast.StructType:
		if e.Fields == nil || len(e.Fields.List) == 0 {
			return "struct{}"
		}
		return "struct{...}"
	case *ast.Ellipsis:
		return "..." + exprToString(e.Elt)
	case *ast.BasicLit:
		return e.Value
	case *ast.IndexExpr:
		// 泛型类型 T[U]
		return exprToString(e.X) + "[" + exprToString(e.Index) + "]"
	case *ast.IndexListExpr:
		// 泛型类型 T[U, V]
		var indices []string
		for _, idx := range e.Indices {
			indices = append(indices, exprToString(idx))
		}
		return exprToString(e.X) + "[" + strings.Join(indices, ", ") + "]"
	case *ast.ParenExpr:
		return "(" + exprToString(e.X) + ")"
	case *ast.UnaryExpr:
		return e.Op.String() + exprToString(e.X)
	case *ast.BinaryExpr:
		return exprToString(e.X) + " " + e.Op.String() + " " + exprToString(e.Y)
	default:
		return "any"
	}
}

// funcTypeToString 将函数类型转换为字符串
func funcTypeToString(ft *ast.FuncType) string {
	var params []string
	if ft.Params != nil {
		for _, p := range ft.Params.List {
			typeStr := exprToString(p.Type)
			if len(p.Names) == 0 {
				params = append(params, typeStr)
			} else {
				for range p.Names {
					params = append(params, typeStr)
				}
			}
		}
	}

	var results []string
	if ft.Results != nil {
		for _, r := range ft.Results.List {
			typeStr := exprToString(r.Type)
			if len(r.Names) == 0 {
				results = append(results, typeStr)
			} else {
				for range r.Names {
					results = append(results, typeStr)
				}
			}
		}
	}

	result := "func(" + strings.Join(params, ", ") + ")"
	if len(results) == 1 {
		result += " " + results[0]
	} else if len(results) > 1 {
		result += " (" + strings.Join(results, ", ") + ")"
	}

	return result
}

// findProjectRootFromDir 从指定目录开始查找项目根目录（包含go.mod的目录）
func findProjectRootFromDir(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
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

	return "", fmt.Errorf("未找到项目根目录（go.mod文件）从 %s 开始", startDir)
}

// getModuleName 从go.mod文件获取模块名称
func getModuleName(projectRoot string) (string, error) {
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

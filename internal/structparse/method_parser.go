package structparse

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/donutnomad/gogen/internal/xast"
)

// parseMethodsFromPackage 从包中的所有文件解析指定结构体的方法
func parseMethodsFromPackage(targetFile, structName string) ([]MethodInfo, error) {
	var allMethods []MethodInfo

	// 获取目标文件所在的目录
	dir := filepath.Dir(targetFile)

	// 查找目录中的所有Go文件
	files, err := findGoFilesInDir(dir)
	if err != nil {
		return nil, fmt.Errorf("查找包文件失败: %v", err)
	}

	// 遍历所有文件
	for _, file := range files {
		// 首先用字符串匹配检查文件是否可能包含该结构体的方法
		mayContain := fileMayContainStructMethods(file, structName)
		if !mayContain {
			continue
		}

		// 解析这个文件中的方法
		methods, err := parseMethodsFromFile(file, structName)
		if err != nil {
			// 记录错误但继续处理其他文件
			continue
		}
		allMethods = append(allMethods, methods...)
	}

	return allMethods, nil
}

// fileMayContainStructMethods 检查文件是否可能包含指定结构体的方法
func fileMayContainStructMethods(filename, structName string) bool {
	content, err := os.ReadFile(filename)
	if err != nil {
		return false
	}

	contentStr := string(content)

	// 检查是否包含该结构体作为接收器的方法
	// 使用更灵活的匹配，考虑接收器名称和空格
	patterns := []string{
		fmt.Sprintf("*%s)", structName), // 指针接收器，如 (u *User)
		fmt.Sprintf("%s)", structName),  // 值接收器，如 (u User)
	}

	for _, pattern := range patterns {
		if strings.Contains(contentStr, pattern) {
			return true
		}
	}

	return false
}

// parseMethodsFromFile 从单个文件解析指定结构体的方法
func parseMethodsFromFile(filename, structName string) ([]MethodInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析文件失败: %w", err)
	}

	// 获取文件的绝对路径
	absPath, err := filepath.Abs(filename)
	if err != nil {
		absPath = filename // 如果获取绝对路径失败，使用原始路径
	}

	var methods []MethodInfo

	ast.Inspect(node, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// 检查是否是方法（有接收器）
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				// 获取接收器信息
				recv := funcDecl.Recv.List[0]

				// 获取接收器类型和名称
				var recvType, recvName string

				// 处理接收器名称
				if len(recv.Names) > 0 {
					recvName = recv.Names[0].Name
				}

				// 处理接收器类型
				if starExpr, ok := recv.Type.(*ast.StarExpr); ok {
					// 指针类型接收器，如 *User
					if ident, ok := starExpr.X.(*ast.Ident); ok {
						recvType = "*" + ident.Name
					}
				} else if ident, ok := recv.Type.(*ast.Ident); ok {
					// 值类型接收器，如 User
					recvType = ident.Name
				}

				// 检查是否是指定结构体的方法
				if recvType == structName || recvType == "*"+structName {
					method := MethodInfo{
						Name:         funcDecl.Name.Name,
						ReceiverName: recvName,
						ReceiverType: recvType,
						FilePath:     absPath,
					}

					// 解析返回类型
					if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
						var returnTypes []string
						for _, result := range funcDecl.Type.Results.List {
							resultType := xast.GetFieldType(result.Type, nil)
							returnTypes = append(returnTypes, resultType)
						}
						if len(returnTypes) > 0 {
							method.ReturnType = strings.Join(returnTypes, ", ")
						}
					}

					methods = append(methods, method)
				}
			}
		}
		return true
	})

	return methods, nil
}

// findGoFilesInDir 查找目录中的所有Go文件（包括当前目录）
func findGoFilesInDir(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 包含所有.go文件，但不包含测试文件
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

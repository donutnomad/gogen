// Package structparse 提供了强大的Go结构体静态分析工具。
//
// 本包支持以下核心功能：
//
//  1. 结构体字段解析 - 提取结构体的所有字段信息，包括类型、标签等
//  2. 嵌入字段展开 - 自动展开匿名嵌入字段和gorm:"embedded"标签字段
//  3. 方法信息收集 - 扫描包内所有文件，收集结构体的方法信息
//  4. 跨包类型解析 - 解析来自其他包的类型引用，支持别名导入
//  5. 循环引用检测 - 使用栈机制避免嵌入字段的循环引用
//
// # 基本用法
//
// 最简单的使用方式是直接调用包级函数：
//
//	info, err := structparse.ParseStruct("path/to/file.go", "StructName")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 访问结构体信息
//	fmt.Printf("结构体: %s.%s\n", info.PackageName, info.Name)
//	for _, field := range info.Fields {
//	    fmt.Printf("  字段: %s %s\n", field.Name, field.Type)
//	}
//
// # 依赖注入与测试
//
// 为了支持测试和自定义行为，可以使用 ParseContext：
//
//	// 使用默认配置
//	ctx := structparse.NewParseContext()
//	info, err := ctx.ParseStruct(filename, structName)
//
//	// 使用自定义包解析器（用于测试）
//	mockResolver := &MyMockResolver{}
//	ctx := structparse.NewParseContextWithResolver(mockResolver)
//	info, err := ctx.ParseStruct(filename, structName)
//
// # 嵌入字段展开
//
// 本包会自动展开两种类型的嵌入字段：
//
//  1. 匿名嵌入字段
//     type User struct {
//     BaseModel  // 字段会被自动展开
//     Name string
//     }
//
//  2. gorm:"embedded" 标签字段
//     type User struct {
//     ID int64
//     Account Account `gorm:"embedded;embeddedPrefix:account_"`
//     // Account的字段会被展开，且带有 account_ 前缀
//     }
//
// 展开的字段会在 FieldInfo.SourceType 中标记来源，嵌入前缀会保存在 FieldInfo.EmbeddedPrefix 中。
//
// # 跨包类型解析
//
// 本包支持解析来自其他包的类型，包括：
//
//   - 项目内部包：通过go.mod自动定位
//   - 第三方包：从Go模块缓存中查找
//   - 标准库：直接识别但不展开
//   - 别名导入：正确处理 import alias "package/path"
//
// # 方法解析
//
// ParseStruct 会自动扫描结构体所在包的所有文件，收集该结构体的所有方法：
//
//	info, _ := structparse.ParseStruct("user.go", "User")
//	for _, method := range info.Methods {
//	    fmt.Printf("方法: %s %s.%s() %s\n",
//	        method.ReceiverType,
//	        info.Name,
//	        method.Name,
//	        method.ReturnType)
//	}
//
// # 架构设计
//
// 本包采用模块化设计，将不同职责拆分到独立文件：
//
//   - types.go:          数据类型定义
//   - context.go:        解析上下文和依赖注入
//   - parser.go:         核心解析入口
//   - importer.go:       导入信息提取
//   - embedded.go:       嵌入字段展开逻辑
//   - field_parser.go:   字段解析
//   - method_parser.go:  方法解析
//   - package_finder.go: 包查找和路径解析
//   - file_utils.go:     文件工具函数
//
// 这种设计使得每个模块职责清晰，易于测试和维护。
//
// # 性能考虑
//
// 本包针对性能进行了多项优化：
//
//   - 延迟初始化：PackageResolver只在需要时才创建
//   - 字符串匹配预筛选：在AST解析前先用字符串匹配筛选文件
//   - 循环引用检测：使用栈避免无限递归
//   - 深度限制：maxEmbeddingDepth 防止过深嵌套
//
// # 限制
//
//   - 不支持标准库类型的结构体展开（如time.Time会保持原样）
//   - 第三方包需要已安装到Go模块缓存
//   - 最大嵌套深度为10层
package structparse

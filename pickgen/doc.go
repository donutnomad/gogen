// Package pickgen 提供基于注解的结构体字段选择代码生成器。
//
// # 概述
//
// pickgen 支持两个注解：
//   - @Pick: 从源结构体中选择指定字段生成新结构体
//   - @Omit: 从源结构体中排除指定字段生成新结构体
//
// # 基本用法
//
// 在结构体上添加注解：
//
//	// @Pick(name=UserBasic, fields=`[ID,Name,Email]`)
//	// @Omit(name=UserPublic, fields=`[Password,Salt]`)
//	type User struct {
//	    ID       uint64
//	    Name     string
//	    Email    string
//	    Password string
//	    Salt     string
//	}
//
// 运行 gogen 后将生成：
//
//	type UserBasic struct {
//	    ID    uint64
//	    Name  string
//	    Email string
//	}
//
//	type UserPublic struct {
//	    ID    uint64
//	    Name  string
//	    Email string
//	}
//
// # 注解参数
//
// @Pick 和 @Omit 支持以下参数：
//
//	name    (必填) 生成的新结构体名称
//	fields  (必填) 字段列表，格式: [Field1,Field2,Field3]
//	source  (可选) 源结构体，格式: pkg.Type 或完整路径
//
// # 独立注解语法 (//go:gen:)
//
// 对于引用第三方包或其他包的类型，可以使用独立注解语法：
//
//	//go:gen: @Pick(name=GormBasic, source=`gorm.io/gorm.Model`, fields=`[ID,CreatedAt,UpdatedAt]`)
//	//go:gen: @Omit(name=UserPublic, source=`github.com/myapp/models.User`, fields=`[Password]`)
//
// 独立注解的特点：
//   - 不附加到任何声明上，可以放在文件任意位置
//   - 必须提供 source 参数指定源结构体
//   - 支持引用第三方包、本地模块其他包的类型
//
// # Source 参数格式
//
// source 参数支持以下格式：
//
//  1. 完整包路径:
//     source=`gorm.io/gorm.Model`
//     source=`github.com/user/repo/pkg.Type`
//
//  2. 当前文件已导入的包:
//     source=`models.User`     // 需要文件中有 import "xxx/models"
//     source=`gormModel.Model` // 需要文件中有 import gormModel "gorm.io/gorm"
//
//  3. 当前包内的类型:
//     source=`LocalType`       // 无包前缀
//
// # 包名处理
//
// 对于包含特殊字符的目录名，pickgen 会自动转换为有效的 Go 标识符：
//
//	目录名 special-pkg  -> 包别名 specialpkg
//	目录名 v2-api       -> 包别名 v2api
//	目录名 123pkg       -> 包别名 _123pkg (数字开头添加下划线前缀)
//
// 这与 Go 语言的包命名惯例一致（移除连字符而非转换为下划线）。
//
// # 生成的代码
//
// 对于每个 Pick/Omit 注解，生成器会创建：
//
//  1. 新结构体定义（包含字段和 tag）
//  2. From 方法（从源结构体复制字段值）
//  3. New 构造函数（创建新实例并调用 From）
//
// 示例：
//
//	type UserBasic struct {
//	    ID   uint64 `json:"id"`
//	    Name string `json:"name"`
//	}
//
//	func (t *UserBasic) From(src *User) {
//	    t.ID = src.ID
//	    t.Name = src.Name
//	}
//
//	func NewUserBasic(src *User) UserBasic {
//	    var result UserBasic
//	    result.From(src)
//	    return result
//	}
//
// # 示例目录
//
// 完整示例请参考 examples/ 目录：
//
//	examples/basic/       - 基础 Pick/Omit 用法
//	examples/multiple/    - 多个注解示例
//	examples/embedded/    - 嵌入字段处理
//	examples/remote/      - 第三方包引用 (//go:gen:)
//	examples/crossref/    - 跨包引用示例
//	examples/alias/       - 包别名场景
//	examples/special-pkg/ - 特殊包名处理
//	examples/v2-api/      - 版本化路径
package pickgen

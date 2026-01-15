package crossref

// 这个文件展示从其他 examples 子包引用类型

// 从 basic 包引用 User
//go:gen: @Pick(name=UserID, source=`github.com/donutnomad/gogen/pickgen/examples/basic.User`, fields=`[ID]`)

// 从 basic 包引用 Product，只取价格相关字段
//go:gen: @Pick(name=ProductPrice, source=`github.com/donutnomad/gogen/pickgen/examples/basic.Product`, fields=`[ID,Name,Price]`)

// 从 embedded 包引用 Employee
//go:gen: @Pick(name=EmployeeID, source=`github.com/donutnomad/gogen/pickgen/examples/embedded.Employee`, fields=`[ID,Name]`)

// 从 special-pkg 包引用（注意包名与文件夹名不同）
//go:gen: @Pick(name=ConfigKey, source=`github.com/donutnomad/gogen/pickgen/examples/special-pkg.Config`, fields=`[ID,Key,Value]`)

// 从 v2-api 包引用（版本化路径）
//go:gen: @Pick(name=RequestID, source=`github.com/donutnomad/gogen/pickgen/examples/v2-api.Request`, fields=`[ID,Method,Path]`)

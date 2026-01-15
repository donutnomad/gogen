package remote

// 这个文件展示对其他常用第三方库的 Pick/Omit 操作

// ===== samber/lo 相关类型 =====
// lo 库主要是泛型函数，没有太多可 Pick 的结构体

// ===== protobuf 生成的类型示例 =====
// 注意：protobuf 生成的结构体通常包含很多内部字段

// ===== grpc 相关类型 =====
// grpc.Server 等配置结构体

// ===== 实际应用示例：从第三方模型创建 DTO =====

// 假设我们引用了一个外部订单系统的模型
// 外部包路径: github.com/example/orders.Order
// 我们只需要其中的部分字段创建本地 DTO

// 示例：从外部订单模型提取摘要
//go:gen: @Pick(name=ExternalOrderSummary, source=`github.com/donutnomad/gogen/pickgen/examples/basic.Product`, fields=`[ID,Name,Price]`)

// 示例：从本项目其他包的类型进行 Pick
//go:gen: @Pick(name=UserFromBasic, source=`github.com/donutnomad/gogen/pickgen/examples/basic.User`, fields=`[ID,Name,Email]`)

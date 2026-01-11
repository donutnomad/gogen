package basic

//go:generate go run github.com/donutnomad/gogen gen ./...

// 基本流转测试：无审批标记，纯直接流转
// @StateFlow(name="Order")
// @Flow: Created     => [ Paid ]
// @Flow: Paid        => [ Shipped ]
// @Flow: Shipped     => [ Delivered ]
// @Flow: Delivered   => [ Completed ]
const _ = ""

package lowercase

//go:generate go run github.com/donutnomad/gogen gen ./...

// 小写状态名测试
// Flow 中的状态名是小写的，生成的字符串值保持小写
// 但 Go 代码命名仍然首字母大写（符合 Go 规范）
// @StateFlow(name="Ticket")
// @Flow: open       => [ pending ]
// @Flow: pending    => [ resolved, rejected ]
// @Flow: rejected   => [ open ]
// @Flow: resolved   => [ closed ]
const _ = ""

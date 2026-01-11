package self_transition

//go:generate go run github.com/donutnomad/gogen gen ./...

// 自我流转测试：= 符号
// (=) 表示保持当前状态不变，常用于"刷新"或"重试"场景
// @StateFlow(name="Connection")
// @Flow: Disconnected => [ Connected ]
// @Flow: Connected    => [ Connected? via Reconnecting ]
// @Flow: Connected    => [ Disconnected ]
const _ = ""

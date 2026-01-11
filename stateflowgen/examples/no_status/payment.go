package no_status

//go:generate go run github.com/donutnomad/gogen gen ./...

// 无 Status 测试：纯 Phase 流转
// 当整个流程没有 Status 时，Stage 是 Phase 的别名
// @StateFlow(name="Payment")
// @Flow: Pending   => [ Processing ]
// @Flow: Processing=> [ Completed, Failed ]
// @Flow: Failed    => [ Processing ]
const _ = ""

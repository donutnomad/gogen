package optional_approval

//go:generate go run github.com/donutnomad/gogen gen ./...

// 可选审批测试：? 符号
// 当 withApproval=true 时进入审批流程，否则直接流转
// @StateFlow(name="Task")
// @Flow: Draft       => [ Submitted ]
// @Flow: Submitted   => [ Approved? via Reviewing ]
// @Flow: Approved    => [ Done ]
const _ = ""

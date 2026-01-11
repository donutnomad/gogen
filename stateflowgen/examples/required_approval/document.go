package required_approval

//go:generate go run github.com/donutnomad/gogen gen ./...

// 必须审批测试：! 符号
// 无论 withApproval 参数如何，都必须进入审批流程
// @StateFlow(name="Document")
// @Flow: Draft       => [ Published! via Reviewing ]
// @Flow: Published   => [ Archived ]
const _ = ""

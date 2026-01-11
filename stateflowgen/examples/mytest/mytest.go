package wildcard

//go:generate go run github.com/donutnomad/gogen gen ./...

// @StateFlow(output=types_state.go)
// @Flow: none => [ creation_initiated? via none ]
// @Flow: creation_initiated => [ creation_submitted ]
// @Flow: creation_submitted => [ deployed, deploy_failed ]
const _ = ""

//type Status uint32
//
//var StatusEnums = struct {
//	Step1Pending  Status // 已提交审批，等待审批
//	Step1Rejected Status // 审批拒绝
//
//	Step2Pending   Status // 审批通过，等待提交部署
//	Step2Committed Status // 已提交部署，等待结果
//
//	Step3Deployed       Status // 部署成功
//	Step3DeployedFailed Status // 部署失败
//}{
//	Step1Pending:  1000,
//	Step1Rejected: 1010,
//
//	Step2Pending:   2000,
//	Step2Committed: 2001,
//
//	Step3Deployed:       3000,
//	Step3DeployedFailed: 3001,
//}

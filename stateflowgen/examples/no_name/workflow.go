package no_name

//go:generate go run github.com/donutnomad/gogen gen ./...

// 测试不填写 name 参数的情况
// @StateFlow
// @Flow: pending => [ approved, rejected ]
// @Flow: approved => [ done ]
// @Flow: rejected => [ pending ]
const workflowStateFlow = ""

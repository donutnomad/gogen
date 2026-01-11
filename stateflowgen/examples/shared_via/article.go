package shared_via

//go:generate go run github.com/donutnomad/gogen gen ./...

// 复用中间态测试
// 多个状态流转使用同一个 via 中间态（Reviewing）
// @StateFlow(name="Article")
// @Flow: Draft      => [ Published! via Reviewing ]
// @Flow: Published  => [ Updated! via Reviewing ]
// @Flow: Updated    => [ Archived! via Reviewing ]
// @Flow: Archived   => [ Deleted ]
const _ = ""

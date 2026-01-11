package else_fallback

//go:generate go run github.com/donutnomad/gogen gen ./...

// else 回退状态测试
// 审批拒绝后不是回退到原状态，而是前往 else 指定的状态
// @StateFlow(name="Release")
// @Flow: Development => [ Testing ]
// @Flow: Testing     => [ Production! via Deploying else Rollback ]
// @Flow: Rollback    => [ Development ]
// @Flow: Production  => [ Archived ]
const _ = ""

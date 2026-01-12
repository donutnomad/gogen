package approvalnode

// Phase 阶段枚举类型
type Phase string

const (
	PhaseNone    Phase = "none"
	PhaseCreated Phase = "created"
	PhaseDeleted Phase = "deleted"
)

// StateColumns 状态列，用于测试跨包嵌入时的类型引用
type StateColumns struct {
	Phase   Phase  `gorm:"column:phase"`
	Pending string `gorm:"column:pending"`
}

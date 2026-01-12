package aliased_embedded

import (
	// 使用别名导入 approvalnode 包，模拟真实场景：
	// 包目录名是 approvalnode，但使用 domain 作为别名
	domain "github.com/donutnomad/gogen/internal/structparse/testdata/aliased_embedded/approvalnode"
)

// NodeSetPO 节点集合持久化对象
// 用于测试：当嵌入的外部包字段展开后，字段类型需要带包前缀，
// 并且 PkgPath 和 PkgAlias 需要正确设置
type NodeSetPO struct {
	ID    uint64              `gorm:"column:id"`
	Name  string              `gorm:"column:name"`
	State domain.StateColumns `gorm:"embedded"` // 嵌入外部包的结构体
}

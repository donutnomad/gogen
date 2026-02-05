//go:generate gotoolkit gen .

package embedded

// TableInfo 用于测试保留名冲突
// 当 embeddedPrefix="table_" + "Name" = "TableName" 时，应该自动重命名为 "TableNameT"
type TableInfo struct {
	Name string `gorm:"column:name"`
}

// Order 订单模型 - 测试保留名冲突
// @Gsql
type Order struct {
	ID    uint64    `gorm:"column:id;primaryKey"`
	Table TableInfo `gorm:"embedded;embeddedPrefix:table_"`
}

func (Order) TableName() string {
	return "orders"
}

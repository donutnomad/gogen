package alias

// 这个文件展示带有包别名的导入场景

import (
	"time"

	// 包别名示例
	gormModel "gorm.io/gorm"
)

// AccountWithAlias 使用别名导入 gorm 的账户模型
// @Pick(name=AccountBasic, fields=`[ID,Name,Balance,CreatedAt]`)
type AccountWithAlias struct {
	gormModel.Model
	Name    string  `json:"name" gorm:"column:name;size:100"`
	Balance float64 `json:"balance" gorm:"column:balance"`
	Secret  string  `json:"-" gorm:"column:secret"`
}

// 另一种常见场景：多个包有相同名称时使用别名

// Transaction 交易记录
// @Omit(name=TransactionPublic, fields=`[InternalRef,ProcessedBy]`)
type Transaction struct {
	ID          uint64    `json:"id" gorm:"primaryKey"`
	AccountID   uint64    `json:"account_id" gorm:"column:account_id;index"`
	Amount      float64   `json:"amount" gorm:"column:amount"`
	Type        string    `json:"type" gorm:"column:type;size:20"`
	InternalRef string    `json:"-" gorm:"column:internal_ref"`
	ProcessedBy string    `json:"-" gorm:"column:processed_by"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

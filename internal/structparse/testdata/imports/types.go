package imports

import (
	"time"

	"github.com/shopspring/decimal"
	orm "gorm.io/gorm"
)

// Order 订单结构体，用于测试导入信息提取和别名处理
type Order struct {
	ID        int64
	Amount    decimal.Decimal
	CreatedAt time.Time
	DeletedAt orm.DeletedAt `gorm:"index"`
}

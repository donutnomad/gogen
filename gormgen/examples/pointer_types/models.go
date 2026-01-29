//go:generate gotoolkit gen .

package pointer_types

import "time"

// Product 产品模型 - 指针类型作为可选字段示例
// 展示 *string, *float64, *int, *bool, *float32, *uint64 类型的 Field 映射
// @Gsql
type Product struct {
	ID          uint64   `gorm:"column:id;primaryKey"`
	Name        string   `gorm:"column:name"`
	Description *string  `gorm:"column:description"`
	Price       *float64 `gorm:"column:price"`
	Stock       *int     `gorm:"column:stock"`
	IsActive    *bool    `gorm:"column:is_active"`
	Weight      *float32 `gorm:"column:weight"`
	CategoryID  *uint64  `gorm:"column:category_id"`
}

func (Product) TableName() string {
	return "products"
}

// Event 事件模型 - 指针时间类型示例
// 展示 *time.Time 与不同 SQL 类型 (datetime, date, time) 的组合
// @Gsql
type Event struct {
	ID        uint64     `gorm:"column:id;primaryKey"`
	Name      string     `gorm:"column:name"`
	StartTime *time.Time `gorm:"column:start_time;type:datetime"`
	EventDate *time.Time `gorm:"column:event_date;type:date"`
	Duration  *time.Time `gorm:"column:duration;type:time"`
}

func (Event) TableName() string {
	return "events"
}

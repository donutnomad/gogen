//go:generate gotoolkit gen .

package indexes

import "time"

// User 用户模型 - 索引示例
// 展示 primaryKey, uniqueIndex, index, autoIncrement 等标志
// @Gsql
type User struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`     // FlagPrimaryKey | FlagAutoIncrement
	Username  string    `gorm:"column:username;uniqueIndex"`            // FlagUniqueIndex
	Email     string    `gorm:"column:email;unique"`                    // FlagUniqueIndex
	Phone     string    `gorm:"column:phone;index"`                     // FlagIndex
	Status    int       `gorm:"column:status;index"`                    // FlagIndex
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;index"`  // FlagIndex
}

func (User) TableName() string {
	return "users"
}

// Product 产品模型 - 复合索引示例
// @Gsql
type Product struct {
	ID         uint64  `gorm:"column:id;primaryKey"`
	SKU        string  `gorm:"column:sku;uniqueIndex:idx_sku_category"` // FlagUniqueIndex
	CategoryID uint64  `gorm:"column:category_id;index"`               // FlagIndex
	Name       string  `gorm:"column:name"`
	Price      float64 `gorm:"column:price"`
}

func (Product) TableName() string {
	return "products"
}

// Order 订单模型 - 多索引示例
// @Gsql
type Order struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	OrderNo     string    `gorm:"column:order_no;uniqueIndex"`
	UserID      uint64    `gorm:"column:user_id;index"`
	Status      int       `gorm:"column:status;index"`
	TotalAmount float64   `gorm:"column:total_amount"`
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime;index"`
}

func (Order) TableName() string {
	return "orders"
}

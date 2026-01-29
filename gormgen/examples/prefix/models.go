//go:generate gotoolkit gen .

package prefix

import "time"

// User 用户模型 - 无前缀示例
// 生成 UserSchemaType 和 UserSchema
// @Gsql
type User struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	Email     string    `gorm:"column:email"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime"`
}

func (User) TableName() string {
	return "users"
}

// Product 产品模型 - 带前缀示例
// 生成 TbProductSchemaType 和 TbProductSchema
// @Gsql(prefix=`Tb`)
type Product struct {
	ID    uint64  `gorm:"column:id;primaryKey"`
	Name  string  `gorm:"column:name"`
	Price float64 `gorm:"column:price"`
}

func (Product) TableName() string {
	return "products"
}

// Order 订单模型 - 带长前缀示例
// 生成 MyAppOrderSchemaType 和 MyAppOrderSchema
// @Gsql(prefix=`MyApp`)
type Order struct {
	ID          uint64    `gorm:"column:id;primaryKey"`
	UserID      uint64    `gorm:"column:user_id"`
	TotalAmount float64   `gorm:"column:total_amount"`
	Status      int       `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime"`
}

func (Order) TableName() string {
	return "orders"
}

// AccountPO 账户持久化模型 - PO 后缀会被移除
// 生成 AccountSchemaType（注意：AccountPO 的 PO 后缀被移除）
// @Gsql(prefix=``)
type AccountPO struct {
	ID   uint64 `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name"`
}

func (AccountPO) TableName() string {
	return "accounts"
}

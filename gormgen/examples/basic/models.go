//go:generate gotoolkit gen .

package basic

// User 用户模型 - 基础类型示例
// 展示 int, uint, string, bool, float 类型的 Field 映射
// @Gsql
type User struct {
	ID       uint64  `gorm:"column:id;primaryKey"`
	Name     string  `gorm:"column:name"`
	Email    string  `gorm:"column:email"`
	Age      int     `gorm:"column:age"`
	Score    float64 `gorm:"column:score"`
	IsActive bool    `gorm:"column:is_active"`
}

func (User) TableName() string {
	return "users"
}

// Product 产品模型 - 更多整数类型
// @Gsql
type Product struct {
	ID       uint64 `gorm:"column:id;primaryKey"`
	Name     string `gorm:"column:name"`
	Price    int64  `gorm:"column:price"`
	Stock    int32  `gorm:"column:stock"`
	Category uint8  `gorm:"column:category"`
	Weight   uint16 `gorm:"column:weight"`
}

func (Product) TableName() string {
	return "products"
}

// Order 订单模型 - 浮点类型
// @Gsql
type Order struct {
	ID          uint64  `gorm:"column:id;primaryKey"`
	UserID      uint64  `gorm:"column:user_id"`
	TotalAmount float64 `gorm:"column:total_amount"`
	Discount    float32 `gorm:"column:discount"`
	Status      int8    `gorm:"column:status"`
}

func (Order) TableName() string {
	return "orders"
}

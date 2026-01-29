//go:generate gotoolkit gen .

package decimal_types

// Price 价格模型 - decimal 类型示例
// 展示 gorm type:decimal 的正确映射到 gsql.DecimalField
// @Gsql
type Price struct {
	ID          uint64  `gorm:"column:id;primaryKey"`
	ProductName string  `gorm:"column:product_name"`
	Amount      float64 `gorm:"column:amount;type:decimal(10,2)"`  // float64 存储
	TaxRate     string  `gorm:"column:tax_rate;type:decimal(5,4)"` // string 存储（高精度）
	Discount    float64 `gorm:"column:discount;type:decimal(3,2)"` // 折扣率
	Total       float64 `gorm:"column:total;type:decimal(12,2)"`   // 总价
}

func (Price) TableName() string {
	return "prices"
}

// Account 账户模型 - 混合 decimal 和其他类型
// @Gsql
type Account struct {
	ID           uint64  `gorm:"column:id;primaryKey"`
	UserID       uint64  `gorm:"column:user_id;index"`
	Balance      float64 `gorm:"column:balance;type:decimal(15,2)"`      // 余额
	CreditLimit  float64 `gorm:"column:credit_limit;type:decimal(15,2)"` // 信用额度
	InterestRate float64 `gorm:"column:interest_rate;type:decimal(5,4)"` // 利率
	Status       int     `gorm:"column:status"`                          // 状态（非 decimal）
}

func (Account) TableName() string {
	return "accounts"
}

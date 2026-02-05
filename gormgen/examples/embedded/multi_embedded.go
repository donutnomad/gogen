//go:generate gotoolkit gen .

package embedded

// Address 地址信息 - 用于多重嵌入测试
type Address struct {
	Country  string `gorm:"column:country"`
	Province string `gorm:"column:province"`
	City     string `gorm:"column:city"`
}

// Company 公司模型 - 多重 embedded 测试
// @Gsql
type Company struct {
	ID          uint64  `gorm:"column:id;primaryKey"`
	Name        string  `gorm:"column:name"`
	HomeAddress Address `gorm:"embedded;embeddedPrefix:home_"`
	WorkAddress Address `gorm:"embedded;embeddedPrefix:work_"`
}

func (Company) TableName() string {
	return "companies"
}

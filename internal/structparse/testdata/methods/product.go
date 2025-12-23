package methods

import "fmt"

// Product 产品结构体，用于测试方法解析
type Product struct {
	ID    int64
	Name  string
	Price float64
}

// GetDisplayName 获取显示名称（值接收器方法）
func (p Product) GetDisplayName() string {
	return fmt.Sprintf("%s ($%.2f)", p.Name, p.Price)
}

// UpdatePrice 更新价格（指针接收器方法）
func (p *Product) UpdatePrice(newPrice float64) {
	p.Price = newPrice
}

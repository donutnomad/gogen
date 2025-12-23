package methods

import "fmt"

// Validate 验证产品数据（跨文件方法）
func (p *Product) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("product name is required")
	}
	if p.Price < 0 {
		return fmt.Errorf("product price cannot be negative")
	}
	return nil
}

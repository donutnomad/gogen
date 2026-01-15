package embedded

import "time"

// BaseModel 基础模型（模拟 gorm.Model）
type BaseModel struct {
	ID        uint64     `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
}

// Timestamps 时间戳字段
type Timestamps struct {
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Employee 员工模型 - 包含嵌入结构体
// Pick/Omit 会先展开嵌入字段，再进行操作
// @Pick(name=EmployeeBasic, fields=`[ID,Name,Email,Department]`)
// @Omit(name=EmployeePublic, fields=`[Salary,SSN,DeletedAt]`)
type Employee struct {
	BaseModel
	Name       string  `json:"name" gorm:"column:name;size:100"`
	Email      string  `json:"email" gorm:"column:email;uniqueIndex"`
	Department string  `json:"department" gorm:"column:department;size:50"`
	Salary     float64 `json:"-" gorm:"column:salary"`
	SSN        string  `json:"-" gorm:"column:ssn"`
}

// Order 订单模型 - 包含多层嵌入
// @Pick(name=OrderSummary, fields=`[ID,OrderNo,Total,Status,CreatedAt]`)
// @Omit(name=OrderList, fields=`[UpdatedAt,InternalNote]`)
type Order struct {
	Timestamps
	ID           uint64  `json:"id" gorm:"primaryKey"`
	OrderNo      string  `json:"order_no" gorm:"column:order_no;uniqueIndex"`
	UserID       uint64  `json:"user_id" gorm:"column:user_id;index"`
	Total        float64 `json:"total" gorm:"column:total"`
	Status       string  `json:"status" gorm:"column:status;size:20"`
	InternalNote string  `json:"-" gorm:"column:internal_note"`
}

// 测试匿名嵌入和命名嵌入的区别

// Contact 联系信息
type Contact struct {
	Phone   string `json:"phone" gorm:"column:phone;size:20"`
	Address string `json:"address" gorm:"column:address;size:255"`
}

// Customer 客户模型 - 包含命名嵌入（不会被展开）
// @Pick(name=CustomerBasic, fields=`[ID,Name,Email]`)
type Customer struct {
	ID      uint64 `json:"id" gorm:"primaryKey"`
	Name    string `json:"name" gorm:"column:name;size:100"`
	Email   string `json:"email" gorm:"column:email;uniqueIndex"`
	Contact        // 匿名嵌入，字段会被展开
}

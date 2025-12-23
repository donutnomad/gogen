package testdata

import "time"

// 示例 1: 基本使用 - 所有字段，指针类型
// @Slice
type User struct {
	ID        int64
	Name      string
	Email     string
	Age       int
	CreatedAt time.Time
}

// 示例 2: 排除字段
// @Slice(exclude=[CreatedAt,UpdatedAt])
type Product struct {
	ID          int64
	Name        string
	Description string
	Price       float64
	Stock       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// 示例 3: 包含指定字段（优先于 exclude）
// @Slice(include=[ID,Name,Email])
type Customer struct {
	ID        int64
	Name      string
	Email     string
	Phone     string
	Address   string
	CreatedAt time.Time
}

// 示例 4: 非指针类型
// @Slice(ptr=false)
type Order struct {
	ID         int64
	OrderNo    string
	TotalPrice float64
	Status     string
}

// 示例 5: 带额外方法
// @Slice(methods=[filter,map,sort])
type Article struct {
	ID        int64
	Title     string
	Content   string
	AuthorID  int64
	Views     int
	CreatedAt time.Time
}

// 示例 6: 完整配置
// @Slice(exclude=[DeletedAt], ptr=true, methods=[filter,map,reduce,sort,groupby])
type Employee struct {
	ID         int64
	Name       string
	Department string
	Salary     float64
	JoinDate   time.Time
	DeletedAt  *time.Time
}

// 示例 7: 嵌入结构体测试
type BaseModel struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// @Slice(methods=[filter,sort])
type Category struct {
	BaseModel
	Name        string
	Description string
	ParentID    *int64
}

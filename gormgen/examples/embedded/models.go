//go:generate gotoolkit gen .

package embedded

import "time"

// BaseModel 基础模型 - 用于嵌入
// 注意：time.Time 不添加 type 标签，GORM MySQL 默认使用 datetime
type BaseModel struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at"` // 无 type 标签，应 fallback 为 DateTimeField
	UpdatedAt time.Time `gorm:"column:updated_at"` // 无 type 标签，应 fallback 为 DateTimeField
}

// Audit 审计信息 - 用于嵌入
// 混合测试：有些字段有 type 标签，有些没有
type Audit struct {
	CreatedBy string    `gorm:"column:created_by"`
	CreatedAt time.Time `gorm:"column:created_at"`              // 无 type 标签
	UpdatedBy string    `gorm:"column:updated_by"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime"` // 有 type 标签
}

// User 用户模型 - 无前缀嵌入示例
// @Gsql
type User struct {
	BaseModel        // 嵌入基础模型（无前缀）
	Name      string `gorm:"column:name"`
	Email     string `gorm:"column:email"`
}

func (User) TableName() string {
	return "users"
}

// Article 文章模型 - 带前缀嵌入示例
// @Gsql
type Article struct {
	ID      uint64 `gorm:"column:id;primaryKey"`
	Title   string `gorm:"column:title"`
	Content string `gorm:"column:content"`
	Audit   Audit  `gorm:"embedded;embeddedPrefix:audit_"` // 嵌入带前缀
}

func (Article) TableName() string {
	return "articles"
}

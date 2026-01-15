package basic

import "time"

// User 用户模型 - 基础 Pick 示例
// @Pick(name=UserBasic, fields=`[ID,Name,Email]`)
// @Pick(name=UserProfile, fields=`[ID,Name,Email,CreatedAt,UpdatedAt]`)
type User struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"column:name;size:100"`
	Email     string    `json:"email" gorm:"column:email;uniqueIndex"`
	Password  string    `json:"-" gorm:"column:password"`
	Salt      string    `json:"-" gorm:"column:salt"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Article 文章模型 - 基础 Omit 示例
// @Omit(name=ArticlePublic, fields=`[DeletedAt,AuthorID]`)
// @Omit(name=ArticlePreview, fields=`[Content,DeletedAt,AuthorID]`)
type Article struct {
	ID        uint64     `json:"id" gorm:"primaryKey"`
	Title     string     `json:"title" gorm:"column:title;size:255"`
	Content   string     `json:"content" gorm:"column:content;type:text"`
	AuthorID  uint64     `json:"author_id" gorm:"column:author_id;index"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
}

// Product 产品模型 - Pick 和 Omit 混合使用
// @Pick(name=ProductSummary, fields=`[ID,Name,Price]`)
// @Omit(name=ProductDetail, fields=`[InternalCode,CostPrice]`)
type Product struct {
	ID           uint64  `json:"id" gorm:"primaryKey"`
	Name         string  `json:"name" gorm:"column:name;size:200"`
	Description  string  `json:"description" gorm:"column:description;type:text"`
	Price        float64 `json:"price" gorm:"column:price"`
	CostPrice    float64 `json:"-" gorm:"column:cost_price"`
	Stock        int     `json:"stock" gorm:"column:stock"`
	InternalCode string  `json:"-" gorm:"column:internal_code"`
}

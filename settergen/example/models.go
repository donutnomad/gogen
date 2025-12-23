package example

import "time"

// 示例 0: 默认模式 - 不生成代码
// @Setter  # 等同于 @Setter(patch="none")，不生成任何代码
type Config struct {
	ID   int64
	Name string
}

// 示例 1: v2 模式 - 使用 automap 生成 ToPatch 方法，默认查找 ToPO
// @Setter(patch="v2")
type User struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt time.Time
}

// ToPO 是默认的 mapper 方法
func (u *User) ToPO() *UserPO {
	return &UserPO{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

type UserPO struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	Email     string    `gorm:"column:email"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// 示例 2: v2 模式，使用自定义 mapper
// @Setter(patch="v2", patch_mapper="Article.ToArticlePO")
type Article struct {
	ID        int64
	Title     string
	Content   string
	AuthorID  int64
	CreatedAt time.Time
}

func (a *Article) ToArticlePO() *ArticlePO {
	return &ArticlePO{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		AuthorID:  a.AuthorID,
		CreatedAt: a.CreatedAt,
	}
}

type ArticlePO struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Title     string    `gorm:"column:title"`
	Content   string    `gorm:"column:content"`
	AuthorID  int64     `gorm:"column:author_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// 示例 3: full 模式 - 生成 ToMap 方法，直接将所有字段转换为 map
// @Setter(patch="full")
type Product struct {
	ID          int64
	Name        string
	Description string
	Price       float64
	Stock       int
	CategoryID  int64
}

// 示例 4: full 模式示例2
// @Setter(patch="full")
type Order struct {
	ID         int64
	OrderNo    string
	UserID     int64
	TotalPrice float64
	Status     string
}

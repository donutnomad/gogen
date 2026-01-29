//go:generate gotoolkit gen .

package nullable_types

import (
	"database/sql"
	"time"
)

// User 用户模型 - sql.Null* 类型示例
// 展示 sql.NullInt32, sql.NullFloat64, sql.NullBool, sql.NullString, sql.NullTime 类型的 Field 映射
// @Gsql
type User struct {
	ID        uint64          `gorm:"column:id;primaryKey"`
	Age       sql.NullInt32   `gorm:"column:age"`
	Score     sql.NullFloat64 `gorm:"column:score"`
	IsVIP     sql.NullBool    `gorm:"column:is_vip"`
	Nickname  sql.NullString  `gorm:"column:nickname"`
	LoginAt   sql.NullTime    `gorm:"column:login_at;type:datetime"`
	BirthDate sql.NullTime    `gorm:"column:birth_date;type:date"`
}

func (User) TableName() string {
	return "users"
}

// Profile 用户档案模型 - 混合可空类型示例
// 展示 sql.Null* 类型与普通类型混用
// @Gsql
type Profile struct {
	ID          uint64         `gorm:"column:id;primaryKey"`
	UserID      uint64         `gorm:"column:user_id;index"`
	Bio         sql.NullString `gorm:"column:bio"`
	Website     sql.NullString `gorm:"column:website"`
	ViewCount   int64          `gorm:"column:view_count"`
	Rating      sql.NullFloat64 `gorm:"column:rating"`
	IsPublic    bool           `gorm:"column:is_public"`
	IsVerified  sql.NullBool   `gorm:"column:is_verified"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   sql.NullTime   `gorm:"column:updated_at;type:datetime"`
	ExpiredAt   sql.NullTime   `gorm:"column:expired_at;type:date"`
}

func (Profile) TableName() string {
	return "profiles"
}

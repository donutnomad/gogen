package testdata

import "time"

// User 用户模型
// @Pick(name=UserBasic, fields=`[ID,Name,Email]`)
// @Omit(name=UserPublic, fields=`[Password,Salt]`)
type User struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"column:name;size:100"`
	Email     string    `json:"email" gorm:"column:email;uniqueIndex"`
	Password  string    `json:"-" gorm:"column:password"`
	Salt      string    `json:"-" gorm:"column:salt"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Product 产品模型
// @Pick(name=ProductSummary, fields=`[ID,Name,Price]`)
type Product struct {
	ID          uint64  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

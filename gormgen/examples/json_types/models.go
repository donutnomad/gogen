//go:generate gotoolkit gen .

package json_types

import "gorm.io/datatypes"

// Settings 设置信息
type Settings struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
}

// Metadata 元数据
type Metadata struct {
	Tags     []string `json:"tags"`
	Version  string   `json:"version"`
	Priority int      `json:"priority"`
}

// User 用户模型 - JSON 类型示例
// JSON 类型通过 GormDataType() 方法识别，映射为 ScalarField[T]（保留原始类型）
// @Gsql
type User struct {
	ID       uint64                       `gorm:"column:id;primaryKey"`
	Name     string                       `gorm:"column:name"`
	Settings datatypes.JSONType[Settings] `gorm:"column:settings;type:json"` // ScalarField[datatypes.JSONType[Settings]]
	Metadata datatypes.JSONType[Metadata] `gorm:"column:metadata;type:json"` // ScalarField[datatypes.JSONType[Metadata]]
}

func (User) TableName() string {
	return "users"
}

// Article 文章模型 - 原始 JSON 类型
// @Gsql
type Article struct {
	ID         uint64         `gorm:"column:id;primaryKey"`
	Title      string         `gorm:"column:title"`
	RawContent datatypes.JSON `gorm:"column:raw_content;type:json"` // ScalarField[datatypes.JSON]
}

func (Article) TableName() string {
	return "articles"
}

// Product 产品模型 - JSON 切片类型
// @Gsql
type Product struct {
	ID     uint64                      `gorm:"column:id;primaryKey"`
	Name   string                      `gorm:"column:name"`
	Tags   datatypes.JSONSlice[string] `gorm:"column:tags;type:json"`   // ScalarField[datatypes.JSONSlice[string]]
	Images datatypes.JSONSlice[string] `gorm:"column:images;type:json"` // ScalarField[datatypes.JSONSlice[string]]
}

func (Product) TableName() string {
	return "products"
}

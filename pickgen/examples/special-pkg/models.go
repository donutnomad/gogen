// Package specialpkg 演示文件夹名与包名不一致的情况
// 文件夹名: special-pkg (带连字符)
// 包名: specialpkg (Go 标识符规范)
package specialpkg

import "time"

// Config 配置模型
// @Pick(name=ConfigBasic, fields=`[ID,Key,Value]`)
// @Omit(name=ConfigPublic, fields=`[SecretValue,InternalFlag]`)
type Config struct {
	ID           uint64    `json:"id" gorm:"primaryKey"`
	Key          string    `json:"key" gorm:"column:key;uniqueIndex"`
	Value        string    `json:"value" gorm:"column:value"`
	SecretValue  string    `json:"-" gorm:"column:secret_value"`
	InternalFlag bool      `json:"-" gorm:"column:internal_flag"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Settings 设置模型
// @Pick(name=SettingsCore, fields=`[ID,Name,Enabled]`)
type Settings struct {
	ID          uint64 `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"column:name;size:100"`
	Description string `json:"description" gorm:"column:description"`
	Enabled     bool   `json:"enabled" gorm:"column:enabled"`
	Priority    int    `json:"priority" gorm:"column:priority"`
}

package remote

// 这个文件展示如何对第三方包的结构体使用 Pick/Omit
// 使用 //go:gen: 注释 + source 参数指定远程类型

// ===== GORM Model 示例 =====
// gorm.Model 包含: ID, CreatedAt, UpdatedAt, DeletedAt

// 从 gorm.Model 提取基础字段
//go:gen: @Pick(name=GormBasic, source=`gorm.io/gorm.Model`, fields=`[ID,CreatedAt,UpdatedAt]`)

// 从 gorm.Model 排除软删除字段
//go:gen: @Omit(name=GormNoDelete, source=`gorm.io/gorm.Model`, fields=`[DeletedAt]`)

// ===== GORM datatypes 示例 =====
// datatypes.JSON 等类型

// ===== Gin Context 相关示例 =====
// 从 gin.Context 提取常用字段（仅作示例，实际字段需根据源码确定）

// ===== 本地使用远程类型的结构体 =====

import (
	"gorm.io/gorm"
)

// UserWithGorm 使用 gorm.Model 的用户模型
// @Pick(name=UserGormBasic, fields=`[ID,Name,Email,CreatedAt]`)
type UserWithGorm struct {
	gorm.Model
	Name     string `json:"name" gorm:"column:name;size:100"`
	Email    string `json:"email" gorm:"column:email;uniqueIndex"`
	Password string `json:"-" gorm:"column:password"`
}

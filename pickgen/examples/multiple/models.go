package multiple

import "time"

// Account 账户模型 - 演示同一结构体上的多重注解
// 可以同时使用多个 @Pick 和 @Omit 生成不同的派生结构体
// @Pick(name=AccountID, fields=`[ID]`)
// @Pick(name=AccountBasic, fields=`[ID,Name,Email]`)
// @Pick(name=AccountProfile, fields=`[ID,Name,Email,Phone,Address,CreatedAt]`)
// @Pick(name=AccountFull, fields=`[ID,Name,Email,Phone,Address,Balance,Status,CreatedAt,UpdatedAt]`)
// @Omit(name=AccountPublic, fields=`[Password,Salt,InternalNote]`)
// @Omit(name=AccountSafe, fields=`[Password,Salt,InternalNote,Balance]`)
type Account struct {
	ID           uint64    `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"column:name;size:100"`
	Email        string    `json:"email" gorm:"column:email;uniqueIndex"`
	Phone        string    `json:"phone" gorm:"column:phone;size:20"`
	Address      string    `json:"address" gorm:"column:address;size:255"`
	Password     string    `json:"-" gorm:"column:password"`
	Salt         string    `json:"-" gorm:"column:salt"`
	Balance      float64   `json:"balance" gorm:"column:balance"`
	Status       string    `json:"status" gorm:"column:status;size:20"`
	InternalNote string    `json:"-" gorm:"column:internal_note"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Permission 权限模型 - 另一个多重注解示例
// @Pick(name=PermissionBasic, fields=`[ID,Name,Code]`)
// @Pick(name=PermissionDetail, fields=`[ID,Name,Code,Description,Module]`)
// @Omit(name=PermissionPublic, fields=`[InternalCode]`)
type Permission struct {
	ID           uint64    `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"column:name;size:100"`
	Code         string    `json:"code" gorm:"column:code;uniqueIndex"`
	Description  string    `json:"description" gorm:"column:description"`
	Module       string    `json:"module" gorm:"column:module;size:50"`
	InternalCode string    `json:"-" gorm:"column:internal_code"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
}

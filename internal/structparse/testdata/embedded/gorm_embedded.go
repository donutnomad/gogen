package embedded

import "time"

// Account 账户信息，用于测试 gorm:embedded
type Account struct {
	AccountID   string    `gorm:"column:account_id"`
	AccountName string    `gorm:"column:account_name"`
	Balance     float64   `gorm:"column:balance"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

// UserWithAccount 带账户信息的用户，测试 gorm embedded 和 embeddedPrefix
type UserWithAccount struct {
	ID      int64   `gorm:"primaryKey"`
	Name    string  `gorm:"column:name"`
	Account Account `gorm:"embedded;embeddedPrefix:account_"`
}

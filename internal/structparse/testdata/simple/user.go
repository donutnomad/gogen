package simple

import "time"

// User 简单的用户结构体，用于测试基本字段解析
type User struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

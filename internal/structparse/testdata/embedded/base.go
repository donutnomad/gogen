package embedded

import "time"

// BaseModel 基础模型，用于测试匿名嵌入字段
type BaseModel struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

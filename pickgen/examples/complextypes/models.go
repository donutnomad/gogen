package complextypes

import (
	"encoding/json"
	"time"
)

// JSONField 自定义 JSON 字段类型
type JSONField json.RawMessage

// Status 状态枚举
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusPending  Status = "pending"
)

// Metadata 元数据结构
type Metadata struct {
	Version   string            `json:"version"`
	Tags      []string          `json:"tags"`
	Labels    map[string]string `json:"labels"`
	CreatedBy string            `json:"created_by"`
}

// Document 文档模型 - 包含复杂字段类型
// @Pick(name=DocumentBasic, fields=`[ID,Title,Status,CreatedAt]`)
// @Omit(name=DocumentPublic, fields=`[InternalData,AdminNotes]`)
type Document struct {
	ID           uint64         `json:"id" gorm:"primaryKey"`
	Title        string         `json:"title" gorm:"column:title;size:255"`
	Content      string         `json:"content" gorm:"column:content;type:text"`
	Status       Status         `json:"status" gorm:"column:status;size:20"`
	Metadata     Metadata       `json:"metadata" gorm:"column:metadata;type:json"`
	Tags         []string       `json:"tags" gorm:"column:tags;type:json"`
	Extra        map[string]any `json:"extra" gorm:"column:extra;type:json"`
	InternalData JSONField      `json:"-" gorm:"column:internal_data"`
	AdminNotes   string         `json:"-" gorm:"column:admin_notes"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

// Event 事件模型 - 包含指针和接口类型
// @Pick(name=EventBasic, fields=`[ID,Type,Timestamp]`)
// @Omit(name=EventPublic, fields=`[InternalID,Handler]`)
type Event struct {
	ID          uint64     `json:"id" gorm:"primaryKey"`
	Type        string     `json:"type" gorm:"column:type;size:50"`
	Payload     any        `json:"payload" gorm:"column:payload;type:json"`
	Timestamp   time.Time  `json:"timestamp" gorm:"column:timestamp"`
	ProcessedAt *time.Time `json:"processed_at" gorm:"column:processed_at"`
	InternalID  string     `json:"-" gorm:"column:internal_id"`
	Handler     func()     `json:"-" gorm:"-"`
}

// GenericContainer 泛型容器示例（Go 1.18+）
// 注意：当前 Pick/Omit 不支持泛型类型，此示例仅作展示
type GenericContainer[T any] struct {
	ID      uint64 `json:"id" gorm:"primaryKey"`
	Name    string `json:"name" gorm:"column:name;size:100"`
	Items   []T    `json:"items" gorm:"column:items;type:json"`
	Count   int    `json:"count" gorm:"column:count"`
	Private bool   `json:"-" gorm:"column:private"`
}

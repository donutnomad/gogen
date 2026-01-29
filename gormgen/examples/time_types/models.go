//go:generate gotoolkit gen .

package time_types

import "time"

// Event 事件模型 - 时间类型示例
// 展示 datetime, date, time 的细分 Field 类型映射
// @Gsql
type Event struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	StartTime time.Time `gorm:"column:start_time;type:datetime(3)"` // DateTimeField
	EndTime   time.Time `gorm:"column:end_time;type:timestamp"`     // DateTimeField
	EventDate time.Time `gorm:"column:event_date;type:date"`        // DateField
	EventTime time.Time `gorm:"column:event_time;type:time"`        // TimeField
	CreatedAt time.Time `gorm:"column:created_at"`                  // DateTimeField (默认)
}

func (Event) TableName() string {
	return "events"
}

// Schedule 日程模型 - 纯日期类型
// @Gsql
type Schedule struct {
	ID          uint64    `gorm:"column:id;primaryKey"`
	Title       string    `gorm:"column:title"`
	ScheduleDay time.Time `gorm:"column:schedule_day;type:date"` // DateField
	StartHour   time.Time `gorm:"column:start_hour;type:time"`   // TimeField
	EndHour     time.Time `gorm:"column:end_hour;type:time"`     // TimeField
}

func (Schedule) TableName() string {
	return "schedules"
}

// Timestamp 时间戳模型 - 各种 datetime 变体
// @Gsql
type Timestamp struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime"`    // DateTimeField
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime(6)"` // DateTimeField
	DeletedAt time.Time `gorm:"column:deleted_at;type:timestamp"`   // DateTimeField
	ExpiredAt time.Time `gorm:"column:expired_at"`                  // DateTimeField (无标签 fallback)
}

func (Timestamp) TableName() string {
	return "timestamps"
}

// DefaultTime 默认时间模型 - 测试无 type 标签的 fallback 行为
// GORM MySQL 默认对 time.Time 使用 datetime 类型
// @Gsql
type DefaultTime struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at"` // 无 type 标签，fallback 为 DateTimeField
	UpdatedAt time.Time `gorm:"column:updated_at"` // 无 type 标签，fallback 为 DateTimeField
	LoginTime time.Time `gorm:"column:login_time"` // 无 type 标签，fallback 为 DateTimeField
}

func (DefaultTime) TableName() string {
	return "default_times"
}

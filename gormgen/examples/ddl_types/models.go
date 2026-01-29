//go:generate gotoolkit gen .

package ddl_types

import "time"

// Event 事件模型 - 通过 MysqlCreateTable 方法定义 SQL 类型
// 当 gorm 标签没有 type 时，会从 MysqlCreateTable() 方法的 DDL 中解析
// @Gsql
type Event struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	StartTime time.Time `gorm:"column:start_time"` // 从 DDL 解析为 datetime
	EventDate time.Time `gorm:"column:event_date"` // 从 DDL 解析为 date
	EventTime time.Time `gorm:"column:event_time"` // 从 DDL 解析为 time
	CreatedAt time.Time `gorm:"column:created_at"` // 从 DDL 解析为 datetime
}

func (Event) TableName() string {
	return "events"
}

// MysqlCreateTable 返回 MySQL 建表语句
// gormgen 会解析此方法来获取列的 SQL 类型
func (Event) MysqlCreateTable() string {
	return `
CREATE TABLE events (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    start_time DATETIME(3) NOT NULL,
    event_date DATE NOT NULL,
    event_time TIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`
}

// Schedule 日程模型 - 混合使用 gorm 标签和 DDL
// @Gsql
type Schedule struct {
	ID          uint64    `gorm:"column:id;primaryKey"`
	Title       string    `gorm:"column:title"`
	ScheduleDay time.Time `gorm:"column:schedule_day;type:date"` // 优先使用 gorm 标签
	StartHour   time.Time `gorm:"column:start_hour"`             // 从 DDL 解析
	EndHour     time.Time `gorm:"column:end_hour"`               // 从 DDL 解析
}

func (Schedule) TableName() string {
	return "schedules"
}

func (Schedule) MysqlCreateTable() string {
	return `
CREATE TABLE schedules (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    title VARCHAR(200) NOT NULL,
    schedule_day DATE NOT NULL,
    start_hour TIME NOT NULL,
    end_hour TIME NOT NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`
}

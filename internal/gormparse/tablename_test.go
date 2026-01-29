package gormparse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInferTableName(t *testing.T) {
	tests := []struct {
		name        string
		structName  string
		fileContent string
		expected    string
	}{
		{
			name:       "默认规则 - User",
			structName: "User",
			fileContent: `package test

type User struct {
	ID   int
	Name string
}`,
			expected: "users",
		},
		{
			name:       "默认规则 - Product",
			structName: "Product",
			fileContent: `package test

type Product struct {
	ID    int
	Title string
}`,
			expected: "products",
		},
		{
			name:       "默认规则 - OrderItem",
			structName: "OrderItem",
			fileContent: `package test

type OrderItem struct {
	ID int
}`,
			expected: "order_items",
		},
		{
			name:       "有 TableName 方法 - 指针接收者",
			structName: "User",
			fileContent: `package test

type User struct {
	ID   int
	Name string
}

func (u *User) TableName() string {
	return "custom_users"
}`,
			expected: "custom_users",
		},
		{
			name:       "有 TableName 方法 - 值接收者",
			structName: "Product",
			fileContent: `package test

type Product struct {
	ID    int
	Title string
}

func (p Product) TableName() string {
	return "my_products"
}`,
			expected: "my_products",
		},
		{
			name:       "有 TableName 方法 - 包含前缀",
			structName: "Order",
			fileContent: `package test

type Order struct {
	ID int
}

func (o *Order) TableName() string {
	return "t_orders"
}`,
			expected: "t_orders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时文件
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")

			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// 测试 InferTableName
			tableName, err := InferTableName(tmpFile, tt.structName)
			if err != nil {
				t.Fatalf("InferTableName() error = %v", err)
			}

			if tableName != tt.expected {
				t.Errorf("InferTableName() = %v, want %v", tableName, tt.expected)
			}
		})
	}
}

func TestExtractTableNameFromMethod(t *testing.T) {
	tests := []struct {
		name        string
		structName  string
		fileContent string
		expected    string
		shouldFind  bool
	}{
		{
			name:       "找到 TableName 方法",
			structName: "User",
			fileContent: `package test

type User struct {
	ID int
}

func (u *User) TableName() string {
	return "users_table"
}`,
			expected:   "users_table",
			shouldFind: true,
		},
		{
			name:       "没有 TableName 方法",
			structName: "User",
			fileContent: `package test

type User struct {
	ID int
}`,
			expected:   "",
			shouldFind: false,
		},
		{
			name:       "有其他方法但没有 TableName",
			structName: "User",
			fileContent: `package test

type User struct {
	ID int
}

func (u *User) GetName() string {
	return "test"
}`,
			expected:   "",
			shouldFind: false,
		},
		{
			name:       "多个结构体 - 只匹配指定的",
			structName: "User",
			fileContent: `package test

type User struct {
	ID int
}

func (u *User) TableName() string {
	return "users"
}

type Product struct {
	ID int
}

func (p *Product) TableName() string {
	return "products"
}`,
			expected:   "users",
			shouldFind: true,
		},
		{
			name:       "值接收者",
			structName: "Order",
			fileContent: `package test

type Order struct {
	ID int
}

func (o Order) TableName() string {
	return "orders"
}`,
			expected:   "orders",
			shouldFind: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时文件
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")

			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// 测试 ExtractTableNameFromMethod
			tableName, err := ExtractTableNameFromMethod(tmpFile, tt.structName)
			if err != nil {
				t.Fatalf("ExtractTableNameFromMethod() error = %v", err)
			}

			if tt.shouldFind {
				if tableName != tt.expected {
					t.Errorf("ExtractTableNameFromMethod() = %v, want %v", tableName, tt.expected)
				}
			} else {
				if tableName != "" {
					t.Errorf("ExtractTableNameFromMethod() should return empty string, got %v", tableName)
				}
			}
		})
	}
}

func TestInferTableName_DefaultNaming(t *testing.T) {
	tests := []struct {
		structName string
		expected   string
	}{
		{"User", "users"},
		{"Product", "products"},
		{"Order", "orders"},
		{"OrderItem", "order_items"},
		{"UserProfile", "user_profiles"},
		{"HTTPRequest", "http_requests"},
		{"XMLParser", "xml_parsers"},
		{"APIKey", "api_keys"},
	}

	for _, tt := range tests {
		t.Run(tt.structName, func(t *testing.T) {
			// 创建一个没有 TableName 方法的临时文件
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")

			content := "package test\n\ntype " + tt.structName + " struct { ID int }"
			err := os.WriteFile(tmpFile, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			tableName, err := InferTableName(tmpFile, tt.structName)
			if err != nil {
				t.Fatalf("InferTableName() error = %v", err)
			}

			if tableName != tt.expected {
				t.Errorf("InferTableName(%s) = %v, want %v", tt.structName, tableName, tt.expected)
			}
		})
	}
}

func TestExtractSQLTypeFromDDL(t *testing.T) {
	tests := []struct {
		name        string
		structName  string
		fileContent string
		expected    map[string]string
		shouldFind  bool
	}{
		{
			name:       "基本 DDL 解析",
			structName: "Event",
			fileContent: `package test

type Event struct {
	ID        uint64
	Name      string
	StartTime time.Time
	EventDate time.Time
}

func (Event) MysqlCreateTable() string {
	return ` + "`" + `
CREATE TABLE events (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    start_time DATETIME(3) NOT NULL,
    event_date DATE NOT NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB;
` + "`" + `
}`,
			expected: map[string]string{
				"id":         "bigint",
				"name":       "varchar",
				"start_time": "datetime",
				"event_date": "date",
			},
			shouldFind: true,
		},
		{
			name:       "包含 TIME 类型",
			structName: "Schedule",
			fileContent: `package test

type Schedule struct {
	ID        uint64
	StartHour time.Time
	EndHour   time.Time
}

func (Schedule) MysqlCreateTable() string {
	return ` + "`" + `
CREATE TABLE schedules (
    id BIGINT UNSIGNED NOT NULL,
    start_hour TIME NOT NULL,
    end_hour TIME NOT NULL,
    PRIMARY KEY (id)
);
` + "`" + `
}`,
			expected: map[string]string{
				"id":         "bigint",
				"start_hour": "time",
				"end_hour":   "time",
			},
			shouldFind: true,
		},
		{
			name:       "没有 MysqlCreateTable 方法",
			structName: "User",
			fileContent: `package test

type User struct {
	ID   int
	Name string
}`,
			expected:   nil,
			shouldFind: false,
		},
		{
			name:       "双引号字符串 - 单行",
			structName: "Product",
			fileContent: `package test

type Product struct {
	ID    uint64
	Name  string
	Price float64
}

func (Product) MysqlCreateTable() string {
	return "id BIGINT NOT NULL, name VARCHAR(100), price DECIMAL(10,2)"
}`,
			expected: map[string]string{
				"id":    "bigint",
				"name":  "varchar",
				"price": "decimal",
			},
			shouldFind: true,
		},
		{
			name:       "多种数据类型",
			structName: "AllTypes",
			fileContent: `package test

type AllTypes struct {
	ID          uint64
	SmallNum    int16
	BigNum      int64
	FloatNum    float32
	DoubleNum   float64
	Text        string
	LongText    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	BirthDate   time.Time
	CheckInTime time.Time
}

func (AllTypes) MysqlCreateTable() string {
	return ` + "`" + `
CREATE TABLE all_types (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    small_num SMALLINT,
    big_num BIGINT,
    float_num FLOAT,
    double_num DOUBLE,
    text VARCHAR(255),
    long_text TEXT,
    created_at DATETIME(6),
    updated_at TIMESTAMP,
    birth_date DATE,
    check_in_time TIME,
    PRIMARY KEY (id),
    INDEX idx_created (created_at)
);
` + "`" + `
}`,
			expected: map[string]string{
				"id":            "bigint",
				"small_num":     "smallint",
				"big_num":       "bigint",
				"float_num":     "float",
				"double_num":    "double",
				"text":          "varchar",
				"long_text":     "text",
				"created_at":    "datetime",
				"updated_at":    "timestamp",
				"birth_date":    "date",
				"check_in_time": "time",
			},
			shouldFind: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")

			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			result, err := ExtractSQLTypeFromDDL(tmpFile, tt.structName)
			if err != nil {
				t.Fatalf("ExtractSQLTypeFromDDL() error = %v", err)
			}

			if tt.shouldFind {
				if result == nil {
					t.Fatalf("ExtractSQLTypeFromDDL() returned nil, expected %v", tt.expected)
				}
				for col, expectedType := range tt.expected {
					if gotType, exists := result[col]; !exists {
						t.Errorf("Column %q not found in result", col)
					} else if gotType != expectedType {
						t.Errorf("Column %q: got %q, want %q", col, gotType, expectedType)
					}
				}
			} else {
				if result != nil && len(result) > 0 {
					t.Errorf("ExtractSQLTypeFromDDL() should return nil or empty map, got %v", result)
				}
			}
		})
	}
}

func TestParseDDLColumnTypes(t *testing.T) {
	tests := []struct {
		name     string
		ddl      string
		expected map[string]string
	}{
		{
			name: "基本 CREATE TABLE",
			ddl: `CREATE TABLE users (
    id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(255),
    email VARCHAR(100),
    PRIMARY KEY (id)
)`,
			expected: map[string]string{
				"id":    "bigint",
				"name":  "varchar",
				"email": "varchar",
			},
		},
		{
			name: "带反引号的列名",
			ddl: "CREATE TABLE `users` (\n    `id` BIGINT NOT NULL,\n    `user_name` VARCHAR(255)\n)",
			expected: map[string]string{
				"id":        "bigint",
				"user_name": "varchar",
			},
		},
		{
			name: "时间类型",
			ddl: `CREATE TABLE events (
    id BIGINT,
    start_time DATETIME(3),
    end_time TIMESTAMP,
    event_date DATE,
    check_time TIME
)`,
			expected: map[string]string{
				"id":         "bigint",
				"start_time": "datetime",
				"end_time":   "timestamp",
				"event_date": "date",
				"check_time": "time",
			},
		},
		{
			name: "跳过 INDEX 和 KEY 行",
			ddl: `CREATE TABLE products (
    id BIGINT,
    name VARCHAR(100),
    INDEX idx_name (name),
    UNIQUE KEY uk_id (id),
    PRIMARY KEY (id)
)`,
			expected: map[string]string{
				"id":   "bigint",
				"name": "varchar",
			},
		},
		{
			name: "包含 INDEX 行 - 和 created_at",
			ddl: `CREATE TABLE all_types (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(6),
    PRIMARY KEY (id),
    INDEX idx_created (created_at)
)`,
			expected: map[string]string{
				"id":         "bigint",
				"created_at": "datetime",
			},
		},
		{
			name:     "空 DDL",
			ddl:      "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDDLColumnTypes(tt.ddl)

			if len(result) != len(tt.expected) {
				t.Errorf("ParseDDLColumnTypes() returned %d columns, want %d", len(result), len(tt.expected))
			}

			for col, expectedType := range tt.expected {
				if gotType, exists := result[col]; !exists {
					t.Errorf("Column %q not found in result", col)
				} else if gotType != expectedType {
					t.Errorf("Column %q: got %q, want %q", col, gotType, expectedType)
				}
			}
		})
	}
}

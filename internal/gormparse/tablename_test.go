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

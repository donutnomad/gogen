package gormparse

import "testing"

// TestExtractColumnName 测试列名提取函数
func TestExtractColumnName(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		fieldTag  string
		expected  string
	}{
		{
			name:      "无标签",
			fieldName: "UserName",
			fieldTag:  "",
			expected:  "user_name",
		},
		{
			name:      "有column标签",
			fieldName: "UserName",
			fieldTag:  `gorm:"column:custom_name"`,
			expected:  "custom_name",
		},
		{
			name:      "有其他标签但无column",
			fieldName: "CreatedAt",
			fieldTag:  `gorm:"type:datetime"`,
			expected:  "created_at",
		},
		{
			name:      "多个标签包含column",
			fieldName: "UpdatedAt",
			fieldTag:  `gorm:"column:updated_time;type:datetime"`,
			expected:  "updated_time",
		},
		{
			name:      "ID字段",
			fieldName: "ID",
			fieldTag:  "",
			expected:  "id",
		},
		{
			name:      "EmployeeID字段",
			fieldName: "EmployeeID",
			fieldTag:  "",
			expected:  "employee_id",
		},
		{
			name:      "HTTPStatus字段",
			fieldName: "HTTPStatus",
			fieldTag:  "",
			expected:  "http_status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractColumnName(tt.fieldName, tt.fieldTag)
			if result != tt.expected {
				t.Errorf("ExtractColumnName(%q, %q) = %q, want %q",
					tt.fieldName, tt.fieldTag, result, tt.expected)
			}
		})
	}
}

// TestExtractColumnNameWithPrefix 测试带前缀的列名提取
func TestExtractColumnNameWithPrefix(t *testing.T) {
	tests := []struct {
		name           string
		fieldName      string
		fieldTag       string
		embeddedPrefix string
		expected       string
	}{
		{
			name:           "无前缀",
			fieldName:      "Name",
			fieldTag:       "",
			embeddedPrefix: "",
			expected:       "name",
		},
		{
			name:           "有前缀",
			fieldName:      "Name",
			fieldTag:       "",
			embeddedPrefix: "author_",
			expected:       "author_name",
		},
		{
			name:           "有前缀和column标签",
			fieldName:      "Name",
			fieldTag:       `gorm:"column:custom_name"`,
			embeddedPrefix: "author_",
			expected:       "author_custom_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractColumnNameWithPrefix(tt.fieldName, tt.fieldTag, tt.embeddedPrefix)
			if result != tt.expected {
				t.Errorf("ExtractColumnNameWithPrefix(%q, %q, %q) = %q, want %q",
					tt.fieldName, tt.fieldTag, tt.embeddedPrefix, result, tt.expected)
			}
		})
	}
}

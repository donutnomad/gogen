package gormparse

import "testing"

// TestExtractSQLType 测试从 gorm 标签提取 SQL 类型
func TestExtractSQLType(t *testing.T) {
	tests := []struct {
		name     string
		fieldTag string
		expected string
	}{
		// 正常情况
		{
			name:     "datetime 类型",
			fieldTag: `gorm:"type:datetime"`,
			expected: "datetime",
		},
		{
			name:     "datetime(3) 带精度",
			fieldTag: `gorm:"type:datetime(3)"`,
			expected: "datetime",
		},
		{
			name:     "varchar(255) 类型",
			fieldTag: `gorm:"type:varchar(255)"`,
			expected: "varchar",
		},
		{
			name:     "date 类型",
			fieldTag: `gorm:"type:date"`,
			expected: "date",
		},
		{
			name:     "time 类型",
			fieldTag: `gorm:"type:time"`,
			expected: "time",
		},
		{
			name:     "timestamp 类型",
			fieldTag: `gorm:"type:timestamp"`,
			expected: "timestamp",
		},
		{
			name:     "bigint 类型",
			fieldTag: `gorm:"type:bigint"`,
			expected: "bigint",
		},
		{
			name:     "text 类型",
			fieldTag: `gorm:"type:text"`,
			expected: "text",
		},
		{
			name:     "json 类型",
			fieldTag: `gorm:"type:json"`,
			expected: "json",
		},
		// 大小写测试
		{
			name:     "DATETIME 大写",
			fieldTag: `gorm:"type:DATETIME"`,
			expected: "datetime",
		},
		{
			name:     "DateTime(3) 混合大小写",
			fieldTag: `gorm:"type:DateTime(3)"`,
			expected: "datetime",
		},
		// 组合标签
		{
			name:     "column在前type在后",
			fieldTag: `gorm:"column:name;type:varchar(100)"`,
			expected: "varchar",
		},
		{
			name:     "type在前column在后",
			fieldTag: `gorm:"type:datetime;column:created_at"`,
			expected: "datetime",
		},
		// 空值边界
		{
			name:     "空标签",
			fieldTag: "",
			expected: "",
		},
		{
			name:     "无gorm标签",
			fieldTag: `json:"name"`,
			expected: "",
		},
		{
			name:     "gorm无type",
			fieldTag: `gorm:"column:name"`,
			expected: "",
		},
		{
			name:     "type空值",
			fieldTag: `gorm:"type:"`,
			expected: "",
		},
		// 特殊格式
		{
			name:     "decimal(10,2) 多参数",
			fieldTag: `gorm:"type:decimal(10,2)"`,
			expected: "decimal",
		},
		{
			name:     "varbinary(32) 类型",
			fieldTag: `gorm:"type:varbinary(32)"`,
			expected: "varbinary",
		},
		{
			name:     "多标签组合复杂场景",
			fieldTag: `gorm:"column:amount;type:decimal(10,2);not null"`,
			expected: "decimal",
		},
		{
			name:     "带primaryKey标签",
			fieldTag: `gorm:"type:bigint;primaryKey;autoIncrement"`,
			expected: "bigint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSQLType(tt.fieldTag)
			if result != tt.expected {
				t.Errorf("ExtractSQLType(%q) = %q, want %q",
					tt.fieldTag, result, tt.expected)
			}
		})
	}
}

// TestInferGormDataType 测试推断 GORM 数据类型
func TestInferGormDataType(t *testing.T) {
	tests := []struct {
		name      string
		fieldType string
		fieldTag  string
		expected  string
	}{
		// datatypes 包类型
		{
			name:      "datatypes.JSON",
			fieldType: "datatypes.JSON",
			fieldTag:  "",
			expected:  "json",
		},
		{
			name:      "*datatypes.JSON 指针类型",
			fieldType: "*datatypes.JSON",
			fieldTag:  "",
			expected:  "json",
		},
		{
			name:      "datatypes.JSONType[T] 泛型类型",
			fieldType: "datatypes.JSONType[MyStruct]",
			fieldTag:  "",
			expected:  "json",
		},
		{
			name:      "datatypes.JSONSlice[T] 泛型切片",
			fieldType: "datatypes.JSONSlice[string]",
			fieldTag:  "",
			expected:  "json",
		},
		{
			name:      "datatypes.JSONMap[K,V] 泛型Map",
			fieldType: "datatypes.JSONMap[string,any]",
			fieldTag:  "",
			expected:  "json",
		},
		{
			name:      "*datatypes.JSONType[T] 指针泛型",
			fieldType: "*datatypes.JSONType[Config]",
			fieldTag:  "",
			expected:  "json",
		},
		{
			name:      "datatypes.JSONSlice[int] 基础类型切片",
			fieldType: "datatypes.JSONSlice[int]",
			fieldTag:  "",
			expected:  "json",
		},
		// serializer 标签
		{
			name:      "serializer:json 标签",
			fieldType: "[]string",
			fieldTag:  `gorm:"serializer:json"`,
			expected:  "json",
		},
		{
			name:      "serializer:gob 标签",
			fieldType: "[]string",
			fieldTag:  `gorm:"serializer:gob"`,
			expected:  "",
		},
		{
			name:      "serializer:json 组合标签",
			fieldType: "map[string]any",
			fieldTag:  `gorm:"column:data;serializer:json"`,
			expected:  "json",
		},
		// 非 JSON 类型
		{
			name:      "string 类型",
			fieldType: "string",
			fieldTag:  "",
			expected:  "",
		},
		{
			name:      "int 类型",
			fieldType: "int",
			fieldTag:  "",
			expected:  "",
		},
		{
			name:      "time.Time 类型",
			fieldType: "time.Time",
			fieldTag:  "",
			expected:  "",
		},
		{
			name:      "float64 类型",
			fieldType: "float64",
			fieldTag:  "",
			expected:  "",
		},
		{
			name:      "bool 类型",
			fieldType: "bool",
			fieldTag:  "",
			expected:  "",
		},
		{
			name:      "[]byte 切片类型",
			fieldType: "[]byte",
			fieldTag:  "",
			expected:  "",
		},
		// 边界情况
		{
			name:      "空类型",
			fieldType: "",
			fieldTag:  "",
			expected:  "",
		},
		{
			name:      "仅datatypes前缀",
			fieldType: "datatypes.Date",
			fieldTag:  "",
			expected:  "",
		},
		{
			name:      "非datatypes包的类似类型",
			fieldType: "mypkg.JSON",
			fieldTag:  "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferGormDataType(tt.fieldType, tt.fieldTag)
			if result != tt.expected {
				t.Errorf("InferGormDataType(%q, %q) = %q, want %q",
					tt.fieldType, tt.fieldTag, result, tt.expected)
			}
		})
	}
}

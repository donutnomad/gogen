package gormgen

import (
	"testing"

	"github.com/donutnomad/gogen/internal/gormparse"
)

// TestMapFieldTypeInfo 测试类型映射核心函数
func TestMapFieldTypeInfo(t *testing.T) {
	tests := []struct {
		name              string
		field             gormparse.GormFieldInfo
		expectedFieldType string
		expectedConstr    string
		expectedCategory  string
	}{
		// 整数类型
		{
			name:              "int64类型",
			field:             gormparse.GormFieldInfo{Type: "int64"},
			expectedFieldType: "gsql.IntField[int64]",
			expectedConstr:    "gsql.IntFieldOf[int64]",
			expectedCategory:  "int",
		},
		{
			name:              "uint64类型",
			field:             gormparse.GormFieldInfo{Type: "uint64"},
			expectedFieldType: "gsql.IntField[uint64]",
			expectedConstr:    "gsql.IntFieldOf[uint64]",
			expectedCategory:  "int",
		},
		{
			name:              "指针int类型",
			field:             gormparse.GormFieldInfo{Type: "*int"},
			expectedFieldType: "gsql.IntField[*int]",
			expectedConstr:    "gsql.IntFieldOf[*int]",
			expectedCategory:  "int",
		},
		{
			name:              "sql.NullInt64类型",
			field:             gormparse.GormFieldInfo{Type: "sql.NullInt64"},
			expectedFieldType: "gsql.IntField[sql.NullInt64]",
			expectedConstr:    "gsql.IntFieldOf[sql.NullInt64]",
			expectedCategory:  "int",
		},
		// 布尔类型 -> IntField
		{
			name:              "bool类型映射到IntField",
			field:             gormparse.GormFieldInfo{Type: "bool"},
			expectedFieldType: "gsql.IntField[bool]",
			expectedConstr:    "gsql.IntFieldOf[bool]",
			expectedCategory:  "int",
		},
		{
			name:              "sql.NullBool类型映射到IntField",
			field:             gormparse.GormFieldInfo{Type: "sql.NullBool"},
			expectedFieldType: "gsql.IntField[sql.NullBool]",
			expectedConstr:    "gsql.IntFieldOf[sql.NullBool]",
			expectedCategory:  "int",
		},
		// 浮点类型
		{
			name:              "float32类型",
			field:             gormparse.GormFieldInfo{Type: "float32"},
			expectedFieldType: "gsql.FloatField[float32]",
			expectedConstr:    "gsql.FloatFieldOf[float32]",
			expectedCategory:  "float",
		},
		{
			name:              "float64类型",
			field:             gormparse.GormFieldInfo{Type: "float64"},
			expectedFieldType: "gsql.FloatField[float64]",
			expectedConstr:    "gsql.FloatFieldOf[float64]",
			expectedCategory:  "float",
		},
		{
			name:              "sql.NullFloat64类型",
			field:             gormparse.GormFieldInfo{Type: "sql.NullFloat64"},
			expectedFieldType: "gsql.FloatField[sql.NullFloat64]",
			expectedConstr:    "gsql.FloatFieldOf[sql.NullFloat64]",
			expectedCategory:  "float",
		},
		// 字符串类型
		{
			name:              "string类型",
			field:             gormparse.GormFieldInfo{Type: "string"},
			expectedFieldType: "gsql.StringField[string]",
			expectedConstr:    "gsql.StringFieldOf[string]",
			expectedCategory:  "string",
		},
		{
			name:              "sql.NullString类型",
			field:             gormparse.GormFieldInfo{Type: "sql.NullString"},
			expectedFieldType: "gsql.StringField[sql.NullString]",
			expectedConstr:    "gsql.StringFieldOf[sql.NullString]",
			expectedCategory:  "string",
		},
		{
			name:              "[]byte类型",
			field:             gormparse.GormFieldInfo{Type: "[]byte"},
			expectedFieldType: "gsql.StringField[[]byte]",
			expectedConstr:    "gsql.StringFieldOf[[]byte]",
			expectedCategory:  "string",
		},
		// 时间类型
		{
			name:              "time.Time无SQLType默认datetime",
			field:             gormparse.GormFieldInfo{Type: "time.Time"},
			expectedFieldType: "gsql.DateTimeField[time.Time]",
			expectedConstr:    "gsql.DateTimeFieldOf[time.Time]",
			expectedCategory:  "datetime",
		},
		{
			name:              "time.Time带date类型",
			field:             gormparse.GormFieldInfo{Type: "time.Time", SQLType: "date"},
			expectedFieldType: "gsql.DateField[time.Time]",
			expectedConstr:    "gsql.DateFieldOf[time.Time]",
			expectedCategory:  "date",
		},
		{
			name:              "time.Time带time类型",
			field:             gormparse.GormFieldInfo{Type: "time.Time", SQLType: "time"},
			expectedFieldType: "gsql.TimeField[time.Time]",
			expectedConstr:    "gsql.TimeFieldOf[time.Time]",
			expectedCategory:  "time",
		},
		{
			name:              "指针time.Time类型",
			field:             gormparse.GormFieldInfo{Type: "*time.Time"},
			expectedFieldType: "gsql.DateTimeField[*time.Time]",
			expectedConstr:    "gsql.DateTimeFieldOf[*time.Time]",
			expectedCategory:  "datetime",
		},
		{
			name:              "sql.NullTime类型",
			field:             gormparse.GormFieldInfo{Type: "sql.NullTime"},
			expectedFieldType: "gsql.DateTimeField[sql.NullTime]",
			expectedConstr:    "gsql.DateTimeFieldOf[sql.NullTime]",
			expectedCategory:  "datetime",
		},
		{
			name:              "time.Time带datetime类型",
			field:             gormparse.GormFieldInfo{Type: "time.Time", SQLType: "datetime"},
			expectedFieldType: "gsql.DateTimeField[time.Time]",
			expectedConstr:    "gsql.DateTimeFieldOf[time.Time]",
			expectedCategory:  "datetime",
		},
		{
			name:              "time.Time带timestamp类型",
			field:             gormparse.GormFieldInfo{Type: "time.Time", SQLType: "timestamp"},
			expectedFieldType: "gsql.DateTimeField[time.Time]",
			expectedConstr:    "gsql.DateTimeFieldOf[time.Time]",
			expectedCategory:  "datetime",
		},
		// JSON类型
		{
			name:              "JSON类型映射到ScalarField",
			field:             gormparse.GormFieldInfo{Type: "MyJsonType", GormDataType: "json"},
			expectedFieldType: "gsql.ScalarField[MyJsonType]",
			expectedConstr:    "gsql.ScalarFieldOf[MyJsonType]",
			expectedCategory:  "json",
		},
		{
			name:              "指针JSON类型映射到ScalarField",
			field:             gormparse.GormFieldInfo{Type: "*JsonData", GormDataType: "json"},
			expectedFieldType: "gsql.ScalarField[*JsonData]",
			expectedConstr:    "gsql.ScalarFieldOf[*JsonData]",
			expectedCategory:  "json",
		},
		// 未知类型
		{
			name:              "自定义类型映射到ScalarField",
			field:             gormparse.GormFieldInfo{Type: "CustomType"},
			expectedFieldType: "gsql.ScalarField[CustomType]",
			expectedConstr:    "gsql.ScalarFieldOf[CustomType]",
			expectedCategory:  "scalar",
		},
		{
			name:              "指针自定义类型映射到ScalarField",
			field:             gormparse.GormFieldInfo{Type: "*CustomStruct"},
			expectedFieldType: "gsql.ScalarField[*CustomStruct]",
			expectedConstr:    "gsql.ScalarFieldOf[*CustomStruct]",
			expectedCategory:  "scalar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapFieldTypeInfo(tt.field)
			if result.FieldType != tt.expectedFieldType {
				t.Errorf("FieldType = %q, want %q", result.FieldType, tt.expectedFieldType)
			}
			if result.Constructor != tt.expectedConstr {
				t.Errorf("Constructor = %q, want %q", result.Constructor, tt.expectedConstr)
			}
			if result.FieldCategory != tt.expectedCategory {
				t.Errorf("FieldCategory = %q, want %q", result.FieldCategory, tt.expectedCategory)
			}
		})
	}
}

// TestIsTimeType 测试时间类型判断
func TestIsTimeType(t *testing.T) {
	tests := []struct {
		name     string
		goType   string
		expected bool
	}{
		{
			name:     "time.Time是时间类型",
			goType:   "time.Time",
			expected: true,
		},
		{
			name:     "*time.Time是时间类型",
			goType:   "*time.Time",
			expected: true,
		},
		{
			name:     "sql.NullTime是时间类型",
			goType:   "sql.NullTime",
			expected: true,
		},
		{
			name:     "string不是时间类型",
			goType:   "string",
			expected: false,
		},
		{
			name:     "int64不是时间类型",
			goType:   "int64",
			expected: false,
		},
		{
			name:     "空字符串不是时间类型",
			goType:   "",
			expected: false,
		},
		{
			name:     "Time不是时间类型",
			goType:   "Time",
			expected: false,
		},
		{
			name:     "datetime不是时间类型",
			goType:   "datetime",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimeType(tt.goType)
			if result != tt.expected {
				t.Errorf("isTimeType(%q) = %v, want %v", tt.goType, result, tt.expected)
			}
		})
	}
}

// TestIsIntType 测试整数类型判断
func TestIsIntType(t *testing.T) {
	tests := []struct {
		name     string
		goType   string
		expected bool
	}{
		{
			name:     "int是整数类型",
			goType:   "int",
			expected: true,
		},
		{
			name:     "int8是整数类型",
			goType:   "int8",
			expected: true,
		},
		{
			name:     "int16是整数类型",
			goType:   "int16",
			expected: true,
		},
		{
			name:     "int32是整数类型",
			goType:   "int32",
			expected: true,
		},
		{
			name:     "int64是整数类型",
			goType:   "int64",
			expected: true,
		},
		{
			name:     "uint是整数类型",
			goType:   "uint",
			expected: true,
		},
		{
			name:     "uint8是整数类型",
			goType:   "uint8",
			expected: true,
		},
		{
			name:     "uint16是整数类型",
			goType:   "uint16",
			expected: true,
		},
		{
			name:     "uint32是整数类型",
			goType:   "uint32",
			expected: true,
		},
		{
			name:     "uint64是整数类型",
			goType:   "uint64",
			expected: true,
		},
		{
			name:     "sql.NullInt16是整数类型",
			goType:   "sql.NullInt16",
			expected: true,
		},
		{
			name:     "sql.NullInt32是整数类型",
			goType:   "sql.NullInt32",
			expected: true,
		},
		{
			name:     "sql.NullInt64是整数类型",
			goType:   "sql.NullInt64",
			expected: true,
		},
		{
			name:     "string不是整数类型",
			goType:   "string",
			expected: false,
		},
		{
			name:     "float64不是整数类型",
			goType:   "float64",
			expected: false,
		},
		{
			name:     "bool不是整数类型",
			goType:   "bool",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIntType(tt.goType)
			if result != tt.expected {
				t.Errorf("isIntType(%q) = %v, want %v", tt.goType, result, tt.expected)
			}
		})
	}
}

// TestIsFloatType 测试浮点类型判断
func TestIsFloatType(t *testing.T) {
	tests := []struct {
		name     string
		goType   string
		expected bool
	}{
		{
			name:     "float32是浮点类型",
			goType:   "float32",
			expected: true,
		},
		{
			name:     "float64是浮点类型",
			goType:   "float64",
			expected: true,
		},
		{
			name:     "sql.NullFloat64是浮点类型",
			goType:   "sql.NullFloat64",
			expected: true,
		},
		{
			name:     "int不是浮点类型",
			goType:   "int",
			expected: false,
		},
		{
			name:     "string不是浮点类型",
			goType:   "string",
			expected: false,
		},
		{
			name:     "float不是浮点类型",
			goType:   "float",
			expected: false,
		},
		{
			name:     "double不是浮点类型",
			goType:   "double",
			expected: false,
		},
		{
			name:     "空字符串不是浮点类型",
			goType:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFloatType(tt.goType)
			if result != tt.expected {
				t.Errorf("isFloatType(%q) = %v, want %v", tt.goType, result, tt.expected)
			}
		})
	}
}

// TestIsBoolType 测试布尔类型判断
func TestIsBoolType(t *testing.T) {
	tests := []struct {
		name     string
		goType   string
		expected bool
	}{
		{
			name:     "bool是布尔类型",
			goType:   "bool",
			expected: true,
		},
		{
			name:     "sql.NullBool是布尔类型",
			goType:   "sql.NullBool",
			expected: true,
		},
		{
			name:     "Boolean不是布尔类型",
			goType:   "Boolean",
			expected: false,
		},
		{
			name:     "int不是布尔类型",
			goType:   "int",
			expected: false,
		},
		{
			name:     "string不是布尔类型",
			goType:   "string",
			expected: false,
		},
		{
			name:     "true不是布尔类型",
			goType:   "true",
			expected: false,
		},
		{
			name:     "空字符串不是布尔类型",
			goType:   "",
			expected: false,
		},
		{
			name:     "*bool不是布尔类型",
			goType:   "*bool",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBoolType(tt.goType)
			if result != tt.expected {
				t.Errorf("isBoolType(%q) = %v, want %v", tt.goType, result, tt.expected)
			}
		})
	}
}

// TestIsStringType 测试字符串类型判断
func TestIsStringType(t *testing.T) {
	tests := []struct {
		name     string
		goType   string
		expected bool
	}{
		{
			name:     "string是字符串类型",
			goType:   "string",
			expected: true,
		},
		{
			name:     "sql.NullString是字符串类型",
			goType:   "sql.NullString",
			expected: true,
		},
		{
			name:     "[]byte是字符串类型",
			goType:   "[]byte",
			expected: true,
		},
		{
			name:     "[]rune是字符串类型",
			goType:   "[]rune",
			expected: true,
		},
		{
			name:     "text类型包含text",
			goType:   "LongText",
			expected: true,
		},
		{
			name:     "blob类型包含blob",
			goType:   "TinyBlob",
			expected: true,
		},
		{
			name:     "int不是字符串类型",
			goType:   "int",
			expected: false,
		},
		{
			name:     "bool不是字符串类型",
			goType:   "bool",
			expected: false,
		},
		{
			name:     "空字符串不是字符串类型",
			goType:   "",
			expected: false,
		},
		{
			name:     "*string不是字符串类型",
			goType:   "*string",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStringType(tt.goType)
			if result != tt.expected {
				t.Errorf("isStringType(%q) = %v, want %v", tt.goType, result, tt.expected)
			}
		})
	}
}

// TestParseGormTag 测试gorm标签解析
func TestParseGormTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected map[string]string
	}{
		{
			name:     "单个键值对",
			tag:      `gorm:"column:user_name"`,
			expected: map[string]string{"column": "user_name"},
		},
		{
			name:     "多个键值对",
			tag:      `gorm:"column:id;type:bigint"`,
			expected: map[string]string{"column": "id", "type": "bigint"},
		},
		{
			name:     "布尔标签",
			tag:      `gorm:"primaryKey"`,
			expected: map[string]string{"primaryKey": ""},
		},
		{
			name:     "混合标签",
			tag:      `gorm:"column:id;primaryKey;autoIncrement"`,
			expected: map[string]string{"column": "id", "primaryKey": "", "autoIncrement": ""},
		},
		{
			name:     "空标签",
			tag:      `gorm:""`,
			expected: map[string]string{"": ""},
		},
		{
			name:     "无gorm标签",
			tag:      `json:"name"`,
			expected: map[string]string{},
		},
		{
			name:     "gorm与其他标签",
			tag:      `json:"id" gorm:"column:user_id"`,
			expected: map[string]string{"column": "user_id"},
		},
		{
			name:     "index带名称",
			tag:      `gorm:"index:idx_name"`,
			expected: map[string]string{"index": "idx_name"},
		},
		{
			name:     "uniqueIndex带名称",
			tag:      `gorm:"uniqueIndex:idx_email"`,
			expected: map[string]string{"uniqueIndex": "idx_email"},
		},
		{
			name:     "空字符串",
			tag:      "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGormTag(tt.tag)
			if len(result) != len(tt.expected) {
				t.Errorf("parseGormTag(%q) returned %d items, want %d", tt.tag, len(result), len(tt.expected))
				return
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("parseGormTag(%q)[%q] = %q, want %q", tt.tag, k, result[k], v)
				}
			}
		})
	}
}

// TestGetFieldFlags 测试标志位生成
func TestGetFieldFlags(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "primaryKey标志",
			tag:      `gorm:"primaryKey"`,
			expected: "field.FlagPrimaryKey",
		},
		{
			name:     "primarykey小写标志",
			tag:      `gorm:"primarykey"`,
			expected: "field.FlagPrimaryKey",
		},
		{
			name:     "autoIncrement标志",
			tag:      `gorm:"autoIncrement"`,
			expected: "field.FlagAutoIncrement",
		},
		{
			name:     "index标志",
			tag:      `gorm:"index"`,
			expected: "field.FlagIndex",
		},
		{
			name:     "index带名称标志",
			tag:      `gorm:"index:idx_name"`,
			expected: "field.FlagIndex",
		},
		{
			name:     "uniqueIndex标志",
			tag:      `gorm:"uniqueIndex"`,
			expected: "field.FlagUniqueIndex",
		},
		{
			name:     "unique标志",
			tag:      `gorm:"unique"`,
			expected: "field.FlagUniqueIndex",
		},
		{
			name:     "组合标志primaryKey和autoIncrement",
			tag:      `gorm:"primaryKey;autoIncrement"`,
			expected: "field.FlagPrimaryKey | field.FlagAutoIncrement",
		},
		{
			name:     "组合标志primaryKey和index",
			tag:      `gorm:"primaryKey;index"`,
			expected: "field.FlagPrimaryKey | field.FlagIndex",
		},
		{
			name:     "空标签",
			tag:      "",
			expected: "",
		},
		{
			name:     "无标志标签",
			tag:      `gorm:"column:name"`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFieldFlags(tt.tag)
			if result != tt.expected {
				t.Errorf("getFieldFlags(%q) = %q, want %q", tt.tag, result, tt.expected)
			}
		})
	}
}

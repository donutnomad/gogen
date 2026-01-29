package gormgen

import (
	"fmt"
	"slices"
	"strings"

	"github.com/donutnomad/gogen/internal/gormparse"
)

// FieldTypeInfo 字段类型信息
type FieldTypeInfo struct {
	FieldType     string // 字段类型，如 "gsql.IntField[uint64]"
	Constructor   string // 构造函数，如 "gsql.IntFieldOf[uint64]"
	FieldCategory string // 字段类别: int, float, string, datetime, date, time, scalar
}

// MapFieldTypeInfo 根据字段信息映射到新的 gsql Field 类型
func MapFieldTypeInfo(field gormparse.GormFieldInfo) FieldTypeInfo {
	goType := field.Type
	sqlType := field.SQLType
	gormDataType := field.GormDataType

	// 保留原始类型(包括包前缀)
	originalType := goType

	// 移除指针标记用于判断类型
	typeForCheck := strings.TrimPrefix(goType, "*")

	// JSON 类型 -> ScalarField[T]（通过 GormDataType 判断）
	// gsql 包没有专门的 JsonField，使用 ScalarField 存储 JSON 类型
	if gormDataType == "json" {
		return FieldTypeInfo{
			FieldType:     fmt.Sprintf("gsql.ScalarField[%s]", originalType),
			Constructor:   fmt.Sprintf("gsql.ScalarFieldOf[%s]", originalType),
			FieldCategory: "json",
		}
	}

	// decimal SQL 类型 -> DecimalField
	if sqlType == "decimal" {
		return FieldTypeInfo{
			FieldType:     fmt.Sprintf("gsql.DecimalField[%s]", originalType),
			Constructor:   fmt.Sprintf("gsql.DecimalFieldOf[%s]", originalType),
			FieldCategory: "decimal",
		}
	}

	// time.Time 类型根据 SQLType 细分
	if isTimeType(typeForCheck) {
		switch sqlType {
		case "date":
			return FieldTypeInfo{
				FieldType:     fmt.Sprintf("gsql.DateField[%s]", originalType),
				Constructor:   fmt.Sprintf("gsql.DateFieldOf[%s]", originalType),
				FieldCategory: "date",
			}
		case "time":
			return FieldTypeInfo{
				FieldType:     fmt.Sprintf("gsql.TimeField[%s]", originalType),
				Constructor:   fmt.Sprintf("gsql.TimeFieldOf[%s]", originalType),
				FieldCategory: "time",
			}
		default:
			// datetime, timestamp 或无标签，默认为 DateTimeField
			return FieldTypeInfo{
				FieldType:     fmt.Sprintf("gsql.DateTimeField[%s]", originalType),
				Constructor:   fmt.Sprintf("gsql.DateTimeFieldOf[%s]", originalType),
				FieldCategory: "datetime",
			}
		}
	}

	// int*, uint*, bool -> IntField
	if isIntType(typeForCheck) || isBoolType(typeForCheck) {
		return FieldTypeInfo{
			FieldType:     fmt.Sprintf("gsql.IntField[%s]", originalType),
			Constructor:   fmt.Sprintf("gsql.IntFieldOf[%s]", originalType),
			FieldCategory: "int",
		}
	}

	// float32, float64 -> FloatField
	if isFloatType(typeForCheck) {
		return FieldTypeInfo{
			FieldType:     fmt.Sprintf("gsql.FloatField[%s]", originalType),
			Constructor:   fmt.Sprintf("gsql.FloatFieldOf[%s]", originalType),
			FieldCategory: "float",
		}
	}

	// string, sql.NullString, []byte -> StringField
	if isStringType(typeForCheck) {
		return FieldTypeInfo{
			FieldType:     fmt.Sprintf("gsql.StringField[%s]", originalType),
			Constructor:   fmt.Sprintf("gsql.StringFieldOf[%s]", originalType),
			FieldCategory: "string",
		}
	}

	// 其他/未知类型 -> ScalarField
	return FieldTypeInfo{
		FieldType:     fmt.Sprintf("gsql.ScalarField[%s]", originalType),
		Constructor:   fmt.Sprintf("gsql.ScalarFieldOf[%s]", originalType),
		FieldCategory: "scalar",
	}
}

// isTimeType 判断是否为时间类型
func isTimeType(goType string) bool {
	timeTypes := []string{
		"time.Time",
		"*time.Time",
		"sql.NullTime",
	}
	return slices.Contains(timeTypes, goType)
}

// isIntType 判断是否为整数类型
func isIntType(goType string) bool {
	intTypes := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"sql.NullInt16", "sql.NullInt32", "sql.NullInt64",
	}
	return slices.Contains(intTypes, goType)
}

// isFloatType 判断是否为浮点类型
func isFloatType(goType string) bool {
	floatTypes := []string{
		"float32", "float64",
		"sql.NullFloat64",
	}
	return slices.Contains(floatTypes, goType)
}

// isBoolType 判断是否为布尔类型
func isBoolType(goType string) bool {
	boolTypes := []string{
		"bool",
		"sql.NullBool",
	}
	return slices.Contains(boolTypes, goType)
}

// isStringType 判断是否为字符串类型
func isStringType(goType string) bool {
	stringTypes := []string{
		"string",
		"sql.NullString",
		"[]byte",
		"[]rune",
	}

	if slices.Contains(stringTypes, goType) {
		return true
	}

	// 检查是否是text或blob类型(通常通过标签判断,这里简化处理)
	return strings.Contains(strings.ToLower(goType), "text") ||
		strings.Contains(strings.ToLower(goType), "blob")
}

// getFieldFlags 根据字段标签获取标志位
func getFieldFlags(tag string) string {
	if tag == "" {
		return ""
	}

	// 解析gorm标签
	gormTags := parseGormTag(tag)

	var flags []string

	// 检查是否为主键
	if _, hasPrimaryKey := gormTags["primarykey"]; hasPrimaryKey {
		flags = append(flags, "field.FlagPrimaryKey")
	}
	if _, hasPrimaryKey := gormTags["primaryKey"]; hasPrimaryKey {
		flags = append(flags, "field.FlagPrimaryKey")
	}

	// 检查是否有唯一索引
	if uniqueIdx, hasUniqueIndex := gormTags["uniqueIndex"]; hasUniqueIndex || uniqueIdx != "" {
		flags = append(flags, "field.FlagUniqueIndex")
	}
	if uniqueIdx, hasUniqueIndex := gormTags["unique"]; hasUniqueIndex || uniqueIdx != "" {
		flags = append(flags, "field.FlagUniqueIndex")
	}

	// 检查是否有普通索引
	if idx, hasIndex := gormTags["index"]; hasIndex || idx != "" {
		flags = append(flags, "field.FlagIndex")
	}

	// 检查是否自增
	if _, hasAutoIncrement := gormTags["autoIncrement"]; hasAutoIncrement {
		flags = append(flags, "field.FlagAutoIncrement")
	}

	if len(flags) == 0 {
		return ""
	}

	// 使用 | 组合多个标志
	return strings.Join(flags, " | ")
}

// parseGormTag 解析gorm标签
func parseGormTag(tag string) map[string]string {
	result := make(map[string]string)

	// 查找gorm标签
	start := strings.Index(tag, `gorm:"`)
	if start == -1 {
		return result
	}

	start += 6 // 跳过 gorm:"
	end := strings.Index(tag[start:], `"`)
	if end == -1 {
		return result
	}

	gormTag := tag[start : start+end]

	// 解析标签内的各个部分
	parts := strings.Split(gormTag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 {
				result[kv[0]] = kv[1]
			}
		} else {
			result[part] = ""
		}
	}

	return result
}

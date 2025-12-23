package gormgen

import (
	"fmt"
	"strings"
)

// mapFieldType 映射字段类型到field类型
func mapFieldType(goType string) string {
	// 保留原始类型(包括包前缀)
	originalType := goType

	// JSON类型忽略
	if isJSONType(goType) {
		return fmt.Sprintf("field.Pattern[%s]", "string")
	}

	// 移除指针标记用于判断类型
	typeForCheck := strings.TrimPrefix(goType, "*")

	// 字符串类型使用Pattern
	if isStringType(typeForCheck) {
		return fmt.Sprintf("field.Pattern[%s]", originalType)
	}

	// 其他类型使用Comparable
	return fmt.Sprintf("field.Comparable[%s]", originalType)
}

// isStringType 判断是否为字符串类型
func isStringType(goType string) bool {
	stringTypes := []string{
		"string",
		"sql.NullString",
		"[]byte",
		"[]rune",
	}

	for _, t := range stringTypes {
		if goType == t {
			return true
		}
	}

	// 检查是否是text或blob类型(通常通过标签判断,这里简化处理)
	return strings.Contains(strings.ToLower(goType), "text") ||
		strings.Contains(strings.ToLower(goType), "blob")
}

// isJSONType 判断是否为JSON类型
func isJSONType(goType string) bool {
	return strings.Contains(strings.ToLower(goType), "json") ||
		goType == "datatypes.JSON" ||
		goType == "gorm.datatypes.JSON"
}

// getFieldConstructor 获取字段构造函数
func getFieldConstructor(fieldType string) string {
	if strings.Contains(fieldType, "Pattern") {
		// 提取泛型参数
		start := strings.Index(fieldType, "[")
		end := strings.LastIndex(fieldType, "]")
		if start != -1 && end != -1 {
			typeParam := fieldType[start+1 : end]
			return fmt.Sprintf("field.NewPattern[%s]", typeParam)
		}
	}

	if strings.Contains(fieldType, "Comparable") {
		// 提取泛型参数
		start := strings.Index(fieldType, "[")
		end := strings.LastIndex(fieldType, "]")
		if start != -1 && end != -1 {
			typeParam := fieldType[start+1 : end]
			return fmt.Sprintf("field.NewComparable[%s]", typeParam)
		}
	}

	return "field.NewComparable[any]"
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

// isJSONType2 检查类型是否是datatypes.JSONType[T]格式
func isJSONType2(fieldType string) bool {
	return strings.HasPrefix(fieldType, "datatypes.JSONType[") && strings.HasSuffix(fieldType, "]")
}

// isJSONSliceType 检查类型是否是datatypes.JSONSlice[T]格式
func isJSONSliceType(fieldType string) bool {
	return strings.HasPrefix(fieldType, "datatypes.JSONSlice[") && strings.HasSuffix(fieldType, "]")
}

// extractJSONTypeParameter 从datatypes.JSONType[T]中提取T
func extractJSONTypeParameter(fieldType string) string {
	if !isJSONType2(fieldType) {
		return fieldType
	}

	// 去除"datatypes.JSONType["前缀和"]"后缀
	start := len("datatypes.JSONType[")
	end := len(fieldType) - 1
	return fieldType[start:end]
}

// extractJSONSliceParameter 从datatypes.JSONSlice[T]中提取T
func extractJSONSliceParameter(fieldType string) string {
	if !isJSONSliceType(fieldType) {
		return fieldType
	}

	// 去除"datatypes.JSONSlice["前缀和"]"后缀
	start := len("datatypes.JSONSlice[")
	end := len(fieldType) - 1
	return fieldType[start:end]
}

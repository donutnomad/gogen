package pickgen

import (
	"fmt"

	"github.com/donutnomad/gogen/internal/structparse"
)

// filterFields 根据模式过滤字段
func filterFields(allFields []structparse.FieldInfo, fieldNames []string, mode SelectionMode) ([]structparse.FieldInfo, error) {
	// 构建字段名集合用于快速查找
	fieldSet := make(map[string]bool)
	for _, name := range fieldNames {
		fieldSet[name] = true
	}

	// 构建所有可用字段名集合（用于错误提示）
	availableFields := make(map[string]bool)
	for _, field := range allFields {
		availableFields[field.Name] = true
	}

	// 验证字段是否存在
	var missingFields []string
	for _, name := range fieldNames {
		if !availableFields[name] {
			missingFields = append(missingFields, name)
		}
	}
	if len(missingFields) > 0 {
		availableList := make([]string, 0, len(availableFields))
		for name := range availableFields {
			availableList = append(availableList, name)
		}
		return nil, fmt.Errorf("字段不存在: %v，可用字段: %v", missingFields, availableList)
	}

	// 过滤字段
	var result []structparse.FieldInfo
	for _, field := range allFields {
		shouldInclude := false
		switch mode {
		case ModePick:
			// Pick 模式：只包含指定字段
			shouldInclude = fieldSet[field.Name]
		case ModeOmit:
			// Omit 模式：排除指定字段
			shouldInclude = !fieldSet[field.Name]
		}
		if shouldInclude {
			result = append(result, field)
		}
	}

	return result, nil
}

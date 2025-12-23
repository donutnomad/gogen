package plugin

import (
	"regexp"
	"strings"
)

// annotationRegex 匹配注解 @Name 或 @Name(params)
var annotationRegex = regexp.MustCompile(`@(\w+)(?:\(([^)]*)\))?`)

// paramRegex 匹配参数:
// - key=`value` (反引号格式)
// - key="value" (双引号格式)
// - key=value (普通格式)
var paramRegex = regexp.MustCompile("(\\w+)\\s*=\\s*`([^`]*)`|(\\w+)\\s*=\\s*\"([^\"]*)\"|(\\w+)\\s*=\\s*([^,\\s]+)")

// ParseAnnotations 从注释文本中解析所有注解
func ParseAnnotations(comment string) []*Annotation {
	var annotations []*Annotation

	// 按行处理
	lines := strings.Split(comment, "\n")
	for _, line := range lines {
		// 去除注释前缀
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(line)

		// 查找所有注解
		matches := annotationRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			ann := &Annotation{
				Name:   match[1],
				Params: make(map[string]string),
				Raw:    match[0],
			}

			// 解析参数
			if len(match) > 2 && match[2] != "" {
				ann.Params = parseParams(match[2])
			}

			annotations = append(annotations, ann)
		}
	}

	return annotations
}

// parseParams 解析注解参数
func parseParams(content string) map[string]string {
	params := make(map[string]string)

	matches := paramRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		var key, value string
		if match[1] != "" {
			// 反引号格式: key=`value`
			key = strings.ToLower(match[1])
			value = match[2]
		} else if match[3] != "" {
			// 双引号格式: key="value"
			key = strings.ToLower(match[3])
			value = match[4]
		} else if match[5] != "" {
			// 普通格式: key=value
			key = strings.ToLower(match[5])
			value = match[6]
		}
		if key != "" {
			params[key] = value
		}
	}

	return params
}

// ParseAnnotationsFromDoc 从 ast.CommentGroup 中解析注解
func ParseAnnotationsFromDoc(text string) []*Annotation {
	return ParseAnnotations(text)
}

// FilterByNames 过滤指定名称的注解
func FilterByNames(annotations []*Annotation, names ...string) []*Annotation {
	if len(names) == 0 {
		return annotations
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	var result []*Annotation
	for _, ann := range annotations {
		if nameSet[ann.Name] {
			result = append(result, ann)
		}
	}
	return result
}

// HasAnnotation 检查是否包含指定注解
func HasAnnotation(annotations []*Annotation, name string) bool {
	for _, ann := range annotations {
		if ann.Name == name {
			return true
		}
	}
	return false
}

// GetAnnotation 获取指定名称的注解
func GetAnnotation(annotations []*Annotation, name string) *Annotation {
	for _, ann := range annotations {
		if ann.Name == name {
			return ann
		}
	}
	return nil
}

// GetParam 获取注解参数
func (a *Annotation) GetParam(key string) string {
	return a.Params[strings.ToLower(key)]
}

// GetParamOr 获取注解参数，如果不存在返回默认值
func (a *Annotation) GetParamOr(key, defaultValue string) string {
	if v, ok := a.Params[strings.ToLower(key)]; ok {
		return v
	}
	return defaultValue
}

// HasParam 检查是否有指定参数
func (a *Annotation) HasParam(key string) bool {
	_, ok := a.Params[strings.ToLower(key)]
	return ok
}

package structparse

import (
	"fmt"
	"path/filepath"
	"strings"
)

// parseGormEmbeddedTag 解析 gorm 标签中的 embedded 和 embeddedPrefix
// 返回: (是否embedded, embeddedPrefix值)
func parseGormEmbeddedTag(tag string) (bool, string) {
	// 查找 gorm 标签
	gormStart := strings.Index(tag, `gorm:"`)
	if gormStart == -1 {
		return false, ""
	}

	// 安全检查：确保有足够的长度
	if len(tag) < gormStart+7 { // gorm:" 是6个字符 + 至少1个字符
		return false, ""
	}

	gormStart += 6 // 跳过 gorm:"
	gormEnd := strings.Index(tag[gormStart:], `"`)
	if gormEnd == -1 {
		return false, ""
	}

	gormTag := tag[gormStart : gormStart+gormEnd]

	// 解析标签内的各个部分
	parts := strings.Split(gormTag, ";")
	isEmbedded := false
	embeddedPrefix := ""

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "embedded" {
			isEmbedded = true
		} else if strings.HasPrefix(part, "embeddedPrefix:") {
			embeddedPrefix = strings.TrimPrefix(part, "embeddedPrefix:")
		}
	}

	return isEmbedded, embeddedPrefix
}

// shouldExpandEmbeddedField 判断是否应该展开嵌入字段
func shouldExpandEmbeddedField(fieldType string) bool {
	// 内置类型不展开
	builtinTypes := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"string", "bool",
		"byte", "rune",
		"error",
		"time.Time", "time.Duration",
	}

	for _, builtin := range builtinTypes {
		if fieldType == builtin {
			return false
		}
	}

	// 跳过指针、切片、映射等复合类型
	if strings.HasPrefix(fieldType, "*") ||
		strings.HasPrefix(fieldType, "[]") ||
		strings.HasPrefix(fieldType, "map[") ||
		strings.HasPrefix(fieldType, "chan ") ||
		strings.HasPrefix(fieldType, "func(") {
		return false
	}

	// 其他所有结构体类型都尝试展开
	return true
}

// parseEmbeddedStructWithStack 带栈的递归解析，避免循环引用
// baseDir: 原始文件所在的目录，用于同包结构体查找
func (c *ParseContext) parseEmbeddedStructWithStack(structType string, stack map[string]bool, imports map[string]*ImportInfo, baseDir string) ([]FieldInfo, error) {
	// 检查嵌套深度限制
	if len(stack) >= maxEmbeddingDepth {
		return nil, fmt.Errorf("嵌入字段深度超过限制 %d: %s", maxEmbeddingDepth, structType)
	}

	// 检查是否已经在解析栈中（避免循环引用）
	if stack[structType] {
		return nil, nil
	}

	// 将当前类型加入解析栈
	stack[structType] = true
	defer delete(stack, structType) // 解析完成后从栈中移除

	// 解析包名和结构体名
	packageName, structName := parseTypePackageAndName(structType)

	var targetFile string
	var err error

	if packageName == "" {
		// 同包内的结构体，在原始文件所在目录查找
		files, err := findGoFiles(baseDir)
		if err != nil {
			return nil, fmt.Errorf("查找目录 %s 中的Go文件失败: %v", baseDir, err)
		}

		for _, file := range files {
			if containsStruct(file, structName) {
				targetFile = file
				break
			}
		}

		if targetFile == "" {
			return nil, fmt.Errorf("未在包目录 %s 中找到结构体 %s", baseDir, structName)
		}
	} else {
		// 跨包结构体，需要根据import路径查找
		targetFile, err = c.findStructInPackageWithImportsAndBaseDir(packageName, structName, imports, baseDir)
		if err != nil {
			// 明确报告找不到第三方包的错误
			return nil, fmt.Errorf("无法解析嵌入的结构体 %s.%s: %v", packageName, structName, err)
		}

		if targetFile == "" {
			return nil, fmt.Errorf("未在包 %s 中找到结构体 %s", packageName, structName)
		}
	}

	// 递归解析该结构体
	structInfo, err := c.parseStructWithStackAndImportsAndBaseDir(targetFile, structName, stack, imports, filepath.Dir(targetFile))
	if err != nil {
		return nil, fmt.Errorf("解析嵌入结构体 %s 失败: %v", structType, err)
	}

	// 为从嵌入结构体来的字段标记来源
	fields := make([]FieldInfo, len(structInfo.Fields))
	for i, field := range structInfo.Fields {
		fields[i] = field
		// 如果字段已经有来源标记，保持原来的来源；否则标记为当前嵌入类型
		if field.SourceType == "" {
			fields[i].SourceType = structType
		}
	}

	return fields, nil
}

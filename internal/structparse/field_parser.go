package structparse

import (
	"go/ast"
	"strings"

	"github.com/donutnomad/gogen/internal/xast"
)

// parseStructFieldsWithStackAndImports 带栈和导入信息的字段解析（向后兼容）
func (c *ParseContext) parseStructFieldsWithStackAndImports(fieldList []*ast.Field, stack map[string]bool, imports map[string]*ImportInfo) ([]FieldInfo, error) {
	// 使用当前工作目录作为默认 baseDir（向后兼容）
	return c.parseStructFieldsWithStackAndImportsAndBaseDir(fieldList, stack, imports, ".")
}

// parseStructFieldsWithStackAndImportsAndBaseDir 带栈、导入信息和基础目录的字段解析
func (c *ParseContext) parseStructFieldsWithStackAndImportsAndBaseDir(fieldList []*ast.Field, stack map[string]bool, imports map[string]*ImportInfo, baseDir string) ([]FieldInfo, error) {
	var fields []FieldInfo

	for _, field := range fieldList {
		fieldType := xast.GetFieldType(field.Type, nil)

		// 提取 PkgPath
		pkgPath := extractPkgPath(fieldType, imports)

		// 获取字段标签
		var fieldTag string
		if field.Tag != nil {
			fieldTag = field.Tag.Value
		}

		if len(field.Names) == 0 {
			// 匿名字段 (嵌入字段)
			if shouldExpandEmbeddedField(fieldType) {
				// 需要扩展的嵌入字段，尝试递归解析
				embeddedFields, err := c.parseEmbeddedStructWithStack(fieldType, stack, imports, baseDir)
				if err != nil {
					return nil, err // 传递错误给上层
				}
				fields = append(fields, embeddedFields...)
			} else {
				// 不需要扩展的嵌入字段，保持原样
				fields = append(fields, FieldInfo{
					Name:    fieldType,
					Type:    fieldType,
					PkgPath: pkgPath,
					Tag:     fieldTag,
				})
			}
		} else {
			// 有名字段
			for _, name := range field.Names {
				// 检查是否有 gorm:"embedded" 标签
				isEmbedded, embeddedPrefix := parseGormEmbeddedTag(fieldTag)
				if isEmbedded && shouldExpandEmbeddedField(fieldType) {
					// 需要展开的 embedded 字段，递归解析
					embeddedFields, err := c.parseEmbeddedStructWithStack(fieldType, stack, imports, baseDir)
					if err != nil {
						return nil, err
					}
					// 为展开的字段添加 embeddedPrefix
					for i := range embeddedFields {
						if embeddedPrefix != "" {
							// 累加 prefix（支持多层嵌套）
							if embeddedFields[i].EmbeddedPrefix != "" {
								embeddedFields[i].EmbeddedPrefix = embeddedPrefix + embeddedFields[i].EmbeddedPrefix
							} else {
								embeddedFields[i].EmbeddedPrefix = embeddedPrefix
							}
						}
					}
					fields = append(fields, embeddedFields...)
				} else {
					fields = append(fields, FieldInfo{
						Name:    name.Name,
						Type:    fieldType,
						PkgPath: pkgPath,
						Tag:     fieldTag,
					})
				}
			}
		}
	}

	for i, field := range fields {
		if field.SourceType != "" {
			if idx := strings.Index(field.SourceType, "."); idx >= 0 {
				if !strings.Contains(field.Type, ".") && (field.Type[0] >= 'A' && field.Type[0] <= 'Z') {
					fields[i].Type = field.SourceType[:idx] + "." + field.Type
				}
			}
		}
	}

	return fields, nil
}

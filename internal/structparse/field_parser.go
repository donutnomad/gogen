package structparse

import (
	"go/ast"

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

		// 提取 PkgPath 和 PkgAlias
		pkgPath, pkgAlias := extractPkgPathAndAlias(fieldType, imports)

		// 获取字段标签
		var fieldTag string
		if field.Tag != nil {
			fieldTag = field.Tag.Value
		}

		if len(field.Names) == 0 {
			// 匿名字段 (嵌入字段) - 直接通过字段名访问，不需要设置 SourceField
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
					Name:     fieldType,
					Type:     fieldType,
					PkgPath:  pkgPath,
					PkgAlias: pkgAlias,
					Tag:      fieldTag,
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
					// 为展开的字段添加 embeddedPrefix 和 SourceField
					for i := range embeddedFields {
						// 设置嵌入字段在主结构体中的访问路径
						// 支持多层嵌套：Outer.Inner.Field
						if embeddedFields[i].SourceField == "" {
							embeddedFields[i].SourceField = name.Name
						} else {
							// 多层嵌套时，累加路径
							embeddedFields[i].SourceField = name.Name + "." + embeddedFields[i].SourceField
						}
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
						Name:     name.Name,
						Type:     fieldType,
						PkgPath:  pkgPath,
						PkgAlias: pkgAlias,
						Tag:      fieldTag,
					})
				}
			}
		}
	}

	return fields, nil
}

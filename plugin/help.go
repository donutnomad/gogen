package plugin

import (
	"fmt"
	"strings"
)

// FormatHelpText 为所有注册的生成器生成帮助文本
func FormatHelpText(registry *Registry) string {
	generators := registry.Generators()
	if len(generators) == 0 {
		return "  (暂无已注册的生成器)\n"
	}

	var sb strings.Builder

	for _, gen := range generators {
		annotations := gen.Annotations()
		if len(annotations) == 0 {
			continue
		}

		// 获取主注解名
		mainAnnotation := annotations[0]
		paramDefs := gen.ParamDefs()

		// 生成器描述
		sb.WriteString(fmt.Sprintf("  @%s - %s\n", mainAnnotation, gen.Name()))

		// 显示参数定义
		sb.WriteString("    参数:\n")

		// 通用参数
		sb.WriteString("      output - 输出文件路径（支持模板变量）\n")

		// 生成器特定参数
		if len(paramDefs) > 0 {
			for _, param := range paramDefs {
				// 生成参数行
				required := ""
				if param.Required {
					required = " (必填)"
				}

				defaultVal := ""
				if param.Default != "" {
					defaultVal = fmt.Sprintf(" [默认: %s]", param.Default)
				}

				sb.WriteString(fmt.Sprintf("      %s%s%s - %s\n",
					param.Name, required, defaultVal, param.Description))
			}
		}

		// 生成用法示例
		sb.WriteString("    示例:\n")
		sb.WriteString(fmt.Sprintf("      @%s\n", mainAnnotation))
		sb.WriteString(fmt.Sprintf("      @%s(output={{FileName}}_query.go)\n", mainAnnotation))
		sb.WriteString(fmt.Sprintf("      @%s(output={{StructName}}_query.go)\n", mainAnnotation))

		// 单个参数示例（如果有）
		if len(paramDefs) > 0 {
			for i, param := range paramDefs {
				if i >= 2 {
					break // 只显示前2个参数的示例
				}
				if param.Default != "" {
					sb.WriteString(fmt.Sprintf("      @%s(%s=%s)\n",
						mainAnnotation, param.Name, param.Default))
				}
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatParamDef 格式化单个参数定义
func FormatParamDef(param ParamDef) string {
	parts := []string{param.Name}

	if param.Required {
		parts = append(parts, "required")
	} else {
		parts = append(parts, "optional")
	}

	if param.Default != "" {
		parts = append(parts, fmt.Sprintf("default=%s", param.Default))
	}

	if param.Description != "" {
		parts = append(parts, param.Description)
	}

	return strings.Join(parts, ", ")
}

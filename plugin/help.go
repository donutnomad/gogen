package plugin

import (
	"fmt"
	"strings"
)

// ExtraHelpProvider 可选接口，生成器可实现此接口提供额外帮助文本
type ExtraHelpProvider interface {
	// ExtraHelp 返回额外的帮助文本（如辅助注解说明）
	ExtraHelp() string
}

// AnnotationFormatProvider 可选接口，生成器可实现此接口自定义触发注解的显示格式
// 例如 swaggen 可以返回 []string{"GET(path)", "POST(path)"} 来显示参数
type AnnotationFormatProvider interface {
	// AnnotationFormats 返回每个触发注解的显示格式（不含@前缀）
	// 返回 nil 使用默认格式
	AnnotationFormats() []string
}

// NoDefaultParamsProvider 可选接口，生成器实现此接口表示不显示默认参数和示例
// 适用于方法级别注解（如 @GET）而非类型级别注解（如 @Gsql）
type NoDefaultParamsProvider interface {
	// NoDefaultParams 返回 true 表示不显示默认的 output 参数和示例
	NoDefaultParams() bool
}

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

		// 生成器描述，显示所有触发注解
		if len(annotations) == 1 {
			sb.WriteString(fmt.Sprintf("  @%s - %s\n", mainAnnotation, gen.Name()))
		} else {
			// 多个注解时显示所有
			annotationList := make([]string, len(annotations))
			for i, ann := range annotations {
				annotationList[i] = "@" + ann
			}
			sb.WriteString(fmt.Sprintf("  %s - %s\n", strings.Join(annotationList, ", "), gen.Name()))
		}

		// 检查是否不显示默认参数
		noDefaultParams := false
		if provider, ok := gen.(NoDefaultParamsProvider); ok {
			noDefaultParams = provider.NoDefaultParams()
		}

		// 获取自定义注解格式
		var annotationFormats []string
		if provider, ok := gen.(AnnotationFormatProvider); ok {
			annotationFormats = provider.AnnotationFormats()
		}

		// 显示触发注解说明（如果有多个）
		if len(annotations) > 1 {
			sb.WriteString("    触发注解:\n")
			for i, ann := range annotations {
				// 使用自定义格式或默认格式
				if annotationFormats != nil && i < len(annotationFormats) {
					sb.WriteString(fmt.Sprintf("      @%s\n", annotationFormats[i]))
				} else {
					sb.WriteString(fmt.Sprintf("      @%s\n", ann))
				}
			}
		}

		// 显示参数定义（如果不是 noDefaultParams 模式）
		if !noDefaultParams {
			sb.WriteString("    参数:\n")

			// 检查生成器是否定义了自己的 output 参数
			hasOutput := false
			for _, param := range paramDefs {
				if param.Name == "output" {
					hasOutput = true
					break
				}
			}

			// 如果没有自定义 output，显示通用参数
			if !hasOutput {
				sb.WriteString("      output - 输出文件路径（支持模板变量）\n")
			}

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
			sb.WriteString(fmt.Sprintf("      @%s(output=$FILE_query.go)\n", mainAnnotation))
			sb.WriteString(fmt.Sprintf("      @%s(output=$PACKAGE_query.go)\n", mainAnnotation))

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
		}

		// 检查是否有额外帮助文本
		if extraHelper, ok := gen.(ExtraHelpProvider); ok {
			if extraHelp := extraHelper.ExtraHelp(); extraHelp != "" {
				sb.WriteString(extraHelp)
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

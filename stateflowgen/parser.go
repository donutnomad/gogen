package stateflowgen

import (
	"fmt"
	"regexp"
	"strings"
)

// StateFlowConfig 配置注解解析结果
type StateFlowConfig struct {
	Name   string // 类型前缀，如 "Server"
	Output string // 可选：输出文件路径
}

// FlowRule 单条流转规则
type FlowRule struct {
	Source  StateRef
	Targets []TargetRef
}

// StateRef 源状态引用
type StateRef struct {
	Phase    string // Phase 名称
	Status   string // Status 名称，可为空
	Wildcard bool   // 是否为 * 通配符
}

// TargetRef 目标状态引用（包含审批信息）
type TargetRef struct {
	Phase            string // Phase 名称，可为空（纯状态切换时继承源 Phase）
	Status           string // Status 名称，可为空
	Self             bool   // 是否为 = 自我流转
	ApprovalRequired bool   // ! 标记
	ApprovalOptional bool   // ? 标记
	Via              string // via 中间状态（Phase 名称）
	ViaStatus        string // via 中间状态的 Status（可为空）
	Else             string // else 拒绝后状态（Phase 名称），为空则回退原状态
	ElseStatus       string // else 拒绝后状态的 Status（可为空）
}

// stateFlowConfigRegex 匹配 @StateFlow(name="xxx") 或 @StateFlow() 或 @StateFlow
var stateFlowConfigRegex = regexp.MustCompile(`@StateFlow(?:\(([^)]*)\))?`)

// flowRuleRegex 匹配 @Flow: Source => [ Targets ]
var flowRuleRegex = regexp.MustCompile(`@Flow:\s*(.+)`)

// paramRegex 匹配参数 key="value" 或 key=`value` 或 key=value
var paramRegex = regexp.MustCompile(`(\w+)\s*=\s*(?:"([^"]*)"|` + "`" + `([^` + "`" + `]*)` + "`" + `|(\w+))`)

// ParseStateFlowConfig 从文本中解析 @StateFlow 配置
// 支持以下格式:
//   - @StateFlow(name="xxx") - 明确指定名称
//   - @StateFlow() - 空括号，name 为空
//   - @StateFlow - 无括号，name 为空
func ParseStateFlowConfig(text string) (*StateFlowConfig, error) {
	matches := stateFlowConfigRegex.FindStringSubmatch(text)
	if len(matches) == 0 {
		return nil, fmt.Errorf("invalid @StateFlow format: %s", text)
	}

	config := &StateFlowConfig{}

	// matches[1] 是括号内的参数，可能为空字符串或不存在
	if len(matches) > 1 && matches[1] != "" {
		params := matches[1]

		// 解析参数
		paramMatches := paramRegex.FindAllStringSubmatch(params, -1)
		for _, m := range paramMatches {
			key := strings.ToLower(m[1])
			var value string
			if m[2] != "" {
				value = m[2] // 双引号格式
			} else if m[3] != "" {
				value = m[3] // 反引号格式
			} else {
				value = m[4] // 无引号格式
			}

			switch key {
			case "name":
				config.Name = value
			case "output":
				config.Output = value
			}
		}
	}

	// name 为空时允许，由调用方提供默认值
	return config, nil
}

// ParseFlowRule 从文本行中解析 @Flow 规则
// 格式: @Flow: Source(Status) => [ Target1!, Target2? ]
// 或: @Flow: Init (无流转，单节点声明)
func ParseFlowRule(line string) (*FlowRule, error) {
	matches := flowRuleRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid @Flow format: %s", line)
	}

	content := strings.TrimSpace(matches[1])

	// 检查是否有 => 符号
	parts := strings.SplitN(content, "=>", 2)

	// 解析源状态
	sourceStr := strings.TrimSpace(parts[0])
	source, err := parseStateRef(sourceStr)
	if err != nil {
		return nil, fmt.Errorf("invalid source state '%s': %w", sourceStr, err)
	}

	rule := &FlowRule{
		Source: *source,
	}

	// 如果没有 => 符号，这是单节点声明
	if len(parts) == 1 {
		return rule, nil
	}

	// 解析目标列表
	targetsPart := strings.TrimSpace(parts[1])
	if !strings.HasPrefix(targetsPart, "[") || !strings.HasSuffix(targetsPart, "]") {
		return nil, fmt.Errorf("targets must be enclosed in brackets: %s", targetsPart)
	}

	// 去除括号
	targetsPart = strings.TrimPrefix(targetsPart, "[")
	targetsPart = strings.TrimSuffix(targetsPart, "]")
	targetsPart = strings.TrimSpace(targetsPart)

	if targetsPart == "" {
		return nil, fmt.Errorf("empty target list")
	}

	// 分割目标（按逗号分隔，但要注意 via 和 else 关键字）
	targets, err := splitTargets(targetsPart)
	if err != nil {
		return nil, err
	}

	for _, targetStr := range targets {
		target, err := parseTargetRef(targetStr)
		if err != nil {
			return nil, fmt.Errorf("invalid target '%s': %w", targetStr, err)
		}
		rule.Targets = append(rule.Targets, *target)
	}

	return rule, nil
}

// parseStateRef 解析源状态引用
func parseStateRef(s string) (*StateRef, error) {
	s = strings.TrimSpace(s)

	ref := &StateRef{}

	// 检查是否有括号
	if idx := strings.Index(s, "("); idx != -1 {
		if !strings.HasSuffix(s, ")") {
			return nil, fmt.Errorf("unmatched parenthesis")
		}
		ref.Phase = s[:idx]
		status := s[idx+1 : len(s)-1]

		if status == "*" {
			ref.Wildcard = true
		} else {
			ref.Status = status
		}
	} else {
		ref.Phase = s
	}

	if ref.Phase == "" && ref.Status == "" && !ref.Wildcard {
		return nil, fmt.Errorf("empty state reference")
	}

	return ref, nil
}

// splitTargets 分割目标列表，处理 via/else 关键字
func splitTargets(s string) ([]string, error) {
	var targets []string
	var current strings.Builder
	parenDepth := 0

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c == '(' {
			parenDepth++
			current.WriteByte(c)
		} else if c == ')' {
			parenDepth--
			current.WriteByte(c)
		} else if c == ',' && parenDepth == 0 {
			if current.Len() > 0 {
				targets = append(targets, strings.TrimSpace(current.String()))
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}

	if current.Len() > 0 {
		targets = append(targets, strings.TrimSpace(current.String()))
	}

	return targets, nil
}

// parseTargetRef 解析目标状态引用
// 格式: Phase(Status)! via Intermediate else Fallback
// 或: (Status)! via Intermediate else Fallback
// 或: (=)? via Intermediate
func parseTargetRef(s string) (*TargetRef, error) {
	s = strings.TrimSpace(s)

	ref := &TargetRef{}

	// 分割 via 和 else 部分
	mainPart := s
	viaPart := ""
	elsePart := ""

	// 查找 via 关键字
	if idx := strings.Index(strings.ToLower(s), " via "); idx != -1 {
		mainPart = s[:idx]
		rest := s[idx+5:] // len(" via ") = 5

		// 查找 else 关键字
		if elseIdx := strings.Index(strings.ToLower(rest), " else "); elseIdx != -1 {
			viaPart = strings.TrimSpace(rest[:elseIdx])
			elsePart = strings.TrimSpace(rest[elseIdx+6:]) // len(" else ") = 6
		} else {
			viaPart = strings.TrimSpace(rest)
		}
	}

	mainPart = strings.TrimSpace(mainPart)

	// 检查审批标记
	if strings.HasSuffix(mainPart, "!") {
		ref.ApprovalRequired = true
		mainPart = strings.TrimSuffix(mainPart, "!")
	} else if strings.HasSuffix(mainPart, "?") {
		ref.ApprovalOptional = true
		mainPart = strings.TrimSuffix(mainPart, "?")
	}

	mainPart = strings.TrimSpace(mainPart)

	// 解析主状态部分
	if mainPart == "(=)" {
		ref.Self = true
	} else if strings.HasPrefix(mainPart, "(") && strings.HasSuffix(mainPart, ")") {
		// 纯状态切换 (Status)
		inner := mainPart[1 : len(mainPart)-1]
		if inner == "=" {
			ref.Self = true
		} else {
			ref.Status = inner
		}
	} else if idx := strings.Index(mainPart, "("); idx != -1 {
		// Phase(Status) 格式
		if !strings.HasSuffix(mainPart, ")") {
			return nil, fmt.Errorf("unmatched parenthesis in '%s'", mainPart)
		}
		ref.Phase = mainPart[:idx]
		ref.Status = mainPart[idx+1 : len(mainPart)-1]
	} else {
		// 只有 Phase
		ref.Phase = mainPart
	}

	// 解析 via 部分
	if viaPart != "" {
		viaRef, err := parseStateRef(viaPart)
		if err != nil {
			return nil, fmt.Errorf("invalid via state '%s': %w", viaPart, err)
		}
		ref.Via = viaRef.Phase
		ref.ViaStatus = viaRef.Status
	}

	// 解析 else 部分
	if elsePart != "" {
		elseRef, err := parseStateRef(elsePart)
		if err != nil {
			return nil, fmt.Errorf("invalid else state '%s': %w", elsePart, err)
		}
		ref.Else = elseRef.Phase
		ref.ElseStatus = elseRef.Status
	}

	return ref, nil
}

// ParseFlowAnnotations 从完整注释文本中解析所有 @StateFlow 和 @Flow 注解
func ParseFlowAnnotations(text string) (*StateFlowConfig, []*FlowRule, error) {
	var config *StateFlowConfig
	var rules []*FlowRule

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// 检查 @StateFlow
		if strings.Contains(line, "@StateFlow") {
			cfg, err := ParseStateFlowConfig(line)
			if err != nil {
				return nil, nil, err
			}
			if config != nil {
				return nil, nil, fmt.Errorf("multiple @StateFlow annotations found")
			}
			config = cfg
			continue
		}

		// 检查 @Flow
		if strings.Contains(line, "@Flow:") {
			rule, err := ParseFlowRule(line)
			if err != nil {
				return nil, nil, err
			}
			rules = append(rules, rule)
		}
	}

	return config, rules, nil
}

package stateflowgen

import (
	"fmt"
	"regexp"
	"strings"
)

type StateFlowV2Config struct {
	Name         string
	Output       string
	StatusType   string
	StatusValues map[string]string
}

var stateFlowV2ConfigRegex = regexp.MustCompile(`@StateFlowV2(?:\(([^)]*)\))?`)

func ParseStateFlowV2Config(text string) (*StateFlowV2Config, error) {
	matches := stateFlowV2ConfigRegex.FindStringSubmatch(text)
	if len(matches) == 0 {
		return nil, fmt.Errorf("invalid @StateFlowV2 format: %s", text)
	}

	config := &StateFlowV2Config{}
	if len(matches) > 1 && matches[1] != "" {
		paramMatches := paramRegex.FindAllStringSubmatch(matches[1], -1)
		for _, match := range paramMatches {
			key := strings.ToLower(match[1])
			value := annotationParamValue(match)

			switch key {
			case "name":
				config.Name = value
			case "output":
				config.Output = value
			case "statustype":
				config.StatusType = value
			case "statusvalues":
				statusValues, err := parseStatusValues(value)
				if err != nil {
					return nil, err
				}
				config.StatusValues = statusValues
			}
		}
	}
	return config, nil
}

func ParseFlowV2Annotations(text string) (*StateFlowV2Config, []*FlowRule, error) {
	var config *StateFlowV2Config
	var rules []*FlowRule

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = normalizeAnnotationLine(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "@StateFlowV2") {
			cfg, err := ParseStateFlowV2Config(line)
			if err != nil {
				return nil, nil, err
			}
			if config != nil {
				return nil, nil, fmt.Errorf("multiple @StateFlowV2 annotations found")
			}
			config = cfg
			continue
		}

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

func annotationParamValue(match []string) string {
	if match[2] != "" {
		return match[2]
	}
	if match[3] != "" {
		return match[3]
	}
	return match[4]
}

func parseStatusValues(raw string) (map[string]string, error) {
	values := make(map[string]string)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items := strings.SplitN(part, "=", 2)
		if len(items) != 2 {
			return nil, fmt.Errorf("invalid statusValues item: %s", part)
		}
		key := strings.TrimSpace(items[0])
		value := strings.TrimSpace(items[1])
		if key == "" || value == "" {
			return nil, fmt.Errorf("invalid statusValues item: %s", part)
		}
		values[key] = value
	}
	return values, nil
}

func normalizeAnnotationLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "//")
	line = strings.TrimPrefix(line, "/*")
	line = strings.TrimSuffix(line, "*/")
	return strings.TrimSpace(line)
}

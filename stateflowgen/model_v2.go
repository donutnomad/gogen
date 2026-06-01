package stateflowgen

import "fmt"

type StateFlowV2Model struct {
	Name                string
	StatusType          string
	StatusValues        map[string]string
	Statuses            []string
	Stages              []string
	HasApproval         bool
	HasOptionalApproval bool
	Transitions         []TransitionV2
	InitStatus          string
}

type TransitionV2 struct {
	From             string
	To               string
	ApprovalRequired bool
	ApprovalOptional bool
	Via              string
	Fallback         string
}

func BuildStateFlowV2Model(config *StateFlowV2Config, rules []*FlowRule) (*StateFlowV2Model, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("no flow rules defined")
	}

	model := &StateFlowV2Model{
		Name:         config.Name,
		StatusType:   config.StatusType,
		StatusValues: config.StatusValues,
		Stages:       []string{"none"},
	}
	if model.StatusType == "" {
		model.StatusType = "string"
	}

	for _, rule := range rules {
		if rule.Source.Phase == "" || rule.Source.Status != "" || rule.Source.Wildcard {
			return nil, fmt.Errorf("StateFlowV2 source must be a stable status phase: phase=%s status=%s wildcard=%v", rule.Source.Phase, rule.Source.Status, rule.Source.Wildcard)
		}
		addOrderedString(&model.Statuses, rule.Source.Phase)
		if model.InitStatus == "" {
			model.InitStatus = rule.Source.Phase
		}

		for _, target := range rule.Targets {
			to := target.Phase
			if target.Self {
				to = rule.Source.Phase
			}
			if to == "" || target.Status != "" {
				return nil, fmt.Errorf("StateFlowV2 target must be a stable status phase")
			}
			addOrderedString(&model.Statuses, to)

			fallback := rule.Source.Phase
			if target.Else != "" {
				fallback = target.Else
				addOrderedString(&model.Statuses, fallback)
			}

			if target.ApprovalRequired || target.ApprovalOptional {
				if target.Via == "" || target.ViaStatus != "" {
					return nil, fmt.Errorf("StateFlowV2 approval transition requires via stage")
				}
				addOrderedString(&model.Stages, target.Via)
				model.HasApproval = true
			}
			if target.ApprovalOptional {
				model.HasOptionalApproval = true
			}

			model.Transitions = append(model.Transitions, TransitionV2{
				From:             rule.Source.Phase,
				To:               to,
				ApprovalRequired: target.ApprovalRequired,
				ApprovalOptional: target.ApprovalOptional,
				Via:              target.Via,
				Fallback:         fallback,
			})
		}
	}

	if len(model.Statuses) == 0 {
		return nil, fmt.Errorf("no statuses defined")
	}
	return model, nil
}

func addOrderedString(values *[]string, value string) {
	for _, item := range *values {
		if item == value {
			return
		}
	}
	*values = append(*values, value)
}

package stateflowgen

import (
	"fmt"
	"sort"
)

// StateModel 完整状态模型
type StateModel struct {
	Name        string              // 类型前缀
	Phases      []string            // 所有 Phase（保持定义顺序）
	PhaseStatus map[string][]string // Phase -> Status 列表
	HasStatus   bool                // 是否有任何 Status 定义
	HasApproval bool                // 是否有任何审批标记
	Transitions []Transition        // 展开后的所有流转
	InitStage   Stage               // 初始阶段
	ViaPhases   []string            // via 中间状态列表
}

// Stage 阶段（Phase + Status）
type Stage struct {
	Phase  string
	Status string
}

// String 返回阶段的字符串表示
func (s Stage) String() string {
	if s.Status == "" {
		return s.Phase
	}
	return fmt.Sprintf("%s(%s)", s.Phase, s.Status)
}

// Equal 比较两个阶段是否相等
func (s Stage) Equal(other Stage) bool {
	return s.Phase == other.Phase && s.Status == other.Status
}

// Transition 展开后的单条流转
type Transition struct {
	From             Stage // 源阶段
	To               Stage // 目标阶段
	ApprovalRequired bool  // ! 标记
	ApprovalOptional bool  // ? 标记
	Via              Stage // via 中间阶段（审批时）
	Fallback         Stage // else 拒绝后阶段（为空则等于 From）
}

// BuildModel 从配置和规则构建状态模型
func BuildModel(config *StateFlowConfig, rules []*FlowRule) (*StateModel, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("no flow rules defined")
	}

	model := &StateModel{
		Name:        config.Name,
		PhaseStatus: make(map[string][]string),
	}

	// 第一遍：收集所有 Phase 和 Status
	phaseSet := make(map[string]bool)
	phaseOrder := []string{}
	statusSet := make(map[string]map[string]bool) // phase -> status set
	viaSet := make(map[string]bool)

	for _, rule := range rules {
		// 处理源状态
		if rule.Source.Phase != "" {
			if !phaseSet[rule.Source.Phase] {
				phaseSet[rule.Source.Phase] = true
				phaseOrder = append(phaseOrder, rule.Source.Phase)
			}
			if rule.Source.Status != "" && !rule.Source.Wildcard {
				if statusSet[rule.Source.Phase] == nil {
					statusSet[rule.Source.Phase] = make(map[string]bool)
				}
				statusSet[rule.Source.Phase][rule.Source.Status] = true
			}
		}

		// 处理目标状态
		for _, target := range rule.Targets {
			if target.Phase != "" && !target.Self {
				if !phaseSet[target.Phase] {
					phaseSet[target.Phase] = true
					phaseOrder = append(phaseOrder, target.Phase)
				}
			}
			if target.Status != "" {
				phase := target.Phase
				if phase == "" {
					phase = rule.Source.Phase
				}
				if statusSet[phase] == nil {
					statusSet[phase] = make(map[string]bool)
				}
				statusSet[phase][target.Status] = true
			}

			// 处理 via 状态
			if target.Via != "" {
				if !phaseSet[target.Via] {
					phaseSet[target.Via] = true
					phaseOrder = append(phaseOrder, target.Via)
				}
				viaSet[target.Via] = true
				if target.ViaStatus != "" {
					if statusSet[target.Via] == nil {
						statusSet[target.Via] = make(map[string]bool)
					}
					statusSet[target.Via][target.ViaStatus] = true
				}
			}

			// 处理 else 状态
			if target.Else != "" {
				if !phaseSet[target.Else] {
					phaseSet[target.Else] = true
					phaseOrder = append(phaseOrder, target.Else)
				}
				if target.ElseStatus != "" {
					if statusSet[target.Else] == nil {
						statusSet[target.Else] = make(map[string]bool)
					}
					statusSet[target.Else][target.ElseStatus] = true
				}
			}

			// 检查审批标记
			if target.ApprovalRequired || target.ApprovalOptional {
				model.HasApproval = true
			}
		}
	}

	// 设置 Phases
	model.Phases = phaseOrder

	// 设置 PhaseStatus
	for phase, statuses := range statusSet {
		var statusList []string
		for status := range statuses {
			statusList = append(statusList, status)
		}
		sort.Strings(statusList)
		model.PhaseStatus[phase] = statusList
		if len(statusList) > 0 {
			model.HasStatus = true
		}
	}

	// 设置 ViaPhases
	for via := range viaSet {
		model.ViaPhases = append(model.ViaPhases, via)
	}
	sort.Strings(model.ViaPhases)

	// 第二遍：展开通配符并生成 Transitions
	for _, rule := range rules {
		transitions, err := expandRule(rule, model.PhaseStatus)
		if err != nil {
			return nil, err
		}
		model.Transitions = append(model.Transitions, transitions...)
	}

	// 设置初始状态（第一条规则的源状态）
	if len(rules) > 0 && rules[0].Source.Phase != "" {
		model.InitStage = Stage{
			Phase:  rules[0].Source.Phase,
			Status: rules[0].Source.Status,
		}
	}

	// 验证模型
	if err := validateModel(model); err != nil {
		return nil, err
	}

	return model, nil
}

// expandRule 展开单条规则，处理通配符
func expandRule(rule *FlowRule, phaseStatus map[string][]string) ([]Transition, error) {
	var transitions []Transition

	// 获取源状态列表（展开通配符）
	sourceStages := expandSourceStage(rule.Source, phaseStatus)

	for _, source := range sourceStages {
		for _, target := range rule.Targets {
			// 验证：有审批标记必须有 via
			if (target.ApprovalRequired || target.ApprovalOptional) && target.Via == "" {
				targetDesc := target.Phase
				if targetDesc == "" {
					targetDesc = fmt.Sprintf("(%s)", target.Status)
				}
				mark := "!"
				if target.ApprovalOptional {
					mark = "?"
				}
				return nil, fmt.Errorf("approval mark '%s' requires 'via' keyword: %s%s", mark, targetDesc, mark)
			}

			// 计算目标状态
			toStage := Stage{}
			if target.Self {
				toStage = source
			} else {
				toStage.Phase = target.Phase
				if toStage.Phase == "" {
					toStage.Phase = source.Phase
				}
				toStage.Status = target.Status
			}

			// 跳过自我流转（通配符展开时）
			if rule.Source.Wildcard && toStage.Equal(source) && !target.Self {
				continue
			}

			trans := Transition{
				From:             source,
				To:               toStage,
				ApprovalRequired: target.ApprovalRequired,
				ApprovalOptional: target.ApprovalOptional,
			}

			// 设置 via 状态
			if target.Via != "" {
				trans.Via = Stage{
					Phase:  target.Via,
					Status: target.ViaStatus,
				}
			}

			// 设置 fallback 状态
			if target.Else != "" {
				trans.Fallback = Stage{
					Phase:  target.Else,
					Status: target.ElseStatus,
				}
			} else if target.ApprovalRequired || target.ApprovalOptional {
				// 默认回退到源状态
				trans.Fallback = source
			}

			transitions = append(transitions, trans)
		}
	}

	return transitions, nil
}

// expandSourceStage 展开源状态，处理通配符
func expandSourceStage(source StateRef, phaseStatus map[string][]string) []Stage {
	if !source.Wildcard {
		return []Stage{{Phase: source.Phase, Status: source.Status}}
	}

	// 通配符展开
	statuses := phaseStatus[source.Phase]
	if len(statuses) == 0 {
		return []Stage{{Phase: source.Phase}}
	}

	var stages []Stage
	for _, status := range statuses {
		stages = append(stages, Stage{Phase: source.Phase, Status: status})
	}
	return stages
}

// validateModel 验证模型的有效性
func validateModel(model *StateModel) error {
	if len(model.Phases) == 0 {
		return fmt.Errorf("no phases defined")
	}

	// 检查是否有流转（单节点声明时允许无流转）
	if len(model.Transitions) == 0 && len(model.Phases) > 1 {
		return fmt.Errorf("multiple phases defined but no transitions")
	}

	// 单节点或无流转时不需要进一步验证
	if len(model.Transitions) == 0 {
		return nil
	}

	// 收集所有边
	inEdges := make(map[string]bool)  // 有入边的阶段
	outEdges := make(map[string]bool) // 有出边的阶段
	viaPhases := make(map[string]bool)
	fallbackTargets := make(map[string]bool) // else/fallback 目标

	for _, via := range model.ViaPhases {
		viaPhases[via] = true
	}

	for _, trans := range model.Transitions {
		fromKey := trans.From.String()
		toKey := trans.To.String()
		outEdges[fromKey] = true
		inEdges[toKey] = true

		// via 状态也计入
		if trans.Via.Phase != "" {
			viaKey := trans.Via.String()
			inEdges[viaKey] = true
			outEdges[viaKey] = true
		}

		// fallback 目标也计入（作为潜在的入边）
		if trans.Fallback.Phase != "" {
			fallbackKey := trans.Fallback.String()
			fallbackTargets[fallbackKey] = true
		}
	}

	// 使用 BFS 检测连通性
	// 从初始状态开始，检查所有节点是否可达
	reachable := make(map[string]bool)

	// 构建邻接表
	adjList := make(map[string][]string)

	for _, trans := range model.Transitions {
		fromKey := trans.From.String()
		toKey := trans.To.String()
		adjList[fromKey] = append(adjList[fromKey], toKey)

		if trans.Via.Phase != "" {
			viaKey := trans.Via.String()
			adjList[fromKey] = append(adjList[fromKey], viaKey)
			adjList[viaKey] = append(adjList[viaKey], toKey)
		}

		if trans.Fallback.Phase != "" {
			fallbackKey := trans.Fallback.String()
			adjList[fromKey] = append(adjList[fromKey], fallbackKey)
		}
	}

	// BFS 从初始状态开始
	initKey := model.InitStage.String()
	queue := []string{initKey}
	reachable[initKey] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, next := range adjList[current] {
			if !reachable[next] {
				reachable[next] = true
				queue = append(queue, next)
			}
		}
	}

	// 反向 BFS 检查是否有路径到达终态或可从初始状态到达
	// 简化：只检查是否有节点既不可达也不在图中
	allNodes := make(map[string]bool)
	for _, phase := range model.Phases {
		if viaPhases[phase] {
			continue
		}
		statuses := model.PhaseStatus[phase]
		if len(statuses) == 0 {
			allNodes[phase] = true
		} else {
			for _, status := range statuses {
				allNodes[fmt.Sprintf("%s(%s)", phase, status)] = true
			}
		}
	}

	// 检查是否有节点无法从初始状态到达
	for node := range allNodes {
		hasInEdge := inEdges[node] || fallbackTargets[node]
		hasOutEdge := outEdges[node]
		isReachable := reachable[node]

		// 起始态：只有出边，无入边 - 允许（就是初始状态）
		// 终态：只有入边，无出边，但必须从初始状态可达 - 允许
		// 中间态：有入边也有出边，但必须从初始状态可达 - 允许
		// 孤立节点：无入边也无出边 - 错误
		// 断开的子图：有边但无法从初始状态到达 - 错误

		if !hasInEdge && !hasOutEdge {
			return fmt.Errorf("isolated node detected: %s", node)
		}

		// 如果节点有出边但不可达，则是断开的子图
		if !isReachable && hasOutEdge {
			return fmt.Errorf("disconnected subgraph detected at: %s (not reachable from initial state)", node)
		}

		// 如果节点只有入边但不可达（这不应该发生，但以防万一）
		if !isReachable && hasInEdge && !hasOutEdge {
			return fmt.Errorf("unreachable terminal state: %s", node)
		}
	}

	return nil
}

// GetAllStages 获取所有有效的阶段组合
func (m *StateModel) GetAllStages() []Stage {
	var stages []Stage

	for _, phase := range m.Phases {
		statuses := m.PhaseStatus[phase]
		if len(statuses) == 0 {
			stages = append(stages, Stage{Phase: phase})
		} else {
			for _, status := range statuses {
				stages = append(stages, Stage{Phase: phase, Status: status})
			}
		}
	}

	return stages
}

// GetTransitionsFrom 获取从指定阶段出发的所有流转
func (m *StateModel) GetTransitionsFrom(from Stage) []Transition {
	var transitions []Transition
	for _, trans := range m.Transitions {
		if trans.From.Equal(from) {
			transitions = append(transitions, trans)
		}
	}
	return transitions
}

// GetValidTargets 获取从指定阶段可以流转到的所有目标
func (m *StateModel) GetValidTargets(from Stage) []Stage {
	var targets []Stage
	seen := make(map[string]bool)

	for _, trans := range m.GetTransitionsFrom(from) {
		key := trans.To.String()
		if !seen[key] {
			seen[key] = true
			targets = append(targets, trans.To)
		}
	}

	return targets
}

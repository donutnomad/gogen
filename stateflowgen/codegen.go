package stateflowgen

import (
	"fmt"
	"strings"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/internal/utils"
)

type CodeGenerator struct {
	model *StateModel
	gen   *gg.Generator
}

func NewCodeGenerator(model *StateModel, packageName string) *CodeGenerator {
	gen := gg.New()
	gen.SetPackage(packageName)
	return &CodeGenerator{
		model: model,
		gen:   gen,
	}
}

func (c *CodeGenerator) Generate() (*gg.Generator, error) {
	group := c.gen.Body()

	// 生成流程图注释
	c.generateFlowDiagram(group)

	// 生成 Phase 枚举
	c.generatePhaseEnum(group)

	// 生成 Status 枚举（如果有）
	if c.model.HasStatus {
		c.generateStatusEnum(group)
	}

	// 生成 Stage 结构体
	c.generateStageType(group)

	// 生成预定义阶段变量
	c.generateStageVars(group)

	// 如果有流转，生成 State 和相关方法
	if len(c.model.Transitions) > 0 {
		// 生成审批相关类型（如果有）
		if c.model.HasApproval {
			c.generatePendingTransitionType(group)
		}

		// 生成 State 结构体
		c.generateStateType(group)

		// 生成 StateColumns 结构体（用于数据库存储）
		c.generateStateColumnsType(group)

		// 生成 State <-> StateColumns 转换方法
		c.generateToColumnsMethod(group)
		c.generateFromColumnsMethod(group)

		// 生成错误变量
		c.generateErrors(group)

		// 生成核心方法
		c.generateTransitionMethod(group)
		if c.model.HasApproval {
			c.generateCommitMethod(group)
			c.generateRejectMethod(group)
			c.generateIsApprovalPendingMethod(group)
		}
		c.generateValidTransitionsMethod(group)
	}

	return c.gen, nil
}

// generatePhaseEnum 生成 Phase 枚举
func (c *CodeGenerator) generatePhaseEnum(group *gg.Group) {
	typeName := c.model.Name + "Phase"

	group.AddLine()
	group.Append(gg.LineComment("%s 阶段枚举", typeName))
	group.Append(gg.Type(typeName, "string"))

	group.AddLine()

	// 生成常量
	constGroup := gg.Const()
	for _, phase := range c.model.Phases {
		constName := typeName + utils.UpperCamelCase(phase)          // 标识符首字母大写
		constGroup.AddTypedField(constName, typeName, gg.Lit(phase)) // 字符串值保持原样
	}
	group.Append(constGroup)

	// 生成枚举聚合变量
	c.generateEnumAggregateVar(group, typeName, c.model.Phases)
}

// generateStatusEnum 生成 Status 枚举
func (c *CodeGenerator) generateStatusEnum(group *gg.Group) {
	typeName := c.model.Name + "Status"

	group.AddLine()
	group.Append(gg.LineComment("%s 状态枚举", typeName))
	group.Append(gg.Type(typeName, "string"))

	group.AddLine()

	// 收集所有 Status（保持顺序）
	var statusList []string
	statusSet := make(map[string]bool)
	for _, statuses := range c.model.PhaseStatus {
		for _, status := range statuses {
			if !statusSet[status] {
				statusSet[status] = true
				statusList = append(statusList, status)
			}
		}
	}

	// 生成常量
	constGroup := gg.Const()
	constGroup.AddTypedField(typeName+"None", typeName, gg.Lit(""))
	for _, status := range statusList {
		constName := typeName + utils.UpperCamelCase(status)          // 标识符首字母大写
		constGroup.AddTypedField(constName, typeName, gg.Lit(status)) // 字符串值保持原样
	}
	group.Append(constGroup)

	// 生成枚举聚合变量（包含 None）
	allStatuses := append([]string{"None"}, statusList...)
	c.generateEnumAggregateVar(group, typeName, allStatuses)
}

// generateEnumAggregateVar 生成枚举聚合变量
// 例如: var ArticlePhaseEnums = struct { Draft ArticlePhase; ... }{ Draft: ArticlePhaseDraft, ... }
func (c *CodeGenerator) generateEnumAggregateVar(group *gg.Group, typeName string, values []string) {
	if len(values) == 0 {
		return
	}

	varName := typeName + "Enums"

	// 构建结构体字段定义
	var structFields []string
	for _, v := range values {
		fieldName := utils.UpperCamelCase(v)
		structFields = append(structFields, fmt.Sprintf("\t%s %s", fieldName, typeName))
	}

	// 构建结构体字面量
	var literalFields []string
	for _, v := range values {
		fieldName := utils.UpperCamelCase(v)
		constName := typeName + fieldName
		literalFields = append(literalFields, fmt.Sprintf("\t%s: %s,", fieldName, constName))
	}

	// 生成完整的变量声明
	group.AddLine()
	group.Append(gg.S("var %s = struct {\n%s\n}{\n%s\n}",
		varName,
		strings.Join(structFields, "\n"),
		strings.Join(literalFields, "\n"),
	))
}

// generateStageType 生成 Stage 结构体
func (c *CodeGenerator) generateStageType(group *gg.Group) {
	typeName := c.model.Name + "Stage"
	phaseType := c.model.Name + "Phase"

	group.AddLine()
	group.Append(gg.LineComment("%s 阶段（Phase + Status）", typeName))

	if !c.model.HasStatus {
		// 无 Status 时，Stage 是 Phase 的别名
		group.Append(gg.TypeAlias(typeName, phaseType))
	} else {
		statusType := c.model.Name + "Status"
		st := gg.Struct(typeName)
		st.AddField("Phase", fmt.Sprintf("%s `json:\"phase\"`", phaseType))
		st.AddField("Status", fmt.Sprintf("%s `json:\"status\"`", statusType))
		group.Append(st)
	}
}

// generateStageVars 生成预定义阶段变量
func (c *CodeGenerator) generateStageVars(group *gg.Group) {
	stageType := c.model.Name + "Stage"
	phaseType := c.model.Name + "Phase"
	statusType := c.model.Name + "Status"

	group.AddLine()
	group.Append(gg.LineComment("预定义阶段"))

	varGroup := gg.Var()
	for _, stage := range c.model.GetAllStages() {
		varName := "Stage" + c.model.Name + utils.UpperCamelCase(stage.Phase)
		if stage.Status != "" {
			varName += utils.UpperCamelCase(stage.Status)
		}

		if !c.model.HasStatus {
			// 无 Status 时
			varGroup.AddField(varName, gg.S("%s%s", phaseType, utils.UpperCamelCase(stage.Phase)))
		} else {
			// 有 Status 时
			statusConst := statusType + "None"
			if stage.Status != "" {
				statusConst = statusType + utils.UpperCamelCase(stage.Status)
			}
			varGroup.AddField(varName, gg.S("%s{%s%s, %s}", stageType, phaseType, utils.UpperCamelCase(stage.Phase), statusConst))
		}
	}
	group.Append(varGroup)
}

// generatePendingTransitionType 生成审批事务类型
func (c *CodeGenerator) generatePendingTransitionType(group *gg.Group) {
	typeName := c.model.Name + "PendingTransition"
	stageType := c.model.Name + "Stage"

	group.AddLine()
	group.Append(gg.LineComment("%s 审批事务", typeName))

	st := gg.Struct(typeName)
	st.AddField("From", fmt.Sprintf("%s `json:\"from\"`", stageType))
	st.AddField("To", fmt.Sprintf("%s `json:\"to\"`", stageType))
	st.AddField("Fallback", fmt.Sprintf("%s `json:\"fallback\"`", stageType))
	group.Append(st)
}

// generateStateType 生成 State 结构体
func (c *CodeGenerator) generateStateType(group *gg.Group) {
	typeName := c.model.Name + "State"
	stageType := c.model.Name + "Stage"

	group.AddLine()
	group.Append(gg.LineComment("%s 完整状态", typeName))

	st := gg.Struct(typeName)
	st.AddField("Current", fmt.Sprintf("%s `json:\"current\"`", stageType))

	if c.model.HasApproval {
		pendingType := "*" + c.model.Name + "PendingTransition"
		st.AddField("Pending", fmt.Sprintf("%s `json:\"pending,omitempty\"`", pendingType))
	}
	group.Append(st)
}

// generateErrors 生成错误变量
func (c *CodeGenerator) generateErrors(group *gg.Group) {
	errorsP := c.gen.P("errors")

	group.AddLine()
	group.Append(gg.LineComment("错误定义"))

	varGroup := gg.Var()
	varGroup.AddField(
		"Err"+c.model.Name+"InvalidTransition",
		errorsP.Call("New", gg.Lit("invalid transition")),
	)
	if c.model.HasApproval {
		varGroup.AddField(
			"Err"+c.model.Name+"ApprovalInProgress",
			errorsP.Call("New", gg.Lit("approval in progress")),
		)
		varGroup.AddField(
			"Err"+c.model.Name+"NotInApproval",
			errorsP.Call("New", gg.Lit("not in approval")),
		)
	}
	group.Append(varGroup)
}

// generateTransitionMethod 生成 TransitionTo 方法
func (c *CodeGenerator) generateTransitionMethod(group *gg.Group) {
	stateType := c.model.Name + "State"
	stageType := c.model.Name + "Stage"

	group.AddLine()

	// 方法签名
	fn := gg.Function("TransitionTo").
		WithReceiver("s", stateType).
		AddParameter("to", stageType)

	if c.model.HasApproval {
		fn.AddParameter("withApproval", "bool")
	}

	fn.AddResult("", stateType).
		AddResult("", "error")

	// 方法体
	var body []any

	// 注意：不在此处统一检查 Pending
	// 只有需要审批的流转才需要检查 Pending，这在各个 case 分支中处理
	// 直接流转（无 ! 或 ? 标记）即使有 Pending 也应该允许执行

	// 生成流转规则检查
	body = append(body, c.generateTransitionRulesSwitch()...)

	fn.AddBody(body...)
	group.Append(fn)
}

// generateTransitionRulesSwitch 生成流转规则的 switch 语句
func (c *CodeGenerator) generateTransitionRulesSwitch() []any {
	// 按源状态分组
	rulesByFrom := make(map[string][]Transition)
	for _, trans := range c.model.Transitions {
		key := trans.From.String()
		rulesByFrom[key] = append(rulesByFrom[key], trans)
	}

	stateType := c.model.Name + "State"
	pendingType := c.model.Name + "PendingTransition"

	// 构建外层 switch
	outerSwitch := gg.Switch("s.Current")

	for _, stage := range c.model.GetAllStages() {
		key := stage.String()
		rules, ok := rulesByFrom[key]
		if !ok || len(rules) == 0 {
			continue
		}

		// case 语句
		varName := c.getStageVarName(stage)

		// 内层 switch
		innerSwitch := gg.Switch("to")

		for _, rule := range rules {
			toVarName := c.getStageVarName(rule.To)

			var caseBody []any

			if c.model.HasApproval && (rule.ApprovalRequired || rule.ApprovalOptional) && rule.Via.Phase != "" {
				// 需要审批的流转：先检查是否有进行中的审批
				viaVarName := c.getStageVarName(rule.Via)
				fallbackVarName := c.getStageVarName(rule.Fallback)

				// 在创建新的 Pending 之前，必须检查是否已有进行中的审批
				pendingCheck := gg.If("s.Pending != nil").AddBody(
					gg.S("return s, Err%sApprovalInProgress", c.model.Name),
				)

				if rule.ApprovalRequired {
					// 必须审批
					caseBody = append(caseBody,
						pendingCheck,
						gg.S("return %s{Current: %s, Pending: &%s{From: s.Current, To: to, Fallback: %s}}, nil",
							stateType, viaVarName, pendingType, fallbackVarName))
				} else {
					// 可选审批
					caseBody = append(caseBody,
						gg.If("withApproval").AddBody(
							pendingCheck,
							gg.S("return %s{Current: %s, Pending: &%s{From: s.Current, To: to, Fallback: %s}}, nil",
								stateType, viaVarName, pendingType, fallbackVarName),
						),
						gg.S("return %s{Current: to}, nil", stateType),
					)
				}
			} else {
				// 直接流转：不需要检查 Pending，直接执行
				caseBody = append(caseBody, gg.S("return %s{Current: to}, nil", stateType))
			}

			innerSwitch.NewCase(gg.S(toVarName)).AddBody(caseBody...)
		}

		outerSwitch.NewCase(gg.S(varName)).AddBody(innerSwitch)
	}

	return []any{
		outerSwitch,
		gg.S("return s, Err%sInvalidTransition", c.model.Name),
	}
}

// getStageVarName 获取阶段变量名
func (c *CodeGenerator) getStageVarName(stage Stage) string {
	varName := "Stage" + c.model.Name + utils.UpperCamelCase(stage.Phase)
	if stage.Status != "" {
		varName += utils.UpperCamelCase(stage.Status)
	}
	return varName
}

// generateCommitMethod 生成 Commit 方法
func (c *CodeGenerator) generateCommitMethod(group *gg.Group) {
	stateType := c.model.Name + "State"

	group.AddLine()

	fn := gg.Function("Commit").
		WithReceiver("s", stateType).
		AddResult("", stateType).
		AddResult("", "error").
		AddBody(
			gg.If("s.Pending == nil").AddBody(
				gg.S("return s, Err%sNotInApproval", c.model.Name),
			),
			gg.S("return %s{Current: s.Pending.To}, nil", stateType),
		)
	group.Append(fn)
}

// generateRejectMethod 生成 Reject 方法
func (c *CodeGenerator) generateRejectMethod(group *gg.Group) {
	stateType := c.model.Name + "State"

	group.AddLine()

	fn := gg.Function("Reject").
		WithReceiver("s", stateType).
		AddResult("", stateType).
		AddResult("", "error").
		AddBody(
			gg.If("s.Pending == nil").AddBody(
				gg.S("return s, Err%sNotInApproval", c.model.Name),
			),
			gg.S("return %s{Current: s.Pending.Fallback}, nil", stateType),
		)
	group.Append(fn)
}

// generateIsApprovalPendingMethod 生成 IsApprovalPending 方法
func (c *CodeGenerator) generateIsApprovalPendingMethod(group *gg.Group) {
	stateType := c.model.Name + "State"

	group.AddLine()

	fn := gg.Function("IsApprovalPending").
		WithReceiver("s", stateType).
		AddResult("", "bool").
		AddBody(gg.S("return s.Pending != nil"))
	group.Append(fn)
}

// generateValidTransitionsMethod 生成 ValidTransitions 方法
func (c *CodeGenerator) generateValidTransitionsMethod(group *gg.Group) {
	stateType := c.model.Name + "State"
	stageType := c.model.Name + "Stage"

	// 按源状态分组
	rulesByFrom := make(map[string][]Stage)
	seen := make(map[string]map[string]bool)
	for _, trans := range c.model.Transitions {
		key := trans.From.String()
		toKey := trans.To.String()
		if seen[key] == nil {
			seen[key] = make(map[string]bool)
		}
		if !seen[key][toKey] {
			seen[key][toKey] = true
			rulesByFrom[key] = append(rulesByFrom[key], trans.To)
		}
	}

	group.AddLine()

	fn := gg.Function("ValidTransitions").
		WithReceiver("s", stateType).
		AddResult("", fmt.Sprintf("[]%s", stageType))

	// 构建 switch
	sw := gg.Switch("s.Current")

	for _, stage := range c.model.GetAllStages() {
		key := stage.String()
		targets, ok := rulesByFrom[key]
		if !ok || len(targets) == 0 {
			continue
		}

		varName := c.getStageVarName(stage)

		var targetVars []string
		for _, t := range targets {
			targetVars = append(targetVars, c.getStageVarName(t))
		}

		sw.NewCase(gg.S(varName)).AddBody(gg.S("return []%s{%s}", stageType, strings.Join(targetVars, ", ")))
	}

	fn.AddBody(sw, gg.S("return nil"))
	group.Append(fn)
}

// generateStateColumnsType 生成用于数据库存储的 StateColumns 结构体
func (c *CodeGenerator) generateStateColumnsType(group *gg.Group) {
	typeName := c.model.Name + "StateColumns"
	phaseType := c.model.Name + "Phase"

	group.AddLine()
	group.Append(gg.LineComment("%s 数据库存储结构", typeName))

	st := gg.Struct(typeName)
	st.AddField("Phase", fmt.Sprintf("%s `gorm:\"column:phase\" json:\"phase\"`", phaseType))

	if c.model.HasStatus {
		statusType := c.model.Name + "Status"
		st.AddField("Status", fmt.Sprintf("%s `gorm:\"column:status\" json:\"status\"`", statusType))
	}

	if c.model.HasApproval {
		datatypesP := c.gen.P("gorm.io/datatypes")
		pendingType := c.model.Name + "PendingTransition"
		st.AddField("Pending", gg.NewInlineGroup().Append(
			datatypesP.Type("JSONType"),
			gg.S("[*%s] `gorm:\"column:pending\" json:\"pending\"`", pendingType),
		))
	}

	group.Append(st)
}

// generateToColumnsMethod 生成 State.ToColumns() 方法
func (c *CodeGenerator) generateToColumnsMethod(group *gg.Group) {
	stateType := c.model.Name + "State"
	columnsType := c.model.Name + "StateColumns"

	group.AddLine()

	fn := gg.Function("ToColumns").
		WithReceiver("s", stateType).
		AddResult("", columnsType)

	var bodyParts []string

	if c.model.HasStatus {
		bodyParts = append(bodyParts, "Phase: s.Current.Phase,")
		bodyParts = append(bodyParts, "Status: s.Current.Status,")
	} else {
		// 无 Status 时，Stage 是 Phase 的别名，Current 本身就是 Phase
		bodyParts = append(bodyParts, "Phase: s.Current,")
	}

	if c.model.HasApproval {
		c.gen.P("gorm.io/datatypes")
		bodyParts = append(bodyParts, "Pending: datatypes.NewJSONType(s.Pending),")
	}

	fn.AddBody(gg.S("return %s{\n\t%s\n}", columnsType, strings.Join(bodyParts, "\n\t")))
	group.Append(fn)
}

// generateFromColumnsMethod 生成 StateColumns.ToState() 方法
func (c *CodeGenerator) generateFromColumnsMethod(group *gg.Group) {
	stateType := c.model.Name + "State"
	columnsType := c.model.Name + "StateColumns"
	stageType := c.model.Name + "Stage"

	group.AddLine()

	fn := gg.Function("ToState").
		WithReceiver("c", columnsType).
		AddResult("", stateType)

	var currentExpr string
	if c.model.HasStatus {
		currentExpr = fmt.Sprintf("%s{Phase: c.Phase, Status: c.Status}", stageType)
	} else {
		// 无 Status 时，Stage 是 Phase 的别名
		currentExpr = "c.Phase"
	}

	var bodyParts []string
	bodyParts = append(bodyParts, fmt.Sprintf("Current: %s,", currentExpr))

	if c.model.HasApproval {
		bodyParts = append(bodyParts, "Pending: c.Pending.Data(),")
	}

	fn.AddBody(gg.S("return %s{\n\t%s\n}", stateType, strings.Join(bodyParts, "\n\t")))
	group.Append(fn)
}

// generateFlowDiagram 生成流程图注释
func (c *CodeGenerator) generateFlowDiagram(group *gg.Group) {
	if len(c.model.Transitions) == 0 {
		return
	}

	renderer := NewDiagramRenderer()
	renderer.ArrowSymbol = "──"

	// 收集可选审批的状态，用于自定义分支符号
	for _, trans := range c.model.Transitions {
		fromStr := c.formatStage(trans.From)
		toStr := c.formatStage(trans.To)

		if trans.ApprovalOptional && trans.Via.Phase != "" {
			// 可选审批：创建中间判别节点
			// from ──▶ 🔶<APPROVAL?> ──┬──▶ via (via) ──┬── <COMMIT> ──▶ to
			//                          │                └── <REJECT> ──▶ fallback
			//                          │
			//                          └──▶ to (直接)
			viaStr := c.formatStage(trans.Via)
			fallbackStr := c.formatStage(trans.Fallback)

			// 创建中间判别节点
			decisionNode := fromStr + "_decision"
			renderer.AddNode(decisionNode, "<?APPROVAL?>")
			renderer.AddEdge(fromStr, decisionNode, "──▶ ")

			// 1. 审批路径：decision -> via -> (Commit/Reject)
			// 如果 via 和 from 相同，使用 shadow 名字避免冲突
			viaNodeID := viaStr
			if viaStr == fromStr {
				viaNodeID = viaStr + "_via"
			}
			renderer.AddNode(viaNodeID, viaStr+" (via)")
			renderer.AddEdge(decisionNode, viaNodeID, "──▶ ")

			// via 分叉出 Commit 和 Reject
			if toStr != "" {
				renderer.AddEdge(viaNodeID, toStr, "── <COMMIT> ──▶ ")
			}
			if fallbackStr != "" {
				// 如果 fallback 和 from 相同，使用 shadow 节点显示回到原状态
				if fallbackStr == fromStr {
					fallbackNodeID := fallbackStr + "_fallback"
					renderer.AddNode(fallbackNodeID, fallbackStr+" 🔁")
					renderer.AddEdge(viaNodeID, fallbackNodeID, "── <REJECT> ──▶ ")
				} else {
					renderer.AddEdge(viaNodeID, fallbackStr, "── <REJECT> ──▶ ")
				}
			}

			// 2. 直接路径：decision -> to
			renderer.AddEdge(decisionNode, toStr, "──▶ ")
		} else if trans.Via.Phase != "" {
			// 必须审批：from -> via -> (Commit/Reject)
			//                          ┌── <COMMIT> ──▶ to
			//                          │
			// from ──▶ via (via) ──┤
			//                          │
			//                          └── <REJECT> ──▶ fallback

			viaStr := c.formatStage(trans.Via)
			toStr := c.formatStage(trans.To)
			fallbackStr := c.formatStage(trans.Fallback)

			// 为每个 from 创建独立的 via 节点，避免不同转换共用同一个 via 节点
			viaNodeID := fromStr + "_" + viaStr + "_via"

			// from -> via
			renderer.AddNode(viaNodeID, viaStr+" (via)")
			renderer.AddEdge(fromStr, viaNodeID, "──▶ ")

			// via 分叉出 Commit 和 Reject
			if toStr != "" {
				renderer.AddEdge(viaNodeID, toStr, "── <COMMIT> ──▶ ")
			}
			if fallbackStr != "" {
				// 如果 fallback 和 from 相同，使用 shadow 节点
				if fallbackStr == fromStr {
					fallbackNodeID := fallbackStr + "_fallback"
					renderer.AddNode(fallbackNodeID, fallbackStr+" 🔁")
					renderer.AddEdge(viaNodeID, fallbackNodeID, "── <REJECT> ──▶ ")
				} else {
					renderer.AddEdge(viaNodeID, fallbackStr, "── <REJECT> ──▶ ")
				}
			}
		} else {
			// 直接流转
			renderer.AddEdge(fromStr, toStr, "──▶ ")
		}
	}

	// 渲染并输出
	comment := renderer.RenderAsComment()
	if comment != "" {
		group.Append(gg.S(comment))
	}
}

// formatStage 格式化阶段显示
func (c *CodeGenerator) formatStage(stage Stage) string {
	if stage.Status != "" {
		return fmt.Sprintf("%s(%s)", stage.Phase, stage.Status)
	}
	return stage.Phase
}

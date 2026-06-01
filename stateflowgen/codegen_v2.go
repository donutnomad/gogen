package stateflowgen

import (
	"fmt"
	"strings"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/internal/utils"
)

type CodeGeneratorV2 struct {
	model *StateFlowV2Model
	gen   *gg.Generator
}

func NewCodeGeneratorV2(model *StateFlowV2Model, packageName string) *CodeGeneratorV2 {
	gen := gg.New()
	gen.SetPackage(packageName)
	return &CodeGeneratorV2{
		model: model,
		gen:   gen,
	}
}

func (c *CodeGeneratorV2) Generate() (*gg.Generator, error) {
	group := c.gen.Body()

	c.generateV2StateType(group)
	c.generateV2TransitionType(group)
	c.generateV2StateColumnsType(group)
	c.generateV2ColumnMethods(group)
	c.generateV2Errors(group)
	c.generateV2TransitionMethod(group)
	c.generateV2CommitMethod(group)
	c.generateV2RejectMethod(group)
	c.generateV2IsTransitionPendingMethod(group)
	c.generateV2ValidTransitionsMethod(group)
	c.generateV2NextMethod(group)
	c.generateV2StatusEnum(group)
	c.generateV2StatusStringMethods(group)
	c.generateV2ViaEnum(group)

	return c.gen, nil
}

func (c *CodeGeneratorV2) generateV2StatusEnum(group *gg.Group) {
	typeName := c.model.Name + "Status"

	group.AddLine()
	group.Append(gg.Type(typeName, c.model.StatusType))
	c.generateV2EnumAggregateVar(group, typeName, c.model.Statuses)
}

func (c *CodeGeneratorV2) generateV2ViaEnum(group *gg.Group) {
	typeName := c.model.Name + "Via"

	group.AddLine()
	group.Append(gg.Type(typeName, "string"))
	c.generateV2EnumAggregateVar(group, typeName, c.model.Stages)
}

func (c *CodeGeneratorV2) generateV2TransitionType(group *gg.Group) {
	typeName := c.model.Name + "Transition"
	statusType := c.model.Name + "Status"

	group.AddLine()
	st := gg.Struct(typeName)
	st.AddField("From", fmt.Sprintf("%s `json:\"from\"`", statusType))
	st.AddField("To", fmt.Sprintf("%s `json:\"to\"`", statusType))
	st.AddField("Fallback", fmt.Sprintf("%s `json:\"fallback\"`", statusType))
	group.Append(st)
}

func (c *CodeGeneratorV2) generateV2StateType(group *gg.Group) {
	stateType := c.model.Name + "State"
	statusType := c.model.Name + "Status"
	viaType := c.model.Name + "Via"
	transitionType := "*" + c.model.Name + "Transition"

	group.AddLine()
	st := gg.Struct(stateType)
	st.AddField("Status", fmt.Sprintf("%s `json:\"status\"`", statusType))
	st.AddField("Via", fmt.Sprintf("%s `json:\"via\"`", viaType))
	st.AddField("Transition", fmt.Sprintf("%s `json:\"transition,omitempty\"`", transitionType))
	group.Append(st)
}

func (c *CodeGeneratorV2) generateV2StateColumnsType(group *gg.Group) {
	columnsType := c.model.Name + "StateColumns"
	statusType := c.model.Name + "Status"
	viaType := c.model.Name + "Via"

	group.AddLine()
	st := gg.Struct(columnsType)
	st.AddField("Status", fmt.Sprintf("%s `gorm:\"column:status\" json:\"status\"`", statusType))
	st.AddField("Via", fmt.Sprintf("%s `gorm:\"column:via\" json:\"via\"`", viaType))
	st.AddField("TransitionFrom", fmt.Sprintf("*%s `gorm:\"column:transition_from\" json:\"transition_from\"`", c.model.StatusType))
	st.AddField("TransitionTo", fmt.Sprintf("*%s `gorm:\"column:transition_to\" json:\"transition_to\"`", c.model.StatusType))
	st.AddField("TransitionFallback", fmt.Sprintf("*%s `gorm:\"column:transition_fallback\" json:\"transition_fallback\"`", c.model.StatusType))
	group.Append(st)
}

func (c *CodeGeneratorV2) generateV2ColumnMethods(group *gg.Group) {
	stateType := c.model.Name + "State"
	columnsType := c.model.Name + "StateColumns"

	group.AddLine()
	group.Append(gg.Function("New"+stateType).
		AddResult("", stateType).
		AddBody(gg.S("return %s{Status: %s, Via: %s}", stateType, c.statusValue(c.model.InitStatus), c.viaValue("none"))))

	group.AddLine()
	group.Append(gg.Function("ToColumns").
		WithReceiver("s", stateType).
		AddResult("", columnsType).
		AddBody(
			gg.S("ret := %s{Status: s.Status, Via: s.Via}", columnsType),
			gg.S("if s.Transition != nil {"),
			gg.S("	transitionFrom := %s(s.Transition.From)", c.model.StatusType),
			gg.S("	transitionTo := %s(s.Transition.To)", c.model.StatusType),
			gg.S("	transitionFallback := %s(s.Transition.Fallback)", c.model.StatusType),
			gg.S("	ret.TransitionFrom = &transitionFrom"),
			gg.S("	ret.TransitionTo = &transitionTo"),
			gg.S("	ret.TransitionFallback = &transitionFallback"),
			gg.S("}"),
			gg.S("return ret"),
		))

	group.AddLine()
	group.Append(gg.Function("ToState").
		WithReceiver("c", columnsType).
		AddResult("", stateType).
		AddBody(
			gg.S("ret := %s{Status: c.Status, Via: c.Via}", stateType),
			gg.S("if c.TransitionFrom != nil || c.TransitionTo != nil || c.TransitionFallback != nil {"),
			gg.S("	ret.Transition = &%s{From: %s(%s(c.TransitionFrom)), To: %s(%s(c.TransitionTo)), Fallback: %s(%s(c.TransitionFallback))}", c.model.Name+"Transition", c.model.Name+"Status", c.valueOrZeroFunc(), c.model.Name+"Status", c.valueOrZeroFunc(), c.model.Name+"Status", c.valueOrZeroFunc()),
			gg.S("}"),
			gg.S("return ret"),
		))

	group.AddLine()
	group.Append(gg.S(`func %s[T any](value *T) T {
	if value == nil {
		var zero T
		return zero
	}
	return *value
}`, c.valueOrZeroFunc()))
}

func (c *CodeGeneratorV2) generateV2Errors(group *gg.Group) {
	errorsP := c.gen.P("errors")

	group.AddLine()
	varGroup := gg.Var()
	varGroup.AddField("Err"+c.model.Name+"InvalidTransition", errorsP.Call("New", gg.Lit("invalid transition")))
	varGroup.AddField("Err"+c.model.Name+"ApprovalInProgress", errorsP.Call("New", gg.Lit("approval in progress")))
	varGroup.AddField("Err"+c.model.Name+"NotInApproval", errorsP.Call("New", gg.Lit("not in approval")))
	group.Append(varGroup)
}

func (c *CodeGeneratorV2) generateV2TransitionMethod(group *gg.Group) {
	stateType := c.model.Name + "State"
	statusType := c.model.Name + "Status"

	group.AddLine()
	group.Append(gg.S(`func (s %s) TransitionTo(to %s, withApproval bool) (%s, error) {
	switch s.Status {
%s
	}
	return s, Err%sInvalidTransition
}`,
		stateType,
		statusType,
		stateType,
		c.transitionCases(),
		c.model.Name,
	))
}

func (c *CodeGeneratorV2) generateV2CommitMethod(group *gg.Group) {
	stateType := c.model.Name + "State"

	group.AddLine()
	group.Append(gg.S(`func (s %s) Commit() (%s, error) {
	if s.Transition == nil {
		return s, Err%sNotInApproval
	}
	if s.Status != s.Transition.From {
		return s, Err%sInvalidTransition
	}
	return %s{Status: s.Transition.To, Via: %s}, nil
}`,
		stateType,
		stateType,
		c.model.Name,
		c.model.Name,
		stateType,
		c.viaValue("none"),
	))
}

func (c *CodeGeneratorV2) generateV2RejectMethod(group *gg.Group) {
	stateType := c.model.Name + "State"

	group.AddLine()
	group.Append(gg.S(`func (s %s) Reject() (%s, error) {
	if s.Transition == nil {
		return s, Err%sNotInApproval
	}
	if s.Status != s.Transition.From {
		return s, Err%sInvalidTransition
	}
	return %s{Status: s.Transition.Fallback, Via: %s}, nil
}`,
		stateType,
		stateType,
		c.model.Name,
		c.model.Name,
		stateType,
		c.viaValue("none"),
	))
}

func (c *CodeGeneratorV2) generateV2IsTransitionPendingMethod(group *gg.Group) {
	stateType := c.model.Name + "State"

	group.AddLine()
	group.Append(gg.S(`func (s %s) IsTransitionPending() bool {
	return s.Transition != nil
}`, stateType))
}

func (c *CodeGeneratorV2) generateV2ValidTransitionsMethod(group *gg.Group) {
	stateType := c.model.Name + "State"
	statusType := c.model.Name + "Status"

	group.AddLine()
	group.Append(gg.S(`func (s %s) ValidTransitions() []%s {
	switch s.Status {
%s
	}
	return nil
}`,
		stateType,
		statusType,
		c.validTransitionCases(),
	))
}

func (c *CodeGeneratorV2) generateV2NextMethod(group *gg.Group) {
	stateType := c.model.Name + "State"

	group.AddLine()
	group.Append(gg.S(`func (s %s) Next() []%s {
	if s.Transition != nil {
		return nil
	}
	var result []%s
	for _, status := range s.ValidTransitions() {
		result = append(result, %s{Status: status, Via: %s})
	}
	return result
}`,
		stateType,
		stateType,
		stateType,
		stateType,
		c.viaValue("none"),
	))
}

func (c *CodeGeneratorV2) generateV2StatusStringMethods(group *gg.Group) {
	statusType := c.model.Name + "Status"

	var stringCases []string
	for _, status := range c.model.Statuses {
		stringCases = append(stringCases, fmt.Sprintf("\tcase %s:", c.statusValue(status)))
		stringCases = append(stringCases, fmt.Sprintf("\t\treturn %q", status))
	}
	group.AddLine()
	group.Append(gg.S(`func (s %s) String() string {
	switch s {
%s
	default:
		return "unknown"
	}
}`, statusType, strings.Join(stringCases, "\n")))

	var parseCases []string
	for _, status := range c.model.Statuses {
		parseCases = append(parseCases, fmt.Sprintf("\tcase %q:", status))
		parseCases = append(parseCases, fmt.Sprintf("\t\treturn %s", c.statusValue(status)))
	}
	group.AddLine()
	group.Append(gg.S(`func New%s(raw string) %s {
	switch raw {
%s
	default:
		return %s
	}
}`, statusType, statusType, strings.Join(parseCases, "\n"), c.statusValue(c.model.InitStatus)))
}

func (c *CodeGeneratorV2) transitionCases() string {
	transitionsByFrom := make(map[string][]TransitionV2)
	for _, transition := range c.model.Transitions {
		transitionsByFrom[transition.From] = append(transitionsByFrom[transition.From], transition)
	}

	var lines []string
	for _, status := range c.model.Statuses {
		transitions := transitionsByFrom[status]
		if len(transitions) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("\tcase %s:", c.statusValue(status)))
		lines = append(lines, "\t\tswitch to {")
		for _, transition := range transitions {
			lines = append(lines, fmt.Sprintf("\t\tcase %s:", c.statusValue(transition.To)))
			lines = append(lines, c.transitionBody(transition)...)
		}
		lines = append(lines, "\t\t}")
	}
	return strings.Join(lines, "\n")
}

func (c *CodeGeneratorV2) transitionBody(transition TransitionV2) []string {
	stateType := c.model.Name + "State"
	if transition.ApprovalRequired || transition.ApprovalOptional {
		transitionType := c.model.Name + "Transition"
		approvalBody := []string{
			"\t\t\tif s.Transition != nil {",
			fmt.Sprintf("\t\t\t\treturn s, Err%sApprovalInProgress", c.model.Name),
			"\t\t\t}",
			fmt.Sprintf("\t\t\treturn %s{Status: s.Status, Via: %s, Transition: &%s{From: s.Status, To: to, Fallback: %s}}, nil",
				stateType,
				c.viaValue(transition.Via),
				transitionType,
				c.statusValue(transition.Fallback),
			),
		}
		if transition.ApprovalRequired {
			return approvalBody
		}
		return append(
			append([]string{"\t\t\tif withApproval {"}, indentLines(approvalBody, "\t")...),
			"\t\t\t}",
			fmt.Sprintf("\t\t\treturn %s{Status: to, Via: %s}, nil", stateType, c.viaValue("none")),
		)
	}

	return []string{fmt.Sprintf("\t\t\treturn %s{Status: to, Via: %s}, nil", stateType, c.viaValue("none"))}
}

func (c *CodeGeneratorV2) validTransitionCases() string {
	statusType := c.model.Name + "Status"
	targetsByFrom := make(map[string][]string)
	seen := make(map[string]map[string]bool)
	for _, transition := range c.model.Transitions {
		if seen[transition.From] == nil {
			seen[transition.From] = make(map[string]bool)
		}
		if seen[transition.From][transition.To] {
			continue
		}
		seen[transition.From][transition.To] = true
		targetsByFrom[transition.From] = append(targetsByFrom[transition.From], transition.To)
	}

	var lines []string
	for _, status := range c.model.Statuses {
		targets := targetsByFrom[status]
		if len(targets) == 0 {
			continue
		}
		var targetConsts []string
		for _, target := range targets {
			targetConsts = append(targetConsts, c.statusValue(target))
		}
		lines = append(lines,
			fmt.Sprintf("\tcase %s:", c.statusValue(status)),
			fmt.Sprintf("\t\treturn []%s{%s}", statusType, strings.Join(targetConsts, ", ")),
		)
	}
	return strings.Join(lines, "\n")
}

func indentLines(lines []string, prefix string) []string {
	ret := make([]string, 0, len(lines))
	for _, line := range lines {
		ret = append(ret, prefix+line)
	}
	return ret
}

func (c *CodeGeneratorV2) generateV2EnumAggregateVar(group *gg.Group, typeName string, values []string) {
	varName := typeName + "Enums"

	var structFields []string
	for _, value := range values {
		structFields = append(structFields, fmt.Sprintf("\t%s %s", utils.UpperCamelCase(value), typeName))
	}

	var literalFields []string
	for _, value := range values {
		fieldName := utils.UpperCamelCase(value)
		literalFields = append(literalFields, fmt.Sprintf("\t%s: %s,", fieldName, c.enumLiteral(typeName, value)))
	}

	group.AddLine()
	group.Append(gg.S("var %s = struct {\n%s\n}{\n%s\n}", varName, strings.Join(structFields, "\n"), strings.Join(literalFields, "\n")))
}

func (c *CodeGeneratorV2) enumLiteral(typeName, value string) string {
	if typeName == c.model.Name+"Status" {
		if literal, ok := c.model.StatusValues[value]; ok {
			return fmt.Sprintf("%s(%s)", typeName, literal)
		}
	}
	return fmt.Sprintf("%s(%q)", typeName, value)
}

func (c *CodeGeneratorV2) valueOrZeroFunc() string {
	return "valueOrZero" + c.model.Name + "State"
}

func (c *CodeGeneratorV2) statusValue(status string) string {
	return fmt.Sprintf("%sStatusEnums.%s", c.model.Name, utils.UpperCamelCase(status))
}

func (c *CodeGeneratorV2) viaValue(via string) string {
	return fmt.Sprintf("%sViaEnums.%s", c.model.Name, utils.UpperCamelCase(via))
}

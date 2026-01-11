package stateflowgen

import "strings"

// 流程图渲染符号常量
const (
	symbolArrow        = " --> "
	symbolBranch       = "+--> "
	symbolJunction     = "+"
	symbolVertical     = "|"
	symbolSpace        = " "
	symbolLoop         = " 🔁"
	symbolViaSuffix    = " (via)"
	symbolCommitPrefix = "+-- <Commit> -->"
	symbolRejectPrefix = "+-- <Reject> -->"
)

// lineBuilder 行构建辅助
type lineBuilder string

func (b lineBuilder) vertical() string {
	return string(b) + symbolVertical
}

func (b lineBuilder) verticalWith(indent, content string) string {
	return string(b) + symbolVertical + indent + content
}

func (b lineBuilder) spacedWith(indent, content string) string {
	return string(b) + symbolSpace + indent + content
}

// DiagramRenderer 流程图渲染器
type DiagramRenderer struct {
	transitions map[string][]string
	approvals   map[string]*ApprovalInfo
	order       []string
}

// ApprovalInfo 审批信息
type ApprovalInfo struct {
	Via    string
	Commit string
	Reject string
}

func NewDiagramRenderer() *DiagramRenderer {
	return &DiagramRenderer{
		transitions: make(map[string][]string),
		approvals:   make(map[string]*ApprovalInfo),
	}
}

func (r *DiagramRenderer) AddDirectTransition(from, to string) {
	if _, exists := r.transitions[from]; !exists {
		r.order = append(r.order, from)
	}
	r.transitions[from] = append(r.transitions[from], to)
}

func (r *DiagramRenderer) AddApprovalTransition(from, via, commit, reject string) {
	if _, exists := r.transitions[from]; !exists && r.approvals[from] == nil {
		r.order = append(r.order, from)
	}
	r.approvals[from] = &ApprovalInfo{
		Via:    via,
		Commit: commit,
		Reject: reject,
	}
}

func (r *DiagramRenderer) Render() string {
	if len(r.transitions) == 0 && len(r.approvals) == 0 {
		return ""
	}

	entry := r.findEntryState()
	if entry == "" {
		return ""
	}

	visited := make(map[string]bool)
	lines, _ := r.renderFlow(entry, visited)
	return strings.Join(lines, "\n")
}

func (r *DiagramRenderer) findEntryState() string {
	targets := make(map[string]bool)

	for _, tos := range r.transitions {
		for _, to := range tos {
			targets[to] = true
		}
	}

	for _, info := range r.approvals {
		targets[info.Via] = true
		targets[info.Commit] = true
		targets[info.Reject] = true
	}

	for _, src := range r.order {
		if !targets[src] {
			return src
		}
	}

	if len(r.order) > 0 {
		return r.order[0]
	}
	return ""
}

func (r *DiagramRenderer) renderFlow(state string, visited map[string]bool) ([]string, int) {
	if visited[state] {
		return []string{state + symbolLoop}, 0
	}

	if approval, ok := r.approvals[state]; ok {
		return r.renderApprovalFlow(state, approval, visited)
	}

	targets := r.transitions[state]
	if len(targets) == 0 {
		return []string{state}, 0
	}

	visited[state] = true

	if len(targets) == 1 {
		return r.renderSingleTarget(state, targets[0], visited)
	}
	return r.renderBranches(state, targets, visited)
}

func (r *DiagramRenderer) renderSingleTarget(state, target string, visited map[string]bool) ([]string, int) {
	subLines, subAnchor := r.renderFlow(target, copyVisited(visited))
	if len(subLines) == 0 {
		return []string{state}, 0
	}

	prefix := state + symbolArrow
	indent := strings.Repeat(" ", len(prefix))

	var result []string
	for i, line := range subLines {
		if i == subAnchor {
			result = append(result, prefix+line)
		} else {
			result = append(result, indent+line)
		}
	}
	return result, subAnchor
}

// branchInfo 分支渲染信息
type branchInfo struct {
	lines       []string
	anchor      int
	aboveAnchor int
	belowAnchor int
	padAbove    int
	padBelow    int
}

// renderLine 渲染行信息
type renderLine struct {
	isAnchor    bool
	isCenterSep bool
	isPad       bool
	content     string
	branchIndex int
}

func (r *DiagramRenderer) renderBranches(state string, targets []string, visited map[string]bool) ([]string, int) {
	if len(targets) == 0 {
		return nil, 0
	}

	branches := r.collectBranches(targets, visited)
	r.applyInnerPadding(branches)
	allLines := r.buildRenderLines(branches)
	centerLineIndex := r.findCenterLine(allLines, len(branches))

	return r.formatBranchOutput(state, allLines, centerLineIndex, len(branches)), centerLineIndex
}

func (r *DiagramRenderer) collectBranches(targets []string, visited map[string]bool) []branchInfo {
	var branches []branchInfo
	for _, to := range targets {
		lines, anchor := r.renderFlow(to, copyVisited(visited))
		branches = append(branches, branchInfo{
			lines:       lines,
			anchor:      anchor,
			aboveAnchor: anchor,
			belowAnchor: len(lines) - 1 - anchor,
		})
	}
	return branches
}

func (r *DiagramRenderer) applyInnerPadding(branches []branchInfo) {
	upperHalf := len(branches) / 2
	lowerStartIndex := len(branches) - upperHalf

	upperInnerBelow := 0
	if upperHalf > 0 {
		upperInnerBelow = branches[upperHalf-1].belowAnchor
	}

	lowerInnerAbove := 0
	if lowerStartIndex < len(branches) {
		lowerInnerAbove = branches[lowerStartIndex].aboveAnchor
	}

	maxExtend := max(upperInnerBelow, lowerInnerAbove)
	if len(branches)%2 == 0 && maxExtend < 1 {
		maxExtend = 1
	}

	if upperHalf > 0 {
		branches[upperHalf-1].padBelow = maxExtend - branches[upperHalf-1].belowAnchor
	}
	if lowerStartIndex < len(branches) {
		branches[lowerStartIndex].padAbove = maxExtend - branches[lowerStartIndex].aboveAnchor
	}
}

func (r *DiagramRenderer) buildRenderLines(branches []branchInfo) []renderLine {
	upperHalf := len(branches) / 2
	var allLines []renderLine

	for i, b := range branches {
		for k := 0; k < b.padAbove; k++ {
			allLines = append(allLines, renderLine{isPad: true})
		}

		for j, line := range b.lines {
			allLines = append(allLines, renderLine{
				isAnchor:    j == b.anchor,
				content:     line,
				branchIndex: i,
			})
		}

		for k := 0; k < b.padBelow; k++ {
			allLines = append(allLines, renderLine{isPad: true})
		}

		if i < len(branches)-1 {
			isCenter := i == upperHalf-1 && len(branches)%2 == 0
			allLines = append(allLines, renderLine{isCenterSep: isCenter})
		}
	}
	return allLines
}

func (r *DiagramRenderer) findCenterLine(allLines []renderLine, branchCount int) int {
	if branchCount%2 == 1 {
		midIdx := branchCount / 2
		for i, line := range allLines {
			if line.isAnchor && line.branchIndex == midIdx {
				return i
			}
		}
	} else {
		for i, line := range allLines {
			if line.isCenterSep {
				return i
			}
		}
	}
	return 0
}

func (r *DiagramRenderer) formatBranchOutput(state string, allLines []renderLine, centerLineIndex, branchCount int) []string {
	firstAnchor, lastAnchor := r.findAnchorRange(allLines)

	prefix := state + symbolArrow[:len(symbolArrow)-1]
	junctionIndent := strings.Repeat(" ", len(prefix))
	subIndent := strings.Repeat(" ", len(symbolBranch)-1)

	var result []string
	for i, lineData := range allLines {
		var lineStr string

		if i == centerLineIndex {
			if branchCount%2 == 1 {
				lineStr = prefix + symbolBranch + lineData.content
			} else {
				lineStr = prefix + symbolJunction
			}
		} else {
			marker := symbolSpace
			if i > firstAnchor && i < lastAnchor {
				marker = symbolVertical
			}

			switch {
			case lineData.isAnchor:
				lineStr = junctionIndent + symbolBranch + lineData.content
			case lineData.isPad || lineData.content == "":
				lineStr = junctionIndent + marker
			default:
				lineStr = junctionIndent + marker + subIndent + lineData.content
			}
		}
		result = append(result, lineStr)
	}
	return result
}

func (r *DiagramRenderer) findAnchorRange(allLines []renderLine) (first, last int) {
	first, last = -1, -1
	for i, line := range allLines {
		if line.isAnchor {
			if first == -1 {
				first = i
			}
			last = i
		}
	}
	return first, last
}

func (r *DiagramRenderer) renderApprovalFlow(state string, approval *ApprovalInfo, visited map[string]bool) ([]string, int) {
	visited[state] = true
	prefix := state + symbolArrow
	lb := lineBuilder(strings.Repeat(" ", len(prefix)))

	var result []string

	commitLines, commitAnchor := r.renderFlow(approval.Commit, copyVisited(visited))
	commitBelowAnchor := len(commitLines) - 1 - commitAnchor

	commitPrefix := symbolCommitPrefix + symbolSpace
	commitIndent := strings.Repeat(" ", len(commitPrefix))
	commitVerticalIndent := strings.Repeat(" ", len(commitPrefix)-1)

	for j, line := range commitLines {
		switch {
		case j < commitAnchor:
			result = append(result, string(lb)+commitIndent+line)
		case j == commitAnchor:
			result = append(result, string(lb)+commitPrefix+line)
		default:
			result = append(result, lb.verticalWith(commitVerticalIndent, line))
		}
	}

	rejectLines, rejectAnchor := r.renderFlow(approval.Reject, copyVisited(visited))
	rejectBelowAnchor := len(rejectLines) - 1 - rejectAnchor

	gapTop := max(0, rejectAnchor-commitAnchor)
	gapBottom := max(0, commitBelowAnchor-rejectBelowAnchor)

	for i := 0; i < gapTop; i++ {
		result = append(result, lb.vertical())
	}

	result = append(result, lb.vertical())
	result = append(result, prefix+approval.Via+symbolViaSuffix)
	result = append(result, lb.vertical())

	for i := 0; i < gapBottom; i++ {
		result = append(result, lb.vertical())
	}

	rejectPrefix := symbolRejectPrefix + symbolSpace
	rejectVerticalIndent := strings.Repeat(" ", len(rejectPrefix)-1)

	for j, line := range rejectLines {
		switch {
		case j < rejectAnchor:
			result = append(result, lb.verticalWith(rejectVerticalIndent, line))
		case j == rejectAnchor:
			result = append(result, string(lb)+rejectPrefix+line)
		default:
			result = append(result, lb.spacedWith(rejectVerticalIndent, line))
		}
	}

	viaLineIndex := len(commitLines) + gapTop + 1
	return result, viaLineIndex
}

func (r *DiagramRenderer) RenderAsComment() string {
	content := r.Render()
	if content == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("// State Flow Diagram:\n")
	sb.WriteString("// ```\n")
	for _, line := range strings.Split(content, "\n") {
		sb.WriteString("// ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString("// ```\n")
	return sb.String()
}

func copyVisited(visited map[string]bool) map[string]bool {
	result := make(map[string]bool, len(visited))
	for k, v := range visited {
		result[k] = v
	}
	return result
}

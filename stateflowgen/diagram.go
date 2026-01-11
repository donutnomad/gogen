package stateflowgen

import "strings"

// 流程图渲染符号常量
const (
	symbolArrow        = " --> "            // 箭头
	symbolBranch       = "+--> "            // 分支箭头
	symbolJunction     = "+"                // 分支节点
	symbolVertical     = "|"                // 垂直连接线
	symbolSpace        = " "                // 空格
	symbolLoop         = " 🔁"               // 回环标记
	symbolViaSuffix    = " (via)"           // via 后缀
	symbolCommitPrefix = "+-- <Commit> -->" // 提交分支前缀（不含尾空格）
	symbolRejectPrefix = "+-- <Reject> -->" // 拒绝分支前缀（不含尾空格）
)

// lineBuilder 行构建辅助
type lineBuilder string

// vertical 返回带竖线的行
func (b lineBuilder) vertical() string {
	return string(b) + symbolVertical
}

// verticalWith 返回带竖线和内容的行
func (b lineBuilder) verticalWith(indent, content string) string {
	return string(b) + symbolVertical + indent + content
}

// spacedWith 返回带空格和内容的行
func (b lineBuilder) spacedWith(indent, content string) string {
	return string(b) + symbolSpace + indent + content
}

// DiagramRenderer 流程图渲染器
type DiagramRenderer struct {
	transitions map[string][]string // from -> []to
	approvals   map[string]*ApprovalInfo
	order       []string // 保持添加顺序
}

// ApprovalInfo 审批信息
type ApprovalInfo struct {
	Via    string
	Commit string
	Reject string
}

// NewDiagramRenderer 创建渲染器
func NewDiagramRenderer() *DiagramRenderer {
	return &DiagramRenderer{
		transitions: make(map[string][]string),
		approvals:   make(map[string]*ApprovalInfo),
	}
}

// AddDirectTransition 添加直接流转
func (r *DiagramRenderer) AddDirectTransition(from, to string) {
	if _, exists := r.transitions[from]; !exists {
		r.order = append(r.order, from)
	}
	r.transitions[from] = append(r.transitions[from], to)
}

// AddApprovalTransition 添加审批流转
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

// Render 渲染流程图
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

// findEntryState 找到入口状态
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

// renderFlow 递归渲染流程（从后往前生成）
// 返回渲染结果和锚点行索引
func (r *DiagramRenderer) renderFlow(state string, visited map[string]bool) ([]string, int) {
	return r.renderFlowWithMinHeight(state, visited, 0)
}

// renderFlowWithMinHeight 带最小高度约束的递归渲染
// minHeight: 最小渲染高度要求（0表示无约束）
func (r *DiagramRenderer) renderFlowWithMinHeight(state string, visited map[string]bool, minHeight int) ([]string, int) {
	// 检查回环
	if visited[state] {
		return []string{state + symbolLoop}, 0
	}

	// 检查是否有审批流转
	if approval, ok := r.approvals[state]; ok {
		return r.renderApprovalFlowWithMinHeight(state, approval, visited)
	}

	// 获取目标状态
	targets := r.transitions[state]
	if len(targets) == 0 {
		// 终态
		return []string{state}, 0
	}

	visited[state] = true

	if len(targets) == 1 {
		return r.renderSingleTargetWithMinHeight(state, targets[0], visited, minHeight)
	}

	return r.renderBranchesWithMinHeight(state, targets, visited)
}

// renderSingleTarget 渲染单目标（线性流转）
func (r *DiagramRenderer) renderSingleTarget(state, target string, visited map[string]bool) ([]string, int) {
	return r.renderSingleTargetWithMinHeight(state, target, visited, 0)
}

// renderSingleTargetWithMinHeight 带最小高度约束的单目标渲染
func (r *DiagramRenderer) renderSingleTargetWithMinHeight(state, target string, visited map[string]bool, minHeight int) ([]string, int) {
	// 先递归渲染目标，传递最小高度约束
	subLines, subAnchor := r.renderFlowWithMinHeight(target, copyVisited(visited), minHeight)

	if len(subLines) == 0 {
		return []string{state}, 0
	}

	// 在锚点行前面加上 "state --> "
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

// renderBranches 渲染多分支
func (r *DiagramRenderer) renderBranches(state string, targets []string, visited map[string]bool) ([]string, int) {
	return r.renderBranchesWithMinHeight(state, targets, visited)
}

// renderBranchesWithMinHeight 带最小高度约束的多分支渲染
// 核心：从后往前生成，先得到所有子分支的完整渲染，再组装
// 关键规则：
// 1. 上半分支的 belowAnchor 应等于下半分支的 aboveAnchor（中心对称）
// 2. 最末尾的分支，每个分支的空间永远为1行
func (r *DiagramRenderer) renderBranchesWithMinHeight(state string, targets []string, visited map[string]bool) ([]string, int) {
	if len(targets) == 0 {
		return []string{}, 0
	}

	type branchInfo struct {
		lines       []string
		anchor      int
		aboveAnchor int
		belowAnchor int
		padAbove    int
		padBelow    int
	}

	var branches []branchInfo
	for _, to := range targets {
		branchVisited := copyVisited(visited)
		lines, anchor := r.renderFlow(to, branchVisited)
		branches = append(branches, branchInfo{
			lines:       lines,
			anchor:      anchor,
			aboveAnchor: anchor,
			belowAnchor: len(lines) - 1 - anchor,
		})
	}

	// Calculate maxExtend based on INNERMOST branches only
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

	maxExtend := upperInnerBelow
	if lowerInnerAbove > maxExtend {
		maxExtend = lowerInnerAbove
	}

	// Enforce spacing for even branches
	if len(branches)%2 == 0 && maxExtend < 1 {
		maxExtend = 1
	}

	// Apply padding to innermost branches
	if upperHalf > 0 {
		b := &branches[upperHalf-1]
		b.padBelow = maxExtend - b.belowAnchor
	}
	if lowerStartIndex < len(branches) {
		b := &branches[lowerStartIndex]
		b.padAbove = maxExtend - b.aboveAnchor
	}

	// Collect all lines to determine center and connectivity
	type renderLine struct {
		isAnchor    bool
		isCenterSep bool
		isSeparator bool
		isPad       bool
		content     string
		branchIndex int
	}

	var allLines []renderLine

	for i := range branches {
		b := branches[i]

		// Pad Above
		for k := 0; k < b.padAbove; k++ {
			allLines = append(allLines, renderLine{isPad: true})
		}

		// Content
		for j, line := range b.lines {
			allLines = append(allLines, renderLine{
				isAnchor:    j == b.anchor,
				content:     line,
				branchIndex: i,
			})
		}

		// Pad Below
		for k := 0; k < b.padBelow; k++ {
			allLines = append(allLines, renderLine{isPad: true})
		}

		// Separator (unless last)
		if i < len(branches)-1 {
			isCenter := (i == upperHalf-1) && (len(branches)%2 == 0)
			allLines = append(allLines, renderLine{
				isSeparator: true,
				isCenterSep: isCenter,
			})
		}
	}

	// Determine center line index
	var centerLineIndex int
	if len(branches)%2 == 1 {
		// Odd: center is the anchor of the middle branch
		midIdx := len(branches) / 2
		for i, line := range allLines {
			if line.isAnchor && line.branchIndex == midIdx {
				centerLineIndex = i
				break
			}
		}
	} else {
		// Even: center is the middle separator
		for i, line := range allLines {
			if line.isCenterSep {
				centerLineIndex = i
				break
			}
		}
	}

	// Determine vertical connectivity range (from first anchor to last anchor)
	firstAnchor := -1
	lastAnchor := -1
	for i, line := range allLines {
		if line.isAnchor {
			if firstAnchor == -1 {
				firstAnchor = i
			}
			lastAnchor = i
		}
	}

	// Render results
	var result []string
	prefix := state + symbolArrow[:len(symbolArrow)-1] // 去掉尾空格
	branchPrefix := symbolBranch
	junctionIndent := strings.Repeat(" ", len(prefix))

	// Sub-indent for content lines
	subIndent := strings.Repeat(" ", len(branchPrefix)-1)

	for i, lineData := range allLines {
		var lineStr string

		if i == centerLineIndex {
			if len(branches)%2 == 1 {
				// Odd center: prefix + content (which starts with branchPrefix)
				// The content from renderFlow/renderBranches is just the line.
				// But wait, b.lines[anchor] from renderFlow puts the target name?
				// renderFlow returns just lines. Target name is inside the lines?
				// No, renderFlow returns lines of the TARGET.
				// E.g. "L1".
				// So lineData.content is "L1".
				// We need to output "A -->+--> L1".
				// prefix ("A -->") + branchPrefix ("+--> ") + content ("L1") ?
				// Matches "A -->+--> L1".
				lineStr = prefix + branchPrefix + lineData.content
			} else {
				// Even center: separator '+' (Caller preprends parent arrow)
				// Wait, NO. We are the ones rendering the parent arrow!
				// result = append(result, prefix + "+")
				lineStr = prefix + symbolJunction
			}
		} else {
			needsBar := i > firstAnchor && i < lastAnchor
			marker := symbolSpace
			if needsBar {
				marker = symbolVertical
			}

			if lineData.isAnchor {
				lineStr = junctionIndent + branchPrefix + lineData.content
			} else if lineData.isPad || (lineData.isCenterSep == false && lineData.content == "") {
				lineStr = junctionIndent + marker
			} else {
				// Regular content
				lineStr = junctionIndent + marker + subIndent + lineData.content
			}
		}
		result = append(result, lineStr)
	}

	return result, centerLineIndex
}

// renderApprovalFlow 渲染审批流转
func (r *DiagramRenderer) renderApprovalFlow(state string, approval *ApprovalInfo, visited map[string]bool) ([]string, int) {
	return r.renderApprovalFlowWithMinHeight(state, approval, visited)
}

// renderApprovalFlowWithMinHeight 带最小高度约束的审批流转渲染
func (r *DiagramRenderer) renderApprovalFlowWithMinHeight(state string, approval *ApprovalInfo, visited map[string]bool) ([]string, int) {
	visited[state] = true
	prefix := state + symbolArrow
	lb := lineBuilder(strings.Repeat(" ", len(prefix)))

	var result []string

	// Commit 分支（先递归渲染）
	commitVisited := copyVisited(visited)
	commitLines, commitAnchor := r.renderFlow(approval.Commit, commitVisited)
	commitAboveAnchor := commitAnchor
	commitBelowAnchor := len(commitLines) - 1 - commitAnchor

	commitPrefix := symbolCommitPrefix + symbolSpace
	commitIndent := strings.Repeat(" ", len(commitPrefix))
	// 竖线行缩进少1位（因为有|字符）
	commitVerticalIndent := strings.Repeat(" ", len(commitPrefix)-1)

	for j, line := range commitLines {
		switch {
		case j < commitAnchor:
			// Commit 分支上方没有竖线，直接使用完整缩进
			result = append(result, string(lb)+commitIndent+line)
		case j == commitAnchor:
			result = append(result, string(lb)+commitPrefix+line)
		default:
			// Commit 分支下方有竖线，连接 Via
			result = append(result, lb.verticalWith(commitVerticalIndent, line))
		}
	}

	// Reject 分支（先递归渲染）
	rejectVisited := copyVisited(visited)
	rejectLines, rejectAnchor := r.renderFlow(approval.Reject, rejectVisited)
	rejectAboveAnchor := rejectAnchor
	rejectBelowAnchor := len(rejectLines) - 1 - rejectAnchor

	// 基于锚点位置计算 gap，使上下对称
	// gapTop: 当 reject 的上半部分比 commit 的上半部分更高时，在 via 之前添加空间
	// gapBottom: 当 commit 的下半部分比 reject 的下半部分更长时，在 via 之后添加空间
	gapTop := 0
	gapBottom := 0
	if rejectAboveAnchor > commitAboveAnchor {
		gapTop = rejectAboveAnchor - commitAboveAnchor
	}
	if commitBelowAnchor > rejectBelowAnchor {
		gapBottom = commitBelowAnchor - rejectBelowAnchor
	}

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
	// Reject 分支上方有竖线，连接 Via
	// 竖线行缩进少1位
	rejectVerticalIndent := strings.Repeat(" ", len(rejectPrefix)-1)

	for j, line := range rejectLines {
		switch {
		case j < rejectAnchor:
			// Reject 分支上方有竖线
			result = append(result, lb.verticalWith(rejectVerticalIndent, line))
		case j == rejectAnchor:
			result = append(result, string(lb)+rejectPrefix+line)
		default:
			// Reject 分支下方只是缩进
			result = append(result, lb.spacedWith(rejectVerticalIndent, line))
		}
	}

	// 锚点在 via 行
	// via 行位置 = commitLines + gapTop + 1（第一个 |）+ 1（via 行本身在结果中的偏移）
	viaLineIndex := len(commitLines) + gapTop + 1

	return result, viaLineIndex
}

// RenderAsComment 渲染为注释格式
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

// copyVisited 复制已访问集合
func copyVisited(visited map[string]bool) map[string]bool {
	newVisited := make(map[string]bool)
	for k, v := range visited {
		newVisited[k] = v
	}
	return newVisited
}

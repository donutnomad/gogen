package stateflowgen

import "strings"

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

// renderResult 渲染结果
type renderResult struct {
	lines  []string // 渲染后的所有行
	anchor int      // 锚点行索引（父节点应连接到此行）
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
		return []string{state + " 🔁"}, 0
	}

	// 检查是否有审批流转
	if approval, ok := r.approvals[state]; ok {
		return r.renderApprovalFlowWithMinHeight(state, approval, visited, minHeight)
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

	return r.renderBranchesWithMinHeight(state, targets, visited, minHeight)
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
	prefix := state + " --> "
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
	return r.renderBranchesWithMinHeight(state, targets, visited, 0)
}

// renderBranchesWithMinHeight 带最小高度约束的多分支渲染
// 核心：从后往前生成，先得到所有子分支的完整渲染，再组装
// 关键规则：
// 1. 上半分支的 belowAnchor 应等于下半分支的 aboveAnchor（中心对称）
// 2. 最末尾的分支，每个分支的空间永远为1行
func (r *DiagramRenderer) renderBranchesWithMinHeight(state string, targets []string, visited map[string]bool, minHeight int) ([]string, int) {
	// 第一步：递归渲染所有分支，计算自然高度
	type branchInfo struct {
		target      string
		lines       []string
		anchor      int
		aboveAnchor int
		belowAnchor int
		padAbove    int // 需要在内容前添加的竖线行数
		padBelow    int // 需要在内容后添加的竖线行数
	}
	var branches []branchInfo
	for _, to := range targets {
		branchVisited := copyVisited(visited)
		lines, anchor := r.renderFlow(to, branchVisited)
		branches = append(branches, branchInfo{
			target:      to,
			lines:       lines,
			anchor:      anchor,
			aboveAnchor: anchor,
			belowAnchor: len(lines) - 1 - anchor,
		})
	}

	// 第二步：计算上半部分和下半部分需要对称的空间
	var upperMaxBelow, lowerMaxAbove int
	var midIndex int

	if len(branches)%2 == 1 {
		midIndex = len(branches) / 2
		for i := 0; i < midIndex; i++ {
			if branches[i].belowAnchor > upperMaxBelow {
				upperMaxBelow = branches[i].belowAnchor
			}
		}
		for i := midIndex + 1; i < len(branches); i++ {
			if branches[i].aboveAnchor > lowerMaxAbove {
				lowerMaxAbove = branches[i].aboveAnchor
			}
		}
	} else {
		midIndex = -1
		upperHalf := len(branches) / 2
		for i := 0; i < upperHalf; i++ {
			if branches[i].belowAnchor > upperMaxBelow {
				upperMaxBelow = branches[i].belowAnchor
			}
		}
		for i := upperHalf; i < len(branches); i++ {
			if branches[i].aboveAnchor > lowerMaxAbove {
				lowerMaxAbove = branches[i].aboveAnchor
			}
		}
	}

	// 确保上下对称：上半的 belowAnchor 要等于下半的 aboveAnchor
	maxExtend := upperMaxBelow
	if lowerMaxAbove > maxExtend {
		maxExtend = lowerMaxAbove
	}

	// 第三步：计算自然总高度
	naturalHeight := 0
	for i, b := range branches {
		// 每个分支的最终高度 = 自然高度 + 需要的 padding
		branchFinalHeight := len(b.lines)
		if len(branches)%2 == 1 {
			if i < midIndex {
				branchFinalHeight += maxExtend - b.belowAnchor
			} else if i > midIndex {
				branchFinalHeight += maxExtend - b.aboveAnchor
			}
		} else {
			upperHalf := len(branches) / 2
			if i < upperHalf {
				branchFinalHeight += maxExtend - b.belowAnchor
			} else {
				branchFinalHeight += maxExtend - b.aboveAnchor
			}
		}
		naturalHeight += branchFinalHeight
		if i < len(branches)-1 {
			naturalHeight++ // 分隔行
		}
	}

	// 第四步：如果有 minHeight 约束，需要扩展
	extraFromMinHeight := 0
	if minHeight > naturalHeight {
		extraNeeded := minHeight - naturalHeight
		// 均分到各个分支，通过增加 maxExtend
		extraFromMinHeight = (extraNeeded + 1) / 2
		maxExtend += extraFromMinHeight
	}

	// 第五步：计算每个分支需要的目标高度，如果需要则重新渲染
	for i := range branches {
		b := &branches[i]
		var targetHeight int

		if len(branches)%2 == 1 {
			if i < midIndex {
				// 上半分支：需要 belowAnchor 达到 maxExtend
				targetHeight = b.aboveAnchor + 1 + maxExtend
			} else if i > midIndex {
				// 下半分支：需要 aboveAnchor 达到 maxExtend
				targetHeight = maxExtend + 1 + b.belowAnchor
			} else {
				// 中间分支：不需要扩展
				targetHeight = len(b.lines)
			}
		} else {
			upperHalf := len(branches) / 2
			if i < upperHalf {
				targetHeight = b.aboveAnchor + 1 + maxExtend
			} else {
				targetHeight = maxExtend + 1 + b.belowAnchor
			}
		}

		// 如果需要更大高度，重新渲染
		if targetHeight > len(b.lines) {
			branchVisited := copyVisited(visited)
			b.lines, b.anchor = r.renderFlowWithMinHeight(b.target, branchVisited, targetHeight)
			b.aboveAnchor = b.anchor
			b.belowAnchor = len(b.lines) - 1 - b.anchor
		}
	}

	// 第六步：重新计算 maxExtend（重新渲染后可能变化）
	// 但要确保不低于 minHeight 约束所需的值
	upperMaxBelow = 0
	lowerMaxAbove = 0
	if len(branches)%2 == 1 {
		for i := 0; i < midIndex; i++ {
			if branches[i].belowAnchor > upperMaxBelow {
				upperMaxBelow = branches[i].belowAnchor
			}
		}
		for i := midIndex + 1; i < len(branches); i++ {
			if branches[i].aboveAnchor > lowerMaxAbove {
				lowerMaxAbove = branches[i].aboveAnchor
			}
		}
	} else {
		upperHalf := len(branches) / 2
		for i := 0; i < upperHalf; i++ {
			if branches[i].belowAnchor > upperMaxBelow {
				upperMaxBelow = branches[i].belowAnchor
			}
		}
		for i := upperHalf; i < len(branches); i++ {
			if branches[i].aboveAnchor > lowerMaxAbove {
				lowerMaxAbove = branches[i].aboveAnchor
			}
		}
	}

	// 取自然对称值和 minHeight 扩展值中的较大者
	newMaxExtend := upperMaxBelow
	if lowerMaxAbove > newMaxExtend {
		newMaxExtend = lowerMaxAbove
	}
	// 确保不低于 minHeight 带来的扩展
	if maxExtend > newMaxExtend {
		newMaxExtend = maxExtend
	}
	maxExtend = newMaxExtend

	// 第七步：计算 padding
	for i := range branches {
		b := &branches[i]
		b.padAbove = 0
		b.padBelow = 0

		if len(branches)%2 == 1 {
			if i < midIndex {
				b.padBelow = maxExtend - b.belowAnchor
			} else if i > midIndex {
				b.padAbove = maxExtend - b.aboveAnchor
			}
		} else {
			upperHalf := len(branches) / 2
			if i < upperHalf {
				b.padBelow = maxExtend - b.belowAnchor
			} else {
				b.padAbove = maxExtend - b.aboveAnchor
			}
		}
	}

	// 第八步：计算每个分支块的位置（含填充）
	type blockPos struct {
		startLine int
		endLine   int
		anchor    int // 全局锚点行（含填充）
	}
	var blocks []blockPos

	currentLine := 0
	for i, b := range branches {
		// 实际高度 = padAbove + 内容高度 + padBelow
		height := b.padAbove + len(b.lines) + b.padBelow
		pos := blockPos{
			startLine: currentLine,
			endLine:   currentLine + height - 1,
			anchor:    currentLine + b.padAbove + b.anchor,
		}
		blocks = append(blocks, pos)
		currentLine += height

		if i < len(branches)-1 {
			currentLine++ // 分隔行
		}
	}

	// 第九步：确定中心行
	var centerLine int
	if len(blocks)%2 == 1 {
		centerLine = blocks[len(blocks)/2].anchor
	} else {
		upperBlock := len(blocks)/2 - 1
		centerLine = blocks[upperBlock].endLine + 1
	}

	// 竖线范围
	firstAnchor := blocks[0].anchor
	lastAnchor := blocks[len(blocks)-1].anchor

	// 第十步：构建输出
	prefix := state + " -->"
	junctionIndent := strings.Repeat(" ", len(prefix))
	branchPrefix := "+--> "
	branchIndent := strings.Repeat(" ", len(branchPrefix))

	var result []string

	for i, b := range branches {
		blockStart := blocks[i].startLine

		// 输出 padAbove 行
		for k := 0; k < b.padAbove; k++ {
			globalLine := blockStart + k
			inVerticalRange := globalLine > firstAnchor && globalLine < lastAnchor
			isCenter := globalLine == centerLine
			if isCenter {
				result = append(result, prefix+"|")
			} else if inVerticalRange {
				result = append(result, junctionIndent+"|")
			} else {
				result = append(result, junctionIndent+" ")
			}
		}

		// 输出分支内容
		for j, line := range b.lines {
			globalLine := blockStart + b.padAbove + j
			isAnchor := j == b.anchor
			isCenter := globalLine == centerLine
			inVerticalRange := globalLine > firstAnchor && globalLine < lastAnchor

			var out string
			if isCenter && isAnchor {
				out = prefix + branchPrefix + line
			} else if isAnchor {
				out = junctionIndent + branchPrefix + line
			} else if inVerticalRange {
				if isCenter {
					out = prefix + "|" + branchIndent + line
				} else {
					out = junctionIndent + "|" + branchIndent + line
				}
			} else {
				if isCenter {
					out = prefix + " " + branchIndent + line
				} else {
					out = junctionIndent + " " + branchIndent + line
				}
			}
			result = append(result, out)
		}

		// 输出 padBelow 行
		for k := 0; k < b.padBelow; k++ {
			globalLine := blockStart + b.padAbove + len(b.lines) + k
			inVerticalRange := globalLine > firstAnchor && globalLine < lastAnchor
			isCenter := globalLine == centerLine
			if isCenter {
				result = append(result, prefix+"|")
			} else if inVerticalRange {
				result = append(result, junctionIndent+"|")
			} else {
				result = append(result, junctionIndent+" ")
			}
		}

		// 分支之间添加分隔行
		if i < len(branches)-1 {
			sepLine := blocks[i].endLine + 1
			if sepLine == centerLine {
				result = append(result, prefix+"+")
			} else {
				result = append(result, junctionIndent+"|")
			}
		}
	}

	return result, centerLine
}

// renderApprovalFlow 渲染审批流转
func (r *DiagramRenderer) renderApprovalFlow(state string, approval *ApprovalInfo, visited map[string]bool) ([]string, int) {
	return r.renderApprovalFlowWithMinHeight(state, approval, visited, 0)
}

// renderApprovalFlowWithMinHeight 带最小高度约束的审批流转渲染
func (r *DiagramRenderer) renderApprovalFlowWithMinHeight(state string, approval *ApprovalInfo, visited map[string]bool, minHeight int) ([]string, int) {
	visited[state] = true
	prefix := state + " --> "
	junctionIndent := strings.Repeat(" ", len(prefix))

	var result []string

	// Commit 分支（先递归渲染）
	commitVisited := copyVisited(visited)
	commitLines, commitAnchor := r.renderFlow(approval.Commit, commitVisited)
	commitAboveAnchor := commitAnchor
	commitBelowAnchor := len(commitLines) - 1 - commitAnchor

	commitPrefix := "+-- <Commit> --> "
	commitIndent := strings.Repeat(" ", len(commitPrefix))
	// 竖线行缩进少1位（因为有|字符）
	commitVerticalIndent := ""
	if len(commitPrefix) > 1 {
		commitVerticalIndent = strings.Repeat(" ", len(commitPrefix)-1)
	}

	for j, line := range commitLines {
		switch {
		case j < commitAnchor:
			// Commit 分支上方没有竖线，直接使用完整缩进
			result = append(result, junctionIndent+commitIndent+line)
		case j == commitAnchor:
			result = append(result, junctionIndent+commitPrefix+line)
		default:
			// Commit 分支下方有竖线，连接 Via
			result = append(result, junctionIndent+"|"+commitVerticalIndent+line)
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
		result = append(result, junctionIndent+"|")
	}

	result = append(result, junctionIndent+"|")
	result = append(result, prefix+approval.Via+" (via)")
	result = append(result, junctionIndent+"|")

	for i := 0; i < gapBottom; i++ {
		result = append(result, junctionIndent+"|")
	}

	rejectPrefix := "+-- <Reject> --> "
	// Reject 分支上方有竖线，连接 Via
	// 竖线行缩进少1位
	rejectVerticalIndent := ""
	if len(rejectPrefix) > 1 {
		rejectVerticalIndent = strings.Repeat(" ", len(rejectPrefix)-1)
	}

	for j, line := range rejectLines {
		switch {
		case j < rejectAnchor:
			// Reject 分支上方有竖线
			result = append(result, junctionIndent+"|"+rejectVerticalIndent+line)
		case j == rejectAnchor:
			result = append(result, junctionIndent+rejectPrefix+line)
		default:
			// Reject 分支下方只是缩进
			result = append(result, junctionIndent+" "+rejectVerticalIndent+line)
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

// inflateBranchLines 膨胀分支渲染结果到目标高度
// 在锚点上下均匀添加竖线行
// 注意：竖线需要正确的缩进，这里假设每行的前缀宽度一致
func inflateBranchLines(lines []string, anchor int, targetHeight int) ([]string, int) {
	currentHeight := len(lines)
	if currentHeight >= targetHeight {
		return lines, anchor
	}

	needed := targetHeight - currentHeight
	// 在锚点上下均匀添加
	addBelow := needed / 2
	addAbove := needed - addBelow

	// 找到每行的前缀宽度（到第一个非空格字符的距离）
	// 我们需要在正确的位置添加竖线
	findPrefixWidth := func(s string) int {
		for i, c := range s {
			if c != ' ' {
				return i
			}
		}
		return len(s)
	}

	// 使用锚点行的前缀宽度作为参考
	prefixWidth := 0
	if anchor < len(lines) {
		prefixWidth = findPrefixWidth(lines[anchor])
	}

	result := make([]string, 0, targetHeight)

	// 添加上方的竖线
	for i := 0; i < addAbove; i++ {
		result = append(result, strings.Repeat(" ", prefixWidth)+"|")
	}

	// 添加原始内容
	result = append(result, lines...)

	// 添加下方的竖线
	for i := 0; i < addBelow; i++ {
		result = append(result, strings.Repeat(" ", prefixWidth)+"|")
	}

	return result, anchor + addAbove
}

// abs returns absolute value for integers to avoid pulling in math just for this helper.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

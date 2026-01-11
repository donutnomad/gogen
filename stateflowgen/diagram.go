package stateflowgen

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// Const symbols for default consistent look
const (
	defaultLoop = " 🔁"
)

// Node represents a node in the graph
type Node struct {
	ID           string
	Content      string // Text to display (defaults to ID)
	Junction     string // Custom junction symbol (defaults to global setting if empty)
	CornerTop    string // Custom top corner symbol
	CornerBottom string // Custom bottom corner symbol
	Intersection string // Custom intersection symbol (middle branches)
	Style        string // "approval" = classic approval style (no junction on center branch)
	Align        string // Alignment of branches relative to junction ("right" = branch from right)
}

// Edge represents a directed connection
type Edge struct {
	From  string
	To    string
	Label string // E.g., "--> ", "-- Yes --> ". If empty, defaults to "--> "
}

// DiagramRenderer Generic ASCII diagram renderer
type DiagramRenderer struct {
	nodes *OrderedMap[string, Node]
	edges *OrderedMap[string, []Edge]

	// Configuration
	ArrowSymbol    string // 默认箭头符号,用于节点间连接
	VerticalSymbol string

	Junction     string // Custom junction symbol (defaults to global setting if empty)
	CornerTop    string // Custom top corner symbol
	CornerBottom string // Custom bottom corner symbol
	Intersection string // Custom intersection symbol (middle branches)
}

// NewDiagramRenderer creates a new generic renderer
func NewDiagramRenderer() *DiagramRenderer {
	return &DiagramRenderer{
		nodes:          NewOrderedMap[string, Node](),
		edges:          NewOrderedMap[string, []Edge](),
		ArrowSymbol:    "-->",
		VerticalSymbol: "│",
		Junction:       "┤",
		CornerTop:      "┌",
		CornerBottom:   "└",
		Intersection:   "├",
	}
}

// AddNode adds or updates a node
func (r *DiagramRenderer) AddNode(id, content string) {
	if content == "" {
		content = id
	}
	r.ensureNode(id)
	n, _ := r.nodes.Get(id)
	n.Content = content
	r.nodes.Set(id, n)
}

func (r *DiagramRenderer) ensureNode(id string) {
	if !r.nodes.Has(id) {
		r.nodes.Set(id, Node{ID: id, Content: id})
	}
}

// SetJunction sets a custom junction symbol for a node and its alignment
// align: "right" means branches start after the junction symbol
func (r *DiagramRenderer) SetJunction(id, junction, align string) {
	r.ensureNode(id)
	n, _ := r.nodes.Get(id)
	n.Junction = junction
	n.Align = align
	r.nodes.Set(id, n)
}

// SetCorner sets the symbol for BOTH top and bottom corners
func (r *DiagramRenderer) SetCorner(id, top, bottom string) {
	r.ensureNode(id)
	n, _ := r.nodes.Get(id)
	n.CornerTop = top
	n.CornerBottom = bottom
	r.nodes.Set(id, n)
}

// SetCornerTop sets the symbol for the top corner
func (r *DiagramRenderer) SetCornerTop(id, symbol string) {
	r.ensureNode(id)
	n, _ := r.nodes.Get(id)
	n.CornerTop = symbol
	r.nodes.Set(id, n)
}

// SetCornerBottom sets the symbol for the bottom corner
func (r *DiagramRenderer) SetCornerBottom(id, symbol string) {
	r.ensureNode(id)
	n, _ := r.nodes.Get(id)
	n.CornerBottom = symbol
	r.nodes.Set(id, n)
}

// SetIntersection sets the symbol for intermediate branch points
func (r *DiagramRenderer) SetIntersection(id, symbol string) {
	r.ensureNode(id)
	n, _ := r.nodes.Get(id)
	n.Intersection = symbol
	r.nodes.Set(id, n)
}

// SetNodeStyle sets render style for a node
func (r *DiagramRenderer) SetNodeStyle(id, style string) {
	r.ensureNode(id)
	n, _ := r.nodes.Get(id)
	n.Style = style
	r.nodes.Set(id, n)
}

// SetArrowSymbol 设置默认箭头符号
func (r *DiagramRenderer) SetArrowSymbol(symbol string) {
	r.ArrowSymbol = symbol
}

// AddEdge adds a directed edge
// AddEdge adds a directed edge
// If label is empty, defaults to "--> ".
// To assume no label, pass " " (single space) or handle specifically.
func (r *DiagramRenderer) AddEdge(from, to, label string) {
	r.ensureNode(from)
	r.ensureNode(to)

	if label == "" {
		label = strings.TrimSpace(r.ArrowSymbol) + " "
	}

	edges, _ := r.edges.Get(from)
	r.edges.Set(from, append(edges, Edge{From: from, To: to, Label: label}))
}

// Render generates the ASCII diagram
func (r *DiagramRenderer) Render() string {
	if r.nodes.Len() == 0 {
		return ""
	}

	// 计算每个节点的入度
	inDegree := make(map[string]int)
	for _, from := range r.edges.Keys() {
		edges, _ := r.edges.Get(from)
		for _, e := range edges {
			inDegree[e.To]++
		}
	}

	var root string
	// 按插入顺序查找第一个入度为 0 的节点
	for _, id := range r.nodes.Keys() {
		if inDegree[id] == 0 {
			root = id
			break
		}
	}

	// 如果没有入度为 0 的节点(循环图),使用第一个插入的节点
	if root == "" {
		keys := r.nodes.Keys()
		if len(keys) > 0 {
			root = keys[0]
		}
	}

	visited := make(map[string]bool)
	lines, _ := r.renderFlow(root, visited)
	return strings.Join(lines, "\n")
}

func (r *DiagramRenderer) renderFlow(state string, visited map[string]bool) ([]string, int) {
	// Check loop
	if visited[state] {
		content := state
		if n, ok := r.nodes.Get(state); ok {
			content = n.Content
		}
		return []string{content + defaultLoop}, 0
	}

	edges, _ := r.edges.Get(state)
	if len(edges) == 0 {
		// Leaf
		content := state
		if n, ok := r.nodes.Get(state); ok {
			content = n.Content
		}
		return []string{content}, 0
	}

	visited[state] = true

	if len(edges) == 1 {
		return r.renderSingleTarget(state, edges[0], visited)
	}

	return r.renderBranches(state, edges, visited)
}

func (r *DiagramRenderer) renderSingleTarget(state string, edge Edge, visited map[string]bool) ([]string, int) {
	subLines, subAnchor := r.renderFlow(edge.To, copyVisited(visited))

	nodeContent := state
	if n, ok := r.nodes.Get(state); ok {
		nodeContent = n.Content
	}

	// Label usually goes: Node -- Label --> Target
	prefix := nodeContent + " " + edge.Label

	indent := genSpace(prefix)

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

// branchInfo holds rendering data for a branch
type branchInfo struct {
	lines       []string
	anchor      int
	aboveAnchor int
	belowAnchor int
	padAbove    int
	padBelow    int
	edgeLabel   string
}

func (r *DiagramRenderer) renderBranches(state string, edges []Edge, visited map[string]bool) ([]string, int) {
	if len(edges) == 0 {
		return nil, 0
	}

	branches := r.collectBranches(edges, visited)
	r.applyInnerPadding(branches)
	allLines := r.buildRenderLines(branches)
	centerLineIndex := r.findCenterLine(allLines, len(branches))

	return r.formatBranchOutput(state, allLines, centerLineIndex, branches), centerLineIndex
}

func (r *DiagramRenderer) collectBranches(edges []Edge, visited map[string]bool) []branchInfo {
	var branches []branchInfo
	for _, edge := range edges {
		lines, anchor := r.renderFlow(edge.To, copyVisited(visited))
		branches = append(branches, branchInfo{
			lines:       lines,
			anchor:      anchor,
			aboveAnchor: anchor,
			belowAnchor: len(lines) - 1 - anchor,
			edgeLabel:   edge.Label,
		})
	}
	return branches
}

// applyInnerPadding calculates and applies vertical spacing/padding
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

	// Enforce min spacing for even branches
	if len(branches)%2 == 0 && maxExtend < 1 {
		maxExtend = 1
	}

	// Apply padding
	if upperHalf > 0 {
		branches[upperHalf-1].padBelow = maxExtend - branches[upperHalf-1].belowAnchor
	}
	if lowerStartIndex < len(branches) {
		branches[lowerStartIndex].padAbove = maxExtend - branches[lowerStartIndex].aboveAnchor
	}
}

type renderLine struct {
	isAnchor    bool
	isCenterSep bool
	isSeparator bool
	isPad       bool
	content     string
	branchIndex int
}

func (r *DiagramRenderer) buildRenderLines(branches []branchInfo) []renderLine {
	upperHalf := len(branches) / 2
	var allLines []renderLine

	for i, b := range branches {
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
			isCenter := i == upperHalf-1 && len(branches)%2 == 0
			allLines = append(allLines, renderLine{
				isSeparator: true,
				isCenterSep: isCenter,
			})
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

func (r *DiagramRenderer) formatBranchOutput(state string, allLines []renderLine, centerLineIndex int, branches []branchInfo) []string {

	nodeContent := state

	// Defaults
	stemSymbol := r.Junction          // Center Stem
	cornerTopSymbol := r.CornerTop    // Top
	cornerBotSymbol := r.CornerBottom // Bottom
	interSymbol := r.Intersection     // Middle intersections

	isApprovalStyle := false
	align := ""

	if n, ok := r.nodes.Get(state); ok {
		nodeContent = n.Content
		if n.Junction != "" {
			stemSymbol = n.Junction
		}
		if n.CornerTop != "" {
			cornerTopSymbol = n.CornerTop
		}
		if n.CornerBottom != "" {
			cornerBotSymbol = n.CornerBottom
		}
		if n.Intersection != "" {
			interSymbol = n.Intersection
		}
		if n.Style == "approval" {
			isApprovalStyle = true
		}
		align = n.Align
	}

	// Stem prefix: "Node " + ArrowSymbol.
	stemStr := nodeContent + " " + strings.TrimSpace(r.ArrowSymbol)

	// Calculate indentation base
	indentRef := stemStr

	if align == "right" {
		// If right-aligned:
		// 1. Always include the Stem Symbol (Junction)
		indentRef += stemSymbol

		// 2. If we have a center branch (Odd number), we ALSO include its content
		//    This pushes the fork (vertical line) to the right of the center branch.
		if len(branches)%2 == 1 {
			midIdx := len(branches) / 2
			centerBranch := branches[midIdx]

			// Center Branch Content: Label + LineContent
			// Note: We need the content of the anchor line of the center branch
			centerLabel := centerBranch.edgeLabel
			centerContent := centerBranch.lines[centerBranch.anchor]

			indentRef += centerLabel + centerContent
		}
	} else if align == "center" {
		// align="center": Center the fork relative to [Junction + Label + Content]
		// Base is stemStr. indentation adds padding to reach the center of that block.

		totalWidth := 0

		// 1. Junction
		totalWidth += runewidth.StringWidth(stemSymbol)

		if len(branches)%2 == 1 {
			midIdx := len(branches) / 2
			centerBranch := branches[midIdx]

			// 2. Label + Content
			centerLabel := centerBranch.edgeLabel
			centerContent := centerBranch.lines[centerBranch.anchor]

			totalWidth += runewidth.StringWidth(centerLabel + centerContent)
		}

		// Calculate padding: (Total - 1) / 2
		// Example: 18 chars. (18-1)/2 = 8 spaces.
		// Resulting line is at 9th char (index 8).
		if totalWidth > 0 {
			pad := (totalWidth - 1) / 2
			indentRef += strings.Repeat(" ", pad)
		}
	} else if align != "" {
		// Other non-empty alignments (future proofing), simplistic behavior
		indentRef += stemSymbol
	}

	junctionIndent := genSpace(indentRef)

	var result []string

	firstAnchor, lastAnchor := r.findAnchorRange(allLines)

	for i, lineData := range allLines {
		var lineStr string

		if i == centerLineIndex {
			if len(branches)%2 == 1 {
				// Odd center: Stem + Junction + Label + Content
				bIdx := lineData.branchIndex
				label := branches[bIdx].edgeLabel

				// Approval style: Don't render junction on center branch if it's "approval" style?
				// Actually, if we are shifting alignment, we usually want the stemSymbol rendered.
				// The user's hack sets SetJunction("Node -> via").
				// So we generally MUST render stemSymbol here.
				centerJunction := stemSymbol
				if isApprovalStyle && align == "" { // Legacy behavior safeguard
					centerJunction = ""
				}
				// If aligning right, we've implicitly accounted for stemSymbol in indentation of other lines.
				// So we must render it here.

				lineStr = stemStr + centerJunction + label + lineData.content
			} else {
				// Even center: Stem + Socket (Intersection/Stem?)
				// Just the stem connecting to the vertical line
				lineStr = stemStr + stemSymbol
			}
		} else {
			marker := " "
			if i > firstAnchor && i < lastAnchor {
				marker = r.VerticalSymbol
			}

			if lineData.isAnchor {
				bIdx := lineData.branchIndex
				label := branches[bIdx].edgeLabel

				// Determine symbol based on position
				currentSymbol := interSymbol
				if i == firstAnchor {
					currentSymbol = cornerTopSymbol
				} else if i == lastAnchor {
					currentSymbol = cornerBotSymbol
				}

				lineStr = junctionIndent + currentSymbol + label + lineData.content
			} else if lineData.isPad || lineData.isSeparator || lineData.content == "" {
				lineStr = junctionIndent + marker
			} else {
				// Content line within a branch
				bIdx := lineData.branchIndex
				label := branches[bIdx].edgeLabel

				subIndent := genSpace(label)

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

// RenderAsComment renders the diagram wrapped in a Go comment block
func (r *DiagramRenderer) RenderAsComment() string {
	content := r.Render()
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	var sb strings.Builder
	sb.WriteString("// 流程图：\n")
	sb.WriteString("// ```\n")
	for _, line := range lines {
		sb.WriteString("// ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString("// ```\n")
	return sb.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func copyVisited(visited map[string]bool) map[string]bool {
	result := make(map[string]bool, len(visited))
	for k, v := range visited {
		result[k] = v
	}
	return result
}

func genSpace(s string) string {
	var sb strings.Builder
	for _, r := range s {
		w := runewidth.RuneWidth(r)
		if w == 2 {
			sb.WriteString("　")
		} else {
			sb.WriteString(strings.Repeat(" ", w))
		}
	}
	return sb.String()
}

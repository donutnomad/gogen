package stateflowgen

import (
	"strings"
)

// Const symbols for default consistent look
const (
	defaultArrow     = " --> " // Stem arrow
	defaultJunction  = "+"
	defaultVertical  = "│"
	defaultLoop      = " 🔁"
	defaultEdgeLabel = "--> " // Standard branch edge label (should include space at end)
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
	JunctionSymbol string
	VerticalSymbol string
}

// NewDiagramRenderer creates a new generic renderer
func NewDiagramRenderer() *DiagramRenderer {
	return &DiagramRenderer{
		nodes:          NewOrderedMap[string, Node](),
		edges:          NewOrderedMap[string, []Edge](),
		JunctionSymbol: defaultJunction,
		VerticalSymbol: defaultVertical,
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

// SetJunction sets a custom junction symbol for a node
func (r *DiagramRenderer) SetJunction(id, junction string) {
	r.ensureNode(id)
	n, _ := r.nodes.Get(id)
	n.Junction = junction
	r.nodes.Set(id, n)
}

// SetCorner sets the symbol for BOTH top and bottom corners
func (r *DiagramRenderer) SetCorner(id, symbol string) {
	r.ensureNode(id)
	n, _ := r.nodes.Get(id)
	n.CornerTop = symbol
	n.CornerBottom = symbol
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

// AddEdge adds a directed edge
// AddEdge adds a directed edge
// If label is empty, defaults to "--> ".
// To assume no label, pass " " (single space) or handle specifically.
func (r *DiagramRenderer) AddEdge(from, to, label string) {
	r.ensureNode(from)
	r.ensureNode(to)

	if label == "" {
		label = defaultEdgeLabel
	}

	edges, _ := r.edges.Get(from)
	r.edges.Set(from, append(edges, Edge{From: from, To: to, Label: label}))
}

// --- Compatibility API ---

// AddLegacyDirectTransition maps old API to new Generic API
// In original code this was AddDirectTransition. We use that name.
func (r *DiagramRenderer) AddDirectTransition(from, to string) {
	r.AddEdge(from, to, "")
}

// AddApprovalTransition adds an approval flow (Draft -> Reviewing -> Published/Rejected)
// Renders as:
//
//	+-- <Commit> --> to
//	|
//
// from --> via (via)
//
//	|
//	+-- <Reject> --> fallback
func (r *DiagramRenderer) AddApprovalTransition(from, via, to, fallback string) {
	// Replicate visual structure using generic edges

	// 1. Top: Commit path
	if to != "" {
		r.AddEdge(from, to, "-- <Commit> --> ")
	}

	// 2. Middle: Via path
	// Uses " " label and "approval" style to suppress center junction
	r.AddEdge(from, via, " ")
	r.AddNode(via, via+" (via)")

	// 3. Bottom: Reject path
	if fallback != "" {
		r.AddEdge(from, fallback, "-- <Reject> --> ")
	}

	// Set style to "approval" to affect formatBranchOutput behavior
	r.SetNodeStyle(from, "approval")
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
	junctionSymbol := r.JunctionSymbol // The global default, e.g. "+"
	stemSymbol := junctionSymbol       // Center Stem
	cornerTopSymbol := junctionSymbol  // Top
	cornerBotSymbol := junctionSymbol  // Bottom
	interSymbol := junctionSymbol      // Middle intersections

	isApprovalStyle := false

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
	}

	// Stem prefix: "Node " + defaultArrow.
	stemStr := nodeContent + " " + strings.TrimSpace(defaultArrow)
	junctionIndent := strings.Repeat(" ", len(stemStr))

	var result []string

	firstAnchor, lastAnchor := r.findAnchorRange(allLines)

	for i, lineData := range allLines {
		var lineStr string

		if i == centerLineIndex {
			if len(branches)%2 == 1 {
				// Odd center: Stem + Junction + Label + Content
				bIdx := lineData.branchIndex
				label := branches[bIdx].edgeLabel

				// Approval style: Don't render junction on center branch
				centerJunction := stemSymbol
				if isApprovalStyle {
					centerJunction = ""
				}

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

				// Here we need indent matching the symbol above.
				// Since symbol length might vary, we should ideally use len(currentSymbol).
				// But we are inside the branch content lines, the symbol line was printed above.
				// We'll assume symbol length 1 or use `junctionSymbol` length as fallback approximation
				// or just use spaces equal to the indentation logic.
				// The logic above was: strings.Repeat(" ", len(junctionSymbol)+len(label))
				// Now symbols differ. We'll use stemSymbol's length or max?
				// Usually these are 1 char. Let's use `stemSymbol` length for consistency with `junctionIndent`?
				// No, subIndent depends on the `+` used in `+-->`.

				// Wait, currentSymbol depends on i. But here we are in content lines for branch `bIdx`.
				// We need to know which symbol was used for this branch's anchor.

				// Logic check: content lines follow an anchor.
				// The anchor used `currentSymbol`.
				// So subIndent should technically allow for `len(currentSymbol) + len(label)`.

				// Since this loop iterates lines, retrieving the specific symbol for the branch of this content line is tricky without tracking.
				// However, usually these symbols are 1 char.
				// Let's stick to `len(interSymbol)` or `len(cornerSymbol)`?
				// Safe bet: len(junctionSymbol) (global default) if we assume monospaced single chars.
				// Or better: The anchor line for this branch used a symbol.
				// The indentation needs to align with `Indent + Symbol + Label`.
				// So subIndent = " " * (len(Symbol) + len(Label)).

				// Simplification for now: use len(interSymbol) as a proxy,
				// assuming user won't set wildly different lengths for symbols.
				subIndent := strings.Repeat(" ", len(label))

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

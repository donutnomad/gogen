package stateflowgen

import (
	"strings"
)

// Const symbols for default consistent look
const (
	defaultArrow     = " --> " // Stem arrow
	defaultJunction  = "+"
	defaultVertical  = "|"
	defaultLoop      = " 🔁"
	defaultEdgeLabel = "--> " // Standard branch edge label (should include space at end)
)

// Node represents a node in the graph
type Node struct {
	ID       string
	Content  string // Text to display (defaults to ID)
	Junction string // Custom junction symbol (defaults to global setting if empty)
	Style    string // "approval" = classic approval style (no junction on center branch)
}

// Edge represents a directed connection
type Edge struct {
	From  string
	To    string
	Label string // E.g., "--> ", "-- Yes --> ". If empty, defaults to "--> "
}

// DiagramRenderer Generic ASCII diagram renderer
type DiagramRenderer struct {
	nodes map[string]Node
	edges map[string][]Edge

	// Configuration
	JunctionSymbol string
	VerticalSymbol string
}

// NewDiagramRenderer creates a new generic renderer
func NewDiagramRenderer() *DiagramRenderer {
	return &DiagramRenderer{
		nodes:          make(map[string]Node),
		edges:          make(map[string][]Edge),
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
	n := r.nodes[id]
	n.Content = content
	r.nodes[id] = n
}

func (r *DiagramRenderer) ensureNode(id string) {
	if _, ok := r.nodes[id]; !ok {
		r.nodes[id] = Node{ID: id, Content: id}
	}
}

// SetJunction sets a custom junction symbol for a node
func (r *DiagramRenderer) SetJunction(id, junction string) {
	r.ensureNode(id)
	n := r.nodes[id]
	n.Junction = junction
	r.nodes[id] = n
}

// SetNodeStyle sets render style for a node
func (r *DiagramRenderer) SetNodeStyle(id, style string) {
	r.ensureNode(id)
	n := r.nodes[id]
	n.Style = style
	r.nodes[id] = n
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

	r.edges[from] = append(r.edges[from], Edge{From: from, To: to, Label: label})
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
	if len(r.nodes) == 0 {
		return ""
	}

	// Heuristic: Find root (node with in-degree 0)
	inDegree := make(map[string]int)
	for _, edges := range r.edges {
		for _, e := range edges {
			inDegree[e.To]++
		}
	}

	var root string
	// Priority list for root candidates to ensure deterministic start
	candidates := []string{"Start", "Draft", "Entry", "Init", "open", "Open"}

	// 1. Check priority list with InDegree 0
	for _, id := range candidates {
		if _, ok := r.nodes[id]; ok && inDegree[id] == 0 {
			root = id
			break
		}
	}

	if root == "" {
		for id := range r.nodes {
			if inDegree[id] == 0 {
				root = id
				break
			}
		}
	}

	// 3. Cycle interaction: Check priority list for ANY existence (even if InDegree > 0)
	// This handles "open" in ComplexWorkflow loop or others
	if root == "" {
		for _, id := range candidates {
			if _, ok := r.nodes[id]; ok {
				root = id
				break
			}
		}
	}

	if root == "" {
		// Cycle? Pick any.
		for id := range r.nodes {
			root = id
			break
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
		if n, ok := r.nodes[state]; ok {
			content = n.Content
		}
		return []string{content + defaultLoop}, 0
	}

	edges := r.edges[state]
	if len(edges) == 0 {
		// Leaf
		content := state
		if n, ok := r.nodes[state]; ok {
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
	if n, ok := r.nodes[state]; ok {
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
	junctionSymbol := r.JunctionSymbol
	isApprovalStyle := false

	if n, ok := r.nodes[state]; ok {
		nodeContent = n.Content
		if n.Junction != "" {
			junctionSymbol = n.Junction
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
				centerJunction := junctionSymbol
				if isApprovalStyle {
					centerJunction = ""
				}

				lineStr = stemStr + centerJunction + label + lineData.content
			} else {
				// Even center: Stem + Junction
				lineStr = stemStr + junctionSymbol
			}
		} else {
			marker := " "
			if i > firstAnchor && i < lastAnchor {
				marker = r.VerticalSymbol
			}

			if lineData.isAnchor {
				bIdx := lineData.branchIndex
				label := branches[bIdx].edgeLabel

				lineStr = junctionIndent + junctionSymbol + label + lineData.content
			} else if lineData.isPad || lineData.isSeparator || lineData.content == "" {
				lineStr = junctionIndent + marker
			} else {
				// Content line within a branch
				bIdx := lineData.branchIndex
				label := branches[bIdx].edgeLabel

				// subIndent should be spaces equal to len(Junction + Label)
				subIndent := strings.Repeat(" ", len(junctionSymbol)+len(label))

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

package stateflowgen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
)

func AddApprovalTransition(r *DiagramRenderer, from, via, to, fallback string) {
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

// Helper to normalized expectation
func assertRender(t *testing.T, renderer *DiagramRenderer, expectedLines []string) {
	result := renderer.Render()
	expected := strings.Join(expectedLines, "\n")
	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// AddLegacyDirectTransition maps old API to new Generic API
func (r *DiagramRenderer) AddLegacyDirectTransition(from, to string) {
	r.AddEdge(from, to, "--> ")
}

// 测试1：简单线性流程
func TestDiagramRenderer_SimpleLinear(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddEdge("A", "B", "--> ")
	renderer.AddEdge("B", "C", "--> ")

	result := renderer.Render()
	expected := "A --> B --> C"

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试2：两分支（偶数分支，中心在分隔行）
func TestDiagramRenderer_TwoBranches(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddEdge("A", "B", "--> ")
	renderer.AddEdge("A", "C", "--> ")

	result := renderer.Render()
	// 2个分支：B, |, C = 3行，中心行=1
	expected := strings.Join([]string{
		"     ┌--> B",
		"     │",
		"A -->┤",
		"     │",
		"     └--> C",
	}, "\n")

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试3：三分支（奇数分支，中心在中间分支）
func TestDiagramRenderer_ThreeBranches(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddEdge("A", "B", "--> ")
	renderer.AddEdge("A", "C", "--> ")
	renderer.AddEdge("A", "D", "--> ")

	result := renderer.Render()
	// 3个分支：B, |, C, |, D = 5行
	expected := strings.Join([]string{
		"     ┌--> B",
		"     │",
		"A -->┤--> C",
		"     │",
		"     └--> D",
	}, "\n")

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试4：四分支
func TestDiagramRenderer_FourBranches(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddEdge("A", "B", "--> ")
	renderer.AddEdge("A", "C", "--> ")
	renderer.AddEdge("A", "D", "--> ")
	renderer.AddEdge("A", "E", "--> ")

	result := renderer.Render()
	expected := strings.Join([]string{
		"     ┌--> B",
		"     │",
		"     ├--> C",
		"     │",
		"A -->┤",
		"     │",
		"     ├--> D",
		"     │",
		"     └--> E",
	}, "\n")

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试5：嵌套分支
func TestDiagramRenderer_NestedBranches(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddEdge("A", "B", "--> ")
	renderer.AddEdge("B", "C", "--> ")
	renderer.AddEdge("B", "D", "--> ")
	renderer.AddEdge("C", "E", "--> ")

	result := renderer.Render()
	expected := strings.Join([]string{
		"           ┌--> C --> E",
		"           │",
		"A --> B -->┤",
		"           │",
		"           └--> D",
	}, "\n")

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试6：回环
func TestDiagramRenderer_Cycle(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddEdge("A", "B", "--> ")
	renderer.AddEdge("B", "A", "--> ")

	result := renderer.Render()
	expected := "A --> B --> A 🔁"

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

func TestDiagramRenderer_Cycle2(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddEdge("B", "A", "--> ")
	renderer.AddEdge("A", "B", "--> ")

	result := renderer.Render()
	expected := "B --> A --> B 🔁"

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试7：复杂工作流
func TestDiagramRenderer_ComplexWorkflow(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddEdge("open", "pending", "--> ")
	renderer.AddEdge("pending", "resolved", "--> ")
	renderer.AddEdge("pending", "rejected", "--> ")
	renderer.AddEdge("resolved", "closed", "--> ")
	renderer.AddEdge("rejected", "open", "--> ")

	result := renderer.Render()
	expected := strings.Join([]string{
		"                    ┌--> resolved --> closed",
		"                    │",
		"open --> pending -->┤",
		"                    │",
		"                    └--> rejected --> open 🔁",
	}, "\n")

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试8：审批流转 (Using AddApprovalTransition for Legacy Style)
func TestDiagramRenderer_ApprovalTransition(t *testing.T) {
	renderer := NewDiagramRenderer()
	AddApprovalTransition(renderer, "Draft", "Reviewing", "Published", "Draft")

	result := renderer.Render()
	// Legacy style: "Draft --> Reviewing (via)"
	expected := strings.Join([]string{
		"         ┌-- <Commit> --> Published",
		"         │",
		"Draft --> Reviewing (via)",
		"         │",
		"         └-- <Reject> --> Draft 🔁",
	}, "\n")

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试9：审批流转带后续
func TestDiagramRenderer_ApprovalWithContinuation(t *testing.T) {
	renderer := NewDiagramRenderer()
	AddApprovalTransition(renderer, "Draft", "Reviewing", "Published", "Draft")
	renderer.AddEdge("Published", "Archived", "--> ")

	result := renderer.Render()
	expected := strings.Join([]string{
		"         ┌-- <Commit> --> Published --> Archived",
		"         │",
		"Draft --> Reviewing (via)",
		"         │",
		"         └-- <Reject> --> Draft 🔁",
	}, "\n")

	if result != expected {
		t.Errorf("Expected:\n%q\n\nGot:\n%q", expected, result)
	}
}

func TestDiagramRenderer_ApprovalWithContinuation2(t *testing.T) {
	renderer := NewDiagramRenderer()
	AddApprovalTransition(renderer, "Draft", "Reviewing", "Published", "Draft")
	renderer.SetJunction("Draft", "xx", "right")
	renderer.AddEdge("Published", "Archived", "--> ")

	result := renderer.Render()
	expected := strings.Join([]string{
		"                           ┌-- <Commit> --> Published --> Archived",
		"                           │",
		"Draft -->xx Reviewing (via)",
		"                           │",
		"                           └-- <Reject> --> Draft 🔁",
	}, "\n")

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试：居中对齐分叉
func TestDiagramRenderer_ApprovalWithContinuation3_Center(t *testing.T) {
	renderer := NewDiagramRenderer()
	AddApprovalTransition(renderer, "Draft", "Reviewing", "Published", "Draft")
	renderer.SetJunction("Draft", "xx", "center")
	renderer.AddEdge("Published", "Archived", "--> ")

	result := renderer.Render()

	// Calculation:
	// "Draft -->" (9) + "xx" (2) = 11
	// Center Content: " Reviewing (via)" (label " " + content)
	// Label " " width 1.
	// Content "Reviewing (via)" width 15.
	// Total center branch added width = 1 + 15 = 16.

	// IndentRef += stemSymbol ("xx") -> 11 chars base.
	// IndentRef += centerLabel (" ") -> 12 chars.
	// IndentRef += spaces(halfWidth of content). 15/2 = 7.
	// Total indent = 12 + 7 = 19 spaces.

	expected := strings.Join([]string{
		"                 ┌-- <Commit> --> Published --> Archived",
		"                 │",
		"Draft -->xx Reviewing (via)",
		"                 │",
		"                 └-- <Reject> --> Draft 🔁",
	}, "\n")

	fmt.Println(result)
	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

func TestDiagramRenderer_CustomSymbols(t *testing.T) {
	renderer := NewDiagramRenderer()

	// Case 1: Odd branches (3) - Top/Bottom=Corner, Middle=Stem
	renderer.AddNode("A", "NodeA")
	renderer.SetJunction("A", "*", "") // Stem
	renderer.SetCorner("A", "@", "@")  // Top/Bottom
	renderer.SetIntersection("A", "%") // Intermediate (won't appear in 3 branches)

	renderer.AddEdge("A", "B", "--> ")
	renderer.AddEdge("A", "C", "--> ")
	renderer.AddEdge("A", "D", "--> ")

	result := renderer.Render()
	// B (Top) -> Corner @
	// C (Mid) -> Stem *
	// D (Bot) -> Corner @
	expected := strings.Join([]string{
		"         @--> B",
		"         │",
		"NodeA -->*--> C",
		"         │",
		"         @--> D",
	}, "\n")

	if result != expected {
		t.Errorf("Odd Branches Mismatch!\nExpected:\n%q\n\nGot:\n%q", expected, result)
	}

	// Case 2: Even branches (4) - Top/Bot=Corner, Mids=Inter, Stem=Junction
	renderer2 := NewDiagramRenderer()
	renderer2.AddNode("X", "NodeX")
	renderer2.SetJunction("X", "*", "")
	renderer2.SetCorner("X", "@", "@")
	renderer2.SetIntersection("X", "%")

	renderer2.AddEdge("X", "1", "--> ")
	renderer2.AddEdge("X", "2", "--> ")
	renderer2.AddEdge("X", "3", "--> ")
	renderer2.AddEdge("X", "4", "--> ")

	result2 := renderer2.Render()
	// 1 (Top) -> Corner @
	// 2 (Mid) -> Inter %
	// Stem    -> *
	// 3 (Mid) -> Inter %
	// 4 (Bot) -> Corner @
	expected2 := strings.Join([]string{
		"         @--> 1",
		"         │",
		"         %--> 2",
		"         │",
		"NodeX -->*", // Stem connector
		"         │",
		"         %--> 3",
		"         │",
		"         @--> 4",
	}, "\n")

	if result2 != expected2 {
		t.Errorf("Even Branches Mismatch!\nExpected:\n%q\n\nGot:\n%q", expected2, result2)
	}
}

// 测试：分别设置 Top 和 Bottom Corner (Test Separate Corners)
func TestDiagramRenderer_SplitCorners(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddNode("A", "Root")
	renderer.SetCornerTop("A", "T")
	renderer.SetCornerBottom("A", "B")

	renderer.AddEdge("A", "1", "--> ")
	renderer.AddEdge("A", "2", "--> ")
	renderer.AddEdge("A", "3", "--> ")

	result := renderer.Render()
	// 1 (Top) -> T
	// 2 (Mid) -> + (default stem/inter)
	// 3 (Bot) -> B
	expected := strings.Join([]string{
		"        T--> 1",
		"        │",
		"Root -->┤--> 2",
		"        │",
		"        B--> 3",
	}, "\n")

	if result != expected {
		t.Errorf("Split Corners Mismatch!\nExpected:\n%q\n\nGot:\n%q", expected, result)
	}
}

// 测试10：空渲染器
func TestDiagramRenderer_Empty(t *testing.T) {
	renderer := NewDiagramRenderer()

	result := renderer.Render()
	if result != "" {
		t.Errorf("Expected empty string, got:\n%s", result)
	}

	comment := renderer.RenderAsComment()
	if comment != "" {
		t.Errorf("Expected empty comment, got:\n%s", comment)
	}
}

// 测试11：RenderAsComment
func TestDiagramRenderer_RenderAsComment(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddEdge("Init", "Done", "--> ")

	result := renderer.RenderAsComment()
	expected := strings.Join([]string{
		"/* Flowchart:",
		"Init --> Done",
		"*/",
		"",
	}, "\n")

	if result != expected {
		t.Errorf("Expected:\n%q\n\nGot:\n%q", expected, result)
	}
}

// 测试13：单个直接流转
func TestDiagramRenderer_SingleTransition(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddEdge("Init", "Running", "--> ")

	result := renderer.Render()
	expected := "Init --> Running"

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试14：单个终态
func TestDiagramRenderer_SingleTerminal(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddEdge("Start", "End", "--> ")

	result := renderer.Render()
	expected := "Start --> End"

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试：10层深度节点展开，带审批流转
// Re-implemented using generic API edges
func TestDiagramRenderer_DeepWithApproval(t *testing.T) {
	renderer := NewDiagramRenderer()

	// Layer 1
	renderer.AddEdge("Start", "L1", "--> ")

	// Layer 2: L1 -> L2A, L2B
	renderer.AddEdge("L1", "L2A", "--> ")
	renderer.AddEdge("L1", "L2B", "--> ")

	// Layer 3: L2A (Approval)
	AddApprovalTransition(renderer, "L2A", "L2A_Review", "L3A", "L2A")

	// Layer 3: L2B -> L3B, L3C
	renderer.AddEdge("L2B", "L3B", "--> ")
	renderer.AddEdge("L2B", "L3C", "--> ")

	// Layer 4: L3A -> L4A, L4B
	renderer.AddEdge("L3A", "L4A", "--> ")
	renderer.AddEdge("L3A", "L4B", "--> ")

	// Layer 4: L3B (Approval)
	AddApprovalTransition(renderer, "L3B", "L3B_Review", "L4C", "L3B")

	// Layer 4: L3C -> L4D
	renderer.AddEdge("L3C", "L4D", "--> ")

	// Layer 5
	renderer.AddEdge("L4A", "L5A", "--> ")
	renderer.AddEdge("L4B", "L5B", "--> ")
	renderer.AddEdge("L4C", "L5C", "--> ")
	renderer.AddEdge("L4D", "L5D", "--> ")

	// Layer 6
	renderer.AddEdge("L5A", "L6A", "--> ")
	renderer.AddEdge("L5B", "L6B", "--> ")
	renderer.AddEdge("L5C", "L6C", "--> ")
	renderer.AddEdge("L5D", "L6D", "--> ")

	// Layer 7: L6A -> L7A, L7B
	renderer.AddEdge("L6A", "L7A", "--> ")
	renderer.AddEdge("L6A", "L7B", "--> ")

	// Layer 7: L6B (Approval)
	AddApprovalTransition(renderer, "L6B", "L6B_Review", "L7C", "L6B")

	// Layer 7: L6C -> L7D, L6D -> L7E
	renderer.AddEdge("L6C", "L7D", "--> ")
	renderer.AddEdge("L6D", "L7E", "--> ")

	// Layer 8
	renderer.AddEdge("L7A", "L8A", "--> ")
	renderer.AddEdge("L7B", "L8B", "--> ")
	renderer.AddEdge("L7C", "L8C", "--> ")
	renderer.AddEdge("L7D", "L8D", "--> ")
	renderer.AddEdge("L7E", "L8E", "--> ")

	// Layer 9
	renderer.AddEdge("L8A", "L9A", "--> ")
	renderer.AddEdge("L8B", "L9B", "--> ")
	renderer.AddEdge("L8C", "L9C", "--> ")
	renderer.AddEdge("L8D", "L9D", "--> ")
	renderer.AddEdge("L8E", "L9E", "--> ")

	// Layer 10: End
	renderer.AddEdge("L9A", "End", "--> ")
	renderer.AddEdge("L9B", "End", "--> ")
	renderer.AddEdge("L9C", "End", "--> ")
	renderer.AddEdge("L9D", "End", "--> ")
	renderer.AddEdge("L9E", "End", "--> ")

	result := renderer.Render()

	expected := strings.Join([]string{
		"                                                                                ┌--> L7A --> L8A --> L9A --> End",
		"                                                                                │",
		"                                                    ┌--> L4A --> L5A --> L6A -->┤",
		"                                                    │                           │",
		"                                                    │                           └--> L7B --> L8B --> L9B --> End",
		"                            ┌-- <Commit> --> L3A -->┤",
		"                            │                       │                           ┌-- <Commit> --> L7C --> L8C --> L9C --> End",
		"                            │                       │                           │",
		"                            │                       └--> L4B --> L5B --> L6B --> L6B_Review (via)",
		"                            │                                                   │",
		"                            │                                                   └-- <Reject> --> L6B 🔁",
		"                            │",
		"                ┌--> L2A --> L2A_Review (via)",
		"                │           │",
		"                │           │",
		"                │           │",
		"                │           │",
		"                │           │",
		"                │           │",
		"                │           └-- <Reject> --> L2A 🔁",
		"Start --> L1 -->┤",
		"                │",
		"                │",
		"                │                       ┌-- <Commit> --> L4C --> L5C --> L6C --> L7D --> L8D --> L9D --> End",
		"                │                       │",
		"                │           ┌--> L3B --> L3B_Review (via)",
		"                │           │           │",
		"                │           │           └-- <Reject> --> L3B 🔁",
		"                └--> L2B -->┤",
		"                            │",
		"                            │",
		"                            └--> L3C --> L4D --> L5D --> L6D --> L7E --> L8E --> L9E --> End",
	}, "\n")

	if result != expected {
		diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(expected),
			B:        difflib.SplitLines(result),
			FromFile: "Original",
			ToFile:   "Current",
		})
		fmt.Print(diff)
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试：间接环路不会导致指数级膨胀
// A ↔ B 通过中间节点形成间接环，深度限制应阻止无限递归
func TestDiagramRenderer_IndirectCycle(t *testing.T) {
	renderer := NewDiagramRenderer()
	// A -> B, A -> C
	// B -> A (indirect cycle back)
	// C -> A (indirect cycle back)
	renderer.AddEdge("A", "B", "--> ")
	renderer.AddEdge("A", "C", "--> ")
	renderer.AddEdge("B", "A", "--> ")
	renderer.AddEdge("C", "A", "--> ")

	result := renderer.Render()

	// 确保输出有限且合理
	lines := strings.Split(result, "\n")
	if len(lines) > 50 {
		t.Errorf("Output too large (%d lines), likely infinite recursion", len(lines))
	}
	// 确保包含回环标记
	if !strings.Contains(result, "🔁") {
		t.Errorf("Expected loop marker in output, got:\n%s", result)
	}
}

// 测试：模拟 stateflowgen 中 active/inactive 自环场景
func TestDiagramRenderer_SelfTransitionViaApproval(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.ArrowSymbol = "──"

	// 模拟: active => [ inactive? via updating, (=)? via updating ]
	// active -> decision -> updating_via -> COMMIT -> inactive
	//                                    -> COMMIT -> active (self)
	//                                    -> REJECT -> active (fallback)
	//                    -> inactive
	//                    -> active (direct)

	renderer.AddNode("active_decision", "<?APPROVAL?>")
	renderer.AddEdge("active", "active_decision", "──▶ ")

	renderer.AddNode("active_updating_via", "updating (via)")
	renderer.AddEdge("active_decision", "active_updating_via", "──▶ ")
	renderer.AddEdge("active_updating_via", "inactive", "── <COMMIT> ──▶ ")
	renderer.AddNode("active_fallback", "active 🔁")
	renderer.AddEdge("active_updating_via", "active_fallback", "── <REJECT> ──▶ ")
	renderer.AddEdge("active_updating_via", "active", "── <COMMIT> ──▶ ")
	renderer.AddEdge("active_updating_via", "active_fallback", "── <REJECT> ──▶ ")
	renderer.AddEdge("active_decision", "active", "──▶ ")

	// inactive has similar transitions back
	renderer.AddNode("inactive_decision", "<?APPROVAL?>")
	renderer.AddEdge("inactive", "inactive_decision", "──▶ ")
	renderer.AddNode("inactive_updating_via", "updating (via)")
	renderer.AddEdge("inactive_decision", "inactive_updating_via", "──▶ ")
	renderer.AddEdge("inactive_updating_via", "active", "── <COMMIT> ──▶ ")
	renderer.AddNode("inactive_fallback", "inactive 🔁")
	renderer.AddEdge("inactive_updating_via", "inactive_fallback", "── <REJECT> ──▶ ")
	renderer.AddEdge("inactive_updating_via", "inactive", "── <COMMIT> ──▶ ")
	renderer.AddEdge("inactive_updating_via", "inactive_fallback", "── <REJECT> ──▶ ")
	renderer.AddEdge("inactive_decision", "inactive", "──▶ ")

	result := renderer.Render()
	lines := strings.Split(result, "\n")

	// 关键断言：输出应该有限（不超过100行）
	if len(lines) > 100 {
		t.Errorf("Output too large (%d lines), depth limiting failed", len(lines))
	}
	t.Logf("Output has %d lines (should be bounded)", len(lines))
}

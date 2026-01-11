package stateflowgen

import (
	"strings"
	"testing"
)

// 测试1：简单线性流程
func TestDiagramRenderer_SimpleLinear(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddDirectTransition("A", "B")
	renderer.AddDirectTransition("B", "C")

	result := renderer.Render()
	expected := `A --> B --> C`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试2：两分支（偶数分支，中心在分隔行）
func TestDiagramRenderer_TwoBranches(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddDirectTransition("A", "B")
	renderer.AddDirectTransition("A", "C")

	result := renderer.Render()
	// 2个分支：B, |, C = 3行，中心行=1，在分隔符|上
	expected := `     +--> B
A -->+
     +--> C`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试3：三分支（奇数分支，中心在中间分支）
func TestDiagramRenderer_ThreeBranches(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddDirectTransition("A", "B")
	renderer.AddDirectTransition("A", "C")
	renderer.AddDirectTransition("A", "D")

	result := renderer.Render()
	// 3个分支：B, |, C, |, D = 5行，中心行=2，在C分支上
	expected := `     +--> B
     |
A -->+--> C
     |
     +--> D`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试4：四分支
func TestDiagramRenderer_FourBranches(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddDirectTransition("A", "B")
	renderer.AddDirectTransition("A", "C")
	renderer.AddDirectTransition("A", "D")
	renderer.AddDirectTransition("A", "E")

	result := renderer.Render()
	// 4个分支：B, |, C, |, D, |, E = 7行，中心行=3，在D分隔符|上
	expected := `     +--> B
     |
     +--> C
A -->+
     +--> D
     |
     +--> E`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试5：嵌套分支
func TestDiagramRenderer_NestedBranches(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddDirectTransition("A", "B")
	renderer.AddDirectTransition("B", "C")
	renderer.AddDirectTransition("B", "D")
	renderer.AddDirectTransition("C", "E")

	result := renderer.Render()
	// B有两个分支：C-->E, D。总3行，中心行=1在分隔符上
	// "A --> B -->" = 11字符
	expected := `           +--> C --> E
A --> B -->+
           +--> D`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试6：回环
func TestDiagramRenderer_Cycle(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddDirectTransition("A", "B")
	renderer.AddDirectTransition("B", "A")

	result := renderer.Render()
	expected := `A --> B --> A 🔁`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试7：复杂工作流（设计文档示例）
func TestDiagramRenderer_ComplexWorkflow(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddDirectTransition("open", "pending")
	renderer.AddDirectTransition("pending", "resolved")
	renderer.AddDirectTransition("pending", "rejected")
	renderer.AddDirectTransition("resolved", "closed")
	renderer.AddDirectTransition("rejected", "open")

	result := renderer.Render()
	// pending有两个分支：resolved-->closed, rejected-->open🔁，3行，中心=1
	// "open --> pending -->" = 20字符
	expected := `                    +--> resolved --> closed
open --> pending -->+
                    +--> rejected --> open 🔁`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试8：审批流转
func TestDiagramRenderer_ApprovalTransition(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddApprovalTransition("Draft", "Reviewing", "Published", "Draft")

	result := renderer.Render()
	expected := `          +-- <Commit> --> Published
          |
Draft --> Reviewing (via)
          |
          +-- <Reject> --> Draft 🔁`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试9：审批流转带后续流程
func TestDiagramRenderer_ApprovalWithContinuation(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddApprovalTransition("Draft", "Reviewing", "Published", "Draft")
	renderer.AddDirectTransition("Published", "Archived")

	result := renderer.Render()
	expected := `          +-- <Commit> --> Published --> Archived
          |
Draft --> Reviewing (via)
          |
          +-- <Reject> --> Draft 🔁`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
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
	renderer.AddDirectTransition("Init", "Done")

	result := renderer.RenderAsComment()
	expected := `// State Flow Diagram:
// ` + "```" + `
// Init --> Done
// ` + "```" + `
`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试12：嵌套分支的子分支也有多个目标
func TestDiagramRenderer_DeepNestedBranches(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddDirectTransition("A", "B")
	renderer.AddDirectTransition("B", "C")
	renderer.AddDirectTransition("B", "D")
	renderer.AddDirectTransition("C", "E")
	renderer.AddDirectTransition("C", "F")

	result := renderer.Render()
	t.Logf("Deep Nested Branches output:\n%s", result)
}

// TestDiagramRenderer_AllScenarios 展示所有场景的输出
func TestDiagramRenderer_AllScenarios(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*DiagramRenderer)
	}{
		{
			name: "简单线性 A->B->C",
			setup: func(r *DiagramRenderer) {
				r.AddDirectTransition("A", "B")
				r.AddDirectTransition("B", "C")
			},
		},
		{
			name: "两分支",
			setup: func(r *DiagramRenderer) {
				r.AddDirectTransition("A", "B")
				r.AddDirectTransition("A", "C")
			},
		},
		{
			name: "三分支",
			setup: func(r *DiagramRenderer) {
				r.AddDirectTransition("A", "B")
				r.AddDirectTransition("A", "C")
				r.AddDirectTransition("A", "D")
			},
		},
		{
			name: "四分支",
			setup: func(r *DiagramRenderer) {
				r.AddDirectTransition("A", "B")
				r.AddDirectTransition("A", "C")
				r.AddDirectTransition("A", "D")
				r.AddDirectTransition("A", "E")
			},
		},
		{
			name: "嵌套分支",
			setup: func(r *DiagramRenderer) {
				r.AddDirectTransition("A", "B")
				r.AddDirectTransition("B", "C")
				r.AddDirectTransition("B", "D")
				r.AddDirectTransition("C", "E")
			},
		},
		{
			name: "深层嵌套",
			setup: func(r *DiagramRenderer) {
				r.AddDirectTransition("A", "B")
				r.AddDirectTransition("B", "C")
				r.AddDirectTransition("B", "D")
				r.AddDirectTransition("C", "E")
				r.AddDirectTransition("C", "F")
			},
		},
		{
			name: "回环",
			setup: func(r *DiagramRenderer) {
				r.AddDirectTransition("A", "B")
				r.AddDirectTransition("B", "A")
			},
		},
		{
			name: "复杂工作流 open->pending->(resolved->closed, rejected->open)",
			setup: func(r *DiagramRenderer) {
				r.AddDirectTransition("open", "pending")
				r.AddDirectTransition("pending", "resolved")
				r.AddDirectTransition("pending", "rejected")
				r.AddDirectTransition("resolved", "closed")
				r.AddDirectTransition("rejected", "open")
			},
		},
		{
			name: "审批流转",
			setup: func(r *DiagramRenderer) {
				r.AddApprovalTransition("Draft", "Reviewing", "Published", "Draft")
			},
		},
		{
			name: "审批流转带后续",
			setup: func(r *DiagramRenderer) {
				r.AddApprovalTransition("Draft", "Reviewing", "Published", "Draft")
				r.AddDirectTransition("Published", "Archived")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewDiagramRenderer()
			tt.setup(renderer)
			result := renderer.Render()
			t.Logf("\n=== %s ===\n%s\n", tt.name, result)
		})
	}
}

// 测试：10层深度节点展开，带审批流转
func TestDiagramRenderer_DeepWithApproval(t *testing.T) {
	renderer := NewDiagramRenderer()

	// 构建10层深度的流程，中间夹带审批
	// Layer 1: Start
	renderer.AddDirectTransition("Start", "L1")

	// Layer 2: L1 分支
	renderer.AddDirectTransition("L1", "L2A")
	renderer.AddDirectTransition("L1", "L2B")

	// Layer 3: L2A 需要审批
	renderer.AddApprovalTransition("L2A", "L2A_Review", "L3A", "L2A")

	// Layer 3: L2B 普通分支
	renderer.AddDirectTransition("L2B", "L3B")
	renderer.AddDirectTransition("L2B", "L3C")

	// Layer 4: L3A 继续分支
	renderer.AddDirectTransition("L3A", "L4A")
	renderer.AddDirectTransition("L3A", "L4B")

	// Layer 4: L3B 需要审批
	renderer.AddApprovalTransition("L3B", "L3B_Review", "L4C", "L3B")

	// Layer 4: L3C 普通
	renderer.AddDirectTransition("L3C", "L4D")

	// Layer 5: L4A, L4B, L4C, L4D
	renderer.AddDirectTransition("L4A", "L5A")
	renderer.AddDirectTransition("L4B", "L5B")
	renderer.AddDirectTransition("L4C", "L5C")
	renderer.AddDirectTransition("L4D", "L5D")

	// Layer 6
	renderer.AddDirectTransition("L5A", "L6A")
	renderer.AddDirectTransition("L5B", "L6B")
	renderer.AddDirectTransition("L5C", "L6C")
	renderer.AddDirectTransition("L5D", "L6D")

	// Layer 7: L6A 分支
	renderer.AddDirectTransition("L6A", "L7A")
	renderer.AddDirectTransition("L6A", "L7B")

	// Layer 7: L6B 审批
	renderer.AddApprovalTransition("L6B", "L6B_Review", "L7C", "L6B")

	// Layer 7: L6C, L6D
	renderer.AddDirectTransition("L6C", "L7D")
	renderer.AddDirectTransition("L6D", "L7E")

	// Layer 8
	renderer.AddDirectTransition("L7A", "L8A")
	renderer.AddDirectTransition("L7B", "L8B")
	renderer.AddDirectTransition("L7C", "L8C")
	renderer.AddDirectTransition("L7D", "L8D")
	renderer.AddDirectTransition("L7E", "L8E")

	// Layer 9
	renderer.AddDirectTransition("L8A", "L9A")
	renderer.AddDirectTransition("L8B", "L9B")
	renderer.AddDirectTransition("L8C", "L9C")
	renderer.AddDirectTransition("L8D", "L9D")
	renderer.AddDirectTransition("L8E", "L9E")

	// Layer 10: 终态
	renderer.AddDirectTransition("L9A", "End")
	renderer.AddDirectTransition("L9B", "End")
	renderer.AddDirectTransition("L9C", "End")
	renderer.AddDirectTransition("L9D", "End")
	renderer.AddDirectTransition("L9E", "End")

	result := renderer.Render()
	expected := strings.Join([]string{
		"                                                                                   +--> L7A --> L8A --> L9A --> End",
		"                                                                                   |",
		"                                                      +--> L4A --> L5A --> L6A -->+",
		"                                                      |                            |",
		"                                                      |                            +--> L7B --> L8B --> L9B --> End",
		"                              +-- <Commit> --> L3A -->+",
		"                              |                        |                             +-- <Commit> --> L7C --> L8C --> L9C --> End",
		"                              |                        |                             |",
		"                              |                        +--> L4B --> L5B --> L6B --> L6B_Review (via)",
		"                              |                                                      |",
		"                              |                                                      +-- <Reject> --> L6B 🔁",
		"                              |",
		"                +--> L2A --> L2A_Review (via)",
		"                |             |",
		"                |             |",
		"                |             |",
		"                |             |",
		"                |             |",
		"                |             |",
		"                |             +-- <Reject> --> L2A 🔁",
		"Start --> L1 -->+",
		"                |",
		"                |                          +-- <Commit> --> L4C --> L5C --> L6C --> L7D --> L8D --> L9D --> End",
		"                |                          |",
		"                |            +--> L3B --> L3B_Review (via)",
		"                |            |             |",
		"                |            |             +-- <Reject> --> L3B 🔁",
		"                |            |",
		"                +--> L2B -->+",
		"                             |",
		"                             |",
		"                             |",
		"                             +--> L3C --> L4D --> L5D --> L6D --> L7E --> L8E --> L9E --> End",
	}, "\n")

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试13：单个直接流转
func TestDiagramRenderer_SingleTransition(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddDirectTransition("Init", "Running")

	result := renderer.Render()
	expected := `Init --> Running`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试14：单个终态（没有后续）
func TestDiagramRenderer_SingleTerminal(t *testing.T) {
	renderer := NewDiagramRenderer()
	renderer.AddDirectTransition("Start", "End")

	result := renderer.Render()
	expected := `Start --> End`

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

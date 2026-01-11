package stateflowgen

import (
	"strings"
	"testing"
)

// 测试自定义箭头符号
func TestDiagramRenderer_CustomArrow(t *testing.T) {
	renderer := NewDiagramRenderer()

	// 设置自定义箭头符号
	renderer.SetArrowSymbol(" => ")

	renderer.AddEdge("Start", "Middle", "")
	renderer.AddEdge("Middle", "End", "")

	result := renderer.Render()
	expected := "Start => Middle => End"

	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

// 测试自定义箭头符号 - 分支场景
func TestDiagramRenderer_CustomArrowWithBranches(t *testing.T) {
	renderer := NewDiagramRenderer()

	// 使用 Unicode 箭头
	renderer.SetArrowSymbol(" ──> ")

	renderer.AddEdge("A", "B", "")
	renderer.AddEdge("A", "C", "")

	result := renderer.Render()
	expected := strings.Join([]string{
		"     +──> B",
		"     │",
		"A ──>+",
		"     │",
		"     +──> C",
	}, "\n")

	if result != expected {
		t.Errorf("Expected:\n%q\n\nGot:\n%q", expected, result)
	}
}

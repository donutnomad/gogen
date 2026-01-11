package stateflowgen

import (
	"strings"
	"testing"
)

// 测试各种 Unicode 箭头符号的宽度计算
func TestDiagramRenderer_UnicodeWidth(t *testing.T) {
	tests := []struct {
		name   string
		arrow  string
		expect string
	}{
		{
			name:  "ASCII arrow",
			arrow: " --> ",
			expect: strings.Join([]string{
				"     +--> B",
				"     │",
				"A -->+",
				"     │",
				"     +--> C",
			}, "\n"),
		},
		{
			name:  "Unicode box arrow",
			arrow: " ──> ",
			expect: strings.Join([]string{
				"     +──> B",
				"     │",
				"A ──>+",
				"     │",
				"     +──> C",
			}, "\n"),
		},
		{
			name:  "Unicode right arrow",
			arrow: " → ",
			expect: strings.Join([]string{
				"   +→ B",
				"   │",
				"A →+",
				"   │",
				"   +→ C",
			}, "\n"),
		},
		{
			name:  "Unicode double arrow",
			arrow: " ⇒ ",
			expect: strings.Join([]string{
				"   +⇒ B",
				"   │",
				"A ⇒+",
				"   │",
				"   +⇒ C",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewDiagramRenderer()
			renderer.SetArrowSymbol(tt.arrow)
			renderer.AddEdge("A", "B", "")
			renderer.AddEdge("A", "C", "")

			result := renderer.Render()
			if result != tt.expect {
				t.Errorf("Arrow %q failed:\nExpected:\n%q\n\nGot:\n%q", tt.arrow, tt.expect, result)
			}
		})
	}
}

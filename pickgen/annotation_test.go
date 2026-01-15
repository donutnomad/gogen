package pickgen

import (
	"testing"

	"github.com/donutnomad/gogen/plugin"
)

func TestAnnotationParsing(t *testing.T) {
	tests := []struct {
		input      string
		wantName   string
		wantFields string
	}{
		// 使用反引号包裹数组参数，避免逗号被错误解析
		{"@Pick(name=UserBasic, fields=`[ID,Name,Email]`)", "Pick", "[ID,Name,Email]"},
		{"@Omit(name=UserPublic, fields=`[Password,Salt]`)", "Omit", "[Password,Salt]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			anns := plugin.ParseAnnotations(tt.input)
			if len(anns) != 1 {
				t.Errorf("expected 1 annotation, got %d", len(anns))
				return
			}
			if anns[0].Name != tt.wantName {
				t.Errorf("name = %q, want %q", anns[0].Name, tt.wantName)
			}
			if got := anns[0].GetParam("fields"); got != tt.wantFields {
				t.Errorf("fields = %q, want %q", got, tt.wantFields)
			}
		})
	}
}

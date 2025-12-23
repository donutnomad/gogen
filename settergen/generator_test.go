package settergen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAllMethodsFromFile(t *testing.T) {
	exampleFile := filepath.Join(".", "example", "models.go")

	methods := parseAllMethodsFromFile(exampleFile)

	if len(methods) != 2 {
		t.Errorf("parseAllMethodsFromFile() got %d methods, want 2", len(methods))
	}

	// 检查是否包含预期的方法
	methodNames := make(map[string]bool)
	for _, m := range methods {
		methodNames[m.Name] = true
	}

	if !methodNames["ToPO"] {
		t.Error("parseAllMethodsFromFile() missing ToPO method")
	}
	if !methodNames["ToArticlePO"] {
		t.Error("parseAllMethodsFromFile() missing ToArticlePO method")
	}
}

func TestTrimPtr(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"*User", "User"},
		{"User", "User"},
		{"**User", "*User"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := trimPtr(tt.input); got != tt.want {
				t.Errorf("trimPtr(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewSetterGenerator(t *testing.T) {
	g := NewSetterGenerator()
	if g == nil {
		t.Error("NewSetterGenerator() returned nil")
	}
}

// 辅助函数：创建临时测试目录
func createTempTestDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "setterg_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

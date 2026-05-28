package settergen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donutnomad/gg"
)

// TestBugReproOnlyPatchV2 重现 setter=false, patch=v2 时的 bug
// 模拟 automap 返回带 import 引用的 funcCode 场景
func TestBugReproOnlyPatchV2(t *testing.T) {
	gen := gg.New()
	gen.SetPackage("emailroutingrepo")

	// 模拟 automap 返回的数据
	funcCode := `// Missing fields: default_id, deleted_at
func (e *EmailRoutingPO) ToPatch(input *domain.EmailRouting) map[string]any {
	b := e.ToPO(input)
	fields := input.ExportPatch()
	values := make(map[string]any, 9)
	// Embedded: Model
	if fields.ID.IsPresent() {
		values["id"] = b.Model.ID
	}
	return values
}
`
	// 模拟 imports 列表
	importPath := "github.com/example/emailrouting"
	alias := "domain"

	// 这是 settergen/generator.go 里的逻辑：先注册 import，再 AddString
	gen.PAlias(importPath, alias)
	gen.Body().AddString(funcCode)

	output := gen.String()
	fmt.Println("=== Generated Output ===")
	fmt.Println(output)

	if !strings.Contains(output, `import domain "github.com/example/emailrouting"`) {
		t.Errorf("生成的代码缺少 domain import\n输出:\n%s", output)
	}
	if !strings.Contains(output, `domain "github.com/example/emailrouting"`) {
		t.Errorf("生成的代码缺少 domain import\n输出:\n%s", output)
	}
}

// TestBugReproOnlyPatchV2_EmptyImports 模拟 imports 为空时的情况
func TestBugReproOnlyPatchV2_EmptyImports(t *testing.T) {
	gen := gg.New()
	gen.SetPackage("emailroutingrepo")

	funcCode := `// Missing fields: ...
func (e *EmailRoutingPO) ToPatch(input *domain.EmailRouting) map[string]any {
	return nil
}
`
	// 如果 imports 为空，不调用 gen.PAlias
	// 但 funcCode 里引用了 domain.EmailRouting
	gen.Body().AddString(funcCode)

	output := gen.String()
	fmt.Println("=== Output when imports empty ===")
	fmt.Println(output)

	// 此时应该没有 import 块，但 funcCode 里有 domain 引用
	// 这就是 bug 出现的场景
	if strings.Contains(output, "import (") {
		t.Log("有 import 块（预期外）")
	} else {
		t.Log("没有 import 块（可能是 bug 场景）")
	}
}

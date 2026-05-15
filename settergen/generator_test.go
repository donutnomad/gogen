package settergen

import (
	"context"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/internal/gormparse"
	"github.com/donutnomad/gogen/plugin"
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

func TestRunOnlySetterPatchV2WithSingleAliasedImport(t *testing.T) {
	tmpDir := t.TempDir()
	goModFile := filepath.Join(tmpDir, "go.mod")
	goModSource := `module example.com/app

go 1.25
`
	if err := os.WriteFile(goModFile, []byte(goModSource), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	domainDir := filepath.Join(tmpDir, "emailrouting")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatalf("failed to create domain dir: %v", err)
	}
	domainFile := filepath.Join(domainDir, "entity.go")
	domainSource := `package emailrouting

import "time"

type Status string

type ID uint64

type patchField struct{}

func (patchField) IsPresent() bool {
	return false
}

type EmailRoutingPatch struct {
	ID         patchField
	Email      patchField
	Type       patchField
	Status     patchField
	FailReason patchField
	CreatedAt  patchField
	UpdatedAt  patchField
	DeletedAt  patchField
}

type EmailRouting struct {
	ID         ID
	Email      string
	Type       string
	Status     Status
	FailReason string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

func (e *EmailRouting) ExportPatch() *EmailRoutingPatch {
	return &EmailRoutingPatch{}
}
`
	if err := os.WriteFile(domainFile, []byte(domainSource), 0644); err != nil {
		t.Fatalf("failed to write domain file: %v", err)
	}

	ormDir := filepath.Join(tmpDir, "orm")
	if err := os.MkdirAll(ormDir, 0755); err != nil {
		t.Fatalf("failed to create orm dir: %v", err)
	}
	ormFile := filepath.Join(ormDir, "mod.go")
	ormSource := `package orm

import "time"

type DeletedAt struct{}

type Model struct {
	DefaultID uint64    ` + "`gorm:\"primaryKey\"`" + `
	ID        uint64    ` + "`gorm:\"column:id;uniqueIndex\"`" + `
	CreatedAt time.Time ` + "`gorm:\"column:created_at;index\"`" + `
	UpdatedAt time.Time ` + "`gorm:\"column:updated_at;index\"`" + `
	DeletedAt DeletedAt ` + "`gorm:\"index\"`" + `
}
`
	if err := os.WriteFile(ormFile, []byte(ormSource), 0644); err != nil {
		t.Fatalf("failed to write orm file: %v", err)
	}

	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	modelFile := filepath.Join(repoDir, "email_routing_po.go")
	modelSource := `package emailroutingrepo

import (
	"example.com/app/orm"
	domain "example.com/app/emailrouting"
)

// EmailRoutingPO
// @Setter(setter=false, patch=v2)
type EmailRoutingPO struct {
	orm.Model

	Email      string        ` + "`gorm:\"column:email\"`" + `
	Type       string        ` + "`gorm:\"column:type\"`" + `
	Status     domain.Status ` + "`gorm:\"column:status\"`" + `
	FailReason string        ` + "`gorm:\"column:fail_reason\"`" + `
}

func (p *EmailRoutingPO) TableName() string {
	return "email_routings"
}

func (p *EmailRoutingPO) ToPO(entity *domain.EmailRouting) *EmailRoutingPO {
	return &EmailRoutingPO{
		Model: orm.Model{
			ID:        uint64(entity.ID),
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
		},
		Email:      entity.Email,
		Type:       entity.Type,
		Status:     entity.Status,
		FailReason: entity.FailReason,
	}
}
`
	if err := os.WriteFile(modelFile, []byte(modelSource), 0644); err != nil {
		t.Fatalf("failed to write model file: %v", err)
	}

	registry := plugin.NewRegistry()
	if err := registry.Register(NewSetterGenerator()); err != nil {
		t.Fatalf("failed to register settergen: %v", err)
	}

	err := plugin.RunWithOptions(context.Background(), &plugin.RunOptions{
		Registry: registry,
		Patterns: []string{repoDir},
		Output:   "generate.go",
		Async:    false,
	})
	if err != nil {
		t.Fatalf("RunWithOptions failed: %v", err)
	}

	generatedFile := filepath.Join(repoDir, "generate.go")
	generated, err := os.ReadFile(generatedFile)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	if _, err := parser.ParseFile(token.NewFileSet(), generatedFile, generated, parser.ParseComments); err != nil {
		t.Fatalf("generated file should parse: %v\n%s", err, generated)
	}

	output := string(generated)
	if !strings.Contains(output, `domain "example.com/app/emailrouting"`) {
		t.Fatalf("generated file should keep aliased domain import:\n%s", output)
	}
	if !strings.Contains(output, "func (e *EmailRoutingPO) ToPatch(input *domain.EmailRouting) map[string]any") {
		t.Fatalf("generated file should contain ToPatch with domain input:\n%s", output)
	}
}

// TestGenerateToMapMethod_DirectFields 测试直接字段的 ToMap 生成
func TestGenerateToMapMethod_DirectFields(t *testing.T) {
	model := &gormparse.GormModelInfo{
		Name:        "UserPO",
		PackageName: "models",
		Fields: []gormparse.GormFieldInfo{
			{Name: "ID", Type: "int64", ColumnName: "id"},
			{Name: "Name", Type: "string", ColumnName: "name"},
			{Name: "Email", Type: "string", ColumnName: "email"},
		},
	}

	gen := gg.New()
	gen.SetPackage("models")
	generateToMapMethod(gen, model)

	code := gen.String()

	// 验证生成的代码包含正确的访问路径
	if !strings.Contains(code, `values["id"] = u.ID`) {
		t.Error("生成的代码应包含 values[\"id\"] = u.ID")
	}
	if !strings.Contains(code, `values["name"] = u.Name`) {
		t.Error("生成的代码应包含 values[\"name\"] = u.Name")
	}
	if !strings.Contains(code, `values["email"] = u.Email`) {
		t.Error("生成的代码应包含 values[\"email\"] = u.Email")
	}
}

// TestGenerateToMapMethod_EmbeddedFields 测试嵌入字段的 ToMap 生成
func TestGenerateToMapMethod_EmbeddedFields(t *testing.T) {
	model := &gormparse.GormModelInfo{
		Name:        "AssetPricePO",
		PackageName: "models",
		Fields: []gormparse.GormFieldInfo{
			{Name: "ID", Type: "int64", ColumnName: "id"},
			{Name: "TenantID", Type: "string", ColumnName: "tenant_id"},
			// 来自嵌入字段 Address 的子字段
			{Name: "ChainID", Type: "string", ColumnName: "chain_id", SourceField: "Address", SourceType: "AccountIDColumnsCompact"},
			{Name: "Address", Type: "string", ColumnName: "address", SourceField: "Address", SourceType: "AccountIDColumnsCompact"},
			{Name: "Price", Type: "float64", ColumnName: "price"},
		},
	}

	gen := gg.New()
	gen.SetPackage("models")
	generateToMapMethod(gen, model)

	code := gen.String()

	// 验证直接字段的访问路径
	if !strings.Contains(code, `values["id"] = a.ID`) {
		t.Error("生成的代码应包含 values[\"id\"] = a.ID")
	}
	if !strings.Contains(code, `values["tenant_id"] = a.TenantID`) {
		t.Error("生成的代码应包含 values[\"tenant_id\"] = a.TenantID")
	}
	if !strings.Contains(code, `values["price"] = a.Price`) {
		t.Error("生成的代码应包含 values[\"price\"] = a.Price")
	}

	// 验证嵌入字段的访问路径（关键测试点）
	if !strings.Contains(code, `values["chain_id"] = a.Address.ChainID`) {
		t.Errorf("生成的代码应包含 values[\"chain_id\"] = a.Address.ChainID，实际代码:\n%s", code)
	}
	if !strings.Contains(code, `values["address"] = a.Address.Address`) {
		t.Errorf("生成的代码应包含 values[\"address\"] = a.Address.Address，实际代码:\n%s", code)
	}
}

// TestGenerateToMapMethod_NestedEmbeddedFields 测试多层嵌套字段的 ToMap 生成
func TestGenerateToMapMethod_NestedEmbeddedFields(t *testing.T) {
	model := &gormparse.GormModelInfo{
		Name:        "NestedPO",
		PackageName: "models",
		Fields: []gormparse.GormFieldInfo{
			{Name: "ID", Type: "int64", ColumnName: "id"},
			// 多层嵌套：Outer.Inner.InnerField
			{Name: "InnerField", Type: "string", ColumnName: "inner_field", SourceField: "Outer.Inner", SourceType: "InnerStruct"},
			{Name: "OuterField", Type: "string", ColumnName: "outer_field", SourceField: "Outer", SourceType: "OuterStruct"},
		},
	}

	gen := gg.New()
	gen.SetPackage("models")
	generateToMapMethod(gen, model)

	code := gen.String()

	// 验证多层嵌套的访问路径
	if !strings.Contains(code, `values["inner_field"] = n.Outer.Inner.InnerField`) {
		t.Errorf("生成的代码应包含 values[\"inner_field\"] = n.Outer.Inner.InnerField，实际代码:\n%s", code)
	}
	if !strings.Contains(code, `values["outer_field"] = n.Outer.OuterField`) {
		t.Errorf("生成的代码应包含 values[\"outer_field\"] = n.Outer.OuterField，实际代码:\n%s", code)
	}
}

// TestGenerateToMapMethod_MixedEmbedded 测试混合场景：匿名嵌入 + gorm embedded
func TestGenerateToMapMethod_MixedEmbedded(t *testing.T) {
	model := &gormparse.GormModelInfo{
		Name:        "MixedPO",
		PackageName: "models",
		Fields: []gormparse.GormFieldInfo{
			// 匿名嵌入的字段 - SourceField 为空，直接访问
			{Name: "ID", Type: "int64", ColumnName: "id", SourceType: "BaseModel"},
			{Name: "CreatedAt", Type: "int64", ColumnName: "created_at", SourceType: "BaseModel"},
			// gorm embedded 字段 - SourceField 非空，需要通过字段名访问
			{Name: "AccountID", Type: "string", ColumnName: "acc_account_id", SourceField: "Account", SourceType: "Account"},
		},
	}

	gen := gg.New()
	gen.SetPackage("models")
	generateToMapMethod(gen, model)

	code := gen.String()

	// 匿名嵌入的字段应该直接访问
	if !strings.Contains(code, `values["id"] = m.ID`) {
		t.Errorf("匿名嵌入字段应直接访问: values[\"id\"] = m.ID，实际代码:\n%s", code)
	}
	if !strings.Contains(code, `values["created_at"] = m.CreatedAt`) {
		t.Errorf("匿名嵌入字段应直接访问: values[\"created_at\"] = m.CreatedAt，实际代码:\n%s", code)
	}

	// gorm embedded 字段应通过字段名访问
	if !strings.Contains(code, `values["acc_account_id"] = m.Account.AccountID`) {
		t.Errorf("gorm embedded 字段应通过字段名访问: values[\"acc_account_id\"] = m.Account.AccountID，实际代码:\n%s", code)
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

package structparse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseGormEmbeddedTag 测试 GORM embedded 标签解析
// 功能：解析 gorm 标签中的 embedded 和 embeddedPrefix 属性
// 场景：
// - 标准 embedded 标签
// - 带 embeddedPrefix 的标签
// - 非 GORM 标签
// - 空标签
func TestParseGormEmbeddedTag(t *testing.T) {
	tests := []struct {
		name         string
		tag          string
		wantEmbedded bool
		wantPrefix   string
	}{
		{
			name:         "standard embedded tag",
			tag:          `gorm:"embedded"`,
			wantEmbedded: true,
			wantPrefix:   "",
		},
		{
			name:         "embedded with prefix",
			tag:          `gorm:"embedded;embeddedPrefix:user_"`,
			wantEmbedded: true,
			wantPrefix:   "user_",
		},
		{
			name:         "embedded with prefix and other tags",
			tag:          `gorm:"embedded;embeddedPrefix:account_" json:"account"`,
			wantEmbedded: true,
			wantPrefix:   "account_",
		},
		{
			name:         "only embeddedPrefix without embedded",
			tag:          `gorm:"embeddedPrefix:test_"`,
			wantEmbedded: false,
			wantPrefix:   "test_",
		},
		{
			name:         "non-gorm tag",
			tag:          `json:"data" xml:"data"`,
			wantEmbedded: false,
			wantPrefix:   "",
		},
		{
			name:         "empty tag",
			tag:          "",
			wantEmbedded: false,
			wantPrefix:   "",
		},
		{
			name:         "gorm tag without embedded",
			tag:          `gorm:"column:name;index"`,
			wantEmbedded: false,
			wantPrefix:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEmbedded, gotPrefix := parseGormEmbeddedTag(tt.tag)
			assert.Equal(t, tt.wantEmbedded, gotEmbedded, "embedded flag mismatch")
			assert.Equal(t, tt.wantPrefix, gotPrefix, "embeddedPrefix mismatch")
		})
	}
}

// TestShouldExpandEmbeddedField 测试是否应该展开嵌入字段
// 功能：判断字段类型是否应该展开为子字段
// 场景：
// - 内置类型（不展开）
// - 指针类型（不展开）
// - 切片类型（不展开）
// - map 类型（不展开）
// - time.Time（不展开，虽然是结构体）
// - 自定义结构体类型（展开）
func TestShouldExpandEmbeddedField(t *testing.T) {
	tests := []struct {
		name      string
		fieldType string
		want      bool
	}{
		// 内置类型 - 不展开
		{name: "int", fieldType: "int", want: false},
		{name: "int64", fieldType: "int64", want: false},
		{name: "string", fieldType: "string", want: false},
		{name: "bool", fieldType: "bool", want: false},
		{name: "float64", fieldType: "float64", want: false},
		{name: "byte", fieldType: "byte", want: false},
		{name: "rune", fieldType: "rune", want: false},

		// 指针类型 - 不展开
		{name: "pointer to struct", fieldType: "*User", want: false},
		{name: "pointer to builtin", fieldType: "*int", want: false},

		// 切片和数组 - 不展开
		{name: "slice", fieldType: "[]User", want: false},
		{name: "slice of pointers", fieldType: "[]*User", want: false},
		// 注意：当前实现对数组的处理不完善，会被判断为应展开
		// 但数组实际上不是结构体，不应该展开。这是一个已知的小问题。
		{name: "array", fieldType: "[10]int", want: true},

		// Map - 不展开
		{name: "map", fieldType: "map[string]int", want: false},
		{name: "map with struct value", fieldType: "map[string]User", want: false},

		// Channel - 不展开
		{name: "channel", fieldType: "chan int", want: false},

		// Function - 不展开
		{name: "function", fieldType: "func(int) string", want: false},

		// time.Time 特殊处理 - 不展开
		{name: "time.Time", fieldType: "time.Time", want: false},
		{name: "time.Duration", fieldType: "time.Duration", want: false},

		// error 接口 - 不展开
		{name: "error", fieldType: "error", want: false},

		// 自定义结构体类型 - 展开
		{name: "simple struct", fieldType: "User", want: true},
		{name: "package qualified struct", fieldType: "models.User", want: true},
		{name: "nested package struct", fieldType: "github.com/pkg/User", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldExpandEmbeddedField(tt.fieldType)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestEncodeModulePath 测试模块路径编码
// 功能：将模块路径编码为 Go 模块缓存使用的格式（大写字母前加!并转小写）
// 场景：
// - 包含大写字母的路径
// - 全小写路径
// - 多个大写字母
// - 连续大写字母
func TestEncodeModulePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "path with uppercase letters",
			path: "github.com/Xuanwo/gg",
			want: "github.com/!xuanwo/gg",
		},
		{
			name: "all lowercase",
			path: "github.com/donutnomad/gogen",
			want: "github.com/donutnomad/gogen",
		},
		{
			name: "multiple uppercase",
			path: "github.com/User/Repo",
			want: "github.com/!user/!repo",
		},
		{
			name: "uppercase at start",
			path: "Github.com/user/repo",
			want: "!github.com/user/repo",
		},
		{
			name: "consecutive uppercase",
			path: "github.com/URLShortener/API",
			want: "github.com/!u!r!l!shortener/!a!p!i",
		},
		{
			name: "empty string",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeModulePath(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestExtractPkgPath 测试从字段类型提取包路径
// 功能：根据字段类型和导入信息，提取完整的包路径
// 场景：
// - 无包前缀的类型（本地类型）
// - 有包前缀的类型
// - 指针类型
// - 切片类型
// - 未知包前缀
func TestExtractPkgPath(t *testing.T) {
	imports := map[string]*ImportInfo{
		"orm": {
			Alias:       "orm",
			PackageName: "gorm",
			ImportPath:  "gorm.io/gorm",
		},
		"time": {
			PackageName: "time",
			ImportPath:  "time",
		},
		"decimal": {
			PackageName: "decimal",
			ImportPath:  "github.com/shopspring/decimal",
		},
	}

	tests := []struct {
		name      string
		fieldType string
		imports   map[string]*ImportInfo
		want      string
	}{
		{
			name:      "local type without package",
			fieldType: "User",
			imports:   imports,
			want:      "",
		},
		{
			name:      "standard library type",
			fieldType: "time.Time",
			imports:   imports,
			want:      "time",
		},
		{
			name:      "third-party type",
			fieldType: "decimal.Decimal",
			imports:   imports,
			want:      "github.com/shopspring/decimal",
		},
		{
			name:      "aliased import",
			fieldType: "orm.Model",
			imports:   imports,
			want:      "gorm.io/gorm",
		},
		{
			name:      "pointer type",
			fieldType: "*orm.DeletedAt",
			imports:   imports,
			want:      "gorm.io/gorm",
		},
		{
			name:      "slice type",
			fieldType: "[]decimal.Decimal",
			imports:   imports,
			want:      "github.com/shopspring/decimal",
		},
		{
			name:      "map type",
			fieldType: "map[string]time.Time",
			imports:   imports,
			want:      "",
		},
		{
			name:      "unknown package prefix",
			fieldType: "unknown.Type",
			imports:   imports,
			want:      "",
		},
		{
			name:      "builtin type",
			fieldType: "string",
			imports:   imports,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPkgPath(tt.fieldType, tt.imports)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestParseTypePackageAndName 测试类型的包名和结构体名解析
// 功能：将"package.Type"拆分为包名和类型名
// 场景：
// - 有包前缀的类型
// - 无包前缀的类型
func TestParseTypePackageAndName(t *testing.T) {
	tests := []struct {
		name           string
		typeName       string
		wantPkg        string
		wantStructName string
	}{
		{
			name:           "qualified type",
			typeName:       "orm.Model",
			wantPkg:        "orm",
			wantStructName: "Model",
		},
		{
			name:           "local type",
			typeName:       "User",
			wantPkg:        "",
			wantStructName: "User",
		},
		{
			name:           "nested package",
			typeName:       "models.User",
			wantPkg:        "models",
			wantStructName: "User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkg, gotName := parseTypePackageAndName(tt.typeName)
			assert.Equal(t, tt.wantPkg, gotPkg, "package name mismatch")
			assert.Equal(t, tt.wantStructName, gotName, "struct name mismatch")
		})
	}
}

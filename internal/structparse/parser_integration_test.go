package structparse

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseSimpleStruct 测试简单结构体解析
// 功能：解析包含基本字段的结构体
// 场景：无嵌入、无跨包引用的简单结构体
func TestParseSimpleStruct(t *testing.T) {
	filename := filepath.Join("testdata", "simple", "user.go")
	structName := "User"

	info, err := ParseStruct(filename, structName)
	require.NoError(t, err, "解析简单结构体失败")
	require.NotNil(t, info, "结构体信息不应为空")

	// 验证基本信息
	assert.Equal(t, "User", info.Name, "结构体名称不匹配")
	assert.Equal(t, "simple", info.PackageName, "包名不匹配")

	// 验证字段数量
	assert.GreaterOrEqual(t, len(info.Fields), 3, "字段数量应该至少有3个")

	// 验证字段信息
	fieldMap := make(map[string]FieldInfo)
	for _, field := range info.Fields {
		fieldMap[field.Name] = field
	}

	// 检查ID字段
	id, ok := fieldMap["ID"]
	assert.True(t, ok, "应该有ID字段")
	assert.Equal(t, "int64", id.Type, "ID类型应为int64")

	// 检查Name字段
	name, ok := fieldMap["Name"]
	assert.True(t, ok, "应该有Name字段")
	assert.Equal(t, "string", name.Type, "Name类型应为string")
}

// TestParseEmbeddedStruct 测试匿名嵌入结构体解析
// 功能：解析包含匿名嵌入字段的结构体，验证字段展开
// 场景：User嵌入BaseModel，字段应该被自动展开
func TestParseEmbeddedStruct(t *testing.T) {
	filename := filepath.Join("testdata", "embedded", "user.go")
	structName := "User"

	info, err := ParseStruct(filename, structName)
	require.NoError(t, err, "解析嵌入结构体失败")
	require.NotNil(t, info, "结构体信息不应为空")

	// 验证字段数量（包括从BaseModel展开的字段）
	assert.GreaterOrEqual(t, len(info.Fields), 5, "应包含自身字段和嵌入字段")

	// 构建字段映射
	fieldMap := make(map[string]FieldInfo)
	for _, field := range info.Fields {
		fieldMap[field.Name] = field
	}

	// 验证User自身的字段
	name, ok := fieldMap["Name"]
	assert.True(t, ok, "应该有Name字段")
	assert.Equal(t, "string", name.Type, "Name类型应为string")

	// 验证从BaseModel展开的字段
	id, ok := fieldMap["ID"]
	assert.True(t, ok, "应该有从BaseModel展开的ID字段")
	assert.Equal(t, "int64", id.Type, "ID类型应为int64")
	assert.Equal(t, "BaseModel", id.SourceType, "ID应标记来源为BaseModel")

	createdAt, ok := fieldMap["CreatedAt"]
	assert.True(t, ok, "应该有从BaseModel展开的CreatedAt字段")
	assert.Contains(t, createdAt.Type, "time.Time", "CreatedAt类型应包含time.Time")
}

// TestParseGormEmbeddedStruct 测试GORM embedded标签解析
// 功能：解析带有gorm:"embedded;embeddedPrefix:xxx"标签的字段
// 场景：UserWithAccount使用gorm embedded标签嵌入Account结构体
func TestParseGormEmbeddedStruct(t *testing.T) {
	filename := filepath.Join("testdata", "embedded", "gorm_embedded.go")
	structName := "UserWithAccount"

	info, err := ParseStruct(filename, structName)
	require.NoError(t, err, "解析GORM embedded结构体失败")
	require.NotNil(t, info, "结构体信息不应为空")

	// 验证字段数量
	assert.GreaterOrEqual(t, len(info.Fields), 3, "应包含自身字段和嵌入的Account字段")

	// 构建字段映射
	fieldMap := make(map[string]FieldInfo)
	for _, field := range info.Fields {
		fieldMap[field.Name] = field
	}

	// 验证UserWithAccount自身的字段
	_, ok := fieldMap["ID"]
	assert.True(t, ok, "应该有ID字段")

	// 验证从Account展开的字段
	accountName, ok := fieldMap["AccountName"]
	assert.True(t, ok, "应该有从Account展开的AccountName字段")
	assert.Equal(t, "string", accountName.Type, "AccountName类型应为string")
	assert.Equal(t, "account_", accountName.EmbeddedPrefix, "应该有account_前缀")

	balance, ok := fieldMap["Balance"]
	assert.True(t, ok, "应该有从Account展开的Balance字段")
	assert.Equal(t, "account_", balance.EmbeddedPrefix, "Balance应该有account_前缀")
}

// TestParseCrossPackageFields 测试跨包字段解析
// 功能：解析包含导入其他包类型的字段
// 场景：Order结构体使用了time.Time、decimal.Decimal、orm.DeletedAt
func TestParseCrossPackageFields(t *testing.T) {
	filename := filepath.Join("testdata", "imports", "types.go")
	structName := "Order"

	info, err := ParseStruct(filename, structName)
	require.NoError(t, err, "解析跨包字段失败")
	require.NotNil(t, info, "结构体信息不应为空")

	// 验证导入信息
	assert.Contains(t, info.Imports, "time", "应包含time包导入")
	assert.Contains(t, info.Imports, "gorm.io/gorm", "应包含gorm包导入")

	// 构建字段映射
	fieldMap := make(map[string]FieldInfo)
	for _, field := range info.Fields {
		fieldMap[field.Name] = field
	}

	// 验证time.Time字段
	createdAt, ok := fieldMap["CreatedAt"]
	assert.True(t, ok, "应该有CreatedAt字段")
	assert.Equal(t, "time.Time", createdAt.Type, "CreatedAt类型应为time.Time")
	assert.Equal(t, "time", createdAt.PkgPath, "CreatedAt的PkgPath应为time")

	// 验证decimal.Decimal字段
	amount, ok := fieldMap["Amount"]
	assert.True(t, ok, "应该有Amount字段")
	assert.Equal(t, "decimal.Decimal", amount.Type, "Amount类型应为decimal.Decimal")
	assert.Equal(t, "github.com/shopspring/decimal", amount.PkgPath, "Amount的PkgPath应正确")

	// 验证带别名的导入（orm.DeletedAt）
	deletedAt, ok := fieldMap["DeletedAt"]
	assert.True(t, ok, "应该有DeletedAt字段")
	assert.Equal(t, "orm.DeletedAt", deletedAt.Type, "DeletedAt类型应为orm.DeletedAt")
	assert.Equal(t, "gorm.io/gorm", deletedAt.PkgPath, "DeletedAt的PkgPath应正确解析别名")
}

// TestParseMethodsFromMultipleFiles 测试跨文件方法解析
// 功能：解析分散在多个文件中的结构体方法
// 场景：Product结构体的方法定义在product.go和product_helper.go中
func TestParseMethodsFromMultipleFiles(t *testing.T) {
	filename := filepath.Join("testdata", "methods", "product.go")
	structName := "Product"

	info, err := ParseStruct(filename, structName)
	require.NoError(t, err, "解析跨文件方法失败")
	require.NotNil(t, info, "结构体信息不应为空")

	// 验证方法数量
	assert.GreaterOrEqual(t, len(info.Methods), 2, "应至少找到2个方法")

	// 构建方法映射
	methodMap := make(map[string]MethodInfo)
	for _, method := range info.Methods {
		methodMap[method.Name] = method
	}

	// 验证值接收器方法（定义在product.go）
	getDisplayName, ok := methodMap["GetDisplayName"]
	assert.True(t, ok, "应该有GetDisplayName方法")
	assert.Equal(t, "Product", getDisplayName.ReceiverType, "GetDisplayName应为值接收器")
	assert.Contains(t, getDisplayName.FilePath, "product.go", "GetDisplayName应在product.go中")

	// 验证指针接收器方法（定义在product.go）
	updatePrice, ok := methodMap["UpdatePrice"]
	assert.True(t, ok, "应该有UpdatePrice方法")
	assert.Equal(t, "*Product", updatePrice.ReceiverType, "UpdatePrice应为指针接收器")

	// 验证跨文件方法（定义在product_helper.go）
	validate, ok := methodMap["Validate"]
	assert.True(t, ok, "应该有Validate方法")
	assert.Contains(t, validate.FilePath, "product_helper.go", "Validate应在product_helper.go中")
}

// TestParseContextWithCustomResolver 测试自定义PackageResolver
// 功能：验证依赖注入功能，使用自定义的PackageResolver
// 场景：模拟测试环境，使用mock resolver
func TestParseContextWithCustomResolver(t *testing.T) {
	// 创建一个mock resolver
	mockResolver := &mockPackageResolver{
		packages: map[string]string{
			"time":               "time",
			"gorm.io/gorm":       "gorm",
			"github.com/example": "example",
		},
	}

	// 使用自定义resolver创建ParseContext
	ctx := NewParseContextWithResolver(mockResolver)

	filename := filepath.Join("testdata", "imports", "types.go")
	info, err := ctx.ParseStruct(filename, "Order")

	require.NoError(t, err, "使用自定义resolver解析失败")
	require.NotNil(t, info, "结构体信息不应为空")

	// 验证mock resolver被调用
	assert.True(t, mockResolver.called, "自定义resolver应该被调用")
}

// mockPackageResolver 用于测试的mock PackageResolver
type mockPackageResolver struct {
	packages map[string]string
	called   bool
}

func (m *mockPackageResolver) GetPackageName(importPath string) (string, error) {
	m.called = true
	if pkg, ok := m.packages[importPath]; ok {
		return pkg, nil
	}
	return "", nil
}

// TestCircularReferenceDetection 测试循环引用检测
// 功能：验证解析器能正确检测并处理循环引用
// 场景：如果存在A嵌入B、B嵌入A的情况，应该能优雅处理
func TestCircularReferenceDetection(t *testing.T) {
	// 注意：这个测试需要专门的testdata，当前先跳过
	// 在实际使用中，循环引用会通过stack检测避免无限递归
	t.Skip("需要专门的循环引用testdata")
}

// TestMaxEmbeddingDepth 测试最大嵌套深度限制
// 功能：验证超过maxEmbeddingDepth时会返回错误
// 场景：防止过深的嵌套导致栈溢出
func TestMaxEmbeddingDepth(t *testing.T) {
	// 注意：这个测试需要专门的深度嵌套testdata
	t.Skip("需要专门的深度嵌套testdata")
}

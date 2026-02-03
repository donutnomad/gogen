package embedded

// InnerStruct 内层嵌入结构体
type InnerStruct struct {
	InnerField1 string `gorm:"column:inner_field1"`
	InnerField2 int    `gorm:"column:inner_field2"`
}

// OuterStruct 外层嵌入结构体，包含内层嵌入
type OuterStruct struct {
	OuterField string      `gorm:"column:outer_field"`
	Inner      InnerStruct `gorm:"embedded"`
}

// NestedEmbeddedPO 测试多层嵌套的结构体
type NestedEmbeddedPO struct {
	ID    int64       `gorm:"primaryKey"`
	Name  string      `gorm:"column:name"`
	Outer OuterStruct `gorm:"embedded;embeddedPrefix:outer_"`
}

// MixedEmbeddedPO 测试混合场景：匿名嵌入 + gorm embedded
type MixedEmbeddedPO struct {
	BaseModel             // 匿名嵌入，直接访问
	Name      string      `gorm:"column:name"`
	Account   InnerStruct `gorm:"embedded;embeddedPrefix:acc_"` // gorm embedded，需要通过 Account 访问
}

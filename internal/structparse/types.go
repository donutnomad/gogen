package structparse

// ImportInfo 导入信息
type ImportInfo struct {
	Alias       string // 显式别名（如果有）
	PackageName string // 真实包名（从 package 声明读取）
	ImportPath  string // 完整导入路径
}

// MethodInfo 表示方法信息
type MethodInfo struct {
	Name         string // 方法名
	ReceiverName string // 接收器名称
	ReceiverType string // 接收器类型
	ReturnType   string // 返回类型
	FilePath     string // 方法所在文件的绝对路径
}

// FieldInfo 表示结构体字段信息
type FieldInfo struct {
	Name           string // 字段名
	Type           string // 字段类型
	PkgPath        string // 类型所在包路径
	PkgAlias       string // 包在源文件中的别名（如果有）
	Tag            string // 字段标签
	SourceType     string // 字段来源类型，为空表示来自结构体本身，否则表示来自嵌入的结构体
	EmbeddedPrefix string // gorm embedded 字段的 prefix，用于列名生成
}

// StructInfo 表示结构体信息
type StructInfo struct {
	Name        string       // 结构体名称
	PackageName string       // 包名
	FilePath    string       // 结构体所在文件路径
	Fields      []FieldInfo  // 字段列表
	Methods     []MethodInfo // 方法列表
	Imports     []string     // 导入的包
}

// maxEmbeddingDepth 最大嵌套深度限制
const maxEmbeddingDepth = 10

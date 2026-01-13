package templategen

// TemplateData 提供给模板的数据
type TemplateData struct {
	// 文件信息
	File FileInfo

	// 带 @Define 注解的结构体（只有带注解的才会出现）
	Structs []StructData

	// 带 @Define 注解的接口
	Interfaces []InterfaceData

	// 带 @Define 注解的函数
	Functions []FunctionData

	// 文件级 @Import 定义
	ImportAliases map[string]string // alias -> full path

	// 导入管理器（供模板动态添加 import）
	Imports *ImportManager
}

// FileInfo 文件信息
type FileInfo struct {
	Path        string // 源文件完整路径
	Dir         string // 所在目录
	Name        string // 文件名（不含扩展名）
	PackageName string // 包名
}

// StructData 结构体数据（带注解的）
type StructData struct {
	Name    string      // 结构体名
	Fields  []FieldData // 字段列表
	Defines DefineGroup // @Define 定义的元数据

	// 带 @Define 注解的方法（按 receiver 分组）
	Methods []MethodData
}

// InterfaceData 接口数据
type InterfaceData struct {
	Name    string      // 接口名
	Methods []MethodSig // 方法签名
	Defines DefineGroup // @Define 定义的元数据
}

// FunctionData 包级函数数据
type FunctionData struct {
	Name    string       // 函数名
	Params  []ParamData  // 参数列表
	Returns []ReturnData // 返回值列表
	Defines DefineGroup  // @Define 定义的元数据
}

// MethodData 方法数据（带注解的）
type MethodData struct {
	Name         string       // 方法名
	ReceiverName string       // receiver 变量名
	ReceiverType string       // receiver 类型
	IsPointer    bool         // receiver 是否为指针
	Params       []ParamData  // 参数列表
	Returns      []ReturnData // 返回值列表
	Defines      DefineGroup  // 该方法的 @Define 元数据
}

// MethodSig 接口方法签名
type MethodSig struct {
	Name    string       // 方法名
	Params  []ParamData  // 参数列表
	Returns []ReturnData // 返回值列表
}

// DefineGroup 按 name 分组的定义
// 模板访问: .Defines.Config.reader
type DefineGroup map[string]map[string]TypeRef

// TypeRef 类型引用
type TypeRef struct {
	Raw       string // 原始值
	IsString  bool   // 是否为字符串值
	StringVal string // 字符串值（当 IsString=true）
	TypeName  string // 类型名 (如 Reader, Helper)
	PkgPath   string // 完整包路径
	PkgAlias  string // 使用的别名
	FullType  string // 完整类型表达式 (如 io.Reader)
}

// FieldData 字段信息
type FieldData struct {
	Name    string
	Type    string
	Tag     string
	Comment string
}

// ParamData 参数信息
type ParamData struct {
	Name string
	Type string
}

// ReturnData 返回值信息
type ReturnData struct {
	Name string // 可能为空
	Type string
}

// ImportManager 管理模板生成过程中的 import
type ImportManager struct {
	imports map[string]string // path -> alias (empty string means no alias)
}

// NewImportManager 创建新的 ImportManager
func NewImportManager() *ImportManager {
	return &ImportManager{
		imports: make(map[string]string),
	}
}

// Add 添加 import，返回包名（用于模板中引用）
func (m *ImportManager) Add(path string) string {
	if _, exists := m.imports[path]; !exists {
		m.imports[path] = ""
	}
	// 返回包名（路径最后一部分）
	return getPackageName(path)
}

// AddAlias 添加带别名的 import，返回别名
func (m *ImportManager) AddAlias(path, alias string) string {
	m.imports[path] = alias
	return alias
}

// All 返回所有 import
func (m *ImportManager) All() map[string]string {
	return m.imports
}

// getPackageName 从路径获取包名
func getPackageName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

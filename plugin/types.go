package plugin

import (
	"go/ast"
	"go/token"
	"path/filepath"

	"github.com/donutnomad/gg"
)

// TargetKind 表示注解目标的类型
type TargetKind int

const (
	TargetStruct    TargetKind = iota + 1 // 结构体
	TargetInterface                       // 接口
	TargetFunc                            // 包级函数
	TargetMethod                          // 结构体方法
	TargetVar                             // 变量声明
	TargetConst                           // 常量声明
	TargetComment                         // 独立注释 (//go:gen:)
)

func (k TargetKind) String() string {
	switch k {
	case TargetStruct:
		return "struct"
	case TargetInterface:
		return "interface"
	case TargetFunc:
		return "func"
	case TargetMethod:
		return "method"
	case TargetVar:
		return "var"
	case TargetConst:
		return "const"
	case TargetComment:
		return "comment"
	default:
		return "unknown"
	}
}

// ParamDef 定义注解参数的元信息
type ParamDef struct {
	Name        string // 参数名称
	Required    bool   // 是否必填
	Default     string // 默认值（如果不是必填）
	Description string // 参数描述
}

// Annotation 表示解析后的注解
type Annotation struct {
	Name   string            // 注解名称，如 "Gsql", "Mapper"
	Params map[string]string // 注解参数，如 prefix=`xxx`
	Raw    string            // 原始注解文本
}

// Target 表示注解的目标
type Target struct {
	Kind        TargetKind // 目标类型
	Name        string     // 名称（结构体名、接口名、函数名、方法名）
	PackageName string     // 包名
	FilePath    string     // 文件路径
	Position    token.Pos  // 位置信息

	// 方法特有字段
	ReceiverName string // 接收者名称（仅方法）
	ReceiverType string // 接收者类型（仅方法）

	// AST 节点（可选，用于深度解析）
	Node ast.Node
}

// AnnotatedTarget 表示带注解的目标
type AnnotatedTarget struct {
	Target       *Target       // 目标信息
	Annotations  []*Annotation // 注解列表
	ParsedParams any           // 解析后的参数结构体
}

// ScanResult 表示扫描结果
type ScanResult struct {
	Structs    []*AnnotatedTarget // 带注解的结构体
	Interfaces []*AnnotatedTarget // 带注解的接口
	Funcs      []*AnnotatedTarget // 带注解的包级函数
	Methods    []*AnnotatedTarget // 带注解的方法
	Vars       []*AnnotatedTarget // 带注解的变量
	Consts     []*AnnotatedTarget // 带注解的常量
	Comments   []*AnnotatedTarget // 独立注释 (//go:gen:)

	// PackageConfigs 包级配置
	// key: 包目录路径（绝对路径）
	PackageConfigs map[string]*PackageConfig
}

// All 返回所有带注解的目标
func (r *ScanResult) All() []*AnnotatedTarget {
	result := make([]*AnnotatedTarget, 0, len(r.Structs)+len(r.Interfaces)+len(r.Funcs)+len(r.Methods)+len(r.Vars)+len(r.Consts)+len(r.Comments))
	result = append(result, r.Structs...)
	result = append(result, r.Interfaces...)
	result = append(result, r.Funcs...)
	result = append(result, r.Methods...)
	result = append(result, r.Vars...)
	result = append(result, r.Consts...)
	result = append(result, r.Comments...)
	return result
}

// ByAnnotation 按注解名称过滤
func (r *ScanResult) ByAnnotation(name string) []*AnnotatedTarget {
	var result []*AnnotatedTarget
	for _, t := range r.All() {
		for _, a := range t.Annotations {
			if a.Name == name {
				result = append(result, t)
				break
			}
		}
	}
	return result
}

// GenerateContext 生成上下文，传递给 Generator
type GenerateContext struct {
	Targets        []*AnnotatedTarget        // 该 Generator 需要处理的目标
	PackageConfigs map[string]*PackageConfig // 包级配置，key: 包目录路径
	DefaultOutput  string                    // 命令行指定的默认输出路径（最低优先级）
	Verbose        bool                      // 详细输出
}

// GetPackageConfig 获取指定文件所在包的配置
func (c *GenerateContext) GetPackageConfig(filePath string) *PackageConfig {
	if c.PackageConfigs == nil {
		return nil
	}
	pkgDir := filepath.Dir(filePath)
	return c.PackageConfigs[pkgDir]
}

// GetFileConfig 获取指定文件的配置（兼容旧 API，实际返回包级配置）
// Deprecated: 请使用 GetPackageConfig
func (c *GenerateContext) GetFileConfig(filePath string) *PackageConfig {
	return c.GetPackageConfig(filePath)
}

// GenerateResult 生成结果
// Generator 返回 gg 定义，由聚合器统一处理
type GenerateResult struct {
	// Definitions 是生成的 gg 定义
	// key: 输出文件路径（相对路径或绝对路径）
	// value: gg.Generator 定义
	Definitions map[string]*gg.Generator

	// RawOutputs 是原始字节输出，用于不使用 gg 库的生成器
	// key: 输出文件路径（相对路径或绝对路径）
	// value: 原始字节内容
	// 注意: RawOutputs 中的文件不会与其他生成器的输出合并
	RawOutputs map[string][]byte

	// Errors 错误列表
	Errors []error

	// Skipped 跳过的数量
	Skipped int
}

// PackageConfig 包级生成配置
// 通过 //go:gogen: 或 // go:gogen: 注释定义，作用于整个包
// 示例:
//
//	//go:gogen: -output `$FILE_query`
//	// go:gogen: plugin:gsql -output `$FILE_query` plugin:setter -output `0api_generated`
type PackageConfig struct {
	PackageDir string // 包目录路径

	// DefaultOutput 默认输出路径（对所有插件生效）
	// 来自: //go:gogen: -output `xxx`
	DefaultOutput string

	// PluginOutputs 插件特定的输出路径
	// key: 插件名（小写）, value: 输出路径
	// 来自: //go:gogen: plugin:gsql -output `xxx`
	PluginOutputs map[string]string
}

// GetPluginOutput 获取指定插件的输出路径
// 优先返回插件特定配置，其次返回默认配置，最后返回空字符串
func (c *PackageConfig) GetPluginOutput(pluginName string) string {
	if c == nil {
		return ""
	}
	if output, ok := c.PluginOutputs[pluginName]; ok {
		return output
	}
	return c.DefaultOutput
}

// FileConfig 文件级生成配置（已废弃，使用 PackageConfig）
// Deprecated: 请使用 PackageConfig
type FileConfig = PackageConfig

// NewGenerateResult 创建新的生成结果
func NewGenerateResult() *GenerateResult {
	return &GenerateResult{
		Definitions: make(map[string]*gg.Generator),
		RawOutputs:  make(map[string][]byte),
	}
}

// ImportWithAlias 带别名的 import 信息
// 用于代码生成时处理需要别名的包导入
type ImportWithAlias struct {
	Path  string // 导入路径，如 "github.com/xxx/domain"
	Alias string // 别名，如 "domain"（当与路径最后一部分不同时需要，空表示无别名）
}

// AddDefinition 添加 gg 定义
func (r *GenerateResult) AddDefinition(path string, gen *gg.Generator) {
	if r.Definitions == nil {
		r.Definitions = make(map[string]*gg.Generator)
	}
	r.Definitions[path] = gen
}

// AddRawOutput 添加原始字节输出
// 用于不使用 gg 库的生成器
func (r *GenerateResult) AddRawOutput(path string, data []byte) {
	r.RawOutputs[path] = data
}

// AddError 添加错误
func (r *GenerateResult) AddError(err error) {
	r.Errors = append(r.Errors, err)
}

// HasErrors 检查是否有错误
func (r *GenerateResult) HasErrors() bool {
	return len(r.Errors) > 0
}

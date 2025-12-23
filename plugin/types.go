package plugin

import (
	"go/ast"
	"go/token"

	"github.com/donutnomad/gg"
)

// TargetKind 表示注解目标的类型
type TargetKind int

const (
	TargetStruct    TargetKind = iota + 1 // 结构体
	TargetInterface                       // 接口
	TargetFunc                            // 包级函数
	TargetMethod                          // 结构体方法
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

	// FileConfigs 文件级配置
	// key: 文件路径
	FileConfigs map[string]*FileConfig
}

// All 返回所有带注解的目标
func (r *ScanResult) All() []*AnnotatedTarget {
	result := make([]*AnnotatedTarget, 0, len(r.Structs)+len(r.Interfaces)+len(r.Funcs)+len(r.Methods))
	result = append(result, r.Structs...)
	result = append(result, r.Interfaces...)
	result = append(result, r.Funcs...)
	result = append(result, r.Methods...)
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
	Targets       []*AnnotatedTarget     // 该 Generator 需要处理的目标
	FileConfigs   map[string]*FileConfig // 文件级配置，key: 文件路径
	DefaultOutput string                 // 命令行指定的默认输出路径（最低优先级）
	Verbose       bool                   // 详细输出
}

// GetFileConfig 获取指定文件的配置
func (c *GenerateContext) GetFileConfig(filePath string) *FileConfig {
	if c.FileConfigs == nil {
		return nil
	}
	return c.FileConfigs[filePath]
}

// GenerateResult 生成结果
// Generator 返回 gg 定义，由聚合器统一处理
type GenerateResult struct {
	// Definitions 是生成的 gg 定义
	// key: 输出文件路径（相对路径或绝对路径）
	// value: gg.Generator 定义
	Definitions map[string]*gg.Generator

	// Errors 错误列表
	Errors []error

	// Skipped 跳过的数量
	Skipped int
}

// FileConfig 文件级生成配置
// 通过 // go:gogen: 注释定义
// 示例:
//
//	// go:gogen: -output `{{FileName}}_query`
//	// go:gogen: plugin:gsql -output `{{FileName}}_query` plugin:setter -output `0api_generated`
type FileConfig struct {
	FilePath string // 文件路径

	// DefaultOutput 默认输出路径（对所有插件生效）
	// 来自: // go:gogen: -output `xxx`
	DefaultOutput string

	// PluginOutputs 插件特定的输出路径
	// key: 插件名（小写）, value: 输出路径
	// 来自: // go:gogen: plugin:gsql -output `xxx`
	PluginOutputs map[string]string
}

// GetPluginOutput 获取指定插件的输出路径
// 优先返回插件特定配置，其次返回默认配置，最后返回空字符串
func (c *FileConfig) GetPluginOutput(pluginName string) string {
	if c == nil {
		return ""
	}
	if output, ok := c.PluginOutputs[pluginName]; ok {
		return output
	}
	return c.DefaultOutput
}

// NewGenerateResult 创建新的生成结果
func NewGenerateResult() *GenerateResult {
	return &GenerateResult{
		Definitions: make(map[string]*gg.Generator),
	}
}

// AddDefinition 添加 gg 定义
func (r *GenerateResult) AddDefinition(path string, gen *gg.Generator) {
	if r.Definitions == nil {
		r.Definitions = make(map[string]*gg.Generator)
	}
	r.Definitions[path] = gen
}

// AddError 添加错误
func (r *GenerateResult) AddError(err error) {
	r.Errors = append(r.Errors, err)
}

// HasErrors 检查是否有错误
func (r *GenerateResult) HasErrors() bool {
	return len(r.Errors) > 0
}

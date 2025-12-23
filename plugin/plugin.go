package plugin

import "reflect"

// Generator 是代码生成器接口
// 每个 gen（如 gormgen、mapper）需要实现此接口
type Generator interface {
	// Name 返回生成器名称
	Name() string

	// Annotations 返回该生成器支持的注解列表
	// 一个注解只能绑定一个生成器
	Annotations() []string

	// SupportedTargets 返回支持的目标类型
	// 例如：gormgen 只支持 TargetStruct
	SupportedTargets() []TargetKind

	// ParamDefs 返回注解支持的参数定义
	// 包括参数名、是否必填、默认值等
	ParamDefs() []ParamDef

	// NewParams 创建并返回该生成器的参数结构体实例
	// 返回 nil 表示该生成器不需要参数
	NewParams() any

	// Priority 返回生成器优先级
	// 数字越小优先级越高，输出合并时优先级高的在前面
	// 默认值为 100
	Priority() int

	// Generate 执行代码生成
	// 返回的 GenerateResult 包含 gg 定义，由聚合器统一处理
	Generate(ctx *GenerateContext) (*GenerateResult, error)
}

// GeneratorOption 生成器选项
type GeneratorOption func(*generatorConfig)

type generatorConfig struct {
	verbose bool
}

// WithVerbose 设置详细输出
func WithVerbose(v bool) GeneratorOption {
	return func(c *generatorConfig) {
		c.verbose = v
	}
}

// BaseGenerator 提供基础实现，可嵌入
type BaseGenerator struct {
	name        string
	annotations []string
	targets     []TargetKind
	paramDefs   []ParamDef
	paramsProto any // 参数结构体原型，用于创建新实例
	priority    int // 优先级，数字越小优先级越高
}

func NewBaseGenerator(name string, annotations []string, targets []TargetKind) *BaseGenerator {
	return &BaseGenerator{
		name:        name,
		annotations: annotations,
		targets:     targets,
		priority:    100,
	}
}

// NewBaseGeneratorWithParams 创建带参数定义的基础生成器
func NewBaseGeneratorWithParams(name string, annotations []string, targets []TargetKind, params []ParamDef) *BaseGenerator {
	return &BaseGenerator{
		name:        name,
		annotations: annotations,
		targets:     targets,
		paramDefs:   params,
		priority:    100,
	}
}

// NewBaseGeneratorWithParamsStruct 创建带参数结构体的基础生成器
// paramsProto: 参数结构体的零值实例，例如 GsqlParams{}
func NewBaseGeneratorWithParamsStruct(name string, annotations []string, targets []TargetKind, paramsProto any) *BaseGenerator {
	return &BaseGenerator{
		name:        name,
		annotations: annotations,
		targets:     targets,
		paramDefs:   ParseParamsFromStruct(paramsProto),
		paramsProto: paramsProto,
		priority:    100,
	}
}

func (g *BaseGenerator) Name() string {
	return g.name
}

func (g *BaseGenerator) Annotations() []string {
	return g.annotations
}

func (g *BaseGenerator) SupportedTargets() []TargetKind {
	return g.targets
}

func (g *BaseGenerator) ParamDefs() []ParamDef {
	return g.paramDefs
}

// NewParams 创建参数结构体的新实例
// 使用反射创建新实例，返回指针类型以便设置字段值
func (g *BaseGenerator) NewParams() any {
	if g.paramsProto == nil {
		return nil
	}
	// 使用反射创建新的结构体实例
	typ := reflect.TypeOf(g.paramsProto)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	// 返回指向新实例的指针
	return reflect.New(typ).Interface()
}

// SetParamDefs 设置参数定义
func (g *BaseGenerator) SetParamDefs(params []ParamDef) *BaseGenerator {
	g.paramDefs = params
	return g
}

// Priority 返回生成器优先级
func (g *BaseGenerator) Priority() int {
	return g.priority
}

// SetPriority 设置生成器优先级，数字越小优先级越高
func (g *BaseGenerator) SetPriority(priority int) *BaseGenerator {
	g.priority = priority
	return g
}

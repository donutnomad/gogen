package plugin

import (
	"fmt"
	"sync"
)

// Registry 注解注册表
// 管理注解到生成器的映射，确保一个注解只绑定一个生成器
type Registry struct {
	mu sync.RWMutex

	// annotations 注解名 -> 生成器
	annotations map[string]Generator

	// generators 生成器名 -> 生成器
	generators map[string]Generator
}

// NewRegistry 创建新的注册表
func NewRegistry() *Registry {
	return &Registry{
		annotations: make(map[string]Generator),
		generators:  make(map[string]Generator),
	}
}

// Register 注册生成器
// 如果注解已被其他生成器注册，返回错误
func (r *Registry) Register(gen Generator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := gen.Name()

	// 检查生成器是否已注册
	if existing, ok := r.generators[name]; ok {
		return fmt.Errorf("生成器 %q 已注册", existing.Name())
	}

	// 检查注解是否已被其他生成器绑定
	for _, ann := range gen.Annotations() {
		if existing, ok := r.annotations[ann]; ok {
			return fmt.Errorf("注解 @%s 已被生成器 %q 绑定，无法被 %q 再次绑定",
				ann, existing.Name(), name)
		}
	}

	// 注册生成器
	r.generators[name] = gen

	// 绑定注解
	for _, ann := range gen.Annotations() {
		r.annotations[ann] = gen
	}

	return nil
}

// MustRegister 注册生成器，失败时 panic
func (r *Registry) MustRegister(gen Generator) {
	if err := r.Register(gen); err != nil {
		panic(err)
	}
}

// Unregister 取消注册生成器
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	gen, ok := r.generators[name]
	if !ok {
		return fmt.Errorf("生成器 %q 未注册", name)
	}

	// 移除注解绑定
	for _, ann := range gen.Annotations() {
		delete(r.annotations, ann)
	}

	// 移除生成器
	delete(r.generators, name)

	return nil
}

// GetByAnnotation 根据注解名获取生成器
func (r *Registry) GetByAnnotation(annotation string) (Generator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gen, ok := r.annotations[annotation]
	return gen, ok
}

// GetByName 根据生成器名获取生成器
func (r *Registry) GetByName(name string) (Generator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gen, ok := r.generators[name]
	return gen, ok
}

// Generators 返回所有已注册的生成器
func (r *Registry) Generators() []Generator {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Generator, 0, len(r.generators))
	for _, gen := range r.generators {
		result = append(result, gen)
	}
	return result
}

// Annotations 返回所有已注册的注解
func (r *Registry) Annotations() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0, len(r.annotations))
	for ann := range r.annotations {
		result = append(result, ann)
	}
	return result
}

// IsRegistered 检查注解是否已注册
func (r *Registry) IsRegistered(annotation string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.annotations[annotation]
	return ok
}

// DispatchTargets 将扫描结果分发给对应的生成器
// 返回 map[生成器名] -> 该生成器需要处理的目标
func (r *Registry) DispatchTargets(result *ScanResult) map[string][]*AnnotatedTarget {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dispatch := make(map[string][]*AnnotatedTarget)

	for _, target := range result.All() {
		for _, ann := range target.Annotations {
			if gen, ok := r.annotations[ann.Name]; ok {
				// 检查目标类型是否被支持
				if r.isTargetSupported(gen, target.Target.Kind) {
					dispatch[gen.Name()] = append(dispatch[gen.Name()], target)
				}
			}
		}
	}

	return dispatch
}

// isTargetSupported 检查生成器是否支持该目标类型
func (r *Registry) isTargetSupported(gen Generator, kind TargetKind) bool {
	for _, k := range gen.SupportedTargets() {
		if k == kind {
			return true
		}
	}
	return false
}

// 全局注册表
var globalRegistry = NewRegistry()

// Global 返回全局注册表
func Global() *Registry {
	return globalRegistry
}

// Register 向全局注册表注册生成器
func Register(gen Generator) error {
	return globalRegistry.Register(gen)
}

// MustRegister 向全局注册表注册生成器，失败时 panic
func MustRegister(gen Generator) {
	globalRegistry.MustRegister(gen)
}

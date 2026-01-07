package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/internal/utils"
)

// Run 运行代码生成
// 1. 扫描指定路径的注解
// 2. 将目标分发给对应的生成器
// 3. 执行生成器
// 4. 合并同一文件的 gg 定义并写入文件
func Run(ctx context.Context, registry *Registry, patterns ...string) error {
	opts := &RunOptions{
		Registry: registry,
		Patterns: patterns,
	}
	return RunWithOptions(ctx, opts)
}

// RunGlobal 使用全局注册表运行
func RunGlobal(ctx context.Context, patterns ...string) error {
	return Run(ctx, globalRegistry, patterns...)
}

// RunOptions 运行选项
type RunOptions struct {
	Registry *Registry
	Patterns []string
	Verbose  bool
	Output   string // 命令行指定的默认输出路径（最低优先级）
	Async    bool   // 是否异步执行生成器，默认 true
}

// RunStats 运行统计信息
type RunStats struct {
	ScanDuration     time.Duration // 扫描耗时
	GenerateDuration time.Duration // 生成耗时
	TotalDuration    time.Duration // 总耗时
	TargetCount      int           // 目标数量
	FileCount        int           // 生成文件数量
}

// RunWithOptions 带选项运行
func RunWithOptions(ctx context.Context, opts *RunOptions) error {
	_, err := RunWithOptionsAndStats(ctx, opts)
	return err
}

// RunWithOptionsAndStats 带选项运行并返回统计信息
func RunWithOptionsAndStats(ctx context.Context, opts *RunOptions) (*RunStats, error) {
	totalStart := time.Now()
	stats := &RunStats{}

	registry := opts.Registry
	if registry == nil {
		registry = globalRegistry
	}

	// 获取所有已注册的注解
	annotations := registry.Annotations()
	if len(annotations) == 0 {
		return nil, fmt.Errorf("没有已注册的生成器")
	}

	// 扫描
	scanStart := time.Now()
	scanner := NewScanner(
		WithAnnotationFilter(annotations...),
		WithScannerVerbose(opts.Verbose),
	)
	result, err := scanner.Scan(ctx, opts.Patterns...)
	if err != nil {
		return nil, fmt.Errorf("扫描失败: %w", err)
	}
	stats.ScanDuration = time.Since(scanStart)

	if len(result.All()) == 0 {
		if opts.Verbose {
			fmt.Println("没有找到任何带注解的目标")
		}
		stats.TotalDuration = time.Since(totalStart)
		return stats, nil
	}

	stats.TargetCount = len(result.All())
	if opts.Verbose {
		fmt.Printf("找到 %d 个带注解的目标 (扫描耗时: %v)\n", stats.TargetCount, stats.ScanDuration)
	}

	generateStart := time.Now()

	// 分发目标
	dispatch := registry.DispatchTargets(result)

	// 收集所有 gg 定义，按输出路径分组
	// key: 输出文件路径, value: []*gg.Generator (多个生成器可能输出到同一文件)
	fileDefinitions := make(map[string][]*gg.Generator)
	var allErrors []error

	// 按优先级排序生成器名称（优先级数字越小越靠前）
	genNames := make([]string, 0, len(dispatch))
	for genName := range dispatch {
		genNames = append(genNames, genName)
	}
	slices.SortFunc(genNames, func(a, b string) int {
		genA, _ := registry.GetByName(a)
		genB, _ := registry.GetByName(b)
		return genA.Priority() - genB.Priority()
	})

	// 收集生成器名称，用于添加分隔符
	// key: 输出文件路径, value: 生成器名称列表（按优先级顺序）
	fileGenNames := make(map[string][]string)

	// 先串行解析所有目标的参数（避免并发修改共享数据）
	for _, genName := range genNames {
		targets := dispatch[genName]
		gen, ok := registry.GetByName(genName)
		if !ok {
			continue
		}

		paramDefs := gen.ParamDefs()
		for _, target := range targets {
			// 创建参数结构体实例
			paramsProto := gen.NewParams()
			if paramsProto == nil {
				continue // 该生成器不需要参数
			}

			// 找到目标的注解
			var targetAnn *Annotation
			for _, ann := range target.Annotations {
				// 检查注解是否属于当前生成器
				for _, supportedAnn := range gen.Annotations() {
					if ann.Name == supportedAnn {
						targetAnn = ann
						break
					}
				}
				if targetAnn != nil {
					break
				}
			}

			if targetAnn != nil {
				// 解析注解参数到结构体
				if err := ParseAnnotationParams(targetAnn, paramsProto, paramDefs); err != nil {
					allErrors = append(allErrors, fmt.Errorf("解析参数失败: %w", err))
					continue
				}
				// 存储解析后的参数（解引用指针）
				val := reflect.ValueOf(paramsProto)
				if val.Kind() != reflect.Ptr {
					allErrors = append(allErrors, fmt.Errorf("NewParams() 必须返回指针类型, 得到: %T", paramsProto))
					continue
				}
				target.ParsedParams = val.Elem().Interface()
			}
		}
	}

	// genResultItem 存储单个生成器的执行结果
	type genResultItem struct {
		genName string
		result  *GenerateResult
		err     error
	}

	// 执行生成器的函数
	executeGenerator := func(genName string) genResultItem {
		targets := dispatch[genName]
		gen, ok := registry.GetByName(genName)
		if !ok {
			return genResultItem{genName: genName}
		}

		if opts.Verbose {
			fmt.Printf("执行生成器: %s (开始处理 %d 个目标)\n", genName, len(targets))
		}

		genCtx := &GenerateContext{
			Targets:        targets,
			PackageConfigs: result.PackageConfigs,
			DefaultOutput:  opts.Output,
			Verbose:        opts.Verbose,
		}

		nt1 := time.Now()
		genResult, err := gen.Generate(genCtx)
		if opts.Verbose {
			fmt.Printf("执行生成器: %s (耗时: %v)\n", genName, time.Since(nt1))
		}

		return genResultItem{genName: genName, result: genResult, err: err}
	}

	// 收集结果
	genResults := make(map[string]*GenerateResult)

	if opts.Async {
		// 异步执行每个生成器
		resultChan := make(chan genResultItem, len(genNames))
		var wg sync.WaitGroup

		for _, genName := range genNames {
			wg.Add(1)
			go func(genName string) {
				defer wg.Done()
				resultChan <- executeGenerator(genName)
			}(genName)
		}

		// 等待所有生成器完成
		go func() {
			wg.Wait()
			close(resultChan)
		}()

		// 收集结果
		for item := range resultChan {
			if item.err != nil {
				allErrors = append(allErrors, fmt.Errorf("生成器 %s 执行失败: %w", item.genName, item.err))
				continue
			}
			if item.result != nil {
				genResults[item.genName] = item.result
			}
		}
	} else {
		// 同步执行每个生成器
		for _, genName := range genNames {
			item := executeGenerator(genName)
			if item.err != nil {
				allErrors = append(allErrors, fmt.Errorf("生成器 %s 执行失败: %w", item.genName, item.err))
				continue
			}
			if item.result != nil {
				genResults[item.genName] = item.result
			}
		}
	}

	// 按优先级顺序处理结果
	for _, genName := range genNames {
		genResult, ok := genResults[genName]
		if !ok {
			continue
		}

		// 收集 gg 定义，按文件分组
		for path, def := range genResult.Definitions {
			fileDefinitions[path] = append(fileDefinitions[path], def)
			fileGenNames[path] = append(fileGenNames[path], genName)
		}

		// 收集原始字节输出，转换为 gg.Generator 后加入 fileDefinitions
		for path, data := range genResult.RawOutputs {
			parsedGen, err := ParseSourceToGG(data)
			if err != nil {
				allErrors = append(allErrors, fmt.Errorf("解析原始输出 %s 失败: %w", path, err))
				continue
			}
			fileDefinitions[path] = append(fileDefinitions[path], parsedGen)
			fileGenNames[path] = append(fileGenNames[path], genName)
		}

		allErrors = append(allErrors, genResult.Errors...)
	}

	// 合并同一文件的定义并写入
	for path, definitions := range fileDefinitions {
		genNames := fileGenNames[path]
		merged, err := mergeDefinitionsWithSeparator(definitions, genNames)
		if err != nil {
			allErrors = append(allErrors, fmt.Errorf("合并文件 %s 的定义失败: %w", path, err))
			continue
		}

		if err := writeGGFile(path, merged); err != nil {
			allErrors = append(allErrors, fmt.Errorf("写入文件 %s 失败: %w", path, err))
		} else {
			stats.FileCount++
			fmt.Printf("生成文件: %s\n", path)
		}
	}

	stats.GenerateDuration = time.Since(generateStart)
	stats.TotalDuration = time.Since(totalStart)

	if len(allErrors) > 0 {
		for _, e := range allErrors {
			fmt.Printf("错误: %v\n", e)
		}
		return stats, fmt.Errorf("生成过程中出现 %d 个错误", len(allErrors))
	}

	return stats, nil
}

// mergeDefinitions 合并多个 gg.Generator 定义到一个文件
func mergeDefinitions(definitions []*gg.Generator) (*gg.Generator, error) {
	if len(definitions) == 0 {
		return nil, fmt.Errorf("没有定义需要合并")
	}

	if len(definitions) == 1 {
		return definitions[0], nil
	}

	// 使用第一个定义作为基础
	merged := definitions[0]

	// 合并其他定义
	for i := 1; i < len(definitions); i++ {
		other := definitions[i]

		// 检查包名是否一致
		if merged.PackageName() != "" && other.PackageName() != "" &&
			merged.PackageName() != other.PackageName() {
			return nil, fmt.Errorf("包名不一致: %s vs %s", merged.PackageName(), other.PackageName())
		}

		// 如果基础定义没有包名，使用其他定义的
		if merged.PackageName() == "" && other.PackageName() != "" {
			merged.SetPackage(other.PackageName())
		}

		// 合并
		merged.Merge(other)
	}

	return merged, nil
}

// mergeDefinitionsWithSeparator 合并多个 gg.Generator 定义到一个文件，并添加分隔符
func mergeDefinitionsWithSeparator(definitions []*gg.Generator, genNames []string) (*gg.Generator, error) {
	if len(definitions) == 0 {
		return nil, fmt.Errorf("没有定义需要合并")
	}

	// 创建新的 generator 用于合并
	merged := gg.New()
	merged.SetHeader("Code generated by gogen. DO NOT EDIT.")

	// 收集包名
	var pkgName string
	for _, def := range definitions {
		if def.PackageName() != "" {
			if pkgName == "" {
				pkgName = def.PackageName()
			} else if pkgName != def.PackageName() {
				return nil, fmt.Errorf("包名不一致: %s vs %s", pkgName, def.PackageName())
			}
		}
	}
	if pkgName != "" {
		merged.SetPackage(pkgName)
	}

	// 合并每个定义的 body，并添加分隔符
	// 注意：不要手动收集 imports，因为 def.Imports() 只返回路径不包含别名
	// 直接使用 Merge 方法，它会正确处理 imports 和别名
	for i, def := range definitions {
		// 添加分隔符注释
		genName := "unknown"
		if i < len(genNames) {
			genName = genNames[i]
		}
		separator := fmt.Sprintf("// ================ %s ================", genName)
		merged.Body().AddLine()
		merged.Body().AddString(separator)
		merged.Body().AddLine()

		// 合并 body 内容
		merged.Merge(def)
	}

	return merged, nil
}

// writeGGFile 将 gg 定义写入文件
func writeGGFile(path string, gen *gg.Generator) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件并格式化
	return utils.WriteFormat(path, gen.Bytes())
}

// GetOutputPath 根据注解参数和默认规则计算输出路径
// 优先级：注解参数 > 包级插件配置 > 包级默认配置 > 命令行参数 > 默认文件名
// 模板变量：
//   - $FILE: 源文件名（不含 .go 后缀）
//   - $PACKAGE: 包名
func GetOutputPath(target *Target, ann *Annotation, defaultFileName string, pkgConfig *PackageConfig, pluginName string, cmdOutput string) string {
	var output string

	// 1. 优先使用注解参数
	output = ann.GetParam("output")

	// 2. 其次使用包级配置
	if output == "" && pkgConfig != nil {
		output = pkgConfig.GetPluginOutput(strings.ToLower(pluginName))
	}

	// 3. 再次使用命令行参数
	if output == "" && cmdOutput != "" {
		output = cmdOutput
	}

	// 4. 如果都没有，使用默认输出
	if output == "" {
		return GetDefaultOutputPath(target, defaultFileName)
	}

	// 处理模板变量
	output = replaceTemplateVars(output, target)

	// 确保有 .go 后缀
	if !strings.HasSuffix(output, ".go") {
		output += ".go"
	}

	if filepath.IsAbs(output) {
		return output
	}
	// 相对于源文件目录
	return filepath.Join(filepath.Dir(target.FilePath), output)
}

// replaceTemplateVars 替换模板变量
// 支持的变量：
//   - $FILE: 源文件名（不含 .go 后缀）
//   - $PACKAGE: 包名
func replaceTemplateVars(template string, target *Target) string {
	fileName := strings.TrimSuffix(filepath.Base(target.FilePath), ".go")
	template = strings.ReplaceAll(template, "$FILE", fileName)
	template = strings.ReplaceAll(template, "$PACKAGE", target.PackageName)
	return template
}

// GetDefaultOutputPath 获取包级别的默认输出路径
// 同一个包内的所有注解默认输出到同一个文件
func GetDefaultOutputPath(target *Target, defaultFileName string) string {
	if defaultFileName == "" {
		defaultFileName = "generate.go"
	}
	defaultFileName = replaceTemplateVars(defaultFileName, target)
	return filepath.Join(filepath.Dir(target.FilePath), defaultFileName)
}

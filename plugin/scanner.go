package plugin

import (
	"bufio"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// Scanner 两阶段并行注解扫描器
// 第一阶段：快速文本匹配，找出可能包含注解的文件
// 第二阶段：对匹配的文件进行 AST 解析
type Scanner struct {
	workers int
	verbose bool

	// 注解过滤器（可选）
	annotationFilter []string
}

// ScannerOption 扫描器选项
type ScannerOption func(*Scanner)

func WithWorkers(n int) ScannerOption {
	return func(s *Scanner) {
		if n > 0 {
			s.workers = n
		}
	}
}

func WithScannerVerbose(v bool) ScannerOption {
	return func(s *Scanner) {
		s.verbose = v
	}
}

func WithAnnotationFilter(annotations ...string) ScannerOption {
	return func(s *Scanner) {
		s.annotationFilter = annotations
	}
}

func NewScanner(opts ...ScannerOption) *Scanner {
	s := &Scanner{
		workers: runtime.NumCPU(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// quickMatchRegex 快速匹配注解的正则
// 匹配 @Name 或 @Name(...) 模式
var quickMatchRegex = regexp.MustCompile(`@(\w+)(?:\([^)]*\))?`)

// Scan 扫描指定路径
// 支持: ./... ./pkg/... ./pkg /abs/path/...
func (s *Scanner) Scan(ctx context.Context, patterns ...string) (*ScanResult, error) {
	// 收集所有文件
	allFiles, err := s.collectFiles(patterns)
	if err != nil {
		return nil, err
	}

	if len(allFiles) == 0 {
		return &ScanResult{}, nil
	}

	// ========== 第一阶段：快速匹配 ==========
	matchedFiles, err := s.quickMatch(ctx, allFiles)
	if err != nil {
		return nil, err
	}

	if len(matchedFiles) == 0 {
		return &ScanResult{}, nil
	}

	// ========== 第二阶段：AST 解析 ==========
	return s.parseFiles(ctx, matchedFiles)
}

// quickMatch 第一阶段：快速文本匹配
// 并行读取文件，检查是否包含 @xxx 模式
func (s *Scanner) quickMatch(ctx context.Context, files []string) ([]string, error) {
	type matchResult struct {
		file    string
		matched bool
		err     error
	}

	resultCh := make(chan matchResult, len(files))
	fileCh := make(chan string, len(files))

	// 启动工作者
	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case file, ok := <-fileCh:
					if !ok {
						return
					}
					matched, err := s.QuickMatchFile(file)
					resultCh <- matchResult{file: file, matched: matched, err: err}
				}
			}
		}()
	}

	// 发送文件
	go func() {
		for _, file := range files {
			select {
			case <-ctx.Done():
				break
			case fileCh <- file:
			}
		}
		close(fileCh)
	}()

	// 等待完成
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 收集匹配的文件
	var matchedFiles []string
	for r := range resultCh {
		if r.err != nil {
			continue // 跳过错误文件
		}
		if r.matched {
			matchedFiles = append(matchedFiles, r.file)
		}
	}

	return matchedFiles, nil
}

// QuickMatchFile 快速检查文件是否包含注解或 go:gogen 配置
// 用于 dev 模式判断文件是否需要触发代码生成
func (s *Scanner) QuickMatchFile(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 只检查注释行
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
			continue
		}

		// 检查 go:gogen: 配置（支持 //go:gogen: 和 // go:gogen:）
		if strings.Contains(trimmed, "go:gogen:") {
			return true, nil
		}

		// 查找 @xxx 模式
		matches := quickMatchRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				annName := match[1]
				// 如果有过滤器，检查是否匹配
				if len(s.annotationFilter) > 0 {
					for _, filter := range s.annotationFilter {
						if annName == filter {
							return true, nil
						}
					}
				} else {
					return true, nil
				}
			}
		}
	}

	return false, scanner.Err()
}

// parseFiles 第二阶段：AST 解析
func (s *Scanner) parseFiles(ctx context.Context, files []string) (*ScanResult, error) {
	type parseResult struct {
		structs    []*AnnotatedTarget
		interfaces []*AnnotatedTarget
		funcs      []*AnnotatedTarget
		methods    []*AnnotatedTarget
		vars       []*AnnotatedTarget
		consts     []*AnnotatedTarget
		pkgConfig  *PackageConfig
		err        error
	}

	resultCh := make(chan parseResult, len(files))
	fileCh := make(chan string, len(files))

	// 启动工作者
	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case file, ok := <-fileCh:
					if !ok {
						return
					}
					r := s.parseFile(file)
					resultCh <- parseResult{
						structs:    r.structs,
						interfaces: r.interfaces,
						funcs:      r.funcs,
						methods:    r.methods,
						vars:       r.vars,
						consts:     r.consts,
						pkgConfig:  r.pkgConfig,
						err:        r.err,
					}
				}
			}
		}()
	}

	// 发送文件
	go func() {
		for _, file := range files {
			select {
			case <-ctx.Done():
				break
			case fileCh <- file:
			}
		}
		close(fileCh)
	}()

	// 等待完成
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 收集结果
	result := &ScanResult{
		PackageConfigs: make(map[string]*PackageConfig),
	}
	for r := range resultCh {
		if r.err != nil {
			continue
		}
		result.Structs = append(result.Structs, r.structs...)
		result.Interfaces = append(result.Interfaces, r.interfaces...)
		result.Funcs = append(result.Funcs, r.funcs...)
		result.Methods = append(result.Methods, r.methods...)
		result.Vars = append(result.Vars, r.vars...)
		result.Consts = append(result.Consts, r.consts...)
		if r.pkgConfig != nil {
			pkgDir := r.pkgConfig.PackageDir
			// 如果该包已有配置，检查是否冲突
			if existing, ok := result.PackageConfigs[pkgDir]; ok {
				// 合并配置：如果新配置有值，覆盖旧配置
				if r.pkgConfig.DefaultOutput != "" {
					if existing.DefaultOutput != "" && existing.DefaultOutput != r.pkgConfig.DefaultOutput {
						fmt.Printf("警告: 包 %s 中存在多个不同的 go:gogen 默认输出配置，使用后发现的配置\n", pkgDir)
					}
					existing.DefaultOutput = r.pkgConfig.DefaultOutput
				}
				for k, v := range r.pkgConfig.PluginOutputs {
					if existingV, ok := existing.PluginOutputs[k]; ok && existingV != v {
						fmt.Printf("警告: 包 %s 中插件 %s 存在多个不同的输出配置，使用后发现的配置\n", pkgDir, k)
					}
					existing.PluginOutputs[k] = v
				}
			} else {
				result.PackageConfigs[pkgDir] = r.pkgConfig
			}
		}
	}

	return result, nil
}

// parseFile AST 解析单个文件
func (s *Scanner) parseFile(filePath string) (result struct {
	structs    []*AnnotatedTarget
	interfaces []*AnnotatedTarget
	funcs      []*AnnotatedTarget
	methods    []*AnnotatedTarget
	vars       []*AnnotatedTarget
	consts     []*AnnotatedTarget
	pkgConfig  *PackageConfig
	err        error
}) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		result.err = err
		return
	}

	packageName := file.Name.Name

	// 解析包级 go:gogen: 配置
	result.pkgConfig = s.parsePackageConfig(file, filePath)

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok == token.TYPE {
				s.parseTypeDecl(fset, filePath, packageName, d, &result)
			} else if d.Tok == token.VAR {
				s.parseVarConstDecl(fset, filePath, packageName, d, TargetVar, &result)
			} else if d.Tok == token.CONST {
				s.parseVarConstDecl(fset, filePath, packageName, d, TargetConst, &result)
			}
		case *ast.FuncDecl:
			s.parseFuncDecl(fset, filePath, packageName, d, &result)
		}
	}

	return
}

// parseTypeDecl 解析类型声明
func (s *Scanner) parseTypeDecl(fset *token.FileSet, filePath, packageName string, decl *ast.GenDecl, result *struct {
	structs    []*AnnotatedTarget
	interfaces []*AnnotatedTarget
	funcs      []*AnnotatedTarget
	methods    []*AnnotatedTarget
	vars       []*AnnotatedTarget
	consts     []*AnnotatedTarget
	pkgConfig  *PackageConfig
	err        error
}) {
	var docText string
	if decl.Doc != nil {
		docText = decl.Doc.Text()
	}

	annotations := ParseAnnotations(docText)

	// 应用过滤器
	if len(s.annotationFilter) > 0 && len(annotations) > 0 {
		annotations = FilterByNames(annotations, s.annotationFilter...)
	}

	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		target := &Target{
			Name:        typeSpec.Name.Name,
			PackageName: packageName,
			FilePath:    filePath,
			Position:    typeSpec.Pos(),
			Node:        typeSpec,
		}

		switch t := typeSpec.Type.(type) {
		case *ast.StructType:
			if len(annotations) == 0 {
				continue
			}
			target.Kind = TargetStruct
			result.structs = append(result.structs, &AnnotatedTarget{
				Target:      target,
				Annotations: annotations,
			})

		case *ast.InterfaceType:
			target.Kind = TargetInterface
			// 对于接口，还需要检查其方法的注解
			methodAnnotations := s.parseInterfaceMethodAnnotations(t)

			// 合并接口级注解和方法级注解
			allAnnotations := append([]*Annotation{}, annotations...)
			allAnnotations = append(allAnnotations, methodAnnotations...)

			// 如果有任何注解，则添加到结果中
			if len(allAnnotations) > 0 {
				result.interfaces = append(result.interfaces, &AnnotatedTarget{
					Target:      target,
					Annotations: allAnnotations,
				})
			}
		}
	}
}

// parseInterfaceMethodAnnotations 解析接口方法的注解
func (s *Scanner) parseInterfaceMethodAnnotations(interfaceType *ast.InterfaceType) []*Annotation {
	var annotations []*Annotation

	if interfaceType.Methods == nil {
		return annotations
	}

	for _, method := range interfaceType.Methods.List {
		if method.Doc == nil {
			continue
		}

		methodAnnotations := ParseAnnotations(method.Doc.Text())
		if len(s.annotationFilter) > 0 {
			methodAnnotations = FilterByNames(methodAnnotations, s.annotationFilter...)
		}
		annotations = append(annotations, methodAnnotations...)
	}

	return annotations
}

// parseFuncDecl 解析函数声明
func (s *Scanner) parseFuncDecl(fset *token.FileSet, filePath, packageName string, decl *ast.FuncDecl, result *struct {
	structs    []*AnnotatedTarget
	interfaces []*AnnotatedTarget
	funcs      []*AnnotatedTarget
	methods    []*AnnotatedTarget
	vars       []*AnnotatedTarget
	consts     []*AnnotatedTarget
	pkgConfig  *PackageConfig
	err        error
}) {
	var docText string
	if decl.Doc != nil {
		docText = decl.Doc.Text()
	}

	annotations := ParseAnnotations(docText)
	if len(annotations) == 0 {
		return
	}

	if len(s.annotationFilter) > 0 {
		annotations = FilterByNames(annotations, s.annotationFilter...)
		if len(annotations) == 0 {
			return
		}
	}

	target := &Target{
		Name:        decl.Name.Name,
		PackageName: packageName,
		FilePath:    filePath,
		Position:    decl.Pos(),
		Node:        decl,
	}

	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		target.Kind = TargetMethod
		recv := decl.Recv.List[0]

		if len(recv.Names) > 0 {
			target.ReceiverName = recv.Names[0].Name
		}
		target.ReceiverType = exprToString(recv.Type)

		result.methods = append(result.methods, &AnnotatedTarget{
			Target:      target,
			Annotations: annotations,
		})
	} else {
		target.Kind = TargetFunc
		result.funcs = append(result.funcs, &AnnotatedTarget{
			Target:      target,
			Annotations: annotations,
		})
	}
}

// collectFiles 收集所有需要扫描的文件
func (s *Scanner) collectFiles(patterns []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		recursive := strings.HasSuffix(pattern, "/...")
		if recursive {
			pattern = strings.TrimSuffix(pattern, "/...")
		}

		absPath, err := filepath.Abs(pattern)
		if err != nil {
			return nil, err
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return nil, err
		}

		if info.IsDir() {
			err := filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					name := info.Name()
					if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" {
						return filepath.SkipDir
					}
					if !recursive && path != absPath {
						return filepath.SkipDir
					}
					return nil
				}

				if strings.HasSuffix(path, ".go") &&
					!strings.HasSuffix(path, "_test.go") &&
					!strings.HasSuffix(path, "_gen.go") &&
					!strings.HasSuffix(path, "_query.go") &&
					!strings.HasSuffix(path, "_patch.go") {
					if !seen[path] {
						seen[path] = true
						files = append(files, path)
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else if strings.HasSuffix(absPath, ".go") {
			if !seen[absPath] {
				seen[absPath] = true
				files = append(files, absPath)
			}
		}
	}

	return files, nil
}

func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.IndexExpr:
		return exprToString(e.X) + "[" + exprToString(e.Index) + "]"
	default:
		return ""
	}
}

// 默认扫描器
var defaultScanner = NewScanner()

func Scan(ctx context.Context, patterns ...string) (*ScanResult, error) {
	return defaultScanner.Scan(ctx, patterns...)
}

func ScanWithFilter(ctx context.Context, annotations []string, patterns ...string) (*ScanResult, error) {
	scanner := NewScanner(WithAnnotationFilter(annotations...))
	return scanner.Scan(ctx, patterns...)
}

// goGenRegex 匹配 go:gogen: 指令
// 支持两种格式：//go:gogen: 和 // go:gogen:
var goGenRegex = regexp.MustCompile(`go:gogen:\s*(.*)`)

// parsePackageConfig 解析包级 go:gogen: 配置
// 支持格式:
//
//	//go:gogen: -output `$FILE_query`
//	// go:gogen: plugin:gsql -output `$FILE_query` plugin:setter -output `0api_generated`
func (s *Scanner) parsePackageConfig(file *ast.File, filePath string) *PackageConfig {
	var gogenLines []string

	// 收集所有 go:gogen: 注释
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			text := strings.TrimPrefix(c.Text, "//")
			text = strings.TrimPrefix(text, "/*")
			text = strings.TrimSuffix(text, "*/")
			text = strings.TrimSpace(text)

			if matches := goGenRegex.FindStringSubmatch(text); len(matches) > 1 {
				gogenLines = append(gogenLines, matches[1])
			}
		}
	}

	if len(gogenLines) == 0 {
		return nil
	}

	// 检查是否有多个 go:gogen: 定义
	if len(gogenLines) > 1 {
		fmt.Printf("警告: 文件 %s 定义了多个 go:gogen: 指令，将被忽略\n", filePath)
		return nil
	}

	return parseGogenLine(gogenLines[0], filePath)
}

// parseFileConfig 解析文件级 go:gogen: 配置（已废弃）
// Deprecated: 请使用 parsePackageConfig
func (s *Scanner) parseFileConfig(file *ast.File, filePath string) *PackageConfig {
	return s.parsePackageConfig(file, filePath)
}

// parseGogenLine 解析单行 go:gogen: 配置
// 格式:
//
//	-output `xxx`                                    // 默认输出
//	plugin:gsql -output `xxx` plugin:setter -output `yyy`  // 插件特定输出
func parseGogenLine(line string, filePath string) *PackageConfig {
	pkgDir := filepath.Dir(filePath)
	config := &PackageConfig{
		PackageDir:    pkgDir,
		PluginOutputs: make(map[string]string),
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	// 解析配置项
	// 使用简单的状态机解析
	parts := splitGogenArgs(line)

	var currentPlugin string
	for i := 0; i < len(parts); i++ {
		part := parts[i]

		if strings.HasPrefix(part, "plugin:") {
			// 切换到特定插件
			currentPlugin = strings.ToLower(strings.TrimPrefix(part, "plugin:"))
		} else if part == "-output" && i+1 < len(parts) {
			i++
			output := trimQuotes(parts[i])
			if currentPlugin == "" {
				config.DefaultOutput = output
			} else {
				config.PluginOutputs[currentPlugin] = output
			}
		}
	}

	// 如果没有任何配置，返回 nil
	if config.DefaultOutput == "" && len(config.PluginOutputs) == 0 {
		return nil
	}

	return config
}

// splitGogenArgs 分割 go:gogen 参数，支持引号内的空格
func splitGogenArgs(line string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(line); i++ {
		c := line[i]

		if !inQuote && (c == '`' || c == '"' || c == '\'') {
			inQuote = true
			quoteChar = c
			current.WriteByte(c)
		} else if inQuote && c == quoteChar {
			inQuote = false
			current.WriteByte(c)
			quoteChar = 0
		} else if !inQuote && c == ' ' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// trimQuotes 去除引号
func trimQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '`' && s[len(s)-1] == '`') ||
			(s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// parseVarConstDecl 解析 var/const 声明
func (s *Scanner) parseVarConstDecl(fset *token.FileSet, filePath, packageName string, decl *ast.GenDecl, kind TargetKind, result *struct {
	structs    []*AnnotatedTarget
	interfaces []*AnnotatedTarget
	funcs      []*AnnotatedTarget
	methods    []*AnnotatedTarget
	vars       []*AnnotatedTarget
	consts     []*AnnotatedTarget
	pkgConfig  *PackageConfig
	err        error
}) {
	var docText string
	if decl.Doc != nil {
		docText = decl.Doc.Text()
	}

	annotations := ParseAnnotations(docText)
	if len(annotations) == 0 {
		return
	}

	if len(s.annotationFilter) > 0 {
		annotations = FilterByNames(annotations, s.annotationFilter...)
		if len(annotations) == 0 {
			return
		}
	}

	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for _, name := range valueSpec.Names {
			if name.Name == "_" {
				continue // 跳过匿名变量
			}

			target := &Target{
				Kind:        kind,
				Name:        name.Name,
				PackageName: packageName,
				FilePath:    filePath,
				Position:    valueSpec.Pos(),
				Node:        valueSpec,
			}

			annotatedTarget := &AnnotatedTarget{
				Target:      target,
				Annotations: annotations,
			}

			if kind == TargetVar {
				result.vars = append(result.vars, annotatedTarget)
			} else {
				result.consts = append(result.consts, annotatedTarget)
			}
		}
	}
}

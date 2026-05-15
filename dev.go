package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/donutnomad/gogen/plugin"
	"github.com/fsnotify/fsnotify"
	"golang.org/x/tools/imports"
)

// DevOptions dev 命令选项
type DevOptions struct {
	Patterns        []string      // 监听的路径模式
	Verbose         bool          // 详细输出
	Output          string        // 默认输出路径
	NoOutput        bool          // 禁用默认输出
	Async           bool          // 异步执行
	Debounce        time.Duration // 防抖动时间
	OriginalArgs    []string      // 原始命令参数，用于重启
	ToolArgs        []string      // 传递给 go tool gogen 的全局参数
	GenerateCommand func(context.Context, []string) error
	RestartCommand  func([]string) error
}

// devRunner 处理文件变动的核心逻辑
type devRunner struct {
	opts     *DevOptions
	registry *plugin.Registry
	watcher  *fsnotify.Watcher
	scanner  *plugin.Scanner
	ctx      context.Context // 用于响应退出信号

	// 防抖动相关
	mu             sync.Mutex
	pendingDirs    map[string]*time.Timer // key: 包目录路径
	restartPending *time.Timer
}

// runDev 启动开发模式
func runDev(args []string) {
	patterns := args
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	registry := plugin.Global()
	if len(registry.Generators()) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 没有已注册的生成器")
		os.Exit(1)
	}

	outputPath := *output
	if *noOutput {
		outputPath = ""
	}

	opts := &DevOptions{
		Patterns:     patterns,
		Verbose:      *verbose,
		Output:       outputPath,
		NoOutput:     *noOutput,
		Async:        *async,
		Debounce:     1 * time.Second,
		OriginalArgs: append([]string(nil), os.Args[1:]...),
	}
	opts.ToolArgs = buildToolArgs(opts)

	if err := dev(opts); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

// dev 启动开发模式
func dev(opts *DevOptions) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n正在退出...")
		cancel()
	}()

	// 创建 watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建文件监听器失败: %w", err)
	}
	defer watcher.Close()

	registry := plugin.Global()
	annotations := registry.Annotations()

	runner := &devRunner{
		opts:        opts,
		registry:    registry,
		watcher:     watcher,
		scanner:     plugin.NewScanner(plugin.WithAnnotationFilter(annotations...)),
		ctx:         ctx,
		pendingDirs: make(map[string]*time.Timer),
	}

	// 清理函数：退出时停止所有待处理的定时器
	defer func() {
		runner.mu.Lock()
		for _, timer := range runner.pendingDirs {
			timer.Stop()
		}
		if runner.restartPending != nil {
			runner.restartPending.Stop()
		}
		runner.mu.Unlock()
	}()

	// 收集并添加监听目录
	dirs, err := collectWatchDirs(opts.Patterns)
	if err != nil {
		return fmt.Errorf("收集监听目录失败: %w", err)
	}

	if len(dirs) == 0 {
		return fmt.Errorf("没有找到需要监听的目录")
	}

	moduleDir, err := moduleRoot()
	if err != nil {
		return fmt.Errorf("定位模块根目录失败: %w", err)
	}
	if !containsDir(dirs, moduleDir) {
		dirs = append(dirs, moduleDir)
	}

	for _, dir := range dirs {
		if err := watcher.Add(dir); err != nil {
			return fmt.Errorf("添加监听目录失败 %s: %w", dir, err)
		}
		if opts.Verbose {
			fmt.Printf("监听目录: %s\n", dir)
		}
	}

	fmt.Printf("开发模式已启动，监听 %d 个目录\n", len(dirs))
	fmt.Println("按 Ctrl+C 退出")
	fmt.Println()

	// 启动事件处理循环
	return runner.watchLoop(ctx)
}

// watchLoop 事件处理循环
func (r *devRunner) watchLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-r.watcher.Events:
			if !ok {
				return nil
			}
			r.handleEvent(event)

		case err, ok := <-r.watcher.Errors:
			if !ok {
				return nil
			}
			if r.opts.Verbose {
				fmt.Printf("监听错误: %v\n", err)
			}
		}
	}
}

// handleEvent 处理文件事件
func (r *devRunner) handleEvent(event fsnotify.Event) {
	filePath := event.Name

	if isModuleFile(filePath) && event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
		r.scheduleRestart()
		return
	}

	// 处理删除事件：从监听列表中移除已删除的目录
	if event.Op&fsnotify.Remove != 0 {
		// fsnotify 会自动移除已删除的目录，但我们记录日志
		if r.opts.Verbose {
			fmt.Printf("检测到删除: %s\n", filePath)
		}
		return
	}

	// 只关注 Write 和 Create 事件
	if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
		return
	}

	// 检查是否是新建目录，如果是则添加到监听列表
	if event.Op&fsnotify.Create != 0 {
		if info, err := os.Stat(filePath); err == nil && info.IsDir() {
			// 跳过隐藏目录、vendor 和 testdata
			name := info.Name()
			if !strings.HasPrefix(name, ".") && name != "vendor" && name != "testdata" {
				if err := r.watcher.Add(filePath); err == nil {
					if r.opts.Verbose {
						fmt.Printf("添加监听目录: %s\n", filePath)
					}
				}
			}
			return
		}
	}

	// 只处理 .go 文件
	if !strings.HasSuffix(filePath, ".go") {
		return
	}

	// 跳过生成的文件（通过文件名后缀判断）
	if isGeneratedFile(filePath) {
		return
	}

	// 跳过生成的文件（通过文件头部注释判断）
	if isGeneratedByContent(filePath) {
		if r.opts.Verbose {
			fmt.Printf("跳过生成文件: %s\n", filePath)
		}
		return
	}

	if r.opts.Verbose {
		fmt.Printf("检测到文件变化: %s\n", filePath)
	}

	// 检查文件是否包含注解
	hasAnnotation, err := r.scanner.QuickMatchFile(filePath)
	if err != nil {
		if r.opts.Verbose {
			fmt.Printf("检查注解失败 %s: %v\n", filePath, err)
		}
		return
	}

	if !hasAnnotation {
		if r.opts.Verbose {
			fmt.Printf("跳过文件（无注解）: %s\n", filePath)
		}
		return
	}

	// 检查语法错误
	if err := checkSyntax(filePath); err != nil {
		fmt.Printf("语法错误 %s: %v\n", filePath, err)
		return
	}

	// 获取包目录并触发防抖动生成
	pkgDir := filepath.Dir(filePath)
	r.scheduleGenerate(pkgDir)
}

// scheduleGenerate 防抖动调度生成
func (r *devRunner) scheduleGenerate(pkgDir string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 取消之前的 timer
	if timer, exists := r.pendingDirs[pkgDir]; exists {
		timer.Stop()
	}

	// 创建新的 timer
	r.pendingDirs[pkgDir] = time.AfterFunc(r.opts.Debounce, func() {
		// 检查 context 是否已取消
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		r.runGenerate(pkgDir)

		r.mu.Lock()
		delete(r.pendingDirs, pkgDir)
		r.mu.Unlock()
	})
}

func (r *devRunner) scheduleRestart() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.restartPending != nil {
		r.restartPending.Stop()
	}

	r.restartPending = time.AfterFunc(r.opts.Debounce, func() {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		r.restart()
	})
}

// runGenerate 执行实际的代码生成
func (r *devRunner) runGenerate(pkgDir string) {
	if r.opts.Verbose {
		fmt.Printf("触发代码生成: %s\n", pkgDir)
	}

	args := buildGenerateToolArgs(r.opts, pkgDir)
	err := runGenerateCommand(r.ctx, r.opts, args)
	if err != nil {
		fmt.Printf("生成失败: %v\n", err)
		return
	}

	if r.opts.Verbose {
		fmt.Printf("生成命令完成: go %s\n", strings.Join(args, " "))
	}
}

func (r *devRunner) restart() {
	args := buildRestartToolArgs(r.opts)
	fmt.Printf("检测到 go.mod/go.sum 变化，重启: go %s\n", strings.Join(args, " "))

	if err := runRestartCommand(r.opts, args); err != nil {
		fmt.Printf("重启失败: %v\n", err)
	}
}

func buildGenerateToolArgs(opts *DevOptions, pkgDir string) []string {
	args := []string{"tool", "gogen"}
	args = append(args, buildToolArgs(opts)...)
	args = append(args, "gen", pkgDir)
	return args
}

func buildRestartToolArgs(opts *DevOptions) []string {
	args := []string{"tool", "gogen"}
	args = append(args, opts.OriginalArgs...)
	return args
}

func buildToolArgs(opts *DevOptions) []string {
	if len(opts.ToolArgs) > 0 {
		return append([]string(nil), opts.ToolArgs...)
	}

	var args []string
	if opts.Verbose {
		args = append(args, "-v")
	}
	if opts.NoOutput {
		args = append(args, "-no-output")
	} else if opts.Output != "" {
		args = append(args, "-output", opts.Output)
	}
	args = append(args, fmt.Sprintf("-async=%t", opts.Async))
	return args
}

func runGenerateCommand(ctx context.Context, opts *DevOptions, args []string) error {
	if opts.GenerateCommand != nil {
		return opts.GenerateCommand(ctx, args)
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runRestartCommand(opts *DevOptions, args []string) error {
	if opts.RestartCommand != nil {
		return opts.RestartCommand(args)
	}

	goPath, err := exec.LookPath("go")
	if err != nil {
		return err
	}

	return syscall.Exec(goPath, append([]string{"go"}, args...), os.Environ())
}

// checkSyntax 检查文件语法
func checkSyntax(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	_, err = imports.Process(filePath, content, &imports.Options{
		Fragment:   true,
		AllErrors:  true,
		Comments:   true,
		FormatOnly: true, // 只检查语法，不修改 imports
	})

	return err
}

// collectWatchDirs 收集所有需要监听的目录
func collectWatchDirs(patterns []string) ([]string, error) {
	var dirs []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		recursive := strings.HasSuffix(pattern, "/...")
		baseDir := strings.TrimSuffix(pattern, "/...")

		absDir, err := filepath.Abs(baseDir)
		if err != nil {
			return nil, err
		}

		info, err := os.Stat(absDir)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			continue
		}

		if recursive {
			// 递归收集所有子目录
			err := filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if !info.IsDir() {
					return nil
				}

				// 跳过隐藏目录、vendor 和 testdata
				name := info.Name()
				if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" {
					return filepath.SkipDir
				}

				if !seen[path] {
					seen[path] = true
					dirs = append(dirs, path)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			if !seen[absDir] {
				seen[absDir] = true
				dirs = append(dirs, absDir)
			}
		}
	}

	return dirs, nil
}

func moduleRoot() (string, error) {
	absDir, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(absDir, "go.mod")); err == nil {
			return absDir, nil
		}

		parent := filepath.Dir(absDir)
		if parent == absDir {
			return "", fmt.Errorf("go.mod not found")
		}
		absDir = parent
	}
}

func containsDir(dirs []string, target string) bool {
	for _, dir := range dirs {
		if dir == target {
			return true
		}
	}
	return false
}

func isModuleFile(filePath string) bool {
	base := filepath.Base(filePath)
	return base == "go.mod" || base == "go.sum"
}

// isGeneratedFile 检查是否是生成的文件（通过文件名后缀）
func isGeneratedFile(filePath string) bool {
	base := filepath.Base(filePath)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "_gen.go") ||
		strings.HasSuffix(base, "_query.go") ||
		strings.HasSuffix(base, "_patch.go") ||
		strings.HasSuffix(base, "_setter.go") ||
		strings.HasSuffix(base, "_slice.go") ||
		strings.HasSuffix(base, "_mock.go")
}

// isGeneratedByContent 检查文件是否包含代码生成标记（通过读取文件头部）
func isGeneratedByContent(filePath string) bool {
	// 只读取文件的前几行来检查
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// 读取前 512 字节足够检查头部注释
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && n == 0 {
		return false
	}

	content := string(buf[:n])
	// 检查是否包含 "DO NOT EDIT" 标记
	return strings.Contains(content, "DO NOT EDIT")
}

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/donutnomad/gogen/codegen"
	"github.com/donutnomad/gogen/gormgen"
	"github.com/donutnomad/gogen/mockgen"
	"github.com/donutnomad/gogen/plugin"
	"github.com/donutnomad/gogen/settergen"
	"github.com/donutnomad/gogen/slicegen"
	"github.com/donutnomad/gogen/stateflowgen"
	"github.com/donutnomad/gogen/swaggen"
	"github.com/samber/lo"
)

func init() {
	// 集中注册所有生成器
	plugin.MustRegister(gormgen.NewGsqlGenerator())
	plugin.MustRegister(settergen.NewSetterGenerator())
	plugin.MustRegister(slicegen.NewSliceGenerator())
	plugin.MustRegister(mockgen.NewMockGenerator())
	plugin.MustRegister(swaggen.NewSwagGenerator())
	plugin.MustRegister(codegen.NewCodeGenerator())
	plugin.MustRegister(stateflowgen.NewStateFlowGenerator())
}

var (
	verbose  = flag.Bool("v", false, "详细输出")
	help     = flag.Bool("h", false, "显示帮助信息")
	output   = flag.String("output", "generate.go", "默认输出路径（支持模板变量 $FILE, $PACKAGE）")
	noOutput = flag.Bool("no-output", false, "禁用默认输出（每个生成器输出到独立文件）")
	async    = flag.Bool("async", true, "异步执行生成器（默认 true）")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	if *help {
		usage()
		os.Exit(0)
	}

	args := flag.Args()

	// 默认命令是 gen
	if len(args) == 0 {
		runGen([]string{"./..."})
		return
	}

	// 检查是否是子命令
	cmd := args[0]
	switch cmd {
	case "gen":
		runGen(args[1:])
	case "dev":
		runDev(args[1:])
	default:
		// 不是子命令，当作路径参数处理，执行 gen
		runGen(args)
	}
}

func runGen(args []string) {
	// 获取扫描路径
	patterns := args
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	// 检查是否有已注册的生成器
	registry := plugin.Global()
	if len(registry.Generators()) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 没有已注册的生成器")
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("已注册 %d 个生成器:\n", len(registry.Generators()))
		for _, gen := range registry.Generators() {
			anns := lo.Map(gen.Annotations(), func(item string, index int) string {
				return "@" + item
			})
			fmt.Printf("  - %s (%s)\n", gen.Name(), strings.Join(anns, ","))
		}
		fmt.Println()
	}

	// 运行代码生成
	ctx := context.Background()

	// 确定输出路径：-no-output 时传空字符串，否则使用 -output 的值
	outputPath := *output
	if *noOutput {
		outputPath = ""
	}

	opts := &plugin.RunOptions{
		Registry: registry,
		Patterns: patterns,
		Verbose:  *verbose,
		Output:   outputPath,
		Async:    *async,
	}

	stats, err := plugin.RunWithOptionsAndStats(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	// 输出统计信息
	if stats != nil && (stats.FileCount > 0 || *verbose) {
		fmt.Printf("\n统计: 扫描 %d 个目标, 生成 %d 个文件\n", stats.TargetCount, stats.FileCount)
		fmt.Printf("耗时: 扫描 %v, 生成 %v, 总计 %v\n", stats.ScanDuration, stats.GenerateDuration, stats.TotalDuration)
	}
}

func usage() {
	_, _ = fmt.Fprintf(os.Stderr, `gogen - Go 代码生成工具

用法:
  gogen [选项] [路径...]
  gogen gen [选项] [路径...]
  gogen dev [选项] [路径...]

命令:
  gen     执行代码生成（默认）
  dev     启动开发模式，监听文件变动自动生成

路径:
  支持 Go 包路径模式，如:
    ./...          递归扫描当前目录及子目录（默认）
    ./pkg/...      递归扫描指定目录
    ./models/...   递归扫描 models 目录

选项:
`)
	flag.PrintDefaults()

	// 动态生成注解帮助信息
	registry := plugin.Global()
	if len(registry.Generators()) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "\n支持的注解:\n")
		_, _ = fmt.Fprint(os.Stderr, plugin.FormatHelpText(registry))
	}

	_, _ = fmt.Fprintf(os.Stderr, `模板变量:
  $FILE     - 源文件名（不含 .go 后缀）
  $PACKAGE  - 包名

示例:
  gogen                                     扫描当前目录（默认 ./...）
  gogen ./...                               递归扫描当前目录
  gogen -v ./models/...                     详细模式扫描 models 目录
  gogen -output $FILE_gen ./...             指定输出文件名
  gogen -no-output ./...                    每个生成器输出到独立文件
  gogen dev ./...                           开发模式，监听文件变动
  gogen -v dev ./models/...                 开发模式，详细输出
`)
}

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/donutnomad/gogen/gormgen"
	"github.com/donutnomad/gogen/plugin"
	"github.com/donutnomad/gogen/settergen"
	"github.com/donutnomad/gogen/slicegen"
)

func init() {
	// 集中注册所有生成器
	plugin.MustRegister(gormgen.NewGsqlGenerator())
	plugin.MustRegister(settergen.NewSetterGenerator())
	plugin.MustRegister(slicegen.NewSliceGenerator())
}

var (
	verbose = flag.Bool("v", false, "详细输出")
	help    = flag.Bool("h", false, "显示帮助信息")
	output  = flag.String("output", "", "默认输出路径（支持模板变量 $FILE, $PACKAGE）")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	if *help {
		usage()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	// 检查子命令
	cmd := args[0]
	switch cmd {
	case "gen":
		runGen(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", cmd)
		usage()
		os.Exit(1)
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
			fmt.Printf("  - %s (注解: %v)\n", gen.Name(), gen.Annotations())
		}
		fmt.Println()
	}

	// 运行代码生成
	ctx := context.Background()

	opts := &plugin.RunOptions{
		Registry: registry,
		Patterns: patterns,
		Verbose:  *verbose,
		Output:   *output,
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
	fmt.Fprintf(os.Stderr, `gotoolkit - Go 代码生成工具

用法:
  gotoolkit <命令> [选项] [路径...]

命令:
  gen     运行代码生成

路径:
  支持 Go 包路径模式，如:
    ./...          递归扫描当前目录及子目录
    ./pkg/...      递归扫描指定目录
    ./models/...   递归扫描 models 目录

选项:
`)
	flag.PrintDefaults()

	// 动态生成注解帮助信息
	registry := plugin.Global()
	if len(registry.Generators()) > 0 {
		fmt.Fprintf(os.Stderr, "\n支持的注解:\n")
		fmt.Fprint(os.Stderr, plugin.FormatHelpText(registry))
	}

	fmt.Fprintf(os.Stderr, `模板变量:
  $FILE     - 源文件名（不含 .go 后缀）
  $PACKAGE  - 包名

示例:
  gotoolkit gen ./...                       递归扫描当前目录
  gotoolkit -v gen ./models/...             详细模式扫描 models 目录
  gotoolkit -output $FILE_gen gen ./...     指定输出文件名
`)
}

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/donutnomad/gogen/approvegen/generator"
	"github.com/donutnomad/gogen/internal/utils"
)

var (
	paths           = flag.String("path", "", "dir paths, separated by comma")
	outputFileName_ = flag.String("out", "", "output filename")
	pkgName         = flag.String("pkgname", "", "package name prefix for MethodName()")
)

func main() {
	flag.Parse()
	outputName := *outputFileName_
	if *paths == "" || outputName == "" {
		fmt.Println("type parameter is required")
		return
	}
	bs, err := generator.Generate(*paths, *pkgName)
	if err != nil {
		panic(err)
	}
	if len(bs) == 0 {
		fmt.Println("[approveGen] 没有内容输出")
		return
	}
	err = utils.WriteFormat(outputName, bs)
	if err != nil {
		panic(err)
	}
	pwd, _ := os.Getwd()
	fmt.Println("[approveGen] Success:", filepath.Join(pwd, outputName))
}

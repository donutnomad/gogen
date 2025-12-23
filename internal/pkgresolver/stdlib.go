package pkgresolver

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// StdLibScanner 标准库扫描器
type StdLibScanner struct {
	goroot   string
	stdPkgs  map[string]bool // 标准库包路径集合
	initOnce sync.Once
	initErr  error
}

// NewStdLibScanner 创建标准库扫描器
func NewStdLibScanner() *StdLibScanner {
	return &StdLibScanner{
		stdPkgs: make(map[string]bool),
	}
}

// Init 初始化扫描器（延迟初始化）
func (s *StdLibScanner) Init() error {
	s.initOnce.Do(func() {
		// 获取 GOROOT
		s.goroot = build.Default.GOROOT
		if s.goroot == "" {
			s.goroot = os.Getenv("GOROOT")
		}
		if s.goroot == "" {
			s.initErr = fmt.Errorf("无法获取 GOROOT")
			return
		}

		// 扫描 $GOROOT/src
		srcDir := filepath.Join(s.goroot, "src")
		s.initErr = s.scanDir(srcDir, "")
	})
	return s.initErr
}

// scanDir 递归扫描目录
func (s *StdLibScanner) scanDir(dir, pkgPath string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // 忽略无法读取的目录
	}

	hasGoFiles := false
	for _, entry := range entries {
		name := entry.Name()

		// 跳过隐藏目录、testdata、vendor 等
		if strings.HasPrefix(name, ".") ||
			name == "testdata" ||
			name == "vendor" ||
			name == "internal" { // 标准库的 internal 不对外
			continue
		}

		fullPath := filepath.Join(dir, name)

		if entry.IsDir() {
			// 构建包路径
			newPkgPath := name
			if pkgPath != "" {
				newPkgPath = pkgPath + "/" + name
			}
			s.scanDir(fullPath, newPkgPath)
		} else if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			hasGoFiles = true
		}
	}

	// 如果当前目录包含 Go 文件，记录为标准库包
	if hasGoFiles && pkgPath != "" {
		s.stdPkgs[pkgPath] = true
	}

	return nil
}

// IsStdLib 判断是否是标准库
func (s *StdLibScanner) IsStdLib(importPath string) (bool, error) {
	if err := s.Init(); err != nil {
		return false, err
	}

	// 快速判断：标准库不含域名（不含第一个斜杠前的点）
	firstSlash := strings.Index(importPath, "/")
	if firstSlash == -1 {
		// 没有斜杠，如 fmt, os, math
		_, isStd := s.stdPkgs[importPath]
		return isStd, nil
	}

	firstPart := importPath[:firstSlash]
	if strings.Contains(firstPart, ".") {
		// 第一部分包含点，肯定是第三方包
		return false, nil
	}

	// 查表确认
	_, isStd := s.stdPkgs[importPath]
	return isStd, nil
}

// GetStdLibPath 获取标准库的磁盘路径
func (s *StdLibScanner) GetStdLibPath(importPath string) (string, error) {
	isStd, err := s.IsStdLib(importPath)
	if err != nil {
		return "", err
	}
	if !isStd {
		return "", fmt.Errorf("%s 不是标准库", importPath)
	}

	return filepath.Join(s.goroot, "src", filepath.FromSlash(importPath)), nil
}

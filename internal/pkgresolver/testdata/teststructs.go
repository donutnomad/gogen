package testdata

import (
	cc "github.com/donutnomad/gogen/internal/pkgresolver/testdata/aliasedpkg"
	"github.com/donutnomad/gogen/internal/pkgresolver/testdata/gg"
	"time"
)

// TestStructWithAlias 测试显式别名场景
// import cc "github.com/donutnomad/gogen/internal/pkgresolver/testdata/aliasedpkg"
// 使用 cc.SomeType
type TestStructWithAlias struct {
	ID        int
	Name      string
	CreatedAt time.Time
	Data      cc.SomeType // 使用别名 cc
}

// TestStructWithMismatchedPkgName 测试包名与文件夹名不一致的场景
// 文件夹是 gg，但 package 声明是 g2
// import "github.com/donutnomad/gogen/internal/pkgresolver/testdata/gg"
// 使用 g2.Type
type TestStructWithMismatchedPkgName struct {
	ID    int
	Value gg.Type // 文件夹是 gg，但实际包名是 g2
}

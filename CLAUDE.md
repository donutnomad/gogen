# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

gogen 是一个基于注解的 Go 代码生成工具集，通过扫描 Go 源文件中的注解自动生成相关代码。

## 常用命令

```bash
# 构建主工具
go build -o gotoolkit ./cmd

# 运行代码生成（扫描当前目录及子目录）
./gotoolkit gen ./...

# 详细模式运行
./gotoolkit -v gen ./models/...

# 运行测试
go test ./...

# 运行单个包的测试
go test ./automap/...

# 运行特定测试
go test -run TestGeneratorBasic ./automap/
```

## 架构设计

### 插件系统 (`plugin/`)

核心框架，定义了代码生成器的接口和注册机制：

- `Generator` 接口：所有生成器必须实现此接口
- `Registry`：生成器注册表，管理所有已注册的生成器
- 注解解析：支持 `@Name(key=value)` 格式的注解

```go
// 实现新生成器的基本模式
type MyGenerator struct {
    plugin.BaseGenerator
}

func NewMyGenerator() *MyGenerator {
    return &MyGenerator{
        BaseGenerator: *plugin.NewBaseGeneratorWithParamsStruct(
            "name",
            []string{"AnnotationName"},
            []plugin.TargetKind{plugin.TargetStruct},
            MyParams{},
        ),
    }
}
```

### 内置生成器

1. **gormgen** (`gormgen/`)
   - 注解：`@Gsql(prefix="xxx")`
   - 功能：为 GORM 模型生成类型安全的 Schema 和查询辅助代码
   - 输出：`*_query.go`

2. **settergen** (`settergen/`)
   - 注解：`@Setter(patch="v2|full|none", patch_mapper="Type.Method")`
   - 功能：生成 Patch/Setter 相关代码
   - 输出：`*_setter.go`

3. **slicegen** (`slicegen/`)
   - 注解：`@Slice(exclude=[a,b,c], include=[a,b,c], ptr=true, methods=[filter,map,reduce,sort,groupby])`
   - 功能：为结构体切片生成 Filter/Map/Sort 等辅助方法
   - 输出：`*_slice.go`
   - 参数说明：
     - `exclude`: 排除的字段列表
     - `include`: 包含的字段列表（优先于 exclude）
     - `ptr`: 是否生成指针类型（默认 true）
     - `methods`: 额外生成的方法（filter, map, reduce, sort, groupby）

### 核心内部包

- `internal/structparse/`：Go 结构体解析，支持嵌入字段、方法解析
- `internal/gormparse/`：GORM 模型解析，提取字段、列名、表名
- `internal/pkgresolver/`：包路径解析和类型查找
- `internal/xast/`：AST 辅助工具
- `automap/`：Domain 到 PO 的映射关系分析

## 注解格式

```go
// @AnnotationName(param1=`value1`, param2="value2")
type MyStruct struct {}
```

参数值支持三种格式：
- 反引号：`key=\`value\``
- 双引号：`key="value"`
- 无引号：`key=value`

## 代码生成使用 gg 库

所有生成器使用 `github.com/donutnomad/gg` 库构建生成代码：

```go
gen := gg.New()
gen.SetPackage("mypackage")
gen.NewStruct("MyType").AddField("Name", "string")
```

## Go LSP 工具

本项目已配置 Go LSP (gopls)，可通过 LSP 工具进行代码智能分析。

### 可用操作

| 操作 | 说明 |
|------|------|
| `goToDefinition` | 跳转到符号定义位置 |
| `findReferences` | 查找符号的所有引用 |
| `hover` | 获取悬停信息（文档、类型） |
| `documentSymbol` | 获取文件中的所有符号 |
| `workspaceSymbol` | 在整个工作区搜索符号 |
| `goToImplementation` | 查找接口的实现 |
| `prepareCallHierarchy` | 获取调用层次项 |
| `incomingCalls` | 查找调用某函数的所有位置 |
| `outgoingCalls` | 查找某函数调用的所有函数 |

### 使用示例

```
# 查找 ParseStruct 函数的所有调用者
LSP incomingCalls /path/to/parser.go:13:6

# 获取文件中的所有符号
LSP documentSymbol /path/to/plugin.go:1:1

# 跳转到定义
LSP goToDefinition /path/to/file.go:line:character
```

### 注意事项

- `line` 和 `character` 参数均为 1-based（从 1 开始计数）
- 使用前需确保光标位置在有效的标识符上
- 对于方法调用分析，优先使用 `incomingCalls` 和 `outgoingCalls`

- 禁止向/tmp中写入代码进行执行和测试
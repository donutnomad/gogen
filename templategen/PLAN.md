# templategen 实现计划

## 概述

创建一个基于 Go template 的代码生成器 `templategen`，通过 `//go:gogen:` 文件级指令触发，支持 `@Define` 和 `@Import` 注解提供元数据，智能处理 import 路径映射，并集成 Sprig 库提供丰富的模板函数。

## 核心设计

### 触发方式

使用文件级 `go:gogen:` 指令触发 templategen：

```go
//go:gogen: plugin:templategen -template ./templates/service.tmpl
//go:gogen: plugin:templategen -template ./templates/repo.tmpl -output $FILE_repo.go

package myservice
```

参数：
- `-template`: 模板文件路径（必需，支持多个 -template 指令）
- `-output`: 输出文件路径（可选，默认 `$FILE_gen.go`）

### 元数据注解

#### `@Define` - 变量定义注解

可修饰：struct、interface、method。**只有带注解的目标才会被模板访问到。**

```go
// @Define(name=Config, reader=io.Reader, timeout="30s")
type MyService struct {}

// @Define(name=Handler, errType=errors.Error)
func (s *MyService) Handle() error {}
```

参数：
- `name`: 变量分组名，模板中通过 `.Defines.Config.reader` 访问
- 其他 `k=v` 对：
  - 带双引号 `"value"` → 字符串值
  - 不带引号 `Type` → 类型引用（需要解析 import）

#### `@Import` - 包导入注解（辅助）

```go
// @Import(alias=k8s, path="k8s.io/api/core/v1")
```

用于预定义长路径包的别名，避免在 @Define 中写冗长路径。

## Import 路径解析策略（四层优先级）

### 优先级 1：当前文件 import 上下文
```go
import (
    "io"
    myutil "github.com/company/project/pkg/util"
)

// @Define(name=Config, reader=io.Reader, helper=myutil.Helper)
```
- 解析当前文件的 import 声明
- 建立 `别名 → 完整路径` 映射
- 用户使用 IDE 已知的包名，体验最佳

### 优先级 2：`@Import` 注解定义
```go
// @Import(alias=k8s, path="k8s.io/api/core/v1")
// @Define(name=PodWrapper, pod=k8s.Pod)
```
- 检查是否有对应的 @Import 注解
- 适用于当前文件未引用但生成代码需要的包

### 优先级 3：全限定路径（引号包裹）
```go
// @Define(name=User, errType="github.com/pkg/errors.StackTracer")
```
- 带引号的值视为 `包路径.类型名` 格式
- 自动拆分为 import path 和类型名
- 推导包名（取路径最后一段）

### 优先级 4：标准库内置白名单
```go
// @Define(name=IO, reader=io.Reader, ctx=context.Context)
```
- 内置常用标准库映射（io, fmt, context, time, net/http 等）
- 无需 import 或 @Import 即可直接使用

## 模板数据结构

```go
// TemplateData 提供给模板的数据
type TemplateData struct {
    // 文件信息
    File FileInfo

    // 带 @Define 注解的结构体（只有带注解的才会出现）
    Structs []StructData

    // 带 @Define 注解的接口
    Interfaces []InterfaceData

    // 文件级 @Import 定义
    ImportAliases map[string]string // alias -> full path

    // 导入管理器（供模板动态添加 import）
    Imports *ImportManager
}

// FileInfo 文件信息
type FileInfo struct {
    Path        string // 源文件完整路径
    Dir         string // 所在目录
    Name        string // 文件名（不含扩展名）
    PackageName string // 包名
}

// StructData 结构体数据（带注解的）
type StructData struct {
    Name    string      // 结构体名
    Fields  []FieldData // 字段列表
    Defines DefineGroup // @Define 定义的元数据

    // 带 @Define 注解的方法（按 receiver 分组）
    Methods []MethodData
}

// InterfaceData 接口数据
type InterfaceData struct {
    Name    string       // 接口名
    Methods []MethodSig  // 方法签名
    Defines DefineGroup  // @Define 定义的元数据
}

// MethodData 方法数据（带注解的）
type MethodData struct {
    Name         string       // 方法名
    ReceiverName string       // receiver 变量名
    ReceiverType string       // receiver 类型
    IsPointer    bool         // receiver 是否为指针
    Params       []ParamData  // 参数列表
    Returns      []ReturnData // 返回值列表
    Defines      DefineGroup  // 该方法的 @Define 元数据
}

// DefineGroup 按 name 分组的定义
// 模板访问: .Defines.Config.reader
type DefineGroup map[string]map[string]TypeRef

// TypeRef 类型引用
type TypeRef struct {
    IsString   bool   // 是否为字符串值
    StringVal  string // 字符串值（当 IsString=true）
    TypeName   string // 类型名 (如 Reader, Helper)
    PkgPath    string // 完整包路径
    PkgAlias   string // 使用的别名
    FullType   string // 完整类型表达式 (如 io.Reader)
}

// FieldData 字段信息
type FieldData struct {
    Name    string
    Type    string
    Tag     string
    Comment string
}

// ParamData 参数信息
type ParamData struct {
    Name string
    Type string
}

// ReturnData 返回值信息
type ReturnData struct {
    Name string // 可能为空
    Type string
}
```

## 模板函数

### Sprig 库函数
集成 [Sprig](https://masterminds.github.io/sprig/) 提供的 100+ 函数：
- 字符串：`trim`, `lower`, `upper`, `camelcase`, `snakecase`
- 列表：`first`, `last`, `append`, `prepend`, `reverse`
- 字典：`dict`, `get`, `set`, `keys`, `values`
- 类型转换：`int`, `float64`, `toString`
- 日期：`now`, `date`, `dateModify`

### 自定义 Helper 函数

```go
// 类型相关
"typeName":    func(t TypeRef) string { return t.TypeName }
"fullType":    func(t TypeRef) string { return t.FullType }
"isString":    func(t TypeRef) bool { return t.IsString }

// Import 管理
"import":      func(path string) string { /* 添加 import 并返回包名 */ }
"importAlias": func(path, alias string) string { /* 带别名导入 */ }

// 代码生成
"receiver":    func(name string) string { return strings.ToLower(name[:1]) }
"exported":    func(name string) string { return strings.Title(name) }
"unexported":  func(name string) string { return strings.ToLower(name[:1]) + name[1:] }
```

## 文件结构

```
templategen/
├── generator.go         # Generator 接口实现（入口）
├── params.go            # 参数定义（-template, -output）
├── collector.go         # 收集带注解的目标（struct/interface/method）
├── import_resolver.go   # Import 路径解析（四层优先级）
├── template_data.go     # TemplateData 数据结构
├── template_funcs.go    # Sprig + 自定义模板函数
├── stdlib_imports.go    # 标准库白名单
├── template_loader.go   # 模板加载（智能路径搜索 + 继承支持）
└── generator_test.go    # 测试
```

## 实现步骤

### 阶段 1：基础框架
1. 创建 `templategen/` 目录
2. 定义参数结构（处理 `-template`, `-output` 参数）
3. 实现 `TemplateGenerator` 基础结构，注册 `Define` 和 `Import` 注解
4. 在 `cmd/main.go` 注册生成器

### 阶段 2：目标收集
1. 识别文件级 `go:gogen: plugin:templategen` 指令
2. 收集文件中带 `@Define` 注解的 struct/interface/method
3. 解析 @Define 参数，区分字符串值和类型引用
4. 按 receiver 将方法分组到对应结构体

### 阶段 3：Import 解析器
1. 实现当前文件 import 提取（复用 `structparse.extractImports`）
2. 实现 @Import 注解解析
3. 实现全限定路径解析（带引号的 "path.Type"）
4. 实现标准库白名单
5. 整合四层优先级查找逻辑

### 阶段 4：模板引擎
1. 集成 Sprig 库
2. 实现自定义 Helper 函数
3. 实现模板加载（智能路径搜索）
4. 支持模板继承（define/template）
5. 实现 ImportManager 动态收集 import

### 阶段 5：代码生成
1. 构建 TemplateData
2. 执行模板渲染
3. 收集 imports 并生成 gg.Generator
4. 处理多模板输出

### 阶段 6：测试
1. 编写单元测试
2. 创建 `templategen/testdata/` 示例

## 关键文件清单

| 文件 | 用途 |
|------|------|
| `templategen/generator.go` | Generator 接口实现，处理 go:gogen 指令 |
| `templategen/params.go` | 解析 -template, -output 参数 |
| `templategen/collector.go` | 收集带注解目标，构建 TemplateData |
| `templategen/import_resolver.go` | 四层 import 解析逻辑 |
| `templategen/template_data.go` | TemplateData 结构定义 |
| `templategen/template_funcs.go` | Sprig + 自定义函数 |
| `templategen/stdlib_imports.go` | 标准库包名映射 |
| `templategen/template_loader.go` | 智能模板加载 + 继承 |
| `cmd/main.go` | 注册 templategen |

## 使用示例

### 示例 1：基础用法 - 生成结构体方法

**源文件 `service.go`:**
```go
//go:gogen: plugin:templategen -template ./templates/service.tmpl

package myservice

import "io"

// @Define(name=IO, reader=io.Reader)
type MyService struct {
    reader io.Reader
}
```

**模板 `templates/service.tmpl`:**
```gotemplate
{{- range .Structs }}
// {{ .Name }}Methods 为 {{ .Name }} 生成的方法
func (s *{{ .Name }}) GetReader() {{ (.Defines.IO.reader).FullType }} {
    return s.reader
}
{{- end }}
```

**输出 `service_gen.go`:**
```go
package myservice

import "io"

// MyServiceMethods 为 MyService 生成的方法
func (s *MyService) GetReader() io.Reader {
    return s.reader
}
```

### 示例 2：方法元数据

**源文件 `handler.go`:**
```go
//go:gogen: plugin:templategen -template ./templates/handler.tmpl -output $FILE_wrap.go

package api

import "context"

// @Define(name=Config, timeout="30s")
type Handler struct {}

// @Define(name=Meta, permission="admin", audit="true")
func (h *Handler) Delete(ctx context.Context, id string) error {
    return nil
}

// @Define(name=Meta, permission="user", audit="false")
func (h *Handler) Get(ctx context.Context, id string) (any, error) {
    return nil, nil
}
```

**模板 `templates/handler.tmpl`:**
```gotemplate
{{- range .Structs }}
// {{ .Name }} wrapper with permission checks
{{- range .Methods }}

func (h *{{ $.Name }}) {{ .Name }}WithAuth(ctx context.Context{{ range .Params }}, {{ .Name }} {{ .Type }}{{ end }}) {{ range .Returns }}{{ .Type }}{{ end }} {
    // Permission: {{ (.Defines.Meta.permission).StringVal }}
    // Audit: {{ (.Defines.Meta.audit).StringVal }}
    {{- if eq (.Defines.Meta.audit).StringVal "true" }}
    log.Printf("Audit: %s called {{ .Name }}", getUserID(ctx))
    {{- end }}
    return h.{{ .Name }}(ctx{{ range .Params }}, {{ .Name }}{{ end }})
}
{{- end }}
{{- end }}
```

### 示例 3：使用 @Import 处理长路径

**源文件 `k8s.go`:**
```go
//go:gogen: plugin:templategen -template ./templates/k8s.tmpl

package controller

import "context"

// @Import(alias=corev1, path="k8s.io/api/core/v1")
// @Define(name=Types, pod=corev1.Pod, ctx=context.Context)
type PodController struct {}

// @Define(name=Action, operation="create")
func (c *PodController) Create(ctx context.Context) error {
    return nil
}
```

### 示例 4：多模板生成

**源文件 `model.go`:**
```go
//go:gogen: plugin:templategen -template ./templates/repo.tmpl -output $FILE_repo.go
//go:gogen: plugin:templategen -template ./templates/service.tmpl -output $FILE_service.go

package domain

// @Define(name=DB, table="users")
type User struct {
    ID   int
    Name string
}
```

### 示例 5：模板继承

**基础模板 `templates/base.tmpl`:**
```gotemplate
{{- define "header" -}}
// Code generated by templategen. DO NOT EDIT.
{{- end -}}

{{- define "receiver" -}}
{{ . | lower | substr 0 1 }}
{{- end -}}
```

**扩展模板 `templates/crud.tmpl`:**
```gotemplate
{{- template "header" }}

package {{ .File.PackageName }}

{{- range .Structs }}
{{- $r := template "receiver" .Name }}

func ({{ $r }} *{{ .Name }}) Create() error {
    // table: {{ (.Defines.DB.table).StringVal }}
    return nil
}
{{- end }}
```

## 验证方式

1. **单元测试**
   - Import 解析器各层级测试
   - 模板渲染测试
   - TypeRef 解析测试

2. **集成测试**
   - 创建 `templategen/testdata/` 目录
   - 编写完整的 input/expected output 测试用例

3. **手动验证**
   ```bash
   # 构建
   go build -o gotoolkit ./cmd

   # 运行
   ./gotoolkit gen ./templategen/testdata/...

   # 检查输出
   cat templategen/testdata/*_gen.go
   ```

## 模板继承实现

### 加载机制

支持通过参数指定额外的模板文件用于继承：

```go
//go:gogen: plugin:templategen -template ./templates/crud.tmpl -include ./templates/base.tmpl
```

或自动加载同目录下的 `_*.tmpl` 文件作为基础模板：
- `templates/_base.tmpl` - 自动作为所有模板的基础
- `templates/crud.tmpl` - 主模板

### 模板加载器实现思路

```go
func loadTemplates(mainPath string, includePaths []string, searchDirs []string) (*template.Template, error) {
    // 1. 创建根模板，加载 Sprig 函数
    tmpl := template.New("").Funcs(sprig.FuncMap())

    // 2. 自动发现同目录下的 _*.tmpl 作为基础模板
    dir := filepath.Dir(mainPath)
    baseFiles, _ := filepath.Glob(filepath.Join(dir, "_*.tmpl"))

    // 3. 加载 -include 参数指定的文件
    allFiles := append(baseFiles, includePaths...)
    allFiles = append(allFiles, mainPath)

    // 4. 解析所有模板
    return tmpl.ParseFiles(allFiles...)
}
```

### 继承示例

**`templates/_helpers.tmpl`:**
```gotemplate
{{- define "pkg" -}}
package {{ .File.PackageName }}
{{- end -}}

{{- define "imports" -}}
import (
{{- range $path, $alias := .Imports.All }}
    {{ if $alias }}{{ $alias }} {{ end }}"{{ $path }}"
{{- end }}
)
{{- end -}}
```

**`templates/service.tmpl`:**
```gotemplate
{{ template "pkg" . }}

{{ template "imports" . }}

{{- range .Structs }}
// ...
{{- end }}
```

## 依赖

- `github.com/Masterminds/sprig/v3` - 模板函数库
- `github.com/donutnomad/gg` - 代码生成框架（已有）

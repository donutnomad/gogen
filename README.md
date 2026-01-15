# gogen

基于注解的 Go 代码生成工具集，通过扫描源文件中的注解自动生成相关代码。

## 安装

```bash
go install github.com/donutnomad/gogen@latest
```

## 快速开始

```bash
# 扫描当前目录及子目录，生成代码
gogen gen ./...

# 详细模式
gogen -v gen ./...
```

---

## pickgen - 结构体字段选择生成器

从现有结构体中选择或排除字段，生成新的结构体类型。

### 注解

| 注解 | 说明 |
|------|------|
| `@Pick` | 从源结构体中**选择**指定字段生成新结构体 |
| `@Omit` | 从源结构体中**排除**指定字段生成新结构体 |

### 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `name` | 是 | 生成的新结构体名称 |
| `fields` | 是 | 字段列表，格式: `[Field1,Field2,Field3]` |
| `source` | 否 | 源结构体，格式: `pkg.Type` 或完整路径 |

### 使用方式

#### 方式一：直接注解在结构体上

```go
// @Pick(name=UserBasic, fields=`[ID,Name,Email]`)
// @Omit(name=UserPublic, fields=`[Password,Salt]`)
type User struct {
    ID       uint64 `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"-"`
    Salt     string `json:"-"`
}
```

生成结果：

```go
// UserBasic 从 User Pick 生成
type UserBasic struct {
    ID    uint64 `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (t *UserBasic) From(src *User) {
    t.ID = src.ID
    t.Name = src.Name
    t.Email = src.Email
}

func NewUserBasic(src *User) UserBasic {
    var result UserBasic
    result.From(src)
    return result
}

// UserPublic 从 User Omit 生成
type UserPublic struct {
    ID    uint64 `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// ... From 和 New 方法
```

#### 方式二：独立注解 (`//go:gen:`) - 引用外部类型

在任意 `.go` 文件中使用独立注解，可以引用第三方包或其他包的类型：

```go
package mymodels

import "gorm.io/gorm"

// 引用第三方包
//go:gen: @Pick(name=GormBasic, source=`gorm.io/gorm.Model`, fields=`[ID,CreatedAt,UpdatedAt]`)

// 引用已导入的包（使用别名）
//go:gen: @Omit(name=GormWithoutDelete, source=`gorm.Model`, fields=`[DeletedAt]`)

// 引用本模块其他包
//go:gen: @Pick(name=UserID, source=`github.com/myapp/models.User`, fields=`[ID]`)
```

### source 参数格式

| 格式 | 示例 | 说明 |
|------|------|------|
| 完整包路径 | `gorm.io/gorm.Model` | 第三方包的完整导入路径 |
| 完整包路径 | `github.com/user/repo/pkg.Type` | 任意包的完整路径 |
| 已导入包 | `models.User` | 当前文件已导入的包 |
| 已导入包（别名） | `gormModel.Model` | 使用别名导入的包 |
| 当前包类型 | `LocalType` | 无包前缀，表示当前包内的类型 |

### 特殊包名处理

对于包含特殊字符的目录名，pickgen 会自动转换为有效的 Go 标识符：

```
目录名 special-pkg  → 包别名 specialpkg
目录名 v2-api       → 包别名 v2api
目录名 123pkg       → 包别名 _123pkg
```

---

## stateflowgen - 状态流转代码生成器

根据状态流转规则自动生成类型安全的状态机代码，支持审批流程、通配符展开等高级特性。

### 注解

| 注解 | 说明 |
|------|------|
| `@StateFlow` | 定义状态机配置 |
| `@Flow` | 定义单条状态流转规则 |

### @StateFlow 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `name` | 否 | 类型前缀，如 `Order` 生成 `OrderPhase`、`OrderState` 等 |
| `output` | 否 | 输出文件路径 |

### @Flow 语法

```
@Flow: 源状态 => [ 目标状态1, 目标状态2 ]
```

#### 状态格式

| 格式 | 说明 |
|------|------|
| `Phase` | 简单阶段 |
| `Phase(Status)` | 阶段 + 子状态 |
| `Phase(*)` | 通配符，匹配该阶段的所有子状态 |
| `(Status)` | 仅切换子状态，保持阶段不变 |
| `(=)` | 自我流转，保持当前状态 |

#### 审批标记

| 标记 | 说明 |
|------|------|
| `!` | 必须审批 - 无论参数如何，都进入审批流程 |
| `?` | 可选审批 - 根据 `withApproval` 参数决定 |

#### 中间态和回退

| 关键字 | 说明 |
|------|------|
| `via` | 指定审批中间状态 |
| `else` | 指定审批拒绝后的回退状态（默认回退到原状态） |

### 基础示例

```go
// @StateFlow(name="Order")
// @Flow: Created   => [ Paid ]
// @Flow: Paid      => [ Shipped ]
// @Flow: Shipped   => [ Delivered ]
// @Flow: Delivered => [ Completed ]
const _ = ""
```

生成流程图：
```
Created ──▶ Paid ──▶ Shipped ──▶ Delivered ──▶ Completed
```

生成代码包括：
- `OrderPhase` - 阶段枚举类型
- `OrderStage` - 阶段类型别名
- `OrderState` - 完整状态结构
- `OrderStateColumns` - 数据库存储结构
- `TransitionTo()` - 状态流转方法
- `ValidTransitions()` - 获取有效目标状态
- `Next()` - 获取下一个可能的状态

### 必须审批示例 (`!`)

```go
// @StateFlow(name="Document")
// @Flow: Draft     => [ Published! via Reviewing ]
// @Flow: Published => [ Archived ]
const _ = ""
```

- `Published!` 表示必须进入审批流程
- `via Reviewing` 指定审批中间状态为 `Reviewing`
- 审批通过调用 `Commit()` 进入 `Published`
- 审批拒绝调用 `Reject()` 回退到 `Draft`

### 可选审批示例 (`?`)

```go
// @StateFlow(name="Task")
// @Flow: Draft     => [ Submitted ]
// @Flow: Submitted => [ Approved? via Reviewing ]
// @Flow: Approved  => [ Done ]
const _ = ""
```

生成流程图：
```
                                              ┌── <COMMIT> ──▶ Approved ──▶ Done
                                              │
                       ┌──▶ Reviewing (via) ──┤
                       │                      │
                       │                      └── <REJECT> ──▶ Submitted 🔁
Draft ──▶ Submitted ──▶ <?APPROVAL?> ──┤
                       │
                       │
                       └──▶ Approved ──▶ Done
```

- `Approved?` 表示可选审批
- 调用 `TransitionTo(StageTaskApproved, true)` 进入审批流程
- 调用 `TransitionTo(StageTaskApproved, false)` 直接流转

### 自定义回退状态 (`else`)

```go
// @StateFlow(name="Release")
// @Flow: Development => [ Testing ]
// @Flow: Testing     => [ Production! via Deploying else Rollback ]
// @Flow: Rollback    => [ Development ]
// @Flow: Production  => [ Archived ]
const _ = ""
```

- `else Rollback` 指定审批拒绝后进入 `Rollback` 状态（而非回退到 `Testing`）

### 通配符示例 (`*`)

```go
// @StateFlow(name="Machine")
// @Flow: Init           => [ Ready(Running) ]
// @Flow: Ready(Running) => [ (Stopped) ]
// @Flow: Ready(Stopped) => [ (Running) ]
// @Flow: Ready(*)       => [ Terminated! via Terminating ]
const _ = ""
```

- `Ready(*)` 展开为 `Ready(Running)` 和 `Ready(Stopped)`
- 通配符展开时不包含自我流转

### 自我流转示例 (`=`)

```go
// @StateFlow(name="Connection")
// @Flow: Disconnected => [ Connected ]
// @Flow: Connected    => [ Connected? via Reconnecting ]
// @Flow: Connected    => [ Disconnected ]
const _ = ""
```

- 用于"刷新"或"重试"场景

### 复用中间态示例

```go
// @StateFlow(name="Article")
// @Flow: Draft     => [ Published! via Reviewing ]
// @Flow: Published => [ Updated! via Reviewing ]
// @Flow: Updated   => [ Archived! via Reviewing ]
// @Flow: Archived  => [ Deleted ]
const _ = ""
```

- 多个状态流转共享同一个 `via Reviewing` 中间态

### 生成的 API

```go
// 状态流转
state, err := state.TransitionTo(StageOrderPaid)

// 带审批参数的流转
state, err := state.TransitionTo(StageTaskApproved, true)

// 审批通过
state, err := state.Commit()

// 审批拒绝
state, err := state.Reject()

// 检查是否在审批中
if state.IsApprovalPending() { ... }

// 获取有效目标状态
targets := state.ValidTransitions()

// 获取下一个可能的状态
nextStates := state.Next()

// 数据库存储转换
columns := state.ToColumns()
state := columns.ToState()
```

### 错误类型

| 错误 | 说明 |
|------|------|
| `ErrInvalidTransition` | 无效的状态流转 |
| `ErrApprovalInProgress` | 已有审批在进行中 |
| `ErrNotInApproval` | 当前不在审批状态 |

---

## 其他生成器

### gormgen

为 GORM 模型生成类型安全的 Schema 和查询辅助代码。

```go
// @Gsql(prefix="xxx")
type User struct {
    ID   uint64 `gorm:"primaryKey"`
    Name string `gorm:"column:name"`
}
```

### settergen

生成 Patch/Setter 相关代码。

```go
// @Setter(patch="v2", patch_mapper="Type.Method")
type Config struct {
    Host string
    Port int
}
```

### slicegen

为结构体切片生成 Filter/Map/Sort 等辅助方法。

```go
// @Slice(methods=[filter,map,sort,groupby])
type User struct {
    ID   uint64
    Name string
}
```

---

## 配置

### 包级配置

在包目录下创建配置注释：

```go
//go:gogen output=$DIR/gen/$NAME_gen.go
```

### 输出路径变量

| 变量 | 说明 |
|------|------|
| `$FILE` | 源文件路径（不含扩展名） |
| `$DIR` | 源文件目录 |
| `$NAME` | 源文件名（不含扩展名） |

---

## License

MIT

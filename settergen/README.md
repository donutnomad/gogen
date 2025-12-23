# Settergen - Go Setter 代码生成器

Settergen 是一个基于注解的 Go 代码生成器，用于为结构体生成 Patch 更新相关的代码。它支持多种模式，可以根据不同的需求生成不同类型的代码。

## 注解说明

### @Setter

`@Setter` 注解用于标记需要生成 Setter 相关代码的结构体。

**支持的参数：**

- `patch`: 生成模式（默认值: `none`）
  - `"none"` 或留空: 不生成代码（跳过）
  - `"v2"`: 使用 automap 生成 `ToPatch` 方法
  - `"full"`: 生成 `ToMap` 方法，将所有字段转换为 map

- `patch_mapper`: Mapper 方法（默认值: 空字符串，仅 v2 模式使用）
  - 格式: `Type.Method`
  - 如果不指定，自动查找 `ToPO` 方法
  - 示例: `patch_mapper="Order.ToOrderPO"`

## 使用示例

### 0. 不生成代码（默认）

如果不指定 patch 参数或设置为 none，将不生成任何代码。

```go
// @Setter               # 等同于 @Setter(patch="none")，不生成代码
type User struct {
    ID   int64
    Name string
}

// @Setter(patch="none") # 明确指定不生成
type Product struct {
    ID   int64
    Name string
}
```

### 1. V2 模式 - ToPatch 方法

使用 automap 生成 `ToPatch` 方法，需要提供一个 mapper 方法。

```go
// @Setter(patch="v2")
type Article struct {
    ID      int64
    Title   string
    Content string
}

func (a *Article) ToPO() *ArticlePO {
    return &ArticlePO{
        ID:      a.ID,
        Title:   a.Title,
        Content: a.Content,
    }
}

type ArticlePO struct {
    ID      int64  `gorm:"column:id"`
    Title   string `gorm:"column:title"`
    Content string `gorm:"column:content"`
}
```

### 2. V2 模式 - 自定义 Mapper

指定自定义的 mapper 方法。

```go
// @Setter(patch="v2", patch_mapper="Order.ToOrderPO")
type Order struct {
    ID      int64
    OrderNo string
}

func (o *Order) ToOrderPO() *OrderPO {
    return &OrderPO{
        ID:      o.ID,
        OrderNo: o.OrderNo,
    }
}
```

### 3. Full 模式 - ToMap 方法

生成 `ToMap` 方法，将所有字段转换为 map。

```go
// @Setter(patch="full")
type Product struct {
    ID    int64
    Name  string
    Price float64
}
```

生成的代码：

```go
func (p *Product) ToMap() map[string]any {
    values := make(map[string]any, 3)
    values["id"] = p.ID
    values["name"] = p.Name
    values["price"] = p.Price
    return values
}
```

### 3. V2 模式 - ToPatch 方法

使用 automap 生成 `ToPatch` 方法，需要提供一个 mapper 方法。

```go
// @Setter(patch="v2")
type Article struct {
    ID      int64
    Title   string
    Content string
}

func (a *Article) ToPO() *ArticlePO {
    return &ArticlePO{
        ID:      a.ID,
        Title:   a.Title,
        Content: a.Content,
    }
}

type ArticlePO struct {
    ID      int64  `gorm:"column:id"`
    Title   string `gorm:"column:title"`
    Content string `gorm:"column:content"`
}
```

### 4. V2 模式 - 自定义 Mapper

指定自定义的 mapper 方法。

```go
// @Setter(patch="v2", patch_mapper="Order.ToOrderPO")
type Order struct {
    ID      int64
    OrderNo string
}

func (o *Order) ToOrderPO() *OrderPO {
    return &OrderPO{
        ID:      o.ID,
        OrderNo: o.OrderNo,
    }
}
```

### 5. V2/Full 模式 - 混合模式

同时生成 v2 和 full 模式的代码。

```go
// @Setter(patch="v2/full", patch_mapper="Category.ToPO")
type Category struct {
    ID   int64
    Name string
}

func (c *Category) ToPO() *CategoryPO {
    return &CategoryPO{
        ID:   c.ID,
        Name: c.Name,
    }
}
```

## 运行

```bash
# 在包含 @Setter 注解的包目录下运行
gotoolkit gen ./...
```

## 与 Gormgen 的区别

Settergen 专注于 Patch/Setter 相关的代码生成，从 Gormgen 中提取了 Patch 相关功能：

- **Gormgen**: 专注于 GORM Schema 和查询代码生成（`@Gsql` 注解）
- **Settergen**: 专注于 Patch/Setter 代码生成（`@Setter` 注解）

## 参数说明

### patch 参数

- `"none"`（默认）: 不生成任何代码
- `"v2"`: 使用 automap 生成 `ToPatch()` 方法，需要配合 `patch_mapper` 参数
- `"full"`: 生成 `ToMap()` 方法，将结构体所有字段转换为 `map[string]any`

**注意**: 默认值为 `"none"`，表示不生成代码。如果使用 `@Setter` 注解但不指定 patch 参数，将不会生成任何代码。

### patch_mapper 参数

仅在 `patch="v2"` 时使用。

格式: `Type.Method`

例如: `patch_mapper="User.ToPO"`

如果不指定，会自动查找名为 `ToPO` 的方法。

## 依赖

- `github.com/donutnomad/gogen/automap`: 自动映射（v2 模式）

## 注意事项

1. v2 模式需要提供一个 mapper 方法，该方法应该返回一个带 `gorm` tag 的 PO 结构体
2. 生成的代码文件默认为 `setter_gen.go`，可以通过 `output` 参数自定义
3. 生成的代码包含 `DO NOT EDIT` 标记，不应手动修改

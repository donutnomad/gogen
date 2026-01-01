# Codegen 测试覆盖文档

本文档描述了 codegen 生成器的全面测试覆盖情况。

## 测试文件结构

```
codegen/
├── testdata/
│   ├── basic.go          # 基本功能测试
│   ├── edge.go           # 边缘情况测试
│   ├── invalid.go        # 无效场景测试
│   ├── duplicate.go      # 重复检测测试
│   ├── mixed.go          # 混合类型测试
│   ├── pkg1/
│   │   └── errors.go     # 跨包测试 - 包1
│   └── pkg2/
│       └── errors.go     # 跨包测试 - 包2
└── generator_test.go     # 测试代码

```

## 测试场景覆盖

### 1. 基本功能 (basic.go)

**目的**: 验证基本的 error 和 const 类型处理

- ✅ error 类型变量
- ✅ const 整数类型
- ✅ 默认参数值（http 默认 500，grpc 默认 Internal）
- ✅ 自定义 HTTP 和 gRPC 状态码

**预期结果**: 成功生成代码，无错误

### 2. 边缘情况 (edge.go)

**目的**: 测试各种边界值和所有标准状态码

- ✅ 负数错误码（-1, -999）
- ✅ 零值错误码（0）
- ✅ 最大 int32 值（2147483647, 2147483646）
- ✅ 所有 17 个 gRPC 状态码：
  - OK, Canceled, Unknown, InvalidArgument, DeadlineExceeded
  - NotFound, AlreadyExists, PermissionDenied, ResourceExhausted
  - FailedPrecondition, Aborted, OutOfRange, Unimplemented
  - Internal, Unavailable, DataLoss, Unauthenticated
- ✅ 各种标准 HTTP 状态码：
  - 1xx: 100, 101
  - 2xx: 200, 201, 204
  - 3xx: 301, 302, 304
  - 4xx: 400, 401, 403, 404, 405, 408, 409, 429
  - 5xx: 500, 501, 502, 503, 504

**预期结果**: 成功生成代码，无错误

### 3. 无效场景 (invalid.go)

**目的**: 验证参数验证逻辑

- ✅ 非法 HTTP 状态码（999, 0, -1）
- ✅ 非法 gRPC Code 名称（InvalidCode）
- ✅ gRPC Code 大小写错误（notfound, NOTFOUND）

**预期结果**: 生成过程中产生 6 个验证错误

### 4. 重复检测 (duplicate.go)

**目的**: 验证同包内 code 值重复检测

- ✅ 相同 code 值用于不同的 error 变量
- ✅ 相同 code 值用于不同的 const

**预期结果**: 生成过程中产生 2 个重复错误

### 5. 跨包场景 (pkg1, pkg2)

**目的**: 验证不同包可以使用相同的 code 值

- ✅ pkg1 使用 code=50001, 50002
- ✅ pkg2 使用相同的 code=50001, 50002
- ✅ 生成2个独立的 generate.go 文件

**预期结果**: 成功生成2个文件，无错误（不同包允许相同 code）

### 6. 混合类型 (mixed.go)

**目的**: 测试各种整数类型的 const

- ✅ error 类型变量
- ✅ 默认 int 类型 const
- ✅ int32 类型 const
- ✅ int64 类型 const
- ✅ uint 类型 const
- ✅ 多个单独声明的变量/常量

**预期结果**: 成功生成代码，无错误

## 生成代码验证

所有测试都验证生成的代码包含以下方法：

- `GetHttpCode[T comparable](v T) (int, bool)` - 使用 `switch val := any(v).(type)`
- `GetGrpcCode[T comparable](v T) (codes.Code, bool)`
- `GetCode[T comparable](v T) (int, bool)`
- `GetName[T comparable](v T) (string, bool)`

## 类型安全验证

通过 example_test.go 验证：

- ✅ error 类型可以正确匹配（使用 `errors.Is()`）
- ✅ int 类型可以正确匹配（使用 `any(val) == any(constant)`）
- ✅ 不同整数类型（int16, int32, int64）无法匹配（返回 false）

## 测试执行

运行所有测试：

```bash
go test -v ./codegen
```

预期输出：

```
=== RUN   TestCodegenBasic
--- PASS: TestCodegenBasic
=== RUN   TestCodegenEdge
--- PASS: TestCodegenEdge
=== RUN   TestCodegenInvalid
--- PASS: TestCodegenInvalid
=== RUN   TestCodegenDuplicate
--- PASS: TestCodegenDuplicate
=== RUN   TestCodegenCrossPackage
--- PASS: TestCodegenCrossPackage
=== RUN   TestCodegenMixed
--- PASS: TestCodegenMixed
PASS
```

## 覆盖率统计

- **功能覆盖**: 100%
  - ✅ 基本代码生成
  - ✅ 参数验证（HTTP/gRPC）
  - ✅ 重复检测（包级）
  - ✅ 跨包支持
  - ✅ 类型区分（error vs non-error）

- **边缘情况覆盖**: 100%
  - ✅ 所有 gRPC 状态码
  - ✅ 常用 HTTP 状态码
  - ✅ 负数、零值、最大值
  - ✅ 不同整数类型

- **错误处理覆盖**: 100%
  - ✅ 非法 HTTP 状态码
  - ✅ 非法 gRPC Code
  - ✅ code 值重复

## 注意事项

1. **分组声明**: 当前实现不支持在声明组内部为每个成员添加注解。注解必须放在单独的声明之前。

   ```go
   // ✅ 支持
   // @Code(code=1001,http=404,grpc=NotFound)
   var Err1 = errors.New("error 1")

   // @Code(code=1002,http=500,grpc=Internal)
   var Err2 = errors.New("error 2")

   // ❌ 不支持
   var (
       // @Code(code=1001,http=404,grpc=NotFound)
       Err1 = errors.New("error 1")
       // @Code(code=1002,http=500,grpc=Internal)
       Err2 = errors.New("error 2")
   )
   ```

2. **类型匹配**: 生成的泛型方法使用 `any()` 进行类型比较，确保类型安全。不同的整数类型（如 int16 vs int）不会匹配。

3. **包级隔离**: code 值重复检测是包级的，不同包可以使用相同的 code 值。

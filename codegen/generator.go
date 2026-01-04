package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/plugin"
)

const generatorName = "codegen"

// CodeParams 定义 Code 注解支持的参数
type CodeParams struct {
	Code int    `param:"name=code,required=true,default=0,description=业务错误码"`
	HTTP string `param:"name=http,required=false,default=500,description=HTTP 状态码"`
	GRPC string `param:"name=grpc,required=false,default=Internal,description=gRPC Code 名称"`
}

// CodeGenerator 实现 plugin.Generator 接口
type CodeGenerator struct {
	plugin.BaseGenerator
}

func NewCodeGenerator() *CodeGenerator {
	gen := &CodeGenerator{
		BaseGenerator: *plugin.NewBaseGeneratorWithParamsStruct(
			generatorName,
			[]string{"Code"},
			[]plugin.TargetKind{plugin.TargetVar, plugin.TargetConst},
			CodeParams{},
		),
	}
	gen.SetPriority(40)
	return gen
}

// Generate 执行代码生成
func (g *CodeGenerator) Generate(ctx *plugin.GenerateContext) (*plugin.GenerateResult, error) {
	result := plugin.NewGenerateResult()

	if len(ctx.Targets) == 0 {
		return result, nil
	}

	// 按输出路径分组
	fileTargets := make(map[string][]*codeInfo)

	// 用于包级重复检测
	pkgCodeValues := make(map[string]map[int]string) // key: 包名, value: code值到变量名的映射

	for _, at := range ctx.Targets {
		ann := plugin.GetAnnotation(at.Annotations, "Code")
		if ann == nil {
			continue
		}

		// 获取解析好的参数
		var params CodeParams
		if at.ParsedParams != nil {
			var ok bool
			params, ok = at.ParsedParams.(CodeParams)
			if !ok {
				result.AddError(fmt.Errorf("ParsedParams 类型断言失败: %T", at.ParsedParams))
				continue
			}
		}

		// 验证参数
		if err := validateParams(&params); err != nil {
			result.AddError(fmt.Errorf("验证参数失败 %s: %w", at.Target.Name, err))
			continue
		}

		// 检测包内 code 重复
		pkgKey := at.Target.PackageName
		if pkgCodeValues[pkgKey] == nil {
			pkgCodeValues[pkgKey] = make(map[int]string)
		}
		if existingName, exists := pkgCodeValues[pkgKey][params.Code]; exists {
			result.AddError(fmt.Errorf("包 %s 中错误码重复: %s 和 %s 都使用了 code=%d",
				pkgKey, existingName, at.Target.Name, params.Code))
			continue
		}
		pkgCodeValues[pkgKey][params.Code] = at.Target.Name

		// 计算输出路径
		fileConfig := ctx.GetFileConfig(at.Target.FilePath)
		outputPath := plugin.GetOutputPath(at.Target, ann, "generate.go", fileConfig, g.Name(), ctx.DefaultOutput)

		fileTargets[outputPath] = append(fileTargets[outputPath], &codeInfo{
			name:       at.Target.Name,
			code:       params.Code,
			httpStatus: params.HTTP,
			grpcCode:   params.GRPC,
			pkgName:    at.Target.PackageName,
			kind:       at.Target.Kind,
		})

		if ctx.Verbose {
			fmt.Printf("[codegen] 处理 %s %s (code=%d, http=%s, grpc=%s) -> %s\n",
				at.Target.Kind, at.Target.Name, params.Code, params.HTTP, params.GRPC, outputPath)
		}
	}

	// 为每个输出文件生成代码
	for outputPath, codes := range fileTargets {
		gen, err := g.generateDefinition(codes)
		if err != nil {
			result.AddError(fmt.Errorf("生成 %s 失败: %w", outputPath, err))
			continue
		}
		result.AddDefinition(outputPath, gen)

		if ctx.Verbose {
			fmt.Printf("[codegen] 生成定义 %s\n", outputPath)
		}
	}

	return result, nil
}

type codeInfo struct {
	name       string
	code       int
	httpStatus string
	grpcCode   string
	pkgName    string
	kind       plugin.TargetKind
}

// generateDefinition 生成 gg 定义
func (g *CodeGenerator) generateDefinition(codes []*codeInfo) (*gg.Generator, error) {
	if len(codes) == 0 {
		return nil, fmt.Errorf("没有代码需要生成")
	}

	gen := gg.New()
	gen.SetPackage(codes[0].pkgName)

	// 添加必要的包导入
	gen.P("errors")
	grpcPkg := gen.P("google.golang.org/grpc/codes")

	body := gen.Body()

	// 生成 GetHttpCode 方法
	body.AddLine()
	body.AddString("// GetHttpCode returns the HTTP status code for the given value.")
	body.AddString("// The bool return indicates whether the value was found, not whether http is defined.")
	body.AddString("// Returns 0 if the value is not registered.")
	body.AddString("func GetHttpCode[T any](v T) (int, bool) {")
	body.AddString("\t_, httpCode, _, _, ok := _codegen_getInfo(v)")
	body.AddString("\treturn httpCode, ok")
	body.AddString("}")

	// 生成 GetGrpcCode 方法
	body.AddLine()
	body.AddString("// GetGrpcCode returns the gRPC status code for the given value.")
	body.AddString("// The bool return indicates whether the value was found, not whether grpc is defined.")
	body.AddString("// Returns codes.Unknown if the value is not registered.")
	body.AddString(fmt.Sprintf("func GetGrpcCode[T any](v T) (%s, bool) {", grpcPkg.Type("Code")))
	body.AddString("\t_, _, grpcCode, _, ok := _codegen_getInfo(v)")
	body.AddString("\treturn grpcCode, ok")
	body.AddString("}")

	// 生成 GetCode 方法
	body.AddLine()
	body.AddString("// GetCode returns the business error code for the given value.")
	body.AddString("// The bool return indicates whether the value was found.")
	body.AddString("// Returns 0 if the value is not registered.")
	body.AddString("func GetCode[T any](v T) (int, bool) {")
	body.AddString("\tcode, _, _, _, ok := _codegen_getInfo(v)")
	body.AddString("\treturn code, ok")
	body.AddString("}")

	// 生成 GetName 方法
	body.AddLine()
	body.AddString("// GetName returns the variable name for the given value.")
	body.AddString("// The bool return indicates whether the value was found.")
	body.AddString("// Returns empty string if the value is not registered.")
	body.AddString("func GetName[T any](v T) (string, bool) {")
	body.AddString("\t_, _, _, name, ok := _codegen_getInfo(v)")
	body.AddString("\treturn name, ok")
	body.AddString("}")

	// 生成 AllCodedValues 方法
	body.AddLine()
	body.AddString("// AllCodedValues returns all registered code values.")
	body.AddString("// Each element in the slice is the original value that was annotated with @Code.")
	body.AddString("func AllCodedValues() []any {")
	body.AddString("\treturn []any{")
	for _, c := range codes {
		body.AddString(fmt.Sprintf("\t\t%s,", c.name))
	}
	body.AddString("\t}")
	body.AddString("}")

	// 生成内部辅助方法 _codegen_getInfo
	body.AddLine()
	body.AddString(fmt.Sprintf("func _codegen_getInfo[T any](v T) (code int, httpCode int, grpcCode %s, name string, ok bool) {", grpcPkg.Type("Code")))
	body.AddString("\tval := any(v)")
	for _, c := range codes {
		httpStatus := c.httpStatus
		if httpStatus == "" {
			httpStatus = "500"
		}
		grpcCode := c.grpcCode
		if grpcCode == "" {
			grpcCode = "Internal"
		}
		body.AddString(fmt.Sprintf("\tif _codegen_equal(val, %s) {", c.name))
		body.AddString(fmt.Sprintf("\t\treturn %d, %s, %s, %q, true", c.code, httpStatus, grpcPkg.Dot(grpcCode), c.name))
		body.AddString("\t}")
	}
	body.AddString(fmt.Sprintf("\treturn 0, 0, %s, \"\", false", grpcPkg.Dot("Unknown")))
	body.AddString("}")

	// 生成 _codegen_asInt64 辅助方法
	body.AddLine()
	body.AddString("func _codegen_asInt64(v any) (i int64, u uint64, signed bool, ok bool) {")
	body.AddString("\tswitch x := v.(type) {")
	body.AddString("\tcase int:")
	body.AddString("\t\treturn int64(x), 0, true, true")
	body.AddString("\tcase int8:")
	body.AddString("\t\treturn int64(x), 0, true, true")
	body.AddString("\tcase int16:")
	body.AddString("\t\treturn int64(x), 0, true, true")
	body.AddString("\tcase int32:")
	body.AddString("\t\treturn int64(x), 0, true, true")
	body.AddString("\tcase int64:")
	body.AddString("\t\treturn x, 0, true, true")
	body.AddString("\tcase uint:")
	body.AddString("\t\treturn 0, uint64(x), false, true")
	body.AddString("\tcase uint8:")
	body.AddString("\t\treturn 0, uint64(x), false, true")
	body.AddString("\tcase uint16:")
	body.AddString("\t\treturn 0, uint64(x), false, true")
	body.AddString("\tcase uint32:")
	body.AddString("\t\treturn 0, uint64(x), false, true")
	body.AddString("\tcase uint64:")
	body.AddString("\t\treturn 0, x, false, true")
	body.AddString("\t}")
	body.AddString("\treturn 0, 0, false, false")
	body.AddString("}")

	// 生成 _codegen_equalInt 辅助方法
	body.AddLine()
	body.AddString("func _codegen_equalInt(a, b any) bool {")
	body.AddString("\tai, au, as, aok := _codegen_asInt64(a)")
	body.AddString("\tbi, bu, bs, bok := _codegen_asInt64(b)")
	body.AddString("\tif !aok || !bok {")
	body.AddString("\t\treturn false")
	body.AddString("\t}")
	body.AddString("\tif as && bs {")
	body.AddString("\t\treturn ai == bi")
	body.AddString("\t}")
	body.AddString("\tif !as && !bs {")
	body.AddString("\t\treturn au == bu")
	body.AddString("\t}")
	body.AddString("\tif as {")
	body.AddString("\t\treturn ai >= 0 && uint64(ai) == bu")
	body.AddString("\t}")
	body.AddString("\treturn bi >= 0 && au == uint64(bi)")
	body.AddString("}")

	// 生成 _codegen_equal 辅助方法（处理 error、字符串和整数）
	body.AddLine()
	body.AddString("func _codegen_equal(a, b any) bool {")
	body.AddString("\tif a == nil || b == nil {")
	body.AddString("\t\treturn a == b")
	body.AddString("\t}")
	body.AddString("\tif ea, ok := a.(error); ok {")
	body.AddString("\t\teb, ok := b.(error)")
	body.AddString("\t\treturn ok && errors.Is(ea, eb)")
	body.AddString("\t}")
	body.AddString("\tif sa, ok := a.(string); ok {")
	body.AddString("\t\tsb, ok := b.(string)")
	body.AddString("\t\treturn ok && sa == sb")
	body.AddString("\t}")

	body.AddString("\tif _codegen_equalInt(a, b) {")
	body.AddString("\t\treturn false")
	body.AddString("\t}")

	body.AddString("\tva := reflect.ValueOf(a)")
	body.AddString("\tvb := reflect.ValueOf(b)")
	body.AddString("\tif va.Type() != vb.Type() {")
	body.AddString("\t\treturn false")
	body.AddString("\t}")
	body.AddString("\tif va.Kind() == reflect.Func {")
	body.AddString("\t\tefaceA := *(*[2]unsafe.Pointer)(unsafe.Pointer(&a))")
	body.AddString("\t\tefaceB := *(*[2]unsafe.Pointer)(unsafe.Pointer(&b))")
	body.AddString("\t\treturn efaceA[1] == efaceB[1]")
	body.AddString("\t}")

	body.AddString("\treturn false")
	body.AddString("}")

	return gen, nil
}

// validateParams 验证参数
func validateParams(params *CodeParams) error {
	// 验证 HTTP 状态码
	if params.HTTP != "" {
		status, err := strconv.Atoi(params.HTTP)
		if err != nil {
			return fmt.Errorf("无效的 HTTP 状态码: %s", params.HTTP)
		}
		if !isValidHTTPStatus(status) {
			return fmt.Errorf("非标准 HTTP 状态码: %d", status)
		}
	}

	// 验证 gRPC Code
	if params.GRPC != "" {
		if !isValidGRPCCode(params.GRPC) {
			return fmt.Errorf("无效的 gRPC Code: %s (必须是: %s)", params.GRPC, strings.Join(validCodes, ", "))
		}
	}

	return nil
}

// isValidHTTPStatus 验证是否为标准 HTTP 状态码
func isValidHTTPStatus(status int) bool {
	validStatuses := []int{
		// 1xx Informational
		100, 101, 102, 103,
		// 2xx Success
		200, 201, 202, 203, 204, 205, 206, 207, 208, 226,
		// 3xx Redirection
		300, 301, 302, 303, 304, 305, 307, 308,
		// 4xx Client Error
		400, 401, 402, 403, 404, 405, 406, 407, 408, 409,
		410, 411, 412, 413, 414, 415, 416, 417, 418, 421,
		422, 423, 424, 425, 426, 428, 429, 431, 451,
		// 5xx Server Error
		500, 501, 502, 503, 504, 505, 506, 507, 508, 510, 511,
	}

	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

var validCodes = []string{
	"OK", "Canceled", "Unknown", "InvalidArgument",
	"DeadlineExceeded", "NotFound", "AlreadyExists",
	"PermissionDenied", "ResourceExhausted", "FailedPrecondition",
	"Aborted", "OutOfRange", "Unimplemented", "Internal",
	"Unavailable", "DataLoss", "Unauthenticated",
}

// isValidGRPCCode 验证是否为有效的 gRPC Code（严格匹配）
func isValidGRPCCode(code string) bool {
	for _, valid := range validCodes {
		if code == valid {
			return true
		}
	}
	return false
}

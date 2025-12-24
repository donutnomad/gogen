package swaggen

import (
	"fmt"
	"go/token"

	"github.com/donutnomad/gogen/internal/xast"
	parsers "github.com/donutnomad/gogen/swaggen/parser"
)

// ============================================================================
// 常量定义
// ============================================================================

const (
	ParamSourcePath = "path"
)

const (
	DefaultOutputFile = "swagger_generated.go"
	DefaultPackage    = ""
	GinContextType    = "*gin.Context"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

var logLevelPriority = map[string]int{
	LogLevelDebug: 0,
	LogLevelInfo:  1,
	LogLevelWarn:  2,
	LogLevelError: 3,
}

// version 控制生成代码的版本格式
// 1: 方法返回单一值 (result)
// 2: 方法返回值和错误 (result, err)
var version = func() *int { v := 2; return &v }()

// ============================================================================
// 核心类型定义
// ============================================================================

// TypeInfo 表示类型信息
type TypeInfo struct {
	FullName    string     // types2.BaseResponse[string]
	Package     string     // go.com/pkg/v2/types
	Alias       string     // types2
	TypeName    string     // BaseResponse
	GenericArgs []TypeInfo // [string]
	IsGeneric   bool       // 是否是泛型
	IsSlice     bool       // 是否是切片
	IsPointer   bool       // 是否是指针
}

// Parameter 表示方法参数
type Parameter struct {
	Name     string   // 参数名
	PathName string   // 路径中的参数名
	Alias    string   // 别名
	Type     TypeInfo // 参数类型
	Source   string   // path,header,query
	Required bool     // 是否必需
	Comment  string   // 参数注释
}

// SwaggerMethod 表示 Swagger 方法
type SwaggerMethod struct {
	Name         string      // 方法名
	Parameters   []Parameter // 参数列表
	ResponseType TypeInfo    // 返回类型
	Summary      string      // 摘要
	Description  string      // 描述
	Def          DefSlice
}

func (s SwaggerMethod) GetPaths() []string {
	var ret []string
	for _, item := range s.Def {
		switch v := item.(type) {
		case *parsers.GET:
			ret = append(ret, v.Value)
		case *parsers.POST:
			ret = append(ret, v.Value)
		case *parsers.PUT:
			ret = append(ret, v.Value)
		case *parsers.DELETE:
			ret = append(ret, v.Value)
		case *parsers.PATCH:
			ret = append(ret, v.Value)
		}
	}
	return ret
}

func (s SwaggerMethod) GetHTTPMethod() string {
	for _, item := range s.Def {
		switch v := item.(type) {
		case *parsers.GET:
			return v.Name()
		case *parsers.POST:
			return v.Name()
		case *parsers.PUT:
			return v.Name()
		case *parsers.DELETE:
			return v.Name()
		case *parsers.PATCH:
			return v.Name()
		}
	}
	return "GET"
}

// DefSlice 定义切片
type DefSlice []parsers.Definition

func (s DefSlice) GetPrefix() string {
	for _, item := range s {
		if v, ok := item.(*parsers.Prefix); ok {
			return v.Value
		}
	}
	return ""
}

func (s DefSlice) IsRemoved() bool {
	return FindDef[*parsers.Removed](s)
}

func (s DefSlice) IsExcludeFromBindAll() bool {
	return FindDef[*parsers.ExcludeFromBindAll](s)
}

func (s DefSlice) GetAcceptType() (string, bool) {
	for _, item := range s {
		switch v := item.(type) {
		case *parsers.FormReq:
			return "x-www-form-urlencoded", true
		case *parsers.JsonReq:
			return "json", true
		case *parsers.MimeReq:
			return v.Value, true
		}
	}
	return "json", false
}

func (s DefSlice) GetContentType() (string, bool) {
	for _, item := range s {
		switch v := item.(type) {
		case *parsers.JSON:
			return "json", true
		case *parsers.MIME:
			return v.Value, true
		}
	}
	return "json", false
}

func CollectDef[T any](inputs ...DefSlice) []T {
	var ret []T
	for _, input := range inputs {
		for _, item := range input {
			if v, ok := item.(T); ok {
				ret = append(ret, v)
			}
		}
	}
	return ret
}

func FindDef[T any](inputs ...DefSlice) bool {
	for _, input := range inputs {
		for _, item := range input {
			if _, ok := item.(T); ok {
				return true
			}
		}
	}
	return false
}

// CommonAnnotation 表示可应用于接口中所有方法的通用注释
type CommonAnnotation struct {
	Value   string   // 注释的值
	Exclude []string // 要从此注释中排除的方法名列表
}

// SwaggerInterface 表示 Swagger 接口
type SwaggerInterface struct {
	Name        string               // 接口名
	Methods     []SwaggerMethod      // 方法列表
	PackagePath string               // 包路径
	Comments    []string             // 接口注释
	Imports     xast.ImportInfoSlice // 导入信息
	CommonDef   DefSlice
}

func (w SwaggerInterface) GetWrapperName() string {
	n := fmt.Sprintf("%sWrap", w.Name)
	if n[0] == 'I' {
		n = n[1:]
	}
	return n
}

// InterfaceCollection 表示接口集合
type InterfaceCollection struct {
	Interfaces []SwaggerInterface // 接口列表
}

// ============================================================================
// 解析器和生成器类型
// ============================================================================

// AnnotationParser 注释解析器
type AnnotationParser struct {
	fileSet    *token.FileSet
	tagsParser *parsers.Parser
}

// ReturnTypeParser 返回类型解析器
type ReturnTypeParser struct {
	imports xast.ImportInfoSlice
}

// SwaggerGenerator Swagger 生成器
type SwaggerGenerator struct {
	collection *InterfaceCollection
	tagsParser *parsers.Parser
}

// GinGenerator Gin 绑定代码生成器
type GinGenerator struct {
	collection *InterfaceCollection
}

func NewParseError(message, detail string, original error) error {
	msg := ""
	if detail != "" {
		msg = fmt.Sprintf("[%s] %s: %s", "parse", message, detail)
	}
	msg = fmt.Sprintf("[%s] %s", "parse", message)
	return fmt.Errorf("%s %w", msg, original)
}

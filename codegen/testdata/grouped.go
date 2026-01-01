package grouped

import "errors"

// 测试分组 var 声明中每个变量的注解

var (
	// ErrNotFound 资源未找到错误
	// @Code(code=100012,http=404,grpc=NotFound)
	ErrNotFound = errors.New("not found")
	// ErrBadRequest 错误的请求参数
	// @Code(code=100013,http=400,grpc=InvalidArgument)
	ErrBadRequest = errors.New("bad request")
	// ErrOperateNotAllowed 操作不被允许
	// @Code(code=100014,http=403,grpc=PermissionDenied)
	ErrOperateNotAllowed = errors.New("operate not allowed")
	// ErrorForbidden 禁止访问
	// @Code(code=100015,http=403,grpc=PermissionDenied)
	ErrorForbidden = errors.New("forbidden")
	// ErrServer 服务器内部错误
	// @Code(code=100016,http=500,grpc=Internal)
	ErrServer = errors.New("internal server error")
	// RecordNotFound 记录未找到（没有注解，不应该被处理）
	RecordNotFound = errors.New("record not found")
)

// 测试分组 const 声明
const (
	// CodeSuccess 成功
	// @Code(code=200001,http=200,grpc=OK)
	CodeSuccess = 200001
	// CodeCreated 创建成功
	// @Code(code=200002,http=201,grpc=OK)
	CodeCreated = 200002
	// CodeNoContent 没有内容（没有注解，不应该被处理）
	CodeNoContent = 200003
)

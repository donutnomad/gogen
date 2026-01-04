package reuse

import "errors"

// 测试 reuse=true 功能：允许重复使用相同的 code 值

// @Code(code=70001,http=404,grpc=NotFound)
var ErrNotFound = errors.New("not found")

// @Code(code=70001,http=404,grpc=NotFound,reuse=true)
var ErrUserNotFound = errors.New("user not found")

// @Code(code=70001,http=404,grpc=NotFound,reuse=true)
var ErrResourceNotFound = errors.New("resource not found")

// @Code(code=70002,http=500,grpc=Internal)
const CodeInternal = 70002

// @Code(code=70002,http=500,grpc=Internal,reuse=true)
const CodeServerError = 70002

// 测试不同的 http/grpc 组合，但 code 相同
// @Code(code=70003,http=400,grpc=InvalidArgument)
var ErrBadRequest = errors.New("bad request")

// @Code(code=70003,http=400,grpc=InvalidArgument,reuse=true)
var ErrInvalidInput = errors.New("invalid input")

package basic

import "errors"

// 基本场景：error 类型

// @Code(code=1001,http=404,grpc=NotFound)
var ErrNotFound = errors.New("not found")

// @Code(code=1002,http=500,grpc=Internal)
var ErrInternal = errors.New("internal error")

// @Code(code=1003,http=400,grpc=InvalidArgument)
var ErrBadRequest = errors.New("bad request")

// 基本场景：const 类型

// @Code(code=2001,http=200,grpc=OK)
const CodeSuccess = 2001

// @Code(code=2002,http=201,grpc=OK)
const CodeCreated = 2002

// 默认值测试

// @Code(code=3001)
var ErrDefault = errors.New("default values")

// @Code(code=3002)
const CodeDefault = 3002

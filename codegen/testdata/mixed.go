package mixed

import "errors"

// 混合场景：error 和各种整数类型的 const

// @Code(code=60001,http=404,grpc=NotFound)
var ErrMixed1 = errors.New("mixed error 1")

// @Code(code=60002,http=500,grpc=Internal)
var ErrMixed2 = errors.New("mixed error 2")

// @Code(code=60003,http=200,grpc=OK)
const CodeInt = 60003

// @Code(code=60004,http=201,grpc=OK)
const CodeInt32 int32 = 60004

// @Code(code=60005,http=400,grpc=InvalidArgument)
const CodeInt64 int64 = 60005

// @Code(code=60006,http=403,grpc=PermissionDenied)
const CodeUint uint = 60006

// 多个 var 在单独声明

// @Code(code=60007,http=404,grpc=NotFound)
var ErrGroup1 = errors.New("group error 1")

// @Code(code=60008,http=500,grpc=Internal)
var ErrGroup2 = errors.New("group error 2")

// 多个 const 在单独声明

// @Code(code=60009,http=200,grpc=OK)
const CodeGroup1 = 60009

// @Code(code=60010,http=201,grpc=OK)
const CodeGroup2 = 60010

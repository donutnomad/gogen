package invalid

import "errors"

// 无效场景：非法 HTTP 状态码（应该在代码生成时报错）

// @Code(code=30001,http=999,grpc=OK)
var ErrInvalidHTTP = errors.New("invalid http code")

// @Code(code=30002,http=0,grpc=OK)
var ErrZeroHTTP = errors.New("zero http code")

// @Code(code=30003,http=-1,grpc=OK)
var ErrNegativeHTTP = errors.New("negative http code")

// 无效场景：非法 gRPC 码（应该在代码生成时报错）

// @Code(code=30004,http=500,grpc=InvalidCode)
var ErrInvalidGRPC = errors.New("invalid grpc code")

// @Code(code=30005,http=500,grpc=notfound)
var ErrWrongCaseGRPC = errors.New("wrong case grpc code")

// @Code(code=30006,http=500,grpc=NOTFOUND)
var ErrUpperCaseGRPC = errors.New("upper case grpc code")

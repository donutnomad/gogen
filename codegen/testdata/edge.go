package edge

import "errors"

// 边缘情况：负数错误码

// @Code(code=-1,http=400,grpc=InvalidArgument)
var ErrNegativeCode = errors.New("negative code")

// @Code(code=-999,http=500,grpc=Internal)
const CodeNegative = -999

// 边缘情况：零值

// @Code(code=0,http=200,grpc=OK)
var ErrZero = errors.New("zero code")

// @Code(code=1,http=200,grpc=OK)
const CodeZero = 1

// 边缘情况：最大值

// @Code(code=2147483647,http=200,grpc=OK)
var ErrMaxInt = errors.New("max int32")

// @Code(code=2147483646,http=200,grpc=OK)
const CodeMaxInt = 2147483646

// 边缘情况：所有 gRPC 状态码

// @Code(code=10001,http=200,grpc=OK)
var ErrOK = errors.New("ok")

// @Code(code=10002,http=408,grpc=Canceled)
var ErrCanceled = errors.New("canceled")

// @Code(code=10003,http=500,grpc=Unknown)
var ErrUnknown = errors.New("unknown")

// @Code(code=10004,http=400,grpc=InvalidArgument)
var ErrInvalidArgument = errors.New("invalid argument")

// @Code(code=10005,http=504,grpc=DeadlineExceeded)
var ErrDeadlineExceeded = errors.New("deadline exceeded")

// @Code(code=10006,http=404,grpc=NotFound)
var ErrNotFound = errors.New("not found")

// @Code(code=10007,http=409,grpc=AlreadyExists)
var ErrAlreadyExists = errors.New("already exists")

// @Code(code=10008,http=403,grpc=PermissionDenied)
var ErrPermissionDenied = errors.New("permission denied")

// @Code(code=10009,http=429,grpc=ResourceExhausted)
var ErrResourceExhausted = errors.New("resource exhausted")

// @Code(code=10010,http=400,grpc=FailedPrecondition)
var ErrFailedPrecondition = errors.New("failed precondition")

// @Code(code=10011,http=409,grpc=Aborted)
var ErrAborted = errors.New("aborted")

// @Code(code=10012,http=400,grpc=OutOfRange)
var ErrOutOfRange = errors.New("out of range")

// @Code(code=10013,http=501,grpc=Unimplemented)
var ErrUnimplemented = errors.New("unimplemented")

// @Code(code=10014,http=500,grpc=Internal)
var ErrInternal = errors.New("internal")

// @Code(code=10015,http=503,grpc=Unavailable)
var ErrUnavailable = errors.New("unavailable")

// @Code(code=10016,http=500,grpc=DataLoss)
var ErrDataLoss = errors.New("data loss")

// @Code(code=10017,http=401,grpc=Unauthenticated)
var ErrUnauthenticated = errors.New("unauthenticated")

// 边缘情况：各种 HTTP 状态码

// @Code(code=20001,http=100,grpc=OK)
var Err100 = errors.New("continue")

// @Code(code=20002,http=101,grpc=OK)
var Err101 = errors.New("switching protocols")

// @Code(code=20003,http=200,grpc=OK)
var Err200 = errors.New("ok")

// @Code(code=20004,http=201,grpc=OK)
var Err201 = errors.New("created")

// @Code(code=20005,http=204,grpc=OK)
var Err204 = errors.New("no content")

// @Code(code=20006,http=301,grpc=OK)
var Err301 = errors.New("moved permanently")

// @Code(code=20007,http=302,grpc=OK)
var Err302 = errors.New("found")

// @Code(code=20008,http=304,grpc=OK)
var Err304 = errors.New("not modified")

// @Code(code=20009,http=400,grpc=InvalidArgument)
var Err400 = errors.New("bad request")

// @Code(code=20010,http=401,grpc=Unauthenticated)
var Err401 = errors.New("unauthorized")

// @Code(code=20011,http=403,grpc=PermissionDenied)
var Err403 = errors.New("forbidden")

// @Code(code=20012,http=404,grpc=NotFound)
var Err404 = errors.New("not found")

// @Code(code=20013,http=405,grpc=InvalidArgument)
var Err405 = errors.New("method not allowed")

// @Code(code=20014,http=408,grpc=DeadlineExceeded)
var Err408 = errors.New("request timeout")

// @Code(code=20015,http=409,grpc=Aborted)
var Err409 = errors.New("conflict")

// @Code(code=20016,http=429,grpc=ResourceExhausted)
var Err429 = errors.New("too many requests")

// @Code(code=20017,http=500,grpc=Internal)
var Err500 = errors.New("internal server error")

// @Code(code=20018,http=501,grpc=Unimplemented)
var Err501 = errors.New("not implemented")

// @Code(code=20019,http=502,grpc=Unavailable)
var Err502 = errors.New("bad gateway")

// @Code(code=20020,http=503,grpc=Unavailable)
var Err503 = errors.New("service unavailable")

// @Code(code=20021,http=504,grpc=DeadlineExceeded)
var Err504 = errors.New("gateway timeout")

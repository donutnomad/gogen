package pkg2

import "errors"

// pkg2: 使用相同的 code=50001（应该允许，因为不同包）

// @Code(code=50001,http=400,grpc=InvalidArgument)
var ErrBadRequest = errors.New("bad request in pkg2")

// @Code(code=50002,http=403,grpc=PermissionDenied)
const CodeForbidden = 50002

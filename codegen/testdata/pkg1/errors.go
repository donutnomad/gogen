package pkg1

import "errors"

// pkg1: 使用 code=50001

// @Code(code=50001,http=404,grpc=NotFound)
var ErrNotFound = errors.New("not found in pkg1")

// @Code(code=50002,http=500,grpc=Internal)
const CodeInternal = 50002

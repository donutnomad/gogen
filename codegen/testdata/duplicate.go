package duplicate

import "errors"

// 重复场景：同包内重复的 code 值（应该在代码生成时报错）

// @Code(code=40001,http=404,grpc=NotFound)
var ErrDuplicate1 = errors.New("duplicate 1")

// @Code(code=40001,http=500,grpc=Internal)
var ErrDuplicate2 = errors.New("duplicate 2")

// @Code(code=40002,http=400,grpc=InvalidArgument)
const CodeDuplicate1 = 40002

// @Code(code=40002,http=404,grpc=NotFound)
const CodeDuplicate2 = 40002

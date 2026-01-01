package grouped_merge

import "errors"

// 测试分组声明中只使用每个变量自己的注解
// var ( 上方的注解不起作用

// @GlobalTag（这个注解对分组声明中的变量不起作用）
var (
	// @Code(code=300001,http=404,grpc=NotFound)
	ErrWithSpec = errors.New("has spec annotation")
	// 没有注解，不会被识别
	ErrNoAnnotation = errors.New("no annotation, should be skipped")
	ErrLineComment  = errors.New("line comment annotation") // @Code(code=300004,http=401,grpc=Unauthenticated)
)

// 分组声明上方的注解不起作用
// @Code(code=300002,http=500,grpc=Internal)
var (
	ErrNoSpecAnnotation = errors.New("decl annotation ignored for grouped")
)

// 测试单行声明使用 decl 注解
// @Code(code=300003,http=400,grpc=InvalidArgument)
var ErrSingleLine = errors.New("single line declaration")

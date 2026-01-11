package codegen_test

import "errors"

//go:generate bash -c "cd ../../ && go build -o gogen && ./gogen gen ./codegen/codegen_test && go test ./codegen/codegen_test/..."

// ErrUserNotFound
// @Code(code=11001,http=404,grpc=NotFound)
var ErrUserNotFound = errors.New("user not found")

// ErrDatabaseError
// @Code(code=11002,http=500,grpc=Internal)
var ErrDatabaseError = errors.New("database error")

// ErrInvalidInput
// @Code(code=11003,http=400,grpc=InvalidArgument)
var ErrInvalidInput = errors.New("invalid input")

// CodeInvalidInput
// @Code(code=20001,http=400,grpc=InvalidArgument)
const CodeInvalidInput = 20001

// CodeNotFound
// @Code(code=20002,http=404,grpc=NotFound)
const CodeNotFound = 20002

// CodeStrUnauthorized
// @Code(code=30001,http=401,grpc=Unauthenticated)
const CodeStrUnauthorized = "UNAUTHORIZED"

// CodeStrForbidden
// @Code(code=30002,http=403,grpc=PermissionDenied)
const CodeStrForbidden = "FORBIDDEN"

package codegen_test_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/donutnomad/gogen/codegen/codegen_test"
	"google.golang.org/grpc/codes"
)

// TestGeneratedCodeWithErrorTypes 测试生成的代码处理 error 类型
func TestGeneratedCodeWithErrorTypes(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedHTTP int
		expectedGRPC codes.Code
		expectedCode int
		expectedName string
		expectHTTPOK bool
		expectGRPCOK bool
		expectCodeOK bool
		expectNameOK bool
	}{
		{
			name:         "ErrUserNotFound",
			err:          codegen_test.ErrUserNotFound,
			expectedHTTP: 404,
			expectedGRPC: codes.NotFound,
			expectedCode: 11001,
			expectedName: "ErrUserNotFound",
			expectHTTPOK: true,
			expectGRPCOK: true,
			expectCodeOK: true,
			expectNameOK: true,
		},
		{
			name:         "ErrDatabaseError",
			err:          codegen_test.ErrDatabaseError,
			expectedHTTP: 500,
			expectedGRPC: codes.Internal,
			expectedCode: 11002,
			expectedName: "ErrDatabaseError",
			expectHTTPOK: true,
			expectGRPCOK: true,
			expectCodeOK: true,
			expectNameOK: true,
		},
		{
			name:         "ErrInvalidInput",
			err:          codegen_test.ErrInvalidInput,
			expectedHTTP: 400,
			expectedGRPC: codes.InvalidArgument,
			expectedCode: 11003,
			expectedName: "ErrInvalidInput",
			expectHTTPOK: true,
			expectGRPCOK: true,
			expectCodeOK: true,
			expectNameOK: true,
		},
		{
			name:         "Unknown error",
			err:          errors.New("unknown"),
			expectedHTTP: 0,
			expectedGRPC: codes.Unknown,
			expectedCode: 0,
			expectedName: "",
			expectHTTPOK: false,
			expectGRPCOK: false,
			expectCodeOK: false,
			expectNameOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test GetHttpCode
			httpCode, ok := codegen_test.GetHttpCode(tt.err)
			if ok != tt.expectHTTPOK {
				t.Errorf("GetHttpCode() ok = %v, want %v", ok, tt.expectHTTPOK)
			}
			if httpCode != tt.expectedHTTP {
				t.Errorf("GetHttpCode() = %v, want %v", httpCode, tt.expectedHTTP)
			}

			// Test GetGrpcCode
			grpcCode, ok := codegen_test.GetGrpcCode(tt.err)
			if ok != tt.expectGRPCOK {
				t.Errorf("GetGrpcCode() ok = %v, want %v", ok, tt.expectGRPCOK)
			}
			if grpcCode != tt.expectedGRPC {
				t.Errorf("GetGrpcCode() = %v, want %v", grpcCode, tt.expectedGRPC)
			}

			// Test GetCode
			code, ok := codegen_test.GetCode(tt.err)
			if ok != tt.expectCodeOK {
				t.Errorf("GetCode() ok = %v, want %v", ok, tt.expectCodeOK)
			}
			if code != tt.expectedCode {
				t.Errorf("GetCode() = %v, want %v", code, tt.expectedCode)
			}

			// Test GetName
			name, ok := codegen_test.GetName(tt.err)
			if ok != tt.expectNameOK {
				t.Errorf("GetName() ok = %v, want %v", ok, tt.expectNameOK)
			}
			if name != tt.expectedName {
				t.Errorf("GetName() = %v, want %v", name, tt.expectedName)
			}
		})
	}
}

// TestGeneratedCodeWithConstTypes 测试生成的代码处理 const 类型
func TestGeneratedCodeWithConstTypes(t *testing.T) {
	tests := []struct {
		name         string
		code         int
		expectedHTTP int
		expectedGRPC codes.Code
		expectedCode int
		expectedName string
		expectHTTPOK bool
		expectGRPCOK bool
		expectCodeOK bool
		expectNameOK bool
	}{
		{
			name:         "CodeInvalidInput",
			code:         codegen_test.CodeInvalidInput,
			expectedHTTP: 400,
			expectedGRPC: codes.InvalidArgument,
			expectedCode: 20001,
			expectedName: "CodeInvalidInput",
			expectHTTPOK: true,
			expectGRPCOK: true,
			expectCodeOK: true,
			expectNameOK: true,
		},
		{
			name:         "CodeNotFound",
			code:         codegen_test.CodeNotFound,
			expectedHTTP: 404,
			expectedGRPC: codes.NotFound,
			expectedCode: 20002,
			expectedName: "CodeNotFound",
			expectHTTPOK: true,
			expectGRPCOK: true,
			expectCodeOK: true,
			expectNameOK: true,
		},
		{
			name:         "Unknown code",
			code:         99999,
			expectedHTTP: 0,
			expectedGRPC: codes.Unknown,
			expectedCode: 0,
			expectedName: "",
			expectHTTPOK: false,
			expectGRPCOK: false,
			expectCodeOK: false,
			expectNameOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test GetHttpCode
			httpCode, ok := codegen_test.GetHttpCode(tt.code)
			if ok != tt.expectHTTPOK {
				t.Errorf("GetHttpCode() ok = %v, want %v", ok, tt.expectHTTPOK)
			}
			if httpCode != tt.expectedHTTP {
				t.Errorf("GetHttpCode() = %v, want %v", httpCode, tt.expectedHTTP)
			}

			// Test GetGrpcCode
			grpcCode, ok := codegen_test.GetGrpcCode(tt.code)
			if ok != tt.expectGRPCOK {
				t.Errorf("GetGrpcCode() ok = %v, want %v", ok, tt.expectGRPCOK)
			}
			if grpcCode != tt.expectedGRPC {
				t.Errorf("GetGrpcCode() = %v, want %v", grpcCode, tt.expectedGRPC)
			}

			// Test GetCode
			code, ok := codegen_test.GetCode(tt.code)
			if ok != tt.expectCodeOK {
				t.Errorf("GetCode() ok = %v, want %v", ok, tt.expectCodeOK)
			}
			if code != tt.expectedCode {
				t.Errorf("GetCode() = %v, want %v", code, tt.expectedCode)
			}

			// Test GetName
			name, ok := codegen_test.GetName(tt.code)
			if ok != tt.expectNameOK {
				t.Errorf("GetName() ok = %v, want %v", ok, tt.expectNameOK)
			}
			if name != tt.expectedName {
				t.Errorf("GetName() = %v, want %v", name, tt.expectedName)
			}
		})
	}
}

// TestGeneratedCodeWithDifferentIntTypes 测试不同整数类型的行为
func TestGeneratedCodeWithDifferentIntTypes(t *testing.T) {
	t.Run("测试相同值但不同类型", func(t *testing.T) {
		// CodeInvalidInput 是 int 类型，值为 20001
		tests := []struct {
			name     string
			val      any
			expectOK bool
			desc     string
		}{
			{
				name:     "exact int match",
				val:      20001,
				expectOK: true,
				desc:     "int 类型完全匹配",
			},
			{
				name:     "int match via const",
				val:      codegen_test.CodeInvalidInput,
				expectOK: true,
				desc:     "通过常量名匹配（类型和值都相同）",
			},
			{
				name:     "int8 different value",
				val:      int8(127), // 不同值
				expectOK: false,
				desc:     "int8 类型，值不同",
			},
			{
				name:     "int16 same value",
				val:      int16(20001),
				expectOK: true,
				desc:     "int16 类型，值相同应该匹配",
			},
			{
				name:     "int32 same value",
				val:      int32(20001),
				expectOK: true,
				desc:     "int32 类型，值相同应该匹配",
			},
			{
				name:     "int64 same value",
				val:      int64(20001),
				expectOK: true,
				desc:     "int64 类型，值相同应该匹配",
			},
			{
				name:     "uint same value",
				val:      uint(20001),
				expectOK: true,
				desc:     "uint 类型，值相同应该匹配",
			},
			{
				name:     "uint8 different value",
				val:      uint8(201),
				expectOK: false,
				desc:     "uint8 类型，值不同",
			},
			{
				name:     "uint16 same value",
				val:      uint16(20001),
				expectOK: true,
				desc:     "uint16 类型，值相同应该匹配",
			},
			{
				name:     "uint32 same value",
				val:      uint32(20001),
				expectOK: true,
				desc:     "uint32 类型，值相同应该匹配",
			},
			{
				name:     "uint64 same value",
				val:      uint64(20001),
				expectOK: true,
				desc:     "uint64 类型，值相同应该匹配",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var httpCode int
				var ok bool

				// 使用类型切换调用 GetHttpCode
				switch v := tt.val.(type) {
				case int:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case int8:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case int16:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case int32:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case int64:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case uint:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case uint8:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case uint16:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case uint32:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case uint64:
					httpCode, ok = codegen_test.GetHttpCode(v)
				default:
					t.Fatalf("unexpected type: %T", v)
				}

				if ok != tt.expectOK {
					t.Errorf("%s: GetHttpCode(%T=%v) ok = %v, want %v (httpCode=%v)",
						tt.desc, tt.val, tt.val, ok, tt.expectOK, httpCode)
				}

				if tt.expectOK && httpCode != 400 {
					t.Errorf("%s: GetHttpCode(%T=%v) = %v, want 400",
						tt.desc, tt.val, tt.val, httpCode)
				}

				if !tt.expectOK && httpCode != 0 {
					t.Errorf("%s: GetHttpCode(%T=%v) 应该返回 0，实际返回 %v",
						tt.desc, tt.val, tt.val, httpCode)
				}
			})
		}
	})

	t.Run("测试所有 Get 方法的类型安全性", func(t *testing.T) {
		// 测试 int32 类型的值
		var val32 int32 = 20001

		// GetHttpCode
		httpCode, ok := codegen_test.GetHttpCode(val32)
		if !ok {
			t.Errorf("GetHttpCode(int32) 应该返回 ok=true（值相同应该匹配），实际 ok=false")
		}
		if ok && httpCode != 400 {
			t.Errorf("GetHttpCode(int32=20001) 应该返回 httpCode=400，实际 httpCode=%v", httpCode)
		}

		// GetGrpcCode
		grpcCode, ok := codegen_test.GetGrpcCode(val32)
		if !ok {
			t.Errorf("GetGrpcCode(int32) 应该返回 ok=true（值相同应该匹配），实际 ok=false")
		}
		if ok && grpcCode != codes.InvalidArgument {
			t.Errorf("GetGrpcCode(int32=20001) 应该返回 grpcCode=InvalidArgument，实际 grpcCode=%v", grpcCode)
		}

		// GetCode
		code, ok := codegen_test.GetCode(val32)
		if !ok {
			t.Errorf("GetCode(int32) 应该返回 ok=true（值相同应该匹配），实际 ok=false")
		}
		if ok && code != 20001 {
			t.Errorf("GetCode(int32=20001) 应该返回 code=20001，实际 code=%v", code)
		}

		// GetName
		name, ok := codegen_test.GetName(val32)
		if !ok {
			t.Errorf("GetName(int32) 应该返回 ok=true（值相同应该匹配），实际 ok=false")
		}
		if ok && name != "CodeInvalidInput" {
			t.Errorf("GetName(int32=20001) 应该返回 name=CodeInvalidInput，实际 name=%v", name)
		}
	})

	t.Run("验证类型转换会正确匹配相同的值", func(t *testing.T) {
		// 值相同的不同整数类型应该匹配
		testCases := []struct {
			val         any
			shouldMatch bool
			desc        string
		}{
			{int8(127), false, "int8(127) 值不匹配 20001"},
			{int16(20001), true, "int16(20001) 值匹配"},
			{int32(20001), true, "int32(20001) 值匹配"},
			{int64(20001), true, "int64(20001) 值匹配"},
			{uint(20001), true, "uint(20001) 值匹配"},
			{uint8(255), false, "uint8(255) 值不匹配 20001"},
			{uint16(20001), true, "uint16(20001) 值匹配"},
			{uint32(20001), true, "uint32(20001) 值匹配"},
			{uint64(20001), true, "uint64(20001) 值匹配"},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("type_%T", tc.val), func(t *testing.T) {
				var httpCode int
				var ok bool

				switch v := tc.val.(type) {
				case int8:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case int16:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case int32:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case int64:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case uint:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case uint8:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case uint16:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case uint32:
					httpCode, ok = codegen_test.GetHttpCode(v)
				case uint64:
					httpCode, ok = codegen_test.GetHttpCode(v)
				}

				if ok != tc.shouldMatch {
					t.Errorf("%s: 类型 %T 值 %v ok=%v, 期望 ok=%v (httpCode=%v)",
						tc.desc, tc.val, tc.val, ok, tc.shouldMatch, httpCode)
				}

				if tc.shouldMatch && ok && httpCode != 400 {
					t.Errorf("%s: 期望 httpCode=400, 实际 httpCode=%v", tc.desc, httpCode)
				}
			})
		}
	})
}

// TestGeneratedCodeWithEdgeCases 测试边缘情况
func TestGeneratedCodeWithEdgeCases(t *testing.T) {
	// 测试零值
	httpCode, ok := codegen_test.GetHttpCode(0)
	if ok {
		t.Errorf("GetHttpCode(0) should return ok=false for unregistered code, got ok=true with httpCode=%v", httpCode)
	}

	// 测试负数
	httpCode, ok = codegen_test.GetHttpCode(-1)
	if ok {
		t.Errorf("GetHttpCode(-1) should return ok=false for unregistered code, got ok=true with httpCode=%v", httpCode)
	}

	// 测试最大值
	httpCode, ok = codegen_test.GetHttpCode(2147483647)
	if ok {
		t.Errorf("GetHttpCode(2147483647) should return ok=false for unregistered code, got ok=true with httpCode=%v", httpCode)
	}

	// 测试 nil error
	var nilErr error
	httpCode, ok = codegen_test.GetHttpCode(nilErr)
	if ok {
		t.Errorf("GetHttpCode(nil) should return ok=false, got ok=true with httpCode=%v", httpCode)
	}
}

// TestGeneratedCodeWithStringTypes 测试生成的代码处理字符串类型
func TestGeneratedCodeWithStringTypes(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		expectedHTTP int
		expectedGRPC codes.Code
		expectedCode int
		expectedName string
		expectHTTPOK bool
		expectGRPCOK bool
		expectCodeOK bool
		expectNameOK bool
	}{
		{
			name:         "CodeStrUnauthorized",
			code:         codegen_test.CodeStrUnauthorized,
			expectedHTTP: 401,
			expectedGRPC: codes.Unauthenticated,
			expectedCode: 30001,
			expectedName: "CodeStrUnauthorized",
			expectHTTPOK: true,
			expectGRPCOK: true,
			expectCodeOK: true,
			expectNameOK: true,
		},
		{
			name:         "CodeStrForbidden",
			code:         codegen_test.CodeStrForbidden,
			expectedHTTP: 403,
			expectedGRPC: codes.PermissionDenied,
			expectedCode: 30002,
			expectedName: "CodeStrForbidden",
			expectHTTPOK: true,
			expectGRPCOK: true,
			expectCodeOK: true,
			expectNameOK: true,
		},
		{
			name:         "Unknown string code",
			code:         "UNKNOWN",
			expectedHTTP: 0,
			expectedGRPC: codes.Unknown,
			expectedCode: 0,
			expectedName: "",
			expectHTTPOK: false,
			expectGRPCOK: false,
			expectCodeOK: false,
			expectNameOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test GetHttpCode
			httpCode, ok := codegen_test.GetHttpCode(tt.code)
			if ok != tt.expectHTTPOK {
				t.Errorf("GetHttpCode() ok = %v, want %v", ok, tt.expectHTTPOK)
			}
			if httpCode != tt.expectedHTTP {
				t.Errorf("GetHttpCode() = %v, want %v", httpCode, tt.expectedHTTP)
			}

			// Test GetGrpcCode
			grpcCode, ok := codegen_test.GetGrpcCode(tt.code)
			if ok != tt.expectGRPCOK {
				t.Errorf("GetGrpcCode() ok = %v, want %v", ok, tt.expectGRPCOK)
			}
			if grpcCode != tt.expectedGRPC {
				t.Errorf("GetGrpcCode() = %v, want %v", grpcCode, tt.expectedGRPC)
			}

			// Test GetCode
			code, ok := codegen_test.GetCode(tt.code)
			if ok != tt.expectCodeOK {
				t.Errorf("GetCode() ok = %v, want %v", ok, tt.expectCodeOK)
			}
			if code != tt.expectedCode {
				t.Errorf("GetCode() = %v, want %v", code, tt.expectedCode)
			}

			// Test GetName
			name, ok := codegen_test.GetName(tt.code)
			if ok != tt.expectNameOK {
				t.Errorf("GetName() ok = %v, want %v", ok, tt.expectNameOK)
			}
			if name != tt.expectedName {
				t.Errorf("GetName() = %v, want %v", name, tt.expectedName)
			}
		})
	}
}

// BenchmarkGetHttpCode 性能基准测试
func BenchmarkGetHttpCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		codegen_test.GetHttpCode(codegen_test.ErrUserNotFound)
	}
}

// BenchmarkGetCodeWithConst 性能基准测试
func BenchmarkGetCodeWithConst(b *testing.B) {
	for i := 0; i < b.N; i++ {
		codegen_test.GetCode(codegen_test.CodeInvalidInput)
	}
}

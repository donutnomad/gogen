package mockgen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseInterface_EmbeddedInterfaces(t *testing.T) {
	// 测试嵌入接口的解析
	info, err := ParseInterface("example/interfaces.go", "ReadWriter")
	require.NoError(t, err)
	require.NotNil(t, info)

	require.Equal(t, "ReadWriter", info.Name)
	require.Equal(t, "testdata", info.PackageName)

	// ReadWriter 应该包含 Reader 的 Read 方法、Writer 的 Write 方法、以及自身的 Close 方法
	// 总共 3 个方法
	require.Len(t, info.Methods, 3, "ReadWriter should have 3 methods: Read, Write, Close")

	// 检查方法名
	methodNames := make(map[string]bool)
	for _, m := range info.Methods {
		methodNames[m.Name] = true
	}
	require.True(t, methodNames["Read"], "should have Read method from Reader")
	require.True(t, methodNames["Write"], "should have Write method from Writer")
	require.True(t, methodNames["Close"], "should have Close method")
}

func TestParseInterface_SimpleInterface(t *testing.T) {
	// 测试简单接口的解析
	info, err := ParseInterface("example/interfaces.go", "UserService")
	require.NoError(t, err)
	require.NotNil(t, info)

	require.Equal(t, "UserService", info.Name)
	require.Len(t, info.Methods, 3)
}

func TestParseInterface_GenericInterface(t *testing.T) {
	// 测试泛型接口的解析
	info, err := ParseInterface("example/generic_interfaces.go", "GenericRepository")
	require.NoError(t, err)
	require.NotNil(t, info)

	require.Equal(t, "GenericRepository", info.Name)
	require.Len(t, info.TypeParams, 1)
	require.Equal(t, "T", info.TypeParams[0].Name)
	require.Equal(t, "any", info.TypeParams[0].Constraint)
}

func TestParseInterface_StdLibEmbedded(t *testing.T) {
	// 测试嵌入标准库接口的解析
	info, err := ParseInterface("example/interfaces.go", "IOReadWriter")
	require.NoError(t, err)
	require.NotNil(t, info)

	require.Equal(t, "IOReadWriter", info.Name)

	// IOReadWriter 应该包含 io.Reader 的 Read 方法、io.Writer 的 Write 方法、以及自身的 Sync 方法
	// 总共 3 个方法
	require.Len(t, info.Methods, 3, "IOReadWriter should have 3 methods: Read, Write, Sync")

	// 检查方法名
	methodNames := make(map[string]bool)
	for _, m := range info.Methods {
		methodNames[m.Name] = true
	}
	require.True(t, methodNames["Read"], "should have Read method from io.Reader")
	require.True(t, methodNames["Write"], "should have Write method from io.Writer")
	require.True(t, methodNames["Sync"], "should have Sync method")
}

func TestParseInterface_NamedReturns(t *testing.T) {
	// 测试命名返回值的解析
	info, err := ParseInterface("example/interfaces.go", "NamedReturnsInterface")
	require.NoError(t, err)
	require.NotNil(t, info)

	require.Equal(t, "NamedReturnsInterface", info.Name)
	require.Len(t, info.Methods, 2)

	// 查找 GetData 方法
	var getData *MethodInfo
	var process *MethodInfo
	for _, m := range info.Methods {
		if m.Name == "GetData" {
			getData = m
		} else if m.Name == "Process" {
			process = m
		}
	}

	// 测试 GetData 方法的命名返回值
	require.NotNil(t, getData, "should have GetData method")
	require.Len(t, getData.Results, 3, "GetData should have 3 return values")
	require.Equal(t, "data", getData.Results[0].Name, "first return should be named 'data'")
	require.Equal(t, "[]byte", getData.Results[0].Type)
	require.Equal(t, "found", getData.Results[1].Name, "second return should be named 'found'")
	require.Equal(t, "bool", getData.Results[1].Type)
	require.Equal(t, "err", getData.Results[2].Name, "third return should be named 'err'")
	require.Equal(t, "error", getData.Results[2].Type)

	// 测试 Process 方法（全部未命名返回值）
	require.NotNil(t, process, "should have Process method")
	require.Len(t, process.Results, 2, "Process should have 2 return values")
	// 未命名的返回值 Name 为空
	require.Equal(t, "", process.Results[0].Name, "first return should be unnamed")
	require.Equal(t, "string", process.Results[0].Type)
	require.Equal(t, "", process.Results[1].Name, "second return should be unnamed")
	require.Equal(t, "error", process.Results[1].Type)
}

func TestParseInterface_EmbeddedNamedReturns(t *testing.T) {
	// 测试嵌入接口中的命名返回值（Reader 接口的 Read 方法有 n int, err error）
	info, err := ParseInterface("example/interfaces.go", "ReadWriter")
	require.NoError(t, err)
	require.NotNil(t, info)

	// 查找 Read 方法
	var read *MethodInfo
	for _, m := range info.Methods {
		if m.Name == "Read" {
			read = m
			break
		}
	}

	require.NotNil(t, read, "should have Read method")
	require.Len(t, read.Results, 2, "Read should have 2 return values")
	require.Equal(t, "n", read.Results[0].Name, "first return should be named 'n'")
	require.Equal(t, "int", read.Results[0].Type)
	require.Equal(t, "err", read.Results[1].Name, "second return should be named 'err'")
	require.Equal(t, "error", read.Results[1].Type)
}

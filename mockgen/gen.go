package mockgen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/donutnomad/gg"
)

// generateDefinition 为一组目标生成 gg 定义
func (g *MockGenerator) generateDefinition(targets []*mockTargetInfo) (*gg.Generator, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("没有目标需要生成")
	}

	gen := gg.New()
	gen.SetPackage(targets[0].interface_.PackageName)

	// 添加 gomock 依赖
	gomockPkg := gen.P("go.uber.org/mock/gomock")

	// 检查是否需要 reflect 包
	needReflect := false
	for _, t := range targets {
		if len(t.interface_.Methods) > 0 {
			needReflect = true
			break
		}
	}
	var reflectPkg *gg.PackageRef
	if needReflect {
		reflectPkg = gen.P("reflect")
	}

	// 收集所有需要的导入
	importPaths := make(map[string]bool)
	for _, t := range targets {
		for path := range t.interface_.Imports {
			importPaths[path] = true
		}
	}
	for path := range importPaths {
		if path != "go.uber.org/mock/gomock" && path != "reflect" {
			gen.P(path)
		}
	}

	// 为每个接口生成 mock
	for i, t := range targets {
		if i > 0 {
			gen.Body().AddLine()
		}
		generateMockInterface(gen, t.interface_, t.params, gomockPkg, reflectPkg)
	}

	return gen, nil
}

// generateMockInterface 生成单个接口的 mock 代码
func generateMockInterface(gen *gg.Generator, iface *InterfaceInfo, params *MockParams, gomockPkg, reflectPkg *gg.PackageRef) {
	mockName := params.MockName
	if mockName == "" {
		mockName = "Mock" + iface.Name
	}

	// 生成类型参数字符串
	typeParamsLong, typeParamsShort := formatTypeParams(iface.TypeParams)

	body := gen.Body()

	// ====== Mock 结构体
	body.Append(gg.S("// %s is a mock of %s interface.", mockName, iface.Name))
	mockStruct := body.NewStruct(mockName + typeParamsLong)
	mockStruct.AddField("ctrl", gomockPkg.Ptr("Controller"))
	mockStruct.AddField("recorder", fmt.Sprintf("*%sMockRecorder%s", mockName, typeParamsShort))
	mockStruct.AddField("isgomock", "struct{}")

	body.AddLine()

	// ====== Recorder 结构体
	body.Append(gg.S("// %sMockRecorder is the mock recorder for %s.", mockName, mockName))
	recorderStruct := body.NewStruct(mockName + "MockRecorder" + typeParamsLong)
	recorderStruct.AddField("mock", fmt.Sprintf("*%s%s", mockName, typeParamsShort))

	body.AddLine()

	// ====== New 构造函数
	body.Append(gg.S("// New%s creates a new mock instance.", mockName))
	body.NewFunction("New"+mockName+typeParamsLong).
		AddParameter("ctrl", gomockPkg.Ptr("Controller")).
		AddResult("", fmt.Sprintf("*%s%s", mockName, typeParamsShort)).
		AddBody(
			gg.S("mock := &%s%s{ctrl: ctrl}", mockName, typeParamsShort),
			gg.S("mock.recorder = &%sMockRecorder%s{mock}", mockName, typeParamsShort),
			gg.String("return mock"),
		)

	body.AddLine()

	// ====== EXPECT 方法
	body.Append(gg.String("// EXPECT returns an object that allows the caller to indicate expected use."))
	body.NewFunction("EXPECT").
		WithReceiver("m", fmt.Sprintf("*%s%s", mockName, typeParamsShort)).
		AddResult("", fmt.Sprintf("*%sMockRecorder%s", mockName, typeParamsShort)).
		AddBody(gg.String("return m.recorder"))

	// ====== 生成每个方法的 mock
	// 按方法名排序
	methods := make([]*MethodInfo, len(iface.Methods))
	copy(methods, iface.Methods)
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})

	for _, method := range methods {
		body.AddLine()
		generateMockMethod(body, mockName, typeParamsShort, method, gomockPkg)
		body.AddLine()
		generateRecorderMethod(body, mockName, typeParamsShort, method, gomockPkg, reflectPkg, params.Typed)

		if params.Typed {
			body.AddLine()
			generateTypedCall(body, mockName, typeParamsLong, typeParamsShort, method)
		}
	}
}

// generateMockMethod 生成方法的 mock 实现
func generateMockMethod(body *gg.Group, mockName, typeParams string, method *MethodInfo, _ *gg.PackageRef) {
	argNames := getArgNames(method)
	argTypes := getArgTypes(method)
	retTypes := getRetTypes(method)

	// 方法签名
	body.Append(gg.S("// %s mocks base method.", method.Name))

	fn := body.NewFunction(method.Name).
		WithReceiver("m", fmt.Sprintf("*%s%s", mockName, typeParams))

	// 添加参数
	for i, name := range argNames {
		fn.AddParameter(name, argTypes[i])
	}

	// 添加返回值
	for _, ret := range retTypes {
		fn.AddResult("", ret)
	}

	// 生成方法体
	var bodyLines []any
	bodyLines = append(bodyLines, gg.String("m.ctrl.T.Helper()"))

	// 构建 Call 参数
	if method.Variadic == nil {
		// 无可变参数
		callArgs := ""
		if len(argNames) > 0 {
			callArgs = ", " + strings.Join(argNames, ", ")
		}

		if len(retTypes) == 0 {
			bodyLines = append(bodyLines, gg.S(`m.ctrl.Call(m, %q%s)`, method.Name, callArgs))
		} else {
			bodyLines = append(bodyLines, gg.S(`ret := m.ctrl.Call(m, %q%s)`, method.Name, callArgs))
			for i, ret := range retTypes {
				bodyLines = append(bodyLines, gg.S("ret%d, _ := ret[%d].(%s)", i, i, ret))
			}
			retVars := make([]string, len(retTypes))
			for i := range retTypes {
				retVars[i] = fmt.Sprintf("ret%d", i)
			}
			bodyLines = append(bodyLines, gg.S("return %s", strings.Join(retVars, ", ")))
		}
	} else {
		// 有可变参数
		nonVariadicArgs := argNames[:len(argNames)-1]
		variadicArg := argNames[len(argNames)-1]

		bodyLines = append(bodyLines, gg.S("varargs := []any{%s}", strings.Join(nonVariadicArgs, ", ")))
		bodyLines = append(bodyLines, gg.S("for _, a := range %s {", variadicArg))
		bodyLines = append(bodyLines, gg.String("\tvarargs = append(varargs, a)"))
		bodyLines = append(bodyLines, gg.String("}"))

		if len(retTypes) == 0 {
			bodyLines = append(bodyLines, gg.S(`m.ctrl.Call(m, %q, varargs...)`, method.Name))
		} else {
			bodyLines = append(bodyLines, gg.S(`ret := m.ctrl.Call(m, %q, varargs...)`, method.Name))
			for i, ret := range retTypes {
				bodyLines = append(bodyLines, gg.S("ret%d, _ := ret[%d].(%s)", i, i, ret))
			}
			retVars := make([]string, len(retTypes))
			for i := range retTypes {
				retVars[i] = fmt.Sprintf("ret%d", i)
			}
			bodyLines = append(bodyLines, gg.S("return %s", strings.Join(retVars, ", ")))
		}
	}

	fn.AddBody(bodyLines...)
}

// generateRecorderMethod 生成 Recorder 方法
func generateRecorderMethod(body *gg.Group, mockName, typeParams string, method *MethodInfo, gomockPkg, reflectPkg *gg.PackageRef, typed bool) {
	argNames := getArgNames(method)

	// 构建参数字符串 (Recorder 方法的参数都是 any 类型)
	var params []string
	if method.Variadic == nil {
		params = append(params, argNames...)
	} else {
		params = append(params, argNames[:len(argNames)-1]...)
	}

	body.Append(gg.S("// %s indicates an expected call of %s.", method.Name, method.Name))

	var retType any
	if typed {
		retType = fmt.Sprintf("*%s%sCall%s", mockName, method.Name, typeParams)
	} else {
		retType = gomockPkg.Ptr("Call")
	}

	fn := body.NewFunction(method.Name).
		WithReceiver("mr", fmt.Sprintf("*%sMockRecorder%s", mockName, typeParams)).
		AddResult("", retType)

	// 添加参数 - 使用 AddParameters 合并相同类型的参数
	if len(params) > 0 {
		fn.AddParameters(params, "any")
	}
	if method.Variadic != nil {
		fn.AddParameter(argNames[len(argNames)-1], "...any")
	}

	// 生成方法体
	var bodyLines []any
	bodyLines = append(bodyLines, gg.String("mr.mock.ctrl.T.Helper()"))

	// 构建调用参数
	var callArgs string
	if method.Variadic == nil {
		if len(argNames) > 0 {
			callArgs = ", " + strings.Join(argNames, ", ")
		}
	} else {
		if len(argNames) == 1 {
			callArgs = ", " + argNames[0] + "..."
		} else {
			bodyLines = append(bodyLines, gg.S("varargs := append([]any{%s}, %s...)",
				strings.Join(argNames[:len(argNames)-1], ", "),
				argNames[len(argNames)-1]))
			callArgs = ", varargs..."
		}
	}

	reflectName := "reflect"
	if reflectPkg != nil {
		reflectName = reflectPkg.Alias()
	}

	if typed {
		bodyLines = append(bodyLines, gg.S(
			`call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, %q, %s.TypeOf((*%s%s)(nil).%s)%s)`,
			method.Name, reflectName, mockName, typeParams, method.Name, callArgs))
		bodyLines = append(bodyLines, gg.S("return &%s%sCall%s{Call: call}", mockName, method.Name, typeParams))
	} else {
		bodyLines = append(bodyLines, gg.S(
			`return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, %q, %s.TypeOf((*%s%s)(nil).%s)%s)`,
			method.Name, reflectName, mockName, typeParams, method.Name, callArgs))
	}

	fn.AddBody(bodyLines...)
}

// generateTypedCall 生成类型安全的 Call 包装
func generateTypedCall(body *gg.Group, mockName, typeParamsLong, typeParamsShort string, method *MethodInfo) {
	callStructName := mockName + method.Name + "Call"
	argTypes := getArgTypes(method)
	retTypes := getRetTypes(method)
	retNames := getRetNames(method)

	// ====== Call 结构体
	body.Append(gg.S("// %s%sCall wrap *gomock.Call", mockName, method.Name))
	callStruct := body.NewStruct(callStructName + typeParamsLong)
	callStruct.AddField("", "*gomock.Call")

	body.AddLine()

	// ====== Return 方法
	body.Append(gg.String("// Return rewrite *gomock.Call.Return"))
	returnFn := body.NewFunction("Return").
		WithReceiver("c", fmt.Sprintf("*%s%s", callStructName, typeParamsShort)).
		AddResult("", fmt.Sprintf("*%s%s", callStructName, typeParamsShort))

	for i, ret := range retTypes {
		returnFn.AddParameter(retNames[i], ret)
	}

	retArgsJoin := strings.Join(retNames, ", ")
	returnFn.AddBody(
		gg.S("c.Call = c.Call.Return(%s)", retArgsJoin),
		gg.String("return c"),
	)

	body.AddLine()

	// ====== Do 方法
	argString := strings.Join(argTypes, ", ")
	var retString string
	if len(retTypes) == 1 {
		retString = " " + retTypes[0]
	} else if len(retTypes) > 1 {
		retString = " (" + strings.Join(retTypes, ", ") + ")"
	}

	body.Append(gg.String("// Do rewrite *gomock.Call.Do"))
	body.NewFunction("Do").
		WithReceiver("c", fmt.Sprintf("*%s%s", callStructName, typeParamsShort)).
		AddParameter("f", fmt.Sprintf("func(%s)%s", argString, retString)).
		AddResult("", fmt.Sprintf("*%s%s", callStructName, typeParamsShort)).
		AddBody(
			gg.String("c.Call = c.Call.Do(f)"),
			gg.String("return c"),
		)

	body.AddLine()

	// ====== DoAndReturn 方法
	body.Append(gg.String("// DoAndReturn rewrite *gomock.Call.DoAndReturn"))
	body.NewFunction("DoAndReturn").
		WithReceiver("c", fmt.Sprintf("*%s%s", callStructName, typeParamsShort)).
		AddParameter("f", fmt.Sprintf("func(%s)%s", argString, retString)).
		AddResult("", fmt.Sprintf("*%s%s", callStructName, typeParamsShort)).
		AddBody(
			gg.String("c.Call = c.Call.DoAndReturn(f)"),
			gg.String("return c"),
		)
}

// formatTypeParams 格式化类型参数
func formatTypeParams(params []*TypeParamInfo) (long, short string) {
	if len(params) == 0 {
		return "", ""
	}

	var longParts, shortParts []string
	for _, p := range params {
		longParts = append(longParts, p.Name+" "+p.Constraint)
		shortParts = append(shortParts, p.Name)
	}

	return "[" + strings.Join(longParts, ", ") + "]", "[" + strings.Join(shortParts, ", ") + "]"
}

// getArgNames 获取参数名列表
func getArgNames(method *MethodInfo) []string {
	var names []string

	for i, p := range method.Params {
		name := p.Name
		if name == "" || name == "_" {
			name = fmt.Sprintf("arg%d", i)
		}
		names = append(names, name)
	}

	if method.Variadic != nil {
		name := method.Variadic.Name
		if name == "" {
			name = fmt.Sprintf("arg%d", len(method.Params))
		}
		names = append(names, name)
	}

	return names
}

// getArgTypes 获取参数类型列表
func getArgTypes(method *MethodInfo) []string {
	var types []string

	for _, p := range method.Params {
		types = append(types, p.Type)
	}

	if method.Variadic != nil {
		types = append(types, "..."+method.Variadic.Type)
	}

	return types
}

// getRetTypes 获取返回类型列表
func getRetTypes(method *MethodInfo) []string {
	var types []string

	for _, r := range method.Results {
		types = append(types, r.Type)
	}

	return types
}

// getRetNames 获取返回值名称列表
func getRetNames(method *MethodInfo) []string {
	var names []string

	for i, r := range method.Results {
		name := r.Name
		if name == "" || name == "_" {
			name = fmt.Sprintf("arg%d", i)
		}
		names = append(names, name)
	}

	return names
}

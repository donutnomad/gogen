package plugin

import (
	"reflect"
	"strconv"
)

// ParseParamsFromStruct 从结构体的tag解析参数定义
// 支持的tag: name, required, default, description
//
// 示例:
//
//	type Params struct {
//	    Prefix      string `param:"name=prefix,required=false,default=,description=生成的 Schema 结构体前缀"`
//	    Patch       string `param:"name=patch,required=false,default=,description=Patch 模式: v2 或 full"`
//	    PatchMapper string `param:"name=patch_mapper,required=false,default=,description=Patch mapper 方法，格式: Type.Method"`
//	}
//
//	params := plugin.ParseParamsFromStruct(Params{})
func ParseParamsFromStruct(v any) []ParamDef {
	val := reflect.ValueOf(v)
	typ := val.Type()

	// 如果是指针,解引用
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// 必须是结构体
	if typ.Kind() != reflect.Struct {
		return nil
	}

	var params []ParamDef

	// 遍历所有字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// 获取 param tag
		tag := field.Tag.Get("param")
		if tag == "" {
			continue
		}

		// 解析tag
		paramDef := parseParamTag(tag)
		if paramDef.Name != "" {
			params = append(params, paramDef)
		}
	}

	return params
}

// parseParamTag 解析 param tag 字符串
// 格式: name=xxx,required=true,default=xxx,description=xxx
func parseParamTag(tag string) ParamDef {
	var param ParamDef

	// 简单的键值对解析
	pairs := splitTag(tag)
	for key, value := range pairs {
		switch key {
		case "name":
			param.Name = value
		case "required":
			param.Required = value == "true"
		case "default":
			param.Default = value
		case "description":
			param.Description = value
		}
	}

	return param
}

// splitTag 分割tag字符串为键值对
// 格式: key1=value1,key2=value2,...
func splitTag(tag string) map[string]string {
	result := make(map[string]string)

	var key, value string
	var inKey = true
	var escaped = false

	for i := 0; i < len(tag); i++ {
		ch := tag[i]

		// 处理转义
		if escaped {
			if inKey {
				key += string(ch)
			} else {
				value += string(ch)
			}
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		// 处理分隔符
		if ch == '=' && inKey {
			inKey = false
			continue
		}

		if ch == ',' {
			// 保存当前键值对
			if key != "" {
				result[key] = value
			}
			key = ""
			value = ""
			inKey = true
			continue
		}

		// 累积字符
		if inKey {
			key += string(ch)
		} else {
			value += string(ch)
		}
	}

	// 保存最后一个键值对
	if key != "" {
		result[key] = value
	}

	return result
}

// ParseParamBool 解析参数为bool值
func ParseParamBool(value string) bool {
	b, _ := strconv.ParseBool(value)
	return b
}

// ParseAnnotationParams 将注解的参数解析到目标结构体中
// annotation: 注解对象，包含参数键值对
// target: 目标结构体（必须是指针）
// paramDefs: 参数定义列表，用于应用默认值
//
// 示例:
//
//	var params GsqlParams
//	err := plugin.ParseAnnotationParams(annotation, &params, paramDefs)
func ParseAnnotationParams(annotation *Annotation, target any, paramDefs []ParamDef) error {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return nil // 必须是非nil指针
	}

	val = val.Elem()
	typ := val.Type()

	if typ.Kind() != reflect.Struct {
		return nil // 必须是结构体
	}

	// 创建参数定义的映射，方便查找默认值
	defMap := make(map[string]ParamDef)
	for _, def := range paramDefs {
		defMap[def.Name] = def
	}

	// 遍历结构体字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		// 解析 param tag 获取参数名
		tag := field.Tag.Get("param")
		if tag == "" {
			continue
		}

		paramDef := parseParamTag(tag)
		paramName := paramDef.Name
		if paramName == "" {
			continue
		}

		// 从注解中获取参数值
		paramValue := annotation.GetParam(paramName)

		// 如果注解中没有该参数，使用默认值
		if paramValue == "" {
			if def, ok := defMap[paramName]; ok {
				paramValue = def.Default
			}
		}

		// 设置字段值
		if err := setFieldValue(fieldVal, paramValue); err != nil {
			return err
		}
	}

	return nil
}

// setFieldValue 设置字段值，支持 string, int, bool 等基本类型
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == "" {
			value = "0"
		}
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value == "" {
			value = "0"
		}
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil && value != "" {
			return err
		}
		field.SetBool(boolVal)
	case reflect.Float32, reflect.Float64:
		if value == "" {
			value = "0"
		}
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	}
	return nil
}

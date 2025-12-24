package settergen

import (
	"strings"
	"unicode"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/internal/gormparse"
)

// generateToMapMethod 生成 ToMap 方法（full 模式）
func generateToMapMethod(gen *gg.Generator, model *gormparse.GormModelInfo) {
	rawModelName := model.Name
	receiverVar := strings.ToLower(rawModelName[:1])

	// 计算字段数量
	fieldCount := 0
	for _, f := range model.Fields {
		if f.ColumnName != "" {
			fieldCount++
		}
	}

	// 构建方法体
	body := []any{
		gg.S("values := make(map[string]any, %d)", fieldCount),
	}

	for _, f := range model.Fields {
		if f.ColumnName == "" {
			continue
		}
		body = append(body,
			gg.S("values[%s] = %s.%s", gg.Lit(f.ColumnName), receiverVar, f.Name),
		)
	}

	body = append(body, gg.Return(gg.S("values")))

	gen.Body().NewFunction("ToMap").
		WithReceiver(receiverVar, "*"+rawModelName).
		AddResult("", "map[string]any").
		AddBody(body...)
}

// generateSetterV1 生成 setter v1 模式的代码（Patch 结构体 + setter 方法）
func generateSetterV1(gen *gg.Generator, model *gormparse.GormModelInfo) {
	// 添加 mo 包导入
	moPkg := gen.P("github.com/samber/mo")

	// 生成 Patch 结构体
	generatePatchStruct(gen, model, moPkg)

	// 生成 setter 方法
	generateSetterMethods(gen, model, moPkg)
}

// generatePatchStruct 生成 Patch 结构体
func generatePatchStruct(gen *gg.Generator, model *gormparse.GormModelInfo, moPkg *gg.PackageRef) {
	patchName := model.Name + "Patch"
	structDef := gen.Body().NewStruct(patchName)

	for _, field := range model.Fields {
		// 跳过 patch 字段本身
		if strings.ToLower(field.Name) == "patch" {
			continue
		}

		// 使用字符串格式生成 mo.Option[Type]，避免类型参数被加上包前缀
		optionType := gg.NewInlineGroup().Append(
			moPkg.Type("Option"),
			gg.S("[%s]", field.Type),
		)
		structDef.AddField(field.Name, optionType)
	}
}

// generateSetterMethods 生成 setter 方法
func generateSetterMethods(gen *gg.Generator, model *gormparse.GormModelInfo, moPkg *gg.PackageRef) {
	rawModelName := model.Name
	receiverVar := strings.ToLower(rawModelName[:1])
	patchTypeName := rawModelName + "Patch"

	for _, field := range model.Fields {
		// 跳过 patch 字段本身
		if strings.ToLower(field.Name) == "patch" {
			continue
		}

		methodName := "set" + field.Name
		paramName := safeParamName(field.Name)

		gen.Body().NewFunction(methodName).
			WithReceiver(receiverVar, "*"+rawModelName).
			AddParameter(paramName, field.Type).
			AddBody(
				gg.S("%s.%s = %s", receiverVar, field.Name, paramName),
				gg.NewInlineGroup().Append(
					gg.S("%s.patch.%s = ", receiverVar, field.Name),
					moPkg.Call("Some", paramName),
				),
			)
		gen.Body().AddLine()
	}

	// 生成 ClearPatch 方法
	gen.Body().NewFunction("ClearPatch").
		WithReceiver(receiverVar, "*"+rawModelName).
		AddBody(
			gg.S("%s.patch = %s{}", receiverVar, patchTypeName),
		)

	gen.Body().AddLine()

	// 生成 ExportPatch 方法
	gen.Body().NewFunction("ExportPatch").
		WithReceiver(receiverVar, "*"+rawModelName).
		AddResult("", "*"+patchTypeName).
		AddBody(
			gg.If(gg.S("%s == nil", receiverVar)).
				AddBody(
					gg.S("var def %s", patchTypeName),
					gg.Return(gg.S("&def")),
				),
			gg.Return(gg.S("&%s.patch", receiverVar)),
		)
}

// lowerFirst 将首字母转换为小写
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// isGoKeyword 检查是否是 Go 关键词
func isGoKeyword(s string) bool {
	keywords := map[string]bool{
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true,
		"interface": true, "map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true, "var": true,
	}
	return keywords[s]
}

// safeParamName 生成安全的参数名（避免 Go 关键词）
func safeParamName(fieldName string) string {
	paramName := lowerFirst(fieldName)
	if isGoKeyword(paramName) {
		return paramName + "Val"
	}
	return paramName
}

// ImportWithAlias 带别名的 import 信息
type ImportWithAlias struct {
	Path  string
	Alias string
}

// getSetterImports 获取 setter 模式所需的额外 imports
func getSetterImports(model *gormparse.GormModelInfo) []ImportWithAlias {
	// 使用 map 去重，key 是 path，value 是 alias
	imports := make(map[string]string)

	for _, f := range model.Fields {
		// 跳过 patch 字段本身
		if strings.ToLower(f.Name) == "patch" {
			continue
		}
		// 直接使用 PkgPath（已经正确填充）
		if f.PkgPath != "" {
			// 如果已经存在，保留已有的别名（优先使用第一个遇到的别名）
			if _, exists := imports[f.PkgPath]; !exists {
				imports[f.PkgPath] = f.PkgAlias
			}
		}
	}

	result := make([]ImportWithAlias, 0, len(imports))
	for path, alias := range imports {
		result = append(result, ImportWithAlias{Path: path, Alias: alias})
	}
	return result
}

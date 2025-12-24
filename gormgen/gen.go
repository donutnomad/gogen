package gormgen

import (
	"slices"
	"strings"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/internal/gormparse"
)

// generateModelCode 使用 gg 生成单个模型的代码
func generateModelCode(gen *gg.Generator, model *gormparse.GormModelInfo, gsqlPkg, fieldPkg *gg.PackageRef) {
	rawModelName := model.Name
	modelName := rawModelName
	if len(modelName) >= 2 && strings.ToLower(modelName[len(modelName)-2:]) == "po" {
		modelName = modelName[:len(modelName)-2]
	}

	// 处理字段名称冲突
	reservedNames := []string{
		"TableName", "Alias", "WithTable", "As",
		"ModelType", "ModelTypeAny", "AllFields", "Star",
	}
	for idx, f := range model.Fields {
		if slices.Contains(reservedNames, f.Name) {
			f.Name += "T"
		}
		model.Fields[idx] = f
	}

	structName := model.Prefix + modelName + "SchemaType"

	group := gen.Body()

	// 生成结构体定义
	{
		s := group.NewStruct(structName)
		for _, f := range model.Fields {
			fieldType := mapFieldType(f.Type)
			if fieldType == "" {
				continue
			}
			s.AddField(f.Name, fieldType)
		}
		s.AddField("fieldType", rawModelName)
		s.AddField("alias", "string")
		s.AddField("tableName", "string")
	}

	group.AddLine()

	// ====== Method: TableName
	{
		group.NewFunction("TableName").
			WithReceiver("t", structName).
			AddResult("", "string").
			AddBody("return t.tableName")
	}

	group.AddLine()

	// ====== Method: Alias
	{
		group.NewFunction("Alias").
			WithReceiver("t", structName).
			AddResult("", "string").
			AddBody("return t.alias")
	}

	group.AddLine()

	// ====== Method: WithTable
	{
		// 构建 tn := gsqlPkg.TableName(tableName)
		tnDecl := gg.NewInlineGroup().Append(
			gg.S("tn := "),
			gsqlPkg.Call("TableName", "tableName"),
		)

		body := []any{tnDecl}
		for _, f := range model.Fields {
			fieldType := mapFieldType(f.Type)
			if fieldType == "" {
				continue
			}
			body = append(body,
				gg.S("t.%s = t.%s.WithTable(&tn)", f.Name, f.Name),
			)
		}
		group.NewFunction("WithTable").
			WithReceiver("t", "*"+structName).
			AddParameter("tableName", "string").
			AddBody(body...)
	}

	group.AddLine()

	// ====== Method: As
	{
		group.NewFunction("As").
			WithReceiver("t", structName).
			AddParameter("alias", "string").
			AddResult("", structName).
			AddBody(
				"var ret = t",
				"ret.alias = alias",
				"ret.WithTable(alias)",
				"return ret",
			)
	}

	group.AddLine()

	// ====== Method: ModelType
	{
		group.NewFunction("ModelType").
			WithReceiver("t", structName).
			AddResult("", "*"+rawModelName).
			AddBody("return &t.fieldType")
	}

	group.AddLine()

	// ====== Method: ModelTypeAny
	{
		group.NewFunction("ModelTypeAny").
			WithReceiver("t", structName).
			AddResult("", "any").
			AddBody("return &t.fieldType")
	}

	group.AddLine()

	// ====== Method: AllFields
	{
		// 收集所有字段作为切片元素
		var fieldElements []any
		for _, f := range model.Fields {
			fieldType := mapFieldType(f.Type)
			if fieldType == "" {
				continue
			}
			fieldElements = append(fieldElements, gg.S("t.%s", f.Name))
		}

		// 使用 Slice 构建 field.BaseFields{t.Field1, t.Field2, ...}
		sliceLiteral := gg.Value(fieldPkg.Type("BaseFields")).AddElement(fieldElements...).MultiLine()

		group.NewFunction("AllFields").
			WithReceiver("t", structName).
			AddResult("", fieldPkg.Type("BaseFields")).
			AddBody(gg.Return(sliceLiteral))
	}

	group.AddLine()

	// ====== Method: Star
	{
		group.NewFunction("Star").
			WithReceiver("t", structName).
			AddResult("", fieldPkg.Type("IField")).
			AddBody(
				gg.If(`t.alias != ""`).
					AddBody(gg.Return(gsqlPkg.Call("StarWith", "t.alias"))),
				gg.Return(gsqlPkg.Call("StarWith", "t.tableName")),
			)
	}

	group.AddLine()

	// ====== Variable: Schema Instance
	{
		// 构建一个匿名结构体
		anyStruct := gg.Value(structName).
			AddField("tableName", gg.Lit(model.TableName)).MultiLine()
		for _, f := range model.Fields {
			fieldType := mapFieldType(f.Type)
			if fieldType == "" {
				continue
			}
			constructor := getFieldConstructor(fieldType)
			flags := getFieldFlags(f.Tag)
			call := gg.Call(constructor).
				AddParameter(gg.Lit(model.TableName), gg.Lit(f.ColumnName))
			if flags != "" {
				call.AddParameter(gg.S(flags))
			}
			anyStruct.AddField(f.Name, call)
		}
		anyStruct.AddField("fieldType", gg.Value(rawModelName))

		// 声明包级变量
		group.NewVar().AddField(model.Prefix+modelName+"Schema", anyStruct)
	}
}

// ImportWithAlias 带别名的 import 信息
type ImportWithAlias struct {
	Path  string
	Alias string
}

// getGormQueryImports 获取 Query 模式所需的额外 imports
func getGormQueryImports(model *gormparse.GormModelInfo) []ImportWithAlias {
	// 使用 map 去重，key 是 path，value 是 alias
	imports := make(map[string]string)

	for _, f := range model.Fields {
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

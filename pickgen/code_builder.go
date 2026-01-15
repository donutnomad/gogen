package pickgen

import (
	"fmt"
	"strings"

	"github.com/donutnomad/gg"
	"github.com/donutnomad/gogen/internal/structparse"
)

// buildStruct 生成结构体定义
func buildStruct(gen *gg.Generator, targetName, sourceName string, mode SelectionMode, fields []structparse.FieldInfo) {
	group := gen.Body()

	// 生成注释
	modeStr := "Pick"
	if mode == ModeOmit {
		modeStr = "Omit"
	}

	group.AddLine()
	group.Append(gg.LineComment("%s 从 %s %s 生成", targetName, sourceName, modeStr))

	// 创建结构体
	st := gg.Struct(targetName)

	for _, field := range fields {
		// 处理字段类型（可能需要包前缀）
		fieldType := field.Type
		if field.PkgAlias != "" && !strings.Contains(field.Type, ".") {
			fieldType = field.PkgAlias + "." + field.Type
		}

		// 构建带标签的类型字符串
		typeStr := fieldType
		if field.Tag != "" {
			// Tag 已经包含反引号，直接使用
			typeStr = fmt.Sprintf("%s %s", fieldType, field.Tag)
		}

		st.AddField(field.Name, typeStr)
	}

	group.Append(st)
}

// buildFromMethod 生成 From 方法
// func (t *TargetType) From(src *SourceType)
func buildFromMethod(gen *gg.Generator, targetName, sourceType string, fields []structparse.FieldInfo) {
	group := gen.Body()

	group.AddLine()
	group.Append(gg.LineComment("From 从 %s 复制字段值", sourceType))

	fn := group.NewFunction("From").
		WithReceiver("t", "*"+targetName).
		AddParameter("src", "*"+sourceType)

	for _, field := range fields {
		fn.AddBody(gg.S("t.%s = src.%s", field.Name, field.Name))
	}
}

// buildNewFunction 生成构造函数
// func NewTargetType(src *SourceType) TargetType
func buildNewFunction(gen *gg.Generator, targetName, sourceType string, fields []structparse.FieldInfo) {
	group := gen.Body()

	group.AddLine()
	group.Append(gg.LineComment("New%s 从 %s 创建 %s", targetName, sourceType, targetName))

	group.NewFunction("New"+targetName).
		AddParameter("src", "*"+sourceType).
		AddResult("", targetName).
		AddBody(
			gg.S("var result %s", targetName),
			gg.S("result.From(src)"),
			gg.Return(gg.S("result")),
		)
}

package generator

import (
	"strings"
	"unicode"
)

// Pluralize 将英文单词转换为复数形式
// 支持驼峰命名的组合词，只复数化最后一个单词
// 例如: CompanyName -> CompanyNames, UserID -> UserIDs
func Pluralize(word string) string {
	if word == "" {
		return word
	}

	// 拆分驼峰命名的组合词
	parts := splitCamelCase(word)
	if len(parts) == 0 {
		return word
	}

	// 只复数化最后一个单词
	lastPart := parts[len(parts)-1]
	pluralizedLast := pluralizeSingleWord(lastPart)

	// 重新组合
	if len(parts) == 1 {
		return pluralizedLast
	}
	return strings.Join(parts[:len(parts)-1], "") + pluralizedLast
}

// splitCamelCase 拆分驼峰命名
// 例如: "CompanyName" -> ["Company", "Name"]
// 特殊处理缩写词: "UserID" -> ["User", "ID"]
func splitCamelCase(s string) []string {
	if s == "" {
		return nil
	}

	var parts []string
	var current strings.Builder

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if i == 0 {
			current.WriteRune(r)
			continue
		}

		// 检测单词边界
		if unicode.IsUpper(r) {
			// 检查是否是缩写词的一部分 (连续大写字母)
			prevIsUpper := unicode.IsUpper(runes[i-1])
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])

			// 如果前一个是大写，当前是大写，下一个是小写，说明缩写词结束
			// 例如: "UserID" 中的 'I' 和 "HTMLParser" 中的 'P'
			if prevIsUpper && nextIsLower {
				// 将前面累积的缩写词保存
				str := current.String()
				if len(str) > 1 {
					parts = append(parts, str[:len(str)-1])
					current.Reset()
					current.WriteRune(runes[i-1])
				}
			} else if !prevIsUpper {
				// 普通驼峰边界
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			}
		}
		current.WriteRune(r)
	}

	// 添加最后一部分
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// pluralizeSingleWord 对单个单词进行复数化
func pluralizeSingleWord(word string) string {
	if word == "" {
		return word
	}

	// 保留原始大小写信息
	lower := strings.ToLower(word)

	// 检查不规则复数
	if plural, ok := irregularPlurals[lower]; ok {
		return matchCase(word, plural)
	}

	// 检查不可数名词（复数形式与单数相同）
	if uncountableNouns[lower] {
		return word
	}

	// 检查已经是复数形式
	if isAlreadyPlural(lower) {
		return word
	}

	// 应用规则并匹配大小写
	plural := applyPluralRules(lower)
	return matchCase(word, plural)
}

// irregularPlurals 不规则复数映射表
var irregularPlurals = map[string]string{
	// 人相关
	"person": "people",
	"child":  "children",
	"man":    "men",
	"woman":  "women",
	"foot":   "feet",
	"tooth":  "teeth",
	"goose":  "geese",
	"mouse":  "mice",
	"louse":  "lice",
	"ox":     "oxen",

	// 动物
	"fish":   "fish",
	"sheep":  "sheep",
	"deer":   "deer",
	"moose":  "moose",
	"swine":  "swine",
	"bison":  "bison",
	"salmon": "salmon",
	"trout":  "trout",
	"shrimp": "shrimp",

	// 拉丁/希腊词源
	"datum":       "data",
	"medium":      "media",
	"bacterium":   "bacteria",
	"curriculum":  "curricula",
	"memorandum":  "memoranda",
	"criterion":   "criteria",
	"phenomenon":  "phenomena",
	"analysis":    "analyses",
	"basis":       "bases",
	"crisis":      "crises",
	"diagnosis":   "diagnoses",
	"hypothesis":  "hypotheses",
	"oasis":       "oases",
	"parenthesis": "parentheses",
	"synopsis":    "synopses",
	"thesis":      "theses",
	"appendix":    "appendices",
	"index":       "indices",
	"matrix":      "matrices",
	"vertex":      "vertices",
	"apex":        "apices",
	"focus":       "foci",
	"nucleus":     "nuclei",
	"radius":      "radii",
	"stimulus":    "stimuli",
	"cactus":      "cacti",
	"fungus":      "fungi",
	"alumnus":     "alumni",
	"syllabus":    "syllabi",

	// 编程常用
	"alias":  "aliases",
	"status": "statuses",
	"quiz":   "quizzes",
	"bus":    "buses",
}

// uncountableNouns 不可数名词（复数形式与单数相同）
var uncountableNouns = map[string]bool{
	"equipment":   true,
	"information": true,
	"rice":        true,
	"money":       true,
	"species":     true,
	"series":      true,
	"news":        true,
	"sheep":       true,
	"deer":        true,
	"fish":        true,
	"means":       true,
	"offspring":   true,
	"data":        true,
	"metadata":    true,
	"config":      true,
	"settings":    true,
	"contents":    true,
}

// isAlreadyPlural 检查单词是否已经是复数形式
func isAlreadyPlural(word string) bool {
	// 常见复数后缀检查
	pluralSuffixes := []string{"ies", "ves", "oes", "ses", "xes", "zes", "ches", "shes"}
	for _, suffix := range pluralSuffixes {
		if strings.HasSuffix(word, suffix) && len(word) > len(suffix) {
			return true
		}
	}

	// 检查是否是不规则复数的结果
	for _, plural := range irregularPlurals {
		if word == plural {
			return true
		}
	}

	return false
}

// applyPluralRules 应用复数规则
func applyPluralRules(word string) string {
	if len(word) == 0 {
		return word
	}

	lower := strings.ToLower(word)
	lastChar := lower[len(lower)-1]
	secondLastChar := byte(0)
	if len(lower) > 1 {
		secondLastChar = lower[len(lower)-2]
	}

	// 规则 1: 以 s, x, z, ch, sh 结尾 -> 加 es
	if lastChar == 's' || lastChar == 'x' || lastChar == 'z' {
		return word + "es"
	}
	if len(lower) >= 2 {
		suffix := lower[len(lower)-2:]
		if suffix == "ch" || suffix == "sh" {
			return word + "es"
		}
	}

	// 规则 2: 以辅音 + y 结尾 -> 变 y 为 ies
	if lastChar == 'y' && !isVowel(secondLastChar) {
		return word[:len(word)-1] + "ies"
	}

	// 规则 3: 以 o 结尾
	// 辅音 + o -> 加 es（有例外）
	// 元音 + o -> 加 s
	if lastChar == 'o' {
		// 一些常见的加 es 的词
		oEsWords := map[string]bool{
			"hero": true, "potato": true, "tomato": true, "echo": true,
			"veto": true, "torpedo": true, "embargo": true,
		}
		if oEsWords[lower] {
			return word + "es"
		}
		// 其他情况加 s
		return word + "s"
	}

	// 规则 4: 以 f 或 fe 结尾 -> 变为 ves
	if lastChar == 'f' {
		// 特殊情况：有些词直接加 s
		fSWords := map[string]bool{
			"roof": true, "chief": true, "belief": true, "proof": true,
			"cliff": true, "brief": true, "chef": true, "safe": true,
		}
		if fSWords[lower] {
			return word + "s"
		}
		return word[:len(word)-1] + "ves"
	}
	if len(lower) >= 2 && lower[len(lower)-2:] == "fe" {
		return word[:len(word)-2] + "ves"
	}

	// 规则 5: 以 is 结尾 -> 变为 es（拉丁词源）
	if len(lower) >= 2 && lower[len(lower)-2:] == "is" {
		return word[:len(word)-2] + "es"
	}

	// 规则 6: 以 us 结尾 -> 变为 i（拉丁词源，但很多现代词直接加 es）
	// 这里保守处理，默认加 es
	if len(lower) >= 2 && lower[len(lower)-2:] == "us" {
		return word + "es"
	}

	// 默认规则: 加 s
	return word + "s"
}

// isVowel 检查是否是元音字母
func isVowel(c byte) bool {
	return c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u'
}

// matchCase 匹配原始单词的大小写模式
func matchCase(original, result string) string {
	if len(original) == 0 || len(result) == 0 {
		return result
	}

	// 全大写（缩写词如 ID, URL）
	// 复数形式应为 IDs, URLs（保持原词大写，只加小写 s）
	if strings.ToUpper(original) == original {
		// 检查结果是否只是在原词基础上加了后缀
		lowerOriginal := strings.ToLower(original)
		lowerResult := strings.ToLower(result)
		if strings.HasPrefix(lowerResult, lowerOriginal) {
			suffix := result[len(original):]
			return original + suffix
		}
		// 不规则复数，全部转大写
		return strings.ToUpper(result)
	}

	// 首字母大写
	if unicode.IsUpper(rune(original[0])) {
		runes := []rune(result)
		runes[0] = unicode.ToUpper(runes[0])
		return string(runes)
	}

	// 全小写
	return result
}

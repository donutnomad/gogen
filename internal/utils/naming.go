package utils

import "strings"

// commonInitialisms 常见首字母缩略词列表，与 GORM 保持一致
var commonInitialisms = []string{
	"API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS",
	"ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP",
	"SSH", "TLS", "TTL", "UID", "UI", "UUID", "URI", "URL", "UTF8", "VM",
	"XML", "XSRF", "XSS",
}

// commonInitialismsReplacer 用于将缩略词转换为首字母大写形式
var commonInitialismsReplacer *strings.Replacer

func init() {
	replacerArgs := make([]string, 0, len(commonInitialisms)*2)
	for _, initialism := range commonInitialisms {
		// API -> Api, HTTP -> Http
		replacerArgs = append(replacerArgs, initialism, toTitleCase(initialism))
	}
	commonInitialismsReplacer = strings.NewReplacer(replacerArgs...)
}

// toTitleCase 将字符串转换为首字母大写形式
func toTitleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// ToSnakeCase 将驼峰命名转换为蛇形(下划线)命名，与 GORM 的 toDBName 保持一致
// 参考: gorm/schema/naming.go:131-188
func ToSnakeCase(name string) string {
	if name == "" {
		return ""
	}

	// 首字母缩略词处理: API -> Api, HTTP -> Http
	value := commonInitialismsReplacer.Replace(name)

	var (
		buf                            strings.Builder
		lastCase, nextCase, nextNumber bool
		curCase                        = value[0] <= 'Z' && value[0] >= 'A'
	)

	for i, v := range value[:len(value)-1] {
		nextCase = value[i+1] <= 'Z' && value[i+1] >= 'A'
		nextNumber = value[i+1] >= '0' && value[i+1] <= '9'

		if curCase {
			if lastCase && (nextCase || nextNumber) {
				buf.WriteRune(v + 32) // 转小写
			} else {
				if i > 0 && value[i-1] != '_' && value[i+1] != '_' {
					buf.WriteByte('_') // 插入下划线
				}
				buf.WriteRune(v + 32) // 转小写
			}
		} else {
			buf.WriteRune(v)
		}

		lastCase = curCase
		curCase = nextCase
	}

	// 处理最后一个字符
	if curCase {
		if !lastCase && len(value) > 1 {
			buf.WriteByte('_')
		}
		buf.WriteByte(value[len(value)-1] + 32)
	} else {
		buf.WriteByte(value[len(value)-1])
	}

	return buf.String()
}

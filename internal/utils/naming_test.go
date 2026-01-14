package utils

import (
	"testing"
)

// TestToSnakeCase 测试 ToSnakeCase 函数，参考 GORM 的测试用例
// 参考: gorm/schema/naming_test.go
func TestToSnakeCase(t *testing.T) {
	tests := map[string]string{
		"":                          "",
		"x":                         "x",
		"X":                         "x",
		"userRestrictions":          "user_restrictions",
		"ThisIsATest":               "this_is_a_test",
		"PFAndESI":                  "pf_and_esi",
		"AbcAndJkl":                 "abc_and_jkl",
		"EmployeeID":                "employee_id",
		"SKU_ID":                    "sku_id",
		"FieldX":                    "field_x",
		"HTTPAndSMTP":               "http_and_smtp",
		"HTTPServerHandlerForURLID": "http_server_handler_for_url_id",
		"UUID":                      "uuid",
		"HTTPURL":                   "http_url",
		"HTTP_URL":                  "http_url",
		"SHA256Hash":                "sha256_hash",
		"SHA256HASH":                "sha256_hash",
		"UserID":                    "user_id",
		"APIKey":                    "api_key",
		"HTTPRequest":               "http_request",
		"XMLParser":                 "xml_parser",
		"JSONData":                  "json_data",
		"IPAddress":                 "ip_address",
		"URLPath":                   "url_path",
		"SSHKey":                    "ssh_key",
		"TLSConfig":                 "tls_config",
		"CPUUsage":                  "cpu_usage",
		"RAMSize":                   "ram_size",
		"HappyBodyIDs":              "happy_body_ids",
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			result := ToSnakeCase(input)
			if result != expected {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", input, result, expected)
			}
		})
	}
}

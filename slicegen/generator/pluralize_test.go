package generator

import (
	"testing"
)

func TestPluralize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// 基本规则：加 s
		{"Name", "Names"},
		{"Email", "Emails"},
		{"User", "Users"},
		{"Product", "Products"},
		{"Order", "Orders"},
		{"Item", "Items"},
		{"Price", "Prices"},
		{"Value", "Values"},

		// 以 s, x, z 结尾：加 es
		{"Class", "Classes"},
		{"Status", "Statuses"},
		{"Box", "Boxes"},
		{"Index", "Indices"}, // 不规则
		{"Quiz", "Quizzes"},  // 不规则

		// 以 ch, sh 结尾：加 es
		{"Match", "Matches"},
		{"Batch", "Batches"},
		{"Branch", "Branches"},
		{"Flash", "Flashes"},
		{"Brush", "Brushes"},

		// 以辅音 + y 结尾：变 y 为 ies
		{"Category", "Categories"},
		{"Country", "Countries"},
		{"Company", "Companies"},
		{"City", "Cities"},
		{"Body", "Bodies"},
		{"Story", "Stories"},
		{"Query", "Queries"},
		{"Entry", "Entries"},
		{"Reply", "Replies"},
		{"Policy", "Policies"},

		// 以元音 + y 结尾：加 s
		{"Key", "Keys"},
		{"Day", "Days"},
		{"Boy", "Boys"},
		{"Toy", "Toys"},
		{"Way", "Ways"},
		{"Array", "Arrays"},
		{"Survey", "Surveys"},

		// 以 o 结尾
		{"Photo", "Photos"},
		{"Video", "Videos"},
		{"Radio", "Radios"},
		{"Hero", "Heroes"},
		{"Potato", "Potatoes"},
		{"Tomato", "Tomatoes"},

		// 以 f/fe 结尾
		{"Leaf", "Leaves"},
		{"Life", "Lives"},
		{"Wife", "Wives"},
		{"Knife", "Knives"},
		{"Half", "Halves"},
		{"Self", "Selves"},
		{"Shelf", "Shelves"},
		// 例外
		{"Roof", "Roofs"},
		{"Chief", "Chiefs"},
		{"Belief", "Beliefs"},

		// 不规则复数
		{"Person", "People"},
		{"Child", "Children"},
		{"Man", "Men"},
		{"Woman", "Women"},
		{"Foot", "Feet"},
		{"Tooth", "Teeth"},
		{"Mouse", "Mice"},
		{"Goose", "Geese"},

		// 拉丁/希腊词源
		{"Datum", "Data"},
		{"Medium", "Media"},
		{"Analysis", "Analyses"},
		{"Basis", "Bases"},
		{"Crisis", "Crises"},
		{"Criterion", "Criteria"},
		{"Phenomenon", "Phenomena"},

		// 不可数/相同形式
		{"Fish", "Fish"},
		{"Sheep", "Sheep"},
		{"Deer", "Deer"},
		{"Species", "Species"},
		{"Series", "Series"},
		{"Data", "Data"},
		{"Metadata", "Metadata"},

		// 编程常用
		{"ID", "IDs"},
		{"URL", "URLs"},
		{"Config", "Config"},
		{"Settings", "Settings"},
		{"Alias", "Aliases"},

		// 组合词（驼峰命名）- 只复数化最后一个单词
		{"CompanyName", "CompanyNames"},
		{"UserID", "UserIDs"},
		{"OrderStatus", "OrderStatuses"},
		{"ProductCategory", "ProductCategories"},
		{"FirstName", "FirstNames"},
		{"LastName", "LastNames"},
		{"CreatedAt", "CreatedAts"},
		{"UpdatedAt", "UpdatedAts"},
		{"EmailAddress", "EmailAddresses"},
		{"PhoneNumber", "PhoneNumbers"},

		// 组合词中包含不规则复数
		{"ActivePerson", "ActivePeople"},
		{"OldChild", "OldChildren"},
		{"RedFish", "RedFish"},
		{"BaseAnalysis", "BaseAnalyses"},

		// 小写
		{"name", "names"},
		{"category", "categories"},
		{"child", "children"},

		// 空字符串
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Pluralize(tt.input)
			if result != tt.expected {
				t.Errorf("Pluralize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPluralizePreservesCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// 首字母大写（Go 标准命名）
		{"Category", "Categories"},
		{"Person", "People"},
		{"Child", "Children"},
		// 小写
		{"category", "categories"},
		{"person", "people"},
		// 缩写词（保持大写，加小写 s）
		{"ID", "IDs"},
		{"URL", "URLs"},
		{"API", "APIs"},
		// 组合词中的缩写
		{"UserID", "UserIDs"},
		{"OrderURL", "OrderURLs"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Pluralize(tt.input)
			if result != tt.expected {
				t.Errorf("Pluralize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplitCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"Name", []string{"Name"}},
		{"CompanyName", []string{"Company", "Name"}},
		{"UserID", []string{"User", "ID"}},
		{"OrderURL", []string{"Order", "URL"}},
		{"firstName", []string{"first", "Name"}},
		{"ID", []string{"ID"}},
		{"URL", []string{"URL"}},
		{"ABC", []string{"ABC"}},
		{"ProductCategory", []string{"Product", "Category"}},
		{"EmailAddress", []string{"Email", "Address"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitCamelCase(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitCamelCase(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splitCamelCase(%q) = %v, want %v", tt.input, result, tt.expected)
					return
				}
			}
		})
	}
}

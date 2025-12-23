package embedded

// User 带嵌入字段的用户结构体
type User struct {
	BaseModel        // 匿名嵌入
	Name      string `json:"name"`
	Email     string `json:"email"`
	Age       int    `json:"age"`
}

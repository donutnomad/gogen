package example

import "github.com/gin-gonic/gin"

// onGinResponse 响应处理辅助函数（需用户实现）
func onGinResponse[T any](c *gin.Context, data T, err error) {
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, data)
}

// onGinBind 绑定辅助函数（需用户实现）
func onGinBind(c *gin.Context, val any, typ string) bool {
	var err error
	switch typ {
	case "JSON":
		err = c.ShouldBindJSON(val)
	case "FORM":
		err = c.ShouldBind(val)
	case "QUERY":
		err = c.ShouldBindQuery(val)
	default:
		err = c.ShouldBind(val)
	}
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return false
	}
	return true
}

package response

import (
	"github.com/gin-gonic/gin"
)

// RespondWithError 响应错误处理
func RespondWithError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"error": message,
		"code":  code,
	})
}

// RespondWithJSON 响应JSON数据
func RespondWithJSON(c *gin.Context, code int, payload interface{}) {
	c.JSON(code, payload)
}
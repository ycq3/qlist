package middleware

import (
	"github.com/gin-gonic/gin"
)

// SEOMiddleware 添加SEO友好的HTTP头部
func SEOMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 添加X-Robots-Tag头部，允许搜索引擎索引
		c.Header("X-Robots-Tag", "index, follow")

		// 添加X-Content-Type-Options头部
		c.Header("X-Content-Type-Options", "nosniff")

		// 添加X-Frame-Options头部
		c.Header("X-Frame-Options", "SAMEORIGIN")

		// 添加Referrer-Policy头部
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 添加Feature-Policy头部
		c.Header("Feature-Policy", "camera 'none'; microphone 'none'; geolocation 'none'")

		// 继续处理请求
		c.Next()
	}
}

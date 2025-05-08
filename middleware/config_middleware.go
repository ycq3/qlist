package middleware

import (
	"net/http"
	"os"
)

// ConfigMiddleware 配置中间件结构体
type ConfigMiddleware struct{}

// CheckConfig 检查配置文件状态的中间件
func (m *ConfigMiddleware) CheckConfig(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := os.Stat("config.json")
		if os.IsNotExist(err) {
			// 配置文件不存在，允许访问配置页面
			next(w, r)
			return
		}
		// 配置文件存在，重定向到管理后台
		http.Redirect(w, r, "/dist/admin.html", http.StatusMovedPermanently)
	}
}

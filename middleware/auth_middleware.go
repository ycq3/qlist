package middleware

import (
	"net/http"
	"qlist/config"
	"strings"
)

// AuthMiddleware 认证中间件结构体
type AuthMiddleware struct{}

// BasicAuth 实现Basic认证的中间件
func (m *AuthMiddleware) BasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 检查管理账号配置是否存在
		if config.Instance.Username == "" || config.Instance.Password == "" {
			http.Error(w, "管理员账号未配置", http.StatusServiceUnavailable)
			return
		}

		// 进行Basic认证
		user, pass, ok := r.BasicAuth()
		if !ok || user != config.Instance.Username || pass != config.Instance.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="请输入管理员账号密码"`)
			http.Error(w, "未授权访问", http.StatusUnauthorized)
			return
		}

		// 认证通过，继续处理请求
		next(w, r)
	}
}

// APIKeyAuth 实现API Key认证的中间件
func (m *AuthMiddleware) APIKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从请求头中获取API Key
		apiKey := r.Header.Get("X-API-Key")

		// 如果请求头中没有API Key，尝试从URL参数中获取
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		// 检查API Key是否有效（这里使用管理员密码作为API Key）
		if apiKey == "" || apiKey != config.Instance.Password {
			http.Error(w, "无效的API Key", http.StatusUnauthorized)
			return
		}

		// 认证通过，继续处理请求
		next(w, r)
	}
}

// RequireAuth 根据请求类型选择合适的认证方式
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 检查请求头中是否包含API Key
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" || strings.Contains(r.URL.RawQuery, "api_key=") {
			// 使用API Key认证
			m.APIKeyAuth(next)(w, r)
			return
		}

		// 默认使用Basic认证
		m.BasicAuth(next)(w, r)
	}
}

package handlers

import (
	"net/http"
	"path"
	"qlist/config"
	"qlist/public"
)

// StaticHandler 处理静态文件请求
type StaticHandler struct{}

// ServeHTTP 实现http.Handler接口
func (h *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 清理和标准化请求路径
	urlPath := path.Clean(r.URL.Path)

	// 添加SEO友好的头部
	w.Header().Set("X-Robots-Tag", "index, follow")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// 处理特殊路由
	switch urlPath {
	case "/":
		// 重定向到首页
		http.Redirect(w, r, "/dist/index.html", http.StatusMovedPermanently)
		return
	case "/admin":
		// 重定向到admin.html
		http.Redirect(w, r, "/dist/admin.html", http.StatusMovedPermanently)
		return
	case "/save-config":
		// 使用 ConfigHandler 处理配置保存
		(&ConfigHandler{}).SaveConfig(w, r)
		return
	case "/check-config":
		// 使用 ConfigHandler 处理配置检查
		(&ConfigHandler{}).CheckConfigExists(w, r)
		return
	}

	// 对admin.html进行认证检查
	if urlPath == "/dist/admin.html" {
		// 检查管理账号配置是否存在
		if config.Instance.Username == "" || config.Instance.Password == "" {
			// 重定向到配置页面
			http.Redirect(w, r, "/dist/config.html", http.StatusTemporaryRedirect)
			return
		}

		// 进行Basic认证
		user, pass, ok := r.BasicAuth()
		if !ok || user != config.Instance.Username || pass != config.Instance.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="请输入管理员账号密码"`)
			http.Error(w, "未授权访问", http.StatusUnauthorized)
			return
		}
	}

	// 对于config.html的特殊处理
	if urlPath == "/config.html" {
		// 如果配置文件已存在，重定向到管理页面
		if err := config.LoadConfig("config.json"); err == nil {
			http.Redirect(w, r, "/admin", http.StatusMovedPermanently)
			return
		}
	}

	// 使用http.FileServer处理其他静态文件
	http.FileServer(http.FS(public.Public)).ServeHTTP(w, r)
}

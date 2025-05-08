package handlers

import (
	"encoding/json"
	"net/http"
	"os"
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

	// 处理特殊路由
	switch urlPath {
	case "/admin":
		// 重定向到admin.html
		http.Redirect(w, r, "/dist/admin.html", http.StatusMovedPermanently)
		return
	case "/save-config":
		if r.Method != http.MethodPost {
			http.Error(w, "无效的请求方法", http.StatusMethodNotAllowed)
			return
		}
		// 检查 config.json 文件是否存在
		if _, err := os.Stat("config.json"); err == nil {
			// 文件存在，拒绝修改
			http.Error(w, "配置文件已存在，不允许修改", http.StatusForbidden)
			return
		}
		var newConfig config.AppConfig
		err := json.NewDecoder(r.Body).Decode(&newConfig)
		if err != nil {
			http.Error(w, "解析请求体失败", http.StatusBadRequest)
			return
		}
		config.Instance = newConfig
		err = config.SaveConfig("config.json")
		if err != nil {
			http.Error(w, "保存配置文件失败", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	case "/check-config":
		var status struct {
			Exists   bool   `json:"exists"`
			Redirect string `json:"redirect"`
		}
		if _, err := os.Stat("config.json"); os.IsNotExist(err) {
			status = struct {
				Exists   bool   `json:"exists"`
				Redirect string `json:"redirect"`
			}{Exists: false, Redirect: "/config.html"}
		} else {
			status = struct {
				Exists   bool   `json:"exists"`
				Redirect string `json:"redirect"`
			}{Exists: true, Redirect: "/admin"}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(status)
		return
	}

	// 对admin.html进行认证检查
	if urlPath == "/admin.html" {
		// 检查管理账号配置是否存在
		if config.Instance.Username == "" || config.Instance.Password == "" {
			http.Error(w, "管理账号未配置", http.StatusForbidden)
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

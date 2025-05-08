package handlers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"qlist/config"
	"qlist/public"
)

// ConfigHandler 处理配置相关的请求
type ConfigHandler struct{}

// ConfigStatus 配置状态响应结构
type ConfigStatus struct {
	Exists   bool   `json:"exists"`
	Redirect string `json:"redirect"`
}

// ServeConfigPage 处理配置页面请求
func (h *ConfigHandler) ServeConfigPage(w http.ResponseWriter, r *http.Request) {
	// 设置响应头为HTML格式
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// 从embed文件系统读取配置页面
	file, err := public.Public.Open("dist/config.html")
	if err != nil {
		log.Printf("读取配置页面失败: %v", err)
		http.Error(w, "配置页面加载失败，请检查系统配置", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		log.Printf("读取配置页面内容失败: %v", err)
		http.Error(w, "配置页面加载失败，请检查系统配置", http.StatusInternalServerError)
		return
	}

	// 直接写入响应
	w.Write(content)
}

// CheckConfigExists 检查配置文件存在状态
func (h *ConfigHandler) CheckConfigExists(w http.ResponseWriter, r *http.Request) {
	var status ConfigStatus
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		status = ConfigStatus{Exists: false, Redirect: "/config.html"}
	} else {
		status = ConfigStatus{Exists: true, Redirect: "/admin"}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// SaveConfig 保存配置
func (h *ConfigHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
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
}

// ServeAdminPage 处理管理后台页面请求
func (h *ConfigHandler) ServeAdminPage(w http.ResponseWriter, r *http.Request) {
	// 检查管理账号配置是否存在
	if config.Instance.Username == "" || config.Instance.Password == "" {
		http.Error(w, "管理账号未配置，请先设置配置文件", http.StatusInternalServerError)
		return
	}

	// 简单的身份验证
	if r.Header.Get("Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte(config.Instance.Username+":"+config.Instance.Password)) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "未授权访问", http.StatusUnauthorized)
		return
	}

	// 设置响应头为 HTML 格式
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// 从embed文件系统读取admin.html
	file, err := public.Public.Open("dist/admin.html")
	if err != nil {
		http.Error(w, "管理后台页面不存在", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	fileContent, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "读取管理后台页面失败", http.StatusInternalServerError)
		return
	}

	w.Write(fileContent)
}

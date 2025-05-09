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
	"time"
)

// ConfigHandler 处理配置相关的请求
type ConfigHandler struct{}

// ConfigStatus 配置状态响应结构
type ConfigStatus struct {
	Exists   bool   `json:"exists"`
	Redirect string `json:"redirect"`
}

// APIKeyResponse API Key 响应结构体
type APIKeyResponse struct {
	APIKey string `json:"api_key"`
}

// GenerateAPIKey 生成新的 API Key 并保存到配置
// @Summary 生成新的 API Key
// @Description 管理员生成新的 API Key 并保存到配置文件，返回生成的 API Key
// @Tags 配置管理
// @Accept json
// @Produce json
// @Success 200 {object} APIKeyResponse "生成成功，返回新的 API Key"
// @Failure 503 {string} string "管理员账号未配置"
// @Failure 405 {string} string "无效的请求方法"
// @Failure 500 {string} string "保存API Key失败"
// @Router /generateApiKey [post]
func (h *ConfigHandler) GenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	if config.Instance.Username == "" || config.Instance.Password == "" {
		http.Error(w, "管理员账号未配置", http.StatusServiceUnavailable)
		return
	}
	// 仅允许 POST
	if r.Method != http.MethodPost {
		http.Error(w, "无效的请求方法", http.StatusMethodNotAllowed)
		return
	}
	// 生成随机 API Key（此处用时间戳+用户名简单实现，可替换为更安全的生成方式）
	apiKey := base64.StdEncoding.EncodeToString([]byte(config.Instance.Username + time.Now().Format("20060102150405")))
	config.Instance.APIKey = apiKey
	err := config.SaveConfig("config.json")
	if err != nil {
		http.Error(w, "保存API Key失败", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIKeyResponse{APIKey: apiKey})
}

// ServeConfigPage 处理配置页面请求
// @Summary 获取配置页面
// @Description 返回系统配置页面的 HTML 内容
// @Tags 配置管理
// @Produce html
// @Success 200 {string} string "配置页面 HTML"
// @Failure 500 {string} string "配置页面加载失败"
// @Router /config.html [get]
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
// @Summary 检查配置文件是否存在
// @Description 检查 config.json 是否存在，返回存在状态和跳转路径
// @Tags 配置管理
// @Produce json
// @Success 200 {object} ConfigStatus "配置状态"
// @Router /api/checkConfig [get]
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
// @Summary 保存系统配置
// @Description 保存系统配置到 config.json，仅允许首次设置
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param config body config.AppConfig true "系统配置信息"
// @Success 200 {object} map[string]string "保存成功"
// @Failure 405 {string} string "无效的请求方法"
// @Failure 403 {string} string "配置文件已存在，不允许修改"
// @Failure 400 {string} string "解析请求体失败"
// @Failure 500 {string} string "保存配置文件失败"
// @Router /api/saveConfig [post]
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

	// 自动生成 API Key（以用户名+时间戳为基础，建议后续替换为更安全的生成方式）
	apiKey := base64.StdEncoding.EncodeToString([]byte(config.Instance.Username + time.Now().Format("20060102150405")))
	config.Instance.APIKey = apiKey
	// 自动生成 JWTSecret
	if config.Instance.JWTSecret == "" {
		config.Instance.JWTSecret = config.GenerateRandomSecret(32)
	}
	err = config.SaveConfig("config.json")
	if err != nil {
		http.Error(w, "保存配置文件失败", http.StatusInternalServerError)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// 返回成功消息
	json.NewEncoder(w).Encode(map[string]string{"message": "配置保存成功，服务器将在3秒后重启"})

	// 异步重启服务器
	go func() {
		// 等待3秒，确保响应已发送
		time.Sleep(3 * time.Second)
		// 退出程序，依赖系统服务管理器（如 systemd）重启服务
		os.Exit(0)
	}()
}

// GetApiKey 获取当前 API Key
// @Summary 获取当前 API Key
// @Description 获取当前配置文件中的 API Key，仅管理员可见
// @Tags 配置管理
// @Produce json
// @Success 200 {object} APIKeyResponse
// @Router /api/getApiKey [get]
func (h *ConfigHandler) GetApiKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIKeyResponse{APIKey: config.Instance.APIKey})
}

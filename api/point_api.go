package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"qlist/config"
	"qlist/models"
	"qlist/storage"
	"strconv"
	"strings"

	"time"

	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

var db *gorm.DB

// @title 积分管理系统API
// @version 1.0
// @description 提供用户积分管理、积分配置、积分日志查询和文件下载等功能
// @BasePath /api

func InitDB() error {
	var err error
	dialector, err := config.GetDialector()
	if err != nil {
		return fmt.Errorf("请先完成数据库配置: %w", err)
	}

	db, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 自动迁移数据库结构
	db.AutoMigrate(&models.User{}, &models.PointConfig{}, &models.PointLog{})
	return nil
}

// 响应错误处理
func respondWithError(w http.ResponseWriter, code int, message string) {
	if code == http.StatusServiceUnavailable {
		w.Header().Set("Location", "/config.html")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
		"code":  code,
	})
}

// 响应JSON数据
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// @Summary 获取积分配置列表
// @Description 获取所有文件的积分配置
// @Tags 积分配置
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /getPointsList [get]
func GetPointsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var configs []models.PointConfig
	if result := db.Find(&configs); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "获取积分配置列表失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code": http.StatusOK,
		"data": configs,
	})
}

// @Summary 配置文件积分
// @Description 为文件设置积分值
// @Tags 积分配置
// @Accept json
// @Produce json
// @Param config body models.PointConfig true "积分配置信息"
// @Success 200 {object} map[string]interface{}
// @Router /configurePoints [post]
func ConfigurePoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var config models.PointConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondWithError(w, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 去除文件链接中的空格
	config.FileUrl = strings.TrimSpace(config.FileUrl)

	// 确保文件链接不以'/'开头
	if strings.HasPrefix(config.FileUrl, "/") {
		config.FileUrl = strings.TrimPrefix(config.FileUrl, "/")
	}

	// 查找是否已存在配置
	var existingConfig models.PointConfig
	result := db.Where("file_url = ?", config.FileUrl).First(&existingConfig)

	if result.Error == nil {
		// 更新现有配置
		existingConfig.Points = config.Points
		existingConfig.Description = config.Description
		if result := db.Save(&existingConfig); result.Error != nil {
			respondWithError(w, http.StatusInternalServerError, "更新积分配置失败")
			return
		}
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"code":    http.StatusOK,
			"message": "配置更新成功",
		})
		return
	}

	// 创建新配置
	if result := db.Create(&config); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "保存积分配置失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code":    http.StatusOK,
		"message": "配置保存成功",
	})
}

// @Summary 获取用户列表
// @Description 获取所有用户及其积分信息
// @Tags 用户管理
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /getUsersList [get]
func GetUsersList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var users []models.User
	if result := db.Find(&users); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "获取用户列表失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code":  http.StatusOK,
		"users": users,
	})
}

func DownloadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var data struct {
		FileUrl string `json:"fileUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, "请求参数错误")
		return
	}
	// 去除文件链接中的空格
	data.FileUrl = strings.TrimSpace(data.FileUrl)
	// 确保文件链接不以'/'开头
	if strings.HasPrefix(data.FileUrl, "/") {
		data.FileUrl = strings.TrimPrefix(data.FileUrl, "/")
	}
	// 检查用户是否已登录
	userId, ok := requireLogin(w, r)
	if !ok {
		return
	}
	// 查找文件积分配置
	var configModel models.PointConfig
	if result := db.Where("file_url = ?", data.FileUrl).First(&configModel); result.Error != nil {
		// 使用默认积分配置
		configModel = models.PointConfig{
			FileUrl:     data.FileUrl,
			Points:      config.Instance.DefaultPoints,
			Description: "无",
		}
	}
	// 查找或创建用户
	var user models.User
	if result := db.Where("id = ?", userId).First(&user); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "用户操作失败")
		return
	}

	// 检查用户积分是否足够
	if user.Points < configModel.Points {
		respondWithError(w, http.StatusForbidden, "积分不足")
		return
	}

	// 开始事务
	tx := db.Begin()

	// 扣除用户积分
	user.Points -= configModel.Points
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "扣除积分失败")
		return
	}

	// 创建积分变更日志
	log := models.PointLog{
		UserID:      user.ID,
		Points:      -configModel.Points,
		Type:        "file_access",
		Description: fmt.Sprintf("下载文件：%s", data.FileUrl),
		FileUrl:     data.FileUrl,
	}

	if err := tx.Create(&log).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "创建积分日志失败")
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "操作失败")
		return
	}

	// 获取文件下载地址
	uploader := &storage.AlistUploader{}
	downloadUrl, err := uploader.GetDownloadUrl(data.FileUrl)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "获取下载地址失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code":        http.StatusOK,
		"downloadUrl": downloadUrl,
	})
}

// 生成JWT Token（通用函数）
func generateJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(), // 7天有效期
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Instance.JWTSecret))
}

// 解析JWT Token（通用函数）
func parseJWT(tokenStr string) (uint, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Instance.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("token无效")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("token解析失败")
	}
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("token缺少user_id")
	}
	return uint(userID), nil
}

// 修改requireLogin，支持JWT校验
func requireLogin(w http.ResponseWriter, r *http.Request) (uint, bool) {
	cookie, err := r.Cookie("qlist_token")
	if err != nil || cookie.Value == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":         "未登录，请先登录",
			"login_options": getAvailableLoginOptions(),
		})
		return 0, false
	}
	userID, err := parseJWT(cookie.Value)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":         "登录已过期或无效，请重新登录",
			"login_options": getAvailableLoginOptions(),
		})
		return 0, false
	}
	return userID, true
}

// GetUserInfo使用token校验
func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	cookie, err := r.Cookie("qlist_token")
	if err != nil || cookie.Value == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":         "未登录，请先登录",
			"login_options": getAvailableLoginOptions(),
		})
		return
	}
	userID, err := parseJWT(cookie.Value)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":         "登录已过期或无效，请重新登录",
			"login_options": getAvailableLoginOptions(),
		})
		return
	}
	var user models.User
	if result := db.Where("id = ?", userID).First(&user); result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			respondWithError(w, http.StatusNotFound, "用户不存在")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "获取用户信息失败")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code": http.StatusOK,
		"user": user,
	})
}

// @Summary 获取文件信息
// @Description 获取文件名称和所需积分
// @Tags 文件下载
// @Produce json
// @Param fileUrl query string true "文件URL"
// @Success 200 {object} map[string]interface{}
// @Router /getFileInfo [get]
func GetFileInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	fileUrl := r.URL.Query().Get("fileUrl")
	if fileUrl == "" {
		respondWithError(w, http.StatusBadRequest, "文件URL不能为空")
		return
	}

	// 去除文件链接中的空格
	fileUrl = strings.TrimSpace(fileUrl)

	// 确保文件链接不以'/'开头
	if strings.HasPrefix(fileUrl, "/") {
		fileUrl = strings.TrimPrefix(fileUrl, "/")
	}

	// 查找文件积分配置
	var config models.PointConfig
	if result := db.Where("file_url = ?", fileUrl).First(&config); result.Error != nil {
		respondWithError(w, http.StatusNotFound, "未找到该文件的积分配置")
		return
	}

	// 提取文件名
	fileName := fileUrl
	if lastSlash := strings.LastIndex(fileUrl, "/"); lastSlash >= 0 {
		fileName = fileUrl[lastSlash+1:]
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code": http.StatusOK,
		"data": map[string]interface{}{
			"fileName":    fileName,
			"points":      config.Points,
			"fileUrl":     config.FileUrl,
			"description": config.Description,
		},
	})
}

func AdminGrantPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var data struct {
		Username    string `json:"username"`
		Points      int    `json:"points"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 开始事务
	tx := db.Begin()

	// 查找或创建用户
	var user models.User
	result := tx.Where("username = ?", data.Username).FirstOrCreate(&user, models.User{Username: data.Username})
	if result.Error != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "用户操作失败")
		return
	}

	// 更新用户积分
	user.Points += data.Points
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "更新积分失败")
		return
	}

	// 创建积分变更日志
	log := models.PointLog{
		UserID:      user.ID,
		Points:      data.Points,
		Description: data.Description,
	}

	if err := tx.Create(&log).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "创建积分日志失败")
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "操作失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code":    http.StatusOK,
		"message": "积分操作成功",
	})
}

// @Summary 获取用户积分
// @Description 获取指定用户的积分信息
// @Tags 用户积分
// @Produce json
// @Param username query string true "用户名"
// @Success 200 {object} map[string]interface{}
// @Router /getUserPoints [get]
func GetUserPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		respondWithError(w, http.StatusBadRequest, "用户名不能为空")
		return
	}

	var user models.User
	if result := db.Where("username = ? AND provider = ?", username, "local").First(&user); result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			respondWithError(w, http.StatusNotFound, "用户不存在")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "获取用户信息失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"username": user.Username,
		"points":   user.Points,
	})
}

// @Summary 获取积分变更日志
// @Description 获取用户积分变更历史记录
// @Tags 积分日志
// @Produce json
// @Param username query string false "用户名（可选）"
// @Param limit query int false "返回记录数量限制（默认50）"
// @Success 200 {object} map[string]interface{}
// @Router /getPointsLog [get]
func GetPointsLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 获取查询参数
	username := r.URL.Query().Get("username")
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // 默认限制
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// 构建查询
	query := db.Model(&models.PointLog{}).Order("created_at desc").Limit(limit)

	// 如果指定了用户名，则只查询该用户的日志
	if username != "" {
		var user models.User
		if result := db.Where("username = ? AND provider = ?", username, "local").First(&user); result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				respondWithError(w, http.StatusNotFound, "用户不存在")
				return
			}
			respondWithError(w, http.StatusInternalServerError, "查询用户失败")
			return
		}
		query = query.Where("user_id = ?", user.ID)
	}

	// 执行查询
	var logs []models.PointLog
	if err := query.Find(&logs).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "获取积分日志失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code": http.StatusOK,
		"logs": logs,
	})
}

// Google 登录跳转
func LoginGoogle(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.GoogleOAuth
	if cfg.ClientID == "" || cfg.RedirectURI == "" {
		respondWithError(w, http.StatusBadRequest, "未配置 Google 登录参数")
		return
	}
	authURL := "https://accounts.google.com/o/oauth2/v2/auth?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURI +
		"&response_type=code&scope=openid%20email"
	http.Redirect(w, r, authURL, http.StatusFound)
}

// GitHub 登录跳转
func LoginGitHub(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.GitHubOAuth
	if cfg.ClientID == "" || cfg.RedirectURI == "" {
		respondWithError(w, http.StatusBadRequest, "未配置 GitHub 登录参数")
		return
	}
	authURL := "https://github.com/login/oauth/authorize?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURI +
		"&scope=user:email"
	http.Redirect(w, r, authURL, http.StatusFound)
}

// 微信登录跳转
func LoginWechat(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.WechatOAuth
	if cfg.AppID == "" || cfg.RedirectURI == "" {
		respondWithError(w, http.StatusBadRequest, "未配置微信登录参数")
		return
	}
	authURL := "https://open.weixin.qq.com/connect/qrconnect?appid=" + cfg.AppID +
		"&redirect_uri=" + url.QueryEscape(cfg.RedirectURI) +
		"&response_type=code&scope=snsapi_login#wechat_redirect"
	http.Redirect(w, r, authURL, http.StatusFound)
}

// 本地邮箱注册逻辑
func RegisterLocal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var data struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, "请求参数错误")
		return
	}
	if data.Email == "" || data.Password == "" {
		respondWithError(w, http.StatusBadRequest, "邮箱和密码不能为空")
		return
	}
	// 检查邮箱是否已注册
	var user models.User
	if result := db.Where("username = ? AND provider = ?", data.Email, "local").First(&user); result.Error == nil {
		respondWithError(w, http.StatusConflict, "该邮箱已注册")
		return
	}
	// 创建新用户，密码建议加密存储（此处简化处理）
	user = models.User{Username: data.Email, Provider: "local", Password: data.Password, Points: 0}
	if err := db.Create(&user).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "注册失败")
		return
	}
	token, err := generateJWT(user.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "生成token失败")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "qlist_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	})
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "注册成功", "user": user})
}

// 本地邮箱登录逻辑
func LoginLocal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var data struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, "请求参数错误")
		return
	}
	if data.Email == "" || data.Password == "" {
		respondWithError(w, http.StatusBadRequest, "邮箱和密码不能为空")
		return
	}
	var user models.User
	if result := db.Where("username = ? AND provider = ?", data.Email, "local").First(&user); result.Error != nil {
		respondWithError(w, http.StatusUnauthorized, "用户不存在或密码错误")
		return
	}
	if user.Password != data.Password {
		respondWithError(w, http.StatusUnauthorized, "用户不存在或密码错误")
		return
	}
	token, err := generateJWT(user.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "生成token失败")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "qlist_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	})
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "登录成功", "user": user})
}

// 获取可用的三方登录选项，根据配置动态生成
func getAvailableLoginOptions() []map[string]string {
	options := make([]map[string]string, 0)
	cfg := config.Instance
	if cfg.GoogleOAuth.ClientID != "" && cfg.GoogleOAuth.RedirectURI != "" {
		options = append(options, map[string]string{"name": "Google", "url": "/api/login/google"})
	}
	if cfg.GitHubOAuth.ClientID != "" && cfg.GitHubOAuth.RedirectURI != "" {
		options = append(options, map[string]string{"name": "GitHub", "url": "/api/login/github"})
	}
	if cfg.WechatOAuth.AppID != "" && cfg.WechatOAuth.RedirectURI != "" {
		options = append(options, map[string]string{"name": "微信", "url": "/api/login/wechat"})
	}
	return options
}

// Google 登录回调
func GoogleCallback(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.GoogleOAuth
	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, http.StatusBadRequest, "缺少 code 参数")
		return
	}
	// 1. 用 code 换取 access_token
	tokenURL := "https://oauth2.googleapis.com/token"
	params := map[string]string{
		"client_id":     cfg.ClientID,
		"client_secret": cfg.ClientSecret,
		"code":          code,
		"grant_type":    "authorization_code",
		"redirect_uri":  cfg.RedirectURI,
	}
	resp, err := http.PostForm(tokenURL, toValues(params))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "请求 Google token 失败")
		return
	}
	defer resp.Body.Close()
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IdToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		respondWithError(w, http.StatusInternalServerError, "解析 Google token 响应失败")
		return
	}
	// 2. 用 access_token 获取用户信息
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + tokenResp.AccessToken
	userResp, err := http.Get(userInfoURL)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "获取 Google 用户信息失败")
		return
	}
	defer userResp.Body.Close()
	var userInfo struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		respondWithError(w, http.StatusInternalServerError, "解析 Google 用户信息失败")
		return
	}
	// 3. 设置 cookie 并重定向
	if userInfo.Email == "" {
		respondWithError(w, http.StatusInternalServerError, "未获取到 Google 邮箱")
		return
	}
	// 在数据库中查找或创建用户
	var user models.User
	if result := db.Where("username = ? AND provider = ?", userInfo.Email, "google").FirstOrCreate(&user, models.User{Username: userInfo.Email, Provider: "google"}); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "用户信息写入数据库失败")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "qlist_user",
		Value:  userInfo.Email,
		Path:   "/",
		MaxAge: 86400 * 7,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// GitHub 登录回调
func GitHubCallback(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.GitHubOAuth
	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, http.StatusBadRequest, "缺少 code 参数")
		return
	}
	// 1. 用 code 换取 access_token
	tokenURL := "https://github.com/login/oauth/access_token"
	params := map[string]string{
		"client_id":     cfg.ClientID,
		"client_secret": cfg.ClientSecret,
		"code":          code,
		"redirect_uri":  cfg.RedirectURI,
	}
	resp, err := http.PostForm(tokenURL, toValues(params))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "请求 GitHub token 失败")
		return
	}
	defer resp.Body.Close()
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := decodeFormOrJSON(resp.Body, &tokenResp); err != nil {
		respondWithError(w, http.StatusInternalServerError, "解析 GitHub token 响应失败")
		return
	}
	// 2. 用 access_token 获取用户信息
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "token "+tokenResp.AccessToken)
	userResp, err := http.DefaultClient.Do(req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "获取 GitHub 用户信息失败")
		return
	}
	defer userResp.Body.Close()
	var userInfo struct {
		Login string `json:"login"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		respondWithError(w, http.StatusInternalServerError, "解析 GitHub 用户信息失败")
		return
	}
	if userInfo.Login == "" && userInfo.Email == "" {
		respondWithError(w, http.StatusInternalServerError, "未获取到 GitHub 用户名")
		return
	}
	username := userInfo.Email
	if username == "" {
		username = userInfo.Login
	}
	// 在数据库中查找或创建用户
	var user models.User
	if result := db.Where("username = ? AND provider = ?", username, "github").FirstOrCreate(&user, models.User{Username: username, Provider: "github"}); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "用户信息写入数据库失败")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "qlist_user",
		Value:  username,
		Path:   "/",
		MaxAge: 86400 * 7,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// 微信登录回调
func WechatCallback(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.WechatOAuth
	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, http.StatusBadRequest, "缺少 code 参数")
		return
	}
	// 1. 用 code 换取 access_token
	tokenURL := "https://api.weixin.qq.com/sns/oauth2/access_token?appid=" + cfg.AppID + "&secret=" + cfg.AppSecret + "&code=" + code + "&grant_type=authorization_code"
	resp, err := http.Get(tokenURL)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "请求微信 token 失败")
		return
	}
	defer resp.Body.Close()
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		OpenID      string `json:"openid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		respondWithError(w, http.StatusInternalServerError, "解析微信 token 响应失败")
		return
	}
	if tokenResp.OpenID == "" {
		respondWithError(w, http.StatusInternalServerError, "未获取到微信 openid")
		return
	}
	// 在数据库中查找或创建用户
	var user models.User
	if result := db.Where("username = ? AND provider = ?", tokenResp.OpenID, "wechat").FirstOrCreate(&user, models.User{Username: tokenResp.OpenID, Provider: "wechat"}); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "用户信息写入数据库失败")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "qlist_user",
		Value:  tokenResp.OpenID,
		Path:   "/",
		MaxAge: 86400 * 7,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// 工具函数：map[string]string 转 url.Values
func toValues(m map[string]string) (v map[string][]string) {
	v = make(map[string][]string)
	for k, val := range m {
		v[k] = []string{val}
	}
	return
}

// 工具函数：兼容 application/x-www-form-urlencoded 或 json 响应
func decodeFormOrJSON(body io.Reader, out interface{}) error {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(body)
	if err != nil {
		return err
	}
	// 先尝试 json
	if err := json.Unmarshal(buf.Bytes(), out); err == nil {
		return nil
	}
	// 再尝试 form
	m, err := url.ParseQuery(string(buf.Bytes()))
	if err != nil {
		return err
	}
	if token, ok := m["access_token"]; ok && len(token) > 0 {
		outVal := out.(*struct {
			AccessToken string `json:"access_token"`
		})
		outVal.AccessToken = token[0]
		return nil
	}
	return fmt.Errorf("无法解析响应")
}

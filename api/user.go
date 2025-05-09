package api

import (
	"encoding/json"
	"net/http"
	"qlist/config"
	"qlist/models"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

// 生成JWT Token
func generateJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Instance.JWTSecret))
}

// 解析JWT Token
func parseJWT(tokenStr string) (uint, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Instance.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return 0, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, err
	}
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, err
	}
	return uint(userID), nil
}

// requireLogin 校验
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

// GetUserInfo
func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Method not allowed"})
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
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "用户不存在"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "获取用户信息失败"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": http.StatusOK,
		"user": user,
	})
}

// LocalRegisterHandler 本地邮箱注册处理函数
func LocalRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Method not allowed"})
		return
	}
	var data struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "请求参数错误"})
		return
	}
	if data.Email == "" || data.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "邮箱和密码不能为空"})
		return
	}
	var user models.User
	if result := db.Where("username = ? AND provider = ?", data.Email, "local").First(&user); result.Error == nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "该邮箱已注册"})
		return
	}
	user = models.User{Username: data.Email, Provider: "local", Password: data.Password, Points: 0}
	if err := db.Create(&user).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "注册失败"})
		return
	}
	token, err := generateJWT(user.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "生成token失败"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "qlist_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "注册成功", "user": user})
}

// LocalLoginHandler 本地邮箱登录处理函数
func LocalLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Method not allowed"})
		return
	}
	var data struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "请求参数错误"})
		return
	}
	if data.Email == "" || data.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "邮箱和密码不能为空"})
		return
	}
	var user models.User
	if result := db.Where("username = ? AND provider = ?", data.Email, "local").First(&user); result.Error != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "用户不存在或密码错误"})
		return
	}
	if user.Password != data.Password {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "用户不存在或密码错误"})
		return
	}
	token, err := generateJWT(user.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "生成token失败"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "qlist_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "登录成功", "user": user})
}

// 获取可用登录选项（包括三方和本地邮箱）
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
	// 添加本地邮箱登录和注册选项
	options = append(options, map[string]string{"name": "邮箱登录", "url": "/login.html"})    // 指向登录页面
	options = append(options, map[string]string{"name": "邮箱注册", "url": "/register.html"}) // 指向注册页面
	return options
}

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
	"strings" // 导入 strings 包
)

// Google 登录跳转
func LoginGoogle(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.GoogleOAuth
	if cfg.ClientID == "" || cfg.RedirectURI == "" {
		respondWithError(w, http.StatusBadRequest, "未配置 Google 登录参数")
		return
	}
	redirectAfterLogin := r.URL.Query().Get("redirect_url") // 获取用户希望最终跳转的地址

	// 构造 OAuth 回调 URL，并附加上 redirect_after_login
	callbackURL := cfg.RedirectURI
	if redirectAfterLogin != "" {
		parsedCallbackURL, err := url.Parse(callbackURL)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "处理回调URL失败")
			return
		}
		query := parsedCallbackURL.Query()
		query.Set("redirect_after_login", redirectAfterLogin)
		parsedCallbackURL.RawQuery = query.Encode()
		callbackURL = parsedCallbackURL.String()
	}

	authURL := "https://accounts.google.com/o/oauth2/v2/auth?client_id=" + cfg.ClientID +
		"&redirect_uri=" + url.QueryEscape(callbackURL) + // 使用新的回调URL，并确保编码
		"&response_type=code&scope=openid%20email"
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Google 登录回调
func GoogleCallback(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.GoogleOAuth
	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, http.StatusBadRequest, "缺少 code 参数")
		return
	}
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
	if userInfo.Email == "" {
		respondWithError(w, http.StatusInternalServerError, "未获取到 Google 邮箱")
		return
	}
	// 将邮箱转换为小写，避免大小写问题
	userEmail := strings.ToLower(userInfo.Email)
	var user models.User
	if result := db.Where("username = ? AND provider = ?", userEmail, "google").FirstOrCreate(&user, models.User{Username: userEmail, Provider: "google"}); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "用户信息写入数据库失败")
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

	// 获取之前传递的 redirect_after_login 参数
	redirectURL := r.URL.Query().Get("redirect_after_login")
	if redirectURL == "" {
		redirectURL = "/" // 默认跳转到首页
	}
	http.Redirect(w, r, redirectURL, http.StatusFound) // 跳转到 redirectURL
}

// GitHub 登录跳转
func LoginGitHub(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.GitHubOAuth
	if cfg.ClientID == "" || cfg.RedirectURI == "" {
		respondWithError(w, http.StatusBadRequest, "未配置 GitHub 登录参数")
		return
	}
	redirectAfterLogin := r.URL.Query().Get("redirect_url") // 获取用户希望最终跳转的地址

	// 构造 OAuth 回调 URL，并附加上 redirect_after_login
	callbackURL := cfg.RedirectURI
	if redirectAfterLogin != "" {
		parsedCallbackURL, err := url.Parse(callbackURL)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "处理回调URL失败")
			return
		}
		query := parsedCallbackURL.Query()
		query.Set("redirect_after_login", redirectAfterLogin)
		parsedCallbackURL.RawQuery = query.Encode()
		callbackURL = parsedCallbackURL.String()
	}

	authURL := "https://github.com/login/oauth/authorize?client_id=" + cfg.ClientID +
		"&redirect_uri=" + url.QueryEscape(callbackURL) + // 使用新的回调URL，并确保编码
		"&scope=user:email"
	http.Redirect(w, r, authURL, http.StatusFound)
}

// GitHub 登录回调
func GitHubCallback(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.GitHubOAuth
	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, http.StatusBadRequest, "缺少 code 参数")
		return
	}
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
		respondWithError(w, http.StatusInternalServerError, "未获取到 GitHub 用户名或邮箱")
		return
	}
	var username string
	if userInfo.Email != "" {
		username = strings.ToLower(userInfo.Email) // 优先使用邮箱并转为小写
	} else {
		username = strings.ToLower(userInfo.Login) // 邮箱为空则使用登录名并转为小写
	}

	var user models.User
	if result := db.Where("username = ? AND provider = ?", username, "github").FirstOrCreate(&user, models.User{Username: username, Provider: "github"}); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "用户信息写入数据库失败")
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

	// 获取之前传递的 redirect_after_login 参数
	redirectURL := r.URL.Query().Get("redirect_after_login")
	if redirectURL == "" {
		redirectURL = "/" // 默认跳转到首页
	}
	http.Redirect(w, r, redirectURL, http.StatusFound) // 跳转到 redirectURL
}

// 微信登录跳转
func LoginWechat(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.WechatOAuth
	if cfg.AppID == "" || cfg.RedirectURI == "" {
		respondWithError(w, http.StatusBadRequest, "未配置微信登录参数")
		return
	}
	redirectAfterLogin := r.URL.Query().Get("redirect_url") // 获取用户希望最终跳转的地址

	// 构造 OAuth 回调 URL，并附加上 redirect_after_login
	callbackURL := cfg.RedirectURI
	if redirectAfterLogin != "" {
		parsedCallbackURL, err := url.Parse(callbackURL)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "处理回调URL失败")
			return
		}
		query := parsedCallbackURL.Query()
		query.Set("redirect_after_login", redirectAfterLogin)
		parsedCallbackURL.RawQuery = query.Encode()
		callbackURL = parsedCallbackURL.String()
	}

	authURL := "https://open.weixin.qq.com/connect/qrconnect?appid=" + cfg.AppID +
		"&redirect_uri=" + url.QueryEscape(callbackURL) + // 使用新的回调URL，并确保编码
		"&response_type=code&scope=snsapi_login#wechat_redirect"
	http.Redirect(w, r, authURL, http.StatusFound)
}

// 微信登录回调
func WechatCallback(w http.ResponseWriter, r *http.Request) {
	cfg := config.Instance.WechatOAuth
	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, http.StatusBadRequest, "缺少 code 参数")
		return
	}
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
	var user models.User
	if result := db.Where("username = ? AND provider = ?", tokenResp.OpenID, "wechat").FirstOrCreate(&user, models.User{Username: tokenResp.OpenID, Provider: "wechat"}); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "用户信息写入数据库失败")
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

	// 获取之前传递的 redirect_after_login 参数
	redirectURL := r.URL.Query().Get("redirect_after_login")
	if redirectURL == "" {
		redirectURL = "/" // 默认跳转到首页
	}
	http.Redirect(w, r, redirectURL, http.StatusFound) // 跳转到 redirectURL
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
	if err := json.Unmarshal(buf.Bytes(), out); err == nil {
		return nil
	}
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

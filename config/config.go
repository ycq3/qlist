package config

import (
	"encoding/json"
	"errors"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func GetDialector() (gorm.Dialector, error) {
	switch Instance.DBType {
	case "mysql":
		return mysql.Open(Instance.DBConn), nil
	case "postgres":
		return postgres.Open(Instance.DBConn), nil
	case "sqlite":
		return sqlite.Open(Instance.DBConn), nil
	default:
		return nil, errors.New("不支持的数据库类型: " + Instance.DBType)
	}
}

type AppConfig struct {
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	APIKey        string `json:"api_key"` // 独立的 API Key 字段
	DBType        string `json:"db_type"`
	DBConn        string `json:"db_conn"`
	DefaultPoints int    `json:"default_points"` // 默认积分配置
	JWTSecret     string `json:"jwt_secret"`
	Alist         struct {
		Host     string `json:"host"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"alist"`
	// 三方登录配置，均为非必填，未配置则屏蔽对应登录方式
	GoogleOAuth struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RedirectURI  string `json:"redirect_uri"`
	} `json:"google_oauth,omitempty"`
	GitHubOAuth struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RedirectURI  string `json:"redirect_uri"`
	} `json:"github_oauth,omitempty"`
	WechatOAuth struct {
		AppID       string `json:"appid"`
		AppSecret   string `json:"app_secret"`
		RedirectURI string `json:"redirect_uri"`
	} `json:"wechat_oauth,omitempty"`
}

var Instance AppConfig

func LoadConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&Instance)
}

func SaveConfig(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(Instance)
}

// 生成指定长度的随机字符串，用于 JWT 密钥
func GenerateRandomSecret(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	f, err := os.Open("/dev/urandom")
	if err != nil {
		for i := range b {
			b[i] = charset[i%len(charset)]
		}
		return string(b)
	}
	defer f.Close()
	_, err = f.Read(b)
	if err != nil {
		for i := range b {
			b[i] = charset[i%len(charset)]
		}
		return string(b)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

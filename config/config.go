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
	DBType        string `json:"db_type"`
	DBConn        string `json:"db_conn"`
	DefaultPoints int    `json:"default_points"` // 默认积分配置
	Alist         struct {
		Host     string `json:"host"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"alist"`
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

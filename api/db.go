package api

import (
	"fmt"
	"qlist/config"
	"qlist/models"

	"gorm.io/gorm"
)

var db *gorm.DB

// InitDB 初始化数据库连接并进行自动迁移
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
	return db.AutoMigrate(&models.Site{}, &models.User{}, &models.PointConfig{}, &models.PointLog{})
}

// GetDB 返回数据库连接实例
func GetDB() *gorm.DB {
	return db
}
package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型，支持多渠道（provider）和本地密码
type User struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	SiteID    uint       `gorm:"index:idx_user_site_provider,unique;not null,default:0" json:"siteId"`
	Username  string     `gorm:"index:idx_user_site_provider,unique;size:128" json:"username"` // 用户名或邮箱
	Provider  string     `gorm:"index:idx_user_site_provider,unique;size:32" json:"provider"`  // 用户来源渠道 local/google/github/wechat
	Password  string     `gorm:"size:255" json:"password,omitempty"`                           // 本地用户密码，三方登录为空
	Points    int        `json:"points"`
	IsAdmin   bool       `gorm:"default:false" json:"isAdmin"`
	Logs      []PointLog `gorm:"foreignKey:UserID" json:"logs,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	Site      Site       `gorm:"foreignKey:SiteID"`
}

// PointConfig 积分配置
type PointConfig struct {
	gorm.Model
	SiteID      uint   `gorm:"uniqueIndex:idx_site_path;not null,default:0" json:"siteId"`
	FileID      uint   `gorm:"uniqueIndex:idx_site_path;not null,default:0" json:"fileId"`
	Points      int    `gorm:"column:points" json:"points"`                                                    // 积分值
	Description string `gorm:"column:description;type:varchar(255)" json:"description"`                        // 积分描述
	Site        Site   `gorm:"foreignKey:SiteID"`
}

// PointLog 积分变更日志
type PointLog struct {
	gorm.Model
	UserID    uint      `gorm:"column:user_id;index" json:"userId"` // 用户ID
	SiteID    uint      `gorm:"column:site_id;index;not null,default:0" json:"siteId"`
	Points    int       `gorm:"column:points" json:"points"`                                // 变更积分值（正数为增加，负数为减少）
	Action    string    `gorm:"column:action;type:varchar(50)" json:"action"`               // 变更类型：file_access（文件访问）, admin_grant（管理员授予）
	Details   string    `gorm:"column:details;type:varchar(255)" json:"details"`            // 变更描述
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"createdAt"` // 变更时间
	Site      Site      `gorm:"foreignKey:SiteID"`
}

func (User) TableName() string {
	return "users"
}

func (PointConfig) TableName() string {
	return "point_configs"
}

func (PointLog) TableName() string {
	return "point_logs"
}

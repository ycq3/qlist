package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型，支持多渠道（provider）和本地密码
type User struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Username  string     `gorm:"index:idx_user_provider,unique;size:128" json:"username"` // 用户名或邮箱
	Provider  string     `gorm:"index:idx_user_provider,unique;size:32" json:"provider"`  // 用户来源渠道 local/google/github/wechat
	Password  string     `gorm:"size:255" json:"password,omitempty"`                      // 本地用户密码，三方登录为空
	Points    int        `json:"points"`
	Logs      []PointLog `gorm:"foreignKey:UserID" json:"logs,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"createdAt"`
}

// PointConfig 积分配置
type PointConfig struct {
	gorm.Model
	FileUrl     string `gorm:"column:file_url;type:varchar(255);uniqueIndex" json:"fileUrl"` // 文件路径
	Points      int    `gorm:"column:points" json:"points"`                                  // 积分值
	Description string `gorm:"column:description;type:varchar(255)" json:"description"`      // 积分描述
}

// PointLog 积分变更日志
type PointLog struct {
	gorm.Model
	UserID      uint      `gorm:"column:user_id;index" json:"userId"`                           // 用户ID
	Points      int       `gorm:"column:points" json:"points"`                                  // 变更积分值（正数为增加，负数为减少）
	Type        string    `gorm:"column:type;type:varchar(20)" json:"type"`                     // 变更类型：file_access（文件访问）, admin_grant（管理员授予）
	Description string    `gorm:"column:description;type:varchar(255)" json:"description"`      // 变更描述
	FileUrl     string    `gorm:"column:file_url;type:varchar(255)" json:"fileUrl,omitempty"`   // 相关文件路径（可选）
	CreatedAt   time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"createdAt"` // 变更时间
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

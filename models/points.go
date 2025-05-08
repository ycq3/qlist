package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户信息
type User struct {
	gorm.Model
	Username string     `gorm:"column:username;type:varchar(50);uniqueIndex" json:"username"`
	Points   int        `gorm:"column:points;default:0" json:"points"` // 用户当前积分
	Logs     []PointLog `gorm:"foreignKey:UserID" json:"logs"`         // 用户积分变更记录
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

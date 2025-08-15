package models

import (
	"time"

	"gorm.io/gorm"
)

// File 文件模型，用于存储上传的文件信息
type File struct {
	gorm.Model
	SiteID      uint      `gorm:"column:site_id;index;not null,default:0" json:"siteId"`
	Path        string    `gorm:"column:path;type:varchar(255);index" json:"path"`             // 文件路径
	Name        string    `gorm:"column:name;type:varchar(255)" json:"name"`                   // 文件名称
	Size        int64     `gorm:"column:size" json:"size"`                                     // 文件大小（字节）
	ContentType string    `gorm:"column:content_type;type:varchar(100)" json:"contentType"`     // 文件类型
	Downloads   int       `gorm:"column:downloads;default:0" json:"downloads"`                 // 下载次数
	UploadedAt  time.Time `gorm:"column:uploaded_at;default:CURRENT_TIMESTAMP" json:"uploadedAt"` // 上传时间
	Site        Site      `gorm:"foreignKey:SiteID"`
	PointConfig PointConfig `json:"pointConfig,omitempty"` // 关联的积分配置
}

// TableName 指定表名
func (File) TableName() string {
	return "files"
}
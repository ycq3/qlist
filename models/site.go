package models

import (
	"time"
)

// Site 站点模型，用于区分不同站点的数据
type Site struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Domain    string    `gorm:"size:255;uniqueIndex" json:"domain"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// TableName 指定表名
func (Site) TableName() string {
	return "sites"
}
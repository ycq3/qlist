package middleware

import (
	"net/http"
	"os"
	"qlist/db"
	"qlist/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SiteContextKey 定义用于在上下文中存储站点信息的键
const SiteContextKey = "site"

// SiteMiddleware 站点中间件，用于识别当前站点
func SiteMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host
		var site models.Site
		if err := db.GetDB().Where("domain = ?", host).First(&site).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// 如果是开发环境，可以创建一个默认站点
				if IsDev() {
					defaultSite := &models.Site{Name: "Default Site", Domain: host}
					if err := db.GetDB().Create(defaultSite).Error; err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "创建默认站点失败"})
						c.Abort()
						return
					}
					c.Set(string(SiteContextKey), defaultSite)
				} else {
					// 生产环境下，如果站点不存在，则显示特定的提示页面
					c.File("public/dist/site_not_found.html")
					c.Abort()
					return
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "获取站点信息失败"})
				c.Abort()
				return
			}
		} else {
			c.Set(string(SiteContextKey), &site)
		}
		c.Next()
	}
}

// GetSiteFromContext 从 Gin 上下文中获取站点信息
func GetSiteFromContext(c *gin.Context) (*models.Site, bool) {
	site, exists := c.Get(string(SiteContextKey))
	if !exists {
		return nil, false
	}
	return site.(*models.Site), true
}

// IsDev 判断是否为开发环境
func IsDev() bool {
	return os.Getenv("ENV") == "development"
}
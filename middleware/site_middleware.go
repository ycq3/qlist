package middleware

import (
	"context"
	"net/http"
	"os"
	"qlist/api"
	"qlist/models"

	"gorm.io/gorm"
)

// SiteContextKey 定义用于在上下文中存储站点信息的键
const SiteContextKey = "site"

// SiteMiddleware 站点中间件，用于识别当前站点
type SiteMiddleware struct{}

// GetSiteByHost 根据域名获取站点信息
func (sm *SiteMiddleware) GetSiteByHost(host string) (*models.Site, error) {
	var site models.Site
	if err := api.GetDB().Where("domain = ?", host).First(&site).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 未找到站点，不一定是错误
		}
		return nil, err
	}
	return &site, nil
}

// Handler 中间件处理器
func (sm *SiteMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		site, err := sm.GetSiteByHost(host)
		if err != nil {
			// 数据库查询错误
			api.RespondWithError(w, http.StatusInternalServerError, "获取站点信息失败")
			return
		}

		if site == nil {
			// 如果是开发环境，可以创建一个默认站点
			if IsDev() { // 假设 IsDev() 用于判断是否为开发环境
				defaultSite := &models.Site{Name: "Default Site", Domain: host}
				if err := api.GetDB().Create(defaultSite).Error; err != nil {
					api.RespondWithError(w, http.StatusInternalServerError, "创建默认站点失败")
					return
				}
				site = defaultSite
			} else {
				// 生产环境下，如果站点不存在，则显示特定的提示页面
				http.ServeFile(w, r, "public/dist/site_not_found.html")
				return
			}
		}

		// 将站点信息存入请求上下文
		ctx := context.WithValue(r.Context(), SiteContextKey, site)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetSiteFromContext 从请求上下文中获取站点信息
func GetSiteFromContext(ctx context.Context) *models.Site {
	if site, ok := ctx.Value(SiteContextKey).(*models.Site); ok {
		return site
	}
	return nil
}

// IsDev 判断是否为开发环境
func IsDev() bool {
	return os.Getenv("ENV") == "development"
}
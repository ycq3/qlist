package api

import (
	"fmt"
	"net/http"
	"qlist/db"
	"qlist/middleware"
	"qlist/models"
	"qlist/pkg/response"
	"qlist/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetPointsList godoc
// @Summary 获取积分配置列表
// @Description 获取当前站点的所有积分配置
// @Tags Points
// @Accept json
// @Produce json
// @Success 200 {array} models.PointConfig
// @Router /api/points [get]
func GetPointsList(c *gin.Context) {
	site, exists := middleware.GetSiteFromContext(c)
	if !exists {
		response.RespondWithError(c, http.StatusInternalServerError, "无法获取站点信息")
		return
	}

	var configs []models.PointConfig
	if err := db.GetDB().Where("site_id = ?", site.ID).Find(&configs).Error; err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, "无法获取积分配置列表")
		return
	}
	response.RespondWithJSON(c, http.StatusOK, configs)
}

// ConfigurePoints godoc
// @Summary 配置积分
// @Description 为指定路径配置所需积分
// @Tags Points
// @Accept json
// @Produce json
// @Param config body models.PointConfig true "积分配置"
// @Success 200 {object} models.PointConfig
// @Router /api/points/configure [post]
func ConfigurePoints(c *gin.Context) {
	site, exists := middleware.GetSiteFromContext(c)
	if !exists {
		response.RespondWithError(c, http.StatusInternalServerError, "无法获取站点信息")
		return
	}

	var config models.PointConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.RespondWithError(c, http.StatusBadRequest, "无效的请求数据")
		return
	}
	config.SiteID = site.ID

	var existingConfig models.PointConfig
	err := db.GetDB().Where("site_id = ? AND file_id = ?", site.ID, config.FileID).First(&existingConfig).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		response.RespondWithError(c, http.StatusInternalServerError, "查询积分配置失败")
		return
	}

	if err == gorm.ErrRecordNotFound {
		if err := db.GetDB().Create(&config).Error; err != nil {
			response.RespondWithError(c, http.StatusInternalServerError, "创建积分配置失败")
			return
		}
	} else {
		if err := db.GetDB().Model(&existingConfig).Updates(config).Error; err != nil {
			response.RespondWithError(c, http.StatusInternalServerError, "更新积分配置失败")
			return
		}
		config.ID = existingConfig.ID
	}

	response.RespondWithJSON(c, http.StatusOK, config)
}

// GetUsersList godoc
// @Summary 获取用户列表
// @Description 获取当前站点的所有用户列表
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {array} models.User
// @Router /api/users [get]
func GetUsersList(c *gin.Context) {
	site, exists := middleware.GetSiteFromContext(c)
	if !exists {
		response.RespondWithError(c, http.StatusInternalServerError, "无法获取站点信息")
		return
	}

	var users []models.User
	if err := db.GetDB().Where("site_id = ?", site.ID).Find(&users).Error; err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, "无法获取用户列表")
		return
	}
	response.RespondWithJSON(c, http.StatusOK, users)
}

// DownloadFile godoc
// @Summary 下载文件
// @Description 用户下载文件，扣除相应积分
// @Tags Points
// @Accept json
// @Produce json
// @Param path query string true "文件路径"
// @Success 200 {object} map[string]string "包含下载链接"
// @Router /api/download [get]
func DownloadFile(c *gin.Context) {
	site, exists := middleware.GetSiteFromContext(c)
	if !exists {
		response.RespondWithError(c, http.StatusInternalServerError, "无法获取站点信息")
		return
	}

	user, exists := c.Get("user")
	if !exists {
		response.RespondWithError(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	currentUser := user.(*models.User)

	filePath := c.Query("path")
	if filePath == "" {
		response.RespondWithError(c, http.StatusBadRequest, "文件路径不能为空")
		return
	}

	var config models.PointConfig
	if err := db.GetDB().Where("site_id = ? AND path = ?", site.ID, filePath).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.RespondWithError(c, http.StatusNotFound, "文件未配置积分")
			return
		}
		response.RespondWithError(c, http.StatusInternalServerError, "查询文件积分配置失败")
		return
	}

	if currentUser.Points < config.Points {
		response.RespondWithError(c, http.StatusForbidden, "积分不足")
		return
	}

	tx := db.GetDB().Begin()
	// 更新用户积分
	if err := tx.Model(currentUser).Update("points", gorm.Expr("points - ?", config.Points)).Error; err != nil {
		tx.Rollback()
		response.RespondWithError(c, http.StatusInternalServerError, "扣除积分失败")
		return
	}

	log := models.PointLog{
		UserID:  currentUser.ID,
		SiteID:  site.ID,
		Points:  -config.Points,
		Action:  "file_access",
		Details: fmt.Sprintf("下载文件: %s", filePath),
	}
	if err := tx.Create(&log).Error; err != nil {
		tx.Rollback()
		response.RespondWithError(c, http.StatusInternalServerError, "记录积分日志失败")
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.RespondWithError(c, http.StatusInternalServerError, "提交事务失败")
		return
	}

	// 更新文件下载次数
	if err := UpdateFileDownloadCount(site.ID, filePath); err != nil {
		// 仅记录错误，不影响用户下载
		fmt.Printf("更新文件下载次数失败: %v\n", err)
	}

	// 获取下载链接
	uploader := &storage.AlistUploader{}
	downloadUrl, err := uploader.GetDownloadUrl(filePath)
	if err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, "获取下载链接失败")
		return
	}

	response.RespondWithJSON(c, http.StatusOK, gin.H{
		"url": downloadUrl,
	})
}

// AdminGrantPointsRequest 定义管理员授予积分的请求体
type AdminGrantPointsRequest struct {
	UserID uint `json:"user_id"`
	Points int  `json:"points"`
}

// AdminGrantPoints godoc
// @Summary 管理员授予积分
// @Description 管理员为指定用户授予积分
// @Tags Users
// @Accept json
// @Produce json
// @Param grant_request body AdminGrantPointsRequest true "授予积分请求"
// @Success 200 {object} models.User
// @Router /api/users/grant [post]
func AdminGrantPoints(c *gin.Context) {
	site, exists := middleware.GetSiteFromContext(c)
	if !exists {
		response.RespondWithError(c, http.StatusInternalServerError, "无法获取站点信息")
		return
	}

	var req AdminGrantPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondWithError(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	var user models.User
	if err := db.GetDB().Where("id = ? AND site_id = ?", req.UserID, site.ID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.RespondWithError(c, http.StatusNotFound, "用户不存在")
			return
		}
		response.RespondWithError(c, http.StatusInternalServerError, "查询用户失败")
		return
	}

	tx := db.GetDB().Begin()
	user.Points += req.Points
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		response.RespondWithError(c, http.StatusInternalServerError, "更新用户积分失败")
		return
	}

	log := models.PointLog{
		UserID:  user.ID,
		SiteID:  site.ID,
		Points:  req.Points,
		Action:  "管理员授予",
		Details: fmt.Sprintf("管理员授予 %d 积分", req.Points),
	}
	if err := tx.Create(&log).Error; err != nil {
		tx.Rollback()
		response.RespondWithError(c, http.StatusInternalServerError, "记录日志失败")
		return
	}

	if err := tx.Commit().Error; err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, "事务提交失败")
		return
	}

	response.RespondWithJSON(c, http.StatusOK, user)
}

// GetUserPoints godoc
// @Summary 获取用户积分
// @Description 获取当前登录用户的积分信息
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} models.User
// @Router /api/user/points [get]
func GetUserPoints(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		response.RespondWithError(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	response.RespondWithJSON(c, http.StatusOK, user.(*models.User))
}

// GetPointsLog godoc
// @Summary 获取积分日志
// @Description 获取当前用户的积分变动日志
// @Tags Points
// @Accept json
// @Produce json
// @Success 200 {array} models.PointLog
// @Router /api/points/log [get]
func GetPointsLog(c *gin.Context) {
	site, exists := middleware.GetSiteFromContext(c)
	if !exists {
		response.RespondWithError(c, http.StatusInternalServerError, "无法获取站点信息")
		return
	}

	user, exists := c.Get("user")
	if !exists {
		response.RespondWithError(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	currentUser := user.(*models.User)

	var logs []models.PointLog
	if err := db.GetDB().Where("user_id = ? AND site_id = ?", currentUser.ID, site.ID).Order("created_at desc").Find(&logs).Error; err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, "无法获取积分日志")
		return
	}
	response.RespondWithJSON(c, http.StatusOK, logs)
}

// GetFileInfo godoc
// @Summary 获取文件信息
// @Description 根据路径获取文件的积分配置信息
// @Tags Points
// @Accept json
// @Produce json
// @Param path query string true "文件路径"
// @Success 200 {object} models.PointConfig
// @Router /api/fileinfo [get]
func GetFileInfo(c *gin.Context) {
	site, exists := middleware.GetSiteFromContext(c)
	if !exists {
		response.RespondWithError(c, http.StatusInternalServerError, "无法获取站点信息")
		return
	}

	filePath := c.Query("path")
	if filePath == "" {
		response.RespondWithError(c, http.StatusBadRequest, "文件路径不能为空")
		return
	}

	var config models.PointConfig
	if err := db.GetDB().Where("site_id = ? AND path = ?", site.ID, filePath).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.RespondWithError(c, http.StatusNotFound, "文件未配置积分")
			return
		}
		response.RespondWithError(c, http.StatusInternalServerError, "查询文件积分配置失败")
		return
	}

	response.RespondWithJSON(c, http.StatusOK, config)
}

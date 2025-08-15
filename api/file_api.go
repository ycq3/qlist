package api

import (
	"net/http"
	"qlist/db"
	"qlist/middleware"
	"qlist/models"
	"qlist/pkg/response"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetRecentFiles godoc
// @Summary 获取最近上传的文件
// @Description 获取站点最近上传的文件列表
// @Tags Files
// @Accept json
// @Produce json
// @Param limit query int false "限制返回的文件数量" default(10)
// @Param offset query int false "分页偏移量" default(0)
// @Success 200 {array} models.File
// @Router /api/files/recent [get]
func GetRecentFiles(c *gin.Context) {
	site, exists := middleware.GetSiteFromContext(c)
	if !exists {
		response.RespondWithError(c, http.StatusInternalServerError, "无法获取站点信息")
		return
	}

	// 获取分页参数
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 10 // 默认值或无效值处理
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0 // 默认值或无效值处理
	}

	// 查询最近上传的文件
	var files []models.File
	if err := db.GetDB().Where("site_id = ?", site.ID).Order("uploaded_at DESC").Limit(limit).Offset(offset).Find(&files).Error; err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, "查询文件列表失败")
		return
	}

	// 获取每个文件的积分配置
	for i := range files {
		var config models.PointConfig
		if err := db.GetDB().Where("site_id = ? AND path = ?", site.ID, files[i].Path).First(&config).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				// 只记录非"未找到"的错误
				files[i].PointConfig = models.PointConfig{Points: 0}
			}
		} else {
			files[i].PointConfig = config
		}
	}

	response.RespondWithJSON(c, http.StatusOK, files)
}

// RecordFileUpload 记录文件上传信息
// 此函数在文件上传成功后调用，用于记录文件信息
func RecordFileUpload(siteID uint, path string, name string, size int64, contentType string) error {
	file := models.File{
		SiteID:      siteID,
		Path:        path,
		Name:        name,
		Size:        size,
		ContentType: contentType,
		UploadedAt:  time.Now(),
	}

	return db.GetDB().Create(&file).Error
}

// UpdateFileDownloadCount 更新文件下载次数
// 在文件被下载时调用此函数
func UpdateFileDownloadCount(siteID uint, path string) error {
	return db.GetDB().Model(&models.File{}).Where("site_id = ? AND path = ?", siteID, path).UpdateColumn("downloads", gorm.Expr("downloads + ?", 1)).Error
}
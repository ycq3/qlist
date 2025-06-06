package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"qlist/config"
	"qlist/models"
	"qlist/storage"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

var db *gorm.DB

// @title 积分管理系统API
// @version 1.0
// @description 提供用户积分管理、积分配置、积分日志查询和文件下载等功能
// @BasePath /api

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
	db.AutoMigrate(&models.User{}, &models.PointConfig{}, &models.PointLog{})
	return nil
}

// @Summary 获取积分配置列表
// @Description 获取所有文件的积分配置
// @Tags 积分配置
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /getPointsList [get]
func GetPointsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var configs []models.PointConfig
	if result := db.Find(&configs); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "获取积分配置列表失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code": http.StatusOK,
		"data": configs,
	})
}

// @Summary 配置文件积分
// @Description 为文件设置积分值
// @Tags 积分配置
// @Accept json
// @Produce json
// @Param config body models.PointConfig true "积分配置信息"
// @Success 200 {object} map[string]interface{}
// @Router /configurePoints [post]
func ConfigurePoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var config models.PointConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondWithError(w, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 去除文件链接中的空格
	config.FileUrl = strings.TrimSpace(config.FileUrl)

	// 确保文件链接不以'/'开头
	if strings.HasPrefix(config.FileUrl, "/") {
		config.FileUrl = strings.TrimPrefix(config.FileUrl, "/")
	}

	// 查找是否已存在配置
	var existingConfig models.PointConfig
	result := db.Where("file_url = ?", config.FileUrl).First(&existingConfig)

	if result.Error == nil {
		// 更新现有配置
		existingConfig.Points = config.Points
		existingConfig.Description = config.Description
		if result := db.Save(&existingConfig); result.Error != nil {
			respondWithError(w, http.StatusInternalServerError, "更新积分配置失败")
			return
		}
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"code":    http.StatusOK,
			"message": "配置更新成功",
		})
		return
	}

	// 创建新配置
	if result := db.Create(&config); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "保存积分配置失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code":    http.StatusOK,
		"message": "配置保存成功",
	})
}

// @Summary 获取用户列表
// @Description 获取所有用户及其积分信息
// @Tags 用户管理
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /getUsersList [get]
func GetUsersList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var users []models.User
	// Preload Logs to avoid N+1 query problem if logs are needed in the future, though not directly used now.
	if result := db.Preload("Logs").Find(&users); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "获取用户列表失败")
		return
	}

	// Optionally, transform users if needed, e.g., to format CreatedAt or customize output
	// For now, directly returning users as is, assuming GORM handles CreatedAt population and JSON marshaling correctly.

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code":  http.StatusOK,
		"users": users, // Ensure 'users' includes 'Provider' and 'CreatedAt' due to model changes
	})
}

func DownloadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var data struct {
		FileUrl string `json:"fileUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, "请求参数错误")
		return
	}
	// 去除文件链接中的空格
	data.FileUrl = strings.TrimSpace(data.FileUrl)
	// 确保文件链接不以'/'开头
	if strings.HasPrefix(data.FileUrl, "/") {
		data.FileUrl = strings.TrimPrefix(data.FileUrl, "/")
	}
	// 检查用户是否已登录
	userId, ok := requireLogin(w, r)
	if !ok {
		return
	}
	// 查找文件积分配置
	var configModel models.PointConfig
	if result := db.Where("file_url = ?", data.FileUrl).First(&configModel); result.Error != nil {
		// 使用默认积分配置
		configModel = models.PointConfig{
			FileUrl:     data.FileUrl,
			Points:      config.Instance.DefaultPoints,
			Description: "无",
		}
	}
	// 查找或创建用户
	var user models.User
	if result := db.Where("id = ?", userId).First(&user); result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "用户操作失败")
		return
	}

	// 检查用户积分是否足够
	if user.Points < configModel.Points {
		respondWithError(w, http.StatusForbidden, "积分不足")
		return
	}

	// 开始事务
	tx := db.Begin()

	// 扣除用户积分
	user.Points -= configModel.Points
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "扣除积分失败")
		return
	}

	// 创建积分变更日志
	log := models.PointLog{
		UserID:      user.ID,
		Points:      -configModel.Points,
		Type:        "file_access",
		Description: fmt.Sprintf("下载文件：%s", data.FileUrl),
		FileUrl:     data.FileUrl,
	}

	if err := tx.Create(&log).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "创建积分日志失败")
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "操作失败")
		return
	}

	// 获取文件下载地址
	uploader := &storage.AlistUploader{}
	downloadUrl, err := uploader.GetDownloadUrl(data.FileUrl)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "获取下载地址失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code":        http.StatusOK,
		"downloadUrl": downloadUrl,
	})
}

func AdminGrantPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var data struct {
		Username    string `json:"username"`
		Points      int    `json:"points"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 开始事务
	tx := db.Begin()

	// 查找或创建用户
	var user models.User
	result := tx.Where("username = ?", data.Username).FirstOrCreate(&user, models.User{Username: data.Username})
	if result.Error != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "用户操作失败")
		return
	}

	// 更新用户积分
	user.Points += data.Points
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "更新积分失败")
		return
	}

	// 创建积分变更日志
	log := models.PointLog{
		UserID:      user.ID,
		Points:      data.Points,
		Description: data.Description,
	}

	if err := tx.Create(&log).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "创建积分日志失败")
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "操作失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code":    http.StatusOK,
		"message": "积分操作成功",
	})
}

// @Summary 获取用户积分
// @Description 获取指定用户的积分信息
// @Tags 用户积分
// @Produce json
// @Param username query string true "用户名"
// @Success 200 {object} map[string]interface{}
// @Router /getUserPoints [get]
func GetUserPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		respondWithError(w, http.StatusBadRequest, "用户名不能为空")
		return
	}

	var user models.User
	if result := db.Where("username = ? AND provider = ?", username, "local").First(&user); result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			respondWithError(w, http.StatusNotFound, "用户不存在")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "获取用户信息失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"username": user.Username,
		"points":   user.Points,
	})
}

// @Summary 获取积分变更日志
// @Description 获取用户积分变更历史记录
// @Tags 积分日志
// @Produce json
// @Param username query string false "用户名（可选）"
// @Param limit query int false "返回记录数量限制（默认50）"
// @Success 200 {object} map[string]interface{}
// @Router /getPointsLog [get]
func GetPointsLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 获取查询参数
	username := r.URL.Query().Get("username")
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // 默认限制
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// 构建查询
	query := db.Model(&models.PointLog{}).Order("created_at desc").Limit(limit)

	// 如果指定了用户名，则只查询该用户的日志
	if username != "" {
		var user models.User
		if result := db.Where("username = ? AND provider = ?", username, "local").First(&user); result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				respondWithError(w, http.StatusNotFound, "用户不存在")
				return
			}
			respondWithError(w, http.StatusInternalServerError, "查询用户失败")
			return
		}
		query = query.Where("user_id = ?", user.ID)
	}

	// 执行查询
	var logs []models.PointLog
	if err := query.Find(&logs).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "获取积分日志失败")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code": http.StatusOK,
		"logs": logs,
	})
}

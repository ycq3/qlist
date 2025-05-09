package api

import (
	"net/http"
	"qlist/models"
	"strings"
)

// 获取文件信息
func GetFileInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	fileUrl := r.URL.Query().Get("fileUrl")
	if fileUrl == "" {
		respondWithError(w, http.StatusBadRequest, "文件URL不能为空")
		return
	}
	fileUrl = strings.TrimSpace(fileUrl)
	if strings.HasPrefix(fileUrl, "/") {
		fileUrl = strings.TrimPrefix(fileUrl, "/")
	}
	var config models.PointConfig
	if result := db.Where("file_url = ?", fileUrl).First(&config); result.Error != nil {
		respondWithError(w, http.StatusNotFound, "未找到该文件的积分配置")
		return
	}
	fileName := fileUrl
	if lastSlash := strings.LastIndex(fileUrl, "/"); lastSlash >= 0 {
		fileName = fileUrl[lastSlash+1:]
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"code": http.StatusOK,
		"data": map[string]interface{}{
			"fileName":    fileName,
			"points":      config.Points,
			"fileUrl":     config.FileUrl,
			"description": config.Description,
		},
	})
}

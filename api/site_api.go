package api

import (
	"encoding/json"
	"net/http"
	"qlist/models"

	"gorm.io/gorm"
)

// CreateSite 创建新站点
// @Summary 创建新站点
// @Description 创建一个新站点
// @Tags Sites
// @Accept json
// @Produce json
// @Param site body models.Site true "站点信息"
// @Success 200 {object} models.Site
// @Router /sites [post]
func CreateSite(w http.ResponseWriter, r *http.Request) {
	var site models.Site
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		respondWithError(w, http.StatusBadRequest, "无效的请求")
		return
	}

	if err := db.Create(&site).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "创建站点失败")
		return
	}

	respondWithJSON(w, http.StatusOK, site)
}

// GetSites 获取所有站点
// @Summary 获取所有站点
// @Description 获取所有站点的列表
// @Tags Sites
// @Produce json
// @Success 200 {array} models.Site
// @Router /sites [get]
func GetSites(w http.ResponseWriter, r *http.Request) {
	var sites []models.Site
	if err := db.Find(&sites).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "获取站点列表失败")
		return
	}
	respondWithJSON(w, http.StatusOK, sites)
}

// GetSite 获取单个站点
// @Summary 获取单个站点
// @Description 根据ID获取单个站点的信息
// @Tags Sites
// @Produce json
// @Param id path int true "站点ID"
// @Success 200 {object} models.Site
// @Router /sites/{id} [get]
func GetSite(w http.ResponseWriter, r *http.Request) {
	// 从路由参数中获取站点ID，这需要您的路由支持，例如使用 gorilla/mux
	// vars := mux.Vars(r)
	// id, err := strconv.Atoi(vars["id"])
	// if err != nil {
	// 	respondWithError(w, http.StatusBadRequest, "无效的站点ID")
	// 	return
	// }

	// 这里我们暂时从查询参数中获取ID，因为标准库不支持路径参数
	id := r.URL.Query().Get("id")
	var site models.Site
	if err := db.First(&site, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondWithError(w, http.StatusNotFound, "未找到站点")
		} else {
			respondWithError(w, http.StatusInternalServerError, "获取站点信息失败")
		}
		return
	}
	respondWithJSON(w, http.StatusOK, site)
}

// UpdateSite 更新站点信息
// @Summary 更新站点信息
// @Description 根据ID更新站点信息
// @Tags Sites
// @Accept json
// @Produce json
// @Param id path int true "站点ID"
// @Param site body models.Site true "站点信息"
// @Success 200 {object} models.Site
// @Router /sites/{id} [put]
func UpdateSite(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	var site models.Site
	if err := db.First(&site, id).Error; err != nil {
		respondWithError(w, http.StatusNotFound, "未找到站点")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		respondWithError(w, http.StatusBadRequest, "无效的请求")
		return
	}

	if err := db.Save(&site).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "更新站点失败")
		return
	}
	respondWithJSON(w, http.StatusOK, site)
}

// DeleteSite 删除站点
// @Summary 删除站点
// @Description 根据ID删除站点
// @Tags Sites
// @Produce json
// @Param id path int true "站点ID"
// @Success 200 {object} map[string]string
// @Router /sites/{id} [delete]
func DeleteSite(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := db.Delete(&models.Site{}, id).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "删除站点失败")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "站点删除成功"})
}
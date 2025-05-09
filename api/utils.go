package api

import (
	"encoding/json"
	"net/http"
)

// 响应错误处理
func respondWithError(w http.ResponseWriter, code int, message string) {
	if code == http.StatusServiceUnavailable {
		w.Header().Set("Location", "/config.html")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
		"code":  code,
	})
}

// 响应JSON数据
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

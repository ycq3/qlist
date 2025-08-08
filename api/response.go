package api

import (
	"encoding/json"
	"net/http"
)

// RespondWithError 响应错误处理
func RespondWithError(w http.ResponseWriter, code int, message string) {
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

// RespondWithJSON 响应JSON数据
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}
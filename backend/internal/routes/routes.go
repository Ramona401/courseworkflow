package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"tedna/internal/config"
)

// Setup 注册所有路由，返回 http.ServeMux
func Setup(cfg *config.Config) *http.ServeMux {
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("/api/v1/health", healthHandler)

	// TODO P1-1: 认证路由
	// mux.HandleFunc("/api/v1/auth/login", ...)
	// mux.HandleFunc("/api/v1/auth/logout", ...)
	// mux.HandleFunc("/api/v1/auth/me", ...)

	return mux
}

// healthHandler 健康检查接口
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"version": "0.0.1",
		"time":    time.Now().Format(time.RFC3339),
	})
}

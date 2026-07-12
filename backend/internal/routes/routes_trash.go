package routes

import (
	"net/http"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// RegisterTrashRoutes 注册回收站路由
func RegisterTrashRoutes(mux *http.ServeMux, authMW func(http.Handler) http.Handler, trashHandler *handlers.TrashHandler) {
	// GET /api/v1/trash — 获取回收站列表（教案+课件）
	mux.Handle("/api/v1/trash", middleware.Chain(http.HandlerFunc(trashHandler.ListTrash), authMW))

	// POST /api/v1/trash/{id}/restore — 恢复项目
	// DELETE /api/v1/trash/{id}/permanent — 永久删除
	mux.Handle("/api/v1/trash/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case len(path) > len("/api/v1/trash/") && hasSuffix(path, "/restore") && r.Method == http.MethodPost:
			trashHandler.RestoreItem(w, r)
		case len(path) > len("/api/v1/trash/") && hasSuffix(path, "/permanent") && r.Method == http.MethodDelete:
			trashHandler.PermanentDelete(w, r)
		default:
			http.NotFound(w, r)
		}
	}), authMW))
}

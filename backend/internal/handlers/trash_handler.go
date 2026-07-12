package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// TrashHandler 回收站处理器
type TrashHandler struct {
	trashService *services.TrashService
}

// NewTrashHandler 创建回收站处理器
func NewTrashHandler(trashService *services.TrashService) *TrashHandler {
	return &TrashHandler{trashService: trashService}
}

// ListTrash 获取回收站列表
// GET /api/v1/trash
func (h *TrashHandler) ListTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	result, err := h.trashService.ListTrash(claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, result)
}

// trashItemRequest 恢复/永久删除请求体
type trashItemRequest struct {
	Type string `json:"type"` // "lesson_plan" 或 "courseware"
}

// RestoreItem 恢复回收站中的项目
// POST /api/v1/trash/{id}/restore
func (h *TrashHandler) RestoreItem(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 从路径提取ID：/api/v1/trash/{id}/restore
	itemID := extractTrashItemID(r.URL.Path, "/restore")
	if itemID == "" {
		utils.BadRequest(w, "无效的ID")
		return
	}

	var req trashItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Type == "" {
		utils.BadRequest(w, "请指定类型(type): lesson_plan 或 courseware")
		return
	}

	if err := h.trashService.RestoreItem(itemID, req.Type, claims.UserID); err != nil {
		utils.Fail(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(w, map[string]string{"message": "恢复成功"})
}

// PermanentDelete 永久删除回收站中的项目
// DELETE /api/v1/trash/{id}/permanent
func (h *TrashHandler) PermanentDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 从路径提取ID：/api/v1/trash/{id}/permanent
	itemID := extractTrashItemID(r.URL.Path, "/permanent")
	if itemID == "" {
		utils.BadRequest(w, "无效的ID")
		return
	}

	var req trashItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Type == "" {
		utils.BadRequest(w, "请指定类型(type): lesson_plan 或 courseware")
		return
	}

	if err := h.trashService.PermanentDeleteItem(itemID, req.Type, claims.UserID); err != nil {
		utils.Fail(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(w, map[string]string{"message": "已永久删除"})
}

// extractTrashItemID 从路径中提取回收站项目ID
// 路径格式: /api/v1/trash/{id}/restore 或 /api/v1/trash/{id}/permanent
func extractTrashItemID(path, suffix string) string {
	// 去掉后缀
	path = strings.TrimSuffix(path, suffix)
	// 去掉前缀 /api/v1/trash/
	const prefix = "/api/v1/trash/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	if len(id) < 10 {
		return ""
	}
	return id
}

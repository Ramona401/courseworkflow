package handlers

// kb_admin_handler.go — 知识库白名单管理处理器（仅 admin，挂在 /admin 下）
//
// 提供 /api/v1/admin/kb-authorized 的增删查：
//   GET    列出全部白名单成员（JOIN users 带用户名/角色/授权人）
//   POST   添加白名单成员（body: {user_id, note}）
//   DELETE 移除白名单成员（路径尾段为 user_id）

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// KBAdminHandler 知识库白名单管理处理器
type KBAdminHandler struct{}

// NewKBAdminHandler 创建白名单管理处理器
func NewKBAdminHandler() *KBAdminHandler {
	return &KBAdminHandler{}
}

// ListAuthorized GET /api/v1/admin/kb-authorized — 列出白名单
func (h *KBAdminHandler) ListAuthorized(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	items, err := repository.ListKBAuthorized(r.Context())
	if err != nil {
		utils.InternalError(w, "查询白名单失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"items": items, "total": len(items)})
}

// AddAuthorized POST /api/v1/admin/kb-authorized — 添加白名单成员
func (h *KBAdminHandler) AddAuthorized(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	var req models.KBAddAuthorizedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		utils.BadRequest(w, "缺少 user_id")
		return
	}
	if err := repository.AddKBAuthorized(r.Context(), req.UserID, claims.UserID, req.Note); err != nil {
		utils.InternalError(w, "添加白名单失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已添加"})
}

// RemoveAuthorized DELETE /api/v1/admin/kb-authorized/{user_id} — 移除白名单成员
func (h *KBAdminHandler) RemoveAuthorized(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	// 取路径尾段作 user_id
	path := strings.TrimRight(r.URL.Path, "/")
	idx := strings.LastIndex(path, "/")
	if idx < 0 || idx == len(path)-1 {
		utils.BadRequest(w, "缺少 user_id")
		return
	}
	userID := path[idx+1:]
	if userID == "" {
		utils.BadRequest(w, "缺少 user_id")
		return
	}
	if err := repository.RemoveKBAuthorized(r.Context(), userID); err != nil {
		if err == repository.ErrMemberNotFound {
			utils.BadRequest(w, "该成员不在白名单中")
			return
		}
		utils.InternalError(w, "移除白名单失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已移除"})
}

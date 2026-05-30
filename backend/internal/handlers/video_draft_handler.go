package handlers

// video_draft_handler.go — 视频编辑器草稿HTTP处理器(v0.42.5)
//
// 接口:
//   POST   /api/v1/coursewares/{id}/video-drafts            — 保存草稿
//   GET    /api/v1/coursewares/{id}/video-drafts             — 列出草稿
//   DELETE /api/v1/coursewares/{id}/video-drafts/{draft_id}  — 删除草稿

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// VideoDraftHandler 视频编辑器草稿处理器
type VideoDraftHandler struct{}

// NewVideoDraftHandler 创建草稿处理器
func NewVideoDraftHandler() *VideoDraftHandler { return &VideoDraftHandler{} }

// HandleDrafts 统一路由分发
func (h *VideoDraftHandler) HandleDrafts(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")
	if strings.HasSuffix(path, "/video-drafts") {
		switch r.Method {
		case http.MethodGet:
			h.ListDrafts(w, r)
		case http.MethodPost:
			h.SaveDraft(w, r)
		default:
			utils.Fail(w, http.StatusMethodNotAllowed, "仅支持 GET/POST")
		}
		return
	}
	if r.Method == http.MethodDelete && strings.Contains(path, "/video-drafts/") {
		h.DeleteDraft(w, r)
		return
	}
	utils.Fail(w, http.StatusNotFound, "未找到路由")
}

// SaveDraft POST /api/v1/coursewares/{id}/video-drafts
func (h *VideoDraftHandler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	cwID := extractDraftCoursewareID(r.URL.Path)
	if cwID == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	var req struct {
		Name      string          `json:"name"`
		ClipsData json.RawMessage `json:"clips_data"`
		ClipCount int             `json:"clip_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if len(req.ClipsData) == 0 || req.ClipCount <= 0 {
		utils.BadRequest(w, "草稿数据不能为空")
		return
	}
	// 超过10个时自动删除最旧的
	count, _ := repository.CountVideoDrafts(r.Context(), cwID, claims.UserID)
	if count >= 10 {
		_ = repository.DeleteOldestVideoDraft(r.Context(), cwID, claims.UserID)
	}
	id, createdAt, err := repository.CreateVideoDraft(r.Context(), cwID, claims.UserID, req.Name, string(req.ClipsData), req.ClipCount)
	if err != nil {
		utils.InternalError(w, "保存草稿失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"id": id, "created_at": createdAt, "message": "草稿保存成功"})
}

// ListDrafts GET /api/v1/coursewares/{id}/video-drafts
func (h *VideoDraftHandler) ListDrafts(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	cwID := extractDraftCoursewareID(r.URL.Path)
	if cwID == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	drafts, err := repository.ListVideoDrafts(r.Context(), cwID, claims.UserID)
	if err != nil {
		utils.InternalError(w, "查询草稿失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"drafts": drafts, "total": len(drafts)})
}

// DeleteDraft DELETE /api/v1/coursewares/{id}/video-drafts/{draft_id}
func (h *VideoDraftHandler) DeleteDraft(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	draftID := extractDraftID(r.URL.Path)
	if draftID == "" {
		utils.BadRequest(w, "缺少草稿ID")
		return
	}
	if err := repository.DeleteVideoDraft(r.Context(), draftID, claims.UserID); err != nil {
		utils.InternalError(w, "删除草稿失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "草稿已删除"})
}

// extractDraftCoursewareID 从 /api/v1/coursewares/{id}/video-drafts[/...] 提取课件ID
func extractDraftCoursewareID(path string) string {
	const prefix = "/api/v1/coursewares/"
	const marker = "/video-drafts"
	trimmed := strings.TrimRight(path, "/")
	idx := strings.Index(trimmed, marker)
	if idx < 0 || !strings.HasPrefix(trimmed, prefix) {
		return ""
	}
	cwPart := trimmed[len(prefix):idx]
	if cwPart == "" || strings.Contains(cwPart, "/") {
		return ""
	}
	return cwPart
}

// extractDraftID 从 /api/v1/coursewares/{id}/video-drafts/{draft_id} 提取草稿ID
func extractDraftID(path string) string {
	const marker = "/video-drafts/"
	idx := strings.LastIndex(path, marker)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimRight(path[idx+len(marker):], "/")
	if rest == "" || strings.Contains(rest, "/") {
		return ""
	}
	return rest
}

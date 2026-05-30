package handlers

// video_edit_handler.go — 课件视频编辑HTTP处理器
//
// v0.42.4 新增：
//   POST /api/v1/coursewares/{id}/videos/mute           — 视频静音（去除音轨）
//   POST /api/v1/coursewares/{id}/videos/extract-audio   — 音轨分离（提取MP3）
//
// v0.42.1 原有：
//   POST /api/v1/coursewares/{id}/videos/concat          — 多视频拼接
//   POST /api/v1/coursewares/{id}/videos/trim            — 视频裁剪
//   POST /api/v1/coursewares/{id}/videos/advanced-concat  — 高级拼接(含转场)

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// VideoEditHandler 视频编辑处理器
type VideoEditHandler struct {
	editService *services.VideoEditService
}

// NewVideoEditHandler 创建视频编辑处理器
func NewVideoEditHandler(editService *services.VideoEditService) *VideoEditHandler {
	return &VideoEditHandler{editService: editService}
}

// ==================== 视频拼接 ====================

// ConcatVideos POST /api/v1/coursewares/{id}/videos/concat
// 请求体: { "asset_ids": ["id1", "id2", "id3"] }
func (h *VideoEditHandler) ConcatVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID := extractVideoEditCoursewareID(r.URL.Path, "/videos/concat")
	if cwID == "" {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	var req struct {
		AssetIDs []string `json:"asset_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if len(req.AssetIDs) < 2 {
		utils.BadRequest(w, "至少需要选择2个视频进行拼接")
		return
	}

	svcReq := &services.ConcatVideosRequest{
		CoursewareID: cwID,
		AssetIDs:     req.AssetIDs,
		UserID:       claims.UserID,
	}

	resp, err := h.editService.ConcatVideos(r.Context(), svcReq)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 视频裁剪 ====================

// TrimVideo POST /api/v1/coursewares/{id}/videos/trim
// 请求体: { "asset_id": "uuid", "start_sec": 1.5, "end_sec": 4.0 }
func (h *VideoEditHandler) TrimVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID := extractVideoEditCoursewareID(r.URL.Path, "/videos/trim")
	if cwID == "" {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	var req struct {
		AssetID  string  `json:"asset_id"`
		StartSec float64 `json:"start_sec"`
		EndSec   float64 `json:"end_sec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.AssetID == "" {
		utils.BadRequest(w, "asset_id不能为空")
		return
	}

	svcReq := &services.TrimVideoRequest{
		CoursewareID: cwID,
		AssetID:      req.AssetID,
		StartSec:     req.StartSec,
		EndSec:       req.EndSec,
		UserID:       claims.UserID,
	}

	resp, err := h.editService.TrimVideo(r.Context(), svcReq)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 高级拼接 ====================

// AdvancedConcat POST /api/v1/coursewares/{id}/videos/advanced-concat
// 请求体: { "clips": [{ "asset_id": "...", "start_sec": 0, "end_sec": 3.5, "transition": "fade", "trans_dur": 0.5 }, ...] }
func (h *VideoEditHandler) AdvancedConcat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID := extractVideoEditCoursewareID(r.URL.Path, "/videos/advanced-concat")
	if cwID == "" {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	var req struct {
		Clips []services.VideoClip `json:"clips"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if len(req.Clips) < 1 {
		utils.BadRequest(w, "至少需要1个视频片段")
		return
	}

	svcReq := &services.AdvancedConcatRequest{
		CoursewareID: cwID,
		Clips:        req.Clips,
		UserID:       claims.UserID,
	}

	resp, err := h.editService.AdvancedConcat(r.Context(), svcReq)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 视频静音（v0.42.4新增） ====================

// MuteVideo POST /api/v1/coursewares/{id}/videos/mute
// 请求体: { "asset_id": "uuid" }
// 去除视频音轨，生成新的静音视频资产
func (h *VideoEditHandler) MuteVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID := extractVideoEditCoursewareID(r.URL.Path, "/videos/mute")
	if cwID == "" {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	var req struct {
		AssetID string `json:"asset_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.AssetID == "" {
		utils.BadRequest(w, "asset_id不能为空")
		return
	}

	resp, err := h.editService.MuteVideo(r.Context(), &services.MuteVideoRequest{
		CoursewareID: cwID,
		AssetID:      req.AssetID,
		UserID:       claims.UserID,
	})
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 音轨分离（v0.42.4新增） ====================

// ExtractAudio POST /api/v1/coursewares/{id}/videos/extract-audio
// 请求体: { "asset_id": "uuid" }
// 从视频中分离音频轨道，输出MP3文件
func (h *VideoEditHandler) ExtractAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID := extractVideoEditCoursewareID(r.URL.Path, "/videos/extract-audio")
	if cwID == "" {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	var req struct {
		AssetID string `json:"asset_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.AssetID == "" {
		utils.BadRequest(w, "asset_id不能为空")
		return
	}

	resp, err := h.editService.ExtractAudio(r.Context(), &services.ExtractAudioRequest{
		CoursewareID: cwID,
		AssetID:      req.AssetID,
		UserID:       claims.UserID,
	})
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 路径解析 ====================

// extractVideoEditCoursewareID 从 /api/v1/coursewares/{id}/videos/{action} 提取课件ID
func extractVideoEditCoursewareID(path string, suffix string) string {
	if !strings.HasSuffix(path, suffix) && !strings.HasSuffix(path, suffix+"/") {
		return ""
	}
	trimmed := strings.TrimSuffix(strings.TrimSuffix(path, "/"), suffix)
	const prefix = "/api/v1/coursewares/"
	if !strings.HasPrefix(trimmed, prefix) {
		return ""
	}
	cwID := trimmed[len(prefix):]
	if cwID == "" || strings.Contains(cwID, "/") {
		return ""
	}
	return cwID
}

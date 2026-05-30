package handlers

// courseware_subtitle_handler.go — 课件字幕轨 HTTP 处理器
//
// v0.42.8 新增 5 个端点：
//   POST   /api/v1/coursewares/{id}/subtitles          — 创建/更新字幕轨
//   GET    /api/v1/coursewares/{id}/subtitles           — 查询字幕轨列表
//   DELETE /api/v1/coursewares/{id}/subtitles/{sub_id}  — 删除字幕轨
//   POST   /api/v1/coursewares/{id}/subtitles/{sub_id}/export-srt   — 导出 SRT
//   POST   /api/v1/coursewares/{id}/subtitles/{sub_id}/burn-in      — FFmpeg 硬字幕烧录

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CoursewareSubtitleHandler 字幕轨处理器
type CoursewareSubtitleHandler struct {
	subtitleService *services.CoursewareSubtitleService
}

// NewCoursewareSubtitleHandler 创建字幕轨处理器
func NewCoursewareSubtitleHandler(svc *services.CoursewareSubtitleService) *CoursewareSubtitleHandler {
	return &CoursewareSubtitleHandler{subtitleService: svc}
}

// ==================== 创建/更新字幕轨 ====================

// UpsertSubtitle POST /api/v1/coursewares/{id}/subtitles
func (h *CoursewareSubtitleHandler) UpsertSubtitle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST")
		return
	}

	// 从路径提取课件ID
	coursewareID := extractSubtitleCoursewareID(r.URL.Path)
	if coursewareID == "" {
		utils.BadRequest(w, "无效的课件ID")
		return
	}

	// 获取当前用户
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	// 校验课件存在和权限
	cw, err := repository.GetCoursewareByID(r.Context(), coursewareID)
	if err != nil {
		utils.Fail(w, http.StatusNotFound, "课件不存在")
		return
	}
	if cw.UserID != claims.UserID {
		utils.Forbidden(w, "无权操作此课件")
		return
	}

	// 解析请求体
	var req models.UpsertSubtitleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}

	// 校验必填字段
	if req.ScopeType == "" || req.Language == "" || req.Segments == "" {
		utils.BadRequest(w, "scope_type、language、segments 为必填字段")
		return
	}
	// 校验 scope_type 合法性
	if req.ScopeType != models.SubScopeVideoAsset &&
		req.ScopeType != models.SubScopeEditorDraft &&
		req.ScopeType != models.SubScopePage {
		utils.BadRequest(w, "scope_type 无效，应为 video_asset/editor_draft/page")
		return
	}
	// 校验 segments 是合法 JSON 数组
	var testParse []models.SubtitleSegment
	if err := json.Unmarshal([]byte(req.Segments), &testParse); err != nil {
		utils.BadRequest(w, "segments 不是合法的 JSON 数组")
		return
	}

	// 构建模型
	userID := claims.UserID
	sub := &models.CoursewareSubtitle{
		CoursewareID: coursewareID,
		ScopeType:    req.ScopeType,
		ScopeID:      req.ScopeID,
		Language:     req.Language,
		Segments:     req.Segments,
		StyleConfig:  req.StyleConfig,
		TTSConfig:    req.TTSConfig,
		CreatedBy:    &userID,
	}

	// UPSERT
	if err := repository.UpsertCoursewareSubtitle(r.Context(), sub); err != nil {
		utils.InternalError(w, "保存字幕失败: "+err.Error())
		return
	}

	utils.Success(w, sub)
}

// ==================== 查询字幕轨列表 ====================

// ListSubtitles GET /api/v1/coursewares/{id}/subtitles?scope_type=x&scope_id=y
func (h *CoursewareSubtitleHandler) ListSubtitles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET")
		return
	}

	coursewareID := extractSubtitleCoursewareID(r.URL.Path)
	if coursewareID == "" {
		utils.BadRequest(w, "无效的课件ID")
		return
	}

	scopeType := r.URL.Query().Get("scope_type")
	scopeID := r.URL.Query().Get("scope_id")

	subs, err := repository.ListCoursewareSubtitles(r.Context(), coursewareID, scopeType, scopeID)
	if err != nil {
		utils.InternalError(w, "查询字幕列表失败: "+err.Error())
		return
	}
	if subs == nil {
		subs = []*models.CoursewareSubtitle{}
	}

	utils.Success(w, subs)
}

// ==================== 删除字幕轨 ====================

// DeleteSubtitle DELETE /api/v1/coursewares/{id}/subtitles/{sub_id}
func (h *CoursewareSubtitleHandler) DeleteSubtitle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE")
		return
	}

	coursewareID := extractSubtitleCoursewareID(r.URL.Path)
	subID := extractSubtitleID(r.URL.Path)
	if coursewareID == "" || subID == "" {
		utils.BadRequest(w, "无效的路径参数")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	// 校验课件权限
	cw, err := repository.GetCoursewareByID(r.Context(), coursewareID)
	if err != nil {
		utils.Fail(w, http.StatusNotFound, "课件不存在")
		return
	}
	if cw.UserID != claims.UserID {
		utils.Forbidden(w, "无权操作此课件")
		return
	}

	// 校验字幕属于此课件
	sub, err := repository.GetCoursewareSubtitleByID(r.Context(), subID)
	if err != nil {
		utils.Fail(w, http.StatusNotFound, "字幕不存在")
		return
	}
	if sub.CoursewareID != coursewareID {
		utils.Forbidden(w, "字幕不属于此课件")
		return
	}

	if err := repository.DeleteCoursewareSubtitle(r.Context(), subID); err != nil {
		utils.InternalError(w, "删除字幕失败: "+err.Error())
		return
	}

	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 导出 SRT ====================

// ExportSRT POST /api/v1/coursewares/{id}/subtitles/{sub_id}/export-srt
func (h *CoursewareSubtitleHandler) ExportSRT(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST")
		return
	}

	subID := extractSubtitleID(r.URL.Path)
	if subID == "" {
		utils.BadRequest(w, "无效的字幕ID")
		return
	}

	srtContent, err := h.subtitleService.ExportSRT(r.Context(), subID)
	if err != nil {
		utils.InternalError(w, "导出SRT失败: "+err.Error())
		return
	}

	// 返回 SRT 文本文件
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=subtitle.srt")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(srtContent))
}

// ==================== 硬字幕烧录 ====================

// BurnInSubtitle POST /api/v1/coursewares/{id}/subtitles/{sub_id}/burn-in
func (h *CoursewareSubtitleHandler) BurnInSubtitle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST")
		return
	}

	coursewareID := extractSubtitleCoursewareID(r.URL.Path)
	subID := extractSubtitleID(r.URL.Path)
	if coursewareID == "" || subID == "" {
		utils.BadRequest(w, "无效的路径参数")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	var req models.BurnInSubtitleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	if req.VideoAssetID == "" {
		utils.BadRequest(w, "video_asset_id 为必填")
		return
	}

	result, err := h.subtitleService.BurnInSubtitle(r.Context(), subID, req.VideoAssetID, coursewareID, claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, result)
}


// ==================== v0.42.9 TTS 配音 ====================

// GenerateTTS POST /api/v1/coursewares/{id}/subtitles/{sub_id}/generate-tts
func (h *CoursewareSubtitleHandler) GenerateTTS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST")
		return
	}

	coursewareID := extractSubtitleCoursewareID(r.URL.Path)
	subID := extractSubtitleID(r.URL.Path)
	if coursewareID == "" || subID == "" {
		utils.BadRequest(w, "无效的路径参数")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	var req models.GenerateTTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	if req.Voice == "" {
		utils.BadRequest(w, "voice（音色代码）为必填")
		return
	}

	result, err := h.subtitleService.GenerateTTS(r.Context(), subID, coursewareID, claims.UserID, req.Voice, req.Speed, req.SegmentIDs)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, result)
}

// ListTTSVoices GET /api/v1/tts-voices?language=zh-CN
func (h *CoursewareSubtitleHandler) ListTTSVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET")
		return
	}

	language := r.URL.Query().Get("language")
	voices := ai.GetTTSVoicesByLanguage(language)

	utils.Success(w, map[string]interface{}{
		"voices": voices,
		"total":  len(voices),
	})
}

// ==================== 路径解析辅助函数 ====================

// extractSubtitleCoursewareID 从 /api/v1/coursewares/{id}/subtitles... 提取课件ID
func extractSubtitleCoursewareID(path string) string {
	// 路径格式: /api/v1/coursewares/{courseware_id}/subtitles[/...]
	const prefix = "/api/v1/coursewares/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := path[len(prefix):]
	// 找到下一个 /
	slashIdx := strings.Index(rest, "/")
	if slashIdx <= 0 {
		return ""
	}
	return rest[:slashIdx]
}

// extractSubtitleID 从 /api/v1/coursewares/{id}/subtitles/{sub_id}[/...] 提取字幕ID
func extractSubtitleID(path string) string {
	// 路径格式: /api/v1/coursewares/{cw_id}/subtitles/{sub_id}[/export-srt|/burn-in]
	idx := strings.Index(path, "/subtitles/")
	if idx < 0 {
		return ""
	}
	rest := path[idx+len("/subtitles/"):]
	// 去掉后缀 /export-srt 或 /burn-in
	slashIdx := strings.Index(rest, "/")
	if slashIdx > 0 {
		return rest[:slashIdx]
	}
	// 去除尾部斜杠
	return strings.TrimSuffix(rest, "/")
}

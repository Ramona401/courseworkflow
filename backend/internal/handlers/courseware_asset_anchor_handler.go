package handlers

// courseware_asset_anchor_handler.go — 课件「风格锚点」HTTP处理器（VAOCI 课程级风格一致性，轮2）
//
// 拆分说明：方法挂在 CoursewareAssetHandler 上（与 courseware_asset_handler.go 同类型、同包），
//   单独成文件是为给主 handler 文件（已超600行约定）减负，符合既有拆分范式。
//
// 端点：
//   POST   /api/v1/coursewares/{id}/style-anchor   — 设置风格锚点（一步式同步：取URL→提取VAOCI→落库）
//   DELETE /api/v1/coursewares/{id}/style-anchor   — 清除风格锚点
//
// 查锚点：无独立端点——前端直接读 GET /api/v1/coursewares/{id} 详情中的
//   style_anchor_asset_id / style_anchor_vaoci 字段（轮1契约①已在装配处补齐）。
//
// 设锚点为多模态读图调用，耗时数秒到十几秒，前端以 loading 兜底。

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/utils"
)

// SetStyleAnchor POST /api/v1/coursewares/{id}/style-anchor
//
//	请求体: { "asset_id": "uuid" }   — 要设为锚点的图片资产ID（须属于本课件、且为图片）
//	响应:   { "asset_id", "anchor_url", "vaoci" }
//
// 一步式同步：内部完成「校验资产归属 → 取公网URL → 多模态提取VAOCI → 落库」。
func (h *CoursewareAssetHandler) SetStyleAnchor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID := extractAnchorCoursewareID(r.URL.Path)
	if cwID == "" {
		utils.BadRequest(w, "缺少课件ID")
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

	result, err := h.assetService.SetStyleAnchor(r.Context(), cwID, req.AssetID, claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, result)
}

// ClearStyleAnchor DELETE /api/v1/coursewares/{id}/style-anchor
// 清除课件当前的风格锚点（两字段置NULL）。
func (h *CoursewareAssetHandler) ClearStyleAnchor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID := extractAnchorCoursewareID(r.URL.Path)
	if cwID == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	if err := h.assetService.ClearStyleAnchor(r.Context(), cwID, claims.UserID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "风格锚点已清除"})
}

// extractAnchorCoursewareID 从 /api/v1/coursewares/{id}/style-anchor 提取课件ID
func extractAnchorCoursewareID(path string) string {
	const suffix = "/style-anchor"
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

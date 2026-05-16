package handlers

// lesson_plan_asset_handler.go — 教案附属资产 HTTP 处理器
//
// v123 新增:5 个 REST 接口
//   POST   /api/v1/lesson-plans/plans/{id}/assets         — 上传图片(multipart)
//   GET    /api/v1/lesson-plans/plans/{id}/assets         — 列出该教案所有资产
//   GET    /api/v1/lesson-plans/assets/{asset_id}         — 单条资产详情
//   PUT    /api/v1/lesson-plans/assets/{asset_id}         — 更新 alt 文本
//   DELETE /api/v1/lesson-plans/assets/{asset_id}         — 删除资产

import (
        "encoding/json"
        "errors"
        "net/http"
        "strings"

        "tedna/internal/middleware"
        "tedna/internal/models"
        "tedna/internal/services"
        "tedna/internal/utils"
)

// LessonPlanAssetHandler 资产处理器
type LessonPlanAssetHandler struct {
        assetService *services.LessonPlanAssetService
}

// NewLessonPlanAssetHandler 创建处理器
func NewLessonPlanAssetHandler(svc *services.LessonPlanAssetService) *LessonPlanAssetHandler {
        return &LessonPlanAssetHandler{assetService: svc}
}

// ==================== 上传图片 ====================

// UploadAsset POST /api/v1/lesson-plans/plans/{id}/assets
// multipart/form-data 字段:
//   - file:图片文件(必填)
//   - alt_text:图片描述(可选,空则用原文件名)
func (h *LessonPlanAssetHandler) UploadAsset(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, utils.MsgNotLoggedIn)
                return
        }

        planID := extractPlanIDFromAssetsPath(r.URL.Path)
        if planID == "" {
                utils.BadRequest(w, utils.MsgMissingLessonPlanID)
                return
        }

        // 解析 multipart(最大 5MB)
        if err := r.ParseMultipartForm(services.MaxAssetFileSize); err != nil {
                utils.BadRequest(w, "文件过大或格式无效")
                return
        }

        file, header, err := r.FormFile("file")
        if err != nil {
                utils.BadRequest(w, "请选择要上传的图片文件")
                return
        }
        defer file.Close()

        altText := r.FormValue("alt_text")

        resp, err := h.assetService.UploadAsset(r.Context(), planID, file, header, altText, claims.UserID)
        if err != nil {
                handleAssetError(w, err)
                return
        }
        utils.Success(w, resp)
}

// ==================== 列表 ====================

// ListAssets GET /api/v1/lesson-plans/plans/{id}/assets
func (h *LessonPlanAssetHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
                return
        }
        planID := extractPlanIDFromAssetsPath(r.URL.Path)
        if planID == "" {
                utils.BadRequest(w, utils.MsgMissingLessonPlanID)
                return
        }
        resp, err := h.assetService.ListAssets(r.Context(), planID)
        if err != nil {
                handleAssetError(w, err)
                return
        }
        utils.Success(w, resp)
}

// ==================== 详情 ====================

// GetAsset GET /api/v1/lesson-plans/assets/{asset_id}
func (h *LessonPlanAssetHandler) GetAsset(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
                return
        }
        assetID := extractAssetID(r.URL.Path)
        if assetID == "" {
                utils.BadRequest(w, "缺少资产 ID")
                return
        }
        item, err := h.assetService.GetAsset(r.Context(), assetID)
        if err != nil {
                handleAssetError(w, err)
                return
        }
        utils.Success(w, item)
}

// ==================== 更新 alt ====================

// UpdateAsset PUT /api/v1/lesson-plans/assets/{asset_id}
func (h *LessonPlanAssetHandler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPut {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, utils.MsgNotLoggedIn)
                return
        }
        assetID := extractAssetID(r.URL.Path)
        if assetID == "" {
                utils.BadRequest(w, "缺少资产 ID")
                return
        }
        var req models.UpdateAssetRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, utils.MsgBadRequestBody)
                return
        }
        if err := h.assetService.UpdateAssetAltText(r.Context(), assetID, req.AltText, claims.UserID); err != nil {
                handleAssetError(w, err)
                return
        }
        utils.Success(w, map[string]string{"message": "更新成功"})
}

// ==================== 删除 ====================

// DeleteAsset DELETE /api/v1/lesson-plans/assets/{asset_id}
func (h *LessonPlanAssetHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodDelete {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, utils.MsgNotLoggedIn)
                return
        }
        assetID := extractAssetID(r.URL.Path)
        if assetID == "" {
                utils.BadRequest(w, "缺少资产 ID")
                return
        }
        if err := h.assetService.DeleteAsset(r.Context(), assetID, claims.UserID); err != nil {
                handleAssetError(w, err)
                return
        }
        utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 路径解析辅助 ====================

// extractPlanIDFromAssetsPath 从路径 /api/v1/lesson-plans/plans/{id}/assets 提取 plan ID
// 兼容路径末尾有无斜杠
func extractPlanIDFromAssetsPath(path string) string {
        prefix := "/api/v1/lesson-plans/plans/"
        if !strings.HasPrefix(path, prefix) {
                return ""
        }
        rest := strings.TrimPrefix(path, prefix)
        rest = strings.TrimSuffix(rest, "/")
        // rest 应是 "{plan_id}/assets" 形式
        idx := strings.Index(rest, "/assets")
        if idx <= 0 {
                return ""
        }
        return rest[:idx]
}

// extractAssetID 从路径 /api/v1/lesson-plans/assets/{asset_id} 提取 asset ID
func extractAssetID(path string) string {
        prefix := "/api/v1/lesson-plans/assets/"
        if !strings.HasPrefix(path, prefix) {
                return ""
        }
        rest := strings.TrimPrefix(path, prefix)
        rest = strings.TrimSuffix(rest, "/")
        // 防止 rest 包含子路径(如未来扩展)
        if idx := strings.Index(rest, "/"); idx > 0 {
                rest = rest[:idx]
        }
        return rest
}

// ==================== 错误处理 ====================

func handleAssetError(w http.ResponseWriter, err error) {
        switch {
        case errors.Is(err, services.ErrAssetNotFound),
                errors.Is(err, services.ErrAssetPlanNotFound):
                utils.Fail(w, http.StatusNotFound, err.Error())
        case errors.Is(err, services.ErrAssetUnauthorized),
                errors.Is(err, services.ErrAssetNotPlanAuthor):
                utils.Forbidden(w, err.Error())
        case errors.Is(err, services.ErrAssetFileInvalid),
                errors.Is(err, services.ErrAssetFileTooLarge):
                utils.BadRequest(w, err.Error())
        default:
                utils.InternalError(w, "操作失败: "+err.Error())
        }
}

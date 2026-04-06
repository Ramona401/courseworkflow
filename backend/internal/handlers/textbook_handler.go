package handlers

// textbook_handler.go — 课本页面图片HTTP处理器
//
// 迭代7新增：6个REST接口
//   POST   /api/v1/lesson-plans/textbooks/upload     — 上传课本图片（multipart）
//   GET    /api/v1/lesson-plans/textbooks             — 列表查询
//   GET    /api/v1/lesson-plans/textbooks/{id}        — 获取详情
//   PUT    /api/v1/lesson-plans/textbooks/{id}        — 更新元数据
//   DELETE /api/v1/lesson-plans/textbooks/{id}        — 删除
//   POST   /api/v1/lesson-plans/textbooks/{id}/ocr    — 触发AI OCR识别

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// TextbookHandler 课本页面处理器
type TextbookHandler struct {
	tbService *services.TextbookService
}

// NewTextbookHandler 创建课本处理器
func NewTextbookHandler(ts *services.TextbookService) *TextbookHandler {
	return &TextbookHandler{tbService: ts}
}

// ==================== 上传课本图片 ====================

// UploadTextbook POST /api/v1/lesson-plans/textbooks/upload
// multipart/form-data 格式：file(图片文件) + 元数据字段
func (h *TextbookHandler) UploadTextbook(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 解析 multipart 表单（最大10MB）
	if err := r.ParseMultipartForm(services.MaxTextbookFileSize); err != nil {
		utils.BadRequest(w, "文件过大或格式无效")
		return
	}

	// 获取文件
	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequest(w, "请选择要上传的图片文件")
		return
	}
	defer file.Close()

	// 从表单字段构建请求
	pageNumber, _ := strconv.Atoi(r.FormValue("page_number"))
	req := &models.UploadTextbookRequest{
		Subject:      r.FormValue("subject"),
		GradeRange:   r.FormValue("grade_range"),
		TextbookName: r.FormValue("textbook_name"),
		Chapter:      r.FormValue("chapter"),
		PageNumber:   pageNumber,
		Description:  r.FormValue("description"),
		Scope:        r.FormValue("scope"),
		ScopeRefID:   r.FormValue("scope_ref_id"),
	}

	page, err := h.tbService.UploadTextbookPage(r.Context(), file, header, req, claims.UserID)
	if err != nil {
		handleTextbookError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"id":        page.ID,
		"file_name": page.FileName,
		"file_size": page.FileSize,
		"image_url": "/uploads/textbooks/" + page.FilePath,
		"message":   "上传成功",
	})
}

// ==================== 列表查询 ====================

// ListTextbooks GET /api/v1/lesson-plans/textbooks
func (h *TextbookHandler) ListTextbooks(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	q := r.URL.Query()
	subject := q.Get("subject")
	gradeRange := q.Get("grade_range")
	textbookName := q.Get("textbook_name")
	scope := q.Get("scope")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	resp, err := h.tbService.ListTextbookPages(r.Context(), claims.UserID, subject, gradeRange, textbookName, scope, limit, offset)
	if err != nil {
		utils.InternalError(w, "查询课本列表失败")
		return
	}
	utils.Success(w, resp)
}

// ==================== 获取详情 ====================

// GetTextbook GET /api/v1/lesson-plans/textbooks/{id}
func (h *TextbookHandler) GetTextbook(w http.ResponseWriter, r *http.Request) {
	tbID := extractTextbookID(r.URL.Path)
	if tbID == "" {
		utils.BadRequest(w, "ID无效")
		return
	}

	resp, err := h.tbService.GetTextbookPage(r.Context(), tbID)
	if err != nil {
		handleTextbookError(w, err)
		return
	}
	utils.Success(w, resp)
}

// ==================== 更新元数据 ====================

// UpdateTextbook PUT /api/v1/lesson-plans/textbooks/{id}
func (h *TextbookHandler) UpdateTextbook(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	tbID := extractTextbookID(r.URL.Path)
	if tbID == "" {
		utils.BadRequest(w, "ID无效")
		return
	}

	var req models.UpdateTextbookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数无效")
		return
	}

	if err := h.tbService.UpdateTextbookPage(r.Context(), tbID, &req, claims.UserID); err != nil {
		handleTextbookError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// ==================== 删除 ====================

// DeleteTextbook DELETE /api/v1/lesson-plans/textbooks/{id}
func (h *TextbookHandler) DeleteTextbook(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	tbID := extractTextbookID(r.URL.Path)
	if tbID == "" {
		utils.BadRequest(w, "ID无效")
		return
	}

	if err := h.tbService.DeleteTextbookPage(r.Context(), tbID, claims.UserID); err != nil {
		handleTextbookError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 触发OCR识别 ====================

// TriggerOCR POST /api/v1/lesson-plans/textbooks/{id}/ocr
func (h *TextbookHandler) TriggerOCR(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	tbID := extractTextbookOCRID(r.URL.Path)
	if tbID == "" {
		utils.BadRequest(w, "ID无效")
		return
	}

	ocrText, err := h.tbService.RecognizeTextbookPage(r.Context(), tbID, claims.UserID)
	if err != nil {
		handleTextbookError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{
		"ocr_text": ocrText,
		"message":  "识别完成",
	})
}

// ==================== 辅助函数 ====================

// extractTextbookID 从路径 .../textbooks/{id} 提取末尾ID
func extractTextbookID(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) < 1 {
		return ""
	}
	id := parts[len(parts)-1]
	if len(id) < 10 {
		return ""
	}
	return id
}

// extractTextbookOCRID 从路径 .../textbooks/{id}/ocr 提取ID
func extractTextbookOCRID(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i, p := range parts {
		if p == "textbooks" && i+1 < len(parts) {
			id := parts[i+1]
			if len(id) >= 10 {
				return id
			}
		}
	}
	return ""
}

// handleTextbookError 统一错误处理
func handleTextbookError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrTextbookNotFound):
		utils.Fail(w, 404, "课本页面不存在")
	case errors.Is(err, services.ErrTextbookUnauthorized):
		utils.Fail(w, 403, "无权操作此课本页面")
	case errors.Is(err, services.ErrTextbookFileInvalid):
		utils.BadRequest(w, "文件格式无效，仅支持JPG/PNG/WEBP图片")
	case errors.Is(err, services.ErrTextbookFileTooLarge):
		utils.BadRequest(w, "文件过大，最大支持10MB")
	default:
		errMsg := err.Error()
		if strings.Contains(errMsg, "不能为空") {
			utils.BadRequest(w, errMsg)
		} else {
			utils.InternalError(w, "操作失败: "+errMsg)
		}
	}
}

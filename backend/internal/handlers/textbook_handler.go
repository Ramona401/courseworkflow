package handlers

// textbook_handler.go — 课本页面图片HTTP处理器
//
// 迭代7新增：6个REST接口
//   POST   /api/v1/lesson-plans/textbooks/upload     — 上传课本图片（multipart）
//   GET    /api/v1/lesson-plans/textbooks            — 列表查询
//   GET    /api/v1/lesson-plans/textbooks/{id}       — 获取详情
//   PUT    /api/v1/lesson-plans/textbooks/{id}       — 更新元数据
//   DELETE /api/v1/lesson-plans/textbooks/{id}       — 删除
//   POST   /api/v1/lesson-plans/textbooks/{id}/ocr   — 触发AI OCR识别
//
// v231新增：教材照片归档维度扩展
//   - 上传读取表单的 semester(学期) + unit(单元)
//   - 列表查询接收 semester/unit 两个筛选参数并透传给 service
//
// 上下文15新增：
//   - 上传在ParseMultipartForm前执行K12前置授权，避免无权请求占用内存和临时文件
//   - 详情直接ID读取也必须携带当前登录用户并执行K12教育域校验
//   - 非K12列表由Service返回成功空数组；详情和所有写操作统一返回403
//   - Handler不信任任何education_domain查询参数或请求体字段

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

	// 在解析multipart和创建临时文件前先做K12前置授权。
	//
	// 业务Service内部仍会再次校验，本次预检只负责尽早拒绝无权请求。
	if err := h.tbService.AuthorizeK12TextbookWrite(r.Context(), claims.UserID); err != nil {
		handleTextbookError(w, err)
		return
	}

	// 解析 multipart 表单（最大10MB）
	if err := r.ParseMultipartForm(services.MaxTextbookFileSize); err != nil {
		utils.BadRequest(w, "文件过大或格式无效")
		return
	}

	// ParseMultipartForm可能把超过内存阈值的文件部分写入临时目录。
	//
	// 从这里开始，无论后续是缺少file字段、业务校验失败、
	// 数据库写入失败还是上传成功，都必须删除multipart临时文件。
	// file.Close在后面注册，defer按后进先出执行，因此会先关闭文件，
	// 再安全执行RemoveAll清理临时目录。
	if r.MultipartForm != nil {
		defer func() {
			_ = r.MultipartForm.RemoveAll()
		}()
	}

	// 获取文件
	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequest(w, "请选择要上传的图片文件")
		return
	}
	defer file.Close()

	// 从表单字段构建请求（v231：新增 semester 学期 + unit 单元）
	pageNumber, _ := strconv.Atoi(r.FormValue("page_number"))
	req := &models.UploadTextbookRequest{
		Subject:      r.FormValue("subject"),
		GradeRange:   r.FormValue("grade_range"),
		Semester:     r.FormValue("semester"),
		Unit:         r.FormValue("unit"),
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
		"image_url": "/api/v1/lesson-plans/textbooks/" + page.ID + "/image",
		"message":   "上传成功",
	})
}

// ==================== 列表查询 ====================

// ListTextbooks GET /api/v1/lesson-plans/textbooks
// v231：新增 semester(学期) + unit(单元) 两个筛选参数
func (h *TextbookHandler) ListTextbooks(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	q := r.URL.Query()
	subject := q.Get("subject")
	gradeRange := q.Get("grade_range")
	semester := q.Get("semester")
	unit := q.Get("unit")
	textbookName := q.Get("textbook_name")
	scope := q.Get("scope")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	resp, err := h.tbService.ListTextbookPages(r.Context(), claims.UserID, subject, gradeRange, semester, unit, textbookName, scope, limit, offset)
	if err != nil {
		utils.InternalError(w, "查询课本列表失败")
		return
	}
	utils.Success(w, resp)
}

// ==================== 获取详情 ====================

// GetTextbook GET /api/v1/lesson-plans/textbooks/{id}
func (h *TextbookHandler) GetTextbook(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.tbService.GetTextbookPage(r.Context(), tbID, claims.UserID)
	if err != nil {
		handleTextbookError(w, err)
		return
	}
	utils.Success(w, resp)
}

// ==================== 获取鉴权图片 ====================

// GetTextbookImage GET /api/v1/lesson-plans/textbooks/{id}/image
//
// 与公开静态文件不同，本端点必须经过AuthMiddleware，
// Service还会实时重新解析用户教育域和课本active状态。
// 非K12、伪造ID、已归档记录和路径异常都不能读取原图。
func (h *TextbookHandler) GetTextbookImage(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok :=
		middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	textbookID :=
		extractTextbookImageID(
			r.URL.Path,
		)
	if textbookID == "" {
		utils.BadRequest(w, "ID无效")
		return
	}

	file,
		fileInfo,
		mimeType,
		err :=
		h.tbService.OpenTextbookImage(
			r.Context(),
			textbookID,
			claims.UserID,
		)
	if err != nil {
		handleTextbookError(w, err)
		return
	}
	defer file.Close()

	// 禁止浏览器和中间代理长期缓存。
	//
	// 用户教育域或课本状态变化后，下一次展示必须重新经过后端授权。
	w.Header().Set(
		"Cache-Control",
		"private, no-store, max-age=0",
	)
	w.Header().Set(
		"Pragma",
		"no-cache",
	)
	w.Header().Set(
		"X-Content-Type-Options",
		"nosniff",
	)
	w.Header().Set(
		"Content-Type",
		mimeType,
	)

	http.ServeContent(
		w,
		r,
		fileInfo.Name(),
		fileInfo.ModTime(),
		file,
	)
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

// extractTextbookImageID 从路径
// .../textbooks/{id}/image 中提取正式课本ID。
func extractTextbookImageID(
	path string,
) string {
	parts := strings.Split(
		strings.TrimSuffix(path, "/"),
		"/",
	)

	for index, part := range parts {
		if part != "textbooks" ||
			index+2 >= len(parts) ||
			parts[index+2] != "image" {
			continue
		}

		id := parts[index+1]
		if len(id) >= 10 {
			return id
		}
	}

	return ""
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
		utils.Fail(w, http.StatusNotFound, "课本页面不存在")
	case errors.Is(err, services.ErrTextbookUnauthorized):
		utils.Fail(w, http.StatusForbidden, "无权操作此课本页面")
	case errors.Is(err, services.ErrTextbookK12Only):
		utils.Fail(w, http.StatusForbidden, "当前教育域暂无课本能力")
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

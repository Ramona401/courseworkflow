package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== 课件工坊HTTP处理器 ====================
// 课件CRUD + 页面操作 + 状态流转 + 风格模板 + Logo上传
// Phase 4A: 新增UploadLogo/SaveStyleFull/ConfirmStyle

// CoursewareHandler 课件工坊处理器
type CoursewareHandler struct {
	cwService *services.CoursewareService
}

// NewCoursewareHandler 创建课件工坊处理器
func NewCoursewareHandler(cwService *services.CoursewareService) *CoursewareHandler {
	return &CoursewareHandler{cwService: cwService}
}

// ==================== 课件CRUD ====================

// ListCoursewares GET /api/v1/coursewares — 我的课件列表
func (h *CoursewareHandler) ListCoursewares(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	status := r.URL.Query().Get("status")
	subject := r.URL.Query().Get("subject")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	resp, err := h.cwService.ListCoursewares(r.Context(), claims.UserID, status, subject, limit, offset)
	if err != nil {
		utils.InternalError(w, "查询课件列表失败: "+err.Error())
		return
	}
	utils.Success(w, resp)
}

// CreateCourseware POST /api/v1/coursewares — 创建课件
func (h *CoursewareHandler) CreateCourseware(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.CreateCoursewareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	cw, err := h.cwService.CreateCourseware(
		r.Context(),
		actor,
		&req,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			services.ErrCoursewareLessonPlanRequired,
		):
			utils.BadRequest(w, err.Error())

		case errors.Is(
			err,
			services.ErrCoursewareLessonPlanNotFound,
		):
			utils.Fail(
				w,
				http.StatusNotFound,
				err.Error(),
			)

		case errors.Is(
			err,
			services.ErrCoursewareActorRequired,
		),
			errors.Is(
				err,
				services.ErrCoursewareCreationDomainRequired,
			),
			errors.Is(
				err,
				services.ErrCoursewareLessonPlanNotOwned,
			):
			utils.Fail(
				w,
				http.StatusForbidden,
				err.Error(),
			)

		case errors.Is(
			err,
			services.ErrCoursewareLessonPlanDomainInvalid,
		):
			utils.InternalError(
				w,
				"关联教案教育域异常，请联系管理员处理",
			)

		default:
			utils.InternalError(
				w,
				"创建课件失败: "+err.Error(),
			)
		}
		return
	}

	utils.Success(w, cw)
}

// GetCourseware GET /api/v1/coursewares/{id} — 课件详情
func (h *CoursewareHandler) GetCourseware(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	resp, err := h.cwService.GetCoursewareForView(
		r.Context(),
		id,
		actor,
	)
	if err != nil {
		handleCoursewareAccessError(
			w,
			err,
			"获取课件详情失败",
		)
		return
	}

	utils.Success(w, resp)
}

// UpdateCourseware PUT /api/v1/coursewares/{id} — 更新课件标题
func (h *CoursewareHandler) UpdateCourseware(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	var req models.UpdateCoursewareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.UpdateCoursewareTitleForActor(
		r.Context(),
		id,
		actor,
		req.Title,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "更新成功"})
}

// DeleteCourseware DELETE /api/v1/coursewares/{id} — 删除课件
func (h *CoursewareHandler) DeleteCourseware(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	if err := h.cwService.DeleteCoursewareForActor(
		r.Context(),
		id,
		actor,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 课件页面操作 ====================

// GetCoursewarePages GET /api/v1/coursewares/{id}/pages — 获取全部页面
func (h *CoursewareHandler) GetCoursewarePages(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/pages",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	pages, err := h.cwService.GetPagesForView(
		r.Context(),
		id,
		actor,
	)
	if err != nil {
		handleCoursewareAccessError(
			w,
			err,
			"获取页面列表失败",
		)
		return
	}

	utils.Success(w, pages)
}

// UpdatePageIndex PUT /api/v1/coursewares/{id}/pages/{num} — 更新单页索引
func (h *CoursewareHandler) UpdatePageIndex(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCoursewarePagePath(r.URL.Path)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		coursewareID,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	var req models.UpdateCWPageIndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.UpdatePageIndexForActor(
		r.Context(),
		coursewareID,
		actor,
		pageNumber,
		&req,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "更新成功"})
}

// addCWPageRequest 扩展原有新增页面请求，增加指定插入位置。
//
// 为保持旧客户端兼容：
//   - insert_at缺失或小于等于0时，服务端自动追加到最后；
//   - insert_at有效时，新页面直接成为该页，原页面整体后移。
//
// 使用匿名嵌入后，原有页面方案字段仍保持平铺JSON格式。
type addCWPageRequest struct {
	models.AddCWPageRequest
	InsertAt int `json:"insert_at"`
}

// AddPage POST /api/v1/coursewares/{id}/pages — 在指定位置添加页面
func (h *CoursewareHandler) AddPage(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/pages",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	var req addCWPageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	page, err := h.cwService.AddPageAtForActor(
		r.Context(),
		id,
		actor,
		&req.AddCWPageRequest,
		req.InsertAt,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(w, page)
}

// ReorderPages PUT /api/v1/coursewares/{id}/pages/reorder — 页面排序
func (h *CoursewareHandler) ReorderPages(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/pages/reorder",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	var req models.ReorderCWPagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.ReorderPagesForActor(
		r.Context(),
		id,
		actor,
		req.PageIDs,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "排序成功"})
}

// ==================== 状态流转 ====================

// ConfirmIndex POST /api/v1/coursewares/{id}/confirm-index — 确认索引
func (h *CoursewareHandler) ConfirmIndex(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/confirm-index",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	if err := h.cwService.ConfirmIndexForActor(
		r.Context(),
		id,
		actor,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "索引确认成功，请选择风格",
		},
	)
}

// SaveStyle PUT /api/v1/coursewares/{id}/style — 保存风格（兼容旧接口）
func (h *CoursewareHandler) SaveStyle(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/style",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	var req models.UpdateCoursewareStyleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.SaveStyleForActor(
		r.Context(),
		id,
		actor,
		req.StyleConfig,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "风格保存成功"})
}

// SaveStyleFull POST /api/v1/coursewares/{id}/save-style — Phase 4A: 保存完整风格配置
func (h *CoursewareHandler) SaveStyleFull(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/save-style",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	var req models.SaveStyleFullRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.SaveStyleFullForActor(
		r.Context(),
		id,
		actor,
		&req,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "风格配置保存成功",
		},
	)
}

// ConfirmStyle POST /api/v1/coursewares/{id}/confirm-style — Phase 4A: 确认风格
func (h *CoursewareHandler) ConfirmStyle(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/confirm-style",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	if err := h.cwService.ConfirmStyleForActor(
		r.Context(),
		id,
		actor,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "风格确认成功，准备生成课件",
		},
	)
}

// UploadLogo POST /api/v1/coursewares/{id}/upload-logo — Phase 4A: 上传Logo
func (h *CoursewareHandler) UploadLogo(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/upload-logo",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	// 必须在作者域预检通过后才解析multipart并打开上传文件。
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		utils.BadRequest(
			w,
			"文件解析失败: "+err.Error(),
		)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequest(w, "缺少文件字段 file")
		return
	}
	defer file.Close()

	response, err := h.cwService.UploadLogoForActor(
		r.Context(),
		id,
		actor,
		file,
		header,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(w, response)
}

// ConfirmCourseware POST /api/v1/coursewares/{id}/confirm — 确认课件
func (h *CoursewareHandler) ConfirmCourseware(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/confirm",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	if err := h.cwService.ConfirmCoursewareForActor(
		r.Context(),
		id,
		actor,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "课件确认成功",
		},
	)
}

// ==================== 风格模板查询 ====================

// ListTemplates GET /api/v1/courseware-templates — 获取风格模板列表
func (h *CoursewareHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	templates, err := services.ListCWTemplates(r.Context(), true)
	if err != nil {
		utils.InternalError(w, "获取风格模板失败: "+err.Error())
		return
	}
	utils.Success(w, templates)
}

// GetTemplatePreview GET /api/v1/courseware-templates/{id}/preview — 模板样例预览
func (h *CoursewareHandler) GetTemplatePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractCWTemplateID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少模板ID")
		return
	}

	t, err := services.GetCWTemplateByID(r.Context(), id)
	if err != nil {
		utils.InternalError(w, "模板不存在: "+err.Error())
		return
	}
	utils.Success(w, t)
}

// ListLogoHistory GET /api/v1/coursewares/logo-history — 查询当前用户历史用过的 Logo
// 需求2：风格页"历史 Logo 复用"，去重、最近优先，避免每次重新上传
func (h *CoursewareHandler) ListLogoHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	urls, err := repository.ListUserLogoURLs(r.Context(), claims.UserID, limit)
	if err != nil {
		utils.InternalError(w, "查询历史Logo失败: "+err.Error())
		return
	}
	if urls == nil {
		urls = []string{}
	}
	utils.Success(w, map[string]interface{}{"logos": urls})
}

// DeleteLogoHistory DELETE /api/v1/coursewares/logo-history?url=xxx — 删除一条历史 Logo
// 需求2：清空当前用户名下所有使用该 logo_url 的课件的 Logo，使其不再出现在历史中（由用户自行判断）
func (h *CoursewareHandler) DeleteLogoHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	logoURL := r.URL.Query().Get("url")
	if strings.TrimSpace(logoURL) == "" {
		utils.BadRequest(w, "缺少 url 参数")
		return
	}
	affected, err := repository.DeleteUserLogoURL(r.Context(), claims.UserID, logoURL)
	if err != nil {
		utils.InternalError(w, "删除历史Logo失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"affected": affected, "message": "已从历史中删除"})
}

// ==================== 路径解析辅助函数 ====================

// extractCoursewareID 从 /api/v1/coursewares/{id} 提取ID
func extractCoursewareID(path string) string {
	const prefix = "/api/v1/coursewares/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := path[len(prefix):]
	rest = strings.TrimRight(rest, "/")
	if strings.Contains(rest, "/") {
		return ""
	}
	return rest
}

// extractCoursewareMiddleID 从 /api/v1/coursewares/{id}/{suffix} 提取ID
func extractCoursewareMiddleID(path string, suffix string) string {
	const prefix = "/api/v1/coursewares/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := path[len(prefix):]
	idx := strings.Index(rest, suffix)
	if idx <= 0 {
		return ""
	}
	return rest[:idx]
}

// extractCoursewarePagePath 从 /api/v1/coursewares/{id}/pages/{num} 提取ID和页码
func extractCoursewarePagePath(path string) (string, int) {
	const prefix = "/api/v1/coursewares/"
	if !strings.HasPrefix(path, prefix) {
		return "", 0
	}
	rest := path[len(prefix):]
	idx := strings.Index(rest, "/pages/")
	if idx <= 0 {
		return "", 0
	}
	cwID := rest[:idx]
	numStr := rest[idx+len("/pages/"):]
	numStr = strings.TrimRight(numStr, "/")
	if slashIdx := strings.Index(numStr, "/"); slashIdx >= 0 {
		numStr = numStr[:slashIdx]
	}
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return cwID, 0
	}
	return cwID, num
}

// extractCWTemplateID 从 /api/v1/courseware-templates/{id}/... 提取ID
func extractCWTemplateID(path string) string {
	const prefix = "/api/v1/courseware-templates/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := path[len(prefix):]
	if idx := strings.Index(rest, "/"); idx > 0 {
		return rest[:idx]
	}
	return strings.TrimRight(rest, "/")
}

// ==================== v0.42新增：从主题创建课件 ====================

// CreateFromTopic POST /api/v1/coursewares/from-topic — 从主题直接创建课件
func (h *CoursewareHandler) CreateFromTopic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	if _, err := services.ResolveCoursewareCreationEducationDomain(actor); err != nil {
		utils.Fail(w, http.StatusForbidden, err.Error())
		return
	}

	var req models.CreateCoursewareFromTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	cw, err := h.cwService.CreateCoursewareFromTopic(r.Context(), actor, &req)
	if err != nil {
		utils.InternalError(w, "创建课件失败: "+err.Error())
		return
	}
	utils.Success(w, cw)
}

// ==================== v0.42.11新增：创建3D互动单页课件 ====================

// CreateFrom3D POST /api/v1/coursewares/from-3d — 创建3D互动单页课件
// 创建后 source_type='3d_single'，状态直接为 generating，自动创建1个页面记录
func (h *CoursewareHandler) CreateFrom3D(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	if _, err := services.ResolveCoursewareCreationEducationDomain(actor); err != nil {
		utils.Fail(w, http.StatusForbidden, err.Error())
		return
	}

	var req struct {
		Subject     string `json:"subject"`
		Grade       string `json:"grade"`
		Topic       string `json:"topic"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	cw, err := h.cwService.CreateCoursewareFrom3D(r.Context(), actor, req.Subject, req.Grade, req.Topic, req.Description)
	if err != nil {
		utils.InternalError(w, "创建3D课件失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"id":               cw.ID,
		"title":            cw.Title,
		"source_type":      cw.SourceType,
		"education_domain": cw.EducationDomain,
		"status":           cw.Status,
		"message":          "3D互动单页课件创建成功，请触发生成",
	})
}

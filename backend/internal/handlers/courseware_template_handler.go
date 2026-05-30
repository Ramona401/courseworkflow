package handlers

// courseware_template_handler.go - 课件风格模板 HTTP 处理器(v139.1-fix)
//
// v137 原有 API:
//   1. GET    /api/v1/courseware-templates/with-user
//   2. POST   /api/v1/coursewares/{id}/save-as-template
//   3. DELETE /api/v1/courseware-templates/personal/{id}
//
// v139 新增 8 个 API:
//   4. POST   /api/v1/coursewares/templates/extract
//   5. GET    /api/v1/coursewares/templates/my-drafts
//   6. DELETE /api/v1/coursewares/templates/drafts/{id}
//   7. POST   /api/v1/coursewares/templates/{id}/refine
//   8. GET    /api/v1/sse/template-refine/{id}
//   9. GET    /api/v1/coursewares/templates/{id}/history
//  10. POST   /api/v1/coursewares/templates/{id}/rollback
//  11. POST   /api/v1/coursewares/templates/{id}/publish
//
// v139.1 新增:
//  12. GET    /api/v1/coursewares/templates/publish-targets
//
// v139.1-fix 修复:
//   - fmt.Printf 全部替换为 logger.WithModule 结构化日志

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tedna/internal/logger"
	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// 模块级日志器
var tplHandlerLog = logger.WithModule("template_handler")

// CoursewareTemplateHandler 课件模板 HTTP 处理器
// v139 增加 3 个服务依赖,但保留零参数构造函数向后兼容
type CoursewareTemplateHandler struct {
	extractService *services.TemplateExtractService
	refineService  *services.TemplateRefineService
	authService    *services.AuthService
}

// NewCoursewareTemplateHandler 零参数构造函数(v137 兼容)
func NewCoursewareTemplateHandler() *CoursewareTemplateHandler {
	return &CoursewareTemplateHandler{}
}

// NewCoursewareTemplateHandlerV139 v139 完整构造函数
func NewCoursewareTemplateHandlerV139(
	extractSvc *services.TemplateExtractService,
	refineSvc *services.TemplateRefineService,
	authSvc *services.AuthService,
) *CoursewareTemplateHandler {
	return &CoursewareTemplateHandler{
		extractService: extractSvc,
		refineService:  refineSvc,
		authService:    authSvc,
	}
}

// ==================== v137 原有 API ====================

// ListTemplatesWithUser GET /api/v1/courseware-templates/with-user
func (h *CoursewareTemplateHandler) ListTemplatesWithUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	templates, err := repository.ListCWTemplatesWithUser(r.Context(), claims.UserID, true)
	if err != nil {
		utils.InternalError(w, "查询模板失败: "+err.Error())
		return
	}
	utils.Success(w, templates)
}

// SaveAsMyTemplate POST /api/v1/coursewares/{id}/save-as-template
func (h *CoursewareTemplateHandler) SaveAsMyTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	cwID := extractCoursewareMiddleID(r.URL.Path, "/save-as-template")
	if cwID == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	var req models.SaveAsMyTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		utils.BadRequest(w, "模板名称不能为空")
		return
	}

	cw, err := repository.GetCoursewareByID(r.Context(), cwID)
	if err != nil {
		utils.InternalError(w, "课件不存在: "+err.Error())
		return
	}
	if cw.UserID != claims.UserID {
		utils.Forbidden(w, "无权操作此课件")
		return
	}

	styleCategory := req.StyleCategory
	if styleCategory == "" {
		styleCategory = extractStyleCategoryFromConfig(cw.StyleConfig)
	}
	if styleCategory == "" {
		styleCategory = models.CWStyleMinimalist
	}

	pages, _ := repository.ListCoursewarePages(r.Context(), cwID)
	samplePagesJSON := "[]"
	if len(pages) > 0 {
		for _, p := range pages {
			if p.PageNumber == 1 && p.HTMLContent != "" {
				sampleJSON, _ := json.Marshal([]string{p.HTMLContent})
				samplePagesJSON = string(sampleJSON)
				break
			}
		}
	}

	userID := claims.UserID
	tpl := &models.CoursewareTemplate{
		Name:               strings.TrimSpace(req.Name),
		Description:        req.Description,
		StyleCategory:      styleCategory,
		ColorScheme:        extractFieldFromConfig(cw.StyleConfig, "color_scheme_raw"),
		CSSVariables:       extractFieldFromConfig(cw.StyleConfig, "css_variables_raw"),
		SamplePages:        samplePagesJSON,
		IsActive:           true,
		SortOrder:          0,
		UserID:             &userID,
		Scope:              models.CWTemplateScopePersonal,
		SourceCoursewareID: &cwID,
	}

	templateID := extractFieldFromConfig(cw.StyleConfig, "template_id")
	if templateID != "" {
		origTpl, origErr := repository.GetCWTemplateByID(r.Context(), templateID)
		if origErr == nil {
			tpl.ColorScheme = origTpl.ColorScheme
			tpl.CSSVariables = origTpl.CSSVariables
			if tpl.StyleCategory == "" || tpl.StyleCategory == models.CWStyleMinimalist {
				tpl.StyleCategory = origTpl.StyleCategory
			}
		}
	}

	if err := repository.CreatePersonalTemplate(r.Context(), tpl); err != nil {
		utils.InternalError(w, "保存模板失败: "+err.Error())
		return
	}

	utils.Success(w, map[string]interface{}{
		"id":      tpl.ID,
		"name":    tpl.Name,
		"message": fmt.Sprintf("模板「%s」保存成功", tpl.Name),
	})
}

// DeleteMyTemplate DELETE /api/v1/courseware-templates/personal/{id}
func (h *CoursewareTemplateHandler) DeleteMyTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	tplID := extractPersonalTemplateID(r.URL.Path)
	if tplID == "" {
		utils.BadRequest(w, utils.MsgMissingTemplateID)
		return
	}

	if err := repository.DeletePersonalTemplate(r.Context(), tplID, claims.UserID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "模板删除成功"})
}

// ==================== v139: AI 提取草稿 ====================

// ExtractFromHTML POST /api/v1/coursewares/templates/extract
// 用户粘贴 HTML, AI 提取风格模板入库为草稿
// v145: 改为异步执行,通过 SSE 推送进度(复用微调 SSE 模式)
func (h *CoursewareTemplateHandler) ExtractFromHTML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	if h.extractService == nil {
		utils.InternalError(w, "AI 提取服务未初始化(v139)")
		return
	}

	var req models.ExtractTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if len(req.SamplePages) == 0 {
		utils.BadRequest(w, "请至少提供一页 HTML 内容")
		return
	}

	// v145: 改为异步执行, 延迟 800ms 等前端 SSE 连接建立
	userID := claims.UserID
	samplePages := req.SamplePages
	sourceType := req.SourceType
	go func() {
		time.Sleep(800 * time.Millisecond)
		asyncCtx := context.Background()
		h.extractService.ExtractFromHTMLAsync(asyncCtx, userID, samplePages, sourceType)
	}()

	utils.Success(w, map[string]interface{}{
		"message": "AI 提取已启动,请通过 SSE 监听进度",
	})
}

// ListMyDrafts GET /api/v1/coursewares/templates/my-drafts
// 查询当前用户的所有 AI 提取草稿
func (h *CoursewareTemplateHandler) ListMyDrafts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	drafts, err := repository.ListMyDrafts(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, "查询草稿失败: "+err.Error())
		return
	}
	utils.Success(w, drafts)
}

// DeleteDraft DELETE /api/v1/coursewares/templates/drafts/{id}
// 删除指定草稿(仅限本人)
func (h *CoursewareTemplateHandler) DeleteDraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	tplID := extractTemplateIDFromDraftsPath(r.URL.Path)
	if tplID == "" {
		utils.BadRequest(w, utils.MsgMissingTemplateID)
		return
	}

	if err := repository.DeleteDraftTemplate(r.Context(), tplID, claims.UserID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "草稿删除成功"})
}

// ==================== v139: AI 微调(异步,SSE 推送) ====================


// ==================== v145 新增: AI 提取 SSE 端点 ====================

// ExtractStream GET /api/v1/sse/template-extract?token=xxx
// SSE 订阅模板 AI 提取进度. Token 通过 URL 参数传递(EventSource 不支持 Header)
// SSE Key: "extract_" + userID (提取时还没有 templateID,用用户维度唯一标识)
func (h *CoursewareTemplateHandler) ExtractStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	if h.authService == nil {
		http.Error(w, `{"code":-1,"message":"auth service not initialized"}`, http.StatusInternalServerError)
		return
	}

	token := extractTokenFromQuery(r)
	if token == "" {
		http.Error(w, `{"code":-1,"message":"缺少 token 参数"}`, http.StatusUnauthorized)
		return
	}
	claims, err := h.authService.ValidateToken(token)
	if err != nil || claims == nil {
		http.Error(w, `{"code":-1,"message":"token 无效或已过期"}`, http.StatusUnauthorized)
		return
	}

	sseKey := "extract_" + claims.UserID

	// SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持流式响应", http.StatusInternalServerError)
		return
	}

	ch := services.GlobalCWSSEHub.Subscribe(sseKey)
	defer services.GlobalCWSSEHub.Unsubscribe(sseKey, ch)

	writeCWSSEEvent(w, flusher, services.CWSSEConnected, map[string]string{
		"message": "提取 SSE 连接已建立",
	})

	// 提取比微调耗时更久(AI 调用 3-8 分钟),设 12 分钟超时
	timeout := time.After(12 * time.Minute)
	for {
		select {
		case event, open := <-ch:
			if !open {
				return
			}
			writeCWSSEEvent(w, flusher, event.EventType, event.Data)
			// 终态事件:完成或失败后关闭连接
			if event.EventType == services.CWSSEExtractDone || event.EventType == services.CWSSEExtractError {
				return
			}
		 case <-r.Context().Done():
			return
		case <-timeout:
			writeCWSSEEvent(w, flusher, "timeout", map[string]string{
				"message": "SSE 连接超时(12 分钟)",
			})
			return
		}
	}
}

// RefineTemplate POST /api/v1/coursewares/templates/{id}/refine
// 触发 AI 微调,异步执行,通过 SSE 推送进度
func (h *CoursewareTemplateHandler) RefineTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	if h.refineService == nil {
		utils.InternalError(w, "AI 微调服务未初始化(v139)")
		return
	}

	tplID := extractTemplateMiddleID(r.URL.Path, "/refine")
	if tplID == "" {
		utils.BadRequest(w, utils.MsgMissingTemplateID)
		return
	}

	var req models.RefineTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if strings.TrimSpace(req.Instruction) == "" {
		utils.BadRequest(w, "修改指令不能为空")
		return
	}

	// 异步执行, 延迟 800ms 等前端 SSE 连接建立
	userID := claims.UserID
	go func() {
		time.Sleep(800 * time.Millisecond)
		asyncCtx := context.Background()
		if err := h.refineService.RefineTemplate(asyncCtx, tplID, userID, req.Instruction); err != nil {
			tplHandlerLog.Error("微调异步执行失败", "template", tplID, "error", err)
		}
	}()

	utils.Success(w, map[string]interface{}{
		"message":     "AI 微调已启动, 请通过 SSE 监听进度",
		"template_id": tplID,
	})
}

// RefineStream GET /api/v1/sse/template-refine/{id}?token=xxx
// SSE 订阅模板微调进度. Token 通过 URL 参数传递(EventSource 不支持 Header)
func (h *CoursewareTemplateHandler) RefineStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	if h.authService == nil {
		http.Error(w, `{"code":-1,"message":"auth service not initialized"}`, http.StatusInternalServerError)
		return
	}

	token := extractTokenFromQuery(r)
	if token == "" {
		http.Error(w, `{"code":-1,"message":"缺少 token 参数"}`, http.StatusUnauthorized)
		return
	}
	_, err := h.authService.ValidateToken(token)
	if err != nil {
		http.Error(w, `{"code":-1,"message":"token 无效或已过期"}`, http.StatusUnauthorized)
		return
	}

	tplID := extractTemplateRefineSSEID(r.URL.Path)
	if tplID == "" {
		utils.BadRequest(w, utils.MsgMissingTemplateID)
		return
	}

	// SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持流式响应", http.StatusInternalServerError)
		return
	}

	ch := services.GlobalCWSSEHub.Subscribe(tplID)
	defer services.GlobalCWSSEHub.Unsubscribe(tplID, ch)

	writeCWSSEEvent(w, flusher, services.CWSSEConnected, map[string]string{
		"template_id": tplID,
		"message":     "微调 SSE 连接已建立",
	})

	timeout := time.After(10 * time.Minute)
	for {
		select {
		case event, open := <-ch:
			if !open {
				return
			}
			writeCWSSEEvent(w, flusher, event.EventType, event.Data)
			if event.EventType == services.CWSSERefineDone || event.EventType == services.CWSSERefineError {
				return
			}
		case <-r.Context().Done():
			return
		case <-timeout:
			writeCWSSEEvent(w, flusher, "timeout", map[string]string{
				"message": "SSE 连接超时",
			})
			return
		}
	}
}

// ==================== v139: 微调历史 ====================

// GetHistory GET /api/v1/coursewares/templates/{id}/history
func (h *CoursewareTemplateHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	if h.refineService == nil {
		utils.InternalError(w, "AI 微调服务未初始化(v139)")
		return
	}

	tplID := extractTemplateMiddleID(r.URL.Path, "/history")
	if tplID == "" {
		utils.BadRequest(w, utils.MsgMissingTemplateID)
		return
	}

	history, err := h.refineService.GetRefineHistory(r.Context(), tplID, claims.UserID)
	if err != nil {
		utils.InternalError(w, "查询历史失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"template_id": tplID,
		"history":     history,
		"total":       len(history),
	})
}

// RollbackToHistory POST /api/v1/coursewares/templates/{id}/rollback
func (h *CoursewareTemplateHandler) RollbackToHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	if h.refineService == nil {
		utils.InternalError(w, "AI 微调服务未初始化(v139)")
		return
	}

	tplID := extractTemplateMiddleID(r.URL.Path, "/rollback")
	if tplID == "" {
		utils.BadRequest(w, utils.MsgMissingTemplateID)
		return
	}

	var req models.RollbackToHistoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if req.HistoryIndex < 0 {
		utils.BadRequest(w, "历史索引不能为负数")
		return
	}

	tpl, err := h.refineService.RollbackToHistory(r.Context(), tplID, claims.UserID, req.HistoryIndex)
	if err != nil {
		utils.InternalError(w, "回退失败: "+err.Error())
		return
	}

	utils.Success(w, map[string]interface{}{
		"template_id":    tpl.ID,
		"color_scheme":   tpl.ColorScheme,
		"css_variables":  tpl.CSSVariables,
		"sample_pages":   tpl.SamplePages,
		"style_category": tpl.StyleCategory,
		"message":        "已恢复到历史版本",
	})
}

// ==================== v139: 草稿发布为正式 ====================

// PublishDraft POST /api/v1/coursewares/templates/{id}/publish
// 把草稿模板发布为指定 scope 的正式模板
// 权限规则(按 scope 分级):
//   - personal: 任何人(发布自己的草稿为个人模板)
//   - school:   senior_operator 必须是该学校管理员; admin 可指定任意学校
//   - group:    该教研组的 lead/backbone (admin 可绕过)
//   - system:   仅 admin
func (h *CoursewareTemplateHandler) PublishDraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	tplID := extractTemplateMiddleID(r.URL.Path, "/publish")
	if tplID == "" {
		utils.BadRequest(w, utils.MsgMissingTemplateID)
		return
	}

	var req models.PublishDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		utils.BadRequest(w, "模板名称不能为空")
		return
	}
	if !models.IsValidCWTemplateScope(req.Scope) {
		utils.BadRequest(w, "无效的 scope: "+req.Scope)
		return
	}

	// 加载草稿确认所有权
	draft, err := repository.GetCWTemplateByID(r.Context(), tplID)
	if err != nil {
		utils.InternalError(w, "草稿不存在: "+err.Error())
		return
	}
	if !draft.IsDraft {
		utils.BadRequest(w, "此模板已发布, 无需重复发布")
		return
	}
	if draft.UserID == nil || *draft.UserID != claims.UserID {
		utils.Forbidden(w, "无权发布此草稿")
		return
	}

	// 按 scope 分级校验权限
	scopeTargetID := strings.TrimSpace(req.ScopeTargetID)
	switch req.Scope {
	case models.CWTemplateScopePersonal:
		scopeTargetID = ""

	case models.CWTemplateScopeSchool:
		if claims.Role != models.RoleSeniorOperator && claims.Role != models.RoleAdmin {
			utils.Forbidden(w, "只有学校管理员或系统管理员可以发布学校模板")
			return
		}
		if claims.Role == models.RoleSeniorOperator {
			school, sErr := repository.GetSchoolByAdminUserID(r.Context(), claims.UserID)
			if sErr != nil || school == nil {
				utils.Forbidden(w, "您未管理任何学校, 无法发布学校模板")
				return
			}
			if scopeTargetID == "" {
				scopeTargetID = school.ID
			} else if scopeTargetID != school.ID {
				utils.Forbidden(w, "只能发布到您管理的学校")
				return
			}
		} else {
			if scopeTargetID == "" {
				utils.BadRequest(w, "管理员发布学校模板时必须指定 scope_target_id(学校 ID)")
				return
			}
		}

	case models.CWTemplateScopeGroup:
		if scopeTargetID == "" {
			utils.BadRequest(w, "发布教研组模板必须指定 scope_target_id(教研组 ID)")
			return
		}
		if claims.Role != models.RoleAdmin {
			isLeadOrBackbone, lbErr := repository.IsGroupLeadOrBackbone(r.Context(), scopeTargetID, claims.UserID)
			if lbErr != nil {
				utils.InternalError(w, "校验教研组权限失败: "+lbErr.Error())
				return
			}
			if !isLeadOrBackbone {
				utils.Forbidden(w, "只有教研组组长或骨干可以发布教研组模板")
				return
			}
		}

	case models.CWTemplateScopeSystem:
		if claims.Role != models.RoleAdmin {
			utils.Forbidden(w, "只有系统管理员可以发布为系统模板")
			return
		}
		scopeTargetID = ""

	default:
		utils.BadRequest(w, "不支持的 scope")
		return
	}

	styleCategory := req.StyleCategory
	if styleCategory == "" {
		styleCategory = draft.StyleCategory
	}
	if !models.IsValidCWStyleCategory(styleCategory) {
		styleCategory = models.CWStyleMinimalist
	}

	err = repository.PublishDraft(
		r.Context(),
		tplID,
		name,
		strings.TrimSpace(req.Description),
		styleCategory,
		req.Scope,
		scopeTargetID,
	)
	if err != nil {
		utils.InternalError(w, "发布失败: "+err.Error())
		return
	}

	utils.Success(w, map[string]interface{}{
		"template_id":  tplID,
		"name":         name,
		"scope":        req.Scope,
		"scope_target": scopeTargetID,
		"message":      fmt.Sprintf("模板「%s」已发布为%s", name, models.CWTemplateScopeNameMap[req.Scope]),
	})
}

// ==================== v142 新增:撤回已发布模板 ====================

// UnpublishTemplate POST /api/v1/coursewares/templates/{id}/unpublish
// 撤回已发布的正式模板,转为个人草稿
//
// 权限规则(按原 scope 分级校验):
//   - personal: 模板所有者本人
//   - school:   学校管理员(senior_operator) 或 admin
//   - group:    教研组 lead/backbone 或 admin
//   - system:   仅 admin
func (h *CoursewareTemplateHandler) UnpublishTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	tplID := extractTemplateMiddleID(r.URL.Path, "/unpublish")
	if tplID == "" {
		utils.BadRequest(w, utils.MsgMissingTemplateID)
		return
	}

	// 加载模板,确认存在且已发布
	tpl, err := repository.GetCWTemplateByID(r.Context(), tplID)
	if err != nil {
		utils.InternalError(w, "模板不存在: "+err.Error())
		return
	}
	if tpl.IsDraft {
		utils.BadRequest(w, "此模板已是草稿状态,无需撤回")
		return
	}

	// 按 scope 分级校验权限
	switch tpl.Scope {
	case models.CWTemplateScopePersonal:
		// 个人模板:只允许本人撤回
		if tpl.UserID == nil || *tpl.UserID != claims.UserID {
			utils.Forbidden(w, "无权撤回此模板")
			return
		}

	case models.CWTemplateScopeSchool:
		// 学校模板:学校管理员或 admin
		if claims.Role != models.RoleSeniorOperator && claims.Role != models.RoleAdmin {
			utils.Forbidden(w, "只有学校管理员或系统管理员可以撤回学校模板")
			return
		}
		if claims.Role == models.RoleSeniorOperator {
			school, sErr := repository.GetSchoolByAdminUserID(r.Context(), claims.UserID)
			if sErr != nil || school == nil {
				utils.Forbidden(w, "您未管理任何学校")
				return
			}
			if tpl.ScopeTargetID == nil || *tpl.ScopeTargetID != school.ID {
				utils.Forbidden(w, "只能撤回您管理的学校的模板")
				return
			}
		}

	case models.CWTemplateScopeGroup:
		// 教研组模板:lead/backbone 或 admin
		if claims.Role != models.RoleAdmin {
			if tpl.ScopeTargetID == nil {
				utils.Forbidden(w, "模板数据异常: 缺少教研组 ID")
				return
			}
			isLeadOrBackbone, lbErr := repository.IsGroupLeadOrBackbone(r.Context(), *tpl.ScopeTargetID, claims.UserID)
			if lbErr != nil {
				utils.InternalError(w, "校验教研组权限失败: "+lbErr.Error())
				return
			}
			if !isLeadOrBackbone {
				utils.Forbidden(w, "只有教研组组长或骨干可以撤回教研组模板")
				return
			}
		}

	case models.CWTemplateScopeSystem:
		// 系统模板:仅 admin
		if claims.Role != models.RoleAdmin {
			utils.Forbidden(w, "只有系统管理员可以撤回系统模板")
			return
		}

	default:
		utils.BadRequest(w, "未知的模板 scope")
		return
	}

	// 执行撤回
	if err := repository.UnpublishTemplate(r.Context(), tplID); err != nil {
		utils.InternalError(w, "撤回失败: "+err.Error())
		return
	}

	tplHandlerLog.Info("模板已撤回", "template", tplID, "original_scope", tpl.Scope, "user", claims.UserID)
	utils.Success(w, map[string]interface{}{
		"template_id": tplID,
		"message":     fmt.Sprintf("模板「%s」已撤回为草稿", tpl.Name),
	})
}

// ==================== style_config JSON 解析辅助函数(v137 保留) ====================

// extractStyleCategoryFromConfig 从 style_config JSON 中提取 style_category
func extractStyleCategoryFromConfig(styleConfig string) string {
	if styleConfig == "" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(styleConfig), &m); err != nil {
		return ""
	}
	if v, ok := m["style_category"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractFieldFromConfig 从 style_config JSON 中提取指定字段
func extractFieldFromConfig(styleConfig string, field string) string {
	if styleConfig == "" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(styleConfig), &m); err != nil {
		return ""
	}
	if v, ok := m[field]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ==================== URL 路径解析辅助函数 ====================

// extractPersonalTemplateID 从 /api/v1/courseware-templates/personal/{id} 提取 ID(v137 旧路径)
func extractPersonalTemplateID(path string) string {
	const prefix = "/api/v1/courseware-templates/personal/"
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

// extractTemplateIDFromDraftsPath 从 /api/v1/coursewares/templates/drafts/{id} 提取 ID
func extractTemplateIDFromDraftsPath(path string) string {
	const prefix = "/api/v1/coursewares/templates/drafts/"
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

// extractTemplateMiddleID 从 /api/v1/coursewares/templates/{id}/<suffix> 提取 ID
func extractTemplateMiddleID(path string, suffix string) string {
	const prefix = "/api/v1/coursewares/templates/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := path[len(prefix):]
	if !strings.HasSuffix(rest, suffix) {
		return ""
	}
	rest = rest[:len(rest)-len(suffix)]
	rest = strings.TrimRight(rest, "/")
	if rest == "" || strings.Contains(rest, "/") {
		return ""
	}
	return rest
}

// extractTemplateRefineSSEID 从 /api/v1/sse/template-refine/{id} 提取模板 ID
func extractTemplateRefineSSEID(path string) string {
	const prefix = "/api/v1/sse/template-refine/"
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

// ==================== v139.1 新增:发布目标查询 ====================

// GetPublishTargets GET /api/v1/coursewares/templates/publish-targets
//
// 聚合查询当前登录用户可发布到的所有 scope 目标,前端据此渲染发布表单的下拉选项
func (h *CoursewareTemplateHandler) GetPublishTargets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	resp := models.PublishTargetsResponse{}

	// personal: 任何登录用户都可以发布到个人模板库
	resp.Personal.Available = true

	// system: 仅 admin 角色
	if claims.Role == models.RoleAdmin {
		resp.System.Available = true
	} else {
		resp.System.Available = false
		resp.System.Reason = "只有系统管理员可以发布为系统模板"
	}

	// school: 查 organizations 表 admin_user_id == 当前用户 且 type='school'
	school, err := repository.GetSchoolByAdminUserID(r.Context(), claims.UserID)
	if err == nil && school != nil {
		resp.School.Available = true
		resp.School.SchoolID = school.ID
		resp.School.Name = school.Name
	} else {
		resp.School.Available = false
		if err != nil {
			tplHandlerLog.Warn("查询用户管理的学校失败", "user", claims.UserID, "error", err)
		}
	}

	// groups: 查 teaching_group_members 中 user_id=当前用户 且 role IN (lead, backbone)
	targetGroups, gErr := repository.ListMyLeadOrBackboneGroups(r.Context(), claims.UserID)
	if gErr != nil {
		// 查询失败不阻塞整个接口,只把数组置空并记录日志
		tplHandlerLog.Warn("查询用户管理的教研组失败", "user", claims.UserID, "error", gErr)
		resp.Groups = []models.PublishTargetGroup{}
	} else {
		resp.Groups = targetGroups
	}

	utils.Success(w, resp)
}

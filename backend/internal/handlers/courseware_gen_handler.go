package handlers

// courseware_gen_handler.go — 课件HTML生成HTTP处理器（Phase 4C P0-1~P0-5 + 单页重生 + 全自动装配）
//
// 提供接口：
//   1. POST /api/v1/coursewares/{id}/generate-preview        — 仅生成预览页（封面）
//   2. POST /api/v1/coursewares/{id}/save-nav-template        — 保存导航栏HTML模板
//   3. POST /api/v1/coursewares/{id}/generate-pages           — 用固定导航栏批量生成剩余页
//   4. POST /api/v1/coursewares/{id}/refine-nav                — 导航栏AI微调
//   5. POST /api/v1/coursewares/{id}/pages/{num}/refine        — 单页AI微调（支持可选截图多模态）
//   6. POST /api/v1/coursewares/{id}/pages/{num}/regenerate    — 单页重新生成（从方案从零重做）
//   7. POST /api/v1/coursewares/{id}/cancel-generate           — 中途中断生成
//   8. POST /api/v1/coursewares/{id}/generate-3d-page          — 3D互动单页生成
//   9. GET  /api/v1/coursewares/{id}/pages/{num}/versions            — 页面版本列表（轻量，不含html）
//  10. GET  /api/v1/coursewares/{id}/pages/{num}/versions/{versionId} — 取单个历史版本完整HTML（版本对比UI用）
//  11. POST /api/v1/coursewares/{id}/pages/{num}/rollback           — 回退到指定历史版本
//  12. POST /api/v1/coursewares/{id}/auto-assemble            — 全自动一键装配（HTML+配图+视频占位总装线）
//  13. POST /api/v1/coursewares/{id}/pages/{num}/save-html    — 【就地文字编辑】保存前端改过的整页HTML
//  14. POST /api/v1/coursewares/{id}/pages/{num}/import-html  — 【粘贴HTML建页·批次B】导入外部HTML到指定页

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== 课件生成处理器 ====================

// CoursewareGenHandler 课件HTML生成处理器
type CoursewareGenHandler struct {
	genService *services.CoursewareGenService
	cwService  *services.CoursewareService
	// autoAssemblyService 全自动一键装配主编排服务（HTML生成+配图+视频占位总装线）
	autoAssemblyService *services.CoursewareAutoAssemblyService
}

// NewCoursewareGenHandler 创建课件HTML生成处理器
// autoAssemblyService：全自动装配服务，供 AutoAssemble 端点异步调用。
func NewCoursewareGenHandler(
	genService *services.CoursewareGenService,
	cwService *services.CoursewareService,
	autoAssemblyService *services.CoursewareAutoAssemblyService,
) *CoursewareGenHandler {
	return &CoursewareGenHandler{
		genService:          genService,
		cwService:           cwService,
		autoAssemblyService: autoAssemblyService,
	}
}

// authorizeCoursewareOwnerRuntime 构造可信Actor并执行作者专属课件运行预检。
//
// 异步生成端点必须在启动goroutine之前调用本函数；授权通过后返回的Actor
// 已收敛到课件历史教育域快照。后台Service仍会再次校验，形成双层保护。
func (h *CoursewareGenHandler) authorizeCoursewareOwnerRuntime(
	ctx context.Context,
	coursewareID string,
	userID string,
	role string,
) (*services.CoursewareActorContext, error) {
	actor := services.BuildCoursewareActorFromClaims(
		ctx,
		userID,
		role,
	)

	_, scopedActor, err :=
		h.cwService.LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return scopedActor, nil
}

// writeCoursewareOwnerRuntimeError 统一映射生成链作者域授权错误。
func writeCoursewareOwnerRuntimeError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrCoursewareAccessNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			"课件不存在",
		)

	case errors.Is(
		err,
		services.ErrCoursewareActorRequired,
	),
		errors.Is(
			err,
			services.ErrCoursewareOwnerRuntimeDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainMismatch,
		):
		utils.Fail(
			w,
			http.StatusForbidden,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewarePageNotFound,
	),
		errors.Is(
			err,
			services.ErrCoursewarePageVersionNotFound,
		):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewarePageMutationConflict,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewarePageHTMLInvalid,
	):
		utils.BadRequest(
			w,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewarePageVersionSnapshotFailed,
	):
		utils.InternalError(
			w,
			"保存页面历史版本失败，请稍后重试",
		)

	case errors.Is(
		err,
		services.ErrCoursewareEducationDomainInvalid,
	),
		errors.Is(
			err,
			services.ErrCoursewareRuntimeDomainRequired,
		):
		utils.InternalError(
			w,
			err.Error(),
		)

	default:
		utils.InternalError(
			w,
			err.Error(),
		)
	}
}

// ==================== Step 1: 生成预览页 ====================

// GeneratePreview POST /api/v1/coursewares/{id}/generate-preview
// GeneratePreview 保留旧内部方法名，统一转发受Tracker治理的正式入口。
func (h *CoursewareGenHandler) GeneratePreview(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.GeneratePreviewTracked(w, r)
}

// ==================== Step 2: 保存导航栏模板 ====================

// SaveNavTemplate POST /api/v1/coursewares/{id}/save-nav-template
func (h *CoursewareGenHandler) SaveNavTemplate(
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
		"/save-nav-template",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := h.authorizeCoursewareOwnerRuntime(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	var req models.SaveNavTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.SaveNavTemplateForActor(
		r.Context(),
		id,
		actor,
		req.NavTemplateHTML,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "导航栏模板保存成功",
		},
	)
}

// ==================== Step 3: 批量生成剩余页 ====================

// GeneratePages POST /api/v1/coursewares/{id}/generate-pages
// GeneratePages 保留旧内部方法名，统一转发受Tracker治理的正式入口。
func (h *CoursewareGenHandler) GeneratePages(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.GeneratePagesTracked(w, r)
}

// ==================== 全自动一键装配 ====================

// AutoAssemble POST /api/v1/coursewares/{id}/auto-assemble
// 全自动装配总装线：逐页HTML生成 + AI配图（生图→上云→融图）+ 视频首帧占位，一次交付图文齐备课件。
//
// 请求体（可选）: { "skip_video": true }
//   - skip_video 缺省/false = 全自动装配（HTML + 配图 + 视频首帧占位，视频按关键词命中页决定）
//   - skip_video = true      = 交付模式"HTML+配图不做视频"（中间档），所有页一律跳过视频占位
//     body 为空（老调用/不传）时 json.Decode 会得到零值 SkipVideo=false，等同全自动，向后兼容。
//
// 前置强约束：必须已设风格锚点（由 service 层 prepareAssembly 校验，未设则经SSE推 error 并中止）。
// 异步执行（照 GeneratePages 范式），进度经 SSE "assembly_*" 事件推送。
// AutoAssemble 保留旧内部方法名，统一转发受Tracker治理的正式入口。
func (h *CoursewareGenHandler) AutoAssemble(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.AutoAssembleTracked(w, r)
}

// ==================== 导航栏AI微调 ====================

// RefineNav POST /api/v1/coursewares/{id}/refine-nav
// 请求体: { "instruction": "Logo再大一点" }
// 同步返回微调后的导航栏HTML
func (h *CoursewareGenHandler) RefineNav(
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

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID :=
		extractCoursewareMiddleID(
			r.URL.Path,
			"/refine-nav",
		)
	if coursewareID == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	// 必须在解析修改指令正文前完成教研微调授权。
	scopedActor, err :=
		h.authorizeCoursewareRefineForHandler(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareRefineError(
			w,
			err,
		)
		return
	}

	var req struct {
		Instruction string `json:"instruction"`
	}
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}

	if strings.TrimSpace(
		req.Instruction,
	) == "" {
		utils.BadRequest(
			w,
			"修改意见不能为空",
		)
		return
	}

	result, err :=
		h.genService.RefineNav(
			r.Context(),
			coursewareID,
			scopedActor,
			req.Instruction,
		)
	if err != nil {
		writeCoursewareRefineError(
			w,
			err,
		)
		return
	}

	utils.Success(w, map[string]interface{}{
		"nav_html": result,
		"message":  "导航栏微调完成",
	})
}

// ==================== 单页AI微调（支持可选截图多模态） ====================

// RefinePage POST /api/v1/coursewares/{id}/pages/{num}/refine
// 请求体: { "instruction": "标题字号再大一些", "image": "data:image/png;base64,xxx"(可选) }
// 截图非空时让AI看到该页实际渲染来定位版面问题；instruction与image至少一项非空。
func (h *CoursewareGenHandler) RefinePage(
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

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNum :=
		extractCWPageRefinePath(
			r.URL.Path,
		)
	if coursewareID == "" ||
		pageNum <= 0 {
		utils.BadRequest(
			w,
			"路径格式错误",
		)
		return
	}

	// 截图正文可能很大，必须在Decode前完成课件微调授权。
	scopedActor, err :=
		h.authorizeCoursewareRefineForHandler(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareRefineError(
			w,
			err,
		)
		return
	}

	var req struct {
		Instruction string `json:"instruction"`
		Image       string `json:"image"`
		Mode        string `json:"mode"`
	}

	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}

	instruction :=
		strings.TrimSpace(
			req.Instruction,
		)
	image :=
		strings.TrimSpace(
			req.Image,
		)
	mode :=
		strings.ToLower(
			strings.TrimSpace(
				req.Mode,
			),
		)

	if mode == "" {
		mode = "preserve"
	}
	if mode != "preserve" &&
		mode != "rebuild" {
		utils.BadRequest(
			w,
			"修改模式无效，仅支持preserve或rebuild",
		)
		return
	}

	if instruction == "" &&
		image == "" {
		utils.BadRequest(
			w,
			"请提供修改意见或粘贴截图",
		)
		return
	}

	if image != "" {
		if !strings.HasPrefix(
			image,
			"data:image/",
		) {
			utils.BadRequest(
				w,
				"截图格式无效，请直接粘贴图片",
			)
			return
		}

		const maxImageLen = 12 * 1024 * 1024

		if len(image) > maxImageLen {
			utils.BadRequest(
				w,
				"截图过大，请压缩后重试（建议不超过8MB）",
			)
			return
		}
	}

	if instruction == "" {
		if mode == "rebuild" {
			instruction =
				"请参考截图重新设计本页内容区，保留导航栏和模板风格。"
		} else {
			instruction =
				"请参考截图修复页面版面问题，其余结构、内容和交互保持不变。"
		}
	}

	result, err :=
		h.genService.RefinePageWithMode(
			r.Context(),
			coursewareID,
			scopedActor,
			pageNum,
			instruction,
			image,
			mode,
		)
	if err != nil {
		writeCoursewareRefineError(
			w,
			err,
		)
		return
	}

	message :=
		fmt.Sprintf(
			"第%d页微调完成",
			pageNum,
		)
	if mode == "rebuild" {
		message =
			fmt.Sprintf(
				"第%d页全页重构完成",
				pageNum,
			)
	}

	utils.Success(w, map[string]interface{}{
		"page_number":  pageNum,
		"html_content": result,
		"mode":         mode,
		"message":      message,
	})
}

// ==================== 单页重新生成 ====================

// RegeneratePage POST /api/v1/coursewares/{id}/pages/{num}/regenerate
// 整页重做：依据页面方案从零重画内容区后拼接导航栏（不基于现有HTML），同步返回完整页面HTML。
func (h *CoursewareGenHandler) RegeneratePage(
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

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNum :=
		extractCWPageRegeneratePath(
			r.URL.Path,
		)
	if coursewareID == "" ||
		pageNum <= 0 {
		utils.BadRequest(
			w,
			"路径格式错误",
		)
		return
	}

	scopedActor, err :=
		h.authorizeCoursewareRefineForHandler(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareRefineError(
			w,
			err,
		)
		return
	}

	result, err :=
		h.genService.RegenerateSinglePage(
			r.Context(),
			coursewareID,
			scopedActor,
			pageNum,
		)
	if err != nil {
		writeCoursewareRefineError(
			w,
			err,
		)
		return
	}

	utils.Success(w, map[string]interface{}{
		"page_number":  pageNum,
		"html_content": result,
		"message": fmt.Sprintf(
			"第%d页重新生成完成",
			pageNum,
		),
	})
}

// ==================== 就地文字编辑保存 ====================

// SavePageHTML POST /api/v1/coursewares/{id}/pages/{num}/save-html
// 【就地文字编辑】保存老师在预览 iframe 里就地改过的整页 HTML。
//
// 请求体: { "html_content": "<div class=\"cw-page\">...</div>" }
//
// 本端点属于作者专属整页源码覆盖能力。Handler先完成可信Actor作者域
// 预检，Service仍会重新加载正式课件执行二次校验。
func (h *CoursewareGenHandler) SavePageHTML(
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

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNum :=
		extractCWPageSaveHTMLPath(
			r.URL.Path,
		)
	if coursewareID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	scopedActor, err :=
		h.authorizeCoursewareOwnerRuntime(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	var req struct {
		HTMLContent string `json:"html_content"`
	}
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}

	if strings.TrimSpace(
		req.HTMLContent,
	) == "" {
		utils.BadRequest(
			w,
			"编辑后的内容为空，未保存",
		)
		return
	}

	if len(req.HTMLContent) > services.CoursewarePageHTMLMaxBytes {
		utils.BadRequest(
			w,
			"页面内容过大，无法保存",
		)
		return
	}

	result, err :=
		h.genService.SaveManualEditedPage(
			r.Context(),
			coursewareID,
			scopedActor,
			pageNum,
			req.HTMLContent,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	utils.Success(w, map[string]interface{}{
		"page_number":  pageNum,
		"html_content": result,
		"message": fmt.Sprintf(
			"第%d页修改已保存",
			pageNum,
		),
	})
}

// ==================== 粘贴HTML导入（批次B） ====================

// ImportPageHTML POST /api/v1/coursewares/{id}/pages/{num}/import-html
// 【粘贴HTML建页】把老师粘贴的外部完整HTML导入指定页。
//
// 本端点保持作者专属，不向admin或集体备课参与者扩权。
func (h *CoursewareGenHandler) ImportPageHTML(
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

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNum :=
		extractCWPageImportHTMLPath(
			r.URL.Path,
		)
	if coursewareID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	scopedActor, err :=
		h.authorizeCoursewareOwnerRuntime(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	var req struct {
		HTMLContent string `json:"html_content"`
	}
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}

	if strings.TrimSpace(
		req.HTMLContent,
	) == "" {
		utils.BadRequest(
			w,
			"粘贴的内容为空，未导入",
		)
		return
	}

	if len(req.HTMLContent) > services.CoursewarePageHTMLMaxBytes {
		utils.BadRequest(
			w,
			"粘贴的内容过大，无法导入",
		)
		return
	}

	result, err := h.genService.ImportPageHTML(
		r.Context(),
		coursewareID,
		scopedActor,
		pageNum,
		req.HTMLContent,
	)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	utils.Success(w, map[string]interface{}{
		"page_number":  pageNum,
		"html_content": result,
		"message": fmt.Sprintf(
			"第%d页HTML导入完成",
			pageNum,
		),
	})
}

// ==================== v0.42.11: 3D互动单页生成 ====================

// Generate3DPage POST /api/v1/coursewares/{id}/generate-3d-page
// 一次性生成完整的3D互动HTML单页（Three.js + OrbitControls）
// 异步执行，通过SSE推送进度
// Generate3DPage 保留旧内部方法名，统一转发受Tracker治理的正式入口。
func (h *CoursewareGenHandler) Generate3DPage(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.Generate3DPageTracked(w, r)
}

// ==================== P0-5: 中途中断生成 ====================

// CancelGenerate POST /api/v1/coursewares/{id}/cancel-generate
func (h *CoursewareGenHandler) CancelGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/cancel-generate")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	scopedActor, err := h.authorizeCoursewareOwnerRuntime(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareOwnerRuntimeError(w, err)
		return
	}

	if err := h.genService.CancelGenerate(
		r.Context(),
		id,
		scopedActor,
	); err != nil {
		writeCoursewareOwnerRuntimeError(w, err)
		return
	}

	utils.Success(w, map[string]string{
		"message":       "已发送停止信号",
		"courseware_id": id,
	})
}

// ==================== 路径解析辅助函数 ====================

// extractCWPageRefinePath 从 /api/v1/coursewares/{id}/pages/{num}/refine 提取课件ID和页码
func extractCWPageRefinePath(path string) (string, int) {
	return extractCWPageActionPath(path, "/refine")
}

// extractCWPageRegeneratePath 从 /api/v1/coursewares/{id}/pages/{num}/regenerate 提取课件ID和页码
func extractCWPageRegeneratePath(path string) (string, int) {
	return extractCWPageActionPath(path, "/regenerate")
}

// extractCWPageSaveHTMLPath 从 /api/v1/coursewares/{id}/pages/{num}/save-html 提取课件ID和页码
func extractCWPageSaveHTMLPath(path string) (string, int) {
	return extractCWPageActionPath(path, "/save-html")
}

// extractCWPageImportHTMLPath 从 /api/v1/coursewares/{id}/pages/{num}/import-html 提取课件ID和页码（批次B）
func extractCWPageImportHTMLPath(path string) (string, int) {
	return extractCWPageActionPath(path, "/import-html")
}

// extractCWPageActionPath 从 /api/v1/coursewares/{id}/pages/{num}/{action} 提取课件ID和页码
// action 形如 "/refine" / "/regenerate"
func extractCWPageActionPath(path string, action string) (string, int) {
	if !strings.HasSuffix(path, action) {
		return "", 0
	}
	trimmed := strings.TrimSuffix(path, action)
	pagesIdx := strings.LastIndex(trimmed, "/pages/")
	if pagesIdx < 0 {
		return "", 0
	}
	numStr := trimmed[pagesIdx+len("/pages/"):]
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		return "", 0
	}
	prefix := trimmed[:pagesIdx]
	cwPrefix := "/api/v1/coursewares/"
	if !strings.HasPrefix(prefix, cwPrefix) {
		return "", 0
	}
	coursewareID := prefix[len(cwPrefix):]
	if coursewareID == "" {
		return "", 0
	}
	return coursewareID, num
}

// ==================== 页面级版本与回退 ====================

// ListPageVersions GET /api/v1/coursewares/{id}/pages/{num}/versions
// 返回该页的版本列表（按 version_no 倒序，最新在前；轻量，不含 html_content）。
func (h *CoursewareGenHandler) ListPageVersions(
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

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNum :=
		extractCWPageVersionsPath(
			r.URL.Path,
		)
	if coursewareID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	scopedActor, err :=
		h.authorizeCoursewareOwnerRuntime(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	items, err := h.genService.ListCWPageVersions(
		r.Context(),
		coursewareID,
		scopedActor,
		pageNum,
	)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	list := make(
		[]map[string]interface{},
		0,
		len(items),
	)
	for _, item := range items {
		label :=
			models.CWPageVersionSourceNameMap[item.Source]
		if label == "" {
			label = item.Source
		}

		list = append(
			list,
			map[string]interface{}{
				"id":           item.ID,
				"version_no":   item.VersionNo,
				"source":       item.Source,
				"source_label": label,
				"note":         item.Note,
				"created_at":   item.CreatedAt,
			},
		)
	}

	utils.Success(w, map[string]interface{}{
		"page_number": pageNum,
		"versions":    list,
		"total":       len(list),
	})
}

// GetPageVersionDetail GET /api/v1/coursewares/{id}/pages/{num}/versions/{versionId}
// 取某个历史版本的完整HTML，只允许课件作者本人读取。
func (h *CoursewareGenHandler) GetPageVersionDetail(
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

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNum, versionID :=
		extractCWPageVersionDetailPath(
			r.URL.Path,
		)
	if coursewareID == "" ||
		pageNum <= 0 ||
		versionID == "" {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	scopedActor, err :=
		h.authorizeCoursewareOwnerRuntime(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	html, versionNo, source, err :=
		h.genService.GetCWPageVersionHTML(
			r.Context(),
			coursewareID,
			scopedActor,
			pageNum,
			versionID,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	label :=
		models.CWPageVersionSourceNameMap[source]
	if label == "" {
		label = source
	}

	utils.Success(w, map[string]interface{}{
		"page_number":  pageNum,
		"version_id":   versionID,
		"version_no":   versionNo,
		"source":       source,
		"source_label": label,
		"html_content": html,
	})
}

// RollbackPage POST /api/v1/coursewares/{id}/pages/{num}/rollback
// 请求体: { "version_id": "xxx" }
func (h *CoursewareGenHandler) RollbackPage(
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

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNum :=
		extractCWPageRollbackPath(
			r.URL.Path,
		)
	if coursewareID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	scopedActor, err :=
		h.authorizeCoursewareOwnerRuntime(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	var req struct {
		VersionID string `json:"version_id"`
	}
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}

	versionID := strings.TrimSpace(
		req.VersionID,
	)
	if versionID == "" {
		utils.BadRequest(
			w,
			"缺少目标版本ID",
		)
		return
	}

	result, err := h.genService.RollbackCWPage(
		r.Context(),
		coursewareID,
		scopedActor,
		pageNum,
		versionID,
	)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	utils.Success(w, map[string]interface{}{
		"page_number":  pageNum,
		"html_content": result,
		"message": fmt.Sprintf(
			"第%d页已回退",
			pageNum,
		),
	})
}

// extractCWPageVersionsPath 从 /api/v1/coursewares/{id}/pages/{num}/versions 提取课件ID和页码
func extractCWPageVersionsPath(path string) (string, int) {
	return extractCWPageActionPath(path, "/versions")
}

// extractCWPageRollbackPath 从 /api/v1/coursewares/{id}/pages/{num}/rollback 提取课件ID和页码
func extractCWPageRollbackPath(path string) (string, int) {
	return extractCWPageActionPath(path, "/rollback")
}

// extractCWPageVersionDetailPath 从 /api/v1/coursewares/{id}/pages/{num}/versions/{versionId}
// 提取课件ID、页码、版本ID三段。
//
// 解析思路：先定位 "/versions/" 分隔——其后是 versionId，其前是 "/api/v1/coursewares/{id}/pages/{num}"。
// 再对前半段复用 "/pages/" 拆出 coursewareID 与 pageNum。
func extractCWPageVersionDetailPath(path string) (coursewareID string, pageNum int, versionID string) {
	marker := "/versions/"
	vIdx := strings.LastIndex(path, marker)
	if vIdx < 0 {
		return "", 0, ""
	}
	// versionId = "/versions/" 之后的部分（去掉可能的尾斜杠）
	versionID = strings.TrimSuffix(path[vIdx+len(marker):], "/")
	if versionID == "" {
		return "", 0, ""
	}
	// 前半段形如 /api/v1/coursewares/{id}/pages/{num}
	front := path[:vIdx]
	pagesIdx := strings.LastIndex(front, "/pages/")
	if pagesIdx < 0 {
		return "", 0, ""
	}
	numStr := front[pagesIdx+len("/pages/"):]
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		return "", 0, ""
	}
	prefix := front[:pagesIdx]
	cwPrefix := "/api/v1/coursewares/"
	if !strings.HasPrefix(prefix, cwPrefix) {
		return "", 0, ""
	}
	cwID := prefix[len(cwPrefix):]
	if cwID == "" {
		return "", 0, ""
	}
	return cwID, num, versionID
}

package handlers

// courseware_gen_handler.go
//
// 课件HTML生成、导航栏确认与异步生成主处理器。
//
// 本文件保留：
//   1. 生成预览页；
//   2. 保存导航栏模板；
//   3. 批量生成剩余页；
//   4. 全自动一键装配；
//   5. 导航栏AI微调；
//   6. 3D互动单页生成；
//   7. 中途中断生成；
//   8. 课件作者运行权限和通用错误映射。
//
// 以下页级业务已按职责迁移到courseware_page_mutation_handler.go：
//   - 单页AI微调；
//   - 单页重新生成；
//   - 就地文字编辑保存；
//   - 粘贴HTML导入；
//   - 页面历史版本查看与回退；
//   - 对应页级路径解析。
//
// 拆分只改变代码组织，不改变既有HTTP路径、权限或业务语义。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CoursewareGenHandler 课件HTML生成处理器。
type CoursewareGenHandler struct {
	genService *services.CoursewareGenService
	cwService  *services.CoursewareService

	// autoAssemblyService 全自动一键装配主编排服务。
	autoAssemblyService *services.CoursewareAutoAssemblyService
}

// NewCoursewareGenHandler 创建课件HTML生成处理器。
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

// GeneratePreview POST /api/v1/coursewares/{id}/generate-preview。
func (h *CoursewareGenHandler) GeneratePreview(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.GeneratePreviewTracked(w, r)
}

// SaveNavTemplate POST /api/v1/coursewares/{id}/save-nav-template。
func (h *CoursewareGenHandler) SaveNavTemplate(
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

// GeneratePages POST /api/v1/coursewares/{id}/generate-pages。
func (h *CoursewareGenHandler) GeneratePages(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.GeneratePagesTracked(w, r)
}

// AutoAssemble POST /api/v1/coursewares/{id}/auto-assemble。
//
// 请求体可选：
//
//	{"skip_video": true}
//
// 异步执行，进度通过SSE assembly_*事件推送。
func (h *CoursewareGenHandler) AutoAssemble(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.AutoAssembleTracked(w, r)
}

// RefineNav POST /api/v1/coursewares/{id}/refine-nav。
//
// 请求体：
//
//	{"instruction":"Logo再大一点"}
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

	utils.Success(
		w,
		map[string]interface{}{
			"nav_html": result,
			"message":  "导航栏微调完成",
		},
	)
}

// Generate3DPage POST /api/v1/coursewares/{id}/generate-3d-page。
func (h *CoursewareGenHandler) Generate3DPage(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.Generate3DPageTracked(w, r)
}

// CancelGenerate POST /api/v1/coursewares/{id}/cancel-generate。
func (h *CoursewareGenHandler) CancelGenerate(
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

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/cancel-generate",
	)
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

	if err := h.genService.CancelGenerateVersioned(
		r.Context(),
		id,
		scopedActor,
	); err != nil {
		switch {
		case errors.Is(
			err,
			services.ErrCoursewareBatchRunKindMismatch,
		):
			utils.Fail(
				w,
				http.StatusConflict,
				"当前运行是全自动装配，请使用“停止装配”操作",
			)

		default:
			writeCoursewareOwnerRuntimeError(
				w,
				err,
			)
		}
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message":       "已发送停止信号；已完成页面保留，未完成页可稍后只补生成",
			"courseware_id": id,
		},
	)
}

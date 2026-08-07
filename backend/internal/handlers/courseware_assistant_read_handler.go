package handlers

// courseware_assistant_read_handler.go
//
// 本文件实现教师端课件教学智能体的三个读取接口：
//
//   GET /api/v1/coursewares/{id}/assistant-slots
//   GET /api/v1/coursewares/{id}/pages/{page_id}/assistant-slot
//   GET /api/v1/coursewares/{id}/pages/{page_id}/assistant-context
//
// 安全边界：
//   - 所有入口都要求登录JWT；
//   - Actor只根据服务端JWT Claims和正式组织关系构建；
//   - 插槽响应不包含助手完整提示词；
//   - 上下文响应不包含页面完整HTML或教案全文；
//   - 上下文装配Service仍执行作者权限和教育域二次校验；
//   - 本文件不调用AI、不写数据库。

import (
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ListSlots GET /api/v1/coursewares/{id}/assistant-slots。
func (h *CoursewareAssistantHandler) ListSlots(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	coursewareID :=
		extractCoursewareAssistantSlotCollectionPath(
			r.URL.Path,
		)
	if coursewareID == "" {
		utils.BadRequest(
			w,
			"课件教学智能体路径参数无效",
		)
		return
	}

	if h == nil ||
		h.slotService == nil {
		writeCoursewareAssistantHandlerError(
			w,
			services.ErrCoursewareAssistantPlanServiceUnavailable,
		)
		return
	}

	actor :=
		services.BuildCoursewareActorFromClaims(
			r.Context(),
			claims.UserID,
			claims.Role,
		)

	response, err :=
		h.slotService.
			ListCoursewareAssistantSlots(
				r.Context(),
				coursewareID,
				actor,
			)
	if err != nil {
		writeCoursewareAssistantHandlerError(
			w,
			err,
		)
		return
	}

	writeCoursewareAssistantSlotListResponse(
		w,
		response,
	)
}

// GetPageSlot GET /api/v1/coursewares/{id}/pages/{page_id}/assistant-slot。
func (h *CoursewareAssistantHandler) GetPageSlot(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	coursewareID,
		pageID :=
		extractCoursewareAssistantPageActionPath(
			r.URL.Path,
			"/assistant-slot",
		)
	if coursewareID == "" ||
		pageID == "" {
		utils.BadRequest(
			w,
			"课件或稳定页面ID无效",
		)
		return
	}

	if h == nil ||
		h.slotService == nil {
		writeCoursewareAssistantHandlerError(
			w,
			services.ErrCoursewareAssistantPlanServiceUnavailable,
		)
		return
	}

	actor :=
		services.BuildCoursewareActorFromClaims(
			r.Context(),
			claims.UserID,
			claims.Role,
		)

	response, err :=
		h.slotService.
			GetCoursewareAssistantSlotByPage(
				r.Context(),
				coursewareID,
				pageID,
				actor,
			)
	if err != nil {
		writeCoursewareAssistantHandlerError(
			w,
			err,
		)
		return
	}

	writeCoursewareAssistantSlotResponse(
		w,
		response,
	)
}

// GetContextPreview GET /api/v1/coursewares/{id}/pages/{page_id}/assistant-context。
//
// 当前端点使用MVP最大安全默认范围。
// 后续教师保存context_config时仍由Service重新执行协议校验。
func (h *CoursewareAssistantHandler) GetContextPreview(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	coursewareID,
		pageID :=
		extractCoursewareAssistantPageActionPath(
			r.URL.Path,
			"/assistant-context",
		)
	if coursewareID == "" ||
		pageID == "" {
		utils.BadRequest(
			w,
			"课件或稳定页面ID无效",
		)
		return
	}

	if h == nil ||
		h.contextService == nil {
		writeCoursewareAssistantHandlerError(
			w,
			services.ErrCoursewareAssistantPlanServiceUnavailable,
		)
		return
	}

	actor :=
		services.BuildCoursewareActorFromClaims(
			r.Context(),
			claims.UserID,
			claims.Role,
		)

	response, err :=
		h.contextService.
			BuildCoursewareAssistantContextPreview(
				r.Context(),
				coursewareID,
				pageID,
				actor,
				models.DefaultCoursewareAssistantContextConfig(),
			)
	if err != nil {
		writeCoursewareAssistantHandlerError(
			w,
			err,
		)
		return
	}

	writeCoursewareAssistantContextResponse(
		w,
		response,
	)
}

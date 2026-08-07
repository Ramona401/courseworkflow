package handlers

// courseware_assistant_write_handler.go
//
// 本文件实现教师端课件教学智能体的四个写入或消费型接口：
//
//   POST   /api/v1/coursewares/{id}/pages/{page_id}/assistant-slot
//   PUT    /api/v1/coursewares/{id}/assistant-slots/{slot_id}
//   DELETE /api/v1/coursewares/{id}/assistant-slots/{slot_id}
//   POST   /api/v1/coursewares/{id}/pages/{page_id}/assistant-plan
//
// 安全边界：
//   - 所有入口都要求登录JWT；
//   - 在读取JSON正文或产生AI消费之前完成课件作者预检；
//   - Handler预检不能替代Service授权；
//   - Service仍会重新加载正式课件并检查审核写锁；
//   - JSON正文执行字节限长、未知字段拒绝和单对象限制；
//   - 方案生成成功只返回可编辑草稿，不保存插槽、不创建部署；
//   - 不返回助手完整提示词、页面完整HTML、教案全文或模型配置。

import (
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CreatePageSlot POST /api/v1/coursewares/{id}/pages/{page_id}/assistant-slot。
//
// 作者预检发生在JSON正文读取之前。
func (h *CoursewareAssistantHandler) CreatePageSlot(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
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

	actor, err :=
		authorizeCoursewareOwnerRuntimeForHandler(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareAssistantHandlerError(
			w,
			err,
		)
		return
	}

	var request models.CreateCoursewareAssistantSlotRequest

	if !decodeCoursewareAssistantJSON(
		w,
		r,
		&request,
		coursewareAssistantSlotRequestMaxBytes,
	) {
		return
	}

	response, err :=
		h.slotService.
			CreateCoursewareAssistantSlot(
				r.Context(),
				coursewareID,
				pageID,
				actor,
				&request,
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

// UpdateSlot PUT /api/v1/coursewares/{id}/assistant-slots/{slot_id}。
//
// 作者预检发生在JSON正文读取之前。
func (h *CoursewareAssistantHandler) UpdateSlot(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPutOnly,
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
		slotID :=
		extractCoursewareAssistantSlotItemPath(
			r.URL.Path,
		)
	if coursewareID == "" ||
		slotID == "" {
		utils.BadRequest(
			w,
			"课件或插槽ID无效",
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

	actor, err :=
		authorizeCoursewareOwnerRuntimeForHandler(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareAssistantHandlerError(
			w,
			err,
		)
		return
	}

	var request models.UpdateCoursewareAssistantSlotRequest

	if !decodeCoursewareAssistantJSON(
		w,
		r,
		&request,
		coursewareAssistantSlotRequestMaxBytes,
	) {
		return
	}

	response, err :=
		h.slotService.
			UpdateCoursewareAssistantSlot(
				r.Context(),
				coursewareID,
				slotID,
				actor,
				&request,
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

// DeleteSlot DELETE /api/v1/coursewares/{id}/assistant-slots/{slot_id}。
func (h *CoursewareAssistantHandler) DeleteSlot(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodDeleteOnly,
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
		slotID :=
		extractCoursewareAssistantSlotItemPath(
			r.URL.Path,
		)
	if coursewareID == "" ||
		slotID == "" {
		utils.BadRequest(
			w,
			"课件或插槽ID无效",
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

	actor, err :=
		authorizeCoursewareOwnerRuntimeForHandler(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareAssistantHandlerError(
			w,
			err,
		)
		return
	}

	if err :=
		h.slotService.
			DeleteCoursewareAssistantSlot(
				r.Context(),
				coursewareID,
				slotID,
				actor,
			); err != nil {
		writeCoursewareAssistantHandlerError(
			w,
			err,
		)
		return
	}

	writeCoursewareAssistantDeleteResponse(
		w,
	)
}

// GeneratePlan POST /api/v1/coursewares/{id}/pages/{page_id}/assistant-plan。
//
// 作者预检发生在正文读取和AI消费之前。
// 成功响应只是可编辑草稿，不会保存插槽或创建部署。
func (h *CoursewareAssistantHandler) GeneratePlan(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
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
			"/assistant-plan",
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
		h.planService == nil {
		writeCoursewareAssistantHandlerError(
			w,
			services.ErrCoursewareAssistantPlanServiceUnavailable,
		)
		return
	}

	actor, err :=
		authorizeCoursewareOwnerRuntimeForHandler(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareAssistantHandlerError(
			w,
			err,
		)
		return
	}

	var request models.GenerateCoursewareAssistantPlanRequest

	if !decodeCoursewareAssistantJSON(
		w,
		r,
		&request,
		coursewareAssistantPlanRequestMaxBytes,
	) {
		return
	}

	response, err :=
		h.planService.
			GenerateCoursewareAssistantPlan(
				r.Context(),
				coursewareID,
				pageID,
				actor,
				&request,
			)
	if err != nil {
		writeCoursewareAssistantHandlerError(
			w,
			err,
		)
		return
	}

	writeCoursewareAssistantPlanResponse(
		w,
		response,
	)
}

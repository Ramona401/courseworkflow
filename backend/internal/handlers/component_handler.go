package handlers

// component_handler.go — 组件CRUD与匹配HTTP处理器。
//
// 教育域规则：
//   - 每个接口统一建立可信Actor；
//   - 普通用户只读取同域或common；
//   - mixed管理用户可跨域查看；
//   - 普通用户不能修改common或异域组件；
//   - 异域详情和无权写操作统一表现为404；
//   - 普通用户提交的education_domain不能扩大权限；
//   - mixed匹配必须显式选择具体教学域。

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ComponentHandler 组件库接口处理器。
type ComponentHandler struct {
	compService *services.ComponentService
}

// NewComponentHandler 创建组件库处理器实例。
func NewComponentHandler(
	compService *services.ComponentService,
) *ComponentHandler {
	return &ComponentHandler{
		compService: compService,
	}
}

// ListComponents 获取当前Actor可见的组件列表。
func (h *ComponentHandler) ListComponents(
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

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	query := r.URL.Query()

	limit, _ := strconv.Atoi(
		query.Get("limit"),
	)

	offset, _ := strconv.Atoi(
		query.Get("offset"),
	)

	result, err :=
		h.compService.ListComponentsForActor(
			r.Context(),
			actor,
			query.Get("education_domain"),
			query.Get("library_type"),
			query.Get("subject"),
			query.Get("review_status"),
			query.Get("scope"),
			limit,
			offset,
		)
	if err != nil {
		h.handleCompError(w, err)
		return
	}

	utils.Success(w, result)
}

// CreateComponent 使用可信Actor教育域创建组件。
func (h *ComponentHandler) CreateComponent(
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

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	var request models.CreateComponentRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	component, err :=
		h.compService.CreateComponentForActor(
			r.Context(),
			actor,
			&request,
		)
	if err != nil {
		h.handleCompError(w, err)
		return
	}

	utils.Success(w, component)
}

// GetComponent 按可信Actor教育域获取组件详情。
func (h *ComponentHandler) GetComponent(
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

	componentID := extractComponentID(
		r.URL.Path,
	)
	if componentID == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	component, err :=
		h.compService.GetComponentForActor(
			r.Context(),
			actor,
			componentID,
		)
	if err != nil {
		h.handleCompError(w, err)
		return
	}

	utils.Success(w, component)
}

// UpdateComponent 更新Actor有权管理的组件。
func (h *ComponentHandler) UpdateComponent(
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

	componentID := extractComponentID(
		r.URL.Path,
	)
	if componentID == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	var request models.UpdateComponentRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	err = h.compService.UpdateComponentForActor(
		r.Context(),
		actor,
		componentID,
		&request,
	)
	if err != nil {
		h.handleCompError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "更新成功",
		},
	)
}

// DeleteComponent 软删除Actor有权管理的组件。
func (h *ComponentHandler) DeleteComponent(
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

	componentID := extractComponentID(
		r.URL.Path,
	)
	if componentID == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	err = h.compService.DeleteComponentForActor(
		r.Context(),
		actor,
		componentID,
	)
	if err != nil {
		h.handleCompError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "删除成功",
		},
	)
}

// ReviewComponent 审核Actor有权管理的待审组件。
func (h *ComponentHandler) ReviewComponent(
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

	componentID := extractMiddleSegment(
		r.URL.Path,
		"/api/v1/lesson-plans/components/",
		"/review",
	)
	if componentID == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	var request models.ReviewComponentRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	err = h.compService.ReviewComponentForActor(
		r.Context(),
		actor,
		componentID,
		&request,
	)
	if err != nil {
		h.handleCompError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "审核成功",
		},
	)
}

// MatchComponents 按可信Actor或mixed显式教学域匹配组件。
func (h *ComponentHandler) MatchComponents(
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

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	var request models.MatchComponentsRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	result, err :=
		h.compService.MatchComponentsForActor(
			r.Context(),
			actor,
			&request,
		)
	if err != nil {
		h.handleCompError(w, err)
		return
	}

	utils.Success(w, result)
}

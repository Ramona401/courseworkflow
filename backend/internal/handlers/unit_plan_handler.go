package handlers

// unit_plan_handler.go — 单元方案 HTTP 处理器（大单元备课独立模块）
//
// 大单元挂载入口：
//   GET /api/v1/unit-plans/mountable[?subject=xxx]
//
// 上下文16教育域错误映射：
//   - 请求体伪造非K12具名出版社：400；
//   - 当前账号没有确定教学域、域冲突或与资源归属域不一致：403；
//   - 数据库或正式归属教育域解析失败：500；
//   - 其它既有权限、字段和资源不存在错误保持原语义。
//
// Service层会实时读取users.role并解析教育域。
// Handler继续传递claims.Role只是为了兼容其它既有方法签名，
// StartSession不再信任该JWT角色参数。

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// UnitPlanHandler 单元方案HTTP处理器。
type UnitPlanHandler struct {
	svc *services.UnitPlanService
}

// NewUnitPlanHandler 创建单元方案处理器。
func NewUnitPlanHandler(
	svc *services.UnitPlanService,
) *UnitPlanHandler {
	return &UnitPlanHandler{
		svc: svc,
	}
}

const unitPlanPathPrefix = "/api/v1/unit-plans"

// HandleCollection 处理：
//   - GET  /api/v1/unit-plans
//   - POST /api/v1/unit-plans
func (h *UnitPlanHandler) HandleCollection(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok {
		utils.Unauthorized(
			w,
			"未登录",
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		items, err :=
			h.svc.ListUnitPlans(
				r.Context(),
				claims.Role,
				claims.UserID,
			)
		if err != nil {
			utils.InternalError(
				w,
				err.Error(),
			)
			return
		}

		utils.Success(
			w,
			map[string]interface{}{
				"unit_plans": items,
				"total":      len(items),
			},
		)

	case http.MethodPost:
		var req models.StartUnitPlanRequest

		if err := json.NewDecoder(
			r.Body,
		).Decode(&req); err != nil {
			utils.BadRequest(
				w,
				"请求体解析失败",
			)
			return
		}

		plan, opening, err :=
			h.svc.StartSession(
				r.Context(),
				claims.Role,
				claims.UserID,
				&req,
			)
		if err != nil {
			h.mapError(
				w,
				err,
			)
			return
		}

		utils.Success(
			w,
			map[string]interface{}{
				"plan":    plan,
				"opening": opening,
			},
		)

	default:
		utils.BadRequest(
			w,
			"不支持的方法",
		)
	}
}

// HandleItem 处理：
//   - /api/v1/unit-plans/{id}
//   - /api/v1/unit-plans/{id}/chat
//   - /api/v1/unit-plans/{id}/save
//   - /api/v1/unit-plans/mountable
func (h *UnitPlanHandler) HandleItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok {
		utils.Unauthorized(
			w,
			"未登录",
		)
		return
	}

	rest := strings.Trim(
		strings.TrimPrefix(
			r.URL.Path,
			unitPlanPathPrefix+"/",
		),
		"/",
	)
	if rest == "" {
		utils.BadRequest(
			w,
			"缺少ID",
		)
		return
	}

	if rest == "mountable" {
		h.handleMountable(
			w,
			r,
			claims.Role,
			claims.UserID,
		)
		return
	}

	if strings.HasSuffix(
		rest,
		"/chat",
	) {
		h.handleChat(
			w,
			r,
			claims.Role,
			claims.UserID,
			strings.TrimSuffix(
				rest,
				"/chat",
			),
		)
		return
	}

	if strings.HasSuffix(
		rest,
		"/save",
	) {
		h.handleSave(
			w,
			r,
			claims.Role,
			claims.UserID,
			strings.TrimSuffix(
				rest,
				"/save",
			),
		)
		return
	}

	id := rest

	switch r.Method {
	case http.MethodGet:
		plan, messages, err :=
			h.svc.GetUnitPlan(
				r.Context(),
				claims.Role,
				claims.UserID,
				id,
			)
		if err != nil {
			h.mapError(
				w,
				err,
			)
			return
		}

		canEdit :=
			plan.CreatedBy ==
				claims.UserID

		utils.Success(
			w,
			map[string]interface{}{
				"plan":     plan,
				"messages": messages,
				"can_edit": canEdit,
			},
		)

	case http.MethodDelete:
		if err := h.svc.Delete(
			r.Context(),
			claims.Role,
			claims.UserID,
			id,
		); err != nil {
			h.mapError(
				w,
				err,
			)
			return
		}

		utils.Success(
			w,
			map[string]interface{}{
				"deleted": true,
			},
		)

	default:
		utils.BadRequest(
			w,
			"不支持的方法",
		)
	}
}

// handleMountable 列出可被教案挂载的active单元方案。
func (h *UnitPlanHandler) handleMountable(
	w http.ResponseWriter,
	r *http.Request,
	role string,
	userID string,
) {
	if r.Method != http.MethodGet {
		utils.BadRequest(
			w,
			"不支持的方法",
		)
		return
	}

	subject := strings.TrimSpace(
		r.URL.Query().Get(
			"subject",
		),
	)

	items, err :=
		h.svc.ListMountableUnitPlans(
			r.Context(),
			role,
			userID,
			subject,
		)
	if err != nil {
		utils.InternalError(
			w,
			err.Error(),
		)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"unit_plans": items,
			"total":      len(items),
		},
	)
}

// handleChat 处理单元方案一轮对话。
func (h *UnitPlanHandler) handleChat(
	w http.ResponseWriter,
	r *http.Request,
	role string,
	userID string,
	id string,
) {
	if r.Method != http.MethodPost {
		utils.BadRequest(
			w,
			"不支持的方法",
		)
		return
	}

	var req models.UnitPlanChatRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求体解析失败",
		)
		return
	}

	reply, err :=
		h.svc.Chat(
			r.Context(),
			role,
			userID,
			id,
			req.Message,
		)
	if err != nil {
		h.mapError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"reply": reply,
		},
	)
}

// handleSave 处理单元方案定稿保存。
func (h *UnitPlanHandler) handleSave(
	w http.ResponseWriter,
	r *http.Request,
	role string,
	userID string,
	id string,
) {
	if r.Method != http.MethodPost {
		utils.BadRequest(
			w,
			"不支持的方法",
		)
		return
	}

	var req models.SaveUnitPlanRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求体解析失败",
		)
		return
	}

	if err := h.svc.Save(
		r.Context(),
		role,
		userID,
		id,
		&req,
	); err != nil {
		h.mapError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"saved": true,
		},
	)
}

// mapError 统一映射单元方案业务错误。
func (h *UnitPlanHandler) mapError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		repository.ErrUnitPlanNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrUnitPlanFieldRequired,
	),
		errors.Is(
			err,
			services.ErrUnitPlanScopeInvalid,
		),
		errors.Is(
			err,
			services.ErrUnitPlanSaveEmpty,
		),
		errors.Is(
			err,
			services.ErrOutlinePublisherNotAllowed,
		),
		errors.Is(
			err,
			services.ErrOutlinePublisherUnavailable,
		):
		utils.BadRequest(
			w,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrUnitPlanNoPermission,
	),
		errors.Is(
			err,
			services.ErrUnitPlanNotOwner,
		),
		errors.Is(
			err,
			services.ErrOutlineEducationDomainRequired,
		),
		errors.Is(
			err,
			services.ErrOutlineEducationDomainConflict,
		),
		errors.Is(
			err,
			services.ErrOutlineEducationDomainMismatch,
		):
		utils.Forbidden(
			w,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrOutlineEducationDomainResolveFailed,
	):
		utils.InternalError(
			w,
			"单元方案教育域解析失败，请稍后重试",
		)

	default:
		utils.InternalError(
			w,
			err.Error(),
		)
	}
}

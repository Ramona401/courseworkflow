package handlers

// unit_plan_material_handler.go — 大单元方案参考资料HTTP处理器
//
// 接口：
//   GET    /api/v1/unit-plan-materials?unit_plan_id={id}
//   POST   /api/v1/unit-plan-materials?unit_plan_id={id}
//   DELETE /api/v1/unit-plan-materials/{material_id}?unit_plan_id={id}
//
// 权限：
//   - 可见单元方案的用户可以读取资料轻量列表；
//   - 只有单元方案创建者可以新增和删除资料；
//   - 真正的权限判断由UnitPlanMaterialService完成。

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

// UnitPlanMaterialHandler 大单元资料处理器。
type UnitPlanMaterialHandler struct {
	svc *services.UnitPlanMaterialService
}

// NewUnitPlanMaterialHandler 创建处理器。
func NewUnitPlanMaterialHandler(
	svc *services.UnitPlanMaterialService,
) *UnitPlanMaterialHandler {
	return &UnitPlanMaterialHandler{svc: svc}
}

const unitPlanMaterialPathPrefix = "/api/v1/unit-plan-materials"

// HandleCollection 处理资料列表和新增。
func (h *UnitPlanMaterialHandler) HandleCollection(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	unitPlanID := strings.TrimSpace(r.URL.Query().Get("unit_plan_id"))
	if unitPlanID == "" {
		utils.BadRequest(w, "缺少unit_plan_id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		items, canManage, err := h.svc.List(
			r.Context(),
			claims.Role,
			claims.UserID,
			unitPlanID,
		)
		if err != nil {
			h.mapError(w, err)
			return
		}

		utils.Success(w, map[string]interface{}{
			"materials":  items,
			"total":      len(items),
			"can_manage": canManage,
		})

	case http.MethodPost:
		var req models.CreateUnitPlanMaterialRequest

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			utils.BadRequest(w, "请求体解析失败")
			return
		}

		material, err := h.svc.Create(
			r.Context(),
			claims.Role,
			claims.UserID,
			unitPlanID,
			&req,
		)
		if err != nil {
			h.mapError(w, err)
			return
		}

		utils.Success(w, map[string]interface{}{
			"material": material,
		})

	default:
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET或POST",
		)
	}
}

// HandleItem 处理单条资料删除。
func (h *UnitPlanMaterialHandler) HandleItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持DELETE",
		)
		return
	}

	unitPlanID := strings.TrimSpace(r.URL.Query().Get("unit_plan_id"))
	if unitPlanID == "" {
		utils.BadRequest(w, "缺少unit_plan_id")
		return
	}

	materialID := strings.Trim(
		strings.TrimPrefix(
			r.URL.Path,
			unitPlanMaterialPathPrefix+"/",
		),
		"/",
	)

	if materialID == "" || strings.Contains(materialID, "/") {
		utils.BadRequest(w, "资料ID非法")
		return
	}

	if err := h.svc.Delete(
		r.Context(),
		claims.Role,
		claims.UserID,
		unitPlanID,
		materialID,
	); err != nil {
		h.mapError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"deleted": true,
	})
}

func (h *UnitPlanMaterialHandler) mapError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, repository.ErrUnitPlanNotFound),
		errors.Is(err, repository.ErrUnitPlanMaterialNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())

	case errors.Is(err, services.ErrUnitPlanMaterialNoPermission):
		utils.Forbidden(w, err.Error())

	case errors.Is(err, services.ErrUnitPlanMaterialTypeInvalid),
		errors.Is(err, services.ErrUnitPlanMaterialNameRequired),
		errors.Is(err, services.ErrUnitPlanMaterialContentEmpty),
		errors.Is(err, services.ErrUnitPlanMaterialTooLong):
		utils.BadRequest(w, err.Error())

	default:
		utils.InternalError(w, err.Error())
	}
}

package handlers

// unit_plan_handler.go — 单元方案 HTTP 处理器（大单元备课·独立模块）
//
// 大单元挂载（前端入口）新增：
//   GET /api/v1/unit-plans/mountable[?subject=xxx]
//   —— 供「教案挂载单元方案选择器」列出可挂载的单元方案（只列 active）。
//   该路径会被 /api/v1/unit-plans/ 前缀匹配到 HandleItem，故在 HandleItem 里
//   于解析 ID 前先拦截 rest == "mountable"，不影响现有 {id}/chat/save 等分支。

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

type UnitPlanHandler struct {
	svc *services.UnitPlanService
}

func NewUnitPlanHandler(svc *services.UnitPlanService) *UnitPlanHandler {
	return &UnitPlanHandler{svc: svc}
}

const unitPlanPathPrefix = "/api/v1/unit-plans"

// HandleCollection 处理 /api/v1/unit-plans（GET 列表 / POST 开始会话）
func (h *UnitPlanHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := h.svc.ListUnitPlans(r.Context(), claims.Role, claims.UserID)
		if err != nil {
			utils.InternalError(w, err.Error())
			return
		}
		utils.Success(w, map[string]interface{}{"unit_plans": items, "total": len(items)})
	case http.MethodPost:
		var req models.StartUnitPlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.BadRequest(w, "请求体解析失败")
			return
		}
		plan, opening, err := h.svc.StartSession(r.Context(), claims.Role, claims.UserID, &req)
		if err != nil {
			h.mapError(w, err)
			return
		}
		utils.Success(w, map[string]interface{}{"plan": plan, "opening": opening})
	default:
		utils.BadRequest(w, "不支持的方法")
	}
}

// HandleItem 处理 /api/v1/unit-plans/{id}[/chat|/save] 以及 /api/v1/unit-plans/mountable
func (h *UnitPlanHandler) HandleItem(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, unitPlanPathPrefix+"/"), "/")
	if rest == "" {
		utils.BadRequest(w, "缺少ID")
		return
	}

	// 大单元挂载：可挂载单元方案列表（只列 active），专属固定路径，先于 {id} 分支拦截。
	// GET /api/v1/unit-plans/mountable[?subject=xxx]
	if rest == "mountable" {
		h.handleMountable(w, r, claims.Role, claims.UserID)
		return
	}

	if strings.HasSuffix(rest, "/chat") {
		h.handleChat(w, r, claims.Role, claims.UserID, strings.TrimSuffix(rest, "/chat"))
		return
	}
	if strings.HasSuffix(rest, "/save") {
		h.handleSave(w, r, claims.Role, claims.UserID, strings.TrimSuffix(rest, "/save"))
		return
	}
	id := rest
	switch r.Method {
	case http.MethodGet:
		plan, msgs, err := h.svc.GetUnitPlan(r.Context(), claims.Role, claims.UserID, id)
		if err != nil {
			h.mapError(w, err)
			return
		}

		// can_edit 由后端根据当前登录用户与方案创建者确定，前端据此决定：
		//   - 创建者：草稿和已发布方案均可重新进入 AI 会话继续优化；
		//   - 其他可见用户：只能查看正式方案，不能进入续作编辑。
		// 真正的写权限仍由 Chat/Save 服务层再次校验，避免仅依赖前端按钮。
		canEdit := plan.CreatedBy == claims.UserID
		utils.Success(w, map[string]interface{}{
			"plan":     plan,
			"messages": msgs,
			"can_edit": canEdit,
		})
	case http.MethodDelete:
		if err := h.svc.Delete(r.Context(), claims.Role, claims.UserID, id); err != nil {
			h.mapError(w, err)
			return
		}
		utils.Success(w, map[string]interface{}{"deleted": true})
	default:
		utils.BadRequest(w, "不支持的方法")
	}
}

// handleMountable 列出可被教案挂载的单元方案（只列 active；可选 ?subject= 收窄学科）
func (h *UnitPlanHandler) handleMountable(w http.ResponseWriter, r *http.Request, role, userID string) {
	if r.Method != http.MethodGet {
		utils.BadRequest(w, "不支持的方法")
		return
	}
	subject := strings.TrimSpace(r.URL.Query().Get("subject"))
	items, err := h.svc.ListMountableUnitPlans(r.Context(), role, userID, subject)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"unit_plans": items, "total": len(items)})
}

func (h *UnitPlanHandler) handleChat(w http.ResponseWriter, r *http.Request, role, userID, id string) {
	if r.Method != http.MethodPost {
		utils.BadRequest(w, "不支持的方法")
		return
	}
	var req models.UnitPlanChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	reply, err := h.svc.Chat(r.Context(), role, userID, id, req.Message)
	if err != nil {
		h.mapError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{"reply": reply})
}

func (h *UnitPlanHandler) handleSave(w http.ResponseWriter, r *http.Request, role, userID, id string) {
	if r.Method != http.MethodPost {
		utils.BadRequest(w, "不支持的方法")
		return
	}
	var req models.SaveUnitPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	if err := h.svc.Save(r.Context(), role, userID, id, &req); err != nil {
		h.mapError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{"saved": true})
}

func (h *UnitPlanHandler) mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrUnitPlanNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrUnitPlanFieldRequired),
		errors.Is(err, services.ErrUnitPlanScopeInvalid),
		errors.Is(err, services.ErrUnitPlanSaveEmpty):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrUnitPlanNoPermission),
		errors.Is(err, services.ErrUnitPlanNotOwner):
		utils.Forbidden(w, err.Error())
	default:
		utils.InternalError(w, err.Error())
	}
}

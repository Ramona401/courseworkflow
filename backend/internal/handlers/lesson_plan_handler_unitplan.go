package handlers

// lesson_plan_handler_unitplan.go — 教案挂载/解除单元方案 HTTP 端点（大单元备课·前端入口）
//
// PUT /api/v1/lesson-plans/plans/{id}/unit-plan
// 请求体：{"unit_plan_id": "xxx"}（传空串 "" = 解除挂载，等于取消大单元绑定）
//
// 路径解析复用 extractLPMiddleID，错误映射复用 handleLPError（400/403/404 哨兵齐全）。
// 校验（仅作者本人 + 可编辑状态）在 service 层 UpdateLessonPlanUnitPlan，与课本挂载同款。

import (
	"encoding/json"
	"net/http"

	"tedna/internal/utils"
)

// updateLPUnitPlanRequest 单元方案挂载更新请求体
type updateLPUnitPlanRequest struct {
	UnitPlanID string `json:"unit_plan_id"`
}

// UpdateLessonPlanUnitPlan 挂载/解除教案关联的单元方案
//
// 起步首屏选定、对话中途挂载/更换、解除，三种操作都走这一个端点：
//   - 挂载/更换：unit_plan_id 传目标单元方案 ID
//   - 解除：    unit_plan_id 传空串 ""
// 后端注入层下一轮对话自动重读 lesson_plans.unit_plan_id 生效，无需刷新会话。
func (h *LessonPlanHandler) UpdateLessonPlanUnitPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	id := extractLPMiddleID(r.URL.Path, "/unit-plan")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req updateLPUnitPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.lpService.UpdateLessonPlanUnitPlan(r.Context(), id, userID, req.UnitPlanID); err != nil {
		h.handleLPError(w, err)
		return
	}
	// unit_plan_id 为空串表示解除，前端据此切换"已挂载/未挂载"展示
	mounted := req.UnitPlanID != ""
	utils.Success(w, map[string]interface{}{
		"message":      "单元方案关联已更新",
		"mounted":      mounted,
		"unit_plan_id": req.UnitPlanID,
	})
}

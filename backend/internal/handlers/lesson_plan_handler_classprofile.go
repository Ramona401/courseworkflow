package handlers

// lesson_plan_handler_classprofile.go — 教案挂载/解除班级学情卡 HTTP 端点（差异化教学·前端入口）
//
// PUT /api/v1/lesson-plans/plans/{id}/class-profile
// 请求体：{"class_profile_id": "xxx"}（传空串 "" = 解除挂载，等于取消班级关联）
//
// 路径解析复用 extractLPMiddleID，错误映射复用 handleLPError（400/403/404 哨兵齐全）。
// 校验（仅作者本人 + 可编辑状态）在 service 层 UpdateLessonPlanClassProfile，与单元方案/课本挂载同款。

import (
	"encoding/json"
	"net/http"

	"tedna/internal/utils"
)

// updateLPClassProfileRequest 班级学情卡挂载更新请求体
type updateLPClassProfileRequest struct {
	ClassProfileID string `json:"class_profile_id"`
}

// UpdateLessonPlanClassProfile 挂载/解除教案关联的班级学情卡
//
// 起步首屏选定、对话中途挂载/更换、解除，三种操作都走这一个端点：
//   - 挂载/更换：class_profile_id 传目标班级学情卡 ID
//   - 解除：    class_profile_id 传空串 ""
//
// 后端注入层下一轮对话自动重读 lesson_plans.class_profile_id 生效（analyze/design/write 三阶段），无需刷新会话。
func (h *LessonPlanHandler) UpdateLessonPlanClassProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	id := extractLPMiddleID(r.URL.Path, "/class-profile")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req updateLPClassProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.lpService.UpdateLessonPlanClassProfile(r.Context(), id, userID, req.ClassProfileID); err != nil {
		h.handleLPError(w, err)
		return
	}
	// class_profile_id 为空串表示解除，前端据此切换"已挂载/未挂载"展示
	mounted := req.ClassProfileID != ""
	utils.Success(w, map[string]interface{}{
		"message":          "班级学情关联已更新",
		"mounted":          mounted,
		"class_profile_id": req.ClassProfileID,
	})
}

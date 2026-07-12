package handlers

// lesson_plan_handler_courseoutline.go — 教案设置/清除「课程大纲教材版本」HTTP 端点（教材版本增强·前端入口）
//
// PUT /api/v1/lesson-plans/plans/{id}/course-outline-publisher
// 请求体：{"course_outline_publisher": "人教版"}
//
// 三态语义（关键差异：教材版本的空串是有意义的"通用版"，不能像班级学情那样用空串表达解除）：
//   - 字段为 null 或不传     → 解除关联（写 NULL，注入层不注入大纲）
//   - 字段为 ""（空串）       → 老师选了"通用/不限版本"（只注入 publisher 为空串的大纲）
//   - 字段为 "人教版" 等具名  → 选了该版本（只注入该版本大纲，零跨版本兜底）
//
// 用 *string 接收正是为了区分"传了空串"与"没传/传null"。路径解析复用 extractLPMiddleID，
// 错误映射复用 handleLPError。校验（作者本人 + 可编辑状态）在 service 层。
//
// 后端注入层下一轮对话自动重读 lesson_plans.course_outline_publisher 生效（analyze/design 阶段），无需刷新会话。

import (
	"encoding/json"
	"net/http"

	"tedna/internal/utils"
)

// updateLPCourseOutlinePublisherRequest 课程大纲教材版本设置请求体
//
// CourseOutlinePublisher 用指针：
//
//	nil（字段缺省/传 null）→ 解除关联
//	非 nil（含空串）        → 设置版本（空串=通用版）
type updateLPCourseOutlinePublisherRequest struct {
	CourseOutlinePublisher *string `json:"course_outline_publisher"`
}

// UpdateLessonPlanCourseOutlinePublisher 设置/解除教案选定的课程大纲教材版本
//
// 备课首屏选定、对话中途更换、解除，三种操作都走这一个端点：
//   - 设置/更换：course_outline_publisher 传目标版本（"人教版" 或 "" 表示通用版）
//   - 解除：    course_outline_publisher 传 null（或不传该字段）
func (h *LessonPlanHandler) UpdateLessonPlanCourseOutlinePublisher(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	id := extractLPMiddleID(r.URL.Path, "/course-outline-publisher")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req updateLPCourseOutlinePublisherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.lpService.UpdateLessonPlanCourseOutlinePublisher(r.Context(), id, userID, req.CourseOutlinePublisher); err != nil {
		h.handleLPError(w, err)
		return
	}
	// nil = 解除（未关联大纲）；非 nil = 已选定版本（含空串=通用版）。前端据此切换展示。
	mounted := req.CourseOutlinePublisher != nil
	resp := map[string]interface{}{
		"message": "课程大纲教材版本已更新",
		"mounted": mounted,
	}
	if req.CourseOutlinePublisher != nil {
		resp["course_outline_publisher"] = *req.CourseOutlinePublisher
	} else {
		resp["course_outline_publisher"] = nil
	}
	utils.Success(w, resp)
}

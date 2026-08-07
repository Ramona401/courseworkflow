package handlers

// lesson_plan_handler_courseoutline.go — 教案精确课程大纲与旧出版社挂载HTTP端点
//
// 正式端点：
//   PUT /api/v1/lesson-plans/plans/{id}/course-outline
//   请求体：{"course_outline_id": "唯一UUID"} 或 {"course_outline_id": null}
//
// 语义：
//   - 非空UUID：设置或更换唯一精确课程大纲；
//   - null、字段缺省或空字符串：解除关联并清空旧publisher-only残留；
//   - Service校验作者本人、可编辑状态、实时教育域、可见范围、学科和具体年级；
//   - Repository写入course_outline_id，数据库触发器固化出版社、册次和学制快照。
//
// 旧兼容端点：
//   PUT /api/v1/lesson-plans/plans/{id}/course-outline-publisher
//   仅供尚未迁移的旧调用点临时使用，新前端不得继续调用。

import (
	"encoding/json"
	"net/http"

	"tedna/internal/utils"
)

// updateLPCourseOutlineRequest 是正式精确课程大纲设置请求体。
//
// CourseOutlineID使用指针区分非空唯一ID与解除操作：
//   - 非nil且非空：设置或更换；
//   - nil或空字符串：解除。
//
// 字段缺省与显式null都按解除处理，保证接口幂等。
type updateLPCourseOutlineRequest struct {
	CourseOutlineID *string `json:"course_outline_id"`
}

// updateLPCourseOutlinePublisherRequest 是publisher-only旧端点请求体。
//
// CourseOutlinePublisher用指针保留旧三态：
//   - nil：解除关联；
//   - 非nil空串：通用版；
//   - 非nil具名字符串：指定出版社。
type updateLPCourseOutlinePublisherRequest struct {
	CourseOutlinePublisher *string `json:"course_outline_publisher"`
}

// UpdateLessonPlanCourseOutline 设置、更换或解除教案的唯一精确课程大纲。
func (h *LessonPlanHandler) UpdateLessonPlanCourseOutline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}

	id := extractLPMiddleID(r.URL.Path, "/course-outline")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}

	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var req updateLPCourseOutlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	snapshot, err := h.lpService.UpdateLessonPlanCourseOutline(
		r.Context(),
		id,
		userID,
		req.CourseOutlineID,
	)
	if err != nil {
		h.handleLPError(w, err)
		return
	}

	mounted := snapshot != nil && snapshot.CourseOutlineID != nil
	resp := map[string]interface{}{
		"message":                  "精确课程大纲关联已更新",
		"mounted":                  mounted,
		"course_outline_id":        nil,
		"course_outline_publisher": nil,
		"course_outline_volume":    nil,
		"school_system":            nil,
	}

	if snapshot != nil {
		resp["course_outline_id"] = snapshot.CourseOutlineID
		resp["course_outline_publisher"] = snapshot.CourseOutlinePublisher
		resp["course_outline_volume"] = snapshot.CourseOutlineVolume
		resp["school_system"] = snapshot.SchoolSystem
	}

	utils.Success(w, resp)
}

// UpdateLessonPlanCourseOutlinePublisher 设置或解除publisher-only旧挂载。
//
// @deprecated 新前端必须调用UpdateLessonPlanCourseOutline并提交唯一ID。
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

	if err := h.lpService.UpdateLessonPlanCourseOutlinePublisher(
		r.Context(),
		id,
		userID,
		req.CourseOutlinePublisher,
	); err != nil {
		h.handleLPError(w, err)
		return
	}

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

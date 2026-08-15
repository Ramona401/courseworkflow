package handlers

// courseware_ai_review_impact_plan_handler.go
//
// R-07 结构化影响方案HTTP入口。
//
// 路由：
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/impact-plans
//       从一条可信全局assistant消息生成不可变候选方案。
//       浏览器正文只能提交message_id。
//
//   GET /api/v1/courseware-ai-reviews/{session_id}/impact-plans/{plan_id}
//       读取已经冻结的教师Preview。
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/impact-plans/{plan_id}/apply
//       教师一次确认并原子应用明确勾选的operation。
//       请求正文只能提交version和selected_operation_ids。
//
// 安全边界：
//   1. session_id和plan_id均来自URL；
//   2. actor只从JWT重新构建；
//   3. 浏览器不能提交operations_json、payload、preconditions；
//   4. 浏览器不能提交AI正文、source_message_id或身份字段；
//   5. Draft生成时后端重读可信assistant消息；
//   6. Apply时Repository再次重读可信消息和全部业务目标；
//   7. 响应不暴露operations_hash、source_message_hash、created_by、applied_by或preconditions。

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"tedna/internal/services"
	"tedna/internal/utils"
)

const coursewareAIReviewImpactPlanBodyMaxBytes = 64 * 1024

type createCWAIReviewImpactPlanRequest struct {
	MessageID string `json:"message_id"`
}

type applyCWAIReviewImpactPlanRequest struct {
	Version              int      `json:"version"`
	SelectedOperationIDs []string `json:"selected_operation_ids"`
}

// isCoursewareAIReviewImpactPlanRoute 判断是否属于R-07结构化影响方案路径。
func isCoursewareAIReviewImpactPlanRoute(
	parts []string,
) bool {
	if len(parts) < 2 ||
		len(parts) > 4 ||
		parts[0] == "items" ||
		parts[1] != "impact-plans" {
		return false
	}

	if len(parts) == 2 {
		return true
	}

	if len(parts) == 3 {
		return strings.TrimSpace(parts[2]) != ""
	}

	return strings.TrimSpace(parts[2]) != "" &&
		parts[3] == "apply"
}

// HandleImpactPlanRoute 处理Draft、Preview和最终原子Apply。
func (h *CoursewareAIReviewHandler) HandleImpactPlanRoute(
	w http.ResponseWriter,
	r *http.Request,
	parts []string,
) {
	actor, ok := buildCoursewareAIReviewActor(r)
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	if h == nil || h.runner == nil {
		utils.InternalError(w, "课件AI审核执行器未初始化")
		return
	}

	sessionID := ""
	if len(parts) > 0 {
		sessionID = strings.TrimSpace(parts[0])
	}

	switch {
	case len(parts) == 2 &&
		r.Method == http.MethodPost:
		h.createImpactPlan(
			w,
			r,
			sessionID,
			actor,
		)

	case len(parts) == 3 &&
		r.Method == http.MethodGet:
		h.getImpactPlan(
			w,
			r,
			sessionID,
			parts[2],
			actor,
		)

	case len(parts) == 4 &&
		parts[3] == "apply" &&
		r.Method == http.MethodPost:
		h.applyImpactPlan(
			w,
			r,
			sessionID,
			parts[2],
			actor,
		)

	default:
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"影响方案路由或请求方法无效",
		)
	}
}

func (h *CoursewareAIReviewHandler) createImpactPlan(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req createCWAIReviewImpactPlanRequest
	if !decodeCWAIReviewImpactPlanRequest(
		w,
		r,
		&req,
	) {
		return
	}

	req.MessageID = strings.TrimSpace(req.MessageID)
	if req.MessageID == "" {
		utils.BadRequest(w, "缺少可信全局讨论消息ID")
		return
	}

	record, err :=
		h.runner.CreateCWAIReviewImpactPlanDraft(
			r.Context(),
			sessionID,
			req.MessageID,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewImpactPlanView(record),
	)
}

func (h *CoursewareAIReviewHandler) getImpactPlan(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	planID string,
	actor *services.CoursewareActorContext,
) {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		utils.BadRequest(w, "缺少影响方案ID")
		return
	}

	record, err :=
		h.runner.GetCWAIReviewImpactPlan(
			r.Context(),
			sessionID,
			planID,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewImpactPlanView(record),
	)
}

func (h *CoursewareAIReviewHandler) applyImpactPlan(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	planID string,
	actor *services.CoursewareActorContext,
) {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		utils.BadRequest(w, "缺少影响方案ID")
		return
	}

	var req applyCWAIReviewImpactPlanRequest
	if !decodeCWAIReviewImpactPlanRequest(
		w,
		r,
		&req,
	) {
		return
	}

	// V1计划只允许draft/version=1进入一次最终Apply。
	if req.Version != 1 {
		utils.BadRequest(w, "影响方案版本无效，请刷新后重试")
		return
	}

	if len(req.SelectedOperationIDs) == 0 {
		utils.BadRequest(w, "请至少选择一项需要应用的操作")
		return
	}

	record, err :=
		h.runner.ApplyCWAIReviewImpactPlan(
			r.Context(),
			sessionID,
			planID,
			req.Version,
			req.SelectedOperationIDs,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewImpactPlanView(record),
	)
}

func decodeCWAIReviewImpactPlanRequest(
	w http.ResponseWriter,
	r *http.Request,
	target interface{},
) bool {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		coursewareAIReviewImpactPlanBodyMaxBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误或内容过大",
		)
		return false
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		utils.BadRequest(
			w,
			"请求正文只能包含一个JSON对象",
		)
		return false
	}

	return true
}

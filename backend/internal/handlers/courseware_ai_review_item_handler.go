package handlers

// courseware_ai_review_item_handler.go
//
// 课件AI审核整改项子路由。
//
// 主要入口：
//   - POST|GET /courseware-ai-reviews/{session_id}/items
//   - GET /courseware-ai-reviews/items?courseware_id={id}
//   - GET /courseware-ai-reviews/items/{item_id}
//   - POST /items/{item_id}/messages
//   - POST /items/{item_id}/execution-note
//   - POST /items/{item_id}/create-related-improvement
//   - POST /items/{item_id}/generate-instruction
//   - POST /items/{item_id}/confirm
//   - POST /items/{item_id}/resolve
//   - POST /items/{item_id}/recheck
//   - POST /items/{item_id}/dismiss
//   - POST /items/{item_id}/restore
//
// create-related-improvement只创建独立新问题，不修改来源问题、页面或审核决定。

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"tedna/internal/services"
	"tedna/internal/utils"
)

const (
	coursewareAIReviewItemBodyMaxBytes = 128 * 1024

	coursewareAIReviewDismissReasonMaxRunes = 500
)

type materializeCWAIReviewItemsRequest struct {
	FindingIDs []string `json:"finding_ids"`
}

type messageCWAIReviewItemRequest struct {
	Content string `json:"content"`
}

type addCWAIReviewItemExecutionNoteRequest struct {
	Content string `json:"content"`
}

type createCWAIReviewRelatedImprovementRequest struct {
	Content string `json:"content"`
}

type confirmCWAIReviewItemRequest struct {
	Instruction string `json:"instruction"`
}

type dismissCWAIReviewItemRequest struct {
	Reason string `json:"reason"`
}

// HandleReviewItemRoute 处理现有AI审核通配路由下的整改项子路径。
func (h *CoursewareAIReviewHandler) HandleReviewItemRoute(
	w http.ResponseWriter,
	r *http.Request,
	parts []string,
) {
	actor, ok := buildCoursewareAIReviewActor(r)
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	if isCWAIReviewSessionItemsRoute(parts) {
		switch r.Method {
		case http.MethodGet:
			h.listSessionReviewItems(w, r, parts[0], actor)

		case http.MethodPost:
			h.materializeReviewItems(w, r, parts[0], actor)

		default:
			utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET或POST请求")
		}
		return
	}

	if len(parts) == 0 || parts[0] != "items" {
		utils.BadRequest(w, "课件整改项路由无效")
		return
	}

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		h.listOwnerReviewItems(w, r, actor)

	case len(parts) == 2 && r.Method == http.MethodGet:
		h.getReviewItemDiscussion(w, r, parts[1], actor)

	case len(parts) == 3 &&
		parts[2] == "messages" &&
		r.Method == http.MethodPost:
		h.messageReviewItem(w, r, parts[1], actor)

	case len(parts) == 3 &&
		parts[2] == "execution-note" &&
		r.Method == http.MethodPost:
		h.addReviewItemExecutionNote(w, r, parts[1], actor)

	case len(parts) == 3 &&
		parts[2] == "create-related-improvement" &&
		r.Method == http.MethodPost:
		h.createRelatedImprovement(w, r, parts[1], actor)

	case len(parts) == 3 &&
		parts[2] == "generate-instruction" &&
		r.Method == http.MethodPost:
		h.generateReviewItemInstruction(w, r, parts[1], actor)

	case len(parts) == 3 &&
		parts[2] == "confirm" &&
		r.Method == http.MethodPost:
		h.confirmReviewItem(w, r, parts[1], actor)

	case len(parts) == 3 &&
		parts[2] == "resolve" &&
		r.Method == http.MethodPost:
		h.resolveSelfReviewItem(w, r, parts[1], actor)

	case len(parts) == 3 &&
		parts[2] == "recheck" &&
		r.Method == http.MethodPost:
		h.recheckReviewItem(w, r, parts[1], actor)

	case len(parts) == 3 &&
		parts[2] == "dismiss" &&
		r.Method == http.MethodPost:
		h.dismissReviewItem(w, r, parts[1], actor)

	case len(parts) == 3 &&
		parts[2] == "restore" &&
		r.Method == http.MethodPost:
		h.restoreReviewItem(w, r, parts[1], actor)

	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "整改项路由或请求方法无效")
	}
}

func isCWAIReviewSessionItemsRoute(parts []string) bool {
	return len(parts) == 2 &&
		parts[0] != "items" &&
		parts[1] == "items"
}

func (h *CoursewareAIReviewHandler) materializeReviewItems(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.service == nil {
		utils.InternalError(w, "课件AI审核服务未初始化")
		return
	}

	var req materializeCWAIReviewItemsRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	items, err := h.service.MaterializeCWAIReviewFindings(
		r.Context(),
		sessionID,
		req.FindingIDs,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"items":   buildCoursewareAIReviewItemViews(items),
			"message": "已创建或复用选中的课件整改项",
		},
	)
}

func (h *CoursewareAIReviewHandler) listSessionReviewItems(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.service == nil {
		utils.InternalError(w, "课件AI审核服务未初始化")
		return
	}

	items, err := h.service.ListCWAIReviewSessionItems(
		r.Context(),
		sessionID,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"items": buildCoursewareAIReviewItemViews(items),
		},
	)
}

func (h *CoursewareAIReviewHandler) listOwnerReviewItems(
	w http.ResponseWriter,
	r *http.Request,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.service == nil {
		utils.InternalError(w, "课件AI审核服务未初始化")
		return
	}

	coursewareID := strings.TrimSpace(
		r.URL.Query().Get("courseware_id"),
	)
	if coursewareID == "" {
		utils.BadRequest(w, "缺少courseware_id")
		return
	}

	bundle, err := h.service.GetCWOwnerReviewRemediation(
		r.Context(),
		coursewareID,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewOwnerRemediationView(bundle),
	)
}

func (h *CoursewareAIReviewHandler) getReviewItemDiscussion(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.runner == nil {
		utils.InternalError(w, "课件AI审核执行器未初始化")
		return
	}

	result, err := h.runner.GetCWReviewItemDiscussion(
		r.Context(),
		itemID,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemDiscussionView(result),
	)
}

func (h *CoursewareAIReviewHandler) messageReviewItem(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.runner == nil {
		utils.InternalError(w, "课件AI审核执行器未初始化")
		return
	}

	var req messageCWAIReviewItemRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	result, err := h.runner.MessageCWReviewItem(
		r.Context(),
		itemID,
		req.Content,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemDiscussionView(result),
	)
}

// addReviewItemExecutionNote 保存课件作者对已正式交付整改项的本次执行补充。
//
// 该入口只追加作者可见文本消息，不调用AI、不修改确认要求、不推进或回退整改状态。
func (h *CoursewareAIReviewHandler) addReviewItemExecutionNote(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.service == nil {
		utils.InternalError(w, "课件AI审核服务未初始化")
		return
	}

	var req addCWAIReviewItemExecutionNoteRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	result, err := h.service.AddCWReviewItemExecutionNote(
		r.Context(),
		itemID,
		req.Content,
		actor,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCWReviewItemExecutionNoteInvalid):
			utils.BadRequest(w, "请填写不超过2000字的本次执行补充")

		case errors.Is(err, services.ErrCWReviewItemExecutionNoteForbidden):
			utils.Fail(
				w,
				http.StatusForbidden,
				"只有课件作者可以补充本次执行说明",
			)

		case errors.Is(err, services.ErrCWReviewItemExecutionNoteUnavailable):
			utils.Fail(
				w,
				http.StatusConflict,
				"当前整改状态不能添加本次执行补充",
			)

		default:
			h.handleError(w, err)
		}
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemDiscussionView(result),
	)
}

// createRelatedImprovement 把疑似目标漂移的文字保存为新的独立改进项。
//
// 本动作不会确认、覆盖或删除来源问题的任何内容。
func (h *CoursewareAIReviewHandler) createRelatedImprovement(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.service == nil {
		utils.InternalError(w, "课件AI审核服务未初始化")
		return
	}

	var req createCWAIReviewRelatedImprovementRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	item, err := h.service.CreateCWReviewGoalDriftItem(
		r.Context(),
		itemID,
		req.Content,
		actor,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCWReviewGoalDriftContentInvalid):
			utils.BadRequest(
				w,
				"请填写需要作为新问题单独处理的内容",
			)

		case errors.Is(err, services.ErrCWReviewGoalDriftForbidden):
			utils.Fail(
				w,
				http.StatusForbidden,
				"当前身份不能从这条问题创建新的改进项",
			)

		case errors.Is(err, services.ErrCWReviewGoalDriftUnavailable):
			utils.Fail(
				w,
				http.StatusConflict,
				"当前问题已经不能继续拆分，请刷新后重新检查",
			)

		default:
			h.handleError(w, err)
		}
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"item":    buildCoursewareAIReviewItemView(item),
			"message": "已创建新的独立改进项",
		},
	)
}

func (h *CoursewareAIReviewHandler) generateReviewItemInstruction(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.runner == nil {
		utils.InternalError(w, "课件AI审核执行器未初始化")
		return
	}

	result, err := h.runner.GenerateCWReviewItemInstruction(
		r.Context(),
		itemID,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemDiscussionView(result),
	)
}

func (h *CoursewareAIReviewHandler) confirmReviewItem(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.runner == nil {
		utils.InternalError(w, "课件AI审核执行器未初始化")
		return
	}

	var req confirmCWAIReviewItemRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	result, err := h.runner.ConfirmCWReviewItemInstruction(
		r.Context(),
		itemID,
		req.Instruction,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemDiscussionView(result),
	)
}

// resolveSelfReviewItem 由课件作者确认自己的自审问题已经解决。
func (h *CoursewareAIReviewHandler) resolveSelfReviewItem(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.service == nil {
		utils.InternalError(w, "课件AI审核服务未初始化")
		return
	}

	result, err := h.service.ResolveSelfCWReviewItem(
		r.Context(),
		itemID,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"item": buildCoursewareAIReviewItemView(result.Item),
			"messages": buildCoursewareAIReviewItemDiscussionView(
				result,
			).Messages,
			"message": "已确认这条自审问题已经解决",
		},
	)
}

func (h *CoursewareAIReviewHandler) dismissReviewItem(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.service == nil {
		utils.InternalError(w, "课件AI审核服务未初始化")
		return
	}

	var req dismissCWAIReviewItemRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" ||
		utf8.RuneCountInString(req.Reason) >
			coursewareAIReviewDismissReasonMaxRunes {
		utils.BadRequest(w, "请填写不超过500字的忽略原因")
		return
	}

	result, err := h.service.DismissCWReviewItem(
		r.Context(),
		itemID,
		req.Reason,
		actor,
	)
	if err != nil {
		if errors.Is(
			err,
			services.ErrCWReviewItemDismissReasonInvalid,
		) {
			utils.BadRequest(w, "请填写不超过500字的忽略原因")
			return
		}

		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemDiscussionView(result),
	)
}

func (h *CoursewareAIReviewHandler) restoreReviewItem(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.service == nil {
		utils.InternalError(w, "课件AI审核服务未初始化")
		return
	}

	result, err := h.service.RestoreCWReviewItem(
		r.Context(),
		itemID,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemDiscussionView(result),
	)
}

func decodeCWAIReviewItemRequest(
	w http.ResponseWriter,
	r *http.Request,
	target interface{},
) bool {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		coursewareAIReviewItemBodyMaxBytes,
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

	return true
}

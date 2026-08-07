package handlers

// courseware_ai_review_global_discussion_handler.go
//
// 课件AI审核跨页面、跨问题全局讨论及人工治理HTTP入口。
//
// 讨论路由：
//   GET /api/v1/courseware-ai-reviews/{session_id}/global-discussion
//       读取当前审核员自己的全局讨论历史、最新AI建议和已确认关系。
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/global-discussion
//       提交综合问题和2至12条整改项ID，AI返回关系分析及逐项候选指令。
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/global-discussion/adopt
//       采用可信全局助手消息中的一条候选修改指令。
//       采用后仍须在单条整改项中独立确认。
//
// 人工治理路由：
//   POST /api/v1/courseware-ai-reviews/{session_id}/global-discussion/manual-items
//       人工创建一个整课问题，或按多个稳定页面拆分为多条页级问题。
//
//   GET /api/v1/courseware-ai-reviews/{session_id}/global-discussion/relations
//       读取已经人工确认的结构化关系及不可变操作历史。
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/global-discussion/relations
//       人工确认AI建议的重复、冲突、合并、依赖或可能连带解决关系。
//       已取消的同一关系会通过同一入口重新确认。
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/global-discussion/relations/{relation_id}
//       人工取消一条已经确认的关系，必须填写原因。
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/global-discussion/dismiss
//       人工确认可信AI的consider_dismiss建议，最终复用原有独立忽略服务。
//
// 安全边界：
//   1. 候选正文、关系说明和建议类型均从后端可信消息重新读取；
//   2. 浏览器不能提交AI关系说明来替换可信元数据；
//   3. AI不能自动创建问题、确认关系或忽略问题；
//   4. 所有状态动作都需要独立POST请求；
//   5. 所有操作都不会修改页面或提交人工审核决定。

import (
	"net/http"

	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

type messageCWAIReviewGlobalDiscussionRequest struct {
	Content string   `json:"content"`
	ItemIDs []string `json:"item_ids"`
}

type adoptCWAIReviewGlobalProposalRequest struct {
	MessageID string `json:"message_id"`
	ItemID    string `json:"item_id"`
}

type createCWAIReviewGlobalManualItemsRequest struct {
	MessageID string `json:"message_id"`

	Title                string   `json:"title"`
	Description          string   `json:"description"`
	CandidateInstruction string   `json:"candidate_instruction"`
	Severity             string   `json:"severity"`
	Dimension            string   `json:"dimension"`
	PageIDs              []string `json:"page_ids"`
}

type confirmCWAIReviewGlobalRelationRequest struct {
	MessageID string `json:"message_id"`

	RelationType string `json:"relation_type"`
	SourceItemID string `json:"source_item_id"`
	TargetItemID string `json:"target_item_id"`
}

type cancelCWAIReviewGlobalRelationRequest struct {
	Reason string `json:"reason"`
}

type confirmCWAIReviewGlobalDismissalRequest struct {
	MessageID string `json:"message_id"`
	ItemID    string `json:"item_id"`
	Reason    string `json:"reason"`
}

type coursewareAIReviewGlobalDiscussionView struct {
	Messages []*coursewareAIReviewItemMessageView `json:"messages"`

	Summary         string                              `json:"summary"`
	Relations       []services.CWAIReviewGlobalRelation `json:"relations"`
	Proposals       []services.CWAIReviewGlobalProposal `json:"proposals"`
	SelectedItemIDs []string                            `json:"selected_item_ids"`
	LatestMessageID string                              `json:"latest_message_id"`

	// GovernanceRelations只包含人工明确确认过的持久化关系。
	// 与上方AI临时建议Relations严格区分，不能把AI建议误认为已确认事实。
	GovernanceRelations []*coursewareAIReviewItemRelationView `json:"governance_relations"`
}

// isCoursewareAIReviewGlobalDiscussionRoute 判断是否属于会话级全局讨论路径。
func isCoursewareAIReviewGlobalDiscussionRoute(
	parts []string,
) bool {
	if len(parts) < 2 || len(parts) > 4 {
		return false
	}
	if parts[0] == "items" ||
		parts[1] != "global-discussion" {
		return false
	}

	if len(parts) == 2 {
		return true
	}

	switch parts[2] {
	case "adopt",
		"manual-items",
		"relations",
		"dismiss":
	default:
		return false
	}

	// 四段路径只允许：
	// /{session}/global-discussion/relations/{relation_id}
	return len(parts) == 3 ||
		(len(parts) == 4 && parts[2] == "relations")
}

// HandleGlobalDiscussionRoute 处理全局讨论、候选采用和人工治理。
func (h *CoursewareAIReviewHandler) HandleGlobalDiscussionRoute(
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
		sessionID = parts[0]
	}

	switch {
	case len(parts) == 2 &&
		r.Method == http.MethodGet:
		h.getGlobalDiscussion(
			w,
			r,
			sessionID,
			actor,
		)

	case len(parts) == 2 &&
		r.Method == http.MethodPost:
		h.messageGlobalDiscussion(
			w,
			r,
			sessionID,
			actor,
		)

	case len(parts) == 3 &&
		parts[2] == "adopt" &&
		r.Method == http.MethodPost:
		h.adoptGlobalDiscussionProposal(
			w,
			r,
			sessionID,
			actor,
		)

	case len(parts) == 3 &&
		parts[2] == "manual-items" &&
		r.Method == http.MethodPost:
		h.createGlobalDiscussionManualItems(
			w,
			r,
			sessionID,
			actor,
		)

	case len(parts) == 3 &&
		parts[2] == "relations" &&
		r.Method == http.MethodGet:
		h.listGlobalDiscussionRelations(
			w,
			r,
			sessionID,
			actor,
		)

	case len(parts) == 3 &&
		parts[2] == "relations" &&
		r.Method == http.MethodPost:
		h.confirmGlobalDiscussionRelation(
			w,
			r,
			sessionID,
			actor,
		)

	case len(parts) == 4 &&
		parts[2] == "relations" &&
		r.Method == http.MethodPost:
		h.cancelGlobalDiscussionRelation(
			w,
			r,
			sessionID,
			parts[3],
			actor,
		)

	case len(parts) == 3 &&
		parts[2] == "dismiss" &&
		r.Method == http.MethodPost:
		h.confirmGlobalDiscussionDismissal(
			w,
			r,
			sessionID,
			actor,
		)

	default:
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"全局讨论路由或请求方法无效",
		)
	}
}

func (h *CoursewareAIReviewHandler) getGlobalDiscussion(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	result, err :=
		h.runner.GetCWAIReviewGlobalDiscussion(
			r.Context(),
			sessionID,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	records, err :=
		h.runner.ListCWAIReviewGlobalRelations(
			r.Context(),
			sessionID,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	view :=
		buildCoursewareAIReviewGlobalDiscussionView(
			result,
		)
	view.GovernanceRelations =
		buildCoursewareAIReviewGlobalRelationRecordViews(
			records,
		)

	utils.Success(w, view)
}

func (h *CoursewareAIReviewHandler) messageGlobalDiscussion(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req messageCWAIReviewGlobalDiscussionRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	result, err :=
		h.runner.MessageCWAIReviewGlobalDiscussion(
			r.Context(),
			sessionID,
			req.Content,
			req.ItemIDs,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	records, err :=
		h.runner.ListCWAIReviewGlobalRelations(
			r.Context(),
			sessionID,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	view :=
		buildCoursewareAIReviewGlobalDiscussionView(
			result,
		)
	view.GovernanceRelations =
		buildCoursewareAIReviewGlobalRelationRecordViews(
			records,
		)

	utils.Success(w, view)
}

func (h *CoursewareAIReviewHandler) adoptGlobalDiscussionProposal(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req adoptCWAIReviewGlobalProposalRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	result, err :=
		h.runner.AdoptCWAIReviewGlobalProposal(
			r.Context(),
			sessionID,
			req.MessageID,
			req.ItemID,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemDiscussionView(
			result,
		),
	)
}

func (h *CoursewareAIReviewHandler) createGlobalDiscussionManualItems(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req createCWAIReviewGlobalManualItemsRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	items, err :=
		h.runner.CreateCWAIReviewGlobalManualItems(
			r.Context(),
			sessionID,
			req.MessageID,
			&services.CWAIReviewGlobalManualItemInput{
				Title:                req.Title,
				Description:          req.Description,
				CandidateInstruction: req.CandidateInstruction,
				Severity:             req.Severity,
				Dimension:            req.Dimension,
				PageIDs:              req.PageIDs,
			},
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"items": buildCoursewareAIReviewItemViews(
				items,
			),
			"message": "已人工创建整改项；候选指令仍需逐条独立确认",
		},
	)
}

func (h *CoursewareAIReviewHandler) listGlobalDiscussionRelations(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	records, err :=
		h.runner.ListCWAIReviewGlobalRelations(
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
			"relations": buildCoursewareAIReviewGlobalRelationRecordViews(
				records,
			),
		},
	)
}

func (h *CoursewareAIReviewHandler) confirmGlobalDiscussionRelation(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req confirmCWAIReviewGlobalRelationRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	record, err :=
		h.runner.ConfirmCWAIReviewGlobalRelation(
			r.Context(),
			sessionID,
			req.MessageID,
			&services.CWAIReviewGlobalRelationConfirmInput{
				RelationType: req.RelationType,
				SourceItemID: req.SourceItemID,
				TargetItemID: req.TargetItemID,
			},
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewGlobalRelationRecordView(
			record,
		),
	)
}

func (h *CoursewareAIReviewHandler) cancelGlobalDiscussionRelation(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	relationID string,
	actor *services.CoursewareActorContext,
) {
	var req cancelCWAIReviewGlobalRelationRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	record, err :=
		h.runner.CancelCWAIReviewGlobalRelation(
			r.Context(),
			sessionID,
			relationID,
			req.Reason,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewGlobalRelationRecordView(
			record,
		),
	)
}

func (h *CoursewareAIReviewHandler) confirmGlobalDiscussionDismissal(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req confirmCWAIReviewGlobalDismissalRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	result, err :=
		h.runner.ConfirmCWAIReviewGlobalDismissal(
			r.Context(),
			sessionID,
			req.MessageID,
			req.ItemID,
			req.Reason,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemDiscussionView(
			result,
		),
	)
}

func buildCoursewareAIReviewGlobalDiscussionView(
	result *services.CWAIReviewGlobalDiscussionResult,
) *coursewareAIReviewGlobalDiscussionView {
	if result == nil {
		return &coursewareAIReviewGlobalDiscussionView{
			Messages:            []*coursewareAIReviewItemMessageView{},
			Relations:           []services.CWAIReviewGlobalRelation{},
			Proposals:           []services.CWAIReviewGlobalProposal{},
			SelectedItemIDs:     []string{},
			GovernanceRelations: []*coursewareAIReviewItemRelationView{},
		}
	}

	return &coursewareAIReviewGlobalDiscussionView{
		Messages: buildCoursewareAIReviewGlobalMessageViews(
			result.Messages,
		),
		Summary:             result.Summary,
		Relations:           result.Relations,
		Proposals:           result.Proposals,
		SelectedItemIDs:     result.SelectedItemIDs,
		LatestMessageID:     result.LatestMessageID,
		GovernanceRelations: []*coursewareAIReviewItemRelationView{},
	}
}

func buildCoursewareAIReviewGlobalMessageViews(
	messages []*models.CoursewareAIReviewMessage,
) []*coursewareAIReviewItemMessageView {
	result := make(
		[]*coursewareAIReviewItemMessageView,
		0,
		len(messages),
	)

	for _, message := range messages {
		if message == nil {
			continue
		}

		result = append(
			result,
			&coursewareAIReviewItemMessageView{
				ID:         message.ID,
				Role:       message.Role,
				Content:    message.Content,
				MetaJSON:   message.CitationsJSON,
				ModelUsed:  message.ModelUsed,
				TokensUsed: message.TokensUsed,
				CreatedAt:  message.CreatedAt,
			},
		)
	}

	return result
}

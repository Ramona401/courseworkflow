package handlers

// courseware_ai_review_item_relation_handler.go
//
// 课件AI审核问题清单的会话级关系治理HTTP入口。
//
// 路由：
//   GET /api/v1/courseware-ai-reviews/{session_id}/relations
//       读取当前用户已经明确确认的关系及追加式事件历史。
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/relations
//       不依赖AI建议，由用户直接建立重复、冲突、合并、依赖或
//       可能连带解决关系。已取消的同一关系会重新启用。
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/relations/{relation_id}/cancel
//       取消一条有效关系，必须填写原因；关系实体和历史不会删除。
//
// 这些接口不会改变整改项状态、确认指令、页面内容或审核决定。

import (
	"net/http"

	"tedna/internal/services"
	"tedna/internal/utils"
)

type confirmCWAIReviewManualRelationRequest struct {
	RelationType string `json:"relation_type"`
	SourceItemID string `json:"source_item_id"`
	TargetItemID string `json:"target_item_id"`
	Explanation  string `json:"explanation"`
}

type cancelCWAIReviewItemRelationRequest struct {
	Reason string `json:"reason"`
}

// isCoursewareAIReviewRelationRoute 判断是否属于会话级问题关系治理路径。
func isCoursewareAIReviewRelationRoute(
	parts []string,
) bool {
	if len(parts) == 2 {
		return parts[0] != "items" &&
			parts[1] == "relations"
	}

	return len(parts) == 4 &&
		parts[0] != "items" &&
		parts[1] == "relations" &&
		parts[3] == "cancel"
}

// HandleReviewItemRelationRoute 处理会话级关系读取、确认和取消。
func (h *CoursewareAIReviewHandler) HandleReviewItemRelationRoute(
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
		h.listReviewItemRelations(
			w,
			r,
			sessionID,
			actor,
		)

	case len(parts) == 2 &&
		r.Method == http.MethodPost:
		h.confirmManualReviewItemRelation(
			w,
			r,
			sessionID,
			actor,
		)

	case len(parts) == 4 &&
		r.Method == http.MethodPost:
		h.cancelReviewItemRelation(
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
			"整改项关系路由或请求方法无效",
		)
	}
}

func (h *CoursewareAIReviewHandler) listReviewItemRelations(
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

func (h *CoursewareAIReviewHandler) confirmManualReviewItemRelation(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req confirmCWAIReviewManualRelationRequest
	if !decodeCWAIReviewItemRequest(w, r, &req) {
		return
	}

	record, err :=
		h.runner.ConfirmCWAIReviewManualRelation(
			r.Context(),
			sessionID,
			&services.CWAIReviewManualRelationConfirmInput{
				RelationType: req.RelationType,
				SourceItemID: req.SourceItemID,
				TargetItemID: req.TargetItemID,
				Explanation:  req.Explanation,
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

func (h *CoursewareAIReviewHandler) cancelReviewItemRelation(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	relationID string,
	actor *services.CoursewareActorContext,
) {
	var req cancelCWAIReviewItemRelationRequest
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

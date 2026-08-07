package handlers

// courseware_ai_review_item_recheck_handler.go
//
// 作者重新检查页面变化问题的HTTP入口。
//
// POST /api/v1/courseware-ai-reviews/items/{item_id}/recheck
//
// 请求不接收页面HTML、页面指纹、问题状态或确认人。
// 后端重新读取当前页面和当前登录用户，完成全部权限与状态校验。

import (
	"net/http"

	"tedna/internal/services"
	"tedna/internal/utils"
)

// recheckReviewItem 由课件作者重新检查当前页面。
func (h *CoursewareAIReviewHandler) recheckReviewItem(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil ||
		h.service == nil {
		utils.InternalError(
			w,
			"课件AI审核服务未初始化",
		)
		return
	}

	result, err :=
		h.service.RecheckCWReviewItem(
			r.Context(),
			itemID,
			actor,
		)
	if err != nil {
		h.handleError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"item": buildCoursewareAIReviewItemView(
				result.Item,
			),
			"messages": buildCoursewareAIReviewItemDiscussionView(
				result,
			).Messages,
			"message": "已重新检查当前页面，现进入人工确认阶段",
		},
	)
}

package handlers

// lesson_plan_handler_textbook.go — 教案课本关联HTTP端点（迭代3.5 A2-2 新增）
//
// PUT /api/v1/lesson-plans/plans/{id}/textbooks
// 请求体：{"textbook_page_ids": ["id1","id2",...]}（传空数组=解除全部关联）
//
// 路径解析复用 extractLPMiddleID，错误映射复用 handleLPError（400/403/404哨兵齐全）。

import (
	"encoding/json"
	"net/http"

	"tedna/internal/utils"
)

// updateLPTextbooksRequest 课本关联更新请求体
type updateLPTextbooksRequest struct {
	TextbookPageIDs []string `json:"textbook_page_ids"`
}

// UpdateLessonPlanTextbooks 更新教案关联的课本页面（对话模式中途挂载入口）
func (h *LessonPlanHandler) UpdateLessonPlanTextbooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	id := extractLPMiddleID(r.URL.Path, "/textbooks")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req updateLPTextbooksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	// 软上限：单份教案最多关联20张课本页（OCR文本拼接进提示词，防止上下文爆量）
	if len(req.TextbookPageIDs) > 20 {
		utils.BadRequest(w, "一份教案最多关联20张课本页")
		return
	}
	if err := h.lpService.UpdateLessonPlanTextbooks(r.Context(), id, userID, req.TextbookPageIDs); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{
		"message": "课本关联已更新",
		"count":   len(req.TextbookPageIDs),
	})
}

package handlers

// component_extraction_handler.go — 组件萃取队列HTTP处理器。
//
// 列表和确认操作必须先从JWT Claims构建可信Actor。
// 普通Actor只能处理完全同域萃取，mixed管理Actor可以跨具体教学域处理。
// 异域、来源缺失和脏关联统一表现为404。

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tedna/internal/models"
	"tedna/internal/utils"
)

// ListExtractions 获取Actor教育域可见的待审萃取队列。
func (h *ComponentHandler) ListExtractions(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(
			w,
			err.Error(),
		)
		return
	}

	limit, _ := strconv.Atoi(
		r.URL.Query().Get("limit"),
	)

	if limit <= 0 {
		limit = 50
	}

	result, err :=
		h.compService.ListPendingExtractionItemsForActor(
			r.Context(),
			actor,
			limit,
		)
	if err != nil {
		h.handleCompError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		result,
	)
}

// ConfirmExtraction 确认或拒绝Actor有权处理的萃取记录。
func (h *ComponentHandler) ConfirmExtraction(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	extractionID := extractMiddleSegment(
		r.URL.Path,
		"/api/v1/lesson-plans/extractions/",
		"/confirm",
	)

	if extractionID == "" {
		utils.BadRequest(
			w,
			"缺少萃取记录ID",
		)
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(
			w,
			err.Error(),
		)
		return
	}

	var request models.ConfirmExtractionRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	err = h.compService.ConfirmExtractionByIDForActor(
		r.Context(),
		actor,
		extractionID,
		request.Decision,
	)

	if err != nil {
		h.handleCompError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "操作成功",
		},
	)
}

package handlers

// lesson_plan_version_handler.go — 教案正文版本历史HTTP接口
//
// 路由：
//   GET  /api/v1/lesson-plans/plans/{id}/versions
//   GET  /api/v1/lesson-plans/plans/{id}/versions/{version_id}
//   POST /api/v1/lesson-plans/plans/{id}/versions/{version_id}/restore
//
// 三个接口都要求登录，作者权限和状态校验由service层统一负责。

import (
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/utils"
)

// ListContentVersions 获取教案正文版本列表。
func (h *LessonPlanHandler) ListContentVersions(
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

	planID := extractLPMiddleID(r.URL.Path, "/versions")
	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}

	callerID := getCurrentUserID(r)
	if callerID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	result, err := h.lpService.ListContentVersions(
		r.Context(),
		planID,
		callerID,
		limit,
		offset,
	)
	if err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(w, result)
}

// GetContentVersion 获取一个完整历史版本。
func (h *LessonPlanHandler) GetContentVersion(
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

	planID := extractLPID(r.URL.Path)
	versionID := extractLessonPlanVersionID(r.URL.Path)

	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	if versionID == "" {
		utils.BadRequest(w, "缺少教案历史版本ID")
		return
	}

	callerID := getCurrentUserID(r)
	if callerID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	result, err := h.lpService.GetContentVersion(
		r.Context(),
		planID,
		versionID,
		callerID,
	)
	if err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(w, result)
}

// RestoreContentVersion 恢复指定历史版本。
func (h *LessonPlanHandler) RestoreContentVersion(
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

	planID := extractLPID(r.URL.Path)
	versionID := extractLessonPlanVersionID(r.URL.Path)

	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	if versionID == "" {
		utils.BadRequest(w, "缺少教案历史版本ID")
		return
	}

	callerID := getCurrentUserID(r)
	if callerID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	result, err := h.lpService.RestoreContentVersion(
		r.Context(),
		planID,
		versionID,
		callerID,
	)
	if err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(w, result)
}

// extractLessonPlanVersionID 从版本子路径提取version_id。
//
// 支持：
//
//	/plans/{planID}/versions/{versionID}
//	/plans/{planID}/versions/{versionID}/restore
func extractLessonPlanVersionID(path string) string {
	const marker = "/versions/"

	index := strings.Index(path, marker)
	if index < 0 {
		return ""
	}

	rest := path[index+len(marker):]
	rest = strings.TrimSuffix(rest, "/")
	rest = strings.TrimSuffix(rest, "/restore")
	rest = strings.TrimSuffix(rest, "/")

	if rest == "" || strings.Contains(rest, "/") {
		return ""
	}

	return rest
}

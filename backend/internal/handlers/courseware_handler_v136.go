package handlers

import (
	"encoding/json"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/utils"
)

// v136: 步骤回退 + 方案预设处理器

// RollbackStatus POST /api/v1/coursewares/{id}/rollback-status
func (h *CoursewareHandler) RollbackStatus(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/rollback-status",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		id,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	var req models.RollbackStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.TargetStatus == "" {
		utils.BadRequest(w, "target_status不能为空")
		return
	}

	if err := h.cwService.RollbackStatusForActor(
		r.Context(),
		id,
		actor,
		req.TargetStatus,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "回退成功",
			"status":  req.TargetStatus,
		},
	)
}

// GetSchemePresets GET /api/v1/courseware-presets
func (h *CoursewareHandler) GetSchemePresets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	utils.Success(w, models.CoursewareSchemePresets)
}

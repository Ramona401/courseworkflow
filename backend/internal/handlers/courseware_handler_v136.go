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
func (h *CoursewareHandler) RollbackStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "not logged in")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/rollback-status")
	if id == "" {
		utils.BadRequest(w, "missing courseware ID")
		return
	}

	var req models.RollbackStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "invalid request body")
		return
	}
	if req.TargetStatus == "" {
		utils.BadRequest(w, "target_status required")
		return
	}

	if err := h.cwService.RollbackStatus(r.Context(), id, claims.UserID, req.TargetStatus); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "rollback ok", "status": req.TargetStatus})
}

// GetSchemePresets GET /api/v1/courseware-presets
func (h *CoursewareHandler) GetSchemePresets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	utils.Success(w, models.CoursewareSchemePresets)
}

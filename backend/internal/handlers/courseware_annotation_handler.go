package handlers

// courseware_annotation_handler.go — 课件页级批注 HTTP 处理器(阶段2)
//
// 挂在既有 *CoursewareHandler 上的新增端点,不碰原有方法:
//   - POST   /api/v1/coursewares/{id}/annotations              创建批注
//   - GET    /api/v1/coursewares/{id}/annotations              列出课件全部批注
//   - PUT    /api/v1/coursewares/annotations/{aid}/resolve     标记已处理/待处理
//   - DELETE /api/v1/coursewares/annotations/{aid}             删除批注
//
// 路径解析:
//   - 创建/列表复用 extractCoursewareMiddleID(从 .../{id}/annotations 提课件ID)
//   - 标记/删除用本文件 extractCWAnnotationID(从 .../annotations/{aid}[/resolve] 提批注ID)
//
// 错误映射:ErrCWAnnotationNotFound → 404;其余业务错误 → 400(含无权,延续课件模块惯例
// 用 BadRequest 携带中文文案,前端直接展示)。

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// extractCWAnnotationID 从路径提取批注ID
//   - /api/v1/coursewares/annotations/{aid}          → {aid}
//   - /api/v1/coursewares/annotations/{aid}/resolve  → {aid}
func extractCWAnnotationID(path string) string {
	const marker = "/annotations/"
	idx := strings.Index(path, marker)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimPrefix(path[idx+len(marker):], "")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return ""
	}
	// 去掉可能的尾缀 action(如 /resolve)
	if slash := strings.Index(rest, "/"); slash >= 0 {
		rest = rest[:slash]
	}
	return rest
}

// CreateCWAnnotation POST /api/v1/coursewares/{id}/annotations — 创建批注
// body: {"page_number":N,"content":"..."}
func (h *CoursewareHandler) CreateCWAnnotation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	cwID := extractCoursewareMiddleID(r.URL.Path, "/annotations")
	if cwID == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	var req models.CreateCWAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	// reviewer_name 取登录态用户名(冗余存,免列表时 JOIN)
	reviewerName := claims.Username

	a, err := h.cwService.CreateCWAnnotation(r.Context(), cwID, claims.UserID, claims.Role, reviewerName, &req)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	utils.Success(w, a)
}

// ListCWAnnotations GET /api/v1/coursewares/{id}/annotations — 列出课件全部批注
func (h *CoursewareHandler) ListCWAnnotations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	cwID := extractCoursewareMiddleID(r.URL.Path, "/annotations")
	if cwID == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	resp, err := h.cwService.ListCWAnnotations(r.Context(), cwID, claims.UserID, claims.Role)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ResolveCWAnnotation PUT /api/v1/coursewares/annotations/{aid}/resolve — 标记已处理/待处理
// body: {"status":"resolved"|"pending"}
func (h *CoursewareHandler) ResolveCWAnnotation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	aid := extractCWAnnotationID(r.URL.Path)
	if aid == "" {
		utils.BadRequest(w, "缺少批注ID")
		return
	}

	var req models.ResolveCWAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.ResolveCWAnnotation(r.Context(), aid, claims.UserID, claims.Role, req.Status); err != nil {
		if errors.Is(err, repository.ErrCWAnnotationNotFound) {
			utils.Fail(w, http.StatusNotFound, err.Error())
			return
		}
		utils.BadRequest(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已更新"})
}

// DeleteCWAnnotation DELETE /api/v1/coursewares/annotations/{aid} — 删除批注
func (h *CoursewareHandler) DeleteCWAnnotation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	aid := extractCWAnnotationID(r.URL.Path)
	if aid == "" {
		utils.BadRequest(w, "缺少批注ID")
		return
	}

	if err := h.cwService.DeleteCWAnnotation(r.Context(), aid, claims.UserID, claims.Role); err != nil {
		if errors.Is(err, repository.ErrCWAnnotationNotFound) {
			utils.Fail(w, http.StatusNotFound, err.Error())
			return
		}
		utils.BadRequest(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已删除"})
}

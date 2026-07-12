package handlers

// courseware_share_handler.go — 课件工坊·发布与共享 HTTP 处理器（阶段1）
//
// 挂在既有 *CoursewareHandler 上的新增端点，不碰 courseware_handler.go 原有方法：
//   - PublishCourseware    POST /api/v1/coursewares/{id}/publish            发布/撤回
//   - SetCodeShareScope    PUT  /api/v1/coursewares/{id}/code-share-scope   设源代码开放范围
//   - ListSharedCoursewares GET /api/v1/coursewares/shared                  共享课件库列表
//   - ForkCourseware       POST /api/v1/coursewares/{id}/fork               复制到我的
//
// 路径解析复用 courseware_handler.go 已有的 extractCoursewareMiddleID。

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/utils"
)

// PublishCourseware POST /api/v1/coursewares/{id}/publish — 发布 / 撤回
// body: {"target":"published_personal"|"published_shared"|"private"}
func (h *CoursewareHandler) PublishCourseware(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/publish")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	var req models.PublishCoursewareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.SetPublishState(r.Context(), id, claims.UserID, req.Target); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "操作成功"})
}

// SetCodeShareScope PUT /api/v1/coursewares/{id}/code-share-scope — 设源代码开放范围
// body: {"code_share_scope":"none"|"group"|"school"|"region"|"public"}
func (h *CoursewareHandler) SetCodeShareScope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/code-share-scope")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	var req models.SetCodeShareScopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.SetCodeShareScope(r.Context(), id, claims.UserID, req.CodeShareScope); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "代码开放范围已更新"})
}

// ListSharedCoursewares GET /api/v1/coursewares/shared — 共享课件库列表
// query: subject（可选）、limit、offset
func (h *CoursewareHandler) ListSharedCoursewares(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	subject := r.URL.Query().Get("subject")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	resp, err := h.cwService.ListSharedCoursewares(r.Context(), claims.UserID, claims.Role, subject, limit, offset)
	if err != nil {
		utils.InternalError(w, "查询共享课件失败: "+err.Error())
		return
	}
	utils.Success(w, resp)
}

// ForkCourseware POST /api/v1/coursewares/{id}/fork — 复制共享课件到我的
func (h *CoursewareHandler) ForkCourseware(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	srcID := extractCoursewareMiddleID(r.URL.Path, "/fork")
	if srcID == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	cw, err := h.cwService.ForkCourseware(r.Context(), srcID, claims.UserID, claims.Role)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"id":      cw.ID,
		"title":   cw.Title,
		"message": "已复制到我的课件",
	})
}

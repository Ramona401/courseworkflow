package handlers

// course_outline_handler.go — 课程大纲处理器（大单元备课能力·批次一 + 教材版本增强）
//
// 提供接口（/api/v1/course-outlines）：
//   GET    /api/v1/course-outlines             — 列出可见大纲（全员登录可查）
//   POST   /api/v1/course-outlines             — 创建（组长/校管/admin）
//   GET    /api/v1/course-outlines/publishers  — 查某学科+年级可选教材版本（备课首屏选择器用）★新增
//   GET    /api/v1/course-outlines/{id}        — 查单条详情（含正文，全员可查）
//   PUT    /api/v1/course-outlines/{id}        — 更新（归属者）
//   DELETE /api/v1/course-outlines/{id}        — 软删除（归属者）
//
// 路由说明：标准库 ServeMux 用 "/api/v1/course-outlines/"（带尾斜杠）通配全部子路径到
//   HandleItem，无法按前缀分优先级。因此 /publishers 这个"非ID"子路径在 HandleItem 内
//   以 id=="publishers" 特判分流，不新增路由（最贴合 ServeMux 的工作方式）。
//
// 写权限的归属校验在 service 层（canManageScope）。本 handler 只做路径解析 +
// claims 提取 + 错误映射，风格对齐 curriculum_handler.go。

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CourseOutlineHandler 课程大纲处理器
type CourseOutlineHandler struct {
	svc *services.CourseOutlineService
}

// NewCourseOutlineHandler 创建处理器
func NewCourseOutlineHandler(svc *services.CourseOutlineService) *CourseOutlineHandler {
	return &CourseOutlineHandler{svc: svc}
}

// extractCourseOutlineID 从 /api/v1/course-outlines/{id} 提取 ID
func extractCourseOutlineID(path string) string {
	const prefix = "/api/v1/course-outlines/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	return id
}

// HandleCollection 处理 /api/v1/course-outlines（无尾 ID）：GET 列表 / POST 创建
func (h *CourseOutlineHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.list(w, r, claims.Role, claims.UserID)
	case http.MethodPost:
		h.create(w, r, claims.Role, claims.UserID)
	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET/POST请求")
	}
}

// HandleItem 处理 /api/v1/course-outlines/{id}：GET 详情 / PUT 更新 / DELETE 删除
//
// 特判：id=="publishers" 时分流到「查可用教材版本」（GET，备课首屏选择器用），
// 这是个非ID的功能子路径，借 ServeMux 的尾斜杠通配落到这里再分流。
func (h *CourseOutlineHandler) HandleItem(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCourseOutlineID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少大纲ID")
		return
	}

	// 非ID功能子路径特判：/api/v1/course-outlines/publishers
	if id == "publishers" {
		if r.Method != http.MethodGet {
			utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
			return
		}
		h.listPublishers(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.detail(w, r, id)
	case http.MethodPut:
		h.update(w, r, claims.Role, claims.UserID, id)
	case http.MethodDelete:
		h.delete(w, r, claims.Role, claims.UserID, id)
	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET/PUT/DELETE请求")
	}
}

func (h *CourseOutlineHandler) list(w http.ResponseWriter, r *http.Request, role, userID string) {
	items, err := h.svc.ListOutlines(r.Context(), role, userID)
	if err != nil {
		utils.InternalError(w, "查询课程大纲失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"outlines": items,
		"total":    len(items),
	})
}

// listPublishers 查某学科+年级下可选的教材版本列表（备课首屏教材版本选择器用）
//
// 入参：query 的 subject、grade（均必填）。
// 返回：publishers 字符串数组（空串元素代表"通用/不限版本"，前端负责显示成中文）；
//       该学科年级没有任何相交大纲时返回空数组（前端据此不显示版本选择、不关联大纲）。
func (h *CourseOutlineHandler) listPublishers(w http.ResponseWriter, r *http.Request) {
	subject := strings.TrimSpace(r.URL.Query().Get("subject"))
	grade := strings.TrimSpace(r.URL.Query().Get("grade"))
	if subject == "" || grade == "" {
		utils.BadRequest(w, "缺少学科或年级参数")
		return
	}
	publishers, err := h.svc.ListAvailablePublishers(r.Context(), subject, grade)
	if err != nil {
		utils.InternalError(w, "查询可用教材版本失败: "+err.Error())
		return
	}
	if publishers == nil {
		publishers = []string{}
	}
	utils.Success(w, map[string]interface{}{
		"publishers": publishers,
		"total":      len(publishers),
	})
}

func (h *CourseOutlineHandler) create(w http.ResponseWriter, r *http.Request, role, userID string) {
	var req models.CreateCourseOutlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	o, err := h.svc.CreateOutline(r.Context(), role, userID, &req)
	if err != nil {
		h.mapError(w, err)
		return
	}
	utils.Success(w, o)
}

func (h *CourseOutlineHandler) detail(w http.ResponseWriter, r *http.Request, id string) {
	o, err := repository.GetCourseOutlineByID(r.Context(), id)
	if err != nil {
		h.mapError(w, err)
		return
	}
	utils.Success(w, o)
}

func (h *CourseOutlineHandler) update(w http.ResponseWriter, r *http.Request, role, userID, id string) {
	var req models.UpdateCourseOutlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	if err := h.svc.UpdateOutline(r.Context(), role, userID, id, &req); err != nil {
		h.mapError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{"message": "更新成功"})
}

func (h *CourseOutlineHandler) delete(w http.ResponseWriter, r *http.Request, role, userID, id string) {
	if err := h.svc.DeleteOutline(r.Context(), role, userID, id); err != nil {
		h.mapError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{"message": "删除成功"})
}

// mapError 业务错误 → HTTP 状态码
func (h *CourseOutlineHandler) mapError(w http.ResponseWriter, err error) {
	switch err {
	case services.ErrOutlineFieldRequired, services.ErrOutlineScopeInvalid:
		utils.BadRequest(w, err.Error())
	case services.ErrOutlineNoPermission:
		utils.Fail(w, http.StatusForbidden, err.Error())
	case repository.ErrCourseOutlineNotFound:
		utils.Fail(w, http.StatusNotFound, err.Error())
	default:
		utils.InternalError(w, err.Error())
	}
}

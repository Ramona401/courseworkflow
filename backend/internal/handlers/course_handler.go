package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== CourseHandler ====================

// CourseHandler 课程管理API处理器
type CourseHandler struct {
	courseService *services.CourseService
}

// NewCourseHandler 创建课程处理器实例
func NewCourseHandler(courseService *services.CourseService) *CourseHandler {
	return &CourseHandler{
		courseService: courseService,
	}
}

// ==================== API接口 ====================

// ListCourses 获取课程列表
// GET /api/v1/courses
// 权限：登录即可（按角色过滤可见课程）
func (h *CourseHandler) ListCourses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 获取当前用户信息
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 获取课程列表（按角色过滤）
	resp, err := h.courseService.ListCourses(claims.UserID, claims.Role)
	if err != nil {
		utils.InternalError(w, "获取课程列表失败: "+err.Error())
		return
	}

	utils.Success(w, resp)
}

// CreateCourse 注册新课程
// POST /api/v1/courses
// 权限：仅admin
func (h *CourseHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 解析请求体
	var req models.CreateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	// 注册课程
	course, err := h.courseService.CreateCourse(&req)
	if err != nil {
		handleCourseError(w, err)
		return
	}

	utils.Success(w, course)
}

// FetchIndex 从OSS拉取课程索引
// POST /api/v1/courses/{code}/fetch-index
// 权限：仅admin
func (h *CourseHandler) FetchIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 提取课程编号
	code := extractCourseCode(r.URL.Path, "/fetch-index")
	if code == "" {
		utils.BadRequest(w, "缺少课程编号")
		return
	}

	// 拉取索引
	idx, err := h.courseService.FetchIndex(code)
	if err != nil {
		handleCourseError(w, err)
		return
	}

	// 返回索引摘要（不返回完整内容，避免大数据传输）
	utils.Success(w, map[string]interface{}{
		"course_code":  code,
		"page_count":   idx.PageCount,
		"total_length": idx.TotalLength,
		"index_hash":   idx.IndexHash,
		"fetched_at":   idx.FetchedAt,
		"message":      "索引拉取成功",
	})
}

// GetIndexFull 获取完整索引内容
// GET /api/v1/courses/{code}/index
// 权限：仅admin（原文可见）
func (h *CourseHandler) GetIndexFull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 提取课程编号
	code := extractCourseCode(r.URL.Path, "/index")
	if code == "" {
		utils.BadRequest(w, "缺少课程编号")
		return
	}

	// 获取完整索引
	resp, err := h.courseService.GetIndexFull(code)
	if err != nil {
		handleCourseError(w, err)
		return
	}

	utils.Success(w, resp)
}

// GetIndexSummary 获取索引摘要
// GET /api/v1/courses/{code}/index-summary
// 权限：登录即可（只能看摘要）
func (h *CourseHandler) GetIndexSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 提取课程编号
	code := extractCourseCode(r.URL.Path, "/index-summary")
	if code == "" {
		utils.BadRequest(w, "缺少课程编号")
		return
	}

	// 获取索引摘要
	resp, err := h.courseService.GetIndexSummary(code)
	if err != nil {
		handleCourseError(w, err)
		return
	}

	utils.Success(w, resp)
}

// GetOSSCatalog 获取OSS目录（含注册状态）
// GET /api/v1/courses/oss-catalog
// 权限：仅admin
func (h *CourseHandler) GetOSSCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 获取OSS目录
	resp, err := h.courseService.GetOSSCatalog()
	if err != nil {
		utils.InternalError(w, "获取OSS目录失败: "+err.Error())
		return
	}

	utils.Success(w, resp)
}

// ==================== 路径提取工具 ====================

// extractCourseCode 从URL路径中提取课程编号
// 路径格式: /api/v1/courses/{code}/{suffix}
// 例如: /api/v1/courses/G1-01/fetch-index → G1-01
func extractCourseCode(path string, suffix string) string {
	// 移除后缀
	if suffix != "" {
		idx := strings.LastIndex(path, suffix)
		if idx > 0 {
			path = path[:idx]
		}
	}

	// 移除尾部斜杠
	path = strings.TrimSuffix(path, "/")

	// 提取最后一段作为课程编号
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash < 0 {
		return ""
	}

	code := path[lastSlash+1:]
	if code == "" || code == "courses" {
		return ""
	}

	return code
}

// ==================== 错误处理 ====================

// handleCourseError 统一课程相关错误处理
func handleCourseError(w http.ResponseWriter, err error) {
	switch err {
	case services.ErrCourseCodeRequired:
		utils.BadRequest(w, err.Error())
	case services.ErrModuleIDRequired:
		utils.BadRequest(w, err.Error())
	case services.ErrCourseCodeExists:
		utils.Fail(w, http.StatusConflict, err.Error())
	case services.ErrCourseNotFound:
		utils.Fail(w, http.StatusNotFound, err.Error())
	case services.ErrModuleIDAlreadyBound:
		utils.Fail(w, http.StatusConflict, err.Error())
	case services.ErrIndexNotAvailable:
		utils.Fail(w, http.StatusNotFound, err.Error())
	case services.ErrIndexContentEmpty:
		utils.BadRequest(w, err.Error())
	default:
		utils.InternalError(w, err.Error())
	}
}

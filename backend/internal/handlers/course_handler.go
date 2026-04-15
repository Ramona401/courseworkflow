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

// CourseHandler 课程管理API处理器
type CourseHandler struct {
	courseService *services.CourseService
}

func NewCourseHandler(courseService *services.CourseService) *CourseHandler {
	return &CourseHandler{courseService: courseService}
}

func (h *CourseHandler) ListCourses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	resp, err := h.courseService.ListCourses(claims.UserID, claims.Role)
	if err != nil {
		utils.InternalError(w, "获取课程列表失败: "+err.Error())
		return
	}
	utils.Success(w, resp)
}

func (h *CourseHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	var req models.CreateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	course, err := h.courseService.CreateCourse(&req)
	if err != nil {
		handleCourseError(w, err)
		return
	}
	utils.Success(w, course)
}

func (h *CourseHandler) RegisterAndFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	var req models.CreateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	result, err := h.courseService.RegisterAndFetch(&req)
	if err != nil {
		handleCourseError(w, err)
		return
	}
	utils.Success(w, result)
}

func (h *CourseHandler) BatchRegisterAndFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	result, err := h.courseService.BatchRegisterAndFetch()
	if err != nil {
		utils.InternalError(w, "批量注册失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}

func (h *CourseHandler) BatchFetchIndexes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	result, err := h.courseService.BatchFetchIndexes()
	if err != nil {
		utils.InternalError(w, "批量拉取失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}

func (h *CourseHandler) FetchIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	code := extractCourseCode(r.URL.Path, "/fetch-index")
	if code == "" {
		utils.BadRequest(w, utils.MsgMissingCourseCode)
		return
	}
	idx, err := h.courseService.FetchIndex(code)
	if err != nil {
		handleCourseError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{
		"course_code": code, "page_count": idx.PageCount,
		"total_length": idx.TotalLength, "index_hash": idx.IndexHash,
		"fetched_at": idx.FetchedAt, "message": "索引拉取成功",
	})
}

func (h *CourseHandler) GetIndexFull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	code := extractCourseCode(r.URL.Path, "/index")
	if code == "" {
		utils.BadRequest(w, utils.MsgMissingCourseCode)
		return
	}
	resp, err := h.courseService.GetIndexFull(code)
	if err != nil {
		handleCourseError(w, err)
		return
	}
	utils.Success(w, resp)
}

func (h *CourseHandler) GetIndexSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	code := extractCourseCode(r.URL.Path, "/index-summary")
	if code == "" {
		utils.BadRequest(w, utils.MsgMissingCourseCode)
		return
	}
	resp, err := h.courseService.GetIndexSummary(code)
	if err != nil {
		handleCourseError(w, err)
		return
	}
	utils.Success(w, resp)
}

func (h *CourseHandler) GetOSSCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	resp, err := h.courseService.GetOSSCatalog()
	if err != nil {
		utils.InternalError(w, "获取OSS目录失败: "+err.Error())
		return
	}
	utils.Success(w, resp)
}

func extractCourseCode(path string, suffix string) string {
	if suffix != "" {
		idx := strings.LastIndex(path, suffix)
		if idx > 0 {
			path = path[:idx]
		}
	}
	path = strings.TrimSuffix(path, "/")
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

func handleCourseError(w http.ResponseWriter, err error) {
	switch err {
	case services.ErrCourseCodeRequired, services.ErrModuleIDRequired, services.ErrIndexContentEmpty:
		utils.BadRequest(w, err.Error())
	case services.ErrCourseCodeExists, services.ErrModuleIDAlreadyBound:
		utils.Fail(w, http.StatusConflict, err.Error())
	case services.ErrCourseNotFound, services.ErrIndexNotAvailable:
		utils.Fail(w, http.StatusNotFound, err.Error())
	default:
		utils.InternalError(w, err.Error())
	}
}

package handlers

// course_outline_handler.go — 课程大纲HTTP主处理器
//
// 精确候选、响应组装和错误映射拆至
// course_outline_handler_support.go。

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CourseOutlineHandler 课程大纲处理器。
type CourseOutlineHandler struct {
	svc *services.CourseOutlineService
}

// NewCourseOutlineHandler 创建处理器。
func NewCourseOutlineHandler(
	svc *services.CourseOutlineService,
) *CourseOutlineHandler {
	return &CourseOutlineHandler{
		svc: svc,
	}
}

func extractCourseOutlineID(
	path string,
) string {
	const prefix = "/api/v1/course-outlines/"

	if !strings.HasPrefix(
		path,
		prefix,
	) {
		return ""
	}

	id := strings.TrimPrefix(
		path,
		prefix,
	)
	return strings.TrimSuffix(
		id,
		"/",
	)
}

// HandleCollection 处理列表和创建。
func (h *CourseOutlineHandler) HandleCollection(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.list(
			w,
			r,
			claims.UserID,
		)
	case http.MethodPost:
		h.create(
			w,
			r,
			claims.UserID,
		)
	default:
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET/POST请求",
		)
	}
}

// HandleItem 处理候选、出版社、详情、更新和删除。
func (h *CourseOutlineHandler) HandleItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCourseOutlineID(
		r.URL.Path,
	)
	if id == "" {
		utils.BadRequest(w, "缺少大纲ID")
		return
	}

	switch id {
	case "candidates":
		if r.Method != http.MethodGet {
			utils.Fail(
				w,
				http.StatusMethodNotAllowed,
				"仅支持GET请求",
			)
			return
		}
		h.listExactCandidates(
			w,
			r,
			claims.UserID,
		)
		return

	case "publishers":
		if r.Method != http.MethodGet {
			utils.Fail(
				w,
				http.StatusMethodNotAllowed,
				"仅支持GET请求",
			)
			return
		}
		h.listPublishers(
			w,
			r,
			claims.UserID,
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.detail(
			w,
			r,
			claims.UserID,
			id,
		)
	case http.MethodPut:
		h.update(
			w,
			r,
			claims.UserID,
			id,
		)
	case http.MethodDelete:
		h.delete(
			w,
			r,
			claims.UserID,
			id,
		)
	default:
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET/PUT/DELETE请求",
		)
	}
}

func (h *CourseOutlineHandler) list(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
) {
	items, domain, err :=
		h.svc.ListOutlines(
			r.Context(),
			userID,
		)
	if err != nil {
		h.mapError(w, err)
		return
	}

	responseItems := make(
		[]map[string]interface{},
		0,
		len(items),
	)
	for _, item := range items {
		responseItems = append(
			responseItems,
			courseOutlineListItemResponse(
				item,
				domain,
			),
		)
	}

	utils.Success(
		w,
		map[string]interface{}{
			"outlines": responseItems,
			"total":    len(responseItems),
		},
	)
}

func (h *CourseOutlineHandler) listPublishers(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
) {
	subject := strings.TrimSpace(
		r.URL.Query().Get("subject"),
	)
	grade := strings.TrimSpace(
		r.URL.Query().Get("grade"),
	)
	if subject == "" || grade == "" {
		utils.BadRequest(
			w,
			"缺少学科或年级参数",
		)
		return
	}

	publishers, err :=
		h.svc.ListAvailablePublishers(
			r.Context(),
			userID,
			subject,
			grade,
		)
	if err != nil {
		h.mapError(w, err)
		return
	}
	if publishers == nil {
		publishers = []string{}
	}

	utils.Success(
		w,
		map[string]interface{}{
			"publishers": publishers,
			"total":      len(publishers),
		},
	)
}

func (h *CourseOutlineHandler) create(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
) {
	var req models.CreateCourseOutlineRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求体解析失败",
		)
		return
	}

	outline, domain, err :=
		h.svc.CreateOutline(
			r.Context(),
			userID,
			&req,
		)
	if err != nil {
		h.mapError(w, err)
		return
	}

	utils.Success(
		w,
		courseOutlineDetailResponse(
			outline,
			domain,
		),
	)
}

func (h *CourseOutlineHandler) detail(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
	id string,
) {
	outline, domain, err :=
		h.svc.GetOutline(
			r.Context(),
			userID,
			id,
		)
	if err != nil {
		h.mapError(w, err)
		return
	}

	utils.Success(
		w,
		courseOutlineDetailResponse(
			outline,
			domain,
		),
	)
}

func (h *CourseOutlineHandler) update(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
	id string,
) {
	var req models.UpdateCourseOutlineRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求体解析失败",
		)
		return
	}

	if err := h.svc.UpdateOutline(
		r.Context(),
		userID,
		id,
		&req,
	); err != nil {
		h.mapError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"message": "更新成功",
		},
	)
}

func (h *CourseOutlineHandler) delete(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
	id string,
) {
	if err := h.svc.DeleteOutline(
		r.Context(),
		userID,
		id,
	); err != nil {
		h.mapError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"message": "删除成功",
		},
	)
}

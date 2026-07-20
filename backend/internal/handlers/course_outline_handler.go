package handlers

// course_outline_handler.go — 课程大纲HTTP处理器
//
// 上下文16教育域收口：
//   1. Handler只从JWT取得userID，Service会实时读取数据库角色；
//   2. 出版社列表将userID传给Service，非K12及异常域返回成功空数组；
//   3. 详情不再直接调用Repository，统一经过Service的同域和可见范围校验；
//   4. K12响应保留publisher；
//   5. vocational/adult响应完全省略publisher字段，避免泄露K12出版社语义；
//   6. 创建响应同样按当前教育域裁剪；
//   7. 教育域不可用、跨域和无管理权限统一返回403；
//   8. 数据库或基础设施解析失败返回5xx。

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
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

// extractCourseOutlineID 从单条路径提取ID。
func extractCourseOutlineID(
	path string,
) string {
	const prefix =
		"/api/v1/course-outlines/"

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
	id = strings.TrimSuffix(
		id,
		"/",
	)

	return id
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
		utils.Unauthorized(
			w,
			"未登录",
		)
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

// HandleItem 处理出版社列表、详情、更新和删除。
func (h *CourseOutlineHandler) HandleItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			"未登录",
		)
		return
	}

	id := extractCourseOutlineID(
		r.URL.Path,
	)
	if id == "" {
		utils.BadRequest(
			w,
			"缺少大纲ID",
		)
		return
	}

	if id == "publishers" {
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

	responseItems :=
		make(
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

// listPublishers 查询可用出版社。
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

// courseOutlineListItemResponse
// K12保留publisher，非K12完全省略该字段。
func courseOutlineListItemResponse(
	item *models.CourseOutlineListItem,
	educationDomain string,
) map[string]interface{} {
	if item == nil {
		return map[string]interface{}{}
	}

	response := map[string]interface{}{
		"id":              item.ID,
		"scope":           item.Scope,
		"scope_target_id": item.ScopeTargetID,
		"scope_name":      item.ScopeName,
		"subject":         item.Subject,
		"grade":           item.Grade,
		"volume":          item.Volume,
		"title":           item.Title,
		"creator_name":    item.CreatorName,
		"updated_at":      item.UpdatedAt,
	}

	if educationDomain ==
		models.EducationDomainK12 {
		response["publisher"] =
			item.Publisher
	}

	return response
}

// courseOutlineDetailResponse
// K12保留publisher，非K12完全省略该字段。
func courseOutlineDetailResponse(
	outline *models.CourseOutline,
	educationDomain string,
) map[string]interface{} {
	if outline == nil {
		return map[string]interface{}{}
	}

	response := map[string]interface{}{
		"id":               outline.ID,
		"scope":            outline.Scope,
		"scope_target_id":  outline.ScopeTargetID,
		"subject":          outline.Subject,
		"grade":            outline.Grade,
		"volume":           outline.Volume,
		"title":            outline.Title,
		"content":          outline.Content,
		"source_file_path": outline.SourceFilePath,
		"source_type":      outline.SourceType,
		"created_by":       outline.CreatedBy,
		"status":           outline.Status,
		"created_at":       outline.CreatedAt,
		"updated_at":       outline.UpdatedAt,
	}

	if educationDomain ==
		models.EducationDomainK12 {
		response["publisher"] =
			outline.Publisher
	}

	return response
}

// mapError 统一映射课程大纲业务错误。
func (h *CourseOutlineHandler) mapError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrOutlineFieldRequired,
	),
		errors.Is(
			err,
			services.ErrOutlineScopeInvalid,
		),
		errors.Is(
			err,
			services.ErrOutlinePublisherNotAllowed,
		),
		errors.Is(
			err,
			services.ErrOutlinePublisherUnavailable,
		):
		utils.BadRequest(
			w,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrOutlineNoPermission,
	),
		errors.Is(
			err,
			services.
				ErrOutlineEducationDomainRequired,
		),
		errors.Is(
			err,
			services.
				ErrOutlineEducationDomainConflict,
		),
		errors.Is(
			err,
			services.
				ErrOutlineEducationDomainMismatch,
		):
		utils.Fail(
			w,
			http.StatusForbidden,
			err.Error(),
		)

	case errors.Is(
		err,
		repository.ErrCourseOutlineNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(
		err,
		services.
			ErrOutlineEducationDomainResolveFailed,
	):
		utils.InternalError(
			w,
			services.
				ErrOutlineEducationDomainResolveFailed.
				Error(),
		)

	default:
		utils.InternalError(
			w,
			"课程大纲操作失败，请稍后重试",
		)
	}
}

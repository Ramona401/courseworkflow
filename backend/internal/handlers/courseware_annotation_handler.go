package handlers

// courseware_annotation_handler.go — 课件页级批注HTTP处理器
//
//   - POST   /api/v1/coursewares/{id}/annotations
//   - GET    /api/v1/coursewares/{id}/annotations
//   - PUT    /api/v1/coursewares/annotations/{aid}/resolve
//   - DELETE /api/v1/coursewares/annotations/{aid}

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/utils"
)

func extractCWAnnotationID(
	path string,
) string {
	const marker = "/annotations/"

	index := strings.Index(
		path,
		marker,
	)
	if index < 0 {
		return ""
	}

	rest := strings.Trim(
		path[index+
			len(marker):],
		"/",
	)
	if rest == "" {
		return ""
	}

	if slash := strings.Index(
		rest,
		"/",
	); slash >= 0 {
		rest = rest[:slash]
	}

	return strings.TrimSpace(rest)
}

// CreateCWAnnotation 创建课件页级批注。
func (h *CoursewareHandler) CreateCWAnnotation(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

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

	coursewareID :=
		extractCoursewareMiddleID(
			r.URL.Path,
			"/annotations",
		)
	if coursewareID == "" {
		utils.BadRequest(
			w,
			"缺少课件ID",
		)
		return
	}

	// 批注正文解析前先确认作者或合法集体备课参与者身份。
	scopedActor, err :=
		authorizeCoursewareAnnotationRefine(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareAnnotationError(
			w,
			err,
		)
		return
	}

	var request models.CreateCWAnnotationRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}

	annotation, err :=
		h.cwService.CreateCWAnnotation(
			r.Context(),
			coursewareID,
			scopedActor,
			claims.Username,
			&request,
		)
	if err != nil {
		writeCoursewareAnnotationError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		annotation,
	)
}

// ListCWAnnotations 列出当前用户有权查看的课件批注。
func (h *CoursewareHandler) ListCWAnnotations(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

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

	coursewareID :=
		extractCoursewareMiddleID(
			r.URL.Path,
			"/annotations",
		)
	if coursewareID == "" {
		utils.BadRequest(
			w,
			"缺少课件ID",
		)
		return
	}

	actor, err :=
		authorizeCoursewareAnnotationView(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareAnnotationError(
			w,
			err,
		)
		return
	}

	response, err :=
		h.cwService.ListCWAnnotations(
			r.Context(),
			coursewareID,
			actor,
		)
	if err != nil {
		writeCoursewareAnnotationError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		response,
	)
}

// ResolveCWAnnotation 标记批注已处理或重新待处理。
func (h *CoursewareHandler) ResolveCWAnnotation(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持PUT请求",
		)
		return
	}

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

	annotationID :=
		extractCWAnnotationID(
			r.URL.Path,
		)
	if annotationID == "" {
		utils.BadRequest(
			w,
			"缺少批注ID",
		)
		return
	}

	// 状态正文解析前先加载正式批注并验证管理身份。
	scopedActor, err :=
		authorizeCoursewareAnnotationManage(
			r.Context(),
			annotationID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareAnnotationError(
			w,
			err,
		)
		return
	}

	var request models.ResolveCWAnnotationRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}

	if err :=
		h.cwService.ResolveCWAnnotation(
			r.Context(),
			annotationID,
			scopedActor,
			request.Status,
		); err != nil {
		writeCoursewareAnnotationError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "已更新",
		},
	)
}

// DeleteCWAnnotation 删除批注。
func (h *CoursewareHandler) DeleteCWAnnotation(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持DELETE请求",
		)
		return
	}

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

	annotationID :=
		extractCWAnnotationID(
			r.URL.Path,
		)
	if annotationID == "" {
		utils.BadRequest(
			w,
			"缺少批注ID",
		)
		return
	}

	scopedActor, err :=
		authorizeCoursewareAnnotationManage(
			r.Context(),
			annotationID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareAnnotationError(
			w,
			err,
		)
		return
	}

	if err :=
		h.cwService.DeleteCWAnnotation(
			r.Context(),
			annotationID,
			scopedActor,
		); err != nil {
		writeCoursewareAnnotationError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "已删除",
		},
	)
}

package handlers

import (
	"encoding/json"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CWSeedHandler 课件种子与模板管理处理器。
type CWSeedHandler struct {
	seedService *services.CoursewareSeedService
}

// NewCWSeedHandler 创建种子数据处理器。
func NewCWSeedHandler(
	seedService *services.CoursewareSeedService,
) *CWSeedHandler {
	return &CWSeedHandler{
		seedService: seedService,
	}
}

// SeedAll POST /api/v1/admin/courseware-seed。
//
// 仅admin可执行；组件部分固定使用K12域安全种子入口。
func (h *CWSeedHandler) SeedAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok ||
		claims == nil ||
		claims.Role != models.RoleAdmin {
		utils.Forbidden(
			w,
			"仅管理员可执行种子数据填充",
		)
		return
	}

	var request struct {
		Force bool `json:"force"`
	}

	if r.Body != nil {
		// 允许空body；格式错误时保持force=false。
		_ = json.NewDecoder(
			r.Body,
		).Decode(&request)
	}

	result, err :=
		h.seedService.
			SeedAllForEducationDomain(
				r.Context(),
				request.Force,
			)
	if err != nil {
		utils.InternalError(
			w,
			"种子数据填充失败: "+
				err.Error(),
		)
		return
	}

	utils.Success(w, result)
}

// CreateTemplate POST /api/v1/admin/courseware-templates。
func (h *CWSeedHandler) CreateTemplate(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok ||
		claims == nil ||
		claims.Role != models.RoleAdmin {
		utils.Forbidden(
			w,
			"仅管理员可创建模板",
		)
		return
	}

	var request models.CreateCWTemplateRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	if request.Name == "" ||
		request.StyleCategory == "" {
		utils.BadRequest(
			w,
			"名称和风格类别为必填项",
		)
		return
	}

	template := &models.CoursewareTemplate{
		Name:            request.Name,
		Description:     request.Description,
		StyleCategory:   request.StyleCategory,
		PreviewImageURL: request.PreviewImageURL,
		ColorScheme:     request.ColorScheme,
		CSSVariables:    request.CSSVariables,
		SamplePages:     request.SamplePages,
		IsActive:        true,
		SortOrder:       0,
	}

	if err := repository.CreateCWTemplate(
		r.Context(),
		template,
	); err != nil {
		utils.InternalError(
			w,
			"创建模板失败: "+err.Error(),
		)
		return
	}

	utils.Success(w, template)
}

// UpdateTemplate PUT /api/v1/admin/courseware-templates/{id}。
func (h *CWSeedHandler) UpdateTemplate(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPutOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok ||
		claims == nil ||
		claims.Role != models.RoleAdmin {
		utils.Forbidden(
			w,
			"仅管理员可更新模板",
		)
		return
	}

	templateID :=
		extractAdminCWTemplateID(
			r.URL.Path,
		)
	if templateID == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingTemplateID,
		)
		return
	}

	var request models.UpdateCWTemplateRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	if err := repository.UpdateCWTemplate(
		r.Context(),
		templateID,
		&request,
	); err != nil {
		utils.InternalError(
			w,
			err.Error(),
		)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "模板更新成功",
		},
	)
}

// DeleteTemplate DELETE /api/v1/admin/courseware-templates/{id}。
func (h *CWSeedHandler) DeleteTemplate(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodDeleteOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok ||
		claims == nil ||
		claims.Role != models.RoleAdmin {
		utils.Forbidden(
			w,
			"仅管理员可删除模板",
		)
		return
	}

	templateID :=
		extractAdminCWTemplateID(
			r.URL.Path,
		)
	if templateID == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingTemplateID,
		)
		return
	}

	if err := repository.DeleteCWTemplate(
		r.Context(),
		templateID,
	); err != nil {
		utils.InternalError(
			w,
			err.Error(),
		)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "模板删除成功",
		},
	)
}

// extractAdminCWTemplateID 从管理员模板路径提取ID。
func extractAdminCWTemplateID(
	path string,
) string {
	const prefix =
		"/api/v1/admin/courseware-templates/"

	if len(path) <= len(prefix) {
		return ""
	}

	rest := path[len(prefix):]

	for len(rest) > 0 &&
		rest[len(rest)-1] == '/' {
		rest = rest[:len(rest)-1]
	}

	return rest
}

package handlers

// cw_component_handler.go — 课件组件库HTTP处理器。
//
// 教育域规则：
//   - 所有入口统一从JWT建立可信AssistantActorContext；
//   - 普通用户列表、详情和匹配只允许同域或common；
//   - 普通用户提交education_domain不能扩大读取范围；
//   - mixed管理Actor可以跨域查看并按目标域筛选；
//   - mixed匹配必须明确选择具体教学域；
//   - 创建、更新和删除继续保持admin专属；
//   - 创建时mixed admin必须明确资源域；
//   - 更新协议不含education_domain，不能原地迁移资源；
//   - 异域直接ID与不存在统一返回404。

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CWComponentHandler 课件组件库处理器。
type CWComponentHandler struct{}

// NewCWComponentHandler 创建课件组件库处理器。
func NewCWComponentHandler() *CWComponentHandler {
	return &CWComponentHandler{}
}

// resolveCWComponentActor 从JWT Claims建立可信组件Actor。
func resolveCWComponentActor(
	r *http.Request,
) (*services.AssistantActorContext, error) {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)

	if !ok ||
		claims == nil ||
		strings.TrimSpace(
			claims.UserID,
		) == "" {
		return nil,
			errors.New(
				utils.MsgNotLoggedIn,
			)
	}

	return services.BuildActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	), nil
}

// ListComponents GET /api/v1/courseware-components。
func (h *CWComponentHandler) ListComponents(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	actor, err :=
		resolveCWComponentActor(r)
	if err != nil {
		utils.Unauthorized(
			w,
			err.Error(),
		)
		return
	}

	query := r.URL.Query()

	limit, _ := strconv.Atoi(
		query.Get("limit"),
	)
	offset, _ := strconv.Atoi(
		query.Get("offset"),
	)

	activeOnly := true

	result, err :=
		services.ListCWComponentsForActor(
			r.Context(),
			actor,
			query.Get(
				"education_domain",
			),
			query.Get(
				"component_type",
			),
			query.Get(
				"subject_scope",
			),
			query.Get(
				"grade_scope",
			),
			&activeOnly,
			limit,
			offset,
		)
	if err != nil {
		handleCWComponentError(
			w,
			err,
		)
		return
	}

	utils.Success(w, result)
}

// CreateComponent POST /api/v1/courseware-components。
func (h *CWComponentHandler) CreateComponent(
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

	actor, err :=
		resolveCWComponentActor(r)
	if err != nil {
		utils.Unauthorized(
			w,
			err.Error(),
		)
		return
	}

	var request models.CreateCWComponentDomainRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	resource, err :=
		services.CreateCWComponentForActor(
			r.Context(),
			actor,
			&request,
		)
	if err != nil {
		handleCWComponentError(
			w,
			err,
		)
		return
	}

	utils.Success(w, resource)
}

// GetComponent GET /api/v1/courseware-components/{id}。
func (h *CWComponentHandler) GetComponent(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	componentID := extractCWCompID(
		r.URL.Path,
	)
	if componentID == "" {
		utils.BadRequest(
			w,
			"缺少组件ID",
		)
		return
	}

	actor, err :=
		resolveCWComponentActor(r)
	if err != nil {
		utils.Unauthorized(
			w,
			err.Error(),
		)
		return
	}

	resource, err :=
		services.GetCWComponentForActor(
			r.Context(),
			actor,
			componentID,
		)
	if err != nil {
		handleCWComponentError(
			w,
			err,
		)
		return
	}

	utils.Success(w, resource)
}

// UpdateComponent PUT /api/v1/courseware-components/{id}。
func (h *CWComponentHandler) UpdateComponent(
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

	componentID := extractCWCompID(
		r.URL.Path,
	)
	if componentID == "" {
		utils.BadRequest(
			w,
			"缺少组件ID",
		)
		return
	}

	actor, err :=
		resolveCWComponentActor(r)
	if err != nil {
		utils.Unauthorized(
			w,
			err.Error(),
		)
		return
	}

	var request models.UpdateCWComponentRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	err = services.UpdateCWComponentForActor(
		r.Context(),
		actor,
		componentID,
		&request,
	)
	if err != nil {
		handleCWComponentError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "更新成功",
		},
	)
}

// DeleteComponent DELETE /api/v1/courseware-components/{id}。
func (h *CWComponentHandler) DeleteComponent(
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

	componentID := extractCWCompID(
		r.URL.Path,
	)
	if componentID == "" {
		utils.BadRequest(
			w,
			"缺少组件ID",
		)
		return
	}

	actor, err :=
		resolveCWComponentActor(r)
	if err != nil {
		utils.Unauthorized(
			w,
			err.Error(),
		)
		return
	}

	err = services.DeleteCWComponentForActor(
		r.Context(),
		actor,
		componentID,
	)
	if err != nil {
		handleCWComponentError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "删除成功",
		},
	)
}

// MatchComponents POST /api/v1/courseware-components/match。
func (h *CWComponentHandler) MatchComponents(
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

	actor, err :=
		resolveCWComponentActor(r)
	if err != nil {
		utils.Unauthorized(
			w,
			err.Error(),
		)
		return
	}

	var request models.MatchCWComponentsDomainRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	matched, err :=
		services.MatchCWComponentsForActor(
			r.Context(),
			actor,
			&request,
		)
	if err != nil {
		handleCWComponentError(
			w,
			err,
		)
		return
	}

	utils.Success(w, matched)
}

// CompressIndex 保留既有AOCI占位端点。
//
// 上下文19不实现AOCI压缩能力，也不把该占位响应视为真实索引已生成。
func (h *CWComponentHandler) CompressIndex(
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

	utils.Success(
		w,
		map[string]string{
			"message": "索引压缩功能将在Phase 2实现",
		},
	)
}

// handleCWComponentError 统一映射课件组件业务错误。
func handleCWComponentError(
	w http.ResponseWriter,
	err error,
) {
	badRequestErrors := []error{
		services.ErrCWComponentRequestRequired,
		services.ErrCWComponentNameRequired,
		services.ErrCWComponentTypeRequired,
		services.ErrCWComponentTypeInvalid,
		services.ErrCWComponentCodeRequired,
		services.ErrCWComponentReviewInvalid,
		services.ErrCWComponentEducationDomainRequired,
		services.ErrCWComponentEducationDomainInvalid,
		services.ErrCWComponentSelectionInvalid,
	}

	for _, target := range badRequestErrors {
		if errors.Is(err, target) {
			utils.BadRequest(
				w,
				err.Error(),
			)
			return
		}
	}

	if errors.Is(
		err,
		services.ErrCWComponentEducationDomainForbidden,
	) {
		utils.Forbidden(
			w,
			err.Error(),
		)
		return
	}

	if errors.Is(
		err,
		services.ErrCWComponentNotFound,
	) {
		utils.Fail(
			w,
			http.StatusNotFound,
			"课件组件不存在",
		)
		return
	}

	log.Printf(
		"课件组件操作失败: %v",
		err,
	)

	utils.InternalError(
		w,
		"课件组件操作失败，请稍后重试",
	)
}

// extractCWCompID 从组件详情或子操作路径中提取组件ID。
func extractCWCompID(
	path string,
) string {
	const prefix = "/api/v1/courseware-components/"

	if !strings.HasPrefix(
		path,
		prefix,
	) {
		return ""
	}

	rest := strings.TrimPrefix(
		path,
		prefix,
	)
	rest = strings.TrimRight(
		rest,
		"/",
	)

	if separatorIndex := strings.Index(
		rest,
		"/",
	); separatorIndex > 0 {
		return rest[:separatorIndex]
	}

	return rest
}

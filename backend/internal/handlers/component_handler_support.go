package handlers

// component_handler_support.go — 组件Handler公共辅助。
//
// 集中处理可信Actor构建、Service错误到HTTP状态码映射和组件ID解析，
// 避免主Handler重复样板代码并保持单文件低于600行。

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// resolveActor 从JWT Claims构建可信组件Actor。
func (h *ComponentHandler) resolveActor(
	r *http.Request,
) (*services.AssistantActorContext, error) {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)

	if !ok {
		return nil,
			errors.New(utils.MsgNotLoggedIn)
	}

	if strings.TrimSpace(
		claims.UserID,
	) == "" {
		return nil,
			errors.New(utils.MsgNotLoggedIn)
	}

	return services.BuildActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	), nil
}

// handleCompError 将组件Service错误映射为HTTP响应。
func (h *ComponentHandler) handleCompError(
	w http.ResponseWriter,
	err error,
) {
	if isComponentBadRequest(err) {
		utils.BadRequest(w, err.Error())
		return
	}

	if errors.Is(
		err,
		services.ErrComponentEducationDomainForbidden,
	) {
		utils.Forbidden(w, err.Error())
		return
	}

	notFound :=
		errors.Is(
			err,
			services.ErrComponentNotFound,
		) ||
			errors.Is(
				err,
				services.ErrExtractionNotFound,
			)

	if notFound {
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)
		return
	}

	log.Printf(
		"组件库操作失败: %v",
		err,
	)

	utils.InternalError(
		w,
		"操作失败，请稍后重试",
	)
}

// isComponentBadRequest 判断是否属于客户端参数错误。
func isComponentBadRequest(
	err error,
) bool {
	targets := []error{
		services.ErrComponentLibTypeRequired,
		services.ErrComponentLibTypeInvalid,
		services.ErrComponentLabelRequired,
		services.ErrComponentReviewInvalid,
		services.ErrComponentInvalidInjectionMode,
		services.ErrComponentInvalidScope,
		services.ErrComponentNotReviewable,
		services.ErrComponentMatchRequestRequired,
		services.ErrComponentMatchSubjectRequired,
		services.ErrExtractionDecisionInvalid,
		services.ErrComponentEducationDomainRequired,
		services.ErrComponentEducationDomainInvalid,
		services.ErrComponentSelectionInvalid,
	}

	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}

	return false
}

// extractComponentID 从组件详情路径中提取组件ID。
func extractComponentID(
	path string,
) string {
	const prefix = "/api/v1/lesson-plans/components/"

	if !strings.HasPrefix(
		path,
		prefix,
	) {
		return ""
	}

	componentID := strings.TrimPrefix(
		path,
		prefix,
	)

	componentID = strings.TrimSuffix(
		componentID,
		"/",
	)

	separatorIndex := strings.Index(
		componentID,
		"/",
	)

	if separatorIndex > 0 {
		componentID =
			componentID[:separatorIndex]
	}

	return componentID
}

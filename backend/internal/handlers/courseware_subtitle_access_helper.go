package handlers

// courseware_subtitle_access_helper.go — 字幕Handler可信Actor授权与错误映射

import (
	"context"
	"errors"
	"net/http"

	"tedna/internal/services"
	"tedna/internal/utils"
)

// authorizeCoursewareSubtitleView 执行字幕读取预检。
func authorizeCoursewareSubtitleView(
	ctx context.Context,
	coursewareID string,
	userID string,
	role string,
) (*services.CoursewareActorContext, error) {
	actor :=
		services.BuildCoursewareActorFromClaims(
			ctx,
			userID,
			role,
		)

	if _, err :=
		(&services.CoursewareService{}).
			LoadCoursewareForView(
				ctx,
				coursewareID,
				actor,
			); err != nil {
		return nil, err
	}

	return actor, nil
}

// authorizeCoursewareSubtitleRefine 执行字幕文本编辑预检。
func authorizeCoursewareSubtitleRefine(
	ctx context.Context,
	coursewareID string,
	userID string,
	role string,
) (*services.CoursewareActorContext, error) {
	actor :=
		services.BuildCoursewareActorFromClaims(
			ctx,
			userID,
			role,
		)

	_, scopedActor, err :=
		(&services.CoursewareService{}).
			LoadCoursewareForRefine(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	return scopedActor, nil
}

// authorizeCoursewareSubtitleOwnerControl 执行高成本字幕能力作者控制预检。
func authorizeCoursewareSubtitleOwnerControl(
	ctx context.Context,
	coursewareID string,
	userID string,
	role string,
) (*services.CoursewareActorContext, error) {
	actor :=
		services.BuildCoursewareActorFromClaims(
			ctx,
			userID,
			role,
		)

	_, scopedActor, err :=
		(&services.CoursewareService{}).
			LoadCoursewareForOwnerControlMutation(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	return scopedActor, nil
}

// writeCoursewareSubtitleError 统一映射字幕访问、编辑与Scope错误。
func writeCoursewareSubtitleError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrCoursewareAccessNotFound,
	),
		errors.Is(
			err,
			services.ErrCoursewareSubtitleNotFound,
		),
		errors.Is(
			err,
			services.ErrCoursewareSubtitleScopeTargetMismatch,
		):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewareActorRequired,
	),
		errors.Is(
			err,
			services.ErrCoursewareOwnerRuntimeDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareViewDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEditDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainMismatch,
		):
		utils.Fail(
			w,
			http.StatusForbidden,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewareSubtitleInputInvalid,
	),
		errors.Is(
			err,
			services.ErrCoursewareSubtitleScopeInvalid,
		),
		errors.Is(
			err,
			services.ErrCoursewareSubtitleScopeTargetRequired,
		):
		utils.BadRequest(
			w,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewareSubtitleMutationConflict,
	),
		errors.Is(
			err,
			services.ErrCoursewarePageMutationConflict,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewareEducationDomainInvalid,
	),
		errors.Is(
			err,
			services.ErrCoursewareRuntimeDomainRequired,
		):
		utils.InternalError(
			w,
			err.Error(),
		)

	default:
		utils.InternalError(
			w,
			err.Error(),
		)
	}
}

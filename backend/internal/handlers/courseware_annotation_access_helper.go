package handlers

// courseware_annotation_access_helper.go — 批注Handler可信Actor授权与错误映射

import (
	"context"
	"errors"
	"net/http"

	"tedna/internal/services"
	"tedna/internal/utils"
)

func authorizeCoursewareAnnotationView(
	ctx context.Context,
	coursewareID string,
	userID string,
	role string,
) (
	*services.CoursewareActorContext,
	error,
) {
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

func authorizeCoursewareAnnotationRefine(
	ctx context.Context,
	coursewareID string,
	userID string,
	role string,
) (
	*services.CoursewareActorContext,
	error,
) {
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

func authorizeCoursewareAnnotationManage(
	ctx context.Context,
	annotationID string,
	userID string,
	role string,
) (
	*services.CoursewareActorContext,
	error,
) {
	actor :=
		services.BuildCoursewareActorFromClaims(
			ctx,
			userID,
			role,
		)

	_, _, scopedActor, err :=
		(&services.CoursewareService{}).
			LoadCoursewareAnnotationForManage(
				ctx,
				annotationID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	return scopedActor, nil
}

func writeCoursewareAnnotationError(
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
			services.ErrCoursewareAnnotationNotFound,
		),
		errors.Is(
			err,
			services.ErrCoursewareAnnotationPageNotFound,
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
			services.ErrCoursewareViewDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEditDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareAnnotationManageDenied,
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
		services.ErrCoursewareAnnotationInputInvalid,
	):
		utils.BadRequest(
			w,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewareAnnotationMutationConflict,
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

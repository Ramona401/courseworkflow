package handlers

// courseware_refine_handler_helper.go
//
// 课件教研微调Handler可信Actor辅助和错误映射。

import (
	"context"
	"errors"
	"net/http"

	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// authorizeCoursewareRefineForHandler 在解析大正文、截图或微调指令前，
// 构造可信Actor并执行教研微调预检。
func (h *CoursewareGenHandler) authorizeCoursewareRefineForHandler(
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
		h.cwService.LoadCoursewareForRefine(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	return scopedActor, nil
}

// writeCoursewareRefineError 统一映射教研微调、页面冲突和整改项应用错误。
func writeCoursewareRefineError(
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
			services.ErrCoursewarePageNotFound,
		),
		errors.Is(
			err,
			repository.ErrCoursewareReviewItemNotFound,
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
			services.ErrCWAIReviewActorRequired,
		),
		errors.Is(
			err,
			services.ErrCoursewareEditDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainMismatch,
		),
		errors.Is(
			err,
			services.ErrCWAIReviewNoPermission,
		),
		errors.Is(
			err,
			services.ErrCWReviewItemNotDelivered,
		):
		utils.Fail(
			w,
			http.StatusForbidden,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewarePageMutationConflict,
	),
		errors.Is(
			err,
			repository.ErrCoursewareReviewItemConflict,
		),
		errors.Is(
			err,
			services.ErrCWReviewItemStale,
		),
		errors.Is(
			err,
			services.ErrCWReviewItemOrphaned,
		),
		errors.Is(
			err,
			services.ErrCWReviewItemNotActionable,
		),
		errors.Is(
			err,
			services.ErrCWReviewItemApplicationPageMismatch,
		),
		errors.Is(
			err,
			services.ErrCWReviewItemApplicationInstructionMissing,
		),
		errors.Is(
			err,
			services.ErrCWReviewItemApplicationInstructionMismatch,
		),
		errors.Is(
			err,
			services.ErrCWReviewItemApplicationVersionMismatch,
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

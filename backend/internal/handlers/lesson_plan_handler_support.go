package handlers

// lesson_plan_handler_support.go — 教案Handler公共支持模块
//
// 本文件集中承载：
//   - Service错误到HTTP状态码的统一映射；
//   - 教案ID、动作路径ID和模板ID解析。
//
// 所有教案Handler入口共用这里的错误语义，
// 避免不同文件重复维护。

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"tedna/internal/services"
	"tedna/internal/utils"
)

// handleLPError 统一映射教案业务错误。
func (h *LessonPlanHandler) handleLPError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrLPTitleRequired,
	),
		errors.Is(
			err,
			services.ErrLPSubjectRequired,
		),
		errors.Is(
			err,
			services.ErrLPGradeRequired,
		),
		errors.Is(
			err,
			services.ErrLPTopicRequired,
		),
		errors.Is(
			err,
			services.ErrLPGroupRequired,
		),
		errors.Is(
			err,
			services.ErrLPContentEmpty,
		),
		errors.Is(
			err,
			services.ErrTemplateNameRequired,
		),
		errors.Is(
			err,
			services.ErrTemplateLevelInvalid,
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
		services.ErrLPNotAuthor,
	),
		errors.Is(
			err,
			services.ErrLPNotPublisher,
		),
		errors.Is(
			err,
			services.
				ErrLPCreationEducationDomainRequired,
		),
		errors.Is(
			err,
			services.
				ErrLPCreationEducationDomainConflict,
		),
		errors.Is(
			err,
			services.ErrLPForkNotAllowed,
		),
		errors.Is(
			err,
			services.
				ErrLPForkEducationDomainMismatch,
		),
		errors.Is(
			err,
			services.ErrLPCannotEdit,
		),
		errors.Is(
			err,
			services.ErrLPCannotSubmit,
		),
		errors.Is(
			err,
			services.ErrLPCannotDevelop,
		),
		errors.Is(
			err,
			services.ErrLPAlreadyDeveloping,
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
		services.ErrLPNotFound,
	),
		errors.Is(
			err,
			services.ErrLPVersionNotFound,
		),
		errors.Is(
			err,
			services.ErrTemplateNotFound,
		):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(
		err,
		services.
			ErrLPCreationEducationDomainResolveFailed,
	),
		errors.Is(
			err,
			services.
				ErrOutlineEducationDomainResolveFailed,
		):
		log.Printf(
			"教案教育域解析失败: %v",
			err,
		)
		utils.InternalError(
			w,
			"教案教育域解析失败，请稍后重试",
		)

	default:
		log.Printf(
			"教案操作失败: %v",
			err,
		)
		utils.InternalError(
			w,
			"操作失败，请稍后重试",
		)
	}
}

// extractLPID 从普通教案详情路径提取教案ID。
func extractLPID(path string) string {
	prefix :=
		"/api/v1/lesson-plans/plans/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	id := strings.TrimPrefix(
		path,
		prefix,
	)
	id = strings.TrimSuffix(id, "/")

	if index := strings.Index(
		id,
		"/",
	); index > 0 {
		id = id[:index]
	}

	return id
}

// extractLPMiddleID 从教案动作路径提取教案ID。
func extractLPMiddleID(
	path string,
	suffix string,
) string {
	return extractMiddleSegment(
		path,
		"/api/v1/lesson-plans/plans/",
		suffix,
	)
}

// extractTemplateID 从提示词模板路径提取模板ID。
func extractTemplateID(path string) string {
	prefix :=
		"/api/v1/lesson-plans/templates/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	id := strings.TrimPrefix(
		path,
		prefix,
	)
	id = strings.TrimSuffix(id, "/")

	if index := strings.Index(
		id,
		"/",
	); index > 0 {
		id = id[:index]
	}

	return id
}

package services

// courseware_comic_reference_sources.go — 参考资源可信来源解析
//
// 本文件负责服务端可信正式来源：
//   - 教材单元按当前课件学科、年级重新读取；
//   - 已有课件重新经过作者运行通道读取；
//   - 课程大纲重新经过同教育域和可见范围服务读取；
//   - 正式来源忽略浏览器提交的标题和正文，防止伪造快照。
//
// 上传文档、上传图片及其它文本来源拆分到
// courseware_comic_reference_uploads.go；通用辅助函数拆分到
// courseware_comic_reference_helpers.go。

import (
	"context"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	coursewareComicReferenceContentMaxRunes = 120000

	coursewareComicReferenceSummaryMaxRunes = 30000
)

// buildTrustedCoursewareComicReference
// 根据资源类型构造服务端可信参考资源实体。
func (s *CoursewareComicProjectService) buildTrustedCoursewareComicReference(
	ctx context.Context,
	courseware *models.Courseware,
	project *models.CoursewareComicProject,
	scopedActor *CoursewareActorContext,
	originalActor *CoursewareActorContext,
	request *models.CreateCoursewareComicReferenceRequest,
) (*models.CoursewareComicReferenceResource, error) {
	if s == nil ||
		courseware == nil ||
		project == nil ||
		scopedActor == nil ||
		originalActor == nil ||
		request == nil ||
		!models.IsValidCWComicReferenceResourceType(
			request.ResourceType,
		) {
		return nil,
			ErrCoursewareComicReferenceInvalidRequest
	}

	item :=
		&models.CoursewareComicReferenceResource{
			ProjectID:    project.ID,
			CoursewareID: courseware.ID,
			CreatedBy:    scopedActor.UserID,
			ResourceType: request.ResourceType,
			SortOrder:    request.SortOrder,
		}

	var err error

	switch request.ResourceType {
	case models.CWComicReferenceTextbookUnit:
		err =
			buildCoursewareComicTextbookReference(
				ctx,
				courseware,
				request,
				item,
			)

	case models.CWComicReferenceCourseware:
		err =
			s.buildCoursewareComicCoursewareReference(
				ctx,
				courseware,
				originalActor,
				request,
				item,
			)

	case models.CWComicReferenceCourseOutline:
		err =
			buildCoursewareComicOutlineReference(
				ctx,
				courseware,
				scopedActor.UserID,
				request,
				item,
			)

	case models.CWComicReferenceUploadedDocument:
		err =
			buildCoursewareComicDocumentReference(
				request,
				item,
			)

	case models.CWComicReferenceUploadedImage:
		err =
			buildCoursewareComicImageReference(
				ctx,
				courseware,
				request,
				item,
			)

	case models.CWComicReferenceOtherText:
		err =
			buildCoursewareComicOtherTextReference(
				request,
				item,
			)

	default:
		err =
			ErrCoursewareComicReferenceInvalidRequest
	}

	if err != nil {
		return nil, err
	}

	item.Title =
		coursewareComicReferenceTruncateRunes(
			item.Title,
			500,
		)

	item.FileName =
		coursewareComicReferenceTruncateRunes(
			item.FileName,
			255,
		)

	item.MimeType =
		strings.ToLower(
			strings.TrimSpace(
				item.MimeType,
			),
		)

	item.ContentText =
		coursewareComicReferenceTruncateRunes(
			item.ContentText,
			coursewareComicReferenceContentMaxRunes,
		)

	item.SummaryText =
		coursewareComicReferenceTruncateRunes(
			item.SummaryText,
			coursewareComicReferenceSummaryMaxRunes,
		)

	item.OriginalLength =
		utf8.RuneCountInString(
			item.ContentText,
		)

	item.SummaryLength =
		utf8.RuneCountInString(
			item.SummaryText,
		)

	if item.Title == "" ||
		item.SortOrder < 0 {
		return nil,
			ErrCoursewareComicReferenceInvalidRequest
	}

	return item, nil
}

func buildCoursewareComicTextbookReference(
	ctx context.Context,
	courseware *models.Courseware,
	request *models.CreateCoursewareComicReferenceRequest,
	item *models.CoursewareComicReferenceResource,
) error {
	if !isCoursewareComicReferenceUUID(
		request.SourceID,
	) {
		return ErrCoursewareComicReferenceInvalidRequest
	}

	gradeNum :=
		parseCoursewareComicGradeNum(
			courseware.Grade,
		)

	if gradeNum <= 0 {
		return ErrCoursewareComicProjectGradeInvalid
	}

	units, err :=
		repository.ListTextbookUnits(
			ctx,
			models.EducationDomainK12,
			strings.TrimSpace(
				courseware.Subject,
			),
			"",
			gradeNum,
			"",
		)
	if err != nil {
		return err
	}

	var selected *models.TextbookUnit

	for _, unit := range units {
		if unit == nil ||
			strings.TrimSpace(
				unit.ID,
			) !=
				request.SourceID {
			continue
		}

		selected = unit
		break
	}

	if selected == nil ||
		strings.TrimSpace(
			selected.Subject,
		) !=
			strings.TrimSpace(
				courseware.Subject,
			) ||
		selected.GradeNum !=
			gradeNum {
		return ErrCoursewareComicReferenceSourceUnavailable
	}

	sourceID :=
		strings.TrimSpace(
			selected.ID,
		)

	item.SourceID =
		&sourceID

	item.Title =
		strings.TrimSpace(
			selected.Publisher +
				" · " +
				selected.UnitTitle,
		)

	if strings.TrimSpace(
		selected.LessonTitle,
	) != "" {
		item.Title +=
			" · " +
				strings.TrimSpace(
					selected.LessonTitle,
				)
	}

	item.ContentText =
		buildCoursewareComicTextbookReferenceContent(
			selected,
		)

	return nil
}

func (s *CoursewareComicProjectService) buildCoursewareComicCoursewareReference(
	ctx context.Context,
	currentCourseware *models.Courseware,
	actor *CoursewareActorContext,
	request *models.CreateCoursewareComicReferenceRequest,
	item *models.CoursewareComicReferenceResource,
) error {
	if !isCoursewareComicReferenceUUID(
		request.SourceID,
	) {
		return ErrCoursewareComicReferenceInvalidRequest
	}

	source, _, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				request.SourceID,
				actor,
			)
	if err != nil {
		return ErrCoursewareComicReferenceSourceUnavailable
	}

	if source == nil ||
		strings.ToLower(
			strings.TrimSpace(
				source.EducationDomain,
			),
		) !=
			models.EducationDomainK12 ||
		strings.TrimSpace(
			source.Subject,
		) !=
			strings.TrimSpace(
				currentCourseware.Subject,
			) ||
		strings.TrimSpace(
			source.Grade,
		) !=
			strings.TrimSpace(
				currentCourseware.Grade,
			) {
		return ErrCoursewareComicReferenceSourceUnavailable
	}

	sourceID :=
		strings.TrimSpace(
			source.ID,
		)

	item.SourceID =
		&sourceID

	item.Title =
		strings.TrimSpace(
			source.Title,
		)

	var builder strings.Builder

	builder.WriteString(
		"参考课件：",
	)

	builder.WriteString(
		strings.TrimSpace(
			source.Title,
		),
	)

	builder.WriteString(
		"\n学科：",
	)

	builder.WriteString(
		strings.TrimSpace(
			source.Subject,
		),
	)

	builder.WriteString(
		"\n年级：",
	)

	builder.WriteString(
		strings.TrimSpace(
			source.Grade,
		),
	)

	if strings.TrimSpace(
		source.IndexOverview,
	) != "" {
		builder.WriteString(
			"\n课件方案概览：\n",
		)

		builder.WriteString(
			strings.TrimSpace(
				source.IndexOverview,
			),
		)
	}

	item.ContentText =
		builder.String()

	return nil
}

func buildCoursewareComicOutlineReference(
	ctx context.Context,
	courseware *models.Courseware,
	userID string,
	request *models.CreateCoursewareComicReferenceRequest,
	item *models.CoursewareComicReferenceResource,
) error {
	if !isCoursewareComicReferenceUUID(
		request.SourceID,
	) {
		return ErrCoursewareComicReferenceInvalidRequest
	}

	outline, domain, err :=
		NewCourseOutlineService().
			GetOutline(
				ctx,
				userID,
				request.SourceID,
			)
	if err != nil {
		return ErrCoursewareComicReferenceSourceUnavailable
	}

	if outline == nil ||
		domain !=
			models.EducationDomainK12 ||
		strings.TrimSpace(
			outline.Subject,
		) !=
			strings.TrimSpace(
				courseware.Subject,
			) ||
		strings.TrimSpace(
			outline.Grade,
		) !=
			strings.TrimSpace(
				courseware.Grade,
			) {
		return ErrCoursewareComicReferenceSourceUnavailable
	}

	content :=
		strings.TrimSpace(
			outline.Content,
		)

	if content == "" {
		return ErrCoursewareComicReferenceSourceUnavailable
	}

	sourceID :=
		strings.TrimSpace(
			outline.ID,
		)

	item.SourceID =
		&sourceID

	item.Title =
		strings.TrimSpace(
			outline.Title,
		)

	item.ContentText =
		content

	return nil
}

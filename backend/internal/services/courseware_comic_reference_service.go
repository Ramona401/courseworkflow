package services

// courseware_comic_reference_service.go — 知识点漫画参考资源业务入口
//
// 本文件负责：
//   - 重新校验当前课件作者运行边界；
//   - 校验漫画项目真实属于当前课件和教师；
//   - 创建、列出和删除项目可选参考资源；
//   - 只向浏览器返回不含正文和摘要的安全元数据；
//   - 参考图片URL必须经过同课件图片资产校验。
//
// 具体教材、课件、课程大纲、文档和图片快照构造位于：
// courseware_comic_reference_sources.go。
//
// AI上下文容量控制位于：
// courseware_comic_reference_prompt.go。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrCoursewareComicReferenceInvalidRequest = errors.New(
		"知识点漫画参考资源请求无效",
	)

	ErrCoursewareComicReferenceSourceUnavailable = errors.New(
		"选择的参考资源不存在、不可见或与当前课件不匹配",
	)

	ErrCoursewareComicReferenceDocumentTypeUnsupported = errors.New(
		"参考文档仅支持DOCX或文字型PDF",
	)

	ErrCoursewareComicReferenceContentTooLong = errors.New(
		"参考资料正文不能超过120000个字符",
	)

	ErrCoursewareComicReferenceSummaryTooLong = errors.New(
		"参考资料摘要不能超过30000个字符",
	)

	ErrCoursewareComicReferenceImageUnavailable = errors.New(
		"参考图片不存在、尚未上传完成或不属于当前课件",
	)
)

// CreateProjectReference
// 为已有漫画项目新增一项可选参考资源。
func (s *CoursewareComicProjectService) CreateProjectReference(
	ctx context.Context,
	coursewareID string,
	projectID string,
	actor *CoursewareActorContext,
	request *models.CreateCoursewareComicReferenceRequest,
) (*models.CoursewareComicReferenceResourceView, error) {
	if s == nil ||
		request == nil {
		return nil,
			ErrCoursewareComicReferenceInvalidRequest
	}

	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	projectID =
		strings.TrimSpace(
			projectID,
		)

	if coursewareID == "" ||
		projectID == "" ||
		actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" {
		return nil,
			ErrCoursewareComicReferenceInvalidRequest
	}

	normalizeCoursewareComicReferenceRequest(
		request,
	)

	if !models.IsValidCWComicReferenceResourceType(
		request.ResourceType,
	) ||
		request.SortOrder < 0 {
		return nil,
			ErrCoursewareComicReferenceInvalidRequest
	}

	courseware, scopedActor, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareControlMutationState(
			courseware,
		); err != nil {
		return nil, err
	}

	project, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			courseware.ID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if !models.IsEditableCWComicProjectStatus(
		project.Status,
	) {
		return nil,
			repository.
				ErrCoursewareComicProjectNotEditable
	}

	item, err :=
		s.buildTrustedCoursewareComicReference(
			ctx,
			courseware,
			project,
			scopedActor,
			actor,
			request,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		repository.
			CreateCoursewareComicReferenceResource(
				ctx,
				item,
			); err != nil {
		return nil, err
	}

	return buildCoursewareComicReferenceView(
		ctx,
		item,
	), nil
}

// ListProjectReferencesForBrowser
// 返回当前项目的安全参考资源列表。
func (s *CoursewareComicProjectService) ListProjectReferencesForBrowser(
	ctx context.Context,
	coursewareID string,
	projectID string,
	actor *CoursewareActorContext,
) (*models.CoursewareComicReferenceResourceListView, error) {
	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	projectID =
		strings.TrimSpace(
			projectID,
		)

	if coursewareID == "" ||
		projectID == "" ||
		actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" {
		return nil,
			ErrCoursewareComicReferenceInvalidRequest
	}

	courseware, scopedActor, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	if _, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			courseware.ID,
			projectID,
			scopedActor.UserID,
		); err != nil {
		return nil, err
	}

	views, err :=
		listCoursewareComicReferenceViewsForUser(
			ctx,
			courseware.ID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return &models.CoursewareComicReferenceResourceListView{
		References: views,
		Total:      len(views),
	}, nil
}

// DeleteProjectReference
// 删除项目参考资源绑定，不删除独立图片资产。
func (s *CoursewareComicProjectService) DeleteProjectReference(
	ctx context.Context,
	coursewareID string,
	projectID string,
	referenceID string,
	actor *CoursewareActorContext,
) error {
	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	projectID =
		strings.TrimSpace(
			projectID,
		)

	referenceID =
		strings.TrimSpace(
			referenceID,
		)

	if coursewareID == "" ||
		projectID == "" ||
		referenceID == "" ||
		actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" ||
		!isCoursewareComicReferenceUUID(
			referenceID,
		) {
		return ErrCoursewareComicReferenceInvalidRequest
	}

	courseware, scopedActor, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return err
	}

	if err :=
		validateCoursewareControlMutationState(
			courseware,
		); err != nil {
		return err
	}

	return repository.
		DeleteCoursewareComicReferenceResourceForUser(
			ctx,
			courseware.ID,
			projectID,
			referenceID,
			scopedActor.UserID,
		)
}

// listCoursewareComicReferenceViewsForUser
// 将后端完整实体转换成不含正文和摘要的浏览器安全视图。
func listCoursewareComicReferenceViewsForUser(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
) ([]*models.CoursewareComicReferenceResourceView, error) {
	items, err :=
		repository.
			ListCoursewareComicReferenceResourcesByProjectForUser(
				ctx,
				strings.TrimSpace(
					coursewareID,
				),
				strings.TrimSpace(
					projectID,
				),
				strings.TrimSpace(
					userID,
				),
			)
	if err != nil {
		return nil, err
	}

	views := make(
		[]*models.CoursewareComicReferenceResourceView,
		0,
		len(items),
	)

	for _, item := range items {
		if item == nil {
			continue
		}

		view :=
			buildCoursewareComicReferenceView(
				ctx,
				item,
			)

		if view != nil {
			views = append(
				views,
				view,
			)
		}
	}

	return views, nil
}

func buildCoursewareComicReferenceView(
	ctx context.Context,
	item *models.CoursewareComicReferenceResource,
) *models.CoursewareComicReferenceResourceView {
	if item == nil {
		return nil
	}

	view :=
		&models.CoursewareComicReferenceResourceView{
			ID:
				item.ID,
			ProjectID:
				item.ProjectID,
			CoursewareID:
				item.CoursewareID,
			ResourceType:
				item.ResourceType,
			SourceID:
				item.SourceID,
			AssetID:
				item.AssetID,
			Title:
				item.Title,
			FileName:
				item.FileName,
			MimeType:
				item.MimeType,
			OriginalLength:
				item.OriginalLength,
			SummaryLength:
				item.SummaryLength,
			SortOrder:
				item.SortOrder,
			CreatedAt:
				item.CreatedAt,
			UpdatedAt:
				item.UpdatedAt,
		}

	if item.ResourceType ==
		models.CWComicReferenceUploadedImage {
		view.ImageURL =
			loadCoursewareComicBrowserAssetURL(
				ctx,
				item.CoursewareID,
				item.AssetID,
			)
	}

	return view
}

func normalizeCoursewareComicReferenceRequest(
	request *models.CreateCoursewareComicReferenceRequest,
) {
	if request == nil {
		return
	}

	request.ResourceType =
		strings.TrimSpace(
			request.ResourceType,
		)

	request.SourceID =
		strings.TrimSpace(
			request.SourceID,
		)

	request.AssetID =
		strings.TrimSpace(
			request.AssetID,
		)

	request.Title =
		strings.TrimSpace(
			request.Title,
		)

	request.FileName =
		strings.TrimSpace(
			request.FileName,
		)

	request.MimeType =
		strings.ToLower(
			strings.TrimSpace(
				request.MimeType,
			),
		)

	request.ContentText =
		strings.TrimSpace(
			request.ContentText,
		)

	request.SummaryText =
		strings.TrimSpace(
			request.SummaryText,
		)
}

// isCoursewareComicReferenceUUID
// 使用标准库执行轻量UUID格式校验，避免为单个参数增加第三方依赖。
func isCoursewareComicReferenceUUID(
	value string,
) bool {
	value =
		strings.TrimSpace(
			value,
		)

	if len(value) != 36 {
		return false
	}

	for index, char :=
		range value {
		switch index {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}

		default:
			isDigit :=
				char >= '0' &&
					char <= '9'

			isLowerHex :=
				char >= 'a' &&
					char <= 'f'

			isUpperHex :=
				char >= 'A' &&
					char <= 'F'

			if !isDigit &&
				!isLowerHex &&
				!isUpperHex {
				return false
			}
		}
	}

	return true
}

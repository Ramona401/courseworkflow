package repository

// courseware_comic_reference_validation.go — 知识点漫画参考资源规范化与校验
//
// 把参考资源字段清洗、长度限制、类型组合约束和可空字符串转换
// 从仓储文件中拆出，避免单文件超过600行。

import (
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
)

func normalizeCoursewareComicReferenceRecord(
	item *models.CoursewareComicReferenceResource,
) {
	if item == nil {
		return
	}

	item.ProjectID =
		strings.TrimSpace(
			item.ProjectID,
		)

	item.CoursewareID =
		strings.TrimSpace(
			item.CoursewareID,
		)

	item.CreatedBy =
		strings.TrimSpace(
			item.CreatedBy,
		)

	item.ResourceType =
		strings.TrimSpace(
			item.ResourceType,
		)

	item.Title =
		strings.TrimSpace(
			item.Title,
		)

	item.FileName =
		strings.TrimSpace(
			item.FileName,
		)

	item.MimeType =
		strings.ToLower(
			strings.TrimSpace(
				item.MimeType,
			),
		)

	item.ContentText =
		strings.TrimSpace(
			item.ContentText,
		)

	item.SummaryText =
		strings.TrimSpace(
			item.SummaryText,
		)

	if item.SourceID != nil {
		value :=
			strings.TrimSpace(
				*item.SourceID,
			)

		if value == "" {
			item.SourceID = nil
		} else {
			item.SourceID = &value
		}
	}

	if item.AssetID != nil {
		value :=
			strings.TrimSpace(
				*item.AssetID,
			)

		if value == "" {
			item.AssetID = nil
		} else {
			item.AssetID = &value
		}
	}
}

func validateCoursewareComicReferenceRecord(
	item *models.CoursewareComicReferenceResource,
) error {
	if item == nil ||
		item.ProjectID == "" ||
		item.CoursewareID == "" ||
		item.CreatedBy == "" ||
		!models.IsValidCWComicReferenceResourceType(
			item.ResourceType,
		) ||
		item.Title == "" ||
		utf8.RuneCountInString(
			item.Title,
		) > 500 ||
		utf8.RuneCountInString(
			item.FileName,
		) > 255 ||
		utf8.RuneCountInString(
			item.MimeType,
		) > 255 ||
		utf8.RuneCountInString(
			item.ContentText,
		) > 120000 ||
		utf8.RuneCountInString(
			item.SummaryText,
		) > 30000 ||
		item.OriginalLength < 0 ||
		item.SummaryLength < 0 ||
		item.SortOrder < 0 {
		return ErrCoursewareComicReferenceInvalid
	}

	if models.IsCWComicReferenceOfficialSource(
		item.ResourceType,
	) {
		if item.SourceID == nil ||
			item.AssetID != nil {
			return ErrCoursewareComicReferenceInvalid
		}
	} else if item.SourceID != nil {
		return ErrCoursewareComicReferenceInvalid
	}

	if item.ResourceType ==
		models.CWComicReferenceUploadedImage {
		if item.AssetID == nil ||
			item.FileName == "" ||
			!strings.HasPrefix(
				item.MimeType,
				"image/",
			) {
			return ErrCoursewareComicReferenceInvalid
		}
	} else {
		if item.AssetID != nil ||
			(item.ContentText == "" &&
				item.SummaryText == "") {
			return ErrCoursewareComicReferenceInvalid
		}
	}

	if item.ResourceType ==
		models.CWComicReferenceUploadedDocument &&
		(item.FileName == "" ||
			item.MimeType == "") {
		return ErrCoursewareComicReferenceInvalid
	}

	return nil
}

func coursewareComicReferenceNullableString(
	value *string,
) interface{} {
	if value == nil ||
		strings.TrimSpace(
			*value,
		) == "" {
		return nil
	}

	return strings.TrimSpace(
		*value,
	)
}

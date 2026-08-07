package services

// courseware_comic_reference_uploads.go — 漫画上传参考资源可信解析
//
// 本文件负责：
//   - DOCX/PDF仅保存浏览器提取文字，不保存原始二进制；
//   - 图片只绑定当前课件中已经上传完成的图片资产；
//   - 其它文本执行长度与标题校验。

import (
	"context"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func buildCoursewareComicDocumentReference(
	request *models.CreateCoursewareComicReferenceRequest,
	item *models.CoursewareComicReferenceResource,
) error {
	fileName :=
		strings.TrimSpace(
			request.FileName,
		)

	if fileName == "" ||
		utf8.RuneCountInString(
			fileName,
		) > 255 {
		return ErrCoursewareComicReferenceInvalidRequest
	}

	mimeType :=
		resolveCoursewareComicReferenceDocumentMimeType(
			fileName,
			request.MimeType,
		)

	if mimeType == "" {
		return ErrCoursewareComicReferenceDocumentTypeUnsupported
	}

	content :=
		strings.TrimSpace(
			request.ContentText,
		)

	summary :=
		strings.TrimSpace(
			request.SummaryText,
		)

	if content == "" &&
		summary == "" {
		return ErrCoursewareComicReferenceInvalidRequest
	}

	if utf8.RuneCountInString(
		content,
	) >
		coursewareComicReferenceContentMaxRunes {
		return ErrCoursewareComicReferenceContentTooLong
	}

	if utf8.RuneCountInString(
		summary,
	) >
		coursewareComicReferenceSummaryMaxRunes {
		return ErrCoursewareComicReferenceSummaryTooLong
	}

	title :=
		strings.TrimSpace(
			request.Title,
		)

	if title == "" {
		title = fileName
	}

	if utf8.RuneCountInString(
		title,
	) > 500 {
		return ErrCoursewareComicReferenceInvalidRequest
	}

	item.Title = title
	item.FileName = fileName
	item.MimeType = mimeType
	item.ContentText = content
	item.SummaryText = summary

	return nil
}

func buildCoursewareComicImageReference(
	ctx context.Context,
	courseware *models.Courseware,
	request *models.CreateCoursewareComicReferenceRequest,
	item *models.CoursewareComicReferenceResource,
) error {
	if !isCoursewareComicReferenceUUID(
		request.AssetID,
	) {
		return ErrCoursewareComicReferenceInvalidRequest
	}

	asset, err :=
		repository.GetCWAssetByID(
			ctx,
			request.AssetID,
		)

	if err != nil ||
		asset == nil ||
		strings.TrimSpace(
			asset.CoursewareID,
		) !=
			strings.TrimSpace(
				courseware.ID,
			) ||
		asset.AssetType !=
			models.CWAssetTypeImage ||
		(asset.Status !=
			models.CWAssetStatusUploaded &&
			asset.Status !=
				models.CWAssetStatusConfirmed) {
		return ErrCoursewareComicReferenceImageUnavailable
	}

	mimeType :=
		strings.ToLower(
			strings.TrimSpace(
				asset.MimeType,
			),
		)

	if mimeType == "" {
		mimeType =
			"image/png"
	}

	if !strings.HasPrefix(
		mimeType,
		"image/",
	) {
		return ErrCoursewareComicReferenceImageUnavailable
	}

	assetID :=
		strings.TrimSpace(
			asset.ID,
		)

	item.AssetID =
		&assetID

	item.Title =
		strings.TrimSpace(
			request.Title,
		)

	if item.Title == "" {
		item.Title =
			"参考图片"
	}

	item.FileName =
		strings.TrimSpace(
			request.FileName,
		)

	if item.FileName == "" {
		item.FileName =
			defaultCoursewareComicReferenceImageName(
				mimeType,
			)
	}

	item.MimeType =
		mimeType

	return nil
}

func buildCoursewareComicOtherTextReference(
	request *models.CreateCoursewareComicReferenceRequest,
	item *models.CoursewareComicReferenceResource,
) error {
	content :=
		strings.TrimSpace(
			request.ContentText,
		)

	summary :=
		strings.TrimSpace(
			request.SummaryText,
		)

	if content == "" &&
		summary == "" {
		return ErrCoursewareComicReferenceInvalidRequest
	}

	if utf8.RuneCountInString(
		content,
	) >
		coursewareComicReferenceContentMaxRunes {
		return ErrCoursewareComicReferenceContentTooLong
	}

	if utf8.RuneCountInString(
		summary,
	) >
		coursewareComicReferenceSummaryMaxRunes {
		return ErrCoursewareComicReferenceSummaryTooLong
	}

	item.Title =
		strings.TrimSpace(
			request.Title,
		)

	if item.Title == "" {
		item.Title =
			"其他参考资料"
	}

	if utf8.RuneCountInString(
		item.Title,
	) > 500 {
		return ErrCoursewareComicReferenceInvalidRequest
	}

	item.ContentText = content
	item.SummaryText = summary

	return nil
}

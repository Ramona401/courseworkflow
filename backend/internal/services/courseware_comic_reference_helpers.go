package services

// courseware_comic_reference_helpers.go — 漫画参考资源辅助函数
//
// 本文件集中维护教材文本拼装、文档MIME判定、默认图片名和Rune安全截断。

import (
	"strings"

	"tedna/internal/models"
)

func buildCoursewareComicTextbookReferenceContent(
	unit *models.TextbookUnit,
) string {
	if unit == nil {
		return ""
	}

	var builder strings.Builder

	builder.WriteString(
		"教材版本：",
	)

	builder.WriteString(
		strings.TrimSpace(
			unit.Publisher,
		),
	)

	builder.WriteString(
		"\n单元：",
	)

	builder.WriteString(
		strings.TrimSpace(
			unit.UnitTitle,
		),
	)

	if strings.TrimSpace(
		unit.LessonTitle,
	) != "" {
		builder.WriteString(
			"\n课题：",
		)

		builder.WriteString(
			strings.TrimSpace(
				unit.LessonTitle,
			),
		)
	}

	if strings.TrimSpace(
		unit.ContentSummary,
	) != "" {
		builder.WriteString(
			"\n内容概述：",
		)

		builder.WriteString(
			strings.TrimSpace(
				unit.ContentSummary,
			),
		)
	}

	return strings.TrimSpace(
		builder.String(),
	)
}

func resolveCoursewareComicReferenceDocumentMimeType(
	fileName string,
	mimeType string,
) string {
	fileName =
		strings.ToLower(
			strings.TrimSpace(
				fileName,
			),
		)

	mimeType =
		strings.ToLower(
			strings.TrimSpace(
				mimeType,
			),
		)

	switch {
	case strings.HasSuffix(
		fileName,
		".pdf",
	):
		switch mimeType {
		case "",
			"application/pdf",
			"application/x-pdf",
			"application/octet-stream":
			return "application/pdf"
		}

	case strings.HasSuffix(
		fileName,
		".docx",
	):
		switch mimeType {
		case "",
			"application/octet-stream",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document":
			return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		}
	}

	return ""
}

func defaultCoursewareComicReferenceImageName(
	mimeType string,
) string {
	switch strings.ToLower(
		strings.TrimSpace(
			mimeType,
		),
	) {
	case "image/jpeg":
		return "reference-image.jpg"

	case "image/webp":
		return "reference-image.webp"

	case "image/gif":
		return "reference-image.gif"

	case "image/svg+xml":
		return "reference-image.svg"

	default:
		return "reference-image.png"
	}
}

func coursewareComicReferenceTruncateRunes(
	value string,
	limit int,
) string {
	value =
		strings.TrimSpace(
			value,
		)

	if limit <= 0 {
		return ""
	}

	runes :=
		[]rune(value)

	if len(runes) <= limit {
		return value
	}

	return string(
		runes[:limit],
	)
}

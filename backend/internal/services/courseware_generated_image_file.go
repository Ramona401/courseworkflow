package services

// courseware_generated_image_file.go — 生成图片本地文件元数据。
//
// 供应商响应头只用于错误诊断。正式MIME必须来自已经写盘文件的
// 实际文件签名，避免HTML错误页伪装成image/png等图片响应。

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
)

// coursewareGeneratedImageFile 保存已经确认写盘的图片结果。
type coursewareGeneratedImageFile struct {
	URL      string
	FileSize int64
	MimeType string
}

var generatedImageMIMEExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// validateCoursewareGeneratedImageFile 校验并规范生成图片元数据。
func validateCoursewareGeneratedImageFile(
	file *coursewareGeneratedImageFile,
) error {
	if file == nil {
		return fmt.Errorf(
			"生成图片文件结果为空",
		)
	}

	file.URL =
		strings.TrimSpace(
			file.URL,
		)

	if file.URL == "" {
		return fmt.Errorf(
			"生成图片本地URL为空",
		)
	}

	if file.FileSize <= 0 {
		return fmt.Errorf(
			"生成图片文件大小无效",
		)
	}

	file.MimeType =
		normalizeGeneratedImageMIMEType(
			file.MimeType,
		)

	if file.MimeType == "" {
		return fmt.Errorf(
			"生成图片MIME无效",
		)
	}

	return nil
}

// normalizeGeneratedImageMIMEType 规范允许的图片MIME。
func normalizeGeneratedImageMIMEType(
	value string,
) string {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if value == "" {
		return ""
	}

	parsed, _, err :=
		mime.ParseMediaType(
			value,
		)

	if err == nil {
		value =
			strings.ToLower(
				strings.TrimSpace(
					parsed,
				),
			)
	} else {
		value =
			strings.TrimSpace(
				strings.SplitN(
					value,
					";",
					2,
				)[0],
			)
	}

	switch value {
	case "image/jpeg",
		"image/jpg",
		"image/pjpeg":
		return "image/jpeg"

	case "image/png":
		return "image/png"

	case "image/webp":
		return "image/webp"

	default:
		return ""
	}
}

// generatedImageExtension 返回正式MIME对应的安全扩展名。
func generatedImageExtension(
	mimeType string,
) (string, error) {
	normalized :=
		normalizeGeneratedImageMIMEType(
			mimeType,
		)

	extension, exists :=
		generatedImageMIMEExtensions[normalized]

	if !exists {
		return "",
			fmt.Errorf(
				"生成图片文件类型不受支持: %s",
				strings.TrimSpace(
					mimeType,
				),
			)
	}

	return extension, nil
}

// detectGeneratedImageMIMEType 从真实文件签名识别图片类型。
//
// 真实文件只允许JPEG、PNG或WebP。若文件签名显示为HTML、JSON、
// 文本或其它内容，即使供应商响应头声称是图片也必须拒绝。
func detectGeneratedImageMIMEType(
	filePath string,
	responseContentType string,
) (string, error) {
	file, err :=
		os.Open(
			filePath,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"打开生成图片文件失败: %w",
				err,
			)
	}
	defer file.Close()

	buffer :=
		make(
			[]byte,
			512,
		)

	readSize, readErr :=
		file.Read(
			buffer,
		)

	if readErr != nil &&
		readErr != io.EOF {
		return "",
			fmt.Errorf(
				"读取生成图片文件失败: %w",
				readErr,
			)
	}

	if readSize <= 0 {
		return "",
			fmt.Errorf(
				"生成图片文件为空",
			)
	}

	detectedRaw :=
		http.DetectContentType(
			buffer[:readSize],
		)

	detected :=
		normalizeGeneratedImageMIMEType(
			detectedRaw,
		)

	if detected != "" {
		return detected, nil
	}

	return "",
		fmt.Errorf(
			"生成图片实际文件类型不受支持: actual=%s response=%s",
			detectedRaw,
			strings.TrimSpace(
				responseContentType,
			),
		)
}

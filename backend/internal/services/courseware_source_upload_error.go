package services

import (
	"errors"
	"strings"
)

// CoursewareSourceUploadErrorKind 标记“老师可自行纠正”的来源文件上传错误。
// 这类错误必须由HTTP层映射为4xx并展示明确原因，不能冒充服务器故障或网络故障。
type CoursewareSourceUploadErrorKind string

const (
	CoursewareSourceUploadInvalidExtension CoursewareSourceUploadErrorKind = "invalid_extension"
	CoursewareSourceUploadTooLarge         CoursewareSourceUploadErrorKind = "too_large"
	CoursewareSourceUploadInvalidFormat    CoursewareSourceUploadErrorKind = "invalid_format"
	CoursewareSourceUploadNoUsableContent  CoursewareSourceUploadErrorKind = "no_usable_content"
)

// CoursewareSourceUploadError 同时保存教师可读文案与仅供服务端日志/排查的底层原因。
// Error() 只返回教师可读文案，避免zip/XML/文件系统内部细节进入浏览器。
type CoursewareSourceUploadError struct {
	Kind    CoursewareSourceUploadErrorKind
	Message string
	cause   error
}

func (e *CoursewareSourceUploadError) Error() string {
	if e == nil {
		return "上传文件无法处理"
	}

	if message := strings.TrimSpace(e.Message); message != "" {
		return message
	}

	return "上传文件无法处理"
}

func (e *CoursewareSourceUploadError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

func newCoursewareSourceUploadError(
	kind CoursewareSourceUploadErrorKind,
	message string,
	cause error,
) error {
	return &CoursewareSourceUploadError{
		Kind:    kind,
		Message: strings.TrimSpace(message),
		cause:   cause,
	}
}

// AsCoursewareSourceUploadError 供HTTP层稳定区分4xx文件错误与真正5xx内部错误。
func AsCoursewareSourceUploadError(
	err error,
) (*CoursewareSourceUploadError, bool) {
	if err == nil {
		return nil, false
	}

	var target *CoursewareSourceUploadError
	if !errors.As(err, &target) || target == nil {
		return nil, false
	}

	return target, true
}

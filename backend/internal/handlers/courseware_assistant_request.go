package handlers

// courseware_assistant_request.go
//
// 本文件集中处理教师端教学智能体HTTP请求的安全输入：
//   - 限制JSON正文总字节数；
//   - 拒绝未知字段；
//   - 拒绝空正文；
//   - 拒绝一个请求中包含多个JSON对象；
//   - 只从URL提取courseware_id、page_id和slot_id；
//   - 不从正文接受owner_id、school_id或education_domain。
//
// 路径解析使用稳定page_id，不把可变化页码作为插槽主关联。

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"tedna/internal/utils"
)

const (
	// 插槽方案包含完整互动链、分层提示、学习困难方案和上下文配置。
	//
	// 旧值1MiB与方案协议允许的结构规模不一致，导致AI生成的合法方案
	// 可能在保存阶段被HTTP入口拒绝。这里调整为4MiB：
	//   - 足以保存合理的结构化教学方案；
	//   - 仍远低于Nginx站点55MB总上限；
	//   - 后端字段长度、数组数量、引用关系和答案保护校验继续生效；
	//   - 不能被用于上传任意大文件。
	coursewareAssistantSlotRequestMaxBytes int64 = 4 * 1024 * 1024

	// 方案生成正文只包含assistant_id、教学方式和教师补充要求。
	coursewareAssistantPlanRequestMaxBytes int64 = 32 * 1024
)

// decodeCoursewareAssistantJSON 严格读取一个JSON对象。
func decodeCoursewareAssistantJSON(
	w http.ResponseWriter,
	r *http.Request,
	target interface{},
	maxBytes int64,
) bool {
	if r == nil ||
		r.Body == nil ||
		target == nil {
		utils.BadRequest(
			w,
			"请求正文不能为空",
		)
		return false
	}

	if maxBytes <= 0 {
		utils.BadRequest(
			w,
			"请求正文限制无效",
		)
		return false
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxBytes,
	)

	decoder := json.NewDecoder(
		r.Body,
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		target,
	); err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(
			err,
			&maxBytesError,
		) {
			// 只记录路径、方法、声明长度和限制值，不记录请求正文，
			// 避免教学内容、学生信息或内部方案进入服务日志。
			slog.Warn(
				"教学智能体请求正文超过限制",
				"method",
				r.Method,
				"path",
				r.URL.Path,
				"content_length",
				r.ContentLength,
				"max_bytes",
				maxBytes,
			)
		}

		writeCoursewareAssistantDecodeError(
			w,
			err,
		)
		return false
	}

	var trailing interface{}

	if err := decoder.Decode(
		&trailing,
	); err != io.EOF {
		utils.BadRequest(
			w,
			"请求正文只能包含一个JSON对象",
		)
		return false
	}

	return true
}

// writeCoursewareAssistantDecodeError 返回不泄露内部结构的解析错误。
func writeCoursewareAssistantDecodeError(
	w http.ResponseWriter,
	err error,
) {
	var maxBytesError *http.MaxBytesError

	switch {
	case errors.As(
		err,
		&maxBytesError,
	):
		utils.Fail(
			w,
			http.StatusRequestEntityTooLarge,
			"教学智能体方案数据超过保存上限，请减少过长的互动步骤、提示或学习困难方案",
		)

	case errors.Is(
		err,
		io.EOF,
	):
		utils.BadRequest(
			w,
			"请求正文不能为空",
		)

	case strings.HasPrefix(
		err.Error(),
		"json: unknown field ",
	):
		utils.BadRequest(
			w,
			"请求包含未支持的字段",
		)

	default:
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
	}
}

// normalizeCoursewareAssistantRequestPath 统一移除尾部斜杠。
func normalizeCoursewareAssistantRequestPath(
	path string,
) string {
	path = strings.TrimSpace(path)

	if path == "/" {
		return path
	}

	return strings.TrimRight(
		path,
		"/",
	)
}

// extractCoursewareAssistantSlotCollectionPath 解析：
//
// /api/v1/coursewares/{courseware_id}/assistant-slots
func extractCoursewareAssistantSlotCollectionPath(
	path string,
) string {
	const prefix = "/api/v1/coursewares/"
	const suffix = "/assistant-slots"

	normalized := normalizeCoursewareAssistantRequestPath(
		path,
	)

	if !strings.HasPrefix(
		normalized,
		prefix,
	) ||
		!strings.HasSuffix(
			normalized,
			suffix,
		) {
		return ""
	}

	coursewareID := strings.TrimSuffix(
		strings.TrimPrefix(
			normalized,
			prefix,
		),
		suffix,
	)

	if !isCoursewareAssistantPathID(
		coursewareID,
	) {
		return ""
	}

	return coursewareID
}

// extractCoursewareAssistantSlotItemPath 解析：
//
// /api/v1/coursewares/{courseware_id}/assistant-slots/{slot_id}
func extractCoursewareAssistantSlotItemPath(
	path string,
) (
	string,
	string,
) {
	const prefix = "/api/v1/coursewares/"
	const marker = "/assistant-slots/"

	normalized := normalizeCoursewareAssistantRequestPath(
		path,
	)

	if !strings.HasPrefix(
		normalized,
		prefix,
	) {
		return "", ""
	}

	rest := strings.TrimPrefix(
		normalized,
		prefix,
	)

	markerIndex := strings.Index(
		rest,
		marker,
	)
	if markerIndex <= 0 {
		return "", ""
	}

	coursewareID := rest[:markerIndex]
	slotID := rest[markerIndex+len(marker):]

	if !isCoursewareAssistantPathID(
		coursewareID,
	) ||
		!isCoursewareAssistantPathID(
			slotID,
		) {
		return "", ""
	}

	return coursewareID,
		slotID
}

// extractCoursewareAssistantPageActionPath 解析：
//
// /api/v1/coursewares/{courseware_id}/pages/{page_id}/{action}
func extractCoursewareAssistantPageActionPath(
	path string,
	actionSuffix string,
) (
	string,
	string,
) {
	const prefix = "/api/v1/coursewares/"
	const pagesMarker = "/pages/"

	normalized := normalizeCoursewareAssistantRequestPath(
		path,
	)
	actionSuffix = normalizeCoursewareAssistantActionSuffix(
		actionSuffix,
	)

	if actionSuffix == "" ||
		!strings.HasPrefix(
			normalized,
			prefix,
		) ||
		!strings.HasSuffix(
			normalized,
			actionSuffix,
		) {
		return "", ""
	}

	withoutAction := strings.TrimSuffix(
		normalized,
		actionSuffix,
	)
	rest := strings.TrimPrefix(
		withoutAction,
		prefix,
	)

	pagesIndex := strings.Index(
		rest,
		pagesMarker,
	)
	if pagesIndex <= 0 {
		return "", ""
	}

	coursewareID := rest[:pagesIndex]
	pageID := rest[pagesIndex+len(pagesMarker):]

	if !isCoursewareAssistantPathID(
		coursewareID,
	) ||
		!isCoursewareAssistantPathID(
			pageID,
		) {
		return "", ""
	}

	return coursewareID,
		pageID
}

// normalizeCoursewareAssistantActionSuffix 规范化动作后缀。
func normalizeCoursewareAssistantActionSuffix(
	value string,
) string {
	value = strings.TrimSpace(value)

	if value == "" ||
		value == "/" {
		return ""
	}

	if !strings.HasPrefix(
		value,
		"/",
	) {
		value = "/" + value
	}

	return strings.TrimRight(
		value,
		"/",
	)
}

// isCoursewareAssistantPathID 校验单段资源ID。
//
// UUID合法性继续由数据库和Service负责；本函数只拒绝空值、路径穿透和多段路径。
func isCoursewareAssistantPathID(
	value string,
) bool {
	value = strings.TrimSpace(value)

	return value != "" &&
		value != "." &&
		value != ".." &&
		!strings.Contains(
			value,
			"/",
		) &&
		!strings.Contains(
			value,
			"\\",
		)
}

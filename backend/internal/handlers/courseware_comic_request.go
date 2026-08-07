package handlers

// courseware_comic_request.go — 知识点漫画HTTP请求安全读取
//
// 规则：
//   - 限制正文总字节数；
//   - 拒绝未知字段；
//   - 拒绝空正文和多个JSON对象；
//   - 不从正文接受用户ID、角色、学校ID和教育域；
//   - 路径中的课件、项目和漫画格ID由独立路由匹配器提供。

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"tedna/internal/utils"
)

const (
	coursewareComicCreateRequestMaxBytes int64 =
		64 * 1024

	coursewareComicPlanRequestMaxBytes int64 =
		32 * 1024

	coursewareComicOverlayRequestMaxBytes int64 =
		1 * 1024 * 1024

	coursewareComicPromptRequestMaxBytes int64 =
		256 * 1024
)

// decodeCoursewareComicJSON 严格读取唯一JSON对象。
func decodeCoursewareComicJSON(
	w http.ResponseWriter,
	r *http.Request,
	target interface{},
	maxBytes int64,
) bool {
	if r == nil ||
		r.Body == nil ||
		target == nil ||
		maxBytes <= 0 {
		utils.BadRequest(
			w,
			"请求正文不能为空",
		)
		return false
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		target,
	); err != nil {
		writeCoursewareComicDecodeError(
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

func writeCoursewareComicDecodeError(
	w http.ResponseWriter,
	err error,
) {
	var maxBytesError *http.MaxBytesError

	switch {
	case errors.As(
		err,
		&maxBytesError,
	):
		utils.BadRequest(
			w,
			"请求正文超过允许大小",
		)

	case errors.Is(err, io.EOF):
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

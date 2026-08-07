package handlers

// courseware_asset_anchor_handler.go — 课件风格锚点HTTP处理器
//
// 端点：
//   - POST   /api/v1/coursewares/{id}/style-anchor
//   - DELETE /api/v1/coursewares/{id}/style-anchor
//
// POST支持两种互不混用的业务模式：
//
//  1. 快捷预设模式：
//
//     {
//       "preset_style_key": "ghibli"
//     }
//
//     asset_id不再必填。
//     后端直接使用服务器白名单及系统预生成高清图创建课程锚点，
//     不调用图片模型，也不调用多模态模型。
//
//  2. 原有图片识别模式：
//
//     {
//       "asset_id": "uuid"
//     }
//
//     不提交preset_style_key时保持原有契约：
//     校验资产 → 取得公网URL → 多模态提取IAOCI → 保存锚点。
//
// 兼容性：
//   - 旧前端若同时提交asset_id和preset_style_key仍可使用；
//   - 快捷预设模式会忽略asset_id；
//   - 自定义风格工作室继续使用自身会话确认事务。

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const coursewareStyleAnchorRequestMaxBytes int64 =
	32 << 10

// SetStyleAnchor POST /api/v1/coursewares/{id}/style-anchor。
func (h *CoursewareAssetHandler) SetStyleAnchor(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)

	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			"未登录",
		)
		return
	}

	coursewareID :=
		extractAnchorCoursewareID(
			r.URL.Path,
		)

	if coursewareID == "" {
		utils.BadRequest(
			w,
			"缺少课件ID",
		)
		return
	}

	// 必须先授权，再读取请求正文。
	actor, allowed :=
		requireCoursewareAssetOwnerActor(
			w,
			r,
			coursewareID,
			claims.UserID,
			claims.Role,
		)

	if !allowed {
		return
	}

	var request struct {
		AssetID string `json:"asset_id"`

		PresetStyleKey string `json:"preset_style_key"`
	}

	if !decodeCoursewareStyleAnchorRequest(
		w,
		r,
		&request,
	) {
		return
	}

	assetID :=
		strings.TrimSpace(
			request.AssetID,
		)

	presetStyleKey :=
		strings.ToLower(
			strings.TrimSpace(
				request.PresetStyleKey,
			),
		)

	if presetStyleKey != "" &&
		!services.IsCoursewarePresetStyleKey(
			presetStyleKey,
		) {
		utils.BadRequest(
			w,
			"不支持的快捷预设画风",
		)
		return
	}

	var (
		result *services.SetStyleAnchorResult
		err    error
	)

	if presetStyleKey != "" {
		// 快捷预设：
		// 直接使用系统预生成高清图和服务器白名单，
		// 不要求asset_id，也不调用图片或多模态AI。
		result, err =
			h.assetService.SetPresetStyleAnchor(
				r.Context(),
				coursewareID,
				assetID,
				presetStyleKey,
				actor,
			)
	} else {
		// 手动图片识别模式仍然必须提交asset_id。
		if assetID == "" {
			utils.BadRequest(
				w,
				"asset_id不能为空",
			)
			return
		}

		result, err =
			h.assetService.SetStyleAnchor(
				r.Context(),
				coursewareID,
				assetID,
				actor,
			)
	}

	if err != nil {
		handleCoursewareAssetServiceError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		result,
	)
}

// ClearStyleAnchor DELETE /api/v1/coursewares/{id}/style-anchor。
func (h *CoursewareAssetHandler) ClearStyleAnchor(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持DELETE请求",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)

	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			"未登录",
		)
		return
	}

	coursewareID :=
		extractAnchorCoursewareID(
			r.URL.Path,
		)

	if coursewareID == "" {
		utils.BadRequest(
			w,
			"缺少课件ID",
		)
		return
	}

	actor, allowed :=
		requireCoursewareAssetOwnerActor(
			w,
			r,
			coursewareID,
			claims.UserID,
			claims.Role,
		)

	if !allowed {
		return
	}

	if err :=
		h.assetService.ClearStyleAnchor(
			r.Context(),
			coursewareID,
			actor,
		); err != nil {
		handleCoursewareAssetServiceError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message":
				"风格锚点已清除",
		},
	)
}

// decodeCoursewareStyleAnchorRequest 严格读取设置锚点请求。
func decodeCoursewareStyleAnchorRequest(
	w http.ResponseWriter,
	r *http.Request,
	target interface{},
) bool {
	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			coursewareStyleAnchorRequestMaxBytes,
		)

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err :=
		decoder.Decode(
			target,
		); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return false
	}

	var extra interface{}

	if err :=
		decoder.Decode(
			&extra,
		); err != io.EOF {
		utils.BadRequest(
			w,
			"请求正文只能包含一个JSON对象",
		)
		return false
	}

	return true
}

// extractAnchorCoursewareID 从风格锚点路径提取课件ID。
func extractAnchorCoursewareID(
	path string,
) string {
	const suffix =
		"/style-anchor"

	if !strings.HasSuffix(
		path,
		suffix,
	) &&
		!strings.HasSuffix(
			path,
			suffix+"/",
		) {
		return ""
	}

	trimmed :=
		strings.TrimSuffix(
			strings.TrimSuffix(
				path,
				"/",
			),
			suffix,
		)

	const prefix =
		"/api/v1/coursewares/"

	if !strings.HasPrefix(
		trimmed,
		prefix,
	) {
		return ""
	}

	coursewareID :=
		trimmed[
			len(prefix):
		]

	if coursewareID == "" ||
		strings.Contains(
			coursewareID,
			"/",
		) {
		return ""
	}

	return coursewareID
}

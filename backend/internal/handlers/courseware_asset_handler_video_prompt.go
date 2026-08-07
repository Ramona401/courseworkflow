package handlers

// courseware_asset_handler_video_prompt.go — 视频生成、状态和媒体提示词接口
//
// 视频生成安全边界：
//   - 浏览器只提交业务operation_id，不能指定供应商、模型或计费节点；
//   - operation_id必须是安全UUID；
//   - 同一次提交重试继续复用原operation_id；
//   - 后端固定使用courseware_video_generate计费节点；
//   - 普通响应不返回内部计费键、价格或成本数据。

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// GenerateVideo 提交异步AI视频生成任务。
func (h *CoursewareAssetHandler) GenerateVideo(
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
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/generate-video",
		)
	if coursewareID == "" ||
		pageNumber <= 0 {
		utils.BadRequest(
			w,
			"路径参数错误",
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

	var request struct {
		Prompt             string `json:"prompt"`
		RefImageURL        string `json:"ref_image_url"`
		SourceFrameAssetID string `json:"source_frame_asset_id"`
		OperationID        string `json:"operation_id"`
	}

	if err :=
		json.NewDecoder(
			r.Body,
		).Decode(
			&request,
		); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}

	request.Prompt =
		strings.TrimSpace(
			request.Prompt,
		)
	request.OperationID =
		strings.TrimSpace(
			request.OperationID,
		)

	if request.Prompt == "" {
		utils.BadRequest(
			w,
			"视频描述提示词不能为空",
		)
		return
	}

	if _, err :=
		uuid.Parse(
			request.OperationID,
		); err != nil {
		utils.BadRequest(
			w,
			"视频任务标识无效，请刷新页面后重试",
		)
		return
	}

	response, err :=
		h.assetService.GenerateVideo(
			r.Context(),
			&services.GenerateVideoServiceRequest{
				CoursewareID: coursewareID,
				PageNumber:   pageNumber,
				Prompt:       request.Prompt,
				RefImageURL: strings.TrimSpace(
					request.RefImageURL,
				),
				SourceFrameAssetID: strings.TrimSpace(
					request.SourceFrameAssetID,
				),
				OperationID: request.OperationID,
				Actor:       actor,
			},
		)
	if err != nil {
		handleCoursewareVideoServiceError(
			w,
			err,
		)
		return
	}

	utils.Success(w, response)
}

// QueryVideoStatus 查询异步视频生成任务状态。
func (h *CoursewareAssetHandler) QueryVideoStatus(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID :=
		extractCWAssetCoursewareID(
			r.URL.Path,
		)
	assetID :=
		extractCWVideoStatusAssetID(
			r.URL.Path,
		)

	if coursewareID == "" ||
		assetID == "" {
		utils.BadRequest(
			w,
			"课件ID或资产ID无效",
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

	response, err :=
		h.assetService.QueryVideoStatus(
			r.Context(),
			coursewareID,
			assetID,
			actor,
		)
	if err != nil {
		handleCoursewareVideoServiceError(
			w,
			err,
		)
		return
	}

	utils.Success(w, response)
}

// handleCoursewareVideoServiceError 映射视频生成和积分错误。
//
// 浏览器不接收供应商、模型、内部幂等键或真实成本信息。
func handleCoursewareVideoServiceError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrMediaBillingPriceNotConfigured,
	):
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"视频积分计费尚未配置，请联系管理员",
		)

	case errors.Is(
		err,
		repository.ErrInsufficientBalance,
	):
		utils.Fail(
			w,
			http.StatusPaymentRequired,
			"积分余额不足，暂时无法生成视频",
		)

	case errors.Is(
		err,
		repository.ErrTokenAccountNotFound,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"尚未开通个人积分账户，暂时无法生成视频",
		)

	case errors.Is(
		err,
		repository.ErrAccountSuspended,
	):
		utils.Fail(
			w,
			http.StatusForbidden,
			"积分账户当前不可用，请联系管理员",
		)

	case errors.Is(
		err,
		services.ErrCoursewareVideoBillingInProgress,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"同一视频任务正在提交，请稍后查看本页视频列表",
		)

	case errors.Is(
		err,
		services.ErrCoursewareVideoBillingTerminal,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"该视频提交任务已经结束，请重新发起生成",
		)

	case errors.Is(
		err,
		services.ErrCoursewareVideoBillingIdentityMismatch,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"视频任务身份校验失败，请刷新页面后重试",
		)

	case errors.Is(
		err,
		services.ErrCoursewareVideoBillingOutputMissing,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"视频任务已经提交，但业务资产未正确恢复，请联系管理员处理",
		)

	default:
		handleCoursewareAssetServiceError(
			w,
			err,
		)
	}
}

// SuggestImagePrompt 生成一条或多条页面图片提示词。
func (h *CoursewareAssetHandler) SuggestImagePrompt(
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
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/suggest-image-prompt",
		)
	if coursewareID == "" ||
		pageNumber <= 0 {
		utils.BadRequest(
			w,
			"路径参数错误",
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

	items, err :=
		h.assetService.SuggestImagePrompt(
			r.Context(),
			coursewareID,
			pageNumber,
			actor,
		)
	if err != nil {
		handleCoursewareAssetServiceError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"prompts": items,
		},
	)
}

// SuggestVideoPrompt 生成页面视频分镜物料。
func (h *CoursewareAssetHandler) SuggestVideoPrompt(
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
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/suggest-video-prompt",
		)
	if coursewareID == "" ||
		pageNumber <= 0 {
		utils.BadRequest(
			w,
			"路径参数错误",
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

	items, err :=
		h.assetService.SuggestVideoPrompt(
			r.Context(),
			coursewareID,
			pageNumber,
			actor,
		)
	if err != nil {
		handleCoursewareAssetServiceError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"storyboards": items,
		},
	)
}

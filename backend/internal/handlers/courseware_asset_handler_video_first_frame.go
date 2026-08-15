package handlers

// courseware_asset_handler_video_first_frame.go — 视频分镜首帧图片生成HTTP入口
//
// 安全边界：
//   - 浏览器只提交提示词、参考图URL和业务operation_id；
//   - billing_node_code不属于HTTP协议；
//   - Handler固定使用video_first_frame计费节点；
//   - 图片尺寸固定为2560×1440，保证分镜首帧为16:9；
//   - 正式首帧落盘后必须具备Nginx只读访问权限，供浏览器和视频供应商读取；
//   - operation_id必须为UUID，同一失败重试复用同一值，
//     防止网络异常导致重复调用图片供应商和重复扣积分。

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// GenerateVideoFirstFrame 生成视频分镜首帧图片。
func (h *CoursewareAssetHandler) GenerateVideoFirstFrame(
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
			"/generate-video-first-frame",
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
		Prompt      string `json:"prompt"`
		RefImageURL string `json:"ref_image_url"`
		OperationID string `json:"operation_id"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(
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
	if request.Prompt == "" {
		utils.BadRequest(
			w,
			"首帧图片提示词不能为空",
		)
		return
	}

	parsedOperationID, err :=
		uuid.Parse(
			strings.TrimSpace(
				request.OperationID,
			),
		)
	if err != nil {
		utils.BadRequest(
			w,
			"首帧任务标识无效，请刷新页面后重试",
		)
		return
	}

	response, err :=
		h.assetService.GenerateImage(
			r.Context(),
			&services.GenerateImageServiceRequest{
				CoursewareID:  coursewareID,
				PageNumber:    pageNumber,
				PlaceholderID: "video-first-frame",
				Prompt:        request.Prompt,
				Size:          "2560x1440",
				RefImageURL: strings.TrimSpace(
					request.RefImageURL,
				),
				OperationID: parsedOperationID.String(),

				// 计费节点只能由可信后端入口设置。
				// 浏览器请求体没有对应字段。
				BillingNodeCode: "video_first_frame",
				Actor:           actor,
			},
		)
	if err != nil {
		handleCoursewareAssetServiceError(
			w,
			err,
		)
		return
	}

	utils.Success(w, response)
}

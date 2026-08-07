package handlers

// courseware_asset_handler_image.go — 课件图片生成与素材管理HTTP接口

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// GenerateImage 生成普通课件页面图片。
//
// 请求中的operation_id由前端一次点击生成一个UUID。
// 同一次HTTP网络重放必须沿用同一个值，
// 防止重复调用图片供应商和重复扣除积分。
func (h *CoursewareAssetHandler) GenerateImage(
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
			"/generate-image",
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
		Prompt        string `json:"prompt"`
		PlaceholderID string `json:"placeholder_id"`
		Size          string `json:"size"`
		RefImageURL   string `json:"ref_image_url"`
		OperationID   string `json:"operation_id"`
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
			"图片生成提示词不能为空",
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
			"图片任务标识无效，请刷新页面后重试",
		)
		return
	}

	response, err :=
		h.assetService.GenerateImage(
			r.Context(),
			&services.GenerateImageServiceRequest{
				CoursewareID: coursewareID,
				PageNumber:   pageNumber,
				PlaceholderID: strings.TrimSpace(
					request.PlaceholderID,
				),
				Prompt: request.Prompt,
				Size: strings.TrimSpace(
					request.Size,
				),
				RefImageURL: strings.TrimSpace(
					request.RefImageURL,
				),
				OperationID: parsedOperationID.String(),
				Actor:       actor,
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

// UploadImage 手动上传课件图片。
func (h *CoursewareAssetHandler) UploadImage(
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
			"/upload-image",
		)
	if coursewareID == "" ||
		pageNumber <= 0 {
		utils.BadRequest(
			w,
			"路径参数错误",
		)
		return
	}

	// 必须先授权，再解析multipart。
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
		r.ParseMultipartForm(
			6 << 20,
		); err != nil {
		utils.BadRequest(
			w,
			"文件解析失败: "+err.Error(),
		)
		return
	}

	file, header, err :=
		r.FormFile("file")
	if err != nil {
		utils.BadRequest(
			w,
			"缺少文件字段 file",
		)
		return
	}
	defer file.Close()

	response, err :=
		h.assetService.UploadAsset(
			r.Context(),
			&services.UploadAssetRequest{
				CoursewareID: coursewareID,
				PageNumber:   pageNumber,
				PlaceholderID: r.FormValue(
					"placeholder_id",
				),
				Actor: actor,
			},
			file,
			header,
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

// ListPageAssets 查询指定页面的全部素材。
func (h *CoursewareAssetHandler) ListPageAssets(
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

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/assets",
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

	assets, err :=
		h.assetService.ListPageAssets(
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

	if assets == nil {
		assets =
			[]*models.CoursewareAsset{}
	}

	utils.Success(
		w,
		map[string]interface{}{
			"assets": assets,
			"total":  len(assets),
		},
	)
}

// ListCoursewareAssets 查询课件全部素材。
func (h *CoursewareAssetHandler) ListCoursewareAssets(
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
		extractCoursewareMiddleID(
			r.URL.Path,
			"/assets",
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

	assets, err :=
		h.assetService.ListCoursewareAssets(
			r.Context(),
			coursewareID,
			actor,
		)
	if err != nil {
		handleCoursewareAssetServiceError(
			w,
			err,
		)
		return
	}

	if assets == nil {
		assets =
			[]*models.CoursewareAsset{}
	}

	utils.Success(
		w,
		map[string]interface{}{
			"assets": assets,
			"total":  len(assets),
		},
	)
}

// DeleteAsset 删除课件素材。
func (h *CoursewareAssetHandler) DeleteAsset(
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
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID :=
		extractCWAssetCoursewareID(
			r.URL.Path,
		)
	assetID :=
		extractCWAssetID(
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

	if err :=
		h.assetService.DeleteAsset(
			r.Context(),
			coursewareID,
			assetID,
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
			"message": "资产删除成功",
		},
	)
}

// InsertImage 将已有图片插入页面HTML。
func (h *CoursewareAssetHandler) InsertImage(
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
			"/insert-image",
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
		AssetID string `json:"asset_id"`
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

	request.AssetID =
		strings.TrimSpace(
			request.AssetID,
		)
	if request.AssetID == "" {
		utils.BadRequest(
			w,
			"asset_id不能为空",
		)
		return
	}

	updatedHTML, err :=
		h.assetService.InsertImageToPage(
			r.Context(),
			coursewareID,
			pageNumber,
			request.AssetID,
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
			"page_number":  pageNumber,
			"html_content": updatedHTML,
			"message":      "图片已插入页面",
		},
	)
}

// UploadToOSS 将已有课件资产上传到OSS。
func (h *CoursewareAssetHandler) UploadToOSS(
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

	coursewareID :=
		extractCWAssetCoursewareID(
			r.URL.Path,
		)
	assetID :=
		extractUploadOSSAssetID(
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

	result, err :=
		h.assetService.UploadCoursewareAssetToOSS(
			r.Context(),
			coursewareID,
			assetID,
			actor,
		)
	if err != nil {
		handleCoursewareAssetServiceError(
			w,
			err,
		)
		return
	}

	utils.Success(w, result)
}

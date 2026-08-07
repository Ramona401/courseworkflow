package handlers

// courseware_asset_handler.go — 课件多媒体资产HTTP处理器公共模块
//
// 具体接口按职责拆分：
//   - courseware_asset_handler_image.go：图片生成、上传、查询、删除和插入；
//   - courseware_asset_handler_upload.go：视频和音频手动上传；
//   - courseware_asset_handler_video_prompt.go：视频生成、状态查询和提示词生成；
//   - 其它既有课件资产处理器文件继续承载风格锚点和物料存储接口。

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CoursewareAssetHandler 课件多媒体资产处理器。
type CoursewareAssetHandler struct {
	assetService *services.CoursewareAssetService
	ossService   *services.OSSService
}

// NewCoursewareAssetHandler 创建课件多媒体资产处理器。
func NewCoursewareAssetHandler(
	assetService *services.CoursewareAssetService,
	ossService *services.OSSService,
) *CoursewareAssetHandler {
	return &CoursewareAssetHandler{
		assetService: assetService,
		ossService:   ossService,
	}
}

// requireCoursewareAssetOwnerActor 构造可信课件Actor并完成作者权限预检。
//
// 上传接口必须在ParseMultipartForm之前调用，
// 避免无权请求提前消耗服务器内存和临时磁盘。
func requireCoursewareAssetOwnerActor(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	userID string,
	role string,
) (*services.CoursewareActorContext, bool) {
	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		userID,
		role,
	)

	_, scopedActor, err :=
		(&services.CoursewareService{}).
			LoadCoursewareForOwnerRuntime(
				r.Context(),
				coursewareID,
				actor,
			)
	if err != nil {
		handleCoursewareAccessError(
			w,
			err,
			"课件素材操作授权失败",
		)
		return nil, false
	}

	return scopedActor, true
}

// handleCoursewareAssetServiceError 映射课件资产和积分计费错误。
//
// 普通用户只看到积分和业务状态，
// 不返回媒体供应商、真实模型、内部幂等键或成本数据。
func handleCoursewareAssetServiceError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrCoursewareActorRequired,
	),
		errors.Is(
			err,
			services.ErrCoursewareAccessNotFound,
		),
		errors.Is(
			err,
			services.ErrCoursewareOwnerRuntimeDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainInvalid,
		),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainMismatch,
		),
		errors.Is(
			err,
			services.ErrCoursewareRuntimeDomainRequired,
		):
		handleCoursewareAccessError(
			w,
			err,
			"课件素材操作授权失败",
		)

	case errors.Is(
		err,
		services.ErrMediaBillingPriceNotConfigured,
	):
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"图片积分计费尚未配置，请联系管理员",
		)

	case errors.Is(
		err,
		repository.ErrInsufficientBalance,
	):
		utils.Fail(
			w,
			http.StatusPaymentRequired,
			"积分余额不足，暂时无法生成图片",
		)

	case errors.Is(
		err,
		repository.ErrTokenAccountNotFound,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"尚未开通个人积分账户，暂时无法生成图片",
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
		services.ErrCoursewareImageBillingInProgress,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"同一图片任务正在处理中，请稍后查看结果",
		)

	case errors.Is(
		err,
		services.ErrCoursewareImageBillingTerminal,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"该图片任务已经结束，请重新发起生成",
		)

	case errors.Is(
		err,
		services.ErrCoursewareImageBillingOutputMissing,
	),
		errors.Is(
			err,
			services.ErrCoursewareImageBillingAssetLost,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			"图片调用已经完成，但业务资产未正确形成，请联系管理员处理",
		)

	case errors.Is(
		err,
		services.ErrCoursewareImageBillingIdentityMismatch,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"图片任务身份校验失败，请刷新页面后重试",
		)

	default:
		utils.InternalError(
			w,
			err.Error(),
		)
	}
}

// extractCWAssetPageActionPath 从页面级资产路径提取课件ID和页码。
func extractCWAssetPageActionPath(
	path string,
	action string,
) (string, int) {
	if !strings.HasSuffix(path, action) &&
		!strings.HasSuffix(path, action+"/") {
		return "", 0
	}

	trimmed := strings.TrimSuffix(
		strings.TrimSuffix(path, "/"),
		action,
	)

	pagesIndex := strings.LastIndex(
		trimmed,
		"/pages/",
	)
	if pagesIndex < 0 {
		return "", 0
	}

	numberText := trimmed[pagesIndex+len("/pages/"):]
	numberText = strings.TrimRight(
		numberText,
		"/",
	)

	pageNumber, err :=
		strconv.Atoi(numberText)
	if err != nil || pageNumber <= 0 {
		return "", 0
	}

	prefix := trimmed[:pagesIndex]
	const coursewarePrefix = "/api/v1/coursewares/"

	if !strings.HasPrefix(
		prefix,
		coursewarePrefix,
	) {
		return "", 0
	}

	coursewareID := prefix[len(coursewarePrefix):]
	if coursewareID == "" ||
		strings.Contains(coursewareID, "/") {
		return "", 0
	}

	return coursewareID, pageNumber
}

// extractCWAssetCoursewareID 从包含/assets/的路径提取课件ID。
func extractCWAssetCoursewareID(
	path string,
) string {
	const prefix = "/api/v1/coursewares/"
	const marker = "/assets/"

	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	rest := strings.TrimPrefix(
		path,
		prefix,
	)
	index := strings.Index(
		rest,
		marker,
	)
	if index <= 0 {
		return ""
	}

	coursewareID := rest[:index]
	if coursewareID == "" ||
		strings.Contains(coursewareID, "/") {
		return ""
	}

	return coursewareID
}

// extractCWAssetID 从资产删除路径提取资产ID。
func extractCWAssetID(
	path string,
) string {
	const marker = "/assets/"

	index := strings.LastIndex(
		path,
		marker,
	)
	if index < 0 {
		return ""
	}

	assetID := path[index+len(marker):]
	assetID = strings.TrimRight(
		assetID,
		"/",
	)

	if assetID == "" ||
		strings.Contains(assetID, "/") {
		return ""
	}

	return assetID
}

// extractCWVideoStatusAssetID 从视频状态路径提取资产ID。
func extractCWVideoStatusAssetID(
	path string,
) string {
	const suffix = "/video-status"

	if !strings.HasSuffix(path, suffix) &&
		!strings.HasSuffix(path, suffix+"/") {
		return ""
	}

	trimmed := strings.TrimSuffix(
		strings.TrimSuffix(path, "/"),
		suffix,
	)

	const marker = "/assets/"
	index := strings.LastIndex(
		trimmed,
		marker,
	)
	if index < 0 {
		return ""
	}

	assetID := trimmed[index+len(marker):]
	if assetID == "" ||
		strings.Contains(assetID, "/") {
		return ""
	}

	return assetID
}

// extractUploadOSSAssetID 从资产上云路径提取资产ID。
func extractUploadOSSAssetID(
	path string,
) string {
	const suffix = "/upload-oss"

	if !strings.HasSuffix(path, suffix) &&
		!strings.HasSuffix(path, suffix+"/") {
		return ""
	}

	trimmed := strings.TrimSuffix(
		strings.TrimSuffix(path, "/"),
		suffix,
	)

	const marker = "/assets/"
	index := strings.LastIndex(
		trimmed,
		marker,
	)
	if index < 0 {
		return ""
	}

	assetID := trimmed[index+len(marker):]
	if assetID == "" ||
		strings.Contains(assetID, "/") {
		return ""
	}

	return assetID
}

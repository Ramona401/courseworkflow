package handlers

// courseware_asset_handler.go — 课件多媒体资产HTTP处理器
//
// v0.42 多媒体:AI图片生成(含参考图) + 手动上传 + 列表 + 删除 + 插入HTML
// v0.42.1 新增:AI视频生成(提交任务+查询状态)
// v0.42.5 新增:手动上传视频(UploadVideo)
// v0.42.6+ P2.4:UploadVideo 成功后写入审计日志 audit_logs (courseware.video_upload)
// v0.42.10 新增:上传资产到阿里云OSS(UploadToOSS)，返回公网URL供复制使用
// 图片多提示词(本轮):SuggestImagePrompt 响应由 {prompt} 改为 {prompts:[{caption,prompt}]}，
//   AI 按本页配图需求自主判断该页要几张图(1-N 条)，前端渲染为建议卡片列表。
// 视频锚点轮(本轮):GenerateVideo 请求体新增 source_frame_asset_id(首帧图资产ID)，
//   两步流"先出首帧图再生视频"时传入，透传给 service 写 metadata 溯源；空=直接文字生视频。
//
// 接口:
//   POST   /api/v1/coursewares/{id}/pages/{num}/generate-image  — AI生成图片
//   POST   /api/v1/coursewares/{id}/pages/{num}/upload-image    — 手动上传图片
//   POST   /api/v1/coursewares/{id}/pages/{num}/upload-video    — v0.42.5 手动上传视频
//   GET    /api/v1/coursewares/{id}/pages/{num}/assets           — 获取页面图片/视频列表
//   GET    /api/v1/coursewares/{id}/assets                       — 获取课件全部图片/视频
//   DELETE /api/v1/coursewares/{id}/assets/{asset_id}            — 删除图片/视频
//   POST   /api/v1/coursewares/{id}/pages/{num}/insert-image     — 将图片插入到页面HTML
//   POST   /api/v1/coursewares/{id}/pages/{num}/generate-video   — v0.42.1 AI生成视频(异步提交)
//   GET    /api/v1/coursewares/{id}/assets/{asset_id}/video-status — v0.42.1 查询视频生成状态
//   POST   /api/v1/coursewares/{id}/assets/{asset_id}/upload-oss  — v0.42.10 上传资产到阿里云OSS
//   POST   /api/v1/coursewares/{id}/pages/{num}/suggest-image-prompt — AI 写详细生图提示词(多条)
//   POST   /api/v1/coursewares/{id}/pages/{num}/suggest-video-prompt — AI 写视频三件物料

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== 课件多媒体资产处理器 ====================

// CoursewareAssetHandler 课件多媒体资产处理器
type CoursewareAssetHandler struct {
	assetService *services.CoursewareAssetService
	ossService   *services.OSSService // v0.42.10: OSS上传服务
}

// NewCoursewareAssetHandler 创建课件多媒体资产处理器
// v0.42.10: 新增ossService参数，用于上传资产到阿里云OSS
func NewCoursewareAssetHandler(assetService *services.CoursewareAssetService, ossService *services.OSSService) *CoursewareAssetHandler {
	return &CoursewareAssetHandler{
		assetService: assetService,
		ossService:   ossService,
	}
}

// requireCoursewareAssetOwnerActor 在进入作者私有素材库前构造可信Actor并完成预检。
//
// 上传接口必须在ParseMultipartForm之前调用，避免无权请求先消耗内存或临时磁盘。
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

// handleCoursewareAssetServiceError 保留原素材业务错误正文，
// 仅把可信Actor和教育域授权错误交给统一课件访问错误映射。
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
	default:
		utils.InternalError(
			w,
			err.Error(),
		)
	}
}

// ==================== AI生成图片 ====================

// GenerateImage POST /api/v1/coursewares/{id}/pages/{num}/generate-image
//
//	请求体: {
//	  "prompt": "一张展示AI机器人的卡通插图",
//	  "placeholder_id": "IMG_01",    // 可选:占位符ID
//	  "size": "2560x1440",           // 可选:图片尺寸,默认1920x1920
//	  "ref_image_url": "/uploads/courseware-assets/xxx/p1/xxx.jpg"  // 可选:参考图URL
//	}
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/generate-image",
		)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径参数错误")
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
	}

	if err := json.NewDecoder(r.Body).Decode(
		&request,
	); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}
	if strings.TrimSpace(request.Prompt) == "" {
		utils.BadRequest(
			w,
			"图片生成提示词不能为空",
		)
		return
	}

	response, err := h.assetService.GenerateImage(
		r.Context(),
		&services.GenerateImageServiceRequest{
			CoursewareID:  coursewareID,
			PageNumber:    pageNumber,
			PlaceholderID: request.PlaceholderID,
			Prompt:        request.Prompt,
			Size:          request.Size,
			RefImageURL:   request.RefImageURL,
			Actor:         actor,
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

// ==================== 手动上传图片 ====================

// UploadImage POST /api/v1/coursewares/{id}/pages/{num}/upload-image
// Content-Type: multipart/form-data
// 字段: file(图片) + placeholder_id(可选)
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/upload-image",
		)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径参数错误")
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

	if err := r.ParseMultipartForm(
		6 << 20,
	); err != nil {
		utils.BadRequest(
			w,
			"文件解析失败: "+err.Error(),
		)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequest(
			w,
			"缺少文件字段 file",
		)
		return
	}
	defer file.Close()

	response, err := h.assetService.UploadAsset(
		r.Context(),
		&services.UploadAssetRequest{
			CoursewareID:  coursewareID,
			PageNumber:    pageNumber,
			PlaceholderID: r.FormValue("placeholder_id"),
			Actor:         actor,
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

// ==================== v0.42.5 手动上传视频 ====================

// UploadVideo POST /api/v1/coursewares/{id}/pages/{num}/upload-video
// Content-Type: multipart/form-data
// 字段: file(视频文件,仅 file 字段,不接收 placeholder_id)
// 支持格式: MP4/WebM/MOV/AVI
// 大小限制: ≤ 50MB(Nginx client_max_body_size=55M 已支持)
//
// 说明:
//   - 视频不替换占位符,上传后直接加入素材库,前端在视频编辑器中使用
//   - 存储路径与 AI 生成视频一致: /uploads/courseware-assets/{cwID}/videos/
//   - ParseMultipartForm 缓冲设为 10MB,超出部分自动写入 /tmp 临时文件
//     由 Go 标准库自动清理,避免大视频常驻内存
//
// v0.42.6+ P2.4: 上传成功后异步写入 audit_logs(courseware.video_upload),
// detail JSONB 含 courseware_id/page_number/asset_id/file_size/mime_type/original_filename,
// 便于后续审计追溯"谁在什么时候上传了什么视频"。审计日志是 fire-and-forget,
// 写入失败不影响上传响应。
func (h *CoursewareAssetHandler) UploadVideo(
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/upload-video",
		)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	// 大体积视频必须先授权，再解析multipart。
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

	if err := r.ParseMultipartForm(
		10 << 20,
	); err != nil {
		utils.BadRequest(
			w,
			"视频文件解析失败: "+err.Error(),
		)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequest(
			w,
			"缺少文件字段 file",
		)
		return
	}
	defer file.Close()

	originalFilename := header.Filename

	response, err :=
		h.assetService.UploadVideoAsset(
			r.Context(),
			&services.UploadVideoAssetRequest{
				CoursewareID: coursewareID,
				PageNumber:   pageNumber,
				Actor:        actor,
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

	repository.WriteAuditLog(
		claims.UserID,
		"courseware.video_upload",
		map[string]interface{}{
			"courseware_id":     coursewareID,
			"page_number":       pageNumber,
			"asset_id":          response.AssetID,
			"file_size":         response.FileSize,
			"mime_type":         response.MimeType,
			"original_filename": originalFilename,
			"stored_filename":   response.FileName,
		},
		repository.GetClientIP(
			r.RemoteAddr,
		),
	)

	utils.Success(w, response)
}

// ==================== 手动上传音频 ====================

// UploadAudio POST /api/v1/coursewares/{id}/pages/{num}/upload-audio
// Content-Type: multipart/form-data
// 字段: file（音频文件，仅 file 字段）
// 支持格式: MP3/WAV/OGG/AAC/FLAC/M4A
// 大小限制: ≤ 20MB
//
// 说明:
//   - 音频上传后加入素材库，老师可上云获取公网链接
//   - 存储路径: /uploads/courseware-assets/{cwID}/audios/
//   - 上传成功后写入审计日志（courseware.audio_upload）
func (h *CoursewareAssetHandler) UploadAudio(
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/upload-audio",
		)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	// 音频必须先授权，再解析multipart。
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

	if err := r.ParseMultipartForm(
		10 << 20,
	); err != nil {
		utils.BadRequest(
			w,
			"音频文件解析失败: "+err.Error(),
		)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequest(
			w,
			"缺少文件字段 file",
		)
		return
	}
	defer file.Close()

	originalFilename := header.Filename

	response, err :=
		h.assetService.UploadAudioAsset(
			r.Context(),
			&services.UploadAudioAssetRequest{
				CoursewareID: coursewareID,
				PageNumber:   pageNumber,
				Actor:        actor,
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

	repository.WriteAuditLog(
		claims.UserID,
		"courseware.audio_upload",
		map[string]interface{}{
			"courseware_id":     coursewareID,
			"page_number":       pageNumber,
			"asset_id":          response.AssetID,
			"file_size":         response.FileSize,
			"mime_type":         response.MimeType,
			"original_filename": originalFilename,
			"stored_filename":   response.FileName,
		},
		repository.GetClientIP(
			r.RemoteAddr,
		),
	)

	utils.Success(w, response)
}

// ==================== 查询资产列表 ====================

// ListPageAssets GET /api/v1/coursewares/{id}/pages/{num}/assets
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/assets",
		)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径参数错误")
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

	assets, err := h.assetService.ListPageAssets(
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
		assets = []*models.CoursewareAsset{}
	}

	utils.Success(
		w,
		map[string]interface{}{
			"assets": assets,
			"total":  len(assets),
		},
	)
}

// ListCoursewareAssets GET /api/v1/coursewares/{id}/assets
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID := extractCoursewareMiddleID(
		r.URL.Path,
		"/assets",
	)
	if coursewareID == "" {
		utils.BadRequest(w, "缺少课件ID")
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
		assets = []*models.CoursewareAsset{}
	}

	utils.Success(
		w,
		map[string]interface{}{
			"assets": assets,
			"total":  len(assets),
		},
	)
}

// ==================== 删除资产 ====================

// DeleteAsset DELETE /api/v1/coursewares/{id}/assets/{asset_id}
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID :=
		extractCWAssetCoursewareID(
			r.URL.Path,
		)
	assetID := extractCWAssetID(
		r.URL.Path,
	)

	if coursewareID == "" || assetID == "" {
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

	if err := h.assetService.DeleteAsset(
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

// ==================== 插入图片到HTML ====================

// InsertImage POST /api/v1/coursewares/{id}/pages/{num}/insert-image
// 请求体: { "asset_id": "uuid" }
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/insert-image",
		)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径参数错误")
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

	if err := json.NewDecoder(r.Body).Decode(
		&request,
	); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}
	if strings.TrimSpace(request.AssetID) == "" {
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
			strings.TrimSpace(
				request.AssetID,
			),
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

// ==================== v0.42.1 AI生成视频(异步提交) ====================

// GenerateVideo POST /api/v1/coursewares/{id}/pages/{num}/generate-video
//
//	请求体: {
//	  "prompt": "一位教师在讲台前讲解人工智能的基本概念",
//	  "ref_image_url": "/uploads/courseware-assets/xxx/p1/xxx.jpg",  // 可选:参考图(图生视频)
//	  "source_frame_asset_id": "uuid"  // 可选:首帧图资产ID(两步流时传,写metadata溯源;空=直接文字生视频)
//	}
//
// 视频锚点轮:两步流"先出首帧图→确认→生视频"时，前端把已确认首帧图的 URL 作为 ref_image_url、
// 首帧图的资产ID作为 source_frame_asset_id 一并传入。前者实现图生视频锁定风格人物，
// 后者由 service 写入视频资产 metadata 建立"视频←首帧图"血缘。
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/generate-video",
		)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径参数错误")
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
	}

	if err := json.NewDecoder(r.Body).Decode(
		&request,
	); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}
	if strings.TrimSpace(request.Prompt) == "" {
		utils.BadRequest(
			w,
			"视频描述提示词不能为空",
		)
		return
	}

	response, err := h.assetService.GenerateVideo(
		r.Context(),
		&services.GenerateVideoServiceRequest{
			CoursewareID: coursewareID,
			PageNumber:   pageNumber,
			Prompt: strings.TrimSpace(
				request.Prompt,
			),
			RefImageURL: strings.TrimSpace(
				request.RefImageURL,
			),
			Actor: actor,
			SourceFrameAssetID: strings.TrimSpace(
				request.SourceFrameAssetID,
			),
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

// ==================== v0.42.1 查询视频生成状态 ====================

// QueryVideoStatus GET /api/v1/coursewares/{id}/assets/{asset_id}/video-status
// 前端轮询此接口,直到返回 status=uploaded(成功)或 status=failed(失败)
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

	claims, ok := middleware.GetClaims(r.Context())
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

	if coursewareID == "" || assetID == "" {
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
		handleCoursewareAssetServiceError(
			w,
			err,
		)
		return
	}

	utils.Success(w, response)
}

// ==================== v0.42.10 上传资产到阿里云OSS ====================

// UploadToOSS POST /api/v1/coursewares/{id}/assets/{asset_id}/upload-oss
// 将已有的课件资产（图片/视频/音频）从本地磁盘上传到阿里云OSS
// 返回公网可访问的URL，用户可以复制到微调HTML等场景使用
//
//	响应: {
//	  "asset_id": "uuid",
//	  "local_url": "/uploads/courseware-assets/xxx/p1/xxx.jpg",
//	  "oss_public_url": "https://20260525zuo.oss-cn-beijing.aliyuncs.com/courseware-assets/xxx/p1/xxx.jpg",
//	  "message": "上传云盘成功"
//	}
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

	claims, ok := middleware.GetClaims(r.Context())
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

	if coursewareID == "" || assetID == "" {
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

// ==================== 批次4c+: AI 写详细生图/视频提示词 ====================

// SuggestImagePrompt POST /api/v1/coursewares/{id}/pages/{num}/suggest-image-prompt
// 读本页方案 + 课件风格，调模型生成【一条或多条】详细、可控、紧扣本页教学需求的生图提示词。
// 图片多提示词改造：响应由旧版 {prompt:string} 改为 {prompts:[{caption,prompt}]}，
//
//	AI 按本页配图需求自主判断该页要几张图(1-N 条)，前端渲染为可勾选的建议卡片列表。
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/suggest-image-prompt",
		)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径参数错误")
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

// SuggestVideoPrompt POST /api/v1/coursewares/{id}/pages/{num}/suggest-video-prompt
// 视频分镜(本轮): 返回分镜数组 {storyboards:[{scene,image_prompt,video_prompt,narration}]},
// AI 按本页内容自主拆 1-N 个分镜, 每镜各有首帧图提示词/图生视频提示词/台词, 前端渲染为可切换的镜头卡片。
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/suggest-video-prompt",
		)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径参数错误")
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

// ==================== 路径解析辅助函数 ====================

// extractCWAssetPageActionPath 从 /api/v1/coursewares/{id}/pages/{num}/{action} 提取课件ID和页码
func extractCWAssetPageActionPath(path string, action string) (string, int) {
	if !strings.HasSuffix(path, action) && !strings.HasSuffix(path, action+"/") {
		return "", 0
	}
	trimmed := strings.TrimSuffix(strings.TrimSuffix(path, "/"), action)
	pagesIdx := strings.LastIndex(trimmed, "/pages/")
	if pagesIdx < 0 {
		return "", 0
	}
	numStr := trimmed[pagesIdx+len("/pages/"):]
	numStr = strings.TrimRight(numStr, "/")
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		return "", 0
	}
	prefix := trimmed[:pagesIdx]
	cwPrefix := "/api/v1/coursewares/"
	if !strings.HasPrefix(prefix, cwPrefix) {
		return "", 0
	}
	coursewareID := prefix[len(cwPrefix):]
	if coursewareID == "" {
		return "", 0
	}
	return coursewareID, num
}

// extractCWAssetCoursewareID 从包含/assets/的课件素材路径提取课件ID。
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

// extractCWAssetID 从 /api/v1/coursewares/{id}/assets/{asset_id} 提取资产ID
func extractCWAssetID(path string) string {
	const marker = "/assets/"
	idx := strings.LastIndex(path, marker)
	if idx < 0 {
		return ""
	}
	rest := path[idx+len(marker):]
	rest = strings.TrimRight(rest, "/")
	if strings.Contains(rest, "/") {
		return ""
	}
	return rest
}

// extractCWVideoStatusAssetID 从 /api/v1/coursewares/{id}/assets/{asset_id}/video-status 提取资产ID
func extractCWVideoStatusAssetID(path string) string {
	const suffix = "/video-status"
	if !strings.HasSuffix(path, suffix) && !strings.HasSuffix(path, suffix+"/") {
		return ""
	}
	trimmed := strings.TrimSuffix(strings.TrimSuffix(path, "/"), suffix)
	const marker = "/assets/"
	idx := strings.LastIndex(trimmed, marker)
	if idx < 0 {
		return ""
	}
	assetID := trimmed[idx+len(marker):]
	if assetID == "" || strings.Contains(assetID, "/") {
		return ""
	}
	return assetID
}

// extractUploadOSSAssetID 从 /api/v1/coursewares/{id}/assets/{asset_id}/upload-oss 提取资产ID
func extractUploadOSSAssetID(path string) string {
	const suffix = "/upload-oss"
	if !strings.HasSuffix(path, suffix) && !strings.HasSuffix(path, suffix+"/") {
		return ""
	}
	trimmed := strings.TrimSuffix(strings.TrimSuffix(path, "/"), suffix)
	const marker = "/assets/"
	idx := strings.LastIndex(trimmed, marker)
	if idx < 0 {
		return ""
	}
	assetID := trimmed[idx+len(marker):]
	if assetID == "" || strings.Contains(assetID, "/") {
		return ""
	}
	return assetID
}

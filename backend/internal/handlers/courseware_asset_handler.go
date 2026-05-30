package handlers

// courseware_asset_handler.go — 课件多媒体资产HTTP处理器
//
// v0.42 多媒体:AI图片生成(含参考图) + 手动上传 + 列表 + 删除 + 插入HTML
// v0.42.1 新增:AI视频生成(提交任务+查询状态)
// v0.42.5 新增:手动上传视频(UploadVideo)
// v0.42.6+ P2.4:UploadVideo 成功后写入审计日志 audit_logs (courseware.video_upload)
// v0.42.10 新增:上传资产到阿里云OSS(UploadToOSS)，返回公网URL供复制使用
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

import (
	"encoding/json"
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

// ==================== AI生成图片 ====================

// GenerateImage POST /api/v1/coursewares/{id}/pages/{num}/generate-image
// 请求体: {
//   "prompt": "一张展示AI机器人的卡通插图",
//   "placeholder_id": "IMG_01",    // 可选:占位符ID
//   "size": "2560x1440",           // 可选:图片尺寸,默认1920x1920
//   "ref_image_url": "/uploads/courseware-assets/xxx/p1/xxx.jpg"  // 可选:参考图URL
// }
func (h *CoursewareAssetHandler) GenerateImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID, pageNum := extractCWAssetPageActionPath(r.URL.Path, "/generate-image")
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	var req struct {
		Prompt        string `json:"prompt"`
		PlaceholderID string `json:"placeholder_id"`
		Size          string `json:"size"`
		RefImageURL   string `json:"ref_image_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.Prompt == "" {
		utils.BadRequest(w, "图片生成提示词不能为空")
		return
	}

	svcReq := &services.GenerateImageServiceRequest{
		CoursewareID:  cwID,
		PageNumber:    pageNum,
		PlaceholderID: req.PlaceholderID,
		Prompt:        req.Prompt,
		Size:          req.Size,
		RefImageURL:   req.RefImageURL,
		UserID:        claims.UserID,
	}

	resp, err := h.assetService.GenerateImage(r.Context(), svcReq)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 手动上传图片 ====================

// UploadImage POST /api/v1/coursewares/{id}/pages/{num}/upload-image
// Content-Type: multipart/form-data
// 字段: file(图片) + placeholder_id(可选)
func (h *CoursewareAssetHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID, pageNum := extractCWAssetPageActionPath(r.URL.Path, "/upload-image")
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	if err := r.ParseMultipartForm(6 << 20); err != nil {
		utils.BadRequest(w, "文件解析失败: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequest(w, "缺少文件字段 file")
		return
	}
	defer file.Close()

	placeholderID := r.FormValue("placeholder_id")

	svcReq := &services.UploadAssetRequest{
		CoursewareID:  cwID,
		PageNumber:    pageNum,
		PlaceholderID: placeholderID,
		UserID:        claims.UserID,
	}

	resp, err := h.assetService.UploadAsset(r.Context(), svcReq, file, header)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
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
func (h *CoursewareAssetHandler) UploadVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 路径解析: 复用图片上传相同的工具函数
	cwID, pageNum := extractCWAssetPageActionPath(r.URL.Path, "/upload-video")
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	// 视频文件较大,10MB 内存缓冲(超出部分自动落盘到 /tmp)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.BadRequest(w, "视频文件解析失败: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequest(w, "缺少文件字段 file")
		return
	}
	defer file.Close()

	// 保存原始文件名供审计日志使用(header 在 defer file.Close 后不能再访问 underlying)
	// header 本身是 *multipart.FileHeader,字段是值类型,这里安全
	originalFilename := header.Filename

	svcReq := &services.UploadVideoAssetRequest{
		CoursewareID: cwID,
		PageNumber:   pageNum,
		UserID:       claims.UserID,
	}

	resp, err := h.assetService.UploadVideoAsset(r.Context(), svcReq, file, header)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	// ========== P2.4: 上传成功后异步写入审计日志 ==========
	// 操作名 "courseware.video_upload" 已在 audit_repo.go 的 actionNameMap 注册为"上传视频"
	// detail JSONB 字段帮助后续审计/统计/合规检查
	// WriteAuditLog 是 fire-and-forget 异步,失败仅 logger 记录不影响主响应
	repository.WriteAuditLog(
		claims.UserID,
		"courseware.video_upload",
		map[string]interface{}{
			"courseware_id":     cwID,
			"page_number":       pageNum,
			"asset_id":          resp.AssetID,
			"file_size":         resp.FileSize,
			"mime_type":         resp.MimeType,
			"original_filename": originalFilename,
			"stored_filename":   resp.FileName,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	utils.Success(w, resp)
}

// ==================== 查询资产列表 ====================

// ListPageAssets GET /api/v1/coursewares/{id}/pages/{num}/assets
func (h *CoursewareAssetHandler) ListPageAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID, pageNum := extractCWAssetPageActionPath(r.URL.Path, "/assets")
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	assets, err := h.assetService.ListPageAssets(r.Context(), cwID, pageNum, claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	if assets == nil {
		assets = []*models.CoursewareAsset{}
	}
	utils.Success(w, map[string]interface{}{
		"assets": assets,
		"total":  len(assets),
	})
}

// ListCoursewareAssets GET /api/v1/coursewares/{id}/assets
func (h *CoursewareAssetHandler) ListCoursewareAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID := extractCoursewareMiddleID(r.URL.Path, "/assets")
	if cwID == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	assets, err := h.assetService.ListCoursewareAssets(r.Context(), cwID, claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	if assets == nil {
		assets = []*models.CoursewareAsset{}
	}
	utils.Success(w, map[string]interface{}{
		"assets": assets,
		"total":  len(assets),
	})
}

// ==================== 删除资产 ====================

// DeleteAsset DELETE /api/v1/coursewares/{id}/assets/{asset_id}
func (h *CoursewareAssetHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	assetID := extractCWAssetID(r.URL.Path)
	if assetID == "" {
		utils.BadRequest(w, "缺少资产ID")
		return
	}

	if err := h.assetService.DeleteAsset(r.Context(), assetID, claims.UserID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "资产删除成功"})
}

// ==================== 插入图片到HTML ====================

// InsertImage POST /api/v1/coursewares/{id}/pages/{num}/insert-image
// 请求体: { "asset_id": "uuid" }
func (h *CoursewareAssetHandler) InsertImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID, pageNum := extractCWAssetPageActionPath(r.URL.Path, "/insert-image")
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	var req struct {
		AssetID string `json:"asset_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.AssetID == "" {
		utils.BadRequest(w, "asset_id不能为空")
		return
	}

	updatedHTML, err := h.assetService.InsertImageToPage(r.Context(), cwID, pageNum, req.AssetID, claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, map[string]interface{}{
		"page_number":  pageNum,
		"html_content": updatedHTML,
		"message":      "图片已插入页面",
	})
}

// ==================== v0.42.1 AI生成视频(异步提交) ====================

// GenerateVideo POST /api/v1/coursewares/{id}/pages/{num}/generate-video
// 请求体: {
//   "prompt": "一位教师在讲台前讲解人工智能的基本概念",
//   "ref_image_url": "/uploads/courseware-assets/xxx/p1/xxx.jpg"  // 可选:参考图(图生视频)
// }
func (h *CoursewareAssetHandler) GenerateVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID, pageNum := extractCWAssetPageActionPath(r.URL.Path, "/generate-video")
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	var req struct {
		Prompt      string `json:"prompt"`
		RefImageURL string `json:"ref_image_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.Prompt == "" {
		utils.BadRequest(w, "视频描述提示词不能为空")
		return
	}

	svcReq := &services.GenerateVideoServiceRequest{
		CoursewareID: cwID,
		PageNumber:   pageNum,
		Prompt:       req.Prompt,
		RefImageURL:  req.RefImageURL,
		UserID:       claims.UserID,
	}

	resp, err := h.assetService.GenerateVideo(r.Context(), svcReq)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== v0.42.1 查询视频生成状态 ====================

// QueryVideoStatus GET /api/v1/coursewares/{id}/assets/{asset_id}/video-status
// 前端轮询此接口,直到返回 status=uploaded(成功)或 status=failed(失败)
func (h *CoursewareAssetHandler) QueryVideoStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 从路径 /api/v1/coursewares/{id}/assets/{asset_id}/video-status 提取 asset_id
	assetID := extractCWVideoStatusAssetID(r.URL.Path)
	if assetID == "" {
		utils.BadRequest(w, "缺少资产ID")
		return
	}

	resp, err := h.assetService.QueryVideoStatus(r.Context(), assetID, claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== v0.42.10 上传资产到阿里云OSS ====================

// UploadToOSS POST /api/v1/coursewares/{id}/assets/{asset_id}/upload-oss
// 将已有的课件资产（图片/视频/音频）从本地磁盘上传到阿里云OSS
// 返回公网可访问的URL，用户可以复制到微调HTML等场景使用
//
// 响应: {
//   "asset_id": "uuid",
//   "local_url": "/uploads/courseware-assets/xxx/p1/xxx.jpg",
//   "oss_public_url": "https://20260525zuo.oss-cn-beijing.aliyuncs.com/courseware-assets/xxx/p1/xxx.jpg",
//   "message": "上传云盘成功"
// }
func (h *CoursewareAssetHandler) UploadToOSS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 从路径 /api/v1/coursewares/{id}/assets/{asset_id}/upload-oss 提取 asset_id
	assetID := extractUploadOSSAssetID(r.URL.Path)
	if assetID == "" {
		utils.BadRequest(w, "缺少资产ID")
		return
	}

	// 1. 查询资产记录
	asset, err := repository.GetCWAssetByID(r.Context(), assetID)
	if err != nil {
		utils.InternalError(w, "资产不存在: "+err.Error())
		return
	}

	// 2. 权限校验：只有课件所有者可以上传
	cw, err := repository.GetCoursewareByID(r.Context(), asset.CoursewareID)
	if err != nil {
		utils.InternalError(w, "课件不存在: "+err.Error())
		return
	}
	if cw.UserID != claims.UserID {
		utils.Fail(w, http.StatusForbidden, "无权操作此课件")
		return
	}

	// 3. 检查资产URL是否为本地路径
	if asset.OssURL == "" || !strings.HasPrefix(asset.OssURL, "/uploads/") {
		utils.BadRequest(w, "资产没有本地文件或已经是外部URL")
		return
	}

	// 4. 调用OSS上传服务
	publicURL, err := h.ossService.UploadAssetToOSS(asset.OssURL)
	if err != nil {
		utils.InternalError(w, "上传云盘失败: "+err.Error())
		return
	}

	// 5. 返回结果
	utils.Success(w, map[string]interface{}{
		"asset_id":       assetID,
		"local_url":      asset.OssURL,
		"oss_public_url": publicURL,
		"message":        "上传云盘成功",
	})
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

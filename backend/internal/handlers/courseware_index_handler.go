package handlers

// courseware_index_handler.go — 课件索引生成HTTP处理器 v3
//
// v3 (v0.42 入口B) 变更:
//   - CoursewareIndexHandler 新增 pptService 字段
//   - 新增 GenerateIndexFromPPT 端点（从PPT内容生成索引）
//   - NewCoursewareIndexHandler 签名不变（PPT服务通过 SetPPTService 注入）
//
// v2 修复：
//   - GenerateIndex 异步goroutine启动前增加800ms延迟，确保前端SSE连接建立后再执行
//
// 提供接口：
//   1. POST /api/v1/coursewares/{id}/generate-index          — 触发AI生成索引（异步）
//   2. GET  /api/v1/sse/courseware/{id}                       — SSE订阅索引生成进度
//   3. DELETE /api/v1/coursewares/{id}/pages/{num}            — 删除单页
//   4. POST /api/v1/coursewares/{id}/generate-index-topic     — 从主题生成索引（v0.42）
//   5. POST /api/v1/coursewares/{id}/generate-index-ppt      — 从PPT内容生成索引（v0.42 入口B）

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== 课件索引处理器 ====================

// CoursewareIndexHandler 课件索引生成处理器
type CoursewareIndexHandler struct {
	indexService *services.CoursewareIndexService
	cwService    *services.CoursewareService
	authService  *services.AuthService
	pptService   *services.CoursewarePPTService // v0.42 入口B: PPT解析服务（可选注入）
}

// NewCoursewareIndexHandler 创建课件索引处理器
func NewCoursewareIndexHandler(
	indexService *services.CoursewareIndexService,
	cwService *services.CoursewareService,
	authService *services.AuthService,
) *CoursewareIndexHandler {
	return &CoursewareIndexHandler{
		indexService: indexService,
		cwService:    cwService,
		authService:  authService,
	}
}

// SetPPTService v0.42 入口B: 注入PPT解析服务（在routes.go中调用）
func (h *CoursewareIndexHandler) SetPPTService(pptService *services.CoursewarePPTService) {
	h.pptService = pptService
}

// ==================== 触发索引生成 ====================

// GenerateIndex POST /api/v1/coursewares/{id}/generate-index — 触发AI生成课件索引
// 异步执行：立即返回200，通过SSE推送进度
// v2: goroutine启动前延迟800ms，确保前端SSE连接建立后再执行
func (h *CoursewareIndexHandler) GenerateIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(r.URL.Path, "/generate-index")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	userID := claims.UserID
	go func() {
		time.Sleep(800 * time.Millisecond)
		asyncCtx := context.Background()
		if err := h.indexService.GenerateIndex(asyncCtx, id, userID, ""); err != nil {
			fmt.Printf("[courseware_index_handler] 索引生成失败: courseware=%s err=%v\n", id, err)
		}
	}()

	utils.Success(w, map[string]interface{}{
		"message":       "课件索引生成已启动，请通过SSE监听进度",
		"courseware_id": id,
	})
}

// ==================== SSE订阅索引生成进度 ====================

// IndexStream GET /api/v1/sse/courseware/{id}?token=xxx — SSE订阅课件索引生成进度
func (h *CoursewareIndexHandler) IndexStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	token := extractTokenFromQuery(r)
	if token == "" {
		http.Error(w, `{"code":-1,"message":"缺少token参数"}`, http.StatusUnauthorized)
		return
	}
	_, err := h.authService.ValidateToken(token)
	if err != nil {
		http.Error(w, `{"code":-1,"message":"token无效或已过期"}`, http.StatusUnauthorized)
		return
	}

	id := extractCWSSEID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持流式响应", http.StatusInternalServerError)
		return
	}

	ch := services.GlobalCWSSEHub.Subscribe(id)
	defer services.GlobalCWSSEHub.Unsubscribe(id, ch)

	writeCWSSEEvent(w, flusher, services.CWSSEConnected, map[string]string{
		"courseware_id": id,
		"message":      "SSE连接已建立",
	})

	timeout := time.After(20 * time.Minute)
	for {
		select {
		case event, open := <-ch:
			if !open {
				return
			}
			writeCWSSEEvent(w, flusher, event.EventType, event.Data)
			if event.EventType == services.CWSSEIndexDone || event.EventType == services.CWSSEGenDone || event.EventType == services.CWSSEError {
				return
			}
		case <-r.Context().Done():
			return
		case <-timeout:
			writeCWSSEEvent(w, flusher, "timeout", map[string]string{
				"message": "SSE连接超时",
			})
			return
		}
	}
}

// ==================== 删除单页 ====================

// DeletePage DELETE /api/v1/coursewares/{id}/pages/{num} — 删除课件单页
func (h *CoursewareIndexHandler) DeletePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID, pageNum := extractCoursewarePagePath(r.URL.Path)
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	if err := h.cwService.DeletePage(r.Context(), cwID, pageNum, claims.UserID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "页面删除成功"})
}

// ==================== SSE辅助函数 ====================

func writeCWSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(dataBytes))
	flusher.Flush()
}

// ==================== 路径解析 ====================

func extractCWSSEID(path string) string {
	const ssePrefix = "/api/v1/sse/courseware/"
	if strings.HasPrefix(path, ssePrefix) {
		rest := path[len(ssePrefix):]
		rest = strings.TrimRight(rest, "/")
		if idx := strings.Index(rest, "/"); idx > 0 {
			return rest[:idx]
		}
		return rest
	}
	return extractCoursewareMiddleID(path, "/index-stream")
}

func extractTokenFromQuery(r *http.Request) string {
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// ==================== v0.42: 从主题生成索引 ====================

// GenerateIndexFromTopic POST /api/v1/coursewares/{id}/generate-index-topic — 从主题直接生成课件索引
func (h *CoursewareIndexHandler) GenerateIndexFromTopic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(r.URL.Path, "/generate-index-topic")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	var reqBody struct {
		Subject    string `json:"subject"`
		Grade      string `json:"grade"`
		Topic      string `json:"topic"`
		PageRange  string `json:"page_range"`
		ExtraNotes string `json:"extra_notes"`
		Preset     string `json:"preset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	req := &models.CreateCoursewareFromTopicRequest{
		Subject:    reqBody.Subject,
		Grade:      reqBody.Grade,
		Topic:      reqBody.Topic,
		PageRange:  reqBody.PageRange,
		ExtraNotes: reqBody.ExtraNotes,
	}

	go func() {
		time.Sleep(800 * time.Millisecond)
		ctx := context.Background()
		if err := h.indexService.GenerateIndexFromTopic(ctx, id, claims.UserID, req, reqBody.Preset); err != nil {
			_ = err
		}
	}()

	utils.Success(w, map[string]string{
		"courseware_id": id,
		"message":      "主题课件方案生成已启动，请通过SSE接收进度",
	})
}

// ==================== v136: AI修改方案+预设支持 ====================

// GenerateIndexWithPreset POST /api/v1/coursewares/{id}/generate-index — 带预设参数的索引生成
func (h *CoursewareIndexHandler) GenerateIndexWithPreset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(r.URL.Path, "/generate-index")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	// 解析可选的preset参数
	var reqBody struct {
		Preset string `json:"preset"`
	}
	// 请求体可能为空（兼容旧前端），解析失败不报错
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	userID := claims.UserID
	preset := reqBody.Preset
	go func() {
		time.Sleep(800 * time.Millisecond)
		asyncCtx := context.Background()
		if err := h.indexService.GenerateIndex(asyncCtx, id, userID, preset); err != nil {
			fmt.Printf("[courseware_index_handler] 索引生成失败: courseware=%s err=%v\n", id, err)
		}
	}()

	utils.Success(w, map[string]interface{}{
		"message":       "课件索引生成已启动，请通过SSE监听进度",
		"courseware_id": id,
	})
}

// RefineIndex POST /api/v1/coursewares/{id}/refine-index — AI修改方案
func (h *CoursewareIndexHandler) RefineIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(r.URL.Path, "/refine-index")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	var req models.RefineIndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.Feedback == "" {
		utils.BadRequest(w, "修改意见不能为空")
		return
	}

	go func() {
		time.Sleep(800 * time.Millisecond)
		ctx := context.Background()
		if err := h.indexService.RefineIndex(ctx, id, claims.UserID, req.Feedback); err != nil {
			_ = err
		}
	}()

	utils.Success(w, map[string]string{
		"courseware_id": id,
		"message":      "AI修改方案已启动，请通过SSE接收进度",
	})
}

// ==================== v0.42 入口B: 从PPT内容生成索引 ====================

// GenerateIndexFromPPT POST /api/v1/coursewares/{id}/generate-index-ppt — 从PPT内容生成课件索引
func (h *CoursewareIndexHandler) GenerateIndexFromPPT(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(r.URL.Path, "/generate-index-ppt")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	// 检查PPT服务是否已注入
	if h.pptService == nil {
		utils.InternalError(w, "PPT解析服务未初始化")
		return
	}

	// 解析可选preset参数
	var reqBody struct {
		Preset string `json:"preset"`
	}
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	userID := claims.UserID
	preset := reqBody.Preset

	go func() {
		time.Sleep(800 * time.Millisecond)
		ctx := context.Background()
		if err := h.pptService.GenerateIndexFromPPT(ctx, id, userID, preset); err != nil {
			fmt.Printf("[courseware_index_handler] PPT索引生成失败: courseware=%s err=%v\n", id, err)
		}
	}()

	utils.Success(w, map[string]string{
		"courseware_id": id,
		"message":      "PPT课件方案生成已启动，请通过SSE接收进度",
	})
}

// ==================== v0.42 入口B: PPT上传创建课件 ====================

// CreateFromPPT POST /api/v1/coursewares/from-ppt — 上传PPT创建课件
// Content-Type: multipart/form-data
// 字段: file(.pptx) + subject + grade + title(可选)
func (h *CoursewareIndexHandler) CreateFromPPT(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	if h.pptService == nil {
		utils.InternalError(w, "PPT解析服务未初始化")
		return
	}

	// 解析multipart表单（最大52MB缓冲，略大于50MB文件限制）
	if err := r.ParseMultipartForm(52 << 20); err != nil {
		utils.BadRequest(w, "文件解析失败: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequest(w, "缺少文件字段 file")
		return
	}
	defer file.Close()

	subject := r.FormValue("subject")
	grade := r.FormValue("grade")
	title := r.FormValue("title")

	if subject == "" {
		utils.BadRequest(w, "学科不能为空")
		return
	}
	if grade == "" {
		utils.BadRequest(w, "年级不能为空")
		return
	}

	cw, extractResult, err := h.pptService.UploadAndCreateCourseware(
		r.Context(), claims.UserID, file, header, subject, grade, title,
	)
	if err != nil {
		utils.InternalError(w, "创建课件失败: "+err.Error())
		return
	}

	// 返回课件信息和PPT解析概要
	utils.Success(w, map[string]interface{}{
		"id":          cw.ID,
		"title":       cw.Title,
		"subject":     cw.Subject,
		"grade":       cw.Grade,
		"source_type": cw.SourceType,
		"slide_count": extractResult.SlideCount,
		"message":     fmt.Sprintf("PPT上传成功（%d页），课件已创建", extractResult.SlideCount),
	})
}

// ==================== v0.42 入口C: Word文档上传创建课件 ====================

// CreateFromDoc POST /api/v1/coursewares/from-doc — 上传Word文档创建课件
// Content-Type: multipart/form-data
// 字段: file(.docx) + subject + grade + title(可选)
func (h *CoursewareIndexHandler) CreateFromDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	if h.pptService == nil {
		utils.InternalError(w, "文档解析服务未初始化")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		utils.BadRequest(w, "文件解析失败: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequest(w, "缺少文件字段 file")
		return
	}
	defer file.Close()

	subject := r.FormValue("subject")
	grade := r.FormValue("grade")
	title := r.FormValue("title")

	if subject == "" {
		utils.BadRequest(w, "学科不能为空")
		return
	}
	if grade == "" {
		utils.BadRequest(w, "年级不能为空")
		return
	}

	cw, extractResult, err := h.pptService.UploadDocAndCreateCourseware(
		r.Context(), claims.UserID, file, header, subject, grade, title,
	)
	if err != nil {
		utils.InternalError(w, "创建课件失败: "+err.Error())
		return
	}

	utils.Success(w, map[string]interface{}{
		"id":          cw.ID,
		"title":       cw.Title,
		"subject":     cw.Subject,
		"grade":       cw.Grade,
		"source_type": cw.SourceType,
		"word_count":  extractResult.WordCount,
		"message":     fmt.Sprintf("文档上传成功（%d字），课件已创建", extractResult.WordCount),
	})
}

// GenerateIndexFromDoc POST /api/v1/coursewares/{id}/generate-index-doc — 从Word文档生成课件索引
func (h *CoursewareIndexHandler) GenerateIndexFromDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(r.URL.Path, "/generate-index-doc")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	if h.pptService == nil {
		utils.InternalError(w, "文档解析服务未初始化")
		return
	}

	var reqBody struct {
		Preset string `json:"preset"`
	}
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	userID := claims.UserID
	preset := reqBody.Preset

	go func() {
		time.Sleep(800 * time.Millisecond)
		ctx := context.Background()
		if err := h.pptService.GenerateIndexFromDoc(ctx, id, userID, preset); err != nil {
			fmt.Printf("[courseware_index_handler] Doc索引生成失败: courseware=%s err=%v\n", id, err)
		}
	}()

	utils.Success(w, map[string]string{
		"courseware_id": id,
		"message":      "教案文档课件方案生成已启动，请通过SSE接收进度",
	})
}


package handlers

// courseware_index_handler.go — 课件索引生成HTTP处理器 v4
//
// v4 (课件↔教案对齐报告) 变更:
//   - CoursewareIndexHandler 新增 alignmentService 字段 + SetAlignmentService 注入口
//   - 新增 GetAlignmentReport 端点（GET 查询对齐报告，前端 Step1 加载+短轮询用）
//   - 新增 RecheckAlignment 端点（POST 手动重算对齐，老师改完方案可主动重新校验）
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
//   6. GET  /api/v1/coursewares/{id}/alignment-report         — 查询课件↔教案对齐报告（v4）
//   7. POST /api/v1/coursewares/{id}/recheck-alignment        — 手动重算对齐报告（v4）

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== 课件索引处理器 ====================

// CoursewareIndexHandler 课件索引生成处理器
type CoursewareIndexHandler struct {
	indexService     *services.CoursewareIndexService
	cwService        *services.CoursewareService
	authService      *services.AuthService
	pptService       *services.CoursewarePPTService       // v0.42 入口B: PPT解析服务（可选注入）
	alignmentService *services.CoursewareAlignmentService // v4: 对齐校验服务（可选注入）
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

// SetAlignmentService v4: 注入对齐校验服务（在routes.go中调用）
func (h *CoursewareIndexHandler) SetAlignmentService(alignmentService *services.CoursewareAlignmentService) {
	h.alignmentService = alignmentService
}

// ==================== 触发索引生成 ====================

// GenerateIndex POST /api/v1/coursewares/{id}/generate-index — 触发AI生成课件索引
// 异步执行：立即返回200，通过SSE推送进度
// v2: goroutine启动前延迟800ms，确保前端SSE连接建立后再执行
func (h *CoursewareIndexHandler) GenerateIndex(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.GenerateIndexWithPresetTracked(w, r)
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

	finishSSEHandshake, handshakeOK := beginSSEHandshake(w)
	if !handshakeOK {
		return
	}
	defer finishSSEHandshake()

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

	finishSSEHandshake()

	writeCWSSEEvent(w, flusher, services.CWSSEConnected, map[string]string{
		"courseware_id": id,
		"message":       "SSE连接已建立",
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
func (h *CoursewareIndexHandler) DeletePage(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCoursewarePagePath(r.URL.Path)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	actor, err := authorizeCoursewareOwnerRuntimeForHandler(
		r.Context(),
		coursewareID,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	if err := h.cwService.DeletePageForActor(
		r.Context(),
		coursewareID,
		actor,
		pageNumber,
	); err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "页面删除成功",
		},
	)
}

// ==================== v4: 查询对齐报告 ====================

// GetAlignmentReport GET /api/v1/coursewares/{id}/alignment-report — 查询课件↔教案对齐报告
// 前端 Step1 加载时调用一次；若返回 status=generating 则前端短轮询直到 done/failed。
// 无报告（非教案来源/从未校验）返回 has_report=false，前端不显示对齐卡片。
func (h *CoursewareIndexHandler) GetAlignmentReport(
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

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/alignment-report",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	if _, err := h.cwService.LoadCoursewareForView(
		r.Context(),
		id,
		actor,
	); err != nil {
		handleCoursewareAccessError(
			w,
			err,
			"查询对齐报告失败",
		)
		return
	}

	report, err :=
		repository.GetAlignmentReportByCoursewareID(
			r.Context(),
			id,
		)
	if err != nil {
		utils.InternalError(
			w,
			"查询对齐报告失败: "+err.Error(),
		)
		return
	}

	if report == nil {
		utils.Success(
			w,
			models.AlignmentReportResponse{
				HasReport: false,
				Report:    nil,
			},
		)
		return
	}

	utils.Success(
		w,
		models.AlignmentReportResponse{
			HasReport: true,
			Report:    report,
		},
	)
}

// ==================== v4: 手动重算对齐报告 ====================

// RecheckAlignment POST /api/v1/coursewares/{id}/recheck-alignment — 手动触发对齐重算
// 老师改完方案后可主动重新校验。立即返回，校验异步进行；前端随后短轮询 GetAlignmentReport。
func (h *CoursewareIndexHandler) RecheckAlignment(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.RecheckAlignmentTracked(w, r)
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
func (h *CoursewareIndexHandler) GenerateIndexFromTopic(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.GenerateIndexFromTopicTracked(w, r)
}

// ==================== v136: AI修改方案+预设支持 ====================

// GenerateIndexWithPreset POST /api/v1/coursewares/{id}/generate-index — 带预设参数的索引生成
func (h *CoursewareIndexHandler) GenerateIndexWithPreset(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.GenerateIndexWithPresetTracked(w, r)
}

// RefineIndex POST /api/v1/coursewares/{id}/refine-index — AI修改方案
func (h *CoursewareIndexHandler) RefineIndex(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.RefineIndexTracked(w, r)
}

// ==================== v0.42 入口B: 从PPT内容生成索引 ====================

// GenerateIndexFromPPT POST /api/v1/coursewares/{id}/generate-index-ppt — 从PPT内容生成课件索引
func (h *CoursewareIndexHandler) GenerateIndexFromPPT(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.GenerateIndexFromPPTTracked(w, r)
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

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	if _, err := services.ResolveCoursewareCreationEducationDomain(actor); err != nil {
		utils.Fail(w, http.StatusForbidden, err.Error())
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
		r.Context(), actor, file, header, subject, grade, title,
	)
	if err != nil {
		utils.InternalError(w, "创建课件失败: "+err.Error())
		return
	}

	// 返回课件信息和PPT解析概要
	utils.Success(w, map[string]interface{}{
		"id":               cw.ID,
		"title":            cw.Title,
		"subject":          cw.Subject,
		"grade":            cw.Grade,
		"education_domain": cw.EducationDomain,
		"source_type":      cw.SourceType,
		"slide_count":      extractResult.SlideCount,
		"message":          fmt.Sprintf("PPT上传成功（%d页），课件已创建", extractResult.SlideCount),
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

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	if _, err := services.ResolveCoursewareCreationEducationDomain(actor); err != nil {
		utils.Fail(w, http.StatusForbidden, err.Error())
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
		r.Context(), actor, file, header, subject, grade, title,
	)
	if err != nil {
		utils.InternalError(w, "创建课件失败: "+err.Error())
		return
	}

	utils.Success(w, map[string]interface{}{
		"id":               cw.ID,
		"title":            cw.Title,
		"subject":          cw.Subject,
		"grade":            cw.Grade,
		"education_domain": cw.EducationDomain,
		"source_type":      cw.SourceType,
		"word_count":       extractResult.WordCount,
		"message":          fmt.Sprintf("文档上传成功（%d字），课件已创建", extractResult.WordCount),
	})
}

// GenerateIndexFromDoc POST /api/v1/coursewares/{id}/generate-index-doc — 从Word文档生成课件索引
func (h *CoursewareIndexHandler) GenerateIndexFromDoc(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.GenerateIndexFromDocTracked(w, r)
}

// ==================== 断裂B: 取课件关联教案正文（对照抽屉用） ====================

// GetLessonPlanContent GET /api/v1/coursewares/{id}/lesson-plan-content
// 返回课件关联教案的纯文本正文，供 Step4/Step5 工作台的"原教案对照抽屉"展示。
// 复用 services.ExtractLessonPlanContentForCW 的优先级链（content_markdown→
// conversation_log 最长assistant消息→ai_review_result→ai_review_history），
// 故对话生成型教案也能拿到正文（前端直接读 content_markdown 会落空）。
// 非教案来源 / 无关联教案：返回 has_lesson_plan=false，前端不显示抽屉入口。
func (h *CoursewareIndexHandler) GetLessonPlanContent(
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

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/lesson-plan-content",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	cw, err := h.cwService.LoadCoursewareForView(
		r.Context(),
		id,
		actor,
	)
	if err != nil {
		handleCoursewareAccessError(
			w,
			err,
			"获取课件原始内容失败",
		)
		return
	}

	// Word上传课件的完整原文属于课件内容的一部分。
	// 只有通过该课件统一查看权后才能返回。
	if cw.SourceType == models.CWSourceDocUpload {
		if docText :=
			services.ExtractDocUploadFullText(cw); docText != "" {
			utils.Success(
				w,
				map[string]interface{}{
					"has_lesson_plan": true,
					"title": cw.Title +
						"（上传文档原文）",
					"content": docText,
				},
			)
			return
		}
	}

	if cw.LessonPlanID == nil ||
		strings.TrimSpace(*cw.LessonPlanID) == "" {
		utils.Success(
			w,
			map[string]interface{}{
				"has_lesson_plan": false,
				"title":           "",
				"content":         "",
			},
		)
		return
	}

	lp, err := repository.GetLessonPlanByID(
		r.Context(),
		strings.TrimSpace(*cw.LessonPlanID),
	)
	if err != nil {
		utils.Success(
			w,
			map[string]interface{}{
				"has_lesson_plan": false,
				"title":           "",
				"content":         "",
			},
		)
		return
	}

	// 派生课件与来源教案必须属于完全相同的具体教学教育域。
	// 存量异常关联不能因为课件被共享而泄露其它域的教案正文。
	coursewareDomain := strings.ToLower(
		strings.TrimSpace(cw.EducationDomain),
	)
	lessonPlanDomain := strings.ToLower(
		strings.TrimSpace(lp.EducationDomain),
	)

	if !models.IsTeachingEducationDomain(
		coursewareDomain,
	) ||
		!models.IsTeachingEducationDomain(
			lessonPlanDomain,
		) ||
		coursewareDomain != lessonPlanDomain {
		utils.InternalError(
			w,
			"关联教案教育域异常，请联系管理员处理",
		)
		return
	}

	content :=
		services.ExtractLessonPlanContentForCW(lp)

	utils.Success(
		w,
		map[string]interface{}{
			"has_lesson_plan": true,
			"title":           lp.Title,
			"content":         content,
		},
	)
}

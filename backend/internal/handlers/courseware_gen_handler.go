package handlers

// courseware_gen_handler.go — 课件HTML生成HTTP处理器（Phase 4C P0-1~P0-5 + 单页重生 + 全自动装配）
//
// 提供接口：
//   1. POST /api/v1/coursewares/{id}/generate-preview        — 仅生成预览页（封面）
//   2. POST /api/v1/coursewares/{id}/save-nav-template        — 保存导航栏HTML模板
//   3. POST /api/v1/coursewares/{id}/generate-pages           — 用固定导航栏批量生成剩余页
//   4. POST /api/v1/coursewares/{id}/refine-nav                — 导航栏AI微调
//   5. POST /api/v1/coursewares/{id}/pages/{num}/refine        — 单页AI微调（支持可选截图多模态）
//   6. POST /api/v1/coursewares/{id}/pages/{num}/regenerate    — 单页重新生成（从方案从零重做）
//   7. POST /api/v1/coursewares/{id}/cancel-generate           — 中途中断生成
//   8. POST /api/v1/coursewares/{id}/generate-3d-page          — 3D互动单页生成
//   9. GET  /api/v1/coursewares/{id}/pages/{num}/versions            — 页面版本列表（轻量，不含html）
//  10. GET  /api/v1/coursewares/{id}/pages/{num}/versions/{versionId} — 取单个历史版本完整HTML（版本对比UI用）
//  11. POST /api/v1/coursewares/{id}/pages/{num}/rollback           — 回退到指定历史版本
//  12. POST /api/v1/coursewares/{id}/auto-assemble            — 全自动一键装配（HTML+配图+视频占位总装线）
//  13. POST /api/v1/coursewares/{id}/pages/{num}/save-html    — 【就地文字编辑】保存前端改过的整页HTML
//  14. POST /api/v1/coursewares/{id}/pages/{num}/import-html  — 【粘贴HTML建页·批次B】导入外部HTML到指定页

import (
        "context"
        "encoding/json"
        "fmt"
        "net/http"
        "strconv"
        "strings"

        "time"

        "tedna/internal/middleware"
        "tedna/internal/models"
        "tedna/internal/services"
        "tedna/internal/utils"
)

// ==================== 课件生成处理器 ====================

// CoursewareGenHandler 课件HTML生成处理器
type CoursewareGenHandler struct {
        genService *services.CoursewareGenService
        cwService  *services.CoursewareService
        // autoAssemblyService 全自动一键装配主编排服务（HTML生成+配图+视频占位总装线）
        autoAssemblyService *services.CoursewareAutoAssemblyService
}

// NewCoursewareGenHandler 创建课件HTML生成处理器
// autoAssemblyService：全自动装配服务，供 AutoAssemble 端点异步调用。
func NewCoursewareGenHandler(
        genService *services.CoursewareGenService,
        cwService *services.CoursewareService,
        autoAssemblyService *services.CoursewareAutoAssemblyService,
) *CoursewareGenHandler {
        return &CoursewareGenHandler{
                genService:          genService,
                cwService:           cwService,
                autoAssemblyService: autoAssemblyService,
        }
}

// ==================== Step 1: 生成预览页 ====================

// GeneratePreview POST /api/v1/coursewares/{id}/generate-preview
func (h *CoursewareGenHandler) GeneratePreview(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        id := extractCoursewareMiddleID(r.URL.Path, "/generate-preview")
        if id == "" {
                utils.BadRequest(w, "缺少课件ID")
                return
        }
        userID := claims.UserID
        go func() {
                // 延迟800ms等待前端SSE连接建立，避免立即失败时error事件丢失
                time.Sleep(800 * time.Millisecond)
                asyncCtx := context.Background()
                if err := h.genService.GeneratePreviewPages(asyncCtx, id, userID); err != nil {
                        fmt.Printf("[courseware_gen_handler] 预览页生成失败: courseware=%s err=%v\n", id, err)
                }
        }()
        utils.Success(w, map[string]interface{}{
                "message":       "预览页生成已启动，请通过SSE监听进度",
                "courseware_id": id,
        })
}

// ==================== Step 2: 保存导航栏模板 ====================

// SaveNavTemplate POST /api/v1/coursewares/{id}/save-nav-template
func (h *CoursewareGenHandler) SaveNavTemplate(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        id := extractCoursewareMiddleID(r.URL.Path, "/save-nav-template")
        if id == "" {
                utils.BadRequest(w, "缺少课件ID")
                return
        }
        var req models.SaveNavTemplateRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, "请求参数格式错误")
                return
        }
        if err := h.cwService.SaveNavTemplate(r.Context(), id, claims.UserID, req.NavTemplateHTML); err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        utils.Success(w, map[string]string{"message": "导航栏模板保存成功"})
}

// ==================== Step 3: 批量生成剩余页 ====================

// GeneratePages POST /api/v1/coursewares/{id}/generate-pages
func (h *CoursewareGenHandler) GeneratePages(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        id := extractCoursewareMiddleID(r.URL.Path, "/generate-pages")
        if id == "" {
                utils.BadRequest(w, "缺少课件ID")
                return
        }
        userID := claims.UserID
        go func() {
                // 延迟800ms等待前端SSE连接建立，避免立即失败时error事件丢失
                time.Sleep(800 * time.Millisecond)
                asyncCtx := context.Background()
                if err := h.genService.GenerateRemainingPages(asyncCtx, id, userID); err != nil {
                        fmt.Printf("[courseware_gen_handler] 课件生成失败: courseware=%s err=%v\n", id, err)
                }
        }()
        utils.Success(w, map[string]interface{}{
                "message":       "课件生成已启动（使用固定导航栏），请通过SSE监听进度",
                "courseware_id": id,
        })
}

// ==================== 全自动一键装配 ====================

// AutoAssemble POST /api/v1/coursewares/{id}/auto-assemble
// 全自动装配总装线：逐页HTML生成 + AI配图（生图→上云→融图）+ 视频首帧占位，一次交付图文齐备课件。
//
// 请求体（可选）: { "skip_video": true }
//   - skip_video 缺省/false = 全自动装配（HTML + 配图 + 视频首帧占位，视频按关键词命中页决定）
//   - skip_video = true      = 交付模式"HTML+配图不做视频"（中间档），所有页一律跳过视频占位
//     body 为空（老调用/不传）时 json.Decode 会得到零值 SkipVideo=false，等同全自动，向后兼容。
//
// 前置强约束：必须已设风格锚点（由 service 层 prepareAssembly 校验，未设则经SSE推 error 并中止）。
// 异步执行（照 GeneratePages 范式），进度经 SSE "assembly_*" 事件推送。
func (h *CoursewareGenHandler) AutoAssemble(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        id := extractCoursewareMiddleID(r.URL.Path, "/auto-assemble")
        if id == "" {
                utils.BadRequest(w, "缺少课件ID")
                return
        }

        // 解析可选请求体 skip_video（body 为空时 Decode 报 EOF，此处忽略以保持向后兼容——
        //   不传 body 的老调用等同 skip_video=false 走全自动）。
        var req struct {
                SkipVideo bool `json:"skip_video"`
        }
        if r.Body != nil {
                _ = json.NewDecoder(r.Body).Decode(&req) // 忽略解析错误：空body/格式问题一律回退默认 false
        }

        userID := claims.UserID
        skipVideo := req.SkipVideo
        go func() {
                // 延迟800ms等待前端SSE连接建立，避免立即失败时error/assembly事件丢失
                time.Sleep(800 * time.Millisecond)
                asyncCtx := context.Background()
                if err := h.autoAssemblyService.AutoAssemble(asyncCtx, id, userID, skipVideo); err != nil {
                        fmt.Printf("[courseware_gen_handler] 全自动装配失败: courseware=%s skip_video=%v err=%v\n", id, skipVideo, err)
                }
        }()
        utils.Success(w, map[string]interface{}{
                "message":       "全自动装配已启动，请通过SSE监听 assembly_* 进度事件",
                "courseware_id": id,
                "skip_video":    skipVideo,
        })
}

// ==================== 导航栏AI微调 ====================

// RefineNav POST /api/v1/coursewares/{id}/refine-nav
// 请求体: { "instruction": "Logo再大一点" }
// 同步返回微调后的导航栏HTML
func (h *CoursewareGenHandler) RefineNav(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        id := extractCoursewareMiddleID(r.URL.Path, "/refine-nav")
        if id == "" {
                utils.BadRequest(w, "缺少课件ID")
                return
        }
        var req struct {
                Instruction string `json:"instruction"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, "请求参数格式错误")
                return
        }
        if strings.TrimSpace(req.Instruction) == "" {
                utils.BadRequest(w, "修改意见不能为空")
                return
        }
        result, err := h.genService.RefineNav(r.Context(), id, claims.UserID, req.Instruction)
        if err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        utils.Success(w, map[string]interface{}{
                "nav_html": result,
                "message":  "导航栏微调完成",
        })
}

// ==================== 单页AI微调（支持可选截图多模态） ====================

// RefinePage POST /api/v1/coursewares/{id}/pages/{num}/refine
// 请求体: { "instruction": "标题字号再大一些", "image": "data:image/png;base64,xxx"(可选) }
// 截图非空时让AI看到该页实际渲染来定位版面问题；instruction与image至少一项非空。
func (h *CoursewareGenHandler) RefinePage(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        coursewareID, pageNum := extractCWPageRefinePath(r.URL.Path)
        if coursewareID == "" || pageNum <= 0 {
                utils.BadRequest(w, "路径格式错误")
                return
        }
        var req struct {
                Instruction string `json:"instruction"`
                Image       string `json:"image"` // 可选 base64 data URI: data:image/png;base64,xxx
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, "请求参数格式错误")
                return
        }
        instruction := strings.TrimSpace(req.Instruction)
        image := strings.TrimSpace(req.Image)
        if instruction == "" && image == "" {
                utils.BadRequest(w, "请提供修改意见或粘贴截图")
                return
        }
        // 截图校验
        if image != "" {
                if !strings.HasPrefix(image, "data:image/") {
                        utils.BadRequest(w, "截图格式无效，请直接粘贴图片")
                        return
                }
                const maxImageLen = 12 * 1024 * 1024 // base64 约对应 ≤8MB 原图
                if len(image) > maxImageLen {
                        utils.BadRequest(w, "截图过大，请压缩后重试（建议不超过8MB）")
                        return
                }
        }
        // 无文字仅截图时给默认指令
        if instruction == "" {
                instruction = "请参考截图，修复页面中存在的版面/排版问题（如内容出界、文字与图片重叠、被裁切、错位等），其余内容与样式保持不变。"
        }
        result, err := h.genService.RefinePage(r.Context(), coursewareID, claims.UserID, pageNum, instruction, image)
        if err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        utils.Success(w, map[string]interface{}{
                "page_number":  pageNum,
                "html_content": result,
                "message":      fmt.Sprintf("第%d页微调完成", pageNum),
        })
}

// ==================== 单页重新生成 ====================

// RegeneratePage POST /api/v1/coursewares/{id}/pages/{num}/regenerate
// 整页重做：依据页面方案从零重画内容区后拼接导航栏（不基于现有HTML），同步返回完整页面HTML。
func (h *CoursewareGenHandler) RegeneratePage(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        coursewareID, pageNum := extractCWPageRegeneratePath(r.URL.Path)
        if coursewareID == "" || pageNum <= 0 {
                utils.BadRequest(w, "路径格式错误")
                return
        }
        result, err := h.genService.RegenerateSinglePage(r.Context(), coursewareID, claims.UserID, pageNum)
        if err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        utils.Success(w, map[string]interface{}{
                "page_number":  pageNum,
                "html_content": result,
                "message":      fmt.Sprintf("第%d页重新生成完成", pageNum),
        })
}

// ==================== 就地文字编辑保存 ====================

// SavePageHTML POST /api/v1/coursewares/{id}/pages/{num}/save-html
// 【就地文字编辑】保存老师在预览 iframe 里就地改过的整页 HTML。
//
// 请求体: { "html_content": "<div class=\"cw-page\">...</div>" }
//
// 前端「✏️ 就地改文字」编辑器已在 iframe 内完成"只改文字/字号/颜色"的确定性编辑
// （纯 DOM 操作，不新增节点、不产生脏 DOM），并清理掉编辑器自身的高亮/脚本痕迹后回传整页纯净 HTML。
// 批次A起「✏️ 编辑源码」（Step5 源码视图直接编辑/整页替换）也复用本端点保存。
// 本端点不调 AI，只做"存旧版(manual) + 写新版"两步落库，与背景/字体秒换、回退同属确定性覆盖。
// 归属校验 + in_pipeline 拦截由 service 层 SaveManualEditedPage 完成。
func (h *CoursewareGenHandler) SavePageHTML(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        coursewareID, pageNum := extractCWPageSaveHTMLPath(r.URL.Path)
        if coursewareID == "" || pageNum <= 0 {
                utils.BadRequest(w, "路径格式错误")
                return
        }
        var req struct {
                HTMLContent string `json:"html_content"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, "请求参数格式错误")
                return
        }
        if strings.TrimSpace(req.HTMLContent) == "" {
                utils.BadRequest(w, "编辑后的内容为空，未保存")
                return
        }
        // 体量上限保护：单页 HTML 一般数十 KB，给 5MB 上限防异常超大 body
        const maxHTMLLen = 5 * 1024 * 1024
        if len(req.HTMLContent) > maxHTMLLen {
                utils.BadRequest(w, "页面内容过大，无法保存")
                return
        }
        result, err := h.genService.SaveManualEditedPage(r.Context(), coursewareID, claims.UserID, pageNum, req.HTMLContent)
        if err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        utils.Success(w, map[string]interface{}{
                "page_number":  pageNum,
                "html_content": result,
                "message":      fmt.Sprintf("第%d页修改已保存", pageNum),
        })
}

// ==================== 粘贴HTML导入（批次B） ====================

// ImportPageHTML POST /api/v1/coursewares/{id}/pages/{num}/import-html
// 【粘贴HTML建页】把老师粘贴的外部完整HTML导入指定页。
//
// 请求体: { "html_content": "<div style=\"width:1920px...\">...</div>" }
//
// 与 save-html（就地编辑/源码编辑保存）的区别：save-html 回传的是本课件生成的规范页、直接落库；
// 本端点接收的是外来HTML，service 层 ImportPageHTML（courseware_page_import.go）会做
// 画布契约归一(normalizeRootCanvas)、导航栏替换重编号(仅当代码自带NAV标记)、
// 背景幂等补注、覆盖前版本快照 后再落库并置 generated 状态。
// 归属校验(仅作者) + in_pipeline 拦截由 service 层完成。
func (h *CoursewareGenHandler) ImportPageHTML(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        coursewareID, pageNum := extractCWPageImportHTMLPath(r.URL.Path)
        if coursewareID == "" || pageNum <= 0 {
                utils.BadRequest(w, "路径格式错误")
                return
        }
        var req struct {
                HTMLContent string `json:"html_content"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, "请求参数格式错误")
                return
        }
        if strings.TrimSpace(req.HTMLContent) == "" {
                utils.BadRequest(w, "粘贴的内容为空，未导入")
                return
        }
        // 体量上限保护：与 save-html 同口径，5MB 上限防异常超大 body
        const maxHTMLLen = 5 * 1024 * 1024
        if len(req.HTMLContent) > maxHTMLLen {
                utils.BadRequest(w, "粘贴的内容过大，无法导入")
                return
        }
        result, err := h.genService.ImportPageHTML(r.Context(), coursewareID, claims.UserID, pageNum, req.HTMLContent)
        if err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        utils.Success(w, map[string]interface{}{
                "page_number":  pageNum,
                "html_content": result,
                "message":      fmt.Sprintf("第%d页HTML导入完成", pageNum),
        })
}

// ==================== v0.42.11: 3D互动单页生成 ====================

// Generate3DPage POST /api/v1/coursewares/{id}/generate-3d-page
// 一次性生成完整的3D互动HTML单页（Three.js + OrbitControls）
// 异步执行，通过SSE推送进度
func (h *CoursewareGenHandler) Generate3DPage(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        id := extractCoursewareMiddleID(r.URL.Path, "/generate-3d-page")
        if id == "" {
                utils.BadRequest(w, "缺少课件ID")
                return
        }
        userID := claims.UserID
        go func() {
                // 延迟800ms等待前端SSE连接建立
                time.Sleep(800 * time.Millisecond)
                asyncCtx := context.Background()
                if err := h.genService.Generate3DSinglePage(asyncCtx, id, userID); err != nil {
                        fmt.Printf("[courseware_gen_handler] 3D单页生成失败: courseware=%s err=%v\n", id, err)
                }
        }()
        utils.Success(w, map[string]interface{}{
                "message":       "3D互动单页生成已启动，请通过SSE监听进度",
                "courseware_id": id,
        })
}

// ==================== P0-5: 中途中断生成 ====================

// CancelGenerate POST /api/v1/coursewares/{id}/cancel-generate
func (h *CoursewareGenHandler) CancelGenerate(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        id := extractCoursewareMiddleID(r.URL.Path, "/cancel-generate")
        if id == "" {
                utils.BadRequest(w, "缺少课件ID")
                return
        }
        h.genService.CancelGenerate(id)
        utils.Success(w, map[string]string{
                "message":       "已发送停止信号",
                "courseware_id": id,
        })
}

// ==================== 路径解析辅助函数 ====================

// extractCWPageRefinePath 从 /api/v1/coursewares/{id}/pages/{num}/refine 提取课件ID和页码
func extractCWPageRefinePath(path string) (string, int) {
        return extractCWPageActionPath(path, "/refine")
}

// extractCWPageRegeneratePath 从 /api/v1/coursewares/{id}/pages/{num}/regenerate 提取课件ID和页码
func extractCWPageRegeneratePath(path string) (string, int) {
        return extractCWPageActionPath(path, "/regenerate")
}

// extractCWPageSaveHTMLPath 从 /api/v1/coursewares/{id}/pages/{num}/save-html 提取课件ID和页码
func extractCWPageSaveHTMLPath(path string) (string, int) {
        return extractCWPageActionPath(path, "/save-html")
}

// extractCWPageImportHTMLPath 从 /api/v1/coursewares/{id}/pages/{num}/import-html 提取课件ID和页码（批次B）
func extractCWPageImportHTMLPath(path string) (string, int) {
        return extractCWPageActionPath(path, "/import-html")
}

// extractCWPageActionPath 从 /api/v1/coursewares/{id}/pages/{num}/{action} 提取课件ID和页码
// action 形如 "/refine" / "/regenerate"
func extractCWPageActionPath(path string, action string) (string, int) {
        if !strings.HasSuffix(path, action) {
                return "", 0
        }
        trimmed := strings.TrimSuffix(path, action)
        pagesIdx := strings.LastIndex(trimmed, "/pages/")
        if pagesIdx < 0 {
                return "", 0
        }
        numStr := trimmed[pagesIdx+len("/pages/"):]
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

// ==================== 页面级版本与回退 ====================

// ListPageVersions GET /api/v1/coursewares/{id}/pages/{num}/versions
// 返回该页的版本列表（按 version_no 倒序，最新在前；轻量，不含 html_content）。
// 每条附来源中文标签 source_label，前端直接显示。
func (h *CoursewareGenHandler) ListPageVersions(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        coursewareID, pageNum := extractCWPageVersionsPath(r.URL.Path)
        if coursewareID == "" || pageNum <= 0 {
                utils.BadRequest(w, "路径格式错误")
                return
        }
        items, err := h.genService.ListCWPageVersions(r.Context(), coursewareID, claims.UserID, pageNum)
        if err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        // 组装返回：版本基础字段 + 来源中文标签
        list := make([]map[string]interface{}, 0, len(items))
        for _, it := range items {
                label := models.CWPageVersionSourceNameMap[it.Source]
                if label == "" {
                        label = it.Source
                }
                list = append(list, map[string]interface{}{
                        "id":           it.ID,
                        "version_no":   it.VersionNo,
                        "source":       it.Source,
                        "source_label": label,
                        "note":         it.Note,
                        "created_at":   it.CreatedAt,
                })
        }
        utils.Success(w, map[string]interface{}{
                "page_number": pageNum,
                "versions":    list,
                "total":       len(list),
        })
}

// GetPageVersionDetail GET /api/v1/coursewares/{id}/pages/{num}/versions/{versionId}
// 取某个历史版本的完整 HTML（版本对比UI用，只读，不改任何状态）。
// 前端对比弹窗左侧渲染此历史版 HTML，右侧渲染当前版（当前版前端已有，无需再拉）。
func (h *CoursewareGenHandler) GetPageVersionDetail(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        coursewareID, pageNum, versionID := extractCWPageVersionDetailPath(r.URL.Path)
        if coursewareID == "" || pageNum <= 0 || versionID == "" {
                utils.BadRequest(w, "路径格式错误")
                return
        }
        html, versionNo, source, err := h.genService.GetCWPageVersionHTML(r.Context(), coursewareID, claims.UserID, pageNum, versionID)
        if err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        // 附带来源中文标签，供弹窗标题展示"正在对比 v3 · 微调前"
        label := models.CWPageVersionSourceNameMap[source]
        if label == "" {
                label = source
        }
        utils.Success(w, map[string]interface{}{
                "page_number":  pageNum,
                "version_id":   versionID,
                "version_no":   versionNo,
                "source":       source,
                "source_label": label,
                "html_content": html,
        })
}

// RollbackPage POST /api/v1/coursewares/{id}/pages/{num}/rollback
// 请求体: { "version_id": "xxx" }
// 回退该页到指定历史版本，返回回退后的完整 HTML。回退前会把当前版本另存为 rollback 版（可逆）。
func (h *CoursewareGenHandler) RollbackPage(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
                return
        }
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        coursewareID, pageNum := extractCWPageRollbackPath(r.URL.Path)
        if coursewareID == "" || pageNum <= 0 {
                utils.BadRequest(w, "路径格式错误")
                return
        }
        var req struct {
                VersionID string `json:"version_id"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, "请求参数格式错误")
                return
        }
        if strings.TrimSpace(req.VersionID) == "" {
                utils.BadRequest(w, "缺少目标版本ID")
                return
        }
        result, err := h.genService.RollbackCWPage(r.Context(), coursewareID, claims.UserID, pageNum, strings.TrimSpace(req.VersionID))
        if err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        utils.Success(w, map[string]interface{}{
                "page_number":  pageNum,
                "html_content": result,
                "message":      fmt.Sprintf("第%d页已回退", pageNum),
        })
}

// extractCWPageVersionsPath 从 /api/v1/coursewares/{id}/pages/{num}/versions 提取课件ID和页码
func extractCWPageVersionsPath(path string) (string, int) {
        return extractCWPageActionPath(path, "/versions")
}

// extractCWPageRollbackPath 从 /api/v1/coursewares/{id}/pages/{num}/rollback 提取课件ID和页码
func extractCWPageRollbackPath(path string) (string, int) {
        return extractCWPageActionPath(path, "/rollback")
}

// extractCWPageVersionDetailPath 从 /api/v1/coursewares/{id}/pages/{num}/versions/{versionId}
// 提取课件ID、页码、版本ID三段。
//
// 解析思路：先定位 "/versions/" 分隔——其后是 versionId，其前是 "/api/v1/coursewares/{id}/pages/{num}"。
// 再对前半段复用 "/pages/" 拆出 coursewareID 与 pageNum。
func extractCWPageVersionDetailPath(path string) (coursewareID string, pageNum int, versionID string) {
        marker := "/versions/"
        vIdx := strings.LastIndex(path, marker)
        if vIdx < 0 {
                return "", 0, ""
        }
        // versionId = "/versions/" 之后的部分（去掉可能的尾斜杠）
        versionID = strings.TrimSuffix(path[vIdx+len(marker):], "/")
        if versionID == "" {
                return "", 0, ""
        }
        // 前半段形如 /api/v1/coursewares/{id}/pages/{num}
        front := path[:vIdx]
        pagesIdx := strings.LastIndex(front, "/pages/")
        if pagesIdx < 0 {
                return "", 0, ""
        }
        numStr := front[pagesIdx+len("/pages/"):]
        num, err := strconv.Atoi(numStr)
        if err != nil || num <= 0 {
                return "", 0, ""
        }
        prefix := front[:pagesIdx]
        cwPrefix := "/api/v1/coursewares/"
        if !strings.HasPrefix(prefix, cwPrefix) {
                return "", 0, ""
        }
        cwID := prefix[len(cwPrefix):]
        if cwID == "" {
                return "", 0, ""
        }
        return cwID, num, versionID
}

package handlers

// courseware_snippet_handler.go — 【代码收藏库·批次C新增】代码收藏HTTP处理器
//
// 提供接口（路由前缀 /api/v1/code-snippets，登录即可，归属校验在本 handler 内完成）：
//   1. GET    /api/v1/code-snippets        — 我的收藏列表（轻量，不含HTML全文，附字节数）
//   2. POST   /api/v1/code-snippets        — 收藏某课件某页的当前HTML（服务端自取页面内容做快照）
//   3. GET    /api/v1/code-snippets/{id}   — 取单条收藏完整HTML（预览/注入微调用）
//   4. DELETE /api/v1/code-snippets/{id}   — 删除收藏
//
// 分层说明：收藏是纯 CRUD + 简单归属校验，无业务编排/不调AI，故 handler 直接调
// repository（镜像 CoursewareBackgroundHandler 的轻量集合处理器范式），不另设 service 层。
//
// 收藏权限口径：仅课件作者本人可收藏自己课件的页（与就地编辑/源码编辑同口径）。
// 老师想收藏共享库里别人的好页面时，走既有路径：复制源码 → 粘贴HTML建页（批次B）→ 收藏，
// 产权边界由共享库的 code_share_scope 把守，本处理器不做二次裁决。

import (
        "encoding/json"
        "fmt"
        "net/http"
        "strings"

        "tedna/internal/middleware"
        "tedna/internal/repository"
        "tedna/internal/utils"
)

// CoursewareSnippetHandler 代码收藏处理器（无状态，直接调 repository）
type CoursewareSnippetHandler struct{}

// NewCoursewareSnippetHandler 创建代码收藏处理器（无参构造，镜像 bgHandler 范式）
func NewCoursewareSnippetHandler() *CoursewareSnippetHandler {
        return &CoursewareSnippetHandler{}
}

// HandleCollection 集合级路由分发：GET=列表 / POST=收藏
// 挂载路径 /api/v1/code-snippets（不带尾段）
func (h *CoursewareSnippetHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:
                h.listSnippets(w, r)
        case http.MethodPost:
                h.createSnippet(w, r)
        default:
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET/POST请求")
        }
}

// HandleItem 单条级路由分发：GET=完整详情 / DELETE=删除
// 挂载路径 /api/v1/code-snippets/{id}
func (h *CoursewareSnippetHandler) HandleItem(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:
                h.getSnippet(w, r)
        case http.MethodDelete:
                h.deleteSnippet(w, r)
        default:
                utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET/DELETE请求")
        }
}

// listSnippets GET /api/v1/code-snippets — 当前用户的收藏列表（倒序，轻量）
func (h *CoursewareSnippetHandler) listSnippets(w http.ResponseWriter, r *http.Request) {
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        list, err := repository.ListCodeSnippetsByUser(r.Context(), claims.UserID)
        if err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        utils.Success(w, map[string]interface{}{
                "snippets": list,
                "total":    len(list),
        })
}

// createSnippet POST /api/v1/code-snippets — 收藏某课件某页的当前HTML
//
// 请求体: { "courseware_id": "xxx", "page_number": 3, "title": "对比卡片布局", "note": "左右对比排版很清爽"(可选) }
//
// 服务端自取页面当前 HTML 做快照（前端不传大HTML，省带宽且保证快照与库内一致）。
// 归属校验：课件必须属于当前用户（仅作者可收藏自己的页）。
func (h *CoursewareSnippetHandler) createSnippet(w http.ResponseWriter, r *http.Request) {
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        var req struct {
                CoursewareID string `json:"courseware_id"`
                PageNumber   int    `json:"page_number"`
                Title        string `json:"title"`
                Note         string `json:"note"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, "请求参数格式错误")
                return
        }
        title := strings.TrimSpace(req.Title)
        note := strings.TrimSpace(req.Note)
        if req.CoursewareID == "" || req.PageNumber <= 0 {
                utils.BadRequest(w, "缺少课件ID或页码")
                return
        }
        if title == "" {
                utils.BadRequest(w, "请给这条收藏起个名称")
                return
        }
        // 长度限制：名称200字符、备注2000字符（rune计，防中文被字节截断误判）
        if len([]rune(title)) > 200 {
                utils.BadRequest(w, "收藏名称过长（最多200字）")
                return
        }
        if len([]rune(note)) > 2000 {
                utils.BadRequest(w, "备注过长（最多2000字）")
                return
        }

        // 归属校验：仅作者本人可收藏自己课件的页
        cw, err := repository.GetCoursewareByID(r.Context(), req.CoursewareID)
        if err != nil {
                utils.BadRequest(w, "课件不存在")
                return
        }
        if cw.UserID != claims.UserID {
                utils.Fail(w, http.StatusForbidden, "只能收藏自己课件的页面")
                return
        }

        // 取页面当前HTML做快照
        page, err := repository.GetCoursewarePageByNumber(r.Context(), req.CoursewareID, req.PageNumber)
        if err != nil {
                utils.BadRequest(w, "页面不存在")
                return
        }
        if strings.TrimSpace(page.HTMLContent) == "" {
                utils.BadRequest(w, "该页尚未生成HTML，无法收藏")
                return
        }

        s, err := repository.CreateCodeSnippet(r.Context(), claims.UserID, title, note,
                page.HTMLContent, req.CoursewareID, req.PageNumber)
        if err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        utils.Success(w, map[string]interface{}{
                "id":         s.ID,
                "title":      s.Title,
                "created_at": s.CreatedAt,
                "message":    fmt.Sprintf("已收藏「%s」到我的代码库", s.Title),
        })
}

// getSnippet GET /api/v1/code-snippets/{id} — 取单条收藏完整HTML（预览/注入微调用）
func (h *CoursewareSnippetHandler) getSnippet(w http.ResponseWriter, r *http.Request) {
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        id := extractSnippetID(r.URL.Path)
        if id == "" {
                utils.BadRequest(w, "缺少收藏ID")
                return
        }
        s, err := repository.GetCodeSnippet(r.Context(), id)
        if err != nil {
                utils.BadRequest(w, "收藏不存在")
                return
        }
        // 归属校验：只能看自己的收藏
        if s.UserID != claims.UserID {
                utils.Fail(w, http.StatusForbidden, "无权查看此收藏")
                return
        }
        utils.Success(w, s)
}

// deleteSnippet DELETE /api/v1/code-snippets/{id} — 删除收藏（repo 层双条件天然防越权）
func (h *CoursewareSnippetHandler) deleteSnippet(w http.ResponseWriter, r *http.Request) {
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims == nil {
                utils.Unauthorized(w, "未登录")
                return
        }
        id := extractSnippetID(r.URL.Path)
        if id == "" {
                utils.BadRequest(w, "缺少收藏ID")
                return
        }
        deleted, err := repository.DeleteCodeSnippet(r.Context(), id, claims.UserID)
        if err != nil {
                utils.InternalError(w, err.Error())
                return
        }
        if !deleted {
                utils.BadRequest(w, "收藏不存在或已删除")
                return
        }
        utils.Success(w, map[string]string{"message": "收藏已删除"})
}

// extractSnippetID 从 /api/v1/code-snippets/{id} 提取收藏ID（去掉可能的尾斜杠）
func extractSnippetID(path string) string {
        prefix := "/api/v1/code-snippets/"
        if !strings.HasPrefix(path, prefix) {
                return ""
        }
        return strings.TrimSuffix(path[len(prefix):], "/")
}

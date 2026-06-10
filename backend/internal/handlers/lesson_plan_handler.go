package handlers

// 教案管理HTTP处理器
// 负责教案CRUD、状态流转、评审、Fork、提示词模板管理的HTTP接口
//
// v168改动（第二批治本·功能A·正文产出硬门控 切片A）：
//   handleLPError 新增 ErrLPContentEmpty 分支，归入 400 BadRequest。
//   service 层 SubmitForReview/PublishPersonal/PublishShared 在教案正文为空时
//   返回该错误，handler 需将其映射为 400 + 明确文案，前端 catch 后可直接展示，
//   而非落入 default 分支被吞成 500「操作失败，请稍后重试」。
//   语义上"正文为空"是请求内容不合格（400），区别于"状态不允许"（403）。
//
// 迭代一 Phase 4 改动（数据隔离收口）：
//   ListLessonPlans 端点原先完全不取 claims，所有筛选参数都来自 URL query——
//   任何登录用户只要不传 author_id 就能列出全库教案，是数据越权的裸洞。
//   本次改为：从 JWT claims 取出 role+userID，调 services.ResolveDataScope 解析出
//   "该请求者能看到哪些教案"的数据范围（DataScope），再传给 service 层。
//   可见性规则（本人 ∪ 管辖成员 ∪ 共享可见）的实际过滤在 repo 层 SQL 拼接。
//   未取到 claims（理论上 authMW 已拦截未登录请求）时按未认证 401 处理。
//
// 迭代一 Phase 4 收尾改动（共享发布越权封堵）：
//   handleLPError 新增 ErrLPNotPublisher 分支，归入 403 Forbidden。
//   service 层 PublishShared 在调用者既非作者本人、也非 admin 时返回该错误，
//   语义上属于"权限不足"（与 ErrLPNotAuthor 同类），故映射 403 而非 400/500。

import (
        "encoding/json"
        "errors"
        "log"
        "net/http"
        "strconv"
        "strings"

        "tedna/internal/middleware"
        "tedna/internal/models"
        "tedna/internal/services"
        "tedna/internal/utils"
)

// LessonPlanHandler 教案管理接口处理器
type LessonPlanHandler struct {
        lpService *services.LessonPlanService
}

// NewLessonPlanHandler 创建教案管理处理器实例
func NewLessonPlanHandler(lpService *services.LessonPlanService) *LessonPlanHandler {
        return &LessonPlanHandler{lpService: lpService}
}

// ==================== 教案列表 ====================

// ListLessonPlans 教案列表端点
//
// 迭代一 Phase 4（数据隔离收口）：取 claims 解析数据范围后再查询。
//   - 取不到 claims → 401（authMW 通常已挡，这里是双保险）。
//   - 取到 claims → ResolveDataScope(role,userID) 得 DataScope，传给 service。
//     admin 看全部；senior/region_admin 看管辖成员；operator/viewer 看本人；
//     此外任何登录用户都能看到 published_shared/approved 的共享教案（教案库浏览）。
func (h *LessonPlanHandler) ListLessonPlans(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
                return
        }

        // Phase 4：取 JWT claims，解析当前请求者的数据可见范围
        claims, ok := middleware.GetClaims(r.Context())
        if !ok || claims.UserID == "" {
                utils.Unauthorized(w, utils.MsgNotLoggedIn)
                return
        }
        scope := services.ResolveDataScope(r.Context(), claims.Role, claims.UserID)

        q := r.URL.Query()
        authorID := q.Get("author_id")
        groupID := q.Get("group_id")
        status := q.Get("status")
        subject := q.Get("subject")
        grade := q.Get("grade")
        limit, _ := strconv.Atoi(q.Get("limit"))
        offset, _ := strconv.Atoi(q.Get("offset"))
        qualityLevel, _ := strconv.Atoi(q.Get("quality_level"))
        structureType, _ := strconv.Atoi(q.Get("structure_type"))
        cognitiveLevel, _ := strconv.Atoi(q.Get("cognitive_level"))
        pedagogyIntensity, _ := strconv.Atoi(q.Get("pedagogy_intensity"))

        result, err := h.lpService.ListLessonPlans(r.Context(), authorID, groupID, status, subject, grade, limit, offset, qualityLevel, structureType, cognitiveLevel, pedagogyIntensity, &scope)
        if err != nil {
                log.Printf("获取教案列表失败: %v", err)
                utils.InternalError(w, "获取教案列表失败")
                return
        }
        utils.Success(w, result)
}

// ==================== 创建教案 ====================

func (h *LessonPlanHandler) CreateLessonPlan(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
                return
        }
        userID := getCurrentUserID(r)
        if userID == "" {
                utils.Unauthorized(w, utils.MsgNotLoggedIn)
                return
        }
        var req models.CreateLessonPlanRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, utils.MsgBadRequestBody)
                return
        }
        lp, err := h.lpService.CreateLessonPlan(r.Context(), &req, userID)
        if err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, lp)
}

// ==================== 获取教案详情 ====================

func (h *LessonPlanHandler) GetLessonPlan(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
                return
        }
        id := extractLPID(r.URL.Path)
        if id == "" {
                utils.BadRequest(w, utils.MsgMissingLessonPlanID)
                return
        }
        detail, err := h.lpService.GetLessonPlan(r.Context(), id)
        if err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, detail)
}

// ==================== 更新教案 ====================

func (h *LessonPlanHandler) UpdateLessonPlan(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPut {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
                return
        }
        id := extractLPID(r.URL.Path)
        if id == "" {
                utils.BadRequest(w, utils.MsgMissingLessonPlanID)
                return
        }
        userID := getCurrentUserID(r)
        var req models.UpdateLessonPlanRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, utils.MsgBadRequestBody)
                return
        }
        if err := h.lpService.UpdateLessonPlan(r.Context(), id, userID, &req); err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, map[string]string{"message": "更新成功"})
}

// ==================== 删除教案 ====================

func (h *LessonPlanHandler) DeleteLessonPlan(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodDelete {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
                return
        }
        id := extractLPID(r.URL.Path)
        if id == "" {
                utils.BadRequest(w, utils.MsgMissingLessonPlanID)
                return
        }
        userID := getCurrentUserID(r)
        if err := h.lpService.DeleteLessonPlan(r.Context(), id, userID); err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 教案状态操作 ====================

func (h *LessonPlanHandler) PublishPersonal(w http.ResponseWriter, r *http.Request) {
        id := extractLPMiddleID(r.URL.Path, "/publish-personal")
        if id == "" {
                utils.BadRequest(w, utils.MsgMissingLessonPlanID)
                return
        }
        userID := getCurrentUserID(r)
        if err := h.lpService.PublishPersonal(r.Context(), id, userID); err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, map[string]string{"message": "个人发布成功"})
}

func (h *LessonPlanHandler) SubmitForReview(w http.ResponseWriter, r *http.Request) {
        id := extractLPMiddleID(r.URL.Path, "/submit-review")
        if id == "" {
                utils.BadRequest(w, utils.MsgMissingLessonPlanID)
                return
        }
        userID := getCurrentUserID(r)
        var req models.SubmitLessonPlanReviewRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, utils.MsgBadRequestBody)
                return
        }
        if err := h.lpService.SubmitForReview(r.Context(), id, userID, req.GroupID); err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, map[string]string{"message": "已提交评审"})
}

func (h *LessonPlanHandler) ReviewLessonPlan(w http.ResponseWriter, r *http.Request) {
        id := extractLPMiddleID(r.URL.Path, "/review")
        if id == "" {
                utils.BadRequest(w, utils.MsgMissingLessonPlanID)
                return
        }
        userID := getCurrentUserID(r)
        var req models.CreateLessonPlanReviewRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, utils.MsgBadRequestBody)
                return
        }
        if err := h.lpService.ReviewLessonPlan(r.Context(), id, userID, &req); err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, map[string]string{"message": "评审完成"})
}

func (h *LessonPlanHandler) PublishShared(w http.ResponseWriter, r *http.Request) {
        id := extractLPMiddleID(r.URL.Path, "/publish-shared")
        if id == "" {
                utils.BadRequest(w, utils.MsgMissingLessonPlanID)
                return
        }
        userID := getCurrentUserID(r)
        if err := h.lpService.PublishShared(r.Context(), id, userID); err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, map[string]string{"message": "共享发布成功"})
}

func (h *LessonPlanHandler) StartDevelopment(w http.ResponseWriter, r *http.Request) {
        id := extractLPMiddleID(r.URL.Path, "/start-development")
        if id == "" {
                utils.BadRequest(w, utils.MsgMissingLessonPlanID)
                return
        }
        userID := getCurrentUserID(r)
        result, err := h.lpService.StartDevelopment(r.Context(), id, userID)
        if err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, result)
}

func (h *LessonPlanHandler) ForkLessonPlan(w http.ResponseWriter, r *http.Request) {
        id := extractLPMiddleID(r.URL.Path, "/fork")
        if id == "" {
                utils.BadRequest(w, utils.MsgMissingLessonPlanID)
                return
        }
        userID := getCurrentUserID(r)
        newLP, err := h.lpService.ForkLessonPlan(r.Context(), id, userID)
        if err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, newLP)
}

// ==================== 提示词模板管理 ====================

func (h *LessonPlanHandler) ListPromptTemplates(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
                return
        }
        q := r.URL.Query()
        level := q.Get("level")
        ownerID := q.Get("owner_id")
        result, err := h.lpService.ListPromptTemplates(r.Context(), level, ownerID)
        if err != nil {
                log.Printf("获取模板列表失败: %v", err)
                utils.InternalError(w, "获取模板列表失败")
                return
        }
        utils.Success(w, result)
}

func (h *LessonPlanHandler) CreatePromptTemplate(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
                return
        }
        userID := getCurrentUserID(r)
        var req models.CreatePromptTemplateRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, utils.MsgBadRequestBody)
                return
        }
        pt, err := h.lpService.CreatePromptTemplate(r.Context(), &req, userID)
        if err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, pt)
}

func (h *LessonPlanHandler) GetPromptTemplate(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
                return
        }
        id := extractTemplateID(r.URL.Path)
        if id == "" {
                utils.BadRequest(w, "缺少模板ID")
                return
        }
        pt, err := h.lpService.GetPromptTemplate(r.Context(), id)
        if err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, pt)
}

func (h *LessonPlanHandler) UpdatePromptTemplate(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPut {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
                return
        }
        id := extractTemplateID(r.URL.Path)
        if id == "" {
                utils.BadRequest(w, "缺少模板ID")
                return
        }
        var req models.UpdatePromptTemplateRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, utils.MsgBadRequestBody)
                return
        }
        if err := h.lpService.UpdatePromptTemplate(r.Context(), id, &req); err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, map[string]string{"message": "更新成功"})
}

func (h *LessonPlanHandler) ResolvePromptTemplate(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
                return
        }
        prefix := "/api/v1/lesson-plans/templates/"
        suffix := "/resolved"
        id := extractMiddleSegment(r.URL.Path, prefix, suffix)
        if id == "" {
                utils.BadRequest(w, "缺少模板ID")
                return
        }
        resolved, err := h.lpService.ResolvePromptTemplate(r.Context(), id)
        if err != nil {
                h.handleLPError(w, err)
                return
        }
        utils.Success(w, resolved)
}

// ==================== 错误处理 ====================

func (h *LessonPlanHandler) handleLPError(w http.ResponseWriter, err error) {
        switch {
        case errors.Is(err, services.ErrLPTitleRequired),
                errors.Is(err, services.ErrLPSubjectRequired),
                errors.Is(err, services.ErrLPGradeRequired),
                errors.Is(err, services.ErrLPTopicRequired),
                errors.Is(err, services.ErrLPGroupRequired),
                errors.Is(err, services.ErrLPContentEmpty),
                errors.Is(err, services.ErrTemplateNameRequired),
                errors.Is(err, services.ErrTemplateLevelInvalid):
                // v168：ErrLPContentEmpty 归入 400——"教案正文为空"是请求内容不合格，
                // 前端 catch 后直接展示该明确文案，引导用户去备课工坊补全正文。
                utils.BadRequest(w, err.Error())
        case errors.Is(err, services.ErrLPNotAuthor),
                errors.Is(err, services.ErrLPNotPublisher),
                errors.Is(err, services.ErrLPCannotEdit),
                errors.Is(err, services.ErrLPCannotSubmit),
                errors.Is(err, services.ErrLPCannotDevelop),
                errors.Is(err, services.ErrLPAlreadyDeveloping):
                // Phase 4 收尾：ErrLPNotPublisher 归入 403——"非作者非管理员不能共享发布"
                // 属于权限不足，与 ErrLPNotAuthor 同类。
                utils.Fail(w, http.StatusForbidden, err.Error())
        case errors.Is(err, services.ErrLPNotFound),
                errors.Is(err, services.ErrTemplateNotFound):
                utils.Fail(w, http.StatusNotFound, err.Error())
        default:
                log.Printf("教案操作失败: %v", err)
                utils.InternalError(w, "操作失败，请稍后重试")
        }
}

// ==================== 辅助函数 ====================

func extractLPID(path string) string {
        prefix := "/api/v1/lesson-plans/plans/"
        if !strings.HasPrefix(path, prefix) {
                return ""
        }
        id := strings.TrimPrefix(path, prefix)
        id = strings.TrimSuffix(id, "/")
        if idx := strings.Index(id, "/"); idx > 0 {
                id = id[:idx]
        }
        return id
}

func extractLPMiddleID(path string, suffix string) string {
        prefix := "/api/v1/lesson-plans/plans/"
        return extractMiddleSegment(path, prefix, suffix)
}

func extractTemplateID(path string) string {
        prefix := "/api/v1/lesson-plans/templates/"
        if !strings.HasPrefix(path, prefix) {
                return ""
        }
        id := strings.TrimPrefix(path, prefix)
        id = strings.TrimSuffix(id, "/")
        if idx := strings.Index(id, "/"); idx > 0 {
                id = id[:idx]
        }
        return id
}

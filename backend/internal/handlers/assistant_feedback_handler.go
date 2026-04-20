package handlers

// assistant_feedback_handler.go — AI 助手反馈 HTTP 处理器
//
// 路由:
//   POST   /api/v1/assistant-feedback                    创建反馈(登录用户均可)
//   DELETE /api/v1/assistant-feedback/{id}               删除自己的反馈
//   GET    /api/v1/assistant-feedback                    列表(admin only,含筛选+分页)
//   GET    /api/v1/assistants/{id}/feedback-stats        某助手的反馈统计
//
// 权限策略:
//   - 创建反馈:任何已登录用户(老师主动行为)
//   - 删除反馈:只能删自己的(service 层校验 user_id)
//   - 列表查询:仅 admin(routes.go 中用 adminOnly 中间件)
//   - 统计查询:任何已登录用户(看看自己/同事觉得某助手怎么样,完全公开)

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 处理器 ====================

// AssistantFeedbackHandler AI 助手反馈处理器(无依赖 service,直接调 repository)
//
// 设计取舍:反馈逻辑简单(写入+删除+查询),没有复杂业务规则,不单独建 service 层
// 只有"删除前校验所有权"这一点权限逻辑,放在 handler 内即可
type AssistantFeedbackHandler struct{}

// NewAssistantFeedbackHandler 构造函数
func NewAssistantFeedbackHandler() *AssistantFeedbackHandler {
	return &AssistantFeedbackHandler{}
}

// ==================== 路径解析辅助 ====================

const feedbackPathPrefix = "/api/v1/assistant-feedback/"

// extractFeedbackID 从 path 中解析 {id}
func extractFeedbackID(path string) string {
	rest := strings.TrimPrefix(path, feedbackPathPrefix)
	rest = strings.TrimSuffix(rest, "/")
	if rest == "" {
		return ""
	}
	parts := strings.Split(rest, "/")
	return parts[0]
}

// ==================== 1. POST /assistant-feedback 创建反馈 ====================

// Create 创建反馈
//
// 请求体:CreateFeedbackRequest
// 响应:反馈实体(含 id 和 created_at)
//
// 校验:
//   - rating 必须是 up/down
//   - scene_code 不能为空
//   - assistant_id 不能为空(外键约束会拦住不存在的 id,这里只做格式校验)
func (h *AssistantFeedbackHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var req models.CreateFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	// 必填字段校验
	if strings.TrimSpace(req.AssistantID) == "" {
		utils.BadRequest(w, "缺少 assistant_id")
		return
	}
	if !models.IsValidFeedbackRating(req.Rating) {
		utils.BadRequest(w, "rating 必须是 up 或 down")
		return
	}
	if strings.TrimSpace(req.SceneCode) == "" {
		utils.BadRequest(w, "缺少 scene_code")
		return
	}

	// 轻量清洗:comment 空字符串视为 nil(数据库保持 NULL 语义)
	if req.Comment != nil && strings.TrimSpace(*req.Comment) == "" {
		req.Comment = nil
	}

	// ai_response_preview 截断至 300 字(与数据库约束对齐,按 rune 计数避免中文截半)
	if req.AIResponsePreview != nil {
		truncated := utils.SafeTruncate(*req.AIResponsePreview, 300)
		req.AIResponsePreview = &truncated
	}

	// 组装实体
	feedback := &models.AssistantFeedback{
		AssistantID:       req.AssistantID,
		UserID:            claims.UserID,
		Rating:            req.Rating,
		Comment:           req.Comment,
		SceneCode:         req.SceneCode,
		LessonPlanID:      req.LessonPlanID,
		StageCode:         req.StageCode,
		AIResponsePreview: req.AIResponsePreview,
		TraceID:           req.TraceID,
	}

	if err := repository.CreateAssistantFeedback(r.Context(), feedback); err != nil {
		utils.InternalError(w, "保存反馈失败: "+err.Error())
		return
	}

	utils.Success(w, feedback)
}

// ==================== 2. DELETE /assistant-feedback/{id} 删除反馈 ====================

// Delete 删除反馈(只能删自己的)
//
// 权限:
//   - 反馈创建者本人
//   - admin 可删除任何反馈(作为管理员兜底)
func (h *AssistantFeedbackHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}

	id := extractFeedbackID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少反馈 ID")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	// 查出反馈,校验所有权
	feedback, err := repository.GetAssistantFeedbackByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrFeedbackNotFound) {
			utils.Fail(w, http.StatusNotFound, err.Error())
			return
		}
		utils.InternalError(w, err.Error())
		return
	}

	// 权限校验:本人 OR admin
	if feedback.UserID != claims.UserID && claims.Role != models.RoleAdmin {
		utils.Forbidden(w, "只能删除自己的反馈")
		return
	}

	if err := repository.DeleteAssistantFeedback(r.Context(), id); err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, map[string]interface{}{"message": "删除成功"})
}

// ==================== 3. GET /assistant-feedback 列表(admin only) ====================

// List 列表查询(admin 后台使用,支持多维筛选+分页)
//
// Query 参数:
//   assistant_id, user_id, rating, scene_code, start_date, end_date, page, page_size
//
// 权限:路由层用 adminOnly 中间件保护,这里不再二次校验角色
func (h *AssistantFeedbackHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	params := &models.ListFeedbackParams{
		AssistantID: strings.TrimSpace(q.Get("assistant_id")),
		UserID:      strings.TrimSpace(q.Get("user_id")),
		Rating:      strings.TrimSpace(q.Get("rating")),
		SceneCode:   strings.TrimSpace(q.Get("scene_code")),
		StartDate:   strings.TrimSpace(q.Get("start_date")),
		EndDate:     strings.TrimSpace(q.Get("end_date")),
		Page:        page,
		PageSize:    pageSize,
	}

	resp, err := repository.ListAssistantFeedback(r.Context(), params)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, resp)
}

// ==================== 4. GET /assistants/{id}/feedback-stats 单助手统计 ====================

// Stats 某助手的反馈统计
//
// 路径:/api/v1/assistants/{id}/feedback-stats
// 用途:展示在助手详情页或 Selector 上,老师能直接看到某助手的受欢迎程度
//
// 权限:任何已登录用户均可查(公开透明,有助于老师选择更好的助手)
func (h *AssistantFeedbackHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	// 路径格式:/api/v1/assistants/{id}/feedback-stats
	// 解析 {id}:去掉前缀 /api/v1/assistants/,去掉后缀 /feedback-stats
	path := r.URL.Path
	const prefix = "/api/v1/assistants/"
	const suffix = "/feedback-stats"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		utils.BadRequest(w, "路径格式错误")
		return
	}
	assistantID := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if assistantID == "" {
		utils.BadRequest(w, "缺少助手 ID")
		return
	}

	stats, err := repository.GetAssistantFeedbackStats(r.Context(), assistantID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}

	utils.Success(w, stats)
}

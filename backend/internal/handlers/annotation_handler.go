package handlers

// annotation_handler.go — 教案段落批注HTTP处理器
//
// API列表：
//   GET    /api/v1/lesson-plans/plans/{id}/annotations                      — 查询教案全部批注
//   POST   /api/v1/lesson-plans/plans/{id}/annotations                      — 创建批注（评审员）
//   PUT    /api/v1/lesson-plans/plans/{id}/annotations/{aid}                — 修改批注内容（评审员本人）
//   DELETE /api/v1/lesson-plans/plans/{id}/annotations/{aid}                — 删除批注（评审员本人或admin）
//   PUT    /api/v1/lesson-plans/plans/{id}/annotations/{aid}/resolve        — 标记处理状态（作者）
//   POST   /api/v1/lesson-plans/plans/{id}/annotations/{aid}/ai-fix        — AI辅助修改（SSE流式，作者）
//
// v104改动：
//   - ListAnnotations 响应增加 current_round 字段（前端据此按轮次分组显示）
//   - CreateAnnotation 自动查询当前最大 review_round 并填入新批注

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// AnnotationHandler 批注接口处理器
type AnnotationHandler struct {
	cfg *config.Config
}

// NewAnnotationHandler 创建批注处理器实例
func NewAnnotationHandler(cfg *config.Config) *AnnotationHandler {
	return &AnnotationHandler{cfg: cfg}
}

// ==================== 查询批注列表 ====================

// ListAnnotations GET /plans/{id}/annotations
// v104：响应体增加 current_round，前端可据此按轮次分组，区分历史批注和当前轮次批注
func (h *AnnotationHandler) ListAnnotations(w http.ResponseWriter, r *http.Request) {
	planID := extractLPID(r.URL.Path)
	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	list, err := repository.ListAnnotationsByPlanID(r.Context(), planID)
	if err != nil {
		log.Printf("查询批注列表失败: %v", err)
		utils.InternalError(w, "查询批注失败")
		return
	}
	if list == nil {
		list = []*models.LessonPlanAnnotation{}
	}

	// 计算当前最大轮次（供前端按轮次分组显示）
	currentRound := 1
	for _, a := range list {
		if a.ReviewRound > currentRound {
			currentRound = a.ReviewRound
		}
	}

	utils.Success(w, &models.AnnotationListResponse{
		Annotations:  list,
		Total:        len(list),
		CurrentRound: currentRound,
	})
}

// ==================== 创建批注 ====================

// CreateAnnotation POST /plans/{id}/annotations
// v104：自动查询当前最大 review_round 填入新批注，确保轮次正确
func (h *AnnotationHandler) CreateAnnotation(w http.ResponseWriter, r *http.Request) {
	planID := extractLPID(r.URL.Path)
	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var req models.CreateAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		utils.BadRequest(w, "批注内容不能为空")
		return
	}

	// 获取评审员显示名称
	reviewerName := claims.Username
	if user, err := repository.FindUserByID(r.Context(), claims.UserID); err == nil {
		reviewerName = user.DisplayName
	}

	// v104：自动推断当前评审轮次（从现有批注中取最大轮次+1，或使用请求体指定的值）
	// 逻辑：如果请求体显式传了review_round则使用之；否则自动从现有批注中推断当前轮次
	reviewRound := req.ReviewRound
	if reviewRound <= 0 {
		// 自动查当前最大轮次，新批注属于该轮次（评审员在当前轮次内添加批注）
		if maxRound, err := repository.GetCurrentAnnotationRound(r.Context(), planID); err == nil {
			reviewRound = maxRound
		} else {
			reviewRound = 1
		}
	}

	a := &models.LessonPlanAnnotation{
		LessonPlanID:     planID,
		ReviewerID:       claims.UserID,
		ReviewerName:     reviewerName,
		ParagraphIndex:   req.ParagraphIndex,
		ParagraphPreview: req.ParagraphPreview,
		Content:          strings.TrimSpace(req.Content),
		ReviewRound:      reviewRound,
	}

	if err := repository.CreateAnnotation(r.Context(), a); err != nil {
		log.Printf("创建批注失败: %v", err)
		utils.InternalError(w, "创建批注失败")
		return
	}
	utils.Success(w, a)
}

// ==================== 修改批注内容 ====================

// UpdateAnnotation PUT /plans/{id}/annotations/{aid}
func (h *AnnotationHandler) UpdateAnnotation(w http.ResponseWriter, r *http.Request) {
	aid := extractAnnotationID(r.URL.Path)
	if aid == "" {
		utils.BadRequest(w, "缺少批注ID")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	// 验证批注归属
	existing, err := repository.GetAnnotationByID(r.Context(), aid)
	if err != nil {
		if errors.Is(err, repository.ErrAnnotationNotFound) {
			utils.Fail(w, http.StatusNotFound, "批注不存在")
			return
		}
		utils.InternalError(w, "查询批注失败")
		return
	}
	if existing.ReviewerID != claims.UserID && claims.Role != "admin" {
		utils.Forbidden(w, "只能修改自己的批注")
		return
	}

	var req models.UpdateAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		utils.BadRequest(w, "批注内容不能为空")
		return
	}

	if err := repository.UpdateAnnotationContent(r.Context(), aid, strings.TrimSpace(req.Content)); err != nil {
		utils.InternalError(w, "更新批注失败")
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// ==================== 删除批注 ====================

// DeleteAnnotation DELETE /plans/{id}/annotations/{aid}
func (h *AnnotationHandler) DeleteAnnotation(w http.ResponseWriter, r *http.Request) {
	aid := extractAnnotationID(r.URL.Path)
	if aid == "" {
		utils.BadRequest(w, "缺少批注ID")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	existing, err := repository.GetAnnotationByID(r.Context(), aid)
	if err != nil {
		if errors.Is(err, repository.ErrAnnotationNotFound) {
			utils.Fail(w, http.StatusNotFound, "批注不存在")
			return
		}
		utils.InternalError(w, "查询批注失败")
		return
	}
	if existing.ReviewerID != claims.UserID && claims.Role != "admin" {
		utils.Forbidden(w, "只能删除自己的批注")
		return
	}

	if err := repository.DeleteAnnotation(r.Context(), aid); err != nil {
		utils.InternalError(w, "删除批注失败")
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 标记批注处理状态 ====================

// ResolveAnnotation PUT /plans/{id}/annotations/{aid}/resolve
func (h *AnnotationHandler) ResolveAnnotation(w http.ResponseWriter, r *http.Request) {
	aid := extractAnnotationID(r.URL.Path)
	if aid == "" {
		utils.BadRequest(w, "缺少批注ID")
		return
	}

	var req models.ResolveAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if req.Status != models.AnnotationStatusPending && req.Status != models.AnnotationStatusResolved {
		utils.BadRequest(w, "状态只能是 pending 或 resolved")
		return
	}

	if err := repository.UpdateAnnotationStatus(r.Context(), aid, req.Status); err != nil {
		if errors.Is(err, repository.ErrAnnotationNotFound) {
			utils.Fail(w, http.StatusNotFound, "批注不存在")
			return
		}
		utils.InternalError(w, "更新状态失败")
		return
	}
	utils.Success(w, map[string]string{"message": "状态已更新"})
}

// ==================== AI辅助修改（SSE流式） ====================

// AIFixAnnotationRequest AI辅助修改请求体
// v104新增：PlanContext 教案全貌上下文，让AI理解整体教学设计后再给出段落修改建议
type AIFixAnnotationRequest struct {
	ParagraphContent  string `json:"paragraph_content"`  // 原段落文字
	AnnotationContent string `json:"annotation_content"` // 批注意见
	PlanContext        string `json:"plan_context"`        // 教案全貌（学科/年级/课题/完整正文）
}

// annotationAIFixSystemPrompt AI辅助修改专用系统提示词
// v104更新：系统提示词中明确说明会提供教案全貌上下文，让AI基于整体设计给出精准修改建议
const annotationAIFixSystemPrompt = `你是一位专业的教案修改助手。你将收到一份教案的完整背景信息、需要修改的具体段落、以及评审员的批注意见。

## 任务说明
- 先理解教案的整体教学设计和风格（学科/年级/课题/完整教案）
- 在整体上下文中理解被批注段落的定位和作用
- 充分分析评审员批注指出的具体问题
- 给出与整体教案风格一致、可直接替换的段落修改建议

## 输出要求
1. 先简要分析批注指出的问题（1-2句话，点明问题本质）
2. 给出修改后的完整段落内容（可直接替换原段落，保持与教案整体风格一致）
3. 修改要精准，只改有问题的部分，不要过度改写无关内容

## 输出格式
【问题分析】
（简要说明评审员批注指出的核心问题）

【修改建议】
（修改后的完整段落内容，可直接用于替换原段落）`

// AIFixAnnotation POST /plans/{id}/annotations/{aid}/ai-fix
// SSE流式返回AI修改建议，前端展示后可"采用"（替换段落）或"忽略"
func (h *AnnotationHandler) AIFixAnnotation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	// 提取路径参数
	planID := extractLPID(r.URL.Path)
	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	aid := extractAnnotationID(r.URL.Path)
	if aid == "" {
		utils.BadRequest(w, "缺少批注ID")
		return
	}

	// 鉴权：必须已登录
	_, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	// 解析请求体
	var req AIFixAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if strings.TrimSpace(req.ParagraphContent) == "" {
		utils.BadRequest(w, "段落内容不能为空")
		return
	}
	if strings.TrimSpace(req.AnnotationContent) == "" {
		utils.BadRequest(w, "批注内容不能为空")
		return
	}

	// 验证批注存在且属于该教案
	annotation, err := repository.GetAnnotationByID(r.Context(), aid)
	if err != nil {
		if errors.Is(err, repository.ErrAnnotationNotFound) {
			utils.Fail(w, http.StatusNotFound, "批注不存在")
			return
		}
		utils.InternalError(w, "查询批注失败")
		return
	}
	if annotation.LessonPlanID != planID {
		utils.Forbidden(w, "批注与教案不匹配")
		return
	}

	// 获取AI配置（复用lesson_plan场景）
	aiCfg, err := ai.GetEffectiveConfig(
		h.cfg.AESKey, "lesson_plan",
		h.cfg.AIAPIBaseURL, h.cfg.AIAPIKey, h.cfg.AIDefaultModel,
	)
	if err != nil {
		utils.InternalError(w, "获取AI配置失败")
		return
	}

	// 构建用户提示词：加入教案全貌上下文（v104新增）
	var userPromptBuilder strings.Builder
	if strings.TrimSpace(req.PlanContext) != "" {
		userPromptBuilder.WriteString("【教案全貌】\n")
		userPromptBuilder.WriteString(strings.TrimSpace(req.PlanContext))
		userPromptBuilder.WriteString("\n\n")
	}
	userPromptBuilder.WriteString("【需要修改的段落】\n")
	userPromptBuilder.WriteString(strings.TrimSpace(req.ParagraphContent))
	userPromptBuilder.WriteString("\n\n【评审员批注意见】\n")
	userPromptBuilder.WriteString(strings.TrimSpace(req.AnnotationContent))
	userPrompt := userPromptBuilder.String()

	// 切换为SSE流式响应头
	flusher, ok := w.(http.Flusher)
	if !ok {
		utils.InternalError(w, "不支持流式响应")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	// 发送connected事件
	writeAnnotationSSEEvent(w, flusher, "connected", map[string]string{
		"plan_id":       planID,
		"annotation_id": aid,
	})

	// 流式调用AI，逐token推送chunk事件
	startTime := time.Now()
	callResult, callErr := ai.CallAIStream(
		aiCfg,
		annotationAIFixSystemPrompt,
		userPrompt,
		func(chunk string) error {
			writeAnnotationSSEEvent(w, flusher, "chunk", map[string]string{"chunk": chunk})
			return nil
		},
		nil,
	)

	if callErr != nil {
		log.Printf("[annotation ai-fix] AI调用失败 plan=%s aid=%s err=%v", planID, aid, callErr)
		writeAnnotationSSEEvent(w, flusher, "error", map[string]string{
			"error": "AI修改建议生成失败，请稍后重试",
		})
		return
	}

	latencyMs := time.Since(startTime).Milliseconds()
	log.Printf("[annotation ai-fix] 完成 plan=%s aid=%s tokens=%d latency=%dms",
		planID, aid, callResult.TokensUsed, latencyMs)

	// 发送done事件，携带完整内容
	writeAnnotationSSEEvent(w, flusher, "done", map[string]interface{}{
		"full_content": callResult.Content,
		"tokens_used":  callResult.TokensUsed,
	})
}

// writeAnnotationSSEEvent 向客户端写入一条SSE事件
func writeAnnotationSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	flusher.Flush()
}

// ==================== 辅助函数 ====================

// extractAnnotationID 从路径提取批注ID
// 支持路径格式：
//
//	/api/v1/lesson-plans/plans/{planId}/annotations/{annotationId}
//	/api/v1/lesson-plans/plans/{planId}/annotations/{annotationId}/resolve
//	/api/v1/lesson-plans/plans/{planId}/annotations/{annotationId}/ai-fix
func extractAnnotationID(path string) string {
	const marker = "/annotations/"
	idx := strings.Index(path, marker)
	if idx < 0 {
		return ""
	}
	rest := path[idx+len(marker):]
	// 去掉尾部的子路径（/resolve、/ai-fix 等）
	if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
		rest = rest[:slashIdx]
	}
	return rest
}

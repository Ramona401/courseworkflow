package handlers

// ai_assistant_handler.go — AI 助手 HTTP 处理器
//
// 注册路由(详见 routes/routes_ai_assistant.go):
//   GET    /api/v1/ai-assistants            列表(按场景/学科/年级筛选)
//   POST   /api/v1/ai-assistants            创建(按角色自动选 source)
//   GET    /api/v1/ai-assistants/{id}       详情
//   PUT    /api/v1/ai-assistants/{id}       编辑
//   DELETE /api/v1/ai-assistants/{id}       删除(system 禁止)
//   POST   /api/v1/ai-assistants/{id}/fork  复制到"我的"

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// AIAssistantHandler AI 助手处理器
type AIAssistantHandler struct {
	service *services.AIAssistantService
}

// NewAIAssistantHandler 构造函数
func NewAIAssistantHandler(svc *services.AIAssistantService) *AIAssistantHandler {
	return &AIAssistantHandler{service: svc}
}

// ==================== 路径解析辅助 ====================

const aiAssistantPathPrefix = "/api/v1/ai-assistants/"

// extractAssistantID 从 path 中解析 {id} 段,支持 /{id} 和 /{id}/fork 两种
func extractAssistantID(path string) string {
	rest := strings.TrimPrefix(path, aiAssistantPathPrefix)
	rest = strings.TrimSuffix(rest, "/")
	if rest == "" {
		return ""
	}
	parts := strings.Split(rest, "/")
	return parts[0]
}

// isForkPath 判断 path 是否为 /{id}/fork
func isForkPath(path string) bool {
	return strings.HasSuffix(path, "/fork")
}

// ==================== 公共:解析 actor ====================

// resolveActor 从 request 中解析当前用户的 ActorContext
func (h *AIAssistantHandler) resolveActor(r *http.Request) (*services.AssistantActorContext, error) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		return nil, errors.New(utils.MsgNotLoggedIn)
	}
	return services.BuildActorFromClaims(r.Context(), claims.UserID, claims.Role), nil
}

// ==================== 1. GET /ai-assistants 列表 ====================

// List GET /api/v1/ai-assistants?scene=xxx&subject=xxx&grade=xxx&only_active=1
func (h *AIAssistantHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	q := r.URL.Query()
	scene := strings.TrimSpace(q.Get("scene"))
	subject := strings.TrimSpace(q.Get("subject"))
	gradeRange := strings.TrimSpace(q.Get("grade"))
	onlyActive := q.Get("only_active") != "0" // 默认只返回激活的

	resp, err := h.service.ListAssistants(r.Context(), actor, scene, subject, gradeRange, onlyActive)
	if err != nil {
		utils.InternalError(w, "获取 AI 助手列表失败: "+err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 2. GET /ai-assistants/{id} 详情 ====================

// Get GET /api/v1/ai-assistants/{id}
func (h *AIAssistantHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	id := extractAssistantID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少助手 ID")
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	a, err := h.service.GetAssistant(r.Context(), actor, id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	utils.Success(w, a)
}

// ==================== 3. POST /ai-assistants 创建 ====================

// Create POST /api/v1/ai-assistants
func (h *AIAssistantHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	var req models.CreateAIAssistantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	a, err := h.service.CreateAssistant(r.Context(), actor, &req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	// 审计日志
	repository.WriteAuditLog(actor.UserID, "ai_assistant.create",
		map[string]interface{}{
			"assistant_id": a.ID,
			"name":         a.Name,
			"source":       a.Source,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	utils.Success(w, a)
}

// ==================== 4. PUT /ai-assistants/{id} 编辑 ====================

// Update PUT /api/v1/ai-assistants/{id}
func (h *AIAssistantHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}

	id := extractAssistantID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少助手 ID")
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	var req models.UpdateAIAssistantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if err := h.service.UpdateAssistant(r.Context(), actor, id, &req); err != nil {
		h.writeServiceError(w, err)
		return
	}

	// 审计日志
	repository.WriteAuditLog(actor.UserID, "ai_assistant.update",
		map[string]interface{}{
			"assistant_id": id,
			"name":         req.Name,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	utils.Success(w, map[string]interface{}{"message": "更新成功"})
}

// ==================== 5. DELETE /ai-assistants/{id} 删除 ====================

// Delete DELETE /api/v1/ai-assistants/{id}
func (h *AIAssistantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}

	id := extractAssistantID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少助手 ID")
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	if err := h.service.DeleteAssistant(r.Context(), actor, id); err != nil {
		h.writeServiceError(w, err)
		return
	}

	// 审计日志
	repository.WriteAuditLog(actor.UserID, "ai_assistant.delete",
		map[string]interface{}{
			"assistant_id": id,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	utils.Success(w, map[string]interface{}{"message": "删除成功"})
}

// ==================== 6. POST /ai-assistants/{id}/fork 复制到"我的" ====================

// Fork POST /api/v1/ai-assistants/{id}/fork
func (h *AIAssistantHandler) Fork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	id := extractAssistantID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少助手 ID")
		return
	}

	actor, err := h.resolveActor(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	forked, err := h.service.ForkAssistant(r.Context(), actor, id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	// 审计日志
	repository.WriteAuditLog(actor.UserID, "ai_assistant.fork",
		map[string]interface{}{
			"source_id": id,
			"new_id":    forked.ID,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	utils.Success(w, forked)
}

// ==================== 错误映射 ====================

// writeServiceError 将 service 层错误映射为合适的 HTTP 状态码
func (h *AIAssistantHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrAIAssistantNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	case errors.Is(err, repository.ErrAIAssistantInactive):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrAssistantPermDenied):
		utils.Forbidden(w, err.Error())
	case errors.Is(err, services.ErrAssistantNameRequired),
		errors.Is(err, services.ErrAssistantPromptRequired),
		errors.Is(err, services.ErrAssistantScenesRequired),
		errors.Is(err, services.ErrAssistantInvalidSource),
		errors.Is(err, services.ErrAssistantInvalidScene),
		errors.Is(err, services.ErrAssistantPromptTooLong),
		errors.Is(err, services.ErrSchoolBindingRequired):
		utils.BadRequest(w, err.Error())
	default:
		utils.InternalError(w, err.Error())
	}
}

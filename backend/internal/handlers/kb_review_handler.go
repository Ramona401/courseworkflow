package handlers

// kb_review_handler.go — 知识库课标审核处理器
//
// 端点（均需白名单守卫）：
//   GET  /api/v1/kb/jobs/{id}/review-queue   审核队列（解码人话）
//   POST /api/v1/kb/items/{id}/review        三选一动作（confirm/select/reject）
//   POST /api/v1/kb/jobs/{id}/commit         灌入目标表（候选态）body:{batch_tag}
//   POST /api/v1/kb/switch-batch             蓝绿切换 body:{batch_tag}

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// KBReviewHandler 知识库审核处理器
type KBReviewHandler struct {
	reviewService *services.KBReviewService
}

// NewKBReviewHandler 创建审核处理器
func NewKBReviewHandler(reviewService *services.KBReviewService) *KBReviewHandler {
	return &KBReviewHandler{reviewService: reviewService}
}

// GetReviewQueue GET /api/v1/kb/jobs/{id}/review-queue
func (h *KBReviewHandler) GetReviewQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	jobID := kbExtractJobIDMiddle(r.URL.Path, "/review-queue")
	if jobID == "" {
		utils.BadRequest(w, "缺少任务ID")
		return
	}
	views, err := h.reviewService.GetReviewQueue(r.Context(), jobID)
	if err != nil {
		utils.InternalError(w, "获取审核队列失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"items": views, "total": len(views)})
}

// ReviewAction POST /api/v1/kb/items/{id}/review
func (h *KBReviewHandler) ReviewAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	itemID := kbExtractItemID(r.URL.Path)
	if itemID == "" {
		utils.BadRequest(w, "缺少单元ID")
		return
	}
	var req models.KBReviewActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if err := h.reviewService.ReviewAction(r.Context(), itemID, &req, claims.UserID); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "审核已提交"})
}

// CommitBatch POST /api/v1/kb/jobs/{id}/commit  body:{batch_tag}
func (h *KBReviewHandler) CommitBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	jobID := kbExtractJobIDMiddle(r.URL.Path, "/commit")
	if jobID == "" {
		utils.BadRequest(w, "缺少任务ID")
		return
	}
	var body struct {
		BatchTag string `json:"batch_tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if strings.TrimSpace(body.BatchTag) == "" {
		utils.BadRequest(w, "缺少 batch_tag")
		return
	}
	result, err := h.reviewService.CommitBatch(r.Context(), jobID, body.BatchTag)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	utils.Success(w, result)
}

// SwitchBatch POST /api/v1/kb/switch-batch  body:{batch_tag}
func (h *KBReviewHandler) SwitchBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	var body struct {
		BatchTag string `json:"batch_tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if strings.TrimSpace(body.BatchTag) == "" {
		utils.BadRequest(w, "缺少 batch_tag")
		return
	}
	result, err := h.reviewService.SwitchBatch(r.Context(), body.BatchTag)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	utils.Success(w, result)
}

// ==================== 路径辅助 ====================

// kbExtractJobIDMiddle 从 /api/v1/kb/jobs/{id}/{suffix} 抠 id
func kbExtractJobIDMiddle(path, suffix string) string {
	const prefix = "/api/v1/kb/jobs/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimSuffix(strings.TrimRight(path, "/"), suffix)
	rest = rest[len(prefix):]
	rest = strings.TrimRight(rest, "/")
	if idx := strings.Index(rest, "/"); idx > 0 {
		return rest[:idx]
	}
	return rest
}

// kbExtractItemID 从 /api/v1/kb/items/{id}/review 抠 id
func kbExtractItemID(path string) string {
	const prefix = "/api/v1/kb/items/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := path[len(prefix):]
	rest = strings.TrimRight(rest, "/")
	if idx := strings.Index(rest, "/"); idx > 0 {
		return rest[:idx]
	}
	return rest
}

var _ = repository.GetKBItemByID

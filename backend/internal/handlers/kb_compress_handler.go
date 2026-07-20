package handlers

// kb_compress_handler.go — 知识库课标压缩处理器
//
// 端点（均需白名单守卫，SSE 端点内部验 token + 查白名单）：
//   POST /api/v1/kb/jobs                      创建压缩任务（立即返回，异步压缩）
//   GET  /api/v1/kb/jobs?kind=curriculum      任务列表
//   GET  /api/v1/kb/jobs/{id}                 任务详情（含进度）
//   GET  /api/v1/sse/kb/{id}?token=xxx        SSE 订阅压缩进度

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// KBCompressHandler 知识库压缩处理器
type KBCompressHandler struct {
	compressService *services.KBCompressService
	authService     *services.AuthService
}

// NewKBCompressHandler 创建压缩处理器
func NewKBCompressHandler(compressService *services.KBCompressService, authService *services.AuthService) *KBCompressHandler {
	return &KBCompressHandler{compressService: compressService, authService: authService}
}

// CreateJob POST /api/v1/kb/jobs — 创建压缩任务并异步触发压缩
func (h *KBCompressHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	var req models.KBCreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	jobID, err := h.compressService.CreateJob(r.Context(), &req, claims.UserID, "")
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	// 异步触发压缩（延迟 800ms 确保前端 SSE 先连上）
	go func() {
		time.Sleep(800 * time.Millisecond)
		h.compressService.RunCompressAsync(jobID)
	}()
	utils.Success(w, map[string]interface{}{
		"job_id":  jobID,
		"message": "压缩任务已创建，请通过SSE监听进度",
	})
}

// ListJobs GET /api/v1/kb/jobs?kind=curriculum&batch_tag=xxx — 任务列表
func (h *KBCompressHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = models.KBKindCurriculum
	}
	batchTag := r.URL.Query().Get("batch_tag")
	jobs, err := repository.ListKBJobsByKind(r.Context(), kind, batchTag)
	if err != nil {
		utils.InternalError(w, "查询任务列表失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"jobs": jobs, "total": len(jobs)})
}

// GetJob GET /api/v1/kb/jobs/{id} — 任务详情
func (h *KBCompressHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := kbExtractJobID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少任务ID")
		return
	}
	job, err := repository.GetKBJobByID(r.Context(), id)
	if err != nil {
		utils.InternalError(w, "任务不存在: "+err.Error())
		return
	}
	utils.Success(w, job)
}

// ProgressStream GET /api/v1/sse/kb/{id}?token=xxx — SSE 压缩进度
func (h *KBCompressHandler) ProgressStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	// SSE 内部 token 验证（EventSource 不能设 header，走 query）
	token := r.URL.Query().Get("token")
	if token == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if token == "" {
		http.Error(w, `{"code":-1,"message":"缺少token参数"}`, http.StatusUnauthorized)
		return
	}
	claims, err := h.authService.ValidateToken(token)
	if err != nil {
		http.Error(w, `{"code":-1,"message":"token无效或已过期"}`, http.StatusUnauthorized)
		return
	}
	// SSE 绕过了 guard 中间件，这里手动查白名单（admin 恒通过）
	if claims.Role != "admin" {
		authorized, aerr := repository.IsKBAuthorized(r.Context(), claims.UserID)
		if aerr != nil || !authorized {
			http.Error(w, `{"code":-1,"message":"无权访问知识库压缩系统"}`, http.StatusForbidden)
			return
		}
	}

	id := kbExtractSSEID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少任务ID")
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

	ch := services.GlobalKBSSEHub.Subscribe(id)
	defer services.GlobalKBSSEHub.Unsubscribe(id, ch)

	finishSSEHandshake()

	writeKBSSEEvent(w, flusher, services.KBSSEConnected, map[string]string{
		"job_id":  id,
		"message": "SSE连接已建立",
	})

	timeout := time.After(30 * time.Minute)
	for {
		select {
		case event, open := <-ch:
			if !open {
				return
			}
			writeKBSSEEvent(w, flusher, event.EventType, event.Data)
			if event.EventType == services.KBSSEJobDone || event.EventType == services.KBSSEError {
				return
			}
		case <-r.Context().Done():
			return
		case <-timeout:
			writeKBSSEEvent(w, flusher, "timeout", map[string]string{"message": "SSE连接超时"})
			return
		}
	}
}

// ==================== SSE/路径辅助 ====================

func writeKBSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	writeRawKB(w, eventType, string(dataBytes))
	flusher.Flush()
}

func writeRawKB(w http.ResponseWriter, eventType, data string) {
	_, _ = w.Write([]byte("event: " + eventType + "\ndata: " + data + "\n\n"))
}

// kbExtractJobID 从 /api/v1/kb/jobs/{id} 抠 id
func kbExtractJobID(path string) string {
	const prefix = "/api/v1/kb/jobs/"
	if strings.HasPrefix(path, prefix) {
		rest := strings.TrimRight(path[len(prefix):], "/")
		if idx := strings.Index(rest, "/"); idx > 0 {
			return rest[:idx]
		}
		return rest
	}
	return ""
}

// kbExtractSSEID 从 /api/v1/sse/kb/{id} 抠 id
func kbExtractSSEID(path string) string {
	const prefix = "/api/v1/sse/kb/"
	if strings.HasPrefix(path, prefix) {
		rest := strings.TrimRight(path[len(prefix):], "/")
		if idx := strings.Index(rest, "/"); idx > 0 {
			return rest[:idx]
		}
		return rest
	}
	return ""
}

var _ = context.Background

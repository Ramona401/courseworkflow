package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== PipelineHandler ====================

// PipelineHandler Pipeline管理API处理器
type PipelineHandler struct {
	pipelineService *services.PipelineService
}

// NewPipelineHandler 创建Pipeline处理器实例
func NewPipelineHandler(pipelineService *services.PipelineService) *PipelineHandler {
	return &PipelineHandler{pipelineService: pipelineService}
}

// ==================== Pipeline 接口 ====================

// CreatePipeline POST /api/v1/pipelines
// 创建新Pipeline（admin/operator可操作）
func (h *PipelineHandler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 获取当前用户信息
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 解析请求体
	var req models.CreatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	// 创建Pipeline
	resp, err := h.pipelineService.CreatePipeline(&req, claims.UserID)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, resp)
}

// ListPipelines GET /api/v1/pipelines
// 获取Pipeline列表（按角色过滤可见范围）
func (h *PipelineHandler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	resp, err := h.pipelineService.ListPipelines(claims.UserID, claims.Role)
	if err != nil {
		utils.InternalError(w, "获取Pipeline列表失败: "+err.Error())
		return
	}

	utils.Success(w, resp)
}

// GetPipelineDetail GET /api/v1/pipelines/{id}
// 获取Pipeline详情（含步骤列表）
func (h *PipelineHandler) GetPipelineDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	id := extractPipelineID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	resp, err := h.pipelineService.GetPipelineDetail(id)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, resp)
}

// StartPipeline POST /api/v1/pipelines/{id}/start
// 启动Pipeline执行（从dbCheck开始）
func (h *PipelineHandler) StartPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	id := extractPipelineIDWithSuffix(r.URL.Path, "/start")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	resp, err := h.pipelineService.StartPipeline(id)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, resp)
}

// CancelPipeline POST /api/v1/pipelines/{id}/cancel
// 取消Pipeline（仅pending或running状态可取消）
func (h *PipelineHandler) CancelPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	id := extractPipelineIDWithSuffix(r.URL.Path, "/cancel")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	if err := h.pipelineService.CancelPipeline(id); err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"message": "Pipeline已取消",
	})
}

// DeletePipeline DELETE /api/v1/pipelines/{id}
// 删除Pipeline（运行中的不可删除）
func (h *PipelineHandler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}

	id := extractPipelineID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	if err := h.pipelineService.DeletePipeline(id); err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"message": "Pipeline已删除",
	})
}

// GetSteps GET /api/v1/pipelines/{id}/steps
// 获取Pipeline的步骤列表
func (h *PipelineHandler) GetSteps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	id := extractPipelineIDWithSuffix(r.URL.Path, "/steps")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	// 获取完整详情（含步骤列表）
	resp, err := h.pipelineService.GetPipelineDetail(id)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	// 只返回步骤列表部分
	utils.Success(w, map[string]interface{}{
		"pipeline_id": resp.ID,
		"course_code": resp.CourseCode,
		"steps":       resp.Steps,
		"total":       len(resp.Steps),
	})
}

// GetStepDetail GET /api/v1/pipelines/{id}/steps/{name}
// 获取指定步骤的详细信息（含完整step_data）
func (h *PipelineHandler) GetStepDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 从路径中提取pipeline_id和step_name
	// 路径格式: /api/v1/pipelines/{id}/steps/{name}
	pipelineID, stepName := extractPipelineIDAndStepName(r.URL.Path)
	if pipelineID == "" || stepName == "" {
		utils.BadRequest(w, "缺少Pipeline ID或步骤名称")
		return
	}

	resp, err := h.pipelineService.GetStepDetail(pipelineID, stepName)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, resp)
}

// ==================== 路径解析辅助函数 ====================

// extractPipelineID 从路径提取Pipeline ID
// 路径格式: /api/v1/pipelines/{id}
func extractPipelineID(path string) string {
	path = strings.TrimSuffix(path, "/")
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash < 0 {
		return ""
	}
	id := path[lastSlash+1:]
	if id == "" || id == "pipelines" {
		return ""
	}
	return id
}

// extractPipelineIDWithSuffix 从带后缀的路径提取Pipeline ID
// 路径格式: /api/v1/pipelines/{id}/start 或 /api/v1/pipelines/{id}/cancel
func extractPipelineIDWithSuffix(path string, suffix string) string {
	idx := strings.LastIndex(path, suffix)
	if idx <= 0 {
		return ""
	}
	// 去掉后缀部分，得到 /api/v1/pipelines/{id}
	path = path[:idx]
	return extractPipelineID(path)
}

// extractPipelineIDAndStepName 从路径提取Pipeline ID和步骤名称
// 路径格式: /api/v1/pipelines/{id}/steps/{name}
func extractPipelineIDAndStepName(path string) (string, string) {
	// 查找 /steps/ 的位置
	stepsIdx := strings.Index(path, "/steps/")
	if stepsIdx < 0 {
		return "", ""
	}

	// 步骤名称: /steps/ 之后的部分
	stepName := strings.TrimSuffix(path[stepsIdx+len("/steps/"):], "/")
	if stepName == "" {
		return "", ""
	}

	// Pipeline ID: /steps/ 之前的路径的最后一段
	beforeSteps := path[:stepsIdx]
	pipelineID := extractPipelineID(beforeSteps)

	return pipelineID, stepName
}

// ==================== 错误处理 ====================

// handlePipelineError Pipeline统一错误处理
func handlePipelineError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	// 参数错误 (400)
	case err == services.ErrPipelineCourseRequired:
		utils.BadRequest(w, errMsg)

	// 未找到 (404)
	case err == services.ErrPipelineNotFound,
		err == services.ErrPipelineCourseNotFound:
		utils.Fail(w, http.StatusNotFound, errMsg)

	// 状态冲突 (409)
	case strings.Contains(errMsg, "已有运行中的Pipeline"),
		err == services.ErrPipelineNotPending,
		err == services.ErrPipelineNotCancellable,
		err == services.ErrPipelineNotDeletable:
		utils.Fail(w, http.StatusConflict, errMsg)

	// dbCheck验证失败 -> 返回200但Pipeline状态是failed（不是HTTP错误）
	case err == services.ErrDbCheckIndexMissing,
		err == services.ErrDbCheckIndexTooShort,
		err == services.ErrDbCheckIndexHashMismatch:
		// 这些错误不应该到这里，因为StartPipeline已经处理了
		utils.Fail(w, http.StatusUnprocessableEntity, errMsg)

	// 其他服务器错误 (500)
	default:
		utils.InternalError(w, errMsg)
	}
}

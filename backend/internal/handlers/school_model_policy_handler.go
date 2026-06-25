package handlers

// school_model_policy_handler.go — 学校境外模型授权策略处理器（批二新增，admin专属）
//
// 业务：
//   平台 AI 文本调用走「双网关分流」。默认所有学校只能用境内模型（qwen-max）；
//   仅被 admin 显式授权（school_model_policies.overseas_enabled=true）的学校，
//   才放行境外模型（claude/gemini 等）。本处理器把「授权/取消授权」从 SQL 直改
//   搬到 /admin 前端，让 admin 可视化管理「哪些学校被授权走境外」。
//
// 端点（路由层已套 authMW + adminOnly，仅系统管理员可操作；平台级境外放行不下放）：
//   GET    /api/v1/admin/school-model-policies            — 列出全部已登记策略的学校（带学校名/授权人/备注）
//   GET    /api/v1/admin/school-model-policies/{schoolID} — 查单个学校的当前策略（无记录返回默认境内态）
//   PUT    /api/v1/admin/school-model-policies/{schoolID} — 授权/取消授权（body: {overseas_enabled, note}）
//   DELETE /api/v1/admin/school-model-policies/{schoolID} — 删除策略记录（=回到默认境内）
//
// 写操作不需要手动清分流缓存：分流模块对「学校是否授权」是每次实时查库
// （isOverseasAllowedForTrace → IsSchoolOverseasEnabled，2秒短超时，无缓存），
// 与境内通道三键的 5 分钟缓存不同。故本 handler 不调任何缓存失效函数。

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// smpQueryTimeout 单次数据库操作短超时
const smpQueryTimeout = 5 * time.Second

// SchoolModelPolicyHandler 学校境外授权策略处理器
type SchoolModelPolicyHandler struct{}

// NewSchoolModelPolicyHandler 创建处理器（无依赖，repository 为包级函数）
func NewSchoolModelPolicyHandler() *SchoolModelPolicyHandler {
	return &SchoolModelPolicyHandler{}
}

// ==================== 列表 ====================

// ListPolicies GET /api/v1/admin/school-model-policies
// 仅列出 school_model_policies 表中已有记录的学校（未登记=默认境内，不在此列表）。
func (h *SchoolModelPolicyHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), smpQueryTimeout)
	defer cancel()

	items, err := repository.ListSchoolModelPolicies(ctx)
	if err != nil {
		utils.InternalError(w, "查询学校授权列表失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// ==================== 单查 / 更新 / 删除 分发 ====================

// HandlePolicyByID 按 schoolID 分发：GET=查 / PUT=授权或取消 / DELETE=删除
// 路径形如 /api/v1/admin/school-model-policies/{schoolID}
func (h *SchoolModelPolicyHandler) HandlePolicyByID(w http.ResponseWriter, r *http.Request) {
	schoolID := extractSMPSchoolID(r.URL.Path)
	if schoolID == "" {
		utils.BadRequest(w, "学校ID不能为空")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getPolicy(w, r, schoolID)
	case http.MethodPut:
		h.putPolicy(w, r, schoolID)
	case http.MethodDelete:
		h.deletePolicy(w, r, schoolID)
	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET/PUT/DELETE")
	}
}

// smpPolicyView 单个学校策略响应（无记录时返回默认境内态）
type smpPolicyView struct {
	SchoolID        string `json:"school_id"`
	OverseasEnabled bool   `json:"overseas_enabled"`
	Note            string `json:"note"`
	HasRecord       bool   `json:"has_record"` // false=该校从未登记策略（=默认境内）
}

// getPolicy GET .../{schoolID}
func (h *SchoolModelPolicyHandler) getPolicy(w http.ResponseWriter, r *http.Request, schoolID string) {
	ctx, cancel := context.WithTimeout(r.Context(), smpQueryTimeout)
	defer cancel()

	p, err := repository.GetSchoolModelPolicy(ctx, schoolID)
	if err != nil {
		utils.InternalError(w, "查询学校策略失败: "+err.Error())
		return
	}
	if p == nil {
		// 无记录：默认境内态
		utils.Success(w, smpPolicyView{SchoolID: schoolID, OverseasEnabled: false, Note: "", HasRecord: false})
		return
	}
	utils.Success(w, smpPolicyView{
		SchoolID:        p.SchoolID,
		OverseasEnabled: p.OverseasEnabled,
		Note:            p.Note,
		HasRecord:       true,
	})
}

// updatePolicyRequest 授权/取消授权请求体
type updatePolicyRequest struct {
	OverseasEnabled bool   `json:"overseas_enabled"` // true=授权境外 / false=取消授权（保留记录但关闭）
	Note            string `json:"note"`             // 可选备注（授权原因/用途）
}

// putPolicy PUT .../{schoolID} —— 授权或取消授权（UPSERT，granted_by 记当前 admin）
func (h *SchoolModelPolicyHandler) putPolicy(w http.ResponseWriter, r *http.Request, schoolID string) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	var req updatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	req.Note = strings.TrimSpace(req.Note)

	ctx, cancel := context.WithTimeout(r.Context(), smpQueryTimeout)
	defer cancel()

	if err := repository.UpsertSchoolModelPolicy(ctx, schoolID, req.OverseasEnabled, req.Note, claims.UserID); err != nil {
		utils.InternalError(w, "保存学校策略失败: "+err.Error())
		return
	}

	// 回读最新状态返回前端
	utils.Success(w, smpPolicyView{
		SchoolID:        schoolID,
		OverseasEnabled: req.OverseasEnabled,
		Note:            req.Note,
		HasRecord:       true,
	})
}

// deletePolicy DELETE .../{schoolID} —— 删除策略记录（=回到默认境内）
func (h *SchoolModelPolicyHandler) deletePolicy(w http.ResponseWriter, r *http.Request, schoolID string) {
	ctx, cancel := context.WithTimeout(r.Context(), smpQueryTimeout)
	defer cancel()

	if err := repository.DeleteSchoolModelPolicy(ctx, schoolID); err != nil {
		utils.InternalError(w, "删除学校策略失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"school_id": schoolID,
		"deleted":   true,
	})
}

// extractSMPSchoolID 从路径 /api/v1/admin/school-model-policies/{schoolID} 提取学校ID
func extractSMPSchoolID(path string) string {
	const prefix = "/api/v1/admin/school-model-policies/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	// 防御：不应再含子路径斜杠
	if strings.Contains(id, "/") {
		id = strings.SplitN(id, "/", 2)[0]
	}
	return strings.TrimSpace(id)
}

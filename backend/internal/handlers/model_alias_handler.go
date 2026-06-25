package handlers

// model_alias_handler.go — 模型别名映射规则处理器（批三-2新增，admin专属）
//
// 把「真实模型名→业务别名」映射规则的增删改查搬到 /ai-center 前端，admin 可视化维护。
// 另提供「预览」端点：输入一个真实模型名，返回它当前会被显示成的别名（自测匹配逻辑）。
//
// 端点（路由层已套 authMW + adminOnly）：
//   GET    /api/v1/admin/model-alias/rules            — 列出全部规则
//   POST   /api/v1/admin/model-alias/rules            — 新增规则
//   PUT    /api/v1/admin/model-alias/rules/{id}       — 更新规则
//   DELETE /api/v1/admin/model-alias/rules/{id}       — 删除规则
//   GET    /api/v1/admin/model-alias/fallback         — 查兜底别名
//   PUT    /api/v1/admin/model-alias/fallback         — 改兜底别名
//   POST   /api/v1/admin/model-alias/preview          — 预览：{model} → {alias}
//
// 说明：本 handler 仅 admin 侧管理。老师侧据规则渲染替换在批三-3 接。
// 别名映射无内存缓存（管理操作低频；老师侧批三-3 自行决定是否加缓存），故无失效逻辑。

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

const maQueryTimeout = 5 * time.Second

// 兜底别名配置键（与建表步骤写入 ai_configs 的键一致）
const maFallbackKey = "model_alias_fallback"

// ModelAliasHandler 模型别名处理器
type ModelAliasHandler struct{}

// NewModelAliasHandler 创建处理器
func NewModelAliasHandler() *ModelAliasHandler {
	return &ModelAliasHandler{}
}

// ==================== 规则列表 / 新增 ====================

// HandleRules GET=列表 / POST=新增（路径 /model-alias/rules）
func (h *ModelAliasHandler) HandleRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listRules(w, r)
	case http.MethodPost:
		h.createRule(w, r)
	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET/POST")
	}
}

func (h *ModelAliasHandler) listRules(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), maQueryTimeout)
	defer cancel()

	items, err := repository.ListModelAliasRules(ctx)
	if err != nil {
		utils.InternalError(w, "查询别名规则失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"items": items, "total": len(items)})
}

// ruleReq 新增/更新规则请求体
type ruleReq struct {
	MatchType string `json:"match_type"` // exact / prefix
	Pattern   string `json:"pattern"`
	Alias     string `json:"alias"`
	Priority  int    `json:"priority"`
	Enabled   bool   `json:"enabled"`
	Note      string `json:"note"`
}

func (h *ModelAliasHandler) createRule(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}
	var req ruleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	req.MatchType = strings.TrimSpace(req.MatchType)
	req.Pattern = strings.TrimSpace(req.Pattern)
	req.Alias = strings.TrimSpace(req.Alias)
	req.Note = strings.TrimSpace(req.Note)
	if req.Pattern == "" || req.Alias == "" {
		utils.BadRequest(w, "匹配内容与别名不能为空")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), maQueryTimeout)
	defer cancel()

	id, err := repository.CreateModelAliasRule(ctx, req.MatchType, req.Pattern, req.Alias, req.Priority, req.Enabled, req.Note, claims.UserID)
	if err != nil {
		// 唯一索引冲突 → 转人话
		if strings.Contains(err.Error(), "uniq_model_alias_type_pattern") || strings.Contains(err.Error(), "duplicate key") {
			utils.BadRequest(w, "已存在相同【匹配类型+匹配内容】的规则")
			return
		}
		utils.InternalError(w, "创建别名规则失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"id": id})
}

// ==================== 规则更新 / 删除（带 {id}）====================

// HandleRuleByID PUT=更新 / DELETE=删除（路径 /model-alias/rules/{id}）
func (h *ModelAliasHandler) HandleRuleByID(w http.ResponseWriter, r *http.Request) {
	id := extractMARuleID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "规则ID不能为空")
		return
	}
	switch r.Method {
	case http.MethodPut:
		h.updateRule(w, r, id)
	case http.MethodDelete:
		h.deleteRule(w, r, id)
	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT/DELETE")
	}
}

func (h *ModelAliasHandler) updateRule(w http.ResponseWriter, r *http.Request, id string) {
	var req ruleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	req.MatchType = strings.TrimSpace(req.MatchType)
	req.Pattern = strings.TrimSpace(req.Pattern)
	req.Alias = strings.TrimSpace(req.Alias)
	req.Note = strings.TrimSpace(req.Note)

	ctx, cancel := context.WithTimeout(r.Context(), maQueryTimeout)
	defer cancel()

	if err := repository.UpdateModelAliasRule(ctx, id, req.MatchType, req.Pattern, req.Alias, req.Priority, req.Enabled, req.Note); err != nil {
		if strings.Contains(err.Error(), "uniq_model_alias_type_pattern") || strings.Contains(err.Error(), "duplicate key") {
			utils.BadRequest(w, "已存在相同【匹配类型+匹配内容】的规则")
			return
		}
		utils.InternalError(w, "更新别名规则失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"id": id, "updated": true})
}

func (h *ModelAliasHandler) deleteRule(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), maQueryTimeout)
	defer cancel()

	if err := repository.DeleteModelAliasRule(ctx, id); err != nil {
		utils.InternalError(w, "删除别名规则失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"id": id, "deleted": true})
}

// ==================== 兜底别名 ====================

// HandleFallback GET=查 / PUT=改（路径 /model-alias/fallback）
func (h *ModelAliasHandler) HandleFallback(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		utils.Success(w, map[string]interface{}{"fallback": h.readFallback()})
	case http.MethodPut:
		h.putFallback(w, r)
	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET/PUT")
	}
}

// readFallback 读兜底别名，缺失用默认值
func (h *ModelAliasHandler) readFallback() string {
	c, err := repository.GetConfigByKey(maFallbackKey)
	if err != nil || c == nil || strings.TrimSpace(c.ConfigValue) == "" {
		return models.DefaultModelAliasFallback
	}
	return strings.TrimSpace(c.ConfigValue)
}

func (h *ModelAliasHandler) putFallback(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}
	var body struct {
		Fallback string `json:"fallback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	body.Fallback = strings.TrimSpace(body.Fallback)
	if body.Fallback == "" {
		utils.BadRequest(w, "兜底别名不能为空")
		return
	}
	if err := repository.UpsertConfigValue(maFallbackKey, body.Fallback, "模型别名兜底名（无规则命中时显示）", claims.UserID); err != nil {
		utils.InternalError(w, "保存兜底别名失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"fallback": body.Fallback})
}

// ==================== 预览 ====================

// PreviewAlias POST /model-alias/preview —— 输入模型名，返回当前会显示的别名（自测匹配逻辑）
func (h *ModelAliasHandler) PreviewAlias(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST")
		return
	}
	var body struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	body.Model = strings.TrimSpace(body.Model)
	if body.Model == "" {
		utils.BadRequest(w, "模型名不能为空")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), maQueryTimeout)
	defer cancel()

	alias := repository.ResolveModelAlias(ctx, body.Model, h.readFallback())
	utils.Success(w, map[string]interface{}{"model": body.Model, "alias": alias})
}

// extractMARuleID 从 /api/v1/admin/model-alias/rules/{id} 提取规则ID
func extractMARuleID(path string) string {
	const prefix = "/api/v1/admin/model-alias/rules/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	if strings.Contains(id, "/") {
		id = strings.SplitN(id, "/", 2)[0]
	}
	return strings.TrimSpace(id)
}

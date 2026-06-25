package handlers

// gateway_naming_handler.go — 双网关展示名（对外可读别名）配置处理器（批三-1新增，admin专属）
//
// 背景：
//   平台 AI 文本调用走「双网关分流」——境外主网关（claude/gemini）与境内网关（qwen）。
//   为了对外（尤其老师侧）不直接暴露"境外/claude"等敏感字眼，给两个网关各起一个
//   业务可读展示名（如"星云国际通道"/"星云境内通道"），供配置界面与将来老师侧渲染读取。
//
// 配置键（ai_configs 表，经 UpsertConfigValue 动态建立，无需迁移）：
//   overseas_gateway_label — 境外网关展示名
//   domestic_gateway_label — 境内网关展示名
//
// 端点（路由层已套 authMW + adminOnly）：
//   GET /api/v1/admin/gateway-naming  — 查看两网关展示名（未配置返回空串，前端用默认兜底）
//   PUT /api/v1/admin/gateway-naming  — 更新两展示名（任一字段留空=不修改对应项）
//
// 说明：本处理器仅做 admin 侧存/取。老师侧的「公开可读」入口在批三-3 统一接
//       （批三-3 才动老师侧渲染），故此处不提供免鉴权读接口。

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// 网关展示名配置键
const (
	gnOverseasLabelKey = "overseas_gateway_label" // 境外网关展示名
	gnDomesticLabelKey = "domestic_gateway_label" // 境内网关展示名
)

// GatewayNamingHandler 双网关展示名处理器（无依赖，复用 ai_configs 键值存储）
type GatewayNamingHandler struct{}

// NewGatewayNamingHandler 创建处理器
func NewGatewayNamingHandler() *GatewayNamingHandler {
	return &GatewayNamingHandler{}
}

// HandleGatewayNaming GET=查看 / PUT=更新
func (h *GatewayNamingHandler) HandleGatewayNaming(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getNaming(w, r)
	case http.MethodPut:
		h.putNaming(w, r)
	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET/PUT")
	}
}

// gatewayNamingView 展示名响应结构
type gatewayNamingView struct {
	OverseasLabel string `json:"overseas_label"` // 境外网关展示名（未配置为空串）
	DomesticLabel string `json:"domestic_label"` // 境内网关展示名（未配置为空串）
}

// gnReadConfigOrEmpty 读单个配置键，缺失或为空返回空串
func gnReadConfigOrEmpty(key string) string {
	c, err := repository.GetConfigByKey(key)
	if err != nil || c == nil {
		return ""
	}
	return strings.TrimSpace(c.ConfigValue)
}

// getNaming GET /api/v1/admin/gateway-naming
func (h *GatewayNamingHandler) getNaming(w http.ResponseWriter, _ *http.Request) {
	utils.Success(w, gatewayNamingView{
		OverseasLabel: gnReadConfigOrEmpty(gnOverseasLabelKey),
		DomesticLabel: gnReadConfigOrEmpty(gnDomesticLabelKey),
	})
}

// updateGatewayNamingRequest 更新请求体（两字段均可选，留空=不修改对应项）
type updateGatewayNamingRequest struct {
	OverseasLabel string `json:"overseas_label"`
	DomesticLabel string `json:"domestic_label"`
}

// putNaming PUT /api/v1/admin/gateway-naming
func (h *GatewayNamingHandler) putNaming(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	var req updateGatewayNamingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}
	req.OverseasLabel = strings.TrimSpace(req.OverseasLabel)
	req.DomesticLabel = strings.TrimSpace(req.DomesticLabel)

	userID := claims.UserID

	// 逐项 UPSERT（留空不修改）
	if req.OverseasLabel != "" {
		if err := repository.UpsertConfigValue(gnOverseasLabelKey, req.OverseasLabel,
			"境外网关对外展示名", userID); err != nil {
			utils.InternalError(w, "保存境外网关展示名失败: "+err.Error())
			return
		}
	}
	if req.DomesticLabel != "" {
		if err := repository.UpsertConfigValue(gnDomesticLabelKey, req.DomesticLabel,
			"境内网关对外展示名", userID); err != nil {
			utils.InternalError(w, "保存境内网关展示名失败: "+err.Error())
			return
		}
	}

	utils.Success(w, gatewayNamingView{
		OverseasLabel: gnReadConfigOrEmpty(gnOverseasLabelKey),
		DomesticLabel: gnReadConfigOrEmpty(gnDomesticLabelKey),
	})
}

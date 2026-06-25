package models

import (
	"strings"
	"time"
)

// ==================== 学校模型策略模型 ====================
//
// 对应数据库 school_model_policies 表。
// 业务规则：
//   - 默认所有学校只能用境内模型（文本=通义千问，图像/视频=豆包）。
//   - 仅被 admin 显式授权（overseas_enabled=true）的学校，才放行境外模型（Claude/Gemini 等）。
//   - 判定境内/境外靠"模型名前缀"，不依赖 api_base_url（百炼网关境内外模型都能中转）。
//   - fail-closed：策略查询失败或无记录时，一律按境内处理。

// SchoolModelPolicy 对应 school_model_policies 表的一行
type SchoolModelPolicy struct {
	SchoolID        string    `json:"school_id"`        // 学校组织ID（organizations.id，type=school）
	OverseasEnabled bool      `json:"overseas_enabled"` // 是否授权使用境外模型
	Note            string    `json:"note"`             // 备注（授权原因/用途）
	GrantedBy       *string   `json:"granted_by"`       // 授权操作人（admin用户ID，可空）
	CreatedAt       time.Time `json:"created_at"`       // 创建时间
	UpdatedAt       time.Time `json:"updated_at"`       // 更新时间
}

// SchoolModelPolicyItem 返回给前端的列表项（带学校名与授权人名）
type SchoolModelPolicyItem struct {
	SchoolID        string    `json:"school_id"`        // 学校组织ID
	SchoolName      string    `json:"school_name"`      // 学校名称（JOIN organizations）
	OverseasEnabled bool      `json:"overseas_enabled"` // 是否授权境外模型
	Note            string    `json:"note"`             // 备注
	GrantedByName   string    `json:"granted_by_name"`  // 授权人显示名（JOIN users，可空时为空串）
	CreatedAt       time.Time `json:"created_at"`       // 创建时间
	UpdatedAt       time.Time `json:"updated_at"`       // 更新时间
}

// ==================== 境内/境外模型判定 ====================

// overseasModelPrefixes 境外模型名前缀清单。
// 模型名（如 "anthropic/claude-opus-4.6"、"google/gemini-3.1-pro-preview"）
// 以这些前缀开头即判为境外模型。新增境外厂商时在此追加。
var overseasModelPrefixes = []string{
	"anthropic/", // Claude 系列
	"google/",    // Gemini 系列
	"openai/",    // GPT 系列
	"x-ai/",      // Grok 系列
	"meta-llama/", // Llama 系列
}

// IsOverseasModel 判断给定模型名是否为境外模型。
// 仅按前缀判定，大小写不敏感；空串视为非境外（境内/未知一律不拦）。
func IsOverseasModel(modelName string) bool {
	if modelName == "" {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(modelName))
	for _, prefix := range overseasModelPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// ==================== 境内降级目标 ====================

// DomesticTextModel 境内文本主力模型——通义千问最高级。
// 非授权学校的所有"文本类"境外模型调用，统一降级到此模型。
// 走全局 ai_configs 的 api_base_url（百炼兼容入口）。
// 如需换型号（如 qwen3-max），只改这一个常量即可。
const DomesticTextModel = "qwen-max"

// ResolveDomesticModel 把一个境外文本模型名映射为境内降级模型。
// 当前所有文本类境外模型统一落到 DomesticTextModel。
// 图像/视频/TTS 场景本就用豆包（境内），不会进入本函数。
// 入参 overseasModel 仅用于将来按原模型做差异化映射（如重活给更强型号）时扩展，
// 当前实现忽略具体值统一返回 qwen-max。
func ResolveDomesticModel(overseasModel string) string {
	return DomesticTextModel
}

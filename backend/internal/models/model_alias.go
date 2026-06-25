package models

import "time"

// ==================== 模型别名映射（批三-2）====================
//
// 对应 model_alias_rules 表。把真实厂商模型名（anthropic/claude-sonnet-4-5、qwen-max 等）
// 映射为业务可读别名（「智学大模型·标准版」等），供老师侧渲染替换，避免暴露真实厂商/模型。
//
// 匹配优先级（见 ResolveModelAlias）：精确(exact) > 前缀(prefix, 最长+最高优先) > 兜底。

// MatchTypeExact / MatchTypePrefix 匹配类型常量
const (
	MatchTypeExact  = "exact"  // 精确匹配：pattern 完全等于模型名
	MatchTypePrefix = "prefix" // 前缀匹配：模型名以 pattern 开头
)

// ModelAliasRule 对应 model_alias_rules 表的一行
type ModelAliasRule struct {
	ID         string    `json:"id"`
	MatchType  string    `json:"match_type"` // exact / prefix
	Pattern    string    `json:"pattern"`    // 模型名或前缀
	Alias      string    `json:"alias"`      // 业务别名
	Priority   int       `json:"priority"`   // 同时命中时大者优先
	Enabled    bool      `json:"enabled"`    // 是否启用
	Note       string    `json:"note"`       // 备注
	CreatedBy  *string   `json:"created_by"` // 创建人（可空）
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// DefaultModelAliasFallback 兜底别名硬编码默认值。
// 实际兜底名存于 ai_configs(model_alias_fallback)，本常量仅在配置缺失时用。
const DefaultModelAliasFallback = "智学大模型"

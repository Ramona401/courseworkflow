package models

import (
	"time"
)

// ==================== AI 全局配置模型 ====================

// AIConfig 对应数据库 ai_configs 表（键值对存储）
type AIConfig struct {
	ID          string     `json:"id"`           // UUID 主键
	ConfigKey   string     `json:"config_key"`   // 配置键名
	ConfigValue string     `json:"config_value"` // 配置值（api_key_enc 为密文）
	Description string     `json:"description"`  // 配置描述
	UpdatedBy   *string    `json:"updated_by"`   // 最后更新者ID
	UpdatedAt   *time.Time `json:"updated_at"`   // 最后更新时间
}

// AIConfigItem 返回给前端的单条配置（API Key 脱敏）
type AIConfigItem struct {
	ConfigKey   string     `json:"config_key"`   // 配置键名
	ConfigValue string     `json:"config_value"` // 配置值（API Key 已脱敏）
	Description string     `json:"description"`  // 配置描述
	UpdatedAt   *time.Time `json:"updated_at"`   // 最后更新时间
}

// GlobalConfigResponse 全局配置响应（聚合为一个对象）
type GlobalConfigResponse struct {
	APIBaseURL   string     `json:"api_base_url"`  // AI API 基础地址
	APIKey       string     `json:"api_key"`        // API Key（脱敏显示）
	APIKeySet    bool       `json:"api_key_set"`    // API Key 是否已配置
	DefaultModel string     `json:"default_model"`  // 默认模型
	Temperature  string     `json:"temperature"`    // 默认温度
	MaxTokens    string     `json:"max_tokens"`     // 默认最大Token数
	UpdatedAt    *time.Time `json:"updated_at"`     // 最近更新时间
}

// UpdateGlobalConfigRequest 更新全局配置请求体
type UpdateGlobalConfigRequest struct {
	APIBaseURL   string `json:"api_base_url"`  // AI API 基础地址
	APIKey       string `json:"api_key"`        // API Key（明文，后端加密存储；空字符串表示不修改）
	DefaultModel string `json:"default_model"`  // 默认模型
	Temperature  string `json:"temperature"`    // 默认温度（字符串，如 "0.7"）
	MaxTokens    string `json:"max_tokens"`     // 默认最大Token数（字符串，如 "8000"）
}

// ==================== AI 场景配置模型 ====================

// AISceneConfig 对应数据库 ai_scene_configs 表
type AISceneConfig struct {
	ID             string     `json:"id"`               // UUID 主键
	SceneCode      string     `json:"scene_code"`       // 场景代码
	Model          *string    `json:"model"`            // 模型（可为空，继承全局）
	Temperature    *float64   `json:"temperature"`      // 温度（可为空，继承全局）
	MaxTokens      *int       `json:"max_tokens"`       // 最大Token（可为空，继承全局）
	SystemPromptID *string    `json:"system_prompt_id"` // 关联提示词ID
	IsActive       bool       `json:"is_active"`        // 是否启用
	UpdatedBy      *string    `json:"updated_by"`       // 最后更新者ID
	UpdatedAt      *time.Time `json:"updated_at"`       // 最后更新时间
}

// SceneConfigResponse 返回给前端的场景配置（含场景中文名）
type SceneConfigResponse struct {
	ID             string     `json:"id"`               // UUID
	SceneCode      string     `json:"scene_code"`       // 场景代码
	SceneName      string     `json:"scene_name"`       // 场景中文名
	Model          *string    `json:"model"`            // 模型
	Temperature    *float64   `json:"temperature"`      // 温度
	MaxTokens      *int       `json:"max_tokens"`       // 最大Token
	SystemPromptID *string    `json:"system_prompt_id"` // 关联提示词ID
	IsActive       bool       `json:"is_active"`        // 是否启用
	UpdatedAt      *time.Time `json:"updated_at"`       // 最后更新时间
}

// UpdateSceneConfigRequest 更新场景配置请求体
type UpdateSceneConfigRequest struct {
	Model          *string  `json:"model"`            // 模型（null表示继承全局）
	Temperature    *float64 `json:"temperature"`      // 温度（null表示继承全局）
	MaxTokens      *int     `json:"max_tokens"`       // 最大Token（null表示继承全局）
	SystemPromptID *string  `json:"system_prompt_id"` // 关联提示词ID
	IsActive       *bool    `json:"is_active"`        // 是否启用
}

// ==================== 场景代码常量与映射 ====================

// 场景代码常量
// 核心6个场景对应Pipeline的6个AI步骤
// Generator额外拆分为3个子场景：modify(Sonnet)/create(Opus)/merge(Opus)
const (
	SceneScanner         = "scanner"          // 扫描定位（Prompt A）→ Sonnet
	SceneEvaluator       = "evaluator"        // 评估打分（Prompt B）→ Opus
	SceneMeta            = "meta"             // 元评估仲裁（Prompt E）→ Opus
	SceneTranslator      = "translator"       // 翻译转换（Prompt C）→ Opus
	SceneReviewer        = "reviewer"         // 审核检查（Prompt D）→ Opus
	SceneGenerator       = "generator"        // 页面生成-修改（Prompt F）→ Sonnet
	SceneGeneratorCreate = "generator_create" // 页面生成-新增（Prompt F）→ Opus（从零创建）
	SceneGeneratorMerge  = "generator_merge"  // 页面生成-合并（Prompt F）→ Opus（复杂合并）
)

// ValidSceneCodes 有效场景代码列表（含新增的generator子场景）
var ValidSceneCodes = []string{
	SceneScanner, SceneEvaluator, SceneMeta,
	SceneTranslator, SceneReviewer,
	SceneGenerator, SceneGeneratorCreate, SceneGeneratorMerge,
}

// SceneNameMap 场景代码→中文名映射
var SceneNameMap = map[string]string{
	SceneScanner:         "扫描定位",
	SceneEvaluator:       "评估打分",
	SceneMeta:            "元评估仲裁",
	SceneTranslator:      "翻译转换",
	SceneReviewer:        "审核检查",
	SceneGenerator:       "页面生成-修改",
	SceneGeneratorCreate: "页面生成-新增",
	SceneGeneratorMerge:  "页面生成-合并",
}

// IsValidSceneCode 检查场景代码是否有效
func IsValidSceneCode(code string) bool {
	for _, c := range ValidSceneCodes {
		if c == code {
			return true
		}
	}
	return false
}

// ==================== 全局配置键名常量 ====================

const (
	ConfigKeyAPIBaseURL   = "api_base_url"  // AI API 基础地址
	ConfigKeyAPIKeyEnc    = "api_key_enc"   // API Key（AES加密存储）
	ConfigKeyDefaultModel = "default_model" // 默认模型
	ConfigKeyTemperature  = "temperature"   // 默认温度
	ConfigKeyMaxTokens    = "max_tokens"    // 默认最大Token数
)

// ConfigKeyDescriptions 配置键名→描述映射
var ConfigKeyDescriptions = map[string]string{
	ConfigKeyAPIBaseURL:   "AI API 基础地址",
	ConfigKeyAPIKeyEnc:    "API Key（管理界面配置）",
	ConfigKeyDefaultModel: "默认模型",
	ConfigKeyTemperature:  "默认温度",
	ConfigKeyMaxTokens:    "默认最大Token数",
}

package models

import (
	"encoding/json"
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
	APIBaseURL   string     `json:"api_base_url"`   // AI API 基础地址
	APIKey       string     `json:"api_key"`        // API Key（脱敏显示）
	APIKeySet    bool       `json:"api_key_set"`    // API Key 是否已配置
	DefaultModel string     `json:"default_model"`  // 默认模型
	Temperature  string     `json:"temperature"`    // 默认温度
	MaxTokens    string     `json:"max_tokens"`     // 默认最大Token数
	UpdatedAt    *time.Time `json:"updated_at"`     // 最近更新时间
}

// UpdateGlobalConfigRequest 更新全局配置请求体
type UpdateGlobalConfigRequest struct {
	APIBaseURL   string `json:"api_base_url"`   // AI API 基础地址
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
	FallbackModels []string   `json:"-"`                // v85新增：降级模型列表（从JSONB解析）
}

// ParseFallbackModels 从原始JSONB字节解析降级模型列表
// 在repository层Scan后调用
func ParseFallbackModels(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var models []string
	if err := json.Unmarshal(raw, &models); err != nil {
		return nil
	}
	return models
}

// SceneConfigResponse 返回给前端的场景配置（含中文名）
type SceneConfigResponse struct {
	ID             string     `json:"id"`               // UUID
	SceneCode      string     `json:"scene_code"`       // 场景代码
	SceneName      string     `json:"scene_name"`       // 场景中文名
	SceneGroup     string     `json:"scene_group"`      // 场景分组：lesson_plan=教案备课 / pipeline=Pipeline
	Model          *string    `json:"model"`            // 模型
	Temperature    *float64   `json:"temperature"`      // 温度
	MaxTokens      *int       `json:"max_tokens"`       // 最大Token
	SystemPromptID *string    `json:"system_prompt_id"` // 关联提示词ID
	IsActive       bool       `json:"is_active"`        // 是否启用
	FallbackModels []string   `json:"fallback_models"`  // v85新增：降级模型列表
	UpdatedAt      *time.Time `json:"updated_at"`       // 最后更新时间
}

// UpdateSceneConfigRequest 更新场景配置请求体
type UpdateSceneConfigRequest struct {
	Model          *string  `json:"model"`            // 模型（null表示继承全局）
	Temperature    *float64 `json:"temperature"`      // 温度（null表示继承全局）
	MaxTokens      *int     `json:"max_tokens"`       // 最大Token（null表示继承全局）
	SystemPromptID *string  `json:"system_prompt_id"` // 关联提示词ID
	IsActive       *bool    `json:"is_active"`        // 是否启用
	FallbackModels []string `json:"fallback_models"`  // v85新增：降级模型列表（nil表示不修改）
}

// ==================== 场景代码常量与映射 ====================

// 场景代码常量
// 核心6个场景对应Pipeline的6个AI步骤
// Generator额外拆分为3个子场景：modify(Sonnet)/create(Opus)/merge(Opus)
const (
	SceneScanner         = "scanner"          // 扫描定位（Prompt A）→ Haiku
	SceneEvaluator       = "evaluator"        // 评估打分（Prompt B）→ Opus
	SceneMeta            = "meta"             // 元评估仲裁（Prompt E）→ Opus
	SceneTranslator      = "translator"       // 翻译转换（Prompt C）→ Sonnet
	SceneReviewer        = "reviewer"         // 审核检查（Prompt D）→ Sonnet
	SceneGenerator       = "generator"        // 页面生成-修改（Prompt F）→ Sonnet
	SceneGeneratorCreate = "generator_create" // 页面生成-新增（Prompt F）→ Opus（从零创建）
	SceneGeneratorMerge  = "generator_merge"  // 页面生成-合并（Prompt F）→ Opus（复杂合并）
	SceneAIFix           = "ai_fix"           // AI快修（Sonnet）
)

// 教案备课场景代码常量（v78新增）
const (
	SceneLessonPlan = "lesson_plan" // 教案备课对话
)

// v87新增：AI教练场景代码常量
const (
	SceneStageCoach = "stage_coach" // 阶段教练评估（Haiku，低成本）
)

// v114新增：AI 助手对话式创作场景代码常量(TE-DNA 3.0 P0.5)
// 对应 services/assistant_designer_service.go 的两阶段 AI 调用
// 从 lesson_plan 场景中独立出来,便于单独调模型/温度/降级链(管理员可在 AI 管理中心前端界面直接调整,无需改代码)
const (
	SceneAssistantDesigner = "assistant_designer" // AI 助手对话式创作(Sonnet,可在管理界面切换)

	// 课件工坊场景代码常量
	SceneCWNavRefine       = "courseware_nav_refine"      // 课件导航栏样式微调
	SceneCWPageRefine      = "courseware_page_refine"     // 课件单页内容微调
	SceneCWIndex           = "courseware_index"            // 课件索引生成
	SceneCWScheme          = "courseware_scheme"           // 课件方案翻译
	SceneCWGenerate        = "courseware_generate"         // 课件HTML生成
	SceneCWTemplateExtract = "courseware_template_extract" // 课件模板AI提取
	SceneCWTemplateRefine  = "courseware_template_refine"  // 课件模板AI微调
)

// v0.41新增：v0.42预留场景代码常量（多入口+多媒体阶段启用）
const (
	SceneCWImageGen    = "courseware_image_gen"    // 课件图片生成提示词优化（Haiku）
	SceneCWPPTExtract  = "courseware_ppt_extract"  // PPT内容提取与索引生成（Sonnet）
	SceneCWTopicDirect = "courseware_topic_direct" // 主题直接生成课件索引（Sonnet）
)

// v0.42.1新增：视频生成场景代码常量
const (
	SceneCWVideoGen = "courseware_video_gen" // 课件视频生成（豆包Seedance）
)

// v0.42.9新增：TTS语音合成场景代码常量
const (
	SceneCWSubtitleTTS = "courseware_subtitle_tts" // 课件字幕TTS配音（豆包seed-tts-2.0）
)

// v0.42.11新增：3D单页课件生成场景代码常量
const (
	SceneCW3DSingle = "courseware_3d_single" // 3D互动单页课件生成（Claude Sonnet 4.6）
)

// ValidSceneCodes 有效场景代码列表
// v87新增 stage_coach;v114新增 assistant_designer
// v0.41新增 courseware_image_gen/courseware_ppt_extract/courseware_topic_direct（预留v0.42）
// v0.42.1新增 courseware_video_gen
// v0.42.11新增 courseware_3d_single
var ValidSceneCodes = []string{
	SceneScanner, SceneEvaluator, SceneMeta,
	SceneTranslator, SceneReviewer,
	SceneGenerator, SceneGeneratorCreate, SceneGeneratorMerge,
	SceneAIFix, SceneLessonPlan,
	SceneStageCoach,
	SceneAssistantDesigner,
	SceneCWNavRefine, SceneCWPageRefine,
	SceneCWIndex, SceneCWScheme, SceneCWGenerate,
	SceneCWTemplateExtract, SceneCWTemplateRefine,
	// v0.41 预留（v0.42 启用）
	SceneCWImageGen, SceneCWPPTExtract, SceneCWTopicDirect,
	// v0.42.1 新增
	SceneCWVideoGen,
	// v0.42.9 新增
	SceneCWSubtitleTTS,
	// v0.42.11 新增
	SceneCW3DSingle,
}

// SceneNameMap 场景代码→中文名映射
// v0.41新增3个预留场景的中文名
// v0.42.1新增 courseware_video_gen
// v0.42.11新增 courseware_3d_single
var SceneNameMap = map[string]string{
	SceneScanner:           "扫描定位",
	SceneEvaluator:         "评估打分",
	SceneMeta:              "元评估仲裁",
	SceneTranslator:        "翻译转换",
	SceneReviewer:          "审核检查",
	SceneGenerator:         "页面生成-修改",
	SceneGeneratorCreate:   "页面生成-新增",
	SceneGeneratorMerge:    "页面生成-合并",
	SceneAIFix:             "AI快修",
	SceneLessonPlan:        "教案备课对话",
	SceneStageCoach:        "阶段教练评估",
	SceneAssistantDesigner: "AI助手对话式创作",
	SceneCWIndex:           "课件索引生成",
	SceneCWScheme:          "课件方案翻译",
	SceneCWGenerate:        "课件HTML生成",
	SceneCWNavRefine:       "课件导航栏微调",
	SceneCWPageRefine:      "课件单页微调",
	SceneCWTemplateExtract: "课件模板AI提取",
	SceneCWTemplateRefine:  "课件模板AI微调",
	// v0.41 预留（v0.42 启用）
	SceneCWImageGen:    "课件图片生成",
	SceneCWPPTExtract:  "PPT内容提取",
	SceneCWTopicDirect: "主题直接生成课件",
	// v0.42.1 新增
	SceneCWVideoGen: "课件视频生成",
	// v0.42.9 新增
	SceneCWSubtitleTTS: "课件字幕TTS配音",
	// v0.42.11 新增
	SceneCW3DSingle: "3D互动单页生成",
}

// SceneGroupMap 场景代码→分组映射（v78新增，v87补充stage_coach，v114补充 assistant_designer）
// 归入 lesson_plan 组:Designer 也服务于教案备课生态(老师创建 AI 助手→助手帮教师备课)
// v0.41新增3个预留场景归入 courseware 组
// v0.42.1新增 courseware_video_gen 归入 courseware 组
// v0.42.11新增 courseware_3d_single 归入 courseware 组
var SceneGroupMap = map[string]string{
	SceneScanner:           "pipeline",
	SceneEvaluator:         "pipeline",
	SceneMeta:              "pipeline",
	SceneTranslator:        "pipeline",
	SceneReviewer:          "pipeline",
	SceneGenerator:         "pipeline",
	SceneGeneratorCreate:   "pipeline",
	SceneGeneratorMerge:    "pipeline",
	SceneAIFix:             "pipeline",
	SceneLessonPlan:        "lesson_plan",
	SceneStageCoach:        "lesson_plan",
	SceneAssistantDesigner: "lesson_plan",
	SceneCWIndex:           "courseware",
	SceneCWScheme:          "courseware",
	SceneCWGenerate:        "courseware",
	SceneCWNavRefine:       "courseware",
	SceneCWPageRefine:      "courseware",
	SceneCWTemplateExtract: "courseware",
	SceneCWTemplateRefine:  "courseware",
	// v0.41 预留（v0.42 启用）
	SceneCWImageGen:    "courseware",
	SceneCWPPTExtract:  "courseware",
	SceneCWTopicDirect: "courseware",
	// v0.42.1 新增
	SceneCWVideoGen: "courseware",
	// v0.42.9 新增
	SceneCWSubtitleTTS: "courseware",
	// v0.42.11 新增
	SceneCW3DSingle: "courseware",
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

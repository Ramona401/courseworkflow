package models

import (
        "encoding/json"
        "time"
)

// ==================== AI 全局配置模型 ====================

// AIConfig 对应数据库 ai_configs 表（键值对存储）
type AIConfig struct {
        ID          string     `json:"id"`
        ConfigKey   string     `json:"config_key"`
        ConfigValue string     `json:"config_value"`
        Description string     `json:"description"`
        UpdatedBy   *string    `json:"updated_by"`
        UpdatedAt   *time.Time `json:"updated_at"`
}

// AIConfigItem 返回给前端的单条配置（API Key脱敏）
type AIConfigItem struct {
        ConfigKey   string     `json:"config_key"`
        ConfigValue string     `json:"config_value"`
        Description string     `json:"description"`
        UpdatedAt   *time.Time `json:"updated_at"`
}

// GlobalConfigResponse 全局配置响应
type GlobalConfigResponse struct {
        APIBaseURL   string     `json:"api_base_url"`
        APIKey       string     `json:"api_key"`
        APIKeySet    bool       `json:"api_key_set"`
        DefaultModel string     `json:"default_model"`
        Temperature  string     `json:"temperature"`
        MaxTokens    string     `json:"max_tokens"`
        UpdatedAt    *time.Time `json:"updated_at"`
}

// UpdateGlobalConfigRequest 更新全局配置请求体
type UpdateGlobalConfigRequest struct {
        APIBaseURL   string `json:"api_base_url"`
        APIKey       string `json:"api_key"`
        DefaultModel string `json:"default_model"`
        Temperature  string `json:"temperature"`
        MaxTokens    string `json:"max_tokens"`
}

// ==================== AI 场景配置模型 ====================

// AISceneConfig 对应数据库 ai_scene_configs 表
type AISceneConfig struct {
        ID             string     `json:"id"`
        SceneCode      string     `json:"scene_code"`
        Model          *string    `json:"model"`
        Temperature    *float64   `json:"temperature"`
        MaxTokens      *int       `json:"max_tokens"`
        SystemPromptID *string    `json:"system_prompt_id"`
        IsActive       bool       `json:"is_active"`
        UpdatedBy      *string    `json:"updated_by"`
        UpdatedAt      *time.Time `json:"updated_at"`
        FallbackModels []string   `json:"-"`
}

// ParseFallbackModels 从原始JSONB解析降级模型列表
func ParseFallbackModels(raw []byte) []string {
        if len(raw) == 0 {
                return nil
        }

        var models []string
        if err := json.Unmarshal(
                raw,
                &models,
        ); err != nil {
                return nil
        }

        return models
}

// SceneConfigResponse 返回给前端的场景配置
type SceneConfigResponse struct {
        ID             string     `json:"id"`
        SceneCode      string     `json:"scene_code"`
        SceneName      string     `json:"scene_name"`
        SceneGroup     string     `json:"scene_group"`
        Model          *string    `json:"model"`
        Temperature    *float64   `json:"temperature"`
        MaxTokens      *int       `json:"max_tokens"`
        SystemPromptID *string    `json:"system_prompt_id"`
        IsActive       bool       `json:"is_active"`
        FallbackModels []string   `json:"fallback_models"`
        UpdatedAt      *time.Time `json:"updated_at"`
}

// UpdateSceneConfigRequest 更新场景配置请求体
type UpdateSceneConfigRequest struct {
        Model          *string  `json:"model"`
        Temperature    *float64 `json:"temperature"`
        MaxTokens      *int     `json:"max_tokens"`
        SystemPromptID *string  `json:"system_prompt_id"`
        IsActive       *bool    `json:"is_active"`
        FallbackModels []string `json:"fallback_models"`
}

// ==================== 场景代码常量与映射 ====================

// Pipeline场景
const (
        SceneScanner         = "scanner"
        SceneEvaluator       = "evaluator"
        SceneMeta            = "meta"
        SceneTranslator      = "translator"
        SceneReviewer        = "reviewer"
        SceneGenerator       = "generator"
        SceneGeneratorCreate = "generator_create"
        SceneGeneratorMerge  = "generator_merge"
        SceneAIFix           = "ai_fix"
)

// 教案备课场景
const (
        // SceneLessonPlan 是教师正式备课生成场景。
        SceneLessonPlan = "lesson_plan"

        // SceneLessonPlanHarness 是课程大纲展示前Judge和平台自动修正场景。
        //
        // 未在ai_scene_configs中单独配置时，GetEffectiveConfig自动继承全局配置；
        // 后续管理员可为该场景独立指定低温度Judge模型和Fallback链。
        SceneLessonPlanHarness = "lesson_plan_harness"
)

// AI教练场景
const (
        SceneStageCoach = "stage_coach"
)

// AI助手与课件工坊场景
const (
        SceneAssistantDesigner = "assistant_designer"

        SceneCWNavRefine       = "courseware_nav_refine"
        SceneCWPageRefine      = "courseware_page_refine"
        SceneCWIndex           = "courseware_index"
        SceneCWScheme          = "courseware_scheme"
        SceneCWGenerate        = "courseware_generate"
        SceneCWTemplateExtract = "courseware_template_extract"
        SceneCWTemplateRefine  = "courseware_template_refine"
)

// 课件多入口与媒体场景
const (
        SceneCWImageGen    = "courseware_image_gen"
        SceneCWPPTExtract  = "courseware_ppt_extract"
        SceneCWTopicDirect = "courseware_topic_direct"
        SceneCWVideoGen    = "courseware_video_gen"
        SceneCWSubtitleTTS = "courseware_subtitle_tts"
        SceneCW3DSingle    = "courseware_3d_single"
)

// ValidSceneCodes 有效场景代码列表。
var ValidSceneCodes = []string{
        SceneScanner,
        SceneEvaluator,
        SceneMeta,
        SceneTranslator,
        SceneReviewer,
        SceneGenerator,
        SceneGeneratorCreate,
        SceneGeneratorMerge,
        SceneAIFix,

        SceneLessonPlan,
        SceneLessonPlanHarness,
        SceneStageCoach,
        SceneAssistantDesigner,

        SceneCWNavRefine,
        SceneCWPageRefine,
        SceneCWIndex,
        SceneCWScheme,
        SceneCWGenerate,
        SceneCWTemplateExtract,
        SceneCWTemplateRefine,
        SceneCWImageGen,
        SceneCWPPTExtract,
        SceneCWTopicDirect,
        SceneCWVideoGen,
        SceneCWSubtitleTTS,
        SceneCW3DSingle,

        "kb_extract",
        "kb_compress",
        "kb_arbitrate",
}

// SceneNameMap 场景代码到中文名映射。
var SceneNameMap = map[string]string{
        SceneScanner:         "扫描定位",
        SceneEvaluator:       "评估打分",
        SceneMeta:            "元评估仲裁",
        SceneTranslator:      "翻译转换",
        SceneReviewer:        "审核检查",
        SceneGenerator:       "页面生成-修改",
        SceneGeneratorCreate: "页面生成-新增",
        SceneGeneratorMerge:  "页面生成-合并",
        SceneAIFix:           "AI快修",

        SceneLessonPlan:        "教案备课对话",
        SceneLessonPlanHarness: "教案课程大纲Harness",
        SceneStageCoach:        "阶段教练评估",
        SceneAssistantDesigner: "AI助手对话式创作",

        SceneCWIndex:           "课件索引生成",
        SceneCWScheme:          "课件方案翻译",
        SceneCWGenerate:        "课件HTML生成",
        SceneCWNavRefine:       "课件导航栏微调",
        SceneCWPageRefine:      "课件单页微调",
        SceneCWTemplateExtract: "课件模板AI提取",
        SceneCWTemplateRefine:  "课件模板AI微调",
        SceneCWImageGen:        "课件图片生成",
        SceneCWPPTExtract:      "PPT内容提取",
        SceneCWTopicDirect:     "主题直接生成课件",
        SceneCWVideoGen:        "课件视频生成",
        SceneCWSubtitleTTS:     "课件字幕TTS配音",
        SceneCW3DSingle:        "3D互动单页生成",

        "kb_extract":   "知识库-知识点抽取",
        "kb_compress":  "知识库-课标压缩",
        "kb_arbitrate": "知识库-语义仲裁",
}

// SceneGroupMap 场景代码到管理界面分组映射。
var SceneGroupMap = map[string]string{
        SceneScanner:         "pipeline",
        SceneEvaluator:       "pipeline",
        SceneMeta:            "pipeline",
        SceneTranslator:      "pipeline",
        SceneReviewer:        "pipeline",
        SceneGenerator:       "pipeline",
        SceneGeneratorCreate: "pipeline",
        SceneGeneratorMerge:  "pipeline",
        SceneAIFix:           "pipeline",

        SceneLessonPlan:        "lesson_plan",
        SceneLessonPlanHarness: "lesson_plan",
        SceneStageCoach:        "lesson_plan",
        SceneAssistantDesigner: "lesson_plan",

        SceneCWIndex:           "courseware",
        SceneCWScheme:          "courseware",
        SceneCWGenerate:        "courseware",
        SceneCWNavRefine:       "courseware",
        SceneCWPageRefine:      "courseware",
        SceneCWTemplateExtract: "courseware",
        SceneCWTemplateRefine:  "courseware",
        SceneCWImageGen:        "courseware",
        SceneCWPPTExtract:      "courseware",
        SceneCWTopicDirect:     "courseware",
        SceneCWVideoGen:        "courseware",
        SceneCWSubtitleTTS:     "courseware",
        SceneCW3DSingle:        "courseware",

        "kb_extract":   "knowledge_base",
        "kb_compress":  "knowledge_base",
        "kb_arbitrate": "knowledge_base",
}

// IsValidSceneCode 检查场景代码是否合法。
func IsValidSceneCode(code string) bool {
        for _, validCode := range ValidSceneCodes {
                if validCode == code {
                        return true
                }
        }

        return false
}

// ==================== 全局配置键名常量 ====================

const (
        ConfigKeyAPIBaseURL   = "api_base_url"
        ConfigKeyAPIKeyEnc    = "api_key_enc"
        ConfigKeyDefaultModel = "default_model"
        ConfigKeyTemperature  = "temperature"
        ConfigKeyMaxTokens    = "max_tokens"
)

// ConfigKeyDescriptions 配置键名到中文说明。
var ConfigKeyDescriptions = map[string]string{
        ConfigKeyAPIBaseURL:   "AI API 基础地址",
        ConfigKeyAPIKeyEnc:    "API Key（管理界面配置）",
        ConfigKeyDefaultModel: "默认模型",
        ConfigKeyTemperature:  "默认温度",
        ConfigKeyMaxTokens:    "默认最大Token数",
}

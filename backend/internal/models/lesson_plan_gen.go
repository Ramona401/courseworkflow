package models

// ==================== 教案生成相关模型 ====================
// Phase3：教案生成核心流程
// Phase5：新增萃取提示事件类型（extraction_hint）
// Phase 7A：StartConversationRequest 新增 RecipeID 字段
// Phase 7B-8：LPSSEEvent 新增 StageData 字段，支持阶段化SSE事件推送

import "time"

// ==================== 会话消息模型 ====================

type ConversationRole string

const (
	ConvRoleUser      ConversationRole = "user"
	ConvRoleAssistant ConversationRole = "assistant"
	ConvRoleSystem    ConversationRole = "system"
)

type ConvMsgType string

const (
	ConvMsgTypeText       ConvMsgType = "text"
	ConvMsgTypeOptions    ConvMsgType = "options"
	ConvMsgTypeComponents ConvMsgType = "components"
	ConvMsgTypeGenerate   ConvMsgType = "generate"
	ConvMsgTypeContent    ConvMsgType = "content"
	ConvMsgTypeReview     ConvMsgType = "review"
	ConvMsgTypeAction     ConvMsgType = "action"
)

// ConversationMessage 单条会话消息
type ConversationMessage struct {
	ID         string                 `json:"id"`
	Role       ConversationRole       `json:"role"`
	Type       ConvMsgType            `json:"type"`
	Content    string                 `json:"content"`
	Options    []ConvOption           `json:"options,omitempty"`
	Components []ConvComponent        `json:"components,omitempty"`
	Actions    []ConvAction           `json:"actions,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

type ConvOption struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Emoji    string `json:"emoji"`
	Selected bool   `json:"selected"`
}

type ConvComponent struct {
	ID             string  `json:"id"`
	LibraryType    string  `json:"library_type"`
	DisplayLabel   string  `json:"display_label"`
	DesignLogic    string  `json:"design_logic"`
	ExampleSnippet string  `json:"example_snippet"`
	QualityScore   float64 `json:"quality_score"`
	UsageCount     int     `json:"usage_count"`
	Selected       bool    `json:"selected"`
}

type ConvAction struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Style string `json:"style"`
}

// ==================== 生成会话请求/响应 ====================

// StartConversationRequest 开始备课会话请求
// Phase 7A：新增 RecipeID 字段，选配方后AI带着全局知识工作
type StartConversationRequest struct {
	Subject         string `json:"subject"`          // 学科（必填）
	Grade           string `json:"grade"`            // 年级（必填）
	Topic           string `json:"topic"`            // 课题（必填）
	DurationMinutes int    `json:"duration_minutes"` // 课时时长（可选，默认45）
	TemplateID      string `json:"template_id"`      // 提示词模板ID（可选）
	GroupID         string `json:"group_id"`         // 教研组ID（可选）
	RecipeID        string `json:"recipe_id"`        // 备课配方ID（可选，Phase 7A新增）
}

type LessonPlanChatRequest struct {
	PlanID             string   `json:"plan_id"`
	Message            string   `json:"message"`
	SelectedOptions    []string `json:"selected_options,omitempty"`
	SelectedComponents []string `json:"selected_components,omitempty"`
	CurrentSection     string   `json:"current_section,omitempty"`
}

type GenerateSectionRequest struct {
	PlanID               string   `json:"plan_id"`
	Section              string   `json:"section"`
	UserRequirement      string   `json:"user_requirement"`
	SelectedComponentIDs []string `json:"selected_component_ids"`
	Stream               bool     `json:"stream"`
}

type ApplyAISuggestionsRequest struct {
	PlanID      string   `json:"plan_id"`
	Suggestions []string `json:"suggestions"`
}

// ==================== AI评审模型 ====================

type AIReviewDimension struct {
	Code    string  `json:"code"`
	Name    string  `json:"name"`
	Score   float64 `json:"score"`
	Comment string  `json:"comment"`
	Good    bool    `json:"good"`
}

type AIReviewResult struct {
	TotalScore   float64               `json:"total_score"`
	GoodPoints   []string              `json:"good_points"`
	Improvements []AIReviewImprovement `json:"improvements"`
	Dimensions   []AIReviewDimension   `json:"dimensions"`
	Summary      string                `json:"summary"`
	ReviewedAt   time.Time             `json:"reviewed_at"`
}

type AIReviewImprovement struct {
	ID         string `json:"id"`
	Issue      string `json:"issue"`
	Suggestion string `json:"suggestion"`
	Section    string `json:"section"`
	Applied    bool   `json:"applied"`
}

type TriggerAIReviewRequest struct {
	PlanID string `json:"plan_id"`
}

// ==================== SSE推送事件 ====================

type LPSSEEventType string

const (
	LPSSEConnected      LPSSEEventType = "connected"
	LPSSEThinking       LPSSEEventType = "thinking"
	LPSSEChunk          LPSSEEventType = "chunk"
	LPSSEMessageDone    LPSSEEventType = "message_done"
	LPSSEContentUpdate  LPSSEEventType = "content_update"
	LPSSEReviewDone     LPSSEEventType = "review_done"
	LPSSEExtractionHint LPSSEEventType = "extraction_hint"
	LPSSEError          LPSSEEventType = "error"
	LPSSEDone           LPSSEEventType = "done"
)

// ExtractionHint Phase5萃取提示数据
type ExtractionHint struct {
	HintID         string `json:"hint_id"`
	DisplayText    string `json:"display_text"`
	SourceContent  string `json:"source_content"`
	ExtractionType string `json:"extraction_type"`
	PlanID         string `json:"plan_id"`
}

// LPSSEEvent 教案SSE推送事件结构体
// Phase 7B-8新增：StageData字段，承载阶段化备课工坊的stage_started/stage_complete/stage_output事件数据
type LPSSEEvent struct {
	EventType      LPSSEEventType       `json:"type"`
	PlanID         string               `json:"plan_id"`
	MessageID      string               `json:"message_id,omitempty"`
	Chunk          string               `json:"chunk,omitempty"`
	Message        *ConversationMessage `json:"message,omitempty"`
	Content        string               `json:"content,omitempty"`
	Review         *AIReviewResult      `json:"review,omitempty"`
	ExtractionHint *ExtractionHint      `json:"extraction_hint,omitempty"`
	StageData      *StageEventData      `json:"stage_data,omitempty"` // Phase 7B-8：阶段事件数据
	Error          string               `json:"error,omitempty"`
}

// ==================== 生成步骤枚举 ====================

type LPGenStep string

const (
	LPGenStepInit      LPGenStep = "init"
	LPGenStepCollect   LPGenStep = "collect"
	LPGenStepRecommend LPGenStep = "recommend"
	LPGenStepGenerate  LPGenStep = "generate"
	LPGenStepReview    LPGenStep = "review"
	LPGenStepDone      LPGenStep = "done"
)

// ==================== Phase5：萃取确认请求 ====================

type ConfirmExtractionRequest struct {
	Decision string `json:"decision"`
}

type ExtractionListItem struct {
	ID             string `json:"id"`
	SourceType     string `json:"source_type"`
	SourceContent  string `json:"source_content"`
	ExtractionType string `json:"extraction_type"`
	LibraryName    string `json:"library_name"`
	Status         string `json:"status"`
	PlanTitle      string `json:"plan_title"`
	CreatedByName  string `json:"created_by_name"`
	CreatedAt      string `json:"created_at"`
}

type ExtractionListResponse struct {
	Extractions []*ExtractionListItem `json:"extractions"`
	Total       int                   `json:"total"`
}

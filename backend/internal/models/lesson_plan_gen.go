package models

// ==================== 教案生成相关模型 ====================
// Phase3：教案生成核心流程
// Phase5：新增萃取提示事件类型（extraction_hint）
// Phase 7A：StartConversationRequest 新增 RecipeID 字段
// Phase 7B-8：LPSSEEvent 新增 StageData 字段，支持阶段化SSE事件推送
// v110(TE-DNA 3.0 P0 STEP 3)：LessonPlanChatRequest 新增 AssistantID 字段
//   前端若选中 AI 助手,透过此字段透传到 service 层,用助手 full_prompt 替换第4层阶段角色
// v168(第二批治本·功能B·一键生成完整教案)：LessonPlanChatRequest 新增 FullGenerate 字段
//   前端在 write 阶段点击「⚡一键生成完整教案」时置 true,后端据此在 write 阶段
//   注入"全委托一次性出稿"指令,让 AI 一次性产出完整教案正文(不依赖逐轮"继续"),
//   产出仍走现有 handleWriteStageOutput 链路落库。默认 false 时行为 100% 不变。
// 参考资料附件(PDF/Word)：LessonPlanChatRequest 新增 RefMaterial 字段
//   老师在对话框上传 PDF/Word 后,前端在浏览器端提取文字(短文档=原文,长文档=经压缩端点
//   压成结构化要点),把"最终注入文本"放进本字段,每轮 chat 都携带(会话级参考资料)。
//   后端在 processChatStageAsync 注入前把它拼成"参考资料"块置于 system prompt(并做防御性
//   截断兜底)。不落库、不复用——纯请求级透传,老师移除附件或关闭对话即失。默认空串时行为不变。

import "time"

// RecipeSelectionMode 表示开始备课时老师对配方的选择方式。
//
// 三态语义：
//   - auto：由平台根据学校默认、教研组和学科规则自动选择；
//   - selected：老师明确选择了 recipe_id；
//   - none：老师明确要求不使用配方，只使用系统阶段骨架。
//
// 兼容规则：旧客户端不传 recipe_mode 时，有 recipe_id 按 selected 处理，
// 没有 recipe_id 按 auto 处理，保证已有专家模式和对话模式请求不回归。
type RecipeSelectionMode string

const (
	RecipeSelectionModeAuto     RecipeSelectionMode = "auto"
	RecipeSelectionModeSelected RecipeSelectionMode = "selected"
	RecipeSelectionModeNone     RecipeSelectionMode = "none"
)

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

// StartConversationRequest 开始备课会话请求。
//
// CourseOutlineID是老师在首屏明确选择的唯一课程大纲ID。
// 空字符串表示不关联；非空时必须在任何教案INSERT前完成可见性、教育域、
// 学科和具体年级校验，并与教案在同一INSERT事务中原子写入。
type StartConversationRequest struct {
	Subject         string              `json:"subject"`                     // 学科（必填）
	Grade           string              `json:"grade"`                       // 年级（必填）
	Topic           string              `json:"topic"`                       // 课题（必填）
	DurationMinutes int                 `json:"duration_minutes"`            // 课时时长（可选，默认45）
	TemplateID      string              `json:"template_id"`                 // 提示词模板ID（可选）
	GroupID         string              `json:"group_id"`                    // 教研组ID（可选）
	RecipeID        string              `json:"recipe_id"`                   // 备课配方ID（可选）
	RecipeMode      RecipeSelectionMode `json:"recipe_mode,omitempty"`       // 配方选择三态
	TextbookPageIDs []string            `json:"textbook_page_ids"`           // 关联课本图片ID
	CourseOutlineID string              `json:"course_outline_id,omitempty"` // 精确课程大纲ID
}

// LessonPlanChatRequest 教案对话请求
type LessonPlanChatRequest struct {
	PlanID             string   `json:"plan_id"`
	Message            string   `json:"message"`
	SelectedOptions    []string `json:"selected_options,omitempty"`
	SelectedComponents []string `json:"selected_components,omitempty"`
	CurrentSection     string   `json:"current_section,omitempty"`
	AssistantID        string   `json:"assistant_id,omitempty"`
	FullGenerate       bool     `json:"full_generate,omitempty"`
	ClientTurnID       string   `json:"client_turn_id,omitempty"`
	RefMaterial        string   `json:"ref_material,omitempty"`
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

// ==================== 参考资料附件压缩(PDF/Word) ====================

type CompressRefMaterialRequest struct {
	Content  string `json:"content"`
	FileName string `json:"file_name"`
	Subject  string `json:"subject"`
	Grade    string `json:"grade"`
}

type CompressRefMaterialResponse struct {
	Compressed    string `json:"compressed"`
	OriginalLen   int    `json:"original_len"`
	CompressedLen int    `json:"compressed_len"`
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

	LPSSESuggestedActions LPSSEEventType = "suggested_actions"
	LPSSERetryNotice      LPSSEEventType = "retry_notice"
	LPSSEContextCapsule   LPSSEEventType = "context_capsule"
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
type LPSSEEvent struct {
	EventType        LPSSEEventType                     `json:"type"`
	PlanID           string                             `json:"plan_id"`
	MessageID        string                             `json:"message_id,omitempty"`
	Chunk            string                             `json:"chunk,omitempty"`
	Message          *ConversationMessage               `json:"message,omitempty"`
	Content          string                             `json:"content,omitempty"`
	Review           *AIReviewResult                    `json:"review,omitempty"`
	ExtractionHint   *ExtractionHint                    `json:"extraction_hint,omitempty"`
	StageData        *StageEventData                    `json:"stage_data,omitempty"`
	Error            string                             `json:"error,omitempty"`
	SuggestedActions []SuggestedAction                  `json:"suggested_actions,omitempty"`
	ClientTurnID     string                             `json:"client_turn_id,omitempty"`
	AssistantLabel   string                             `json:"assistant_label,omitempty"`
	ContextCapsule   *LessonPlanContextCapsuleEventData `json:"context_capsule,omitempty"`
}

// LessonPlanContextCapsuleEventData 是核心共识胶囊的非终态SSE载荷。
//
// 该事件只在旁路更新产生新版本后广播，不影响主回复完成状态，
// 也不会使SSE处理器关闭连接。
type LessonPlanContextCapsuleEventData struct {
	Version int                                  `json:"version"`
	Status  string                               `json:"status"`
	Display *LessonPlanContextCapsuleDisplayView `json:"display,omitempty"`
}

// SuggestedAction 单个建议芯片
type SuggestedAction struct {
	ID         string                 `json:"id"`
	Emoji      string                 `json:"emoji,omitempty"`
	Label      string                 `json:"label"`
	ActionType string                 `json:"action_type"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
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

// ==================== v108新增：已有教案导入 ====================

type ImportExistingPlanRequest struct {
	Subject             string   `json:"subject"`
	Grade               string   `json:"grade"`
	Topic               string   `json:"topic"`
	DurationMinutes     int      `json:"duration_minutes"`
	ContentMarkdown     string   `json:"content_markdown"`
	RecipeID            string   `json:"recipe_id"`
	GroupID             string   `json:"group_id"`
	TextbookPageIDs     []string `json:"textbook_page_ids"`
	SourceType          string   `json:"source_type"`
	WordImportSessionID string   `json:"word_import_session_id,omitempty"`
}

type ImportExistingPlanResponse struct {
	Plan           *LessonPlan             `json:"plan"`
	OpeningMessage *ConversationMessage    `json:"opening_message"`
	SkippedStages  []string                `json:"skipped_stages"`
	WordDocument   *LessonPlanWordDocument `json:"word_document,omitempty"`
}

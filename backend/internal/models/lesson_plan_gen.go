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
	Subject         string   `json:"subject"`           // 学科（必填）
	Grade           string   `json:"grade"`             // 年级（必填）
	Topic           string   `json:"topic"`             // 课题（必填）
	DurationMinutes int      `json:"duration_minutes"`  // 课时时长（可选，默认45）
	TemplateID      string   `json:"template_id"`       // 提示词模板ID（可选）
	GroupID         string   `json:"group_id"`          // 教研组ID（可选）
	RecipeID        string   `json:"recipe_id"`         // 备课配方ID（可选，Phase 7A新增）
	TextbookPageIDs []string `json:"textbook_page_ids"` // 迭代7B：备课工坊勾选的课本图片ID列表，注入写教案上下文
}

// LessonPlanChatRequest 教案对话请求
// v110(TE-DNA 3.0 P0 STEP 3) 新增 AssistantID 字段:
//   - 空字符串 → 走原逻辑(使用阶段默认 system prompt)
//   - 非空字符串 → 根据 ID 加载 AI 助手,用其 full_prompt 替换第4层阶段角色
//   - 助手加载失败时静默降级到原逻辑(不中断对话流程),并记录告警日志
//
// v168 新增 FullGenerate 字段(功能B·一键生成完整教案):
//   - false(默认) → 走原逐轮对话逻辑,行为完全不变
//   - true → 仅在 write 阶段生效,后端注入"全委托一次性出稿"指令,
//     让 AI 一次性产出完整教案正文,产出走现有 handleWriteStageOutput 链路落库
type LessonPlanChatRequest struct {
	PlanID             string   `json:"plan_id"`
	Message            string   `json:"message"`
	SelectedOptions    []string `json:"selected_options,omitempty"`
	SelectedComponents []string `json:"selected_components,omitempty"`
	CurrentSection     string   `json:"current_section,omitempty"`
	AssistantID        string   `json:"assistant_id,omitempty"`  // v110 新增:可选的 AI 助手 ID
	FullGenerate       bool     `json:"full_generate,omitempty"` // v168 新增:全委托一键生成(仅 write 阶段生效)
	// 子轮一·B(B2 轮次序号)：前端每发起一轮 chat 自增的客户端轮次 id。
	// 后端原样回带到该轮所有 SSE 事件的 ClientTurnID 字段，前端据此丢弃"过期轮次"的迟到回复。
	// 空字符串(前端未传)时后端行为 100% 不变——回带空串，前端不过滤。
	ClientTurnID       string   `json:"client_turn_id,omitempty"` // 子轮一·B 新增:客户端轮次序号(B2)
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
	// 迭代3.5 Phase B：建议芯片(suggested_actions)动态下发事件
	LPSSESuggestedActions LPSSEEventType = "suggested_actions"
	// 子轮一·B(重试可见性)：空流自动重试时广播，提示文案放在 Content 字段。
	// 前端收到后把"思考中"替换为"刚才没接上、正在重试…"，让重试期间的等待有解释。
	LPSSERetryNotice LPSSEEventType = "retry_notice"
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
	EventType        LPSSEEventType       `json:"type"`
	PlanID           string               `json:"plan_id"`
	MessageID        string               `json:"message_id,omitempty"`
	Chunk            string               `json:"chunk,omitempty"`
	Message          *ConversationMessage `json:"message,omitempty"`
	Content          string               `json:"content,omitempty"`
	Review           *AIReviewResult      `json:"review,omitempty"`
	ExtractionHint   *ExtractionHint      `json:"extraction_hint,omitempty"`
	StageData        *StageEventData      `json:"stage_data,omitempty"` // Phase 7B-8：阶段事件数据
	Error            string               `json:"error,omitempty"`
	SuggestedActions []SuggestedAction    `json:"suggested_actions,omitempty"` // 迭代3.5 Phase B：动态建议芯片
	// 子轮一·B(B2 轮次序号)：标识本事件归属哪一轮 chat。
	// 仅 processChatStageAsync(及其派生的软兜底/硬错误/教练建议)显式赋值为本轮 turnID；
	// 系统旁路推送(开场白/评审/手动按钮)留空——前端对空串不过滤、照常处理。
	ClientTurnID     string               `json:"client_turn_id,omitempty"` // 子轮一·B 新增:轮次序号(B2)
	// 助手轻量选择入口·可见性补丁：本轮实际注入(匹配)的助手显示名。
	// 仅 message_done 事件在自动匹配/偏好命中/手动选择有助手时填写;纯骨架(无助手)留空。
	// 前端据此把顶栏"自动匹配"替换为真实助手名,让"已自动匹配"对老师可见(不写偏好不冻结)。
	AssistantLabel   string               `json:"assistant_label,omitempty"`
}

// SuggestedAction 单个建议芯片（迭代3.5 Phase B：对话式备课唤起式交互原语）
// 定义在 models 包，供 LPSSEEvent.SuggestedActions 字段与 services 包共同引用。
// 字段对齐设计文档 2.3 与前端 conversationScript.ts 的 ChipDef 下发契约。
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

// ImportExistingPlanRequest 导入已有教案请求
// 支持三种来源：粘贴文本、Word解析后文本、PDF解析后文本（均在前端完成解析）
type ImportExistingPlanRequest struct {
	Subject         string   `json:"subject"`           // 学科（必填）
	Grade           string   `json:"grade"`             // 年级（必填）
	Topic           string   `json:"topic"`             // 课题（必填）
	DurationMinutes int      `json:"duration_minutes"`  // 课时（默认45）
	ContentMarkdown string   `json:"content_markdown"`  // 教案正文（必填，前端已解析为纯文本）
	RecipeID        string   `json:"recipe_id"`         // 配方ID（可选）
	GroupID         string   `json:"group_id"`          // 教研组ID（可选）
	TextbookPageIDs []string `json:"textbook_page_ids"` // 关联课本图片（可选）
	SourceType      string   `json:"source_type"`       // 来源：paste/docx/pdf
}

// ImportExistingPlanResponse 导入已有教案响应
type ImportExistingPlanResponse struct {
	Plan           *LessonPlan          `json:"plan"`
	OpeningMessage *ConversationMessage `json:"opening_message"`
	SkippedStages  []string             `json:"skipped_stages"` // 跳过的阶段列表
}

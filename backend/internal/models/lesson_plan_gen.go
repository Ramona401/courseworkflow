package models

// ==================== 教案生成相关模型 ====================
// Phase3：教案生成核心流程
// Phase5：新增萃取提示事件类型（extraction_hint）
//
// 覆盖：会话消息、生成请求/响应、AI评审请求/响应、SSE推送事件

import "time"

// ==================== 会话消息模型 ====================

// ConversationRole 会话角色
type ConversationRole string

const (
	ConvRoleUser      ConversationRole = "user"      // 教师发言
	ConvRoleAssistant ConversationRole = "assistant"  // AI回复
	ConvRoleSystem    ConversationRole = "system"     // 系统消息
)

// ConvMsgType 消息类型
type ConvMsgType string

const (
	ConvMsgTypeText       ConvMsgType = "text"        // 普通文本
	ConvMsgTypeOptions    ConvMsgType = "options"     // 单选/多选选项
	ConvMsgTypeComponents ConvMsgType = "components"  // 推荐组件卡片
	ConvMsgTypeGenerate   ConvMsgType = "generate"    // 触发生成
	ConvMsgTypeContent    ConvMsgType = "content"     // 生成的教案内容片段
	ConvMsgTypeReview     ConvMsgType = "review"      // AI评审结果
	ConvMsgTypeAction     ConvMsgType = "action"      // 操作按钮
)

// ConversationMessage 单条会话消息
// 存储在 lesson_plans.conversation_log（JSONB数组）
type ConversationMessage struct {
	ID         string                 `json:"id"`                   // 消息唯一ID
	Role       ConversationRole       `json:"role"`                 // 发言角色
	Type       ConvMsgType            `json:"type"`                 // 消息类型
	Content    string                 `json:"content"`              // 文本内容
	Options    []ConvOption           `json:"options,omitempty"`    // 选项列表（type=options时）
	Components []ConvComponent        `json:"components,omitempty"` // 推荐组件（type=components时）
	Actions    []ConvAction           `json:"actions,omitempty"`    // 操作按钮（type=action时）
	Metadata   map[string]interface{} `json:"metadata,omitempty"`   // 附加元数据
	CreatedAt  time.Time              `json:"created_at"`
}

// ConvOption 选项（单选/多选）
type ConvOption struct {
	Key      string `json:"key"`      // 选项键
	Label    string `json:"label"`    // 显示文本
	Emoji    string `json:"emoji"`    // 图标
	Selected bool   `json:"selected"` // 是否已选
}

// ConvComponent 推荐的组件卡片
type ConvComponent struct {
	ID             string  `json:"id"`              // 组件库ID
	LibraryType    string  `json:"library_type"`    // 组件类型
	DisplayLabel   string  `json:"display_label"`   // 显示标签（第一层）
	DesignLogic    string  `json:"design_logic"`    // 设计逻辑（第二层）
	ExampleSnippet string  `json:"example_snippet"` // 参考案例（第三层）
	QualityScore   float64 `json:"quality_score"`   // 质量分
	UsageCount     int     `json:"usage_count"`     // 使用次数
	Selected       bool    `json:"selected"`        // 是否已选
}

// ConvAction 操作按钮
type ConvAction struct {
	Key   string `json:"key"`   // 操作键
	Label string `json:"label"` // 按钮文字
	Style string `json:"style"` // primary/secondary/danger
}

// ==================== 生成会话请求/响应 ====================

// StartConversationRequest 开始备课会话请求（第一屏：选年级+学科+课题）
type StartConversationRequest struct {
	Subject         string `json:"subject"`          // 学科（必填）
	Grade           string `json:"grade"`            // 年级（必填）
	Topic           string `json:"topic"`            // 课题（必填）
	DurationMinutes int    `json:"duration_minutes"` // 课时时长（可选，默认45）
	TemplateID      string `json:"template_id"`      // 提示词模板ID（可选）
	GroupID         string `json:"group_id"`         // 教研组ID（可选）
}

// LessonPlanChatRequest 会话消息请求（发送用户输入，获得AI回复）
type LessonPlanChatRequest struct {
	PlanID             string   `json:"plan_id"`                       // 教案ID（必填）
	Message            string   `json:"message"`                       // 用户文本输入
	SelectedOptions    []string `json:"selected_options,omitempty"`    // 已选选项的key列表
	SelectedComponents []string `json:"selected_components,omitempty"` // 已选组件的ID列表
	CurrentSection     string   `json:"current_section,omitempty"`     // 当前生成到的环节
}

// GenerateSectionRequest 生成某个教案环节的请求
type GenerateSectionRequest struct {
	PlanID               string   `json:"plan_id"`               // 教案ID
	Section              string   `json:"section"`               // 要生成的环节：intro/core/summary
	UserRequirement      string   `json:"user_requirement"`      // 教师的具体要求
	SelectedComponentIDs []string `json:"selected_component_ids"` // 已选组件
	Stream               bool     `json:"stream"`                // 是否流式输出
}

// ApplyAISuggestionsRequest 应用AI评审建议
type ApplyAISuggestionsRequest struct {
	PlanID      string   `json:"plan_id"`    // 教案ID
	Suggestions []string `json:"suggestions"` // 要应用的建议ID列表（空=全部）
}

// ==================== AI评审模型 ====================

// AIReviewDimension 单个评审维度
type AIReviewDimension struct {
	Code    string  `json:"code"`    // 维度代码：T1/T2/T3/T4/T5/S1/S2/S3/S4/S5
	Name    string  `json:"name"`    // 维度名称（对教师友好）
	Score   float64 `json:"score"`   // 得分（0-10）
	Comment string  `json:"comment"` // 具体说明（用对话口吻）
	Good    bool    `json:"good"`    // 是否属于"做得好的"
}

// AIReviewResult AI评审结果（存入 lesson_plans.ai_review_result）
type AIReviewResult struct {
	TotalScore   float64               `json:"total_score"`  // 总分（0-10）
	GoodPoints   []string              `json:"good_points"`  // 做得好的（✅列表）
	Improvements []AIReviewImprovement `json:"improvements"` // 可以更好（💡列表）
	Dimensions   []AIReviewDimension   `json:"dimensions"`   // 各维度详情
	Summary      string                `json:"summary"`      // 整体总结（对话口吻）
	ReviewedAt   time.Time             `json:"reviewed_at"`
}

// AIReviewImprovement 单条改进建议
type AIReviewImprovement struct {
	ID         string `json:"id"`         // 建议ID（用于Accept）
	Issue      string `json:"issue"`      // 问题描述
	Suggestion string `json:"suggestion"` // 具体改进方案（对话口吻）
	Section    string `json:"section"`    // 涉及环节（可选）
	Applied    bool   `json:"applied"`    // 是否已应用
}

// TriggerAIReviewRequest 触发AI评审请求
type TriggerAIReviewRequest struct {
	PlanID string `json:"plan_id"` // 教案ID（必填）
}

// ==================== SSE推送事件（教案生成专用） ====================

// LPSSEEventType 教案生成SSE事件类型
type LPSSEEventType string

const (
	LPSSEConnected       LPSSEEventType = "connected"       // 连接建立
	LPSSEThinking        LPSSEEventType = "thinking"        // AI思考中
	LPSSEChunk           LPSSEEventType = "chunk"           // 流式文本块
	LPSSEMessageDone     LPSSEEventType = "message_done"    // 单条AI消息完成
	LPSSEContentUpdate   LPSSEEventType = "content_update"  // 教案内容更新
	LPSSEReviewDone      LPSSEEventType = "review_done"     // AI评审完成
	LPSSEExtractionHint  LPSSEEventType = "extraction_hint" // Phase5：萃取提示（AI发现高价值片段）
	LPSSEError           LPSSEEventType = "error"           // 错误
	LPSSEDone            LPSSEEventType = "done"            // 流结束
)

// ExtractionHint 萃取提示数据（随extraction_hint事件推送）
// 老师看到的是"💡 这个点子很棒，要保存到您的备课灵感集吗？"
type ExtractionHint struct {
	HintID        string `json:"hint_id"`        // 临时ID，用于老师确认/拒绝
	DisplayText   string `json:"display_text"`   // 展示给老师的简短说明（大白话，不出现"组件"等术语）
	SourceContent string `json:"source_content"` // 被萃取的原始内容片段
	ExtractionType string `json:"extraction_type"` // 萃取类型（对应组件库类型，仅供后端使用）
	PlanID        string `json:"plan_id"`        // 所属教案ID
}

// LPSSEEvent 教案生成SSE推送事件
type LPSSEEvent struct {
	EventType      LPSSEEventType       `json:"type"`                    // 事件类型
	PlanID         string               `json:"plan_id"`                 // 教案ID
	MessageID      string               `json:"message_id,omitempty"`    // 消息ID
	Chunk          string               `json:"chunk,omitempty"`         // 流式文本块（type=chunk）
	Message        *ConversationMessage `json:"message,omitempty"`       // 完整消息（type=message_done）
	Content        string               `json:"content,omitempty"`       // 教案内容更新（type=content_update）
	Review         *AIReviewResult      `json:"review,omitempty"`        // 评审结果（type=review_done）
	ExtractionHint *ExtractionHint      `json:"extraction_hint,omitempty"` // 萃取提示（type=extraction_hint）
	Error          string               `json:"error,omitempty"`         // 错误信息（type=error）
}

// ==================== 生成步骤枚举 ====================

// LPGenStep 教案生成步骤（对话流程中的阶段）
type LPGenStep string

const (
	LPGenStepInit      LPGenStep = "init"      // 初始化（静默注入背景知识）
	LPGenStepCollect   LPGenStep = "collect"   // 采集学情（AI提问）
	LPGenStepRecommend LPGenStep = "recommend" // 推荐教学方案
	LPGenStepGenerate  LPGenStep = "generate"  // 逐环节生成
	LPGenStepReview    LPGenStep = "review"    // AI质量评审
	LPGenStepDone      LPGenStep = "done"      // 完成
)

// ==================== Phase5：萃取确认请求 ====================

// ConfirmExtractionRequest 老师确认/拒绝萃取请求
type ConfirmExtractionRequest struct {
	Decision string `json:"decision"` // confirmed / rejected
}

// ExtractionListItem 待审萃取列表项（教研组长视角）
type ExtractionListItem struct {
	ID             string  `json:"id"`              // 萃取记录ID
	SourceType     string  `json:"source_type"`     // 来源类型：conversation/lesson_plan
	SourceContent  string  `json:"source_content"`  // 原始内容片段
	ExtractionType string  `json:"extraction_type"` // 萃取类型（组件库类型）
	LibraryName    string  `json:"library_name"`    // 组件库中文名
	Status         string  `json:"status"`          // pending/confirmed/rejected
	PlanTitle      string  `json:"plan_title"`      // 来源教案标题（可选）
	CreatedByName  string  `json:"created_by_name"` // 创建者名称
	CreatedAt      string  `json:"created_at"`      // 创建时间
}

// ExtractionListResponse 待审萃取列表响应
type ExtractionListResponse struct {
	Extractions []*ExtractionListItem `json:"extractions"`
	Total       int                   `json:"total"`
}

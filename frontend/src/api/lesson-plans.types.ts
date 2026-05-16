/**
 * lesson-plans.types.ts — 教案系统类型定义
 *
 * 从 lesson-plans.ts 拆分,包含所有 interface/type/enum 定义
 * API 函数在 lesson-plans.ts 中
 */

/**
 * 教案系统 API 封装
 *
 * 覆盖教案系统全部后端接口:
 * - 组织管理(organizations)
 * - 教研组管理(teaching-groups)
 * - 组件库管理(components)
 * - 教案管理(plans)
 * - 提示词模板管理(templates)
 * - 教案生成(Phase3:对话/评审/建议应用)
 * - 萃取队列管理(Phase5:萃取列表/确认拒绝)
 * - 阶段化备课工坊(Phase 7B-9:阶段查询/前进/跳过/回退/产出物)
 *
 * v88新增:SSE自动重连机制(指数退避+连接状态回调+重连补齐)
 * v101修复:submitLessonPlanForReview 增加 groupId 参数,修复提交评审参数错误
 * v112(P0 STEP 8)新增:LessonPlanChatRequest 新增 assistant_id 可选字段,
 *   用于在备课工坊每轮对话时透传 AI 助手 ID 到后端,后端按 ID 加载 full_prompt 注入系统提示词。
 *   向后兼容:不传或传 null 时,后端走兜底默认 prompt。
 */

/* ==================== 类型定义:组织 ==================== */

export type OrgType = 'region' | 'school'

export interface Organization {
  id: string
  name: string
  type: OrgType
  parent_id: string | null
  admin_user_id: string | null
  settings: Record<string, unknown>
  status: string
  created_at: string
  updated_at: string
  group_count?: number
  member_count?: number
  admin_name?: string
  parent_name?: string
}

export interface OrganizationListResponse {
  items: Organization[]
  total: number
}

/* ==================== 类型定义:教研组 ==================== */

export interface TeachingGroup {
  id: string
  name: string
  school_id: string
  subject: string
  grade_range: string
  lead_user_id: string | null
  description: string
  settings: Record<string, unknown>
  status: string
  created_at: string
  updated_at: string
  school_name?: string
  lead_user_name?: string
  member_count?: number
}

export interface TeachingGroupDetail extends TeachingGroup {
  members: GroupMember[]
}

export interface GroupMember {
  id: string
  user_id: string
  username: string
  display_name: string
  role: 'member' | 'backbone'
  joined_at: string
}

export interface TeachingGroupListResponse {
  items: TeachingGroup[]
  total: number
}

/* ==================== 类型定义:组件库 ==================== */

export type LibraryType =
  | 'curriculum_standard'
  | 'knowledge_graph'
  | 'student_profile'
  | 'pedagogy'
  | 'assessment_strategy'
  | 'activity_design'
  | 'questioning_strategy'
  | 'cross_subject'
  | 'teaching_tool'
  | 'scenario_material'
  | 'quality_rubric'
  | 'design_defect'
  | 'review_rubric'

export type InjectionMode = 'silent' | 'recommend' | 'on_demand'

export interface LessonPlanComponent {
  id: string
  library_type: LibraryType
  subject: string
  grade_range: string | null
  tags: string[]
  injection_mode: InjectionMode
  display_label: string
  design_logic: string | null
  example_snippet: string | null
  full_guide: string | null
  content: Record<string, unknown>
  source: string
  quality_score: number
  usage_count: number
  select_count: number
  like_count: number
  dislike_count: number
  scope: string
  review_status: string
  status: string
  created_at: string
  updated_at: string
}

export interface ComponentListItem {
  id: string
  library_type: LibraryType
  library_name: string
  subject: string
  grade_range: string
  injection_mode: InjectionMode
  display_label: string
  quality_score: number
  usage_count: number
  select_count: number
  source: string
  review_status: string
  scope: string
  status: string
  created_at: string
}

export interface ComponentListResponse {
  components: ComponentListItem[]
  total: number
}

/* ==================== 类型定义:提示词模板 ==================== */

export type TemplateLevel = 'region' | 'school' | 'group' | 'personal'

export interface PromptTemplate {
  id: string
  name: string
  description: string | null
  level: TemplateLevel
  owner_id: string
  parent_template_id: string | null
  system_prompt: string | null
  context_rules: Record<string, unknown>
  generation_rules: Record<string, unknown>
  review_rules: Record<string, unknown>
  output_format: Record<string, unknown>
  custom_instructions: string | null
  subject: string | null
  grade_range: string | null
  is_default: boolean
  version: number
  status: string
  created_by: string | null
  created_at: string
  updated_at: string
}

export interface PromptTemplateListResponse {
  items: PromptTemplate[]
  total: number
}

export interface ResolvedPromptTemplate {
  id: string
  name: string
  level: TemplateLevel
  system_prompt: string
  context_rules: Record<string, unknown>
  generation_rules: Record<string, unknown>
  review_rules: Record<string, unknown>
  output_format: Record<string, unknown>
  custom_instructions: string
  chain: Array<{ id: string; name: string; level: string }>
}

export interface CreatePromptTemplateRequest {
  name: string
  description?: string
  level: TemplateLevel
  owner_id: string
  parent_template_id?: string
  system_prompt?: string
  context_rules?: Record<string, unknown>
  generation_rules?: Record<string, unknown>
  review_rules?: Record<string, unknown>
  output_format?: Record<string, unknown>
  custom_instructions?: string
  subject?: string
  grade_range?: string
  is_default?: boolean
}

export interface UpdatePromptTemplateRequest {
  name?: string
  description?: string
  parent_template_id?: string | null
  system_prompt?: string | null
  context_rules?: Record<string, unknown> | null
  generation_rules?: Record<string, unknown> | null
  review_rules?: Record<string, unknown> | null
  output_format?: Record<string, unknown> | null
  custom_instructions?: string | null
  subject?: string | null
  grade_range?: string | null
  is_default?: boolean
}

/* ==================== 类型定义:教案 ==================== */

export type LessonPlanStatus =
  | 'draft'
  | 'published_personal'
  | 'submitted'
  | 'revision'
  | 'approved'
  | 'published_shared'
  | 'developing'
  | 'completed'

export interface LessonPlan {
  id: string
  title: string
  subject: string
  grade: string
  topic: string
  duration_minutes: number
  content_markdown: string | null
  content_structured: string | Record<string, unknown>
  status: LessonPlanStatus
  visibility: string
  author_id: string
  group_id: string | null
  school_id: string | null
  ai_review_score: number | null
  ai_review_result: string | null
  linked_pipeline_id: string | null
  recipe_id: string | null
  recipe_name: string | null
  version: number
  current_stage: string | null       // Phase 7B-9:当前阶段代码
  stage_config: string | null        // Phase 7B-9:阶段配置快照JSON
  created_at: string
  updated_at: string
  author_name?: string
}

export interface LessonPlanListResponse {
  lesson_plans: LessonPlan[]
  total: number
}

/* ==================== 类型定义:对话与生成(Phase3)==================== */

export type ConvRole = 'user' | 'assistant' | 'system'
export type ConvMsgType = 'text' | 'options' | 'components' | 'generate' | 'content' | 'review' | 'action'

export interface ConvOption {
  key: string
  label: string
  emoji: string
  selected: boolean
}

export interface ConvComponent {
  id: string
  library_type: string
  display_label: string
  design_logic: string
  example_snippet: string
  quality_score: number
  usage_count: number
  selected: boolean
}

export interface ConvAction {
  key: string
  label: string
  style: 'primary' | 'secondary' | 'danger'
}

export interface ConversationMessage {
  id: string
  role: ConvRole
  type: ConvMsgType
  content: string
  options?: ConvOption[]
  components?: ConvComponent[]
  actions?: ConvAction[]
  metadata?: Record<string, unknown>
  created_at: string
}

export interface StartConversationRequest {
  subject: string
  grade: string
  topic: string
  duration_minutes?: number
  template_id?: string
  group_id?: string
  recipe_id?: string
  textbook_page_ids?: string[]  // 迭代7B:关联课本图片ID列表
}

export interface StartConversationResponse {
  plan: LessonPlan
  opening_message: ConversationMessage
}

/**
 * 教案对话请求体
 *
 * v112(P0 STEP 8)新增 assistant_id 可选字段:
 *   - 传入后端接收到助手 ID,在 Chat service 中调 LoadActiveAssistantForUse
 *     加载助手的 full_prompt,注入到第 4 层(阶段角色)替换默认 AI 角色
 *   - 不传/null/空字符串时后端走兜底默认 prompt,行为与 v110 及之前完全一致
 *   - 前端由 AssistantSelector 组件产生该字段,跟随当前阶段 scene 自动匹配默认助手
 */
export interface LessonPlanChatRequest {
  plan_id: string
  message: string
  selected_options?: string[]
  selected_components?: string[]
  current_section?: string
  /** v112(P0 STEP 8):AI 助手 ID,空/null 表示不使用助手走兜底 */
  assistant_id?: string | null
}

export interface ApplyAISuggestionsRequest {
  plan_id: string
  suggestions?: string[]
}

export interface AIReviewDimension {
  code: string
  name: string
  score: number
  comment: string
  good: boolean
}

export interface AIReviewImprovement {
  id: string
  issue: string
  suggestion: string
  section?: string
  applied: boolean
}

export interface AIReviewResult {
  total_score: number
  good_points: string[]
  improvements: AIReviewImprovement[]
  dimensions: AIReviewDimension[]
  summary: string
  reviewed_at: string
}

/* ==================== 类型定义:阶段化备课工坊(Phase 7B-9 新增)==================== */

/** 阶段进度条目 */
export interface StageProgressItem {
  stage_code: string
  stage_name: string
  stage_order: number
  ai_role: string
  gate_mode: 'suggest' | 'force' | 'auto'
  skippable: boolean
  status: 'pending' | 'in_progress' | 'completed' | 'skipped'
  has_output: boolean
  completed_at: string | null
}

/** 阶段状态响应 */
export interface StageStatusResponse {
  current_stage: string
  total_stages: number
  stages: StageProgressItem[]
}

/** 阶段产出物响应 */
export interface StageOutputResponse {
  stage_code: string
  stage_name: string
  structured_output: string
  narrative_output: string
  status: string
  model_used: string
  tokens_used: number
}

/** 系统默认阶段项 */
export interface DefaultStageItem {
  stage_code: string
  stage_name: string
  stage_order: number
  ai_role: string
  gate_mode: string
  skippable: boolean
  component_types: string
}

/** 阶段SSE事件数据 */
export interface StageEventData {
  stage_code: string
  stage_name: string
  stage_order: number
  total_stages: number
  next_stage?: string
  can_skip?: boolean
}

/* ==================== SSE事件类型(Phase 7B-9:新增3个阶段事件)==================== */

export type LPSSEEventType =
  | 'connected'
  | 'thinking'
  | 'chunk'
  | 'message_done'
  | 'content_update'
  | 'review_done'
  | 'extraction_hint'
  | 'stage_started'     // Phase 7B-9:进入新阶段
  | 'stage_complete'    // Phase 7B-9:AI建议完成当前阶段
  | 'stage_output'      // Phase 7B-9:阶段产出物已生成
  | 'error'
  | 'done'

/** Phase5:萃取提示数据 */
export interface ExtractionHint {
  hint_id: string
  display_text: string
  source_content: string
  extraction_type: string
  plan_id: string
}

/** SSE事件完整结构(Phase 7B-9:新增stage_data字段) */
export interface LPSSEEvent {
  type: LPSSEEventType
  plan_id: string
  message_id?: string
  chunk?: string
  message?: ConversationMessage
  content?: string
  review?: AIReviewResult
  extraction_hint?: ExtractionHint
  stage_data?: StageEventData         // Phase 7B-9:阶段事件数据
  error?: string
}

/* ==================== 类型定义:萃取队列(Phase5新增)==================== */

export interface ExtractionListItem {
  id: string
  source_type: 'conversation' | 'lesson_plan' | 'manual'
  source_content: string
  extraction_type: string
  library_name: string
  status: 'pending' | 'confirmed' | 'rejected'
  plan_title?: string
  created_by_name?: string
  created_at: string
}

export interface ExtractionListResponse {
  extractions: ExtractionListItem[]
  total: number
}

/* ==================== v88新增:SSE连接状态类型 ==================== */

/** SSE连接状态枚举 */
export type SSEConnectionState = 'connected' | 'reconnecting' | 'disconnected'

/** SSE重连配置常量 */
const SSE_RECONNECT_MAX_RETRIES = 5           // 最大重连次数
const SSE_RECONNECT_BASE_DELAY_MS = 1000      // 基础重连延迟(1秒)
const SSE_RECONNECT_MAX_DELAY_MS = 30000      // 最大重连延迟(30秒)

/* ==================== v88新增:可控SSE连接管理器 ==================== */

/**
 * SSEConnection — 可控的SSE连接包装器
 *
 * v88新增:封装EventSource,提供:
 *   - 自动重连(指数退避,最多5次)
 *   - 连接状态变化回调(connected/reconnecting/disconnected)
 *   - 重连成功后自动拉取最新对话补齐丢失消息
 *   - 手动关闭(close方法)
 */
export interface SSEConnection {
  /** 手动关闭连接(同时停止重连计时器) */
  close: () => void
}

/* ==================== API函数:组织管理 ==================== */


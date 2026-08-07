/**
 * 课件教学智能体 —— 浏览器安全类型与运行SSE守卫。
 *
 * 本文件只描述后端明确允许返回浏览器的安全协议：
 *   - 教师端插槽、教学方案、上下文预览和部署元数据；
 *   - 公开端短时运行会话、正式可见消息与聊天结果；
 *   - 公开运行SSE事件载荷。
 *
 * 明确不包含：
 *   - 助手完整提示词；
 *   - 页面完整HTML或教案全文；
 *   - 教师、学校、积分账户及模型密钥；
 *   - JTI、匿名客户端和IP哈希；
 *   - 模型隐藏推理。
 */

// ==================== 基础协议 ====================

/**
 * 上下文配置协议当前仍为v1。
 */
export type CoursewareAssistantProtocolVersion =
  'v1'

/**
 * 教学方案v1是历史苏格拉底式协议，v2增加明确教学方式。
 */
export type CoursewareAssistantGuidancePlanVersion =
  | 'v1'
  | 'v2'

export type CoursewareAssistantTeachingMode =
  | 'guided_reasoning'
  | 'explain_back'
  | 'predict_observe_explain'
  | 'worked_example'
  | 'coached_practice'
  | 'retrieval_check'
  | 'compare_contrast'
  | 'evidence_argument'

export type CoursewareAssistantSlotStatus =
  | 'active'
  | 'disabled'

export type CoursewareAssistantDisplayMode =
  'floating'

export type CoursewareAssistantDisplayPosition =
  'bottom_right'

export type CoursewareAssistantDeploymentStatus =
  | 'active'
  | 'paused'
  | 'revoked'

export type CoursewareAssistantDeploymentAccessMode =
  'origin_allowlist'

export type AssistantRuntimeSessionKind =
  | 'external'
  | 'teacher_preview'

export type AssistantRuntimeSessionStatus =
  | 'active'
  | 'completed'
  | 'expired'
  | 'revoked'

export type AssistantRuntimeMessageRole =
  | 'student'
  | 'assistant'

export type AssistantRuntimeISODateTime =
  | string
  | null

// ==================== 通用结构化教学方案 ====================

export interface CoursewareAssistantQuestionStep {
  id: string
  prompt: string
  teaching_intent: string
  expected_signals: string[]
  hint_ladder: string[]
  misconception_branch_ids: string[]
  next_step_id?: string
  completion_signal?: string
}

export interface CoursewareAssistantMisconceptionBranch {
  id: string
  match_signals: string[]
  response_strategy: string
  follow_up_question: string
  return_to_step_id: string
}

export interface CoursewareAssistantAnswerLeakPolicy {
  direct_answer_allowed: boolean
  require_student_try: boolean
  maximum_hint_level: number
  prohibited_behaviors: string[]
  safe_closure_guidance?: string
}

export interface CoursewareAssistantGuidancePlan {
  version: CoursewareAssistantGuidancePlanVersion
  teaching_mode: CoursewareAssistantTeachingMode
  guiding_principles: string[]
  question_chain: CoursewareAssistantQuestionStep[]
  misconception_branches:
    CoursewareAssistantMisconceptionBranch[]
  forbidden_behaviors: string[]
  completion_criteria: string[]
  answer_leak_policy:
    CoursewareAssistantAnswerLeakPolicy
}

// ==================== 上下文配置与预览 ====================

export interface CoursewareAssistantContextConfig {
  version: CoursewareAssistantProtocolVersion
  include_visible_text: boolean
  include_page_plan: boolean
  include_interaction_evidence: boolean
  include_lesson_plan_excerpt: boolean
  include_previous_page_summary: boolean
  include_next_page_summary: boolean
  max_lesson_plan_excerpt_chars: number
}

export interface CoursewareAssistantPagePreview {
  page_id: string
  page_number: number
  title: string
  purpose: string
  content_summary: string
  visible_text?: string
}

export interface CoursewareAssistantLessonPlanPreview {
  lesson_plan_id: string | null
  title: string
  excerpt_preview: string
  character_count: number
}

export interface CoursewareAssistantInteractionPreview {
  declared_type: string
  contract_ok: boolean
  event_count: number
  dom_target_count: number
  risk_flags: string[]
  manual_review_required: boolean
}

export interface CoursewareAssistantContextPreview {
  current_page: CoursewareAssistantPagePreview
  previous_page?: CoursewareAssistantPagePreview
  next_page?: CoursewareAssistantPagePreview
  lesson_plan?: CoursewareAssistantLessonPlanPreview
  interaction: CoursewareAssistantInteractionPreview
  context_config: CoursewareAssistantContextConfig
}

// ==================== 教师端插槽 ====================

export interface CoursewareAssistantSlotView {
  id: string
  courseware_id: string
  page_id: string
  assistant_id: string | null
  assistant_name: string
  assistant_active: boolean
  display_mode: CoursewareAssistantDisplayMode
  display_position: CoursewareAssistantDisplayPosition
  title: string
  welcome_message: string
  teaching_role: string
  learning_objective: string
  guidance_plan: CoursewareAssistantGuidancePlan
  context_config: CoursewareAssistantContextConfig
  status: CoursewareAssistantSlotStatus
  created_at: AssistantRuntimeISODateTime
  updated_at: AssistantRuntimeISODateTime
}

export interface CoursewareAssistantSlotListResponse {
  slots: CoursewareAssistantSlotView[]
  total: number
}

export interface CreateCoursewareAssistantSlotRequest {
  assistant_id: string | null
  title: string
  welcome_message: string
  teaching_role: string
  learning_objective: string
  guidance_plan: CoursewareAssistantGuidancePlan
  context_config: CoursewareAssistantContextConfig
}

export interface UpdateCoursewareAssistantSlotRequest
  extends CreateCoursewareAssistantSlotRequest {
  status: CoursewareAssistantSlotStatus
}

export interface GenerateCoursewareAssistantPlanRequest {
  assistant_id: string | null
  teaching_mode: CoursewareAssistantTeachingMode
  teacher_instruction: string
}

export interface CoursewareAssistantPlanResult {
  title: string
  welcome_message: string
  teaching_role: string
  learning_objective: string
  guidance_plan: CoursewareAssistantGuidancePlan
  context_config: CoursewareAssistantContextConfig
}

// ==================== 教师端部署管理 ====================

export interface CoursewareAssistantDeploymentView {
  id: string
  public_id: string
  slot_id: string | null
  courseware_id: string
  page_id: string
  education_domain: string
  current_version: number
  access_mode:
    CoursewareAssistantDeploymentAccessMode
  status: CoursewareAssistantDeploymentStatus
  daily_call_limit: number
  per_session_turn_limit: number
  allowed_origins: string[]
  valid_from: AssistantRuntimeISODateTime
  valid_until: AssistantRuntimeISODateTime
  created_at: AssistantRuntimeISODateTime
  updated_at: AssistantRuntimeISODateTime
}

export interface CoursewareAssistantDeploymentVersionView {
  version: number
  assistant_id: string | null
  assistant_prompt_hash: string
  context_snapshot_hash: string
  page_html_hash: string
  created_at: AssistantRuntimeISODateTime
}

export interface CoursewareAssistantDeploymentListResponse {
  deployments: CoursewareAssistantDeploymentView[]
  total: number
}

export interface PublishCoursewareAssistantDeploymentRequest {
  daily_call_limit: number
  per_session_turn_limit: number
  allowed_origins: string[]
  valid_until: string | null
}

/**
 * 当前更新策略请求与发布策略字段完全相同。
 */
export type UpdateCoursewareAssistantDeploymentPolicyRequest =
  PublishCoursewareAssistantDeploymentRequest

// ==================== 公开运行安全模型 ====================

export interface AssistantRuntimePublicDescriptor {
  public_id: string
  title: string
  welcome_message: string
  display_mode: CoursewareAssistantDisplayMode
  display_position:
    CoursewareAssistantDisplayPosition
  maximum_session_turns: number
}

export interface AssistantRuntimeMessage {
  role: AssistantRuntimeMessageRole
  content: string
  created_at: AssistantRuntimeISODateTime
}

/**
 * 公开iframe创建短时会话的请求。
 *
 * parent_origin来自document.referrer解析出的父页面精确Origin，
 * 后端仍会将它作为不可信字段重新规范化并与部署白名单比较。
 */
export interface AssistantRuntimeStartRequest {
  anonymous_client_id: string
  parent_origin: string
}

export interface AssistantRuntimeStartResponse {
  session_id: string
  runtime_token: string
  status: AssistantRuntimeSessionStatus
  max_turns: number
  expires_at: AssistantRuntimeISODateTime
  welcome_message: string
}

export interface AssistantRuntimeSessionView {
  id: string
  deployment_version: number
  session_kind: AssistantRuntimeSessionKind
  status: AssistantRuntimeSessionStatus
  turn_count: number
  max_turns: number
  remaining_turns: number
  messages: AssistantRuntimeMessage[]
  expires_at: AssistantRuntimeISODateTime
  last_active_at: AssistantRuntimeISODateTime
}

export interface AssistantRuntimeChatRequest {
  message: string
}

export interface AssistantRuntimeChatResponse {
  turn_id: string
  message: AssistantRuntimeMessage
  turn_count: number
  remaining_turns: number
  session_status: AssistantRuntimeSessionStatus
}

// ==================== 公开运行SSE ====================

export interface AssistantRuntimeConnectedEvent {
  phase: 'ready'
}

export interface AssistantRuntimeChunkEvent {
  chunk: string
}

export interface AssistantRuntimeErrorEvent {
  error: string
}

export interface AssistantRuntimeChatHandlers {
  onConnected?: (
    payload: AssistantRuntimeConnectedEvent,
  ) => void
  onChunk?: (chunk: string) => void
  onDone?: (
    result: AssistantRuntimeChatResponse,
  ) => void
  onError?: (message: string) => void
}

export interface AssistantRuntimeRequestOptions {
  /**
   * 默认使用当前站点。
   * 外部壳页面显式指定服务域名时传入。
   */
  baseURL?: string

  /**
   * 可选外部取消信号。
   * 组件卸载也可以直接调用连接句柄的close()。
   */
  signal?: AbortSignal
}

export interface AssistantRuntimeChatConnection {
  close: () => void

  /**
   * 流正常结束、服务端error或本地取消后完成。
   * 业务错误通过onError回调返回，避免无人消费的Promise rejection。
   */
  finished: Promise<void>
}

// ==================== 运行时结构守卫 ====================

type UnknownRecord =
  Record<string, unknown>

function isRecord(
  value: unknown,
): value is UnknownRecord {
  return (
    typeof value === 'object' &&
    value !== null
  )
}

function isString(
  value: unknown,
): value is string {
  return typeof value === 'string'
}

function isFiniteNumber(
  value: unknown,
): value is number {
  return (
    typeof value === 'number' &&
    Number.isFinite(value)
  )
}

function isNullableDate(
  value: unknown,
): value is AssistantRuntimeISODateTime {
  return (
    value === null ||
    typeof value === 'string'
  )
}

function isRuntimeSessionStatus(
  value: unknown,
): value is AssistantRuntimeSessionStatus {
  return (
    value === 'active' ||
    value === 'completed' ||
    value === 'expired' ||
    value === 'revoked'
  )
}

function isRuntimeSessionKind(
  value: unknown,
): value is AssistantRuntimeSessionKind {
  return (
    value === 'external' ||
    value === 'teacher_preview'
  )
}

function isRuntimeMessageRole(
  value: unknown,
): value is AssistantRuntimeMessageRole {
  return (
    value === 'student' ||
    value === 'assistant'
  )
}

export function isAssistantRuntimeMessage(
  value: unknown,
): value is AssistantRuntimeMessage {
  if (!isRecord(value)) {
    return false
  }

  return (
    isRuntimeMessageRole(value.role) &&
    isString(value.content) &&
    isNullableDate(value.created_at)
  )
}

export function isAssistantRuntimeStartResponse(
  value: unknown,
): value is AssistantRuntimeStartResponse {
  if (!isRecord(value)) {
    return false
  }

  return (
    isString(value.session_id) &&
    value.session_id.length > 0 &&
    isString(value.runtime_token) &&
    value.runtime_token.length > 0 &&
    isRuntimeSessionStatus(value.status) &&
    isFiniteNumber(value.max_turns) &&
    isNullableDate(value.expires_at) &&
    isString(value.welcome_message)
  )
}

export function isAssistantRuntimeSessionView(
  value: unknown,
): value is AssistantRuntimeSessionView {
  if (
    !isRecord(value) ||
    !Array.isArray(value.messages)
  ) {
    return false
  }

  return (
    isString(value.id) &&
    isFiniteNumber(value.deployment_version) &&
    isRuntimeSessionKind(value.session_kind) &&
    isRuntimeSessionStatus(value.status) &&
    isFiniteNumber(value.turn_count) &&
    isFiniteNumber(value.max_turns) &&
    isFiniteNumber(value.remaining_turns) &&
    value.messages.every(
      isAssistantRuntimeMessage,
    ) &&
    isNullableDate(value.expires_at) &&
    isNullableDate(value.last_active_at)
  )
}

export function isAssistantRuntimeChatResponse(
  value: unknown,
): value is AssistantRuntimeChatResponse {
  if (!isRecord(value)) {
    return false
  }

  return (
    isString(value.turn_id) &&
    value.turn_id.length > 0 &&
    isAssistantRuntimeMessage(value.message) &&
    isFiniteNumber(value.turn_count) &&
    isFiniteNumber(value.remaining_turns) &&
    isRuntimeSessionStatus(
      value.session_status,
    )
  )
}

export function isAssistantRuntimeConnectedEvent(
  value: unknown,
): value is AssistantRuntimeConnectedEvent {
  return (
    isRecord(value) &&
    value.phase === 'ready'
  )
}

export function isAssistantRuntimeChunkEvent(
  value: unknown,
): value is AssistantRuntimeChunkEvent {
  return (
    isRecord(value) &&
    isString(value.chunk)
  )
}

export function isAssistantRuntimeErrorEvent(
  value: unknown,
): value is AssistantRuntimeErrorEvent {
  return (
    isRecord(value) &&
    isString(value.error) &&
    value.error.trim().length > 0
  )
}

/**
 * 教案系统 API 封装
 *
 * 覆盖教案系统全部后端接口：
 * - 组织管理（organizations）
 * - 教研组管理（teaching-groups）
 * - 组件库管理（components）
 * - 教案管理（plans）
 * - 提示词模板管理（templates）
 * - 教案生成（Phase3：对话/评审/建议应用）
 * - 萃取队列管理（Phase5：萃取列表/确认拒绝）
 * - 阶段化备课工坊（Phase 7B-9：阶段查询/前进/跳过/回退/产出物）
 */
import apiClient from './client'

/* ==================== 类型定义：组织 ==================== */

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

/* ==================== 类型定义：教研组 ==================== */

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

/* ==================== 类型定义：组件库 ==================== */

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

/* ==================== 类型定义：提示词模板 ==================== */

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

/* ==================== 类型定义：教案 ==================== */

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
  content_structured: Record<string, unknown>
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
  current_stage: string | null       // Phase 7B-9：当前阶段代码
  stage_config: string | null        // Phase 7B-9：阶段配置快照JSON
  created_at: string
  updated_at: string
  author_name?: string
}

export interface LessonPlanListResponse {
  lesson_plans: LessonPlan[]
  total: number
}

/* ==================== 类型定义：对话与生成（Phase3）==================== */

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
  textbook_page_ids?: string[]  // 迭代7B：关联课本图片ID列表
}

export interface StartConversationResponse {
  plan: LessonPlan
  opening_message: ConversationMessage
}

export interface LessonPlanChatRequest {
  plan_id: string
  message: string
  selected_options?: string[]
  selected_components?: string[]
  current_section?: string
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

/* ==================== 类型定义：阶段化备课工坊（Phase 7B-9 新增）==================== */

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

/* ==================== SSE事件类型（Phase 7B-9：新增3个阶段事件）==================== */

export type LPSSEEventType =
  | 'connected'
  | 'thinking'
  | 'chunk'
  | 'message_done'
  | 'content_update'
  | 'review_done'
  | 'extraction_hint'
  | 'stage_started'     // Phase 7B-9：进入新阶段
  | 'stage_complete'    // Phase 7B-9：AI建议完成当前阶段
  | 'stage_output'      // Phase 7B-9：阶段产出物已生成
  | 'error'
  | 'done'

/** Phase5：萃取提示数据 */
export interface ExtractionHint {
  hint_id: string
  display_text: string
  source_content: string
  extraction_type: string
  plan_id: string
}

/** SSE事件完整结构（Phase 7B-9：新增stage_data字段） */
export interface LPSSEEvent {
  type: LPSSEEventType
  plan_id: string
  message_id?: string
  chunk?: string
  message?: ConversationMessage
  content?: string
  review?: AIReviewResult
  extraction_hint?: ExtractionHint
  stage_data?: StageEventData         // Phase 7B-9：阶段事件数据
  error?: string
}

/* ==================== 类型定义：萃取队列（Phase5新增）==================== */

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

/* ==================== API函数：组织管理 ==================== */

export async function getOrganizations(params?: { type?: OrgType; parent_id?: string }) {
  const query = new URLSearchParams()
  if (params?.type) query.set('type', params.type)
  if (params?.parent_id) query.set('parent_id', params.parent_id)
  const qs = query.toString()
  const resp = await apiClient.get(`/lesson-plans/organizations${qs ? '?' + qs : ''}`)
  return resp.data.data as OrganizationListResponse
}

export async function getOrganization(id: string) {
  const resp = await apiClient.get(`/lesson-plans/organizations/${id}`)
  return resp.data.data as Organization
}

export async function createOrganization(data: { name: string; type: OrgType; parent_id?: string; admin_user_id?: string }) {
  const resp = await apiClient.post('/lesson-plans/organizations', data)
  return resp.data.data as Organization
}

export async function updateOrganization(id: string, data: { name?: string; admin_user_id?: string; status?: string }) {
  const resp = await apiClient.put(`/lesson-plans/organizations/${id}`, data)
  return resp.data.data as Organization
}

export async function deleteOrganization(id: string) {
  const resp = await apiClient.delete(`/lesson-plans/organizations/${id}`)
  return resp.data.data as void
}

/* ==================== API函数：教研组管理 ==================== */

export async function getTeachingGroups(params?: { school_id?: string }) {
  const query = new URLSearchParams()
  if (params?.school_id) query.set('school_id', params.school_id)
  const qs = query.toString()
  const resp = await apiClient.get(`/lesson-plans/teaching-groups${qs ? '?' + qs : ''}`)
  return resp.data.data as TeachingGroupListResponse
}

export async function getTeachingGroupDetail(id: string) {
  const resp = await apiClient.get(`/lesson-plans/teaching-groups/${id}`)
  return resp.data.data as TeachingGroupDetail
}

export async function createTeachingGroup(data: {
  name: string; school_id: string; subject: string;
  grade_range?: string; lead_user_id?: string; description?: string
}) {
  const resp = await apiClient.post('/lesson-plans/teaching-groups', data)
  return resp.data.data as TeachingGroup
}

export async function updateTeachingGroup(id: string, data: Record<string, unknown>) {
  const resp = await apiClient.put(`/lesson-plans/teaching-groups/${id}`, data)
  return resp.data.data as TeachingGroup
}

export async function deleteTeachingGroup(id: string) {
  const resp = await apiClient.delete(`/lesson-plans/teaching-groups/${id}`)
  return resp.data.data as void
}

export async function addGroupMember(groupId: string, data: { user_id: string; role?: string }) {
  const resp = await apiClient.post(`/lesson-plans/teaching-groups/${groupId}/members`, data)
  return resp.data.data as void
}

export async function removeGroupMember(groupId: string, userId: string) {
  const resp = await apiClient.delete(`/lesson-plans/teaching-groups/${groupId}/members/${userId}`)
  return resp.data.data as void
}

export async function getMyGroups() {
  const resp = await apiClient.get('/lesson-plans/my-groups')
  return resp.data.data as TeachingGroup[]
}

/* ==================== API函数：组件库管理 ==================== */

export async function getComponents(params?: {
  library_type?: LibraryType; subject?: string;
  review_status?: string; scope?: string; limit?: number; offset?: number
}) {
  const query = new URLSearchParams()
  if (params?.library_type) query.set('library_type', params.library_type)
  if (params?.subject) query.set('subject', params.subject)
  if (params?.review_status) query.set('review_status', params.review_status)
  if (params?.scope) query.set('scope', params.scope)
  if (params?.limit) query.set('limit', String(params.limit))
  if (params?.offset) query.set('offset', String(params.offset))
  const qs = query.toString()
  const resp = await apiClient.get(`/lesson-plans/components${qs ? '?' + qs : ''}`)
  return resp.data.data as ComponentListResponse
}

export async function getComponent(id: string) {
  const resp = await apiClient.get(`/lesson-plans/components/${id}`)
  return resp.data.data as LessonPlanComponent
}

export async function createComponent(data: Record<string, unknown>) {
  const resp = await apiClient.post('/lesson-plans/components', data)
  return resp.data.data as LessonPlanComponent
}

export async function updateComponent(id: string, data: Record<string, unknown>) {
  const resp = await apiClient.put(`/lesson-plans/components/${id}`, data)
  return resp.data.data as LessonPlanComponent
}

export async function deleteComponent(id: string) {
  const resp = await apiClient.delete(`/lesson-plans/components/${id}`)
  return resp.data.data as void
}

export async function reviewComponent(id: string, data: { decision: string }) {
  const resp = await apiClient.post(`/lesson-plans/components/${id}/review`, data)
  return resp.data.data as void
}

export async function matchComponents(data: { subject: string; grade: string; topic: string; tags?: string[] }) {
  const resp = await apiClient.post('/lesson-plans/components/match', data)
  return resp.data.data as Record<string, unknown>
}

/* ==================== API函数：提示词模板管理 ==================== */

export async function getPromptTemplates(params?: {
  level?: TemplateLevel; subject?: string; limit?: number; offset?: number
}) {
  const query = new URLSearchParams()
  if (params?.level) query.set('level', params.level)
  if (params?.subject) query.set('subject', params.subject)
  if (params?.limit) query.set('limit', String(params.limit))
  if (params?.offset) query.set('offset', String(params.offset))
  const qs = query.toString()
  const resp = await apiClient.get(`/lesson-plans/templates${qs ? '?' + qs : ''}`)
  return resp.data.data as PromptTemplateListResponse
}

export async function getPromptTemplate(id: string) {
  const resp = await apiClient.get(`/lesson-plans/templates/${id}`)
  return resp.data.data as PromptTemplate
}

export async function createPromptTemplate(data: CreatePromptTemplateRequest) {
  const resp = await apiClient.post('/lesson-plans/templates', data)
  return resp.data.data as PromptTemplate
}

export async function updatePromptTemplate(id: string, data: UpdatePromptTemplateRequest) {
  const resp = await apiClient.put(`/lesson-plans/templates/${id}`, data)
  return resp.data.data as PromptTemplate
}

export async function resolvePromptTemplate(id: string) {
  const resp = await apiClient.get(`/lesson-plans/templates/${id}/resolved`)
  return resp.data.data as ResolvedPromptTemplate
}

/* ==================== API函数：教案管理 ==================== */

export async function getLessonPlans(params?: {
  author_id?: string; group_id?: string; status?: string;
  subject?: string; grade?: string; limit?: number; offset?: number
}) {
  const query = new URLSearchParams()
  if (params?.author_id) query.set('author_id', params.author_id)
  if (params?.group_id) query.set('group_id', params.group_id)
  if (params?.status) query.set('status', params.status)
  if (params?.subject) query.set('subject', params.subject)
  if (params?.grade) query.set('grade', params.grade)
  if (params?.limit) query.set('limit', String(params.limit))
  if (params?.offset) query.set('offset', String(params.offset))
  const qs = query.toString()
  const resp = await apiClient.get(`/lesson-plans/plans${qs ? '?' + qs : ''}`)
  return resp.data.data as LessonPlanListResponse
}

export async function getLessonPlan(id: string) {
  const resp = await apiClient.get(`/lesson-plans/plans/${id}`)
  return resp.data.data as LessonPlan
}

export async function createLessonPlan(data: {
  title: string; subject: string; grade: string; topic: string;
  duration_minutes?: number; template_id?: string; group_id?: string; school_id?: string
}) {
  const resp = await apiClient.post('/lesson-plans/plans', data)
  return resp.data.data as LessonPlan
}

export async function updateLessonPlan(id: string, data: Record<string, unknown>) {
  const resp = await apiClient.put(`/lesson-plans/plans/${id}`, data)
  return resp.data.data as LessonPlan
}

export async function deleteLessonPlan(id: string) {
  const resp = await apiClient.delete(`/lesson-plans/plans/${id}`)
  return resp.data.data as void
}

export async function publishLessonPlanPersonal(id: string) {
  const resp = await apiClient.post(`/lesson-plans/plans/${id}/publish-personal`)
  return resp.data.data as void
}

export async function submitLessonPlanForReview(id: string) {
  const resp = await apiClient.post(`/lesson-plans/plans/${id}/submit-review`)
  return resp.data.data as void
}

export async function reviewLessonPlan(id: string, data: {
  decision: string; score?: number; comments?: string; suggestions?: string[]
}) {
  const resp = await apiClient.post(`/lesson-plans/plans/${id}/review`, data)
  return resp.data.data as void
}

export async function publishLessonPlanShared(id: string) {
  const resp = await apiClient.post(`/lesson-plans/plans/${id}/publish-shared`)
  return resp.data.data as void
}

export async function startDevelopment(id: string) {
  const resp = await apiClient.post(`/lesson-plans/plans/${id}/start-development`)
  return resp.data.data as StartDevelopmentResult
}

export interface StartDevelopmentResult {
  pipeline_id: string
  message: string
}

export async function forkLessonPlan(id: string) {
  const resp = await apiClient.post(`/lesson-plans/plans/${id}/fork`)
  return resp.data.data as LessonPlan
}

/* ==================== API函数：教案生成（Phase3）==================== */

export async function startConversation(data: StartConversationRequest): Promise<StartConversationResponse> {
  const resp = await apiClient.post('/lesson-plans/plans/start-conversation', data)
  return resp.data.data as StartConversationResponse
}

export async function sendChatMessage(planId: string, data: Omit<LessonPlanChatRequest, 'plan_id'>) {
  const resp = await apiClient.post(`/lesson-plans/plans/${planId}/chat`, { ...data, plan_id: planId })
  return resp.data.data as { status: string; message: string }
}

export async function triggerAIReview(planId: string) {
  const resp = await apiClient.post(`/lesson-plans/plans/${planId}/trigger-review`)
  return resp.data.data as { status: string; message: string }
}

export async function applyAISuggestions(planId: string, suggestionIds?: string[]) {
  const resp = await apiClient.post(`/lesson-plans/plans/${planId}/apply-suggestions`, {
    plan_id: planId,
    suggestions: suggestionIds || [],
  })
  return resp.data.data as { status: string; message: string }
}

export async function getConversation(planId: string) {
  const resp = await apiClient.get(`/lesson-plans/plans/${planId}/conversation`)
  return resp.data.data as { messages: ConversationMessage[]; total: number }
}

/**
 * 创建教案SSE连接
 * Phase 7B-9新增：onStageStarted/onStageComplete/onStageOutput 三个阶段事件回调
 */
export function createLessonPlanSSE(
  planId: string,
  token: string,
  handlers: {
    onThinking?: () => void
    onChunk?: (chunk: string) => void
    onMessageDone?: (msg: ConversationMessage) => void
    onContentUpdate?: (content: string) => void
    onReviewDone?: (review: AIReviewResult) => void
    onExtractionHint?: (hint: ExtractionHint) => void
    onStageStarted?: (data: StageEventData) => void    // Phase 7B-9：进入新阶段
    onStageComplete?: (data: StageEventData) => void   // Phase 7B-9：AI建议完成当前阶段
    onStageOutput?: (data: StageEventData) => void     // Phase 7B-9：阶段产出物已生成
    onError?: (error: string) => void
    onDone?: () => void
  }
): EventSource {
  const url = `/api/v1/lesson-plans/sse/plans/${planId}/stream?token=${encodeURIComponent(token)}`
  const es = new EventSource(url)

  es.addEventListener('connected', () => { /* 连接建立 */ })

  es.addEventListener('thinking', () => {
    handlers.onThinking?.()
  })

  es.addEventListener('chunk', (e: MessageEvent) => {
    try {
      const event: LPSSEEvent = JSON.parse(e.data)
      if (event.chunk) handlers.onChunk?.(event.chunk)
    } catch { /* 忽略解析错误 */ }
  })

  es.addEventListener('message_done', (e: MessageEvent) => {
    try {
      const event: LPSSEEvent = JSON.parse(e.data)
      if (event.message) handlers.onMessageDone?.(event.message)
    } catch { /* 忽略解析错误 */ }
  })

  es.addEventListener('content_update', (e: MessageEvent) => {
    try {
      const event: LPSSEEvent = JSON.parse(e.data)
      if (event.content) handlers.onContentUpdate?.(event.content)
    } catch { /* 忽略解析错误 */ }
  })

  es.addEventListener('review_done', (e: MessageEvent) => {
    try {
      const event: LPSSEEvent = JSON.parse(e.data)
      if (event.review) handlers.onReviewDone?.(event.review)
    } catch { /* 忽略解析错误 */ }
  })

  es.addEventListener('extraction_hint', (e: MessageEvent) => {
    try {
      const event: LPSSEEvent = JSON.parse(e.data)
      if (event.extraction_hint) handlers.onExtractionHint?.(event.extraction_hint)
    } catch { /* 忽略解析错误 */ }
  })

  // Phase 7B-9：监听阶段化备课工坊事件
  es.addEventListener('stage_started', (e: MessageEvent) => {
    try {
      const event: LPSSEEvent = JSON.parse(e.data)
      if (event.stage_data) handlers.onStageStarted?.(event.stage_data)
    } catch { /* 忽略解析错误 */ }
  })

  es.addEventListener('stage_complete', (e: MessageEvent) => {
    try {
      const event: LPSSEEvent = JSON.parse(e.data)
      if (event.stage_data) handlers.onStageComplete?.(event.stage_data)
    } catch { /* 忽略解析错误 */ }
  })

  es.addEventListener('stage_output', (e: MessageEvent) => {
    try {
      const event: LPSSEEvent = JSON.parse(e.data)
      if (event.stage_data) handlers.onStageOutput?.(event.stage_data)
    } catch { /* 忽略解析错误 */ }
  })

  es.addEventListener('error', (e: MessageEvent) => {
    // 连接级断开（e.data为空）：静默忽略，不报错给用户
    // 这是SSE空闲超时或网络中断的正常行为，不是业务错误
    if (!e.data) return
    try {
      const event: LPSSEEvent = JSON.parse(e.data)
      // 只有明确的业务错误才报给用户
      if (event.error) {
        handlers.onError?.(event.error)
      }
    } catch {
      // JSON解析失败也静默忽略，避免误报
    }
  })

  es.addEventListener('done', () => {
    handlers.onDone?.()
    es.close()
  })

  return es
}

/* ==================== API函数：阶段化备课工坊（Phase 7B-9 新增 6个接口）==================== */

/** 获取系统默认阶段列表 */
export async function getDefaultStages() {
  const resp = await apiClient.get('/lesson-plans/workshop/stages/defaults')
  return resp.data.data as { stages: DefaultStageItem[] }
}

/** 获取教案的阶段进度 */
export async function getStageStatus(planId: string) {
  const resp = await apiClient.get(`/lesson-plans/plans/${planId}/stages`)
  return resp.data.data as StageStatusResponse
}

/** 获取某阶段的产出物 */
export async function getStageOutput(planId: string, stageCode: string) {
  const resp = await apiClient.get(`/lesson-plans/plans/${planId}/stages/${stageCode}/output`)
  return resp.data.data as StageOutputResponse
}

/** 进入下一阶段（可指定目标阶段） */
export async function advanceStage(planId: string, targetStageCode?: string) {
  const resp = await apiClient.post(`/lesson-plans/plans/${planId}/stages/advance`, {
    target_stage_code: targetStageCode || '',
  })
  return resp.data.data as { stage_code: string; stage_name: string }
}

/** 跳过当前阶段 */
export async function skipStage(planId: string, targetStageCode?: string) {
  const resp = await apiClient.post(`/lesson-plans/plans/${planId}/stages/skip`, {
    target_stage_code: targetStageCode || '',
  })
  return resp.data.data as { stage_code: string; stage_name: string }
}

/** 回退到上一阶段 */
export async function backStage(planId: string) {
  const resp = await apiClient.post(`/lesson-plans/plans/${planId}/stages/back`, {})
  return resp.data.data as { stage_code: string; stage_name: string }
}

/* ==================== API函数：萃取队列管理（Phase5新增）==================== */

export async function getExtractions(params?: { limit?: number }) {
  const query = new URLSearchParams()
  if (params?.limit) query.set('limit', String(params.limit))
  const qs = query.toString()
  const resp = await apiClient.get(`/lesson-plans/extractions${qs ? '?' + qs : ''}`)
  return resp.data.data as ExtractionListResponse
}

export async function confirmExtraction(id: string, decision: 'confirmed' | 'rejected') {
  const resp = await apiClient.post(`/lesson-plans/extractions/${id}/confirm`, { decision })
  return resp.data.data as { message: string }
}


/* ==================== API函数：阶段管理（Admin专用，Phase 7B新增）==================== */

/** 获取全部系统阶段（含disabled，admin专用） */
/** 获取全部系统阶段（admin专用，含disabled） — 迭代1：增加prompt_variants字段 */
export async function getAdminStages() {
  const resp = await apiClient.get('/admin/workshop-stages')
  return resp.data.data as { stages: Array<{
    id: string; stage_code: string; stage_name: string; stage_order: number;
    source: string; ai_role: string; system_prompt: string;
    prompt_variants: string;
    output_format: string; component_types: string;
    gate_mode: string; skippable: boolean; status: string;
    created_at: string; updated_at: string;
  }> }
}

/** 更新系统阶段（admin专用） — 迭代1：增加prompt_variants字段 */
export async function updateAdminStage(stageCode: string, data: {
  stage_name: string; ai_role: string; system_prompt: string;
  prompt_variants: string;
  output_format: string; component_types: string;
  gate_mode: string; skippable: boolean; status: string;
}) {
  const resp = await apiClient.put(`/admin/workshop-stages/${stageCode}`, data)
  return resp.data.data
}

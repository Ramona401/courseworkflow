/**
 * coursewares.comic.ts — 知识点漫画浏览器安全协议。
 *
 * 安全边界：
 *   - 浏览器不接收教育域、作者或教材快照作为授权依据；
 *   - 浏览器不接收IAOCI、图片生成提示词、图片关系索引或内部ImageKey；
 *   - 教师只编辑文字、题目、气泡和覆盖层布局；
 *   - 图片重生成继续由后端依据内部稳定索引执行；
 *   - 所有修改请求携带服务端version，防止多标签页静默覆盖。
 */

import apiClient from './client'
import { extractData } from './coursewares.types'

// ==================== 基础状态 ====================

export type CoursewareComicProjectStatus =
  | 'draft' | 'planning' | 'planned' | 'generating'
  | 'ready' | 'inserted' | 'failed' | 'archived'

export type CoursewareComicPanelStatus =
  | 'planned' | 'generating' | 'generated' | 'failed' | 'stale'

export type CoursewareComicLayoutMode = 'grid' | 'spotlight' | 'carousel'

export type CoursewareComicAnswerMode = 'static' | 'click_reveal'

export type CoursewareComicOverlayElementType =
  | 'speech_bubble'
  | 'thought_bubble'
  | 'narration'
  | 'knowledge_card'
  | 'warning_card'
  | 'question_card'
  | 'answer_card'
  | 'caption'
  | 'emphasis'

// ==================== 教材和知识点 ====================

export interface CoursewareComicKnowledgePointSnapshot {
  kp_code: string
  kp_name: string
  content_requirement: string
  academic_requirement: string
  teaching_hint: string
  depth_level: number
  source_ref: string
}

export interface CoursewareComicTextbookUnit {
  id: string
  subject: string
  publisher: string
  grade_num: number
  semester: string
  unit_number: number
  unit_title: string
  lesson_title: string
  content_summary: string

  /**
   * 不同历史后端模型可能返回kp_codes或kp_codes_json。
   * 统一通过parseCoursewareComicUnitKPCodes读取。
   */
  kp_codes?: string[] | string | null
  kp_codes_json?: string | null

  idx_depth_level: number
  source_type: string
  confidence: number
  sort_order: number
}

export interface CoursewareComicPublisherResponse {
  subject: string
  grade: number
  publishers: string[]
}

export interface CoursewareComicTextbookUnitResponse {
  subject: string
  publisher: string
  grade: number
  semester: string
  units: CoursewareComicTextbookUnit[]
  total: number
}

export function parseCoursewareComicUnitKPCodes(
  unit: CoursewareComicTextbookUnit | null,
): string[] {
  if (!unit) {
    return []
  }

  const raw = unit.kp_codes ?? unit.kp_codes_json

  if (Array.isArray(raw)) {
    return raw
      .filter((value): value is string => typeof value === 'string')
      .map(value => value.trim())
      .filter(Boolean)
  }

  if (typeof raw !== 'string' || !raw.trim()) {
    return []
  }

  try {
    const parsed = JSON.parse(raw) as unknown

    if (!Array.isArray(parsed)) {
      return []
    }

    return parsed
      .filter((value): value is string => typeof value === 'string')
      .map(value => value.trim())
      .filter(Boolean)
  } catch {
    return []
  }
}

// ==================== 人物与覆盖层协议 ====================

export interface CoursewareComicCharacter {
  id: string
  name: string
  role: string
  subject_type: 'person' | 'animal' | 'object'
  appearance: string
  default_position: string
  fixed_features: string[]
  forbidden_changes: string[]
  reference_asset_id?: string | null
}

export interface CoursewareComicCharacterBible {
  version: number
  characters: CoursewareComicCharacter[]
}

export interface CoursewareComicDialogue {
  id: string
  character_id: string
  content: string
  bubble_style: string
  emotion: string
}

export interface CoursewareComicBubbleTail {
  type: string

  /**
   * 整格画布0至1坐标，表示尾巴最终指向位置。
   */
  target_x: number
  target_y: number

  /**
   * 气泡自身0至1本地坐标，表示尾巴与边框的连接点。
   * 历史数据没有这两个字段时由编辑器自动推导。
   */
  origin_x?: number
  origin_y?: number
}

export interface CoursewareComicTextStyle {
  font_family: string
  font_size: number
  font_weight: number
  line_height: number
  align: string
  color: string

  /** 历史缺失时按auto处理，由渲染器依据背景明暗自动选择黑字或白字。 */
  color_mode?: 'auto' | 'manual'
  /** 气泡或卡片背景透明度，历史零值或缺失按1处理。 */
  background_opacity?: number

  /**
   * 说话气泡主体与尾巴共用的整体描边宽度，单位为CSS像素。
   * 历史零值或缺失按1处理，合法范围0.5至3。
   */
  outline_width?: number
}

export interface CoursewareComicQuestionContent {
  question: string
  options: string[]
  answer_index: number
  explanation: string
  answer_mode: CoursewareComicAnswerMode
}

export interface CoursewareComicOverlayElement {
  id: string
  type: CoursewareComicOverlayElementType
  content: string
  original_content: string

  speaker_id?: string
  target_character_id?: string
  target_anchor?: string
  style_id: string

  auto_layout_region: string
  priority: number

  x: number
  y: number
  width: number
  height: number
  rotation: number
  z_index: number

  tail?: CoursewareComicBubbleTail | null
  text_style: CoursewareComicTextStyle

  question?: CoursewareComicQuestionContent | null
  original_question?: CoursewareComicQuestionContent | null

  locked: boolean
  content_dirty: boolean
  layout_dirty: boolean
}

export interface CoursewareComicOverlayDocument {
  version: number
  canvas: {
    width: number
    height: number
  }
  elements: CoursewareComicOverlayElement[]
}

// ==================== 浏览器安全项目与分格视图 ====================

export interface CoursewareComicProject {
  id: string
  courseware_id: string
  education_domain: string

  title: string
  subject: string
  grade: string

  publisher: string
  semester: string

  textbook_unit: CoursewareComicTextbookUnit
  knowledge_points: CoursewareComicKnowledgePointSnapshot[]
  knowledge_content: string
  teacher_focus: string

  assistant_id: string | null

  narrative_mode: string
  visual_style: string
  panel_count: number
  layout_mode: CoursewareComicLayoutMode

  page_layout: Record<string, unknown>
  interaction_config: Record<string, unknown>

  character_bible: CoursewareComicCharacterBible
  character_sheet_asset_id: string | null
  character_sheet_url: string

  status: CoursewareComicProjectStatus
  inserted_page_id: string | null
  inserted_page_number_snapshot: number

  version: number
  last_error: string

  created_at: string | null
  updated_at: string | null
}

export interface CoursewareComicPanel {
  id: string
  project_id: string
  panel_no: number

  story_purpose: string
  knowledge_claim: string
  scene_text: string
  character_ids: string[]
  action_text: string
  camera_text: string
  narration_text: string
  dialogues: CoursewareComicDialogue[]
  knowledge_presentation: string

  overlay_document: CoursewareComicOverlayDocument
  overlay_version: number

  status: CoursewareComicPanelStatus
  current_asset_id: string | null
  current_asset_url: string

  version: number
  last_error: string

  created_at: string | null
  updated_at: string | null
}

export interface CoursewareComicProjectDetail {
  project: CoursewareComicProject
  panels: CoursewareComicPanel[]
}

export interface CoursewareComicProjectList {
  projects: CoursewareComicProject[]
  total: number
}

// ==================== 请求和响应 ====================

export interface CreateCoursewareComicProjectInput {
  title: string
  publisher: string
  semester: string
  textbook_unit_id: string
  kp_codes: string[]

  assistant_id: string | null

  narrative_mode: string
  visual_style: string
  panel_count: number
  layout_mode: CoursewareComicLayoutMode
  teacher_focus: string
}

export interface PlanCoursewareComicProjectInput {
  expected_version: number
  teacher_instruction: string
}

export interface UpdateCoursewareComicOverlayInput {
  expected_version: number
  narration_text: string
  overlay_document: CoursewareComicOverlayDocument
}

export interface CoursewareComicGenerationStartResult {
  status: string
  courseware_id: string
  project_id: string
  panel_id?: string
  message: string
}

export interface CoursewareComicPageResult {
  courseware_id: string
  project_id: string
  page_id: string
  page_number: number
  status: string
  created: boolean
  updated: boolean
}

// ==================== 漫画规划助手 ====================

export interface CoursewareComicAssistantOption {
  id: string
  name: string
  avatar_emoji: string
  description: string
  source: 'system' | 'group' | 'personal'
  source_label: string
  subject: string
  grade_range: string
  is_active: boolean
  is_default_here: boolean
}

interface CoursewareComicAssistantListResponse {
  assistants: CoursewareComicAssistantOption[] | null
  total: number
}

// ==================== 路径辅助 ====================

function pathSegment(value: string): string {
  const normalized = value.trim()

  if (!normalized) {
    throw new Error('知识点漫画资源ID不能为空')
  }

  return encodeURIComponent(normalized)
}

function projectEndpoint(
  coursewareId: string,
  projectId?: string,
): string {
  const base =
    `/coursewares/${pathSegment(coursewareId)}/comic-projects`

  return projectId
    ? `${base}/${pathSegment(projectId)}`
    : base
}

// ==================== 教材和助手API ====================

export async function listCoursewareComicPublishers(
  subject: string,
  grade: number,
): Promise<string[]> {
  const response = await apiClient.get(
    '/curriculum/publishers',
    {
      params: {
        subject,
        grade,
      },
    },
  )

  const data =
    extractData<CoursewareComicPublisherResponse>(
      response,
    )

  return data.publishers || []
}

export async function listCoursewareComicTextbookUnits(
  params: {
    subject: string
    publisher: string
    grade: number
    semester?: string
  },
): Promise<CoursewareComicTextbookUnit[]> {
  const response = await apiClient.get(
    '/curriculum/textbook-units',
    {
      params: {
        subject: params.subject,
        publisher: params.publisher,
        grade: params.grade,
        semester: params.semester || '',
      },
    },
  )

  const data =
    extractData<CoursewareComicTextbookUnitResponse>(
      response,
    )

  return data.units || []
}

export async function listCoursewareComicAssistants(
  subject: string,
  grade: string,
): Promise<CoursewareComicAssistantOption[]> {
  const response = await apiClient.get(
    '/ai-assistants',
    {
      params: {
        scene: 'courseware_comic_plan',
        subject,
        grade,
        only_active: true,
      },
    },
  )

  const data =
    extractData<CoursewareComicAssistantListResponse>(
      response,
    )

  return data.assistants || []
}

// ==================== 项目API ====================

export async function listCoursewareComicProjects(
  coursewareId: string,
): Promise<CoursewareComicProjectList> {
  const response = await apiClient.get(
    projectEndpoint(coursewareId),
  )

  const data =
    extractData<CoursewareComicProjectList>(
      response,
    )

  return {
    projects: data.projects || [],
    total: Number.isFinite(data.total)
      ? data.total
      : data.projects?.length || 0,
  }
}

export async function getCoursewareComicProject(
  coursewareId: string,
  projectId: string,
): Promise<CoursewareComicProjectDetail> {
  const response = await apiClient.get(
    projectEndpoint(
      coursewareId,
      projectId,
    ),
  )

  return extractData<CoursewareComicProjectDetail>(
    response,
  )
}

export async function createCoursewareComicProject(
  coursewareId: string,
  input: CreateCoursewareComicProjectInput,
): Promise<CoursewareComicProject> {
  const response = await apiClient.post(
    projectEndpoint(coursewareId),
    input,
  )

  return extractData<CoursewareComicProject>(
    response,
  )
}

export async function planCoursewareComicProject(
  coursewareId: string,
  projectId: string,
  input: PlanCoursewareComicProjectInput,
): Promise<CoursewareComicProjectDetail> {
  const response = await apiClient.post(
    `${projectEndpoint(
      coursewareId,
      projectId,
    )}/plan`,
    input,
    {
      timeout: 300000,
    },
  )

  return extractData<CoursewareComicProjectDetail>(
    response,
  )
}

// ==================== 单格覆盖层编辑API ====================

export async function updateCoursewareComicPanelOverlay(
  coursewareId: string,
  projectId: string,
  panelId: string,
  input: UpdateCoursewareComicOverlayInput,
): Promise<CoursewareComicPanel> {
  const response = await apiClient.put(
    `${projectEndpoint(
      coursewareId,
      projectId,
    )}/panels/${pathSegment(panelId)}/overlay`,
    input,
  )

  return extractData<CoursewareComicPanel>(
    response,
  )
}

// ==================== 图片生成API ====================

export async function generateCoursewareComicProject(
  coursewareId: string,
  projectId: string,
  expectedVersion: number,
): Promise<CoursewareComicGenerationStartResult> {
  const response = await apiClient.post(
    `${projectEndpoint(
      coursewareId,
      projectId,
    )}/generate`,
    {
      expected_version: expectedVersion,
    },
  )

  return extractData<CoursewareComicGenerationStartResult>(
    response,
  )
}

export async function regenerateCoursewareComicPanel(
  coursewareId: string,
  projectId: string,
  panelId: string,
  expectedVersion: number,
  regenerationInstruction = '',
): Promise<CoursewareComicGenerationStartResult> {
  const normalizedInstruction =
    regenerationInstruction.trim()

  const response = await apiClient.post(
    `${projectEndpoint(
      coursewareId,
      projectId,
    )}/panels/${pathSegment(panelId)}/regenerate`,
    {
      expected_version: expectedVersion,

      regeneration_instruction:
        normalizedInstruction,
    },
  )

  return extractData<CoursewareComicGenerationStartResult>(
    response,
  )
}

// ==================== 课件插页API ====================

export async function insertCoursewareComicPage(
  coursewareId: string,
  projectId: string,
  input: {
    expected_version: number
    insert_at: number
  },
): Promise<CoursewareComicPageResult> {
  const response = await apiClient.post(
    `${projectEndpoint(
      coursewareId,
      projectId,
    )}/insert-page`,
    input,
  )

  return extractData<CoursewareComicPageResult>(
    response,
  )
}

export async function syncCoursewareComicPanelPage(
  coursewareId: string,
  projectId: string,
  panelId: string,
  expectedVersion: number,
): Promise<CoursewareComicPageResult> {
  const response = await apiClient.post(
    `${projectEndpoint(
      coursewareId,
      projectId,
    )}/panels/${pathSegment(panelId)}/sync-page`,
    {
      expected_version: expectedVersion,
    },
  )

  return extractData<CoursewareComicPageResult>(
    response,
  )
}

// ==================== 漫画SSE ====================

export type CoursewareComicGenerationStage =
  | 'project_start'
  | 'character_sheet_generating'
  | 'character_sheet_done'
  | 'character_sheet_warning'
  | 'panel_generating'
  | 'panel_done'
  | 'panel_failed'
  | 'project_done'
  | 'project_failed'

export interface CoursewareComicGenerationEvent {
  stage: CoursewareComicGenerationStage | string

  project_id: string
  panel_id?: string
  panel_no?: number
  panel_total?: number

  asset_id?: string
  asset_url?: string
  public_url?: string

  reference_role?: string
  message: string
}

export interface CoursewareComicSSECallbacks {
  onConnected?: () => void
  onReconnected?: () => void
  onEvent?: (
    event: CoursewareComicGenerationEvent,
  ) => void
  onTransportError?: (
    message: string,
  ) => void
}

export interface CoursewareComicSSEConnection {
  close: () => void
}

function isUnknownRecord(
  value: unknown,
): value is Record<string, unknown> {
  return (
    typeof value === 'object' &&
    value !== null &&
    !Array.isArray(value)
  )
}

function parseCoursewareComicGenerationEvent(
  raw: string,
): CoursewareComicGenerationEvent | null {
  try {
    const value = JSON.parse(raw) as unknown

    if (
      !isUnknownRecord(value) ||
      typeof value.stage !== 'string'
    ) {
      return null
    }

    return {
      stage: value.stage,
      project_id:
        typeof value.project_id === 'string'
          ? value.project_id
          : '',
      panel_id:
        typeof value.panel_id === 'string'
          ? value.panel_id
          : undefined,
      panel_no:
        typeof value.panel_no === 'number'
          ? value.panel_no
          : undefined,
      panel_total:
        typeof value.panel_total === 'number'
          ? value.panel_total
          : undefined,
      asset_id:
        typeof value.asset_id === 'string'
          ? value.asset_id
          : undefined,
      asset_url:
        typeof value.asset_url === 'string'
          ? value.asset_url
          : undefined,
      public_url:
        typeof value.public_url === 'string'
          ? value.public_url
          : undefined,
      reference_role:
        typeof value.reference_role === 'string'
          ? value.reference_role
          : undefined,
      message:
        typeof value.message === 'string'
          ? value.message
          : '',
    }
  } catch {
    return null
  }
}

/**
 * 订阅同一课件SSE通道中的comic_generation事件。
 *
 * 生成任务在后端继续运行；浏览器网络中断时执行指数退避重连。
 * 连接不会在project_done后自动关闭，由调用组件按生命周期关闭。
 */
export function subscribeCoursewareComicGeneration(
  coursewareId: string,
  callbacks: CoursewareComicSSECallbacks,
): CoursewareComicSSEConnection {
  const normalizedID = pathSegment(coursewareId)
  const token = localStorage.getItem('token') || ''

  const url =
    `${window.location.origin}/api/v1/sse/courseware/${normalizedID}` +
    `?token=${encodeURIComponent(token)}`

  let eventSource: EventSource | null = null
  let reconnectTimer:
    | ReturnType<typeof setTimeout>
    | null = null

  let retryCount = 0
  let closed = false
  let firstConnection = true

  const maximumRetries = 5

  const connect = () => {
    if (closed) {
      return
    }

    const source = new EventSource(url)
    eventSource = source

    source.addEventListener(
      'connected',
      () => {
        retryCount = 0
        callbacks.onConnected?.()

        if (!firstConnection) {
          callbacks.onReconnected?.()
        }

        firstConnection = false
      },
    )

    source.addEventListener(
      'comic_generation',
      event => {
        const parsed =
          parseCoursewareComicGenerationEvent(
            (event as MessageEvent).data,
          )

        if (parsed) {
          callbacks.onEvent?.(parsed)
        }
      },
    )

    source.onerror = () => {
      if (closed) {
        return
      }

      source.close()

      if (eventSource === source) {
        eventSource = null
      }

      if (retryCount >= maximumRetries) {
        callbacks.onTransportError?.(
          '漫画生成进度连接多次恢复失败。后台任务仍会继续，可刷新项目查看最新结果。',
        )
        return
      }

      const delay =
        Math.min(
          1000 * Math.pow(2, retryCount),
          30000,
        )

      retryCount += 1
      reconnectTimer = setTimeout(connect, delay)
    }
  }

  connect()

  return {
    close: () => {
      closed = true

      if (reconnectTimer) {
        clearTimeout(reconnectTimer)
        reconnectTimer = null
      }

      if (eventSource) {
        eventSource.close()
        eventSource = null
      }
    },
  }
}

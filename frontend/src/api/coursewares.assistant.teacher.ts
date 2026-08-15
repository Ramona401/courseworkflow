/**
 * 课件教学智能体 —— 教师端API
 *
 * 本模块只调用需要教师登录JWT的接口，统一使用现有apiClient：
 *   - 插槽读取、创建、更新、删除；
 *   - 安全上下文预览；
 *   - AI方案草稿生成；
 *   - 首次发布、追加版本、策略和状态管理；
 *   - 为部署所有者创建teacher_preview短时运行会话；
 *   - 为老师端完整回答生成豆包MP3朗读。
 *
 * 教师JWT只用于教师管理接口、创建预览会话和豆包朗读。
 * 创建成功后返回的runtime_token必须交给公开运行API使用，不能重新放入apiClient。
 */

import apiClient from './client'
import { extractData } from './coursewares.types'
import type {
  AssistantRuntimeStartResponse,
  CoursewareAssistantContextPreview,
  CoursewareAssistantDeploymentListResponse,
  CoursewareAssistantDeploymentVersionView,
  CoursewareAssistantDeploymentView,
  CoursewareAssistantPlanResult,
  CoursewareAssistantSlotListResponse,
  CoursewareAssistantSlotView,
  CreateCoursewareAssistantSlotRequest,
  GenerateCoursewareAssistantPlanRequest,
  PublishCoursewareAssistantDeploymentRequest,
  UpdateCoursewareAssistantDeploymentPolicyRequest,
  UpdateCoursewareAssistantSlotRequest,
} from './coursewares.assistant.types'

/**
 * 必须与后端coursewareAssistantSlotRequestMaxBytes保持一致。
 *
 * Nginx站点总请求上限为55MB，因此教学智能体保存仍由更严格的
 * 业务专属4MiB限制负责保护。
 */
const COURSEWARE_ASSISTANT_SLOT_REQUEST_MAX_BYTES = 4 * 1024 * 1024

export interface CoursewareAssistantTTSResult {
  audio: Blob
  voiceCode: string
  voiceName: string
  language: string
  cacheHit: boolean
}

export type CoursewareAssistantTTSCharacter = 'female' | 'male'

function pathSegment(value: string): string {
  const normalized = value.trim()

  if (!normalized) {
    throw new Error('教学智能体资源ID不能为空')
  }

  return encodeURIComponent(normalized)
}

/**
 * 保存前按照浏览器实际发送的UTF-8 JSON字节数执行镜像检查。
 *
 * JavaScript字符串length统计的是UTF-16代码单元，不能准确代表
 * 中文JSON的网络字节数，因此必须使用TextEncoder。
 */
function assertCoursewareAssistantSlotRequestSize(
  request:
    | CreateCoursewareAssistantSlotRequest
    | UpdateCoursewareAssistantSlotRequest,
): void {
  let encoded: string

  try {
    encoded = JSON.stringify(request)
  } catch {
    throw new Error('教学智能体方案无法序列化，请刷新页面后重试')
  }

  const byteLength = new TextEncoder().encode(encoded).byteLength

  if (byteLength <= COURSEWARE_ASSISTANT_SLOT_REQUEST_MAX_BYTES) {
    return
  }

  const actualMiB = (byteLength / 1024 / 1024).toFixed(2)

  throw new Error(
    `当前教学智能体方案约${actualMiB} MiB，超过4 MiB保存上限。请减少过长的互动步骤、分层提示或学习困难方案后再保存。`,
  )
}

function responseHeader(headers: unknown, name: string): string {
  if (!headers || typeof headers !== 'object') {
    return ''
  }

  const candidate = headers as {
    get?: (headerName: string) => unknown
    [key: string]: unknown
  }

  const getterValue = candidate.get?.(name)

  if (typeof getterValue === 'string') {
    return getterValue.trim()
  }

  const directValue = candidate[name.toLowerCase()] ?? candidate[name]

  return typeof directValue === 'string' ? directValue.trim() : ''
}

function coursewareAssistantTTSVoiceName(voiceCode: string): string {
  switch (voiceCode) {
  case 'zh_female_vv_uranus_bigtts':
    return 'vivi 2.0'
  case 'zh_male_m191_uranus_bigtts':
    return '云舟'
  case 'en_male_tim_uranus_bigtts':
    return 'Tim'
  default:
    return voiceCode || '豆包自然音色'
  }
}

// ==================== 插槽读取 ====================

export async function listCoursewareAssistantSlots(
  coursewareId: string,
): Promise<CoursewareAssistantSlotListResponse> {
  const response = await apiClient.get(
    `/coursewares/${pathSegment(coursewareId)}/assistant-slots`,
  )

  const data = extractData<{
    slots: CoursewareAssistantSlotView[] | null
    total: number
  }>(response)

  const slots = data.slots || []

  return {
    slots,
    total: Number.isFinite(data.total) ? data.total : slots.length,
  }
}

export async function getCoursewareAssistantSlot(
  coursewareId: string,
  pageId: string,
): Promise<CoursewareAssistantSlotView> {
  const response = await apiClient.get(
    `/coursewares/${pathSegment(coursewareId)}/pages/${pathSegment(pageId)}/assistant-slot`,
  )

  return extractData<CoursewareAssistantSlotView>(response)
}

export async function getCoursewareAssistantContextPreview(
  coursewareId: string,
  pageId: string,
): Promise<CoursewareAssistantContextPreview> {
  const response = await apiClient.get(
    `/coursewares/${pathSegment(coursewareId)}/pages/${pathSegment(pageId)}/assistant-context`,
  )

  return extractData<CoursewareAssistantContextPreview>(response)
}

// ==================== 方案草稿 ====================

export async function generateCoursewareAssistantPlan(
  coursewareId: string,
  pageId: string,
  request: GenerateCoursewareAssistantPlanRequest,
): Promise<CoursewareAssistantPlanResult> {
  const response = await apiClient.post(
    `/coursewares/${pathSegment(coursewareId)}/pages/${pathSegment(pageId)}/assistant-plan`,
    request,
    {
      timeout: 300000,
    },
  )

  return extractData<CoursewareAssistantPlanResult>(response)
}

// ==================== 插槽写入 ====================

export async function createCoursewareAssistantSlot(
  coursewareId: string,
  pageId: string,
  request: CreateCoursewareAssistantSlotRequest,
): Promise<CoursewareAssistantSlotView> {
  assertCoursewareAssistantSlotRequestSize(request)

  const response = await apiClient.post(
    `/coursewares/${pathSegment(coursewareId)}/pages/${pathSegment(pageId)}/assistant-slot`,
    request,
  )

  return extractData<CoursewareAssistantSlotView>(response)
}

export async function updateCoursewareAssistantSlot(
  coursewareId: string,
  slotId: string,
  request: UpdateCoursewareAssistantSlotRequest,
): Promise<CoursewareAssistantSlotView> {
  assertCoursewareAssistantSlotRequestSize(request)

  const response = await apiClient.put(
    `/coursewares/${pathSegment(coursewareId)}/assistant-slots/${pathSegment(slotId)}`,
    request,
  )

  return extractData<CoursewareAssistantSlotView>(response)
}

export async function deleteCoursewareAssistantSlot(
  coursewareId: string,
  slotId: string,
): Promise<{ message: string }> {
  const response = await apiClient.delete(
    `/coursewares/${pathSegment(coursewareId)}/assistant-slots/${pathSegment(slotId)}`,
  )

  return extractData<{ message: string }>(response)
}

// ==================== 部署发布 ====================

export async function publishCoursewareAssistantDeployment(
  coursewareId: string,
  pageId: string,
  request: PublishCoursewareAssistantDeploymentRequest,
): Promise<CoursewareAssistantDeploymentView> {
  const response = await apiClient.post(
    `/coursewares/${pathSegment(coursewareId)}/pages/${pathSegment(pageId)}/assistant-deployment`,
    request,
  )

  return extractData<CoursewareAssistantDeploymentView>(response)
}

export async function listCoursewareAssistantDeployments(
  coursewareId: string,
): Promise<CoursewareAssistantDeploymentListResponse> {
  const response = await apiClient.get(
    `/coursewares/${pathSegment(coursewareId)}/assistant-deployments`,
  )

  const data = extractData<{
    deployments: CoursewareAssistantDeploymentView[] | null
    total: number
  }>(response)

  const deployments = data.deployments || []

  return {
    deployments,
    total: Number.isFinite(data.total) ? data.total : deployments.length,
  }
}

// ==================== 不可变版本 ====================

export async function listCoursewareAssistantDeploymentVersions(
  deploymentId: string,
): Promise<CoursewareAssistantDeploymentVersionView[]> {
  const response = await apiClient.get(
    `/assistant-deployments/${pathSegment(deploymentId)}/versions`,
  )

  const versions = extractData<CoursewareAssistantDeploymentVersionView[] | null>(
    response,
  )

  return versions || []
}

export async function publishCoursewareAssistantDeploymentVersion(
  deploymentId: string,
): Promise<CoursewareAssistantDeploymentVersionView> {
  const response = await apiClient.post(
    `/assistant-deployments/${pathSegment(deploymentId)}/versions`,
  )

  return extractData<CoursewareAssistantDeploymentVersionView>(response)
}

// ==================== 部署状态与策略 ====================

export async function pauseCoursewareAssistantDeployment(
  deploymentId: string,
): Promise<CoursewareAssistantDeploymentView> {
  const response = await apiClient.post(
    `/assistant-deployments/${pathSegment(deploymentId)}/pause`,
  )

  return extractData<CoursewareAssistantDeploymentView>(response)
}

export async function resumeCoursewareAssistantDeployment(
  deploymentId: string,
): Promise<CoursewareAssistantDeploymentView> {
  const response = await apiClient.post(
    `/assistant-deployments/${pathSegment(deploymentId)}/resume`,
  )

  return extractData<CoursewareAssistantDeploymentView>(response)
}

export async function revokeCoursewareAssistantDeployment(
  deploymentId: string,
): Promise<CoursewareAssistantDeploymentView> {
  const response = await apiClient.post(
    `/assistant-deployments/${pathSegment(deploymentId)}/revoke`,
  )

  return extractData<CoursewareAssistantDeploymentView>(response)
}

export async function updateCoursewareAssistantDeploymentPolicy(
  deploymentId: string,
  request: UpdateCoursewareAssistantDeploymentPolicyRequest,
): Promise<CoursewareAssistantDeploymentView> {
  const response = await apiClient.put(
    `/assistant-deployments/${pathSegment(deploymentId)}/policy`,
    request,
  )

  return extractData<CoursewareAssistantDeploymentView>(response)
}

// ==================== 教师内部预览 ====================

/**
 * 为当前登录教师本人拥有的部署创建teacher_preview会话。
 *
 * 请求正文为空，部署所有者和付费身份由后端通过JWT与数据库确定。
 * 返回的runtime_token只保存在当前页面内存中，不写localStorage。
 */
export async function startCoursewareAssistantPreviewSession(
  deploymentId: string,
): Promise<AssistantRuntimeStartResponse> {
  const response = await apiClient.post(
    `/assistant-deployments/${pathSegment(deploymentId)}/preview-session`,
  )

  return extractData<AssistantRuntimeStartResponse>(response)
}

// ==================== 老师端豆包朗读 ====================

/**
 * 为当前教师本人拥有的教学智能体部署生成豆包MP3朗读。
 *
 * operationId由浏览器为一次完整回答生成并在恢复重试时复用；
 * character只允许female/male，后端仍负责把人物安全映射为批准的豆包speaker；
 * 女老师中文使用vivi 2.0，男老师中文使用云舟，英文继续使用Tim。
 */
export async function synthesizeCoursewareAssistantSpeech(
  deploymentId: string,
  text: string,
  operationId: string,
  character: CoursewareAssistantTTSCharacter,
  signal?: AbortSignal,
): Promise<CoursewareAssistantTTSResult> {
  const normalizedText = text.trim()
  const normalizedOperationID = operationId.trim()

  if (!normalizedText) {
    throw new Error('当前回答没有可朗读的文字')
  }

  if (!normalizedOperationID) {
    throw new Error('朗读任务标识为空，请重新朗读')
  }

  const response = await apiClient.post<Blob>(
    `/assistant-deployments/${pathSegment(deploymentId)}/tts`,
    {
      text: normalizedText,
      operation_id: normalizedOperationID,
      character,
    },
    {
      responseType: 'blob',
      timeout: 120000,
      signal,
    },
  )

  const audio = response.data

  if (!(audio instanceof Blob) || audio.size < 100) {
    throw new Error('豆包朗读音频未正确返回')
  }

  const voiceCode = responseHeader(response.headers, 'x-tedna-tts-voice')
  const language = responseHeader(response.headers, 'x-tedna-tts-language')
  const cacheValue = responseHeader(response.headers, 'x-tedna-tts-cache')

  return {
    audio,
    voiceCode,
    voiceName: coursewareAssistantTTSVoiceName(voiceCode),
    language,
    cacheHit: cacheValue.toLowerCase() === 'true',
  }
}

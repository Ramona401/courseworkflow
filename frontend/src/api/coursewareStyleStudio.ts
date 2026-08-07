/**
 * AI美术风格工作室API。
 *
 * 职责：
 *   - 创建和恢复风格共创会话；
 *   - 发送文字要求或参考图提取请求；
 *   - 上传课程级参考图片；
 *   - 按当前reference_mode生成三类测试预览；
 *   - 按当前reference_mode确认正式风格锚点；
 *   - 读取课件资产，为预览asset_id补齐可显示图片地址。
 *
 * 一致性要求：
 *   - 生成预览必须显式提交当前界面模式；
 *   - 确认锚点必须显式提交当前界面模式；
 *   - 后端会将请求模式与会话持久化模式进行事务级核验。
 *
 * 资产URL兼容性：
 *   - 新资产协议优先public_oss_url，其次oss_url；
 *   - 兼容部分接口或历史数据中的public_url、url；
 *   - 相对uploads路径统一归一为站点根路径；
 *   - 不在前端拼接OSS域名、密钥或桶信息。
 */

import client from './client'
import type { ApiResponse } from './client'

export type CoursewareStyleReferenceMode =
  | 'style_only'
  | 'style_character'
  | 'inspiration'

export type CoursewareStyleSessionStatus =
  | 'draft'
  | 'previewing'
  | 'confirmed'
  | 'archived'

export type CoursewareStylePreviewType =
  | 'character'
  | 'object'
  | 'diagram'

export type CoursewareStylePreviewStatus =
  | 'pending'
  | 'generating'
  | 'generated'
  | 'failed'
  | 'stale'

export interface CoursewareStyleSession {
  id: string
  courseware_id: string
  user_id: string
  status: CoursewareStyleSessionStatus
  reference_mode: CoursewareStyleReferenceMode
  reference_asset_id: string | null
  confirmed_asset_id: string | null
  style_aoci_text: string
  style_summary: string
  version: number
  confirmed_at: string | null
  created_at: string | null
  updated_at: string | null
}

export interface CoursewareStyleMessage {
  id: string
  session_id: string
  courseware_id: string
  role: 'user' | 'assistant'
  content: string
  reference_asset_id: string | null
  style_aoci_text: string
  sequence_no: number
  created_at: string | null
}

export interface CoursewareStylePreview {
  id: string
  session_id: string
  courseware_id: string
  preview_type: CoursewareStylePreviewType
  asset_id: string | null
  generation_prompt: string
  status: CoursewareStylePreviewStatus
  last_error: string
  version: number
  created_at: string | null
  updated_at: string | null
}

export interface CoursewareStyleStudioState {
  session: CoursewareStyleSession | null
  messages: CoursewareStyleMessage[]
  previews: CoursewareStylePreview[]
}

export interface CoursewareStyleTurnResult {
  reply: string
  state: CoursewareStyleStudioState
}

export interface CoursewareStyleReferenceAssetResult {
  asset_id: string
  url: string
  public_url: string
  file_name: string
  file_size: number
  mime_type: string
}

/**
 * 风格工作室使用的轻量资产协议。
 *
 * oss_url/public_oss_url是CoursewareAsset正式字段；
 * url/public_url用于兼容上传响应、安全视图或历史接口。
 */
export interface CoursewareStyleAsset {
  id: string
  courseware_id: string
  page_id: string | null
  placeholder_id: string
  asset_type: string
  generation_prompt: string
  oss_url?: string
  public_oss_url?: string
  url?: string
  public_url?: string
  mime_type: string
  status: string
}

export interface CreateCoursewareStyleSessionRequest {
  reference_mode?: CoursewareStyleReferenceMode
  reference_asset_id?: string
}

export interface CoursewareStyleTurnRequest {
  content: string
  reference_mode?: CoursewareStyleReferenceMode
  reference_asset_id?: string
}

/**
 * 生成预览请求。
 *
 * reference_mode必须来自当前界面选择，
 * 后端会在生图前持久化并规范化该模式。
 */
export interface GenerateCoursewareStylePreviewsRequest {
  reference_mode: CoursewareStyleReferenceMode
  operation_id?: string
}

/**
 * 确认风格请求。
 *
 * 后端会核验：
 *   - 请求模式等于会话持久化模式；
 *   - 图片来自当前会话；
 *   - 非固定主体模式不能确认原始参考图。
 */
export interface ConfirmCoursewareStyleSessionRequest {
  asset_id: string
  reference_mode: CoursewareStyleReferenceMode
}

function createCoursewareStylePreviewOperationID(): string {
  const runtimeCrypto =
    globalThis.crypto

  if (
    !runtimeCrypto ||
    typeof runtimeCrypto.randomUUID !==
      'function'
  ) {
    throw new Error(
      '当前浏览器不支持安全的预览任务标识，请升级浏览器后重试',
    )
  }

  return runtimeCrypto.randomUUID()
}

function extractData<T>(
  response: {
    data: ApiResponse<T>
  },
): T {
  if (response.data.data === undefined) {
    throw new Error(
      response.data.message ||
        '接口未返回有效数据',
    )
  }

  return response.data.data
}

/** 恢复当前活动风格会话。 */
export async function getActiveCoursewareStyleStudio(
  coursewareId: string,
): Promise<CoursewareStyleStudioState> {
  const response = await client.get<
    ApiResponse<CoursewareStyleStudioState>
  >(
    `/coursewares/${coursewareId}/style-studio`,
  )

  return extractData(response)
}

/** 创建新会话；后端会自动归档原活动会话。 */
export async function createCoursewareStyleStudioSession(
  coursewareId: string,
  request: CreateCoursewareStyleSessionRequest = {},
): Promise<CoursewareStyleStudioState> {
  const response = await client.post<
    ApiResponse<CoursewareStyleStudioState>
  >(
    `/coursewares/${coursewareId}/style-studio/sessions`,
    request,
  )

  return extractData(response)
}

/** 读取指定会话完整状态。 */
export async function getCoursewareStyleStudioSession(
  coursewareId: string,
  sessionId: string,
): Promise<CoursewareStyleStudioState> {
  const response = await client.get<
    ApiResponse<CoursewareStyleStudioState>
  >(
    `/coursewares/${coursewareId}/style-studio/sessions/${sessionId}`,
  )

  return extractData(response)
}

/** 发送一轮文字共创或参考图提取请求。 */
export async function sendCoursewareStyleStudioTurn(
  coursewareId: string,
  sessionId: string,
  request: CoursewareStyleTurnRequest,
): Promise<CoursewareStyleTurnResult> {
  const response = await client.post<
    ApiResponse<CoursewareStyleTurnResult>
  >(
    `/coursewares/${coursewareId}/style-studio/sessions/${sessionId}/messages`,
    request,
    {
      // 多模态读取和AI风格整理可能超过普通接口120秒限制。
      timeout: 600000,
    },
  )

  return extractData(response)
}

/** 上传课程级风格参考图。 */
export async function uploadCoursewareStyleReference(
  coursewareId: string,
  file: File,
): Promise<CoursewareStyleReferenceAssetResult> {
  const formData = new FormData()
  formData.append('file', file)

  const response = await client.post<
    ApiResponse<CoursewareStyleReferenceAssetResult>
  >(
    `/coursewares/${coursewareId}/style-studio/upload-reference`,
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      timeout: 180000,
    },
  )

  return extractData(response)
}

/**
 * 生成角色、知识对象和教学图解三类预览。
 *
 * 必须显式提交当前reference_mode，
 * 避免界面已切换模式但后端仍按旧模式生成。
 */
export async function generateCoursewareStylePreviews(
  coursewareId: string,
  sessionId: string,
  request: GenerateCoursewareStylePreviewsRequest,
): Promise<CoursewareStyleStudioState> {
  // 一次函数调用对应老师的一次点击。
  // Axios在同一次请求内部重放时沿用同一个operation_id；
  // 老师再次点击会重新调用本函数并获得新UUID。
  const operationID =
    request.operation_id?.trim() ||
    createCoursewareStylePreviewOperationID()

  const payload:
    GenerateCoursewareStylePreviewsRequest = {
      ...request,
      operation_id: operationID,
    }

  const response = await client.post<
    ApiResponse<CoursewareStyleStudioState>
  >(
    `/coursewares/${coursewareId}/style-studio/sessions/${sessionId}/previews`,
    payload,
    {
      // 三张图片依次生成，需要使用服务器同步AI请求上限。
      timeout: 600000,
    },
  )

  return extractData(response)
}

/**
 * 确认选定图片和当前IAOCI为课程正式风格锚点。
 *
 * referenceMode必须是当前界面模式。
 */
export async function confirmCoursewareStyleStudio(
  coursewareId: string,
  sessionId: string,
  assetId: string,
  referenceMode: CoursewareStyleReferenceMode,
): Promise<CoursewareStyleStudioState> {
  const request:
    ConfirmCoursewareStyleSessionRequest = {
      asset_id: assetId,
      reference_mode: referenceMode,
    }

  const response = await client.post<
    ApiResponse<CoursewareStyleStudioState>
  >(
    `/coursewares/${coursewareId}/style-studio/sessions/${sessionId}/confirm`,
    request,
    {
      timeout: 180000,
    },
  )

  return extractData(response)
}

/** 归档当前未确认会话。 */
export async function archiveCoursewareStyleStudioSession(
  coursewareId: string,
  sessionId: string,
): Promise<void> {
  await client.delete(
    `/coursewares/${coursewareId}/style-studio/sessions/${sessionId}`,
  )
}

/**
 * 读取课件全部资产。
 *
 * 风格工作室状态为保持轻量只返回asset_id；
 * 本接口用来把参考图和三类预览图映射成实际图片地址。
 */
export async function listCoursewareStyleAssets(
  coursewareId: string,
): Promise<CoursewareStyleAsset[]> {
  const response = await client.get<
    ApiResponse<{
      assets: CoursewareStyleAsset[]
      total: number
    }>
  >(
    `/coursewares/${coursewareId}/assets`,
  )

  const data = extractData(response)

  return Array.isArray(data.assets)
    ? data.assets
    : []
}

/**
 * 规范化一条后端资产地址。
 *
 * 支持：
 *   - https://或http://公网地址；
 *   - //开头的协议相对地址；
 *   - /uploads/...站点相对地址；
 *   - uploads/...或./uploads/...历史相对地址。
 *
 * 不对data:、blob:或其它非HTTP协议开放，避免把异常数据直接交给img标签。
 */
function normalizeCoursewareStyleAssetURL(
  value: string | undefined,
): string {
  const url = value?.trim() || ''

  if (!url) return ''

  if (
    url.startsWith('https://') ||
    url.startsWith('http://')
  ) {
    return url
  }

  if (url.startsWith('//')) {
    const protocol =
      typeof window !== 'undefined'
        ? window.location.protocol
        : 'https:'

    return `${protocol}${url}`
  }

  if (url.startsWith('/')) {
    return url
  }

  const normalized =
    url
      .replace(/^\.\//, '')
      .replace(/^\/+/, '')

  if (
    normalized.startsWith('uploads/')
  ) {
    return `/${normalized}`
  }

  return ''
}

/**
 * 统一解析课件资产可显示地址。
 *
 * 优先读取稳定的OSS公网快照；未上云时回退本地uploads地址。
 * public_url/url仅作为接口兼容字段，不改变正式OSS双桶裁决。
 */
export function resolveCoursewareStyleAssetURL(
  asset: CoursewareStyleAsset | undefined,
): string {
  if (!asset) return ''

  const candidates = [
    asset.public_oss_url,
    asset.public_url,
    asset.oss_url,
    asset.url,
  ]

  for (const candidate of candidates) {
    const resolved =
      normalizeCoursewareStyleAssetURL(
        candidate,
      )

    if (resolved) {
      return resolved
    }
  }

  return ''
}

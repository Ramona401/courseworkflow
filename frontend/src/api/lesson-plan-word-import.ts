/**
 * lesson-plan-word-import.ts — 保留原Word格式的导入与下载前端协议
 *
 * 与普通浏览器端Word纯文本导入分离：
 *   - previewLessonPlanWordImport：上传原DOCX，由后端安全解析并私有保存；
 *   - importWordFidelityPlan：只提交短时会话ID和课程元信息；
 *   - downloadLessonPlanWordFidelity：下载后端已校验的当前原格式DOCX；
 *   - 浏览器不提交或接收服务器路径、文件哈希或OOXML结构；
 *   - 正式确认时后端忽略浏览器重新提交的正文，使用可信会话正文；
 *   - 下载只适用于作者本人且Word文档仍与当前平台正文同步的情况。
 */

import apiClient from './client'
import type {
  ConversationMessage,
  LessonPlan,
} from './lesson-plans.types'

export const MAX_WORD_FIDELITY_FILE_SIZE =
  30 * 1024 * 1024

/**
 * Word结构解析和原格式文件下载可能包含较多表格、图片和公式，
 * 为相关接口提供独立的三分钟客户端超时。
 */
const WORD_FIDELITY_REQUEST_TIMEOUT_MS =
  180000

/**
 * 已有教案导入的四种前端来源。
 *
 * docx仍表示浏览器纯文本提取；
 * docx_fidelity表示后端安全解析并保留原DOCX结构。
 */
export type LessonPlanImportSourceType =
  | 'paste'
  | 'docx'
  | 'docx_fidelity'
  | 'pdf'

export interface LessonPlanWordPreviewMetrics {
  block_count?: number
  editable_block_count?: number
  paragraph_count?: number
  table_count?: number
  row_count?: number
  cell_count?: number
  image_count?: number
  formula_count?: number
  warning_count?: number
}

export interface LessonPlanWordPreviewWarning {
  code?: string
  message?: string
  detail?: string
  block_id?: string
  [key: string]: unknown
}

export interface LessonPlanWordImportPreview {
  session_id: string
  status: string
  original_file_name: string
  file_size: number
  parser_version: string
  structure_schema_version: number
  semantic_markdown: string
  metrics: LessonPlanWordPreviewMetrics
  warnings: Array<
    LessonPlanWordPreviewWarning | string
  >
  document: unknown
  expires_at: string
  can_confirm: boolean
}

export interface LessonPlanWordDocumentSummary {
  id: string
  lesson_plan_id: string
  education_domain: string
  status: string
  version: number
  source_format: string
  original_file_name: string
  parser_version: string
  structure_schema_version: number
  last_change_source: string
  last_change_summary: string
  generated_at: string
  created_at: string
  updated_at: string
}

export interface ImportWordFidelityPlanRequest {
  subject: string
  grade: string
  topic: string
  duration_minutes?: number

  /**
   * 后端确认时会用可信会话正文覆盖此字段。
   * 前端固定传空字符串，避免把浏览器预览正文误当成正式事实源。
   */
  content_markdown: string

  source_type: 'docx_fidelity'
  word_import_session_id: string
  recipe_id?: string
  group_id?: string
  textbook_page_ids?: string[]
}

export interface ImportWordFidelityPlanResponse {
  plan: LessonPlan
  opening_message: ConversationMessage
  skipped_stages: string[]
  word_document?: LessonPlanWordDocumentSummary
}

export function validateWordFidelityFile(
  file: File,
): string {
  const lowerName =
    file.name.toLowerCase()

  if (!lowerName.endsWith('.docx')) {
    return '仅支持普通 .docx 文档，不支持旧版 .doc 或带宏文档'
  }

  if (file.size <= 0) {
    return 'Word文档为空，请重新选择'
  }

  if (
    file.size >
    MAX_WORD_FIDELITY_FILE_SIZE
  ) {
    return 'Word文档超过30MB，请压缩图片或拆分后重新上传'
  }

  return ''
}

/**
 * 后端安全预解析原DOCX。
 *
 * 不设置Content-Type。apiClient的请求拦截器检测到FormData后，
 * 会删除JSON默认值，由浏览器自动生成带boundary的multipart请求头。
 */
export async function previewLessonPlanWordImport(
  file: File,
): Promise<LessonPlanWordImportPreview> {
  const validationError =
    validateWordFidelityFile(file)

  if (validationError) {
    throw new Error(validationError)
  }

  const formData = new FormData()
  formData.append(
    'file',
    file,
    file.name,
  )

  const response = await apiClient.post(
    '/lesson-plans/plans/import-word/preview',
    formData,
    {
      timeout:
        WORD_FIDELITY_REQUEST_TIMEOUT_MS,
    },
  )

  return response.data.data as
    LessonPlanWordImportPreview
}

/**
 * 确认创建正式教案并进入现有AI评审流程。
 */
export async function importWordFidelityPlan(
  request: ImportWordFidelityPlanRequest,
): Promise<ImportWordFidelityPlanResponse> {
  const response = await apiClient.post(
    '/lesson-plans/plans/import-existing',
    request,
  )

  return response.data.data as
    ImportWordFidelityPlanResponse
}

/**
 * 下载作者本人的当前原格式Word文档。
 *
 * apiClient对blob响应会跳过JSON业务信封校验并返回完整Axios响应。
 * 后端失败时，统一拦截器仍会解析错误正文并抛出教师可读的Error。
 */
export async function downloadLessonPlanWordFidelity(
  lessonPlanID: string,
): Promise<void> {
  const normalizedID =
    lessonPlanID.trim()

  if (!normalizedID) {
    throw new Error('教案ID不能为空')
  }

  const response = await apiClient.get(
    `/lesson-plans/plans/${encodeURIComponent(normalizedID)}/word-document/download`,
    {
      responseType: 'blob',
      timeout:
        WORD_FIDELITY_REQUEST_TIMEOUT_MS,
    },
  )

  const blob = response.data as Blob
  if (!(blob instanceof Blob) || blob.size <= 0) {
    throw new Error('原格式Word下载结果为空')
  }

  const contentDisposition =
    response.headers?.['content-disposition'] as
      | string
      | undefined

  const fileName =
    resolveLessonPlanWordDownloadFileName(
      contentDisposition,
    )

  const objectURL =
    URL.createObjectURL(blob)

  try {
    const anchor =
      document.createElement('a')

    anchor.href = objectURL
    anchor.download = fileName
    anchor.style.display = 'none'

    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
  } finally {
    // click触发后延迟释放，避免部分浏览器尚未读取对象URL就被撤销。
    window.setTimeout(
      () => URL.revokeObjectURL(objectURL),
      1000,
    )
  }
}

function resolveLessonPlanWordDownloadFileName(
  contentDisposition?: string,
): string {
  const fallbackName =
    '原格式教案.docx'

  if (!contentDisposition) {
    return fallbackName
  }

  const utf8Match =
    contentDisposition.match(
      /filename\*=UTF-8''([^;]+)/i,
    )

  if (utf8Match?.[1]) {
    try {
      return sanitizeDownloadedWordFileName(
        decodeURIComponent(
          utf8Match[1].trim(),
        ),
      )
    } catch {
      // RFC 5987文件名解码失败时继续尝试普通filename。
    }
  }

  const plainMatch =
    contentDisposition.match(
      /filename="?([^";]+)"?/i,
    )

  if (plainMatch?.[1]) {
    return sanitizeDownloadedWordFileName(
      plainMatch[1],
    )
  }

  return fallbackName
}

function sanitizeDownloadedWordFileName(
  fileName: string,
): string {
  const normalized =
    fileName
      .replace(/[\\/:*?"<>|]/g, '')
      .replace(/[\u0000-\u001F\u007F]/g, '')
      .trim()
      .slice(0, 200)

  if (!normalized) {
    return '原格式教案.docx'
  }

  return normalized
    .toLowerCase()
    .endsWith('.docx')
    ? normalized
    : `${normalized}.docx`
}

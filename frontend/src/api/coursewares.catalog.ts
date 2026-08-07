/**
 * coursewares.catalog.ts
 *
 * 课件模板、组件库、方案预设和模板AI工作流API。
 */

import type {
  ResourceEducationDomain,
} from '@/education-domain/types'

import apiClient from './client'
import { extractData } from './coursewares.types'
import type {
  CWComponentListItem,
  CWComponentFull,
  CoursewareTemplate,
  SeedResult,
  SchemePreset,
  RefineSSECallbacks,
  ExtractSSECallbacks,
  RefineHistoryEntry,
  PublishTargetsResponse,
} from './coursewares.types'

// ==================== 风格模板 ====================

export async function getCWTemplates(): Promise<CoursewareTemplate[]> {
  const response = await apiClient.get('/courseware-templates')
  return extractData(response)
}

export async function getCWTemplatePreview(
  id: string,
): Promise<CoursewareTemplate> {
  const response = await apiClient.get(
    `/courseware-templates/${id}/preview`,
  )
  return extractData(response)
}

// ==================== 课件组件库 ====================

export async function getCWComponents(params?: {
  component_type?: string
  subject_scope?: string
  grade_scope?: string
  education_domain?: ResourceEducationDomain
  limit?: number
  offset?: number
}): Promise<{
  components: CWComponentListItem[]
  total: number
}> {
  const response = await apiClient.get(
    '/courseware-components',
    {
      params,
    },
  )

  return extractData(response)
}

export async function createCWComponent(data: {
  name: string
  description?: string
  component_type: string
  code_content: string
  preview_html?: string
  subject_scope?: string
  grade_scope?: string
  education_domain?: ResourceEducationDomain
}): Promise<CWComponentListItem> {
  const response = await apiClient.post(
    '/courseware-components',
    data,
  )

  return extractData(response)
}

export async function getCWComponent(
  id: string,
): Promise<CWComponentFull> {
  const response = await apiClient.get(
    `/courseware-components/${id}`,
  )

  return extractData(response)
}

export async function updateCWComponent(
  id: string,
  data: Record<string, unknown>,
): Promise<void> {
  await apiClient.put(
    `/courseware-components/${id}`,
    data,
  )
}

export async function deleteCWComponent(
  id: string,
): Promise<void> {
  await apiClient.delete(
    `/courseware-components/${id}`,
  )
}

export async function matchCWComponents(data: {
  component_type?: string
  subject_scope?: string
  grade_scope?: string
  interaction_level?: number
  visual_format?: string
  limit?: number
  education_domain?: Exclude<
    ResourceEducationDomain,
    'common'
  >
}): Promise<CWComponentListItem[]> {
  const response = await apiClient.post(
    '/courseware-components/match',
    data,
  )

  return extractData(response)
}

export async function seedCoursewareData(
  force?: boolean,
): Promise<SeedResult> {
  const response = await apiClient.post(
    '/admin/courseware-seed',
    {
      force: !!force,
    },
  )

  return extractData(response)
}

// ==================== Admin模板管理 ====================

export async function createCWTemplate(data: {
  name: string
  description?: string
  style_category: string
  color_scheme?: string
  css_variables?: string
  sample_pages?: string
  preview_urls?: string
}): Promise<CoursewareTemplate> {
  const response = await apiClient.post(
    '/admin/courseware-templates',
    data,
  )

  return extractData(response)
}

export async function updateCWTemplate(
  id: string,
  data: Record<string, unknown>,
): Promise<void> {
  await apiClient.put(
    `/admin/courseware-templates/${id}`,
    data,
  )
}

export async function deleteCWTemplate(
  id: string,
): Promise<void> {
  await apiClient.delete(
    `/admin/courseware-templates/${id}`,
  )
}

// ==================== 步骤回退与方案预设 ====================

export async function rollbackCWStatus(
  coursewareId: string,
  targetStatus: string,
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/rollback-status`,
    {
      target_status: targetStatus,
    },
  )
}

export async function refineCWIndex(
  coursewareId: string,
  feedback: string,
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/refine-index`,
    {
      feedback,
    },
  )
}

export async function getSchemePresets(): Promise<SchemePreset[]> {
  const response = await apiClient.get('/courseware-presets')
  return extractData(response)
}

// ==================== 个人模板与AI模板 ====================

export async function getCWTemplatesWithUser(): Promise<CoursewareTemplate[]> {
  const response = await apiClient.get(
    '/courseware-templates/with-user',
  )

  return extractData(response)
}

export async function saveAsMyTemplate(
  coursewareId: string,
  data: {
    name: string
    description?: string
    style_category?: string
  },
): Promise<{
  id: string
  name: string
  message: string
}> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/save-as-template`,
    data,
  )

  return extractData(response)
}

export async function deleteMyTemplate(
  templateId: string,
): Promise<void> {
  await apiClient.delete(
    `/courseware-templates/personal/${templateId}`,
  )
}

export async function extractTemplateFromHTML(
  samplePages: string[],
  sourceType = 'paste',
): Promise<void> {
  await apiClient.post(
    '/coursewares/templates/extract',
    {
      sample_pages: samplePages,
      source_type: sourceType,
    },
  )
}

export function subscribeExtractSSE(
  callbacks: ExtractSSECallbacks,
): {
  close: () => void
} {
  const token =
    localStorage.getItem('token') || ''

  const url =
    `${window.location.origin}/api/v1/sse/template-extract`
    + `?token=${encodeURIComponent(token)}`

  const eventSource =
    new EventSource(url)

  eventSource.addEventListener(
    'extract_start',
    event => {
      try {
        callbacks.onStart?.(
          JSON.parse(
            (event as MessageEvent).data,
          ),
        )
      } catch {
        // 单条SSE消息解析失败不终止连接。
      }
    },
  )

  eventSource.addEventListener(
    'extract_progress',
    event => {
      try {
        callbacks.onProgress?.(
          JSON.parse(
            (event as MessageEvent).data,
          ),
        )
      } catch {
        // 单条SSE消息解析失败不终止连接。
      }
    },
  )

  eventSource.addEventListener(
    'extract_done',
    event => {
      try {
        callbacks.onDone?.(
          JSON.parse(
            (event as MessageEvent).data,
          ),
        )
      } catch {
        // 完成消息解析失败仍关闭连接。
      }

      eventSource.close()
    },
  )

  eventSource.addEventListener(
    'extract_error',
    event => {
      try {
        callbacks.onError?.(
          JSON.parse(
            (event as MessageEvent).data,
          ),
        )
      } catch {
        // 错误消息解析失败仍关闭连接。
      }

      eventSource.close()
    },
  )

  eventSource.onerror = () => {
    eventSource.close()
  }

  return {
    close: () =>
      eventSource.close(),
  }
}

export async function listMyDrafts(): Promise<CoursewareTemplate[]> {
  const response = await apiClient.get(
    '/coursewares/templates/my-drafts',
  )

  return extractData(response)
}

export async function deleteDraft(
  templateId: string,
): Promise<void> {
  await apiClient.delete(
    `/coursewares/templates/drafts/${templateId}`,
  )
}

export async function refineTemplate(
  templateId: string,
  instruction: string,
): Promise<void> {
  await apiClient.post(
    `/coursewares/templates/${templateId}/refine`,
    {
      instruction,
    },
  )
}

export function subscribeTemplateRefineSSE(
  templateId: string,
  callbacks: RefineSSECallbacks,
): {
  close: () => void
} {
  const token =
    localStorage.getItem('token') || ''

  const url =
    `${window.location.origin}/api/v1/sse/template-refine/${templateId}`
    + `?token=${encodeURIComponent(token)}`

  const eventSource =
    new EventSource(url)

  eventSource.addEventListener(
    'refine_start',
    event => {
      try {
        callbacks.onStart?.(
          JSON.parse(
            (event as MessageEvent).data,
          ),
        )
      } catch {
        // 单条SSE消息解析失败不终止连接。
      }
    },
  )

  eventSource.addEventListener(
    'refine_chunk',
    event => {
      try {
        callbacks.onChunk?.(
          JSON.parse(
            (event as MessageEvent).data,
          ),
        )
      } catch {
        // 单条SSE消息解析失败不终止连接。
      }
    },
  )

  eventSource.addEventListener(
    'refine_progress',
    event => {
      try {
        callbacks.onProgress?.(
          JSON.parse(
            (event as MessageEvent).data,
          ),
        )
      } catch {
        // 单条SSE消息解析失败不终止连接。
      }
    },
  )

  eventSource.addEventListener(
    'refine_done',
    event => {
      try {
        callbacks.onDone?.(
          JSON.parse(
            (event as MessageEvent).data,
          ),
        )
      } catch {
        // 完成消息解析失败仍关闭连接。
      }

      eventSource.close()
    },
  )

  eventSource.addEventListener(
    'refine_error',
    event => {
      try {
        callbacks.onError?.(
          JSON.parse(
            (event as MessageEvent).data,
          ),
        )
      } catch {
        // 错误消息解析失败仍关闭连接。
      }

      eventSource.close()
    },
  )

  eventSource.onerror = () => {
    eventSource.close()
  }

  return {
    close: () =>
      eventSource.close(),
  }
}

export async function getTemplateHistory(
  templateId: string,
): Promise<{
  template_id: string
  history: RefineHistoryEntry[]
  total: number
}> {
  const response = await apiClient.get(
    `/coursewares/templates/${templateId}/history`,
  )

  return extractData(response)
}

export async function rollbackTemplate(
  templateId: string,
  historyIndex: number,
): Promise<{
  template_id: string
  color_scheme: string
  css_variables: string
  sample_pages: string
  style_category: string
  message: string
}> {
  const response = await apiClient.post(
    `/coursewares/templates/${templateId}/rollback`,
    {
      history_index: historyIndex,
    },
  )

  return extractData(response)
}

export async function publishDraft(
  templateId: string,
  data: {
    name: string
    description?: string
    style_category?: string
    scope: string
    scope_target_id?: string
  },
): Promise<{
  template_id: string
  name: string
  scope: string
  message: string
}> {
  const response = await apiClient.post(
    `/coursewares/templates/${templateId}/publish`,
    data,
  )

  return extractData(response)
}

export async function unpublishTemplate(
  templateId: string,
): Promise<{
  template_id: string
  message: string
}> {
  const response = await apiClient.post(
    `/coursewares/templates/${templateId}/unpublish`,
  )

  return extractData(response)
}

export async function getPublishTargets(): Promise<PublishTargetsResponse> {
  const response = await apiClient.get(
    '/coursewares/templates/publish-targets',
  )

  return extractData(response)
}

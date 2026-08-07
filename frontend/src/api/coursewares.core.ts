/**
 * coursewares.core.ts
 *
 * 课件工坊基础流程API。
 *
 * 本文件只负责：
 *   1. 课件CRUD；
 *   2. 页面索引操作；
 *   3. 课件状态流转；
 *   4. 索引和页面生成启动。
 *
 * 其他职责已经按业务拆分：
 *   - 工坊SSE：coursewares.sse.ts；
 *   - 模板、组件库和方案预设：coursewares.catalog.ts；
 *   - 多入口创建和教案对齐：coursewares.creation.ts；
 *   - 发布、共享和自动装配启动：coursewares.sharing.ts；
 *   - 单页修改和页面历史：coursewares.page-mutation.ts。
 *
 * 文件末尾继续转出全部迁移模块，兼容历史代码直接从
 * coursewares.core导入已经迁出的函数和类型。
 */

import apiClient from './client'
import { extractData } from './coursewares.types'
import type {
  CoursewareListResponse,
  CoursewareDetail,
  CoursewarePage,
} from './coursewares.types'

// ==================== 课件CRUD ====================

export async function getCoursewares(params?: {
  status?: string
  subject?: string
  limit?: number
  offset?: number
}): Promise<CoursewareListResponse> {
  const response = await apiClient.get(
    '/coursewares',
    {
      params,
    },
  )

  return extractData(response)
}

export async function createCourseware(data: {
  lesson_plan_id: string
  title?: string
}): Promise<{ id: string }> {
  const response = await apiClient.post(
    '/coursewares',
    data,
  )

  return extractData(response)
}

export async function getCourseware(
  id: string,
): Promise<CoursewareDetail> {
  const response = await apiClient.get(
    `/coursewares/${id}`,
  )

  return extractData(response)
}

export async function updateCourseware(
  id: string,
  data: {
    title: string
  },
): Promise<void> {
  await apiClient.put(
    `/coursewares/${id}`,
    data,
  )
}

export async function deleteCourseware(
  id: string,
): Promise<void> {
  await apiClient.delete(
    `/coursewares/${id}`,
  )
}

// ==================== 课件页面索引操作 ====================

export async function getCoursewarePages(
  coursewareId: string,
): Promise<CoursewarePage[]> {
  const response = await apiClient.get(
    `/coursewares/${coursewareId}/pages`,
  )

  return extractData(response)
}

export async function updateCWPageIndex(
  coursewareId: string,
  pageNumber: number,
  data: {
    title?: string
    purpose?: string
    content_summary?: string
    interaction_type?: string
    visual_format?: string
    media_requirements?: string
    estimated_complexity?: number
  },
): Promise<void> {
  await apiClient.put(
    `/coursewares/${coursewareId}/pages/${pageNumber}`,
    data,
  )
}

export async function addCWPage(
  coursewareId: string,
  data: {
    title: string
    purpose?: string
    content_summary?: string
    interaction_type?: string
    visual_format?: string
    media_requirements?: string
    estimated_complexity?: number
  },
): Promise<CoursewarePage> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/pages`,
    data,
  )

  return extractData(response)
}

export async function deleteCWPage(
  coursewareId: string,
  pageNumber: number,
): Promise<void> {
  await apiClient.delete(
    `/coursewares/${coursewareId}/pages/${pageNumber}`,
  )
}

export async function reorderCWPages(
  coursewareId: string,
  pageIds: string[],
): Promise<void> {
  await apiClient.put(
    `/coursewares/${coursewareId}/pages/reorder`,
    {
      page_ids: pageIds,
    },
  )
}

// ==================== 课件状态流转 ====================

export async function confirmCWIndex(
  coursewareId: string,
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/confirm-index`,
  )
}

/**
 * 旧兼容入口：直接保存风格JSON字符串。
 */
export async function saveCWStyle(
  coursewareId: string,
  styleConfig: string,
): Promise<void> {
  await apiClient.put(
    `/coursewares/${coursewareId}/style`,
    {
      style_config: styleConfig,
    },
  )
}

/**
 * 保存完整风格配置。
 */
export async function saveStyleFull(
  coursewareId: string,
  data: {
    template_id: string
    logo_url?: string
    org_name?: string
    custom_primary_color?: string
  },
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/save-style`,
    data,
  )
}

export async function confirmCWStyle(
  coursewareId: string,
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/confirm-style`,
  )
}

export async function uploadCWLogo(
  coursewareId: string,
  file: File,
): Promise<{ url: string }> {
  const formData = new FormData()
  formData.append(
    'file',
    file,
  )

  const response = await apiClient.post(
    `/coursewares/${coursewareId}/upload-logo`,
    formData,
    {
      headers: {
        'Content-Type':
          'multipart/form-data',
      },
    },
  )

  return extractData(response)
}

export async function getLogoHistory(
  limit = 20,
): Promise<string[]> {
  const response = await apiClient.get(
    '/coursewares/logo-history',
    {
      params: {
        limit,
      },
    },
  )

  const data = extractData<{
    logos: string[]
  }>(response)

  return data.logos || []
}

export async function deleteLogoHistory(
  logoURL: string,
): Promise<{
  affected: number
}> {
  const response = await apiClient.delete(
    '/coursewares/logo-history',
    {
      params: {
        url: logoURL,
      },
    },
  )

  return extractData(response)
}

export async function confirmCourseware(
  coursewareId: string,
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/confirm`,
  )
}

export async function generateCWPreview(
  coursewareId: string,
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/generate-preview`,
  )
}

export async function saveCWNavTemplate(
  coursewareId: string,
  navTemplateHTML: string,
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/save-nav-template`,
    {
      nav_template_html:
        navTemplateHTML,
    },
  )
}

export async function generateCWPages(
  coursewareId: string,
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/generate-pages`,
  )
}

export async function refineNav(
  coursewareId: string,
  instruction: string,
): Promise<{
  nav_html: string
  message: string
}> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/refine-nav`,
    {
      instruction,
    },
  )

  return extractData(response)
}

export async function cancelGenerate(
  coursewareId: string,
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/cancel-generate`,
  )
}

// ==================== 课件索引生成 ====================

export async function generateCWIndex(
  coursewareId: string,
  preset?: string,
  customPromptHint?: string,
): Promise<void> {
  const body:
    Record<string, string> = {}

  if (preset) {
    body.preset = preset
  }

  if (customPromptHint) {
    body.custom_prompt_hint =
      customPromptHint
  }

  await apiClient.post(
    `/coursewares/${coursewareId}/generate-index`,
    body,
  )
}

/**
 * 保持coursewares.core历史导入路径兼容。
 */
export * from './coursewares.sse'
export * from './coursewares.page-mutation'
export * from './coursewares.catalog'
export * from './coursewares.creation'
export * from './coursewares.sharing'

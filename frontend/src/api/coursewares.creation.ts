/**
 * coursewares.creation.ts
 *
 * 课件多入口创建、课程知识库和课件教案对齐API。
 */

import apiClient from './client'
import { extractData } from './coursewares.types'
import type {
  CurriculumKPResponse,
  AlignmentReportResponse,
} from './coursewares.types'

// ==================== 多入口创建 ====================

export async function createCoursewareFromTopic(data: {
  subject: string
  grade: string
  topic: string
  page_range?: string
  extra_notes?: string
  kp_codes?: string[]
}): Promise<{ id: string }> {
  const response = await apiClient.post(
    '/coursewares/from-topic',
    data,
  )

  return extractData(response)
}

export async function generateCWIndexFromTopic(
  coursewareId: string,
  data: {
    subject: string
    grade: string
    topic: string
    page_range?: string
    extra_notes?: string
    preset?: string
    custom_prompt_hint?: string
  },
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/generate-index-topic`,
    data,
  )
}

export async function createCoursewareFromPPT(
  file: File,
  subject: string,
  grade: string,
  title?: string,
): Promise<{
  id: string
  title: string
  subject: string
  grade: string
  source_type: string
  slide_count: number
  message: string
}> {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('subject', subject)
  formData.append('grade', grade)

  if (title) {
    formData.append('title', title)
  }

  const response = await apiClient.post(
    '/coursewares/from-ppt',
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      timeout: 300000,
    },
  )

  return extractData(response)
}

export async function generateCWIndexFromPPT(
  coursewareId: string,
  preset?: string,
  customPromptHint?: string,
): Promise<void> {
  const body: Record<string, string> = {}

  if (preset) {
    body.preset = preset
  }

  if (customPromptHint) {
    body.custom_prompt_hint = customPromptHint
  }

  await apiClient.post(
    `/coursewares/${coursewareId}/generate-index-ppt`,
    body,
  )
}

export async function createCoursewareFromDoc(
  file: File,
  subject: string,
  grade: string,
  title?: string,
): Promise<{
  id: string
  title: string
  subject: string
  grade: string
  source_type: string
  word_count: number
  message: string
}> {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('subject', subject)
  formData.append('grade', grade)

  if (title) {
    formData.append('title', title)
  }

  const response = await apiClient.post(
    '/coursewares/from-doc',
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      timeout: 300000,
    },
  )

  return extractData(response)
}

export async function generateCWIndexFromDoc(
  coursewareId: string,
  preset?: string,
  customPromptHint?: string,
): Promise<void> {
  const body: Record<string, string> = {}

  if (preset) {
    body.preset = preset
  }

  if (customPromptHint) {
    body.custom_prompt_hint = customPromptHint
  }

  await apiClient.post(
    `/coursewares/${coursewareId}/generate-index-doc`,
    body,
  )
}

export async function generate3DPage(
  coursewareId: string,
): Promise<{
  message: string
  courseware_id: string
}> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/generate-3d-page`,
    {},
    {
      timeout: 180000,
    },
  )

  return extractData(response)
}

export async function createCoursewareFrom3D(data: {
  subject: string
  grade: string
  topic: string
  description: string
}): Promise<{
  id: string
  title: string
  source_type: string
  status: string
  message: string
}> {
  const response = await apiClient.post(
    '/coursewares/from-3d',
    data,
  )

  return extractData(response)
}

// ==================== 课程知识库 ====================

export async function getCurriculumKnowledgePoints(
  subject: string,
  grade?: number,
): Promise<CurriculumKPResponse> {
  const params: Record<string, string | number> = {
    subject,
  }

  if (grade && grade > 0) {
    params.grade = grade
  }

  const response = await apiClient.get(
    '/curriculum/knowledge-points',
    {
      params,
    },
  )

  return extractData(response)
}

// ==================== 课件与教案对齐 ====================

export async function getAlignmentReport(
  coursewareId: string,
): Promise<AlignmentReportResponse> {
  const response = await apiClient.get(
    `/coursewares/${coursewareId}/alignment-report`,
  )

  return extractData(response)
}

export async function recheckAlignment(
  coursewareId: string,
): Promise<void> {
  await apiClient.post(
    `/coursewares/${coursewareId}/recheck-alignment`,
  )
}

export interface CoursewareLessonPlanContent {
  has_lesson_plan: boolean
  title: string
  content: string
}

/**
 * 读取课件关联的来源教案正文。
 *
 * signal由抽屉生命周期传入：
 *   - 抽屉关闭时取消未完成请求；
 *   - 切换课件时取消旧课件请求；
 *   - 取消请求不会被当成业务错误展示。
 */
export async function getCoursewareLessonPlanContent(
  coursewareId: string,
  signal?: AbortSignal,
): Promise<CoursewareLessonPlanContent> {
  const response = await apiClient.get(
    `/coursewares/${coursewareId}/lesson-plan-content`,
    {
      signal,
    },
  )

  return extractData(response)
}

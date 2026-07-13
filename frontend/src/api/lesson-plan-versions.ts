/**
 * lesson-plan-versions.ts — 教案正文版本历史前端API
 *
 * 后端接口：
 *   GET  /lesson-plans/plans/{id}/versions
 *   GET  /lesson-plans/plans/{id}/versions/{versionId}
 *   POST /lesson-plans/plans/{id}/versions/{versionId}/restore
 */

import apiClient from './client'

// ==================== 类型定义 ====================

export type LessonPlanVersionSource =
  | 'manual'
  | 'ai'
  | 'import'
  | 'restore'
  | 'system'

export interface LessonPlanContentVersionListItem {
  id: string
  version_number: number
  title: string
  content_preview: string
  character_count: number
  duration_minutes: number
  change_source: LessonPlanVersionSource
  changed_by: string | null
  changed_by_name: string
  change_summary: string
  created_at: string
}

export interface LessonPlanContentVersion {
  id: string
  lesson_plan_id: string
  version_number: number
  title: string
  content_markdown: string
  content_structured: string
  duration_minutes: number
  change_source: LessonPlanVersionSource
  changed_by: string | null
  changed_by_name: string
  change_summary: string
  created_at: string
}

export interface LessonPlanContentVersionListResponse {
  versions: LessonPlanContentVersionListItem[]
  total: number
  current_version: number
}

export interface LessonPlanContentRestoreResponse {
  restored_from_version: number
  current_version: number
  title: string
  content_markdown: string
  duration_minutes: number
}

// ==================== API函数 ====================

/** 获取一份教案的历史版本列表 */
export async function getLessonPlanContentVersions(
  planID: string,
  params?: {
    limit?: number
    offset?: number
  },
): Promise<LessonPlanContentVersionListResponse> {
  const query = new URLSearchParams()

  if (params?.limit) {
    query.set('limit', String(params.limit))
  }
  if (params?.offset) {
    query.set('offset', String(params.offset))
  }

  const suffix = query.toString()
  const response = await apiClient.get(
    `/lesson-plans/plans/${planID}/versions${suffix ? `?${suffix}` : ''}`,
  )

  return response.data.data as LessonPlanContentVersionListResponse
}

/** 获取单个历史版本的完整正文 */
export async function getLessonPlanContentVersion(
  planID: string,
  versionID: string,
): Promise<LessonPlanContentVersion> {
  const response = await apiClient.get(
    `/lesson-plans/plans/${planID}/versions/${versionID}`,
  )

  return response.data.data as LessonPlanContentVersion
}

/** 恢复指定历史版本，恢复前后端会自动保存当前正文 */
export async function restoreLessonPlanContentVersion(
  planID: string,
  versionID: string,
): Promise<LessonPlanContentRestoreResponse> {
  const response = await apiClient.post(
    `/lesson-plans/plans/${planID}/versions/${versionID}/restore`,
    {},
  )

  return response.data.data as LessonPlanContentRestoreResponse
}

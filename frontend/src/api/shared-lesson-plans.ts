/**
 * shared-lesson-plans.ts — 共享教案市场专用API
 *
 * 前端只提交展示筛选条件，不提交education_domain或作者白名单。
 * 最终教育域和组织授权始终由后端根据当前登录用户实时判断。
 */

import apiClient from './client'
import type { ApiResponse } from './client'
import type {
  LessonPlan,
  LessonPlanListResponse,
} from './lesson-plans'

export interface SharedLessonPlanListParams {
  group_id?: string
  status: 'approved' | 'published_shared'
  subject?: string
  grade?: string
  quality_level?: number
  structure_type?: number
  limit?: number
  offset?: number
}

export async function getSharedLessonPlans(
  params: SharedLessonPlanListParams,
): Promise<LessonPlanListResponse> {
  const query = new URLSearchParams()

  if (params.group_id) {
    query.set('group_id', params.group_id)
  }
  query.set('status', params.status)

  if (params.subject) {
    query.set('subject', params.subject)
  }
  if (params.grade) {
    query.set('grade', params.grade)
  }
  if (params.quality_level) {
    query.set(
      'quality_level',
      String(params.quality_level),
    )
  }
  if (params.structure_type) {
    query.set(
      'structure_type',
      String(params.structure_type),
    )
  }
  if (params.limit) {
    query.set('limit', String(params.limit))
  }
  if (params.offset) {
    query.set('offset', String(params.offset))
  }

  const response = await apiClient.get<
    ApiResponse<LessonPlanListResponse>
  >(
    `/lesson-plans/plans?${query.toString()}`,
  )
  return response.data.data!
}

export async function forkSharedLessonPlan(
  lessonPlanId: string,
): Promise<LessonPlan> {
  const response = await apiClient.post<
    ApiResponse<LessonPlan>
  >(
    `/lesson-plans/plans/${lessonPlanId}/fork`,
  )
  return response.data.data!
}

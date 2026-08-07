/**
 * courseware-add-page-discussion.ts — 新增课件页AI讨论与指定位置建页 API。
 *
 * 讨论接口只返回老师可见回复和结构化页面方案，不创建页面、不生成HTML。
 * 正式建页必须由界面独立按钮调用 addCWPageAtPosition，随后再调用既有
 * regenerateCWPage 完成页面HTML生成。
 */
import apiClient from './client'
import {
  extractData,
} from './coursewares.types'

import type {
  CoursewarePage,
} from './coursewares.types'

export interface CoursewareAddPageDiscussionMessage {
  role: 'teacher' | 'assistant'
  content: string
}

export interface CoursewareAddPagePlan {
  title: string
  purpose: string
  content_summary: string
  interaction_type: string
  visual_format: string
  media_requirements: string
  estimated_complexity: number
}

export interface CoursewareAddPageDiscussionResult {
  reply: string
  summary: string
  ready_for_confirmation: boolean
  insert_at: number
  plan: CoursewareAddPagePlan
}

interface CoursewareAddPageDiscussionEnvelope {
  discussion: CoursewareAddPageDiscussionResult
  message?: string
}

export interface AddCWPageAtPositionInput {
  insert_at: number
  title: string
  purpose?: string
  content_summary?: string
  interaction_type?: string
  visual_format?: string
  media_requirements?: string
  estimated_complexity?: number
}

/**
 * 与AI讨论新增页需求。
 *
 * messages只包含历史消息，本轮消息通过message单独提交。
 * 服务端会重新读取课件和真实页面顺序，并校验insert_at。
 */
export async function discussCoursewareAddPage(
  coursewareId: string,
  data: {
    message: string
    messages: CoursewareAddPageDiscussionMessage[]
    insert_at: number
    current_plan: CoursewareAddPagePlan
  },
): Promise<CoursewareAddPageDiscussionResult> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/pages/new/rebuild-discussion`,
    data,
    {
      timeout: 300000,
    },
  )

  return extractData<CoursewareAddPageDiscussionEnvelope>(
    response,
  ).discussion
}

/**
 * 在指定位置创建课件页面。
 *
 * 后端以事务方式移动原位置及之后的页面，并返回页面最终真实页码。
 */
export async function addCWPageAtPosition(
  coursewareId: string,
  data: AddCWPageAtPositionInput,
): Promise<CoursewarePage> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/pages`,
    data,
  )

  return extractData<CoursewarePage>(
    response,
  )
}

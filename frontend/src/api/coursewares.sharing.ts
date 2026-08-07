/**
 * coursewares.sharing.ts
 *
 * 课件发布、共享、复制和全自动装配启动API。
 */

import apiClient from './client'
import { extractData } from './coursewares.types'
import type {
  SharedCoursewareListResponse,
} from './coursewares.types'

// ==================== 发布、共享与复制 ====================

export async function publishCourseware(
  coursewareId: string,
  target:
    | 'published_personal'
    | 'published_shared'
    | 'private',
): Promise<{ message: string }> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/publish`,
    {
      target,
    },
  )

  return extractData(response)
}

export async function setCodeShareScope(
  coursewareId: string,
  scope:
    | 'none'
    | 'group'
    | 'school'
    | 'region'
    | 'public',
): Promise<{ message: string }> {
  const response = await apiClient.put(
    `/coursewares/${coursewareId}/code-share-scope`,
    {
      code_share_scope: scope,
    },
  )

  return extractData(response)
}

export async function listSharedCoursewares(params?: {
  subject?: string
  limit?: number
  offset?: number
}): Promise<SharedCoursewareListResponse> {
  const response = await apiClient.get(
    '/coursewares/shared',
    {
      params,
    },
  )

  return extractData(response)
}

export async function forkCourseware(
  coursewareId: string,
): Promise<{
  id: string
  title: string
  message: string
}> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/fork`,
    {},
  )

  return extractData(response)
}

// ==================== 全自动一键装配 ====================

export async function autoAssemble(
  coursewareId: string,
  skipVideo = false,
): Promise<{
  message: string
  courseware_id: string
  skip_video: boolean
}> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/auto-assemble`,
    {
      skip_video: skipVideo,
    },
  )

  return extractData(response)
}

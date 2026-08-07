/**
 * 课件全页重构讨论 API。
 *
 * 讨论阶段只保存并返回老师可见的正式交流内容，不生成HTML、不修改页面。
 * 只有调用 confirmRebuildDiscussion 时，后端才进入既有全页重构链路。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'

export type CWRebuildDiscussionStatus =
  | 'discussing'
  | 'awaiting_confirmation'
  | 'executing'
  | 'completed'
  | 'cancelled'
  | 'stale'

export interface CWRebuildDiscussionMessage {
  role: 'teacher' | 'assistant'
  content: string
  created_at: string
}

export interface CWRebuildDiscussion {
  id: string
  status: CWRebuildDiscussionStatus
  page_number: number
  messages: CWRebuildDiscussionMessage[]
  ai_summary: string
  final_instruction: string
  error_message: string
  ready_for_confirmation: boolean
  executing: boolean
  created_at: string
  updated_at: string
}

export interface CWRebuildDiscussionConfirmResult {
  discussion: CWRebuildDiscussion
  page_number: number
  html_content: string
  message: string
}

export interface CWRebuildDiscussionEnvelope {
  discussion: CWRebuildDiscussion | null
  message?: string
}

function endpoint(
  coursewareId: string,
  pageNumber: number,
): string {
  return `/coursewares/${coursewareId}/pages/${pageNumber}/rebuild-discussion`
}

/**
 * 恢复当前老师在当前页面上的活动讨论。
 *
 * 没有活动讨论时返回 null。
 * 本操作不调用AI，也不会修改页面。
 */
export async function loadRebuildDiscussion(
  coursewareId: string,
  pageNumber: number,
): Promise<CWRebuildDiscussion | null> {
  const response = await apiClient.post(
    endpoint(coursewareId, pageNumber),
    {
      action: 'load',
    },
  )

  return extractData<CWRebuildDiscussionEnvelope>(
    response,
  ).discussion
}

/**
 * 向AI发送一轮讨论消息。
 *
 * 本接口只讨论需求并形成方案，不生成HTML、不写回课件页面。
 */
export async function sendRebuildDiscussionMessage(
  coursewareId: string,
  pageNumber: number,
  data: {
    discussionId?: string
    content: string
    referenceContext?: string
    image?: string
  },
): Promise<CWRebuildDiscussionEnvelope> {
  const response = await apiClient.post(
    endpoint(coursewareId, pageNumber),
    {
      action: 'message',
      discussion_id:
        data.discussionId || '',
      content:
        data.content,
      reference_context:
        data.referenceContext || '',
      image:
        data.image || '',
    },
    {
      timeout: 300000,
    },
  )

  return extractData<CWRebuildDiscussionEnvelope>(
    response,
  )
}

/**
 * 老师通过独立按钮明确确认最终执行方案。
 *
 * 只有本接口会触发既有全页重构、版本快照、HTML校验和页面写回链路。
 */
export async function confirmRebuildDiscussion(
  coursewareId: string,
  pageNumber: number,
  discussionId: string,
): Promise<CWRebuildDiscussionConfirmResult> {
  const response = await apiClient.post(
    endpoint(coursewareId, pageNumber),
    {
      action: 'confirm',
      discussion_id:
        discussionId,
    },
    {
      timeout: 300000,
    },
  )

  return extractData<CWRebuildDiscussionConfirmResult>(
    response,
  )
}

/**
 * 取消尚未执行的讨论。
 *
 * 取消不会修改课件页面。
 */
export async function cancelRebuildDiscussion(
  coursewareId: string,
  pageNumber: number,
  discussionId: string,
): Promise<CWRebuildDiscussionEnvelope> {
  const response = await apiClient.post(
    endpoint(coursewareId, pageNumber),
    {
      action: 'cancel',
      discussion_id:
        discussionId,
    },
  )

  return extractData<CWRebuildDiscussionEnvelope>(
    response,
  )
}

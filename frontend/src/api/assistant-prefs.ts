/**
 * assistant-prefs.ts — 对话式备课助手偏好API
 *
 * teacher_assistant_prefs数据库仍按“老师×学科”保存。
 * grade和stage作为每次请求的运行时适用性条件：
 *   - 同一条存量偏好只在匹配的具体年级和阶段生效；
 *   - 不适用时后端返回has_record=false，但不删除原偏好；
 *   - 高三不会使用高中、高一、高二或空年级助手。
 */
import apiClient from './client'

// ==================== 类型定义 ====================

/** 助手偏好三态。 */
export interface AssistantPref {
  subject: string
  /** 本次运行时校验所使用的具体年级。 */
  grade: string
  /** false=本次没有有效偏好，继续走同年级自动匹配。 */
  has_record: boolean
  /** true=老师明确选择系统默认纯骨架。 */
  is_system_default: boolean
  /** 当前有效的助手ID。 */
  assistant_id: string
  /** 当前有效助手的显示名。 */
  assistant_name?: string
}

/** assistant-options候选项。 */
export interface AssistantOption {
  id: string
  name: string
  description: string
  source: string
  source_label: string
  /** 必须与当前课程学科精确一致。 */
  subject: string
  /** 具体年级，如高三、十二年级或12。 */
  grade_range: string
  is_default_here: boolean
}

/** assistant-options响应。 */
export interface AssistantOptionsResponse {
  assistants: AssistantOption[]
  total: number
}

// ==================== API函数 ====================

/**
 * 读取老师在当前学科、具体年级和阶段下的有效助手偏好。
 */
export async function getAssistantPref(
  subject: string,
  grade: string,
  stage?: string,
): Promise<AssistantPref> {
  const params: Record<string, string> = {
    subject,
    grade,
  }
  if (stage) params.stage = stage

  const res = await apiClient.get(
    '/lesson-plans/assistant-prefs',
    { params },
  )
  return res.data?.data ?? res.data
}

/**
 * 写入助手偏好。
 *
 * assistantId为空串表示老师明确选择系统默认纯骨架。
 * 非空助手必须通过当前学科、具体年级和阶段的后端复核。
 */
export async function putAssistantPref(
  subject: string,
  grade: string,
  stage: string | undefined,
  assistantId: string,
): Promise<AssistantPref> {
  const res = await apiClient.put(
    '/lesson-plans/assistant-prefs',
    {
      subject,
      grade,
      stage: stage || '',
      assistant_id: assistantId,
    },
  )
  return res.data?.data ?? res.data
}

/**
 * 列出当前学科、具体年级和阶段可用的助手。
 */
export async function getAssistantOptions(
  subject: string,
  grade: string,
  stage?: string,
): Promise<AssistantOptionsResponse> {
  const params: Record<string, string> = {
    subject,
    grade,
  }
  if (stage) params.stage = stage

  const res = await apiClient.get(
    '/lesson-plans/assistant-options',
    { params },
  )
  return res.data?.data ?? res.data
}

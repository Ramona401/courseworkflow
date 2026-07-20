/**
 * assistant-prefs.ts — 对话式备课助手偏好API
 *
 * teacher_assistant_prefs数据库仍按“老师×学科”保存。
 * grade用于候选相关性排序，stage用于运行时场景校验：
 *   - 老师手动偏好不再受具体年级阻断；
 *   - 同学科、当前阶段下，可以选择其它年级、学段或不限年级助手；
 *   - 平台自动匹配仍严格要求同一具体年级；
 *   - 偏好不适用于当前学科或阶段时返回has_record=false，但不删除原记录。
 */
import apiClient from './client'

// ==================== 类型定义 ====================

/** 助手偏好三态。 */
export interface AssistantPref {
  subject: string
  /** 本次运行时校验所使用的具体年级。 */
  grade: string
  /** false=本次没有有效偏好，继续走具体年级严格自动匹配。 */
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
  /** 与当前课程学科精确一致。 */
  subject: string
  /** 可为具体年级、小学/初中/高中学段，或空字符串表示不限年级。 */
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
 * 非空助手必须通过active、可见性、当前学科和阶段复核；具体年级不阻断手动偏好。
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
 * 列出当前学科和阶段可手动使用的助手，具体年级仅用于相关性排序。
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

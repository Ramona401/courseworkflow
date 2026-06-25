/**
 * assistant-prefs.ts — 对话式备课·助手轻量选择入口 API 封装（Phase 1）
 *
 * 对应后端三接口（均已联调验证）：
 *   GET  /api/v1/lesson-plans/assistant-prefs?subject=语文            读当前老师在某学科的助手偏好(三态)
 *   PUT  /api/v1/lesson-plans/assistant-prefs                        写偏好(body:{subject,assistant_id})
 *   GET  /api/v1/lesson-plans/assistant-options?subject=语文&stage=design  列出该学科(该阶段)可选助手
 *
 * 三态语义（与后端 teacher_assistant_prefs 表一致）：
 *   - has_record=false                      → 老师从没选过（系统走学科推荐兜底）
 *   - has_record=true & is_system_default   → 老师显式选了「系统默认」(纯骨架，不挂助手)
 *   - has_record=true & assistant_id 非空    → 老师为该学科选定了某助手
 *
 * 空串 assistant_id 是合法写入值（= 显式系统默认），与「删除记录(回到从没选过)」语义不同。
 */
import apiClient from './client'

// ==================== 类型定义 ====================

/** 助手偏好（三态）——GET assistant-prefs / PUT 回显 共用 */
export interface AssistantPref {
  subject: string
  /** false=从没选过（走学科推荐兜底） */
  has_record: boolean
  /** true=显式选了「系统默认」(纯骨架)；仅 has_record=true 时有意义 */
  is_system_default: boolean
  /** 选定的助手ID；has_record=false 或 is_system_default=true 时为空串 */
  assistant_id: string
  /** 选定助手的显示名（仅 assistant_id 非空时由后端回填；助手已删则为空，前端 fallback 文案兜底） */
  assistant_name?: string
}

/**
 * 可选助手条目（assistant-options 列表项）。
 * 字段对齐后端 ListAssistants 返回结构，此处只声明面板要用到的部分。
 */
export interface AssistantOption {
  id: string
  name: string
  /** 一句话特长（来自 ai_assistants.description），面板二级文字 */
  description: string
  /** 来源：system / group / personal */
  source: string
  /** 来源中文标签：系统 / 学校 / 个人 */
  source_label: string
  /** 学科（空串=通用兜底助手） */
  subject: string
  /** 学段（如 高中/初中/小学；空串=通用） */
  grade_range: string
  /** 是否在当前场景被标记为默认推荐（仅作展示提示，非选中态） */
  is_default_here: boolean
}

/** assistant-options 响应 */
export interface AssistantOptionsResponse {
  assistants: AssistantOption[]
  total: number
}

// ==================== API 函数 ====================

/**
 * 读取当前老师在某学科的助手偏好（三态）。
 * GET /api/v1/lesson-plans/assistant-prefs?subject=
 */
export async function getAssistantPref(subject: string): Promise<AssistantPref> {
  const res = await apiClient.get('/lesson-plans/assistant-prefs', {
    params: { subject },
  })
  return res.data?.data ?? res.data
}

/**
 * 写入/更新当前老师在某学科的助手偏好。
 * PUT /api/v1/lesson-plans/assistant-prefs
 *
 * @param assistantId 选定的助手ID；传空串 '' 表示显式选择「系统默认」(纯骨架)。
 * 返回写入后的三态结果（前端据此即时更新面板高亮）。
 */
export async function putAssistantPref(
  subject: string,
  assistantId: string
): Promise<AssistantPref> {
  const res = await apiClient.put('/lesson-plans/assistant-prefs', {
    subject,
    assistant_id: assistantId,
  })
  return res.data?.data ?? res.data
}

/**
 * 列出某学科（可选某阶段）的可选助手。
 * GET /api/v1/lesson-plans/assistant-options?subject=&stage=
 *
 * @param stage 可选；传了按对应助手场景过滤(与后端默认助手解析同口径)，
 *              不传则列出该学科所有可见助手。
 */
export async function getAssistantOptions(
  subject: string,
  stage?: string
): Promise<AssistantOptionsResponse> {
  const params: Record<string, string> = { subject }
  if (stage) params.stage = stage
  const res = await apiClient.get('/lesson-plans/assistant-options', { params })
  return res.data?.data ?? res.data
}

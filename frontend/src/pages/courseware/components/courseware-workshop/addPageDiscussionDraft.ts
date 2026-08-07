/**
 * addPageDiscussionDraft.ts — 新增页AI讨论草稿的数据清理工具。
 *
 * 这里只处理浏览器草稿中的普通业务字段，不执行接口调用。
 * 所有解析均采用白名单和安全默认值，损坏草稿会回退为空方案。
 */
import type {
  CoursewareAddPageDiscussionMessage,
  CoursewareAddPagePlan,
} from '@/api/courseware-add-page-discussion'

export interface AddPageDiscussionDraft {
  input: string
  messages: CoursewareAddPageDiscussionMessage[]
  summary: string
  readyForConfirmation: boolean
  lastDiscussedInsertAt: number
  plan: CoursewareAddPagePlan
}

export const EMPTY_ADD_PAGE_PLAN: CoursewareAddPagePlan = {
  title: '',
  purpose: '',
  content_summary: '',
  interaction_type: 'static',
  visual_format: 'text_heavy',
  media_requirements: '',
  estimated_complexity: 3,
}

export function createAddPageDiscussionDraft(): AddPageDiscussionDraft {
  return {
    input: '',
    messages: [],
    summary: '',
    readyForConfirmation: false,
    lastDiscussedInsertAt: 0,
    plan: { ...EMPTY_ADD_PAGE_PLAN },
  }
}

export function parseAddPagePlan(value: unknown): CoursewareAddPagePlan {
  const source = value && typeof value === 'object'
    ? value as Partial<CoursewareAddPagePlan>
    : {}

  const complexity = typeof source.estimated_complexity === 'number'
    && Number.isFinite(source.estimated_complexity)
    ? Math.min(5, Math.max(1, Math.round(source.estimated_complexity)))
    : 3

  return {
    title: typeof source.title === 'string' ? source.title : '',
    purpose: typeof source.purpose === 'string' ? source.purpose : '',
    content_summary: typeof source.content_summary === 'string'
      ? source.content_summary
      : '',
    interaction_type: typeof source.interaction_type === 'string'
      && source.interaction_type.trim()
      ? source.interaction_type
      : 'static',
    visual_format: typeof source.visual_format === 'string'
      && source.visual_format.trim()
      ? source.visual_format
      : 'text_heavy',
    media_requirements: typeof source.media_requirements === 'string'
      ? source.media_requirements
      : '',
    estimated_complexity: complexity,
  }
}

export function parseAddPageDiscussionDraft(raw: string): AddPageDiscussionDraft {
  const fallback = createAddPageDiscussionDraft()
  if (!raw.trim()) return fallback

  try {
    const parsed = JSON.parse(raw) as Partial<AddPageDiscussionDraft>
    const messages = Array.isArray(parsed.messages)
      ? parsed.messages
          .filter(item => item
            && (item.role === 'teacher' || item.role === 'assistant')
            && typeof item.content === 'string'
            && item.content.trim() !== '')
          .slice(-24)
          .map(item => ({
            role: item.role,
            content: item.content.trim(),
          }))
      : []

    return {
      input: typeof parsed.input === 'string' ? parsed.input : '',
      messages,
      summary: typeof parsed.summary === 'string' ? parsed.summary : '',
      readyForConfirmation: parsed.readyForConfirmation === true,
      lastDiscussedInsertAt: typeof parsed.lastDiscussedInsertAt === 'number'
        ? Math.max(0, Math.round(parsed.lastDiscussedInsertAt))
        : 0,
      plan: parseAddPagePlan(parsed.plan),
    }
  } catch {
    return fallback
  }
}

export function addPagePlanIsReady(plan: CoursewareAddPagePlan): boolean {
  return Boolean(
    plan.title.trim()
    && plan.purpose.trim()
    && plan.content_summary.trim(),
  )
}

export function addPagePlanField(value: string, fallback = '待讨论'): string {
  return value.trim() || fallback
}

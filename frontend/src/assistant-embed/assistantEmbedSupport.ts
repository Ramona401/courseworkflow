/**
 * 教学智能体公开iframe学生端共享类型和纯辅助函数。
 *
 * 本文件不持有React状态、不发起网络请求，也不访问教师登录信息。
 * 只负责：
 *   - 解析Go安全壳注入的公开data属性；
 *   - 构造短时会话的前端过渡视图；
 *   - 合并欢迎语与服务端正式消息；
 *   - 解析父页面精确Origin和会话有效期；
 *   - 提供学生端统一颜色与按钮样式。
 */

import type { CSSProperties } from 'react'
import type {
  AssistantRuntimeMessage,
  AssistantRuntimeSessionView,
} from '../api/coursewares.assistant.types'

export interface AssistantEmbedBootstrap {
  publicId: string
  title: string
  welcomeMessage: string
  displayMode: 'floating'
  displayPosition: 'bottom_right'
  maximumSessionTurns: number
}

export type AssistantEmbedStartupState = 'loading' | 'ready' | 'blocked' | 'error'

export interface AssistantEmbedNotice {
  kind: 'info' | 'success' | 'error'
  text: string
}

export const ASSISTANT_EMBED_COLORS = {
  primary: '#4F7BE8',
  primarySoft: 'rgba(79,123,232,0.10)',
  text: '#1F2937',
  textSecondary: '#64748B',
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  background: '#F8FAFC',
  white: '#FFFFFF',
  success: '#047857',
  danger: '#B91C1C',
}

/**
 * 从Go动态安全壳的根元素读取公开启动数据。
 *
 * 根元素只包含学生端展示字段，不包含教师、学校、内部部署UUID、
 * 模型、积分账户、提示词、上下文快照或允许来源列表。
 */
export function readAssistantEmbedBootstrap(root: HTMLElement): AssistantEmbedBootstrap | null {
  const publicId = root.dataset.publicId?.trim() || ''
  const title = root.dataset.title?.trim() || ''
  const welcomeMessage = root.dataset.welcomeMessage?.trim() || ''
  const displayMode = root.dataset.displayMode?.trim() || ''
  const displayPosition = root.dataset.displayPosition?.trim() || ''
  const maximumSessionTurns = Number(root.dataset.maximumSessionTurns || '')

  if (
    !publicId ||
    !title ||
    !welcomeMessage ||
    displayMode !== 'floating' ||
    displayPosition !== 'bottom_right' ||
    !Number.isInteger(maximumSessionTurns) ||
    maximumSessionTurns < 1 ||
    maximumSessionTurns > 100
  ) {
    return null
  }

  return {
    publicId,
    title,
    welcomeMessage,
    displayMode,
    displayPosition,
    maximumSessionTurns,
  }
}

/**
 * 从document.referrer解析父页面精确Origin。
 *
 * 父页面主动使用no-referrer时结果为空，此时仅禁用自动高度通信，
 * 不退化成postMessage通配符。
 */
export function assistantEmbedParentOriginFromReferrer(): string {
  const referrer = document.referrer.trim()
  if (!referrer) return ''

  try {
    const parsed = new URL(referrer)
    if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') return ''
    return parsed.origin
  } catch {
    return ''
  }
}

export function assistantEmbedWelcomeMessages(content: string): AssistantRuntimeMessage[] {
  const normalized = content.trim()
  return normalized
    ? [{ role: 'assistant', content: normalized, created_at: null }]
    : []
}

export function assistantEmbedVisibleMessages(
  welcomeMessage: string,
  formalMessages: AssistantRuntimeMessage[],
): AssistantRuntimeMessage[] {
  return [
    ...assistantEmbedWelcomeMessages(welcomeMessage),
    ...(formalMessages || []),
  ]
}

/**
 * 会话创建响应暂不返回部署版本，因此同步正式会话视图前使用0作为未知版本。
 * 学生UI不展示该过渡值；正式读取完成后会被真实版本替换。
 */
export function assistantEmbedFallbackSession(
  sessionId: string,
  maximumTurns: number,
  expiresAt: string | null,
): AssistantRuntimeSessionView {
  return {
    id: sessionId,
    deployment_version: 0,
    session_kind: 'external',
    status: 'active',
    turn_count: 0,
    max_turns: maximumTurns,
    remaining_turns: maximumTurns,
    messages: [],
    expires_at: expiresAt,
    last_active_at: null,
  }
}

export function assistantEmbedFormatExpiry(value: string | null): string {
  if (!value) return '短时会话'

  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return '短时会话'

  return `会话有效至 ${date.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  })}`
}

export function assistantEmbedPrimaryButtonStyle(disabled: boolean): CSSProperties {
  return {
    padding: '8px 15px',
    borderRadius: 8,
    border: 'none',
    background: ASSISTANT_EMBED_COLORS.primary,
    color: ASSISTANT_EMBED_COLORS.white,
    fontSize: 11,
    fontWeight: 700,
    cursor: disabled ? 'default' : 'pointer',
    opacity: disabled ? 0.45 : 1,
  }
}

export function assistantEmbedSecondaryButtonStyle(disabled: boolean): CSSProperties {
  return {
    padding: '8px 13px',
    borderRadius: 8,
    border: `1px solid ${ASSISTANT_EMBED_COLORS.border}`,
    background: ASSISTANT_EMBED_COLORS.white,
    color: ASSISTANT_EMBED_COLORS.textSecondary,
    fontSize: 11,
    fontWeight: 700,
    cursor: disabled ? 'default' : 'pointer',
    opacity: disabled ? 0.45 : 1,
  }
}

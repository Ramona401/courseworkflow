/**
 * conversationActionIntent.ts — 对话模式终态动作意图识别。
 *
 * 只识别明确、短促、可直接执行的发布或定稿确认语。
 * 普通讨论即使包含“发布”“定稿”等词，也不能自动升级为写操作。
 */

import type {
  ChipDef,
} from './conversationScript'

const publishIntentTexts =
  new Set<string>([
    '教案我满意了就这样定稿',
    '我满意了就这样定稿',
    '就这样定稿',
    '完成并发布',
    '不用改了发布',
    '确认发布',
    '确认发布教案',
    '发布教案',
    '就按这个版本发布',
    '就按这个版本定稿',
    '确定这个版本',
    '确认这个版本',
    '就这个版本',
  ])

function normalizeConversationActionIntent(
  value: string,
): string {
  return (value || '')
    .toLowerCase()
    .replace(
      /[\s，,。.!！?？；;：:]/g,
      '',
    )
    .trim()
}

export function isConversationPublishIntent(
  value: string,
): boolean {
  const normalized =
    normalizeConversationActionIntent(
      value,
    )

  return (
    normalized.length > 0 &&
    publishIntentTexts.has(normalized)
  )
}

export function buildConversationChipActionKey(
  chip: ChipDef,
): string {
  return [
    chip.id || '',
    chip.action_type || '',
    chip.payload?.text || '',
    chip.payload?.stage || '',
    chip.payload?.tool || '',
  ].join('\u001f')
}

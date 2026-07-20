/**
 * contextReceiptVisibility.ts — 消息列表层的回执显示与去重规则
 *
 * 规则：
 *   1. 没有实际加载项且没有警告，不展示；
 *   2. 第一份有意义的回执展示；
 *   3. 实际加载资源签名变化时展示；
 *   4. 连续相同签名不重复展示；
 *   5. 真实失败警告每次都展示。
 *
 * 去重放在消息列表层，而不是单个AIBubble内部。
 * 因为只有列表层知道前一条AI消息使用了哪些资源。
 */

import type {
  ConversationMessage,
} from '@/api/lesson-plans'
import {
  buildContextReceiptView,
} from './contextReceiptViewModel'

export function getContextReceiptDisplayMessageIds(
  messages: ConversationMessage[],
): Set<string> {
  const visibleMessageIDs = new Set<string>()

  let previousMeaningfulSignature = ''

  for (const message of messages) {
    if (message.role !== 'assistant') {
      continue
    }

    const receipt =
      message.metadata?.context_receipt

    if (!receipt) {
      continue
    }

    const view =
      buildContextReceiptView(receipt)

    const hasLoadedResources =
      view.usedItems.length > 0

    const hasWarnings =
      view.warningItems.length > 0

    if (
      !hasLoadedResources &&
      !hasWarnings
    ) {
      continue
    }

    // 警告每次都要出现，避免老师错过真实失败。
    if (
      hasWarnings ||
      view.signature !==
        previousMeaningfulSignature
    ) {
      visibleMessageIDs.add(message.id)
    }

    previousMeaningfulSignature =
      view.signature
  }

  return visibleMessageIDs
}

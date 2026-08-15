/**
 * conversationAttachmentQueue.ts — 对话附件队列的纯状态与组装规则
 *
 * 参考 ChatGPT composer 的多附件心智：
 * - 可一次选择多个，也可连续追加；
 * - 每个附件独立显示状态、重试、删除；
 * - 成功发送后只消费本轮已就绪附件；
 * - 失败附件明确标记“未加入本轮”，不拖累其它已就绪附件。
 */

export type ConversationAttachmentStatus =
  | 'processing'
  | 'ready'
  | 'error'

export interface ConversationAttachmentItem {
  id: string
  file: File
  fileName: string
  fingerprint: string
  status: ConversationAttachmentStatus
  progress: string
  text: string
  charCount: number
  error: string
}

export interface ConversationAttachmentQueueStats {
  readyCount: number
  processingCount: number
  errorCount: number
  totalCount: number
  totalInjectRunes: number
  blocking: boolean
  blockingReason: string
}

/** 与当前 ChatGPT 多附件交互保持同一数量级；TE-DNA仍保留自己的单文件10MB限制。 */
export const MAX_CONVERSATION_ATTACHMENTS = 20

/** 同时做浏览器解析/视觉转录的文件数，避免多PDF、多图片瞬间打满浏览器和后端。 */
export const CONVERSATION_ATTACHMENT_PROCESS_CONCURRENCY = 2

/**
 * 本轮直接注入的附件文本总预算。
 *
 * 单个长文件会先经过已有compressRefMaterial提炼；这里再做总量闸门，
 * 超出时明确提示老师移除部分附件，不做静默截断。
 */
export const MAX_CONVERSATION_ATTACHMENT_INJECT_RUNES = 80_000

export function conversationAttachmentFingerprint(
  file: File,
): string {
  return [
    file.name,
    file.size,
    file.lastModified,
    file.type,
  ].join('\u001f')
}

export function createConversationAttachmentItem(
  file: File,
  sequence: number,
): ConversationAttachmentItem {
  return {
    id: `attachment_${Date.now()}_${sequence}_${Math.random()
      .toString(36)
      .slice(2, 8)}`,
    file,
    fileName: file.name,
    fingerprint: conversationAttachmentFingerprint(file),
    status: 'processing',
    progress: '等待处理…',
    text: '',
    charCount: 0,
    error: '',
  }
}

export function buildConversationAttachmentMaterial(
  items: ConversationAttachmentItem[],
): string {
  return items
    .filter(
      item =>
        item.status === 'ready' &&
        item.text.trim(),
    )
    .map(
      (item, index) =>
        `【附件${index + 1}：${item.fileName}】\n${item.text.trim()}`,
    )
    .join('\n\n')
}

export function getConversationAttachmentQueueStats(
  items: ConversationAttachmentItem[],
): ConversationAttachmentQueueStats {
  const ready = items.filter(item => item.status === 'ready')
  const processingCount =
    items.filter(item => item.status === 'processing').length
  const errorCount =
    items.filter(item => item.status === 'error').length

  const material =
    buildConversationAttachmentMaterial(ready)
  const totalInjectRunes =
    Array.from(material).length

  const overBudget =
    totalInjectRunes >
    MAX_CONVERSATION_ATTACHMENT_INJECT_RUNES

  return {
    readyCount: ready.length,
    processingCount,
    errorCount,
    totalCount: items.length,
    totalInjectRunes,
    blocking:
      processingCount > 0 ||
      overBudget,
    blockingReason:
      processingCount > 0
        ? `还有 ${processingCount} 个附件正在处理`
        : overBudget
          ? '本轮附件内容合计过长，请移除部分附件后再发送'
          : '',
  }
}

export function isConversationAttachmentImage(
  file: File,
): boolean {
  const name = file.name.toLowerCase()

  return (
    file.type === 'image/jpeg' ||
    file.type === 'image/png' ||
    file.type === 'image/webp' ||
    /\.(jpe?g|png|webp)$/i.test(name)
  )
}

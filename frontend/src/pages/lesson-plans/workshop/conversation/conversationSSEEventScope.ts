/**
 * conversationSSEEventScope.ts — 对话SSE事件作用域判定。
 *
 * 后端对老师主动发起的聊天轮次返回clientTurnId。
 * 导入后的后台AI评审、阶段通知等系统旁路事件不带clientTurnId。
 *
 * 必须严格隔离两类事件：
 *   - current_turn：允许控制输入框的“AI思考中”状态；
 *   - background：允许写入最终消息、评审报告和阶段状态，
 *     但绝不能锁定老师的输入框、正文编辑或图片移除；
 *   - stale_turn：过期轮次，直接丢弃。
 */

export type ConversationSSEEventScope =
  | 'current_turn'
  | 'background'
  | 'stale_turn'

export function classifyConversationSSEEvent(
  clientTurnId: string | undefined,
  currentTurnId: string,
): ConversationSSEEventScope {
  const eventTurnID =
    clientTurnId?.trim() || ''

  const activeTurnID =
    currentTurnId.trim()

  if (!eventTurnID) {
    return 'background'
  }

  if (
    activeTurnID &&
    eventTurnID === activeTurnID
  ) {
    return 'current_turn'
  }

  return 'stale_turn'
}

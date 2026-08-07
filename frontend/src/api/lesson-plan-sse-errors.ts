/**
 * lesson-plan-sse-errors.ts — 教案SSE错误分类与教师端安全文案
 *
 * 同一个EventSource既可能收到：
 *   1. 后端主动发送的命名事件“event: error”；
 *   2. 浏览器因网络、代理或服务关闭触发的原生onerror。
 *
 * 两者拥有相同的事件名称，但含义完全不同：
 *   - 命名业务错误是带data的MessageEvent，只结束当前生成轮次；
 *   - 原生连接错误通常不带data，才应该进入指数退避重连。
 *
 * 本模块只做确定性分类和教师端文案清理，不改变主对话内容。
 */

/**
 * 判断当前error事件是否为后端主动发送的业务错误。
 *
 * 自定义SSE事件会携带字符串data；真实连接异常事件没有业务data。
 * 不使用instanceof MessageEvent，避免部分浏览器或测试环境的构造器差异。
 */
export function isLessonPlanSSEBusinessErrorEvent(
  event: Event,
): boolean {
  const candidate =
    event as Event & {
      data?: unknown
    }

  return (
    event.type === 'error' &&
    typeof candidate.data === 'string' &&
    candidate.data.trim().length > 0
  )
}

const INTERNAL_ERROR_SIGNALS = [
  'SQLSTATE',
  'panic:',
  'stack trace',
  'context deadline exceeded',
  'connection refused',
  'dial tcp',
  'unexpected EOF',
  'runtime error',
  'http://',
  'https://',
]

/**
 * 把后端业务错误整理为教师可读文案。
 *
 * 后端多数错误已经是安全中文提示；这里只压平空白、限制长度，
 * 并拦截明显的数据库、网络栈或运行时内部信息。
 */
export function normalizeLessonPlanBusinessErrorMessage(
  error: string,
): string {
  const normalized =
    error
      .replace(/\s+/g, ' ')
      .trim()

  if (!normalized) {
    return '本轮生成没有完成，请稍后重试。'
  }

  const containsInternalDetail =
    INTERNAL_ERROR_SIGNALS.some(
      signal =>
        normalized
          .toLowerCase()
          .includes(
            signal.toLowerCase(),
          ),
    )

  if (containsInternalDetail) {
    return '本轮生成没有完成，请稍后重试。之前的对话和教案内容仍然保留。'
  }

  const runes =
    Array.from(normalized)

  if (runes.length <= 240) {
    return normalized
  }

  return (
    runes
      .slice(0, 240)
      .join('') +
    '…'
  )
}

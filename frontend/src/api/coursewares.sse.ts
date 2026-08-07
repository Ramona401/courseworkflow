/**
 * coursewares.sse.ts
 *
 * 课件索引生成、页面批量生成和全自动装配共用SSE连接管理。
 *
 * 业务边界：
 *   1. 业务事件映射由coursewares.sse.events.ts负责；
 *   2. JSON载荷类型分发由coursewares.sse.dispatch.ts负责；
 *   3. 本文件只管理连接、正常完成、主动关闭和指数退避重连；
 *   4. 带JSON数据的业务error事件不会关闭整条连接；
 *   5. 正常完成或业务层主动关闭后不再重连。
 */

import type { CWSSECallbacks } from './coursewares.types'
import { readCWSSEEventData } from './coursewares.sse.dispatch'
import { bindCoursewareSSEEvents } from './coursewares.sse.events'

const CW_SSE_RECONNECT_MAX_RETRIES = 5
const CW_SSE_RECONNECT_BASE_DELAY_MS = 1000
const CW_SSE_RECONNECT_MAX_DELAY_MS = 30000

/**
 * 订阅课件工坊SSE。
 */
export function subscribeCWIndexSSE(
  coursewareId: string,
  callbacks: CWSSECallbacks,
): {
  close: () => void
} {
  const token = localStorage.getItem('token') || ''
  const url =
    `${window.location.origin}/api/v1/sse/courseware/${coursewareId}`
    + `?token=${encodeURIComponent(token)}`

  let currentES: EventSource | null = null
  let retryCount = 0
  let retryTimer: ReturnType<typeof setTimeout> | null = null
  let isClosed = false
  let isFirstConnect = true

  /**
   * 正常完成index、gen或assembly流程。
   *
   * 先标记关闭，再关闭EventSource，防止浏览器随后触发
   * onerror并错误进入重连流程。
   */
  const finishConnection = (
    eventSource: EventSource,
  ) => {
    isClosed = true

    if (retryTimer) {
      clearTimeout(retryTimer)
      retryTimer = null
    }

    eventSource.close()

    if (currentES === eventSource) {
      currentES = null
    }
  }

  const bindConnection = (
    eventSource: EventSource,
  ) => {
    bindCoursewareSSEEvents(
      eventSource,
      callbacks,
      {
        onConnected: () => {
          retryCount = 0

          if (retryTimer) {
            clearTimeout(retryTimer)
            retryTimer = null
          }

          if (!isFirstConnect) {
            callbacks.onReconnected?.()
          }

          isFirstConnect = false
        },

        onTerminal: () => {
          finishConnection(eventSource)
        },
      },
    )

    /**
     * 处理EventSource传输层断线。
     *
     * 服务端自定义的error业务事件也会触发同名onerror属性；
     * 只要事件带有JSON数据，就说明事件已经交给业务监听器处理，
     * 这里必须直接返回，不能关闭连接。
     */
    eventSource.onerror = event => {
      if (readCWSSEEventData(event)) {
        return
      }

      if (
        isClosed ||
        currentES !== eventSource
      ) {
        return
      }

      eventSource.close()
      currentES = null

      if (retryTimer) {
        return
      }

      if (
        retryCount >=
        CW_SSE_RECONNECT_MAX_RETRIES
      ) {
        callbacks.onError?.({
          message:
            '连接已断开且多次重连失败，请刷新页面查看最新进度'
            + '（生成仍在后台继续，不会丢失）',
        })
        return
      }

      const delay = Math.min(
        CW_SSE_RECONNECT_BASE_DELAY_MS
          * Math.pow(2, retryCount),
        CW_SSE_RECONNECT_MAX_DELAY_MS,
      )

      retryCount += 1

      console.log(
        `[CW-SSE] 连接断开，${delay / 1000}秒后`
        + `第${retryCount}次重连…`
        + `（courseware: ${coursewareId}）`,
      )

      retryTimer = setTimeout(() => {
        retryTimer = null

        if (
          !isClosed &&
          !currentES
        ) {
          connectSSE()
        }
      }, delay)
    }
  }

  const connectSSE = () => {
    if (
      isClosed ||
      currentES
    ) {
      return
    }

    const eventSource = new EventSource(url)

    currentES = eventSource
    bindConnection(eventSource)
  }

  connectSSE()

  return {
    close: () => {
      isClosed = true

      if (retryTimer) {
        clearTimeout(retryTimer)
        retryTimer = null
      }

      if (currentES) {
        currentES.close()
        currentES = null
      }
    },
  }
}

/**
 * coursewares.sse.events.ts
 *
 * 课件工坊SSE业务事件与CWSSECallbacks之间的映射。
 *
 * 本文件负责：
 *   1. 注册index、gen和assembly三个事件族；
 *   2. 将服务端事件映射到类型匹配的业务回调；
 *   3. 隔离单条JSON解析错误；
 *   4. 通知连接管理层处理连接建立和正常完成。
 *
 * 断线重连、计时器与主动关闭由coursewares.sse.ts管理。
 */

import type { CWSSECallbacks } from './coursewares.types'
import {
  dispatchCWSSEPayload,
  readCWSSEEventData,
  type CWSSEPayloadCallbackName,
} from './coursewares.sse.dispatch'

export interface CoursewareSSEEventLifecycle {
  /**
   * 收到connected事件后调用。
   *
   * 即使该事件的JSON格式异常，也应恢复连接生命周期，
   * 避免连接已经建立但重连状态未复位。
   */
  onConnected: () => void

  /**
   * 收到正常完成事件后调用。
   */
  onTerminal: () => void
}

interface BindPayloadEventOptions {
  ignoreEmptyData?: boolean
  afterDispatch?: () => void
}

/**
 * 注册一个带JSON载荷的业务事件。
 *
 * callbackName同时决定目标业务回调和反序列化后的静态载荷类型。
 */
function bindPayloadEvent<
  CallbackName extends CWSSEPayloadCallbackName,
>(
  eventSource: EventSource,
  eventName: string,
  callbacks: CWSSECallbacks,
  callbackName: CallbackName,
  options: BindPayloadEventOptions = {},
): void {
  eventSource.addEventListener(eventName, event => {
    if (
      options.ignoreEmptyData &&
      !readCWSSEEventData(event)
    ) {
      return
    }

    try {
      dispatchCWSSEPayload(
        callbacks,
        callbackName,
        event,
      )
    } catch {
      // 单条SSE消息格式异常不终止整条连接。
    }

    options.afterDispatch?.()
  })
}

/**
 * 绑定课件工坊全部业务事件。
 */
export function bindCoursewareSSEEvents(
  eventSource: EventSource,
  callbacks: CWSSECallbacks,
  lifecycle: CoursewareSSEEventLifecycle,
): void {
  bindPayloadEvent(
    eventSource,
    'connected',
    callbacks,
    'onConnected',
    {
      afterDispatch: lifecycle.onConnected,
    },
  )

  bindPayloadEvent(
    eventSource,
    'index_start',
    callbacks,
    'onIndexStart',
  )

  bindPayloadEvent(
    eventSource,
    'index_page',
    callbacks,
    'onIndexPage',
  )

  bindPayloadEvent(
    eventSource,
    'index_progress',
    callbacks,
    'onIndexProgress',
  )

  bindPayloadEvent(
    eventSource,
    'index_done',
    callbacks,
    'onIndexDone',
    {
      afterDispatch: lifecycle.onTerminal,
    },
  )

  bindPayloadEvent(
    eventSource,
    'gen_start',
    callbacks,
    'onGenStart',
  )

  bindPayloadEvent(
    eventSource,
    'gen_page',
    callbacks,
    'onGenPage',
  )

  bindPayloadEvent(
    eventSource,
    'gen_progress',
    callbacks,
    'onGenProgress',
  )

  bindPayloadEvent(
    eventSource,
    'gen_done',
    callbacks,
    'onGenDone',
    {
      afterDispatch: lifecycle.onTerminal,
    },
  )

  /**
   * 自定义error业务事件可能只表示某一页失败。
   *
   * 带data的error在这里交给业务回调；
   * 没有data的浏览器传输错误交给EventSource.onerror重连。
   */
  bindPayloadEvent(
    eventSource,
    'error',
    callbacks,
    'onError',
    {
      ignoreEmptyData: true,
    },
  )

  bindPayloadEvent(
    eventSource,
    'assembly_start',
    callbacks,
    'onAssemblyStart',
  )

  bindPayloadEvent(
    eventSource,
    'assembly_page_html',
    callbacks,
    'onAssemblyPageHtml',
  )

  bindPayloadEvent(
    eventSource,
    'assembly_progress',
    callbacks,
    'onAssemblyProgress',
  )

  bindPayloadEvent(
    eventSource,
    'assembly_page_image',
    callbacks,
    'onAssemblyPageMedia',
  )

  bindPayloadEvent(
    eventSource,
    'assembly_page_video',
    callbacks,
    'onAssemblyPageMedia',
  )

  bindPayloadEvent(
    eventSource,
    'assembly_page_done',
    callbacks,
    'onAssemblyPageDone',
  )

  bindPayloadEvent(
    eventSource,
    'assembly_done',
    callbacks,
    'onAssemblyDone',
    {
      afterDispatch: lifecycle.onTerminal,
    },
  )
}

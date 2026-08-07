/**
 * coursewares.sse.contract.ts — 课件SSE编译期回调协议
 *
 * 本文件不发起网络连接，也不在运行时执行。
 *
 * tsconfig.app.json会把src下的本文件纳入严格类型检查。这里把所有SSE
 * 业务回调与统一解析器逐一接线，专门防止以下回归：
 *
 *   - 解析器返回unknown，导致unknown直接传入具体类型回调；
 *   - 某个回调协议调整后，SSE监听器仍按旧类型发送；
 *   - 通过any或无边界断言绕过严格类型检查。
 */

import type {
  CWSSECallbacks,
} from './coursewares.types'

import {
  parseCoursewareSSEEvent,
} from './coursewares.sse-payload'

/**
 * verifyCoursewareSSECallbackContract 只用于TypeScript编译期检查。
 *
 * 泛型负载类型会由每个callback的参数自动反向推导；如果解析器失去
 * 泛型能力或某个回调类型不兼容，tsc会在此文件直接失败。
 */
export function verifyCoursewareSSECallbackContract(
  event: Event,
  callbacks: CWSSECallbacks,
): void {
  callbacks.onConnected?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onIndexStart?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onIndexPage?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onIndexProgress?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onIndexDone?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onGenStart?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onGenPage?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onGenProgress?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onGenDone?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onError?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onAssemblyStart?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onAssemblyPageHtml?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onAssemblyProgress?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onAssemblyPageMedia?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onAssemblyPageDone?.(
    parseCoursewareSSEEvent(event),
  )

  callbacks.onAssemblyDone?.(
    parseCoursewareSSEEvent(event),
  )
}

/**
 * coursewares.sse.dispatch.ts
 *
 * 课件工坊SSE事件的类型安全JSON分发边界。
 *
 * JSON.parse只能在运行时得到未知结构；具体业务载荷类型由
 * CWSSECallbacks中对应回调的第一个参数自动推导。
 *
 * 类型断言集中保留在JSON反序列化边界，事件监听器不使用any，
 * 也不会把unknown直接传给具体业务回调。
 */

import type { CWSSECallbacks } from './coursewares.types'

/**
 * 具有服务端JSON载荷的回调名称。
 *
 * onReconnected由客户端连接生命周期触发，没有服务端JSON载荷。
 */
export type CWSSEPayloadCallbackName =
  Exclude<keyof CWSSECallbacks, 'onReconnected'>

/**
 * 从指定回调的第一个参数推导其载荷类型。
 */
export type CWSSECallbackPayload<
  CallbackName extends CWSSEPayloadCallbackName,
> =
  NonNullable<CWSSECallbacks[CallbackName]> extends (
    payload: infer Payload,
  ) => void
    ? Payload
    : never

/**
 * 读取自定义EventSource事件中的原始文本。
 *
 * 网络层error事件通常不是MessageEvent，或者没有data，
 * 此时返回空字符串。
 */
export function readCWSSEEventData(event: Event): string {
  const data = (event as MessageEvent<unknown>).data

  return typeof data === 'string' ? data : ''
}

/**
 * 解析服务端JSON，并投递给名称匹配的业务回调。
 */
export function dispatchCWSSEPayload<
  CallbackName extends CWSSEPayloadCallbackName,
>(
  callbacks: CWSSECallbacks,
  callbackName: CallbackName,
  event: Event,
): void {
  const callback = callbacks[callbackName] as
    | ((
        payload: CWSSECallbackPayload<CallbackName>,
      ) => void)
    | undefined

  if (!callback) {
    return
  }

  const rawData = readCWSSEEventData(event)

  if (!rawData) {
    throw new Error('SSE事件缺少JSON数据')
  }

  const payload = JSON.parse(
    rawData,
  ) as CWSSECallbackPayload<CallbackName>

  callback(payload)
}

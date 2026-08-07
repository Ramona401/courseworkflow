/**
 * coursewares.sse-payload.ts — 课件SSE负载解析边界
 *
 * 职责：
 *   - 把EventSource自定义事件中的字符串data解析为JSON；
 *   - 拒绝空数据、数组、null和基础类型；
 *   - 通过调用位置的回调参数类型推导具体负载类型；
 *   - 将必要的类型断言集中在唯一边界，业务监听器不使用any或重复断言。
 *
 * 说明：
 *   EventSource只为open、message和error提供内置事件类型。
 *   自定义SSE事件在TypeScript中表现为Event，但浏览器运行时携带
 *   MessageEvent.data。解析后的JSON天然是unknown，只有在通过基础结构
 *   校验后，才可依据当前事件对应的回调协议收窄为T。
 */

/**
 * parseCoursewareSSEEvent 解析并收窄一条课件SSE自定义事件。
 *
 * 泛型T由调用处的回调参数上下文自动推导。例如：
 *
 * callbacks.onGenPage?.(
 *   parseCoursewareSSEEvent(event),
 * )
 *
 * 此时T会被推导为onGenPage所要求的负载类型。
 */
export function parseCoursewareSSEEvent<T>(
  event: Event,
): T {
  const messageEvent =
    event as MessageEvent<unknown>

  if (
    typeof messageEvent.data !== 'string' ||
    messageEvent.data.trim() === ''
  ) {
    throw new Error(
      '课件SSE事件缺少有效JSON数据',
    )
  }

  const parsed: unknown =
    JSON.parse(messageEvent.data)

  if (
    parsed === null ||
    typeof parsed !== 'object' ||
    Array.isArray(parsed)
  ) {
    throw new Error(
      '课件SSE事件负载必须是JSON对象',
    )
  }

  return parsed as T
}

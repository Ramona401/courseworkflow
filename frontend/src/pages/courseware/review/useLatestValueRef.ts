/**
 * useLatestValueRef.ts
 *
 * 在保持ref对象身份稳定的同时，始终保存最新值。
 *
 * 典型用途：
 *   - effect只应由资源ID触发，不应因父组件回调函数身份变化而重新执行；
 *   - 稳定外层回调需要调用控制器最新实现；
 *   - 避免把业务回调放入数据加载effect依赖，造成重复请求或循环刷新。
 */

import {
  useLayoutEffect,
  useRef,
} from "react";

import type {
  MutableRefObject,
} from "react";

/**
 * 返回身份稳定、内容始终为最新值的ref。
 */
export function useLatestValueRef<T>(
  value: T,
): MutableRefObject<T> {
  const valueRef =
    useRef(value);

  useLayoutEffect(() => {
    valueRef.current = value;
  }, [
    value,
  ]);

  return valueRef;
}

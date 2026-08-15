/**
 * coursewareAssemblyStateFacts.ts — 自动装配状态事实与安全收窄。
 *
 * 设计目的：
 * - CoursewareAssemblyState 同时承载 assembly、batch 和空闲状态；
 * - integrity / image_repair 都是可空服务端事实，调用方不得自行推算；
 * - 类型守卫必须只收窄到真正满足条件的子类型，不能把任意
 *   CoursewareAssemblyState 作为守卫结果，否则 false 分支会把合法状态错误收窄为 never。
 */

import type {
  CoursewareAssemblyState,
  CoursewareGenerationIntegrity,
  CoursewareGenerationRunKind,
  CoursewareImageRepairState,
} from '@/api/coursewareAssembly'

export type CoursewareAssemblyIncompleteIntegrityState =
  CoursewareAssemblyState & {
    run_kind: 'assembly'
    integrity: CoursewareGenerationIntegrity & {
      complete: false
    }
  }

export type CoursewareAssemblyRetryableImageRepairState =
  CoursewareAssemblyState & {
    run_kind: 'assembly'
    integrity: CoursewareGenerationIntegrity & {
      complete: true
    }
    image_repair: CoursewareImageRepairState
  }

/**
 * 判断上一轮是否为需要定向补生成的 assembly HTML 不完整终态。
 *
 * 注意：返回类型只声明“真正满足该条件”的子类型，避免 TypeScript 在
 * false 分支把整个 CoursewareAssemblyState 排除掉。
 */
export function isCoursewareAssemblyIntegrityRetry(
  state: CoursewareAssemblyState | null | undefined,
): state is CoursewareAssemblyIncompleteIntegrityState {
  return Boolean(
    state &&
      state.run_kind === 'assembly' &&
      state.integrity &&
      state.integrity.complete === false,
  )
}

/** 判断是否存在服务端确认可再次智能补配的失败图片。 */
export function isCoursewareAssemblyImageRepairRetry(
  state: CoursewareAssemblyState | null | undefined,
): state is CoursewareAssemblyRetryableImageRepairState {
  return Boolean(
    state &&
      state.run_kind === 'assembly' &&
      state.integrity?.complete === true &&
      state.image_repair &&
      state.image_repair.retryable_count > 0,
  )
}

/**
 * 按运行类型读取服务端完整性事实。
 * expectedRunKind 未提供时只要求存在 integrity；提供时必须严格匹配。
 */
export function getCoursewareAssemblyIntegrity(
  state: CoursewareAssemblyState | null | undefined,
  expectedRunKind?: CoursewareGenerationRunKind,
): CoursewareGenerationIntegrity | null {
  if (
    !state ||
    !state.integrity ||
    (
      expectedRunKind &&
      state.run_kind !== expectedRunKind
    )
  ) {
    return null
  }

  return state.integrity
}

/** 按运行类型读取服务端图片修复事实。 */
export function getCoursewareAssemblyImageRepair(
  state: CoursewareAssemblyState | null | undefined,
  expectedRunKind?: CoursewareGenerationRunKind,
): CoursewareImageRepairState | null {
  if (
    !state ||
    !state.image_repair ||
    (
      expectedRunKind &&
      state.run_kind !== expectedRunKind
    )
  ) {
    return null
  }

  return state.image_repair
}

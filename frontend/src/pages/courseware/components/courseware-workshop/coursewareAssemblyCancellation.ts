/**
 * coursewareAssemblyCancellation.ts — 自动装配取消请求的网络收敛辅助
 *
 * 本文件只处理“发送取消 → 失败时立即重读数据库 → 生成教师可读提示”。
 * React状态、取消中的本地ref以及后续轮询仍由 useCoursewareAssemblyRuntime 持有。
 */
import { cancelCoursewareAutoAssembly } from '@/api/coursewareAssembly'
import type { CoursewareAssemblyState } from '@/api/coursewareAssembly'

export interface CoursewareAssemblyCancellationResolution {
  /** 空字符串表示 refresh 已经把终态写回主Hook，不再覆盖其文案。 */
  message: string
}

/**
 * 请求停止自动装配。
 *
 * HTTP报错不等于服务端一定没有收到，因此失败时必须立即读取数据库状态：
 * starting/cancel_requested 继续等待，running 提示本次停止未生效，
 * 非active则让主Hook保留数据库终态，不用错误文案覆盖真实结果。
 */
export async function resolveCoursewareAssemblyCancellation(
  coursewareId: string,
  refreshAssemblyState: () => Promise<CoursewareAssemblyState | null>,
): Promise<CoursewareAssemblyCancellationResolution> {
  try {
    const result =
      await cancelCoursewareAutoAssembly(coursewareId)

    window.setTimeout(() => {
      void refreshAssemblyState()
    }, 300)

    return {
      message: result.message,
    }
  } catch (error) {
    const failureMessage =
      '❌ 停止失败：' +
      (
        error instanceof Error
          ? error.message
          : '未知错误'
      )

    const refreshed =
      await refreshAssemblyState()

    if (!refreshed) {
      return {
        message:
          `${failureMessage} 系统会继续确认后台状态。`,
      }
    }

    if (
      refreshed.is_active &&
      refreshed.run_kind === 'assembly' &&
      refreshed.runtime_status === 'starting'
    ) {
      return {
        message:
          '停止请求状态暂未确认，系统会继续检查；请勿重复启动。',
      }
    }

    if (
      refreshed.is_active &&
      refreshed.run_kind === 'assembly' &&
      refreshed.runtime_status ===
        'cancel_requested'
    ) {
      return {
        message:
          '停止请求已进入后台，正在等待当前工作收敛。',
      }
    }

    if (
      refreshed.is_active &&
      refreshed.run_kind === 'assembly'
    ) {
      return {
        message: failureMessage,
      }
    }

    return {
      message: '',
    }
  }
}

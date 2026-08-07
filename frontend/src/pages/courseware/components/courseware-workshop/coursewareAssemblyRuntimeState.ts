/**
 * coursewareAssemblyRuntimeState.ts — 自动装配前端纯状态辅助
 *
 * 这里只保留无副作用的页面初态、数据库页面映射与运行文案判断。
 * 网络请求、React状态、SSE和取消动作仍由useCoursewareAssemblyRuntime管理。
 */
import type {
  CoursewarePage,
} from '@/api/coursewares'
import type {
  CoursewareAssemblyRuntimeStatus,
  CoursewareAssemblyState,
} from '@/api/coursewareAssembly'

import type {
  AssemblyPageState,
} from './AssemblyProgressView'

/** 构造一页装配流水线初态。 */
export function createBlankAssemblyPage(
  pageNumber: number,
  title: string,
  skipVideo: boolean,
): AssemblyPageState {
  return {
    page_number: pageNumber,
    title,
    html: 'pending',
    image: 'pending',
    video: skipVideo
      ? 'skipped'
      : 'pending',
  }
}

/** 把数据库页面快照映射为前端装配进度初态。 */
export function createAssemblyPagesFromStoredCourseware(
  pages: CoursewarePage[],
  skipVideo: boolean,
): AssemblyPageState[] {
  return pages
    .slice()
    .sort(
      (left, right) =>
        left.page_number -
        right.page_number,
    )
    .map(page => ({
      ...createBlankAssemblyPage(
        page.page_number,
        page.title ||
          `第 ${page.page_number} 页`,
        skipVideo,
      ),
      html:
        page.html_content?.trim()
          ? 'ok'
          : 'pending',
    }))
}

/**
 * 用最新数据库页面合并现有SSE阶段状态。
 *
 * HTML是否落库以数据库为准；配图和视频阶段保留当前SSE快照。
 */
export function mergeAssemblyPagesFromStoredCourseware(
  pages: CoursewarePage[],
  previous: AssemblyPageState[],
  skipVideo: boolean,
): AssemblyPageState[] {
  return pages
    .slice()
    .sort(
      (left, right) =>
        left.page_number -
        right.page_number,
    )
    .map(page => {
      const existing = previous.find(
        item =>
          item.page_number ===
          page.page_number,
      )
      const base =
        createBlankAssemblyPage(
          page.page_number,
          page.title ||
            `第 ${page.page_number} 页`,
          skipVideo,
        )

      return {
        ...base,
        ...existing,
        title:
          page.title ||
          existing?.title ||
          base.title,
        html:
          page.html_content?.trim()
            ? 'ok'
            : existing?.html ||
              'pending',
        video: skipVideo
          ? 'skipped'
          : existing?.video ||
            'pending',
      }
    })
}

/** 媒体阶段事件对应的单页局部更新。 */
export function createAssemblyMediaStagePatch(
  stage: string,
  note: string,
): Partial<AssemblyPageState> {
  if (stage === 'video_storyboard') {
    return {
      video: 'running',
      note,
    }
  }

  return {
    image: 'running',
    note,
  }
}

/** 当前终态是否属于本次启动或恢复的数据库运行。 */
export function shouldAcceptAssemblyTerminal(
  state: CoursewareAssemblyState,
  identity: {
    observedRunVersion: number
    localStart: boolean
    preRunObserved: boolean
    baselineKnown: boolean
    baselineVersion: number
    sseDone: boolean
  },
): boolean {
  if (identity.observedRunVersion > 0) {
    return (
      state.assembly_version ===
      identity.observedRunVersion
    )
  }

  if (
    (
      identity.localStart ||
      identity.preRunObserved
    ) &&
    identity.baselineKnown
  ) {
    return (
      state.assembly_version >
      identity.baselineVersion
    )
  }

  return identity.sseDone
}

/** 数据库业务运行状态对应的老师可读文案。 */
export function getAssemblyRuntimeMessage(
  status: CoursewareAssemblyRuntimeStatus,
): string {
  switch (status) {
    case 'starting':
      return '正在建立装配运行，请稍候…'
    case 'running':
      return '装配正在后台继续，页面可安全刷新。'
    case 'cancel_requested':
      return '正在停止继续派发；已经落库的页面会保留。'
    case 'completed':
      return '装配已完成，正在同步课件页面。'
    case 'cancelled':
      return '装配已取消。已完成页面保留，可继续自动装配剩余页面。'
    case 'failed':
      return '装配运行失败。请查看错误提示后重新尝试。'
    case 'interrupted':
      return '服务重启中断了本次运行。已落库结果保留，可继续断点装配。'
    default:
      return ''
  }
}

/** 判断数据库状态是否为不可继续写入的终态。 */
export function isAssemblyTerminalRuntime(
  status: CoursewareAssemblyRuntimeStatus,
): boolean {
  return (
    status === 'completed' ||
    status === 'cancelled' ||
    status === 'failed' ||
    status === 'interrupted'
  )
}

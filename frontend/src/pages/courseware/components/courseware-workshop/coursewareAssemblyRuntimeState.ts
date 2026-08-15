/**
 * coursewareAssemblyRuntimeState.ts — 自动装配前端纯状态辅助
 *
 * 这里只保留无副作用的页面初态、数据库页面映射与运行文案判断。
 * 网络请求、React状态、SSE和取消动作仍由生命周期Hook管理。
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

import {
  isCoursewareAssemblyImageRepairRetry,
  isCoursewareAssemblyIntegrityRetry,
} from './coursewareAssemblyStateFacts'

export {
  isCoursewareAssemblyImageRepairRetry,
  isCoursewareAssemblyIntegrityRetry,
} from './coursewareAssemblyStateFacts'

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

/**
 * 按页码合并一条SSE阶段补丁。
 *
 * 纯函数只处理页面数组；React状态更新和运行身份仍由生命周期Hook负责。
 */
export function patchAssemblyPageState(
  previous: AssemblyPageState[],
  pageNumber: number,
  patch: Partial<AssemblyPageState>,
  skipVideo: boolean,
): AssemblyPageState[] {
  const index = previous.findIndex(
    page => page.page_number === pageNumber,
  )

  if (index === -1) {
    return [
      ...previous,
      {
        ...createBlankAssemblyPage(
          pageNumber,
          patch.title || `第 ${pageNumber} 页`,
          skipVideo,
        ),
        ...patch,
      },
    ].sort(
      (left, right) =>
        left.page_number - right.page_number,
    )
  }

  const next = previous.slice()
  next[index] = {
    ...next[index],
    ...patch,
  }
  return next
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

function mergeUntouchedAssemblyPage(
  base: AssemblyPageState,
  previous: AssemblyPageState[],
  skipVideo: boolean,
): AssemblyPageState {
  const existing = previous.find(
    item => item.page_number === base.page_number,
  )

  if (!existing) {
    return {
      ...base,
      image: 'skipped',
      video: 'skipped',
    }
  }

  return {
    ...base,
    ...existing,
    html: base.html,
    video: skipVideo ? 'skipped' : existing.video,
  }
}

function createAssemblyPagesForImageRepair(
  base: AssemblyPageState[],
  previous: AssemblyPageState[],
  skipVideo: boolean,
  baseline: CoursewareAssemblyState,
  repairLaunching: boolean,
): AssemblyPageState[] {
  const repairState = baseline.image_repair
  if (!repairState) {
    return base
  }

  const repairByPage = new Map<
    number,
    string[]
  >()

  repairState.items.forEach(item => {
    if (item.page_number <= 0) {
      return
    }

    const messages =
      repairByPage.get(item.page_number) || []
    messages.push(item.message)
    repairByPage.set(item.page_number, messages)
  })

  const activelyRepairing =
    repairLaunching ||
    (
      baseline.is_active &&
      baseline.repair_failed_images
    )

  return base.map(page => {
    const messages = repairByPage.get(page.page_number)
    if (!messages) {
      return mergeUntouchedAssemblyPage(
        page,
        previous,
        skipVideo,
      )
    }

    const existing = previous.find(
      item => item.page_number === page.page_number,
    )

    return {
      ...page,
      ...existing,
      html: page.html,
      image: activelyRepairing
        ? 'running'
        : 'failed',
      video: 'skipped',
      note: activelyRepairing
        ? '正在智能修复失败配图…'
        : messages.join('；'),
    }
  })
}

/**
 * 为一次新启动构造页面阶段初态。
 *
 * 普通启动沿用数据库页面初态；R-04补生成只把失败/取消/缺失页重置。
 * 配图智能补配则保持非目标页状态，只把服务端确认的失败图片页切换为修复中。
 */
export function createAssemblyPagesForLaunch(
  pages: CoursewarePage[],
  previous: AssemblyPageState[],
  skipVideo: boolean,
  baseline: CoursewareAssemblyState | null | undefined,
  repairLaunching = false,
): AssemblyPageState[] {
  const base = createAssemblyPagesFromStoredCourseware(
    pages,
    skipVideo,
  )

  if (
    isCoursewareAssemblyImageRepairRetry(baseline)
  ) {
    return createAssemblyPagesForImageRepair(
      base,
      previous,
      skipVideo,
      baseline,
      repairLaunching,
    )
  }

  const integrity =
    isCoursewareAssemblyIntegrityRetry(baseline)
      ? baseline.integrity
      : null

  if (!integrity) {
    return base
  }

  const retryPageNumbers = new Set(
    [
      ...integrity.failed_pages,
      ...integrity.cancelled_pages,
      ...integrity.missing_pages,
    ]
      .map(item => item.page_number)
      .filter(pageNumber => pageNumber > 0),
  )

  return base.map(page => {
    if (retryPageNumbers.has(page.page_number)) {
      return {
        ...page,
        html: 'pending',
        image: 'pending',
        video: skipVideo ? 'skipped' : 'pending',
      }
    }

    return mergeUntouchedAssemblyPage(
      page,
      previous,
      skipVideo,
    )
  })
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
  repairFailedImages = false,
): string {
  if (repairFailedImages) {
    switch (status) {
      case 'starting':
        return '正在建立失败配图智能补配运行，请稍候…'
      case 'running':
        return '正在智能修复失败配图；成功图片不会重新生成。'
      case 'cancel_requested':
        return '正在停止智能补配；已经成功写入的图片会保留。'
      case 'completed':
        return '智能补配已完成，正在同步课件页面。'
      case 'cancelled':
        return '智能补配已取消。已经完成的配图会保留。'
      case 'failed':
        return '智能补配仍有未成功图片，请查看失败原因后按需再次补配。'
      case 'interrupted':
        return '服务重启中断了智能补配，已经落库的图片会保留。'
      default:
        return ''
    }
  }

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

/**
 * shouldRecoverIncompleteAssemblyState 判断数据库终态是否仍需回到自动装配恢复面板。
 *
 * 两类终态需要恢复：
 *   - R-04 HTML完整性未通过，需要只补生成未成功页；
 *   - HTML已完整，但仍存在服务端确认可智能补配的图片失败。
 *
 * 完整且无媒体失败的历史终态不会恢复，避免长期劫持课件工坊路由。
 */
export function shouldRecoverIncompleteAssemblyState(
  state: CoursewareAssemblyState | null | undefined,
): state is CoursewareAssemblyState {
  if (
    !state ||
    state.run_kind !== 'assembly' ||
    state.is_active ||
    !isAssemblyTerminalRuntime(state.runtime_status)
  ) {
    return false
  }

  const needsHTMLRecovery = Boolean(
    state.integrity &&
      !state.integrity.complete,
  )
  const needsImageRepair = Boolean(
    state.image_repair &&
      state.image_repair.retryable_count > 0,
  )

  return needsHTMLRecovery || needsImageRepair
}

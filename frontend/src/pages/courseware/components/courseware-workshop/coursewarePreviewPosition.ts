/**
 * 课件工坊当前预览页位置记忆。
 *
 * 设计原则：
 *   - 只保存稳定page_id，不保存可能因排序变化的page_number；
 *   - 使用sessionStorage，仅在当前浏览器标签页生命周期内保留；
 *   - 存储键按courseware_id隔离，避免不同课件串页；
 *   - sessionStorage不可用时退回模块内存，不影响正常课件操作；
 *   - 刷新后若原页面已删除，则清理失效记录并沿用现有首页兜底。
 */

const COURSEWARE_PREVIEW_POSITION_STORAGE_PREFIX =
  'tedna:courseware-preview:last-page:'

const memoryPositions = new Map<string, string>()

function normalizeID(value: string): string {
  return value.trim()
}

function storageKey(coursewareId: string): string {
  return `${COURSEWARE_PREVIEW_POSITION_STORAGE_PREFIX}${coursewareId}`
}

function resolveCoursewareIDFromLocation(): string {
  if (typeof window === 'undefined') return ''

  const match = window.location.pathname.match(
    /^\/courseware\/([^/?#]+)/,
  )
  if (!match?.[1]) return ''

  try {
    return normalizeID(decodeURIComponent(match[1]))
  } catch {
    return normalizeID(match[1])
  }
}

function resolveCoursewareID(coursewareId?: string): string {
  return normalizeID(
    coursewareId || resolveCoursewareIDFromLocation(),
  )
}

function clearRememberedCoursewarePreviewPage(coursewareId: string): void {
  const normalizedCoursewareID = normalizeID(coursewareId)
  if (!normalizedCoursewareID) return

  memoryPositions.delete(normalizedCoursewareID)

  if (typeof window === 'undefined') return

  try {
    window.sessionStorage.removeItem(
      storageKey(normalizedCoursewareID),
    )
  } catch {
    // sessionStorage不可用时模块内存已完成清理。
  }
}

/**
 * 记录当前稳定page_id。
 *
 * 全屏与放映组件不直接持有courseware_id时，会从当前课件路由解析；
 * 普通工坊预览则显式传入courseware_id，优先使用显式值。
 */
export function rememberCoursewarePreviewPage(
  pageId: string,
  coursewareId?: string,
): void {
  const normalizedCoursewareID = resolveCoursewareID(coursewareId)
  const normalizedPageID = normalizeID(pageId)

  if (!normalizedCoursewareID || !normalizedPageID) return

  memoryPositions.set(
    normalizedCoursewareID,
    normalizedPageID,
  )

  if (typeof window === 'undefined') return

  try {
    window.sessionStorage.setItem(
      storageKey(normalizedCoursewareID),
      normalizedPageID,
    )
  } catch {
    // 浏览器禁用sessionStorage时保留当前标签页模块内存即可。
  }
}

export function readRememberedCoursewarePreviewPage(
  coursewareId: string,
): string {
  const normalizedCoursewareID = normalizeID(coursewareId)
  if (!normalizedCoursewareID) return ''

  const memoryPageID = memoryPositions.get(
    normalizedCoursewareID,
  )
  if (memoryPageID) return memoryPageID

  if (typeof window === 'undefined') return ''

  try {
    const stored = normalizeID(
      window.sessionStorage.getItem(
        storageKey(normalizedCoursewareID),
      ) || '',
    )

    if (stored) {
      memoryPositions.set(
        normalizedCoursewareID,
        stored,
      )
    }

    return stored
  } catch {
    return ''
  }
}

/**
 * 根据稳定page_id恢复当前页码。
 *
 * 页面排序后仍可恢复到同一页面；如果页面已删除，则主动清理失效记录，
 * 让调用方继续使用现有默认页逻辑。
 */
export function resolveRememberedCoursewarePreviewPageNumber(
  coursewareId: string,
  pages: Array<{
    id?: string
    page_number: number
  }>,
): number | null {
  const normalizedCoursewareID = normalizeID(coursewareId)
  const rememberedPageID = readRememberedCoursewarePreviewPage(
    normalizedCoursewareID,
  )

  if (!normalizedCoursewareID || !rememberedPageID) {
    return null
  }

  const restoredPage = pages.find(
    (page) => normalizeID(page.id || '') === rememberedPageID,
  )

  if (restoredPage) {
    return restoredPage.page_number
  }

  clearRememberedCoursewarePreviewPage(
    normalizedCoursewareID,
  )
  return null
}

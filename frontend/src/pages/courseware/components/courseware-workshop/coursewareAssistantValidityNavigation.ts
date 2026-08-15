/**
 * 教学智能体“使用时间已到”恢复导航。
 *
 * 该模块只负责同一课件工坊页面内的轻量导航协调：
 *   - 预览悬浮层记录“打开当前页使用时间设置”的一次性请求；
 *   - 已在教学智能体工作台时直接通知当前面板消费请求；
 *   - 尚未进入教学智能体工作台时，只点击真正的工作台Tab，排除悬浮层自身按钮；
 *   - 教学智能体面板挂载后消费请求并打开“试用与发布”的有效期设置；
 *   - sessionStorage不可用时退回当前页面内存，不影响正常预览与发布。
 *
 * 不保存部署数据、不调用API，也不改变后端有效期与权限校验。
 */

export const COURSEWARE_ASSISTANT_VALIDITY_NAVIGATION_EVENT =
  'tedna:courseware-assistant:open-validity-settings'

export const COURSEWARE_ASSISTANT_VALIDITY_SECTION_ID =
  'courseware-assistant-validity-section'

const COURSEWARE_ASSISTANT_VALIDITY_NAVIGATION_STORAGE_KEY =
  'tedna:courseware-assistant:pending-validity-settings'

const COURSEWARE_ASSISTANT_OVERLAY_ROOT_SELECTOR =
  '[data-courseware-assistant-overlay-root="true"]'

interface CoursewareAssistantValidityNavigationRequest {
  coursewareId: string
  pageId: string
}

let memoryRequest: CoursewareAssistantValidityNavigationRequest | null = null

function normalizeNavigationID(value: string): string {
  return value.trim()
}

function normalizeButtonText(value: string | null): string {
  return (value || '').replace(/\s+/g, ' ').trim()
}

function storeRequest(request: CoursewareAssistantValidityNavigationRequest): void {
  memoryRequest = request

  if (typeof window === 'undefined') return

  try {
    window.sessionStorage.setItem(
      COURSEWARE_ASSISTANT_VALIDITY_NAVIGATION_STORAGE_KEY,
      JSON.stringify(request),
    )
  } catch {
    // 浏览器禁用sessionStorage时保留模块内存请求即可。
  }
}

function readStoredRequest(): CoursewareAssistantValidityNavigationRequest | null {
  if (memoryRequest) return memoryRequest
  if (typeof window === 'undefined') return null

  try {
    const raw = window.sessionStorage.getItem(
      COURSEWARE_ASSISTANT_VALIDITY_NAVIGATION_STORAGE_KEY,
    )
    if (!raw) return null

    const parsed = JSON.parse(raw) as Partial<CoursewareAssistantValidityNavigationRequest>
    const coursewareId = normalizeNavigationID(parsed.coursewareId || '')
    const pageId = normalizeNavigationID(parsed.pageId || '')

    return coursewareId && pageId ? { coursewareId, pageId } : null
  } catch {
    return null
  }
}

function clearStoredRequest(): void {
  memoryRequest = null

  if (typeof window === 'undefined') return

  try {
    window.sessionStorage.removeItem(
      COURSEWARE_ASSISTANT_VALIDITY_NAVIGATION_STORAGE_KEY,
    )
  } catch {
    // 清理失败不影响当前页面继续工作。
  }
}

function dispatchNavigationEvent(): void {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new Event(COURSEWARE_ASSISTANT_VALIDITY_NAVIGATION_EVENT))
}

function findWorkshopAssistantTab(): HTMLButtonElement | null {
  if (typeof document === 'undefined') return null

  const candidates = Array.from(
    document.querySelectorAll<HTMLButtonElement>('button'),
  )

  return candidates.find((button) => {
    if (button.disabled || button.closest(COURSEWARE_ASSISTANT_OVERLAY_ROOT_SELECTOR)) {
      return false
    }

    return normalizeButtonText(button.textContent).includes('教学智能体')
  }) || null
}

/**
 * 请求打开当前课件、当前稳定page_id对应的使用时间设置。
 *
 * 返回false只表示当前页面找不到可到达的教学智能体工作台入口；
 * 找到入口后先保存请求，再切换外层工作台Tab。面板挂载时会从内存/sessionStorage
 * 消费请求，因此不依赖“点击Tab后立刻同步挂载”的时序假设。
 */
export function requestCoursewareAssistantValiditySettings(
  coursewareId: string,
  pageId: string,
): boolean {
  const normalizedCoursewareID = normalizeNavigationID(coursewareId)
  const normalizedPageID = normalizeNavigationID(pageId)

  if (
    !normalizedCoursewareID
    || !normalizedPageID
    || typeof window === 'undefined'
    || typeof document === 'undefined'
  ) {
    return false
  }

  const validitySection = document.getElementById(
    COURSEWARE_ASSISTANT_VALIDITY_SECTION_ID,
  )
  const assistantTab = validitySection ? null : findWorkshopAssistantTab()

  if (!validitySection && !assistantTab) {
    return false
  }

  storeRequest({
    coursewareId: normalizedCoursewareID,
    pageId: normalizedPageID,
  })

  dispatchNavigationEvent()

  if (validitySection) {
    return true
  }

  assistantTab?.click()

  window.requestAnimationFrame(() => {
    dispatchNavigationEvent()
  })
  window.setTimeout(() => {
    dispatchNavigationEvent()
  }, 80)

  return true
}

export function consumeCoursewareAssistantValiditySettingsRequest(
  coursewareId: string,
  pageId: string,
): boolean {
  const request = readStoredRequest()
  if (!request) return false

  if (
    request.coursewareId !== normalizeNavigationID(coursewareId)
    || request.pageId !== normalizeNavigationID(pageId)
  ) {
    return false
  }

  clearStoredRequest()
  return true
}

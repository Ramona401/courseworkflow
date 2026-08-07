/**
 * 教师端前端版本自动检测与安全刷新。
 *
 * 目标：
 *   1. 部署新前端后，已打开页面不再要求教师手动Ctrl+Shift+R；
 *   2. 页面无未保存内容时自动切换到新版本；
 *   3. 检测到受保护草稿或正在编辑的输入框时，等待保存后再刷新；
 *   4. 通过版本查询参数绕过旧index.html缓存；
 *   5. 防止异常缓存导致无限刷新循环。
 *
 * 本模块不依赖React状态和业务路由，可在main.tsx启动。
 */

interface TednaVersionManifest {
  build_id: string
  built_at?: string
}

interface UpdateBannerElements {
  root: HTMLDivElement
  message: HTMLSpanElement
  action: HTMLButtonElement
}

const CURRENT_BUILD_ID =
  String(
    import.meta.env
      .VITE_TEDNA_BUILD_ID ||
      '',
  ).trim()

const VERSION_MANIFEST_PATH =
  '/version.json'

const VERSION_CHECK_INTERVAL_MS =
  60 * 1000

const VERSION_CHECK_START_DELAY_MS =
  3000

const SAFE_REFRESH_RECHECK_MS =
  2000

const AUTO_REFRESH_DELAY_MS =
  700

const REFRESH_LOOP_GUARD_MS =
  30 * 1000

const PROTECTED_DRAFT_PREFIX =
  'tedna_protected_draft_v1:'

const REFRESH_ATTEMPT_PREFIX =
  'tedna_frontend_refresh_attempt_v1:'

const UPDATE_BANNER_ID =
  'tedna-frontend-update-banner'

const NON_TEXT_INPUT_TYPES =
  new Set([
    'button',
    'checkbox',
    'color',
    'file',
    'hidden',
    'image',
    'radio',
    'range',
    'reset',
    'submit',
  ])

let started = false
let checking = false
let pendingBuildID = ''
let pendingRefreshTimer:
  number | null = null
let automaticRefreshTimer:
  number | null = null
let bannerElements:
  UpdateBannerElements | null = null

/**
 * 安全读取sessionStorage。
 */
function getSessionStorageSafe():
  | Storage
  | null {
  try {
    return window.sessionStorage
  } catch {
    return null
  }
}

/**
 * 判断当前标签页是否存在未提交的受保护草稿。
 *
 * useProtectedDraft只会在实际编辑后写入sessionStorage，
 * 保存成功调用clear后会删除对应键。
 */
function hasProtectedDraft(): boolean {
  const storage =
    getSessionStorageSafe()

  if (!storage) {
    return false
  }

  try {
    for (
      let index = 0;
      index < storage.length;
      index += 1
    ) {
      const key =
        storage.key(index)

      if (
        !key ||
        !key.startsWith(
          PROTECTED_DRAFT_PREFIX,
        )
      ) {
        continue
      }

      const raw =
        storage.getItem(key)

      if (!raw) {
        continue
      }

      try {
        const parsed =
          JSON.parse(raw) as {
            value?: unknown
          }

        if (
          typeof parsed.value ===
            'string' &&
          parsed.value.trim() !== ''
        ) {
          return true
        }
      } catch {
        /**
         * 无法识别的草稿记录按“存在未保存内容”处理，
         * 宁可延后刷新，也不能冒险丢失教师输入。
         */
        return true
      }
    }
  } catch {
    return false
  }

  return false
}

/**
 * 判断教师当前是否正在输入。
 *
 * 这可以保护尚未接入useProtectedDraft的普通输入框。
 * 输入框失去焦点或完成保存后，检测器会再次判断并自动刷新。
 */
function hasActiveEditableInput():
  boolean {
  const active =
    document.activeElement

  if (
    active instanceof
      HTMLTextAreaElement
  ) {
    return (
      !active.disabled &&
      !active.readOnly &&
      active.value.trim() !== ''
    )
  }

  if (
    active instanceof
      HTMLInputElement
  ) {
    return (
      !active.disabled &&
      !active.readOnly &&
      !NON_TEXT_INPUT_TYPES.has(
        active.type.toLowerCase(),
      ) &&
      active.value.trim() !== ''
    )
  }

  if (
    active instanceof HTMLElement &&
    active.isContentEditable
  ) {
    return (
      active.innerText.trim() !== ''
    )
  }

  return false
}

/**
 * 当前是否应该延后刷新。
 */
function hasUnsavedWork(): boolean {
  return (
    hasProtectedDraft() ||
    hasActiveEditableInput()
  )
}

/**
 * 创建固定在页面顶部的版本更新提示。
 */
function ensureUpdateBanner():
  UpdateBannerElements {
  if (bannerElements) {
    return bannerElements
  }

  const existing =
    document.getElementById(
      UPDATE_BANNER_ID,
    )

  if (
    existing instanceof
      HTMLDivElement
  ) {
    const existingMessage =
      existing.querySelector(
        '[data-role="message"]',
      )

    const existingAction =
      existing.querySelector(
        '[data-role="action"]',
      )

    if (
      existingMessage instanceof
        HTMLSpanElement &&
      existingAction instanceof
        HTMLButtonElement
    ) {
      bannerElements = {
        root: existing,
        message:
          existingMessage,
        action:
          existingAction,
      }

      return bannerElements
    }

    existing.remove()
  }

  const root =
    document.createElement('div')

  root.id = UPDATE_BANNER_ID
  root.setAttribute(
    'role',
    'status',
  )
  root.style.position = 'fixed'
  root.style.top = '0'
  root.style.left = '0'
  root.style.right = '0'
  root.style.zIndex = '2147483647'
  root.style.display = 'none'
  root.style.alignItems = 'center'
  root.style.justifyContent =
    'center'
  root.style.gap = '12px'
  root.style.padding = '10px 16px'
  root.style.background =
    '#1E3A8A'
  root.style.color = '#FFFFFF'
  root.style.fontSize = '13px'
  root.style.fontWeight = '600'
  root.style.lineHeight = '1.5'
  root.style.boxShadow =
    '0 4px 18px rgba(15, 23, 42, 0.22)'

  const message =
    document.createElement('span')

  message.dataset.role = 'message'
  message.style.textAlign = 'center'

  const action =
    document.createElement('button')

  action.dataset.role = 'action'
  action.type = 'button'
  action.style.display = 'none'
  action.style.padding = '6px 11px'
  action.style.borderRadius = '7px'
  action.style.border =
    '1px solid rgba(255, 255, 255, 0.72)'
  action.style.background =
    'rgba(255, 255, 255, 0.12)'
  action.style.color = '#FFFFFF'
  action.style.fontSize = '12px'
  action.style.fontWeight = '700'
  action.style.cursor = 'pointer'

  root.append(
    message,
    action,
  )

  document.body.appendChild(root)

  bannerElements = {
    root,
    message,
    action,
  }

  return bannerElements
}

/**
 * 显示版本更新状态。
 */
function showUpdateBanner(
  messageText: string,
  actionText?: string,
  onAction?: () => void,
): void {
  const elements =
    ensureUpdateBanner()

  elements.message.textContent =
    messageText

  if (
    actionText &&
    onAction
  ) {
    elements.action.textContent =
      actionText
    elements.action.onclick =
      onAction
    elements.action.style.display =
      'inline-flex'
  } else {
    elements.action.textContent = ''
    elements.action.onclick = null
    elements.action.style.display =
      'none'
  }

  elements.root.style.display =
    'flex'
}

/**
 * 清除本模块添加的版本查询参数。
 *
 * 新版本已经成功加载后恢复原始业务URL，
 * 不影响其它查询参数、路由和浏览器历史。
 */
function cleanSuccessfulRefreshQuery():
  void {
  try {
    const currentURL =
      new URL(
        window.location.href,
      )

    if (
      currentURL.searchParams.get(
        '__tedna_build',
      ) !== CURRENT_BUILD_ID
    ) {
      return
    }

    currentURL.searchParams.delete(
      '__tedna_build',
    )
    currentURL.searchParams.delete(
      '__tedna_refresh',
    )

    window.history.replaceState(
      window.history.state,
      '',
      currentURL.toString(),
    )
  } catch {
    // URL清理失败不影响版本检测。
  }
}

/**
 * 读取服务器当前前端版本。
 */
async function fetchLatestVersion():
  Promise<TednaVersionManifest | null> {
  const requestURL =
    `${VERSION_MANIFEST_PATH}` +
    `?checked_at=${Date.now()}`

  try {
    const response =
      await fetch(
        requestURL,
        {
          method: 'GET',
          cache: 'no-store',
          credentials:
            'same-origin',
          headers: {
            'Cache-Control':
              'no-cache, no-store, must-revalidate',
            Pragma: 'no-cache',
          },
        },
      )

    if (!response.ok) {
      return null
    }

    const payload =
      await response.json() as
        Partial<TednaVersionManifest>

    const buildID =
      typeof payload.build_id ===
        'string'
        ? payload.build_id.trim()
        : ''

    if (!buildID) {
      return null
    }

    return {
      build_id: buildID,
      built_at:
        typeof payload.built_at ===
          'string'
          ? payload.built_at
          : undefined,
    }
  } catch {
    /**
     * 部署切换瞬间或临时网络异常时静默等待下一次检查，
     * 不能影响当前业务页面。
     */
    return null
  }
}

/**
 * 判断是否刚刚尝试过加载同一版本，防止异常缓存造成刷新循环。
 */
function recentlyAttemptedBuild(
  buildID: string,
): boolean {
  const storage =
    getSessionStorageSafe()

  if (!storage) {
    return false
  }

  try {
    const attemptedAt =
      Number(
        storage.getItem(
          REFRESH_ATTEMPT_PREFIX +
            buildID,
        ) || '0',
      )

    return (
      Number.isFinite(
        attemptedAt,
      ) &&
      Date.now() - attemptedAt <
        REFRESH_LOOP_GUARD_MS
    )
  } catch {
    return false
  }
}

/**
 * 记录即将加载的目标版本。
 */
function recordRefreshAttempt(
  buildID: string,
): void {
  const storage =
    getSessionStorageSafe()

  if (!storage) {
    return
  }

  try {
    storage.setItem(
      REFRESH_ATTEMPT_PREFIX +
        buildID,
      String(Date.now()),
    )
  } catch {
    // 记录失败不阻断刷新。
  }
}

/**
 * 使用版本参数重新加载当前业务地址。
 */
function refreshToBuild(
  buildID: string,
  force: boolean,
): void {
  if (
    !force &&
    recentlyAttemptedBuild(
      buildID,
    )
  ) {
    showUpdateBanner(
      '检测到新版本，但浏览器仍在使用旧缓存。请点击按钮完成刷新。',
      '刷新到新版本',
      () =>
        refreshToBuild(
          buildID,
          true,
        ),
    )
    return
  }

  recordRefreshAttempt(
    buildID,
  )

  try {
    const targetURL =
      new URL(
        window.location.href,
      )

    targetURL.searchParams.set(
      '__tedna_build',
      buildID,
    )
    targetURL.searchParams.set(
      '__tedna_refresh',
      String(Date.now()),
    )

    window.location.replace(
      targetURL.toString(),
    )
  } catch {
    window.location.reload()
  }
}

/**
 * 当前页面安全时安排自动刷新。
 */
function scheduleAutomaticRefresh(
  buildID: string,
): void {
  pendingBuildID = buildID

  if (
    pendingRefreshTimer !== null
  ) {
    window.clearInterval(
      pendingRefreshTimer,
    )
    pendingRefreshTimer = null
  }

  showUpdateBanner(
    '系统已更新，正在自动刷新到新版本…',
  )

  if (
    automaticRefreshTimer !== null
  ) {
    return
  }

  automaticRefreshTimer =
    window.setTimeout(
      () => {
        automaticRefreshTimer =
          null

        refreshToBuild(
          buildID,
          false,
        )
      },
      AUTO_REFRESH_DELAY_MS,
    )
}

/**
 * 有未保存内容时等待保存完成。
 */
function waitForSafeRefresh(
  buildID: string,
): void {
  pendingBuildID = buildID

  showUpdateBanner(
    '系统已更新。检测到当前页面可能有未保存内容，保存后将自动刷新。',
    '仍要立即刷新',
    () => {
      const confirmed =
        window.confirm(
          '立即刷新可能丢失尚未保存的内容。确定继续吗？',
        )

      if (confirmed) {
        refreshToBuild(
          buildID,
          true,
        )
      }
    },
  )

  if (
    pendingRefreshTimer !== null
  ) {
    return
  }

  pendingRefreshTimer =
    window.setInterval(
      () => {
        if (
          !pendingBuildID ||
          document.visibilityState !==
            'visible' ||
          hasUnsavedWork()
        ) {
          return
        }

        const readyBuildID =
          pendingBuildID

        window.clearInterval(
          pendingRefreshTimer!,
        )
        pendingRefreshTimer = null

        scheduleAutomaticRefresh(
          readyBuildID,
        )
      },
      SAFE_REFRESH_RECHECK_MS,
    )
}

/**
 * 处理发现的新版本。
 */
function handleDetectedVersion(
  buildID: string,
): void {
  if (
    !buildID ||
    buildID === CURRENT_BUILD_ID
  ) {
    return
  }

  pendingBuildID = buildID

  if (
    document.visibilityState !==
    'visible'
  ) {
    return
  }

  if (hasUnsavedWork()) {
    waitForSafeRefresh(
      buildID,
    )
    return
  }

  scheduleAutomaticRefresh(
    buildID,
  )
}

/**
 * 执行一次版本检查。
 */
async function checkForNewVersion():
  Promise<void> {
  if (
    checking ||
    !CURRENT_BUILD_ID ||
    document.visibilityState ===
      'hidden'
  ) {
    return
  }

  checking = true

  try {
    const latest =
      await fetchLatestVersion()

    if (
      !latest ||
      latest.build_id ===
        CURRENT_BUILD_ID
    ) {
      return
    }

    handleDetectedVersion(
      latest.build_id,
    )
  } finally {
    checking = false
  }
}

/**
 * 启动全局版本检测。
 */
export function startAppVersionRefresh():
  void {
  if (
    started ||
    typeof window ===
      'undefined' ||
    typeof document ===
      'undefined' ||
    !import.meta.env.PROD ||
    !CURRENT_BUILD_ID
  ) {
    return
  }

  started = true

  cleanSuccessfulRefreshQuery()

  window.setTimeout(
    () => {
      void checkForNewVersion()
    },
    VERSION_CHECK_START_DELAY_MS,
  )

  window.setInterval(
    () => {
      void checkForNewVersion()
    },
    VERSION_CHECK_INTERVAL_MS,
  )

  window.addEventListener(
    'focus',
    () => {
      if (pendingBuildID) {
        handleDetectedVersion(
          pendingBuildID,
        )
        return
      }

      void checkForNewVersion()
    },
  )

  window.addEventListener(
    'online',
    () => {
      void checkForNewVersion()
    },
  )

  document.addEventListener(
    'visibilitychange',
    () => {
      if (
        document.visibilityState !==
        'visible'
      ) {
        return
      }

      if (pendingBuildID) {
        handleDetectedVersion(
          pendingBuildID,
        )
        return
      }

      void checkForNewVersion()
    },
  )
}

export default startAppVersionRefresh

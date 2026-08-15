/**
 * useClassroomAssistantVisibility.ts
 *
 * 全屏/放映课堂数字人的显示偏好。
 *
 * 职责边界：
 * - 只管理“隐藏 / 唤醒”视觉偏好，不拥有ASR、会话或TTS状态；
 * - 使用sessionStorage按课件保存，翻页或组件重挂载后仍保持隐藏偏好；
 * - 浏览器禁用sessionStorage时自动退化为当前页面内存状态。
 */

import { useCallback, useEffect, useMemo, useState } from 'react'

const CLASSROOM_ASSISTANT_HIDDEN_STORAGE_PREFIX =
  'tedna:courseware-assistant:hidden'

interface UseClassroomAssistantVisibilityOptions {
  coursewareId: string
  enabled: boolean
}

interface ClassroomAssistantVisibilityResult {
  hidden: boolean
  hide: () => void
  wake: () => void
}

function readHiddenPreference(storageKey: string): boolean {
  if (!storageKey || typeof window === 'undefined') {
    return false
  }

  try {
    return window.sessionStorage.getItem(storageKey) === '1'
  } catch {
    return false
  }
}

function writeHiddenPreference(
  storageKey: string,
  hidden: boolean,
): void {
  if (!storageKey || typeof window === 'undefined') {
    return
  }

  try {
    window.sessionStorage.setItem(storageKey, hidden ? '1' : '0')
  } catch {
    // sessionStorage不可用时只保留当前组件状态，课堂功能仍可继续。
  }
}

export default function useClassroomAssistantVisibility({
  coursewareId,
  enabled,
}: UseClassroomAssistantVisibilityOptions): ClassroomAssistantVisibilityResult {
  const storageKey = useMemo(() => {
    const normalizedCoursewareID = coursewareId.trim()

    if (!enabled || !normalizedCoursewareID) {
      return ''
    }

    return `${CLASSROOM_ASSISTANT_HIDDEN_STORAGE_PREFIX}:${normalizedCoursewareID}`
  }, [
    coursewareId,
    enabled,
  ])

  const [hidden, setHidden] = useState(
    () => enabled && readHiddenPreference(storageKey),
  )

  useEffect(() => {
    if (!enabled) {
      setHidden(false)
      return
    }

    setHidden(readHiddenPreference(storageKey))
  }, [
    enabled,
    storageKey,
  ])

  const hide = useCallback(() => {
    if (!enabled) {
      return
    }

    setHidden(true)
    writeHiddenPreference(storageKey, true)
  }, [
    enabled,
    storageKey,
  ])

  const wake = useCallback(() => {
    setHidden(false)
    writeHiddenPreference(storageKey, false)
  }, [storageKey])

  return {
    hidden,
    hide,
    wake,
  }
}


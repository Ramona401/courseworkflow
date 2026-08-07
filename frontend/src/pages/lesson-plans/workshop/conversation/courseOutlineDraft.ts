/**
 * courseOutlineDraft.ts — 对话模式旧课程大纲草稿兼容辅助
 *
 * 历史前端曾把课程大纲选择保存为出版社字符串：
 *   - 字段名：course-publisher
 *   - null哨兵：__TEDNA_COURSE_PUBLISHER_NONE__
 *   - 空字符串：通用 / 不限版本
 *   - 其它字符串：具体出版社
 *
 * 精确课程大纲上线后，出版社字符串无法安全转换成唯一大纲ID。
 * 本模块只负责检测并提示老师重新选择，绝不执行自动映射。
 */

import {
  buildProtectedDraftKey,
} from '@/hooks/useProtectedDraft'

const LEGACY_COURSE_PUBLISHER_NONE =
  '__TEDNA_COURSE_PUBLISHER_NONE__'

interface LegacyDraftRecord {
  value?: unknown
}

export interface LegacyCourseOutlineDraftState {
  /** 是否存在不能自动迁移的旧出版社选择。 */
  hasLegacySelection: boolean

  /** 教师可读的旧选择文案。 */
  label: string
}

interface LegacyCourseOutlineDraftIdentity {
  userId?: string | null
  resourceId: string
}

/**
 * 安全取得当前标签页的sessionStorage。
 */
function getSessionStorageSafe():
  | Storage
  | null {
  try {
    if (
      typeof window === 'undefined' ||
      !window.sessionStorage
    ) {
      return null
    }

    return window.sessionStorage
  } catch {
    return null
  }
}

/**
 * 构造旧版出版社草稿键。
 */
function buildLegacyCoursePublisherDraftKey(
  identity: LegacyCourseOutlineDraftIdentity,
): string {
  return buildProtectedDraftKey({
    userId: identity.userId,
    scope: 'lesson-plan-conversation-start',
    resourceId: identity.resourceId,
    field: 'course-publisher',
  })
}

/**
 * 检测旧版出版社草稿。
 *
 * 返回结果只用于提示，不会把出版社字符串转换成课程大纲ID。
 */
export function readLegacyCourseOutlineDraft(
  identity: LegacyCourseOutlineDraftIdentity,
): LegacyCourseOutlineDraftState {
  const storage = getSessionStorageSafe()

  if (!storage) {
    return {
      hasLegacySelection: false,
      label: '',
    }
  }

  try {
    const raw = storage.getItem(
      buildLegacyCoursePublisherDraftKey(
        identity,
      ),
    )

    if (!raw) {
      return {
        hasLegacySelection: false,
        label: '',
      }
    }

    const record = JSON.parse(
      raw,
    ) as LegacyDraftRecord

    if (typeof record.value !== 'string') {
      return {
        hasLegacySelection: false,
        label: '',
      }
    }

    if (
      record.value ===
      LEGACY_COURSE_PUBLISHER_NONE
    ) {
      return {
        hasLegacySelection: false,
        label: '',
      }
    }

    return {
      hasLegacySelection: true,
      label:
        record.value.trim() ||
        '通用 / 不限版本',
    }
  } catch {
    return {
      hasLegacySelection: false,
      label: '',
    }
  }
}

/**
 * 老师完成重新选择后清除旧版出版社草稿。
 *
 * 清理失败不影响当前精确ID草稿和正式创建请求。
 */
export function clearLegacyCourseOutlineDraft(
  identity: LegacyCourseOutlineDraftIdentity,
): void {
  const storage = getSessionStorageSafe()
  if (!storage) return

  try {
    storage.removeItem(
      buildLegacyCoursePublisherDraftKey(
        identity,
      ),
    )
  } catch {
    // sessionStorage异常不能阻断正常备课。
  }
}

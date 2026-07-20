/**
 * useProtectedDraft.ts — 全平台用户输入草稿保护 Hook
 *
 * 适用范围：
 * 1. AI 对话输入框；
 * 2. AI 微调指令；
 * 3. 较长的审核意见、方案说明和编辑草稿；
 * 4. 刷新页面或切换页面后仍应保留的未提交文字。
 *
 * 不适用范围：
 * 1. 密码、Token、API 密钥等敏感凭证；
 * 2. 图片 Base64、上传文件正文等大体积内容；
 * 3. 已由后端实时持久化、无需浏览器草稿保护的数据。
 *
 * 核心能力：
 * 1. 使用 sessionStorage，仅在当前浏览器标签页会话内保存；
 * 2. 按用户、场景、业务对象和字段隔离，避免不同用户或业务串稿；
 * 3. 输入即保存，页面刷新或切换后返回可自动恢复；
 * 4. 保存有限数量的历史快照；
 * 5. 支持 Ctrl/Command+Z 撤销；
 * 6. 支持 Ctrl/Command+Shift+Z 或 Ctrl+Y 重做；
 * 7. 提交后可把输入框清空，但保留一个可撤销快照；
 * 8. sessionStorage 不可用、隐私模式或配额满时静默降级，
 *    绝不阻断正常输入和业务请求。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import type {
  KeyboardEvent as ReactKeyboardEvent,
} from 'react'

const PROTECTED_DRAFT_PREFIX =
  'tedna_protected_draft_v1'

const PROTECTED_DRAFT_VERSION = 1

const DEFAULT_MAX_HISTORY = 40
const DEFAULT_COALESCE_MS = 700

/**
 * sessionStorage 中保存的完整草稿记录。
 *
 * history[cursor] 始终应当等于 value。
 * history 既用于当前组件生命周期内撤销，也用于刷新页面后的撤销恢复。
 */
interface ProtectedDraftRecord {
  version: number
  value: string
  history: string[]
  cursor: number
  updatedAt: number
}

/**
 * 草稿隔离身份。
 *
 * 最终键由以下四部分组成：
 * userId + scope + resourceId + field
 */
export interface ProtectedDraftIdentity {
  /** 当前登录用户ID；空值时使用anonymous兜底。 */
  userId?: string | null
  /** 功能场景，例如lesson-plan-conversation。 */
  scope: string
  /** 业务对象，例如教案ID、课件ID或页面ID。 */
  resourceId?: string | number | null
  /** 当前字段，例如message、instruction或comments。 */
  field: string
}

export interface UseProtectedDraftOptions
  extends ProtectedDraftIdentity {
  /** 没有历史草稿时使用的初始内容。 */
  initialValue?: string
  /** 是否启用浏览器草稿存储，默认启用。 */
  enabled?: boolean
  /** 最多保存多少个历史快照，默认40。 */
  maxHistory?: number
  /**
   * 连续普通输入的合并窗口。
   *
   * 在窗口内逐字输入会合并为一个撤销步骤，
   * 避免Ctrl+Z只能一个字符一个字符撤销。
   */
  coalesceMs?: number
}

export type ProtectedDraftSetter = (
  next:
    | string
    | ((previous: string) => string),
) => void

export interface UseProtectedDraftResult {
  /** 当前输入值。 */
  value: string
  /** 与React setState相同用法的更新函数。 */
  setValue: ProtectedDraftSetter
  /** 是否存在可撤销历史。 */
  canUndo: boolean
  /** 是否存在可重做历史。 */
  canRedo: boolean
  /** 撤销一次。 */
  undo: () => void
  /** 重做一次。 */
  redo: () => void
  /**
   * 提交完成后清空当前值，但保留撤销快照。
   *
   * 因此即使消息刚发送后发现发错或发送失败，
   * 仍可在空输入框中按Ctrl/Command+Z恢复。
   */
  commit: () => void
  /**
   * 完全清除当前草稿及历史。
   *
   * 仅适合用户明确要求清空，或业务对象已经永久结束。
   */
  clear: () => void
  /**
   * 输入框键盘事件处理。
   *
   * 返回true表示本次快捷键已被草稿系统消费，
   * 调用方应停止后续Enter等按键逻辑。
   */
  handleKeyDown: (
    event: ReactKeyboardEvent<HTMLElement>,
  ) => boolean
  /** 实际使用的sessionStorage键，便于调试。 */
  storageKey: string
  /** 最近一次草稿更新时间戳。 */
  updatedAt: number
}

/**
 * 对键片段进行安全编码。
 */
function encodeDraftKeyPart(
  value: string | number | null | undefined,
  fallback: string,
): string {
  const normalized =
    value === null ||
    value === undefined ||
    String(value).trim() === ''
      ? fallback
      : String(value).trim()

  return encodeURIComponent(normalized)
}

/**
 * 构造全平台统一草稿键。
 */
export function buildProtectedDraftKey(
  identity: ProtectedDraftIdentity,
): string {
  return [
    PROTECTED_DRAFT_PREFIX,
    encodeDraftKeyPart(
      identity.userId,
      'anonymous',
    ),
    encodeDraftKeyPart(
      identity.scope,
      'unknown-scope',
    ),
    encodeDraftKeyPart(
      identity.resourceId,
      'global',
    ),
    encodeDraftKeyPart(
      identity.field,
      'value',
    ),
  ].join(':')
}

/**
 * 安全取得sessionStorage。
 *
 * 浏览器隐私模式、存储被禁用或非浏览器环境时返回null。
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
 * 创建一条新的草稿记录。
 */
function createDraftRecord(
  initialValue: string,
): ProtectedDraftRecord {
  return {
    version: PROTECTED_DRAFT_VERSION,
    value: initialValue,
    history: [initialValue],
    cursor: 0,
    updatedAt: Date.now(),
  }
}

/**
 * 限制整数范围。
 */
function clampInteger(
  value: number,
  minimum: number,
  maximum: number,
): number {
  if (!Number.isFinite(value)) return minimum
  if (value < minimum) return minimum
  if (value > maximum) return maximum
  return Math.trunc(value)
}

/**
 * 规范化从sessionStorage读出的未知数据。
 */
function normalizeDraftRecord(
  raw: unknown,
  initialValue: string,
  maxHistory: number,
): ProtectedDraftRecord {
  if (
    !raw ||
    typeof raw !== 'object'
  ) {
    return createDraftRecord(initialValue)
  }

  const candidate =
    raw as Partial<ProtectedDraftRecord>

  const history = Array.isArray(candidate.history)
    ? candidate.history.filter(
        (item): item is string =>
          typeof item === 'string',
      )
    : []

  const normalizedHistory =
    history.length > 0
      ? history.slice(-maxHistory)
      : [initialValue]

  const cursor = clampInteger(
    typeof candidate.cursor === 'number'
      ? candidate.cursor
      : normalizedHistory.length - 1,
    0,
    normalizedHistory.length - 1,
  )

  const candidateValue =
    typeof candidate.value === 'string'
      ? candidate.value
      : normalizedHistory[cursor]

  normalizedHistory[cursor] =
    candidateValue

  return {
    version: PROTECTED_DRAFT_VERSION,
    value: candidateValue,
    history: normalizedHistory,
    cursor,
    updatedAt:
      typeof candidate.updatedAt === 'number'
        ? candidate.updatedAt
        : Date.now(),
  }
}

/**
 * 安全读取一条草稿。
 */
function readDraftRecord(
  storageKey: string,
  initialValue: string,
  maxHistory: number,
  enabled: boolean,
): ProtectedDraftRecord {
  if (!enabled) {
    return createDraftRecord(initialValue)
  }

  const storage = getSessionStorageSafe()
  if (!storage) {
    return createDraftRecord(initialValue)
  }

  try {
    const raw = storage.getItem(storageKey)
    if (!raw) {
      return createDraftRecord(initialValue)
    }

    return normalizeDraftRecord(
      JSON.parse(raw),
      initialValue,
      maxHistory,
    )
  } catch {
    return createDraftRecord(initialValue)
  }
}

/**
 * 安全保存一条草稿。
 */
function writeDraftRecord(
  storageKey: string,
  record: ProtectedDraftRecord,
  enabled: boolean,
): void {
  if (!enabled) return

  const storage = getSessionStorageSafe()
  if (!storage) return

  try {
    storage.setItem(
      storageKey,
      JSON.stringify(record),
    )
  } catch {
    /**
     * 配额已满、隐私模式或存储异常时静默忽略。
     * 当前React状态仍正常工作，不能因缓存失败影响业务。
     */
  }
}

/**
 * 安全删除一条草稿。
 */
function removeDraftRecord(
  storageKey: string,
  enabled: boolean,
): void {
  if (!enabled) return

  const storage = getSessionStorageSafe()
  if (!storage) return

  try {
    storage.removeItem(storageKey)
  } catch {
    // 删除缓存失败不影响正常业务。
  }
}

/**
 * 把历史裁剪到最大数量，并同步游标。
 */
function trimHistory(
  history: string[],
  maxHistory: number,
): {
  history: string[]
  cursor: number
} {
  if (history.length <= maxHistory) {
    return {
      history,
      cursor: history.length - 1,
    }
  }

  const trimmed =
    history.slice(history.length - maxHistory)

  return {
    history: trimmed,
    cursor: trimmed.length - 1,
  }
}

/**
 * 全平台文字草稿保护Hook。
 */
export function useProtectedDraft(
  options: UseProtectedDraftOptions,
): UseProtectedDraftResult {
  const {
    userId,
    scope,
    resourceId,
    field,
    initialValue = '',
    enabled = true,
    maxHistory: requestedMaxHistory =
      DEFAULT_MAX_HISTORY,
    coalesceMs: requestedCoalesceMs =
      DEFAULT_COALESCE_MS,
  } = options

  const maxHistory = Math.max(
    2,
    Math.trunc(requestedMaxHistory),
  )

  const coalesceMs = Math.max(
    0,
    Math.trunc(requestedCoalesceMs),
  )

  const storageKey = useMemo(
    () =>
      buildProtectedDraftKey({
        userId,
        scope,
        resourceId,
        field,
      }),
    [
      userId,
      scope,
      resourceId,
      field,
    ],
  )

  const [record, setRecord] =
    useState<ProtectedDraftRecord>(() =>
      readDraftRecord(
        storageKey,
        initialValue,
        maxHistory,
        enabled,
      ),
    )

  /**
   * 上一次普通输入的时间，用于把连续逐字输入合并成一个撤销步骤。
   */
  const lastEditAtRef = useRef(0)

  /**
   * 用户、场景、业务对象或字段变化时，
   * 切换到对应隔离键并恢复该键自己的草稿。
   */
  useEffect(() => {
    const nextRecord = readDraftRecord(
      storageKey,
      initialValue,
      maxHistory,
      enabled,
    )

    setRecord(nextRecord)
    lastEditAtRef.current = 0
  }, [
    storageKey,
    initialValue,
    maxHistory,
    enabled,
  ])

  /**
   * 更新当前草稿。
   */
  const setValue =
    useCallback<ProtectedDraftSetter>(
      (next) => {
        setRecord((previous) => {
          const nextValue =
            typeof next === 'function'
              ? next(previous.value)
              : next

          if (nextValue === previous.value) {
            return previous
          }

          const now = Date.now()

          /**
           * 从撤销后的历史位置重新输入时，
           * 丢弃游标之后的旧重做分支。
           */
          const activeHistory =
            previous.history.slice(
              0,
              previous.cursor + 1,
            )

          const lengthDelta = Math.abs(
            nextValue.length -
              previous.value.length,
          )

          /**
           * 下列操作强制建立独立快照：
           * 1. 一次性清空；
           * 2. 粘贴多字符；
           * 3. 选中一大段后删除或替换。
           */
          const forceBoundary =
            (
              nextValue === '' &&
              previous.value !== ''
            ) ||
            lengthDelta > 1

          const withinCoalesceWindow =
            now - lastEditAtRef.current <=
            coalesceMs

          let nextHistory: string[]
          let nextCursor: number

          if (
            !forceBoundary &&
            withinCoalesceWindow &&
            activeHistory.length > 0
          ) {
            /**
             * 普通连续输入或连续退格：
             * 替换当前快照，而不是每个字符生成一条历史。
             */
            nextHistory = [...activeHistory]
            nextHistory[
              nextHistory.length - 1
            ] = nextValue
            nextCursor =
              nextHistory.length - 1
          } else {
            const trimmed = trimHistory(
              [...activeHistory, nextValue],
              maxHistory,
            )
            nextHistory = trimmed.history
            nextCursor = trimmed.cursor
          }

          const nextRecord: ProtectedDraftRecord =
            {
              version:
                PROTECTED_DRAFT_VERSION,
              value: nextValue,
              history: nextHistory,
              cursor: nextCursor,
              updatedAt: now,
            }

          writeDraftRecord(
            storageKey,
            nextRecord,
            enabled,
          )

          lastEditAtRef.current = now

          return nextRecord
        })
      },
      [
        coalesceMs,
        enabled,
        maxHistory,
        storageKey,
      ],
    )

  /**
   * 撤销到上一个历史快照。
   */
  const undo = useCallback(() => {
    setRecord((previous) => {
      if (previous.cursor <= 0) {
        return previous
      }

      const nextCursor =
        previous.cursor - 1

      const nextRecord: ProtectedDraftRecord =
        {
          ...previous,
          value:
            previous.history[nextCursor] || '',
          cursor: nextCursor,
          updatedAt: Date.now(),
        }

      writeDraftRecord(
        storageKey,
        nextRecord,
        enabled,
      )

      lastEditAtRef.current = 0

      return nextRecord
    })
  }, [enabled, storageKey])

  /**
   * 重做到下一个历史快照。
   */
  const redo = useCallback(() => {
    setRecord((previous) => {
      if (
        previous.cursor >=
        previous.history.length - 1
      ) {
        return previous
      }

      const nextCursor =
        previous.cursor + 1

      const nextRecord: ProtectedDraftRecord =
        {
          ...previous,
          value:
            previous.history[nextCursor] || '',
          cursor: nextCursor,
          updatedAt: Date.now(),
        }

      writeDraftRecord(
        storageKey,
        nextRecord,
        enabled,
      )

      lastEditAtRef.current = 0

      return nextRecord
    })
  }, [enabled, storageKey])

  /**
   * 提交后清空输入框，但保留一个空值历史节点。
   *
   * 用户可以在空输入框按Ctrl/Command+Z恢复刚提交的文字。
   */
  const commit = useCallback(() => {
    setRecord((previous) => {
      if (previous.value === '') {
        return previous
      }

      const activeHistory =
        previous.history.slice(
          0,
          previous.cursor + 1,
        )

      const trimmed = trimHistory(
        [...activeHistory, ''],
        maxHistory,
      )

      const nextRecord: ProtectedDraftRecord =
        {
          version:
            PROTECTED_DRAFT_VERSION,
          value: '',
          history: trimmed.history,
          cursor: trimmed.cursor,
          updatedAt: Date.now(),
        }

      writeDraftRecord(
        storageKey,
        nextRecord,
        enabled,
      )

      lastEditAtRef.current = 0

      return nextRecord
    })
  }, [
    enabled,
    maxHistory,
    storageKey,
  ])

  /**
   * 完全删除当前草稿和全部历史。
   */
  const clear = useCallback(() => {
    const emptyRecord =
      createDraftRecord('')

    removeDraftRecord(
      storageKey,
      enabled,
    )

    setRecord(emptyRecord)
    lastEditAtRef.current = 0
  }, [enabled, storageKey])

  const canUndo = record.cursor > 0

  const canRedo =
    record.cursor <
    record.history.length - 1

  /**
   * 统一处理撤销和重做快捷键。
   */
  const handleKeyDown = useCallback(
    (
      event: ReactKeyboardEvent<HTMLElement>,
    ): boolean => {
      const key = event.key.toLowerCase()
      const hasPrimaryModifier =
        event.ctrlKey || event.metaKey

      if (
        hasPrimaryModifier &&
        key === 'z'
      ) {
        if (event.shiftKey) {
          if (!canRedo) return false

          event.preventDefault()
          redo()
          return true
        }

        if (!canUndo) return false

        event.preventDefault()
        undo()
        return true
      }

      if (
        event.ctrlKey &&
        key === 'y' &&
        canRedo
      ) {
        event.preventDefault()
        redo()
        return true
      }

      return false
    },
    [
      canRedo,
      canUndo,
      redo,
      undo,
    ],
  )

  return {
    value: record.value,
    setValue,
    canUndo,
    canRedo,
    undo,
    redo,
    commit,
    clear,
    handleKeyDown,
    storageKey,
    updatedAt: record.updatedAt,
  }
}

export default useProtectedDraft

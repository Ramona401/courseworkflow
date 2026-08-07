/**
 * useCoursewareComicPanelEditor.ts
 *
 * 知识点漫画单格编辑器状态与持久化逻辑：
 *   - 所有操作立即写入当前标签页受保护草稿；
 *   - 停止编辑约1.1秒后自动保存到服务器；
 *   - 拖动和缩放只在指针释放后形成一次保存；
 *   - 保存期间允许继续输入，完成后自动续存较新的修改；
 *   - 服务端无关字段导致版本变化时安全升级草稿基线；
 *   - 服务端覆盖层真实变化时保留本地草稿并提示冲突；
 *   - 手动保存作为网络失败或版本冲突后的明确重试入口。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'

import {
  updateCoursewareComicPanelOverlay,
} from '@/api/coursewares'

import type {
  CoursewareComicPanel,
  CoursewareComicProjectStatus,
  CoursewareComicTextStyle,
} from '@/api/coursewares'

import {
  useAuth,
} from '@/store/auth'

import {
  useProtectedDraft,
} from '@/hooks/useProtectedDraft'

import {
  coursewareComicOverlayDraftEquals,
  coursewareComicOverlayDraftHasVersionConflict,
  createCoursewareComicOverlayDraft,
  cycleCoursewareComicOverlayStyle,
  deleteCoursewareComicOverlayElement,
  duplicateCoursewareComicOverlayElement,
  fitCoursewareComicOverlayElementByID,
  parseCoursewareComicOverlayDraft,
  rebaseCoursewareComicOverlayDraft,
  serializeCoursewareComicOverlayDraft,
  updateCoursewareComicOverlayContent,
  updateCoursewareComicOverlayLayout,
  updateCoursewareComicOverlayTextStyle,
  updateCoursewareComicQuestion,
} from './coursewareComicEditorDraft'

import type {
  CoursewareComicOverlayDraft,
} from './coursewareComicEditorDraft'

import {
  coursewareComicEditorErrorMessage,
  validateCoursewareComicOverlayDocument,
} from './coursewareComicPanelEditorValidation'

const COURSEWARE_COMIC_REGENERATION_INSTRUCTION_MAX_RUNES = 1200
const COURSEWARE_COMIC_OVERLAY_AUTO_SAVE_DELAY_MS = 1100

export type CoursewareComicAutoSaveState =
  | 'saved'
  | 'pending'
  | 'saving'
  | 'invalid'
  | 'error'
  | 'conflict'

type CoursewareComicSaveReason = 'auto' | 'manual'

interface UseCoursewareComicPanelEditorParams {
  coursewareId: string
  projectId: string
  projectStatus: CoursewareComicProjectStatus
  panel: CoursewareComicPanel
  disabled: boolean
  regenerating: boolean
  syncing: boolean
  onPanelUpdated: (panel: CoursewareComicPanel) => void
}

function isCoursewareComicVersionConflictMessage(message: string): boolean {
  const normalized = message.toLowerCase()

  return (
    normalized.includes('冲突') ||
    normalized.includes('版本') ||
    normalized.includes('conflict') ||
    normalized.includes('409')
  )
}

export function useCoursewareComicPanelEditor({
  coursewareId,
  projectId,
  projectStatus,
  panel,
  disabled,
  regenerating,
  syncing,
  onPanelUpdated,
}: UseCoursewareComicPanelEditorParams) {
  const { user } = useAuth()

  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState('')
  const [autoSaveState, setAutoSaveState] =
    useState<CoursewareComicAutoSaveState>('saved')
  const [autoSaveMessage, setAutoSaveMessage] = useState('')
  const [selectedElementID, setSelectedElementID] = useState('')
  const [editingElementID, setEditingElementID] = useState('')

  useEffect(() => {
    setNotice('')
    setAutoSaveState('saved')
    setAutoSaveMessage('')
    setSelectedElementID('')
    setEditingElementID('')
  }, [panel.id])

  const serverOverlayDraft = useMemo(
    () => createCoursewareComicOverlayDraft(panel),
    [panel],
  )

  const initialOverlayValue = useMemo(
    () => serializeCoursewareComicOverlayDraft(
      createCoursewareComicOverlayDraft(panel),
    ),
    // 只随漫画格切换，服务器轮询不能覆盖当前标签页未提交内容。
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [panel.id],
  )

  const overlayProtected = useProtectedDraft({
    userId: user?.id,
    scope: 'courseware-comic-panel',
    resourceId: `${projectId}:${panel.id}`,
    field: 'text-overlay',
    initialValue: initialOverlayValue,
    maxHistory: 50,
  })

  const regenerationInstructionProtected = useProtectedDraft({
    userId: user?.id,
    scope: 'courseware-comic-panel',
    resourceId: `${projectId}:${panel.id}`,
    field: 'regeneration-instruction',
    initialValue: '',
    maxHistory: 30,
  })

  const overlayDraft = parseCoursewareComicOverlayDraft(
    overlayProtected.value,
    serverOverlayDraft,
  )

  const overlayDirty = !coursewareComicOverlayDraftEquals(
    overlayDraft,
    serverOverlayDraft,
  )

  const overlayVersionConflict =
    coursewareComicOverlayDraftHasVersionConflict(
      overlayDraft,
      serverOverlayDraft,
    )

  const overlayValidationError = overlayDirty
    ? validateCoursewareComicOverlayDocument(
        overlayDraft.overlayDocument,
      )
    : ''

  /*
   * 保存状态不能禁用画布，否则每次自动保存都会打断输入。
   * 只有外部业务锁、重画和同步期间才冻结编辑。
   */
  const editorDisabled = disabled || regenerating || syncing

  const panelRef = useRef(panel)
  const serverOverlayDraftRef = useRef(serverOverlayDraft)
  const overlayValueRef = useRef(overlayProtected.value)
  const onPanelUpdatedRef = useRef(onPanelUpdated)
  const blockedRef = useRef({ disabled, regenerating, syncing })
  const saveInFlightRef = useRef(false)

  panelRef.current = panel
  serverOverlayDraftRef.current = serverOverlayDraft
  overlayValueRef.current = overlayProtected.value
  onPanelUpdatedRef.current = onPanelUpdated
  blockedRef.current = { disabled, regenerating, syncing }

  /**
   * 服务器只更新了图片或状态等无关字段时，版本会增长但覆盖层指纹相同。
   * 此时只升级本地草稿基线，不改变教师的任何内容。
   */
  useEffect(() => {
    if (
      overlayDraft.sourceVersion === serverOverlayDraft.sourceVersion ||
      coursewareComicOverlayDraftHasVersionConflict(
        overlayDraft,
        serverOverlayDraft,
      )
    ) {
      return
    }

    overlayProtected.setValue(previous => {
      const current = parseCoursewareComicOverlayDraft(
        previous,
        serverOverlayDraft,
      )

      if (
        current.sourceVersion === serverOverlayDraft.sourceVersion ||
        coursewareComicOverlayDraftHasVersionConflict(
          current,
          serverOverlayDraft,
        )
      ) {
        return previous
      }

      return serializeCoursewareComicOverlayDraft(
        rebaseCoursewareComicOverlayDraft(
          current,
          serverOverlayDraft,
        ),
      )
    })
  }, [
    overlayDraft.sourceFingerprint,
    overlayDraft.sourceVersion,
    overlayProtected.setValue,
    serverOverlayDraft.sourceFingerprint,
    serverOverlayDraft.sourceVersion,
  ])

  const regenerationInstruction =
    regenerationInstructionProtected.value

  const normalizedRegenerationInstruction =
    regenerationInstruction.trim()

  const regenerationInstructionLength =
    Array.from(regenerationInstruction).length

  const handleRegenerationInstructionChange = (value: string) => {
    regenerationInstructionProtected.setValue(
      Array.from(value)
        .slice(0, COURSEWARE_COMIC_REGENERATION_INSTRUCTION_MAX_RUNES)
        .join(''),
    )
  }

  /**
   * 任何编辑先立即进入sessionStorage；服务端版本和内容指纹始终保留
   * 建立编辑分支时的基线，不能被当前轮询版本静默覆盖。
   */
  const setOverlayPatch = (
    patch: Partial<CoursewareComicOverlayDraft>,
  ) => {
    setAutoSaveState('pending')
    setAutoSaveMessage('修改已安全保存到当前标签页，等待自动同步。')

    overlayProtected.setValue(previous =>
      serializeCoursewareComicOverlayDraft({
        ...parseCoursewareComicOverlayDraft(
          previous,
          serverOverlayDraftRef.current,
        ),
        ...patch,
      }),
    )
  }

  const handleContentChange = (
    elementID: string,
    content: string,
  ) => {
    const element = overlayDraft.overlayDocument.elements.find(
      item => item.id === elementID,
    )

    setOverlayPatch({
      narrationText: element?.type === 'narration'
        ? content
        : overlayDraft.narrationText,
      overlayDocument: updateCoursewareComicOverlayContent(
        overlayDraft.overlayDocument,
        elementID,
        content,
      ),
    })
  }

  const handleQuestionTextChange = (
    elementID: string,
    field: 'question' | 'explanation',
    value: string,
  ) => {
    setOverlayPatch({
      overlayDocument: updateCoursewareComicQuestion(
        overlayDraft.overlayDocument,
        elementID,
        question => ({
          ...question,
          [field]: value,
        }),
      ),
    })
  }

  const handleQuestionOptionsChange = (
    elementID: string,
    value: string,
  ) => {
    const options = value
      .split('\n')
      .map(item => item.trim())
      .filter(Boolean)

    setOverlayPatch({
      overlayDocument: updateCoursewareComicQuestion(
        overlayDraft.overlayDocument,
        elementID,
        question => ({
          ...question,
          options,
          answer_index: Math.min(
            question.answer_index,
            Math.max(options.length - 1, 0),
          ),
        }),
      ),
    })
  }

  const handleQuestionAnswerChange = (
    elementID: string,
    answerIndex: number,
  ) => {
    setOverlayPatch({
      overlayDocument: updateCoursewareComicQuestion(
        overlayDraft.overlayDocument,
        elementID,
        question => ({
          ...question,
          answer_index: answerIndex,
        }),
      ),
    })
  }

  const handleLayoutChange = (
    elementID: string,
    patch: {
      x?: number
      y?: number
      width?: number
      height?: number
      tail_type?: 'auto' | 'manual'
      tail_origin_x?: number
      tail_origin_y?: number
      tail_target_x?: number
      tail_target_y?: number
    },
  ) => {
    setOverlayPatch({
      overlayDocument: updateCoursewareComicOverlayLayout(
        overlayDraft.overlayDocument,
        elementID,
        patch,
      ),
    })
  }

  const handleTextStyleChange = (
    elementID: string,
    patch: Partial<CoursewareComicTextStyle>,
  ) => {
    setOverlayPatch({
      overlayDocument: updateCoursewareComicOverlayTextStyle(
        overlayDraft.overlayDocument,
        elementID,
        patch,
      ),
    })
  }

  const handleCycleStyle = (
    elementID: string,
    styleID = '',
  ) => {
    setOverlayPatch({
      overlayDocument: cycleCoursewareComicOverlayStyle(
        overlayDraft.overlayDocument,
        elementID,
        styleID,
      ),
    })
  }

  const handleAutoFit = (elementID: string) => {
    const updated = fitCoursewareComicOverlayElementByID(
      overlayDraft.overlayDocument,
      elementID,
    )

    const currentElement = overlayDraft.overlayDocument.elements.find(
      element => element.id === elementID,
    )
    const updatedElement = updated.elements.find(
      element => element.id === elementID,
    )

    const geometryChanged =
      !currentElement ||
      !updatedElement ||
      Math.abs(currentElement.x - updatedElement.x) > 0.0005 ||
      Math.abs(currentElement.y - updatedElement.y) > 0.0005 ||
      Math.abs(currentElement.width - updatedElement.width) > 0.0005 ||
      Math.abs(currentElement.height - updatedElement.height) > 0.0005 ||
      currentElement.text_style.font_size !==
        updatedElement.text_style.font_size

    setOverlayPatch({ overlayDocument: updated })

    setNotice(
      geometryChanged
        ? '✅ 已按当前文字重新适配，并保留正常呼吸空间。'
        : 'ℹ️ 当前元素已经是自动适配尺寸。',
    )
  }

  const handleDuplicate = (elementID: string) => {
    const updated = duplicateCoursewareComicOverlayElement(
      overlayDraft.overlayDocument,
      elementID,
    )

    if (
      updated.elements.length ===
      overlayDraft.overlayDocument.elements.length
    ) {
      setNotice('⚠️ 每格最多保留8个文字或题目元素。')
      return
    }

    const created = updated.elements[updated.elements.length - 1]

    if (!created) {
      setNotice('❌ 元素复制失败。')
      return
    }

    setOverlayPatch({ overlayDocument: updated })
    setSelectedElementID(created.id)
  }

  const handleDelete = (elementID: string) => {
    if (overlayDraft.overlayDocument.elements.length <= 1) {
      setNotice('⚠️ 当前格至少需要保留一个文字或题目元素。')
      return
    }

    setOverlayPatch({
      overlayDocument: deleteCoursewareComicOverlayElement(
        overlayDraft.overlayDocument,
        elementID,
      ),
    })

    setSelectedElementID('')
    setEditingElementID('')
  }

  /**
   * 自动保存遇到真实版本冲突时停止，绝不静默覆盖；
   * 手动保存代表教师明确重试，使用当前最新panel.version做CAS提交。
   * 请求期间出现的新编辑会换到新服务端基线，再由下一轮自动保存。
   */
  const saveOverlay = useCallback(
    async (reason: CoursewareComicSaveReason) => {
      if (saveInFlightRef.current) {
        if (reason === 'manual') {
          setAutoSaveMessage('当前保存完成后会继续处理最新修改。')
        }
        return
      }

      const blocked = blockedRef.current
      if (blocked.disabled || blocked.regenerating || blocked.syncing) {
        return
      }

      const currentPanel = panelRef.current
      const currentServerDraft = serverOverlayDraftRef.current
      const snapshot = parseCoursewareComicOverlayDraft(
        overlayValueRef.current,
        currentServerDraft,
      )

      if (
        coursewareComicOverlayDraftEquals(
          snapshot,
          currentServerDraft,
        )
      ) {
        setAutoSaveState('saved')
        setAutoSaveMessage('')
        return
      }

      const hasConflict =
        coursewareComicOverlayDraftHasVersionConflict(
          snapshot,
          currentServerDraft,
        )

      if (hasConflict && reason === 'auto') {
        setAutoSaveState('conflict')
        setAutoSaveMessage(
          '服务器覆盖层已有新版本；本地草稿仍完整保留，请核对后手动保存。',
        )
        return
      }

      const validationError =
        validateCoursewareComicOverlayDocument(
          snapshot.overlayDocument,
        )

      if (validationError) {
        setAutoSaveState('invalid')
        setAutoSaveMessage(validationError)

        if (reason === 'manual') {
          setNotice(`⚠️ ${validationError}`)
        }
        return
      }

      saveInFlightRef.current = true
      setSaving(true)
      setAutoSaveState('saving')
      setAutoSaveMessage(
        reason === 'manual'
          ? '正在保存本地草稿…'
          : '正在自动同步到服务器…',
      )

      /*
       * 自动保存不能结束正在输入的textarea；
       * 只有教师点击手动保存时才主动退出内联编辑。
       */
      if (reason === 'manual') {
        setEditingElementID('')
        setNotice('⏳ 正在保存画布排版与文字…')
      }

      try {
        const updated = await updateCoursewareComicPanelOverlay(
          coursewareId,
          projectId,
          currentPanel.id,
          {
            expected_version: currentPanel.version,
            narration_text: snapshot.narrationText.trim(),
            overlay_document: snapshot.overlayDocument,
          },
        )

        const updatedServerDraft =
          createCoursewareComicOverlayDraft(updated)

        panelRef.current = updated
        serverOverlayDraftRef.current = updatedServerDraft

        overlayProtected.setValue(previous => {
          const latest = parseCoursewareComicOverlayDraft(
            previous,
            currentServerDraft,
          )

          const next = coursewareComicOverlayDraftEquals(
            latest,
            snapshot,
          )
            ? updatedServerDraft
            : rebaseCoursewareComicOverlayDraft(
                latest,
                updatedServerDraft,
              )

          return serializeCoursewareComicOverlayDraft(next)
        })

        onPanelUpdatedRef.current(updated)
        setAutoSaveState('saved')
        setAutoSaveMessage(
          reason === 'manual'
            ? '当前格已保存。'
            : '最近一次修改已自动同步。',
        )

        if (reason === 'manual') {
          setNotice('✅ 当前格文字和画布排版已保存。')
        }
      } catch (error) {
        const message = coursewareComicEditorErrorMessage(
          error,
          '覆盖层保存失败',
        )

        if (isCoursewareComicVersionConflictMessage(message)) {
          setAutoSaveState('conflict')
          setAutoSaveMessage(
            '服务器版本已变化；本地草稿仍完整保留，请刷新核对或再次手动保存。',
          )
        } else {
          setAutoSaveState('error')
          setAutoSaveMessage(
            `自动保存失败：${message}。本地草稿仍完整保留。`,
          )
        }

        if (reason === 'manual') {
          setNotice(`❌ ${message}`)
        }
      } finally {
        saveInFlightRef.current = false
        setSaving(false)
      }
    },
    [
      coursewareId,
      overlayProtected.setValue,
      projectId,
    ],
  )

  /**
   * 文字输入、样式调整和指针释放后的最终布局都会重置计时器；
   * 无效草稿、真实版本冲突和外部业务锁期间只保留本地草稿。
   */
  useEffect(() => {
    if (!overlayDirty) {
      if (!saving) {
        setAutoSaveState('saved')
        setAutoSaveMessage('')
      }
      return
    }

    if (overlayValidationError) {
      setAutoSaveState('invalid')
      setAutoSaveMessage(
        `${overlayValidationError} 修改已保留在当前标签页。`,
      )
      return
    }

    if (overlayVersionConflict) {
      setAutoSaveState('conflict')
      setAutoSaveMessage(
        '服务器覆盖层已有新版本；本地草稿仍完整保留，请核对后手动保存。',
      )
      return
    }

    if (disabled || regenerating || syncing) {
      setAutoSaveState('pending')
      setAutoSaveMessage(
        '修改已保留在当前标签页，业务操作结束后自动同步。',
      )
      return
    }

    if (saving) {
      return
    }

    setAutoSaveState('pending')
    setAutoSaveMessage(
      '修改已安全保存到当前标签页，等待自动同步。',
    )

    const timer = window.setTimeout(() => {
      void saveOverlay('auto')
    }, COURSEWARE_COMIC_OVERLAY_AUTO_SAVE_DELAY_MS)

    return () => {
      window.clearTimeout(timer)
    }
  }, [
    disabled,
    overlayDirty,
    overlayProtected.updatedAt,
    overlayValidationError,
    overlayVersionConflict,
    regenerating,
    saveOverlay,
    saving,
    syncing,
  ])

  /**
   * 即使sessionStorage已保护草稿，离开页面时仍提醒正在保存或未同步状态。
   */
  useEffect(() => {
    if (!overlayDirty && !saving) {
      return
    }

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ''
    }

    window.addEventListener('beforeunload', handleBeforeUnload)

    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload)
    }
  }, [overlayDirty, saving])

  const handleSaveOverlay = useCallback(
    () => saveOverlay('manual'),
    [saveOverlay],
  )

  const canRegenerate =
    !editorDisabled &&
    !saving &&
    !overlayDirty &&
    normalizedRegenerationInstruction !== '' &&
    projectStatus !== 'generating' &&
    projectStatus !== 'archived'

  const canSync =
    !editorDisabled &&
    !saving &&
    !overlayDirty &&
    projectStatus === 'inserted' &&
    panel.status === 'generated' &&
    Boolean(panel.current_asset_id)

  return {
    saving,
    notice,
    autoSaveState,
    autoSaveMessage,
    overlayVersionConflict,
    selectedElementID,
    editingElementID,
    overlayDocument: overlayDraft.overlayDocument,
    overlayDirty,
    editorDisabled,
    canRegenerate,
    canSync,

    regenerationInstruction,
    normalizedRegenerationInstruction,
    regenerationInstructionLength,
    regenerationInstructionMaxLength:
      COURSEWARE_COMIC_REGENERATION_INSTRUCTION_MAX_RUNES,
    handleRegenerationInstructionChange,

    canUndo: overlayProtected.canUndo,
    canRedo: overlayProtected.canRedo,
    undo: overlayProtected.undo,
    redo: overlayProtected.redo,
    handleKeyDown: overlayProtected.handleKeyDown,

    selectElement: setSelectedElementID,

    beginEditing: (elementID: string) => {
      setSelectedElementID(elementID)
      setEditingElementID(elementID)
    },

    endEditing: () => {
      setEditingElementID('')
    },

    handleContentChange,
    handleQuestionTextChange,
    handleQuestionOptionsChange,
    handleQuestionAnswerChange,
    handleLayoutChange,
    handleTextStyleChange,
    handleCycleStyle,
    handleAutoFit,
    handleDuplicate,
    handleDelete,
    handleSaveOverlay,
  }
}

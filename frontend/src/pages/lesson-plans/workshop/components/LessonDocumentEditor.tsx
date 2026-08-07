/**
 * LessonDocumentEditor.tsx — 备课工坊共享教案正文编辑器。
 *
 * 对话模式和专家模式共用本组件：
 *   - 完整Markdown正文预览和手动编辑；
 *   - 图片上传、拖拽和粘贴；
 *   - 正文版本历史和恢复；
 *   - 教案目录、章节滚动定位和当前章节高亮；
 *   - 章节AI修改预览与原子应用。
 *
 * 正文唯一事实源仍是父页面传入的content。
 * AI段落应用完成后通过onSectionRewriteApplied通知父页面同步正文和版本。
 */

import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type ClipboardEvent,
  type DragEvent,
  type KeyboardEvent,
  type ReactNode,
} from 'react'
import { uploadAsset, validateImageFile } from '@/api/lesson-plan-assets'
import type { LessonPlanContentRestoreResponse } from '@/api/lesson-plan-versions'
import type {
  LessonPlanSectionRewriteApplyResponse,
} from '@/api/lesson-plan-section-rewrite'
import LessonVersionHistoryPanel from './LessonVersionHistoryPanel'
import LessonDocumentOutline from './LessonDocumentOutline'
import LessonSectionAIEditor from './LessonSectionAIEditor'
import LessonDocumentToolbar from './LessonDocumentToolbar'
import LessonDocumentEditPane from './LessonDocumentEditPane'
import LessonDocumentPreview from './LessonDocumentPreview'
import {
  parseLessonDocumentStructure,
  type LessonDocumentSection,
} from './lessonDocumentStructure'
import {
  useLessonDocumentImageRemoval,
} from './useLessonDocumentImageRemoval'

interface LessonDocumentEditorProps {
  /** 当前数据库或SSE同步到前端的正式正文 */
  content: string
  /** 当前教案ID，用于图片、版本和段落AI修改 */
  planID: string
  /** 当前数据库正式版本号 */
  currentVersion: number
  /** AI生成、评审或状态锁定时禁用修改 */
  disabled?: boolean
  /** 禁用原因 */
  disabledReason?: string
  /** 保存完整Markdown正文 */
  onSave: (nextContent: string) => Promise<void>
  /** 历史版本恢复成功后同步父页面 */
  onRestored: (result: LessonPlanContentRestoreResponse) => void
  /** AI段落修改应用成功后同步父页面正文和版本 */
  onSectionRewriteApplied?: (
    result: LessonPlanSectionRewriteApplyResponse,
  ) => void | Promise<void>
  /** 无正文时由业务页面提供的说明 */
  emptyState?: ReactNode
  /** 右侧窄画布使用紧凑尺寸 */
  compact?: boolean
}

export default function LessonDocumentEditor({
  content,
  planID,
  currentVersion,
  disabled = false,
  disabledReason = '',
  onSave,
  onRestored,
  onSectionRewriteApplied,
  emptyState,
  compact = false,
}: LessonDocumentEditorProps) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [outlineOpen, setOutlineOpen] = useState(false)
  const [activeSectionID, setActiveSectionID] = useState('')
  const [rewriteSectionID, setRewriteSectionID] = useState('')
  const [message, setMessage] = useState<{
    text: string
    type: 'success' | 'error' | 'info'
  } | null>(null)

  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const sectionRefs = useRef(new Map<string, HTMLDivElement>())
  const sourceAtEditStartRef = useRef('')
  const messageTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const hasContent = Boolean(content && content.trim())
  const hasDraft = Boolean(draft && draft.trim())
  const changed = draft.trim() !== (content || '').trim()
  const externalChanged =
    editing &&
    (content || '') !== sourceAtEditStartRef.current

  const structure = useMemo(
    () => parseLessonDocumentStructure(content || ''),
    [content],
  )

  const firstSectionID = structure.sections[0]?.id || ''
  const effectiveActiveSectionID =
    structure.sections.some(section => section.id === activeSectionID)
      ? activeSectionID
      : firstSectionID

  const activeSection =
    structure.sections.find(
      section => section.id === effectiveActiveSectionID,
    ) || structure.sections[0] || null

  const rewriteSection =
    structure.sections.find(
      section => section.id === rewriteSectionID,
    ) || null

  const rewriteAvailable = Boolean(
    onSectionRewriteApplied &&
    planID &&
    hasContent,
  )

  const showMessage = (
    text: string,
    type: 'success' | 'error' | 'info' = 'info',
  ) => {
    setMessage({ text, type })

    if (messageTimerRef.current) {
      clearTimeout(messageTimerRef.current)
    }

    messageTimerRef.current = setTimeout(
      () => setMessage(null),
      4000,
    )
  }

  const {
    removingImageKey,
    removePreviewImage,
  } = useLessonDocumentImageRemoval({
    content,
    planID,
    disabled,
    disabledReason,
    saving,
    uploading,
    onSave,
    showMessage,
  })

  const operationDisabled =
    Boolean(removingImageKey)

  const rewriteDisabled =
    !rewriteAvailable ||
    disabled ||
    editing ||
    saving ||
    uploading ||
    operationDisabled

  useEffect(() => {
    return () => {
      if (messageTimerRef.current) {
        clearTimeout(messageTimerRef.current)
      }
    }
  }, [])

  const enterEdit = () => {
    if (
      disabled ||
      operationDisabled
    ) {
      showMessage(
        operationDisabled
          ? '当前正在移除图片，请稍候'
          : disabledReason ||
            '当前暂不可编辑正文',
        'info',
      )
      return
    }

    setOutlineOpen(false)
    setRewriteSectionID('')
    setDraft(content || '')
    sourceAtEditStartRef.current = content || ''
    setEditing(true)

    setTimeout(
      () => textareaRef.current?.focus(),
      0,
    )
  }

  const cancelEdit = () => {
    setDraft('')
    sourceAtEditStartRef.current = content || ''
    setEditing(false)
  }

  const loadLatestContent = () => {
    setDraft(content || '')
    sourceAtEditStartRef.current = content || ''

    showMessage(
      '已载入AI更新后的最新正文，可继续编辑',
      'info',
    )

    setTimeout(
      () => textareaRef.current?.focus(),
      0,
    )
  }

  const insertAtCursor = (insertText: string) => {
    const textarea = textareaRef.current

    if (!textarea) {
      setDraft(previous => {
        const prefix =
          previous && !previous.endsWith('\n')
            ? '\n'
            : ''

        return `${previous}${prefix}${insertText}\n`
      })
      return
    }

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const before = draft.slice(0, start)
    const after = draft.slice(end)

    const prefix =
      before && !before.endsWith('\n')
        ? '\n'
        : ''
    const suffix =
      after && !after.startsWith('\n')
        ? '\n'
        : ''

    const inserted = `${prefix}${insertText}${suffix}`
    const nextDraft = before + inserted + after

    setDraft(nextDraft)

    setTimeout(() => {
      const nextPosition = start + inserted.length
      textarea.focus()
      textarea.setSelectionRange(
        nextPosition,
        nextPosition,
      )
    }, 0)
  }

  const handleImageUpload = async (file: File) => {
    if (
      disabled ||
      saving ||
      uploading ||
      operationDisabled
    ) {
      return
    }

    if (!planID) {
      showMessage(
        '教案ID缺失，暂时无法上传图片',
        'error',
      )
      return
    }

    const validationError = validateImageFile(file)
    if (validationError) {
      showMessage(validationError, 'error')
      return
    }

    setUploading(true)

    try {
      const altText =
        file.name.replace(/\.[^.]+$/, '') ||
        '教案图片'

      const response = await uploadAsset(
        planID,
        file,
        altText,
      )

      insertAtCursor(response.markdown)
      showMessage(
        '图片已上传并插入正文',
        'success',
      )
    } catch (error) {
      const text = error instanceof Error
        ? error.message
        : '图片上传失败'

      showMessage(
        `图片上传失败：${text}`,
        'error',
      )
    } finally {
      setUploading(false)
    }
  }

  const handleDrop = (
    event: DragEvent<HTMLTextAreaElement>,
  ) => {
    event.preventDefault()

    const file = event.dataTransfer.files?.[0]
    if (file && file.type.startsWith('image/')) {
      void handleImageUpload(file)
    }
  }

  const handlePaste = (
    event: ClipboardEvent<HTMLTextAreaElement>,
  ) => {
    const items = event.clipboardData?.items
    if (!items) return

    for (
      let index = 0;
      index < items.length;
      index += 1
    ) {
      const item = items[index]
      if (!item.type.startsWith('image/')) {
        continue
      }

      const file = item.getAsFile()
      if (!file) continue

      event.preventDefault()
      void handleImageUpload(file)
      return
    }
  }

  const handleSave = async () => {
    const nextContent = draft.trim()

    if (!nextContent) {
      showMessage(
        '教案正文不能为空',
        'error',
      )
      return
    }

    if (externalChanged) {
      showMessage(
        'AI已更新正文，请先载入最新版本再保存，避免覆盖新内容',
        'error',
      )
      return
    }

    if (!changed) {
      setEditing(false)
      setDraft('')
      showMessage(
        '正文没有变化',
        'info',
      )
      return
    }

    setSaving(true)

    try {
      await onSave(nextContent)

      sourceAtEditStartRef.current = nextContent
      setDraft('')
      setEditing(false)

      showMessage(
        '教案正文已保存',
        'success',
      )
    } catch (error) {
      const text = error instanceof Error
        ? error.message
        : '保存失败'

      showMessage(
        `保存失败：${text}`,
        'error',
      )
    } finally {
      setSaving(false)
    }
  }

  const handleEditorKeyDown = (
    event: KeyboardEvent<HTMLTextAreaElement>,
  ) => {
    if (
      (event.ctrlKey || event.metaKey) &&
      event.key.toLowerCase() === 's'
    ) {
      event.preventDefault()
      void handleSave()
    }
  }

  const scrollToSection = (
    section: LessonDocumentSection,
  ) => {
    setActiveSectionID(section.id)

    const element = sectionRefs.current.get(section.id)
    element?.scrollIntoView({
      behavior: 'smooth',
      block: 'start',
    })

    if (compact) {
      setOutlineOpen(false)
    }
  }

  const openSectionRewrite = (
    section: LessonDocumentSection,
  ) => {
    if (!rewriteAvailable) {
      showMessage(
        '页面尚未接入段落修改后的正文同步',
        'info',
      )
      return
    }

    if (disabled) {
      showMessage(
        disabledReason || '当前暂不能使用AI修改',
        'info',
      )
      return
    }

    if (
      editing ||
      saving ||
      uploading ||
      operationDisabled
    ) {
      showMessage(
        operationDisabled
          ? '当前正在移除图片，请稍候'
          : '请先结束全文编辑或图片上传',
        'info',
      )
      return
    }

    setOutlineOpen(false)
    setActiveSectionID(section.id)
    setRewriteSectionID(section.id)
  }

  const charCount = draft
    .replace(/[#*`>_~[\]()!-]/g, '')
    .replace(/\s+/g, '')
    .length

  const messageStyle = message?.type === 'error'
    ? {
        background: 'rgba(239,68,68,0.08)',
        border: '1px solid rgba(239,68,68,0.24)',
        color: '#DC2626',
      }
    : message?.type === 'success'
      ? {
          background: 'rgba(16,185,129,0.08)',
          border: '1px solid rgba(16,185,129,0.24)',
          color: '#047857',
        }
      : {
          background: 'rgba(79,123,232,0.07)',
          border: '1px solid rgba(79,123,232,0.18)',
          color: '#365FB8',
        }

  return (
    <div style={{
      height: '100%',
      display: 'flex',
      flexDirection: 'column',
      minHeight: 0,
      background: '#FFFFFF',
    }}>
      <LessonDocumentToolbar
        compact={compact}
        editing={editing}
        hasContent={hasContent}
        disabled={disabled}
        disabledReason={
          disabledReason
        }
        operationDisabled={
          operationDisabled
        }
        operationDisabledReason="当前正在移除图片，请稍候"
        planID={planID}
        currentVersion={
          currentVersion
        }
        saving={saving}
        uploading={uploading}
        externalChanged={
          externalChanged
        }
        hasDraft={hasDraft}
        rewriteDisabled={
          rewriteDisabled
        }
        activeSection={
          activeSection
        }
        onOpenOutline={() =>
          setOutlineOpen(true)
        }
        onOpenSectionRewrite={
          openSectionRewrite
        }
        onOpenHistory={() =>
          setHistoryOpen(true)
        }
        onEnterEdit={enterEdit}
        onChooseImage={() =>
          fileInputRef
            .current
            ?.click()
        }
        onCancelEdit={
          cancelEdit
        }
        onSave={() =>
          void handleSave()
        }
      />

      {message && (
        <div style={{
          ...messageStyle,
          margin: '8px 12px 0',
          padding: '7px 10px',
          borderRadius: '7px',
          fontSize: '12px',
          lineHeight: 1.5,
          flexShrink: 0,
        }}>
          {message.text}
        </div>
      )}

      {editing && externalChanged && (
        <div style={{
          margin: '8px 12px 0',
          padding: '9px 11px',
          borderRadius: '8px',
          background: '#FFF7ED',
          border: '1px solid #FDBA74',
          color: '#9A3412',
          fontSize: '12px',
          lineHeight: 1.6,
          flexShrink: 0,
        }}>
          <div style={{
            fontWeight: 700,
            marginBottom: '5px',
          }}>
            ⚠️ 编辑期间AI更新了教案正文
          </div>

          <div>
            为避免用旧草稿覆盖AI刚生成的新内容，当前已暂停保存。
          </div>

          <button
            type="button"
            onClick={loadLatestContent}
            style={{
              marginTop: '7px',
              padding: '5px 10px',
              borderRadius: '6px',
              border: '1px solid #F97316',
              background: '#FFFFFF',
              color: '#C2410C',
              fontSize: '12px',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            载入AI最新正文
          </button>
        </div>
      )}

      <div style={{
        flex: 1,
        minHeight: 0,
        position: 'relative',
        overflow: 'hidden',
      }}>
        {editing ? (
          <LessonDocumentEditPane
            compact={compact}
            textareaRef={
              textareaRef
            }
            draft={draft}
            saving={saving}
            uploading={
              uploading
            }
            disabled={disabled}
            externalChanged={
              externalChanged
            }
            charCount={
              charCount
            }
            onDraftChange={
              setDraft
            }
            onDrop={handleDrop}
            onPaste={
              handlePaste
            }
            onKeyDown={
              handleEditorKeyDown
            }
          />
        ) : hasContent ? (
          <LessonDocumentPreview
            content={content}
            structure={structure}
            activeSectionID={
              effectiveActiveSectionID
            }
            compact={compact}
            rewriteDisabled={
              rewriteDisabled
            }
            disabledReason={
              disabledReason
            }
            sectionRefs={
              sectionRefs
            }
            onActiveSectionChange={
              setActiveSectionID
            }
            onOpenSectionRewrite={
              openSectionRewrite
            }
            imageActionDisabled={
              disabled ||
              saving ||
              uploading ||
              operationDisabled
            }
            removingImageKey={
              removingImageKey
            }
            onRemoveImage={(
              image,
              rangeStart,
              rangeEnd,
            ) => {
              void removePreviewImage(
                image,
                rangeStart,
                rangeEnd,
              )
            }}
          />
        ) : (
          <div style={{
            height: '100%',
            overflowY: 'auto',
            padding: compact
              ? '12px 14px'
              : '16px 20px',
            boxSizing: 'border-box',
          }}>
            {emptyState || (
              <div style={{
                height: '100%',
                minHeight: '260px',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                textAlign: 'center',
                color: '#9CA3AF',
                lineHeight: 1.8,
              }}>
                <div style={{
                  fontSize: '34px',
                  marginBottom: '12px',
                }}>
                  📝
                </div>
                <div style={{
                  fontSize: '13px',
                }}>
                  当前还没有教案正文
                </div>
              </div>
            )}
          </div>
        )}

        {outlineOpen && !editing && hasContent && (
          <div style={{
            position: 'absolute',
            inset: 0,
            zIndex: 20,
            display: 'flex',
            background: 'rgba(17,24,39,0.12)',
          }}>
            <div style={{
              width: compact
                ? '82%'
                : '240px',
              maxWidth: '300px',
              height: '100%',
              boxShadow:
                '8px 0 24px rgba(17,24,39,0.12)',
            }}>
              <LessonDocumentOutline
                sections={structure.sections}
                activeSectionID={
                  effectiveActiveSectionID
                }
                disabled={rewriteDisabled}
                compact={compact}
                onSelect={scrollToSection}
                onRewrite={openSectionRewrite}
                onClose={() =>
                  setOutlineOpen(false)
                }
              />
            </div>

            <button
              type="button"
              onClick={() =>
                setOutlineOpen(false)
              }
              aria-label="关闭目录遮罩"
              style={{
                flex: 1,
                border: 'none',
                background: 'transparent',
                cursor: 'pointer',
              }}
            />
          </div>
        )}

        {rewriteSection && !editing && (
          <div style={{
            position: 'absolute',
            inset: 0,
            zIndex: 30,
            background: '#FFFFFF',
          }}>
            <LessonSectionAIEditor
              key={rewriteSection.id}
              planID={planID}
              currentVersion={currentVersion}
              section={rewriteSection}
              disabled={
                disabled ||
                saving ||
                uploading ||
                operationDisabled
              }
              onApplied={async result => {
                await onSectionRewriteApplied?.(
                  result,
                )
              }}
              onClose={() =>
                setRewriteSectionID('')
              }
            />
          </div>
        )}
      </div>

      <LessonVersionHistoryPanel
        open={historyOpen}
        planID={planID}
        currentContent={content}
        currentVersion={currentVersion}
        restoreDisabled={
          disabled ||
          saving ||
          uploading ||
          operationDisabled
        }
        restoreDisabledReason={
          disabled
            ? disabledReason ||
              '当前状态不允许恢复正文'
            : operationDisabled
              ? '当前正在移除图片，请稍后恢复'
              : saving || uploading
                ? '当前正在保存或上传图片，请稍后恢复'
                : ''
        }
        onClose={() =>
          setHistoryOpen(false)
        }
        onRestored={result => {
          setHistoryOpen(false)
          setEditing(false)
          setDraft('')
          setOutlineOpen(false)
          setRewriteSectionID('')
          sourceAtEditStartRef.current =
            result.content_markdown

          onRestored(result)

          showMessage(
            `已恢复历史v${result.restored_from_version}，当前为v${result.current_version}`,
            'success',
          )
        }}
      />

      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp,image/gif"
        style={{ display: 'none' }}
        onChange={event => {
          const file = event.target.files?.[0]
          if (file) {
            void handleImageUpload(file)
          }
          event.target.value = ''
        }}
      />
    </div>
  )
}

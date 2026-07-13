/**
 * LessonDocumentEditor.tsx — 备课工坊共享教案正文编辑器
 *
 * 目标：
 *   1. 对话模式和专家模式共用同一套正文编辑能力，避免两套实现继续分叉。
 *   2. 保留 Markdown 作为正文唯一存储格式，继续兼容详情页、Word导出和AI生成链路。
 *   3. 复用现有教案图片资产接口，支持点击、拖拽和粘贴图片。
 *   4. 编辑期间若AI推送了新正文，立即提示冲突并阻止旧稿直接覆盖新稿。
 *
 * 本组件只负责交互和编辑草稿：
 *   - 真实保存由父页面通过 onSave 调用 updateLessonPlan。
 *   - 保存成功后父页面同步 planContent，SSE协议和后端正文生成链路保持不变。
 */

import {
  useEffect,
  useRef,
  useState,
  type ClipboardEvent,
  type DragEvent,
  type KeyboardEvent,
  type ReactNode,
} from 'react'
import { renderMarkdown } from '@/pages/lesson-plans/plan-detail/components/planDetailConstants'
import { uploadAsset, validateImageFile } from '@/api/lesson-plan-assets'
import type { LessonPlanContentRestoreResponse } from '@/api/lesson-plan-versions'
import LessonVersionHistoryPanel from './LessonVersionHistoryPanel'

interface LessonDocumentEditorProps {
  /** 当前数据库/SSE同步到前端的正式正文 */
  content: string
  /** 当前教案ID，用于图片资产上传和版本查询 */
  planID: string
  /** 当前正式教案版本号 */
  currentVersion: number
  /** AI生成、评审或状态锁定时禁用人工编辑 */
  disabled?: boolean
  /** 禁用原因，展示给老师理解 */
  disabledReason?: string
  /** 保存完整Markdown正文 */
  onSave: (nextContent: string) => Promise<void>
  /** 历史版本恢复成功后同步父页面正式状态 */
  onRestored: (result: LessonPlanContentRestoreResponse) => void
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
  emptyState,
  compact = false,
}: LessonDocumentEditorProps) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(content || '')
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [externalChanged, setExternalChanged] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [message, setMessage] = useState<{
    text: string
    type: 'success' | 'error' | 'info'
  } | null>(null)

  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const sourceAtEditStartRef = useRef(content || '')
  const messageTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const hasContent = Boolean(content && content.trim())
  const hasDraft = Boolean(draft && draft.trim())
  const changed = draft.trim() !== (content || '').trim()

  const showMessage = (
    text: string,
    type: 'success' | 'error' | 'info' = 'info',
  ) => {
    setMessage({ text, type })
    if (messageTimerRef.current) clearTimeout(messageTimerRef.current)
    messageTimerRef.current = setTimeout(() => setMessage(null), 4000)
  }

  useEffect(() => {
    return () => {
      if (messageTimerRef.current) clearTimeout(messageTimerRef.current)
    }
  }, [])

  /**
   * 非编辑状态始终跟随外部正式正文。
   * 编辑状态下若正文被SSE或其它入口更新，则标记冲突，不静默覆盖老师草稿。
   */
  useEffect(() => {
    if (!editing) {
      setDraft(content || '')
      sourceAtEditStartRef.current = content || ''
      setExternalChanged(false)
      return
    }

    if ((content || '') !== sourceAtEditStartRef.current) {
      setExternalChanged(true)
    }
  }, [content, editing])

  const enterEdit = () => {
    if (disabled) {
      showMessage(disabledReason || '当前暂不可编辑正文', 'info')
      return
    }
    setDraft(content || '')
    sourceAtEditStartRef.current = content || ''
    setExternalChanged(false)
    setEditing(true)
    setTimeout(() => textareaRef.current?.focus(), 0)
  }

  const cancelEdit = () => {
    setDraft(content || '')
    sourceAtEditStartRef.current = content || ''
    setExternalChanged(false)
    setEditing(false)
  }

  const loadLatestContent = () => {
    setDraft(content || '')
    sourceAtEditStartRef.current = content || ''
    setExternalChanged(false)
    showMessage('已载入AI更新后的最新正文，可继续编辑', 'info')
    setTimeout(() => textareaRef.current?.focus(), 0)
  }

  const insertAtCursor = (insertText: string) => {
    const textarea = textareaRef.current

    if (!textarea) {
      setDraft(prev => {
        const prefix = prev && !prev.endsWith('\n') ? '\n' : ''
        return `${prev}${prefix}${insertText}\n`
      })
      return
    }

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const before = draft.slice(0, start)
    const after = draft.slice(end)

    const prefix = before && !before.endsWith('\n') ? '\n' : ''
    const suffix = after && !after.startsWith('\n') ? '\n' : ''
    const inserted = `${prefix}${insertText}${suffix}`
    const nextDraft = before + inserted + after

    setDraft(nextDraft)

    setTimeout(() => {
      const nextPosition = start + inserted.length
      textarea.focus()
      textarea.setSelectionRange(nextPosition, nextPosition)
    }, 0)
  }

  const handleImageUpload = async (file: File) => {
    if (disabled || saving || uploading) return

    if (!planID) {
      showMessage('教案ID缺失，暂时无法上传图片', 'error')
      return
    }

    const validationError = validateImageFile(file)
    if (validationError) {
      showMessage(validationError, 'error')
      return
    }

    setUploading(true)
    try {
      const altText = file.name.replace(/\.[^.]+$/, '') || '教案图片'
      const response = await uploadAsset(planID, file, altText)
      insertAtCursor(response.markdown)
      showMessage('图片已上传并插入正文', 'success')
    } catch (error) {
      const text = error instanceof Error ? error.message : '图片上传失败'
      showMessage(`图片上传失败：${text}`, 'error')
    } finally {
      setUploading(false)
    }
  }

  const handleDrop = (event: DragEvent<HTMLTextAreaElement>) => {
    event.preventDefault()
    const file = event.dataTransfer.files?.[0]
    if (file && file.type.startsWith('image/')) {
      void handleImageUpload(file)
    }
  }

  const handlePaste = (event: ClipboardEvent<HTMLTextAreaElement>) => {
    const items = event.clipboardData?.items
    if (!items) return

    for (let index = 0; index < items.length; index += 1) {
      const item = items[index]
      if (!item.type.startsWith('image/')) continue

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
      showMessage('教案正文不能为空', 'error')
      return
    }

    if (externalChanged) {
      showMessage('AI已更新正文，请先载入最新版本再保存，避免覆盖新内容', 'error')
      return
    }

    if (!changed) {
      setEditing(false)
      showMessage('正文没有变化', 'info')
      return
    }

    setSaving(true)
    try {
      await onSave(nextContent)
      sourceAtEditStartRef.current = nextContent
      setDraft(nextContent)
      setExternalChanged(false)
      setEditing(false)
      showMessage('教案正文已保存', 'success')
    } catch (error) {
      const text = error instanceof Error ? error.message : '保存失败'
      showMessage(`保存失败：${text}`, 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleEditorKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
      event.preventDefault()
      void handleSave()
    }
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
      {/* 编辑器工具栏 */}
      <div style={{
        padding: compact ? '9px 12px' : '11px 16px',
        borderBottom: '1px solid #F3F4F6',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: '10px',
        flexShrink: 0,
      }}>
        <div style={{ minWidth: 0 }}>
          <div style={{
            fontSize: '12px',
            fontWeight: 700,
            color: '#374151',
          }}>
            {editing ? '✏️ 正在编辑完整教案' : '📄 教案正文'}
          </div>
          {disabled && disabledReason && (
            <div style={{
              marginTop: '2px',
              fontSize: '10px',
              color: '#9CA3AF',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}>
              {disabledReason}
            </div>
          )}
        </div>

        {!editing ? (
          <div style={{ display: 'flex', gap: '6px', alignItems: 'center' }}>
            <button
              onClick={() => setHistoryOpen(true)}
              disabled={!planID}
              title="查看正文修改记录、对比差异或恢复历史版本"
              style={{
                padding: '5px 10px',
                borderRadius: '7px',
                border: '1px solid #D1D5DB',
                background: '#FFFFFF',
                color: '#6B7280',
                fontSize: '12px',
                fontWeight: 600,
                cursor: planID ? 'pointer' : 'not-allowed',
                whiteSpace: 'nowrap',
              }}
            >
              🕘 版本 v{currentVersion}
            </button>
            <button
              onClick={enterEdit}
              disabled={disabled || !planID}
              title={disabled ? disabledReason : '直接编辑完整教案正文'}
              style={{
                padding: '5px 11px',
                borderRadius: '7px',
                border: '1px solid #D1D5DB',
                background: disabled ? '#F3F4F6' : '#FFFFFF',
                color: disabled ? '#9CA3AF' : '#374151',
                fontSize: '12px',
                fontWeight: 600,
                cursor: disabled ? 'not-allowed' : 'pointer',
                whiteSpace: 'nowrap',
              }}
            >
              ✏️ {hasContent ? '编辑正文' : '手动填写'}
            </button>
          </div>
        ) : (
          <div style={{ display: 'flex', gap: '6px', alignItems: 'center' }}>
            <button
              onClick={() => fileInputRef.current?.click()}
              disabled={saving || uploading || disabled}
              title="支持点击选择、拖拽或粘贴图片"
              style={{
                padding: '5px 9px',
                borderRadius: '7px',
                border: '1px solid #D1D5DB',
                background: '#FFFFFF',
                color: uploading ? '#9CA3AF' : '#374151',
                fontSize: '12px',
                cursor: uploading ? 'wait' : 'pointer',
                whiteSpace: 'nowrap',
              }}
            >
              {uploading ? '⏳ 上传中' : '📷 插图'}
            </button>
            <button
              onClick={cancelEdit}
              disabled={saving}
              style={{
                padding: '5px 9px',
                borderRadius: '7px',
                border: '1px solid #E5E7EB',
                background: '#FFFFFF',
                color: '#6B7280',
                fontSize: '12px',
                cursor: saving ? 'not-allowed' : 'pointer',
              }}
            >
              取消
            </button>
            <button
              onClick={() => void handleSave()}
              disabled={
                saving ||
                uploading ||
                disabled ||
                externalChanged ||
                !hasDraft
              }
              style={{
                padding: '5px 11px',
                borderRadius: '7px',
                border: 'none',
                background:
                  saving ||
                  uploading ||
                  disabled ||
                  externalChanged ||
                  !hasDraft
                    ? '#E5E7EB'
                    : '#10B981',
                color:
                  saving ||
                  uploading ||
                  disabled ||
                  externalChanged ||
                  !hasDraft
                    ? '#9CA3AF'
                    : '#FFFFFF',
                fontSize: '12px',
                fontWeight: 700,
                cursor:
                  saving ||
                  uploading ||
                  disabled ||
                  externalChanged ||
                  !hasDraft
                    ? 'not-allowed'
                    : 'pointer',
              }}
            >
              {saving ? '保存中…' : '✓ 保存'}
            </button>
          </div>
        )}
      </div>

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
          <div style={{ fontWeight: 700, marginBottom: '5px' }}>
            ⚠️ 编辑期间AI更新了教案正文
          </div>
          <div>
            为避免用旧草稿覆盖AI刚生成的新内容，当前已暂停保存。
          </div>
          <button
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

      {/* 正文预览或全文编辑区 */}
      <div style={{
        flex: 1,
        minHeight: 0,
        overflow: editing ? 'hidden' : 'auto',
        padding: compact ? '12px 14px' : '16px 20px',
        boxSizing: 'border-box',
      }}>
        {editing ? (
          <div style={{
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
            minHeight: compact ? '300px' : '420px',
          }}>
            <textarea
              ref={textareaRef}
              value={draft}
              onChange={event => setDraft(event.target.value)}
              onDrop={handleDrop}
              onDragOver={event => event.preventDefault()}
              onPaste={handlePaste}
              onKeyDown={handleEditorKeyDown}
              disabled={saving || uploading || disabled}
              placeholder="在这里直接编辑完整教案正文，支持Markdown格式……"
              style={{
                width: '100%',
                flex: 1,
                minHeight: compact ? '300px' : '420px',
                padding: '12px 14px',
                boxSizing: 'border-box',
                border: `1px solid ${externalChanged ? '#F97316' : '#D1D5DB'}`,
                borderRadius: '9px',
                outline: 'none',
                resize: 'none',
                fontFamily: 'inherit',
                fontSize: compact ? '13px' : '14px',
                lineHeight: 1.75,
                color: '#1F2937',
                background: saving || uploading || disabled ? '#F9FAFB' : '#FFFFFF',
              }}
            />
            <div style={{
              marginTop: '7px',
              display: 'flex',
              justifyContent: 'space-between',
              gap: '10px',
              fontSize: '11px',
              color: '#9CA3AF',
              flexShrink: 0,
            }}>
              <span>
                支持Markdown · 拖拽/粘贴图片 · Ctrl+S保存
              </span>
              <span>约 {charCount} 字</span>
            </div>
          </div>
        ) : hasContent ? (
          <div style={{
            fontSize: compact ? '13px' : '14px',
            lineHeight: 1.85,
          }}>
            {renderMarkdown(content)}
          </div>
        ) : (
          emptyState || (
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
              <div style={{ fontSize: '34px', marginBottom: '12px' }}>📝</div>
              <div style={{ fontSize: '13px' }}>当前还没有教案正文</div>
            </div>
          )
        )}
      </div>

      <LessonVersionHistoryPanel
        open={historyOpen}
        planID={planID}
        currentContent={content}
        currentVersion={currentVersion}
        restoreDisabled={disabled || saving || uploading}
        restoreDisabledReason={
          disabled
            ? disabledReason || '当前状态不允许恢复正文'
            : saving || uploading
              ? '当前正在保存或上传图片，请稍后恢复'
              : ''
        }
        onClose={() => setHistoryOpen(false)}
        onRestored={(result) => {
          setHistoryOpen(false)
          setEditing(false)
          setDraft(result.content_markdown)
          sourceAtEditStartRef.current = result.content_markdown
          setExternalChanged(false)
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
          if (file) void handleImageUpload(file)
          event.target.value = ''
        }}
      />
    </div>
  )
}

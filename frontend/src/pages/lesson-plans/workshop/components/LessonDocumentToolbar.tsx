/**
 * LessonDocumentToolbar.tsx — 教案正文编辑器顶部工具栏。
 *
 * 负责目录、章节AI修改、版本历史、全文编辑、
 * 图片上传、取消和保存按钮。
 *
 * 业务状态与操作仍由父组件统一维护。
 */

import type {
  LessonDocumentSection,
} from './lessonDocumentStructure'

interface LessonDocumentToolbarProps {
  compact: boolean
  editing: boolean
  hasContent: boolean
  disabled: boolean
  disabledReason: string
  operationDisabled?: boolean
  operationDisabledReason?: string
  planID: string
  currentVersion: number
  saving: boolean
  uploading: boolean
  externalChanged: boolean
  hasDraft: boolean
  rewriteDisabled: boolean
  activeSection:
    LessonDocumentSection | null
  onOpenOutline: () => void
  onOpenSectionRewrite: (
    section: LessonDocumentSection,
  ) => void
  onOpenHistory: () => void
  onEnterEdit: () => void
  onChooseImage: () => void
  onCancelEdit: () => void
  onSave: () => void
}

export default function LessonDocumentToolbar({
  compact,
  editing,
  hasContent,
  disabled,
  disabledReason,
  operationDisabled = false,
  operationDisabledReason = '',
  planID,
  currentVersion,
  saving,
  uploading,
  externalChanged,
  hasDraft,
  rewriteDisabled,
  activeSection,
  onOpenOutline,
  onOpenSectionRewrite,
  onOpenHistory,
  onEnterEdit,
  onChooseImage,
  onCancelEdit,
  onSave,
}: LessonDocumentToolbarProps) {
  const controlsDisabled =
    disabled ||
    operationDisabled

  const controlsDisabledReason =
    operationDisabled
      ? operationDisabledReason ||
        '当前操作正在进行，请稍候'
      : disabledReason

  const outlineDisabled =
    !hasContent ||
    operationDisabled

  const historyDisabled =
    !planID ||
    operationDisabled

  const editDisabled =
    controlsDisabled ||
    !planID

  const saveDisabled =
    saving ||
    uploading ||
    controlsDisabled ||
    externalChanged ||
    !hasDraft

  return (
    <div style={{
      padding: compact
        ? '9px 12px'
        : '11px 16px',
      borderBottom:
        '1px solid #F3F4F6',
      display: 'flex',
      alignItems: 'center',
      justifyContent:
        'space-between',
      gap: '10px',
      flexShrink: 0,
    }}>
      <div style={{
        minWidth: 0,
      }}>
        <div style={{
          fontSize: '12px',
          fontWeight: 700,
          color: '#374151',
        }}>
          {editing
            ? '✏️ 正在编辑完整教案'
            : '📄 教案正文'}
        </div>

        {controlsDisabled &&
          controlsDisabledReason && (
          <div style={{
            marginTop: '2px',
            fontSize: '10px',
            color: '#9CA3AF',
            overflow: 'hidden',
            textOverflow:
              'ellipsis',
            whiteSpace: 'nowrap',
          }}>
            {controlsDisabledReason}
          </div>
        )}
      </div>

      {!editing ? (
        <div style={{
          display: 'flex',
          gap: '6px',
          alignItems: 'center',
        }}>
          <button
            type="button"
            onClick={onOpenOutline}
            disabled={outlineDisabled}
            title="查看教案目录并快速定位章节"
            style={{
              padding: '5px 9px',
              borderRadius: '7px',
              border:
                '1px solid #D1D5DB',
              background: '#FFFFFF',
              color: outlineDisabled
                ? '#D1D5DB'
                : '#4F7BE8',
              fontSize: '12px',
              fontWeight: 600,
              cursor: outlineDisabled
                ? 'not-allowed'
                : 'pointer',
              whiteSpace: 'nowrap',
            }}
          >
            ☰ 目录
          </button>

          {activeSection && (
            <button
              type="button"
              onClick={() =>
                onOpenSectionRewrite(
                  activeSection,
                )
              }
              disabled={
                rewriteDisabled
              }
              title={
                rewriteDisabled
                  ? disabledReason ||
                    '当前暂不能使用AI修改'
                  : `让AI修改“${activeSection.title}”`
              }
              style={{
                padding: '5px 9px',
                borderRadius: '7px',
                border:
                  '1px solid rgba(79,123,232,0.25)',
                background:
                  rewriteDisabled
                    ? '#F9FAFB'
                    : 'rgba(79,123,232,0.07)',
                color:
                  rewriteDisabled
                    ? '#D1D5DB'
                    : '#365FB8',
                fontSize: '12px',
                fontWeight: 600,
                cursor:
                  rewriteDisabled
                    ? 'not-allowed'
                    : 'pointer',
                whiteSpace: 'nowrap',
              }}
            >
              ✨ AI修改
            </button>
          )}

          <button
            type="button"
            onClick={onOpenHistory}
            disabled={historyDisabled}
            title={
              operationDisabled
                ? controlsDisabledReason
                : '查看正文修改记录、对比差异或恢复历史版本'
            }
            style={{
              padding: '5px 10px',
              borderRadius: '7px',
              border:
                '1px solid #D1D5DB',
              background: '#FFFFFF',
              color: '#6B7280',
              fontSize: '12px',
              fontWeight: 600,
              cursor: historyDisabled
                ? 'not-allowed'
                : 'pointer',
              whiteSpace: 'nowrap',
            }}
          >
            🕘 v{currentVersion}
          </button>

          <button
            type="button"
            onClick={onEnterEdit}
            disabled={editDisabled}
            title={
              controlsDisabled
                ? controlsDisabledReason
                : '直接编辑完整教案正文'
            }
            style={{
              padding: '5px 11px',
              borderRadius: '7px',
              border:
                '1px solid #D1D5DB',
              background:
                editDisabled
                  ? '#F3F4F6'
                  : '#FFFFFF',
              color:
                editDisabled
                  ? '#9CA3AF'
                  : '#374151',
              fontSize: '12px',
              fontWeight: 600,
              cursor:
                editDisabled
                  ? 'not-allowed'
                  : 'pointer',
              whiteSpace: 'nowrap',
            }}
          >
            ✏️ {hasContent
              ? '编辑'
              : '填写'}
          </button>
        </div>
      ) : (
        <div style={{
          display: 'flex',
          gap: '6px',
          alignItems: 'center',
        }}>
          <button
            type="button"
            onClick={onChooseImage}
            disabled={
              saving ||
              uploading ||
              controlsDisabled
            }
            title="支持点击选择、拖拽或粘贴图片"
            style={{
              padding: '5px 9px',
              borderRadius: '7px',
              border:
                '1px solid #D1D5DB',
              background: '#FFFFFF',
              color: uploading
                ? '#9CA3AF'
                : '#374151',
              fontSize: '12px',
              cursor: uploading
                ? 'wait'
                : 'pointer',
              whiteSpace: 'nowrap',
            }}
          >
            {uploading
              ? '⏳ 上传中'
              : '📷 插图'}
          </button>

          <button
            type="button"
            onClick={onCancelEdit}
            disabled={saving}
            style={{
              padding: '5px 9px',
              borderRadius: '7px',
              border:
                '1px solid #E5E7EB',
              background: '#FFFFFF',
              color: '#6B7280',
              fontSize: '12px',
              cursor: saving
                ? 'not-allowed'
                : 'pointer',
            }}
          >
            取消
          </button>

          <button
            type="button"
            onClick={onSave}
            disabled={saveDisabled}
            style={{
              padding: '5px 11px',
              borderRadius: '7px',
              border: 'none',
              background:
                saveDisabled
                  ? '#E5E7EB'
                  : '#10B981',
              color:
                saveDisabled
                  ? '#9CA3AF'
                  : '#FFFFFF',
              fontSize: '12px',
              fontWeight: 700,
              cursor:
                saveDisabled
                  ? 'not-allowed'
                  : 'pointer',
            }}
          >
            {saving
              ? '保存中…'
              : '✓ 保存'}
          </button>
        </div>
      )}
    </div>
  )
}

/**
 * CoursewareComicPanelEditorFooter.tsx
 *
 * 单格编辑器的轻量工具栏：
 *   - 自动保存只显示短状态，异常时才展开解释；
 *   - 撤销、重做和手动保存保持常驻；
 *   - 画面重画要求默认折叠，减少持续信息占用；
 *   - 已插入项目仍可同步当前格。
 */

import type {
  CSSProperties,
} from 'react'

import type {
  CoursewareComicAutoSaveState,
} from './useCoursewareComicPanelEditor'

const C = {
  success: '#059669',
  warning: '#D97706',
  danger: '#DC2626',
  info: '#2563EB',
  text: '#1F2937',
  textSecondary: '#64748B',
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  background: '#F8FAFC',
  white: '#FFFFFF',
}

export const coursewareComicPanelEditorStyles = {
  container: {
    padding: 14,
    borderRadius: 14,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    boxShadow:
      '0 10px 28px rgba(15,23,42,0.05)',
  },

  footer: {
    display: 'grid',
    gap: 10,
    marginTop: 12,
    paddingTop: 12,
    borderTop:
      `1px solid ${C.border}`,
  },

  primaryRow: {
    display: 'flex',
    alignItems: 'center',
    justifyContent:
      'space-between',
    gap: 10,
    flexWrap: 'wrap',
  },

  footerActions: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: 7,
    flexWrap: 'wrap',
  },

  secondaryButton: {
    minHeight: 36,
    padding: '7px 11px',
    borderRadius: 9,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    color: C.textSecondary,
    fontSize: 13,
    fontWeight: 800,
    cursor: 'pointer',
  },

  primaryButton: {
    minHeight: 38,
    padding: '8px 13px',
    borderRadius: 9,
    border: 'none',
    background:
      'linear-gradient(135deg,#7C3AED,#4F46E5)',
    color: '#FFFFFF',
    fontSize: 13,
    fontWeight: 900,
    cursor: 'pointer',
  },

  successButton: {
    minHeight: 38,
    padding: '8px 13px',
    borderRadius: 9,
    border: '1px solid #10B981',
    background: '#ECFDF5',
    color: '#047857',
    fontSize: 13,
    fontWeight: 900,
    cursor: 'pointer',
  },

  toolDisclosure: {
    borderRadius: 10,
    border:
      `1px solid ${C.border}`,
    background: C.background,
  },

  toolSummary: {
    minHeight: 40,
    display: 'flex',
    alignItems: 'center',
    justifyContent:
      'space-between',
    gap: 10,
    padding: '8px 11px',
    color: C.text,
    fontSize: 13,
    fontWeight: 900,
    cursor: 'pointer',
    listStyle: 'none',
  },

  toolSummaryHint: {
    color: C.textMuted,
    fontSize: 12,
    fontWeight: 700,
  },

  regenerationPanel: {
    padding: '0 11px 11px',
  },

  regenerationTextarea: {
    width: '100%',
    minHeight: 88,
    boxSizing: 'border-box',
    padding: '10px 11px',
    resize: 'vertical',
    borderRadius: 9,
    border: '1px solid #F59E0B',
    background: C.white,
    color: C.text,
    fontFamily: 'inherit',
    fontSize: 14,
    lineHeight: 1.55,
    outline: 'none',
  },

  regenerationMeta: {
    display: 'flex',
    alignItems: 'center',
    justifyContent:
      'space-between',
    gap: 10,
    marginTop: 7,
  },

  regenerationHelp: {
    color: C.textSecondary,
    fontSize: 12,
    lineHeight: 1.45,
  },

  warningButton: {
    minHeight: 38,
    padding: '8px 13px',
    borderRadius: 9,
    border: '1px solid #F59E0B',
    background: '#FFFBEB',
    color: '#B45309',
    fontSize: 13,
    fontWeight: 900,
    cursor: 'pointer',
  },
} satisfies Record<
  string,
  CSSProperties
>

interface CoursewareComicPanelFooterProps {
  version: number
  overlayDirty: boolean
  editorDisabled: boolean
  saving: boolean
  autoSaveState:
    CoursewareComicAutoSaveState
  autoSaveMessage: string
  overlayVersionConflict: boolean
  regenerating: boolean
  syncing: boolean
  canRegenerate: boolean
  canSync: boolean
  canUndo: boolean
  canRedo: boolean
  showSync: boolean
  regenerationInstruction: string
  regenerationInstructionLength: number
  regenerationInstructionMaxLength: number
  onRegenerationInstructionChange: (
    value: string,
  ) => void
  onUndo: () => void
  onRedo: () => void
  onSave: () => void
  onRegenerate: () => void
  onSync: () => void
}

function defaultCoursewareComicSaveStatus(
  state:
    CoursewareComicAutoSaveState,
  overlayDirty: boolean,
): string {
  switch (state) {
  case 'saving':
    return '自动保存中'

  case 'pending':
    return '等待自动保存'

  case 'invalid':
    return '草稿需要修正'

  case 'error':
    return '自动保存失败'

  case 'conflict':
    return '服务器版本冲突'

  default:
    return overlayDirty
      ? '本地修改未同步'
      : '已保存'
  }
}

function coursewareComicSaveStatusColor(
  state:
    CoursewareComicAutoSaveState,
  overlayDirty: boolean,
): string {
  switch (state) {
  case 'saving':
  case 'pending':
    return C.info

  case 'invalid':
    return C.warning

  case 'error':
  case 'conflict':
    return C.danger

  default:
    return overlayDirty
      ? C.warning
      : C.success
  }
}

export function CoursewareComicPanelFooter({
  version,
  overlayDirty,
  editorDisabled,
  saving,
  autoSaveState,
  autoSaveMessage,
  overlayVersionConflict,
  regenerating,
  syncing,
  canRegenerate,
  canSync,
  canUndo,
  canRedo,
  showSync,
  regenerationInstruction,
  regenerationInstructionLength,
  regenerationInstructionMaxLength,
  onRegenerationInstructionChange,
  onUndo,
  onRedo,
  onSave,
  onRegenerate,
  onSync,
}: CoursewareComicPanelFooterProps) {
  const styles =
    coursewareComicPanelEditorStyles

  const saveStatus =
    defaultCoursewareComicSaveStatus(
      autoSaveState,
      overlayDirty,
    )

  const statusColor =
    coursewareComicSaveStatusColor(
      autoSaveState,
      overlayDirty,
    )

  const showSaveDetail =
    (
      autoSaveState === 'invalid' ||
      autoSaveState === 'error' ||
      autoSaveState === 'conflict'
    ) &&
    Boolean(
      autoSaveMessage.trim(),
    )

  const regenerateTitle =
    overlayDirty
      ? '请等待自动保存完成或先手动保存'
      : regenerationInstruction.trim()
        ? '按本次要求重新生成底图'
        : '请先填写本次画面要求'

  return (
    <div style={styles.footer}>
      <div style={styles.primaryRow}>
        <div
          title={
            `服务器版本 ${version}`
          }
          style={saveStatusStyle}
        >
          <span
            style={{
              ...saveDotStyle,
              background:
                statusColor,
            }}
          />

          <span
            style={{
              color:
                statusColor,
            }}
          >
            {saveStatus}
          </span>

          {showSaveDetail && (
            <span style={saveDetailStyle}>
              {autoSaveMessage}
            </span>
          )}
        </div>

        <div style={styles.footerActions}>
          <button
            type="button"
            onClick={onUndo}
            disabled={
              editorDisabled ||
              !canUndo
            }
            style={styles.secondaryButton}
            title="撤销"
          >
            ↶ 撤销
          </button>

          <button
            type="button"
            onClick={onRedo}
            disabled={
              editorDisabled ||
              !canRedo
            }
            style={styles.secondaryButton}
            title="重做"
          >
            ↷ 重做
          </button>

          <button
            type="button"
            onClick={onSave}
            disabled={
              editorDisabled ||
              saving ||
              !overlayDirty
            }
            title={
              overlayVersionConflict
                ? '本地草稿已保留；按当前服务器版本明确重试'
                : '立即同步当前本地草稿'
            }
            style={{
              ...styles.primaryButton,
              opacity:
                editorDisabled ||
                saving ||
                !overlayDirty
                  ? 0.5
                  : 1,
            }}
          >
            {saving
              ? '保存中…'
              : overlayVersionConflict
                ? '保存本地草稿'
                : '立即保存'}
          </button>

          {showSync && (
            <button
              type="button"
              onClick={onSync}
              disabled={
                !canSync
              }
              title={
                overlayDirty
                  ? '请等待自动保存完成或先手动保存'
                  : '同步当前格到课件'
              }
              style={styles.successButton}
            >
              {syncing
                ? '同步中…'
                : '同步课件'}
            </button>
          )}
        </div>
      </div>

      <details style={styles.toolDisclosure}>
        <summary style={styles.toolSummary}>
          <span>
            重新生成底图
          </span>

          <span style={styles.toolSummaryHint}>
            {regenerationInstructionLength > 0
              ? `已填写${regenerationInstructionLength}字`
              : '按需展开'}
          </span>
        </summary>

        <div style={styles.regenerationPanel}>
          <textarea
            value={
              regenerationInstruction
            }
            onChange={event =>
              onRegenerationInstructionChange(
                event.target.value,
              )
            }
            disabled={
              editorDisabled
            }
            rows={4}
            placeholder="例如：镜头拉近到半身；减少背景杂物；保持人物服装和画风不变。"
            style={styles.regenerationTextarea}
          />

          <div style={styles.regenerationMeta}>
            <span style={styles.regenerationHelp}>
              只影响本次重画，不修改分镜和人物设定
              {' · '}
              {regenerationInstructionLength}
              /
              {regenerationInstructionMaxLength}
            </span>

            <button
              type="button"
              onClick={onRegenerate}
              disabled={
                !canRegenerate
              }
              title={
                regenerateTitle
              }
              style={{
                ...styles.warningButton,
                opacity:
                  canRegenerate
                    ? 1
                    : 0.55,
              }}
            >
              {regenerating
                ? '生成中…'
                : '按要求重画'}
            </button>
          </div>
        </div>
      </details>
    </div>
  )
}

const saveStatusStyle:
  CSSProperties = {
    minWidth: 0,
    display: 'flex',
    alignItems: 'center',
    gap: 7,
    color: C.textSecondary,
    fontSize: 13,
    fontWeight: 800,
    flexWrap: 'wrap',
  }

const saveDotStyle:
  CSSProperties = {
    width: 8,
    height: 8,
    flexShrink: 0,
    borderRadius: 999,
  }

const saveDetailStyle:
  CSSProperties = {
    maxWidth: 520,
    color: C.textSecondary,
    fontSize: 12,
    fontWeight: 500,
    lineHeight: 1.45,
  }

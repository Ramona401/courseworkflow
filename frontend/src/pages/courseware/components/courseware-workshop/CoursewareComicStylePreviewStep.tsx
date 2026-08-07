/**
 * CoursewareComicStylePreviewStep.tsx
 *
 * 第三步后半段以样张为视觉中心：
 *   - 画风、比例和清晰度压缩为顶部标签；
 *   - 大画面占据主要空间；
 *   - 补充要求与原理说明按需展开；
 *   - 底部只保留“重新生成”和“确认继续”两个动作。
 */

import type {
  CoursewareComicPanel,
  CoursewareComicWorkflowProject,
} from '@/api/coursewares'

import {
  canConfirmCoursewareComicPreview,
  canGenerateCoursewareComicPreview,
  coursewareComicVisualLabel,
  findCoursewareComicPreviewPanel,
} from './coursewareComicWorkflow'

import CoursewareComicPanelPreview from './CoursewareComicPanelPreview'

interface CoursewareComicStylePreviewStepProps {
  project: CoursewareComicWorkflowProject
  panels: CoursewareComicPanel[]
  busy: boolean
  onRegenerate: () => void
  onConfirm: (
    panel: CoursewareComicPanel,
  ) => void
}

const C = {
  primary: '#7C3AED',
  success: '#059669',
  warning: '#D97706',
  text: '#1F2937',
  textSecondary: '#64748B',
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  background: '#F8FAFC',
  white: '#FFFFFF',
}

export default function CoursewareComicStylePreviewStep({
  project,
  panels,
  busy,
  onRegenerate,
  onConfirm,
}: CoursewareComicStylePreviewStepProps) {
  const preview =
    findCoursewareComicPreviewPanel(
      project,
      panels,
    )

  const generating =
    preview?.status ===
      'generating'

  const canRegenerate =
    canGenerateCoursewareComicPreview(
      project,
      panels,
    )

  const canConfirm =
    canConfirmCoursewareComicPreview(
      project,
      panels,
    )

  const hasImage =
    Boolean(
      preview?.current_asset_id &&
      preview.current_asset_url,
    )

  const statusLabel =
    generating
      ? '生成中'
      : hasImage
        ? '可以确认'
        : '等待生成'

  return (
    <section style={containerStyle}>
      <header style={headerStyle}>
        <div>
          <div style={eyebrowStyle}>
            样张检查
          </div>

          <h2 style={titleStyle}>
            看这一格，决定整套视觉
          </h2>
        </div>

        <div style={{
          ...statusBadgeStyle,
          color:
            generating
              ? C.warning
              : hasImage
                ? C.success
                : C.textMuted,
          background:
            generating
              ? '#FEF3C7'
              : hasImage
                ? '#ECFDF5'
                : C.background,
        }}>
          {statusLabel}
        </div>
      </header>

      <div style={chipRowStyle}>
        <MetaChip
          label={
            coursewareComicVisualLabel(
              project.visual_style,
            )
          }
        />
        <MetaChip
          label={
            project.workflow.aspect_ratio
          }
        />
        <MetaChip
          label={
            project.workflow.image_quality ===
              'high'
              ? '高清'
              : '标准'
          }
        />
        <MetaChip label="第1格" />
      </div>

      <div style={previewFrameStyle}>
        {preview ? (
          <CoursewareComicPanelPreview
            panel={preview}
            aspectRatio={
              project.workflow.aspect_ratio
            }
            emptyLabel={
              generating
                ? '正在生成首格样张…'
                : '点击下方按钮生成样张'
            }
          />
        ) : (
          <div style={emptyStyle}>
            缺少第1格分镜，请返回分镜步骤检查。
          </div>
        )}
      </div>

      {preview?.last_error && (
        <div style={errorStyle}>
          {preview.last_error}
        </div>
      )}

      {project.workflow.style_instruction && (
        <details style={detailsStyle}>
          <summary style={summaryStyle}>
            查看画风补充要求
          </summary>
          <div style={detailsContentStyle}>
            {
              project.workflow
                .style_instruction
            }
          </div>
        </details>
      )}

      <details style={detailsStyle}>
        <summary style={summaryStyle}>
          这张样张会影响什么？
        </summary>
        <div style={detailsContentStyle}>
          确认后，其余分格会沿用相同画风、画幅和清晰度，
          并使用人物设定图与已完成分格保持连续性。
          图片本身不含文字，课堂文字仍由可编辑覆盖层呈现。
        </div>
      </details>

      <footer style={footerStyle}>
        <button
          type="button"
          onClick={onRegenerate}
          disabled={
            busy ||
            generating ||
            !canRegenerate
          }
          style={{
            ...secondaryButtonStyle,
            opacity:
              busy ||
              generating ||
              !canRegenerate
                ? 0.5
                : 1,
          }}
        >
          {generating
            ? '样张生成中…'
            : hasImage
              ? '重新生成'
              : '生成样张'}
        </button>

        <button
          type="button"
          onClick={() => {
            if (preview) {
              onConfirm(preview)
            }
          }}
          disabled={
            busy ||
            generating ||
            !canConfirm ||
            !preview
          }
          style={{
            ...primaryButtonStyle,
            opacity:
              busy ||
              generating ||
              !canConfirm ||
              !preview
                ? 0.55
                : 1,
          }}
        >
          {busy
            ? '处理中…'
            : '确认视觉并继续 →'}
        </button>
      </footer>
    </section>
  )
}

function MetaChip({
  label,
}: {
  label: string
}) {
  return (
    <span style={chipStyle}>
      {label}
    </span>
  )
}

const containerStyle:
  React.CSSProperties = {
    marginBottom: 18,
    padding: 20,
    borderRadius: 16,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    boxShadow:
      '0 12px 32px rgba(15,23,42,0.05)',
  }

const headerStyle:
  React.CSSProperties = {
    display: 'flex',
    alignItems: 'flex-start',
    justifyContent:
      'space-between',
    gap: 16,
    marginBottom: 12,
  }

const eyebrowStyle:
  React.CSSProperties = {
    marginBottom: 4,
    color: C.primary,
    fontSize: 12,
    fontWeight: 900,
  }

const titleStyle:
  React.CSSProperties = {
    margin: 0,
    color: C.text,
    fontSize: 20,
    lineHeight: 1.3,
    fontWeight: 900,
  }

const statusBadgeStyle:
  React.CSSProperties = {
    flexShrink: 0,
    padding: '6px 11px',
    borderRadius: 999,
    fontSize: 12,
    fontWeight: 900,
  }

const chipRowStyle:
  React.CSSProperties = {
    display: 'flex',
    gap: 7,
    marginBottom: 14,
    flexWrap: 'wrap',
  }

const chipStyle:
  React.CSSProperties = {
    padding: '6px 10px',
    borderRadius: 999,
    background: C.background,
    color: C.textSecondary,
    fontSize: 12,
    fontWeight: 800,
  }

const previewFrameStyle:
  React.CSSProperties = {
    overflow: 'hidden',
    borderRadius: 14,
    border:
      `1px solid ${C.border}`,
    background: '#F1F5F9',
    boxShadow:
      'inset 0 0 0 1px rgba(255,255,255,0.6)',
  }

const emptyStyle:
  React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: 300,
    padding: 24,
    color: C.textMuted,
    textAlign: 'center',
    fontSize: 14,
  }

const errorStyle:
  React.CSSProperties = {
    marginTop: 12,
    padding: '11px 13px',
    borderRadius: 10,
    border:
      '1px solid #FECACA',
    background: '#FEF2F2',
    color: '#DC2626',
    fontSize: 13,
    lineHeight: 1.55,
  }

const detailsStyle:
  React.CSSProperties = {
    marginTop: 10,
    padding: '9px 11px',
    borderRadius: 9,
    background: C.background,
  }

const summaryStyle:
  React.CSSProperties = {
    color: C.textSecondary,
    fontSize: 12,
    fontWeight: 800,
    cursor: 'pointer',
  }

const detailsContentStyle:
  React.CSSProperties = {
    marginTop: 8,
    color: C.textSecondary,
    fontSize: 13,
    lineHeight: 1.6,
  }

const footerStyle:
  React.CSSProperties = {
    position: 'sticky',
    bottom: 8,
    zIndex: 2,
    display: 'flex',
    justifyContent: 'flex-end',
    gap: 9,
    marginTop: 16,
    padding: '12px 14px',
    border:
      `1px solid ${C.border}`,
    borderRadius: 12,
    background:
      'rgba(255,255,255,0.96)',
    boxShadow:
      '0 10px 24px rgba(15,23,42,0.10)',
    backdropFilter: 'blur(10px)',
    flexWrap: 'wrap',
  }

const secondaryButtonStyle:
  React.CSSProperties = {
    minHeight: 42,
    padding: '10px 16px',
    borderRadius: 10,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    color: C.text,
    fontSize: 14,
    fontWeight: 800,
    cursor: 'pointer',
  }

const primaryButtonStyle:
  React.CSSProperties = {
    minHeight: 42,
    padding: '10px 17px',
    borderRadius: 10,
    border: 'none',
    background:
      'linear-gradient(135deg,#059669,#047857)',
    color: '#FFFFFF',
    fontSize: 14,
    fontWeight: 900,
    cursor: 'pointer',
  }

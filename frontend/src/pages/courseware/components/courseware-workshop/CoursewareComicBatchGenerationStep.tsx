/**
 * CoursewareComicBatchGenerationStep.tsx
 *
 * 第四步采用视觉化进度面板：
 *   - 顶部只显示完成数和当前并发状态；
 *   - 使用四个数字概览替代长段说明；
 *   - 分格卡片以缩略图和状态为主，文字仅保留一行；
 *   - 错误信息只在失败时出现。
 */

import type {
  CoursewareComicPanel,
  CoursewareComicWorkflowProject,
} from '@/api/coursewares'

import {
  canGenerateCoursewareComicBatch,
} from './coursewareComicWorkflow'

interface CoursewareComicBatchGenerationStepProps {
  project: CoursewareComicWorkflowProject
  panels: CoursewareComicPanel[]
  busy: boolean
  onGenerate: () => void
}

const C = {
  primary: '#7C3AED',
  success: '#059669',
  warning: '#D97706',
  danger: '#DC2626',
  text: '#1F2937',
  textSecondary: '#64748B',
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  background: '#F8FAFC',
  white: '#FFFFFF',
}

export default function CoursewareComicBatchGenerationStep({
  project,
  panels,
  busy,
  onGenerate,
}: CoursewareComicBatchGenerationStepProps) {
  const generatedCount =
    panels.filter(
      panel =>
        panel.status === 'generated' &&
        Boolean(
          panel.current_asset_id,
        ),
    ).length

  const generatingCount =
    panels.filter(
      panel =>
        panel.status === 'generating',
    ).length

  const failedCount =
    panels.filter(
      panel =>
        panel.status === 'failed' ||
        panel.status === 'stale',
    ).length

  const pendingCount =
    Math.max(
      0,
      panels.length -
        generatedCount -
        generatingCount -
        failedCount,
    )

  const percent =
    panels.length > 0
      ? Math.round(
          generatedCount /
            panels.length *
            100,
        )
      : 0

  const generating =
    project.status === 'generating' ||
    generatingCount > 0

  const completed =
    panels.length > 0 &&
    generatedCount === panels.length

  const canGenerate =
    canGenerateCoursewareComicBatch(
      project,
    )

  return (
    <section style={containerStyle}>
      <header style={headerStyle}>
        <div>
          <div style={eyebrowStyle}>
            第4步
          </div>

          <h2 style={titleStyle}>
            {completed
              ? '全部图片已完成'
              : generating
                ? '正在并发生成'
                : '准备生成全部图片'}
          </h2>

          <div style={subtitleStyle}>
            最多同时处理4格，可随时离开页面，后台任务会继续。
          </div>
        </div>

        <div style={bigProgressStyle}>
          <strong style={bigNumberStyle}>
            {generatedCount}
          </strong>
          <span style={bigTotalStyle}>
            / {panels.length}
          </span>
        </div>
      </header>

      <div
        aria-label={`已完成${percent}%`}
        style={progressTrackStyle}
      >
        <div style={{
          ...progressValueStyle,
          width: `${percent}%`,
        }} />
      </div>

      <div style={metricGridStyle}>
        <Metric
          value={generatedCount}
          label="已完成"
          tone="success"
        />
        <Metric
          value={generatingCount}
          label="生成中"
          tone="warning"
        />
        <Metric
          value={pendingCount}
          label="待处理"
          tone="neutral"
        />
        <Metric
          value={failedCount}
          label="需重试"
          tone="danger"
        />
      </div>

      <div style={panelGridStyle}>
        {panels.map(panel => (
          <PanelStatusCard
            key={panel.id}
            panel={panel}
          />
        ))}
      </div>

      {project.last_error && (
        <details style={errorDetailsStyle}>
          <summary style={errorSummaryStyle}>
            查看失败原因
          </summary>
          <div style={errorContentStyle}>
            {project.last_error}
          </div>
        </details>
      )}

      <footer style={footerStyle}>
        <div style={footerStateStyle}>
          {completed
            ? '可以进入精修与使用'
            : generating
              ? `当前有${generatingCount}格正在处理`
              : failedCount > 0
                ? '成功图片已保留，只需继续未完成部分'
                : '确认后开始后台生成'}
        </div>

        <button
          type="button"
          onClick={onGenerate}
          disabled={
            busy ||
            generating ||
            !canGenerate
          }
          style={{
            ...primaryButtonStyle,
            opacity:
              busy ||
              generating ||
              !canGenerate
                ? 0.55
                : 1,
          }}
        >
          {generating
            ? '生成进行中…'
            : project.status === 'failed'
              ? '继续生成 →'
              : completed
                ? '已全部完成'
                : '开始生成 →'}
        </button>
      </footer>
    </section>
  )
}

function Metric({
  value,
  label,
  tone,
}: {
  value: number
  label: string
  tone:
    | 'success'
    | 'warning'
    | 'danger'
    | 'neutral'
}) {
  const colors = {
    success: {
      color: C.success,
      background: '#ECFDF5',
    },
    warning: {
      color: C.warning,
      background: '#FFFBEB',
    },
    danger: {
      color: C.danger,
      background: '#FEF2F2',
    },
    neutral: {
      color: C.textSecondary,
      background: C.background,
    },
  }

  return (
    <div style={{
      ...metricStyle,
      background:
        colors[tone].background,
    }}>
      <div style={{
        ...metricValueStyle,
        color: colors[tone].color,
      }}>
        {value}
      </div>
      <div style={metricLabelStyle}>
        {label}
      </div>
    </div>
  )
}

function PanelStatusCard({
  panel,
}: {
  panel: CoursewareComicPanel
}) {
  const status =
    panelStatus(panel)

  return (
    <article style={{
      ...cardStyle,
      borderColor:
        status.borderColor,
      background:
        status.background,
    }}>
      <div style={thumbnailStyle}>
        {panel.current_asset_url ? (
          <img
            src={
              panel.current_asset_url
            }
            alt={
              `第${panel.panel_no}格`
            }
            draggable={false}
            style={imageStyle}
          />
        ) : (
          <div style={emptyStyle}>
            <span style={emptyIconStyle}>
              {status.icon}
            </span>
            {status.label}
          </div>
        )}

        <span style={numberBadgeStyle}>
          {panel.panel_no}
        </span>

        <span style={{
          ...statusBadgeStyle,
          color: status.color,
          background:
            status.badgeBackground,
        }}>
          {status.label}
        </span>
      </div>

      <div style={cardBodyStyle}>
        <div
          title={panel.story_purpose}
          style={purposeStyle}
        >
          {panel.story_purpose ||
            `第${panel.panel_no}格`}
        </div>
      </div>
    </article>
  )
}

function panelStatus(
  panel: CoursewareComicPanel,
): {
  label: string
  color: string
  icon: string
  borderColor: string
  background: string
  badgeBackground: string
} {
  switch (panel.status) {
  case 'generated':
    return {
      label: '已完成',
      color: C.success,
      icon: '✓',
      borderColor: '#A7F3D0',
      background: C.white,
      badgeBackground: '#ECFDF5',
    }

  case 'generating':
    return {
      label: '生成中',
      color: C.warning,
      icon: '…',
      borderColor: '#FDE68A',
      background: '#FFFBEB',
      badgeBackground: '#FEF3C7',
    }

  case 'failed':
    return {
      label: '失败',
      color: C.danger,
      icon: '!',
      borderColor: '#FECACA',
      background: '#FEF2F2',
      badgeBackground: '#FEE2E2',
    }

  case 'stale':
    return {
      label: '需重画',
      color: C.warning,
      icon: '↻',
      borderColor: '#FDE68A',
      background: '#FFFBEB',
      badgeBackground: '#FEF3C7',
    }

  default:
    return {
      label: '待生成',
      color: C.textMuted,
      icon: '○',
      borderColor: C.border,
      background: C.white,
      badgeBackground: C.background,
    }
  }
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
    marginBottom: 14,
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

const subtitleStyle:
  React.CSSProperties = {
    marginTop: 5,
    color: C.textSecondary,
    fontSize: 13,
    lineHeight: 1.5,
  }

const bigProgressStyle:
  React.CSSProperties = {
    display: 'flex',
    alignItems: 'baseline',
    flexShrink: 0,
    padding: '8px 12px',
    borderRadius: 12,
    background:
      'rgba(124,58,237,0.08)',
    color: C.primary,
  }

const bigNumberStyle:
  React.CSSProperties = {
    fontSize: 26,
    lineHeight: 1,
    fontWeight: 900,
  }

const bigTotalStyle:
  React.CSSProperties = {
    marginLeft: 3,
    fontSize: 13,
    fontWeight: 800,
  }

const progressTrackStyle:
  React.CSSProperties = {
    height: 10,
    overflow: 'hidden',
    marginBottom: 14,
    borderRadius: 999,
    background: '#E2E8F0',
  }

const progressValueStyle:
  React.CSSProperties = {
    height: '100%',
    borderRadius: 999,
    background:
      'linear-gradient(90deg,#7C3AED,#10B981)',
    transition:
      'width 300ms',
  }

const metricGridStyle:
  React.CSSProperties = {
    display: 'grid',
    gridTemplateColumns:
      'repeat(auto-fit,minmax(105px,1fr))',
    gap: 8,
    marginBottom: 14,
  }

const metricStyle:
  React.CSSProperties = {
    padding: '10px 12px',
    borderRadius: 10,
    textAlign: 'center',
  }

const metricValueStyle:
  React.CSSProperties = {
    fontSize: 19,
    fontWeight: 900,
  }

const metricLabelStyle:
  React.CSSProperties = {
    marginTop: 2,
    color: C.textSecondary,
    fontSize: 11,
    fontWeight: 700,
  }

const panelGridStyle:
  React.CSSProperties = {
    display: 'grid',
    gridTemplateColumns:
      'repeat(auto-fit,minmax(170px,1fr))',
    gap: 10,
  }

const cardStyle:
  React.CSSProperties = {
    overflow: 'hidden',
    borderRadius: 12,
    border: '1px solid',
    transition:
      'border-color 180ms, box-shadow 180ms',
  }

const thumbnailStyle:
  React.CSSProperties = {
    position: 'relative',
    aspectRatio: '16 / 9',
    background: C.background,
  }

const imageStyle:
  React.CSSProperties = {
    width: '100%',
    height: '100%',
    objectFit: 'cover',
    display: 'block',
  }

const emptyStyle:
  React.CSSProperties = {
    height: '100%',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 5,
    color: C.textMuted,
    fontSize: 12,
    fontWeight: 700,
  }

const emptyIconStyle:
  React.CSSProperties = {
    fontSize: 22,
    lineHeight: 1,
  }

const numberBadgeStyle:
  React.CSSProperties = {
    position: 'absolute',
    top: 8,
    left: 8,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    width: 26,
    height: 26,
    borderRadius: 999,
    background:
      'rgba(15,23,42,0.78)',
    color: '#FFFFFF',
    fontSize: 12,
    fontWeight: 900,
  }

const statusBadgeStyle:
  React.CSSProperties = {
    position: 'absolute',
    top: 8,
    right: 8,
    padding: '4px 8px',
    borderRadius: 999,
    fontSize: 11,
    fontWeight: 900,
  }

const cardBodyStyle:
  React.CSSProperties = {
    padding: '10px 11px',
  }

const purposeStyle:
  React.CSSProperties = {
    overflow: 'hidden',
    color: C.textSecondary,
    fontSize: 12,
    lineHeight: 1.4,
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  }

const errorDetailsStyle:
  React.CSSProperties = {
    marginTop: 12,
    padding: '10px 12px',
    borderRadius: 10,
    border: '1px solid #FECACA',
    background: '#FEF2F2',
  }

const errorSummaryStyle:
  React.CSSProperties = {
    color: C.danger,
    fontSize: 12,
    fontWeight: 900,
    cursor: 'pointer',
  }

const errorContentStyle:
  React.CSSProperties = {
    marginTop: 8,
    color: '#991B1B',
    fontSize: 13,
    lineHeight: 1.55,
  }

const footerStyle:
  React.CSSProperties = {
    position: 'sticky',
    bottom: 8,
    zIndex: 2,
    display: 'flex',
    alignItems: 'center',
    justifyContent:
      'space-between',
    gap: 12,
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

const footerStateStyle:
  React.CSSProperties = {
    color: C.textSecondary,
    fontSize: 13,
  }

const primaryButtonStyle:
  React.CSSProperties = {
    minHeight: 42,
    padding: '10px 17px',
    borderRadius: 10,
    border: 'none',
    background:
      'linear-gradient(135deg,#7C3AED,#4F46E5)',
    color: '#FFFFFF',
    fontSize: 14,
    fontWeight: 900,
    cursor: 'pointer',
  }

/**
 * AssemblyProgressView.tsx — 全自动装配页面网格进度视图
 *
 * 纯展示组件：
 *   - 按页展示HTML、配图、视频三条流水线；
 *   - 展示数据库业务运行状态；
 *   - 区分完成、取消、失败和服务重启中断；
 *   - 不发请求、不订阅SSE、不持有取消逻辑。
 */
import type { CoursewareAssemblyRuntimeStatus } from '@/api/coursewareAssembly'

import { C } from './workshopConstants'

export type AssemblyStageState =
  | 'pending'
  | 'running'
  | 'ok'
  | 'skipped'
  | 'failed'

export type AssemblyRuntimeStatus = CoursewareAssemblyRuntimeStatus

export interface AssemblyPageState {
  page_number: number
  title: string
  html: AssemblyStageState
  image: AssemblyStageState
  video: AssemblyStageState
  note?: string
}

export interface AssemblySummary {
  total_pages: number
  skip_video: boolean
  running: boolean
  done: boolean
  runtime_status: AssemblyRuntimeStatus
  message: string
  restored?: boolean
  html_success?: number
  html_fail?: number
  image_success?: number
  image_fail?: number
  image_skip?: number
  video_success?: number
  video_skip?: number
  elapsed_ms?: number
  errors?: string[]
}

interface Props {
  summary: AssemblySummary
  pages: AssemblyPageState[]
}

const STAGE_BADGE: Record<
  AssemblyStageState,
  {
    label: string
    color: string
    bg: string
    dot?: boolean
  }
> = {
  pending: {
    label: '待处理',
    color: '#6B7280',
    bg: '#F3F4F6',
  },
  running: {
    label: '进行中',
    color: '#2563EB',
    bg: '#DBEAFE',
    dot: true,
  },
  ok: {
    label: '完成',
    color: '#059669',
    bg: '#D1FAE5',
  },
  skipped: {
    label: '无需',
    color: '#9CA3AF',
    bg: '#F3F4F6',
  },
  failed: {
    label: '失败',
    color: '#DC2626',
    bg: '#FEE2E2',
  },
}

const RUNTIME_BADGE: Record<
  AssemblyRuntimeStatus,
  {
    label: string
    icon: string
    color: string
    bg: string
  }
> = {
  idle: {
    label: '尚未启动',
    icon: '⚪',
    color: '#6B7280',
    bg: '#F3F4F6',
  },
  starting: {
    label: '正在启动',
    icon: '⏳',
    color: '#2563EB',
    bg: '#DBEAFE',
  },
  running: {
    label: '后台装配中',
    icon: '⚡',
    color: '#7C3AED',
    bg: '#EDE9FE',
  },
  cancel_requested: {
    label: '正在停止',
    icon: '⏸',
    color: '#D97706',
    bg: '#FEF3C7',
  },
  completed: {
    label: '装配完成',
    icon: '✅',
    color: '#059669',
    bg: '#D1FAE5',
  },
  cancelled: {
    label: '已取消',
    icon: '⏸',
    color: '#D97706',
    bg: '#FEF3C7',
  },
  failed: {
    label: '装配失败',
    icon: '❌',
    color: '#DC2626',
    bg: '#FEE2E2',
  },
  interrupted: {
    label: '服务重启中断',
    icon: '⚠️',
    color: '#B45309',
    bg: '#FEF3C7',
  },
}

function StageBadge({
  icon,
  label,
  state,
}: {
  icon: string
  label: string
  state: AssemblyStageState
}) {
  const badge = STAGE_BADGE[state]

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 5,
        padding: '3px 8px',
        borderRadius: 6,
        background: badge.bg,
      }}
    >
      <span style={{ fontSize: 12 }}>{icon}</span>
      <span
        style={{
          fontSize: 11,
          color: C.textSecondary,
        }}
      >
        {label}
      </span>
      <span
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 4,
          fontSize: 11,
          fontWeight: 600,
          color: badge.color,
        }}
      >
        {badge.dot && (
          <span
            style={{
              display: 'inline-block',
              width: 6,
              height: 6,
              borderRadius: '50%',
              background: badge.color,
              animation:
                'cwAsmPulse 1s ease-in-out infinite',
            }}
          />
        )}
        {state === 'ok' ? '✓' : badge.label}
      </span>
    </div>
  )
}

function PageCard({
  page,
  skipVideo,
}: {
  page: AssemblyPageState
  skipVideo: boolean
}) {
  const anyFailed =
    page.html === 'failed' ||
    page.image === 'failed' ||
    page.video === 'failed'
  const htmlDone = page.html === 'ok'
  const imageSettled =
    page.image === 'ok' ||
    page.image === 'skipped'
  const videoSettled =
    skipVideo ||
    page.video === 'ok' ||
    page.video === 'skipped'
  const allSettled =
    htmlDone && imageSettled && videoSettled
  const anyRunning =
    page.html === 'running' ||
    page.image === 'running' ||
    page.video === 'running'

  const edge = anyFailed
    ? '#EF4444'
    : allSettled
      ? '#10B981'
      : anyRunning
        ? '#3B82F6'
        : '#E5E7EB'

  return (
    <div
      style={{
        border: `1px solid ${C.border}`,
        borderLeft: `4px solid ${edge}`,
        borderRadius: 10,
        padding: '12px 14px',
        background: '#fff',
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        transition: 'border-color 300ms',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
        }}
      >
        <span
          style={{
            flexShrink: 0,
            width: 26,
            height: 26,
            borderRadius: 7,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 12,
            fontWeight: 700,
            color: C.primary,
            background: C.primaryBg,
          }}
        >
          {page.page_number}
        </span>
        <span
          style={{
            fontSize: 13,
            fontWeight: 600,
            color: C.textPrimary,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
          title={page.title}
        >
          {page.title ||
            `第 ${page.page_number} 页`}
        </span>
      </div>

      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: 6,
        }}
      >
        <StageBadge
          icon="📝"
          label="HTML"
          state={page.html}
        />
        <StageBadge
          icon="🖼"
          label="配图"
          state={page.image}
        />
        {!skipVideo && (
          <StageBadge
            icon="🎬"
            label="视频"
            state={page.video}
          />
        )}
      </div>

      {page.note && (
        <div
          style={{
            fontSize: 11.5,
            color: C.textMuted,
            lineHeight: 1.5,
          }}
        >
          {page.note}
        </div>
      )}
    </div>
  )
}

export default function AssemblyProgressView({
  summary,
  pages,
}: Props) {
  const settledCount = pages.filter(page => {
    const htmlSettled =
      page.html === 'ok' ||
      page.html === 'failed' ||
      page.html === 'skipped'
    const imageSettled =
      page.image === 'ok' ||
      page.image === 'skipped' ||
      page.image === 'failed'
    const videoSettled =
      summary.skip_video ||
      page.video === 'ok' ||
      page.video === 'skipped' ||
      page.video === 'failed'

    return (
      htmlSettled &&
      imageSettled &&
      videoSettled
    )
  }).length

  const total =
    summary.total_pages || pages.length
  const percent =
    total > 0
      ? Math.round(
          (settledCount / total) * 100,
        )
      : 0
  const elapsedSec =
    summary.elapsed_ms != null
      ? (
          summary.elapsed_ms / 1000
        ).toFixed(1)
      : null
  const runtime =
    RUNTIME_BADGE[summary.runtime_status]
  const hasStats = [
    summary.html_success,
    summary.html_fail,
    summary.image_success,
    summary.image_fail,
    summary.image_skip,
    summary.video_success,
    summary.video_skip,
  ].some(value => value !== undefined)

  return (
    <div>
      <style>
        {`@keyframes cwAsmPulse{0%,100%{opacity:1}50%{opacity:0.3}}`}
      </style>

      <div style={{ marginBottom: 16 }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            marginBottom: 8,
            flexWrap: 'wrap',
          }}
        >
          <span
            style={{
              padding: '3px 10px',
              borderRadius: 12,
              fontSize: 12,
              fontWeight: 600,
              color: summary.skip_video
                ? '#0891B2'
                : '#7C3AED',
              background: summary.skip_video
                ? '#CFFAFE'
                : '#EDE9FE',
            }}
          >
            {summary.skip_video
              ? '🖼 HTML+配图（不做视频）'
              : '⚡ 全自动装配'}
          </span>

          <span
            style={{
              padding: '3px 10px',
              borderRadius: 12,
              fontSize: 12,
              fontWeight: 600,
              color: runtime.color,
              background: runtime.bg,
            }}
          >
            {runtime.icon} {runtime.label}
          </span>

          {summary.restored &&
            summary.running && (
              <span
                style={{
                  fontSize: 12,
                  color: C.textMuted,
                }}
              >
                已从后台运行状态恢复
              </span>
            )}

          {summary.done &&
            summary.runtime_status ===
              'completed' &&
            elapsedSec && (
              <span
                style={{
                  fontSize: 12,
                  color: C.success,
                }}
              >
                耗时 {elapsedSec}s
              </span>
            )}
        </div>

        {summary.running && (
          <div
            style={{
              marginBottom: 8,
              fontSize: 12,
              color: C.textMuted,
            }}
          >
            装配在后台运行。刷新或关闭页面不会丢失已经落库的结果。
          </div>
        )}

        {summary.message && (
          <div
            style={{
              fontSize: 13,
              color: C.textSecondary,
              lineHeight: 1.6,
              marginBottom: 10,
            }}
          >
            {summary.message}
          </div>
        )}

        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            gap: 12,
            flexWrap: 'wrap',
            fontSize: 12,
            color: C.textSecondary,
            marginBottom: 6,
          }}
        >
          <span>装配进度（HTML与媒体并行推进）</span>
          <span>
            共 {total} 页 · 已结算{' '}
            <b style={{ color: C.success }}>
              {settledCount}
            </b>{' '}
            页
          </span>
        </div>

        <div
          style={{
            height: 8,
            borderRadius: 4,
            background: '#F3F4F6',
            overflow: 'hidden',
          }}
        >
          <div
            style={{
              height: '100%',
              borderRadius: 4,
              transition: 'width 500ms',
              width: `${percent}%`,
              background:
                'linear-gradient(90deg, #7C3AED, #2563EB)',
            }}
          />
        </div>
      </div>

      {pages.length > 0 ? (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns:
              'repeat(auto-fill, minmax(220px, 1fr))',
            gap: 12,
          }}
        >
          {pages.map(page => (
            <PageCard
              key={page.page_number}
              page={page}
              skipVideo={summary.skip_video}
            />
          ))}
        </div>
      ) : (
        <div
          style={{
            textAlign: 'center',
            padding: '32px 0',
            color: C.textMuted,
            fontSize: 13,
          }}
        >
          正在同步装配状态…
        </div>
      )}

      {summary.done && hasStats && (
        <div
          style={{
            marginTop: 16,
            padding: '14px 16px',
            borderRadius: 10,
            background: '#F9FAFB',
            border: `1px solid ${C.border}`,
          }}
        >
          <div
            style={{
              fontSize: 13,
              fontWeight: 600,
              color: C.textPrimary,
              marginBottom: 8,
            }}
          >
            装配汇总
          </div>
          <div
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              gap: '6px 18px',
              fontSize: 12.5,
              color: C.textSecondary,
              lineHeight: 1.8,
            }}
          >
            <span>
              📝 HTML 成功{' '}
              <b style={{ color: C.success }}>
                {summary.html_success ?? 0}
              </b>{' '}
              页
              {(summary.html_fail ?? 0) > 0
                ? `，失败 ${summary.html_fail} 页`
                : ''}
            </span>
            <span>
              🖼 配图成功{' '}
              <b style={{ color: C.success }}>
                {summary.image_success ?? 0}
              </b>{' '}
              页
              {(summary.image_skip ?? 0) > 0
                ? `，无需 ${summary.image_skip} 页`
                : ''}
              {(summary.image_fail ?? 0) > 0
                ? `，失败 ${summary.image_fail} 页`
                : ''}
            </span>
            {!summary.skip_video && (
              <span>
                🎬 视频占位{' '}
                <b style={{ color: C.success }}>
                  {summary.video_success ?? 0}
                </b>{' '}
                页
                {(summary.video_skip ?? 0) > 0
                  ? `，无需 ${summary.video_skip} 页`
                  : ''}
              </span>
            )}
          </div>
        </div>
      )}

      {summary.errors &&
        summary.errors.length > 0 && (
          <details
            style={{
              marginTop: 12,
              padding: '10px 12px',
              borderRadius: 8,
              background: '#FEF2F2',
            }}
          >
            <summary
              style={{
                fontSize: 12,
                color: C.danger,
                cursor: 'pointer',
              }}
            >
              查看 {summary.errors.length} 条失败详情
            </summary>
            <div
              style={{
                marginTop: 6,
                maxHeight: 160,
                overflow: 'auto',
                display: 'flex',
                flexDirection: 'column',
                gap: 4,
              }}
            >
              {summary.errors.map(
                (error, index) => (
                  <div
                    key={index}
                    style={{
                      fontSize: 11.5,
                      color: '#991B1B',
                      background: '#fff',
                      borderRadius: 6,
                      padding: '5px 8px',
                      lineHeight: 1.5,
                    }}
                  >
                    {error}
                  </div>
                ),
              )}
            </div>
          </details>
        )}
    </div>
  )
}

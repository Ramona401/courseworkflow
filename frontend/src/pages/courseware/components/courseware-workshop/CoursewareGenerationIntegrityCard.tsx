/**
 * CoursewareGenerationIntegrityCard.tsx — R-04 课件页数完整性教师视图
 *
 * 事实边界：
 *   - 所有数量、页面分类与 complete 均直接使用服务端 reconciliation；
 *   - 前端不得通过 pages.length、SSE 百分比或 html_content 自行推导 complete；
 *   - 只展示教师安全的 page_id 对应标题、页码和服务端安全原因；
 *   - 数据库实际页数与方案期望页数不一致时必须明确告警，不得用 page_count 掩盖。
 */

import type {
  CoursewareAssemblyState,
  CoursewareGenerationIntegrity,
  CoursewareGenerationPageRef,
  CoursewareGenerationRunKind,
} from '@/api/coursewareAssembly'

import {
  getCoursewareAssemblyIntegrity,
} from './coursewareAssemblyStateFacts'
import { C } from './workshopConstants'

interface Props {
  state: CoursewareAssemblyState | null
  expectedRunKind?: CoursewareGenerationRunKind
  busy?: boolean
  retryLabel?: string
  onRetry?: () => void
  onRefresh?: () => void
}

interface MetricProps {
  label: string
  value: number
  emphasis?: 'normal' | 'success' | 'warning' | 'danger'
}

function metricColor(emphasis: MetricProps['emphasis']): string {
  switch (emphasis) {
    case 'success':
      return '#047857'
    case 'warning':
      return '#B45309'
    case 'danger':
      return '#B91C1C'
    default:
      return C.textPrimary
  }
}

function Metric({ label, value, emphasis = 'normal' }: MetricProps) {
  return (
    <div
      style={{
        minWidth: 104,
        flex: '1 1 104px',
        padding: '10px 12px',
        borderRadius: 9,
        border: `1px solid ${C.border}`,
        background: '#fff',
      }}
    >
      <div style={{ marginBottom: 4, color: C.textMuted, fontSize: 11.5 }}>{label}</div>
      <div style={{ color: metricColor(emphasis), fontSize: 18, fontWeight: 700 }}>{value}</div>
    </div>
  )
}

function PageList({
  title,
  icon,
  pages,
  tone,
}: {
  title: string
  icon: string
  pages: CoursewareGenerationPageRef[]
  tone: 'danger' | 'warning'
}) {
  if (pages.length === 0) return null

  const border = tone === 'danger' ? '#FECACA' : '#FDE68A'
  const background = tone === 'danger' ? '#FEF2F2' : '#FFFBEB'
  const heading = tone === 'danger' ? '#991B1B' : '#92400E'

  return (
    <div style={{ padding: '10px 12px', borderRadius: 9, border: `1px solid ${border}`, background }}>
      <div style={{ marginBottom: 7, color: heading, fontSize: 12.5, fontWeight: 700 }}>
        {icon} {title} · {pages.length} 页
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
        {pages.map(page => (
          <div
            key={page.page_id}
            style={{
              padding: '7px 9px',
              borderRadius: 7,
              background: '#fff',
              color: C.textSecondary,
              fontSize: 12,
              lineHeight: 1.55,
            }}
          >
            <span style={{ color: C.textPrimary, fontWeight: 600 }}>
              P{page.page_number} · {page.title || `第 ${page.page_number} 页`}
            </span>
            {page.reason && <span> — {page.reason}</span>}
          </div>
        ))}
      </div>
    </div>
  )
}

function integrityTitle(
  integrity: CoursewareGenerationIntegrity,
  state: CoursewareAssemblyState,
): {
  icon: string
  text: string
  color: string
  background: string
  border: string
} {
  if (integrity.complete) {
    return {
      icon: '✅',
      text: '页数已完整对账',
      color: '#047857',
      background: '#ECFDF5',
      border: '#A7F3D0',
    }
  }

  if (state.is_active) {
    return {
      icon: '🔄',
      text: '正在生成并持续对账',
      color: '#1D4ED8',
      background: '#EFF6FF',
      border: '#BFDBFE',
    }
  }

  return {
    icon: '⚠️',
    text: '未完整生成',
    color: '#B45309',
    background: '#FFFBEB',
    border: '#FDE68A',
  }
}

export default function CoursewareGenerationIntegrityCard({
  state,
  expectedRunKind,
  busy = false,
  retryLabel = '只补生成缺失页',
  onRetry,
  onRefresh,
}: Props) {
  const integrity = getCoursewareAssemblyIntegrity(
    state,
    expectedRunKind,
  )

  if (!state || !integrity) {
    return null
  }

  const title = integrityTitle(integrity, state)
  const countMismatch = integrity.actual_page_count !== integrity.expected_count
  const retryableCount =
    integrity.failed_count +
    integrity.cancelled_count +
    integrity.missing_count
  const canRetry =
    Boolean(onRetry) &&
    !state.is_active &&
    !integrity.complete &&
    retryableCount > 0

  return (
    <div
      style={{
        marginBottom: 16,
        padding: '14px 16px',
        borderRadius: 11,
        border: `1px solid ${title.border}`,
        background: title.background,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent: 'space-between',
          gap: 12,
          flexWrap: 'wrap',
          marginBottom: 12,
        }}
      >
        <div>
          <div style={{ color: title.color, fontSize: 14, fontWeight: 700 }}>
            {title.icon} {title.text}
          </div>
          <div style={{ marginTop: 4, color: C.textSecondary, fontSize: 12, lineHeight: 1.6 }}>
            完整性以本次运行启动时冻结的稳定页面清单和服务端最终对账为准。
          </div>
        </div>

        {onRefresh && (
          <button
            type="button"
            disabled={busy}
            onClick={onRefresh}
            style={{
              padding: '6px 11px',
              borderRadius: 7,
              border: `1px solid ${C.border}`,
              background: '#fff',
              color: C.textSecondary,
              cursor: busy ? 'not-allowed' : 'pointer',
              fontSize: 12,
              opacity: busy ? 0.6 : 1,
            }}
          >
            {busy ? '同步中…' : '🔄 重新对账'}
          </button>
        )}
      </div>

      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 12 }}>
        <Metric label="方案期望" value={integrity.expected_count} />
        <Metric
          label="数据库实际"
          value={integrity.actual_page_count}
          emphasis={countMismatch ? 'danger' : 'normal'}
        />
        <Metric label="成功" value={integrity.success_count} emphasis="success" />
        <Metric
          label="失败"
          value={integrity.failed_count}
          emphasis={integrity.failed_count > 0 ? 'danger' : 'normal'}
        />
        <Metric
          label="取消"
          value={integrity.cancelled_count}
          emphasis={integrity.cancelled_count > 0 ? 'warning' : 'normal'}
        />
        <Metric
          label="缺失"
          value={integrity.missing_count}
          emphasis={integrity.missing_count > 0 ? 'danger' : 'normal'}
        />
        {integrity.pending_count > 0 && (
          <Metric label="待处理" value={integrity.pending_count} emphasis="warning" />
        )}
      </div>

      {countMismatch && (
        <div
          style={{
            marginBottom: 10,
            padding: '9px 11px',
            borderRadius: 8,
            border: '1px solid #FCA5A5',
            background: '#FEF2F2',
            color: '#991B1B',
            fontSize: 12.5,
            lineHeight: 1.6,
          }}
        >
          ⚠️ 数据库当前实际有 <b>{integrity.actual_page_count}</b> 页，当前方案要求{' '}
          <b>{integrity.expected_count}</b> 页，页数尚未对齐。系统不会通过修改 page_count 掩盖这项差异。
        </div>
      )}

      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <PageList title="生成或校验失败" icon="❌" pages={integrity.failed_pages} tone="danger" />
        <PageList title="任务停止前尚未成功" icon="⏸" pages={integrity.cancelled_pages} tone="warning" />
        <PageList title="当前方案缺失" icon="⚠️" pages={integrity.missing_pages} tone="danger" />
      </div>

      {canRetry && (
        <div
          style={{
            marginTop: 12,
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            flexWrap: 'wrap',
          }}
        >
          <button
            type="button"
            disabled={busy}
            onClick={onRetry}
            style={{
              padding: '9px 16px',
              borderRadius: 8,
              border: 'none',
              background: busy ? '#D1D5DB' : '#2563EB',
              color: '#fff',
              cursor: busy ? 'not-allowed' : 'pointer',
              fontSize: 13,
              fontWeight: 700,
            }}
          >
            {busy ? '正在提交…' : retryLabel}
          </button>

          <span style={{ color: C.textMuted, fontSize: 11.5, lineHeight: 1.55 }}>
            再次提交前服务端会重新校验当前方案和页面状态；已经成功的页面不会被覆盖。
          </span>
        </div>
      )}
    </div>
  )
}

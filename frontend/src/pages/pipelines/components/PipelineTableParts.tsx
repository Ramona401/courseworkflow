/**
 * PipelineTableParts.tsx — Pipeline表格内部组件
 *   StatusBadge / ProgressBar / ScoreCell / SortableHeader / Pagination
 */
import {
  Clock, Loader, Eye, CheckCircle, AlertTriangle,
  XCircle, Square, ArrowUp, ArrowDown,
  ChevronsLeft, ChevronsRight, ChevronLeft, ChevronRight,
} from 'lucide-react'
import type { SortField, SortDirection } from './pipelinesConstants'
import { PAGE_SIZE_OPTIONS } from './pipelinesConstants'

// ==================== 状态徽章（紧凑版）====================

const STATUS_COLOR_MAP: Record<string, { bg: string; fg: string }> = {
  pending:          { bg: 'rgba(142,142,147,0.12)', fg: '#8e8e93' },
  running:          { bg: 'rgba(0,122,255,0.12)',   fg: '#007aff' },
  review_queue:     { bg: 'rgba(255,149,0,0.12)',   fg: '#ff9500' },
  finalized:        { bg: 'rgba(52,199,89,0.12)',   fg: '#34c759' },
  needs_human:      { bg: 'rgba(255,204,0,0.12)',   fg: '#cc9900' },
  failed:           { bg: 'rgba(255,59,48,0.12)',   fg: '#ff3b30' },
  cancelled:        { bg: 'rgba(142,142,147,0.08)', fg: '#aeaeb2' },
  verified:         { bg: 'rgba(52,199,89,0.15)',   fg: '#248a3d' },
  verify_failed:    { bg: 'rgba(255,59,48,0.15)',   fg: '#d70015' },
  pending_finalize: { bg: 'rgba(204,102,0,0.12)',   fg: '#cc6600' },
  published:        { bg: 'rgba(88,86,214,0.12)',   fg: '#5856d6' },
}

export function StatusBadge({ status, statusName, reviewRound }: { status: string; statusName: string; reviewRound?: number }) {
  const c = STATUS_COLOR_MAP[status] || STATUS_COLOR_MAP.pending
  const iconMap: Record<string, React.ReactNode> = {
    pending:          <Clock size={11} />,
    running:          <Loader size={11} style={{ animation: 'spin 1s linear infinite' }} />,
    review_queue:     <Eye size={11} />,
    finalized:        <CheckCircle size={11} />,
    needs_human:      <AlertTriangle size={11} />,
    failed:           <XCircle size={11} />,
    cancelled:        <Square size={11} />,
    verified:         <CheckCircle size={11} />,
    verify_failed:    <XCircle size={11} />,
    pending_finalize: <Clock size={11} />,
  }
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 3,
      padding: '2px 8px', borderRadius: 20, fontSize: 11, fontWeight: 500,
      background: c.bg, color: c.fg, whiteSpace: 'nowrap',
    }}>
      {iconMap[status]}{statusName}{reviewRound && reviewRound >= 2 ? <span style={{ marginLeft: 3, fontSize: 9, fontWeight: 700, padding: '0 3px', borderRadius: 3, background: 'rgba(0,0,0,0.08)' }}>R{reviewRound}</span> : null}
    </span>
  )
}

// ==================== 进度条（紧凑版）====================

export function ProgressBar({ completed, total }: { completed: number; total: number }) {
  const pct = total > 0 ? (completed / total) * 100 : 0
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <div style={{ flex: 1, height: 5, background: 'rgba(0,0,0,0.06)', borderRadius: 3, overflow: 'hidden', minWidth: 50 }}>
        <div style={{
          width: pct + '%', height: '100%',
          background: completed === total ? '#34c759' : '#007aff',
          borderRadius: 3, transition: 'width 0.3s ease',
        }} />
      </div>
      <span style={{ fontSize: 11, color: '#8e8e93', fontWeight: 500, whiteSpace: 'nowrap' }}>
        {completed}/{total}
      </span>
    </div>
  )
}

// ==================== 分数单元格 ====================

export function ScoreCell({ value }: { value: number | null }) {
  if (value === null || value === undefined) {
    return <span style={{ color: '#c7c7cc', fontSize: 12 }}>-</span>
  }
  let color = '#ff3b30'
  if (value >= 9.0) color = '#34c759'
  else if (value >= 7.0) color = '#ff9500'
  return (
    <span style={{ color, fontSize: 13, fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>
      {value.toFixed(1)}
    </span>
  )
}

// ==================== 可排序表头 ====================

export function SortableHeader({ label, field, currentSort, currentDir, onSort, align }: {
  label: string
  field: SortField
  currentSort: SortField
  currentDir: SortDirection
  onSort: (field: SortField) => void
  align?: 'center' | 'left'
}) {
  const isActive = currentSort === field
  return (
    <span
      onClick={e => { e.stopPropagation(); onSort(field) }}
      style={{
        cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 3,
        userSelect: 'none',
        color: isActive ? '#007aff' : '#8e8e93',
        justifyContent: align === 'center' ? 'center' : 'flex-start',
      }}
      title={`点击按${label}排序`}
    >
      {label}
      {isActive
        ? (currentDir === 'asc' ? <ArrowUp size={14} strokeWidth={2.5} /> : <ArrowDown size={14} strokeWidth={2.5} />)
        : <ArrowDown size={12} style={{ opacity: 0.4 }} />
      }
    </span>
  )
}

// ==================== 分页器 ====================

export function Pagination({ currentPage, totalPages, totalItems, pageSize, onPageChange, onPageSizeChange }: {
  currentPage: number
  totalPages: number
  totalItems: number
  pageSize: number
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
}) {
  if (totalItems === 0) return null

  const startItem = (currentPage - 1) * pageSize + 1
  const endItem   = Math.min(currentPage * pageSize, totalItems)

  const pgBtn: React.CSSProperties = {
    width: 32, height: 32, borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
    cursor: 'pointer', fontSize: 13, fontWeight: 500, color: '#3c3c43',
    transition: 'all 0.15s ease',
  }
  const pgBtnDisabled: React.CSSProperties = {
    ...pgBtn, opacity: 0.3, cursor: 'not-allowed', color: '#aeaeb2',
  }

  // 最多显示5个页码按钮
  const pages: number[] = []
  let start = Math.max(1, currentPage - 2)
  let end   = Math.min(totalPages, start + 4)
  if (end - start < 4) start = Math.max(1, end - 4)
  for (let i = start; i <= end; i++) pages.push(i)

  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
      padding: '12px 16px', borderTop: '1px solid rgba(0,0,0,0.06)',
      background: 'rgba(249,249,249,0.8)', borderRadius: '0 0 16px 16px',
      flexWrap: 'wrap', gap: 12,
    }}>
      {/* 左侧：显示信息 + 每页条数 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, fontSize: 12, color: '#8e8e93' }}>
        <span>共 <b style={{ color: '#1c1c1e' }}>{totalItems}</b> 条，显示 {startItem}-{endItem}</span>
        <span style={{ color: '#d1d1d6' }}>|</span>
        <span>每页</span>
        <select
          value={pageSize} onChange={e => onPageSizeChange(Number(e.target.value))}
          style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid rgba(0,0,0,0.1)', fontSize: 12, background: '#fff', cursor: 'pointer', outline: 'none', color: '#1c1c1e', fontWeight: 500 }}>
          {PAGE_SIZE_OPTIONS.map(size => (
            <option key={size} value={size}>{size} 条</option>
          ))}
        </select>
      </div>

      {/* 右侧：翻页按钮 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        <button onClick={() => currentPage > 1 && onPageChange(1)} disabled={currentPage <= 1}
          style={currentPage <= 1 ? pgBtnDisabled : pgBtn} title="首页">
          <ChevronsLeft size={14} />
        </button>
        <button onClick={() => currentPage > 1 && onPageChange(currentPage - 1)} disabled={currentPage <= 1}
          style={currentPage <= 1 ? pgBtnDisabled : pgBtn} title="上一页">
          <ChevronLeft size={14} />
        </button>
        {pages.map(p => (
          <button key={p} onClick={() => onPageChange(p)} style={{
            ...pgBtn,
            background: p === currentPage ? '#007aff' : '#fff',
            color:      p === currentPage ? '#fff'    : '#3c3c43',
            border:     p === currentPage ? '1px solid #007aff' : '1px solid rgba(0,0,0,0.08)',
            fontWeight: p === currentPage ? 700 : 500,
          }}>{p}</button>
        ))}
        <button onClick={() => currentPage < totalPages && onPageChange(currentPage + 1)} disabled={currentPage >= totalPages}
          style={currentPage >= totalPages ? pgBtnDisabled : pgBtn} title="下一页">
          <ChevronRight size={14} />
        </button>
        <button onClick={() => currentPage < totalPages && onPageChange(totalPages)} disabled={currentPage >= totalPages}
          style={currentPage >= totalPages ? pgBtnDisabled : pgBtn} title="末页">
          <ChevronsRight size={14} />
        </button>
      </div>
    </div>
  )
}

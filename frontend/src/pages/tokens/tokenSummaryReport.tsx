/**
 * tokenSummaryReport.tsx — 积分消费汇总报告组件
 *
 * 积分消费汇总报告 batch（前端）：
 *   挂在 TokenDashboardPage 的"📈 汇总报告"Tab。后端 6 维度聚合，前端按角色给维度入口。
 *
 * 维度入口（角色差异化）：
 *   - 超级管理员      → 区域/学校/老师/模型/场景/趋势
 *   - 二线管理员      → 区域/学校/老师/场景/趋势，不含模型和成本
 *   - region_admin   → 学校/老师/场景/趋势
 *   - senior_operator→ 老师/场景/趋势
 *   美元成本列仅超级管理员展示。
 *
 * 三层下钻（你要的"汇总→点进去看明细"）：
 *   第1层 维度排行（默认）
 *     ├ 点学校行 → 第2层：该校各老师（school_filter）
 *     ├ 点老师行 → 第2.5层：该老师按模型/场景分布（user_filter，Q2=b）
 *     │            └ 点"查看明细流水" → 第3层：原始流水（复用 ConsumptionTable）
 *   面包屑导航可逐层返回。
 *
 * 趋势图：纯 CSS div 柱状图（不引入 recharts，避免新依赖），hover 显示数值。
 *
 * 重名处理：同 label（如两个"李老师"）→ 展示时附 user_id 后4位区分。
 */
import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '@/store/auth'
import { loadAliasResolver } from '@/api/modelAliasResolve'
import {
  getConsumptionSummary, getTokenConsumption,
  type ConsumptionSummaryRow, type ConsumptionListItem,
} from '@/api/tokens'
import { C, StatCard, ConsumptionTable, Pagination, cardStyle } from './tokenDashboardParts'

// 维度定义
type Dim = 'region' | 'school' | 'user' | 'model' | 'scene' | 'time'
const DIM_LABELS: Record<Dim, string> = {
  region: '🗺️ 按区域', school: '🏫 按学校', user: '👤 按老师',
  model: '🤖 按模型', scene: '🎬 按场景', time: '📅 按趋势',
}

// 时间范围快捷选项
type RangeKey = 'all' | '7d' | '30d' | 'month'
const RANGE_LABELS: Record<RangeKey, string> = {
  all: '全部', '7d': '近7天', '30d': '近30天', month: '本月',
}

// 计算快捷范围的 from/to（YYYY-MM-DD）
function calcRange(key: RangeKey): { from?: string; to?: string } {
  const today = new Date()
  // 本地日期拼接(避免 toISOString 返回 UTC 导致北京时间凌晨差一天)
  const fmt = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  if (key === 'all') return {}
  if (key === '7d') {
    const f = new Date(today); f.setDate(f.getDate() - 6)
    return { from: fmt(f), to: fmt(today) }
  }
  if (key === '30d') {
    const f = new Date(today); f.setDate(f.getDate() - 29)
    return { from: fmt(f), to: fmt(today) }
  }
  // month：本月1号到今天
  const f = new Date(today.getFullYear(), today.getMonth(), 1)
  return { from: fmt(f), to: fmt(today) }
}

const PAGE_SIZE = 20

// 下钻层级
type DrillLevel =
  | { kind: 'summary' }                                          // 第1层：维度排行
  | { kind: 'schoolUsers'; schoolId: string; schoolName: string } // 第2层：某校各老师
  | { kind: 'userBreakdown'; userId: string; userName: string; subDim: 'model' | 'scene' } // 第2.5层：某老师模型/场景
  | { kind: 'detail'; userId: string; userName: string }          // 第3层：明细流水

export default function TokenSummaryReport() {
  const { user } = useAuth()
  const role = user?.role || ''
  const isAdmin = role === 'admin'
  const isSuperAdmin = isAdmin && user?.is_super === true
  const isRegionAdmin = role === 'region_admin'
  const isSchoolAdmin = role === 'senior_operator'

  // 角色可见维度入口：模型维度只向超级管理员开放。
  const visibleDims: Dim[] = isSuperAdmin
    ? ['region', 'school', 'user', 'model', 'scene', 'time']
    : isAdmin
      ? ['region', 'school', 'user', 'scene', 'time']
      : isRegionAdmin
      ? ['school', 'user', 'scene', 'time']
      : isSchoolAdmin
        ? ['user', 'scene', 'time']
        : ['scene', 'time'] // 兜底（普通角色一般不进此报告）
  // 非超级管理员下钻默认走scene，不暴露真实模型维度。
  const defaultSubDim: 'model' | 'scene' = isSuperAdmin ? 'model' : 'scene'

  const [dim, setDim] = useState<Dim>(visibleDims[0])
  const [range, setRange] = useState<RangeKey>('all')
  const [rows, setRows] = useState<ConsumptionSummaryRow[]>([])
  const [totalCredits, setTotalCredits] = useState(0)
  const [totalCostUSD, setTotalCostUSD] = useState(0)
  const [totalCalls, setTotalCalls] = useState(0)
  const [loading, setLoading] = useState(false)
  const [scopeMsg, setScopeMsg] = useState('')
  const [drill, setDrill] = useState<DrillLevel>({ kind: 'summary' })

  // 第3层明细流水 state
  const [detailItems, setDetailItems] = useState<ConsumptionListItem[]>([])
  const [detailTotal, setDetailTotal] = useState(0)
  const [detailPage, setDetailPage] = useState(1)

  // 仅超级管理员加载模型别名解析器。
  const [resolveAlias, setResolveAlias] = useState<((m: string) => string) | null>(null)
  useEffect(() => {
    if (!isSuperAdmin) { setResolveAlias(null); return }
    let alive = true
    loadAliasResolver().then(fn => { if (alive) setResolveAlias(() => fn) })
    return () => { alive = false }
  }, [isSuperAdmin])

  // 加载汇总（第1层/第2层/第2.5层都走 getConsumptionSummary，差别在 dimension 和 filter）
  const loadSummary = useCallback(async () => {
    setLoading(true); setScopeMsg('')
    const r = calcRange(range)
    try {
      let params: Parameters<typeof getConsumptionSummary>[0]
      if (drill.kind === 'schoolUsers') {
        params = { dimension: 'user', school_filter: drill.schoolId, ...r }
      } else if (drill.kind === 'userBreakdown') {
        params = { dimension: drill.subDim, user_filter: drill.userId, ...r }
      } else {
        params = { dimension: dim, ...r }
      }
      const data = await getConsumptionSummary(params)
      setRows(data?.rows || [])
      setTotalCredits(data?.total_credits || 0)
      setTotalCostUSD(data?.total_cost_usd ?? 0)
      setTotalCalls(data?.total_calls || 0)
      if (data?.scope_message) setScopeMsg(data.scope_message)
    } catch { setRows([]) }
    setLoading(false)
  }, [dim, range, drill])

  // 加载第3层明细流水（复用 getTokenConsumption）
  const loadDetail = useCallback(async () => {
    if (drill.kind !== 'detail') return
    setLoading(true)
    const offset = (detailPage - 1) * PAGE_SIZE
    try {
      const data = await getTokenConsumption({ user_id: drill.userId, limit: PAGE_SIZE, offset })
      setDetailItems(data?.items || [])
      setDetailTotal(data?.total || 0)
    } catch { setDetailItems([]) }
    setLoading(false)
  }, [drill, detailPage])

  useEffect(() => {
    if (drill.kind === 'detail') loadDetail()
    else loadSummary()
  }, [drill, loadSummary, loadDetail])

  // 切维度/时间范围 → 回到第1层
  const switchDim = (d: Dim) => { setDim(d); setDrill({ kind: 'summary' }) }
  const switchRange = (rk: RangeKey) => { setRange(rk); setDrill({ kind: 'summary' }) }

  // 重名检测：同 label 出现多次时，附 user_id 后4位
  const labelCounts = rows.reduce<Record<string, number>>((acc, r) => { acc[r.label] = (acc[r.label] || 0) + 1; return acc }, {})
  // 当前是否模型维度；只有超级管理员可以到达该维度。
  const isModelView = dim === 'model' || (drill.kind === 'userBreakdown' && drill.subDim === 'model')
  const displayLabel = (r: ConsumptionSummaryRow) => {
    // 模型维度且已加载解析器：把真实模型名转业务别名（不暴露真名）
    const base = (isModelView && resolveAlias) ? resolveAlias(r.label) : r.label
    return labelCounts[r.label] > 1 ? `${base} #${r.key.slice(-4)}` : base
  }

  // 点击行的下钻逻辑（取决于当前维度/层级）
  const onRowClick = (r: ConsumptionSummaryRow) => {
    if (drill.kind === 'summary') {
      if (dim === 'school') setDrill({ kind: 'schoolUsers', schoolId: r.key, schoolName: r.label })
      else if (dim === 'user') setDrill({ kind: 'userBreakdown', userId: r.key, userName: r.label, subDim: defaultSubDim })
      // region/model/scene/time 维度第1层不下钻（region 看区域对比；model/scene/time 是末层洞察）
    } else if (drill.kind === 'schoolUsers') {
      // 校内某老师 → 老师模型/场景分布
      setDrill({ kind: 'userBreakdown', userId: r.key, userName: r.label, subDim: defaultSubDim })
    }
    // userBreakdown 层的行（模型/场景）不再下钻
  }

  const rowClickable = (drill.kind === 'summary' && (dim === 'school' || dim === 'user')) || drill.kind === 'schoolUsers'

  // ============ 渲染 ============
  return (
    <div>
      {scopeMsg && (
        <div style={{ marginBottom: '16px', padding: '12px 16px', borderRadius: '12px', background: 'rgba(245,158,11,0.08)', border: `1px solid ${C.orange}`, color: C.orange, fontSize: '13px' }}>
          ⚠️ {scopeMsg}
        </div>
      )}

      {/* 总览卡片 */}
      <div style={{ display: 'flex', gap: '16px', marginBottom: '20px', flexWrap: 'wrap' }}>
        <StatCard label="总消费积分" value={`${totalCredits.toLocaleString(undefined, { maximumFractionDigits: 2 })}`} color={C.primary} />
        {isSuperAdmin && <StatCard label="总成本(USD)" value={`$${totalCostUSD.toFixed(2)}`} color={C.orange} />}
        <StatCard label="总调用次数" value={totalCalls.toLocaleString()} color={C.purple} />
      </div>

      {/* 维度切换 + 时间范围 */}
      <div style={{ display: 'flex', gap: '8px', marginBottom: '12px', flexWrap: 'wrap' }}>
        {visibleDims.map(d => (
          <button key={d} onClick={() => switchDim(d)}
            style={{ padding: '8px 16px', borderRadius: '8px', border: 'none', cursor: 'pointer', fontSize: '13px',
              fontWeight: dim === d ? 600 : 400, color: dim === d ? C.white : C.textSec,
              background: dim === d ? C.primary : C.primaryLight, transition: 'all 150ms ease' }}>
            {DIM_LABELS[d]}
          </button>
        ))}
      </div>
      <div style={{ display: 'flex', gap: '6px', marginBottom: '20px', flexWrap: 'wrap', alignItems: 'center' }}>
        <span style={{ fontSize: '12px', color: C.textMuted, marginRight: '4px' }}>时间范围:</span>
        {(['all', '7d', '30d', 'month'] as RangeKey[]).map(rk => (
          <button key={rk} onClick={() => switchRange(rk)}
            style={{ padding: '5px 12px', borderRadius: '14px', cursor: 'pointer', fontSize: '12px',
              fontWeight: range === rk ? 600 : 400,
              border: `1px solid ${range === rk ? C.primary : C.border}`,
              background: range === rk ? C.primary : C.white,
              color: range === rk ? C.white : C.textSec }}>
            {RANGE_LABELS[rk]}
          </button>
        ))}
      </div>

      {/* 面包屑（下钻时显示） */}
      {drill.kind !== 'summary' && (
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px', fontSize: '13px', flexWrap: 'wrap' }}>
          <button onClick={() => setDrill({ kind: 'summary' })}
            style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: C.white, color: C.primary, cursor: 'pointer', fontSize: '12px' }}>
            ← {DIM_LABELS[dim]}
          </button>
          {drill.kind === 'schoolUsers' && <span style={{ color: C.textSec }}>▸ {drill.schoolName} 的老师</span>}
          {drill.kind === 'userBreakdown' && (
            <>
              <span style={{ color: C.textSec }}>▸ {drill.userName}</span>
              <button onClick={() => setDrill({ kind: 'detail', userId: drill.userId, userName: drill.userName })}
                style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.primary}`, background: C.primaryLight, color: C.primary, cursor: 'pointer', fontSize: '12px', marginLeft: '4px' }}>
                查看明细流水 →
              </button>
              {/* 模型/场景子维度切换 */}
              <div style={{ display: 'flex', gap: '4px', marginLeft: '8px' }}>
                {(isSuperAdmin ? (['model', 'scene'] as ('model' | 'scene')[]) : (['scene'] as ('model' | 'scene')[])).map(sd => (
                  <button key={sd} onClick={() => setDrill({ ...drill, subDim: sd })}
                    style={{ padding: '3px 10px', borderRadius: '12px', cursor: 'pointer', fontSize: '11px',
                      border: `1px solid ${drill.subDim === sd ? C.purple : C.border}`,
                      background: drill.subDim === sd ? C.purple : C.white,
                      color: drill.subDim === sd ? C.white : C.textSec }}>
                    {sd === 'model' ? '按模型' : '按场景'}
                  </button>
                ))}
              </div>
            </>
          )}
          {drill.kind === 'detail' && (
            <>
              <button onClick={() => setDrill({ kind: 'userBreakdown', userId: drill.userId, userName: drill.userName, subDim: defaultSubDim })}
                style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: C.white, color: C.primary, cursor: 'pointer', fontSize: '12px' }}>
                ← {drill.userName} 分布
              </button>
              <span style={{ color: C.textSec }}>▸ 明细流水</span>
            </>
          )}
        </div>
      )}

      {/* 内容区 */}
      {loading ? (
        <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px' }}>加载中...</div>
      ) : drill.kind === 'detail' ? (
        <>
          <ConsumptionTable items={detailItems} isSuperAdmin={isSuperAdmin} />
          <Pagination page={detailPage} pageSize={PAGE_SIZE} total={detailTotal} onChange={setDetailPage} />
        </>
      ) : dim === 'time' && drill.kind === 'summary' ? (
        <BarChart rows={rows} />
      ) : (
        <RankList rows={rows} totalCredits={totalCredits} isSuperAdmin={isSuperAdmin}
          displayLabel={displayLabel} clickable={rowClickable} onRowClick={onRowClick}
          scopeHint={drill.kind === 'schoolUsers' ? `${drill.schoolName} 全校` : drill.kind === 'userBreakdown' ? `${drill.userName} 个人` : undefined} />
      )}
    </div>
  )
}

// ============ 排行榜（占比条）============
function RankList({ rows, totalCredits, isSuperAdmin, displayLabel, clickable, onRowClick, scopeHint }: {
  rows: ConsumptionSummaryRow[]; totalCredits: number; isSuperAdmin: boolean
  displayLabel: (r: ConsumptionSummaryRow) => string
  clickable: boolean; onRowClick: (r: ConsumptionSummaryRow) => void
  scopeHint?: string
}) {
  if (!rows.length) return <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px' }}>暂无数据</div>
  const maxCredits = Math.max(...rows.map(r => r.credits), 1)
  return (
    <div style={{ background: C.white, borderRadius: '12px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
      {scopeHint && (
        <div style={{ padding: '8px 18px', fontSize: '11px', color: C.textMuted, background: C.bg, borderBottom: `1px solid ${C.border}` }}>
          ℹ️ 占比与总额基于：{scopeHint}
        </div>
      )}
      {rows.map((r, i) => (
        <div key={r.key + i}
          onClick={() => clickable && onRowClick(r)}
          style={{ padding: '14px 18px', borderBottom: i < rows.length - 1 ? `1px solid ${C.border}` : 'none',
            cursor: clickable ? 'pointer' : 'default', transition: 'background 120ms ease' }}
          onMouseEnter={e => { if (clickable) (e.currentTarget as HTMLDivElement).style.background = C.primaryLight }}
          onMouseLeave={e => { (e.currentTarget as HTMLDivElement).style.background = 'transparent' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '6px' }}>
            <span style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>
              <span style={{ color: C.textMuted, marginRight: '8px', fontSize: '12px' }}>#{i + 1}</span>
              {displayLabel(r)}
              {clickable && <span style={{ color: C.primary, fontSize: '11px', marginLeft: '6px' }}>点击下钻 ›</span>}
            </span>
            <span style={{ display: 'flex', gap: '16px', alignItems: 'baseline' }}>
              <span style={{ fontSize: '15px', fontWeight: 700, color: C.primary }}>{r.credits.toLocaleString(undefined, { maximumFractionDigits: 2 })}</span>
              {isSuperAdmin && <span style={{ fontSize: '12px', color: C.orange }}>${(r.cost_usd ?? 0).toFixed(2)}</span>}
              <span style={{ fontSize: '12px', color: C.textSec }}>{r.calls.toLocaleString()}次</span>
              <span style={{ fontSize: '12px', color: C.textMuted, width: '44px', textAlign: 'right' }}>{r.percent.toFixed(1)}%</span>
            </span>
          </div>
          <div style={{ height: '6px', background: '#F3F4F6', borderRadius: '3px', overflow: 'hidden' }}>
            <div style={{ width: `${(r.credits / maxCredits) * 100}%`, height: '100%', background: C.primary, borderRadius: '3px', transition: 'width 300ms ease' }} />
          </div>
        </div>
      ))}
    </div>
  )
}

// ============ CSS 柱状图（趋势）============
function BarChart({ rows }: { rows: ConsumptionSummaryRow[] }) {
  if (!rows.length) return <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px' }}>暂无数据</div>
  const maxC = Math.max(...rows.map(r => r.credits), 1)
  return (
    <div style={{ ...cardStyle, flex: 'unset' }}>
      <div style={{ display: 'flex', alignItems: 'flex-end', gap: '3px', height: '220px', overflowX: 'auto', paddingBottom: '8px' }}>
        {rows.map(r => (
          <div key={r.key} title={`${r.label}\n${r.credits.toFixed(2)} 积分 / ${r.calls}次`}
            style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', minWidth: '22px', flex: '1 0 auto', cursor: 'default' }}>
            <div style={{ fontSize: '9px', color: C.textMuted, marginBottom: '2px', whiteSpace: 'nowrap' }}>
              {r.credits >= 1 ? Math.round(r.credits) : ''}
            </div>
            <div style={{ width: '70%', height: `${(r.credits / maxC) * 180}px`, minHeight: r.credits > 0 ? '2px' : '0',
              background: `linear-gradient(180deg, ${C.primary}, ${C.purple})`, borderRadius: '3px 3px 0 0', transition: 'height 300ms ease' }} />
            <div style={{ fontSize: '8px', color: C.textMuted, marginTop: '4px', whiteSpace: 'nowrap', transform: 'rotate(-45deg)', transformOrigin: 'center', height: '36px' }}>
              {r.label.split('-').slice(1).join('-')}
            </div>
          </div>
        ))}
      </div>
      <div style={{ fontSize: '11px', color: C.textMuted, textAlign: 'center', marginTop: '8px' }}>
        悬停柱子查看每日消费详情（共 {rows.length} 天）
      </div>
    </div>
  )
}

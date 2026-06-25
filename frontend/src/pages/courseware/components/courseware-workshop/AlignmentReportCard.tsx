/**
 * AlignmentReportCard.tsx — 课件↔教案对齐报告卡片（插在 Step1 确认方案）
 *
 * 作用：比对"课件逐页方案是否忠实还原教案教学意图"，给老师明确信号：
 *   ✅已对齐 / ⚠️小幅偏差 / ❗需注意 / ⚪校验失败。可展开看覆盖度/AI新增/意图偏移明细。
 *
 * 显示逻辑（关键）：
 *   - 仅 lesson_plan 来源课件显示本卡片（其它来源无教案可比对，整体不渲染）。
 *   - 有报告：展示结论+明细+重新校验。
 *   - 无报告（老课件部署前生成、从未校验）：显示"🔍 立即校验方案与教案对齐度"主动触发按钮，
 *     点击后调 recheckAlignment 触发校验并轮询，老课件无需重新生成方案即可使用本功能。
 *
 * 数据：挂载即 GET 一次；generating 时每3秒短轮询（最多6次≈18秒）。
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import { getAlignmentReport, recheckAlignment } from '@/api/coursewares'
import type { CoursewareAlignmentReport, AlignmentResultJSON } from '@/api/coursewares'
import { CW_ALIGNMENT_OVERALL_CONFIG, CW_ALIGNMENT_COVERAGE_CONFIG } from '@/api/coursewares'
import { C } from './workshopConstants'

interface Props {
  coursewareId: string
  /** 课件来源类型：仅 'lesson_plan' 显示本卡片 */
  sourceType: string
}

const POLL_INTERVAL_MS = 3000
const POLL_MAX_TIMES = 6

export default function AlignmentReportCard({ coursewareId, sourceType }: Props) {
  const [report, setReport] = useState<CoursewareAlignmentReport | null>(null)
  const [hasReport, setHasReport] = useState(false)
  const [loading, setLoading] = useState(true)
  const [expanded, setExpanded] = useState(false)
  const [rechecking, setRechecking] = useState(false)
  const pollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const pollCountRef = useRef(0)

  const clearPoll = useCallback(() => {
    if (pollTimerRef.current) { clearTimeout(pollTimerRef.current); pollTimerRef.current = null }
  }, [])

  const fetchOnce = useCallback(async () => {
    try {
      const res = await getAlignmentReport(coursewareId)
      setHasReport(res.has_report)
      setReport(res.report)
      setLoading(false)
      if (res.has_report && res.report && res.report.status === 'generating' && pollCountRef.current < POLL_MAX_TIMES) {
        pollCountRef.current += 1
        pollTimerRef.current = setTimeout(fetchOnce, POLL_INTERVAL_MS)
      }
    } catch {
      setLoading(false)
    }
  }, [coursewareId])

  useEffect(() => {
    pollCountRef.current = 0
    setLoading(true)
    fetchOnce()
    return clearPoll
  }, [coursewareId, fetchOnce, clearPoll])

  const handleRecheck = async () => {
    if (rechecking) return
    setRechecking(true)
    clearPoll()
    try {
      await recheckAlignment(coursewareId)
      pollCountRef.current = 0
      setReport(prev => prev ? { ...prev, status: 'generating' } : prev)
      setHasReport(true)
      pollTimerRef.current = setTimeout(fetchOnce, 1500)
    } catch {
      /* 静默 */
    } finally {
      setRechecking(false)
    }
  }

  // 非教案来源：整体不渲染
  if (sourceType !== 'lesson_plan') return null

  // 首屏加载占位
  if (loading && !report) {
    return (
      <div style={cardWrap}>
        <div style={{ fontSize: 13, color: C.textMuted }}>🔍 正在读取对齐校验状态…</div>
      </div>
    )
  }

  // 无报告（老课件从未校验）：显示主动触发入口
  if (!hasReport && !report) {
    return (
      <div style={{ ...cardWrap, background: 'linear-gradient(135deg, #F5F3FF, #FFF)', borderColor: '#DDD6FE' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap' }}>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 14, fontWeight: 600, color: '#6D28D9' }}>🔍 课件方案 ↔ 教案对齐校验</div>
            <div style={{ fontSize: 12, color: C.textSecondary, marginTop: 4, lineHeight: 1.5 }}>
              让 AI 比对这份课件方案是否忠实还原了教案的教学意图，指出遗漏环节、AI 新增、教学目标偏移。
            </div>
          </div>
          <button onClick={handleRecheck} disabled={rechecking}
            style={{ padding: '10px 20px', borderRadius: 9, border: 'none', whiteSpace: 'nowrap',
              background: rechecking ? '#E5E7EB' : 'linear-gradient(135deg, #7C3AED, #6D28D9)',
              color: rechecking ? '#9CA3AF' : '#fff', fontSize: 14, fontWeight: 600,
              cursor: rechecking ? 'default' : 'pointer', boxShadow: rechecking ? 'none' : '0 2px 8px rgba(124,58,237,0.3)' }}>
            {rechecking ? '启动中…' : '开始校验'}
          </button>
        </div>
      </div>
    )
  }

  if (!report) return null

  // generating 态
  if (report.status === 'generating') {
    return (
      <div style={cardWrap}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 14 }}>🔍</span>
          <span style={{ fontSize: 13, color: C.textSecondary }}>AI 正在比对课件方案与教案教学意图，请稍候（约 10-15 秒）…</span>
        </div>
      </div>
    )
  }

  // failed 态
  if (report.status === 'failed' || report.overall === 'failed') {
    const cfg = CW_ALIGNMENT_OVERALL_CONFIG.failed
    return (
      <div style={{ ...cardWrap, borderColor: cfg.bg }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8, flexWrap: 'wrap' }}>
          <span style={{ fontSize: 13, color: C.textSecondary }}>
            {cfg.emoji} 对齐校验未完成{report.error_message ? `：${report.error_message}` : ''}
          </span>
          <button onClick={handleRecheck} disabled={rechecking} style={recheckBtn}>
            {rechecking ? '重试中…' : '🔄 重新校验'}
          </button>
        </div>
      </div>
    )
  }

  // done 态
  let parsed: AlignmentResultJSON | null = null
  try { parsed = JSON.parse(report.report_json) as AlignmentResultJSON } catch { parsed = null }

  const overallCfg = CW_ALIGNMENT_OVERALL_CONFIG[report.overall] || CW_ALIGNMENT_OVERALL_CONFIG.minor
  const missingCount = parsed?.coverage?.filter(c => c.status === 'missing').length || 0
  const partialCount = parsed?.coverage?.filter(c => c.status === 'partial').length || 0
  const additionsCount = parsed?.additions?.length || 0
  const shiftsCount = parsed?.intent_shifts?.length || 0

  return (
    <div style={{ ...cardWrap, borderColor: overallCfg.bg, borderWidth: 2 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10, flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flex: 1, minWidth: 0 }}>
          <span style={{ padding: '3px 12px', borderRadius: 14, fontSize: 13, fontWeight: 700, color: overallCfg.color, background: overallCfg.bg, whiteSpace: 'nowrap' }}>
            {overallCfg.emoji} {overallCfg.label}
          </span>
          <span style={{ fontSize: 13, color: C.textSecondary, lineHeight: 1.5, overflow: 'hidden', textOverflow: 'ellipsis' }} title={report.summary}>
            {report.summary || '已完成方案与教案的对齐校验'}
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, whiteSpace: 'nowrap' }}>
          <button onClick={handleRecheck} disabled={rechecking} style={recheckBtn}>
            {rechecking ? '重算中…' : '🔄 重新校验'}
          </button>
          {parsed && (missingCount + partialCount + additionsCount + shiftsCount > 0) && (
            <button onClick={() => setExpanded(e => !e)} style={expandBtn}>
              {expanded ? '收起明细 ▲' : '查看明细 ▼'}
            </button>
          )}
        </div>
      </div>

      {parsed && !expanded && (missingCount + partialCount + additionsCount + shiftsCount > 0) && (
        <div style={{ marginTop: 8, display: 'flex', gap: 12, flexWrap: 'wrap', fontSize: 12, color: C.textMuted }}>
          {missingCount > 0 && <span style={{ color: '#DC2626' }}>✕ 遗漏环节 {missingCount}</span>}
          {partialCount > 0 && <span style={{ color: '#D97706' }}>◐ 简略环节 {partialCount}</span>}
          {additionsCount > 0 && <span>ℹ️ AI新增 {additionsCount}</span>}
          {shiftsCount > 0 && <span style={{ color: '#DC2626' }}>↔ 意图偏移 {shiftsCount}</span>}
        </div>
      )}

      {expanded && parsed && (
        <div style={{ marginTop: 14, display: 'flex', flexDirection: 'column', gap: 16 }}>
          {parsed.coverage && parsed.coverage.length > 0 && (
            <div>
              <div style={sectionTitle}>📋 教案环节覆盖度</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                {parsed.coverage.map((c, i) => {
                  const cc = CW_ALIGNMENT_COVERAGE_CONFIG[c.status] || CW_ALIGNMENT_COVERAGE_CONFIG.partial
                  return (
                    <div key={i} style={{ display: 'flex', alignItems: 'flex-start', gap: 8, padding: '6px 10px', borderRadius: 8, background: c.status === 'missing' ? '#FEF2F2' : c.status === 'partial' ? '#FFFBEB' : '#F9FAFB' }}>
                      <span style={{ fontSize: 12, fontWeight: 600, color: cc.color, background: cc.bg, padding: '1px 8px', borderRadius: 10, whiteSpace: 'nowrap' }}>{cc.emoji} {cc.label}</span>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <span style={{ fontSize: 13, color: C.textPrimary, fontWeight: 500 }}>{c.plan_segment}</span>
                        {c.page_nums && c.page_nums.length > 0 && (
                          <span style={{ fontSize: 12, color: C.textMuted, marginLeft: 8 }}>对应 P{c.page_nums.join(' P')}</span>
                        )}
                        {c.note && <div style={{ fontSize: 12, color: C.textSecondary, marginTop: 2 }}>{c.note}</div>}
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {parsed.additions && parsed.additions.length > 0 && (
            <div>
              <div style={sectionTitle}>ℹ️ 课件新增内容（教案中没有，请确认是否合适）</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                {parsed.additions.map((a, i) => (
                  <div key={i} style={{ fontSize: 13, color: C.textSecondary, padding: '4px 10px', background: '#F9FAFB', borderRadius: 6 }}>
                    <b style={{ color: C.textPrimary }}>P{a.page_num}</b> · {a.desc}
                  </div>
                ))}
              </div>
            </div>
          )}

          {parsed.intent_shifts && parsed.intent_shifts.length > 0 && (
            <div>
              <div style={sectionTitle}>↔ 教学意图偏移（课件目的与教案目标有出入）</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                {parsed.intent_shifts.map((s, i) => (
                  <div key={i} style={{ fontSize: 13, padding: '6px 10px', background: '#FEF2F2', borderRadius: 8 }}>
                    <div style={{ fontWeight: 600, color: C.textPrimary }}>P{s.page_num}</div>
                    <div style={{ fontSize: 12, color: C.textSecondary, marginTop: 2 }}>教案目标：{s.plan_intent}</div>
                    <div style={{ fontSize: 12, color: C.textSecondary }}>课件目的：{s.scheme_purpose}</div>
                    {s.note && <div style={{ fontSize: 12, color: '#DC2626', marginTop: 2 }}>⚠️ {s.note}</div>}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

const cardWrap: React.CSSProperties = {
  marginTop: 16, marginBottom: 4, padding: '14px 16px', borderRadius: 12,
  border: '1.5px solid #E5E7EB', background: '#fff',
}
const recheckBtn: React.CSSProperties = {
  padding: '5px 12px', borderRadius: 7, border: '1px solid #E5E7EB',
  background: 'transparent', color: '#6B7280', fontSize: 12, cursor: 'pointer', whiteSpace: 'nowrap',
}
const expandBtn: React.CSSProperties = {
  padding: '5px 12px', borderRadius: 7, border: '1px solid #7C3AED',
  background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 12, fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap',
}
const sectionTitle: React.CSSProperties = {
  fontSize: 13, fontWeight: 600, color: '#374151', marginBottom: 8,
}

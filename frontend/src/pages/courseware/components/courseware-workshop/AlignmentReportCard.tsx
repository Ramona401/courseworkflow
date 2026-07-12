/**
 * AlignmentReportCard.tsx — 课件↔教案对齐报告卡片（插在 Step1 确认方案）
 *
 * 【修正前后对比】记住修正前的问题数（prevIssues），校验完成后如果有改善则显示绿色改善提示。
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import { getAlignmentReport, recheckAlignment } from '@/api/coursewares'
import type { CoursewareAlignmentReport, AlignmentResultJSON } from '@/api/coursewares'
import { CW_ALIGNMENT_OVERALL_CONFIG, CW_ALIGNMENT_COVERAGE_CONFIG } from '@/api/coursewares'
import { C } from './workshopConstants'

interface Props {
  coursewareId: string
  sourceType: string
  onAutoFix?: (instruction: string) => void
}

const POLL_INTERVAL_MS = 3000
const POLL_MAX_TIMES = 6

/** 把对齐报告中的问题拼成给 refineCWIndex 的修正指令 */
function buildFixInstruction(parsed: AlignmentResultJSON): string {
  const parts: string[] = ['请根据以下对齐校验结果修正课件方案：\n']

  // 收集所有涉及问题的页码（用于末尾显式标注"哪些页需要改"）
  const affectedPageNums = new Set<number>()

  const missing = (parsed.coverage || []).filter(c => c.status === 'missing')
  if (missing.length > 0) {
    parts.push('【未覆盖的教案环节】需要在方案中补充对应页面：')
    missing.forEach(c => {
      parts.push(`  - "${c.plan_segment}"${c.note ? '（' + c.note + '）' : ''}`)
      // missing 没有关联页码（因为就是没覆盖），补充页面由AI决定位置
    })
    parts.push('')
  }
  const partial = (parsed.coverage || []).filter(c => c.status === 'partial')
  if (partial.length > 0) {
    parts.push('【部分覆盖的教案环节】需要加强覆盖：')
    partial.forEach(c => {
      const pageInfo = c.page_nums && c.page_nums.length > 0 ? `（当前在 P${c.page_nums.join('、P')}）` : ''
      parts.push(`  - "${c.plan_segment}"${pageInfo}${c.note ? '，' + c.note : ''}`)
      // partial 关联的页码需要修改
      if (c.page_nums) c.page_nums.forEach(n => affectedPageNums.add(n))
    })
    parts.push('')
  }
  const shifts = parsed.intent_shifts || []
  if (shifts.length > 0) {
    parts.push('【教学意图偏移】以下页面的方案目的与教案不一致，请修正：')
    shifts.forEach(s => {
      parts.push(`  - P${s.page_num}：教案目标是"${s.plan_intent}"，但课件目的写成了"${s.scheme_purpose}"${s.note ? '。' + s.note : ''}`)
      affectedPageNums.add(s.page_num)
    })
    parts.push('')
  }

  // 显式标注需要修改的页码和不需要修改的页码——让AI严格只改有问题的页
  if (affectedPageNums.size > 0) {
    const sortedNums = Array.from(affectedPageNums).sort((a, b) => a - b)
    parts.push(`⚠️ 【严格要求】仅修改以下页面的方案：${sortedNums.map(n => 'P' + n).join('、')}。`)
    parts.push('其余所有页面必须原样保留——title、purpose、content_summary、interaction_type、visual_format、media_requirements、estimated_complexity 逐字不改地照抄原方案输出。')
    parts.push('如果需要新增页面来补充缺失的教案环节，请插入新页面，但不要修改已有的无问题页面。')
  } else {
    // 只有 missing（无关联页码的纯新增需求），让AI只新增不改旧页
    parts.push('⚠️ 【严格要求】现有所有页面的方案必须原样保留不做任何修改（title、purpose、content_summary等所有字段逐字照抄），仅在合适位置插入新页面来补充缺失的教案环节。')
  }

  return parts.join('\n')
}

/** 统计报告中的问题数 */
function countIssues(parsed: AlignmentResultJSON | null): { missing: number; partial: number; shifts: number; total: number } {
  if (!parsed) return { missing: 0, partial: 0, shifts: 0, total: 0 }
  const missing = (parsed.coverage || []).filter(c => c.status === 'missing').length
  const partial = (parsed.coverage || []).filter(c => c.status === 'partial').length
  const shifts = (parsed.intent_shifts || []).length
  return { missing, partial, shifts, total: missing + partial + shifts }
}

export default function AlignmentReportCard({ coursewareId, sourceType, onAutoFix }: Props) {
  const [report, setReport] = useState<CoursewareAlignmentReport | null>(null)
  const [hasReport, setHasReport] = useState(false)
  const [loading, setLoading] = useState(true)
  const [expanded, setExpanded] = useState(false)
  const [rechecking, setRechecking] = useState(false)
  const [fixConfirming, setFixConfirming] = useState(false)
  // 修正前的问题数快照——点"确认修正"时记录，重新校验完成后用于对比显示改善效果
  const [prevIssues, setPrevIssues] = useState<{ missing: number; partial: number; shifts: number; total: number } | null>(null)
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
    setFixConfirming(false)
    fetchOnce()
    return clearPoll
  }, [coursewareId, fetchOnce, clearPoll])

  const handleRecheck = async () => {
    if (rechecking) return
    setRechecking(true); clearPoll()
    try {
      await recheckAlignment(coursewareId)
      pollCountRef.current = 0
      setReport(prev => prev ? { ...prev, status: 'generating' } : prev)
      setHasReport(true)
      pollTimerRef.current = setTimeout(fetchOnce, 1500)
    } catch { /* 静默 */ }
    finally { setRechecking(false) }
  }

  if (sourceType !== 'lesson_plan') return null

  if (loading && !report) {
    return <div style={cardWrap}><div style={{ fontSize: 13, color: C.textMuted }}>🔍 正在读取对齐校验状态…</div></div>
  }

  if (!hasReport && !report) {
    return (
      <div style={{ ...cardWrap, background: 'linear-gradient(135deg, #F5F3FF, #FFF)', borderColor: '#DDD6FE' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap' }}>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 14, fontWeight: 600, color: '#6D28D9' }}>🔍 课件方案 ↔ 教案对齐校验</div>
            <div style={{ fontSize: 12, color: C.textSecondary, marginTop: 4, lineHeight: 1.5 }}>让 AI 比对这份课件方案是否忠实还原了教案的教学意图，指出遗漏环节、AI 新增、教学目标偏移。</div>
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

  if (report.status === 'generating') {
    return <div style={cardWrap}><div style={{ display: 'flex', alignItems: 'center', gap: 8 }}><span style={{ fontSize: 14 }}>🔍</span><span style={{ fontSize: 13, color: C.textSecondary }}>AI 正在比对课件方案与教案教学意图，请稍候（约 10-15 秒）…</span></div></div>
  }

  if (report.status === 'failed' || report.overall === 'failed') {
    const cfg = CW_ALIGNMENT_OVERALL_CONFIG.failed
    return (
      <div style={{ ...cardWrap, borderColor: cfg.bg }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8, flexWrap: 'wrap' }}>
          <span style={{ fontSize: 13, color: C.textSecondary }}>{cfg.emoji} 对齐校验未完成{report.error_message ? `：${report.error_message}` : ''}</span>
          <button onClick={handleRecheck} disabled={rechecking} style={recheckBtn}>{rechecking ? '重试中…' : '🔄 重新校验'}</button>
        </div>
      </div>
    )
  }

  // ========== done 态 ==========
  let parsed: AlignmentResultJSON | null = null
  try { parsed = JSON.parse(report.report_json) as AlignmentResultJSON } catch { parsed = null }

  const overallCfg = CW_ALIGNMENT_OVERALL_CONFIG[report.overall] || CW_ALIGNMENT_OVERALL_CONFIG.minor
  const issues = countIssues(parsed)
  const additionsCount = parsed?.additions?.length || 0
  const hasFixableIssues = issues.total > 0
  const showAutoFix = !!onAutoFix && !!parsed && report.overall !== 'aligned' && hasFixableIssues

  // 构建修正前后对比文案（prevIssues 非空说明刚经历了一轮修正+重新校验）
  const improvementParts: string[] = []
  if (prevIssues) {
    if (prevIssues.shifts > 0 && issues.shifts === 0) improvementParts.push(`意图偏移 ${prevIssues.shifts}→0 ✅`)
    else if (prevIssues.shifts > issues.shifts) improvementParts.push(`意图偏移 ${prevIssues.shifts}→${issues.shifts}`)
    if (prevIssues.missing > 0 && issues.missing === 0) improvementParts.push(`遗漏环节 ${prevIssues.missing}→0 ✅`)
    else if (prevIssues.missing > issues.missing) improvementParts.push(`遗漏环节 ${prevIssues.missing}→${issues.missing}`)
    if (prevIssues.partial > 0 && issues.partial === 0) improvementParts.push(`简略环节 ${prevIssues.partial}→0 ✅`)
    else if (prevIssues.partial > issues.partial) improvementParts.push(`简略环节 ${prevIssues.partial}→${issues.partial}`)
  }
  const hasImprovement = improvementParts.length > 0

  /** AI自动修正：记住当前问题数 → 构建指令 → 冒泡给父组件执行 */
  const doAutoFix = () => {
    if (!parsed || !onAutoFix) return
    const instruction = buildFixInstruction(parsed)
    if (!instruction.trim()) return
    // 记住修正前的问题数，供重新校验后对比
    setPrevIssues(countIssues(parsed))
    setFixConfirming(false)
    onAutoFix(instruction)
  }

  return (
    <div style={{ ...cardWrap, borderColor: overallCfg.bg, borderWidth: 2 }}>
      {/* 修正效果改善提示条——修正+重新校验后如果问题数减少，显示绿色对比 */}
      {hasImprovement && (
        <div style={{ marginBottom: 10, padding: '8px 14px', borderRadius: 8, background: '#F0FDF4', border: '1px solid #86EFAC', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8 }}>
          <span style={{ fontSize: 13, color: '#166534', fontWeight: 500 }}>
            🎯 修正效果：{improvementParts.join('，')}
            {report.overall === 'aligned' ? '  — 方案已完全对齐教案！' : ''}
            {report.overall !== 'aligned' && additionsCount > 0 && issues.total === 0 ? '  — 仅剩 AI 新增内容（非教案遗漏，可酌情保留）' : ''}
          </span>
          <button onClick={() => setPrevIssues(null)} style={{ background: 'none', border: 'none', fontSize: 14, cursor: 'pointer', color: '#166534', opacity: 0.5, padding: '0 4px' }}>✕</button>
        </div>
      )}

      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10, flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flex: 1, minWidth: 0 }}>
          <span style={{ padding: '3px 12px', borderRadius: 14, fontSize: 13, fontWeight: 700, color: overallCfg.color, background: overallCfg.bg, whiteSpace: 'nowrap' }}>
            {overallCfg.emoji} {overallCfg.label}
          </span>
          <span style={{ fontSize: 13, color: C.textSecondary, lineHeight: 1.5, overflow: 'hidden', textOverflow: 'ellipsis' }} title={report.summary}>
            {report.summary || '已完成方案与教案的对齐校验'}
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, whiteSpace: 'nowrap', flexWrap: 'wrap' }}>
          {showAutoFix && !fixConfirming && (
            <button onClick={() => setFixConfirming(true)} style={autoFixBtn}>🔧 AI修正方案</button>
          )}
          <button onClick={handleRecheck} disabled={rechecking} style={recheckBtn}>{rechecking ? '重算中…' : '🔄 重新校验'}</button>
          {parsed && (issues.total + additionsCount > 0) && (
            <button onClick={() => setExpanded(e => !e)} style={expandBtn}>{expanded ? '收起明细 ▲' : '查看明细 ▼'}</button>
          )}
        </div>
      </div>

      {/* AI修正内联确认区 */}
      {fixConfirming && (
        <div style={{ marginTop: 12, padding: '12px 16px', borderRadius: 10, background: '#FFF7ED', border: '1.5px solid #FB923C' }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: '#9A3412', marginBottom: 6 }}>🔧 确认AI修正方案</div>
          <div style={{ fontSize: 12, color: '#78350F', lineHeight: 1.6, marginBottom: 10 }}>
            AI 将根据对齐报告自动修正方案：补充遗漏的教案环节、修正教学意图偏移的页面。修正完成后会自动重新校验对齐度。方案未变化的页面会保留已生成的课件，方案有变化的页面需要在 Step4 重新生成。
          </div>
          <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
            <button onClick={() => setFixConfirming(false)} style={{ padding: '6px 16px', borderRadius: 7, border: '1px solid #D1D5DB', background: '#fff', color: '#6B7280', fontSize: 13, cursor: 'pointer' }}>取消</button>
            <button onClick={doAutoFix} style={{ padding: '6px 18px', borderRadius: 7, border: 'none', background: 'linear-gradient(135deg, #F59E0B, #EA580C)', color: '#fff', fontSize: 13, fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 6px rgba(245,158,11,0.3)' }}>确认修正</button>
          </div>
        </div>
      )}

      {/* 折叠摘要 */}
      {parsed && !expanded && (issues.total + additionsCount > 0) && (
        <div style={{ marginTop: 8, display: 'flex', gap: 12, flexWrap: 'wrap', fontSize: 12, color: C.textMuted }}>
          {issues.missing > 0 && <span style={{ color: '#DC2626' }}>✕ 遗漏环节 {issues.missing}</span>}
          {issues.partial > 0 && <span style={{ color: '#D97706' }}>◐ 简略环节 {issues.partial}</span>}
          {additionsCount > 0 && <span>ℹ️ AI新增 {additionsCount}</span>}
          {issues.shifts > 0 && <span style={{ color: '#DC2626' }}>↔ 意图偏移 {issues.shifts}</span>}
        </div>
      )}

      {/* 展开明细 */}
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
                        {c.page_nums && c.page_nums.length > 0 && <span style={{ fontSize: 12, color: C.textMuted, marginLeft: 8 }}>对应 P{c.page_nums.join(' P')}</span>}
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
          {showAutoFix && !fixConfirming && (
            <div style={{ display: 'flex', justifyContent: 'center', paddingTop: 8 }}>
              <button onClick={() => setFixConfirming(true)} style={{ ...autoFixBtn, padding: '10px 24px', fontSize: 14 }}>🔧 根据以上问题，让 AI 自动修正方案</button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

const cardWrap: React.CSSProperties = { marginTop: 16, marginBottom: 4, padding: '14px 16px', borderRadius: 12, border: '1.5px solid #E5E7EB', background: '#fff' }
const recheckBtn: React.CSSProperties = { padding: '5px 12px', borderRadius: 7, border: '1px solid #E5E7EB', background: 'transparent', color: '#6B7280', fontSize: 12, cursor: 'pointer', whiteSpace: 'nowrap' }
const expandBtn: React.CSSProperties = { padding: '5px 12px', borderRadius: 7, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 12, fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap' }
const autoFixBtn: React.CSSProperties = { padding: '6px 14px', borderRadius: 8, border: 'none', background: 'linear-gradient(135deg, #F59E0B, #EA580C)', color: '#fff', fontSize: 13, fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap', boxShadow: '0 2px 6px rgba(245,158,11,0.25)' }
const sectionTitle: React.CSSProperties = { fontSize: 13, fontWeight: 600, color: '#374151', marginBottom: 8 }

/**
 * SchemeSteps.tsx — 课件工坊 Step0(AI生成方案)+Step1(确认方案)（批次5b-2从主页面拆出）
 *
 * 拆出范围：两步的全部JSX + 专属7个state/2个effect/3个处理函数
 * （生成方案四来源分流/SSE订阅/10秒兜底轮询/确认方案/AI修改方案/方案预设选择）。
 *
 * 与父级的接缝：
 *   - pages/setPages：方案页面列表是父级真相源（Step4派生统计/loadCourseware恢复都用），传下来；
 *   - sseRef：全工坊共享的单一SSE连接句柄（父级持有，卸载时父级统一close）；
 *   - goToStep/loadCourseware：步骤推进与课件刷新；
 *   - onCoursewareUpdate：兜底轮询发现完成时把最新课件对象回写父级状态。
 */
import { useState, useEffect } from 'react'
import type { Dispatch, SetStateAction, MutableRefObject } from 'react'
import {
  getCourseware, generateCWIndex, generateCWIndexFromTopic, generateCWIndexFromPPT,
  generateCWIndexFromDoc, subscribeCWIndexSSE, confirmCWIndex, refineCWIndex, getSchemePresets,
} from '@/api/coursewares'
import type { SchemePreset, CoursewareDetail, CoursewarePage } from '@/api/coursewares'
import IndexEditor from '../IndexEditor'
import AlignmentReportCard from './AlignmentReportCard'
import { C } from './workshopConstants'
import { MsgBar } from './PagePreviewBlock'

interface Props {
  coursewareId: string
  courseware: CoursewareDetail
  isAdmin: boolean
  /** 0=生成方案 1=确认方案（其余值不渲染任何内容） */
  activeStep: number
  pages: CoursewarePage[]
  setPages: Dispatch<SetStateAction<CoursewarePage[]>>
  sseRef: MutableRefObject<{ close: () => void } | null>
  goToStep: (n: number) => void
  loadCourseware: () => void
  /** 兜底轮询发现完成时回写父级 courseware 状态 */
  onCoursewareUpdate: (d: CoursewareDetail) => void
}

export default function SchemeSteps({ coursewareId, courseware, isAdmin, activeStep, pages, setPages, sseRef, goToStep, loadCourseware, onCoursewareUpdate }: Props) {
  // ==================== 两步专属状态（5b-2自主页面整体迁入） ====================
  const [generating, setGenerating] = useState(false)
  const [sseMessage, setSseMessage] = useState('')
  const [confirming, setConfirming] = useState(false)
  // v136: 方案预设+AI修改方案
  const [presets, setPresets] = useState<SchemePreset[]>([])
  const [selectedPreset, setSelectedPreset] = useState('auto')
  const [refineFeedback, setRefineFeedback] = useState('')
  const [refining, setRefining] = useState(false)

  // v136: 加载方案预设
  useEffect(() => {
    getSchemePresets().then(p => setPresets(p)).catch(() => {})
  }, [])

  // 生成中的10秒兜底轮询（SSE漏event时仍能推进）：状态离开draft/indexing即视为完成
  useEffect(() => {
    if (!generating || !coursewareId) return
    const t = setInterval(async () => { try { const d = await getCourseware(coursewareId); if (d.status !== 'draft' && d.status !== 'indexing') { setGenerating(false); onCoursewareUpdate(d); setPages(d.pages || []); goToStep(1); setSseMessage('✅ 完成'); sseRef.current?.close() } } catch {} }, 10000)
    return () => clearInterval(t)
  }, [generating, coursewareId])

  // Step 0: 生成方案（按来源类型分流四个端点）
  const handleGenerate = async () => {
    if (!coursewareId) return; setGenerating(true); setSseMessage('正在启动...'); setPages([])
    try {
      if (courseware.source_type === 'topic_direct') {
        await generateCWIndexFromTopic(coursewareId, {
          subject: courseware.subject,
          grade: courseware.grade,
          topic: courseware.title,
          preset: selectedPreset,
        })
      } else if (courseware.source_type === 'ppt_upload') {
        await generateCWIndexFromPPT(coursewareId, selectedPreset)
      } else if (courseware.source_type === 'doc_upload') {
        await generateCWIndexFromDoc(coursewareId, selectedPreset)
      } else {
        await generateCWIndex(coursewareId, selectedPreset)
      }
      sseRef.current?.close()
      sseRef.current = subscribeCWIndexSSE(coursewareId, {
        onConnected: () => setSseMessage('已连接，正在分析教案...'),
        onIndexStart: d => setSseMessage(String((d as Record<string, unknown>).message || '')),
        onIndexProgress: d => setSseMessage(String((d as Record<string, unknown>).message || '')),
        onIndexPage: page => setPages(prev => {
          const next = prev.some(p => p.page_number === page.page_number)
            ? prev.map(p => p.page_number === page.page_number ? page : p)
            : [...prev, page]
          return next.slice().sort((a, b) => a.page_number - b.page_number)
        }),
        onIndexDone: d => { setSseMessage(`✅ ${d.message}`); setGenerating(false); goToStep(1); loadCourseware() },
        onError: d => { setSseMessage(`❌ ${d.message}`); setGenerating(false) },
      })
    } catch { setSseMessage('❌ 启动失败'); setGenerating(false) }
  }

  // Step 1: 确认方案 → 进入选风格
  const handleConfirm = async () => {
    if (!coursewareId || !pages.length) return; setConfirming(true)
    try { await confirmCWIndex(coursewareId); goToStep(2); loadCourseware() } catch { alert('确认失败') } finally { setConfirming(false) }
  }

  // v136: AI修改方案（按老师整体意见重出方案，SSE增量刷新页面列表）
  const handleRefineIndex = async () => {
    if (!coursewareId || !refineFeedback.trim() || refining) return
    setRefining(true); setSseMessage('正在根据意见修改方案...')
    try {
      await refineCWIndex(coursewareId, refineFeedback.trim())
      sseRef.current?.close()
      sseRef.current = subscribeCWIndexSSE(coursewareId, {
        onConnected: () => setSseMessage('已连接，AI正在修改方案...'),
        onIndexStart: d => setSseMessage(String((d as Record<string, unknown>).message || '')),
        onIndexProgress: d => setSseMessage(String((d as Record<string, unknown>).message || '')),
        onIndexPage: page => setPages(prev => {
          const next = prev.some(p => p.page_number === page.page_number)
            ? prev.map(p => p.page_number === page.page_number ? page : p)
            : [...prev, page]
          return next.slice().sort((a, b) => a.page_number - b.page_number)
        }),
        onIndexDone: d => { setSseMessage('\u2705 ' + d.message); setRefining(false); setRefineFeedback(''); loadCourseware() },
        onError: d => { setSseMessage('\u274c ' + d.message); setRefining(false) },
      })
    } catch { setSseMessage('\u274c 启动失败'); setRefining(false) }
  }

  // ==================== JSX（与拆分前 Step0/Step1 逐行一致） ====================
  return <>
    {/* Step 0: AI生成方案 */}
    {activeStep === 0 && <div>
      <h3 style={{ fontSize: 18, fontWeight: 600, color: C.textPrimary, margin: '0 0 8px' }}>🤖 AI生成课件方案</h3>
      <p style={{ fontSize: 14, color: C.textSecondary, margin: '0 0 20px' }}>AI将分析教案内容，自动为每页设计方案。</p>
      {/* v136: 方案结构预设选择 */}
      {presets.length > 0 && !generating && (
        <div style={{ marginBottom: 20 }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: C.textPrimary, marginBottom: 10 }}>选择课件结构预设</div>
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            {presets.map(p => (
              <button key={p.key} onClick={() => setSelectedPreset(p.key)}
                style={{
                  flex: '1 1 200px', maxWidth: 240, padding: '12px 16px', borderRadius: 10, cursor: 'pointer',
                  border: `2px solid ${selectedPreset === p.key ? C.primary : C.border}`,
                  background: selectedPreset === p.key ? C.primaryBg : C.white,
                  textAlign: 'left', transition: 'all 200ms',
                }}>
                <div style={{ fontSize: 20, marginBottom: 4 }}>{p.emoji}</div>
                <div style={{ fontSize: 14, fontWeight: 600, color: selectedPreset === p.key ? C.primary : C.textPrimary }}>{p.name}</div>
                <div style={{ fontSize: 12, color: C.textSecondary, marginTop: 2 }}>{p.description}</div>
                <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>{p.page_range}</div>
              </button>
            ))}
          </div>
        </div>
      )}
      <MsgBar msg={sseMessage} />
      {generating && pages.length > 0 && <div style={{ marginBottom: 16 }}><div style={{ fontSize: 13, color: C.textMuted, marginBottom: 8 }}>已生成 {pages.length} 页方案...</div><IndexEditor coursewareId={coursewareId} pages={pages} onPagesChange={setPages} isAdmin={isAdmin} indexOverview={courseware.index_overview} /></div>}
      <button onClick={handleGenerate} disabled={generating} style={{ padding: '12px 32px', borderRadius: 10, border: 'none', background: generating ? '#E5E7EB' : 'linear-gradient(135deg, #F59E0B, #EF4444)', color: generating ? '#9CA3AF' : '#fff', fontSize: 15, fontWeight: 600, cursor: generating ? 'default' : 'pointer', boxShadow: generating ? 'none' : '0 4px 16px rgba(245,158,11,0.3)' }}>
        {generating ? '⏳ 生成中...' : pages.length > 0 ? '🔄 重新生成' : '🤖 开始AI生成方案'}
      </button>
      {!generating && pages.length > 0 && <button onClick={() => goToStep(1)} style={{ marginLeft: 12, padding: '12px 24px', borderRadius: 10, border: `1px solid ${C.primary}`, background: C.primaryBg, color: C.primary, fontSize: 15, fontWeight: 600, cursor: 'pointer' }}>✏️ 确认方案 →</button>}
    </div>}

    {/* Step 1: 确认方案 */}
    {activeStep === 1 && <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <div><h3 style={{ fontSize: 18, fontWeight: 600, color: C.textPrimary, margin: 0 }}>✏️ 确认方案</h3><p style={{ fontSize: 13, color: C.textSecondary, margin: '4px 0 0' }}>确认每页内容，可调整顺序或修改细节</p></div>
        <div style={{ display: 'flex', gap: 10 }}>
          <button onClick={() => goToStep(0)} style={{ padding: '8px 16px', borderRadius: 8, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 13, cursor: 'pointer' }}>← 重新生成</button>
          <button onClick={handleConfirm} disabled={confirming || !pages.length} style={{ padding: '8px 20px', borderRadius: 8, border: 'none', background: pages.length ? 'linear-gradient(135deg, #F59E0B, #EF4444)' : '#E5E7EB', color: pages.length ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: pages.length && !confirming ? 'pointer' : 'default' }}>{confirming ? '确认中...' : '确认方案，选择风格 →'}</button>
        </div>
      </div>
      {/* 课件↔教案对齐报告卡片（仅教案来源课件后端才有报告；非教案来源组件内部自渲染为 null） */}
      <AlignmentReportCard coursewareId={coursewareId} sourceType={courseware.source_type} />
      <IndexEditor coursewareId={coursewareId} pages={pages} onPagesChange={setPages} isAdmin={isAdmin} indexOverview={courseware.index_overview} />
      {/* v136: AI修改方案输入区 */}
      {pages.length > 0 && !refining && (
        <div style={{ marginTop: 16, padding: '16px', borderRadius: 10, border: '1px solid ' + C.border, background: '#FAFAFA' }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🤖 对整体方案不满意？输入修改意见让AI重新调整</div>
          <div style={{ display: 'flex', gap: 10 }}>
            <input value={refineFeedback} onChange={e => setRefineFeedback(e.target.value)}
              placeholder="例如：小学生不需要学习目标页、增加互动练习、减少纯文字页面..."
              onKeyDown={e => { if (e.key === 'Enter' && refineFeedback.trim()) handleRefineIndex() }}
              style={{ flex: 1, padding: '10px 14px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 14, outline: 'none' }} />
            <button onClick={handleRefineIndex} disabled={!refineFeedback.trim()}
              style={{ padding: '10px 20px', borderRadius: 8, border: 'none', background: refineFeedback.trim() ? '#7C3AED' : '#E5E7EB', color: refineFeedback.trim() ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: refineFeedback.trim() ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
              🤖 AI修改方案
            </button>
          </div>
        </div>
      )}
      {refining && <div style={{ marginTop: 16, textAlign: 'center', padding: 20, color: C.textMuted, fontSize: 14 }}><div style={{ fontSize: 32, marginBottom: 8 }}>🤖</div>AI正在根据您的意见修改方案，请稍候...</div>}
      <MsgBar msg={sseMessage} />
    </div>}
  </>
}

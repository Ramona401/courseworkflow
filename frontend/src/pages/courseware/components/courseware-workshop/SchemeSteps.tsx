/**
 * SchemeSteps.tsx — 课件工坊 Step0(AI生成方案)+Step1(确认方案)（批次5b-2从主页面拆出）
 *
 * 【AI自动修正方案】AlignmentReportCard 经 onAutoFix 回调把对齐报告问题拼成的修正指令
 *   传入本组件，复用 refineCWIndex + SSE 流程执行修正。
 *   修正完成后自动调 recheckAlignment 触发重新校验，形成"诊断→修正→复查"闭环。
 *   autoFixMessage 独立于 sseMessage，不会被 loadCourseware 冲掉，持久显示修正结果。
 */
import { useState, useEffect } from 'react'
import { useAuth } from '@/store/auth'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import type { Dispatch, SetStateAction, MutableRefObject } from 'react'
import {
  getCourseware, generateCWIndex, generateCWIndexFromTopic, generateCWIndexFromPPT,
  generateCWIndexFromDoc, subscribeCWIndexSSE, confirmCWIndex, refineCWIndex, getSchemePresets,
  recheckAlignment,
} from '@/api/coursewares'
import type { SchemePreset, CoursewareDetail, CoursewarePage } from '@/api/coursewares'
import IndexEditor from '../IndexEditor'
import AlignmentReportCard from './AlignmentReportCard'
import { C } from './workshopConstants'
import { MsgBar } from './PagePreviewBlock'
import CustomSchemeBuilder from './CustomSchemeBuilder'

interface Props {
  coursewareId: string
  courseware: CoursewareDetail
  isAdmin: boolean
  activeStep: number
  pages: CoursewarePage[]
  setPages: Dispatch<SetStateAction<CoursewarePage[]>>
  sseRef: MutableRefObject<{ close: () => void } | null>
  goToStep: (n: number) => void
  loadCourseware: () => void
  onCoursewareUpdate: (d: CoursewareDetail) => void
}

export default function SchemeSteps({ coursewareId, courseware, isAdmin, activeStep, pages, setPages, sseRef, goToStep, loadCourseware, onCoursewareUpdate }: Props) {
  const { user } = useAuth()

  const [generating, setGenerating] = useState(false)
  const [sseMessage, setSseMessage] = useState('')
  const [confirming, setConfirming] = useState(false)
  const [presets, setPresets] = useState<SchemePreset[]>([])

  /**
   * 课件方案入口草稿按当前用户和课件ID隔离。
   *
   * 结构预设和自定义结构说明会保留，便于返回本步骤后继续生成；
   * 整体方案修改意见仅在AI成功完成修改后提交清空。
   */
  const presetDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'courseware-scheme',
    resourceId: coursewareId,
    field: 'preset',
    initialValue: 'auto',
    maxHistory: 12,
  })
  const selectedPreset =
    presetDraft.value || 'auto'
  const setSelectedPreset =
    presetDraft.setValue

  const customPromptDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'courseware-scheme',
    resourceId: coursewareId,
    field: 'custom-prompt',
    initialValue: '',
    maxHistory: 30,
  })
  const customPromptHint =
    customPromptDraft.value
  const setCustomPromptHint =
    customPromptDraft.setValue

  const refineFeedbackDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'courseware-scheme',
    resourceId: coursewareId,
    field: 'refine-feedback',
    initialValue: '',
    maxHistory: 40,
  })
  const refineFeedback =
    refineFeedbackDraft.value
  const setRefineFeedback =
    refineFeedbackDraft.setValue
  const [refining, setRefining] = useState(false)
  // AI自动修正后自动重新校验的标记（修正完成递增，AlignmentReportCard的key变化触发重挂载重拉）
  const [alignmentRecheckKey, setAlignmentRecheckKey] = useState(0)
  // AI自动修正的持久成功提示（独立于sseMessage，不会被loadCourseware冲掉）
  // 格式：{ text: 显示文案, type: 'success'|'info' }
  const [autoFixMessage, setAutoFixMessage] = useState<{ text: string; type: 'success' | 'info' } | null>(null)

  useEffect(() => {
    getSchemePresets().then(p => setPresets(p)).catch(() => {})
  }, [])

  // 已缓存的预设已下线时，安全回退到自动方案。
  useEffect(() => {
    if (presets.length === 0) return
    if (presets.some(preset => preset.key === selectedPreset)) return
    setSelectedPreset('auto')
  }, [presets, selectedPreset, setSelectedPreset])

  // 生成中10秒兜底轮询
  useEffect(() => {
    if (!generating || !coursewareId) return
    const t = setInterval(async () => { try { const d = await getCourseware(coursewareId); if (d.status !== 'draft' && d.status !== 'indexing') { setGenerating(false); onCoursewareUpdate(d); setPages(d.pages || []); goToStep(1); setSseMessage('✅ 完成'); sseRef.current?.close() } } catch {} }, 10000)
    return () => clearInterval(t)
  }, [generating, coursewareId])

  // Step 0: 生成方案（按来源类型分流四个端点）
  const handleGenerate = async () => {
    if (!coursewareId) return; setGenerating(true); setSseMessage('正在启动...'); setPages([])
    setAutoFixMessage(null) // 清除旧的修正提示
    try {
      if (courseware.source_type === 'topic_direct') {
        await generateCWIndexFromTopic(coursewareId, { subject: courseware.subject, grade: courseware.grade, topic: courseware.title, preset: selectedPreset, custom_prompt_hint: selectedPreset === 'custom' ? customPromptHint : undefined })
      } else if (courseware.source_type === 'ppt_upload') {
        await generateCWIndexFromPPT(coursewareId, selectedPreset, selectedPreset === 'custom' ? customPromptHint : undefined)
      } else if (courseware.source_type === 'doc_upload') {
        await generateCWIndexFromDoc(coursewareId, selectedPreset, selectedPreset === 'custom' ? customPromptHint : undefined)
      } else {
        await generateCWIndex(coursewareId, selectedPreset, selectedPreset === 'custom' ? customPromptHint : undefined)
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

  // Step 1: 确认方案
  const handleConfirm = async () => {
    if (!coursewareId || !pages.length) return; setConfirming(true)
    try { await confirmCWIndex(coursewareId); goToStep(2); loadCourseware() } catch { alert('确认失败') } finally { setConfirming(false) }
  }

  /**
   * AI修改方案（通用入口）——按指令重出方案，SSE增量刷新页面列表。
   * @param instruction 修改指令文本
   * @param autoRecheck 修正完成后是否自动触发对齐重新校验
   */
  const doRefineIndex = async (instruction: string, autoRecheck = false) => {
    if (!coursewareId || !instruction.trim() || refining) return
    setRefining(true)
    setSseMessage('🔧 正在根据意见修改方案...')
    setAutoFixMessage(autoRecheck ? { text: '🔧 AI 正在修正方案，请稍候…', type: 'info' } : null)
    try {
      await refineCWIndex(coursewareId, instruction.trim())
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
        onIndexDone: d => {
          setSseMessage('✅ ' + d.message)
          setRefining(false)
          // 自动对齐修正不消费老师手工输入框；手工修改成功后才提交草稿。
          if (!autoRecheck) {
            refineFeedbackDraft.commit()
          }
          loadCourseware()
          // AI自动修正后：显示成功提示并触发重新校验
          if (autoRecheck) {
            setAutoFixMessage({ text: '✅ 方案已修正完成！正在重新校验对齐度…', type: 'success' })
            recheckAlignment(coursewareId)
              .then(() => {
                setAlignmentRecheckKey(prev => prev + 1)
                // 延迟更新提示，给校验轮询留时间（校验约10-15秒，卡片自己轮询）
                setTimeout(() => {
                  setAutoFixMessage({ text: '✅ 方案修正完成，对齐报告已更新。请查看上方报告了解改善情况。', type: 'success' })
                }, 2000)
              })
              .catch(() => {
                setAutoFixMessage({ text: '✅ 方案已修正完成，但重新校验未能启动。可点击报告卡片的"🔄 重新校验"手动触发。', type: 'success' })
              })
          }
        },
        onError: d => {
          setSseMessage('❌ ' + d.message)
          setRefining(false)
          if (autoRecheck) {
            setAutoFixMessage({ text: '❌ 方案修正失败：' + d.message, type: 'info' })
          }
        },
      })
    } catch {
      setSseMessage('❌ 启动失败')
      setRefining(false)
      if (autoRecheck) {
        setAutoFixMessage({ text: '❌ 方案修正启动失败，请稍后重试', type: 'info' })
      }
    }
  }

  // 手动输入修改意见（不自动重新校验）
  const handleRefineIndex = () => doRefineIndex(refineFeedback)

  // AI自动修正（由AlignmentReportCard的onAutoFix回调触发，修正后自动重新校验）
  const handleAutoFix = (instruction: string) => doRefineIndex(instruction, true)

  return <>
    {/* Step 0: AI生成方案 */}
    {activeStep === 0 && <div>
      <h3 style={{ fontSize: 18, fontWeight: 600, color: C.textPrimary, margin: '0 0 8px' }}>🤖 AI生成课件方案</h3>
      <p style={{ fontSize: 14, color: C.textSecondary, margin: '0 0 20px' }}>AI将分析教案内容，自动为每页设计方案。</p>
      {presets.length > 0 && !generating && (
        <div style={{ marginBottom: 20 }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: C.textPrimary, marginBottom: 10 }}>选择课件结构预设</div>
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            {presets.map(p => (
              <button key={p.key} onClick={() => setSelectedPreset(p.key)}
                style={{ flex: '1 1 200px', maxWidth: 240, padding: '12px 16px', borderRadius: 10, cursor: 'pointer',
                  border: `2px solid ${selectedPreset === p.key ? C.primary : C.border}`,
                  background: selectedPreset === p.key ? C.primaryBg : C.white, textAlign: 'left', transition: 'all 200ms' }}>
                <div style={{ fontSize: 20, marginBottom: 4 }}>{p.emoji}</div>
                <div style={{ fontSize: 14, fontWeight: 600, color: selectedPreset === p.key ? C.primary : C.textPrimary }}>{p.name}</div>
                <div style={{ fontSize: 12, color: C.textSecondary, marginTop: 2 }}>{p.description}</div>
                <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>{p.page_range}</div>
              </button>
            ))}
          </div>
        </div>
      )}
      {/* 自定义预设：选中custom时显示环节搭建器 */}
      {selectedPreset === 'custom' && !generating && (
        <div
          onKeyDown={event => {
            customPromptDraft.handleKeyDown(event)
          }}
        >
          <CustomSchemeBuilder
            value={customPromptHint}
            onChange={setCustomPromptHint}
          />
          <div style={{ marginTop: 6, fontSize: 11, color: C.textMuted }}>
            已自动保存自定义结构 · Ctrl/Command+Z恢复误删
          </div>
        </div>
      )}
      <MsgBar msg={sseMessage} />
      {generating && pages.length > 0 && <div style={{ marginBottom: 16 }}><div style={{ fontSize: 13, color: C.textMuted, marginBottom: 8 }}>已生成 {pages.length} 页方案...</div><IndexEditor coursewareId={coursewareId} pages={pages} onPagesChange={setPages} isAdmin={isAdmin} indexOverview={courseware.index_overview} /></div>}
      <button onClick={handleGenerate} disabled={generating || (selectedPreset === 'custom' && !customPromptHint.trim())} style={{ padding: '12px 32px', borderRadius: 10, border: 'none', background: generating ? '#E5E7EB' : 'linear-gradient(135deg, #F59E0B, #EF4444)', color: generating ? '#9CA3AF' : '#fff', fontSize: 15, fontWeight: 600, cursor: generating ? 'default' : 'pointer', boxShadow: generating ? 'none' : '0 4px 16px rgba(245,158,11,0.3)' }}>
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

      {/* AI修正持久成功提示条——独立于sseMessage，不会被loadCourseware冲掉 */}
      {autoFixMessage && (
        <div style={{
          marginBottom: 12, padding: '10px 16px', borderRadius: 10, fontSize: 13, lineHeight: 1.6,
          display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8,
          background: autoFixMessage.type === 'success' ? '#F0FDF4' : '#EFF6FF',
          border: `1.5px solid ${autoFixMessage.type === 'success' ? '#86EFAC' : '#93C5FD'}`,
          color: autoFixMessage.type === 'success' ? '#166534' : '#1E40AF',
        }}>
          <span>{autoFixMessage.text}</span>
          <button onClick={() => setAutoFixMessage(null)}
            style={{ background: 'none', border: 'none', fontSize: 16, cursor: 'pointer', color: 'inherit', padding: '0 4px', opacity: 0.6 }}>✕</button>
        </div>
      )}

      {/* 课件↔教案对齐报告卡片 */}
      <AlignmentReportCard
        key={alignmentRecheckKey}
        coursewareId={coursewareId}
        sourceType={courseware.source_type}
        onAutoFix={handleAutoFix}
      />
      <IndexEditor coursewareId={coursewareId} pages={pages} onPagesChange={setPages} isAdmin={isAdmin} indexOverview={courseware.index_overview} />

      {/* AI修改方案输入区 */}
      {pages.length > 0 && !refining && (
        <div style={{ marginTop: 16, padding: '16px', borderRadius: 10, border: '1px solid ' + C.border, background: '#FAFAFA' }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🤖 对整体方案不满意？输入修改意见让AI重新调整</div>
          <div style={{ display: 'flex', gap: 10 }}>
            <input value={refineFeedback} onChange={e => setRefineFeedback(e.target.value)}
              placeholder="例如：小学生不需要学习目标页、增加互动练习、减少纯文字页面..."
              onKeyDown={e => {
                if (refineFeedbackDraft.handleKeyDown(e)) return
                if (e.key === 'Enter' && refineFeedback.trim()) handleRefineIndex()
              }}
              style={{ flex: 1, padding: '10px 14px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 14, outline: 'none' }} />
            <button onClick={handleRefineIndex} disabled={!refineFeedback.trim()}
              style={{ padding: '10px 20px', borderRadius: 8, border: 'none', background: refineFeedback.trim() ? '#7C3AED' : '#E5E7EB', color: refineFeedback.trim() ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: refineFeedback.trim() ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
              🤖 AI修改方案
            </button>
          </div>
          <div style={{ marginTop: 6, fontSize: 11, color: C.textMuted }}>
            修改意见已自动保存 · AI修改成功后清除 · Ctrl/Command+Z恢复误删
          </div>
        </div>
      )}
      {refining && <div style={{ marginTop: 16, textAlign: 'center', padding: 20, color: C.textMuted, fontSize: 14 }}><div style={{ fontSize: 32, marginBottom: 8 }}>🤖</div>AI正在根据您的意见修改方案，请稍候...</div>}
      <MsgBar msg={sseMessage} />
    </div>}
  </>
}

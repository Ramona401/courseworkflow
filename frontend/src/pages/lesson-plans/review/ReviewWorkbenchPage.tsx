/**
 * ReviewWorkbenchPage — 独立全屏评审工作台
 *
 * v106新增：从列表页跳转到此页面，URL为 /lesson-plans/review/:id
 * 三列布局：教案预览（左44%）+ 评审面板（中26%）+ AI辅助侧边栏（右300px）
 * 布局脱离 LPLayout，使用全屏固定布局，最大化操作空间
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getLessonPlan,
  reviewLessonPlan,
  type LessonPlan,
} from '@/api/lesson-plans'
import { getAnnotations, createAnnotation, deleteAnnotation, type Annotation } from '@/api/annotations'

/* ==================== 样式常量 ==================== */
const C = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent:       '#F59E0B',
  success:      '#10B981',
  warning:      '#F97316',
  danger:       '#EF4444',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
}

const DECISION_CONFIG = {
  approved: { label: '评审通过', color: C.success, bg: 'rgba(16,185,129,0.08)', icon: '✅' },
  revision: { label: '退回修改', color: C.warning, bg: 'rgba(249,115,22,0.08)', icon: '↩️' },
}

const REVIEW_DIMENSIONS = [
  { code: 'T1', name: '目标清晰度', hint: '三维目标是否具体、可观察、可评估' },
  { code: 'T2', name: '结构完整性', hint: '环节是否齐全、时间分配是否合理' },
  { code: 'T3', name: '学生参与度', hint: '学生主动参与vs被动接收，讲授占比' },
  { code: 'T4', name: '评估对齐度', hint: '评估方式能否检验目标达成' },
  { code: 'T5', name: '可操作性',   hint: '活动步骤清晰、材料可获得' },
]

function splitParagraphs(md: string): string[] {
  if (!md) return []
  return md.split(/\n\s*\n/).map(p => p.trim()).filter(p => p.length > 0)
}

function renderPara(text: string): string {
  let result = text
    .replace(/^### (.+)$/gm, '<h3 style="font-size:15px;font-weight:700;color:#1F2937;margin:12px 0 6px">$1</h3>')
    .replace(/^## (.+)$/gm, '<h2 style="font-size:17px;font-weight:700;color:#1F2937;margin:16px 0 8px">$1</h2>')
    .replace(/^# (.+)$/gm, '<h1 style="font-size:19px;font-weight:700;color:#1F2937;margin:0 0 12px">$1</h1>')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/^[*-] (.+)$/gm, '<li style="margin:3px 0;color:#374151">$1</li>')
  result = result.replace(/(<li[^>]*>.*?<\/li>\n?)+/gs, (m) => `<ul style="margin:6px 0 6px 18px">${m}</ul>`)
  if (!result.includes('<')) {
    result = `<p style="margin:6px 0;line-height:1.8;color:#374151;font-size:14px">${text}</p>`
  }
  return result
}

/* ==================== AI辅助侧边栏 ==================== */

interface ChatMsg { role: 'user' | 'assistant'; content: string }

function ReviewAISidebar({ plan }: { plan: LessonPlan }) {
  const [overview, setOverview]         = useState<string>('')
  const [overviewLoading, setOvLoading] = useState(false)
  const [chatMsgs, setChatMsgs]         = useState<ChatMsg[]>([])
  const [chatInput, setChatInput]       = useState('')
  const [chatLoading, setChatLoading]   = useState(false)
  const [activePanel, setActivePanel]   = useState<'overview' | 'chat'>('overview')
  const chatEndRef = useRef<HTMLDivElement>(null)

  const planContent = plan.content_markdown || ''
  const planMeta = `学科：${plan.subject}  年级：${plan.grade}  课题：${plan.topic}  课时：${plan.duration_minutes}分钟`

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [chatMsgs, chatLoading])

  const loadOverview = useCallback(async () => {
    if (overviewLoading || overview) return
    if (!planContent) { setOverview('教案正文内容为空，无法生成概览。'); return }
    setOvLoading(true)
    try {
      const token = localStorage.getItem('token') || ''
      const resp = await fetch('/api/v1/lesson-plans/review-ai/overview', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({ plan_meta: planMeta, plan_content: planContent.slice(0, 3000) }),
      })
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
      const data = await resp.json()
      setOverview(data.data?.overview || '概览生成失败，请重试。')
    } catch {
      setOverview('概览生成失败，请重试。')
    } finally {
      setOvLoading(false)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [overviewLoading, overview, planContent, planMeta])

  useEffect(() => {
    if (planContent) loadOverview()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [planContent])

  const sendChat = async () => {
    if (!chatInput.trim() || chatLoading) return
    const userMsg = chatInput.trim()
    setChatInput('')
    setChatMsgs(prev => [...prev, { role: 'user', content: userMsg }])
    setChatLoading(true)
    try {
      const token = localStorage.getItem('token') || ''
      const history = chatMsgs.map(m => ({ role: m.role, content: m.content }))
      const resp = await fetch('/api/v1/lesson-plans/review-ai/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({
          plan_meta: planMeta,
          plan_content: planContent.slice(0, 4000),
          history,
          message: userMsg,
        }),
      })
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
      const data = await resp.json()
      setChatMsgs(prev => [...prev, { role: 'assistant', content: data.data?.reply || '抱歉，AI暂时无法回答。' }])
    } catch {
      setChatMsgs(prev => [...prev, { role: 'assistant', content: '⚠️ 请求失败，请检查网络后重试。' }])
    } finally {
      setChatLoading(false)
    }
  }

  return (
    <div style={{ width: '400px', flexShrink: 0, display: 'flex', flexDirection: 'column', background: '#F8FAFF', borderLeft: `1px solid ${C.border}`, overflow: 'hidden' }}>
      {/* 头部Tab切换 */}
      <div style={{ padding: '12px 14px', borderBottom: `1px solid ${C.border}`, background: 'linear-gradient(135deg,rgba(79,123,232,0.08),rgba(129,140,248,0.06))', flexShrink: 0 }}>
        <div style={{ fontSize: '13px', fontWeight: 700, color: C.primary, marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '6px' }}>
          🤖 AI 评审助手
        </div>
        <div style={{ display: 'flex', gap: '4px' }}>
          {(['overview', 'chat'] as const).map(p => (
            <button key={p} onClick={() => setActivePanel(p)}
              style={{ flex: 1, padding: '5px 0', borderRadius: '6px', border: `1px solid ${activePanel === p ? C.primary : C.border}`, background: activePanel === p ? C.primaryLight : 'transparent', fontSize: '12px', color: activePanel === p ? C.primary : C.textSec, cursor: 'pointer', fontWeight: activePanel === p ? 600 : 400 }}>
              {p === 'overview' ? '📋 整体概览' : '💬 对话评审'}
            </button>
          ))}
        </div>
      </div>

      {/* 概览面板 */}
      {activePanel === 'overview' && (
        <div style={{ flex: 1, overflow: 'auto', padding: '14px' }}>
          {/* 教案基础信息 */}
          <div style={{ padding: '10px 12px', background: C.card, borderRadius: '8px', border: `1px solid ${C.border}`, marginBottom: '12px', fontSize: '12px', color: C.textSec, lineHeight: 1.7 }}>
            <div style={{ fontWeight: 600, color: C.text, marginBottom: '4px' }}>📌 教案基本信息</div>
            <div>{plan.subject} · {plan.grade} · {plan.topic}</div>
            <div>课时 {plan.duration_minutes}分钟 · 作者 {plan.author_name || '教师'}</div>
            {plan.ai_review_score != null && (
              <div style={{ marginTop: '4px', color: plan.ai_review_score >= 8.5 ? C.success : C.accent, fontWeight: 600 }}>
                🤖 AI自评分 {plan.ai_review_score.toFixed(1)}
              </div>
            )}
          </div>

          {/* AI概览 */}
          <div style={{ marginBottom: '12px' }}>
            <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '8px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span>🔍 AI整体概览</span>
              {overview && !overviewLoading && (
                <button onClick={() => { setOverview(''); setTimeout(loadOverview, 100) }}
                  style={{ background: 'none', border: 'none', fontSize: '11px', color: C.primary, cursor: 'pointer', padding: 0 }}>
                  🔄 重新生成
                </button>
              )}
            </div>
            {overviewLoading ? (
              <div style={{ padding: '16px', background: C.card, borderRadius: '8px', border: `1px solid ${C.border}`, textAlign: 'center' }}>
                <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '8px' }}>AI正在解读教案...</div>
                <div style={{ display: 'flex', gap: '4px', justifyContent: 'center' }}>
                  {[0,1,2].map(i => (
                    <div key={i} style={{ width: '5px', height: '5px', borderRadius: '50%', background: C.primary, animation: `pulse 1.2s ease-in-out ${i*0.2}s infinite` }} />
                  ))}
                </div>
                <style>{`@keyframes pulse{0%,80%,100%{opacity:0.3;transform:scale(0.8)}40%{opacity:1;transform:scale(1.2)}}`}</style>
              </div>
            ) : overview ? (
              <div style={{ padding: '12px', background: C.card, borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', color: C.text, lineHeight: 1.8, whiteSpace: 'pre-wrap' }}>
                {overview}
              </div>
            ) : (
              <button onClick={loadOverview} style={{ width: '100%', padding: '10px', borderRadius: '8px', border: `1px dashed ${C.primary}`, background: 'transparent', fontSize: '13px', color: C.primary, cursor: 'pointer' }}>
                🔍 点击生成AI概览
              </button>
            )}
          </div>

          {/* 评审维度参考 */}
          <div>
            <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '8px' }}>📊 评审维度参考</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
              {REVIEW_DIMENSIONS.map(dim => (
                <div key={dim.code} style={{ padding: '6px 10px', background: C.card, borderRadius: '6px', border: `1px solid ${C.border}`, fontSize: '12px' }}>
                  <span style={{ fontWeight: 700, color: C.primary, marginRight: '6px' }}>{dim.code}</span>
                  <span style={{ fontWeight: 600, color: C.text }}>{dim.name}</span>
                  <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '2px', lineHeight: 1.5 }}>{dim.hint}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* 对话面板 */}
      {activePanel === 'chat' && (
        <>
          <div style={{ flex: 1, overflow: 'auto', padding: '12px 14px', display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {chatMsgs.length === 0 && (
              <div style={{ padding: '12px', background: C.primaryLight, borderRadius: '8px', border: '1px solid rgba(79,123,232,0.2)', fontSize: '12px', color: C.textSec, lineHeight: 1.7 }}>
                <div style={{ fontWeight: 600, color: C.primary, marginBottom: '6px' }}>💡 评审对话提示</div>
                AI已阅读完整教案，你可以问：
                <div style={{ marginTop: '6px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  {['这个导入设计有什么问题？','教学目标写法是否规范？','活动时间分配合理吗？','与优秀教案相比如何？'].map(q => (
                    <button key={q} onClick={() => setChatInput(q)}
                      style={{ textAlign: 'left', padding: '4px 8px', borderRadius: '6px', border: '1px solid rgba(79,123,232,0.2)', background: 'rgba(79,123,232,0.04)', fontSize: '11px', color: C.primary, cursor: 'pointer' }}>
                      {q}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {chatMsgs.map((msg, i) => (
              <div key={i} style={{ display: 'flex', justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start' }}>
                {msg.role === 'assistant' && (
                  <div style={{ width: '22px', height: '22px', borderRadius: '50%', background: 'linear-gradient(135deg,#4F7BE8,#818CF8)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '11px', flexShrink: 0, marginRight: '6px', marginTop: '2px' }}>✨</div>
                )}
                <div style={{ maxWidth: '85%', padding: '8px 11px', borderRadius: msg.role === 'user' ? '12px 2px 12px 12px' : '2px 12px 12px 12px', background: msg.role === 'user' ? C.primary : C.card, color: msg.role === 'user' ? '#fff' : C.text, fontSize: '12px', lineHeight: 1.7, border: msg.role === 'assistant' ? `1px solid ${C.border}` : 'none', whiteSpace: 'pre-wrap' }}>
                  {msg.content}
                </div>
              </div>
            ))}

            {chatLoading && (
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: '6px' }}>
                <div style={{ width: '22px', height: '22px', borderRadius: '50%', background: 'linear-gradient(135deg,#4F7BE8,#818CF8)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '11px', flexShrink: 0 }}>✨</div>
                <div style={{ padding: '8px 12px', background: C.card, borderRadius: '2px 12px 12px 12px', border: `1px solid ${C.border}`, display: 'flex', gap: '4px', alignItems: 'center' }}>
                  {[0,1,2].map(i => (
                    <div key={i} style={{ width: '5px', height: '5px', borderRadius: '50%', background: C.primary, animation: `pulse 1.2s ease-in-out ${i*0.2}s infinite` }} />
                  ))}
                </div>
              </div>
            )}
            <div ref={chatEndRef} />
          </div>

          <div style={{ padding: '10px 12px', borderTop: `1px solid ${C.border}`, background: C.card, flexShrink: 0 }}>
            <div style={{ display: 'flex', gap: '6px', alignItems: 'flex-end', background: C.bg, borderRadius: '8px', border: `1px solid ${C.border}`, padding: '6px 8px' }}>
              <textarea
                value={chatInput}
                onChange={e => setChatInput(e.target.value)}
                onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendChat() } }}
                placeholder="问AI关于这份教案的问题..."
                rows={2}
                disabled={chatLoading}
                style={{ flex: 1, background: 'transparent', border: 'none', outline: 'none', fontSize: '12px', color: C.text, resize: 'none', fontFamily: 'inherit', lineHeight: 1.5, opacity: chatLoading ? 0.5 : 1 }}
              />
              <button
                onClick={sendChat}
                disabled={chatLoading || !chatInput.trim()}
                style={{ width: '26px', height: '26px', borderRadius: '50%', border: 'none', background: chatLoading || !chatInput.trim() ? '#E5E7EB' : C.primary, color: '#fff', cursor: 'pointer', fontSize: '12px', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                →
              </button>
            </div>
            <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px', textAlign: 'center' }}>
              Enter发送 · Shift+Enter换行
            </div>
          </div>
        </>
      )}
    </div>
  )
}

/* ==================== 教案预览面板（左）==================== */

function PlanPreviewPanel({ plan, annotations, onAnnotationCreated, onAnnotationDeleted }: {
  plan: LessonPlan
  annotations: Annotation[]
  onAnnotationCreated: (a: Annotation) => void
  onAnnotationDeleted: (id: string) => void
}) {
  const [activeAnnotIdx, setActiveAnnotIdx] = useState<number | null>(null)
  const [annotInput, setAnnotInput]         = useState('')
  const [savingAnnot, setSavingAnnot]       = useState(false)
  const { user } = useAuth()
  const content = plan.content_markdown || ''

  return (
    <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', background: C.card, borderRight: `1px solid ${C.border}`, overflow: 'hidden' }}>
      {/* 教案头部信息 */}
      <div style={{ padding: '14px 18px', borderBottom: `1px solid ${C.border}`, background: C.bg, flexShrink: 0 }}>
        <div style={{ fontSize: '16px', fontWeight: 700, color: C.text, marginBottom: '6px' }}>{plan.title}</div>
        <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', fontSize: '12px', color: C.textSec }}>
          <span>📚 {plan.subject}</span>
          <span>🎓 {plan.grade}</span>
          <span>⏱ {plan.duration_minutes}分钟</span>
          <span>📌 {plan.topic}</span>
          <span>✍️ {plan.author_name || '教师'}</span>
          {plan.ai_review_score != null && (
            <span style={{ padding: '1px 8px', borderRadius: '20px', background: plan.ai_review_score >= 8.5 ? 'rgba(16,185,129,0.1)' : 'rgba(245,158,11,0.1)', color: plan.ai_review_score >= 8.5 ? C.success : C.accent, fontWeight: 600 }}>
              🤖 {plan.ai_review_score.toFixed(1)}
            </span>
          )}
        </div>
      </div>

      {/* 正文内容（可滚动）*/}
      <div style={{ flex: 1, overflow: 'auto', padding: '16px 20px' }}>
        {content ? (
          <div>
            <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '10px', padding: '5px 8px', background: '#FFF7ED', borderRadius: '5px', border: '1px solid #FED7AA' }}>
              💡 点击段落旁 <strong>💬</strong> 可添加评审批注
            </div>
            {splitParagraphs(content).map((para, idx) => {
              const paraAnnotations = annotations.filter(a => a.paragraph_index === idx)
              const isActive = activeAnnotIdx === idx
              return (
                <div key={idx} style={{ marginBottom: '6px', position: 'relative' }}>
                  <div style={{ display: 'flex', gap: '6px', alignItems: 'flex-start', borderLeft: paraAnnotations.length > 0 ? '3px solid #F97316' : '3px solid transparent', paddingLeft: paraAnnotations.length > 0 ? '8px' : '0' }}>
                    <div
                      style={{ flex: 1, fontSize: '13px', lineHeight: 1.8, color: C.text }}
                      dangerouslySetInnerHTML={{ __html: renderPara(para) }}
                    />
                    <button
                      onClick={() => { setActiveAnnotIdx(isActive ? null : idx); setAnnotInput('') }}
                      title="添加批注"
                      style={{ flexShrink: 0, marginTop: '4px', padding: '2px 7px', borderRadius: '5px', border: 'none', background: isActive ? '#FEF3C7' : paraAnnotations.length > 0 ? '#FFF7ED' : '#F3F4F6', color: paraAnnotations.length > 0 ? '#92400E' : '#6B7280', fontSize: '11px', fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap' }}>
                      💬 {paraAnnotations.length > 0 ? paraAnnotations.length : '+'}
                    </button>
                  </div>

                  {paraAnnotations.length > 0 && (
                    <div style={{ margin: '4px 0 6px 11px' }}>
                      {paraAnnotations.map(a => (
                        <div key={a.id} style={{ padding: '6px 10px', borderRadius: '5px', marginBottom: '3px', background: '#FFFBEB', border: '1px solid #FED7AA', display: 'flex', gap: '6px', alignItems: 'flex-start' }}>
                          <div style={{ flex: 1 }}>
                            <div style={{ fontSize: '10px', fontWeight: 600, color: '#92400E', marginBottom: '2px' }}>
                              {a.reviewer_name} · {new Date(a.created_at).toLocaleDateString('zh-CN')}
                            </div>
                            <div style={{ fontSize: '12px', color: '#374151', lineHeight: 1.6 }}>{a.content}</div>
                          </div>
                          {(a.reviewer_id === user?.id || user?.role === 'admin') && (
                            <button
                              onClick={async () => {
                                if (!confirm('确认删除？')) return
                                try { await deleteAnnotation(plan.id, a.id); onAnnotationDeleted(a.id) } catch { alert('删除失败') }
                              }}
                              style={{ background: 'none', border: 'none', color: '#9CA3AF', cursor: 'pointer', fontSize: '13px', flexShrink: 0 }}>
                              ×
                            </button>
                          )}
                        </div>
                      ))}
                    </div>
                  )}

                  {isActive && (
                    <div style={{ margin: '4px 0 8px 11px', padding: '8px 10px', background: '#FFFBEB', borderRadius: '7px', border: '1px solid #FED7AA' }}>
                      <div style={{ fontSize: '11px', fontWeight: 600, color: '#92400E', marginBottom: '5px' }}>✍️ 添加批注</div>
                      <textarea
                        value={annotInput}
                        onChange={e => setAnnotInput(e.target.value)}
                        placeholder="写下评审意见..."
                        rows={3}
                        style={{ width: '100%', padding: '7px 9px', borderRadius: '5px', border: '1px solid #FED7AA', fontSize: '12px', lineHeight: 1.6, outline: 'none', resize: 'vertical', boxSizing: 'border-box', fontFamily: 'inherit', background: '#fff' }}
                      />
                      <div style={{ display: 'flex', gap: '5px', marginTop: '6px', justifyContent: 'flex-end' }}>
                        <button
                          onClick={() => { setActiveAnnotIdx(null); setAnnotInput('') }}
                          style={{ padding: '4px 10px', borderRadius: '5px', border: '1px solid #E5E7EB', background: '#fff', fontSize: '11px', color: '#6B7280', cursor: 'pointer' }}>
                          取消
                        </button>
                        <button
                          onClick={async () => {
                            if (!annotInput.trim()) return
                            setSavingAnnot(true)
                            try {
                              const newA = await createAnnotation(plan.id, {
                                paragraph_index: idx,
                                paragraph_preview: para.slice(0, 50),
                                content: annotInput.trim(),
                              })
                              onAnnotationCreated(newA)
                              setAnnotInput('')
                              setActiveAnnotIdx(null)
                            } catch { alert('保存失败') }
                            setSavingAnnot(false)
                          }}
                          disabled={savingAnnot || !annotInput.trim()}
                          style={{ padding: '4px 12px', borderRadius: '5px', border: 'none', background: annotInput.trim() ? '#F97316' : '#E5E7EB', color: annotInput.trim() ? '#fff' : '#9CA3AF', fontSize: '11px', fontWeight: 600, cursor: annotInput.trim() ? 'pointer' : 'not-allowed' }}>
                          {savingAnnot ? '保存中...' : '保存批注'}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        ) : (
          <div style={{ textAlign: 'center', padding: '60px 0', color: C.textMuted }}>
            <div style={{ fontSize: '32px', marginBottom: '12px' }}>📄</div>
            <div style={{ fontSize: '14px' }}>该教案暂无可预览的文本内容</div>
          </div>
        )}
      </div>
    </div>
  )
}

/* ==================== 评审操作面板（中）==================== */

function ReviewActionPanel({ plan, onSubmit, onCancel, submitting }: {
  plan: LessonPlan
  onSubmit: (decision: string, score: number, comments: string, suggestions: string[]) => Promise<void>
  onCancel: () => void
  submitting: boolean
}) {
  const [decision, setDecision]       = useState<'approved' | 'revision'>('approved')
  const [score, setScore]             = useState<number>(8)
  const [comments, setComments]       = useState('')
  const [suggestion, setSuggestion]   = useState('')
  const [suggestions, setSuggestions] = useState<string[]>([])

  const addSuggestion = () => {
    if (!suggestion.trim()) return
    setSuggestions(prev => [...prev, suggestion.trim()])
    setSuggestion('')
  }
  const removeSuggestion = (i: number) => setSuggestions(prev => prev.filter((_, idx) => idx !== i))

  const handleSubmit = () => {
    if (!comments.trim()) { alert('请填写评审意见'); return }
    onSubmit(decision, score, comments, suggestions)
  }

  const decisionCfg = DECISION_CONFIG[decision]

  // 抑制 plan 未使用警告（保留 prop 以便后续扩展）
  void plan

  return (
    <div style={{ width: '320px', flexShrink: 0, display: 'flex', flexDirection: 'column', background: C.bg, borderLeft: `1px solid ${C.border}`, overflow: 'hidden' }}>
      <div style={{ padding: '14px 16px', borderBottom: `1px solid ${C.border}`, background: C.card, flexShrink: 0 }}>
        <span style={{ fontSize: '14px', fontWeight: 700, color: C.text }}>✍️ 评审意见</span>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '14px 16px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
        {/* 综合评分 */}
        <div>
          <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>综合评分</div>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: '5px', marginBottom: '6px' }}>
            <span style={{ fontSize: '28px', fontWeight: 700, lineHeight: 1, color: score >= 8.5 ? C.success : score >= 7 ? C.accent : C.danger }}>{score.toFixed(1)}</span>
            <span style={{ fontSize: '12px', color: C.textMuted }}>/ 10</span>
          </div>
          <input type="range" min={1} max={10} step={0.5} value={score}
            onChange={e => setScore(parseFloat(e.target.value))}
            style={{ width: '100%', accentColor: C.primary, cursor: 'pointer' }} />
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '10px', color: C.textMuted, marginTop: '3px' }}>
            <span>1较差</span><span>5一般</span><span>8良好</span><span>10优秀</span>
          </div>
        </div>

        {/* 评审意见 */}
        <div>
          <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>
            评审意见 <span style={{ color: C.danger }}>*</span>
          </div>
          <textarea
            value={comments}
            onChange={e => setComments(e.target.value)}
            placeholder="整体教学设计评价..."
            rows={6}
            style={{ width: '100%', padding: '9px 11px', borderRadius: '7px', border: `1px solid ${comments ? C.primary : C.border}`, fontSize: '13px', color: C.text, outline: 'none', boxSizing: 'border-box', resize: 'vertical', lineHeight: 1.7, fontFamily: 'inherit', background: C.card }}
            onFocus={e => { e.target.style.borderColor = C.primary }}
            onBlur={e  => { e.target.style.borderColor = comments ? C.primary : C.border }}
          />
        </div>

        {/* 改进建议 */}
        <div>
          <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>
            具体改进建议 <span style={{ fontWeight: 400, color: C.textMuted }}>（选填）</span>
          </div>
          {suggestions.length > 0 && (
            <div style={{ marginBottom: '6px', display: 'flex', flexDirection: 'column', gap: '3px' }}>
              {suggestions.map((s, i) => (
                <div key={i} style={{ display: 'flex', alignItems: 'flex-start', gap: '5px', padding: '5px 8px', background: 'rgba(245,158,11,0.06)', borderRadius: '5px', border: '1px solid rgba(245,158,11,0.2)' }}>
                  <span style={{ fontSize: '11px', color: C.accent, flexShrink: 0 }}>💡</span>
                  <span style={{ fontSize: '11px', color: C.text, flex: 1, lineHeight: 1.5 }}>{s}</span>
                  <button onClick={() => removeSuggestion(i)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.textMuted, fontSize: '13px', lineHeight: 1, flexShrink: 0 }}>×</button>
                </div>
              ))}
            </div>
          )}
          <div style={{ display: 'flex', gap: '5px' }}>
            <input
              type="text"
              value={suggestion}
              onChange={e => setSuggestion(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && addSuggestion()}
              placeholder="输入建议，回车添加..."
              style={{ flex: 1, padding: '6px 9px', borderRadius: '5px', border: `1px solid ${C.border}`, fontSize: '12px', color: C.text, outline: 'none', fontFamily: 'inherit', background: C.card }}
              onFocus={e => { e.target.style.borderColor = C.primary }}
              onBlur={e  => { e.target.style.borderColor = C.border }}
            />
            <button
              onClick={addSuggestion}
              disabled={!suggestion.trim()}
              style={{ padding: '6px 10px', borderRadius: '5px', border: 'none', background: suggestion.trim() ? C.primaryLight : C.border, color: suggestion.trim() ? C.primary : C.textMuted, fontSize: '12px', fontWeight: 600, cursor: suggestion.trim() ? 'pointer' : 'not-allowed' }}>
              +
            </button>
          </div>
        </div>
      </div>

      {/* 底部结论区 */}
      <div style={{ flexShrink: 0, borderTop: `1px solid ${C.border}`, background: C.card, padding: '12px 16px' }}>
        <div style={{ fontSize: '11px', fontWeight: 600, color: C.textSec, marginBottom: '8px' }}>给出结论</div>
        <div style={{ display: 'flex', gap: '5px', marginBottom: '10px' }}>
          {(Object.entries(DECISION_CONFIG) as Array<['approved' | 'revision', typeof DECISION_CONFIG['approved']]>).map(([key, cfg]) => (
            <button key={key} onClick={() => setDecision(key)}
              style={{ flex: 1, padding: '7px 4px', borderRadius: '7px', cursor: 'pointer', border: `2px solid ${decision === key ? cfg.color : C.border}`, background: decision === key ? cfg.bg : 'transparent', fontSize: '11px', fontWeight: decision === key ? 700 : 400, color: decision === key ? cfg.color : C.textSec, transition: 'all 150ms ease', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '3px' }}>
              <span>{cfg.icon}</span><span>{cfg.label}</span>
            </button>
          ))}
        </div>
        <div style={{ padding: '6px 10px', borderRadius: '5px', background: decisionCfg.bg, border: `1px solid ${decisionCfg.color}30`, fontSize: '11px', color: decisionCfg.color, fontWeight: 500, marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '5px' }}>
          <span>{decisionCfg.icon}</span>
          <span>将标记为：<strong>{decisionCfg.label}</strong></span>
          <span style={{ marginLeft: 'auto', fontWeight: 700 }}>{score.toFixed(1)}分</span>
        </div>
        <button
          onClick={handleSubmit}
          disabled={submitting || !comments.trim()}
          style={{ width: '100%', padding: '10px', borderRadius: '8px', border: 'none', background: submitting || !comments.trim() ? C.border : decisionCfg.color, color: submitting || !comments.trim() ? C.textMuted : '#fff', fontSize: '13px', fontWeight: 700, cursor: submitting || !comments.trim() ? 'not-allowed' : 'pointer', transition: 'all 150ms ease' }}>
          {submitting ? '提交中...' : `${decisionCfg.icon} 提交评审`}
        </button>
        {!comments.trim() && (
          <div style={{ fontSize: '10px', color: C.textMuted, textAlign: 'center', marginTop: '4px' }}>请先填写评审意见</div>
        )}
        <button
          onClick={onCancel}
          style={{ width: '100%', marginTop: '6px', padding: '7px', borderRadius: '7px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '12px', color: C.textSec, cursor: 'pointer' }}>
          返回列表
        </button>
      </div>
    </div>
  )
}

/* ==================== 主组件 ==================== */

export default function ReviewWorkbenchPage() {
  const { id }    = useParams<{ id: string }>()
  const navigate  = useNavigate()

  const [plan, setPlan]             = useState<LessonPlan | null>(null)
  const [loading, setLoading]       = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [annotations, setAnnotations] = useState<Annotation[]>([])
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  // 加载教案数据
  useEffect(() => {
    if (!id) return
    setLoading(true)
    getLessonPlan(id)
      .then(data => { setPlan(data); setLoading(false) })
      .catch(() => { setLoading(false) })
  }, [id])

  // 加载批注数据
  useEffect(() => {
    if (!id) return
    getAnnotations(id)
      .then(resp => setAnnotations(resp.annotations || []))
      .catch(() => setAnnotations([]))
  }, [id])

  // 提交评审
  const handleSubmitReview = async (decision: string, score: number, comments: string, suggestions: string[]) => {
    if (!id || submitting) return
    setSubmitting(true)
    try {
      await reviewLessonPlan(id, { decision, score, comments, suggestions: suggestions.join('\n') })
      showToast(`评审完成：${decision === 'approved' ? '✅ 评审通过' : '↩️ 退回修改'} ✓`)
      setTimeout(() => navigate('/lesson-plans/review'), 1500)
    } catch (e) {
      console.error('提交评审失败:', e)
      showToast('提交失败，请稍后重试', 'error')
    } finally {
      setSubmitting(false)
    }
  }

  // 加载中
  if (loading) {
    return (
      <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: C.bg }}>
        <div style={{ textAlign: 'center', color: C.textMuted }}>
          <div style={{ width: '28px', height: '28px', border: `3px solid ${C.border}`, borderTopColor: C.primary, borderRadius: '50%', animation: 'spin 0.8s linear infinite', margin: '0 auto 12px' }} />
          <div>加载教案中...</div>
          <style>{`@keyframes spin{from{transform:rotate(0deg)}to{transform:rotate(360deg)}}`}</style>
        </div>
      </div>
    )
  }

  // 教案不存在
  if (!plan) {
    return (
      <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', flexDirection: 'column', gap: '16px', background: C.bg }}>
        <div style={{ fontSize: '48px' }}>😕</div>
        <div style={{ fontSize: '16px', color: C.text }}>教案不存在或无权限查看</div>
        <button onClick={() => navigate('/lesson-plans/review')}
          style={{ padding: '9px 20px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>
          返回评审列表
        </button>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', background: C.bg }}>
      {/* 顶部导航条 */}
      <div style={{ height: '48px', background: C.card, borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', padding: '0 20px', gap: '12px', flexShrink: 0, boxShadow: '0 1px 4px rgba(0,0,0,0.06)' }}>
        <button
          onClick={() => navigate('/lesson-plans/review')}
          style={{ display: 'flex', alignItems: 'center', gap: '5px', background: 'none', border: 'none', fontSize: '13px', color: C.textSec, cursor: 'pointer', padding: '4px 8px', borderRadius: '6px' }}
          onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = C.bg }}
          onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'none' }}>
          ← 返回列表
        </button>
        <div style={{ width: '1px', height: '16px', background: C.border }} />
        <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          📋 评审：{plan.title}
        </div>
        <div style={{ fontSize: '12px', color: C.textSec, flexShrink: 0 }}>
          {plan.subject} · {plan.grade} · {plan.author_name || '教师'}
        </div>
      </div>

      {/* 主体三列布局（撑满剩余高度）*/}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {/* 左：教案预览（可滚动）*/}
        <PlanPreviewPanel
          plan={plan}
          annotations={annotations}
          onAnnotationCreated={a => setAnnotations(prev => [...prev, a])}
          onAnnotationDeleted={aid => setAnnotations(prev => prev.filter(a => a.id !== aid))}
        />
        {/* 中：AI辅助侧边栏 */}
        <ReviewAISidebar plan={plan} />
        {/* 右：评审操作面板 */}
        <ReviewActionPanel
          plan={plan}
          onSubmit={handleSubmitReview}
          onCancel={() => navigate('/lesson-plans/review')}
          submitting={submitting}
        />
      </div>

      {/* Toast通知 */}
      {toast && (
        <div style={{ position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)', padding: '12px 24px', borderRadius: '10px', background: toast.type === 'error' ? '#FEF2F2' : '#1F2937', color: toast.type === 'error' ? C.danger : '#fff', fontSize: '14px', fontWeight: 500, boxShadow: '0 8px 24px rgba(0,0,0,0.15)', zIndex: 9999, whiteSpace: 'nowrap', border: toast.type === 'error' ? '1px solid #FECACA' : 'none' }}>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}

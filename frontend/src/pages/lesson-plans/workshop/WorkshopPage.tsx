/**
 * 备课工坊首页 — WorkshopPage（v3）
 *
 * v3 新增：恢复模式
 *   - 从 location.state.resumePlanId 接收已有教案ID
 *   - 有resumePlanId时：跳过StartForm，直接进入chatting阶段
 *   - 调用 getConversation(planId) 恢复历史对话记录
 *   - 恢复后与新建流程完全一致（SSE连接/发消息/评审均可用）
 *
 * v2 改动：
 *   1. AI气泡支持 Markdown 渲染
 *   2. 流式输出：chunk事件逐字追加，message_done替换正式消息
 *   3. 修复 handlePublish 静态 import
 *
 * PRD §3.1 完整用户旅程
 * PRD §8.4 AI对话区设计规范
 */
import { useState, useRef, useEffect, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  startConversation,
  sendChatMessage,
  triggerAIReview,
  applyAISuggestions,
  publishLessonPlanPersonal,
  createLessonPlanSSE,
  getLessonPlan,
  getConversation,
  type LessonPlan,
  type ConversationMessage,
  type AIReviewResult,
  type ConvComponent,
} from '@/api/lesson-plans'

/* ==================== 样式常量（PRD §8.2）==================== */
const C = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent:       '#F59E0B',
  success:      '#10B981',
  danger:       '#EF4444',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
  aiBubble:     '#EEF4FF',
  userBubble:   '#FFFFFF',
}

/* ==================== 学科和年级选项 ==================== */
const SUBJECTS = ['AI', '人工智能', '语文', '数学', '英语', '物理', '化学', '生物', '历史', '地理', '政治', '信息技术']
const GRADES   = ['七年级', '八年级', '九年级', '高一', '高二', '高三', '小学低段', '小学中段', '小学高段']

/* ==================== 轻量 Markdown 渲染器 ==================== */
function renderMarkdown(text: string): React.ReactNode {
  if (!text) return null
  const lines = text.split('\n')
  const nodes: React.ReactNode[] = []
  let listItems: React.ReactNode[] = []
  let listType: 'ul' | 'ol' | null = null
  let key = 0

  const parseInline = (line: string): React.ReactNode => {
    const parts = line.split(/(\*\*[^*]+\*\*)/)
    if (parts.length === 1) return line
    return (
      <>
        {parts.map((part, i) => {
          if (part.startsWith('**') && part.endsWith('**')) {
            return <strong key={i} style={{ fontWeight: 700, color: C.text }}>{part.slice(2, -2)}</strong>
          }
          return part
        })}
      </>
    )
  }

  const flushList = () => {
    if (!listItems.length) return
    nodes.push(
      listType === 'ul'
        ? <ul key={key++} style={{ margin: '6px 0 6px 16px', padding: 0, listStyle: 'disc' }}>{listItems}</ul>
        : <ol key={key++} style={{ margin: '6px 0 6px 16px', padding: 0, listStyle: 'decimal' }}>{listItems}</ol>
    )
    listItems = []; listType = null
  }

  for (const line of lines) {
    const t = line.trim()
    if (!t) { flushList(); continue }
    if (/^---+$/.test(t)) {
      flushList()
      nodes.push(<hr key={key++} style={{ border: 'none', borderTop: `1px solid ${C.border}`, margin: '10px 0' }} />)
      continue
    }
    const h3 = t.match(/^###\s+(.+)/); if (h3) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '14px', fontWeight: 700, color: C.text, margin: '10px 0 4px' }}>{parseInline(h3[1])}</div>); continue }
    const h2 = t.match(/^##\s+(.+)/);  if (h2) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '15px', fontWeight: 700, color: C.text, margin: '12px 0 4px' }}>{parseInline(h2[1])}</div>); continue }
    const h1 = t.match(/^#\s+(.+)/);   if (h1) { flushList(); nodes.push(<div key={key++} style={{ fontSize: '16px', fontWeight: 700, color: C.text, margin: '14px 0 6px' }}>{parseInline(h1[1])}</div>); continue }
    const ul = t.match(/^[-*]\s+(.+)/); if (ul) { if (listType !== 'ul') { flushList(); listType = 'ul' }; listItems.push(<li key={key++} style={{ fontSize: '14px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(ul[1])}</li>); continue }
    const ol = t.match(/^\d+\.\s+(.+)/); if (ol) { if (listType !== 'ol') { flushList(); listType = 'ol' }; listItems.push(<li key={key++} style={{ fontSize: '14px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(ol[1])}</li>); continue }
    flushList()
    nodes.push(<div key={key++} style={{ fontSize: '15px', color: C.text, lineHeight: 1.7, marginBottom: '2px' }}>{parseInline(t)}</div>)
  }
  flushList()
  return <>{nodes}</>
}

/* ==================== 子组件：首屏表单 ==================== */
interface StartFormProps {
  onStart: (subject: string, grade: string, topic: string, duration: number) => void
  loading: boolean
}

function StartForm({ onStart, loading }: StartFormProps) {
  const [subject, setSubject]   = useState('AI')
  const [grade, setGrade]       = useState('七年级')
  const [topic, setTopic]       = useState('')
  const [duration, setDuration] = useState(45)
  const navigate = useNavigate()

  const handleSubmit = () => { if (!topic.trim()) return; onStart(subject, grade, topic.trim(), duration) }

  const shortcuts = [
    { icon: '📋', text: '我的教案',   path: '/lesson-plans/my-plans' },
    { icon: '📚', text: '教案库',     path: '/lesson-plans/library' },
    { icon: '📐', text: '提示词模板', path: '/lesson-plans/templates' },
  ]

  return (
    <div style={{ maxWidth: '580px', margin: '0 auto', padding: '48px 0' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '24px' }}>
        <span style={{ fontSize: '28px', lineHeight: 1 }}>✨</span>
        <div>
          <h1 style={{ fontSize: '20px', fontWeight: 700, color: C.text, margin: '0 0 2px' }}>开始今天的备课</h1>
          <p style={{ fontSize: '13px', color: C.textSec, margin: 0 }}>告诉AI你要上什么课，它会全程陪你设计出高质量教案</p>
        </div>
      </div>

      <div style={{ background: C.card, borderRadius: '16px', padding: '32px', boxShadow: '0 4px 24px rgba(0,0,0,0.06)', border: `1px solid ${C.border}` }}>
        {/* 学科 */}
        <div style={{ marginBottom: '20px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>学科</label>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
            {SUBJECTS.map(s => (
              <button key={s} onClick={() => setSubject(s)} style={{ padding: '6px 14px', borderRadius: '20px', border: `1px solid ${subject === s ? C.primary : C.border}`, background: subject === s ? C.primaryLight : 'transparent', color: subject === s ? C.primary : C.textSec, fontSize: '13px', fontWeight: subject === s ? 600 : 400, cursor: 'pointer', transition: 'all 150ms ease' }}>{s}</button>
            ))}
          </div>
        </div>

        {/* 年级 */}
        <div style={{ marginBottom: '20px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>年级</label>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
            {GRADES.map(g => (
              <button key={g} onClick={() => setGrade(g)} style={{ padding: '6px 14px', borderRadius: '20px', border: `1px solid ${grade === g ? C.primary : C.border}`, background: grade === g ? C.primaryLight : 'transparent', color: grade === g ? C.primary : C.textSec, fontSize: '13px', fontWeight: grade === g ? 600 : 400, cursor: 'pointer', transition: 'all 150ms ease' }}>{g}</button>
            ))}
          </div>
        </div>

        {/* 课题 */}
        <div style={{ marginBottom: '20px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>课题 <span style={{ color: C.danger }}>*</span></label>
          <input
            type="text" value={topic} onChange={e => setTopic(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleSubmit()}
            placeholder="例如：认识人工智能、图像识别应用..."
            style={{ width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '15px', color: C.text, outline: 'none', boxSizing: 'border-box', transition: 'border-color 150ms ease' }}
            onFocus={e => { e.target.style.borderColor = C.primary }}
            onBlur={e  => { e.target.style.borderColor = C.border  }}
          />
        </div>

        {/* 课时 */}
        <div style={{ marginBottom: '28px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>课时时长</label>
          <div style={{ display: 'flex', gap: '8px' }}>
            {[40, 45, 50, 60].map(d => (
              <button key={d} onClick={() => setDuration(d)} style={{ padding: '6px 16px', borderRadius: '20px', border: `1px solid ${duration === d ? C.primary : C.border}`, background: duration === d ? C.primaryLight : 'transparent', color: duration === d ? C.primary : C.textSec, fontSize: '13px', fontWeight: duration === d ? 600 : 400, cursor: 'pointer', transition: 'all 150ms ease' }}>{d}分钟</button>
            ))}
          </div>
        </div>

        <button
          onClick={handleSubmit} disabled={!topic.trim() || loading}
          style={{ width: '100%', padding: '14px', borderRadius: '10px', border: 'none', background: !topic.trim() || loading ? C.border : C.primary, color: !topic.trim() || loading ? C.textMuted : '#fff', fontSize: '16px', fontWeight: 600, cursor: !topic.trim() || loading ? 'not-allowed' : 'pointer', transition: 'all 200ms ease' }}
        >
          {loading ? '正在准备备课环境...' : '开始备课 →'}
        </button>
      </div>

      <div style={{ display: 'flex', justifyContent: 'center', gap: '8px', marginTop: '24px' }}>
        {shortcuts.map(item => (
          <button key={item.path} onClick={() => navigate(item.path)} style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', color: C.textSec, background: 'transparent', border: 'none', padding: '6px 12px', borderRadius: '8px', cursor: 'pointer', transition: 'all 150ms ease' }}
            onMouseEnter={e => { const el = e.currentTarget as HTMLButtonElement; el.style.background = C.primaryLight; el.style.color = C.primary }}
            onMouseLeave={e => { const el = e.currentTarget as HTMLButtonElement; el.style.background = 'transparent'; el.style.color = C.textSec }}
          ><span>{item.icon}</span><span>{item.text}</span></button>
        ))}
      </div>
    </div>
  )
}

/* ==================== 子组件：AI消息气泡 ==================== */
interface AIBubbleProps {
  msg: ConversationMessage
  streaming?: boolean
  onSelectComponent: (comp: ConvComponent) => void
  selectedComponentIds: Set<string>
}

function AIBubble({ msg, streaming = false, onSelectComponent, selectedComponentIds }: AIBubbleProps) {
  const [expandedComponent, setExpandedComponent] = useState<string | null>(null)
  return (
    <div style={{ display: 'flex', gap: '10px', marginBottom: '16px', alignItems: 'flex-start' }}>
      <div style={{ width: '32px', height: '32px', flexShrink: 0, background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '14px' }}>✨</div>
      <div style={{ flex: 1, maxWidth: 'calc(100% - 42px)' }}>
        {msg.content && (
          <div style={{ background: C.aiBubble, borderRadius: '0 12px 12px 12px', padding: '12px 16px', wordBreak: 'break-word' }}>
            {renderMarkdown(msg.content)}
            {streaming && (
              <span style={{ display: 'inline-block', width: '2px', height: '1em', background: C.primary, marginLeft: '2px', verticalAlign: 'text-bottom', animation: 'cursor-blink 0.8s step-end infinite' }} />
            )}
            <style>{`@keyframes cursor-blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }`}</style>
          </div>
        )}

        {msg.type === 'components' && msg.components && msg.components.length > 0 && (
          <div style={{ marginTop: '10px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {msg.components.map(comp => {
              const isSelected = selectedComponentIds.has(comp.id)
              const isExpanded = expandedComponent === comp.id
              return (
                <div key={comp.id} style={{ background: C.card, borderRadius: '10px', border: `1px solid ${isSelected ? C.primary : C.border}`, borderLeft: `3px solid ${C.accent}`, padding: '12px 14px', transition: 'all 200ms ease' }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{comp.display_label}</div>
                      {comp.usage_count > 0 && <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>{comp.usage_count}位老师用过 · 质量分{comp.quality_score.toFixed(1)}</div>}
                    </div>
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexShrink: 0, marginLeft: '12px' }}>
                      {comp.design_logic && (
                        <button onClick={() => setExpandedComponent(isExpanded ? null : comp.id)} style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '12px', color: C.textSec, cursor: 'pointer' }}>{isExpanded ? '收起' : '看逻辑'}</button>
                      )}
                      <button onClick={() => onSelectComponent(comp)} style={{ padding: '4px 12px', borderRadius: '6px', border: `1px solid ${isSelected ? C.primary : C.border}`, background: isSelected ? C.primaryLight : 'transparent', fontSize: '13px', color: isSelected ? C.primary : C.textSec, fontWeight: isSelected ? 600 : 400, cursor: 'pointer', transition: 'all 150ms ease' }}>{isSelected ? '✓ 已选' : '选择✓'}</button>
                    </div>
                  </div>
                  {isExpanded && comp.design_logic && (
                    <div style={{ marginTop: '10px', padding: '10px 12px', background: '#F9FAFB', borderRadius: '8px', fontSize: '13px', color: C.textSec, lineHeight: 1.7 }}>{comp.design_logic}</div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

/* ==================== 子组件：用户消息气泡 ==================== */
function UserBubble({ msg }: { msg: ConversationMessage }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: '16px' }}>
      <div style={{ maxWidth: '75%', background: C.userBubble, border: `1px solid ${C.border}`, borderRadius: '12px 0 12px 12px', padding: '10px 14px', fontSize: '15px', color: C.text, lineHeight: 1.7, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
        {msg.content}
      </div>
    </div>
  )
}

/* ==================== 子组件：思考中动画 ==================== */
function ThinkingIndicator() {
  return (
    <div style={{ display: 'flex', gap: '10px', marginBottom: '16px', alignItems: 'flex-start' }}>
      <div style={{ width: '32px', height: '32px', flexShrink: 0, background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '14px' }}>✨</div>
      <div style={{ background: C.aiBubble, borderRadius: '0 12px 12px 12px', padding: '14px 18px', display: 'flex', alignItems: 'center', gap: '6px' }}>
        {[0,1,2].map(i => (
          <div key={i} style={{ width: '6px', height: '6px', borderRadius: '50%', background: C.primary, animation: `lp-pulse 1.2s ease-in-out ${i * 0.2}s infinite` }} />
        ))}
        <style>{`@keyframes lp-pulse { 0%, 80%, 100% { opacity: 0.3; transform: scale(0.8); } 40% { opacity: 1; transform: scale(1.2); } }`}</style>
      </div>
    </div>
  )
}

/* ==================== 子组件：AI评审面板 ==================== */
interface ReviewPanelProps { review: AIReviewResult; onApply: (ids?: string[]) => void; applying: boolean }

function ReviewPanel({ review, onApply, applying }: ReviewPanelProps) {
  return (
    <div style={{ padding: '16px', height: '100%', overflowY: 'auto', boxSizing: 'border-box' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '16px', padding: '14px 16px', background: review.total_score >= 8.5 ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)', borderRadius: '10px', border: `1px solid ${review.total_score >= 8.5 ? '#10B98130' : '#F59E0B30'}` }}>
        <div style={{ fontSize: '28px', fontWeight: 700, flexShrink: 0, color: review.total_score >= 8.5 ? C.success : C.accent }}>{review.total_score.toFixed(1)}</div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>AI综合评分</div>
          <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px', lineHeight: 1.5 }}>{review.summary}</div>
        </div>
      </div>

      {review.good_points.length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.success, marginBottom: '8px' }}>✅ 做得好的</div>
          {review.good_points.map((point, i) => (
            <div key={i} style={{ fontSize: '13px', color: C.text, lineHeight: 1.6, padding: '6px 10px', marginBottom: '4px', background: 'rgba(16,185,129,0.06)', borderRadius: '6px' }}>{point}</div>
          ))}
        </div>
      )}

      {review.improvements.length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.accent, marginBottom: '8px' }}>💡 可以更好</div>
          {review.improvements.map(imp => (
            <div key={imp.id} style={{ marginBottom: '8px', padding: '10px 12px', background: 'rgba(245,158,11,0.06)', borderRadius: '8px', border: '1px solid rgba(245,158,11,0.15)' }}>
              <div style={{ fontSize: '13px', fontWeight: 500, color: C.text, marginBottom: '4px' }}>{imp.issue}</div>
              <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.6 }}>{imp.suggestion}</div>
            </div>
          ))}
        </div>
      )}

      <div style={{ display: 'flex', gap: '8px', flexDirection: 'column' }}>
        <button onClick={() => onApply()} disabled={applying} style={{ padding: '10px', borderRadius: '8px', border: 'none', background: applying ? C.border : C.primary, color: applying ? C.textMuted : '#fff', fontSize: '14px', fontWeight: 600, cursor: applying ? 'not-allowed' : 'pointer' }}>{applying ? '优化中...' : 'AI帮我优化 ✨'}</button>
        <button onClick={() => onApply([])} disabled={applying} style={{ padding: '10px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>就这样够用了 👍</button>
      </div>
    </div>
  )
}

/* ==================== 流式消息状态 ==================== */
interface StreamingState { id: string; content: string }

/* ==================== 主组件 ==================== */
export default function WorkshopPage() {
  const { token } = useAuth()
  const navigate   = useNavigate()
  const location   = useLocation()

  /* 恢复模式：从 location.state 获取已有教案ID */
  const resumePlanId = (location.state as { resumePlanId?: string } | null)?.resumePlanId

  /* 阶段状态：有resumePlanId时直接进入chatting */
  const [phase, setPhase]               = useState<'start' | 'chatting' | 'resuming'>(resumePlanId ? 'resuming' : 'start')
  const [startLoading, setStartLoading] = useState(false)
  const [resumeError, setResumeError]   = useState<string | null>(null)

  /* 教案数据 */
  const [plan, setPlan] = useState<LessonPlan | null>(null)

  /* 对话消息 */
  const [messages, setMessages]     = useState<ConversationMessage[]>([])
  const [isThinking, setIsThinking] = useState(false)
  const [streaming, setStreaming]   = useState<StreamingState | null>(null)

  /* 用户输入 */
  const [inputText, setInputText]                   = useState('')
  const [selectedComponentIds, setSelectedComponentIds] = useState<Set<string>>(new Set())

  /* 右侧面板 */
  const [planContent, setPlanContent]         = useState('')
  const [rightPanel, setRightPanel]           = useState<'preview' | 'review'>('preview')
  const [review, setReview]                   = useState<AIReviewResult | null>(null)
  const [reviewLoading, setReviewLoading]     = useState(false)
  const [applyingReview, setApplyingReview]   = useState(false)

  /* SSE引用 */
  const sseRef         = useRef<EventSource | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  /* 消息滚动 */
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isThinking, streaming?.content])

  /* 卸载时关闭SSE */
  useEffect(() => { return () => { sseRef.current?.close() } }, [])

  /* ==================== 建立SSE连接 ==================== */
  const connectSSE = useCallback((planId: string) => {
    if (!token) return
    sseRef.current?.close()
    sseRef.current = createLessonPlanSSE(planId, token, {
      onThinking: () => { setIsThinking(true); setStreaming(null) },
      onChunk: (chunk: string) => {
        setIsThinking(false)
        setStreaming(prev => prev
          ? { ...prev, content: prev.content + chunk }
          : { id: `stream_${Date.now()}`, content: chunk }
        )
      },
      onMessageDone: (msg: ConversationMessage) => {
        setIsThinking(false); setStreaming(null)
        setMessages(prev => [...prev, msg])
      },
      onContentUpdate: (content: string) => setPlanContent(content),
      onReviewDone: r => {
        setReviewLoading(false); setApplyingReview(false)
        setReview(r); setRightPanel('review')
      },
      onError: err => {
        setIsThinking(false); setStreaming(null)
        setReviewLoading(false); setApplyingReview(false)
        setMessages(prev => [...prev, {
          id: `err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
          content: `抱歉，遇到了一点问题：${err}。你可以重试或换个方式表达。`,
          created_at: new Date().toISOString(),
        }])
      },
    })
  }, [token])

  /* ==================== 恢复模式：加载已有教案和对话历史 ==================== */
  useEffect(() => {
    if (!resumePlanId || phase !== 'resuming') return

    const resumePlan = async () => {
      try {
        /* 并发拉取教案信息和对话历史 */
        const [planData, convData] = await Promise.all([
          getLessonPlan(resumePlanId),
          getConversation(resumePlanId),
        ])

        setPlan(planData)

        /* 恢复对话历史（过滤掉system角色消息，只显示user和assistant）*/
        const visibleMsgs = (convData.messages || []).filter(
          m => m.role === 'user' || m.role === 'assistant'
        )
        setMessages(visibleMsgs)

        /* 恢复教案内容到预览区 */
        if (planData.content_markdown) {
          setPlanContent(planData.content_markdown)
        }

        /* 恢复AI评审结果 */
        if (planData.ai_review_result) {
          try {
            const r = typeof planData.ai_review_result === 'string'
              ? JSON.parse(planData.ai_review_result)
              : planData.ai_review_result
            if (r && r.total_score) setReview(r)
          } catch { /* 忽略解析失败 */ }
        }

        setPhase('chatting')
        connectSSE(resumePlanId)
      } catch (e) {
        console.error('恢复教案失败:', e)
        setResumeError('加载教案失败，请重试')
        setPhase('start')
      }
    }

    resumePlan()
  }, [resumePlanId, phase, connectSSE])

  /* ==================== 开始新备课 ==================== */
  const handleStart = async (subject: string, grade: string, topic: string, duration: number) => {
    setStartLoading(true)
    try {
      const resp = await startConversation({ subject, grade, topic, duration_minutes: duration })
      setPlan(resp.plan)
      setMessages([resp.opening_message])
      setPhase('chatting')
      connectSSE(resp.plan.id)
    } catch (err) {
      console.error('开始备课失败:', err)
      alert('开始备课失败，请稍后重试')
    } finally {
      setStartLoading(false)
    }
  }

  /* ==================== 发送消息 ==================== */
  const handleSend = async () => {
    if (!plan || (!inputText.trim() && selectedComponentIds.size === 0)) return
    const msgText = inputText.trim()
    setInputText('')
    setMessages(prev => [...prev, {
      id: `local_${Date.now()}`, role: 'user' as const, type: 'text' as const,
      content: msgText || `已选择${selectedComponentIds.size}个组件`,
      created_at: new Date().toISOString(),
    }])
    setIsThinking(true)
    try {
      await sendChatMessage(plan.id, { message: msgText, selected_components: Array.from(selectedComponentIds) })
      setSelectedComponentIds(new Set())
    } catch (err) {
      setIsThinking(false); console.error('发送消息失败:', err)
    }
  }

  const handleSelectComponent = (comp: ConvComponent) => {
    setSelectedComponentIds(prev => { const next = new Set(prev); next.has(comp.id) ? next.delete(comp.id) : next.add(comp.id); return next })
  }

  const handleTriggerReview = async () => {
    if (!plan || reviewLoading) return
    setReviewLoading(true)
    try { await triggerAIReview(plan.id) } catch (err) { setReviewLoading(false); console.error('触发评审失败:', err) }
  }

  const handleApplySuggestions = async (ids?: string[]) => {
    if (!plan || applyingReview) return
    setApplyingReview(true)
    try { await applyAISuggestions(plan.id, ids) } catch (err) { setApplyingReview(false); console.error('应用建议失败:', err) }
  }

  const handlePublish = async () => {
    if (!plan) return
    try { await publishLessonPlanPersonal(plan.id); navigate('/lesson-plans/my-plans') }
    catch (err) { console.error('发布失败:', err); alert('发布失败，请稍后重试') }
  }

  /* ==================== 恢复中加载状态 ==================== */
  if (phase === 'resuming') {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '60vh', gap: '16px' }}>
        <div style={{ width: '36px', height: '36px', border: `3px solid ${C.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
        <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
        <div style={{ fontSize: '15px', color: C.textSec }}>正在恢复备课进度...</div>
        {resumeError && (
          <div style={{ fontSize: '14px', color: C.danger, marginTop: '8px' }}>
            {resumeError}
            <button onClick={() => navigate('/lesson-plans/my-plans')} style={{ marginLeft: '12px', color: C.primary, background: 'none', border: 'none', cursor: 'pointer', textDecoration: 'underline', fontSize: '14px' }}>返回我的教案</button>
          </div>
        )}
      </div>
    )
  }

  /* ==================== 首屏 ==================== */
  if (phase === 'start') {
    return (
      <div style={{ height: 'calc(100vh - 120px)', overflow: 'hidden', margin: '-28px -32px', display: 'flex', flexDirection: 'column' }}>
        <div style={{ flex: 1, overflowY: 'auto', padding: '0 32px' }}>
          <StartForm onStart={handleStart} loading={startLoading} />
        </div>
      </div>
    )
  }

  /* ==================== 备课中：三栏布局 ==================== */
  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 120px)', overflow: 'hidden', margin: '-28px -32px' }}>
      {/* 左栏：进度 */}
      <div style={{ width: '180px', flexShrink: 0, borderRight: `1px solid ${C.border}`, padding: '20px 12px', background: C.card, display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <div style={{ fontSize: '12px', fontWeight: 600, color: C.textMuted, marginBottom: '12px', letterSpacing: '0.5px' }}>备课进度</div>
        {([
          { key: 'info',     label: '了解学情', done: messages.length >= 2 },
          { key: 'plan',     label: '确认方案', done: messages.length >= 4 },
          { key: 'generate', label: '生成教案', done: !!planContent },
          { key: 'review',   label: 'AI评审',   done: !!review },
          { key: 'save',     label: '保存发布', done: false },
        ] as const).map((step, i) => (
          <div key={step.key} style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 10px', borderRadius: '8px' }}>
            <div style={{ width: '20px', height: '20px', borderRadius: '50%', flexShrink: 0, background: step.done ? C.success : i === 0 ? C.primary : C.border, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '10px', color: '#fff', fontWeight: 700 }}>
              {step.done ? '✓' : i + 1}
            </div>
            <span style={{ fontSize: '12px', color: step.done ? C.success : i === 0 ? C.text : C.textMuted, fontWeight: step.done ? 600 : 400 }}>{step.label}</span>
          </div>
        ))}
        {plan && (
          <div style={{ marginTop: 'auto', padding: '12px', background: C.bg, borderRadius: '10px', fontSize: '12px' }}>
            <div style={{ color: C.textMuted, marginBottom: '4px' }}>当前教案</div>
            <div style={{ color: C.text, fontWeight: 500, lineHeight: 1.5 }}>{plan.title}</div>
            {/* 恢复模式标记 */}
            {resumePlanId && (
              <div style={{ marginTop: '6px', fontSize: '11px', color: C.primary, background: C.primaryLight, padding: '2px 6px', borderRadius: '4px', display: 'inline-block' }}>
                🔄 继续编辑
              </div>
            )}
          </div>
        )}
      </div>

      {/* 中栏：对话区 */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', borderRight: `1px solid ${C.border}` }}>
        <div style={{ flex: 1, overflowY: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column' }}>
          {/* 恢复模式提示条 */}
          {resumePlanId && messages.length > 0 && (
            <div style={{ textAlign: 'center', marginBottom: '16px', padding: '8px 16px', background: C.primaryLight, borderRadius: '20px', fontSize: '12px', color: C.primary, alignSelf: 'center' }}>
              🔄 已恢复历史对话，可继续备课
            </div>
          )}

          {messages.map(msg =>
            msg.role === 'assistant'
              ? <AIBubble key={msg.id} msg={msg} streaming={false} onSelectComponent={handleSelectComponent} selectedComponentIds={selectedComponentIds} />
              : <UserBubble key={msg.id} msg={msg} />
          )}

          {streaming && (
            <AIBubble
              key={streaming.id}
              msg={{ id: streaming.id, role: 'assistant', type: 'text', content: streaming.content, created_at: new Date().toISOString() }}
              streaming={true}
              onSelectComponent={handleSelectComponent}
              selectedComponentIds={selectedComponentIds}
            />
          )}

          {isThinking && !streaming && <ThinkingIndicator />}
          <div ref={messagesEndRef} />
        </div>

        {selectedComponentIds.size > 0 && (
          <div style={{ padding: '8px 24px', background: C.primaryLight, borderTop: `1px solid ${C.border}`, fontSize: '13px', color: C.primary, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span>已选择 {selectedComponentIds.size} 个教学组件</span>
            <button onClick={() => setSelectedComponentIds(new Set())} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.textMuted, fontSize: '13px' }}>清空</button>
          </div>
        )}

        <div style={{ padding: '16px 24px', borderTop: `1px solid ${C.border}`, background: C.card }}>
          <div style={{ display: 'flex', gap: '10px', alignItems: 'flex-end', background: '#F9FAFB', borderRadius: '12px', border: `1px solid ${C.border}`, padding: '10px 12px' }}>
            <textarea
              value={inputText} onChange={e => setInputText(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }}
              placeholder="告诉AI你的想法... (Enter发送，Shift+Enter换行)"
              rows={2}
              style={{ flex: 1, background: 'transparent', border: 'none', outline: 'none', fontSize: '15px', color: C.text, resize: 'none', fontFamily: 'inherit', lineHeight: 1.6 }}
            />
            <button
              onClick={handleSend}
              disabled={isThinking || !!streaming || (!inputText.trim() && selectedComponentIds.size === 0)}
              style={{ width: '36px', height: '36px', flexShrink: 0, borderRadius: '50%', border: 'none', background: isThinking || streaming || (!inputText.trim() && selectedComponentIds.size === 0) ? C.border : C.primary, color: '#fff', cursor: 'pointer', fontSize: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center', transition: 'all 200ms ease' }}
            >→</button>
          </div>
          <div style={{ display: 'flex', gap: '8px', marginTop: '10px', flexWrap: 'wrap' }}>
            {([
              { label: '📝 生成教案', action: () => setInputText('好的，请开始生成完整教案') },
              { label: '🔍 AI评审',   action: handleTriggerReview },
              { label: '📋 查看预览', action: () => setRightPanel('preview') },
            ] as const).map(btn => (
              <button key={btn.label} onClick={btn.action} disabled={isThinking || !!streaming || reviewLoading}
                style={{ padding: '6px 12px', borderRadius: '20px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '12px', color: C.textSec, cursor: isThinking || streaming || reviewLoading ? 'not-allowed' : 'pointer', transition: 'all 150ms ease' }}
                onMouseEnter={e => { if (!isThinking && !streaming) (e.currentTarget as HTMLButtonElement).style.borderColor = C.primary }}
                onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = C.border }}
              >{btn.label}</button>
            ))}
          </div>
        </div>
      </div>

      {/* 右栏：预览/评审 */}
      <div style={{ width: '360px', flexShrink: 0, display: 'flex', flexDirection: 'column', background: C.card }}>
        <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, padding: '0 16px' }}>
          {([
            { key: 'preview', label: '📄 教案预览' },
            { key: 'review',  label: `🤖 AI评审${review ? ` ${review.total_score.toFixed(1)}` : ''}` },
          ] as const).map(tab => (
            <button key={tab.key} onClick={() => setRightPanel(tab.key)}
              style={{ padding: '14px 16px', border: 'none', background: 'transparent', fontSize: '13px', fontWeight: rightPanel === tab.key ? 600 : 400, color: rightPanel === tab.key ? C.primary : C.textSec, cursor: 'pointer', borderBottom: rightPanel === tab.key ? `2px solid ${C.primary}` : '2px solid transparent', marginBottom: '-1px', transition: 'all 150ms ease' }}
            >{tab.label}</button>
          ))}
        </div>

        <div style={{ flex: 1, overflow: 'hidden' }}>
          {rightPanel === 'preview' && (
            <div style={{ height: '100%', overflowY: 'auto', padding: '16px', boxSizing: 'border-box' }}>
              {planContent
                ? <div style={{ fontSize: '13px', lineHeight: 1.8 }}>{renderMarkdown(planContent)}</div>
                : (
                  <div style={{ height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: C.textMuted, textAlign: 'center', padding: '24px' }}>
                    <div style={{ fontSize: '32px', marginBottom: '12px' }}>📄</div>
                    <div style={{ fontSize: '14px', lineHeight: 1.6 }}>教案内容将在这里实时显示<br />点击"生成教案"开始</div>
                  </div>
                )
              }
            </div>
          )}

          {rightPanel === 'review' && (
            review
              ? <ReviewPanel review={review} onApply={handleApplySuggestions} applying={applyingReview} />
              : (
                <div style={{ height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: C.textMuted, textAlign: 'center', padding: '24px' }}>
                  <div style={{ fontSize: '32px', marginBottom: '12px' }}>🤖</div>
                  <div style={{ fontSize: '14px', lineHeight: 1.6, marginBottom: '16px' }}>生成教案后可触发AI评审<br />获取质量分析和改进建议</div>
                  {reviewLoading
                    ? <div style={{ fontSize: '13px', color: C.primary }}>AI正在评审中...</div>
                    : <button onClick={handleTriggerReview} disabled={!planContent} style={{ padding: '10px 20px', borderRadius: '8px', border: 'none', background: !planContent ? C.border : C.primary, color: !planContent ? C.textMuted : '#fff', fontSize: '14px', fontWeight: 600, cursor: !planContent ? 'not-allowed' : 'pointer' }}>触发AI评审</button>
                  }
                </div>
              )
          )}
        </div>

        {plan && (
          <div style={{ padding: '12px 16px', borderTop: `1px solid ${C.border}`, display: 'flex', gap: '8px' }}>
            <button onClick={() => navigate('/lesson-plans/my-plans')} style={{ flex: 1, padding: '9px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '13px', color: C.textSec, cursor: 'pointer' }}>保存草稿</button>
            <button onClick={handlePublish} style={{ flex: 1, padding: '9px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>发布教案 →</button>
          </div>
        )}
      </div>
    </div>
  )
}

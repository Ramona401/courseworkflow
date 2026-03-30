/**
 * WorkshopPage — 备课工坊首页（主文件）
 *
 * v3 新增：恢复模式（location.state.resumePlanId）
 * v2 改动：AI气泡Markdown渲染 + 流式输出
 *
 * 子组件均从 ./components/ 引入，本文件只保留：
 *   - SSE连接管理
 *   - 恢复模式加载逻辑
 *   - 所有操作函数（handleStart/handleSend/handleTriggerReview/...）
 *   - 三栏布局渲染框架
 */
import { useState, useRef, useEffect, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  startConversation, sendChatMessage, triggerAIReview, applyAISuggestions,
  publishLessonPlanPersonal, createLessonPlanSSE, getLessonPlan, getConversation,
  type LessonPlan, type ConversationMessage, type AIReviewResult, type ConvComponent,
} from '@/api/lesson-plans'

// ---- 子组件 ----
import { C, renderMarkdown, type StreamingState } from './components/workshopConstants'
import { StartForm, AIBubble, UserBubble, ThinkingIndicator, ReviewPanel } from './components/WorkshopPanels'

// ==================== 主组件 ====================
export default function WorkshopPage() {
  const { token } = useAuth()
  const navigate  = useNavigate()
  const location  = useLocation()

  // 恢复模式：从 location.state 获取已有教案ID
  const resumePlanId = (location.state as { resumePlanId?: string } | null)?.resumePlanId

  // 阶段状态：有resumePlanId时直接进入resuming
  const [phase, setPhase]               = useState<'start' | 'chatting' | 'resuming'>(resumePlanId ? 'resuming' : 'start')
  const [startLoading, setStartLoading] = useState(false)
  const [resumeError, setResumeError]   = useState<string | null>(null)

  // 教案数据
  const [plan, setPlan] = useState<LessonPlan | null>(null)

  // 对话消息
  const [messages, setMessages]     = useState<ConversationMessage[]>([])
  const [isThinking, setIsThinking] = useState(false)
  const [streaming, setStreaming]   = useState<StreamingState | null>(null)

  // 用户输入
  const [inputText, setInputText]                       = useState('')
  const [selectedComponentIds, setSelectedComponentIds] = useState<Set<string>>(new Set())

  // 右侧面板
  const [planContent, setPlanContent]       = useState('')
  const [rightPanel, setRightPanel]         = useState<'preview' | 'review'>('preview')
  const [review, setReview]                 = useState<AIReviewResult | null>(null)
  const [reviewLoading, setReviewLoading]   = useState(false)
  const [applyingReview, setApplyingReview] = useState(false)

  // SSE / 滚动引用
  const sseRef         = useRef<EventSource | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  // 消息自动滚动到底部
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isThinking, streaming?.content])

  // 卸载时关闭SSE
  useEffect(() => { return () => { sseRef.current?.close() } }, [])

  // ==================== 建立SSE连接 ====================
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

  // ==================== 恢复模式：加载教案和对话历史 ====================
  useEffect(() => {
    if (!resumePlanId || phase !== 'resuming') return
    const resumePlan = async () => {
      try {
        const [planData, convData] = await Promise.all([
          getLessonPlan(resumePlanId),
          getConversation(resumePlanId),
        ])
        setPlan(planData)
        // 只显示user和assistant消息，过滤system
        setMessages((convData.messages || []).filter(m => m.role === 'user' || m.role === 'assistant'))
        if (planData.content_markdown) setPlanContent(planData.content_markdown)
        // 恢复AI评审结果
        if (planData.ai_review_result) {
          try {
            const r = typeof planData.ai_review_result === 'string'
              ? JSON.parse(planData.ai_review_result) : planData.ai_review_result
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

  // ==================== 开始新备课 ====================
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
    } finally { setStartLoading(false) }
  }

  // ==================== 发送消息 ====================
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
    } catch (err) { setIsThinking(false); console.error('发送消息失败:', err) }
  }

  const handleSelectComponent = (comp: ConvComponent) => {
    setSelectedComponentIds(prev => { const next = new Set(prev); next.has(comp.id) ? next.delete(comp.id) : next.add(comp.id); return next })
  }

  const handleTriggerReview = async () => {
    if (!plan || reviewLoading) return
    setReviewLoading(true)
    try { await triggerAIReview(plan.id) }
    catch (err) { setReviewLoading(false); console.error('触发评审失败:', err) }
  }

  const handleApplySuggestions = async (ids?: string[]) => {
    if (!plan || applyingReview) return
    setApplyingReview(true)
    try { await applyAISuggestions(plan.id, ids) }
    catch (err) { setApplyingReview(false); console.error('应用建议失败:', err) }
  }

  const handlePublish = async () => {
    if (!plan) return
    try { await publishLessonPlanPersonal(plan.id); navigate('/lesson-plans/my-plans') }
    catch (err) { console.error('发布失败:', err); alert('发布失败，请稍后重试') }
  }

  // ==================== 恢复中加载状态 ====================
  if (phase === 'resuming') {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '60vh', gap: '16px' }}>
        <div style={{ width: '36px', height: '36px', border: `3px solid ${C.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
        <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
        <div style={{ fontSize: '15px', color: C.textSec }}>正在恢复备课进度...</div>
        {resumeError && (
          <div style={{ fontSize: '14px', color: C.danger, marginTop: '8px' }}>
            {resumeError}
            <button onClick={() => navigate('/lesson-plans/my-plans')} style={{ marginLeft: '12px', color: C.primary, background: 'none', border: 'none', cursor: 'pointer', textDecoration: 'underline', fontSize: '14px' }}>
              返回我的教案
            </button>
          </div>
        )}
      </div>
    )
  }

  // ==================== 首屏 ====================
  if (phase === 'start') {
    return (
      <div style={{ height: 'calc(100vh - 120px)', overflow: 'hidden', margin: '-28px -32px', display: 'flex', flexDirection: 'column' }}>
        <div style={{ flex: 1, overflowY: 'auto', padding: '0 32px' }}>
          <StartForm onStart={handleStart} loading={startLoading} />
        </div>
      </div>
    )
  }

  // ==================== 备课中：三栏布局 ====================
  const isBusy = isThinking || !!streaming || reviewLoading

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 120px)', overflow: 'hidden', margin: '-28px -32px' }}>

      {/* ---- 左栏：备课进度 ---- */}
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
            {resumePlanId && (
              <div style={{ marginTop: '6px', fontSize: '11px', color: C.primary, background: C.primaryLight, padding: '2px 6px', borderRadius: '4px', display: 'inline-block' }}>
                🔄 继续编辑
              </div>
            )}
          </div>
        )}
      </div>

      {/* ---- 中栏：对话区 ---- */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', borderRight: `1px solid ${C.border}` }}>
        {/* 消息列表 */}
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

        {/* 已选组件提示栏 */}
        {selectedComponentIds.size > 0 && (
          <div style={{ padding: '8px 24px', background: C.primaryLight, borderTop: `1px solid ${C.border}`, fontSize: '13px', color: C.primary, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span>已选择 {selectedComponentIds.size} 个教学组件</span>
            <button onClick={() => setSelectedComponentIds(new Set())} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.textMuted, fontSize: '13px' }}>清空</button>
          </div>
        )}

        {/* 输入区 */}
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
              disabled={isBusy || (!inputText.trim() && selectedComponentIds.size === 0)}
              style={{ width: '36px', height: '36px', flexShrink: 0, borderRadius: '50%', border: 'none', background: isBusy || (!inputText.trim() && selectedComponentIds.size === 0) ? C.border : C.primary, color: '#fff', cursor: 'pointer', fontSize: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center', transition: 'all 200ms ease' }}>
              →
            </button>
          </div>
          {/* 快捷操作按钮 */}
          <div style={{ display: 'flex', gap: '8px', marginTop: '10px', flexWrap: 'wrap' }}>
            {[
              { label: '📝 生成教案', action: () => setInputText('好的，请开始生成完整教案') },
              { label: '🔍 AI评审',   action: handleTriggerReview },
              { label: '📋 查看预览', action: () => setRightPanel('preview') },
            ].map(btn => (
              <button key={btn.label} onClick={btn.action} disabled={isBusy}
                style={{ padding: '6px 12px', borderRadius: '20px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '12px', color: C.textSec, cursor: isBusy ? 'not-allowed' : 'pointer', transition: 'all 150ms ease' }}
                onMouseEnter={e => { if (!isBusy) (e.currentTarget as HTMLButtonElement).style.borderColor = C.primary }}
                onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = C.border }}>
                {btn.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* ---- 右栏：预览/评审 ---- */}
      <div style={{ width: '360px', flexShrink: 0, display: 'flex', flexDirection: 'column', background: C.card }}>
        {/* Tab切换 */}
        <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, padding: '0 16px' }}>
          {([
            { key: 'preview', label: '📄 教案预览' },
            { key: 'review',  label: `🤖 AI评审${review ? ` ${review.total_score.toFixed(1)}` : ''}` },
          ] as const).map(tab => (
            <button key={tab.key} onClick={() => setRightPanel(tab.key)}
              style={{ padding: '14px 16px', border: 'none', background: 'transparent', fontSize: '13px', fontWeight: rightPanel === tab.key ? 600 : 400, color: rightPanel === tab.key ? C.primary : C.textSec, cursor: 'pointer', borderBottom: rightPanel === tab.key ? `2px solid ${C.primary}` : '2px solid transparent', marginBottom: '-1px', transition: 'all 150ms ease' }}>
              {tab.label}
            </button>
          ))}
        </div>

        <div style={{ flex: 1, overflow: 'hidden' }}>
          {/* 教案预览 */}
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

          {/* AI评审 */}
          {rightPanel === 'review' && (
            review
              ? <ReviewPanel review={review} onApply={handleApplySuggestions} applying={applyingReview} />
              : (
                <div style={{ height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: C.textMuted, textAlign: 'center', padding: '24px' }}>
                  <div style={{ fontSize: '32px', marginBottom: '12px' }}>🤖</div>
                  <div style={{ fontSize: '14px', lineHeight: 1.6, marginBottom: '16px' }}>
                    生成教案后可触发AI评审<br />获取质量分析和改进建议
                  </div>
                  {reviewLoading
                    ? <div style={{ fontSize: '13px', color: C.primary }}>AI正在评审中...</div>
                    : <button onClick={handleTriggerReview} disabled={!planContent} style={{ padding: '10px 20px', borderRadius: '8px', border: 'none', background: !planContent ? C.border : C.primary, color: !planContent ? C.textMuted : '#fff', fontSize: '14px', fontWeight: 600, cursor: !planContent ? 'not-allowed' : 'pointer' }}>触发AI评审</button>
                  }
                </div>
              )
          )}
        </div>

        {/* 底部操作：保存草稿 / 发布教案 */}
        {plan && (
          <div style={{ padding: '12px 16px', borderTop: `1px solid ${C.border}`, display: 'flex', gap: '8px' }}>
            <button onClick={() => navigate('/lesson-plans/my-plans')} style={{ flex: 1, padding: '9px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '13px', color: C.textSec, cursor: 'pointer' }}>
              保存草稿
            </button>
            <button onClick={handlePublish} style={{ flex: 1, padding: '9px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
              发布教案 →
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

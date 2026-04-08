/**
 * WorkshopPage — 备课工坊主页面
 *
 * 迭代8 重大重构（P1+P2+P3）：
 *   P1：阶段隔离 + 用户手动完成触发
 *   P2：阶段过渡弹窗 + 结构化产出展示（方案B）
 *   P3：叙事式过渡动画
 * 迭代12 新增：
 *   阶段过渡时弹出组件推荐弹窗（方案B组件交互）
 * v88 新增（P2-3 断线恢复与SSE韧性）：
 *   - 网络状态指示器（绿/黄/红）
 *   - SSE自动重连（指数退避，最多5次）
 *   - 重连后自动拉取最新对话补齐丢失消息
 *   - 消息发送失败自动重试1次
 */
import { useState, useRef, useEffect, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  startConversation, sendChatMessage, triggerAIReview, applyAISuggestions,
  publishLessonPlanPersonal, createLessonPlanSSE, getLessonPlan, getConversation,
  getStageStatus, advanceStage, skipStage, backStage, getStageOutput, resetStage, switchToStage, getStageCompleteness,
  type LessonPlan, type ConversationMessage, type AIReviewResult, type ConvComponent,
  type StageProgressItem, type StageEventData, type StageCompletenessResponse,
  type SSEConnectionState, type SSEConnection,
} from '@/api/lesson-plans'
import {
  C, renderMarkdown, type StreamingState,
  STAGE_STATUS_ICON, STAGE_STATUS_COLOR, STAGE_CODE_EMOJI,
} from './components/workshopConstants'
import {
  StartForm, AIBubble, UserBubble, ThinkingIndicator, ReviewPanel,
} from './components/WorkshopPanels'
import {
  StageSummaryModal, StageTransitionView, StageSeparatorBubble,
} from './components/WorkshopTransitionComponents'
import StageComponentsModal from './components/StageComponentsModal'
import { getAssessmentResult } from '@/api/assessment'

const STAGE_SEP_PREFIX = '__STAGE_SEP__'

// 迭代12：有组件映射的阶段列表（revise无组件）
const STAGES_WITH_COMPONENTS = ['analyze', 'design', 'write', 'review']

// v88：消息发送最大重试次数
const SEND_RETRY_MAX = 1

export default function WorkshopPage() {
  const { token } = useAuth()
  const navigate  = useNavigate()
  const location  = useLocation()

  const resumePlanId = (location.state as { resumePlanId?: string } | null)?.resumePlanId
  const sessionPlanId = sessionStorage.getItem('workshop_active_plan_id')
  const effectivePlanId = resumePlanId || sessionPlanId || null

  const [phase, setPhase]               = useState<'start' | 'chatting' | 'resuming'>(effectivePlanId ? 'resuming' : 'start')
  const [startLoading, setStartLoading] = useState(false)
  const [resumeError, setResumeError]   = useState<string | null>(null)

  const [plan, setPlan] = useState<LessonPlan | null>(null)
  const [messages, setMessages]     = useState<ConversationMessage[]>([])
  const [isThinking, setIsThinking] = useState(false)
  const [streaming, setStreaming]   = useState<StreamingState | null>(null)
  const [inputText, setInputText]   = useState('')
  const [selectedComponentIds, setSelectedComponentIds] = useState<Set<string>>(new Set())

  const [planContent, setPlanContent]       = useState('')
  const [rightPanel, setRightPanel]         = useState<'preview' | 'review' | 'stages'>('preview')
  const [review, setReview]                 = useState<AIReviewResult | null>(null)
  const [reviewLoading, setReviewLoading]   = useState(false)
  const [applyingReview, setApplyingReview] = useState(false)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)

  const [stageItems, setStageItems]     = useState<StageProgressItem[]>([])
  const [currentStage, setCurrentStage] = useState<string>('')
  const [isStageMode, setIsStageMode]   = useState(false)
  const isStageModeRef = useRef(false)
  const [needsAssessment, setNeedsAssessment] = useState<boolean | null>(null)
  const [isStageProcessing, setIsStageProcessing] = useState(false)

  // P1：AI建议完成提示
  const [aiSuggestsComplete, setAiSuggestsComplete] = useState(false)

  // P2：弹窗状态（方案B新增 stageCode + structuredOutput）
  const [showSummaryModal, setShowSummaryModal] = useState(false)
  const [summaryLoading, setSummaryLoading]     = useState(false)
  const [stageSummary, setStageSummary]         = useState('')
  const [stageStructured, setStageStructured]   = useState('{}')

  // P3：过渡动画
  const [isTransitioning, setIsTransitioning]   = useState(false)
  const [transitionStep, setTransitionStep]     = useState(0)
  const [transitionInfo, setTransitionInfo]     = useState<{
    currentName: string; nextName: string; nextRole: string
  } | null>(null)

  // P0-2：阶段完成度状态
  const [stageCompleteness, setStageCompleteness] = useState<StageCompletenessResponse | null>(null)

  // 迭代12：阶段组件推荐弹窗状态
  const [showComponentsModal, setShowComponentsModal] = useState(false)
  const [pendingTransitionStage, setPendingTransitionStage] = useState<string | null>(null)

  // v77：阶段视图切换状态（null=显示当前阶段，指定stageCode=查看该阶段历史对话）
  const [viewingStage, setViewingStage] = useState<string | null>(null)

  // v88新增：SSE连接状态（connected=绿色 | reconnecting=黄色 | disconnected=红色）
  const [sseConnectionState, setSseConnectionState] = useState<SSEConnectionState>('connected')

  // v88：SSE连接引用改为SSEConnection类型（支持close方法）
  const sseRef         = useRef<SSEConnection | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  // v88：保存planId的ref，供重连回调使用（避免闭包捕获旧值）
  const planIdRef = useRef<string | null>(null)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isThinking, streaming?.content])

  useEffect(() => { return () => { sseRef.current?.close() } }, [])

  useEffect(() => {
    if (phase !== 'start') return
    getAssessmentResult()
      .then(resp => { setNeedsAssessment(!resp.has_profile) })
      .catch(() => { setNeedsAssessment(false) })
  }, [phase])

  const refreshStages = useCallback(async (planId: string) => {
    try {
      const resp = await getStageStatus(planId)
      setStageItems(resp.stages)
      setCurrentStage(resp.current_stage)
      setIsStageMode(true)
      isStageModeRef.current = true
    } catch {
      setIsStageMode(false)
      isStageModeRef.current = false
    }
  }, [])

  // v88重构：connectSSE增加连接状态回调和重连补齐逻辑
  const connectSSE = useCallback((planId: string) => {
    if (!token) return
    sseRef.current?.close()
    planIdRef.current = planId

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
        if (isStageModeRef.current) {
          setIsStageProcessing(true)
          setTimeout(() => setIsStageProcessing(false), 5000)
        }
      },
      onContentUpdate: (content: string) => setPlanContent(content),
      onReviewDone: r => {
        setReviewLoading(false); setApplyingReview(false)
        setReview(r); setRightPanel('review')
      },
      onStageStarted: (_data: StageEventData) => { refreshStages(planId) },
      onStageComplete: (_data: StageEventData) => {
        setIsStageProcessing(false)
        refreshStages(planId)
      },
      onStageOutput: (_data: StageEventData) => { refreshStages(planId) },
      onError: err => {
        setIsThinking(false); setStreaming(null)
        setReviewLoading(false); setApplyingReview(false)
        setIsStageProcessing(false)
        setMessages(prev => [...prev, {
          id: `err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
          content: `抱歉，遇到了一点问题：${err}。你可以重试或换个方式表达。`,
          created_at: new Date().toISOString(),
        }])
      },

      // v88新增：连接状态变化回调 → 驱动顶部指示器颜色
      onConnectionStateChange: (state: SSEConnectionState) => {
        setSseConnectionState(state)
      },

      // v88新增：重连成功后自动补齐丢失消息
      onReconnected: async () => {
        const currentPlanId = planIdRef.current
        if (!currentPlanId) return
        try {
          console.log('[SSE] 重连成功，开始补齐对话消息...')
          // 拉取服务端最新的完整对话记录
          const convData = await getConversation(currentPlanId)
          const serverMsgs = (convData.messages || []).filter(
            (m: ConversationMessage) => m.role === 'user' || m.role === 'assistant' || m.role === 'system'
          )
          // 用服务端完整消息替换本地消息（服务端是真实来源）
          // 只在服务端消息数量大于本地时才替换，避免重连瞬间本地正在streaming的消息被覆盖
          setMessages(prev => {
            if (serverMsgs.length > prev.length) {
              console.log(`[SSE] 补齐完成：本地${prev.length}条 → 服务端${serverMsgs.length}条`)
              return serverMsgs
            }
            console.log(`[SSE] 无需补齐：本地${prev.length}条 >= 服务端${serverMsgs.length}条`)
            return prev
          })
          // 同时刷新教案内容和阶段状态
          const planData = await getLessonPlan(currentPlanId)
          if (planData.content_markdown) setPlanContent(planData.content_markdown)
          if (planData.current_stage && planData.stage_config) {
            await refreshStages(currentPlanId)
          }
          // 清理可能残留的streaming状态
          setIsThinking(false)
          setStreaming(null)
        } catch (err) {
          console.error('[SSE] 重连后补齐消息失败:', err)
        }
      },
    })
  }, [token, refreshStages])

  useEffect(() => {
    if (!effectivePlanId || phase !== 'resuming') return
    const resumePlan = async () => {
      try {
        const [planData, convData] = await Promise.all([
          getLessonPlan(effectivePlanId),
          getConversation(effectivePlanId),
        ])
        setPlan(planData)
        setMessages((convData.messages || []).filter(m => m.role === 'user' || m.role === 'assistant' || m.role === 'system'))
        if (planData.content_markdown) setPlanContent(planData.content_markdown)
        if (planData.ai_review_result) {
          try {
            const r = typeof planData.ai_review_result === 'string'
              ? JSON.parse(planData.ai_review_result) : planData.ai_review_result
            if (r && r.total_score) setReview(r)
          } catch { /* 忽略 */ }
        }
        if (planData.current_stage && planData.stage_config) {
          await refreshStages(effectivePlanId)
        }
        setPhase('chatting')
        sessionStorage.setItem('workshop_active_plan_id', effectivePlanId)
        connectSSE(effectivePlanId)
      } catch (e) {
        console.error('恢复教案失败:', e)
        setResumeError('加载教案失败，请重试')
        setPhase('start')
      }
    }
    resumePlan()
  }, [effectivePlanId, phase, connectSSE, refreshStages])

  const handleStart = async (subject: string, grade: string, topic: string, duration: number, recipeId?: string, textbookPageIds?: string[]) => {
    setStartLoading(true)
    try {
      const req: Record<string, unknown> = { subject, grade, topic, duration_minutes: duration }
      if (recipeId) req.recipe_id = recipeId
      if (textbookPageIds && textbookPageIds.length > 0) req.textbook_page_ids = textbookPageIds
      const resp = await startConversation(req as Parameters<typeof startConversation>[0])
      setPlan(resp.plan)
      setMessages([resp.opening_message])
      setPhase('chatting')
      sessionStorage.setItem('workshop_active_plan_id', resp.plan.id)
      connectSSE(resp.plan.id)
      if (resp.plan.current_stage && resp.plan.stage_config) {
        await refreshStages(resp.plan.id)
      }
    } catch (err) {
      console.error('开始备课失败:', err)
      alert('开始备课失败，请稍后重试')
    } finally { setStartLoading(false) }
  }

  // v88增强：消息发送增加重试机制
  const handleSend = async () => {
    if (!plan || (!inputText.trim() && selectedComponentIds.size === 0)) return
    const msgText = inputText.trim()
    setInputText('')

    // 乐观更新：立即将用户消息显示在UI中
    const localMsg: ConversationMessage = {
      id: `local_${Date.now()}`, role: 'user' as const, type: 'text' as const,
      content: msgText || `已选择${selectedComponentIds.size}个组件`,
      created_at: new Date().toISOString(),
    }
    setMessages(prev => [...prev, localMsg])
    setIsThinking(true)

    const componentIds = Array.from(selectedComponentIds)

    // 带重试的发送逻辑
    let lastErr: unknown = null
    for (let attempt = 0; attempt <= SEND_RETRY_MAX; attempt++) {
      try {
        await sendChatMessage(plan.id, { message: msgText, selected_components: componentIds })
        setSelectedComponentIds(new Set())
        lastErr = null
        break
      } catch (err) {
        lastErr = err
        if (attempt < SEND_RETRY_MAX) {
          console.warn(`[Send] 发送失败，${1}秒后重试第${attempt + 1}次...`)
          await new Promise(resolve => setTimeout(resolve, 1000))
        }
      }
    }

    // 所有重试都失败
    if (lastErr) {
      setIsThinking(false)
      console.error('发送消息失败（含重试）:', lastErr)
      setMessages(prev => [...prev, {
        id: `send_err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
        content: '⚠️ 消息发送失败，请检查网络后重试。你刚才的内容已保留在输入框中。',
        created_at: new Date().toISOString(),
      }])
      // 将消息内容恢复到输入框，方便用户重试
      setInputText(msgText)
    }
  }

  const handleSelectComponent = (comp: ConvComponent) => {
    setSelectedComponentIds(prev => {
      const next = new Set(prev); next.has(comp.id) ? next.delete(comp.id) : next.add(comp.id); return next
    })
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
    try {
      await publishLessonPlanPersonal(plan.id)
      sessionStorage.removeItem('workshop_active_plan_id')
      navigate('/lesson-plans/my-plans')
    } catch (err) { console.error('发布失败:', err); alert('发布失败，请稍后重试') }
  }

  // ==================== v79-2：退出备课（保存草稿+回到首屏）====================
  const handleExitWorkshop = () => {
    if (!plan) return
    const confirmMsg = '确定退出当前备课吗？\n\n教案已自动保存为草稿，你可以随时从「我的教案」继续。'
    if (!confirm(confirmMsg)) return
    // 关闭SSE连接
    sseRef.current?.close()
    sseRef.current = null
    // 清除session状态
    sessionStorage.removeItem('workshop_active_plan_id')
    // 重置所有状态回到首屏
    setPlan(null)
    setMessages([])
    setPlanContent('')
    setReview(null)
    setStageItems([])
    setCurrentStage('')
    setIsStageMode(false)
    isStageModeRef.current = false
    setViewingStage(null)
    setAiSuggestsComplete(false)
    setIsThinking(false)
    setStreaming(null)
    setInputText('')
    setSelectedComponentIds(new Set())
    setSseConnectionState('connected')  // v88：重置连接状态
    setPhase('start')
  }

  // v88新增：手动重连按钮（disconnected状态时可用）
  const handleManualReconnect = () => {
    if (!plan) return
    setSseConnectionState('reconnecting')
    connectSSE(plan.id)
  }

  // ==================== P2：点击完成本阶段 ====================
  const handleCompleteStageClick = async () => {
    if (!plan || !currentStage) return
    setSummaryLoading(true)
    setShowSummaryModal(true)
    setStageSummary('')
    setStageStructured('{}')
    setStageCompleteness(null)
    try {
      const [output, completeness] = await Promise.all([
        getStageOutput(plan.id, currentStage),
        getStageCompleteness(plan.id, currentStage).catch(() => null),
      ])
      setStageSummary(output.narrative_output || '')
      setStageStructured(output.structured_output || '{}')
      if (completeness) setStageCompleteness(completeness)
    } catch {
      setStageSummary('')
      setStageStructured('{}')
    }
    setSummaryLoading(false)
  }

  // ==================== 迭代12：实际执行 advanceStage 并插入分隔符 ====================
  const doAdvanceStage = async (planId: string, nextStageItem: StageProgressItem | null, selectedCompIds: string[]) => {
    // 插入阶段分隔符气泡
    if (nextStageItem) {
      const sepMsg = {
        id: `stage_sep_${Date.now()}`,
        role: 'system' as const,
        type: 'text' as const,
        content: `${STAGE_SEP_PREFIX}${nextStageItem.stage_name}__${nextStageItem.ai_role}`,
        created_at: new Date().toISOString(),
      }
      setMessages(prev => [...prev, sepMsg as ConversationMessage])
    }
    try {
      await advanceStage(planId, undefined, selectedCompIds.length > 0 ? selectedCompIds : undefined)
      await refreshStages(planId)
      setAiSuggestsComplete(false)
      setViewingStage(null)
    } catch (err) { console.error('进入下一阶段失败:', err) }
  }

  // ==================== 迭代12：组件弹窗回调 ====================
  const handleComponentsConfirm = async (selectedIds: string[]) => {
    if (!plan) return
    setShowComponentsModal(false)
    const nextItem = stageItems.find(s => s.stage_code === pendingTransitionStage) || null
    await doAdvanceStage(plan.id, nextItem, selectedIds)
    setPendingTransitionStage(null)
  }

  const handleComponentsSkip = async () => {
    if (!plan) return
    setShowComponentsModal(false)
    const nextItem = stageItems.find(s => s.stage_code === pendingTransitionStage) || null
    await doAdvanceStage(plan.id, nextItem, [])
    setPendingTransitionStage(null)
  }

  // ==================== P2+P3：确认进入下一阶段 ====================
  const handleConfirmTransition = async () => {
    if (!plan) return
    setShowSummaryModal(false)

    const currentIdx = stageItems.findIndex(s => s.stage_code === currentStage)
    const currentStageItem = stageItems[currentIdx]
    const nextStageItem = currentIdx >= 0 && currentIdx < stageItems.length - 1
      ? stageItems[currentIdx + 1] : null
    const isLastStage = !nextStageItem

    if (isLastStage) {
      sessionStorage.removeItem('workshop_active_plan_id')
      navigate('/lesson-plans/my-plans')
      return
    }

    // P3：启动过渡动画
    setTransitionInfo({
      currentName: currentStageItem?.stage_name || currentStage,
      nextName: nextStageItem.stage_name,
      nextRole: nextStageItem.ai_role,
    })
    setIsTransitioning(true)
    setTransitionStep(0)

    const t1 = setTimeout(() => setTransitionStep(1), 700)
    const t2 = setTimeout(() => setTransitionStep(2), 1400)

    // 过渡动画结束后：判断是否需要组件弹窗
    const t3 = setTimeout(() => {
      setIsTransitioning(false)
      setTransitionStep(0)
      setTransitionInfo(null)

      // 迭代12：判断下一阶段是否有组件映射
      if (nextStageItem && STAGES_WITH_COMPONENTS.includes(nextStageItem.stage_code)) {
        // 有组件映射 → 弹出组件推荐弹窗，暂不调用 advanceStage
        setPendingTransitionStage(nextStageItem.stage_code)
        setShowComponentsModal(true)
      } else {
        // 无组件映射（如revise）→ 直接进入下一阶段
        doAdvanceStage(plan.id, nextStageItem, [])
      }
    }, 2200)

    return () => { clearTimeout(t1); clearTimeout(t2); clearTimeout(t3) }
  }

  const handleSkipStageQuick = async () => {
    if (!plan) return
    try { await skipStage(plan.id); await refreshStages(plan.id) }
    catch (err) { console.error('跳过阶段失败:', err) }
  }

  const handleBackStageQuick = async () => {
    if (!plan) return
    try {
      await backStage(plan.id)
      setAiSuggestsComplete(false)
      await refreshStages(plan.id)
    } catch (err) { console.error('回退阶段失败:', err) }
  }

  // 迭代12新增：重启指定阶段
  const handleResetStage = async (stageCode: string) => {
    if (!plan) return
    const stageName = stageItems.find(s => s.stage_code === stageCode)?.stage_name || stageCode
    if (!confirm(`确定要重启「${stageName}」阶段吗？\n\n该阶段及之后阶段的产出物和对话将被清空。`)) return
    try {
      await resetStage(plan.id, stageCode)
      // v77：截断前端消息（保留目标阶段分隔符之前的消息）
      const targetItem = stageItems.find(s => s.stage_code === stageCode)
      if (targetItem) {
        const sepIdx = messages.findIndex(m =>
          (m.role as string) === 'system' && m.content.startsWith(STAGE_SEP_PREFIX) &&
          m.content.includes(targetItem.stage_name)
        )
        if (sepIdx >= 0) {
          setMessages(prev => prev.slice(0, sepIdx))
        } else {
          setMessages([])
        }
      } else {
        setMessages([])
      }
      if (stageCode === 'write' || stageCode === 'revise') setPlanContent('')
      setReview(null)
      setAiSuggestsComplete(false)
      setViewingStage(null)
      await refreshStages(plan.id)
      connectSSE(plan.id)
    } catch (err) { console.error('重启阶段失败:', err); alert('重启阶段失败，请重试') }
  }

  // ==================== 恢复中 ====================
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
          {needsAssessment === true && (
            <div style={{ margin: '24px auto 20px', maxWidth: '600px', padding: '24px 28px', background: 'linear-gradient(135deg, rgba(79,123,232,0.06) 0%, rgba(16,185,129,0.06) 100%)', borderRadius: '16px', border: '1px solid rgba(79,123,232,0.15)', textAlign: 'center' }}>
              <div style={{ fontSize: '32px', marginBottom: '12px' }}>🎓</div>
              <h3 style={{ fontSize: '18px', fontWeight: 700, color: '#1F2937', margin: '0 0 8px' }}>欢迎！先来了解一下您的备课风格</h3>
              <p style={{ fontSize: '14px', color: '#6B7280', lineHeight: 1.6, margin: '0 0 20px' }}>只需2-3分钟的轻松对话，AI就能为您量身定制备课配方</p>
              <div style={{ display: 'flex', gap: '12px', justifyContent: 'center' }}>
                <button onClick={() => navigate('/lesson-plans/assessment')} style={{ padding: '10px 24px', borderRadius: '10px', border: 'none', background: '#4F7BE8', color: '#fff', fontSize: '15px', fontWeight: 600, cursor: 'pointer' }}>开始风格前测 →</button>
                <button onClick={() => setNeedsAssessment(false)} style={{ padding: '10px 20px', borderRadius: '10px', border: '1px solid #E5E7EB', background: 'transparent', fontSize: '14px', color: '#9CA3AF', cursor: 'pointer' }}>以后再说</button>
              </div>
            </div>
          )}
          <StartForm onStart={handleStart} loading={startLoading} />
        </div>
      </div>
    )
  }

  // ==================== 备课中 ====================
  const isAIActive = isThinking || !!streaming || isStageProcessing
  const isViewingHistory = !!(viewingStage && viewingStage !== currentStage)
  const isBusy = isAIActive || reviewLoading || isTransitioning || isViewingHistory
  const canCompleteStage = isStageMode && currentStage && !isAIActive && !isTransitioning && !summaryLoading

  const currentStageIdx = stageItems.findIndex(s => s.stage_code === currentStage)
  const nextStageForSummary = currentStageIdx >= 0 && currentStageIdx < stageItems.length - 1
    ? stageItems[currentStageIdx + 1] : null

  const planAny = plan as Record<string, unknown> | null
  const recipeName = planAny?.recipe_name ? String(planAny.recipe_name) : ''
  const recipeId   = planAny?.recipe_id   ? String(planAny.recipe_id)   : ''

  const fallbackSteps = [
    { key: 'info', label: '了解学情', done: messages.length >= 2 },
    { key: 'plan', label: '确认方案', done: messages.length >= 4 },
    { key: 'generate', label: '生成教案', done: !!planContent },
    { key: 'review', label: 'AI评审', done: !!review },
    { key: 'save', label: '保存发布', done: false },
  ]

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 120px)', overflow: 'hidden', margin: '-28px -32px' }}>

      {/* 左栏 */}
      <div style={{ width: sidebarCollapsed ? '48px' : '180px', flexShrink: 0, borderRight: `1px solid ${C.border}`, padding: sidebarCollapsed ? '12px 4px' : '20px 12px', background: C.card, display: 'flex', flexDirection: 'column', gap: '4px', transition: 'width 200ms ease, padding 200ms ease', overflow: 'hidden' }}>
        <button onClick={() => setSidebarCollapsed(prev => !prev)} title={sidebarCollapsed ? '展开侧栏' : '收起侧栏'} style={{ display: 'flex', alignItems: 'center', justifyContent: sidebarCollapsed ? 'center' : 'space-between', width: '100%', padding: '6px 8px', borderRadius: '8px', border: 'none', background: 'transparent', cursor: 'pointer', fontSize: '12px', color: C.textMuted, marginBottom: '8px', whiteSpace: 'nowrap' }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = '#F3F4F6' }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}>
          {sidebarCollapsed ? <span style={{ fontSize: '14px' }}>»</span> : <><span style={{ fontWeight: 600, letterSpacing: '0.5px' }}>备课进度</span><span style={{ fontSize: '14px' }}>«</span></>}
        </button>

        {/* v79-2：退出备课快捷入口（展开时显示） */}
        {!sidebarCollapsed && plan && (
          <button onClick={handleExitWorkshop} style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '4px', width: '100%', padding: '5px 8px', borderRadius: '6px', border: `1px dashed ${C.border}`, background: 'transparent', fontSize: '11px', color: C.textMuted, cursor: 'pointer', marginBottom: '4px', transition: 'all 150ms ease' }}
            onMouseEnter={e => { const el = e.currentTarget as HTMLElement; el.style.borderColor = '#EF4444'; el.style.color = '#EF4444'; el.style.background = 'rgba(239,68,68,0.04)' }}
            onMouseLeave={e => { const el = e.currentTarget as HTMLElement; el.style.borderColor = C.border; el.style.color = C.textMuted; el.style.background = 'transparent' }}>
            🚪 退出备课
          </button>
        )}

        {!sidebarCollapsed && (
          isStageMode && stageItems.length > 0
            ? stageItems.map(stage => {
                const isCurrent = stage.stage_code === currentStage
                const isViewing = viewingStage === stage.stage_code
                const statusColor = STAGE_STATUS_COLOR[stage.status] || C.textMuted
                const statusIcon  = STAGE_STATUS_ICON[stage.status]  || '○'
                // v77：点击阶段切换查看历史对话（已完成/进行中的阶段才可点击）
                const canView = stage.status === 'completed' || stage.status === 'in_progress' || isCurrent
                return (
                  <div key={stage.stage_code} onClick={() => { if (canView) setViewingStage(isViewing || isCurrent ? null : stage.stage_code) }} style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 10px', borderRadius: '8px', background: isViewing ? 'rgba(79,123,232,0.12)' : isCurrent ? C.primaryLight : 'transparent', transition: 'background 150ms ease', cursor: canView ? 'pointer' : 'default', border: isViewing ? '1px solid rgba(79,123,232,0.3)' : '1px solid transparent' }}>
                    <div style={{ width: '22px', height: '22px', borderRadius: '50%', flexShrink: 0, background: stage.status === 'completed' ? C.success : isCurrent ? C.primary : '#E5E7EB', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '10px', color: '#fff', fontWeight: 700, border: isCurrent && stage.status !== 'completed' ? `2px solid ${C.primary}` : 'none' }}>
                      {stage.status === 'completed' ? '✓' : stage.status === 'skipped' ? '⊘' : stage.stage_order}
                    </div>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: '12px', fontWeight: isCurrent ? 600 : 400, color: isCurrent ? C.primary : stage.status === 'completed' ? C.success : C.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{stage.stage_name}</div>
                      <div style={{ fontSize: '10px', color: statusColor, marginTop: '1px' }}>{statusIcon} {stage.ai_role}</div>
                      {isCurrent && stageCompleteness && stageCompleteness.stage_code === stage.stage_code && (
                        <div style={{ fontSize: '10px', marginTop: '2px', color: stageCompleteness.percentage >= 80 ? '#10B981' : '#F59E0B', fontWeight: 600 }}>
                          {stageCompleteness.percentage}% 完成
                        </div>
                      )}
                    </div>
                  </div>
                )
              })
            : fallbackSteps.map((step, i) => (
                <div key={step.key} style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 10px', borderRadius: '8px' }}>
                  <div style={{ width: '20px', height: '20px', borderRadius: '50%', flexShrink: 0, background: step.done ? C.success : i === 0 ? C.primary : '#E5E7EB', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '10px', color: '#fff', fontWeight: 700 }}>{step.done ? '✓' : i + 1}</div>
                  <span style={{ fontSize: '12px', color: step.done ? C.success : i === 0 ? C.text : C.textMuted, fontWeight: step.done ? 600 : 400 }}>{step.label}</span>
                </div>
              ))
        )}

        {sidebarCollapsed && (
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '12px', marginTop: '4px' }}>
            {isStageMode && stageItems.length > 0
              ? stageItems.map(s => (
                  <div key={s.stage_code} title={`${s.stage_name} — ${s.ai_role}`} style={{ width: '28px', height: '28px', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '13px', background: s.status === 'completed' ? 'rgba(16,185,129,0.1)' : s.stage_code === currentStage ? C.primaryLight : '#F3F4F6', border: s.status === 'completed' ? '1px solid rgba(16,185,129,0.3)' : s.stage_code === currentStage ? `1px solid ${C.primary}` : '1px solid transparent' }}>
                    {STAGE_CODE_EMOJI[s.stage_code] || '📋'}
                  </div>
                ))
              : [{icon:'📋',done:messages.length>=2,title:'了解学情'},{icon:'📝',done:messages.length>=4,title:'确认方案'},{icon:'📄',done:!!planContent,title:'生成教案'},{icon:'🤖',done:!!review,title:'AI评审'},{icon:'💾',done:false,title:'保存发布'}].map(s => (
                  <div key={s.title} title={s.title} style={{ width: '28px', height: '28px', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '13px', background: s.done ? 'rgba(16,185,129,0.1)' : '#F3F4F6', border: s.done ? '1px solid rgba(16,185,129,0.3)' : '1px solid transparent' }}>{s.icon}</div>
                ))
            }
          </div>
        )}

        {!sidebarCollapsed && plan && (
          <div style={{ marginTop: 'auto', padding: '12px', background: C.bg, borderRadius: '10px', fontSize: '12px' }}>
            <div style={{ color: C.textMuted, marginBottom: '4px' }}>当前教案</div>
            <div style={{ color: C.text, fontWeight: 500, lineHeight: 1.5 }}>{plan.title}</div>
            {recipeName ? (
              <div style={{ marginTop: '8px', padding: '8px 10px', background: 'rgba(245,158,11,0.06)', borderRadius: '8px', border: '1px solid rgba(245,158,11,0.12)' }}>
                <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '3px' }}>📦 备课配方</div>
                <button onClick={() => navigate(`/lesson-plans/recipes/${recipeId}/edit`, { state: { from: '/lesson-plans' } })} style={{ fontSize: '12px', color: C.primary, background: 'none', border: 'none', cursor: 'pointer', padding: 0, textDecoration: 'underline', fontWeight: 500, textAlign: 'left' }}>{recipeName}</button>
              </div>
            ) : (
              <button onClick={() => navigate('/lesson-plans/recipes')} style={{ marginTop: '8px', display: 'flex', alignItems: 'center', gap: '4px', fontSize: '11px', color: C.textMuted, background: 'none', border: `1px dashed ${C.border}`, borderRadius: '6px', padding: '6px 8px', cursor: 'pointer', width: '100%', justifyContent: 'center' }}>📦 添加配方</button>
            )}
          </div>
        )}

        {sidebarCollapsed && plan && (
          <div style={{ marginTop: 'auto', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
            <div title={plan.title} style={{ fontSize: '18px' }}>📝</div>
            {recipeName && <button onClick={() => navigate(`/lesson-plans/recipes/${recipeId}/edit`, { state: { from: '/lesson-plans' } })} title={`配方：${recipeName}`} style={{ fontSize: '18px', cursor: 'pointer', background: 'none', border: 'none', padding: 0 }}>📦</button>}
          </div>
        )}
      </div>

      {/* 中栏 */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', borderRight: `1px solid ${C.border}`, position: 'relative' }}>
        {isTransitioning && transitionInfo && (
          <StageTransitionView currentStageName={transitionInfo.currentName} nextStageName={transitionInfo.nextName} nextStageRole={transitionInfo.nextRole} step={transitionStep} />
        )}

        {/* v88新增：网络状态指示器（仅在非connected状态时显示） */}
        {phase === 'chatting' && sseConnectionState !== 'connected' && (
          <div style={{
            padding: '7px 16px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            fontSize: '13px',
            fontWeight: 500,
            borderBottom: `1px solid ${sseConnectionState === 'reconnecting' ? 'rgba(245,158,11,0.3)' : 'rgba(239,68,68,0.3)'}`,
            background: sseConnectionState === 'reconnecting'
              ? 'linear-gradient(135deg, rgba(245,158,11,0.08), rgba(251,191,36,0.05))'
              : 'linear-gradient(135deg, rgba(239,68,68,0.08), rgba(248,113,113,0.05))',
            color: sseConnectionState === 'reconnecting' ? '#92400E' : '#991B1B',
            animation: sseConnectionState === 'reconnecting' ? 'sseReconnectPulse 2s ease-in-out infinite' : 'none',
          }}>
            {/* 状态圆点 */}
            <div style={{
              width: '8px', height: '8px', borderRadius: '50%', flexShrink: 0,
              background: sseConnectionState === 'reconnecting' ? '#F59E0B' : '#EF4444',
              boxShadow: sseConnectionState === 'reconnecting'
                ? '0 0 6px rgba(245,158,11,0.5)'
                : '0 0 6px rgba(239,68,68,0.5)',
            }} />
            {sseConnectionState === 'reconnecting' ? (
              <span>网络连接中断，正在尝试重新连接...</span>
            ) : (
              <>
                <span>网络连接已断开</span>
                <button onClick={handleManualReconnect} style={{
                  padding: '3px 12px', borderRadius: '12px',
                  border: '1px solid rgba(239,68,68,0.4)',
                  background: 'rgba(239,68,68,0.08)',
                  fontSize: '12px', color: '#DC2626', cursor: 'pointer',
                  fontWeight: 600,
                }}>点击重连</button>
              </>
            )}
          </div>
        )}

        {isStageMode && currentStage && (() => {
          const cur = stageItems.find(s => s.stage_code === currentStage)
          return (
            <div style={{ padding: '8px 20px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: 'linear-gradient(135deg, rgba(79,123,232,0.06), rgba(129,140,248,0.04))' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <span style={{ fontSize: '16px' }}>{STAGE_CODE_EMOJI[currentStage] || '📋'}</span>
                <div>
                  <span style={{ fontSize: '13px', fontWeight: 600, color: C.primary }}>{cur?.stage_name || currentStage}</span>
                  {cur?.ai_role && <span style={{ fontSize: '11px', color: C.textMuted, marginLeft: '8px' }}>· {cur.ai_role}</span>}
                </div>
                {/* v88：阶段标题栏右侧的小型连接状态点 */}
                <div title={
                  sseConnectionState === 'connected' ? '连接正常' :
                  sseConnectionState === 'reconnecting' ? '重连中...' : '连接断开'
                } style={{
                  width: '6px', height: '6px', borderRadius: '50%', marginLeft: '4px',
                  background: sseConnectionState === 'connected' ? '#10B981' :
                              sseConnectionState === 'reconnecting' ? '#F59E0B' : '#EF4444',
                  boxShadow: sseConnectionState === 'connected'
                    ? '0 0 4px rgba(16,185,129,0.4)'
                    : sseConnectionState === 'reconnecting'
                    ? '0 0 4px rgba(245,158,11,0.4)'
                    : '0 0 4px rgba(239,68,68,0.4)',
                  transition: 'background 300ms ease, box-shadow 300ms ease',
                }} />
              </div>
              <span style={{ fontSize: '12px', color: C.textMuted }}>{currentStageIdx + 1} / {stageItems.length}</span>
            </div>
          )
        })()}

        <div style={{ flex: 1, overflowY: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column' }}>
          {resumePlanId && messages.length > 0 && (
            <div style={{ textAlign: 'center', marginBottom: '16px', padding: '8px 16px', background: C.primaryLight, borderRadius: '20px', fontSize: '12px', color: C.primary, alignSelf: 'center' }}>
              🔄 已恢复历史对话，可继续备课
            </div>
          )}
          {(() => {
            // v77：按阶段分隔符过滤消息 — viewingStage非null时只显示该阶段的消息
            const targetStage = viewingStage || currentStage
            let filteredMsgs = messages
            if (isStageMode && targetStage && stageItems.length > 0) {
              // 找到目标阶段的分隔符索引和下一个分隔符索引
              let startIdx = -1
              let endIdx = messages.length
              for (let i = 0; i < messages.length; i++) {
                const m = messages[i]
                if ((m.role as string) === 'system' && m.content.startsWith(STAGE_SEP_PREFIX)) {
                  const rest = m.content.slice(STAGE_SEP_PREFIX.length)
                  const sepStageName = rest.split('__')[0] || ''
                  const matchItem = stageItems.find(s => s.stage_name === sepStageName || s.stage_code === sepStageName)
                  if (matchItem && matchItem.stage_code === targetStage) {
                    startIdx = i  // 包含分隔符本身
                  } else if (startIdx >= 0 && endIdx === messages.length) {
                    endIdx = i  // 下一个分隔符位置
                  }
                }
              }
              if (startIdx >= 0) {
                filteredMsgs = messages.slice(startIdx, endIdx)
              } else if (targetStage === stageItems[0]?.stage_code) {
                // 第一个阶段没有分隔符，显示从头到第一个分隔符之前的消息
                const firstSepIdx = messages.findIndex(m => (m.role as string) === 'system' && m.content.startsWith(STAGE_SEP_PREFIX))
                filteredMsgs = firstSepIdx >= 0 ? messages.slice(0, firstSepIdx) : messages
              }
            }
            // v77：过滤掉系统自动触发的指令消息（不显示给用户）
            return filteredMsgs.filter(m => {
              if (m.role === 'user' && m.content.startsWith('我们进入') && m.content.includes('阶段了。请先简要介绍')) return false
              if (m.role === 'user' && m.content === '请对上一阶段完成的教案进行全面专业评审，直接输出评审报告，包含各维度评分和改进建议。') return false
              return true
            })
          })().map(msg => {
            if ((msg.role as string) === 'system' && msg.content.startsWith(STAGE_SEP_PREFIX)) {
              const rest = msg.content.slice(STAGE_SEP_PREFIX.length)
              const [stageName, aiRole] = rest.split('__')
              return <StageSeparatorBubble key={msg.id} stageName={stageName || ''} aiRole={aiRole || ''} />
            }
            return msg.role === 'assistant'
              ? <AIBubble key={msg.id} msg={msg} streaming={false} onSelectComponent={handleSelectComponent} selectedComponentIds={selectedComponentIds} />
              : <UserBubble key={msg.id} msg={msg} />
          })}
          {streaming && (
            <AIBubble key={streaming.id} msg={{ id: streaming.id, role: 'assistant', type: 'text', content: streaming.content, created_at: new Date().toISOString() }} streaming={true} onSelectComponent={handleSelectComponent} selectedComponentIds={selectedComponentIds} />
          )}
          {isThinking && !streaming && <ThinkingIndicator />}
          <div ref={messagesEndRef} />
        </div>

        {isStageMode && viewingStage && viewingStage !== currentStage && (
          <div style={{ padding: '9px 20px', background: 'linear-gradient(135deg, rgba(79,123,232,0.08), rgba(129,140,248,0.05))', borderTop: '1px solid rgba(79,123,232,0.18)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', fontSize: '13px' }}>
            <span style={{ color: C.primary }}>📖 正在查看「{stageItems.find(s => s.stage_code === viewingStage)?.stage_name || viewingStage}」阶段的历史对话</span>
            <div style={{ display: 'flex', gap: '6px' }}>
              <button onClick={async () => { if (!plan) return; try { await switchToStage(plan.id, viewingStage!); await refreshStages(plan.id); setViewingStage(null) } catch { alert('回退失败') } }} style={{ padding: '4px 12px', borderRadius: '12px', border: '1px solid #10B981', background: 'transparent', fontSize: '12px', color: '#10B981', cursor: 'pointer' }}>💬 继续该阶段对话</button>
              <button onClick={() => { handleResetStage(viewingStage!) }} style={{ padding: '4px 12px', borderRadius: '12px', border: '1px solid #EF4444', background: 'transparent', fontSize: '12px', color: '#EF4444', cursor: 'pointer' }}>🔄 重启该阶段</button>
              <button onClick={() => setViewingStage(null)} style={{ padding: '4px 12px', borderRadius: '12px', border: `1px solid ${C.primary}`, background: 'transparent', fontSize: '12px', color: C.primary, cursor: 'pointer' }}>回到当前阶段 →</button>
            </div>
          </div>
        )}

        {isStageMode && aiSuggestsComplete && !isTransitioning && (
          <div style={{ padding: '9px 20px', background: 'linear-gradient(135deg, rgba(16,185,129,0.08), rgba(52,211,153,0.05))', borderTop: '1px solid rgba(16,185,129,0.18)', display: 'flex', alignItems: 'center', gap: '10px', fontSize: '13px' }}>
            <span style={{ fontSize: '16px' }}>✨</span>
            <span style={{ color: '#065F46', fontWeight: 500 }}>AI认为本阶段工作已完成，你可以继续深入探讨，或点击下方按钮进入下一阶段</span>
          </div>
        )}

        {isStageProcessing && (
          <div style={{ padding: '8px 20px', background: 'rgba(79,123,232,0.07)', borderTop: `1px solid rgba(79,123,232,0.14)`, display: 'flex', alignItems: 'center', gap: '10px', fontSize: '13px', color: C.primary }}>
            <div style={{ width: '14px', height: '14px', border: `2px solid ${C.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite', flexShrink: 0 }} />
            <span>正在整理阶段成果，请稍候...</span>
            <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
          </div>
        )}

        {selectedComponentIds.size > 0 && (
          <div style={{ padding: '8px 20px', background: C.primaryLight, borderTop: `1px solid ${C.border}`, fontSize: '13px', color: C.primary, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span>已选择 {selectedComponentIds.size} 个教学组件</span>
            <button onClick={() => setSelectedComponentIds(new Set())} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.textMuted, fontSize: '13px' }}>清空</button>
          </div>
        )}

        <div style={{ padding: '14px 20px', borderTop: `1px solid ${C.border}`, background: C.card }}>
          <div style={{ display: 'flex', gap: '10px', alignItems: 'flex-end', background: '#F9FAFB', borderRadius: '12px', border: `1px solid ${C.border}`, padding: '10px 12px' }}>
            <textarea value={inputText} onChange={e => setInputText(e.target.value)} onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }} placeholder={isBusy ? 'AI处理中...' : sseConnectionState === 'disconnected' ? '网络已断开，请先重连...' : '告诉AI你的想法... (Enter发送，Shift+Enter换行)'} rows={2} disabled={isBusy} style={{ flex: 1, background: 'transparent', border: 'none', outline: 'none', fontSize: '15px', color: C.text, resize: 'none', fontFamily: 'inherit', lineHeight: 1.6, opacity: isBusy ? 0.5 : 1 }} />
            <button onClick={handleSend} disabled={isBusy || (!inputText.trim() && selectedComponentIds.size === 0)} style={{ width: '36px', height: '36px', flexShrink: 0, borderRadius: '50%', border: 'none', background: isBusy || (!inputText.trim() && selectedComponentIds.size === 0) ? '#E5E7EB' : C.primary, color: '#fff', cursor: 'pointer', fontSize: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center', transition: 'all 200ms ease' }}>→</button>
          </div>

          <div style={{ display: 'flex', gap: '8px', marginTop: '10px', flexWrap: 'wrap', alignItems: 'center' }}>
            {[
              ...(!isStageMode ? [{ label: '🔍 AI评审', action: handleTriggerReview, disabled: isBusy }] : []),
              { label: '📄 预览教案', action: () => setRightPanel('preview'), disabled: false },
              ...(isStageMode ? [{ label: '📊 阶段产出', action: () => setRightPanel('stages'), disabled: false }] : []),
            ].map(btn => (
              <button key={btn.label} onClick={btn.action} disabled={btn.disabled} style={{ padding: '6px 12px', borderRadius: '20px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '12px', color: C.textSec, cursor: btn.disabled ? 'not-allowed' : 'pointer', opacity: btn.disabled ? 0.5 : 1, transition: 'all 150ms ease' }}
                onMouseEnter={e => { if (!btn.disabled) (e.currentTarget as HTMLButtonElement).style.borderColor = C.primary }}
                onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = C.border }}>
                {btn.label}
              </button>
            ))}

            {isStageMode && currentStage && (
              <button onClick={handleCompleteStageClick} disabled={!canCompleteStage} style={{ marginLeft: 'auto', padding: '7px 16px', borderRadius: '20px', border: 'none', background: !canCompleteStage ? '#E5E7EB' : aiSuggestsComplete ? 'linear-gradient(135deg, #10B981, #34D399)' : 'linear-gradient(135deg, #4F7BE8, #818CF8)', color: !canCompleteStage ? C.textMuted : '#fff', fontSize: '13px', fontWeight: 600, cursor: !canCompleteStage ? 'not-allowed' : 'pointer', transition: 'all 200ms ease', boxShadow: canCompleteStage && aiSuggestsComplete ? '0 3px 12px rgba(16,185,129,0.35)' : canCompleteStage ? '0 3px 10px rgba(79,123,232,0.3)' : 'none', animation: canCompleteStage && aiSuggestsComplete ? 'completePulse 2s ease-in-out infinite' : 'none', whiteSpace: 'nowrap' }}>
                {summaryLoading ? '加载中...' : nextStageForSummary ? `✅ 完成本阶段，进入${nextStageForSummary.stage_name} →` : '🎉 完成备课'}
              </button>
            )}
          </div>

          {isStageMode && currentStageIdx > 0 && (
            <div style={{ display: 'flex', gap: '6px', marginTop: '6px' }}>
              <button onClick={handleBackStageQuick} disabled={isBusy} style={{ padding: '4px 10px', borderRadius: '12px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '11px', color: C.textMuted, cursor: isBusy ? 'not-allowed' : 'pointer' }}>← 回到上一阶段</button>
              <button onClick={() => handleResetStage(currentStage)} disabled={isBusy} style={{ padding: '4px 10px', borderRadius: '12px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '11px', color: '#EF4444', cursor: isBusy ? 'not-allowed' : 'pointer' }}>🔄 重启本阶段</button>
              {nextStageForSummary?.skippable && (
                <button onClick={handleSkipStageQuick} disabled={isBusy} style={{ padding: '4px 10px', borderRadius: '12px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '11px', color: C.textMuted, cursor: isBusy ? 'not-allowed' : 'pointer' }}>跳过下一阶段 →</button>
              )}
            </div>
          )}
        </div>
      </div>

      {/* 右栏 */}
      <div style={{ width: '420px', flexShrink: 0, display: 'flex', flexDirection: 'column', background: C.card }}>
        <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, padding: '0 16px' }}>
          {([
            { key: 'preview' as const, label: '📄 教案预览' },
            { key: 'review'  as const, label: `🤖 AI评审${review ? ` ${review.total_score.toFixed(1)}` : ''}` },
            ...(isStageMode ? [{ key: 'stages' as const, label: '📊 阶段产出' }] : []),
          ]).map(tab => (
            <button key={tab.key} onClick={() => setRightPanel(tab.key)} style={{ padding: '14px 16px', border: 'none', background: 'transparent', fontSize: '13px', fontWeight: rightPanel === tab.key ? 600 : 400, color: rightPanel === tab.key ? C.primary : C.textSec, cursor: 'pointer', borderBottom: rightPanel === tab.key ? `2px solid ${C.primary}` : '2px solid transparent', marginBottom: '-1px', transition: 'all 150ms ease' }}>
              {tab.label}
            </button>
          ))}
        </div>

        <div style={{ flex: 1, overflow: 'hidden' }}>
          {rightPanel === 'preview' && (
            <div style={{ height: '100%', overflowY: 'auto', padding: '16px', boxSizing: 'border-box' }}>
              {planContent
                ? <div style={{ fontSize: '13px', lineHeight: 1.8 }}>{renderMarkdown(planContent)}</div>
                : <div style={{ height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: C.textMuted, textAlign: 'center', padding: '24px' }}>
                    <div style={{ fontSize: '32px', marginBottom: '12px' }}>📄</div>
                    <div style={{ fontSize: '14px', lineHeight: 1.6 }}>教案内容将在这里实时显示<br />进行到"教案撰写"阶段后自动更新</div>
                  </div>
              }
            </div>
          )}
          {rightPanel === 'review' && (
            review && review.total_score
              ? <ReviewPanel review={review} onApply={handleApplySuggestions} applying={applyingReview} isStageMode={isStageMode} />
              : <div style={{ height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: C.textMuted, textAlign: 'center', padding: '24px' }}>
                  <div style={{ fontSize: '32px', marginBottom: '12px' }}>🤖</div>
                  {isStageMode ? (
                    <div style={{ fontSize: '14px', lineHeight: 1.6, color: C.textMuted }}>
                      进行到「AI评审」阶段后<br />评审报告将自动显示在这里
                    </div>
                  ) : (
                    <>
                      <div style={{ fontSize: '14px', lineHeight: 1.6, marginBottom: '16px' }}>生成教案后可触发AI评审<br />获取质量分析和改进建议</div>
                      {reviewLoading ? <div style={{ fontSize: '13px', color: C.primary }}>AI正在评审中...</div>
                        : <button onClick={handleTriggerReview} disabled={!planContent} style={{ padding: '10px 20px', borderRadius: '8px', border: 'none', background: !planContent ? '#E5E7EB' : C.primary, color: !planContent ? C.textMuted : '#fff', fontSize: '14px', fontWeight: 600, cursor: !planContent ? 'not-allowed' : 'pointer' }}>触发AI评审</button>
                      }
                    </>
                  )}
                </div>
          )}
          {rightPanel === 'stages' && (
            <div style={{ height: '100%', overflowY: 'auto', padding: '16px', boxSizing: 'border-box' }}>
              {stageItems.length > 0 ? (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                  {stageItems.map(stage => (
                    <div key={stage.stage_code} onClick={() => { const canClick = stage.status === 'completed' || stage.status === 'in_progress'; if (canClick) setViewingStage(stage.stage_code === currentStage ? null : stage.stage_code) }} style={{ padding: '14px 16px', borderRadius: '10px', border: `1px solid ${viewingStage === stage.stage_code ? 'rgba(79,123,232,0.5)' : stage.stage_code === currentStage ? C.primary : C.border}`, background: viewingStage === stage.stage_code ? 'rgba(79,123,232,0.06)' : stage.status === 'completed' ? 'rgba(16,185,129,0.04)' : C.card, cursor: stage.status === 'completed' || stage.status === 'in_progress' ? 'pointer' : 'default', transition: 'all 150ms ease' }}>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '6px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <span style={{ fontSize: '14px' }}>{STAGE_CODE_EMOJI[stage.stage_code] || '📋'}</span>
                          <span style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{stage.stage_name}</span>
                        </div>
                        <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '10px', background: stage.status === 'completed' ? 'rgba(16,185,129,0.1)' : stage.status === 'in_progress' ? C.primaryLight : '#F3F4F6', color: stage.status === 'completed' ? C.success : stage.status === 'in_progress' ? C.primary : C.textMuted }}>
                          {stage.status === 'completed' ? '已完成' : stage.status === 'in_progress' ? '进行中' : stage.status === 'skipped' ? '已跳过' : '待开始'}
                        </span>
                      </div>
                      <div style={{ fontSize: '12px', color: C.textMuted }}>{stage.ai_role}</div>
                      {stage.has_output && <div style={{ marginTop: '8px', fontSize: '12px', color: C.primary }}>📎 已有阶段产出物</div>}
                    </div>
                  ))}
                </div>
              ) : (
                <div style={{ height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: C.textMuted, textAlign: 'center', padding: '24px' }}>
                  <div style={{ fontSize: '32px', marginBottom: '12px' }}>📊</div>
                  <div style={{ fontSize: '14px', lineHeight: 1.6 }}>各阶段产出物将在这里展示</div>
                </div>
              )}
            </div>
          )}
        </div>

        {plan && (
          <div style={{ padding: '12px 16px', borderTop: `1px solid ${C.border}`, display: 'flex', gap: '8px' }}>
            <button onClick={handleExitWorkshop} style={{ flex: 1, padding: '9px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '13px', color: C.textSec, cursor: 'pointer' }} title="退出备课，教案自动保存为草稿">🚪 退出备课</button>
            <button onClick={handlePublish} style={{ flex: 1, padding: '9px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>发布教案 →</button>
          </div>
        )}
      </div>

      {/* 迭代12：阶段组件推荐弹窗 */}
      {showComponentsModal && plan && pendingTransitionStage && (
        <StageComponentsModal
          planId={plan.id}
          stageCode={pendingTransitionStage}
          stageName={stageItems.find(s => s.stage_code === pendingTransitionStage)?.stage_name || pendingTransitionStage}
          onConfirm={handleComponentsConfirm}
          onSkip={handleComponentsSkip}
          onCancel={() => { setShowComponentsModal(false); setPendingTransitionStage(null) }}
        />
      )}

      {/* P2弹窗 */}
      {showSummaryModal && plan && (
        <StageSummaryModal
          stageCode={currentStage}
          stageName={stageItems.find(s => s.stage_code === currentStage)?.stage_name || currentStage}
          stageOrder={currentStageIdx + 1}
          totalStages={stageItems.length}
          nextStageItem={nextStageForSummary}
          structuredOutput={stageStructured}
          narrative={stageSummary}
          loading={summaryLoading}
          onConfirm={handleConfirmTransition}
          onCancel={() => setShowSummaryModal(false)}
          completeness={stageCompleteness}
        />
      )}

      <style>{`
        @keyframes completePulse {
          0%, 100% { box-shadow: 0 3px 12px rgba(16,185,129,0.35); }
          50%       { box-shadow: 0 3px 20px rgba(16,185,129,0.6); transform: translateY(-1px); }
        }
        @keyframes sseReconnectPulse {
          0%, 100% { opacity: 1; }
          50%       { opacity: 0.7; }
        }
      `}</style>
    </div>
  )
}

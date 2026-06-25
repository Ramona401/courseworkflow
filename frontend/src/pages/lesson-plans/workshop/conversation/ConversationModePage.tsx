/**
 * ConversationModePage.tsx — 唤起式对话备课主页面
 * （迭代3.5 Phase A + A2系列 + B-2 + 路线乙第2步 + 子轮二 B2轮次/三层超时）
 *
 * 双栏布局：左=导师对话 / 右=教案画布；阶段对老师隐藏；剧本芯片驱动推进。
 *
 * 已拆分子模块：
 *   ConversationStartScreen / ConversationCanvas / ConversationInputBar / ConversationTopBar /
 *   ConversationChipRow / TextbookAttachModal / RetryControls /
 *   useConversationSSE（SSE+turnID过滤+三层超时） / useRetryLastMessage（重试harness） /
 *   conversationChips（纯逻辑）
 *
 * 子轮二（B2 轮次序号 + 三层超时兜底）——本页承担的编排：
 *   - currentTurnRef：每发起一轮 chat 自增的轮次序号；发请求时带 client_turn_id，
 *     并同步给 useConversationSSE 做事件过滤（丢弃过期轮次的迟到回复）；
 *   - startTurnTimers：每轮发起时启动 8s 软提示 + 90s 看门狗；
 *   - onSlowHint / onRetryNotice：设置 slowHintText，在思考指示器旁显示安抚/重试文案；
 *   - onWatchdogTimeout：90s 无任何本轮事件 → 复位 thinking + 插一条人话 + 让重试可用 +
 *     推进 currentTurnRef 作废本轮（此后该轮迟到回复都会被前端过滤掉，不污染上下文）。
 *
 * 大单元挂载（前端入口·起步选）：
 *   首屏 ConversationStartScreen 的「所属单元方案（选填）」下拉选中的 unitPlanId 由本页持有，
 *   handleStart 建会话拿到 plan.id 后，若 unitPlanId 非空则调 updatePlanUnitPlan 落库
 *   （独立 try-catch 吞错——挂载失败绝不阻断建会话主流程，教案已建好，归属是锦上添花）。
 *   后端注入层每轮对话重读 lesson_plans.unit_plan_id 决定是否注入单元方案上下文（仅 active），
 *   故落库后第一条用户消息起自动生效，无需刷新。挂载放在 connectSSE 之前，语义最干净。
 */
import { useState, useRef, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  startConversation, sendChatMessage, publishLessonPlanPersonal,
  getLessonPlan, getConversation,
  getStageStatus, advanceStage, switchToStage,
  type LessonPlan, type ConversationMessage, type ConvComponent,
  type StageProgressItem,
} from '@/api/lesson-plans'
import { updatePlanUnitPlan } from '@/api/unit-plans'
import { C, SUBJECTS, GRADES, STAGE_CODE_NAME, type StreamingState } from '../components/workshopConstants'
import { AIBubble, UserBubble, ThinkingIndicator } from '../components/WorkshopPanels'
import ImportPlanModal from '../components/ImportPlanModal'
import StageComponentsModal from '../components/StageComponentsModal'
import ConversationCanvas from './ConversationCanvas'
import ConversationStartScreen from './ConversationStartScreen'
import ConversationInputBar from './ConversationInputBar'
import ConversationTopBar from './ConversationTopBar'
import ConversationChipRow from './ConversationChipRow'
import RetryControls from './RetryControls'
import TextbookAttachModal from './TextbookAttachModal'
import { useConversationSSE } from './useConversationSSE'
import { useRetryLastMessage } from './useRetryLastMessage'
import { computeVisibleChips, shouldHideHistoryMessage, isFullLessonPlanMessage } from './conversationChips'
import {
  STAGE_SEP_PREFIX, STAGES_WITH_COMPONENTS, FULL_GEN_STAGE_META, INPUT_PLACEHOLDERS,
} from './conversationScript'
import { dispatchChip, type ChipContext } from './chipActions'
import { recordPlanMode } from './workshopMode'
import AssistantSwitcher from './AssistantSwitcher'
import { getAssistantPref, type AssistantPref } from '@/api/assistant-prefs'

interface ConversationModePageProps {
  onSwitchMode?: () => void
}

export default function ConversationModePage({ onSwitchMode }: ConversationModePageProps) {
  const { token } = useAuth()
  const navigate = useNavigate()

  const sessionPlanId = sessionStorage.getItem('workshop_active_plan_id')
  const [phase, setPhase] = useState<'start' | 'chatting' | 'resuming'>(sessionPlanId ? 'resuming' : 'start')

  const [subject, setSubject] = useState(SUBJECTS[0])
  const [grade, setGrade] = useState(GRADES[0])
  const [topic, setTopic] = useState('')
  // 大单元挂载（起步选）：首屏「所属单元方案（选填）」下拉选中的方案ID（空串=不关联）。
  // 本页持有，透传首屏；handleStart 建会话后落库，落库成功即清空（临时选择已固化到教案）。
  const [unitPlanId, setUnitPlanId] = useState('')
  // v203 新增：对话模式首屏配方选择（选了传 recipe_id，没选就不传）
  const [recipeId, setRecipeId] = useState('')
  const [startLoading, setStartLoading] = useState(false)
  const [showImportModal, setShowImportModal] = useState(false)

  const [plan, setPlan] = useState<LessonPlan | null>(null)
  const [messages, setMessages] = useState<ConversationMessage[]>([])
  const [isThinking, setIsThinking] = useState(false)
  const [streaming, setStreaming] = useState<StreamingState | null>(null)
  const [planContent, setPlanContent] = useState('')
  const [fullGenerating, setFullGenerating] = useState(false)
  const [contentToast, setContentToast] = useState<string | null>(null)

  // #2-乙：被后续定稿取代的初稿教案默认折叠；记录被老师手动展开的初稿 id
  const [expandedDraftIds, setExpandedDraftIds] = useState<Set<string>>(new Set())

  // 子轮二：思考指示器旁的安抚/重试文案（软提示 8s 触发、retry_notice 到达时设置；空串=不显示）
  const [slowHintText, setSlowHintText] = useState('')

  const [dynamicChips, setDynamicChips] = useState<import('./conversationScript').ChipDef[]>([])

  const [selectedComponentIds, setSelectedComponentIds] = useState<Set<string>>(new Set())
  const [showComponentsModal, setShowComponentsModal] = useState(false)

  const [showTextbookModal, setShowTextbookModal] = useState(false)
  const [attachedTextbookIds, setAttachedTextbookIds] = useState<string[]>([])

  const [stageItems, setStageItems] = useState<StageProgressItem[]>([])
  const [currentStage, setCurrentStage] = useState<string>('')
  // 助手轻量选择入口 Phase 1：当前学科的助手偏好(三态) + 切换面板开关
  const [assistantPref, setAssistantPref] = useState<AssistantPref | null>(null)
  const [showAssistantPanel, setShowAssistantPanel] = useState(false)
  // 助手轻量选择入口·可见性补丁：本轮 message_done 回传的自动匹配助手名(空=纯骨架/未匹配)。
  // 仅用于「老师没显式选(found=false)」时在顶栏显示真实匹配名;不写偏好、不冻结、随阶段刷新。
  const [liveAssistantLabel, setLiveAssistantLabel] = useState('')
  const [isStageMode, setIsStageMode] = useState(false)

  const [isNarrow, setIsNarrow] = useState(() => window.innerWidth < 1024)
  const [narrowTab, setNarrowTab] = useState<'chat' | 'canvas'>('chat')

  const inputRef = useRef<import('./ConversationInputBar').ConversationInputBarHandle>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  // B-P1-22: 可滚动消息容器ref——滚动前据此判断用户是否贴近底部，决定是否自动跟随
  const messagesScrollRef = useRef<HTMLDivElement>(null)

  // 助手轻量选择入口 Phase 1：教案就绪且学科已知时，加载老师×学科助手偏好(三态)。
  // 用 plan.subject 而非组件 subject state 作 key —— 偏好是「老师×学科」维度，必须对齐教案实际学科。
  useEffect(() => {
    const subj = plan?.subject
    // 可见性补丁·修复跨教案残留：学科一变(含新建/切换教案)立即清空上一课的自动匹配名,
    // 顶栏回退"自动匹配",待新课第一条消息回来再填真名(避免显示上一课助手名的串台)。
    setLiveAssistantLabel('')
    if (!subj) { setAssistantPref(null); return }
    let cancelled = false
    getAssistantPref(subj)
      .then((p) => { if (!cancelled) setAssistantPref(p) })
      .catch(() => { if (!cancelled) setAssistantPref(null) })
    return () => { cancelled = true }
  }, [plan?.subject])

  const lastSentComponentIdsRef = useRef<string[]>([])

  // 子轮二：当前轮次序号 ref（每发起一轮自增），用于带 client_turn_id + SSE 事件过滤
  const currentTurnRef = useRef<string>('')
  const turnSeqRef = useRef(0)
  /** 生成并切换到新一轮 turnID（发起每轮 chat 前调用） */
  const nextTurn = useCallback((): string => {
    turnSeqRef.current += 1
    const t = `t${turnSeqRef.current}_${Date.now()}`
    currentTurnRef.current = t
    return t
  }, [])

  useEffect(() => {
    const onResize = () => setIsNarrow(window.innerWidth < 1024)
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])

  useEffect(() => {
    // B-P1-22: 自由滚动——仅当用户当前贴近底部(容差120px)时才自动跟随到最新；
    // 用户主动往上翻看历史时不再被流式输出强拽回底部。
    const el = messagesScrollRef.current
    const nearBottom = !el || (el.scrollHeight - el.scrollTop - el.clientHeight < 120)
    if (nearBottom) messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isThinking, streaming?.content, slowHintText])

  const showToast = (msg: string) => {
    setContentToast(msg)
    setTimeout(() => setContentToast(null), 4000)
  }

  const refreshStages = useCallback(async (planId: string) => {
    try {
      const resp = await getStageStatus(planId)
      setStageItems(resp.stages)
      setCurrentStage(resp.current_stage)
      setIsStageMode(true)
    } catch {
      setIsStageMode(false)
    }
  }, [])

  const isBusy = isThinking || !!streaming || fullGenerating

  // 重试 harness Hook（doSilentResend 用 ref 桥接前向引用）
  const doSilentResendRef = useRef<(text: string) => void>(() => {})
  const retry = useRetryLastMessage({
    messages,
    isBusy,
    doSilentResend: (text: string) => doSilentResendRef.current(text),
    stageSepPrefix: STAGE_SEP_PREFIX,
  })

  // SSE 连接管理（含 turnID 过滤 + 三层超时）
  const { sseState, connectSSE, closeSSE, manualReconnect, startTurnTimers, clearTurnTimers } = useConversationSSE({
    token,
    setIsThinking, setStreaming, setFullGenerating,
    setMessages, setPlanContent, setDynamicChips,
    showToast, refreshStages,
    onSoftFailure: () => { retry.recordFailure() },
    onNormalReply: () => { retry.recordSuccess(); setSlowHintText('') },
    // 可见性补丁：每轮 message_done 把后端实际匹配的助手名回写,空字符串=纯骨架(顶栏回退"自动匹配")
    onAssistantLabel: (label: string) => setLiveAssistantLabel(label),
    currentTurnRef,
    // 第一层·软提示（8s 仍无首 chunk）
    onSlowHint: () => { setSlowHintText('AI 正在认真思考，请稍候…') },
    // 第二层·重试可见性（后端 retry_notice 到达）
    onRetryNotice: (content: string) => { setSlowHintText(content || '刚才没接上话，正在帮你重试，请稍候…') },
    // 第三层·看门狗（90s 无任何本轮事件 → 判定挂起兜底）
    onWatchdogTimeout: () => {
      // 复位生成态、清空软提示
      setIsThinking(false)
      setStreaming(null)
      setFullGenerating(false)
      setSlowHintText('')
      // 作废本轮：推进 turnID，使本轮后续任何迟到回复都被 SSE 层过滤丢弃（B2 核心）
      currentTurnRef.current = `void_${Date.now()}`
      // 插一条人话，并让"重新回答"可用（canRetry 在 isBusy 复位后即为真）
      setMessages(prev => [...prev, {
        id: `watchdog_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
        content: '这次等得有点久，可能是没接上。你之前的内容都还在、不会丢——点下面的「重新回答」再试一次，或把刚才那句重发一遍就好。',
        created_at: new Date().toISOString(),
      }])
      retry.recordFailure()
    },
  })

  // 升级引导插入（连续失败 2 次）
  const escalateInsertedRef = useRef(false)
  useEffect(() => {
    if (retry.shouldEscalate && !escalateInsertedRef.current) {
      escalateInsertedRef.current = true
      setMessages(prev => [...prev, {
        id: `escalate_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
        content: '试了两次还是没接上。你可以换个说法、把需求说得更具体一点，或点右上角切到「专家模式」分步来，可能更顺。之前的内容都还在，不会丢。',
        created_at: new Date().toISOString(),
      }])
    }
    if (!retry.shouldEscalate && escalateInsertedRef.current) {
      escalateInsertedRef.current = false
    }
  }, [retry.shouldEscalate])

  // 恢复已有教案
  useEffect(() => {
    if (!sessionPlanId || phase !== 'resuming') return
    const resume = async () => {
      try {
        const [planData, convData] = await Promise.all([getLessonPlan(sessionPlanId), getConversation(sessionPlanId)])
        setPlan(planData)
        setMessages((convData.messages || []).filter(m => m.role === 'user' || m.role === 'assistant' || m.role === 'system'))
        if (planData.content_markdown) setPlanContent(planData.content_markdown)
        if (planData.current_stage && planData.stage_config) await refreshStages(sessionPlanId)
        setPhase('chatting')
        connectSSE(sessionPlanId)
        recordPlanMode(sessionPlanId, 'conversation')
        try {
          const targetStage = sessionStorage.getItem('workshop_target_stage')
          if (targetStage) {
            sessionStorage.removeItem('workshop_target_stage')
            if (planData.current_stage && planData.stage_config && planData.current_stage !== targetStage) {
              await switchToStage(sessionPlanId, targetStage)
              await refreshStages(sessionPlanId)
            }
          }
        } catch (e) { console.error('恢复后切换目标阶段失败:', e) }
      } catch (e) {
        console.error('恢复教案失败:', e)
        sessionStorage.removeItem('workshop_active_plan_id')
        setPhase('start')
      }
    }
    resume()
  }, [sessionPlanId, phase, connectSSE, refreshStages])

  // 开始备课
  const handleStart = async () => {
    if (!topic.trim() || startLoading) return
    setStartLoading(true)
    try {
      const resp = await startConversation({ subject, grade, topic: topic.trim(), duration_minutes: 45, recipe_id: recipeId || undefined })
      setPlan(resp.plan)
      setMessages([resp.opening_message])
      setPhase('chatting')
      sessionStorage.setItem('workshop_active_plan_id', resp.plan.id)
      recordPlanMode(resp.plan.id, 'conversation')
      // 大单元挂载（起步选）：若首屏选了单元方案，建会话后立即落库归属。
      // 独立 try-catch 吞错——挂载失败绝不阻断主流程（教案已建好），失败仅提示一句，
      // 老师可继续普通备课。落库成功后第一条用户消息起，后端注入层自动注入该单元方案上下文。
      if (unitPlanId) {
        try {
          await updatePlanUnitPlan(resp.plan.id, unitPlanId)
          setMessages(prev => [...prev, {
            id: `unitplan_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
            content: '📐 已关联所属单元方案，接下来我会贴着这份单元的整体设计来和你一起备这节课。',
            created_at: new Date().toISOString(),
          }])
        } catch (mountErr) {
          console.error('单元方案挂载失败:', mountErr)
          setMessages(prev => [...prev, {
            id: `unitplan_err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
            content: '⚠️ 单元方案关联未成功，不影响正常备课。如需关联，可稍后重试。',
            created_at: new Date().toISOString(),
          }])
        }
      }
      setUnitPlanId('') // 临时选择已固化到教案，回到首屏应重置
      connectSSE(resp.plan.id)
      if (resp.plan.current_stage && resp.plan.stage_config) await refreshStages(resp.plan.id)
    } catch (err) {
      console.error('开始备课失败:', err)
      alert('开始备课失败，请稍后重试')
    } finally { setStartLoading(false) }
  }

  const handleImportSuccess = async (planId: string, openingMessage: ConversationMessage) => {
    setShowImportModal(false)
    try {
      const [planData, convData] = await Promise.all([getLessonPlan(planId), getConversation(planId)])
      setPlan(planData)
      const serverMsgs = (convData.messages || []).filter(
        (m: ConversationMessage) => m.role === 'user' || m.role === 'assistant' || m.role === 'system'
      )
      setMessages(serverMsgs.length > 0 ? serverMsgs : [openingMessage])
      if (planData.content_markdown) setPlanContent(planData.content_markdown)
      setPhase('chatting')
      sessionStorage.setItem('workshop_active_plan_id', planId)
      recordPlanMode(planId, 'conversation')
      connectSSE(planId)
      if (planData.current_stage && planData.stage_config) await refreshStages(planId)
    } catch (err) {
      console.error('导入后加载失败:', err)
      alert('导入成功但加载失败，请刷新页面重试')
    }
  }

  // ===== 芯片上下文能力实现 =====

  /**
   * 发送文本消息。子轮二：生成新 turnID 带进请求 + 启动本轮超时计时。
   */
  const sendText = async (text: string) => {
    if (!plan || !text.trim()) return
    const componentIds = Array.from(selectedComponentIds)
    lastSentComponentIdsRef.current = componentIds
    setMessages(prev => [...prev, {
      id: `local_${Date.now()}`, role: 'user' as const, type: 'text' as const,
      content: componentIds.length > 0 ? `${text}\n（已附 ${componentIds.length} 个参考组件）` : text,
      created_at: new Date().toISOString(),
    }])
    setSlowHintText('')
    setIsThinking(true)
    const turnId = nextTurn()
    startTurnTimers()
    try {
      await sendChatMessage(plan.id, {
        message: text,
        assistant_id: null,
        selected_components: componentIds.length > 0 ? componentIds : undefined,
        client_turn_id: turnId,
      })
      setSelectedComponentIds(new Set())
    } catch (err) {
      clearTurnTimers()
      setIsThinking(false)
      setSlowHintText('')
      console.error('发送失败:', err)
      setMessages(prev => [...prev, {
        id: `send_err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
        content: '⚠️ 消息发送失败，请检查网络后重试。', created_at: new Date().toISOString(),
      }])
    }
  }

  /**
   * 静默重发（重试按钮专用）：不插重复 user 气泡，复用上次组件，同样生成新 turnID + 启动计时。
   */
  const doSilentResend = async (text: string) => {
    if (!plan || !text.trim() || isBusy) return
    const componentIds = lastSentComponentIdsRef.current
    setSlowHintText('')
    setIsThinking(true)
    const turnId = nextTurn()
    startTurnTimers()
    try {
      await sendChatMessage(plan.id, {
        message: text,
        assistant_id: null,
        selected_components: componentIds.length > 0 ? componentIds : undefined,
        client_turn_id: turnId,
      })
    } catch (err) {
      clearTurnTimers()
      setIsThinking(false)
      setSlowHintText('')
      console.error('重试发送失败:', err)
      setMessages(prev => [...prev, {
        id: `retry_err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
        content: '⚠️ 重试发送失败，请检查网络后再试。', created_at: new Date().toISOString(),
      }])
    }
  }
  doSilentResendRef.current = doSilentResend

  /**
   * 一键生成指定阶段（v191-2 简化：去掉分级弹窗，任何 full_generate 都直接出稿）。
   *
   * 背景：v191 曾给一键生成加"充分度判断 + 分级 confirm 弹窗 + 扫描分支"，
   * 实测发现弹窗到处误伤——老师在"扫描后核对报告"里点「就这样生成」、或在 revise 点
   * 「AI帮我改一版」，都是明确且知情的主动出稿，却仍被弹窗拦截，甚至造成"扫描完又问要不要扫描"
   * 的死循环。结论：弹窗是错误方向。改为——任何 full_generate 直接出稿、不弹任何窗；
   * 幻觉风险提醒改由对话区底部一行常驻提示承载（不打断、覆盖全部出稿路径）。
   * "扫描后生成"的核对报告+两枚芯片流程仍保留（提示词侧改动B），点「就这样生成」即走本函数直接出稿。
   */
  const handleFullGenerate = async (stage: string) => {
    if (!plan || fullGenerating) return
    const meta = FULL_GEN_STAGE_META[stage]
    if (!meta) return
    try {
      if (currentStage !== stage) {
        await switchToStage(plan.id, stage)
        await refreshStages(plan.id)
      }
      setFullGenerating(true)
      setSlowHintText('')
      setIsThinking(true)
      lastSentComponentIdsRef.current = []
      setMessages(prev => [...prev, {
        id: `local_${Date.now()}`, role: 'user' as const, type: 'text' as const,
        content: meta.trigger, created_at: new Date().toISOString(),
      }])
      const turnId = nextTurn()
      startTurnTimers()
      await sendChatMessage(plan.id, { message: meta.trigger, assistant_id: null, full_generate: true, client_turn_id: turnId })
    } catch (err) {
      clearTurnTimers()
      setIsThinking(false); setFullGenerating(false); setSlowHintText('')
      const errMsg = err instanceof Error ? err.message : ''
      setMessages(prev => [...prev, {
        id: `fullgen_err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
        content: errMsg && errMsg !== '请求失败' ? `⚠️ 一键生成未能开始：${errMsg}` : '⚠️ 一键生成请求发送失败，请稍后重试。',
        created_at: new Date().toISOString(),
      }])
    }
  }

  /** 进入下一阶段（推进不是 chat 主轮次，不带 turnID、不启动超时——开场白是旁路） */
  const advanceNext = async () => {
    if (!plan) return
    try {
      const componentIds = Array.from(selectedComponentIds)
      await advanceStage(plan.id, undefined, componentIds.length > 0 ? componentIds : undefined, true)
      if (componentIds.length > 0) setSelectedComponentIds(new Set())
      await refreshStages(plan.id)
    } catch (err) {
      console.error('推进失败:', err)
      setMessages(prev => [...prev, {
        id: `adv_err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
        content: '⚠️ 推进下一步失败，请稍后重试，或直接告诉我你想做什么。', created_at: new Date().toISOString(),
      }])
    }
  }

  const handlePublish = async () => {
    if (!plan) return
    if (!planContent || planContent.trim().length === 0) {
      alert('教案正文还没有生成，先让AI写出正文再发布吧。\n（可以点「⚡一键写出完整正文」）')
      return
    }
    if (!window.confirm('确认发布这份教案吗？\n发布后可在「我的教案」中随时查看和继续完善。')) return
    try {
      await publishLessonPlanPersonal(plan.id)
      sessionStorage.removeItem('workshop_active_plan_id')
      navigate('/lesson-plans/my-plans')
    } catch (err) {
      console.error('发布失败:', err)
      const errMsg = err instanceof Error ? err.message : ''
      alert(errMsg && errMsg !== '请求失败' ? `发布失败：${errMsg}` : '发布失败，请稍后重试')
    }
  }

  const openTool = (tool: string) => {
    if (tool === 'components') {
      if (isStageMode && STAGES_WITH_COMPONENTS.includes(currentStage)) {
        setShowComponentsModal(true)
      } else {
        setMessages(prev => [...prev, {
          id: `tool_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
          content: '🧩 当前环节没有可推荐的教学组件（修订定稿环节不注入组件）。', created_at: new Date().toISOString(),
        }])
      }
      return
    }
    if (tool === 'textbook') { setShowTextbookModal(true); return }
    if (tool === 'import') { setShowImportModal(true); return }
    setMessages(prev => [...prev, {
      id: `tool_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
      content: '🔧 这个能力即将在后续版本接入对话，敬请期待。', created_at: new Date().toISOString(),
    }])
  }

  const handleSelectComponent = (comp: ConvComponent) => {
    setSelectedComponentIds(prev => {
      const next = new Set(prev)
      if (next.has(comp.id)) { next.delete(comp.id) } else { next.add(comp.id) }
      return next
    })
  }

  const handleComponentsPicked = (ids: string[]) => {
    setSelectedComponentIds(prev => {
      const next = new Set(prev)
      ids.forEach(id => next.add(id))
      return next
    })
    setShowComponentsModal(false)
  }

  const handleTextbookAttached = (pageIds: string[]) => {
    setAttachedTextbookIds(pageIds)
    setShowTextbookModal(false)
    setMessages(prev => [...prev, {
      id: `tb_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
      content: pageIds.length > 0
        ? `📷 已关联 ${pageIds.length} 张课本页，从你的下一条消息开始，我会贴着课文原文来设计。`
        : '📷 已解除全部课本关联，之后的设计不再参考课本原文。',
      created_at: new Date().toISOString(),
    }])
  }

  const plusItemAvailability = (tool: string): { enabled: boolean; reason: string } => {
    if (tool === 'components') {
      if (!isStageMode) return { enabled: false, reason: '当前教案不是阶段模式' }
      if (!STAGES_WITH_COMPONENTS.includes(currentStage)) return { enabled: false, reason: '修订定稿环节不注入组件' }
      return { enabled: true, reason: '' }
    }
    if (tool === 'textbook') return { enabled: true, reason: '' }
    if (tool === 'import') return { enabled: true, reason: '' }
    return { enabled: false, reason: '即将上线' }
  }

  const chipCtx: ChipContext = {
    sendText,
    fullGenerate: handleFullGenerate,
    advanceNext,
    switchStage: async (code: string) => { if (plan) { await switchToStage(plan.id, code); await refreshStages(plan.id) } },
    publish: handlePublish,
    focusInput: () => inputRef.current?.prefill('我想修改：'),
    openTool,
  }

  const handleExit = () => {
    if (!plan) return
    if (!confirm('确定退出当前备课吗？\n\n教案已自动保存为草稿，你可以随时从「我的教案」继续。')) return
    closeSSE()
    sessionStorage.removeItem('workshop_active_plan_id')
    setPlan(null); setMessages([]); setPlanContent(''); setStageItems([]); setCurrentStage('')
    setIsStageMode(false); setIsThinking(false); setStreaming(null)
    setFullGenerating(false); setDynamicChips([]); setSlowHintText('')
    setLiveAssistantLabel('') // 可见性补丁:退出/返回入口时一并清空自动匹配名
    setUnitPlanId('') // 大单元挂载
    setRecipeId('') // 配方选择：回到首屏清空上一次的配方选择:回到首屏清空上一次的单元方案选择
    setSelectedComponentIds(new Set()); setShowComponentsModal(false)
    setShowTextbookModal(false); setAttachedTextbookIds([])
    retry.resetStreak(); escalateInsertedRef.current = false; lastSentComponentIdsRef.current = []
    currentTurnRef.current = ''
    setPhase('start')
  }

  const visibleChips = computeVisibleChips({
    phase, isBusy, isStageMode, messages, dynamicChips, currentStage, planContent,
  })

  const renderMessages = () => {
    const visible = messages.filter(m => !shouldHideHistoryMessage(m))
    // #2-乙：定位所有"完整教案"消息，最后一份=最终定稿默认展开，其之前的初稿默认折叠
    const planMsgIds = visible.filter(isFullLessonPlanMessage).map(m => m.id)
    const latestPlanId = planMsgIds.length > 0 ? planMsgIds[planMsgIds.length - 1] : null

    return visible.map(msg => {
      if ((msg.role as string) === 'system' && msg.content.startsWith(STAGE_SEP_PREFIX)) {
        return <div key={msg.id} style={{ height: '1px', background: C.border, margin: '14px 32px', opacity: 0.6 }} />
      }
      // #2-乙：被后续定稿取代的初稿教案 → 折叠成一行(点击展开)，避免整份教案在对话区出现两遍
      const isSupersededPlan = !!latestPlanId && isFullLessonPlanMessage(msg) && msg.id !== latestPlanId
      if (isSupersededPlan && !expandedDraftIds.has(msg.id)) {
        return (
          <div
            key={msg.id}
            onClick={() => setExpandedDraftIds(prev => { const n = new Set(prev); n.add(msg.id); return n })}
            style={{ margin: '8px 32px', padding: '9px 14px', border: `1px dashed ${C.border}`, borderRadius: '8px', background: '#F7F7F8', color: C.textMuted, fontSize: '13px', cursor: 'pointer' }}
          >
            📄 教案初稿（已被下方定稿取代）· 点击展开
          </div>
        )
      }
      return msg.role === 'assistant'
        ? <AIBubble key={msg.id} msg={msg} streaming={false} onSelectComponent={handleSelectComponent} selectedComponentIds={selectedComponentIds} />
        : <UserBubble key={msg.id} msg={msg} />
    })
  }

  if (phase === 'resuming') {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '60vh', gap: '16px' }}>
        <div style={{ width: '36px', height: '36px', border: `3px solid ${C.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
        <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
        <div style={{ fontSize: '15px', color: C.textSec }}>正在恢复备课进度…</div>
      </div>
    )
  }

  if (phase === 'start') {
    return (
      <>
        <ConversationStartScreen
          subject={subject} setSubject={setSubject}
          grade={grade} setGrade={setGrade}
          topic={topic} setTopic={setTopic}
          unitPlanId={unitPlanId} setUnitPlanId={setUnitPlanId}
          recipeId={recipeId} setRecipeId={setRecipeId}
          startLoading={startLoading}
          onStart={handleStart}
          onImport={() => setShowImportModal(true)}
          onSwitchMode={onSwitchMode}
        />
        {showImportModal && <ImportPlanModal onSuccess={handleImportSuccess} onCancel={() => setShowImportModal(false)} />}
      </>
    )
  }

  const placeholder = isBusy ? 'AI思考中…' : INPUT_PLACEHOLDERS[messages.length % INPUT_PLACEHOLDERS.length]
  const showChat = !isNarrow || narrowTab === 'chat'
  const showCanvas = !isNarrow || narrowTab === 'canvas'

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 120px)', overflow: 'hidden', margin: '-28px -32px' }}>

      {/* 助手轻量选择入口 Phase 1：相对定位容器，承载顶栏助手指示器 + 其下浮层面板 */}
      <div style={{ position: 'relative', flexShrink: 0 }}>
        <ConversationTopBar
          title={plan?.title || '备课对话'}
          textbookCount={attachedTextbookIds.length}
          sseState={sseState}
          isNarrow={isNarrow}
          narrowTab={narrowTab}
          onNarrowTabChange={setNarrowTab}
          onSwitchMode={onSwitchMode}
          onExit={handleExit}
          onReconnect={() => { if (plan) manualReconnect(plan.id) }}
          assistantLabel={
            plan?.subject
              ? (assistantPref
                  ? (assistantPref.is_system_default
                      ? '系统默认'
                      : (assistantPref.assistant_id
                          ? (assistantPref.assistant_name || '已选助手')
                          : (liveAssistantLabel || '自动匹配')))
                  : (liveAssistantLabel || '自动匹配'))
              : undefined
          }
          onAssistantClick={plan?.subject ? () => setShowAssistantPanel((v) => !v) : undefined}
        />
        {/* 切换面板：浮层，锚定在指示器下方（容器 position:relative 提供定位上下文） */}
        <div style={{ position: 'absolute', left: '18px', top: 0 }}>
          <AssistantSwitcher
            open={showAssistantPanel && !!plan?.subject}
            subject={plan?.subject || ''}
            stage={currentStage || undefined}
            pref={assistantPref}
            onClose={() => setShowAssistantPanel(false)}
            onChanged={(next) => setAssistantPref(next)}
          />
        </div>
      </div>

      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>

        {showChat && (
          <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden', borderRight: isNarrow ? 'none' : `1px solid ${C.border}` }}>
            <div ref={messagesScrollRef} style={{ flex: 1, overflowY: 'auto', padding: '18px 22px', display: 'flex', flexDirection: 'column' }}>
              {renderMessages()}
              {streaming && (
                <AIBubble key={streaming.id} msg={{ id: streaming.id, role: 'assistant', type: 'text', content: streaming.content, created_at: new Date().toISOString() }} streaming={true} onSelectComponent={handleSelectComponent} selectedComponentIds={selectedComponentIds} />
              )}
              {isThinking && !streaming && <ThinkingIndicator />}

              {/* 子轮二：思考/重试期间的安抚文案（软提示 8s 或 retry_notice 到达时显示） */}
              {slowHintText && (isThinking || !!streaming) && (
                <div style={{ margin: '2px 0 6px 42px', fontSize: '12px', color: C.textMuted, fontStyle: 'italic' }}>
                  {slowHintText}
                </div>
              )}

              {/* 重试按钮（独立于芯片协议，canRetry 为真时显示） */}
              {retry.canRetry && <RetryControls onRetry={retry.handleRetry} />}

              {/* 建议芯片行 */}
              <ConversationChipRow chips={visibleChips} onChipClick={chip => dispatchChip(chipCtx, chip)} />
              <div ref={messagesEndRef} />
              {/* v191-2 常驻幻觉提示：替代原一键生成弹窗，不打断、覆盖全部出稿路径 */}
              <div style={{ margin: '6px 0 2px 42px', fontSize: '12px', color: C.textMuted, opacity: 0.85, lineHeight: 1.5 }}>
                💡 AI 生成的内容可能存在不准确或与真实学情不符之处，请把握关键环节、核对后再使用。
              </div>
            </div>

            <ConversationInputBar
              ref={inputRef}
              isBusy={isBusy}
              placeholder={placeholder}
              selectedCount={selectedComponentIds.size}
              onClearSelected={() => setSelectedComponentIds(new Set())}
              onSend={sendText}
              hasContent={!!(planContent && planContent.trim().length > 0)}
              onPublish={handlePublish}
              plusItemAvailability={plusItemAvailability}
              onOpenTool={openTool}
            />
          </div>
        )}

        {showCanvas && (
          <div style={{ width: isNarrow ? '100%' : '440px', flexShrink: 0, overflow: 'hidden' }}>
            <ConversationCanvas
              content={planContent}
              busy={!!streaming && (currentStage === 'write' || currentStage === 'revise')}
              onFillMissing={(label) => sendText(`请帮我补充「${label}」部分的内容，直接更新到教案正文里。`)}
            />
          </div>
        )}
      </div>

      {showComponentsModal && plan && currentStage && (
        <StageComponentsModal
          planId={plan.id}
          stageCode={currentStage}
          stageName={STAGE_CODE_NAME[currentStage] || currentStage}
          mode="pick-only"
          onConfirm={handleComponentsPicked}
          onSkip={() => setShowComponentsModal(false)}
          onCancel={() => setShowComponentsModal(false)}
        />
      )}

      {showTextbookModal && plan && (
        <TextbookAttachModal
          planId={plan.id}
          subject={plan.subject}
          grade={plan.grade}
          currentPageIds={attachedTextbookIds}
          onSuccess={handleTextbookAttached}
          onCancel={() => setShowTextbookModal(false)}
        />
      )}

      {showImportModal && <ImportPlanModal onSuccess={handleImportSuccess} onCancel={() => setShowImportModal(false)} />}

      {contentToast && (
        <div style={{ position: 'fixed', bottom: '28px', left: '50%', transform: 'translateX(-50%)', maxWidth: '560px', padding: '12px 22px', borderRadius: '10px', background: 'linear-gradient(135deg, #10B981, #34D399)', color: '#fff', fontSize: '13px', fontWeight: 500, lineHeight: 1.6, boxShadow: '0 6px 24px rgba(16,185,129,0.4)', zIndex: 10000, textAlign: 'center' }}>
          {contentToast}
        </div>
      )}
    </div>
  )
}

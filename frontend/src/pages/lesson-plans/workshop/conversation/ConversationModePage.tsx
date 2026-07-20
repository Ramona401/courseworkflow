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
import { useEducationProfile } from '@/hooks/useEducationProfile'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import {
  startConversation, sendChatMessage, publishLessonPlanPersonal, updateLessonPlan,
  getLessonPlan, getConversation,
  getStageStatus, advanceStage, switchToStage,
  type LessonPlan, type ConversationMessage, type ConvComponent,
  type StageProgressItem, type RecipeSelectionMode,
} from '@/api/lesson-plans'
import { updatePlanUnitPlan } from '@/api/unit-plans'
import type { LessonPlanContentRestoreResponse } from '@/api/lesson-plan-versions'
import { updatePlanTextbooks } from '@/api/lesson-plan-textbooks'
import { updatePlanClassProfile } from '@/api/lesson-plan-class-profiles'
import { setLessonPlanCourseOutlinePublisher } from '@/api/course-outlines'
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
import RefMaterialAttachModal from './RefMaterialAttachModal'
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
import { getContextReceiptDisplayMessageIds } from '../components/context-receipt/contextReceiptVisibility'

interface ConversationModePageProps {
  onSwitchMode?: () => void
}

export default function ConversationModePage({ onSwitchMode }: ConversationModePageProps) {
  const { token, user } = useAuth()
  const {
    domain,
    organizationId,
    isK12,
    profile,
    ready: educationReady,
  } = useEducationProfile()
  const startDraftResourceId = `${domain}::${organizationId || 'no-org'}::new-plan`
  const navigate = useNavigate()

  const sessionPlanId = sessionStorage.getItem('workshop_active_plan_id')
  const [phase, setPhase] = useState<'start' | 'chatting' | 'resuming'>(sessionPlanId ? 'resuming' : 'start')

  /**
   * 开始备课首屏草稿。
   *
   * 全部字段按“当前用户 + 新教案首屏 + 字段”隔离，
   * 页面刷新或切走后返回时会恢复。
   *
   * 课题是主要自由文本输入，额外把handleKeyDown传给首屏，
   * 以支持跨刷新历史的Ctrl/Command+Z与重做。
   */
  const subjectDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'lesson-plan-conversation-start',
    resourceId: startDraftResourceId,
    field: 'subject',
    initialValue: SUBJECTS[0],
  })
  const gradeDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'lesson-plan-conversation-start',
    resourceId: startDraftResourceId,
    field: 'grade',
    initialValue: GRADES[0],
  })
  const topicDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'lesson-plan-conversation-start',
    resourceId: startDraftResourceId,
    field: 'topic',
    initialValue: '',
    maxHistory: 40,
  })
  const durationDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'lesson-plan-conversation-start',
    resourceId: startDraftResourceId,
    field: 'duration',
    initialValue: '45',
  })
  const unitPlanDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'lesson-plan-conversation-start',
    resourceId: startDraftResourceId,
    field: 'unit-plan',
    initialValue: '',
  })
  const classProfileDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'lesson-plan-conversation-start',
    resourceId: startDraftResourceId,
    field: 'class-profile',
    initialValue: '',
  })

  /**
   * 课程大纲版本需要区分：
   * null = 不关联；
   * '' = 通用版；
   * 具名字符串 = 指定教材版本。
   *
   * sessionStorage只能保存字符串，因此用独立哨兵编码null。
   */
  const coursePublisherNoneDraft =
    '__TEDNA_COURSE_PUBLISHER_NONE__'

  const coursePublisherDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'lesson-plan-conversation-start',
    resourceId: startDraftResourceId,
    field: 'course-publisher',
    initialValue: coursePublisherNoneDraft,
  })
  const recipeModeDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'lesson-plan-conversation-start',
    resourceId: startDraftResourceId,
    field: 'recipe-mode',
    initialValue: 'auto',
  })
  const recipeIDDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'lesson-plan-conversation-start',
    resourceId: startDraftResourceId,
    field: 'recipe-id',
    initialValue: '',
  })

  const subject =
    subjectDraft.value || SUBJECTS[0]
  const setSubject = (value: string) =>
    subjectDraft.setValue(value)

  const grade =
    gradeDraft.value || GRADES[0]
  const setGrade = (value: string) =>
    gradeDraft.setValue(value)

  const topic = topicDraft.value
  const setTopic = (value: string) =>
    topicDraft.setValue(value)

  const parsedDuration = Number.parseInt(
    durationDraft.value,
    10,
  )
  const duration = [40, 45, 50, 60].includes(
    parsedDuration,
  )
    ? parsedDuration
    : 45
  const setDuration = (value: number) =>
    durationDraft.setValue(String(value))

  const unitPlanId = unitPlanDraft.value
  const setUnitPlanId = (value: string) =>
    unitPlanDraft.setValue(value)

  const classProfileId =
    classProfileDraft.value
  const setClassProfileId = (value: string) =>
    classProfileDraft.setValue(value)

  const coursePublisher =
    coursePublisherDraft.value ===
    coursePublisherNoneDraft
      ? null
      : coursePublisherDraft.value

  const setCoursePublisher = (
    value: string | null,
  ) =>
    coursePublisherDraft.setValue(
      value === null
        ? coursePublisherNoneDraft
        : value,
    )

  const recipeModeValue =
    recipeModeDraft.value

  const recipeMode: RecipeSelectionMode =
    recipeModeValue === 'selected' ||
    recipeModeValue === 'none'
      ? recipeModeValue
      : 'auto'

  const setRecipeMode = (
    value: RecipeSelectionMode,
  ) =>
    recipeModeDraft.setValue(value)

  const recipeId = recipeIDDraft.value
  const setRecipeId = (value: string) =>
    recipeIDDraft.setValue(value)

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

  /**
   * 当前教育域切换为非K12时，立即关闭可能残留的课本弹窗。
   *
   * 该处理只负责前端体验；后端仍会独立执行K12硬闸，
   * 因此缓存状态和手工调用都不能绕过权限。
   */
  useEffect(() => {
    if (isK12) return

    setShowTextbookModal(false)
    setAttachedTextbookIds([])
  }, [isK12])

  // 参考资料附件（会话级，不落库）：注入文本 + 文件名。
  // refMaterial 用 ref 镜像给 sendChatMessage 闭包读，避免闭包捕获旧 state（同 lastSentComponentIdsRef 思路）。
  const [showRefModal, setShowRefModal] = useState(false)
  const [refMaterialName, setRefMaterialName] = useState('')
  const refMaterialRef = useRef<string>('')

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

  // 教案就绪或阶段变化时，按“学科+具体年级+当前阶段”加载有效助手偏好。
  //
  // 数据库存量偏好仍按老师×学科保存；后端会在每次读取时重新校验
  // 当前具体年级和阶段。不适用时返回has_record=false，并继续同年级自动匹配。
  useEffect(() => {
    const subj = plan?.subject
    const planGrade = plan?.grade

    // 学科、年级或阶段变化时立即清除上一条件下的自动匹配名称。
    setLiveAssistantLabel('')

    if (!subj || !planGrade) {
      setAssistantPref(null)
      return
    }

    let cancelled = false

    getAssistantPref(
      subj,
      planGrade,
      currentStage || undefined,
    )
      .then((nextPref) => {
        if (!cancelled) setAssistantPref(nextPref)
      })
      .catch(() => {
        if (!cancelled) setAssistantPref(null)
      })

    return () => {
      cancelled = true
    }
  }, [plan?.subject, plan?.grade, currentStage])


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

  /**
   * v233：正文变化后同步服务端正式教案状态。
   *
   * AI正文更新通过SSE只会直接更新planContent，数据库中的version已经递增，
   * 但页面plan仍可能保留进入页面时的旧版本号。
   *
   * 对planContent变化做300ms合并后重新读取教案，统一同步：
   *   - version
   *   - title
   *   - content_markdown
   *   - duration_minutes
   *   - status等其它正式字段
   *
   * 依赖只包含plan.id和planContent，setPlan不会再次触发本效果，
   * 因此不会形成循环请求。
   */
  useEffect(() => {
    if (!plan?.id || !planContent) return

    const planID = plan.id
    const timer = window.setTimeout(() => {
      void getLessonPlan(planID)
        .then(latestPlan => {
          setPlan(previous =>
            previous?.id === latestPlan.id
              ? latestPlan
              : previous
          )
        })
        .catch(error => {
          console.error(
            '正文更新后同步教案正式版本失败:',
            error,
          )
        })
    }, 300)

    return () => window.clearTimeout(timer)
  }, [plan?.id, planContent])

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
    if (
      !topic.trim() ||
      startLoading ||
      (recipeMode === 'selected' && !recipeId)
    ) return
    setStartLoading(true)
    try {
      const resp = await startConversation({
        subject,
        grade,
        topic: topic.trim(),
        duration_minutes: duration,
        recipe_mode: recipeMode,
        recipe_id:
          recipeMode === 'selected'
            ? recipeId || undefined
            : undefined,
      })
      setPlan(resp.plan)
      setMessages([resp.opening_message])
      setPhase('chatting')
      sessionStorage.setItem('workshop_active_plan_id', resp.plan.id)
      recordPlanMode(resp.plan.id, 'conversation')
      // 教案已经成功创建，课题草稿已被正式业务消费。
      // 清除课题当前值和历史，避免下一份教案误带上一次课题。
      topicDraft.clear()
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
      // 班级学情挂载（起步选）：若首屏选了班级卡，建会话后立即落库归属。
      // 独立 try-catch 吞错——挂载失败绝不阻断主流程（教案已建好），失败仅提示一句。
      // 落库成功后第一条用户消息起，后端注入层在 analyze/design/write 三阶段注入该班级学情。
      if (classProfileId) {
        try {
          await updatePlanClassProfile(resp.plan.id, classProfileId)
          setMessages(prev => [...prev, {
            id: `classprofile_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
            content: '👥 已关联本班学情，接下来我会针对这个班的分层结构与薄弱点，和你一起做差异化的教学设计。',
            created_at: new Date().toISOString(),
          }])
        } catch (cpErr) {
          console.error('班级学情挂载失败:', cpErr)
          setMessages(prev => [...prev, {
            id: `classprofile_err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
            content: '⚠️ 本班学情关联未成功，不影响正常备课。如需关联，可稍后重试。',
            created_at: new Date().toISOString(),
          }])
        }
      }
      /**
       * 创建完成后写入课程大纲挂载三态。
       *
       * K12：
       *   - 按老师在首屏选择的出版社版本挂载；
       *   - null表示不关联课程大纲。
       *
       * vocational/adult：
       *   - 首屏不展示出版社；
       *   - 自动写入空字符串，表示挂载同域普通课程大纲；
       *   - 后端按教案教育域快照、课程和学习层级匹配；
       *   - 不会查询或泄漏K12出版社数据。
       *
       * mixed或教育域未就绪时保持不挂载。
       */
      const isOrdinaryNonK12 =
        educationReady &&
        (
          profile.code === 'vocational' ||
          profile.code === 'adult'
        )

      const outlineMountPublisher =
        isOrdinaryNonK12
          ? ''
          : coursePublisher

      if (
        outlineMountPublisher !== null &&
        outlineMountPublisher !== undefined
      ) {
        try {
          await setLessonPlanCourseOutlinePublisher(
            resp.plan.id,
            outlineMountPublisher,
          )

          const successMessage =
            isOrdinaryNonK12
              ? '📚 已关联当前课程的教学依据，接下来我会参考同教育域的课程大纲和学习层级来和你一起备课。'
              : `📚 已关联「${
                  outlineMountPublisher === ''
                    ? '通用 / 不限版本'
                    : outlineMountPublisher
                }」课程大纲，接下来我会贴着这版教材的大纲来和你一起备课。`

          setMessages(previous => [
            ...previous,
            {
              id: `courseoutline_${Date.now()}`,
              role: 'assistant' as const,
              type: 'text' as const,
              content: successMessage,
              created_at:
                new Date().toISOString(),
            },
          ])
        } catch (outlineError) {
          console.error(
            '课程大纲关联失败:',
            outlineError,
          )

          setMessages(previous => [
            ...previous,
            {
              id: `courseoutline_err_${Date.now()}`,
              role: 'assistant' as const,
              type: 'text' as const,
              content:
                '⚠️ 课程大纲关联未成功，不影响正常备课。如需关联，可稍后重试。',
              created_at:
                new Date().toISOString(),
            },
          ])
        }
      }
      setUnitPlanId('') // 临时选择已固化到教案，回到首屏应重置
      setClassProfileId('') // 班级学情同款：临时选择已固化到教案，回到首屏应重置
      setCoursePublisher(null) // 教材版本同款：临时选择已固化到教案，回到首屏应重置
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
  /**
   * 发送文本消息。
   *
   * 返回boolean给ConversationInputBar：
   * true = 后端已经接受本轮任务，可以清空输入框；
   * false = 请求未被接受，输入框必须保留原文。
   *
   * 失败时同时撤回刚插入的本地用户气泡，
   * 防止老师再次发送时页面出现两条相同用户消息。
   */
  const sendText = async (
    text: string,
  ): Promise<boolean> => {
    if (!plan || !text.trim()) {
      return false
    }

    const componentIds =
      Array.from(selectedComponentIds)

    lastSentComponentIdsRef.current =
      componentIds

    const localMessageID =
      `local_${Date.now()}`

    setMessages(prev => [
      ...prev,
      {
        id: localMessageID,
        role: 'user' as const,
        type: 'text' as const,
        content:
          componentIds.length > 0
            ? `${text}\n（已附 ${componentIds.length} 个参考组件）`
            : text,
        created_at:
          new Date().toISOString(),
      },
    ])

    setSlowHintText('')
    setIsThinking(true)

    const turnId = nextTurn()
    startTurnTimers()

    try {
      await sendChatMessage(plan.id, {
        message: text,
        assistant_id: null,
        selected_components:
          componentIds.length > 0
            ? componentIds
            : undefined,
        client_turn_id: turnId,
        ref_material:
          refMaterialRef.current ||
          undefined,
      })

      setSelectedComponentIds(
        new Set(),
      )

      return true
    } catch (err) {
      clearTurnTimers()
      setIsThinking(false)
      setSlowHintText('')

      console.error('发送失败:', err)

      setMessages(prev => [
        ...prev.filter(
          message =>
            message.id !== localMessageID,
        ),
        {
          id: `send_err_${Date.now()}`,
          role: 'assistant' as const,
          type: 'text' as const,
          content:
            '⚠️ 消息发送失败，输入内容已经保留，请检查网络后重试。',
          created_at:
            new Date().toISOString(),
        },
      ])

      return false
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
        ref_material: refMaterialRef.current || undefined,
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
      await sendChatMessage(plan.id, { message: meta.trigger, assistant_id: null, full_generate: true, client_turn_id: turnId, ref_material: refMaterialRef.current || undefined })
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

  /**
   * 保存老师在右侧教案画布中直接编辑的完整正文。
   *
   * 保存走现有 PUT /lesson-plans/plans/{id}，不新增接口。
   * AI忙碌期间禁止保存，避免人工旧稿与SSE新稿互相覆盖。
   */
  const handleManualContentSave = async (nextContent: string) => {
    if (!plan) {
      throw new Error('当前教案尚未加载')
    }
    if (isBusy) {
      throw new Error('AI正在处理正文，请等待完成后再保存')
    }

    await updateLessonPlan(plan.id, {
      content_markdown: nextContent,
    })

    // 不再基于页面旧version执行+1。
    // AI可能已经在后台多次更新正文，页面缓存版本可能落后于数据库。
    // 保存成功后重新读取正式教案，确保按钮、版本弹窗和数据库完全一致。
    const latestPlan = await getLessonPlan(plan.id)
    setPlan(latestPlan)
    setPlanContent(
      latestPlan.content_markdown || nextContent,
    )

    showToast(
      `✅ 教案正文已保存，当前版本v${latestPlan.version}`,
    )
  }

  /**
   * 历史版本恢复成功后，同步画布、教案标题、课时时长和版本号。
   */
  const handleContentRestored = (
    result: LessonPlanContentRestoreResponse,
  ) => {
    setPlanContent(result.content_markdown)
    setPlan(previous => previous
      ? {
          ...previous,
          title: result.title,
          content_markdown: result.content_markdown,
          duration_minutes: result.duration_minutes,
          version: result.current_version,
        }
      : previous
    )
    showToast(
      `✅ 已恢复历史v${result.restored_from_version}，当前为v${result.current_version}`,
    )
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
    if (tool === 'textbook') {
      if (!isK12) {
        setMessages(prev => [...prev, {
          id: `tool_${Date.now()}`,
          role: 'assistant' as const,
          type: 'text' as const,
          content: '当前教育域暂无课本能力',
          created_at: new Date().toISOString(),
        }])
        return
      }

      setShowTextbookModal(true)
      return
    }
    if (tool === 'import') { setShowImportModal(true); return }
    if (tool === 'ref_material') { console.log('[REF] openTool ref_material 被调用, plan=', !!plan, 'showRefModal将设为true'); setShowRefModal(true); return }
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

  const handleTextbookAttached = async (pageIds: string[]) => {
    setAttachedTextbookIds(pageIds)
    setShowTextbookModal(false)
    // 课本中途挂载落库：把选中的课本页ID写入 lesson_plans.textbook_page_ids。
    // 后端注入层每轮对话重读该列，落库后下一条用户消息起自动注入课本OCR文字。
    // 独立 try-catch 吞错——落库失败仅提示，不阻断对话主流程。
    if (plan) {
      try {
        await updatePlanTextbooks(plan.id, pageIds)
      } catch (err) {
        console.error('课本关联落库失败:', err)
        setMessages(prev => [...prev, {
          id: `tb_err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
          content: '⚠️ 课本关联保存失败，AI可能无法读取课本内容。请重新打开课本面板再试一次。',
          created_at: new Date().toISOString(),
        }])
        return
      }
    }
    setMessages(prev => [...prev, {
      id: `tb_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
      content: pageIds.length > 0
        ? `📷 已关联 ${pageIds.length} 张课本页，从你的下一条消息开始，我会贴着课文原文来设计。`
        : '📷 已解除全部课本关联，之后的设计不再参考课本原文。',
      created_at: new Date().toISOString(),
    }])
  }

  const plusItemAvailability = (
    tool: string,
  ): {
    visible?: boolean
    enabled: boolean
    reason: string
  } => {
    if (tool === 'components') {
      if (!isStageMode) return { enabled: false, reason: '当前教案不是阶段模式' }
      if (!STAGES_WITH_COMPONENTS.includes(currentStage)) return { enabled: false, reason: '修订定稿环节不注入组件' }
      return { enabled: true, reason: '' }
    }
    if (tool === 'textbook') {
      return isK12
        ? {
            visible: true,
            enabled: true,
            reason: '',
          }
        : {
            // 非K12不显示课本入口，而不是只显示禁用按钮。
            visible: false,
            enabled: false,
            reason: '当前教育域暂无课本能力',
          }
    }
    if (tool === 'import') return { enabled: true, reason: '' }
    if (tool === 'ref_material') return { enabled: true, reason: '' }
    return { enabled: false, reason: '即将上线' }
  }

  const chipCtx: ChipContext = {
    /**
     * ChipContext历史协议要求sendText返回void。
     *
     * 页面主sendText保留Promise<boolean>：
     * true表示后端已接受任务，输入框可以清空；
     * false表示发送失败，输入草稿必须保留。
     *
     * 芯片动作不消费这个布尔值，因此在此处包成
     * Promise<void>，避免修改全局ChipContext协议。
     */
    sendText: async (text: string) => {
      await sendText(text)
    },
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
    setClassProfileId('') // 班级学情挂载：回到首屏清空上一次的班级学情选择
    setCoursePublisher(null) // 教材版本：回到首屏清空上一次的版本选择
    setRecipeId('') // 配方选择：回到首屏清空上一次的配方选择:回到首屏清空上一次的单元方案选择
    setRecipeMode('auto') // 配方模式：回到首屏恢复智能选择
    setSelectedComponentIds(new Set()); setShowComponentsModal(false)
    setShowTextbookModal(false); setAttachedTextbookIds([])
    setShowRefModal(false); setRefMaterialName(''); refMaterialRef.current = '' // 参考资料附件：退出即清空
    retry.resetStreak(); escalateInsertedRef.current = false; lastSentComponentIdsRef.current = []
    currentTurnRef.current = ''
    setPhase('start')
  }

  const visibleChips = computeVisibleChips({
    phase, isBusy, isStageMode, messages, dynamicChips, currentStage, planContent,
  })


  const renderMessages = () => {
    const visible = messages.filter(
      message =>
        !shouldHideHistoryMessage(message),
    )

    // 定位所有完整教案消息。
    // 最后一份为最终定稿，其前面的初稿默认折叠。
    const planMessageIDs = visible
      .filter(isFullLessonPlanMessage)
      .map(message => message.id)

    const latestPlanID =
      planMessageIDs.length > 0
        ? planMessageIDs[
            planMessageIDs.length - 1
          ]
        : null

    // 只有实际以AIBubble渲染的消息才参与回执去重。
    // 被折叠的旧教案初稿不应抢占后续回执的首次展示机会。
    const receiptCandidates = visible.filter(
      message => {
        const isSupersededPlan =
          Boolean(latestPlanID) &&
          isFullLessonPlanMessage(message) &&
          message.id !== latestPlanID

        return !(
          isSupersededPlan &&
          !expandedDraftIds.has(message.id)
        )
      },
    )

    const visibleReceiptMessageIDs =
      getContextReceiptDisplayMessageIds(
        receiptCandidates,
      )

    return visible.map(message => {
      if (
        (
          message.role as string
        ) === 'system' &&
        message.content.startsWith(
          STAGE_SEP_PREFIX,
        )
      ) {
        return (
          <div
            key={message.id}
            style={{
              height: '1px',
              background: C.border,
              margin: '14px 32px',
              opacity: 0.6,
            }}
          />
        )
      }

      const isSupersededPlan =
        Boolean(latestPlanID) &&
        isFullLessonPlanMessage(message) &&
        message.id !== latestPlanID

      if (
        isSupersededPlan &&
        !expandedDraftIds.has(message.id)
      ) {
        return (
          <div
            key={message.id}
            onClick={() =>
              setExpandedDraftIds(previous => {
                const next = new Set(previous)
                next.add(message.id)
                return next
              })
            }
            style={{
              margin: '8px 32px',
              padding: '9px 14px',
              border: `1px dashed ${C.border}`,
              borderRadius: '8px',
              background: '#F7F7F8',
              color: C.textMuted,
              fontSize: '13px',
              cursor: 'pointer',
            }}
          >
            📄 教案初稿（已被下方定稿取代）· 点击展开
          </div>
        )
      }

      return message.role === 'assistant'
        ? (
            <AIBubble
              key={message.id}
              msg={message}
              streaming={false}
              onSelectComponent={
                handleSelectComponent
              }
              selectedComponentIds={
                selectedComponentIds
              }
              showContextReceipt={
                visibleReceiptMessageIDs.has(
                  message.id,
                )
              }
            />
          )
        : (
            <UserBubble
              key={message.id}
              msg={message}
            />
          )
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
          onTopicDraftKeyDown={topicDraft.handleKeyDown}
          duration={duration} setDuration={setDuration}
          unitPlanId={unitPlanId} setUnitPlanId={setUnitPlanId}
          classProfileId={classProfileId} setClassProfileId={setClassProfileId}
          coursePublisher={coursePublisher} setCoursePublisher={setCoursePublisher}
          recipeMode={recipeMode} setRecipeMode={setRecipeMode}
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
            grade={plan?.grade || ''}
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
              refMaterialName={refMaterialName}
              onClearRefMaterial={() => { refMaterialRef.current = ''; setRefMaterialName('') }}
            />
          </div>
        )}

        {showCanvas && (
          <div style={{ width: isNarrow ? '100%' : '440px', flexShrink: 0, overflow: 'hidden' }}>
            <ConversationCanvas
              planID={plan?.id || ''}
              content={planContent}
              currentVersion={plan?.version || 1}
              busy={isBusy}
              canEdit={Boolean(
                plan &&
                [
                  'draft',
                  'published_personal',
                  'revision',
                  'approved',
                  'published_shared',
                ].includes(plan.status)
              )}
              onSaveContent={handleManualContentSave}
              onContentRestored={handleContentRestored}
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

      {showTextbookModal && plan && isK12 && (
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
      {showRefModal && plan && (
        <RefMaterialAttachModal
          subject={plan.subject}
          grade={plan.grade}
          onAttached={({ text, fileName }) => {
            refMaterialRef.current = text
            setRefMaterialName(fileName)
            setShowRefModal(false)
            setMessages(prev => [...prev, {
              id: `ref_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
              content: `📎 已附参考资料「${fileName}」，从你的下一条消息开始，我会参考其中的知识点和要求。`,
              created_at: new Date().toISOString(),
            }])
          }}
          onCancel={() => setShowRefModal(false)}
        />
      )}

      {contentToast && (
        <div style={{ position: 'fixed', bottom: '28px', left: '50%', transform: 'translateX(-50%)', maxWidth: '560px', padding: '12px 22px', borderRadius: '10px', background: 'linear-gradient(135deg, #10B981, #34D399)', color: '#fff', fontSize: '13px', fontWeight: 500, lineHeight: 1.6, boxShadow: '0 6px 24px rgba(16,185,129,0.4)', zIndex: 10000, textAlign: 'center' }}>
          {contentToast}
        </div>
      )}
    </div>
  )
}

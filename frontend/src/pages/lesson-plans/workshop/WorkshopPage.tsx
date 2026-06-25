/**
 * WorkshopPage — 备课工坊主页面
 *
 * 迭代8 重大重构(P1+P2+P3):
 *   P1:阶段隔离 + 用户手动完成触发
 *   P2:阶段过渡弹窗 + 结构化产出展示(方案B)
 *   P3:叙事式过渡动画
 * 迭代12 新增:
 *   阶段过渡时弹出组件推荐弹窗(方案B组件交互)
 * v88 新增(P2-3 断线恢复与SSE韧性):
 *   - 网络状态指示器(绿/黄/红)
 *   - SSE自动重连(指数退避,最多5次)
 *   - 重连后自动拉取最新对话补齐丢失消息
 *   - 消息发送失败自动重试1次
 * v108 新增:
 *   - 首屏双入口:新建备课 / 导入已有教案(并列卡片,点击左卡展开表单,点击右卡弹导入弹窗)
 * v112 (TE-DNA 3.0 P0 STEP 8)新增:
 *   - 阶段头嵌入 AssistantSelector(紧凑模式,compact=true)
 *   - 按当前 currentStage 动态映射到 6 种 workshop_* 场景
 *   - assistantId state 随切换阶段自动重置为 null(让新阶段按 scene 重新匹配默认助手)
 *   - handleSend 发送消息时透传 assistant_id 到后端
 *   - 对话进行中(isBusy=true)禁用选择器防止上下文切换错乱
 *   - 非阶段模式(isStageMode=false)不显示选择器(仅历史遗留教案)
 *   - 快速出稿/极速模式天然覆盖:只要有阶段头就有选择器,无需特殊判断
 * v113 (P0 STEP 6)新增:
 *   - 引入 AssistantEditModal 组件,给 AssistantSelector 挂 onEdit + onCreateNew 回调
 *   - 点击 "+ 新建个人助手" → 打开 Modal(create-personal 模式)
 *     defaultScene 随当前阶段动态变化,新助手的场景勾选与所在阶段一致
 *   - 点击个人助手的 "✏️ 改" → 打开 Modal(edit 模式,预填原内容)
 *   - 保存成功后 setAssistantId(新ID) + 变更 selectorKey 强制 Selector 刷新列表
 *   - Modal 的显示与否独立于 Selector,互不干扰
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
import { StageSummaryModal } from './components/StageSummaryModal'
import { StageTransitionView } from './components/StageTransitionView'
import { StageSeparatorBubble } from './components/StageSeparatorBubble'
import StageComponentsModal from './components/StageComponentsModal'
import { ResumingView, StartScreen } from './components/WorkshopStartScreen'
import { getAssessmentResult } from '@/api/assessment'
// 迭代7B：备课中展示本次关联的课本图
import { getTextbook, type TextbookDetail } from '@/api/textbooks'
import ImportPlanModal from './components/ImportPlanModal'
// v112 (P0 STEP 8):引入 AI 助手选择器和场景类型
import AssistantSelector from '@/components/ai-assistants/AssistantSelector'
import type { AssistantScene } from '@/api/ai-assistants'
// v121 Bug 2 修复:个人助手删除操作
import { deleteAssistant } from '@/api/ai-assistants'
// v113 (P0 STEP 6):引入 AI 助手编辑弹窗
import AssistantEditModal, { type AssistantEditMode } from '@/components/ai-assistants/AssistantEditModal'

const STAGE_SEP_PREFIX = '__STAGE_SEP__'

// 迭代12:有组件映射的阶段列表(revise无组件)
const STAGES_WITH_COMPONENTS = ['analyze', 'design', 'write', 'review']

// v88:消息发送最大重试次数
const SEND_RETRY_MAX = 1

/**
 * v112 (P0 STEP 8):阶段码 → AI 助手场景码映射
 *
 * 工坊 5 个系统阶段对应 5 种助手场景,每个场景可独立匹配默认助手:
 *   - analyze(教学分析) → workshop_analyze
 *   - design (教学设计) → workshop_design
 *   - write  (教案撰写) → workshop_write
 *   - review (AI评审) → workshop_review
 *   - revise (修订定稿) → workshop_revise
 *
 * 自定义阶段或无法映射时回退到 workshop_write(最通用的撰写场景)
 */
const STAGE_CODE_TO_SCENE: Record<string, AssistantScene> = {
  analyze: 'workshop_analyze',
  design:  'workshop_design',
  write:   'workshop_write',
  review:  'workshop_review',
  revise:  'workshop_revise',
}

export default function WorkshopPage() {
  const { token } = useAuth()
  const navigate  = useNavigate()
  const location  = useLocation()

  const resumePlanId  = (location.state as { resumePlanId?: string } | null)?.resumePlanId
  const sessionPlanId = sessionStorage.getItem('workshop_active_plan_id')
  const effectivePlanId = resumePlanId || sessionPlanId || null

  const [phase, setPhase]               = useState<'start' | 'chatting' | 'resuming'>(effectivePlanId ? 'resuming' : 'start')
  const [startLoading, setStartLoading] = useState(false)
  const [resumeError, setResumeError]   = useState<string | null>(null)

  // v108:首屏模式:选择卡片 | 展开新建表单
  const [startMode, setStartMode] = useState<'choose' | 'new'>('choose')

  const [plan, setPlan] = useState<LessonPlan | null>(null)
  // 迭代7B：本次备课关联的课本图详情（进入备课后按 plan.textbook_page_ids 查出，仅用于展示）
  const [linkedTextbooks, setLinkedTextbooks] = useState<TextbookDetail[]>([])
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
  const [showImportModal, setShowImportModal] = useState(false)  // v108:导入弹窗
  const [isStageProcessing, setIsStageProcessing] = useState(false)

  // P1:AI建议完成提示
  const [aiSuggestsComplete, setAiSuggestsComplete] = useState(false)

  // P2:弹窗状态(方案B新增 stageCode + structuredOutput)
  const [showSummaryModal, setShowSummaryModal] = useState(false)
  const [summaryLoading, setSummaryLoading]     = useState(false)
  const [stageSummary, setStageSummary]         = useState('')
  const [stageStructured, setStageStructured]   = useState('{}')

  // P3:过渡动画
  const [isTransitioning, setIsTransitioning]   = useState(false)
  const [transitionStep, setTransitionStep]     = useState(0)
  const [transitionInfo, setTransitionInfo]     = useState<{
    currentName: string; nextName: string; nextRole: string
  } | null>(null)

  // P0-2:阶段完成度状态
  const [stageCompleteness, setStageCompleteness] = useState<StageCompletenessResponse | null>(null)

  // 迭代12:阶段组件推荐弹窗状态
  const [showComponentsModal, setShowComponentsModal] = useState(false)
  const [pendingTransitionStage, setPendingTransitionStage] = useState<string | null>(null)

  // v121 任务B:"随时选组件"弹窗(独立于阶段过渡)
  // 老师在阶段进行中点阶段头的"📚 选组件"按钮触发,打开后选中的组件会追加到 selectedComponentIds
  // 这和阶段过渡时的 showComponentsModal 共用 StageComponentsModal 组件,但走独立的回调
  const [showPickComponentsModal, setShowPickComponentsModal] = useState(false)

  // v77:阶段视图切换状态(null=显示当前阶段,指定stageCode=查看该阶段历史对话)
  const [viewingStage, setViewingStage] = useState<string | null>(null)

  // v88新增:SSE连接状态(connected=绿色 | reconnecting=黄色 | disconnected=红色)
  const [sseConnectionState, setSseConnectionState] = useState<SSEConnectionState>('connected')

  // v105 P1-7:退回修改模式提示条(用户手动关闭后不再显示)
  const [revisionBannerDismissed, setRevisionBannerDismissed] = useState(false)

  // v112 (P0 STEP 8):当前选中的 AI 助手 ID(null 表示不使用助手走兜底)
  // 切换阶段时会自动重置为 null,让新阶段按 scene 重新匹配默认助手
  const [assistantId, setAssistantId] = useState<string | null>(null)

  // v113 (P0 STEP 6):AssistantEditModal 状态
  // modalOpen=false 时 Modal 不渲染;modalMode 和 modalEditId 由触发点决定
  const [modalOpen, setModalOpen]     = useState(false)
  const [modalMode, setModalMode]     = useState<AssistantEditMode>('create-personal')
  const [modalEditId, setModalEditId] = useState<string | undefined>(undefined)
  // 保存成功后强制 AssistantSelector 重新 mount 重拉列表
  const [selectorKey, setSelectorKey] = useState(0)

  // v124: 图片插入引导条 — localStorage 记忆关闭状态
  const [imageTipDismissed, setImageTipDismissed] = useState(() => {
    try { return localStorage.getItem('workshop_image_tip_dismissed') === '1' } catch { return false }
  })

  // v168: 正文生成成功提示 toast（3秒自动消失）
  // 老师疑问"到底生成没生成"的根因是缺少正向反馈——正文落库时主动弹一条提示，
  // 明确告知"已生成，可在右侧预览/我的教案查看"，消除疑虑。
  const [contentToast, setContentToast] = useState<string | null>(null)
  const showContentToast = (msg: string) => {
    setContentToast(msg)
    setTimeout(() => setContentToast(null), 4000)
  }

  // v168(功能B): 一键生成完整教案的进行中状态（防重复点击）
  // 仅在 write 阶段且正文为空时按钮可见，点击后置 true，直到 SSE 出稿落库（onContentUpdate）或报错（onError）复位
  const [fullGenerating, setFullGenerating] = useState(false)

  // v88:SSE连接引用改为SSEConnection类型(支持close方法)
  const sseRef         = useRef<SSEConnection | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  // B-P1-22: 可滚动消息容器ref——滚动前据此判断用户是否贴近底部，决定是否自动跟随
  const messagesScrollRef = useRef<HTMLDivElement>(null)
  // v88:保存planId的ref,供重连回调使用(避免闭包捕获旧值)
  const planIdRef = useRef<string | null>(null)

  // 迭代7B：进入备课（或恢复教案）后，按 plan.textbook_page_ids 查出关联课本图详情用于展示。
  // 解析 plan.textbook_page_ids（JSON数组字符串），批量 getTextbook 拿缩略图+章节名。
  // plan 为空或无关联时清空。查询失败静默忽略，不影响备课主流程。
  useEffect(() => {
    const ids = (() => {
      try {
        const raw = plan?.textbook_page_ids
        if (!raw || raw === '[]') return [] as string[]
        const arr = JSON.parse(raw)
        return Array.isArray(arr) ? (arr as string[]) : []
      } catch { return [] as string[] }
    })()
    if (ids.length === 0) { setLinkedTextbooks([]); return }
    let cancelled = false
    Promise.all(ids.map(id => getTextbook(id).catch(() => null)))
      .then(list => {
        if (cancelled) return
        setLinkedTextbooks(list.filter((t): t is TextbookDetail => t !== null))
      })
    return () => { cancelled = true }
  }, [plan])

  useEffect(() => {
    // B-P1-22: 自由滚动——仅当用户当前贴近底部(容差120px)时才自动跟随到最新；
    // 用户主动往上翻看历史时不再被流式输出强拽回底部。
    const el = messagesScrollRef.current
    const nearBottom = !el || (el.scrollHeight - el.scrollTop - el.clientHeight < 120)
    if (nearBottom) messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isThinking, streaming?.content])

  useEffect(() => { return () => { sseRef.current?.close() } }, [])

  useEffect(() => {
    if (phase !== 'start') return
    getAssessmentResult()
      .then(resp => { setNeedsAssessment(!resp.has_profile) })
      .catch(() => { setNeedsAssessment(false) })
  }, [phase])

  // v112 (P0 STEP 8):切换阶段时重置助手选择
  // 因为不同阶段对应不同的 scene,默认推荐助手也不同,
  // 保留上一阶段的助手会变成"幽灵选中态"(新 scene 列表里可能不包含此助手)。
  // 重置为 null 后 AssistantSelector 会按新 scene 自动匹配 is_default_here=true 的助手。
  useEffect(() => {
    setAssistantId(null)
  }, [currentStage])

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

  // v88重构:connectSSE增加连接状态回调和重连补齐逻辑
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
        // v172：message_done 是每轮 AI 必发的"本轮已说完话"信号。
        // 把"生成中"类状态的复位统一挂在这里（而非只挂在 content_update），
        // 这样 analyze/design 等不发 content_update 的阶段，AI 一回复完
        // 橙色"⏳生成中..."按钮也会立刻恢复，根治"出来了还显示生成中"。
        setIsThinking(false); setStreaming(null)
        setFullGenerating(false)  // v172：一键生成出稿完成，复位按钮，不再只依赖 content_update
        setMessages(prev => [...prev, msg])
        if (isStageModeRef.current) {
          setIsStageProcessing(true)
          setTimeout(() => setIsStageProcessing(false), 5000)
        }
      },
      onContentUpdate: (content: string) => {
        // v168：检测正文从"无"到"有"的跃迁——首次拿到非空正文时弹成功提示
        setPlanContent(prev => {
          const wasEmpty = !prev || prev.trim().length === 0
          const nowHas = !!(content && content.trim().length > 0)
          if (wasEmpty && nowHas) {
            showContentToast('✅ 教案正文已生成！可在右侧「教案预览」查看，发布后也可在「我的教案」中随时查看')
            setFullGenerating(false)  // v168(功能B): 全委托出稿落库成功，复位按钮状态
          }
          return content
        })
      },
      onReviewDone: r => {
        setReviewLoading(false); setApplyingReview(false)
        setReview(r); setRightPanel('review')
      },
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      onStageStarted: (_data: StageEventData) => { refreshStages(planId) },
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      onStageComplete: (_data: StageEventData) => {
        setIsStageProcessing(false)
        refreshStages(planId)
      },
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      onStageOutput: (_data: StageEventData) => { refreshStages(planId) },
      onError: err => {
        setIsThinking(false); setStreaming(null)
        setReviewLoading(false); setApplyingReview(false)
        setIsStageProcessing(false)
        setFullGenerating(false)  // v168(功能B): 出错时复位一键生成状态，避免按钮卡死
        setMessages(prev => [...prev, {
          id: `err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
          content: `抱歉,遇到了一点问题:${err}。你可以重试或换个方式表达。`,
          created_at: new Date().toISOString(),
        }])
      },
      // v172（已撤销后端 done 补发后的现状说明）：
      // 后端曾短暂尝试每轮补发 done，但因 SSE 是「教案会话级共享长连接」，发 done 会触发
      // createLessonPlanSSE 的 es.close() 关掉整条连接，导致进入阶段的开场白那轮 done 关连接后、
      // 随后「一键生成」无连接可收 → 已撤销后端补发。故当前后端正常流程【不再主动发 done】，
      // 本回调实际不会被触发，仅作为无害兜底保留：若将来后端在「真正会话结束」时再发 done，
      // 这里能复位「生成中」类状态。日常「生成中」复位由上方 onMessageDone 负责，不依赖本回调。
      onDone: () => {
        setIsThinking(false); setStreaming(null)
        setIsStageProcessing(false)
        setFullGenerating(false)
      },
      // v88新增:连接状态变化回调 → 驱动顶部指示器颜色
      onConnectionStateChange: (state: SSEConnectionState) => {
        setSseConnectionState(state)
      },
      // v88新增:重连成功后自动补齐丢失消息
      onReconnected: async () => {
        const currentPlanId = planIdRef.current
        if (!currentPlanId) return
        try {
          console.log('[SSE] 重连成功,开始补齐对话消息...')
          const convData = await getConversation(currentPlanId)
          const serverMsgs = (convData.messages || []).filter(
            (m: ConversationMessage) => m.role === 'user' || m.role === 'assistant' || m.role === 'system'
          )
          setMessages(prev => {
            if (serverMsgs.length > prev.length) {
              console.log(`[SSE] 补齐完成:本地${prev.length}条 → 服务端${serverMsgs.length}条`)
              return serverMsgs
            }
            console.log(`[SSE] 无需补齐:本地${prev.length}条 >= 服务端${serverMsgs.length}条`)
            return prev
          })
          const planData = await getLessonPlan(currentPlanId)
          if (planData.content_markdown) setPlanContent(planData.content_markdown)
          if (planData.current_stage && planData.stage_config) {
            await refreshStages(currentPlanId)
          }
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

        // v168：从详情页「前往补全正文」入口跳来时，会带 workshop_target_stage 信号，
        // 恢复完成后自动切到指定阶段（通常是 write 教案撰写），消除"承诺去撰写、实际停在评审/修订"的落差。
        // 用 switchToStage（与"继续该阶段对话"按钮同款 API，阶段可逆），用完即清，不影响下次正常恢复。
        try {
          const targetStage = sessionStorage.getItem('workshop_target_stage')
          if (targetStage) {
            sessionStorage.removeItem('workshop_target_stage')
            // 仅当该阶段存在于当前教案阶段列表时才切，避免对旧模式/无该阶段的教案误操作
            const exists = (planData.current_stage && planData.stage_config)
              ? true : false
            if (exists && planData.current_stage !== targetStage) {
              await switchToStage(effectivePlanId, targetStage)
              await refreshStages(effectivePlanId)
            }
          }
        } catch (switchErr) {
          // 切阶段失败不阻断恢复流程，用户仍可手动在左侧阶段栏切换
          console.error('恢复后切换目标阶段失败:', switchErr)
        }
      } catch (e) {
        console.error('恢复教案失败:', e)
        setResumeError('加载教案失败,请重试')
        setPhase('start')
      }
    }
    resumePlan()
  }, [effectivePlanId, phase, connectSSE, refreshStages])

  // v108:导入教案成功后回调
  const handleImportSuccess = async (planId: string, openingMessage: ConversationMessage) => {
    setShowImportModal(false)
    try {
      const [planData, convData] = await Promise.all([
        getLessonPlan(planId),
        getConversation(planId),
      ])
      setPlan(planData)
      const serverMsgs = (convData.messages || []).filter(
        (m: ConversationMessage) => m.role === 'user' || m.role === 'assistant' || m.role === 'system'
      )
      setMessages(serverMsgs.length > 0 ? serverMsgs : [openingMessage])
      if (planData.content_markdown) setPlanContent(planData.content_markdown)
      setPhase('chatting')
      sessionStorage.setItem('workshop_active_plan_id', planId)
      connectSSE(planId)
      if (planData.current_stage && planData.stage_config) {
        await refreshStages(planId)
      }
      setRightPanel('review')  // 自动切到评审面板等待AI评审
    } catch (err) {
      console.error('导入后加载教案失败:', err)
      alert('导入成功但加载失败,请刷新页面重试')
    }
  }

  const handleStart = async (subject: string, grade: string, topic: string, duration: number, recipeId?: string, textbookPageIds?: string[]) => {
    setStartLoading(true)
    try {
      const req: Record<string, unknown> = { subject, grade, topic, duration_minutes: duration }
      if (recipeId) req.recipe_id = recipeId
      if (textbookPageIds && textbookPageIds.length > 0) req.textbook_page_ids = textbookPageIds
      const resp = await startConversation(req as unknown as Parameters<typeof startConversation>[0])
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
      alert('开始备课失败,请稍后重试')
    } finally { setStartLoading(false) }
  }

  // v88增强:消息发送增加重试机制
  // v112 (P0 STEP 8):发送消息时透传 assistant_id 给后端
  const handleSend = async () => {
    if (!plan || (!inputText.trim() && selectedComponentIds.size === 0)) return
    const msgText = inputText.trim()
    setInputText('')

    const localMsg: ConversationMessage = {
      id: `local_${Date.now()}`, role: 'user' as const, type: 'text' as const,
      content: msgText || `已选择${selectedComponentIds.size}个组件`,
      created_at: new Date().toISOString(),
    }
    setMessages(prev => [...prev, localMsg])
    setIsThinking(true)

    const componentIds = Array.from(selectedComponentIds)
    let lastErr: unknown = null
    for (let attempt = 0; attempt <= SEND_RETRY_MAX; attempt++) {
      try {
        // v112:assistant_id 透传,null 表示走后端兜底默认 prompt
        await sendChatMessage(plan.id, {
          message: msgText,
          selected_components: componentIds,
          assistant_id: assistantId,
        })
        setSelectedComponentIds(new Set())
        lastErr = null
        break
      } catch (err) {
        lastErr = err
        if (attempt < SEND_RETRY_MAX) {
          console.warn(`[Send] 发送失败,${1}秒后重试第${attempt + 1}次...`)
          await new Promise(resolve => setTimeout(resolve, 1000))
        }
      }
    }

    if (lastErr) {
      setIsThinking(false)
      console.error('发送消息失败(含重试):', lastErr)
      setMessages(prev => [...prev, {
        id: `send_err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
        content: '⚠️ 消息发送失败,请检查网络后重试。你刚才的内容已保留在输入框中。',
        created_at: new Date().toISOString(),
      }])
      setInputText(msgText)
    }
  }

  // v168(功能B): 一键生成完整教案
  // 仅 write 阶段 + 正文为空时可触发。二次确认后，用固定触发语 + full_generate:true 调 chat，
  // 后端注入全委托出稿指令一次性产出完整教案，走现有 SSE → onContentUpdate 落库反馈（自动弹成功 toast）。
  const handleFullGenerate = async (stageOverride?: string) => {
    if (!plan || fullGenerating || isBusy) return
    // v169：按阶段返回文案配置；不在表中的阶段（如 review）不支持一键生成
    const FULL_GEN_STAGE_META: Record<string, { name: string; trigger: string; confirmBody: string }> = {
      analyze: {
        name: '教学分析',
        trigger: '请一次性完成本节课的完整教学分析（教材分析、课程标准对接、学情分析、核心概念与重难点预判）。',
        confirmBody: '将由 AI 一次性生成完整的教学分析（教材分析、课程标准、学情分析、重难点预判）。',
      },
      design: {
        name: '教学设计',
        trigger: '请一次性完成本节课的完整教学设计方案（教学目标、重难点、教学策略、活动设计、评价设计）。',
        confirmBody: '将由 AI 一次性生成完整的教学设计方案（教学目标、重难点、教学策略、活动设计、评价设计）。',
      },
      write: {
        name: '教案撰写',
        trigger: '请一次性生成这节课的完整教案正文。',
        confirmBody: '将由 AI 一次性生成完整的教案正文（教学目标、重难点、教学过程、作业、板书等）。',
      },
      revise: {
        name: '修订定稿',
        trigger: '请基于已有教案正文和 AI 评审建议，一次性输出修订后的完整教案。',
        confirmBody: '将由 AI 参考已有教案正文与评审建议，一次性输出修订后的完整教案。',
      },
    }
    const stage = stageOverride || currentStage
    const meta = FULL_GEN_STAGE_META[stage]
    if (!meta) return

    const ok = window.confirm(
      `⚡ 一键完成「${meta.name}」\n\n` +
      meta.confirmBody +
      ' 整个过程无需你逐步确认。\n\n' +
      '⚠️ 重要提醒：这是纯 AI 一次性生成的内容，可能存在幻觉，或与你班级的真实学情、所用教材不完全相符。' +
      '请务必通读核对，结合实际情况修改后再使用。\n\n' +
      '生成后可继续对话局部调整，或用「🔄 重启本阶段」重新来过。\n\n' +
      '确定现在开始生成吗？'
    )
    if (!ok) return

    setFullGenerating(true)
    setIsThinking(true)
    const triggerMsg = meta.trigger
    const localMsg: ConversationMessage = {
      id: `local_${Date.now()}`, role: 'user' as const, type: 'text' as const,
      content: triggerMsg,
      created_at: new Date().toISOString(),
    }
    setMessages(prev => [...prev, localMsg])
    try {
      await sendChatMessage(plan.id, {
        message: triggerMsg,
        assistant_id: assistantId,
        full_generate: true,
      })
    } catch (err) {
      setIsThinking(false)
      setFullGenerating(false)
      console.error('一键生成失败:', err)
      // v169：优先显示后端返回的真实错误（如"教案当前状态不可编辑"），而非笼统的网络提示
      const errMsg = err instanceof Error ? err.message : ''
      const friendly = errMsg && errMsg !== '请求失败'
        ? `⚠️ 一键生成未能开始：${errMsg}。${errMsg.includes('状态') ? '该教案可能已提交评审或已完成，无法继续生成。' : '请稍后重试。'}`
        : '⚠️ 一键生成请求发送失败，请检查网络后重试。'
      setMessages(prev => [...prev, {
        id: `fullgen_err_${Date.now()}`, role: 'assistant' as const, type: 'text' as const,
        content: friendly,
        created_at: new Date().toISOString(),
      }])
    }
  }

  const handleSelectComponent = (comp: ConvComponent) => {
    setSelectedComponentIds(prev => {
      // eslint-disable-next-line @typescript-eslint/no-unused-expressions
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
    } catch (err) { console.error('发布失败:', err); alert('发布失败,请稍后重试') }
  }

  // v79-2:退出备课(保存草稿+回到首屏)
  // v112 (P0 STEP 8):退出时清理 assistantId,和其他 state 清理对齐
  const handleExitWorkshop = () => {
    if (!plan) return
    const confirmMsg = '确定退出当前备课吗?\n\n教案已自动保存为草稿,你可以随时从「我的教案」继续。'
    if (!confirm(confirmMsg)) return
    sseRef.current?.close()
    sseRef.current = null
    sessionStorage.removeItem('workshop_active_plan_id')
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
    setSseConnectionState('connected')
    setRevisionBannerDismissed(false)
    setAssistantId(null)  // v112:清理助手选择
    setStartMode('choose')  // v108:退出后回到选择卡片状态
    setPhase('start')
  }

  // v88新增:手动重连
  const handleManualReconnect = () => {
    if (!plan) return
    setSseConnectionState('reconnecting')
    connectSSE(plan.id)
  }

  // v113 (P0 STEP 6):Modal 触发函数
  const handleEditAssistant = (id: string) => {
    setModalMode('edit')
    setModalEditId(id)
    setModalOpen(true)
  }
  const handleCreateAssistant = () => {
    setModalMode('create-personal')
    setModalEditId(undefined)
    setModalOpen(true)
  }
  // Modal 保存成功后的回调:切换到新 ID + 强制 Selector 刷新列表
  // v121 Bug 2 修复:AssistantSelector 的 onDelete 回调
  const handleDeleteAssistant = async (id: string) => {
    try {
      await deleteAssistant(id)
      if (assistantId === id) setAssistantId(null)
      setSelectorKey(k => k + 1)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '删除失败'
      alert('删除失败:' + msg)
    }
  }
  const handleAssistantSaved = (id: string) => {
    setAssistantId(id)
    setSelectorKey(k => k + 1)
  }

  // P2:点击完成本阶段
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

  // 迭代12:实际执行 advanceStage 并插入分隔符
  const doAdvanceStage = async (planId: string, nextStageItem: StageProgressItem | null, selectedCompIds: string[]) => {
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

  // 迭代12:组件弹窗回调
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

  // P2+P3:确认进入下一阶段
  const handleConfirmTransition = async () => {
    if (!plan) return

    // v168：离开「教案撰写」阶段时，若正文仍未生成 → confirm 拦截
    // 背景：用户在 write 阶段只走开场白没真正生成正文就推进，会导致"有分无正文"。
    // 这里在确认进入下一阶段前主动提示，把决定权明确交还用户（可取消回去补，也可坚持推进）。
    if (currentStage === 'write' && (!planContent || planContent.trim().length === 0)) {
      const proceed = confirm(
        '⚠️ 检测到教案正文尚未生成。\n\n' +
        '建议先在「教案撰写」阶段让 AI 写出完整的教案正文，' +
        '否则后续 AI 评审将无内容可评、教案详情页也会显示「暂无教案内容」。\n\n' +
        '确定仍要继续进入下一阶段吗？'
      )
      if (!proceed) return
    }

    setShowSummaryModal(false)

    const currentIdx      = stageItems.findIndex(s => s.stage_code === currentStage)
    const currentStageItem = stageItems[currentIdx]
    const nextStageItem   = currentIdx >= 0 && currentIdx < stageItems.length - 1
      ? stageItems[currentIdx + 1] : null
    const isLastStage = !nextStageItem

    if (isLastStage) {
      sessionStorage.removeItem('workshop_active_plan_id')
      navigate('/lesson-plans/my-plans')
      return
    }

    setTransitionInfo({
      currentName: currentStageItem?.stage_name || currentStage,
      nextName: nextStageItem.stage_name,
      nextRole: nextStageItem.ai_role,
    })
    setIsTransitioning(true)
    setTransitionStep(0)

    const t1 = setTimeout(() => setTransitionStep(1), 700)
    const t2 = setTimeout(() => setTransitionStep(2), 1400)
    const t3 = setTimeout(() => {
      setIsTransitioning(false)
      setTransitionStep(0)
      setTransitionInfo(null)
      if (nextStageItem && STAGES_WITH_COMPONENTS.includes(nextStageItem.stage_code)) {
        setPendingTransitionStage(nextStageItem.stage_code)
        setShowComponentsModal(true)
      } else {
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

  // 迭代12新增:重启指定阶段
  const handleResetStage = async (stageCode: string) => {
    if (!plan) return
    const stageName = stageItems.find(s => s.stage_code === stageCode)?.stage_name || stageCode
    if (!confirm(`确定要重启「${stageName}」阶段吗?\n\n该阶段及之后阶段的产出物和对话将被清空。`)) return
    try {
      await resetStage(plan.id, stageCode)
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
    } catch (err) { console.error('重启阶段失败:', err); alert('重启阶段失败,请重试') }
  }

  // ==================== 恢复中 ====================
  if (phase === 'resuming') {
    return <ResumingView resumeError={resumeError} />
  }

  // ==================== 首屏 ====================
  if (phase === 'start') {
    return (
      <StartScreen
        needsAssessment={needsAssessment}
        setNeedsAssessment={setNeedsAssessment}
        startMode={startMode}
        setStartMode={setStartMode}
        startLoading={startLoading}
        showImportModal={showImportModal}
        setShowImportModal={setShowImportModal}
        onStart={handleStart}
        onImportSuccess={handleImportSuccess}
      />
    )
  }

  // ==================== 备课中 ====================
  const isAIActive = isThinking || !!streaming || isStageProcessing
  const isViewingHistory = !!(viewingStage && viewingStage !== currentStage)
  const isBusy = isAIActive || reviewLoading || isTransitioning || isViewingHistory
  const canCompleteStage = isStageMode && currentStage && !isAIActive && !isTransitioning && !summaryLoading

  const currentStageIdx   = stageItems.findIndex(s => s.stage_code === currentStage)
  const nextStageForSummary = currentStageIdx >= 0 && currentStageIdx < stageItems.length - 1
    ? stageItems[currentStageIdx + 1] : null

  const planAny    = plan as Record<string, unknown> | null
  const recipeName = planAny?.recipe_name ? String(planAny.recipe_name) : ''
  const recipeId   = planAny?.recipe_id   ? String(planAny.recipe_id)   : ''

  const fallbackSteps = [
    { key: 'info',     label: '了解学情', done: messages.length >= 2 },
    { key: 'plan',     label: '确认方案', done: messages.length >= 4 },
    { key: 'generate', label: '生成教案', done: !!planContent },
    { key: 'review',   label: 'AI评审',  done: !!review },
    { key: 'save',     label: '保存发布', done: false },
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

        {/* 退出备课快捷入口 */}
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
            {/* 迭代7B：本次参考的课本图缩略展示——让老师明确知道这次备课 AI 参考了哪些课本页 */}
            {linkedTextbooks.length > 0 && (
              <div style={{ marginTop: '8px', padding: '8px 10px', background: 'rgba(16,185,129,0.06)', borderRadius: '8px', border: '1px solid rgba(16,185,129,0.15)' }}>
                <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '6px' }}>📷 参考课本（{linkedTextbooks.length}张）</div>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                  {linkedTextbooks.map(tb => (
                    <div key={tb.id} title={`${tb.chapter || tb.textbook_name}${tb.has_ocr ? '（已识别）' : '（未识别，AI无法参考）'}`}
                      onClick={() => navigate('/lesson-plans/textbooks')}
                      style={{ position: 'relative', cursor: 'pointer', borderRadius: '4px', overflow: 'hidden', border: tb.has_ocr ? '1px solid #10B981' : '1px solid #E5E7EB' }}>
                      <img src={tb.image_url} alt="" style={{ width: '36px', height: '36px', objectFit: 'cover', display: 'block' }}
                        onError={e => { (e.target as HTMLImageElement).style.display = 'none' }} />
                      {!tb.has_ocr && (
                        <div style={{ position: 'absolute', bottom: 0, left: 0, right: 0, background: 'rgba(239,68,68,0.85)', color: '#fff', fontSize: '8px', textAlign: 'center', lineHeight: 1.4 }}>未识别</div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {sidebarCollapsed && plan && (
          <div style={{ marginTop: 'auto', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
            <div title={plan.title} style={{ fontSize: '18px' }}>📝</div>
            {recipeName && <button onClick={() => navigate(`/lesson-plans/recipes/${recipeId}/edit`, { state: { from: '/lesson-plans' } })} title={`配方:${recipeName}`} style={{ fontSize: '18px', cursor: 'pointer', background: 'none', border: 'none', padding: 0 }}>📦</button>}
          </div>
        )}
      </div>

      {/* 中栏 */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', borderRight: `1px solid ${C.border}`, position: 'relative' }}>
        {isTransitioning && transitionInfo && (
          <StageTransitionView currentStageName={transitionInfo.currentName} nextStageName={transitionInfo.nextName} nextStageRole={transitionInfo.nextRole} step={transitionStep} />
        )}

        {/* 退回修改模式提示条 */}
        {plan?.status === 'revision' && !revisionBannerDismissed && (
          <div style={{ padding: '10px 16px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px', background: 'linear-gradient(135deg, #FFF7ED, #FFF3E0)', borderBottom: '2px solid #F97316', flexShrink: 0, flexWrap: 'wrap' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flex: 1 }}>
              <span style={{ fontSize: '18px' }}>📋</span>
              <div>
                <div style={{ fontSize: '13px', fontWeight: 700, color: '#C2410C' }}>当前为退回修改模式</div>
                <div style={{ fontSize: '12px', color: '#92400E', marginTop: '2px', lineHeight: 1.5 }}>教案已被退回,请根据评审批注修改后重新提交——修改完成后前往「教案详情」页提交评审</div>
              </div>
            </div>
            <div style={{ display: 'flex', gap: '8px', flexShrink: 0 }}>
              <button onClick={() => { if (plan) window.open('/lesson-plans/plans/' + plan.id, '_blank') }} style={{ padding: '5px 14px', borderRadius: '8px', border: '1px solid #F97316', background: 'transparent', fontSize: '12px', fontWeight: 600, color: '#C2410C', cursor: 'pointer', whiteSpace: 'nowrap' }}>📝 查看批注</button>
              <button onClick={() => setRevisionBannerDismissed(true)} style={{ padding: '5px 10px', borderRadius: '8px', border: '1px solid #FED7AA', background: 'transparent', fontSize: '12px', color: '#9CA3AF', cursor: 'pointer' }}>×</button>
            </div>
          </div>
        )}

        {/* 网络状态指示器 */}
        {sseConnectionState !== 'connected' && (
          <div style={{ padding: '7px 16px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', fontSize: '13px', fontWeight: 500, borderBottom: `1px solid ${sseConnectionState === 'reconnecting' ? 'rgba(245,158,11,0.3)' : 'rgba(239,68,68,0.3)'}`, background: sseConnectionState === 'reconnecting' ? 'linear-gradient(135deg, rgba(245,158,11,0.08), rgba(251,191,36,0.05))' : 'linear-gradient(135deg, rgba(239,68,68,0.08), rgba(248,113,113,0.05))', color: sseConnectionState === 'reconnecting' ? '#92400E' : '#991B1B', animation: sseConnectionState === 'reconnecting' ? 'sseReconnectPulse 2s ease-in-out infinite' : 'none' }}>
            <div style={{ width: '8px', height: '8px', borderRadius: '50%', flexShrink: 0, background: sseConnectionState === 'reconnecting' ? '#F59E0B' : '#EF4444', boxShadow: sseConnectionState === 'reconnecting' ? '0 0 6px rgba(245,158,11,0.5)' : '0 0 6px rgba(239,68,68,0.5)' }} />
            {sseConnectionState === 'reconnecting' ? (
              <span>网络连接中断,正在尝试重新连接...</span>
            ) : (
              <>
                <span>网络连接已断开</span>
                <button onClick={handleManualReconnect} style={{ padding: '3px 12px', borderRadius: '12px', border: '1px solid rgba(239,68,68,0.4)', background: 'rgba(239,68,68,0.08)', fontSize: '12px', color: '#DC2626', cursor: 'pointer', fontWeight: 600 }}>点击重连</button>
              </>
            )}
          </div>
        )}

        {/*
          工坊阶段头(阶段信息 + AssistantSelector + 页数标签)
          v112 (P0 STEP 8):在原阶段信息 div 右侧插入 AssistantSelector(紧凑模式),
            - scene 随当前阶段动态切换(analyze→workshop_analyze,etc)
            - 切换阶段时 assistantId 会被 useEffect 自动重置,让新阶段匹配新的默认助手
            - 对话进行中禁用,防止中途切助手导致上下文错乱
            - 快速出稿/极速模式只要有阶段头就有选择器,天然覆盖所有模式
          v113 (P0 STEP 6):给 AssistantSelector 挂 onEdit + onCreateNew,触发 AssistantEditModal
        */}
        {isStageMode && currentStage && (() => {
          const cur = stageItems.find(s => s.stage_code === currentStage)
          return (
            <div style={{ padding: '8px 20px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: 'linear-gradient(135deg, rgba(79,123,232,0.06), rgba(129,140,248,0.04))', gap: '12px', flexWrap: 'wrap' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', minWidth: 0 }}>
                <span style={{ fontSize: '16px' }}>{STAGE_CODE_EMOJI[currentStage] || '📋'}</span>
                <div style={{ minWidth: 0 }}>
                  <span style={{ fontSize: '13px', fontWeight: 600, color: C.primary }}>{cur?.stage_name || currentStage}</span>
                  {cur?.ai_role && <span style={{ fontSize: '11px', color: C.textMuted, marginLeft: '8px' }}>· {cur.ai_role}</span>}
                </div>
                <div title={sseConnectionState === 'connected' ? '连接正常' : sseConnectionState === 'reconnecting' ? '重连中...' : '连接断开'} style={{ width: '6px', height: '6px', borderRadius: '50%', marginLeft: '4px', background: sseConnectionState === 'connected' ? '#10B981' : sseConnectionState === 'reconnecting' ? '#F59E0B' : '#EF4444', boxShadow: sseConnectionState === 'connected' ? '0 0 4px rgba(16,185,129,0.4)' : sseConnectionState === 'reconnecting' ? '0 0 4px rgba(245,158,11,0.4)' : '0 0 4px rgba(239,68,68,0.4)', transition: 'background 300ms ease, box-shadow 300ms ease' }} />
              </div>
              {/* v112 (P0 STEP 8):AI 助手选择器
                  v113 (P0 STEP 6):挂 onEdit + onCreateNew + key (保存后强制刷新) */}
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexShrink: 0 }}>
                <AssistantSelector
                  key={selectorKey}
                  scene={STAGE_CODE_TO_SCENE[currentStage] || 'workshop_write'}
                  value={assistantId}
                  onChange={setAssistantId}
                  subject={plan?.subject}
                  grade={plan?.grade}
                  disabled={isBusy}
                  compact
                  onEdit={handleEditAssistant}
                  onDelete={handleDeleteAssistant}
                  onCreateNew={handleCreateAssistant}
                />
                {/* v121 任务B:"📚 选组件"按钮——随时打开组件推荐弹窗,不必等阶段过渡 */}
                {STAGES_WITH_COMPONENTS.includes(currentStage) && (
                  <button
                    onClick={() => setShowPickComponentsModal(true)}
                    disabled={isBusy}
                    title="从组件库挑选参考组件补充到对话上下文"
                    style={{
                      padding: '6px 10px',
                      borderRadius: '8px',
                      border: `1px solid ${C.border}`,
                      background: selectedComponentIds.size > 0 ? C.primaryLight : '#fff',
                      color: selectedComponentIds.size > 0 ? C.primary : C.textSec,
                      fontSize: '12px',
                      fontWeight: selectedComponentIds.size > 0 ? 600 : 500,
                      cursor: isBusy ? 'not-allowed' : 'pointer',
                      opacity: isBusy ? 0.5 : 1,
                      whiteSpace: 'nowrap',
                      transition: 'all 150ms ease',
                    }}
                  >
                    📚 选组件{selectedComponentIds.size > 0 ? `·${selectedComponentIds.size}` : ''}
                  </button>
                )}
                <span style={{ fontSize: '12px', color: C.textMuted }}>{currentStageIdx + 1} / {stageItems.length}</span>
              </div>
            </div>
          )
        })()}

        <div ref={messagesScrollRef} style={{ flex: 1, overflowY: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column' }}>
                    {/* v124: 图片插入引导条 */}
          {!imageTipDismissed && (
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '9px 16px', marginBottom: '12px', background: 'linear-gradient(135deg, rgba(79,123,232,0.06), rgba(16,185,129,0.06))', borderRadius: '10px', border: '1px solid rgba(79,123,232,0.15)', fontSize: '13px', color: '#374151', alignSelf: 'stretch' }}>
              <span style={{ fontSize: '15px', flexShrink: 0 }}>📷</span>
              <span style={{ flex: 1, lineHeight: 1.6 }}>
                想在教案里插入图片？完成备课后，在「<span style={{ fontWeight: 600, color: '#4F7BE8' }}>我的教案 → 详情页</span>」点段落右侧 ✏️ 即可上传图片
              </span>
              <button
                onClick={() => { setImageTipDismissed(true); try { localStorage.setItem('workshop_image_tip_dismissed', '1') } catch {} }}
                style={{ padding: '2px 8px', borderRadius: '6px', border: '1px solid rgba(79,123,232,0.2)', background: 'transparent', fontSize: '12px', color: '#9CA3AF', cursor: 'pointer', flexShrink: 0, lineHeight: 1.4 }}
                title="不再显示此提示"
              >✕</button>
            </div>
          )}
          {resumePlanId && messages.length > 0 && (
            <div style={{ textAlign: 'center', marginBottom: '16px', padding: '8px 16px', background: C.primaryLight, borderRadius: '20px', fontSize: '12px', color: C.primary, alignSelf: 'center' }}>
              🔄 已恢复历史对话,可继续备课
            </div>
          )}
          {(() => {
            const targetStage = viewingStage || currentStage
            let filteredMsgs = messages
            if (isStageMode && targetStage && stageItems.length > 0) {
              let startIdx = -1
              let endIdx   = messages.length
              for (let i = 0; i < messages.length; i++) {
                const m = messages[i]
                if ((m.role as string) === 'system' && m.content.startsWith(STAGE_SEP_PREFIX)) {
                  const rest = m.content.slice(STAGE_SEP_PREFIX.length)
                  const sepStageName = rest.split('__')[0] || ''
                  const matchItem = stageItems.find(s => s.stage_name === sepStageName || s.stage_code === sepStageName)
                  if (matchItem && matchItem.stage_code === targetStage) {
                    startIdx = i
                  } else if (startIdx >= 0 && endIdx === messages.length) {
                    endIdx = i
                  }
                }
              }
              if (startIdx >= 0) {
                filteredMsgs = messages.slice(startIdx, endIdx)
              } else if (targetStage === stageItems[0]?.stage_code) {
                const firstSepIdx = messages.findIndex(m => (m.role as string) === 'system' && m.content.startsWith(STAGE_SEP_PREFIX))
                filteredMsgs = firstSepIdx >= 0 ? messages.slice(0, firstSepIdx) : messages
              }
            }
            return filteredMsgs.filter(m => {
              if (m.role === 'user' && m.content.startsWith('我们进入') && m.content.includes('阶段了。请先简要介绍')) return false
              if (m.role === 'user' && m.content === '请对上一阶段完成的教案进行全面专业评审,直接输出评审报告,包含各维度评分和改进建议。') return false
              return true
            })
          })().map(msg => {
            if ((msg.role as string) === 'system' && msg.content.startsWith(STAGE_SEP_PREFIX)) {
              const rest = msg.content.slice(STAGE_SEP_PREFIX.length)
              const [stageName, aiRole] = rest.split('__')
              // v121 任务C:反查阶段代码 + 上一阶段名,让三段式分段条完整激活
              // - nextStageCode:通过 stage_name 反查对应的 stage_code,匹配 STAGE_CODE_EMOJI 图标
              // - prevStageName:取 stage_order 比当前小 1 的阶段名,渲染顶部"✅ XX 阶段已完成"收束条
              const nextStageItem = stageItems.find(s => s.stage_name === stageName)
              const nextStageCode = nextStageItem?.stage_code
              const prevStageItem = nextStageItem
                ? stageItems.find(s => s.stage_order === nextStageItem.stage_order - 1)
                : null
              return (
                <StageSeparatorBubble
                  key={msg.id}
                  stageName={stageName || ''}
                  aiRole={aiRole || ''}
                  nextStageCode={nextStageCode}
                  prevStageName={prevStageItem?.stage_name}
                />
              )
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
            <span style={{ color: '#065F46', fontWeight: 500 }}>AI认为本阶段工作已完成,你可以继续深入探讨,或点击下方按钮进入下一阶段</span>
          </div>
        )}

        {isStageProcessing && (
          <div style={{ padding: '8px 20px', background: 'rgba(79,123,232,0.07)', borderTop: `1px solid rgba(79,123,232,0.14)`, display: 'flex', alignItems: 'center', gap: '10px', fontSize: '13px', color: C.primary }}>
            <div style={{ width: '14px', height: '14px', border: `2px solid ${C.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite', flexShrink: 0 }} />
            <span>正在整理阶段成果,请稍候...</span>
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
            <textarea value={inputText} onChange={e => setInputText(e.target.value)} onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }} placeholder={isBusy ? 'AI处理中...' : sseConnectionState === 'disconnected' ? '网络已断开,请先重连...' : '告诉AI你的想法... (Enter发送,Shift+Enter换行)'} rows={2} disabled={isBusy} style={{ flex: 1, background: 'transparent', border: 'none', outline: 'none', fontSize: '15px', color: C.text, resize: 'none', fontFamily: 'inherit', lineHeight: 1.6, opacity: isBusy ? 0.5 : 1 }} />
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

            {/* v169(多阶段一键生成): 一键完成本阶段按钮
                显示条件：阶段模式 + 当前阶段支持一键生成(analyze/design/write/revise) + 该阶段尚无产出
                - write/revise：看 planContent(正文)是否为空
                - analyze/design：看 stageItems 里该阶段的 has_output 是否为 false
                review 阶段不在 FULL_GEN_STAGE_BTN 中，不显示按钮（推进过去自动评审）*/}
            {isStageMode && currentStage && (() => {
              // 阶段中文名（按钮文案用），与 handleFullGenerate 内 FULL_GEN_STAGE_META 对齐
              const stageBtnNames: Record<string, string> = {
                analyze: '教学分析', design: '教学设计', write: '教案撰写', revise: '修订定稿',
              }
              const stageName = stageBtnNames[currentStage]
              if (!stageName) return null  // review 等不支持的阶段
              // 判空逻辑（v169修复"第一次进入 analyze/design 看不到按钮"）：
              //   - write/revise：看 planContent(正文)是否为空（正文落在 lesson_plans.content_markdown，准确）
              //   - analyze/design：原先用 has_output 判空，但阶段开场白也会被存为产出物、污染 has_output，
              //     导致一进阶段 has_output 就变 true、按钮被错误隐藏。改为看阶段 status：
              //     只要阶段尚未"完成/跳过"（仍是 pending/in_progress），就应允许一键生成。
              const contentEmpty = !planContent || planContent.trim().length === 0
              const curStageItem = stageItems.find(s => s.stage_code === currentStage)
              const stageFinished = curStageItem?.status === 'completed' || curStageItem?.status === 'skipped'
              const shouldShow = (currentStage === 'write' || currentStage === 'revise')
                ? contentEmpty
                : !stageFinished
              if (!shouldShow) return null
              return (
                <button
                  onClick={() => handleFullGenerate()}
                  disabled={isBusy || fullGenerating}
                  title={`由 AI 一次性生成完整的${stageName}内容，无需逐步确认（生成后请务必核对）`}
                  style={{
                    padding: '6px 14px',
                    borderRadius: '20px',
                    border: 'none',
                    background: (isBusy || fullGenerating) ? '#E5E7EB' : 'linear-gradient(135deg, #F59E0B, #FBBF24)',
                    color: (isBusy || fullGenerating) ? C.textMuted : '#fff',
                    fontSize: '12px',
                    fontWeight: 600,
                    cursor: (isBusy || fullGenerating) ? 'not-allowed' : 'pointer',
                    boxShadow: (isBusy || fullGenerating) ? 'none' : '0 2px 8px rgba(245,158,11,0.3)',
                    transition: 'all 150ms ease',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {fullGenerating ? '⏳ 生成中...' : `⚡ 一键完成${stageName}`}
                </button>
              )
            })()}

            {isStageMode && currentStage && (
              <button onClick={handleCompleteStageClick} disabled={!canCompleteStage} style={{ marginLeft: 'auto', padding: '7px 16px', borderRadius: '20px', border: 'none', background: !canCompleteStage ? '#E5E7EB' : aiSuggestsComplete ? 'linear-gradient(135deg, #10B981, #34D399)' : 'linear-gradient(135deg, #4F7BE8, #818CF8)', color: !canCompleteStage ? C.textMuted : '#fff', fontSize: '13px', fontWeight: 600, cursor: !canCompleteStage ? 'not-allowed' : 'pointer', transition: 'all 200ms ease', boxShadow: canCompleteStage && aiSuggestsComplete ? '0 3px 12px rgba(16,185,129,0.35)' : canCompleteStage ? '0 3px 10px rgba(79,123,232,0.3)' : 'none', animation: canCompleteStage && aiSuggestsComplete ? 'completePulse 2s ease-in-out infinite' : 'none', whiteSpace: 'nowrap' }}>
                {summaryLoading ? '加载中...' : nextStageForSummary ? `✅ 完成本阶段,进入${nextStageForSummary.stage_name} →` : '🎉 完成备课'}
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
                      {reviewLoading
                        ? <div style={{ fontSize: '13px', color: C.primary }}>AI正在评审中...</div>
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
            <button onClick={handleExitWorkshop} style={{ flex: 1, padding: '9px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '13px', color: C.textSec, cursor: 'pointer' }} title="退出备课,教案自动保存为草稿">🚪 退出备课</button>
            <button onClick={handlePublish} style={{ flex: 1, padding: '9px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>发布教案 →</button>
          </div>
        )}
      </div>

      {/* 阶段组件推荐弹窗(阶段过渡时) */}
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

      {/* v121 任务B:随时选组件弹窗(阶段进行中任意时刻打开)
          - 走独立的 confirm 回调:把选中的组件ID合并到 selectedComponentIds,供下次 handleSend 使用
          - 不调用 advanceStage,只是"补充上下文",老师可以继续聊
          - 用 mode="pick-only" 让弹窗隐藏"跳过"按钮(那是过渡专属)
      */}
      {showPickComponentsModal && plan && currentStage && (
        <StageComponentsModal
          planId={plan.id}
          stageCode={currentStage}
          stageName={stageItems.find(s => s.stage_code === currentStage)?.stage_name || currentStage}
          mode="pick-only"
          onConfirm={(ids) => {
            setSelectedComponentIds(prev => {
              const next = new Set(prev)
              ids.forEach(id => next.add(id))
              return next
            })
            setShowPickComponentsModal(false)
          }}
          onSkip={() => setShowPickComponentsModal(false)}
          onCancel={() => setShowPickComponentsModal(false)}
        />
      )}

      {/* 阶段完成弹窗 */}
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

      {/* v108:导入已有教案弹窗 */}
      {showImportModal && (
        <ImportPlanModal
          onSuccess={handleImportSuccess}
          onCancel={() => setShowImportModal(false)}
        />
      )}

      {/*
        v113 (P0 STEP 6):AI 助手编辑弹窗挂载点
        defaultScene 随当前阶段动态变化,新助手的场景勾选与所在阶段一致
        subject/grade 从 plan 透传,便于用户新建助手时快速填好匹配维度
      */}
      <AssistantEditModal
        open={modalOpen}
        mode={modalMode}
        assistantId={modalEditId}
        defaultScene={STAGE_CODE_TO_SCENE[currentStage]}
        defaultSubject={plan?.subject}
        defaultGrade={plan?.grade}
        onClose={() => setModalOpen(false)}
        onSaved={(id) => handleAssistantSaved(id)}
      />

      {/* v168：正文生成成功提示 toast（fixed 定位，复用 PlanDetailTabs 的轻提示风格） */}
      {contentToast && (
        <div style={{ position: 'fixed', bottom: '28px', left: '50%', transform: 'translateX(-50%)', maxWidth: '560px', padding: '12px 22px', borderRadius: '10px', background: 'linear-gradient(135deg, #10B981, #34D399)', color: '#fff', fontSize: '13px', fontWeight: 500, lineHeight: 1.6, boxShadow: '0 6px 24px rgba(16,185,129,0.4)', zIndex: 10000, textAlign: 'center' }}>
          {contentToast}
        </div>
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

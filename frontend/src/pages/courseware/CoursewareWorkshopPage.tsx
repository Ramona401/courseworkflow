/**
 * 课件工坊主页面 — CoursewareWorkshopPage.tsx（批次5b-2模块化拆分后·编排壳）
 *
 * 六步向导主页面（生成方案→确认方案→选风格→确认导航栏→批量生成→确认提交）。
 *
 * 【模块化拆分史】原1877行：
 *   - W1/W2/W3批次：常量(workshopConstants)/预览注入(previewInject)/全屏预览/放映/
 *     多媒体管理(MediaManagerPanel)/单页微调(RefinePanel)/外观(AppearancePanel)/
 *     模板保存(TemplateSavePanel)已陆续抽出；
 *   - 批次5b-2：再抽 PagePreviewBlock(胶片条+页预览+MsgBar) / SchemeSteps(Step0+1) /
 *     NavConfirmStep(Step3)，并剪除W1/W2搬家后遗留的死导入，本文件回到600行红线内。
 *
 * 【批次1a·学科工具聚合Tab，2026-07-08】Step5 工作台 Tab 从 12 收敛到 10：
 *   笔顺/公式/五线谱三个学科工具 Tab 收编为一个「🧪学科工具」聚合 Tab
 *   （SubjectToolsPanel 内部宫格选择器），为后续数学动态图形/分子实验室/
 *   物理场景/PhET 等组件入驻铺路——新组件只加宫格卡片，Tab 栏零膨胀。
 *   MediaManagerPanel 的 mediaTab 同步收窄回 image|video|audio 三值。
 *
 * 本文件保留职责（编排壳）：
 *   - 路由/课件加载(loadCourseware)/局部刷新(refreshPagesOnly)/步骤状态机；
 *   - 风格锚点设/清（顶部锚点条与MediaManagerPanel共用，必须留在页面级）；
 *   - Step2选风格、Step4批量生成（SSE并发收尾口径+断线重连补齐）、Step5工作台Tab；
 *   - 全屏预览/放映两弹窗与共享SSE句柄(sseRef)的生命周期。
 *
 * 关键设计（保留供理解）：
 *   - buildPreviewNum 是 Step5 唯一选中页真相源；pages 是方案页面真相源；
 *     previewPages/generatedPages 由 loadCourseware 恢复、SSE 增量更新。
 *   - source_type==='3d_single' 时早返回走 ThreeDSingleView，不走标准六步。
 *   - 批量生成支持中断续传（跳过已生成页）。
 *
 * 【P1-02 体验修复·解读A单一真相源·定稿】
 *   "退出全屏/放映后停留当前页" 采用解读A：全屏里翻到哪页，退出后整个工作台就停在哪页。
 *   实现关键——子组件退出时经 onClose(finalPage) 回传最终页，父组件把它直接写进
 *   buildPreviewNum（Step5 选中页的唯一真相源），而非写进 fullscreenPageNum/slideshowInitPage
 *   这两个"开窗初值"中间态。于是列表高亮、页预览、下次再开全屏全部自动同步到该页，
 *   一处真相、永不打架。fullscreenPageNum/slideshowInitPage 仅作"开窗那一刻喂初值"用，
 *   始终由 buildPreviewNum 派生，不再各记各的（早期各记各的导致回写被开窗值反复覆盖）。
 *   翻页全程子组件不回写父组件，杜绝翻页回弹。
 *
 * 【教案对照抽屉扩展】
 *   LessonPlanRefDrawer 挂载条件从 Step4/5 扩展到 Step1+（确认方案时也能对照教案原文）。
 *   抽屉本身懒加载（首次点开才请求）+ has_lesson_plan=false 时自隐藏，零回归。
 */
import { useState, useEffect, useRef, useCallback, lazy, Suspense } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  getCourseware, getCoursewarePages, subscribeCWIndexSSE,
  generateCWPages, CW_STATUS_CONFIG, cancelGenerate,
  rollbackCWStatus, setStyleAnchor, clearStyleAnchor,
} from '@/api/coursewares'
import type { CoursewareDetail, CoursewarePage, SetStyleAnchorResult } from '@/api/coursewares'
import StyleSelector from './components/StyleSelector'
import { useAuth } from '@/store/auth'
import ThreeDSingleView from './components/ThreeDSingleView'
import { C, STEPS, statusToStep } from './components/courseware-workshop/workshopConstants'
import CWFullscreenPreview from './components/courseware-workshop/CWFullscreenPreview'
import SlideshowPlayer from './components/courseware-workshop/SlideshowPlayer'
import MediaManagerPanel from './components/courseware-workshop/MediaManagerPanel'
import RefinePanel from './components/courseware-workshop/RefinePanel'
import AnnotationPanel from './components/courseware-workshop/AnnotationPanel'
import CollabPanel from './components/courseware-workshop/CollabPanel'
import AppearancePanel from './components/courseware-workshop/AppearancePanel'
import TemplateSavePanel from './components/courseware-workshop/TemplateSavePanel'
import PagePreviewBlock, { MsgBar } from './components/courseware-workshop/PagePreviewBlock'
import type { PageItem } from './components/courseware-workshop/PagePreviewBlock'
import { listCWAnnotations, type CoursewareAnnotation } from '@/api/coursewares'
import SchemeSteps from './components/courseware-workshop/SchemeSteps'
import LessonPlanRefDrawer from './components/courseware-workshop/LessonPlanRefDrawer'
import NavConfirmStep from './components/courseware-workshop/NavConfirmStep'
import DeliveryModeSelect, { type DeliveryMode } from './components/courseware-workshop/DeliveryModeSelect'
import AutoAssemblyPanel from './components/courseware-workshop/AutoAssemblyPanel'

/**
 * 学科工具面板按需加载边界。
 *
 * SubjectToolsPanel会继续引入地理、生命科学、物理、化学等全部实验室。
 * 只有老师真正切换到“🧪 学科工具”Tab时，浏览器才下载并解析整套代码。
 * 不改变任何模板ID、分组、顺序、参数、预览和融入课件逻辑。
 */
const LazySubjectToolsPanel = lazy(
  () => import(
    './components/courseware-workshop/SubjectToolsPanel'
  ),
)

// ==================== 主组件 ====================
export default function CoursewareWorkshopPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const [courseware, setCourseware] = useState<CoursewareDetail | null>(null)
  // 阶段4 集体备课·受限视图：当前用户是"参与者"（非作者、非admin）来访被邀请的课件。
  //   参与者只能在 Step5 工作台微调页面，不能走生成向导前几步、不能改课件级配置。
  //   必须放在 courseware state 声明之后(它引用 courseware),否则 tsc -b 报 used before declaration。
  // 阶段4治本补丁：真集体备课 = 标记态 in_session 且当前在场参与者数 > 0。
  //   作者忘记结束会留下 in_session 但 0 人的'幽灵会话'，此时不应触发任何集体备课 UI。
  //   collab_member_count 由后端 GetCourseware 用 CountCollabMembers 填充。
  const activeCollab = courseware?.collab_state === 'in_session' && (courseware?.collab_member_count ?? 0) > 0
  // 参与者视图（非作者非admin）只在'真集体备课'时才成立：这样幽灵会话不会把普通访客误标为参与者、
  //   也不会把 Step5 工作台错误锁定给他。真会话下参与者体验完全不变。
  const isParticipant = !!(user && courseware && user.id !== courseware.user_id && user.role !== 'admin' && activeCollab)
  const [loading, setLoading] = useState(true)
  const [activeStep, setActiveStep] = useState(0)
  const [maxStepReached, setMaxStepReached] = useState(0)

  // v136: 跳转步骤并追踪最远到达
  const goToStep = (step: number) => {
    setActiveStep(step)
    setMaxStepReached(prev => Math.max(prev, step))
  }
  const [pages, setPages] = useState<CoursewarePage[]>([])

  // Step 3 数据真相源：封面预览页（loadCourseware恢复时填充，生成逻辑在NavConfirmStep内）
  const [previewPages, setPreviewPages] = useState<PageItem[]>([])

  // Step 4: 批量生成状态
  const [buildRunning, setBuildRunning] = useState(false)
  const [buildMessage, setBuildMessage] = useState('')
  const [buildProgress, setBuildProgress] = useState({ current: 0, total: 0 })
  const [generatedPages, setGeneratedPages] = useState<PageItem[]>([])
  const [buildPreviewNum, setBuildPreviewNum] = useState(0)
  // 交付模式（三档）：null=尚未选择(显示三档选择器)；manual走现有批量生成；no_video/full走全自动装配面板。
  //   仅在"全新未开始生成"时才让老师选；已生成/续传/完成态不介入(装配与手动互斥,只在起点分流)。
  const [deliveryMode, setDeliveryMode] = useState<DeliveryMode | null>(null)

  // 批次W2: 页面微调/模板保存已迁入 RefinePanel/TemplateSavePanel
  // 批次5b-2: Step0/1方案状态迁入 SchemeSteps; Step3封面生成状态迁入 NavConfirmStep

  // v136: 步骤回退运行态
  const [rollingBack, setRollingBack] = useState(false)

  // v137: 全屏预览状态（带工具栏，非放映模式）
  // P1-02解读A: fullscreenPageNum/slideshowInitPage 仅作"开窗那一刻喂给子组件的初值"，
  //   退出后真相回写到 buildPreviewNum；这两个 state 由 buildPreviewNum 在开窗时派生。
  const [fullscreenOpen, setFullscreenOpen] = useState(false)
  const [fullscreenPageNum, setFullscreenPageNum] = useState(1)
  const [fullscreenCodeView, setFullscreenCodeView] = useState(false)

  const [slideshowOpen, setSlideshowOpen] = useState(false)
  const [slideshowInitPage, setSlideshowInitPage] = useState(1)
  // 批次W2: Step5工作台Tab(默认「页面微调」=最高频动作; 图/视频/音频三Tab共用MediaManagerPanel实例)
  // 批次1a: stroke/formula/music 三值收编为 subject 一值(SubjectToolsPanel聚合宫格), Tab 12→10
  const [wsTab, setWsTab] = useState<'refine' | 'background' | 'font' | 'image' | 'video' | 'audio' | 'subject' | 'template' | 'annotation' | 'collab'>('refine')  // W3: 背景/字体独立Tab
  // 阶段2: 页级批注全集(loadCourseware/手动刷新时加载;按 buildPreviewNum 过滤当前页)
  const [cwAnnotations, setCwAnnotations] = useState<CoursewareAnnotation[]>([])
  const reloadAnnotations = useCallback(async () => {
    if (!id) return
    try { const r = await listCWAnnotations(id); setCwAnnotations(r.annotations || []) } catch { /* 静默:批注非核心流程,失败不阻断工坊 */ }
  }, [id])

  // ---- 风格锚点（轮3）：设/清锚点运行态 + 操作方法 ----
  const [anchorSetting, setAnchorSetting] = useState('')   // 正在设为锚点的资产ID（''=无）
  const [anchorClearing, setAnchorClearing] = useState(false)

  // 设为锚点：一步式同步（后端取URL→多模态提取VAOCI→落库），成功后乐观更新 courseware
  const handleSetAnchor = async (assetId: string, notify?: (msg: string) => void) => {  // W1: notify把提示路由回MediaManagerPanel消息条
    if (!id || anchorSetting) return
    setAnchorSetting(assetId)
    notify?.('🎨 正在将该图设为风格锚点（AI读图提取风格DNA，约需数秒）...')
    try {
      const res = await setStyleAnchor(id, assetId)
      // 乐观更新：把锚点两字段写回 courseware，无需整页 reload
      setCourseware(prev => prev ? { ...prev, style_anchor_asset_id: res.asset_id, style_anchor_vaoci: res.vaoci, style_anchor_url: res.anchor_url } : prev)
      notify?.('✅ 已设为风格锚点！后续生成的配图将自动参考此图风格，保持全课件视觉统一')
    } catch (e) {
      notify?.('❌ 设置锚点失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setAnchorSetting('') }
  }

  // 清除锚点：成功后乐观更新 courseware 锚点字段为空
  const handleClearAnchor = async (notify?: (msg: string) => void) => {  // W1: notify同上, 顶部锚点条调用时不传(静默)
    if (!id || anchorClearing) return
    if (!confirm('确定清除风格锚点？清除后新生成的配图将不再自动套用此风格。')) return
    setAnchorClearing(true)
    try {
      await clearStyleAnchor(id)
      setCourseware(prev => prev ? { ...prev, style_anchor_asset_id: null, style_anchor_vaoci: '', style_anchor_url: '' } : prev)
      notify?.('✅ 已清除风格锚点')
    } catch (e) {
      notify?.('❌ 清除锚点失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setAnchorClearing(false) }
  }

  // 【本轮体验修复·关键】全自动装配画风弹窗设锚点成功后的「乐观更新」回调。
  //   ⚠ 绝不能在这里调 loadCourseware()——那会 setLoading(true) 让整页进入 loading 态，
  //     把正在显示的画风弹窗(AnchorStylePicker)连同 AutoAssemblyPanel 一起卸载重挂，
  //     导致"预览没看到就跳了 / 弹窗莫名消失 / 回到大页面重新点自动生成"三连坑。
  //   正确做法：拿画风弹窗回传的 SetStyleAnchorResult，用 setCourseware 只局部更新锚点三字段，
  //     整页不刷新、弹窗稳定停留，老师看清预览再确认，确认后一气呵成进入装配进度视图。
  const handleAnchorOptimistic = (res: SetStyleAnchorResult) => {
    setCourseware(prev => prev ? {
      ...prev,
      style_anchor_asset_id: res.asset_id,
      style_anchor_vaoci: res.vaoci,
      style_anchor_url: res.anchor_url,
    } : prev)
  }

  // 全工坊共享的单一SSE连接句柄（方案/封面/批量三场景轮流持有，卸载统一关闭）
  const sseRef = useRef<{ close: () => void } | null>(null)

  useEffect(() => { if (id) { loadCourseware(); reloadAnnotations() } return () => { sseRef.current?.close() } }, [id])

  const loadCourseware = useCallback(async () => {
    if (!id) return; setLoading(true)
    try {
      const d = await getCourseware(id); setCourseware(d); setPages(d.pages || [])
      const hasNav = !!(d.nav_template_html && d.nav_template_html.trim())
      // P0-1: 预览页只检查第1页（封面页）
      const hasPreview = (d.pages || []).some(p => p.html_content && p.page_number === 1)
      const targetStep = statusToStep(d.status, hasNav, hasPreview)
      // 修复"进入微调不跳转"：如果当前已在Step5（由goToStep(5)或onDone设置），
      // 且后端status对应的目标步骤也是5（preview/confirmed/in_pipeline），不回退。
      // 如果当前在Step5但后端status对应更早步骤（如generating竞态），也不回退——
      // 避免全自动装配onDone先goToStep(5)再loadCourseware异步拉回的竞态问题。
      setActiveStep(prev => {
        if (prev >= 5 && targetStep <= prev) return prev  // 已在Step5+不回退
        return targetStep
      })
      if (targetStep > maxStepReached) setMaxStepReached(targetStep)
      // 集体备课参与者：不走生成向导，直接锁定到 Step5 工作台（前提是课件已生成到可微调）
      if (user && d.user_id !== user.id && user.role !== 'admin') {
        setActiveStep(5)
        setMaxStepReached(5)
      }
      // 恢复已生成的页面数据
      const gp = (d.pages || []).filter(p => p.html_content).map(p => ({ id: p.id, page_number: p.page_number, title: p.title, html_content: p.html_content }))
      if (gp.length > 0) {
        const pp = gp.filter(p => p.page_number === 1)
        if (pp.length > 0) setPreviewPages(pp)
        setGeneratedPages(gp)
        if (buildPreviewNum === 0 && gp.length > 0) {
          // v0.42.13 中断态默认预览"最后一张已生成页"，避免误以为从第1页重做；全部完成才默认第1页便于从头审阅
          const cwAllDone = gp.length >= (d.pages || []).length
          setBuildPreviewNum(cwAllDone ? gp[0].page_number : gp[gp.length - 1].page_number)
        }
      }
    } catch { alert('加载课件失败'); navigate('/courseware') } finally { setLoading(false) }
  }, [id, navigate])

  // 字体F2b: 背景/字体秒换后的局部刷新——不整页loading重载, 保住选择器内的"已秒换N页"成功提示
  //   只重拉页面列表更新预览(秒换只改页面HTML, 不动步骤/状态), 失败静默(老师可手动刷新)
  const refreshPagesOnly = useCallback(async () => {
    if (!id) return
    try {
      const freshPages = await getCoursewarePages(id)
      setPages(freshPages)
      const gp = freshPages.filter(p => p.html_content && p.html_content.trim())
        .map(p => ({ id: p.id, page_number: p.page_number, title: p.title, html_content: p.html_content }))
        .slice().sort((a, b) => a.page_number - b.page_number)
      if (gp.length > 0) {
        setGeneratedPages(gp)
        const pp = gp.filter(p => p.page_number === 1)
        if (pp.length > 0) setPreviewPages(pp)
      }
    } catch { /* 局部刷新失败不打断流程 */ }
  }, [id])

  // v136: 通用步骤回退
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const handleRollback = async (targetStatus: string, targetStep: number) => {
    if (!id || rollingBack) return
    setRollingBack(true)
    try {
      await rollbackCWStatus(id, targetStatus)
      setActiveStep(targetStep)
      await loadCourseware()
    } catch (e) { alert('回退失败: ' + (e instanceof Error ? e.message : '未知错误')) }
    finally { setRollingBack(false) }
  }

  const handleStyleConfirmed = () => { goToStep(3); loadCourseware() }

  // Step 4: 批量生成剩余页
  const handleBuildStart = async () => {
    if (!id) return; setBuildRunning(true); setBuildMessage('正在启动...'); setBuildProgress({ current: 0, total: 0 })
    // v0.42.12 续传：先把预览跳到最后一张已生成页，避免误以为从第1页重做（生成新页后会自动往后跳）
    const cwLastDone = generatedPages.reduce((m, p) => Math.max(m, p.page_number), 0)
    if (cwLastDone > 0) setBuildPreviewNum(cwLastDone)
    try {
      await generateCWPages(id); sseRef.current?.close()
      sseRef.current = subscribeCWIndexSSE(id, {
        onConnected: () => setBuildMessage('已连接...'),
        onGenStart: d => {
          // 迭代二P1并发收尾：只记录本次需生成总数；不再把后端那句含单个 Px 的 message
          // 直接当进度展示（并发下 Px 会乱跳）。完成数改由 generatedPages 真实落库数计算。
          setBuildMessage('正在同时生成多页课件，请稍候…')
          setBuildProgress({ current: 0, total: d.total_pages })
        },
        onGenProgress: d => {
          // 并发下 d.current_page 是"最后到达事件的序号"，会忽大忽小，不能当完成数。
          // 这里只维持一句固定提示，真实进度（已完成/剩余）由下方按 generatedPages 实时算。
          void d
          setBuildMessage('正在同时生成多页课件，请稍候…')
        },
        onGenPage: d => {
          // 迭代二P1并发收尾：后端并发生成，gen_page 事件按"哪页先画完先到"顺序抵达，
          // 不再等于页码顺序。这里按 page_number 升序维护已生成列表，保证上方标签恒为 P1 P2 P3... 有序。
          setGeneratedPages(p => {
            const next = p.some(x => x.page_number === d.page_number)
              ? p.map(x => x.page_number === d.page_number ? { id: d.page_id, page_number: d.page_number, title: d.title, html_content: d.html_content } : x)
              : [...p, { id: d.page_id, page_number: d.page_number, title: d.title, html_content: d.html_content }]
            return next.slice().sort((a, b) => a.page_number - b.page_number)
          })
          // 预览框只单向前移到"已到达的最大页号"：既有"在推进"的反馈，又不会因乱序到达而来回乱跳
          setBuildPreviewNum(prev => d.page_number > prev ? d.page_number : prev)
        },
        // P2 需求②：完成后——全部成功才不动（停在批量生成页让老师自行点"进入微调→"）；
        //   有失败页则明确提示并【留在批量生成页】，老师可对失败页点"继续生成"自动补齐，
        //   不再自动跳到确认提交页（避免带着缺页进入下一步）。loadCourseware 会刷新真实状态与页面。
        onGenDone: d => {
          setBuildRunning(false)
          if (d.fail_count > 0) {
            setBuildMessage(`⚠️ ${d.message}。失败的页面可点下方"继续生成"自动重试补齐（成功页不会重做）。`)
            loadCourseware()
          } else {
            setBuildMessage(`✅ ${d.message}`)
            // 全部成功：先跳到Step5再loadCourseware，防止loadCourseware内statusToStep竞态拉回
            goToStep(5)
            loadCourseware()
          }
        },
        onError: d => { setBuildMessage(`❌ ${d.message}`); /* 单页错误不终止整批：不再 setBuildRunning(false)，等 gen_done 统一收尾 */ },
        // P2 需求（假死根治）：SSE 断线重连成功后，主动拉一次课件最新状态 + 页面列表，
        //   把断线期间漏收的已生成页补齐到 generatedPages，进度立刻对上，不再"假死"。
        onReconnected: async () => {
          setBuildMessage('🔄 连接已恢复，正在同步最新进度…')
          try {
            const fresh = await getCourseware(id)
            const freshPages = await getCoursewarePages(id)
            setCourseware(fresh)
            setPages(freshPages)
            const gp = freshPages
              .filter(p => p.html_content && p.html_content.trim())
              .map(p => ({ id: p.id, page_number: p.page_number, title: p.title, html_content: p.html_content }))
              .slice().sort((a, b) => a.page_number - b.page_number)
            setGeneratedPages(gp)
            // 若后端其实已完成（状态推进到 preview），则结束运行态
            if (fresh.status === 'preview' || fresh.status === 'confirmed') {
              setBuildRunning(false)
              setBuildMessage('✅ 已同步：课件生成已完成')
            } else {
              setBuildMessage('正在同时生成多页课件，请稍候…')
            }
          } catch {
            setBuildMessage('🔄 连接已恢复，但同步进度失败，可手动刷新页面查看')
          }
        },
      })
    } catch { setBuildMessage('❌ 启动失败'); setBuildRunning(false) }
  }
  useEffect(() => {
    if (!buildRunning || !id) return
    const t = setInterval(async () => { try { const d = await getCourseware(id); if (d.status === 'preview') { setBuildRunning(false); setCourseware(d); setPages(d.pages || []); goToStep(5); setBuildMessage('✅ 完成'); const gp = d.pages.filter(p => p.html_content).map(p => ({ id: p.id, page_number: p.page_number, title: p.title, html_content: p.html_content })); setGeneratedPages(gp); /* v0.42.13: 不再重置预览到第1页，保留当前/最后生成页 */ sseRef.current?.close() } } catch {} }, 15000)
    return () => clearInterval(t)
  }, [buildRunning, id])

  // v0.42.12 中途停止批量生成——已生成页面保留，状态仍为 generating，可在本步骤点击「继续生成」续传
  const handleCancelBuild = async () => {
    if (!id) return
    try {
      await cancelGenerate(id)
      setBuildMessage('⏸ 已发送停止信号，正在结束当前页（已生成页面已保留）...')
    } catch (e) {
      setBuildMessage('❌ 停止失败: ' + (e instanceof Error ? e.message : '未知错误'))
    }
  }

  // P1-02解读A: 开放映时——用传入页 / 列表选中页(buildPreviewNum) 派生初值喂给子组件
  const openSlideshow = (pn?: number) => {
    const allPages = generatedPages.length > 0 ? generatedPages : previewPages
    setSlideshowInitPage(pn || buildPreviewNum || allPages[0]?.page_number || 1)
    setSlideshowOpen(true)
  }

  // v137: 全屏预览入口（PagePreviewBlock 的「🔍 全屏预览」按钮回调）
  //   P1-02解读A: pn 来自列表选中页，作为开窗初值喂给子组件
  const handleFullscreen = (pn: number) => {
    setFullscreenPageNum(pn); setFullscreenOpen(true); setFullscreenCodeView(false)
  }

  // P1-02解读A: 全屏预览退出——把子组件回传的最终页写进 buildPreviewNum（唯一真相源），
  //   于是列表高亮/页预览/下次开窗初值全部同步到该页；同时同步 fullscreenPageNum 保持初值一致。
  const handleFullscreenClose = (finalPage?: number) => {
    if (typeof finalPage === 'number') {
      setBuildPreviewNum(finalPage)
      setFullscreenPageNum(finalPage)
    }
    setFullscreenOpen(false)
  }

  // P1-02解读A: 放映退出——同上，回传最终页写进 buildPreviewNum 唯一真相源
  const handleSlideshowClose = (finalPage?: number) => {
    if (typeof finalPage === 'number') {
      setBuildPreviewNum(finalPage)
      setSlideshowInitPage(finalPage)
    }
    setSlideshowOpen(false)
  }

  if (loading) return <div style={{ textAlign: 'center', padding: '80px 0', color: C.textMuted }}><div style={{ fontSize: 40, marginBottom: 12 }}>🎨</div>加载中...</div>
  if (!courseware) return <div style={{ textAlign: 'center', padding: '80px 0', color: C.textMuted }}>课件不存在<br/><button onClick={() => navigate('/courseware')} style={{ marginTop: 12, color: C.primary, background: 'none', border: 'none', cursor: 'pointer' }}>返回列表</button></div>

  // === v0.42.11: 3D 互动单页分支早返回 ===
  // 当课件来源是 3d_single 时跳过标准六步流程，进入简化版 3D 工坊视图
  // 该视图独立管理课件刷新+SSE+生成+预览+确认，不依赖主组件的 activeStep 状态机
  if (courseware.source_type === '3d_single') {
    return <ThreeDSingleView initialCourseware={courseware} />
  }

  const sc = CW_STATUS_CONFIG[courseware.status] || { label: courseware.status, color: '#6B7280', bg: '#F3F4F6' }

  const allSlideshowPages = generatedPages.length > 0 ? generatedPages : previewPages
  // v0.42.12 续生成统计：基于完整页面列表 pages 计算已生成/剩余页数
  const cwTotalCount = pages.length
  const cwDoneCount = pages.filter(p => p.html_content && p.html_content.trim()).length
  const cwRemainingCount = Math.max(0, cwTotalCount - cwDoneCount)
  // 中断态：封面之外已生成部分页，但仍有剩余未生成（说明上次生成被中断）
  const cwInterrupted = cwDoneCount > 1 && cwRemainingCount > 0
  // 迭代二P1并发收尾：批量生成进度的"真实口径"派生值（替代会乱跳的 buildProgress.current）
  //   batchTotal —— 本次需生成的页数（后端 onGenStart 下发的 total；为 0 时回退用剩余页数）
  //   batchDone  —— 本次已真实落库完成的页数 = 当前已落库总数 - 开跑前就已完成的数，夹在 [0, batchTotal]
  //     该数源自有序的 generatedPages（已落库才计数），单调递增、永不回跳、与上方标签数一致
  const batchTotal = buildProgress.total > 0 ? buildProgress.total : cwRemainingCount
  // 本次已完成数直接取 generatedPages 实时长度（SSE 增量更新，与左下"已生成 N 页"同源），
  // cap 在 [0, batchTotal] 内防越界。不再用 cwDoneCount 推算（那是 loadCourseware 快照，生成中不更新会恒 0）。
  const batchDone = Math.max(0, Math.min(batchTotal, generatedPages.length))
  const batchRemaining = Math.max(0, batchTotal - batchDone)
  const batchPercent = batchTotal > 0 ? Math.round((batchDone / batchTotal) * 100) : 0

  return (
    <div style={{ maxWidth: 960, margin: '0 auto' }}>
      {/* 顶部 */}
      <div style={{ marginBottom: 24 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 8 }}>
          <button onClick={() => navigate('/courseware')} style={{ background: 'none', border: 'none', fontSize: 14, color: C.textSecondary, cursor: 'pointer' }}>← 返回列表</button>
          <span style={{ padding: '2px 10px', borderRadius: 12, fontSize: 12, fontWeight: 500, color: sc.color, background: sc.bg }}>{sc.label}</span>
        </div>
        <h2 style={{ fontSize: 22, fontWeight: 700, color: C.textPrimary, margin: 0 }}>{courseware.title}</h2>
        <div style={{ fontSize: 13, color: C.textMuted, marginTop: 4 }}>{courseware.source_type === 'topic_direct' ? '💡 主题创建' : courseware.source_type === 'ppt_upload' ? '📊 PPT上传' : courseware.source_type === 'doc_upload' ? '📄 文档上传' : ('📝 ' + (courseware.lesson_plan_title || '未知'))} &nbsp;|&nbsp; 📚 {courseware.subject} &nbsp;|&nbsp; 🎓 {courseware.grade}</div>
        {/* 阶段4：集体备课参与者提示条——告知这是别人的课件、你来一起微调 */}
        {isParticipant && (
          <div style={{ marginTop: 10, padding: '8px 14px', borderRadius: 10, background: '#f6ffed', border: '1px solid #b7eb8f', fontSize: 13, color: '#10893e', lineHeight: 1.6 }}>
            👥 你正在参与本课件的<b>集体备课</b>，可在下方工作台微调页面、添加批注。课件的生成、风格、发布等由作者管理。
          </div>
        )}
        {/* 风格锚点展示条（轮3）：已设锚点时显示缩略图+VAOCI摘要+清除；未设则不显示 */}
        {courseware.style_anchor_asset_id && !isParticipant && (
          <div style={{ marginTop: 10, display: 'flex', alignItems: 'center', gap: 10, padding: '8px 12px', borderRadius: 10, background: 'rgba(124,58,237,0.06)', border: '1px solid rgba(124,58,237,0.25)' }}>
            <span style={{ fontSize: 13, fontWeight: 600, color: '#7C3AED', whiteSpace: 'nowrap' }}>⭐ 风格锚点</span>
            <span style={{ flex: 1, fontSize: 12, color: C.textSecondary, lineHeight: 1.5, overflow: 'hidden', textOverflow: 'ellipsis', display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' }} title={courseware.style_anchor_vaoci}>
              {courseware.style_anchor_vaoci ? courseware.style_anchor_vaoci : '（已设锚点，后续配图自动套用此风格）'}
            </span>
            <button onClick={() => handleClearAnchor()} disabled={anchorClearing}
              style={{ padding: '4px 12px', borderRadius: 6, border: '1px solid ' + C.danger, background: 'transparent', color: C.danger, fontSize: 12, cursor: anchorClearing ? 'default' : 'pointer', whiteSpace: 'nowrap' }}>
              {anchorClearing ? '清除中...' : '✕ 清除锚点'}
            </button>
          </div>
        )}
      </div>

      {/* 步骤条（参与者只在 Step5 工作台微调，不显示生成向导步骤条） */}
      {!isParticipant && (
      <div style={{ display: 'flex', gap: 4, marginBottom: 28, padding: '16px 20px', background: C.white, borderRadius: 12, border: `1px solid ${C.border}` }}>
        {STEPS.map((s, i) => {
          const active = i === activeStep, done = i < activeStep, reached = i <= maxStepReached
          return <div key={s.key} onClick={() => { if (reached && !active) goToStep(i) }} style={{ flex: 1, textAlign: 'center', cursor: (reached && !active) ? 'pointer' : 'default' }}>
            <div style={{ width: 32, height: 32, borderRadius: '50%', margin: '0 auto 6px', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 16, background: done ? C.success : active ? C.primary : reached ? '#A7F3D0' : '#F3F4F6', color: done || active ? '#fff' : C.textMuted, fontWeight: 700, transition: 'all 300ms' }}>{done ? '✓' : s.emoji}</div>
            <div style={{ fontSize: 11, fontWeight: active ? 600 : 400, color: active ? C.primary : done ? C.success : C.textMuted }}>{s.label}</div>
          </div>
        })}
      </div>
      )}

      {/* 内容区 */}
      <div style={{ background: C.white, borderRadius: 12, border: `1px solid ${C.border}`, padding: 24, minHeight: 400 }}>

        {/* Step 0+1: 生成方案+确认方案（批次5b-2拆出为 SchemeSteps） */}
        {(activeStep === 0 || activeStep === 1) && (
          <SchemeSteps
            coursewareId={id!}
            courseware={courseware}
            isAdmin={isAdmin}
            activeStep={activeStep}
            pages={pages}
            setPages={setPages}
            sseRef={sseRef}
            goToStep={goToStep}
            loadCourseware={loadCourseware}
            onCoursewareUpdate={setCourseware}
          />
        )}

        {/* Step 2: 选择风格 */}
        {activeStep === 2 && courseware && <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <div><h3 style={{ fontSize: 18, fontWeight: 600, color: C.textPrimary, margin: 0 }}>🎨 课件风格定制</h3><p style={{ fontSize: 13, color: C.textSecondary, margin: '4px 0 0' }}>选择视觉风格，配置机构品牌</p></div>
            <button onClick={() => goToStep(1)} style={{ padding: '8px 16px', borderRadius: 8, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 13, cursor: 'pointer' }}>← 返回编辑</button>
          </div>
          <StyleSelector courseware={courseware} coursewareId={id!} onStyleConfirmed={handleStyleConfirmed} onAnchorChanged={loadCourseware} />
        </div>}

        {/* Step 3: 确认导航栏（批次5b-2拆出为 NavConfirmStep） */}
        {activeStep === 3 && (
          <NavConfirmStep
            coursewareId={id!}
            courseware={courseware}
            previewPages={previewPages}
            setPreviewPages={setPreviewPages}
            buildRunning={buildRunning}
            sseRef={sseRef}
            goToStep={goToStep}
            loadCourseware={loadCourseware}
            refreshPagesOnly={refreshPagesOnly}
            onSlideshow={openSlideshow}
            onFullscreen={handleFullscreen}
          />
        )}

        {/* Step 4: 批量生成剩余页 */}
        {activeStep === 4 && <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <div><h3 style={{ fontSize: 18, fontWeight: 600, color: C.textPrimary, margin: 0 }}>⚡ 批量生成课件</h3><p style={{ fontSize: 13, color: C.textSecondary, margin: '4px 0 0' }}>使用已确认的导航栏样式，逐页生成剩余课件</p></div>
            {!buildRunning && <button onClick={() => goToStep(3)} style={{ padding: '8px 16px', borderRadius: 8, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 13, cursor: 'pointer' }}>← 返回确认导航栏</button>}
          </div>
          <MsgBar msg={buildMessage} />
          {batchTotal > 0 && <div style={{ marginBottom: 20 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, color: C.textSecondary, marginBottom: 6 }}>
              <span>{cwInterrupted ? '续传进度（已完成页不会重做）' : '生成进度（多页同时进行）'}</span>
              <span>本次需生成 {batchTotal} 页 · 已完成 <b style={{ color: C.success }}>{batchDone}</b> 页 · 剩余 {batchRemaining} 页</span>
            </div>
            <div style={{ height: 8, borderRadius: 4, background: '#F3F4F6', overflow: 'hidden' }}><div style={{ height: '100%', borderRadius: 4, transition: 'width 500ms', width: `${batchPercent}%`, background: 'linear-gradient(90deg, #F59E0B, #EF4444)' }} /></div>
          </div>}
          <PagePreviewBlock pages={generatedPages} currentNum={buildPreviewNum} onSelectPage={setBuildPreviewNum} showSlideshow onSlideshow={openSlideshow} onFullscreen={handleFullscreen} />
          {/* v0.42.12 续生成提示条：检测到上次生成被中断（已生成部分页但仍有剩余） */}
          {!buildRunning && cwInterrupted && (
            <div style={{ padding: '12px 16px', borderRadius: 8, marginBottom: 14, background: '#FEF3C7', color: '#92400E', fontSize: 14, lineHeight: 1.6 }}>
              ⚠️ 检测到上次生成被中断，已完成 <b>{cwDoneCount}/{cwTotalCount}</b> 页。点击「继续生成」会<b>跳过已生成的页面</b>，仅生成剩余 <b>{cwRemainingCount}</b> 页，无需从头再来。
            </div>
          )}
          {/* 场景A：尚未开始批量生成（仅封面或全空，仍有剩余页待生成）——先选交付模式(三档)再分流 */}
          {!buildRunning && cwDoneCount <= 1 && cwRemainingCount > 0 && (
            <div>
              {/* A-0 尚未选交付模式：显示三档选择器（未设风格锚点时后两档禁用） */}
              {deliveryMode === null && (
                <DeliveryModeSelect
                  hasAnchor={!!courseware.style_anchor_asset_id}
                  remainingCount={cwRemainingCount}
                  onSelect={setDeliveryMode}
                />
              )}
              {/* A-1 纯手动：走现有批量生成逻辑（一行未改，仅包了层选择态） */}
              {deliveryMode === 'manual' && (
                <div>
                  <div style={{ marginBottom: 14, display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                    <span style={{ fontSize: 14, fontWeight: 600, color: '#059669' }}>✋ 纯手动生成</span>
                    <button onClick={() => setDeliveryMode(null)} style={{ padding: '4px 12px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer' }}>← 重选交付模式</button>
                  </div>
                  <button onClick={handleBuildStart} style={{ padding: '14px 36px', borderRadius: 10, border: 'none', background: 'linear-gradient(135deg, #F59E0B, #EF4444)', color: '#fff', fontSize: 16, fontWeight: 600, cursor: 'pointer', boxShadow: '0 4px 16px rgba(245,158,11,0.3)' }}>⚡ 开始批量生成剩余页面</button>
                </div>
              )}
              {/* A-2 全自动/中间档：走自包含装配面板（自持SSE+二次确认+进度视图） */}
              {(deliveryMode === 'no_video' || deliveryMode === 'full') && (
                <AutoAssemblyPanel
                  coursewareId={id!}
                  courseware={courseware}
                  skipVideo={deliveryMode === 'no_video'}
                  onDone={() => { goToStep(5); loadCourseware() }}
                  onAnchorChanged={handleAnchorOptimistic}
                  onBack={() => setDeliveryMode(null)}
                />
              )}
            </div>
          )}
          {/* 场景B：已生成部分页且仍有剩余（中断续传——跳过已生成页只补剩余） */}
          {!buildRunning && cwInterrupted && <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            <button onClick={handleBuildStart} style={{ padding: '12px 28px', borderRadius: 8, border: 'none', background: 'linear-gradient(135deg, #059669, #10B981)', color: '#fff', fontSize: 15, fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 8px rgba(5,150,105,0.3)' }}>▶️ 继续生成剩余 {cwRemainingCount} 页</button>
            <button onClick={() => openSlideshow()} style={{ padding: '12px 24px', borderRadius: 8, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 14, fontWeight: 600, cursor: 'pointer' }}>🖥️ 预览已生成页</button>
          </div>}
          {/* 场景C：全部页面已生成完毕 */}
          {!buildRunning && cwRemainingCount === 0 && cwTotalCount > 0 && <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            <button onClick={() => openSlideshow()} style={{ padding: '10px 24px', borderRadius: 8, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 14, fontWeight: 600, cursor: 'pointer' }}>🖥️ 全屏放映</button>
            <button onClick={() => { goToStep(5); loadCourseware() }} style={{ padding: '10px 24px', borderRadius: 8, border: 'none', background: 'linear-gradient(135deg, #059669, #10B981)', color: '#fff', fontSize: 14, fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 8px rgba(5,150,105,0.3)' }}>进入微调 →</button>
          </div>}
          {/* 生成进行中：显示进度 + 可中途停止（已生成页面会保留，可稍后继续） */}
          {buildRunning && <div style={{ textAlign: 'center', padding: 20, color: C.textMuted, fontSize: 14 }}>
            <div style={{ fontSize: 32, marginBottom: 8 }}>⚡</div>
            {cwInterrupted ? `续传中：第 1–${cwDoneCount} 页已完成、不会重做，正在生成剩余页…` : 'AI正在逐页生成，请耐心等待...'}
            <div style={{ marginTop: 14 }}>
              <button onClick={handleCancelBuild} style={{ padding: '8px 20px', borderRadius: 8, border: '1px solid #EF4444', background: 'rgba(239,68,68,0.06)', color: '#EF4444', fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>⏸ 停止生成（已生成页面会保留，可稍后继续）</button>
            </div>
          </div>}
        </div>}

        {/* Step 5: 确认提交 */}
        {activeStep >= 5 && <div>
          {/* W3: 紧凑页头——撤掉大图标仪式区, 把首屏垂直空间还给预览主体 */}
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12, flexWrap: 'wrap', gap: 8 }}>
            <div style={{ fontSize: 16, fontWeight: 700, color: C.textPrimary }}>✅ 课件预览与确认
              <span style={{ fontSize: 13, fontWeight: 400, color: C.textMuted, marginLeft: 8 }}>共 {generatedPages.length} 页</span>
            </div>
            <button onClick={() => goToStep(4)} style={{ padding: '8px 16px', borderRadius: 8, border: '1px solid ' + C.border, background: 'transparent', color: C.textSecondary, fontSize: 13, cursor: 'pointer' }}>← 返回重新生成</button>
          </div>
          <PagePreviewBlock pages={generatedPages} currentNum={buildPreviewNum} onSelectPage={setBuildPreviewNum} showSlideshow onSlideshow={openSlideshow} onFullscreen={handleFullscreen} editable={!buildRunning} coursewareId={id} onPagesChanged={refreshPagesOnly} />

          {/* 批次W2: Step5工作台Tab——预览区下方收纳全部工具, 页面高度恒定; 默认「页面微调」(最高频)
              批次1a: 笔顺/公式/五线谱收编为「🧪学科工具」聚合Tab, 12→10 */}
          {generatedPages.length > 0 && <>
            <div style={{ display: 'flex', gap: 8, marginTop: 20, flexWrap: 'wrap', borderBottom: '2px solid ' + C.border, paddingBottom: 12 }}>
              {(([['refine', '🛠 页面微调'], ['background', '🎨 背景'], ['font', '🔤 字体'], ['image', '🖼 图片'], ['video', '🎬 视频'], ['audio', '🎵 音频'], ['subject', '🧪 学科工具'], ['template', '💾 保存模板'], ['annotation', '💬 批注'], ['collab', '👥 集体备课']] as const)
                // 参与者从严：只留"改页面内容"的 tab，隐藏背景/字体/保存模板（课件级配置，作者专属）
                .filter(([k]) => !isParticipant || (['refine', 'image', 'video', 'audio', 'subject', 'annotation', 'collab'] as string[]).includes(k))
              ).map(([k, label]) => (
                <button key={k} onClick={() => setWsTab(k)}
                  style={{ padding: '10px 22px', borderRadius: 10, cursor: 'pointer',
                    border: '2px solid ' + (wsTab === k ? C.primary : C.border),
                    background: wsTab === k ? C.primaryBg : '#fff',
                    color: wsTab === k ? C.primary : C.textSecondary,
                    fontSize: 14, fontWeight: wsTab === k ? 600 : 400, transition: 'all 200ms', position: 'relative' }}>
                  {label}
                  {/* 集体备课进行中红点：collab tab 且课件 in_session 时,提示"正在进行" */}
                  {k === 'collab' && activeCollab && (
                    <span style={{ display: 'inline-block', width: 8, height: 8, borderRadius: '50%', background: '#10893e', marginLeft: 6, verticalAlign: 'middle' }} />
                  )}
                </button>
              ))}
            </div>
            {wsTab === 'refine' && activeCollab && (
              <div style={{ margin: '14px 0 0', padding: '8px 14px', borderRadius: 8, background: '#f6ffed', border: '1px solid #b7eb8f', fontSize: 13, color: '#10893e' }}>
                🟢 本课件正在集体备课中，你的修改会与其他参与老师共同作用于同一课件。
              </div>
            )}
            {wsTab === 'refine' && (
              <RefinePanel coursewareId={id!} pageNum={buildPreviewNum}
                onPageUpdated={(pn, html) => setGeneratedPages(prev => prev.map(p => p.page_number === pn ? { ...p, html_content: html } : p))} />
            )}
            {wsTab === 'background' && (
              <AppearancePanel mode="background" coursewareId={id!} onSwapped={refreshPagesOnly} disabled={buildRunning}
                pageNum={buildPreviewNum > 0 ? buildPreviewNum : undefined}
                cwTitle={courseware.title} cwSubject={courseware.subject} cwGrade={courseware.grade} />
            )}
            {wsTab === 'font' && (
              <AppearancePanel mode="font" coursewareId={id!} onSwapped={refreshPagesOnly} disabled={buildRunning} />
            )}
            {(wsTab === 'image' || wsTab === 'video' || wsTab === 'audio') && (
              <MediaManagerPanel coursewareId={id!} pageNum={buildPreviewNum} courseware={courseware} mediaTab={wsTab}
                anchorSetting={anchorSetting} anchorClearing={anchorClearing}
                onSetAnchor={handleSetAnchor} onClearAnchor={handleClearAnchor}
                onPageUpdated={(pn, html) => setGeneratedPages(prev => prev.map(p => p.page_number === pn ? { ...p, html_content: html } : p))} />
            )}
            {/* 批次1a: 学科工具聚合面板（笔顺/公式/五线谱 + 后续数学/分子/物理宫格入驻位） */}
            {wsTab === 'subject' && (
              <Suspense
                fallback={
                  <div
                    style={{
                      marginTop: 16,
                      padding: '32px 20px',
                      borderRadius: 14,
                      border: `1px solid ${C.border}`,
                      background: '#FAFBFF',
                      color: C.textMuted,
                      fontSize: 13,
                      textAlign: 'center',
                    }}
                  >
                    🧪 学科工具加载中...
                  </div>
                }
              >
                <LazySubjectToolsPanel
                  coursewareId={id!}
                  pageNum={buildPreviewNum}
                  onPageUpdated={(pn, html) =>
                    setGeneratedPages(prev =>
                      prev.map(p =>
                        p.page_number === pn
                          ? { ...p, html_content: html }
                          : p,
                      ),
                    )
                  }
                />
              </Suspense>
            )}
            {wsTab === 'template' && <TemplateSavePanel coursewareId={id!} />}
            {wsTab === 'annotation' && (
              <AnnotationPanel coursewareId={id!} pageNumber={buildPreviewNum} annotations={cwAnnotations} onChanged={reloadAnnotations} />
            )}
            {wsTab === 'collab' && courseware && user && (
              <CollabPanel
                coursewareId={id!}
                ownerId={courseware.user_id}
                currentUserId={user.id}
                onChanged={() => { getCourseware(id!).then(setCourseware).catch(() => {}) }}
              />
            )}
          </>}
        </div>}
      </div>

      {/* v137: 全屏预览（带工具栏+键盘导航+resize响应）
          P1-02解读A: onClose 退出时把当前页回写进 buildPreviewNum 唯一真相源，列表/预览/下次开窗全同步 */}
      {fullscreenOpen && allSlideshowPages.length > 0 && <CWFullscreenPreview
        pages={allSlideshowPages}
        initialPageNum={fullscreenPageNum}
        codeView={fullscreenCodeView}
        onToggleCode={() => setFullscreenCodeView(!fullscreenCodeView)}
        onClose={handleFullscreenClose}
        onSlideshow={(pn) => { setBuildPreviewNum(pn); setFullscreenPageNum(pn); setFullscreenOpen(false); setSlideshowInitPage(pn); setSlideshowOpen(true) }}
      />}

      {/* P1-02解读A: onClose 退出时把当前页回写进 buildPreviewNum 唯一真相源 */}
      {slideshowOpen && allSlideshowPages.length > 0 && <SlideshowPlayer pages={allSlideshowPages} initialPage={slideshowInitPage} onClose={handleSlideshowClose} />}

      {/* 教案对照抽屉——Step1(确认方案)+Step4(批量生成)+Step5(工作台)均可展开对照教案原文
          抽屉本身懒加载(首次点开才请求)+has_lesson_plan=false时自隐藏(非教案来源零回归) */}
      {activeStep >= 1 && <LessonPlanRefDrawer coursewareId={id!} />}

    </div>
  )
}

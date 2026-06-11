/**
 * 课件工坊主页面 — CoursewareWorkshopPage.tsx
 *
 * 六步向导主页面（生成方案→确认方案→选风格→确认导航栏→批量生成→确认提交）。
 *
 * 【模块化拆分】本文件原 1877 行，已将以下无状态/独立部分抽到 components/courseware-workshop/：
 *   - workshopConstants.ts : C(配色) / CW_WIDTH / CW_HEIGHT / CW_IMG_STYLES(图片风格快选) /
 *                            STEPS(六步定义) / statusToStep(status→step 映射)
 *   - previewInject.ts     : PREVIEW_INJECT_SCRIPT / injectPreviewMode(iframe 预览降级注入)
 *   - CWFullscreenPreview.tsx : 带工具栏的全屏预览组件
 *   - SlideshowPlayer.tsx     : 全屏幻灯片放映组件
 * 主组件逻辑与交互完全不变，仅改为 import 上述模块。
 *
 * 关键设计（保留供理解）：
 *   - buildPreviewNum 是 Step5 唯一选中页真相源；多媒体管理已整体迁入
 *     MediaManagerPanel.tsx（批次W1），以 pageNum 接收选中页，内部自动拉取本页媒体防串页。
 *   - source_type==='3d_single' 时早返回走 ThreeDSingleView，不走标准六步。
 *   - 图片配图建议/批量生成/视频编辑等逻辑见 MediaManagerPanel.tsx。
 *   - 单页微调/重生已迁入 RefinePanel.tsx（批次W2），保存模板迁入 TemplateSavePanel.tsx。
 *   - 批量生成支持中断续传（跳过已生成页）。
 *   - 「☁️云盘」上传成功写 public_oss_url；删除已上云资产弹强警告确认。
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  getCourseware, getCoursewarePages, generateCWIndex, generateCWIndexFromTopic, subscribeCWIndexSSE,
  confirmCWIndex, generateCWPreview, saveCWNavTemplate,
  generateCWPages, CW_STATUS_CONFIG, refineNav, refinePage, cancelGenerate,
  generateCWIndexFromPPT,
  generateCWIndexFromDoc,
  rollbackCWStatus, refineCWIndex, getSchemePresets, saveAsMyTemplate,
  generateCWImage, uploadCWImage, listPageAssets, deleteCWAsset, insertImageToPage,
  advancedConcatCWVideos, uploadCWVideo, uploadAssetToOSS, regenerateCWPage,
  suggestImagePrompt, getStoredImageSuggestions,
  setStyleAnchor, clearStyleAnchor,
} from '@/api/coursewares'
import type { SchemePreset, ImagePromptSuggestion } from '@/api/coursewares'
import type { CoursewareDetail, CoursewarePage } from '@/api/coursewares'
import IndexEditor from './components/IndexEditor'
import StyleSelector from './components/StyleSelector'
import { useAuth } from '@/store/auth'
import VideoEditorModal from './components/VideoEditorModal'
import ThreeDSingleView from './components/ThreeDSingleView'
// 模块化拆分：常量/注入/两个预览组件抽到 courseware-workshop 子目录
import { C, CW_WIDTH, CW_HEIGHT, CW_IMG_STYLES, STEPS, statusToStep } from './components/courseware-workshop/workshopConstants'
import { injectPreviewMode } from './components/courseware-workshop/previewInject'
import CWFullscreenPreview from './components/courseware-workshop/CWFullscreenPreview'
import SlideshowPlayer from './components/courseware-workshop/SlideshowPlayer'
import VideoStoryboardPanel from './components/VideoStoryboardPanel'
import BackgroundPicker from './components/courseware-workshop/BackgroundPicker'
import FontPicker from './components/courseware-workshop/FontPicker'
import MediaManagerPanel from './components/courseware-workshop/MediaManagerPanel'
import RefinePanel from './components/courseware-workshop/RefinePanel'
import AppearancePanel from './components/courseware-workshop/AppearancePanel'
import TemplateSavePanel from './components/courseware-workshop/TemplateSavePanel'

// ==================== 主组件 ====================
export default function CoursewareWorkshopPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const [courseware, setCourseware] = useState<CoursewareDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [activeStep, setActiveStep] = useState(0)
  const [maxStepReached, setMaxStepReached] = useState(0)

  // v136: 跳转步骤并追踪最远到达
  const goToStep = (step: number) => {
    setActiveStep(step)
    setMaxStepReached(prev => Math.max(prev, step))
  }
  const [pages, setPages] = useState<CoursewarePage[]>([])
  const [generating, setGenerating] = useState(false)
  const [sseMessage, setSseMessage] = useState('')
  const [confirming, setConfirming] = useState(false)

  // Step 3: 预览生成状态（P0-1: 只有1页封面预览）
  const [previewGenRunning, setPreviewGenRunning] = useState(false)
  const [previewGenMessage, setPreviewGenMessage] = useState('')
  const [previewPages, setPreviewPages] = useState<{ page_number: number; title: string; html_content: string }[]>([])
  const [navSaving, setNavSaving] = useState(false)

  // P0-2: 导航栏微调状态
  const [navRefineInput, setNavRefineInput] = useState('')
  const [navRefining, setNavRefining] = useState(false)

  // Step 4: 批量生成状态
  const [buildRunning, setBuildRunning] = useState(false)
  const [buildMessage, setBuildMessage] = useState('')
  const [buildProgress, setBuildProgress] = useState({ current: 0, total: 0 })
  const [generatedPages, setGeneratedPages] = useState<{ page_number: number; title: string; html_content: string }[]>([])
  const [buildPreviewNum, setBuildPreviewNum] = useState(0)

  // 批次W2: 页面微调状态/逻辑已迁入 RefinePanel.tsx; 模板保存已迁入 TemplateSavePanel.tsx

  // v136: 方案预设+AI修改方案+回退
  const [presets, setPresets] = useState<SchemePreset[]>([])
  const [selectedPreset, setSelectedPreset] = useState('auto')
  const [refineFeedback, setRefineFeedback] = useState('')
  const [refining, setRefining] = useState(false)
  const [rollingBack, setRollingBack] = useState(false)

  // v137: 源代码查看状态
  const [codeViewPageNum, setCodeViewPageNum] = useState(0)
  // v137: 全屏预览状态（带工具栏，非放映模式）
  const [fullscreenOpen, setFullscreenOpen] = useState(false)
  const [fullscreenPageNum, setFullscreenPageNum] = useState(1)
  const [fullscreenCodeView, setFullscreenCodeView] = useState(false)


  const [slideshowOpen, setSlideshowOpen] = useState(false)
  const [slideshowInitPage, setSlideshowInitPage] = useState(1)
  // 批次W1: 多媒体管理的全部状态/effect/处理函数已整体迁入 MediaManagerPanel.tsx
  // 批次W2: Step5工作台Tab(默认「页面微调」=最高频动作; 图片/视频两Tab共用MediaManagerPanel实例)
  const [wsTab, setWsTab] = useState<'refine' | 'background' | 'font' | 'image' | 'video' | 'template'>('refine')  // W3: 背景/字体独立Tab

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

  const sseRef = useRef<{ close: () => void } | null>(null)


  



  useEffect(() => { if (id) loadCourseware(); return () => { sseRef.current?.close() } }, [id])

  // v136: 加载方案预设
  useEffect(() => {
    getSchemePresets().then(p => setPresets(p)).catch(() => {})
  }, [])

  const loadCourseware = useCallback(async () => {
    if (!id) return; setLoading(true)
    try {
      const d = await getCourseware(id); setCourseware(d); setPages(d.pages || [])
      const hasNav = !!(d.nav_template_html && d.nav_template_html.trim())
      // P0-1: 预览页只检查第1页（封面页）
      const hasPreview = (d.pages || []).some(p => p.html_content && p.page_number === 1)
      goToStep(statusToStep(d.status, hasNav, hasPreview))
      // 恢复已生成的页面数据
      const gp = (d.pages || []).filter(p => p.html_content).map(p => ({ page_number: p.page_number, title: p.title, html_content: p.html_content }))
      if (gp.length > 0) {
        const pp = gp.filter(p => p.page_number === 1)
        if (pp.length > 0) setPreviewPages(pp)
        setGeneratedPages(gp)
        if (buildPreviewNum === 0 && gp.length > 0) {
          // v0.42.13 中断态默认预览“最后一张已生成页”，避免误以为从第1页重做；全部完成才默认第1页便于从头审阅
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
        .map(p => ({ page_number: p.page_number, title: p.title, html_content: p.html_content }))
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

  // v136: AI修改方案
  const handleRefineIndex = async () => {
    if (!id || !refineFeedback.trim() || refining) return
    setRefining(true); setSseMessage('正在根据意见修改方案...')
    try {
      await refineCWIndex(id, refineFeedback.trim())
      sseRef.current?.close()
      sseRef.current = subscribeCWIndexSSE(id, {
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

  // Step 0: 生成方案
  const handleGenerate = async () => {
    if (!id) return; setGenerating(true); setSseMessage('正在启动...'); setPages([])
    try {
      if (courseware?.source_type === 'topic_direct') {
        await generateCWIndexFromTopic(id, {
          subject: courseware.subject,
          grade: courseware.grade,
          topic: courseware.title,
          preset: selectedPreset,
        })
      } else if (courseware?.source_type === 'ppt_upload') {
        await generateCWIndexFromPPT(id, selectedPreset)
      } else if (courseware?.source_type === 'doc_upload') {
        await generateCWIndexFromDoc(id, selectedPreset)
      } else {
        await generateCWIndex(id, selectedPreset)
      }
      sseRef.current?.close()
      sseRef.current = subscribeCWIndexSSE(id, {
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
  useEffect(() => {
    if (!generating || !id) return
    const t = setInterval(async () => { try { const d = await getCourseware(id); if (d.status !== 'draft' && d.status !== 'indexing') { setGenerating(false); setCourseware(d); setPages(d.pages || []); goToStep(1); setSseMessage('✅ 完成'); sseRef.current?.close() } } catch {} }, 10000)
    return () => clearInterval(t)
  }, [generating, id])

  const handleConfirm = async () => {
    if (!id || !pages.length) return; setConfirming(true)
    try { await confirmCWIndex(id); goToStep(2); loadCourseware() } catch { alert('确认失败') } finally { setConfirming(false) }
  }
  const handleStyleConfirmed = () => { goToStep(3); loadCourseware() }

  // Step 3: 生成预览页（P0-1: 仅封面1页）
  const handleGenPreview = async () => {
    if (!id) return; setPreviewGenRunning(true); setPreviewGenMessage('正在启动...'); setPreviewPages([])
    try {
      await generateCWPreview(id); sseRef.current?.close()
      sseRef.current = subscribeCWIndexSSE(id, {
        onConnected: () => setPreviewGenMessage('已连接...'),
        onGenStart: d => setPreviewGenMessage(d.message),
        onGenProgress: d => setPreviewGenMessage(d.message),
        onGenPage: d => { setPreviewPages(p => [...p, { page_number: d.page_number, title: d.title, html_content: d.html_content }]) },
        onGenDone: d => { setPreviewGenRunning(false); if (d.fail_count > 0) { setPreviewGenMessage(`❌ ${d.message}`) } else { setPreviewGenMessage(`✅ ${d.message}`); loadCourseware() } },
        onError: d => { setPreviewGenMessage(`❌ ${d.message}`); setPreviewGenRunning(false) },
      })
    } catch { setPreviewGenMessage('❌ 启动失败'); setPreviewGenRunning(false) }
  }

  // Step 3: 确认导航栏（P0-1: 传"auto"让后端自动提取）
  const handleSaveNav = async () => {
    if (!id || previewPages.length === 0) return
    setNavSaving(true)
    try {
      await saveCWNavTemplate(id, 'auto')
      goToStep(4)
      loadCourseware()
    } catch (e) { alert('保存导航栏失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setNavSaving(false) }
  }

  // P0-2: 导航栏AI微调
  const handleRefineNav = async () => {
    if (!id || !navRefineInput.trim()) return
    setNavRefining(true)
    try {
      await refineNav(id, navRefineInput.trim())
      await loadCourseware()
      setNavRefineInput('')
      setPreviewGenMessage('\u2705 导航栏微调完成')
    } catch (e) { setPreviewGenMessage('\u274c 微调失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setNavRefining(false) }
  }






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
              ? p.map(x => x.page_number === d.page_number ? { page_number: d.page_number, title: d.title, html_content: d.html_content } : x)
              : [...p, { page_number: d.page_number, title: d.title, html_content: d.html_content }]
            return next.slice().sort((a, b) => a.page_number - b.page_number)
          })
          // 预览框只单向前移到"已到达的最大页号"：既有"在推进"的反馈，又不会因乱序到达而来回乱跳
          setBuildPreviewNum(prev => d.page_number > prev ? d.page_number : prev)
        },
        // P2 需求②：完成后——全部成功才不动（停在批量生成页让老师自行点"确认课件→"）；
        //   有失败页则明确提示并【留在批量生成页】，老师可对失败页点"继续生成"自动补齐，
        //   不再自动跳到确认提交页（避免带着缺页进入下一步）。loadCourseware 会刷新真实状态与页面。
        onGenDone: d => {
          setBuildRunning(false)
          if (d.fail_count > 0) {
            setBuildMessage(`⚠️ ${d.message}。失败的页面可点下方"继续生成"自动重试补齐（成功页不会重做）。`)
          } else {
            setBuildMessage(`✅ ${d.message}`)
          }
          loadCourseware()
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
              .map(p => ({ page_number: p.page_number, title: p.title, html_content: p.html_content }))
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
    const t = setInterval(async () => { try { const d = await getCourseware(id); if (d.status === 'preview') { setBuildRunning(false); setCourseware(d); setPages(d.pages || []); goToStep(5); setBuildMessage('✅ 完成'); const gp = d.pages.filter(p => p.html_content).map(p => ({ page_number: p.page_number, title: p.title, html_content: p.html_content })); setGeneratedPages(gp); /* v0.42.13: 不再重置预览到第1页，保留当前/最后生成页 */ sseRef.current?.close() } } catch {} }, 15000)
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

  const openSlideshow = (pn?: number) => {
    const allPages = generatedPages.length > 0 ? generatedPages : previewPages
    setSlideshowInitPage(pn || buildPreviewNum || allPages[0]?.page_number || 1)
    setSlideshowOpen(true)
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
  const containerWidth = 912
  const previewScale = containerWidth / CW_WIDTH

  const msgBar = (msg: string) => msg ? <div style={{ padding: '12px 16px', borderRadius: 8, marginBottom: 16, background: msg.startsWith('❌') ? '#FEE2E2' : msg.startsWith('✅') ? '#D1FAE5' : msg.startsWith('⚠️') ? '#FEF3C7' : '#EFF6FF', color: msg.startsWith('❌') ? '#DC2626' : msg.startsWith('✅') ? '#059669' : msg.startsWith('⚠️') ? '#D97706' : '#2563EB', fontSize: 14 }}>{msg}</div> : null

  // v0.41: renderPagePreview 中的 iframe srcDoc 统一注入预览降级
  const renderPagePreview = (pageList: { page_number: number; title: string; html_content: string }[], currentNum: number, setCurrentNum: (n: number) => void, showSlideshow: boolean) => {
    const activePage = currentNum > 0 ? currentNum : (pageList[0]?.page_number || 0)
    const html = pageList.find(p => p.page_number === activePage)?.html_content || ''
    const previewHtml = injectPreviewMode(html)
    return <>
      {pageList.length > 0 && (
        <div style={{ marginBottom: 20 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
            <div style={{ fontSize: 14, fontWeight: 600, color: C.textPrimary }}>📄 已生成 {pageList.length} 页</div>
            {showSlideshow && <button onClick={() => openSlideshow()} style={{ padding: '6px 14px', borderRadius: 8, border: `1px solid ${C.primary}`, background: C.primaryBg, color: C.primary, fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>🖥️ 全屏放映</button>}
          </div>
          {/* W3: 胶片条——单行横向滚动, 页数再多也只占一行高度(PPT/Canva同款导航模式) */}
          <div style={{ display: 'flex', gap: 6, flexWrap: 'nowrap', overflowX: 'auto', paddingBottom: 6 }}>
            {pageList.map(gp => (
              <button key={gp.page_number} onClick={() => setCurrentNum(gp.page_number)} title={'P' + gp.page_number + ' ' + gp.title} style={{
                padding: '6px 10px', borderRadius: 8, cursor: 'pointer', flexShrink: 0, whiteSpace: 'nowrap',
                border: `2px solid ${activePage === gp.page_number ? C.primary : C.border}`,
                background: activePage === gp.page_number ? C.primaryBg : C.white,
                color: activePage === gp.page_number ? C.primary : C.textPrimary,
                fontSize: 12, fontWeight: activePage === gp.page_number ? 600 : 400, transition: 'all 200ms',
              }}>
                <span style={{ fontWeight: 700 }}>P{gp.page_number}</span>
                <span style={{ marginLeft: 5, color: C.textSecondary, fontSize: 11 }}>{gp.title.length > 6 ? gp.title.slice(0, 6) + '…' : gp.title}</span>
              </button>
            ))}
          </div>
        </div>
      )}
      {html && (
        <div style={{ marginBottom: 20 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
            <div style={{ fontSize: 14, fontWeight: 600, color: C.textPrimary }}>{codeViewPageNum === activePage ? '💻' : '📺'} 第 {activePage} 页{codeViewPageNum === activePage ? '源代码' : '预览'}</div>
            <div style={{ display: 'flex', gap: 6 }}>
              <button onClick={() => { if (codeViewPageNum === activePage) setCodeViewPageNum(0); else setCodeViewPageNum(activePage) }} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${codeViewPageNum === activePage ? '#7C3AED' : C.border}`, background: codeViewPageNum === activePage ? 'rgba(124,58,237,0.06)' : 'transparent', color: codeViewPageNum === activePage ? '#7C3AED' : C.textSecondary, fontSize: 12, cursor: 'pointer' }}>{codeViewPageNum === activePage ? '📺 预览' : '💻 源代码'}</button>
              <button onClick={() => { navigator.clipboard.writeText(html).then(() => alert('源代码已复制到剪贴板')).catch(() => {}) }} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer' }}>📋 复制代码</button>
              <button onClick={() => { setFullscreenPageNum(activePage); setFullscreenOpen(true); setFullscreenCodeView(false) }} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer' }}>🔍 全屏预览</button>
              <button onClick={() => openSlideshow(activePage)} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer' }}>🖥️ 放映</button>
            </div>
          </div>
          {codeViewPageNum === activePage ? (
            <div style={{ width: '100%', maxHeight: 500, overflow: 'auto', borderRadius: 14, border: `1px solid ${C.border}`, background: '#1e1e1e', fontFamily: 'Monaco, Consolas, "Courier New", monospace', fontSize: 12, lineHeight: 1.7 }}>
              <table style={{ borderCollapse: 'collapse', width: '100%' }}><tbody>
                {html.split('\n').map((line: string, i: number) => (
                  <tr key={i}>
                    <td style={{ width: 50, minWidth: 50, textAlign: 'right', padding: '0 10px 0 8px', color: '#858585', userSelect: 'none', verticalAlign: 'top', borderRight: '1px solid #333', whiteSpace: 'nowrap' }}>{i + 1}</td>
                    <td style={{ padding: '0 12px', color: '#d4d4d4', whiteSpace: 'pre', wordBreak: 'break-all' }}>{line || ' '}</td>
                  </tr>
                ))}
              </tbody></table>
            </div>
          ) : (
            <div onClick={() => openSlideshow(activePage)} style={{
              width: '100%', height: Math.ceil(CW_HEIGHT * previewScale), position: 'relative', overflow: 'hidden',
              borderRadius: 14, border: `1px solid ${C.border}`, background: '#f8fafc', cursor: 'pointer',
            }}>
              <iframe srcDoc={previewHtml} scrolling="no" style={{ width: CW_WIDTH, height: CW_HEIGHT, border: 'none', pointerEvents: 'none', transform: `scale(${previewScale})`, transformOrigin: 'top left', position: 'absolute', top: 0, left: 0, overflow: 'hidden' }} sandbox="allow-scripts" title={`预览-P${activePage}`} />
            </div>
          )}
        </div>
      )}
    </>
  }

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
        {/* 风格锚点展示条（轮3）：已设锚点时显示缩略图+VAOCI摘要+清除；未设则不显示 */}
        {courseware.style_anchor_asset_id && (
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

      {/* 步骤条 */}
      <div style={{ display: 'flex', gap: 4, marginBottom: 28, padding: '16px 20px', background: C.white, borderRadius: 12, border: `1px solid ${C.border}` }}>
        {STEPS.map((s, i) => {
          const active = i === activeStep, done = i < activeStep, reached = i <= maxStepReached
          return <div key={s.key} onClick={() => { if (reached && !active) goToStep(i) }} style={{ flex: 1, textAlign: 'center', cursor: (reached && !active) ? 'pointer' : 'default' }}>
            <div style={{ width: 32, height: 32, borderRadius: '50%', margin: '0 auto 6px', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 16, background: done ? C.success : active ? C.primary : reached ? '#A7F3D0' : '#F3F4F6', color: done || active ? '#fff' : C.textMuted, fontWeight: 700, transition: 'all 300ms' }}>{done ? '✓' : s.emoji}</div>
            <div style={{ fontSize: 11, fontWeight: active ? 600 : 400, color: active ? C.primary : done ? C.success : C.textMuted }}>{s.label}</div>
          </div>
        })}
      </div>

      {/* 内容区 */}
      <div style={{ background: C.white, borderRadius: 12, border: `1px solid ${C.border}`, padding: 24, minHeight: 400 }}>

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
          {msgBar(sseMessage)}
          {generating && pages.length > 0 && <div style={{ marginBottom: 16 }}><div style={{ fontSize: 13, color: C.textMuted, marginBottom: 8 }}>已生成 {pages.length} 页方案...</div><IndexEditor coursewareId={id!} pages={pages} onPagesChange={setPages} isAdmin={isAdmin} indexOverview={courseware?.index_overview} /></div>}
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
          <IndexEditor coursewareId={id!} pages={pages} onPagesChange={setPages} isAdmin={isAdmin} indexOverview={courseware?.index_overview} />
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
          {msgBar(sseMessage)}
        </div>}

        {/* Step 2: 选择风格 */}
        {activeStep === 2 && courseware && <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <div><h3 style={{ fontSize: 18, fontWeight: 600, color: C.textPrimary, margin: 0 }}>🎨 课件风格定制</h3><p style={{ fontSize: 13, color: C.textSecondary, margin: '4px 0 0' }}>选择视觉风格，配置机构品牌</p></div>
            <button onClick={() => goToStep(1)} style={{ padding: '8px 16px', borderRadius: 8, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 13, cursor: 'pointer' }}>← 返回编辑</button>
          </div>
          <StyleSelector courseware={courseware} coursewareId={id!} onStyleConfirmed={handleStyleConfirmed} />
        </div>}

        {/* Step 3: 确认导航栏（P0-1: 只生成1页封面预览） */}
        {activeStep === 3 && <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <div><h3 style={{ fontSize: 18, fontWeight: 600, color: C.textPrimary, margin: 0 }}>🧭 确认导航栏样式</h3><p style={{ fontSize: 13, color: C.textSecondary, margin: '4px 0 0' }}>AI先生成封面页，请确认顶部导航栏是否满意</p></div>
            {!previewGenRunning && <button onClick={() => goToStep(2)} style={{ padding: '8px 16px', borderRadius: 8, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 13, cursor: 'pointer' }}>← 返回选择风格</button>}
          </div>

          {msgBar(previewGenMessage)}

          {/* P0-1: 只展示1页封面预览 */}
          {previewPages.length > 0 && renderPagePreview(previewPages, previewPages[0]?.page_number || 1, () => {}, false)}

          {/* 操作按钮 */}
          <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            {!previewGenRunning && previewPages.length === 0 && (
              <button onClick={handleGenPreview} style={{ padding: '14px 36px', borderRadius: 10, border: 'none', background: 'linear-gradient(135deg, #F59E0B, #EF4444)', color: '#fff', fontSize: 16, fontWeight: 600, cursor: 'pointer', boxShadow: '0 4px 16px rgba(245,158,11,0.3)' }}>🧭 生成封面预览页</button>
            )}
            {!previewGenRunning && previewPages.length > 0 && <>
              <button onClick={handleGenPreview} style={{ padding: '10px 24px', borderRadius: 8, border: `1px solid ${C.primary}`, background: C.primaryBg, color: C.primary, fontSize: 14, fontWeight: 600, cursor: 'pointer' }}>🔄 重新生成预览</button>
              <button onClick={handleSaveNav} disabled={navSaving} style={{ padding: '10px 24px', borderRadius: 8, border: 'none', background: 'linear-gradient(135deg, #059669, #10B981)', color: '#fff', fontSize: 14, fontWeight: 600, cursor: navSaving ? 'default' : 'pointer', boxShadow: '0 2px 8px rgba(5,150,105,0.3)' }}>
                {navSaving ? '保存中...' : '✅ 导航栏样式满意，开始批量生成 →'}
              </button>
            </>}
            {previewGenRunning && <div style={{ textAlign: 'center', padding: 20, color: C.textMuted, fontSize: 14, width: '100%' }}><div style={{ fontSize: 32, marginBottom: 8 }}>🧭</div>AI正在生成封面预览页，请稍候...</div>}
          </div>

          {/* 提示信息 */}
          {previewPages.length > 0 && !previewGenRunning && (
            <div style={{ marginTop: 16, padding: '12px 16px', borderRadius: 8, background: '#EFF6FF', color: '#2563EB', fontSize: 13 }}>
              💡 请仔细查看封面页的导航栏样式（顶部Logo、机构名、页码位置和颜色）。确认满意后点击"开始批量生成"，后续所有页面将自动使用完全相同的导航栏。
            </div>
          )}

          {/* 批次2（背景图库）：选背景秒换封面（零token零等待），后续批量生成的内页自动带内页底纹 */}
          {previewPages.length > 0 && !previewGenRunning && (
            <AppearancePanel coursewareId={id!} onSwapped={refreshPagesOnly} disabled={buildRunning}
              cwTitle={courseware.title} cwSubject={courseware.subject} cwGrade={courseware.grade} />
          )}

          {/* P0-2: 导航栏AI微调输入区 */}
          {previewPages.length > 0 && !previewGenRunning && (
            <div style={{ marginTop: 16, padding: '16px', borderRadius: 10, border: `1px solid ${C.border}`, background: '#FAFAFA' }}>
              <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🎨 导航栏不满意？输入修改意见让AI微调</div>
              <div style={{ display: 'flex', gap: 10 }}>
                <input value={navRefineInput} onChange={e => setNavRefineInput(e.target.value)}
                  placeholder="例如：Logo再大一点、页码改成右对齐、背景色改为深蓝..."
                  onKeyDown={e => { if (e.key === 'Enter' && !navRefining) handleRefineNav() }}
                  style={{ flex: 1, padding: '10px 14px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 14, outline: 'none' }}
                  disabled={navRefining} />
                <button onClick={handleRefineNav} disabled={navRefining || !navRefineInput.trim()}
                  style={{ padding: '10px 20px', borderRadius: 8, border: 'none', background: navRefineInput.trim() && !navRefining ? '#7C3AED' : '#E5E7EB', color: navRefineInput.trim() && !navRefining ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: navRefineInput.trim() && !navRefining ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
                  {navRefining ? '⏳ 微调中...' : '🎨 AI微调'}
                </button>
              </div>
            </div>
          )}
        </div>}

        {/* Step 4: 批量生成剩余页 */}
        {activeStep === 4 && <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <div><h3 style={{ fontSize: 18, fontWeight: 600, color: C.textPrimary, margin: 0 }}>⚡ 批量生成课件</h3><p style={{ fontSize: 13, color: C.textSecondary, margin: '4px 0 0' }}>使用已确认的导航栏样式，逐页生成剩余课件</p></div>
            {!buildRunning && <button onClick={() => goToStep(3)} style={{ padding: '8px 16px', borderRadius: 8, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 13, cursor: 'pointer' }}>← 返回确认导航栏</button>}
          </div>
          {msgBar(buildMessage)}
          {batchTotal > 0 && <div style={{ marginBottom: 20 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, color: C.textSecondary, marginBottom: 6 }}>
              <span>{cwInterrupted ? '续传进度（已完成页不会重做）' : '生成进度（多页同时进行）'}</span>
              <span>本次需生成 {batchTotal} 页 · 已完成 <b style={{ color: C.success }}>{batchDone}</b> 页 · 剩余 {batchRemaining} 页</span>
            </div>
            <div style={{ height: 8, borderRadius: 4, background: '#F3F4F6', overflow: 'hidden' }}><div style={{ height: '100%', borderRadius: 4, transition: 'width 500ms', width: `${batchPercent}%`, background: 'linear-gradient(90deg, #F59E0B, #EF4444)' }} /></div>
          </div>}
          {renderPagePreview(generatedPages, buildPreviewNum, setBuildPreviewNum, true)}
          {/* v0.42.12 续生成提示条：检测到上次生成被中断（已生成部分页但仍有剩余） */}
          {!buildRunning && cwInterrupted && (
            <div style={{ padding: '12px 16px', borderRadius: 8, marginBottom: 14, background: '#FEF3C7', color: '#92400E', fontSize: 14, lineHeight: 1.6 }}>
              ⚠️ 检测到上次生成被中断，已完成 <b>{cwDoneCount}/{cwTotalCount}</b> 页。点击「继续生成」会<b>跳过已生成的页面</b>，仅生成剩余 <b>{cwRemainingCount}</b> 页，无需从头再来。
            </div>
          )}
          {/* 场景A：尚未开始批量生成（仅封面或全空，仍有剩余页待生成） */}
          {!buildRunning && cwDoneCount <= 1 && cwRemainingCount > 0 && (
            <button onClick={handleBuildStart} style={{ padding: '14px 36px', borderRadius: 10, border: 'none', background: 'linear-gradient(135deg, #F59E0B, #EF4444)', color: '#fff', fontSize: 16, fontWeight: 600, cursor: 'pointer', boxShadow: '0 4px 16px rgba(245,158,11,0.3)' }}>⚡ 开始批量生成剩余页面</button>
          )}
          {/* 场景B：已生成部分页且仍有剩余（中断续传——跳过已生成页只补剩余） */}
          {!buildRunning && cwInterrupted && <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            <button onClick={handleBuildStart} style={{ padding: '12px 28px', borderRadius: 8, border: 'none', background: 'linear-gradient(135deg, #059669, #10B981)', color: '#fff', fontSize: 15, fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 8px rgba(5,150,105,0.3)' }}>▶️ 继续生成剩余 {cwRemainingCount} 页</button>
            <button onClick={() => openSlideshow()} style={{ padding: '12px 24px', borderRadius: 8, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 14, fontWeight: 600, cursor: 'pointer' }}>🖥️ 预览已生成页</button>
          </div>}
          {/* 场景C：全部页面已生成完毕 */}
          {!buildRunning && cwRemainingCount === 0 && cwTotalCount > 0 && <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            <button onClick={() => openSlideshow()} style={{ padding: '10px 24px', borderRadius: 8, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 14, fontWeight: 600, cursor: 'pointer' }}>🖥️ 全屏放映</button>
            <button onClick={() => { goToStep(5); loadCourseware() }} style={{ padding: '10px 24px', borderRadius: 8, border: 'none', background: 'linear-gradient(135deg, #059669, #10B981)', color: '#fff', fontSize: 14, fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 8px rgba(5,150,105,0.3)' }}>确认课件 →</button>
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
          {renderPagePreview(generatedPages, buildPreviewNum, setBuildPreviewNum, true)}
          




          {/* 批次W2: Step5工作台Tab——预览区下方收纳全部工具, 页面高度恒定; 默认「页面微调」(最高频) */}
          {generatedPages.length > 0 && <>
            <div style={{ display: 'flex', gap: 8, marginTop: 20, flexWrap: 'wrap', borderBottom: '2px solid ' + C.border, paddingBottom: 12 }}>
              {([['refine', '🛠 页面微调'], ['background', '🎨 背景'], ['font', '🔤 字体'], ['image', '🖼 图片'], ['video', '🎬 视频'], ['template', '💾 保存模板']] as const).map(([k, label]) => (
                <button key={k} onClick={() => setWsTab(k)}
                  style={{ padding: '10px 22px', borderRadius: 10, cursor: 'pointer',
                    border: '2px solid ' + (wsTab === k ? C.primary : C.border),
                    background: wsTab === k ? C.primaryBg : '#fff',
                    color: wsTab === k ? C.primary : C.textSecondary,
                    fontSize: 14, fontWeight: wsTab === k ? 600 : 400, transition: 'all 200ms' }}>
                  {label}
                </button>
              ))}
            </div>
            {wsTab === 'refine' && (
              <RefinePanel coursewareId={id!} pageNum={buildPreviewNum}
                onPageUpdated={(pn, html) => setGeneratedPages(prev => prev.map(p => p.page_number === pn ? { ...p, html_content: html } : p))} />
            )}
            {wsTab === 'background' && (
              <AppearancePanel mode="background" coursewareId={id!} onSwapped={refreshPagesOnly} disabled={buildRunning}
                cwTitle={courseware.title} cwSubject={courseware.subject} cwGrade={courseware.grade} />
            )}
            {wsTab === 'font' && (
              <AppearancePanel mode="font" coursewareId={id!} onSwapped={refreshPagesOnly} disabled={buildRunning} />
            )}
            {(wsTab === 'image' || wsTab === 'video') && (
              <MediaManagerPanel coursewareId={id!} pageNum={buildPreviewNum} courseware={courseware} mediaTab={wsTab}
                anchorSetting={anchorSetting} anchorClearing={anchorClearing}
                onSetAnchor={handleSetAnchor} onClearAnchor={handleClearAnchor} />
            )}
            {wsTab === 'template' && <TemplateSavePanel coursewareId={id!} />}
          </>}
        </div>}
      </div>

      {/* v137: 全屏预览（带工具栏+键盘导航+resize响应） */}
      {fullscreenOpen && allSlideshowPages.length > 0 && <CWFullscreenPreview
        pages={allSlideshowPages}
        initialPageNum={fullscreenPageNum}
        codeView={fullscreenCodeView}
        onToggleCode={() => setFullscreenCodeView(!fullscreenCodeView)}
        onClose={() => setFullscreenOpen(false)}
        onSlideshow={(pn) => { setFullscreenOpen(false); setSlideshowInitPage(pn); setSlideshowOpen(true) }}
      />}

      {slideshowOpen && allSlideshowPages.length > 0 && <SlideshowPlayer pages={allSlideshowPages} initialPage={slideshowInitPage} onClose={() => setSlideshowOpen(false)} />}

      {/* 批次W1: 视频编辑器弹窗已随多媒体逻辑迁入 MediaManagerPanel */}

    </div>
  )
}

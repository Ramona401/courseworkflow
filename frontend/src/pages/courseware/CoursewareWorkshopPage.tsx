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
 *   - buildPreviewNum 是 Step5 唯一选中页真相源；mediaPageNum 是其只读别名；
 *     一个 useEffect 监听其变化自动 listPageAssets 拉当前页媒体并以 cancelled 标志防串页。
 *   - source_type==='3d_single' 时早返回走 ThreeDSingleView，不走标准六步。
 *   - 图片 Tab 进页/切未拉过页自动调 suggestImagePrompt 出多条配图建议（按页 imgSuggestCache 缓存）。
 *   - 单页微调支持附截图(refineImage,data URI,≤8MB)走多模态 + 输入框 Ctrl+V 粘贴截图；
 *     重生(handleRegeneratePage)带二次确认、清本页插图、与微调互斥。
 *   - 批量生成支持中断续传（跳过已生成页）。
 *   - 「☁️云盘」上传成功写 public_oss_url；删除已上云资产弹强警告确认。
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  getCourseware, generateCWIndex, generateCWIndexFromTopic, subscribeCWIndexSSE,
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

  // P0-4: 页面微调状态(批次4b: refinePageNum 已删除, 统一用 buildPreviewNum 作为唯一选中页)
  const [refineInput, setRefineInput] = useState('')
  const [refineRunning, setRefineRunning] = useState(false)
  // 批次4a: 单页微调截图(data URI, 走多模态) + 重生运行态
  const [refineImage, setRefineImage] = useState('')
  const [regenRunning, setRegenRunning] = useState(false)

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
  // v137: 保存模板状态
  const [saveTplName, setSaveTplName] = useState('')
  const [savingTpl, setSavingTpl] = useState(false)

  const [slideshowOpen, setSlideshowOpen] = useState(false)
  const [slideshowInitPage, setSlideshowInitPage] = useState(1)
  // v0.42 多媒体: 媒体管理状态(批次4b: mediaPageNum 改为 buildPreviewNum 的只读别名,
  //   下方所有读 mediaPageNum 的引用无需改动; 已删除独立的页码 state 和下拉选择器)
  const [mediaAssets, setMediaAssetsRaw] = useState<import('@/api/coursewares').CoursewareAsset[]>([])
  const setMediaAssets = setMediaAssetsRaw  // 批次4b: 保持下方 setMediaAssets 调用不变
  const [mediaGenPrompt, setMediaGenPrompt] = useState('')
  const [mediaSize, setMediaSize] = useState('1920x1920')
  const [mediaGenerating, setMediaGenerating] = useState(false)
  const [mediaMessage, setMediaMessage] = useState('')
  const [mediaPreviewUrl, setMediaPreviewUrl] = useState('')
  const [mediaRefUrl, setMediaRefUrl] = useState('')  // 参考图URL（图生图）
  // 批次4c+: AI 写详细提示词运行态 + 视频三件物料中的分镜图提示词/台词展示
  const [mediaPromptSuggesting, setMediaPromptSuggesting] = useState(false)
  // 图片多提示词: AI 返回的多条配图建议(每条含 caption/prompt/各自尺寸)
  //   仅当 AI 返回 >1 条时以卡片列表呈现; ==1 条时沿用老行为直接填入生成框
  const [imgSuggestions, setImgSuggestions] = useState<(ImagePromptSuggestion & { size: string })[]>([])
  // 勾选了哪几条做批量生成(默认不选, 老师主动勾); 存下标集合
  const [imgSuggestSelected, setImgSuggestSelected] = useState<Set<number>>(new Set())
  // 批量生成运行态 + 进度(已完成/总数)
  const [batchGenRunning, setBatchGenRunning] = useState(false)
  const [batchGenProgress, setBatchGenProgress] = useState({ done: 0, total: 0 })
  // 风格快选+Tab(本轮): 当前展示的建议 Tab 下标; 当前选中的图片风格 key(空=未选)
  const [activeImgSuggestTab, setActiveImgSuggestTab] = useState(0)
  const [mediaStyleKey, setMediaStyleKey] = useState('')
  // 遗留项②：显式记住上次追加的风格后缀确切文本，剥离时直接 replace 该串，不靠 endsWith 位置判断
  const [styleSuffixText, setStyleSuffixText] = useState('')
  // 图片多提示词(新交互): 自动拉取建议的运行态 + 每页建议缓存(切回不重复调 AI)
  const [imgSuggestLoading, setImgSuggestLoading] = useState(false)
  const imgSuggestCache = useRef<Map<number, (ImagePromptSuggestion & { size: string })[]>>(new Map())
  // v0.42.1 视频生成状态
  const [mediaTab, setMediaTab] = useState<'image'|'video'>('image')  // 多媒体Tab切换
  // v0.42.1 视频编辑状态
  const [editorOpen, setEditorOpen] = useState(false)         // 视频编辑器弹窗
  const [editorExporting, setEditorExporting] = useState(false) // 编辑器导出中

  // ---- 风格锚点（轮3）：设/清锚点运行态 + 操作方法 ----
  const [anchorSetting, setAnchorSetting] = useState('')   // 正在设为锚点的资产ID（''=无）
  const [anchorClearing, setAnchorClearing] = useState(false)

  // 设为锚点：一步式同步（后端取URL→多模态提取VAOCI→落库），成功后乐观更新 courseware
  const handleSetAnchor = async (assetId: string) => {
    if (!id || anchorSetting) return
    setAnchorSetting(assetId)
    setMediaMessage('🎨 正在将该图设为风格锚点（AI读图提取风格DNA，约需数秒）...')
    try {
      const res = await setStyleAnchor(id, assetId)
      // 乐观更新：把锚点两字段写回 courseware，无需整页 reload
      setCourseware(prev => prev ? { ...prev, style_anchor_asset_id: res.asset_id, style_anchor_vaoci: res.vaoci, style_anchor_url: res.anchor_url } : prev)
      setMediaMessage('✅ 已设为风格锚点！后续生成的配图将自动参考此图风格，保持全课件视觉统一')
    } catch (e) {
      setMediaMessage('❌ 设置锚点失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setAnchorSetting('') }
  }

  // 清除锚点：成功后乐观更新 courseware 锚点字段为空
  const handleClearAnchor = async () => {
    if (!id || anchorClearing) return
    if (!confirm('确定清除风格锚点？清除后新生成的配图将不再自动套用此风格。')) return
    setAnchorClearing(true)
    try {
      await clearStyleAnchor(id)
      setCourseware(prev => prev ? { ...prev, style_anchor_asset_id: null, style_anchor_vaoci: '', style_anchor_url: '' } : prev)
      setMediaMessage('✅ 已清除风格锚点')
    } catch (e) {
      setMediaMessage('❌ 清除锚点失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setAnchorClearing(false) }
  }

  const sseRef = useRef<{ close: () => void } | null>(null)


  

  // 批次4b: mediaPageNum 作为 buildPreviewNum 的只读别名——唯一选中页真相源
  const mediaPageNum = buildPreviewNum

  // 批次4b: 选中页(buildPreviewNum)变化时, 自动拉取当前页媒体资产并清空旧的
  //   替代原来媒体下拉 onChange 里的手动拉取逻辑; 切页即换页媒体, 防串页
  useEffect(() => {
    if (!id || buildPreviewNum <= 0) { setMediaAssetsRaw([]); return }
    let cancelled = false
    setMediaAssetsRaw([])
    listPageAssets(id, buildPreviewNum)
      .then(res => { if (!cancelled) setMediaAssetsRaw(res.assets || []) })
      .catch(() => { if (!cancelled) setMediaAssetsRaw([]) })
    return () => { cancelled = true }
  }, [id, buildPreviewNum])

  // 图片多提示词: 切页或切 Tab 时清空 AI 多条建议列表 + 生成框 + 参考图(防串页残留)
  //   新页默认空生成框, 老师点卡片「填入」才填; 避免上一页手输/填入内容串到新页
  useEffect(() => {
    setImgSuggestions([]); setImgSuggestSelected(new Set())
    setMediaGenPrompt(''); setMediaRefUrl('')
    setActiveImgSuggestTab(0); setMediaStyleKey('')
  }, [buildPreviewNum, mediaTab])

  // 图片多提示词(新交互): 进入图片Tab/切到未拉过的页时, 自动出本页配图建议
  //   顺序: 本会话缓存 → 库已存(getStoredImageSuggestions, 不调AI省token) → 都没有才调AI(那条会自动写库)
  useEffect(() => {
    if (mediaTab !== 'image' || !id || buildPreviewNum <= 0) return
    const cached = imgSuggestCache.current.get(buildPreviewNum)
    if (cached) { setImgSuggestions(cached); return }
    let cancelled = false
    setImgSuggestLoading(true)
    const pageNum = buildPreviewNum
    getStoredImageSuggestions(id, pageNum)
      .then(stored => {
        if (cancelled) return null
        const list = (stored.prompts || []).map(it => ({ ...it, size: mediaSize }))
        if (list.length > 0) {
          imgSuggestCache.current.set(pageNum, list)
          setImgSuggestions(list)
          setMediaMessage('✅ 已载入本页已存的 ' + list.length + ' 条配图建议（未消耗AI）')
          return null
        }
        return suggestImagePrompt(id, pageNum)
      })
      .then(res => {
        if (cancelled || !res) return
        const list = (res.prompts || []).map(it => ({ ...it, size: mediaSize }))
        imgSuggestCache.current.set(pageNum, list)
        setImgSuggestions(list)
        setMediaMessage(list.length > 0
          ? '✅ AI 建议本页配 ' + list.length + ' 张图，请在下方勾选或逐条填入生成框'
          : '⚠️ AI 未给出配图建议，可手动填写提示词生成')
      })
      .catch(e => { if (!cancelled) setMediaMessage('❌ 生成配图建议失败: ' + (e instanceof Error ? e.message : '未知错误') + '（可点「重新生成配图建议」重试，或手动填写）') })
      .finally(() => { if (!cancelled) setImgSuggestLoading(false) })
    return () => { cancelled = true }
  }, [mediaTab, buildPreviewNum, id])

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
        onIndexPage: page => setPages(prev => prev.some(p => p.page_number === page.page_number) ? prev.map(p => p.page_number === page.page_number ? page : p) : [...prev, page]),
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
        onIndexPage: page => setPages(prev => prev.some(p => p.page_number === page.page_number) ? prev.map(p => p.page_number === page.page_number ? page : p) : [...prev, page]),
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
        onGenDone: d => { setPreviewGenRunning(false); setPreviewGenMessage(d.fail_count > 0 ? `⚠️ ${d.message}` : `✅ ${d.message}`); loadCourseware() },
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

  // P0-4: 单页AI微调(批次4a: 支持随附截图走多模态; 微调=保留页内已插入图片)
  const handleRefinePage = async () => {
    // 批次4b: 选中页统一用 buildPreviewNum(大预览框当前页)
    if (!id || buildPreviewNum <= 0 || !refineInput.trim()) return
    setRefineRunning(true)
    try {
      // refineImage 非空时随请求传入, 后端走多模态微调(CallAIMultimodal)
      const result = await refinePage(id, buildPreviewNum, refineInput.trim(), refineImage || undefined)
      if (result.html_content) {
        setGeneratedPages(prev => prev.map(p => p.page_number === buildPreviewNum ? { ...p, html_content: result.html_content } : p))
      }
      setRefineInput(''); setRefineImage('')
      setBuildMessage('\u2705 ' + result.message)
    } catch (e) { setBuildMessage('\u274c 微调失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setRefineRunning(false) }
  }

  // 批次4a: 单页从零重生(重生=不保留页内已插入图片, 异于微调的增量改; 后端无并发锁故运行态禁用按钮)
  const handleRegeneratePage = async () => {
    // 批次4b: 选中页统一用 buildPreviewNum
    if (!id || buildPreviewNum <= 0 || regenRunning || refineRunning) return
    // 二次确认: 重生会清空本页已插入的图片
    if (!confirm('⚠️ 重生第 ' + buildPreviewNum + ' 页将按方案从零重画整页，会清空本页已插入的图片（图片资产仍在多媒体库，可重新插入）。确定重生？')) return
    setRegenRunning(true); setBuildMessage('🔄 正在重生第 ' + buildPreviewNum + ' 页，请稍候...')
    try {
      const result = await regenerateCWPage(id, buildPreviewNum)
      if (result.html_content) {
        setGeneratedPages(prev => prev.map(p => p.page_number === buildPreviewNum ? { ...p, html_content: result.html_content } : p))
      }
      setBuildMessage('\u2705 ' + result.message)
    } catch (e) { setBuildMessage('\u274c 重生失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setRegenRunning(false) }
  }

  // 批次4c: 共用——将图片文件读为 dataURI 存入 refineImage(8MB上限, 截图微调走多模态)
  //   fromPaste=true 表示来自剪贴板粘贴, 额外给出"已粘贴"提示
  const loadRefineImageFile = (f: File, fromPaste = false) => {
    if (f.size > 8 * 1024 * 1024) { setBuildMessage('\u274c 截图不能超过8MB'); return }
    const reader = new FileReader()
    reader.onload = () => {
      setRefineImage(typeof reader.result === 'string' ? reader.result : '')
      if (fromPaste) setBuildMessage('\u2705 已从剪贴板粘贴截图，微调将参考该图')
    }
    reader.onerror = () => setBuildMessage('\u274c 截图读取失败')
    reader.readAsDataURL(f)
  }

  // 批次4c 需求②: 微调输入框 Ctrl+V 粘贴剪贴板图片 → 作为微调参考截图
  //   仅当剪贴板含图片时 preventDefault 并消费; 纯文本粘贴走默认行为不受影响(绑在input上防全页冲突)
  const handleRefinePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    const items = e.clipboardData?.items
    if (!items) return
    for (let i = 0; i < items.length; i++) {
      const it = items[i]
      if (it.type && it.type.startsWith('image/')) {
        const f = it.getAsFile()
        if (!f) continue
        e.preventDefault()  // 阻止二进制内容污染文本输入框
        loadRefineImageFile(f, true)
        return
      }
    }
    // 剪贴板无图片 → 不拦截, 让默认文本粘贴正常进行
  }


  // 图片多提示词(新交互): 「🔄重新生成配图建议」——重新调 AI 覆盖本页建议缓存并出卡片
  //   统一以卡片呈现(含单条), 不再自动填入生成框; 老师选中卡片「填入」才进框
  const handleSuggestImagePrompt = async () => {
    if (!id || buildPreviewNum <= 0 || mediaPromptSuggesting) return
    setMediaPromptSuggesting(true); setMediaMessage('🤖 AI 正在按本页方案重新撰写配图建议...')
    try {
      const res = await suggestImagePrompt(id, buildPreviewNum)
      const list = (res.prompts || []).map(it => ({ ...it, size: mediaSize }))
      imgSuggestCache.current.set(buildPreviewNum, list)
      setImgSuggestions(list)
      setImgSuggestSelected(new Set())
      setMediaMessage(list.length > 0
        ? '✅ AI 建议本页配 ' + list.length + ' 张图，请在下方勾选或逐条填入生成框'
        : '⚠️ AI 未给出配图建议，可手动填写提示词生成')
    } catch (e) { setMediaMessage('❌ 生成配图建议失败: ' + (e instanceof Error ? e.message : '未知错误')) }
    finally { setMediaPromptSuggesting(false) }
  }

  // 图片多提示词: 切换某条建议的勾选状态
  const toggleImgSuggest = (idx: number) => {
    setImgSuggestSelected(prev => {
      const next = new Set(prev)
      if (next.has(idx)) next.delete(idx); else next.add(idx)
      return next
    })
  }

  // 图片多提示词: 修改某条建议的尺寸
  const setImgSuggestSize = (idx: number, size: string) => {
    setImgSuggestions(prev => prev.map((it, i) => i === idx ? { ...it, size } : it))
  }

  // 风格快选: 点风格把其描述作为后缀融入生成框(同风格再点=取消; 换风格=替换旧后缀)
  //   不污染正文: 维护 mediaStyleKey, 替换时先剥掉上一个风格后缀("，"+旧desc)再接新的
  const toggleImgStyle = (key: string) => {
    const next = CW_IMG_STYLES.find(s => s.key === key)
    setMediaGenPrompt(prev => {
      let base = prev
      // 遗留项②：用 styleSuffixText 记录的"上次追加的确切后缀串"直接剥除，
      //   不依赖 endsWith 位置——即便用户在后缀之后又手输了内容，也能精准剥掉旧风格描述。
      if (styleSuffixText) {
        // 优先整体替换"，+后缀"，否则替换裸后缀，最后兜底清尾部标点
        if (base.includes('，' + styleSuffixText)) base = base.replace('，' + styleSuffixText, '')
        else if (base.includes(styleSuffixText)) base = base.replace(styleSuffixText, '')
        base = base.replace(/[，,\s]+$/, '')
      }
      // 再点同一风格 = 取消(只剥不加)
      if (key === mediaStyleKey) { setStyleSuffixText(''); return base }
      // 换/选新风格 = 追加新后缀，并记住这次追加的确切文本供下次剥离
      const desc = next ? next.desc : ''
      setStyleSuffixText(desc)
      return base ? base + '，' + desc : desc
    })
    setMediaStyleKey(key === mediaStyleKey ? '' : key)
  }

  // 图片多提示词: 把某条建议填入生成框(老师可微调后单独生成)
  const fillImgSuggest = (idx: number) => {
    const it = imgSuggestions[idx]
    if (!it) return
    setMediaGenPrompt(it.prompt)
    setMediaSize(it.size)
    setMediaMessage('✅ 已把第 ' + (idx + 1) + ' 条提示词填入生成框，可微调后点「生成图片」')
  }

  // 图片多提示词: 批量生成勾选的多张图(串行, 避免并发打爆豆包 API; 逐张塞入 mediaAssets)
  const handleBatchGenImages = async () => {
    if (!id || buildPreviewNum <= 0 || batchGenRunning) return
    const idxList = Array.from(imgSuggestSelected).sort((a, b) => a - b)
    if (idxList.length === 0) { setMediaMessage('⚠️ 请先勾选要生成的图片'); return }
    setBatchGenRunning(true)
    setBatchGenProgress({ done: 0, total: idxList.length })
    setMediaMessage('🤖 开始批量生成 ' + idxList.length + ' 张图，逐张生成中...')
    let okCount = 0; let failCount = 0
    for (let k = 0; k < idxList.length; k++) {
      const it = imgSuggestions[idxList[k]]
      if (!it || !it.prompt.trim()) { failCount++; continue }
      setBatchGenProgress({ done: k, total: idxList.length })
      setMediaMessage('🤖 正在生成第 ' + (k + 1) + '/' + idxList.length + ' 张：' + (it.caption || it.prompt.slice(0, 20)))
      try {
        const res = await generateCWImage(id, buildPreviewNum, it.prompt.trim(), undefined, it.size)
        setMediaAssets(prev => [{ id: res.asset_id, courseware_id: id, page_id: null, placeholder_id: '', asset_type: 'image', generation_prompt: it.prompt, oss_url: res.url, file_size: 0, mime_type: 'image/png', status: 'uploaded', created_at: new Date().toISOString() }, ...prev])
        okCount++
      } catch { failCount++ }
    }
    setBatchGenProgress({ done: idxList.length, total: idxList.length })
    setMediaMessage((failCount > 0 ? '⚠️' : '✅') + ' 批量生成完成：成功 ' + okCount + ' 张' + (failCount > 0 ? '，失败 ' + failCount + ' 张' : ''))
    setBatchGenRunning(false)
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
        onGenStart: d => { setBuildMessage(d.message); setBuildProgress({ current: 0, total: d.total_pages }) },
        onGenProgress: d => { setBuildMessage(d.message); setBuildProgress(p => ({ ...p, current: d.current_page })) },
        onGenPage: d => { setGeneratedPages(p => {
          const exists = p.some(x => x.page_number === d.page_number)
          if (exists) return p.map(x => x.page_number === d.page_number ? { page_number: d.page_number, title: d.title, html_content: d.html_content } : x)
          return [...p, { page_number: d.page_number, title: d.title, html_content: d.html_content }]
        }); setBuildPreviewNum(d.page_number) },
        onGenDone: d => { setBuildRunning(false); setBuildMessage(d.fail_count > 0 ? `⚠️ ${d.message}` : `✅ ${d.message}`); loadCourseware() },
        onError: d => { setBuildMessage(`❌ ${d.message}`); setBuildRunning(false) },
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
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            {pageList.map(gp => (
              <button key={gp.page_number} onClick={() => setCurrentNum(gp.page_number)} style={{
                padding: '8px 14px', borderRadius: 10, cursor: 'pointer',
                border: `2px solid ${activePage === gp.page_number ? C.primary : C.border}`,
                background: activePage === gp.page_number ? C.primaryBg : C.white,
                color: activePage === gp.page_number ? C.primary : C.textPrimary,
                fontSize: 13, fontWeight: activePage === gp.page_number ? 600 : 400, transition: 'all 200ms',
              }}>
                <span style={{ fontWeight: 700 }}>P{gp.page_number}</span>
                <span style={{ marginLeft: 6, color: C.textSecondary, fontSize: 12 }}>{gp.title.length > 10 ? gp.title.slice(0, 10) + '...' : gp.title}</span>
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
            <button onClick={handleClearAnchor} disabled={anchorClearing}
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
          {buildProgress.total > 0 && <div style={{ marginBottom: 20 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, color: C.textSecondary, marginBottom: 6 }}><span>{cwInterrupted ? '续传进度（已完成页不会重做）' : '生成进度'}</span><span>本次需生成 {buildProgress.total} 页 · 已完成 {buildProgress.current} 页</span></div>
            <div style={{ height: 8, borderRadius: 4, background: '#F3F4F6', overflow: 'hidden' }}><div style={{ height: '100%', borderRadius: 4, transition: 'width 500ms', width: `${(buildProgress.current / buildProgress.total) * 100}%`, background: 'linear-gradient(90deg, #F59E0B, #EF4444)' }} /></div>
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
          <div style={{ textAlign: 'center', marginBottom: 20 }}>
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
              <button onClick={() => goToStep(4)} style={{ padding: '8px 16px', borderRadius: 8, border: '1px solid ' + C.border, background: 'transparent', color: C.textSecondary, fontSize: 13, cursor: 'pointer' }}>{'← 返回重新生成'}</button>
            </div>
            <div style={{ fontSize: 48, marginBottom: 8 }}>✅</div>
            <div style={{ fontSize: 18, fontWeight: 600, color: C.textPrimary, marginBottom: 4 }}>课件预览与确认</div>
          </div>
          {renderPagePreview(generatedPages, buildPreviewNum, setBuildPreviewNum, true)}
          
          {/* P0-4: 每页AI微调 */}
          {generatedPages.length > 0 && (
            <div style={{ marginTop: 16, padding: '16px', borderRadius: 10, border: `1px solid ${C.border}`, background: '#FAFAFA' }}>
              <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🎨 对某页不满意？在上方预览区选中该页，输入修改意见</div>
              {/* 批次4b: 删除独立页码下拉, 统一跟随上方大预览框选中页 buildPreviewNum */}
              <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', alignItems: 'center' }}>
                <span style={{ padding: '8px 12px', borderRadius: 8, background: C.primaryBg, color: C.primary, fontSize: 13, fontWeight: 600, whiteSpace: 'nowrap' }}>
                  当前：第 {buildPreviewNum || '—'} 页
                </span>
                <input value={refineInput} onChange={e => setRefineInput(e.target.value)}
                  placeholder="例如：标题字号再大一些、增加图片占位...（可 Ctrl+V 粘贴截图，先在上方选要改的页）"
                  onKeyDown={e => { if (e.key === 'Enter' && !refineRunning && buildPreviewNum > 0) handleRefinePage() }}
                  onPaste={handleRefinePaste}
                  style={{ flex: 1, padding: '10px 14px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 14, outline: 'none', minWidth: 200 }}
                  disabled={refineRunning} />
                <button onClick={handleRefinePage} disabled={refineRunning || buildPreviewNum <= 0 || !refineInput.trim()}
                  style={{ padding: '10px 20px', borderRadius: 8, border: 'none', background: buildPreviewNum > 0 && refineInput.trim() && !refineRunning ? '#7C3AED' : '#E5E7EB', color: buildPreviewNum > 0 && refineInput.trim() && !refineRunning ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: buildPreviewNum > 0 && refineInput.trim() && !refineRunning ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
                  {refineRunning ? '⏳ 微调中...' : '🎨 AI微调'}
                </button>
              </div>
              {/* 批次4a: 截图粘贴 + 重生本页 */}
              <div style={{ display: 'flex', gap: 10, marginTop: 10, flexWrap: 'wrap', alignItems: 'center' }}>
                {refineImage ? (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <img src={refineImage} alt="参考截图" style={{ width: 40, height: 40, objectFit: 'cover', borderRadius: 6, border: '2px solid #7C3AED' }} />
                    <span style={{ fontSize: 11, color: '#7C3AED' }}>已附截图(微调将参考)</span>
                    <button onClick={() => setRefineImage('')} disabled={refineRunning || regenRunning} style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid #EF4444', background: 'transparent', color: '#EF4444', fontSize: 11, cursor: (refineRunning || regenRunning) ? 'default' : 'pointer' }}>移除</button>
                  </div>
                ) : (
                  <button onClick={() => {
                    const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*'
                    inp.onchange = (ev) => {
                      const f = (ev.target as HTMLInputElement).files?.[0]
                      if (!f) return
                      loadRefineImageFile(f)
                    }; inp.click()
                  }} disabled={refineRunning || regenRunning} style={{ padding: '8px 14px', borderRadius: 8, border: '1px dashed #7C3AED', background: 'rgba(124,58,237,0.04)', color: '#7C3AED', fontSize: 13, cursor: (refineRunning || regenRunning) ? 'default' : 'pointer' }}>📷 附截图微调（或在输入框 Ctrl+V 粘贴）</button>
                )}
                <div style={{ flex: 1 }} />
                <button onClick={handleRegeneratePage} disabled={buildPreviewNum <= 0 || regenRunning || refineRunning}
                  title={buildPreviewNum <= 0 ? '请先在上方预览区选中页' : '按方案从零重画本页(会清空本页已插入的图片)'}
                  style={{ padding: '8px 18px', borderRadius: 8, border: 'none', background: (buildPreviewNum > 0 && !regenRunning && !refineRunning) ? 'linear-gradient(135deg, #F59E0B, #EF4444)' : '#E5E7EB', color: (buildPreviewNum > 0 && !regenRunning && !refineRunning) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (buildPreviewNum > 0 && !regenRunning && !refineRunning) ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
                  {regenRunning ? '⏳ 重生中...' : '🔄 重生本页'}
                </button>
              </div>
              <div style={{ marginTop: 6, fontSize: 11, color: '#9CA3AF' }}>💡 微调=在现有页面上增量修改、保留已插图片；重生=按方案从零重画、不保留已插图片。页面变形/损坏时用重生补救。截图除「附截图微调」选文件外，也可在微调输入框直接 Ctrl+V 粘贴。</div>
            </div>
          )}
          {/* v137: 保存为我的模板 */}
          {generatedPages.length > 0 && (
            <div style={{ marginTop: 16, padding: '16px', borderRadius: 10, border: `1px solid ${C.border}`, background: '#FAFAFA' }}>
              <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>💾 保存为我的模板（下次生成课件可复用当前风格和导航栏）</div>
              <div style={{ display: 'flex', gap: 10 }}>
                <input value={saveTplName} onChange={e => setSaveTplName(e.target.value)}
                  placeholder="输入模板名称，如：我的品牌模板-蓝色版"
                  style={{ flex: 1, padding: '10px 14px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 14, outline: 'none' }} />
                <button onClick={async () => {
                  if (!id || !saveTplName.trim() || savingTpl) return
                  setSavingTpl(true)
                  try {
                    const res = await saveAsMyTemplate(id, { name: saveTplName.trim() })
                    alert(res.message || '模板保存成功！')
                    setSaveTplName('')
                  } catch (e) { alert('保存失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                  finally { setSavingTpl(false) }
                }} disabled={savingTpl || !saveTplName.trim()}
                  style={{ padding: '10px 20px', borderRadius: 8, border: 'none', background: saveTplName.trim() && !savingTpl ? '#059669' : '#E5E7EB', color: saveTplName.trim() && !savingTpl ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: saveTplName.trim() && !savingTpl ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
                  {savingTpl ? '⏳ 保存中...' : '💾 保存模板'}
                </button>
              </div>
            </div>
          )}
          <div style={{ marginTop: 16, padding: 20, borderRadius: 12, border: '1px solid ' + C.border, background: '#FAFAFA' }}>
              <div style={{ fontSize: 15, fontWeight: 600, color: C.textPrimary, marginBottom: 12 }}>🖼️ 多媒体管理</div>

              {/* v0.42.1: 图片/视频Tab切换 */}
              <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
                <button onClick={() => setMediaTab('image')} style={{ padding: '8px 20px', borderRadius: 8, border: '1px solid ' + (mediaTab === 'image' ? C.primary : C.border), background: mediaTab === 'image' ? C.primaryBg : '#fff', color: mediaTab === 'image' ? C.primary : C.textSecondary, fontSize: 14, fontWeight: mediaTab === 'image' ? 600 : 400, cursor: 'pointer' }}>🖼️ 图片</button>
                <button onClick={() => setMediaTab('video')} style={{ padding: '8px 20px', borderRadius: 8, border: '1px solid ' + (mediaTab === 'video' ? '#7C3AED' : C.border), background: mediaTab === 'video' ? 'rgba(124,58,237,0.06)' : '#fff', color: mediaTab === 'video' ? '#7C3AED' : C.textSecondary, fontSize: 14, fontWeight: mediaTab === 'video' ? 600 : 400, cursor: 'pointer' }}>🎬 视频</button>
              </div>

              {/* 批次4b: 删除独立页码下拉, 媒体管理跟随上方大预览框选中页 buildPreviewNum */}
              <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 16, alignItems: 'center' }}>
                <span style={{ padding: '10px 14px', borderRadius: 8, background: C.primaryBg, color: C.primary, fontSize: 14, fontWeight: 600, whiteSpace: 'nowrap' }}>
                  正在管理：第 {mediaPageNum || '—'} 页的{mediaTab === 'video' ? '视频' : '图片'}
                </span>
                {mediaPageNum > 0 && (
                  <button
                    onClick={async () => {
                      if (!id || mediaPageNum <= 0) return
                      try {
                        const res = await listPageAssets(id, mediaPageNum)
                        setMediaAssets(res.assets || [])
                      } catch { setMediaAssets([]) }
                    }}
                    style={{ padding: '10px 16px', borderRadius: 8, border: '1px solid ' + C.border, background: '#fff', color: C.textSecondary, fontSize: 13, cursor: 'pointer' }}
                  >
                    🔄 刷新列表
                  </button>
                )}
                <span style={{ fontSize: 12, color: C.textMuted }}>（切换上方预览页即可管理对应页的媒体）</span>
              </div>

              {mediaPageNum > 0 && mediaTab === 'image' && (
                <>
                {/* 轮3：每页图列表顶部常驻锚点缩略图条——锚点是课件级，会话内从 courseware 状态直接读，跨页无需请求 */}
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12, padding: '8px 12px', borderRadius: 10, background: courseware.style_anchor_asset_id ? 'rgba(245,158,11,0.07)' : '#FAFAFA', border: '1px solid ' + (courseware.style_anchor_asset_id ? 'rgba(245,158,11,0.3)' : C.border) }}>
                  {courseware.style_anchor_asset_id ? (
                    <>
                      {courseware.style_anchor_url && (
                        <img src={courseware.style_anchor_url} alt="风格锚点" onClick={() => setMediaPreviewUrl(courseware.style_anchor_url)} style={{ width: 44, height: 44, objectFit: 'cover', borderRadius: 8, border: '2px solid #F59E0B', cursor: 'pointer', flexShrink: 0 }} title="点击查看锚点大图" />
                      )}
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontSize: 12, fontWeight: 600, color: '#B45309' }}>⭐ 当前风格锚点</div>
                        <div style={{ fontSize: 11, color: C.textMuted, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>本页及全课件配图将自动参考此图，保持风格与人物一致</div>
                      </div>
                      <button onClick={handleClearAnchor} disabled={anchorClearing}
                        style={{ padding: '4px 12px', borderRadius: 6, border: '1px solid ' + C.danger, background: 'transparent', color: C.danger, fontSize: 12, cursor: anchorClearing ? 'default' : 'pointer', whiteSpace: 'nowrap', flexShrink: 0 }}>
                        {anchorClearing ? '清除中...' : '✕ 清除锚点'}
                      </button>
                    </>
                  ) : (
                    <div style={{ fontSize: 12, color: C.textMuted, lineHeight: 1.5 }}>
                      💡 还未设风格锚点。在下方任意图片上点「⭐设为锚点」，之后全课件配图都会自动参考它，保持<b>画风与人物形象一致</b>。
                    </div>
                  )}
                </div>
                {/* 上半: 左右两栏 —— 左=AI配图建议(Tab切换), 右=生成图片(含风格快选) */}
                <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>

                  {/* 左栏: AI 配图建议(进页面自动出; 多条用 Tab 切换, 紧凑) */}
                  <div style={{ flex: '1 1 320px', padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, flexWrap: 'wrap', gap: 6 }}>
                      <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>✨ AI 配图建议</div>
                      <button onClick={handleSuggestImagePrompt} disabled={mediaPromptSuggesting || mediaGenerating || imgSuggestLoading}
                        style={{ padding: '4px 12px', borderRadius: 6, border: '1px dashed #7C3AED', background: 'rgba(124,58,237,0.04)', color: '#7C3AED', fontSize: 12, fontWeight: 600, cursor: (mediaPromptSuggesting || mediaGenerating || imgSuggestLoading) ? 'default' : 'pointer' }}>
                        {mediaPromptSuggesting ? '⏳ 撰写中...' : '🔄 重新生成'}
                      </button>
                    </div>

                    {imgSuggestLoading && (
                      <div style={{ padding: '16px 12px', borderRadius: 8, background: 'rgba(124,58,237,0.04)', color: '#7C3AED', fontSize: 12, textAlign: 'center' }}>⏳ AI 正在按本页方案生成配图建议...</div>
                    )}

                    {!imgSuggestLoading && imgSuggestions.length === 0 && (
                      <div style={{ padding: '16px 12px', borderRadius: 8, background: '#FAFAFA', color: C.textMuted, fontSize: 12, textAlign: 'center', lineHeight: 1.6 }}>暂无配图建议<br/>本页可能无图片占位（用SVG/CSS自绘）。<br/>可点「🔄 重新生成」，或在右侧手动描述生成。</div>
                    )}

                    {imgSuggestions.length >= 1 && (() => {
                      // Tab 越界保护(重新生成后条数变少时)
                      const safeTab = activeImgSuggestTab < imgSuggestions.length ? activeImgSuggestTab : 0
                      const it = imgSuggestions[safeTab]
                      return (
                      <div>
                        <div style={{ fontSize: 12, color: C.textSecondary, marginBottom: 8 }}>AI 建议本页配 {imgSuggestions.length} 张图（{imgSuggestions.length > 1 ? '切 Tab 查看，勾选可批量生成' : '可填入右侧微调后生成'}）· 已勾选 {imgSuggestSelected.size}/{imgSuggestions.length}</div>

                        {/* Tab 标签条(每个标签带勾选框 — 选甲: 切着看, 勾着选) */}
                        {imgSuggestions.length > 1 && (
                          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 10 }}>
                            {imgSuggestions.map((s, idx) => {
                              const active = idx === safeTab
                              const checked = imgSuggestSelected.has(idx)
                              return (
                                <div key={idx} onClick={() => setActiveImgSuggestTab(idx)}
                                  style={{ display: 'flex', alignItems: 'center', gap: 5, padding: '5px 10px', borderRadius: 8, cursor: 'pointer', border: '1px solid ' + (active ? '#7C3AED' : C.border), background: active ? 'rgba(124,58,237,0.08)' : '#fff' }}>
                                  <input type="checkbox" checked={checked} onChange={(e) => { e.stopPropagation(); toggleImgSuggest(idx) }} onClick={(e) => e.stopPropagation()} disabled={batchGenRunning}
                                    style={{ width: 14, height: 14, cursor: batchGenRunning ? 'default' : 'pointer', accentColor: '#7C3AED' }} />
                                  <span style={{ fontSize: 12, fontWeight: active ? 600 : 400, color: active ? '#7C3AED' : C.textSecondary }}>图{idx + 1}</span>
                                </div>
                              )
                            })}
                          </div>
                        )}

                        {/* 当前 Tab 的建议内容 */}
                        {it && (
                          <div style={{ padding: 10, borderRadius: 8, border: '1px solid ' + C.border, background: '#FAFAFA' }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                              {imgSuggestions.length === 1 && (
                                <input type="checkbox" checked={imgSuggestSelected.has(safeTab)} onChange={() => toggleImgSuggest(safeTab)} disabled={batchGenRunning}
                                  style={{ width: 15, height: 15, cursor: batchGenRunning ? 'default' : 'pointer', accentColor: '#7C3AED' }} />
                              )}
                              <span style={{ fontSize: 12, fontWeight: 600, color: C.textPrimary }}>{'图 ' + (safeTab + 1)}{it.caption ? '：' + it.caption : ''}</span>
                            </div>
                            <div style={{ fontSize: 12, color: C.textSecondary, lineHeight: 1.6, whiteSpace: 'pre-wrap', maxHeight: 180, overflowY: 'auto' }}>{it.prompt}</div>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8, flexWrap: 'wrap' }}>
                              <span style={{ fontSize: 11, color: '#6B7280' }}>尺寸:</span>
                              <select value={it.size} onChange={e => setImgSuggestSize(safeTab, e.target.value)} disabled={batchGenRunning}
                                style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid ' + C.border, fontSize: 11 }}>
                                <option value="1920x1920">1:1 正方形</option>
                                <option value="2560x1440">16:9 宽屏</option>
                                <option value="3072x1280">2.4:1 超宽</option>
                                <option value="1440x2560">9:16 竖屏</option>
                              </select>
                              <button onClick={() => fillImgSuggest(safeTab)} disabled={batchGenRunning}
                                style={{ padding: '4px 10px', borderRadius: 6, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: batchGenRunning ? 'default' : 'pointer' }}>
                                → 填入右侧
                              </button>
                            </div>
                          </div>
                        )}

                        {/* 批量生成进度条 */}
                        {batchGenRunning && batchGenProgress.total > 0 && (
                          <div style={{ marginTop: 10 }}>
                            <div style={{ height: 6, borderRadius: 3, background: '#EDE9FE', overflow: 'hidden' }}>
                              <div style={{ height: '100%', borderRadius: 3, transition: 'width 400ms', width: (batchGenProgress.done / batchGenProgress.total * 100) + '%', background: 'linear-gradient(90deg, #7C3AED, #6D28D9)' }} />
                            </div>
                            <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>已完成 {batchGenProgress.done} / {batchGenProgress.total} 张</div>
                          </div>
                        )}

                        {/* 批量生成(选甲: 勾选多个 Tab 后批量) */}
                        {imgSuggestions.length > 1 && (
                          <div style={{ display: 'flex', gap: 8, marginTop: 10, flexWrap: 'wrap' }}>
                            <button onClick={handleBatchGenImages} disabled={batchGenRunning || mediaGenerating || imgSuggestSelected.size === 0}
                              style={{ padding: '8px 16px', borderRadius: 8, border: 'none', background: (!batchGenRunning && !mediaGenerating && imgSuggestSelected.size > 0) ? 'linear-gradient(135deg, #7C3AED, #6D28D9)' : '#E5E7EB', color: (!batchGenRunning && !mediaGenerating && imgSuggestSelected.size > 0) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (!batchGenRunning && !mediaGenerating && imgSuggestSelected.size > 0) ? 'pointer' : 'default' }}>
                              {batchGenRunning ? '⏳ 批量生成中...' : '🤖 批量生成勾选的 ' + imgSuggestSelected.size + ' 张'}
                            </button>
                            <button onClick={() => { setImgSuggestSelected(new Set()) }} disabled={batchGenRunning}
                              style={{ padding: '8px 14px', borderRadius: 8, border: '1px solid ' + C.border, background: '#fff', color: C.textSecondary, fontSize: 13, cursor: batchGenRunning ? 'default' : 'pointer' }}>
                              取消勾选
                            </button>
                          </div>
                        )}
                        <div style={{ marginTop: 6, fontSize: 11, color: '#9CA3AF' }}>💡 批量生成串行调用 AI，生成的图自动进入下方「本页图片」列表。</div>
                      </div>
                      )
                    })()}
                  </div>

                  {/* 右栏: 生成图片(风格快选 + 生成框 + 尺寸 + 参考图 + 生成按钮) */}
                  <div style={{ flex: '1 1 320px', padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
                    <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🤖 生成图片</div>

                    {/* 风格快选: 点选把风格描述融入提示词(同风格再点=取消, 换风格=替换)
                        遗留项③：已设风格锚点时，配图由后端自动套用锚点VAOCI，优先级 锚点 > 快选预设；
                        此时禁用快选并提示，避免双重风格指令打架 */}
                    <div style={{ marginBottom: 8 }}>
                      {courseware.style_anchor_asset_id ? (
                        <div style={{ padding: '8px 12px', borderRadius: 8, background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.3)', fontSize: 11, color: '#B45309', lineHeight: 1.6 }}>
                          ⭐ 已设风格锚点，配图将<b>自动套用锚点风格</b>以保持全课件统一，无需再选画面风格（如需手动指定风格，请先在顶部清除锚点）。
                        </div>
                      ) : (
                        <>
                          <div style={{ fontSize: 11, color: '#6B7280', marginBottom: 6 }}>🎨 画面风格（点选融入提示词，让出图更好看）:</div>
                          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                            {CW_IMG_STYLES.map(s => {
                              const on = mediaStyleKey === s.key
                              return (
                                <button key={s.key} onClick={() => toggleImgStyle(s.key)} disabled={mediaGenerating}
                                  style={{ padding: '5px 10px', borderRadius: 14, border: '1px solid ' + (on ? '#7C3AED' : C.border), background: on ? 'rgba(124,58,237,0.1)' : '#fff', color: on ? '#7C3AED' : C.textSecondary, fontSize: 11, fontWeight: on ? 600 : 400, cursor: mediaGenerating ? 'default' : 'pointer' }}>
                                  {s.label}
                                </button>
                              )
                            })}
                          </div>
                        </>
                      )}
                    </div>

                    <textarea
                      value={mediaGenPrompt}
                      onChange={e => setMediaGenPrompt(e.target.value)}
                      placeholder="从左侧建议点「→ 填入右侧」，或在此手动描述要生成的图片；可点上方风格按钮美化"
                      rows={5}
                      style={{ width: '100%', padding: '10px 12px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 13, resize: 'vertical', outline: 'none', boxSizing: 'border-box' }}
                      disabled={mediaGenerating}
                    />
                    <div style={{ marginTop: 8 }}>
                      <span style={{ fontSize: 12, color: '#6B7280', marginRight: 8 }}>图片比例:</span>
                      <select value={mediaSize} onChange={e => setMediaSize(e.target.value)}
                        style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid #E5E7EB', fontSize: 12 }}
                        disabled={mediaGenerating}>
                        <option value="1920x1920">1:1 正方形</option>
                        <option value="2560x1440">16:9 宽屏</option>
                        <option value="3072x1280">2.4:1 超宽</option>
                        <option value="1440x2560">9:16 竖屏</option>
                      </select>
                    </div>
                    <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                      <span style={{ fontSize: 12, color: '#6B7280' }}>参考图:</span>
                      {mediaRefUrl ? (
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          <img src={mediaRefUrl} alt="参考图" style={{ width: 40, height: 40, objectFit: 'cover', borderRadius: 6, border: '2px solid #7C3AED' }} />
                          <span style={{ fontSize: 11, color: '#7C3AED' }}>已选择参考图</span>
                          <button onClick={() => setMediaRefUrl('')} style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid #EF4444', background: 'transparent', color: '#EF4444', fontSize: 11, cursor: 'pointer' }}>取消</button>
                        </div>
                      ) : (
                        <>
                        <span style={{ fontSize: 11, color: '#9CA3AF' }}>无</span>
                        <button onClick={() => {
                          const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*'
                          inp.onchange = async (ev) => {
                            const f = (ev.target as HTMLInputElement).files?.[0]
                            if (!f || !id || mediaPageNum <= 0) return
                            if (f.size > 5 * 1024 * 1024) { setMediaMessage('❌ 参考图不能超过5MB'); return }
                            setMediaMessage('⏳ 上传参考图中...')
                            try {
                              const res = await uploadCWImage(id, mediaPageNum, f)
                              setMediaRefUrl(res.url)
                              setMediaAssets(prev => [{ id: res.asset_id, courseware_id: id!, page_id: null, placeholder_id: '', asset_type: 'image', generation_prompt: '', oss_url: res.url, file_size: res.file_size, mime_type: res.mime_type, status: 'uploaded', created_at: new Date().toISOString() }, ...prev])
                              setMediaMessage('✅ 参考图上传成功，已自动选为参考图')
                            } catch (e) { setMediaMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                          }; inp.click()
                        }} style={{ padding: '3px 10px', borderRadius: 5, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: 'pointer' }}>📤 上传参考图</button>
                        </>
                      )}
                    </div>
                    <button
                      onClick={async () => {
                        if (!id || mediaPageNum <= 0 || !mediaGenPrompt.trim() || mediaGenerating) return
                        setMediaGenerating(true); setMediaMessage('')
                        try {
                          const res = await generateCWImage(id, mediaPageNum, mediaGenPrompt.trim(), undefined, mediaSize, mediaRefUrl || undefined)
                          setMediaMessage('✅ 图片生成成功！')
                          setMediaAssets(prev => [{ id: res.asset_id, courseware_id: id, page_id: null, placeholder_id: '', asset_type: 'image', generation_prompt: mediaGenPrompt, oss_url: res.url, file_size: 0, mime_type: 'image/png', status: 'uploaded', created_at: new Date().toISOString() }, ...prev])
                        } catch (e) { setMediaMessage('❌ 生成失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                        finally { setMediaGenerating(false) }
                      }}
                      disabled={mediaGenerating || !mediaGenPrompt.trim()}
                      style={{ marginTop: 8, padding: '10px 20px', borderRadius: 8, border: 'none', background: mediaGenPrompt.trim() && !mediaGenerating ? 'linear-gradient(135deg, #7C3AED, #6D28D9)' : '#E5E7EB', color: mediaGenPrompt.trim() && !mediaGenerating ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: mediaGenPrompt.trim() && !mediaGenerating ? 'pointer' : 'default', width: '100%' }}
                    >
                      {mediaGenerating ? '⏳ AI生成中（约10-30秒）...' : '🤖 生成图片'}
                    </button>
                  </div>
                </div>

                {/* 下半: 手动上传(通栏) */}
                <div style={{ marginTop: 16, padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
                  <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>📤 手动上传图片</div>
                  <div style={{ padding: '20px 16px', borderRadius: 8, border: '2px dashed ' + C.border, textAlign: 'center', cursor: 'pointer', background: '#FAFAFA' }}
                    onClick={() => { const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*'; inp.onchange = async (ev) => { const f = (ev.target as HTMLInputElement).files?.[0]; if (!f || !id) return; if (f.size > 5 * 1024 * 1024) { setMediaMessage('❌ 图片不能超过5MB'); return } setMediaGenerating(true); setMediaMessage(''); try { const res = await uploadCWImage(id, mediaPageNum, f); setMediaMessage('✅ 上传成功！'); setMediaAssets(prev => [{ id: res.asset_id, courseware_id: id!, page_id: null, placeholder_id: '', asset_type: 'image', generation_prompt: '', oss_url: res.url, file_size: res.file_size, mime_type: res.mime_type, status: 'uploaded', created_at: new Date().toISOString() }, ...prev]) } catch (e) { setMediaMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setMediaGenerating(false) } }; inp.click() }}
                  >
                    <div style={{ fontSize: 28, marginBottom: 6 }}>📷</div>
                    <div style={{ fontSize: 13, color: C.textSecondary }}>点击选择图片</div>
                    <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>支持 JPG/PNG/WEBP/GIF/SVG，最大5MB</div>
                  </div>
                </div>
                </>
              )}

              {/* 提示消息 */}
              {mediaTab === 'image' && mediaMessage && <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, background: mediaMessage.startsWith('❌') ? '#FEE2E2' : '#D1FAE5', color: mediaMessage.startsWith('❌') ? '#DC2626' : '#059669', fontSize: 13 }}>{mediaMessage}</div>}

              {/* 已上传图片列表 */}
              {(mediaAssets.filter(a => a.asset_type === 'image').length > 0 || courseware.style_anchor_asset_id) && mediaTab === 'image' && (
                <div style={{ marginTop: 16 }}>
                  <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>📎 第 {mediaPageNum} 页的图片（{mediaAssets.filter(a => a.asset_type === 'image').length}张）</div>
                  <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
                    {/* 做法B：锚点图作为置顶特殊项，永远排在每页图片列表最前（读 courseware 会话缓存，跨页常驻）。
                        点「参考」即把锚点图塞进右栏参考图，手动图生图取用；下方本页 .map 已排除锚点图本身避免重复。 */}
                    {courseware.style_anchor_asset_id && courseware.style_anchor_url && (
                      <div key={'anchor-pinned'} style={{ width: 'calc(25% - 9px)', minWidth: 180, borderRadius: 10, border: '2px solid #F59E0B', overflow: 'hidden', background: 'rgba(245,158,11,0.04)' }}>
                        <img src={courseware.style_anchor_url} alt="风格锚点" onClick={() => setMediaPreviewUrl(courseware.style_anchor_url)} style={{ width: '100%', height: 140, objectFit: 'cover', display: 'block', cursor: 'pointer' }} title="点击查看锚点大图" />
                        <div style={{ padding: '8px 10px' }}>
                          <div style={{ fontSize: 11, color: '#B45309', fontWeight: 600, marginBottom: 6 }}>★ 课件风格锚点</div>
                          <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                            <button
                              onClick={() => { setMediaRefUrl(courseware.style_anchor_url); setMediaMessage('✅ 已选锚点图为参考图，本次生成将参考锚点风格与人物') }}
                              style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: 'pointer' }}
                            >参考</button>
                            <button onClick={handleClearAnchor} disabled={anchorClearing}
                              style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid ' + C.danger, background: 'transparent', color: C.danger, fontSize: 11, cursor: anchorClearing ? 'default' : 'pointer' }}
                            >{anchorClearing ? '清除中' : '✕清除锚点'}</button>
                          </div>
                        </div>
                      </div>
                    )}
                    {mediaAssets.filter(a => a.asset_type === 'image' && a.id !== courseware.style_anchor_asset_id).map(asset => (
                      <div key={asset.id} style={{ width: 'calc(25% - 9px)', minWidth: 180, borderRadius: 10, border: '1px solid ' + C.border, overflow: 'hidden', background: '#fff' }}>
                        <img src={asset.oss_url} alt="课件图片" onClick={() => setMediaPreviewUrl(asset.oss_url)} style={{ width: '100%', height: 140, objectFit: 'cover', display: 'block', cursor: 'pointer' }} title="点击查看大图" />
                        <div style={{ padding: '8px 10px' }}>
                          <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 6 }}>{asset.generation_prompt ? '🤖 AI生成' : '📤 手动上传'}{asset.public_oss_url ? ' · ☁️已上云' : ''}</div>
                          <div style={{ display: 'flex', gap: 4 }}>
                            
                            {/* 风格锚点（轮3）：当前锚点显示高亮态，否则可点击设为锚点 */}
                            {courseware.style_anchor_asset_id === asset.id ? (
                              <span style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #F59E0B', background: 'rgba(245,158,11,0.12)', color: '#B45309', fontSize: 11, fontWeight: 600, whiteSpace: 'nowrap' }} title="当前风格锚点">★ 锚点</span>
                            ) : (
                              <button
                                onClick={() => handleSetAnchor(asset.id)}
                                disabled={!!anchorSetting}
                                title="设为风格锚点：后续配图将自动参考此图风格，保持全课件视觉统一"
                                style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #F59E0B', background: 'rgba(245,158,11,0.06)', color: '#B45309', fontSize: 11, cursor: anchorSetting ? 'default' : 'pointer', whiteSpace: 'nowrap' }}
                              >{anchorSetting === asset.id ? '⏳设置中' : '⭐设为锚点'}</button>
                            )}
                            <button
                              onClick={() => { setMediaRefUrl(asset.oss_url); setMediaMessage('✅ 已选为参考图，AI将参考此图风格生成新图片') }}
                              style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: 'pointer' }}
                            >参考</button>
                            {asset.public_oss_url ? (
                              <button
                                onClick={async () => {
                                  try {
                                    await navigator.clipboard.writeText(asset.public_oss_url!)
                                    setMediaMessage('📋 云盘链接已复制到剪贴板')
                                  } catch { setMediaMessage('❌ 复制失败，请手动复制') }
                                }}
                                style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #059669', background: 'rgba(5,150,105,0.06)', color: '#059669', fontSize: 11, cursor: 'pointer' }}
                                title="已上传云盘，点击复制公网链接"
                              >📋 复制链接</button>
                            ) : (
                              <button
                                onClick={async () => {
                                  if (!id) return
                                  setMediaMessage('⏳ 正在上传到云盘...')
                                  try {
                                    const res = await uploadAssetToOSS(id, asset.id)
                                    await navigator.clipboard.writeText(res.oss_public_url)
                                    setMediaAssets(prev => prev.map(a => a.id === asset.id ? { ...a, public_oss_url: res.oss_public_url } : a))
                                    setMediaMessage('✅ 已上传云盘，链接已复制到剪贴板')
                                  } catch (e) { setMediaMessage('❌ 上传云盘失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                                }}
                                style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #0891B2', background: 'rgba(8,145,178,0.06)', color: '#0891B2', fontSize: 11, cursor: 'pointer' }}
                                title="上传到云盘获取公网链接"
                              >☁️云盘</button>
                            )}
                            <button
                              onClick={async () => {
                                if (!id) return
                                const warnMsg = asset.public_oss_url
                                  ? '⚠️ 这张图片已上传云盘。删除将同时移除云盘副本，若课件中已使用该云盘链接，图片将无法显示，且不可恢复。确定删除？'
                                  : '确定删除这张图片？'
                                if (!confirm(warnMsg)) return
                                try {
                                  await deleteCWAsset(id, asset.id)
                                  setMediaAssets(prev => prev.filter(a => a.id !== asset.id))
                                  setMediaMessage('✅ 已删除')
                                } catch (e) { setMediaMessage('❌ 删除失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                              }}
                              style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #EF4444', background: 'rgba(239,68,68,0.06)', color: '#EF4444', fontSize: 11, cursor: 'pointer' }}
                            >删除</button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              
              {/* 视频生成区: 抽成 VideoStoryboardPanel(AI分镜两步法), 见 components/VideoStoryboardPanel.tsx */}
              {mediaTab === 'video' && mediaPageNum > 0 && (
                <VideoStoryboardPanel
                  coursewareId={id!}
                  pageNum={mediaPageNum}
                  styleAnchorAssetId={courseware.style_anchor_asset_id}
                  onAssetCreated={(asset) => setMediaAssets(prev => prev.some(a => a.id === asset.id) ? prev : [asset, ...prev])}
                  onPreviewImage={(url) => setMediaPreviewUrl(url)}
                />
              )}

              {/* v0.42.1: 视频列表 + 拼接 + 裁剪（视频Tab下） */}
              {mediaTab === 'video' && mediaPageNum > 0 && mediaAssets.filter(a => a.asset_type === 'video').length > 0 && (
                <div style={{ marginTop: 16 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                    <span style={{ fontSize: 13, fontWeight: 600, color: '#7C3AED' }}>🎬 第 {mediaPageNum} 页的视频（{mediaAssets.filter(a => a.asset_type === 'video').length}个）</span>
                    <button onClick={() => setEditorOpen(true)} style={{ padding: '6px 14px', borderRadius: 8, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 12, fontWeight: 600, cursor: 'pointer' }}>🎬 打开视频编辑器</button>
                  </div>
                  <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
                    {mediaAssets.filter(a => a.asset_type === 'video').map(asset => (
                      <div key={asset.id} style={{ width: 240, borderRadius: 10, border: '1px solid ' + C.border, overflow: 'hidden', background: '#fff' }}>
                        <video src={asset.oss_url} controls style={{ width: '100%', height: 135, display: 'block', background: '#000', objectFit: 'contain' }} />
                        <div style={{ padding: '8px 10px' }}>
                          <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 4 }}>{asset.generation_prompt ? '🤖 ' + asset.generation_prompt.slice(0,25) + (asset.generation_prompt.length > 25 ? '...' : '') : '📤 手动上传'}{asset.public_oss_url ? ' · ☁️已上云' : ''}</div>
                          <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                            
                            {asset.public_oss_url ? (
                              <button onClick={async () => {
                                try {
                                  await navigator.clipboard.writeText(asset.public_oss_url!)
                                  setMediaMessage('📋 云盘链接已复制到剪贴板')
                                } catch { setMediaMessage('❌ 复制失败，请手动复制') }
                              }} style={{ padding: '3px 8px', borderRadius: 5, border: '1px solid #059669', background: 'rgba(5,150,105,0.06)', color: '#059669', fontSize: 10, cursor: 'pointer' }} title="已上传云盘，点击复制公网链接">📋 复制链接</button>
                            ) : (
                              <button onClick={async () => {
                                if (!id) return
                                setMediaMessage('⏳ 正在上传到云盘...')
                                try {
                                  const res = await uploadAssetToOSS(id, asset.id)
                                  await navigator.clipboard.writeText(res.oss_public_url)
                                  setMediaAssets(prev => prev.map(a => a.id === asset.id ? { ...a, public_oss_url: res.oss_public_url } : a))
                                  setMediaMessage('✅ 已上传云盘，链接已复制到剪贴板')
                                } catch (e) { setMediaMessage('❌ 上传云盘失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                              }} style={{ padding: '3px 8px', borderRadius: 5, border: '1px solid #0891B2', background: 'rgba(8,145,178,0.06)', color: '#0891B2', fontSize: 10, cursor: 'pointer' }} title="上传到云盘获取公网链接">☁️云盘</button>
                            )}
                            <button onClick={async () => {
                                if (!id) return
                                const warnMsg = asset.public_oss_url
                                  ? '⚠️ 这个视频已上传云盘。删除将同时移除云盘副本，若课件中已使用该云盘链接，视频将无法播放，且不可恢复。确定删除？'
                                  : '确定删除此视频？'
                                if (!confirm(warnMsg)) return
                                try { await deleteCWAsset(id, asset.id); setMediaAssets(prev => prev.filter(a => a.id !== asset.id)); setMediaMessage('✅ 已删除') } catch (e) { setMediaMessage('❌ 删除失败: ' + (e instanceof Error ? e.message : '')) }
                              }} style={{ padding: '3px 8px', borderRadius: 5, border: '1px solid #EF4444', background: 'rgba(239,68,68,0.06)', color: '#EF4444', fontSize: 10, cursor: 'pointer' }}>删除</button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>


                </div>
              )}


              {/* 图片大图预览弹窗 */}
              {mediaPreviewUrl && (
                <div onClick={() => setMediaPreviewUrl('')} style={{ position: 'fixed', inset: 0, zIndex: 99990, background: 'rgba(0,0,0,0.75)', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'zoom-out' }}>
                  <img src={mediaPreviewUrl} alt="大图预览" style={{ maxWidth: '90vw', maxHeight: '90vh', borderRadius: 12, boxShadow: '0 8px 40px rgba(0,0,0,0.5)' }} />
                  <button onClick={(e) => { e.stopPropagation(); setMediaPreviewUrl('') }} style={{ position: 'absolute', top: 24, right: 24, width: 40, height: 40, borderRadius: '50%', border: 'none', background: 'rgba(255,255,255,0.2)', color: '#fff', fontSize: 20, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✕</button>
                </div>
              )}
            </div>
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

      {/* 视频编辑器弹窗（类剪映多片段时间轴编辑） */}
      {editorOpen && (
        <VideoEditorModal
          coursewareId={id!}
          videos={mediaAssets.filter(a => a.asset_type === 'video' && a.oss_url).map(a => ({
            id: a.id,
            url: a.oss_url,
            label: a.generation_prompt || a.oss_url.split('/').pop()?.slice(0, 30) || '视频',
          }))}
          exporting={editorExporting}
          onClose={() => setEditorOpen(false)}
          onUploadVideo={async (file, onProgress) => {
            if (!id || mediaPageNum <= 0) return null
            const res = await uploadCWVideo(id, mediaPageNum, file, onProgress)
            const newAsset = {
              id: res.asset_id, courseware_id: id, page_id: null, placeholder_id: '',
              asset_type: 'video' as const, generation_prompt: file.name,
              oss_url: res.url, file_size: res.file_size, mime_type: res.mime_type,
              status: 'uploaded', created_at: new Date().toISOString(),
            }
            setMediaAssets(prev => [newAsset, ...prev])
            return { id: res.asset_id, url: res.url, label: file.name }
          }}
          onExport={async (clips) => {
            if (!id || editorExporting) return
            setEditorExporting(true); setMediaMessage('')
            try {
              const res = await advancedConcatCWVideos(id, clips)
              setMediaMessage('\u2705 ' + res.message)
              setMediaAssets(prev => [{
                id: res.asset_id, courseware_id: id!, page_id: null, placeholder_id: '',
                asset_type: 'video', generation_prompt: '\u7f16\u8f91\u5bfc\u51fa ' + clips.length + '\u4e2a\u7247\u6bb5',
                oss_url: res.url, file_size: 0, mime_type: 'video/mp4', status: 'uploaded',
                created_at: new Date().toISOString(),
              }, ...prev])
              setEditorOpen(false)
            } catch (e) {
              setMediaMessage('\u274c \u5bfc\u51fa\u5931\u8d25: ' + (e instanceof Error ? e.message : ''))
            } finally { setEditorExporting(false) }
          }}
        />
      )}

    </div>
  )
}

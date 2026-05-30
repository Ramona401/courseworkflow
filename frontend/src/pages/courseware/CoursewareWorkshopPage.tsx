/**
 * 课件工坊主页面 — CoursewareWorkshopPage.tsx v5.3 (v0.41: 预览降级注入)
 *
 * v5.3 (v0.41) 新增：
 *   - injectPreviewMode()：统一为所有iframe的srcDoc注入预览环境降级脚本
 *   - 拦截课件HTML中的 /api/* fetch调用，返回降级JSON响应
 *   - 降级时课件内的API区域显示蓝色信息条而非红色错误
 *   - 设置 window.__TEDNA_PREVIEW_MODE__ = true 供课件JS检测
 *
 * v5.2 修复：
 *   - SlideshowPlayer全屏背景从黑色改为白色，消除上下黑框
 *   - 去掉手动绘制的黑色填充div
 *   - iframe添加scrolling="no"消除内部滚动条
 *   - 外层容器overflow:hidden消除外部滚动条
 *   - 缩放策略优化：优先以宽度填满，高度按比例自适应
 *
 * v5.1 P0-1改造：
 *   - Step 3: AI只生成1页封面预览（不再是2页）
 *   - 去掉前端extractNavFromHTML（导航栏由后端自动从封面页按标记提取）
 *   - 确认导航栏时传"auto"给后端，后端自动提取+替换页码占位符
 *   - Step 4: 批量生成（后端硬拼接导航栏，AI只生成内容区）
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
  generateCWVideo, queryVideoStatus,
  advancedConcatCWVideos, uploadCWVideo, uploadAssetToOSS,
} from '@/api/coursewares'
import type { SchemePreset } from '@/api/coursewares'
import type { CoursewareDetail, CoursewarePage } from '@/api/coursewares'
import IndexEditor from './components/IndexEditor'
import StyleSelector from './components/StyleSelector'
import { useAuth } from '@/store/auth'
import VideoEditorModal from './components/VideoEditorModal'
import ThreeDSingleView from './components/ThreeDSingleView'

// ==================== 常量 ====================
const C = {
  primary: '#F59E0B', primaryBg: 'rgba(245,158,11,0.08)',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
  border: '#E5E7EB', success: '#059669', white: '#fff', danger: '#EF4444',
}

const CW_WIDTH = 1920
const CW_HEIGHT = 1080

const STEPS = [
  { key: 'generate', label: 'AI生成方案', emoji: '🤖', desc: 'AI分析教案，生成页面方案' },
  { key: 'edit',     label: '确认方案',   emoji: '✏️', desc: '确认每页课件内容设计' },
  { key: 'style',    label: '选择风格',   emoji: '🎨', desc: '选择视觉风格和配色' },
  { key: 'preview',  label: '确认导航栏', emoji: '🧭', desc: '确认导航栏样式并固定' },
  { key: 'build',    label: '批量生成',   emoji: '⚡', desc: '用固定导航栏生成全部页面' },
  { key: 'confirm',  label: '确认提交',   emoji: '✅', desc: '预览课件效果，确认提交' },
]

function statusToStep(s: string, hasNavTemplate: boolean, hasPreviewPages: boolean): number {
  if (s === 'draft') return 0
  if (s === 'indexing') return 1
  if (s === 'styling') return 2
  if (s === 'generating') {
    if (hasNavTemplate) return 4
    if (hasPreviewPages) return 3
    return 3
  }
  if (s === 'preview') return 5
  if (s === 'confirmed' || s === 'in_pipeline') return 5
  return 0
}

// ==================== v0.41: 预览降级注入 ====================
// 在TE-DNA的iframe预览中，课件HTML调用 /api/* 会失败（edu平台API不可用）
// 此函数为srcDoc注入一段脚本，拦截fetch请求并返回降级响应
// 课件内的降级代码检测到 _preview_mode: true 时显示蓝色信息条而非红色错误
const PREVIEW_INJECT_SCRIPT = `
<script>
// TE-DNA 预览模式标记
window.__TEDNA_PREVIEW_MODE__ = true;

// 拦截 fetch，对 /api/* 请求返回降级响应
(function() {
  var originalFetch = window.fetch;
  window.fetch = function(url, options) {
    if (typeof url === 'string' && url.startsWith('/api/')) {
      console.log('[预览模式] API 调用已降级:', url);
      return Promise.resolve(new Response(JSON.stringify({
        success: false,
        _preview_mode: true,
        message: '预览模式下 API 不可用，请在授课平台查看完整效果',
        data: { content: '预览模式：AI功能需在授课平台使用', audio_url: '' }
      }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }));
    }
    return originalFetch.call(this, url, options);
  };
})();
</script>
`;

/**
 * injectPreviewMode — 为课件HTML注入预览降级脚本
 * 在 <head> 或第一个 <script> 前注入，确保在课件JS执行前生效
 * 如果HTML不包含 /api/ 调用（纯展示页），注入也不会产生副作用
 */
function injectPreviewMode(html: string): string {
  if (!html || !html.trim()) return html

  // 策略1: 如果有 </head>，在其前面注入
  const headClose = html.indexOf('</head>')
  if (headClose >= 0) {
    return html.slice(0, headClose) + PREVIEW_INJECT_SCRIPT + html.slice(headClose)
  }

  // 策略2: 如果有 <script，在第一个 <script 前注入
  const firstScript = html.indexOf('<script')
  if (firstScript >= 0) {
    return html.slice(0, firstScript) + PREVIEW_INJECT_SCRIPT + html.slice(firstScript)
  }

  // 策略3: 在HTML最前面注入（兜底）
  return PREVIEW_INJECT_SCRIPT + html
}

// ==================== 全屏幻灯片放映（v5.2重写：白色背景+无滚动条） ====================
// ==================== v137: 全屏预览组件（带工具栏+键盘导航+resize响应） ====================
function CWFullscreenPreview({ pages, initialPageNum, codeView, onToggleCode, onClose, onSlideshow }: {
  pages: { page_number: number; title: string; html_content: string }[]
  initialPageNum: number
  codeView: boolean
  onToggleCode: () => void
  onClose: () => void
  onSlideshow: (pn: number) => void
}) {
  const [curPageNum, setCurPageNum] = useState(initialPageNum)
  const [viewSize, setViewSize] = useState({ w: window.innerWidth, h: window.innerHeight })

  const idx = pages.findIndex(p => p.page_number === curPageNum)
  const page = pages[idx] || pages[0]
  const html = page?.html_content || ''

  // v0.41: 注入预览降级脚本
  const previewHtml = injectPreviewMode(html)

  const hasPrev = idx > 0
  const hasNext = idx < pages.length - 1

  // 响应窗口resize
  useEffect(() => {
    const fn = () => setViewSize({ w: window.innerWidth, h: window.innerHeight })
    window.addEventListener('resize', fn)
    return () => window.removeEventListener('resize', fn)
  }, [])

  // 键盘导航：左右箭头翻页 + ESC退出
  useEffect(() => {
    const fn = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { onClose(); return }
      if (e.key === 'ArrowLeft' && hasPrev) setCurPageNum(pages[idx - 1].page_number)
      if (e.key === 'ArrowRight' && hasNext) setCurPageNum(pages[idx + 1].page_number)
    }
    window.addEventListener('keydown', fn)
    return () => window.removeEventListener('keydown', fn)
  }, [idx, hasPrev, hasNext, pages, onClose])

  // 缩放计算：工具栏高度60px，内容区占满剩余空间
  const toolbarH = 60
  const contentH = viewSize.h - toolbarH
  const scale = Math.min(viewSize.w / CW_WIDTH, contentH / CW_HEIGHT)
  const scaledW = CW_WIDTH * scale
  const scaledH = CW_HEIGHT * scale
  const ox = (viewSize.w - scaledW) / 2
  const oy = (contentH - scaledH) / 2

  const tbtn: React.CSSProperties = { padding: '6px 14px', borderRadius: 8, border: '1px solid #E5E7EB', background: '#fff', color: '#6B7280', fontSize: 13, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 4 }

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 99998, background: '#fff', display: 'flex', flexDirection: 'column' }}>
      {/* 工具栏 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 20px', background: 'rgba(255,255,255,0.95)', borderBottom: '1px solid rgba(0,0,0,0.08)', flexShrink: 0, height: toolbarH, boxSizing: 'border-box' }}>
        <button onClick={() => hasPrev && setCurPageNum(pages[idx - 1].page_number)} disabled={!hasPrev} style={{ ...tbtn, opacity: hasPrev ? 1 : 0.3, cursor: hasPrev ? 'pointer' : 'not-allowed' }}>‹</button>
        <span style={{ fontSize: 14, fontWeight: 600, color: '#1F2937' }}>P{page?.page_number} — {page?.title}</span>
        <span style={{ fontSize: 12, color: '#9CA3AF' }}>{(idx >= 0 ? idx : 0) + 1}/{pages.length}</span>
        <button onClick={() => hasNext && setCurPageNum(pages[idx + 1].page_number)} disabled={!hasNext} style={{ ...tbtn, opacity: hasNext ? 1 : 0.3, cursor: hasNext ? 'pointer' : 'not-allowed' }}>›</button>
        <div style={{ flex: 1 }} />
        <button onClick={onToggleCode} style={{ ...tbtn, border: `1px solid ${codeView ? '#7C3AED' : '#E5E7EB'}`, background: codeView ? 'rgba(124,58,237,0.06)' : '#fff', color: codeView ? '#7C3AED' : '#6B7280' }}>{codeView ? '📺 预览' : '💻 代码'}</button>
        <button onClick={() => { navigator.clipboard.writeText(html).then(() => alert('已复制')).catch(() => {}) }} style={tbtn}>📋 复制</button>
        <button onClick={() => onSlideshow(page?.page_number || 1)} style={{ ...tbtn, border: '1px solid #F59E0B', background: 'rgba(245,158,11,0.06)', color: '#F59E0B' }}>🖥️ 放映</button>
        <button onClick={onClose} style={tbtn}>✕ 退出</button>
      </div>
      {/* 内容区 */}
      <div style={{ flex: 1, overflow: 'hidden', position: 'relative', background: codeView ? '#1e1e1e' : '#fff' }}>
        {codeView ? (
          <div style={{ width: '100%', height: '100%', overflow: 'auto', fontFamily: 'Monaco, Consolas, "Courier New", monospace', fontSize: 13, lineHeight: 1.7 }}>
            <table style={{ borderCollapse: 'collapse', width: '100%' }}><tbody>
              {html.split('\n').map((line: string, i: number) => (
                <tr key={i}><td style={{ width: 55, minWidth: 55, textAlign: 'right', padding: '0 10px 0 8px', color: '#858585', userSelect: 'none', verticalAlign: 'top', borderRight: '1px solid #333', whiteSpace: 'nowrap' }}>{i + 1}</td><td style={{ padding: '0 12px', color: '#d4d4d4', whiteSpace: 'pre', wordBreak: 'break-all' }}>{line || ' '}</td></tr>
              ))}
            </tbody></table>
          </div>
        ) : (
          <div style={{ position: 'absolute', left: ox, top: oy, width: CW_WIDTH, height: CW_HEIGHT, transform: `scale(${scale})`, transformOrigin: 'top left' }}>
            <iframe srcDoc={previewHtml} scrolling="no" style={{ width: CW_WIDTH, height: CW_HEIGHT, border: 'none', display: 'block', overflow: 'hidden' }} sandbox="allow-scripts" title={`全屏预览-P${page?.page_number}`} />
          </div>
        )}
      </div>
    </div>
  )
}

function SlideshowPlayer({ pages, initialPage, onClose }: {
  pages: { page_number: number; title: string; html_content: string }[]
  initialPage: number
  onClose: () => void
}) {
  const [curPage, setCurPage] = useState(initialPage)
  const [showUI, setShowUI] = useState(true)
  // v5.2: 使用innerWidth/innerHeight作为默认值，全屏后切换为screen尺寸
  const [containerSize, setContainerSize] = useState({ w: window.innerWidth, h: window.innerHeight })
  const uiTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const boxRef = useRef<HTMLDivElement>(null)

  const data = pages.find(p => p.page_number === curPage)
  const idx = pages.findIndex(p => p.page_number === curPage)
  const hasPrev = idx > 0
  const hasNext = idx < pages.length - 1

  // v0.41: 注入预览降级脚本
  const previewHtml = data ? injectPreviewMode(data.html_content) : ''

  // 请求浏览器全屏API
  useEffect(() => {
    const el = boxRef.current
    if (el?.requestFullscreen) el.requestFullscreen().catch(() => {})
    const onFs = () => {
      if (!document.fullscreenElement) {
        onClose()
      } else {
        // 全屏后延迟获取准确尺寸
        requestAnimationFrame(() => {
          setTimeout(() => {
            setContainerSize({ w: screen.width, h: screen.height })
          }, 100)
        })
      }
    }
    document.addEventListener('fullscreenchange', onFs)
    return () => {
      document.removeEventListener('fullscreenchange', onFs)
      if (document.fullscreenElement) document.exitFullscreen().catch(() => {})
    }
  }, [onClose])

  // 监听resize事件，实时更新容器尺寸
  useEffect(() => {
    const fn = () => {
      if (document.fullscreenElement) {
        setContainerSize({ w: screen.width, h: screen.height })
      } else {
        setContainerSize({ w: window.innerWidth, h: window.innerHeight })
      }
    }
    window.addEventListener('resize', fn)
    const t = setTimeout(fn, 500)
    return () => { window.removeEventListener('resize', fn); clearTimeout(t) }
  }, [])

  // 键盘导航
  useEffect(() => {
    const fn = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { onClose(); return }
      if ((e.key === 'ArrowLeft' || e.key === 'PageUp') && hasPrev) setCurPage(pages[idx - 1].page_number)
      if ((e.key === 'ArrowRight' || e.key === 'PageDown' || e.key === ' ') && hasNext) { e.preventDefault(); setCurPage(pages[idx + 1].page_number) }
      flashUI()
    }
    window.addEventListener('keydown', fn)
    return () => window.removeEventListener('keydown', fn)
  }, [curPage, idx, hasPrev, hasNext, pages, onClose])

  // UI自动隐藏定时器
  const flashUI = () => {
    setShowUI(true)
    if (uiTimer.current) clearTimeout(uiTimer.current)
    uiTimer.current = setTimeout(() => setShowUI(false), 3000)
  }
  useEffect(() => {
    uiTimer.current = setTimeout(() => setShowUI(false), 3000)
    return () => { if (uiTimer.current) clearTimeout(uiTimer.current) }
  }, [])

  if (!data) return null

  // v5.2: 缩放计算 — 以宽度为基准，确保内容完全可见无黑框
  const scale = Math.min(containerSize.w / CW_WIDTH, containerSize.h / CW_HEIGHT)
  const scaledW = CW_WIDTH * scale
  const scaledH = CW_HEIGHT * scale
  // 居中偏移（水平和垂直）
  const ox = (containerSize.w - scaledW) / 2
  const oy = (containerSize.h - scaledH) / 2

  return (
    <div ref={boxRef} data-slideshow="1" onMouseMove={flashUI}
      onClick={(e) => {
        const r = boxRef.current?.getBoundingClientRect()
        if (!r) return
        const x = e.clientX - r.left
        if (x < r.width * 0.25 && hasPrev) setCurPage(pages[idx - 1].page_number)
        else if (x > r.width * 0.75 && hasNext) setCurPage(pages[idx + 1].page_number)
        flashUI()
      }}
      style={{
        position: 'fixed', inset: 0, zIndex: 99999,
        background: '#fff',              /* v5.2: 白色背景替代黑色 */
        cursor: showUI ? 'default' : 'none',
        overflow: 'hidden',              /* v5.2: 消除外层滚动条 */
      }}>
      {/* 课件内容区域：绝对定位+缩放居中 */}
      <div style={{
        position: 'absolute',
        left: ox, top: oy,
        width: CW_WIDTH, height: CW_HEIGHT,
        transform: `scale(${scale})`,
        transformOrigin: 'top left',
      }}>
        <iframe
          srcDoc={previewHtml}
          scrolling="no"                  /* v5.2: 禁用iframe内部滚动条 */
          style={{
            width: CW_WIDTH, height: CW_HEIGHT,
            border: 'none', display: 'block',
            overflow: 'hidden',           /* v5.2: 双重保险消除iframe滚动条 */
          }}
          sandbox="allow-scripts"
          title={`放映-第${curPage}页`}
        />
      </div>

      {/* v5.2: 去掉了手动绘制的黑色填充div，白色背景自然融合 */}

      {/* 底部控制条（自动隐藏） */}
      <div style={{ position: 'absolute', inset: 0, zIndex: 2, pointerEvents: 'none', opacity: showUI ? 1 : 0, transition: 'opacity 400ms' }}>
        <div style={{
          position: 'absolute', bottom: 24, left: '50%', transform: 'translateX(-50%)',
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '10px 20px', borderRadius: 999,
          background: 'rgba(0,0,0,0.65)', backdropFilter: 'blur(12px)',
          pointerEvents: 'auto',
        }}>
          {/* 上一页按钮 */}
          <button onClick={e => { e.stopPropagation(); if (hasPrev) setCurPage(pages[idx - 1].page_number) }}
            style={{ width: 36, height: 36, borderRadius: '50%', border: 'none',
              background: hasPrev ? 'rgba(255,255,255,0.15)' : 'transparent',
              color: hasPrev ? '#fff' : 'rgba(255,255,255,0.2)',
              fontSize: 20, cursor: hasPrev ? 'pointer' : 'default',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>‹</button>

          {/* 页面圆点指示器 */}
          <div style={{ display: 'flex', gap: 5, padding: '0 8px' }}>
            {pages.map(p => (
              <button key={p.page_number} onClick={e => { e.stopPropagation(); setCurPage(p.page_number) }}
                style={{
                  width: p.page_number === curPage ? 28 : 10, height: 10, borderRadius: 5,
                  border: 'none', cursor: 'pointer', transition: 'all 250ms',
                  background: p.page_number === curPage ? C.primary : 'rgba(255,255,255,0.3)',
                }}
                title={`第${p.page_number}页: ${p.title}`} />
            ))}
          </div>

          {/* 下一页按钮 */}
          <button onClick={e => { e.stopPropagation(); if (hasNext) setCurPage(pages[idx + 1].page_number) }}
            style={{ width: 36, height: 36, borderRadius: '50%', border: 'none',
              background: hasNext ? 'rgba(255,255,255,0.15)' : 'transparent',
              color: hasNext ? '#fff' : 'rgba(255,255,255,0.2)',
              fontSize: 20, cursor: hasNext ? 'pointer' : 'default',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>›</button>

          {/* 页码显示 */}
          <div style={{ color: 'rgba(255,255,255,0.5)', fontSize: 13, fontWeight: 600, minWidth: 54, textAlign: 'center' }}>
            {curPage} / {pages.length}
          </div>

          {/* 退出按钮 */}
          <button onClick={e => { e.stopPropagation(); onClose() }}
            style={{ width: 36, height: 36, borderRadius: '50%', border: 'none',
              background: 'rgba(255,255,255,0.12)', color: '#fff',
              fontSize: 18, cursor: 'pointer',
              display: 'flex', alignItems: 'center', justifyContent: 'center', marginLeft: 4,
            }} title="退出 (ESC)">×</button>
        </div>
      </div>
    </div>
  )
}

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

  // P0-4: 页面微调状态
  const [refinePageNum, setRefinePageNum] = useState(0)
  const [refineInput, setRefineInput] = useState('')
  const [refineRunning, setRefineRunning] = useState(false)

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
  // v0.42 多媒体: 媒体管理状态
  const [mediaPageNum, setMediaPageNum] = useState(0)
  const [mediaAssets, setMediaAssets] = useState<import('@/api/coursewares').CoursewareAsset[]>([])
  const [mediaGenPrompt, setMediaGenPrompt] = useState('')
  const [mediaSize, setMediaSize] = useState('1920x1920')
  const [mediaGenerating, setMediaGenerating] = useState(false)
  const [mediaMessage, setMediaMessage] = useState('')
  const [mediaPreviewUrl, setMediaPreviewUrl] = useState('')
  const [mediaRefUrl, setMediaRefUrl] = useState('')  // 参考图URL（图生图）
  // v0.42.1 视频生成状态
  const [mediaTab, setMediaTab] = useState<'image'|'video'>('image')  // 多媒体Tab切换
  const [videoPrompt, setVideoPrompt] = useState('')
  const [videoRefUrl, setVideoRefUrl] = useState('')  // 视频参考图URL
  const [videoGenerating, setVideoGenerating] = useState(false)
  const [, setVideoTaskId] = useState('') // eslint-disable-line @typescript-eslint/no-unused-vars
  const [videoAssetId, setVideoAssetId] = useState('')
  const [videoPolling, setVideoPolling] = useState(false)
  const [videoMessage, setVideoMessage] = useState('')
  const [videoResult, setVideoResult] = useState<{url:string;duration:number;resolution:string}|null>(null)
  // v0.42.1 视频编辑状态
  const [editorOpen, setEditorOpen] = useState(false)         // 视频编辑器弹窗
  const [editorExporting, setEditorExporting] = useState(false) // 编辑器导出中

  const sseRef = useRef<{ close: () => void } | null>(null)

  
  // v0.42.1: 视频生成轮询（每5秒查询一次，直到完成或失败）
  const videoPromptRef = useRef(videoPrompt)
  videoPromptRef.current = videoPrompt
  useEffect(() => {
    if (!videoPolling || !videoAssetId || !id) return
    const timer = setInterval(async () => {
      try {
        const res = await queryVideoStatus(id, videoAssetId)
        if (res.status === 'uploaded') {
          setVideoPolling(false)
          setVideoResult({ url: res.video_url, duration: res.duration, resolution: res.resolution })
          setVideoMessage('\u2705 ' + res.message)
          setMediaAssets(prev => {
            const exists = prev.some(a => a.id === videoAssetId)
            if (exists) return prev.map(a => a.id === videoAssetId ? { ...a, oss_url: res.video_url, status: 'uploaded' } : a)
            return [{ id: videoAssetId, courseware_id: id!, page_id: null, placeholder_id: '', asset_type: 'video', generation_prompt: videoPromptRef.current, oss_url: res.video_url, file_size: 0, mime_type: 'video/mp4', status: 'uploaded', created_at: new Date().toISOString() }, ...prev]
          })
        } else if (res.status === 'failed') {
          setVideoPolling(false)
          setVideoMessage('\u274c ' + res.message)
        } else {
          setVideoMessage('\u23f3 视频生成中...')
        }
      } catch (e) {
        setVideoMessage('\u26a0\ufe0f 查询状态失败: ' + (e instanceof Error ? e.message : '未知错误'))
      }
    }, 5000)
    return () => clearInterval(timer)
  }, [videoPolling, videoAssetId, id])

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
        if (buildPreviewNum === 0 && gp.length > 0) setBuildPreviewNum(gp[0].page_number)
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

  // P0-4: 单页AI微调
  const handleRefinePage = async () => {
    if (!id || refinePageNum <= 0 || !refineInput.trim()) return
    setRefineRunning(true)
    try {
      const result = await refinePage(id, refinePageNum, refineInput.trim())
      if (result.html_content) {
        setGeneratedPages(prev => prev.map(p => p.page_number === refinePageNum ? { ...p, html_content: result.html_content } : p))
        setBuildPreviewNum(refinePageNum)
      }
      setRefineInput(''); setRefinePageNum(0)
      setBuildMessage('\u2705 ' + result.message)
    } catch (e) { setBuildMessage('\u274c 微调失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setRefineRunning(false) }
  }

  // Step 4: 批量生成剩余页
  const handleBuildStart = async () => {
    if (!id) return; setBuildRunning(true); setBuildMessage('正在启动...'); setBuildProgress({ current: 0, total: 0 })
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
    const t = setInterval(async () => { try { const d = await getCourseware(id); if (d.status === 'preview') { setBuildRunning(false); setCourseware(d); setPages(d.pages || []); goToStep(5); setBuildMessage('✅ 完成'); const gp = d.pages.filter(p => p.html_content).map(p => ({ page_number: p.page_number, title: p.title, html_content: p.html_content })); setGeneratedPages(gp); if (gp.length) setBuildPreviewNum(gp[0].page_number); sseRef.current?.close() } } catch {} }, 15000)
    return () => clearInterval(t)
  }, [buildRunning, id])

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
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, color: C.textSecondary, marginBottom: 6 }}><span>生成进度</span><span>{buildProgress.current} / {buildProgress.total} 页</span></div>
            <div style={{ height: 8, borderRadius: 4, background: '#F3F4F6', overflow: 'hidden' }}><div style={{ height: '100%', borderRadius: 4, transition: 'width 500ms', width: `${(buildProgress.current / buildProgress.total) * 100}%`, background: 'linear-gradient(90deg, #F59E0B, #EF4444)' }} /></div>
          </div>}
          {renderPagePreview(generatedPages, buildPreviewNum, setBuildPreviewNum, true)}
          {!buildRunning && generatedPages.filter(p => p.page_number > 1).length === 0 && (
            <button onClick={handleBuildStart} style={{ padding: '14px 36px', borderRadius: 10, border: 'none', background: 'linear-gradient(135deg, #F59E0B, #EF4444)', color: '#fff', fontSize: 16, fontWeight: 600, cursor: 'pointer', boxShadow: '0 4px 16px rgba(245,158,11,0.3)' }}>⚡ 开始批量生成剩余页面</button>
          )}
          {!buildRunning && generatedPages.filter(p => p.page_number > 1).length > 0 && <div style={{ display: 'flex', gap: 12 }}>
            <button onClick={handleBuildStart} style={{ padding: '10px 24px', borderRadius: 8, border: `1px solid ${C.primary}`, background: C.primaryBg, color: C.primary, fontSize: 14, fontWeight: 600, cursor: 'pointer' }}>🔄 重新生成</button>
            <button onClick={() => openSlideshow()} style={{ padding: '10px 24px', borderRadius: 8, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 14, fontWeight: 600, cursor: 'pointer' }}>🖥️ 全屏放映</button>
            <button onClick={() => { goToStep(5); loadCourseware() }} style={{ padding: '10px 24px', borderRadius: 8, border: 'none', background: 'linear-gradient(135deg, #059669, #10B981)', color: '#fff', fontSize: 14, fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 8px rgba(5,150,105,0.3)' }}>确认课件 →</button>
          </div>}
          {buildRunning && <div style={{ textAlign: 'center', padding: 20, color: C.textMuted, fontSize: 14 }}><div style={{ fontSize: 32, marginBottom: 8 }}>⚡</div>AI正在逐页生成，请耐心等待...</div>}
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
              <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🎨 对某页不满意？选择页码输入修改意见</div>
              <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
                <select value={refinePageNum} onChange={e => setRefinePageNum(Number(e.target.value))}
                  style={{ padding: '10px 14px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 14, minWidth: 120 }}>
                  <option value={0}>选择页码</option>
                  {generatedPages.map(p => <option key={p.page_number} value={p.page_number}>P{p.page_number}: {p.title.slice(0,15)}</option>)}
                </select>
                <input value={refineInput} onChange={e => setRefineInput(e.target.value)}
                  placeholder="例如：标题字号再大一些、增加一个图片占位..."
                  onKeyDown={e => { if (e.key === 'Enter' && !refineRunning && refinePageNum > 0) handleRefinePage() }}
                  style={{ flex: 1, padding: '10px 14px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 14, outline: 'none', minWidth: 200 }}
                  disabled={refineRunning} />
                <button onClick={handleRefinePage} disabled={refineRunning || refinePageNum <= 0 || !refineInput.trim()}
                  style={{ padding: '10px 20px', borderRadius: 8, border: 'none', background: refinePageNum > 0 && refineInput.trim() && !refineRunning ? '#7C3AED' : '#E5E7EB', color: refinePageNum > 0 && refineInput.trim() && !refineRunning ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: refinePageNum > 0 && refineInput.trim() && !refineRunning ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
                  {refineRunning ? '⏳ 微调中...' : '🎨 AI微调'}
                </button>
              </div>
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

              {/* 页码选择 */}
              <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 16 }}>
                <select
                  value={mediaPageNum}
                  onChange={async e => {
                    const pn = Number(e.target.value)
                    setMediaPageNum(pn); setMediaAssets([]); setMediaGenPrompt(''); setMediaRefUrl('')
                    if (pn > 0 && id) {
                      try { const res = await listPageAssets(id, pn); setMediaAssets(res.assets || []) } catch { /* */ }
                    }
                  }}
                  style={{ padding: '10px 14px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 14, minWidth: 160 }}
                >
                  <option value={0}>选择页码添加{mediaTab === "video" ? "视频" : "图片"}</option>
                  {generatedPages.map(p => <option key={p.page_number} value={p.page_number}>P{p.page_number}: {p.title.slice(0,15)}</option>)}
                </select>

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
                    🔄 刷新图片列表
                  </button>
                )}
              </div>

              {mediaPageNum > 0 && mediaTab === 'image' && (
                <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
                  {/* 左栏：AI生成 */}
                  <div style={{ flex: '1 1 300px', padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
                    <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🤖 AI生成图片</div>
                    <textarea
                      value={mediaGenPrompt}
                      onChange={e => setMediaGenPrompt(e.target.value)}
                      placeholder="描述你想要的图片，例如：一张展示AI机器人帮助学生学习的卡通插图，色彩明亮，适合小学生"
                      rows={3}
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
                    {/* 参考图选择 */}
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

                  {/* 右栏：手动上传 */}
                  <div style={{ flex: '1 1 300px', padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
                    <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>📤 手动上传图片</div>
                    <div style={{ padding: '24px 16px', borderRadius: 8, border: '2px dashed ' + C.border, textAlign: 'center', cursor: 'pointer', background: '#FAFAFA' }}
                      onClick={() => { const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*'; inp.onchange = async (ev) => { const f = (ev.target as HTMLInputElement).files?.[0]; if (!f || !id) return; if (f.size > 5 * 1024 * 1024) { setMediaMessage('❌ 图片不能超过5MB'); return } setMediaGenerating(true); setMediaMessage(''); try { const res = await uploadCWImage(id, mediaPageNum, f); setMediaMessage('✅ 上传成功！'); setMediaAssets(prev => [{ id: res.asset_id, courseware_id: id!, page_id: null, placeholder_id: '', asset_type: 'image', generation_prompt: '', oss_url: res.url, file_size: res.file_size, mime_type: res.mime_type, status: 'uploaded', created_at: new Date().toISOString() }, ...prev]) } catch (e) { setMediaMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setMediaGenerating(false) } }; inp.click() }}
                    >
                      <div style={{ fontSize: 28, marginBottom: 6 }}>📷</div>
                      <div style={{ fontSize: 13, color: C.textSecondary }}>点击选择图片</div>
                      <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>支持 JPG/PNG/WEBP/GIF/SVG，最大5MB</div>
                    </div>
                  </div>
                </div>
              )}

              {/* 提示消息 */}
              {mediaMessage && <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, background: mediaMessage.startsWith('❌') ? '#FEE2E2' : '#D1FAE5', color: mediaMessage.startsWith('❌') ? '#DC2626' : '#059669', fontSize: 13 }}>{mediaMessage}</div>}

              {/* 已上传图片列表 */}
              {mediaAssets.filter(a => a.asset_type === 'image').length > 0 && mediaTab === 'image' && (
                <div style={{ marginTop: 16 }}>
                  <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>📎 第 {mediaPageNum} 页的图片（{mediaAssets.filter(a => a.asset_type === 'image').length}张）</div>
                  <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
                    {mediaAssets.filter(a => a.asset_type === 'image').map(asset => (
                      <div key={asset.id} style={{ width: 'calc(25% - 9px)', minWidth: 180, borderRadius: 10, border: '1px solid ' + C.border, overflow: 'hidden', background: '#fff' }}>
                        <img src={asset.oss_url} alt="课件图片" onClick={() => setMediaPreviewUrl(asset.oss_url)} style={{ width: '100%', height: 140, objectFit: 'cover', display: 'block', cursor: 'pointer' }} title="点击查看大图" />
                        <div style={{ padding: '8px 10px' }}>
                          <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 6 }}>{asset.generation_prompt ? '🤖 AI生成' : '📤 手动上传'}</div>
                          <div style={{ display: 'flex', gap: 4 }}>
                            
                            <button
                              onClick={() => { setMediaRefUrl(asset.oss_url); setMediaMessage('✅ 已选为参考图，AI将参考此图风格生成新图片') }}
                              style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: 'pointer' }}
                            >参考</button>
                            <button
                              onClick={async () => {
                                if (!id) return
                                setMediaMessage('⏳ 正在上传到云盘...')
                                try {
                                  const res = await uploadAssetToOSS(id, asset.id)
                                  await navigator.clipboard.writeText(res.oss_public_url)
                                  setMediaMessage('✅ 已上传云盘，链接已复制: ' + res.oss_public_url)
                                } catch (e) { setMediaMessage('❌ 上传云盘失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                              }}
                              style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #0891B2', background: 'rgba(8,145,178,0.06)', color: '#0891B2', fontSize: 11, cursor: 'pointer' }}
                            >☁️云盘</button>
                            <button
                              onClick={async () => {
                                if (!id || !confirm('确定删除这张图片？')) return
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

              
              {/* v0.42.1: 视频生成区（mediaTab==='video'时显示） */}
              {mediaTab === 'video' && mediaPageNum > 0 && (
                <div style={{ padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff', marginTop: 16 }}>
                  <div style={{ fontSize: 13, fontWeight: 600, color: '#7C3AED', marginBottom: 8 }}>🎬 AI视频生成（豆包Seedance）</div>
                  <textarea
                    value={videoPrompt}
                    onChange={e => setVideoPrompt(e.target.value)}
                    placeholder="描述你想要的视频场景，例如：一位女教师站在讲台前微笑着讲课，背景是绿色黑板，教室明亮温馨"
                    rows={3}
                    style={{ width: '100%', padding: '10px 12px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 13, resize: 'vertical', outline: 'none', boxSizing: 'border-box' }}
                    disabled={videoGenerating || videoPolling}
                  />
                  {/* 参考图选择（图生视频） */}
                  <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                    <span style={{ fontSize: 12, color: '#6B7280' }}>参考图(可选):</span>
                    {videoRefUrl ? (
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <img src={videoRefUrl} alt="参考图" style={{ width: 40, height: 40, objectFit: 'cover', borderRadius: 6, border: '2px solid #7C3AED' }} />
                        <span style={{ fontSize: 11, color: '#7C3AED' }}>已选择</span>
                        <button onClick={() => setVideoRefUrl('')} style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid #EF4444', background: 'transparent', color: '#EF4444', fontSize: 11, cursor: 'pointer' }}>取消</button>
                      </div>
                    ) : (
                      <span style={{ fontSize: 11, color: '#9CA3AF' }}>无（纯文生视频）</span>
                    )}
                    {mediaAssets.filter(a => a.asset_type === 'image').length > 0 && !videoRefUrl && (
                      <select onChange={e => { if (e.target.value) setVideoRefUrl(e.target.value) }} value="" style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #7C3AED', fontSize: 11, color: '#7C3AED' }}>
                        <option value="">从已有图片选择...</option>
                        {mediaAssets.filter(a => a.asset_type === 'image').map(a => <option key={a.id} value={a.oss_url}>{a.generation_prompt ? '🤖' : '📤'} {a.oss_url.split('/').pop()?.slice(0,25)}</option>)}
                      </select>
                    )}
                  </div>
                  {/* 生成按钮 */}
                  <button
                    onClick={async () => {
                      if (!id || mediaPageNum <= 0 || !videoPrompt.trim() || videoGenerating || videoPolling) return
                      setVideoGenerating(true); setVideoMessage(''); setVideoResult(null)
                      try {
                        const res = await generateCWVideo(id, mediaPageNum, videoPrompt.trim(), videoRefUrl || undefined)
                        setVideoAssetId(res.asset_id)
                        setVideoTaskId(res.task_id)
                        setVideoMessage('✅ ' + res.message)
                        setVideoGenerating(false)
                        // 开始轮询
                        setVideoPolling(true)
                      } catch (e) {
                        setVideoMessage('❌ 提交失败: ' + (e instanceof Error ? e.message : '未知错误'))
                        setVideoGenerating(false)
                      }
                    }}
                    disabled={videoGenerating || videoPolling || !videoPrompt.trim()}
                    style={{ marginTop: 8, padding: '10px 20px', borderRadius: 8, border: 'none', background: videoPrompt.trim() && !videoGenerating && !videoPolling ? 'linear-gradient(135deg, #7C3AED, #6D28D9)' : '#E5E7EB', color: videoPrompt.trim() && !videoGenerating && !videoPolling ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: videoPrompt.trim() && !videoGenerating && !videoPolling ? 'pointer' : 'default', width: '100%' }}
                  >
                    {videoGenerating ? '⏳ 提交中...' : videoPolling ? '⏳ 视频生成中（约30-120秒）...' : '🎬 生成视频'}
                  </button>
                  {/* 视频消息 */}
                  {videoMessage && <div style={{ marginTop: 8, padding: '10px 14px', borderRadius: 8, background: videoMessage.startsWith('❌') ? '#FEE2E2' : videoMessage.startsWith('⚠') ? '#FEF3C7' : '#D1FAE5', color: videoMessage.startsWith('❌') ? '#DC2626' : videoMessage.startsWith('⚠') ? '#D97706' : '#059669', fontSize: 13 }}>{videoMessage}</div>}
                  {/* 视频结果预览 */}
                  {videoResult && (
                    <div style={{ marginTop: 12, borderRadius: 10, border: '1px solid #7C3AED', overflow: 'hidden' }}>
                      <video src={videoResult.url} controls style={{ width: '100%', maxHeight: 360, display: 'block', background: '#000' }} />
                      <div style={{ padding: '8px 12px', background: 'rgba(124,58,237,0.04)', fontSize: 12, color: '#6B7280' }}>
                        🎬 时长 {videoResult.duration}秒 | 分辨率 {videoResult.resolution}
                      </div>
                    </div>
                  )}
                  {/* 提示 */}
                  <div style={{ marginTop: 8, fontSize: 11, color: '#9CA3AF' }}>💡 视频默认5秒720p。生成完成后可点击「☁️云盘」上传获取公网链接。</div>
                </div>
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
                          <div style={{ fontSize: 11, color: C.textMuted, marginBottom: 4 }}>{asset.generation_prompt ? '🤖 ' + asset.generation_prompt.slice(0,25) + (asset.generation_prompt.length > 25 ? '...' : '') : '📤 手动上传'}</div>
                          <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                            
                            <button onClick={async () => {
                                if (!id) return
                                setVideoMessage('⏳ 正在上传到云盘...')
                                try {
                                  const res = await uploadAssetToOSS(id, asset.id)
                                  await navigator.clipboard.writeText(res.oss_public_url)
                                  setVideoMessage('✅ 已上传云盘，链接已复制: ' + res.oss_public_url)
                                } catch (e) { setVideoMessage('❌ 上传云盘失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                              }} style={{ padding: '3px 8px', borderRadius: 5, border: '1px solid #0891B2', background: 'rgba(8,145,178,0.06)', color: '#0891B2', fontSize: 10, cursor: 'pointer' }}>☁️云盘</button>
                            <button onClick={async () => { if (!id || !confirm('确定删除此视频？')) return; try { await deleteCWAsset(id, asset.id); setMediaAssets(prev => prev.filter(a => a.id !== asset.id)); setVideoMessage('✅ 已删除') } catch (e) { setVideoMessage('❌ 删除失败: ' + (e instanceof Error ? e.message : '')) } }} style={{ padding: '3px 8px', borderRadius: 5, border: '1px solid #EF4444', background: 'rgba(239,68,68,0.06)', color: '#EF4444', fontSize: 10, cursor: 'pointer' }}>删除</button>
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
            setEditorExporting(true); setVideoMessage('')
            try {
              const res = await advancedConcatCWVideos(id, clips)
              setVideoMessage('\u2705 ' + res.message)
              setMediaAssets(prev => [{
                id: res.asset_id, courseware_id: id!, page_id: null, placeholder_id: '',
                asset_type: 'video', generation_prompt: '\u7f16\u8f91\u5bfc\u51fa ' + clips.length + '\u4e2a\u7247\u6bb5',
                oss_url: res.url, file_size: 0, mime_type: 'video/mp4', status: 'uploaded',
                created_at: new Date().toISOString(),
              }, ...prev])
              setEditorOpen(false)
            } catch (e) {
              setVideoMessage('\u274c \u5bfc\u51fa\u5931\u8d25: ' + (e instanceof Error ? e.message : ''))
            } finally { setEditorExporting(false) }
          }}
        />
      )}

    </div>
  )
}

/**
 * 全屏预览组件 (CWFullscreenPreview.tsx)
 *
 * 从 CoursewareWorkshopPage.tsx 抽出（v137 全屏预览，带工具栏+键盘导航+resize响应）。
 * 区别于 SlideshowPlayer 的纯放映：本组件顶部有 60px 工具栏（翻页/代码切换/复制/放映/退出），
 * 内容区 flex 居中 + 缩放层 transformOrigin:center 消除左侧亚像素白边。
 * iframe srcDoc 经 injectPreviewMode 注入预览降级脚本。
 *
 * 【P1-02 体验修复·定稿做法】当前页号只存本组件 curPageNum 一处（单一真相源），
 *   父组件不再每次翻页回写（双源打架会导致翻页回弹）。改为退出时经 onClose(finalPage)
 *   一次性回传当前停留页；切到放映时经 onSlideshow(pn) 带上当前页。
 *   工具栏在顶部、不遮挡课件，故不做放映态那种「手动隐藏控制条」处理。
 *
 * 【v5.5 焦点穿透修复】用户点击课件互动元素后焦点进入 iframe，键盘翻页失效。
 *   修复：监听来自 iframe 的 postMessage 导航按键转发消息，在父窗口执行翻页/退出。
 */
import { useState, useEffect, useRef } from 'react'
import { CW_WIDTH, CW_HEIGHT } from './workshopConstants'
import { injectPreviewMode, NAV_KEY_MSG_TYPE } from './previewInject'
import PlatformCoursewareAssistantOverlay from './PlatformCoursewareAssistantOverlay'
import { rememberCoursewarePreviewPage } from './coursewarePreviewPosition'

export default function CWFullscreenPreview({ pages, initialPageNum, codeView, onToggleCode, onClose, onSlideshow }: {
  pages: { id?: string; page_number: number; title: string; html_content: string }[]
  initialPageNum: number
  codeView: boolean
  onToggleCode: () => void
  // P1-02: 退出时回传「当前停留页」给父组件（可选参数），翻页过程中绝不回调
  onClose: (finalPage?: number) => void
  onSlideshow: (pn: number) => void
}) {
  const [curPageNum, setCurPageNum] = useState(initialPageNum)
  const [viewSize, setViewSize] = useState({ w: window.innerWidth, h: window.innerHeight })
  // P1-02: ref 始终持有最新当前页，供退出回调读取（闭包不读旧值）
  const curPageRef = useRef(initialPageNum)

  const idx = pages.findIndex(p => p.page_number === curPageNum)
  const page = pages[idx] || pages[0]
  const html = page?.html_content || ''

  /**
   * 全屏翻页时同步稳定page_id。
   *
   * 浏览器在全屏状态直接刷新会退出全屏并重新挂载工坊，
   * 因此必须在每次全屏页变化时提前写入sessionStorage。
   */
  useEffect(() => {
    if (page?.id) {
      rememberCoursewarePreviewPage(page.id)
    }
  }, [page?.id])

  // v0.41: 注入预览降级脚本
  const previewHtml = injectPreviewMode(html)

  const hasPrev = idx > 0
  const hasNext = idx < pages.length - 1

  // P1-02: 统一翻页入口——只改本组件 state（单一真相源）+ 同步刷新 ref，不回调父组件
  const gotoPage = (pn: number) => {
    setCurPageNum(pn)
    curPageRef.current = pn
  }
  // P1-02: 统一退出入口——退出时把最新当前页一次性回传父组件
  const doClose = () => { onClose(curPageRef.current) }

  // 响应窗口resize
  useEffect(() => {
    const fn = () => setViewSize({ w: window.innerWidth, h: window.innerHeight })
    window.addEventListener('resize', fn)
    return () => window.removeEventListener('resize', fn)
  }, [])

  // 键盘导航：左右箭头翻页 + ESC退出
  useEffect(() => {
    const fn = (e: KeyboardEvent) => {
      // v5.5: 如果焦点在输入框内，不处理（全屏预览场景通常不会有，但防御性编码）
      const tag = (e.target as HTMLElement)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
      if (e.key === 'Escape') { doClose(); return }
      if (e.key === 'ArrowLeft' && hasPrev) gotoPage(pages[idx - 1].page_number)
      if (e.key === 'ArrowRight' && hasNext) gotoPage(pages[idx + 1].page_number)
    }
    window.addEventListener('keydown', fn)
    return () => window.removeEventListener('keydown', fn)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [idx, hasPrev, hasNext, pages])

  // 【v5.5 焦点穿透修复】监听来自 iframe 的 postMessage 导航按键转发
  // iframe 内用户点击互动元素后焦点被 iframe 捕获，keydown 不再冒泡到父 window，
  // 但 previewInject.ts 注入的脚本会把导航按键经 postMessage 转发到这里。
  useEffect(() => {
    const onMessage = (e: MessageEvent) => {
      // 校验消息格式：必须是来自 iframe 的导航按键转发
      if (!e.data || e.data.type !== NAV_KEY_MSG_TYPE) return
      const key = e.data.key as string
      if (!key) return

      if (key === 'Escape') { doClose(); return }
      if (key === 'ArrowLeft' && hasPrev) {
        gotoPage(pages[idx - 1].page_number)
      }
      if (key === 'ArrowRight' && hasNext) {
        gotoPage(pages[idx + 1].page_number)
      }
    }
    window.addEventListener('message', onMessage)
    return () => window.removeEventListener('message', onMessage)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [idx, hasPrev, hasNext, pages])

  // 缩放计算：工具栏高度60px，内容区占满剩余空间
  // v5.4: 只算 scale，居中交给外层 flex（不再手动算 ox/oy 偏移，避免亚像素白边）
  const toolbarH = 60
  const contentH = viewSize.h - toolbarH
  const scale = Math.min(viewSize.w / CW_WIDTH, contentH / CW_HEIGHT)

  const tbtn: React.CSSProperties = { padding: '6px 14px', borderRadius: 8, border: '1px solid #E5E7EB', background: '#fff', color: '#6B7280', fontSize: 13, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 4 }

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 99998, background: '#fff', display: 'flex', flexDirection: 'column' }}>
      {/* 工具栏 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 20px', background: 'rgba(255,255,255,0.95)', borderBottom: '1px solid rgba(0,0,0,0.08)', flexShrink: 0, height: toolbarH, boxSizing: 'border-box' }}>
        <button onClick={() => hasPrev && gotoPage(pages[idx - 1].page_number)} disabled={!hasPrev} style={{ ...tbtn, opacity: hasPrev ? 1 : 0.3, cursor: hasPrev ? 'pointer' : 'not-allowed' }}>‹</button>
        <span style={{ fontSize: 14, fontWeight: 600, color: '#1F2937' }}>P{page?.page_number} — {page?.title}</span>
        <span style={{ fontSize: 12, color: '#9CA3AF' }}>{(idx >= 0 ? idx : 0) + 1}/{pages.length}</span>
        <button onClick={() => hasNext && gotoPage(pages[idx + 1].page_number)} disabled={!hasNext} style={{ ...tbtn, opacity: hasNext ? 1 : 0.3, cursor: hasNext ? 'pointer' : 'not-allowed' }}>›</button>
        <div style={{ flex: 1 }} />
        <button onClick={onToggleCode} style={{ ...tbtn, border: `1px solid ${codeView ? '#7C3AED' : '#E5E7EB'}`, background: codeView ? 'rgba(124,58,237,0.06)' : '#fff', color: codeView ? '#7C3AED' : '#6B7280' }}>{codeView ? '📺 预览' : '💻 代码'}</button>
        <button onClick={() => { navigator.clipboard.writeText(html).then(() => alert('已复制')).catch(() => {}) }} style={tbtn}>📋 复制</button>
        <button onClick={() => onSlideshow(page?.page_number || 1)} style={{ ...tbtn, border: '1px solid #F59E0B', background: 'rgba(245,158,11,0.06)', color: '#F59E0B' }}>🖥️ 放映</button>
        <button onClick={doClose} style={tbtn}>✕ 退出</button>
      </div>
      {/* 内容区：v5.4 改为 flex 居中，缩放层 transformOrigin:center，消除左侧白边 */}
      <div style={{ flex: 1, overflow: 'hidden', position: 'relative', background: codeView ? '#1e1e1e' : '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        {codeView ? (
          <div style={{ width: '100%', height: '100%', overflow: 'auto', fontFamily: 'Monaco, Consolas, "Courier New", monospace', fontSize: 13, lineHeight: 1.7 }}>
            <table style={{ borderCollapse: 'collapse', width: '100%' }}><tbody>
              {html.split('\n').map((line: string, i: number) => (
                <tr key={i}><td style={{ width: 55, minWidth: 55, textAlign: 'right', padding: '0 10px 0 8px', color: '#858585', userSelect: 'none', verticalAlign: 'top', borderRight: '1px solid #333', whiteSpace: 'nowrap' }}>{i + 1}</td><td style={{ padding: '0 12px', color: '#d4d4d4', whiteSpace: 'pre', wordBreak: 'break-all' }}>{line || ' '}</td></tr>
              ))}
            </tbody></table>
          </div>
        ) : (
          <div style={{ width: CW_WIDTH, height: CW_HEIGHT, flexShrink: 0, transform: `scale(${scale})`, transformOrigin: 'center center' }}>
            <iframe srcDoc={previewHtml} scrolling="no" style={{ width: CW_WIDTH, height: CW_HEIGHT, border: 'none', display: 'block', overflow: 'hidden' }} sandbox="allow-scripts" title={`全屏预览-P${page?.page_number}`} />
          </div>
        )}
      </div>

      {!codeView && page?.id && (
        <PlatformCoursewareAssistantOverlay
          key={`platform-assistant-${page.id}`}
          pageId={page.id}
          pageTitle={page.title}
          variant="fullscreen"
        />
      )}
    </div>
  )
}

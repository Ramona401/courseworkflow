/**
 * 全屏幻灯片放映组件 (SlideshowPlayer.tsx)
 *
 * 从 CoursewareWorkshopPage.tsx 抽出（v5.2 重写：白色背景+无滚动条；v5.4 flex 居中消白边）。
 * 调用浏览器原生 requestFullscreen 进入真全屏，白底铺满；左右键/点击两侧/圆点指示器翻页；
 * UI 控制条 3 秒自动隐藏；iframe srcDoc 经 injectPreviewMode 注入预览降级脚本。
 */
import { useState, useEffect, useRef } from 'react'
import { C, CW_WIDTH, CW_HEIGHT } from './workshopConstants'
import { injectPreviewMode } from './previewInject'

export default function SlideshowPlayer({ pages, initialPage, onClose }: {
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
  // v5.4: 只算 scale，居中交给外层 flex（不再手动算 ox/oy 偏移，避免亚像素白边）
  const scale = Math.min(containerSize.w / CW_WIDTH, containerSize.h / CW_HEIGHT)

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
        /* v5.4: flex 居中课件，消除左侧亚像素白边 */
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}>
      {/* 课件内容区域：v5.4 flex 居中 + 缩放层 transformOrigin:center（不再绝对定位偏移） */}
      <div style={{
        width: CW_WIDTH, height: CW_HEIGHT, flexShrink: 0,
        transform: `scale(${scale})`,
        transformOrigin: 'center center',
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

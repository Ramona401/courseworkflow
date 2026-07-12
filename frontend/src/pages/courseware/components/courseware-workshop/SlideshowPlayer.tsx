/**
 * 全屏幻灯片放映组件 (SlideshowPlayer.tsx)
 *
 * 从 CoursewareWorkshopPage.tsx 抽出（v5.2 重写：白色背景+无滚动条；v5.4 flex 居中消白边）。
 * 调用浏览器原生 requestFullscreen 进入真全屏，白底铺满；左右键/点击两侧/圆点指示器翻页；
 * iframe srcDoc 经 injectPreviewMode 注入预览降级脚本。
 *
 * 【P1-01 体验修复】放映控制条原 bottom:24 常驻居中、3 秒淡出但 mousemove 又弹回，
 *   课件底部若有按钮/答题区会被压住。本次两处改进：
 *   ① 控制条整体下移贴底（bottom:8）+ 透明度降一档，减少遮挡面积；
 *   ② 新增「手动隐藏」——点控制条上的「∨」收起后进入 controlsHidden 态，此后 mousemove
 *      不再自动弹出（flashUI 在隐藏态短路），屏幕底缘出现半透明小把手「⌃」点击才唤回。
 *
 * 【P1-02 体验修复·定稿做法】当前页号只存本组件 curPage 一处（单一真相源），
 *   父组件不再每次翻页回写（那会形成 state 双源打架，导致点下一页又被旧 prop 拉回）。
 *   改为退出时经 onClose(finalPage) 把最终停留页一次性回传父组件，父组件据此记住下次初值。
 *   initialPage 仅作挂载初值，翻页全程父组件不参与，彻底消除翻页回弹。
 *
 * 【v5.5 焦点穿透修复】用户点击课件互动元素（按钮/答题区/拖拽等）后焦点进入 iframe，
 *   此后键盘事件不冒泡到父 window，翻页快捷键失效。修复：监听来自 iframe 的 postMessage
 *   导航按键转发消息（由 previewInject.ts 注入脚本发出），在父窗口执行翻页/退出。
 *   同时在放映容器获得点击时主动 focus 回父文档，确保后续键盘事件能被父 window 监听到。
 */
import { useState, useEffect, useRef } from 'react'
import { C, CW_WIDTH, CW_HEIGHT } from './workshopConstants'
import { injectPreviewMode, NAV_KEY_MSG_TYPE } from './previewInject'

export default function SlideshowPlayer({ pages, initialPage, onClose }: {
  pages: { page_number: number; title: string; html_content: string }[]
  initialPage: number
  // P1-02: 退出时回传「当前停留页」给父组件（可选参数），父组件据此记住下次初值；
  //   不传参时父组件按各自逻辑处理（保持兼容）。翻页过程中绝不回调，避免双源打架。
  onClose: (finalPage?: number) => void
}) {
  const [curPage, setCurPage] = useState(initialPage)
  const [showUI, setShowUI] = useState(true)
  // P1-01: 手动隐藏控制条态——true 时 mousemove 不再自动弹出控制条，只能点底缘把手唤回
  const [controlsHidden, setControlsHidden] = useState(false)
  // v5.2: 使用innerWidth/innerHeight作为默认值，全屏后切换为screen尺寸
  const [containerSize, setContainerSize] = useState({ w: window.innerWidth, h: window.innerHeight })
  const uiTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const boxRef = useRef<HTMLDivElement>(null)
  // P1-02: 用 ref 始终持有「最新当前页」，供 onClose 与全屏退出回调读取（闭包不会读到旧值）
  const curPageRef = useRef(initialPage)

  const data = pages.find(p => p.page_number === curPage)
  const idx = pages.findIndex(p => p.page_number === curPage)
  const hasPrev = idx > 0
  const hasNext = idx < pages.length - 1

  // v0.41: 注入预览降级脚本
  const previewHtml = data ? injectPreviewMode(data.html_content) : ''

  // P1-02: 统一翻页入口——只改本组件 state（单一真相源）+ 同步刷新 ref，不回调父组件
  const gotoPage = (pn: number) => {
    setCurPage(pn)
    curPageRef.current = pn
  }
  // P1-02: 统一退出入口——退出时把最新当前页一次性回传父组件
  const doClose = () => { onClose(curPageRef.current) }

  // 请求浏览器全屏API
  useEffect(() => {
    const el = boxRef.current
    if (el?.requestFullscreen) el.requestFullscreen().catch(() => {})
    const onFs = () => {
      if (!document.fullscreenElement) {
        // 浏览器退出全屏（ESC/系统手势）→ 走统一退出，回传当前页
        doClose()
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

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

  // 键盘导航（直接在父 window 上监听）
  useEffect(() => {
    const fn = (e: KeyboardEvent) => {
      // v5.5: 如果焦点在父文档的 input/textarea/select 内，不处理（放映场景通常不会有，但防御性编码）
      const tag = (e.target as HTMLElement)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
      if (e.key === 'Escape') { doClose(); return }
      if ((e.key === 'ArrowLeft' || e.key === 'PageUp') && hasPrev) gotoPage(pages[idx - 1].page_number)
      if ((e.key === 'ArrowRight' || e.key === 'PageDown' || e.key === ' ') && hasNext) { e.preventDefault(); gotoPage(pages[idx + 1].page_number) }
      flashUI()
    }
    window.addEventListener('keydown', fn)
    return () => window.removeEventListener('keydown', fn)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [curPage, idx, hasPrev, hasNext, pages])

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
      if ((key === 'ArrowLeft' || key === 'PageUp') && hasPrev) {
        gotoPage(pages[idx - 1].page_number)
      }
      if ((key === 'ArrowRight' || key === 'PageDown' || key === ' ') && hasNext) {
        gotoPage(pages[idx + 1].page_number)
      }
      flashUI()
    }
    window.addEventListener('message', onMessage)
    return () => window.removeEventListener('message', onMessage)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [curPage, idx, hasPrev, hasNext, pages])

  // UI自动隐藏定时器
  // P1-01: controlsHidden 为 true 时直接短路——手动收起后不再因 mousemove/翻页自动弹出
  const flashUI = () => {
    if (controlsHidden) return
    setShowUI(true)
    if (uiTimer.current) clearTimeout(uiTimer.current)
    uiTimer.current = setTimeout(() => setShowUI(false), 3000)
  }
  useEffect(() => {
    uiTimer.current = setTimeout(() => setShowUI(false), 3000)
    return () => { if (uiTimer.current) clearTimeout(uiTimer.current) }
  }, [])

  // P1-01: 手动收起控制条——清掉自动隐藏定时器，进入隐藏态，mousemove 不再唤起
  const hideControls = () => {
    if (uiTimer.current) clearTimeout(uiTimer.current)
    setControlsHidden(true)
    setShowUI(false)
  }
  // P1-01: 点底缘把手唤回控制条——退出隐藏态并显示（不自动再隐藏，让老师自己决定）
  const showControls = () => {
    setControlsHidden(false)
    setShowUI(true)
    if (uiTimer.current) clearTimeout(uiTimer.current)
  }

  if (!data) return null

  // v5.2: 缩放计算 — 以宽度为基准，确保内容完全可见无黑框
  // v5.4: 只算 scale，居中交给外层 flex（不再手动算 ox/oy 偏移，避免亚像素白边）
  const scale = Math.min(containerSize.w / CW_WIDTH, containerSize.h / CW_HEIGHT)

  return (
    <div ref={boxRef} data-slideshow="1"
      onMouseMove={flashUI}
      onClick={(e) => {
        // 【v5.5 焦点穿透修复】用户点击放映容器的非 iframe 区域时，
        // 主动把焦点拉回父文档，确保后续键盘事件能被父 window 的 keydown 监听器捕获。
        // 点击 iframe 内部时此 onClick 不会触发（iframe 吞掉了事件），
        // 点击控制条按钮时有 stopPropagation 不会到这里，
        // 所以这里只处理「点击课件两侧空白区域翻页」的场景。
        if (document.activeElement instanceof HTMLIFrameElement) {
          // 焦点在 iframe 上 → 把焦点拉回放映容器
          boxRef.current?.focus()
        }
        const r = boxRef.current?.getBoundingClientRect()
        if (!r) return
        const x = e.clientX - r.left
        if (x < r.width * 0.25 && hasPrev) gotoPage(pages[idx - 1].page_number)
        else if (x > r.width * 0.75 && hasNext) gotoPage(pages[idx + 1].page_number)
        flashUI()
      }}
      // v5.5: 添加 tabIndex 使 div 可聚焦（focus 回拉需要），outline:none 去掉聚焦框
      tabIndex={-1}
      style={{
        position: 'fixed', inset: 0, zIndex: 99999,
        background: '#fff',              /* v5.2: 白色背景替代黑色 */
        cursor: showUI ? 'default' : 'none',
        overflow: 'hidden',              /* v5.2: 消除外层滚动条 */
        outline: 'none',                 /* v5.5: 去掉 tabIndex 带来的聚焦框 */
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

      {/* P1-01: 底缘唤回把手——仅在控制条被手动收起后出现，点击恢复控制条；
          半透明贴底居中，不挡内容，只占一个小圆角条 */}
      {controlsHidden && (
        <div style={{ position: 'absolute', inset: 0, zIndex: 3, pointerEvents: 'none' }}>
          <button onClick={e => { e.stopPropagation(); showControls() }}
            style={{
              position: 'absolute', bottom: 6, left: '50%', transform: 'translateX(-50%)',
              pointerEvents: 'auto', cursor: 'pointer',
              padding: '2px 18px', borderRadius: 999, border: 'none',
              background: 'rgba(0,0,0,0.35)', color: 'rgba(255,255,255,0.85)',
              fontSize: 14, lineHeight: 1.4, backdropFilter: 'blur(8px)',
            }} title="显示控制条">⌃</button>
        </div>
      )}

      {/* 底部控制条（自动隐藏 / 可手动收起）：P1-01 整体下移贴底 bottom:8 + 透明度降一档 */}
      <div style={{ position: 'absolute', inset: 0, zIndex: 2, pointerEvents: 'none', opacity: (showUI && !controlsHidden) ? 1 : 0, transition: 'opacity 400ms' }}>
        <div style={{
          position: 'absolute', bottom: 8, left: '50%', transform: 'translateX(-50%)',
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '8px 16px', borderRadius: 999,
          background: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(12px)',  /* P1-01: 0.65→0.5 透明度降一档 */
          pointerEvents: (showUI && !controlsHidden) ? 'auto' : 'none',
        }}>
          {/* 上一页按钮 */}
          <button onClick={e => { e.stopPropagation(); if (hasPrev) gotoPage(pages[idx - 1].page_number) }}
            style={{ width: 36, height: 36, borderRadius: '50%', border: 'none',
              background: hasPrev ? 'rgba(255,255,255,0.15)' : 'transparent',
              color: hasPrev ? '#fff' : 'rgba(255,255,255,0.2)',
              fontSize: 20, cursor: hasPrev ? 'pointer' : 'default',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>‹</button>

          {/* 页面圆点指示器 */}
          <div style={{ display: 'flex', gap: 5, padding: '0 8px' }}>
            {pages.map(p => (
              <button key={p.page_number} onClick={e => { e.stopPropagation(); gotoPage(p.page_number) }}
                style={{
                  width: p.page_number === curPage ? 28 : 10, height: 10, borderRadius: 5,
                  border: 'none', cursor: 'pointer', transition: 'all 250ms',
                  background: p.page_number === curPage ? C.primary : 'rgba(255,255,255,0.3)',
                }}
                title={`第${p.page_number}页: ${p.title}`} />
            ))}
          </div>

          {/* 下一页按钮 */}
          <button onClick={e => { e.stopPropagation(); if (hasNext) gotoPage(pages[idx + 1].page_number) }}
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

          {/* P1-01: 手动隐藏控制条按钮——收起后讲底部交互页不再被挡 */}
          <button onClick={e => { e.stopPropagation(); hideControls() }}
            style={{ width: 36, height: 36, borderRadius: '50%', border: 'none',
              background: 'rgba(255,255,255,0.12)', color: '#fff',
              fontSize: 16, cursor: 'pointer',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }} title="隐藏控制条（不再自动弹出）">∨</button>

          {/* 退出按钮 */}
          <button onClick={e => { e.stopPropagation(); doClose() }}
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

/**
 * 课件模板缩略图组件 — TemplateThumb v2.4 (2026-06-10)
 *
 * 解决核心问题（v1.0 已修）：
 * 1. iframe 内容自带 body margin → 出滚动条 → wrapHTML 强制 overflow:hidden
 * 2. 个人模板是 1920×1080、系统模板是 960×540 → 自动检测内容尺寸
 * 3. 缩放后右下留白 → ResizeObserver 监听容器宽度动态算 scale
 *
 * v2.0（P0 优化）:
 * 4. wrapHTML 兼容完整 HTML 文档：已含 <html>/<!DOCTYPE> 时只注入 <style>，不再嵌套
 * 5. 首屏闪烁修复：useEffect → useLayoutEffect，浏览器绘制前拿到尺寸
 * 6. ResizeObserver 节流：rAF batching + 宽度变化阈值（≥2px 才更新）
 *
 * v2.1（缩略图脚本崩溃根治，2026-06-10）:
 * 7. sampleHTML 分支渲染前 stripScripts() 剥 <script>，iframe sandbox 收紧为 ""（禁脚本，双保险）。
 *    3D（preview_url）分支保持 "allow-scripts allow-same-origin"（Three.js ESM 需要）。
 *
 * v2.2（缩略图"加载不出来"根治，2026-06-10）:
 * 8. detectSize 旧贪婪正则在"全程无 `}`"的纯内联样式片段里会命中最后一个 height（48px 小图标），
 *    把画布判成 960×48。重写为「找最外层画布容器」策略。
 *
 * v2.3（加载提速，2026-06-10）:
 * 9. 仅卡片版 TemplateThumb：懒挂载（IntersectionObserver，进视口才建 iframe）+ useMemo 缓存预处理。
 *
 * v2.4（"加载过就缓存、不再刷新"，2026-06-10）:
 * 10. 现象：卡片滚出视口被父级重渲染拆掉、回来又重建 → 重建后懒挂载视它为全新卡片 → 重走"占位→加载"，
 *     表现为"加载过的卡片滚回来又重载/再闪"。
 *     修复（仅卡片版 TemplateThumb，全在 useInViewOnce 内）：
 *     (a) 会话级"已渲染记忆"：模块级 Set 按内容 key 记录已渲染过的缩略图；卡片重建再挂载时若命中记忆，
 *         直接以 inView=true 起步 → 瞬间显真预览、不回占位、不等观察器。模块级故跨重建持久有效。
 *     (b) 同步视口预判：useLayoutEffect 在首帧绘制前用 getBoundingClientRect 判断是否已在视口（含 400px
 *         提前量），是则同步置可见 → 首屏可见卡片第一帧即真图，消除"整体闪一下"。
 *     边界：24 个静态模板自此回来即时无感；3 个 3D 模板重建仍是全新 iframe、会重跑 Three.js（本缓存只消
 *     占位闪，不能阻止全新 iframe 的文档重载）；要 3D 也不重跑须在父页面堵住"滚动拆建卡片"的根因。
 *
 * 使用方式：
 *   <TemplateThumb previewUrl="..." sampleHTML="..." fallback={<div/>} />
 *   优先级：previewUrl(3D) > sampleHTML > fallback
 */
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'

// ==================== 模块级"已渲染记忆" ====================

/**
 * 会话级缩略图渲染记忆（v2.4）
 *
 * 记录"哪些模板内容已经被挂载渲染过"，key = previewUrl || sampleHTML（内容自身即稳定身份）。
 * 作用：卡片被父级重渲染拆掉又重建时，重建实例可据此直接以"已可见"起步，
 *      跳过占位与 IntersectionObserver 等待 → 加载过的卡片滚回来瞬间显真预览、不再重复闪。
 * 生命周期：模块级，整个 SPA 会话有效（刷新页面才清空）；条目数受模板数量约束，可忽略。
 * 内容变化（如模板被微调）→ key 随内容变 → 自然视为新内容、重新渲染，不会用脏缓存。
 */
const renderedThumbKeys = new Set<string>()

// ==================== 辅助函数 ====================

/**
 * 剥除 HTML 中的所有 <script> 块（含内联脚本与 src 外链脚本）
 * 缩略图只看风格（HTML+CSS），脚本无价值且会在缺 allow-same-origin 沙箱里访问 localStorage 抛错中断；
 * 配合 iframe sandbox="" 双保险。
 */
function stripScripts(html: string): string {
  return html
    .replace(/<script\b[^>]*>[\s\S]*?<\/script\s*>/gi, '')
    .replace(/<script\b[^>]*\/>/gi, '')
}

/**
 * 检测 HTML 内容声明的画布尺寸（v2.2 策略）
 *   1) 首个【同时含 width:Npx 与 height:Npx】的内联 style="..." 属性 = 最外层画布容器；
 *   2) 退化：面积最大的 CSS 规则块 {...}；
 *   3) 退化：属性写法 width=1920 height=1080；
 *   4) 默认 960×540（16:9 标清）。
 */
function detectSize(html: string): { w: number; h: number } {
  // 1) 首个同时含 width:Npx 与 height:Npx 的内联 style 属性 = 最外层画布容器
  const styleAttrs = html.match(/style\s*=\s*"[^"]*"/gi) || []
  for (const attr of styleAttrs) {
    const mw = attr.match(/width\s*:\s*(\d+)px/i)
    const mh = attr.match(/height\s*:\s*(\d+)px/i)
    if (mw && mh) return { w: parseInt(mw[1]), h: parseInt(mh[1]) }
  }
  // 2) 退化：CSS 规则块 {...} 内成对出现，取面积最大者
  let best = { w: 0, h: 0, area: 0 }
  const cssBlocks = html.match(/\{[^{}]*\}/g) || []
  for (const blk of cssBlocks) {
    const mw = blk.match(/width\s*:\s*(\d+)px/i)
    const mh = blk.match(/height\s*:\s*(\d+)px/i)
    if (mw && mh) {
      const w = parseInt(mw[1]), h = parseInt(mh[1]), area = w * h
      if (area > best.area) best = { w, h, area }
    }
  }
  if (best.area > 0) return { w: best.w, h: best.h }
  // 3) 退化：属性写法
  const m2 = html.match(/width\s*=\s*["']?(\d+)["']?[^>]*height\s*=\s*["']?(\d+)["']?/i)
  if (m2) return { w: parseInt(m2[1]), h: parseInt(m2[2]) }
  // 4) 默认 16:9 标清
  return { w: 960, h: 540 }
}

/**
 * 包裹 HTML 片段，强制注入 overflow:hidden 等 reset 样式
 * - 完整文档（含 <html>/<!DOCTYPE>）：只往 <head> 追加 reset <style>，不外层嵌套
 * - 纯片段：外层包裹完整文档结构
 */
function wrapHTML(html: string, w: number, h: number): string {
  const resetCSS = `html,body{margin:0;padding:0;overflow:hidden;width:${w}px;height:${h}px;}::-webkit-scrollbar{display:none;}`
  const isFullDoc = /<!DOCTYPE|<html[\s>]/i.test(html)
  if (isFullDoc) {
    if (/<head[\s>]/i.test(html)) {
      return html.replace(/<head[^>]*>/i, match => `${match}<style>${resetCSS}</style>`)
    }
    if (/<html[\s>]/i.test(html)) {
      return html.replace(/<html[^>]*>/i, match => `${match}<head><style>${resetCSS}</style></head>`)
    }
    return `<head><style>${resetCSS}</style></head>${html}`
  }
  return `<!DOCTYPE html>
<html style="margin:0;padding:0;overflow:hidden;width:${w}px;height:${h}px;">
<head>
<meta charset="UTF-8">
<style>${resetCSS} body>*{display:block;}</style>
</head>
<body>${html}</body>
</html>`
}

// ==================== 自定义 Hook：带节流的容器宽度监听 ====================

/**
 * 监听元素宽度变化（带 rAF 节流 + 阈值过滤）
 * - useLayoutEffect：浏览器绘制前拿到尺寸，消除首屏闪烁
 * - rAF batching：多次触发合并为一帧一次 setState
 * - 阈值过滤：宽度变化 < 2px 不更新（避免亚像素抖动）
 */
function useContainerWidth(ref: React.RefObject<HTMLDivElement | null>, initialWidth = 0): number {
  const [width, setWidth] = useState(initialWidth)
  const rafIdRef = useRef<number | null>(null)
  const lastWidthRef = useRef<number>(initialWidth)

  useLayoutEffect(() => {
    if (!ref.current) return
    const el = ref.current
    const initW = Math.floor(el.clientWidth)
    if (initW > 0) {
      lastWidthRef.current = initW
      setWidth(initW)
    }
    const observer = new ResizeObserver(entries => {
      if (rafIdRef.current !== null) cancelAnimationFrame(rafIdRef.current)
      rafIdRef.current = requestAnimationFrame(() => {
        rafIdRef.current = null
        for (const entry of entries) {
          const newW = Math.floor(entry.contentRect.width)
          if (Math.abs(newW - lastWidthRef.current) >= 2 && newW > 0) {
            lastWidthRef.current = newW
            setWidth(newW)
          }
        }
      })
    })
    observer.observe(el)
    return () => {
      observer.disconnect()
      if (rafIdRef.current !== null) cancelAnimationFrame(rafIdRef.current)
    }
  }, [ref])

  return width
}

// ==================== 自定义 Hook：懒挂载（进入视口才渲染，且记忆已渲染） ====================

/**
 * 懒挂载 + 渲染记忆 Hook（v2.4）
 *
 * 返回是否应渲染真 iframe（一旦 true 永远 true）。三条触发路径：
 *  1) 初始即命中模块级渲染记忆（cacheKey 之前渲染过）→ useState 初值直接 true（重建瞬间显真图，不闪）；
 *  2) 同步视口预判（useLayoutEffect，首帧绘制前）：已在视口（含提前量）→ 同步置 true（消首屏闪）；
 *  3) IntersectionObserver：后续滚动进入视口 → 置 true 并停止观察。
 * 任一路径置 true 时都写入模块级记忆，供该内容下次重建时走路径 1。
 *
 * @param ref 要观察的容器 ref
 * @param cacheKey 内容稳定身份（previewUrl || sampleHTML），空串则不参与记忆
 * @param rootMargin 视口提前量（默认上下各 400px）
 */
function useInViewOnce(
  ref: React.RefObject<HTMLDivElement | null>,
  cacheKey: string,
  rootMargin = '400px 0px',
): boolean {
  // 路径1：初值即查记忆——加载过的卡片重建后直接可见，不回占位
  const [inView, setInView] = useState<boolean>(() => !!cacheKey && renderedThumbKeys.has(cacheKey))
  const margin = 400 // 与 rootMargin 一致的同步预判提前量

  // 路径2：同步视口预判（绘制前），消除"先占位下一帧再换 iframe"的整屏闪
  useLayoutEffect(() => {
    if (inView) {
      if (cacheKey) renderedThumbKeys.add(cacheKey)
      return
    }
    const el = ref.current
    if (!el) return
    const rect = el.getBoundingClientRect()
    const vh = window.innerHeight || document.documentElement.clientHeight || 0
    if (rect.top < vh + margin && rect.bottom > -margin) {
      if (cacheKey) renderedThumbKeys.add(cacheKey)
      setInView(true)
    }
  }, [ref, cacheKey, inView])

  // 路径3：IntersectionObserver 处理后续滚动进入视口的卡片
  useEffect(() => {
    if (inView) return
    const el = ref.current
    if (!el) return
    const io = new IntersectionObserver(
      entries => {
        for (const e of entries) {
          if (e.isIntersecting) {
            if (cacheKey) renderedThumbKeys.add(cacheKey)
            setInView(true)
            io.disconnect()
            break
          }
        }
      },
      { root: null, rootMargin, threshold: 0 },
    )
    io.observe(el)
    return () => io.disconnect()
  }, [ref, rootMargin, inView, cacheKey])

  return inView
}

// ==================== 组件 Props ====================
interface TemplateThumbProps {
  /** 3D 模板的预览 URL（优先级最高） */
  previewUrl?: string
  /** 普通模板的 HTML 内容（次优先级） */
  sampleHTML?: string
  /** 兜底渲染（最低优先级，亦作懒挂载前的占位） */
  fallback?: React.ReactNode
  /** 容器高度 px（默认 160）。宽度始终 100%自适应 */
  height?: number
  /** iframe title */
  title?: string
}

/**
 * 模板缩略图组件 — 卡片版（固定高度 160px）
 * v2.4：懒挂载 + 会话级渲染记忆 + 同步视口预判（见文件头）。
 */
export default function TemplateThumb({
  previewUrl, sampleHTML, fallback, height = 160, title = '模板预览',
}: TemplateThumbProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const containerW = useContainerWidth(containerRef, 0)
  // 内容自身即稳定缓存身份：3D 用 previewUrl，普通用 sampleHTML
  const inView = useInViewOnce(containerRef, previewUrl || sampleHTML || '')

  // 预处理只随 sampleHTML 变，memo 掉避免每次重渲染重复跑正则
  const sample = useMemo(() => {
    if (!sampleHTML) return null
    const clean = stripScripts(sampleHTML)
    const { w, h } = detectSize(clean)
    return { wrapped: wrapHTML(clean, w, h), w, h }
  }, [sampleHTML])

  // 分支1：3D 模板（preview_url 加载本地 HTML，需脚本 + same-origin 跑 Three.js ESM）
  if (previewUrl) {
    const iframeW = 1440, iframeH = 900
    const scale = containerW > 0 ? containerW / iframeW : 0
    return (
      <div ref={containerRef} style={{
        width: '100%', height: `${height}px`, position: 'relative',
        overflow: 'hidden', background: '#0a0a0a',
      }}>
        {inView && scale > 0 ? (
          <iframe
            src={previewUrl}
            style={{
              width: `${iframeW}px`, height: `${iframeH}px`, border: 'none',
              pointerEvents: 'none',
              transform: `scale(${scale})`, transformOrigin: 'top left',
              position: 'absolute', top: 0, left: 0,
            }}
            sandbox="allow-scripts allow-same-origin"
            title={title}
          />
        ) : (
          <div style={{ width: '100%', height: '100%' }}>{fallback}</div>
        )}
      </div>
    )
  }

  // 分支2：普通模板（sampleHTML 剥脚本 → 包裹 → srcDoc 纯静态渲染，sandbox="" 禁脚本）
  if (sampleHTML && sample) {
    const iframeW = sample.w, iframeH = sample.h
    const scale = containerW > 0 ? containerW / iframeW : 0
    return (
      <div ref={containerRef} style={{
        width: '100%', height: `${height}px`, position: 'relative',
        overflow: 'hidden', background: '#f1f5f9',
      }}>
        {inView && scale > 0 ? (
          <iframe
            srcDoc={sample.wrapped}
            style={{
              width: `${iframeW}px`, height: `${iframeH}px`, border: 'none',
              pointerEvents: 'none',
              transform: `scale(${scale})`, transformOrigin: 'top left',
              position: 'absolute', top: 0, left: 0,
            }}
            sandbox=""
            title={title}
          />
        ) : (
          <div style={{ width: '100%', height: '100%' }}>{fallback}</div>
        )}
      </div>
    )
  }

  // 分支3：兜底渲染
  return (
    <div ref={containerRef} style={{
      width: '100%', height: `${height}px`, overflow: 'hidden',
    }}>{fallback}</div>
  )
}

// ==================== 弹窗大预览版（自适应宽度） ====================

/**
 * 模板缩略图组件 — 弹窗版（高度按 16:9 自适应宽度）
 * 单实例，非性能瓶颈，故不做懒挂载/缓存，保持即时渲染。
 */
export function TemplateThumbAuto({
  previewUrl, sampleHTML, fallback, maxWidth = 1024, title = '模板预览',
}: {
  previewUrl?: string
  sampleHTML?: string
  fallback?: React.ReactNode
  maxWidth?: number
  title?: string
}) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const containerW = useContainerWidth(containerRef, maxWidth)

  // 分支1：3D 模板
  if (previewUrl) {
    const iframeW = 1440, iframeH = 900
    const scale = containerW > 0 ? containerW / iframeW : 0
    const containerH = Math.ceil(iframeH * scale)
    return (
      <div ref={containerRef} style={{
        width: '100%', height: `${containerH}px`, position: 'relative',
        overflow: 'hidden', borderRadius: '14px', border: '1px solid #E5E7EB',
        background: '#0a0a0a',
      }}>
        {scale > 0 && (
          <iframe
            src={previewUrl}
            style={{
              width: `${iframeW}px`, height: `${iframeH}px`, border: 'none',
              transform: `scale(${scale})`, transformOrigin: 'top left',
              position: 'absolute', top: 0, left: 0,
            }}
            sandbox="allow-scripts allow-same-origin"
            title={title}
          />
        )}
      </div>
    )
  }

  // 分支2：普通模板
  if (sampleHTML) {
    const clean = stripScripts(sampleHTML)
    const { w: iframeW, h: iframeH } = detectSize(clean)
    const wrapped = wrapHTML(clean, iframeW, iframeH)
    const scale = containerW > 0 ? containerW / iframeW : 0
    const containerH = Math.ceil(iframeH * scale)
    return (
      <div ref={containerRef} style={{
        width: '100%', height: `${containerH}px`, position: 'relative',
        overflow: 'hidden', borderRadius: '14px', border: '1px solid #E5E7EB',
        background: '#f8fafc',
      }}>
        {scale > 0 && (
          <iframe
            srcDoc={wrapped}
            style={{
              width: `${iframeW}px`, height: `${iframeH}px`, border: 'none',
              transform: `scale(${scale})`, transformOrigin: 'top left',
              position: 'absolute', top: 0, left: 0,
            }}
            sandbox=""
            title={title}
          />
        )}
      </div>
    )
  }

  // 分支3：兜底
  return (
    <div ref={containerRef} style={{
      width: '100%', height: '300px',
      borderRadius: '14px', border: '1px solid #E5E7EB',
      background: '#F8FAFC', display: 'flex', alignItems: 'center', justifyContent: 'center',
      color: '#9CA3AF', fontSize: '14px',
    }}>{fallback || '暂无预览'}</div>
  )
}

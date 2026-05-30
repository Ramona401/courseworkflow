/**
 * 课件模板缩略图组件 — TemplateThumb v2.0 (2026-05-19)
 *
 * 解决核心问题（v1.0 已修）：
 * 1. iframe 内容自带 body margin → 出滚动条 → wrapHTML 强制 overflow:hidden
 * 2. 个人模板是 1920×1080、系统模板是 960×540 → 自动检测内容尺寸
 * 3. 缩放后右下留白 → ResizeObserver 监听容器宽度动态算 scale
 *
 * v2.0 新增（P0 优化）:
 * 4. wrapHTML 兼容完整 HTML 文档：已含 <html>/<!DOCTYPE> 时只注入 <style>，不再嵌套
 * 5. 首屏闪烁修复：useEffect → useLayoutEffect，浏览器绘制前拿到尺寸
 * 6. ResizeObserver 节流：rAF batching + 宽度变化阈值（≥2px 才更新）
 * 7. iframe 懒加载：添加 loading="lazy" 属性，27 个卡片不再同时创建
 *
 * 使用方式：
 *   <TemplateThumb previewUrl="..." sampleHTML="..." fallback={<div/>} />
 *   优先级：previewUrl(3D) > sampleHTML > fallback
 */
import { useEffect, useLayoutEffect, useRef, useState } from 'react'

// ==================== 辅助函数 ====================

/**
 * 检测 HTML 内容声明的画布尺寸
 * 优先匹配 width:XXXpx;height:YYYpx 或 width=XXX height=YYY
 * 默认返回 960×540（16:9 标清）
 */
function detectSize(html: string): { w: number; h: number } {
  // 匹配 width:1920px;height:1080px 等内联样式
  const m1 = html.match(/width\s*:\s*(\d+)px[^}]*height\s*:\s*(\d+)px/i)
  if (m1) return { w: parseInt(m1[1]), h: parseInt(m1[2]) }
  // 匹配 width=1920 height=1080 等属性
  const m2 = html.match(/width\s*=\s*["']?(\d+)["']?[^>]*height\s*=\s*["']?(\d+)["']?/i)
  if (m2) return { w: parseInt(m2[1]), h: parseInt(m2[2]) }
  // 默认 16:9 标清
  return { w: 960, h: 540 }
}

/**
 * 包裹 HTML 片段，强制注入 overflow:hidden 等 reset 样式
 *
 * v2.0 兼容性增强：
 * - 如果原 HTML 已含 <html> 或 <!DOCTYPE>（完整文档）：
 *   只在 <head> 内追加 reset <style>，不外层嵌套
 * - 如果是纯片段（如 <div>...</div>）：
 *   外层包裹完整文档结构
 *
 * 目的：避免双层 <html> 嵌套破坏 CSS 变量作用域
 */
function wrapHTML(html: string, w: number, h: number): string {
  // reset CSS：清零 body margin、隐藏所有滚动条、固定画布尺寸
  const resetCSS = `html,body{margin:0;padding:0;overflow:hidden;width:${w}px;height:${h}px;}::-webkit-scrollbar{display:none;}`

  // 检测是否完整文档（不区分大小写）
  const isFullDoc = /<!DOCTYPE|<html[\s>]/i.test(html)

  if (isFullDoc) {
    // 完整文档：尝试把 reset 样式注入到现有 <head> 内
    if (/<head[\s>]/i.test(html)) {
      return html.replace(/<head[^>]*>/i, match => `${match}<style>${resetCSS}</style>`)
    }
    // 没 <head> 但有 <html>：在 <html> 后插入 <head>
    if (/<html[\s>]/i.test(html)) {
      return html.replace(/<html[^>]*>/i, match => `${match}<head><style>${resetCSS}</style></head>`)
    }
    // 兜底：在最前面追加 <head>（极端情况）
    return `<head><style>${resetCSS}</style></head>${html}`
  }

  // 纯片段：外层包裹完整文档结构
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
 *
 * 设计要点：
 * - useLayoutEffect 替代 useEffect：浏览器绘制前就拿到尺寸，消除首屏闪烁
 * - rAF batching：多次 ResizeObserver 触发合并为一帧内一次 setState
 * - 阈值过滤：宽度变化 < 2px 不触发更新（避免亚像素抖动）
 *
 * @param ref 要监听的 div ref
 * @param initialWidth 初始默认宽度（首次 hook 调用时返回，避免 0）
 * @returns 当前容器宽度（整数 px）
 */
function useContainerWidth(ref: React.RefObject<HTMLDivElement | null>, initialWidth = 0): number {
  const [width, setWidth] = useState(initialWidth)
  const rafIdRef = useRef<number | null>(null)
  const lastWidthRef = useRef<number>(initialWidth)

  useLayoutEffect(() => {
    if (!ref.current) return
    const el = ref.current

    // 首次同步读取宽度（在绘制前拿到，无闪烁）
    const initW = Math.floor(el.clientWidth)
    if (initW > 0) {
      lastWidthRef.current = initW
      setWidth(initW)
    }

    // ResizeObserver 监听后续变化
    const observer = new ResizeObserver(entries => {
      // rAF 节流：取消上一帧未执行的回调
      if (rafIdRef.current !== null) cancelAnimationFrame(rafIdRef.current)
      rafIdRef.current = requestAnimationFrame(() => {
        rafIdRef.current = null
        for (const entry of entries) {
          const newW = Math.floor(entry.contentRect.width)
          // 阈值过滤：变化太小不更新
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

// ==================== 组件 Props ====================
interface TemplateThumbProps {
  /** 3D 模板的预览 URL（优先级最高） */
  previewUrl?: string
  /** 普通模板的 HTML 内容（次优先级） */
  sampleHTML?: string
  /** 兜底渲染（最低优先级） */
  fallback?: React.ReactNode
  /** 容器高度 px（默认 160）。宽度始终 100%自适应 */
  height?: number
  /** iframe title */
  title?: string
}

/**
 * 模板缩略图组件 — 卡片版（固定高度 160px）
 *
 * 容器宽度 100%自适应父元素，iframe 按内容声明尺寸（1920/960）等比缩放至容器宽度
 */
export default function TemplateThumb({
  previewUrl, sampleHTML, fallback, height = 160, title = '模板预览',
}: TemplateThumbProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const containerW = useContainerWidth(containerRef, 0)

  // 分支1：3D 模板（preview_url 加载本地 HTML）
  if (previewUrl) {
    // 3D HTML 默认按 1440×900 设计（参考 StyleSelector v1.3 历史经验）
    const iframeW = 1440, iframeH = 900
    const scale = containerW > 0 ? containerW / iframeW : 0
    return (
      <div ref={containerRef} style={{
        width: '100%', height: `${height}px`, position: 'relative',
        overflow: 'hidden', background: '#0a0a0a',
      }}>
        {scale > 0 && (
          <iframe
            src={previewUrl}
            loading="lazy"
            style={{
              width: `${iframeW}px`, height: `${iframeH}px`, border: 'none',
              pointerEvents: 'none',
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

  // 分支2：普通模板（sampleHTML 包裹后 srcDoc 渲染）
  if (sampleHTML) {
    const { w: iframeW, h: iframeH } = detectSize(sampleHTML)
    const wrapped = wrapHTML(sampleHTML, iframeW, iframeH)
    const scale = containerW > 0 ? containerW / iframeW : 0
    return (
      <div ref={containerRef} style={{
        width: '100%', height: `${height}px`, position: 'relative',
        overflow: 'hidden', background: '#f1f5f9',
      }}>
        {scale > 0 && (
          <iframe
            srcDoc={wrapped}
            loading="lazy"
            style={{
              width: `${iframeW}px`, height: `${iframeH}px`, border: 'none',
              pointerEvents: 'none',
              transform: `scale(${scale})`, transformOrigin: 'top left',
              position: 'absolute', top: 0, left: 0,
            }}
            sandbox="allow-scripts"
            title={title}
          />
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
 *
 * 与 TemplateThumb 的差异：
 * - 高度不固定，按容器宽度 × 9/16 比例算出（保持 1440×900 或 960×540 原比例）
 * - 自带边框和圆角（卡片版没有，由父容器控制）
 *
 * 用于弹窗大预览、编辑器实时预览等场景
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
  // 弹窗版默认初始宽度 = maxWidth，避免首屏 height=0 弹跳
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
            loading="lazy"
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
    const { w: iframeW, h: iframeH } = detectSize(sampleHTML)
    const wrapped = wrapHTML(sampleHTML, iframeW, iframeH)
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
            loading="lazy"
            style={{
              width: `${iframeW}px`, height: `${iframeH}px`, border: 'none',
              transform: `scale(${scale})`, transformOrigin: 'top left',
              position: 'absolute', top: 0, left: 0,
            }}
            sandbox="allow-scripts"
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

// useEffect 标记导出（避免 ESLint unused import 警告，本组件未直接用到 useEffect 但保留兼容性）
export const _unusedHookGuard = useEffect

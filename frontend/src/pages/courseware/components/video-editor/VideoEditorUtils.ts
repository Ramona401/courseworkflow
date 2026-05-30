/**
 * VideoEditorUtils.ts — 视频编辑器纯函数工具集
 *
 * 从 VideoEditorModal.tsx v6.1 拆分而来(v0.42.7)
 *
 * 所有函数均为纯函数无副作用,可独立单测
 *
 * 包含:
 *   - 时间格式化   fmt / fmtMS / fmtSize / timeAgo / fmtDate
 *   - 视频首帧     withFirstFrame / seekToFirstFrame
 *   - 网络        urlToBlobUrl (streaming 进度回调)
 *   - 波形生成     buildWaveformPath (基于 clip.id 哈希的差异化 SVG path)
 *   - 时间轴坐标   clipPxWidth / timeToPixel / pixelToTime / totalPxWidth / buildTicks
 *   - 转场动画     getOutgoingStyle (15种 xfade 对应的 CSS 模拟)
 */
import type { EditorClip } from './VideoEditorTypes'
import { TL_GAP, PX_PER_SEC } from './VideoEditorConstants'
import type React from 'react'

// ==================== 时间格式化 ====================
/** 浮点秒 → "X.Ys" 格式,无穷或负数兜底 0 */
export function fmt(s: number): string {
  return (!isFinite(s) || s < 0) ? '0.0s' : s.toFixed(1) + 's'
}

/** 秒 → "MM:SS" 格式 */
export function fmtMS(s: number): string {
  if (!isFinite(s) || s < 0) return '00:00'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return String(m).padStart(2, '0') + ':' + String(sec).padStart(2, '0')
}

/** 字节数 → 人类可读 (B/KB/MB) */
export function fmtSize(bytes: number): string {
  if (bytes <= 0) return ''
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1048576).toFixed(1) + ' MB'
}

/** ISO 时间 → 相对描述("刚刚"/"X分钟前"/"X小时前"/"X天前") */
export function timeAgo(iso: string): string {
  const d = Date.now() - new Date(iso).getTime()
  if (d < 60000) return '刚刚'
  if (d < 3600000) return Math.floor(d / 60000) + '分钟前'
  if (d < 86400000) return Math.floor(d / 3600000) + '小时前'
  return Math.floor(d / 86400000) + '天前'
}

/** ISO 时间 → "MM-DD HH:mm" 格式 */
export function fmtDate(iso: string): string {
  const dt = new Date(iso)
  const M = String(dt.getMonth() + 1).padStart(2, '0')
  const D = String(dt.getDate()).padStart(2, '0')
  const h = String(dt.getHours()).padStart(2, '0')
  const m = String(dt.getMinutes()).padStart(2, '0')
  return M + '-' + D + ' ' + h + ':' + m
}

// ==================== 视频首帧 ====================
/** URL 加 #t=0.1 fragment,触发浏览器渲染首帧而非显示黑屏 */
export function withFirstFrame(url: string): string {
  if (!url || url.startsWith('blob:')) return url
  return url.includes('#') ? url : url + '#t=0.1'
}

/** onLoadedMetadata 回调: 主动 seek 0.1s,与 fragment 形成双保险 */
export function seekToFirstFrame(e: React.SyntheticEvent<HTMLVideoElement>): void {
  const v = e.currentTarget
  if (v.currentTime < 0.05 && v.duration > 0.2) {
    try { v.currentTime = 0.1 } catch { /* 忽略 seek 失败 */ }
  }
}

// ==================== 网络 (streaming 下载) ====================
/**
 * 把远程 URL 完整下载到内存,转成 blob URL(供 video 元素流畅播放无网络抖动)
 * 支持 streaming 进度回调,60 秒超时
 *
 * 失败场景: HTTP 错误 / 超时 / AbortController 中止
 * 调用方应 try-catch 并降级到直接使用原始 URL
 */
export async function urlToBlobUrl(url: string, onProgress?: (rcv: number, total: number) => void): Promise<string> {
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), 60000)
  try {
    const resp = await fetch(url, { credentials: 'same-origin', signal: ctrl.signal })
    if (!resp.ok) throw new Error('下载失败 HTTP ' + resp.status)
    const total = Number(resp.headers.get('Content-Length') || 0)
    const reader = resp.body?.getReader()
    if (!reader || !total || !onProgress) {
      const b = await resp.blob()
      return URL.createObjectURL(b)
    }
    const chunks: Uint8Array[] = []
    let rcv = 0
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      chunks.push(value)
      rcv += value.length
      onProgress(rcv, total)
    }
    return URL.createObjectURL(new Blob(chunks as BlobPart[], { type: resp.headers.get('Content-Type') || 'video/mp4' }))
  } finally { clearTimeout(timer) }
}

// ==================== 波形生成 (基于 clip.id 哈希) ====================
/**
 * 基于 clip.id 字符 charCode 生成稳定的伪随机数序列
 * 同一 id 永远生成同一波形,不同 id 视觉差异明显
 *
 * 用途: 音频轨上每个片段渲染独有的 SVG 波形,而非全部共用同一固定 path
 * 性能: O(id长度) + O(n点数),纯计算无网络无 FFT,毫秒级
 *
 * 简化版 Mulberry32 PRNG (32位整数线性同余)
 * 钟形包络让首末摆幅较小,中段较大,模拟正常音频淡入淡出
 */
export function buildWaveformPath(clipId: string, points: number = 24): { topD: string; botD: string } {
  // 1. 用 clip.id 累加 charCode 生成种子
  let seed = 0
  for (let i = 0; i < clipId.length; i++) seed = (seed * 31 + clipId.charCodeAt(i)) >>> 0

  // 2. Mulberry32 PRNG
  const rand = () => {
    seed = (seed + 0x6D2B79F5) >>> 0
    let t = seed
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }

  // 3. 生成 points 个 [-1, 1] 间的振幅
  const amps: number[] = []
  for (let i = 0; i < points; i++) {
    const envelope = Math.sin((i / (points - 1)) * Math.PI) // 0→1→0 钟形包络
    amps.push((rand() * 2 - 1) * envelope * 8) // -8 ~ +8 像素摆幅
  }

  // 4. 拼成两条 path: 上波形(主) + 下波形(辅)
  // viewBox 为 "0 0 100 20",x 从 0 到 100 等分布,中线 y=10
  const step = 100 / (points - 1)
  const topPoints: string[] = []
  const botPoints: string[] = []
  for (let i = 0; i < points; i++) {
    const x = (i * step).toFixed(1)
    const yTop = (10 - Math.abs(amps[i])).toFixed(1)
    const yBot = (10 + Math.abs(amps[i]) * 0.7).toFixed(1)
    topPoints.push(`${x},${yTop}`)
    botPoints.push(`${x},${yBot}`)
  }
  return {
    topD: 'M' + topPoints.join(' L '),
    botD: 'M' + botPoints.join(' L '),
  }
}

// ==================== 时间轴坐标转换 ====================
/**
 * 片段视觉宽度(px)
 * 80px 下限保证短片段可点击,300px 上限避免单个长视频霸屏
 * PX_PER_SEC=40 表示「每秒 40 像素」
 */
export function clipPxWidth(dur: number): number {
  return Math.max(80, Math.min(300, dur * PX_PER_SEC))
}

/**
 * 全局时间(秒) → 时间轴左起像素位置
 * 累加每个片段的有效时长(trimEnd - trimStart) + 片段间距(TL_GAP)
 */
export function timeToPixel(t: number, cls: EditorClip[]): number {
  let px = 0
  let acc = 0
  for (let i = 0; i < cls.length; i++) {
    const d = cls[i].trimEnd - cls[i].trimStart
    const w = clipPxWidth(d)
    if (t <= acc + d) return px + ((t - acc) / d) * w
    acc += d
    px += w
    if (i < cls.length - 1) px += TL_GAP
  }
  return px
}

/**
 * 时间轴左起像素位置 → 全局时间(秒)
 * 落在片段内: 按片段宽度比例反推
 * 落在间距上: 归入下一个片段起点
 */
export function pixelToTime(px: number, cls: EditorClip[]): number {
  let accPx = 0
  let accT = 0
  for (let i = 0; i < cls.length; i++) {
    const d = cls[i].trimEnd - cls[i].trimStart
    const w = clipPxWidth(d)
    if (px <= accPx + w) return accT + Math.max(0, ((px - accPx) / w)) * d
    accPx += w
    accT += d
    if (i < cls.length - 1) accPx += TL_GAP
  }
  return accT
}

/** 时间轴总像素宽度(用于滚动区 minWidth) */
export function totalPxWidth(cls: EditorClip[]): number {
  let w = 0
  cls.forEach((c, i) => {
    w += clipPxWidth(c.trimEnd - c.trimStart)
    if (i < cls.length - 1) w += TL_GAP
  })
  return w
}

/** 构建标尺刻度数组(每秒一个 tick,每 5 秒一个大刻度带标签) */
export function buildTicks(td: number, cls: EditorClip[]): { px: number; major: boolean; label: string }[] {
  const ticks: { px: number; major: boolean; label: string }[] = []
  for (let s = 0; s <= Math.ceil(td); s++) {
    if (s > td + 0.1) break
    const major = s % 5 === 0
    ticks.push({ px: timeToPixel(s, cls), major, label: major ? fmtMS(s) : '' })
  }
  return ticks
}

// ==================== 转场动画 (出场片段 CSS) ====================
/**
 * 根据转场类型 + 进度 p (0~1),返回出场片段的 CSS style
 * 进场片段始终保持 opacity:1 在底层,由出场片段的透明/裁剪叠加实现转场视觉
 */
export function getOutgoingStyle(type: string, p: number): React.CSSProperties {
  switch (type) {
    case 'fade': return { opacity: 1 - p }
    case 'fadeblack': return { opacity: p < 0.5 ? 1 - p * 2 : 0 }
    case 'fadewhite': return { opacity: p < 0.5 ? 1 : 2 - p * 2, filter: p < 0.5 ? `brightness(${1 + p * 4})` : 'none' }
    case 'dissolve': return { opacity: 1 - p, filter: `blur(${p * 12}px)` }
    case 'wipeleft': return { clipPath: `inset(0 ${p * 100}% 0 0)` }
    case 'wiperight': return { clipPath: `inset(0 0 0 ${p * 100}%)` }
    case 'wipeup': return { clipPath: `inset(0 0 ${p * 100}% 0)` }
    case 'wipedown': return { clipPath: `inset(${p * 100}% 0 0 0)` }
    case 'slideleft': return { transform: `translateX(-${p * 100}%)` }
    case 'slideright': return { transform: `translateX(${p * 100}%)` }
    case 'slideup': return { transform: `translateY(-${p * 100}%)` }
    case 'slidedown': return { transform: `translateY(${p * 100}%)` }
    case 'circleopen':
    case 'circleclose': {
      const r = 75 * (1 - p)
      return { clipPath: `circle(${r}% at 50% 50%)` }
    }
    default: return {}
  }
}

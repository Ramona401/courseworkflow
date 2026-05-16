/**
 * exportWordImages.ts — Word导出图片处理模块
 *
 * 从 exportWord.ts 拆分,负责:
 *   - 图片URL格式推断
 *   - 图片下载(fetch → Uint8Array)
 *   - 图片尺寸读取(createObjectURL + Image)
 *   - 批量预加载
 *   - 图片段落构建(ImageRun + 降级占位)
 *
 * docx 9.x 浏览器端 ImageRun.data 必须传 Uint8Array(非 ArrayBuffer)
 */

import { Paragraph, TextRun, ImageRun, AlignmentType } from 'docx'

// ==================== 类型定义 ====================

/** 图片 markdown 正则(整行只有一张图片时) */
export const IMG_LINE_RE = /^!\[([^\]]*)\]\(([^)]+)\)$/

/** 单张图片下载结果 */
export interface ImageData {
  alt: string
  url: string
  /** Uint8Array: docx 9.x 浏览器端必须 */
  buffer?: Uint8Array
  width: number
  height: number
  ext: 'png' | 'jpg' | 'gif' | 'bmp'
  error?: string
}

// ==================== 格式推断 ====================

/** 从 URL 推断图片格式(docx ImageRun 需要明确 type) */
export function inferImageType(url: string): 'png' | 'jpg' | 'gif' | 'bmp' {
  const lower = url.toLowerCase()
  if (lower.includes('.png')) return 'png'
  if (lower.includes('.gif')) return 'gif'
  if (lower.includes('.bmp')) return 'bmp'
  if (lower.includes('.webp')) return 'png'
  return 'jpg'
}

/** 推断 MIME type(用于 Blob 构造) */
export function inferMimeType(ext: 'png' | 'jpg' | 'gif' | 'bmp'): string {
  switch (ext) {
    case 'png': return 'image/png'
    case 'gif': return 'image/gif'
    case 'bmp': return 'image/bmp'
    default:    return 'image/jpeg'
  }
}

// ==================== 尺寸读取 ====================

/**
 * 通过 Blob + createObjectURL + Image 读取图片真实尺寸
 * 避免对同一 URL 重复发起网络请求
 */
export async function getImageDimensions(
  arrayBuf: ArrayBuffer,
  mimeType: string
): Promise<{ w: number; h: number }> {
  return new Promise((resolve, reject) => {
    const blob = new Blob([arrayBuf], { type: mimeType })
    const objectUrl = URL.createObjectURL(blob)
    const img = new Image()
    img.onload = () => {
      const w = img.naturalWidth
      const h = img.naturalHeight
      URL.revokeObjectURL(objectUrl)
      resolve({ w, h })
    }
    img.onerror = () => {
      URL.revokeObjectURL(objectUrl)
      reject(new Error('图片解码失败'))
    }
    img.src = objectUrl
  })
}

// ==================== 下载单张图片 ====================

/**
 * 下载单张图片,返回 Uint8Array + 尺寸信息
 * 失败不抛异常,返回 error 字段
 */
export async function downloadImage(alt: string, url: string): Promise<ImageData> {
  const ext = inferImageType(url)
  const result: ImageData = { alt, url, width: 480, height: 320, ext }

  console.log(`[exportWord] 开始下载图片: ${alt || '(无alt)'} → ${url}`)

  try {
    const resp = await fetch(url, { credentials: 'same-origin' })
    if (!resp.ok) {
      result.error = `HTTP ${resp.status}`
      console.warn(`[exportWord] 图片下载失败: ${url} → ${result.error}`)
      return result
    }
    const arrayBuf = await resp.arrayBuffer()
    console.log(`[exportWord] 图片下载成功: ${url}, 大小=${arrayBuf.byteLength} bytes`)

    // ArrayBuffer → Uint8Array(docx 9.x 必须)
    result.buffer = new Uint8Array(arrayBuf)

    // 读取真实尺寸并等比缩放
    try {
      const mimeType = inferMimeType(ext)
      const dims = await getImageDimensions(arrayBuf, mimeType)
      console.log(`[exportWord] 图片原始尺寸: ${dims.w}×${dims.h}`)

      const MAX_W = 480
      if (dims.w > MAX_W) {
        result.width = MAX_W
        result.height = Math.round(dims.h * (MAX_W / dims.w))
      } else if (dims.w > 0) {
        result.width = dims.w
        result.height = dims.h
      }
      console.log(`[exportWord] 图片最终尺寸: ${result.width}×${result.height}`)
    } catch (dimErr) {
      console.warn(`[exportWord] 图片尺寸读取失败,使用默认 480×320:`, dimErr)
    }

    return result
  } catch (e: unknown) {
    result.error = e instanceof Error ? e.message : '下载失败'
    console.error(`[exportWord] 图片下载异常: ${url}`, e)
    return result
  }
}

// ==================== 批量预加载 ====================

/**
 * 预扫描 markdown,提取所有图片 URL,并行下载
 * @param lines 预处理后的行数组
 * @returns url → ImageData 映射
 */
export async function preloadAllImages(lines: string[]): Promise<Map<string, ImageData>> {
  const urls: Array<{ alt: string; url: string }> = []
  for (const line of lines) {
    const m = line.trim().match(IMG_LINE_RE)
    if (m) urls.push({ alt: m[1], url: m[2] })
  }

  console.log(`[exportWord] 扫描到 ${urls.length} 张图片需要下载`)
  if (urls.length === 0) return new Map()

  const results = await Promise.all(urls.map(({ alt, url }) => downloadImage(alt, url)))
  const map = new Map<string, ImageData>()
  for (const r of results) map.set(r.url, r)

  const ok = results.filter(r => r.buffer).length
  console.log(`[exportWord] 图片下载完成: ${ok} 成功, ${results.length - ok} 失败`)

  return map
}

// ==================== 构建图片段落 ====================

/**
 * 将图片数据构建为 docx Paragraph 数组
 * ImageRun 创建用 try-catch,失败降级为红色占位文字
 * @returns 1~2 个 Paragraph(图片 + 可选图注)
 */
export function buildImageParagraphs(img: ImageData): Paragraph[] {
  const result: Paragraph[] = []

  if (img.buffer) {
    try {
      console.log(`[exportWord] 创建 ImageRun: ${img.url}, ` +
        `${img.width}×${img.height}, type=${img.ext}, ` +
        `bufferType=${img.buffer.constructor.name}, bufferLen=${img.buffer.byteLength}`)

      const imageRun = new ImageRun({
        data: img.buffer,
        transformation: { width: img.width, height: img.height },
        type: img.ext,
      })

      result.push(new Paragraph({
        alignment: AlignmentType.CENTER,
        children: [imageRun],
        spacing: { before: 100, after: 60 },
      }))

      console.log(`[exportWord] ImageRun 创建成功: ${img.url}`)

      if (img.alt) {
        result.push(new Paragraph({
          alignment: AlignmentType.CENTER,
          children: [new TextRun({ text: img.alt, size: 18, color: '9CA3AF', italics: true })],
          spacing: { before: 0, after: 80 },
        }))
      }
    } catch (err) {
      console.error(`[exportWord] ImageRun 创建失败: ${img.url}`, err)
      result.push(new Paragraph({
        alignment: AlignmentType.CENTER,
        children: [new TextRun({
          text: `[图片嵌入失败:${img.alt || img.url}]`,
          size: 20, color: 'DC2626', italics: true,
        })],
        spacing: { before: 60, after: 60 },
      }))
    }
  } else {
    console.warn(`[exportWord] 图片无数据,降级为占位: ${img.url}, error=${img.error}`)
    result.push(new Paragraph({
      alignment: AlignmentType.CENTER,
      children: [new TextRun({
        text: `[图片加载失败:${img.alt || '未命名'} - ${img.error || '未知错误'}]`,
        size: 20, color: 'DC2626', italics: true,
      })],
      spacing: { before: 60, after: 60 },
    }))
  }

  return result
}

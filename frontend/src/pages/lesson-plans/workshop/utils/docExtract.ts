/**
 * docExtract.ts — 浏览器端文档文字提取工具（PDF/Word）
 *
 * 参考资料附件保持“不落盘、不落库”：
 * - DOCX：浏览器用 JSZip 提取文字；
 * - 文字型 PDF：浏览器用 pdf.js 逐页读取文字层；
 * - 扫描型/混合型 PDF：缺少可用文字层的页面逐页渲染为 JPEG，
 *   再由附件弹窗调用后端多模态能力忠实转录。
 *
 * 依赖库一律走本站同源相对路径，避免固定域名、跨域或 Worker 访问失败。
 */

const LIBS_BASE = '/uploads/courseware-assets/libs'
const JSZIP_URL = `${LIBS_BASE}/jszip/3.10.1/jszip.min.js`
const PDFJS_URL = `${LIBS_BASE}/pdfjs-dist/3.11.174/build/pdf.min.js`
const PDFJS_WORKER_URL = `${LIBS_BASE}/pdfjs-dist/3.11.174/build/pdf.worker.min.js`

export const MAX_REF_FILE_SIZE = 10 * 1024 * 1024
export const REF_COMPRESS_THRESHOLD = 3000
export const MAX_SCAN_PDF_PAGES = 12

const MIN_USABLE_PDF_TEXT_CHARS = 20
const PDF_RENDER_TARGET_WIDTH = 1600
const PDF_RENDER_MAX_PIXELS = 5_000_000

interface PdfTextItem {
  str?: string
}

interface PdfViewport {
  width: number
  height: number
}

interface PdfPageProxy {
  getTextContent: () => Promise<{ items: PdfTextItem[] }>
  getViewport: (options: { scale: number }) => PdfViewport
  render: (options: {
    canvasContext: CanvasRenderingContext2D
    viewport: PdfViewport
  }) => { promise: Promise<void> }
  cleanup?: () => void
}

interface PdfDocumentProxy {
  numPages: number
  getPage: (pageNumber: number) => Promise<PdfPageProxy>
}

interface PdfJsLibrary {
  GlobalWorkerOptions: { workerSrc: string }
  getDocument: (options: { data: ArrayBuffer }) => {
    promise: Promise<PdfDocumentProxy>
  }
}

export interface ExtractedDocumentPage {
  pageNumber: number
  text: string
  imageDataUri?: string
}

export interface ExtractProgress {
  phase: 'loading_library' | 'reading_document' | 'rendering_page'
  current?: number
  total?: number
  message: string
}

export interface ExtractResult {
  text: string
  charCount: number
  pages?: ExtractedDocumentPage[]
  totalPages?: number
  scanPageCount?: number
}

function loadScript(src: string, globalKey: string): Promise<void> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  if ((window as any)[globalKey]) return Promise.resolve()

  return new Promise((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(
      `script[src="${src}"]`,
    )
    if (existing) {
      existing.addEventListener('load', () => resolve(), { once: true })
      existing.addEventListener(
        'error',
        () => reject(new Error(`本地解析组件加载失败：${src}`)),
        { once: true },
      )
      return
    }

    const script = document.createElement('script')
    script.src = src
    script.async = true
    script.onload = () => resolve()
    script.onerror = () =>
      reject(new Error(`本地解析组件加载失败：${src}`))
    document.head.appendChild(script)
  })
}

export async function parseDocxFile(file: File): Promise<string> {
  await loadScript(JSZIP_URL, 'JSZip')
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const JSZip = (window as any).JSZip
  if (!JSZip) {
    throw new Error('Word 解析组件未正确加载，请刷新页面后重试')
  }

  const zip = await JSZip.loadAsync(await file.arrayBuffer())
  const docXmlFile = zip.file('word/document.xml')
  if (!docXmlFile) throw new Error('该文件不是有效的 DOCX 文档')

  const xmlDoc = new DOMParser().parseFromString(
    await docXmlFile.async('string'),
    'application/xml',
  )
  const lines: string[] = []
  xmlDoc.querySelectorAll('w\\:p, p').forEach(paragraph => {
    const line = Array.from(paragraph.querySelectorAll('w\\:t, t'))
      .map(text => text.textContent || '')
      .join('')
      .trim()
    if (line) lines.push(line)
  })
  return lines.join('\n')
}

function hasUsablePdfText(text: string): boolean {
  const compact = text.replace(/\s/g, '')
  if (compact.length < MIN_USABLE_PDF_TEXT_CHARS) return false

  const meaningful =
    compact.match(/[\u3400-\u9FFF\uF900-\uFAFFA-Za-z0-9]/g)?.length || 0
  return meaningful / Math.max(compact.length, 1) >= 0.55
}

async function renderPdfPageToJpeg(page: PdfPageProxy): Promise<string> {
  const baseViewport = page.getViewport({ scale: 1 })
  if (baseViewport.width <= 0 || baseViewport.height <= 0) {
    throw new Error('PDF 页面尺寸无效')
  }

  let scale = Math.max(
    1,
    Math.min(PDF_RENDER_TARGET_WIDTH / baseViewport.width, 3),
  )
  let viewport = page.getViewport({ scale })
  const pixelCount = viewport.width * viewport.height

  if (pixelCount > PDF_RENDER_MAX_PIXELS) {
    scale *= Math.sqrt(PDF_RENDER_MAX_PIXELS / pixelCount)
    viewport = page.getViewport({ scale })
  }

  const canvas = document.createElement('canvas')
  canvas.width = Math.max(1, Math.round(viewport.width))
  canvas.height = Math.max(1, Math.round(viewport.height))

  const canvasContext = canvas.getContext('2d', { alpha: false })
  if (!canvasContext) throw new Error('浏览器无法创建 PDF 页面画布')

  canvasContext.fillStyle = '#FFFFFF'
  canvasContext.fillRect(0, 0, canvas.width, canvas.height)
  await page.render({ canvasContext, viewport }).promise

  const dataUri = canvas.toDataURL('image/jpeg', 0.9)
  canvas.width = 1
  canvas.height = 1

  if (!dataUri.startsWith('data:image/jpeg;base64,')) {
    throw new Error('PDF 页面图片生成失败')
  }
  return dataUri
}

export async function parsePdfFileDetailed(
  file: File,
  onProgress: (progress: ExtractProgress) => void = () => {},
): Promise<ExtractResult> {
  onProgress({
    phase: 'loading_library',
    message: '正在加载本站 PDF 解析组件…',
  })
  await loadScript(PDFJS_URL, 'pdfjsLib')

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const pdfjsLib = (window as any).pdfjsLib as PdfJsLibrary | undefined
  if (!pdfjsLib) {
    throw new Error('PDF 解析组件未正确加载，请刷新页面后重试')
  }
  pdfjsLib.GlobalWorkerOptions.workerSrc = PDFJS_WORKER_URL

  onProgress({ phase: 'reading_document', message: '正在读取 PDF 页面…' })
  const pdf = await pdfjsLib
    .getDocument({ data: await file.arrayBuffer() })
    .promise

  if (pdf.numPages <= 0) throw new Error('PDF 中没有可读取的页面')
  if (pdf.numPages > MAX_SCAN_PDF_PAGES) {
    throw new Error(
      `PDF 共 ${pdf.numPages} 页，当前最多处理 ${MAX_SCAN_PDF_PAGES} 页，请拆分后上传`,
    )
  }

  const pages: ExtractedDocumentPage[] = []
  let scanPageCount = 0

  for (let pageNumber = 1; pageNumber <= pdf.numPages; pageNumber++) {
    const page = await pdf.getPage(pageNumber)
    try {
      onProgress({
        phase: 'reading_document',
        current: pageNumber,
        total: pdf.numPages,
        message: `正在读取第 ${pageNumber}/${pdf.numPages} 页文字层…`,
      })
      const textContent = await page.getTextContent()
      const pageText = textContent.items
        .map(item => (typeof item.str === 'string' ? item.str : ''))
        .join(' ')
        .replace(/\s+/g, ' ')
        .trim()

      if (hasUsablePdfText(pageText)) {
        pages.push({ pageNumber, text: pageText })
        continue
      }

      onProgress({
        phase: 'rendering_page',
        current: pageNumber,
        total: pdf.numPages,
        message: `第 ${pageNumber} 页没有可用文字层，正在生成清晰页面图…`,
      })
      pages.push({
        pageNumber,
        text: '',
        imageDataUri: await renderPdfPageToJpeg(page),
      })
      scanPageCount++
    } finally {
      page.cleanup?.()
    }
  }

  const text = pages
    .filter(page => page.text.trim())
    .map(page => `【第${page.pageNumber}页】\n${page.text.trim()}`)
    .join('\n\n')

  return {
    text,
    charCount: text.replace(/\s/g, '').length,
    pages,
    totalPages: pdf.numPages,
    scanPageCount,
  }
}

/**
 * 保留历史导出。只需要纯文字的旧调用方遇到扫描页时给出明确错误；
 * 扫描 PDF 应通过 extractDocFile 返回逐页图片继续处理。
 */
export async function parsePdfFile(file: File): Promise<string> {
  const result = await parsePdfFileDetailed(file)
  if ((result.scanPageCount || 0) > 0) {
    throw new Error('该 PDF 含扫描页面，请通过参考资料附件的逐页识别流程处理')
  }
  return result.text
}

export async function extractDocFile(
  file: File,
  onProgress: (progress: ExtractProgress) => void = () => {},
): Promise<ExtractResult> {
  if (file.size > MAX_REF_FILE_SIZE) {
    throw new Error('文件超过 10MB，请压缩或拆分后再上传')
  }

  const name = file.name.toLowerCase()
  if (name.endsWith('.doc')) {
    throw new Error(
      '暂不支持老版本 .doc 格式，请另存为 .docx 后上传，或直接粘贴文字',
    )
  }

  if (name.endsWith('.docx')) {
    onProgress({ phase: 'reading_document', message: '正在读取 Word 文档…' })
    const text = (await parseDocxFile(file)).trim()
    if (!text) {
      throw new Error('文档内容为空或无法提取文字，请检查文件或改用粘贴')
    }
    return { text, charCount: text.replace(/\s/g, '').length }
  }

  if (name.endsWith('.pdf')) {
    const result = await parsePdfFileDetailed(file, onProgress)
    if (!result.text.trim() && (result.scanPageCount || 0) === 0) {
      throw new Error('PDF 内容为空或页面无法读取')
    }
    return result
  }

  throw new Error('仅支持 .docx 或 .pdf 文件')
}

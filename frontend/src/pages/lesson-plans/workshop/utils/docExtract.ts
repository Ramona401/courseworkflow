/**
 * docExtract.ts — 浏览器端文档文字提取工具（PDF/Word）
 *
 * 从 ImportPlanModal.tsx 抽出的 loadScript/parseDocxFile/parsePdfFile，供"参考资料附件"
 * 功能复用（对话模式 + 后续专家模式共用同一份，避免两处各写一遍）。
 *
 * 纯浏览器端、零落盘零落库、零 npm 依赖：
 *   - docx：运行时加载 JSZip 解压取 word/document.xml，DOMParser 提取 <w:t> 文字；
 *   - pdf ：运行时加载 pdf.js 逐页取文字（仅文字型 PDF，扫描件提取为空）。
 *
 * 自托管说明（批次0b 换源，2026-07-08）：
 *   JSZip 3.10.1 与 pdf.js 3.11.174 已从 cdnjs.cloudflare.com 迁移为本服务器自托管，
 *   版本与原 CDN 完全一致，纯换源零行为变化。文件位于
 *   /www/wwwroot/tedna/uploads/courseware-assets/libs/{jszip,pdfjs-dist}/ 下，
 *   走 Nginx /uploads/courseware-assets/ 映射（CORS 头 + 30 天缓存）。
 *   注意：pdf.js 刻意锁定 3.11.174 不升 4.x（4.x 改为 .mjs 模块会破坏现有加载方式）。
 *
 * 与 ImportPlanModal 的关系：ImportPlanModal 仍保留自己那份实现（两份并存无害，
 * 后续可选收敛）；两处的库地址已在同一批次统一换为自托管。
 *
 * 相较 ImportPlanModal 版新增：
 *   - MAX_FILE_SIZE 10MB 体积上限（超限直接抛错，避免大文件卡浏览器）；
 *   - .doc（老格式二进制）显式拦截：浏览器端无法解析，提示改用 .docx 或粘贴。
 */

/** 自托管前端库基础地址（详见文件头"自托管说明"） */
const LIBS_BASE = 'https://workflow.pkuailab.com/uploads/courseware-assets/libs'

/** JSZip 自托管地址（原 cdnjs jszip@3.10.1 纯换源） */
const JSZIP_URL = LIBS_BASE + '/jszip/3.10.1/jszip.min.js'

/** pdf.js 自托管地址（原 cdnjs pdf.js@3.11.174 纯换源，主库 + worker 两个文件） */
const PDFJS_URL = LIBS_BASE + '/pdfjs-dist/3.11.174/build/pdf.min.js'
const PDFJS_WORKER_URL = LIBS_BASE + '/pdfjs-dist/3.11.174/build/pdf.worker.min.js'

/** 参考资料附件单文件体积上限（10MB） */
export const MAX_REF_FILE_SIZE = 10 * 1024 * 1024

/** 长文档压缩阈值（rune/字符数）：达到此长度前端才调后端压缩端点，短文档直接原文注入 */
export const REF_COMPRESS_THRESHOLD = 3000

/**
 * loadScript — 运行时动态加载脚本（幂等：已加载则直接 resolve）
 */
function loadScript(src: string, globalKey: string): Promise<void> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  if ((window as any)[globalKey]) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const existing = document.querySelector(`script[src="${src}"]`)
    if (existing) {
      existing.addEventListener('load', () => resolve())
      existing.addEventListener('error', () => reject(new Error(`加载失败: ${src}`)))
      return
    }
    const script = document.createElement('script')
    script.src = src
    script.onload = () => resolve()
    script.onerror = () => reject(new Error(`脚本加载失败: ${src}`))
    document.head.appendChild(script)
  })
}

/**
 * parseDocxFile — 解析 .docx 提取纯文本（JSZip + DOMParser）
 */
export async function parseDocxFile(file: File): Promise<string> {
  await loadScript(JSZIP_URL, 'JSZip')
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const JSZip = (window as any).JSZip
  if (!JSZip) throw new Error('JSZip加载失败')

  const arrayBuffer = await file.arrayBuffer()
  const zip = await JSZip.loadAsync(arrayBuffer)

  const docXmlFile = zip.file('word/document.xml')
  if (!docXmlFile) throw new Error('不是有效的docx文件')

  const xmlStr = await docXmlFile.async('string')
  const parser = new DOMParser()
  const xmlDoc = parser.parseFromString(xmlStr, 'application/xml')

  const paragraphs = xmlDoc.querySelectorAll('w\\:p, p')
  const lines: string[] = []
  paragraphs.forEach(para => {
    const texts = para.querySelectorAll('w\\:t, t')
    const line = Array.from(texts).map(t => t.textContent || '').join('').trim()
    if (line) lines.push(line)
  })
  return lines.join('\n')
}

/**
 * parsePdfFile — 解析文字型 PDF 提取纯文本（pdf.js 逐页）
 */
export async function parsePdfFile(file: File): Promise<string> {
  await loadScript(PDFJS_URL, 'pdfjsLib')
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const pdfjsLib = (window as any).pdfjsLib
  if (!pdfjsLib) throw new Error('pdf.js加载失败')

  pdfjsLib.GlobalWorkerOptions.workerSrc = PDFJS_WORKER_URL

  const arrayBuffer = await file.arrayBuffer()
  const pdf = await pdfjsLib.getDocument({ data: arrayBuffer }).promise
  const textParts: string[] = []
  for (let i = 1; i <= pdf.numPages; i++) {
    const page = await pdf.getPage(i)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const textContent = await page.getTextContent()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const pageText = textContent.items.map((item: any) => item.str).join(' ').trim()
    if (pageText) textParts.push(pageText)
  }
  return textParts.join('\n\n')
}

/** 提取结果 */
export interface ExtractResult {
  text: string
  /** 字符数（去空白后），供前端判断是否需压缩 */
  charCount: number
}

/**
 * extractDocFile — 统一入口：按文件类型分发提取，含体积/格式校验。
 *
 * 抛错情形（调用方 catch 后按 message 提示）：
 *   - 体积超限、.doc 老格式、非 docx/pdf、脚本加载失败、解析失败、内容为空（扫描件）。
 */
export async function extractDocFile(file: File): Promise<ExtractResult> {
  if (file.size > MAX_REF_FILE_SIZE) {
    throw new Error('文件超过 10MB，请压缩或拆分后再上传')
  }
  const name = file.name.toLowerCase()

  // .doc 老格式（非 .docx）：浏览器端无法解析，显式拦截
  if (name.endsWith('.doc')) {
    throw new Error('暂不支持老版本 .doc 格式，请另存为 .docx 后上传，或直接粘贴文字')
  }

  let text = ''
  if (name.endsWith('.docx')) {
    text = await parseDocxFile(file)
  } else if (name.endsWith('.pdf')) {
    text = await parsePdfFile(file)
  } else {
    throw new Error('仅支持 .docx 或文字版 .pdf 文件')
  }

  text = text.trim()
  if (!text) {
    if (name.endsWith('.pdf')) {
      throw new Error('该 PDF 为扫描件或无可提取文字，请改用课本图片拍照或复制文字后粘贴')
    }
    throw new Error('文档内容为空或无法提取文字，请检查文件或改用粘贴')
  }

  return { text, charCount: text.replace(/\s/g, '').length }
}

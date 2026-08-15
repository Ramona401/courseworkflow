/**
 * conversationAttachmentProcessing.ts — 对话附件解析与教材图片桥
 *
 * 只负责“把一个文件变成可注入文字”：
 * DOCX/PDF复用既有docExtract；PPTX浏览器读取slide XML；
 * 图片和扫描PDF复用现有视觉转录；长内容复用现有参考资料压缩。
 */

import {
  compressRefMaterial,
  transcribeRefMaterialPage,
} from '@/api/lesson-plans-ref'
import {
  extractDocFile,
  MAX_REF_FILE_SIZE,
  REF_COMPRESS_THRESHOLD,
  type ExtractedDocumentPage,
} from '../utils/docExtract'
import {
  isConversationAttachmentImage,
} from './conversationAttachmentQueue'

const JSZIP_URL =
  '/uploads/courseware-assets/libs/jszip/3.10.1/jszip.min.js'
const PPT_MAX_SLIDES = 100
const PPT_XML_MAX_CHARS = 2_000_000
const IMAGE_MAX_EDGE = 1800
const IMAGE_MAX_PIXELS = 4_000_000
const VISION_CONCURRENCY = 2

export const CONVERSATION_ATTACHMENT_ACCEPT =
  '.docx,.pdf,.pptx,.txt,.md,image/jpeg,image/png,image/webp'

let pendingTextbookImageFiles: File[] = []

export function queuePendingTextbookImageFiles(
  files: File[],
) {
  pendingTextbookImageFiles =
    files.filter(isConversationAttachmentImage)
}

export function consumePendingTextbookImageFiles(): File[] {
  const files = pendingTextbookImageFiles
  pendingTextbookImageFiles = []
  return files
}

function validateConversationAttachmentFile(
  file: File,
) {
  const name = file.name.toLowerCase()

  if (!file.name.trim()) {
    throw new Error('文件名为空，无法读取')
  }

  if (file.size <= 0) {
    throw new Error('文件内容为空，请重新选择')
  }

  if (file.size > MAX_REF_FILE_SIZE) {
    throw new Error(
      '文件超过 10MB，请压缩或拆分后再添加',
    )
  }

  if (name.endsWith('.doc')) {
    throw new Error(
      '暂不支持老版本 .doc，请用 Word/WPS 另存为 .docx 后再添加。',
    )
  }

  if (name.endsWith('.ppt')) {
    throw new Error(
      '暂不支持老版本 .ppt，请用 PowerPoint/WPS 另存为 .pptx 后再添加。',
    )
  }

  if (
    isConversationAttachmentImage(file) ||
    name.endsWith('.docx') ||
    name.endsWith('.pdf') ||
    name.endsWith('.pptx') ||
    name.endsWith('.txt') ||
    name.endsWith('.md')
  ) {
    return
  }

  throw new Error(
    '暂不支持这个文件格式。请使用 PDF、DOCX、PPTX、TXT/MD 或 JPG/PNG/WEBP 图片。',
  )
}

function loadJSZip(): Promise<void> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  if ((window as any).JSZip) {
    return Promise.resolve()
  }

  return new Promise((resolve, reject) => {
    const existing =
      document.querySelector<HTMLScriptElement>(
        `script[src="${JSZIP_URL}"]`,
      )

    if (existing) {
      existing.addEventListener(
        'load',
        () => resolve(),
        { once: true },
      )
      existing.addEventListener(
        'error',
        () =>
          reject(
            new Error(
              'PPT 解析组件加载失败，请刷新页面后重试',
            ),
          ),
        { once: true },
      )
      return
    }

    const script = document.createElement('script')
    script.src = JSZIP_URL
    script.async = true
    script.onload = () => resolve()
    script.onerror = () =>
      reject(
        new Error(
          'PPT 解析组件加载失败，请刷新页面后重试',
        ),
      )
    document.head.appendChild(script)
  })
}

async function parsePPTX(
  file: File,
  onProgress: (message: string) => void,
): Promise<string> {
  onProgress('正在读取 PPT 页面…')
  await loadJSZip()

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const JSZip = (window as any).JSZip
  if (!JSZip) {
    throw new Error(
      'PPT 解析组件未正确加载，请刷新页面后重试',
    )
  }

  let zip
  try {
    zip = await JSZip.loadAsync(
      await file.arrayBuffer(),
    )
  } catch {
    throw new Error(
      '该文件不是有效的 PPTX 文档，请用 PowerPoint/WPS 重新另存为 .pptx 后再添加。',
    )
  }

  const slides =
    Object.keys(zip.files)
      .map(name => {
        const match =
          /^ppt\/slides\/slide(\d+)\.xml$/.exec(
            name,
          )

        return match
          ? {
              name,
              number: Number(match[1]),
            }
          : null
      })
      .filter(
        (
          item,
        ): item is {
          name: string
          number: number
        } => Boolean(item),
      )
      .sort(
        (a, b) =>
          a.number - b.number,
      )
      .slice(0, PPT_MAX_SLIDES)

  if (slides.length === 0) {
    throw new Error(
      'PPTX 内没有找到可读取的幻灯片，请检查文件是否损坏。',
    )
  }

  const parts: string[] = []

  for (
    let index = 0;
    index < slides.length;
    index++
  ) {
    const slide = slides[index]

    onProgress(
      `正在读取 PPT 第 ${index + 1}/${slides.length} 页…`,
    )

    const xml =
      await zip.files[slide.name].async(
        'string',
      )

    if (xml.length > PPT_XML_MAX_CHARS) {
      throw new Error(
        `PPT 第 ${slide.number} 页结构过大，无法安全解析，请简化该页后重试。`,
      )
    }

    const xmlDoc =
      new DOMParser().parseFromString(
        xml,
        'application/xml',
      )

    if (
      xmlDoc.querySelector(
        'parsererror',
      )
    ) {
      throw new Error(
        `PPT 第 ${slide.number} 页结构异常，无法解析。`,
      )
    }

    const text =
      Array.from(
        xmlDoc.getElementsByTagNameNS(
          '*',
          't',
        ),
      )
        .map(
          node =>
            node.textContent || '',
        )
        .join(' ')
        .replace(/\s+/g, ' ')
        .trim()

    if (text) {
      parts.push(
        `【第${slide.number}页】\n${text}`,
      )
    }
  }

  if (parts.length === 0) {
    throw new Error(
      'PPT 中没有提取到可用文字；如果主要是图片，请导出为 PDF 或图片后再添加。',
    )
  }

  return parts.join('\n\n')
}

async function transcribeScanningPages(
  pages: ExtractedDocumentPage[],
  fileName: string,
  totalPages: number,
  onProgress: (message: string) => void,
): Promise<string> {
  const pageText =
    new Map<number, string>()

  const scanPages =
    pages.filter(
      page =>
        !page.text.trim() &&
        Boolean(
          page.imageDataUri,
        ),
    )

  pages.forEach(page => {
    if (page.text.trim()) {
      pageText.set(
        page.pageNumber,
        page.text.trim(),
      )
    }
  })

  let cursor = 0
  let completed = 0

  const worker = async () => {
    while (true) {
      const index = cursor++
      if (
        index >=
        scanPages.length
      ) {
        return
      }

      const page =
        scanPages[index]

      if (!page.imageDataUri) {
        throw new Error(
          `PDF 第 ${page.pageNumber} 页没有可识别的页面图`,
        )
      }

      onProgress(
        `正在识别扫描页 ${completed + 1}/${scanPages.length}（PDF 第 ${page.pageNumber} 页）…`,
      )

      try {
        const response =
          await transcribeRefMaterialPage({
            image_data_uri:
              page.imageDataUri,
            file_name: fileName,
            page_number:
              page.pageNumber,
            total_pages:
              totalPages,
          })

        const text =
          (response.text || '')
            .trim()

        if (!text) {
          throw new Error(
            '识别结果为空',
          )
        }

        pageText.set(
          page.pageNumber,
          text,
        )

        page.imageDataUri =
          undefined

        completed++
      } catch (error) {
        const detail =
          error instanceof Error
            ? error.message
            : '未知错误'

        throw new Error(
          `PDF 第 ${page.pageNumber} 页识别失败：${detail}`,
        )
      }
    }
  }

  if (scanPages.length > 0) {
    await Promise.all(
      Array.from(
        {
          length:
            Math.min(
              VISION_CONCURRENCY,
              scanPages.length,
            ),
        },
        () => worker(),
      ),
    )
  }

  return pages
    .map(page => {
      const text =
        pageText.get(
          page.pageNumber,
        )

      return text
        ? `【第${page.pageNumber}页】\n${text}`
        : ''
    })
    .filter(Boolean)
    .join('\n\n')
}

async function imageFileToJpegDataURI(
  file: File,
): Promise<string> {
  let bitmap: ImageBitmap

  try {
    bitmap =
      await createImageBitmap(file)
  } catch {
    throw new Error(
      '图片无法读取，请确认文件没有损坏并使用 JPG、PNG 或 WEBP。',
    )
  }

  try {
    const {
      width,
      height,
    } = bitmap

    if (
      width <= 0 ||
      height <= 0
    ) {
      throw new Error(
        '图片尺寸无效',
      )
    }

    let scale =
      Math.min(
        1,
        IMAGE_MAX_EDGE /
          Math.max(
            width,
            height,
          ),
      )

    const pixels =
      width *
      height *
      scale *
      scale

    if (
      pixels >
      IMAGE_MAX_PIXELS
    ) {
      scale *= Math.sqrt(
        IMAGE_MAX_PIXELS /
          pixels,
      )
    }

    const canvas =
      document.createElement(
        'canvas',
      )

    canvas.width =
      Math.max(
        1,
        Math.round(
          width * scale,
        ),
      )

    canvas.height =
      Math.max(
        1,
        Math.round(
          height * scale,
        ),
      )

    const context =
      canvas.getContext(
        '2d',
        {
          alpha: false,
        },
      )

    if (!context) {
      throw new Error(
        '浏览器无法创建图片画布',
      )
    }

    context.fillStyle =
      '#FFFFFF'
    context.fillRect(
      0,
      0,
      canvas.width,
      canvas.height,
    )
    context.drawImage(
      bitmap,
      0,
      0,
      canvas.width,
      canvas.height,
    )

    const dataURI =
      canvas.toDataURL(
        'image/jpeg',
        0.88,
      )

    canvas.width = 1
    canvas.height = 1

    if (
      !dataURI.startsWith(
        'data:image/jpeg;base64,',
      )
    ) {
      throw new Error(
        '图片转换失败',
      )
    }

    return dataURI
  } finally {
    bitmap.close()
  }
}

async function extractConversationAttachmentText(
  file: File,
  onProgress: (
    message: string,
  ) => void,
): Promise<string> {
  const name =
    file.name.toLowerCase()

  if (
    isConversationAttachmentImage(
      file,
    )
  ) {
    onProgress(
      '正在读取图片…',
    )

    const imageDataURI =
      await imageFileToJpegDataURI(
        file,
      )

    onProgress(
      '正在识别图片文字与内容…',
    )

    const response =
      await transcribeRefMaterialPage({
        image_data_uri:
          imageDataURI,
        file_name: file.name,
        page_number: 1,
        total_pages: 1,
      })

    const text =
      (response.text || '')
        .trim()

    if (!text) {
      throw new Error(
        '图片识别结果为空，请换一张更清晰的图片后重试。',
      )
    }

    return text
  }

  if (
    name.endsWith(
      '.txt',
    ) ||
    name.endsWith(
      '.md',
    )
  ) {
    onProgress(
      '正在读取文本文件…',
    )

    const text =
      (await file.text())
        .trim()

    if (!text) {
      throw new Error(
        '文本文件内容为空',
      )
    }

    return text
  }

  if (
    name.endsWith(
      '.pptx',
    )
  ) {
    return parsePPTX(
      file,
      onProgress,
    )
  }

  const extracted =
    await extractDocFile(
      file,
      progress => {
        onProgress(
          progress.message,
        )
      },
    )

  let text =
    extracted.text.trim()

  if (
    extracted.pages?.length
  ) {
    text =
      await transcribeScanningPages(
        extracted.pages,
        file.name,
        extracted.totalPages ||
          extracted.pages.length,
        onProgress,
      )
  }

  if (!text.trim()) {
    throw new Error(
      '没有获得可用文字，请检查文件内容后重试。',
    )
  }

  return text.trim()
}

export async function prepareConversationAttachment(
  file: File,
  onProgress: (
    message: string,
  ) => void,
): Promise<{
  text: string
  charCount: number
}> {
  validateConversationAttachmentFile(
    file,
  )

  let text =
    await extractConversationAttachmentText(
      file,
      onProgress,
    )

  const charCount =
    text.replace(
      /\s/g,
      '',
    ).length

  if (charCount <= 0) {
    throw new Error(
      '没有获得可用内容，请检查文件后重试。',
    )
  }

  if (
    charCount >=
    REF_COMPRESS_THRESHOLD
  ) {
    onProgress(
      '内容较长，正在提炼并保留关键事实与页码…',
    )

    const response =
      await compressRefMaterial({
        content: text,
        file_name:
          file.name,
      })

    text =
      (
        response.compressed ||
        ''
      ).trim() ||
      text
  }

  return {
    text,
    charCount,
  }
}

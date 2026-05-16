/**
 * exportWord.ts — 教案导出 Word 文档工具 v5
 *
 * 职责: Markdown解析 + 元信息表格 + Document构建 + 打包下载
 * 图片处理已拆分到 exportWordImages.ts
 */

import {
  Document,
  Packer,
  Paragraph,
  TextRun,
  HeadingLevel,
  AlignmentType,
  BorderStyle,
  Table,
  TableRow,
  TableCell,
  WidthType,
  ShadingType,
} from 'docx'
import { saveAs } from 'file-saver'
import {
  IMG_LINE_RE,
  type ImageData,
  preloadAllImages,
  buildImageParagraphs,
} from './exportWordImages'

// ==================== 导出数据类型 ====================

export interface LessonPlanExportData {
  title: string
  subject: string
  grade: string
  topic: string
  duration_minutes: number
  content_markdown: string | null
  author_name?: string
  ai_review_score?: number | null
  created_at?: string
}

// ==================== 行内格式解析 ====================

/** 解析行内粗体 **text** → TextRun 数组 */
function parseInlineRuns(text: string, fontSize = 22): TextRun[] {
  const parts = text.split(/(\*\*[^*]+\*\*)/)
  return parts.map(part => {
    if (part.startsWith('**') && part.endsWith('**')) {
      return new TextRun({ text: part.slice(2, -2), bold: true, size: fontSize })
    }
    return new TextRun({ text: part, size: fontSize })
  })
}

// ==================== Markdown 预处理 ====================

/**
 * 预处理 Markdown:
 * 1. 合并连续空行为单个空行
 * 2. 去掉 > blockquote 前缀,保留内容
 */
function preprocessMarkdown(markdown: string): string[] {
  const raw = markdown.split('\n')
  const result: string[] = []
  let lastWasEmpty = false

  for (const line of raw) {
    let processed = line
    if (/^>\s*/.test(processed)) {
      processed = processed.replace(/^>\s*/, '')
    }
    const isEmpty = processed.trim() === ''
    if (isEmpty) {
      if (!lastWasEmpty) result.push('')
      lastWasEmpty = true
    } else {
      result.push(processed)
      lastWasEmpty = false
    }
  }
  return result
}

// ==================== Markdown → docx Paragraph 数组 ====================

/** Markdown → docx Paragraph 数组(图片走 buildImageParagraphs) */
function parseMarkdownToParagraphs(
  markdown: string,
  imageMap: Map<string, ImageData>
): Paragraph[] {
  if (!markdown || !markdown.trim()) {
    return [new Paragraph({
      children: [new TextRun({ text: '(暂无教案内容)', color: '9CA3AF', size: 22 })],
    })]
  }

  const lines = preprocessMarkdown(markdown)
  const paragraphs: Paragraph[] = []

  for (const line of lines) {
    const t = line.trim()

    // 图片行
    const imgMatch = t.match(IMG_LINE_RE)
    if (imgMatch) {
      const url = imgMatch[2]
      const img = imageMap.get(url)
      if (img) {
        paragraphs.push(...buildImageParagraphs(img))
      } else {
        console.warn(`[exportWord] imageMap 中未找到: ${url}`)
        paragraphs.push(new Paragraph({
          alignment: AlignmentType.CENTER,
          children: [new TextRun({
            text: `[图片未加载:${imgMatch[1] || url}]`,
            size: 20, color: 'DC2626', italics: true,
          })],
          spacing: { before: 60, after: 60 },
        }))
      }
      continue
    }

    // 空行
    if (!t) {
      paragraphs.push(new Paragraph({
        children: [],
        spacing: { before: 60, after: 0, line: 240 },
      }))
      continue
    }

    // 分割线 ---
    if (/^---+$/.test(t)) {
      paragraphs.push(new Paragraph({
        children: [],
        border: { bottom: { style: BorderStyle.SINGLE, size: 4, color: 'CCCCCC' } },
        spacing: { before: 80, after: 80 },
      }))
      continue
    }

    // ### 三级标题
    const h3 = t.match(/^###\s+(.+)/)
    if (h3) {
      paragraphs.push(new Paragraph({
        heading: HeadingLevel.HEADING_3,
        children: parseInlineRuns(h3[1], 22),
        spacing: { before: 100, after: 30, line: 276 },
      }))
      continue
    }

    // ## 二级标题
    const h2 = t.match(/^##\s+(.+)/)
    if (h2) {
      paragraphs.push(new Paragraph({
        heading: HeadingLevel.HEADING_2,
        children: parseInlineRuns(h2[1], 24),
        spacing: { before: 140, after: 40, line: 276 },
      }))
      continue
    }

    // # 一级标题
    const h1 = t.match(/^#\s+(.+)/)
    if (h1) {
      paragraphs.push(new Paragraph({
        heading: HeadingLevel.HEADING_1,
        children: parseInlineRuns(h1[1], 26),
        spacing: { before: 180, after: 60, line: 276 },
      }))
      continue
    }

    // - 无序列表
    const ul = t.match(/^[-*]\s+(.+)/)
    if (ul) {
      paragraphs.push(new Paragraph({
        bullet: { level: 0 },
        children: parseInlineRuns(ul[1], 22),
        spacing: { before: 10, after: 10, line: 260 },
      }))
      continue
    }

    // 1. 有序列表
    const ol = t.match(/^\d+\.\s+(.+)/)
    if (ol) {
      paragraphs.push(new Paragraph({
        numbering: { reference: 'default-numbering', level: 0 },
        children: parseInlineRuns(ol[1], 22),
        spacing: { before: 10, after: 10, line: 260 },
      }))
      continue
    }

    // 普通段落
    paragraphs.push(new Paragraph({
      children: parseInlineRuns(t, 22),
      spacing: { before: 10, after: 10, line: 260 },
    }))
  }

  return paragraphs
}

// ==================== 元信息表格 ====================

/** 构建元信息表格 */
function buildMetaTable(plan: LessonPlanExportData): Table {
  const makeRow = (label: string, value: string): TableRow =>
    new TableRow({
      children: [
        new TableCell({
          width: { size: 18, type: WidthType.PERCENTAGE },
          shading: { type: ShadingType.CLEAR, color: 'F3F4F6', fill: 'F3F4F6' },
          children: [new Paragraph({
            children: [new TextRun({ text: label, bold: true, size: 20, color: '374151' })],
            spacing: { before: 40, after: 40 },
          })],
        }),
        new TableCell({
          width: { size: 82, type: WidthType.PERCENTAGE },
          children: [new Paragraph({
            children: [new TextRun({ text: value, size: 20, color: '1F2937' })],
            spacing: { before: 40, after: 40 },
          })],
        }),
      ],
    })

  const rows: TableRow[] = [
    makeRow('学科', plan.subject),
    makeRow('年级', plan.grade),
    makeRow('课题', plan.topic),
    makeRow('课时', `${plan.duration_minutes} 分钟`),
  ]
  if (plan.author_name) rows.push(makeRow('作者', plan.author_name))
  if (plan.ai_review_score != null) {
    rows.push(makeRow('AI评分', `${plan.ai_review_score.toFixed(1)} 分`))
  }
  if (plan.created_at) {
    try {
      const d = new Date(plan.created_at)
      rows.push(makeRow('日期',
        `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`
      ))
    } catch { /* 忽略日期解析失败 */ }
  }

  return new Table({
    width: { size: 100, type: WidthType.PERCENTAGE },
    rows,
    borders: {
      top:     { style: BorderStyle.SINGLE, size: 4, color: 'E5E7EB' },
      bottom:  { style: BorderStyle.SINGLE, size: 4, color: 'E5E7EB' },
      left:    { style: BorderStyle.SINGLE, size: 4, color: 'E5E7EB' },
      right:   { style: BorderStyle.SINGLE, size: 4, color: 'E5E7EB' },
      insideHorizontal: { style: BorderStyle.SINGLE, size: 4, color: 'E5E7EB' },
      insideVertical:   { style: BorderStyle.SINGLE, size: 4, color: 'E5E7EB' },
    },
  })
}

// ==================== 主导出函数 ====================

export async function exportLessonPlanToWord(plan: LessonPlanExportData): Promise<void> {
  console.log('[exportWord] ========== 开始导出 Word ==========')
  console.log(`[exportWord] 教案: ${plan.title}, 学科: ${plan.subject}, 年级: ${plan.grade}`)

  // 1. 预处理 Markdown 并下载图片
  const markdownText = plan.content_markdown || ''
  const lines = preprocessMarkdown(markdownText)
  const imageMap = await preloadAllImages(lines)
  console.log(`[exportWord] imageMap 大小: ${imageMap.size}`)

  // 2. 解析为 docx 段落
  const contentParagraphs = parseMarkdownToParagraphs(markdownText, imageMap)
  console.log(`[exportWord] 生成段落数: ${contentParagraphs.length}`)

  // 3. 构建 Document
  const doc = new Document({
    numbering: {
      config: [{
        reference: 'default-numbering',
        levels: [{
          level: 0,
          format: 'decimal',
          text: '%1.',
          alignment: AlignmentType.LEFT,
          style: {
            paragraph: { indent: { left: 320, hanging: 220 } },
            run: { size: 22 },
          },
        }],
      }],
    },
    styles: {
      default: {
        document: {
          run: { font: 'Microsoft YaHei', size: 22, color: '1F2937' },
          paragraph: { spacing: { line: 260, before: 0, after: 0 } },
        },
      },
      paragraphStyles: [
        {
          id: 'Heading1', name: 'Heading 1', basedOn: 'Normal', next: 'Normal',
          run: { bold: true, size: 28, color: '1F2937', font: 'Microsoft YaHei' },
          paragraph: { spacing: { before: 180, after: 60, line: 276 } },
        },
        {
          id: 'Heading2', name: 'Heading 2', basedOn: 'Normal', next: 'Normal',
          run: { bold: true, size: 26, color: '1F2937', font: 'Microsoft YaHei' },
          paragraph: { spacing: { before: 140, after: 40, line: 276 } },
        },
        {
          id: 'Heading3', name: 'Heading 3', basedOn: 'Normal', next: 'Normal',
          run: { bold: true, size: 24, color: '374151', font: 'Microsoft YaHei' },
          paragraph: { spacing: { before: 100, after: 30, line: 276 } },
        },
      ],
    },
    sections: [{
      properties: {
        page: { margin: { top: 1134, bottom: 1134, left: 1134, right: 1134 } },
      },
      children: [
        new Paragraph({
          alignment: AlignmentType.CENTER,
          children: [new TextRun({
            text: plan.title, bold: true, size: 36, color: '1F2937', font: 'Microsoft YaHei',
          })],
          spacing: { before: 0, after: 120, line: 276 },
        }),
        buildMetaTable(plan),
        new Paragraph({
          children: [],
          border: { bottom: { style: BorderStyle.SINGLE, size: 6, color: 'E5E7EB' } },
          spacing: { before: 100, after: 100 },
        }),
        new Paragraph({
          children: [new TextRun({
            text: '教案内容', bold: true, size: 26, color: '1F2937', font: 'Microsoft YaHei',
          })],
          spacing: { before: 0, after: 60, line: 276 },
        }),
        ...contentParagraphs,
        new Paragraph({
          alignment: AlignmentType.CENTER,
          children: [],
          border: { top: { style: BorderStyle.SINGLE, size: 4, color: 'E5E7EB' } },
          spacing: { before: 160, after: 60 },
        }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          children: [new TextRun({
            text: '由 TE-DNA 2.0 · 备课工坊 生成',
            size: 18, color: '9CA3AF', font: 'Microsoft YaHei',
          })],
        }),
      ],
    }],
  })

  // 4. 打包下载
  console.log('[exportWord] 开始 Packer.toBlob...')
  const blob = await Packer.toBlob(doc)
  console.log(`[exportWord] Blob 生成完成, 大小=${blob.size} bytes`)

  const safeName = plan.title.replace(/[\\/:*?"<>|]/g, '').replace(/\s+/g, '_').slice(0, 50)
  saveAs(blob, `${safeName}.docx`)
  console.log(`[exportWord] ========== 导出完成: ${safeName}.docx ==========`)
}

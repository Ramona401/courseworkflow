/**
 * textbookOCRReadiness.ts — 课本选择OCR就绪状态的纯函数
 *
 * 只负责判断“当前选中的课本页能否安全提交关联”：
 * - 未选任何页面：视为就绪；
 * - 选中ID不在当前列表：视为缺失，禁止提交；
 * - 选中页面has_ocr=false：视为未识别，禁止提交；
 * - 只有全部选中页面都存在且has_ocr=true时才允许提交。
 *
 * 不发请求、不改状态、不改变后端OCR或课本一致性硬闸。
 */

import type { TextbookListItem } from '@/api/textbooks'

export interface SelectedTextbookOCRReadiness {
  ready: boolean
  missingIds: string[]
  unrecognizedIds: string[]
}

export function getSelectedTextbookOCRReadiness(
  pages: TextbookListItem[],
  selectedIds: string[],
): SelectedTextbookOCRReadiness {
  if (selectedIds.length === 0) {
    return {
      ready: true,
      missingIds: [],
      unrecognizedIds: [],
    }
  }

  const pageByID = new Map(
    pages.map(page => [page.id, page] as const),
  )

  const missingIds: string[] = []
  const unrecognizedIds: string[] = []
  const seen = new Set<string>()

  for (const rawID of selectedIds) {
    const id = rawID.trim()
    if (!id || seen.has(id)) continue
    seen.add(id)

    const page = pageByID.get(id)
    if (!page) {
      missingIds.push(id)
      continue
    }

    if (!page.has_ocr) {
      unrecognizedIds.push(id)
    }
  }

  return {
    ready:
      missingIds.length === 0 &&
      unrecognizedIds.length === 0,
    missingIds,
    unrecognizedIds,
  }
}

/**
 * 课件工坊 API —— 代码收藏库层 (coursewares.snippets.ts)【批次C新增】
 *
 * 单一职责：老师"打星收藏"的课件页HTML代码快照的增删查。
 * 收藏后可在任何课件的单页微调（RefinePanel）中选一条注入为参考代码，
 * 让AI按范本的布局骨架/交互方式/视觉手法微调当前页。
 *
 * 经桶文件 coursewares.ts re-export，对外统一 import 路径：
 *   import { listCodeSnippets, createCodeSnippet, ... } from '@/api/coursewares'
 *
 * 后端对应（handler: courseware_snippet_handler.go，表: courseware_code_snippets）：
 *   GET    /api/v1/code-snippets        — 我的收藏列表（轻量，不含HTML全文，附字节数 html_len）
 *   POST   /api/v1/code-snippets        — 收藏某课件某页（服务端自取页面当前HTML做快照，前端不传大HTML）
 *   GET    /api/v1/code-snippets/{id}   — 单条完整详情（含HTML全文，预览/注入微调时用）
 *   DELETE /api/v1/code-snippets/{id}   — 删除收藏
 *
 * 快照语义：收藏时定格页面HTML，源页之后修改/删除不影响收藏；
 * 溯源字段 source_courseware_id/source_page_number 仅供展示。每用户上限200条（超限后端报错提示清理）。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'

/** 收藏列表项（轻量，不含HTML全文；html_len 为快照字节数，供展示体量） */
export interface CodeSnippetListItem {
  id: string
  title: string
  note: string
  html_len: number
  source_courseware_id: string
  source_page_number: number
  created_at: string
}

/** 收藏完整详情（含HTML全文，注入微调/预览时按 id 单独拉取） */
export interface CodeSnippetDetail {
  id: string
  user_id: string
  title: string
  note: string
  html_content: string
  source_courseware_id: string
  source_page_number: number
  created_at: string
}

/**
 * 我的代码收藏列表（倒序，最新在前）。
 * 后端空列表可能回 null，此处归一为空数组，调用方无需判空。
 */
export async function listCodeSnippets(): Promise<{ snippets: CodeSnippetListItem[]; total: number }> {
  const resp = await apiClient.get('/code-snippets')
  const d = extractData(resp) as { snippets: CodeSnippetListItem[] | null; total: number }
  return { snippets: d.snippets || [], total: d.total || 0 }
}

/**
 * 收藏某课件某页的当前HTML代码。
 * 前端只传定位信息与名称备注，HTML快照由服务端自取（省带宽、保证与库内一致）。
 * 权限：仅课件作者本人可收藏自己课件的页（后端校验）。
 *
 * @param coursewareId 课件ID
 * @param pageNumber   页码（1-based）
 * @param title        收藏名称（必填，最多200字）
 * @param note         可选备注（最多2000字）
 */
export async function createCodeSnippet(
  coursewareId: string,
  pageNumber: number,
  title: string,
  note?: string,
): Promise<{ id: string; title: string; created_at: string; message: string }> {
  const resp = await apiClient.post('/code-snippets', {
    courseware_id: coursewareId,
    page_number: pageNumber,
    title,
    note: note || '',
  })
  return extractData(resp)
}

/** 取单条收藏完整详情（含HTML全文）。只能取自己的收藏（后端校验归属）。 */
export async function getCodeSnippet(snippetId: string): Promise<CodeSnippetDetail> {
  const resp = await apiClient.get(`/code-snippets/${snippetId}`)
  return extractData(resp)
}

/** 删除一条收藏（后端 WHERE id AND user_id 双条件，天然防越权）。 */
export async function deleteCodeSnippet(snippetId: string): Promise<{ message: string }> {
  const resp = await apiClient.delete(`/code-snippets/${snippetId}`)
  return extractData(resp)
}

/**
 * 课件工坊 API —— 背景图库层 (coursewares.bg.ts)（批次2新建）
 *
 * 背景图库：一集 = 头图(封面用) + 内页图 两张OSS公网图。
 * 课件选中某集后，后端把两URL快照写入课件，并对已生成页做"秒换"
 * （纯字符串替换注入的背景<style>块，零AI调用零token，秒级完成）。
 * 经桶文件 coursewares.ts 透出，对外 import 路径不变。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'

// ==================== 类型 ====================

/** 背景图集（系统图库 scope=system / 个人 scope=personal） */
export interface CWBackgroundSet {
  id: string
  name: string
  description: string
  style_category: string
  scope: 'system' | 'personal'
  user_id: string | null
  cover_oss_url: string
  cover_public_url: string
  content_oss_url: string
  content_public_url: string
  status: string
  sort_order: number
  created_at: string | null
  updated_at: string | null
}

/** 课件当前背景选择（两URL快照；空串=未选） */
export interface CWBackgroundSelection {
  cover_bg_url: string
  content_bg_url: string
}

/** 选择/清除背景的执行结果 */
export interface CWBackgroundResult {
  cover_bg_url: string
  content_bg_url: string
  swapped_pages: number  // 被秒换背景的已生成页数
}

// ==================== API ====================

/** 背景图集列表（系统图库 + 我的个人集） */
export async function listBackgroundSets(): Promise<{ sets: CWBackgroundSet[]; total: number }> {
  const resp = await apiClient.get('/courseware-backgrounds')
  return extractData(resp)
}

/** 查询课件当前背景选择 */
export async function getCWBackground(coursewareId: string): Promise<CWBackgroundSelection> {
  const resp = await apiClient.get('/coursewares/' + coursewareId + '/background')
  return extractData(resp)
}

/** 课件选用某背景图集（写快照 + 秒换全部已生成页）。秒换是字符串操作但页多时仍给足超时 */
export async function setCWBackground(coursewareId: string, setId: string): Promise<CWBackgroundResult> {
  const resp = await apiClient.put(
    '/coursewares/' + coursewareId + '/background',
    { set_id: setId },
    { timeout: 30000 },
  )
  return extractData(resp)
}

/** 清除课件背景选择（已生成页回退到模板自带背景或无背景） */
export async function clearCWBackground(coursewareId: string): Promise<CWBackgroundResult> {
  const resp = await apiClient.put(
    '/coursewares/' + coursewareId + '/background',
    { clear: true },
    { timeout: 30000 },
  )
  return extractData(resp)
}

/**
 * 课件工坊 API —— 背景图库层 (coursewares.bg.ts)（批次2新建，批次3扩展生产入口）
 *
 * 背景图库：一集 = 头图(封面用) + 内页图 两张OSS公网图。
 * 课件选中某集后，后端把两URL快照写入课件，并对已生成页做"秒换"
 * （纯字符串替换注入的背景<style>块，零AI调用零token，秒级完成）。
 *
 * 批次3新增：AI生成一套(generateBackgroundSet) / 上传一套(uploadBackgroundSet) /
 * 归档删除(deleteBackgroundSet) / admin升级系统图库(promoteBackgroundSet)。
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

/** 批次3: AI生成一套背景的请求参数 */
export interface GenerateBackgroundSetParams {
  courseware_id?: string   // 非空=生成后自动应用到该课件
  name?: string            // 集名称（空=后端默认"AI背景·MMDD-HHmm"）
  cover_prompt: string     // 封面图提示词
  content_prompt: string   // 内页图提示词（后端会强制追加"浅色低对比、适合做底纹"约束）
}

/** 批次3: 图集生产（AI生成/上传）结果——selection非空表示已自动应用到课件 */
export interface CWBackgroundProduceResult {
  set: CWBackgroundSet
  selection?: CWBackgroundResult
}

// ==================== 选择/清除（批次2） ====================

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

// ==================== 生产入口（批次3） ====================

/**
 * AI生成一套背景（封面+内页两张16:9, 2560×1440）→ 上OSS → 建个人集 → 自动应用到课件。
 * 两张图串行生成+下载+上云，全程可能需1-3分钟，超时给到5分钟。
 */
export async function generateBackgroundSet(params: GenerateBackgroundSetParams): Promise<CWBackgroundProduceResult> {
  const resp = await apiClient.post('/courseware-backgrounds/generate', params, { timeout: 300000 })
  return extractData(resp)
}

/**
 * 上传一套背景（两张图, ≤5MB, JPG/PNG/WEBP）→ 上OSS → 建个人集 → 自动应用到课件。
 * multipart字段: name / courseware_id / cover / content
 */
export async function uploadBackgroundSet(params: {
  name?: string
  coursewareId?: string
  cover: File
  content: File
}): Promise<CWBackgroundProduceResult> {
  const fd = new FormData()
  if (params.name) fd.append('name', params.name)
  if (params.coursewareId) fd.append('courseware_id', params.coursewareId)
  fd.append('cover', params.cover)
  fd.append('content', params.content)
  const resp = await apiClient.post('/courseware-backgrounds/upload', fd, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000,
  })
  return extractData(resp)
}

/** 删除（归档）图集：个人集=本人/admin，系统集=仅admin。已选用该集的课件不受影响（URL快照） */
export async function deleteBackgroundSet(setId: string): Promise<{ id: string }> {
  const resp = await apiClient.delete('/courseware-backgrounds/' + setId)
  return extractData(resp)
}

/** admin专属：把个人图集升级为系统图库（全体用户可见） */
export async function promoteBackgroundSet(setId: string): Promise<CWBackgroundSet> {
  const resp = await apiClient.post('/courseware-backgrounds/' + setId + '/promote', {})
  return extractData(resp)
}

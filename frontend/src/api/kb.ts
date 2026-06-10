/**
 * 知识库课标压缩入库系统 API 封装 (kb.ts)
 *
 * 对应后端：
 *   - 压缩任务：POST/GET /api/v1/kb/jobs、GET /api/v1/kb/jobs/{id}
 *   - 进度SSE：GET /api/v1/sse/kb/{id}?token=（EventSource，token走query参数）
 *   - 审核：GET /api/v1/kb/jobs/{id}/review-queue、POST /api/v1/kb/items/{id}/review
 *   - 入库：POST /api/v1/kb/jobs/{id}/commit
 *   - 蓝绿切换：POST /api/v1/kb/switch-batch
 *   - 白名单管理（仅admin）：GET/POST /api/v1/admin/kb-authorized、DELETE /api/v1/admin/kb-authorized/{user_id}
 *
 * 字段名严格对齐后端真实 json tag（见 models/kb_compress.go 与 services/kb_review_service.go）：
 *   - CreateJob 返回 {job_id, message}（非 id）
 *   - ListJobs 返回 {jobs, total}（非 items）
 *   - GetReviewQueue / ListAuthorized 返回 {items, total}
 *   - CommitBatch 返回 {committed, skipped, errors}
 *   - SwitchBatch 返回 {archived, activated}
 *   - KBDecodedField 为 {label, tag, content}
 *   - KBRoundView.decoded 为指针，出错轮次可能为 null，前端须判空
 *
 * 专利保护边界：审核展示全是字典解码后的「人话」，索引原文绝不出现在前端任何结构里。
 *
 * 封装风格对齐 coursewares.core.ts：apiClient + 自带 extractData；SSE 照 subscribeCWIndexSSE。
 */
import apiClient from './client'

// ==================== 通用提取函数 ====================

/** 统一从 axios 响应里取 {code:0,data} 的 data，异常即抛 */
function extractData<T>(resp: { data?: { code?: number; data?: T } }): T {
  const d = resp?.data
  if (d && d.code === 0 && d.data !== undefined) return d.data
  throw new Error('接口返回异常')
}

// ==================== 常量 ====================

/** 任务类型：本迭代只做课标 */
export const KB_KIND_CURRICULUM = 'curriculum'
export const KB_KIND_TEXTBOOK = 'textbook'

/** 默认压缩轮数（与后端 KBDefaultRounds 对齐） */
export const KB_DEFAULT_ROUNDS = 3

/** 任务状态（与后端 KBJobStatus 七态对齐） */
export const KB_JOB_STATUS_CONFIG: Record<string, { label: string; color: string; bg: string }> = {
  uploaded:    { label: '已上传',   color: '#6B7280', bg: '#F3F4F6' },
  parsing:     { label: '解析中',   color: '#D97706', bg: '#FEF3C7' },
  compressing: { label: '压缩中',   color: '#2563EB', bg: '#DBEAFE' },
  arbitrating: { label: '仲裁中',   color: '#7C3AED', bg: '#EDE9FE' },
  reviewing:   { label: '待审核',   color: '#0891B2', bg: '#CFFAFE' },
  done:        { label: '已完成',   color: '#059669', bg: '#D1FAE5' },
  failed:      { label: '失败',     color: '#DC2626', bg: '#FEE2E2' },
}

/** 单元审核状态（与后端 KBReviewStatus 六态对齐） */
export const KB_REVIEW_STATUS_CONFIG: Record<string, { label: string; color: string; bg: string }> = {
  pending:     { label: '待处理',   color: '#6B7280', bg: '#F3F4F6' },
  auto_passed: { label: '自动通过', color: '#059669', bg: '#D1FAE5' },
  need_review: { label: '待人工',   color: '#D97706', bg: '#FEF3C7' },
  approved:    { label: '已确认',   color: '#2563EB', bg: '#DBEAFE' },
  rejected:    { label: '已退回',   color: '#DC2626', bg: '#FEE2E2' },
  archived:    { label: '已归档',   color: '#9CA3AF', bg: '#F3F4F6' },
}

// ==================== 类型定义 ====================

/** 压缩任务列表项 / 详情（对应后端 KBCompressJob） */
export interface KBCompressJob {
  id: string
  kind: string                 // curriculum / textbook
  batch_tag: string            // 批次标识（蓝绿切换单位）
  source_file: string
  compress_mode: string        // fast / precise
  subject: string
  publisher: string
  grade_num: number
  semester: string
  unit_number: number
  status: string               // uploaded / parsing / compressing / arbitrating / reviewing / done / failed
  total_items: number          // 抽取出的待压缩单元数
  done_items: number           // 已完成数
  created_by: string
  created_at: string
  updated_at: string
}

/** 任务列表响应（后端返回 {jobs, total}） */
export interface KBJobListResponse {
  jobs: KBCompressJob[]
  total: number
}

/**
 * 创建任务请求（对应后端 KBCreateJobRequest）
 * PDF 已砍：只收 text_content（粘贴文本）+ image_data_uris（图片多模态，data URI 数组）。
 * rounds 省略时后端用默认 3。
 */
export interface KBCreateJobRequest {
  kind?: string                // 省略默认 curriculum
  batch_tag: string            // 批次标识（必填，蓝绿切换以此为单位）
  rounds?: number              // 压缩轮数，省略默认 3
  subject?: string             // 课标定位：学科
  grade_num?: number           // 课标定位：年级
  text_content?: string        // 粘贴的课标原文文本
  image_data_uris?: string[]   // 课标原文图片（data URI，多模态识别）
}

/** 创建任务响应（后端返回 {job_id, message}） */
export interface KBCreateJobResponse {
  job_id: string
  message?: string
}

// ---------- 审核解码展示结构（专利保护核心：以下结构均为字典解码后的「人话」，不含索引原文） ----------

/**
 * 解码后的单个语义字段（对应后端 KBDecodedField）
 * 后端真实字段：label（中文标签名）/ tag（原标签字母，调试用，前端可不显示）/ content（字段内容）
 */
export interface KBDecodedField {
  label: string      // 中文标签名（如「学业要求」）
  tag: string        // 原标签字母（如「E」，仅供调试）
  content: string    // 字段内容文本
}

/** 解码后的一行索引（对应后端 KBDecodedIndex；无索引原文） */
export interface KBDecodedIndex {
  kp_code: string                 // 知识点编码（标识，非机密符号，可展示）
  subject_name: string            // 学科中文
  stage_name: string              // 学段中文
  grade_name: string              // 年级中文
  depth_name: string              // 深度档中文
  fields: KBDecodedField[]        // 语义字段人话列表
  decode_failed: boolean          // 解码失败标记（fail-safe，true 时该轮可能含降级内容）
}

/**
 * 审核界面里的「一轮草稿」视图（对应后端 KBRoundView）
 * 注意：后端 decoded 为指针，出错轮次（error 非空）该字段可能为 null，前端务必判空。
 */
export interface KBRoundView {
  round: number                       // 第几轮
  model: string                       // 该轮使用的模型
  decoded?: KBDecodedIndex | null     // 解码后的人话（出错轮次为 null）
  error?: string                      // 该轮失败原因（成功为空）
}

/** 审核队列里的「一个待审单元」视图（对应后端 KBReviewItemView） */
export interface KBReviewItemView {
  item_id: string                 // 单元ID（审核动作按此ID提交）
  seq: number                     // 单元序号
  confidence: string              // high / low
  review_status: string           // pending / auto_passed / need_review / approved / rejected / archived
  source_excerpt: string          // 原文片段（供审核员对照判断，非索引）
  page_label: string              // 页码标签
  conflicts: string[]             // 仲裁识别出的冲突点（中文描述，供高亮）
  chosen_round: number            // 仲裁选中的轮次（confirm 时默认采纳这一轮）
  rounds: KBRoundView[]           // 多轮解码后的人话并排
}

/** 审核队列响应（后端返回 {items, total}） */
export interface KBReviewQueueResponse {
  items: KBReviewItemView[]
  total: number
}

/**
 * 审核动作请求（对应后端 KBReviewActionRequest）
 * action 三选一：
 *   - confirm：采纳仲裁选中的轮次（chosen_round 由后端按仲裁结论取，可不传）
 *   - select ：采纳指定轮次（必须传 chosen_round）
 *   - reject ：退回重压
 */
export interface KBReviewActionRequest {
  action: 'confirm' | 'select' | 'reject'
  chosen_round?: number          // action=select 时必填；confirm 时可省（后端用仲裁选中轮）
  review_note?: string           // 审核意见（专利「校准锚点」留痕）
}

/**
 * commit 入库结果（对应后端 CommitBatchResult）
 * 重要：skipped 的条目必须明确回显给审核员（单条解码缺字段会被 skip 并记进 errors），
 * 否则被跳过的知识点会悄悄消失。前端不能只显示「成功 N 条」。
 */
export interface KBCommitBatchResult {
  committed: number              // 成功写入目标表的条数
  skipped: number                // 被跳过的条数
  errors: string[]               // 跳过原因明细（逐条显示给审核员）
}

/** 蓝绿切换结果（对应后端 SwitchBatchResult） */
export interface KBSwitchBatchResult {
  archived: number               // 归档的旧 active 条数
  activated: number              // 激活的新批条数
}

/** 白名单成员项（对应后端 KBAuthorizedUserItem） */
export interface KBAuthorizedUserItem {
  user_id: string
  username: string
  display_name: string
  role: string                   // 被授权人的系统角色（仅展示，不影响授权——白名单不绑角色）
  granted_by: string             // 授权人显示名（后端 COALESCE 兜底为显示名）
  note: string                   // 备注
  created_at: string
}

/** 白名单列表响应 */
export interface KBAuthorizedListResponse {
  items: KBAuthorizedUserItem[]
  total: number
}

/** 新增白名单成员请求（对应后端 KBAddAuthorizedRequest） */
export interface KBAddAuthorizedRequest {
  user_id: string
  note?: string
}

// ---------- SSE 进度回调 ----------

/**
 * KB 压缩进度 SSE 回调（事件名与后端 kb_sse_hub 常量对齐）：
 *   connected / extract_start / extract_done / item_start / item_done / progress / job_done / error
 */
export interface KBSSECallbacks {
  onConnected?: (data: Record<string, unknown>) => void
  onExtractStart?: (data: Record<string, unknown>) => void
  onExtractDone?: (data: { total_items?: number; message?: string }) => void
  onItemStart?: (data: { seq?: number; total?: number; message?: string }) => void
  onItemDone?: (data: { seq?: number; total?: number; confidence?: string; review_status?: string; message?: string }) => void
  onProgress?: (data: { done_items?: number; total_items?: number; message?: string }) => void
  onJobDone?: (data: { job_id?: string; total_items?: number; auto_passed?: number; need_review?: number; message?: string }) => void
  onError?: (data: { message?: string }) => void
}

// ==================== 压缩任务 API ====================

/**
 * 建任务（后端建库后异步触发压缩流水线；前端随后用 SSE 订阅进度）。
 * 注意：建任务成功立刻返回 job_id，后端 go func + 800ms 延迟才真正跑压缩，
 * 以确保前端来得及先连上 SSE。所以调用方应在拿到 job_id 后立即订阅 SSE。
 */
export async function createKBJob(req: KBCreateJobRequest): Promise<KBCreateJobResponse> {
  const resp = await apiClient.post('/kb/jobs', req)
  return extractData(resp)
}

/** 任务列表（默认 kind=curriculum；可按 batch_tag 过滤） */
export async function listKBJobs(params?: {
  kind?: string
  batch_tag?: string
}): Promise<KBJobListResponse> {
  const resp = await apiClient.get('/kb/jobs', { params })
  return extractData(resp)
}

/** 任务详情 */
export async function getKBJob(id: string): Promise<KBCompressJob> {
  const resp = await apiClient.get(`/kb/jobs/${id}`)
  return extractData(resp)
}

/**
 * 订阅某任务的压缩进度 SSE。
 * EventSource 无法设置请求头，故 token 走 query 参数（?token=），与 subscribeCWIndexSSE 一致。
 * 返回 { close }，组件卸载时务必调用以关闭连接。
 */
export function subscribeKBJobSSE(
  jobId: string,
  callbacks: KBSSECallbacks,
): { close: () => void } {
  const token = localStorage.getItem('token') || ''
  const url = `${window.location.origin}/api/v1/sse/kb/${jobId}?token=${encodeURIComponent(token)}`

  const evtSource = new EventSource(url)

  evtSource.addEventListener('connected', (e: MessageEvent) => {
    try { callbacks.onConnected?.(JSON.parse(e.data)) } catch { /* 忽略解析失败 */ }
  })
  evtSource.addEventListener('extract_start', (e: MessageEvent) => {
    try { callbacks.onExtractStart?.(JSON.parse(e.data)) } catch { /* 忽略解析失败 */ }
  })
  evtSource.addEventListener('extract_done', (e: MessageEvent) => {
    try { callbacks.onExtractDone?.(JSON.parse(e.data)) } catch { /* 忽略解析失败 */ }
  })
  evtSource.addEventListener('item_start', (e: MessageEvent) => {
    try { callbacks.onItemStart?.(JSON.parse(e.data)) } catch { /* 忽略解析失败 */ }
  })
  evtSource.addEventListener('item_done', (e: MessageEvent) => {
    try { callbacks.onItemDone?.(JSON.parse(e.data)) } catch { /* 忽略解析失败 */ }
  })
  evtSource.addEventListener('progress', (e: MessageEvent) => {
    try { callbacks.onProgress?.(JSON.parse(e.data)) } catch { /* 忽略解析失败 */ }
  })
  evtSource.addEventListener('job_done', (e: MessageEvent) => {
    try { callbacks.onJobDone?.(JSON.parse(e.data)) } catch { /* 忽略解析失败 */ }
    evtSource.close()
  })
  evtSource.addEventListener('error', (e: MessageEvent) => {
    if (e.data) {
      try { callbacks.onError?.(JSON.parse(e.data)) } catch { /* 忽略解析失败 */ }
    }
    evtSource.close()
  })
  evtSource.onerror = () => {
    evtSource.close()
  }

  return { close: () => evtSource.close() }
}

// ==================== 审核 API ====================

/** 获取某任务的审核队列（解码后的人话；后端只返回需要人工看的与可抽查的单元） */
export async function getKBReviewQueue(jobId: string): Promise<KBReviewQueueResponse> {
  const resp = await apiClient.get(`/kb/jobs/${jobId}/review-queue`)
  return extractData(resp)
}

/**
 * 对单个单元执行审核动作（确认/选版/退回三选一）。
 * itemId 为 KBReviewItemView.item_id。
 */
export async function reviewKBItem(itemId: string, req: KBReviewActionRequest): Promise<void> {
  await apiClient.post(`/kb/items/${itemId}/review`, req)
}

// ==================== 入库 + 蓝绿切换 API ====================

/**
 * commit 入库（把已确认 + 自动通过的单元灌入 curriculum_standards，置 status='staged' 候选态）。
 * 返回 {committed, skipped, errors}——调用方必须把 skipped/errors 明确显示给审核员。
 */
export async function commitKBBatch(jobId: string, batchTag: string): Promise<KBCommitBatchResult> {
  const resp = await apiClient.post(`/kb/jobs/${jobId}/commit`, { batch_tag: batchTag })
  return extractData(resp)
}

/**
 * 蓝绿切换：把指定 batch_tag 整批转 active、旧批转 archived。
 * 后端会先核对该批已有候选数据（>0）才切，防止切到空批。
 * 返回 {archived, activated}。
 */
export async function switchKBBatch(batchTag: string): Promise<KBSwitchBatchResult> {
  const resp = await apiClient.post('/kb/switch-batch', { batch_tag: batchTag })
  return extractData(resp)
}

// ==================== 白名单管理 API（仅 admin，挂 /admin 下） ====================

/** 列出知识库访问白名单 */
export async function listKBAuthorized(): Promise<KBAuthorizedListResponse> {
  const resp = await apiClient.get('/admin/kb-authorized')
  return extractData(resp)
}

/** 新增白名单成员（user_id + 可选备注） */
export async function addKBAuthorized(req: KBAddAuthorizedRequest): Promise<void> {
  await apiClient.post('/admin/kb-authorized', req)
}

/** 移除白名单成员 */
export async function removeKBAuthorized(userId: string): Promise<void> {
  await apiClient.delete(`/admin/kb-authorized/${userId}`)
}

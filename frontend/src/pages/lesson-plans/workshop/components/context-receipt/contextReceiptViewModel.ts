/**
 * contextReceiptViewModel.ts — 备课上下文回执的教师视图转换
 *
 * 职责：
 * 1. 将后端确定性生成的ContextReceipt转换为教师能理解的中文摘要；
 * 2. 区分“本轮已使用”与“未使用或未生效”；
 * 3. 不展示提示词正文、内部ID、候选数量、质量分和system prompt长度；
 * 4. 只解释哪些教学依据正在影响本轮备课，以及未使用材料的真实原因。
 */

import type {
  AssistantContextReceipt,
  ComponentContextReceiptItem,
  ComponentsContextReceipt,
  ContextReceipt,
  ContextReceiptStatus,
  MaterialContextReceipt,
} from '@/api/lesson-plans'

export type ContextReceiptTone = 'positive' | 'neutral' | 'warning'

export interface ContextReceiptViewItem {
  key: string
  title: string
  statusLabel: string
  detail: string
  shortLabel: string
  used: boolean
  tone: ContextReceiptTone
}

export interface ContextReceiptViewModel {
  summary: string
  usedItems: ContextReceiptViewItem[]
  unusedItems: ContextReceiptViewItem[]
}

const STAGE_NAMES: Record<string, string> = {
  analyze: '教学分析',
  design: '教学设计',
  write: '教案撰写',
  review: 'AI评审',
  revise: '修订定稿',
}

const STATUS_LABELS: Record<ContextReceiptStatus, string> = {
  loaded: '本轮已使用',
  not_linked: '未关联',
  not_applicable: '本阶段不使用',
  deferred: '稍后读取',
  superseded: '已让位',
  unavailable: '本轮未能读取',
  forbidden: '本轮未读取',
  explicit_none: '已明确不使用',
  not_found: '未匹配到',
}

const ASSISTANT_SELECTION_LABELS: Record<string, string> = {
  manual: '本轮手动选择',
  preference: '学科偏好',
  auto: '系统自动匹配',
  explicit_none: '老师明确选择系统默认',
}

const RECIPE_SELECTION_LABELS: Record<string, string> = {
  auto: '平台自动选择',
  selected: '老师明确选择',
  none: '老师明确不使用',
}

const COMPONENT_SELECTION_LABELS: Record<string, string> = {
  manual: '老师手动选择',
  recipe: '备课配方带入',
  auto: '系统自动匹配',
  reranked: '结合本轮需求匹配',
}

const ASSISTANT_SOURCE_LABELS: Record<string, string> = {
  personal: '个人助手',
  school: '学校助手',
  group: '教研组助手',
  region: '区域助手',
  system: '系统助手',
}

function isLoaded(status?: string): boolean {
  return status === 'loaded'
}

function normalizeStatus(status?: string): ContextReceiptStatus {
  const known: ContextReceiptStatus[] = [
    'loaded',
    'not_linked',
    'not_applicable',
    'deferred',
    'superseded',
    'unavailable',
    'forbidden',
    'explicit_none',
    'not_found',
  ]
  return known.includes(status as ContextReceiptStatus)
    ? status as ContextReceiptStatus
    : 'unavailable'
}

function toneForStatus(status: ContextReceiptStatus): ContextReceiptTone {
  if (status === 'loaded') return 'positive'
  if (
    status === 'unavailable' ||
    status === 'forbidden' ||
    status === 'not_found'
  ) {
    return 'warning'
  }
  return 'neutral'
}

function joinNonEmpty(parts: Array<string | undefined | null>): string {
  return parts
    .map(part => String(part || '').trim())
    .filter(Boolean)
    .join(' · ')
}

function assistantItem(
  assistant?: AssistantContextReceipt,
): ContextReceiptViewItem {
  const status = normalizeStatus(assistant?.status)
  const selection = assistant?.selection_mode
    ? ASSISTANT_SELECTION_LABELS[assistant.selection_mode] ||
      assistant.selection_mode
    : ''
  const source = assistant?.source
    ? ASSISTANT_SOURCE_LABELS[assistant.source] || assistant.source
    : ''

  const name = assistant?.name?.trim()
  const detail = isLoaded(status)
    ? joinNonEmpty([
        name ? `使用「${name}」` : '已使用匹配到的AI助手',
        selection,
        source,
      ])
    : assistant?.reason || '本轮使用系统阶段要求，不叠加额外助手'

  return {
    key: 'assistant',
    title: 'AI助手',
    statusLabel: STATUS_LABELS[status],
    detail,
    shortLabel: name ? `「${name}」助手` : 'AI助手',
    used: isLoaded(status),
    tone: toneForStatus(status),
  }
}

function recipeItem(
  material?: MaterialContextReceipt,
): ContextReceiptViewItem {
  const status = normalizeStatus(material?.status)
  const mode = material?.selection_mode || ''
  const selectionLabel = mode
    ? RECIPE_SELECTION_LABELS[mode] || mode
    : ''

  const name = material?.name?.trim()

  const loadedDetail = joinNonEmpty([
    name
      ? `使用「${name}」中的教学结构、教研要求和相关配置`
      : '使用已关联备课配方',
    selectionLabel,
    material?.reason,
  ])

  const statusLabel = isLoaded(status)
    ? mode === 'auto'
      ? '平台自动选择'
      : mode === 'selected'
        ? '老师明确选择'
        : STATUS_LABELS[status]
    : status === 'not_found' && mode === 'auto'
      ? '自动匹配未命中'
      : STATUS_LABELS[status]

  const detail = isLoaded(status)
    ? loadedDetail
    : material?.reason || '备课配方本轮未生效'

  const shortLabel = isLoaded(status)
    ? name
      ? mode === 'auto'
        ? `自动匹配「${name}」配方`
        : `「${name}」配方`
      : '备课配方'
    : '备课配方'

  return {
    key: 'recipe',
    title: '备课配方',
    statusLabel,
    detail,
    shortLabel,
    used: isLoaded(status),
    tone: toneForStatus(status),
  }
}

function materialItem(
  key: string,
  title: string,
  material: MaterialContextReceipt | undefined,
  loadedDetail: (value: MaterialContextReceipt) => string,
  shortLabel: (value: MaterialContextReceipt) => string,
): ContextReceiptViewItem {
  const status = normalizeStatus(material?.status)
  const detail = material && isLoaded(status)
    ? loadedDetail(material)
    : material?.reason || `${title}本轮未生效`

  return {
    key,
    title,
    statusLabel: STATUS_LABELS[status],
    detail,
    shortLabel: material ? shortLabel(material) : title,
    used: isLoaded(status),
    tone: toneForStatus(status),
  }
}

function componentLabels(
  items: ComponentContextReceiptItem[] | undefined,
): string {
  if (!items || items.length === 0) return ''
  const labels = items
    .map(item => item.display_label?.trim())
    .filter(Boolean)

  if (labels.length === 0) return ''
  if (labels.length <= 3) return labels.join('、')
  return `${labels.slice(0, 3).join('、')}等`
}

function componentsItem(
  components?: ComponentsContextReceipt,
): ContextReceiptViewItem {
  const status = normalizeStatus(components?.status)
  const count = components?.items?.length || 0
  const mode = components?.selection_mode
    ? COMPONENT_SELECTION_LABELS[components.selection_mode] ||
      components.selection_mode
    : ''
  const labels = componentLabels(components?.items)

  const detail = isLoaded(status)
    ? joinNonEmpty([
        `使用${count}个专业组件`,
        mode,
        labels,
      ])
    : components?.reason || '本轮没有使用专业组件'

  return {
    key: 'components',
    title: '专业组件',
    statusLabel: STATUS_LABELS[status],
    detail,
    shortLabel: count > 0 ? `${count}个专业组件` : '专业组件',
    used: isLoaded(status),
    tone: toneForStatus(status),
  }
}

function textbookDetail(material: MaterialContextReceipt): string {
  const total = material.count || 0
  const readable = material.readable_count || 0
  const unreadable = material.unreadable_count || 0
  const titleText = material.titles && material.titles.length > 0
    ? material.titles.join('、')
    : ''

  return joinNonEmpty([
    total > 0 ? `读取${total}页课本` : '读取已关联课本',
    total > 0 ? `${readable}页有可读取文字` : '',
    unreadable > 0 ? `${unreadable}页尚未识别文字` : '',
    titleText,
  ])
}

function courseOutlineDetail(material: MaterialContextReceipt): string {
  const titles = material.titles?.filter(Boolean) || []
  if (titles.length > 0) {
    return `使用${material.count || titles.length}份课程大纲：${titles.join('、')}`
  }
  return material.count
    ? `使用${material.count}份课程大纲`
    : '已使用课程大纲'
}

function refMaterialDetail(material: MaterialContextReceipt): string {
  if (material.character_count) {
    return `读取老师本轮上传的参考资料，约${material.character_count}字`
  }
  return '读取老师本轮上传的参考资料'
}

export function buildContextReceiptView(
  receipt: ContextReceipt,
): ContextReceiptViewModel {
  const stageName = STAGE_NAMES[receipt.stage_code] || receipt.stage_code || '当前'
  const stageItem: ContextReceiptViewItem = {
    key: 'stage',
    title: '当前阶段',
    statusLabel: '本轮已使用',
    detail: `按照「${stageName}」阶段的任务要求和流程开展本轮备课`,
    shortLabel: `${stageName}阶段要求`,
    used: true,
    tone: 'positive',
  }

  const items: ContextReceiptViewItem[] = [
    stageItem,
    assistantItem(receipt.assistant),
    recipeItem(receipt.recipe),
    componentsItem(receipt.components),
    materialItem(
      'textbook',
      '课本原文',
      receipt.textbook,
      textbookDetail,
      material => material.count ? `${material.count}页课本` : '课本原文',
    ),
    materialItem(
      'unit_plan',
      '单元方案',
      receipt.unit_plan,
      material => material.name
        ? `使用「${material.name}」的单元整体设计`
        : '使用已关联单元方案',
      material => material.name ? `「${material.name}」单元方案` : '单元方案',
    ),
    materialItem(
      'course_outline',
      '课程大纲',
      receipt.course_outline,
      courseOutlineDetail,
      material => material.count ? `${material.count}份课程大纲` : '课程大纲',
    ),
    materialItem(
      'class_profile',
      '班级学情',
      receipt.class_profile,
      material => material.name
        ? `依据「${material.name}」中的班级整体学情进行差异化设计`
        : '依据已关联班级学情进行差异化设计',
      material => material.name ? `「${material.name}」班级学情` : '班级学情',
    ),
    materialItem(
      'ref_material',
      '参考资料',
      receipt.ref_material,
      refMaterialDetail,
      () => '本轮参考资料',
    ),
  ]

  const usedItems = items.filter(item => item.used)
  const unusedItems = items.filter(item => !item.used)
  const summaryParts = usedItems.map(item => item.shortLabel)

  return {
    summary: `本轮备课依据：${summaryParts.join('、')}`,
    usedItems,
    unusedItems,
  }
}

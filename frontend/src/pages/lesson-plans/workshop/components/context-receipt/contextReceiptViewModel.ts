/**
 * contextReceiptViewModel.ts — 备课上下文回执的教师视图转换
 *
 * 后端回执继续完整保存所有状态，供审计、恢复和问题排查。
 * 本文件只控制老师日常界面看到的内容：
 *
 * 1. 展示本轮真正加载成功的教学资源；
 * 2. 隐藏未关联、不适用、稍后读取、已让位和明确不使用等普通状态；
 * 3. unavailable / forbidden / not_found 作为真实失败警告展示；
 * 4. 不展示内部提示词正文、内部ID、候选数量和system prompt长度；
 * 5. 当前阶段骨架属于系统基础能力，不单独作为资源回执展示。
 */

import type {
  AssistantContextReceipt,
  ComponentContextReceiptItem,
  ComponentsContextReceipt,
  ContextReceipt,
  ContextReceiptStatus,
  MaterialContextReceipt,
} from '@/api/lesson-plans'

export type ContextReceiptTone =
  | 'positive'
  | 'neutral'
  | 'warning'

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
  warningItems: ContextReceiptViewItem[]
  /**
   * 只包含教师界面可见信息的稳定签名。
   * 用于判断相邻AI回复的实际资源是否发生变化。
   */
  signature: string
}

const STATUS_LABELS: Record<
  ContextReceiptStatus,
  string
> = {
  loaded: '本轮已读取',
  not_linked: '未关联',
  not_applicable: '本阶段不使用',
  deferred: '稍后读取',
  superseded: '已让位',
  unavailable: '未能读取',
  forbidden: '无权读取',
  explicit_none: '已明确不使用',
  not_found: '未找到',
}

const ASSISTANT_SELECTION_LABELS:
  Record<string, string> = {
    manual: '老师本轮选择',
    preference: '老师保存的偏好',
    auto: '平台严格自动匹配',
    explicit_none: '老师明确选择系统默认',
  }

const RECIPE_SELECTION_LABELS:
  Record<string, string> = {
    auto: '平台严格自动选择',
    selected: '老师明确选择',
    none: '老师明确不使用',
  }

const COMPONENT_SELECTION_LABELS:
  Record<string, string> = {
    manual: '老师手动选择',
    recipe: '备课配方带入',
    auto: '系统自动匹配',
    reranked: '结合本轮需求匹配',
  }

const ASSISTANT_SOURCE_LABELS:
  Record<string, string> = {
    personal: '个人助手',
    school: '学校助手',
    group: '教研组助手',
    region: '区域助手',
    system: '系统助手',
  }

const WARNING_STATUSES =
  new Set<ContextReceiptStatus>([
    'unavailable',
    'forbidden',
    'not_found',
  ])

function normalizeStatus(
  status?: string,
): ContextReceiptStatus {
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

  return known.includes(
    status as ContextReceiptStatus,
  )
    ? status as ContextReceiptStatus
    : 'unavailable'
}

function isLoaded(status?: string): boolean {
  return status === 'loaded'
}

function isWarning(
  status: ContextReceiptStatus,
): boolean {
  return WARNING_STATUSES.has(status)
}

function toneForStatus(
  status: ContextReceiptStatus,
): ContextReceiptTone {
  if (status === 'loaded') return 'positive'
  if (isWarning(status)) return 'warning'
  return 'neutral'
}

function joinNonEmpty(
  parts: Array<string | undefined | null>,
): string {
  return parts
    .map(part => String(part || '').trim())
    .filter(Boolean)
    .join(' · ')
}

function assistantItem(
  assistant?: AssistantContextReceipt,
): ContextReceiptViewItem {
  const status = normalizeStatus(
    assistant?.status,
  )

  const selection = assistant?.selection_mode
    ? ASSISTANT_SELECTION_LABELS[
        assistant.selection_mode
      ] || assistant.selection_mode
    : ''

  const source = assistant?.source
    ? ASSISTANT_SOURCE_LABELS[
        assistant.source
      ] || assistant.source
    : ''

  const name = assistant?.name?.trim()

  const detail = isLoaded(status)
    ? joinNonEmpty([
        name
          ? `使用「${name}」`
          : '已使用AI助手',
        selection,
        source,
      ])
    : assistant?.reason ||
      '本轮未能读取所选AI助手'

  return {
    key: 'assistant',
    title: 'AI助手',
    statusLabel: STATUS_LABELS[status],
    detail,
    shortLabel: name
      ? `助手「${name}」`
      : 'AI助手',
    used: isLoaded(status),
    tone: toneForStatus(status),
  }
}

function recipeItem(
  material?: MaterialContextReceipt,
): ContextReceiptViewItem {
  const status = normalizeStatus(
    material?.status,
  )
  const mode = material?.selection_mode || ''

  const selectionLabel = mode
    ? RECIPE_SELECTION_LABELS[mode] || mode
    : ''

  const name = material?.name?.trim()

  const detail = isLoaded(status)
    ? joinNonEmpty([
        name
          ? `使用「${name}」中的教学结构、流程、组件和教研要求`
          : '使用已关联备课配方',
        selectionLabel,
      ])
    : material?.reason ||
      '本轮未能读取所选备课配方'

  return {
    key: 'recipe',
    title: '备课配方',
    statusLabel: STATUS_LABELS[status],
    detail,
    shortLabel: name
      ? `配方「${name}」`
      : '备课配方',
    used: isLoaded(status),
    tone: toneForStatus(status),
  }
}

function materialItem(
  key: string,
  title: string,
  material: MaterialContextReceipt | undefined,
  loadedDetail: (
    value: MaterialContextReceipt,
  ) => string,
  shortLabel: (
    value: MaterialContextReceipt,
  ) => string,
): ContextReceiptViewItem {
  const status = normalizeStatus(
    material?.status,
  )

  const detail =
    material && isLoaded(status)
      ? loadedDetail(material)
      : material?.reason ||
        `本轮未能读取${title}`

  return {
    key,
    title,
    statusLabel: STATUS_LABELS[status],
    detail,
    shortLabel: material
      ? shortLabel(material)
      : title,
    used: isLoaded(status),
    tone: toneForStatus(status),
  }
}

function componentLabels(
  items:
    | ComponentContextReceiptItem[]
    | undefined,
): string {
  if (!items || items.length === 0) return ''

  const labels = items
    .map(item => item.display_label?.trim())
    .filter(Boolean)

  if (labels.length === 0) return ''
  if (labels.length <= 3) {
    return labels.join('、')
  }

  return `${labels.slice(0, 3).join('、')}等`
}

function componentsItem(
  components?: ComponentsContextReceipt,
): ContextReceiptViewItem {
  const status = normalizeStatus(
    components?.status,
  )
  const count = components?.items?.length || 0

  const mode = components?.selection_mode
    ? COMPONENT_SELECTION_LABELS[
        components.selection_mode
      ] || components.selection_mode
    : ''

  const labels = componentLabels(
    components?.items,
  )

  const detail = isLoaded(status)
    ? joinNonEmpty([
        `使用${count}个专业组件`,
        mode,
        labels,
      ])
    : components?.reason ||
      '本轮未能读取所选专业组件'

  return {
    key: 'components',
    title: '专业组件',
    statusLabel: STATUS_LABELS[status],
    detail,
    shortLabel:
      count > 0
        ? `${count}个专业组件`
        : '专业组件',
    used: isLoaded(status),
    tone: toneForStatus(status),
  }
}

function textbookDetail(
  material: MaterialContextReceipt,
): string {
  const total = material.count || 0
  const readable =
    material.readable_count || 0
  const unreadable =
    material.unreadable_count || 0

  const titleText =
    material.titles &&
    material.titles.length > 0
      ? material.titles.join('、')
      : ''

  return joinNonEmpty([
    total > 0
      ? `读取${total}页课本`
      : '读取已关联课本',
    readable > 0
      ? `${readable}页有可读取文字`
      : '',
    unreadable > 0
      ? `${unreadable}页尚未识别文字`
      : '',
    titleText,
  ])
}

function courseOutlineDetail(
  material: MaterialContextReceipt,
): string {
  const titles =
    material.titles?.filter(Boolean) || []

  if (titles.length > 0) {
    return `使用${
      material.count || titles.length
    }份课程大纲：${titles.join('、')}`
  }

  return material.count
    ? `使用${material.count}份课程大纲`
    : '已使用课程大纲'
}

function refMaterialDetail(
  material: MaterialContextReceipt,
): string {
  if (material.character_count) {
    return `读取老师本轮上传的参考资料，约${material.character_count}字`
  }

  return '读取老师本轮上传的参考资料'
}

function buildVisibleSignature(
  usedItems: ContextReceiptViewItem[],
  warningItems: ContextReceiptViewItem[],
): string {
  return [...usedItems, ...warningItems]
    .map(item =>
      [
        item.key,
        item.statusLabel,
        item.shortLabel,
        item.detail,
      ].join('|'),
    )
    .join('||')
}

export function buildContextReceiptView(
  receipt: ContextReceipt,
): ContextReceiptViewModel {
  // 不把“当前阶段”作为可见资源条目。
  // 阶段骨架每轮都会生效，但它属于系统基础能力，
  // 单独展示会导致无额外资源时也重复出现回执。
  const items: ContextReceiptViewItem[] = [
    assistantItem(receipt.assistant),
    recipeItem(receipt.recipe),
    componentsItem(receipt.components),
    materialItem(
      'textbook',
      '课本原文',
      receipt.textbook,
      textbookDetail,
      material =>
        material.count
          ? `${material.count}页课本`
          : '课本原文',
    ),
    materialItem(
      'unit_plan',
      '单元方案',
      receipt.unit_plan,
      material =>
        material.name
          ? `使用「${material.name}」的单元整体设计`
          : '使用已关联单元方案',
      material =>
        material.name
          ? `单元方案「${material.name}」`
          : '单元方案',
    ),
    materialItem(
      'course_outline',
      '课程大纲',
      receipt.course_outline,
      courseOutlineDetail,
      material =>
        material.count
          ? `${material.count}份课程大纲`
          : '课程大纲',
    ),
    materialItem(
      'class_profile',
      '班级学情',
      receipt.class_profile,
      material =>
        material.name
          ? `依据「${material.name}」中的班级整体学情进行差异化设计`
          : '依据已关联班级学情进行差异化设计',
      material =>
        material.name
          ? `班级学情「${material.name}」`
          : '班级学情',
    ),
    materialItem(
      'ref_material',
      '参考资料',
      receipt.ref_material,
      refMaterialDetail,
      () => '本轮参考资料',
    ),
  ]

  const usedItems = items.filter(
    item => item.used,
  )

  const warningItems = items.filter(
    item => {
      const status = normalizeStatus(
        (
          item.key === 'assistant'
            ? receipt.assistant
            : item.key === 'recipe'
              ? receipt.recipe
              : item.key === 'components'
                ? receipt.components
                : item.key === 'textbook'
                  ? receipt.textbook
                  : item.key === 'unit_plan'
                    ? receipt.unit_plan
                    : item.key === 'course_outline'
                      ? receipt.course_outline
                      : item.key === 'class_profile'
                        ? receipt.class_profile
                        : receipt.ref_material
        )?.status,
      )

      return isWarning(status)
    },
  )

  const summaryParts = usedItems.map(
    item => item.shortLabel,
  )

  const summary =
    summaryParts.length > 0
      ? `本轮已读取：${summaryParts.join('、')}`
      : warningItems.length > 0
        ? '本轮有教学资源未能读取'
        : ''

  return {
    summary,
    usedItems,
    warningItems,
    signature: buildVisibleSignature(
      usedItems,
      warningItems,
    ),
  }
}

/**
 * 返回教师可见回执的稳定签名。
 * 没有加载项且没有警告时返回空字符串。
 */
export function buildContextReceiptSignature(
  receipt: ContextReceipt,
): string {
  return buildContextReceiptView(
    receipt,
  ).signature
}

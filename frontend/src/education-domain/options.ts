/**
 * education-domain/options.ts — 各教育域学习层级统一定义
 *
 * 本文件是前端学习层级的单一真相源，统一服务于：
 *   - 教案起步页；
 *   - 配方创建与编辑；
 *   - AI助手创建与编辑；
 *   - AI助手设计画布；
 *   - 层级标签展示。
 *
 * 数据库存储值与页面展示值分离：
 *
 *   页面显示“职一”，实际保存“中职Ⅰ年级”；
 *   页面显示“职二”，实际保存“中职Ⅱ年级”；
 *   页面显示“职三”，实际保存“中职Ⅲ年级”。
 *
 * 这样既符合职业学校老师的日常表达，也保持数据库规范值稳定，
 * 无需迁移现有数据。
 *
 * 自动匹配规则：
 *   - K12：一年级至高三；
 *   - 职业教育：中职Ⅰ、Ⅱ、Ⅲ年级；
 *   - 成人教育：入门、进阶、高级、管理者。
 *
 * “不限年级”“不限层级”和K12学段只供AI助手手动选择，
 * 不进入配方或助手的自动严格匹配。
 */

import type {
  EducationDomain,
} from './types'

/** 教案、配方使用的具体学习层级选项。 */
export interface EducationLevelOption {
  value: string
  label: string
}

/**
 * AI助手使用的层级选项。
 *
 * automatic=true：
 *   该值属于具体层级，可以参与平台自动严格匹配。
 *
 * automatic=false：
 *   该值属于学段或不限值，只供老师手动选择助手。
 */
export interface AssistantLevelOption
  extends EducationLevelOption {
  automatic: boolean
}

/* ==================== 具体层级 ==================== */

const K12_LEVEL_OPTIONS:
  EducationLevelOption[] = [
    { value: '一年级', label: '一年级' },
    { value: '二年级', label: '二年级' },
    { value: '三年级', label: '三年级' },
    { value: '四年级', label: '四年级' },
    { value: '五年级', label: '五年级' },
    { value: '六年级', label: '六年级' },
    { value: '七年级', label: '七年级' },
    { value: '八年级', label: '八年级' },
    { value: '九年级', label: '九年级' },
    { value: '高一', label: '高一' },
    { value: '高二', label: '高二' },
    { value: '高三', label: '高三' },
  ]

const VOCATIONAL_LEVEL_OPTIONS:
  EducationLevelOption[] = [
    {
      value: '中职Ⅰ年级',
      label: '职一',
    },
    {
      value: '中职Ⅱ年级',
      label: '职二',
    },
    {
      value: '中职Ⅲ年级',
      label: '职三',
    },
  ]

const ADULT_LEVEL_OPTIONS:
  EducationLevelOption[] = [
    {
      value: '成人入门',
      label: '入门',
    },
    {
      value: '成人进阶',
      label: '进阶',
    },
    {
      value: '成人高级',
      label: '高级',
    },
    {
      value: '成人管理者',
      label: '管理者',
    },
  ]

/* ==================== 手动助手通用层级 ==================== */

const K12_BROAD_LEVEL_OPTIONS:
  EducationLevelOption[] = [
    {
      value: '',
      label: '不限年级',
    },
    {
      value: '小学',
      label: '小学',
    },
    {
      value: '初中',
      label: '初中',
    },
    {
      value: '高中',
      label: '高中',
    },
  ]

const VOCATIONAL_BROAD_LEVEL_OPTIONS:
  EducationLevelOption[] = [
    {
      value: '中职不限年级',
      label: '不限年级',
    },
    {
      value: '',
      label: '不限层级（兼容旧助手）',
    },
  ]

const ADULT_BROAD_LEVEL_OPTIONS:
  EducationLevelOption[] = [
    {
      value: '成人不限层级',
      label: '不限层级',
    },
    {
      value: '',
      label: '不限层级（兼容旧助手）',
    },
  ]

/* ==================== 历史别名 ==================== */

const K12_LEVEL_ALIASES:
  Record<string, string> = {
    '1': '一年级',
    '1年级': '一年级',
    '一年级': '一年级',

    '2': '二年级',
    '2年级': '二年级',
    '二年级': '二年级',

    '3': '三年级',
    '3年级': '三年级',
    '三年级': '三年级',

    '4': '四年级',
    '4年级': '四年级',
    '四年级': '四年级',

    '5': '五年级',
    '5年级': '五年级',
    '五年级': '五年级',

    '6': '六年级',
    '6年级': '六年级',
    '六年级': '六年级',

    '7': '七年级',
    '7年级': '七年级',
    '七年级': '七年级',
    '初一': '七年级',

    '8': '八年级',
    '8年级': '八年级',
    '八年级': '八年级',
    '初二': '八年级',

    '9': '九年级',
    '9年级': '九年级',
    '九年级': '九年级',
    '初三': '九年级',

    '10': '高一',
    '10年级': '高一',
    '十年级': '高一',
    '高一': '高一',

    '11': '高二',
    '11年级': '高二',
    '十一年级': '高二',
    '高二': '高二',

    '12': '高三',
    '12年级': '高三',
    '十二年级': '高三',
    '高三': '高三',

    '小学': '小学',
    '初中': '初中',
    '高中': '高中',
  }

const VOCATIONAL_LEVEL_ALIASES:
  Record<string, string> = {
    '中职Ⅰ年级': '中职Ⅰ年级',
    '中职I年级': '中职Ⅰ年级',
    '中职1年级': '中职Ⅰ年级',
    '中职一年级': '中职Ⅰ年级',
    '中职Ⅰ': '中职Ⅰ年级',
    '中职I': '中职Ⅰ年级',
    '中职1': '中职Ⅰ年级',
    '中职一': '中职Ⅰ年级',
    '职一': '中职Ⅰ年级',

    '中职Ⅱ年级': '中职Ⅱ年级',
    '中职II年级': '中职Ⅱ年级',
    '中职2年级': '中职Ⅱ年级',
    '中职二年级': '中职Ⅱ年级',
    '中职Ⅱ': '中职Ⅱ年级',
    '中职II': '中职Ⅱ年级',
    '中职2': '中职Ⅱ年级',
    '中职二': '中职Ⅱ年级',
    '职二': '中职Ⅱ年级',

    '中职Ⅲ年级': '中职Ⅲ年级',
    '中职III年级': '中职Ⅲ年级',
    '中职3年级': '中职Ⅲ年级',
    '中职三年级': '中职Ⅲ年级',
    '中职Ⅲ': '中职Ⅲ年级',
    '中职III': '中职Ⅲ年级',
    '中职3': '中职Ⅲ年级',
    '中职三': '中职Ⅲ年级',
    '职三': '中职Ⅲ年级',

    '中职不限年级': '中职不限年级',
    '中职不限': '中职不限年级',
    '不限中职年级': '中职不限年级',
  }

const ADULT_LEVEL_ALIASES:
  Record<string, string> = {
    '成人入门': '成人入门',
    '入门': '成人入门',

    '成人进阶': '成人进阶',
    '进阶': '成人进阶',

    '成人高级': '成人高级',
    '高级': '成人高级',

    '成人管理者': '成人管理者',
    '管理者': '成人管理者',

    '成人不限层级': '成人不限层级',
    '成人不限': '成人不限层级',
    '不限成人层级': '成人不限层级',
  }

/* ==================== 公共方法 ==================== */

/**
 * mixed跨域管理页面继续沿用K12层级选项。
 *
 * mixed不是具体教学资源域。进入具体教案运行后，
 * 后端会以教案education_domain快照收敛到真实教学域。
 */
function concreteDomain(
  domain: EducationDomain,
): Exclude<EducationDomain, 'mixed'> {
  if (domain === 'vocational') {
    return 'vocational'
  }

  if (domain === 'adult') {
    return 'adult'
  }

  return 'k12'
}

/**
 * 返回教案、配方使用的具体学习层级。
 *
 * 本函数不返回学段或不限值，保证配方创建页面只能选择
 * 可以参与自动严格匹配的具体层级。
 */
export function getEducationLevelOptions(
  domain: EducationDomain,
): EducationLevelOption[] {
  const resolved = concreteDomain(domain)

  if (resolved === 'vocational') {
    return VOCATIONAL_LEVEL_OPTIONS.map(
      item => ({ ...item }),
    )
  }

  if (resolved === 'adult') {
    return ADULT_LEVEL_OPTIONS.map(
      item => ({ ...item }),
    )
  }

  return K12_LEVEL_OPTIONS.map(
    item => ({ ...item }),
  )
}

/**
 * 返回AI助手可以选择的全部层级。
 *
 * 具体层级automatic=true；
 * 学段、不限值automatic=false。
 */
export function getAssistantLevelOptions(
  domain: EducationDomain,
): AssistantLevelOption[] {
  const resolved = concreteDomain(domain)

  const specific =
    getEducationLevelOptions(resolved)
      .map(item => ({
        ...item,
        automatic: true,
      }))

  let broad:
    EducationLevelOption[]

  if (resolved === 'vocational') {
    broad =
      VOCATIONAL_BROAD_LEVEL_OPTIONS
  } else if (resolved === 'adult') {
    broad =
      ADULT_BROAD_LEVEL_OPTIONS
  } else {
    broad =
      K12_BROAD_LEVEL_OPTIONS
  }

  return [
    ...specific,
    ...broad.map(item => ({
      ...item,
      automatic: false,
    })),
  ]
}

/**
 * 将历史别名规范化为当前教育域的正式保存值。
 *
 * 无法识别或跨教育域的值返回空字符串，调用方应要求用户重新选择。
 */
export function normalizeEducationLevelValue(
  domain: EducationDomain,
  value: string | null | undefined,
): string {
  const trimmed =
    (value || '').trim()

  if (trimmed === '') {
    return ''
  }

  const resolved = concreteDomain(domain)

  if (resolved === 'vocational') {
    return VOCATIONAL_LEVEL_ALIASES[
      trimmed
    ] || ''
  }

  if (resolved === 'adult') {
    return ADULT_LEVEL_ALIASES[
      trimmed
    ] || ''
  }

  return K12_LEVEL_ALIASES[
    trimmed
  ] || ''
}

/** 判断层级是否可以参与自动严格匹配。 */
export function isAutomaticEducationLevel(
  domain: EducationDomain,
  value: string,
): boolean {
  const normalized =
    normalizeEducationLevelValue(
      domain,
      value,
    )

  return getAssistantLevelOptions(domain)
    .some(item =>
      item.value === normalized &&
      item.automatic,
    )
}

/** 返回层级的人话展示名。 */
export function getEducationLevelLabel(
  domain: EducationDomain,
  value: string,
): string {
  const normalized =
    normalizeEducationLevelValue(
      domain,
      value,
    )

  const matched =
    getAssistantLevelOptions(domain)
      .find(item =>
        item.value === normalized,
      )

  return matched?.label || value
}

/** 返回不同教育域的课题输入示例。 */
export function getTopicPlaceholder(
  domain: EducationDomain,
): string {
  if (domain === 'vocational') {
    return '例如：车削外圆、绘制零件三视图、直播电商商品上架'
  }

  if (domain === 'adult') {
    return '例如：新员工客户沟通、项目复盘方法、管理者反馈技巧'
  }

  return '例如：观潮（第二课时）、认识人工智能、勾股定理'
}

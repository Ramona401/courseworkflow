/**
 * education-domain/types.ts — 前端教育域统一类型
 *
 * 教育域：
 *   k12          中小学教学域
 *   vocational   职业教育教学域
 *   adult        成人教育教学域
 *   mixed        平台、区域或教育局跨域管理上下文
 *
 * mixed只用于管理身份与管理组织，不用于普通教学资源归属。
 */

export type EducationDomain =
  | 'k12'
  | 'vocational'
  | 'adult'
  | 'mixed'

/**
 * 后端随登录信息下发的教育画像。
 *
 * 页面不应散落大量 domain === 'vocational' 判断，
 * 应优先读取这些标签和能力开关。
 */
export interface EducationProfile {
  code: EducationDomain
  name: string

  subject_label: string
  grade_label: string
  topic_label: string

  lesson_plan_label: string
  unit_plan_label: string
  learner_profile_label: string
  course_outline_label: string

  curriculum_enabled: boolean
  publisher_enabled: boolean
  major_enabled: boolean
  practical_training_enabled: boolean
}

/** 教育域固定中文名。 */
export const EDUCATION_DOMAIN_LABELS: Record<EducationDomain, string> = {
  k12: '中小学',
  vocational: '职业教育',
  adult: '成人教育',
  mixed: '跨域管理',
}

/** 判断字符串是否为合法教育域。 */
export function isEducationDomain(value: unknown): value is EducationDomain {
  return value === 'k12'
    || value === 'vocational'
    || value === 'adult'
    || value === 'mixed'
}

/**
 * 规范化教育域。
 *
 * 前端缺少字段时保持历史兼容：
 *   - admin/region_admin/district_inspector → mixed
 *   - 其它教学身份 → k12
 */
export function normalizeEducationDomain(
  value: unknown,
  role?: string,
): EducationDomain {
  if (isEducationDomain(value)) return value

  if (
    role === 'admin'
    || role === 'region_admin'
    || role === 'district_inspector'
  ) {
    return 'mixed'
  }

  return 'k12'
}

/** 字段缺失时使用的前端兼容画像。 */
export function fallbackEducationProfile(
  domain: EducationDomain,
): EducationProfile {
  if (domain === 'vocational') {
    return {
      code: 'vocational',
      name: '职业教育',
      subject_label: '课程',
      grade_label: '年级或学期',
      topic_label: '教学主题或工作任务',
      lesson_plan_label: '教学设计',
      unit_plan_label: '课程模块方案',
      learner_profile_label: '学习者情况',
      course_outline_label: '教学依据',
      curriculum_enabled: false,
      publisher_enabled: false,
      major_enabled: true,
      practical_training_enabled: true,
    }
  }

  if (domain === 'adult') {
    return {
      code: 'adult',
      name: '成人教育',
      subject_label: '培训类别',
      grade_label: '学习基础',
      topic_label: '培训主题',
      lesson_plan_label: '培训方案',
      unit_plan_label: '培训项目方案',
      learner_profile_label: '学习者画像',
      course_outline_label: '培训依据',
      curriculum_enabled: false,
      publisher_enabled: false,
      major_enabled: false,
      practical_training_enabled: false,
    }
  }

  if (domain === 'mixed') {
    return {
      code: 'mixed',
      name: '跨域管理',
      subject_label: '课程',
      grade_label: '学习层级',
      topic_label: '教学主题',
      lesson_plan_label: '教学设计',
      unit_plan_label: '教学方案',
      learner_profile_label: '学习者情况',
      course_outline_label: '教学依据',
      curriculum_enabled: true,
      publisher_enabled: true,
      major_enabled: true,
      practical_training_enabled: true,
    }
  }

  return {
    code: 'k12',
    name: '中小学',
    subject_label: '学科',
    grade_label: '年级',
    topic_label: '课题',
    lesson_plan_label: '教案',
    unit_plan_label: '大单元方案',
    learner_profile_label: '班级学情',
    course_outline_label: '课程大纲',
    curriculum_enabled: true,
    publisher_enabled: true,
    major_enabled: false,
    practical_training_enabled: false,
  }
}

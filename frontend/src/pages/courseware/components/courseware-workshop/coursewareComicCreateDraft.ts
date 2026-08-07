/**
 * coursewareComicCreateDraft.ts
 *
 * 知识点漫画创建表单的稳定浏览器草稿协议。
 *
 * 只保存：
 *   - 教材选择ID；
 *   - 知识点编码；
 *   - 叙事模式、视觉风格、格数与布局；
 *   - 教师输入文字；
 *   - 可选助手ID。
 *
 * 不保存：
 *   - 教材正文快照；
 *   - AI提示词；
 *   - 图片、文件或Base64；
 *   - 用户、学校和教育域授权字段。
 */

import type {
  CoursewareComicLayoutMode,
} from '@/api/coursewares'

export interface CoursewareComicCreateDraft {
  title: string

  publisher: string
  semester: string
  textbookUnitId: string
  kpCodes: string[]

  assistantId: string

  narrativeMode: string
  visualStyle: string
  panelCount: number
  layoutMode:
    CoursewareComicLayoutMode

  teacherFocus: string
  planInstruction: string
}

export interface CoursewareComicOption {
  value: string
  label: string
  description: string
}

export const COURSEWARE_COMIC_NARRATIVE_OPTIONS:
  CoursewareComicOption[] = [
    {
      value: 'knowledge_story',
      label: '知识故事',
      description:
        '把知识点放入完整起因、冲突、发现和总结中。',
    },
    {
      value: 'inquiry_mystery',
      label: '探究解谜',
      description:
        '通过问题、证据、猜想和验证逐步形成结论。',
    },
    {
      value: 'role_dialogue',
      label: '角色对话',
      description:
        '由固定角色用问答和观点碰撞呈现概念。',
    },
    {
      value: 'travel_adventure',
      label: '旅行冒险',
      description:
        '用路线、任务和地点变化串联知识内容。',
    },
    {
      value: 'civic_case',
      label: '社会情境案例',
      description:
        '通过生活案例、选择与后果理解道德法治概念。',
    },
  ]

export const COURSEWARE_COMIC_VISUAL_OPTIONS:
  CoursewareComicOption[] = [
    {
      value: 'science_encyclopedia',
      label: '科学百科漫画',
      description:
        '结构清楚、知识对象准确，适合理科和科普内容。',
    },
    {
      value: 'warm_storybook',
      label: '温暖绘本',
      description:
        '柔和亲切、故事感强，适合低龄和人文主题。',
    },
    {
      value: 'modern_flat',
      label: '现代扁平插画',
      description:
        '轮廓简洁、信息清晰，适合概念和流程表达。',
    },
    {
      value: 'chinese_ink',
      label: '现代国风',
      description:
        '融合水墨、工笔和现代教育插画语言。',
    },
    {
      value: 'cinematic_3d',
      label: '电影级3D',
      description:
        '空间和角色表现强，适合冒险、旅行和实验故事。',
    },
    {
      value: 'realistic_illustration',
      label: '写实教学插画',
      description:
        '场景可信、细节准确，适合地理、历史和社会案例。',
    },
  ]

export const COURSEWARE_COMIC_LAYOUT_OPTIONS:
  Array<{
    value: CoursewareComicLayoutMode
    label: string
    description: string
  }> = [
    {
      value: 'grid',
      label: '规则网格',
      description:
        '4格2×2或6格3×2，便于同步观察和比较。',
    },
    {
      value: 'spotlight',
      label: '主格聚焦',
      description:
        '用一格突出关键场景，其余格补充推进。',
    },
    {
      value: 'carousel',
      label: '逐格步进',
      description:
        '7至8格采用主舞台和编号按钮逐格展示。',
    },
  ]

export function defaultCoursewareComicLayout(
  panelCount: number,
): CoursewareComicLayoutMode {
  if (panelCount >= 7) {
    return 'carousel'
  }

  if (panelCount === 5) {
    return 'spotlight'
  }

  return 'grid'
}

export function createCoursewareComicCreateDraft(
  coursewareTitle: string,
): CoursewareComicCreateDraft {
  const normalizedTitle =
    coursewareTitle.trim()

  return {
    title:
      normalizedTitle
        ? `${normalizedTitle}·知识点漫画`
        : '知识点漫画',

    publisher: '',
    semester: '',
    textbookUnitId: '',
    kpCodes: [],

    assistantId: '',

    narrativeMode:
      'knowledge_story',
    visualStyle:
      'science_encyclopedia',
    panelCount:
      4,
    layoutMode:
      'grid',

    teacherFocus: '',
    planInstruction: '',
  }
}

function normalizeString(
  value: unknown,
  fallback: string,
): string {
  return typeof value === 'string'
    ? value
    : fallback
}

function normalizeStringArray(
  value: unknown,
): string[] {
  if (!Array.isArray(value)) {
    return []
  }

  const seen =
    new Set<string>()

  const result: string[] = []

  for (const item of value) {
    if (typeof item !== 'string') {
      continue
    }

    const normalized =
      item.trim()

    if (
      !normalized ||
      seen.has(normalized)
    ) {
      continue
    }

    seen.add(normalized)
    result.push(normalized)
  }

  return result.slice(0, 12)
}

function normalizePanelCount(
  value: unknown,
  fallback: number,
): number {
  if (
    typeof value !== 'number' ||
    !Number.isFinite(value)
  ) {
    return fallback
  }

  return Math.min(
    8,
    Math.max(
      4,
      Math.trunc(value),
    ),
  )
}

function normalizeLayout(
  value: unknown,
  fallback:
    CoursewareComicLayoutMode,
): CoursewareComicLayoutMode {
  if (
    value === 'grid' ||
    value === 'spotlight' ||
    value === 'carousel'
  ) {
    return value
  }

  return fallback
}

export function parseCoursewareComicCreateDraft(
  raw: string,
  fallback:
    CoursewareComicCreateDraft,
): CoursewareComicCreateDraft {
  if (!raw.trim()) {
    return {
      ...fallback,
      kpCodes:
        [...fallback.kpCodes],
    }
  }

  try {
    const parsed =
      JSON.parse(raw) as
        Record<string, unknown>

    const panelCount =
      normalizePanelCount(
        parsed.panelCount,
        fallback.panelCount,
      )

    return {
      title:
        normalizeString(
          parsed.title,
          fallback.title,
        ),

      publisher:
        normalizeString(
          parsed.publisher,
          fallback.publisher,
        ),

      semester:
        normalizeString(
          parsed.semester,
          fallback.semester,
        ),

      textbookUnitId:
        normalizeString(
          parsed.textbookUnitId,
          fallback.textbookUnitId,
        ),

      kpCodes:
        normalizeStringArray(
          parsed.kpCodes,
        ),

      assistantId:
        normalizeString(
          parsed.assistantId,
          fallback.assistantId,
        ),

      narrativeMode:
        normalizeString(
          parsed.narrativeMode,
          fallback.narrativeMode,
        ),

      visualStyle:
        normalizeString(
          parsed.visualStyle,
          fallback.visualStyle,
        ),

      panelCount,

      layoutMode:
        normalizeLayout(
          parsed.layoutMode,
          defaultCoursewareComicLayout(
            panelCount,
          ),
        ),

      teacherFocus:
        normalizeString(
          parsed.teacherFocus,
          fallback.teacherFocus,
        ),

      planInstruction:
        normalizeString(
          parsed.planInstruction,
          fallback.planInstruction,
        ),
    }
  } catch {
    return {
      ...fallback,
      kpCodes:
        [...fallback.kpCodes],
    }
  }
}

export function serializeCoursewareComicCreateDraft(
  draft:
    CoursewareComicCreateDraft,
): string {
  return JSON.stringify(draft)
}

export function coursewareComicRuneLength(
  value: string,
): number {
  return Array.from(value).length
}

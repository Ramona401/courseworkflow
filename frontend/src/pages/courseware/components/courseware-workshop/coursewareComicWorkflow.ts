/**
 * coursewareComicWorkflow.ts
 *
 * 知识点漫画五步工作台纯函数：
 *   - 工作流步骤和中文标签；
 *   - 叙事、风格、比例、清晰度与使用方式选项；
 *   - 根据项目status修正浏览器有效步骤；
 *   - 判断各步骤当前允许执行的操作；
 *   - 查找和校验首格样张；
 *   - 构建第三步视觉设置草稿。
 *
 * 浏览器只调整教学内容、覆盖层和业务选项。
 * 图片提示词、IAOCI及内部关系索引不属于教师端工作流协议。
 */

import type {
  CoursewareComicAspectRatio,
  CoursewareComicImageQuality,
  CoursewareComicInsertionMode,
  CoursewareComicNarrativeMode,
  CoursewareComicPanel,
  CoursewareComicVisualStyle,
  CoursewareComicWorkflowProject,
  CoursewareComicWorkflowStage,
} from '@/api/coursewares'

export interface CoursewareComicWorkflowOption<
  Value extends string,
> {
  value: Value
  label: string
  description: string
}

export interface CoursewareComicStyleSettingsDraft {
  visualStyleSource:
    'courseware' |
    'selected'
  visualStyle: CoursewareComicVisualStyle
  aspectRatio: CoursewareComicAspectRatio
  imageQuality: CoursewareComicImageQuality
  styleInstruction: string
}

export const COURSEWARE_COMIC_WORKFLOW_STEPS:
  Array<{
    stage: CoursewareComicWorkflowStage
    number: number
    label: string
    description: string
  }> = [
    {
      stage: 'source',
      number: 1,
      label: '知识点',
      description:
        '输入知识点并固化可选教材来源。',
    },
    {
      stage: 'storyboard',
      number: 2,
      label: '确认分镜',
      description:
        '查看AI故事、对白和知识呈现。',
    },
    {
      stage: 'style_preview',
      number: 3,
      label: '确认样张',
      description:
        '选择画风与画幅，生成首格完整样张。',
    },
    {
      stage: 'batch_generation',
      number: 4,
      label: '自动生图',
      description:
        '按确认样张生成其余无文字底图。',
    },
    {
      stage: 'refinement',
      number: 5,
      label: '精修与使用',
      description:
        '调整单格文字、题目、气泡和课件使用方式。',
    },
  ]

export const COURSEWARE_COMIC_NARRATIVE_OPTIONS:
  Array<
    CoursewareComicWorkflowOption<
      CoursewareComicNarrativeMode
    >
  > = [
    {
      value: 'knowledge_story',
      label: '知识故事',
      description:
        '用起因、冲突、发现和总结形成完整故事。',
    },
    {
      value: 'inquiry_mystery',
      label: '探究解谜',
      description:
        '通过问题、证据、猜想和验证形成结论。',
    },
    {
      value: 'role_dialogue',
      label: '角色对话',
      description:
        '通过角色问答和观点碰撞呈现概念。',
    },
    {
      value: 'travel_adventure',
      label: '旅行冒险',
      description:
        '用任务、路线和地点变化串联知识。',
    },
    {
      value: 'civic_case',
      label: '社会情境案例',
      description:
        '通过生活案例、选择和后果理解规则。',
    },
  ]

export const COURSEWARE_COMIC_VISUAL_OPTIONS:
  Array<
    CoursewareComicWorkflowOption<
      CoursewareComicVisualStyle
    >
  > = [
    {
      value: 'science_encyclopedia',
      label: '科学百科漫画',
      description:
        '知识对象准确，结构清晰，适合理科和科普。',
    },
    {
      value: 'warm_storybook',
      label: '温暖教育绘本',
      description:
        '柔和亲切、故事感强，适合低龄和人文主题。',
    },
    {
      value: 'modern_flat',
      label: '现代扁平插画',
      description:
        '轮廓简洁、信息层级清楚，适合概念表达。',
    },
    {
      value: 'chinese_ink',
      label: '现代国风',
      description:
        '融合水墨、工笔和现代教学构图。',
    },
    {
      value: 'cinematic_3d',
      label: '电影级3D',
      description:
        '空间层次和角色表现强，适合冒险与实验。',
    },
    {
      value: 'realistic_illustration',
      label: '写实教学插画',
      description:
        '场景可信、细节准确，适合历史地理和社会案例。',
    },
  ]

export const COURSEWARE_COMIC_ASPECT_OPTIONS:
  Array<
    CoursewareComicWorkflowOption<
      CoursewareComicAspectRatio
    >
  > = [
    {
      value: 'courseware',
      label: '课件横屏',
      description:
        '16:9，默认适配1920×1080课件页面。',
    },
    {
      value: '16:9',
      label: '横向16:9',
      description:
        '适合场景叙事和多人横向关系。',
    },
    {
      value: '4:3',
      label: '横向4:3',
      description:
        '主体集中，适合对话与知识对象并列。',
    },
    {
      value: '1:1',
      label: '正方形1:1',
      description:
        '构图均衡，适合独立知识卡式画面。',
    },
    {
      value: '3:4',
      label: '竖向3:4',
      description:
        '适合人物动作和纵向知识结构。',
    },
    {
      value: '9:16',
      label: '竖向9:16',
      description:
        '适合上下推进和移动端长画幅。',
    },
  ]

export const COURSEWARE_COMIC_QUALITY_OPTIONS:
  Array<
    CoursewareComicWorkflowOption<
      CoursewareComicImageQuality
    >
  > = [
    {
      value: 'standard',
      label: '标准',
      description:
        '教学主体清楚，生成速度和成本更均衡。',
    },
    {
      value: 'high',
      label: '高清',
      description:
        '材质、表情和光影细节更丰富。',
    },
  ]

export const COURSEWARE_COMIC_INSERTION_OPTIONS:
  Array<
    CoursewareComicWorkflowOption<
      CoursewareComicInsertionMode
    >
  > = [
    {
      value: 'single_page',
      label: '合成一页',
      description:
        '默认方式，把全部分格组合为一个课件页面。',
    },
    {
      value: 'smart_pages',
      label: '智能分页',
      description:
        '根据格数和内容密度自动拆分页面。',
    },
    {
      value: 'one_panel_per_page',
      label: '一格一页',
      description:
        '每个漫画格单独成为一个课件页面。',
    },
    {
      value: 'library_only',
      label: '仅保存素材库',
      description:
        '保存图片与编辑结果，不立即写入课件。',
    },
  ]

const STAGE_ORDER:
  Record<
    CoursewareComicWorkflowStage,
    number
  > = {
    source: 0,
    storyboard: 1,
    style_preview: 2,
    batch_generation: 3,
    refinement: 4,
  }

export function resolveCoursewareComicEffectiveStage(
  project: CoursewareComicWorkflowProject,
): CoursewareComicWorkflowStage {
  if (
    project.status === 'ready' ||
    project.status === 'inserted'
  ) {
    return 'refinement'
  }

  return project.workflow.stage
}

export function coursewareComicStageIndex(
  stage: CoursewareComicWorkflowStage,
): number {
  return STAGE_ORDER[stage]
}

export function coursewareComicStepCompleted(
  project: CoursewareComicWorkflowProject,
  stage: CoursewareComicWorkflowStage,
): boolean {
  const effective =
    resolveCoursewareComicEffectiveStage(
      project,
    )

  if (
    coursewareComicStageIndex(effective) >
    coursewareComicStageIndex(stage)
  ) {
    return true
  }

  switch (stage) {
  case 'source':
    return true

  case 'storyboard':
    return Boolean(
      project.workflow
        .storyboard_confirmed_at,
    )

  case 'style_preview':
    return Boolean(
      project.workflow
        .style_confirmed_at,
    )

  case 'batch_generation':
    return (
      project.status === 'ready' ||
      project.status === 'inserted'
    )

  case 'refinement':
    return project.status === 'inserted'
  }
}

export function createCoursewareComicStyleSettingsDraft(
  project: CoursewareComicWorkflowProject,
): CoursewareComicStyleSettingsDraft {
  return {
    visualStyleSource:
      project.workflow
        .visual_style_source,

    visualStyle:
      project.visual_style,

    aspectRatio:
      project.workflow.aspect_ratio,
    imageQuality:
      project.workflow.image_quality,
    styleInstruction:
      project.workflow.style_instruction,
  }
}

export function coursewareComicStyleInstructionLength(
  value: string,
): number {
  return Array.from(
    value.trim(),
  ).length
}

export function validateCoursewareComicStyleSettings(
  draft: CoursewareComicStyleSettingsDraft,
): string {
  if (
    draft.visualStyleSource !==
      'courseware' &&
    draft.visualStyleSource !==
      'selected'
  ) {
    return '请选择有效的画风来源。'
  }

  if (
    !COURSEWARE_COMIC_VISUAL_OPTIONS.some(
      option =>
        option.value ===
        draft.visualStyle,
    )
  ) {
    return '请选择有效的美术风格。'
  }

  if (
    !COURSEWARE_COMIC_ASPECT_OPTIONS.some(
      option =>
        option.value ===
        draft.aspectRatio,
    )
  ) {
    return '请选择有效的图片比例。'
  }

  if (
    !COURSEWARE_COMIC_QUALITY_OPTIONS.some(
      option =>
        option.value ===
        draft.imageQuality,
    )
  ) {
    return '请选择有效的图片清晰度。'
  }

  if (
    coursewareComicStyleInstructionLength(
      draft.styleInstruction,
    ) > 1200
  ) {
    return '风格补充要求不能超过1200个字符。'
  }

  return ''
}

export function findCoursewareComicPreviewPanel(
  project: CoursewareComicWorkflowProject,
  panels: CoursewareComicPanel[],
): CoursewareComicPanel | null {
  const previewID =
    project.workflow
      .style_preview_panel_id

  if (previewID) {
    const saved =
      panels.find(
        panel =>
          panel.id === previewID,
      )

    if (saved) {
      return saved
    }
  }

  return (
    panels.find(
      panel =>
        panel.panel_no === 1,
    ) ||
    null
  )
}

export function canConfirmCoursewareComicStoryboard(
  project: CoursewareComicWorkflowProject,
): boolean {
  return (
    project.status === 'planned' &&
    project.workflow.stage ===
      'storyboard' &&
    !project.workflow
      .storyboard_confirmed_at
  )
}

export function canEditCoursewareComicStyle(
  project: CoursewareComicWorkflowProject,
  panels: CoursewareComicPanel[],
): boolean {
  const firstPanel =
    panels.find(
      panel =>
        panel.panel_no === 1,
    )

  return (
    project.status === 'planned' &&
    project.workflow.stage ===
      'style_preview' &&
    Boolean(
      project.workflow
        .storyboard_confirmed_at,
    ) &&
    firstPanel?.status !== 'generating'
  )
}

export function canGenerateCoursewareComicPreview(
  project: CoursewareComicWorkflowProject,
  panels: CoursewareComicPanel[],
): boolean {
  return canEditCoursewareComicStyle(
    project,
    panels,
  )
}

export function canConfirmCoursewareComicPreview(
  project: CoursewareComicWorkflowProject,
  panels: CoursewareComicPanel[],
): boolean {
  const preview =
    findCoursewareComicPreviewPanel(
      project,
      panels,
    )

  return (
    project.status === 'planned' &&
    project.workflow.stage ===
      'style_preview' &&
    !project.workflow
      .style_confirmed_at &&
    Boolean(
      project.workflow
        .style_preview_panel_id,
    ) &&
    preview?.status === 'generated' &&
    Boolean(preview.current_asset_id)
  )
}

export function canGenerateCoursewareComicBatch(
  project: CoursewareComicWorkflowProject,
): boolean {
  return (
    (
      project.status === 'planned' ||
      project.status === 'failed'
    ) &&
    project.workflow.stage ===
      'batch_generation' &&
    Boolean(
      project.workflow
        .style_confirmed_at,
    ) &&
    Boolean(
      project.workflow
        .style_preview_panel_id,
    )
  )
}

export function coursewareComicAspectRatioCSS(
  value: CoursewareComicAspectRatio,
): string {
  switch (value) {
  case '4:3':
    return '4 / 3'

  case '1:1':
    return '1 / 1'

  case '3:4':
    return '3 / 4'

  case '9:16':
    return '9 / 16'

  default:
    return '16 / 9'
  }
}

export function coursewareComicNarrativeLabel(
  value: CoursewareComicNarrativeMode,
): string {
  return (
    COURSEWARE_COMIC_NARRATIVE_OPTIONS.find(
      option =>
        option.value === value,
    )?.label ||
    value
  )
}

export function coursewareComicVisualLabel(
  value: CoursewareComicVisualStyle,
): string {
  return (
    COURSEWARE_COMIC_VISUAL_OPTIONS.find(
      option =>
        option.value === value,
    )?.label ||
    value
  )
}

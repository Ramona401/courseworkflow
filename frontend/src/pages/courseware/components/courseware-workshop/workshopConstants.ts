/**
 * 课件工坊页面共享常量与纯函数。
 *
 * 从CoursewareWorkshopPage.tsx抽出的无状态常量层：
 *   - 工坊统一配色；
 *   - 固定画布尺寸；
 *   - 图片风格快捷预设；
 *   - 六步向导定义；
 *   - status到步骤索引的确定性映射。
 *
 * 本文件只保存纯数据和纯函数，不发请求、不操作页面状态。
 */

/** 工坊统一配色。 */
export const C = {
  primary: '#F59E0B',
  primaryBg:
    'rgba(245,158,11,0.08)',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  success: '#059669',
  white: '#fff',
  danger: '#EF4444',
}

/** 课件固定画布尺寸：1920×1080全高清契约。 */
export const CW_WIDTH = 1920
export const CW_HEIGHT = 1080

/** 当前支持的快捷预设画风键。 */
export type CWImageStyleKey =
  | 'pixar'
  | 'flat'
  | 'ghibli'
  | 'realistic'
  | 'chinese'
  | 'ink_wash'
  | 'guochao'
  | 'storybook'
  | 'science'
  | 'tech'

/**
 * 快捷预设画风。
 *
 * 字段说明：
 *   - key：前后端共同使用的稳定预设键；
 *   - label：界面主标题；
 *   - category：帮助老师快速理解的类型标签；
 *   - desc：界面说明，也可供手动配图快捷风格使用；
 *   - anchorPrompt：生成当前课件正式风格样板时使用的艺术语言。
 *
 * anchorPrompt只描述媒介、线条、色彩、材质和光影，
 * 不预设固定人物、具体环境、Logo、文字或构图。
 */
export interface CWImageStylePreset {
  key: CWImageStyleKey
  label: string
  category: string
  desc: string
  anchorPrompt: string
}

/**
 * 十种默认画风。
 *
 * 原chinese键继续保留，但显示和定义放宽为“东方雅韵”，
 * 不再锁死水墨、工笔、朝代、服饰或具体传统器物。
 */
export const CW_IMG_STYLES:
CWImageStylePreset[] = [
  {
    key: 'pixar',
    label: '🎬 皮克斯3D',
    category: '三维动画',
    desc:
      '圆润饱满的三维造型、柔和体积光和电影级材质，适合低龄、情境和故事内容。',
    anchorPrompt:
      '高品质三维动画渲染画风，圆润饱满的造型语言，柔和全局光照与体积光，明快温暖的配色，细腻材质质感，电影级CG画面质量',
  },
  {
    key: 'flat',
    label: '🎨 扁平插画',
    category: '教育插画',
    desc:
      '几何色块、清晰描边和简洁层级，信息表达直接，适合大多数教学图解。',
    anchorPrompt:
      '现代扁平矢量插画画风，简洁几何色块，干净利落的描边线条，明快和谐的教育风配色，克制使用渐变与阴影，清晰大方的视觉层次',
  },
  {
    key: 'ghibli',
    label: '🌿 治愈手绘',
    category: '自然手绘',
    desc:
      '温暖水彩、自然光影和细腻手绘颗粒，适合故事、自然和人文主题。',
    anchorPrompt:
      '温暖治愈的日系手绘动画插画画风，细腻笔触，柔和水彩质感与自然手绘颗粒，清新通透的色调，自然柔和的光影氛围，富有故事感',
  },
  {
    key: 'realistic',
    label: '📷 写实摄影',
    category: '照片质感',
    desc:
      '真实光影、景深和材质纹理，适合科学观察、职业场景和真实世界展示。',
    anchorPrompt:
      '高清写实摄影画风，照片级真实感，自然可信的光影与景深，细腻真实的材质纹理，专业摄影构图与画面质量，禁止插画化和卡通化渲染',
  },
  {
    key: 'chinese',
    label: '🏮 东方雅韵',
    category: '宽泛国风',
    desc:
      '新中式东方审美，可按内容灵活采用淡彩、水墨留白、工笔细节或宋式雅色。',
    anchorPrompt:
      '宽泛的新中式东方美学画风，雅致克制的东方配色，讲究留白、节奏和含蓄层次，可按画面内容灵活融合淡彩、水墨晕染、工笔线条或宋式清雅质感，不锁定具体朝代、服饰、器物和单一技法',
  },
  {
    key: 'ink_wash',
    label: '🖌️ 水墨写意',
    category: '传统水墨',
    desc:
      '墨色浓淡、宣纸肌理和大面积留白，适合诗词、山水、传统文化和意境表达。',
    anchorPrompt:
      '中国水墨写意画风，墨色浓淡干湿变化，宣纸肌理，含蓄的淡彩点染，大面积留白，简练有力的笔触，清雅通透的东方意境',
  },
  {
    key: 'guochao',
    label: '🦚 现代国潮',
    category: '潮流东方',
    desc:
      '传统纹样结合现代构成和高识别度配色，适合节庆、历史与文化主题。',
    anchorPrompt:
      '现代国潮插画画风，传统东方纹样与现代平面构成融合，朱红、黛青、鎏金等高识别度配色，装饰性云纹与几何秩序并存，精致醒目但不过度繁复',
  },
  {
    key: 'storybook',
    label: '📚 儿童绘本',
    category: '柔和绘本',
    desc:
      '粉彩、蜡笔和纸张质感，亲切柔和，适合低龄、启蒙、故事和情绪教育。',
    anchorPrompt:
      '温柔儿童绘本插画画风，粉彩与蜡笔般的柔和笔触，轻微纸张颗粒，明亮但不刺眼的配色，简洁亲切的造型语言，富有童真和叙事感',
  },
  {
    key: 'science',
    label: '🔬 科普线稿',
    category: '科学图解',
    desc:
      '精准线条、结构标注感和少量功能色，适合实验、器官、机械和知识结构图。',
    anchorPrompt:
      '现代科普线稿与信息图解画风，清晰准确的结构线条，克制的功能性色彩，干净背景，明确层级，兼顾科学严谨性与教学可读性，避免复杂装饰',
  },
  {
    key: 'tech',
    label: '🚀 未来科技',
    category: '数字科技',
    desc:
      '深色基底、霓虹光效、网格和数据粒子，适合信息技术、工程和未来主题。',
    anchorPrompt:
      '未来数字科技画风，深色基底搭配克制霓虹光效，流动数据线条、粒子光点和几何网格，冷色光影层次清晰，具有高级数字时代质感',
  },
]

/** 六步向导定义。 */
export const STEPS = [
  {
    key: 'generate',
    label: 'AI生成方案',
    emoji: '🤖',
    desc: 'AI分析教案，生成页面方案',
  },
  {
    key: 'edit',
    label: '确认方案',
    emoji: '✏️',
    desc: '确认每页课件内容设计',
  },
  {
    key: 'style',
    label: '选择风格',
    emoji: '🎨',
    desc: '选择视觉风格和配色',
  },
  {
    key: 'preview',
    label: '确认导航栏',
    emoji: '🧭',
    desc: '确认导航栏样式并固定',
  },
  {
    key: 'build',
    label: '批量生成',
    emoji: '⚡',
    desc: '用固定导航栏生成全部页面',
  },
  {
    key: 'confirm',
    label: '确认提交',
    emoji: '✅',
    desc: '预览课件效果，确认提交',
  },
]

/**
 * 后端status映射到当前向导步骤索引。
 *
 * generating状态需要使用hasNavTemplate和hasPreviewPages，
 * 区分停在“确认导航栏”还是“批量生成”。
 */
export function statusToStep(
  status: string,
  hasNavTemplate: boolean,
  hasPreviewPages: boolean,
): number {
  if (status === 'draft') return 0
  if (status === 'indexing') return 1
  if (status === 'styling') return 2

  if (status === 'generating') {
    if (hasNavTemplate) return 4
    if (hasPreviewPages) return 3
    return 3
  }

  if (status === 'preview') return 5

  if (
    status === 'confirmed' ||
    status === 'in_pipeline'
  ) {
    return 5
  }

  return 0
}

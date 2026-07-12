/**
 * 课件工坊页面共享常量与纯函数 (workshopConstants.ts)
 *
 * 从 CoursewareWorkshopPage.tsx 抽出的无状态常量层：配色、固定画布尺寸、
 * 图片风格快选预设、六步向导定义、status→step 映射函数。
 * 供主页面及抽出的 CWFullscreenPreview / SlideshowPlayer 共用。纯数据/纯函数，无副作用。
 */

/** 工坊统一配色 */
export const C = {
  primary: '#F59E0B', primaryBg: 'rgba(245,158,11,0.08)',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
  border: '#E5E7EB', success: '#059669', white: '#fff', danger: '#EF4444',
}

/** 课件固定画布尺寸（1920×1080 全高清契约，缩放铺满任意屏） */
export const CW_WIDTH = 1920
export const CW_HEIGHT = 1080

/**
 * 图片风格快选预设: 点选后把 desc 作为风格后缀融入生成框提示词
 * (同一风格不重复堆叠, 换风格则替换)。
 *
 * 字段说明：
 *   - key    标识当前选中(再次点击同一风格=取消)
 *   - label  按钮文案
 *   - desc   追加进【手动配图生成框】提示词的风格描述（老师手动配图快选用，可含"角色"等表述，
 *            因为手动配图时老师已明确要画什么，不会误导）
 *   - anchorPrompt  【设风格锚点·定调图专用】的画风描述。
 *            关键区别：定调图是用来"提取整套课件画风DNA"的，它决定后续每一页配图的风格基调，
 *            因此【绝不能预设人物主体】——否则整套课件每张图都会被强行塞进人物（哪怕是光合作用、
 *            叶片、地图这类本该是事物/示意图的页面）。
 *            所以 anchorPrompt 只描述"画风质感/光影/色彩/线条"，不出现"角色/人物/卡通小人"字样，
 *            让每种画风忠实呈现自己（写实就写实、扁平就扁平），且主体交由各页配图时按需决定。
 */
export const CW_IMG_STYLES: { key: string; label: string; desc: string; anchorPrompt: string }[] = [
  {
    key: 'pixar',
    label: '🎬 皮克斯3D',
    desc: '皮克斯/迪士尼风格3D渲染，角色圆润可爱，柔和的全局打光，明快温暖的配色，细腻材质质感，电影级画质',
    // 定调图：只锁"3D渲染质感/打光/配色/材质"，不锁"角色圆润可爱"这种预设人物的措辞
    anchorPrompt: '皮克斯/迪士尼风格3D渲染画风，柔和的全局光照，明快温暖的配色，细腻的材质质感与体积光，电影级CG画质，圆润饱满的造型语言',
  },
  {
    key: 'flat',
    label: '🎨 扁平插画',
    desc: '现代扁平矢量插画风格，简洁的几何色块，干净利落的线条，明快和谐的教育风配色，无渐变阴影堆砌，清晰大方',
    anchorPrompt: '现代扁平矢量插画画风，简洁的几何色块，干净利落的描边线条，明快和谐的教育风配色，极简的渐变与阴影，清晰大方的构图',
  },
  {
    key: 'ghibli',
    label: '🌿 吉卜力',
    desc: '吉卜力工作室风格手绘插画，温暖细腻的笔触，柔和的水彩质感，自然清新的色调，富有故事感与治愈氛围',
    anchorPrompt: '吉卜力工作室风格手绘插画画风，温暖细腻的笔触，柔和的水彩质感与手绘颗粒感，自然清新治愈的色调，通透的光影氛围',
  },
  {
    key: 'realistic',
    label: '📷 写实摄影',
    desc: '高清写实摄影风格，自然真实的光影，细腻的材质与景深，专业摄影质感，画面真实可信',
    // 定调图：强调"照片级写实"，明确不是插画/卡通，杜绝被皮克斯化
    anchorPrompt: '高清写实摄影画风，照片级真实感，自然真实的光影与景深，细腻的材质纹理，专业单反摄影质感，绝非插画或卡通渲染',
  },
  {
    key: 'chinese',
    label: '🏮 国风',
    desc: '中国风插画，传统东方美学，雅致的国潮配色，融合水墨/工笔元素，古典而不失现代感，意境优美',
    anchorPrompt: '中国风插画画风，传统东方美学，雅致的国潮配色，融合水墨晕染与工笔线条元素，古典而不失现代感的意境',
  },
  {
    key: 'tech',
    label: '🚀 科技感',
    desc: '未来科技风格，深色背景搭配霓虹光效，流动的数据线条与光点，几何科技元素，富有数字时代的高级感',
    anchorPrompt: '未来科技风画风，深色背景搭配霓虹光效，流动的数据线条与光点粒子，几何科技元素与网格，富有数字时代的高级质感',
  },
]

/** 六步向导定义（生成方案→确认方案→选风格→确认导航栏→批量生成→确认提交） */
export const STEPS = [
  { key: 'generate', label: 'AI生成方案', emoji: '🤖', desc: 'AI分析教案，生成页面方案' },
  { key: 'edit',     label: '确认方案',   emoji: '✏️', desc: '确认每页课件内容设计' },
  { key: 'style',    label: '选择风格',   emoji: '🎨', desc: '选择视觉风格和配色' },
  { key: 'preview',  label: '确认导航栏', emoji: '🧭', desc: '确认导航栏样式并固定' },
  { key: 'build',    label: '批量生成',   emoji: '⚡', desc: '用固定导航栏生成全部页面' },
  { key: 'confirm',  label: '确认提交',   emoji: '✅', desc: '预览课件效果，确认提交' },
]

/**
 * 后端 status 映射到当前向导步骤索引。
 * generating 态需用 hasNavTemplate / hasPreviewPages 区分停在第3步(确认导航栏)还是第4步(批量生成)。
 */
export function statusToStep(s: string, hasNavTemplate: boolean, hasPreviewPages: boolean): number {
  if (s === 'draft') return 0
  if (s === 'indexing') return 1
  if (s === 'styling') return 2
  if (s === 'generating') {
    if (hasNavTemplate) return 4
    if (hasPreviewPages) return 3
    return 3
  }
  if (s === 'preview') return 5
  if (s === 'confirmed' || s === 'in_pipeline') return 5
  return 0
}

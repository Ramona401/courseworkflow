/**
 * lifeScienceLabUtils.ts — 生命科学实验室工具函数模块
 *
 * 第5批A优化：
 *   1. 组件内部控制区统一从左侧栏改为底部课堂控制条。
 *   2. 弹窗右侧参数只作为“初始参数 / 融入设置”，不会整体进入课件页。
 *   3. 课件页里保留的是组件底部的轻量课堂控制条，便于老师上课拖动演示。
 */

export type LifeScienceLabParamValue = number | boolean

export interface LifeScienceLabParamDef {
  key: string
  label: string
  type: 'number' | 'boolean'
  min?: number
  max?: number
  step?: number
  defaultValue: LifeScienceLabParamValue
  hint?: string
}

export interface LifeScienceLabTemplate {
  id: string
  group: string
  name: string
  emoji: string
  desc: string
  params: LifeScienceLabParamDef[]
  buildHTML: (params: Record<string, LifeScienceLabParamValue>, rootId: string) => string
}

export interface LifeScienceLabConfig {
  template: LifeScienceLabTemplate
  params: Record<string, LifeScienceLabParamValue>
  width: number
  height: number
  caption?: string
}

export interface LifeScienceLabRefineConfig {
  lab: LifeScienceLabConfig
  positionHint?: string
}

/** 第5批A：标准尺寸放大，适配更大的弹窗和 16:9 课件画布 */
export const LIFE_SCIENCE_LAB_SIZE_PRESETS = [
  { width: 640, height: 420, label: '小(640×420)' },
  { width: 800, height: 520, label: '标准(800×520)' },
  { width: 940, height: 600, label: '大(940×600)' },
] as const

export const LIFE_SCIENCE_LAB_DEFAULT_SIZE_INDEX = 1

function makeElementId(prefix: string): string {
  const suffix = Date.now().toString(36).slice(-6) + Math.random().toString(36).slice(-4)
  return prefix + '-' + suffix
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

function captionHtml(caption?: string): string {
  return caption
    ? '<div style="text-align:center;font-size:14px;color:#6B7280;margin-top:8px;font-style:italic;">' + escapeHtml(caption) + '</div>'
    : ''
}

/**
 * 第5批A：布局覆盖层。
 * 不改每个模板内部 HTML/JS，只覆盖共同类名：
 *   .bl-body / .bl-controls / .bl-stage / .bl-result
 * 这样所有生命科学观察自动变成“上方实验主体 + 底部课堂控制条”。
 */
export function buildLifeScienceLabLayoutOverride(rootId: string): string {
  return '<style id="' + rootId + '-layout-override">\n'
    + '#' + rootId + '{box-shadow:0 14px 36px rgba(5,150,105,0.08);}\n'
    + '#' + rootId + ' .bl-head{height:44px!important;padding:0 18px!important;background:linear-gradient(135deg,#D1FAE5,#F0FDF4)!important;}\n'
    + '#' + rootId + ' .bl-body{height:calc(100% - 44px)!important;display:grid!important;grid-template-columns:1fr!important;grid-template-rows:minmax(0,1fr) auto!important;grid-template-areas:"stage" "controls"!important;}\n'
    + '#' + rootId + ' .bl-stage{grid-area:stage!important;min-width:0!important;min-height:0!important;background:radial-gradient(circle at 50% 30%,#FFFFFF 0%,#FFFFFF 58%,#F0FDF4 100%)!important;}\n'
    + '#' + rootId + ' .bl-controls{grid-area:controls!important;display:grid!important;grid-template-columns:repeat(auto-fit,minmax(150px,1fr))!important;gap:8px 12px!important;align-items:center!important;padding:10px 14px!important;border-right:none!important;border-top:1px solid #D1FAE5!important;background:linear-gradient(180deg,#F0FDF4,#ECFDF5)!important;box-sizing:border-box!important;max-height:150px!important;overflow:auto!important;}\n'
    + '#' + rootId + ' .bl-controls::-webkit-scrollbar{height:6px;width:6px;}\n'
    + '#' + rootId + ' .bl-controls::-webkit-scrollbar-thumb{background:#A7F3D0;border-radius:999px;}\n'
    + '#' + rootId + ' .bl-row{margin-bottom:0!important;min-width:0!important;}\n'
    + '#' + rootId + ' .bl-label{font-size:12px!important;margin-bottom:4px!important;color:#315B4D!important;}\n'
    + '#' + rootId + ' .bl-value{color:#059669!important;}\n'
    + '#' + rootId + ' input[type=range]{height:6px!important;}\n'
    + '#' + rootId + ' button{box-shadow:0 4px 10px rgba(5,150,105,0.16)!important;}\n'
    + '#' + rootId + ' .bl-result{grid-column:1/-1!important;padding:8px 10px!important;border-radius:10px!important;background:#D1FAE5!important;color:#065F46!important;font-size:12px!important;line-height:1.45!important;font-weight:650!important;max-height:54px!important;overflow:auto!important;}\n'
    + '#' + rootId + ' .bl-stage svg,#' + rootId + ' .bl-stage canvas{width:100%!important;height:100%!important;display:block!important;}\n'
    + '</style>\n'
}

export function generateLifeScienceLabEmbed(config: LifeScienceLabConfig): string {
  const id = makeElementId('life-science-lab')
  const rootId = id + '-root'
  const inner = config.template.buildHTML(config.params, rootId)

  return '<div id="' + id + '" style="display:flex;flex-direction:column;align-items:center;padding:12px 0;">\n'
    + '  <div style="width:' + config.width + 'px;height:' + config.height + 'px;">\n'
    + inner + '\n'
    + buildLifeScienceLabLayoutOverride(rootId) + '\n'
    + '  </div>\n'
    + '  ' + captionHtml(config.caption) + '\n'
    + '</div>'
}

export function buildLifeScienceLabRefineInstruction(config: LifeScienceLabRefineConfig): string {
  const { lab, positionHint } = config
  const embedCode = generateLifeScienceLabEmbed(lab)

  const posDesc = positionHint
    ? '位置偏好: ' + positionHint + '。'
    : '请根据页面现有内容布局，在最合适的位置放置生命科学观察区域。'

  return '请在当前课件页面中融入一个可交互的生命科学观察组件（' + lab.template.name + '）。\n\n'
    + '【融入要求】\n'
    + '1. ' + posDesc + '\n'
    + '2. 这是一个纯 HTML/SVG/Canvas/JavaScript 的生命科学互动观察组件，不是静态图片。\n'
    + '3. 组件上方是实验主体，下方是课堂控制条；控制条里的滑杆、按钮和动态读数是上课演示入口，必须保留。\n'
    + '4. 实验区域尺寸为 ' + lab.width + '×' + lab.height + ' 像素，请保持该尺寸不变，避免 SVG/Canvas 拉伸导致标注和结构错位。\n'
    + '5. 保持与页面整体视觉风格协调，可在实验区域外增加简洁标题，但不要遮挡实验主体和底部控制条。\n'
    + '6. 不要删除或大幅改动页面现有的其他内容和布局结构。\n'
    + '7. 以下代码中的 <script> 逻辑、控件 id、SVG/Canvas 结构必须完整保留、一字不改；AI 只负责把它放进页面合适位置。\n'
    + '8. 该组件无需外部依赖，离线 ZIP 与断网课堂均可运行。\n\n'
    + '【以下是需要融入的完整代码】\n\n'
    + embedCode
}

export function buildDefaultLifeScienceLabParams(template: LifeScienceLabTemplate): Record<string, LifeScienceLabParamValue> {
  const out: Record<string, LifeScienceLabParamValue> = {}
  for (const p of template.params) out[p.key] = p.defaultValue
  return out
}

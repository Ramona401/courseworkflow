/**
 * geographyLabUtils.ts — 地理互动实验室公共类型与工具函数
 *
 * 设计原则：
 *   1. 每个地理模板是独立的纯HTML/SVG/Canvas/JavaScript组件；
 *   2. 统一采用gl-*结构协议；
 *   3. 弹窗右侧参数只控制融入课件时的初始状态；
 *   4. 课件页保留组件自身底部课堂控制条；
 *   5. 所有模板必须离线运行并支持同页多实例；
 *   6. 地理图形均为教学简化模型，不作为真实测绘或决策依据。
 */

export type GeographyLabParamValue =
  | number
  | boolean
  | string

export interface GeographyLabParamOption {
  label: string
  value: string
}

export interface GeographyLabParamDef {
  key: string
  label: string
  type: 'number' | 'boolean' | 'select'
  min?: number
  max?: number
  step?: number
  options?: GeographyLabParamOption[]
  defaultValue: GeographyLabParamValue
  hint?: string
}

export interface GeographyLabTemplate {
  id: string
  group: string
  name: string
  emoji: string
  desc: string
  params: GeographyLabParamDef[]
  buildHTML: (
    params: Record<string, GeographyLabParamValue>,
    rootId: string,
  ) => string
}

export interface GeographyLabConfig {
  template: GeographyLabTemplate
  params: Record<string, GeographyLabParamValue>
  width: number
  height: number
  caption?: string
}

export interface GeographyLabRefineConfig {
  lab: GeographyLabConfig
  positionHint?: string
}

/**
 * 适配16:9课件画布和统一大弹窗。
 * 尺寸只控制嵌入组件的外层显示区域。
 */
export const GEOGRAPHY_LAB_SIZE_PRESETS = [
  {
    width: 640,
    height: 420,
    label: '小(640×420)',
  },
  {
    width: 800,
    height: 520,
    label: '标准(800×520)',
  },
  {
    width: 940,
    height: 600,
    label: '大(940×600)',
  },
] as const

export const GEOGRAPHY_LAB_DEFAULT_SIZE_INDEX = 1

function makeElementId(prefix: string): string {
  const timePart = Date.now()
    .toString(36)
    .slice(-6)

  const randomPart = Math.random()
    .toString(36)
    .slice(-4)

  return prefix + '-' + timePart + randomPart
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function captionHtml(caption?: string): string {
  const trimmed = caption?.trim()

  if (!trimmed) return ''

  return '<div style="text-align:center;'
    + 'font-size:14px;'
    + 'color:#64748B;'
    + 'margin-top:8px;'
    + 'font-style:italic;">'
    + escapeHtml(trimmed)
    + '</div>'
}

/**
 * 地理统一布局覆盖层。
 *
 * 不修改各个模板内部脚本，只覆盖共同类名：
 *   .gl-head
 *   .gl-body
 *   .gl-controls
 *   .gl-stage
 *   .gl-row
 *   .gl-result
 *
 * 模板内部可以按左控制区、右主体区设计；
 * 融入课件时统一转换为：
 *
 *   上方地理互动主体
 *   +
 *   底部课堂控制条
 */
export function buildGeographyLabLayoutOverride(
  rootId: string,
): string {
  return '<style id="' + rootId + '-layout-override">\n'
    + '#' + rootId + '{'
    + 'box-shadow:0 14px 36px rgba(15,118,110,0.10);'
    + '}\n'

    + '#' + rootId + ' .gl-head{'
    + 'height:44px!important;'
    + 'padding:0 18px!important;'
    + 'background:linear-gradient(135deg,#CCFBF1,#EFF6FF)!important;'
    + '}\n'

    + '#' + rootId + ' .gl-body{'
    + 'height:calc(100% - 44px)!important;'
    + 'display:grid!important;'
    + 'grid-template-columns:1fr!important;'
    + 'grid-template-rows:minmax(0,1fr) auto!important;'
    + 'grid-template-areas:"stage" "controls"!important;'
    + '}\n'

    + '#' + rootId + ' .gl-stage{'
    + 'grid-area:stage!important;'
    + 'min-width:0!important;'
    + 'min-height:0!important;'
    + 'background:radial-gradient('
    + 'circle at 50% 30%,'
    + '#FFFFFF 0%,'
    + '#F8FAFC 62%,'
    + '#ECFEFF 100%'
    + ')!important;'
    + '}\n'

    + '#' + rootId + ' .gl-controls{'
    + 'grid-area:controls!important;'
    + 'display:grid!important;'
    + 'grid-template-columns:repeat('
    + 'auto-fit,'
    + 'minmax(150px,1fr)'
    + ')!important;'
    + 'gap:8px 12px!important;'
    + 'align-items:center!important;'
    + 'padding:10px 14px!important;'
    + 'border-right:none!important;'
    + 'border-top:1px solid #99F6E4!important;'
    + 'background:linear-gradient('
    + '180deg,'
    + '#F0FDFA,'
    + '#ECFEFF'
    + ')!important;'
    + 'box-sizing:border-box!important;'
    + 'max-height:158px!important;'
    + 'overflow:auto!important;'
    + '}\n'

    + '#' + rootId
    + ' .gl-controls::-webkit-scrollbar{'
    + 'height:6px;'
    + 'width:6px;'
    + '}\n'

    + '#' + rootId
    + ' .gl-controls::-webkit-scrollbar-thumb{'
    + 'background:#5EEAD4;'
    + 'border-radius:999px;'
    + '}\n'

    + '#' + rootId + ' .gl-row{'
    + 'margin-bottom:0!important;'
    + 'min-width:0!important;'
    + '}\n'

    + '#' + rootId + ' .gl-label{'
    + 'font-size:12px!important;'
    + 'margin-bottom:4px!important;'
    + 'color:#334155!important;'
    + '}\n'

    + '#' + rootId + ' .gl-value{'
    + 'color:#0F766E!important;'
    + 'font-weight:800!important;'
    + '}\n'

    + '#' + rootId + ' input[type=range]{'
    + 'height:6px!important;'
    + '}\n'

    + '#' + rootId + ' button{'
    + 'box-shadow:0 4px 10px '
    + 'rgba(15,118,110,0.16)!important;'
    + '}\n'

    + '#' + rootId + ' .gl-result{'
    + 'grid-column:1/-1!important;'
    + 'padding:8px 10px!important;'
    + 'border-radius:10px!important;'
    + 'background:#CCFBF1!important;'
    + 'color:#115E59!important;'
    + 'font-size:12px!important;'
    + 'line-height:1.45!important;'
    + 'font-weight:650!important;'
    + 'max-height:56px!important;'
    + 'overflow:auto!important;'
    + '}\n'

    + '#' + rootId + ' .gl-stage svg,'
    + '#' + rootId + ' .gl-stage canvas{'
    + 'width:100%!important;'
    + 'height:100%!important;'
    + 'display:block!important;'
    + '}\n'

    + '</style>\n'
}

/**
 * 将模板转换为可以嵌入课件页的完整组件。
 *
 * 每次调用都生成独立外层ID与rootId，
 * 保证同一课件页多个地理组件互不干扰。
 */
export function generateGeographyLabEmbed(
  config: GeographyLabConfig,
): string {
  const id = makeElementId('geography-lab')
  const rootId = id + '-root'

  const inner = config.template.buildHTML(
    config.params,
    rootId,
  )

  return '<div id="' + id + '" '
    + 'style="display:flex;'
    + 'flex-direction:column;'
    + 'align-items:center;'
    + 'padding:12px 0;">\n'

    + '  <div style="width:'
    + config.width
    + 'px;height:'
    + config.height
    + 'px;">\n'

    + inner
    + '\n'

    + buildGeographyLabLayoutOverride(rootId)
    + '\n'

    + '  </div>\n'

    + '  '
    + captionHtml(config.caption)
    + '\n'

    + '</div>'
}

/**
 * 构造传给课件页面微调接口的完整融入指令。
 *
 * HTML代码是已验收的确定性组件。
 * AI只负责把整个组件放入页面合适位置，
 * 不允许重写内部脚本和地理模型。
 */
export function buildGeographyLabRefineInstruction(
  config: GeographyLabRefineConfig,
): string {
  const { lab, positionHint } = config

  const embedCode = generateGeographyLabEmbed(lab)

  const positionDescription = positionHint
    ? '位置偏好：' + positionHint + '。'
    : '请根据页面现有内容布局，在最合适的位置放置地理互动探究区域。'

  return '请在当前课件页面中融入一个可交互的地理互动探究组件（'
    + lab.template.name
    + '）。\n\n'

    + '【融入要求】\n'
    + '1. '
    + positionDescription
    + '\n'

    + '2. 这是纯HTML、SVG、Canvas和原生JavaScript地理互动组件，不是静态图片。\n'

    + '3. 组件上方是地理互动主体，下方是课堂控制条；滑杆、按钮、模式切换、图层开关和动态读数必须保留。\n'

    + '4. 互动区域尺寸为'
    + lab.width
    + '×'
    + lab.height
    + '像素，请保持尺寸不变，避免地图、剖面、坐标、标注或图表发生拉伸错位。\n'

    + '5. 可以在组件外增加简洁标题，但不得遮挡互动主体和底部课堂控制条。\n'

    + '6. 不要删除或大幅改动页面已有的其他内容和布局结构。\n'

    + '7. 以下代码中的script逻辑、root作用域、控件绑定、SVG或Canvas结构必须完整保留；AI只负责将其放到页面合适位置。\n'

    + '8. 该组件不依赖在线地图、外部图片、CDN、字体或网络请求，离线ZIP和断网课堂均可运行。\n'

    + '9. 组件属于课堂教学简化模型。比例、时间、地形、天气、气候和空间关系均为教学示意，不得改写成精确测绘或实时监测结果。\n'

    + '10. 不得把该组件描述为可用于导航、防灾决策、工程选址或真实行政边界判定。\n\n'

    + '【以下是需要融入的完整代码】\n\n'

    + embedCode
}

/**
 * 根据模板参数定义生成初始值对象。
 */
export function buildDefaultGeographyLabParams(
  template: GeographyLabTemplate,
): Record<string, GeographyLabParamValue> {
  const output: Record<string, GeographyLabParamValue> = {}

  for (const param of template.params) {
    output[param.key] = param.defaultValue
  }

  return output
}

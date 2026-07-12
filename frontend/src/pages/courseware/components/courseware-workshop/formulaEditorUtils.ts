/**
 * formulaEditorUtils.ts — 课件公式编辑器工具函数模块
 *
 * 职责：
 * 1. 生成自包含的公式 HTML 代码片段（KaTeX + 渲染后的公式）
 * 2. 拼接用于 RefinePage 的微调指令文本
 * 3. LaTeX 公式校验与常用模板
 *
 * 技术选型：
 *   - 渲染库：KaTeX（MIT，纯前端，输出自包含 HTML+CSS，不依赖外部字体 CDN）
 *   - 编辑库：MathLive（MIT，所见即所得数学输入框，虚拟键盘，支持化学 \ce{}）
 *   两库均运行时动态加载，不进主包。
 *
 * 自托管说明（批次0b 换源，2026-07-08）：
 *   KaTeX 已从 jsdelivr CDN 迁移为本服务器自托管，版本 0.16.18 与原 CDN 完全一致，
 *   纯换源零行为变化。文件位于 /www/wwwroot/tedna/uploads/courseware-assets/libs/katex/0.16.18/，
 *   走 Nginx /uploads/courseware-assets/ 映射（CORS 头 + 30 天缓存，与课件字体、
 *   abcjs 音源同款链路）。用绝对 URL（含域名）是刻意的：课件预览 iframe 为 sandbox
 *   独立源，绝对地址在预览、放映、审核工作台、离线 ZIP 导出等所有场景下均可稳定解析。
 *   文件名保持 katex.min.js 不变，生成片段内 script[src*="katex.min.js"] 的去重逻辑不受影响。
 *   MathLive 当前全平台无代码引用（迭代二预留），刻意保持 CDN 地址不自托管。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/formulaEditorUtils.ts
 * 依赖: 无外部依赖，纯函数模块
 */

// ============================================================
// 类型定义
// ============================================================

/** 单个公式的配置参数 */
export interface FormulaConfig {
  /** LaTeX 公式字符串 */
  latex: string
  /** 显示模式：inline 行内 / display 独立居中 */
  displayMode: 'inline' | 'display'
  /** 公式字号（px，默认 28） */
  fontSize: number
  /** 公式颜色（十六进制，默认 #1F2937） */
  color: string
  /** 可选标注文字（如 "勾股定理"、"化学反应方程式" 等） */
  caption?: string
}

/** 微调指令配置 */
export interface FormulaRefineConfig {
  /** 要融入的公式列表 */
  formulas: FormulaConfig[]
  /** 融入位置偏好提示（可选） */
  positionHint?: string
  /** 布局偏好：single 单列 / side-by-side 并排（多公式时有效） */
  layout?: 'single' | 'side-by-side'
}

// ============================================================
// 常量
// ============================================================

/** 自托管前端库基础地址（详见文件头"自托管说明"） */
const LIBS_BASE = 'https://workflow.pkuailab.com/uploads/courseware-assets/libs'

/** KaTeX 自托管地址（原 jsdelivr katex@0.16.18 纯换源；常量名保留 _CDN 后缀避免调用方改名） */
export const KATEX_CSS_CDN = LIBS_BASE + '/katex/0.16.18/katex.min.css'
export const KATEX_JS_CDN = LIBS_BASE + '/katex/0.16.18/katex.min.js'
/** mhchem 扩展（化学方程式 \ce{} 支持） */
export const KATEX_MHCHEM_CDN = LIBS_BASE + '/katex/0.16.18/contrib/mhchem.min.js'

/**
 * MathLive CDN 地址（所见即所得编辑器）
 * 注意：当前全平台无任何代码引用此常量（FormulaEditorModal 迭代一未引入 MathLive，属预留导出）。
 * 待迭代二真正引入时，先把 mathlive 下载至 libs/ 目录再改为自托管地址；
 * 本轮"纯换源"批次刻意不动它，避免自托管一个无人使用的库。
 */
export const MATHLIVE_CDN = 'https://cdn.jsdelivr.net/npm/mathlive@0.105.1/dist/mathlive.min.js'

/** 默认配置 */
export const FORMULA_DEFAULTS: Omit<FormulaConfig, 'latex'> = {
  displayMode: 'display',
  fontSize: 28,
  color: '#1F2937',
}

/** 预置公式字号方案 */
export const FORMULA_SIZE_PRESETS = [
  { size: 20, label: '小（20px）' },
  { size: 24, label: '中（24px）' },
  { size: 28, label: '标准（28px）' },
  { size: 32, label: '大（32px）' },
  { size: 40, label: '特大（40px）' },
] as const

/** 预置颜色方案 */
export const FORMULA_COLOR_PRESETS = [
  { color: '#1F2937', label: '深灰' },
  { color: '#1E40AF', label: '深蓝' },
  { color: '#DC2626', label: '红色' },
  { color: '#059669', label: '绿色' },
  { color: '#7C3AED', label: '紫色' },
  { color: '#B45309', label: '棕色' },
] as const

/**
 * 常用公式模板——按学科分组
 * 老师可直接点选插入，省去手输 LaTeX
 */
export const FORMULA_TEMPLATES: { group: string; items: { label: string; latex: string }[] }[] = [
  {
    group: '📐 数学 · 基础',
    items: [
      { label: '分数', latex: '\\frac{a}{b}' },
      { label: '平方根', latex: '\\sqrt{x}' },
      { label: 'n次根', latex: '\\sqrt[n]{x}' },
      { label: '上标/指数', latex: 'x^{2}' },
      { label: '下标', latex: 'a_{n}' },
      { label: '绝对值', latex: '|x|' },
      { label: '不等式', latex: 'a \\leq b' },
      { label: '约等于', latex: 'a \\approx b' },
      { label: '正负号', latex: '\\pm' },
      { label: '乘号', latex: 'a \\times b' },
      { label: '除号', latex: 'a \\div b' },
    ],
  },
  {
    group: '📐 数学 · 代数',
    items: [
      { label: '一元二次求根', latex: 'x = \\frac{-b \\pm \\sqrt{b^2 - 4ac}}{2a}' },
      { label: '勾股定理', latex: 'a^2 + b^2 = c^2' },
      { label: '对数', latex: '\\log_{a}{b}' },
      { label: '自然对数', latex: '\\ln{x}' },
      { label: '求和', latex: '\\sum_{i=1}^{n} a_i' },
      { label: '乘积', latex: '\\prod_{i=1}^{n} a_i' },
      { label: '极限', latex: '\\lim_{x \\to \\infty} f(x)' },
      { label: '组合数', latex: '\\binom{n}{k}' },
      { label: '矩阵', latex: '\\begin{pmatrix} a & b \\\\ c & d \\end{pmatrix}' },
    ],
  },
  {
    group: '📐 数学 · 微积分',
    items: [
      { label: '导数', latex: "f'(x)" },
      { label: '偏导数', latex: '\\frac{\\partial f}{\\partial x}' },
      { label: '不定积分', latex: '\\int f(x)\\,dx' },
      { label: '定积分', latex: '\\int_{a}^{b} f(x)\\,dx' },
      { label: '二重积分', latex: '\\iint_D f(x,y)\\,dx\\,dy' },
    ],
  },
  {
    group: '📐 数学 · 几何/三角',
    items: [
      { label: '角度', latex: '\\angle ABC = 90°' },
      { label: '平行', latex: 'AB \\parallel CD' },
      { label: '垂直', latex: 'AB \\perp CD' },
      { label: '三角函数', latex: '\\sin\\theta,\\; \\cos\\theta,\\; \\tan\\theta' },
      { label: '圆面积', latex: 'S = \\pi r^2' },
      { label: '向量', latex: '\\vec{a} \\cdot \\vec{b}' },
    ],
  },
  {
    group: '🔬 物理',
    items: [
      { label: '牛顿第二定律', latex: 'F = ma' },
      { label: '万有引力', latex: 'F = G\\frac{m_1 m_2}{r^2}' },
      { label: '动能', latex: 'E_k = \\frac{1}{2}mv^2' },
      { label: '功', latex: 'W = Fs\\cos\\theta' },
      { label: '欧姆定律', latex: 'U = IR' },
      { label: '电功率', latex: 'P = UI = I^2R = \\frac{U^2}{R}' },
      { label: '质能方程', latex: 'E = mc^2' },
      { label: '密度', latex: '\\rho = \\frac{m}{V}' },
      { label: '压强', latex: 'p = \\frac{F}{S}' },
      { label: '速度', latex: 'v = \\frac{s}{t}' },
    ],
  },
  {
    group: '🧪 化学',
    items: [
      { label: '水的合成', latex: '2H_2 + O_2 \\rightarrow 2H_2O' },
      { label: '光合作用', latex: '6CO_2 + 6H_2O \\xrightarrow{光照} C_6H_{12}O_6 + 6O_2' },
      { label: '硫酸铜晶体', latex: 'CuSO_4 \\cdot 5H_2O' },
      { label: '离子方程式', latex: 'H^+ + OH^- = H_2O' },
      { label: '可逆反应', latex: 'N_2 + 3H_2 \\rightleftharpoons 2NH_3' },
      { label: '化学平衡常数', latex: 'K = \\frac{[C]^c[D]^d}{[A]^a[B]^b}' },
    ],
  },
]

// ============================================================
// HTML 代码片段生成
// ============================================================

/**
 * 为单个公式生成唯一 DOM ID
 */
function makeElementId(): string {
  const suffix = Date.now().toString(36).slice(-6) + Math.random().toString(36).slice(-4)
  return 'formula-' + suffix
}

/**
 * 为单个公式生成完整的自包含 HTML 代码片段
 *
 * 生成的代码包含：
 * - KaTeX CSS 引用（link 标签，去重）
 * - KaTeX JS 引用（script 标签，去重）
 * - 渲染脚本（IIFE 封装，等 KaTeX 加载后用 katex.renderToString 渲染）
 * - 容器 div（带样式和可选标注）
 *
 * @param config 公式配置
 * @returns 完整的自包含 HTML 代码
 */
export function generateFormulaEmbed(config: FormulaConfig): string {
  const id = makeElementId()
  const isDisplay = config.displayMode === 'display'

  // 转义 LaTeX 中的反斜杠和引号，防止 JS 字符串内破坏
  const escapedLatex = config.latex
    .replace(/\\/g, '\\\\')
    .replace(/'/g, "\\'")
    .replace(/\n/g, ' ')

  const captionHtml = config.caption
    ? '<div style="text-align:center;font-size:14px;color:#6B7280;margin-top:8px;font-style:italic;">' + escapeHtml(config.caption) + '</div>'
    : ''

  return '<div id="' + id + '" style="' + (isDisplay ? 'text-align:center;padding:16px 20px;' : 'display:inline;') + '">\n'
    + '  <div id="' + id + '-render" style="font-size:' + config.fontSize + 'px;color:' + config.color + ';line-height:1.6;'
    + (isDisplay ? 'display:flex;justify-content:center;align-items:center;' : 'display:inline;') + '"></div>\n'
    + '  ' + captionHtml + '\n'
    + '  <link rel="stylesheet" href="' + KATEX_CSS_CDN + '" />\n'
    + '  <script>\n'
    + '  (function(){\n'
    + '    function loadKaTeX(cb){\n'
    + '      if(typeof katex!=="undefined"){cb();return;}\n'
    + '      if(!document.querySelector("script[src*=\\"katex.min.js\\"]")){\n'
    + '        var s=document.createElement("script");s.src="' + KATEX_JS_CDN + '";\n'
    + '        s.onload=cb;document.head.appendChild(s);\n'
    + '      } else { setTimeout(function(){loadKaTeX(cb);},200); }\n'
    + '    }\n'
    + '    loadKaTeX(function(){\n'
    + '      var el=document.getElementById("' + id + '-render");\n'
    + '      if(!el)return;\n'
    + '      try{\n'
    + '        el.innerHTML=katex.renderToString(\'' + escapedLatex + '\',{displayMode:' + isDisplay + ',throwOnError:false});\n'
    + '      }catch(e){el.textContent="公式渲染失败: "+e.message;}\n'
    + '    });\n'
    + '  })();\n'
    + '  </' + 'script>\n'
    + '</div>'
}

/** HTML 转义辅助 */
function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

/**
 * 为多个公式批量生成代码片段，按布局方式包裹
 */
export function generateMultiFormulaEmbed(
  formulas: FormulaConfig[],
  layout: 'single' | 'side-by-side' = 'single'
): string {
  if (formulas.length === 0) return ''
  if (formulas.length === 1) return generateFormulaEmbed(formulas[0])

  const embeds = formulas.map(f => generateFormulaEmbed(f))

  const layoutStyles: Record<string, string> = {
    'single': 'display:flex;flex-direction:column;gap:20px;align-items:center;',
    'side-by-side': 'display:flex;flex-wrap:wrap;gap:24px;align-items:center;justify-content:center;',
  }

  return '<div style="' + layoutStyles[layout] + 'padding:16px 0;">\n  ' + embeds.join('\n  ') + '\n</div>'
}

// ============================================================
// 微调指令拼接
// ============================================================

/**
 * 根据公式配置生成 RefinePage 的微调指令文本
 */
export function buildFormulaRefineInstruction(config: FormulaRefineConfig): string {
  const { formulas, positionHint, layout = 'single' } = config
  if (formulas.length === 0) return ''

  const embedCode = generateMultiFormulaEmbed(formulas, layout)
  const count = formulas.length

  const posDesc = positionHint
    ? '位置偏好: ' + positionHint + '。'
    : '请根据页面现有内容布局，在最合适的位置放置公式区域。'

  const layoutDesc = layout === 'side-by-side' ? '并排展示' : '纵向排列'

  return '请在当前课件页面中融入数学/物理/化学公式。\n\n'
    + '【融入要求】\n'
    + '1. ' + posDesc + '\n'
    + '2. 共 ' + count + ' 个公式，' + layoutDesc + '展示。\n'
    + '3. 保持与页面整体视觉风格协调——配色、圆角、间距与现有元素和谐统一。\n'
    + '4. 公式区域应有清晰的视觉边界（如卡片容器），可加适当标题。\n'
    + '5. 不要删除或大幅改动页面现有的其他内容和布局结构。\n'
    + '6. 以下代码中的 <link>、<script> 标签和 JavaScript 逻辑必须完整保留，不要修改脚本内容，尤其是自托管库地址一个字符都不能改。\n'
    + '7. KaTeX CSS link 标签保持在代码片段内即可，浏览器会自动去重加载。\n\n'
    + '【以下是需要融入的完整代码（含 KaTeX 库引用和渲染脚本）】\n\n'
    + embedCode
}

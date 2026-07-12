/**
 * strokeOrderUtils.ts — 课件笔顺动画工具函数模块
 *
 * 职责：
 * 1. 生成自包含的 Hanzi Writer HTML 代码片段（含库引用 + 田字格 + 初始化脚本）
 * 2. 拼接用于 RefinePage 的微调指令文本
 * 3. 汉字输入校验与过滤
 *
 * 自托管说明（批次0b 换源，2026-07-08）：
 *   Hanzi Writer 已从 jsdelivr CDN 迁移为本服务器自托管（版本 3.5.0，原 CDN 锁 3.5 系列，
 *   纯换源零行为变化）。关键点：hanzi-writer.min.js 只是渲染引擎，每个字的笔画数据
 *   默认仍会按需从 jsdelivr 拉取（hanzi-writer-data 包的单字 JSON）——只换 js 不接管
 *   数据加载，笔画数据依旧偷偷走外网。因此本模块生成的所有嵌入片段均在
 *   HanziWriter.create 中显式传入 charDataLoader，指向自托管的 HANZI_DATA_BASE
 *   （hanzi-writer-data@2.0.1，约 9575 个单字 JSON 已全量落盘）。
 *   StrokeOrderModal 的弹窗实时预览同样必须传 charDataLoader（从本模块导入常量，
 *   保证单一事实来源）。走 Nginx /uploads/courseware-assets/ 映射（CORS + 30 天缓存）。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/strokeOrderUtils.ts
 * 依赖: 无外部依赖，纯函数模块
 */

// ============================================================
// 类型定义
// ============================================================

/** 单个汉字的笔顺动画配置参数 */
export interface StrokeCharConfig {
  /** 目标汉字（单个） */
  char: string
  /** 动画播放速度（Hanzi Writer 的 strokeAnimationSpeed，1-10，默认 3） */
  speed: number
  /** 笔画渲染颜色（十六进制色值，默认 #333） */
  strokeColor: string
  /** 田字格/动画区域尺寸（正方形边长像素，默认 240） */
  size: number
  /** 是否显示田字格辅助线（默认 true） */
  showGrid: boolean
  /** 是否启用点击播放（默认 true） */
  clickToAnimate: boolean
  /** 是否启用描红练习模式（默认 false，true 时 clickToAnimate 自动忽略） */
  quizMode: boolean
}

/** 生成的嵌入代码片段结果 */
export interface StrokeEmbedResult {
  /** 完整的自包含 HTML 代码（含 script 标签） */
  html: string
  /** 该代码块唯一 ID（供 DOM 定位） */
  elementId: string
}

/** 微调指令配置 */
export interface StrokeRefineConfig {
  /** 要融入的汉字列表 */
  chars: StrokeCharConfig[]
  /** 融入位置偏好提示（可选，如"放在生字卡片旁边"） */
  positionHint?: string
  /** 布局偏好：horizontal 横排 / vertical 竖排 / grid 网格（多字时有效） */
  layout?: 'horizontal' | 'vertical' | 'grid'
}

// ============================================================
// 常量
// ============================================================

/** 自托管前端库基础地址（详见文件头"自托管说明"） */
const LIBS_BASE = 'https://workflow.pkuailab.com/uploads/courseware-assets/libs'

/**
 * Hanzi Writer 自托管地址（原 jsdelivr hanzi-writer@3.5 纯换源，锁定 3.5.0）
 * 导出供 StrokeOrderModal 复用，弹窗预览与生成片段共用同一地址（单一事实来源）
 */
export const HANZI_WRITER_CDN = LIBS_BASE + '/hanzi-writer/3.5.0/hanzi-writer.min.js'

/**
 * 汉字笔画数据自托管基础地址（hanzi-writer-data@2.0.1，约 9575 个单字 JSON）
 *
 * 所有 HanziWriter.create 调用（生成片段 + 弹窗预览）必须显式传 charDataLoader
 * 指向本地址，否则笔画数据仍会走 jsdelivr 外网 CDN。
 * 地址以 / 结尾，拼接方式：HANZI_DATA_BASE + encodeURIComponent(字) + '.json'
 * （中文文件名 URL 编码后 Nginx 已验证可正常 200 返回）
 */
export const HANZI_DATA_BASE = LIBS_BASE + '/hanzi-writer-data/2.0.1/'

/** 默认配置 */
export const STROKE_DEFAULTS: Omit<StrokeCharConfig, 'char'> = {
  speed: 3,
  strokeColor: '#333',
  size: 240,
  showGrid: true,
  clickToAnimate: true,
  quizMode: false,
}

/** 预置笔画颜色方案（与课件工坊暖色系协调） */
export const STROKE_COLOR_PRESETS = [
  { color: '#333333', label: '墨黑' },
  { color: '#E24B4A', label: '朱红' },
  { color: '#378ADD', label: '湖蓝' },
  { color: '#1D9E75', label: '翠绿' },
  { color: '#D85A30', label: '橘橙' },
  { color: '#7F77DD', label: '靛紫' },
] as const

/** 速度档位标签 */
export const STROKE_SPEED_LABELS: Record<number, string> = {
  1: '极慢',
  2: '慢速',
  3: '适中',
  4: '稍快',
  5: '快速',
}

// ============================================================
// 汉字校验
// ============================================================

/**
 * 判断是否为 CJK 统一汉字（基本区 U+4E00 - U+9FFF）
 * Hanzi Writer 主要覆盖此范围约 9500 字
 */
export function isCJKChar(ch: string): boolean {
  const code = ch.codePointAt(0)
  return code !== undefined && code >= 0x4E00 && code <= 0x9FFF
}

/**
 * 从输入字符串中提取有效汉字列表（去重保序）
 * @param input 用户输入文本
 * @returns 去重后的汉字数组
 */
export function extractValidChars(input: string): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const ch of input) {
    if (isCJKChar(ch) && !seen.has(ch)) {
      seen.add(ch)
      result.push(ch)
    }
  }
  return result
}

// ============================================================
// HTML 代码片段生成
// ============================================================

/**
 * 为单个汉字生成唯一 DOM ID
 * 格式: hz-{汉字}-{时间戳后6位}
 */
function makeElementId(char: string): string {
  const suffix = Date.now().toString(36).slice(-6)
  return `hz-${char}-${suffix}`
}

/**
 * 生成田字格 SVG 辅助线（米字格样式：十字线 + 对角线）
 * @param size 田字格尺寸
 * @returns SVG 字符串
 */
function buildGridSVG(size: number): string {
  const half = size / 2
  return '<svg xmlns="http://www.w3.org/2000/svg" width="' + size + '" height="' + size + '" '
    + 'style="position:absolute;inset:0;pointer-events:none;z-index:0;">'
    + '<line x1="0" y1="' + half + '" x2="' + size + '" y2="' + half + '" stroke="#e0d6cc" stroke-width="1" stroke-dasharray="6,4"/>'
    + '<line x1="' + half + '" y1="0" x2="' + half + '" y2="' + size + '" stroke="#e0d6cc" stroke-width="1" stroke-dasharray="6,4"/>'
    + '<line x1="0" y1="0" x2="' + size + '" y2="' + size + '" stroke="#f0e8df" stroke-width="1" stroke-dasharray="6,4"/>'
    + '<line x1="' + size + '" y1="0" x2="0" y2="' + size + '" stroke="#f0e8df" stroke-width="1" stroke-dasharray="6,4"/>'
    + '</svg>'
}

/**
 * 为单个汉字生成完整的自包含 HTML 代码片段
 *
 * 生成的代码包含：
 * - 外层容器 div（带田字格边框样式）
 * - 可选田字格 SVG 辅助线
 * - Hanzi Writer 挂载目标 div
 * - 库 script 引用（多个笔顺块共享一个库，script 标签内含去重逻辑）
 * - 初始化脚本（IIFE 封装避免全局污染），内含 charDataLoader 指向自托管笔画数据
 *
 * @param config 单字配置
 * @returns 嵌入结果（HTML + 元素 ID）
 */
export function generateStrokeEmbed(config: StrokeCharConfig): StrokeEmbedResult {
  const id = makeElementId(config.char)
  const targetId = id + '-target'
  const gridHtml = config.showGrid ? buildGridSVG(config.size) : ''
  const funcName = 'initHZ_' + id.replace(/-/g, '_')

  // 根据模式决定提示文字
  const hintText = config.quizMode
    ? '用鼠标按笔顺书写'
    : (config.clickToAnimate ? '点击播放笔顺' : '')
  const hintHtml = hintText
    ? '<div style="text-align:center;font-size:14px;color:#9C8B7E;margin-top:8px;">' + hintText + '</div>'
    : ''

  // 汉字标注（显示在田字格上方）
  const charLabel = '<div style="text-align:center;font-size:20px;font-weight:700;color:#43352B;margin-bottom:6px;">' + config.char + '</div>'

  // 描红练习模式的初始化代码
  const quizCode = config.quizMode
    ? 'w.quiz({onMistake:function(){},onCorrectStroke:function(){},onComplete:function(){}});'
    : ''

  // 点击播放模式的事件绑定代码
  const clickCode = config.clickToAnimate && !config.quizMode
    ? "document.getElementById('" + id + "').addEventListener('click',function(){w.animateCharacter();});"
    : ''

  const html = '<div id="' + id + '" style="display:inline-flex;flex-direction:column;align-items:center;gap:4px;">\n'
    + '  ' + charLabel + '\n'
    + '  <div style="width:' + config.size + 'px;height:' + config.size + 'px;border:2px solid #d4c8bb;border-radius:8px;position:relative;background:#fffdf9;overflow:hidden;'
    + (config.clickToAnimate && !config.quizMode ? 'cursor:pointer;' : '') + '">\n'
    + '    ' + gridHtml + '\n'
    + '    <div id="' + targetId + '" style="position:relative;z-index:1;width:100%;height:100%;"></div>\n'
    + '  </div>\n'
    + '  ' + hintHtml + '\n'
    + '  <script>\n'
    + '  (function(){\n'
    + '    function ' + funcName + '(){\n'
    + '      if(typeof HanziWriter==="undefined"){setTimeout(' + funcName + ',200);return;}\n'
    + '      var w=HanziWriter.create("' + targetId + '","' + config.char + '",{\n'
    + '        width:' + config.size + ',height:' + config.size + ',padding:12,\n'
    + '        strokeAnimationSpeed:' + (config.speed * 0.5) + ',\n'
    + '        delayBetweenStrokes:300,\n'
    + '        strokeColor:"' + config.strokeColor + '",\n'
    + '        outlineColor:"#e0d6cc",\n'
    + '        drawingColor:"' + config.strokeColor + '",\n'
    + '        showOutline:true,\n'
    + '        showCharacter:false,\n'
    + '        charDataLoader:function(c){return fetch("' + HANZI_DATA_BASE + '"+encodeURIComponent(c)+".json").then(function(r){if(!r.ok)throw new Error("HTTP "+r.status);return r.json();});}\n'
    + '      });\n'
    + '      ' + quizCode + '\n'
    + '      ' + clickCode + '\n'
    + '    }\n'
    + '    if(!document.querySelector("script[src*=\\"hanzi-writer\\"]")){\n'
    + '      var s=document.createElement("script");\n'
    + '      s.src="' + HANZI_WRITER_CDN + '";\n'
    + '      s.onload=' + funcName + ';\n'
    + '      document.head.appendChild(s);\n'
    + '    } else {\n'
    + '      ' + funcName + '();\n'
    + '    }\n'
    + '  })();\n'
    + '  </' + 'script>\n'
    + '</div>'

  return { html, elementId: id }
}

/**
 * 为多个汉字批量生成代码片段，按布局方式包裹
 *
 * @param chars 多字配置数组
 * @param layout 布局方式
 * @returns 包裹后的完整 HTML
 */
export function generateMultiStrokeEmbed(
  chars: StrokeCharConfig[],
  layout: 'horizontal' | 'vertical' | 'grid' = 'horizontal'
): string {
  if (chars.length === 0) return ''
  if (chars.length === 1) {
    return generateStrokeEmbed(chars[0]).html
  }

  const embeds = chars.map(c => generateStrokeEmbed(c).html)

  // 布局样式
  const layoutStyles: Record<string, string> = {
    horizontal: 'display:flex;flex-wrap:wrap;gap:24px;align-items:flex-start;justify-content:center;',
    vertical: 'display:flex;flex-direction:column;gap:20px;align-items:center;',
    grid: 'display:grid;grid-template-columns:repeat(auto-fit,minmax(' + (chars[0].size + 40) + 'px,1fr));gap:24px;justify-items:center;',
  }

  return '<div style="' + layoutStyles[layout] + 'padding:20px 0;">\n  ' + embeds.join('\n  ') + '\n</div>'
}

// ============================================================
// 微调指令拼接
// ============================================================

/**
 * 根据笔顺配置生成 RefinePage 的微调指令文本
 *
 * 指令结构：
 * 1. 任务描述（告诉 AI 要融入笔顺动画）
 * 2. 布局与位置要求
 * 3. 风格协调要求
 * 4. 完整的自包含代码片段（AI 负责把这段代码融入页面合适位置）
 *
 * @param config 微调指令配置
 * @returns 完整的微调指令字符串
 */
export function buildStrokeRefineInstruction(config: StrokeRefineConfig): string {
  const { chars, positionHint, layout = 'horizontal' } = config

  if (chars.length === 0) return ''

  // 生成代码片段
  const embedCode = generateMultiStrokeEmbed(chars, layout)

  // 汉字列表描述
  const charList = chars.map(c => '"' + c.char + '"').join(', ')
  const charCount = chars.length

  // 布局描述
  const layoutDesc: Record<string, string> = {
    horizontal: '横向排列',
    vertical: '纵向排列',
    grid: '网格排列',
  }

  // 模式描述
  const hasQuiz = chars.some(c => c.quizMode)
  const modeDesc = hasQuiz
    ? '描红练习模式（学生可以用鼠标按笔顺书写，写错会提示）'
    : '点击播放模式（点击可查看笔顺动画）'

  // 位置提示
  const posDesc = positionHint
    ? '位置偏好: ' + positionHint + '。'
    : '请根据页面现有内容布局，在最合适的位置放置笔顺动画区域。'

  return '请在当前课件页面中融入汉字笔顺动画组件。\n\n'
    + '【融入要求】\n'
    + '1. ' + posDesc + '\n'
    + '2. ' + charCount + '个汉字（' + charList + '）' + layoutDesc[layout] + '展示，交互模式为' + modeDesc + '。\n'
    + '3. 保持与页面整体视觉风格协调——配色、圆角、间距、字体等与现有元素和谐统一。\n'
    + '4. 如果页面已有生字展示区域，优先把笔顺动画融入或紧邻该区域，增强教学连贯性。\n'
    + '5. 不要删除或大幅改动页面现有的其他内容和布局结构。\n'
    + '6. 笔顺动画区域应有清晰的视觉边界（如卡片容器），标题可用"笔顺练习"或类似文案。\n'
    + '7. 以下代码中的 <script> 标签和 JavaScript 逻辑必须完整保留，不要修改脚本内容，尤其是库地址和 charDataLoader 数据地址一个字符都不能改（改了笔画会加载失败）。\n\n'
    + '【以下是需要融入的完整代码（含田字格、Hanzi Writer 库引用和初始化脚本）】\n\n'
    + embedCode
}

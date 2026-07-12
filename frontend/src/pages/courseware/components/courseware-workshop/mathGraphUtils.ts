/**
 * mathGraphUtils.ts — 课件数学动态图形工具函数模块(批次1b首发;1e/1f风格升级;1g淡雅化,2026-07-08)
 *
 * 职责:
 * 1. 定义数学图形模板/参数/配置的类型体系(模板本体在 mathGraphTemplates.ts)
 * 2. 生成自包含的 JSXGraph 交互图形 HTML 代码片段(含库加载+画板初始化+构造脚本)
 * 3. 拼接用于 RefinePage 的微调指令文本
 * 4. MATH_BOARD_STYLE_JS 全局画板风格(单一真相源,弹窗预览与课件 embed 共用)
 * 5. (批次1g)applyMathPalette 构造代码调色板映射——48个模板零改动统一"换装":
 *    高饱和原色→莫兰迪系低饱和现代色;同色实心点→彩芯白环点(fancy现代点样式)。
 *    弹窗预览与 embed 两侧在执行构造代码前都过同一个函数,视觉升级不破坏单一真相源。
 *
 * 批次1g 视觉调整(响应操作员反馈"细碎/饱和度高/点不够fancy"):
 *   - 网格:theme 3(带次分格)退回 theme 1(仅主线),主线色再淡一档(#E7EBF3);
 *   - 调色:全部高饱和主题色映射至低饱和版本(见 MATH_PALETTE_MAP 注释);
 *   - 点:fillColor 与 strokeColor 同色的"实心点"自动升级为白色描边环样式。
 *
 * 技术选型:
 *   - 渲染库:JSXGraph 1.12.2(MIT/LGPL 双许可,纯前端,交互式动态几何/函数图像)
 *   - 交互形态:模板内置 JSXGraph 滑杆(slider)与可拖拽点(glider/point),
 *     老师在弹窗里设定参数初值,融入课件后课堂上仍可现场拖动演示。
 *
 * 单一真相源设计(关键架构约定):
 *   每个模板的 buildConstruction(params) 产出一段操作 `board` 变量的 JS 构造代码字符串。
 *   - 弹窗预览:MathGraphModal 加载库后先执行 MATH_BOARD_STYLE_JS,再 initBoard,
 *     再执行 applyMathPalette(构造代码);
 *   - 课件融入:generateMathGraphEmbed 把 MATH_BOARD_STYLE_JS 与 applyMathPalette
 *     处理后的同一段构造代码包进自包含 HTML,经 buildMathRefineInstruction 交 AI 融入。
 *   预览所见即课件所得,风格/调色/构造三条线均共享一份代码,杜绝双实现漂移。
 *
 * 自托管说明(对齐 formulaEditorUtils 范式):
 *   JSXGraph 1.12.2 自托管于 /www/wwwroot/tedna/uploads/courseware-assets/libs/jsxgraph/1.12.2/
 *   (jsxgraphcore.js + jsxgraph.css),走 Nginx /uploads/courseware-assets/ 映射
 *   (CORS 头 + 30 天缓存)。绝对 URL 保证 sandbox iframe/放映/审核/离线 ZIP 全场景可解析。
 *   生成片段内 script[src*="jsxgraphcore.js"] 做浏览器级去重,同页多图只加载一次库。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/mathGraphUtils.ts
 * 依赖: 无外部依赖,纯函数模块(模板注册表在 mathGraphTemplates.ts,避免本文件超600行)
 */

// ============================================================
// 类型定义
// ============================================================

/** 模板参数值类型:数字(滑杆初值)或布尔(显示开关) */
export type MathParamValue = number | boolean

/** 模板单个可调参数的定义 */
export interface MathTemplateParamDef {
  /** 参数键(构造代码内引用) */
  key: string
  /** 中文标签(弹窗参数区显示) */
  label: string
  /** 参数类型:number 数字滑杆 / boolean 复选开关 */
  type: 'number' | 'boolean'
  /** 数字参数的取值范围与步进(type=number 时必填) */
  min?: number
  max?: number
  step?: number
  /** 默认值 */
  defaultValue: MathParamValue
  /** 可选提示文字(参数含义说明) */
  hint?: string
}

/** 数学图形模板定义(模板本体注册在 mathGraphTemplates.ts 的 MATH_GRAPH_TEMPLATES) */
export interface MathGraphTemplate {
  /** 模板唯一ID */
  id: string
  /** 分组名(弹窗左栏分组展示) */
  group: string
  /** 模板名称 */
  name: string
  /** 图标 */
  emoji: string
  /** 一句话教学场景说明 */
  desc: string
  /** JSXGraph 画板坐标范围 [左, 上, 右, 下] */
  boundingBox: [number, number, number, number]
  /** 是否锁定纵横比(几何类模板须为 true 防圆变椭圆) */
  keepAspectRatio: boolean
  /** 是否默认显示坐标轴(批次3a;缺省 true。无坐标语义的模板——如分数圆、角的认识——
   *  设 false 得到干净画板;弹窗内老师可用开关覆盖此默认值) */
  showAxis?: boolean
  /** 可调参数列表 */
  params: MathTemplateParamDef[]
  /**
   * 构造代码生成器——产出操作 `board` 变量的 JS 语句串(可引用全局 JXG)。
   * ⚠ 生成的代码禁止出现 "</script" 字样(会破坏嵌入 HTML);
   * ⚠ 字符串一律用单引号,避免与外层拼接冲突。
   */
  buildConstruction: (params: Record<string, MathParamValue>) => string
}

/** 单个图形的完整生成配置(弹窗组装后交给 embed 生成器) */
export interface MathGraphConfig {
  /** 所用模板 */
  template: MathGraphTemplate
  /** 参数取值(弹窗调参结果) */
  params: Record<string, MathParamValue>
  /** 画板宽度 px */
  width: number
  /** 画板高度 px */
  height: number
  /** 是否显示网格 */
  showGrid: boolean
  /** 是否显示坐标轴(批次3a,弹窗开关最终值) */
  showAxis: boolean
  /** 可选标注文字(图下方斜体小字) */
  caption?: string
}

/** 微调指令配置 */
export interface MathRefineConfig {
  /** 图形配置 */
  graph: MathGraphConfig
  /** 融入位置偏好提示(可选) */
  positionHint?: string
}

// ============================================================
// 常量
// ============================================================

/** 自托管前端库基础地址(详见文件头"自托管说明") */
const LIBS_BASE = 'https://workflow.pkuailab.com/uploads/courseware-assets/libs'

/** JSXGraph 自托管地址(1.12.2,MIT/LGPL;常量命名对齐 KATEX_*_CDN 范式) */
export const JSXGRAPH_JS_CDN = LIBS_BASE + '/jsxgraph/1.12.2/jsxgraphcore.js'
export const JSXGRAPH_CSS_CDN = LIBS_BASE + '/jsxgraph/1.12.2/jsxgraph.css'

/** 默认画板尺寸(1920×1080 课件画布内的合理占位) */
export const MATH_GRAPH_DEFAULTS = {
  width: 640,
  height: 480,
  showGrid: true,
} as const

/** 预置画板尺寸方案(弹窗尺寸选择器用) */
export const MATH_GRAPH_SIZE_PRESETS = [
  { width: 520, height: 400, label: '小(520×400)' },
  { width: 640, height: 480, label: '标准(640×480)' },
  { width: 780, height: 560, label: '大(780×560)' },
] as const

/**
 * 全局画板风格 JS(1e首发/1f精修/1g淡雅化;单一真相源:弹窗预览与课件 embed 共用)——
 * 【字体】中文友好系统字体栈;【网格】theme 1 仅主线(1g撤掉细碎次分格),极淡冷灰;
 * 【坐标轴】柔和灰蓝细线;【刻度数字】11px 中灰退至背景;【滑杆】浅灰底线现代样式。
 * __tednaMathStyledV4 幂等标记(1g换新名,保证旧页面刷新后应用新配置)。
 */
export const MATH_BOARD_STYLE_JS = [
  "if (!JXG.__tednaMathStyledV4) {",
  "  JXG.__tednaMathStyledV4 = true;",
  "  var mgFont = \"font-family:-apple-system,'Segoe UI','PingFang SC','Hiragino Sans GB','Microsoft YaHei',sans-serif;\";",
  "  JXG.Options.text.cssDefaultStyle = mgFont;",
  "  JXG.Options.text.highlightCssDefaultStyle = mgFont;",
  "  /* 批次3b:文字全局钉死,防读数被拖离背板(点的标签是 label 类,不受影响) */",
  "  JXG.Options.text.fixed = true;",
  "  /* 网格:仅主线,极淡(1g:theme 3→1,撤细分格,色再淡一档) */",
  "  JXG.Options.grid.theme = 1;",
  "  JXG.Options.grid.strokeColor = '#E7EBF3';",
  "  JXG.Options.grid.strokeOpacity = 0.9;",
  "  JXG.Options.grid.highlight = false;",
  "  /* 坐标轴:柔和灰蓝,比网格清晰但不喧宾夺主(1g:降一档饱和) */",
  "  JXG.Options.axis.strokeColor = '#8B98AC';",
  "  JXG.Options.axis.highlightStrokeColor = '#8B98AC';",
  "  JXG.Options.axis.strokeWidth = 1.4;",
  "  if (JXG.Options.axis.ticks) {",
  "    JXG.Options.axis.ticks.strokeColor = '#C7CFDC';",
  "    JXG.Options.axis.ticks.highlightStrokeColor = '#C7CFDC';",
  "    JXG.Options.axis.ticks.majorHeight = 6;",
  "    if (JXG.Options.axis.ticks.label) {",
  "      JXG.Options.axis.ticks.label.fontSize = 11;",
  "      JXG.Options.axis.ticks.label.strokeColor = '#9AA6B8';",
  "      JXG.Options.axis.ticks.label.highlightStrokeColor = '#9AA6B8';",
  "    }",
  "  }",
  "  /* defaultAxes 里有一层 label 覆盖,须同步否则字号色值不生效 */",
  "  try {",
  "    var dax = JXG.Options.board.defaultAxes;",
  "    if (dax && dax.x && dax.x.ticks && dax.x.ticks.label) {",
  "      dax.x.ticks.label.fontSize = 11;",
  "      dax.x.ticks.label.strokeColor = '#9AA6B8';",
  "    }",
  "    if (dax && dax.y && dax.y.ticks && dax.y.ticks.label) {",
  "      dax.y.ticks.label.fontSize = 11;",
  "      dax.y.ticks.label.strokeColor = '#9AA6B8';",
  "    }",
  "  } catch(e) { /* 结构不存在则跳过,不阻断渲染 */ }",
  "  /* 滑杆:浅灰底线+中性灰蓝进度线(显式设置的模板不受影响) */",
  "  JXG.Options.slider.baseline.strokeColor = '#E5E7EB';",
  "  JXG.Options.slider.baseline.highlightStrokeColor = '#E5E7EB';",
  "  JXG.Options.slider.baseline.strokeWidth = 2;",
  "  JXG.Options.slider.highline.strokeColor = '#9AA6B8';",
  "  JXG.Options.slider.highline.highlightStrokeColor = '#9AA6B8';",
  "  JXG.Options.slider.highline.strokeWidth = 3;",
  "}",
].join('\n')

// ============================================================
// 调色板映射(批次1g)——构造代码执行前统一"换装"
// ============================================================

/**
 * 高饱和原色 → 莫兰迪系低饱和现代色 映射表。
 * 左列是 48 个模板构造代码里实际使用的原始色值(全大写十六进制),
 * 右列是对应的淡雅版本(降饱和、微提亮,保持色相可辨——教学分色语义不变)。
 * 维护约定:模板新增用色时,若属高饱和原色,在此补一行映射即可全局生效。
 */
const MATH_PALETTE_MAP: [string, string][] = [
  // ============ 批次1i:珊瑚粉主题(儿童友好明快色板,替换1g莫兰迪灰调) ============
  // 蓝系(主图形/斜边/函数曲线):清爽蓝,微提亮提纯
  ['#2563EB', '#6C9BF0'], ['#1E40AF', '#5A85D6'], ['#93C5FD', '#BBD2F7'], ['#DBEAFE', '#EBF2FC'],
  // 红系(强调点/对边/根):珊瑚红,鲜活不脏
  ['#DC2626', '#EE7B70'], ['#FECACA', '#FBE3E0'], ['#FCA5A5', '#F6C3BE'],
  // 绿系(镜像/邻边/第二直线):薄荷绿,清透
  ['#059669', '#5BBFA5'], ['#065F46', '#569E8C'], ['#D1FAE5', '#E7F6F0'], ['#A7F3D0', '#CFEDE2'], ['#6EE7B7', '#A8DECB'],
  // 原橙黄系 → 珊瑚粉系(对称轴/角弧/强调元素整体换色相,告别土黄)
  ['#F59E0B', '#F2879C'], ['#FDE68A', '#FBDDE3'], ['#B45309', '#D4708A'],
  ['#EA580C', '#EF8B9E'], ['#FED7AA', '#FAE2E7'], ['#FDBA74', '#F6C9D3'], ['#9A3412', '#C2687F'],
  // 紫系(滑杆/位似/多边形):薰衣草紫
  ['#7C3AED', '#9B8AE6'], ['#6D28D9', '#8A77DB'], ['#5B21B6', '#7A69C4'], ['#C4B5FD', '#DCD4F5'], ['#DDD6FE', '#EAE5FA'],
  // 深灰(顶点/结论文字)与水蓝(河流等)
  ['#1F2937', '#4B5A6E'], ['#0EA5E9', '#6FC0E8'],
  // ============ 批次1i-2:全量色值穷举后补全(漏网7组;#FFFFFF/#E5E7EB/#D1D5DB 中性色刻意不映射) ============
  // 亮黄(分数圆扇形/高中Ext强调) → 珊瑚粉主题
  ['#FBBF24', '#F7A8B8'],
  // Primary 另一档原色蓝/绿/红 → 对应清爽色系
  ['#60A5FA', '#85ACF2'], ['#34D399', '#6FC9B0'], ['#F87171', '#F19A91'],
  // 高频中性灰(提示文字/辅助虚线) → 更柔和的蓝调灰
  ['#6B7280', '#7B8494'], ['#9CA3AF', '#AAB2BF'],
]

/** 参与"白环点"升级的主题色(映射前的原始色;fill 与 stroke 同色即视为实心点) */
const MATH_POINT_CORE_COLORS = [
  '#2563EB', '#DC2626', '#059669', '#F59E0B', '#7C3AED', '#1F2937', '#EA580C',
]

/**
 * 构造代码调色板映射(批次1g,导出供弹窗预览与 embed 两侧共用):
 *   1. 白环点升级:模板中点的惯用写法是 fillColor 与 strokeColor 同色成对出现
 *      (线段/角弧/文字都是单独 strokeColor,不会误伤)——把这种"实心点"改成
 *      彩色内芯 + 白色描边环(strokeWidth:2),在网格上有干净的"贴纸"分离感;
 *   2. 全局色映射:把表中所有高饱和原色替换为低饱和现代色(含描边/填充/文字)。
 * 纯字符串变换,不解析 JS,顺序敏感:先做成对替换(依赖原始色),再做单色映射。
 */
export function applyMathPalette(code: string): string {
  let out = code
  // 第一步:实心点 → 彩芯白环点(按原始色匹配 fill/stroke 同色对)
  for (const c of MATH_POINT_CORE_COLORS) {
    out = out.split("fillColor:'" + c + "', strokeColor:'" + c + "'")
      .join("fillColor:'" + c + "', strokeColor:'#FFFFFF', strokeWidth:2")
  }
  // 第二步:全局高饱和 → 低饱和映射
  for (const [from, to] of MATH_PALETTE_MAP) {
    out = out.split(from).join(to)
  }
  return out
}

// ============================================================
// HTML 代码片段生成
// ============================================================

/** 为图形容器生成唯一 DOM ID(时间戳36进制+随机,防同页多图冲突) */
function makeElementId(): string {
  const suffix = Date.now().toString(36).slice(-6) + Math.random().toString(36).slice(-4)
  return 'mathgraph-' + suffix
}

/** HTML 转义辅助(标注文字用) */
function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

/**
 * 生成完整的自包含 JSXGraph 图形 HTML 代码片段
 * IIFE 脚本:loadJSXGraph 去重加载库 → 全局风格 → initBoard → 执行调色后的构造代码。
 * 画板交互约定:禁用滚轮缩放与整板平移,保留元素级拖拽(教学交互主体)。
 */
export function generateMathGraphEmbed(config: MathGraphConfig): string {
  const id = makeElementId()
  const bb = config.template.boundingBox
  /* 单一真相源+统一换装:构造代码先过调色板映射(与弹窗预览同一函数) */
  const constructionCode = applyMathPalette(config.template.buildConstruction(config.params))

  const captionHtml = config.caption
    ? '<div style="text-align:center;font-size:14px;color:#6B7280;margin-top:8px;font-style:italic;">' + escapeHtml(config.caption) + '</div>'
    : ''

  return '<div id="' + id + '" style="display:flex;flex-direction:column;align-items:center;padding:12px 0;">\n'
    + '  <link rel="stylesheet" href="' + JSXGRAPH_CSS_CDN + '" />\n'
    + '  <div id="' + id + '-board" class="jxgbox" style="width:' + config.width + 'px;height:' + config.height + 'px;'
    + 'background:#FFFFFF;border:1px solid #E5E7EB;border-radius:12px;overflow:hidden;"></div>\n'
    + '  ' + captionHtml + '\n'
    + '  <script>\n'
    + '  (function(){\n'
    + '    /* 去重加载 JSXGraph:同页多图只注入一次 script */\n'
    + '    function loadJSXGraph(cb){\n'
    + '      if(window.JXG && window.JXG.JSXGraph){cb();return;}\n'
    + '      if(!document.querySelector("script[src*=\\"jsxgraphcore.js\\"]")){\n'
    + '        var s=document.createElement("script");s.src="' + JSXGRAPH_JS_CDN + '";\n'
    + '        s.onload=cb;document.head.appendChild(s);\n'
    + '      } else { setTimeout(function(){loadJSXGraph(cb);},200); }\n'
    + '    }\n'
    + '    loadJSXGraph(function(){\n'
    + '      var el=document.getElementById("' + id + '-board");\n'
    + '      if(!el || el.getAttribute("data-jxg-inited"))return;\n'
    + '      el.setAttribute("data-jxg-inited","1");\n'
    + '      try{\n'
    + '        /* 全局画板风格(字体/网格/坐标轴/滑杆默认值,幂等) */\n'
    + MATH_BOARD_STYLE_JS.split('\n').map(l => '        ' + l).join('\n') + '\n'
    + '        var board=JXG.JSXGraph.initBoard("' + id + '-board",{\n'
    + '          boundingbox:[' + bb.join(',') + '],\n'
    + '          axis:' + (config.showAxis ? 'true' : 'false') + ', grid:' + (config.showGrid ? 'true' : 'false') + ',\n'
    + '          keepaspectratio:' + (config.template.keepAspectRatio ? 'true' : 'false') + ',\n'
    + '          showCopyright:false, showNavigation:false,\n'
    + '          pan:{enabled:false}, zoom:{enabled:false, wheel:false}\n'
    + '        });\n'
    + constructionCode.split('\n').map(l => '        ' + l).join('\n') + '\n'
    + '      }catch(e){ el.innerHTML=\'<div style="padding:20px;color:#DC2626;font-size:14px;">图形渲染失败: \'+e.message+\'</div>\'; }\n'
    + '    });\n'
    + '  })();\n'
    + '  </' + 'script>\n'
    + '</div>'
}

// ============================================================
// 微调指令拼接
// ============================================================

/**
 * 根据图形配置生成 RefinePage 的微调指令文本
 * (范式对齐 buildFormulaRefineInstruction:融入要求 + 完整代码块)
 */
export function buildMathRefineInstruction(config: MathRefineConfig): string {
  const { graph, positionHint } = config
  const embedCode = generateMathGraphEmbed(graph)

  const posDesc = positionHint
    ? '位置偏好: ' + positionHint + '。'
    : '请根据页面现有内容布局,在最合适的位置放置图形区域。'

  return '请在当前课件页面中融入一个交互式数学动态图形(' + graph.template.name + ')。\n\n'
    + '【融入要求】\n'
    + '1. ' + posDesc + '\n'
    + '2. 这是一个可交互的 JSXGraph 画板(学生/老师可拖动滑杆和图上的点进行课堂演示),'
    + '不是静态图片,必须保证画板容器不被遮挡、不被缩小到无法操作。\n'
    + '3. 画板尺寸为 ' + graph.width + '×' + graph.height + ' 像素,请保持该尺寸不变(不要用 CSS 强行拉伸压缩,会导致坐标错位)。\n'
    + '4. 保持与页面整体视觉风格协调——配色、圆角、间距与现有元素和谐统一,图形区域可加适当标题。\n'
    + '5. 不要删除或大幅改动页面现有的其他内容和布局结构。\n'
    + '6. 以下代码中的 <link>、<script> 标签和 JavaScript 逻辑必须完整保留、一字不改——'
    + '尤其是自托管库地址(jsxgraphcore.js / jsxgraph.css)和画板初始化脚本,改动任何字符都会导致图形无法渲染。\n'
    + '7. JSXGraph 的 CSS link 与 JS script 由浏览器自动去重,同页已有其他图形也不冲突,保持在代码片段内即可。\n\n'
    + '【以下是需要融入的完整代码(含 JSXGraph 库引用、画板初始化与构造脚本)】\n\n'
    + embedCode
}

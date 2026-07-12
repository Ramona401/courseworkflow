/**
 * moleculeUtils.ts — 课件分子实验室工具函数模块(批次2首发,2026-07-09)
 *
 * 职责:
 * 1. 定义分子模板/配置的类型体系(模板本体在 moleculeTemplates.ts)
 * 2. 生成自包含的分子可视化 HTML 代码片段——双模式:
 *    - 3D 模式(3Dmol.js):可鼠标拖拽旋转/滚轮缩放的立体分子,球棍/空间填充/棍状三种样式
 *    - 2D 模式(smiles-drawer):SMILES 串渲染平面结构式(canvas 静态图)
 * 3. 拼接用于 RefinePage 的微调指令文本(区分双模式的融入要求措辞)
 *
 * 关键架构约定(比数学组件更严格的自包含要求):
 *   所有分子结构数据(xyz 坐标)一律内联在生成的 HTML 片段里,
 *   运行时【绝不】向 PubChem 等外部数据源发请求——保证离线 ZIP 导出、
 *   断网课堂、内网部署三种场景下分子照常渲染。
 *
 * 单一真相源设计(对齐 mathGraphUtils 范式):
 *   模板的 buildXYZ() 产出标准 xyz 格式字符串(首行原子数/次行注释/逐行元素坐标),
 *   弹窗预览与课件 embed 共用同一份 xyz 与同一套样式常量,所见即所得。
 *
 * 自托管说明(对齐 formulaEditorUtils / mathGraphUtils 范式):
 *   3Dmol 2.5.5 (BSD-3) 与 smiles-drawer 2.4.1 (MIT) 均自托管于
 *   /www/wwwroot/tedna/uploads/courseware-assets/libs/ 对应版本目录,
 *   走 Nginx /uploads/courseware-assets/ 映射(CORS 头 + 30 天缓存)。
 *   生成片段内按 script[src*="文件名"] 做浏览器级去重,同页多分子只加载一次库。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/moleculeUtils.ts
 * 依赖: 无外部依赖,纯函数模块(模板注册表在 moleculeTemplates.ts,避免本文件超600行)
 */

// ============================================================
// 类型定义
// ============================================================

/** 3D 显示样式:球棍模型 / 空间填充 / 棍状(线框) */
export type MoleculeStyleKey = 'ballstick' | 'spacefill' | 'stick'

/** 3D 分子模板定义(注册在 moleculeTemplates.ts 的 MOLECULE_3D_TEMPLATES) */
export interface Molecule3DTemplate {
  /** 模板唯一ID */
  id: string
  /** 分组名(弹窗左栏分组展示) */
  group: string
  /** 中文名称 */
  name: string
  /** 化学式(展示用,可含 Unicode 下标如 H₂O) */
  formula: string
  /** 图标 */
  emoji: string
  /** 一句话教学场景说明 */
  desc: string
  /**
   * 产出标准 xyz 格式字符串(首行原子数,次行注释,后续每行 "元素 x y z")。
   * 简单分子直接返回内联常量;晶体点阵(NaCl/金刚石)用循环程序生成。
   * ⚠ 数据必须完全自包含,禁止任何运行时外部请求。
   */
  buildXYZ: () => string
}

/** 2D 结构式模板定义(注册在 moleculeTemplates.ts 的 MOLECULE_2D_TEMPLATES) */
export interface Molecule2DTemplate {
  /** 模板唯一ID */
  id: string
  /** 分组名 */
  group: string
  /** 中文名称 */
  name: string
  /** 化学式(展示用) */
  formula: string
  /** SMILES 串(smiles-drawer 渲染输入) */
  smiles: string
  /** 一句话说明 */
  desc: string
}

/** 3D 模式完整生成配置(弹窗组装后交给 embed 生成器) */
export interface Molecule3DConfig {
  /** 所用模板 */
  template: Molecule3DTemplate
  /** 显示样式 */
  styleKey: MoleculeStyleKey
  /** 是否自动旋转 */
  spin: boolean
  /** 是否显示元素标签 */
  showLabels: boolean
  /** 画布背景色(十六进制) */
  background: string
  /** 画布宽度 px */
  width: number
  /** 画布高度 px */
  height: number
  /** 可选标注文字(图下方斜体小字) */
  caption?: string
}

/** 2D 模式完整生成配置 */
export interface Molecule2DConfig {
  /** SMILES 串(模板点选或老师手输) */
  smiles: string
  /** 显示名(融入指令里描述用,手输 SMILES 时可为"自定义分子") */
  displayName: string
  /** 画布宽度 px */
  width: number
  /** 画布高度 px */
  height: number
  /** 可选标注文字 */
  caption?: string
}

/** 微调指令配置(双模式判别联合) */
export type MoleculeRefineConfig =
  | { mode: '3d'; mol: Molecule3DConfig; positionHint?: string }
  | { mode: '2d'; mol: Molecule2DConfig; positionHint?: string }

// ============================================================
// 常量
// ============================================================

/** 自托管前端库基础地址(详见文件头"自托管说明") */
const LIBS_BASE = 'https://workflow.pkuailab.com/uploads/courseware-assets/libs'

/** 3Dmol.js 自托管地址(2.5.5,BSD-3;常量命名对齐 JSXGRAPH_JS_CDN 范式) */
export const MOL3D_JS_CDN = LIBS_BASE + '/3dmol/2.5.5/3Dmol-min.js'

/** smiles-drawer 自托管地址(2.4.1,MIT) */
export const SMILES_DRAWER_JS_CDN = LIBS_BASE + '/smiles-drawer/2.4.1/smiles-drawer.min.js'

/** 3D 画布默认配置 */
export const MOLECULE_3D_DEFAULTS = {
  styleKey: 'ballstick' as MoleculeStyleKey,
  spin: true,
  showLabels: false,
  background: '#FFFFFF',
} as const

/** 预置画布尺寸方案(3D/2D 共用,弹窗尺寸选择器用;分子取景偏方形) */
export const MOLECULE_SIZE_PRESETS = [
  { width: 420, height: 360, label: '小(420×360)' },
  { width: 540, height: 440, label: '标准(540×440)' },
  { width: 660, height: 520, label: '大(660×520)' },
] as const

/** 3D 背景色预设(深色背景下空间填充模型的立体感更强) */
export const MOLECULE_BG_PRESETS = [
  { value: '#FFFFFF', label: '白色' },
  { value: '#F1F5F9', label: '雾灰' },
  { value: '#111827', label: '深空' },
] as const

/** 3D 显示样式选项(弹窗分段选择器用) */
export const MOLECULE_STYLE_OPTIONS: { key: MoleculeStyleKey; label: string; hint: string }[] = [
  { key: 'ballstick', label: '球棍', hint: '原子为球、化学键为棍,结构最清晰' },
  { key: 'spacefill', label: '空间填充', hint: '按范德华半径填充,体现分子真实体积' },
  { key: 'stick', label: '棍状', hint: '只画化学键,适合看骨架与键角' },
]

/**
 * 3D 样式 → 3Dmol setStyle 参数(JS 对象字面量字符串,直接拼进生成脚本)。
 * colorscheme 统一 Jmol 元素配色(C灰/O红/N蓝/H白/Cl绿/Na紫,K12 教材惯用)。
 */
const MOLECULE_STYLE_JS: Record<MoleculeStyleKey, string> = {
  ballstick: '{stick:{radius:0.14,colorscheme:"Jmol"},sphere:{scale:0.28,colorscheme:"Jmol"}}',
  spacefill: '{sphere:{colorscheme:"Jmol"}}',
  stick: '{stick:{radius:0.09,colorscheme:"Jmol"}}',
}

// ============================================================
// 辅助函数
// ============================================================

/** 为分子容器生成唯一 DOM ID(时间戳36进制+随机,防同页多分子冲突;范式同 mathGraphUtils) */
function makeElementId(prefix: string): string {
  const suffix = Date.now().toString(36).slice(-6) + Math.random().toString(36).slice(-4)
  return prefix + '-' + suffix
}

/** HTML 转义辅助(标注文字用) */
function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

/** 判断背景是否为深色(决定元素标签/错误文字用浅色还是深色) */
function isDarkBackground(hex: string): boolean {
  const m = /^#?([0-9a-fA-F]{6})$/.exec(hex.trim())
  if (!m) return false
  const v = parseInt(m[1], 16)
  const r = (v >> 16) & 255, g = (v >> 8) & 255, b = v & 255
  // 相对亮度粗算(ITU-R BT.601 加权),阈值 128
  return (r * 299 + g * 587 + b * 114) / 1000 < 128
}

/** 标注 HTML(双模式共用) */
function captionHtml(caption?: string): string {
  return caption
    ? '<div style="text-align:center;font-size:14px;color:#6B7280;margin-top:8px;font-style:italic;">' + escapeHtml(caption) + '</div>'
    : ''
}

// ============================================================
// 3D 模式:HTML 代码片段生成(3Dmol.js)
// ============================================================

/**
 * 生成完整的自包含 3D 分子 HTML 代码片段。
 * IIFE 脚本:load3Dmol 去重加载库 → createViewer → addModel(内联xyz) → setStyle → render。
 * 交互约定:保留 3Dmol 默认的鼠标拖拽旋转与滚轮缩放(3D 观察是核心教学价值),
 * 可选自动旋转(spin)让分子在放映时自行展示立体结构。
 */
export function generateMolecule3DEmbed(config: Molecule3DConfig): string {
  const id = makeElementId('molecule3d')
  /* 单一真相源:xyz 数据与弹窗预览完全同源;JSON.stringify 处理换行与引号转义 */
  const xyzJson = JSON.stringify(config.template.buildXYZ())
  const styleJs = MOLECULE_STYLE_JS[config.styleKey]
  const dark = isDarkBackground(config.background)
  const labelColor = dark ? '#E5E7EB' : '#374151'
  const errColor = dark ? '#FCA5A5' : '#DC2626'

  return '<div id="' + id + '" style="display:flex;flex-direction:column;align-items:center;padding:12px 0;">\n'
    + '  <div id="' + id + '-view" style="width:' + config.width + 'px;height:' + config.height + 'px;'
    + 'position:relative;background:' + config.background + ';border:1px solid #E5E7EB;border-radius:12px;overflow:hidden;"></div>\n'
    + '  ' + captionHtml(config.caption) + '\n'
    + '  <script>\n'
    + '  (function(){\n'
    + '    /* 去重加载 3Dmol.js:同页多分子只注入一次 script */\n'
    + '    function load3Dmol(cb){\n'
    + '      if(window.$3Dmol){cb();return;}\n'
    + '      if(!document.querySelector("script[src*=\\"3Dmol-min.js\\"]")){\n'
    + '        var s=document.createElement("script");s.src="' + MOL3D_JS_CDN + '";\n'
    + '        s.onload=cb;document.head.appendChild(s);\n'
    + '      } else { setTimeout(function(){load3Dmol(cb);},200); }\n'
    + '    }\n'
    + '    load3Dmol(function(){\n'
    + '      var el=document.getElementById("' + id + '-view");\n'
    + '      if(!el || el.getAttribute("data-mol-inited"))return;\n'
    + '      el.setAttribute("data-mol-inited","1");\n'
    + '      try{\n'
    + '        /* 分子结构数据完全内联(xyz 格式),零外部数据请求,离线可用 */\n'
    + '        var xyzData=' + xyzJson + ';\n'
    + '        var viewer=$3Dmol.createViewer(el,{backgroundColor:"' + config.background + '",antialias:true});\n'
    + '        viewer.addModel(xyzData,"xyz");\n'
    + '        viewer.setStyle({}, ' + styleJs + ');\n'
    + (config.showLabels
      ? '        viewer.addPropertyLabels("elem",{},{fontColor:"' + labelColor + '",fontSize:12,showBackground:false,alignment:"center"});\n'
      : '')
    + '        viewer.zoomTo();\n'
    + '        viewer.render();\n'
    + (config.spin ? '        viewer.spin("y",0.6);\n' : '')
    + '      }catch(e){ el.innerHTML=\'<div style="padding:20px;color:' + errColor + ';font-size:14px;">分子渲染失败: \'+e.message+\'</div>\'; }\n'
    + '    });\n'
    + '  })();\n'
    + '  </' + 'script>\n'
    + '</div>'
}

// ============================================================
// 2D 模式:HTML 代码片段生成(smiles-drawer)
// ============================================================

/**
 * 生成完整的自包含 2D 结构式 HTML 代码片段。
 * IIFE 脚本:loadSmilesDrawer 去重加载库 → SmilesDrawer.parse(SMILES) → drawer.draw 到 canvas。
 * 2D 结构式为静态图(canvas),无交互,融入指令措辞与 3D 不同。
 */
export function generateMolecule2DEmbed(config: Molecule2DConfig): string {
  const id = makeElementId('molecule2d')
  const smilesJson = JSON.stringify(config.smiles.trim())

  return '<div id="' + id + '" style="display:flex;flex-direction:column;align-items:center;padding:12px 0;">\n'
    + '  <div id="' + id + '-wrap" style="background:#FFFFFF;border:1px solid #E5E7EB;border-radius:12px;padding:8px;">\n'
    + '    <canvas id="' + id + '-canvas" width="' + config.width + '" height="' + config.height + '"></canvas>\n'
    + '  </div>\n'
    + '  ' + captionHtml(config.caption) + '\n'
    + '  <script>\n'
    + '  (function(){\n'
    + '    /* 去重加载 smiles-drawer:同页多结构式只注入一次 script */\n'
    + '    function loadSmilesDrawer(cb){\n'
    + '      if(window.SmilesDrawer){cb();return;}\n'
    + '      if(!document.querySelector("script[src*=\\"smiles-drawer.min.js\\"]")){\n'
    + '        var s=document.createElement("script");s.src="' + SMILES_DRAWER_JS_CDN + '";\n'
    + '        s.onload=cb;document.head.appendChild(s);\n'
    + '      } else { setTimeout(function(){loadSmilesDrawer(cb);},200); }\n'
    + '    }\n'
    + '    loadSmilesDrawer(function(){\n'
    + '      var cv=document.getElementById("' + id + '-canvas");\n'
    + '      if(!cv || cv.getAttribute("data-sd-inited"))return;\n'
    + '      cv.setAttribute("data-sd-inited","1");\n'
    + '      try{\n'
    + '        var drawer=new SmilesDrawer.Drawer({width:' + config.width + ',height:' + config.height + ',bondThickness:1.2});\n'
    + '        SmilesDrawer.parse(' + smilesJson + ', function(tree){\n'
    + '          drawer.draw(tree, "' + id + '-canvas", "light", false);\n'
    + '        }, function(err){\n'
    + '          cv.parentNode.innerHTML=\'<div style="padding:20px;color:#DC2626;font-size:14px;">SMILES 解析失败,请检查输入</div>\';\n'
    + '        });\n'
    + '      }catch(e){ cv.parentNode.innerHTML=\'<div style="padding:20px;color:#DC2626;font-size:14px;">结构式渲染失败: \'+e.message+\'</div>\'; }\n'
    + '    });\n'
    + '  })();\n'
    + '  </' + 'script>\n'
    + '</div>'
}

// ============================================================
// 微调指令拼接
// ============================================================

/**
 * 根据分子配置生成 RefinePage 的微调指令文本
 * (范式对齐 buildMathRefineInstruction:融入要求 + 完整代码块;双模式措辞分开)
 */
export function buildMoleculeRefineInstruction(config: MoleculeRefineConfig): string {
  const posDesc = config.positionHint
    ? '位置偏好: ' + config.positionHint + '。'
    : '请根据页面现有内容布局,在最合适的位置放置分子展示区域。'

  if (config.mode === '3d') {
    const m = config.mol
    const embedCode = generateMolecule3DEmbed(m)
    return '请在当前课件页面中融入一个可交互的 3D 分子模型(' + m.template.name + ' ' + m.template.formula + ')。\n\n'
      + '【融入要求】\n'
      + '1. ' + posDesc + '\n'
      + '2. 这是一个可交互的 3D 分子视图(学生/老师可用鼠标拖拽旋转、滚轮缩放观察立体结构),'
      + '不是静态图片,必须保证容器不被遮挡、不被缩小到无法操作。\n'
      + '3. 视图尺寸为 ' + m.width + '×' + m.height + ' 像素,请保持该尺寸不变(3D 画布按容器初始化,CSS 拉伸会导致渲染错位)。\n'
      + '4. 保持与页面整体视觉风格协调——配色、圆角、间距与现有元素和谐统一,分子区域可加适当标题(如化学式)。\n'
      + '5. 不要删除或大幅改动页面现有的其他内容和布局结构。\n'
      + '6. 以下代码中的 <script> 标签、JavaScript 逻辑和内联的分子结构数据必须完整保留、一字不改——'
      + '尤其是自托管库地址(3Dmol-min.js)与 xyz 坐标数据,改动任何字符都会导致分子无法渲染。\n'
      + '7. 3Dmol 库的 script 由浏览器自动去重,同页已有其他分子也不冲突,保持在代码片段内即可。\n\n'
      + '【以下是需要融入的完整代码(含 3Dmol 库引用、内联分子数据与渲染脚本)】\n\n'
      + embedCode
  }

  const m = config.mol
  const embedCode = generateMolecule2DEmbed(m)
  return '请在当前课件页面中融入一个化学 2D 平面结构式(' + m.displayName + ')。\n\n'
    + '【融入要求】\n'
    + '1. ' + posDesc + '\n'
    + '2. 结构式渲染在 canvas 上,尺寸为 ' + m.width + '×' + m.height + ' 像素,请保持 canvas 的 width/height 属性不变(CSS 拉伸会导致模糊)。\n'
    + '3. 保持与页面整体视觉风格协调——配色、圆角、间距与现有元素和谐统一,结构式区域可加适当标题。\n'
    + '4. 不要删除或大幅改动页面现有的其他内容和布局结构。\n'
    + '5. 以下代码中的 <script> 标签、JavaScript 逻辑和 SMILES 数据必须完整保留、一字不改——'
    + '尤其是自托管库地址(smiles-drawer.min.js),改动任何字符都会导致结构式无法渲染。\n'
    + '6. smiles-drawer 库的 script 由浏览器自动去重,同页已有其他结构式也不冲突,保持在代码片段内即可。\n\n'
    + '【以下是需要融入的完整代码(含 smiles-drawer 库引用与渲染脚本)】\n\n'
    + embedCode
}

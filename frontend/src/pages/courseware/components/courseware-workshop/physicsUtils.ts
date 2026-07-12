/**
 * physicsUtils.ts — 课件物理场景工具函数模块(批次3首发,2026-07-09)
 *
 * 职责:
 * 1. 定义物理场景模板/参数/配置的类型体系(模板本体在 physicsTemplates.ts)
 * 2. 生成自包含的 Matter.js 物理仿真 HTML 代码片段——与数学/分子组件的关键差异:
 *    物理是【随时间演化的仿真】,片段内置 ▶播放/⏸暂停 切换按钮与 ↺重置 按钮,
 *    默认暂停在初始状态(老师课堂上按播放开始演示),重置回初始状态并暂停。
 * 3. 拼接用于 RefinePage 的微调指令文本(强调控制条必须完整保留)
 *
 * 单一真相源设计(对齐 mathGraphUtils 架构约定):
 *   每个模板的 buildSetup(params) 产出一段 JS 构造代码字符串,可用变量约定为
 *   Matter / engine / world / W / H(画布宽高 px)。
 *   - 弹窗预览:PhysicsSceneModal 加载库后 new Function 执行同一段代码;
 *   - 课件融入:generatePhysicsEmbed 把同一段代码包进自包含 HTML 的 setup() 里,
 *     重置按钮通过 Composite.clear + 重跑 setup() 实现回到初始状态。
 *   预览所见即课件所得,构造代码双侧共享一份,杜绝双实现漂移。
 *
 * 自托管说明(对齐 formulaEditorUtils / mathGraphUtils / moleculeUtils 范式):
 *   Matter.js 0.20.0 (MIT) 自托管于
 *   /www/wwwroot/tedna/uploads/courseware-assets/libs/matter-js/0.20.0/matter.min.js,
 *   走 Nginx /uploads/courseware-assets/ 映射(CORS 头 + 30 天缓存)。
 *   生成片段内按 script[src*="matter.min.js"] 做浏览器级去重,同页多场景只加载一次库。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/physicsUtils.ts
 * 依赖: 无外部依赖,纯函数模块(模板注册表在 physicsTemplates.ts,避免本文件超600行)
 */

// ============================================================
// 类型定义
// ============================================================

/** 模板参数值类型:数字(滑杆初值)或布尔(显示开关);形制对齐 MathParamValue */
export type PhysicsParamValue = number | boolean

/** 模板单个可调参数的定义(形制对齐 MathTemplateParamDef) */
export interface PhysicsTemplateParamDef {
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
  defaultValue: PhysicsParamValue
  /** 可选提示文字(参数含义说明) */
  hint?: string
}

/** 物理场景模板定义(模板本体注册在 physicsTemplates.ts 的 PHYSICS_TEMPLATES) */
export interface PhysicsTemplate {
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
  /** 可调参数列表 */
  params: PhysicsTemplateParamDef[]
  /**
   * 构造代码生成器——产出一段 JS 语句串,可用变量:
   *   Matter(库全局)/ engine / world(=engine.world)/ W / H(画布宽高 px)。
   * 代码职责:设置 engine.gravity、创建全部刚体/约束(含地面墙体)并 add 进 world。
   * 每次"重置"都会重新执行这段代码,所以代码必须是纯构造、无外部状态依赖。
   * ⚠ 生成的代码禁止出现 "</script" 字样;字符串一律用单引号。
   */
  buildSetup: (params: Record<string, PhysicsParamValue>) => string
}

/** 单个场景的完整生成配置(弹窗组装后交给 embed 生成器) */
export interface PhysicsSceneConfig {
  /** 所用模板 */
  template: PhysicsTemplate
  /** 参数取值(弹窗调参结果) */
  params: Record<string, PhysicsParamValue>
  /** 画布宽度 px */
  width: number
  /** 画布高度 px */
  height: number
  /** 融入课件后是否自动播放(false=初始暂停,老师按播放;默认 false 更符合课堂节奏) */
  autoplay: boolean
  /** 是否显示速度矢量(Matter Render 调试叠加,讲解速度方向变化时有用) */
  showVelocity: boolean
  /** 可选标注文字(图下方斜体小字) */
  caption?: string
}

/** 微调指令配置 */
export interface PhysicsRefineConfig {
  /** 场景配置 */
  scene: PhysicsSceneConfig
  /** 融入位置偏好提示(可选) */
  positionHint?: string
}

// ============================================================
// 常量
// ============================================================

/** 自托管前端库基础地址(详见文件头"自托管说明") */
const LIBS_BASE = 'https://workflow.pkuailab.com/uploads/courseware-assets/libs'

/** Matter.js 自托管地址(0.20.0,MIT;常量命名对齐 JSXGRAPH_JS_CDN 范式) */
export const MATTER_JS_CDN = LIBS_BASE + '/matter-js/0.20.0/matter.min.js'

/** 场景默认配置 */
export const PHYSICS_DEFAULTS = {
  autoplay: false,
  showVelocity: false,
} as const

/** 预置画布尺寸方案(物理场景取横向舞台比例) */
export const PHYSICS_SIZE_PRESETS = [
  { width: 560, height: 360, label: '小(560×360)' },
  { width: 680, height: 440, label: '标准(680×440)' },
  { width: 800, height: 520, label: '大(800×520)' },
] as const

// ============================================================
// 辅助函数
// ============================================================

/** 为场景容器生成唯一 DOM ID(时间戳36进制+随机,防同页多场景冲突;范式同 mathGraphUtils) */
function makeElementId(): string {
  const suffix = Date.now().toString(36).slice(-6) + Math.random().toString(36).slice(-4)
  return 'physics-' + suffix
}

/** HTML 转义辅助(标注文字用) */
function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

// ============================================================
// HTML 代码片段生成
// ============================================================

/**
 * 生成完整的自包含 Matter.js 物理场景 HTML 代码片段。
 * IIFE 脚本:loadMatter 去重加载库 → Engine/Render/Runner 创建 → setup()(构造代码,
 * 重置可重跑)→ 默认按 autoplay 决定初始播放态。
 * 控制条:▶播放/⏸暂停 切换按钮 + ↺重置 按钮(重置 = 清空 world 重跑 setup 并暂停)。
 * 仿真时间步由 Matter.Runner 管理;runner.enabled 开关即播放/暂停。
 */
export function generatePhysicsEmbed(config: PhysicsSceneConfig): string {
  const id = makeElementId()
  /* 单一真相源:构造代码与弹窗预览完全同一份 */
  const setupCode = config.template.buildSetup(config.params)
  const autoplayJs = config.autoplay ? 'true' : 'false'

  /* 控制条按钮统一样式(内联,自包含) */
  const btnStyle = 'border:none;border-radius:10px;padding:8px 18px;font-size:14px;font-weight:700;cursor:pointer;'

  return '<div id="' + id + '" style="display:flex;flex-direction:column;align-items:center;padding:12px 0;">\n'
    + '  <div id="' + id + '-stage" style="width:' + config.width + 'px;height:' + config.height + 'px;'
    + 'background:#FFFFFF;border:1px solid #E5E7EB;border-radius:12px;overflow:hidden;"></div>\n'
    + '  <div style="display:flex;gap:10px;margin-top:10px;">\n'
    + '    <button id="' + id + '-toggle" style="' + btnStyle + 'background:linear-gradient(135deg,#F87171,#DC2626);color:#fff;">▶ 播放</button>\n'
    + '    <button id="' + id + '-reset" style="' + btnStyle + 'background:#F3F4F6;color:#374151;border:1px solid #E5E7EB;">↺ 重置</button>\n'
    + '  </div>\n'
    + '  ' + (config.caption
      ? '<div style="text-align:center;font-size:14px;color:#6B7280;margin-top:8px;font-style:italic;">' + escapeHtml(config.caption) + '</div>'
      : '') + '\n'
    + '  <script>\n'
    + '  (function(){\n'
    + '    /* 去重加载 Matter.js:同页多场景只注入一次 script */\n'
    + '    function loadMatter(cb){\n'
    + '      if(window.Matter){cb();return;}\n'
    + '      if(!document.querySelector("script[src*=\\"matter.min.js\\"]")){\n'
    + '        var s=document.createElement("script");s.src="' + MATTER_JS_CDN + '";\n'
    + '        s.onload=cb;document.head.appendChild(s);\n'
    + '      } else { setTimeout(function(){loadMatter(cb);},200); }\n'
    + '    }\n'
    + '    loadMatter(function(){\n'
    + '      var el=document.getElementById("' + id + '-stage");\n'
    + '      if(!el || el.getAttribute("data-phys-inited"))return;\n'
    + '      el.setAttribute("data-phys-inited","1");\n'
    + '      try{\n'
    + '        var W=' + config.width + ', H=' + config.height + ';\n'
    + '        var engine=Matter.Engine.create();\n'
    + '        var world=engine.world;\n'
    + '        var render=Matter.Render.create({element:el,engine:engine,options:{\n'
    + '          width:W,height:H,wireframes:false,background:\'#FFFFFF\',\n'
    + '          showVelocity:' + (config.showVelocity ? 'true' : 'false') + ',\n'
    + '          pixelRatio:window.devicePixelRatio||1\n'
    + '        }});\n'
    + '        var runner=Matter.Runner.create();\n'
    + '        /* setup():纯构造,重置时清空 world 后可安全重跑回到初始状态 */\n'
    + '        function setup(){\n'
    + '          Matter.Composite.clear(world,false);\n'
    + '          engine.gravity.x=0; engine.gravity.y=1;\n'
    + setupCode.split('\n').map(l => '          ' + l).join('\n') + '\n'
    + '        }\n'
    + '        setup();\n'
    + '        Matter.Render.run(render);\n'
    + '        Matter.Runner.run(runner,engine);\n'
    + '        runner.enabled=' + autoplayJs + ';\n'
    + '        /* 控制条:播放暂停切换 + 重置(重置后暂停,老师按播放再开始) */\n'
    + '        var tg=document.getElementById("' + id + '-toggle");\n'
    + '        var rs=document.getElementById("' + id + '-reset");\n'
    + '        function refresh(){ if(tg) tg.textContent = runner.enabled ? \'⏸ 暂停\' : \'▶ 播放\'; }\n'
    + '        if(tg) tg.onclick=function(){ runner.enabled=!runner.enabled; refresh(); };\n'
    + '        if(rs) rs.onclick=function(){ setup(); runner.enabled=false; refresh(); };\n'
    + '        refresh();\n'
    + '      }catch(e){ el.innerHTML=\'<div style="padding:20px;color:#DC2626;font-size:14px;">物理场景渲染失败: \'+e.message+\'</div>\'; }\n'
    + '    });\n'
    + '  })();\n'
    + '  </' + 'script>\n'
    + '</div>'
}

// ============================================================
// 微调指令拼接
// ============================================================

/**
 * 根据场景配置生成 RefinePage 的微调指令文本
 * (范式对齐 buildMathRefineInstruction:融入要求 + 完整代码块;
 *  物理特有强调:控制条按钮是仿真操作入口,必须完整保留)
 */
export function buildPhysicsRefineInstruction(config: PhysicsRefineConfig): string {
  const { scene, positionHint } = config
  const embedCode = generatePhysicsEmbed(scene)

  const posDesc = positionHint
    ? '位置偏好: ' + positionHint + '。'
    : '请根据页面现有内容布局,在最合适的位置放置物理场景区域。'

  return '请在当前课件页面中融入一个可播放的物理仿真场景(' + scene.template.name + ')。\n\n'
    + '【融入要求】\n'
    + '1. ' + posDesc + '\n'
    + '2. 这是一个随时间演化的物理仿真(基于 Matter.js 物理引擎),画布下方带有'
    + '「▶ 播放/⏸ 暂停」和「↺ 重置」两个控制按钮,是课堂演示的核心操作入口——'
    + '这两个按钮及其 id 必须完整保留,不可删除、不可改 id、不可与其他元素合并。\n'
    + '3. 画布尺寸为 ' + scene.width + '×' + scene.height + ' 像素,请保持该尺寸不变(物理画布按此尺寸初始化,CSS 拉伸会导致画面变形)。\n'
    + '4. 保持与页面整体视觉风格协调——配色、圆角、间距与现有元素和谐统一,场景区域可加适当标题。\n'
    + '5. 不要删除或大幅改动页面现有的其他内容和布局结构。\n'
    + '6. 以下代码中的 <script> 标签和 JavaScript 逻辑必须完整保留、一字不改——'
    + '尤其是自托管库地址(matter.min.js)、setup() 构造函数与控制条绑定逻辑,改动任何字符都会导致仿真无法运行。\n'
    + '7. Matter.js 库的 script 由浏览器自动去重,同页已有其他物理场景也不冲突,保持在代码片段内即可。\n\n'
    + '【以下是需要融入的完整代码(含 Matter.js 库引用、场景构造与播放控制脚本)】\n\n'
    + embedCode
}

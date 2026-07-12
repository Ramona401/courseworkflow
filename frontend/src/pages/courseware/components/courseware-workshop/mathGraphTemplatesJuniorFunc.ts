/**
 * mathGraphTemplatesJuniorFunc.ts — 初中·函数图像模板(批次1c拆分首发;2c版式整改,2026-07-08)
 *
 * 覆盖课标:一次函数、二次函数(一般式/顶点式平移)、反比例函数(人教版八九年级)。
 * 批次2c:4个模板全部套用 makeLayout 版式规范(顶部控制面板/滑杆成列/公式读数右置/
 * 底部提示面板),构造逻辑零改动。
 * 编写规范见聚合出口 mathGraphTemplates.ts 文件头。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/mathGraphTemplatesJuniorFunc.ts
 */
import type { MathGraphTemplate } from './mathGraphUtils'
import { n, sliderAttrs, makeLayout } from './mathGraphTemplateShared'

const G = '📗 初中 · 函数图像'

export const MATH_GRAPH_TEMPLATES_JUNIOR_FUNC: MathGraphTemplate[] = [

  {
    id: 'linear',
    group: G,
    name: '一次函数 y = kx + b',
    emoji: '📈',
    desc: '拖动滑杆观察斜率 k 与截距 b 对直线的影响',
    boundingBox: [-10, 10, 10, -10],
    keepAspectRatio: false,
    params: [
      { key: 'k', label: '斜率 k', type: 'number', min: -10, max: 10, step: 0.1, defaultValue: 1, hint: '直线倾斜程度' },
      { key: 'b', label: '截距 b', type: 'number', min: -10, max: 10, step: 0.1, defaultValue: 2, hint: '与 y 轴交点' },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 10, 10, -10])
      return [
        "/* 顶部控制区:面板 + 两根滑杆(左) + 解析式读数(右) */",
        L.panel(2),
        "var sk = board.create('slider', [" + L.slider(0) + ", [-10, " + n(p.k) + ", 10]], {name:'k', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sb = board.create('slider', [" + L.slider(1) + ", [-10, " + n(p.b) + ", 10]], {name:'b', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "board.create('functiongraph', [function(x){ return sk.Value()*x + sb.Value(); }], {strokeColor:'#2563EB', strokeWidth:3});",
        "board.create('point', [0, function(){ return sb.Value(); }], {name:'截距', size:3, fixed:true, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var k = sk.Value(), b = sb.Value();",
        "  return 'y = ' + k.toFixed(1) + 'x ' + (b >= 0 ? '+ ' + b.toFixed(1) : '- ' + Math.abs(b).toFixed(1));",
        "}], {fontSize:17, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", 'k 决定倾斜方向和陡缓,b 决定与 y 轴的交点(红点)'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'quadratic',
    group: G,
    name: '二次函数 y = ax² + bx + c',
    emoji: '⛰️',
    desc: '抛物线开口、顶点与对称轴随参数联动变化',
    boundingBox: [-10, 10, 10, -10],
    keepAspectRatio: false,
    params: [
      { key: 'a', label: '二次项系数 a', type: 'number', min: -3, max: 3, step: 0.1, defaultValue: 1, hint: '开口方向与宽窄' },
      { key: 'b', label: '一次项系数 b', type: 'number', min: -8, max: 8, step: 0.1, defaultValue: 0 },
      { key: 'c', label: '常数项 c', type: 'number', min: -8, max: 8, step: 0.1, defaultValue: -2, hint: '与 y 轴交点' },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 10, 10, -10])
      return [
        "/* 顶部控制区:面板 + 三根滑杆(左) + 顶点读数(右) */",
        L.panel(3),
        "var sa = board.create('slider', [" + L.slider(0) + ", [-3, " + n(p.a) + ", 3]], {name:'a', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sbq = board.create('slider', [" + L.slider(1) + ", [-8, " + n(p.b) + ", 8]], {name:'b', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sc = board.create('slider', [" + L.slider(2) + ", [-8, " + n(p.c) + ", 8]], {name:'c', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "/* a 接近 0 时按 ±0.02 兜底,防除零 */",
        "function safeA(){ var a = sa.Value(); return Math.abs(a) < 0.02 ? (a >= 0 ? 0.02 : -0.02) : a; }",
        "board.create('functiongraph', [function(x){ return sa.Value()*x*x + sbq.Value()*x + sc.Value(); }], {strokeColor:'#2563EB', strokeWidth:3});",
        "var vx = function(){ return -sbq.Value() / (2 * safeA()); };",
        "var vy = function(){ var x = vx(); return sa.Value()*x*x + sbq.Value()*x + sc.Value(); };",
        "board.create('point', [vx, vy], {name:'顶点', size:3, fixed:true, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('line', [[vx, -10], [vx, 10]], {straightFirst:true, straightLast:true, dash:2, strokeColor:'#9CA3AF', strokeWidth:1});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  return '顶点 (' + vx().toFixed(2) + ', ' + vy().toFixed(2) + ')';",
        "}], {fontSize:15, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", 'a 定开口方向宽窄;虚线是对称轴,顶点在对称轴上'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'quad_vertex',
    group: G,
    name: '二次函数顶点式(图像平移)',
    emoji: '🚚',
    desc: 'y = a(x−h)²+k,拖 h、k 看抛物线整体平移',
    boundingBox: [-10, 10, 10, -10],
    keepAspectRatio: false,
    params: [
      { key: 'a', label: '开口系数 a', type: 'number', min: -2, max: 2, step: 0.1, defaultValue: 1 },
      { key: 'h', label: '水平平移 h', type: 'number', min: -6, max: 6, step: 0.1, defaultValue: 2, hint: '顶点横坐标(右移为正)' },
      { key: 'k', label: '竖直平移 k', type: 'number', min: -6, max: 6, step: 0.1, defaultValue: 3, hint: '顶点纵坐标(上移为正)' },
      { key: 'showRef', label: '显示 y=x² 参考虚线', type: 'boolean', defaultValue: true },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 10, 10, -10])
      return [
        "/* 顶部控制区:面板 + 三根滑杆(左) + 顶点式读数(右) */",
        L.panel(3),
        "var sav = board.create('slider', [" + L.slider(0) + ", [-2, " + n(p.a) + ", 2]], {name:'a', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sh = board.create('slider', [" + L.slider(1) + ", [-6, " + n(p.h) + ", 6]], {name:'h', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var skv = board.create('slider', [" + L.slider(2) + ", [-6, " + n(p.k) + ", 6]], {name:'k', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "function safeAv(){ var a = sav.Value(); return Math.abs(a) < 0.02 ? (a >= 0 ? 0.02 : -0.02) : a; }",
        (p.showRef ? "board.create('functiongraph', [function(x){ return x*x; }], {strokeColor:'#9CA3AF', strokeWidth:1.5, dash:2});" : "/* 参考线关闭 */"),
        "board.create('functiongraph', [function(x){ var d = x - sh.Value(); return safeAv()*d*d + skv.Value(); }], {strokeColor:'#2563EB', strokeWidth:3});",
        "board.create('point', [function(){ return sh.Value(); }, function(){ return skv.Value(); }], {name:'顶点(h,k)', size:3, fixed:true, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var a = safeAv(), h = sh.Value(), k = skv.Value();",
        "  return 'y = ' + a.toFixed(1) + '(x ' + (h >= 0 ? '- ' + h.toFixed(1) : '+ ' + Math.abs(h).toFixed(1)) + ')² ' + (k >= 0 ? '+ ' + k.toFixed(1) : '- ' + Math.abs(k).toFixed(1));",
        "}], {fontSize:15, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖 h 左右平移、拖 k 上下平移:虚线 y=x² 是平移前的\"原型\"'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'inverse',
    group: G,
    name: '反比例函数 y = k/x',
    emoji: '🌀',
    desc: '双曲线两支随 k 值变化,观察象限分布',
    boundingBox: [-10, 10, 10, -10],
    keepAspectRatio: false,
    params: [
      { key: 'k', label: '比例系数 k', type: 'number', min: -10, max: 10, step: 0.5, defaultValue: 4, hint: 'k>0 在一三象限,k<0 在二四象限' },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 10, 10, -10])
      return [
        "/* 顶部控制区:面板 + k 滑杆(左) + 解析式读数(右) */",
        L.panel(1),
        "var sk = board.create('slider', [" + L.slider(0) + ", [-10, " + n(p.k) + ", 10]], {name:'k', " + sliderAttrs('#7C3AED', 0.5) + "});",
        "/* 双曲线分左右两支绘制,避开 x=0 间断点 */",
        "board.create('functiongraph', [function(x){ return sk.Value()/x; }, 0.05, 12], {strokeColor:'#2563EB', strokeWidth:3});",
        "board.create('functiongraph', [function(x){ return sk.Value()/x; }, -12, -0.05], {strokeColor:'#2563EB', strokeWidth:3});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  return 'y = ' + sk.Value().toFixed(1) + ' / x';",
        "}], {fontSize:17, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '把 k 从正拖到负:两支曲线从一三象限跳到二四象限'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },
]

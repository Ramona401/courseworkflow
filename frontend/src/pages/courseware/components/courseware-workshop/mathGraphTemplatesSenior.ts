/**
 * mathGraphTemplatesSenior.ts — 高中模板(批次1c拆分首发;2d版式整改,2026-07-08)
 *
 * 覆盖课标:指数/对数/绝对值函数、三角函数变换(必修一/必修四),
 * 单位圆与三角、椭圆定义(必修/选择性必修)。
 * 批次2d:6个模板全部套用 makeLayout 版式规范(顶部控制面板/滑杆成列/读数右置/
 * 底部提示面板),构造逻辑零改动;原缺底部提示的模板补齐同风格提示条。
 * 编写规范见聚合出口 mathGraphTemplates.ts 文件头。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/mathGraphTemplatesSenior.ts
 */
import type { MathGraphTemplate } from './mathGraphUtils'
import { n, sliderAttrs, makeLayout } from './mathGraphTemplateShared'

const GF = '📘 高中 · 函数'
const GG = '📘 高中 · 三角与圆锥曲线'

export const MATH_GRAPH_TEMPLATES_SENIOR: MathGraphTemplate[] = [

  {
    id: 'exponential',
    group: GF,
    name: '指数函数 y = aˣ',
    emoji: '🚀',
    desc: '拖动底数 a 看增长/衰减,恒过定点 (0,1)',
    boundingBox: [-6, 9, 6, -3],
    keepAspectRatio: false,
    params: [
      { key: 'a', label: '底数 a', type: 'number', min: 0.2, max: 4, step: 0.05, defaultValue: 2, hint: 'a>1 递增,0<a<1 递减' },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-6, 9, 6, -3])
      return [
        "/* 顶部控制区:面板 + 底数滑杆(左) + 解析式读数(右) */",
        L.panel(1),
        "var sa = board.create('slider', [" + L.slider(0) + ", [0.2, " + n(p.a) + ", 4]], {name:'a', " + sliderAttrs('#7C3AED', 0.05) + "});",
        "board.create('functiongraph', [function(x){ return Math.pow(sa.Value(), x); }], {strokeColor:'#2563EB', strokeWidth:3});",
        "/* 定点 (0,1):无论 a 取何值都经过 */",
        "board.create('point', [0, 1], {name:'(0,1) 定点', size:3, fixed:true, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var a = sa.Value();",
        "  return 'y = ' + a.toFixed(2) + 'ˣ  (' + (a > 1 ? '递增' : (a < 1 ? '递减' : '恒为1')) + ')';",
        "}], {fontSize:15, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '把 a 从大于1拖到小于1:曲线从递增翻转为递减,但始终经过红色定点'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'logarithm',
    group: GF,
    name: '对数函数 y = logₐx',
    emoji: '📉',
    desc: '拖动底数 a 看图像变化,恒过定点 (1,0)',
    boundingBox: [-3, 6, 12, -6],
    keepAspectRatio: false,
    params: [
      { key: 'a', label: '底数 a', type: 'number', min: 0.2, max: 4, step: 0.05, defaultValue: 2, hint: 'a>1 递增,0<a<1 递减(a≠1)' },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-3, 6, 12, -6])
      return [
        "/* 顶部控制区:面板 + 底数滑杆(左) + 解析式读数(右) */",
        L.panel(1),
        "var sa = board.create('slider', [" + L.slider(0) + ", [0.2, " + n(p.a) + ", 4]], {name:'a', " + sliderAttrs('#7C3AED', 0.05) + "});",
        "/* a 贴近 1 时兜底偏移,防 ln(a)=0 除零 */",
        "function safeLnA(){ var a = sa.Value(); if (Math.abs(a - 1) < 0.03) a = a >= 1 ? 1.03 : 0.97; return Math.log(a); }",
        "board.create('functiongraph', [function(x){ return Math.log(x) / safeLnA(); }, 0.02, 13], {strokeColor:'#2563EB', strokeWidth:3});",
        "/* 定点 (1,0):无论 a 取何值都经过 */",
        "board.create('point', [1, 0], {name:'(1,0) 定点', size:3, fixed:true, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var a = sa.Value();",
        "  return 'y = log_' + a.toFixed(2) + '(x)  (' + (a > 1 ? '递增' : '递减') + ')';",
        "}], {fontSize:15, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '对数函数只在 x>0 有定义;与指数函数互为反函数'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'absolute',
    group: GF,
    name: '绝对值函数 y = a|x−h|+k',
    emoji: '✅',
    desc: 'V 形图像的开口与顶点平移,理解绝对值几何意义',
    boundingBox: [-10, 10, 10, -10],
    keepAspectRatio: false,
    params: [
      { key: 'a', label: '开口系数 a', type: 'number', min: -3, max: 3, step: 0.1, defaultValue: 1, hint: 'a>0 开口向上,a<0 向下' },
      { key: 'h', label: '顶点横坐标 h', type: 'number', min: -6, max: 6, step: 0.1, defaultValue: 0 },
      { key: 'k', label: '顶点纵坐标 k', type: 'number', min: -6, max: 6, step: 0.1, defaultValue: -2 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 10, 10, -10])
      return [
        "/* 顶部控制区:面板 + 三根滑杆(左) + 解析式读数(右) */",
        L.panel(3),
        "var sav = board.create('slider', [" + L.slider(0) + ", [-3, " + n(p.a) + ", 3]], {name:'a', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sh = board.create('slider', [" + L.slider(1) + ", [-6, " + n(p.h) + ", 6]], {name:'h', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var skv = board.create('slider', [" + L.slider(2) + ", [-6, " + n(p.k) + ", 6]], {name:'k', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "board.create('functiongraph', [function(x){ return sav.Value()*Math.abs(x - sh.Value()) + skv.Value(); }], {strokeColor:'#2563EB', strokeWidth:3});",
        "board.create('point', [function(){ return sh.Value(); }, function(){ return skv.Value(); }], {name:'顶点(h,k)', size:3, fixed:true, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var a = sav.Value(), h = sh.Value(), k = skv.Value();",
        "  return 'y = ' + a.toFixed(1) + '|x ' + (h >= 0 ? '- ' + h.toFixed(1) : '+ ' + Math.abs(h).toFixed(1)) + '| ' + (k >= 0 ? '+ ' + k.toFixed(1) : '- ' + Math.abs(k).toFixed(1));",
        "}], {fontSize:15, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", 'V 形顶点在 (h,k);把 a 拖过 0:开口翻转'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'trig',
    group: GF,
    name: '三角函数 y = A·sin(Bx + C)',
    emoji: '🌊',
    desc: '振幅、周期、相位三参数联动,理解正弦波变换',
    boundingBox: [-9.9, 3.8, 9.9, -3.8],
    keepAspectRatio: false,
    params: [
      { key: 'A', label: '振幅 A', type: 'number', min: 0.2, max: 3, step: 0.1, defaultValue: 1 },
      { key: 'B', label: '角频率 B', type: 'number', min: 0.2, max: 4, step: 0.1, defaultValue: 1, hint: '周期 T = 2π/B' },
      { key: 'C', label: '初相 C', type: 'number', min: -3.2, max: 3.2, step: 0.1, defaultValue: 0 },
      { key: 'showRef', label: '显示 y=sin(x) 参考虚线', type: 'boolean', defaultValue: true },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-9.9, 3.8, 9.9, -3.8])
      return [
        "/* 顶部控制区:面板 + 三根滑杆(左) + 解析式读数(右);波形穿过面板属正常分层 */",
        L.panel(3),
        "var sA = board.create('slider', [" + L.slider(0) + ", [0.2, " + n(p.A) + ", 3]], {name:'A', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sB = board.create('slider', [" + L.slider(1) + ", [0.2, " + n(p.B) + ", 4]], {name:'B', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sC = board.create('slider', [" + L.slider(2) + ", [-3.2, " + n(p.C) + ", 3.2]], {name:'C', " + sliderAttrs('#7C3AED', 0.1) + "});",
        (p.showRef ? "board.create('functiongraph', [function(x){ return Math.sin(x); }], {strokeColor:'#9CA3AF', strokeWidth:1.5, dash:2});" : "/* 参考线关闭 */"),
        "board.create('functiongraph', [function(x){ return sA.Value() * Math.sin(sB.Value()*x + sC.Value()); }], {strokeColor:'#2563EB', strokeWidth:3});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  return 'y = ' + sA.Value().toFixed(1) + '·sin(' + sB.Value().toFixed(1) + 'x ' + (sC.Value() >= 0 ? '+ ' : '- ') + Math.abs(sC.Value()).toFixed(1) + ')';",
        "}], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "board.create('text', [" + L.midX + ", " + L.topY(1) + ", function(){",
        "  return '周期 T = 2π/B = ' + (2*Math.PI/sB.Value()).toFixed(2);",
        "}], {fontSize:13, strokeColor:'#6B7280'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", 'A 拉伸上下,B 压缩左右,C 左右平移(灰虚线是 y=sin x 原型)'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'unit_circle',
    group: GG,
    name: '单位圆与三角函数',
    emoji: '🧭',
    desc: '拖动圆上的点,实时联动 sin/cos 值与角度',
    boundingBox: [-1.7, 1.7, 1.7, -1.7],
    keepAspectRatio: true,
    params: [
      { key: 'angle', label: '初始角度(度)', type: 'number', min: 0, max: 360, step: 5, defaultValue: 45 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-1.7, 1.7, 1.7, -1.7])
      const rad = ((p.angle as number) * Math.PI) / 180
      return [
        "var O = board.create('point', [0, 0], {name:'O', fixed:true, size:2, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:12}});",
        "var circ = board.create('circle', [O, 1], {strokeColor:'#2563EB', strokeWidth:2});",
        "var refX = board.create('point', [1, 0], {visible:false});",
        "/* P 为圆上滑动点,可拖动改变角度 */",
        "var P = board.create('glider', [" + n(Math.cos(rad)) + ", " + n(Math.sin(rad)) + ", circ], {name:'P(拖我)', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:12}});",
        "board.create('segment', [O, P], {strokeColor:'#1F2937', strokeWidth:2});",
        "/* sin:P 到 x 轴的垂线段(红);cos:x 轴上的投影段(绿) */",
        "var Px = board.create('point', [function(){ return P.X(); }, 0], {visible:false});",
        "board.create('segment', [P, Px], {strokeColor:'#DC2626', strokeWidth:3});",
        "board.create('segment', [O, Px], {strokeColor:'#059669', strokeWidth:3});",
        "board.create('angle', [refX, O, P], {radius:0.3, name:'θ', strokeColor:'#F59E0B', fillColor:'#FDE68A', fillOpacity:0.5, label:{fontSize:13}});",
        "/* 顶部读数区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){",
        "  var t = Math.atan2(P.Y(), P.X()); if (t < 0) t += 2*Math.PI;",
        "  return 'θ = ' + (t*180/Math.PI).toFixed(0) + '°   sinθ = ' + P.Y().toFixed(2) + '   cosθ = ' + P.X().toFixed(2);",
        "}], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '红段=sinθ 绿段=cosθ:拖 P 转一整圈看正负号变化'], {fontSize:12, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'ellipse_def',
    group: GG,
    name: '椭圆的定义',
    emoji: '🥚',
    desc: '拖动椭圆上的点,验证到两焦点距离之和恒为 2a',
    boundingBox: [-8, 6, 8, -6],
    keepAspectRatio: true,
    params: [
      { key: 'a', label: '长半轴 a', type: 'number', min: 2.5, max: 6, step: 0.1, defaultValue: 4 },
      { key: 'c', label: '半焦距 c', type: 'number', min: 0.5, max: 2.2, step: 0.1, defaultValue: 2, hint: 'c 越大越扁(c < a 恒成立)' },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-8, 6, 8, -6])
      return [
        "/* 顶部控制区:面板 + 两根滑杆(左) + 定义式读数(右);椭圆穿过面板属正常分层 */",
        L.panel(2),
        "var sa3 = board.create('slider', [" + L.slider(0) + ", [2.5, " + n(p.a) + ", 6]], {name:'a', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sc3 = board.create('slider', [" + L.slider(1) + ", [0.5, " + n(p.c) + ", 2.2]], {name:'c', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "/* 两焦点关于原点对称 */",
        "var F1 = board.create('point', [function(){ return -sc3.Value(); }, 0], {name:'F₁', size:3, fixed:true, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}});",
        "var F2 = board.create('point', [function(){ return sc3.Value(); }, 0], {name:'F₂', size:3, fixed:true, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}});",
        "/* 长轴端点 (a,0) 必在椭圆上,用它定义椭圆 */",
        "var Pa = board.create('point', [function(){ return sa3.Value(); }, 0], {visible:false});",
        "var ell = board.create('ellipse', [F1, F2, Pa], {strokeColor:'#2563EB', strokeWidth:2.5});",
        "/* P 为椭圆上滑动点 */",
        "var P = board.create('glider', [0, -3, ell], {name:'P(拖我)', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('segment', [P, F1], {strokeColor:'#059669', strokeWidth:2});",
        "board.create('segment', [P, F2], {strokeColor:'#7C3AED', strokeWidth:2});",
        "function dist(U, V){ var dx = U.X()-V.X(), dy = U.Y()-V.Y(); return Math.sqrt(dx*dx+dy*dy); }",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var d1 = dist(P, F1), d2 = dist(P, F2);",
        "  return '|PF₁|+|PF₂| = ' + d1.toFixed(2) + ' + ' + d2.toFixed(2) + ' = ' + (d1+d2).toFixed(2) + ' = 2a';",
        "}], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖动 P 沿椭圆移动:两段距离此消彼长,总和不变'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },
]

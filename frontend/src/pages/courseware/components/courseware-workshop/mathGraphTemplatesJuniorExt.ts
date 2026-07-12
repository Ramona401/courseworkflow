/**
 * mathGraphTemplatesJuniorExt.ts — 初中扩充模板(批次1c-3首发;批次2a版式整改,2026-07-08)
 *
 * 分组名与 JuniorFunc/JuniorGeo 完全一致,聚合后自动并入同组展示。
 * 覆盖课标:图形的平移与旋转、三角形/多边形内角和、位似、垂径定理(几何组);
 *          动点问题(中考压轴头号题型,几何+函数双区联动)、
 *          二次函数与一元二次方程(判别式与根)(函数组)。
 * 批次2a:7个模板全部套用 makeLayout 版式规范,构造逻辑与原版完全一致,仅排版调整。
 * 编写规范见聚合出口 mathGraphTemplates.ts 文件头。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/mathGraphTemplatesJuniorExt.ts
 */
import type { MathGraphTemplate } from './mathGraphUtils'
import { n, sliderAttrs, makeLayout } from './mathGraphTemplateShared'

const GF = '📗 初中 · 函数图像'
const GG = '📗 初中 · 几何作图'

export const MATH_GRAPH_TEMPLATES_JUNIOR_EXT: MathGraphTemplate[] = [

  // ---------- 并入:初中·几何作图 ----------

  {
    id: 'translate_rotate',
    group: GG,
    name: '图形的平移与旋转',
    emoji: '🔄',
    desc: '同一个图形,平移像与旋转像并排对比,参数全可调',
    boundingBox: [-10, 10, 10, -10],
    keepAspectRatio: true,
    params: [
      { key: 'dx', label: '平移:水平距离', type: 'number', min: -4, max: 10, step: 0.5, defaultValue: 7 },
      { key: 'dy', label: '平移:竖直距离', type: 'number', min: -4, max: 8, step: 0.5, defaultValue: 5 },
      { key: 'theta', label: '旋转:角度(度)', type: 'number', min: -180, max: 180, step: 5, defaultValue: 90 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 10, 10, -10])
      return [
        "/* 顶部控制区:面板 + 三根滑杆(左) + 图例(右) */",
        L.panel(3),
        "var sdx = board.create('slider', [" + L.slider(0) + ", [-4, " + n(p.dx) + ", 10]], {name:'平移x', " + sliderAttrs('#059669', 0.5) + "});",
        "var sdy = board.create('slider', [" + L.slider(1) + ", [-4, " + n(p.dy) + ", 8]], {name:'平移y', " + sliderAttrs('#059669', 0.5) + "});",
        "var sth = board.create('slider', [" + L.slider(2) + ", [-180, " + n(p.theta) + ", 180]], {name:'旋转°', " + sliderAttrs('#EA580C', 5) + "});",
        "function rc(){ return sth.Value() * Math.PI / 180; }",
        "/* 原图形(蓝,顶点可拖) */",
        "var A = board.create('point', [-7, -3], {name:'A', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var B = board.create('point', [-3, -3], {name:'B', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var Cq = board.create('point', [-5, -0.5], {name:'C', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "board.create('polygon', [A, B, Cq], {fillColor:'#DBEAFE', fillOpacity:0.5, borders:{strokeColor:'#2563EB', strokeWidth:2}});",
        "/* 旋转中心(橙,可拖) */",
        "var O = board.create('point', [1, -3], {name:'旋转中心O', size:4, fillColor:'#EA580C', strokeColor:'#EA580C', label:{fontSize:13}});",
        "/* 平移像(绿):每个顶点加 (dx,dy);工厂函数生成坐标闭包 */",
        "function trX(P){ return function(){ return P.X() + sdx.Value(); }; }",
        "function trY(P){ return function(){ return P.Y() + sdy.Value(); }; }",
        "board.create('polygon', [[trX(A), trY(A)], [trX(B), trY(B)], [trX(Cq), trY(Cq)]], {fillColor:'#D1FAE5', fillOpacity:0.5, borders:{strokeColor:'#059669', strokeWidth:2, dash:0}, vertices:{visible:false}});",
        "/* 旋转像(橙):绕 O 转 θ */",
        "function roX(P){ return function(){ return O.X() + (P.X()-O.X())*Math.cos(rc()) - (P.Y()-O.Y())*Math.sin(rc()); }; }",
        "function roY(P){ return function(){ return O.Y() + (P.X()-O.X())*Math.sin(rc()) + (P.Y()-O.Y())*Math.cos(rc()); }; }",
        "board.create('polygon', [[roX(A), roY(A)], [roX(B), roY(B)], [roX(Cq), roY(Cq)]], {fillColor:'#FED7AA', fillOpacity:0.5, borders:{strokeColor:'#EA580C', strokeWidth:2}, vertices:{visible:false}});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", '绿 = 平移像   橙 = 旋转像'], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "board.create('text', [" + L.midX + ", " + L.topY(1) + ", '平移:只挪位置不转向'], {fontSize:12, strokeColor:'#6B7280'});",
        "board.create('text', [" + L.midX + ", " + L.topY(2) + ", '旋转:绕中心 O 转 θ 角'], {fontSize:12, strokeColor:'#6B7280'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖 A/B/C 改原图,拖 O 改旋转中心;两种变换都不改变形状大小'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'triangle_angle_sum',
    group: GG,
    name: '三角形内角和',
    emoji: '🧩',
    desc: '随意拖动三个顶点,三个内角怎么变,和都是180°',
    boundingBox: [-8, 8, 8, -8],
    keepAspectRatio: true,
    params: [],
    buildConstruction: () => {
      const L = makeLayout([-8, 8, 8, -8])
      return [
        "var A = board.create('point', [-4, -3.5], {name:'A', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:14}});",
        "var B = board.create('point', [5, -3.5], {name:'B', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:14}});",
        "var Cq = board.create('point', [0, 3.5], {name:'C', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:14}});",
        "board.create('polygon', [A, B, Cq], {fillColor:'#DBEAFE', fillOpacity:0.35, borders:{strokeColor:'#2563EB', strokeWidth:2}});",
        "/* 三个内角弧(nonreflexangle 保证始终取小于180°的内角,拖动不翻转) */",
        "board.create('nonreflexangle', [B, A, Cq], {radius:1.0, name:'α', strokeColor:'#DC2626', fillColor:'#FECACA', fillOpacity:0.6, label:{fontSize:14}});",
        "board.create('nonreflexangle', [Cq, B, A], {radius:1.0, name:'β', strokeColor:'#059669', fillColor:'#A7F3D0', fillOpacity:0.6, label:{fontSize:14}});",
        "board.create('nonreflexangle', [A, Cq, B], {radius:1.0, name:'γ', strokeColor:'#F59E0B', fillColor:'#FDE68A', fillOpacity:0.6, label:{fontSize:14}});",
        "function angDeg(P, V, Q){",
        "  var a1 = Math.atan2(P.Y()-V.Y(), P.X()-V.X()), a2 = Math.atan2(Q.Y()-V.Y(), Q.X()-V.X());",
        "  var d = Math.abs(a1 - a2) * 180 / Math.PI; if (d > 180) d = 360 - d; return d;",
        "}",
        "/* 顶部读数区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){",
        "  var a = angDeg(B, A, Cq), b = angDeg(Cq, B, A), c = angDeg(A, Cq, B);",
        "  return 'α=' + a.toFixed(1) + '°  β=' + b.toFixed(1) + '°  γ=' + c.toFixed(1) + '°   α+β+γ = ' + (a+b+c).toFixed(1) + '°';",
        "}], {fontSize:15, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖动任意顶点:三个角各自变化,总和恒为 180°'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'polygon_angle_sum',
    group: GG,
    name: '多边形内角和',
    emoji: '⬡',
    desc: '拖动边数滑杆 3~8,内角和 = (n−2)×180°',
    boundingBox: [-8, 8, 8, -8],
    keepAspectRatio: true,
    params: [
      { key: 'nSides', label: '边数 n', type: 'number', min: 3, max: 8, step: 1, defaultValue: 5 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-8, 8, 8, -8])
      return [
        "/* 顶部控制区:面板 + 边数滑杆(左) + 公式读数(右起第二行满宽) */",
        L.panel(2),
        "var sn = board.create('slider', [" + L.slider(0) + ", [3, " + n(p.nSides) + ", 8]], {name:'边数n', " + sliderAttrs('#7C3AED', 1) + "});",
        "function nn(){ return Math.round(sn.Value()); }",
        "var R = 3.6;",
        "/* 预建8个顶点:前 n 个落在正n边形顶点位,多余的收拢到第一个顶点(重复顶点不影响多边形绘制) */",
        "var verts = [];",
        "for (var i = 0; i < 8; i++) {",
        "  (function(idx){",
        "    var pt = board.create('point', [",
        "      function(){ var m = nn(); var a = (idx < m ? idx : 0); return R*Math.cos(2*Math.PI*a/m + Math.PI/2); },",
        "      function(){ var m = nn(); var a = (idx < m ? idx : 0); return R*Math.sin(2*Math.PI*a/m + Math.PI/2) - 1.6; }",
        "    ], {visible:false});",
        "    verts.push(pt);",
        "  })(i);",
        "}",
        "board.create('polygon', verts, {fillColor:'#DDD6FE', fillOpacity:0.45, borders:{strokeColor:'#7C3AED', strokeWidth:2.5}, vertices:{visible:false}});",
        "/* 从顶点0出发的对角线,把多边形分成 n-2 个三角形(内角和公式的由来) */",
        "for (var j = 2; j < 7; j++) {",
        "  (function(idx){",
        "    board.create('segment', [verts[0], verts[idx]], {",
        "      visible: function(){ return idx < nn() - 1; },",
        "      strokeColor:'#F59E0B', strokeWidth:1.5, dash:2",
        "    });",
        "  })(j);",
        "}",
        "board.create('text', [" + L.leftX + ", " + L.topY(1) + ", function(){",
        "  var m = nn();",
        "  return m + ' 边形被对角线分成 ' + (m-2) + ' 个三角形 → 内角和 = (' + m + '−2) × 180° = ' + ((m-2)*180) + '°';",
        "}], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '正多边形每个内角 = 内角和 ÷ n(拖动滑杆观察)'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'similarity',
    group: GG,
    name: '相似与位似',
    emoji: '🔍',
    desc: '拖动位似比 k,图形以中心 O 放大缩小,对应边成比例',
    boundingBox: [-10, 10, 10, -10],
    keepAspectRatio: true,
    params: [
      { key: 'k', label: '位似比 k', type: 'number', min: 0.3, max: 2.5, step: 0.1, defaultValue: 1.8, hint: 'k>1 放大,k<1 缩小' },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 10, 10, -10])
      return [
        "/* 顶部控制区:面板 + 位似比滑杆(左) + 比值读数(右) */",
        L.panel(1),
        "var sk = board.create('slider', [" + L.slider(0) + ", [0.3, " + n(p.k) + ", 2.5]], {name:'k', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "/* 位似中心(可拖) */",
        "var O = board.create('point', [-7, -6], {name:'位似中心O', size:4, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}});",
        "/* 原三角形(蓝,可拖) */",
        "var A = board.create('point', [-3, -2], {name:'A', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var B = board.create('point', [0, -4], {name:'B', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var Cq = board.create('point', [-1, 0.5], {name:'C', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "board.create('polygon', [A, B, Cq], {fillColor:'#DBEAFE', fillOpacity:0.5, borders:{strokeColor:'#2563EB', strokeWidth:2}});",
        "/* 位似像:A′ = O + k·(A−O) */",
        "function hX(P){ return function(){ return O.X() + sk.Value()*(P.X()-O.X()); }; }",
        "function hY(P){ return function(){ return O.Y() + sk.Value()*(P.Y()-O.Y()); }; }",
        "var A2 = board.create('point', [hX(A), hY(A)], {name:'A′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "var B2 = board.create('point', [hX(B), hY(B)], {name:'B′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "var C2 = board.create('point', [hX(Cq), hY(Cq)], {name:'C′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "board.create('polygon', [A2, B2, C2], {fillColor:'#D1FAE5', fillOpacity:0.5, borders:{strokeColor:'#059669', strokeWidth:2}});",
        "/* 位似射线(过中心与对应点) */",
        "board.create('line', [O, A], {straightFirst:false, strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
        "board.create('line', [O, B], {straightFirst:false, strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
        "board.create('line', [O, Cq], {straightFirst:false, strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
        "function dist(U, V){ var dx = U.X()-V.X(), dy = U.Y()-V.Y(); return Math.sqrt(dx*dx+dy*dy); }",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  return 'A′B′/AB = ' + (dist(A2,B2)/dist(A,B)).toFixed(2) + ' = k';",
        "}], {fontSize:15, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '对应点都在过中心 O 的射线上,对应边的比恒等于 k'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'chord_theorem',
    group: GG,
    name: '垂径定理',
    emoji: '🌉',
    desc: '拖动弦的端点,圆心到弦的垂线始终平分弦',
    boundingBox: [-7, 7, 7, -7],
    keepAspectRatio: true,
    params: [
      { key: 'r', label: '圆半径 r', type: 'number', min: 2.5, max: 5, step: 0.5, defaultValue: 4 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-7, 7, 7, -7])
      const r = p.r as number
      return [
        "var O = board.create('point', [0, -0.5], {name:'O', fixed:true, size:3, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:13}});",
        "var circ = board.create('circle', [O, " + n(r) + "], {strokeColor:'#2563EB', strokeWidth:2});",
        "/* 弦 AB:两端点都是圆上滑动点 */",
        "var A = board.create('glider', [" + n(r * Math.cos(2.6)) + ", " + n(r * Math.sin(2.6) - 0.5) + ", circ], {name:'A(拖我)', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "var B = board.create('glider', [" + n(r * Math.cos(0.5)) + ", " + n(r * Math.sin(0.5) - 0.5) + ", circ], {name:'B(拖我)', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "var chord = board.create('segment', [A, B], {strokeColor:'#DC2626', strokeWidth:2.5});",
        "/* 圆心向弦作垂线,垂足 M */",
        "var perp = board.create('perpendicularsegment', [chord, O], {strokeColor:'#059669', strokeWidth:2, dash:2});",
        "var M = board.create('intersection', [perp, chord, 0], {name:'垂足M', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "/* 直角标记 */",
        "board.create('nonreflexangle', [O, M, B], {radius:0.4, name:'', strokeColor:'#059669', fillColor:'#A7F3D0', fillOpacity:0.5, type:'square'});",
        "function dist(U, V){ var dx = U.X()-V.X(), dy = U.Y()-V.Y(); return Math.sqrt(dx*dx+dy*dy); }",
        "/* 顶部读数区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){",
        "  return 'AM = ' + dist(A, M).toFixed(2) + '   MB = ' + dist(M, B).toFixed(2) + '   → 垂足 M 恰是弦的中点';",
        "}], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '垂直于弦的直径(半径)平分这条弦——拖 A、B 验证'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  // ---------- 并入:初中·函数图像 ----------

  {
    id: 'moving_point',
    group: GF,
    name: '动点问题(几何↔函数联动)',
    emoji: '🏃',
    desc: '点 P 沿矩形边运动,右侧同步画出三角形面积随时间的图像',
    boundingBox: [-1.5, 11, 21, -2],
    keepAspectRatio: true,
    params: [
      { key: 'speed', label: '播放位置(时间 t)', type: 'number', min: 0, max: 13, step: 0.1, defaultValue: 3, hint: 'P 从 B 出发,沿 B→C→D→A 匀速运动' },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-1.5, 11, 21, -2])
      return [
        "/* 顶部控制区:面板 + 时间滑杆(左) + 读数(右) */",
        L.panel(1),
        "var st = board.create('slider', [" + L.slider(0) + ", [0, " + n(p.speed) + ", 13]], {name:'时间t(拖我)', " + sliderAttrs('#DC2626', 0.05) + "});",
        "/* 左区:矩形 ABCD,AB=5,BC=4;P 沿 B→C(4s)→D(5s)→A(4s) 共13s */",
        "board.create('polygon', [[0,0],[5,0],[5,4],[0,4]], {fillOpacity:0, borders:{strokeColor:'#6B7280', strokeWidth:2}, vertices:{visible:false}});",
        "board.create('text', [-0.5, 0.35, 'A'], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "board.create('text', [5.2, 0.35, 'B'], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "board.create('text', [5.2, 4.3, 'C'], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "board.create('text', [-0.5, 4.3, 'D'], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "function tv(){ return st.Value(); }",
        "function px(){ var t = tv(); if (t <= 4) return 5; if (t <= 9) return 5 - (t - 4); return 0; }",
        "function py(){ var t = tv(); if (t <= 4) return t; if (t <= 9) return 4; return 4 - (t - 9); }",
        "/* 面积函数:S△ABP,AB 为底(长5),高 = P 到 AB 的距离 */",
        "function areaAt(t){ if (t <= 4) return 2.5*t; if (t <= 9) return 10; return 10 - 2.5*(t - 9); }",
        "var P = board.create('point', [px, py], {name:'P', size:5, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:14}});",
        "/* 动态三角形 ABP */",
        "board.create('polygon', [[0,0],[5,0],P], {fillColor:'#FDE68A', fillOpacity:0.55, borders:{strokeColor:'#B45309', strokeWidth:2}, vertices:{visible:false}});",
        "/* 右区:S-t 图像(横轴 x=7+t,纵轴 y=S×0.8 缩放) */",
        "board.create('segment', [[7, 0], [20.6, 0]], {strokeColor:'#1F2937', strokeWidth:2, lastArrow:true, fixed:true});",
        "board.create('segment', [[7, 0], [7, 8.8]], {strokeColor:'#1F2937', strokeWidth:2, lastArrow:true, fixed:true});",
        "board.create('text', [20.3, 0.4, 't'], {fontSize:14, strokeColor:'#1F2937'});",
        "board.create('text', [6.2, 8.5, 'S'], {fontSize:14, strokeColor:'#1F2937'});",
        "/* 完整函数图像(灰虚线预览全程) */",
        "board.create('functiongraph', [function(x){ return areaAt(x - 7) * 0.8; }, 7, 20], {strokeColor:'#9CA3AF', strokeWidth:1.5, dash:2});",
        "/* 已走过部分(蓝实线):终点随 t 截断 */",
        "board.create('functiongraph', [function(x){ return areaAt(x - 7) * 0.8; }, 7, function(){ return 7 + tv(); }], {strokeColor:'#2563EB', strokeWidth:3});",
        "/* 图像上的同步动点 */",
        "board.create('point', [function(){ return 7 + tv(); }, function(){ return areaAt(tv()) * 0.8; }], {name:'', size:4, fillColor:'#DC2626', strokeColor:'#DC2626'});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  return 't = ' + tv().toFixed(1) + '   S△ABP = ' + areaAt(tv()).toFixed(1);",
        "}], {fontSize:15, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", 'P 沿 B→C→D→A 运动:面积先增大、再不变、后减小(先斜升、平台、斜降)'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'quad_roots',
    group: GF,
    name: '二次函数与一元二次方程',
    emoji: '❎',
    desc: '拖参数看判别式 Δ 与抛物线和 x 轴交点个数的关系',
    boundingBox: [-10, 10, 10, -10],
    keepAspectRatio: false,
    params: [
      { key: 'a', label: '二次项系数 a', type: 'number', min: -3, max: 3, step: 0.1, defaultValue: 1 },
      { key: 'b', label: '一次项系数 b', type: 'number', min: -8, max: 8, step: 0.1, defaultValue: 2 },
      { key: 'c', label: '常数项 c', type: 'number', min: -8, max: 8, step: 0.1, defaultValue: -3 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 10, 10, -10])
      return [
        "/* 顶部控制区:面板 + 三根滑杆(左) + Δ读数(右) */",
        L.panel(3),
        "var sa = board.create('slider', [" + L.slider(0) + ", [-3, " + n(p.a) + ", 3]], {name:'a', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sbq = board.create('slider', [" + L.slider(1) + ", [-8, " + n(p.b) + ", 8]], {name:'b', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sc = board.create('slider', [" + L.slider(2) + ", [-8, " + n(p.c) + ", 8]], {name:'c', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "function safeA(){ var a = sa.Value(); return Math.abs(a) < 0.02 ? (a >= 0 ? 0.02 : -0.02) : a; }",
        "function disc(){ return sbq.Value()*sbq.Value() - 4*sa.Value()*sc.Value(); }",
        "board.create('functiongraph', [function(x){ return sa.Value()*x*x + sbq.Value()*x + sc.Value(); }], {strokeColor:'#2563EB', strokeWidth:3});",
        "/* 两个根(与 x 轴交点):仅 Δ≥0 时可见 */",
        "board.create('point', [function(){ return (-sbq.Value() - Math.sqrt(Math.max(disc(), 0))) / (2*safeA()); }, 0], {",
        "  name:'x₁', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', fixed:true, label:{fontSize:13},",
        "  visible: function(){ return disc() >= 0; }",
        "});",
        "board.create('point', [function(){ return (-sbq.Value() + Math.sqrt(Math.max(disc(), 0))) / (2*safeA()); }, 0], {",
        "  name:'x₂', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', fixed:true, label:{fontSize:13},",
        "  visible: function(){ return disc() > 0; }",
        "});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var d = disc();",
        "  var s = d > 0.0001 ? '两个不等实根' : (d < -0.0001 ? '无实根(不相交)' : '两个相等实根(相切)');",
        "  return 'Δ = b²−4ac = ' + d.toFixed(2) + ' → ' + s;",
        "}], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '红点 = 方程 ax²+bx+c=0 的根;拖 c 让抛物线升降,观察 Δ 的正负切换'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },
]

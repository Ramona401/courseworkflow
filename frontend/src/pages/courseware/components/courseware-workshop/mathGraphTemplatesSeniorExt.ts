/**
 * mathGraphTemplatesSeniorExt.ts — 高中扩充模板(批次1c-4首发;2d版式整改,2026-07-08)
 *
 * 覆盖课标(普通高中数学课程标准):
 *   三角与圆锥曲线组(并入既有组):双曲线的定义 / 抛物线的定义 / 正弦定理
 *   向量与规划组(并入既有函数组):向量加法 / 线性规划可行域
 *   微积分初步组(新组):导数与切线(割线逼近)/ 定积分黎曼和
 * 批次2d:7个模板全部套用 makeLayout 版式规范,构造逻辑零改动。
 * 编写规范见聚合出口 mathGraphTemplates.ts 文件头。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/mathGraphTemplatesSeniorExt.ts
 */
import type { MathGraphTemplate } from './mathGraphUtils'
import { n, sliderAttrs, makeLayout } from './mathGraphTemplateShared'

const GF = '📘 高中 · 函数'
const GG = '📘 高中 · 三角与圆锥曲线'
const GC = '📘 高中 · 微积分初步'

export const MATH_GRAPH_TEMPLATES_SENIOR_EXT: MathGraphTemplate[] = [

  // ---------- 并入:三角与圆锥曲线 ----------

  {
    id: 'hyperbola_def',
    group: GG,
    name: '双曲线的定义',
    emoji: '🦋',
    desc: '拖动曲线上的点,验证到两焦点距离之差的绝对值恒为 2a',
    boundingBox: [-9, 7, 9, -7],
    keepAspectRatio: true,
    params: [
      { key: 'a', label: '实半轴 a', type: 'number', min: 1, max: 3, step: 0.1, defaultValue: 2 },
      { key: 'c', label: '半焦距 c', type: 'number', min: 3.2, max: 5, step: 0.1, defaultValue: 4, hint: 'c > a 恒成立' },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-9, 7, 9, -7])
      return [
        "/* 顶部控制区:面板 + 两根滑杆(左) + 定义式读数(右) */",
        L.panel(2),
        "var sa4 = board.create('slider', [" + L.slider(0) + ", [1, " + n(p.a) + ", 3]], {name:'a', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sc4 = board.create('slider', [" + L.slider(1) + ", [3.2, " + n(p.c) + ", 5]], {name:'c', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var F1 = board.create('point', [function(){ return -sc4.Value(); }, 0], {name:'F₁', size:3, fixed:true, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}});",
        "var F2 = board.create('point', [function(){ return sc4.Value(); }, 0], {name:'F₂', size:3, fixed:true, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}});",
        "/* 顶点 (a,0) 必在右支上,用它定义双曲线(hyperbola:两焦点+曲线上一点) */",
        "var Pa = board.create('point', [function(){ return sa4.Value(); }, 0], {visible:false});",
        "var hyp = board.create('hyperbola', [F1, F2, Pa], {strokeColor:'#2563EB', strokeWidth:2.5});",
        "/* P 为曲线上滑动点(初始在右支) */",
        "var P = board.create('glider', [3.5, -2.5, hyp], {name:'P(拖我)', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('segment', [P, F1], {strokeColor:'#059669', strokeWidth:2});",
        "board.create('segment', [P, F2], {strokeColor:'#7C3AED', strokeWidth:2});",
        "function dist(U, V){ var dx = U.X()-V.X(), dy = U.Y()-V.Y(); return Math.sqrt(dx*dx+dy*dy); }",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var d1 = dist(P, F1), d2 = dist(P, F2);",
        "  return '| |PF₁|−|PF₂| | = |' + d1.toFixed(2) + '−' + d2.toFixed(2) + '| = ' + Math.abs(d1-d2).toFixed(2) + ' = 2a';",
        "}], {fontSize:13, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖动 P 沿双曲线移动:两段距离之差的绝对值不变'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'parabola_def',
    group: GG,
    name: '抛物线的定义',
    emoji: '📡',
    desc: '拖动曲线上的点,到焦点与到准线的距离始终相等',
    boundingBox: [-7, 8, 9, -8],
    keepAspectRatio: true,
    params: [
      { key: 'p', label: '焦准距 p', type: 'number', min: 1, max: 4, step: 0.1, defaultValue: 2, hint: '焦点到准线的距离' },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-7, 8, 9, -8])
      return [
        "/* 顶部控制区:面板 + 焦准距滑杆(左) + 距离对照读数(右) */",
        L.panel(1),
        "var sp = board.create('slider', [" + L.slider(0) + ", [1, " + n(p.p) + ", 4]], {name:'p', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "/* 焦点 F(p/2, 0),准线 x = -p/2(y²=2px 的标准形式) */",
        "var F = board.create('point', [function(){ return sp.Value()/2; }, 0], {name:'焦点F', size:4, fixed:true, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}});",
        "var D1 = board.create('point', [function(){ return -sp.Value()/2; }, -7], {visible:false});",
        "var D2 = board.create('point', [function(){ return -sp.Value()/2; }, 5.5], {visible:false});",
        "var direc = board.create('line', [D1, D2], {strokeColor:'#059669', strokeWidth:2, dash:2});",
        "board.create('text', [function(){ return -sp.Value()/2 - 1.5; }, 5, '准线'], {fontSize:13, strokeColor:'#059669'});",
        "/* 抛物线(parabola:焦点+准线) */",
        "var par = board.create('parabola', [F, direc], {strokeColor:'#2563EB', strokeWidth:2.5});",
        "/* P 为曲线上滑动点 */",
        "var P = board.create('glider', [3, 3.5, par], {name:'P(拖我)', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('segment', [P, F], {strokeColor:'#7C3AED', strokeWidth:2});",
        "/* P 到准线的垂线段:垂足与 P 同高、横坐标在准线上 */",
        "var Q = board.create('point', [function(){ return -sp.Value()/2; }, function(){ return P.Y(); }], {visible:false});",
        "board.create('segment', [P, Q], {strokeColor:'#059669', strokeWidth:2});",
        "function dist(U, V){ var dx = U.X()-V.X(), dy = U.Y()-V.Y(); return Math.sqrt(dx*dx+dy*dy); }",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  return '|PF| = ' + dist(P, F).toFixed(2) + '   P到准线 = ' + dist(P, Q).toFixed(2) + '  (相等!)';",
        "}], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '到焦点的距离 = 到准线的距离,这就是抛物线'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'sine_rule',
    group: GG,
    name: '正弦定理',
    emoji: '⚖️',
    desc: '拖动三角形顶点,a/sinA = b/sinB = c/sinC 恒等于外接圆直径',
    boundingBox: [-8, 8, 8, -8],
    keepAspectRatio: true,
    params: [],
    buildConstruction: () => {
      const L = makeLayout([-8, 8, 8, -8])
      return [
        "var A = board.create('point', [-4, -2.5], {name:'A', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:14}});",
        "var B = board.create('point', [4.5, -2.5], {name:'B', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:14}});",
        "var Cq = board.create('point', [0.5, 3.5], {name:'C', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:14}});",
        "board.create('polygon', [A, B, Cq], {fillColor:'#DBEAFE', fillOpacity:0.3, borders:{strokeColor:'#2563EB', strokeWidth:2}});",
        "/* 外接圆(定理的几何本质:比值=直径2R) */",
        "var cc = board.create('circumcenter', [A, B, Cq], {visible:false});",
        "board.create('circle', [cc, A], {strokeColor:'#F59E0B', strokeWidth:1.5, dash:2});",
        "function dist(U, V){ var dx = U.X()-V.X(), dy = U.Y()-V.Y(); return Math.sqrt(dx*dx+dy*dy); }",
        "function angDeg(P, V, Q){",
        "  var a1 = Math.atan2(P.Y()-V.Y(), P.X()-V.X()), a2 = Math.atan2(Q.Y()-V.Y(), Q.X()-V.X());",
        "  var d = Math.abs(a1 - a2) * 180 / Math.PI; if (d > 180) d = 360 - d; return d;",
        "}",
        "/* 顶部读数区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){",
        "  var a = dist(B, Cq), b = dist(A, Cq), c = dist(A, B);",
        "  var sA = Math.sin(angDeg(B, A, Cq)*Math.PI/180), sB = Math.sin(angDeg(A, B, Cq)*Math.PI/180), sC = Math.sin(angDeg(A, Cq, B)*Math.PI/180);",
        "  return 'a/sinA=' + (a/sA).toFixed(2) + '  b/sinB=' + (b/sB).toFixed(2) + '  c/sinC=' + (c/sC).toFixed(2) + '  2R=' + (2*dist(cc, A)).toFixed(2);",
        "}], {fontSize:13, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖动顶点:四个数始终相等——比值就是外接圆直径'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  // ---------- 并入:高中·函数 ----------

  {
    id: 'vector_add',
    group: GF,
    name: '向量加法',
    emoji: '➕',
    desc: '拖动两向量终点,三角形/平行四边形法则同屏演示',
    boundingBox: [-9, 9, 9, -9],
    keepAspectRatio: true,
    params: [
      { key: 'showPara', label: '显示平行四边形法则', type: 'boolean', defaultValue: true },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-9, 9, 9, -9])
      return [
        "var O = board.create('point', [0, -1], {name:'O', fixed:true, size:3, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:13}});",
        "/* 向量 a(蓝)、b(绿):终点可拖 */",
        "var Pa = board.create('point', [5, 0], {name:'a终点(拖我)', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:12}});",
        "var Pb = board.create('point', [2, 3.5], {name:'b终点(拖我)', size:4, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:12}});",
        "board.create('arrow', [O, Pa], {strokeColor:'#2563EB', strokeWidth:3});",
        "board.create('arrow', [O, Pb], {strokeColor:'#059669', strokeWidth:3});",
        "/* 和向量 a+b(红):相对 O 的分量相加 */",
        "var Ps = board.create('point', [function(){ return Pa.X()+Pb.X()-O.X(); }, function(){ return Pa.Y()+Pb.Y()-O.Y(); }], {name:'a+b', size:3, fixed:true, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('arrow', [O, Ps], {strokeColor:'#DC2626', strokeWidth:3.5});",
        "/* 三角形法则:从 a 终点平移 b(绿虚箭头) */",
        "board.create('arrow', [Pa, Ps], {strokeColor:'#059669', strokeWidth:2, dash:2});",
        (p.showPara ? [
          "/* 平行四边形法则:补出另外一边 */",
          "board.create('arrow', [Pb, Ps], {strokeColor:'#2563EB', strokeWidth:2, dash:2});",
          "board.create('polygon', [O, Pa, Ps, Pb], {fillColor:'#FDE68A', fillOpacity:0.18, borders:{strokeWidth:0}, vertices:{visible:false}});",
        ].join('\n') : "/* 平行四边形关闭 */"),
        "/* 顶部读数区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){",
        "  var ax = Pa.X()-O.X(), ay = Pa.Y()-O.Y(), bx = Pb.X()-O.X(), by = Pb.Y()-O.Y();",
        "  return 'a=(' + ax.toFixed(1) + ',' + ay.toFixed(1) + ')  b=(' + bx.toFixed(1) + ',' + by.toFixed(1) + ')  a+b=(' + (ax+bx).toFixed(1) + ',' + (ay+by).toFixed(1) + ')';",
        "}], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '实线红箭头=和向量;虚线=平移后的 a、b(首尾相接)'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'linear_prog',
    group: GF,
    name: '线性规划可行域',
    emoji: '🗺️',
    desc: '拖动目标函数值 z,直线平移扫过可行域找最值',
    boundingBox: [-2, 10, 12, -2],
    keepAspectRatio: true,
    params: [
      { key: 'z', label: '目标值 z = x + 2y', type: 'number', min: 0, max: 22, step: 0.5, defaultValue: 8 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-2, 10, 12, -2])
      return [
        "/* 约束:x+y≤8, x≤6, y≤7, x≥0, y≥0;可行域顶点:(0,0)(6,0)(6,2)(1,7)(0,7) */",
        "/* 顶部控制区:面板 + z 滑杆(左) + 目标值读数(右) */",
        L.panel(1),
        "var sz = board.create('slider', [" + L.slider(0) + ", [0, " + n(p.z) + ", 22]], {name:'z(拖我)', " + sliderAttrs('#DC2626', 0.5) + "});",
        "/* 可行域多边形 */",
        "board.create('polygon', [[0,0],[6,0],[6,2],[1,7],[0,7]], {fillColor:'#A7F3D0', fillOpacity:0.5, borders:{strokeColor:'#059669', strokeWidth:2}, vertices:{visible:false}});",
        "/* 三条约束边界线 */",
        "board.create('line', [[0, 8], [8, 0]], {strokeColor:'#6B7280', strokeWidth:1.5, dash:2});",
        "board.create('text', [7.6, 1.6, 'x+y=8'], {fontSize:12, strokeColor:'#6B7280'});",
        "board.create('line', [[6, -1], [6, 8]], {strokeColor:'#6B7280', strokeWidth:1.5, dash:2});",
        "board.create('text', [6.2, 7.8, 'x=6'], {fontSize:12, strokeColor:'#6B7280'});",
        "board.create('line', [[-1, 7], [11, 7]], {strokeColor:'#6B7280', strokeWidth:1.5, dash:2});",
        "board.create('text', [10.2, 7.4, 'y=7'], {fontSize:12, strokeColor:'#6B7280'});",
        "/* 目标函数直线 x+2y=z,随 z 平移(红) */",
        "board.create('functiongraph', [function(x){ return (sz.Value() - x) / 2; }], {strokeColor:'#DC2626', strokeWidth:2.5});",
        "/* 顶点标注(最值必在顶点取得) */",
        "var vs = [[0,0],[6,0],[6,2],[1,7],[0,7]];",
        "for (var i = 0; i < vs.length; i++) {",
        "  (function(v){",
        "    board.create('point', [v[0], v[1]], {name:'z=' + (v[0]+2*v[1]), size:3, fixed:true, fillColor:'#7C3AED', strokeColor:'#7C3AED', label:{fontSize:12}});",
        "  })(vs[i]);",
        "}",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var z = sz.Value();",
        "  var s = z > 15.01 ? '(已离开可行域,最大值=15)' : (z >= 14.99 ? '← 最大值! 在(1,7)取得' : '');",
        "  return 'x + 2y = ' + z.toFixed(1) + '  ' + s;",
        "}], {fontSize:13, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '红线平移扫过绿色可行域:最后离开的顶点就是最大值点'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  // ---------- 新组:微积分初步 ----------

  {
    id: 'derivative_tangent',
    group: GC,
    name: '导数与切线(割线逼近)',
    emoji: '🎿',
    desc: '拖动 Δx 趋近 0,割线渐渐变成切线,斜率收敛到导数值',
    boundingBox: [-5, 8, 7, -4],
    keepAspectRatio: false,
    params: [
      { key: 'x0', label: '切点横坐标 x₀', type: 'number', min: -2, max: 2, step: 0.1, defaultValue: 1, hint: '曲线为 y = x²/2 + 1' },
      { key: 'dx', label: 'Δx(拖向0看逼近)', type: 'number', min: 0.05, max: 3, step: 0.05, defaultValue: 2 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-5, 8, 7, -4])
      return [
        "/* 顶部控制区:面板 + 两根滑杆(左) + 斜率对照读数(右) */",
        L.panel(2),
        "var sx0 = board.create('slider', [" + L.slider(0) + ", [-2, " + n(p.x0) + ", 2]], {name:'x₀', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sdx = board.create('slider', [" + L.slider(1) + ", [0.05, " + n(p.dx) + ", 3]], {name:'Δx', " + sliderAttrs('#DC2626', 0.05) + "});",
        "/* 演示曲线 f(x)=x²/2+1,导数 f'(x)=x(数值简洁便于口算验证) */",
        "function f(x){ return x*x/2 + 1; }",
        "board.create('functiongraph', [f], {strokeColor:'#2563EB', strokeWidth:3});",
        "/* 切点 P 与动点 Q */",
        "var P = board.create('point', [function(){ return sx0.Value(); }, function(){ return f(sx0.Value()); }], {name:'P', size:4, fixed:true, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:13}});",
        "var Q = board.create('point', [function(){ return sx0.Value() + sdx.Value(); }, function(){ return f(sx0.Value() + sdx.Value()); }], {name:'Q', size:4, fixed:true, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "/* 割线PQ(红)与真切线(绿虚,对照) */",
        "board.create('line', [P, Q], {strokeColor:'#DC2626', strokeWidth:2.5});",
        "board.create('line', [P, [function(){ return sx0.Value() + 1; }, function(){ return f(sx0.Value()) + sx0.Value(); }]], {strokeColor:'#059669', strokeWidth:2, dash:2});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var x0 = sx0.Value(), dx = sdx.Value();",
        "  var slope = (f(x0+dx) - f(x0)) / dx;",
        "  return '割线斜率 = ' + slope.toFixed(3);",
        "}], {fontSize:14, strokeColor:'#DC2626', cssStyle:'font-weight:bold;'});",
        "board.create('text', [" + L.midX + ", " + L.topY(1) + ", function(){",
        "  return '切线斜率 f\\u2032(x₀) = ' + sx0.Value().toFixed(3);",
        "}], {fontSize:14, strokeColor:'#059669', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '把 Δx 拖向 0:红色割线越来越贴近绿色切线,斜率收敛到导数'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'riemann_sum',
    group: GC,
    name: '定积分与黎曼和',
    emoji: '🧱',
    desc: '增加矩形个数 n,矩形面积和逼近曲边梯形面积',
    boundingBox: [-1.5, 6, 5, -1.5],
    keepAspectRatio: false,
    params: [
      { key: 'nRect', label: '矩形个数 n', type: 'number', min: 2, max: 60, step: 1, defaultValue: 6 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-1.5, 6, 5, -1.5])
      return [
        "/* 顶部控制区:面板 + n 滑杆(左) + 提示(右) */",
        L.panel(1),
        "var snr = board.create('slider', [" + L.slider(0) + ", [2, " + n(p.nRect) + ", 60]], {name:'n', " + sliderAttrs('#7C3AED', 1) + "});",
        "/* 演示曲线 f(x)=x²+0.5,区间 [0,2],精确积分 = 8/3 + 1 = 11/3 ≈ 3.667 */",
        "function f(x){ return x*x + 0.5; }",
        "var graph = board.create('functiongraph', [f, -0.5, 3], {strokeColor:'#2563EB', strokeWidth:3});",
        "/* 黎曼和(左端点法):JSXGraph 内置 riemannsum,矩形数绑定滑杆 */",
        "board.create('riemannsum', [f, function(){ return Math.round(snr.Value()); }, 'left', 0, 2], {fillColor:'#FBBF24', fillOpacity:0.45, strokeColor:'#B45309', strokeWidth:1});",
        "board.create('text', [" + L.midX + " + 0.4, " + L.topY(0) + ", 'n 越大,矩形越贴合曲线'], {fontSize:12, strokeColor:'#6B7280'});",
        "/* 底部读数区(数值较长,放底部满宽) */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", function(){",
        "  var m = Math.round(snr.Value());",
        "  var h = 2/m, s = 0;",
        "  for (var i = 0; i < m; i++) { s += f(i*h) * h; }",
        "  return 'n=' + m + '  矩形和≈' + s.toFixed(4) + '  精确值=11/3≈3.6667  误差=' + Math.abs(s - 11/3).toFixed(4);",
        "}], {fontSize:13, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
      ].join('\n')
    },
  },
]

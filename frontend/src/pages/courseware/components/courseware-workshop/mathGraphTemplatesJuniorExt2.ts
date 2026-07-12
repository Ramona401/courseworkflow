/**
 * mathGraphTemplatesJuniorExt2.ts — 初中查缺模板(批次1c-5首发;1e修锐角三角函数;2b版式整改,2026-07-08)
 *
 * 覆盖课标查缺:
 *   数与代数(新组):数轴·相反数与绝对值 / 不等式解集在数轴上的表示 / 平面直角坐标系
 *   几何作图(并入既有组):中心对称(与轴对称成对) / 锐角三角函数
 *   函数图像(并入既有组):一次函数与二元一次方程组
 * 批次2b:6个模板全部套用 makeLayout 版式规范(顶部控制面板/滑杆成列/读数右置/
 * 底部提示面板),构造逻辑与原版完全一致,仅排版调整。
 * 编写规范见聚合出口 mathGraphTemplates.ts 文件头。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/mathGraphTemplatesJuniorExt2.ts
 */
import type { MathGraphTemplate } from './mathGraphUtils'
import { n, sliderAttrs, makeLayout } from './mathGraphTemplateShared'

const GA = '📗 初中 · 数与代数'
const GF = '📗 初中 · 函数图像'
const GG = '📗 初中 · 几何作图'

export const MATH_GRAPH_TEMPLATES_JUNIOR_EXT2: MathGraphTemplate[] = [

  // ---------- 新组:数与代数 ----------

  {
    id: 'number_line_opposite',
    showAxis: false,  // 批次3a:自带手绘数轴,系统坐标轴默认关闭(弹窗开关可开)
    group: GA,
    name: '数轴·相反数与绝对值',
    emoji: '↔️',
    desc: '拖动点 a,相反数 −a 镜像联动,两点到原点距离都是 |a|',
    boundingBox: [-11, 4.5, 11, -4.5],
    keepAspectRatio: false,
    params: [
      { key: 'start', label: '点 a 的初始位置', type: 'number', min: -9, max: 9, step: 0.5, defaultValue: 4 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-11, 4.5, 11, -4.5])
      return [
        "var base = board.create('line', [[-10, 0], [10, 0]], {straightFirst:false, straightLast:false, firstArrow:true, lastArrow:true, strokeColor:'#1F2937', strokeWidth:2.5, fixed:true});",
        "for (var i = -9; i <= 9; i++) {",
        "  board.create('segment', [[i, -0.18], [i, 0.18]], {strokeColor:'#1F2937', strokeWidth: i === 0 ? 3 : 1.5, fixed:true});",
        "  board.create('text', [i, -0.75, String(i)], {fontSize: i === 0 ? 15 : 12, strokeColor: i === 0 ? '#DC2626' : '#1F2937', anchorX:'middle', cssStyle: i === 0 ? 'font-weight:bold;' : ''});",
        "}",
        "var P = board.create('glider', [" + n(p.start) + ", 0, base], {name:'a(拖我)', size:5, fillColor:'#2563EB', strokeColor:'#2563EB', snapWidth:0.5, label:{fontSize:14, offset:[0, 18]}});",
        "/* 相反数点:横坐标取反,镜像联动 */",
        "board.create('point', [function(){ return -P.X(); }, 0], {name:'−a', size:5, fixed:true, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:14, offset:[0, 18]}});",
        "/* 两段到原点的距离(上下错开的粗线段示意) */",
        "board.create('segment', [[0, 0.9], [function(){ return P.X(); }, 0.9]], {strokeColor:'#2563EB', strokeWidth:3});",
        "board.create('segment', [[0, -1.3], [function(){ return -P.X(); }, -1.3]], {strokeColor:'#059669', strokeWidth:3});",
        "/* 顶部读数区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){",
        "  var a = Math.round(P.X()*2)/2;",
        "  return 'a = ' + a + '   相反数 −a = ' + (-a) + '   |a| = |−a| = ' + Math.abs(a);",
        "}], {fontSize:16, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '相反数关于原点对称;绝对值就是到原点的距离(蓝绿两段一样长)'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'inequality_line',
    showAxis: false,  // 批次3a:自带手绘数轴,系统坐标轴默认关闭(弹窗开关可开)
    group: GA,
    name: '不等式解集在数轴上的表示',
    emoji: '🚦',
    desc: '拖动边界值,切换 >/≥/</≤,空心实心与方向实时变化',
    boundingBox: [-11, 4.5, 11, -4.5],
    keepAspectRatio: false,
    params: [
      { key: 'a', label: '边界值 a', type: 'number', min: -8, max: 8, step: 0.5, defaultValue: 2 },
      { key: 'greater', label: '方向: x > a(取消则 x < a)', type: 'boolean', defaultValue: true },
      { key: 'inclusive', label: '含等号(实心圆点)', type: 'boolean', defaultValue: false },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-11, 4.5, 11, -4.5])
      const greater = p.greater as boolean
      const inclusive = p.inclusive as boolean
      return [
        "/* 顶部控制区:面板 + 边界滑杆(左) + 解集读数(右) */",
        L.panel(1),
        "var sa = board.create('slider', [" + L.slider(0) + ", [-8, " + n(p.a) + ", 8]], {name:'a', " + sliderAttrs('#7C3AED', 0.5) + "});",
        "var base = board.create('line', [[-10, 0], [10, 0]], {straightFirst:false, straightLast:false, firstArrow:true, lastArrow:true, strokeColor:'#1F2937', strokeWidth:2.5, fixed:true});",
        "for (var i = -9; i <= 9; i++) {",
        "  board.create('segment', [[i, -0.18], [i, 0.18]], {strokeColor:'#1F2937', strokeWidth: i === 0 ? 3 : 1.5, fixed:true});",
        "  board.create('text', [i, -0.75, String(i)], {fontSize:12, strokeColor:'#1F2937', anchorX:'middle'});",
        "}",
        "/* 边界圆点:实心=含等号,空心=不含 */",
        "board.create('point', [function(){ return sa.Value(); }, 0], {name:'', size:5, fixed:true, strokeColor:'#DC2626', strokeWidth:2.5, fillColor:" + (inclusive ? "'#DC2626'" : "'#FFFFFF'") + "});",
        "/* 解集射线(粗红线+箭头):方向随参数 */",
        "board.create('arrow', [[function(){ return sa.Value(); }, 0.55], [function(){ return " + (greater ? "9.8" : "-9.8") + "; }, 0.55]], {strokeColor:'#DC2626', strokeWidth:4});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var a = Math.round(sa.Value()*2)/2;",
        "  return '解集: x " + (greater ? (inclusive ? '≥' : '>') : (inclusive ? '≤' : '<')) + " ' + a;",
        "}], {fontSize:16, strokeColor:'#B45309', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '" + (inclusive ? '实心点=边界值本身也是解' : '空心点=不包括边界值') + ";红色箭头覆盖的数都是解'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'coordinate_plane',
    group: GA,
    name: '平面直角坐标系与象限',
    emoji: '🎯',
    desc: '拖动点看坐标与所在象限,横纵坐标虚线投影联动',
    boundingBox: [-8, 8, 8, -8],
    keepAspectRatio: true,
    params: [
      { key: 'x0', label: '初始横坐标', type: 'number', min: -6, max: 6, step: 0.5, defaultValue: 3 },
      { key: 'y0', label: '初始纵坐标', type: 'number', min: -6, max: 6, step: 0.5, defaultValue: 2 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-8, 8, 8, -8])
      return [
        "/* 象限底色(淡色区分四个象限) */",
        "board.create('polygon', [[0,0],[7.8,0],[7.8,7.8],[0,7.8]], {fillColor:'#DBEAFE', fillOpacity:0.25, borders:{strokeWidth:0}, vertices:{visible:false}});",
        "board.create('polygon', [[0,0],[-7.8,0],[-7.8,7.8],[0,7.8]], {fillColor:'#D1FAE5', fillOpacity:0.25, borders:{strokeWidth:0}, vertices:{visible:false}});",
        "board.create('polygon', [[0,0],[-7.8,0],[-7.8,-7.8],[0,-7.8]], {fillColor:'#FDE68A', fillOpacity:0.25, borders:{strokeWidth:0}, vertices:{visible:false}});",
        "board.create('polygon', [[0,0],[7.8,0],[7.8,-7.8],[0,-7.8]], {fillColor:'#FECACA', fillOpacity:0.25, borders:{strokeWidth:0}, vertices:{visible:false}});",
        "board.create('text', [4, 3.5, '第一象限(+,+)'], {fontSize:13, strokeColor:'#2563EB', anchorX:'middle'});",
        "board.create('text', [-4, 3.5, '第二象限(−,+)'], {fontSize:13, strokeColor:'#059669', anchorX:'middle'});",
        "board.create('text', [-4, -4, '第三象限(−,−)'], {fontSize:13, strokeColor:'#B45309', anchorX:'middle'});",
        "board.create('text', [4, -4, '第四象限(+,−)'], {fontSize:13, strokeColor:'#DC2626', anchorX:'middle'});",
        "var P = board.create('point', [" + n(p.x0) + ", " + n(p.y0) + "], {name:'P(拖我)', size:5, fillColor:'#1F2937', strokeColor:'#1F2937', snapToGrid:true, snapSizeX:0.5, snapSizeY:0.5, label:{fontSize:14}});",
        "/* 到两轴的虚线投影 */",
        "board.create('segment', [P, [function(){ return P.X(); }, 0]], {strokeColor:'#9CA3AF', strokeWidth:1.5, dash:2});",
        "board.create('segment', [P, [0, function(){ return P.Y(); }]], {strokeColor:'#9CA3AF', strokeWidth:1.5, dash:2});",
        "/* 顶部读数区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){",
        "  var x = P.X(), y = P.Y();",
        "  var q = '坐标轴上(不属于任何象限)';",
        "  if (x > 0 && y > 0) q = '第一象限'; else if (x < 0 && y > 0) q = '第二象限';",
        "  else if (x < 0 && y < 0) q = '第三象限'; else if (x > 0 && y < 0) q = '第四象限';",
        "  return 'P(' + x.toFixed(1) + ', ' + y.toFixed(1) + ')  位于: ' + q;",
        "}], {fontSize:15, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖动 P 到各个区域:横纵坐标的正负号决定所在象限'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  // ---------- 并入:几何作图 ----------

  {
    id: 'central_symmetry',
    group: GG,
    name: '中心对称',
    emoji: '💠',
    desc: '拖动图形或对称中心,180°旋转像实时联动(与轴对称对照)',
    boundingBox: [-9, 9, 9, -9],
    keepAspectRatio: true,
    params: [
      { key: 'showLinks', label: '显示对应点连线(过中心且被平分)', type: 'boolean', defaultValue: true },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-9, 9, 9, -9])
      return [
        "/* 对称中心 O(可拖) */",
        "var O = board.create('point', [0.5, -0.5], {name:'对称中心O(拖我)', size:5, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}});",
        "/* 原三角形(蓝,顶点可拖) */",
        "var P1 = board.create('point', [3, 1.5], {name:'A', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var P2 = board.create('point', [7, 2.5], {name:'B', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var P3 = board.create('point', [4.5, 5.2], {name:'C', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "board.create('polygon', [P1, P2, P3], {fillColor:'#DBEAFE', fillOpacity:0.4, borders:{strokeColor:'#2563EB', strokeWidth:2}});",
        "/* 中心对称像:P′ = 2O − P(绕 O 转 180°) */",
        "function csX(P){ return function(){ return 2*O.X() - P.X(); }; }",
        "function csY(P){ return function(){ return 2*O.Y() - P.Y(); }; }",
        "var Q1 = board.create('point', [csX(P1), csY(P1)], {name:'A′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "var Q2 = board.create('point', [csX(P2), csY(P2)], {name:'B′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "var Q3 = board.create('point', [csX(P3), csY(P3)], {name:'C′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "board.create('polygon', [Q1, Q2, Q3], {fillColor:'#D1FAE5', fillOpacity:0.4, borders:{strokeColor:'#059669', strokeWidth:2}});",
        (p.showLinks ? [
          "/* 对应点连线:必过中心 O 且被 O 平分 */",
          "board.create('segment', [P1, Q1], {strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
          "board.create('segment', [P2, Q2], {strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
          "board.create('segment', [P3, Q3], {strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
        ].join('\n') : "/* 连线关闭 */"),
        "/* 顶部说明区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", '中心对称 = 绕 O 旋转180°;对应点连线都经过 O 且被 O 平分'], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '对比轴对称:轴对称沿轴翻折(左右手关系),中心对称旋转半圈(方向不变)'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'trig_ratio',
    group: GG,
    name: '锐角三角函数',
    emoji: '📐',
    desc: '拖动角度,直角三角形三边联动,sin/cos/tan 实时读数',
    boundingBox: [-2, 9, 12, -2],
    keepAspectRatio: true,
    params: [
      { key: 'deg', label: '锐角 A(度)', type: 'number', min: 10, max: 80, step: 1, defaultValue: 35 },
      { key: 'hyp', label: '斜边长', type: 'number', min: 4, max: 9, step: 0.5, defaultValue: 7 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-2, 9, 12, -2])
      return [
        "/* 顶部控制区:面板 + 两根滑杆(左) + 三行分色读数(右,2b统一进面板) */",
        L.panel(3),
        "var sdeg = board.create('slider', [" + L.slider(0) + ", [10, " + n(p.deg) + ", 80]], {name:'∠A', " + sliderAttrs('#7C3AED', 1) + "});",
        "var shyp = board.create('slider', [" + L.slider(1) + ", [4, " + n(p.hyp) + ", 9]], {name:'斜边', " + sliderAttrs('#059669', 0.5) + "});",
        "function rad(){ return sdeg.Value() * Math.PI / 180; }",
        "/* 直角三角形:A 在原点,直角在 B,C 在上方 */",
        "var A = board.create('point', [0, 0], {name:'A', fixed:true, size:3, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:14, offset:[-14, -12]}});",
        "var B = board.create('point', [function(){ return shyp.Value()*Math.cos(rad()); }, 0], {name:'B', size:3, fixed:true, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:14, offset:[8, -12]}});",
        "var Cq = board.create('point', [function(){ return shyp.Value()*Math.cos(rad()); }, function(){ return shyp.Value()*Math.sin(rad()); }], {name:'C', size:3, fixed:true, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:14, offset:[8, 6]}});",
        "board.create('polygon', [A, B, Cq], {fillColor:'#DBEAFE', fillOpacity:0.4, borders:{strokeWidth:0}, vertices:{visible:false}});",
        "/* 三边分色:邻边绿/对边红/斜边蓝 */",
        "board.create('segment', [A, B], {strokeColor:'#059669', strokeWidth:3.5});",
        "board.create('segment', [B, Cq], {strokeColor:'#DC2626', strokeWidth:3.5});",
        "board.create('segment', [A, Cq], {strokeColor:'#2563EB', strokeWidth:3.5});",
        "/* 直角标记与角 A 弧(角弧不带名,避免与顶点 A 标签撞车) */",
        "board.create('nonreflexangle', [Cq, B, A], {radius:0.5, name:'', strokeColor:'#6B7280', fillOpacity:0, type:'square'});",
        "board.create('nonreflexangle', [B, A, Cq], {radius:1.0, name:'', strokeColor:'#F59E0B', fillColor:'#FDE68A', fillOpacity:0.6});",
        "/* 三比值读数:三行分色与三边呼应 */",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){ return 'sin A = 对/斜 = ' + Math.sin(rad()).toFixed(3); }], {fontSize:14, strokeColor:'#DC2626', cssStyle:'font-weight:bold;'});",
        "board.create('text', [" + L.midX + ", " + L.topY(1) + ", function(){ return 'cos A = 邻/斜 = ' + Math.cos(rad()).toFixed(3); }], {fontSize:14, strokeColor:'#059669', cssStyle:'font-weight:bold;'});",
        "board.create('text', [" + L.midX + ", " + L.topY(2) + ", function(){ return 'tan A = 对/邻 = ' + Math.tan(rad()).toFixed(3); }], {fontSize:14, strokeColor:'#2563EB', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '红=对边 绿=邻边 蓝=斜边;拖斜边滑杆:三角形变大,但三个比值纹丝不动!'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  // ---------- 并入:函数图像 ----------

  {
    id: 'linear_system',
    group: GF,
    name: '一次函数与二元一次方程组',
    emoji: '✖️',
    desc: '两条直线的交点就是方程组的解,拖参数看相交/平行',
    boundingBox: [-10, 10, 10, -10],
    keepAspectRatio: false,
    params: [
      { key: 'k1', label: '直线1: 斜率 k₁', type: 'number', min: -5, max: 5, step: 0.1, defaultValue: 1 },
      { key: 'b1', label: '直线1: 截距 b₁', type: 'number', min: -8, max: 8, step: 0.5, defaultValue: -1 },
      { key: 'k2', label: '直线2: 斜率 k₂', type: 'number', min: -5, max: 5, step: 0.1, defaultValue: -0.5 },
      { key: 'b2', label: '直线2: 截距 b₂', type: 'number', min: -8, max: 8, step: 0.5, defaultValue: 3.5 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 10, 10, -10])
      return [
        "/* 顶部控制区:面板 + 四根滑杆(左,紫=直线1/绿=直线2) + 解读数(右) */",
        L.panel(4),
        "var sk1 = board.create('slider', [" + L.slider(0) + ", [-5, " + n(p.k1) + ", 5]], {name:'k₁', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sb1 = board.create('slider', [" + L.slider(1) + ", [-8, " + n(p.b1) + ", 8]], {name:'b₁', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sk2 = board.create('slider', [" + L.slider(2) + ", [-5, " + n(p.k2) + ", 5]], {name:'k₂', " + sliderAttrs('#059669', 0.1) + "});",
        "var sb2 = board.create('slider', [" + L.slider(3) + ", [-8, " + n(p.b2) + ", 8]], {name:'b₂', " + sliderAttrs('#059669', 0.1) + "});",
        "board.create('functiongraph', [function(x){ return sk1.Value()*x + sb1.Value(); }], {strokeColor:'#7C3AED', strokeWidth:3});",
        "board.create('functiongraph', [function(x){ return sk2.Value()*x + sb2.Value(); }], {strokeColor:'#059669', strokeWidth:3});",
        "/* 交点(即方程组的解):两斜率几乎相等时视为平行,隐藏交点 */",
        "function parallel(){ return Math.abs(sk1.Value() - sk2.Value()) < 0.02; }",
        "function ix(){ return (sb2.Value() - sb1.Value()) / (sk1.Value() - sk2.Value()); }",
        "board.create('point', [function(){ return parallel() ? 0 : ix(); }, function(){ return parallel() ? 0 : sk1.Value()*ix() + sb1.Value(); }], {",
        "  name:'解', size:5, fixed:true, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:14},",
        "  visible: function(){ return !parallel(); }",
        "});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  if (parallel()) return 'k₁ = k₂:两直线平行 → 无解';",
        "  var x = ix(), y = sk1.Value()*x + sb1.Value();",
        "  return '交点(' + x.toFixed(2) + ', ' + y.toFixed(2) + ') = 方程组的解';",
        "}], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "board.create('text', [" + L.midX + ", " + L.topY(1) + ", '紫线: y = k₁x + b₁'], {fontSize:12, strokeColor:'#7C3AED'});",
        "board.create('text', [" + L.midX + ", " + L.topY(2) + ", '绿线: y = k₂x + b₂'], {fontSize:12, strokeColor:'#059669'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '把 k₂ 拖到等于 k₁:两线平行,红色解点消失(无解)'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },
]

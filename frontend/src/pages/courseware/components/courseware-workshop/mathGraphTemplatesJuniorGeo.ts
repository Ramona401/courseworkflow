/**
 * mathGraphTemplatesJuniorGeo.ts — 初中·几何作图模板(批次1c拆分;批次2a版式整改,2026-07-08)
 *
 * 覆盖课标:圆的切线/圆周角、三角形的心、勾股定理、平行线三线八角、轴对称变换。
 * 几何类模板 keepAspectRatio 一律 true(防圆变椭圆、角弧变形)。
 * 批次2a:6个模板全部套用 makeLayout 版式规范(顶部控制面板/滑杆成列/读数右置/
 * 底部提示面板),几何构造逻辑与原版完全一致,仅排版调整。
 * 编写规范见聚合出口 mathGraphTemplates.ts 文件头。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/mathGraphTemplatesJuniorGeo.ts
 */
import type { MathGraphTemplate } from './mathGraphUtils'
import { n, sliderAttrs, makeLayout } from './mathGraphTemplateShared'

const G = '📗 初中 · 几何作图'

export const MATH_GRAPH_TEMPLATES_JUNIOR_GEO: MathGraphTemplate[] = [

  {
    id: 'circle_tangent',
    group: G,
    name: '圆与切线',
    emoji: '⭕',
    desc: '拖动切点沿圆周移动,直观感受切线垂直于半径',
    boundingBox: [-8, 8, 8, -8],
    keepAspectRatio: true,
    params: [
      { key: 'r', label: '半径 r', type: 'number', min: 1, max: 6, step: 0.5, defaultValue: 3.5 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-8, 8, 8, -8])
      return [
        "/* 顶部控制区:面板 + 半径滑杆 */",
        L.panel(1),
        "var sr = board.create('slider', [" + L.slider(0) + ", [1, " + n(p.r) + ", 6]], {name:'r', " + sliderAttrs('#7C3AED', 0.5) + "});",
        "var O = board.create('point', [0, -0.5], {name:'O', fixed:true, size:3, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:14}});",
        "var circ = board.create('circle', [O, function(){ return sr.Value(); }], {strokeColor:'#2563EB', strokeWidth:2.5, fillColor:'#DBEAFE', fillOpacity:0.25});",
        "/* 切点 P 是圆上滑动点(glider),可沿圆周拖动 */",
        "var P = board.create('glider', [" + n((p.r as number) * 0.7071) + ", " + n((p.r as number) * 0.7071 - 0.5) + ", circ], {name:'P(拖我)', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('segment', [O, P], {strokeColor:'#059669', strokeWidth:2, dash:2});",
        "board.create('tangent', [P], {strokeColor:'#F59E0B', strokeWidth:2.5});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '切线 ⊥ 半径 OP(拖动 P 沿圆周观察)'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'triangle_centers',
    group: G,
    name: '三角形与三心',
    emoji: '🔺',
    desc: '拖动三个顶点,观察重心/外心/内心的位置变化',
    boundingBox: [-8, 8, 8, -8],
    keepAspectRatio: true,
    params: [
      { key: 'showCentroid', label: '重心(中线交点)', type: 'boolean', defaultValue: true },
      { key: 'showCircumcenter', label: '外心(外接圆圆心)', type: 'boolean', defaultValue: true },
      { key: 'showIncenter', label: '内心(内切圆圆心)', type: 'boolean', defaultValue: false },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-8, 8, 8, -8])
      return [
        "var A = board.create('point', [-4, -3], {name:'A', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:14}});",
        "var B = board.create('point', [4.5, -3], {name:'B', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:14}});",
        "var Cp = board.create('point', [0.5, 4.5], {name:'C', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:14}});",
        "board.create('polygon', [A, B, Cp], {fillColor:'#DBEAFE', fillOpacity:0.3, borders:{strokeColor:'#2563EB', strokeWidth:2}});",
        (p.showCentroid ? [
          "/* 重心 G:三顶点坐标平均 */",
          "board.create('point', [function(){ return (A.X()+B.X()+Cp.X())/3; }, function(){ return (A.Y()+B.Y()+Cp.Y())/3; }], {name:'重心G', size:3, fixed:true, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        ].join('\n') : "/* 重心关闭 */"),
        (p.showCircumcenter ? [
          "/* 外心 + 外接圆 */",
          "var cc = board.create('circumcenter', [A, B, Cp], {name:'外心', size:3, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}});",
          "board.create('circle', [cc, A], {strokeColor:'#F59E0B', strokeWidth:1.5, dash:2});",
        ].join('\n') : "/* 外心关闭 */"),
        (p.showIncenter ? [
          "/* 内心 + 内切圆 */",
          "var ic = board.create('incenter', [A, B, Cp], {name:'内心', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
          "board.create('incircle', [A, B, Cp], {strokeColor:'#059669', strokeWidth:1.5, dash:2, fillOpacity:0});",
        ].join('\n') : "/* 内心关闭 */"),
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖动 A、B、C 三个顶点,观察各心的位置变化'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'pythagorean',
    group: G,
    name: '勾股定理演示',
    emoji: '📏',
    desc: '三边正方形面积可视化:a² + b² = c²',
    boundingBox: [-8, 14, 14, -8],
    keepAspectRatio: true,
    params: [
      { key: 'a', label: '直角边 a', type: 'number', min: 1, max: 6, step: 0.5, defaultValue: 4 },
      { key: 'b', label: '直角边 b', type: 'number', min: 1, max: 6, step: 0.5, defaultValue: 3 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-8, 14, 14, -8])
      return [
        "/* 顶部控制区:面板 + 两根滑杆(左) + 等式读数(右) */",
        L.panel(2),
        "var sA2 = board.create('slider', [" + L.slider(0) + ", [1, " + n(p.a) + ", 6]], {name:'a', " + sliderAttrs('#7C3AED', 0.5) + "});",
        "var sB2 = board.create('slider', [" + L.slider(1) + ", [1, " + n(p.b) + ", 6]], {name:'b', " + sliderAttrs('#7C3AED', 0.5) + "});",
        "var av = function(){ return sA2.Value(); }, bv = function(){ return sB2.Value(); };",
        "/* 直角三角形:直角顶点在原点,两直角边沿坐标轴 */",
        "var P0 = board.create('point', [0, 0], {visible:false});",
        "var PA = board.create('point', [av, 0], {visible:false});",
        "var PB = board.create('point', [0, bv], {visible:false});",
        "board.create('polygon', [P0, PA, PB], {fillColor:'#FDE68A', fillOpacity:0.6, borders:{strokeColor:'#B45309', strokeWidth:2}});",
        "/* a 边正方形(x 轴下方,蓝) */",
        "board.create('polygon', [P0, PA, [av, function(){ return -av(); }], [0, function(){ return -av(); }]], {fillColor:'#93C5FD', fillOpacity:0.45, borders:{strokeColor:'#2563EB', strokeWidth:1.5}, vertices:{visible:false}});",
        "/* b 边正方形(y 轴左侧,绿) */",
        "board.create('polygon', [P0, PB, [function(){ return -bv(); }, bv], [function(){ return -bv(); }, 0]], {fillColor:'#6EE7B7', fillOpacity:0.45, borders:{strokeColor:'#059669', strokeWidth:1.5}, vertices:{visible:false}});",
        "/* 斜边正方形(外侧,橙):PA→PB 方向旋转 -90° 得外法向 (b, a) */",
        "board.create('polygon', [PA, PB, [bv, function(){ return bv()+av(); }], [function(){ return av()+bv(); }, av]], {fillColor:'#FDBA74', fillOpacity:0.45, borders:{strokeColor:'#EA580C', strokeWidth:1.5}, vertices:{visible:false}});",
        "/* 三块面积标注 + 等式 */",
        "board.create('text', [function(){ return av()/2; }, function(){ return -av()/2; }, function(){ return 'a²=' + (av()*av()).toFixed(1); }], {fontSize:15, strokeColor:'#1E40AF', anchorX:'middle', cssStyle:'font-weight:bold;'});",
        "board.create('text', [function(){ return -bv()/2; }, function(){ return bv()/2; }, function(){ return 'b²=' + (bv()*bv()).toFixed(1); }], {fontSize:15, strokeColor:'#065F46', anchorX:'middle', cssStyle:'font-weight:bold;'});",
        "board.create('text', [function(){ return (av()+bv())/2 + 0.5; }, function(){ return (av()+bv())/2 + 0.5; }, function(){ return 'c²=' + (av()*av()+bv()*bv()).toFixed(1); }], {fontSize:15, strokeColor:'#9A3412', anchorX:'middle', cssStyle:'font-weight:bold;'});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var a = av(), b = bv();",
        "  return a.toFixed(1) + '² + ' + b.toFixed(1) + '² = ' + (a*a+b*b).toFixed(1) + ',c = ' + Math.sqrt(a*a+b*b).toFixed(2);",
        "}], {fontSize:16, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "board.create('text', [" + L.midX + ", " + L.topY(1) + ", '三个正方形:两直角边的面积和 = 斜边的面积'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'inscribed_angle',
    group: G,
    name: '圆周角定理',
    emoji: '🎯',
    desc: '拖动三点验证:同弧上圆心角 = 2 × 圆周角',
    boundingBox: [-6, 6, 6, -6],
    keepAspectRatio: true,
    params: [
      { key: 'r', label: '圆半径 r', type: 'number', min: 2, max: 4.5, step: 0.5, defaultValue: 3.5 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-6, 6, 6, -6])
      const r = p.r as number
      return [
        "var O = board.create('point', [0, -0.5], {name:'O', fixed:true, size:3, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:13}});",
        "var circ = board.create('circle', [O, " + n(r) + "], {strokeColor:'#2563EB', strokeWidth:2});",
        "/* A、B 是弧的两端,C 是圆周角顶点,三点都可沿圆周拖动 */",
        "var A = board.create('glider', [" + n(r) + ", -0.5, circ], {name:'A', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var B = board.create('glider', [" + n(r * Math.cos(2.094)) + ", " + n(r * Math.sin(2.094) - 0.5) + ", circ], {name:'B', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var Cg = board.create('glider', [" + n(r * Math.cos(4.189)) + ", " + n(r * Math.sin(4.189) - 0.5) + ", circ], {name:'C(拖我)', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('segment', [O, A], {strokeColor:'#F59E0B', strokeWidth:2});",
        "board.create('segment', [O, B], {strokeColor:'#F59E0B', strokeWidth:2});",
        "board.create('segment', [Cg, A], {strokeColor:'#DC2626', strokeWidth:2});",
        "board.create('segment', [Cg, B], {strokeColor:'#DC2626', strokeWidth:2});",
        "/* 夹角计算(取 0~180° 的非反角) */",
        "function angDeg(P, V, Q){",
        "  var a1 = Math.atan2(P.Y()-V.Y(), P.X()-V.X()), a2 = Math.atan2(Q.Y()-V.Y(), Q.X()-V.X());",
        "  var d = Math.abs(a1 - a2) * 180 / Math.PI; if (d > 180) d = 360 - d; return d;",
        "}",
        "/* 顶部读数区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){",
        "  var central = angDeg(A, O, B), inscribed = angDeg(A, Cg, B);",
        "  return '圆心角 ∠AOB = ' + central.toFixed(1) + '°   圆周角 ∠ACB = ' + inscribed.toFixed(1) + '°';",
        "}], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖动 C 沿优弧移动:圆周角不变且为圆心角一半'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'parallel_lines',
    group: G,
    name: '平行线与三线八角',
    emoji: '🛤️',
    desc: '转动截线,观察同位角始终相等',
    boundingBox: [-9, 6, 9, -6],
    keepAspectRatio: true,
    params: [
      { key: 'deg', label: '截线倾角(度)', type: 'number', min: 25, max: 155, step: 1, defaultValue: 60 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-9, 6, 9, -6])
      return [
        "/* 顶部控制区:面板 + 倾角滑杆(左) + 同位角读数(右) */",
        L.panel(1),
        "var st = board.create('slider', [" + L.slider(0) + ", [25, " + n(p.deg) + ", 155]], {name:'倾角', " + sliderAttrs('#7C3AED', 1) + "});",
        "function rad(){ return st.Value() * Math.PI / 180; }",
        "/* 两条固定平行线 a、b */",
        "var l1 = board.create('line', [[-9, 2], [9, 2]], {fixed:true, strokeColor:'#2563EB', strokeWidth:2.5});",
        "var l2 = board.create('line', [[-9, -2], [9, -2]], {fixed:true, strokeColor:'#2563EB', strokeWidth:2.5});",
        "board.create('text', [8, 2.5, 'a'], {fontSize:15, strokeColor:'#2563EB', cssStyle:'font-weight:bold;'});",
        "board.create('text', [8, -1.5, 'b'], {fontSize:15, strokeColor:'#2563EB', cssStyle:'font-weight:bold;'});",
        "/* 截线:过原点,方向随滑杆转动 */",
        "var D = board.create('point', [function(){ return 3*Math.cos(rad()); }, function(){ return 3*Math.sin(rad()); }], {visible:false});",
        "var Org = board.create('point', [0, 0], {visible:false});",
        "var trans = board.create('line', [Org, D], {strokeColor:'#DC2626', strokeWidth:2.5});",
        "/* 与两平行线的交点 */",
        "var I1 = board.create('intersection', [trans, l1, 0], {name:'E', size:3, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:13}});",
        "var I2 = board.create('intersection', [trans, l2, 0], {name:'F', size:3, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:13}});",
        "/* 两个同位角(交点右侧方向 → 截线上方方向),角弧同色示意相等 */",
        "var R1 = board.create('point', [function(){ return I1.X()+2; }, function(){ return I1.Y(); }], {visible:false});",
        "var T1 = board.create('point', [function(){ return I1.X()+2*Math.cos(rad()); }, function(){ return I1.Y()+2*Math.sin(rad()); }], {visible:false});",
        "var R2 = board.create('point', [function(){ return I2.X()+2; }, function(){ return I2.Y(); }], {visible:false});",
        "var T2 = board.create('point', [function(){ return I2.X()+2*Math.cos(rad()); }, function(){ return I2.Y()+2*Math.sin(rad()); }], {visible:false});",
        "board.create('angle', [R1, I1, T1], {radius:0.7, name:'∠1', strokeColor:'#F59E0B', fillColor:'#FDE68A', fillOpacity:0.55, label:{fontSize:13}});",
        "board.create('angle', [R2, I2, T2], {radius:0.7, name:'∠2', strokeColor:'#F59E0B', fillColor:'#FDE68A', fillOpacity:0.55, label:{fontSize:13}});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  return '同位角 ∠1 = ∠2 = ' + st.Value().toFixed(0) + '°';",
        "}], {fontSize:15, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", 'a ∥ b,拖动滑杆转动截线,同位角始终相等'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'reflection',
    group: G,
    name: '轴对称变换',
    emoji: '🪞',
    desc: '拖动三角形顶点或对称轴,镜像图形实时联动',
    boundingBox: [-9, 9, 9, -9],
    keepAspectRatio: true,
    params: [
      { key: 'showLinks', label: '显示对应点连线(垂直于对称轴)', type: 'boolean', defaultValue: true },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-9, 9, 9, -9])
      return [
        "/* 对称轴:两个可拖端点决定,初始为竖直线(端点避开底部提示区) */",
        "var L1 = board.create('point', [0, -6], {name:'', size:3, fillColor:'#F59E0B', strokeColor:'#F59E0B'});",
        "var L2 = board.create('point', [0, 6.5], {name:'', size:3, fillColor:'#F59E0B', strokeColor:'#F59E0B'});",
        "var axis = board.create('line', [L1, L2], {strokeColor:'#F59E0B', strokeWidth:2, dash:2});",
        "/* 原三角形(蓝,三顶点可拖) */",
        "var P1 = board.create('point', [2, 1], {name:'A', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var P2 = board.create('point', [6, 2], {name:'B', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var P3 = board.create('point', [3.5, 5], {name:'C', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "board.create('polygon', [P1, P2, P3], {fillColor:'#DBEAFE', fillOpacity:0.4, borders:{strokeColor:'#2563EB', strokeWidth:2}});",
        "/* 镜像三角形(绿,随原图与轴自动联动) */",
        "var Q1 = board.create('reflection', [P1, axis], {name:'A′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "var Q2 = board.create('reflection', [P2, axis], {name:'B′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "var Q3 = board.create('reflection', [P3, axis], {name:'C′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "board.create('polygon', [Q1, Q2, Q3], {fillColor:'#D1FAE5', fillOpacity:0.4, borders:{strokeColor:'#059669', strokeWidth:2}});",
        (p.showLinks ? [
          "/* 对应点连线(被对称轴垂直平分) */",
          "board.create('segment', [P1, Q1], {strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
          "board.create('segment', [P2, Q2], {strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
          "board.create('segment', [P3, Q3], {strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
        ].join('\n') : "/* 连线关闭 */"),
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖动 A/B/C 或对称轴端点(粉色),观察镜像联动'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },
]

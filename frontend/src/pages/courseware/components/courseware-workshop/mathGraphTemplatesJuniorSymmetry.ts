/**
 * mathGraphTemplatesJuniorSymmetry.ts — 初中·对称与全等专题模板(批次1d首发;1h版式试点,2026-07-08)
 *
 * 批次1h:6个模板全部套用 makeLayout 版式规范(顶部控制区背板+底部提示背板,
 * 坐标由 boundingBox 自动计算),作为全量48模板排版整改的试点组。
 * 几何构造逻辑与1d完全一致,仅排版调整。
 * 编写规范见聚合出口 mathGraphTemplates.ts 文件头。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/mathGraphTemplatesJuniorSymmetry.ts
 */
import type { MathGraphTemplate } from './mathGraphUtils'
import { n, sliderAttrs, makeLayout } from './mathGraphTemplateShared'

const GS = '📗 初中 · 对称与全等'

export const MATH_GRAPH_TEMPLATES_JUNIOR_SYMMETRY: MathGraphTemplate[] = [

  {
    id: 'symmetry_congruent',
    group: GS,
    name: '轴对称与全等验证',
    emoji: '🪞',
    desc: '拖动三角形或对称轴,实时验证对应边、对应角全部相等',
    boundingBox: [-10, 9, 10, -9],
    keepAspectRatio: true,
    params: [
      { key: 'showSides', label: '显示对应边长度对照', type: 'boolean', defaultValue: true },
      { key: 'showAngles', label: '显示对应角(同色角弧)', type: 'boolean', defaultValue: false },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 9, 10, -9])
      return [
        "/* 对称轴:两个可拖端点(橙),初始竖直 */",
        "var L1 = board.create('point', [0, -6.5], {name:'', size:3, fillColor:'#F59E0B', strokeColor:'#F59E0B'});",
        "var L2 = board.create('point', [0, 5.5], {name:'', size:3, fillColor:'#F59E0B', strokeColor:'#F59E0B'});",
        "var axis = board.create('line', [L1, L2], {strokeColor:'#F59E0B', strokeWidth:2, dash:2});",
        "/* 原三角形(蓝,三顶点可拖) */",
        "var A = board.create('point', [2.5, -0.5], {name:'A', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var B = board.create('point', [7, 0.5], {name:'B', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var Cq = board.create('point', [4, 4], {name:'C', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "board.create('polygon', [A, B, Cq], {fillColor:'#DBEAFE', fillOpacity:0.4, borders:{strokeColor:'#2563EB', strokeWidth:2}});",
        "/* 镜像三角形(绿,随原图与轴自动联动) */",
        "var Q1 = board.create('reflection', [A, axis], {name:'A′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "var Q2 = board.create('reflection', [B, axis], {name:'B′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "var Q3 = board.create('reflection', [Cq, axis], {name:'C′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "board.create('polygon', [Q1, Q2, Q3], {fillColor:'#D1FAE5', fillOpacity:0.4, borders:{strokeColor:'#059669', strokeWidth:2}});",
        "function dist(U, V){ var dx = U.X()-V.X(), dy = U.Y()-V.Y(); return Math.sqrt(dx*dx+dy*dy); }",
        (p.showSides ? [
          "/* 顶部读数区:背板 + 两行对应边对照 */",
          L.panel(2),
          "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){ return '原: AB=' + dist(A,B).toFixed(2) + '  BC=' + dist(B,Cq).toFixed(2) + '  CA=' + dist(Cq,A).toFixed(2); }], {fontSize:14, strokeColor:'#1E40AF', cssStyle:'font-weight:bold;'});",
          "board.create('text', [" + L.leftX + ", " + L.topY(1) + ", function(){ return '像: A′B′=' + dist(Q1,Q2).toFixed(2) + '  B′C′=' + dist(Q2,Q3).toFixed(2) + '  C′A′=' + dist(Q3,Q1).toFixed(2) + '  → 全等'; }], {fontSize:14, strokeColor:'#065F46', cssStyle:'font-weight:bold;'});",
        ].join('\n') : "/* 边长对照关闭 */"),
        (p.showAngles ? [
          "/* 对应角同色角弧,肉眼可见对应角相等 */",
          "board.create('nonreflexangle', [B, A, Cq], {radius:0.8, name:'', strokeColor:'#DC2626', fillColor:'#FCA5A5', fillOpacity:0.6});",
          "board.create('nonreflexangle', [Q2, Q1, Q3], {radius:0.8, name:'', strokeColor:'#DC2626', fillColor:'#FCA5A5', fillOpacity:0.6});",
          "board.create('nonreflexangle', [Cq, B, A], {radius:0.8, name:'', strokeColor:'#F59E0B', fillColor:'#FDE68A', fillOpacity:0.6});",
          "board.create('nonreflexangle', [Q3, Q2, Q1], {radius:0.8, name:'', strokeColor:'#F59E0B', fillColor:'#FDE68A', fillOpacity:0.6});",
          "board.create('nonreflexangle', [A, Cq, B], {radius:0.8, name:'', strokeColor:'#7C3AED', fillColor:'#C4B5FD', fillOpacity:0.6});",
          "board.create('nonreflexangle', [Q1, Q3, Q2], {radius:0.8, name:'', strokeColor:'#7C3AED', fillColor:'#C4B5FD', fillOpacity:0.6});",
        ].join('\n') : "/* 角弧关闭 */"),
        "/* 底部提示区:背板 + 灰字 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖动 A/B/C 或橙色轴端点:轴对称是全等变换,对应边角始终相等'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'sss_congruence',
    group: GS,
    name: 'SSS:三边定形',
    emoji: '📐',
    desc: '三边长唯一确定三角形(含三边关系反例,尺规作图思想)',
    boundingBox: [-9, 9.5, 9, -7],
    keepAspectRatio: true,
    params: [
      { key: 'a', label: '边 a(BC)', type: 'number', min: 2, max: 7, step: 0.1, defaultValue: 6 },
      { key: 'b', label: '边 b(AC)', type: 'number', min: 1, max: 6, step: 0.1, defaultValue: 4 },
      { key: 'c', label: '边 c(AB)', type: 'number', min: 1, max: 6, step: 0.1, defaultValue: 5 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-9, 9.5, 9, -7])
      return [
        "/* 顶部控制区:背板 + 三行滑杆(左) + 状态读数(右) */",
        L.panel(3),
        "var sa = board.create('slider', [" + L.slider(0) + ", [2, " + n(p.a) + ", 7]], {name:'a', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var sb = board.create('slider', [" + L.slider(1) + ", [1, " + n(p.b) + ", 6]], {name:'b', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "var scc = board.create('slider', [" + L.slider(2) + ", [1, " + n(p.c) + ", 6]], {name:'c', " + sliderAttrs('#7C3AED', 0.1) + "});",
        "/* 底边 BC 固定在 x 轴上,B 在左端 */",
        "var Bp = board.create('point', [-3.5, 0], {name:'B', fixed:true, size:4, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:13}});",
        "var Cp = board.create('point', [function(){ return -3.5 + sa.Value(); }, 0], {name:'C', size:4, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:13}});",
        "/* 三边关系判定 + 余弦定理推顶点 A 相对 B 的坐标 */",
        "function ok(){ var a=sa.Value(), b=sb.Value(), c=scc.Value(); return (b+c>a+0.001)&&(a+c>b+0.001)&&(a+b>c+0.001); }",
        "function axo(){ var a=sa.Value(), b=sb.Value(), c=scc.Value(); return (c*c + a*a - b*b)/(2*a); }",
        "function ayo(){ var c=scc.Value(); var t=c*c - axo()*axo(); return t>0 ? Math.sqrt(t) : 0; }",
        "var A = board.create('point', [function(){ return -3.5 + axo(); }, ayo], {name:'A', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}, visible:function(){ return ok(); }});",
        "/* 尺规作图痕迹:分别以 B、C 为圆心,c、b 为半径画弧,交点即 A */",
        "board.create('circle', [Bp, function(){ return scc.Value(); }], {strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
        "board.create('circle', [Cp, function(){ return sb.Value(); }], {strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
        "board.create('polygon', [A, Bp, Cp], {fillColor:'#FDE68A', fillOpacity:0.5, borders:{strokeColor:'#B45309', strokeWidth:2.5}, visible:function(){ return ok(); }});",
        "/* 状态读数(控制区右半) */",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  if (ok()) return '✓ 两弧相交,三角形唯一确定';",
        "  return '⚠ 两边之和 ≤ 第三边,无法构成!';",
        "}], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "board.create('text', [" + L.midX + ", " + L.topY(1) + ", 'SSS:三边定形,'], {fontSize:13, strokeColor:'#6B7280'});",
        "board.create('text', [" + L.midX + ", " + L.topY(2) + ", '这是SSS全等判定的根源'], {fontSize:13, strokeColor:'#6B7280'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '尺规视角:以 B 为圆心作半径 c 的弧,以 C 为圆心作半径 b 的弧,交点定 A'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'shortest_path_reflect',
    group: GS,
    name: '将军饮马(最短路径)',
    emoji: '🐎',
    desc: '拖动河边的点 P 探索最短路线,再用轴对称揭示答案',
    boundingBox: [-10, 8, 10, -6],
    keepAspectRatio: true,
    params: [
      { key: 'showHelper', label: '显示对称辅助线(A′ 与最优点 P★)', type: 'boolean', defaultValue: true },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 8, 10, -6])
      return [
        "/* 河流:x 轴上一条蓝色粗线段 */",
        "var river = board.create('segment', [[-9.5, 0], [9.5, 0]], {fixed:true, strokeColor:'#0EA5E9', strokeWidth:4});",
        "board.create('text', [8.6, -0.7, '河'], {fontSize:14, strokeColor:'#0EA5E9', cssStyle:'font-weight:bold;'});",
        "var A = board.create('point', [-5, 4], {name:'A(军营)', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var B = board.create('point', [5.5, 2.8], {name:'B(营地)', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var P = board.create('glider', [0, 0, river], {name:'P(拖我)', size:5, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "board.create('segment', [A, P], {strokeColor:'#DC2626', strokeWidth:2.5});",
        "board.create('segment', [P, B], {strokeColor:'#DC2626', strokeWidth:2.5});",
        "function dist(U, V){ var dx = U.X()-V.X(), dy = U.Y()-V.Y(); return Math.sqrt(dx*dx+dy*dy); }",
        (p.showHelper ? [
          "/* A 关于河(x轴)的对称点 A′;A′B 连线与河的交点就是最优点 P★ */",
          "var A2 = board.create('point', [function(){ return A.X(); }, function(){ return -A.Y(); }], {name:'A′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
          "board.create('segment', [A, A2], {strokeColor:'#9CA3AF', strokeWidth:1, dash:2});",
          "board.create('segment', [A2, B], {strokeColor:'#059669', strokeWidth:2, dash:2});",
          "board.create('point', [function(){",
          "  var den = B.Y() - A2.Y(); if (Math.abs(den) < 0.001) return P.X();",
          "  return A2.X() + (B.X() - A2.X()) * (0 - A2.Y()) / den;",
          "}, 0], {name:'P★', size:4, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}, fixed:true});",
        ].join('\n') : "/* 辅助线关闭:先让学生自己拖 P 探索,再打开揭示 */"),
        "/* 顶部读数区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){",
        "  var cur = dist(A, P) + dist(P, B);",
        "  var best = Math.sqrt((B.X()-A.X())*(B.X()-A.X()) + (B.Y()+A.Y())*(B.Y()+A.Y()));",
        "  return '当前路程 AP+PB = ' + cur.toFixed(2) + '   理论最短 A′B = ' + best.toFixed(2) + (cur - best < 0.05 ? '   🎉 已达最短!' : '');",
        "}], {fontSize:15, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '化折为直:AP+PB = A′P+PB ≥ A′B,当 P 落在 A′B 与河的交点时取等号'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'symmetry_property',
    group: GS,
    name: '对称性质:垂直平分',
    emoji: '📍',
    desc: '对称轴垂直平分对应点连线;轴上任意点到两对称点等距',
    boundingBox: [-9, 9, 9, -9],
    keepAspectRatio: true,
    params: [
      { key: 'showGlider', label: '显示轴上动点 M(等距演示)', type: 'boolean', defaultValue: true },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-9, 9, 9, -9])
      const rows = p.showGlider ? 2 : 1
      return [
        "/* 对称轴:两个可拖端点(橙),初始略倾斜以示一般性 */",
        "var L1 = board.create('point', [-1.5, -6.5], {name:'', size:3, fillColor:'#F59E0B', strokeColor:'#F59E0B'});",
        "var L2 = board.create('point', [1.5, 5], {name:'', size:3, fillColor:'#F59E0B', strokeColor:'#F59E0B'});",
        "var axis = board.create('line', [L1, L2], {strokeColor:'#F59E0B', strokeWidth:2});",
        "var P = board.create('point', [5.5, 1], {name:'P(拖我)', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var P2 = board.create('reflection', [P, axis], {name:'P′', size:4, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "board.create('segment', [P, P2], {strokeColor:'#9CA3AF', strokeWidth:1.5, dash:2});",
        "/* 中点 M₀ 恰落在对称轴上 + 直角标记(PP′ ⊥ 轴) */",
        "var M0 = board.create('midpoint', [P, P2], {name:'M₀', size:3, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:12}});",
        "board.create('nonreflexangle', [L2, M0, P], {radius:0.5, name:'', type:'square', strokeColor:'#DC2626', fillColor:'#FECACA', fillOpacity:0.5});",
        "function dist(U, V){ var dx = U.X()-V.X(), dy = U.Y()-V.Y(); return Math.sqrt(dx*dx+dy*dy); }",
        "/* 顶部读数区 */",
        L.panel(rows),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){ return 'PM₀ = ' + dist(P, M0).toFixed(2) + ' = M₀P′ = ' + dist(M0, P2).toFixed(2) + ',且 PP′ ⊥ 轴 → 垂直平分'; }], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        (p.showGlider ? [
          "/* 轴上动点 M:到 P 与 P′ 的距离恒相等(线段垂直平分线性质) */",
          "var M = board.create('glider', [0.8, 2.5, axis], {name:'M(拖我)', size:4, fillColor:'#7C3AED', strokeColor:'#7C3AED', label:{fontSize:13}});",
          "board.create('segment', [M, P], {strokeColor:'#7C3AED', strokeWidth:2, dash:2});",
          "board.create('segment', [M, P2], {strokeColor:'#7C3AED', strokeWidth:2, dash:2});",
          "board.create('text', [" + L.leftX + ", " + L.topY(1) + ", function(){ return '轴上任取 M: MP = ' + dist(M, P).toFixed(2) + ' = MP′ = ' + dist(M, P2).toFixed(2); }], {fontSize:14, strokeColor:'#5B21B6', cssStyle:'font-weight:bold;'});",
        ].join('\n') : "/* 动点关闭 */"),
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '拖 P 或橙色轴端点:对称轴始终垂直平分对应点连线 PP′'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'rotation_congruence',
    group: GS,
    name: '旋转与全等',
    emoji: '🌀',
    desc: '绕中心旋转 θ 角:形状大小不变,OA=OA′、∠AOA′=θ',
    boundingBox: [-10, 10, 10, -10],
    keepAspectRatio: true,
    params: [
      { key: 'theta', label: '旋转角 θ(度)', type: 'number', min: -180, max: 180, step: 5, defaultValue: 75 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-10, 10, 10, -10])
      return [
        "/* 顶部控制区:滑杆(左) + 读数(右) */",
        L.panel(1),
        "var sth = board.create('slider', [" + L.slider(0) + ", [-180, " + n(p.theta) + ", 180]], {name:'θ°', " + sliderAttrs('#7C3AED', 5) + "});",
        "function rc(){ return sth.Value() * Math.PI / 180; }",
        "var O = board.create('point', [0, -1.5], {name:'O(旋转中心)', size:4, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}});",
        "/* 原三角形(蓝,可拖) */",
        "var A = board.create('point', [-6.5, -5], {name:'A', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var B = board.create('point', [-2.5, -6.5], {name:'B', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "var Cq = board.create('point', [-4, -2.5], {name:'C', size:4, fillColor:'#2563EB', strokeColor:'#2563EB', label:{fontSize:13}});",
        "board.create('polygon', [A, B, Cq], {fillColor:'#DBEAFE', fillOpacity:0.5, borders:{strokeColor:'#2563EB', strokeWidth:2}});",
        "/* 旋转像坐标工厂:绕 O 旋转 θ */",
        "function roX(P){ return function(){ return O.X() + (P.X()-O.X())*Math.cos(rc()) - (P.Y()-O.Y())*Math.sin(rc()); }; }",
        "function roY(P){ return function(){ return O.Y() + (P.X()-O.X())*Math.sin(rc()) + (P.Y()-O.Y())*Math.cos(rc()); }; }",
        "var A2 = board.create('point', [roX(A), roY(A)], {name:'A′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "var B2 = board.create('point', [roX(B), roY(B)], {name:'B′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "var C2 = board.create('point', [roX(Cq), roY(Cq)], {name:'C′', size:3, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "board.create('polygon', [A2, B2, C2], {fillColor:'#D1FAE5', fillOpacity:0.5, borders:{strokeColor:'#059669', strokeWidth:2}});",
        "/* A 的旋转轨迹圆(虚线) + 两条半径连线 + 旋转角弧 θ */",
        "board.create('circle', [O, A], {strokeColor:'#D1D5DB', strokeWidth:1, dash:2});",
        "board.create('segment', [O, A], {strokeColor:'#9CA3AF', strokeWidth:1.5, dash:2});",
        "board.create('segment', [O, A2], {strokeColor:'#9CA3AF', strokeWidth:1.5, dash:2});",
        "board.create('nonreflexangle', [A, O, A2], {radius:1.1, name:'θ', strokeColor:'#F59E0B', fillColor:'#FDE68A', fillOpacity:0.5, label:{fontSize:13}});",
        "function dist(U, V){ var dx = U.X()-V.X(), dy = U.Y()-V.Y(); return Math.sqrt(dx*dx+dy*dy); }",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){ return 'OA=' + dist(O,A).toFixed(2) + '=OA′=' + dist(O,A2).toFixed(2) + '  AB=' + dist(A,B).toFixed(2) + '=A′B′=' + dist(A2,B2).toFixed(2); }], {fontSize:13, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '旋转是全等变换:对应点到中心等距,对应边相等 → △ABC ≌ △A′B′C′(拖 O 或顶点验证)'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'rect_fold',
    group: GS,
    name: '矩形折叠模型',
    emoji: '📄',
    desc: '沿折痕 EF 翻折角 B:折叠即轴对称(中考折叠题基本图)',
    boundingBox: [-2.5, 8, 11.5, -3.5],
    keepAspectRatio: true,
    params: [],
    buildConstruction: () => {
      const L = makeLayout([-2.5, 8, 11.5, -3.5])
      return [
        "/* 矩形 ABCD(9×5,四角固定) */",
        "var Ar = board.create('point', [0, 0], {name:'A', fixed:true, size:3, fillColor:'#6B7280', strokeColor:'#6B7280', label:{fontSize:13}});",
        "var Br = board.create('point', [9, 0], {name:'B', fixed:true, size:4, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "var Cr = board.create('point', [9, 5], {name:'C', fixed:true, size:3, fillColor:'#6B7280', strokeColor:'#6B7280', label:{fontSize:13}});",
        "var Dr = board.create('point', [0, 5], {name:'D', fixed:true, size:3, fillColor:'#6B7280', strokeColor:'#6B7280', label:{fontSize:13}});",
        "var sAB = board.create('segment', [Ar, Br], {strokeColor:'#6B7280', strokeWidth:2});",
        "var sBC = board.create('segment', [Br, Cr], {strokeColor:'#6B7280', strokeWidth:2});",
        "board.create('segment', [Cr, Dr], {strokeColor:'#6B7280', strokeWidth:2});",
        "board.create('segment', [Dr, Ar], {strokeColor:'#6B7280', strokeWidth:2});",
        "/* 折痕两端:E 在 AB 边上滑动,F 在 BC 边上滑动 */",
        "var E = board.create('glider', [5.5, 0, sAB], {name:'E(拖我)', size:5, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}});",
        "var F = board.create('glider', [9, 3, sBC], {name:'F(拖我)', size:5, fillColor:'#F59E0B', strokeColor:'#F59E0B', label:{fontSize:13}});",
        "board.create('segment', [E, F], {strokeColor:'#F59E0B', strokeWidth:2.5});",
        "var lineEF = board.create('line', [E, F], {visible:false});",
        "/* B 关于折痕直线的对称点 B′ = 翻折后的落点 */",
        "var B2 = board.create('reflection', [Br, lineEF], {name:'B′(落点)', size:4, fillColor:'#059669', strokeColor:'#059669', label:{fontSize:13}});",
        "/* 原折叠区(黄虚)与翻折后的像(绿实):二者全等 */",
        "board.create('polygon', [E, Br, F], {fillColor:'#FDE68A', fillOpacity:0.3, borders:{strokeColor:'#B45309', strokeWidth:1.5, dash:2}, vertices:{visible:false}});",
        "board.create('polygon', [E, B2, F], {fillColor:'#A7F3D0', fillOpacity:0.5, borders:{strokeColor:'#059669', strokeWidth:2}, vertices:{visible:false}});",
        "function dist(U, V){ var dx = U.X()-V.X(), dy = U.Y()-V.Y(); return Math.sqrt(dx*dx+dy*dy); }",
        "/* 顶部读数区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){ return 'EB=' + dist(E,Br).toFixed(2) + '=EB′=' + dist(E,B2).toFixed(2) + '   FB=' + dist(F,Br).toFixed(2) + '=FB′=' + dist(F,B2).toFixed(2); }], {fontSize:14, strokeColor:'#1F2937', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '折叠 = 关于折痕的轴对称:翻折前后三角形全等。拖动 E、F 观察落点 B′'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },
]

/**
 * mathGraphTemplatesPrimary.ts — 小学模板(批次1c-2首发;2c版式;3a关轴;3b排版修复,2026-07-08)
 *
 * 批次3b 修复(操作员反馈的三个遗留问题):
 *   1. 角的认识:图形移到画板中部居中,顶点改为可拖(两边随顶点联动),老师可整体挪位置;
 *   2. 平行四边形割补:整体右移 1.5 居中(关轴后原点不再有视觉锚定,图形显得偏左);
 *   3. 条形统计图:手绘刻度线 layer:1 沉底(原默认图层压在柱子上方),
 *      并补深色底边基线(关轴后系统 x 轴不再提供底边)。
 *
 * 小学模板设计原则:参数少而直观,交互以"拖动看变化"为主,读数用大字号,
 * 避免出现超纲符号。8个无坐标语义模板 showAxis:false(批次3a)。
 * 编写规范见聚合出口 mathGraphTemplates.ts 文件头。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/mathGraphTemplatesPrimary.ts
 */
import type { MathGraphTemplate } from './mathGraphUtils'
import { n, sliderAttrs, makeLayout } from './mathGraphTemplateShared'

const G = '🧒 小学'

export const MATH_GRAPH_TEMPLATES_PRIMARY: MathGraphTemplate[] = [

  {
    id: 'fraction_circle',
    showAxis: false,  // 批次3a:无坐标语义,系统坐标轴默认关闭(弹窗开关可开)
    group: G,
    name: '分数的认识(圆形模型)',
    emoji: '🍕',
    desc: '拖动滑杆改变分母与分子,涂色部分就是几分之几',
    boundingBox: [-6, 6, 6, -6],
    keepAspectRatio: true,
    params: [
      { key: 'den', label: '分母(平均分成几份)', type: 'number', min: 2, max: 12, step: 1, defaultValue: 8 },
      { key: 'num', label: '分子(取其中几份)', type: 'number', min: 0, max: 12, step: 1, defaultValue: 3, hint: '超过分母时自动按分母封顶' },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-6, 6, 6, -6])
      return [
        "/* 顶部控制区:面板 + 两根滑杆(左) + 大字分数读数(右) */",
        L.panel(2),
        "var sd = board.create('slider', [" + L.slider(0) + ", [2, " + n(p.den) + ", 12]], {name:'分母', " + sliderAttrs('#7C3AED', 1) + "});",
        "var sn = board.create('slider', [" + L.slider(1) + ", [0, " + n(p.num) + ", 12]], {name:'分子', " + sliderAttrs('#DC2626', 1) + "});",
        "function den(){ return Math.round(sd.Value()); }",
        "function num(){ return Math.min(Math.round(sn.Value()), den()); }",
        "var R = 3.0;",
        "/* 圆心原点偏下,只建一次(圆整体下移给面板让位) */",
        "var Oc = board.create('point', [0, -1], {visible:false});",
        "/* 预建12个扇形,按当前分母控制可见性与涂色(份数动态变化的标准做法) */",
        "for (var i = 0; i < 12; i++) {",
        "  (function(idx){",
        "    var pS = board.create('point', [function(){ return R*Math.cos(2*Math.PI*idx/den()); }, function(){ return -1 + R*Math.sin(2*Math.PI*idx/den()); }], {visible:false});",
        "    var pE = board.create('point', [function(){ return R*Math.cos(2*Math.PI*(idx+1)/den()); }, function(){ return -1 + R*Math.sin(2*Math.PI*(idx+1)/den()); }], {visible:false});",
        "    board.create('sector', [Oc, pS, pE], {",
        "      visible: function(){ return idx < den(); },",
        "      fillColor: '#FBBF24', strokeColor:'#B45309', strokeWidth:1.5,",
        "      fillOpacity: function(){ return idx < num() ? 0.85 : 0.08; }",
        "    });",
        "  })(i);",
        "}",
        "board.create('text', [" + L.midX + " + 0.5, " + L.topY(0) + " - 0.5, function(){",
        "  return '涂色部分 = ' + num() + ' / ' + den();",
        "}], {fontSize:22, strokeColor:'#B45309', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '整个圆平均分成「分母」份,涂色「分子」份'], {fontSize:14, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'angle_size',
    showAxis: false,  // 批次3a:无坐标语义,系统坐标轴默认关闭(弹窗开关可开)
    group: G,
    name: '角的认识(大小与边长无关)',
    emoji: '📐',
    desc: '分别拖动"张口"和"边长",发现角的大小只看张口',
    boundingBox: [-2, 8, 10, -2],
    keepAspectRatio: true,
    params: [
      { key: 'deg', label: '角的张口(度)', type: 'number', min: 10, max: 170, step: 5, defaultValue: 45 },
      { key: 'len', label: '边的长度', type: 'number', min: 2, max: 7, step: 0.5, defaultValue: 4 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-2, 8, 10, -2])
      return [
        "/* 顶部控制区:面板 + 两根滑杆(左) + 大字度数读数(右) */",
        L.panel(2),
        "var sdeg = board.create('slider', [" + L.slider(0) + ", [10, " + n(p.deg) + ", 170]], {name:'张口', " + sliderAttrs('#7C3AED', 5) + "});",
        "var slen = board.create('slider', [" + L.slider(1) + ", [2, " + n(p.len) + ", 7]], {name:'边长', " + sliderAttrs('#059669', 0.5) + "});",
        "function rad(){ return sdeg.Value() * Math.PI / 180; }",
        "/* 批次3b:顶点居中偏左下且可拖,两边随顶点联动——老师可整体挪动整个角 */",
        "var V = board.create('point', [1.5, 0.3], {name:'顶点(拖我挪位置)', size:4, fillColor:'#1F2937', strokeColor:'#1F2937', label:{fontSize:12, offset:[-8, -16]}});",
        "/* 一边固定水平,另一边随张口转动;两边坐标相对顶点计算,长度随边长滑杆伸缩 */",
        "var E1 = board.create('point', [function(){ return V.X() + slen.Value(); }, function(){ return V.Y(); }], {visible:false});",
        "var E2 = board.create('point', [function(){ return V.X() + slen.Value()*Math.cos(rad()); }, function(){ return V.Y() + slen.Value()*Math.sin(rad()); }], {visible:false});",
        "board.create('segment', [V, E1], {strokeColor:'#2563EB', strokeWidth:3});",
        "board.create('segment', [V, E2], {strokeColor:'#2563EB', strokeWidth:3});",
        "board.create('angle', [E1, V, E2], {radius:1.1, name:'', strokeColor:'#F59E0B', fillColor:'#FDE68A', fillOpacity:0.6});",
        "board.create('text', [" + L.midX + " + 1, " + L.topY(0) + " - 0.4, function(){",
        "  return '角 = ' + sdeg.Value().toFixed(0) + '°';",
        "}], {fontSize:22, strokeColor:'#B45309', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '把边拉长或缩短,角的度数变了吗?——角的大小与边的长短无关!'], {fontSize:14, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'parallelogram_area',
    showAxis: false,  // 批次3a:无坐标语义,系统坐标轴默认关闭(弹窗开关可开)
    group: G,
    name: '平行四边形面积(割补法)',
    emoji: '✂️',
    desc: '拖动进度条,把切下的三角形补到另一边,变成长方形',
    boundingBox: [-3, 8, 12, -3],
    keepAspectRatio: true,
    params: [
      { key: 'a', label: '底 a', type: 'number', min: 3, max: 7, step: 0.5, defaultValue: 5 },
      { key: 'h', label: '高 h', type: 'number', min: 2, max: 5, step: 0.5, defaultValue: 3 },
      { key: 's', label: '倾斜程度', type: 'number', min: 1, max: 3, step: 0.5, defaultValue: 2 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-3, 8, 12, -3])
      return [
        "/* 顶部控制区:面板 + 割补进度条(左) + 状态读数(右) */",
        L.panel(1),
        "var st = board.create('slider', [" + L.slider(0) + ", [0, 0, 1]], {name:'割补进度(拖我)', " + sliderAttrs('#DC2626', 0.02) + "});",
        "/* 批次3b:X0 整体右移 1.5 居中(关轴后原点无视觉锚定,原图形显得偏左) */",
        "var A = " + n(p.a) + ", H = " + n(p.h) + ", S = " + n(p.s) + ", X0 = 1.5;",
        "function t(){ return st.Value(); }",
        "/* 原平行四边形虚线轮廓 */",
        "board.create('polygon', [[X0,0],[X0+A,0],[X0+A+S,H],[X0+S,H]], {fillOpacity:0, borders:{strokeColor:'#9CA3AF', strokeWidth:1.5, dash:2}, vertices:{visible:false}});",
        "/* 固定的主体梯形 */",
        "board.create('polygon', [[X0+S,0],[X0+A,0],[X0+A+S,H],[X0+S,H]], {fillColor:'#93C5FD', fillOpacity:0.55, borders:{strokeColor:'#2563EB', strokeWidth:2}, vertices:{visible:false}});",
        "/* 被切下的三角形,随进度整体向右平移 A */",
        "board.create('polygon', [",
        "  [function(){ return X0 + t()*A; }, 0],",
        "  [function(){ return X0 + S + t()*A; }, 0],",
        "  [function(){ return X0 + S + t()*A; }, H]",
        "], {fillColor:'#FBBF24', fillOpacity:0.75, borders:{strokeColor:'#B45309', strokeWidth:2}, vertices:{visible:false}});",
        "/* 高的标注线 */",
        "board.create('segment', [[X0+S, 0], [X0+S, H]], {strokeColor:'#DC2626', strokeWidth:1.5, dash:2});",
        "board.create('text', [function(){ return X0 + S - 0.6; }, H/2, '高h'], {fontSize:14, strokeColor:'#DC2626'});",
        "board.create('text', [X0 + A/2 + S/2, -0.8, '底a'], {fontSize:14, strokeColor:'#2563EB'});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  if (t() >= 0.99) return '拼成长方形! 面积 = 底 × 高 = ' + A + ' × ' + H + ' = ' + (A*H);",
        "  return '平行四边形面积 = ?  (拖动进度条)';",
        "}], {fontSize:15, strokeColor:'#B45309', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '割补法:切下的黄色三角形补到右边,面积不变,形状变成长方形'], {fontSize:13, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'circle_circumference',
    showAxis: false,  // 批次3a:无坐标语义,系统坐标轴默认关闭(弹窗开关可开)
    group: G,
    name: '圆的周长(滚动测量)',
    emoji: '🛞',
    desc: '让圆沿直线滚一整圈,滚过的距离正好是 π 乘直径',
    boundingBox: [-2, 8, 24, -4],
    keepAspectRatio: true,
    params: [
      { key: 'r', label: '半径 r', type: 'number', min: 1, max: 3, step: 0.5, defaultValue: 2 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-2, 8, 24, -4])
      return [
        "/* 顶部控制区:面板 + 滚动进度条(左) + 距离读数(右) */",
        L.panel(1),
        "var st = board.create('slider', [" + L.slider(0) + ", [0, 0, 1]], {name:'滚动(拖我)', " + sliderAttrs('#DC2626', 0.01) + "});",
        "var R = " + n(p.r) + ";",
        "function t(){ return st.Value(); }",
        "function cx(){ return 2*Math.PI*R*t(); }",
        "/* 地面 */",
        "board.create('segment', [[-1, 0], [23, 0]], {strokeColor:'#6B7280', strokeWidth:2, fixed:true});",
        "/* 滚动的圆:圆心 (走过距离, R) */",
        "var O = board.create('point', [cx, R], {visible:false});",
        "board.create('circle', [O, R], {strokeColor:'#2563EB', strokeWidth:2.5, fillColor:'#DBEAFE', fillOpacity:0.3});",
        "/* 圆周上的标记点:滚动时绕圆心反向转动(纯滚动无滑动),起点在接地处 */",
        "board.create('point', [",
        "  function(){ return cx() + R*Math.sin(-2*Math.PI*t()); },",
        "  function(){ return R - R*Math.cos(-2*Math.PI*t()); }",
        "], {name:'标记点', size:4, fillColor:'#DC2626', strokeColor:'#DC2626', label:{fontSize:13}});",
        "/* 已滚过的距离(红色粗线沿地面) */",
        "board.create('segment', [[0, 0], [cx, 0]], {strokeColor:'#DC2626', strokeWidth:4});",
        "board.create('text', [" + L.midX + ", " + L.topY(0) + ", function(){",
        "  var d = cx();",
        "  if (t() >= 0.995) return '滚了一整圈! 周长 = π × 直径 = 3.14 × ' + (2*R) + ' ≈ ' + (2*Math.PI*R).toFixed(2);",
        "  return '已滚过距离 ≈ ' + d.toFixed(2);",
        "}], {fontSize:15, strokeColor:'#B45309', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '红色标记点转回最低点时,恰好滚了一整圈'], {fontSize:14, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'number_line',
    showAxis: false,  // 批次3a:自带手绘数轴,系统坐标轴默认关闭(弹窗开关可开)
    group: G,
    name: '数轴与数的认识',
    emoji: '🌡️',
    desc: '拖动数轴上的点认识正数、负数、小数的位置',
    boundingBox: [-11, 4, 11, -4],
    keepAspectRatio: false,
    params: [
      { key: 'snap', label: '吸附步长', type: 'number', min: 0.5, max: 1, step: 0.5, defaultValue: 0.5, hint: '0.5=可停在半格(小数),1=只停整数' },
      { key: 'start', label: '点的初始位置', type: 'number', min: -9, max: 9, step: 0.5, defaultValue: 2.5 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-11, 4, 11, -4])
      return [
        "/* 数轴主线(带箭头) */",
        "var base = board.create('line', [[-10, 0], [10, 0]], {straightFirst:false, straightLast:false, firstArrow:true, lastArrow:true, strokeColor:'#1F2937', strokeWidth:2.5, fixed:true});",
        "/* 整数刻度与标签 */",
        "for (var i = -9; i <= 9; i++) {",
        "  board.create('segment', [[i, -0.18], [i, 0.18]], {strokeColor:'#1F2937', strokeWidth: i === 0 ? 3 : 1.5, fixed:true});",
        "  board.create('text', [i, -0.75, String(i)], {fontSize: i === 0 ? 16 : 13, strokeColor: i === 0 ? '#DC2626' : '#1F2937', anchorX:'middle', cssStyle: i === 0 ? 'font-weight:bold;' : ''});",
        "}",
        "/* 可拖动的点,吸附到设定步长 */",
        "var P = board.create('glider', [" + n(p.start) + ", 0, base], {name:'拖我', size:5, fillColor:'#DC2626', strokeColor:'#DC2626', snapWidth:" + n(p.snap) + ", label:{fontSize:14, offset:[0, 18]}});",
        "/* 顶部读数区 */",
        L.panel(1),
        "board.create('text', [" + L.leftX + ", " + L.topY(0) + ", function(){",
        "  var v = Math.round(P.X() / " + n(p.snap) + ") * " + n(p.snap) + ";",
        "  var kind = v > 0 ? '正数' : (v < 0 ? '负数' : '零(正负分界)');",
        "  var side = v > 0 ? '在 0 的右边' : (v < 0 ? '在 0 的左边' : '');",
        "  return '当前的数: ' + v + '   它是' + kind + (side ? ',' + side : '');",
        "}], {fontSize:17, strokeColor:'#B45309', cssStyle:'font-weight:bold;'});",
        "/* 底部提示区 */",
        L.hintPanel(),
        "board.create('text', [" + L.leftX + ", " + L.hintY + ", '数轴三要素:原点、正方向、单位长度'], {fontSize:14, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },

  {
    id: 'bar_chart',
    showAxis: false,  // 批次3a:自带刻度体系,系统坐标轴默认关闭(弹窗开关可开)
    group: G,
    name: '条形统计图',
    emoji: '📊',
    desc: '拖动滑杆改变各组数据,条形高度实时变化',
    boundingBox: [-3, 13, 14, -2],
    keepAspectRatio: false,
    params: [
      { key: 'v1', label: '第1组:春', type: 'number', min: 0, max: 10, step: 1, defaultValue: 6 },
      { key: 'v2', label: '第2组:夏', type: 'number', min: 0, max: 10, step: 1, defaultValue: 9 },
      { key: 'v3', label: '第3组:秋', type: 'number', min: 0, max: 10, step: 1, defaultValue: 4 },
      { key: 'v4', label: '第4组:冬', type: 'number', min: 0, max: 10, step: 1, defaultValue: 7 },
    ],
    buildConstruction: (p) => {
      const L = makeLayout([-3, 13, 14, -2])
      return [
        "/* 顶部控制区:面板两行——滑杆保留'各自条形正上方'的特色布局(第一行),结论第二行 */",
        L.panel(2),
        "var names = ['春', '夏', '秋', '冬'];",
        "var inits = [" + n(p.v1) + ", " + n(p.v2) + ", " + n(p.v3) + ", " + n(p.v4) + "];",
        "var colors = ['#60A5FA', '#F87171', '#FBBF24', '#34D399'];",
        "var strokes = ['#2563EB', '#DC2626', '#B45309', '#059669'];",
        "/* y 轴刻度网格线(批次3b:layer:1 沉底,不再压在柱子上方) */",
        "for (var g = 0; g <= 10; g += 2) {",
        "  board.create('segment', [[0, g], [13, g]], {strokeColor:'#E5E7EB', strokeWidth:1, fixed:true, layer:1});",
        "  board.create('text', [-0.9, g, String(g)], {fontSize:12, strokeColor:'#6B7280'});",
        "}",
        "/* 底边基线(批次3b:关轴后系统 x 轴不再提供底边,自绘深色基线) */",
        "board.create('segment', [[-0.3, 0], [13.3, 0]], {strokeColor:'#4B5A6E', strokeWidth:2.5, fixed:true, layer:2});",
        "var sliders = [];",
        "/* 每组:面板首行一根滑杆(位于各自条形正上方) + 一根条形(高度绑定滑杆) */",
        "for (var i = 0; i < 4; i++) {",
        "  (function(idx){",
        "    var x0 = 1 + idx * 3;",
        "    var s = board.create('slider', [[x0, " + L.topY(0) + "], [x0 + 2, " + L.topY(0) + "], [0, inits[idx], 10]], {name:names[idx], " + sliderAttrs('#7C3AED', 1) + "});",
        "    sliders.push(s);",
        "    board.create('polygon', [",
        "      [x0, 0], [x0 + 2, 0],",
        "      [x0 + 2, function(){ return s.Value(); }],",
        "      [x0, function(){ return s.Value(); }]",
        "    ], {fillColor:colors[idx], fillOpacity:0.75, borders:{strokeColor:strokes[idx], strokeWidth:2}, vertices:{visible:false}});",
        "    /* 条形顶部数值 */",
        "    board.create('text', [x0 + 1, function(){ return s.Value() + 0.5; }, function(){ return String(Math.round(s.Value())); }], {fontSize:15, strokeColor:strokes[idx], anchorX:'middle', cssStyle:'font-weight:bold;'});",
        "    board.create('text', [x0 + 1, -0.7, names[idx]], {fontSize:14, strokeColor:'#1F2937', anchorX:'middle'});",
        "  })(i);",
        "}",
        "board.create('text', [" + L.leftX + ", " + L.topY(1) + ", function(){",
        "  var mx = 0, mi = 0;",
        "  for (var j = 0; j < 4; j++) { if (sliders[j].Value() > mx) { mx = sliders[j].Value(); mi = j; } }",
        "  return '最喜欢 ' + names[mi] + ' 的人最多(' + Math.round(mx) + ' 人)';",
        "}], {fontSize:14, strokeColor:'#6B7280'});",
      ].join('\n')
    },
  },
]

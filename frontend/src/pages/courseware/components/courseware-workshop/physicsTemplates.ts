/**
 * physicsTemplates.ts — 物理场景模板注册表(批次3首发;批次3c课标扩充聚合,2026-07-09)
 *
 * 场景清单(首发6 + 扩充9 = 15个;扩充模板在 physicsTemplatesExt.ts):
 *   🍎 运动与力   → 自由落体 / 平抛与斜抛 / 斜面滑块 (+摩擦对比赛跑/抛射角对比/平抛vs自由落体)
 *   🔗 振动与摆   → 单摆 / 弹簧振子 (+双摆)
 *   💥 碰撞与动量 → 动量碰撞 (+牛顿摆/二维斜碰台球)
 *   ⚡ 能量与热   → (Ext新组)弹跳与能量损失/分子热运动/多米诺能量传递
 * 杠杆/浮力等 Matter.js 表现一般的场景刻意不做。
 *
 * 新增模板规范(新增必读):
 *   1. buildSetup 产出的代码可用变量:Matter / engine / world / W / H,
 *      职责是设置 engine.gravity + 创建全部刚体约束(含地面墙体)并 add 进 world;
 *   2. 代码必须是【纯构造】——每次重置都会 Composite.clear 后重跑,
 *      禁止依赖任何外部/上次运行残留的状态;
 *   3. 位置尺寸一律基于 W/H 相对计算(运行时求值),同一模板适配三档画布尺寸;
 *   4. 教学参数在 TS 侧经 n() 插值成字面量烘焙进代码(限制小数位防浮点长尾);
 *   5. 字符串一律用单引号;禁止出现 "</script" 字样;
 *   6. 静态体统一石板灰 #94A3B8,运动主体用珊瑚粉主题色系
 *      (蓝 #6C9BF0 / 珊瑚红 #EE7B70 / 薄荷绿 #5BBFA5 / 薰衣草紫 #9B8AE6),
 *      与数学组件的批次1i色板同源,跨学科视觉一致;
 *   7. Matter 碰撞参数合成:friction 取较小值、restitution 取较大值——
 *      地面 friction 设1/restitution 设0,可让运动体自带参数生效;
 *   8. 对比类多体同点出发时用 collisionFilter:{group:-1} 使其互不碰撞。
 * 分组并组规则:扩充条目沿用已有分组名即自动并入该组(getGroups 按组名归并)。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/physicsTemplates.ts
 */
import type { PhysicsTemplate } from './physicsUtils'
import { PHYSICS_TEMPLATES_EXT } from './physicsTemplatesExt'

// ============================================================
// 共享辅助
// ============================================================

/** 数值插值:限制到3位小数并去长尾零,防浮点长尾污染生成代码(范式同 mathGraphTemplateShared.n) */
function n(v: number): string {
  return parseFloat(v.toFixed(3)).toString()
}

/** 静态体(地面/墙/斜面)统一填色 */
const STATIC_FILL = "render:{fillStyle:'#94A3B8'}"

// ============================================================
// 首发模板注册表(6个;文件尾聚合 Ext 扩充9个)
// ============================================================

const PHYSICS_BASE: PhysicsTemplate[] = [

  // ---------- 🍎 运动与力 ----------
  {
    id: 'phys-freefall', group: '🍎 运动与力', name: '自由落体', emoji: '🍎',
    desc: '轻重双球同高下落,无空气阻力时同时落地(伽利略实验)',
    params: [
      { key: 'g', label: '重力强度', type: 'number', min: 0.3, max: 2, step: 0.1, defaultValue: 1, hint: '1 为标准重力,调小模拟月球等低重力环境' },
      { key: 'fa', label: '空气阻力', type: 'number', min: 0, max: 0.05, step: 0.005, defaultValue: 0, hint: '0 时轻重双球同时落地;调大后轻球明显滞后' },
      { key: 'two', label: '轻重双球对比', type: 'boolean', defaultValue: true, hint: '关闭则只保留一个球' },
    ],
    buildSetup: (p) => [
      "engine.gravity.y = " + n(Number(p.g)) + ";",
      "/* 地面 */",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, H-12, W, 24, {isStatic:true, " + STATIC_FILL + "}));",
      "/* 重球(大):蓝 */",
      "Matter.Composite.add(world, Matter.Bodies.circle(W*" + (p.two ? '0.62' : '0.5') + ", 60, 26, {frictionAir:" + n(Number(p.fa)) + ", restitution:0.15, render:{fillStyle:'#6C9BF0'}}));",
      ...(p.two ? [
        "/* 轻球(小):珊瑚红——半径小、质量小,受同样空气阻力时滞后更明显 */",
        "Matter.Composite.add(world, Matter.Bodies.circle(W*0.38, 60, 13, {frictionAir:" + n(Number(p.fa)) + ", restitution:0.15, render:{fillStyle:'#EE7B70'}}));",
      ] : []),
    ].join('\n'),
  },
  {
    id: 'phys-projectile', group: '🍎 运动与力', name: '平抛与斜抛', emoji: '🏹',
    desc: '以给定初速与角度抛出,观察抛物线轨迹(角度0即平抛)',
    params: [
      { key: 'v', label: '初速度', type: 'number', min: 5, max: 25, step: 1, defaultValue: 14, hint: '出手速度大小' },
      { key: 'angle', label: '抛射角(°)', type: 'number', min: 0, max: 80, step: 5, defaultValue: 45, hint: '0°=水平平抛;45°附近射程最远' },
      { key: 'g', label: '重力强度', type: 'number', min: 0.3, max: 2, step: 0.1, defaultValue: 1 },
    ],
    buildSetup: (p) => {
      const rad = (Number(p.angle) * Math.PI) / 180
      const vx = Number(p.v) * Math.cos(rad)
      const vy = Number(p.v) * Math.sin(rad)
      return [
        "engine.gravity.y = " + n(Number(p.g)) + ";",
        "/* 地面 + 右墙(接住远射) */",
        "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, H-12, W, 24, {isStatic:true, " + STATIC_FILL + "}));",
        "Matter.Composite.add(world, Matter.Bodies.rectangle(W-8, H/2, 16, H, {isStatic:true, " + STATIC_FILL + "}));",
        "/* 发射台(左下角小平台) */",
        "Matter.Composite.add(world, Matter.Bodies.rectangle(56, H-58, 84, 14, {isStatic:true, " + STATIC_FILL + "}));",
        "/* 抛体:创建后立即赋予初速度(x正向右,y负向上) */",
        "var ball = Matter.Bodies.circle(56, H-84, 14, {restitution:0.4, frictionAir:0, render:{fillStyle:'#EE7B70'}});",
        "Matter.Composite.add(world, ball);",
        "Matter.Body.setVelocity(ball, {x:" + n(vx) + ", y:" + n(-vy) + "});",
      ].join('\n')
    },
  },
  {
    id: 'phys-incline', group: '🍎 运动与力', name: '斜面滑块', emoji: '⛷️',
    desc: '滑块沿斜面下滑,调摩擦系数观察加速下滑/匀速/静止的临界变化',
    params: [
      { key: 'angle', label: '斜面倾角(°)', type: 'number', min: 10, max: 45, step: 1, defaultValue: 30 },
      { key: 'mu', label: '摩擦系数', type: 'number', min: 0, max: 1, step: 0.05, defaultValue: 0.3, hint: 'tan(倾角)附近是滑与不滑的临界值,可让学生现场找临界' },
    ],
    buildSetup: (p) => {
      const a = (Number(p.angle) * Math.PI) / 180
      return [
        "engine.gravity.y = 1;",
        "var a = " + n(a) + "; /* 斜面倾角(弧度),正角=右端向下倾斜 */",
        "/* 地面 */",
        "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, H-12, W, 24, {isStatic:true, " + STATIC_FILL + "}));",
        "/* 斜面:旋转的静态长条,中心悬于画布中部 */",
        "var L = W*0.72;",
        "var ramp = Matter.Bodies.rectangle(W*0.46, H*0.55, L, 20, {isStatic:true, angle:a, friction:" + n(Number(p.mu)) + ", " + STATIC_FILL + "});",
        "Matter.Composite.add(world, ramp);",
        "/* 滑块:放置在斜面高端(沿斜面方向上移),并沿法向抬升贴面 */",
        "var t = L*0.36, h = 27; /* t=沿面偏移量, h=半厚10+半块16+缝1 */",
        "var bx = W*0.46 - t*Math.cos(a) + h*Math.sin(a);",
        "var by = H*0.55 - t*Math.sin(a) - h*Math.cos(a);",
        "var block = Matter.Bodies.rectangle(bx, by, 32, 32, {angle:a, friction:" + n(Number(p.mu)) + ", frictionStatic:" + n(Number(p.mu)) + ", restitution:0, render:{fillStyle:'#6C9BF0'}});",
        "Matter.Composite.add(world, block);",
      ].join('\n')
    },
  },

  // ---------- 🔗 振动与摆 ----------
  {
    id: 'phys-pendulum', group: '🔗 振动与摆', name: '单摆', emoji: '🕰️',
    desc: '从初始摆角释放,观察周期运动;调摆长与重力看周期变化',
    params: [
      { key: 'len', label: '摆长', type: 'number', min: 100, max: 260, step: 10, defaultValue: 180, hint: '摆长越长周期越长(与质量无关)' },
      { key: 'angle0', label: '初始摆角(°)', type: 'number', min: 10, max: 80, step: 5, defaultValue: 40 },
      { key: 'g', label: '重力强度', type: 'number', min: 0.3, max: 2, step: 0.1, defaultValue: 1, hint: '重力越强周期越短' },
      { key: 'damp', label: '空气阻尼', type: 'number', min: 0, max: 0.02, step: 0.002, defaultValue: 0, hint: '0 为理想单摆;调大观察振幅衰减' },
    ],
    buildSetup: (p) => {
      const a0 = (Number(p.angle0) * Math.PI) / 180
      return [
        "engine.gravity.y = " + n(Number(p.g)) + ";",
        "var L = " + n(Number(p.len)) + ", a0 = " + n(a0) + ";",
        "var px = W/2, py = 56; /* 悬点 */",
        "/* 悬点标记(静态小方块) */",
        "Matter.Composite.add(world, Matter.Bodies.rectangle(px, py, 14, 14, {isStatic:true, " + STATIC_FILL + "}));",
        "/* 摆球:从初始摆角位置静止释放 */",
        "var bob = Matter.Bodies.circle(px + L*Math.sin(a0), py + L*Math.cos(a0), 20, {frictionAir:" + n(Number(p.damp)) + ", render:{fillStyle:'#9B8AE6'}});",
        "Matter.Composite.add(world, bob);",
        "/* 刚性摆线:stiffness=1 的约束(可视为不可伸长细绳) */",
        "Matter.Composite.add(world, Matter.Constraint.create({pointA:{x:px,y:py}, bodyB:bob, length:L, stiffness:1, render:{strokeStyle:'#94A3B8', lineWidth:2}}));",
      ].join('\n')
    },
  },
  {
    id: 'phys-spring', group: '🔗 振动与摆', name: '弹簧振子', emoji: '🪀',
    desc: '水平光滑面上的弹簧振子,拉离平衡位置释放做简谐振动',
    params: [
      { key: 'k', label: '弹簧劲度', type: 'number', min: 0.005, max: 0.05, step: 0.005, defaultValue: 0.02, hint: '劲度越大振动越快' },
      { key: 'x0', label: '初始位移', type: 'number', min: 40, max: 160, step: 10, defaultValue: 100, hint: '拉离平衡位置的距离(即振幅)' },
      { key: 'damp', label: '阻尼', type: 'number', min: 0, max: 0.05, step: 0.005, defaultValue: 0, hint: '0 为理想简谐;调大观察阻尼振动衰减' },
    ],
    buildSetup: (p) => [
      "engine.gravity.y = 0; /* 水平方向振动,关闭重力(等效光滑水平面俯视) */",
      "var wallX = 60, restX = W*0.52, cy = H/2;",
      "/* 左侧固定墙 */",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(wallX-20, cy, 24, 160, {isStatic:true, " + STATIC_FILL + "}));",
      "/* 平衡位置参考线(静态细杆,仅视觉参考) */",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(restX, cy+90, 2, 60, {isStatic:true, isSensor:true, render:{fillStyle:'#E5E7EB'}}));",
      "/* 振子:从平衡位置右移 x0 处静止释放 */",
      "var box = Matter.Bodies.rectangle(restX + " + n(Number(p.x0)) + ", cy, 44, 44, {friction:0, frictionAir:" + n(Number(p.damp)) + ", render:{fillStyle:'#5BBFA5'}});",
      "Matter.Composite.add(world, box);",
      "/* 弹簧:自然长度=墙到平衡位置的距离,stiffness 即劲度 */",
      "Matter.Composite.add(world, Matter.Constraint.create({pointA:{x:wallX,y:cy}, bodyB:box, length:restX-wallX, stiffness:" + n(Number(p.k)) + ", damping:0, render:{strokeStyle:'#9B8AE6', lineWidth:3}}));",
    ].join('\n'),
  },

  // ---------- 💥 碰撞与动量 ----------
  {
    id: 'phys-collision', group: '💥 碰撞与动量', name: '动量碰撞', emoji: '💥',
    desc: '一维对心碰撞:动球撞静球,调质量比与恢复系数看弹性/非弹性碰撞',
    params: [
      { key: 'rest', label: '恢复系数', type: 'number', min: 0, max: 1, step: 0.05, defaultValue: 1, hint: '1=完全弹性碰撞;0=完全非弹性(碰后同速)' },
      { key: 'm1', label: '动球质量', type: 'number', min: 1, max: 5, step: 0.5, defaultValue: 1 },
      { key: 'm2', label: '静球质量', type: 'number', min: 1, max: 5, step: 0.5, defaultValue: 3, hint: '等质量弹性碰撞会发生速度交换' },
      { key: 'v1', label: '动球初速', type: 'number', min: 2, max: 12, step: 1, defaultValue: 8 },
    ],
    buildSetup: (p) => {
      /* 半径随质量增大(视觉体现质量差异),线性近似即可 */
      const r1 = 13 + Number(p.m1) * 4
      const r2 = 13 + Number(p.m2) * 4
      return [
        "engine.gravity.y = 0; /* 一维水平碰撞,关闭重力,球沿中线运动 */",
        "var cy = H/2;",
        "/* 左右边界墙(恢复系数与球一致,球弹回后可再次观察) */",
        "Matter.Composite.add(world, Matter.Bodies.rectangle(8, cy, 16, H, {isStatic:true, restitution:" + n(Number(p.rest)) + ", " + STATIC_FILL + "}));",
        "Matter.Composite.add(world, Matter.Bodies.rectangle(W-8, cy, 16, H, {isStatic:true, restitution:" + n(Number(p.rest)) + ", " + STATIC_FILL + "}));",
        "/* 动球(蓝,自左向右):无摩擦无空气阻力,保证碰前匀速 */",
        "var b1 = Matter.Bodies.circle(W*0.22, cy, " + n(r1) + ", {friction:0, frictionAir:0, restitution:" + n(Number(p.rest)) + ", render:{fillStyle:'#6C9BF0'}});",
        "Matter.Body.setMass(b1, " + n(Number(p.m1)) + ");",
        "Matter.Composite.add(world, b1);",
        "Matter.Body.setVelocity(b1, {x:" + n(Number(p.v1)) + ", y:0});",
        "/* 静球(珊瑚红,置于中偏右) */",
        "var b2 = Matter.Bodies.circle(W*0.6, cy, " + n(r2) + ", {friction:0, frictionAir:0, restitution:" + n(Number(p.rest)) + ", render:{fillStyle:'#EE7B70'}});",
        "Matter.Body.setMass(b2, " + n(Number(p.m2)) + ");",
        "Matter.Composite.add(world, b2);",
      ].join('\n')
    },
  },
]

// ============================================================
// 聚合出口(首发 + 批次3c课标扩充;同组名自动并组,弹窗零改动)
// ============================================================

/** 全量模板(15个) */
export const PHYSICS_TEMPLATES: PhysicsTemplate[] = [
  ...PHYSICS_BASE,
  ...PHYSICS_TEMPLATES_EXT,
]

// ============================================================
// 查询辅助(签名范式对齐 mathGraphTemplates)
// ============================================================

/** 按 ID 查模板(未命中返回 undefined,调用方自行兜底) */
export function findPhysicsTemplate(id: string): PhysicsTemplate | undefined {
  return PHYSICS_TEMPLATES.find(t => t.id === id)
}

/** 模板分组列表(保持注册表顺序,供弹窗左栏分组渲染;同组名条目自动归并) */
export function getPhysicsTemplateGroups(): { group: string; items: PhysicsTemplate[] }[] {
  const groups: { group: string; items: PhysicsTemplate[] }[] = []
  for (const t of PHYSICS_TEMPLATES) {
    let g = groups.find(x => x.group === t.group)
    if (!g) { g = { group: t.group, items: [] }; groups.push(g) }
    g.items.push(t)
  }
  return groups
}

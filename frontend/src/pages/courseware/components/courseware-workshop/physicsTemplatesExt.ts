/**
 * physicsTemplatesExt.ts — 物理场景课标扩充模板(批次3c,2026-07-09;9个场景)
 *
 * 背景:批次3首发6个场景覆盖面不足,本批按 K12 物理课标补齐:
 *   🍎 运动与力   +3:摩擦对比赛跑 / 抛射角对比(30·45·60三球齐发) / 平抛vs自由落体
 *   🔗 振动与摆   +1:双摆(混沌入门)
 *   💥 碰撞与动量 +2:牛顿摆 / 二维斜碰台球
 *   ⚡ 能量与热   +3(新组):弹跳与能量损失 / 分子热运动(初中分子动理论) / 多米诺能量传递
 * 首发6 + 扩充9 = 15个场景。
 *
 * 分组约定:沿用主文件已有分组名自动并组;「⚡ 能量与热」为本批新组,排在最后。
 * 代码规范与主文件 physicsTemplates.ts 完全一致(纯构造/W-H相对定位/单引号),
 * 新增模板规范见主文件文件头。
 *
 * 本批特有技巧备忘:
 *   - 同点齐发的对比类场景(抛射角对比/平抛vs自由落体/双摆双臂)给运动体设
 *     collisionFilter:{group:-1}——同负组刚体互不碰撞但仍与地面墙体碰撞,
 *     避免对比球起步时互相弹开破坏对比;
 *   - Matter 碰撞参数合成规则:friction 取两体较小值(故地面设1,让物体自带μ生效)、
 *     restitution 取两体较大值(故地面设0,让球自带恢复系数生效);
 *   - 分子热运动用 Math.random 初始化位置与方向——重置即重新随机,天然模拟
 *     "无规则运动"的教学语义,不算违反纯构造(不依赖上次运行状态)。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/physicsTemplatesExt.ts
 */
import type { PhysicsTemplate } from './physicsUtils'

// ============================================================
// 共享辅助(与主文件同名同实现;不跨文件导出以避免循环依赖)
// ============================================================

/** 数值插值:限制到3位小数并去长尾零,防浮点长尾污染生成代码 */
function n(v: number): string {
  return parseFloat(v.toFixed(3)).toString()
}

/** 静态体(地面/墙/斜面)统一填色 */
const STATIC_FILL = "render:{fillStyle:'#94A3B8'}"

// ============================================================
// 扩充模板注册表(9个)
// ============================================================

export const PHYSICS_TEMPLATES_EXT: PhysicsTemplate[] = [

  // ---------- 🍎 运动与力(扩充) ----------
  {
    id: 'phys-frictionrace', group: '🍎 运动与力', name: '摩擦对比赛跑', emoji: '🏁',
    desc: '上下两条赛道,同速出发的滑块因摩擦系数不同,滑行距离不同',
    params: [
      { key: 'v0', label: '出发速度', type: 'number', min: 4, max: 14, step: 1, defaultValue: 9 },
      { key: 'mu1', label: '上道摩擦系数', type: 'number', min: 0, max: 1, step: 0.05, defaultValue: 0.1, hint: '摩擦越小滑得越远;设0体验理想光滑面' },
      { key: 'mu2', label: '下道摩擦系数', type: 'number', min: 0, max: 1, step: 0.05, defaultValue: 0.4 },
    ],
    buildSetup: (p) => [
      "engine.gravity.y = 1;",
      "/* 上下两条静态赛道(地面 friction 设1,合成取较小值→滑块自带μ生效) */",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, H*0.45, W*0.92, 16, {isStatic:true, friction:1, " + STATIC_FILL + "}));",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, H-12, W, 24, {isStatic:true, friction:1, " + STATIC_FILL + "}));",
      "/* 上道滑块(蓝,低摩擦)与下道滑块(珊瑚红,高摩擦),同 x 同速出发 */",
      "var s1 = Matter.Bodies.rectangle(64, H*0.45-25, 34, 34, {friction:" + n(Number(p.mu1)) + ", frictionStatic:" + n(Number(p.mu1)) + ", restitution:0, render:{fillStyle:'#6C9BF0'}});",
      "var s2 = Matter.Bodies.rectangle(64, H-41, 34, 34, {friction:" + n(Number(p.mu2)) + ", frictionStatic:" + n(Number(p.mu2)) + ", restitution:0, render:{fillStyle:'#EE7B70'}});",
      "Matter.Composite.add(world, [s1, s2]);",
      "Matter.Body.setVelocity(s1, {x:" + n(Number(p.v0)) + ", y:0});",
      "Matter.Body.setVelocity(s2, {x:" + n(Number(p.v0)) + ", y:0});",
    ].join('\n'),
  },
  {
    id: 'phys-anglecompare', group: '🍎 运动与力', name: '抛射角对比', emoji: '🎯',
    desc: '30°/45°/60°三球同速齐发,45°射程最远(互不碰撞,纯轨迹对比)',
    params: [
      { key: 'v', label: '出手速度', type: 'number', min: 8, max: 22, step: 1, defaultValue: 15 },
      { key: 'g', label: '重力强度', type: 'number', min: 0.3, max: 2, step: 0.1, defaultValue: 1 },
    ],
    buildSetup: (p) => {
      const v = Number(p.v)
      const mk = (deg: number, color: string) => {
        const rad = (deg * Math.PI) / 180
        return [
          "var b" + deg + " = Matter.Bodies.circle(56, H-40, 13, {frictionAir:0, restitution:0.3, collisionFilter:{group:-1}, render:{fillStyle:'" + color + "'}});",
          "Matter.Composite.add(world, b" + deg + ");",
          "Matter.Body.setVelocity(b" + deg + ", {x:" + n(v * Math.cos(rad)) + ", y:" + n(-v * Math.sin(rad)) + "});",
        ].join('\n')
      }
      return [
        "engine.gravity.y = " + n(Number(p.g)) + ";",
        "/* 地面 + 右墙 */",
        "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, H-12, W, 24, {isStatic:true, " + STATIC_FILL + "}));",
        "Matter.Composite.add(world, Matter.Bodies.rectangle(W-8, H/2, 16, H, {isStatic:true, " + STATIC_FILL + "}));",
        "/* 三球同点同速齐发,负碰撞组互不碰撞:30°薄荷绿 / 45°珊瑚红 / 60°蓝 */",
        mk(30, '#5BBFA5'),
        mk(45, '#EE7B70'),
        mk(60, '#6C9BF0'),
      ].join('\n')
    },
  },
  {
    id: 'phys-dropvsproj', group: '🍎 运动与力', name: '平抛vs自由落体', emoji: '🪂',
    desc: '同高同时释放:一球自由下落、一球水平抛出,落地时刻相同(运动的独立性)',
    params: [
      { key: 'vx', label: '水平抛出速度', type: 'number', min: 4, max: 16, step: 1, defaultValue: 10, hint: '只改变落点远近,不改变落地时间' },
      { key: 'g', label: '重力强度', type: 'number', min: 0.3, max: 2, step: 0.1, defaultValue: 1 },
    ],
    buildSetup: (p) => [
      "engine.gravity.y = " + n(Number(p.g)) + ";",
      "/* 地面 + 右墙 */",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, H-12, W, 24, {isStatic:true, " + STATIC_FILL + "}));",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W-8, H/2, 16, H, {isStatic:true, " + STATIC_FILL + "}));",
      "/* 出发高台标记 */",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W*0.2, 92, 110, 12, {isStatic:true, " + STATIC_FILL + "}));",
      "/* 自由落体球(珊瑚红)与平抛球(蓝):同高释放,负碰撞组互不干扰 */",
      "var bd = Matter.Bodies.circle(W*0.17, 62, 14, {frictionAir:0, restitution:0.3, collisionFilter:{group:-1}, render:{fillStyle:'#EE7B70'}});",
      "var bp = Matter.Bodies.circle(W*0.23, 62, 14, {frictionAir:0, restitution:0.3, collisionFilter:{group:-1}, render:{fillStyle:'#6C9BF0'}});",
      "Matter.Composite.add(world, [bd, bp]);",
      "Matter.Body.setVelocity(bp, {x:" + n(Number(p.vx)) + ", y:0});",
    ].join('\n'),
  },

  // ---------- 🔗 振动与摆(扩充) ----------
  {
    id: 'phys-doublependulum', group: '🔗 振动与摆', name: '双摆', emoji: '🌀',
    desc: '两节摆臂串联,大摆角下呈现混沌运动——初始条件的微小差异导致轨迹迥异',
    params: [
      { key: 'l1', label: '上臂长', type: 'number', min: 80, max: 160, step: 10, defaultValue: 120 },
      { key: 'l2', label: '下臂长', type: 'number', min: 80, max: 160, step: 10, defaultValue: 120 },
      { key: 'angle0', label: '初始摆角(°)', type: 'number', min: 30, max: 170, step: 5, defaultValue: 120, hint: '大角度(>90°)才会进入明显的混沌状态' },
    ],
    buildSetup: (p) => {
      const a0 = (Number(p.angle0) * Math.PI) / 180
      return [
        "engine.gravity.y = 1;",
        "var L1 = " + n(Number(p.l1)) + ", L2 = " + n(Number(p.l2)) + ", a0 = " + n(a0) + ";",
        "var px = W/2, py = 64;",
        "/* 悬点标记 */",
        "Matter.Composite.add(world, Matter.Bodies.rectangle(px, py, 14, 14, {isStatic:true, " + STATIC_FILL + "}));",
        "/* 两节摆锤沿初始摆角方向排成直线释放(负碰撞组防两锤互碰) */",
        "var x1 = px + L1*Math.sin(a0), y1 = py + L1*Math.cos(a0);",
        "var bob1 = Matter.Bodies.circle(x1, y1, 16, {frictionAir:0, collisionFilter:{group:-1}, render:{fillStyle:'#9B8AE6'}});",
        "var bob2 = Matter.Bodies.circle(x1 + L2*Math.sin(a0), y1 + L2*Math.cos(a0), 16, {frictionAir:0, collisionFilter:{group:-1}, render:{fillStyle:'#EE7B70'}});",
        "Matter.Composite.add(world, [bob1, bob2]);",
        "Matter.Composite.add(world, Matter.Constraint.create({pointA:{x:px,y:py}, bodyB:bob1, length:L1, stiffness:1, render:{strokeStyle:'#94A3B8', lineWidth:2}}));",
        "Matter.Composite.add(world, Matter.Constraint.create({bodyA:bob1, bodyB:bob2, length:L2, stiffness:1, render:{strokeStyle:'#94A3B8', lineWidth:2}}));",
      ].join('\n')
    },
  },

  // ---------- 💥 碰撞与动量(扩充) ----------
  {
    id: 'phys-newtoncradle', group: '💥 碰撞与动量', name: '牛顿摆', emoji: '🔮',
    desc: '拉起一端小球释放,动量与动能依次传递,另一端弹出(经典桌面玩具)',
    params: [
      { key: 'count', label: '小球数量', type: 'number', min: 3, max: 7, step: 1, defaultValue: 5 },
      { key: 'angle', label: '拉起角度(°)', type: 'number', min: 20, max: 70, step: 5, defaultValue: 50 },
    ],
    buildSetup: (p) => {
      const a = (Number(p.angle) * Math.PI) / 180
      return [
        "engine.gravity.y = 1;",
        "var nb = " + n(Number(p.count)) + ", r = 17, py = 70, A = " + n(a) + ";",
        "var L = H*0.5;",
        "var start = W/2 - (nb-1)*r;",
        "/* 横梁 */",
        "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, py, (nb+1)*2*r, 10, {isStatic:true, " + STATIC_FILL + "}));",
        "for (var i = 0; i < nb; i++) {",
        "  var px = start + i*2*r;",
        "  var bx = px, by = py + L;",
        "  if (i === 0) { bx = px - L*Math.sin(A); by = py + L*Math.cos(A); } /* 首球拉起 */",
        "  var ball = Matter.Bodies.circle(bx, by, r, {restitution:1, friction:0, frictionAir:0, slop:0.01, render:{fillStyle: i===0 ? '#EE7B70' : '#6C9BF0'}});",
        "  Matter.Composite.add(world, ball);",
        "  Matter.Composite.add(world, Matter.Constraint.create({pointA:{x:px,y:py}, bodyB:ball, length:L, stiffness:1, render:{strokeStyle:'#94A3B8', lineWidth:2}}));",
        "}",
      ].join('\n')
    },
  },
  {
    id: 'phys-billiard', group: '💥 碰撞与动量', name: '二维斜碰台球', emoji: '🎱',
    desc: '母球偏心撞击静止目标球,等质量弹性斜碰后两球运动方向近似垂直',
    params: [
      { key: 'offset', label: '瞄准偏移量', type: 'number', min: 0, max: 30, step: 2, defaultValue: 15, hint: '0=对心正碰;偏移越大散射角差异越明显' },
      { key: 'v', label: '母球速度', type: 'number', min: 4, max: 12, step: 1, defaultValue: 8 },
      { key: 'rest', label: '恢复系数', type: 'number', min: 0.5, max: 1, step: 0.05, defaultValue: 1, hint: '1 时碰后两球方向垂直(等质量弹性斜碰结论)' },
    ],
    buildSetup: (p) => [
      "engine.gravity.y = 0; /* 台球桌俯视视角,无重力 */",
      "/* 四周桌边(restitution 合成取较大值→球自带值生效,桌边设0) */",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, 8, W, 16, {isStatic:true, restitution:0, " + STATIC_FILL + "}));",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, H-8, W, 16, {isStatic:true, restitution:0, " + STATIC_FILL + "}));",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(8, H/2, 16, H, {isStatic:true, restitution:0, " + STATIC_FILL + "}));",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W-8, H/2, 16, H, {isStatic:true, restitution:0, " + STATIC_FILL + "}));",
      "/* 母球(蓝,带瞄准偏移)与目标球(珊瑚红,静止) */",
      "var cue = Matter.Bodies.circle(W*0.2, H/2 - " + n(Number(p.offset)) + ", 16, {friction:0, frictionAir:0, restitution:" + n(Number(p.rest)) + ", render:{fillStyle:'#6C9BF0'}});",
      "var tgt = Matter.Bodies.circle(W*0.55, H/2, 16, {friction:0, frictionAir:0, restitution:" + n(Number(p.rest)) + ", render:{fillStyle:'#EE7B70'}});",
      "Matter.Composite.add(world, [cue, tgt]);",
      "Matter.Body.setVelocity(cue, {x:" + n(Number(p.v)) + ", y:0});",
    ].join('\n'),
  },

  // ---------- ⚡ 能量与热(新组) ----------
  {
    id: 'phys-bounce', group: '⚡ 能量与热', name: '弹跳与能量损失', emoji: '🏀',
    desc: '小球落地反弹,每次反弹高度按恢复系数衰减——机械能逐次损失',
    params: [
      { key: 'rest', label: '恢复系数', type: 'number', min: 0.3, max: 0.95, step: 0.05, defaultValue: 0.7, hint: '反弹高度≈原高度×恢复系数²' },
      { key: 'g', label: '重力强度', type: 'number', min: 0.3, max: 2, step: 0.1, defaultValue: 1 },
    ],
    buildSetup: (p) => [
      "engine.gravity.y = " + n(Number(p.g)) + ";",
      "/* 地面 restitution 设0(合成取较大值→球自带恢复系数生效) */",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, H-12, W, 24, {isStatic:true, restitution:0, friction:0, " + STATIC_FILL + "}));",
      "/* 从顶部释放的球 */",
      "Matter.Composite.add(world, Matter.Bodies.circle(W/2, 56, 18, {restitution:" + n(Number(p.rest)) + ", friction:0, frictionAir:0, render:{fillStyle:'#EE7B70'}}));",
    ].join('\n'),
  },
  {
    id: 'phys-gasmotion', group: '⚡ 能量与热', name: '分子热运动', emoji: '💨',
    desc: '封闭容器内粒子无规则运动、频繁碰撞(分子动理论气体模型)',
    params: [
      { key: 'count', label: '粒子数量', type: 'number', min: 10, max: 40, step: 2, defaultValue: 20 },
      { key: 'speed', label: '粒子速率', type: 'number', min: 2, max: 10, step: 1, defaultValue: 5, hint: '速率越大对应温度越高,碰撞越剧烈' },
    ],
    buildSetup: (p) => [
      "engine.gravity.y = 0; /* 理想气体模型忽略重力 */",
      "/* 封闭容器四壁(弹性碰撞,能量不损失) */",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, 8, W, 16, {isStatic:true, restitution:1, friction:0, " + STATIC_FILL + "}));",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, H-8, W, 16, {isStatic:true, restitution:1, friction:0, " + STATIC_FILL + "}));",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(8, H/2, 16, H, {isStatic:true, restitution:1, friction:0, " + STATIC_FILL + "}));",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W-8, H/2, 16, H, {isStatic:true, restitution:1, friction:0, " + STATIC_FILL + "}));",
      "/* 随机位置随机方向的粒子群(每次重置重新随机,呼应'无规则'教学语义) */",
      "var colors = ['#6C9BF0', '#EE7B70', '#5BBFA5', '#9B8AE6'];",
      "var np = " + n(Number(p.count)) + ", sp = " + n(Number(p.speed)) + ";",
      "for (var i = 0; i < np; i++) {",
      "  var x = 44 + Math.random()*(W-88), y = 44 + Math.random()*(H-88);",
      "  var pt = Matter.Bodies.circle(x, y, 7, {restitution:1, friction:0, frictionAir:0, slop:0.01, render:{fillStyle:colors[i%4]}});",
      "  Matter.Composite.add(world, pt);",
      "  var th = Math.random()*Math.PI*2;",
      "  Matter.Body.setVelocity(pt, {x:sp*Math.cos(th), y:sp*Math.sin(th)});",
      "}",
    ].join('\n'),
  },
  {
    id: 'phys-domino', group: '⚡ 能量与热', name: '多米诺能量传递', emoji: '🁢',
    desc: '小球撞倒首张骨牌,能量沿队列传递;间距过大则传递中断',
    params: [
      { key: 'count', label: '骨牌数量', type: 'number', min: 6, max: 14, step: 1, defaultValue: 10 },
      { key: 'gap', label: '骨牌间距', type: 'number', min: 26, max: 58, step: 2, defaultValue: 34, hint: '间距接近骨牌高度(64)时会传不下去——能量传递的条件' },
    ],
    buildSetup: (p) => [
      "engine.gravity.y = 1;",
      "/* 地面(高摩擦防骨牌滑移) */",
      "Matter.Composite.add(world, Matter.Bodies.rectangle(W/2, H-12, W, 24, {isStatic:true, friction:1, " + STATIC_FILL + "}));",
      "/* 骨牌队列:10×64 竖立薄板 */",
      "var nd = " + n(Number(p.count)) + ", gap = " + n(Number(p.gap)) + ";",
      "var startX = W*0.22;",
      "for (var i = 0; i < nd; i++) {",
      "  Matter.Composite.add(world, Matter.Bodies.rectangle(startX + i*gap, H-56, 10, 64, {friction:0.4, frictionStatic:0.6, restitution:0, render:{fillStyle: i%2===0 ? '#6C9BF0' : '#9B8AE6'}}));",
      "}",
      "/* 触发球(珊瑚红):自左侧滚入撞倒首张骨牌 */",
      "var trigger = Matter.Bodies.circle(startX - 76, H-38, 14, {friction:0.05, frictionAir:0, restitution:0.1, render:{fillStyle:'#EE7B70'}});",
      "Matter.Composite.add(world, trigger);",
      "Matter.Body.setVelocity(trigger, {x:6, y:0});",
    ].join('\n'),
  },
]

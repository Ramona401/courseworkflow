/**
 * mathGraphTemplates.ts — 数学动态图形模板聚合出口(批次1c拆分;批次1d并入对称与全等专题,合计48个)
 *
 * 原单体注册表按学段拆为多文件,本文件仅做聚合与查询,对 MathGraphModal /
 * mathGraphUtils 保持 import 路径与导出签名完全不变(零感知重构)。
 *
 * 学段文件(弹窗左栏按此顺序分组展示;Ext 文件分组名与主文件一致,自动并入同组):
 *   🧒 小学  → mathGraphTemplatesPrimary.ts   (6个:分数/角/平行四边形面积/圆周长/数轴/条形图)
 *   📗 初中  → mathGraphTemplatesJuniorFunc.ts(函数4个)+ mathGraphTemplatesJuniorGeo.ts(几何6个)
 *              + mathGraphTemplatesJuniorSymmetry.ts(批次1d新专题组「对称与全等」6个:
 *                轴对称全等验证/SSS三边定形/将军饮马/垂直平分性质/旋转全等/矩形折叠)
 *              + mathGraphTemplatesJuniorExt.ts(扩充7个:平移旋转/内角和/多边形内角和/
 *                位似/垂径定理→几何组;动点问题/二次函数与方程→函数组)
 *              + mathGraphTemplatesJuniorExt2.ts(查缺6个:数轴相反数绝对值/不等式解集/
 *                坐标系象限→数与代数新组;中心对称/锐角三角函数→几何组;
 *                一次函数与方程组→函数组)
 *   📘 高中  → mathGraphTemplatesSenior.ts    (函数4个 + 三角/圆锥曲线2个)
 *              + mathGraphTemplatesSeniorExt.ts(扩充7个:双曲线/抛物线/正弦定理→圆锥曲线组;
 *                向量加法/线性规划→函数组;导数切线/黎曼和→微积分初步新组)
 *   共享辅助 → mathGraphTemplateShared.ts     (n() 数值插值 / SLIDER_ATTRS 滑杆外观)
 *   合计48个模板,覆盖小初高主干考点。
 *
 * 新增模板规范(各学段文件通用,新增必读):
 *   1. 构造代码内字符串一律用单引号;禁止出现 "</script" 字样;
 *   2. 教学参数做成 JSXGraph 滑杆,老师设的参数值作为滑杆初值——融入课件后
 *      课堂上仍可现场拖动,这是本工具的核心教学价值;
 *   3. 几何类模板 keepAspectRatio 必须 true(否则圆被拉成椭圆、角弧变形);
 *   4. 滑杆/文字的摆放坐标要落在 boundingBox 内的空白区(通常顶部);
 *   5. 数值插值一律经共享 n() 辅助(限制小数位数,防浮点长尾污染生成代码);
 *   6. buildConstruction 产出的代码是"弹窗预览"与"课件融入"的单一真相源
 *      (详见 mathGraphUtils.ts 文件头架构约定);
 *   7. 元素名带撇号时用撇号字符′(如 A′)而非英文单引号,规避引号转义冲突。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/mathGraphTemplates.ts
 */
import type { MathGraphTemplate } from './mathGraphUtils'
import { MATH_GRAPH_TEMPLATES_PRIMARY } from './mathGraphTemplatesPrimary'
import { MATH_GRAPH_TEMPLATES_JUNIOR_FUNC } from './mathGraphTemplatesJuniorFunc'
import { MATH_GRAPH_TEMPLATES_JUNIOR_GEO } from './mathGraphTemplatesJuniorGeo'
import { MATH_GRAPH_TEMPLATES_JUNIOR_SYMMETRY } from './mathGraphTemplatesJuniorSymmetry'
import { MATH_GRAPH_TEMPLATES_JUNIOR_EXT } from './mathGraphTemplatesJuniorExt'
import { MATH_GRAPH_TEMPLATES_JUNIOR_EXT2 } from './mathGraphTemplatesJuniorExt2'
import { MATH_GRAPH_TEMPLATES_SENIOR } from './mathGraphTemplatesSenior'
import { MATH_GRAPH_TEMPLATES_SENIOR_EXT } from './mathGraphTemplatesSeniorExt'

/** 全量模板(学段顺序:小学 → 初中 → 高中;对称与全等专题紧随几何组;Ext 按分组名自动并入) */
export const MATH_GRAPH_TEMPLATES: MathGraphTemplate[] = [
  ...MATH_GRAPH_TEMPLATES_PRIMARY,
  ...MATH_GRAPH_TEMPLATES_JUNIOR_FUNC,
  ...MATH_GRAPH_TEMPLATES_JUNIOR_GEO,
  ...MATH_GRAPH_TEMPLATES_JUNIOR_SYMMETRY,
  ...MATH_GRAPH_TEMPLATES_JUNIOR_EXT,
  ...MATH_GRAPH_TEMPLATES_JUNIOR_EXT2,
  ...MATH_GRAPH_TEMPLATES_SENIOR,
  ...MATH_GRAPH_TEMPLATES_SENIOR_EXT,
]

/** 按 ID 查模板(未命中返回 undefined,调用方自行兜底) */
export function findMathTemplate(id: string): MathGraphTemplate | undefined {
  return MATH_GRAPH_TEMPLATES.find(t => t.id === id)
}

/** 模板分组列表(保持注册表顺序,供弹窗左栏分组渲染) */
export function getMathTemplateGroups(): { group: string; items: MathGraphTemplate[] }[] {
  const groups: { group: string; items: MathGraphTemplate[] }[] = []
  for (const t of MATH_GRAPH_TEMPLATES) {
    let g = groups.find(x => x.group === t.group)
    if (!g) { g = { group: t.group, items: [] }; groups.push(g) }
    g.items.push(t)
  }
  return groups
}

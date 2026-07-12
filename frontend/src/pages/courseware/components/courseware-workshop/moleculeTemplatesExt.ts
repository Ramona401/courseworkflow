/**
 * moleculeTemplatesExt.ts — 分子实验室课标扩充模板(批次2c,2026-07-09)
 *
 * 背景:批次2首发的 14+16 个模板覆盖面不足,本批按 K12 化学课标补齐:
 *   3D +14:CO/Cl₂/SO₂/H₂S/O₃/H₂O₂/CCl₄(初中常见气体+选择性必修2分子构型:
 *          V形/二面角/正四面体)、甲醇/乙烷/甲醛/乙酸(有机构型)、
 *          石墨/C60富勒烯/二氧化硅(配合主文件的金刚石,凑齐人教九上
 *          "碳单质三兄弟"+必修二共价晶体,全部程序生成点阵);
 *   2D +14:乙醛/丙酮/乙酸乙酯/苯酚/甲苯/苯乙烯/氯乙烯/溴乙烷/乙二醇/
 *          丙三醇/甘氨酸/尿素(高中有机主干)+果糖/维生素C。
 *
 * 分组约定:条目沿用主文件已有分组名(常见小分子/有机入门/晶体结构/生活中的分子),
 * getMoleculeXXGroups 按组名自动并组,弹窗左栏零改动;
 * 新增「🎓 高中有机」组承接 2D 有机主干扩充。
 *
 * 数据规范与主文件 moleculeTemplates.ts 完全一致(数据自包含/教科书键长/
 * 晶体程序生成并注明晶格常数),新增模板规范见主文件文件头。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/moleculeTemplatesExt.ts
 */
import type { Molecule3DTemplate, Molecule2DTemplate } from './moleculeUtils'

// ============================================================
// xyz 构造辅助(与主文件同名同实现;不从主文件导出以避免循环依赖)
// ============================================================

/** 把坐标行数组包装成标准 xyz 格式(首行原子数 + 次行注释) */
function xyz(comment: string, lines: string[]): string {
  return lines.length + '\n' + comment + '\n' + lines.join('\n')
}

/**
 * 石墨双层点阵程序生成:层内蜂窝格 C-C 1.42 Å,层间距 3.35 Å(AA堆叠简化,
 * 教学重点是"层内共价强键+层间弱作用力",堆叠方式不影响该结论)。
 * 3Dmol 按距离推断成键:层内 1.42 成键、层间 3.35 不成键——恰好呈现教学要点。
 */
function buildGraphiteXYZ(): string {
  const d = 1.42
  const a0 = d * Math.sqrt(3)            // 晶格常数 2.46 Å
  const half = a0 / 2
  const basisY = a0 / (2 * Math.sqrt(3)) // 第二基原子 y 偏移
  const lines: string[] = []
  for (const z of [0, 3.35]) {
    for (let i = -2; i <= 2; i++) {
      for (let j = -2; j <= 2; j++) {
        const bx = i * a0 + j * half
        const by = j * a0 * Math.sqrt(3) / 2
        for (const [ox, oy] of [[0, 0], [half, basisY]] as [number, number][]) {
          const x = bx + ox, y = by + oy
          if (x * x + y * y <= 30.25) {  // 半径 5.5 Å 圆形裁剪,层片边缘整齐
            lines.push('C ' + x.toFixed(3) + ' ' + y.toFixed(3) + ' ' + z.toFixed(3))
          }
        }
      }
    }
  }
  return xyz('石墨双层(层内C-C 1.42A, 层间3.35A, AA堆叠简化)', lines)
}

/**
 * C60 富勒烯程序生成:截角二十面体的 60 个顶点——三组基坐标
 * (0,±1,±3φ)/(±1,±(2+φ),±2φ)/(±2,±(1+2φ),±φ) 的循环置换,φ 为黄金比。
 * 该坐标系棱长为 2,缩放 0.7075 使 C-C≈1.415 Å(真实 C60 两种键长的均值)。
 */
function buildC60XYZ(): string {
  const phi = (1 + Math.sqrt(5)) / 2
  const s = 0.7075
  const base: [number, number, number][] = []
  for (const sy of [1, -1]) for (const sz of [1, -1]) {
    base.push([0, sy * 1, sz * 3 * phi])
  }
  for (const sx of [1, -1]) for (const sy of [1, -1]) for (const sz of [1, -1]) {
    base.push([sx * 1, sy * (2 + phi), sz * 2 * phi])
    base.push([sx * 2, sy * (1 + 2 * phi), sz * phi])
  }
  const lines: string[] = []
  for (const [x, y, z] of base) {
    for (const p of [[x, y, z], [y, z, x], [z, x, y]]) {
      lines.push('C ' + (p[0] * s).toFixed(3) + ' ' + (p[1] * s).toFixed(3) + ' ' + (p[2] * s).toFixed(3))
    }
  }
  return xyz('C60富勒烯(截角二十面体, 60个碳, C-C约1.42A)', lines)
}

/**
 * 二氧化硅(β-方石英型)程序生成:Si 取金刚石点阵(a=7.16 Å),
 * O 置于每对近邻 Si 连线中点(近邻距 a√3/4≈3.10,故 Si-O≈1.55 Å)。
 * 单晶胞局部(8 Si + 若干桥氧),足以呈现"每个Si连4个O、每个O桥接2个Si"的网络特征。
 */
function buildSiO2XYZ(): string {
  const a = 7.16
  const fcc: [number, number, number][] = [[0, 0, 0], [0.5, 0.5, 0], [0.5, 0, 0.5], [0, 0.5, 0.5]]
  const si: [number, number, number][] = []
  for (const p of fcc) {
    si.push([p[0] * a, p[1] * a, p[2] * a])
    si.push([(p[0] + 0.25) * a, (p[1] + 0.25) * a, (p[2] + 0.25) * a])
  }
  const lines: string[] = si.map(p => 'Si ' + p[0].toFixed(3) + ' ' + p[1].toFixed(3) + ' ' + p[2].toFixed(3))
  /* 近邻 Si 对(距离≈3.10)中点放桥氧,O(n²) 遍历去重 */
  for (let i = 0; i < si.length; i++) {
    for (let j = i + 1; j < si.length; j++) {
      const dx = si[i][0] - si[j][0], dy = si[i][1] - si[j][1], dz = si[i][2] - si[j][2]
      const dist = Math.sqrt(dx * dx + dy * dy + dz * dz)
      if (Math.abs(dist - a * Math.sqrt(3) / 4) < 0.15) {
        lines.push('O ' + ((si[i][0] + si[j][0]) / 2).toFixed(3) + ' ' + ((si[i][1] + si[j][1]) / 2).toFixed(3) + ' ' + ((si[i][2] + si[j][2]) / 2).toFixed(3))
      }
    }
  }
  return xyz('二氧化硅(方石英型局部, a=7.16A, Si-O 1.55A)', lines)
}

// ============================================================
// 3D 扩充模板(14个;分组名沿用主文件,自动并组)
// ============================================================

export const MOLECULE_3D_TEMPLATES_EXT: Molecule3DTemplate[] = [
  // ---------- 🌫️ 常见小分子(扩充:初中气体 + 选必2分子构型) ----------
  {
    id: 'mol3d-co', group: '🌫️ 常见小分子', name: '一氧化碳', formula: 'CO', emoji: '⚠️',
    desc: '有毒气体,具还原性,与CO₂仅差一个氧原子',
    buildXYZ: () => xyz('一氧化碳(C-O 1.13A)', [
      'C -0.564 0.000 0.000',
      'O 0.564 0.000 0.000',
    ]),
  },
  {
    id: 'mol3d-cl2', group: '🌫️ 常见小分子', name: '氯气', formula: 'Cl₂', emoji: '🟡',
    desc: '黄绿色有毒气体,Cl-Cl 单键',
    buildXYZ: () => xyz('氯气(Cl-Cl 1.99A)', [
      'Cl 0.994 0.000 0.000',
      'Cl -0.994 0.000 0.000',
    ]),
  },
  {
    id: 'mol3d-so2', group: '🌫️ 常见小分子', name: '二氧化硫', formula: 'SO₂', emoji: '🌋',
    desc: 'V形分子(键角约119°),酸雨成因之一',
    buildXYZ: () => xyz('二氧化硫(S-O 1.43A, 键角119.5°)', [
      'S 0.000 0.000 0.000',
      'O 1.237 0.721 0.000',
      'O -1.237 0.721 0.000',
    ]),
  },
  {
    id: 'mol3d-h2s', group: '🌫️ 常见小分子', name: '硫化氢', formula: 'H₂S', emoji: '🥚',
    desc: 'V形分子,键角约92°(比水小,可与H₂O对比)',
    buildXYZ: () => xyz('硫化氢(S-H 1.34A, 键角92°)', [
      'S 0.000 0.000 0.000',
      'H 0.964 0.931 0.000',
      'H -0.964 0.931 0.000',
    ]),
  },
  {
    id: 'mol3d-o3', group: '🌫️ 常见小分子', name: '臭氧', formula: 'O₃', emoji: '🌐',
    desc: 'V形分子,O₂的同素异形体,臭氧层主角',
    buildXYZ: () => xyz('臭氧(O-O 1.28A, 键角117°)', [
      'O 0.000 0.000 0.000',
      'O 1.088 0.670 0.000',
      'O -1.088 0.670 0.000',
    ]),
  },
  {
    id: 'mol3d-h2o2', group: '🌫️ 常见小分子', name: '过氧化氢', formula: 'H₂O₂', emoji: '🧴',
    desc: '非平面结构,两个O-H分居二面角两侧(选必2经典构型)',
    buildXYZ: () => xyz('过氧化氢(O-O 1.47A, 二面角约115°)', [
      'O 0.000 0.735 0.000',
      'O 0.000 -0.735 0.000',
      'H 0.936 0.900 0.000',
      'H -0.396 -0.900 0.848',
    ]),
  },
  {
    id: 'mol3d-ccl4', group: '🌫️ 常见小分子', name: '四氯化碳', formula: 'CCl₄', emoji: '🧯',
    desc: '正四面体,含极性键的非极性分子(与CH₄对照)',
    buildXYZ: () => xyz('四氯化碳(C-Cl 1.77A, 正四面体)', [
      'C 0.000 0.000 0.000',
      'Cl 1.022 1.022 1.022',
      'Cl -1.022 -1.022 1.022',
      'Cl -1.022 1.022 -1.022',
      'Cl 1.022 -1.022 -1.022',
    ]),
  },
  // ---------- 🔥 有机入门(扩充:醇/烷/醛/羧酸构型) ----------
  {
    id: 'mol3d-ch3oh', group: '🔥 有机入门', name: '甲醇', formula: 'CH₃OH', emoji: '🧪',
    desc: '最简单的醇,甲基与羟基(与乙醇对比同系物)',
    buildXYZ: () => xyz('甲醇(C-O 1.43A, O-H 0.96A)', [
      'C 0.000 0.000 0.000',
      'O 1.430 0.000 0.000',
      'H 1.727 0.913 0.000',
      'H -0.363 1.028 0.000',
      'H -0.363 -0.514 0.890',
      'H -0.363 -0.514 -0.890',
    ]),
  },
  {
    id: 'mol3d-c2h6', group: '🔥 有机入门', name: '乙烷', formula: 'C₂H₆', emoji: '⛽',
    desc: '碳碳单键可旋转,交叉式构象(与乙烯乙炔对比键型)',
    buildXYZ: () => xyz('乙烷(C-C 1.53A, 交叉式构象)', [
      'C -0.765 0.000 0.000',
      'C 0.765 0.000 0.000',
      'H -1.128 1.028 0.000',
      'H -1.128 -0.514 0.890',
      'H -1.128 -0.514 -0.890',
      'H 1.128 -1.028 0.000',
      'H 1.128 0.514 0.890',
      'H 1.128 0.514 -0.890',
    ]),
  },
  {
    id: 'mol3d-hcho', group: '🔥 有机入门', name: '甲醛', formula: 'HCHO', emoji: '🪑',
    desc: '平面形分子,最简单的醛,含碳氧双键',
    buildXYZ: () => xyz('甲醛(C=O 1.21A, 平面形)', [
      'C 0.000 0.000 0.000',
      'O 0.000 1.210 0.000',
      'H 0.924 -0.578 0.000',
      'H -0.924 -0.578 0.000',
    ]),
  },
  {
    id: 'mol3d-ch3cooh', group: '🔥 有机入门', name: '乙酸', formula: 'CH₃COOH', emoji: '🍾',
    desc: '羧基的立体结构:C=O 与 C-O-H 共平面',
    buildXYZ: () => xyz('乙酸(C-C 1.50A, C=O 1.21A, C-O 1.36A)', [
      'C 0.000 0.000 0.000',
      'C -1.500 0.000 0.000',
      'O 0.641 1.026 0.000',
      'O 0.531 -1.252 0.000',
      'H 1.500 -1.300 0.000',
      'H -1.863 1.028 0.000',
      'H -1.863 -0.514 0.890',
      'H -1.863 -0.514 -0.890',
    ]),
  },
  // ---------- 🧂 晶体结构(扩充:碳单质三兄弟凑齐 + 共价晶体SiO₂) ----------
  {
    id: 'mol3d-graphite', group: '🧂 晶体结构', name: '石墨', formula: 'C(石墨)', emoji: '✏️',
    desc: '层状结构:层内共价强键、层间弱作用力,故软而滑(与金刚石对照)',
    buildXYZ: buildGraphiteXYZ,
  },
  {
    id: 'mol3d-c60', group: '🧂 晶体结构', name: 'C60富勒烯', formula: 'C₆₀', emoji: '⚽',
    desc: '足球状分子:12个五边形+20个六边形,碳的第三种单质形态',
    buildXYZ: buildC60XYZ,
  },
  {
    id: 'mol3d-sio2', group: '🧂 晶体结构', name: '二氧化硅', formula: 'SiO₂', emoji: '🏜️',
    desc: '共价晶体:每个Si连4个O、每个O桥接2个Si的空间网络',
    buildXYZ: buildSiO2XYZ,
  },
]

// ============================================================
// 2D 扩充模板(14个)
// ============================================================

export const MOLECULE_2D_TEMPLATES_EXT: Molecule2DTemplate[] = [
  // ---------- 🎓 高中有机(新组:醛酮酯/芳香族/卤代烃/多元醇/含氮) ----------
  { id: 'mol2d-ch3cho', group: '🎓 高中有机', name: '乙醛', formula: 'CH₃CHO', smiles: 'CC=O', desc: '醛基,银镜反应主角' },
  { id: 'mol2d-acetone', group: '🎓 高中有机', name: '丙酮', formula: 'C₃H₆O', smiles: 'CC(C)=O', desc: '最简单的酮' },
  { id: 'mol2d-ethylacetate', group: '🎓 高中有机', name: '乙酸乙酯', formula: 'C₄H₈O₂', smiles: 'CCOC(C)=O', desc: '酯化反应产物,果香' },
  { id: 'mol2d-phenol', group: '🎓 高中有机', name: '苯酚', formula: 'C₆H₅OH', smiles: 'Oc1ccccc1', desc: '羟基直连苯环,弱酸性' },
  { id: 'mol2d-toluene', group: '🎓 高中有机', name: '甲苯', formula: 'C₇H₈', smiles: 'Cc1ccccc1', desc: '苯的同系物' },
  { id: 'mol2d-styrene', group: '🎓 高中有机', name: '苯乙烯', formula: 'C₈H₈', smiles: 'C=Cc1ccccc1', desc: '聚苯乙烯单体' },
  { id: 'mol2d-vinylchloride', group: '🎓 高中有机', name: '氯乙烯', formula: 'C₂H₃Cl', smiles: 'C=CCl', desc: 'PVC塑料单体' },
  { id: 'mol2d-bromoethane', group: '🎓 高中有机', name: '溴乙烷', formula: 'C₂H₅Br', smiles: 'CCBr', desc: '卤代烃代表,消去/取代反应' },
  { id: 'mol2d-glycol', group: '🎓 高中有机', name: '乙二醇', formula: 'C₂H₆O₂', smiles: 'OCCO', desc: '二元醇,防冻液成分' },
  { id: 'mol2d-glycerol', group: '🎓 高中有机', name: '丙三醇', formula: 'C₃H₈O₃', smiles: 'OCC(O)CO', desc: '甘油,三元醇' },
  { id: 'mol2d-glycine', group: '🎓 高中有机', name: '甘氨酸', formula: 'C₂H₅NO₂', smiles: 'NCC(=O)O', desc: '最简单的氨基酸(氨基+羧基)' },
  { id: 'mol2d-urea', group: '🎓 高中有机', name: '尿素', formula: 'CO(NH₂)₂', smiles: 'NC(N)=O', desc: '首个人工合成的有机物' },
  // ---------- 🍬 生活中的分子(扩充) ----------
  { id: 'mol2d-fructose', group: '🍬 生活中的分子', name: '果糖', formula: 'C₆H₁₂O₆', smiles: 'OCC1OC(O)(CO)C(O)C1O', desc: '葡萄糖的同分异构体' },
  { id: 'mol2d-vitaminc', group: '🍬 生活中的分子', name: '维生素C', formula: 'C₆H₈O₆', smiles: 'OC1=C(O)C(=O)OC1C(O)CO', desc: '抗坏血酸,水果蔬菜中的营养素' },
]

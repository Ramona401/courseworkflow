/**
 * moleculeTemplates.ts — 分子实验室模板注册表(批次2首发;批次2c课标扩充聚合,2026-07-09)
 *
 * 双注册表(批次2c起本文件同时是聚合出口,扩充模板在 moleculeTemplatesExt.ts):
 *   MOLECULE_3D_TEMPLATES — 3D 立体分子(首发14 + 扩充14 = 28个):结构数据以标准
 *     xyz 格式内联,简单分子直接写常量坐标(键长键角取教科书标准值,单位 Å),
 *     晶体点阵(NaCl/金刚石/石墨/C60/SiO₂)用循环程序生成,避免手写大段坐标出错。
 *   MOLECULE_2D_TEMPLATES — 2D 平面结构式(首发16 + 扩充14 = 30个):SMILES 串,
 *     覆盖原子较多、手写 3D 坐标不可靠的分子与高中有机主干。
 *
 * 数据自包含约定(全模块最高优先级):
 *   所有结构数据在构建期/模板文件内确定,生成的课件代码零外部数据请求——
 *   离线 ZIP 导出、断网课堂、内网部署三场景分子照常渲染。
 *
 * 新增 3D 模板规范:
 *   1. buildXYZ 产出标准 xyz:首行原子数,次行中文注释,逐行 "元素符号 x y z";
 *   2. 键长取教科书标准值(3Dmol 按共价半径距离自动推断化学键,
 *      坐标偏差过大会导致键缺失或误连,新增前先在弹窗预览确认成键正确);
 *   3. 分子几何中心尽量落在原点附近(zoomTo 取景更稳);
 *   4. 晶体类优先程序生成点阵,并在注释里写明晶格常数出处。
 *
 * 新增 2D 模板规范:SMILES 一律用单引号字符串;新增前在弹窗预览确认解析通过。
 * 分组并组规则:扩充条目沿用已有分组名即自动并入该组(getGroups 按组名归并)。
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/moleculeTemplates.ts
 */
import type { Molecule3DTemplate, Molecule2DTemplate } from './moleculeUtils'
import { MOLECULE_3D_TEMPLATES_EXT, MOLECULE_2D_TEMPLATES_EXT } from './moleculeTemplatesExt'

// ============================================================
// xyz 构造辅助
// ============================================================

/** 把坐标行数组包装成标准 xyz 格式(首行原子数 + 次行注释) */
function xyz(comment: string, lines: string[]): string {
  return lines.length + '\n' + comment + '\n' + lines.join('\n')
}

/**
 * NaCl 晶体点阵程序生成:简单立方交替排布,Na-Cl 间距 2.82 Å(岩盐结构标准值)。
 * 3×3×3 = 27 个原子,足够看清"每个Na⁺被6个Cl⁻包围"的配位关系。
 */
function buildNaClXYZ(): string {
  const a = 2.82
  const lines: string[] = []
  for (let i = 0; i < 3; i++) {
    for (let j = 0; j < 3; j++) {
      for (let k = 0; k < 3; k++) {
        const elem = (i + j + k) % 2 === 0 ? 'Na' : 'Cl'
        lines.push(elem + ' ' + (i * a).toFixed(3) + ' ' + (j * a).toFixed(3) + ' ' + (k * a).toFixed(3))
      }
    }
  }
  return xyz('氯化钠晶体点阵(3x3x3, a=2.82A)', lines)
}

/**
 * 金刚石晶体点阵程序生成:面心立方 + (1/4,1/4,1/4) 双原子基,晶格常数 3.567 Å。
 * 2×2×2 晶胞 = 64 个碳原子,可看清每个碳的正四面体成键(sp³)。
 */
function buildDiamondXYZ(): string {
  const a = 3.567
  const fcc: [number, number, number][] = [[0, 0, 0], [0.5, 0.5, 0], [0.5, 0, 0.5], [0, 0.5, 0.5]]
  const lines: string[] = []
  for (let cx = 0; cx < 2; cx++) {
    for (let cy = 0; cy < 2; cy++) {
      for (let cz = 0; cz < 2; cz++) {
        for (const p of fcc) {
          const x = (cx + p[0]) * a, y = (cy + p[1]) * a, z = (cz + p[2]) * a
          lines.push('C ' + x.toFixed(3) + ' ' + y.toFixed(3) + ' ' + z.toFixed(3))
          lines.push('C ' + (x + a / 4).toFixed(3) + ' ' + (y + a / 4).toFixed(3) + ' ' + (z + a / 4).toFixed(3))
        }
      }
    }
  }
  return xyz('金刚石晶体点阵(2x2x2晶胞, a=3.567A)', lines)
}

// ============================================================
// 3D 分子模板注册表(首发14个;文件尾聚合 Ext 扩充14个)
// ============================================================

const MOLECULE_3D_BASE: Molecule3DTemplate[] = [
  // ---------- 🌫️ 常见小分子 ----------
  {
    id: 'mol3d-h2o', group: '🌫️ 常见小分子', name: '水', formula: 'H₂O', emoji: '💧',
    desc: 'V形分子,键角约104.5°,极性分子的经典例子',
    buildXYZ: () => xyz('水分子(O-H 0.96A, 键角104.5°)', [
      'O 0.000 0.000 0.000',
      'H 0.757 0.586 0.000',
      'H -0.757 0.586 0.000',
    ]),
  },
  {
    id: 'mol3d-co2', group: '🌫️ 常见小分子', name: '二氧化碳', formula: 'CO₂', emoji: '🫧',
    desc: '直线形分子,虽含极性键但整体非极性',
    buildXYZ: () => xyz('二氧化碳(C=O 1.16A, 直线形)', [
      'C 0.000 0.000 0.000',
      'O 1.160 0.000 0.000',
      'O -1.160 0.000 0.000',
    ]),
  },
  {
    id: 'mol3d-o2', group: '🌫️ 常见小分子', name: '氧气', formula: 'O₂', emoji: '🌬️',
    desc: '双原子分子,O=O 双键',
    buildXYZ: () => xyz('氧气(O=O 1.21A)', [
      'O 0.605 0.000 0.000',
      'O -0.605 0.000 0.000',
    ]),
  },
  {
    id: 'mol3d-n2', group: '🌫️ 常见小分子', name: '氮气', formula: 'N₂', emoji: '🎈',
    desc: 'N≡N 三键,键能极大,化学性质稳定',
    buildXYZ: () => xyz('氮气(N≡N 1.10A)', [
      'N 0.550 0.000 0.000',
      'N -0.550 0.000 0.000',
    ]),
  },
  {
    id: 'mol3d-h2', group: '🌫️ 常见小分子', name: '氢气', formula: 'H₂', emoji: '🪶',
    desc: '最简单的分子,H-H 单键',
    buildXYZ: () => xyz('氢气(H-H 0.74A)', [
      'H 0.370 0.000 0.000',
      'H -0.370 0.000 0.000',
    ]),
  },
  {
    id: 'mol3d-hcl', group: '🌫️ 常见小分子', name: '氯化氢', formula: 'HCl', emoji: '🧪',
    desc: '强极性共价键,溶于水即盐酸',
    buildXYZ: () => xyz('氯化氢(H-Cl 1.27A)', [
      'H -0.640 0.000 0.000',
      'Cl 0.630 0.000 0.000',
    ]),
  },
  {
    id: 'mol3d-nh3', group: '🌫️ 常见小分子', name: '氨', formula: 'NH₃', emoji: '🌿',
    desc: '三角锥形,氮上有孤对电子,键角约107°',
    buildXYZ: () => xyz('氨分子(N-H 1.01A, 三角锥形)', [
      'N 0.000 0.000 0.000',
      'H 0.938 0.000 -0.382',
      'H -0.469 0.812 -0.382',
      'H -0.469 -0.812 -0.382',
    ]),
  },
  // ---------- 🔥 有机入门 ----------
  {
    id: 'mol3d-ch4', group: '🔥 有机入门', name: '甲烷', formula: 'CH₄', emoji: '🔥',
    desc: '正四面体结构,键角109.5°,最简单的有机物',
    buildXYZ: () => xyz('甲烷(C-H 1.09A, 正四面体)', [
      'C 0.000 0.000 0.000',
      'H 0.629 0.629 0.629',
      'H -0.629 -0.629 0.629',
      'H -0.629 0.629 -0.629',
      'H 0.629 -0.629 -0.629',
    ]),
  },
  {
    id: 'mol3d-c2h4', group: '🔥 有机入门', name: '乙烯', formula: 'C₂H₄', emoji: '🍌',
    desc: '平面形分子,C=C 双键,6个原子共平面',
    buildXYZ: () => xyz('乙烯(C=C 1.33A, 平面形)', [
      'C 0.667 0.000 0.000',
      'C -0.667 0.000 0.000',
      'H 1.230 0.920 0.000',
      'H 1.230 -0.920 0.000',
      'H -1.230 0.920 0.000',
      'H -1.230 -0.920 0.000',
    ]),
  },
  {
    id: 'mol3d-c2h2', group: '🔥 有机入门', name: '乙炔', formula: 'C₂H₂', emoji: '⚡',
    desc: '直线形分子,C≡C 三键,4个原子共直线',
    buildXYZ: () => xyz('乙炔(C≡C 1.20A, 直线形)', [
      'C 0.600 0.000 0.000',
      'C -0.600 0.000 0.000',
      'H 1.660 0.000 0.000',
      'H -1.660 0.000 0.000',
    ]),
  },
  {
    id: 'mol3d-c2h5oh', group: '🔥 有机入门', name: '乙醇', formula: 'C₂H₅OH', emoji: '🍶',
    desc: '含羟基官能团,乙基与羟基的空间关系',
    buildXYZ: () => xyz('乙醇(C-C 1.54A, C-O 1.43A)', [
      'C 0.000 0.000 0.000',
      'C 1.257 0.889 0.000',
      'O 2.424 0.064 0.000',
      'H 3.350 0.300 0.000',
      'H -0.360 -1.030 0.000',
      'H -0.360 0.360 0.970',
      'H -0.360 0.360 -0.970',
      'H 1.260 1.400 0.960',
      'H 1.260 1.400 -0.960',
    ]),
  },
  {
    id: 'mol3d-c6h6', group: '🔥 有机入门', name: '苯', formula: 'C₆H₆', emoji: '⬡',
    desc: '正六边形平面结构,碳碳键长完全相等(介于单双键之间)',
    buildXYZ: () => xyz('苯环(C-C 1.397A, 正六边形平面)', [
      'C 1.397 0.000 0.000',
      'C 0.699 1.210 0.000',
      'C -0.699 1.210 0.000',
      'C -1.397 0.000 0.000',
      'C -0.699 -1.210 0.000',
      'C 0.699 -1.210 0.000',
      'H 2.481 0.000 0.000',
      'H 1.241 2.149 0.000',
      'H -1.241 2.149 0.000',
      'H -2.481 0.000 0.000',
      'H -1.241 -2.149 0.000',
      'H 1.241 -2.149 0.000',
    ]),
  },
  // ---------- 🧂 晶体结构 ----------
  {
    id: 'mol3d-nacl', group: '🧂 晶体结构', name: '氯化钠晶体', formula: 'NaCl', emoji: '🧂',
    desc: '离子晶体,每个Na⁺被6个Cl⁻包围(6:6配位)',
    buildXYZ: buildNaClXYZ,
  },
  {
    id: 'mol3d-diamond', group: '🧂 晶体结构', name: '金刚石', formula: 'C(金刚石)', emoji: '💎',
    desc: '共价晶体,每个碳以sp³与4个碳成键构成正四面体网络',
    buildXYZ: buildDiamondXYZ,
  },
]

// ============================================================
// 2D 结构式模板注册表(首发16个;文件尾聚合 Ext 扩充14个)
// ============================================================

const MOLECULE_2D_BASE: Molecule2DTemplate[] = [
  // ---------- ⚗️ 无机与酸碱 ----------
  { id: 'mol2d-h2o', group: '⚗️ 无机与酸碱', name: '水', formula: 'H₂O', smiles: 'O', desc: '生命之源' },
  { id: 'mol2d-co2', group: '⚗️ 无机与酸碱', name: '二氧化碳', formula: 'CO₂', smiles: 'O=C=O', desc: '两个碳氧双键' },
  { id: 'mol2d-nh3', group: '⚗️ 无机与酸碱', name: '氨', formula: 'NH₃', smiles: 'N', desc: '刺激性气味气体' },
  { id: 'mol2d-h2so4', group: '⚗️ 无机与酸碱', name: '硫酸', formula: 'H₂SO₄', smiles: 'OS(=O)(=O)O', desc: '三大强酸之一' },
  { id: 'mol2d-hno3', group: '⚗️ 无机与酸碱', name: '硝酸', formula: 'HNO₃', smiles: 'O[N+](=O)[O-]', desc: '强氧化性酸' },
  { id: 'mol2d-h2co3', group: '⚗️ 无机与酸碱', name: '碳酸', formula: 'H₂CO₃', smiles: 'OC(=O)O', desc: '不稳定的二元弱酸' },
  // ---------- 🧬 有机基础 ----------
  { id: 'mol2d-ch4', group: '🧬 有机基础', name: '甲烷', formula: 'CH₄', smiles: 'C', desc: '最简单的烷烃' },
  { id: 'mol2d-c2h6', group: '🧬 有机基础', name: '乙烷', formula: 'C₂H₆', smiles: 'CC', desc: '碳碳单键' },
  { id: 'mol2d-c2h4', group: '🧬 有机基础', name: '乙烯', formula: 'C₂H₄', smiles: 'C=C', desc: '碳碳双键,加成反应' },
  { id: 'mol2d-c2h2', group: '🧬 有机基础', name: '乙炔', formula: 'C₂H₂', smiles: 'C#C', desc: '碳碳三键' },
  { id: 'mol2d-c2h5oh', group: '🧬 有机基础', name: '乙醇', formula: 'C₂H₅OH', smiles: 'CCO', desc: '羟基官能团' },
  { id: 'mol2d-ch3cooh', group: '🧬 有机基础', name: '乙酸', formula: 'CH₃COOH', smiles: 'CC(=O)O', desc: '羧基官能团,食醋主要成分' },
  { id: 'mol2d-c6h6', group: '🧬 有机基础', name: '苯', formula: 'C₆H₆', smiles: 'c1ccccc1', desc: '芳香环' },
  // ---------- 🍬 生活中的分子 ----------
  { id: 'mol2d-glucose', group: '🍬 生活中的分子', name: '葡萄糖', formula: 'C₆H₁₂O₆', smiles: 'OCC1OC(O)C(O)C(O)C1O', desc: '六碳糖,细胞能量来源' },
  { id: 'mol2d-aspirin', group: '🍬 生活中的分子', name: '阿司匹林', formula: 'C₉H₈O₄', smiles: 'CC(=O)Oc1ccccc1C(=O)O', desc: '乙酰水杨酸,经典药物分子' },
  { id: 'mol2d-caffeine', group: '🍬 生活中的分子', name: '咖啡因', formula: 'C₈H₁₀N₄O₂', smiles: 'Cn1cnc2c1c(=O)n(C)c(=O)n2C', desc: '咖啡与茶中的活性成分' },
]

// ============================================================
// 聚合出口(首发 + 批次2c课标扩充;同组名自动并组,弹窗零改动)
// ============================================================

/** 全量 3D 模板(28个) */
export const MOLECULE_3D_TEMPLATES: Molecule3DTemplate[] = [
  ...MOLECULE_3D_BASE,
  ...MOLECULE_3D_TEMPLATES_EXT,
]

/** 全量 2D 模板(30个) */
export const MOLECULE_2D_TEMPLATES: Molecule2DTemplate[] = [
  ...MOLECULE_2D_BASE,
  ...MOLECULE_2D_TEMPLATES_EXT,
]

// ============================================================
// 查询辅助(签名范式对齐 mathGraphTemplates)
// ============================================================

/** 3D 模板分组列表(保持注册表顺序,供弹窗左栏分组渲染;同组名条目自动归并) */
export function getMolecule3DGroups(): { group: string; items: Molecule3DTemplate[] }[] {
  const groups: { group: string; items: Molecule3DTemplate[] }[] = []
  for (const t of MOLECULE_3D_TEMPLATES) {
    let g = groups.find(x => x.group === t.group)
    if (!g) { g = { group: t.group, items: [] }; groups.push(g) }
    g.items.push(t)
  }
  return groups
}

/** 2D 模板分组列表 */
export function getMolecule2DGroups(): { group: string; items: Molecule2DTemplate[] }[] {
  const groups: { group: string; items: Molecule2DTemplate[] }[] = []
  for (const t of MOLECULE_2D_TEMPLATES) {
    let g = groups.find(x => x.group === t.group)
    if (!g) { g = { group: t.group, items: [] }; groups.push(g) }
    g.items.push(t)
  }
  return groups
}

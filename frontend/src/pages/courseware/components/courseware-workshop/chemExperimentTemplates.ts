/**
 * chemExperimentTemplates.ts — 化学实验模板聚合入口
 *
 * 第6批A1：
 *   - base: 首批基础模板；
 *   - ext/ext2/ext3/ext4/ext5/ext6: 六批扩展模板；
 *   - 对外导出保持不变，ChemExperimentModal.tsx 无需修改。
 */

import type { ChemExperimentTemplate } from './chemExperimentUtils'
import { CHEM_EXPERIMENT_TEMPLATES as CHEM_EXPERIMENT_TEMPLATES_BASE } from './chemExperimentTemplatesBase'
import { CHEM_EXPERIMENT_TEMPLATES_EXT } from './chemExperimentTemplatesExt'
import { CHEM_EXPERIMENT_TEMPLATES_EXT2 } from './chemExperimentTemplatesExt2'
import { CHEM_EXPERIMENT_TEMPLATES_EXT3 } from './chemExperimentTemplatesExt3'
import { CHEM_EXPERIMENT_TEMPLATES_EXT4 } from './chemExperimentTemplatesExt4'
import { CHEM_EXPERIMENT_TEMPLATES_EXT5 } from './chemExperimentTemplatesExt5'
import { CHEM_EXPERIMENT_TEMPLATES_EXT6 } from './chemExperimentTemplatesExt6'

export const CHEM_EXPERIMENT_TEMPLATES: ChemExperimentTemplate[] = [
  ...CHEM_EXPERIMENT_TEMPLATES_BASE,
  ...CHEM_EXPERIMENT_TEMPLATES_EXT,
  ...CHEM_EXPERIMENT_TEMPLATES_EXT2,
  ...CHEM_EXPERIMENT_TEMPLATES_EXT3,
  ...CHEM_EXPERIMENT_TEMPLATES_EXT4,
  ...CHEM_EXPERIMENT_TEMPLATES_EXT5,
  ...CHEM_EXPERIMENT_TEMPLATES_EXT6,
]

export function getChemExperimentGroups(): { group: string; items: ChemExperimentTemplate[] }[] {
  const groups: { group: string; items: ChemExperimentTemplate[] }[] = []
  for (const t of CHEM_EXPERIMENT_TEMPLATES) {
    let g = groups.find(x => x.group === t.group)
    if (!g) {
      g = { group: t.group, items: [] }
      groups.push(g)
    }
    g.items.push(t)
  }
  return groups
}

export function findChemExperimentTemplate(id: string): ChemExperimentTemplate | undefined {
  return CHEM_EXPERIMENT_TEMPLATES.find(t => t.id === id)
}

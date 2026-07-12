/**
 * physicsLabTemplates.ts — 物理实验室模板聚合入口
 *
 * 第6批A1：
 *   - base: 首批基础模板；
 *   - ext/ext2/ext3/ext4/ext5/ext6: 六批扩展模板；
 *   - 对外导出保持不变，PhysicsLabModal.tsx 无需修改。
 */

import type { PhysicsLabTemplate } from './physicsLabUtils'
import { PHYSICS_LAB_TEMPLATES as PHYSICS_LAB_TEMPLATES_BASE } from './physicsLabTemplatesBase'
import { PHYSICS_LAB_TEMPLATES_EXT } from './physicsLabTemplatesExt'
import { PHYSICS_LAB_TEMPLATES_EXT2 } from './physicsLabTemplatesExt2'
import { PHYSICS_LAB_TEMPLATES_EXT3 } from './physicsLabTemplatesExt3'
import { PHYSICS_LAB_TEMPLATES_EXT4 } from './physicsLabTemplatesExt4'
import { PHYSICS_LAB_TEMPLATES_EXT5 } from './physicsLabTemplatesExt5'
import { PHYSICS_LAB_TEMPLATES_EXT6 } from './physicsLabTemplatesExt6'

export const PHYSICS_LAB_TEMPLATES: PhysicsLabTemplate[] = [
  ...PHYSICS_LAB_TEMPLATES_BASE,
  ...PHYSICS_LAB_TEMPLATES_EXT,
  ...PHYSICS_LAB_TEMPLATES_EXT2,
  ...PHYSICS_LAB_TEMPLATES_EXT3,
  ...PHYSICS_LAB_TEMPLATES_EXT4,
  ...PHYSICS_LAB_TEMPLATES_EXT5,
  ...PHYSICS_LAB_TEMPLATES_EXT6,
]

export function getPhysicsLabGroups(): { group: string; items: PhysicsLabTemplate[] }[] {
  const groups: { group: string; items: PhysicsLabTemplate[] }[] = []
  for (const t of PHYSICS_LAB_TEMPLATES) {
    let g = groups.find(x => x.group === t.group)
    if (!g) {
      g = { group: t.group, items: [] }
      groups.push(g)
    }
    g.items.push(t)
  }
  return groups
}

export function findPhysicsLabTemplate(id: string): PhysicsLabTemplate | undefined {
  return PHYSICS_LAB_TEMPLATES.find(t => t.id === id)
}

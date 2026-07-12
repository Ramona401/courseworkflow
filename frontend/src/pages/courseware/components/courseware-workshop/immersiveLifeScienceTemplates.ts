/**
 * immersiveLifeScienceTemplates.ts — 3D生命科学整页实验注册表
 *
 * 定位：
 *   - 与轻量 LifeScienceLabTemplate 分离；
 *   - 每个条目对应一个完整的全屏3D实验页面；
 *   - 前端只保存元数据和静态入口，不把数万行实验源码打进主JS包；
 *   - 新实验只需新增静态资产目录并登记一条元数据。
 */

export interface ImmersiveLifeScienceTemplate {
  id: string
  name: string
  emoji: string
  category: string
  summary: string
  description: string
  stage: string
  knowledgePoints: string[]
  capabilities: string[]
  previewUrl: string
  sourceUrl: string
  accent: string
  softBackground: string
}

const LAB_BASE = '/uploads/courseware-assets/immersive-labs/life-science'

export const IMMERSIVE_LIFE_SCIENCE_TEMPLATES: ImmersiveLifeScienceTemplate[] = [
  {
    id: 'immersive-plant-cell',
    name: '3D植物细胞工作室',
    emoji: '🌿',
    category: '细胞结构',
    summary: '从三维空间观察植物细胞及其主要细胞器，并与真实显微图像对照。',
    description:
      '完整的植物细胞三维学习工作台，支持旋转、缩放、平移、点击选择、结构隐藏、透视、隔离、聚焦、标签切换和多倍显微观察。',
    stage: '初中—高中',
    knowledgePoints: [
      '植物细胞基本结构',
      '细胞器形态与功能',
      '植物细胞与动物细胞差异',
      '显微观察与模型对照',
    ],
    capabilities: ['3D旋转缩放', '细胞器点击选择', '透视与隔离', '实景显微图对照', '100×/400×/1000×观察'],
    previewUrl: LAB_BASE + '/plant-cell/index.html',
    sourceUrl: LAB_BASE + '/plant-cell/index.html',
    accent: '#5B8F2A',
    softBackground: 'linear-gradient(135deg,#F2F8E5,#FFF9EF)',
  },
  {
    id: 'immersive-animal-cell',
    name: '3D动物细胞工作室',
    emoji: '🔬',
    category: '细胞结构',
    summary: '观察动物细胞内部结构、内膜系统与细胞分裂相关结构。',
    description:
      '完整的动物细胞三维学习工作台，覆盖细胞膜、细胞质、细胞核、线粒体、内质网、高尔基体、溶酶体、中心体和核糖体等结构。',
    stage: '初中—高中',
    knowledgePoints: [
      '动物细胞基本结构',
      '细胞器形态与功能',
      '内膜系统',
      '动物细胞与植物细胞差异',
    ],
    capabilities: ['3D旋转缩放', '细胞器点击选择', '透视与隔离', '实景显微图对照', '100×/400×/1000×观察'],
    previewUrl: LAB_BASE + '/animal-cell/index.html',
    sourceUrl: LAB_BASE + '/animal-cell/index.html',
    accent: '#C45A3A',
    softBackground: 'linear-gradient(135deg,#FDF0E5,#FFF7F0)',
  },
]

export function findImmersiveLifeScienceTemplate(
  id: string,
): ImmersiveLifeScienceTemplate | undefined {
  return IMMERSIVE_LIFE_SCIENCE_TEMPLATES.find(template => template.id === id)
}

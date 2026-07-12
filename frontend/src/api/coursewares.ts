/**
 * 课件工坊 API —— 统一出口（桶文件 coursewares.ts）
 *
 * 本文件由原 1474 行单体拆分而来，自身不再含实现，仅 re-export 各子模块：
 *   - coursewares.types  类型定义 + UI 常量配置 + extractData
 *   - coursewares.core   课件CRUD/页面/状态流转/索引SSE/模板/组件/预设/多入口创建/3D/发布共享
 *   - coursewares.media  图片/视频/字幕/TTS/OSS/AI写提示词/离线包
 *   - coursewares.bg     课件背景图库（图集列表/选择秒换/AI生成/上传/归档/升级系统库）
 *   - coursewares.font   课件字体方案（5套OFL预设/选择秒换/清除）
 *   - coursewares.collab 课件协作（阶段2：页级批注 增/列/删/标记已处理）
 *   - coursewares.review 课件多级审核（阶段3：提交审核/L1/L2/历史/详情/待审/已审/统计）
 *   - coursewares.inlineedit 就地文字编辑保存 + 粘贴HTML导入（批次A/B，确定性整页覆盖类接口）
 *   - coursewares.snippets   代码收藏库（批次C：打星收藏页面HTML快照，微调时注入参考代码）
 *
 * 作用：保持既有 `import { X } from '@/api/coursewares'` 的全部命名导入路径不变，
 * 拆分对所有调用方透明（零路径改动）。新增 API 请按域加进对应子模块，勿写回本文件。
 */
export * from './coursewares.types'
export * from './coursewares.core'
export * from './coursewares.media'
export * from './coursewares.bg'
export * from './coursewares.font'
export * from './coursewares.collab'
export * from './coursewares.collabsession'
export * from './coursewares.review'
export * from './coursewares.inlineedit'
export * from './coursewares.snippets'

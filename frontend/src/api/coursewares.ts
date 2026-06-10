/**
 * 课件工坊 API —— 统一出口（桶文件 coursewares.ts）
 *
 * 本文件由原 1474 行单体拆分而来，自身不再含实现，仅 re-export 三个子模块：
 *   - coursewares.types  类型定义 + UI 常量配置 + extractData
 *   - coursewares.core   课件CRUD/页面/状态流转/索引SSE/模板/组件/预设/多入口创建/3D
 *   - coursewares.media  图片/视频/字幕/TTS/OSS/AI写提示词/离线包
 *
 * 作用：保持既有 `import { X } from '@/api/coursewares'` 的全部命名导入路径不变，
 * 拆分对所有调用方透明（零路径改动）。新增 API 请按域加进对应子模块，勿写回本文件。
 */
export * from './coursewares.types'
export * from './coursewares.core'
export * from './coursewares.media'
export * from './coursewares.bg'

/**
 * 课件工坊 API —— 统一出口。
 *
 * coursewares.core继续转出：
 *   - coursewares.sse；
 *   - coursewares.page-mutation；
 *   - coursewares.catalog；
 *   - coursewares.creation；
 *   - coursewares.sharing。
 *
 * 这样既保持统一出口，也兼容历史代码直接从coursewares.core导入。
 * 本文件只re-export业务模块，不实现请求逻辑。
 */

export * from './coursewares.types'
export * from './coursewares.core'
export * from './coursewares.media'
export * from './coursewares.video-first-frame'
export * from './coursewares.bg'
export * from './coursewares.font'
export * from './coursewares.collab'
export * from './coursewares.collabsession'
export * from './coursewares.review'
export * from './coursewares.ai-review'
export * from './coursewares.ai-review-instruction-versions'
export * from './coursewares.ai-review-resolution'
export * from './coursewares.ai-review-governance'
export * from './coursewares.inlineedit'
export * from './coursewares.snippets'
export * from './coursewares.assistant.types'
export * from './coursewares.assistant.teacher'
export * from './coursewares.assistant.runtime'
export * from './courseware-add-page-discussion'
export * from './coursewares.comic'
export * from './coursewares.comic.workflow'
export * from './coursewares.comic.export'
export * from './coursewareAssembly'

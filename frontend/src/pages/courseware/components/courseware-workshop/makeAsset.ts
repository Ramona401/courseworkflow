/**
 * makeAsset.ts — 课件本地资产对象工厂（批次5a引入，批次5b拆分时独立为共享模块）
 *
 * 背景：生成/上传类接口成功后只返回少量字段（asset_id/url等），前端需立即把
 * 新资产插入列表即时展示。统一收口：调用方只传差异字段，其余给安全默认值
 * （spread 在后，partial 覆盖默认），消除逐字段手工构造的重复与漏写风险。
 *
 * 注意：page_id 恒为 null —— 这是"前端本地即时展示对象"的固有口径：后端真实
 * 记录已带页归属，刷新/重拉 listPageAssets 后按页可见；本地对象仅为免刷新体验。
 *
 * 调用方：MediaManagerPanel（单张生成/参考图/手动上传/编辑器上传/导出成片）、
 *         MediaImageSuggestPanel（批量生成）。
 */
import type { CoursewareAsset } from '@/api/coursewares'

export function makeAsset(
  coursewareId: string,
  partial: Partial<CoursewareAsset> & Pick<CoursewareAsset, 'id' | 'oss_url'>,
): CoursewareAsset {
  return {
    courseware_id: coursewareId,
    page_id: null,
    placeholder_id: '',
    asset_type: 'image',
    generation_prompt: '',
    file_size: 0,
    mime_type: 'image/png',
    status: 'uploaded',
    created_at: new Date().toISOString(),
    ...partial,
  }
}

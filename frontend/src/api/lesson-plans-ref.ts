/**
 * lesson-plans-ref.ts — 备课参考资料附件（PDF/Word）压缩端点封装
 *
 * 对应后端 POST /api/v1/lesson-plans/ref-material/compress。
 * 前端在浏览器端提取出的长参考资料原文（≥3000字）POST 到此，后端 AI 压成结构化要点返回。
 * 短文档（<3000字）不调此端点，直接把原文作为 ref_material 注入，省一次 AI 调用。
 *
 * 封装范式对齐 importExistingPlan（apiClient.post + resp.data.data）。
 */
import apiClient from './client'

/** 参考资料压缩请求体 */
export interface CompressRefMaterialRequest {
  content: string    // 前端提取出的原文（必填）
  file_name?: string // 文件名（可选，供 AI 上下文与日志）
  subject?: string   // 学科（可选，压缩时聚焦）
  grade?: string     // 年级（可选，压缩时聚焦）
}

/** 参考资料压缩响应体 */
export interface CompressRefMaterialResponse {
  compressed: string      // 压缩后的结构化要点（注入用）
  original_len: number    // 原文字数
  compressed_len: number  // 压缩后字数
}

/** 压缩长参考资料为结构化要点 */
export async function compressRefMaterial(
  data: CompressRefMaterialRequest
): Promise<CompressRefMaterialResponse> {
  const resp = await apiClient.post('/lesson-plans/ref-material/compress', data)
  return resp.data.data as CompressRefMaterialResponse
}

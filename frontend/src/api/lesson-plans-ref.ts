/**
 * lesson-plans-ref.ts — 备课参考资料附件处理 API
 *
 * 同一个后端端点承载两种模式：
 *   - compress_text：压缩浏览器已提取的长文本；
 *   - vision_transcribe：忠实转录扫描 PDF 的单页图片。
 *
 * 原始文件不永久上传、不落库；扫描页图片仅随单次请求传输。
 */

import apiClient from './client'

export interface CompressRefMaterialRequest {
  content: string
  file_name?: string
  subject?: string
  grade?: string
}

export interface CompressRefMaterialResponse {
  compressed: string
  original_len: number
  compressed_len: number
}

export interface TranscribeRefMaterialPageRequest {
  image_data_uri: string
  file_name?: string
  page_number: number
  total_pages: number
  subject?: string
  grade?: string
}

export interface TranscribeRefMaterialPageResponse {
  text: string
  page_number: number
  total_pages: number
}

/** 压缩长参考资料为结构化要点。 */
export async function compressRefMaterial(
  data: CompressRefMaterialRequest,
): Promise<CompressRefMaterialResponse> {
  const resp =
    await apiClient.post(
      '/lesson-plans/ref-material/compress',
      {
        mode: 'compress_text',
        ...data,
      },
    )

  return resp.data
    .data as CompressRefMaterialResponse
}

/** 对扫描 PDF 的单页清晰图进行忠实视觉转录。 */
export async function transcribeRefMaterialPage(
  data: TranscribeRefMaterialPageRequest,
): Promise<TranscribeRefMaterialPageResponse> {
  const resp =
    await apiClient.post(
      '/lesson-plans/ref-material/compress',
      {
        mode: 'vision_transcribe',
        ...data,
      },
    )

  return resp.data
    .data as TranscribeRefMaterialPageResponse
}

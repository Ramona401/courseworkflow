/**
 * 课件视频分镜首帧 API。
 *
 * 独立于普通课件图片生成接口：
 *   - 后端路由固定使用video_first_frame计费节点；
 *   - 浏览器不能提交billing_node_code；
 *   - 一次首帧业务操作使用一个UUID；
 *   - 同一次失败重试可显式复用原UUID，恢复已有计费或资产结果。
 */

import apiClient from './client'
import {
  extractData,
} from './coursewares.types'
import type {
  GenerateImageResponse,
} from './coursewares.types'

/** 创建视频首帧业务操作UUID。 */
export function createCWVideoFirstFrameOperationID(): string {
  const cryptoAPI = globalThis.crypto

  if (
    !cryptoAPI ||
    typeof cryptoAPI.randomUUID !== 'function'
  ) {
    throw new Error(
      '当前浏览器不支持安全UUID生成，请升级浏览器后重试',
    )
  }

  return cryptoAPI.randomUUID()
}

/**
 * 生成视频分镜首帧。
 *
 * operationId为空时创建新业务操作；
 * 调用方在网络失败后重试时应传回原operationId。
 */
export async function generateCWVideoFirstFrame(
  coursewareId: string,
  pageNumber: number,
  prompt: string,
  refImageUrl?: string,
  operationId?: string,
): Promise<GenerateImageResponse> {
  const stableOperationID =
    operationId?.trim() ||
    createCWVideoFirstFrameOperationID()

  const body: Record<string, string> = {
    prompt,
    operation_id: stableOperationID,
  }

  if (refImageUrl) {
    body.ref_image_url = refImageUrl
  }

  const response = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/generate-video-first-frame`,
    body,
    {
      timeout: 60000,
    },
  )

  return extractData(response)
}

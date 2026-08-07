/**
 * coursewares.page-mutation.ts
 *
 * 课件单页AI修改、重新生成、页面历史与回退API。
 *
 * 从coursewares.core.ts按页级写入职责拆出，避免核心API文件继续超过900行。
 *
 * 整改项页面应用安全边界：
 *   1. 普通页面微调不需要整改项或指令版本；
 *   2. 携带review_item_id时，前端必须先读取后端认可的当前可见版本；
 *   3. 正式整改作者读取到的是正式交付版本，自审作者读取到的是当前确认版本；
 *   4. 页面应用请求同时提交review_item_id和instruction_version_id；
 *   5. 读取版本后若发生并发切换，后端会以409拒绝旧版本；
 *   6. 当前草稿必须完整包含可信版本正文，否则不进入耗时AI页面修改。
 */

import apiClient from './client'
import { extractData } from './coursewares.types'
import type { PageVersionEntry } from './coursewares.types'
import { getCurrentCWAIReviewInstructionVersion } from './coursewares.ai-review-instruction-versions'

export type CWRefineMode = 'preserve' | 'rebuild'

export interface CWRefinePageResponse {
  page_number: number
  html_content: string
  message: string
  mode?: CWRefineMode
  review_item_id?: string
  instruction_version_id?: string
  review_item_status?: string
  review_item_warning?: string
}

/**
 * 单页AI微调。
 *
 * reviewItemId非空时，必须先解析当前可执行版本并绑定到本次请求。
 * 该读取不会替代后端事务校验，只用于形成明确的乐观并发版本ID和提前反馈。
 */
export async function refinePage(
  coursewareId: string,
  pageNumber: number,
  instruction: string,
  image?: string,
  mode: CWRefineMode = 'preserve',
  reviewItemId?: string,
): Promise<CWRefinePageResponse> {
  const normalizedInstruction = instruction.trim()
  const normalizedReviewItemId = reviewItemId?.trim() || ''

  if (!normalizedInstruction && !image) {
    throw new Error('请提供修改意见或粘贴截图')
  }

  const body: Record<string, string> = {
    instruction,
    mode,
  }

  if (image) {
    body.image = image
  }

  if (normalizedReviewItemId) {
    const versionResult = await getCurrentCWAIReviewInstructionVersion(
      normalizedReviewItemId,
    )

    const version = versionResult.version
    const instructionVersionId =
      versionResult.current_instruction_version_id?.trim()
      || version?.id.trim()
      || ''

    if (!version || !instructionVersionId) {
      throw new Error('整改项尚未形成可执行的确认指令版本，请刷新整改中心后重试')
    }

    if (version.status !== 'confirmed') {
      throw new Error('整改项当前指令版本已经失效，请刷新整改中心后重新检查')
    }

    const trustedInstruction = version.content.trim()

    if (
      !trustedInstruction
      || !normalizedInstruction.includes(trustedInstruction)
    ) {
      throw new Error(
        `当前微调草稿未完整包含确认的V${version.version_no}整改指令，请重新从整改中心注入后再执行`,
      )
    }

    body.review_item_id = normalizedReviewItemId
    body.instruction_version_id = instructionVersionId
  }

  const response = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/refine`,
    body,
    {
      timeout: 300000,
    },
  )

  return extractData<CWRefinePageResponse>(response)
}

/**
 * 单页从零重新生成。
 *
 * 本入口不是整改项指令应用入口，不自动绑定review_item_id。
 */
export async function regenerateCWPage(
  coursewareId: string,
  pageNumber: number,
): Promise<{
  page_number: number
  html_content: string
  message: string
}> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/regenerate`,
    {},
    {
      timeout: 300000,
    },
  )

  return extractData(response)
}

/**
 * 获取某页版本快照列表。
 */
export async function listPageVersions(
  coursewareId: string,
  pageNumber: number,
): Promise<{
  page_number: number
  versions: PageVersionEntry[]
  total: number
}> {
  const response = await apiClient.get(
    `/coursewares/${coursewareId}/pages/${pageNumber}/versions`,
  )

  return extractData(response)
}

/**
 * 回退某页到指定历史版本。
 *
 * 后端会先保存当前页面，因此回退操作本身可逆。
 */
export async function rollbackPage(
  coursewareId: string,
  pageNumber: number,
  versionId: string,
): Promise<{
  page_number: number
  html_content: string
  message: string
}> {
  const response = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/rollback`,
    {
      version_id: versionId,
    },
  )

  return extractData(response)
}

/**
 * 获取指定页面历史版本的完整HTML。
 */
export async function getPageVersionDetail(
  coursewareId: string,
  pageNumber: number,
  versionId: string,
): Promise<{
  page_number: number
  version_id: string
  version_no: number
  source: string
  source_label: string
  html_content: string
}> {
  const response = await apiClient.get(
    `/coursewares/${coursewareId}/pages/${pageNumber}/versions/${versionId}`,
  )

  return extractData(response)
}

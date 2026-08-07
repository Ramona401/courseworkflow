/**
 * coursewares.comic.export.ts
 *
 * 知识点漫画导出专用资产查询：
 *   - 复用现有课件资产列表API，不新增数据库或后端接口；
 *   - 根据漫画格保存的current_asset_id定位正式图片资产；
 *   - 优先返回同源oss_url，避免OSS公网地址缺少CORS导致浏览器fetch失败；
 *   - 仅返回导出所需的资产ID与可读取URL，不改变正式资产记录。
 */

import apiClient from './client'

import {
  extractData,
} from './coursewares.types'

interface CoursewareComicExportAssetRecord {
  id: string
  courseware_id: string
  asset_type: string
  oss_url: string
  public_oss_url?: string | null
}

interface CoursewareComicExportAssetListResponse {
  assets: CoursewareComicExportAssetRecord[] | null
  total: number
}

function requiredPathSegment(
  value: string,
): string {
  const normalized =
    value.trim()

  if (!normalized) {
    throw new Error(
      '课件ID不能为空。',
    )
  }

  return encodeURIComponent(
    normalized,
  )
}

/**
 * listCoursewareComicExportAssetURLs
 *
 * 通过现有课件资产列表读取漫画格对应的正式资产地址。
 * 同一资产同时存在本地oss_url与公网public_oss_url时，优先同源地址，
 * 使浏览器可以安全读取为Blob并转换为导出用data URL。
 */
export async function listCoursewareComicExportAssetURLs(
  coursewareId: string,
  assetIDs: string[],
): Promise<Map<string, string>> {
  const normalizedCoursewareID =
    coursewareId.trim()

  const requestedIDs =
    new Set(
      assetIDs
        .map(value => value.trim())
        .filter(Boolean),
    )

  const result =
    new Map<string, string>()

  if (
    !normalizedCoursewareID ||
    requestedIDs.size === 0
  ) {
    return result
  }

  const response =
    await apiClient.get(
      `/coursewares/${requiredPathSegment(
        normalizedCoursewareID,
      )}/assets`,
    )

  const data =
    extractData<CoursewareComicExportAssetListResponse>(
      response,
    )

  for (
    const asset of
      data.assets || []
  ) {
    const assetID =
      typeof asset?.id === 'string'
        ? asset.id.trim()
        : ''

    if (
      !assetID ||
      !requestedIDs.has(
        assetID,
      ) ||
      asset.courseware_id !==
        normalizedCoursewareID ||
      asset.asset_type !==
        'image'
    ) {
      continue
    }

    const resolvedURL =
      resolvePreferredExportAssetURL(
        asset.oss_url,
        asset.public_oss_url || '',
      )

    if (resolvedURL) {
      result.set(
        assetID,
        resolvedURL,
      )
    }
  }

  return result
}

function resolvePreferredExportAssetURL(
  localValue: string,
  publicValue: string,
): string {
  const localURL =
    normalizeHTTPAssetURL(
      localValue,
    )

  const publicURL =
    normalizeHTTPAssetURL(
      publicValue,
    )

  if (
    localURL &&
    isSameOriginURL(
      localURL,
    )
  ) {
    return localURL
  }

  if (
    publicURL &&
    isSameOriginURL(
      publicURL,
    )
  ) {
    return publicURL
  }

  return localURL || publicURL
}

function normalizeHTTPAssetURL(
  value: string,
): string {
  const normalized =
    typeof value === 'string'
      ? value.trim()
      : ''

  if (!normalized) {
    return ''
  }

  try {
    const resolved =
      new URL(
        normalized,
        window.location.origin,
      )

    if (
      resolved.protocol !== 'http:' &&
      resolved.protocol !== 'https:'
    ) {
      return ''
    }

    return resolved.toString()
  } catch {
    return ''
  }
}

function isSameOriginURL(
  value: string,
): boolean {
  try {
    return new URL(
      value,
    ).origin ===
      window.location.origin
  } catch {
    return false
  }
}

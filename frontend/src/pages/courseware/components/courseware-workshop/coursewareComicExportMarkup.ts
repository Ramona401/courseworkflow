/**
 * coursewareComicExportMarkup.ts
 *
 * 知识点漫画导出的资产准备模块：
 *   - 校验全部分格已经生成并具有稳定资产ID；
 *   - 复用现有课件资产列表API，优先读取同源本地图片地址；
 *   - 把图片转换为data URL，供原生Canvas安全绘制；
 *   - 提供统一的画幅比例解析和图片解码入口；
 *   - 不再生成HTML覆盖层，也不再使用SVG foreignObject截图。
 */

import type {
  CoursewareComicAspectRatio,
  CoursewareComicPanel,
  CoursewareComicWorkflowProject,
} from '@/api/coursewares'

import {
  listCoursewareComicExportAssetURLs,
} from '@/api/coursewares'

export interface PreparedCoursewareComicPanel {
  panel: CoursewareComicPanel
  imageDataURL: string
}

export interface CoursewareComicExportAspect {
  width: number
  height: number
}

export async function prepareCoursewareComicExportPanels(
  project: CoursewareComicWorkflowProject,
  panels: CoursewareComicPanel[],
): Promise<PreparedCoursewareComicPanel[]> {
  const ordered = [...panels].sort(
    (left, right) =>
      left.panel_no -
      right.panel_no,
  )

  if (ordered.length < 1) {
    throw new Error(
      '当前漫画没有可导出的分格。',
    )
  }

  const incomplete =
    ordered.find(
      panel =>
        panel.status !==
          'generated' ||
        !panel.current_asset_id
          ?.trim() ||
        !panel.current_asset_url
          .trim(),
    )

  if (incomplete) {
    throw new Error(
      `第${incomplete.panel_no}格图片尚未完成，暂时不能导出完整漫画。`,
    )
  }

  const assetURLs =
    await listCoursewareComicExportAssetURLs(
      project.courseware_id,
      ordered
        .map(
          panel =>
            panel.current_asset_id ||
            '',
        )
        .filter(Boolean),
    )

  return Promise.all(
    ordered.map(
      async panel => {
        const assetID =
          panel.current_asset_id
            ?.trim() ||
          ''

        const preferredURL =
          assetURLs.get(
            assetID,
          ) ||
          panel.current_asset_url

        try {
          return {
            panel,
            imageDataURL:
              await fetchImageAsDataURL(
                preferredURL,
              ),
          }
        } catch (error) {
          const reason =
            error instanceof Error &&
            error.message.trim()
              ? error.message
              : '图片读取失败'

          throw new Error(
            `第${panel.panel_no}格${reason}`,
          )
        }
      },
    ),
  )
}

export function resolveCoursewareComicExportAspect(
  value: CoursewareComicAspectRatio,
): CoursewareComicExportAspect {
  switch (value) {
  case '4:3':
    return {
      width: 4,
      height: 3,
    }

  case '1:1':
    return {
      width: 1,
      height: 1,
    }

  case '3:4':
    return {
      width: 3,
      height: 4,
    }

  case '9:16':
    return {
      width: 9,
      height: 16,
    }

  default:
    return {
      width: 16,
      height: 9,
    }
  }
}

export function loadCoursewareComicExportImage(
  dataURL: string,
): Promise<HTMLImageElement> {
  return new Promise(
    (
      resolve,
      reject,
    ) => {
      const image =
        new Image()

      image.decoding =
        'async'

      image.onload = () =>
        resolve(image)

      image.onerror = () =>
        reject(
          new Error(
            '漫画底图解码失败。',
          ),
        )

      image.src = dataURL
    },
  )
}

async function fetchImageAsDataURL(
  value: string,
): Promise<string> {
  const normalized =
    value.trim()

  if (!normalized) {
    throw new Error(
      '漫画图片地址为空。',
    )
  }

  if (
    normalized.startsWith(
      'data:',
    )
  ) {
    return normalized
  }

  const resolved =
    new URL(
      normalized,
      window.location.href,
    )

  let response: Response

  try {
    response =
      await fetch(
        resolved.toString(),
        {
          credentials:
            resolved.origin ===
              window.location.origin
              ? 'include'
              : 'omit',
          cache:
            'force-cache',
        },
      )
  } catch {
    throw new Error(
      '图片读取失败，请刷新页面后重试。',
    )
  }

  if (!response.ok) {
    throw new Error(
      `图片读取失败（HTTP ${response.status}）。`,
    )
  }

  const blob =
    await response.blob()

  if (
    blob.size <= 0 ||
    !blob.type.startsWith(
      'image/',
    )
  ) {
    throw new Error(
      '图片文件内容无效。',
    )
  }

  return blobToDataURL(
    blob,
  )
}

function blobToDataURL(
  blob: Blob,
): Promise<string> {
  return new Promise(
    (
      resolve,
      reject,
    ) => {
      const reader =
        new FileReader()

      reader.onload = () => {
        if (
          typeof reader.result ===
            'string'
        ) {
          resolve(
            reader.result,
          )
          return
        }

        reject(
          new Error(
            '漫画图片转换失败。',
          ),
        )
      }

      reader.onerror = () =>
        reject(
          new Error(
            '漫画图片读取失败。',
          ),
        )

      reader.readAsDataURL(
        blob,
      )
    },
  )
}

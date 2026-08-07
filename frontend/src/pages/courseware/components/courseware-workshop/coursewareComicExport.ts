/**
 * coursewareComicExport.ts
 *
 * 知识点漫画最终导出入口：
 *   - JPG与PDF统一使用纯Canvas事实渲染，不再使用SVG foreignObject；
 *   - 底图、气泡、尾巴、文字、题目和知识卡一次性扁平化；
 *   - JPG直接下载完整长图；
 *   - PDF打印窗口只承载扁平化页面图片，不再重新排文字和气泡；
 *   - PDF准备失败时保留错误窗口并显示具体原因。
 */

import type {
  CoursewareComicPanel,
  CoursewareComicWorkflowProject,
} from '@/api/coursewares'

import {
  buildCoursewareComicJPGCanvas,
  buildCoursewareComicPDFPageCanvases,
} from './coursewareComicExportCanvas'

import {
  prepareCoursewareComicExportPanels,
} from './coursewareComicExportMarkup'

export async function exportCoursewareComicAsJPG(
  project: CoursewareComicWorkflowProject,
  panels: CoursewareComicPanel[],
): Promise<void> {
  const prepared =
    await prepareCoursewareComicExportPanels(
      project,
      panels,
    )

  const canvas =
    await buildCoursewareComicJPGCanvas(
      project,
      prepared,
    )

  const blob =
    await canvasToJPEGBlob(
      canvas,
    )

  downloadBlob(
    blob,
    buildCoursewareComicExportFilename(
      project.title,
      'jpg',
    ),
  )
}

export async function exportCoursewareComicAsPDF(
  project: CoursewareComicWorkflowProject,
  panels: CoursewareComicPanel[],
): Promise<void> {
  const printWindow =
    window.open(
      '',
      '_blank',
      'width=1180,height=860',
    )

  if (!printWindow) {
    throw new Error(
      '浏览器阻止了PDF排版窗口，请允许本站弹出窗口后重试。',
    )
  }

  writePDFPreparingDocument(
    printWindow,
  )

  try {
    const prepared =
      await prepareCoursewareComicExportPanels(
        project,
        panels,
      )

    const pageCanvases =
      await buildCoursewareComicPDFPageCanvases(
        project,
        prepared,
      )

    const pageDataURLs =
      pageCanvases.map(
        canvas =>
          canvas.toDataURL(
            'image/jpeg',
            0.95,
          ),
      )

    const filename =
      buildCoursewareComicExportFilename(
        project.title,
        'pdf',
      )

    printWindow.document.open()

    printWindow.document.write(
      buildPDFPrintDocument(
        filename,
        pageDataURLs,
      ),
    )

    printWindow.document.close()

    await waitForPrintWindowImages(
      printWindow,
    )

    printWindow.focus()
    printWindow.print()
  } catch (error) {
    writePDFErrorDocument(
      printWindow,
      error,
    )

    throw error
  }
}

function buildPDFPrintDocument(
  filename: string,
  pageDataURLs: string[],
): string {
  const pages =
    pageDataURLs
      .map(
        (
          dataURL,
          index,
        ) =>
          `<section class="pdf-page"><img src="${escapeAttribute(dataURL)}" ` +
          `alt="知识点漫画PDF第${index + 1}页"/></section>`,
      )
      .join('')

  return (
    '<!doctype html><html><head><meta charset="utf-8">' +
    `<title>${escapeHTML(filename)}</title>` +
    '<style>' +
    '@page{size:A4 landscape;margin:0;}' +
    'html,body{margin:0;padding:0;background:#CBD5E1;}' +
    'body{font-family:Arial,Microsoft YaHei,sans-serif;}' +
    '.print-toolbar{position:sticky;top:0;z-index:1000;display:flex;align-items:center;' +
    'justify-content:center;gap:12px;padding:12px;background:#0F172A;color:#fff;font-size:14px;}' +
    '.print-toolbar button{padding:8px 14px;border:0;border-radius:8px;background:#7C3AED;' +
    'color:#fff;font-weight:800;cursor:pointer;}' +
    '.pdf-page{width:297mm;height:210mm;margin:12px auto;background:#fff;' +
    'display:flex;align-items:center;justify-content:center;overflow:hidden;' +
    'box-shadow:0 10px 30px rgba(15,23,42,.18);page-break-after:always;break-after:page;}' +
    '.pdf-page:last-child{page-break-after:auto;break-after:auto;}' +
    '.pdf-page img{display:block;width:100%;height:100%;object-fit:contain;}' +
    '@media print{html,body{background:#fff}.print-toolbar{display:none!important}' +
    '.pdf-page{margin:0;box-shadow:none}}' +
    '</style></head><body>' +
    '<div class="print-toolbar"><span>版面已固定为图片，请选择“另存为PDF”。</span>' +
    '<button type="button" onclick="window.print()">打印 / 另存为PDF</button></div>' +
    pages +
    '</body></html>'
  )
}

function writePDFPreparingDocument(
  printWindow: Window,
): void {
  printWindow.document.open()

  printWindow.document.write(
    '<!doctype html><html><head><meta charset="utf-8"><title>正在准备PDF</title></head>' +
    '<body style="margin:0;background:#F8FAFC;font-family:Arial,Microsoft YaHei,sans-serif;' +
    'display:flex;min-height:100vh;align-items:center;justify-content:center;color:#334155">' +
    '<div style="padding:28px 34px;border-radius:14px;background:#FFFFFF;' +
    'box-shadow:0 12px 34px rgba(15,23,42,.12);font-size:16px;font-weight:700">' +
    '正在按编辑器真实排版生成PDF页面…</div></body></html>',
  )

  printWindow.document.close()
}

function writePDFErrorDocument(
  printWindow: Window,
  error: unknown,
): void {
  if (printWindow.closed) {
    return
  }

  const message =
    error instanceof Error &&
    error.message.trim()
      ? error.message
      : '漫画PDF排版失败。'

  printWindow.document.open()

  printWindow.document.write(
    '<!doctype html><html><head><meta charset="utf-8"><title>PDF排版失败</title></head>' +
    '<body style="margin:0;background:#FEF2F2;font-family:Arial,Microsoft YaHei,sans-serif;' +
    'display:flex;min-height:100vh;align-items:center;justify-content:center;color:#991B1B">' +
    '<div style="max-width:680px;padding:28px 34px;border:1px solid #FECACA;border-radius:14px;' +
    'background:#FFFFFF;box-shadow:0 12px 34px rgba(127,29,29,.12)">' +
    '<div style="font-size:20px;font-weight:900;margin-bottom:10px">PDF排版失败</div>' +
    `<div style="font-size:14px;line-height:1.7">${escapeHTML(message)}</div>` +
    '<button type="button" onclick="window.close()" style="margin-top:18px;padding:9px 16px;' +
    'border:0;border-radius:8px;background:#DC2626;color:#FFFFFF;font-weight:800;cursor:pointer">' +
    '关闭窗口</button></div></body></html>',
  )

  printWindow.document.close()
}

function waitForPrintWindowImages(
  printWindow: Window,
): Promise<void> {
  const images =
    Array.from(
      printWindow.document.images,
    )

  if (
    images.length === 0
  ) {
    return Promise.resolve()
  }

  return Promise.all(
    images.map(
      image =>
        waitForSingleImage(
          image,
        ),
    ),
  ).then(
    () => undefined,
  )
}

function waitForSingleImage(
  image: HTMLImageElement,
): Promise<void> {
  if (
    image.complete &&
    image.naturalWidth >
      0
  ) {
    return Promise.resolve()
  }

  if (
    image.complete &&
    image.naturalWidth <=
      0
  ) {
    return Promise.reject(
      new Error(
        'PDF页面图片加载失败。',
      ),
    )
  }

  return new Promise(
    (
      resolve,
      reject,
    ) => {
      let settled = false

      const finish = (
        error?: Error,
      ) => {
        if (settled) {
          return
        }

        settled = true

        window.clearTimeout(
          timeout,
        )

        if (error) {
          reject(
            error,
          )
          return
        }

        resolve()
      }

      const timeout =
        window.setTimeout(
          () =>
            finish(
              new Error(
                'PDF页面加载超时，请重新尝试。',
              ),
            ),
          15000,
        )

      image.addEventListener(
        'load',
        () =>
          finish(),
        {
          once: true,
        },
      )

      image.addEventListener(
        'error',
        () =>
          finish(
            new Error(
              'PDF页面图片加载失败。',
            ),
          ),
        {
          once: true,
        },
      )
    },
  )
}

function canvasToJPEGBlob(
  canvas: HTMLCanvasElement,
): Promise<Blob> {
  return new Promise(
    (
      resolve,
      reject,
    ) => {
      try {
        canvas.toBlob(
          blob => {
            if (blob) {
              resolve(
                blob,
              )
              return
            }

            reject(
              new Error(
                '浏览器未能生成JPG文件。',
              ),
            )
          },
          'image/jpeg',
          0.95,
        )
      } catch {
        reject(
          new Error(
            'JPG导出失败，请刷新页面后重试。',
          ),
        )
      }
    },
  )
}

function downloadBlob(
  blob: Blob,
  filename: string,
): void {
  const objectURL =
    URL.createObjectURL(
      blob,
    )

  const anchor =
    document.createElement(
      'a',
    )

  anchor.href =
    objectURL

  anchor.download =
    filename

  anchor.style.display =
    'none'

  document.body.appendChild(
    anchor,
  )

  anchor.click()
  anchor.remove()

  window.setTimeout(
    () =>
      URL.revokeObjectURL(
        objectURL,
      ),
    1000,
  )
}

function buildCoursewareComicExportFilename(
  title: string,
  extension: 'jpg' | 'pdf',
): string {
  const normalized =
    title
      .trim()
      .replace(
        /[\\/:*?"<>|]+/g,
        '-',
      )
      .replace(
        /\s+/g,
        ' ',
      )
      .slice(
        0,
        80,
      ) ||
    '知识点漫画'

  return `${normalized}.${extension}`
}

function escapeHTML(
  value: string,
): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function escapeAttribute(
  value: string,
): string {
  return escapeHTML(
    value,
  )
}

/**
 * lesson-plan-section-rewrite.ts — 教案目录段落AI修改API。
 *
 * 两阶段协议：
 *   1. section-rewrite：流式生成预览，不写数据库。
 *   2. section-rewrite/apply：老师确认后原子应用。
 *
 * 应用接口会由后端再次校验作者、教案状态、版本和段落哈希。
 */

import apiClient from './client'

export interface LessonPlanSectionLocator {
  heading_text: string
  occurrence: number
}

export interface LessonPlanServerSection {
  id: string
  title: string
  heading_text: string
  level: number
  heading_path: string[]
  occurrence: number
  body_markdown: string
  section_hash: string
  locator: LessonPlanSectionLocator
}

export interface GenerateLessonPlanSectionRewriteRequest {
  base_version: number
  locator: LessonPlanSectionLocator
  instruction: string
}

export interface LessonPlanSectionRewritePreview {
  base_version: number
  section: LessonPlanServerSection
  replacement_markdown: string
}

export interface ApplyLessonPlanSectionRewriteRequest {
  base_version: number
  locator: LessonPlanSectionLocator
  section_hash: string
  replacement_markdown: string
}

export interface LessonPlanSectionRewriteApplyResponse {
  changed: boolean
  current_version: number
  content_markdown: string
}

export interface LessonPlanSectionRewriteHandlers {
  onConnected?: () => void
  onChunk?: (chunk: string) => void
  onDone?: (preview: LessonPlanSectionRewritePreview) => void
  onError?: (message: string, code?: string) => void
}

export interface LessonPlanSectionRewriteConnection {
  close: () => void
}

interface SectionRewriteSSEError {
  code?: string
  message?: string
}

interface SectionRewriteSSEDone {
  preview?: LessonPlanSectionRewritePreview
}

/**
 * 流式生成教案段落修改预览。
 *
 * 该连接是一次性的，不执行自动重连。
 * 中断后由老师点击“重新生成”，避免同一修改要求重复扣费。
 */
export function generateLessonPlanSectionRewrite(
  planID: string,
  request: GenerateLessonPlanSectionRewriteRequest,
  handlers: LessonPlanSectionRewriteHandlers,
): LessonPlanSectionRewriteConnection {
  const controller = new AbortController()
  const token = localStorage.getItem('token') || ''

  let closed = false
  let terminalEventReceived = false

  const emitError = (message: string, code?: string) => {
    if (closed) return
    terminalEventReceived = true
    handlers.onError?.(message, code)
  }

  void fetch(
    `/api/v1/lesson-plans/plans/${encodeURIComponent(planID)}/section-rewrite`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(request),
      signal: controller.signal,
    },
  )
    .then(async response => {
      if (!response.ok) {
        const payload = await response
          .json()
          .catch(() => null) as { message?: string } | null

        emitError(
          payload?.message || `请求失败：HTTP ${response.status}`,
          `http_${response.status}`,
        )
        return
      }

      if (!response.body) {
        emitError('当前浏览器不支持流式响应', 'stream_unsupported')
        return
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      try {
        while (!closed) {
          const { done, value } = await reader.read()
          if (done) break

          buffer += decoder.decode(value, { stream: true })
          buffer = buffer.replace(/\r\n/g, '\n')

          const blocks = buffer.split('\n\n')
          buffer = blocks.pop() || ''

          for (const block of blocks) {
            if (!block.trim()) continue

            let eventType = ''
            const dataLines: string[] = []

            for (const line of block.split('\n')) {
              if (line.startsWith('event:')) {
                eventType = line.slice(6).trim()
              } else if (line.startsWith('data:')) {
                dataLines.push(line.slice(5).trimStart())
              }
            }

            if (!eventType || dataLines.length === 0) continue

            let payload: unknown
            try {
              payload = JSON.parse(dataLines.join('\n'))
            } catch {
              continue
            }

            if (eventType === 'connected') {
              handlers.onConnected?.()
              continue
            }

            if (eventType === 'chunk') {
              const data = payload as { chunk?: string }
              if (data.chunk) handlers.onChunk?.(data.chunk)
              continue
            }

            if (eventType === 'error') {
              const data = payload as SectionRewriteSSEError
              emitError(
                data.message || 'AI修改建议生成失败',
                data.code,
              )
              closed = true
              break
            }

            if (eventType === 'done') {
              const data = payload as SectionRewriteSSEDone
              if (!data.preview) {
                emitError('AI返回的修改预览不完整', 'invalid_preview')
              } else {
                terminalEventReceived = true
                handlers.onDone?.(data.preview)
              }
              closed = true
              break
            }
          }
        }
      } finally {
        await reader.cancel().catch(() => undefined)
      }

      if (!closed && !terminalEventReceived) {
        emitError('流式连接提前结束，请重新生成', 'stream_interrupted')
      }
    })
    .catch(error => {
      if (closed || error?.name === 'AbortError') return

      const message = error instanceof Error
        ? error.message
        : '网络连接失败'

      emitError(message, 'network_error')
    })

  return {
    close: () => {
      if (closed) return
      closed = true
      controller.abort()
    },
  }
}

/** 原子应用老师确认的段落修改结果。 */
export async function applyLessonPlanSectionRewrite(
  planID: string,
  request: ApplyLessonPlanSectionRewriteRequest,
): Promise<LessonPlanSectionRewriteApplyResponse> {
  const response = await apiClient.post(
    `/lesson-plans/plans/${planID}/section-rewrite/apply`,
    request,
  )

  return response.data.data as LessonPlanSectionRewriteApplyResponse
}

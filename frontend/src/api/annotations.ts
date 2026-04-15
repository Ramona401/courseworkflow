/**
 * annotations.ts — 教案段落批注 API 封装
 *
 * 对应后端 /api/v1/lesson-plans/plans/{id}/annotations 系列接口
 * v102新增：aiFixAnnotation — 批注驱动的AI辅助修改（SSE流式）
 * v104新增：Annotation.review_round 字段；AnnotationListResponse.current_round 字段
 */
import apiClient from './client'

/* ==================== 类型定义 ==================== */

export interface Annotation {
  id: string
  lesson_plan_id: string
  reviewer_id: string
  reviewer_name: string
  paragraph_index: number      // 段落序号（从0开始）
  paragraph_preview: string    // 段落前50字预览
  content: string              // 批注内容
  status: 'pending' | 'resolved' | 'archived'  // v104新增archived状态
  review_round: number         // v104新增：评审轮次（从1开始）
  created_at: string
  updated_at: string
}

export interface AnnotationListResponse {
  annotations: Annotation[]
  total: number
  current_round: number        // v104新增：当前最新评审轮次，前端按此分组显示
}

export interface CreateAnnotationRequest {
  paragraph_index: number
  paragraph_preview: string
  content: string
  review_round?: number        // 可选，不传则后端自动推断
}

/* ==================== SSE连接类型（AI辅助修改）==================== */

/** AI辅助修改SSE连接句柄 */
export interface AIFixSSEConnection {
  /** 手动关闭连接 */
  close: () => void
}

/** AI辅助修改事件回调 */
export interface AIFixHandlers {
  /** 连接建立 */
  onConnected?: () => void
  /** 收到AI输出片段（流式） */
  onChunk?: (chunk: string) => void
  /** AI完成输出（携带完整内容） */
  onDone?: (fullContent: string, tokensUsed: number) => void
  /** 出错 */
  onError?: (error: string) => void
}

/* ==================== API 函数 ==================== */

/** 获取教案全部批注 */
export async function getAnnotations(planId: string): Promise<AnnotationListResponse> {
  const resp = await apiClient.get(`/lesson-plans/plans/${planId}/annotations`)
  return resp.data.data as AnnotationListResponse
}

/** 创建段落批注（评审员） */
export async function createAnnotation(
  planId: string,
  data: CreateAnnotationRequest
): Promise<Annotation> {
  const resp = await apiClient.post(`/lesson-plans/plans/${planId}/annotations`, data)
  return resp.data.data as Annotation
}

/** 修改批注内容（评审员本人） */
export async function updateAnnotation(
  planId: string,
  annotationId: string,
  content: string
): Promise<void> {
  await apiClient.put(`/lesson-plans/plans/${planId}/annotations/${annotationId}`, { content })
}

/** 删除批注（评审员本人或admin） */
export async function deleteAnnotation(
  planId: string,
  annotationId: string
): Promise<void> {
  await apiClient.delete(`/lesson-plans/plans/${planId}/annotations/${annotationId}`)
}

/** 标记批注处理状态（作者） */
export async function resolveAnnotation(
  planId: string,
  annotationId: string,
  status: 'pending' | 'resolved'
): Promise<void> {
  await apiClient.put(
    `/lesson-plans/plans/${planId}/annotations/${annotationId}/resolve`,
    { status }
  )
}

/**
 * AI辅助修改——SSE流式接口
 *
 * 将原段落内容 + 批注意见发给AI，流式返回修改建议。
 * 前端展示建议后可"采用"（替换段落内容）或"忽略"。
 *
 * @param planId           教案ID
 * @param annotationId     批注ID
 * @param paragraphContent 原段落文字
 * @param annotationContent 批注意见
 * @param handlers         事件回调
 * @returns SSE连接句柄（可调用close手动关闭）
 */
export function aiFixAnnotation(
  planId: string,
  annotationId: string,
  paragraphContent: string,
  annotationContent: string,
  planContext: string,        // v104新增：教案全貌上下文（学科/年级/课题+正文）
  handlers: AIFixHandlers
): AIFixSSEConnection {
  let isClosed = false
  const controller = new AbortController()

  // 从localStorage读取token（与axios拦截器保持一致）
  const token = localStorage.getItem('token') || ''

  fetch(`/api/v1/lesson-plans/plans/${planId}/annotations/${annotationId}/ai-fix`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({
      paragraph_content: paragraphContent,
      annotation_content: annotationContent,
      plan_context: planContext,    // v104：教案全貌，供AI理解整体后精准修改段落
    }),
    signal: controller.signal,
  }).then(async (response) => {
    if (!response.ok) {
      handlers.onError?.(`请求失败: HTTP ${response.status}`)
      return
    }
    if (!response.body) {
      handlers.onError?.('不支持流式响应')
      return
    }

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (!isClosed) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      // 按SSE格式解析：每条事件以 \n\n 分隔
      const parts = buffer.split('\n\n')
      buffer = parts.pop() ?? ''

      for (const part of parts) {
        const lines = part.trim().split('\n')
        let eventType = ''
        let dataStr = ''

        for (const line of lines) {
          if (line.startsWith('event: ')) {
            eventType = line.slice(7).trim()
          } else if (line.startsWith('data: ')) {
            dataStr = line.slice(6).trim()
          }
        }

        if (!eventType || !dataStr) continue

        try {
          const data = JSON.parse(dataStr)
          switch (eventType) {
            case 'connected':
              handlers.onConnected?.()
              break
            case 'chunk':
              if (data.chunk) handlers.onChunk?.(data.chunk)
              break
            case 'done':
              handlers.onDone?.(data.full_content ?? '', data.tokens_used ?? 0)
              isClosed = true
              break
            case 'error':
              handlers.onError?.(data.error ?? 'AI修改失败')
              isClosed = true
              break
          }
        } catch {
          // 忽略JSON解析错误
        }
      }
    }

    reader.cancel()
  }).catch((err) => {
    if (isClosed) return  // 主动关闭不算错误
    handlers.onError?.(`连接失败: ${err.message ?? '未知错误'}`)
  })

  return {
    close: () => {
      isClosed = true
      controller.abort()
    },
  }
}

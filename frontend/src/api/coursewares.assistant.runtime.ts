/**
 * 课件教学智能体 —— 公开运行API
 *
 * 安全边界：
 *   - 不使用教师端Axios或教师JWT；
 *   - 只传递短时runtime_token；
 *   - 不伪造浏览器Origin；
 *   - 会话创建显式携带从document.referrer解析的真实父页面Origin；
 *   - 会话创建只向同源API发送当前官方embed页面Referer；
 *   - 支持close()和外部AbortSignal取消；
 *   - JSON与SSE载荷通过结构守卫后才交给UI。
 */

import type {
  AssistantRuntimeChatConnection,
  AssistantRuntimeChatHandlers,
  AssistantRuntimeRequestOptions,
  AssistantRuntimeSessionView,
  AssistantRuntimeStartResponse,
} from './coursewares.assistant.types'
import {
  isAssistantRuntimeChatResponse,
  isAssistantRuntimeChunkEvent,
  isAssistantRuntimeConnectedEvent,
  isAssistantRuntimeErrorEvent,
  isAssistantRuntimeSessionView,
  isAssistantRuntimeStartResponse,
} from './coursewares.assistant.types'

// ==================== 公开错误 ====================

export class AssistantRuntimeAPIError extends Error {
  readonly status: number
  readonly code: number

  constructor(message: string, status: number, code = -1) {
    super(message)
    this.name = 'AssistantRuntimeAPIError'
    this.status = status
    this.code = code
  }
}

type UnknownRecord = Record<string, unknown>

interface PublicAPIEnvelope {
  code: number
  message: string
  data?: unknown
}

interface ParsedSSEEvent {
  event: string
  data: string
}

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === 'object' && value !== null
}

function isPublicAPIEnvelope(
  value: unknown,
): value is PublicAPIEnvelope {
  return isRecord(value)
    && typeof value.code === 'number'
    && typeof value.message === 'string'
}

function normalizeBaseURL(baseURL?: string): string {
  return (baseURL || '').trim().replace(/\/+$/, '')
}

function buildPublicURL(
  path: string,
  options?: AssistantRuntimeRequestOptions,
): string {
  return `${normalizeBaseURL(options?.baseURL)}${path}`
}

function pathSegment(value: string): string {
  const normalized = value.trim()
  if (!normalized) {
    throw new Error('教学智能体资源ID不能为空')
  }
  return encodeURIComponent(normalized)
}

function runtimeTokenHeader(runtimeToken: string): string {
  const normalized = runtimeToken.trim()
  if (!normalized) {
    throw new Error('教学智能体运行令牌不能为空')
  }
  return `Bearer ${normalized}`
}

/**
 * 规范化学生端从document.referrer取得的真实父页面Origin。
 *
 * 这里只做客户端快速失败；后端仍会重新规范化并与部署allowed_origins比较。
 */
function normalizeParentOrigin(parentOrigin: string): string {
  const normalized = parentOrigin.trim()
  if (!normalized) {
    throw new Error('无法识别授权课件来源')
  }

  try {
    const parsed = new URL(normalized)

    if (
      (parsed.protocol !== 'https:' && parsed.protocol !== 'http:')
      || parsed.username
      || parsed.password
      || parsed.pathname !== '/'
      || parsed.search
      || parsed.hash
    ) {
      throw new Error('父页面来源格式无效')
    }

    return parsed.origin
  } catch {
    throw new Error('无法识别授权课件来源')
  }
}

function errorMessage(error: unknown): string {
  return error instanceof Error
    ? error.message
    : '教学智能体连接失败，请重试'
}

// ==================== 严格JSON响应 ====================

async function parsePublicEnvelope(
  response: Response,
): Promise<PublicAPIEnvelope> {
  const raw = await response.text()
  let parsed: unknown

  try {
    parsed = raw ? JSON.parse(raw) : null
  } catch {
    throw new AssistantRuntimeAPIError(
      '教学智能体服务返回了无效响应',
      response.status,
    )
  }

  if (!isPublicAPIEnvelope(parsed)) {
    throw new AssistantRuntimeAPIError(
      '教学智能体服务返回了无效响应',
      response.status,
    )
  }

  return parsed
}

async function readPublicData<T>(
  response: Response,
  guard: (value: unknown) => value is T,
): Promise<T> {
  const envelope = await parsePublicEnvelope(response)

  if (!response.ok || envelope.code !== 0) {
    throw new AssistantRuntimeAPIError(
      envelope.message || '教学智能体请求失败',
      response.status,
      envelope.code,
    )
  }

  if (!guard(envelope.data)) {
    throw new AssistantRuntimeAPIError(
      '教学智能体服务返回的数据格式无效',
      response.status,
      envelope.code,
    )
  }

  return envelope.data
}

async function readPublicError(
  response: Response,
  fallback: string,
): Promise<AssistantRuntimeAPIError> {
  try {
    const envelope = await parsePublicEnvelope(response)
    return new AssistantRuntimeAPIError(
      envelope.message || fallback,
      response.status,
      envelope.code,
    )
  } catch (error) {
    if (error instanceof AssistantRuntimeAPIError) {
      return error
    }
    return new AssistantRuntimeAPIError(
      fallback,
      response.status,
    )
  }
}

// ==================== 随机匿名客户端标识 ====================

/**
 * 生成浏览器随机匿名标识。
 * 不保存姓名、账号、手机号或设备指纹。
 */
export function createAssistantRuntimeAnonymousClientID(): string {
  if (
    typeof crypto === 'undefined'
    || typeof crypto.getRandomValues !== 'function'
  ) {
    throw new Error('当前浏览器不支持安全随机数')
  }

  if (typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }

  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)

  // RFC 4122 version 4 + variant。
  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80

  const hex = Array.from(
    bytes,
    value => value.toString(16).padStart(2, '0'),
  ).join('')

  return [
    hex.slice(0, 8),
    hex.slice(8, 12),
    hex.slice(12, 16),
    hex.slice(16, 20),
    hex.slice(20),
  ].join('-')
}

// ==================== 会话创建与读取 ====================

export async function startAssistantRuntimeSession(
  publicId: string,
  anonymousClientId: string,
  parentOrigin: string,
  options: AssistantRuntimeRequestOptions = {},
): Promise<AssistantRuntimeStartResponse> {
  const normalizedClientId = anonymousClientId.trim()
  const normalizedParentOrigin = normalizeParentOrigin(parentOrigin)

  if (!normalizedClientId) {
    throw new Error('匿名客户端标识不能为空')
  }

  const response = await fetch(
    buildPublicURL(
      `/api/v1/assistant-runtime/deployments/${pathSegment(publicId)}/session`,
      options,
    ),
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        anonymous_client_id: normalizedClientId,
        parent_origin: normalizedParentOrigin,
      }),
      credentials: 'omit',
      cache: 'no-store',

      // 只向同源运行API发送当前官方embed页面完整Referer。
      // 页面响应同时设置Referrer-Policy: same-origin。
      referrerPolicy: 'same-origin',
      signal: options.signal,
    },
  )

  return readPublicData(
    response,
    isAssistantRuntimeStartResponse,
  )
}

export async function getAssistantRuntimeSession(
  sessionId: string,
  runtimeToken: string,
  options: AssistantRuntimeRequestOptions = {},
): Promise<AssistantRuntimeSessionView> {
  const response = await fetch(
    buildPublicURL(
      `/api/v1/assistant-runtime/sessions/${pathSegment(sessionId)}`,
      options,
    ),
    {
      method: 'GET',
      headers: {
        Authorization: runtimeTokenHeader(runtimeToken),
      },
      credentials: 'omit',
      cache: 'no-store',
      referrerPolicy: 'no-referrer',
      signal: options.signal,
    },
  )

  return readPublicData(
    response,
    isAssistantRuntimeSessionView,
  )
}

// ==================== SSE解析 ====================

function normalizeSSELineEndings(value: string): string {
  return value.replace(/\r\n/g, '\n')
}

function parseSSEBlock(
  block: string,
): ParsedSSEEvent | null {
  const lines = normalizeSSELineEndings(block).split('\n')
  let event = ''
  const dataLines: string[] = []

  for (const line of lines) {
    if (line.startsWith(':')) {
      continue
    }

    if (line.startsWith('event:')) {
      event = line.slice('event:'.length).trim()
      continue
    }

    if (line.startsWith('data:')) {
      dataLines.push(
        line.slice('data:'.length).trimStart(),
      )
    }
  }

  if (!event || dataLines.length === 0) {
    return null
  }

  return {
    event,
    data: dataLines.join('\n'),
  }
}

function parseEventJSON(data: string): unknown {
  try {
    return JSON.parse(data)
  } catch {
    throw new Error(
      '教学智能体流式响应格式错误，请重试',
    )
  }
}

function linkExternalAbortSignal(
  external: AbortSignal | undefined,
  controller: AbortController,
  markCancelled: () => void,
): () => void {
  if (!external) {
    return () => undefined
  }

  const abort = () => {
    markCancelled()
    controller.abort()
  }

  if (external.aborted) {
    abort()
    return () => undefined
  }

  external.addEventListener(
    'abort',
    abort,
    { once: true },
  )

  return () => {
    external.removeEventListener(
      'abort',
      abort,
    )
  }
}

// ==================== 流式聊天 ====================

export function streamAssistantRuntimeChat(
  sessionId: string,
  runtimeToken: string,
  message: string,
  handlers: AssistantRuntimeChatHandlers,
  options: AssistantRuntimeRequestOptions = {},
): AssistantRuntimeChatConnection {
  const normalizedMessage = message.trim()

  if (!normalizedMessage) {
    throw new Error('学生消息不能为空')
  }

  const controller = new AbortController()
  let cancelled = false
  let terminal = false
  let errorEmitted = false

  const emitError = (messageText: string) => {
    if (errorEmitted || cancelled) {
      return
    }

    errorEmitted = true
    handlers.onError?.(
      messageText || '教学智能体回复失败，请重试',
    )
  }

  const unlinkExternalSignal =
    linkExternalAbortSignal(
      options.signal,
      controller,
      () => {
        cancelled = true
      },
    )

  const finished = (async () => {
    let reader:
      | ReadableStreamDefaultReader<Uint8Array>
      | null = null

    try {
      const response = await fetch(
        buildPublicURL(
          `/api/v1/assistant-runtime/sessions/${pathSegment(sessionId)}/chat`,
          options,
        ),
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization:
              runtimeTokenHeader(runtimeToken),
          },
          body: JSON.stringify({
            message: normalizedMessage,
          }),
          credentials: 'omit',
          cache: 'no-store',
          referrerPolicy: 'no-referrer',
          signal: controller.signal,
        },
      )

      if (!response.ok) {
        throw await readPublicError(
          response,
          `教学智能体请求失败：HTTP ${response.status}`,
        )
      }

      const contentType =
        response.headers.get('Content-Type') || ''

      if (!contentType
        .toLowerCase()
        .startsWith('text/event-stream')) {
        throw await readPublicError(
          response,
          '服务器未返回教学智能体流式响应',
        )
      }

      if (!response.body) {
        throw new Error('当前浏览器不支持流式响应')
      }

      reader = response.body.getReader()

      const decoder = new TextDecoder()
      let buffer = ''

      const handleBlock = (block: string) => {
        const event = parseSSEBlock(block)

        if (!event || terminal || cancelled) {
          return
        }

        const data = parseEventJSON(event.data)

        switch (event.event) {
        case 'connected':
          if (!isAssistantRuntimeConnectedEvent(data)) {
            throw new Error(
              '教学智能体连接事件格式错误，请重试',
            )
          }
          handlers.onConnected?.(data)
          break

        case 'chunk':
          if (!isAssistantRuntimeChunkEvent(data)) {
            throw new Error(
              '教学智能体增量事件格式错误，请重试',
            )
          }
          handlers.onChunk?.(data.chunk)
          break

        case 'done':
          if (!isAssistantRuntimeChatResponse(data)) {
            throw new Error(
              '教学智能体完成事件格式错误，请重试',
            )
          }
          terminal = true
          handlers.onDone?.(data)
          break

        case 'error':
          if (!isAssistantRuntimeErrorEvent(data)) {
            throw new Error(
              '教学智能体错误事件格式错误，请重试',
            )
          }
          terminal = true
          emitError(data.error)
          break

        default:
          // 前向兼容未知事件，不把未定义载荷交给业务层。
          break
        }
      }

      while (!cancelled && !terminal) {
        const result = await reader.read()

        if (result.done) {
          break
        }

        buffer += decoder.decode(
          result.value,
          { stream: true },
        )
        buffer = normalizeSSELineEndings(buffer)

        const blocks = buffer.split('\n\n')
        buffer = blocks.pop() || ''

        for (const block of blocks) {
          if (!block.trim()) {
            continue
          }

          handleBlock(block)

          if (terminal || cancelled) {
            break
          }
        }
      }

      buffer += decoder.decode()
      buffer = normalizeSSELineEndings(buffer)

      if (
        !cancelled
        && !terminal
        && buffer.trim()
      ) {
        handleBlock(buffer)
      }

      if (!cancelled && !terminal) {
        emitError(
          '教学智能体流式响应意外中断，请重新尝试',
        )
      }
    } catch (error) {
      if (!cancelled) {
        emitError(errorMessage(error))
      }
    } finally {
      unlinkExternalSignal()

      if (reader) {
        try {
          await reader.cancel()
        } catch {
          // 连接可能已经由服务端正常关闭。
        }
      }
    }
  })()

  return {
    finished,
    close: () => {
      if (cancelled) {
        return
      }

      cancelled = true
      controller.abort()
    },
  }
}

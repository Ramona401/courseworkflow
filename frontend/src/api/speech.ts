/**
 * speech.ts — TE-DNA统一语音输入WebSocket客户端
 *
 * 浏览器通信链：
 *   麦克风
 *     → 本站 /api/v1/speech/stream
 *     → TE-DNA Go后端
 *     → 豆包流式语音识别2.0
 *
 * 安全边界：
 * 1. 火山APP ID和Access Token永远不进入浏览器；
 * 2. WebSocket不能设置Authorization请求头，因此沿用现有SSE的query token；
 * 3. 音频只发送二进制帧，不写入sessionStorage；
 * 4. 识别文字只交给业务输入框，不自动调用AI。
 *
 * 生命周期约束：
 * 1. onClosed最多通知一次；
 * 2. 服务端error/final/closed均视为终态事件；
 * 3. 调用close()后不再把本地主动关闭误报为网络中断；
 * 4. 浏览器WebSocket握手失败时，只暴露稳定用户文案。
 */

export type SpeechEventType =
  | 'ready'
  | 'partial'
  | 'final'
  | 'error'
  | 'closed'

export interface SpeechUtterance {
  text: string
  start_time: number
  end_time: number
  definite: boolean
}

export interface SpeechRecognitionEvent {
  event: SpeechEventType
  text?: string
  final?: boolean
  utterances?: SpeechUtterance[]
  duration_ms?: number
  request_id?: string
  log_id?: string
  code?: string
  message?: string
}

export interface SpeechStreamHandlers {
  onReady?: (
    event: SpeechRecognitionEvent,
  ) => void
  onPartial?: (
    event: SpeechRecognitionEvent,
  ) => void
  onFinal?: (
    event: SpeechRecognitionEvent,
  ) => void
  onError?: (
    event: SpeechRecognitionEvent,
  ) => void
  onClosed?: (
    event?: SpeechRecognitionEvent,
  ) => void
  onUnexpectedClose?: (
    message: string,
  ) => void
}

export interface SpeechStreamConnection {
  isReady: () => boolean
  sendAudio: (
    pcm: ArrayBuffer,
  ) => void
  stop: () => void
  cancel: () => void
  close: () => void
}

function buildSpeechWebSocketURL(
  token: string,
): string {
  const protocol =
    window.location.protocol === 'https:'
      ? 'wss:'
      : 'ws:'

  const query =
    new URLSearchParams({
      token,
    })

  return (
    `${protocol}//${window.location.host}` +
    `/api/v1/speech/stream?${query.toString()}`
  )
}

function parseSpeechEvent(
  raw: unknown,
): SpeechRecognitionEvent | null {
  if (
    !raw ||
    typeof raw !== 'object'
  ) {
    return null
  }

  const candidate =
    raw as Partial<SpeechRecognitionEvent>

  const allowed:
    SpeechEventType[] = [
      'ready',
      'partial',
      'final',
      'error',
      'closed',
    ]

  if (
    typeof candidate.event !== 'string' ||
    !allowed.includes(
      candidate.event as SpeechEventType,
    )
  ) {
    return null
  }

  return candidate as SpeechRecognitionEvent
}

function sendControl(
  socket: WebSocket,
  action:
    | 'start'
    | 'stop'
    | 'cancel',
): void {
  if (
    socket.readyState !==
    WebSocket.OPEN
  ) {
    throw new Error(
      '语音连接尚未建立',
    )
  }

  const payload =
    action === 'start'
      ? {
          action,
          sample_rate: 16000,
          bits_per_sample: 16,
          channels: 1,
        }
      : {
          action,
        }

  socket.send(
    JSON.stringify(payload),
  )
}

/**
 * 建立本站语音连接。
 *
 * WebSocket打开后自动发送start控制消息；
 * 调用方必须等onReady后才能发送PCM。
 */
export function createSpeechStream(
  token: string,
  handlers: SpeechStreamHandlers,
): SpeechStreamConnection {
  const normalizedToken =
    token.trim()

  if (!normalizedToken) {
    throw new Error(
      '登录状态已失效，请重新登录',
    )
  }

  const socket =
    new WebSocket(
      buildSpeechWebSocketURL(
        normalizedToken,
      ),
    )

  let ready = false
  let closedByClient = false
  let terminalReceived = false
  let closedDelivered = false

  socket.binaryType = 'arraybuffer'

  const deliverClosed = (
    event?: SpeechRecognitionEvent,
  ) => {
    if (closedDelivered) return

    closedDelivered = true
    ready = false
    handlers.onClosed?.(event)
  }

  socket.onopen = () => {
    try {
      sendControl(
        socket,
        'start',
      )
    } catch {
      closedByClient = true

      try {
        socket.close()
      } catch {
        deliverClosed()
      }
    }
  }

  socket.onmessage = (
    message: MessageEvent,
  ) => {
    if (
      typeof message.data !==
      'string'
    ) {
      return
    }

    let raw: unknown

    try {
      raw = JSON.parse(
        message.data,
      )
    } catch {
      return
    }

    const event =
      parseSpeechEvent(raw)

    if (!event) return

    switch (event.event) {
      case 'ready':
        ready = true
        handlers.onReady?.(event)
        break

      case 'partial':
        handlers.onPartial?.(event)
        break

      case 'final':
        terminalReceived = true
        handlers.onFinal?.(event)
        break

      case 'error':
        terminalReceived = true
        handlers.onError?.(event)
        break

      case 'closed':
        terminalReceived = true
        deliverClosed(event)
        break
    }
  }

  socket.onerror = () => {
    /**
     * WebSocket API不会暴露握手HTTP响应正文。
     * 详细状态由close统一映射为稳定文案。
     */
  }

  socket.onclose = () => {
    ready = false

    if (
      !closedByClient &&
      !terminalReceived
    ) {
      handlers.onUnexpectedClose?.(
        '语音连接意外中断，请重新尝试',
      )
    }

    deliverClosed()
  }

  return {
    isReady: () =>
      ready &&
      socket.readyState ===
        WebSocket.OPEN,

    sendAudio: (
      pcm: ArrayBuffer,
    ) => {
      if (
        !ready ||
        socket.readyState !==
          WebSocket.OPEN
      ) {
        throw new Error(
          '语音识别尚未就绪',
        )
      }

      if (
        pcm.byteLength > 0
      ) {
        socket.send(pcm)
      }
    },

    stop: () => {
      sendControl(
        socket,
        'stop',
      )
    },

    cancel: () => {
      if (
        socket.readyState ===
        WebSocket.OPEN
      ) {
        try {
          sendControl(
            socket,
            'cancel',
          )
        } catch {
          // 连接正在关闭时无需重复报错。
        }
      }
    },

    close: () => {
      closedByClient = true
      ready = false

      if (
        socket.readyState ===
          WebSocket.OPEN ||
        socket.readyState ===
          WebSocket.CONNECTING
      ) {
        try {
          socket.close()
        } catch {
          deliverClosed()
        }
      } else {
        deliverClosed()
      }
    },
  }
}

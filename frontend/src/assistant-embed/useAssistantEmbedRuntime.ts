/**
 * 公开学生端教学智能体运行状态Hook。
 *
 * 本文件负责：
 *   - 学生主动开始后的匿名短时会话创建；
 *   - 会话状态同步与401自动恢复；
 *   - 流式聊天连接、消息状态和会话轮数；
 *   - 页面卸载时关闭请求、SSE连接并清除内存令牌。
 *
 * 安全边界：
 *   - runtime_token只保存在React内存中；
 *   - 匿名客户端标识每次建立会话时重新生成；
 *   - 不使用教师JWT、教师Axios客户端或浏览器持久存储；
 *   - 401恢复不会自动重发学生消息，避免重复调用和重复计费；
 *   - generation变化后，旧请求回调不能污染新会话。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'

import {
  AssistantRuntimeAPIError,
  createAssistantRuntimeAnonymousClientID,
  getAssistantRuntimeSession,
  startAssistantRuntimeSession,
  streamAssistantRuntimeChat,
} from '../api/coursewares.assistant.runtime'

import type {
  AssistantRuntimeChatConnection,
  AssistantRuntimeMessage,
  AssistantRuntimeSessionView,
} from '../api/coursewares.assistant.types'

import {
  assistantEmbedFallbackSession,
  assistantEmbedFormatExpiry,
  assistantEmbedVisibleMessages,
  assistantEmbedWelcomeMessages,
  type AssistantEmbedBootstrap,
  type AssistantEmbedNotice,
  type AssistantEmbedStartupState,
} from './assistantEmbedSupport'

interface UseAssistantEmbedRuntimeOptions {
  bootstrap: AssistantEmbedBootstrap
  framed: boolean
  parentOrigin: string
}

export function useAssistantEmbedRuntime({
  bootstrap,
  framed,
  parentOrigin,
}: UseAssistantEmbedRuntimeOptions) {
  const generationRef = useRef(0)
  const connectionRef = useRef<AssistantRuntimeChatConnection | null>(null)
  const startupAbortRef = useRef<AbortController | null>(null)
  const tokenRecoveryRef = useRef(false)
  const restartSessionRef = useRef<() => void>(() => undefined)

  const [hasStarted, setHasStarted] = useState(false)
  const [startupState, setStartupState] =
    useState<AssistantEmbedStartupState>(framed ? 'loading' : 'blocked')
  const [session, setSession] = useState<AssistantRuntimeSessionView | null>(null)
  const [runtimeToken, setRuntimeToken] = useState('')
  const [messages, setMessages] = useState<AssistantRuntimeMessage[]>([])
  const [streamingText, setStreamingText] = useState('')
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)
  const [notice, setNotice] = useState<AssistantEmbedNotice | null>(null)

  const closeConnection = useCallback(() => {
    connectionRef.current?.close()
    connectionRef.current = null
  }, [])

  /**
   * 从服务端重新读取当前正式会话状态。
   *
   * 返回true表示同步成功；返回false表示请求已经过期、取消或失败。
   */
  const synchronizeSession = useCallback(
    async (
      sessionId: string,
      token: string,
      generation: number,
      signal?: AbortSignal,
    ): Promise<boolean> => {
      try {
        const view = await getAssistantRuntimeSession(
          sessionId,
          token,
          { signal },
        )

        if (generationRef.current !== generation) {
          return false
        }

        setSession(view)
        setMessages(
          assistantEmbedVisibleMessages(
            bootstrap.welcomeMessage,
            view.messages,
          ),
        )

        return true
      } catch (cause) {
        if (
          generationRef.current !== generation ||
          signal?.aborted
        ) {
          return false
        }

        if (
          cause instanceof AssistantRuntimeAPIError &&
          cause.status === 401
        ) {
          if (!tokenRecoveryRef.current) {
            tokenRecoveryRef.current = true
            setNotice({
              kind: 'error',
              text: '本次学习会话已过期，正在自动重新建立。上一条消息不会自动重发。',
            })
            restartSessionRef.current()
          } else {
            setNotice({
              kind: 'error',
              text: '学习会话自动恢复失败，请点击“重新开始”后再次发送。',
            })
          }

          return false
        }

        setNotice({
          kind: 'error',
          text: cause instanceof Error
            ? cause.message
            : '同步学习会话失败',
        })

        return false
      }
    },
    [bootstrap.welcomeMessage],
  )

  /**
   * 建立全新的学生短时会话。
   *
   * recoveringFromTokenExpiry=true时表示由401自动恢复触发。
   * 旧消息不会自动重发，也不会复用旧令牌或旧匿名客户端标识。
   */
  const startSession = useCallback(
    async (
      recoveringFromTokenExpiry = false,
    ): Promise<void> => {
      setHasStarted(true)

      if (!framed) {
        tokenRecoveryRef.current = false
        setStartupState('blocked')
        return
      }

      if (!parentOrigin) {
        tokenRecoveryRef.current = false
        setStartupState('error')
        setNotice({
          kind: 'error',
          text: '无法识别授权课件来源。请从老师提供的课件或授课平台重新打开。',
        })
        return
      }

      if (!recoveringFromTokenExpiry) {
        tokenRecoveryRef.current = false
      }

      const generation = generationRef.current + 1
      generationRef.current = generation

      closeConnection()
      startupAbortRef.current?.abort()

      const controller = new AbortController()
      startupAbortRef.current = controller

      setStartupState('loading')
      setSession(null)
      setRuntimeToken('')
      setMessages([])
      setStreamingText('')
      setInput('')
      setSending(false)

      setNotice(
        recoveringFromTokenExpiry
          ? {
              kind: 'error',
              text: '本次学习会话已过期，正在自动重新建立。上一条消息不会自动重发。',
            }
          : null,
      )

      try {
        const anonymousClientId =
          createAssistantRuntimeAnonymousClientID()

        const started = await startAssistantRuntimeSession(
          bootstrap.publicId,
          anonymousClientId,
          parentOrigin,
          { signal: controller.signal },
        )

        if (
          controller.signal.aborted ||
          generationRef.current !== generation
        ) {
          return
        }

        const welcomeMessage =
          started.welcome_message ||
          bootstrap.welcomeMessage

        setRuntimeToken(started.runtime_token)
        setSession(
          assistantEmbedFallbackSession(
            started.session_id,
            started.max_turns,
            started.expires_at,
          ),
        )
        setMessages(
          assistantEmbedWelcomeMessages(
            welcomeMessage,
          ),
        )
        setStartupState('ready')

        const synchronized = await synchronizeSession(
          started.session_id,
          started.runtime_token,
          generation,
          controller.signal,
        )

        if (
          recoveringFromTokenExpiry &&
          generationRef.current === generation
        ) {
          tokenRecoveryRef.current = false

          if (synchronized) {
            setNotice({
              kind: 'success',
              text: '学习会话已恢复，请重新发送刚才的消息。',
            })
          }
        }
      } catch (cause) {
        if (
          controller.signal.aborted ||
          generationRef.current !== generation
        ) {
          return
        }

        tokenRecoveryRef.current = false
        setStartupState('error')
        setNotice({
          kind: 'error',
          text: cause instanceof Error
            ? cause.message
            : '暂时无法开始学习互动',
        })
      }
    },
    [
      bootstrap.publicId,
      bootstrap.welcomeMessage,
      closeConnection,
      framed,
      parentOrigin,
      synchronizeSession,
    ],
  )

  /*
   * synchronizeSession通过ref触发最新的startSession，
   * 避免两个回调在Hook依赖数组中形成循环。
   */
  useEffect(() => {
    restartSessionRef.current = () => {
      void startSession(true)
    }

    return () => {
      restartSessionRef.current = () => undefined
    }
  }, [startSession])

  /*
   * 不自动创建会话。
   *
   * 学生点击“开始学习”后才调用startSession；
   * 本Effect只负责组件卸载时的资源清理。
   */
  useEffect(() => {
    return () => {
      generationRef.current += 1
      tokenRecoveryRef.current = false
      startupAbortRef.current?.abort()
      closeConnection()
    }
  }, [closeConnection])

  const canSend = Boolean(
    startupState === 'ready' &&
    session &&
    runtimeToken &&
    session.status === 'active' &&
    session.remaining_turns > 0 &&
    !sending,
  )

  const sendMessage = useCallback(() => {
    const normalized = input.trim()

    if (
      !normalized ||
      !canSend ||
      !session ||
      !runtimeToken
    ) {
      return
    }

    const generation = generationRef.current
    const currentSessionId = session.id
    const currentToken = runtimeToken

    closeConnection()
    setSending(true)
    setStreamingText('')
    setNotice(null)
    setInput('')

    setMessages(previous => [
      ...previous,
      {
        role: 'student',
        content: normalized,
        created_at: new Date().toISOString(),
      },
    ])

    let connection: AssistantRuntimeChatConnection

    try {
      connection = streamAssistantRuntimeChat(
        currentSessionId,
        currentToken,
        normalized,
        {
          onConnected: () => {
            if (generationRef.current === generation) {
              setNotice(null)
            }
          },

          onChunk: chunk => {
            if (generationRef.current !== generation) {
              return
            }

            setStreamingText(previous => previous + chunk)
          },

          onDone: result => {
            if (generationRef.current !== generation) {
              return
            }

            setStreamingText('')
            setMessages(previous => [
              ...previous,
              result.message,
            ])
            setSession(previous =>
              previous
                ? {
                    ...previous,
                    turn_count: result.turn_count,
                    remaining_turns: result.remaining_turns,
                    status: result.session_status,
                    last_active_at: result.message.created_at,
                  }
                : previous,
            )
            setNotice({
              kind: 'success',
              text: '这一轮学习互动已完成。',
            })

            void synchronizeSession(
              currentSessionId,
              currentToken,
              generation,
            )
          },

          onError: message => {
            if (generationRef.current !== generation) {
              return
            }

            setStreamingText('')
            setNotice({
              kind: 'error',
              text: message,
            })

            void synchronizeSession(
              currentSessionId,
              currentToken,
              generation,
            )
          },
        },
      )
    } catch (cause) {
      setSending(false)
      setNotice({
        kind: 'error',
        text: cause instanceof Error
          ? cause.message
          : '消息发送失败',
      })
      return
    }

    connectionRef.current = connection

    void connection.finished.finally(() => {
      if (connectionRef.current === connection) {
        connectionRef.current = null
      }

      if (generationRef.current === generation) {
        setSending(false)
        setStreamingText('')
      }
    })
  }, [
    canSend,
    closeConnection,
    input,
    runtimeToken,
    session,
    synchronizeSession,
  ])

  const statusText = useMemo(() => {
    if (!session) {
      return ''
    }

    if (session.status === 'completed') {
      return '本次学习互动已完成'
    }

    if (session.status === 'expired') {
      return '本次学习互动已过期'
    }

    if (session.status === 'revoked') {
      return '本次学习互动已结束'
    }

    if (session.remaining_turns <= 0) {
      return '本次互动轮数已用尽'
    }

    return (
      `剩余 ${session.remaining_turns} 轮 · ` +
      assistantEmbedFormatExpiry(session.expires_at)
    )
  }, [session])

  return {
    hasStarted,
    startupState,
    session,
    messages,
    streamingText,
    input,
    sending,
    notice,
    canSend,
    statusText,
    recovering: tokenRecoveryRef.current,
    setInput,
    startSession,
    sendMessage,
  }
}

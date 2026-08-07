/**
 * 教学智能体教师内部预览状态Hook。
 *
 * 设计规则：
 *   - 只预览当前页面active部署的当前不可变版本；
 *   - 创建会话使用教师JWT，聊天使用短时runtime_token；
 *   - runtime_token只保存在React内存，不写localStorage；
 *   - 聊天复用正式运行SSE、正式积分账户和正式成功/失败结算；
 *   - 教师预览消耗部署所有者积分，但不占外部学生每日额度；
 *   - 页面切换、部署版本变化、暂停撤销或组件卸载时关闭旧流连接；
 *   - 每轮结束后重新读取服务端会话，避免乐观消息与正式持久化结果失步；
 *   - completedReply只在正式done事件后更新，供课堂语音朗读完整回答，
 *     不会把尚未完成的流式片段交给浏览器朗读。
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import {
  getAssistantRuntimeSession,
  listCoursewareAssistantDeployments,
  startCoursewareAssistantPreviewSession,
  streamAssistantRuntimeChat,
} from '@/api/coursewares'

import type {
  AssistantRuntimeChatConnection,
  AssistantRuntimeMessage,
  AssistantRuntimeSessionView,
  CoursewareAssistantDeploymentView,
} from '@/api/coursewares'

export interface CoursewareAssistantPreviewNotice {
  kind: 'info' | 'success' | 'error'
  text: string
}

export interface CoursewareAssistantCompletedReply {
  sequence: number
  text: string
}

interface UseCoursewareAssistantPreviewOptions {
  coursewareId: string
  pageId: string
  refreshKey?: number
}

function welcomeMessageItem(content: string): AssistantRuntimeMessage[] {
  const normalized = content.trim()

  return normalized
    ? [{ role: 'assistant', content: normalized, created_at: null }]
    : []
}

function mergeVisibleMessages(
  welcomeMessage: string,
  formalMessages: AssistantRuntimeMessage[],
): AssistantRuntimeMessage[] {
  return [
    ...welcomeMessageItem(welcomeMessage),
    ...(formalMessages || []),
  ]
}

export function useCoursewareAssistantPreview({
  coursewareId,
  pageId,
  refreshKey = 0,
}: UseCoursewareAssistantPreviewOptions) {
  const resourceKey = `${coursewareId.trim()}:${pageId.trim()}:${refreshKey}`
  const resourceRef = useRef(resourceKey)
  resourceRef.current = resourceKey

  const loadRequestRef = useRef(0)
  const operationRef = useRef(0)
  const completedReplySequenceRef = useRef(0)
  const connectionRef = useRef<AssistantRuntimeChatConnection | null>(null)

  const [deployments, setDeployments] = useState<CoursewareAssistantDeploymentView[]>([])
  const [deploymentLoading, setDeploymentLoading] = useState(false)
  const [deploymentError, setDeploymentError] = useState('')
  const [starting, setStarting] = useState(false)
  const [sending, setSending] = useState(false)
  const [runtimeToken, setRuntimeToken] = useState('')
  const [welcomeMessage, setWelcomeMessage] = useState('')
  const [session, setSession] = useState<AssistantRuntimeSessionView | null>(null)
  const [messages, setMessages] = useState<AssistantRuntimeMessage[]>([])
  const [streamingText, setStreamingText] = useState('')
  const [notice, setNotice] = useState<CoursewareAssistantPreviewNotice | null>(null)
  const [completedReply, setCompletedReply] =
    useState<CoursewareAssistantCompletedReply | null>(null)

  const activeDeployment = useMemo(
    () => deployments.find(
      item => item.page_id === pageId && item.status === 'active',
    ) || null,
    [deployments, pageId],
  )

  const latestPageDeployment = useMemo(
    () => deployments.find(item => item.page_id === pageId) || null,
    [deployments, pageId],
  )

  const closeConnection = useCallback(() => {
    connectionRef.current?.close()
    connectionRef.current = null
  }, [])

  const clearSession = useCallback(() => {
    operationRef.current += 1
    closeConnection()
    setStarting(false)
    setSending(false)
    setRuntimeToken('')
    setWelcomeMessage('')
    setSession(null)
    setMessages([])
    setStreamingText('')
    setNotice(null)
    setCompletedReply(null)
  }, [closeConnection])

  const loadDeployments = useCallback(async () => {
    const normalizedCoursewareID = coursewareId.trim()
    const normalizedPageID = pageId.trim()
    const requestID = loadRequestRef.current + 1

    loadRequestRef.current = requestID

    if (!normalizedCoursewareID || !normalizedPageID) {
      setDeployments([])
      setDeploymentError('')
      setDeploymentLoading(false)
      return
    }

    const capturedResource = resourceKey

    setDeploymentLoading(true)
    setDeploymentError('')

    try {
      const result = await listCoursewareAssistantDeployments(
        normalizedCoursewareID,
      )

      if (
        loadRequestRef.current !== requestID
        || resourceRef.current !== capturedResource
      ) {
        return
      }

      setDeployments(result.deployments || [])
    } catch (cause) {
      if (
        loadRequestRef.current !== requestID
        || resourceRef.current !== capturedResource
      ) {
        return
      }

      setDeployments([])
      setDeploymentError(
        cause instanceof Error
          ? cause.message
          : '读取教学智能体部署状态失败',
      )
    } finally {
      if (
        loadRequestRef.current === requestID
        && resourceRef.current === capturedResource
      ) {
        setDeploymentLoading(false)
      }
    }
  }, [coursewareId, pageId, resourceKey])

  useEffect(() => {
    clearSession()
    void loadDeployments()

    return () => {
      loadRequestRef.current += 1
      operationRef.current += 1
      closeConnection()
    }
  }, [clearSession, closeConnection, loadDeployments, resourceKey])

  const synchronizeSession = useCallback(
    async (
      sessionID: string,
      token: string,
      welcome: string,
      operationID: number,
    ) => {
      try {
        const view = await getAssistantRuntimeSession(sessionID, token)

        if (
          operationRef.current !== operationID
          || resourceRef.current !== resourceKey
        ) {
          return
        }

        setSession(view)
        setMessages(
          mergeVisibleMessages(
            welcome,
            view.messages,
          ),
        )
      } catch (cause) {
        if (
          operationRef.current !== operationID
          || resourceRef.current !== resourceKey
        ) {
          return
        }

        setNotice({
          kind: 'error',
          text: cause instanceof Error
            ? cause.message
            : '同步预览会话失败，请重新建立会话',
        })
      }
    },
    [resourceKey],
  )

  const startPreview = useCallback(async () => {
    if (!activeDeployment || starting) {
      return false
    }

    const operationID = operationRef.current + 1
    const capturedResource = resourceKey

    operationRef.current = operationID

    closeConnection()
    setStarting(true)
    setSending(false)
    setRuntimeToken('')
    setWelcomeMessage('')
    setSession(null)
    setMessages([])
    setStreamingText('')
    setNotice(null)
    setCompletedReply(null)

    try {
      const started = await startCoursewareAssistantPreviewSession(
        activeDeployment.id,
      )

      if (
        operationRef.current !== operationID
        || resourceRef.current !== capturedResource
      ) {
        return false
      }

      const fallbackView: AssistantRuntimeSessionView = {
        id: started.session_id,
        deployment_version: activeDeployment.current_version,
        session_kind: 'teacher_preview',
        status: started.status,
        turn_count: 0,
        max_turns: started.max_turns,
        remaining_turns: started.max_turns,
        messages: [],
        expires_at: started.expires_at,
        last_active_at: null,
      }

      setRuntimeToken(started.runtime_token)
      setWelcomeMessage(started.welcome_message)
      setSession(fallbackView)
      setMessages(
        welcomeMessageItem(
          started.welcome_message,
        ),
      )
      setNotice({
        kind: 'info',
        text: '已建立教师真实预览会话。本次对话会消耗你的个人教学积分。',
      })

      await synchronizeSession(
        started.session_id,
        started.runtime_token,
        started.welcome_message,
        operationID,
      )

      return true
    } catch (cause) {
      if (
        operationRef.current !== operationID
        || resourceRef.current !== capturedResource
      ) {
        return false
      }

      setNotice({
        kind: 'error',
        text: cause instanceof Error
          ? cause.message
          : '启动教师预览失败',
      })

      return false
    } finally {
      if (
        operationRef.current === operationID
        && resourceRef.current === capturedResource
      ) {
        setStarting(false)
      }
    }
  }, [
    activeDeployment,
    closeConnection,
    resourceKey,
    starting,
    synchronizeSession,
  ])

  const sendMessage = useCallback(
    (rawMessage: string): boolean => {
      const normalizedMessage = rawMessage.trim()

      if (
        !normalizedMessage
        || !session
        || !runtimeToken
        || sending
        || session.status !== 'active'
        || session.remaining_turns <= 0
      ) {
        return false
      }

      const operationID = operationRef.current
      const capturedResource = resourceKey
      const currentSessionID = session.id
      const currentToken = runtimeToken
      const currentWelcome = welcomeMessage

      const optimisticStudentMessage: AssistantRuntimeMessage = {
        role: 'student',
        content: normalizedMessage,
        created_at: new Date().toISOString(),
      }

      closeConnection()
      setSending(true)
      setStreamingText('')
      setNotice(null)
      setMessages(previous => [
        ...previous,
        optimisticStudentMessage,
      ])

      let connection: AssistantRuntimeChatConnection

      try {
        connection = streamAssistantRuntimeChat(
          currentSessionID,
          currentToken,
          normalizedMessage,
          {
            onConnected: () => {
              if (
                operationRef.current === operationID
                && resourceRef.current === capturedResource
              ) {
                setNotice(null)
              }
            },

            onChunk: chunk => {
              if (
                operationRef.current === operationID
                && resourceRef.current === capturedResource
              ) {
                setStreamingText(previous => previous + chunk)
              }
            },

            onDone: result => {
              if (
                operationRef.current !== operationID
                || resourceRef.current !== capturedResource
              ) {
                return
              }

              setSession(previous => previous
                ? {
                    ...previous,
                    turn_count: result.turn_count,
                    remaining_turns: result.remaining_turns,
                    status: result.session_status,
                    last_active_at: result.message.created_at,
                  }
                : previous)

              const completedText = result.message.content.trim()

              if (completedText) {
                completedReplySequenceRef.current += 1
                setCompletedReply({
                  sequence: completedReplySequenceRef.current,
                  text: completedText,
                })
              }

              setNotice({
                kind: 'success',
                text: '本轮教师预览已完成并按实际模型用量结算。',
              })

              void synchronizeSession(
                currentSessionID,
                currentToken,
                currentWelcome,
                operationID,
              )
            },

            onError: message => {
              if (
                operationRef.current !== operationID
                || resourceRef.current !== capturedResource
              ) {
                return
              }

              setNotice({
                kind: 'error',
                text: message,
              })

              void synchronizeSession(
                currentSessionID,
                currentToken,
                currentWelcome,
                operationID,
              )
            },
          },
        )
      } catch (cause) {
        setSending(false)
        setMessages(previous => previous.slice(0, -1))
        setNotice({
          kind: 'error',
          text: cause instanceof Error
            ? cause.message
            : '发送预览消息失败',
        })
        return false
      }

      connectionRef.current = connection

      void connection.finished.finally(() => {
        if (connectionRef.current === connection) {
          connectionRef.current = null
        }

        if (
          operationRef.current === operationID
          && resourceRef.current === capturedResource
        ) {
          setSending(false)
          setStreamingText('')
        }
      })

      return true
    },
    [
      closeConnection,
      resourceKey,
      runtimeToken,
      sending,
      session,
      synchronizeSession,
      welcomeMessage,
    ],
  )

  const remainingTurns = session?.remaining_turns ?? 0

  const canSend = Boolean(
    session
    && runtimeToken
    && session.status === 'active'
    && remainingTurns > 0
    && !sending,
  )

  return {
    activeDeployment,
    latestPageDeployment,
    deploymentLoading,
    deploymentError,
    starting,
    sending,
    session,
    messages,
    streamingText,
    notice,
    completedReply,
    remainingTurns,
    canSend,
    loadDeployments,
    startPreview,
    sendMessage,
    clearSession,
  }
}

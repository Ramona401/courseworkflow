/**
 * useVoiceInput.ts — 全平台统一语音输入状态Hook
 *
 * Hook负责：
 * 1. 麦克风权限；
 * 2. 本站WebSocket连接；
 * 3. 录音状态与最长时长；
 * 4. partial和final回调；
 * 5. 取消、报错、页面卸载时的资源释放。
 *
 * PCM采集与降采样由voiceAudio.ts负责；
 * 业务组件只接收文字，不会被本Hook自动触发AI发送。
 *
 * 异步并发约束：
 * - 每次start生成attemptID；
 * - cancel、fail、重新start或卸载都会使旧attempt失效；
 * - getUserMedia、AudioWorklet和WebSocket的迟到回调
 *   不得覆盖新一轮语音状态或重新占用麦克风。
 */

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'
import { useAuth } from '@/store/auth'
import {
  createSpeechStream,
  type SpeechRecognitionEvent,
  type SpeechStreamConnection,
} from '@/api/speech'
import {
  createVoicePCMRecorder,
  microphoneErrorMessage,
  stopMediaStream,
  supportsVoiceInput,
  type VoicePCMRecorder,
} from '@/utils/voiceAudio'

export type VoiceInputStatus =
  | 'idle'
  | 'connecting'
  | 'recording'
  | 'stopping'
  | 'error'

export interface UseVoiceInputOptions {
  disabled?: boolean
  maxDurationSeconds?: number
  onPartial?: (
    text: string,
    event: SpeechRecognitionEvent,
  ) => void
  onFinal?: (
    text: string,
    event: SpeechRecognitionEvent,
  ) => void
  onError?: (
    message: string,
  ) => void
}

export interface UseVoiceInputResult {
  status: VoiceInputStatus
  isActive: boolean
  isSupported: boolean
  elapsedSeconds: number
  error: string
  start: () => Promise<void>
  stop: () => void
  cancel: () => void
}

const STOP_RESULT_TIMEOUT_MS =
  45000

export function useVoiceInput(
  options: UseVoiceInputOptions,
): UseVoiceInputResult {
  const {
    disabled = false,
    maxDurationSeconds = 120,
    onPartial,
    onFinal,
    onError,
  } = options

  const { token } = useAuth()

  const [
    status,
    setStatus,
  ] =
    useState<VoiceInputStatus>(
      'idle',
    )

  const [
    elapsedSeconds,
    setElapsedSeconds,
  ] = useState(0)

  const [
    error,
    setError,
  ] = useState('')

  const mountedRef =
    useRef(true)

  const statusRef =
    useRef<VoiceInputStatus>(
      'idle',
    )

  const attemptRef =
    useRef(0)

  const onPartialRef =
    useRef(onPartial)

  const onFinalRef =
    useRef(onFinal)

  const onErrorRef =
    useRef(onError)

  const connectionRef =
    useRef<
      SpeechStreamConnection | null
    >(null)

  const recorderRef =
    useRef<
      VoicePCMRecorder | null
    >(null)

  const pendingStreamRef =
    useRef<
      MediaStream | null
    >(null)

  const finalReceivedRef =
    useRef(false)

  const elapsedTimerRef =
    useRef<
      ReturnType<
        typeof setInterval
      > | null
    >(null)

  const durationTimerRef =
    useRef<
      ReturnType<
        typeof setTimeout
      > | null
    >(null)

  const stopTimerRef =
    useRef<
      ReturnType<
        typeof setTimeout
      > | null
    >(null)

  const isSupported =
    supportsVoiceInput()

  const setVoiceStatus =
    useCallback(
      (
        next:
          VoiceInputStatus,
      ) => {
        statusRef.current =
          next

        if (
          mountedRef.current
        ) {
          setStatus(next)
        }
      },
      [],
    )

  const isCurrentAttempt =
    useCallback(
      (
        attemptID: number,
      ) =>
        mountedRef.current &&
        attemptRef.current ===
          attemptID,
      [],
    )

  const clearTimers =
    useCallback(() => {
      if (
        elapsedTimerRef.current
      ) {
        clearInterval(
          elapsedTimerRef.current,
        )
      }

      if (
        durationTimerRef.current
      ) {
        clearTimeout(
          durationTimerRef.current,
        )
      }

      if (
        stopTimerRef.current
      ) {
        clearTimeout(
          stopTimerRef.current,
        )
      }

      elapsedTimerRef.current =
        null

      durationTimerRef.current =
        null

      stopTimerRef.current =
        null
    }, [])

  const closeConnection =
    useCallback(
      (
        sendCancel: boolean,
      ) => {
        const connection =
          connectionRef.current

        connectionRef.current =
          null

        if (!connection) return

        if (sendCancel) {
          connection.cancel()
        }

        connection.close()
      },
      [],
    )

  const releaseMedia =
    useCallback(() => {
      recorderRef.current
        ?.destroy()

      recorderRef.current =
        null

      stopMediaStream(
        pendingStreamRef.current,
      )

      pendingStreamRef.current =
        null
    }, [])

  const releaseAll =
    useCallback(
      (
        sendCancel: boolean,
      ) => {
        /**
         * 先使旧异步回调失效，
         * 再释放其占用的资源。
         */
        attemptRef.current += 1

        clearTimers()
        releaseMedia()
        closeConnection(
          sendCancel,
        )

        finalReceivedRef.current =
          false

        if (
          mountedRef.current
        ) {
          setElapsedSeconds(0)
        }
      },
      [
        clearTimers,
        closeConnection,
        releaseMedia,
      ],
    )

  const fail =
    useCallback(
      (
        message: string,
      ) => {
        releaseAll(true)

        if (
          mountedRef.current
        ) {
          setError(message)
          setVoiceStatus(
            'error',
          )
        }

        onErrorRef.current?.(
          message,
        )
      },
      [
        releaseAll,
        setVoiceStatus,
      ],
    )

  const failAttempt =
    useCallback(
      (
        attemptID: number,
        message: string,
      ) => {
        if (
          !isCurrentAttempt(
            attemptID,
          )
        ) {
          return
        }

        fail(message)
      },
      [
        fail,
        isCurrentAttempt,
      ],
    )

  const beginElapsedTimers =
    useCallback(
      (
        attemptID: number,
      ) => {
        const startedAt =
          Date.now()

        elapsedTimerRef.current =
          setInterval(() => {
            if (
              isCurrentAttempt(
                attemptID,
              )
            ) {
              setElapsedSeconds(
                Math.floor(
                  (
                    Date.now() -
                    startedAt
                  ) / 1000,
                ),
              )
            }
          }, 250)

        durationTimerRef.current =
          setTimeout(() => {
            if (
              isCurrentAttempt(
                attemptID,
              ) &&
              statusRef.current ===
                'recording'
            ) {
              stopRef.current()
            }
          }, Math.max(
            5,
            maxDurationSeconds,
          ) * 1000)
      },
      [
        isCurrentAttempt,
        maxDurationSeconds,
      ],
    )

  const activateRecorder =
    useCallback(
      async (
        attemptID: number,
      ) => {
        if (
          !isCurrentAttempt(
            attemptID,
          )
        ) {
          return
        }

        const stream =
          pendingStreamRef.current

        const connection =
          connectionRef.current

        if (
          !stream ||
          !connection ||
          !connection.isReady()
        ) {
          failAttempt(
            attemptID,
            '麦克风或语音连接初始化未完成',
          )
          return
        }

        const recorder =
          createVoicePCMRecorder({
            onPCM: (pcm) => {
              if (
                !isCurrentAttempt(
                  attemptID,
                )
              ) {
                return
              }

              connection.sendAudio(
                pcm,
              )
            },

            onError: (
              message,
            ) => {
              failAttempt(
                attemptID,
                message,
              )
            },
          })

        recorderRef.current =
          recorder

        try {
          await recorder.start(
            stream,
          )

          if (
            !isCurrentAttempt(
              attemptID,
            )
          ) {
            recorder.destroy()
            return
          }

          pendingStreamRef.current =
            null

          setError('')

          setVoiceStatus(
            'recording',
          )

          beginElapsedTimers(
            attemptID,
          )
        } catch (cause) {
          failAttempt(
            attemptID,
            microphoneErrorMessage(
              cause,
            ),
          )
        }
      },
      [
        beginElapsedTimers,
        failAttempt,
        isCurrentAttempt,
        setVoiceStatus,
      ],
    )

  const stopRef =
    useRef<
      () => void
    >(() => {})

  const stop =
    useCallback(() => {
      if (
        statusRef.current !==
        'recording'
      ) {
        return
      }

      const attemptID =
        attemptRef.current

      clearTimers()

      const recorder =
        recorderRef.current

      recorderRef.current =
        null

      const chunkCount =
        recorder?.stop(true) ||
        0

      /**
       * stop(true)发送最后残余时可能同步触发onError，
       * onError会使attempt失效；此时不能继续发送stop控制消息。
       */
      if (
        !isCurrentAttempt(
          attemptID,
        )
      ) {
        return
      }

      if (
        chunkCount === 0
      ) {
        failAttempt(
          attemptID,
          '没有采集到有效语音，请重新录音',
        )
        return
      }

      const connection =
        connectionRef.current

      if (!connection) {
        failAttempt(
          attemptID,
          '语音连接已经关闭',
        )
        return
      }

      try {
        connection.stop()

        setVoiceStatus(
          'stopping',
        )

        stopTimerRef.current =
          setTimeout(() => {
            if (
              isCurrentAttempt(
                attemptID,
              ) &&
              !finalReceivedRef.current
            ) {
              failAttempt(
                attemptID,
                '等待最终识别结果超时，请重新尝试',
              )
            }
          }, STOP_RESULT_TIMEOUT_MS)
      } catch (cause) {
        failAttempt(
          attemptID,
          cause instanceof Error
            ? cause.message
            : '停止语音识别失败',
        )
      }
    }, [
      clearTimers,
      failAttempt,
      isCurrentAttempt,
      setVoiceStatus,
    ])

  stopRef.current = stop

  const cancel =
    useCallback(() => {
      if (
        statusRef.current ===
          'idle' ||
        statusRef.current ===
          'error'
      ) {
        return
      }

      releaseAll(true)

      setError('')

      setVoiceStatus(
        'idle',
      )
    }, [
      releaseAll,
      setVoiceStatus,
    ])

  const start =
    useCallback(
      async () => {
        if (
          disabled ||
          !isSupported ||
          statusRef.current ===
            'connecting' ||
          statusRef.current ===
            'recording' ||
          statusRef.current ===
            'stopping'
        ) {
          return
        }

        const normalizedToken =
          token?.trim() || ''

        if (!normalizedToken) {
          fail(
            '登录状态已失效，请重新登录',
          )
          return
        }

        releaseAll(false)

        const attemptID =
          attemptRef.current

        setError('')
        setElapsedSeconds(0)

        setVoiceStatus(
          'connecting',
        )

        try {
          const stream =
            await navigator
              .mediaDevices
              .getUserMedia({
                audio: {
                  channelCount: 1,
                  echoCancellation:
                    true,
                  noiseSuppression:
                    true,
                  autoGainControl:
                    true,
                },
                video: false,
              })

          /**
           * cancel、卸载或新一轮start发生后，
           * 迟到的麦克风流必须立即释放。
           */
          if (
            !isCurrentAttempt(
              attemptID,
            )
          ) {
            stopMediaStream(stream)
            return
          }

          pendingStreamRef.current =
            stream

          const connection =
            createSpeechStream(
              normalizedToken,
              {
                onReady: () => {
                  if (
                    isCurrentAttempt(
                      attemptID,
                    )
                  ) {
                    void activateRecorder(
                      attemptID,
                    )
                  }
                },

                onPartial: (
                  event,
                ) => {
                  if (
                    !isCurrentAttempt(
                      attemptID,
                    )
                  ) {
                    return
                  }

                  const text =
                    event.text?.trim() ||
                    ''

                  if (text) {
                    onPartialRef.current?.(
                      text,
                      event,
                    )
                  }
                },

                onFinal: (
                  event,
                ) => {
                  if (
                    !isCurrentAttempt(
                      attemptID,
                    )
                  ) {
                    return
                  }

                  const text =
                    event.text?.trim() ||
                    ''

                  if (!text) {
                    failAttempt(
                      attemptID,
                      '没有识别到有效文字，请重新录音',
                    )
                    return
                  }

                  finalReceivedRef.current =
                    true

                  releaseAll(false)

                  if (
                    mountedRef.current
                  ) {
                    setError('')

                    setVoiceStatus(
                      'idle',
                    )
                  }

                  onFinalRef.current?.(
                    text,
                    event,
                  )
                },

                onError: (
                  event,
                ) => {
                  failAttempt(
                    attemptID,
                    event.message ||
                      '语音识别失败，请稍后重试',
                  )
                },

                onClosed: () => {
                  if (
                    isCurrentAttempt(
                      attemptID,
                    ) &&
                    !finalReceivedRef.current
                  ) {
                    failAttempt(
                      attemptID,
                      '语音连接已关闭，未收到最终识别文字',
                    )
                  }
                },

                onUnexpectedClose: (
                  message,
                ) => {
                  if (
                    isCurrentAttempt(
                      attemptID,
                    ) &&
                    !finalReceivedRef.current
                  ) {
                    failAttempt(
                      attemptID,
                      message,
                    )
                  }
                },
              },
            )

          if (
            !isCurrentAttempt(
              attemptID,
            )
          ) {
            connection.close()
            return
          }

          connectionRef.current =
            connection
        } catch (cause) {
          failAttempt(
            attemptID,
            microphoneErrorMessage(
              cause,
            ),
          )
        }
      },
      [
        activateRecorder,
        disabled,
        fail,
        failAttempt,
        isCurrentAttempt,
        isSupported,
        releaseAll,
        setVoiceStatus,
        token,
      ],
    )

  useEffect(() => {
    onPartialRef.current =
      onPartial
  }, [onPartial])

  useEffect(() => {
    onFinalRef.current =
      onFinal
  }, [onFinal])

  useEffect(() => {
    onErrorRef.current =
      onError
  }, [onError])

  useEffect(() => {
    if (
      disabled &&
      (
        statusRef.current ===
          'connecting' ||
        statusRef.current ===
          'recording' ||
        statusRef.current ===
          'stopping'
      )
    ) {
      cancel()
    }
  }, [
    cancel,
    disabled,
  ])

  useEffect(() => {
    mountedRef.current = true

    return () => {
      mountedRef.current = false
      releaseAll(true)
    }
  }, [releaseAll])

  return {
    status,
    isActive:
      status === 'connecting' ||
      status === 'recording' ||
      status === 'stopping',
    isSupported,
    elapsedSeconds,
    error,
    start,
    stop,
    cancel,
  }
}

export default useVoiceInput

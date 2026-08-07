/**
 * useAssistantSpeechPlayback.ts — 老师端教学智能体豆包优先朗读。
 *
 * 播放策略：
 *   1. 完整回答先请求后端豆包TTS；
 *   2. 中文默认vivi 2.0，英文或字母为主默认Tim，音色由后端统一决定；
 *   3. 成功MP3保留在当前页面内存中，暂停、继续、停止和重播不重复计费；
 *   4. 网络响应丢失后的手动重试复用同一operation_id，由后端幂等恢复；
 *   5. 豆包不可用时自动降级到浏览器本地语音，保证课堂不中断；
 *   6. 开始录音、关闭面板、切换页面或组件卸载时可立即停止和取消合成请求。
 */

import { useCallback, useEffect, useRef, useState } from 'react'
import { synthesizeCoursewareAssistantSpeech } from '@/api/coursewares'

const ASSISTANT_TTS_MAX_RUNES = 3950
const BROWSER_SPEECH_CHUNK_MAX_RUNES = 180

export type AssistantSpeechProvider = 'doubao' | 'browser' | 'none'

export interface AssistantSpeechPlaybackResult {
  isSupported: boolean
  preparing: boolean
  speaking: boolean
  paused: boolean
  error: string
  warning: string
  provider: AssistantSpeechProvider
  voiceLabel: string
  lastText: string
  audioElement: HTMLAudioElement | null
  speak: (text: string) => void
  stop: () => void
  togglePause: () => void
  replay: () => void
}

function normalizeAssistantSpeechText(rawText: string): string {
  const normalized = rawText
    .replace(/```[\s\S]*?```/g, ' 代码内容请查看屏幕。 ')
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/https?:\/\/\S+/g, ' 链接内容请查看屏幕。 ')
    .replace(/[`*_>#|~\-]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()

  const runes = Array.from(normalized)

  if (runes.length <= ASSISTANT_TTS_MAX_RUNES) {
    return normalized
  }

  return `${runes.slice(0, ASSISTANT_TTS_MAX_RUNES).join('')}。后续内容请查看屏幕。`
}

function createAssistantTTSOperationID(): string {
  if (
    typeof crypto !== 'undefined'
    && typeof crypto.randomUUID === 'function'
  ) {
    return crypto.randomUUID()
  }

  if (
    typeof crypto !== 'undefined'
    && typeof crypto.getRandomValues === 'function'
  ) {
    const bytes = new Uint8Array(16)
    crypto.getRandomValues(bytes)
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

  throw new Error('当前浏览器不支持安全朗读任务标识')
}

function splitLongFragment(fragment: string): string[] {
  const runes = Array.from(fragment)

  if (runes.length <= BROWSER_SPEECH_CHUNK_MAX_RUNES) {
    return [fragment]
  }

  const chunks: string[] = []

  for (
    let index = 0;
    index < runes.length;
    index += BROWSER_SPEECH_CHUNK_MAX_RUNES
  ) {
    const chunk = runes
      .slice(index, index + BROWSER_SPEECH_CHUNK_MAX_RUNES)
      .join('')
      .trim()

    if (chunk) {
      chunks.push(chunk)
    }
  }

  return chunks
}

function splitBrowserSpeechText(text: string): string[] {
  const sentences = text.match(/[^。！？!?；;\n]+[。！？!?；;]?/g) || [text]
  const chunks: string[] = []
  let current = ''

  const flushCurrent = () => {
    const normalized = current.trim()

    if (normalized) {
      chunks.push(normalized)
    }

    current = ''
  }

  for (const rawSentence of sentences) {
    const sentence = rawSentence.trim()

    if (!sentence) {
      continue
    }

    if (Array.from(sentence).length > BROWSER_SPEECH_CHUNK_MAX_RUNES) {
      flushCurrent()
      chunks.push(...splitLongFragment(sentence))
      continue
    }

    const candidate = `${current}${current ? ' ' : ''}${sentence}`

    if (Array.from(candidate).length > BROWSER_SPEECH_CHUNK_MAX_RUNES) {
      flushCurrent()
      current = sentence
      continue
    }

    current = candidate
  }

  flushCurrent()
  return chunks
}

function isEnglishDominant(text: string): boolean {
  let latinCount = 0
  let chineseCount = 0

  for (const character of text) {
    if (/[A-Za-z]/.test(character)) {
      latinCount += 1
    } else if (/\p{Script=Han}/u.test(character)) {
      chineseCount += 1
    }
  }

  return latinCount >= 3 && latinCount > chineseCount * 1.35
}

function selectBrowserVoice(
  synthesis: SpeechSynthesis,
  text: string,
): SpeechSynthesisVoice | null {
  const voices = synthesis.getVoices()
  const languagePrefix = isEnglishDominant(text) ? 'en' : 'zh'

  const exactLocal = voices.find(
    voice => voice.localService
      && voice.lang.toLowerCase().startsWith(languagePrefix),
  )

  if (exactLocal) {
    return exactLocal
  }

  return voices.find(
    voice => voice.lang.toLowerCase().startsWith(languagePrefix),
  ) || null
}

function isCancelledRequest(error: unknown): boolean {
  if (!error || typeof error !== 'object') {
    return false
  }

  const candidate = error as { code?: unknown; name?: unknown }

  return candidate.code === 'ERR_CANCELED'
    || candidate.name === 'CanceledError'
    || candidate.name === 'AbortError'
}

function errorMessage(error: unknown): string {
  return error instanceof Error && error.message.trim()
    ? error.message.trim()
    : '豆包朗读请求失败'
}

export function useAssistantSpeechPlayback(
  deploymentId: string,
): AssistantSpeechPlaybackResult {
  const normalizedDeploymentID = deploymentId.trim()
  const browserSpeechSupported =
    typeof window !== 'undefined'
    && 'speechSynthesis' in window
    && 'SpeechSynthesisUtterance' in window

  const doubaoSupported =
    typeof window !== 'undefined'
    && typeof Audio !== 'undefined'
    && Boolean(normalizedDeploymentID)

  const [preparing, setPreparing] = useState(false)
  const [speaking, setSpeaking] = useState(false)
  const [paused, setPaused] = useState(false)
  const [error, setError] = useState('')
  const [warning, setWarning] = useState('')
  const [provider, setProvider] = useState<AssistantSpeechProvider>('none')
  const [voiceLabel, setVoiceLabel] = useState('')
  const [lastText, setLastText] = useState('')
  const [audioElement, setAudioElement] = useState<HTMLAudioElement | null>(null)

  const playbackTokenRef = useRef(0)
  const requestControllerRef = useRef<AbortController | null>(null)
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const audioURLRef = useRef('')
  const lastTextRef = useRef('')
  const lastOperationIDRef = useRef('')
  const browserQueueRef = useRef<string[]>([])
  const browserPlayNextRef = useRef<(token: number, text: string) => void>(
    () => undefined,
  )

  const revokeAudioURL = useCallback(() => {
    if (audioURLRef.current) {
      URL.revokeObjectURL(audioURLRef.current)
      audioURLRef.current = ''
    }
  }, [])

  const stopCurrentPlayback = useCallback(() => {
    playbackTokenRef.current += 1
    requestControllerRef.current?.abort()
    requestControllerRef.current = null
    browserQueueRef.current = []

    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current.currentTime = 0
    }

    if (browserSpeechSupported) {
      window.speechSynthesis.cancel()
    }

    setPreparing(false)
    setSpeaking(false)
    setPaused(false)
  }, [browserSpeechSupported])

  const playBrowserNext = useCallback(
    (token: number, fullText: string) => {
      if (
        !browserSpeechSupported
        || token !== playbackTokenRef.current
      ) {
        return
      }

      const nextChunk = browserQueueRef.current.shift()

      if (!nextChunk) {
        setSpeaking(false)
        setPaused(false)
        return
      }

      const synthesis = window.speechSynthesis
      const utterance = new SpeechSynthesisUtterance(nextChunk)
      const voice = selectBrowserVoice(synthesis, fullText)

      if (voice) {
        utterance.voice = voice
        utterance.lang = voice.lang
        setVoiceLabel(voice.name || '设备语音')
      } else {
        utterance.lang = isEnglishDominant(fullText) ? 'en-US' : 'zh-CN'
        setVoiceLabel('设备语音')
      }

      utterance.rate = 1
      utterance.pitch = 1
      utterance.volume = 1

      utterance.onend = () => {
        if (token === playbackTokenRef.current) {
          browserPlayNextRef.current(token, fullText)
        }
      }

      utterance.onerror = event => {
        if (token !== playbackTokenRef.current) {
          return
        }

        if (event.error === 'canceled' || event.error === 'interrupted') {
          return
        }

        browserQueueRef.current = []
        setSpeaking(false)
        setPaused(false)
        setError('设备语音朗读失败，请点击“重新朗读”重试')
      }

      synthesis.speak(utterance)
    },
    [browserSpeechSupported],
  )

  browserPlayNextRef.current = playBrowserNext

  const startBrowserFallback = useCallback(
    (text: string, reason: string, token: number) => {
      if (!browserSpeechSupported || token !== playbackTokenRef.current) {
        setPreparing(false)
        setSpeaking(false)
        setPaused(false)
        setError(reason || '豆包朗读失败，当前浏览器也不支持设备语音')
        setProvider('none')
        return
      }

      audioRef.current?.pause()
      audioRef.current = null
      setAudioElement(null)

      const chunks = splitBrowserSpeechText(text)

      if (chunks.length === 0) {
        setError('当前回答没有可朗读的文字')
        setProvider('none')
        return
      }

      browserQueueRef.current = chunks
      setPreparing(false)
      setSpeaking(true)
      setPaused(false)
      setProvider('browser')
      setWarning(`豆包朗读暂不可用，已自动切换设备语音：${reason}`)
      setError('')
      browserPlayNextRef.current(token, text)
    },
    [browserSpeechSupported],
  )

  const playDoubaoAudio = useCallback(
    (
      text: string,
      audioURL: string,
      voiceName: string,
      token: number,
    ) => {
      if (token !== playbackTokenRef.current) {
        return
      }

      const audio = new Audio(audioURL)
      audio.preload = 'auto'
      audioRef.current = audio
      setAudioElement(audio)

      audio.onplay = () => {
        if (token !== playbackTokenRef.current) {
          return
        }

        setPreparing(false)
        setSpeaking(true)
        setPaused(false)
        setProvider('doubao')
        setVoiceLabel(voiceName || '豆包自然音色')
        setWarning('')
        setError('')
      }

      audio.onpause = () => {
        if (
          token === playbackTokenRef.current
          && !audio.ended
          && audio.currentTime > 0
        ) {
          setSpeaking(true)
          setPaused(true)
        }
      }

      audio.onended = () => {
        if (token !== playbackTokenRef.current) {
          return
        }

        setSpeaking(false)
        setPaused(false)
      }

      audio.onerror = () => {
        if (token !== playbackTokenRef.current) {
          return
        }

        startBrowserFallback(
          text,
          '豆包音频播放失败',
          token,
        )
      }

      void audio.play().catch(cause => {
        if (token !== playbackTokenRef.current) {
          return
        }

        startBrowserFallback(
          text,
          cause instanceof Error ? cause.message : '浏览器阻止了豆包音频播放',
          token,
        )
      })
    },
    [startBrowserFallback],
  )

  const requestDoubaoSpeech = useCallback(
    async (
      text: string,
      operationID: string,
      token: number,
    ) => {
      if (!doubaoSupported || !normalizedDeploymentID) {
        startBrowserFallback(
          text,
          '当前教学智能体部署暂不能调用豆包朗读',
          token,
        )
        return
      }

      const controller = new AbortController()
      requestControllerRef.current = controller

      setPreparing(true)
      setSpeaking(false)
      setPaused(false)
      setProvider('none')
      setVoiceLabel('')
      setWarning('')
      setError('')

      try {
        const result = await synthesizeCoursewareAssistantSpeech(
          normalizedDeploymentID,
          text,
          operationID,
          controller.signal,
        )

        if (token !== playbackTokenRef.current) {
          return
        }

        requestControllerRef.current = null
        audioRef.current?.pause()
        audioRef.current = null
        setAudioElement(null)
        revokeAudioURL()

        const audioURL = URL.createObjectURL(result.audio)
        audioURLRef.current = audioURL

        playDoubaoAudio(
          text,
          audioURL,
          result.voiceName,
          token,
        )
      } catch (cause) {
        if (
          token !== playbackTokenRef.current
          || isCancelledRequest(cause)
        ) {
          return
        }

        requestControllerRef.current = null
        startBrowserFallback(
          text,
          errorMessage(cause),
          token,
        )
      }
    },
    [
      doubaoSupported,
      normalizedDeploymentID,
      playDoubaoAudio,
      revokeAudioURL,
      startBrowserFallback,
    ],
  )

  const speak = useCallback(
    (rawText: string) => {
      const normalizedText = normalizeAssistantSpeechText(rawText)

      if (!normalizedText) {
        setError('当前回答没有可朗读的文字')
        return
      }

      stopCurrentPlayback()
      revokeAudioURL()
      audioRef.current = null
      setAudioElement(null)

      const token = playbackTokenRef.current
      let operationID = ''

      lastTextRef.current = normalizedText
      lastOperationIDRef.current = ''
      setLastText(normalizedText)

      try {
        operationID = createAssistantTTSOperationID()
      } catch (cause) {
        startBrowserFallback(normalizedText, errorMessage(cause), token)
        return
      }

      lastOperationIDRef.current = operationID

      void requestDoubaoSpeech(
        normalizedText,
        operationID,
        token,
      )
    },
    [
      requestDoubaoSpeech,
      revokeAudioURL,
      startBrowserFallback,
      stopCurrentPlayback,
    ],
  )

  const togglePause = useCallback(() => {
    if (!speaking) {
      return
    }

    if (provider === 'doubao' && audioRef.current) {
      if (paused) {
        void audioRef.current.play().catch(() => {
          setError('继续播放失败，请点击“重新朗读”')
        })
      } else {
        audioRef.current.pause()
      }

      return
    }

    if (provider === 'browser' && browserSpeechSupported) {
      if (paused) {
        window.speechSynthesis.resume()
        setPaused(false)
      } else {
        window.speechSynthesis.pause()
        setPaused(true)
      }
    }
  }, [browserSpeechSupported, paused, provider, speaking])

  const replay = useCallback(() => {
    const text = lastTextRef.current

    if (!text) {
      setError('当前还没有可以重新朗读的回答')
      return
    }

    stopCurrentPlayback()
    const token = playbackTokenRef.current

    if (audioURLRef.current) {
      playDoubaoAudio(
        text,
        audioURLRef.current,
        voiceLabel || '豆包自然音色',
        token,
      )
      return
    }

    const operationID = lastOperationIDRef.current

    if (operationID) {
      void requestDoubaoSpeech(text, operationID, token)
      return
    }

    startBrowserFallback(text, '豆包朗读任务尚未建立', token)
  }, [
    playDoubaoAudio,
    requestDoubaoSpeech,
    startBrowserFallback,
    stopCurrentPlayback,
    voiceLabel,
  ])

  useEffect(() => {
    stopCurrentPlayback()
    revokeAudioURL()
    audioRef.current = null
    setAudioElement(null)
    lastTextRef.current = ''
    lastOperationIDRef.current = ''
    setLastText('')
    setProvider('none')
    setVoiceLabel('')
    setWarning('')
    setError('')
  }, [
    normalizedDeploymentID,
    revokeAudioURL,
    stopCurrentPlayback,
  ])

  useEffect(() => {
    return () => {
      playbackTokenRef.current += 1
      requestControllerRef.current?.abort()
      requestControllerRef.current = null
      audioRef.current?.pause()
      audioRef.current = null
      browserQueueRef.current = []

      if (browserSpeechSupported) {
        window.speechSynthesis.cancel()
      }

      revokeAudioURL()
    }
  }, [browserSpeechSupported, revokeAudioURL])

  return {
    isSupported: doubaoSupported || browserSpeechSupported,
    preparing,
    speaking,
    paused,
    error,
    warning,
    provider,
    voiceLabel,
    lastText,
    audioElement,
    speak,
    stop: stopCurrentPlayback,
    togglePause,
    replay,
  }
}

export default useAssistantSpeechPlayback


/**
 * useClassroomDigitalHumanGesture.ts — 课堂数字人语义低频动作调度。
 *
 * 8.11节奏优化：
 * - 男女老师都支持列举、指向课件、展开、邀请、强调、鼓励、鼓掌；
 * - 仍然优先按回答语义触发，不恢复固定频率随机换动作；
 * - 50字以下保持稳定不动，普通回答最多1次，120字以上最多2次；
 * - 第一个语义动作最早可在朗读约10%位置出现，避免前半段长时间完全静止；
 * - 长回答如果第一个语义点很靠后，会在约24%位置补一个轻量展开动作；
 * - 没有明显语义词的较长回答，在约28%位置安排一次自然展开动作；
 * - 豆包音频可用时继续按audio.currentTime / duration跟随真实朗读进度；
 * - 浏览器语音兜底时使用更早但仍克制的时间近似；
 * - 鼓掌只匹配强正向反馈，普通“很好/不错”使用轻鼓励；
 * - 动作停留时间调整到约1.25～1.55秒，让抬手、指向和强调更完整自然；
 * - 不额外调用AI，不产生模型费用。
 */

import { useEffect, useState } from 'react'

import type {
  ClassroomDigitalHumanCharacter,
  ClassroomDigitalHumanState,
} from './ClassroomDigitalHuman'

export type ClassroomDigitalHumanGesture =
  | 'enumerate'
  | 'point_left'
  | 'expand'
  | 'invite'
  | 'emphasis'
  | 'encourage'
  | 'applause'

interface UseClassroomDigitalHumanGestureOptions {
  state: ClassroomDigitalHumanState
  character: ClassroomDigitalHumanCharacter
  speechText: string
  speechPaused: boolean
  audioElement: HTMLAudioElement | null
}

interface GestureCandidate {
  gesture: ClassroomDigitalHumanGesture
  textIndex: number
  targetProgress: number
}

interface SemanticRule {
  gesture: ClassroomDigitalHumanGesture
  keywords: readonly string[]
}

const SEMANTIC_RULES: readonly SemanticRule[] = [
  {
    gesture: 'applause',
    keywords: ['完全正确', '答对了', '答对', '非常棒', '太棒了', '回答得很好', '做得非常好'],
  },
  {
    gesture: 'encourage',
    keywords: ['很好', '不错', '真棒', '做得很好', '继续加油', '很有进步'],
  },
  {
    gesture: 'enumerate',
    keywords: [
      '第一点',
      '第二点',
      '第三点',
      '第一',
      '第二',
      '第三',
      '首先',
      '其次',
      '最后一点',
      '分为',
      '几点',
    ],
  },
  {
    gesture: 'point_left',
    keywords: ['看左边', '左侧', '看课件', '课件上', '屏幕上', '图中', '图片中', '看这里'],
  },
  {
    gesture: 'invite',
    keywords: ['想一想', '试试看', '试一试', '你觉得', '猜一猜', '一起观察', '你来看看', '请思考'],
  },
  {
    gesture: 'emphasis',
    keywords: ['重点', '关键', '注意', '核心', '特别重要', '一定要', '记住'],
  },
  {
    gesture: 'expand',
    keywords: ['例如', '比如', '举个例子', '具体来说', '换句话说', '展开来说', '我们来看', '可以理解为'],
  },
]

function normalizeGestureText(text: string): string {
  return text.replace(/\s+/g, '').toLowerCase()
}

function maximumGestureCount(textLength: number): number {
  if (textLength < 50) return 0
  if (textLength < 120) return 1
  return 2
}

function clampProgress(value: number): number {
  return Math.max(0.10, Math.min(0.82, value))
}

function firstKeywordIndex(text: string, keywords: readonly string[]): number {
  let result = -1

  for (const keyword of keywords) {
    const index = text.indexOf(keyword.toLowerCase())

    if (index >= 0 && (result < 0 || index < result)) {
      result = index
    }
  }

  return result
}

function buildSemanticCandidates(speechText: string): GestureCandidate[] {
  const normalized = normalizeGestureText(speechText)
  const textLength = Array.from(normalized).length

  if (!normalized || textLength <= 0) return []

  const candidates: GestureCandidate[] = []

  for (const rule of SEMANTIC_RULES) {
    const textIndex = firstKeywordIndex(normalized, rule.keywords)
    if (textIndex < 0) continue

    const rawProgress = textIndex / Math.max(1, normalized.length)
    const speechAlignedProgress = rawProgress * 0.94

    if (rule.gesture === 'applause' && rawProgress < 0.48) {
      candidates.push({
        gesture: 'encourage',
        textIndex,
        targetProgress: clampProgress(speechAlignedProgress),
      })
      continue
    }

    candidates.push({
      gesture: rule.gesture,
      textIndex,
      targetProgress: clampProgress(speechAlignedProgress),
    })
  }

  return candidates.sort((first, second) => first.textIndex - second.textIndex)
}

function fallbackGesture(
  character: ClassroomDigitalHumanCharacter,
): ClassroomDigitalHumanGesture {
  return character === 'female' ? 'expand' : 'expand'
}

function selectCandidates(
  character: ClassroomDigitalHumanCharacter,
  speechText: string,
): GestureCandidate[] {
  const normalized = normalizeGestureText(speechText)
  const textLength = Array.from(normalized).length
  const maximum = maximumGestureCount(textLength)

  if (maximum === 0) return []

  const semantic = buildSemanticCandidates(normalized)

  if (semantic.length === 0) {
    if (textLength < 95) return []

    return [{
      gesture: fallbackGesture(character),
      textIndex: Math.floor(textLength * 0.28),
      targetProgress: 0.28,
    }]
  }

  const selected: GestureCandidate[] = []

  for (const candidate of semantic) {
    if (selected.some(current => current.gesture === candidate.gesture)) continue

    const previous = selected[selected.length - 1]
    if (previous && candidate.targetProgress - previous.targetProgress < 0.14) continue

    selected.push(candidate)
    if (selected.length >= maximum) break
  }

  if (
    maximum >= 2
    && selected.length > 0
    && selected[0].targetProgress > 0.36
  ) {
    const earlyGesture = fallbackGesture(character)

    if (!selected.some(candidate => candidate.gesture === earlyGesture)) {
      selected.unshift({
        gesture: earlyGesture,
        textIndex: Math.floor(textLength * 0.24),
        targetProgress: 0.24,
      })
    }
  }

  return selected
    .slice(0, maximum)
    .sort((first, second) => first.targetProgress - second.targetProgress)
}

function gestureDuration(gesture: ClassroomDigitalHumanGesture): number {
  switch (gesture) {
  case 'applause':
    return 1350
  case 'encourage':
    return 1250
  case 'point_left':
    return 1550
  case 'enumerate':
    return 1450
  default:
    return 1350
  }
}

function fallbackDelay(candidate: GestureCandidate, index: number): number {
  if (index === 0) {
    return 3000 + Math.round(candidate.targetProgress * 1800)
  }

  return 6200
}

export function useClassroomDigitalHumanGesture({
  state,
  character,
  speechText,
  speechPaused,
  audioElement,
}: UseClassroomDigitalHumanGestureOptions): ClassroomDigitalHumanGesture | null {
  const [gesture, setGesture] = useState<ClassroomDigitalHumanGesture | null>(null)

  useEffect(() => {
    setGesture(null)

    if (state !== 'speaking' || speechPaused) return

    const normalizedText = speechText.trim()
    const candidates = selectCandidates(character, normalizedText)

    if (!normalizedText || candidates.length === 0) return

    let cancelled = false
    let gestureActive = false
    let cursor = 0
    let progressTimer = 0
    let endGestureTimer = 0
    let fallbackTimer = 0
    let fallbackFollowupTimer = 0

    const trigger = (candidate: GestureCandidate) => {
      if (cancelled || gestureActive) return

      gestureActive = true
      setGesture(candidate.gesture)

      window.clearTimeout(endGestureTimer)
      endGestureTimer = window.setTimeout(() => {
        if (cancelled) return

        gestureActive = false
        setGesture(null)
      }, gestureDuration(candidate.gesture))
    }

    if (audioElement) {
      progressTimer = window.setInterval(() => {
        if (cancelled || gestureActive || cursor >= candidates.length) return
        if (!Number.isFinite(audioElement.duration) || audioElement.duration <= 0.5) return

        const progress = audioElement.currentTime / audioElement.duration
        const candidate = candidates[cursor]

        if (progress >= candidate.targetProgress) {
          cursor += 1
          trigger(candidate)
        }
      }, 100)

      return () => {
        cancelled = true
        window.clearInterval(progressTimer)
        window.clearTimeout(endGestureTimer)
        window.clearTimeout(fallbackTimer)
        window.clearTimeout(fallbackFollowupTimer)
        setGesture(null)
      }
    }

    /**
     * 浏览器语音兜底没有HTMLAudioElement，只能按时间近似。
     * 第一动作比旧版提前，但仍限制动作数量，避免重新变成频繁切换。
     */
    const scheduleFallback = (index: number) => {
      if (cancelled || index >= candidates.length) return

      const candidate = candidates[index]
      fallbackTimer = window.setTimeout(() => {
        if (cancelled) return

        trigger(candidate)

        fallbackFollowupTimer = window.setTimeout(() => {
          if (!cancelled) scheduleFallback(index + 1)
        }, gestureDuration(candidate.gesture) + 360)
      }, fallbackDelay(candidate, index))
    }

    scheduleFallback(0)

    return () => {
      cancelled = true
      window.clearInterval(progressTimer)
      window.clearTimeout(endGestureTimer)
      window.clearTimeout(fallbackTimer)
      window.clearTimeout(fallbackFollowupTimer)
      setGesture(null)
    }
  }, [
    audioElement,
    character,
    speechPaused,
    speechText,
    state,
  ])

  return gesture
}

export default useClassroomDigitalHumanGesture


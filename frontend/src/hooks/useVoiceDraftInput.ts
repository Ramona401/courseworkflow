/**
 * useVoiceDraftInput.ts — 受保护文字草稿的统一语音输入适配层
 *
 * 适用范围：
 * 1. AI对话消息；
 * 2. AI审核讨论；
 * 3. AI微调和重构指令；
 * 4. 其它由父组件控制的文字输入。
 *
 * 本Hook在useVoiceInput之上统一实现：
 * - 录音开始时保存当前文字；
 * - partial结果覆盖本次语音片段，不重复累加；
 * - final结果写回最终文字；
 * - 识别失败恢复录音前文字；
 * - 最终文字只进入输入框，不自动提交；
 * - 可选聚焦回调把光标放回输入框末尾。
 *
 * 本Hook不创建草稿存储。调用方应继续使用现有useProtectedDraft，
 * 并把其value和setValue传入本Hook。
 */

import {
  useCallback,
  useMemo,
  useRef,
} from 'react'

import {
  useVoiceInput,
} from '@/hooks/useVoiceInput'

import type {
  UseVoiceInputResult,
} from '@/hooks/useVoiceInput'

export interface UseVoiceDraftInputOptions {
  /** 当前受控输入值。 */
  value: string

  /**
   * useProtectedDraft.setValue或普通受控更新函数。
   *
   * 本Hook只会传入完整字符串，不会把函数式更新器传给调用方，
   * 因此也兼容(value: string) => void类型的父组件回调。
   */
  setValue: (
    value: string,
  ) => void

  /** 当前业务是否禁止开始语音。 */
  disabled?: boolean

  /** 单次录音上限，默认120秒。 */
  maxDurationSeconds?: number

  /** 最终文字写入后可选聚焦回调。 */
  onFinalFocus?: (
    finalValue: string,
  ) => void

  /** 语音失败时的额外业务提示。 */
  onError?: (
    message: string,
  ) => void
}

export interface UseVoiceDraftInputResult
  extends UseVoiceInputResult {
  /** 从当前输入值开始一轮语音输入。 */
  begin: () => void

  /** 可直接展示在输入框附近的状态文案。 */
  statusText: string
}

/**
 * 把当前识别文字追加到录音开始前的原草稿。
 *
 * 中文之间不强插空格；
 * 连续英文或数字之间补一个空格，避免单词粘连。
 */
export function mergeVoiceDraftText(
  base: string,
  speech: string,
): string {
  const normalizedSpeech =
    speech.trim()

  if (!normalizedSpeech) {
    return base
  }

  if (!base) {
    return normalizedSpeech
  }

  const needsSpace =
    /[A-Za-z0-9]$/.test(base) &&
    /^[A-Za-z0-9]/.test(
      normalizedSpeech,
    )

  return (
    base +
    (needsSpace ? ' ' : '') +
    normalizedSpeech
  )
}

export function useVoiceDraftInput({
  value,
  setValue,
  disabled = false,
  maxDurationSeconds = 120,
  onFinalFocus,
  onError,
}: UseVoiceDraftInputOptions): UseVoiceDraftInputResult {
  /**
   * 每轮录音开始前的输入值。
   *
   * partial和final都基于这份值重新合并，
   * 避免“你好”→“你好世界”被累加成“你好你好世界”。
   */
  const baseValueRef =
    useRef('')

  const handlePartial =
    useCallback(
      (
        text: string,
      ) => {
        setValue(
          mergeVoiceDraftText(
            baseValueRef.current,
            text,
          ),
        )
      },
      [setValue],
    )

  const handleFinal =
    useCallback(
      (
        text: string,
      ) => {
        const finalValue =
          mergeVoiceDraftText(
            baseValueRef.current,
            text,
          )

        setValue(finalValue)

        if (onFinalFocus) {
          requestAnimationFrame(
            () => {
              onFinalFocus(
                finalValue,
              )
            },
          )
        }
      },
      [
        onFinalFocus,
        setValue,
      ],
    )

  const handleError =
    useCallback(
      (
        message: string,
      ) => {
        /**
         * partial结果尚未经过最终确认。
         * 失败时恢复录音前文字，避免临时识别结果污染草稿。
         */
        setValue(
          baseValueRef.current,
        )

        onError?.(message)
      },
      [
        onError,
        setValue,
      ],
    )

  const voice =
    useVoiceInput({
      disabled,
      maxDurationSeconds,
      onPartial:
        handlePartial,
      onFinal:
        handleFinal,
      onError:
        handleError,
    })

  const begin =
    useCallback(() => {
      baseValueRef.current =
        value

      void voice.start()
    }, [
      value,
      voice.start,
    ])

  const statusText =
    useMemo(() => {
      switch (voice.status) {
        case 'connecting':
          return '正在连接语音识别…'

        case 'recording': {
          const minutes =
            Math.floor(
              voice.elapsedSeconds /
              60,
            )

          const seconds =
            String(
              voice.elapsedSeconds %
              60,
            ).padStart(
              2,
              '0',
            )

          return (
            `正在听写 ${minutes}:${seconds}` +
            ' · 点击红色按钮停止'
          )
        }

        case 'stopping':
          return '正在整理最终文字…'

        case 'error':
          return voice.error
            ? `语音输入未完成：${voice.error}`
            : '语音输入未完成'

        default:
          return ''
      }
    }, [
      voice.elapsedSeconds,
      voice.error,
      voice.status,
    ])

  return {
    ...voice,
    begin,
    statusText,
  }
}

export default useVoiceDraftInput

/**
 * SchemeSteps.tsx
 *
 * 课件工坊Step0（AI生成方案）和Step1（确认方案）。
 *
 * 【页面拖拽排序】
 * - 方案已经稳定生成后，教师可在IndexEditor中拖动页面左侧手柄调整顺序。
 * - AI仍在生成页面时禁止排序，避免SSE持续追加页面与人工排序发生竞争。
 * - AI正在整体修改方案或正在确认方案时禁止排序，避免并发提交两个页面顺序。
 *
 * 【AI自动修正方案】
 * AlignmentReportCard通过onAutoFix回调把对齐报告问题转换为修正指令，
 * 本组件复用refineCWIndex和SSE执行修改。
 * 修改完成后自动触发对齐重新校验，形成“诊断→修正→复查”闭环。
 *
 * autoFixMessage独立于sseMessage，
 * 不会被loadCourseware覆盖，可持续显示AI修正结果。
 */

import {
  useState,
  useEffect,
  useRef,
} from 'react'
import { useAuth } from '@/store/auth'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import { useVoiceDraftInput } from '@/hooks/useVoiceDraftInput'
import VoiceInputButton from '@/components/voice/VoiceInputButton'
import type {
  Dispatch,
  SetStateAction,
  MutableRefObject,
} from 'react'
import {
  getCourseware,
  generateCWIndex,
  generateCWIndexFromTopic,
  generateCWIndexFromPPT,
  generateCWIndexFromDoc,
  subscribeCWIndexSSE,
  confirmCWIndex,
  refineCWIndex,
  getSchemePresets,
  recheckAlignment,
} from '@/api/coursewares'
import type {
  SchemePreset,
  CoursewareDetail,
  CoursewarePage,
} from '@/api/coursewares'
import IndexEditor from '../IndexEditor'
import AlignmentReportCard from './AlignmentReportCard'
import { C } from './workshopConstants'
import { MsgBar } from './PagePreviewBlock'
import CustomSchemeBuilder from './CustomSchemeBuilder'

interface Props {
  coursewareId: string
  courseware: CoursewareDetail
  isAdmin: boolean
  activeStep: number
  pages: CoursewarePage[]
  setPages: Dispatch<
    SetStateAction<CoursewarePage[]>
  >
  sseRef: MutableRefObject<{
    close: () => void
  } | null>
  goToStep: (n: number) => void
  loadCourseware: () => void
  onCoursewareUpdate: (
    detail: CoursewareDetail,
  ) => void
}

export default function SchemeSteps({
  coursewareId,
  courseware,
  isAdmin,
  activeStep,
  pages,
  setPages,
  sseRef,
  goToStep,
  loadCourseware,
  onCoursewareUpdate,
}: Props) {
  const { user } = useAuth()

  const [generating, setGenerating] =
    useState(false)

  const [sseMessage, setSseMessage] =
    useState('')

  const [confirming, setConfirming] =
    useState(false)

  const [presets, setPresets] =
    useState<SchemePreset[]>([])

  /**
   * 课件方案入口草稿按当前用户和课件ID隔离。
   *
   * 结构预设和自定义结构说明会保留，
   * 便于返回本步骤后继续生成。
   *
   * 整体方案修改意见仅在AI成功完成修改后提交清空。
   */
  const presetDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'courseware-scheme',
    resourceId: coursewareId,
    field: 'preset',
    initialValue: 'auto',
    maxHistory: 12,
  })

  const selectedPreset =
    presetDraft.value || 'auto'

  const setSelectedPreset =
    presetDraft.setValue

  const customPromptDraft =
    useProtectedDraft({
      userId: user?.id,
      scope: 'courseware-scheme',
      resourceId: coursewareId,
      field: 'custom-prompt',
      initialValue: '',
      maxHistory: 30,
    })

  const customPromptHint =
    customPromptDraft.value

  const setCustomPromptHint =
    customPromptDraft.setValue

  const refineFeedbackDraft =
    useProtectedDraft({
      userId: user?.id,
      scope: 'courseware-scheme',
      resourceId: coursewareId,
      field: 'refine-feedback',
      initialValue: '',
      maxHistory: 40,
    })

  const refineFeedback =
    refineFeedbackDraft.value

  const setRefineFeedback =
    refineFeedbackDraft.setValue

  const [refining, setRefining] =
    useState(false)

  /**
   * AI自动修正后自动重新校验的标记。
   *
   * 每次递增都会让AlignmentReportCard重新挂载并拉取最新报告。
   */
  const [
    alignmentRecheckKey,
    setAlignmentRecheckKey,
  ] = useState(0)

  /**
   * AI自动修正的持久提示。
   *
   * 独立于SSE提示，避免loadCourseware刷新后成功信息消失。
   */
  const [
    autoFixMessage,
    setAutoFixMessage,
  ] = useState<{
    text: string
    type: 'success' | 'info'
  } | null>(null)

  const refineFeedbackInputRef =
    useRef<HTMLInputElement>(
      null,
    )

  /**
   * 自定义课件结构说明使用现有受保护草稿。
   * 语音结果只补充结构说明，不会自动启动方案生成。
   */
  const customPromptVoice =
    useVoiceDraftInput({
      value:
        customPromptHint,
      setValue:
        setCustomPromptHint,
      disabled:
        generating
        || activeStep !== 0
        || selectedPreset !== 'custom',
      maxDurationSeconds: 120,
      onError: (
        voiceError,
      ) => {
        setSseMessage(
          '❌ 语音输入失败: '
          + voiceError,
        )
      },
    })

  /**
   * 整体方案修改意见使用独立语音会话。
   * final只写入输入框，不自动触发AI修改。
   */
  const refineFeedbackVoice =
    useVoiceDraftInput({
      value:
        refineFeedback,
      setValue:
        setRefineFeedback,
      disabled:
        refining
        || confirming
        || activeStep !== 1,
      maxDurationSeconds: 120,
      onFinalFocus: (
        finalValue,
      ) => {
        const element =
          refineFeedbackInputRef.current

        if (!element) {
          return
        }

        element.focus()
        element.setSelectionRange(
          finalValue.length,
          finalValue.length,
        )
      },
      onError: (
        voiceError,
      ) => {
        setSseMessage(
          '❌ 语音输入失败: '
          + voiceError,
        )
      },
    })

  useEffect(() => {
    /**
     * 步骤切换后隐藏的语音入口必须立即释放麦克风。
     */
    if (
      activeStep !== 0
      && customPromptVoice.isActive
    ) {
      customPromptVoice.cancel()
    }

    if (
      activeStep !== 1
      && refineFeedbackVoice.isActive
    ) {
      refineFeedbackVoice.cancel()
    }
  }, [
    activeStep,
    customPromptVoice.cancel,
    customPromptVoice.isActive,
    refineFeedbackVoice.cancel,
    refineFeedbackVoice.isActive,
  ])

  useEffect(() => {
    getSchemePresets()
      .then(result =>
        setPresets(result),
      )
      .catch(() => {})
  }, [])

  /**
   * 已缓存预设下线时安全回退自动方案。
   */
  useEffect(() => {
    if (presets.length === 0) {
      return
    }

    if (
      presets.some(
        preset =>
          preset.key ===
          selectedPreset,
      )
    ) {
      return
    }

    setSelectedPreset('auto')
  }, [
    presets,
    selectedPreset,
    setSelectedPreset,
  ])

  /**
   * 生成中的10秒兜底轮询。
   *
   * SSE漏收完成事件时，通过课件正式状态恢复界面。
   */
  useEffect(() => {
    if (
      !generating ||
      !coursewareId
    ) {
      return
    }

    const timer = setInterval(
      async () => {
        try {
          const detail =
            await getCourseware(
              coursewareId,
            )

          if (
            detail.status !== 'draft' &&
            detail.status !== 'indexing'
          ) {
            setGenerating(false)
            onCoursewareUpdate(detail)
            setPages(
              detail.pages || [],
            )
            goToStep(1)
            setSseMessage('✅ 完成')
            sseRef.current?.close()
          }
        } catch {
          /**
           * 轮询是SSE兜底。
           * 单次失败不终止后续轮询。
           */
        }
      },
      10000,
    )

    return () =>
      clearInterval(timer)
  }, [
    generating,
    coursewareId,
    goToStep,
    onCoursewareUpdate,
    setPages,
    sseRef,
  ])

  // ==================== Step 0：生成方案 ====================

  /**
   * 按课件来源类型调用对应方案生成端点。
   */
  const handleGenerate = async () => {
    if (
      !coursewareId
      || customPromptVoice.isActive
    ) {
      return
    }

    setGenerating(true)
    setSseMessage('正在启动...')
    setPages([])
    setAutoFixMessage(null)

    try {
      if (
        courseware.source_type ===
        'topic_direct'
      ) {
        await generateCWIndexFromTopic(
          coursewareId,
          {
            subject:
              courseware.subject,
            grade:
              courseware.grade,
            topic:
              courseware.title,
            preset:
              selectedPreset,
            custom_prompt_hint:
              selectedPreset ===
              'custom'
                ? customPromptHint
                : undefined,
          },
        )
      } else if (
        courseware.source_type ===
        'ppt_upload'
      ) {
        await generateCWIndexFromPPT(
          coursewareId,
          selectedPreset,
          selectedPreset === 'custom'
            ? customPromptHint
            : undefined,
        )
      } else if (
        courseware.source_type ===
        'doc_upload'
      ) {
        await generateCWIndexFromDoc(
          coursewareId,
          selectedPreset,
          selectedPreset === 'custom'
            ? customPromptHint
            : undefined,
        )
      } else {
        await generateCWIndex(
          coursewareId,
          selectedPreset,
          selectedPreset === 'custom'
            ? customPromptHint
            : undefined,
        )
      }

      sseRef.current?.close()

      sseRef.current =
        subscribeCWIndexSSE(
          coursewareId,
          {
            onConnected: () =>
              setSseMessage(
                '已连接，正在分析教案...',
              ),

            onIndexStart: data =>
              setSseMessage(
                String(
                  (
                    data as Record<
                      string,
                      unknown
                    >
                  ).message || '',
                ),
              ),

            onIndexProgress:
              data =>
                setSseMessage(
                  String(
                    (
                      data as Record<
                        string,
                        unknown
                      >
                    ).message || '',
                  ),
                ),

            onIndexPage: page =>
              setPages(previous => {
                const next =
                  previous.some(
                    item =>
                      item.page_number ===
                      page.page_number,
                  )
                    ? previous.map(
                        item =>
                          item.page_number ===
                          page.page_number
                            ? page
                            : item,
                      )
                    : [
                        ...previous,
                        page,
                      ]

                return next
                  .slice()
                  .sort(
                    (left, right) =>
                      left.page_number -
                      right.page_number,
                  )
              }),

            onIndexDone: data => {
              setSseMessage(
                `✅ ${data.message}`,
              )
              setGenerating(false)
              goToStep(1)
              loadCourseware()
            },

            onError: data => {
              setSseMessage(
                `❌ ${data.message}`,
              )
              setGenerating(false)
            },
          },
        )
    } catch {
      setSseMessage(
        '❌ 启动失败',
      )
      setGenerating(false)
    }
  }

  // ==================== Step 1：确认方案 ====================

  const handleConfirm = async () => {
    if (
      !coursewareId
      || !pages.length
      || refineFeedbackVoice.isActive
    ) {
      return
    }

    setConfirming(true)

    try {
      await confirmCWIndex(
        coursewareId,
      )
      goToStep(2)
      loadCourseware()
    } catch {
      alert('确认失败')
    } finally {
      setConfirming(false)
    }
  }

  /**
   * AI修改方案通用入口。
   *
   * @param instruction 修改指令。
   * @param autoRecheck 修改完成后是否自动重新校验对齐度。
   */
  const doRefineIndex = async (
    instruction: string,
    autoRecheck = false,
  ) => {
    if (
      !coursewareId ||
      !instruction.trim() ||
      refining ||
      refineFeedbackVoice.isActive
    ) {
      return
    }

    setRefining(true)
    setSseMessage(
      '🔧 正在根据意见修改方案...',
    )

    setAutoFixMessage(
      autoRecheck
        ? {
            text: '🔧 AI 正在修正方案，请稍候…',
            type: 'info',
          }
        : null,
    )

    try {
      await refineCWIndex(
        coursewareId,
        instruction.trim(),
      )

      sseRef.current?.close()

      sseRef.current =
        subscribeCWIndexSSE(
          coursewareId,
          {
            onConnected: () =>
              setSseMessage(
                '已连接，AI正在修改方案...',
              ),

            onIndexStart: data =>
              setSseMessage(
                String(
                  (
                    data as Record<
                      string,
                      unknown
                    >
                  ).message || '',
                ),
              ),

            onIndexProgress:
              data =>
                setSseMessage(
                  String(
                    (
                      data as Record<
                        string,
                        unknown
                      >
                    ).message || '',
                  ),
                ),

            onIndexPage: page =>
              setPages(previous => {
                const next =
                  previous.some(
                    item =>
                      item.page_number ===
                      page.page_number,
                  )
                    ? previous.map(
                        item =>
                          item.page_number ===
                          page.page_number
                            ? page
                            : item,
                      )
                    : [
                        ...previous,
                        page,
                      ]

                return next
                  .slice()
                  .sort(
                    (left, right) =>
                      left.page_number -
                      right.page_number,
                  )
              }),

            onIndexDone: data => {
              setSseMessage(
                `✅ ${data.message}`,
              )
              setRefining(false)

              /**
               * 自动对齐修正不消费教师手工输入框。
               * 手工修改成功后才提交并清除草稿。
               */
              if (!autoRecheck) {
                refineFeedbackDraft.commit()
              }

              loadCourseware()

              if (autoRecheck) {
                setAutoFixMessage({
                  text: '✅ 方案已修正完成！正在重新校验对齐度…',
                  type: 'success',
                })

                recheckAlignment(
                  coursewareId,
                )
                  .then(() => {
                    setAlignmentRecheckKey(
                      previous =>
                        previous + 1,
                    )

                    /**
                     * 给报告卡片轮询留出启动时间。
                     */
                    setTimeout(() => {
                      setAutoFixMessage({
                        text: '✅ 方案修正完成，对齐报告已更新。请查看上方报告了解改善情况。',
                        type: 'success',
                      })
                    }, 2000)
                  })
                  .catch(() => {
                    setAutoFixMessage({
                      text: '✅ 方案已修正完成，但重新校验未能启动。可点击报告卡片的“🔄 重新校验”手动触发。',
                      type: 'success',
                    })
                  })
              }
            },

            onError: data => {
              setSseMessage(
                `❌ ${data.message}`,
              )
              setRefining(false)

              if (autoRecheck) {
                setAutoFixMessage({
                  text:
                    '❌ 方案修正失败：' +
                    data.message,
                  type: 'info',
                })
              }
            },
          },
        )
    } catch {
      setSseMessage(
        '❌ 启动失败',
      )
      setRefining(false)

      if (autoRecheck) {
        setAutoFixMessage({
          text: '❌ 方案修正启动失败，请稍后重试',
          type: 'info',
        })
      }
    }
  }

  /**
   * 手动输入修改意见，不自动重新校验。
   */
  const handleRefineIndex = () =>
    doRefineIndex(
      refineFeedback,
    )

  /**
   * 对齐报告自动修正，完成后自动重新校验。
   */
  const handleAutoFix = (
    instruction: string,
  ) =>
    doRefineIndex(
      instruction,
      true,
    )

  return (
    <>
      {/* Step 0：AI生成方案 */}
      {activeStep === 0 && (
        <div>
          <h3
            style={{
              fontSize: 18,
              fontWeight: 600,
              color: C.textPrimary,
              margin: '0 0 8px',
            }}
          >
            🤖 AI生成课件方案
          </h3>

          <p
            style={{
              fontSize: 14,
              color: C.textSecondary,
              margin: '0 0 20px',
            }}
          >
            AI将分析教案内容，自动为每页设计方案。
          </p>

          {presets.length > 0 &&
            !generating && (
              <div
                style={{
                  marginBottom: 20,
                }}
              >
                <div
                  style={{
                    fontSize: 14,
                    fontWeight: 600,
                    color:
                      C.textPrimary,
                    marginBottom: 10,
                  }}
                >
                  选择课件结构预设
                </div>

                <div
                  style={{
                    display: 'flex',
                    gap: 10,
                    flexWrap: 'wrap',
                  }}
                >
                  {presets.map(
                    preset => (
                      <button
                        key={preset.key}
                        onClick={() =>
                          setSelectedPreset(
                            preset.key,
                          )
                        }
                        disabled={
                          customPromptVoice.isActive
                        }
                        style={{
                          flex:
                            '1 1 200px',
                          maxWidth: 240,
                          padding:
                            '12px 16px',
                          borderRadius:
                            10,
                          cursor:
                            customPromptVoice.isActive
                              ? 'not-allowed'
                              : 'pointer',
                          opacity:
                            customPromptVoice.isActive
                              ? 0.65
                              : 1,
                          border: `2px solid ${
                            selectedPreset ===
                            preset.key
                              ? C.primary
                              : C.border
                          }`,
                          background:
                            selectedPreset ===
                            preset.key
                              ? C.primaryBg
                              : C.white,
                          textAlign:
                            'left',
                          transition:
                            'all 200ms',
                        }}
                      >
                        <div
                          style={{
                            fontSize: 20,
                            marginBottom: 4,
                          }}
                        >
                          {preset.emoji}
                        </div>

                        <div
                          style={{
                            fontSize: 14,
                            fontWeight: 600,
                            color:
                              selectedPreset ===
                              preset.key
                                ? C.primary
                                : C.textPrimary,
                          }}
                        >
                          {preset.name}
                        </div>

                        <div
                          style={{
                            fontSize: 12,
                            color:
                              C.textSecondary,
                            marginTop: 2,
                          }}
                        >
                          {
                            preset.description
                          }
                        </div>

                        <div
                          style={{
                            fontSize: 11,
                            color:
                              C.textMuted,
                            marginTop: 4,
                          }}
                        >
                          {
                            preset.page_range
                          }
                        </div>
                      </button>
                    ),
                  )}
                </div>
              </div>
            )}

          {/* 自定义预设 */}
          {selectedPreset ===
            'custom' &&
            !generating && (
              <div
                onKeyDown={event => {
                  customPromptDraft.handleKeyDown(
                    event,
                  )
                }}
              >
                <CustomSchemeBuilder
                  value={
                    customPromptHint
                  }
                  onChange={
                    setCustomPromptHint
                  }
                />

                <div
                  style={{
                    marginTop: 8,
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                  }}
                >
                  <span
                    style={{
                      flex: 1,
                      fontSize: 11,
                      color:
                        customPromptVoice.status === 'error'
                          ? '#DC2626'
                          : customPromptVoice.isActive
                            ? '#7C3AED'
                            : C.textMuted,
                      lineHeight: 1.5,
                    }}
                  >
                    {customPromptVoice.statusText
                      || '已自动保存自定义结构 · 点击麦克风可语音补充 · Ctrl/Command+Z恢复误删'}
                  </span>

                  <VoiceInputButton
                    status={
                      customPromptVoice.status
                    }
                    isSupported={
                      customPromptVoice.isSupported
                    }
                    elapsedSeconds={
                      customPromptVoice.elapsedSeconds
                    }
                    disabled={
                      generating
                      || selectedPreset !== 'custom'
                    }
                    error={
                      customPromptVoice.error
                    }
                    onStart={() => {
                      const activeElement =
                        document.activeElement

                      if (
                        activeElement instanceof
                        HTMLElement
                      ) {
                        activeElement.blur()
                      }

                      customPromptVoice.begin()
                    }}
                    onStop={
                      customPromptVoice.stop
                    }
                    onCancel={
                      customPromptVoice.cancel
                    }
                  />
                </div>
              </div>
            )}

          <MsgBar msg={sseMessage} />

          {generating &&
            pages.length > 0 && (
              <div
                style={{
                  marginBottom: 16,
                }}
              >
                <div
                  style={{
                    fontSize: 13,
                    color: C.textMuted,
                    marginBottom: 8,
                  }}
                >
                  已生成 {pages.length}{' '}
                  页方案...
                </div>

                <IndexEditor
                  coursewareId={
                    coursewareId
                  }
                  pages={pages}
                  onPagesChange={
                    setPages
                  }
                  isAdmin={isAdmin}
                  indexOverview={
                    courseware.index_overview
                  }
                  /**
                   * SSE仍可能继续追加、替换页面，
                   * 此时不允许人工排序。
                   */
                  reorderDisabled={
                    generating
                  }
                />
              </div>
            )}

          <button
            onClick={handleGenerate}
            disabled={
              generating ||
              customPromptVoice.isActive ||
              (selectedPreset ===
                'custom' &&
                !customPromptHint.trim())
            }
            style={{
              padding: '12px 32px',
              borderRadius: 10,
              border: 'none',
              background:
                generating ||
                customPromptVoice.isActive
                  ? '#E5E7EB'
                : 'linear-gradient(135deg, #F59E0B, #EF4444)',
              color:
                generating ||
                customPromptVoice.isActive
                  ? '#9CA3AF'
                : '#fff',
              fontSize: 15,
              fontWeight: 600,
              cursor:
                generating ||
                customPromptVoice.isActive
                  ? 'default'
                : 'pointer',
              boxShadow:
                generating ||
                customPromptVoice.isActive
                  ? 'none'
                : '0 4px 16px rgba(245,158,11,0.3)',
            }}
          >
            {generating
              ? '⏳ 生成中...'
              : pages.length > 0
                ? '🔄 重新生成'
                : '🤖 开始AI生成方案'}
          </button>

          {!generating &&
            pages.length > 0 && (
              <button
                onClick={() =>
                  goToStep(1)
                }
                disabled={
                  customPromptVoice.isActive
                }
                style={{
                  marginLeft: 12,
                  padding:
                    '12px 24px',
                  borderRadius: 10,
                  border: `1px solid ${C.primary}`,
                  background:
                    C.primaryBg,
                  color: C.primary,
                  fontSize: 15,
                  fontWeight: 600,
                  cursor:
                    customPromptVoice.isActive
                      ? 'not-allowed'
                      : 'pointer',
                  opacity:
                    customPromptVoice.isActive
                      ? 0.6
                      : 1,
                }}
              >
                ✏️ 确认方案 →
              </button>
            )}
        </div>
      )}

      {/* Step 1：确认方案 */}
      {activeStep === 1 && (
        <div>
          <div
            style={{
              display: 'flex',
              justifyContent:
                'space-between',
              alignItems: 'center',
              gap: 12,
              marginBottom: 16,
              flexWrap: 'wrap',
            }}
          >
            <div>
              <h3
                style={{
                  fontSize: 18,
                  fontWeight: 600,
                  color:
                    C.textPrimary,
                  margin: 0,
                }}
              >
                ✏️ 确认方案
              </h3>

              <p
                style={{
                  fontSize: 13,
                  color:
                    C.textSecondary,
                  margin: '4px 0 0',
                }}
              >
                确认每页内容，可拖动页面左侧手柄调整顺序，也可修改页面细节
              </p>
            </div>

            <div
              style={{
                display: 'flex',
                gap: 10,
              }}
            >
              <button
                onClick={() =>
                  goToStep(0)
                }
                disabled={
                  refining ||
                  confirming ||
                  refineFeedbackVoice.isActive
                }
                style={{
                  padding:
                    '8px 16px',
                  borderRadius: 8,
                  border: `1px solid ${C.border}`,
                  background:
                    'transparent',
                  color:
                    C.textSecondary,
                  fontSize: 13,
                  cursor:
                    refining ||
                    confirming
                      ? 'default'
                      : 'pointer',
                  opacity:
                    refining ||
                    confirming
                      ? 0.6
                      : 1,
                }}
              >
                ← 重新生成
              </button>

              <button
                onClick={
                  handleConfirm
                }
                disabled={
                  confirming ||
                  refining ||
                  refineFeedbackVoice.isActive ||
                  !pages.length
                }
                style={{
                  padding:
                    '8px 20px',
                  borderRadius: 8,
                  border: 'none',
                  background:
                    pages.length &&
                    !refining
                      ? 'linear-gradient(135deg, #F59E0B, #EF4444)'
                      : '#E5E7EB',
                  color:
                    pages.length &&
                    !refining
                      ? '#fff'
                      : '#9CA3AF',
                  fontSize: 14,
                  fontWeight: 600,
                  cursor:
                    pages.length &&
                    !confirming &&
                    !refining
                      ? 'pointer'
                      : 'default',
                }}
              >
                {confirming
                  ? '确认中...'
                  : '确认方案，选择风格 →'}
              </button>
            </div>
          </div>

          {/* AI自动修正持久提示 */}
          {autoFixMessage && (
            <div
              style={{
                marginBottom: 12,
                padding:
                  '10px 16px',
                borderRadius: 10,
                fontSize: 13,
                lineHeight: 1.6,
                display: 'flex',
                alignItems:
                  'center',
                justifyContent:
                  'space-between',
                gap: 8,
                background:
                  autoFixMessage.type ===
                  'success'
                    ? '#F0FDF4'
                    : '#EFF6FF',
                border: `1.5px solid ${
                  autoFixMessage.type ===
                  'success'
                    ? '#86EFAC'
                    : '#93C5FD'
                }`,
                color:
                  autoFixMessage.type ===
                  'success'
                    ? '#166534'
                    : '#1E40AF',
              }}
            >
              <span>
                {
                  autoFixMessage.text
                }
              </span>

              <button
                onClick={() =>
                  setAutoFixMessage(
                    null,
                  )
                }
                style={{
                  background: 'none',
                  border: 'none',
                  fontSize: 16,
                  cursor: 'pointer',
                  color: 'inherit',
                  padding: '0 4px',
                  opacity: 0.6,
                }}
              >
                ✕
              </button>
            </div>
          )}

          {/* 课件与教案对齐报告 */}
          <AlignmentReportCard
            key={
              alignmentRecheckKey
            }
            coursewareId={
              coursewareId
            }
            sourceType={
              courseware.source_type
            }
            onAutoFix={
              handleAutoFix
            }
          />

          <IndexEditor
            coursewareId={
              coursewareId
            }
            pages={pages}
            onPagesChange={
              setPages
            }
            isAdmin={isAdmin}
            indexOverview={
              courseware.index_overview
            }
            /**
             * AI整体改方案和确认状态切换期间，
             * 禁止再提交页面排序。
             */
            reorderDisabled={
              refining ||
              confirming ||
              refineFeedbackVoice.isActive
            }
          />

          {/* AI修改方案输入区 */}
          {pages.length > 0 &&
            !refining && (
              <div
                style={{
                  marginTop: 16,
                  padding: '16px',
                  borderRadius: 10,
                  border:
                    '1px solid ' +
                    C.border,
                  background:
                    '#FAFAFA',
                }}
              >
                <div
                  style={{
                    fontSize: 13,
                    fontWeight: 600,
                    color:
                      C.textPrimary,
                    marginBottom: 8,
                  }}
                >
                  🤖
                  对整体方案不满意？输入修改意见让AI重新调整
                </div>

                <div
                  style={{
                    display: 'flex',
                    gap: 10,
                  }}
                >
                  <input
                    ref={
                      refineFeedbackInputRef
                    }
                    value={
                      refineFeedback
                    }
                    onChange={event =>
                      setRefineFeedback(
                        event.target
                          .value,
                      )
                    }
                    placeholder="例如：小学生不需要学习目标页、增加互动练习、减少纯文字页面..."
                    onKeyDown={event => {
                      if (
                        refineFeedbackDraft.handleKeyDown(
                          event,
                        )
                      ) {
                        return
                      }

                      if (
                        event.key ===
                          'Enter' &&
                        refineFeedback.trim() &&
                        !refineFeedbackVoice.isActive
                      ) {
                        handleRefineIndex()
                      }
                    }}
                    disabled={
                      refineFeedbackVoice.isActive
                    }
                    style={{
                      flex: 1,
                      padding:
                        '10px 14px',
                      borderRadius: 8,
                      border:
                        '1px solid ' +
                        C.border,
                      fontSize: 14,
                      outline: 'none',
                      opacity:
                        refineFeedbackVoice.isActive
                          ? 0.6
                          : 1,
                    }}
                  />

                  <VoiceInputButton
                    status={
                      refineFeedbackVoice.status
                    }
                    isSupported={
                      refineFeedbackVoice.isSupported
                    }
                    elapsedSeconds={
                      refineFeedbackVoice.elapsedSeconds
                    }
                    disabled={
                      refining
                      || confirming
                    }
                    error={
                      refineFeedbackVoice.error
                    }
                    onStart={
                      refineFeedbackVoice.begin
                    }
                    onStop={
                      refineFeedbackVoice.stop
                    }
                    onCancel={
                      refineFeedbackVoice.cancel
                    }
                  />

                  <button
                    onClick={
                      handleRefineIndex
                    }
                    disabled={
                      !refineFeedback.trim()
                      || refineFeedbackVoice.isActive
                    }
                    style={{
                      padding:
                        '10px 20px',
                      borderRadius: 8,
                      border: 'none',
                      background:
                        refineFeedback.trim()
                        && !refineFeedbackVoice.isActive
                          ? '#7C3AED'
                          : '#E5E7EB',
                      color:
                        refineFeedback.trim()
                        && !refineFeedbackVoice.isActive
                          ? '#fff'
                          : '#9CA3AF',
                      fontSize: 14,
                      fontWeight: 600,
                      cursor:
                        refineFeedback.trim()
                        && !refineFeedbackVoice.isActive
                          ? 'pointer'
                          : 'default',
                      whiteSpace:
                        'nowrap',
                    }}
                  >
                    🤖 AI修改方案
                  </button>
                </div>

                <div
                  style={{
                    marginTop: 6,
                    fontSize: 11,
                    color: C.textMuted,
                  }}
                >
                  {refineFeedbackVoice.statusText || (
                    <>
                      修改意见已自动保存 ·
                      点击麦克风可语音输入 ·
                      AI修改成功后清除 ·
                      Ctrl/Command+Z恢复误删
                    </>
                  )}
                </div>
              </div>
            )}

          {refining && (
            <div
              style={{
                marginTop: 16,
                textAlign: 'center',
                padding: 20,
                color: C.textMuted,
                fontSize: 14,
              }}
            >
              <div
                style={{
                  fontSize: 32,
                  marginBottom: 8,
                }}
              >
                🤖
              </div>
              AI正在根据您的意见修改方案，请稍候...
            </div>
          )}

          <MsgBar msg={sseMessage} />
        </div>
      )}
    </>
  )
}

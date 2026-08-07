/**
 * 自定义美术风格工作室主体。
 *
 * 页面结构：
 *   - 左栏：参考图模式、参考图上传和继承范围；
 *   - 右栏：当前风格结论 → AI对话 → 三类验证图 → 确认。
 *
 * 保密原则：
 *   - 前端只展示自然语言风格结论；
 *   - 完整技术索引由后端维护；
 *   - 界面没有查看、复制或下载完整索引的入口；
 *   - 前端不依赖索引正文判断是否可生成预览。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import type {
  ChangeEvent,
  CSSProperties,
  ReactNode,
} from 'react'
import {
  confirmCoursewareStyleStudio,
  createCoursewareStyleStudioSession,
  generateCoursewareStylePreviews,
  getActiveCoursewareStyleStudio,
  listCoursewareStyleAssets,
  resolveCoursewareStyleAssetURL,
  sendCoursewareStyleStudioTurn,
  uploadCoursewareStyleReference,
} from '@/api/coursewareStyleStudio'
import type {
  CoursewareStyleAsset,
  CoursewareStyleReferenceMode,
  CoursewareStyleStudioState,
} from '@/api/coursewareStyleStudio'
import type {
  SetStyleAnchorResult,
} from '@/api/coursewares'
import StyleStudioPreviewGrid from './StyleStudioPreviewGrid'
import {
  didCoursewareStyleModeChange,
  isCoursewareStyleModeDirty,
} from './styleStudioMode'

const C = {
  primary: '#7C3AED',
  primaryBg:
    'rgba(124,58,237,0.07)',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  success: '#059669',
  danger: '#EF4444',
  white: '#fff',
}

const PANEL_STYLE: CSSProperties = {
  padding: 16,
  borderRadius: 14,
  border:
    `1px solid ${C.border}`,
  background: '#fff',
}

const REFERENCE_MODES: {
  value: CoursewareStyleReferenceMode
  title: string
  description: string
}[] = [
  {
    value: 'style_only',
    title: '只提取美术风格',
    description:
      '只保留媒介、线条、色彩、材质和光影，不继承人物、环境或构图。',
  },
  {
    value: 'style_character',
    title: '风格 + 固定主体',
    description:
      '仅在确实需要同一人物、动物或标志性物体贯穿课程时使用。',
  },
  {
    value: 'inspiration',
    title: '只取抽象灵感',
    description:
      '进一步弱化复刻，只提炼通用视觉语言和氛围方向。',
  },
]

export interface StyleStudioModalContentProps {
  open: boolean
  coursewareId: string
  coursewareTitle: string
  onClose: () => void
  onConfirmed: (
    result: SetStyleAnchorResult,
  ) => void
}

interface SectionProps {
  title: string
  subtitle?: string
  action?: ReactNode
  children: ReactNode
  style?: CSSProperties
}

function Section({
  title,
  subtitle,
  action,
  children,
  style,
}: SectionProps) {
  return (
    <section
      style={{
        ...PANEL_STYLE,
        ...style,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent:
            'space-between',
          gap: 12,
          marginBottom: 11,
        }}
      >
        <div>
          <div
            style={{
              fontSize: 14,
              fontWeight: 750,
              color: C.textPrimary,
            }}
          >
            {title}
          </div>

          {subtitle && (
            <div
              style={{
                marginTop: 3,
                fontSize: 10.5,
                lineHeight: 1.5,
                color: C.textMuted,
              }}
            >
              {subtitle}
            </div>
          )}
        </div>

        {action}
      </div>

      {children}
    </section>
  )
}

function assetMapFromList(
  assets: CoursewareStyleAsset[],
): Record<
  string,
  CoursewareStyleAsset
> {
  const result: Record<
    string,
    CoursewareStyleAsset
  > = {}

  for (const asset of assets) {
    result[asset.id] = asset
  }

  return result
}

function findFirstGeneratedPreviewAsset(
  state: CoursewareStyleStudioState,
  assets: Record<
    string,
    CoursewareStyleAsset
  >,
): string {
  return (
    state.previews.find(
      preview =>
        preview.status ===
          'generated' &&
        !!preview.asset_id &&
        !!resolveCoursewareStyleAssetURL(
          assets[
            preview.asset_id || ''
          ],
        ),
    )?.asset_id || ''
  )
}

function chooseSafeSelectedAsset(
  state: CoursewareStyleStudioState,
  assets: Record<
    string,
    CoursewareStyleAsset
  >,
  currentAssetID: string,
): string {
  const session = state.session

  if (!session) {
    return ''
  }

  const referenceAssetID =
    session.reference_asset_id || ''

  const currentAssetValid =
    !!currentAssetID &&
    !!resolveCoursewareStyleAssetURL(
      assets[currentAssetID],
    )

  const currentIsRawReference =
    !!referenceAssetID &&
    currentAssetID ===
      referenceAssetID

  if (
    currentAssetValid &&
    (
      !currentIsRawReference ||
      session.reference_mode ===
        'style_character'
    )
  ) {
    return currentAssetID
  }

  const generatedAssetID =
    findFirstGeneratedPreviewAsset(
      state,
      assets,
    )

  if (generatedAssetID) {
    return generatedAssetID
  }

  if (
    session.reference_mode ===
      'style_character' &&
    referenceAssetID &&
    resolveCoursewareStyleAssetURL(
      assets[referenceAssetID],
    )
  ) {
    return referenceAssetID
  }

  return ''
}

function referenceModeLabel(
  mode:
    | CoursewareStyleReferenceMode
    | undefined,
): string {
  return (
    REFERENCE_MODES.find(
      item => item.value === mode,
    )?.title || '未选择'
  )
}

export default function StyleStudioModalContent({
  open,
  coursewareId,
  coursewareTitle,
  onClose,
  onConfirmed,
}: StyleStudioModalContentProps) {
  const [
    state,
    setState,
  ] =
    useState<CoursewareStyleStudioState | null>(
      null,
    )

  const [
    assetsByID,
    setAssetsByID,
  ] = useState<
    Record<
      string,
      CoursewareStyleAsset
    >
  >({})

  const assetsRef = useRef<
    Record<
      string,
      CoursewareStyleAsset
    >
  >({})

  const [
    referenceMode,
    setReferenceMode,
  ] =
    useState<CoursewareStyleReferenceMode>(
      'style_only',
    )

  const [
    message,
    setMessage,
  ] = useState('')

  const [
    selectedAssetID,
    setSelectedAssetID,
  ] = useState('')

  const [
    loading,
    setLoading,
  ] = useState(false)

  const [
    sending,
    setSending,
  ] = useState(false)

  const [
    uploading,
    setUploading,
  ] = useState(false)

  const [
    generating,
    setGenerating,
  ] = useState(false)

  const [
    confirming,
    setConfirming,
  ] = useState(false)

  const [
    resetting,
    setResetting,
  ] = useState(false)

  const [
    error,
    setError,
  ] = useState('')

  const fileInputRef =
    useRef<HTMLInputElement>(null)

  const conversationEndRef =
    useRef<HTMLDivElement>(null)

  const session =
    state?.session || null

  const messages =
    state?.messages || []

  const previews =
    state?.previews || []

  /**
   * 新后端会返回style_ready。
   * 兼容旧服务时，以自然语言摘要存在作为安全回退判断。
   */
  const styleReady =
    Boolean(
      (
        session as {
          style_ready?: boolean
        } | null
      )?.style_ready ||
      session?.style_summary?.trim(),
    )

  const assetURLs =
    useMemo(() => {
      const result: Record<
        string,
        string
      > = {}

      for (
        const [
          assetID,
          asset,
        ] of Object.entries(
          assetsByID,
        )
      ) {
        result[assetID] =
          resolveCoursewareStyleAssetURL(
            asset,
          )
      }

      return result
    }, [assetsByID])

  const referenceAssetID =
    session?.reference_asset_id ||
    ''

  const referenceURL =
    referenceAssetID
      ? assetURLs[
          referenceAssetID
        ] || ''
      : ''

  const selectedAssetURL =
    selectedAssetID
      ? assetURLs[
          selectedAssetID
        ] || ''
      : ''

  const originalReferenceSelectable =
    referenceMode ===
      'style_character'

  const modeDirty =
    isCoursewareStyleModeDirty(
      session?.reference_mode,
      referenceMode,
    )

  const setAssetMap =
    useCallback(
      (
        nextAssets: Record<
          string,
          CoursewareStyleAsset
        >,
      ) => {
        assetsRef.current =
          nextAssets

        setAssetsByID(
          nextAssets,
        )
      },
      [],
    )

  const refreshAssets =
    useCallback(async () => {
      const assets =
        await listCoursewareStyleAssets(
          coursewareId,
        )

      const nextMap =
        assetMapFromList(assets)

      setAssetMap(nextMap)

      return nextMap
    }, [
      coursewareId,
      setAssetMap,
    ])

  const applyState =
    useCallback(
      (
        nextState:
          CoursewareStyleStudioState,
        availableAssets:
          Record<
            string,
            CoursewareStyleAsset
          > =
            assetsRef.current,
      ) => {
        setState(nextState)

        if (nextState.session) {
          setReferenceMode(
            nextState.session
              .reference_mode,
          )
        }

        setSelectedAssetID(
          current =>
            chooseSafeSelectedAsset(
              nextState,
              availableAssets,
              current,
            ),
        )
      },
      [],
    )

  const initialise =
    useCallback(async () => {
      setLoading(true)
      setError('')

      try {
        const [
          activeState,
          assetList,
        ] = await Promise.all([
          getActiveCoursewareStyleStudio(
            coursewareId,
          ),
          listCoursewareStyleAssets(
            coursewareId,
          ),
        ])

        const nextAssetMap =
          assetMapFromList(
            assetList,
          )

        setAssetMap(
          nextAssetMap,
        )

        if (activeState.session) {
          applyState(
            activeState,
            nextAssetMap,
          )
          return
        }

        const newState =
          await createCoursewareStyleStudioSession(
            coursewareId,
            {
              reference_mode:
                'style_only',
            },
          )

        applyState(
          newState,
          nextAssetMap,
        )
      } catch (requestError) {
        setError(
          requestError instanceof Error
            ? requestError.message
            : '风格工作室加载失败',
        )
      } finally {
        setLoading(false)
      }
    }, [
      applyState,
      coursewareId,
      setAssetMap,
    ])

  useEffect(() => {
    if (open) {
      void initialise()
    }
  }, [
    initialise,
    open,
  ])

  useEffect(() => {
    conversationEndRef.current
      ?.scrollIntoView({
        behavior: 'smooth',
        block: 'nearest',
      })
  }, [
    messages.length,
    sending,
  ])

  useEffect(() => {
    if (
      referenceMode !==
        'style_character' &&
      selectedAssetID ===
        referenceAssetID
    ) {
      setSelectedAssetID('')
    }
  }, [
    referenceAssetID,
    referenceMode,
    selectedAssetID,
  ])

  const busy =
    loading ||
    sending ||
    uploading ||
    generating ||
    confirming ||
    resetting

  const handleModeChange = (
    nextMode:
      CoursewareStyleReferenceMode,
  ) => {
    const changed =
      didCoursewareStyleModeChange(
        referenceMode,
        nextMode,
      )

    setReferenceMode(nextMode)

    if (changed) {
      setSelectedAssetID('')
      setError('')
    }
  }

  const handleSend =
    async () => {
      const content =
        message.trim()

      if (
        !session ||
        sending ||
        !content
      ) {
        return
      }

      setSending(true)
      setError('')

      try {
        const result =
          await sendCoursewareStyleStudioTurn(
            coursewareId,
            session.id,
            {
              content,
              reference_mode:
                referenceMode,
            },
          )

        setMessage('')
        applyState(
          result.state,
        )
      } catch (requestError) {
        setError(
          requestError instanceof Error
            ? requestError.message
            : 'AI风格对话失败',
        )
      } finally {
        setSending(false)
      }
    }

  const handleReferenceUpload =
    async (
      event:
        ChangeEvent<HTMLInputElement>,
    ) => {
      const file =
        event.target.files?.[0]

      if (
        !file ||
        !session ||
        uploading
      ) {
        return
      }

      if (
        file.size >
        8 * 1024 * 1024
      ) {
        setError(
          '参考图片不能超过8MB',
        )
        event.target.value = ''
        return
      }

      setUploading(true)
      setError('')

      try {
        const uploaded =
          await uploadCoursewareStyleReference(
            coursewareId,
            file,
          )

        const result =
          await sendCoursewareStyleStudioTurn(
            coursewareId,
            session.id,
            {
              content:
                '请根据这张参考图片提取并更新整套课件的美术风格。',
              reference_mode:
                referenceMode,
              reference_asset_id:
                uploaded.asset_id,
            },
          )

        const nextAssets =
          await refreshAssets()

        applyState(
          result.state,
          nextAssets,
        )

        setSelectedAssetID(
          referenceMode ===
            'style_character'
            ? uploaded.asset_id
            : '',
        )
      } catch (requestError) {
        setError(
          requestError instanceof Error
            ? requestError.message
            : '参考图片上传或风格提取失败',
        )
      } finally {
        setUploading(false)

        if (fileInputRef.current) {
          fileInputRef.current.value =
            ''
        }
      }
    }

  const handleGeneratePreviews =
    async () => {
      if (
        !session ||
        generating ||
        !styleReady
      ) {
        return
      }

      setGenerating(true)
      setError('')

      try {
        const nextState =
          await generateCoursewareStylePreviews(
            coursewareId,
            session.id,
            {
              reference_mode:
                referenceMode,
            },
          )

        const nextAssets =
          await refreshAssets()

        applyState(
          nextState,
          nextAssets,
        )

        setSelectedAssetID(
          findFirstGeneratedPreviewAsset(
            nextState,
            nextAssets,
          ),
        )
      } catch (requestError) {
        setError(
          requestError instanceof Error
            ? requestError.message
            : '风格预览生成失败',
        )
      } finally {
        setGenerating(false)
      }
    }

  const handleConfirm =
    async () => {
      if (
        !session ||
        confirming ||
        !selectedAssetID ||
        !styleReady ||
        modeDirty
      ) {
        return
      }

      setConfirming(true)
      setError('')

      try {
        await confirmCoursewareStyleStudio(
          coursewareId,
          session.id,
          selectedAssetID,
          referenceMode,
        )

        onConfirmed({
          asset_id:
            selectedAssetID,
          anchor_url:
            selectedAssetURL,
          // 保留旧前端回调结构，但不向浏览器传递任何索引正文。
          vaoci: '',
        })
      } catch (requestError) {
        setError(
          requestError instanceof Error
            ? requestError.message
            : '确认课程美术风格失败',
        )
      } finally {
        setConfirming(false)
      }
    }

  const handleNewSession =
    async () => {
      if (resetting) {
        return
      }

      setResetting(true)
      setError('')

      try {
        const nextState =
          await createCoursewareStyleStudioSession(
            coursewareId,
            {
              reference_mode:
                referenceMode,
            },
          )

        setMessage('')
        setSelectedAssetID('')
        applyState(nextState)
      } catch (requestError) {
        setError(
          requestError instanceof Error
            ? requestError.message
            : '新建风格会话失败',
        )
      } finally {
        setResetting(false)
      }
    }

  if (!open) {
    return null
  }

  const confirmDisabled =
    busy ||
    !selectedAssetID ||
    !styleReady ||
    modeDirty

  return (
    <div
      onClick={() => {
        if (!busy) {
          onClose()
        }
      }}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 100500,
        padding: 20,
        background:
          'rgba(15,23,42,0.64)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <style>
        {`
          .tedna-style-studio-layout {
            display: grid;
            grid-template-columns: minmax(280px, 320px) minmax(0, 1fr);
            gap: 18px;
            align-items: start;
          }

          .tedna-style-studio-left {
            position: sticky;
            top: 0;
          }

          @media (max-width: 820px) {
            .tedna-style-studio-layout {
              grid-template-columns: minmax(0, 1fr);
            }

            .tedna-style-studio-left {
              position: static;
            }
          }
        `}
      </style>

      <div
        onClick={event =>
          event.stopPropagation()
        }
        style={{
          width:
            'min(1180px, 100%)',
          maxHeight: '92vh',
          overflow: 'hidden',
          borderRadius: 18,
          background: C.white,
          boxShadow:
            '0 28px 80px rgba(15,23,42,0.36)',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <header
          style={{
            flexShrink: 0,
            padding: '16px 20px',
            borderBottom:
              `1px solid ${C.border}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent:
              'space-between',
            gap: 16,
          }}
        >
          <div>
            <div
              style={{
                fontSize: 18,
                fontWeight: 800,
                color: C.textPrimary,
              }}
            >
              ✨ 自定义美术风格
            </div>

            <div
              style={{
                marginTop: 3,
                fontSize: 12,
                color: C.textSecondary,
              }}
            >
              《{coursewareTitle}》
              · 看着当前结论继续与AI调整
            </div>
          </div>

          <div
            style={{
              display: 'flex',
              gap: 8,
            }}
          >
            <button
              type="button"
              disabled={busy}
              onClick={
                handleNewSession
              }
              style={{
                padding: '7px 12px',
                borderRadius: 8,
                border:
                  `1px solid ${C.border}`,
                background: '#fff',
                color:
                  C.textSecondary,
                cursor: busy
                  ? 'not-allowed'
                  : 'pointer',
                opacity:
                  busy ? 0.55 : 1,
              }}
            >
              {resetting
                ? '新建中...'
                : '新建方案'}
            </button>

            <button
              type="button"
              disabled={busy}
              onClick={onClose}
              style={{
                width: 34,
                height: 34,
                borderRadius: 9,
                border: 'none',
                background: '#F3F4F6',
                color:
                  C.textSecondary,
                fontSize: 18,
                cursor: busy
                  ? 'not-allowed'
                  : 'pointer',
              }}
            >
              ×
            </button>
          </div>
        </header>

        <div
          style={{
            flex: 1,
            overflowY: 'auto',
            padding: 20,
            background: '#FAFAFC',
          }}
        >
          {loading ? (
            <div
              style={{
                padding:
                  '80px 20px',
                textAlign: 'center',
                color: C.textMuted,
              }}
            >
              <div
                style={{
                  fontSize: 38,
                  marginBottom: 10,
                }}
              >
                ✨
              </div>
              正在恢复风格共创会话...
            </div>
          ) : (
            <div className="tedna-style-studio-layout">
              <aside className="tedna-style-studio-left">
                <Section
                  title="参考图片如何使用"
                  subtitle="这只决定参考图中哪些信息可以进入课程统一画风"
                  style={{
                    marginBottom: 14,
                  }}
                >
                  <div
                    style={{
                      display: 'grid',
                      gap: 8,
                    }}
                  >
                    {REFERENCE_MODES.map(
                      mode => {
                        const active =
                          mode.value ===
                          referenceMode

                        return (
                          <button
                            key={
                              mode.value
                            }
                            type="button"
                            disabled={busy}
                            onClick={() =>
                              handleModeChange(
                                mode.value,
                              )
                            }
                            style={{
                              padding:
                                '10px 12px',
                              borderRadius: 10,
                              border:
                                `1.5px solid ${
                                  active
                                    ? C.primary
                                    : C.border
                                }`,
                              background:
                                active
                                  ? C.primaryBg
                                  : '#fff',
                              textAlign:
                                'left',
                              cursor: busy
                                ? 'not-allowed'
                                : 'pointer',
                            }}
                          >
                            <div
                              style={{
                                fontSize: 13,
                                fontWeight: 700,
                                color:
                                  active
                                    ? C.primary
                                    : C.textPrimary,
                              }}
                            >
                              {active
                                ? '✓ '
                                : ''}
                              {mode.title}
                            </div>

                            <div
                              style={{
                                marginTop: 3,
                                fontSize: 11,
                                lineHeight: 1.5,
                                color:
                                  C.textSecondary,
                              }}
                            >
                              {
                                mode.description
                              }
                            </div>
                          </button>
                        )
                      },
                    )}
                  </div>
                </Section>

                <Section
                  title="上传参考图片"
                  subtitle="也可以不上传，直接在右侧通过文字与AI共创"
                >
                  {referenceURL ? (
                    <div
                      style={{
                        marginBottom: 10,
                        height: 170,
                        overflow: 'hidden',
                        borderRadius: 10,
                        background:
                          '#F3F4F6',
                      }}
                    >
                      <img
                        src={referenceURL}
                        alt="风格参考图"
                        style={{
                          width: '100%',
                          height: '100%',
                          objectFit:
                            'contain',
                        }}
                      />
                    </div>
                  ) : (
                    <div
                      style={{
                        marginBottom: 10,
                        padding:
                          '28px 12px',
                        borderRadius: 10,
                        border:
                          `1px dashed ${C.border}`,
                        textAlign:
                          'center',
                        color:
                          C.textMuted,
                        fontSize: 12,
                      }}
                    >
                      暂未上传参考图片
                    </div>
                  )}

                  <input
                    ref={fileInputRef}
                    type="file"
                    accept="image/jpeg,image/png,image/webp"
                    disabled={busy}
                    onChange={
                      handleReferenceUpload
                    }
                    style={{
                      display: 'none',
                    }}
                  />

                  <button
                    type="button"
                    disabled={busy}
                    onClick={() =>
                      fileInputRef.current?.click()
                    }
                    style={{
                      width: '100%',
                      padding:
                        '9px 12px',
                      borderRadius: 9,
                      border:
                        `1px solid ${C.primary}`,
                      background:
                        C.primaryBg,
                      color: C.primary,
                      fontSize: 13,
                      fontWeight: 700,
                      cursor: busy
                        ? 'not-allowed'
                        : 'pointer',
                      opacity:
                        busy ? 0.6 : 1,
                    }}
                  >
                    {uploading
                      ? '正在上传并分析...'
                      : referenceURL
                        ? '更换参考图片'
                        : '上传参考图片'}
                  </button>

                  <div
                    style={{
                      marginTop: 10,
                      padding:
                        '9px 10px',
                      borderRadius: 8,
                      background:
                        '#F9FAFB',
                      fontSize: 10.5,
                      color: C.textMuted,
                      lineHeight: 1.6,
                    }}
                  >
                    支持JPG、PNG、WEBP，最大8MB。
                    系统默认不继承文字、Logo、水印、具体环境、镜头和构图。
                    内部技术规则不会在界面中展示。
                  </div>
                </Section>
              </aside>

              <main
                style={{
                  minWidth: 0,
                  display: 'grid',
                  gap: 14,
                }}
              >
                <Section
                  title="当前风格结论"
                  subtitle="AI每轮都会更新这里，您可以紧接着在下方继续调整"
                  action={
                    session ? (
                      <div
                        style={{
                          display: 'flex',
                          gap: 6,
                          flexWrap: 'wrap',
                          justifyContent:
                            'flex-end',
                        }}
                      >
                        <span
                          style={{
                            padding:
                              '3px 8px',
                            borderRadius: 10,
                            background:
                              '#EDE9FE',
                            color:
                              C.primary,
                            fontSize: 10,
                            fontWeight: 700,
                          }}
                        >
                          {
                            referenceModeLabel(
                              referenceMode,
                            )
                          }
                        </span>

                        <span
                          style={{
                            padding:
                              '3px 8px',
                            borderRadius: 10,
                            background:
                              styleReady
                                ? '#D1FAE5'
                                : '#F3F4F6',
                            color:
                              styleReady
                                ? C.success
                                : C.textMuted,
                            fontSize: 10,
                            fontWeight: 700,
                          }}
                        >
                          V{session.version}
                          {' · '}
                          {styleReady
                            ? '已形成'
                            : '待完善'}
                        </span>
                      </div>
                    ) : null
                  }
                  style={{
                    background:
                      styleReady
                        ? 'linear-gradient(135deg, rgba(124,58,237,0.06), rgba(59,130,246,0.04))'
                        : '#fff',
                    border:
                      styleReady
                        ? '1px solid rgba(124,58,237,0.24)'
                        : `1px solid ${C.border}`,
                  }}
                >
                  {session
                    ?.style_summary?.trim() ? (
                    <div
                      style={{
                        padding:
                          '12px 13px',
                        borderRadius: 10,
                        background: '#fff',
                        border:
                          `1px solid ${C.border}`,
                        fontSize: 13,
                        lineHeight: 1.75,
                        color:
                          C.textSecondary,
                      }}
                    >
                      {
                        session.style_summary
                      }
                    </div>
                  ) : (
                    <div
                      style={{
                        padding:
                          '22px 12px',
                        textAlign:
                          'center',
                        color:
                          C.textMuted,
                        fontSize: 12,
                        lineHeight: 1.6,
                      }}
                    >
                      在下方描述想要的画风，
                      或从左侧上传参考图片。
                      AI形成结论后会持续在这里更新。
                    </div>
                  )}
                </Section>

                <Section
                  title="与AI继续调整"
                  subtitle="先看上方结论，再直接告诉AI需要修改的地方"
                >
                  <div
                    style={{
                      minHeight: 120,
                      maxHeight: 245,
                      overflowY: 'auto',
                      padding:
                        messages.length >
                        0
                          ? '4px 2px 8px'
                          : '24px 10px',
                      display: 'grid',
                      alignContent:
                        messages.length >
                        0
                          ? 'start'
                          : 'center',
                      gap: 8,
                      borderRadius: 10,
                      background:
                        '#FAFAFA',
                      border:
                        `1px solid ${C.border}`,
                    }}
                  >
                    {messages.length ===
                    0 ? (
                      <div
                        style={{
                          textAlign:
                            'center',
                          color:
                            C.textMuted,
                          fontSize: 12,
                          lineHeight: 1.65,
                        }}
                      >
                        例如：颜色更柔和一些；
                        减少卡通感；
                        线条更轻；
                        教学图解要更清晰克制。
                      </div>
                    ) : (
                      messages.map(item => (
                        <div
                          key={item.id}
                          style={{
                            justifySelf:
                              item.role ===
                              'user'
                                ? 'end'
                                : 'start',
                            maxWidth: '88%',
                            padding:
                              '8px 11px',
                            borderRadius:
                              item.role ===
                              'user'
                                ? '11px 11px 3px 11px'
                                : '11px 11px 11px 3px',
                            background:
                              item.role ===
                              'user'
                                ? C.primary
                                : '#fff',
                            border:
                              item.role ===
                              'user'
                                ? 'none'
                                : `1px solid ${C.border}`,
                            color:
                              item.role ===
                              'user'
                                ? '#fff'
                                : C.textPrimary,
                            fontSize: 12,
                            lineHeight: 1.55,
                            whiteSpace:
                              'pre-wrap',
                          }}
                        >
                          {item.content}
                        </div>
                      ))
                    )}

                    {sending && (
                      <div
                        style={{
                          justifySelf:
                            'start',
                          padding:
                            '8px 11px',
                          borderRadius:
                            '11px 11px 11px 3px',
                          background:
                            '#fff',
                          border:
                            `1px solid ${C.border}`,
                          color:
                            C.textMuted,
                          fontSize: 12,
                        }}
                      >
                        AI正在更新风格结论...
                      </div>
                    )}

                    <div
                      ref={
                        conversationEndRef
                      }
                    />
                  </div>

                  <textarea
                    value={message}
                    disabled={busy}
                    onChange={event =>
                      setMessage(
                        event.target.value,
                      )
                    }
                    onKeyDown={event => {
                      if (
                        event.key ===
                          'Enter' &&
                        (
                          event.ctrlKey ||
                          event.metaKey
                        )
                      ) {
                        event.preventDefault()
                        void handleSend()
                      }
                    }}
                    placeholder="看着上面的结论继续调整，例如：保留配色，但把人物改得更自然……"
                    style={{
                      width: '100%',
                      minHeight: 92,
                      marginTop: 10,
                      resize: 'vertical',
                      padding:
                        '10px 12px',
                      borderRadius: 10,
                      border:
                        `1px solid ${C.border}`,
                      outline: 'none',
                      fontSize: 13,
                      lineHeight: 1.6,
                      boxSizing:
                        'border-box',
                    }}
                  />

                  <button
                    type="button"
                    disabled={
                      busy ||
                      !message.trim() ||
                      !session
                    }
                    onClick={
                      handleSend
                    }
                    style={{
                      width: '100%',
                      marginTop: 9,
                      padding:
                        '10px 14px',
                      borderRadius: 9,
                      border: 'none',
                      background:
                        busy ||
                        !message.trim()
                          ? '#D1D5DB'
                          : `linear-gradient(135deg, ${C.primary}, #6366F1)`,
                      color: '#fff',
                      fontSize: 13,
                      fontWeight: 700,
                      cursor:
                        busy ||
                        !message.trim()
                          ? 'not-allowed'
                          : 'pointer',
                    }}
                  >
                    {sending
                      ? 'AI正在更新...'
                      : '发送并更新结论（Ctrl/⌘ + Enter）'}
                  </button>
                </Section>

                {modeDirty && (
                  <div
                    style={{
                      padding:
                        '10px 12px',
                      borderRadius: 9,
                      background:
                        '#FFFBEB',
                      border:
                        '1px solid #FCD34D',
                      color: '#92400E',
                      fontSize: 12,
                      lineHeight: 1.55,
                    }}
                  >
                    ⚠️ 参考图模式已经切换。
                    旧预览暂不可确认；
                    请发送一条新要求、
                    重新上传参考图，
                    或按当前模式重新生成验证图。
                  </div>
                )}

                {error && (
                  <div
                    style={{
                      padding:
                        '10px 12px',
                      borderRadius: 9,
                      background:
                        '#FEF2F2',
                      border:
                        '1px solid #FECACA',
                      color: C.danger,
                      fontSize: 12,
                      lineHeight: 1.55,
                    }}
                  >
                    ❌ {error}
                  </div>
                )}

                <Section
                  title="三类风格验证图"
                  subtitle="验证图只用于检查画风，不会自动插入课程页面"
                  action={
                    <button
                      type="button"
                      disabled={
                        busy ||
                        !styleReady
                      }
                      onClick={
                        handleGeneratePreviews
                      }
                      style={{
                        padding:
                          '7px 12px',
                        borderRadius: 8,
                        border:
                          `1px solid ${C.primary}`,
                        background:
                          C.primaryBg,
                        color:
                          C.primary,
                        fontSize: 12,
                        fontWeight: 700,
                        cursor:
                          busy ||
                          !styleReady
                            ? 'not-allowed'
                            : 'pointer',
                        opacity:
                          busy ||
                          !styleReady
                            ? 0.55
                            : 1,
                      }}
                    >
                      {generating
                        ? '正在生成三张验证图...'
                        : previews.length >
                            0
                          ? '重新生成验证图'
                          : '生成三张验证图'}
                    </button>
                  }
                >
                  <StyleStudioPreviewGrid
                    previews={previews}
                    assetURLs={
                      assetURLs
                    }
                    selectedAssetId={
                      selectedAssetID
                    }
                    disabled={
                      busy ||
                      modeDirty
                    }
                    onSelect={
                      setSelectedAssetID
                    }
                  />
                </Section>

                {referenceURL && (
                  <section
                    onClick={() => {
                      if (
                        !busy &&
                        !modeDirty &&
                        originalReferenceSelectable
                      ) {
                        setSelectedAssetID(
                          referenceAssetID,
                        )
                      }
                    }}
                    style={{
                      ...PANEL_STYLE,
                      display: 'flex',
                      gap: 12,
                      alignItems:
                        'center',
                      border:
                        `2px solid ${
                          selectedAssetID ===
                          referenceAssetID
                            ? C.primary
                            : C.border
                        }`,
                      background:
                        selectedAssetID ===
                        referenceAssetID
                          ? C.primaryBg
                          : '#fff',
                      cursor:
                        !busy &&
                        !modeDirty &&
                        originalReferenceSelectable
                          ? 'pointer'
                          : 'default',
                      opacity:
                        originalReferenceSelectable
                          ? 1
                          : 0.68,
                    }}
                  >
                    <img
                      src={referenceURL}
                      alt="上传的参考图片"
                      style={{
                        width: 92,
                        height: 62,
                        borderRadius: 8,
                        objectFit: 'cover',
                        background:
                          '#F3F4F6',
                      }}
                    />

                    <div
                      style={{
                        flex: 1,
                      }}
                    >
                      <div
                        style={{
                          fontSize: 13,
                          fontWeight: 700,
                          color:
                            C.textPrimary,
                        }}
                      >
                        上传的参考图片
                      </div>

                      <div
                        style={{
                          marginTop: 3,
                          fontSize: 11,
                          lineHeight: 1.5,
                          color:
                            C.textSecondary,
                        }}
                      >
                        {originalReferenceSelectable
                          ? '固定主体模式下，可明确选择原图作为课程锚点'
                          : '当前模式只提取艺术语言，请选择一张AI生成的验证图'}
                      </div>
                    </div>

                    <div
                      style={{
                        width: 22,
                        height: 22,
                        borderRadius:
                          '50%',
                        border:
                          `2px solid ${
                            selectedAssetID ===
                            referenceAssetID
                              ? C.primary
                              : C.border
                          }`,
                        background:
                          selectedAssetID ===
                          referenceAssetID
                            ? C.primary
                            : '#fff',
                        color: '#fff',
                        display: 'flex',
                        alignItems:
                          'center',
                        justifyContent:
                          'center',
                        fontSize: 12,
                      }}
                    >
                      {selectedAssetID ===
                      referenceAssetID
                        ? '✓'
                        : ''}
                    </div>
                  </section>
                )}

                <div
                  style={{
                    padding:
                      '13px 15px',
                    borderRadius: 12,
                    background:
                      '#F9FAFB',
                    border:
                      `1px solid ${C.border}`,
                    display: 'flex',
                    alignItems:
                      'center',
                    justifyContent:
                      'space-between',
                    gap: 14,
                    flexWrap: 'wrap',
                  }}
                >
                  <div
                    style={{
                      flex: 1,
                      minWidth: 220,
                      fontSize: 11,
                      lineHeight: 1.55,
                      color:
                        C.textSecondary,
                    }}
                  >
                    {selectedAssetID
                      ? '已选择课程统一画风锚点。确认后，后续图片生成会在后台自动保持一致。'
                      : referenceURL &&
                          !originalReferenceSelectable
                        ? '请先生成并选择一张验证图。'
                        : '请先形成风格结论，再生成验证图并选择。'}
                  </div>

                  <button
                    type="button"
                    disabled={
                      confirmDisabled
                    }
                    onClick={
                      handleConfirm
                    }
                    style={{
                      padding:
                        '11px 20px',
                      borderRadius: 10,
                      border: 'none',
                      background:
                        confirmDisabled
                          ? '#D1D5DB'
                          : `linear-gradient(135deg, ${C.primary}, #4F46E5)`,
                      color: '#fff',
                      fontSize: 13,
                      fontWeight: 800,
                      cursor:
                        confirmDisabled
                          ? 'not-allowed'
                          : 'pointer',
                      boxShadow:
                        confirmDisabled
                          ? 'none'
                          : '0 4px 14px rgba(124,58,237,0.28)',
                    }}
                  >
                    {confirming
                      ? '正在确认课程画风...'
                      : '就用这个美术风格'}
                  </button>
                </div>
              </main>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

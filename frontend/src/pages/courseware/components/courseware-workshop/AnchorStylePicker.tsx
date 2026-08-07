/**
 * 全自动装配前的插图画风选择器。
 *
 * 快捷预设流程：
 *   加载十张轻量真实缩略图
 *   → 老师点击卡片进行选择
 *   → 点击底部确认按钮
 *   → 后端直接使用系统高清图创建课程锚点
 *   → 立即开始装配。
 *
 * 快捷预设不再调用图片AI，也不再生成第二张正式样板。
 * 自定义美术风格工作室保持原有独立链路。
 */

import {
  useEffect,
  useMemo,
  useState,
} from 'react'

import type {
  SetStyleAnchorResult,
} from '@/api/coursewares'

import {
  setPresetStyleAnchor,
} from '@/api/coursewares.preset-style-anchor'

import {
  C,
  CW_IMG_STYLES,
} from './workshopConstants'

import StyleStudioModal
  from '../style-studio/StyleStudioModal'

interface Props {
  coursewareId: string
  skipVideo: boolean
  onConfirmed: () => void
  onCancel: () => void
  onAnchorChanged: (
    result: SetStyleAnchorResult,
  ) => void
}

interface PresetThumbnailItem {
  key: string
  label: string
  url: string
  file_name: string
  mime_type: string
  file_size: number
}

interface PresetThumbnailManifest {
  version: number
  generated_at: string
  image_size: string
  styles: PresetThumbnailItem[]
}

interface ThumbnailSource {
  primary: string
  fallback: string
}

const CUSTOM_STYLE_KEY =
  'custom'

const PRESET_THUMBNAIL_DIRECTORY =
  '/uploads/courseware-assets/style-presets'

const PRESET_THUMBNAIL_MANIFEST_URL =
  `${PRESET_THUMBNAIL_DIRECTORY}/manifest.json`

function normalizeStyleImageURL(
  value: string | undefined,
): string {
  const original =
    value?.trim() || ''

  if (!original) {
    return ''
  }

  const url =
    original.replaceAll(
      '\\',
      '/',
    )

  if (
    url.startsWith('https://') ||
    url.startsWith('http://')
  ) {
    return url
  }

  if (url.startsWith('//')) {
    return `${window.location.protocol}${url}`
  }

  if (url.startsWith('/')) {
    return url
  }

  const normalized =
    url
      .replace(/^\.\//, '')
      .replace(/^\/+/, '')

  if (
    normalized.startsWith(
      'uploads/',
    )
  ) {
    return `/${normalized}`
  }

  return ''
}

function appendURLVersion(
  url: string,
  version: string,
): string {
  if (!url || !version) {
    return url
  }

  const separator =
    url.includes('?')
      ? '&'
      : '?'

  return `${url}${separator}v=${encodeURIComponent(version)}`
}

/**
 * 清单使用no-cache：
 * 浏览器可以复用已有响应，但会验证清单是否更新。
 *
 * 图片URL附带generated_at版本参数：
 * 管理员重新生成图片后URL自动变化；
 * 普通打开弹窗时则直接命中浏览器缓存。
 */
async function fetchPresetThumbnailSources(
  signal: AbortSignal,
): Promise<Record<string, ThumbnailSource>> {
  const response =
    await fetch(
      PRESET_THUMBNAIL_MANIFEST_URL,
      {
        method: 'GET',
        cache: 'no-cache',
        signal,
      },
    )

  if (!response.ok) {
    throw new Error(
      `HTTP ${response.status}`,
    )
  }

  const manifest =
    await response.json() as
      PresetThumbnailManifest

  if (
    !manifest ||
    !Array.isArray(
      manifest.styles,
    )
  ) {
    throw new Error(
      '缩略图清单格式无效',
    )
  }

  const version =
    manifest.generated_at ||
    String(
      manifest.version || 1,
    )

  const result:
    Record<string, ThumbnailSource> = {}

  for (
    const item of manifest.styles
  ) {
    const key =
      item?.key?.trim()

    if (!key) {
      continue
    }

    const originalURL =
      normalizeStyleImageURL(
        item.url,
      )

    const cardURL =
      `${PRESET_THUMBNAIL_DIRECTORY}/${key}_card.jpg`

    result[key] = {
      primary:
        appendURLVersion(
          cardURL,
          version,
        ),
      fallback:
        appendURLVersion(
          originalURL,
          version,
        ),
    }
  }

  return result
}

export default function AnchorStylePicker({
  coursewareId,
  skipVideo,
  onConfirmed,
  onCancel,
  onAnchorChanged,
}: Props) {
  const [
    selectedKey,
    setSelectedKey,
  ] = useState('')

  const [
    saving,
    setSaving,
  ] = useState(false)

  const [
    message,
    setMessage,
  ] = useState('')

  const [
    showAIStudio,
    setShowAIStudio,
  ] = useState(false)

  const [
    thumbnailSources,
    setThumbnailSources,
  ] = useState<
    Record<string, ThumbnailSource>
  >({})

  const [
    thumbnailsLoading,
    setThumbnailsLoading,
  ] = useState(true)

  const [
    thumbnailError,
    setThumbnailError,
  ] = useState('')

  useEffect(() => {
    const controller =
      new AbortController()

    setThumbnailsLoading(true)
    setThumbnailError('')

    void fetchPresetThumbnailSources(
      controller.signal,
    )
      .then(
        setThumbnailSources,
      )
      .catch((error) => {
        if (
          controller.signal.aborted
        ) {
          return
        }

        setThumbnailError(
          error instanceof Error
            ? error.message
            : '未知错误',
        )
      })
      .finally(() => {
        if (
          !controller.signal.aborted
        ) {
          setThumbnailsLoading(false)
        }
      })

    return () => {
      controller.abort()
    }
  }, [])

  const selectedStyle =
    useMemo(
      () =>
        CW_IMG_STYLES.find(
          style =>
            style.key ===
            selectedKey,
        ) || null,
      [selectedKey],
    )

  const selectedPreviewURL =
    selectedStyle
      ? thumbnailSources[
          selectedStyle.key
        ]?.primary || ''
      : ''

  const handleThumbnailError =
    (
      styleKey: string,
    ) => {
      setThumbnailSources(
        current => {
          const source =
            current[styleKey]

          if (!source) {
            return current
          }

          const next = {
            ...current,
          }

          if (
            source.fallback &&
            source.primary !==
              source.fallback
          ) {
            next[styleKey] = {
              primary:
                source.fallback,
              fallback: '',
            }
          } else {
            delete next[styleKey]
          }

          return next
        },
      )
    }

  const handlePresetConfirm =
    async () => {
      if (
        saving ||
        !selectedStyle
      ) {
        return
      }

      setSaving(true)
      setMessage(
        `正在应用「${selectedStyle.label}」画风并启动装配...`,
      )

      try {
        const result =
          await setPresetStyleAnchor(
            coursewareId,
            selectedStyle.key,
          )

        onAnchorChanged(
          result,
        )

        onConfirmed()
      } catch (error) {
        setMessage(
          `❌ 设置预设画风失败：${
            error instanceof Error
              ? error.message
              : '未知错误'
          }`,
        )

        setSaving(false)
      }
    }

  const handleAIConfirmed =
    (
      result:
        SetStyleAnchorResult,
    ) => {
      onAnchorChanged(result)
      setShowAIStudio(false)
      onConfirmed()
    }

  if (showAIStudio) {
    return (
      <StyleStudioModal
        open
        coursewareId={
          coursewareId
        }
        coursewareTitle="当前课件"
        onClose={() =>
          setShowAIStudio(false)
        }
        onConfirmed={
          handleAIConfirmed
        }
      />
    )
  }

  return (
    <div
      onClick={() => {
        if (!saving) {
          onCancel()
        }
      }}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 99992,
        padding: 20,
        background:
          'rgba(0,0,0,0.58)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <div
        onClick={event =>
          event.stopPropagation()
        }
        style={{
          width: '100%',
          maxWidth: 960,
          maxHeight: '92vh',
          overflow: 'auto',
          borderRadius: 16,
          background: '#fff',
          boxShadow:
            '0 18px 64px rgba(0,0,0,0.38)',
        }}
      >
        <div
          style={{
            padding:
              '18px 22px',
            background:
              'linear-gradient(135deg, #7C3AED, #6366F1)',
          }}
        >
          <div
            style={{
              marginBottom: 4,
              fontSize: 18,
              fontWeight: 800,
              color: '#fff',
            }}
          >
            🎨 选择整套课件的插图画风
          </div>

          <div
            style={{
              maxWidth: 780,
              fontSize: 12.5,
              lineHeight: 1.65,
              color:
                'rgba(255,255,255,0.9)',
            }}
          >
            点击卡片即可预览和选择。
            确认后系统直接使用该预设画风，不再现场生成第二张样板图，
            因此无需等待，也不会产生额外图片生成费用。
          </div>
        </div>

        <div
          style={{
            padding:
              '20px 22px',
          }}
        >
          {thumbnailError && (
            <div
              style={{
                marginBottom: 14,
                padding:
                  '9px 12px',
                borderRadius: 8,
                border:
                  '1px solid #FCD34D',
                background:
                  '#FFFBEB',
                color: '#92400E',
                fontSize: 12,
              }}
            >
              ⚠️ 缩略图加载失败：
              {thumbnailError}。
              仍可按名称选择预设画风。
            </div>
          )}

          <div
            style={{
              display: 'grid',
              gridTemplateColumns:
                'repeat(auto-fill, minmax(178px, 1fr))',
              gap: 12,
              marginBottom: 16,
            }}
          >
            {CW_IMG_STYLES.map(
              (
                style,
                index,
              ) => {
                const active =
                  selectedKey ===
                  style.key

                const source =
                  thumbnailSources[
                    style.key
                  ]

                return (
                  <button
                    key={style.key}
                    type="button"
                    disabled={saving}
                    onClick={() => {
                      setSelectedKey(
                        style.key,
                      )
                      setMessage('')
                    }}
                    style={{
                      minWidth: 0,
                      padding: 0,
                      overflow:
                        'hidden',
                      borderRadius: 12,
                      border:
                        `2px solid ${
                          active
                            ? '#7C3AED'
                            : C.border
                        }`,
                      background:
                        active
                          ? 'rgba(124,58,237,0.055)'
                          : '#fff',
                      textAlign:
                        'left',
                      cursor:
                        saving
                          ? 'default'
                          : 'pointer',
                      opacity:
                        saving &&
                        !active
                          ? 0.55
                          : 1,
                      boxShadow:
                        active
                          ? '0 6px 20px rgba(124,58,237,0.18)'
                          : '0 1px 5px rgba(15,23,42,0.06)',
                    }}
                  >
                    <div
                      style={{
                        height: 102,
                        overflow:
                          'hidden',
                        background:
                          '#F1F5F9',
                        display:
                          'flex',
                        alignItems:
                          'center',
                        justifyContent:
                          'center',
                      }}
                    >
                      {source?.primary ? (
                        <img
                          src={
                            source.primary
                          }
                          alt={`${style.label}画风缩略图`}
                          loading={
                            index < 5
                              ? 'eager'
                              : 'lazy'
                          }
                          decoding="async"
                          fetchPriority={
                            index < 5
                              ? 'high'
                              : 'low'
                          }
                          onError={() =>
                            handleThumbnailError(
                              style.key,
                            )
                          }
                          style={{
                            width:
                              '100%',
                            height:
                              '100%',
                            objectFit:
                              'cover',
                            display:
                              'block',
                          }}
                        />
                      ) : (
                        <div
                          style={{
                            padding:
                              '12px',
                            textAlign:
                              'center',
                            color:
                              C.textMuted,
                            fontSize: 11,
                          }}
                        >
                          <div
                            style={{
                              marginBottom:
                                5,
                              fontSize:
                                25,
                            }}
                          >
                            {thumbnailsLoading
                              ? '⏳'
                              : '🖼️'}
                          </div>
                          {thumbnailsLoading
                            ? '加载中'
                            : '暂无图片'}
                        </div>
                      )}
                    </div>

                    <div
                      style={{
                        padding:
                          '10px 11px 12px',
                      }}
                    >
                      <div
                        style={{
                          marginBottom:
                            4,
                          fontSize:
                            13.5,
                          fontWeight:
                            750,
                          color:
                            active
                              ? '#7C3AED'
                              : C.textPrimary,
                        }}
                      >
                        {active
                          ? '✓ '
                          : ''}
                        {style.label}
                      </div>

                      <div
                        style={{
                          display:
                            'inline-block',
                          marginBottom:
                            6,
                          padding:
                            '2px 7px',
                          borderRadius:
                            10,
                          background:
                            active
                              ? '#EDE9FE'
                              : '#F1F5F9',
                          color:
                            active
                              ? '#6D28D9'
                              : C.textSecondary,
                          fontSize: 9.5,
                          fontWeight:
                            700,
                        }}
                      >
                        {
                          style.category
                        }
                      </div>

                      <div
                        style={{
                          minHeight: 48,
                          overflow:
                            'hidden',
                          display:
                            '-webkit-box',
                          WebkitLineClamp:
                            3,
                          WebkitBoxOrient:
                            'vertical',
                          color:
                            C.textSecondary,
                          fontSize: 10.5,
                          lineHeight:
                            1.5,
                        }}
                      >
                        {style.desc}
                      </div>
                    </div>
                  </button>
                )
              },
            )}

            <button
              type="button"
              disabled={saving}
              onClick={() => {
                setSelectedKey(
                  CUSTOM_STYLE_KEY,
                )
                setMessage('')
                setShowAIStudio(true)
              }}
              style={{
                minWidth: 0,
                padding: 0,
                overflow:
                  'hidden',
                borderRadius: 12,
                border:
                  '2px solid rgba(124,58,237,0.35)',
                background:
                  'linear-gradient(135deg, rgba(124,58,237,0.045), rgba(59,130,246,0.04))',
                textAlign: 'left',
                cursor:
                  saving
                    ? 'default'
                    : 'pointer',
              }}
            >
              <div
                style={{
                  height: 102,
                  background:
                    'linear-gradient(135deg, #7C3AED, #6366F1, #0EA5E9)',
                  display: 'flex',
                  alignItems:
                    'center',
                  justifyContent:
                    'center',
                  color: '#fff',
                }}
              >
                <div
                  style={{
                    textAlign:
                      'center',
                  }}
                >
                  <div
                    style={{
                      fontSize: 34,
                    }}
                  >
                    ✨
                  </div>
                  <div
                    style={{
                      fontSize: 11,
                      fontWeight:
                        700,
                    }}
                  >
                    AI共创画风
                  </div>
                </div>
              </div>

              <div
                style={{
                  padding:
                    '10px 11px 12px',
                }}
              >
                <div
                  style={{
                    marginBottom: 4,
                    fontSize: 13.5,
                    fontWeight: 800,
                    color: '#7C3AED',
                  }}
                >
                  ✨ 自定义
                </div>

                <div
                  style={{
                    display:
                      'inline-block',
                    marginBottom: 6,
                    padding:
                      '2px 7px',
                    borderRadius: 10,
                    background:
                      '#EDE9FE',
                    color: '#6D28D9',
                    fontSize: 9.5,
                    fontWeight: 700,
                  }}
                >
                  文字或参考图
                </div>

                <div
                  style={{
                    minHeight: 48,
                    color:
                      C.textSecondary,
                    fontSize: 10.5,
                    lineHeight: 1.5,
                  }}
                >
                  使用文字对话或参考图片，定制预设之外的艺术语言。
                </div>
              </div>
            </button>
          </div>

          {selectedStyle && (
            <div
              style={{
                marginBottom: 15,
                padding: 14,
                borderRadius: 12,
                border:
                  '2px solid #7C3AED',
                background:
                  '#F9F7FF',
                display: 'flex',
                alignItems:
                  'center',
                gap: 14,
                flexWrap: 'wrap',
              }}
            >
              <div
                style={{
                  width: 200,
                  height: 112,
                  overflow:
                    'hidden',
                  flexShrink: 0,
                  borderRadius: 9,
                  background:
                    '#EDE9FE',
                }}
              >
                {selectedPreviewURL ? (
                  <img
                    src={
                      selectedPreviewURL
                    }
                    alt={`${selectedStyle.label}已选画风`}
                    decoding="async"
                    onError={() =>
                      handleThumbnailError(
                        selectedStyle.key,
                      )
                    }
                    style={{
                      width: '100%',
                      height: '100%',
                      objectFit:
                        'cover',
                      display:
                        'block',
                    }}
                  />
                ) : (
                  <div
                    style={{
                      height: '100%',
                      display:
                        'flex',
                      alignItems:
                        'center',
                      justifyContent:
                        'center',
                      color:
                        C.textMuted,
                      fontSize: 12,
                    }}
                  >
                    已选择该画风
                  </div>
                )}
              </div>

              <div
                style={{
                  flex: 1,
                  minWidth: 250,
                }}
              >
                <div
                  style={{
                    marginBottom: 5,
                    color: '#7C3AED',
                    fontSize: 15,
                    fontWeight: 800,
                  }}
                >
                  已选择：
                  {
                    selectedStyle.label
                  }
                </div>

                <div
                  style={{
                    color:
                      C.textSecondary,
                    fontSize: 12,
                    lineHeight: 1.7,
                  }}
                >
                  {selectedStyle.desc}
                  <br />
                  确认后直接使用系统高清预设图作为风格锚点，
                  不会再调用图片AI生成额外样板。
                </div>
              </div>
            </div>
          )}

          {message && (
            <div
              style={{
                marginBottom: 14,
                padding:
                  '10px 14px',
                borderRadius: 8,
                background:
                  message.startsWith(
                    '❌',
                  )
                    ? '#FEE2E2'
                    : '#EFF6FF',
                color:
                  message.startsWith(
                    '❌',
                  )
                    ? '#DC2626'
                    : '#2563EB',
                fontSize: 13,
              }}
            >
              {message}
            </div>
          )}

          <div
            style={{
              display: 'flex',
              gap: 10,
              justifyContent:
                'flex-end',
              flexWrap: 'wrap',
            }}
          >
            <button
              type="button"
              disabled={saving}
              onClick={onCancel}
              style={{
                padding:
                  '10px 20px',
                borderRadius: 9,
                border:
                  `1px solid ${C.border}`,
                background: '#fff',
                color:
                  C.textSecondary,
                fontSize: 14,
                cursor:
                  saving
                    ? 'default'
                    : 'pointer',
              }}
            >
              取消
            </button>

            <button
              type="button"
              disabled={
                saving ||
                !selectedStyle
              }
              onClick={() =>
                void handlePresetConfirm()
              }
              style={{
                padding:
                  '10px 24px',
                borderRadius: 9,
                border: 'none',
                background:
                  saving ||
                  !selectedStyle
                    ? '#D1D5DB'
                    : 'linear-gradient(135deg, #7C3AED, #6366F1)',
                color: '#fff',
                fontSize: 14,
                fontWeight: 700,
                cursor:
                  saving ||
                  !selectedStyle
                    ? 'not-allowed'
                    : 'pointer',
                boxShadow:
                  saving ||
                  !selectedStyle
                    ? 'none'
                    : '0 3px 12px rgba(124,58,237,0.3)',
              }}
            >
              {saving
                ? '正在应用画风...'
                : selectedStyle
                  ? `✓ 使用该画风，开始${
                      skipVideo
                        ? '装配'
                        : '全自动装配'
                    }`
                  : '请先选择一种画风'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

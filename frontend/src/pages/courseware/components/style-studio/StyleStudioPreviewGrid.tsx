/**
 * AI美术风格工作室三类预览网格。
 *
 * 三张图分别检验：
 *   - 人物和教学情境；
 *   - 非人物知识对象；
 *   - 教学流程图解。
 *
 * 预览图只用于确认艺术语言，不会自动插入课件页面。
 */

import type {
  CoursewareStylePreview,
  CoursewareStylePreviewType,
} from '@/api/coursewareStyleStudio'

const C = {
  primary: '#7C3AED',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  success: '#059669',
  danger: '#EF4444',
  white: '#fff',
}

const PREVIEW_CONFIG: Record<
  CoursewareStylePreviewType,
  {
    title: string
    emoji: string
    description: string
  }
> = {
  character: {
    title: '人物情境',
    emoji: '🧑‍🏫',
    description: '检验人物造型、表情、材质和教学氛围',
  },
  object: {
    title: '知识对象',
    emoji: '🔬',
    description: '检验非人物主体、色彩、材质和细节表现',
  },
  diagram: {
    title: '教学图解',
    emoji: '🧩',
    description: '检验箭头、层级、图标和知识表达清晰度',
  },
}

const PREVIEW_ORDER: CoursewareStylePreviewType[] = [
  'character',
  'object',
  'diagram',
]

interface Props {
  previews: CoursewareStylePreview[]
  assetURLs: Record<string, string>
  selectedAssetId: string
  disabled?: boolean
  onSelect: (assetId: string) => void
}

function previewStatusText(
  preview: CoursewareStylePreview | undefined,
): string {
  if (!preview) return '尚未生成'

  switch (preview.status) {
    case 'pending':
      return '等待生成'
    case 'generating':
      return '生成中...'
    case 'generated':
      return '已生成'
    case 'failed':
      return '生成失败'
    case 'stale':
      return '风格已变化，请重新生成'
    default:
      return preview.status
  }
}

export default function StyleStudioPreviewGrid({
  previews,
  assetURLs,
  selectedAssetId,
  disabled = false,
  onSelect,
}: Props) {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns:
          'repeat(auto-fit, minmax(220px, 1fr))',
        gap: 14,
      }}
    >
      {PREVIEW_ORDER.map((previewType) => {
        const config = PREVIEW_CONFIG[previewType]
        const preview = previews.find(
          (item) => item.preview_type === previewType,
        )
        const assetId = preview?.asset_id || ''
        const imageURL =
          assetId
            ? assetURLs[assetId] || ''
            : ''
        const generated =
          preview?.status === 'generated' &&
          !!assetId &&
          !!imageURL
        const selected =
          generated &&
          selectedAssetId === assetId

        return (
          <button
            key={previewType}
            type="button"
            disabled={!generated || disabled}
            onClick={() => {
              if (generated) onSelect(assetId)
            }}
            style={{
              position: 'relative',
              padding: 0,
              overflow: 'hidden',
              borderRadius: 14,
              border: `2px solid ${
                selected
                  ? C.primary
                  : C.border
              }`,
              background: C.white,
              cursor:
                generated && !disabled
                  ? 'pointer'
                  : 'default',
              textAlign: 'left',
              boxShadow: selected
                ? '0 6px 22px rgba(124,58,237,0.22)'
                : '0 1px 5px rgba(15,23,42,0.06)',
              opacity:
                disabled
                  ? 0.7
                  : 1,
            }}
          >
            <div
              style={{
                height: 145,
                background:
                  'linear-gradient(135deg, #F5F3FF, #EEF2FF)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                overflow: 'hidden',
              }}
            >
              {imageURL ? (
                <img
                  src={imageURL}
                  alt={config.title}
                  style={{
                    width: '100%',
                    height: '100%',
                    objectFit: 'cover',
                  }}
                />
              ) : (
                <div
                  style={{
                    padding: 18,
                    textAlign: 'center',
                    color: C.textMuted,
                  }}
                >
                  <div
                    style={{
                      fontSize: 34,
                      marginBottom: 8,
                    }}
                  >
                    {preview?.status === 'generating'
                      ? '⏳'
                      : config.emoji}
                  </div>
                  <div style={{ fontSize: 12 }}>
                    {previewStatusText(preview)}
                  </div>
                </div>
              )}
            </div>

            <div
              style={{
                padding: '12px 14px 14px',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 7,
                  marginBottom: 5,
                  fontSize: 14,
                  fontWeight: 700,
                  color: C.textPrimary,
                }}
              >
                <span>{config.emoji}</span>
                <span>{config.title}</span>
              </div>

              <div
                style={{
                  minHeight: 34,
                  fontSize: 11,
                  lineHeight: 1.5,
                  color: C.textSecondary,
                }}
              >
                {config.description}
              </div>

              <div
                style={{
                  marginTop: 8,
                  fontSize: 11,
                  color:
                    preview?.status === 'failed'
                      ? C.danger
                      : preview?.status === 'generated'
                        ? C.success
                        : C.textMuted,
                }}
                title={preview?.last_error || ''}
              >
                {preview?.status === 'failed' &&
                preview.last_error
                  ? `失败：${preview.last_error}`
                  : previewStatusText(preview)}
              </div>
            </div>

            {selected && (
              <div
                style={{
                  position: 'absolute',
                  top: 9,
                  right: 9,
                  width: 27,
                  height: 27,
                  borderRadius: '50%',
                  background: C.primary,
                  color: '#fff',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 15,
                  fontWeight: 700,
                  boxShadow:
                    '0 2px 8px rgba(124,58,237,0.35)',
                }}
              >
                ✓
              </div>
            )}
          </button>
        )
      })}
    </div>
  )
}

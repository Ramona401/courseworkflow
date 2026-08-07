/**
 * LessonDocumentOutline.tsx — 教案目录导航。
 *
 * 支持：
 *   - 当前章节高亮；
 *   - 点击平滑滚动；
 *   - 从目录直接唤起当前章节AI修改；
 *   - 紧凑画布中的抽屉关闭。
 */

import type {
  LessonDocumentSection,
} from './lessonDocumentStructure'

interface LessonDocumentOutlineProps {
  sections: LessonDocumentSection[]
  activeSectionID: string
  disabled?: boolean
  compact?: boolean
  onSelect: (section: LessonDocumentSection) => void
  onRewrite: (section: LessonDocumentSection) => void
  onClose?: () => void
}

export default function LessonDocumentOutline({
  sections,
  activeSectionID,
  disabled = false,
  compact = false,
  onSelect,
  onRewrite,
  onClose,
}: LessonDocumentOutlineProps) {
  const minimumLevel = sections.reduce(
    (minimum, section) => Math.min(minimum, section.level),
    sections[0]?.level || 1,
  )

  return (
    <div style={{
      height: '100%',
      display: 'flex',
      flexDirection: 'column',
      background: '#FFFFFF',
      borderRight: '1px solid #E5E7EB',
      boxSizing: 'border-box',
    }}>
      <div style={{
        padding: compact ? '10px 12px' : '12px 14px',
        borderBottom: '1px solid #F3F4F6',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: '8px',
        flexShrink: 0,
      }}>
        <div>
          <div style={{
            fontSize: '13px',
            fontWeight: 700,
            color: '#374151',
          }}>
            ☰ 教案目录
          </div>
          <div style={{
            marginTop: '2px',
            fontSize: '10px',
            color: '#9CA3AF',
          }}>
            点击章节定位，✨可局部修改
          </div>
        </div>

        {onClose && (
          <button
            type="button"
            onClick={onClose}
            aria-label="关闭目录"
            style={{
              width: '28px',
              height: '28px',
              border: 'none',
              borderRadius: '7px',
              background: '#F3F4F6',
              color: '#6B7280',
              cursor: 'pointer',
              fontSize: '16px',
            }}
          >
            ×
          </button>
        )}
      </div>

      <div style={{
        flex: 1,
        minHeight: 0,
        overflowY: 'auto',
        padding: '8px',
      }}>
        {sections.map(section => {
          const active = section.id === activeSectionID
          const indent = Math.max(
            0,
            section.level - minimumLevel,
          ) * 12

          return (
            <div
              key={section.id}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
                marginBottom: '3px',
                marginLeft: `${indent}px`,
              }}
            >
              <button
                type="button"
                onClick={() => onSelect(section)}
                title={section.headingPath.join(' / ')}
                style={{
                  flex: 1,
                  minWidth: 0,
                  padding: '7px 8px',
                  border: active
                    ? '1px solid rgba(79,123,232,0.28)'
                    : '1px solid transparent',
                  borderRadius: '7px',
                  background: active
                    ? 'rgba(79,123,232,0.09)'
                    : 'transparent',
                  color: active ? '#365FB8' : '#4B5563',
                  fontSize: compact ? '11px' : '12px',
                  fontWeight: active ? 700 : 500,
                  textAlign: 'left',
                  cursor: 'pointer',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {section.title}
              </button>

              <button
                type="button"
                onClick={() => onRewrite(section)}
                disabled={disabled}
                title={
                  disabled
                    ? '当前暂不能使用AI修改'
                    : `让AI修改“${section.title}”`
                }
                style={{
                  width: '28px',
                  height: '28px',
                  flexShrink: 0,
                  borderRadius: '7px',
                  border: '1px solid #E5E7EB',
                  background: disabled ? '#F9FAFB' : '#FFFFFF',
                  color: disabled ? '#D1D5DB' : '#4F7BE8',
                  cursor: disabled ? 'not-allowed' : 'pointer',
                  fontSize: '12px',
                }}
              >
                ✨
              </button>
            </div>
          )
        })}
      </div>
    </div>
  )
}

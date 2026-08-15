/**
 * 全屏/放映课堂智能体自动入场状态。
 *
 * 正常情况下不再展示“开始课堂”按钮：
 *   - 外层在老师打开教学智能体后自动建立真实教师预览会话；
 *   - 会话成功后智能体会先主动问候；
 *   - 只有连接失败或使用时间到期时，才提供恢复/重试动作。
 */

import {
  primaryButtonStyle,
} from './PlatformCoursewareAssistantOverlay.styles'

interface ClassroomEntryNotice {
  kind: 'info' | 'success' | 'error'
  text: string
}

interface PlatformCoursewareAssistantClassroomEntryProps {
  mode: 'minimal' | 'panel'
  classroomMode: boolean
  currentVersion: number
  maximumTurns: number
  starting: boolean
  notice: ClassroomEntryNotice | null
  onRecover: () => void
}

function recoveryLabel(notice: ClassroomEntryNotice | null): string {
  if (
    notice?.kind === 'error'
    && notice.text.includes('使用时间')
  ) {
    return '调整使用时间'
  }

  return '重新连接'
}

export default function PlatformCoursewareAssistantClassroomEntry({
  mode,
  classroomMode,
  currentVersion,
  maximumTurns,
  starting,
  notice,
  onRecover,
}: PlatformCoursewareAssistantClassroomEntryProps) {
  const failed = notice?.kind === 'error'

  if (mode === 'minimal') {
    return (
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-end',
          gap: 8,
          pointerEvents: 'auto',
        }}
      >
        {failed ? (
          <>
            <div
              style={{
                maxWidth: classroomMode ? 420 : 360,
                padding: classroomMode ? '10px 13px' : '8px 10px',
                borderRadius: 13,
                border: '1px solid rgba(239,68,68,0.26)',
                background: 'rgba(254,242,242,0.97)',
                color: '#B91C1C',
                boxShadow: '0 8px 24px rgba(15,23,42,0.12)',
                fontSize: classroomMode ? 14 : 11,
                lineHeight: 1.55,
              }}
            >
              {notice.text}
            </div>

            <button
              type="button"
              onClick={onRecover}
              disabled={starting}
              style={{
                ...primaryButtonStyle(starting, classroomMode),
                minHeight: classroomMode ? 44 : 38,
                borderRadius: 999,
                boxShadow: '0 8px 24px rgba(79,123,232,0.20)',
              }}
            >
              {starting ? '正在重试…' : recoveryLabel(notice)}
            </button>
          </>
        ) : (
          <div
            aria-live="polite"
            style={{
              minWidth: classroomMode ? 250 : 210,
              padding: classroomMode ? '12px 15px' : '10px 12px',
              borderRadius: 16,
              border: '1px solid rgba(129,140,248,0.26)',
              background: 'rgba(238,242,255,0.96)',
              color: '#4338CA',
              boxShadow: '0 10px 28px rgba(30,64,175,0.16)',
              backdropFilter: 'blur(10px)',
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                fontSize: classroomMode ? 15 : 12,
                fontWeight: 850,
              }}
            >
              <span
                aria-hidden="true"
                style={{
                  width: 10,
                  height: 10,
                  borderRadius: '50%',
                  background: '#6366F1',
                  boxShadow: '0 0 0 6px rgba(99,102,241,0.10)',
                  animation: 'tednaAssistantClassroomReady 1.15s ease-in-out infinite',
                }}
              />
              智能体正在进入课堂
            </div>

            <div
              style={{
                marginTop: 5,
                color: '#6366F1',
                fontSize: classroomMode ? 12.5 : 10.5,
                lineHeight: 1.55,
              }}
            >
              会话建立后会先主动打招呼，再等待你的语音输入。
            </div>
          </div>
        )}

        <ClassroomReadyAnimation />
      </div>
    )
  }

  return (
    <div
      style={{
        padding: classroomMode ? 18 : 14,
      }}
    >
      <div
        style={{
          padding: classroomMode ? 18 : 14,
          borderRadius: classroomMode ? 16 : 13,
          border: failed
            ? '1px solid #FECACA'
            : '1px solid #C7D2FE',
          background: failed
            ? '#FEF2F2'
            : 'linear-gradient(145deg, #F8FAFF, #EEF2FF)',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 10,
          }}
        >
          <span
            aria-hidden="true"
            style={{
              width: classroomMode ? 12 : 10,
              height: classroomMode ? 12 : 10,
              borderRadius: '50%',
              background: failed ? '#EF4444' : '#6366F1',
              boxShadow: failed
                ? '0 0 0 5px rgba(239,68,68,0.08)'
                : '0 0 0 6px rgba(99,102,241,0.10)',
              animation: failed
                ? 'none'
                : 'tednaAssistantClassroomReady 1.15s ease-in-out infinite',
            }}
          />

          <div
            style={{
              color: failed ? '#991B1B' : '#1E293B',
              fontSize: classroomMode ? 20 : 16,
              fontWeight: 850,
              lineHeight: 1.4,
            }}
          >
            {failed
              ? '课堂连接没有完成'
              : '智能体正在进入课堂'}
          </div>
        </div>

        <div
          style={{
            marginTop: 8,
            color: failed ? '#B91C1C' : '#475569',
            fontSize: classroomMode ? 15 : 12,
            lineHeight: 1.7,
          }}
        >
          {failed
            ? '请根据上方提示处理后重新连接。'
            : '无需点击开始。连接完成后，智能体会先主动问候，然后进入语音聆听状态。'}
        </div>

        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 12,
            marginTop: classroomMode ? 16 : 12,
          }}
        >
          <div
            style={{
              color: '#64748B',
              fontSize: classroomMode ? 13 : 11,
            }}
          >
            V{currentVersion} · 最多{maximumTurns}轮
          </div>

          {failed && (
            <button
              type="button"
              onClick={onRecover}
              disabled={starting}
              style={primaryButtonStyle(starting, classroomMode)}
            >
              {starting ? '正在重试…' : recoveryLabel(notice)}
            </button>
          )}
        </div>
      </div>

      <ClassroomReadyAnimation />
    </div>
  )
}

function ClassroomReadyAnimation() {
  return (
    <style>{`
      @keyframes tednaAssistantClassroomReady {
        0%, 100% {
          transform: scale(0.82);
          opacity: 0.55;
        }
        50% {
          transform: scale(1.08);
          opacity: 1;
        }
      }
    `}</style>
  )
}

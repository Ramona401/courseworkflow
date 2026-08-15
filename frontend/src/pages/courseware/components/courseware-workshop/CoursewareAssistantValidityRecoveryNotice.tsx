/**
 * 教学智能体使用时间到期后的教师恢复提示。
 *
 * 单一职责：
 *   - 用与教学智能体现有视觉一致的恢复卡替代浏览器原生按钮；
 *   - 明确告诉教师“为什么不能开始”和“下一步做什么”；
 *   - 只上抛“调整使用时间”意图，不负责导航、API或部署状态判断。
 */

import type { CSSProperties } from 'react'

interface CoursewareAssistantValidityRecoveryNoticeProps {
  classroomMode: boolean
  onAdjust: () => void
}

export default function CoursewareAssistantValidityRecoveryNotice({
  classroomMode,
  onAdjust,
}: CoursewareAssistantValidityRecoveryNoticeProps) {
  const compact = !classroomMode

  return (
    <div
      role="alert"
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: classroomMode ? 16 : 10,
        padding: classroomMode ? '12px 16px' : '8px 11px',
        borderBottom: '1px solid #FED7AA',
        background: 'linear-gradient(135deg, #FFF7ED 0%, #FFF1F2 100%)',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: classroomMode ? 11 : 7,
          minWidth: 0,
          flex: 1,
        }}
      >
        <span
          aria-hidden="true"
          style={{
            width: classroomMode ? 34 : 24,
            height: classroomMode ? 34 : 24,
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            flex: '0 0 auto',
            borderRadius: 999,
            background: '#FFEDD5',
            color: '#C2410C',
            fontSize: classroomMode ? 18 : 12,
            boxShadow: 'inset 0 0 0 1px rgba(234,88,12,0.12)',
          }}
        >
          ⏱
        </span>

        <div style={{ minWidth: 0 }}>
          <div
            style={{
              color: '#9A3412',
              fontSize: classroomMode ? 15 : 10,
              fontWeight: 850,
              lineHeight: 1.35,
            }}
          >
            课堂使用时间已到
          </div>

          <div
            style={{
              marginTop: compact ? 1 : 3,
              color: '#B45309',
              fontSize: classroomMode ? 13 : 8.8,
              lineHeight: 1.5,
            }}
          >
            延长当前页面的使用时间后，就可以继续开始新的学习互动。
          </div>
        </div>
      </div>

      <button
        type="button"
        onClick={onAdjust}
        style={recoveryButtonStyle(classroomMode)}
      >
        <span>调整使用时间</span>
        <span aria-hidden="true" style={{ fontSize: classroomMode ? 17 : 11 }}>
          →
        </span>
      </button>
    </div>
  )
}

function recoveryButtonStyle(classroomMode: boolean): CSSProperties {
  return {
    minHeight: classroomMode ? 42 : 30,
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: classroomMode ? 7 : 4,
    flex: '0 0 auto',
    padding: classroomMode ? '8px 14px' : '6px 10px',
    borderRadius: classroomMode ? 11 : 8,
    border: '1px solid rgba(79,123,232,0.24)',
    background: '#4F7BE8',
    color: '#FFFFFF',
    boxShadow: '0 4px 12px rgba(79,123,232,0.18)',
    fontFamily: 'inherit',
    fontSize: classroomMode ? 14 : 9.5,
    fontWeight: 850,
    lineHeight: 1,
    whiteSpace: 'nowrap',
    cursor: 'pointer',
  }
}

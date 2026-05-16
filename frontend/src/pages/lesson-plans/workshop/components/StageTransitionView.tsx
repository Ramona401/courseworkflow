/**
 * StageTransitionView.tsx — 阶段切换叙事过渡动画
 *
 * 从 WorkshopTransitionComponents.tsx 拆分
 * 三步动画: 整理结论 → 准备背景 → 唤醒角色
 */

import { C } from './workshopConstants'

interface StageTransitionViewProps {
  currentStageName: string
  nextStageName: string
  nextStageRole: string
  step: number
}

export function StageTransitionView({
  currentStageName, nextStageName, nextStageRole, step,
}: StageTransitionViewProps) {
  const steps = [
    { icon: '📋', text: `正在整理「${currentStageName}」的核心结论...` },
    { icon: '🔗', text: `为「${nextStageName}」阶段准备背景信息...` },
    { icon: '✨', text: `正在唤醒${nextStageRole}...` },
  ]

  return (
    <div style={{
      position: 'absolute', inset: 0,
      background: 'rgba(250,251,252,0.96)', backdropFilter: 'blur(2px)',
      display: 'flex', flexDirection: 'column',
      alignItems: 'center', justifyContent: 'center',
      zIndex: 100,
    }}>
      <div style={{
        padding: '40px 48px', background: C.card, borderRadius: '20px',
        boxShadow: '0 8px 40px rgba(0,0,0,0.08)',
        border: `1px solid ${C.border}`, minWidth: '340px',
      }}>
        <div style={{
          fontSize: '11px', fontWeight: 700, color: C.textMuted,
          textTransform: 'uppercase', letterSpacing: '1.5px',
          marginBottom: '22px', textAlign: 'center',
        }}>
          阶段交接中
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          {steps.map((s, i) => {
            const isDone    = step > i
            const isCurrent = step === i
            const isPending = step < i
            return (
              <div key={i} style={{
                display: 'flex', alignItems: 'center', gap: '14px',
                opacity: isPending ? 0.25 : 1,
                transform: isCurrent ? 'translateX(2px)' : 'none',
                transition: 'all 450ms cubic-bezier(0.34,1.56,0.64,1)',
              }}>
                <div style={{
                  width: '30px', height: '30px', borderRadius: '50%', flexShrink: 0,
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  fontSize: isDone ? '13px' : '14px',
                  background: isDone
                    ? 'linear-gradient(135deg, #10B981, #34D399)'
                    : isCurrent ? C.primaryLight : '#F3F4F6',
                  border: isDone
                    ? '1px solid #6EE7B7'
                    : isCurrent ? `1.5px solid ${C.primary}` : '1px solid transparent',
                  color: isDone ? '#fff' : C.text,
                  transition: 'all 400ms ease',
                  boxShadow: isCurrent ? `0 0 0 4px ${C.primaryLight}` : 'none',
                }}>
                  {isDone ? '✓' : isCurrent ? (
                    <div style={{
                      width: '11px', height: '11px',
                      border: `2px solid ${C.primary}`, borderTopColor: 'transparent',
                      borderRadius: '50%', animation: 'tranSpin 0.7s linear infinite',
                    }} />
                  ) : s.icon}
                </div>
                <span style={{
                  fontSize: '14px', lineHeight: 1.5,
                  color: isDone ? C.success : isCurrent ? C.text : C.textMuted,
                  fontWeight: isCurrent ? 600 : isDone ? 500 : 400,
                  transition: 'all 400ms ease',
                }}>
                  {s.text}
                </span>
              </div>
            )
          })}
        </div>
        <style>{`@keyframes tranSpin { to { transform: rotate(360deg); } }`}</style>
      </div>
    </div>
  )
}

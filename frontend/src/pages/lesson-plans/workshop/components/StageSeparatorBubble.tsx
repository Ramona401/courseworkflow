/**
 * StageSeparatorBubble.tsx — 对话流中的阶段分隔符
 *
 * 从 WorkshopTransitionComponents.tsx 拆分
 * 三段式: 顶部收束条(上一阶段完成) + 中间主徽章(进入新阶段) + 底部角色介绍
 */

import { C, STAGE_CODE_EMOJI } from './workshopConstants'

interface StageSeparatorBubbleProps {
  /** 即将进入的阶段名 */
  stageName: string
  /** 即将进入的阶段角色 */
  aiRole: string
  /** 上一阶段名,用于顶部收束条(不传则不显示) */
  prevStageName?: string
  /** 即将进入的阶段代码,用于匹配图标(不传用✨) */
  nextStageCode?: string
}

export function StageSeparatorBubble({
  stageName, aiRole, prevStageName, nextStageCode,
}: StageSeparatorBubbleProps) {
  const stageIcon = nextStageCode ? (STAGE_CODE_EMOJI[nextStageCode] || '✨') : '✨'

  return (
    <div style={{
      margin: '28px 0 20px',
      padding: '0 4px',
      display: 'flex', flexDirection: 'column', gap: '10px',
    }}>
      {/* 顶部收束条 */}
      {prevStageName && (
        <div style={{
          display: 'flex', alignItems: 'center', gap: '10px',
          opacity: 0.7,
        }}>
          <div style={{
            flex: 1, height: '1px',
            background: `linear-gradient(to right, transparent, ${C.success}40)`,
          }} />
          <div style={{
            display: 'flex', alignItems: 'center', gap: '6px',
            padding: '3px 12px', borderRadius: '12px',
            background: 'rgba(16,185,129,0.08)',
            border: '1px solid rgba(16,185,129,0.15)',
            fontSize: '11px', color: C.success, fontWeight: 500,
            whiteSpace: 'nowrap',
          }}>
            <span>✅</span>
            <span>{prevStageName} 阶段已完成</span>
          </div>
          <div style={{
            flex: 1, height: '1px',
            background: `linear-gradient(to left, transparent, ${C.success}40)`,
          }} />
        </div>
      )}

      {/* 中间主徽章 */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: '14px',
        padding: '0 4px',
      }}>
        <div style={{
          flex: 1, height: '1px',
          background: `linear-gradient(to right, transparent, ${C.border})`,
        }} />
        <div style={{
          padding: '8px 20px', borderRadius: '22px',
          background: 'linear-gradient(135deg, #4F7BE8, #818CF8)',
          color: '#fff', fontSize: '13px', fontWeight: 700,
          whiteSpace: 'nowrap',
          boxShadow: '0 4px 14px rgba(79,123,232,0.32)',
          display: 'flex', alignItems: 'center', gap: '8px',
          animation: 'stageSepFadeIn 500ms ease-out',
        }}>
          <span style={{ fontSize: '15px' }}>{stageIcon}</span>
          <span>进入{stageName}</span>
        </div>
        <div style={{
          flex: 1, height: '1px',
          background: `linear-gradient(to left, transparent, ${C.border})`,
        }} />
      </div>

      {/* 底部角色介绍 */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        gap: '6px',
        fontSize: '12px', color: C.textSec,
        padding: '0 4px',
      }}>
        <span style={{
          display: 'inline-flex', alignItems: 'center', gap: '4px',
          padding: '3px 10px', borderRadius: '10px',
          background: C.primaryLight,
          color: C.primary, fontWeight: 600,
        }}>
          <span>🎭</span>
          <span>{aiRole}</span>
        </span>
        <span style={{ color: C.textMuted }}>即将协助你完成本阶段</span>
      </div>

      <style>{`
        @keyframes stageSepFadeIn {
          from { opacity: 0; transform: translateY(-4px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </div>
  )
}

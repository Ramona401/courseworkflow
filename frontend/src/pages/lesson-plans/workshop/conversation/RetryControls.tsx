/**
 * RetryControls.tsx — 对话模式「🔄 重新回答」按钮（子轮二从 ConversationModePage 抽出）
 *
 * 抽出动机：ConversationModePage.tsx 此前 662 行超 600 红线（已记账待拆）；
 * 子轮二接入 turnID/三层超时又要加代码，借此把重试按钮这块纯渲染逻辑抽成独立组件，
 * 既解决红线、又让页面更聚焦编排。
 *
 * 职责边界：纯展示 + 点击回调冒泡，不持任何业务状态。
 * 是否显示由页面用 canRetry 控制（本组件只负责"显示时长什么样"）。
 * 与建议芯片（ConversationChipRow）独立渲染——重试不进芯片协议（拍点二A）。
 */
import { C } from '../components/workshopConstants'

interface RetryControlsProps {
  /** 点击「重新回答」回调 */
  onRetry: () => void
}

/**
 * 重试按钮行 —— 渲染在芯片行上方、最后一条 AI 回复下方
 */
export default function RetryControls({ onRetry }: RetryControlsProps) {
  return (
    <div style={{ display: 'flex', margin: '2px 0 4px 42px' }}>
      <button
        onClick={onRetry}
        style={{
          display: 'flex', alignItems: 'center', gap: '5px',
          padding: '5px 12px', borderRadius: '16px', fontSize: '12px', fontWeight: 500,
          border: `1px solid ${C.border}`, background: C.card, color: C.textMuted,
          cursor: 'pointer', transition: 'all 150ms ease',
        }}
        onMouseEnter={e => {
          (e.currentTarget as HTMLButtonElement).style.color = C.primary
          ;(e.currentTarget as HTMLButtonElement).style.borderColor = C.primary
        }}
        onMouseLeave={e => {
          (e.currentTarget as HTMLButtonElement).style.color = C.textMuted
          ;(e.currentTarget as HTMLButtonElement).style.borderColor = C.border
        }}
      >
        🔄 重新回答
      </button>
    </div>
  )
}

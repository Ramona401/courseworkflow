/**
 * ConversationChipRow.tsx — 对话模式建议芯片行（迭代3.5 B-2 拆分批次）
 *
 * 从 ConversationModePage.tsx 抽出的芯片行渲染组件（主页面逼近600行红线，
 * B-2 加动态芯片状态前先把这块纯渲染逻辑拆出来）。
 *
 * 职责边界：纯展示 + 点击回调冒泡，不持任何业务状态、不区分芯片来源——
 * 剧本常量芯片与 AI 动态芯片（B-2 起）走同一渲染路径，样式完全一致，
 * 老师无感知差异（设计文档2.3：芯片永远是建议不是必选）。
 */
import { C } from '../components/workshopConstants'
import type { ChipDef } from './conversationScript'

interface ConversationChipRowProps {
  /** 当前应显示的芯片列表（空数组时不渲染任何内容） */
  chips: ChipDef[]
  /** 芯片点击回调（页面侧经 dispatchChip 分发） */
  onChipClick: (chip: ChipDef) => void
}

/**
 * 建议芯片行 —— 渲染在最后一条 AI 回复的下方
 */
export default function ConversationChipRow({ chips, onChipClick }: ConversationChipRowProps) {
  if (chips.length === 0) return null
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', margin: '2px 0 8px 42px' }}>
      {chips.map(chip => (
        <button
          key={chip.id}
          onClick={() => onChipClick(chip)}
          style={{
            padding: '7px 15px', borderRadius: '18px', fontSize: '13px', fontWeight: chip.highlight ? 600 : 500,
            border: chip.highlight ? 'none' : `1px solid ${C.border}`,
            background: chip.highlight ? `linear-gradient(135deg, ${C.primary}, #818CF8)` : C.card,
            color: chip.highlight ? '#fff' : C.textSec,
            cursor: 'pointer', transition: 'all 150ms ease', whiteSpace: 'nowrap',
            boxShadow: chip.highlight ? '0 3px 10px rgba(79,123,232,0.3)' : '0 1px 4px rgba(0,0,0,0.04)',
          }}
          onMouseEnter={e => { if (!chip.highlight) (e.currentTarget as HTMLButtonElement).style.borderColor = C.primary }}
          onMouseLeave={e => { if (!chip.highlight) (e.currentTarget as HTMLButtonElement).style.borderColor = C.border }}
        >
          {/* 动态芯片的 emoji 可能为空串，为空时不渲染前导空格 */}
          {chip.emoji ? `${chip.emoji} ${chip.label}` : chip.label}
        </button>
      ))}
    </div>
  )
}

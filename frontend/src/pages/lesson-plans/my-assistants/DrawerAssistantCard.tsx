/**
 * DrawerAssistantCard.tsx — 「现成助手」抽屉里的单张紧凑卡片
 *
 * 从 MyAssistantsPage 抽离独立成文件(原内联子组件),动机:
 *   - MyAssistantsPage 因新增"编辑/删除按钮 + share_policy 徽章"逼近 600 行红线
 *   - 卡片本身是自洽 UI 单元,抽出后主页面更聚焦于编排逻辑
 *
 * 本次新增(配合 share_policy 全套):
 *   1. ✏️ 编辑按钮:按 item.can_edit 显示——点击回调 onEdit,父组件打开 AssistantEditModal(edit 模式)
 *   2. 🗑️ 删除按钮:按 item.can_delete 显示——点击回调 onDelete,父组件负责二次确认 + 调 deleteAssistant
 *   3. ➕ 复制按钮:按 item.can_fork 显隐——不可复制时直接不显示(个人助手显示"已是你的"),
 *        不向用户解释"为什么不能",界面只呈现用户当前能做的操作。
 *   4. 🔍 分析按钮:按 item.can_view_prompt 显隐——不可分析时同样不显示。
 *
 *   文案原则:只告诉用户能怎么用,不暴露 share_policy 等后台机制术语。
 *
 * 权限说明:can_edit/can_delete/can_fork 全由后端按当前用户算好下发,前端只按布尔显隐。
 *   前端显隐是"体验优化"(不让用户点到注定失败的操作),最终拦截仍在后端。
 *
 * 职责边界:本卡片只做"参考/复制/编辑/删除/分析",不放"用这个→跳工坊"(用助手在备课工坊)。
 */

import { useState } from 'react'
import {
  ASSISTANT_SOURCE_LABELS,
  ASSISTANT_SOURCE_EMOJI,
  type AIAssistantListItem,
} from '@/api/ai-assistants'

/* 样式常量(与 MyAssistantsPage 同源最小集) */
const C = {
  primary:        '#4F7BE8',
  accent:         '#F59E0B',
  danger:         '#EF4444',
  text:           '#1F2937',
  textSec:        '#6B7280',
  textMuted:      '#9CA3AF',
  border:         '#F3F4F6',
  systemAccent:   '#4F7BE8',
  groupAccent:    '#F59E0B',
  personalAccent: '#10B981',
}

/** 小按钮样式(从原 MyAssistantsPage 原样迁移,供本卡片内部使用) */
export function miniBtn(color: string, disabled?: boolean): React.CSSProperties {
  return {
    padding: '3px 9px', borderRadius: '5px',
    border: `1px solid ${color}`, background: '#fff', color,
    fontSize: '11px', fontWeight: 500,
    cursor: disabled ? 'not-allowed' : 'pointer',
    opacity: disabled ? 0.5 : 1,
    whiteSpace: 'nowrap',
  }
}

export interface DrawerCardProps {
  item: AIAssistantListItem
  forking: boolean
  analyzing: boolean
  deleting: boolean
  highlight?: boolean
  onAnalyze: () => void
  onFork: () => void
  onEdit: () => void
  onDelete: () => void
}

export default function DrawerAssistantCard({
  item, forking, analyzing, deleting, highlight,
  onAnalyze, onFork, onEdit, onDelete,
}: DrawerCardProps) {
  const [expanded, setExpanded] = useState(false)
  const accent =
    item.source === 'system'   ? C.systemAccent :
    item.source === 'group'    ? C.groupAccent  :
                                 C.personalAccent

  return (
    <div style={{
      padding: '9px 11px', borderRadius: '8px',
      border: `1px solid ${highlight ? `${accent}66` : C.border}`,
      borderLeft: `3px solid ${accent}`,
      background: highlight ? `${accent}0D` : '#fff',
      marginBottom: '7px',
    }}>
      {/* 标题行(紧凑:名字 + 来源徽章 + 分享策略徽章) */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '5px', flexWrap: 'wrap' }}>
        <span style={{ fontSize: '12.5px', fontWeight: 600, color: C.text }}>
          {item.avatar_emoji} {item.name}
        </span>
        <span style={{
          padding: '1px 5px', borderRadius: '4px', fontSize: '9.5px', fontWeight: 600,
          background: `${accent}1A`, color: accent,
        }}>
          {ASSISTANT_SOURCE_EMOJI[item.source]} {ASSISTANT_SOURCE_LABELS[item.source]}
        </span>
        {item.subject && (
          <span style={{ fontSize: '9.5px', color: C.textMuted }}>📚 {item.subject}</span>
        )}
      </div>

      {/* 描述(默认折叠成1行,点展开看全) */}
      {item.description && (
        <div
          style={{
            fontSize: '11px', color: C.textSec, marginTop: '4px', lineHeight: 1.55,
            overflow: 'hidden',
            display: '-webkit-box',
            WebkitLineClamp: expanded ? 99 : 1,
            WebkitBoxOrient: 'vertical' as const,
            cursor: 'pointer',
          }}
          onClick={() => setExpanded(v => !v)}
          title={expanded ? '点击收起' : '点击展开'}
        >
          {item.description}
        </div>
      )}

      {/* 操作行 */}
      <div style={{ display: 'flex', gap: '6px', marginTop: '7px', flexWrap: 'wrap' }}>
        {/* 丢给 AI 分析:仅在当前用户可读取原文时显示;不可用时不显示(不向用户解释机制) */}
        {item.can_view_prompt && (
          <button
            onClick={onAnalyze}
            disabled={analyzing}
            style={miniBtn(C.accent, analyzing)}
            title="把这个助手的完整设定丢给左侧 AI,让它帮你分析、讨论你可以从哪些角度补充"
          >{analyzing ? '读取中…' : '🔍 丢给 AI 分析'}</button>
        )}

        {/* 复制到我的:可复制时显示按钮;个人助手提示"已是你的";其余情况不显示(不解释机制) */}
        {item.can_fork ? (
          <button
            onClick={onFork}
            disabled={forking}
            style={miniBtn(C.primary, forking)}
          >{forking ? '复制中…' : '➕ 复制到我的'}</button>
        ) : item.source === 'personal' ? (
          <span style={{ fontSize: '10px', color: C.textMuted, alignSelf: 'center' }}>
            ✓ 已是你的
          </span>
        ) : null}

        {/* 编辑:按 can_edit 显示 */}
        {item.can_edit && (
          <button
            onClick={onEdit}
            style={miniBtn(C.textSec)}
            title="编辑这个助手的名称、提示词、适用场景等"
          >✏️ 编辑</button>
        )}

        {/* 删除:按 can_delete 显示(父组件负责二次确认) */}
        {item.can_delete && (
          <button
            onClick={onDelete}
            disabled={deleting}
            style={miniBtn(C.danger, deleting)}
            title="删除这个助手(不可恢复)"
          >{deleting ? '删除中…' : '🗑️ 删除'}</button>
        )}
      </div>
    </div>
  )
}

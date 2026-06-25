/**
 * ConversationTopBar.tsx — 对话模式顶栏 + 断线提示条（迭代3.5 A2-4 拆分）
 *
 * 从 ConversationModePage.tsx 抽出的页面顶部区域，包含两块：
 *   1. 顶栏：标题 + 课本关联徽标 + SSE连接状态圆点 + 助手指示器 + 窄屏Tab切换 + 专家模式/退出按钮；
 *   2. 断线提示条：reconnecting 显示重连中、disconnected 显示手动重连按钮。
 *
 * 纯展示+回调组件，不持任何业务状态，全部经 props 与页面交互。
 *
 * 助手轻量选择入口 Phase 1：新增「当前助手」指示器 chip（assistantLabel + onAssistantClick）。
 *   - 安静展示当前生效的助手名，点击由父组件唤起切换面板（面板浮层在父组件内渲染、定位）。
 *   - 两个 prop 均可选：未注入则不显示指示器（专家模式等无需该入口的场景自动隐藏）。
 */
import { C } from '../components/workshopConstants'
import type { SSEConnectionState } from '@/api/lesson-plans'

/** 顶栏组件 Props */
export interface ConversationTopBarProps {
  /** 教案标题（无教案时显示"备课对话"由页面侧兜底） */
  title: string
  /** 本会话已关联课本页数（>0 显示 📷课本×N 徽标） */
  textbookCount: number
  /** SSE 连接状态（驱动状态圆点与断线提示条） */
  sseState: SSEConnectionState
  /** 是否窄屏（<1024px，显示对话/教案Tab） */
  isNarrow: boolean
  /** 窄屏当前Tab */
  narrowTab: 'chat' | 'canvas'
  /** 窄屏Tab切换回调 */
  onNarrowTabChange: (tab: 'chat' | 'canvas') => void
  /** 切换到专家模式（未注入则不显示按钮） */
  onSwitchMode?: () => void
  /** 退出备课 */
  onExit: () => void
  /** 手动重连（断线提示条按钮） */
  onReconnect: () => void
  /**
   * 当前生效的助手显示名（助手轻量选择入口 Phase 1）。
   * 未注入则不显示助手指示器；典型值如「初中语文备课助手」「系统默认」。
   */
  assistantLabel?: string
  /** 点击助手指示器的回调（由父组件唤起切换面板）。与 assistantLabel 同时注入才显示指示器。 */
  onAssistantClick?: () => void
}

/**
 * 对话模式顶栏组件
 */
export default function ConversationTopBar({
  title, textbookCount, sseState, isNarrow, narrowTab,
  onNarrowTabChange, onSwitchMode, onExit, onReconnect,
  assistantLabel, onAssistantClick,
}: ConversationTopBarProps) {
  const showAssistant = !!assistantLabel && !!onAssistantClick
  return (
    <>
      {/* ===== 顶栏：标题 + 连接状态 + 助手指示器 + 模式切换/退出 ===== */}
      <div style={{ height: '46px', flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 18px', borderBottom: `1px solid ${C.border}`, background: C.card }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', minWidth: 0 }}>
          <span style={{ fontSize: '15px' }}>💬</span>
          <span style={{ fontSize: '14px', fontWeight: 700, color: C.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{title}</span>
          {textbookCount > 0 && (
            <span title="本次备课已关联课本页，AI会参考课文原文" style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '8px', background: 'rgba(16,185,129,0.1)', color: '#059669', flexShrink: 0, whiteSpace: 'nowrap' }}>
              📷 课本×{textbookCount}
            </span>
          )}
          <div title={sseState === 'connected' ? '连接正常' : sseState === 'reconnecting' ? '重连中…' : '连接断开'}
            style={{ width: '7px', height: '7px', borderRadius: '50%', flexShrink: 0, background: sseState === 'connected' ? '#10B981' : sseState === 'reconnecting' ? '#F59E0B' : '#EF4444' }} />
          {/* 助手指示器（Phase 1）：安静展示当前助手，点击由父组件唤起切换面板 */}
          {showAssistant && (
            <button
              onClick={onAssistantClick}
              title="点击切换本学科的备课助手（下一轮起生效）"
              style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '11px', padding: '2px 8px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'rgba(99,102,241,0.06)', color: C.primary, flexShrink: 0, whiteSpace: 'nowrap', cursor: 'pointer', maxWidth: '200px' }}
            >
              <span style={{ flexShrink: 0 }}>🎓</span>
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{assistantLabel}</span>
              <span style={{ flexShrink: 0, opacity: 0.7 }}>▾</span>
            </button>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexShrink: 0 }}>
          {/* 窄屏：对话/教案 Tab 切换 */}
          {isNarrow && (
            <div style={{ display: 'flex', gap: '2px', background: '#F3F4F6', borderRadius: '8px', padding: '2px' }}>
              {([['chat', '💬 对话'], ['canvas', '📄 教案']] as const).map(([k, label]) => (
                <button key={k} onClick={() => onNarrowTabChange(k)}
                  style={{ padding: '4px 12px', borderRadius: '6px', border: 'none', fontSize: '12px', fontWeight: narrowTab === k ? 600 : 400, background: narrowTab === k ? C.card : 'transparent', color: narrowTab === k ? C.primary : C.textSec, cursor: 'pointer' }}>
                  {label}
                </button>
              ))}
            </div>
          )}
          {onSwitchMode && (
            <button onClick={onSwitchMode} title="切换到原版阶段式界面（专家模式），同一教案随时互切"
              style={{ padding: '5px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '12px', color: C.textSec, cursor: 'pointer' }}>
              ⚙ 专家模式
            </button>
          )}
          <button onClick={onExit} title="退出备课，教案自动保存为草稿"
            style={{ padding: '5px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '12px', color: C.textSec, cursor: 'pointer' }}>
            🚪 退出
          </button>
        </div>
      </div>

      {/* ===== 断线提示条 ===== */}
      {sseState !== 'connected' && (
        <div style={{ padding: '6px 16px', textAlign: 'center', fontSize: '12px', fontWeight: 500, flexShrink: 0, background: sseState === 'reconnecting' ? 'rgba(245,158,11,0.08)' : 'rgba(239,68,68,0.08)', color: sseState === 'reconnecting' ? '#92400E' : '#991B1B', borderBottom: `1px solid ${C.border}` }}>
          {sseState === 'reconnecting' ? '网络连接中断，正在尝试重新连接…' : (
            <>网络连接已断开 <button onClick={onReconnect} style={{ marginLeft: '8px', padding: '2px 10px', borderRadius: '10px', border: '1px solid rgba(239,68,68,0.4)', background: 'transparent', fontSize: '12px', color: '#DC2626', cursor: 'pointer' }}>点击重连</button></>
          )}
        </div>
      )}
    </>
  )
}

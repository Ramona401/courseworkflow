/**
 * SharePolicyPicker.tsx — 分享权限策略(share_policy)选择器(可复用)
 *
 * 抽取动机:
 *   SaveAssistantModal(发布时选策略)与 AssistantEditModal(编辑时改策略)需要完全一致的
 *   三档选择器 UI + "选 open 弹黄提醒"逻辑。抽成独立组件:
 *     - 消除两处重复的 JSX(各 ~40 行)
 *     - 让 AssistantEditModal 挂载本组件只需几行,避免文件突破 600 行红线
 *     - 策略文案/样式集中一处维护,改动不发散
 *
 * share_policy 三档(对应后端 models.SharePolicyXxx / api 层常量):
 *   🤝 仅可用(use_only,默认): 别人只能用,不能复制带走,也不能被非维护者改(保护标准与产权)
 *   🔓 可复制(open):          别人可以用,也可以复制一份到自己名下修改
 *   🔒 仅自己(locked):        只有作者本人/admin 能看到和使用(挂共享位但实际私有)
 *
 * 使用约定:
 *   - 仅在【共享场景】(教研组/全校/系统助手)挂载本组件;personal 助手不挂(无意义)
 *   - 受控组件:value + onChange 由父组件持有 state
 *   - 选 'open' 时组件内部自动展示黄色提醒,父组件无需关心
 *
 * Props:
 *   value     - 当前选中的 share_policy
 *   onChange  - 选择变化回调
 *   label?    - 顶部标签文案(默认"别人能怎么用")
 *   disabled? - 是否只读(预留;当前两处调用都可改,默认 false)
 */

import {
  SHARE_POLICY_LABELS,
  SHARE_POLICY_EMOJI,
  SHARE_POLICY_HINTS,
  DEFAULT_SHARE_POLICY,
  type AssistantSharePolicy,
} from '@/api/ai-assistants'

/* 样式常量(与 SaveAssistantModal/AssistantEditModal 的 C 同源,避免跨文件耦合此处自带一份最小集) */
const C = {
  accent:    '#F59E0B',
  text:      '#1F2937',
  textSec:   '#6B7280',
  textMuted: '#9CA3AF',
  border:    '#F3F4F6',
  danger:    '#EF4444',
}

/** 三档展示顺序:默认 use_only 居首(最保护) */
const SHARE_POLICY_ORDER: AssistantSharePolicy[] = ['use_only', 'open', 'locked']

export interface SharePolicyPickerProps {
  value: AssistantSharePolicy
  onChange: (next: AssistantSharePolicy) => void
  label?: string
  disabled?: boolean
}

export default function SharePolicyPicker(props: SharePolicyPickerProps) {
  const { value, onChange, label = '别人能怎么用', disabled = false } = props

  return (
    <div style={{ marginBottom: '16px' }}>
      <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '4px' }}>
        {label} <span style={{ color: C.danger }}>*</span>
        <span style={{ color: C.textMuted, fontWeight: 400 }}> (决定别人能不能复制、能不能改)</span>
      </label>

      <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', marginTop: '4px' }}>
        {SHARE_POLICY_ORDER.map(p => {
          const checked = value === p
          return (
            <label
              key={p}
              style={{
                display: 'flex', alignItems: 'flex-start', gap: '8px',
                padding: '10px 12px', borderRadius: '8px',
                border: `1.5px solid ${checked ? C.accent : C.border}`,
                background: checked ? 'rgba(245,158,11,0.06)' : '#fff',
                cursor: disabled ? 'not-allowed' : 'pointer',
                opacity: disabled && !checked ? 0.5 : 1,
              }}
            >
              <input
                type="radio"
                name="share_policy_picker"
                checked={checked}
                disabled={disabled}
                onChange={() => { if (!disabled) onChange(p) }}
                style={{ cursor: disabled ? 'not-allowed' : 'pointer', accentColor: C.accent, marginTop: '2px' }}
              />
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>
                  {SHARE_POLICY_EMOJI[p]} {SHARE_POLICY_LABELS[p]}
                  {p === DEFAULT_SHARE_POLICY && (
                    <span style={{ fontSize: '10px', fontWeight: 400, color: C.textMuted }}> (默认 · 推荐)</span>
                  )}
                </div>
                <div style={{ fontSize: '11px', color: C.textSec, marginTop: '2px', lineHeight: 1.5 }}>
                  {SHARE_POLICY_HINTS[p]}
                </div>
              </div>
            </label>
          )
        })}
      </div>

      {/* 选"可复制"时的黄色提醒 */}
      {value === 'open' && (
        <div style={{
          marginTop: '8px', padding: '9px 12px', borderRadius: '8px',
          background: 'rgba(245,158,11,0.1)', border: '1px solid rgba(245,158,11,0.3)',
          color: '#92400E', fontSize: '12px', lineHeight: 1.6,
        }}>
          ⚠️ 选了「可复制」后,任何能看到这个助手的老师都可以复制一份带走并自行修改。
          如果这是你想保护的标准或产权,建议改回「仅可用」。
        </div>
      )}
    </div>
  )
}

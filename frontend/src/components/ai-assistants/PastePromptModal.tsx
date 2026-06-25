/**
 * PastePromptModal.tsx — 「粘贴提示词」入口弹窗(提示词工坊 阶段A)
 *
 * 场景:
 *   老师手上已有一段写好的提示词(从别处拷来的、自己琢磨的),不想跟 AI 从头聊,
 *   想直接粘进来用。本弹窗就是这个"快速通道"。
 *
 * 两个出口(对齐 Yuhan 拍板:一个入口满足两种人):
 *   1. 「✓ 直接保存为助手」—— 把粘贴内容原样当草稿,交给父组件走 SaveAssistantModal 落库。
 *                            AI 不介入,老师粘什么存什么。
 *   2. 「💬 让 AI 帮我改改」—— 把粘贴内容交给父组件注入对话画布,老师继续跟 AI 优化。
 *                            复用 designer 的 injectedInput 能力。
 *
 * 职责边界:
 *   本弹窗只负责"收集粘贴的文字 + 让老师选出口",不自己落库、不自己调 AI。
 *   两个出口都通过回调把文字交还父组件(MyAssistantsPage)处理,保持纯受控、易测试。
 *
 * Props 契约:
 *   open        - 是否显示
 *   onClose     - 关闭回调
 *   onSaveDirect- 「直接保存」回调,传回粘贴的文字(父组件据此打开 SaveAssistantModal)
 *   onSendToAI  - 「让 AI 帮我改」回调,传回粘贴的文字(父组件据此注入对话画布)
 */

import { useState, useEffect } from 'react'

/* ==================== 样式常量(与助手系列组件保持一致) ==================== */
const C = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent:       '#F59E0B',
  success:      '#10B981',
  danger:       '#EF4444',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
  borderMid:    '#E5E7EB',
}

/** prompt 长度上限(与后端 maxAssistantPromptLen 对齐) */
const MAX_PROMPT_LEN = 128 * 1024

/* ==================== Props 类型 ==================== */

export interface PastePromptModalProps {
  open: boolean
  onClose: () => void
  /** 「直接保存为助手」:把粘贴文字交还父组件(走 SaveAssistantModal) */
  onSaveDirect: (text: string) => void
  /** 「让 AI 帮我改改」:把粘贴文字交还父组件(注入对话画布) */
  onSendToAI: (text: string) => void
}

/* ==================== 主组件 ==================== */

export default function PastePromptModal(props: PastePromptModalProps) {
  const { open, onClose, onSaveDirect, onSendToAI } = props

  const [text, setText] = useState('')
  const [err, setErr]   = useState<string | null>(null)

  // open 时清空
  useEffect(() => {
    if (!open) return
    setText('')
    setErr(null)
  }, [open])

  // ESC 关闭
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, onClose])

  // 校验(两个出口共用)
  const validate = (): string | null => {
    if (!text.trim()) return '请先粘贴或输入一段提示词'
    if (text.length > MAX_PROMPT_LEN) return `提示词过长(${text.length} 字符),上限 ${MAX_PROMPT_LEN} 字符`
    return null
  }

  const handleSaveDirect = () => {
    const e = validate()
    if (e) { setErr(e); return }
    onSaveDirect(text.trim())
  }

  const handleSendToAI = () => {
    const e = validate()
    if (e) { setErr(e); return }
    onSendToAI(text.trim())
  }

  if (!open) return null

  return (
    <div
      onClick={onClose}
      style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        background: 'rgba(17,24,39,0.5)',
        zIndex: 10001,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: '24px',
      }}
    >
      <div
        onClick={e => e.stopPropagation()}
        style={{
          width: '560px', maxWidth: '100%', maxHeight: '90vh',
          background: C.card, borderRadius: '14px',
          boxShadow: '0 24px 64px rgba(0,0,0,0.18)',
          display: 'flex', flexDirection: 'column', overflow: 'hidden',
        }}
      >
        {/* 标题栏 */}
        <div style={{
          padding: '16px 20px', borderBottom: `1px solid ${C.border}`,
          background: 'linear-gradient(135deg,rgba(79,123,232,0.06),rgba(79,123,232,0.02))',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0,
        }}>
          <span style={{ fontSize: '15px', fontWeight: 700, color: C.text }}>
            ✍️ 粘贴你已有的提示词
          </span>
          <button
            onClick={onClose}
            style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted, lineHeight: 1 }}
          >×</button>
        </div>

        {/* 内容区 */}
        <div style={{ flex: 1, overflow: 'auto', padding: '18px 22px' }}>
          <div style={{
            marginBottom: '12px', fontSize: '12px', color: C.textSec, lineHeight: 1.6,
          }}>
            把你手上现成的提示词粘进来。可以<b style={{ color: C.success }}>直接存成助手</b>,
            也可以<b style={{ color: C.primary }}>交给 AI 帮你润色、补充</b>再存。
          </div>

          <textarea
            value={text}
            onChange={e => setText(e.target.value)}
            placeholder="把提示词粘贴到这里……&#10;&#10;例如:你是一位经验丰富的初中语文老师,擅长……"
            autoFocus
            style={{
              width: '100%', minHeight: '240px', maxHeight: '50vh',
              padding: '12px 14px', borderRadius: '8px',
              border: `1px solid ${C.borderMid}`,
              fontSize: '13px', color: C.text, lineHeight: 1.7,
              outline: 'none', resize: 'vertical', boxSizing: 'border-box',
              fontFamily: 'inherit', background: C.bg,
            }}
          />

          <div style={{ marginTop: '6px', fontSize: '11px', color: C.textMuted, textAlign: 'right' }}>
            {text.length.toLocaleString()} 字符
          </div>

          {err && (
            <div style={{
              marginTop: '10px', padding: '10px 12px', borderRadius: '8px',
              background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.2)',
              color: C.danger, fontSize: '13px',
            }}>
              ⚠️ {err}
            </div>
          )}
        </div>

        {/* 底部双出口 */}
        <div style={{
          padding: '12px 20px', borderTop: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'flex-end', gap: '8px', alignItems: 'center',
          background: C.bg, flexShrink: 0, flexWrap: 'wrap',
        }}>
          <button
            onClick={onClose}
            style={{
              padding: '8px 14px', borderRadius: '7px',
              border: `1px solid ${C.borderMid}`, background: '#fff',
              color: C.textSec, fontSize: '13px', cursor: 'pointer',
            }}
          >取消</button>
          {/* 出口2:交给 AI 改 */}
          <button
            onClick={handleSendToAI}
            style={{
              padding: '8px 16px', borderRadius: '7px',
              border: `1px solid ${C.primary}`, background: '#fff', color: C.primary,
              fontSize: '13px', fontWeight: 600, cursor: 'pointer',
            }}
            title="把这段提示词交给左侧 AI,让它帮你润色、补充,你再继续聊"
          >💬 让 AI 帮我改改</button>
          {/* 出口1:直接保存 */}
          <button
            onClick={handleSaveDirect}
            style={{
              padding: '8px 18px', borderRadius: '7px', border: 'none',
              background: C.success, color: '#fff',
              fontSize: '13px', fontWeight: 600, cursor: 'pointer',
            }}
            title="原样保存成你的助手(AI 不改动)"
          >✓ 直接保存为助手</button>
        </div>
      </div>
    </div>
  )
}

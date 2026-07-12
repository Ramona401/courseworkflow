/**
 * ConversationInputBar.tsx — 对话模式底部输入区（迭代3.5 A2-3 拆分批次）
 *
 * 从 ConversationModePage.tsx 抽出的输入区整体，自上而下三块：
 *   1. 已选组件提示条（A2-1：下一条消息自动携带，可一键清空）；
 *   2. 「+」能力菜单（Phase C 能力注册表最小先行版——菜单项来自 PLUS_MENU_ITEMS 纯数据，
 *      可用性由页面注入的 plusItemAvailability 计算，不可用项灰显+title 说明原因，
 *      对齐设计文档 2.4 铁律②）+ 文本输入框 + 发送按钮；
 *   3. 发布入口（教案正文非空时常驻，最终守门仍在后端 ErrLPContentEmpty）。
 *
 * 状态归属纪律：
 *   - inputText 与 showPlusMenu 为本组件私有状态（退出备课时整个 chatting 视图卸载自动复位）；
 *   - 其余全部经 props 与页面交互，本组件不直接调任何 API；
 *   - textarea 经 forwardRef 暴露给页面（剧本芯片 focus_input 动作需要聚焦输入框）。
 */
import { useState, forwardRef, useRef, useImperativeHandle } from 'react'
import { C } from '../components/workshopConstants'
import { PLUS_MENU_ITEMS } from './conversationScript'

/**
 * 对外暴露的命令句柄（v191 改动E）——
 * 原先 forwardRef 直接暴露 textarea DOM 节点，只能 focus()，无可见反馈（焦点变化老师常感知不到）。
 * 改为暴露 { focus, prefill }：prefill 预填一句引导文字到输入框并聚焦+光标置末尾，
 * 让"继续修改/我要改"类芯片点击后有明确可见反馈（输入框出现文字+光标，老师知道"该我说了"）。
 */
export interface ConversationInputBarHandle {
  /** 仅聚焦输入框 */
  focus: () => void
  /** 预填文字到输入框 + 聚焦 + 光标移到末尾 */
  prefill: (text: string) => void
}

/** 输入区组件 Props */
export interface ConversationInputBarProps {
  /** AI 忙碌中（思考/流式/一键生成）——禁用输入、发送、「+」菜单与发布 */
  isBusy: boolean
  /** 输入框占位文案（页面按消息轮次轮换，忙碌时为"AI思考中…"） */
  placeholder: string
  /** 已选组件数量（>0 时显示提示条） */
  selectedCount: number
  /** 清空已选组件集合 */
  onClearSelected: () => void
  /** 发送一条文本消息（等价老师打字，已选组件由页面侧自动携带） */
  onSend: (text: string) => void
  /** 教案正文是否非空（控制发布按钮显隐） */
  hasContent: boolean
  /** 发布教案（页面侧含正文判空前置提示与二次确认） */
  onPublish: () => void
  /** 「+」菜单各项可用性计算（不可用项灰显+原因） */
  plusItemAvailability: (tool: string) => { enabled: boolean; reason: string }
  /** 唤起能力（菜单项点击后分发，本组件先收起菜单再回调） */
  onOpenTool: (tool: string) => void
  /** 参考资料附件文件名（非空时显示参考资料提示条） */
  refMaterialName?: string
  /** 移除参考资料附件 */
  onClearRefMaterial?: () => void
}

/**
 * 对话模式底部输入区组件（forwardRef 暴露 textarea 供页面聚焦）
 */
const ConversationInputBar = forwardRef<ConversationInputBarHandle, ConversationInputBarProps>(
  function ConversationInputBar(props, ref) {
    const {
      isBusy, placeholder, selectedCount, onClearSelected,
      onSend, hasContent, onPublish, plusItemAvailability, onOpenTool,
      refMaterialName, onClearRefMaterial,
    } = props

    /** 输入框文本（私有状态：发送即清空，退出备课随视图卸载复位） */
    const [inputText, setInputText] = useState('')
    /** 内部真正的 textarea DOM ref（v191 改动E：不再把 textarea 直接暴露给外部） */
    const taRef = useRef<HTMLTextAreaElement>(null)

    /** 对外暴露命令句柄：focus（仅聚焦）/ prefill（预填引导文字+聚焦+光标置末尾） */
    useImperativeHandle(ref, () => ({
      focus: () => taRef.current?.focus(),
      prefill: (text: string) => {
        setInputText(text)
        // 等 React 把值写进 DOM 后再聚焦并把光标移到末尾（给老师明确可见反馈）
        requestAnimationFrame(() => {
          const el = taRef.current
          if (el) { el.focus(); el.setSelectionRange(text.length, text.length) }
        })
      },
    }))
    /** 「+」能力菜单展开态（私有状态） */
    const [showPlusMenu, setShowPlusMenu] = useState(false)

    /** 执行发送：非空且非忙碌时清空输入框并回调页面 */
    const doSend = () => {
      if (!inputText.trim() || isBusy) return
      const t = inputText.trim()
      setInputText('')
      onSend(t)
    }

    /** 菜单项点击：先收起菜单，再分发能力唤起 */
    const handleMenuItem = (tool: string) => {
      setShowPlusMenu(false)
      onOpenTool(tool)
    }

    return (
      <>
        {/* A2-1：已选组件提示条（下一条消息自动携带） */}
        {selectedCount > 0 && (
          <div style={{ padding: '7px 18px', background: C.primaryLight, borderTop: `1px solid ${C.border}`, fontSize: '12px', color: C.primary, display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0 }}>
            <span>🧩 已选 {selectedCount} 个教学组件，下一条消息发出时 AI 会一并参考</span>
            <button onClick={onClearSelected} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.textMuted, fontSize: '12px' }}>清空</button>
          </div>
        )}

        {/* 参考资料附件提示条（会话级，AI 每轮参考；点移除即清空） */}
        {refMaterialName && (
          <div style={{ padding: '7px 18px', background: 'rgba(129,140,248,0.10)', borderTop: `1px solid ${C.border}`, fontSize: '12px', color: '#6366F1', display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0 }}>
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>📎 已附参考资料「{refMaterialName}」，AI 每轮回复都会参考</span>
            {onClearRefMaterial && (
              <button onClick={onClearRefMaterial} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.textMuted, fontSize: '12px', flexShrink: 0, marginLeft: '10px' }}>移除</button>
            )}
          </div>
        )}

        {/* 底部输入区（左侧「+」能力菜单） */}
        <div style={{ padding: '12px 18px', borderTop: `1px solid ${C.border}`, background: C.card, flexShrink: 0 }}>
          <div style={{ display: 'flex', gap: '10px', alignItems: 'flex-end' }}>
            {/* 「+」能力菜单（Phase C 能力注册表最小先行版） */}
            <div style={{ position: 'relative', flexShrink: 0 }}>
              <button onClick={() => setShowPlusMenu(v => !v)} disabled={isBusy}
                title="更多备课能力"
                style={{ width: '38px', height: '38px', borderRadius: '50%', border: `1px solid ${C.border}`, background: showPlusMenu ? C.primaryLight : C.card, color: showPlusMenu ? C.primary : C.textSec, fontSize: '20px', cursor: isBusy ? 'not-allowed' : 'pointer', opacity: isBusy ? 0.5 : 1, display: 'flex', alignItems: 'center', justifyContent: 'center', transition: 'all 150ms ease' }}>
                ＋
              </button>
              {showPlusMenu && (
                <>
                  {/* 点击外部关闭的透明遮罩 */}
                  <div onClick={() => setShowPlusMenu(false)} style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 998 }} />
                  <div style={{ position: 'absolute', bottom: '46px', left: 0, width: '260px', background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, boxShadow: '0 8px 32px rgba(0,0,0,0.12)', padding: '6px', zIndex: 999 }}>
                    {PLUS_MENU_ITEMS.map(item => {
                      const avail = plusItemAvailability(item.tool)
                      return (
                        <button key={item.tool}
                          onClick={() => { if (avail.enabled) handleMenuItem(item.tool) }}
                          disabled={!avail.enabled}
                          title={avail.enabled ? item.desc : avail.reason}
                          style={{ display: 'flex', alignItems: 'flex-start', gap: '10px', width: '100%', padding: '9px 10px', borderRadius: '8px', border: 'none', background: 'transparent', cursor: avail.enabled ? 'pointer' : 'not-allowed', opacity: avail.enabled ? 1 : 0.45, textAlign: 'left', transition: 'background 150ms ease' }}
                          onMouseEnter={e => { if (avail.enabled) (e.currentTarget as HTMLButtonElement).style.background = '#F3F4F6' }}
                          onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'transparent' }}>
                          <span style={{ fontSize: '17px', flexShrink: 0 }}>{item.emoji}</span>
                          <span style={{ minWidth: 0 }}>
                            <span style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text }}>{item.label}</span>
                            <span style={{ display: 'block', fontSize: '11px', color: C.textMuted, marginTop: '1px', lineHeight: 1.4 }}>
                              {avail.enabled ? item.desc : avail.reason}
                            </span>
                          </span>
                        </button>
                      )
                    })}
                  </div>
                </>
              )}
            </div>

            {/* 输入框 + 发送按钮 */}
            <div style={{ flex: 1, display: 'flex', gap: '10px', alignItems: 'flex-end', background: '#F9FAFB', borderRadius: '12px', border: `1px solid ${C.border}`, padding: '9px 12px' }}>
              <textarea ref={taRef} value={inputText}
                onChange={e => setInputText(e.target.value)}
                onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); doSend() } }}
                placeholder={placeholder} rows={2} disabled={isBusy}
                style={{ flex: 1, background: 'transparent', border: 'none', outline: 'none', fontSize: '15px', color: C.text, resize: 'none', fontFamily: 'inherit', lineHeight: 1.6, opacity: isBusy ? 0.5 : 1 }} />
              <button onClick={doSend}
                disabled={isBusy || !inputText.trim()}
                style={{ width: '36px', height: '36px', flexShrink: 0, borderRadius: '50%', border: 'none', background: (isBusy || !inputText.trim()) ? '#E5E7EB' : C.primary, color: '#fff', cursor: (isBusy || !inputText.trim()) ? 'not-allowed' : 'pointer', fontSize: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>→</button>
            </div>
          </div>

          {/* 发布入口：正文非空时常驻（最终守门仍在后端） */}
          {hasContent && (
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '8px' }}>
              <button onClick={onPublish} disabled={isBusy}
                style={{ padding: '6px 16px', borderRadius: '16px', border: 'none', background: isBusy ? '#E5E7EB' : 'linear-gradient(135deg, #10B981, #34D399)', color: isBusy ? C.textMuted : '#fff', fontSize: '12px', fontWeight: 600, cursor: isBusy ? 'not-allowed' : 'pointer' }}>
                🚀 发布教案
              </button>
            </div>
          )}
        </div>
      </>
    )
  }
)

export default ConversationInputBar

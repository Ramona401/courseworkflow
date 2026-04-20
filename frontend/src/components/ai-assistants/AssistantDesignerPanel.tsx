/**
 * AssistantDesignerPanel.tsx — AI 助手对话式创作面板(TE-DNA 3.0 P0.5)
 *
 * 角色:
 *   - 在 AssistantEditModal 的"💬 AI 帮我写"Tab 内渲染
 *   - 通过 designChat SSE 接口,让老师聊着聊着生成 full_prompt 草稿
 *   - 草稿不直接污染 Modal 的 fullPrompt state,老师点"应用到编辑"才写回
 *
 * 布局(左右分栏):
 *   ┌──────────────────┬─────────────────┐
 *   │   对话区 55%      │   草稿预览 45%   │
 *   │                  │                 │
 *   │  消息列表(可滚)   │  Markdown 渲染  │
 *   │  └ 组件参考折叠条 │  字符数统计     │
 *   │                  │                 │
 *   │  输入框 + 发送    │  应用到编辑按钮  │
 *   └──────────────────┴─────────────────┘
 *
 * SSE 事件处理(7 种):
 *   connected    → isStreaming=true,追加空 AI 气泡
 *   searching    → 设置瞬时 statusTip("🔍 ...")
 *   components   → 存 components 数组,清 statusTip
 *   chunk        → 追加 chunk 到最后一条 AI 气泡 content
 *   draft_update → 更新 localDraft(右侧实时同步)
 *   done         → isStreaming=false,用完整 reply 覆盖最后一条(保险)
 *   error        → isStreaming=false,最后一条 AI 气泡换成 ⚠️
 *
 * 关键设计:
 *   - localDraft 独立于 Modal 的 fullPrompt,用户点"应用"才回写
 *   - 每轮发消息都传 currentDraft=localDraft(让后端知道基于哪版迭代)
 *   - useEffect cleanup 调 SSE.close() 防止组件卸载后继续消耗 token
 *
 * v114 Batch 2 第 2 轮(2026-04-20):从空壳填到完整功能
 *
 * v114 修复 C(2026-04-20 晚):
 *   草稿区之前用 <pre> 裸显示,导致 **粗体** 显示为裸星号,影响阅读。
 *   改为 renderMarkdown(复用 workshopConstants 的成熟实现),和对话气泡一致。
 *   老师需要看原文时,点"✓ 应用到编辑"即可进入 textarea 编辑态查看纯文本。
 *   草稿预览区的定位是"阅读态",故走 Markdown 渲染更自然。
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import {
  designChat,
  type AssistantScene,
  type DesignerMessage,
  type DesignerComponentBrief,
  type DesignChatSSEConnection,
} from '@/api/ai-assistants'
import { renderMarkdown } from '@/pages/lesson-plans/plan-detail/components/planDetailConstants'

/* ==================== 样式常量(与 EditModal/Selector 保持一致) ==================== */
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
  aiBubble:     '#EEF4FF',
  userBubble:   '#4F7BE8',
}

/* ==================== Props 类型 ==================== */

export interface AssistantDesignerPanelProps {
  /** 当前表单学科(从 Modal state 透传) */
  subject: string
  /** 当前表单年级(从 Modal state 透传) */
  grade: string
  /** 当前表单勾选的场景(从 Modal state 透传) */
  scenes: AssistantScene[]
  /**
   * 初始草稿:
   *   - create 模式:通常传空字符串
   *   - edit 模式:传当前 fullPrompt,AI 基于此迭代
   */
  initialDraft: string
  /**
   * 用户点"应用到编辑"按钮时回调
   * 参数:AI 生成的最新 draft(完整 prompt 字符串)
   * 父组件(Modal)收到后应:
   *   1. 把 draft 写入 fullPrompt state
   *   2. 自动切回"📝 手动编辑"Tab(本版本 Modal 默认行为)
   */
  onApplyDraft: (draft: string) => void
}

/* ==================== 内部类型 ==================== */

/**
 * 对话消息(前端展示用,比后端 DesignerMessage 多了展示字段)
 * - components/showComponents 只有 assistant 消息才有
 */
interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  /** 当前消息关联的组件参考(只有 assistant 消息才会填) */
  components?: DesignerComponentBrief[]
  /** UI 折叠状态:组件列表是否展开 */
  showComponents?: boolean
  /** 标记为错误消息(以 ⚠️ 样式渲染) */
  isError?: boolean
}

/* ==================== 主组件 ==================== */

export default function AssistantDesignerPanel(props: AssistantDesignerPanelProps) {
  const { subject, grade, scenes, initialDraft, onApplyDraft } = props

  // ==================== state ====================
  const [messages, setMessages]       = useState<ChatMessage[]>([])
  const [input, setInput]             = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [statusTip, setStatusTip]     = useState('')          // "🔍 AI 正在查组件库..." 之类瞬时提示
  const [localDraft, setLocalDraft]   = useState(initialDraft)// AI 生成的草稿,独立于 Modal 的 fullPrompt
  const [errorMsg, setErrorMsg]       = useState('')          // 顶部错误横幅

  // ==================== ref ====================
  const connRef  = useRef<DesignChatSSEConnection | null>(null)  // 当前 SSE 连接句柄
  const scrollEndRef = useRef<HTMLDivElement>(null)               // 对话区自动滚底锚点
  const inputRef = useRef<HTMLTextAreaElement>(null)              // 输入框 ref(发送后聚焦)

  // ==================== 自动滚底 ====================
  useEffect(() => {
    scrollEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, statusTip, isStreaming])

  // ==================== 组件卸载时中止 SSE ====================
  // 避免用户关 Modal 后后端仍在消耗 token
  useEffect(() => {
    return () => {
      connRef.current?.close()
      connRef.current = null
    }
  }, [])

  // ==================== 发送消息 ====================
  const handleSend = useCallback(() => {
    const text = input.trim()
    if (!text || isStreaming) return

    setErrorMsg('')

    // 1. 先把当前 messages 快照转成后端需要的 history 格式
    //    (不含本轮 user 消息,刚刚 push 进去的 AI 占位气泡也不算历史)
    const history: DesignerMessage[] = messages
      .filter(m => !m.isError && m.content.trim() !== '')  // 过滤错误消息和空气泡
      .map(m => ({ role: m.role, content: m.content }))

    // 2. push 用户气泡 + AI 空占位气泡(准备接收流式 chunk)
    setMessages(prev => [
      ...prev,
      { role: 'user', content: text },
      { role: 'assistant', content: '' },
    ])

    // 3. 清输入框 + 进入 streaming 态
    setInput('')
    setIsStreaming(true)
    setStatusTip('')

    // 4. 关上可能残留的旧连接
    connRef.current?.close()

    // 5. 发起 SSE
    const conn = designChat(
      {
        message: text,
        history,
        subject,
        grade,
        scenes,
        current_draft: localDraft,
      },
      {
        onConnected: () => {
          // 流已建立(前面已经 push 了占位气泡,这里不做特殊动作)
        },
        onSearching: (reason) => {
          setStatusTip(`🔍 ${reason || 'AI 正在查阅组件库...'}`)
        },
        onComponents: (briefs) => {
          setStatusTip('')
          // 把组件挂到"最后一条 assistant 消息"上
          setMessages(prev => {
            if (prev.length === 0) return prev
            const next = [...prev]
            const last = next[next.length - 1]
            if (last.role === 'assistant') {
              next[next.length - 1] = {
                ...last,
                components: briefs,
                showComponents: false,
              }
            }
            return next
          })
        },
        onChunk: (chunk) => {
          // 追加 chunk 到最后一条 assistant 气泡
          setMessages(prev => {
            if (prev.length === 0) return prev
            const next = [...prev]
            const last = next[next.length - 1]
            if (last.role === 'assistant') {
              next[next.length - 1] = { ...last, content: last.content + chunk }
            }
            return next
          })
        },
        onDraftUpdate: (draft) => {
          setLocalDraft(draft)
        },
        onDone: (reply, draft /* referenced 当前版本暂不消费 */) => {
          // 用完整 reply 替换(保险,避免 chunk 拼接漏字)
          setMessages(prev => {
            if (prev.length === 0) return prev
            const next = [...prev]
            const last = next[next.length - 1]
            if (last.role === 'assistant' && reply && reply.length >= last.content.length) {
              next[next.length - 1] = { ...last, content: reply }
            }
            return next
          })
          if (draft) setLocalDraft(draft)
          setIsStreaming(false)
          setStatusTip('')
        },
        onError: (err) => {
          // 最后一条 assistant 气泡换成错误消息
          setMessages(prev => {
            if (prev.length === 0) return [...prev, { role: 'assistant', content: `⚠️ ${err}`, isError: true }]
            const next = [...prev]
            const last = next[next.length - 1]
            if (last.role === 'assistant' && !last.content) {
              // 空占位气泡 → 直接替换
              next[next.length - 1] = { role: 'assistant', content: `⚠️ ${err}`, isError: true }
            } else {
              // 非空气泡 → 追加错误消息
              next.push({ role: 'assistant', content: `⚠️ ${err}`, isError: true })
            }
            return next
          })
          setIsStreaming(false)
          setStatusTip('')
          setErrorMsg(err)
        },
      },
    )

    connRef.current = conn

    // 6. 发送后输入框聚焦(下一轮对话可以直接打字)
    setTimeout(() => inputRef.current?.focus(), 50)
  }, [input, isStreaming, messages, subject, grade, scenes, localDraft])

  // ==================== 切换组件参考折叠态 ====================
  const toggleComponents = (msgIndex: number) => {
    setMessages(prev => {
      const next = [...prev]
      const m = next[msgIndex]
      if (m && m.role === 'assistant') {
        next[msgIndex] = { ...m, showComponents: !m.showComponents }
      }
      return next
    })
  }

  // ==================== 应用草稿到 Modal ====================
  const handleApply = () => {
    if (!localDraft.trim()) return
    onApplyDraft(localDraft)
  }

  // ==================== 清空当前对话 ====================
  const handleClearChat = () => {
    if (isStreaming) return
    if (messages.length === 0) return
    if (!confirm('确认清空当前对话?草稿内容不会被清除。')) return
    connRef.current?.close()
    connRef.current = null
    setMessages([])
    setStatusTip('')
    setErrorMsg('')
    setIsStreaming(false)
  }

  // ==================== 渲染 ====================
  return (
    <div
      style={{
        display: 'flex',
        gap: '12px',
        height: '480px',
        minHeight: '480px',
      }}
    >
      {/* ==================== 左列:对话区(55%) ==================== */}
      <div
        style={{
          flex: '1 1 55%',
          display: 'flex',
          flexDirection: 'column',
          background: C.card,
          borderRadius: '10px',
          border: `1px solid ${C.border}`,
          overflow: 'hidden',
          minWidth: 0, // flex 布局防止子元素撑爆
        }}
      >
        {/* 对话区头部 */}
        <div
          style={{
            padding: '8px 12px',
            borderBottom: `1px solid ${C.border}`,
            background: 'linear-gradient(135deg,rgba(79,123,232,0.06),rgba(129,140,248,0.04))',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            flexShrink: 0,
          }}
        >
          <span style={{ fontSize: '12px', fontWeight: 700, color: C.primary }}>
            💬 和 AI 聊着做助手
          </span>
          <button
            onClick={handleClearChat}
            disabled={isStreaming || messages.length === 0}
            style={{
              background: 'none',
              border: 'none',
              fontSize: '11px',
              color: isStreaming || messages.length === 0 ? C.textMuted : C.textSec,
              cursor: isStreaming || messages.length === 0 ? 'not-allowed' : 'pointer',
              padding: '2px 6px',
            }}
          >
            🗑 清空对话
          </button>
        </div>

        {/* 消息列表 */}
        <div
          style={{
            flex: 1,
            overflow: 'auto',
            padding: '12px',
            display: 'flex',
            flexDirection: 'column',
            gap: '10px',
          }}
        >
          {/* 首次引导 */}
          {messages.length === 0 && (
            <div
              style={{
                padding: '14px',
                background: C.primaryLight,
                borderRadius: '8px',
                border: '1px solid rgba(79,123,232,0.2)',
                fontSize: '12px',
                color: C.textSec,
                lineHeight: 1.8,
              }}
            >
              <div style={{ fontWeight: 700, color: C.primary, marginBottom: '6px' }}>
                💡 告诉我您想做什么样的助手
              </div>
              您可以这样描述:
              <ul style={{ margin: '6px 0 0 18px', padding: 0, fontSize: '11px', lineHeight: 1.7 }}>
                <li>"做一个严苛的初中 AI 课审核员,不接受空话"</li>
                <li>"我要教小学 AI 画图,帮我做个能一起备课的助手"</li>
                <li>"像 XX 老师那样耐心地陪我改教案"</li>
              </ul>
              <div style={{ marginTop: '8px', color: C.textMuted, fontSize: '11px' }}>
                AI 会先问清风格,再查阅组件知识库,给您起草完整 Prompt。
              </div>
            </div>
          )}

          {/* 消息气泡 */}
          {messages.map((msg, idx) => {
            const isLastAssistant =
              idx === messages.length - 1 && msg.role === 'assistant'
            const isThisStreaming = isLastAssistant && isStreaming

            if (msg.role === 'user') {
              // ---------- 用户气泡 ----------
              return (
                <div
                  key={idx}
                  style={{ display: 'flex', justifyContent: 'flex-end' }}
                >
                  <div
                    style={{
                      maxWidth: '85%',
                      padding: '8px 12px',
                      borderRadius: '12px 2px 12px 12px',
                      background: C.userBubble,
                      color: '#fff',
                      fontSize: '12px',
                      lineHeight: 1.7,
                      whiteSpace: 'pre-wrap',
                    }}
                  >
                    {msg.content}
                  </div>
                </div>
              )
            }

            // ---------- AI 气泡 ----------
            return (
              <div key={idx} style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <div style={{ display: 'flex', justifyContent: 'flex-start' }}>
                  <div
                    style={{
                      width: '22px',
                      height: '22px',
                      borderRadius: '50%',
                      background: 'linear-gradient(135deg,#4F7BE8,#818CF8)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: '11px',
                      flexShrink: 0,
                      marginRight: '6px',
                      marginTop: '2px',
                    }}
                  >
                    ✨
                  </div>
                  <div
                    style={{
                      maxWidth: '85%',
                      padding: '8px 12px',
                      borderRadius: '2px 12px 12px 12px',
                      background: msg.isError ? 'rgba(239,68,68,0.08)' : C.aiBubble,
                      color: msg.isError ? C.danger : C.text,
                      fontSize: '12px',
                      lineHeight: 1.7,
                      border: msg.isError ? '1px solid rgba(239,68,68,0.25)' : `1px solid ${C.border}`,
                      whiteSpace: 'pre-wrap',
                    }}
                  >
                    {/* 空内容 + 流中:显示三点加载 */}
                    {!msg.content && isThisStreaming ? (
                      <div style={{ display: 'flex', gap: '4px', alignItems: 'center', padding: '2px 0' }}>
                        {[0, 1, 2].map(d => (
                          <div
                            key={d}
                            style={{
                              width: '5px',
                              height: '5px',
                              borderRadius: '50%',
                              background: C.primary,
                              animation: `designerPulse 1.2s ease-in-out ${d * 0.2}s infinite`,
                            }}
                          />
                        ))}
                      </div>
                    ) : (
                      <>
                        {/* AI 消息走 Markdown 渲染(复用工坊同款 renderMarkdown) */}
                        {msg.isError ? msg.content : renderMarkdown(msg.content)}
                        {/* 流式光标 */}
                        {isThisStreaming && msg.content && (
                          <span
                            style={{
                              display: 'inline-block',
                              width: '6px',
                              height: '12px',
                              background: C.primary,
                              marginLeft: '2px',
                              animation: 'designerBlink 1s infinite',
                              verticalAlign: 'middle',
                            }}
                          />
                        )}
                      </>
                    )}
                  </div>
                </div>

                {/* 组件参考折叠条(只有当该条 AI 消息关联组件时显示) */}
                {msg.components && msg.components.length > 0 && (
                  <div style={{ marginLeft: '28px' }}>
                    <button
                      type="button"
                      onClick={() => toggleComponents(idx)}
                      style={{
                        background: msg.showComponents ? C.primaryLight : C.bg,
                        border: `1px solid ${C.border}`,
                        borderRadius: '6px',
                        padding: '4px 8px',
                        fontSize: '11px',
                        color: C.primary,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '4px',
                      }}
                    >
                      📚 AI 参考了 {msg.components.length} 个组件
                      <span style={{ fontSize: '9px' }}>{msg.showComponents ? '▲' : '▼'}</span>
                    </button>
                    {msg.showComponents && (
                      <div
                        style={{
                          marginTop: '4px',
                          padding: '8px 10px',
                          background: C.bg,
                          borderRadius: '6px',
                          border: `1px solid ${C.border}`,
                          display: 'flex',
                          flexDirection: 'column',
                          gap: '3px',
                        }}
                      >
                        {msg.components.map((c, i) => (
                          <div
                            key={c.id || i}
                            style={{
                              fontSize: '11px',
                              color: C.textSec,
                              lineHeight: 1.5,
                            }}
                          >
                            <span style={{ color: C.primary, fontWeight: 600 }}>
                              {c.library_name || c.library_type || '组件'}
                            </span>
                            <span style={{ margin: '0 4px', color: C.textMuted }}>·</span>
                            <span>{c.name}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )
          })}

          {/* 瞬时状态提示(searching 阶段) */}
          {statusTip && (
            <div
              style={{
                alignSelf: 'flex-start',
                marginLeft: '28px',
                fontSize: '11px',
                color: C.textMuted,
                fontStyle: 'italic',
                padding: '2px 0',
              }}
            >
              {statusTip}
            </div>
          )}

          <div ref={scrollEndRef} />
        </div>

        {/* 错误横幅(可选,一般 error 事件已经把错误消息 push 到气泡了) */}
        {errorMsg && !isStreaming && (
          <div
            style={{
              padding: '6px 12px',
              background: 'rgba(239,68,68,0.06)',
              borderTop: '1px solid rgba(239,68,68,0.2)',
              fontSize: '11px',
              color: C.danger,
              flexShrink: 0,
            }}
          >
            ⚠️ {errorMsg}
          </div>
        )}

        {/* 输入框 */}
        <div
          style={{
            padding: '10px 12px',
            borderTop: `1px solid ${C.border}`,
            background: C.card,
            flexShrink: 0,
          }}
        >
          <div
            style={{
              display: 'flex',
              gap: '6px',
              alignItems: 'flex-end',
              background: C.bg,
              borderRadius: '8px',
              border: `1px solid ${C.border}`,
              padding: '6px 8px',
            }}
          >
            <textarea
              ref={inputRef}
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleSend()
                }
              }}
              placeholder={messages.length === 0 ? '描述您想要的助手风格...' : '继续聊,比如"再严苛一点""给我举个例子"...'}
              rows={2}
              disabled={isStreaming}
              style={{
                flex: 1,
                background: 'transparent',
                border: 'none',
                outline: 'none',
                fontSize: '12px',
                color: C.text,
                resize: 'none',
                fontFamily: 'inherit',
                lineHeight: 1.5,
                opacity: isStreaming ? 0.5 : 1,
              }}
            />
            <button
              type="button"
              onClick={handleSend}
              disabled={isStreaming || !input.trim()}
              style={{
                width: '28px',
                height: '28px',
                borderRadius: '50%',
                border: 'none',
                background: isStreaming || !input.trim() ? '#E5E7EB' : C.primary,
                color: '#fff',
                cursor: isStreaming || !input.trim() ? 'not-allowed' : 'pointer',
                fontSize: '13px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flexShrink: 0,
              }}
            >
              →
            </button>
          </div>
          <div
            style={{
              fontSize: '10px',
              color: C.textMuted,
              marginTop: '4px',
              textAlign: 'center',
            }}
          >
            Enter 发送 · Shift+Enter 换行
          </div>
        </div>
      </div>

      {/* ==================== 右列:草稿预览(45%) ==================== */}
      <div
        style={{
          flex: '1 1 45%',
          display: 'flex',
          flexDirection: 'column',
          background: C.card,
          borderRadius: '10px',
          border: `1px solid ${C.border}`,
          overflow: 'hidden',
          minWidth: 0,
        }}
      >
        {/* 草稿区头部 */}
        <div
          style={{
            padding: '8px 12px',
            borderBottom: `1px solid ${C.border}`,
            background: 'linear-gradient(135deg,rgba(16,185,129,0.06),rgba(16,185,129,0.02))',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            flexShrink: 0,
          }}
        >
          <span style={{ fontSize: '12px', fontWeight: 700, color: C.success }}>
            📝 当前草稿
          </span>
          <span style={{ fontSize: '10px', color: C.textMuted }}>
            {localDraft.length.toLocaleString()} 字符
          </span>
        </div>

        {/*
         * 草稿内容区
         * v114 修复 C:
         *   原 <pre>裸显示 → 改走 renderMarkdown 正常渲染 ** 粗体/## 标题/列表 等
         *   lineHeight/color/fontSize 设到包裹 div 上,
         *   renderMarkdown 内部元素会继承这些排版属性
         */}
        <div
          style={{
            flex: 1,
            overflow: 'auto',
            padding: '12px 14px',
            background: C.bg,
            fontSize: '12px',
            lineHeight: 1.8,
            color: C.text,
          }}
        >
          {localDraft ? (
            renderMarkdown(localDraft)
          ) : (
            <div
              style={{
                textAlign: 'center',
                padding: '40px 16px',
                color: C.textMuted,
                fontSize: '12px',
                lineHeight: 1.8,
              }}
            >
              <div style={{ fontSize: '32px', marginBottom: '8px' }}>📄</div>
              AI 还没生成草稿
              <br />
              <span style={{ fontSize: '11px' }}>先在左侧描述您的需求吧</span>
            </div>
          )}
        </div>

        {/* 草稿区底部:应用按钮 */}
        <div
          style={{
            padding: '8px 12px',
            borderTop: `1px solid ${C.border}`,
            background: C.card,
            flexShrink: 0,
          }}
        >
          <button
            type="button"
            onClick={handleApply}
            disabled={!localDraft.trim() || isStreaming}
            style={{
              width: '100%',
              padding: '8px',
              borderRadius: '7px',
              border: 'none',
              background: !localDraft.trim() || isStreaming ? C.borderMid : C.success,
              color: !localDraft.trim() || isStreaming ? C.textMuted : '#fff',
              fontSize: '12px',
              fontWeight: 600,
              cursor: !localDraft.trim() || isStreaming ? 'not-allowed' : 'pointer',
            }}
          >
            {isStreaming ? '⏳ AI 生成中...' : '✓ 应用到编辑'}
          </button>
          <div
            style={{
              fontSize: '10px',
              color: C.textMuted,
              marginTop: '4px',
              textAlign: 'center',
              lineHeight: 1.5,
            }}
          >
            应用后自动切到手动编辑 Tab,可继续微调
          </div>
        </div>
      </div>

      {/* 动画 keyframes(两个面板共用) */}
      <style>{`
        @keyframes designerPulse {
          0%, 80%, 100% { opacity: 0.3; transform: scale(0.8); }
          40% { opacity: 1; transform: scale(1.2); }
        }
        @keyframes designerBlink {
          0%, 50% { opacity: 1; }
          51%, 100% { opacity: 0; }
        }
      `}</style>
    </div>
  )
}

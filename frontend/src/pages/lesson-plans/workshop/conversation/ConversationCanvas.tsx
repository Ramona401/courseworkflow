/**
 * ConversationCanvas.tsx — 对话模式右栏「教案画布」（迭代3.5 Phase A）
 *
 * 设计依据：产品设计文档 2.1 主界面右栏 + 2.5 进度指示。
 *   - 渲染 content_markdown（复用 planDetailConstants 的 renderMarkdown，支持图片/链接）
 *   - 顶部完整度清单：产物驱动（"我的教案还缺什么"），永远不是流程步骤条
 *   - 清单标记词口径对齐后端 DetectLessonPlanContent，不另造判定规则
 *   - 点击缺项 = 等价 send_text 芯片（"帮我补一个XX"），由父组件回调执行
 */

import { renderMarkdown } from '@/pages/lesson-plans/plan-detail/components/planDetailConstants'
import { C } from '../components/workshopConstants'
import { CANVAS_CHECKLIST } from './conversationScript'

interface ConversationCanvasProps {
  /** 教案正文 Markdown（实时随 ContentUpdate SSE 刷新） */
  content: string
  /** AI 是否正在生成中（流式输出时画布顶部显示生成提示） */
  busy: boolean
  /** 点击缺项时回调（label 为缺失部分名称，父组件转为 send_text 补全指令） */
  onFillMissing: (label: string) => void
}

/**
 * 检测正文中各组成部分是否已具备
 * 用最朴素的标记词包含判断 —— 与后端判定口径一致且零成本
 */
function detectChecklist(content: string): Array<{ key: string; label: string; done: boolean }> {
  const text = content || ''
  return CANVAS_CHECKLIST.map(item => ({
    key: item.key,
    label: item.label,
    done: item.patterns.some(p => text.includes(p)),
  }))
}

export default function ConversationCanvas({ content, busy, onFillMissing }: ConversationCanvasProps) {
  const items = detectChecklist(content)
  const doneCount = items.filter(i => i.done).length
  const hasContent = !!(content && content.trim().length > 0)

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', background: C.card }}>

      {/* ===== 顶部：完整度清单（产物驱动） ===== */}
      <div style={{ padding: '12px 16px', borderBottom: `1px solid ${C.border}`, flexShrink: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 700, color: C.text }}>📄 教案画布</span>
          <span style={{ fontSize: '11px', color: doneCount >= items.length ? C.success : C.textMuted, fontWeight: 600 }}>
            {busy ? '✍️ 生成中…' : `完整度 ${doneCount}/${items.length}`}
          </span>
        </div>
        {/* 进度条 */}
        <div style={{ height: '5px', background: '#F3F4F6', borderRadius: '3px', overflow: 'hidden', marginBottom: '8px' }}>
          <div style={{ height: '100%', borderRadius: '3px', width: `${(doneCount / items.length) * 100}%`, background: doneCount >= items.length ? C.success : C.primary, transition: 'width 500ms ease' }} />
        </div>
        {/* 各部分清单胶囊：✓已有(绿) / ○缺失(灰，可点击补全) */}
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
          {items.map(item => (
            <button
              key={item.key}
              onClick={() => { if (!item.done && hasContent) onFillMissing(item.label) }}
              disabled={item.done || !hasContent || busy}
              title={item.done ? `${item.label}已具备` : hasContent ? `点击让AI补充${item.label}` : '正文生成后可点击补全缺项'}
              style={{
                padding: '3px 10px', borderRadius: '12px', fontSize: '11px',
                border: `1px solid ${item.done ? 'rgba(16,185,129,0.3)' : C.border}`,
                background: item.done ? 'rgba(16,185,129,0.08)' : 'transparent',
                color: item.done ? C.success : C.textMuted,
                fontWeight: item.done ? 600 : 400,
                cursor: item.done || !hasContent || busy ? 'default' : 'pointer',
                transition: 'all 150ms ease',
              }}
            >
              {item.done ? '✓' : '○'} {item.label}
            </button>
          ))}
        </div>
      </div>

      {/* ===== 正文：活文档渲染 ===== */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '16px 20px', boxSizing: 'border-box' }}>
        {hasContent ? (
          <div style={{ fontSize: '13px', lineHeight: 1.8 }}>{renderMarkdown(content)}</div>
        ) : (
          <div style={{ height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: C.textMuted, textAlign: 'center', padding: '24px' }}>
            <div style={{ fontSize: '36px', marginBottom: '14px' }}>📝</div>
            <div style={{ fontSize: '14px', lineHeight: 1.8 }}>
              你的教案会在这里一点点长出来<br />
              和左边的AI导师聊聊，或点建议按钮推进
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

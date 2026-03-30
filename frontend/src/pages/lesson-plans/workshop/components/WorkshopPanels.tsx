/**
 * WorkshopPanels.tsx — 备课工坊各面板子组件
 *   StartForm         — 首屏备课表单
 *   AIBubble          — AI消息气泡（支持Markdown+组件选择）
 *   UserBubble        — 用户消息气泡
 *   ThinkingIndicator — AI思考中动画
 *   ReviewPanel       — AI评审结果面板
 */
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { ConversationMessage, AIReviewResult, ConvComponent } from '@/api/lesson-plans'
import { C, SUBJECTS, GRADES, renderMarkdown } from './workshopConstants'

// ==================== 首屏备课表单 ====================

interface StartFormProps {
  onStart: (subject: string, grade: string, topic: string, duration: number) => void
  loading: boolean
}

export function StartForm({ onStart, loading }: StartFormProps) {
  const [subject, setSubject]   = useState('AI')
  const [grade, setGrade]       = useState('七年级')
  const [topic, setTopic]       = useState('')
  const [duration, setDuration] = useState(45)
  const navigate = useNavigate()

  const handleSubmit = () => { if (!topic.trim()) return; onStart(subject, grade, topic.trim(), duration) }

  const shortcuts = [
    { icon: '📋', text: '我的教案',   path: '/lesson-plans/my-plans' },
    { icon: '📚', text: '教案库',     path: '/lesson-plans/library'  },
    { icon: '📐', text: '提示词模板', path: '/lesson-plans/templates' },
  ]

  // 选择按钮通用样式生成
  const selBtn = (active: boolean): React.CSSProperties => ({
    padding: '6px 14px', borderRadius: '20px',
    border: `1px solid ${active ? C.primary : C.border}`,
    background: active ? C.primaryLight : 'transparent',
    color: active ? C.primary : C.textSec,
    fontSize: '13px', fontWeight: active ? 600 : 400,
    cursor: 'pointer', transition: 'all 150ms ease',
  })

  return (
    <div style={{ maxWidth: '580px', margin: '0 auto', padding: '48px 0' }}>
      {/* 标题 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '24px' }}>
        <span style={{ fontSize: '28px', lineHeight: 1 }}>✨</span>
        <div>
          <h1 style={{ fontSize: '20px', fontWeight: 700, color: C.text, margin: '0 0 2px' }}>开始今天的备课</h1>
          <p style={{ fontSize: '13px', color: C.textSec, margin: 0 }}>告诉AI你要上什么课，它会全程陪你设计出高质量教案</p>
        </div>
      </div>

      <div style={{ background: C.card, borderRadius: '16px', padding: '32px', boxShadow: '0 4px 24px rgba(0,0,0,0.06)', border: `1px solid ${C.border}` }}>
        {/* 学科选择 */}
        <div style={{ marginBottom: '20px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>学科</label>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
            {SUBJECTS.map(s => (
              <button key={s} onClick={() => setSubject(s)} style={selBtn(subject === s)}>{s}</button>
            ))}
          </div>
        </div>

        {/* 年级选择 */}
        <div style={{ marginBottom: '20px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>年级</label>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
            {GRADES.map(g => (
              <button key={g} onClick={() => setGrade(g)} style={selBtn(grade === g)}>{g}</button>
            ))}
          </div>
        </div>

        {/* 课题输入 */}
        <div style={{ marginBottom: '20px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
            课题 <span style={{ color: C.danger }}>*</span>
          </label>
          <input
            type="text" value={topic} onChange={e => setTopic(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleSubmit()}
            placeholder="例如：认识人工智能、图像识别应用..."
            style={{ width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '15px', color: C.text, outline: 'none', boxSizing: 'border-box', transition: 'border-color 150ms ease' }}
            onFocus={e => { e.target.style.borderColor = C.primary }}
            onBlur={e  => { e.target.style.borderColor = C.border  }}
          />
        </div>

        {/* 课时时长 */}
        <div style={{ marginBottom: '28px' }}>
          <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>课时时长</label>
          <div style={{ display: 'flex', gap: '8px' }}>
            {[40, 45, 50, 60].map(d => (
              <button key={d} onClick={() => setDuration(d)} style={selBtn(duration === d)}>{d}分钟</button>
            ))}
          </div>
        </div>

        {/* 开始按钮 */}
        <button
          onClick={handleSubmit} disabled={!topic.trim() || loading}
          style={{ width: '100%', padding: '14px', borderRadius: '10px', border: 'none', background: !topic.trim() || loading ? C.border : C.primary, color: !topic.trim() || loading ? C.textMuted : '#fff', fontSize: '16px', fontWeight: 600, cursor: !topic.trim() || loading ? 'not-allowed' : 'pointer', transition: 'all 200ms ease' }}>
          {loading ? '正在准备备课环境...' : '开始备课 →'}
        </button>
      </div>

      {/* 快捷入口 */}
      <div style={{ display: 'flex', justifyContent: 'center', gap: '8px', marginTop: '24px' }}>
        {shortcuts.map(item => (
          <button key={item.path} onClick={() => navigate(item.path)}
            style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', color: C.textSec, background: 'transparent', border: 'none', padding: '6px 12px', borderRadius: '8px', cursor: 'pointer', transition: 'all 150ms ease' }}
            onMouseEnter={e => { const el = e.currentTarget as HTMLButtonElement; el.style.background = C.primaryLight; el.style.color = C.primary }}
            onMouseLeave={e => { const el = e.currentTarget as HTMLButtonElement; el.style.background = 'transparent'; el.style.color = C.textSec }}>
            <span>{item.icon}</span><span>{item.text}</span>
          </button>
        ))}
      </div>
    </div>
  )
}

// ==================== AI消息气泡 ====================

interface AIBubbleProps {
  msg: ConversationMessage
  streaming?: boolean
  onSelectComponent: (comp: ConvComponent) => void
  selectedComponentIds: Set<string>
}

export function AIBubble({ msg, streaming = false, onSelectComponent, selectedComponentIds }: AIBubbleProps) {
  const [expandedComponent, setExpandedComponent] = useState<string | null>(null)

  return (
    <div style={{ display: 'flex', gap: '10px', marginBottom: '16px', alignItems: 'flex-start' }}>
      {/* AI头像 */}
      <div style={{ width: '32px', height: '32px', flexShrink: 0, background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '14px' }}>✨</div>

      <div style={{ flex: 1, maxWidth: 'calc(100% - 42px)' }}>
        {/* 文本内容 */}
        {msg.content && (
          <div style={{ background: C.aiBubble, borderRadius: '0 12px 12px 12px', padding: '12px 16px', wordBreak: 'break-word' }}>
            {renderMarkdown(msg.content)}
            {/* 流式光标 */}
            {streaming && (
              <span style={{ display: 'inline-block', width: '2px', height: '1em', background: C.primary, marginLeft: '2px', verticalAlign: 'text-bottom', animation: 'cursor-blink 0.8s step-end infinite' }} />
            )}
            <style>{`@keyframes cursor-blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }`}</style>
          </div>
        )}

        {/* 组件推荐卡片 */}
        {msg.type === 'components' && msg.components && msg.components.length > 0 && (
          <div style={{ marginTop: '10px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {msg.components.map(comp => {
              const isSelected = selectedComponentIds.has(comp.id)
              const isExpanded = expandedComponent === comp.id
              return (
                <div key={comp.id} style={{ background: C.card, borderRadius: '10px', border: `1px solid ${isSelected ? C.primary : C.border}`, borderLeft: `3px solid ${C.accent}`, padding: '12px 14px', transition: 'all 200ms ease' }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{comp.display_label}</div>
                      {comp.usage_count > 0 && (
                        <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>
                          {comp.usage_count}位老师用过 · 质量分{comp.quality_score.toFixed(1)}
                        </div>
                      )}
                    </div>
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexShrink: 0, marginLeft: '12px' }}>
                      {comp.design_logic && (
                        <button onClick={() => setExpandedComponent(isExpanded ? null : comp.id)}
                          style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '12px', color: C.textSec, cursor: 'pointer' }}>
                          {isExpanded ? '收起' : '看逻辑'}
                        </button>
                      )}
                      <button onClick={() => onSelectComponent(comp)}
                        style={{ padding: '4px 12px', borderRadius: '6px', border: `1px solid ${isSelected ? C.primary : C.border}`, background: isSelected ? C.primaryLight : 'transparent', fontSize: '13px', color: isSelected ? C.primary : C.textSec, fontWeight: isSelected ? 600 : 400, cursor: 'pointer', transition: 'all 150ms ease' }}>
                        {isSelected ? '✓ 已选' : '选择✓'}
                      </button>
                    </div>
                  </div>
                  {isExpanded && comp.design_logic && (
                    <div style={{ marginTop: '10px', padding: '10px 12px', background: '#F9FAFB', borderRadius: '8px', fontSize: '13px', color: C.textSec, lineHeight: 1.7 }}>
                      {comp.design_logic}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

// ==================== 用户消息气泡 ====================

export function UserBubble({ msg }: { msg: ConversationMessage }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: '16px' }}>
      <div style={{ maxWidth: '75%', background: C.userBubble, border: `1px solid ${C.border}`, borderRadius: '12px 0 12px 12px', padding: '10px 14px', fontSize: '15px', color: C.text, lineHeight: 1.7, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
        {msg.content}
      </div>
    </div>
  )
}

// ==================== 思考中动画 ====================

export function ThinkingIndicator() {
  return (
    <div style={{ display: 'flex', gap: '10px', marginBottom: '16px', alignItems: 'flex-start' }}>
      <div style={{ width: '32px', height: '32px', flexShrink: 0, background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '14px' }}>✨</div>
      <div style={{ background: C.aiBubble, borderRadius: '0 12px 12px 12px', padding: '14px 18px', display: 'flex', alignItems: 'center', gap: '6px' }}>
        {[0,1,2].map(i => (
          <div key={i} style={{ width: '6px', height: '6px', borderRadius: '50%', background: C.primary, animation: `lp-pulse 1.2s ease-in-out ${i * 0.2}s infinite` }} />
        ))}
        <style>{`@keyframes lp-pulse { 0%, 80%, 100% { opacity: 0.3; transform: scale(0.8); } 40% { opacity: 1; transform: scale(1.2); } }`}</style>
      </div>
    </div>
  )
}

// ==================== AI评审面板 ====================

interface ReviewPanelProps {
  review: AIReviewResult
  onApply: (ids?: string[]) => void
  applying: boolean
}

export function ReviewPanel({ review, onApply, applying }: ReviewPanelProps) {
  const isGood = review.total_score >= 8.5
  return (
    <div style={{ padding: '16px', height: '100%', overflowY: 'auto', boxSizing: 'border-box' }}>
      {/* 总分卡片 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '16px', padding: '14px 16px', background: isGood ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)', borderRadius: '10px', border: `1px solid ${isGood ? '#10B98130' : '#F59E0B30'}` }}>
        <div style={{ fontSize: '28px', fontWeight: 700, flexShrink: 0, color: isGood ? C.success : C.accent }}>
          {review.total_score.toFixed(1)}
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>AI综合评分</div>
          <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px', lineHeight: 1.5 }}>{review.summary}</div>
        </div>
      </div>

      {/* 做得好的 */}
      {review.good_points.length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.success, marginBottom: '8px' }}>✅ 做得好的</div>
          {review.good_points.map((point, i) => (
            <div key={i} style={{ fontSize: '13px', color: C.text, lineHeight: 1.6, padding: '6px 10px', marginBottom: '4px', background: 'rgba(16,185,129,0.06)', borderRadius: '6px' }}>
              {point}
            </div>
          ))}
        </div>
      )}

      {/* 可以更好 */}
      {review.improvements.length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.accent, marginBottom: '8px' }}>💡 可以更好</div>
          {review.improvements.map(imp => (
            <div key={imp.id} style={{ marginBottom: '8px', padding: '10px 12px', background: 'rgba(245,158,11,0.06)', borderRadius: '8px', border: '1px solid rgba(245,158,11,0.15)' }}>
              <div style={{ fontSize: '13px', fontWeight: 500, color: C.text, marginBottom: '4px' }}>{imp.issue}</div>
              <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.6 }}>{imp.suggestion}</div>
            </div>
          ))}
        </div>
      )}

      {/* 操作按钮 */}
      <div style={{ display: 'flex', gap: '8px', flexDirection: 'column' }}>
        <button onClick={() => onApply()} disabled={applying}
          style={{ padding: '10px', borderRadius: '8px', border: 'none', background: applying ? C.border : C.primary, color: applying ? C.textMuted : '#fff', fontSize: '14px', fontWeight: 600, cursor: applying ? 'not-allowed' : 'pointer' }}>
          {applying ? '优化中...' : 'AI帮我优化 ✨'}
        </button>
        <button onClick={() => onApply([])} disabled={applying}
          style={{ padding: '10px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>
          就这样够用了 👍
        </button>
      </div>
    </div>
  )
}

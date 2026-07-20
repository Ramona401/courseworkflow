/**
 * WorkshopPanels.tsx — 备课工坊各面板子组件
 *
 * 组件列表：
 *   StartForm         — 首屏备课表单
 *   AIBubble          — AI消息气泡（支持Markdown+组件选择）
 *   UserBubble        — 用户消息气泡
 *   ThinkingIndicator — AI思考中动画
 *   ReviewPanel       — AI评审结果面板
 *
 * v203变更（专家模式首屏简化）：
 *   - StartForm 从双栏布局（左基本信息+右320px黄色配方面板）改为单栏布局
 *   - 配方选择从整版面板改为与对话模式一致的下拉选择器
 *   - 去掉"当前学科/全部配方"切换器、去掉"新建配方"入口（需要的去配方管理页）
 *   - 保留课本图片区域不变
 */
import { useState } from 'react'
import type {
  ConversationMessage,
  AIReviewResult,
  ConvComponent,
} from '@/api/lesson-plans'
import { C, renderMarkdown } from './workshopConstants'
import ContextReceiptCard from './context-receipt/ContextReceiptCard'

// ==================== 首屏备课表单 ====================

// ==================== 首屏备课表单 ====================

export { default as StartForm } from './EducationAwareStartForm'

// ==================== AI消息气泡 ====================

interface AIBubbleProps {
  msg: ConversationMessage
  streaming?: boolean
  onSelectComponent: (comp: ConvComponent) => void
  selectedComponentIds: Set<string>
  /**
   * 消息列表层计算后的回执显示结果。
   * 默认true，兼容其它尚未接入列表去重的调用点。
   */
  showContextReceipt?: boolean
}

export function AIBubble({
  msg,
  streaming = false,
  onSelectComponent,
  selectedComponentIds,
  showContextReceipt = true,
}: AIBubbleProps) {
  const [expandedComponent, setExpandedComponent] = useState<string | null>(null)

  return (
    <div style={{ display: 'flex', gap: '10px', marginBottom: '16px', alignItems: 'flex-start' }}>
      <div style={{ width: '32px', height: '32px', flexShrink: 0, background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '14px' }}>✨</div>
      <div style={{ flex: 1, maxWidth: 'calc(100% - 42px)' }}>
        {msg.content && (
          <div style={{ background: C.aiBubble, borderRadius: '0 12px 12px 12px', padding: '12px 16px', wordBreak: 'break-word' }}>
            {renderMarkdown(msg.content)}
            {streaming && (
              <span style={{ display: 'inline-block', width: '2px', height: '1em', background: C.primary, marginLeft: '2px', verticalAlign: 'text-bottom', animation: 'cursor-blink 0.8s step-end infinite' }} />
            )}
            <style>{`@keyframes cursor-blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }`}</style>
          </div>
        )}

        {showContextReceipt && (
          <ContextReceiptCard
            receipt={msg.metadata?.context_receipt}
          />
        )}

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
                      {comp.usage_count > 0 && <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>{comp.usage_count}位老师用过 · 质量分{comp.quality_score.toFixed(1)}</div>}
                    </div>
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexShrink: 0, marginLeft: '12px' }}>
                      {comp.design_logic && (
                        <button onClick={() => setExpandedComponent(isExpanded ? null : comp.id)} style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '12px', color: C.textSec, cursor: 'pointer' }}>
                          {isExpanded ? '收起' : '看逻辑'}
                        </button>
                      )}
                      <button onClick={() => onSelectComponent(comp)} style={{ padding: '4px 12px', borderRadius: '6px', border: `1px solid ${isSelected ? C.primary : C.border}`, background: isSelected ? C.primaryLight : 'transparent', fontSize: '13px', color: isSelected ? C.primary : C.textSec, fontWeight: isSelected ? 600 : 400, cursor: 'pointer', transition: 'all 150ms ease' }}>
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

function stripBoldFE(s: unknown): string {
  return String(s ?? '').replace(/\*/g, '').trim()
}
function isHeaderDimFE(name: string): boolean {
  if (!name) return true
  const headers = ['评审维度', '维度', '评分', '简短评语', '评语', '得分', '分数']
  return headers.includes(name)
}

interface ReviewDimension {
  code?: string
  name?: string
  score?: number
  comment?: string
}

interface ReviewPanelProps {
  review: AIReviewResult
  onApply: (ids?: string[]) => void
  applying: boolean
  isStageMode?: boolean
}

export function ReviewPanel({ review, onApply, applying, isStageMode = false }: ReviewPanelProps) {
  const isGood = review.total_score >= 8.5

  const rawDims = (review as unknown as { dimensions?: ReviewDimension[] }).dimensions
  const cleanDims: ReviewDimension[] = Array.isArray(rawDims)
    ? rawDims
        .map(d => ({
          code: stripBoldFE(d.code),
          name: stripBoldFE(d.name),
          score: typeof d.score === 'number' ? d.score : undefined,
          comment: typeof d.comment === 'string' ? d.comment : '',
        }))
        .filter(d => !isHeaderDimFE(d.name || '') && typeof d.score === 'number')
    : []

  return (
    <div style={{ padding: '16px', height: '100%', overflowY: 'auto', boxSizing: 'border-box' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '16px', padding: '14px 16px', background: isGood ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)', borderRadius: '10px', border: `1px solid ${isGood ? '#10B98130' : '#F59E0B30'}` }}>
        <div style={{ fontSize: '28px', fontWeight: 700, flexShrink: 0, color: isGood ? C.success : C.accent }}>
          {review.total_score.toFixed(1)}
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>AI综合评分</div>
          <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px', lineHeight: 1.5 }}>{review.summary}</div>
        </div>
      </div>

      {cleanDims.length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>📊 各维度评分</div>
          {cleanDims.map((d, i) => {
            const sc = d.score as number
            const barGood = sc >= 8.5
            return (
              <div key={i} style={{ marginBottom: '8px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '3px' }}>
                  <span style={{ fontSize: '12px', color: C.textSec, flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {d.code ? `${d.code} ` : ''}{d.name || `维度${i + 1}`}
                  </span>
                  <span style={{ fontSize: '12px', fontWeight: 600, color: barGood ? C.success : C.accent, width: '32px', textAlign: 'right', flexShrink: 0 }}>
                    {sc.toFixed(1)}
                  </span>
                </div>
                <div style={{ height: '6px', background: '#F3F4F6', borderRadius: '3px', overflow: 'hidden' }}>
                  <div style={{ height: '100%', borderRadius: '3px', width: `${Math.min(100, sc * 10)}%`, background: barGood ? C.success : C.accent, transition: 'width 600ms ease' }} />
                </div>
                {d.comment && (
                  <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '3px', lineHeight: 1.5 }}>{d.comment}</div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {(review.good_points || []).length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.success, marginBottom: '8px' }}>✅ 做得好的</div>
          {(review.good_points || []).map((point, i) => (
            <div key={i} style={{ fontSize: '13px', color: C.text, lineHeight: 1.6, padding: '6px 10px', marginBottom: '4px', background: 'rgba(16,185,129,0.06)', borderRadius: '6px' }}>
              {point}
            </div>
          ))}
        </div>
      )}
      {(review.improvements || []).length > 0 && (
        <div style={{ marginBottom: '16px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.accent, marginBottom: '8px' }}>💡 可以更好</div>
          {(review.improvements || []).map(imp => (
            <div key={imp.id} style={{ marginBottom: '8px', padding: '10px 12px', background: 'rgba(245,158,11,0.06)', borderRadius: '8px', border: '1px solid rgba(245,158,11,0.15)' }}>
              <div style={{ fontSize: '13px', fontWeight: 500, color: C.text, marginBottom: '4px' }}>{imp.issue}</div>
              <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.6 }}>{imp.suggestion}</div>
            </div>
          ))}
        </div>
      )}
      {!isStageMode && (
        <button onClick={() => onApply()} disabled={applying} style={{ width: '100%', padding: '10px', borderRadius: '8px', border: 'none', background: applying ? '#E5E7EB' : C.primary, color: applying ? C.textMuted : '#fff', fontSize: '13px', fontWeight: 600, cursor: applying ? 'not-allowed' : 'pointer' }}>
          {applying ? '应用中...' : '✨ 一键应用全部建议'}
        </button>
      )}
      {isStageMode && (
        <div style={{ padding: '10px 12px', borderRadius: '8px', background: 'rgba(79,123,232,0.06)', fontSize: '12px', color: '#4F7BE8', textAlign: 'center', lineHeight: 1.6 }}>
          💡 进入"修订定稿"阶段与AI讨论如何修改
        </div>
      )}
    </div>
  )
}

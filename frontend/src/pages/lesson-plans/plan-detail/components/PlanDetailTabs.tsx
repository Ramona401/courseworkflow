/**
 * PlanDetailTabs.tsx — 教案详情页四个Tab内容组件
 *
 * v105改动（P2-8 / P2-10 / P3-11）：
 *   P2-8：批注汇总提示条新增「展开全部批注」/「收起全部」快捷按钮
 *   P2-10：新增「批量AI修改」入口，弹出 BatchAIFixPanel 逐条处理
 *   P3-11：AIFixPanel / ManualEditBox / AnnotationBubble 拆分为独立文件
 */
/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState, useEffect, useRef } from 'react'
import type { LessonPlan, AIReviewResult } from '@/api/lesson-plans'
import { updateLessonPlan } from '@/api/lesson-plans'
import { getPipelineDetail } from '@/api/pipelines'
import { resolveAnnotation, type Annotation } from '@/api/annotations'
import { C, PIPELINE_STATUS_LABEL, STEP_ORDER, STEP_NAME_MAP, fmtDate, renderMarkdown } from './planDetailConstants'
import { AIFixPanel } from './AIFixPanel'
import { ManualEditBox } from './ManualEditBox'
import { AnnotationBubble } from './AnnotationBubble'
import { BatchAIFixPanel } from './BatchAIFixPanel'

type PipelineDetail = Awaited<ReturnType<typeof getPipelineDetail>>

function splitIntoParagraphs(md: string): string[] {
  if (!md) return []
  return md.split(/\n\s*\n/).map(p => p.trim()).filter(p => p.length > 0)
}

function joinParagraphs(paragraphs: string[]): string {
  return paragraphs.join('\n\n')
}

// ==================== 段落组件 ====================

function ParagraphWithAnnotation({
  paragraph, index, annotations, currentRound, isOwner,
  planContext, forceExpand, onResolve, onParagraphChange,
}: {
  paragraph: string
  index: number
  annotations: Annotation[]
  currentRound: number
  isOwner: boolean
  planContext: string
  forceExpand: boolean
  onResolve: (id: string, status: 'pending' | 'resolved') => void
  onParagraphChange: (index: number, newContent: string, annotationId?: string) => void
}) {
  const [showAnnotations, setShowAnnotations] = useState(false)
  const [showHistory, setShowHistory]         = useState(false)
  const [activePanel, setActivePanel]         = useState<string | null>(null)
  const [activeAnnotation, setActiveAnnotation] = useState<Annotation | null>(null)

  useEffect(() => { setShowAnnotations(forceExpand) }, [forceExpand])

  const myAnnotations           = annotations.filter(a => a.paragraph_index === index)
  const currentRoundAnnotations = myAnnotations.filter(a => a.review_round === currentRound)
  const historicalAnnotations   = myAnnotations.filter(a => a.review_round < currentRound)
  const pendingCount            = currentRoundAnnotations.filter(a => a.status === 'pending').length
  const hasAnnotations          = myAnnotations.length > 0
  const hasCurrentAnnotations   = currentRoundAnnotations.length > 0

  const handleAIFix = (annotation: Annotation) => {
    setActiveAnnotation(annotation)
    setActivePanel(`ai-fix:${annotation.id}`)
    setShowAnnotations(true)
  }

  return (
    <div style={{ position: 'relative', marginBottom: '4px' }}>
      <div style={{
        display: 'flex',
        borderLeft: hasCurrentAnnotations ? '3px solid #F97316' : hasAnnotations ? '3px solid #E5E7EB' : '3px solid transparent',
        paddingLeft: hasAnnotations ? '12px' : '0',
        transition: 'border-color 200ms ease',
      }}>
        <div style={{ flex: 1, fontSize: '14px', lineHeight: 1.9 }}>{renderMarkdown(paragraph)}</div>
        <div style={{ flexShrink: 0, marginLeft: '8px', alignSelf: 'flex-start', paddingTop: '4px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
          {hasAnnotations && (
            <button
              onClick={() => setShowAnnotations(!showAnnotations)}
              style={{ position: 'relative', padding: '4px 8px', borderRadius: '20px', border: 'none', background: showAnnotations ? '#FEF3C7' : '#FFF7ED', color: '#92400E', fontSize: '12px', fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap', boxShadow: '0 1px 3px rgba(0,0,0,0.08)' }}
            >
              💬 {myAnnotations.length}
              {pendingCount > 0 && (
                <span style={{ position: 'absolute', top: '-4px', right: '-4px', width: '14px', height: '14px', borderRadius: '50%', background: '#EF4444', color: '#fff', fontSize: '9px', fontWeight: 700, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>{pendingCount}</span>
              )}
            </button>
          )}
          {isOwner && (
            <button
              onClick={() => setActivePanel(activePanel === 'manual' ? null : 'manual')}
              style={{ padding: '4px 8px', borderRadius: '20px', border: 'none', background: activePanel === 'manual' ? '#D1FAE5' : '#F3F4F6', color: '#374151', fontSize: '12px', cursor: 'pointer', whiteSpace: 'nowrap', boxShadow: '0 1px 3px rgba(0,0,0,0.06)' }}
            >✏️</button>
          )}
        </div>
      </div>

      {showAnnotations && myAnnotations.length > 0 && (
        <div style={{ margin: '8px 0 4px 15px', padding: '12px', background: '#FFFBEB', borderRadius: '8px', border: '1px solid #FED7AA' }}>
          {currentRoundAnnotations.length > 0 && (
            <>
              <div style={{ fontSize: '12px', fontWeight: 600, color: '#92400E', marginBottom: '8px' }}>
                本轮批注（第{currentRound}轮，{currentRoundAnnotations.length}条）
              </div>
              {currentRoundAnnotations.map(a => (
                <div key={a.id}>
                  <AnnotationBubble annotation={a} isOwner={isOwner} paragraphContent={paragraph} onResolve={onResolve} onAIFix={handleAIFix} isHistorical={false} />
                  {activePanel === `ai-fix:${a.id}` && activeAnnotation?.id === a.id && (
                    <AIFixPanel
                      annotation={a}
                      paragraphContent={paragraph}
                      planContext={planContext}
                      onAdopt={(suggestion) => {
                        onParagraphChange(index, suggestion, a.id)
                        setActivePanel(null)
                        setActiveAnnotation(null)
                      }}
                      onClose={() => { setActivePanel(null); setActiveAnnotation(null) }}
                    />
                  )}
                </div>
              ))}
            </>
          )}
          {historicalAnnotations.length > 0 && (
            <div style={{ marginTop: currentRoundAnnotations.length > 0 ? '8px' : 0 }}>
              <button onClick={() => setShowHistory(!showHistory)} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '12px', color: '#9CA3AF', padding: '2px 0', display: 'flex', alignItems: 'center', gap: '4px' }}>
                {showHistory ? '▾' : '▸'} 历史批注（{historicalAnnotations.length}条，已归档）
              </button>
              {showHistory && historicalAnnotations.map(a => (
                <AnnotationBubble key={a.id} annotation={a} isOwner={isOwner} paragraphContent={paragraph} onResolve={onResolve} onAIFix={handleAIFix} isHistorical={true} />
              ))}
            </div>
          )}
        </div>
      )}

      {activePanel === 'manual' && (
        <ManualEditBox
          initialContent={paragraph}
          onSave={(newContent) => { onParagraphChange(index, newContent); setActivePanel(null) }}
          onClose={() => setActivePanel(null)}
        />
      )}
    </div>
  )
}

// ==================== Tab: 教案内容 ====================

interface ContentTabProps {
  plan: LessonPlan
  isOwner?: boolean
  annotations: Annotation[]
  annotationsLoading: boolean
  onAnnotationsChange: (updated: Annotation[]) => void
}

export function ContentTab({ plan, isOwner, annotations, annotationsLoading, onAnnotationsChange }: ContentTabProps) {
  const [paragraphs, setParagraphs]         = useState<string[]>([])
  const [saveToast, setSaveToast]           = useState<string | null>(null)
  const [allExpanded, setAllExpanded]       = useState(false)
  const [showBatchPanel, setShowBatchPanel] = useState(false)
  const isSavingRef  = useRef(false)
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const hasContent = !!(plan.content_markdown && plan.content_markdown.trim().length > 0)

  useEffect(() => {
    if (hasContent) setParagraphs(splitIntoParagraphs(plan.content_markdown!))
  }, [plan.content_markdown, hasContent])

  const currentRound = annotations.reduce((max, a) => Math.max(max, a.review_round ?? 1), 1)

  const planContext = [
    `学科：${plan.subject}  年级：${plan.grade}  课题：${plan.topic}  课时：${plan.duration_minutes}分钟`,
    plan.content_markdown ? `\n【教案正文】\n${plan.content_markdown}` : '',
  ].filter(Boolean).join('\n')

  const handleResolve = async (annotationId: string, status: 'pending' | 'resolved') => {
    try {
      await resolveAnnotation(plan.id, annotationId, status)
      onAnnotationsChange(annotations.map(a => a.id === annotationId ? { ...a, status } : a))
    } catch { alert('操作失败，请重试') }
  }

  const handleParagraphChange = async (index: number, newContent: string, annotationId?: string) => {
    const updated = paragraphs.map((p, i) => i === index ? newContent : p)
    setParagraphs(updated)
    const newMarkdown = joinParagraphs(updated)
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current)
    saveTimerRef.current = setTimeout(async () => {
      if (isSavingRef.current) { showSaveToast('正在保存中，请稍候...'); return }
      isSavingRef.current = true
      try {
        await updateLessonPlan(plan.id, { content_markdown: newMarkdown })
        showSaveToast('段落已保存 ✓')
        if (annotationId) {
          try {
            await resolveAnnotation(plan.id, annotationId, 'resolved')
            onAnnotationsChange(annotations.map(a => a.id === annotationId ? { ...a, status: 'resolved' as const } : a))
          } catch { /* 批注标记失败不阻断主流程 */ }
        }
      } catch { showSaveToast('保存失败，请重试') }
      finally { isSavingRef.current = false }
    }, 300)
  }

  const showSaveToast = (msg: string) => { setSaveToast(msg); setTimeout(() => setSaveToast(null), 3000) }

  const currentRoundAnnotations = annotations.filter(a => (a.review_round ?? 1) === currentRound)
  const pendingTotal             = currentRoundAnnotations.filter(a => a.status === 'pending').length

  const batchItems = currentRoundAnnotations
    .filter(a => a.status === 'pending')
    .map(a => ({ annotation: a, paragraphContent: paragraphs[a.paragraph_index] ?? '', paragraphIndex: a.paragraph_index }))
    .filter(item => item.paragraphContent.length > 0)

  if (!hasContent) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted }}>
        <div style={{ fontSize: '40px', marginBottom: '12px' }}>📄</div>
        <div style={{ fontSize: '15px', fontWeight: 500, color: C.textSec, marginBottom: '6px' }}>暂无教案内容</div>
        <div style={{ fontSize: '13px', lineHeight: 1.7 }}>前往备课工坊与AI一起生成教案内容</div>
      </div>
    )
  }

  return (
    <div style={{ padding: '28px', position: 'relative' }}>

      {!annotationsLoading && annotations.length > 0 && (
        <div style={{
          padding: '10px 16px', borderRadius: '8px', marginBottom: '20px',
          background: pendingTotal > 0 ? '#FFF7ED' : '#F0FDF4',
          border: `1px solid ${pendingTotal > 0 ? '#FED7AA' : '#BBF7D0'}`,
          fontSize: '13px',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
            <span style={{ fontSize: '16px' }}>{pendingTotal > 0 ? '💬' : '✅'}</span>
            <span style={{ color: pendingTotal > 0 ? '#92400E' : '#166534', fontWeight: 500, flex: 1, minWidth: 0 }}>
              {pendingTotal > 0
                ? `本轮共 ${currentRoundAnnotations.length} 条批注，其中 ${pendingTotal} 条待处理`
                : `本轮 ${currentRoundAnnotations.length} 条批注全部已处理 ✓`
              }
            </span>
            {currentRoundAnnotations.length > 0 && (
              <button
                onClick={() => setAllExpanded(prev => !prev)}
                style={{ padding: '4px 12px', borderRadius: '20px', border: `1px solid ${allExpanded ? '#F97316' : '#FED7AA'}`, background: allExpanded ? 'rgba(249,115,22,0.08)' : 'transparent', fontSize: '12px', fontWeight: 600, color: allExpanded ? '#C2410C' : '#92400E', cursor: 'pointer', whiteSpace: 'nowrap', transition: 'all 150ms ease' }}
              >{allExpanded ? '🔼 收起全部批注' : '🔽 展开全部批注'}</button>
            )}
            {pendingTotal > 0 && isOwner && batchItems.length > 0 && (
              <button
                onClick={() => setShowBatchPanel(true)}
                style={{ padding: '4px 14px', borderRadius: '20px', border: 'none', background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', fontSize: '12px', fontWeight: 600, color: '#fff', cursor: 'pointer', whiteSpace: 'nowrap', boxShadow: '0 2px 8px rgba(79,123,232,0.3)' }}
              >🤖 批量AI修改（{batchItems.length}条）</button>
            )}
          </div>
          {pendingTotal > 0 && (
            <div style={{ marginTop: '6px', fontSize: '12px', color: '#92400E', opacity: 0.8 }}>
              点击段落旁的 💬 图标逐条处理，或使用「批量AI修改」一键处理全部
            </div>
          )}
        </div>
      )}

      {paragraphs.map((para, idx) => (
        <ParagraphWithAnnotation
          key={idx}
          paragraph={para}
          index={idx}
          annotations={annotations}
          currentRound={currentRound}
          isOwner={isOwner ?? false}
          planContext={planContext}
          forceExpand={allExpanded}
          onResolve={handleResolve}
          onParagraphChange={handleParagraphChange}
        />
      ))}

      {saveToast && (
        <div style={{ position: 'fixed', bottom: '80px', right: '32px', padding: '10px 20px', borderRadius: '8px', background: saveToast.includes('失败') ? '#FEF2F2' : '#1F2937', color: saveToast.includes('失败') ? '#DC2626' : '#fff', fontSize: '13px', fontWeight: 500, boxShadow: '0 4px 16px rgba(0,0,0,0.12)', zIndex: 9999 }}>{saveToast}</div>
      )}

      {showBatchPanel && (
        <BatchAIFixPanel
          items={batchItems}
          planContext={planContext}
          onAdopt={(paragraphIndex, newContent, annotationId) => { handleParagraphChange(paragraphIndex, newContent, annotationId) }}
          onSkip={(_annotationId) => { /* 跳过，用户可手动处理 */ }}
          onClose={() => setShowBatchPanel(false)}
        />
      )}
    </div>
  )
}

// ==================== Tab: AI评审 ====================

export function ReviewTab({ plan }: { plan: LessonPlan }) {
  let review: AIReviewResult | null = null
  if (plan.ai_review_result) {
    try {
      review = typeof plan.ai_review_result === 'string' ? JSON.parse(plan.ai_review_result) : plan.ai_review_result as AIReviewResult
      if (!review || !review.total_score) review = null
    } catch { review = null }
  }
  if (!review) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted }}>
        <div style={{ fontSize: '40px', marginBottom: '12px' }}>🤖</div>
        <div style={{ fontSize: '15px', fontWeight: 500, color: C.textSec, marginBottom: '6px' }}>尚未进行AI评审</div>
        <div style={{ fontSize: '13px', lineHeight: 1.7 }}>在备课工坊生成教案后，点击"AI评审"可获取质量分析</div>
      </div>
    )
  }
  const isGood = review.total_score >= 8.5
  return (
    <div style={{ padding: '28px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '20px', padding: '20px 24px', borderRadius: '12px', marginBottom: '24px', background: isGood ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)', border: `1px solid ${isGood ? '#10B98130' : '#F59E0B30'}` }}>
        <div style={{ fontSize: '48px', fontWeight: 700, flexShrink: 0, lineHeight: 1, color: isGood ? C.success : C.accent }}>{review.total_score.toFixed(1)}</div>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
            AI综合评分<span style={{ marginLeft: '10px', fontSize: '12px', fontWeight: 400, color: C.textMuted }}>{review.reviewed_at ? `评审于 ${review.reviewed_at.slice(0, 10)}` : ''}</span>
          </div>
          <div style={{ fontSize: '14px', color: C.textSec, lineHeight: 1.7 }}>{review.summary}</div>
        </div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '24px' }}>
        {review.good_points?.length > 0 && (
          <div>
            <div style={{ fontSize: '14px', fontWeight: 600, color: C.success, marginBottom: '10px' }}>✅ 做得好的</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              {review.good_points.map((p, i) => <div key={i} style={{ padding: '10px 12px', borderRadius: '8px', background: 'rgba(16,185,129,0.06)', border: '1px solid rgba(16,185,129,0.12)', fontSize: '13px', color: C.text, lineHeight: 1.6 }}>{p}</div>)}
            </div>
          </div>
        )}
        {review.improvements?.length > 0 && (
          <div>
            <div style={{ fontSize: '14px', fontWeight: 600, color: C.accent, marginBottom: '10px' }}>💡 可以更好</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {review.improvements.map(imp => (
                <div key={imp.id} style={{ padding: '10px 12px', borderRadius: '8px', background: 'rgba(245,158,11,0.06)', border: '1px solid rgba(245,158,11,0.15)' }}>
                  <div style={{ fontSize: '13px', fontWeight: 500, color: C.text, marginBottom: '4px' }}>{imp.issue}</div>
                  <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.6 }}>{imp.suggestion}</div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
      {review.dimensions?.length > 0 && (
        <div>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '12px' }}>📊 各维度详情</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {review.dimensions.map(dim => {
              const dimColor = dim.score >= 8 ? C.success : dim.score >= 6 ? C.accent : C.danger
              return (
                <div key={dim.code} style={{ display: 'flex', alignItems: 'flex-start', gap: '16px', padding: '12px 16px', borderRadius: '8px', background: C.bg, border: `1px solid ${C.border}` }}>
                  <div style={{ flexShrink: 0, textAlign: 'center', minWidth: '52px' }}>
                    <div style={{ fontSize: '12px', fontWeight: 700, color: C.textMuted }}>{dim.code}</div>
                    <div style={{ fontSize: '22px', fontWeight: 700, lineHeight: 1.2, color: dimColor }}>{dim.score}</div>
                  </div>
                  <div style={{ flex: 1, paddingTop: '4px' }}>
                    <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>{dim.name}{dim.good && <span style={{ marginLeft: '6px', fontSize: '11px', color: C.success }}>✓ 优秀</span>}</div>
                    <div style={{ height: '4px', background: C.border, borderRadius: '2px', overflow: 'hidden', marginBottom: '6px' }}>
                      <div style={{ height: '100%', borderRadius: '2px', width: `${dim.score * 10}%`, background: dimColor }} />
                    </div>
                    <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.6 }}>{dim.comment}</div>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}

// ==================== Tab: 使用统计 ====================

export function StatsTab({ plan }: { plan: LessonPlan }) {
  const statItems = [
    { icon: '👁', label: '浏览次数', value: `${(plan as any).view_count ?? 0} 次` },
    { icon: '📋', label: '使用次数', value: `${(plan as any).use_count ?? 0} 次` },
    { icon: '🔀', label: '版本号',   value: `v${plan.version}` },
    { icon: '📅', label: '创建时间', value: fmtDate(plan.created_at) },
    { icon: '🔄', label: '最后更新', value: fmtDate(plan.updated_at) },
  ]
  return (
    <div style={{ padding: '28px' }}>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '16px', marginBottom: '28px' }}>
        {statItems.map(item => (
          <div key={item.label} style={{ padding: '20px', borderRadius: '10px', background: C.bg, border: `1px solid ${C.border}`, display: 'flex', flexDirection: 'column', gap: '6px' }}>
            <span style={{ fontSize: '22px' }}>{item.icon}</span>
            <span style={{ fontSize: '12px', color: C.textMuted, fontWeight: 500 }}>{item.label}</span>
            <span style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>{item.value}</span>
          </div>
        ))}
      </div>
      {(plan as any).forked_from && (
        <div style={{ padding: '16px 20px', borderRadius: '10px', background: 'rgba(139,92,246,0.06)', border: '1px solid rgba(139,92,246,0.15)', display: 'flex', alignItems: 'center', gap: '12px' }}>
          <span style={{ fontSize: '20px' }}>🔀</span>
          <div>
            <div style={{ fontSize: '13px', fontWeight: 600, color: C.purple, marginBottom: '2px' }}>Fork自其他教案</div>
            <div style={{ fontSize: '12px', color: C.textSec }}>原始教案ID：{(plan as any).forked_from}</div>
          </div>
        </div>
      )}
      {plan.ai_review_score != null && (
        <div style={{ marginTop: '24px' }}>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '12px' }}>🤖 AI评审记录</div>
          <div style={{ padding: '16px 20px', borderRadius: '10px', background: C.bg, border: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', gap: '16px' }}>
            <div style={{ fontSize: '32px', fontWeight: 700, color: plan.ai_review_score >= 8.5 ? C.success : C.accent }}>{plan.ai_review_score.toFixed(1)}</div>
            <div>
              <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>最新AI评分</div>
              <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px' }}>{plan.ai_review_score >= 8.5 ? '✅ 达到共享推荐标准（≥8.5）' : '💡 继续优化可提升至推荐标准'}</div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// ==================== Tab: 关联课件 ====================

interface CoursewareTabProps {
  plan: LessonPlan
  onNavigatePipeline: (pipelineId?: string) => void
}

export function CoursewareTab({ plan, onNavigatePipeline }: CoursewareTabProps) {
  const [pipeline, setPipeline]               = useState<PipelineDetail | null>(null)
  const [pipelineLoading, setPipelineLoading] = useState(false)
  const [pipelineError, setPipelineError]     = useState(false)
  const linkedId = plan.linked_pipeline_id

  useEffect(() => {
    if (!linkedId) return
    setPipelineLoading(true); setPipelineError(false)
    getPipelineDetail(linkedId)
      .then(data => { setPipeline(data); setPipelineLoading(false) })
      .catch(() => { setPipelineError(true); setPipelineLoading(false) })
  }, [linkedId])

  if (!['developing', 'completed'].includes(plan.status) && !linkedId) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted }}>
        <div style={{ fontSize: '40px', marginBottom: '12px' }}>🖥️</div>
        <div style={{ fontSize: '15px', fontWeight: 500, color: C.textSec, marginBottom: '6px' }}>尚未进入课件开发</div>
        <div style={{ fontSize: '13px', lineHeight: 1.7, marginBottom: '20px' }}>教案发布后，可进入课件开发流程</div>
        {['published_personal', 'approved', 'published_shared'].includes(plan.status) && (
          <div style={{ display: 'inline-flex', alignItems: 'center', gap: '6px', padding: '8px 14px', borderRadius: '8px', background: C.primaryLight, border: `1px solid ${C.primary}30`, fontSize: '13px', color: C.primary }}>
            💡 点击上方"进入课件开发"按钮开始
          </div>
        )}
      </div>
    )
  }
  if (pipelineLoading) return <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted, fontSize: '14px' }}>加载课件开发信息...</div>
  if (pipelineError || (!pipeline && linkedId)) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted }}>
        <div style={{ fontSize: '32px', marginBottom: '12px' }}>⚠️</div>
        <div style={{ fontSize: '14px', color: C.textSec, marginBottom: '16px' }}>课件开发信息加载失败</div>
        <button onClick={() => onNavigatePipeline(linkedId!)} style={{ padding: '8px 20px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>前往课件审核系统查看</button>
      </div>
    )
  }

  const pStatus    = pipeline?.status || 'pending'
  const pStatusCfg = PIPELINE_STATUS_LABEL[pStatus] || PIPELINE_STATUS_LABEL['pending']
  const steps      = pipeline?.steps || []
  const doneCount  = steps.filter((s: any) => s.status === 'done').length
  const progressPct = Math.round((doneCount / 8) * 100)

  return (
    <div style={{ padding: '28px' }}>
      <div style={{ padding: '20px 24px', borderRadius: '12px', marginBottom: '20px', background: 'rgba(14,165,233,0.06)', border: '1px solid rgba(14,165,233,0.2)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '16px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <div style={{ fontSize: '32px' }}>🖥️</div>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '6px' }}>
              <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>课件开发任务</div>
              <span style={{ fontSize: '11px', fontWeight: 600, padding: '2px 8px', borderRadius: '6px', color: pStatusCfg.color, background: pStatusCfg.bg }}>{pStatusCfg.label}</span>
            </div>
            {pipeline && <div style={{ fontSize: '13px', color: C.textSec }}>课程：{(pipeline as any).course_name || (pipeline as any).course_code}</div>}
          </div>
        </div>
        <button onClick={() => onNavigatePipeline(linkedId!)} style={{ padding: '9px 18px', borderRadius: '8px', border: 'none', background: '#0EA5E9', color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer', flexShrink: 0 }}>查看课件进度 →</button>
      </div>
      {pipeline && (
        <div style={{ marginBottom: '20px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
            <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>执行进度</span>
            <span style={{ fontSize: '13px', color: C.textSec }}>{doneCount}/8 步完成 ({progressPct}%)</span>
          </div>
          <div style={{ height: '6px', background: C.border, borderRadius: '3px', overflow: 'hidden' }}>
            <div style={{ height: '100%', borderRadius: '3px', width: `${progressPct}%`, background: pStatus === 'verified' ? C.success : pStatus === 'failed' ? C.danger : C.primary, transition: 'width 600ms ease' }} />
          </div>
        </div>
      )}
      {pipeline && steps.length > 0 && (
        <div>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '10px' }}>步骤详情</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            {STEP_ORDER.map((stepKey, idx) => {
              const step   = steps.find((s: any) => s.step_name === stepKey)
              const status = step?.status || 'pending'
              const icon   = status === 'done' ? '✅' : status === 'running' ? '⏳' : status === 'failed' ? '❌' : status === 'skipped' ? '⏭' : '⬜'
              const color  = status === 'done' ? C.success : status === 'running' ? C.primary : status === 'failed' ? C.danger : C.textMuted
              return (
                <div key={stepKey} style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '8px 12px', borderRadius: '8px', background: status === 'running' ? C.primaryLight : C.bg, border: `1px solid ${status === 'running' ? C.primary+'30' : C.border}` }}>
                  <span style={{ fontSize: '14px', flexShrink: 0 }}>{icon}</span>
                  <span style={{ fontSize: '12px', color: C.textMuted, flexShrink: 0 }}>{String(idx + 1).padStart(2, '0')}</span>
                  <span style={{ fontSize: '13px', fontWeight: status === 'running' ? 600 : 400, color, flex: 1 }}>{STEP_NAME_MAP[stepKey] || stepKey}</span>
                  {step?.duration_ms && step.duration_ms > 0 && <span style={{ fontSize: '11px', color: C.textMuted }}>{(step.duration_ms / 1000).toFixed(1)}s</span>}
                </div>
              )
            })}
          </div>
        </div>
      )}
      {!pipeline && linkedId && <div style={{ textAlign: 'center', padding: '20px', color: C.textMuted, fontSize: '13px' }}>课件开发任务已创建，请前往课件审核系统查看详情</div>}
    </div>
  )
}

/* eslint-enable @typescript-eslint/no-explicit-any */

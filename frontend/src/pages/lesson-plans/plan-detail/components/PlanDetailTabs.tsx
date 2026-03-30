/**
 * PlanDetailTabs.tsx — 教案详情页四个Tab内容组件
 *   ContentTab    — 📄 教案内容（Markdown渲染）
 *   ReviewTab     — 🤖 AI评审（评分/做得好的/可以更好/维度详情）
 *   StatsTab      — 📊 使用统计（查看数/版本/Fork来源/AI评分记录）
 *   CoursewareTab — 🔗 关联课件（Phase6：接入真实Pipeline数据）
 */
import { useState, useEffect } from 'react'
import type { LessonPlan, AIReviewResult } from '@/api/lesson-plans'
import { getPipelineDetail, type PipelineDetail } from '@/api/pipelines'
import {
  C, PIPELINE_STATUS_LABEL, STEP_ORDER, STEP_NAME_MAP,
  fmtDate, renderMarkdown,
} from './planDetailConstants'

// ==================== Tab: 教案内容 ====================

export function ContentTab({ plan }: { plan: LessonPlan }) {
  const hasContent = plan.content_markdown && plan.content_markdown.trim().length > 0
  return (
    <div style={{ padding: '28px' }}>
      {hasContent ? (
        <div style={{ fontSize: '14px', lineHeight: 1.9 }}>
          {renderMarkdown(plan.content_markdown!)}
        </div>
      ) : (
        <div style={{ textAlign: 'center', padding: '60px 40px', color: C.textMuted }}>
          <div style={{ fontSize: '40px', marginBottom: '12px' }}>📄</div>
          <div style={{ fontSize: '15px', fontWeight: 500, color: C.textSec, marginBottom: '6px' }}>暂无教案内容</div>
          <div style={{ fontSize: '13px', lineHeight: 1.7 }}>前往备课工坊与AI一起生成教案内容</div>
        </div>
      )}
    </div>
  )
}

// ==================== Tab: AI评审 ====================

export function ReviewTab({ plan }: { plan: LessonPlan }) {
  // 解析AI评审结果
  let review: AIReviewResult | null = null
  if (plan.ai_review_result) {
    try {
      review = typeof plan.ai_review_result === 'string'
        ? JSON.parse(plan.ai_review_result)
        : plan.ai_review_result as AIReviewResult
      if (!review || !review.total_score) review = null
    } catch { review = null }
  }

  if (!review) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted }}>
        <div style={{ fontSize: '40px', marginBottom: '12px' }}>🤖</div>
        <div style={{ fontSize: '15px', fontWeight: 500, color: C.textSec, marginBottom: '6px' }}>尚未进行AI评审</div>
        <div style={{ fontSize: '13px', lineHeight: 1.7 }}>
          在备课工坊生成教案后，点击"AI评审"可获取质量分析
        </div>
      </div>
    )
  }

  const isGood = review.total_score >= 8.5

  return (
    <div style={{ padding: '28px' }}>
      {/* 总分卡片 */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: '20px',
        padding: '20px 24px', borderRadius: '12px', marginBottom: '24px',
        background: isGood ? 'rgba(16,185,129,0.08)' : 'rgba(245,158,11,0.08)',
        border: `1px solid ${isGood ? '#10B98130' : '#F59E0B30'}`,
      }}>
        <div style={{ fontSize: '48px', fontWeight: 700, flexShrink: 0, lineHeight: 1, color: isGood ? C.success : C.accent }}>
          {review.total_score.toFixed(1)}
        </div>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
            AI综合评分
            <span style={{ marginLeft: '10px', fontSize: '12px', fontWeight: 400, color: C.textMuted }}>
              {review.reviewed_at ? `评审于 ${review.reviewed_at.slice(0, 10)}` : ''}
            </span>
          </div>
          <div style={{ fontSize: '14px', color: C.textSec, lineHeight: 1.7 }}>{review.summary}</div>
        </div>
      </div>

      {/* 做得好的 + 可以更好（两列）*/}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '24px' }}>
        {/* 做得好的 */}
        {review.good_points?.length > 0 && (
          <div>
            <div style={{ fontSize: '14px', fontWeight: 600, color: C.success, marginBottom: '10px' }}>✅ 做得好的</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              {review.good_points.map((p, i) => (
                <div key={i} style={{
                  padding: '10px 12px', borderRadius: '8px',
                  background: 'rgba(16,185,129,0.06)', border: '1px solid rgba(16,185,129,0.12)',
                  fontSize: '13px', color: C.text, lineHeight: 1.6,
                }}>{p}</div>
              ))}
            </div>
          </div>
        )}
        {/* 可以更好 */}
        {review.improvements?.length > 0 && (
          <div>
            <div style={{ fontSize: '14px', fontWeight: 600, color: C.accent, marginBottom: '10px' }}>💡 可以更好</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {review.improvements.map(imp => (
                <div key={imp.id} style={{
                  padding: '10px 12px', borderRadius: '8px',
                  background: 'rgba(245,158,11,0.06)', border: '1px solid rgba(245,158,11,0.15)',
                }}>
                  <div style={{ fontSize: '13px', fontWeight: 500, color: C.text, marginBottom: '4px' }}>{imp.issue}</div>
                  <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.6 }}>{imp.suggestion}</div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* 各维度详情 */}
      {review.dimensions?.length > 0 && (
        <div>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '12px' }}>📊 各维度详情</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {review.dimensions.map(dim => {
              const dimColor = dim.score >= 8 ? C.success : dim.score >= 6 ? C.accent : C.danger
              return (
                <div key={dim.code} style={{
                  display: 'flex', alignItems: 'flex-start', gap: '16px',
                  padding: '12px 16px', borderRadius: '8px',
                  background: C.bg, border: `1px solid ${C.border}`,
                }}>
                  {/* 维度代码+分数 */}
                  <div style={{ flexShrink: 0, textAlign: 'center', minWidth: '52px' }}>
                    <div style={{ fontSize: '12px', fontWeight: 700, color: C.textMuted }}>{dim.code}</div>
                    <div style={{ fontSize: '22px', fontWeight: 700, lineHeight: 1.2, color: dimColor }}>{dim.score}</div>
                  </div>
                  {/* 维度内容 */}
                  <div style={{ flex: 1, paddingTop: '4px' }}>
                    <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>
                      {dim.name}
                      {dim.good && <span style={{ marginLeft: '6px', fontSize: '11px', color: C.success }}>✓ 优秀</span>}
                    </div>
                    {/* 进度条 */}
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
    { icon: '📋', label: '使用次数', value: `${(plan as any).use_count ?? 0} 次`  },
    { icon: '🔀', label: '版本号',   value: `v${plan.version}`                   },
    { icon: '📅', label: '创建时间', value: fmtDate(plan.created_at)              },
    { icon: '🔄', label: '最后更新', value: fmtDate(plan.updated_at)              },
  ]
  return (
    <div style={{ padding: '28px' }}>
      {/* 统计卡片网格 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '16px', marginBottom: '28px' }}>
        {statItems.map(item => (
          <div key={item.label} style={{
            padding: '20px', borderRadius: '10px', background: C.bg, border: `1px solid ${C.border}`,
            display: 'flex', flexDirection: 'column', gap: '6px',
          }}>
            <span style={{ fontSize: '22px' }}>{item.icon}</span>
            <span style={{ fontSize: '12px', color: C.textMuted, fontWeight: 500 }}>{item.label}</span>
            <span style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>{item.value}</span>
          </div>
        ))}
      </div>

      {/* Fork来源 */}
      {(plan as any).forked_from && (
        <div style={{
          padding: '16px 20px', borderRadius: '10px',
          background: 'rgba(139,92,246,0.06)', border: '1px solid rgba(139,92,246,0.15)',
          display: 'flex', alignItems: 'center', gap: '12px',
        }}>
          <span style={{ fontSize: '20px' }}>🔀</span>
          <div>
            <div style={{ fontSize: '13px', fontWeight: 600, color: C.purple, marginBottom: '2px' }}>Fork自其他教案</div>
            <div style={{ fontSize: '12px', color: C.textSec }}>原始教案ID：{(plan as any).forked_from}</div>
          </div>
        </div>
      )}

      {/* AI评审记录 */}
      {plan.ai_review_score != null && (
        <div style={{ marginTop: '24px' }}>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '12px' }}>🤖 AI评审记录</div>
          <div style={{
            padding: '16px 20px', borderRadius: '10px', background: C.bg, border: `1px solid ${C.border}`,
            display: 'flex', alignItems: 'center', gap: '16px',
          }}>
            <div style={{ fontSize: '32px', fontWeight: 700, color: plan.ai_review_score >= 8.5 ? C.success : C.accent }}>
              {plan.ai_review_score.toFixed(1)}
            </div>
            <div>
              <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>最新AI评分</div>
              <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px' }}>
                {plan.ai_review_score >= 8.5
                  ? '✅ 达到共享推荐标准（≥8.5）'
                  : '💡 继续优化可提升至推荐标准'
                }
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// ==================== Tab: 关联课件（Phase6）====================

interface CoursewareTabProps {
  plan: LessonPlan
  onNavigatePipeline: (pipelineId?: string) => void
}

export function CoursewareTab({ plan, onNavigatePipeline }: CoursewareTabProps) {
  const [pipeline, setPipeline]           = useState<PipelineDetail | null>(null)
  const [pipelineLoading, setPipelineLoading] = useState(false)
  const [pipelineError, setPipelineError]     = useState(false)

  const linkedId = plan.linked_pipeline_id

  // 有关联ID时加载Pipeline详情
  useEffect(() => {
    if (!linkedId) return
    setPipelineLoading(true)
    setPipelineError(false)
    getPipelineDetail(linkedId)
      .then(data => { setPipeline(data); setPipelineLoading(false) })
      .catch(() => { setPipelineError(true); setPipelineLoading(false) })
  }, [linkedId])

  // 未进入课件开发
  if (!['developing', 'completed'].includes(plan.status) && !linkedId) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted }}>
        <div style={{ fontSize: '40px', marginBottom: '12px' }}>🖥️</div>
        <div style={{ fontSize: '15px', fontWeight: 500, color: C.textSec, marginBottom: '6px' }}>尚未进入课件开发</div>
        <div style={{ fontSize: '13px', lineHeight: 1.7, marginBottom: '20px' }}>
          教案发布后，可进入课件开发流程，系统将自动创建课件开发任务
        </div>
        {['published_personal', 'approved', 'published_shared'].includes(plan.status) && (
          <div style={{
            display: 'inline-flex', alignItems: 'center', gap: '6px',
            padding: '8px 14px', borderRadius: '8px',
            background: C.primaryLight, border: `1px solid ${C.primary}30`,
            fontSize: '13px', color: C.primary,
          }}>
            💡 点击上方"进入课件开发"按钮开始
          </div>
        )}
      </div>
    )
  }

  // 加载中
  if (pipelineLoading) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted, fontSize: '14px' }}>
        加载课件开发信息...
      </div>
    )
  }

  // 加载失败
  if (pipelineError || (!pipeline && linkedId)) {
    return (
      <div style={{ padding: '28px', textAlign: 'center', color: C.textMuted }}>
        <div style={{ fontSize: '32px', marginBottom: '12px' }}>⚠️</div>
        <div style={{ fontSize: '14px', color: C.textSec, marginBottom: '16px' }}>课件开发信息加载失败</div>
        <button
          onClick={() => onNavigatePipeline(linkedId!)}
          style={{ padding: '8px 20px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
          前往课件审核系统查看
        </button>
      </div>
    )
  }

  // 有Pipeline数据
  const pStatus    = pipeline?.status || 'pending'
  const pStatusCfg = PIPELINE_STATUS_LABEL[pStatus] || PIPELINE_STATUS_LABEL['pending']
  const steps      = pipeline?.steps || []
  const doneCount  = steps.filter(s => s.status === 'done').length
  const totalSteps = 8
  const progressPct = Math.round((doneCount / totalSteps) * 100)

  return (
    <div style={{ padding: '28px' }}>
      {/* 状态总览卡片 */}
      <div style={{
        padding: '20px 24px', borderRadius: '12px', marginBottom: '20px',
        background: 'rgba(14,165,233,0.06)', border: '1px solid rgba(14,165,233,0.2)',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '16px',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <div style={{ fontSize: '32px' }}>🖥️</div>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '6px' }}>
              <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>课件开发任务</div>
              <span style={{ fontSize: '11px', fontWeight: 600, padding: '2px 8px', borderRadius: '6px', color: pStatusCfg.color, background: pStatusCfg.bg }}>
                {pStatusCfg.label}
              </span>
            </div>
            {pipeline && (
              <div style={{ fontSize: '13px', color: C.textSec }}>
                课程：{pipeline.course_name || pipeline.course_code}
              </div>
            )}
          </div>
        </div>
        <button
          onClick={() => onNavigatePipeline(linkedId!)}
          style={{ padding: '9px 18px', borderRadius: '8px', border: 'none', background: '#0EA5E9', color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer', flexShrink: 0 }}>
          查看课件进度 →
        </button>
      </div>

      {/* 进度条 */}
      {pipeline && (
        <div style={{ marginBottom: '20px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
            <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>执行进度</span>
            <span style={{ fontSize: '13px', color: C.textSec }}>{doneCount}/{totalSteps} 步完成 ({progressPct}%)</span>
          </div>
          <div style={{ height: '6px', background: C.border, borderRadius: '3px', overflow: 'hidden' }}>
            <div style={{
              height: '100%', borderRadius: '3px', width: `${progressPct}%`,
              background: pStatus === 'verified' ? C.success : pStatus === 'failed' ? C.danger : C.primary,
              transition: 'width 600ms ease',
            }} />
          </div>
        </div>
      )}

      {/* 8步骤进度列表 */}
      {pipeline && steps.length > 0 && (
        <div>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '10px' }}>步骤详情</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            {STEP_ORDER.map((stepKey, idx) => {
              const step   = steps.find(s => s.step_name === stepKey)
              const status = step?.status || 'pending'
              const icon   = status === 'done' ? '✅' : status === 'running' ? '⏳' : status === 'failed' ? '❌' : status === 'skipped' ? '⏭' : '⬜'
              const color  = status === 'done' ? C.success : status === 'running' ? C.primary : status === 'failed' ? C.danger : C.textMuted
              return (
                <div key={stepKey} style={{
                  display: 'flex', alignItems: 'center', gap: '10px',
                  padding: '8px 12px', borderRadius: '8px',
                  background: status === 'running' ? C.primaryLight : C.bg,
                  border: `1px solid ${status === 'running' ? C.primary+'30' : C.border}`,
                }}>
                  <span style={{ fontSize: '14px', flexShrink: 0 }}>{icon}</span>
                  <span style={{ fontSize: '12px', color: C.textMuted, flexShrink: 0 }}>
                    {String(idx + 1).padStart(2, '0')}
                  </span>
                  <span style={{ fontSize: '13px', fontWeight: status === 'running' ? 600 : 400, color, flex: 1 }}>
                    {STEP_NAME_MAP[stepKey] || stepKey}
                  </span>
                  {step?.duration_ms && step.duration_ms > 0 && (
                    <span style={{ fontSize: '11px', color: C.textMuted }}>
                      {(step.duration_ms / 1000).toFixed(1)}s
                    </span>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* 无步骤数据时的简要提示 */}
      {!pipeline && linkedId && (
        <div style={{ textAlign: 'center', padding: '20px', color: C.textMuted, fontSize: '13px' }}>
          课件开发任务已创建，请前往课件审核系统查看详情
        </div>
      )}
    </div>
  )
}

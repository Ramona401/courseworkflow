/**
 * StageSummaryModal.tsx — 阶段完成确认弹窗
 *
 * 从 WorkshopTransitionComponents.tsx 拆分
 * 包含: StageSummaryModal(导出) + StageSummaryCards(内部) + parseStructured(内部)
 */

import { useState, useEffect } from 'react'
import { C } from './workshopConstants'
import type { StageProgressItem, StageCompletenessResponse } from '@/api/lesson-plans'

// ==================== 结构化摘要解析 ====================

function parseStructured(structuredOutput: string): Record<string, unknown> | null {
  if (!structuredOutput || structuredOutput === '{}') return null
  try {
    const parsed = JSON.parse(structuredOutput)
    if (typeof parsed !== 'object' || parsed === null) return null
    return parsed as Record<string, unknown>
  } catch {
    return null
  }
}

// ==================== 各阶段结构化摘要卡片 ====================

function StageSummaryCards({ stageCode, structuredOutput }: {
  stageCode: string
  structuredOutput: string
}) {
  const data = parseStructured(structuredOutput)
  if (!data) return null

  const cardStyle: React.CSSProperties = {
    padding: '10px 14px', borderRadius: '10px',
    background: '#F8FAFF', border: '1px solid rgba(79,123,232,0.12)',
    marginBottom: '8px',
  }
  const labelStyle: React.CSSProperties = {
    fontSize: '11px', fontWeight: 700, color: C.textMuted,
    textTransform: 'uppercase' as const, letterSpacing: '0.8px',
    marginBottom: '4px',
  }
  const valueStyle: React.CSSProperties = {
    fontSize: '13px', color: C.text, lineHeight: 1.6,
  }
  const tagStyle: React.CSSProperties = {
    display: 'inline-block', padding: '2px 8px', borderRadius: '10px',
    background: C.primaryLight, color: C.primary,
    fontSize: '12px', fontWeight: 500, margin: '2px 4px 2px 0',
  }

  const renderTags = (arr: unknown) => {
    if (!Array.isArray(arr) || arr.length === 0) return null
    return (
      <div style={{ marginTop: '2px' }}>
        {arr.slice(0, 6).map((item, i) => (
          <span key={i} style={tagStyle}>{String(item)}</span>
        ))}
        {arr.length > 6 && <span style={{ ...tagStyle, background: '#F3F4F6', color: C.textMuted }}>+{arr.length - 6}</span>}
      </div>
    )
  }

  const renderStr = (val: unknown, fallback?: string) => {
    const s = typeof val === 'string' ? val.trim() : ''
    return s || fallback || null
  }

  switch (stageCode) {
    case 'analyze': {
      const sp = data.student_profile as Record<string, unknown> | null
      return (
        <div>
          {renderStr(data.textbook_analysis) && (
            <div style={cardStyle}>
              <div style={labelStyle}>📚 教材分析</div>
              <div style={valueStyle}>{renderStr(data.textbook_analysis)}</div>
            </div>
          )}
          {Array.isArray(data.curriculum_standards) && data.curriculum_standards.length > 0 && (
            <div style={cardStyle}>
              <div style={labelStyle}>📋 课程标准</div>
              {renderTags(data.curriculum_standards)}
            </div>
          )}
          {sp && (
            <div style={cardStyle}>
              <div style={labelStyle}>👥 学情分析</div>
              {renderStr(sp.prior_knowledge) && (
                <div style={valueStyle}><strong>已有基础：</strong>{renderStr(sp.prior_knowledge)}</div>
              )}
              {Array.isArray(sp.common_difficulties) && sp.common_difficulties.length > 0 && (
                <div style={{ marginTop: '4px' }}>
                  <strong style={{ fontSize: '13px' }}>常见难点：</strong>
                  {renderTags(sp.common_difficulties)}
                </div>
              )}
            </div>
          )}
          {Array.isArray(data.key_concepts) && data.key_concepts.length > 0 && (
            <div style={cardStyle}>
              <div style={labelStyle}>🔑 核心概念</div>
              {renderTags(data.key_concepts)}
            </div>
          )}
        </div>
      )
    }

    case 'design': {
      const obj = data.objectives as Record<string, unknown> | null
      const acts = data.activities as Array<Record<string, unknown>> | null
      return (
        <div>
          {obj && (
            <div style={cardStyle}>
              <div style={labelStyle}>🎯 教学目标</div>
              {Array.isArray(obj.knowledge) && obj.knowledge.length > 0 && (
                <div style={{ marginBottom: '4px' }}>
                  <span style={{ fontSize: '12px', color: C.textMuted }}>知识目标：</span>
                  {renderTags(obj.knowledge)}
                </div>
              )}
              {Array.isArray(obj.ability) && obj.ability.length > 0 && (
                <div style={{ marginBottom: '4px' }}>
                  <span style={{ fontSize: '12px', color: C.textMuted }}>能力目标：</span>
                  {renderTags(obj.ability)}
                </div>
              )}
              {Array.isArray(obj.emotion) && obj.emotion.length > 0 && (
                <div>
                  <span style={{ fontSize: '12px', color: C.textMuted }}>情感目标：</span>
                  {renderTags(obj.emotion)}
                </div>
              )}
            </div>
          )}
          {(Array.isArray(data.key_points) && data.key_points.length > 0) && (
            <div style={cardStyle}>
              <div style={labelStyle}>⭐ 重点难点</div>
              <div style={{ marginBottom: '4px' }}>
                <span style={{ fontSize: '12px', color: C.textMuted }}>重点：</span>
                {renderTags(data.key_points)}
              </div>
              {Array.isArray(data.difficult_points) && data.difficult_points.length > 0 && (
                <div>
                  <span style={{ fontSize: '12px', color: C.textMuted }}>难点：</span>
                  {renderTags(data.difficult_points)}
                </div>
              )}
            </div>
          )}
          {renderStr(data.strategy) && (
            <div style={cardStyle}>
              <div style={labelStyle}>🧭 教学策略</div>
              <div style={valueStyle}>{renderStr(data.strategy)}</div>
            </div>
          )}
          {Array.isArray(acts) && acts.length > 0 && (
            <div style={cardStyle}>
              <div style={labelStyle}>📅 教学活动（{acts.length}个环节）</div>
              {acts.slice(0, 4).map((act, i) => (
                <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
                  <span style={{ ...tagStyle, background: 'rgba(16,185,129,0.08)', color: C.success, flexShrink: 0 }}>
                    {act.duration ? `${act.duration}分钟` : `环节${i+1}`}
                  </span>
                  <span style={{ fontSize: '13px', color: C.text }}>{renderStr(act.name) || `活动${i+1}`}</span>
                </div>
              ))}
              {acts.length > 4 && (
                <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>...还有{acts.length - 4}个环节</div>
              )}
            </div>
          )}
        </div>
      )
    }

    case 'write':
    case 'revise': {
      const cs = data.content_structured as Record<string, unknown> | null
      const hasContent = renderStr(data.content_markdown)
      const revLog = data.revision_log as Array<Record<string, unknown>> | null
      return (
        <div>
          {hasContent && (
            <div style={cardStyle}>
              <div style={labelStyle}>📄 教案内容</div>
              <div style={{ fontSize: '13px', color: C.success, fontWeight: 600, display: 'flex', alignItems: 'center', gap: '6px' }}>
                <span>✅</span>
                <span>完整教案已生成（{Math.round(hasContent.length / 100) * 100}+ 字符）</span>
              </div>
              <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>
                已同步到右侧教案预览，可在预览区查看完整内容
              </div>
            </div>
          )}
          {cs && (
            <div style={cardStyle}>
              <div style={labelStyle}>📑 教案结构</div>
              {Array.isArray(cs.objectives) && (
                <div style={{ fontSize: '13px', color: C.text, marginBottom: '2px' }}>✓ 教学目标已设定</div>
              )}
              {Array.isArray(cs.teaching_process) && cs.teaching_process.length > 0 && (
                <div style={{ fontSize: '13px', color: C.text, marginBottom: '2px' }}>
                  ✓ 教学过程 {cs.teaching_process.length} 个环节
                </div>
              )}
              {renderStr(cs.homework) && (
                <div style={{ fontSize: '13px', color: C.text }}>✓ 作业设计已包含</div>
              )}
            </div>
          )}
          {stageCode === 'revise' && Array.isArray(revLog) && revLog.length > 0 && (
            <div style={cardStyle}>
              <div style={labelStyle}>✏️ 修订记录（{revLog.length}处）</div>
              {revLog.slice(0, 3).map((r, i) => (
                <div key={i} style={{ fontSize: '13px', color: C.text, marginBottom: '4px', display: 'flex', gap: '6px' }}>
                  <span style={{ color: C.primary, flexShrink: 0 }}>#{i+1}</span>
                  <span>{renderStr(r.location) || renderStr(r.change) || `修订${i+1}`}</span>
                </div>
              ))}
              {revLog.length > 3 && (
                <div style={{ fontSize: '12px', color: C.textMuted }}>...还有{revLog.length - 3}处修订</div>
              )}
            </div>
          )}
        </div>
      )
    }

    case 'review': {
      const dims = data.dimensions as Array<Record<string, unknown>> | null
      const imps = data.improvements as Array<Record<string, unknown>> | null
      const score = typeof data.total_score === 'number' ? data.total_score : null
      return (
        <div>
          {score !== null && (
            <div style={{
              ...cardStyle,
              background: score >= 8.5 ? 'rgba(16,185,129,0.06)' : 'rgba(245,158,11,0.06)',
              border: score >= 8.5 ? '1px solid rgba(16,185,129,0.2)' : '1px solid rgba(245,158,11,0.2)',
              display: 'flex', alignItems: 'center', gap: '14px',
            }}>
              <div style={{ fontSize: '32px', fontWeight: 800, flexShrink: 0, color: score >= 8.5 ? C.success : C.accent }}>
                {score.toFixed(1)}
              </div>
              <div>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>AI综合评分</div>
                {renderStr(data.summary) && (
                  <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px', lineHeight: 1.5 }}>
                    {(renderStr(data.summary) || '').slice(0, 80)}
                    {(renderStr(data.summary) || '').length > 80 ? '...' : ''}
                  </div>
                )}
              </div>
            </div>
          )}
          {Array.isArray(dims) && dims.length > 0 && (
            <div style={cardStyle}>
              <div style={labelStyle}>📊 各维度评分</div>
              {dims.map((d, i) => {
                const dimScore = typeof d.score === 'number' ? d.score : null
                return (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
                    <span style={{ fontSize: '12px', color: C.textMuted, width: '60px', flexShrink: 0 }}>
                      {renderStr(d.name) || `维度${i+1}`}
                    </span>
                    <div style={{ flex: 1, height: '6px', background: '#F3F4F6', borderRadius: '3px', overflow: 'hidden' }}>
                      <div style={{
                        height: '100%', borderRadius: '3px',
                        width: `${dimScore ? dimScore * 10 : 0}%`,
                        background: dimScore && dimScore >= 8.5 ? C.success : C.accent,
                        transition: 'width 600ms ease',
                      }} />
                    </div>
                    <span style={{ fontSize: '12px', fontWeight: 600, color: C.text, width: '28px', textAlign: 'right' }}>
                      {dimScore !== null ? dimScore.toFixed(1) : '-'}
                    </span>
                  </div>
                )
              })}
            </div>
          )}
          {Array.isArray(imps) && imps.length > 0 && (
            <div style={cardStyle}>
              <div style={labelStyle}>💡 改进建议（{imps.length}条）</div>
              {imps.slice(0, 3).map((imp, i) => (
                <div key={i} style={{ fontSize: '13px', color: C.text, marginBottom: '4px', display: 'flex', gap: '6px' }}>
                  <span style={{ color: C.accent, flexShrink: 0 }}>▸</span>
                  <span>{renderStr(imp.issue) || `建议${i+1}`}</span>
                </div>
              ))}
              {imps.length > 3 && (
                <div style={{ fontSize: '12px', color: C.textMuted }}>...还有{imps.length - 3}条建议</div>
              )}
            </div>
          )}
        </div>
      )
    }

    default:
      return null
  }
}

// ==================== 阶段完成确认弹窗 ====================

interface StageSummaryModalProps {
  stageCode: string
  stageName: string
  stageOrder: number
  totalStages: number
  nextStageItem: StageProgressItem | null
  structuredOutput: string
  narrative: string
  loading: boolean
  onConfirm: () => void
  onCancel: () => void
  completeness?: StageCompletenessResponse | null
}

export function StageSummaryModal({
  stageCode, stageName, stageOrder, totalStages, nextStageItem,
  structuredOutput, narrative: _narrative, loading, onConfirm, onCancel, completeness,
}: StageSummaryModalProps) {
  const [userNote, setUserNote] = useState('')
  const isLastStage = !nextStageItem || stageOrder >= totalStages
  const hasStructured = structuredOutput && structuredOutput !== '{}'

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { setUserNote('') }, [stageCode])

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
      background: 'rgba(15,23,42,0.6)', backdropFilter: 'blur(6px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      zIndex: 9999,
    }}>
      <div style={{
        width: '560px', maxHeight: '88vh',
        background: C.card, borderRadius: '20px', padding: '28px 32px',
        boxShadow: '0 32px 80px rgba(0,0,0,0.22)',
        animation: 'summaryIn 260ms cubic-bezier(0.34,1.56,0.64,1)',
        display: 'flex', flexDirection: 'column', overflow: 'hidden',
      }}>
        <style>{`
          @keyframes summaryIn {
            from { opacity: 0; transform: translateY(20px) scale(0.96); }
            to   { opacity: 1; transform: translateY(0) scale(1); }
          }
        `}</style>

        {/* 头部 */}
        <div style={{ textAlign: 'center', marginBottom: '20px', flexShrink: 0 }}>
          <div style={{
            width: '52px', height: '52px', borderRadius: '50%',
            margin: '0 auto 12px',
            background: isLastStage
              ? 'linear-gradient(135deg, #10B981, #34D399)'
              : 'linear-gradient(135deg, #4F7BE8, #818CF8)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: '22px',
            boxShadow: isLastStage
              ? '0 8px 24px rgba(16,185,129,0.35)'
              : '0 8px 24px rgba(79,123,232,0.35)',
          }}>
            {isLastStage ? '🎉' : '✅'}
          </div>
          <h2 style={{ fontSize: '18px', fontWeight: 700, color: C.text, margin: '0 0 6px' }}>
            {isLastStage ? '备课流程完成！' : `完成「${stageName}」`}
          </h2>
          <div style={{ fontSize: '13px', color: C.textMuted, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px' }}>
            <span>第 {stageOrder} / {totalStages} 阶段</span>
            {!isLastStage && nextStageItem && (
              <>
                <span style={{ color: C.border }}>·</span>
                <span style={{ color: C.primary, fontWeight: 500 }}>
                  下一步 → {nextStageItem.stage_name}
                </span>
              </>
            )}
          </div>
        </div>

        {/* 内容区 */}
        <div style={{ flex: 1, overflow: 'auto', marginBottom: '16px' }}>
          {loading ? (
            <div style={{
              padding: '32px', borderRadius: '12px',
              background: '#F9FAFB', border: `1px solid ${C.border}`,
              textAlign: 'center', color: C.textMuted, fontSize: '13px',
              display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px',
            }}>
              <div style={{ width: '14px', height: '14px', border: `2px solid ${C.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.7s linear infinite' }} />
              正在加载阶段产出...
              <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
            </div>
          ) : (
            <>
              {hasStructured ? (
                <div style={{ marginBottom: '12px' }}>
                  <div style={{
                    fontSize: '11px', fontWeight: 700, color: C.textMuted,
                    letterSpacing: '0.8px', marginBottom: '10px',
                    display: 'flex', alignItems: 'center', gap: '6px',
                  }}>
                    <span>📊 本阶段产出摘要</span>
                    <span style={{
                      fontSize: '10px', color: C.primary, padding: '1px 6px',
                      background: C.primaryLight, borderRadius: '8px', fontWeight: 500,
                    }}>
                      将注入下一阶段上下文
                    </span>
                  </div>
                  <StageSummaryCards stageCode={stageCode} structuredOutput={structuredOutput} />
                </div>
              ) : (
                !loading && (
                  <div style={{
                    padding: '16px', borderRadius: '10px', marginBottom: '12px',
                    background: 'rgba(245,158,11,0.06)', border: '1px solid rgba(245,158,11,0.15)',
                    fontSize: '13px', color: '#92400E', lineHeight: 1.6,
                    display: 'flex', gap: '8px', alignItems: 'flex-start',
                  }}>
                    <span style={{ flexShrink: 0 }}>💡</span>
                    <span>
                      本阶段尚未生成结构化产出物。AI已了解本阶段的完整对话记录，
                      你可以在下方补充需要特别强调的结论，或直接进入下一阶段。
                    </span>
                  </div>
                )
              )}

              <div>
                <div style={{
                  fontSize: '11px', fontWeight: 700, color: C.textMuted,
                  letterSpacing: '0.8px', marginBottom: '8px',
                }}>
                  ✏️ 补充说明（可选）
                </div>
                <textarea
                  value={userNote}
                  onChange={e => setUserNote(e.target.value)}
                  placeholder={`有什么特别想让AI记住的？\n例如：这节课要照顾到程度差异大的情况，建议多设计分层活动...`}
                  rows={3}
                  style={{
                    width: '100%', padding: '12px 14px', borderRadius: '10px',
                    border: `1px solid ${C.border}`, background: '#FAFBFC',
                    fontSize: '13px', color: C.text, lineHeight: 1.7,
                    resize: 'none', outline: 'none', fontFamily: 'inherit',
                    boxSizing: 'border-box', transition: 'border-color 150ms',
                  }}
                  onFocus={e => { e.target.style.borderColor = C.primary }}
                  onBlur={e  => { e.target.style.borderColor = C.border }}
                />
              </div>
            </>
          )}
        </div>

        {/* 完成度提示 */}
        {completeness && !completeness.is_complete && completeness.missing_hints && completeness.missing_hints.length > 0 && (
          <div style={{
            flexShrink: 0, padding: '12px 16px', borderRadius: '10px', marginBottom: '12px',
            background: '#FFFBEB', border: '1px solid rgba(245,158,11,0.3)',
          }}>
            <div style={{ fontSize: '13px', fontWeight: 600, color: '#92400E', marginBottom: '6px', display: 'flex', alignItems: 'center', gap: '6px' }}>
              <span>⚡</span>
              <span>完成度 {completeness.percentage}%，以下内容可以补充：</span>
            </div>
            {completeness.missing_hints.map((hint: string, i: number) => (
              <div key={i} style={{ fontSize: '12px', color: '#A16207', padding: '2px 0', display: 'flex', alignItems: 'center', gap: '6px' }}>
                <span style={{ color: '#F59E0B', flexShrink: 0 }}>○</span>
                <span>{hint}</span>
              </div>
            ))}
            <div style={{ fontSize: '11px', color: '#B45309', marginTop: '6px', fontStyle: 'italic' }}>
              这不会阻止你继续，只是友好提醒 😊
            </div>
          </div>
        )}

        {/* 操作按钮 */}
        <div style={{ flexShrink: 0, display: 'flex', flexDirection: 'column', gap: '10px' }}>
          {isLastStage ? (
            <>
              <button onClick={onConfirm} style={{
                width: '100%', padding: '14px', borderRadius: '12px', border: 'none',
                background: 'linear-gradient(135deg, #10B981, #34D399)',
                color: '#fff', fontSize: '15px', fontWeight: 600, cursor: 'pointer',
                boxShadow: '0 4px 14px rgba(16,185,129,0.35)',
              }}>
                🎉 完成备课，去保存教案
              </button>
              <button onClick={onCancel} style={{
                width: '100%', padding: '12px', borderRadius: '12px',
                border: `1px solid ${C.border}`, background: 'transparent',
                fontSize: '14px', color: C.textSec, cursor: 'pointer',
              }}>
                继续修改
              </button>
            </>
          ) : (
            <>
              <button onClick={onConfirm} style={{
                width: '100%', padding: '14px', borderRadius: '12px', border: 'none',
                background: 'linear-gradient(135deg, #4F7BE8, #818CF8)',
                color: '#fff', fontSize: '15px', fontWeight: 600, cursor: 'pointer',
                boxShadow: '0 4px 14px rgba(79,123,232,0.35)',
              }}>
                进入{nextStageItem?.stage_name || '下一阶段'} →
              </button>
              <button onClick={onCancel} style={{
                width: '100%', padding: '12px', borderRadius: '12px',
                border: `1px solid ${C.border}`, background: 'transparent',
                fontSize: '14px', color: C.textSec, cursor: 'pointer',
              }}>
                💬 继续完善本阶段
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

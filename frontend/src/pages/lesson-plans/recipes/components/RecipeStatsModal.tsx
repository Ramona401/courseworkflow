/**
 * RecipeStatsModal — 配方效果统计弹窗
 *
 * 迭代6新增：展示配方的使用次数、教案数、平均分、最近使用记录
 */
import { useState, useEffect } from 'react'
import { getRecipeStats, type RecipeStatsResponse } from '@/api/recipes'

const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981', danger: '#EF4444', accent: '#F59E0B',
  text: '#1F2937', textSec: '#6B7280', textMuted: '#9CA3AF',
  card: '#FFFFFF', border: '#F3F4F6', bg: '#FAFBFC',
}

interface Props {
  recipeId: string
  recipeName: string
  onClose: () => void
}

export default function RecipeStatsModal({ recipeId, recipeName, onClose }: Props) {
  const [stats, setStats] = useState<RecipeStatsResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    setLoading(true); setError('')
    getRecipeStats(recipeId)
      .then(resp => setStats(resp))
      .catch(e => setError(e instanceof Error ? e.message : '加载失败'))
      .finally(() => setLoading(false))
  }, [recipeId])

  /** 分数颜色 */
  const scoreColor = (score: number) => {
    if (score >= 8.5) return C.success
    if (score >= 7) return C.accent
    if (score >= 5) return '#F97316'
    return C.danger
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', display: 'flex',
      alignItems: 'center', justifyContent: 'center', zIndex: 10000,
    }} onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{
        background: C.card, borderRadius: '16px', width: '520px', maxHeight: '80vh',
        overflow: 'auto', boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }}>
        {/* 标题 */}
        <div style={{
          padding: '20px 24px 16px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        }}>
          <div>
            <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>📊 配方效果统计</div>
            <div style={{ fontSize: '13px', color: C.textMuted, marginTop: '4px' }}>{recipeName}</div>
          </div>
          <button onClick={onClose} style={{
            border: 'none', background: 'none', cursor: 'pointer', fontSize: '18px', color: C.textMuted, padding: '4px',
          }}>✕</button>
        </div>

        <div style={{ padding: '20px 24px' }}>
          {/* 加载状态 */}
          {loading && (
            <div style={{ textAlign: 'center', padding: '40px', color: C.textMuted, fontSize: '14px' }}>
              加载统计数据...
            </div>
          )}

          {/* 错误状态 */}
          {error && (
            <div style={{ textAlign: 'center', padding: '40px', color: C.danger, fontSize: '14px' }}>
              ⚠️ {error}
            </div>
          )}

          {/* 数据展示 */}
          {stats && !loading && (
            <>
              {/* 核心指标卡片 */}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '12px', marginBottom: '20px' }}>
                <div style={{
                  padding: '16px', borderRadius: '10px', textAlign: 'center',
                  background: C.primaryLight, border: `1px solid rgba(79,123,232,0.15)`,
                }}>
                  <div style={{ fontSize: '24px', fontWeight: 700, color: C.primary }}>{stats.total_usage}</div>
                  <div style={{ fontSize: '12px', color: C.textSec, marginTop: '4px' }}>总使用次数</div>
                </div>
                <div style={{
                  padding: '16px', borderRadius: '10px', textAlign: 'center',
                  background: 'rgba(16,185,129,0.08)', border: '1px solid rgba(16,185,129,0.15)',
                }}>
                  <div style={{ fontSize: '24px', fontWeight: 700, color: C.success }}>{stats.total_plans}</div>
                  <div style={{ fontSize: '12px', color: C.textSec, marginTop: '4px' }}>产出教案数</div>
                </div>
                <div style={{
                  padding: '16px', borderRadius: '10px', textAlign: 'center',
                  background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.15)',
                }}>
                  <div style={{
                    fontSize: '24px', fontWeight: 700,
                    color: stats.avg_score > 0 ? scoreColor(stats.avg_score) : C.textMuted,
                  }}>
                    {stats.avg_score > 0 ? stats.avg_score.toFixed(1) : '—'}
                  </div>
                  <div style={{ fontSize: '12px', color: C.textSec, marginTop: '4px' }}>教案平均分</div>
                </div>
              </div>

              {/* 分数评级说明 */}
              {stats.avg_score > 0 && (
                <div style={{
                  padding: '10px 14px', borderRadius: '8px', marginBottom: '16px',
                  background: stats.avg_score >= 8.5 ? 'rgba(16,185,129,0.06)' : stats.avg_score >= 7 ? 'rgba(245,158,11,0.06)' : 'rgba(239,68,68,0.06)',
                  border: `1px solid ${stats.avg_score >= 8.5 ? 'rgba(16,185,129,0.15)' : stats.avg_score >= 7 ? 'rgba(245,158,11,0.15)' : 'rgba(239,68,68,0.15)'}`,
                  fontSize: '13px', color: scoreColor(stats.avg_score),
                }}>
                  {stats.avg_score >= 8.5 ? '🌟 优秀配方！教案质量稳定在高水平' :
                   stats.avg_score >= 7 ? '👍 良好配方，建议微调组件提升质量' :
                   '💡 有优化空间，建议调整组件和提示词'}
                </div>
              )}

              {/* 最近使用记录 */}
              <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '10px' }}>
                📋 最近使用记录
              </div>
              {stats.recent_usages.length === 0 ? (
                <div style={{ padding: '24px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>
                  暂无使用记录
                </div>
              ) : (
                <div style={{ borderRadius: '8px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
                  <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                    <thead>
                      <tr style={{ background: C.bg }}>
                        <th style={{ padding: '8px 12px', textAlign: 'left', color: C.textSec, fontWeight: 500 }}>使用者</th>
                        <th style={{ padding: '8px 12px', textAlign: 'center', color: C.textSec, fontWeight: 500 }}>教案分数</th>
                        <th style={{ padding: '8px 12px', textAlign: 'right', color: C.textSec, fontWeight: 500 }}>时间</th>
                      </tr>
                    </thead>
                    <tbody>
                      {stats.recent_usages.map((row, i) => (
                        <tr key={i} style={{ borderTop: `1px solid ${C.border}` }}>
                          <td style={{ padding: '8px 12px', color: C.text }}>{row.user_name}</td>
                          <td style={{ padding: '8px 12px', textAlign: 'center' }}>
                            {row.ai_review_score != null ? (
                              <span style={{ fontWeight: 600, color: scoreColor(row.ai_review_score) }}>
                                {row.ai_review_score.toFixed(1)}
                              </span>
                            ) : (
                              <span style={{ color: C.textMuted }}>—</span>
                            )}
                          </td>
                          <td style={{ padding: '8px 12px', textAlign: 'right', color: C.textMuted }}>{row.created_at}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

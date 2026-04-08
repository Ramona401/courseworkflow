/**
 * AITraceDashboardPage — AI调用追踪仪表盘（v80新增，v81增加用户/组织维度）
 *
 * 展示内容：
 *   1. 概览卡片（总调用/总token/总成本/平均延迟/错误率）
 *   2. 按场景聚合表格（每个AI场景的调用量/成本/延迟）
 *   3. 按模型聚合表格
 *   4. 按用户聚合表格（v81新增：每个用户的token消耗和成本）
 *   5. 按组织聚合表格（v81新增：每个学校/区域的整体消耗）
 *   6. 每日趋势（简易柱状图）
 *   7. 最近错误列表
 *
 * 路由：/ai-traces（独立路由，admin专属）
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { getTraceDashboard } from '@/api/ai-traces'
import type { TraceDashboard } from '@/api/ai-traces'

// ==================== 颜色常量 ====================
const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  green: '#10B981',
  greenLight: 'rgba(16,185,129,0.08)',
  orange: '#F59E0B',
  orangeLight: 'rgba(245,158,11,0.08)',
  red: '#EF4444',
  redLight: 'rgba(239,68,68,0.08)',
  purple: '#8B5CF6',
  purpleLight: 'rgba(139,92,246,0.08)',
  cyan: '#06B6D4',
  cyanLight: 'rgba(6,182,212,0.08)',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  bg: '#FAFBFC',
  card: '#FFFFFF',
  border: '#F3F4F6',
  borderDark: '#E5E7EB',
}

// 角色中文映射
const ROLE_NAMES: Record<string, string> = {
  admin: '管理员',
  senior_operator: '高级操作员',
  operator: '操作员',
  viewer: '查看者',
}

// 组织类型中文映射
const ORG_TYPE_NAMES: Record<string, string> = {
  region: '区域',
  school: '学校',
}

// ==================== 主组件 ====================
export default function AITraceDashboardPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [data, setData] = useState<TraceDashboard | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')

  const fetchData = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const params: Record<string, string> = {}
      if (dateFrom) params.date_from = dateFrom
      if (dateTo) params.date_to = dateTo
      const result = await getTraceDashboard(params)
      setData(result)
    } catch (e: any) {
      setError(e?.response?.data?.message || e?.message || '加载失败')
    } finally {
      setLoading(false)
    }
  }, [dateFrom, dateTo])

  useEffect(() => { fetchData() }, [fetchData])

  // 返回来源页面
  const goBack = () => {
    const from = (location.state as any)?.from
    if (from) navigate(from)
    else navigate('/')
  }

  return (
    <div style={{ minHeight: '100vh', background: C.bg }}>
      {/* ── Header ── */}
      <header style={{
        height: '64px',
        background: 'rgba(255,255,255,0.9)',
        backdropFilter: 'blur(20px)',
        borderBottom: `1px solid ${C.border}`,
        display: 'flex',
        alignItems: 'center',
        padding: '0 32px',
        gap: '16px',
      }}>
        <button onClick={goBack} style={{
          background: 'none', border: 'none', cursor: 'pointer',
          fontSize: '14px', color: C.textMuted, padding: '4px 8px',
        }}>← 返回</button>
        <h1 style={{ flex: 1, fontSize: '20px', fontWeight: 600, color: C.textPrimary, margin: 0 }}>
          📊 AI 调用追踪
        </h1>
        {/* 日期筛选 */}
        <div style={{ display: 'flex', gap: '8px', alignItems: 'center', fontSize: '13px' }}>
          <input type="date" value={dateFrom} onChange={e => setDateFrom(e.target.value)}
            style={dateInputStyle} placeholder="开始日期" />
          <span style={{ color: C.textMuted }}>—</span>
          <input type="date" value={dateTo} onChange={e => setDateTo(e.target.value)}
            style={dateInputStyle} placeholder="结束日期" />
          <button onClick={fetchData} style={{
            padding: '6px 16px', borderRadius: '8px', border: `1px solid ${C.borderDark}`,
            background: C.primary, color: '#fff', cursor: 'pointer', fontSize: '13px', fontWeight: 500,
          }}>刷新</button>
        </div>
      </header>

      {/* ── 内容区 ── */}
      <main style={{ padding: '28px 32px', maxWidth: '1400px', margin: '0 auto' }}>
        {loading && <div style={{ textAlign: 'center', padding: '60px', color: C.textMuted }}>加载中...</div>}
        {error && <div style={{ textAlign: 'center', padding: '60px', color: C.red }}>{error}</div>}
        {data && !loading && (
          <>
            {/* ── 1. 概览卡片 ── */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: '16px', marginBottom: '28px' }}>
              <StatCard label="总调用次数" value={data.total_calls.toLocaleString()} icon="🔄" color={C.primary} bg={C.primaryLight} />
              <StatCard label="总Token消耗" value={formatTokens(data.total_tokens)} icon="🎫" color={C.purple} bg={C.purpleLight} />
              <StatCard label="总成本估算" value={'$' + data.total_cost_usd.toFixed(2)} icon="💰" color={C.green} bg={C.greenLight} />
              <StatCard label="平均延迟" value={formatLatency(data.avg_latency_ms)} icon="⏱️" color={C.orange} bg={C.orangeLight} />
              <StatCard label="错误率" value={data.error_rate.toFixed(2) + '%'} icon="⚠️"
                color={data.error_rate > 5 ? C.red : C.green}
                bg={data.error_rate > 5 ? C.redLight : C.greenLight} />
            </div>

            {/* ── 2+3. 按场景 + 按模型 两列 ── */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '28px' }}>
              {/* 按场景 */}
              <div style={sectionStyle}>
                <h3 style={sectionTitleStyle}>📋 按场景统计</h3>
                {data.by_scene.length === 0 ? (
                  <div style={emptyStyle}>暂无数据</div>
                ) : (
                  <table style={tableStyle}>
                    <thead>
                      <tr>
                        <th style={thStyle}>场景</th>
                        <th style={{...thStyle, textAlign: 'right'}}>调用</th>
                        <th style={{...thStyle, textAlign: 'right'}}>成功</th>
                        <th style={{...thStyle, textAlign: 'right'}}>错误</th>
                        <th style={{...thStyle, textAlign: 'right'}}>均延迟</th>
                        <th style={{...thStyle, textAlign: 'right'}}>成本</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.by_scene.map(s => (
                        <tr key={s.scene_code}>
                          <td style={tdStyle}><span style={{ fontWeight: 500 }}>{s.scene_name}</span><br/><span style={{fontSize:'11px',color:C.textMuted}}>{s.scene_code}</span></td>
                          <td style={{...tdStyle, textAlign: 'right'}}>{s.call_count}</td>
                          <td style={{...tdStyle, textAlign: 'right', color: C.green}}>{s.success_count}</td>
                          <td style={{...tdStyle, textAlign: 'right', color: s.error_count > 0 ? C.red : C.textMuted}}>{s.error_count}</td>
                          <td style={{...tdStyle, textAlign: 'right'}}>{formatLatency(s.avg_latency_ms)}</td>
                          <td style={{...tdStyle, textAlign: 'right', fontWeight: 600}}>${s.estimated_cost_usd.toFixed(3)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>

              {/* 按模型 */}
              <div style={sectionStyle}>
                <h3 style={sectionTitleStyle}>🤖 按模型统计</h3>
                {data.by_model.length === 0 ? (
                  <div style={emptyStyle}>暂无数据</div>
                ) : (
                  <table style={tableStyle}>
                    <thead>
                      <tr>
                        <th style={thStyle}>模型</th>
                        <th style={{...thStyle, textAlign: 'right'}}>调用</th>
                        <th style={{...thStyle, textAlign: 'right'}}>成功</th>
                        <th style={{...thStyle, textAlign: 'right'}}>错误</th>
                        <th style={{...thStyle, textAlign: 'right'}}>均延迟</th>
                        <th style={{...thStyle, textAlign: 'right'}}>成本</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.by_model.map(m => (
                        <tr key={m.model_used}>
                          <td style={tdStyle}><span style={{ fontWeight: 500, fontSize: '12px' }}>{m.model_used.replace('anthropic/', '')}</span></td>
                          <td style={{...tdStyle, textAlign: 'right'}}>{m.call_count}</td>
                          <td style={{...tdStyle, textAlign: 'right', color: C.green}}>{m.success_count}</td>
                          <td style={{...tdStyle, textAlign: 'right', color: m.error_count > 0 ? C.red : C.textMuted}}>{m.error_count}</td>
                          <td style={{...tdStyle, textAlign: 'right'}}>{formatLatency(m.avg_latency_ms)}</td>
                          <td style={{...tdStyle, textAlign: 'right', fontWeight: 600}}>${m.estimated_cost_usd.toFixed(3)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </div>

            {/* ── 4+5. 按用户 + 按组织 两列（v81新增）── */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '28px' }}>
              {/* 按用户统计 */}
              <div style={sectionStyle}>
                <h3 style={sectionTitleStyle}>👤 按用户统计</h3>
                {data.by_user.length === 0 ? (
                  <div style={emptyStyle}>暂无数据（需要trace记录包含user_id）</div>
                ) : (
                  <div style={{ overflowX: 'auto' }}>
                    <table style={tableStyle}>
                      <thead>
                        <tr>
                          <th style={thStyle}>用户</th>
                          <th style={{...thStyle, textAlign: 'right'}}>调用</th>
                          <th style={{...thStyle, textAlign: 'right'}}>Token</th>
                          <th style={{...thStyle, textAlign: 'right'}}>均延迟</th>
                          <th style={{...thStyle, textAlign: 'right'}}>成本</th>
                        </tr>
                      </thead>
                      <tbody>
                        {data.by_user.map(u => (
                          <tr key={u.user_id}>
                            <td style={tdStyle}>
                              <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                                <span style={{ fontWeight: 500 }}>{u.display_name || u.username}</span>
                                <span style={{ fontSize: '11px', color: C.textMuted }}>
                                  {u.username} · {ROLE_NAMES[u.role] || u.role}
                                </span>
                              </div>
                            </td>
                            <td style={{...tdStyle, textAlign: 'right'}}>
                              <span>{u.call_count}</span>
                              {u.error_count > 0 && (
                                <span style={{ fontSize: '11px', color: C.red, marginLeft: '4px' }}>({u.error_count}错)</span>
                              )}
                            </td>
                            <td style={{...tdStyle, textAlign: 'right'}}>
                              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '1px' }}>
                                <span style={{ fontWeight: 500 }}>{formatTokens(u.total_tokens)}</span>
                                <span style={{ fontSize: '10px', color: C.textMuted }}>
                                  入{formatTokens(u.total_prompt_tokens)} / 出{formatTokens(u.total_completion_tokens)}
                                </span>
                              </div>
                            </td>
                            <td style={{...tdStyle, textAlign: 'right'}}>{formatLatency(u.avg_latency_ms)}</td>
                            <td style={{...tdStyle, textAlign: 'right', fontWeight: 600, color: C.green}}>${u.estimated_cost_usd.toFixed(3)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>

              {/* 按组织统计 */}
              <div style={sectionStyle}>
                <h3 style={sectionTitleStyle}>🏫 按组织统计</h3>
                {data.by_org.length === 0 ? (
                  <div style={emptyStyle}>暂无数据（需要用户加入教研组且有trace记录）</div>
                ) : (
                  <div style={{ overflowX: 'auto' }}>
                    <table style={tableStyle}>
                      <thead>
                        <tr>
                          <th style={thStyle}>组织</th>
                          <th style={{...thStyle, textAlign: 'right'}}>活跃人数</th>
                          <th style={{...thStyle, textAlign: 'right'}}>调用</th>
                          <th style={{...thStyle, textAlign: 'right'}}>Token</th>
                          <th style={{...thStyle, textAlign: 'right'}}>成本</th>
                        </tr>
                      </thead>
                      <tbody>
                        {data.by_org.map(o => (
                          <tr key={o.org_id}>
                            <td style={tdStyle}>
                              <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                                <span style={{ fontWeight: 500 }}>{o.org_name}</span>
                                <span style={{
                                  fontSize: '11px', color: C.textMuted,
                                  display: 'inline-flex', alignItems: 'center', gap: '4px',
                                }}>
                                  <span style={{
                                    padding: '1px 6px', borderRadius: '3px', fontSize: '10px',
                                    background: o.org_type === 'school' ? C.primaryLight : C.purpleLight,
                                    color: o.org_type === 'school' ? C.primary : C.purple,
                                    fontWeight: 500,
                                  }}>{ORG_TYPE_NAMES[o.org_type] || o.org_type}</span>
                                </span>
                              </div>
                            </td>
                            <td style={{...tdStyle, textAlign: 'right'}}>{o.member_count}人</td>
                            <td style={{...tdStyle, textAlign: 'right'}}>
                              <span>{o.call_count}</span>
                              {o.error_count > 0 && (
                                <span style={{ fontSize: '11px', color: C.red, marginLeft: '4px' }}>({o.error_count}错)</span>
                              )}
                            </td>
                            <td style={{...tdStyle, textAlign: 'right'}}>
                              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '1px' }}>
                                <span style={{ fontWeight: 500 }}>{formatTokens(o.total_tokens)}</span>
                                <span style={{ fontSize: '10px', color: C.textMuted }}>
                                  入{formatTokens(o.total_prompt_tokens)} / 出{formatTokens(o.total_completion_tokens)}
                                </span>
                              </div>
                            </td>
                            <td style={{...tdStyle, textAlign: 'right', fontWeight: 600, color: C.green}}>${o.estimated_cost_usd.toFixed(3)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>

            {/* ── 6. 每日趋势 ── */}
            <div style={{ ...sectionStyle, marginBottom: '28px' }}>
              <h3 style={sectionTitleStyle}>📈 每日趋势（最近30天）</h3>
              {data.daily_trend.length === 0 ? (
                <div style={emptyStyle}>暂无数据</div>
              ) : (
                <div style={{ overflowX: 'auto' }}>
                  <div style={{ display: 'flex', gap: '2px', alignItems: 'flex-end', height: '120px', padding: '8px 0' }}>
                    {[...data.daily_trend].reverse().map(d => {
                      const maxCalls = Math.max(...data.daily_trend.map(t => t.call_count), 1)
                      const h = Math.max(4, (d.call_count / maxCalls) * 100)
                      return (
                        <div key={d.date} title={`${d.date}\n调用: ${d.call_count}\n错误: ${d.error_count}\n成本: $${d.estimated_cost_usd.toFixed(3)}`}
                          style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', flex: 1, minWidth: '16px' }}>
                          <div style={{
                            width: '100%', maxWidth: '24px', height: `${h}px`, borderRadius: '3px 3px 0 0',
                            background: d.error_count > 0 ? `linear-gradient(${C.red}, ${C.orange})` : C.primary,
                            opacity: 0.8,
                          }} />
                          <div style={{ fontSize: '9px', color: C.textMuted, marginTop: '4px', transform: 'rotate(-45deg)', whiteSpace: 'nowrap' }}>
                            {d.date.slice(5)}
                          </div>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}
            </div>

            {/* ── 7. 最近错误 ── */}
            <div style={sectionStyle}>
              <h3 style={sectionTitleStyle}>🚨 最近错误（最多20条）</h3>
              {data.recent_errors.length === 0 ? (
                <div style={emptyStyle}>无错误记录 ✅</div>
              ) : (
                <table style={tableStyle}>
                  <thead>
                    <tr>
                      <th style={thStyle}>时间</th>
                      <th style={thStyle}>场景</th>
                      <th style={thStyle}>模型</th>
                      <th style={thStyle}>状态</th>
                      <th style={thStyle}>延迟</th>
                      <th style={{...thStyle, minWidth: '200px'}}>错误信息</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.recent_errors.map(e => (
                      <tr key={e.id}>
                        <td style={{...tdStyle, fontSize: '12px', whiteSpace: 'nowrap'}}>{new Date(e.created_at).toLocaleString('zh-CN')}</td>
                        <td style={tdStyle}>{e.scene_code}</td>
                        <td style={{...tdStyle, fontSize: '12px'}}>{e.model_used.replace('anthropic/', '')}</td>
                        <td style={tdStyle}>
                          <span style={{
                            padding: '2px 8px', borderRadius: '4px', fontSize: '11px', fontWeight: 600,
                            background: C.redLight, color: C.red,
                          }}>{e.status}</span>
                        </td>
                        <td style={{...tdStyle, textAlign: 'right'}}>{formatLatency(e.latency_ms)}</td>
                        <td style={{...tdStyle, fontSize: '12px', color: C.red, maxWidth: '300px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>
                          {e.error_message.slice(0, 120)}{e.error_message.length > 120 ? '...' : ''}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </>
        )}
      </main>
    </div>
  )
}

// ==================== 子组件 ====================
function StatCard({ label, value, icon, color, bg }: {
  label: string; value: string; icon: string; color: string; bg: string
}) {
  return (
    <div style={{
      background: C.card, borderRadius: '12px', padding: '20px',
      border: `1px solid ${C.border}`, display: 'flex', flexDirection: 'column', gap: '8px',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ fontSize: '13px', color: C.textSecondary }}>{label}</span>
        <span style={{
          width: '32px', height: '32px', borderRadius: '8px', background: bg,
          display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '16px',
        }}>{icon}</span>
      </div>
      <div style={{ fontSize: '24px', fontWeight: 700, color, letterSpacing: '-0.5px' }}>{value}</div>
    </div>
  )
}

// ==================== 格式化工具 ====================
function formatTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return n.toString()
}

function formatLatency(ms: number): string {
  if (ms >= 60000) return (ms / 60000).toFixed(1) + 'min'
  if (ms >= 1000) return (ms / 1000).toFixed(1) + 's'
  return ms + 'ms'
}

// ==================== 样式常量 ====================
const dateInputStyle: React.CSSProperties = {
  padding: '5px 10px', borderRadius: '6px', border: `1px solid ${C.borderDark}`,
  fontSize: '13px', color: C.textPrimary, outline: 'none',
}

const sectionStyle: React.CSSProperties = {
  background: C.card, borderRadius: '12px', padding: '20px',
  border: `1px solid ${C.border}`,
}

const sectionTitleStyle: React.CSSProperties = {
  fontSize: '15px', fontWeight: 600, color: C.textPrimary, margin: '0 0 16px 0',
}

const emptyStyle: React.CSSProperties = {
  textAlign: 'center', padding: '32px', color: C.textMuted, fontSize: '14px',
}

const tableStyle: React.CSSProperties = {
  width: '100%', borderCollapse: 'collapse', fontSize: '13px',
}

const thStyle: React.CSSProperties = {
  padding: '8px 12px', textAlign: 'left', borderBottom: `2px solid ${C.border}`,
  fontSize: '12px', fontWeight: 600, color: C.textSecondary, whiteSpace: 'nowrap',
}

const tdStyle: React.CSSProperties = {
  padding: '10px 12px', borderBottom: `1px solid ${C.border}`, color: C.textPrimary,
}

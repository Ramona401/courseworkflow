/**
 * 仪表盘页面 - Apple风格
 * P4.5-D重构：接入真实数据（/dashboard/stats API）
 * 统计卡片 + 最近Pipeline快捷列表 + 系统信息
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import { getDashboardStats, type DashboardStats } from '@/api/dashboard'
import { getPipelines, type PipelineListItem } from '@/api/pipelines'

// ==================== 统计卡片组件 ====================

/** 单个统计卡片 */
function StatCard({ label, value, icon, color, sub }: {
  label: string; value: string | number; icon: string; color: string; sub?: string
}) {
  return (
    <div style={{
      background: '#fff', borderRadius: 16, padding: '22px 24px',
      border: '1px solid rgba(0,0,0,0.04)', boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
      transition: 'all 0.2s ease', cursor: 'default', flex: 1, minWidth: 160,
    }}
    onMouseEnter={e => {
      (e.currentTarget as HTMLElement).style.transform = 'translateY(-2px)';
      (e.currentTarget as HTMLElement).style.boxShadow = '0 8px 25px rgba(0,0,0,0.08)'
    }}
    onMouseLeave={e => {
      (e.currentTarget as HTMLElement).style.transform = 'translateY(0)';
      (e.currentTarget as HTMLElement).style.boxShadow = '0 1px 3px rgba(0,0,0,0.04)'
    }}>
      <div style={{
        width: 36, height: 36, background: color, borderRadius: 10,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        fontSize: 18, marginBottom: 14,
      }}>{icon}</div>
      <div style={{ fontSize: 12, color: '#86868b', fontWeight: 500, marginBottom: 3 }}>{label}</div>
      <div style={{ fontSize: 28, fontWeight: 700, color: '#1d1d1f', letterSpacing: '-0.5px' }}>{value}</div>
      {sub && <div style={{ fontSize: 11, color: '#86868b', marginTop: 2 }}>{sub}</div>}
    </div>
  )
}

// ==================== 最近Pipeline列表组件 ====================

/** 状态颜色映射 */
const statusColorMap: Record<string, string> = {
  pending: '#8e8e93', running: '#007aff', review_queue: '#ff9500',
  finalized: '#34c759', needs_human: '#cc9900', failed: '#ff3b30', cancelled: '#aeaeb2',
}

/** 最近Pipeline单行 */
function RecentPipelineRow({ p, onClick }: { p: PipelineListItem; onClick: () => void }) {
  const color = statusColorMap[p.status] || '#8e8e93'
  return (
    <div onClick={onClick} style={{
      display: 'flex', alignItems: 'center', gap: 12, padding: '10px 16px',
      cursor: 'pointer', borderRadius: 10, transition: 'background 0.15s ease',
    }}
    onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.04)' }}
    onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}>
      {/* 状态圆点 */}
      <div style={{ width: 8, height: 8, borderRadius: '50%', background: color, flexShrink: 0 }} />
      {/* 课程编号+名称 */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e' }}>{p.course_code}</div>
        <div style={{ fontSize: 11, color: '#86868b', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {p.course_name || p.course_code}
        </div>
      </div>
      {/* 状态+分数 */}
      <div style={{ textAlign: 'right', flexShrink: 0 }}>
        <div style={{ fontSize: 11, color, fontWeight: 500 }}>{p.status_name}</div>
        {p.meta_score !== null && (
          <div style={{ fontSize: 12, fontWeight: 600, color: p.meta_score >= 9.0 ? '#34c759' : '#ff9500' }}>
            {p.meta_score.toFixed(1)}
          </div>
        )}
      </div>
      {/* 箭头 */}
      <div style={{ color: '#c7c7cc', fontSize: 14, flexShrink: 0 }}>›</div>
    </div>
  )
}

// ==================== 格式化工具 ====================

/** 格式化token数量（千/万单位） */
function formatTokens(n: number): string {
  if (n >= 10000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 10000) return (n / 1000).toFixed(0) + 'K'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return String(n)
}

// ==================== 主页面组件 ====================

export default function DashboardPage() {
  const navigate = useNavigate()
  const { user } = useAuth()

  // 统计数据
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [loading, setLoading] = useState(true)

  // 最近Pipeline列表（取前8条）
  const [recentPipelines, setRecentPipelines] = useState<PipelineListItem[]>([])

  // 加载数据
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      // 并行加载统计数据和Pipeline列表
      const [statsData, plData] = await Promise.all([
        getDashboardStats(),
        getPipelines(),
      ])
      setStats(statsData)
      // 取最近8条Pipeline
      setRecentPipelines((plData.pipelines || []).slice(0, 8))
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) {
      console.error('加载仪表盘数据失败:', e)
    }
    setLoading(false)
  }, [])

   
  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { loadData() }, [loadData])

  return (
    <div>
      {/* 欢迎区域 */}
      <div style={{ marginBottom: 24 }}>
        <h1 style={{
          fontSize: 28, fontWeight: 700, color: '#1d1d1f',
          margin: '0 0 6px 0', letterSpacing: '-0.5px',
        }}>
          欢迎回来，{user?.display_name || '用户'}
        </h1>
        <p style={{ fontSize: 15, color: '#86868b', margin: 0 }}>
          TE-DNA 2.0 课程质量评估与自动优化平台
        </p>
      </div>

      {/* 统计卡片行 */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: 40, color: '#86868b' }}>加载中...</div>
      ) : stats ? (
        <>
          {/* 第一行：课程 + Pipeline状态 */}
          <div style={{ display: 'flex', gap: 12, marginBottom: 12, flexWrap: 'wrap' }}>
            <StatCard label="课程总数" value={stats.total_courses} icon="📚"
              color="linear-gradient(135deg, #007aff, #5ac8fa)"
              sub={stats.courses_with_index + ' 有索引'} />
            <StatCard label="Pipeline总数" value={stats.total_pipelines} icon="⚡"
              color="linear-gradient(135deg, #5856d6, #af52de)" />
            <StatCard label="运行中" value={stats.running_pipelines} icon="🔄"
              color="linear-gradient(135deg, #007aff, #64d2ff)" />
            <StatCard label="待审核" value={stats.review_queue} icon="📋"
              color="linear-gradient(135deg, #ff9500, #ffcc00)" />
          </div>

          {/* 第二行：达标 + 已定稿 + 失败 + AI消耗 */}
          <div style={{ display: 'flex', gap: 12, marginBottom: 24, flexWrap: 'wrap' }}>
            <StatCard label="评估达标(≥9.0)" value={stats.passed_count} icon="✅"
              color="linear-gradient(135deg, #34c759, #30d158)" />
            <StatCard label="已定稿" value={stats.finalized} icon="🏁"
              color="linear-gradient(135deg, #32ade6, #5ac8fa)" />
            <StatCard label="失败" value={stats.failed} icon="❌"
              color="linear-gradient(135deg, #ff3b30, #ff6961)" />
            <StatCard label="AI Token消耗" value={formatTokens(stats.total_tokens_used)} icon="🤖"
              color="linear-gradient(135deg, #af52de, #5856d6)" />
          </div>
        </>
      ) : (
        <div style={{ textAlign: 'center', padding: 40, color: '#86868b' }}>暂无数据</div>
      )}

      {/* 最近Pipeline列表 */}
      <div style={{
        background: '#fff', borderRadius: 16, padding: '20px 8px',
        border: '1px solid rgba(0,0,0,0.04)', boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
        marginBottom: 20,
      }}>
        <div style={{
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          padding: '0 16px', marginBottom: 8,
        }}>
          <h3 style={{ fontSize: 15, fontWeight: 600, color: '#1d1d1f', margin: 0 }}>
            最近 Pipeline
          </h3>
          <span onClick={() => navigate('/workflow/pipelines')} style={{
            fontSize: 13, color: '#007aff', cursor: 'pointer', fontWeight: 500,
          }}>
            查看全部 →
          </span>
        </div>

        {recentPipelines.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '24px 0', color: '#86868b', fontSize: 13 }}>
            {loading ? '加载中...' : '暂无Pipeline'}
          </div>
        ) : (
          <div>
            {recentPipelines.map(p => (
              <RecentPipelineRow key={p.id} p={p} onClick={() => navigate('/workflow/pipelines/' + p.id)} />
            ))}
          </div>
        )}
      </div>

      {/* 系统信息卡片 */}
      <div style={{
        background: '#fff', borderRadius: 16, padding: 24,
        border: '1px solid rgba(0,0,0,0.04)', boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
      }}>
        <h3 style={{ fontSize: 15, fontWeight: 600, color: '#1d1d1f', margin: '0 0 8px 0' }}>
          系统信息
        </h3>
        <p style={{ fontSize: 13, color: '#86868b', margin: 0, lineHeight: 1.8 }}>
          版本 0.37.0 · Go + React + PostgreSQL · 8步Pipeline自动评估引擎 + AI备课工坊 + 教案评审系统
        </p>
      </div>
    </div>
  )
}

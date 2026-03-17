/**
 * 仪表盘页面 - Apple 风格
 * - 欢迎卡片 + 统计卡片
 * - Phase 7 接入真实数据，当前为占位
 */
import { useAuth } from '@/store/auth'

export default function DashboardPage() {
  const { user } = useAuth()

  const stats = [
    { label: '课程总数', value: '--', icon: '📚', bg: 'linear-gradient(135deg, #007aff, #5ac8fa)' },
    { label: '运行中 Pipeline', value: '--', icon: '⚡', bg: 'linear-gradient(135deg, #34c759, #30d158)' },
    { label: '待审核', value: '--', icon: '📋', bg: 'linear-gradient(135deg, #ff9500, #ffcc00)' },
    { label: 'AI 调用次数', value: '--', icon: '🤖', bg: 'linear-gradient(135deg, #af52de, #5856d6)' },
  ]

  return (
    <div>
      {/* 欢迎区域 */}
      <div style={{ marginBottom: '28px' }}>
        <h1 style={{
          fontSize: '28px',
          fontWeight: 700,
          color: '#1d1d1f',
          margin: '0 0 6px 0',
          letterSpacing: '-0.5px',
        }}>
          欢迎回来，{user?.display_name || '用户'}
        </h1>
        <p style={{
          fontSize: '15px',
          color: '#86868b',
          margin: 0,
        }}>这里是 TE-DNA 2.0 课程工作流平台的概览</p>
      </div>

      {/* 统计卡片 */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))',
        gap: '16px',
        marginBottom: '28px',
      }}>
        {stats.map((card) => (
          <div key={card.label} style={{
            background: '#fff',
            borderRadius: '16px',
            padding: '24px',
            border: '1px solid rgba(0,0,0,0.04)',
            boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
            transition: 'all 0.2s ease',
            cursor: 'default',
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLElement).style.transform = 'translateY(-2px)'
            ;(e.currentTarget as HTMLElement).style.boxShadow = '0 8px 25px rgba(0,0,0,0.08)'
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLElement).style.transform = 'translateY(0)'
            ;(e.currentTarget as HTMLElement).style.boxShadow = '0 1px 3px rgba(0,0,0,0.04)'
          }}
          >
            <div style={{
              width: '40px',
              height: '40px',
              background: card.bg,
              borderRadius: '12px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '20px',
              marginBottom: '16px',
            }}>{card.icon}</div>
            <div style={{
              fontSize: '13px',
              color: '#86868b',
              marginBottom: '4px',
              fontWeight: 500,
            }}>{card.label}</div>
            <div style={{
              fontSize: '32px',
              fontWeight: 700,
              color: '#1d1d1f',
              letterSpacing: '-1px',
            }}>{card.value}</div>
          </div>
        ))}
      </div>

      {/* 开发进度卡片 */}
      <div style={{
        background: '#fff',
        borderRadius: '16px',
        padding: '24px',
        border: '1px solid rgba(0,0,0,0.04)',
        boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
      }}>
        <h3 style={{
          fontSize: '15px',
          fontWeight: 600,
          color: '#1d1d1f',
          margin: '0 0 8px 0',
        }}>开发进度</h3>
        <p style={{ fontSize: '14px', color: '#86868b', margin: 0, lineHeight: 1.6 }}>
          Phase 1 进行中 — 用户认证、角色权限、基础页面框架已完成。
          仪表盘数据将在 Phase 7 接入真实数据源。
        </p>
      </div>
    </div>
  )
}

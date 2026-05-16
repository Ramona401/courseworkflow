/**
 * 入口选择页面 — PortalPage
 * 登录后首先进入此页面，根据用户权限显示可用入口卡片：
 * - 📝 备课工坊：教案系统（所有active用户可见）
 * - 🖥️ 课件审核：课件审核系统（admin/senior_operator/operator可见）
 * - 🎨 课件工坊：AI课件生成系统（所有active用户可见）
 *
 * 设计遵循PRD v1.0 §8.7：
 * - 简洁的两栏卡片布局
 * - 极浅灰白背景
 * - 白色大卡片，hover轻微上浮+阴影
 */
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'

/* ==================== 入口卡片数据 ==================== */
interface PortalEntry {
  key: string
  icon: string
  title: string
  description: string
  path: string
  roles: string[] | 'all'  // 'all' 表示所有active用户可见
}

const entries: PortalEntry[] = [
  {
    key: 'lesson-plans',
    icon: '📝',
    title: '备课工坊',
    description: 'AI辅助教案开发 · 教案库 · 教研协作',
    path: '/lesson-plans',
    roles: 'all',
  },
  {
    key: 'courseware',
    icon: '🎨',
    title: '课件工坊',
    description: 'AI辅助课件生成 · 模板组件 · 多媒体',
    path: '/courseware',
    roles: 'all',
  },
  {
    key: 'workflow',
    icon: '🖥️',
    title: '课件审核',
    description: '课件质量评估 · 审核 · 定稿 · 验收',
    path: '/workflow',
    roles: ['admin', 'senior_operator', 'operator'],
  },
]

/* ==================== 样式常量 ==================== */
const styles = {
  /* 页面容器 */
  container: {
    minHeight: '100vh',
    background: '#FAFBFC',
    display: 'flex',
    flexDirection: 'column' as const,
    alignItems: 'center',
    justifyContent: 'center',
    padding: '40px 24px',
  },
  /* 欢迎语 */
  greeting: {
    fontSize: '24px',
    fontWeight: 600,
    color: '#1F2937',
    marginBottom: '8px',
  },
  subtitle: {
    fontSize: '15px',
    color: '#6B7280',
    marginBottom: '48px',
  },
  /* 卡片网格 */
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
    gap: '24px',
    maxWidth: '1020px',
    width: '100%',
  },
  /* 单个入口卡片 */
  card: {
    background: '#FFFFFF',
    borderRadius: '12px',
    padding: '32px 28px',
    cursor: 'pointer',
    transition: 'all 250ms ease',
    boxShadow: '0 1px 3px rgba(0,0,0,0.08)',
    border: '1px solid #F3F4F6',
    minHeight: '160px',
    display: 'flex',
    flexDirection: 'column' as const,
    justifyContent: 'center',
  },
  cardHover: {
    transform: 'translateY(-3px)',
    boxShadow: '0 8px 24px rgba(0,0,0,0.1)',
    borderColor: '#E5E7EB',
  },
  icon: {
    fontSize: '36px',
    marginBottom: '16px',
  },
  cardTitle: {
    fontSize: '20px',
    fontWeight: 600,
    color: '#1F2937',
    marginBottom: '8px',
  },
  cardDesc: {
    fontSize: '14px',
    color: '#6B7280',
    lineHeight: 1.5,
  },
  /* 页脚 */
  footer: {
    marginTop: '60px',
    fontSize: '13px',
    color: '#9CA3AF',
  },
} as const

/* ==================== 组件 ==================== */
export default function PortalPage() {
  const navigate = useNavigate()
  const { user } = useAuth()

  // 如果没有用户信息，不渲染（AuthGuard会处理跳转）
  if (!user) return null

  // 根据角色过滤可见入口
  const visibleEntries = entries.filter(entry => {
    if (entry.roles === 'all') return true
    return entry.roles.includes(user.role)
  })

  return (
    <div style={styles.container}>
      {/* 北大实验室 Logo */}
      <div style={{ marginBottom: '32px' }}>
        <img
          src="/pkuailab.png"
          alt="北京大学人工智能应用与创新实验室"
          style={{ height: '44px', objectFit: 'contain' }}
        />
      </div>

      {/* 欢迎语 */}
      <div style={styles.greeting}>欢迎回来，{user.display_name}</div>
      <div style={styles.subtitle}>请选择要进入的工作区</div>

      {/* 入口卡片网格 */}
      <div style={styles.grid}>
        {visibleEntries.map(entry => (
          <CardItem
            key={entry.key}
            entry={entry}
            onClick={() => navigate(entry.path)}
          />
        ))}
      </div>

      {/* 页脚 */}
      <div style={styles.footer}>TE-DNA 2.0 · 北京大学人工智能应用与创新实验室</div>
    </div>
  )
}

/* ==================== 卡片子组件（含hover效果） ==================== */
import { useState } from 'react'

function CardItem({ entry, onClick }: { entry: PortalEntry; onClick: () => void }) {
  const [hovered, setHovered] = useState(false)

  const cardStyle = {
    ...styles.card,
    ...(hovered ? styles.cardHover : {}),
  }

  return (
    <div
      style={cardStyle}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onClick={onClick}
    >
      <div style={styles.icon}>{entry.icon}</div>
      <div style={styles.cardTitle}>{entry.title}</div>
      <div style={styles.cardDesc}>{entry.description}</div>
    </div>
  )
}

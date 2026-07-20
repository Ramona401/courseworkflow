/**
 * EducationDomainGuard — 教育域异常状态统一教学业务守卫
 *
 * 使用范围：
 *   - 登录后的门户首页；
 *   - 教案系统及独立教案审核工作台；
 *   - 课件系统及独立课件审核工作台；
 *   - Pipeline/课件审核系统；
 *   - 教案和课件回收站。
 *
 * 不阻断：
 *   - /admin 用户与组织管理；
 *   - /account 个人中心；
 *   - /tokens 积分管理；
 *   - 退出登录。
 *
 * 安全边界：
 *   - 本组件是前端统一体验和请求阻断边界；
 *   - 真正的数据授权仍由后端负责；
 *   - useSubjects等底层Hook仍会读取ready进行第二层fail-closed保护；
 *   - 不允许各业务页面自行散落education_domain_ready判断。
 */

import type { ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { logout as apiLogout } from '@/api/auth'
import { useEducationProfile } from '@/hooks/useEducationProfile'
import { useAuth } from '@/store/auth'

interface EducationDomainGuardProps {
  children: ReactNode
}

const styles = {
  page: {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background:
      'radial-gradient(circle at top, rgba(79,123,232,0.10), transparent 42%), #F8FAFC',
    padding: '32px 20px',
  },
  card: {
    width: '100%',
    maxWidth: '560px',
    background: '#FFFFFF',
    border: '1px solid #E5E7EB',
    borderRadius: '20px',
    boxShadow: '0 20px 60px rgba(15,23,42,0.10)',
    padding: '40px 36px',
    textAlign: 'center' as const,
  },
  iconWrap: {
    width: '72px',
    height: '72px',
    borderRadius: '22px',
    margin: '0 auto 22px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'rgba(245,158,11,0.12)',
    fontSize: '34px',
  },
  title: {
    margin: '0 0 12px',
    color: '#111827',
    fontSize: '24px',
    fontWeight: 700,
    letterSpacing: '-0.4px',
  },
  message: {
    margin: '0 auto',
    color: '#B45309',
    fontSize: '16px',
    fontWeight: 600,
    lineHeight: 1.7,
  },
  detail: {
    margin: '18px auto 0',
    maxWidth: '440px',
    color: '#6B7280',
    fontSize: '14px',
    lineHeight: 1.8,
  },
  userInfo: {
    margin: '22px auto 0',
    padding: '12px 16px',
    maxWidth: '420px',
    borderRadius: '12px',
    background: '#F9FAFB',
    color: '#6B7280',
    fontSize: '13px',
    lineHeight: 1.6,
  },
  actions: {
    marginTop: '28px',
    display: 'flex',
    flexWrap: 'wrap' as const,
    justifyContent: 'center',
    gap: '10px',
  },
  primaryButton: {
    border: 'none',
    borderRadius: '10px',
    padding: '11px 20px',
    background: 'linear-gradient(135deg, #4F7BE8, #6366F1)',
    color: '#FFFFFF',
    cursor: 'pointer',
    fontSize: '14px',
    fontWeight: 600,
  },
  secondaryButton: {
    border: '1px solid #D1D5DB',
    borderRadius: '10px',
    padding: '10px 18px',
    background: '#FFFFFF',
    color: '#374151',
    cursor: 'pointer',
    fontSize: '14px',
    fontWeight: 500,
  },
  dangerButton: {
    border: '1px solid rgba(239,68,68,0.25)',
    borderRadius: '10px',
    padding: '10px 18px',
    background: 'rgba(239,68,68,0.04)',
    color: '#DC2626',
    cursor: 'pointer',
    fontSize: '14px',
    fontWeight: 500,
  },
} as const

export default function EducationDomainGuard({
  children,
}: EducationDomainGuardProps) {
  const navigate = useNavigate()
  const { user, logout } = useAuth()
  const {
    ready,
    error,
  } = useEducationProfile()

  // 本组件始终位于AuthGuard内层。
  // 防御性处理：用户尚未恢复时不渲染教学内容。
  if (!user) return null

  if (ready) {
    return <>{children}</>
  }

  const canOpenAdmin =
    user.role === 'admin' ||
    user.role === 'senior_operator' ||
    user.role === 'region_admin'

  const handleLogout = async () => {
    try {
      await apiLogout()
    } catch {
      // 服务端登出记录失败不影响本地清理。
    }

    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div style={styles.page}>
      <section
        style={styles.card}
        role="alert"
        aria-live="assertive"
      >
        <div style={styles.iconWrap}>⚠️</div>

        <h1 style={styles.title}>
          教育域尚未正确配置
        </h1>

        <p style={styles.message}>
          {error}
        </p>

        <p style={styles.detail}>
          您的身份认证已经成功，但为防止错误加载默认K12课程或跨教育域数据，
          备课、教案、课件、审核、AI助手和配方等教学业务已暂时停用。
          管理员修复任命教育域后，刷新页面即可恢复。
        </p>

        <div style={styles.userInfo}>
          当前账号：{user.display_name || user.username}
          <br />
          登录身份：{user.role}
        </div>

        <div style={styles.actions}>
          <button
            type="button"
            style={styles.primaryButton}
            onClick={() => window.location.reload()}
          >
            重新检测
          </button>

          {canOpenAdmin && (
            <button
              type="button"
              style={styles.secondaryButton}
              onClick={() => navigate('/admin')}
            >
              进入用户管理
            </button>
          )}

          <button
            type="button"
            style={styles.secondaryButton}
            onClick={() => navigate('/account')}
          >
            个人中心
          </button>

          <button
            type="button"
            style={styles.dangerButton}
            onClick={handleLogout}
          >
            退出登录
          </button>
        </div>
      </section>
    </div>
  )
}

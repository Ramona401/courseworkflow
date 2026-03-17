/**
 * 登录页面 - Apple 风格设计
 * - 毛玻璃效果卡片
 * - 简洁优雅的表单
 * - 流畅的交互动效
 */
import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate, Navigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import { login as apiLogin } from '@/api/auth'

export default function LoginPage() {
  const navigate = useNavigate()
  const { user, login } = useAuth()

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  if (user) {
    return <Navigate to="/" replace />
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')

    if (!username.trim() || !password.trim()) {
      setError('请输入用户名和密码')
      return
    }

    setLoading(true)
    try {
      const result = await apiLogin(username.trim(), password)
      login(result.token, result.user)
      navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  // 输入框聚焦样式
  const handleFocus = (e: React.FocusEvent<HTMLInputElement>) => {
    e.target.style.borderColor = '#007aff'
    e.target.style.boxShadow = '0 0 0 3px rgba(0,122,255,0.15)'
  }
  const handleBlur = (e: React.FocusEvent<HTMLInputElement>) => {
    e.target.style.borderColor = 'rgba(0,0,0,0.1)'
    e.target.style.boxShadow = 'none'
  }

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '12px 16px',
    fontSize: '15px',
    border: '1px solid rgba(0,0,0,0.1)',
    borderRadius: '12px',
    background: 'rgba(255,255,255,0.8)',
    color: '#1d1d1f',
    transition: 'all 0.2s ease',
    boxSizing: 'border-box' as const,
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      padding: '20px',
    }}>
      {/* 背景装饰 */}
      <div style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        background: 'radial-gradient(circle at 30% 20%, rgba(255,255,255,0.15) 0%, transparent 50%), radial-gradient(circle at 70% 80%, rgba(255,255,255,0.1) 0%, transparent 50%)',
        pointerEvents: 'none',
      }} />

      {/* 登录卡片 */}
      <div style={{ width: '100%', maxWidth: '400px', position: 'relative', zIndex: 1 }}>
        {/* Logo */}
        <div style={{ textAlign: 'center', marginBottom: '36px' }}>
          <div style={{
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: '72px',
            height: '72px',
            background: 'rgba(255,255,255,0.2)',
            backdropFilter: 'blur(20px)',
            WebkitBackdropFilter: 'blur(20px)',
            borderRadius: '22px',
            border: '1px solid rgba(255,255,255,0.3)',
            marginBottom: '16px',
            boxShadow: '0 8px 32px rgba(0,0,0,0.1)',
          }}>
            <span style={{ color: '#fff', fontSize: '28px', fontWeight: 700, letterSpacing: '-0.5px' }}>TE</span>
          </div>
          <h1 style={{ color: '#fff', fontSize: '28px', fontWeight: 600, letterSpacing: '-0.5px', margin: '0 0 4px 0' }}>TE-DNA 2.0</h1>
          <p style={{ color: 'rgba(255,255,255,0.7)', fontSize: '15px', margin: 0 }}>课程工作流平台</p>
        </div>

        {/* 表单卡片 */}
        <div style={{
          background: 'rgba(255,255,255,0.85)',
          backdropFilter: 'blur(20px)',
          WebkitBackdropFilter: 'blur(20px)',
          borderRadius: '20px',
          border: '1px solid rgba(255,255,255,0.6)',
          padding: '36px',
          boxShadow: '0 20px 60px rgba(0,0,0,0.15), 0 1px 3px rgba(0,0,0,0.05)',
        }}>
          <form onSubmit={handleSubmit}>
            {/* 错误提示 */}
            {error && (
              <div style={{
                marginBottom: '20px',
                padding: '12px 16px',
                background: 'rgba(255,59,48,0.08)',
                border: '1px solid rgba(255,59,48,0.2)',
                borderRadius: '12px',
                color: '#ff3b30',
                fontSize: '14px',
                fontWeight: 500,
              }}>{error}</div>
            )}

            {/* 用户名 */}
            <div style={{ marginBottom: '16px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: '#1d1d1f', marginBottom: '8px' }}>用户名</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="请输入用户名"
                autoComplete="username"
                autoFocus
                style={inputStyle}
                onFocus={handleFocus}
                onBlur={handleBlur}
              />
            </div>

            {/* 密码 */}
            <div style={{ marginBottom: '28px' }}>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: '#1d1d1f', marginBottom: '8px' }}>密码</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="请输入密码"
                autoComplete="current-password"
                style={inputStyle}
                onFocus={handleFocus}
                onBlur={handleBlur}
              />
            </div>

            {/* 登录按钮 */}
            <button
              type="submit"
              disabled={loading}
              style={{
                width: '100%',
                padding: '13px',
                fontSize: '16px',
                fontWeight: 600,
                color: '#fff',
                background: loading ? '#999' : 'linear-gradient(135deg, #007aff 0%, #5856d6 100%)',
                border: 'none',
                borderRadius: '12px',
                cursor: loading ? 'not-allowed' : 'pointer',
                transition: 'all 0.2s ease',
                letterSpacing: '0.5px',
                boxShadow: loading ? 'none' : '0 4px 15px rgba(0,122,255,0.3)',
              }}
              onMouseEnter={(e) => {
                if (!loading) {
                  (e.target as HTMLElement).style.transform = 'translateY(-1px)'
                  ;(e.target as HTMLElement).style.boxShadow = '0 6px 20px rgba(0,122,255,0.4)'
                }
              }}
              onMouseLeave={(e) => {
                (e.target as HTMLElement).style.transform = 'translateY(0)'
                ;(e.target as HTMLElement).style.boxShadow = '0 4px 15px rgba(0,122,255,0.3)'
              }}
            >
              {loading ? (
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: '8px' }}>
                  <svg width="18" height="18" viewBox="0 0 24 24" style={{ animation: 'spin 1s linear infinite' }}>
                    <circle cx="12" cy="12" r="10" stroke="rgba(255,255,255,0.3)" strokeWidth="3" fill="none" />
                    <path d="M12 2 A10 10 0 0 1 22 12" stroke="#fff" strokeWidth="3" fill="none" strokeLinecap="round" />
                  </svg>
                  登录中...
                </span>
              ) : '登 录'}
            </button>
          </form>
        </div>

        {/* 底部 */}
        <p style={{ textAlign: 'center', color: 'rgba(255,255,255,0.5)', fontSize: '12px', marginTop: '28px' }}>
          TE-DNA 2.0 &copy; 2026 PKU AI Lab
        </p>
      </div>
    </div>
  )
}

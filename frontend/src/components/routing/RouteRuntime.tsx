/**
 * RouteRuntime — 应用路由公共运行时。
 *
 * 职责：
 * - 提供路由懒加载期间的统一Loading页面；
 * - 捕获懒加载Chunk或页面渲染异常；
 * - 不承担认证、角色、教育域或业务授权判断。
 *
 * 从App.tsx拆出，避免根路由文件继续逼近900行硬上限。
 */

import {
  Component,
  type ErrorInfo,
  type ReactNode,
} from 'react'

interface RouteErrorBoundaryProps {
  children: ReactNode
}

interface RouteErrorBoundaryState {
  hasError: boolean
}

export class RouteErrorBoundary extends Component<
  RouteErrorBoundaryProps,
  RouteErrorBoundaryState
> {
  constructor(props: RouteErrorBoundaryProps) {
    super(props)

    this.state = {
      hasError: false,
    }
  }

  static getDerivedStateFromError(
    _error: Error,
  ): RouteErrorBoundaryState {
    return {
      hasError: true,
    }
  }

  componentDidCatch(
    error: Error,
    info: ErrorInfo,
  ) {
    console.error(
      '[RouteErrorBoundary]',
      error,
      info,
    )
  }

  render() {
    if (this.state.hasError) {
      return (
        <div
          style={{
            height: '100vh',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: '#FAFBFC',
          }}
        >
          <div
            style={{
              textAlign: 'center',
              maxWidth: '400px',
              padding: '0 20px',
            }}
          >
            <div
              style={{
                fontSize: '48px',
                marginBottom: '16px',
              }}
            >
              😵
            </div>

            <div
              style={{
                fontSize: '18px',
                fontWeight: 700,
                color: '#1F2937',
                marginBottom: '8px',
              }}
            >
              页面加载失败
            </div>

            <div
              style={{
                fontSize: '13px',
                color: '#6B7280',
                marginBottom: '20px',
                lineHeight: 1.6,
              }}
            >
              可能是网络波动导致资源加载失败，请刷新页面重试。
            </div>

            <button
              onClick={() => window.location.reload()}
              style={{
                padding: '10px 28px',
                borderRadius: '10px',
                border: 'none',
                background:
                  'linear-gradient(135deg, #4F7BE8, #6366F1)',
                color: '#fff',
                fontSize: '14px',
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              刷新页面
            </button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}

export function PageLoading() {
  return (
    <div
      style={{
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#FAFBFC',
      }}
    >
      <div
        style={{
          textAlign: 'center',
        }}
      >
        <div
          style={{
            width: '28px',
            height: '28px',
            border: '2.5px solid #E5E7EB',
            borderTopColor: '#4F7BE8',
            borderRadius: '50%',
            animation: 'spin 0.8s linear infinite',
            margin: '0 auto 10px',
          }}
        />

        <style>
          {'@keyframes spin { to { transform: rotate(360deg); } }'}
        </style>

        <div
          style={{
            color: '#9CA3AF',
            fontSize: '13px',
          }}
        >
          页面加载中...
        </div>
      </div>
    </div>
  )
}

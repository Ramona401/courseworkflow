/**
 * 主布局组件 - Apple 风格
 * - 左侧深色侧边栏 + 右侧浅色内容区
 * - 顶部精致的标题栏
 * - P4-7更新：Pipeline详情页动态标题支持
 */
import { Outlet, useLocation } from 'react-router-dom'
import Sidebar from './Sidebar'

// 路由标题映射（P4-7更新：添加pipelines路由）
const pageTitles: Record<string, string> = {
  '/': '仪表盘',
  '/users': '用户管理',
  '/ai-config': 'AI 配置',
  '/prompts': '提示词管理',
  '/external-data': '外部数据配置',
  '/courses': '课程管理',
  '/pipelines': 'Pipeline',
  '/review': '审核中心',
  '/settings': '系统设置',
}

/** 根据路径获取页面标题（支持动态路由） */
function getPageTitle(pathname: string): string {
  // 精确匹配
  if (pageTitles[pathname]) {
    return pageTitles[pathname]
  }
  // Pipeline详情页：/pipelines/{id}
  if (pathname.startsWith('/pipelines/')) {
    return 'Pipeline 详情'
  }
  return 'TE-DNA 2.0'
}

export default function MainLayout() {
  const location = useLocation()
  const pageTitle = getPageTitle(location.pathname)

  return (
    <div style={{
      display: 'flex',
      height: '100vh',
      overflow: 'hidden',
      background: '#f5f5f7',
    }}>
      {/* 左侧侧边栏 */}
      <Sidebar />

      {/* 右侧内容区 */}
      <div style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}>
        {/* 顶部栏 */}
        <header style={{
          height: '64px',
          background: 'rgba(255,255,255,0.72)',
          backdropFilter: 'blur(20px)',
          WebkitBackdropFilter: 'blur(20px)',
          borderBottom: '1px solid rgba(0,0,0,0.06)',
          display: 'flex',
          alignItems: 'center',
          padding: '0 28px',
          flexShrink: 0,
        }}>
          <h2 style={{
            fontSize: '18px',
            fontWeight: 600,
            color: '#1d1d1f',
            margin: 0,
            letterSpacing: '-0.3px',
          }}>{pageTitle}</h2>
        </header>

        {/* 页面内容 */}
        <main style={{
          flex: 1,
          overflowY: 'auto',
          padding: '28px',
        }}>
          <Outlet />
        </main>
      </div>
    </div>
  )
}

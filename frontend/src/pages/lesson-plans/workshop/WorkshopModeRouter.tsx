/**
 * WorkshopModeRouter.tsx — 备课工坊模式路由器（迭代3.5 Phase A）
 *
 * 职责：在 /lesson-plans index 路由上按偏好决定渲染哪个模式：
 *   - conversation（对话模式，新）→ ConversationModePage
 *   - expert（专家模式，原阶段式界面）→ WorkshopPage（一行未改，仅叠加悬浮切换钮）
 *
 * 模式决策（resolveWorkshopMode）：URL ?mode= > 教案级记忆 > 全局偏好 > 系统默认。
 * 切换 = persistWorkshopMode 持久化 + setState 重挂载另一页面；
 * 两模式共享 sessionStorage workshop_active_plan_id 与全部服务端数据，
 * 重挂载即重拉服务端真相 → 互切状态零丢失（Phase A 验收红线4）。
 *
 * 回滚：workshopMode.ts 的 DEFAULT_WORKSHOP_MODE 改 'expert' 一行全局回退。
 */
import { useState, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import WorkshopPage from './WorkshopPage'
import ConversationModePage from './conversation/ConversationModePage'
import { resolveWorkshopMode, persistWorkshopMode, type WorkshopMode } from './conversation/workshopMode'

export default function WorkshopModeRouter() {
  const [searchParams] = useSearchParams()
  // 模式只在挂载时解析一次；之后由切换按钮驱动（URL参数变化属罕见路径，刷新即重解析）
  const [mode, setMode] = useState<WorkshopMode>(() => resolveWorkshopMode(searchParams.get('mode')))

  /** 切换模式：持久化偏好（全局 + 当前活跃教案级）并重挂载目标页面 */
  const switchMode = useCallback((m: WorkshopMode) => {
    persistWorkshopMode(m)
    setMode(m)
  }, [])

  // ===== 对话模式 =====
  if (mode === 'conversation') {
    return <ConversationModePage onSwitchMode={() => switchMode('expert')} />
  }

  // ===== 专家模式：原版页面零改动 + 右下角悬浮切换钮 =====
  return (
    <>
      <WorkshopPage />
      <button
        onClick={() => switchMode('conversation')}
        title="切换到对话备课模式（同一教案，随时互切，状态不丢失）"
        style={{
          position: 'fixed', right: '18px', bottom: '18px', zIndex: 9000,
          padding: '8px 16px', borderRadius: '22px', border: 'none',
          background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', color: '#fff',
          fontSize: '12px', fontWeight: 600, cursor: 'pointer',
          boxShadow: '0 4px 16px rgba(79,123,232,0.4)',
        }}
      >
        💬 对话模式
      </button>
    </>
  )
}

/**
 * 通知铃铛组件 — NotificationBell v1.1（阶段5c 前端 + 阶段5 收尾教案审核接线）
 *
 * 自包含的顶栏通知入口，三大系统布局（LPLayout/CWLayout/MainLayout）共用：
 *   每个 Layout 在 header 里放一个 <NotificationBell />（用户菜单左侧）即可，逻辑只在此写一遍。
 *
 * v1.1 改动：typeIcon 补 lp_review_submitted（教案提交审核 → 发给 L1 审核员）图标 📤，
 *   与课件 cw_review_submitted 同图标语义。教案审核三事件（submitted/approved/revision）
 *   现已在后端接线（lesson_plan_review_notify.go），前端图标全部就位。
 *
 * 行为：
 *   - 🔔 图标 + 右上角未读红点（99+ 封顶）。
 *   - 轮询未读数：挂载后每 45 秒拉一次 /notifications/unread-count（极轻端点）；
 *     document.hidden（标签页不可见）时跳过该次轮询，省请求；卸载清除定时器。
 *   - 点铃铛弹面板（fixed 定位，点外部关闭，镜像 CWLayout 的 DropdownPortal 模式）；
 *     打开时拉最近 20 条列表。
 *   - 每条：类型图标 + 标题 +（退回意见等 body）+ 相对时间 + 未读蓝点。
 *   - 点某条 → 标该条已读 → navigate(link) 跳转 → 关面板。
 *   - 顶部「全部已读」一键清红点。
 *
 * 样式：对齐三大 Layout 的 inline style 风格（圆角/阴影/中性灰），不引入新依赖、不用 Tailwind。
 *
 * 跳转说明：通知 link 为绝对路径（/courseware/{id}、/courseware/review、
 *   /lesson-plans/review-v2、/lesson-plans/plans/{id} 等）。
 *   优先用 react-router 的 navigate（同一 SPA 路由表内跳转，不刷新页面）；
 *   若目标不在当前路由树内导致 navigate 无效，浏览器仍会停留——本版先用 navigate，
 *   后续若发现跨系统跳不过去再降级 window.location.href（已在 onClickItem 留好切换点）。
 */
import { useState, useRef, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  listNotifications,
  getUnreadCount,
  markRead,
  markAllRead,
  type Notification,
} from '@/api/notifications'

/* ==================== 轮询间隔 ==================== */
const POLL_INTERVAL_MS = 45_000 // 45 秒

/* ==================== 通知类型 → 图标 ==================== */
function typeIcon(type: string): string {
  switch (type) {
    // —— 集体备课 ——
    case 'cw_collab_invited': return '👥'
    case 'cw_collab_removed': return '🚪'
    case 'cw_collab_ended':   return '🏁'
    case 'cw_collab_started': return '🚀'
    // —— 课件审核 ——
    case 'cw_review_submitted': return '📤'
    case 'cw_review_approved':  return '✅'
    case 'cw_review_revision':  return '↩️'
    // —— 教案审核（阶段5 收尾接线，与课件审核同图标语义）——
    case 'lp_review_submitted': return '📤'
    case 'lp_review_approved':  return '✅'
    case 'lp_review_revision':  return '↩️'
    // —— 预留类型的兜底 ——
    case 'lp_interaction':      return '💬'
    case 'cw_inspection':       return '🔍'
    default: return '🔔'
  }
}

/* ==================== 相对时间 ==================== */
function relativeTime(iso: string): string {
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return ''
  const diff = Date.now() - t
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return min + ' 分钟前'
  const hr = Math.floor(min / 60)
  if (hr < 24) return hr + ' 小时前'
  const day = Math.floor(hr / 24)
  if (day < 30) return day + ' 天前'
  const d = new Date(iso)
  return (d.getMonth() + 1) + '月' + d.getDate() + '日'
}

/* ==================== 面板 Portal（镜像 CWLayout.DropdownPortal）==================== */
function PanelPortal({ children, triggerRef }: {
  children: React.ReactNode
  triggerRef: React.RefObject<HTMLDivElement | null>
}) {
  const [pos, setPos] = useState({ top: 0, right: 0 })
  useEffect(() => {
    if (triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect()
      setPos({ top: rect.bottom + 8, right: window.innerWidth - rect.right })
    }
  }, [triggerRef])
  return (
    <div style={{
      position: 'fixed', top: pos.top, right: pos.right, width: '360px', maxHeight: '480px',
      background: '#fff', borderRadius: '16px', border: '1px solid #E5E7EB',
      boxShadow: '0 8px 32px rgba(0,0,0,0.12)', overflow: 'hidden', zIndex: 9999,
      display: 'flex', flexDirection: 'column',
    }}>
      {children}
    </div>
  )
}

/* ==================== 主组件 ==================== */
export default function NotificationBell() {
  const navigate = useNavigate()
  const [unread, setUnread] = useState(0)
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<Notification[]>([])
  const [loading, setLoading] = useState(false)
  const wrapRef = useRef<HTMLDivElement>(null)

  /* —— 轮询未读数 —— */
  const pollUnread = useCallback(async () => {
    if (document.hidden) return // 标签页不可见时跳过，省请求
    try {
      const n = await getUnreadCount()
      setUnread(n)
    } catch {
      // 轮询失败静默（含 401 已由 client 拦截器处理），不打扰用户
    }
  }, [])

  useEffect(() => {
    pollUnread() // 挂载即拉一次
    const timer = window.setInterval(pollUnread, POLL_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [pollUnread])

  /* —— 点外部关闭面板 —— */
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) setOpen(false)
    }
    if (open) document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  /* —— 打开面板：拉列表 —— */
  const togglePanel = async () => {
    const next = !open
    setOpen(next)
    if (next) {
      setLoading(true)
      try {
        const resp = await listNotifications(20, 0, false)
        setItems(resp.notifications)
        setUnread(resp.unread_count) // 顺带刷新红点口径
      } catch {
        setItems([])
      } finally {
        setLoading(false)
      }
    }
  }

  /* —— 点某条：标已读 + 跳转 + 关面板 —— */
  const onClickItem = async (n: Notification) => {
    setOpen(false)
    if (!n.is_read) {
      try {
        await markRead(n.id)
        setUnread(u => Math.max(0, u - 1))
        setItems(list => list.map(it => it.id === n.id ? { ...it, is_read: true } : it))
      } catch {
        // 标已读失败不阻断跳转
      }
    }
    if (n.link) {
      navigate(n.link)
      // 降级点：若后续发现跨系统 navigate 跳不过去，改用 window.location.href = n.link
    }
  }

  /* —— 全部已读 —— */
  const onMarkAll = async () => {
    try {
      await markAllRead()
      setUnread(0)
      setItems(list => list.map(it => ({ ...it, is_read: true })))
    } catch {
      // 忽略
    }
  }

  const badge = unread > 99 ? '99+' : String(unread)

  return (
    <div ref={wrapRef} style={{ position: 'relative', marginRight: '12px' }}>
      {/* 铃铛按钮 */}
      <button
        onClick={togglePanel}
        title="通知"
        style={{
          position: 'relative', width: '38px', height: '38px', borderRadius: '50%',
          border: '1px solid #E5E7EB', background: open ? '#F3F4F6' : 'transparent',
          cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
          fontSize: '18px', transition: 'all 150ms ease',
        }}
        onMouseEnter={e => { if (!open) (e.currentTarget as HTMLElement).style.background = '#F9FAFB' }}
        onMouseLeave={e => { if (!open) (e.currentTarget as HTMLElement).style.background = 'transparent' }}
      >
        🔔
        {unread > 0 && (
          <span style={{
            position: 'absolute', top: '-2px', right: '-2px', minWidth: '18px', height: '18px',
            padding: '0 5px', borderRadius: '9px', background: '#EF4444', color: '#fff',
            fontSize: '11px', fontWeight: 700, lineHeight: '18px', textAlign: 'center',
            boxShadow: '0 0 0 2px #fff',
          }}>{badge}</span>
        )}
      </button>

      {/* 通知面板 */}
      {open && (
        <PanelPortal triggerRef={wrapRef}>
          {/* 头部 */}
          <div style={{
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            padding: '14px 16px', borderBottom: '1px solid #F3F4F6', flexShrink: 0,
          }}>
            <span style={{ fontSize: '15px', fontWeight: 600, color: '#1F2937' }}>通知</span>
            {items.some(it => !it.is_read) && (
              <button onClick={onMarkAll} style={{
                border: 'none', background: 'transparent', cursor: 'pointer',
                fontSize: '12px', color: '#3B82F6', fontWeight: 500,
              }}>全部已读</button>
            )}
          </div>

          {/* 列表 */}
          <div style={{ overflowY: 'auto', flex: 1 }}>
            {loading ? (
              <div style={{ padding: '32px', textAlign: 'center', color: '#9CA3AF', fontSize: '13px' }}>加载中…</div>
            ) : items.length === 0 ? (
              <div style={{ padding: '40px 16px', textAlign: 'center', color: '#9CA3AF', fontSize: '13px' }}>
                <div style={{ fontSize: '28px', marginBottom: '8px' }}>🔕</div>
                暂无通知
              </div>
            ) : (
              items.map(n => (
                <button
                  key={n.id}
                  onClick={() => onClickItem(n)}
                  style={{
                    width: '100%', display: 'flex', gap: '10px', padding: '12px 16px',
                    border: 'none', borderBottom: '1px solid #F9FAFB', cursor: 'pointer',
                    textAlign: 'left', background: n.is_read ? 'transparent' : 'rgba(59,130,246,0.04)',
                    transition: 'background 150ms ease',
                  }}
                  onMouseEnter={e => (e.currentTarget as HTMLElement).style.background = '#F9FAFB'}
                  onMouseLeave={e => (e.currentTarget as HTMLElement).style.background = n.is_read ? 'transparent' : 'rgba(59,130,246,0.04)'}
                >
                  <span style={{ fontSize: '18px', flexShrink: 0, lineHeight: '20px' }}>{typeIcon(n.type)}</span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{
                      fontSize: '13px', color: '#1F2937', fontWeight: n.is_read ? 400 : 600,
                      lineHeight: '18px', marginBottom: '2px',
                    }}>{n.title}</div>
                    {n.body && (
                      <div style={{
                        fontSize: '12px', color: '#6B7280', lineHeight: '16px', marginBottom: '3px',
                        overflow: 'hidden', textOverflow: 'ellipsis', display: '-webkit-box',
                        WebkitLineClamp: 2, WebkitBoxOrient: 'vertical',
                      }}>{n.body}</div>
                    )}
                    <div style={{ fontSize: '11px', color: '#9CA3AF' }}>{relativeTime(n.created_at)}</div>
                  </div>
                  {!n.is_read && (
                    <span style={{
                      width: '8px', height: '8px', borderRadius: '50%', background: '#3B82F6',
                      flexShrink: 0, marginTop: '6px',
                    }} />
                  )}
                </button>
              ))
            )}
          </div>
        </PanelPortal>
      )}
    </div>
  )
}

/**
 * useGroupLead — 「当前用户是否教研组组长」共享判定 Hook
 *
 * 背景（生产端功能与组织角色绑定）：
 *   「备课配方」「组件管理」是教研生产端能力，此前仅绑账户系统身份
 *   （admin / senior_operator）。实际业务中大量教研员账户使用骨干教师
 *   （operator）身份开通，需要在被任命为教研组「组长」后自动解锁这两个功能，
 *   而无需把账户身份改成学校管理员（否则会连带看到用户管理，权限过大）。
 *
 * 判定依据：
 *   复用现成接口 GET /api/v1/ai-assistants/my-groups（登录即可调用），
 *   后端 ListMyLeadOrBackboneGroups 返回当前用户任 lead / backbone 的教研组清单，
 *   每组带 role 字段（'lead' 组长 / 'backbone' 骨干）。
 *   本 Hook 判定：groups 中存在 role 命中 UNLOCK_GROUP_ROLES 即视为解锁。
 *
 * 口径（左总拍板）：
 *   当前严格绑「组长」——UNLOCK_GROUP_ROLES = ['lead']。
 *   将来若要放宽到「组长+骨干」（与平台其他组级管理权口径对齐），
 *   只需把 'backbone' 加进该常量数组，一行改动全局生效。
 *
 * 缓存设计：
 *   模块级缓存（按用户ID）+ 进行中请求去重（inflight Promise），
 *   使 LPSidebar（菜单显隐）与 App.tsx 的 LeadOrRoleGuard（路由守卫）
 *   在同一次会话内共用一次 API 请求，互不重复请求。
 *   请求失败不缓存结果（fail-closed 返回 false，但下次挂载会重试），
 *   避免一次网络抖动让组长整个会话都看不到菜单。
 *
 * 安全边界说明：
 *   本 Hook 与前端守卫只是体验层收口；配方/组件后端接口本就对所有登录
 *   用户放行（管控靠 service 层归属校验），故前端放宽不产生越权风险。
 */
import { useEffect, useState } from 'react'
import { useAuth } from '@/store/auth'
import { getMyPublishGroups } from '@/api/ai-assistants'

/**
 * 解锁生产端菜单所需的教研组内角色白名单。
 * 当前仅组长；将来放宽骨干：改为 ['lead', 'backbone'] 即可。
 */
const UNLOCK_GROUP_ROLES: string[] = ['lead']

/* ==================== 模块级缓存（按用户ID） ==================== */

/** 已缓存判定结果对应的用户ID（换用户登录时自动失效） */
let cachedUserId: string | null = null
/** 已缓存的判定结果；null = 尚未成功判定过（含失败重试场景） */
let cachedIsLead: boolean | null = null
/** 进行中的请求（去重：多个组件同时挂载只发一次请求） */
let inflight: Promise<boolean> | null = null

/**
 * 实际执行「是否组长」判定（带缓存与请求去重）。
 * @param userId 当前登录用户ID
 * @returns 是否为任一教研组的组长
 */
async function checkIsLead(userId: string): Promise<boolean> {
  // 命中缓存直接返回
  if (cachedUserId === userId && cachedIsLead !== null) {
    return cachedIsLead
  }
  // 换了用户：清空旧缓存与旧请求
  if (cachedUserId !== userId) {
    cachedUserId = userId
    cachedIsLead = null
    inflight = null
  }
  // 复用进行中的请求（去重）
  if (!inflight) {
    inflight = (async () => {
      try {
        const resp = await getMyPublishGroups()
        const groups = resp?.groups || []
        const lead = groups.some(g => UNLOCK_GROUP_ROLES.includes(g.role))
        cachedIsLead = lead // 仅成功时写缓存
        return lead
      } catch {
        // 失败不写缓存（cachedIsLead 保持 null），fail-closed 返回 false，
        // 下次组件挂载会自动重试，避免一次网络抖动锁死整个会话。
        return false
      } finally {
        inflight = null
      }
    })()
  }
  return inflight
}

/**
 * useGroupLead — 组件侧使用入口
 *
 * @param enabled 是否需要判定。传 false 时不发任何请求（例如当前用户
 *                身份已在 admin/senior_operator 白名单内，无需组长判定）。
 * @returns isLead   是否为任一教研组组长（enabled=false 时恒为 false）
 * @returns checking 是否判定中（供路由守卫展示加载态，防止误跳首页）
 */
export function useGroupLead(enabled: boolean): { isLead: boolean; checking: boolean } {
  const { user } = useAuth()
  const userId = user?.id || ''

  // 初始态：命中缓存则直接以缓存值初始化，避免闪烁
  const hasCache = enabled && !!userId && cachedUserId === userId && cachedIsLead !== null
  const [isLead, setIsLead] = useState<boolean>(hasCache ? (cachedIsLead as boolean) : false)
  const [checking, setChecking] = useState<boolean>(enabled && !!userId && !hasCache)

  useEffect(() => {
    // 不需要判定 / 未登录：直接落定 false
    if (!enabled || !userId) {
      setIsLead(false)
      setChecking(false)
      return
    }
    // 命中缓存：直接落定缓存值
    if (cachedUserId === userId && cachedIsLead !== null) {
      setIsLead(cachedIsLead)
      setChecking(false)
      return
    }
    // 发起（或复用进行中的）判定请求
    let mounted = true
    setChecking(true)
    checkIsLead(userId).then(lead => {
      if (mounted) {
        setIsLead(lead)
        setChecking(false)
      }
    })
    return () => { mounted = false }
  }, [enabled, userId])

  return { isLead, checking }
}

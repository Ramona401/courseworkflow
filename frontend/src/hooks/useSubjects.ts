/**
 * useSubjects — 全平台学科下拉的统一数据源 Hook
 *
 * 背景：
 *   学科原散落在 8+ 处前端硬编码，各副本不一致（备课下拉缺劳动/道法/美术等）。
 *   现改为单一真相源——数据库 subjects 表，后台可运营增删改，前端各下拉统一用本 Hook。
 *
 * 设计（双保险 + 全局共享）：
 *   - 模块级缓存：学科表全平台共享、与用户无关，故只拉一次并缓存，多组件复用。
 *   - inflight 去重：多个下拉同时首次挂载只发一次请求（对齐 useGroupLead 范式）。
 *   - 失败兜底：接口失败/首屏未返回前，一律回退内置 DEFAULT_SUBJECTS，保证下拉永不空白。
 *     失败不写缓存 → 下次有组件挂载会自动重试，避免一次网络抖动锁死整个会话。
 *
 * 用法：
 *   const { subjects } = useSubjects()                 // 纯学科名数组（下拉/按钮）
 *   const { subjects } = useSubjects({ withAll: true })  // 前置「全部」（筛选场景）
 *   const { subjects } = useSubjects({ withAny: true })  // 前置空串「不限」（AI 助手场景）
 *   subjects 恒为非空数组：加载中/失败时是内置兜底清单，成功后是数据库实时数据。
 *
 * 刷新：
 *   后台学科管理界面增删改后调用 refreshSubjects() 清缓存并重拉，
 *   使本会话内其它已挂载的下拉在下次读取时拿到最新数据。
 */
import { useEffect, useState } from 'react'
import { getSubjects } from '@/api/subjects'
import { DEFAULT_SUBJECTS, withAllOption, withAnyOption, type SubjectItem } from '@/constants/subjects'

/* ==================== 模块级缓存（全局共享，与用户无关） ==================== */

/** 已成功拉取的学科名清单；null = 尚未成功拉取过（含失败重试场景） */
let cachedNames: string[] | null = null
/** 进行中的请求（去重：多个组件同时挂载只发一次） */
let inflight: Promise<string[]> | null = null
/** 订阅者集合：缓存更新（如 refreshSubjects 后）通知所有已挂载 Hook 重渲染 */
const subscribers = new Set<() => void>()

/** 通知所有订阅者刷新 */
function notifyAll() {
  subscribers.forEach(fn => {
    try { fn() } catch { /* 单个订阅者异常不影响其它 */ }
  })
}

/**
 * 实际拉取学科（带缓存与请求去重）。成功写缓存；失败不写缓存、返回内置兜底清单。
 */
async function loadSubjects(): Promise<string[]> {
  if (cachedNames !== null) return cachedNames
  if (!inflight) {
    inflight = (async () => {
      try {
        const items: SubjectItem[] = await getSubjects()
        const names = items.map(s => s.name).filter(Boolean)
        // 极端情况：接口返回空数组也回退兜底，避免下拉空白
        cachedNames = names.length > 0 ? names : DEFAULT_SUBJECTS
        return cachedNames
      } catch {
        // 失败不写缓存（cachedNames 保持 null，下次自动重试），本次返回兜底
        return DEFAULT_SUBJECTS
      } finally {
        inflight = null
      }
    })()
  }
  return inflight
}

/**
 * 清空缓存并重新拉取（后台增删改学科后调用），拉取完成后通知所有订阅者刷新。
 * 供学科管理界面在保存成功后调用，使全站下拉即时同步最新数据。
 */
export async function refreshSubjects(): Promise<void> {
  cachedNames = null
  inflight = null
  await loadSubjects()
  notifyAll()
}

/** useSubjects 选项 */
export interface UseSubjectsOptions {
  /** 前置「全部」选项（筛选场景，如我的教案 / 课本 / 教案库） */
  withAll?: boolean
  /** 前置空串「不限」选项（AI 助手学科偏好等） */
  withAny?: boolean
}

/**
 * useSubjects — 组件侧使用入口
 *
 * @returns subjects 学科名数组（恒非空：加载中/失败=内置兜底清单，成功=数据库实时数据）
 * @returns loading  是否首次加载中（一般无需处理，subjects 已兜底）
 */
export function useSubjects(options: UseSubjectsOptions = {}): { subjects: string[]; loading: boolean } {
  const { withAll = false, withAny = false } = options

  // 初始态：命中缓存直接用，避免闪烁；未命中先用内置兜底清单
  const [names, setNames] = useState<string[]>(cachedNames || DEFAULT_SUBJECTS)
  const [loading, setLoading] = useState<boolean>(cachedNames === null)

  useEffect(() => {
    let mounted = true

    // 订阅缓存更新（refreshSubjects 后被通知）
    const onUpdate = () => {
      if (mounted) setNames(cachedNames || DEFAULT_SUBJECTS)
    }
    subscribers.add(onUpdate)

    // 已有缓存：直接落定，无需请求
    if (cachedNames !== null) {
      setNames(cachedNames)
      setLoading(false)
    } else {
      setLoading(true)
      loadSubjects().then(list => {
        if (mounted) {
          setNames(list)
          setLoading(false)
        }
      })
    }

    return () => {
      mounted = false
      subscribers.delete(onUpdate)
    }
  }, [])

  // 按需前置「全部」/「不限」
  let result = names
  if (withAll) result = withAllOption(names)
  else if (withAny) result = withAnyOption(names)

  return { subjects: result, loading }
}

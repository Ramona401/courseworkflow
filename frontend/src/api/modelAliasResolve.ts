/**
 * modelAliasResolve.ts — 前端模型别名解析（批三-3a）
 *
 * 复现后端 repository.ResolveModelAlias 的匹配逻辑（精确 > 前缀 > 兜底），
 * 供消费表等需要批量把真实模型名转业务别名的场景在前端本地解析，
 * 避免对每行记录都调一次后端 preview 接口。
 *
 * 用法：
 *   const resolver = await loadAliasResolver()   // 拉规则+兜底名，构造解析器（仅admin调，接口adminOnly）
 *   const alias = resolver('anthropic/claude-sonnet-4-5')  // → 业务别名
 *
 * 匹配规则（必须与后端 model_alias_repo.go 完全一致）：
 *   1) 精确：enabled 且 match_type='exact' 且 pattern === 模型名，取 priority 最高；
 *   2) 前缀：enabled 且 match_type='prefix' 且 模型名以 pattern 开头，
 *      按 priority 降序、pattern 长度降序取第一条（最长最高优先）；
 *   3) 都没命中 → 兜底名。
 */
import { getModelAliasRules, getModelAliasFallback } from './ai-config'
import type { ModelAliasRule } from './ai-config'

/** 别名解析器：输入真实模型名，返回业务别名 */
export type AliasResolver = (modelName: string) => string

/**
 * 构造一个本地别名解析器。
 * @param rules    全部别名规则（来自 getModelAliasRules）
 * @param fallback 兜底名（来自 getModelAliasFallback）
 */
export function buildAliasResolver(rules: ModelAliasRule[], fallback: string): AliasResolver {
  const fb = fallback || '智学大模型'

  // 预筛启用规则，拆成精确表和前缀表
  const exactMap = new Map<string, { alias: string; priority: number }>()
  const prefixList: { pattern: string; alias: string; priority: number }[] = []

  for (const r of rules) {
    if (!r.enabled) continue
    if (r.match_type === 'exact') {
      const prev = exactMap.get(r.pattern)
      // 同 pattern 取 priority 最高
      if (!prev || r.priority > prev.priority) {
        exactMap.set(r.pattern, { alias: r.alias, priority: r.priority })
      }
    } else if (r.match_type === 'prefix') {
      prefixList.push({ pattern: r.pattern, alias: r.alias, priority: r.priority })
    }
  }

  // 前缀表排序：priority 降序，再按 pattern 长度降序（最长最高优先）
  prefixList.sort((a, b) => {
    if (b.priority !== a.priority) return b.priority - a.priority
    return b.pattern.length - a.pattern.length
  })

  return (modelName: string): string => {
    const name = (modelName || '').trim()
    if (!name) return fb

    // 1) 精确
    const exact = exactMap.get(name)
    if (exact) return exact.alias

    // 2) 前缀（已排序，首个命中即最优）
    for (const p of prefixList) {
      if (name.startsWith(p.pattern)) return p.alias
    }

    // 3) 兜底
    return fb
  }
}

/**
 * 一次性加载别名规则并构造解析器。
 * ⚠ 仅在 admin 上下文调用：底层接口为 adminOnly，非 admin 会 403。
 * 失败时返回一个"全部兜底"的解析器（fail-safe，绝不抛出导致表格崩溃）。
 */
export async function loadAliasResolver(): Promise<AliasResolver> {
  try {
    const [rules, fallback] = await Promise.all([
      getModelAliasRules(),
      getModelAliasFallback(),
    ])
    return buildAliasResolver(rules, fallback)
  } catch {
    // 加载失败：返回恒定兜底解析器，至少不暴露真名
    const fb = '智学大模型'
    return () => fb
  }
}

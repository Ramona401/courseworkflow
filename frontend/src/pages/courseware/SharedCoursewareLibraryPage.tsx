/**
 * 共享课件库独立页 — SharedCoursewareLibraryPage v1.0
 *
 * 体验对齐教案库:把"共享课件库"从课件列表页的一个内嵌 Tab 升级为
 * 课件工坊侧边栏的独立栏目(路由 /courseware/shared),并按【源代码开放范围】
 * (code_share_scope)分层组织,层级浏览体验更清晰。
 *
 * 设计要点:
 *   1. 纯前端分层,零后端改动。
 *      后端 listSharedCoursewares 不支持按 scope 筛选(只接 subject/limit/offset),
 *      返回"我能看到的全部共享课件"(同校∪同组白名单),每条带 code_share_scope 字段。
 *      故前端一次拉全量(limit=200),按每条的 code_share_scope 本地分组,各层 Tab
 *      显本范围数量徽章。共享库列表本就不大,本地分组零风险、零接口改动。
 *   2. 分层口径 = 源代码开放范围(none/group/school/region/public)。
 *      注意:这是"代码复制权"维度,不是"可见范围"维度——
 *      一张课件能出现在共享库里说明你"看得到它"(后端可见性白名单已过滤),
 *      而它落在哪个 Tab 取决于作者把【源码】开放到了什么范围。
 *      none(源码不开放)的课件依然可见可放映,只是没有复制按钮,单列一个"仅可看"层。
 *   3. 复用既有 SharedCWCard 卡片组件(零改动),复制到我的(Fork)逻辑完全沿用。
 *
 * 空值兜底关键:共享列表后端为空时返回 coursewares:null(非 []),
 *   故 load 用 `resp.coursewares || []` 兜底,避免 .map 崩。
 */

import { useState, useEffect, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { listSharedCoursewares } from '@/api/coursewares'
import type { SharedCoursewareListItem } from '@/api/coursewares'
import { C } from './components/courseware-list/listConstants'
import SharedCWCard from './components/courseware-list/SharedCWCard'

// ==================== 分层 Tab 定义 ====================
//
// scope='all' 是聚合层(显示全部),其余四档对应 code_share_scope 的可复制范围,
// scope='none' 单列"仅可看"层(源码未开放,有可见无复制)。
interface ScopeTab {
  key: string          // 'all' | 'public' | 'region' | 'school' | 'group' | 'none'
  label: string        // Tab 文案
  emoji: string
  desc: string         // 该层说明(列表上方提示条用)
}

// 顺序:全部 → 公开 → 区域 → 本校 → 本组 → 仅可看(源码未开放)
const SCOPE_TABS: ScopeTab[] = [
  { key: 'all',    label: '全部',   emoji: '📚', desc: '同校与同教研组老师共享出来的全部课件。' },
  { key: 'public', label: '公开',   emoji: '🌐', desc: '作者开放给所有可见者复制的课件,可放映、可复制源码二次创作。' },
  { key: 'region', label: '区域',   emoji: '🗺️', desc: '作者开放给本区域复制的课件。' },
  { key: 'school', label: '本校',   emoji: '🏫', desc: '作者开放给本校老师复制的课件。' },
  { key: 'group',  label: '本组',   emoji: '👥', desc: '作者开放给本教研组复制的课件。' },
  { key: 'none',   label: '仅可看', emoji: '🔒', desc: '作者未开放源码复制,可查看效果与放映,但没有"复制到我的"。' },
]

export default function SharedCoursewareLibraryPage() {
  const navigate = useNavigate()

  // 一次拉全量,本地分组
  const [allItems, setAllItems] = useState<SharedCoursewareListItem[]>([])
  const [loading, setLoading] = useState(true)

  // 当前分层 Tab
  const [activeScope, setActiveScope] = useState<string>('all')

  useEffect(() => { loadShared() }, [])

  // 加载共享课件库全量(空值兜底:后端空列表返回 coursewares:null)
  const loadShared = async () => {
    setLoading(true)
    try {
      const resp = await listSharedCoursewares({ limit: 200 })
      setAllItems(resp.coursewares || [])
    } catch {
      setAllItems([])
    } finally {
      setLoading(false)
    }
  }

  // 各层计数(用于 Tab 徽章):按 code_share_scope 本地分组统计
  const scopeCounts = useMemo(() => {
    const counts: Record<string, number> = {
      all: allItems.length, public: 0, region: 0, school: 0, group: 0, none: 0,
    }
    for (const it of allItems) {
      const s = it.code_share_scope
      if (s in counts) counts[s] += 1
    }
    return counts
  }, [allItems])

  // 当前 Tab 下要展示的课件(all 全显,其余按 scope 过滤)
  const visibleItems = useMemo(() => {
    if (activeScope === 'all') return allItems
    return allItems.filter(it => it.code_share_scope === activeScope)
  }, [allItems, activeScope])

  const activeTabDef = SCOPE_TABS.find(t => t.key === activeScope) || SCOPE_TABS[0]

  // 分层 Tab 按钮样式
  const tabStyle = (active: boolean): React.CSSProperties => ({
    display: 'flex', alignItems: 'center', gap: '6px',
    padding: '8px 16px', borderRadius: '10px', fontSize: '14px',
    fontWeight: active ? 700 : 500, cursor: 'pointer',
    border: `1px solid ${active ? C.primary : C.border}`,
    background: active ? C.primaryBg : 'transparent',
    color: active ? C.primary : C.textSecondary,
    transition: 'all 150ms',
  })

  return (
    <div>
      {/* ==================== 页头 ==================== */}
      <div style={{ marginBottom: '20px' }}>
        <div style={{ fontSize: '22px', fontWeight: 700, color: C.textPrimary, marginBottom: '6px' }}>
          🔗 共享课件库
        </div>
        <div style={{ fontSize: '13px', color: C.textMuted }}>
          浏览同校 / 同教研组老师共享出来的课件,按源码开放范围分层查看
        </div>
      </div>

      {/* ==================== 分层 Tab(带数量徽章) ==================== */}
      <div style={{ display: 'flex', gap: '8px', marginBottom: '16px', flexWrap: 'wrap' }}>
        {SCOPE_TABS.map(tab => {
          const active = activeScope === tab.key
          const count = scopeCounts[tab.key] || 0
          return (
            <button key={tab.key} onClick={() => setActiveScope(tab.key)} style={tabStyle(active)}>
              <span>{tab.emoji}</span>
              <span>{tab.label}</span>
              <span style={{
                minWidth: '18px', textAlign: 'center',
                padding: '0 5px', borderRadius: '9px', fontSize: '11px', fontWeight: 600,
                background: active ? C.primary : '#E5E7EB',
                color: active ? '#fff' : C.textMuted,
              }}>{count}</span>
            </button>
          )
        })}
      </div>

      {/* ==================== 当前层说明条 ==================== */}
      <div style={{
        padding: '10px 16px', borderRadius: '10px', marginBottom: '16px',
        background: 'rgba(37,99,235,0.05)', border: '1px solid rgba(37,99,235,0.12)',
        fontSize: '13px', color: '#1E40AF', lineHeight: 1.6,
      }}>
        {activeTabDef.emoji} {activeTabDef.desc}
      </div>

      {/* ==================== 列表 ==================== */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '60px 0', color: C.textMuted }}>加载中...</div>
      ) : visibleItems.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '80px 0' }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>🤝</div>
          <div style={{ fontSize: '16px', color: C.textSecondary, marginBottom: '8px' }}>
            {activeScope === 'all' ? '暂无共享课件' : `「${activeTabDef.label}」分类下暂无课件`}
          </div>
          <div style={{ fontSize: '13px', color: C.textMuted }}>
            {activeScope === 'all'
              ? '当同校 / 同组老师把课件共享出来后,会出现在这里'
              : '可切换到「全部」查看其他范围的共享课件'}
          </div>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: '16px' }}>
          {visibleItems.map(item => (
            <SharedCWCard
              key={item.id}
              item={item}
              onClick={() => navigate('/courseware/' + item.id)}
              onForked={(newId: string) => {
                if (window.confirm('已复制到「我的课件」。是否立即打开副本进行编辑?')) {
                  navigate('/courseware/' + newId)
                }
              }}
            />
          ))}
        </div>
      )}
    </div>
  )
}

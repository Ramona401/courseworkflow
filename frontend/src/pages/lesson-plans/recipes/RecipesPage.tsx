/**
 * RecipesPage — 备课配方列表页
 *
 * Phase 7A：基础配方卡片列表
 * 迭代6：新增"配方市场"Tab + 配方卡片统计按钮 + 排行榜视图
 *
 * 两个Tab：
 *   1. 我的配方：个人配方+共享给我的配方（原有功能）
 *   2. 配方市场：已共享的配方排行榜（迭代6新增）
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getRecipes, deleteRecipe, forkRecipe, getMarketRecipes,
  type RecipeListItem, type MarketRecipeItem,
} from '@/api/recipes'
import RecipeStatsModal from './components/RecipeStatsModal'

/* ==================== 颜色常量 ==================== */
const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  accent: '#F59E0B', success: '#10B981', warning: '#F97316', danger: '#EF4444',
  text: '#1F2937', textSec: '#6B7280', textMuted: '#9CA3AF',
  bg: '#FAFBFC', card: '#FFFFFF', border: '#F3F4F6', borderHover: '#E5E7EB',
}

const SCOPE_CONFIG: Record<string, { label: string; color: string; bg: string; icon: string }> = {
  personal: { label: '个人', color: '#6B7280', bg: '#F3F4F6', icon: '👤' },
  group: { label: '教研组', color: '#8B5CF6', bg: 'rgba(139,92,246,0.08)', icon: '👥' },
  school: { label: '全校', color: '#0EA5E9', bg: 'rgba(14,165,233,0.08)', icon: '🏫' },
}

const SCOPE_FILTERS = [
  { key: 'all', label: '全部' }, { key: 'personal', label: '个人' },
  { key: 'group', label: '教研组共享' }, { key: 'school', label: '全校共享' },
]

const SUBJECTS = ['全部','AI','人工智能','语文','数学','英语','物理','化学','生物','历史','地理','政治','信息技术']

const MARKET_SORTS = [
  { key: 'composite', label: '综合排序' }, { key: 'use_count', label: '使用最多' },
  { key: 'avg_score', label: '评分最高' }, { key: 'fork_count', label: 'Fork最多' },
  { key: 'newest', label: '最新发布' },
]

/* ==================== 小组件 ==================== */
function ScopeBadge({ scope }: { scope: string }) {
  const cfg = SCOPE_CONFIG[scope] || SCOPE_CONFIG.personal
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px', padding: '3px 8px', borderRadius: '20px', background: cfg.bg, fontSize: '12px', fontWeight: 500, color: cfg.color, whiteSpace: 'nowrap' }}>
      <span style={{ fontSize: '11px' }}>{cfg.icon}</span>{cfg.label}
    </span>
  )
}

function MetaTag({ icon, text }: { icon: string; text: string }) {
  return <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px', fontSize: '13px', color: '#6B7280' }}><span>{icon}</span><span>{text}</span></span>
}

const btnBase: React.CSSProperties = {
  padding: '5px 12px', borderRadius: '6px', fontSize: '12px', fontWeight: 500,
  cursor: 'pointer', border: 'none', transition: 'all 150ms ease', whiteSpace: 'nowrap',
}

/* ==================== 配方卡片（我的配方Tab） ==================== */
function RecipeCard({ recipe, isOwner, onEdit, onFork, onDelete, onStats, loadingId }: {
  recipe: RecipeListItem; isOwner: boolean; onEdit: (id: string) => void; onFork: (id: string) => void
  onDelete: (id: string, name: string) => void; onStats: (id: string, name: string) => void; loadingId: string | null
}) {
  const [hovered, setHovered] = useState(false)
  const isLoading = loadingId === recipe.id
  const fmt = (iso: string) => { try { const d = new Date(iso); return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}` } catch { return iso } }

  return (
    <div onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)} style={{
      background: C.card, borderRadius: '12px', border: `1px solid ${hovered ? C.borderHover : C.border}`,
      padding: '20px', transition: 'all 200ms ease',
      boxShadow: hovered ? '0 4px 16px rgba(0,0,0,0.08)' : '0 1px 3px rgba(0,0,0,0.04)',
      transform: hovered ? 'translateY(-2px)' : 'none',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '12px', marginBottom: '10px' }}>
        <h3 style={{ fontSize: '15px', fontWeight: 600, color: C.text, margin: 0, lineHeight: 1.5, flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>📦 {recipe.name}</h3>
        <ScopeBadge scope={recipe.scope} />
      </div>
      <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', marginBottom: '12px' }}>
        <MetaTag icon="📚" text={recipe.subject} /><MetaTag icon="🎓" text={recipe.grade_range} /><MetaTag icon="🧩" text={`${recipe.component_count}个组件`} />
      </div>
      <div style={{ display: 'flex', gap: '16px', marginBottom: '12px' }}>
        <span style={{ fontSize: '12px', color: C.textMuted }}>使用 <span style={{ fontWeight: 600, color: C.text }}>{recipe.use_count}</span> 次</span>
        <span style={{ fontSize: '12px', color: C.textMuted }}>被Fork <span style={{ fontWeight: 600, color: C.text }}>{recipe.fork_count}</span> 次</span>
        <span style={{ fontSize: '12px', color: C.textMuted }}>v{recipe.version}</span>
      </div>
      {!isOwner && (
        <div style={{ display: 'inline-flex', alignItems: 'center', gap: '5px', padding: '4px 10px', borderRadius: '20px', marginBottom: '12px', background: 'rgba(139,92,246,0.06)' }}>
          <span style={{ fontSize: '12px' }}>👤</span><span style={{ fontSize: '12px', color: '#8B5CF6', fontWeight: 500 }}>来自 {recipe.author_name}</span>
        </div>
      )}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingTop: '12px', borderTop: `1px solid ${C.border}`, gap: '12px', flexWrap: 'wrap' }}>
        <span style={{ fontSize: '12px', color: C.textMuted, flexShrink: 0 }}>更新于 {fmt(recipe.updated_at)}</span>
        <div style={{ display: 'flex', gap: '6px', alignItems: 'center' }}>
          {isLoading ? <span style={{ fontSize: '12px', color: C.primary }}>处理中...</span> : <>
            {isOwner && <button onClick={() => onStats(recipe.id, recipe.name)} style={{ ...btnBase, background: 'rgba(245,158,11,0.08)', color: C.accent }}>📊 统计</button>}
            {isOwner && <button onClick={() => onEdit(recipe.id)} style={{ ...btnBase, background: C.primary, color: '#fff' }}>✏️ 编辑</button>}
            <button onClick={() => onFork(recipe.id)} style={{ ...btnBase, background: 'transparent', border: `1px solid ${C.border}`, color: C.textSec }}>🔀 Fork</button>
            {isOwner && <button onClick={() => onDelete(recipe.id, recipe.name)} style={{ ...btnBase, background: 'transparent', border: '1px solid #FEE2E2', color: C.danger }}>删除</button>}
          </>}
        </div>
      </div>
    </div>
  )
}

/* ==================== 市场配方卡片（迭代6新增） ==================== */
function MarketCard({ recipe, onFork, onStats, loadingId }: {
  recipe: MarketRecipeItem; onFork: (id: string) => void; onStats: (id: string, name: string) => void; loadingId: string | null
}) {
  const [hovered, setHovered] = useState(false)
  const scoreColor = recipe.avg_score >= 8.5 ? C.success : recipe.avg_score >= 7 ? C.accent : recipe.avg_score >= 5 ? '#F97316' : C.textMuted

  return (
    <div onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)} style={{
      background: C.card, borderRadius: '12px', border: `1px solid ${hovered ? C.borderHover : C.border}`,
      padding: '20px', transition: 'all 200ms ease',
      boxShadow: hovered ? '0 4px 16px rgba(0,0,0,0.08)' : '0 1px 3px rgba(0,0,0,0.04)',
      transform: hovered ? 'translateY(-2px)' : 'none',
    }}>
      {/* 头部 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '12px', marginBottom: '10px' }}>
        <h3 style={{ fontSize: '15px', fontWeight: 600, color: C.text, margin: 0, flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>📦 {recipe.name}</h3>
        {recipe.avg_score > 0 && (
          <span style={{ padding: '3px 10px', borderRadius: '20px', fontSize: '13px', fontWeight: 700, color: scoreColor, background: recipe.avg_score >= 8.5 ? 'rgba(16,185,129,0.08)' : recipe.avg_score >= 7 ? 'rgba(245,158,11,0.08)' : 'rgba(156,163,175,0.08)' }}>
            {recipe.avg_score.toFixed(1)}
          </span>
        )}
      </div>
      {recipe.description && <div style={{ fontSize: '13px', color: C.textSec, marginBottom: '10px', lineHeight: 1.5, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{recipe.description}</div>}
      <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', marginBottom: '12px' }}>
        <MetaTag icon="📚" text={recipe.subject} /><MetaTag icon="🎓" text={recipe.grade_range} /><MetaTag icon="🧩" text={`${recipe.component_count}个组件`} />
      </div>
      {/* 统计行 */}
      <div style={{ display: 'flex', gap: '16px', marginBottom: '12px' }}>
        <span style={{ fontSize: '12px', color: C.textMuted }}>使用 <span style={{ fontWeight: 600, color: C.text }}>{recipe.use_count}</span> 次</span>
        <span style={{ fontSize: '12px', color: C.textMuted }}>Fork <span style={{ fontWeight: 600, color: C.text }}>{recipe.fork_count}</span> 次</span>
        <span style={{ fontSize: '12px', color: C.textMuted }}>教案 <span style={{ fontWeight: 600, color: C.text }}>{recipe.plan_count}</span> 个</span>
      </div>
      {/* 作者+操作 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingTop: '12px', borderTop: `1px solid ${C.border}` }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
          <ScopeBadge scope={recipe.scope} />
          <span style={{ fontSize: '12px', color: C.textMuted }}>by {recipe.author_name}</span>
        </div>
        <div style={{ display: 'flex', gap: '6px' }}>
          <button onClick={() => onStats(recipe.id, recipe.name)} style={{ ...btnBase, background: 'rgba(245,158,11,0.08)', color: C.accent }}>📊</button>
          <button onClick={() => onFork(recipe.id)} disabled={loadingId === recipe.id} style={{ ...btnBase, background: C.primary, color: '#fff' }}>
            {loadingId === recipe.id ? '...' : '🔀 Fork到我的'}
          </button>
        </div>
      </div>
    </div>
  )
}

/* ==================== 骨架屏 ==================== */
function SkeletonCard() {
  const s: React.CSSProperties = { background: 'linear-gradient(90deg, #F3F4F6 25%, #E5E7EB 50%, #F3F4F6 75%)', backgroundSize: '200% 100%', animation: 'shimmer 1.4s infinite', borderRadius: '4px' }
  return (
    <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '20px' }}>
      <style>{`@keyframes shimmer { 0%{background-position:200% 0} 100%{background-position:-200% 0} }`}</style>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '12px' }}><div style={{ ...s, width: '55%', height: '18px' }} /><div style={{ ...s, width: '18%', height: '18px', borderRadius: '20px' }} /></div>
      <div style={{ display: 'flex', gap: '10px', marginBottom: '14px' }}>{[1,2,3].map(i => <div key={i} style={{ ...s, width: '70px', height: '14px' }} />)}</div>
      <div style={{ ...s, width: '100%', height: '1px', marginBottom: '12px' }} />
      <div style={{ display: 'flex', justifyContent: 'space-between' }}><div style={{ ...s, width: '30%', height: '12px' }} /><div style={{ ...s, width: '25%', height: '26px', borderRadius: '6px' }} /></div>
    </div>
  )
}

/* ==================== 主组件 ==================== */
export default function RecipesPage() {
  const { user } = useAuth()
  const navigate = useNavigate()

  // Tab切换
  const [activeTab, setActiveTab] = useState<'mine' | 'market'>('mine')

  // ---- 我的配方状态 ----
  const [recipes, setRecipes] = useState<RecipeListItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [scopeFilter, setScopeFilter] = useState('all')
  const [subjectFilter, setSubjectFilter] = useState('全部')

  // ---- 市场配方状态（迭代6）----
  const [marketRecipes, setMarketRecipes] = useState<MarketRecipeItem[]>([])
  const [marketTotal, setMarketTotal] = useState(0)
  const [marketLoading, setMarketLoading] = useState(false)
  const [marketSubject, setMarketSubject] = useState('全部')
  const [marketSort, setMarketSort] = useState('composite')

  // ---- 共用状态 ----
  const [loadingId, setLoadingId] = useState<string | null>(null)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)
  const [statsModal, setStatsModal] = useState<{ id: string; name: string } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => { setToast({ msg, type }); setTimeout(() => setToast(null), 3000) }

  // ---- 加载我的配方 ----
  const loadRecipes = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, string> = { limit: '100' }
      if (scopeFilter !== 'all') params.scope = scopeFilter
      if (subjectFilter !== '全部') params.subject = subjectFilter
      const resp = await getRecipes(params)
      setRecipes(resp.recipes || []); setTotal(resp.total || 0)
    } catch { showToast('加载失败', 'error') }
    finally { setLoading(false) }
  }, [scopeFilter, subjectFilter])

  // ---- 加载市场配方（迭代6）----
  const loadMarket = useCallback(async () => {
    setMarketLoading(true)
    try {
      const params: Record<string, string | number> = { limit: 50, sort_by: marketSort }
      if (marketSubject !== '全部') params.subject = marketSubject
      const resp = await getMarketRecipes(params)
      setMarketRecipes(resp.recipes || []); setMarketTotal(resp.total || 0)
    } catch { showToast('加载市场失败', 'error') }
    finally { setMarketLoading(false) }
  }, [marketSubject, marketSort])

  useEffect(() => { if (activeTab === 'mine') loadRecipes() }, [activeTab, loadRecipes])
  useEffect(() => { if (activeTab === 'market') loadMarket() }, [activeTab, loadMarket])

  const isFiltered = scopeFilter !== 'all' || subjectFilter !== '全部'
  const handleReset = () => { setScopeFilter('all'); setSubjectFilter('全部') }

  const handleEdit = (id: string) => navigate(`/lesson-plans/recipes/${id}/edit`, { state: { from: '/lesson-plans/recipes' } })
  const handleFork = async (id: string) => {
    if (loadingId) return; setLoadingId(id)
    try { const f = await forkRecipe(id); showToast(`已Fork配方「${f.name}」✓`); if (activeTab === 'mine') await loadRecipes() }
    catch (e: unknown) { showToast(e instanceof Error ? e.message : 'Fork失败', 'error') }
    finally { setLoadingId(null) }
  }
  const handleDelete = async (id: string, name: string) => {
    if (loadingId) return; if (!confirm(`确定删除配方「${name}」？`)) return; setLoadingId(id)
    try { await deleteRecipe(id); showToast('已删除'); await loadRecipes() }
    catch (e: unknown) { showToast(e instanceof Error ? e.message : '删除失败', 'error') }
    finally { setLoadingId(null) }
  }

  // Tab样式
  const tabStyle = (active: boolean): React.CSSProperties => ({
    padding: '10px 20px', fontSize: '14px', fontWeight: active ? 600 : 400, cursor: 'pointer',
    color: active ? C.primary : C.textSec, borderBottom: active ? `2px solid ${C.primary}` : '2px solid transparent',
    background: 'none', border: 'none', transition: 'all 150ms ease',
  })

  return (
    <div>
      {/* 顶部：Tab切换 + 新建按钮 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}` }}>
          <button onClick={() => setActiveTab('mine')} style={tabStyle(activeTab === 'mine')}>📦 我的配方</button>
          <button onClick={() => setActiveTab('market')} style={tabStyle(activeTab === 'market')}>🏪 配方市场</button>
        </div>
        <button onClick={() => navigate('/lesson-plans/recipes/wizard')} style={{
          display: 'flex', alignItems: 'center', gap: '6px', padding: '9px 18px', borderRadius: '8px',
          border: 'none', background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer',
        }}><span>📦</span><span>新建配方</span></button>
      </div>

      {/* ======== 我的配方 Tab ======== */}
      {activeTab === 'mine' && (
        <>
          <div style={{ marginBottom: '16px', fontSize: '14px', color: C.textSec }}>共 {total} 个配方{isFiltered ? `（筛选后 ${recipes.length} 个）` : ''}</div>
          {/* 筛选栏 */}
          <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '16px 20px', marginBottom: '20px' }}>
            <div style={{ marginBottom: '12px' }}>
              <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec, marginRight: '12px' }}>范围</span>
              <div style={{ display: 'inline-flex', flexWrap: 'wrap', gap: '6px' }}>
                {SCOPE_FILTERS.map(f => (
                  <button key={f.key} onClick={() => setScopeFilter(f.key)} style={{
                    padding: '5px 12px', borderRadius: '20px', border: `1px solid ${scopeFilter === f.key ? C.primary : C.border}`,
                    background: scopeFilter === f.key ? C.primaryLight : 'transparent',
                    color: scopeFilter === f.key ? C.primary : C.textSec, fontSize: '13px', fontWeight: scopeFilter === f.key ? 600 : 400, cursor: 'pointer',
                  }}>{f.label}</button>
                ))}
              </div>
            </div>
            <div style={{ display: 'flex', gap: '16px', alignItems: 'center' }}>
              <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec }}>学科</span>
              <select value={subjectFilter} onChange={e => setSubjectFilter(e.target.value)} style={{
                padding: '5px 10px', borderRadius: '6px', border: `1px solid ${subjectFilter !== '全部' ? C.primary : C.border}`,
                background: subjectFilter !== '全部' ? C.primaryLight : 'transparent', color: subjectFilter !== '全部' ? C.primary : C.textSec, fontSize: '13px', cursor: 'pointer', outline: 'none',
              }}>{SUBJECTS.map(o => <option key={o} value={o}>{o}</option>)}</select>
              {isFiltered && <button onClick={handleReset} style={{ fontSize: '12px', color: C.textMuted, cursor: 'pointer', background: 'none', border: 'none', textDecoration: 'underline' }}>清空筛选</button>}
            </div>
          </div>
          {/* 卡片网格 */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))', gap: '16px' }}>
            {loading && Array.from({ length: 6 }).map((_, i) => <SkeletonCard key={i} />)}
            {!loading && recipes.map(r => (
              <RecipeCard key={r.id} recipe={r} isOwner={user?.id === r.author_id} onEdit={handleEdit} onFork={handleFork} onDelete={handleDelete} onStats={(id, name) => setStatsModal({ id, name })} loadingId={loadingId} />
            ))}
            {!loading && recipes.length === 0 && (
              <div style={{ gridColumn: '1 / -1', textAlign: 'center', padding: '80px 40px', background: C.card, borderRadius: '12px', border: `1px solid ${C.border}` }}>
                <div style={{ fontSize: '48px', marginBottom: '16px' }}>{isFiltered ? '🔍' : '📦'}</div>
                <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>{isFiltered ? '没有符合条件的配方' : '还没有备课配方'}</div>
                <div style={{ fontSize: '14px', color: C.textMuted, marginBottom: '24px' }}>{isFiltered ? '试试调整筛选条件' : '创建您的第一个备课配方'}</div>
                {isFiltered ? <button onClick={handleReset} style={{ padding: '10px 24px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>清空筛选</button>
                  : <button onClick={() => navigate('/lesson-plans/recipes/wizard')} style={{ padding: '10px 24px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>📦 创建配方</button>}
              </div>
            )}
          </div>
        </>
      )}

      {/* ======== 配方市场 Tab（迭代6新增）======== */}
      {activeTab === 'market' && (
        <>
          <div style={{ marginBottom: '16px', fontSize: '14px', color: C.textSec }}>共 {marketTotal} 个共享配方</div>
          {/* 市场筛选栏 */}
          <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '16px 20px', marginBottom: '20px', display: 'flex', gap: '20px', alignItems: 'center', flexWrap: 'wrap' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec }}>学科</span>
              <select value={marketSubject} onChange={e => setMarketSubject(e.target.value)} style={{
                padding: '5px 10px', borderRadius: '6px', border: `1px solid ${marketSubject !== '全部' ? C.primary : C.border}`,
                background: marketSubject !== '全部' ? C.primaryLight : 'transparent', color: marketSubject !== '全部' ? C.primary : C.textSec, fontSize: '13px', cursor: 'pointer', outline: 'none',
              }}>{SUBJECTS.map(o => <option key={o} value={o}>{o}</option>)}</select>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec }}>排序</span>
              <div style={{ display: 'flex', gap: '4px' }}>
                {MARKET_SORTS.map(s => (
                  <button key={s.key} onClick={() => setMarketSort(s.key)} style={{
                    padding: '4px 10px', borderRadius: '6px', fontSize: '12px', cursor: 'pointer',
                    border: `1px solid ${marketSort === s.key ? C.primary : C.border}`,
                    background: marketSort === s.key ? C.primaryLight : 'transparent',
                    color: marketSort === s.key ? C.primary : C.textSec, fontWeight: marketSort === s.key ? 600 : 400,
                  }}>{s.label}</button>
                ))}
              </div>
            </div>
          </div>
          {/* 市场卡片 */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))', gap: '16px' }}>
            {marketLoading && Array.from({ length: 6 }).map((_, i) => <SkeletonCard key={i} />)}
            {!marketLoading && marketRecipes.map(r => (
              <MarketCard key={r.id} recipe={r} onFork={handleFork} onStats={(id, name) => setStatsModal({ id, name })} loadingId={loadingId} />
            ))}
            {!marketLoading && marketRecipes.length === 0 && (
              <div style={{ gridColumn: '1 / -1', textAlign: 'center', padding: '80px 40px', background: C.card, borderRadius: '12px', border: `1px solid ${C.border}` }}>
                <div style={{ fontSize: '48px', marginBottom: '16px' }}>🏪</div>
                <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>配方市场暂无共享配方</div>
                <div style={{ fontSize: '14px', color: C.textMuted }}>当有老师共享配方到教研组或学校后，这里会显示排行榜</div>
              </div>
            )}
          </div>
        </>
      )}

      {/* 统计弹窗（迭代6） */}
      {statsModal && <RecipeStatsModal recipeId={statsModal.id} recipeName={statsModal.name} onClose={() => setStatsModal(null)} />}

      {/* Toast */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)',
          padding: '12px 24px', borderRadius: '10px',
          background: toast.type === 'error' ? '#FEF2F2' : '#1F2937',
          color: toast.type === 'error' ? C.danger : '#fff',
          fontSize: '14px', fontWeight: 500, boxShadow: '0 8px 24px rgba(0,0,0,0.15)',
          zIndex: 9999, whiteSpace: 'nowrap', border: toast.type === 'error' ? '1px solid #FECACA' : 'none',
          animation: 'toast-in 200ms ease',
        }}>
          <style>{`@keyframes toast-in { from{opacity:0;transform:translateX(-50%) translateY(8px)} to{opacity:1;transform:translateX(-50%) translateY(0)} }`}</style>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}

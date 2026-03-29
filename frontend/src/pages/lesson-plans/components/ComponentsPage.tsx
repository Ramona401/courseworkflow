/**
 * 组件管理页面 — ComponentsPage
 *
 * Phase5完整实现：
 *   - 顶部Tab：组件概览 / 萃取待审队列
 *   - 组件概览Tab：13类组件统计卡片 + 实时数量（接入API）
 *   - 萃取待审Tab：待审列表 + 确认/拒绝操作（教研组长/骨干使用）
 *
 * PRD §4.1：13类组件与三种注入模式
 * PRD §6：自成长机制——通道一/二萃取确认
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getComponents, getExtractions, confirmExtraction,
  type ComponentListItem, type ExtractionListItem,
} from '../../../api/lesson-plans'

// ==================== 常量定义 ====================

/** 13类组件元数据（对应后端 library_type 枚举） */
const COMPONENT_TYPES = [
  { key: 'curriculum_standard',  icon: '📖', name: '课标与能力框架库', mode: 'silent' },
  { key: 'knowledge_graph',      icon: '🧠', name: '知识图谱库',       mode: 'silent' },
  { key: 'student_profile',      icon: '👤', name: '学情特征库',       mode: 'silent' },
  { key: 'pedagogy',             icon: '🎓', name: '教学法库',         mode: 'recommend' },
  { key: 'assessment_strategy',  icon: '📊', name: '评估策略库',       mode: 'recommend' },
  { key: 'activity_design',      icon: '🎯', name: '活动设计方案库',   mode: 'recommend' },
  { key: 'questioning_strategy', icon: '❓', name: '提问引导策略库',   mode: 'on_demand' },
  { key: 'cross_subject',        icon: '🔗', name: '跨学科连接库',     mode: 'on_demand' },
  { key: 'teaching_tool',        icon: '🛠️', name: '教学工具库',       mode: 'on_demand' },
  { key: 'scenario_material',    icon: '🎬', name: '素材情境库',       mode: 'on_demand' },
  { key: 'quality_rubric',       icon: '✅', name: '质量评估标准库',   mode: 'silent' },
  { key: 'design_defect',        icon: '⚠️', name: '常见设计缺陷库',   mode: 'silent' },
  { key: 'review_rubric',        icon: '📋', name: '教案评审规则库',   mode: 'ai_only' },
] as const

/** 注入模式中文映射 */
const MODE_LABEL: Record<string, string> = {
  silent:     '静默注入',
  recommend:  '推荐确认',
  on_demand:  '按需调用',
  ai_only:    'AI内部',
}

/** 注入模式颜色 */
const MODE_COLOR: Record<string, { bg: string; fg: string }> = {
  silent:    { bg: 'rgba(79,123,232,0.08)',  fg: '#4F7BE8' },
  recommend: { bg: 'rgba(245,158,11,0.08)',  fg: '#F59E0B' },
  on_demand: { bg: 'rgba(16,185,129,0.08)',  fg: '#10B981' },
  ai_only:   { bg: 'rgba(156,163,175,0.08)', fg: '#9CA3AF' },
}

/** 来源类型中文映射 */
const SOURCE_LABEL: Record<string, string> = {
  conversation: '💬 对话萃取',
  lesson_plan:  '📄 评审萃取',
  manual:       '✍️ 手动入库',
}

// ==================== 子组件：骨架屏 ====================

function SkeletonCard() {
  return (
    <div style={{
      background: '#FFFFFF', borderRadius: '12px', padding: '20px',
      border: '1px solid #F3F4F6',
    }}>
      <div style={{ display: 'flex', gap: '12px', alignItems: 'center', marginBottom: '10px' }}>
        <div style={{ width: 36, height: 36, borderRadius: '8px', background: '#F3F4F6', animation: 'shimmer 1.5s infinite' }} />
        <div style={{ flex: 1 }}>
          <div style={{ height: 14, borderRadius: 4, background: '#F3F4F6', marginBottom: 6, animation: 'shimmer 1.5s infinite' }} />
          <div style={{ height: 10, borderRadius: 4, background: '#F3F4F6', width: '60%', animation: 'shimmer 1.5s infinite' }} />
        </div>
      </div>
      <div style={{ height: 10, borderRadius: 4, background: '#F3F4F6', width: '40%', animation: 'shimmer 1.5s infinite' }} />
    </div>
  )
}

// ==================== 子组件：Toast ====================

interface ToastProps { msg: string; type: 'success' | 'error' }

function Toast({ msg, type }: ToastProps) {
  return (
    <div style={{
      position: 'fixed', bottom: 32, left: '50%', transform: 'translateX(-50%)',
      background: type === 'success' ? '#1F2937' : '#DC2626',
      color: '#FFF', padding: '10px 24px', borderRadius: '8px',
      fontSize: '14px', fontWeight: 500, zIndex: 9999,
      boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
    }}>{msg}</div>
  )
}

// ==================== 主页面 ====================

export default function ComponentsPage() {
  // Tab状态：overview=组件概览 / extractions=萃取待审
  const [activeTab, setActiveTab] = useState<'overview' | 'extractions'>('overview')

  // 组件概览数据（按library_type统计数量）
  const [countMap, setCountMap]     = useState<Record<string, number>>({})
  const [overviewLoading, setOverviewLoading] = useState(true)

  // 萃取待审数据
  const [extractions, setExtractions]           = useState<ExtractionListItem[]>([])
  const [extractionLoading, setExtractionLoading] = useState(false)
  const [confirmingId, setConfirmingId]         = useState<string | null>(null)

  // Toast
  const [toast, setToast] = useState<ToastProps | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  // ---- 加载组件概览：查询全量组件列表，统计各类型数量 ----
  const loadOverview = useCallback(async () => {
    setOverviewLoading(true)
    try {
      const resp = await getComponents({ limit: 500 })
      const map: Record<string, number> = {}
      // resp.components 是 ComponentListItem[]
      const list: ComponentListItem[] = (resp as unknown as { components: ComponentListItem[] }).components || []
      list.forEach(c => {
        map[c.library_type] = (map[c.library_type] || 0) + 1
      })
      setCountMap(map)
    } catch (e) {
      console.error('加载组件概览失败', e)
    } finally {
      setOverviewLoading(false)
    }
  }, [])

  // ---- 加载萃取待审列表 ----
  const loadExtractions = useCallback(async () => {
    setExtractionLoading(true)
    try {
      const resp = await getExtractions({ limit: 100 })
      setExtractions(resp.extractions || [])
    } catch (e) {
      console.error('加载萃取列表失败', e)
    } finally {
      setExtractionLoading(false)
    }
  }, [])

  // 初始加载
  useEffect(() => { loadOverview() }, [loadOverview])

  // 切换到萃取Tab时加载
  useEffect(() => {
    if (activeTab === 'extractions') loadExtractions()
  }, [activeTab, loadExtractions])

  // ---- 确认/拒绝萃取 ----
  const handleConfirm = async (id: string, decision: 'confirmed' | 'rejected') => {
    setConfirmingId(id)
    try {
      await confirmExtraction(id, decision)
      showToast(decision === 'confirmed' ? '✅ 已确认入库' : '已拒绝')
      // 从列表移除
      setExtractions(prev => prev.filter(e => e.id !== id))
    } catch (e) {
      showToast('操作失败，请重试', 'error')
      console.error(e)
    } finally {
      setConfirmingId(null)
    }
  }

  // ==================== 渲染：Tab栏 ====================
  const pendingCount = extractions.filter(e => e.status === 'pending').length

  return (
    <div>
      {/* shimmer动画 */}
      <style>{`
        @keyframes shimmer {
          0%   { opacity: 1 }
          50%  { opacity: 0.4 }
          100% { opacity: 1 }
        }
      `}</style>

      {/* 页面标题 */}
      <div style={{ marginBottom: '24px' }}>
        <h1 style={{ fontSize: '20px', fontWeight: 600, color: '#1F2937', margin: '0 0 8px 0' }}>
          组件管理
        </h1>
        <p style={{ fontSize: '14px', color: '#6B7280', margin: 0 }}>
          管理13类教学设计组件，支持匹配引擎智能推荐，好的设计片段自动沉淀进库
        </p>
      </div>

      {/* Tab切换 */}
      <div style={{ display: 'flex', gap: '4px', marginBottom: '24px', borderBottom: '1px solid #E5E7EB' }}>
        {([
          { key: 'overview',    label: '🧩 组件概览' },
          { key: 'extractions', label: `💡 萃取待审${pendingCount > 0 ? ` (${pendingCount})` : ''}` },
        ] as const).map(tab => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            style={{
              padding: '10px 20px',
              border: 'none',
              background: 'transparent',
              fontSize: '14px',
              fontWeight: activeTab === tab.key ? 600 : 400,
              color: activeTab === tab.key ? '#4F7BE8' : '#6B7280',
              borderBottom: activeTab === tab.key ? '2px solid #4F7BE8' : '2px solid transparent',
              cursor: 'pointer',
              marginBottom: '-1px',
              transition: 'all 150ms ease',
            }}
          >{tab.label}</button>
        ))}
      </div>

      {/* ==================== Tab内容：组件概览 ==================== */}
      {activeTab === 'overview' && (
        <div>
          {/* 统计摘要 */}
          <div style={{
            display: 'flex', gap: '12px', marginBottom: '20px',
            padding: '16px 20px', background: '#F0F4FF',
            borderRadius: '12px', alignItems: 'center',
          }}>
            <span style={{ fontSize: '20px' }}>📊</span>
            <div>
              <div style={{ fontSize: '14px', fontWeight: 600, color: '#1F2937' }}>
                组件库总计：{Object.values(countMap).reduce((a, b) => a + b, 0)} 个组件
              </div>
              <div style={{ fontSize: '12px', color: '#6B7280', marginTop: '2px' }}>
                涵盖13个类别，按学科和学段匹配推荐
              </div>
            </div>
          </div>

          {/* 13类组件网格 */}
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))',
            gap: '16px',
          }}>
            {overviewLoading
              ? Array.from({ length: 13 }).map((_, i) => <SkeletonCard key={i} />)
              : COMPONENT_TYPES.map(ct => {
                  const mc = MODE_COLOR[ct.mode]
                  const count = countMap[ct.key] || 0
                  return (
                    <div key={ct.key} style={{
                      background: '#FFFFFF',
                      borderRadius: '12px',
                      padding: '20px',
                      border: '1px solid #F3F4F6',
                      transition: 'all 200ms ease',
                    }}>
                      <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px', marginBottom: '12px' }}>
                        <span style={{ fontSize: '24px', lineHeight: 1 }}>{ct.icon}</span>
                        <div style={{ flex: 1 }}>
                          <div style={{ fontSize: '14px', fontWeight: 600, color: '#1F2937', marginBottom: '6px' }}>
                            {ct.name}
                          </div>
                          <span style={{
                            fontSize: '10px', fontWeight: 600,
                            color: mc.fg, background: mc.bg,
                            padding: '2px 8px', borderRadius: '6px',
                            display: 'inline-block',
                          }}>{MODE_LABEL[ct.mode]}</span>
                        </div>
                      </div>
                      <div style={{ fontSize: '12px', color: '#9CA3AF' }}>
                        组件数量：
                        <span style={{ fontWeight: 700, color: count > 0 ? '#4F7BE8' : '#9CA3AF' }}>
                          {count}
                        </span>
                      </div>
                    </div>
                  )
                })
            }
          </div>
        </div>
      )}

      {/* ==================== Tab内容：萃取待审 ==================== */}
      {activeTab === 'extractions' && (
        <div>
          {/* 说明卡片 */}
          <div style={{
            background: '#FFFBEB', border: '1px solid #FDE68A',
            borderRadius: '12px', padding: '16px 20px',
            marginBottom: '20px', display: 'flex', gap: '12px', alignItems: 'flex-start',
          }}>
            <span style={{ fontSize: '20px', flexShrink: 0 }}>💡</span>
            <div>
              <div style={{ fontSize: '14px', fontWeight: 600, color: '#92400E', marginBottom: '4px' }}>
                关于萃取待审
              </div>
              <div style={{ fontSize: '13px', color: '#78350F', lineHeight: 1.6 }}>
                系统会从优秀对话和评审通过的高分教案中自动识别可复用的教学设计片段。
                教研组长和骨干教师确认后，这些片段将进入组件库，被匹配引擎推荐给其他老师。
              </div>
            </div>
          </div>

          {/* 加载中 */}
          {extractionLoading && (
            <div style={{ textAlign: 'center', padding: '40px', color: '#9CA3AF', fontSize: '14px' }}>
              加载中...
            </div>
          )}

          {/* 空状态 */}
          {!extractionLoading && extractions.length === 0 && (
            <div style={{
              textAlign: 'center', padding: '60px 20px',
              background: '#FFFFFF', borderRadius: '12px', border: '1px solid #F3F4F6',
            }}>
              <div style={{ fontSize: '40px', marginBottom: '12px' }}>🌱</div>
              <div style={{ fontSize: '16px', fontWeight: 600, color: '#1F2937', marginBottom: '8px' }}>
                暂无待审萃取记录
              </div>
              <div style={{ fontSize: '14px', color: '#6B7280' }}>
                当老师在备课中产生好的设计片段，或有高分教案评审通过时，系统会自动识别并在这里等待您确认
              </div>
            </div>
          )}

          {/* 萃取列表 */}
          {!extractionLoading && extractions.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              {extractions.map(item => (
                <ExtractionCard
                  key={item.id}
                  item={item}
                  isConfirming={confirmingId === item.id}
                  onConfirm={(decision) => handleConfirm(item.id, decision)}
                />
              ))}
            </div>
          )}
        </div>
      )}

      {/* Toast通知 */}
      {toast && <Toast msg={toast.msg} type={toast.type} />}
    </div>
  )
}

// ==================== 子组件：萃取卡片 ====================

interface ExtractionCardProps {
  item: ExtractionListItem
  isConfirming: boolean
  onConfirm: (decision: 'confirmed' | 'rejected') => void
}

function ExtractionCard({ item, isConfirming, onConfirm }: ExtractionCardProps) {
  const [expanded, setExpanded] = useState(false)

  // 状态徽标
  const statusBadge = item.status === 'pending'
    ? { label: '待审核', bg: '#FEF3C7', fg: '#92400E' }
    : item.status === 'confirmed'
    ? { label: '已确认', bg: '#D1FAE5', fg: '#065F46' }
    : { label: '已拒绝', bg: '#FEE2E2', fg: '#991B1B' }

  return (
    <div style={{
      background: '#FFFFFF', borderRadius: '12px',
      border: '1px solid #E5E7EB',
      overflow: 'hidden',
      opacity: isConfirming ? 0.7 : 1,
      transition: 'opacity 200ms ease',
    }}>
      {/* 卡片头部 */}
      <div style={{ padding: '16px 20px', borderBottom: expanded ? '1px solid #F3F4F6' : 'none' }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px' }}>
          {/* 来源标签 */}
          <div style={{
            flexShrink: 0, fontSize: '12px', fontWeight: 600,
            color: '#4F7BE8', background: 'rgba(79,123,232,0.08)',
            padding: '3px 10px', borderRadius: '6px', marginTop: '2px',
          }}>
            {SOURCE_LABEL[item.source_type] || item.source_type}
          </div>

          <div style={{ flex: 1 }}>
            {/* 组件库类型 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
              <span style={{
                fontSize: '12px', color: '#F59E0B',
                background: 'rgba(245,158,11,0.08)',
                padding: '2px 8px', borderRadius: '6px', fontWeight: 600,
              }}>
                {item.library_name || item.extraction_type}
              </span>
              <span style={{
                fontSize: '11px', fontWeight: 600,
                color: statusBadge.fg, background: statusBadge.bg,
                padding: '2px 8px', borderRadius: '6px',
              }}>
                {statusBadge.label}
              </span>
            </div>

            {/* 原始内容摘要（最多两行） */}
            <div style={{
              fontSize: '14px', color: '#1F2937', lineHeight: 1.6,
              display: '-webkit-box', WebkitLineClamp: expanded ? 'none' : 2,
              WebkitBoxOrient: 'vertical', overflow: 'hidden',
            }}>
              {item.source_content}
            </div>
          </div>

          {/* 展开/收起 */}
          <button
            onClick={() => setExpanded(!expanded)}
            style={{
              flexShrink: 0, fontSize: '12px', color: '#6B7280',
              background: 'transparent', border: 'none', cursor: 'pointer',
              padding: '4px 8px', borderRadius: '6px',
              transition: 'background 150ms ease',
            }}
          >{expanded ? '收起 ▲' : '展开 ▼'}</button>
        </div>

        {/* 展开：详细信息 */}
        {expanded && (
          <div style={{ marginTop: '12px', paddingTop: '12px', borderTop: '1px solid #F3F4F6' }}>
            <div style={{ fontSize: '12px', color: '#6B7280', marginBottom: '4px', fontWeight: 600 }}>
              完整内容片段：
            </div>
            <div style={{
              fontSize: '13px', color: '#374151', lineHeight: 1.7,
              background: '#F9FAFB', borderRadius: '8px', padding: '12px',
            }}>
              {item.source_content}
            </div>
          </div>
        )}
      </div>

      {/* 卡片底部：元信息 + 操作按钮 */}
      <div style={{
        padding: '12px 20px', background: '#FAFAFA',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px',
      }}>
        {/* 元信息 */}
        <div style={{ fontSize: '12px', color: '#9CA3AF', display: 'flex', gap: '12px', flexWrap: 'wrap' }}>
          {item.plan_title && (
            <span>📄 {item.plan_title}</span>
          )}
          {item.created_by_name && (
            <span>👤 {item.created_by_name}</span>
          )}
          <span>🕐 {new Date(item.created_at).toLocaleDateString('zh-CN')}</span>
        </div>

        {/* 操作按钮（仅pending状态显示） */}
        {item.status === 'pending' && (
          <div style={{ display: 'flex', gap: '8px', flexShrink: 0 }}>
            <button
              disabled={isConfirming}
              onClick={() => onConfirm('rejected')}
              style={{
                padding: '6px 16px', borderRadius: '8px',
                border: '1px solid #E5E7EB', background: '#FFFFFF',
                color: '#6B7280', fontSize: '13px', fontWeight: 500,
                cursor: isConfirming ? 'not-allowed' : 'pointer',
                transition: 'all 150ms ease',
              }}
            >
              {isConfirming ? '处理中...' : '不入库'}
            </button>
            <button
              disabled={isConfirming}
              onClick={() => onConfirm('confirmed')}
              style={{
                padding: '6px 16px', borderRadius: '8px',
                border: 'none', background: '#4F7BE8',
                color: '#FFFFFF', fontSize: '13px', fontWeight: 600,
                cursor: isConfirming ? 'not-allowed' : 'pointer',
                transition: 'all 150ms ease',
              }}
            >
              {isConfirming ? '处理中...' : '✓ 入库'}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

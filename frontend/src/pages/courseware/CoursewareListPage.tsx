/**
 * 课件工坊列表页 — CoursewareListPage v6.1（接入阶段3提交审核）
 *
 * v6.0：共享课件库独立成栏后，本页回归纯「我的课件」管理页。
 *
 * v6.1 变更（阶段3 课件多级审核）：
 *   - 给 CWCard 接上 onSubmitReview 回调（CWCard 新增了该必填 prop）。
 *   - handleSubmitReview：window.confirm 二次确认 → 调 submitCoursewareForReview →
 *     成功 alert 提示并刷新列表（课件 publish_state 变 submitted 后，提交审核按钮自然隐藏）。
 *   - 提交审核按钮的显隐由 CWCard 内部按 status≥preview 且 publish_state∈{private,published_personal,revision} 判定，
 *     本页只负责执行提交动作。
 *
 * 本页保留：
 *   1) 顶部标题 + 新建按钮
 *   2) 状态筛选 + 列表渲染（CWCard）+ 删除二次确认 + 发布面板挂载 + 提交审核
 *   3) 新建课件弹窗挂载（CreateCoursewareModal，创建成功后跳新课件工坊）
 *
 * 子组件仍在 components/courseware-list/ 子目录：
 *   - listConstants.ts          共享常量
 *   - CWCard.tsx                我的课件卡片
 *   - PublishPanel.tsx          发布面板弹窗（设发布态 + 代码开放范围）
 *   - CreateCoursewareModal.tsx 新建课件弹窗（五入口，自管状态）
 *   - SharedCWCard.tsx          共享课件库卡片（现仅 SharedCoursewareLibraryPage 使用）
 */

import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { getCoursewares, deleteCourseware, submitCoursewareForReview } from '@/api/coursewares'
import type { CoursewareListItem } from '@/api/coursewares'
import { C, btnBase } from './components/courseware-list/listConstants'
import CWCard from './components/courseware-list/CWCard'
import PublishPanel from './components/courseware-list/PublishPanel'
import CreateCoursewareModal from './components/courseware-list/CreateCoursewareModal'
import JoinedCollabSection from './components/courseware-list/JoinedCollabSection'

export default function CoursewareListPage() {
  const navigate = useNavigate()

  // ── 我的课件 ──
  const [items, setItems] = useState<CoursewareListItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState('')

  // ── 发布面板弹窗 ──（点某张课件卡片的"发布/分享"按钮后打开）
  const [publishTarget, setPublishTarget] = useState<CoursewareListItem | null>(null)

  // ── 新建课件弹窗 ──
  const [showCreate, setShowCreate] = useState(false)

  // ── 提交审核进行中标记（防重复点击）──
  const [submittingId, setSubmittingId] = useState<string | null>(null)

  // 状态筛选变化时重新加载
  useEffect(() => { loadData() }, [statusFilter])

  const loadData = async () => {
    setLoading(true)
    try {
      const resp = await getCoursewares({ status: statusFilter || undefined, limit: 50 })
      setItems(resp.coursewares || []); setTotal(resp.total)
    } catch { /* */ } finally { setLoading(false) }
  }

  // 删除课件:window.confirm 二次确认后调后端删除接口,成功后刷新列表
  const handleDelete = async (id: string, title: string) => {
    if (!window.confirm('确定删除课件「' + title + '」?移入回收站后30天内可恢复，过期将自动永久删除。')) return
    try { await deleteCourseware(id); loadData() } catch { alert('删除失败') }
  }

  // 提交审核（阶段3）：二次确认 → 调后端 → 成功提示并刷新列表
  //   后端将课件置 publish_state=submitted、review_level=0，并反查作者学校写 review_school_id。
  //   刷新后该课件进入"审核中"，提交审核按钮自然隐藏。
  const handleSubmitReview = async (id: string, title: string) => {
    if (submittingId) return
    if (!window.confirm('确定提交课件「' + title + '」进入审核流程?\n提交后将由教研组（及学校）审核，期间请勿重复提交。')) return
    setSubmittingId(id)
    try {
      await submitCoursewareForReview(id)
      alert('已提交审核，可在「审核中心」跟踪审核进度')
      loadData()
    } catch (err) {
      alert('提交审核失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally {
      setSubmittingId(null)
    }
  }

  const statusFilters = [
    { value: '', label: '全部' }, { value: 'draft', label: '草稿' },
    { value: 'generating', label: '生成中' }, { value: 'preview', label: '预览中' },
    { value: 'confirmed', label: '已确认' }, { value: 'in_pipeline', label: '审核中' },
  ]

  return (
    <div>
      {/* ==================== 顶部标题 + 新建按钮 ==================== */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <div>
          <div style={{ fontSize: '22px', fontWeight: 700, color: C.textPrimary, marginBottom: '4px' }}>📂 我的课件</div>
          <div style={{ fontSize: '13px', color: C.textMuted }}>管理我创建的全部课件</div>
        </div>
        <button onClick={() => setShowCreate(true)} style={{
          ...btnBase, border: 'none',
          background: 'linear-gradient(135deg, #F59E0B, #EF4444)', color: '#fff',
          fontWeight: 600, boxShadow: '0 2px 8px rgba(245,158,11,0.3)',
        }}>+ 新建课件</button>
      </div>

      {/* ==================== 我参与的集体备课（参与者入口，无则不显示） ==================== */}
      <JoinedCollabSection />

      {/* ==================== 状态筛选栏 ==================== */}
      <div style={{ display: 'flex', gap: '8px', marginBottom: '16px' }}>
        {statusFilters.map(f => (
          <button key={f.value} onClick={() => setStatusFilter(f.value)} style={{
            padding: '6px 16px', borderRadius: '20px',
            border: `1px solid ${statusFilter === f.value ? C.primary : C.border}`,
            background: statusFilter === f.value ? C.primaryBg : 'transparent',
            color: statusFilter === f.value ? C.primary : C.textSecondary,
            fontSize: '13px', fontWeight: statusFilter === f.value ? 600 : 400, cursor: 'pointer',
          }}>{f.label}</button>
        ))}
      </div>

      <div style={{ fontSize: '13px', color: C.textMuted, marginBottom: '16px' }}>共 {total} 套课件</div>

      {/* ==================== 列表 ==================== */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '60px 0', color: C.textMuted }}>加载中...</div>
      ) : items.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '80px 0' }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>🎨</div>
          <div style={{ fontSize: '16px', color: C.textSecondary, marginBottom: '8px' }}>还没有课件</div>
          <div style={{ fontSize: '13px', color: C.textMuted }}>点击"新建课件",选择从教案、主题或PPT开始创建</div>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: '16px' }}>
          {items.map(item => (
            <CWCard
              key={item.id}
              item={item}
              onDelete={handleDelete}
              onClick={() => navigate('/courseware/' + item.id)}
              onPublish={() => setPublishTarget(item)}
              onSubmitReview={handleSubmitReview}
            />
          ))}
        </div>
      )}

      {/* ==================== 发布面板弹窗 ==================== */}
      {publishTarget && (
        <PublishPanel
          item={publishTarget}
          onClose={() => setPublishTarget(null)}
          onChanged={() => { setPublishTarget(null); loadData() }}
        />
      )}

      {/* ==================== 新建课件弹窗 ==================== */}
      <CreateCoursewareModal
        open={showCreate}
        onClose={() => setShowCreate(false)}
        onCreated={(coursewareId: string) => {
          setShowCreate(false)
          navigate('/courseware/' + coursewareId)
        }}
      />
    </div>
  )
}

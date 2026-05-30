/**
 * 课件工坊列表页 — CoursewareListPage v4.0（v0.42 PPT入口激活）
 *
 * v4.0 变更：
 *   - PPT入口从disabled占位变为可用（文件上传+学科+年级）
 *   - 上传.pptx文件后调createCoursewareFromPPT → 跳转课件工坊
 */
import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  getCoursewares, deleteCourseware, createCourseware, createCoursewareFromTopic,
  createCoursewareFromPPT,
  createCoursewareFromDoc,
  createCoursewareFrom3D,
  CW_STATUS_CONFIG,
  downloadCoursewareBundle,
} from '@/api/coursewares'
import type { CoursewareListItem } from '@/api/coursewares'
import apiClient from '@/api/client'

// ==================== 常量 ====================
const C = {
  primary: '#F59E0B', primaryBg: 'rgba(245,158,11,0.08)',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
  border: '#E5E7EB',
}

// 课件来源类型配色
const SOURCE_CONFIG: Record<string, { label: string; color: string; bg: string; emoji: string }> = {
  lesson_plan:  { label: '教案生成', color: '#2563EB', bg: '#DBEAFE', emoji: '📝' },
  topic_direct: { label: '主题创建', color: '#7C3AED', bg: '#EDE9FE', emoji: '💡' },
  ppt_upload:   { label: 'PPT上传', color: '#D97706', bg: '#FEF3C7', emoji: '📊' },
  doc_upload:   { label: '文档上传', color: '#0891B2', bg: '#CFFAFE', emoji: '📄' },
  html_import:  { label: 'HTML导入', color: '#059669', bg: '#D1FAE5', emoji: '🌐' },
  '3d_single':  { label: '3D互动', color: '#DC2626', bg: '#FEE2E2', emoji: '🎮' },
}

// 学科列表（与备课工坊一致）
const SUBJECTS = [
  '语文', '数学', '英语', '物理', '化学', '生物', '历史', '地理', '政治',
  '信息科技', '人工智能', '科学', '音乐', '美术', '体育',
]

interface LPItem { id: string; title: string; subject: string; grade: string; status: string }

// ==================== 主组件 ====================
export default function CoursewareListPage() {
  const navigate = useNavigate()
  const [items, setItems] = useState<CoursewareListItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState('')

  // 弹窗状态
  const [showCreate, setShowCreate] = useState(false)
  const [createMode, setCreateMode] = useState<'select' | 'lesson_plan' | 'topic' | 'ppt' | 'doc' | '3d_single'>('select')

  // 从教案创建相关
  const [plans, setPlans] = useState<LPItem[]>([])
  const [plansLoading, setPlansLoading] = useState(false)
  const [selectedPlanId, setSelectedPlanId] = useState('')

  // 从主题创建相关
  const [topicSubject, setTopicSubject] = useState('')
  const [topicGrade, setTopicGrade] = useState('')
  const [topicName, setTopicName] = useState('')
  const [topicNotes, setTopicNotes] = useState('')

  // v0.42 入口B: 从PPT创建相关
  const [pptFile, setPptFile] = useState<File | null>(null)
  const [pptSubject, setPptSubject] = useState('')
  const [pptGrade, setPptGrade] = useState('')
  const [pptTitle, setPptTitle] = useState('')
  const pptFileRef = useRef<HTMLInputElement>(null)

  // v0.42 入口C: 从Word文档创建相关
  const [docFile, setDocFile] = useState<File | null>(null)
  const [docSubject, setDocSubject] = useState('')
  const [docGrade, setDocGrade] = useState('')
  const [docTitle, setDocTitle] = useState('')
  const docFileRef = useRef<HTMLInputElement>(null)

  // v0.42.11: 3D互动单页创建相关
  const [threeDSubject, setThreeDSubject] = useState('')
  const [threeDGrade, setThreeDGrade] = useState('')
  const [threeDTopic, setThreeDTopic] = useState('')
  const [threeDDesc, setThreeDDesc] = useState('')

  const [creating, setCreating] = useState(false)

  useEffect(() => { loadData() }, [statusFilter])

  const loadData = async () => {
    setLoading(true)
    try {
      const resp = await getCoursewares({ status: statusFilter || undefined, limit: 50 })
      setItems(resp.coursewares || []); setTotal(resp.total)
    } catch { /* */ } finally { setLoading(false) }
  }

  const handleDelete = async (id: string, title: string) => {
    if (!window.confirm('确定删除课件「' + title + '」？')) return
    try { await deleteCourseware(id); loadData() } catch { alert('删除失败') }
  }

  // 打开新建弹窗
  const openCreateModal = () => {
    setShowCreate(true)
    setCreateMode('select')
    setSelectedPlanId('')
    setTopicSubject(''); setTopicGrade(''); setTopicName(''); setTopicNotes('')
    setPptFile(null); setPptSubject(''); setPptGrade(''); setPptTitle('')
    setDocFile(null); setDocSubject(''); setDocGrade(''); setDocTitle('')
  }

  // 选择"从教案创建"后加载教案列表
  const selectLessonPlanMode = async () => {
    setCreateMode('lesson_plan')
    setPlansLoading(true)
    try {
      const resp = await apiClient.get('/lesson-plans/plans', { params: { limit: 100 } })
      const data = resp?.data?.data
      const all: LPItem[] = (data?.plans || data?.lesson_plans || []) as LPItem[]
      setPlans(all.filter((p: LPItem) => ['published_personal', 'approved', 'published_shared'].includes(p.status)))
    } catch { setPlans([]) } finally { setPlansLoading(false) }
  }

  // 从教案创建
  const handleCreateFromPlan = async () => {
    if (!selectedPlanId) { alert('请选择关联的教案'); return }
    setCreating(true)
    try {
      const cw = await createCourseware({ lesson_plan_id: selectedPlanId })
      setShowCreate(false); navigate('/courseware/' + cw.id)
    } catch { alert('创建课件失败') } finally { setCreating(false) }
  }

  // 从主题创建
  const handleCreateFromTopic = async () => {
    if (!topicSubject || !topicGrade || !topicName.trim()) {
      alert('请填写学科、年级和主题名称'); return
    }
    setCreating(true)
    try {
      const cw = await createCoursewareFromTopic({
        subject: topicSubject,
        grade: topicGrade,
        topic: topicName.trim(),
        extra_notes: topicNotes.trim() || undefined,
      })
      setShowCreate(false); navigate('/courseware/' + cw.id)
    } catch { alert('创建课件失败') } finally { setCreating(false) }
  }

  // v0.42 入口B: 从PPT创建
  const handleCreateFromPPT = async () => {
    if (!pptFile) { alert('请选择PPT文件'); return }
    if (!pptSubject || !pptGrade) { alert('请填写学科和年级'); return }
    setCreating(true)
    try {
      const result = await createCoursewareFromPPT(
        pptFile, pptSubject, pptGrade, pptTitle.trim() || undefined,
      )
      setShowCreate(false)
      navigate('/courseware/' + result.id)
    } catch (e) {
      alert('PPT上传失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setCreating(false) }
  }

  // v0.42 入口C: 从Word文档创建
  const handleCreateFromDoc = async () => {
    if (!docFile) { alert('请选择Word文档'); return }
    if (!docSubject || !docGrade) { alert('请填写学科和年级'); return }
    setCreating(true)
    try {
      const result = await createCoursewareFromDoc(
        docFile, docSubject, docGrade, docTitle.trim() || undefined,
      )
      setShowCreate(false)
      navigate('/courseware/' + result.id)
    } catch (e) {
      alert('文档上传失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setCreating(false) }
  }

  // v0.42.11: 从3D互动单页创建
  const handleCreateFrom3D = async () => {
    if (!threeDSubject || !threeDGrade || !threeDTopic.trim()) {
      alert('请填写学科、年级和主题名称'); return
    }
    if (threeDDesc.trim().length < 20) {
      alert('请填写至少20字的详细描述，以便AI生成高质量3D页面'); return
    }
    setCreating(true)
    try {
      const cw = await createCoursewareFrom3D({
        subject: threeDSubject,
        grade: threeDGrade,
        topic: threeDTopic.trim(),
        description: threeDDesc.trim(),
      })
      setShowCreate(false); navigate('/courseware/' + cw.id)
    } catch { alert('创建课件失败') } finally { setCreating(false) }
  }

  // Word文件选择处理
  const handleDocFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    if (!file.name.toLowerCase().endsWith('.docx')) {
      alert('仅支持.docx格式的Word文件')
      e.target.value = ''
      return
    }
    if (file.size > 30 * 1024 * 1024) {
      alert('Word文件过大，最大支持30MB')
      e.target.value = ''
      return
    }
    setDocFile(file)
    if (!docTitle) {
      setDocTitle(file.name.replace(/\.docx$/i, ''))
    }
  }

  // PPT文件选择处理
  const handlePPTFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    if (!file.name.toLowerCase().endsWith('.pptx')) {
      alert('仅支持.pptx格式的PPT文件')
      e.target.value = ''
      return
    }
    if (file.size > 50 * 1024 * 1024) {
      alert('PPT文件过大，最大支持50MB')
      e.target.value = ''
      return
    }
    setPptFile(file)
    // 自动用文件名作为标题（如果用户没填）
    if (!pptTitle) {
      setPptTitle(file.name.replace(/\.pptx$/i, ''))
    }
  }

  const statusFilters = [
    { value: '', label: '全部' }, { value: 'draft', label: '草稿' },
    { value: 'generating', label: '生成中' }, { value: 'preview', label: '预览中' },
    { value: 'confirmed', label: '已确认' }, { value: 'in_pipeline', label: '审核中' },
  ]

  const btnBase: React.CSSProperties = { padding: '8px 20px', borderRadius: '8px', fontSize: '14px', cursor: 'pointer' }

  return (
    <div>
      {/* 顶部筛选栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <div style={{ display: 'flex', gap: '8px' }}>
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
        <button onClick={openCreateModal} style={{
          ...btnBase, border: 'none',
          background: 'linear-gradient(135deg, #F59E0B, #EF4444)', color: '#fff',
          fontWeight: 600, boxShadow: '0 2px 8px rgba(245,158,11,0.3)',
        }}>+ 新建课件</button>
      </div>

      <div style={{ fontSize: '13px', color: C.textMuted, marginBottom: '16px' }}>共 {total} 套课件</div>

      {/* 课件列表 */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '60px 0', color: C.textMuted }}>加载中...</div>
      ) : items.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '80px 0' }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>🎨</div>
          <div style={{ fontSize: '16px', color: C.textSecondary, marginBottom: '8px' }}>还没有课件</div>
          <div style={{ fontSize: '13px', color: C.textMuted }}>点击"新建课件"，选择从教案、主题或PPT开始创建</div>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: '16px' }}>
          {items.map(item => <CWCard key={item.id} item={item} onDelete={handleDelete} onClick={() => navigate('/courseware/' + item.id)} />)}
        </div>
      )}

      {/* ==================== 新建课件弹窗 ==================== */}
      {showCreate && (
        <div style={{ position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.5)', zIndex: 9999, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
          onClick={() => setShowCreate(false)}>
          <div style={{ background: '#fff', borderRadius: '16px', width: '90%', maxWidth: '600px', maxHeight: '80vh', overflow: 'auto', padding: '28px' }}
            onClick={e => e.stopPropagation()}>

            {/* 入口选择页 */}
            {createMode === 'select' && <>
              <div style={{ fontSize: '20px', fontWeight: 700, color: C.textPrimary, marginBottom: '8px' }}>🎨 新建课件</div>
              <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '20px' }}>选择创建方式</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                {/* 入口A: 从教案创建 */}
                <button onClick={selectLessonPlanMode} style={{
                  padding: '20px', borderRadius: '12px', border: `1px solid ${C.border}`, background: '#fff',
                  textAlign: 'left', cursor: 'pointer', transition: 'all 200ms',
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <span style={{ fontSize: '28px' }}>📝</span>
                    <div>
                      <div style={{ fontSize: '16px', fontWeight: 600, color: C.textPrimary }}>从教案创建</div>
                      <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '2px' }}>选择已完成的教案，AI基于教案内容自动生成课件方案</div>
                    </div>
                  </div>
                </button>
                {/* 入口D: 从主题创建 */}
                <button onClick={() => setCreateMode('topic')} style={{
                  padding: '20px', borderRadius: '12px', border: `1px solid ${C.border}`, background: '#fff',
                  textAlign: 'left', cursor: 'pointer', transition: 'all 200ms',
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <span style={{ fontSize: '28px' }}>💡</span>
                    <div>
                      <div style={{ fontSize: '16px', fontWeight: 600, color: C.textPrimary }}>从主题创建</div>
                      <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '2px' }}>输入学科、年级和主题，AI直接规划课件结构</div>
                    </div>
                  </div>
                </button>
                {/* 入口B: 从PPT创建（v0.42 激活） */}
                <button onClick={() => setCreateMode('ppt')} style={{
                  padding: '20px', borderRadius: '12px', border: `1px solid ${C.border}`, background: '#fff',
                  textAlign: 'left', cursor: 'pointer', transition: 'all 200ms',
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <span style={{ fontSize: '28px' }}>📊</span>
                    <div>
                      <div style={{ fontSize: '16px', fontWeight: 600, color: C.textPrimary }}>从 PPT 创建</div>
                      <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '2px' }}>上传已有PPT，AI自动提取内容并转化为交互式课件</div>
                    </div>
                  </div>
                </button>
                {/* 入口C: 从Word文档创建 */}
                <button onClick={() => setCreateMode('doc')} style={{
                  padding: '20px', borderRadius: '12px', border: `1px solid ${C.border}`, background: '#fff',
                  textAlign: 'left', cursor: 'pointer', transition: 'all 200ms',
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <span style={{ fontSize: '28px' }}>📄</span>
                    <div>
                      <div style={{ fontSize: '16px', fontWeight: 600, color: C.textPrimary }}>从教案文档创建</div>
                      <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '2px' }}>上传已有的Word教案文档，AI自动提取内容生成课件</div>
                    </div>
                  </div>
                </button>
                {/* v0.42.11 入口E: 3D互动单页 */}
                <button onClick={() => setCreateMode('3d_single')} style={{
                  padding: '20px', borderRadius: '12px', border: '1px solid #FCA5A5', background: 'linear-gradient(135deg, #FEF2F2, #FFF)',
                  textAlign: 'left', cursor: 'pointer', transition: 'all 200ms',
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <span style={{ fontSize: '28px' }}>🎮</span>
                    <div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <span style={{ fontSize: '16px', fontWeight: 600, color: C.textPrimary }}>3D 互动单页</span>
                        <span style={{ padding: '2px 8px', borderRadius: '8px', fontSize: '11px', color: '#DC2626', background: '#FEE2E2', fontWeight: 600 }}>NEW</span>
                      </div>
                      <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '2px' }}>AI生成Three.js 3D沉浸式互动课件，含粒子系统和分步骤演示</div>
                    </div>
                  </div>
                </button>
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '16px' }}>
                <button onClick={() => setShowCreate(false)} style={{ ...btnBase, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary }}>取消</button>
              </div>
            </>}

            {/* 从教案创建 */}
            {createMode === 'lesson_plan' && <>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
                <button onClick={() => setCreateMode('select')} style={{ background: 'none', border: 'none', fontSize: '14px', color: C.textSecondary, cursor: 'pointer' }}>← 返回</button>
                <div style={{ fontSize: '18px', fontWeight: 700, color: C.textPrimary }}>📝 从教案创建</div>
              </div>
              <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '16px' }}>选择一份已完成的教案，AI将基于教案内容自动生成课件</div>
              {plansLoading ? (
                <div style={{ textAlign: 'center', padding: '40px 0', color: C.textMuted }}>加载教案列表...</div>
              ) : plans.length === 0 ? (
                <div style={{ textAlign: 'center', padding: '40px 0' }}>
                  <div style={{ fontSize: '14px', color: C.textSecondary }}>没有可用的教案</div>
                  <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '6px' }}>请先在备课工坊完成教案开发</div>
                </div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '300px', overflow: 'auto', marginBottom: '20px' }}>
                  {plans.map(p => (
                    <div key={p.id} onClick={() => setSelectedPlanId(p.id)} style={{
                      padding: '14px 16px', borderRadius: '10px', cursor: 'pointer',
                      border: `2px solid ${selectedPlanId === p.id ? C.primary : C.border}`,
                      background: selectedPlanId === p.id ? 'rgba(245,158,11,0.05)' : '#fff',
                    }}>
                      <div style={{ fontSize: '14px', fontWeight: 600, color: C.textPrimary }}>{p.title}</div>
                      <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px', display: 'flex', gap: '10px' }}>
                        {p.subject && <span>📚 {p.subject}</span>}
                        {p.grade && <span>🎓 {p.grade}</span>}
                      </div>
                    </div>
                  ))}
                </div>
              )}
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
                <button onClick={() => setCreateMode('select')} style={{ ...btnBase, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary }}>返回</button>
                <button onClick={handleCreateFromPlan} disabled={!selectedPlanId || creating} style={{
                  ...btnBase, border: 'none',
                  background: selectedPlanId ? 'linear-gradient(135deg, #F59E0B, #EF4444)' : '#E5E7EB',
                  color: selectedPlanId ? '#fff' : '#9CA3AF', fontWeight: 600,
                  cursor: selectedPlanId && !creating ? 'pointer' : 'default',
                }}>{creating ? '创建中...' : '确认创建'}</button>
              </div>
            </>}

            {/* 从主题创建 */}
            {createMode === 'topic' && <>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
                <button onClick={() => setCreateMode('select')} style={{ background: 'none', border: 'none', fontSize: '14px', color: C.textSecondary, cursor: 'pointer' }}>← 返回</button>
                <div style={{ fontSize: '18px', fontWeight: 700, color: C.textPrimary }}>💡 从主题创建</div>
              </div>
              <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '20px' }}>输入学科、年级和主题名称，AI直接规划课件方案</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>学科 *</label>
                  <select value={topicSubject} onChange={e => setTopicSubject(e.target.value)} style={{
                    width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`,
                    fontSize: '14px', outline: 'none', background: '#fff',
                  }}>
                    <option value="">请选择学科</option>
                    {SUBJECTS.map(s => <option key={s} value={s}>{s}</option>)}
                  </select>
                </div>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>年级 *</label>
                  <input value={topicGrade} onChange={e => setTopicGrade(e.target.value)}
                    placeholder="如：三年级、初二、高一"
                    style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
                </div>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>主题名称 *</label>
                  <input value={topicName} onChange={e => setTopicName(e.target.value)}
                    placeholder="如：牛顿第一定律、认识人工智能、二次函数"
                    style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
                </div>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>额外说明（可选）</label>
                  <textarea value={topicNotes} onChange={e => setTopicNotes(e.target.value)}
                    placeholder="如：重点讲解力的合成与分解、需要包含实验环节..."
                    rows={3}
                    style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', resize: 'vertical', boxSizing: 'border-box' }} />
                </div>
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '20px' }}>
                <button onClick={() => setCreateMode('select')} style={{ ...btnBase, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary }}>返回</button>
                <button onClick={handleCreateFromTopic}
                  disabled={!topicSubject || !topicGrade || !topicName.trim() || creating}
                  style={{
                    ...btnBase, border: 'none',
                    background: (topicSubject && topicGrade && topicName.trim()) ? 'linear-gradient(135deg, #7C3AED, #6366F1)' : '#E5E7EB',
                    color: (topicSubject && topicGrade && topicName.trim()) ? '#fff' : '#9CA3AF', fontWeight: 600,
                    cursor: (topicSubject && topicGrade && topicName.trim() && !creating) ? 'pointer' : 'default',
                  }}>{creating ? '创建中...' : '💡 确认创建'}</button>
              </div>
            </>}

            {/* v0.42 入口B: 从PPT创建 */}
            {createMode === 'ppt' && <>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
                <button onClick={() => setCreateMode('select')} style={{ background: 'none', border: 'none', fontSize: '14px', color: C.textSecondary, cursor: 'pointer' }}>← 返回</button>
                <div style={{ fontSize: '18px', fontWeight: 700, color: C.textPrimary }}>📊 从 PPT 创建</div>
              </div>
              <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '20px' }}>上传.pptx文件，AI自动提取内容并转化为交互式课件方案</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                {/* PPT文件上传 */}
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>PPT文件 *</label>
                  <input ref={pptFileRef} type="file" accept=".pptx" onChange={handlePPTFileChange}
                    style={{ display: 'none' }} />
                  <div onClick={() => pptFileRef.current?.click()} style={{
                    padding: '24px', borderRadius: '10px', border: `2px dashed ${pptFile ? '#059669' : C.border}`,
                    background: pptFile ? '#F0FDF4' : '#FAFAFA', cursor: 'pointer', textAlign: 'center',
                    transition: 'all 200ms',
                  }}>
                    {pptFile ? (
                      <div>
                        <div style={{ fontSize: '28px', marginBottom: '6px' }}>✅</div>
                        <div style={{ fontSize: '14px', fontWeight: 600, color: '#059669' }}>{pptFile.name}</div>
                        <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>{(pptFile.size / (1024 * 1024)).toFixed(1)} MB · 点击更换</div>
                      </div>
                    ) : (
                      <div>
                        <div style={{ fontSize: '28px', marginBottom: '6px' }}>📊</div>
                        <div style={{ fontSize: '14px', color: C.textSecondary }}>点击选择.pptx文件</div>
                        <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>仅支持.pptx格式，最大50MB</div>
                      </div>
                    )}
                  </div>
                </div>
                {/* 学科选择 */}
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>学科 *</label>
                  <select value={pptSubject} onChange={e => setPptSubject(e.target.value)} style={{
                    width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`,
                    fontSize: '14px', outline: 'none', background: '#fff',
                  }}>
                    <option value="">请选择学科</option>
                    {SUBJECTS.map(s => <option key={s} value={s}>{s}</option>)}
                  </select>
                </div>
                {/* 年级输入 */}
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>年级 *</label>
                  <input value={pptGrade} onChange={e => setPptGrade(e.target.value)}
                    placeholder="如：三年级、初二、高一"
                    style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
                </div>
                {/* 课件标题（可选） */}
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>课件标题（可选，默认使用PPT文件名）</label>
                  <input value={pptTitle} onChange={e => setPptTitle(e.target.value)}
                    placeholder="留空则使用PPT文件名作为课件标题"
                    style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
                </div>
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '20px' }}>
                <button onClick={() => setCreateMode('select')} style={{ ...btnBase, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary }}>返回</button>
                <button onClick={handleCreateFromPPT}
                  disabled={!pptFile || !pptSubject || !pptGrade || creating}
                  style={{
                    ...btnBase, border: 'none',
                    background: (pptFile && pptSubject && pptGrade) ? 'linear-gradient(135deg, #D97706, #F59E0B)' : '#E5E7EB',
                    color: (pptFile && pptSubject && pptGrade) ? '#fff' : '#9CA3AF', fontWeight: 600,
                    cursor: (pptFile && pptSubject && pptGrade && !creating) ? 'pointer' : 'default',
                  }}>{creating ? '⏳ 上传解析中...' : '📊 上传并创建'}</button>
              </div>
            </>}

            {/* v0.42 入口C: 从Word文档创建 */}
            {createMode === 'doc' && <>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
                <button onClick={() => setCreateMode('select')} style={{ background: 'none', border: 'none', fontSize: '14px', color: C.textSecondary, cursor: 'pointer' }}>← 返回</button>
                <div style={{ fontSize: '18px', fontWeight: 700, color: C.textPrimary }}>📄 从教案文档创建</div>
              </div>
              <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '20px' }}>上传.docx教案文件，AI自动提取内容并转化为交互式课件方案</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>教案文档 *</label>
                  <input ref={docFileRef} type="file" accept=".docx" onChange={handleDocFileChange} style={{ display: 'none' }} />
                  <div onClick={() => docFileRef.current?.click()} style={{
                    padding: '24px', borderRadius: '10px', border: `2px dashed ${docFile ? '#059669' : C.border}`,
                    background: docFile ? '#F0FDF4' : '#FAFAFA', cursor: 'pointer', textAlign: 'center',
                  }}>
                    {docFile ? (
                      <div>
                        <div style={{ fontSize: '28px', marginBottom: '6px' }}>✅</div>
                        <div style={{ fontSize: '14px', fontWeight: 600, color: '#059669' }}>{docFile.name}</div>
                        <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>{(docFile.size / (1024 * 1024)).toFixed(1)} MB · 点击更换</div>
                      </div>
                    ) : (
                      <div>
                        <div style={{ fontSize: '28px', marginBottom: '6px' }}>📄</div>
                        <div style={{ fontSize: '14px', color: C.textSecondary }}>点击选择.docx教案文件</div>
                        <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>仅支持.docx格式，最大30MB</div>
                      </div>
                    )}
                  </div>
                </div>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>学科 *</label>
                  <select value={docSubject} onChange={e => setDocSubject(e.target.value)} style={{
                    width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: '#fff',
                  }}>
                    <option value="">请选择学科</option>
                    {SUBJECTS.map(s => <option key={s} value={s}>{s}</option>)}
                  </select>
                </div>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>年级 *</label>
                  <input value={docGrade} onChange={e => setDocGrade(e.target.value)}
                    placeholder="如：三年级、初二、高一"
                    style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
                </div>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>课件标题（可选，默认使用文档文件名）</label>
                  <input value={docTitle} onChange={e => setDocTitle(e.target.value)}
                    placeholder="留空则使用文档文件名作为课件标题"
                    style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
                </div>
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '20px' }}>
                <button onClick={() => setCreateMode('select')} style={{ ...btnBase, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary }}>返回</button>
                <button onClick={handleCreateFromDoc}
                  disabled={!docFile || !docSubject || !docGrade || creating}
                  style={{
                    ...btnBase, border: 'none',
                    background: (docFile && docSubject && docGrade) ? 'linear-gradient(135deg, #0891B2, #06B6D4)' : '#E5E7EB',
                    color: (docFile && docSubject && docGrade) ? '#fff' : '#9CA3AF', fontWeight: 600,
                    cursor: (docFile && docSubject && docGrade && !creating) ? 'pointer' : 'default',
                  }}>{creating ? '⏳ 上传解析中...' : '📄 上传并创建'}</button>
              </div>
            </>}

            {/* v0.42.11: 3D互动单页创建 */}
            {createMode === '3d_single' && <>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
                <button onClick={() => setCreateMode('select')} style={{ background: 'none', border: 'none', fontSize: '14px', color: C.textSecondary, cursor: 'pointer' }}>← 返回</button>
                <div style={{ fontSize: '18px', fontWeight: 700, color: C.textPrimary }}>🎮 3D 互动单页</div>
                <span style={{ padding: '2px 8px', borderRadius: '8px', fontSize: '11px', color: '#DC2626', background: '#FEE2E2', fontWeight: 600 }}>NEW</span>
              </div>
              <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '12px' }}>AI 将生成一个基于 Three.js 的 3D 沉浸式互动页面，包含粒子系统、分步骤演示和3D模型交互</div>
              <div style={{ padding: '12px', borderRadius: '10px', background: '#FEF3C7', border: '1px solid #FDE68A', fontSize: '13px', color: '#92400E', marginBottom: '16px' }}>
                💡 提示：详细描述越具体，AI 生成的 3D 效果越精细。建议描述清楚要展示的对象、过程和关键知识点。
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>学科 *</label>
                  <select value={threeDSubject} onChange={e => setThreeDSubject(e.target.value)} style={{
                    width: '100%', padding: '10px 12px', borderRadius: '8px', border: '1px solid ' + C.border, fontSize: '14px', outline: 'none', background: '#fff',
                  }}>
                    <option value="">请选择学科</option>
                    {SUBJECTS.map(s => <option key={s} value={s}>{s}</option>)}
                  </select>
                </div>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>年级 *</label>
                  <input value={threeDGrade} onChange={e => setThreeDGrade(e.target.value)}
                    placeholder="如：三年级、初二、高一"
                    style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: '1px solid ' + C.border, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
                </div>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>3D 主题名称 *</label>
                  <input value={threeDTopic} onChange={e => setThreeDTopic(e.target.value)}
                    placeholder="如：水循环、光合作用、细胞结构、太阳系运行"
                    style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: '1px solid ' + C.border, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
                </div>
                <div>
                  <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>详细描述 *（至少20字）</label>
                  <textarea value={threeDDesc} onChange={e => setThreeDDesc(e.target.value)}
                    placeholder="详细描述你想要展示的3D场景，例如：&#10;• 要展示哪些具体物体/结构？&#10;• 有哪些关键过程/步骤需要分步演示？&#10;• 有没有特殊的视觉效果需求？&#10;&#10;示例：展示植物细胞的完整结构，包括细胞壁、细胞膜、细胞核、叶绿体、线粒体等细胞器。需要能点击选择各细胞器查看详细说明，支持透视模式看清内部结构。"
                    rows={5}
                    style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: '1px solid ' + C.border, fontSize: '14px', outline: 'none', resize: 'vertical', boxSizing: 'border-box' }} />
                  <div style={{ fontSize: '12px', color: threeDDesc.trim().length >= 20 ? '#059669' : '#9CA3AF', marginTop: '4px' }}>
                    {threeDDesc.trim().length} / 20 字（最少）
                  </div>
                </div>
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '20px' }}>
                <button onClick={() => setCreateMode('select')} style={{ ...btnBase, border: '1px solid ' + C.border, background: 'transparent', color: C.textSecondary }}>返回</button>
                <button onClick={handleCreateFrom3D}
                  disabled={!threeDSubject || !threeDGrade || !threeDTopic.trim() || threeDDesc.trim().length < 20 || creating}
                  style={{
                    ...btnBase, border: 'none',
                    background: (threeDSubject && threeDGrade && threeDTopic.trim() && threeDDesc.trim().length >= 20) ? 'linear-gradient(135deg, #DC2626, #EF4444)' : '#E5E7EB',
                    color: (threeDSubject && threeDGrade && threeDTopic.trim() && threeDDesc.trim().length >= 20) ? '#fff' : '#9CA3AF', fontWeight: 600,
                    cursor: (threeDSubject && threeDGrade && threeDTopic.trim() && threeDDesc.trim().length >= 20 && !creating) ? 'pointer' : 'default',
                  }}>{creating ? '创建中...' : '🎮 创建3D课件'}</button>
              </div>
            </>}
          </div>
        </div>
      )}
    </div>
  )
}

// ==================== 课件卡片组件 ====================
function CWCard({ item, onDelete, onClick }: {
  item: CoursewareListItem; onDelete: (id: string, t: string) => void; onClick: () => void
}) {
  const [hovered, setHovered] = useState(false)
  const [downloading, setDownloading] = useState(false)
  const sc = CW_STATUS_CONFIG[item.status] || { label: item.status, color: '#6B7280', bg: '#F3F4F6' }
  const src = SOURCE_CONFIG[item.source_type] || SOURCE_CONFIG.lesson_plan
  return (
    <div onClick={onClick} onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}
      style={{
        background: '#fff', borderRadius: '12px', padding: '20px',
        border: `1px solid ${hovered ? 'rgba(245,158,11,0.3)' : '#E5E7EB'}`,
        cursor: 'pointer', transition: 'all 200ms',
        transform: hovered ? 'translateY(-2px)' : 'none',
        boxShadow: hovered ? '0 4px 16px rgba(0,0,0,0.08)' : '0 1px 3px rgba(0,0,0,0.04)',
      }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '12px' }}>
        <div style={{ fontSize: '16px', fontWeight: 600, color: '#1F2937', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{item.title}</div>
        <span style={{ padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 500, color: sc.color, background: sc.bg, flexShrink: 0, marginLeft: '8px' }}>{sc.label}</span>
      </div>
      <div style={{ fontSize: '13px', color: '#6B7280', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '8px' }}>
        <span style={{ padding: '1px 8px', borderRadius: '8px', fontSize: '11px', color: src.color, background: src.bg }}>{src.emoji} {src.label}</span>
        {item.lesson_plan_title && <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>📝 {item.lesson_plan_title}</span>}
      </div>
      <div style={{ display: 'flex', gap: '16px', fontSize: '12px', color: '#9CA3AF' }}>
        {item.subject && <span>📚 {item.subject}</span>}
        {item.grade && <span>🎓 {item.grade}</span>}
        <span>📄 {item.page_count} 页</span>
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '14px', paddingTop: '12px', borderTop: '1px solid #E5E7EB' }}>
        <span style={{ fontSize: '12px', color: '#9CA3AF' }}>{item.created_at ? new Date(item.created_at).toLocaleDateString('zh-CN') : ''}</span>
        {['preview', 'confirmed', 'in_pipeline'].includes(item.status) && (
          <button
            onClick={async e => {
              e.stopPropagation()
              if (downloading) return
              setDownloading(true)
              try { await downloadCoursewareBundle(item.id, item.title) }
              catch (err) { alert('下载失败: ' + (err instanceof Error ? err.message : '未知错误')) }
              finally { setDownloading(false) }
            }}
            disabled={downloading}
            style={{ padding: '2px 10px', borderRadius: '6px', border: '1px solid #BFDBFE', background: downloading ? '#EFF6FF' : 'transparent', color: '#2563EB', fontSize: '12px', cursor: downloading ? 'default' : 'pointer' }}>
            {downloading ? '⏳ 打包中…' : '⬇ 下载离线包'}
          </button>
        )}
        {item.status === 'draft' && (
          <button onClick={e => { e.stopPropagation(); onDelete(item.id, item.title) }}
            style={{ padding: '2px 10px', borderRadius: '6px', border: '1px solid #E5E7EB', background: 'transparent', color: '#EF4444', fontSize: '12px', cursor: 'pointer' }}>
            删除
          </button>
        )}
      </div>
    </div>
  )
}

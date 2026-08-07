/**
 * 新建课件弹窗 — CreateCoursewareModal.tsx
 *
 * 从 CoursewareListPage.tsx 拆出的"新建课件"五入口弹窗(整块连同其全部
 * useState 与 handler 一并搬入,组件自管状态)。主文件只需控制是否打开:
 *   - open      是否显示弹窗
 *   - onClose   关闭弹窗(取消/遮罩点击)
 *   - onCreated 创建成功后回调,父组件据此 navigate 到新课件工坊
 *
 * 五入口(createMode 内部状态机):
 *   select       入口选择页
 *   lesson_plan  从教案创建(拉已发布教案列表 + 搜索)
 *   topic        从主题创建(学科/年级/主题 + 课标知识点选择器)
 *   ppt          从 PPT 上传创建(.pptx ≤50MB)
 *   doc          从 Word 文档创建(.docx ≤30MB)
 *   3d_single    3D 互动单页(学科/年级/主题/详细描述≥20字)
 *
 * 引用 KnowledgePointSelector 的相对路径:本文件在 components/courseware-list/,
 *   需上溯两级到 pages/courseware/ 根目录,故为 '../../KnowledgePointSelector'。
 */

import { useState, useEffect, useMemo, useRef } from 'react'
import {
  createCourseware, createCoursewareFromTopic,
  createCoursewareFromPPT, createCoursewareFromDoc, createCoursewareFrom3D,
} from '@/api/coursewares'
import apiClient from '@/api/client'
import { useAuth } from '@/store/auth'
import { useSubjects } from '@/hooks/useSubjects'
import { useEducationProfile } from '@/hooks/useEducationProfile'
import {
  getEducationLevelOptions,
  getTopicPlaceholder,
} from '@/education-domain/options'
import KnowledgePointSelector from '../../KnowledgePointSelector'
import { C, btnBase, type LPItem } from './listConstants'


/**
 * 后端从教案创建课件只允许使用当前账号本人教案。
 *
 * 教案列表接口可能同时返回本人、管理范围或共享教案。
 * 前端必须把“可查看”和“可创建课件”分开处理。
 */
const CREATABLE_LESSON_PLAN_STATUSES =
  new Set<string>([
    'published_personal',
    'approved',
    'published_shared',
  ])

interface LessonPlanOwnershipFields {
  author_id?: string
  authorId?: string
  created_by?: string
  createdBy?: string
  user_id?: string
  userId?: string
  is_owner?: boolean
}

/**
 * 判断教案是否属于当前登录用户。
 *
 * 优先使用后端显式的is_owner，其次兼容常见作者ID字段。
 * 旧响应完全没有作者字段时，只排除最可能来自共享库的
 * published_shared，最终创建仍由后端重新校验作者身份。
 */
function isCurrentUserLessonPlan(
  plan: LPItem,
  currentUserID: string,
): boolean {
  if (!currentUserID) {
    return false
  }

  const candidate =
    plan as LPItem &
      LessonPlanOwnershipFields

  if (
    typeof candidate.is_owner ===
    'boolean'
  ) {
    return candidate.is_owner
  }

  const ownerID = (
    candidate.author_id ||
    candidate.authorId ||
    candidate.created_by ||
    candidate.createdBy ||
    candidate.user_id ||
    candidate.userId ||
    ''
  ).trim()

  if (ownerID) {
    return ownerID ===
      currentUserID
  }

  return plan.status !==
    'published_shared'
}

/** 提取Axios拦截器已经规范化的真实业务错误。 */
function coursewareCreateErrorMessage(
  error: unknown,
  fallback: string,
): string {
  return error instanceof Error &&
    error.message.trim()
    ? error.message
    : fallback
}

export default function CreateCoursewareModal({ open, onClose, onCreated }: {
  open: boolean
  onClose: () => void
  onCreated: (coursewareId: string) => void
}) {
  const { user } = useAuth()

  /**
   * 课程目录来自当前登录用户的教育域和教学组织。
   * 职业教育、成人教育不会再读取K12静态学科清单。
   */
  const {
    subjects,
    loading: subjectsLoading,
    empty: subjectsEmpty,
  } = useSubjects()

  const {
    domain,
    profile,
  } = useEducationProfile()

  const levelOptions = useMemo(
    () => getEducationLevelOptions(domain),
    [domain],
  )

  const levelValues = useMemo(
    () => levelOptions.map(item => item.value),
    [levelOptions],
  )

  // 入口状态机
  const [createMode, setCreateMode] = useState<'select' | 'lesson_plan' | 'topic' | 'ppt' | 'doc' | '3d_single'>('select')

  // 从教案创建相关
  const [plans, setPlans] = useState<LPItem[]>([])
  const [plansLoading, setPlansLoading] = useState(false)
  const [selectedPlanId, setSelectedPlanId] = useState('')
  const [planSearch, setPlanSearch] = useState('')

  // 从主题创建相关
  const [topicSubject, setTopicSubject] = useState('')
  const [topicGrade, setTopicGrade] = useState('')
  const [topicName, setTopicName] = useState('')
  const [topicNotes, setTopicNotes] = useState('')
  const [topicKPCodes, setTopicKPCodes] = useState<string[]>([])

  // 从 PPT 创建相关
  const [pptFile, setPptFile] = useState<File | null>(null)
  const [pptSubject, setPptSubject] = useState('')
  const [pptGrade, setPptGrade] = useState('')
  const [pptTitle, setPptTitle] = useState('')
  const pptFileRef = useRef<HTMLInputElement>(null)

  // 从 Word 文档创建相关
  const [docFile, setDocFile] = useState<File | null>(null)
  const [docSubject, setDocSubject] = useState('')
  const [docGrade, setDocGrade] = useState('')
  const [docTitle, setDocTitle] = useState('')
  const docFileRef = useRef<HTMLInputElement>(null)

  // 3D 互动单页创建相关
  const [threeDSubject, setThreeDSubject] = useState('')
  const [threeDGrade, setThreeDGrade] = useState('')
  const [threeDTopic, setThreeDTopic] = useState('')
  const [threeDDesc, setThreeDDesc] = useState('')

  const [creating, setCreating] = useState(false)

  // 每次打开弹窗时重置全部表单状态,回到入口选择页(替代原 openCreateModal 的集中重置)
  useEffect(() => {
    if (!open) return
    setCreateMode('select')
    setSelectedPlanId(''); setPlanSearch('')
    setTopicSubject(''); setTopicGrade(''); setTopicName(''); setTopicNotes(''); setTopicKPCodes([])
    setPptFile(null); setPptSubject(''); setPptGrade(''); setPptTitle('')
    setDocFile(null); setDocSubject(''); setDocGrade(''); setDocTitle('')
    setThreeDSubject(''); setThreeDGrade(''); setThreeDTopic(''); setThreeDDesc('')
  }, [open])

  /**
   * 弹窗打开或课程目录刷新后，四个直接创建入口
   * 始终使用当前教育域实际可用的课程。
   */
  useEffect(() => {
    if (!open || subjectsLoading) return

    const firstSubject = subjects[0] || ''

    setTopicSubject(previous =>
      subjects.includes(previous)
        ? previous
        : firstSubject,
    )

    setPptSubject(previous =>
      subjects.includes(previous)
        ? previous
        : firstSubject,
    )

    setDocSubject(previous =>
      subjects.includes(previous)
        ? previous
        : firstSubject,
    )

    setThreeDSubject(previous =>
      subjects.includes(previous)
        ? previous
        : firstSubject,
    )
  }, [
    open,
    subjectsLoading,
    subjects,
  ])

  /**
   * 学习层级按当前教育域呈现：
   * K12使用年级，职教使用中职层级，成人教育使用学习基础。
   */
  useEffect(() => {
    if (!open) return

    const firstLevel = levelValues[0] || ''

    setTopicGrade(previous =>
      levelValues.includes(previous)
        ? previous
        : firstLevel,
    )

    setPptGrade(previous =>
      levelValues.includes(previous)
        ? previous
        : firstLevel,
    )

    setDocGrade(previous =>
      levelValues.includes(previous)
        ? previous
        : firstLevel,
    )

    setThreeDGrade(previous =>
      levelValues.includes(previous)
        ? previous
        : firstLevel,
    )
  }, [
    open,
    levelValues,
  ])

  /**
   * 课标知识点属于K12课程标准能力。
   * 当前教育域未启用课程标准时清除旧选择，避免跨域携带。
   */
  useEffect(() => {
    if (!profile.curriculum_enabled) {
      setTopicKPCodes([])
    }
  }, [profile.curriculum_enabled])

  if (!open) return null

  /**
   * 进入“从教案创建”后，只加载当前账号本人可创建的教案。
   *
   * 状态过滤和作者过滤均为前端体验保护；
   * 后端仍是最终权限事实源。
   */
  const selectLessonPlanMode = async () => {
    setCreateMode('lesson_plan')
    setPlansLoading(true)
    setSelectedPlanId('')

    try {
      const resp = await apiClient.get(
        '/lesson-plans/plans',
        {
          params: {
            limit: 100,
          },
        },
      )

      const data =
        resp?.data?.data

      const all: LPItem[] = (
        data?.plans ||
        data?.lesson_plans ||
        []
      ) as LPItem[]

      const currentUserID =
        (user?.id || '').trim()

      const ownCreatablePlans =
        all.filter(plan =>
          CREATABLE_LESSON_PLAN_STATUSES
            .has(plan.status) &&
          isCurrentUserLessonPlan(
            plan,
            currentUserID,
          ),
        )

      setPlans(
        ownCreatablePlans,
      )
    } catch (error) {
      setPlans([])

      alert(
        '加载可创建教案失败：' +
        coursewareCreateErrorMessage(
          error,
          '请求失败',
        ),
      )
    } finally {
      setPlansLoading(false)
    }
  }

  // 从教案创建
  const handleCreateFromPlan = async () => {
    if (!selectedPlanId) {
      alert('请选择关联的教案')
      return
    }

    setCreating(true)

    try {
      const cw =
        await createCourseware({
          lesson_plan_id:
            selectedPlanId,
        })

      onCreated(cw.id)
    } catch (error) {
      /**
       * 不再吞掉后端的权限和教育域错误。
       *
       * 例如：
       * - 只能从自己的教案创建课件；
       * - 无确定教学教育域不能创建课件；
       * - 教案教育域快照无效。
       */
      alert(
        '创建课件失败：' +
        coursewareCreateErrorMessage(
          error,
          '请求失败',
        ),
      )
    } finally {
      setCreating(false)
    }
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
        kp_codes: topicKPCodes.length > 0 ? topicKPCodes : undefined,
      })
      onCreated(cw.id)
    } catch { alert('创建课件失败') } finally { setCreating(false) }
  }

  // 从 PPT 创建
  const handleCreateFromPPT = async () => {
    if (!pptFile) { alert('请选择PPT文件'); return }
    if (!pptSubject || !pptGrade) { alert('请填写学科和年级'); return }
    setCreating(true)
    try {
      const result = await createCoursewareFromPPT(pptFile, pptSubject, pptGrade, pptTitle.trim() || undefined)
      onCreated(result.id)
    } catch (e) {
      alert('PPT上传失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setCreating(false) }
  }

  // 从 Word 文档创建
  const handleCreateFromDoc = async () => {
    if (!docFile) { alert('请选择Word文档'); return }
    if (!docSubject || !docGrade) { alert('请填写学科和年级'); return }
    setCreating(true)
    try {
      const result = await createCoursewareFromDoc(docFile, docSubject, docGrade, docTitle.trim() || undefined)
      onCreated(result.id)
    } catch (e) {
      alert('文档上传失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setCreating(false) }
  }

  // 从 3D 互动单页创建
  const handleCreateFrom3D = async () => {
    if (!threeDSubject || !threeDGrade || !threeDTopic.trim()) {
      alert('请填写学科、年级和主题名称'); return
    }
    if (threeDDesc.trim().length < 20) {
      alert('请填写至少20字的详细描述,以便AI生成高质量3D页面'); return
    }
    setCreating(true)
    try {
      const cw = await createCoursewareFrom3D({
        subject: threeDSubject,
        grade: threeDGrade,
        topic: threeDTopic.trim(),
        description: threeDDesc.trim(),
      })
      onCreated(cw.id)
    } catch { alert('创建课件失败') } finally { setCreating(false) }
  }

  // Word 文件选择处理(仅 .docx,≤30MB,自动用文件名填标题)
  const handleDocFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    if (!file.name.toLowerCase().endsWith('.docx')) { alert('仅支持.docx格式的Word文件'); e.target.value = ''; return }
    if (file.size > 30 * 1024 * 1024) { alert('Word文件过大,最大支持30MB'); e.target.value = ''; return }
    setDocFile(file)
    if (!docTitle) setDocTitle(file.name.replace(/\.docx$/i, ''))
  }

  // PPT 文件选择处理(仅 .pptx,≤50MB,自动用文件名填标题)
  const handlePPTFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    if (!file.name.toLowerCase().endsWith('.pptx')) { alert('仅支持.pptx格式的PPT文件'); e.target.value = ''; return }
    if (file.size > 50 * 1024 * 1024) { alert('PPT文件过大,最大支持50MB'); e.target.value = ''; return }
    setPptFile(file)
    if (!pptTitle) setPptTitle(file.name.replace(/\.pptx$/i, ''))
  }

  return (
    <div style={{ position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.5)', zIndex: 9999, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={onClose}>
      <div style={{ background: '#fff', borderRadius: '16px', width: '90%', maxWidth: '600px', maxHeight: '80vh', overflow: 'auto', padding: '28px' }}
        onClick={e => e.stopPropagation()}>

        {subjectsEmpty &&
         !subjectsLoading &&
         createMode !== 'select' &&
         createMode !== 'lesson_plan' && (
          <div style={{
            marginBottom: '16px',
            padding: '10px 12px',
            borderRadius: '9px',
            background: '#FEF2F2',
            color: '#DC2626',
            fontSize: '12px',
            lineHeight: 1.6,
          }}>
            当前组织尚未配置可用
            {profile.subject_label}，请联系管理员。
          </div>
        )}

        {/* 入口选择页 */}
        {createMode === 'select' && <>
          <div style={{ fontSize: '20px', fontWeight: 700, color: C.textPrimary, marginBottom: '8px' }}>🎨 新建课件</div>
          <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '20px' }}>选择创建方式</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
            {/* 入口A: 从教案创建 */}
            <button onClick={selectLessonPlanMode} style={{ padding: '20px', borderRadius: '12px', border: `1px solid ${C.border}`, background: '#fff', textAlign: 'left', cursor: 'pointer', transition: 'all 200ms' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <span style={{ fontSize: '28px' }}>📝</span>
                <div>
                  <div style={{ fontSize: '16px', fontWeight: 600, color: C.textPrimary }}>从教案创建</div>
                  <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '2px' }}>选择已完成的教案,AI基于教案内容自动生成课件方案</div>
                </div>
              </div>
            </button>
            {/* 入口D: 从主题创建 */}
            <button onClick={() => setCreateMode('topic')} style={{ padding: '20px', borderRadius: '12px', border: `1px solid ${C.border}`, background: '#fff', textAlign: 'left', cursor: 'pointer', transition: 'all 200ms' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <span style={{ fontSize: '28px' }}>💡</span>
                <div>
                  <div style={{ fontSize: '16px', fontWeight: 600, color: C.textPrimary }}>从主题创建</div>
                  <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '2px' }}>输入学科、年级和主题,AI直接规划课件结构</div>
                </div>
              </div>
            </button>
            {/* 入口B: 从PPT创建 */}
            <button onClick={() => setCreateMode('ppt')} style={{ padding: '20px', borderRadius: '12px', border: `1px solid ${C.border}`, background: '#fff', textAlign: 'left', cursor: 'pointer', transition: 'all 200ms' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <span style={{ fontSize: '28px' }}>📊</span>
                <div>
                  <div style={{ fontSize: '16px', fontWeight: 600, color: C.textPrimary }}>从 PPT 创建</div>
                  <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '2px' }}>上传已有PPT,AI自动提取内容并转化为交互式课件</div>
                </div>
              </div>
            </button>
            {/* 入口C: 从Word文档创建 */}
            <button onClick={() => setCreateMode('doc')} style={{ padding: '20px', borderRadius: '12px', border: `1px solid ${C.border}`, background: '#fff', textAlign: 'left', cursor: 'pointer', transition: 'all 200ms' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <span style={{ fontSize: '28px' }}>📄</span>
                <div>
                  <div style={{ fontSize: '16px', fontWeight: 600, color: C.textPrimary }}>从教案文档创建</div>
                  <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '2px' }}>上传已有的Word教案文档,AI自动提取内容生成课件</div>
                </div>
              </div>
            </button>
            {/* 入口E: 3D互动单页 */}
            <button onClick={() => setCreateMode('3d_single')} style={{ padding: '20px', borderRadius: '12px', border: '1px solid #FCA5A5', background: 'linear-gradient(135deg, #FEF2F2, #FFF)', textAlign: 'left', cursor: 'pointer', transition: 'all 200ms' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <span style={{ fontSize: '28px' }}>🎮</span>
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <span style={{ fontSize: '16px', fontWeight: 600, color: C.textPrimary }}>3D 互动单页</span>
                    <span style={{ padding: '2px 8px', borderRadius: '8px', fontSize: '11px', color: '#DC2626', background: '#FEE2E2', fontWeight: 600 }}>NEW</span>
                  </div>
                  <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '2px' }}>AI生成Three.js 3D沉浸式互动课件,含粒子系统和分步骤演示</div>
                </div>
              </div>
            </button>
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '16px' }}>
            <button onClick={onClose} style={{ ...btnBase, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary }}>取消</button>
          </div>
        </>}

        {/* 从教案创建 */}
        {createMode === 'lesson_plan' && <>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
            <button onClick={() => setCreateMode('select')} style={{ background: 'none', border: 'none', fontSize: '14px', color: C.textSecondary, cursor: 'pointer' }}>← 返回</button>
            <div style={{ fontSize: '18px', fontWeight: 700, color: C.textPrimary }}>📝 从教案创建</div>
          </div>
          <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '16px' }}>选择一份已完成的教案,AI将基于教案内容自动生成课件</div>
          {!plansLoading && plans.length > 0 && (
            <input value={planSearch} onChange={e => setPlanSearch(e.target.value)}
              placeholder="🔍 搜索教案(名称 / 学科 / 年级)"
              style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box', marginBottom: '12px' }} />
          )}
          {plansLoading ? (
            <div style={{ textAlign: 'center', padding: '40px 0', color: C.textMuted }}>加载教案列表...</div>
          ) : plans.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '40px 0' }}>
              <div style={{ fontSize: '14px', color: C.textSecondary }}>当前账号没有可用于创建课件的本人教案</div>
              <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '6px' }}>请先完成自己的教案；共享或他人教案需要先复制到“我的教案”</div>
            </div>
          ) : (() => {
            const kw = planSearch.trim().toLowerCase()
            const plansFiltered = kw
              ? plans.filter(p =>
                  (p.title || '').toLowerCase().includes(kw) ||
                  (p.subject || '').toLowerCase().includes(kw) ||
                  (p.grade || '').toLowerCase().includes(kw))
              : plans
            if (plansFiltered.length === 0) {
              return (
                <div style={{ textAlign: 'center', padding: '32px 0', color: C.textMuted, fontSize: '13px', marginBottom: '12px' }}>
                  未找到匹配「{planSearch}」的教案,换个关键词试试
                </div>
              )
            }
            return (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '300px', overflow: 'auto', marginBottom: '20px' }}>
              {plansFiltered.map(p => (
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
            )
          })()}
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
          <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '20px' }}>输入学科、年级和主题名称,AI直接规划课件方案</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>{profile.subject_label} *</label>
              <select value={topicSubject} onChange={e => setTopicSubject(e.target.value)} style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: '#fff' }}>
                <option value="">请选择学科</option>
                {subjects.map(s => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>{profile.grade_label} *</label>
              <select
                value={topicGrade}
                onChange={e => setTopicGrade(e.target.value)}
                style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: '#fff' }}
              >
                <option value="">
                  请选择{profile.grade_label}
                </option>

                {levelOptions.map(option => (
                  <option
                    key={option.value}
                    value={option.value}
                  >
                    {option.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>主题名称 *</label>
              <input value={topicName} onChange={e => setTopicName(e.target.value)} placeholder={getTopicPlaceholder(domain)}
                style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>额外说明(可选)</label>
              <textarea value={topicNotes} onChange={e => setTopicNotes(e.target.value)} placeholder="如:重点讲解力的合成与分解、需要包含实验环节..." rows={3}
                style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', resize: 'vertical', boxSizing: 'border-box' }} />
            </div>
            {profile.curriculum_enabled && (
              <div>
                <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>课标知识点(可选,勾选后AI按课标难度自动适配)</label>
                <KnowledgePointSelector subject={topicSubject} grade={topicGrade} selectedCodes={topicKPCodes} onChange={setTopicKPCodes} />

                {topicKPCodes.length > 0 && (
                  <div style={{ marginTop: '8px', padding: '8px 12px', borderRadius: '8px', background: 'rgba(124,58,237,0.06)', border: '1px solid rgba(124,58,237,0.2)', fontSize: '12px', color: '#6D28D9', lineHeight: 1.5 }}>
                    ✓ 已选 {topicKPCodes.length} 个课标知识点,创建后 AI 将按这些知识点的课标难度要求自动适配课件深度
                  </div>
                )}
              </div>
            )}
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '20px' }}>
            <button onClick={() => setCreateMode('select')} style={{ ...btnBase, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary }}>返回</button>
            <button onClick={handleCreateFromTopic} disabled={!topicSubject || !topicGrade || !topicName.trim() || creating} style={{
              ...btnBase, border: 'none',
              background: (topicSubject && topicGrade && topicName.trim()) ? 'linear-gradient(135deg, #7C3AED, #6366F1)' : '#E5E7EB',
              color: (topicSubject && topicGrade && topicName.trim()) ? '#fff' : '#9CA3AF', fontWeight: 600,
              cursor: (topicSubject && topicGrade && topicName.trim() && !creating) ? 'pointer' : 'default',
            }}>{creating ? '创建中...' : '💡 确认创建'}</button>
          </div>
        </>}

        {/* 从 PPT 创建 */}
        {createMode === 'ppt' && <>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
            <button onClick={() => setCreateMode('select')} style={{ background: 'none', border: 'none', fontSize: '14px', color: C.textSecondary, cursor: 'pointer' }}>← 返回</button>
            <div style={{ fontSize: '18px', fontWeight: 700, color: C.textPrimary }}>📊 从 PPT 创建</div>
          </div>
          <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '20px' }}>上传.pptx文件,AI自动提取内容并转化为交互式课件方案</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>PPT文件 *</label>
              <input ref={pptFileRef} type="file" accept=".pptx" onChange={handlePPTFileChange} style={{ display: 'none' }} />
              <div onClick={() => pptFileRef.current?.click()} style={{ padding: '24px', borderRadius: '10px', border: `2px dashed ${pptFile ? '#059669' : C.border}`, background: pptFile ? '#F0FDF4' : '#FAFAFA', cursor: 'pointer', textAlign: 'center', transition: 'all 200ms' }}>
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
                    <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>仅支持.pptx格式,最大50MB</div>
                  </div>
                )}
              </div>
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>{profile.subject_label} *</label>
              <select value={pptSubject} onChange={e => setPptSubject(e.target.value)} style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: '#fff' }}>
                <option value="">请选择学科</option>
                {subjects.map(s => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>{profile.grade_label} *</label>
              <select
                value={pptGrade}
                onChange={e => setPptGrade(e.target.value)}
                style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: '#fff' }}
              >
                <option value="">
                  请选择{profile.grade_label}
                </option>

                {levelOptions.map(option => (
                  <option
                    key={option.value}
                    value={option.value}
                  >
                    {option.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>课件标题(可选,默认使用PPT文件名)</label>
              <input value={pptTitle} onChange={e => setPptTitle(e.target.value)} placeholder="留空则使用PPT文件名作为课件标题"
                style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
            </div>
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '20px' }}>
            <button onClick={() => setCreateMode('select')} style={{ ...btnBase, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary }}>返回</button>
            <button onClick={handleCreateFromPPT} disabled={!pptFile || !pptSubject || !pptGrade || creating} style={{
              ...btnBase, border: 'none',
              background: (pptFile && pptSubject && pptGrade) ? 'linear-gradient(135deg, #D97706, #F59E0B)' : '#E5E7EB',
              color: (pptFile && pptSubject && pptGrade) ? '#fff' : '#9CA3AF', fontWeight: 600,
              cursor: (pptFile && pptSubject && pptGrade && !creating) ? 'pointer' : 'default',
            }}>{creating ? '⏳ 上传解析中...' : '📊 上传并创建'}</button>
          </div>
        </>}

        {/* 从 Word 文档创建 */}
        {createMode === 'doc' && <>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
            <button onClick={() => setCreateMode('select')} style={{ background: 'none', border: 'none', fontSize: '14px', color: C.textSecondary, cursor: 'pointer' }}>← 返回</button>
            <div style={{ fontSize: '18px', fontWeight: 700, color: C.textPrimary }}>📄 从教案文档创建</div>
          </div>
          <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '20px' }}>上传.docx教案文件,AI自动提取内容并转化为交互式课件方案</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>教案文档 *</label>
              <input ref={docFileRef} type="file" accept=".docx" onChange={handleDocFileChange} style={{ display: 'none' }} />
              <div onClick={() => docFileRef.current?.click()} style={{ padding: '24px', borderRadius: '10px', border: `2px dashed ${docFile ? '#059669' : C.border}`, background: docFile ? '#F0FDF4' : '#FAFAFA', cursor: 'pointer', textAlign: 'center' }}>
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
                    <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>仅支持.docx格式,最大30MB</div>
                  </div>
                )}
              </div>
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>{profile.subject_label} *</label>
              <select value={docSubject} onChange={e => setDocSubject(e.target.value)} style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: '#fff' }}>
                <option value="">请选择学科</option>
                {subjects.map(s => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>{profile.grade_label} *</label>
              <select
                value={docGrade}
                onChange={e => setDocGrade(e.target.value)}
                style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: '#fff' }}
              >
                <option value="">
                  请选择{profile.grade_label}
                </option>

                {levelOptions.map(option => (
                  <option
                    key={option.value}
                    value={option.value}
                  >
                    {option.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>课件标题(可选,默认使用文档文件名)</label>
              <input value={docTitle} onChange={e => setDocTitle(e.target.value)} placeholder="留空则使用文档文件名作为课件标题"
                style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
            </div>
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '20px' }}>
            <button onClick={() => setCreateMode('select')} style={{ ...btnBase, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary }}>返回</button>
            <button onClick={handleCreateFromDoc} disabled={!docFile || !docSubject || !docGrade || creating} style={{
              ...btnBase, border: 'none',
              background: (docFile && docSubject && docGrade) ? 'linear-gradient(135deg, #0891B2, #06B6D4)' : '#E5E7EB',
              color: (docFile && docSubject && docGrade) ? '#fff' : '#9CA3AF', fontWeight: 600,
              cursor: (docFile && docSubject && docGrade && !creating) ? 'pointer' : 'default',
            }}>{creating ? '⏳ 上传解析中...' : '📄 上传并创建'}</button>
          </div>
        </>}

        {/* 3D 互动单页创建 */}
        {createMode === '3d_single' && <>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
            <button onClick={() => setCreateMode('select')} style={{ background: 'none', border: 'none', fontSize: '14px', color: C.textSecondary, cursor: 'pointer' }}>← 返回</button>
            <div style={{ fontSize: '18px', fontWeight: 700, color: C.textPrimary }}>🎮 3D 互动单页</div>
            <span style={{ padding: '2px 8px', borderRadius: '8px', fontSize: '11px', color: '#DC2626', background: '#FEE2E2', fontWeight: 600 }}>NEW</span>
          </div>
          <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '12px' }}>AI 将生成一个基于 Three.js 的 3D 沉浸式互动页面,包含粒子系统、分步骤演示和3D模型交互</div>
          <div style={{ padding: '12px', borderRadius: '10px', background: '#FEF3C7', border: '1px solid #FDE68A', fontSize: '13px', color: '#92400E', marginBottom: '16px' }}>
            💡 提示:详细描述越具体,AI 生成的 3D 效果越精细。建议描述清楚要展示的对象、过程和关键知识点。
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>{profile.subject_label} *</label>
              <select value={threeDSubject} onChange={e => setThreeDSubject(e.target.value)} style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: '1px solid ' + C.border, fontSize: '14px', outline: 'none', background: '#fff' }}>
                <option value="">请选择学科</option>
                {subjects.map(s => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>{profile.grade_label} *</label>
              <select
                value={threeDGrade}
                onChange={e => setThreeDGrade(e.target.value)}
                style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: '1px solid ' + C.border, fontSize: '14px', outline: 'none', background: '#fff' }}
              >
                <option value="">
                  请选择{profile.grade_label}
                </option>

                {levelOptions.map(option => (
                  <option
                    key={option.value}
                    value={option.value}
                  >
                    {option.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>3D 主题名称 *</label>
              <input value={threeDTopic} onChange={e => setThreeDTopic(e.target.value)} placeholder="如:水循环、光合作用、细胞结构、太阳系运行"
                style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: '1px solid ' + C.border, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
            </div>
            <div>
              <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px', display: 'block' }}>详细描述 *(至少20字)</label>
              <textarea value={threeDDesc} onChange={e => setThreeDDesc(e.target.value)}
                placeholder="详细描述你想要展示的3D场景,例如:&#10;• 要展示哪些具体物体/结构?&#10;• 有哪些关键过程/步骤需要分步演示?&#10;• 有没有特殊的视觉效果需求?&#10;&#10;示例:展示植物细胞的完整结构,包括细胞壁、细胞膜、细胞核、叶绿体、线粒体等细胞器。需要能点击选择各细胞器查看详细说明,支持透视模式看清内部结构。"
                rows={5}
                style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: '1px solid ' + C.border, fontSize: '14px', outline: 'none', resize: 'vertical', boxSizing: 'border-box' }} />
              <div style={{ fontSize: '12px', color: threeDDesc.trim().length >= 20 ? '#059669' : '#9CA3AF', marginTop: '4px' }}>
                {threeDDesc.trim().length} / 20 字(最少)
              </div>
            </div>
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '20px' }}>
            <button onClick={() => setCreateMode('select')} style={{ ...btnBase, border: '1px solid ' + C.border, background: 'transparent', color: C.textSecondary }}>返回</button>
            <button onClick={handleCreateFrom3D} disabled={!threeDSubject || !threeDGrade || !threeDTopic.trim() || threeDDesc.trim().length < 20 || creating} style={{
              ...btnBase, border: 'none',
              background: (threeDSubject && threeDGrade && threeDTopic.trim() && threeDDesc.trim().length >= 20) ? 'linear-gradient(135deg, #DC2626, #EF4444)' : '#E5E7EB',
              color: (threeDSubject && threeDGrade && threeDTopic.trim() && threeDDesc.trim().length >= 20) ? '#fff' : '#9CA3AF', fontWeight: 600,
              cursor: (threeDSubject && threeDGrade && threeDTopic.trim() && threeDDesc.trim().length >= 20 && !creating) ? 'pointer' : 'default',
            }}>{creating ? '创建中...' : '🎮 创建3D课件'}</button>
          </div>
        </>}
      </div>
    </div>
  )
}

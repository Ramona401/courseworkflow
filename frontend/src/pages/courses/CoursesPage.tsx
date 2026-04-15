/**
 * 课程管理页面
 * v36新增：
 *   - 课程表格新增「同步时间」列，显示索引最后拉取时间
 *   - 课程编号可点击，弹出索引摘要弹窗（页数/字数/页面标题列表）
 *   - 同步时间超过7天标橙色警示，提醒及时重新拉取
 */
import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '@/store/auth'
import {
  getCourses, getOSSCatalog, fetchCourseIndex, getIndexSummary,
  type CourseListItem, type OSSModuleItem, type IndexSummary,
} from '@/api/courses'
import client from '@/api/client'
import {
  BookOpen, Download, RefreshCw, Plus, Search,
  CheckCircle, AlertCircle, Zap, Square, CheckSquare, FileText, X,
} from 'lucide-react'

// ==================== Toast 提示组件 ====================

function Toast({ message, type, onClose }: { message: string; type: 'ok' | 'err' | 'info'; onClose: () => void }) {
  useEffect(() => { const t = setTimeout(onClose, 5000); return () => clearTimeout(t) }, [onClose])
  const bg = type === 'ok' ? '#34c759' : type === 'err' ? '#ff3b30' : '#007aff'
  return (
    <div style={{ position: 'fixed', bottom: 24, right: 24, background: bg, color: '#fff', padding: '12px 22px', borderRadius: 12, fontSize: 13, fontWeight: 500, zIndex: 9999, boxShadow: '0 4px 24px rgba(0,0,0,0.18)', maxWidth: 450 }}>
      {message}
    </div>
  )
}

// ==================== 工具函数 ====================

/**
 * formatFetchedAt 格式化索引同步时间
 * 返回格式：MM-DD HH:mm，以及距今天数（超过7天标橙色）
 */
function formatFetchedAt(isoStr: string | null): { text: string; daysAgo: number } {
  if (!isoStr) return { text: '未拉取', daysAgo: 999 }
  const d = new Date(isoStr)
  if (isNaN(d.getTime())) return { text: '未知', daysAgo: 999 }

  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const daysAgo = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  // 格式化为 MM-DD HH:mm
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  const text = mm + '-' + dd + ' ' + hh + ':' + min

  return { text, daysAgo }
}

// ==================== 索引摘要弹窗组件 ====================

function IndexSummaryModal({
  courseCode,
  courseName,
  onClose,
}: {
  courseCode: string
  courseName: string
  onClose: () => void
}) {
  const [summary, setSummary] = useState<IndexSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
     
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true)
    getIndexSummary(courseCode)
      .then(data => { setSummary(data); setLoading(false) })
      .catch(e => { setError(e.message || '加载失败'); setLoading(false) })
  }, [courseCode])

  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', zIndex: 2000, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div style={{ background: '#fff', borderRadius: 20, width: 620, maxWidth: '94vw', maxHeight: '85vh', overflow: 'hidden', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>

        {/* 弹窗头部 */}
        <div style={{ padding: '20px 24px 16px', borderBottom: '1px solid rgba(0,0,0,0.06)', display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <div>
            <div style={{ fontSize: 16, fontWeight: 700, color: '#1c1c1e' }}>
              索引摘要 — {courseCode}
            </div>
            <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 3 }}>{courseName}</div>
          </div>
          <button
            onClick={onClose}
            style={{ padding: '6px 10px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', cursor: 'pointer', display: 'flex', alignItems: 'center' }}
          >
            <X size={16} color="#8e8e93" />
          </button>
        </div>

        {/* 弹窗内容 */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '20px 24px' }}>
          {loading ? (
            <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>加载中...</div>
          ) : error ? (
            <div style={{ textAlign: 'center', padding: 40, color: '#ff3b30' }}>{error}</div>
          ) : !summary || !summary.has_index ? (
            <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>该课程尚未拉取索引</div>
          ) : (
            <>
              {/* 统计信息 */}
              <div style={{ display: 'flex', gap: 12, marginBottom: 20 }}>
                <div style={{ flex: 1, background: 'rgba(0,122,255,0.06)', borderRadius: 12, padding: '14px 18px', border: '1px solid rgba(0,122,255,0.12)' }}>
                  <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 600, marginBottom: 4 }}>总页数</div>
                  <div style={{ fontSize: 26, fontWeight: 700, color: '#007aff' }}>{summary.page_count}</div>
                  <div style={{ fontSize: 11, color: '#8e8e93', marginTop: 2 }}>页</div>
                </div>
                <div style={{ flex: 1, background: 'rgba(52,199,89,0.06)', borderRadius: 12, padding: '14px 18px', border: '1px solid rgba(52,199,89,0.12)' }}>
                  <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 600, marginBottom: 4 }}>索引大小</div>
                  <div style={{ fontSize: 26, fontWeight: 700, color: '#34c759' }}>
                    {summary.total_length > 1000
                      ? (summary.total_length / 1000).toFixed(1) + 'K'
                      : summary.total_length}
                  </div>
                  <div style={{ fontSize: 11, color: '#8e8e93', marginTop: 2 }}>字符</div>
                </div>
              </div>

              {/* 页面标题列表 */}
              {summary.page_titles && summary.page_titles.length > 0 ? (
                <>
                  <div style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e', marginBottom: 10 }}>
                    页面目录（{summary.page_titles.length} 页）
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                    {summary.page_titles.map((title, i) => (
                      <div key={i} style={{
                        display: 'flex', alignItems: 'center', gap: 10,
                        padding: '8px 12px', borderRadius: 8,
                        background: i % 2 === 0 ? 'rgba(0,0,0,0.02)' : 'transparent',
                      }}>
                        <span style={{ fontSize: 11, color: '#aeaeb2', fontFamily: 'monospace', minWidth: 28, textAlign: 'right' }}>
                          P{String(i + 1).padStart(2, '0')}
                        </span>
                        <span style={{ fontSize: 13, color: '#3c3c43', flex: 1 }}>{title}</span>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <div style={{ color: '#8e8e93', fontSize: 13, textAlign: 'center', padding: 20 }}>
                  索引中没有页面标题数据
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

// ==================== 主页面组件 ====================

export default function CoursesPage() {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'
  const [courses, setCourses] = useState<CourseListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [showCatalog, setShowCatalog] = useState(false)
  const [catalogModules, setCatalogModules] = useState<OSSModuleItem[]>([])
  const [catalogLoading, setCatalogLoading] = useState(false)
  const [catalogSearch, setCatalogSearch] = useState('')
  const [fetchingIndex, setFetchingIndex] = useState<string | null>(null)
  const [batchRunning, setBatchRunning] = useState(false)
  const [registeringId, setRegisteringId] = useState<number | null>(null)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [batchRegProgress, setBatchRegProgress] = useState('')
  const [toast, setToast] = useState<{ message: string; type: 'ok' | 'err' | 'info' } | null>(null)
  // v36新增：索引摘要弹窗状态
  const [summaryModal, setSummaryModal] = useState<{ code: string; name: string } | null>(null)

  const loadCourses = useCallback(async () => {
    setLoading(true)
     
    try { const data = await getCourses(); setCourses(data.courses || []) }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    catch (e: any) { setToast({ message: '加载失败: ' + (e.message || ''), type: 'err' }) }
    setLoading(false)
  }, [])

   
  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { loadCourses() }, [loadCourses])

  const loadCatalog = async () => {
     
    setCatalogLoading(true)
    try { const data = await getOSSCatalog(); setCatalogModules(data.modules || []) }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    catch (e: any) { setToast({ message: '加载OSS目录失败: ' + (e.message || ''), type: 'err' }) }
    setCatalogLoading(false)
  }

  const openCatalog = () => { setShowCatalog(true); setSelectedIds(new Set()); loadCatalog() }

  const extractCode = (mod: OSSModuleItem) => {
    const name = (mod.name || '').trim()
    const m = name.match(/^\s*(G\d+-\d+)/)
    return m ? m[1] : 'M' + mod.id
  }

  const handleRegisterOne = async (mod: OSSModuleItem) => {
    setRegisteringId(mod.id)
    try {
      const res = await client.post('/courses/register-fetch', {
         
        external_module_id: mod.id, course_code: extractCode(mod), course_name: (mod.name || '').trim(),
      })
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const d = (res.data as any).data
       
      setToast({ message: extractCode(mod) + ' 注册成功' + (d?.index_fetched ? ' · ' + d.page_count + '页' : ''), type: 'ok' })
      loadCatalog(); loadCourses()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) { setToast({ message: '注册失败: ' + (e.message || ''), type: 'err' }) }
    setRegisteringId(null)
  }

  const toggleSelect = (id: number) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }

  const selectableModules = catalogModules.filter(m => !m.is_registered && m.has_index)
  const filteredSelectable = selectableModules.filter(m => {
    if (!catalogSearch) return true
    const q = catalogSearch.trim().toLowerCase()
    const name = (m.name || '').trim().toLowerCase()
    return name.includes(q) || String(m.id).includes(q)
  })

  const toggleSelectAll = () => {
    const ids = filteredSelectable.map(m => m.id)
    const allSelected = ids.every(id => selectedIds.has(id))
    if (allSelected) {
      setSelectedIds(prev => { const next = new Set(prev); ids.forEach(id => next.delete(id)); return next })
    } else {
      setSelectedIds(prev => { const next = new Set(prev); ids.forEach(id => next.add(id)); return next })
    }
  }

  const handleBatchRegisterSelected = async () => {
    const mods = catalogModules.filter(m => selectedIds.has(m.id) && !m.is_registered && m.has_index)
    if (mods.length === 0) { setToast({ message: '请先勾选要注册的模块', type: 'info' }); return }
    if (!confirm('确认注册并拉取索引：' + mods.length + ' 个模块？')) return

    setBatchRunning(true)
    let ok = 0, fail = 0
    for (let i = 0; i < mods.length; i++) {
      const mod = mods[i]
      setBatchRegProgress('(' + (i + 1) + '/' + mods.length + ') ' + extractCode(mod) + '...')
      try {
        await client.post('/courses/register-fetch', {
          external_module_id: mod.id, course_code: extractCode(mod), course_name: (mod.name || '').trim(),
        })
        ok++
      } catch { fail++ }
    }
    setBatchRunning(false)
    setBatchRegProgress('')
    setSelectedIds(new Set())
    setToast({ message: '批量注册完成: 成功' + ok + '个' + (fail > 0 ? ', 失败' + fail + '个' : ''), type: fail > 0 ? 'info' : 'ok' })
    loadCatalog(); loadCourses()
  }

  const handleBatchFetch = async () => {
    if (!confirm('确认对所有已注册课程重新拉取索引？')) return
    setBatchRunning(true)
    setToast({ message: '批量拉取索引中...', type: 'info' })
     
    try {
      const res = await client.post('/courses/batch-fetch')
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const d = (res.data as any).data
       
      setToast({ message: '批量拉取完成: 成功' + d.success + '个' + (d.failed > 0 ? ', 失败' + d.failed + '个' : ''), type: d.failed > 0 ? 'info' : 'ok' })
      loadCourses()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) { setToast({ message: '批量拉取失败: ' + (e.message || ''), type: 'err' }) }
    setBatchRunning(false)
  }

  const handleFetchIndex = async (code: string) => {
    setFetchingIndex(code)
    try {
      const r = await fetchCourseIndex(code)
       
      setToast({ message: code + ' 索引拉取成功: ' + r.page_count + '页', type: 'ok' })
      loadCourses()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) { setToast({ message: '拉取失败: ' + (e.message || ''), type: 'err' }) }
    setFetchingIndex(null)
  }

  const totalCourses = courses.length
  const withIndex = courses.filter(c => c.has_index).length
  const withoutIndex = courses.filter(c => !c.has_index).length
  // v36新增：统计同步时间超过7天的课程数量
  const staleCount = courses.filter(c => {
    const { daysAgo } = formatFetchedAt(c.index_fetched_at)
    return c.has_index && daysAgo >= 7
  }).length

  const filteredModules = catalogModules.filter(m => {
    if (!catalogSearch) return true
    const q = catalogSearch.trim().toLowerCase()
    const name = (m.name || '').trim().toLowerCase()
    return name.includes(q) || String(m.id).includes(q) || (m.course_code || '').toLowerCase().includes(q)
  })

  const selectedCount = [...selectedIds].filter(id => {
    const m = catalogModules.find(mod => mod.id === id)
    return m && !m.is_registered && m.has_index
  }).length

  const card: React.CSSProperties = { background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)', borderRadius: 16, padding: 20, marginBottom: 16 }
  const stat: React.CSSProperties = { background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)', borderRadius: 14, padding: '16px 20px', flex: 1, minWidth: 140 }
  const btn: React.CSSProperties = { padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6, transition: 'all 0.15s ease' }
  const btnP: React.CSSProperties = { ...btn, background: '#007aff', color: '#fff', border: '1px solid #007aff' }
  const btnG: React.CSSProperties = { ...btn, background: '#34c759', color: '#fff', border: '1px solid #34c759' }
  const btnO: React.CSSProperties = { ...btn, background: '#ff9500', color: '#fff', border: '1px solid #ff9500' }

  return (
    <div>
      {/* 统计卡片 */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap' }}>
        <div style={stat}>
          <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 600, marginBottom: 4 }}>已注册课程</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: '#1c1c1e' }}>{totalCourses}</div>
        </div>
        <div style={stat}>
          <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 600, marginBottom: 4 }}>已有索引</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: '#34c759' }}>{withIndex}</div>
        </div>
        <div style={stat}>
          <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 600, marginBottom: 4 }}>待拉取索引</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: '#ff9500' }}>{withoutIndex}</div>
        </div>
        {/* v36新增：超过7天未更新的课程数 */}
        {staleCount > 0 && (
          <div style={{ ...stat, border: '1px solid rgba(255,149,0,0.25)', background: 'rgba(255,149,0,0.05)' }}>
            <div style={{ fontSize: 11, color: '#ff9500', fontWeight: 600, marginBottom: 4 }}>索引超7天未更新</div>
            <div style={{ fontSize: 28, fontWeight: 700, color: '#ff9500' }}>{staleCount}</div>
            <div style={{ fontSize: 11, color: '#8e8e93', marginTop: 2 }}>建议重新拉取</div>
          </div>
        )}
      </div>

      {/* 操作栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, flexWrap: 'wrap', gap: 8 }}>
        <div style={{ fontSize: 13, color: '#8e8e93' }}>{loading ? '加载中...' : totalCourses + ' 个课程'}</div>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          <button style={btn} onClick={loadCourses}><RefreshCw size={14} /> 刷新</button>
          {isAdmin && courses.length > 0 && (
            <button style={btnO} onClick={handleBatchFetch} disabled={batchRunning}>
              <Download size={14} /> {batchRunning ? '处理中...' : '批量拉取索引'}
            </button>
          )}
          {isAdmin && <button style={btnP} onClick={openCatalog}><Plus size={14} /> 从OSS注册课程</button>}
        </div>
      </div>

      {/* 课程表格 */}
      <div style={card}>
        {loading ? (
          <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>加载中...</div>
        ) : courses.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <BookOpen size={40} style={{ color: '#c7c7cc', marginBottom: 12 }} />
            <div style={{ color: '#8e8e93', fontSize: 14 }}>暂无课程</div>
            {isAdmin && (
              <div style={{ color: '#007aff', fontSize: 13, marginTop: 8, cursor: 'pointer' }} onClick={openCatalog}>
                点击从OSS目录注册 →
              </div>
            )}
          </div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead>
              <tr style={{ borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
                {['课程编号', '课程名称', '模块ID', '索引状态', '页面数', '同步时间'].map(h => (
                  <th key={h} style={{
                    textAlign: (h === '课程编号' || h === '课程名称') ? 'left' : 'center',
                    padding: '10px 12px', color: '#8e8e93', fontWeight: 600, fontSize: 11,
                  }}>{h}</th>
                ))}
                {isAdmin && (
                  <th style={{ textAlign: 'center', padding: '10px 12px', color: '#8e8e93', fontWeight: 600, fontSize: 11 }}>操作</th>
                )}
              </tr>
            </thead>
            <tbody>
              {courses.map(c => {
                const { text: fetchedText, daysAgo } = formatFetchedAt(c.index_fetched_at)
                // 超过7天标橙色，未拉取标灰色
                const timeColor = !c.has_index ? '#c7c7cc' : daysAgo >= 7 ? '#ff9500' : '#34c759'
                const timeTitle = c.has_index
                  ? (daysAgo === 0 ? '今天同步' : daysAgo + '天前同步' + (daysAgo >= 7 ? '（建议重新拉取）' : ''))
                  : '尚未拉取索引'

                return (
                  <tr key={c.id} style={{ borderBottom: '1px solid rgba(0,0,0,0.04)' }}>
                    {/* 课程编号：可点击查看索引摘要 */}
                    <td style={{ padding: 12 }}>
                      <span
                        onClick={() => c.has_index && setSummaryModal({ code: c.course_code, name: c.course_name })}
                        style={{
                          fontWeight: 600, fontFamily: 'monospace',
                          color: c.has_index ? '#007aff' : '#8e8e93',
                          cursor: c.has_index ? 'pointer' : 'default',
                          textDecoration: c.has_index ? 'underline' : 'none',
                          textDecorationStyle: 'dotted',
                          textUnderlineOffset: 3,
                        }}
                        title={c.has_index ? '点击查看索引摘要' : '尚未拉取索引'}
                      >
                        {c.course_code}
                      </span>
                    </td>

                    <td style={{ padding: 12, color: '#1c1c1e' }}>{c.course_name}</td>

                    <td style={{ padding: 12, textAlign: 'center', color: '#8e8e93', fontFamily: 'monospace', fontSize: 12 }}>
                      {c.external_module_id || '-'}
                    </td>

                    <td style={{ padding: 12, textAlign: 'center' }}>
                      {c.has_index
                        ? <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, color: '#34c759', fontSize: 12 }}><CheckCircle size={14} /> 已就绪</span>
                        : <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, color: '#ff9500', fontSize: 12 }}><AlertCircle size={14} /> 待拉取</span>
                      }
                    </td>

                    <td style={{ padding: 12, textAlign: 'center', fontFamily: 'monospace', fontSize: 12, color: c.has_index ? '#1c1c1e' : '#c7c7cc' }}>
                      {c.has_index
                        ? c.index_page_count + '页 / ' + (c.index_total_length > 1000 ? (c.index_total_length / 1000).toFixed(1) + 'K' : c.index_total_length + '字')
                        : '-'
                      }
                    </td>

                    {/* v36新增：同步时间列 */}
                    <td style={{ padding: 12, textAlign: 'center' }}>
                      <span
                        style={{ fontSize: 12, color: timeColor, fontFamily: 'monospace', cursor: 'default' }}
                        title={timeTitle}
                      >
                        {fetchedText}
                        {c.has_index && daysAgo >= 7 && (
                          <span style={{ marginLeft: 4, fontSize: 11 }}>⚠️</span>
                        )}
                      </span>
                    </td>

                    {isAdmin && (
                      <td style={{ padding: 12, textAlign: 'center' }}>
                        <div style={{ display: 'flex', gap: 6, justifyContent: 'center' }}>
                          {/* 查看索引摘要按钮（有索引时显示） */}
                          {c.has_index && (
                            <button
                              style={{ ...btn, fontSize: 12, padding: '5px 10px', color: '#5856d6', borderColor: 'rgba(88,86,214,0.2)' }}
                              onClick={() => setSummaryModal({ code: c.course_code, name: c.course_name })}
                              title="查看索引摘要"
                            >
                              <FileText size={12} />
                            </button>
                          )}
                          {/* 拉取索引按钮 */}
                          <button
                            style={{ ...btn, fontSize: 12, padding: '5px 12px', color: '#007aff', borderColor: 'rgba(0,122,255,0.2)' }}
                            onClick={() => handleFetchIndex(c.course_code)}
                            disabled={fetchingIndex === c.course_code}
                          >
                            {fetchingIndex === c.course_code
                              ? <><RefreshCw size={12} style={{ animation: 'spin 1s linear infinite' }} /> 拉取中...</>
                              : <><Download size={12} /> {c.has_index ? '重新拉取' : '拉取索引'}</>
                            }
                          </button>
                        </div>
                      </td>
                    )}
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* OSS目录弹窗（带多选） */}
      {showCatalog && (
        <div
          style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
          onClick={e => { if (e.target === e.currentTarget) setShowCatalog(false) }}
        >
          <div style={{ background: '#fff', borderRadius: 20, width: 800, maxWidth: '94vw', maxHeight: '90vh', overflow: 'hidden', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>

            {/* 弹窗头部 */}
            <div style={{ padding: '20px 24px 16px', borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <div style={{ fontSize: 17, fontWeight: 700, color: '#1c1c1e' }}>OSS课程目录</div>
                  <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 2 }}>
                    {catalogLoading ? '加载中...' : catalogModules.length + ' 个模块 · ' + catalogModules.filter(m => m.has_index).length + ' 有索引 · ' + selectableModules.length + ' 可注册'}
                  </div>
                </div>
                <button style={{ ...btn, fontSize: 12 }} onClick={() => setShowCatalog(false)}>关闭</button>
              </div>

              {/* 搜索 + 操作按钮 */}
              <div style={{ display: 'flex', gap: 8, marginTop: 12, alignItems: 'center' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, background: '#f5f5f7', borderRadius: 10, padding: '8px 12px', flex: 1 }}>
                  <Search size={14} style={{ color: '#8e8e93' }} />
                  <input
                    type="text"
                    placeholder="搜索课程编号或名称（如 G1-02）..."
                    value={catalogSearch}
                    onChange={e => setCatalogSearch(e.target.value)}
                    style={{ border: 'none', background: 'transparent', outline: 'none', fontSize: 13, flex: 1, color: '#1c1c1e' }}
                  />
                </div>
                {!catalogLoading && filteredSelectable.length > 0 && (
                  <button style={{ ...btn, fontSize: 12, padding: '7px 12px', color: '#007aff' }} onClick={toggleSelectAll}>
                    {filteredSelectable.every(m => selectedIds.has(m.id))
                      ? <><CheckSquare size={13} /> 取消全选</>
                      : <><Square size={13} /> 全选可注册</>
                    }
                  </button>
                )}
              </div>

              {/* 选中后的操作条 */}
              {selectedCount > 0 && (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 10, padding: '10px 14px', background: 'rgba(0,122,255,0.06)', borderRadius: 10, border: '1px solid rgba(0,122,255,0.15)' }}>
                  <span style={{ fontSize: 13, color: '#007aff', fontWeight: 600 }}>
                    已选 {selectedCount} 个模块
                    {batchRegProgress && <span style={{ fontWeight: 400, marginLeft: 8 }}>{batchRegProgress}</span>}
                  </span>
                  <button
                    style={{ ...btnG, fontSize: 12, padding: '6px 16px' }}
                    onClick={handleBatchRegisterSelected}
                    disabled={batchRunning}
                  >
                    {batchRunning
                      ? <><RefreshCw size={12} style={{ animation: 'spin 1s linear infinite' }} /> 注册中...</>
                      : <><Zap size={12} /> 批量注册+拉取</>
                    }
                  </button>
                </div>
              )}
            </div>

            {/* 模块列表 */}
            <div style={{ flex: 1, overflowY: 'auto', padding: '4px 24px 24px' }}>
              {catalogLoading ? (
                <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>加载OSS目录中...</div>
              ) : filteredModules.length === 0 ? (
                <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>无匹配模块</div>
              ) : (
                filteredModules.map(mod => {
                  const canRegister = !mod.is_registered && mod.has_index
                  const isSelected = selectedIds.has(mod.id)
                  return (
                    <div
                      key={mod.id}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 12,
                        padding: '11px 14px', borderRadius: 12, marginTop: 5,
                        background: isSelected ? 'rgba(0,122,255,0.06)' : mod.is_registered ? 'rgba(52,199,89,0.05)' : 'rgba(0,0,0,0.015)',
                        border: '1px solid ' + (isSelected ? 'rgba(0,122,255,0.2)' : mod.is_registered ? 'rgba(52,199,89,0.12)' : 'rgba(0,0,0,0.04)'),
                        cursor: canRegister ? 'pointer' : 'default',
                      }}
                      onClick={() => { if (canRegister) toggleSelect(mod.id) }}
                    >
                      {/* 多选框 */}
                      {canRegister ? (
                        <div style={{ flexShrink: 0 }}>
                          {isSelected
                            ? <CheckSquare size={18} style={{ color: '#007aff' }} />
                            : <Square size={18} style={{ color: '#c7c7cc' }} />
                          }
                        </div>
                      ) : (
                        <div style={{ width: 18, flexShrink: 0 }} />
                      )}

                      {/* 模块信息 */}
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {(mod.name || '').trim()}
                        </div>
                        <div style={{ fontSize: 11, color: '#8e8e93', marginTop: 2, display: 'flex', gap: 12 }}>
                          <span>ID: {mod.id}</span>
                          <span>{mod.lesson_count} 课时</span>
                          {mod.has_index
                            ? <span style={{ color: '#34c759' }}>有索引</span>
                            : <span style={{ color: '#c7c7cc' }}>无索引</span>
                          }
                        </div>
                      </div>

                      {/* 操作按钮 */}
                      <div style={{ flexShrink: 0 }} onClick={e => e.stopPropagation()}>
                        {mod.is_registered ? (
                          <span style={{ fontSize: 12, color: '#34c759', fontWeight: 500, display: 'flex', alignItems: 'center', gap: 4 }}>
                            <CheckCircle size={14} /> 已注册 ({mod.course_code})
                          </span>
                        ) : mod.has_index ? (
                          <button
                            style={{ ...btnP, fontSize: 12, padding: '5px 12px' }}
                            onClick={() => handleRegisterOne(mod)}
                            disabled={registeringId === mod.id || batchRunning}
                          >
                            {registeringId === mod.id
                              ? <><RefreshCw size={12} style={{ animation: 'spin 1s linear infinite' }} /> 注册中...</>
                              : <><Zap size={12} /> 注册+拉取</>
                            }
                          </button>
                        ) : (
                          <span style={{ fontSize: 11, color: '#c7c7cc' }}>无索引</span>
                        )}
                      </div>
                    </div>
                  )
                })
              )}
            </div>
          </div>
        </div>
      )}

      {/* v36新增：索引摘要弹窗 */}
      {summaryModal && (
        <IndexSummaryModal
          courseCode={summaryModal.code}
          courseName={summaryModal.name}
          onClose={() => setSummaryModal(null)}
        />
      )}

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      {/* 旋转动画 */}
      <style>{`
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  )
}

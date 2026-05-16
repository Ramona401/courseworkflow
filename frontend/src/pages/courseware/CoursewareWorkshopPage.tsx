/**
 * 课件工坊主页面 — CoursewareWorkshopPage.tsx v1.2 (Phase 3.5)
 *
 * 六步流程步骤式页面：
 *   Step 1: AI生成方案（教案→每页课件需求方案）
 *   Step 2: 确认方案（人工调整/增删/排序）
 *   Step 3-6: 占位（Phase 4/5/6）
 *
 * v1.2: 两层索引架构 — 传递isAdmin给IndexEditor，admin可展开AOCI索引
 */
import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  getCourseware, generateCWIndex, subscribeCWIndexSSE,
  confirmCWIndex, CW_STATUS_CONFIG,
} from '@/api/coursewares'
import type { CoursewareDetail, CoursewarePage } from '@/api/coursewares'
import IndexEditor from './components/IndexEditor'
import { useAuth } from '@/store/auth'

// ==================== 颜色常量 ====================
const C = {
  primary: '#F59E0B', primaryBg: 'rgba(245,158,11,0.08)',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
  border: '#E5E7EB', success: '#059669', white: '#fff',
}

// ==================== 步骤定义 ====================
const STEPS = [
  { key: 'generate', label: 'AI生成方案', emoji: '🤖', desc: 'AI分析教案内容，自动生成课件页面方案' },
  { key: 'edit',     label: '确认方案',   emoji: '✏️', desc: '确认并调整每页课件的内容和交互设计' },
  { key: 'style',    label: '选择风格',   emoji: '🎨', desc: '为课件选择视觉风格和配色方案' },
  { key: 'build',    label: 'AI生成课件', emoji: '⚡', desc: 'AI根据方案和风格生成完整课件' },
  { key: 'media',    label: '多媒体填充', emoji: '🖼️', desc: '为课件中的图片/视频占位符生成素材' },
  { key: 'confirm',  label: '确认提交',   emoji: '✅', desc: '预览课件效果，确认后提交审核' },
]

// ==================== 状态到步骤映射 ====================
function statusToStep(status: string): number {
  switch (status) {
    case 'draft': return 0
    case 'indexing': return 1
    case 'styling': return 2
    case 'generating': return 3
    case 'preview': return 4
    case 'confirmed': return 5
    case 'in_pipeline': return 5
    default: return 0
  }
}

export default function CoursewareWorkshopPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const [courseware, setCourseware] = useState<CoursewareDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [activeStep, setActiveStep] = useState(0)
  const [pages, setPages] = useState<CoursewarePage[]>([])
  const [generating, setGenerating] = useState(false)
  const [sseMessage, setSseMessage] = useState('')
  const [confirming, setConfirming] = useState(false)
  const sseRef = useRef<{ close: () => void } | null>(null)

  // ==================== 加载课件详情 ====================
  useEffect(() => {
    if (!id) return
    loadCourseware()
    return () => { sseRef.current?.close() }
  }, [id])

  const loadCourseware = async () => {
    if (!id) return
    setLoading(true)
    try {
      const detail = await getCourseware(id)
      setCourseware(detail)
      setPages(detail.pages || [])
      const step = statusToStep(detail.status)
      setActiveStep(step)
    } catch {
      alert('加载课件失败')
      navigate('/courseware')
    } finally { setLoading(false) }
  }

  // ==================== AI生成课件方案 ====================
  const handleGenerate = async () => {
    if (!id) return
    setGenerating(true)
    setSseMessage('正在启动AI方案生成...')
    setPages([])

    try {
      await generateCWIndex(id)

      sseRef.current?.close()
      sseRef.current = subscribeCWIndexSSE(id, {
        onConnected: () => setSseMessage('已连接，正在分析教案内容...'),
        onIndexStart: (data) => setSseMessage(String((data as Record<string, unknown>).message || '开始分析教案...')),
        onIndexProgress: (data) => setSseMessage(String((data as Record<string, unknown>).message || '正在整理方案...')),
        onIndexPage: (page) => {
          setPages(prev => {
            const exists = prev.some(p => p.page_number === page.page_number)
            return exists ? prev.map(p => p.page_number === page.page_number ? page : p) : [...prev, page]
          })
        },
        onIndexDone: (data) => {
          setSseMessage(`✅ ${data.message}`)
          setGenerating(false)
          setActiveStep(1)
          loadCourseware()
        },
        onError: (data) => {
          setSseMessage(`❌ ${data.message}`)
          setGenerating(false)
        },
      })
    } catch {
      setSseMessage('❌ 启动方案生成失败')
      setGenerating(false)
    }
  }

  // ==================== SSE断开兜底：定时检查生成是否已完成 ====================
  useEffect(() => {
    if (!generating || !id) return
    // 每10秒检查一次课件状态，如果已不是indexing/draft则说明生成完成
    const pollTimer = setInterval(async () => {
      try {
        const detail = await getCourseware(id)
        if (detail.status !== 'draft' && detail.status !== 'indexing') {
          // 生成已完成但SSE没收到done事件
          setGenerating(false)
          setCourseware(detail)
          setPages(detail.pages || [])
          setActiveStep(statusToStep(detail.status))
          setSseMessage(`✅ 课件方案生成完成，共 ${detail.page_count} 页`)
          sseRef.current?.close()
        } else if (detail.pages && detail.pages.length > 0 && detail.status === 'indexing') {
          // 状态还是indexing但已有页面数据（两层AI完成写入了数据库但状态未前进）
          // 说明后端已完成生成，SSE没收到done事件
          setGenerating(false)
          setCourseware(detail)
          setPages(detail.pages)
          setActiveStep(1)
          setSseMessage(`✅ 课件方案生成完成，共 ${detail.pages.length} 页`)
          sseRef.current?.close()
        }
      } catch { /* 静默 */ }
    }, 10000)
    return () => clearInterval(pollTimer)
  }, [generating, id])

  // ==================== 确认方案 ====================
  const handleConfirm = async () => {
    if (!id || pages.length === 0) return
    setConfirming(true)
    try {
      await confirmCWIndex(id)
      setSseMessage('✅ 方案确认成功，请选择风格')
      setActiveStep(2)
      loadCourseware()
    } catch { alert('确认方案失败') } finally { setConfirming(false) }
  }

  // ==================== 加载中 ====================
  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '80px 0', color: C.textMuted }}>
        <div style={{ fontSize: '40px', marginBottom: '12px' }}>🎨</div>
        <div>加载课件信息中...</div>
      </div>
    )
  }

  if (!courseware) {
    return (
      <div style={{ textAlign: 'center', padding: '80px 0', color: C.textMuted }}>
        <div>课件不存在</div>
        <button onClick={() => navigate('/courseware')} style={{ marginTop: '12px', color: C.primary, background: 'none', border: 'none', cursor: 'pointer' }}>
          返回课件列表
        </button>
      </div>
    )
  }

  const sc = CW_STATUS_CONFIG[courseware.status] || { label: courseware.status, color: '#6B7280', bg: '#F3F4F6' }

  return (
    <div style={{ maxWidth: '960px', margin: '0 auto' }}>
      {/* ====== 顶部信息栏 ====== */}
      <div style={{ marginBottom: '24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '8px' }}>
          <button onClick={() => navigate('/courseware')} style={{
            background: 'none', border: 'none', fontSize: '14px', color: C.textSecondary, cursor: 'pointer',
          }}>← 返回列表</button>
          <span style={{ padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 500, color: sc.color, background: sc.bg }}>{sc.label}</span>
        </div>
        <h2 style={{ fontSize: '22px', fontWeight: 700, color: C.textPrimary, margin: 0 }}>{courseware.title}</h2>
        <div style={{ fontSize: '13px', color: C.textMuted, marginTop: '4px' }}>
          📝 来源教案：{courseware.lesson_plan_title || '未知'} &nbsp;|&nbsp; 📚 {courseware.subject} &nbsp;|&nbsp; 🎓 {courseware.grade}
        </div>
      </div>

      {/* ====== 步骤进度条 ====== */}
      <div style={{
        display: 'flex', gap: '4px', marginBottom: '28px', padding: '16px 20px',
        background: C.white, borderRadius: '12px', border: `1px solid ${C.border}`,
      }}>
        {STEPS.map((step, i) => {
          const isActive = i === activeStep
          const isDone = i < activeStep
          return (
            <div key={step.key} style={{ flex: 1, textAlign: 'center' }}>
              <div style={{
                width: '32px', height: '32px', borderRadius: '50%', margin: '0 auto 6px',
                display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '16px',
                background: isDone ? C.success : isActive ? C.primary : '#F3F4F6',
                color: isDone || isActive ? '#fff' : C.textMuted,
                fontWeight: 700, transition: 'all 300ms ease',
              }}>
                {isDone ? '✓' : step.emoji}
              </div>
              <div style={{
                fontSize: '12px', fontWeight: isActive ? 600 : 400,
                color: isActive ? C.primary : isDone ? C.success : C.textMuted,
              }}>{step.label}</div>
            </div>
          )
        })}
      </div>

      {/* ====== 步骤内容区 ====== */}
      <div style={{
        background: C.white, borderRadius: '12px', border: `1px solid ${C.border}`,
        padding: '24px', minHeight: '400px',
      }}>
        {/* Step 0: AI生成课件方案 */}
        {activeStep === 0 && (
          <div>
            <h3 style={{ fontSize: '18px', fontWeight: 600, color: C.textPrimary, margin: '0 0 8px' }}>
              🤖 AI生成课件方案
            </h3>
            <p style={{ fontSize: '14px', color: C.textSecondary, margin: '0 0 20px' }}>
              AI将分析教案内容，自动为课件的每一页设计方案——包括知识目标、能力目标、互动设计和内容概要。
            </p>

            {sseMessage && (
              <div style={{
                padding: '12px 16px', borderRadius: '8px', marginBottom: '16px',
                background: sseMessage.startsWith('❌') ? '#FEE2E2' : sseMessage.startsWith('✅') ? '#D1FAE5' : '#FEF3C7',
                color: sseMessage.startsWith('❌') ? '#DC2626' : sseMessage.startsWith('✅') ? '#059669' : '#D97706',
                fontSize: '14px',
              }}>{sseMessage}</div>
            )}

            {generating && pages.length > 0 && (
              <div style={{ marginBottom: '16px' }}>
                <div style={{ fontSize: '13px', color: C.textMuted, marginBottom: '8px' }}>已生成 {pages.length} 页方案...</div>
                <IndexEditor coursewareId={id!} pages={pages} onPagesChange={setPages} isAdmin={isAdmin} indexOverview={courseware?.index_overview} />
              </div>
            )}

            <button onClick={handleGenerate} disabled={generating} style={{
              padding: '12px 32px', borderRadius: '10px', border: 'none',
              background: generating ? '#E5E7EB' : 'linear-gradient(135deg, #F59E0B, #EF4444)',
              color: generating ? '#9CA3AF' : '#fff',
              fontSize: '15px', fontWeight: 600,
              cursor: generating ? 'default' : 'pointer',
              boxShadow: generating ? 'none' : '0 4px 16px rgba(245,158,11,0.3)',
            }}>
              {generating ? '⏳ AI正在生成方案...' : (pages.length > 0 ? '🔄 重新生成方案' : '🤖 开始AI生成方案')}
            </button>

            {!generating && pages.length > 0 && (
              <button onClick={() => setActiveStep(1)} style={{
                marginLeft: '12px', padding: '12px 24px', borderRadius: '10px',
                border: `1px solid ${C.primary}`, background: C.primaryBg, color: C.primary,
                fontSize: '15px', fontWeight: 600, cursor: 'pointer',
              }}>✏️ 确认方案 →</button>
            )}
          </div>
        )}

        {/* Step 1: 确认课件方案 */}
        {activeStep === 1 && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <div>
                <h3 style={{ fontSize: '18px', fontWeight: 600, color: C.textPrimary, margin: 0 }}>✏️ 确认课件方案</h3>
                <p style={{ fontSize: '13px', color: C.textSecondary, margin: '4px 0 0' }}>确认每页内容，可调整页面顺序或修改细节</p>
              </div>
              <div style={{ display: 'flex', gap: '10px' }}>
                <button onClick={() => setActiveStep(0)} style={{
                  padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`,
                  background: 'transparent', color: C.textSecondary, fontSize: '13px', cursor: 'pointer',
                }}>← 重新生成方案</button>
                <button onClick={handleConfirm} disabled={confirming || pages.length === 0} style={{
                  padding: '8px 20px', borderRadius: '8px', border: 'none',
                  background: pages.length > 0 ? 'linear-gradient(135deg, #F59E0B, #EF4444)' : '#E5E7EB',
                  color: pages.length > 0 ? '#fff' : '#9CA3AF',
                  fontSize: '14px', fontWeight: 600,
                  cursor: pages.length > 0 && !confirming ? 'pointer' : 'default',
                  boxShadow: pages.length > 0 ? '0 2px 8px rgba(245,158,11,0.3)' : 'none',
                }}>{confirming ? '确认中...' : '确认方案，选择风格 →'}</button>
              </div>
            </div>

            <IndexEditor coursewareId={id!} pages={pages} onPagesChange={setPages} isAdmin={isAdmin} indexOverview={courseware?.index_overview} />
          </div>
        )}

        {/* Step 2-5: 占位 */}
        {activeStep >= 2 && (
          <div style={{ textAlign: 'center', padding: '60px 0' }}>
            <div style={{ fontSize: '48px', marginBottom: '16px' }}>{STEPS[activeStep]?.emoji || '🔮'}</div>
            <div style={{ fontSize: '18px', fontWeight: 600, color: C.textPrimary, marginBottom: '8px' }}>
              {STEPS[activeStep]?.label || '进行中'}
            </div>
            <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '24px' }}>
              {STEPS[activeStep]?.desc || '此功能即将开放'}
            </div>
            <div style={{
              display: 'inline-block', padding: '10px 20px', borderRadius: '10px',
              background: '#F3F4F6', color: C.textMuted, fontSize: '14px',
            }}>🚧 此步骤将在后续版本开发</div>
          </div>
        )}
      </div>
    </div>
  )
}

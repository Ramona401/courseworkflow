/**
 * LessonPlanRefDrawer.tsx — 原教案对照抽屉（断裂B，纯前端零AI）
 *
 * 作用：在 Step4 批量生成 / Step5 工作台开发课件时，老师可随时呼出右侧抽屉
 *   对照原教案，确认课件是否落实了教案的教学设计。
 *
 * 行为：
 *   - 仅当课件关联教案（has_lesson_plan=true）时显示悬浮触发按钮；否则整体不渲染。
 *   - 首次打开才拉取教案正文（懒加载），之后缓存不重复请求。
 *   - 正文为纯文本/markdown，用轻量样式渲染（pre-wrap 保留换行+段落），零外部依赖。
 *
 * v2 改进：触发按钮改为紫色渐变醒目标签（原灰白贴边版太隐蔽，老师反馈看不见）。
 */
import { useState, useCallback } from 'react'
import { getCoursewareLessonPlanContent } from '@/api/coursewares'
import { C } from './workshopConstants'

interface Props {
  coursewareId: string
}

export default function LessonPlanRefDrawer({ coursewareId }: Props) {
  const [open, setOpen] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const [loading, setLoading] = useState(false)
  const [hasLessonPlan, setHasLessonPlan] = useState(true)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [hiddenEntry, setHiddenEntry] = useState(false)

  const ensureLoaded = useCallback(async () => {
    if (loaded || loading) return
    setLoading(true)
    try {
      const res = await getCoursewareLessonPlanContent(coursewareId)
      setHasLessonPlan(res.has_lesson_plan)
      setTitle(res.title || '')
      setContent(res.content || '')
      setLoaded(true)
      if (!res.has_lesson_plan) setHiddenEntry(true)
    } catch {
      /* 取数失败：保留入口下次重试 */
    } finally {
      setLoading(false)
    }
  }, [coursewareId, loaded, loading])

  const handleToggle = async () => {
    const next = !open
    setOpen(next)
    if (next) await ensureLoaded()
  }

  if (hiddenEntry) return null

  return (
    <>
      {/* 悬浮触发标签（紫色渐变醒目版，固定右侧中部，半外凸） */}
      <button
        onClick={handleToggle}
        title="对照原教案检查课件"
        style={{
          position: 'fixed', right: open ? 'min(560px, 86vw)' : 0, top: '38%',
          transform: 'translateY(-50%)', zIndex: 1001,
          padding: '16px 12px', borderRadius: '12px 0 0 12px', border: 'none',
          background: open ? 'linear-gradient(135deg, #6D28D9, #4C1D95)' : 'linear-gradient(135deg, #7C3AED, #6D28D9)',
          color: '#fff', fontSize: 15, fontWeight: 700, cursor: 'pointer',
          display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6,
          boxShadow: '-3px 3px 14px rgba(124,58,237,0.45)', transition: 'right 280ms',
        }}
      >
        <span style={{ fontSize: 20 }}>📄</span>
        <span style={{ writingMode: 'vertical-rl', letterSpacing: 3 }}>{open ? '收起教案' : '原教案对照'}</span>
      </button>

      {/* 抽屉面板 */}
      <div
        style={{
          position: 'fixed', top: 0, right: 0, height: '100vh',
          width: 'min(560px, 86vw)', zIndex: 1000,
          background: '#fff', borderLeft: '1px solid ' + C.border,
          boxShadow: '-4px 0 24px rgba(0,0,0,0.12)',
          transform: open ? 'translateX(0)' : 'translateX(100%)',
          transition: 'transform 280ms', display: 'flex', flexDirection: 'column',
        }}
      >
        <div style={{ padding: '16px 20px', borderBottom: '1px solid ' + C.border, display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
          <div style={{ minWidth: 0 }}>
            <div style={{ fontSize: 15, fontWeight: 700, color: C.textPrimary }}>📄 原教案对照</div>
            {title && <div style={{ fontSize: 12, color: C.textMuted, marginTop: 2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{title}</div>}
          </div>
          <button onClick={() => setOpen(false)} style={{ border: 'none', background: 'none', fontSize: 20, color: C.textMuted, cursor: 'pointer', lineHeight: 1 }}>✕</button>
        </div>

        <div style={{ flex: 1, overflowY: 'auto', padding: '16px 20px' }}>
          {loading && <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px 0', fontSize: 14 }}>加载教案中...</div>}
          {!loading && loaded && !hasLessonPlan && (
            <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px 0', fontSize: 14 }}>该课件无关联教案</div>
          )}
          {!loading && loaded && hasLessonPlan && content && (
            <div style={{ fontSize: 14, lineHeight: 1.8, color: C.textPrimary, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
              {content}
            </div>
          )}
          {!loading && loaded && hasLessonPlan && !content && (
            <div style={{ textAlign: 'center', color: C.textMuted, padding: '40px 0', fontSize: 14 }}>教案正文为空</div>
          )}
        </div>

        <div style={{ padding: '10px 20px', borderTop: '1px solid ' + C.border, fontSize: 12, color: C.textMuted, background: '#FAFAFA' }}>
          💡 对照教案检查课件是否落实教学设计
        </div>
      </div>
    </>
  )
}

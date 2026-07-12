/**
 * NavConfirmStep.tsx — 课件工坊 Step3 确认导航栏（批次5b-2从主页面拆出）
 *
 * 拆出范围：Step3整块JSX + 专属5个state/3个处理函数
 * （生成封面预览/保存导航栏auto提取/导航栏AI微调）。
 *
 * 与父级的接缝：
 *   - previewPages/setPreviewPages：封面预览页是父级真相源（loadCourseware恢复时填充、
 *     放映兜底列表也用），传下来；
 *   - sseRef：全工坊共享SSE句柄；goToStep/loadCourseware/refreshPagesOnly照传；
 *   - onSlideshow/onFullscreen：透传给 PagePreviewBlock。
 *
 * 已知微变更（可接受）：封面生成进行中若点步骤条离开本步，组件卸载会丢失
 * "生成中"运行态展示（previewPages由父级持有不丢，回来可重新生成或等轮询恢复）。
 */
import { useState } from 'react'
import type { Dispatch, SetStateAction, MutableRefObject } from 'react'
import { generateCWPreview, saveCWNavTemplate, refineNav, subscribeCWIndexSSE } from '@/api/coursewares'
import type { CoursewareDetail } from '@/api/coursewares'
import { C } from './workshopConstants'
import AppearancePanel from './AppearancePanel'
import PagePreviewBlock, { MsgBar } from './PagePreviewBlock'
import type { PageItem } from './PagePreviewBlock'

interface Props {
  coursewareId: string
  courseware: CoursewareDetail
  previewPages: PageItem[]
  setPreviewPages: Dispatch<SetStateAction<PageItem[]>>
  buildRunning: boolean
  sseRef: MutableRefObject<{ close: () => void } | null>
  goToStep: (n: number) => void
  loadCourseware: () => void
  refreshPagesOnly: () => void
  onSlideshow: (pn?: number) => void
  onFullscreen: (pn: number) => void
}

export default function NavConfirmStep({ coursewareId, courseware, previewPages, setPreviewPages, buildRunning, sseRef, goToStep, loadCourseware, refreshPagesOnly, onSlideshow, onFullscreen }: Props) {
  // ==================== Step3专属状态（5b-2自主页面整体迁入） ====================
  const [previewGenRunning, setPreviewGenRunning] = useState(false)
  const [previewGenMessage, setPreviewGenMessage] = useState('')
  const [navSaving, setNavSaving] = useState(false)
  // P0-2: 导航栏微调状态
  const [navRefineInput, setNavRefineInput] = useState('')
  const [navRefining, setNavRefining] = useState(false)

  // Step 3: 生成预览页（P0-1: 仅封面1页）
  const handleGenPreview = async () => {
    if (!coursewareId) return; setPreviewGenRunning(true); setPreviewGenMessage('正在启动...'); setPreviewPages([])
    try {
      await generateCWPreview(coursewareId); sseRef.current?.close()
      sseRef.current = subscribeCWIndexSSE(coursewareId, {
        onConnected: () => setPreviewGenMessage('已连接...'),
        onGenStart: d => setPreviewGenMessage(d.message),
        onGenProgress: d => setPreviewGenMessage(d.message),
        onGenPage: d => { setPreviewPages(p => [...p, { id: d.page_id, page_number: d.page_number, title: d.title, html_content: d.html_content }]) },
        onGenDone: d => { setPreviewGenRunning(false); if (d.fail_count > 0) { setPreviewGenMessage(`❌ ${d.message}`) } else { setPreviewGenMessage(`✅ ${d.message}`); loadCourseware() } },
        onError: d => { setPreviewGenMessage(`❌ ${d.message}`); setPreviewGenRunning(false) },
      })
    } catch { setPreviewGenMessage('❌ 启动失败'); setPreviewGenRunning(false) }
  }

  // Step 3: 确认导航栏（P0-1: 传"auto"让后端自动提取）
  const handleSaveNav = async () => {
    if (!coursewareId || previewPages.length === 0) return
    setNavSaving(true)
    try {
      await saveCWNavTemplate(coursewareId, 'auto')
      goToStep(4)
      loadCourseware()
    } catch (e) { alert('保存导航栏失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setNavSaving(false) }
  }

  // P0-2: 导航栏AI微调
  const handleRefineNav = async () => {
    if (!coursewareId || !navRefineInput.trim()) return
    setNavRefining(true)
    try {
      await refineNav(coursewareId, navRefineInput.trim())
      loadCourseware()
      setNavRefineInput('')
      setPreviewGenMessage('\u2705 导航栏微调完成')
    } catch (e) { setPreviewGenMessage('\u274c 微调失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setNavRefining(false) }
  }

  // ==================== JSX（与拆分前 Step3 逐行一致） ====================
  return <div>
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
      <div><h3 style={{ fontSize: 18, fontWeight: 600, color: C.textPrimary, margin: 0 }}>🧭 确认导航栏样式</h3><p style={{ fontSize: 13, color: C.textSecondary, margin: '4px 0 0' }}>AI先生成封面页，请确认顶部导航栏是否满意</p></div>
      {!previewGenRunning && <button onClick={() => goToStep(2)} style={{ padding: '8px 16px', borderRadius: 8, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 13, cursor: 'pointer' }}>← 返回选择风格</button>}
    </div>

    <MsgBar msg={previewGenMessage} />

    {/* P0-1: 只展示1页封面预览 */}
    {previewPages.length > 0 && (
      <PagePreviewBlock pages={previewPages} currentNum={previewPages[0]?.page_number || 1} onSelectPage={() => {}} showSlideshow={false} onSlideshow={onSlideshow} onFullscreen={onFullscreen} />
    )}

    {/* 操作按钮 */}
    <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
      {!previewGenRunning && previewPages.length === 0 && (
        <button onClick={handleGenPreview} style={{ padding: '14px 36px', borderRadius: 10, border: 'none', background: 'linear-gradient(135deg, #F59E0B, #EF4444)', color: '#fff', fontSize: 16, fontWeight: 600, cursor: 'pointer', boxShadow: '0 4px 16px rgba(245,158,11,0.3)' }}>🧭 生成封面预览页</button>
      )}
      {!previewGenRunning && previewPages.length > 0 && <>
        <button onClick={handleGenPreview} style={{ padding: '10px 24px', borderRadius: 8, border: `1px solid ${C.primary}`, background: C.primaryBg, color: C.primary, fontSize: 14, fontWeight: 600, cursor: 'pointer' }}>🔄 重新生成预览</button>
        <button onClick={handleSaveNav} disabled={navSaving} style={{ padding: '10px 24px', borderRadius: 8, border: 'none', background: 'linear-gradient(135deg, #059669, #10B981)', color: '#fff', fontSize: 14, fontWeight: 600, cursor: navSaving ? 'default' : 'pointer', boxShadow: '0 2px 8px rgba(5,150,105,0.3)' }}>
          {navSaving ? '保存中...' : '✅ 导航栏样式满意，开始批量生成 →'}
        </button>
      </>}
      {previewGenRunning && <div style={{ textAlign: 'center', padding: 20, color: C.textMuted, fontSize: 14, width: '100%' }}><div style={{ fontSize: 32, marginBottom: 8 }}>🧭</div>AI正在生成封面预览页，请稍候...</div>}
    </div>

    {/* 提示信息 */}
    {previewPages.length > 0 && !previewGenRunning && (
      <div style={{ marginTop: 16, padding: '12px 16px', borderRadius: 8, background: '#EFF6FF', color: '#2563EB', fontSize: 13 }}>
        💡 请仔细查看封面页的导航栏样式（顶部Logo、机构名、页码位置和颜色）。确认满意后点击"开始批量生成"，后续所有页面将自动使用完全相同的导航栏。
      </div>
    )}

    {/* 批次2（背景图库）：选背景秒换封面（零token零等待），后续批量生成的内页自动带内页底纹 */}
    {previewPages.length > 0 && !previewGenRunning && (
      <AppearancePanel coursewareId={coursewareId} onSwapped={refreshPagesOnly} disabled={buildRunning}
        cwTitle={courseware.title} cwSubject={courseware.subject} cwGrade={courseware.grade} />
    )}

    {/* P0-2: 导航栏AI微调输入区 */}
    {previewPages.length > 0 && !previewGenRunning && (
      <div style={{ marginTop: 16, padding: '16px', borderRadius: 10, border: `1px solid ${C.border}`, background: '#FAFAFA' }}>
        <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🎨 导航栏不满意？输入修改意见让AI微调</div>
        <div style={{ display: 'flex', gap: 10 }}>
          <input value={navRefineInput} onChange={e => setNavRefineInput(e.target.value)}
            placeholder="例如：Logo再大一点、页码改成右对齐、背景色改为深蓝..."
            onKeyDown={e => { if (e.key === 'Enter' && !navRefining) handleRefineNav() }}
            style={{ flex: 1, padding: '10px 14px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 14, outline: 'none' }}
            disabled={navRefining} />
          <button onClick={handleRefineNav} disabled={navRefining || !navRefineInput.trim()}
            style={{ padding: '10px 20px', borderRadius: 8, border: 'none', background: navRefineInput.trim() && !navRefining ? '#7C3AED' : '#E5E7EB', color: navRefineInput.trim() && !navRefining ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: navRefineInput.trim() && !navRefining ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
            {navRefining ? '⏳ 微调中...' : '🎨 AI微调'}
          </button>
        </div>
      </div>
    )}
  </div>
}

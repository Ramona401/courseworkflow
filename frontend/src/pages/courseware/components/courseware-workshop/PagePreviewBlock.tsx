/**
 * PagePreviewBlock.tsx — 课件页面胶片条、大预览与源码工作区
 *
 * 页面管理能力：拖拽排序、删除、新增、页码校准、源码保存和代码收藏。
 * 源码工作区从本文件拆为 SourceCodeEditor，并使用 React.lazy 按需加载：
 *   - 搜索栏默认展开，打开源码后可直接输入关键词；
 *   - HTML正文、标签、属性、字符串、CSS和函数分别着色；
 *   - 上一处/下一处、替换、大小写、全词匹配与 Minimap 定位；
 *   - 保存仍复用 savePageHtml，后端覆盖前自动创建 manual 历史版本。
 */
import { lazy, Suspense, useCallback, useEffect, useRef, useState } from 'react'
import type { DragEvent, MouseEvent } from 'react'
import { C, CW_HEIGHT, CW_WIDTH } from './workshopConstants'
import { injectPreviewMode } from './previewInject'
import { createCodeSnippet, deleteCWPage, reorderCWPages, savePageHtml } from '@/api/coursewares'
import AddPageModal from './AddPageModal'
import CoursewareBatchIntegritySection from './CoursewareBatchIntegritySection'
import PageNumberCalibrationButton from './PageNumberCalibrationButton'
import PlatformCoursewareAssistantOverlay from './PlatformCoursewareAssistantOverlay'
import {
  rememberCoursewarePreviewPage,
  resolveRememberedCoursewarePreviewPageNumber,
} from './coursewarePreviewPosition'

/** 源码编辑器只在老师打开源码视图时下载，避免进入工坊即加载。 */
const LazySourceCodeEditor = lazy(() => import('./SourceCodeEditor'))

export interface PageItem {
  page_number: number
  title: string
  html_content: string
  id?: string
}

export function MsgBar({ msg }: { msg: string }) {
  if (!msg) return null
  const tone = msg.startsWith('❌')
    ? { bg: '#FEE2E2', color: '#DC2626' }
    : msg.startsWith('✅')
      ? { bg: '#D1FAE5', color: '#059669' }
      : msg.startsWith('⚠️')
        ? { bg: '#FEF3C7', color: '#D97706' }
        : { bg: '#EFF6FF', color: '#2563EB' }
  return <div style={{ padding: '12px 16px', borderRadius: 8, marginBottom: 16, background: tone.bg, color: tone.color, fontSize: 14 }}>{msg}</div>
}

/** 轻量检查最常见的外部代码粘贴残缺；提示但不替代后端完整校验。 */
function divBalanceCheck(source: string): string {
  const open = (source.match(/<div\b/gi) || []).length
  const close = (source.match(/<\/div>/gi) || []).length
  return open === close ? '' : `<div> 开标签 ${open} 个、</div> 闭标签 ${close} 个，数量不一致，页面可能残缺或变形`
}

interface Props {
  pages: PageItem[]
  currentNum: number
  onSelectPage: (pageNumber: number) => void
  showSlideshow: boolean
  onSlideshow: (pageNumber?: number) => void
  onFullscreen: (pageNumber: number) => void
  editable?: boolean
  coursewareId?: string
  onPagesChanged?: () => void
}

export default function PagePreviewBlock({
  pages,
  currentNum,
  onSelectPage,
  showSlideshow,
  onSlideshow,
  onFullscreen,
  editable = false,
  coursewareId,
  onPagesChanged,
}: Props) {
  const [codeViewPageNum, setCodeViewPageNum] = useState(0)
  const [codeEditing, setCodeEditing] = useState(false)
  const [codeDraft, setCodeDraft] = useState('')
  const [codeSaving, setCodeSaving] = useState(false)

  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null)
  const dragCounterRef = useRef(0)

  const [operating, setOperating] = useState(false)
  const [opMsg, setOpMsg] = useState('')
  const [showAddModal, setShowAddModal] = useState(false)

  const [showStarModal, setShowStarModal] = useState(false)
  const [starTitle, setStarTitle] = useState('')
  const [starNote, setStarNote] = useState('')
  const [starSaving, setStarSaving] = useState(false)

  /**
   * 普通预览默认保持安全点击语义：点击画面进入放映。
   * 老师显式开启互动试玩后，鼠标事件才交给iframe内课件。
   */
  const [interactivePreview, setInteractivePreview] = useState(false)
  const restoredPositionCoursewareRef = useRef('')

  const containerWidth = 912
  const previewScale = containerWidth / CW_WIDTH
  const activePage = currentNum > 0 ? currentNum : (pages[0]?.page_number || 0)
  const activePageItem = pages.find(page => page.page_number === activePage) || pages[0]
  const html = activePageItem?.html_content || ''
  const previewHtml = injectPreviewMode(html)
  const hasUnsavedCode = codeEditing && codeDraft !== html

  /**
   * 刷新后恢复上一次真正停留的稳定页面。
   *
   * 首次拿到当前课件页面列表时先尝试用sessionStorage中的page_id恢复；
   * 只有恢复动作完成后才写回当前位置，避免加载默认P1时覆盖掉原来的P7/P12。
   * 后续普通切页、全屏退出回写、放映退出回写都会通过activePage变化自动更新记录。
   */
  useEffect(() => {
    if (!coursewareId || pages.length === 0) return

    if (restoredPositionCoursewareRef.current !== coursewareId) {
      const restoredPageNumber =
        resolveRememberedCoursewarePreviewPageNumber(
          coursewareId,
          pages,
        )

      restoredPositionCoursewareRef.current = coursewareId

      if (
        restoredPageNumber !== null &&
        restoredPageNumber !== activePage
      ) {
        onSelectPage(restoredPageNumber)
        return
      }
    }

    if (activePageItem?.id) {
      rememberCoursewarePreviewPage(
        activePageItem.id,
        coursewareId,
      )
    }
  }, [
    activePage,
    activePageItem?.id,
    coursewareId,
    onSelectPage,
    pages,
  ])

  /**
   * 浏览器刷新、关闭标签或离开站点前保护未保存源码。
   *
   * 页面内部切页已使用 confirmDiscardCodeEdit 做中文确认；这里负责浏览器级离开，
   * 使用标准 beforeunload 提示，避免老师长时间修改后误按刷新导致草稿直接丢失。
   */
  useEffect(() => {
    if (!hasUnsavedCode) return

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ''
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [hasUnsavedCode])

  /** 外部状态切页时兜底退出源码视图，防止上一页草稿串到新页。 */
  useEffect(() => {
    setCodeViewPageNum(0)
    setCodeEditing(false)
    setCodeDraft('')
    setInteractivePreview(false)
  }, [activePage])

  const showMsg = useCallback((msg: string, milliseconds = 3000) => {
    setOpMsg(msg)
    window.setTimeout(() => setOpMsg(''), milliseconds)
  }, [])

  const clearCodeEdit = useCallback(() => {
    setCodeEditing(false)
    setCodeDraft('')
  }, [])

  /** 所有会离开当前源码草稿的动作共用同一确认口径。 */
  const confirmDiscardCodeEdit = useCallback((message: string): boolean => {
    if (hasUnsavedCode && !window.confirm(message)) return false
    clearCodeEdit()
    return true
  }, [clearCodeEdit, hasUnsavedCode])

  const handleSelectPage = useCallback((pageNumber: number) => {
    if (pageNumber === activePage) return
    if (!confirmDiscardCodeEdit('切换页面会放弃当前未保存的源码修改，确定继续？')) return
    onSelectPage(pageNumber)
  }, [activePage, confirmDiscardCodeEdit, onSelectPage])

  const handleOpenStar = useCallback(() => {
    if (!coursewareId || activePage <= 0 || codeEditing) return
    const page = pages.find(item => item.page_number === activePage)
    setStarTitle(page?.title || `第${activePage}页`)
    setStarNote('')
    setShowStarModal(true)
  }, [activePage, codeEditing, coursewareId, pages])

  const handleSaveStar = useCallback(async () => {
    if (!coursewareId || activePage <= 0 || starSaving) return
    if (!starTitle.trim()) {
      showMsg('❌ 请给这条收藏起个名称')
      return
    }
    setStarSaving(true)
    try {
      const result = await createCodeSnippet(coursewareId, activePage, starTitle.trim(), starNote.trim() || undefined)
      setShowStarModal(false)
      showMsg(`✅ ${result.message}，微调面板「📎 注入参考代码」可用`, 4000)
    } catch (error) {
      showMsg(`❌ 收藏失败: ${error instanceof Error ? error.message : '未知错误'}`, 5000)
    } finally {
      setStarSaving(false)
    }
  }, [activePage, coursewareId, showMsg, starNote, starSaving, starTitle])

  const handleStartCodeEdit = useCallback(() => {
    if (!editable || !coursewareId || operating || codeSaving) return
    setCodeDraft(html)
    setCodeEditing(true)
  }, [codeSaving, coursewareId, editable, html, operating])

  const handleCancelCodeEdit = useCallback(() => {
    if (codeSaving) return
    confirmDiscardCodeEdit('放弃未保存的源码修改？')
  }, [codeSaving, confirmDiscardCodeEdit])

  const handleSaveCodeEdit = useCallback(async () => {
    if (!coursewareId || activePage <= 0 || codeSaving) return
    const content = codeDraft
    if (!content.trim()) {
      showMsg('❌ 源码内容为空，未保存')
      return
    }
    if (content === html) {
      showMsg('⚠️ 内容未变化，无需保存')
      clearCodeEdit()
      return
    }

    const warning = divBalanceCheck(content)
    if (warning && !window.confirm(
      `⚠️ 结构自检提示：${warning}。\n\n仍要保存吗？\n（保存前系统会自动把当前版本存入历史，改坏了可在微调面板「📜 历史版本」一键回退）`,
    )) return

    setCodeSaving(true)
    try {
      await savePageHtml(coursewareId, activePage, content)
      showMsg(`✅ 第 ${activePage} 页源码已保存（旧版已存入历史版本，可回退）`, 4000)
      clearCodeEdit()
      onPagesChanged?.()
    } catch (error) {
      showMsg(`❌ 保存失败: ${error instanceof Error ? error.message : '未知错误'}`, 5000)
    } finally {
      setCodeSaving(false)
    }
  }, [activePage, clearCodeEdit, codeDraft, codeSaving, coursewareId, html, onPagesChanged, showMsg])

  const handleToggleCodeView = useCallback(() => {
    setInteractivePreview(false)

    if (codeViewPageNum === activePage) {
      if (!confirmDiscardCodeEdit('切回预览会放弃当前未保存的源码修改，确定继续？')) return
      setCodeViewPageNum(0)
    } else {
      setCodeViewPageNum(activePage)
    }
  }, [activePage, codeViewPageNum, confirmDiscardCodeEdit])

  const handleDragStart = useCallback((event: DragEvent, index: number) => {
    if (!editable || operating || codeEditing) return
    setDragIndex(index)
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', String(index))
  }, [codeEditing, editable, operating])

  const handleDragEnter = useCallback((event: DragEvent, index: number) => {
    event.preventDefault()
    if (dragIndex === null || dragIndex === index) return
    setDragOverIndex(index)
  }, [dragIndex])

  const handleDragOver = useCallback((event: DragEvent) => {
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
  }, [])

  const handleDrop = useCallback(async (event: DragEvent, dropIndex: number) => {
    event.preventDefault()
    if (!editable || !coursewareId || dragIndex === null || dragIndex === dropIndex) {
      setDragIndex(null)
      setDragOverIndex(null)
      return
    }

    const nextPages = [...pages]
    const [moved] = nextPages.splice(dragIndex, 1)
    nextPages.splice(dropIndex, 0, moved)
    const pageIds = nextPages.map(page => page.id).filter(Boolean) as string[]
    if (pageIds.length !== nextPages.length) {
      showMsg('⚠️ 部分页面数据不完整，请刷新后重试')
      setDragIndex(null)
      setDragOverIndex(null)
      return
    }

    setOperating(true)
    setDragIndex(null)
    setDragOverIndex(null)
    try {
      await reorderCWPages(coursewareId, pageIds)
      showMsg('✅ 页面顺序已更新')
      onPagesChanged?.()
    } catch {
      showMsg('❌ 排序失败，请重试')
    } finally {
      setOperating(false)
    }
  }, [coursewareId, dragIndex, editable, onPagesChanged, pages, showMsg])

  const handleDragEnd = useCallback(() => {
    setDragIndex(null)
    setDragOverIndex(null)
    dragCounterRef.current = 0
  }, [])

  const handleDeletePage = useCallback(async (event: MouseEvent, pageNumber: number) => {
    event.stopPropagation()
    if (!editable || !coursewareId || operating || codeSaving) return
    if (pages.length <= 1) {
      showMsg('⚠️ 至少保留一页，不能全部删除')
      return
    }
    const page = pages.find(item => item.page_number === pageNumber)
    const title = page?.title || `第${pageNumber}页`
    const unsavedWarning = pageNumber === activePage && hasUnsavedCode
      ? '\n\n⚠️ 当前页还有未保存的源码修改，删除后草稿也会丢失。'
      : ''
    if (!window.confirm(`确定删除「${title}」(P${pageNumber})？\n\n删除后该页的 HTML 内容和配图资产将一并清除，此操作不可撤销。${unsavedWarning}`)) return
    if (pageNumber === activePage) clearCodeEdit()

    setOperating(true)
    try {
      await deleteCWPage(coursewareId, pageNumber)
      showMsg('✅ 已删除')
      if (activePage === pageNumber) {
        const remaining = pages.filter(item => item.page_number !== pageNumber)
        if (remaining.length > 0) onSelectPage(remaining[0].page_number)
      }
      onPagesChanged?.()
    } catch {
      showMsg('❌ 删除失败，请重试')
    } finally {
      setOperating(false)
    }
  }, [activePage, clearCodeEdit, codeSaving, coursewareId, editable, hasUnsavedCode, onPagesChanged, onSelectPage, operating, pages, showMsg])

  const handleAddDone = useCallback((newPageNumber: number) => {
    setShowAddModal(false)
    onSelectPage(newPageNumber)
    onPagesChanged?.()
  }, [onPagesChanged, onSelectPage])

  const sourceValue = codeEditing ? codeDraft : html
  const sourceMode = codeViewPageNum === activePage

  return <>
    {opMsg && <MsgBar msg={opMsg} />}

    {editable && coursewareId && (
      <CoursewareBatchIntegritySection
        coursewareId={coursewareId}
        onPagesChanged={onPagesChanged}
      />
    )}

    {pages.length > 0 && (
      <div style={{ marginBottom: 20 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8, marginBottom: 10, flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontSize: 14, fontWeight: 600, color: C.textPrimary }}>📄 当前页面列表 {pages.length} 页</span>
            {editable && <span style={{ fontSize: 12, color: C.textMuted }}>拖拽调序 · 点 × 删除</span>}
          </div>
          <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
            {editable && coursewareId && (
              <PageNumberCalibrationButton
                coursewareId={coursewareId}
                pages={pages}
                activePage={activePage}
                disabled={operating || codeSaving || starSaving || codeEditing}
                onBusyChange={setOperating}
                onSelectPage={onSelectPage}
                onPagesChanged={onPagesChanged}
                onMessage={showMsg}
              />
            )}
            {editable && (
              <button onClick={() => setShowAddModal(true)} disabled={operating || codeEditing} style={{ padding: '6px 14px', borderRadius: 8, border: `1px dashed ${C.primary}`, background: C.primaryBg, color: C.primary, fontSize: 13, fontWeight: 600, cursor: operating || codeEditing ? 'default' : 'pointer', opacity: operating || codeEditing ? 0.5 : 1 }}>＋ 添加页面</button>
            )}
            {showSlideshow && <button onClick={() => onSlideshow()} style={{ padding: '6px 14px', borderRadius: 8, border: `1px solid ${C.primary}`, background: C.primaryBg, color: C.primary, fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>🖥️ 全屏放映</button>}
          </div>
        </div>

        <div style={{ overflow: 'visible', position: 'relative' }}>
          <div style={{ display: 'flex', gap: 6, flexWrap: 'nowrap', overflowX: 'auto', paddingBottom: 6, paddingTop: editable ? 10 : 0, alignItems: 'center' }}>
            {pages.map((page, index) => {
              const isActive = activePage === page.page_number
              const isDragging = dragIndex === index
              const isDragOver = dragOverIndex === index
              return (
                <div
                  key={page.id || page.page_number}
                  draggable={editable && !operating && !codeEditing}
                  onDragStart={editable ? event => handleDragStart(event, index) : undefined}
                  onDragEnter={editable ? event => handleDragEnter(event, index) : undefined}
                  onDragOver={editable ? handleDragOver : undefined}
                  onDrop={editable ? event => handleDrop(event, index) : undefined}
                  onDragEnd={editable ? handleDragEnd : undefined}
                  style={{ position: 'relative', flexShrink: 0, opacity: isDragging ? 0.4 : 1, transition: 'all 200ms' }}
                >
                  {editable && isDragOver && dragIndex !== null && dragIndex !== index && <div style={{ position: 'absolute', left: -4, top: 2, bottom: 2, width: 3, background: '#3B82F6', borderRadius: 2, zIndex: 10 }} />}
                  <button
                    onClick={() => handleSelectPage(page.page_number)}
                    title={`P${page.page_number} ${page.title}${editable ? '\n拖拽可调整顺序' : ''}`}
                    style={{ padding: '6px 10px', borderRadius: 8, cursor: editable && !operating && !codeEditing ? 'grab' : 'pointer', whiteSpace: 'nowrap', border: `2px solid ${isActive ? C.primary : C.border}`, background: isActive ? C.primaryBg : C.white, color: isActive ? C.primary : C.textPrimary, fontSize: 12, fontWeight: isActive ? 600 : 400 }}
                  >
                    <span style={{ fontWeight: 700 }}>P{page.page_number}</span>
                    <span style={{ marginLeft: 5, color: C.textSecondary, fontSize: 11 }}>{page.title.length > 6 ? `${page.title.slice(0, 6)}…` : page.title}</span>
                  </button>
                  {editable && !operating && !codeEditing && (
                    <button onClick={event => handleDeletePage(event, page.page_number)} title={`删除 P${page.page_number}`} style={{ position: 'absolute', top: -6, right: -6, width: 18, height: 18, borderRadius: '50%', background: '#EF4444', color: '#fff', border: '2px solid #fff', fontSize: 11, fontWeight: 700, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', lineHeight: 1, padding: 0, boxShadow: '0 1px 4px rgba(0,0,0,0.2)', zIndex: 5 }}>×</button>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      </div>
    )}

    {pages.length === 0 && editable && (
      <div style={{ marginBottom: 20, textAlign: 'center', padding: '30px 0' }}>
        <div style={{ fontSize: 32, marginBottom: 8 }}>📄</div>
        <div style={{ fontSize: 14, color: C.textSecondary, marginBottom: 12 }}>暂无已生成的页面</div>
        <button onClick={() => setShowAddModal(true)} disabled={operating} style={{ padding: '8px 20px', borderRadius: 8, border: `2px dashed ${C.primary}`, background: C.primaryBg, color: C.primary, fontSize: 14, fontWeight: 600, cursor: operating ? 'default' : 'pointer', opacity: operating ? 0.5 : 1 }}>＋ 添加页面</button>
      </div>
    )}

    {html && (
      <div style={{ marginBottom: 20 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8, marginBottom: 10, flexWrap: 'wrap' }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: C.textPrimary }}>
            {sourceMode ? '💻' : '📺'} 第 {activePage} 页{sourceMode ? (codeEditing ? '源代码（编辑中）' : '源代码') : '预览'}
            {sourceMode && <span style={{ marginLeft: 8, color: C.textMuted, fontSize: 11, fontWeight: 400 }}>搜索栏默认展开 · 正文、标签、属性、样式和函数分色</span>}
          </div>
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            {sourceMode && editable && coursewareId && !codeEditing && <button onClick={handleStartCodeEdit} disabled={operating || codeSaving} style={{ padding: '4px 10px', borderRadius: 6, border: '1px solid #059669', background: 'rgba(5,150,105,0.06)', color: '#059669', fontSize: 12, fontWeight: 600, cursor: operating || codeSaving ? 'default' : 'pointer' }}>✏️ 编辑源码</button>}
            {sourceMode && codeEditing && <>
              <button onClick={handleSaveCodeEdit} disabled={codeSaving} style={{ padding: '4px 12px', borderRadius: 6, border: 'none', background: codeSaving ? '#E5E7EB' : '#059669', color: codeSaving ? '#9CA3AF' : '#fff', fontSize: 12, fontWeight: 600, cursor: codeSaving ? 'default' : 'pointer' }}>{codeSaving ? '⏳ 保存中...' : '💾 保存'}</button>
              <button onClick={handleCancelCodeEdit} disabled={codeSaving} style={{ padding: '4px 10px', borderRadius: 6, border: '1px solid #EF4444', background: 'transparent', color: '#EF4444', fontSize: 12, cursor: codeSaving ? 'default' : 'pointer' }}>✖ 取消</button>
            </>}
            {!sourceMode && (
              <button
                type="button"
                onClick={() => {
                  setInteractivePreview(current => !current)
                }}
                title={
                  interactivePreview
                    ? '退出互动试玩，恢复点击画面进入放映'
                    : '让鼠标事件进入课件，可试玩转盘、按钮、拖拽和测验'
                }
                style={{
                  padding: '4px 10px',
                  borderRadius: 6,
                  border: `1px solid ${interactivePreview ? '#059669' : C.border}`,
                  background: interactivePreview
                    ? 'rgba(5,150,105,0.08)'
                    : 'transparent',
                  color: interactivePreview
                    ? '#059669'
                    : C.textSecondary,
                  fontSize: 12,
                  fontWeight: interactivePreview
                    ? 700
                    : 400,
                  cursor: 'pointer',
                }}
              >
                {interactivePreview
                  ? '🛑 退出互动试玩'
                  : '🎮 互动试玩'}
              </button>
            )}
            <button onClick={handleToggleCodeView} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${sourceMode ? '#7C3AED' : C.border}`, background: sourceMode ? 'rgba(124,58,237,0.06)' : 'transparent', color: sourceMode ? '#7C3AED' : C.textSecondary, fontSize: 12, cursor: 'pointer' }}>{sourceMode ? '📺 预览' : '💻 源代码'}</button>
            {editable && coursewareId && <button onClick={handleOpenStar} disabled={starSaving || codeSaving || codeEditing} title={codeEditing ? '请先保存或取消源码编辑，再收藏服务端正式版本' : '把这一页的代码收藏进我的代码库'} style={{ padding: '4px 10px', borderRadius: 6, border: '1px solid #F59E0B', background: 'rgba(245,158,11,0.06)', color: '#D97706', fontSize: 12, fontWeight: 600, cursor: starSaving || codeSaving || codeEditing ? 'default' : 'pointer', opacity: codeEditing ? 0.5 : 1 }}>⭐ 收藏代码</button>}
            <button onClick={() => navigator.clipboard.writeText(sourceValue).then(() => alert('源代码已复制到剪贴板')).catch(() => {})} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer' }}>📋 复制代码</button>
            <button onClick={() => onFullscreen(activePage)} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer' }}>🔍 全屏预览</button>
            <button onClick={() => onSlideshow(activePage)} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer' }}>🖥️ 放映</button>
          </div>
        </div>

        {sourceMode ? (
          <div>
            <Suspense fallback={<div style={{ height: 520, borderRadius: 14, background: '#1E1E1E', color: '#9CA3AF', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 13 }}>💻 源码编辑器加载中...</div>}>
              <LazySourceCodeEditor value={sourceValue} onChange={codeEditing ? setCodeDraft : undefined} readOnly={!codeEditing} disabled={codeSaving} height={520} />
            </Suspense>
            <div style={{ marginTop: 7, fontSize: 11, color: '#9CA3AF', lineHeight: 1.7 }}>
              💡 搜索栏默认打开，直接输入正文或代码关键词即可；正文使用醒目的浅黄色，标签、属性、字符串、样式和函数分别着色。右侧小窗可点击或拖动快速定位。整页替换时请保持最外层
              {' <div style="width:1920px;height:1080px..." class="cw-page"> '}
              画布结构。保存前系统自动创建历史版本，改坏后可在「📜 历史版本」回退。
            </div>
          </div>
        ) : (
          <div>
            <div
              onClick={
                interactivePreview
                  ? undefined
                  : () => onSlideshow(activePage)
              }
              style={{
                width: '100%',
                height: Math.ceil(CW_HEIGHT * previewScale),
                position: 'relative',
                overflow: 'hidden',
                borderRadius: 14,
                border: `1px solid ${
                  interactivePreview
                    ? '#10B981'
                    : C.border
                }`,
                background: '#F8FAFC',
                cursor: interactivePreview
                  ? 'default'
                  : 'pointer',
                boxShadow: interactivePreview
                  ? '0 0 0 3px rgba(16,185,129,0.10)'
                  : 'none',
              }}
            >
              <iframe
                key={`p${activePage}-${previewHtml.length}-${interactivePreview ? 'interactive' : 'safe'}`}
                srcDoc={previewHtml}
                scrolling="no"
                style={{
                  width: CW_WIDTH,
                  height: CW_HEIGHT,
                  border: 'none',
                  pointerEvents: interactivePreview
                    ? 'auto'
                    : 'none',
                  transform: `scale(${previewScale})`,
                  transformOrigin: 'top left',
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  overflow: 'hidden',
                }}
                sandbox="allow-scripts"
                title={`预览-P${activePage}`}
              />

              {activePageItem?.id && (
                <PlatformCoursewareAssistantOverlay
                  key={`platform-assistant-${activePageItem.id}`}
                  coursewareId={coursewareId}
                  pageId={activePageItem.id}
                  pageTitle={activePageItem.title}
                  variant="embedded"
                />
              )}
            </div>

            {interactivePreview && (
              <div
                style={{
                  marginTop: 7,
                  color: '#047857',
                  fontSize: 11.5,
                  lineHeight: 1.6,
                }}
              >
                🎮 互动试玩中：点击会直接操作课件内的转盘、按钮、拖拽或测验；退出试玩后，点击画面恢复进入放映。
              </div>
            )}
          </div>
        )}
      </div>
    )}

    {showStarModal && coursewareId && (
      <div onClick={starSaving ? undefined : () => setShowStarModal(false)} style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', zIndex: 99990, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div onClick={event => event.stopPropagation()} style={{ background: '#fff', borderRadius: 16, padding: '26px 30px', width: '100%', maxWidth: 460, boxShadow: '0 20px 60px rgba(0,0,0,0.15)' }}>
          <h3 style={{ margin: '0 0 6px', fontSize: 17, fontWeight: 600, color: '#1F2937' }}>⭐ 收藏第 {activePage} 页代码</h3>
          <div style={{ fontSize: 12, color: '#9CA3AF', marginBottom: 16, lineHeight: 1.7 }}>收藏的是这一页此刻的完整服务端代码快照，之后修改本页不影响收藏。</div>
          <label style={{ display: 'block', fontSize: 13, color: '#6B7280', marginBottom: 4, fontWeight: 500 }}>收藏名称 *</label>
          <input value={starTitle} onChange={event => setStarTitle(event.target.value)} autoFocus placeholder="例如：左右对比卡片布局" disabled={starSaving} style={{ width: '100%', padding: '8px 12px', borderRadius: 8, border: '1px solid #E5E7EB', fontSize: 14, outline: 'none', boxSizing: 'border-box', marginBottom: 12 }} />
          <label style={{ display: 'block', fontSize: 13, color: '#6B7280', marginBottom: 4, fontWeight: 500 }}>备注<span style={{ color: '#9CA3AF', fontSize: 12, marginLeft: 4 }}>（可选）</span></label>
          <textarea value={starNote} onChange={event => setStarNote(event.target.value)} rows={2} disabled={starSaving} placeholder="例如：适合概念对比页" style={{ width: '100%', padding: '8px 12px', borderRadius: 8, border: '1px solid #E5E7EB', fontSize: 13, outline: 'none', boxSizing: 'border-box', resize: 'vertical', fontFamily: 'inherit', marginBottom: 18 }} />
          <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
            <button onClick={() => setShowStarModal(false)} disabled={starSaving} style={{ padding: '8px 20px', borderRadius: 8, border: '1px solid #E5E7EB', background: 'transparent', color: '#6B7280', fontSize: 14, cursor: starSaving ? 'default' : 'pointer' }}>取消</button>
            <button onClick={handleSaveStar} disabled={starSaving} style={{ padding: '8px 24px', borderRadius: 8, border: 'none', background: starSaving ? '#E5E7EB' : '#F59E0B', color: starSaving ? '#9CA3AF' : '#fff', fontSize: 14, fontWeight: 600, cursor: starSaving ? 'default' : 'pointer' }}>{starSaving ? '⏳ 收藏中...' : '⭐ 收藏'}</button>
          </div>
        </div>
      </div>
    )}

    {showAddModal && coursewareId && <AddPageModal coursewareId={coursewareId} currentPageCount={pages.length} onDone={handleAddDone} onClose={() => setShowAddModal(false)} />}
  </>
}

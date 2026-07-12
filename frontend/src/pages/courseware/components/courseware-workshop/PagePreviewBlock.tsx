/**
 * PagePreviewBlock.tsx — 课件页面胶片条+大预览块
 *
 * v2.2 批次A新增【源码可编辑】：
 *   源码视图新增「✏️ 编辑源码」模式（仅 editable 且提供 coursewareId 时显示按钮）：
 *     · 只读行号表格 ↔ 可编辑 textarea 一键切换；
 *     · 保存复用既有 savePageHtml（POST .../pages/{num}/save-html，不调AI；
 *       后端覆盖前自动把旧版存为 manual 版本快照，可在微调面板「📜 历史版本」回退/对比）；
 *     · 保存前做轻量结构自检（<div> 开/闭标签数量比对），疑似残缺时二次确认再放行；
 *     · 编辑中切页 / 切回预览视图若有未保存改动，先弹确认防误丢；
 *     · Tab 键插入两个空格（不跳焦点），便于调整缩进；
 *     · 老师既可直接改本页源码，也可整页粘贴外部好代码——注意保持最外层
 *       <div style="width:1920px;height:1080px..." class="cw-page"> 画布契约，否则显示会变形
 *       （编辑区下方有常驻提示文案）。
 *   保存成功后经 onPagesChanged 通知父级重拉页面列表刷新预览。
 *   鉴权口径：save-html 后端为"仅作者本人"（与就地编辑一致），非作者保存会收到明确报错。
 *
 * v2.1 修复与优化：
 *   1. 修复 × 删除按钮被胶片条 overflow:auto 截断问题
 *      → 外层 overflow:visible + 内层 padding-top 留空间 + 内滚动层 overflow-x:auto
 *   2. ＋ 新增按钮改为弹出 AddPageModal 快速添加弹窗
 *      → 填方案 → 创建 → 自动生成 HTML，一站式不离开 Step5
 *   3. 拖拽排序 + 删除保持不变
 */
import { useState, useRef, useCallback, useEffect } from 'react'
import { C, CW_WIDTH, CW_HEIGHT } from './workshopConstants'
import { injectPreviewMode } from './previewInject'
import { reorderCWPages, deleteCWPage, savePageHtml, createCodeSnippet } from '@/api/coursewares'
import AddPageModal from './AddPageModal'

/** 页面条目（主页面 previewPages/generatedPages 的统一元素类型） */
export interface PageItem {
  page_number: number
  title: string
  html_content: string
  id?: string // 页面数据库ID，排序 API 需要
}

/** 共享消息条：按消息前缀(❌/✅/⚠️/其它)自动配色，空消息不渲染 */
export function MsgBar({ msg }: { msg: string }) {
  if (!msg) return null
  return (
    <div style={{
      padding: '12px 16px', borderRadius: 8, marginBottom: 16,
      background: msg.startsWith('❌') ? '#FEE2E2' : msg.startsWith('✅') ? '#D1FAE5' : msg.startsWith('⚠️') ? '#FEF3C7' : '#EFF6FF',
      color: msg.startsWith('❌') ? '#DC2626' : msg.startsWith('✅') ? '#059669' : msg.startsWith('⚠️') ? '#D97706' : '#2563EB',
      fontSize: 14,
    }}>{msg}</div>
  )
}

/**
 * 批次A·轻量结构自检：比对 <div 开标签与 </div> 闭标签数量。
 * 不做完整 HTML 解析（外部粘贴代码风格多样，严格解析误伤率高），
 * 只抓"最常见的粘贴残缺/漏尾"问题。数量不一致返回人话提示文案，一致返回空串。
 */
function divBalanceCheck(s: string): string {
  const open = (s.match(/<div\b/gi) || []).length
  const close = (s.match(/<\/div>/gi) || []).length
  if (open !== close) {
    return `<div> 开标签 ${open} 个、</div> 闭标签 ${close} 个，数量不一致，页面可能残缺或变形`
  }
  return ''
}

interface Props {
  /** 已生成页面列表 */
  pages: PageItem[]
  /** 当前选中页号（0=取列表第一页） */
  currentNum: number
  /** 点胶片条某页签时回调 */
  onSelectPage: (n: number) => void
  /** 是否显示顶部「🖥️ 全屏放映」按钮 */
  showSlideshow: boolean
  /** 放映回调 */
  onSlideshow: (pn?: number) => void
  /** 全屏预览回调 */
  onFullscreen: (pn: number) => void

  // ==================== 页面管理能力（可选） ====================
  /** 是否开启编辑模式（拖拽排序 + 删除 + 新增 + 源码编辑），默认 false */
  editable?: boolean
  /** 课件ID（editable=true 时必须提供） */
  coursewareId?: string
  /** 页面变更后通知父级刷新 */
  onPagesChanged?: () => void
}

export default function PagePreviewBlock({
  pages, currentNum, onSelectPage, showSlideshow, onSlideshow, onFullscreen,
  editable = false, coursewareId, onPagesChanged,
}: Props) {
  // ==================== 源码视图状态 ====================
  const [codeViewPageNum, setCodeViewPageNum] = useState(0)

  // ==================== 批次A：源码编辑状态 ====================
  const [codeEditing, setCodeEditing] = useState(false)   // 是否处于源码编辑模式
  const [codeDraft, setCodeDraft] = useState('')          // 编辑草稿（进入编辑时以当前页HTML初始化）
  const [codeSaving, setCodeSaving] = useState(false)     // 保存请求进行中

  // ==================== 拖拽排序状态 ====================
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null)
  const dragCounterRef = useRef(0)

  // ==================== 操作状态 ====================
  const [operating, setOperating] = useState(false)
  const [opMsg, setOpMsg] = useState('')

  // ==================== 新增页面弹窗 ====================
  const [showAddModal, setShowAddModal] = useState(false)

  // ==================== 批次C: 收藏代码弹窗 ====================
  // 老师对当前页满意时点「⭐ 收藏代码」→ 填名称+备注 → 存入个人代码库（表 courseware_code_snippets），
  // 之后微调任何课件都可经 RefinePanel「📎 注入参考代码」把这段代码给AI当范本。
  const [showStarModal, setShowStarModal] = useState(false)  // 收藏弹窗开关
  const [starTitle, setStarTitle] = useState('')             // 收藏名称
  const [starNote, setStarNote] = useState('')               // 可选备注
  const [starSaving, setStarSaving] = useState(false)        // 保存请求进行中

  // ==================== 预览缩放 ====================
  const containerWidth = 912
  const previewScale = containerWidth / CW_WIDTH

  const activePage = currentNum > 0 ? currentNum : (pages[0]?.page_number || 0)
  const html = pages.find(p => p.page_number === activePage)?.html_content || ''
  const previewHtml = injectPreviewMode(html)

  // 批次A：切换选中页时强制退出源码编辑并丢弃草稿（草稿与页强绑定，跨页无意义；
  // 切页动作发生在胶片条点击→父级更新currentNum→本组件被动感知，故用effect兜底重置）
  useEffect(() => {
    setCodeEditing(false)
    setCodeDraft('')
  }, [activePage])

  // ==================== 消息自动清除辅助 ====================
  const showMsg = useCallback((msg: string, ms = 3000) => {
    setOpMsg(msg)
    setTimeout(() => setOpMsg(''), ms)
  }, [])

  // ==================== 批次C: 收藏代码处理 ====================

  /** 打开收藏弹窗：名称默认为当前页标题（老师可改） */
  const handleOpenStar = useCallback(() => {
    if (!coursewareId || activePage <= 0) return
    const page = pages.find(p => p.page_number === activePage)
    setStarTitle(page?.title || `第${activePage}页`)
    setStarNote('')
    setShowStarModal(true)
  }, [coursewareId, activePage, pages])

  /** 提交收藏：调 createCodeSnippet（HTML快照由服务端自取，前端不传大内容） */
  const handleSaveStar = useCallback(async () => {
    if (!coursewareId || activePage <= 0 || starSaving) return
    if (!starTitle.trim()) { showMsg('❌ 请给这条收藏起个名称'); return }
    setStarSaving(true)
    try {
      const r = await createCodeSnippet(coursewareId, activePage, starTitle.trim(), starNote.trim() || undefined)
      setShowStarModal(false)
      showMsg('✅ ' + r.message + '，微调面板「📎 注入参考代码」可用', 4000)
    } catch (e) {
      showMsg('❌ 收藏失败: ' + (e instanceof Error ? e.message : '未知错误'), 5000)
    } finally { setStarSaving(false) }
  }, [coursewareId, activePage, starSaving, starTitle, starNote, showMsg])

  // ==================== 批次A：源码编辑处理 ====================

  /** 进入源码编辑模式：以当前页HTML为初始草稿 */
  const handleStartCodeEdit = useCallback(() => {
    if (!editable || !coursewareId || operating || codeSaving) return
    setCodeDraft(html)
    setCodeEditing(true)
  }, [editable, coursewareId, operating, codeSaving, html])

  /** 取消源码编辑：草稿有改动时二次确认，防手滑丢内容 */
  const handleCancelCodeEdit = useCallback(() => {
    if (codeSaving) return
    if (codeDraft !== html && !window.confirm('放弃未保存的源码修改？')) return
    setCodeEditing(false)
    setCodeDraft('')
  }, [codeSaving, codeDraft, html])

  /** 保存源码编辑：结构自检 → savePageHtml 落库（后端自动存 manual 快照）→ 通知父级刷新 */
  const handleSaveCodeEdit = useCallback(async () => {
    if (!coursewareId || activePage <= 0 || codeSaving) return
    const content = codeDraft
    if (!content.trim()) {
      showMsg('❌ 源码内容为空，未保存')
      return
    }
    if (content === html) {
      showMsg('⚠️ 内容未变化，无需保存')
      setCodeEditing(false)
      setCodeDraft('')
      return
    }
    // 轻量结构自检：疑似残缺时二次确认（不硬拦——外部代码写法多样，老师可自行决定）
    const warn = divBalanceCheck(content)
    if (warn && !window.confirm(
      '⚠️ 结构自检提示：' + warn + '。\n\n仍要保存吗？\n（保存前系统会自动把当前版本存入历史，改坏了可在微调面板「📜 历史版本」一键回退）',
    )) return

    setCodeSaving(true)
    try {
      await savePageHtml(coursewareId, activePage, content)
      showMsg('✅ 第 ' + activePage + ' 页源码已保存（旧版已存入历史版本，可回退）', 4000)
      setCodeEditing(false)
      setCodeDraft('')
      onPagesChanged?.()
    } catch (e) {
      showMsg('❌ 保存失败: ' + (e instanceof Error ? e.message : '未知错误'), 5000)
    } finally {
      setCodeSaving(false)
    }
  }, [coursewareId, activePage, codeSaving, codeDraft, html, showMsg, onPagesChanged])

  /** 编辑区 Tab 键插入两个空格（不跳焦点），便于改缩进 */
  const handleCodeKeyDown = useCallback((e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Tab') {
      e.preventDefault()
      const ta = e.currentTarget
      const start = ta.selectionStart
      const end = ta.selectionEnd
      const next = codeDraft.slice(0, start) + '  ' + codeDraft.slice(end)
      setCodeDraft(next)
      // 恢复光标到插入空格之后（下一帧，等受控值回填）
      requestAnimationFrame(() => {
        ta.selectionStart = ta.selectionEnd = start + 2
      })
    }
  }, [codeDraft])

  // ==================== 拖拽排序处理 ====================

  const handleDragStart = useCallback((e: React.DragEvent, index: number) => {
    if (!editable || operating) return
    setDragIndex(index)
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', String(index))
  }, [editable, operating])

  const handleDragEnter = useCallback((e: React.DragEvent, index: number) => {
    e.preventDefault()
    if (dragIndex === null || dragIndex === index) return
    setDragOverIndex(index)
  }, [dragIndex])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
  }, [])

  const handleDrop = useCallback(async (e: React.DragEvent, dropIndex: number) => {
    e.preventDefault()
    if (!editable || !coursewareId || dragIndex === null || dragIndex === dropIndex) {
      setDragIndex(null)
      setDragOverIndex(null)
      return
    }

    const newPages = [...pages]
    const [moved] = newPages.splice(dragIndex, 1)
    newPages.splice(dropIndex, 0, moved)

    const pageIds = newPages.map(p => p.id).filter(Boolean) as string[]
    if (pageIds.length !== newPages.length) {
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
  }, [editable, coursewareId, dragIndex, pages, onPagesChanged, showMsg])

  const handleDragEnd = useCallback(() => {
    setDragIndex(null)
    setDragOverIndex(null)
    dragCounterRef.current = 0
  }, [])

  // ==================== 删除页面 ====================
  const handleDeletePage = useCallback(async (e: React.MouseEvent, pageNum: number) => {
    e.stopPropagation()
    if (!editable || !coursewareId || operating) return
    if (pages.length <= 1) {
      showMsg('⚠️ 至少保留一页，不能全部删除')
      return
    }
    const page = pages.find(p => p.page_number === pageNum)
    const pageTitle = page?.title || `第${pageNum}页`
    if (!window.confirm(`确定删除「${pageTitle}」(P${pageNum})？\n\n删除后该页的 HTML 内容和配图资产将一并清除，此操作不可撤销。`)) return

    setOperating(true)
    try {
      await deleteCWPage(coursewareId, pageNum)
      showMsg('✅ 已删除')
      if (activePage === pageNum) {
        const remaining = pages.filter(p => p.page_number !== pageNum)
        if (remaining.length > 0) onSelectPage(remaining[0].page_number)
      }
      onPagesChanged?.()
    } catch {
      showMsg('❌ 删除失败，请重试')
    } finally {
      setOperating(false)
    }
  }, [editable, coursewareId, operating, pages, activePage, onSelectPage, onPagesChanged, showMsg])

  // ==================== 新增页面（弹窗完成回调） ====================
  const handleAddDone = useCallback((newPageNumber: number) => {
    setShowAddModal(false)
    onSelectPage(newPageNumber)
    onPagesChanged?.()
  }, [onSelectPage, onPagesChanged])

  return <>
    {/* 操作结果消息条 */}
    {opMsg && <MsgBar msg={opMsg} />}

    {pages.length > 0 && (
      <div style={{ marginBottom: 20 }}>
        {/* 顶部标题栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontSize: 14, fontWeight: 600, color: C.textPrimary }}>📄 已生成 {pages.length} 页</span>
            {editable && (
              <span style={{ fontSize: 12, color: C.textMuted, fontWeight: 400 }}>
                拖拽调序 · 点 × 删除
              </span>
            )}
          </div>
          <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
            {/* 编辑模式：添加页面按钮放在顶栏右侧与全屏放映并排 */}
            {editable && (
              <button onClick={() => setShowAddModal(true)} disabled={operating} style={{
                padding: '6px 14px', borderRadius: 8, border: `1px dashed ${C.primary}`,
                background: C.primaryBg, color: C.primary, fontSize: 13, fontWeight: 600,
                cursor: operating ? 'default' : 'pointer', opacity: operating ? 0.5 : 1,
              }}>＋ 添加页面</button>
            )}
            {showSlideshow && (
              <button onClick={() => onSlideshow()} style={{
                padding: '6px 14px', borderRadius: 8, border: `1px solid ${C.primary}`,
                background: C.primaryBg, color: C.primary, fontSize: 13, fontWeight: 600, cursor: 'pointer',
              }}>🖥️ 全屏放映</button>
            )}
          </div>
        </div>

        {/*
         * 胶片条容器结构（修复 × 按钮截断）：
         * 外层 div：overflow:visible，让 × 按钮超出不被裁剪
         * 内层 div：overflow-x:auto 实现横向滚动，padding-top 给 × 按钮留空间
         */}
        <div style={{ overflow: 'visible', position: 'relative' }}>
          <div style={{
            display: 'flex', gap: 6, flexWrap: 'nowrap', overflowX: 'auto',
            paddingBottom: 6, paddingTop: editable ? 10 : 0, // 给 × 按钮留顶部空间
            alignItems: 'center',
          }}>
            {pages.map((gp, idx) => {
              const isActive = activePage === gp.page_number
              const isDragging = dragIndex === idx
              const isDragOver = dragOverIndex === idx

              return (
                <div
                  key={gp.page_number}
                  draggable={editable && !operating}
                  onDragStart={editable ? (e) => handleDragStart(e, idx) : undefined}
                  onDragEnter={editable ? (e) => handleDragEnter(e, idx) : undefined}
                  onDragOver={editable ? handleDragOver : undefined}
                  onDrop={editable ? (e) => handleDrop(e, idx) : undefined}
                  onDragEnd={editable ? handleDragEnd : undefined}
                  style={{
                    position: 'relative', flexShrink: 0,
                    opacity: isDragging ? 0.4 : 1,
                    transition: 'all 200ms',
                  }}
                >
                  {/* 拖拽插入指示线 */}
                  {editable && isDragOver && dragIndex !== null && dragIndex !== idx && (
                    <div style={{
                      position: 'absolute', left: -4, top: 2, bottom: 2, width: 3,
                      background: '#3B82F6', borderRadius: 2, zIndex: 10,
                    }} />
                  )}

                  {/* 页签按钮 */}
                  <button
                    onClick={() => onSelectPage(gp.page_number)}
                    title={`P${gp.page_number} ${gp.title}${editable ? '\n拖拽可调整顺序' : ''}`}
                    style={{
                      padding: '6px 10px', borderRadius: 8,
                      cursor: editable && !operating ? 'grab' : 'pointer',
                      whiteSpace: 'nowrap',
                      border: `2px solid ${isActive ? C.primary : C.border}`,
                      background: isActive ? C.primaryBg : C.white,
                      color: isActive ? C.primary : C.textPrimary,
                      fontSize: 12, fontWeight: isActive ? 600 : 400,
                      transition: 'all 200ms',
                    }}
                  >
                    <span style={{ fontWeight: 700 }}>P{gp.page_number}</span>
                    <span style={{ marginLeft: 5, color: C.textSecondary, fontSize: 11 }}>
                      {gp.title.length > 6 ? gp.title.slice(0, 6) + '…' : gp.title}
                    </span>
                  </button>

                  {/* × 删除按钮（editable 模式，右上角小红圆） */}
                  {editable && !operating && (
                    <button
                      onClick={(e) => handleDeletePage(e, gp.page_number)}
                      title={`删除 P${gp.page_number}`}
                      style={{
                        position: 'absolute', top: -6, right: -6,
                        width: 18, height: 18, borderRadius: '50%',
                        background: '#EF4444', color: '#fff',
                        border: '2px solid #fff', fontSize: 11, fontWeight: 700,
                        cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
                        lineHeight: 1, padding: 0,
                        boxShadow: '0 1px 4px rgba(0,0,0,0.2)',
                        zIndex: 5,
                      }}
                    >×</button>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      </div>
    )}

    {/* 空状态下也显示新增按钮 */}
    {pages.length === 0 && editable && (
      <div style={{ marginBottom: 20, textAlign: 'center', padding: '30px 0' }}>
        <div style={{ fontSize: 32, marginBottom: 8 }}>📄</div>
        <div style={{ fontSize: 14, color: C.textSecondary, marginBottom: 12 }}>暂无已生成的页面</div>
        <button onClick={() => setShowAddModal(true)} disabled={operating} style={{
          padding: '8px 20px', borderRadius: 8, border: `2px dashed ${C.primary}`,
          background: C.primaryBg, color: C.primary, fontSize: 14, fontWeight: 600,
          cursor: operating ? 'default' : 'pointer', opacity: operating ? 0.5 : 1,
        }}>＋ 添加页面</button>
      </div>
    )}

    {/* 大预览/源代码双视图 */}
    {html && (
      <div style={{ marginBottom: 20 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: C.textPrimary }}>
            {codeViewPageNum === activePage ? '💻' : '📺'} 第 {activePage} 页
            {codeViewPageNum === activePage ? (codeEditing ? '源代码（编辑中）' : '源代码') : '预览'}
          </div>
          <div style={{ display: 'flex', gap: 6 }}>
            {/* 批次A：源码视图下的编辑/保存/取消按钮组（仅 editable 且有课件ID时显示） */}
            {codeViewPageNum === activePage && editable && coursewareId && !codeEditing && (
              <button onClick={handleStartCodeEdit} disabled={operating || codeSaving} style={{
                padding: '4px 10px', borderRadius: 6, border: '1px solid #059669',
                background: 'rgba(5,150,105,0.06)', color: '#059669', fontSize: 12, fontWeight: 600,
                cursor: (operating || codeSaving) ? 'default' : 'pointer',
              }}>✏️ 编辑源码</button>
            )}
            {codeViewPageNum === activePage && codeEditing && (
              <>
                <button onClick={handleSaveCodeEdit} disabled={codeSaving} style={{
                  padding: '4px 12px', borderRadius: 6, border: 'none',
                  background: codeSaving ? '#E5E7EB' : '#059669', color: codeSaving ? '#9CA3AF' : '#fff',
                  fontSize: 12, fontWeight: 600, cursor: codeSaving ? 'default' : 'pointer',
                }}>{codeSaving ? '⏳ 保存中...' : '💾 保存'}</button>
                <button onClick={handleCancelCodeEdit} disabled={codeSaving} style={{
                  padding: '4px 10px', borderRadius: 6, border: '1px solid #EF4444',
                  background: 'transparent', color: '#EF4444', fontSize: 12,
                  cursor: codeSaving ? 'default' : 'pointer',
                }}>✖ 取消</button>
              </>
            )}
            <button onClick={() => {
              if (codeViewPageNum === activePage) {
                // 批次A：从源码切回预览——编辑中且草稿有改动先确认，防误丢
                if (codeEditing && codeDraft !== html && !window.confirm('放弃未保存的源码修改？')) return
                setCodeEditing(false)
                setCodeDraft('')
                setCodeViewPageNum(0)
              } else setCodeViewPageNum(activePage)
            }} style={{
              padding: '4px 10px', borderRadius: 6,
              border: `1px solid ${codeViewPageNum === activePage ? '#7C3AED' : C.border}`,
              background: codeViewPageNum === activePage ? 'rgba(124,58,237,0.06)' : 'transparent',
              color: codeViewPageNum === activePage ? '#7C3AED' : C.textSecondary,
              fontSize: 12, cursor: 'pointer',
            }}>{codeViewPageNum === activePage ? '📺 预览' : '💻 源代码'}</button>
            {/* 批次C: 收藏当前页代码到个人代码库（仅编辑模式；预览/源码视图均可点） */}
            {editable && coursewareId && (
              <button onClick={handleOpenStar} disabled={starSaving || codeSaving}
                title="把这一页的代码收藏进我的代码库，之后微调任何课件都能注入参考"
                style={{
                  padding: '4px 10px', borderRadius: 6, border: '1px solid #F59E0B',
                  background: 'rgba(245,158,11,0.06)', color: '#D97706', fontSize: 12, fontWeight: 600,
                  cursor: (starSaving || codeSaving) ? 'default' : 'pointer',
                }}>⭐ 收藏代码</button>
            )}
            <button onClick={() => {
              navigator.clipboard.writeText(html).then(() => alert('源代码已复制到剪贴板')).catch(() => {})
            }} style={{
              padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`,
              background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer',
            }}>📋 复制代码</button>
            <button onClick={() => onFullscreen(activePage)} style={{
              padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`,
              background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer',
            }}>🔍 全屏预览</button>
            <button onClick={() => onSlideshow(activePage)} style={{
              padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`,
              background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer',
            }}>🖥️ 放映</button>
          </div>
        </div>

        {codeViewPageNum === activePage ? (
          codeEditing ? (
            /* 批次A：源码编辑模式——可编辑 textarea + 常驻画布契约提示 */
            <div>
              <textarea
                value={codeDraft}
                onChange={e => setCodeDraft(e.target.value)}
                onKeyDown={handleCodeKeyDown}
                spellCheck={false}
                disabled={codeSaving}
                style={{
                  width: '100%', height: 500, boxSizing: 'border-box',
                  padding: '12px 14px', borderRadius: 14, border: '2px solid #059669',
                  background: '#1e1e1e', color: '#d4d4d4', outline: 'none',
                  fontFamily: 'Monaco, Consolas, "Courier New", monospace', fontSize: 12, lineHeight: 1.7,
                  resize: 'vertical', whiteSpace: 'pre', overflowWrap: 'normal', overflowX: 'auto',
                }}
              />
              <div style={{ marginTop: 6, fontSize: 11, color: '#9CA3AF' }}>
                💡 可直接修改本页源码，也可整页替换为外部好代码。注意保持最外层
                {' <div style="width:1920px;height:1080px..." class="cw-page"> '}
                画布结构，否则显示会变形。保存前系统会自动把当前版本存入「📜 历史版本」，改坏了可一键回退。Tab 键=插入两个空格。
              </div>
            </div>
          ) : (
            /* 只读源码视图（行号表格，原样保留） */
            <div style={{
              width: '100%', maxHeight: 500, overflow: 'auto', borderRadius: 14,
              border: `1px solid ${C.border}`, background: '#1e1e1e',
              fontFamily: 'Monaco, Consolas, "Courier New", monospace', fontSize: 12, lineHeight: 1.7,
            }}>
              <table style={{ borderCollapse: 'collapse', width: '100%' }}><tbody>
                {html.split('\n').map((line: string, i: number) => (
                  <tr key={i}>
                    <td style={{
                      width: 50, minWidth: 50, textAlign: 'right', padding: '0 10px 0 8px',
                      color: '#858585', userSelect: 'none', verticalAlign: 'top',
                      borderRight: '1px solid #333', whiteSpace: 'nowrap',
                    }}>{i + 1}</td>
                    <td style={{
                      padding: '0 12px', color: '#d4d4d4', whiteSpace: 'pre', wordBreak: 'break-all',
                    }}>{line || ' '}</td>
                  </tr>
                ))}
              </tbody></table>
            </div>
          )
        ) : (
          <div onClick={() => onSlideshow(activePage)} style={{
            width: '100%', height: Math.ceil(CW_HEIGHT * previewScale), position: 'relative',
            overflow: 'hidden', borderRadius: 14, border: `1px solid ${C.border}`,
            background: '#f8fafc', cursor: 'pointer',
          }}>
            <iframe
              key={`p${activePage}-${previewHtml.length}`}
              srcDoc={previewHtml}
              scrolling="no"
              style={{
                width: CW_WIDTH, height: CW_HEIGHT, border: 'none', pointerEvents: 'none',
                transform: `scale(${previewScale})`, transformOrigin: 'top left',
                position: 'absolute', top: 0, left: 0, overflow: 'hidden',
              }}
              sandbox="allow-scripts"
              title={`预览-P${activePage}`}
            />
          </div>
        )}
      </div>
    )}

    {/* 批次C: 收藏代码弹窗（填名称+备注，HTML快照由服务端自取） */}
    {showStarModal && coursewareId && (
      <div onClick={starSaving ? undefined : () => setShowStarModal(false)} style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        background: 'rgba(0,0,0,0.4)', zIndex: 99990,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}>
        <div onClick={e => e.stopPropagation()} style={{
          background: '#fff', borderRadius: 16, padding: '26px 30px', width: '100%', maxWidth: 460,
          boxShadow: '0 20px 60px rgba(0,0,0,0.15)',
        }}>
          <h3 style={{ margin: '0 0 6px', fontSize: 17, fontWeight: 600, color: '#1F2937' }}>⭐ 收藏第 {activePage} 页代码</h3>
          <div style={{ fontSize: 12, color: '#9CA3AF', marginBottom: 16, lineHeight: 1.7 }}>
            收藏的是这一页<b>此刻</b>的完整代码快照（之后改动本页不影响收藏）。收藏后在微调面板「📎 注入参考代码」里可选用。
          </div>
          <div style={{ marginBottom: 12 }}>
            <label style={{ display: 'block', fontSize: 13, color: '#6B7280', marginBottom: 4, fontWeight: 500 }}>收藏名称 *</label>
            <input value={starTitle} onChange={e => setStarTitle(e.target.value)} autoFocus
              placeholder="例如：左右对比卡片布局" disabled={starSaving}
              style={{ width: '100%', padding: '8px 12px', borderRadius: 8, border: '1px solid #E5E7EB', fontSize: 14, outline: 'none', boxSizing: 'border-box' }} />
          </div>
          <div style={{ marginBottom: 18 }}>
            <label style={{ display: 'block', fontSize: 13, color: '#6B7280', marginBottom: 4, fontWeight: 500 }}>备注<span style={{ color: '#9CA3AF', fontSize: 12, marginLeft: 4 }}>（可选：这段代码好在哪/适用什么场景）</span></label>
            <textarea value={starNote} onChange={e => setStarNote(e.target.value)} rows={2} disabled={starSaving}
              placeholder="例如：两栏对比排版很清爽，适合概念对比页"
              style={{ width: '100%', padding: '8px 12px', borderRadius: 8, border: '1px solid #E5E7EB', fontSize: 13, outline: 'none', boxSizing: 'border-box', resize: 'vertical', fontFamily: 'inherit' }} />
          </div>
          <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
            <button onClick={() => setShowStarModal(false)} disabled={starSaving} style={{
              padding: '8px 20px', borderRadius: 8, border: '1px solid #E5E7EB',
              background: 'transparent', color: '#6B7280', fontSize: 14, cursor: starSaving ? 'default' : 'pointer',
            }}>取消</button>
            <button onClick={handleSaveStar} disabled={starSaving} style={{
              padding: '8px 24px', borderRadius: 8, border: 'none',
              background: starSaving ? '#E5E7EB' : '#F59E0B', color: starSaving ? '#9CA3AF' : '#fff',
              fontSize: 14, fontWeight: 600, cursor: starSaving ? 'default' : 'pointer',
            }}>{starSaving ? '⏳ 收藏中...' : '⭐ 收藏'}</button>
          </div>
        </div>
      </div>
    )}

    {/* 快速添加页面弹窗 */}
    {showAddModal && coursewareId && (
      <AddPageModal
        coursewareId={coursewareId}
        currentPageCount={pages.length}
        onDone={handleAddDone}
        onClose={() => setShowAddModal(false)}
      />
    )}
  </>
}

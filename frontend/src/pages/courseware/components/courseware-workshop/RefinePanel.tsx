/**
 * RefinePanel.tsx — 页面微调面板（批次W2从主页面抽出，工作台默认Tab）
 *
 * 内容：单页AI微调（支持附截图/Ctrl+V粘贴截图走多模态）+ 单页从零重生 + 页面级历史版本回退 + 版本对比 + 就地文字编辑。
 * 选中页跟随上方大预览框（批次4b口径，pageNum=父级buildPreviewNum）。
 * W2改进：自带消息条——原Step5没有buildMessage的展示位，微调成功/失败提示此前不可见。
 *
 * 页面级版本与回退：
 *   每次"单页微调/单页重生/就地编辑"前，后端自动把旧 HTML 存为一个版本快照（最多保留近20版）。
 *   本面板提供「📜 历史版本」按钮 → 展开该页版本列表 → 任选一版「回退」（回退前二次确认）。
 *   回退本身可逆：回退时后端会把【当前】内容也另存为一版，故回退后还能再退回。
 *   列表为空表示该页还没产生过版本（首次生成不算"覆盖"，微调/重生后才开始累积）。
 *
 * 【就地文字编辑·新增】工具条「✏️ 就地编辑」按钮 → 打开全屏浮层编辑器 InlineTextEditor：
 *   老师在预览里直接点选某段文字，就地改【文字内容/字号/颜色】三样，纯 DOM 操作、不新增节点、
 *   不产生脏 DOM。保存走 savePageHtml（不调 AI，覆盖前存 manual 版本快照，可回退/对比）。
 *   定位为"改个错字、调个颜色不值得跑一次 AI"的轻量补充，与 AI 微调互补。保存后经 onPageUpdated 刷新大预览。
 *
 * 【版本对比·新增】每条历史版本行新增「👁 对比」按钮：
 *   点击后弹出全屏对比弹窗，左=该历史版渲染、右=当前版渲染（1920×1080 等比缩入），
 *   顶部可切"源码对比"文本视图（左右并排显示两版 HTML 原文，便于精确核对改了什么）。
 *   历史版 HTML 走 getPageVersionDetail 按需拉取；当前版 HTML 走 getCoursewarePages 拿当前页 html_content，
 *   故本组件 props 不变、父组件零改动。对比是只读操作，绝不改任何页面状态。
 *
 * 【P1-05 体验修复】微调输入框草稿 sessionStorage 暂存（防刷新/切走丢失）：
 *   老师在输入框敲了一段修改意见还没点「AI微调」，若误刷新页面或切到别的Tab再回来，
 *   原先 refineInput 是纯 useState，会被清空、白打一遍。本次改为：
 *     · 草稿按「课件ID + 页号」分键存 sessionStorage（不同课件、不同页各存各的，互不串）；
 *     · 输入变化即写入（去抖动靠 React 受控更新天然合并，无需额外定时器）；
 *     · 切换页号/挂载时按当前键回填草稿；
 *     · 微调成功后清空该键（已提交的意见无需再留）。
 *   只暂存文字草稿，截图(refineImage)是大体积 dataURI 不入 sessionStorage（避免超配额）。
 *
 * 【B-P1-11 体验修复】微调输入框由单行 input 改为可拖高的多行 textarea：
 *   原先是单行 <input>，老师写稍长的修改意见（多条诉求）时看不全、无法换行分点。
 *   现改为 <textarea>：
 *     · 默认 2 行高，CSS resize:vertical 可手动向下拖高；
 *     · 回车=提交微调（与原 input 的 Enter 提交行为一致，老师肌肉记忆不变）；
 *     · Shift+回车=插入换行（与备课对话输入框 ConversationInputBar 同一套交互口径）；
 *     · Ctrl+V 粘贴截图逻辑不变（onPaste 仍拦截剪贴板图片）。
 */
import { useState, useEffect } from 'react'
import { refinePage, regenerateCWPage, listPageVersions, rollbackPage, getPageVersionDetail, getCoursewarePages } from '@/api/coursewares'
import type { PageVersionEntry } from '@/api/coursewares'
import { C, CW_WIDTH, CW_HEIGHT } from './workshopConstants'
import { injectPreviewMode } from './previewInject'
import InlineTextEditor from './InlineTextEditor'
import SnippetInjectPicker from './SnippetInjectPicker'
import type { InjectedSnippet } from './SnippetInjectPicker'

interface Props {
  coursewareId: string
  /** 当前选中页（父级 buildPreviewNum） */
  pageNum: number
  /** 微调/重生/回退/就地编辑成功后回写该页HTML（父级更新 generatedPages 刷新预览） */
  onPageUpdated: (pageNum: number, html: string) => void
}

// P1-05: 草稿 sessionStorage 键前缀；按 课件ID + 页号 分键，不同课件/页互不串
const REFINE_DRAFT_PREFIX = 'tedna_cw_refine_draft_'
const refineDraftKey = (cwId: string, pageNum: number) => `${REFINE_DRAFT_PREFIX}${cwId}_${pageNum}`

// P1-05: 安全读取草稿（sessionStorage 不可用/异常时静默返回空串，绝不抛错阻断面板）
function readRefineDraft(cwId: string, pageNum: number): string {
  if (!cwId || pageNum <= 0) return ''
  try {
    return sessionStorage.getItem(refineDraftKey(cwId, pageNum)) || ''
  } catch { return '' }
}

// P1-05: 安全写入/清除草稿（空串视为清除该键，避免留下空草稿）
function saveRefineDraft(cwId: string, pageNum: number, text: string) {
  if (!cwId || pageNum <= 0) return
  try {
    const key = refineDraftKey(cwId, pageNum)
    if (text) sessionStorage.setItem(key, text)
    else sessionStorage.removeItem(key)
  } catch { /* 配额满/隐私模式等：静默忽略，不影响微调主流程 */ }
}

// 版本对比弹窗内部状态：加载中 / 已就绪。承载"某历史版"与"当前版"两份 HTML。
interface CompareState {
  open: boolean            // 弹窗是否打开
  loading: boolean         // 两份 HTML 拉取中
  error: string            // 拉取失败信息（非空则弹窗内显红字）
  versionNo: number        // 正在对比的历史版本号（弹窗标题用）
  sourceLabel: string      // 历史版本来源中文标签（弹窗标题用）
  historyHtml: string      // 历史版完整 HTML（左侧）
  currentHtml: string      // 当前版完整 HTML（右侧）
  mode: 'render' | 'code'  // 视图模式：render=iframe渲染对比 / code=源码文本对比
}

const emptyCompare: CompareState = {
  open: false, loading: false, error: '', versionNo: 0, sourceLabel: '',
  historyHtml: '', currentHtml: '', mode: 'render',
}

export default function RefinePanel({ coursewareId, pageNum, onPageUpdated }: Props) {
  // P1-05: 初值直接从 sessionStorage 回填当前 课件+页 的草稿（首次挂载即恢复）
  const [refineInput, setRefineInput] = useState(() => readRefineDraft(coursewareId, pageNum))
  const [refineRunning, setRefineRunning] = useState(false)
  const [refineImage, setRefineImage] = useState('')   // 截图dataURI(走多模态，不入sessionStorage)
  const [regenRunning, setRegenRunning] = useState(false)
  const [message, setMessage] = useState('')

  // ---- 页面级历史版本与回退 状态 ----
  const [showVersions, setShowVersions] = useState(false)        // 历史版本弹层开关
  const [versions, setVersions] = useState<PageVersionEntry[]>([]) // 当前页版本列表（倒序，最新在前）
  const [versionsLoading, setVersionsLoading] = useState(false)  // 列表加载中
  const [rollbackingId, setRollbackingId] = useState('')         // 正在回退的版本id（禁用对应按钮）

  // ---- 版本对比弹窗 状态 ----
  const [compare, setCompare] = useState<CompareState>(emptyCompare)

  // ---- 就地文字编辑浮层 开关 ----
  const [showInlineEditor, setShowInlineEditor] = useState(false)

  // ---- 批次C: 微调参考代码注入 状态 ----
  // 老师从代码收藏库选中的参考代码；提交微调时以标记块形式拼进指令（前端拼接注入，透明可控）
  const [injectedSnippet, setInjectedSnippet] = useState<InjectedSnippet | null>(null)

  // P1-05: 受控更新输入框——同时写 state 与 sessionStorage 草稿（即敲即存，刷新不丢）
  const updateRefineInput = (text: string) => {
    setRefineInput(text)
    saveRefineDraft(coursewareId, pageNum, text)
  }

  // 切换选中页时：收起弹层并清空上一页的版本列表，避免串页显示；
  // P1-05: 同时按新页号回填该页的草稿（不同页各记各的修改意见）
  useEffect(() => {
    setShowVersions(false)
    setVersions([])
    setRollbackingId('')
    setCompare(emptyCompare)  // 切页关闭对比弹窗，避免残留上一页的对比内容
    setShowInlineEditor(false) // 切页关闭就地编辑浮层，避免编辑器停留在旧页
    setRefineInput(readRefineDraft(coursewareId, pageNum))
    setRefineImage('')  // 切页清掉上一页的截图（截图与页强相关，不跨页保留）
    setInjectedSnippet(null)  // 批次C: 切页清掉已注入的参考代码（注入与本次微调强相关，不跨页保留）
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [coursewareId, pageNum])

  // 单页AI微调(批次4a: 支持随附截图走多模态; 微调=保留页内已插入图片)
  const handleRefinePage = async () => {
    if (!coursewareId || pageNum <= 0 || !refineInput.trim()) return
    setRefineRunning(true)
    try {
      // 批次C: 若已注入参考代码，以标记块形式拼在修改意见之后（截断至约24000字符防token超限）。
      //   措辞明确：参照范本的布局骨架/交互方式/视觉手法，教学内容仍以当前页为准、不照抄范本文字。
      let finalInstruction = refineInput.trim()
      if (injectedSnippet) {
        const MAX_REF_LEN = 24000
        const refHtml = injectedSnippet.html.length > MAX_REF_LEN
          ? injectedSnippet.html.slice(0, MAX_REF_LEN) + '\n<!-- 参考代码过长，已截断 -->'
          : injectedSnippet.html
        finalInstruction = finalInstruction
          + '\n\n【参考代码范本·开始】（收藏名：' + injectedSnippet.title + '）\n'
          + '以下是我指定的参考代码。请在落实上面修改意见时，参照这段范本的布局骨架、交互方式与视觉手法；'
          + '但教学内容（文字/数据/图片）仍以当前页为准，不要照抄范本里的文字内容。\n'
          + '```html\n' + refHtml + '\n```\n【参考代码范本·结束】'
      }
      const result = await refinePage(coursewareId, pageNum, finalInstruction, refineImage || undefined)
      if (result.html_content) onPageUpdated(pageNum, result.html_content)
      // P1-05: 微调成功 → 清空输入 + 清除该页草稿（已提交的意见无需再留）
      updateRefineInput('')
      setRefineImage('')
      setInjectedSnippet(null)  // 批次C: 本次注入已随微调消费，成功后清除
      setMessage('✅ ' + result.message)
      // 微调成功后该页新增了一个版本快照；若历史弹层正展开则刷新列表
      if (showVersions) loadVersions()
    } catch (e) { setMessage('❌ 微调失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setRefineRunning(false) }
  }

  // 单页从零重生(重生=不保留页内已插入图片; 后端无并发锁故运行态禁用按钮)
  const handleRegeneratePage = async () => {
    if (!coursewareId || pageNum <= 0 || regenRunning || refineRunning) return
    if (!confirm('⚠️ 重生第 ' + pageNum + ' 页将按方案从零重画整页，会清空本页已插入的图片（图片资产仍在多媒体库，可重新插入）。确定重生？')) return
    setRegenRunning(true); setMessage('🔄 正在重生第 ' + pageNum + ' 页，请稍候...')
    try {
      const result = await regenerateCWPage(coursewareId, pageNum)
      if (result.html_content) onPageUpdated(pageNum, result.html_content)
      setMessage('✅ ' + result.message)
      // 重生成功后该页新增了一个版本快照；若历史弹层正展开则刷新列表
      if (showVersions) loadVersions()
    } catch (e) { setMessage('❌ 重生失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setRegenRunning(false) }
  }

  // 拉取当前页版本列表
  const loadVersions = async () => {
    if (!coursewareId || pageNum <= 0) return
    setVersionsLoading(true)
    try {
      const result = await listPageVersions(coursewareId, pageNum)
      setVersions(result.versions || [])
    } catch (e) {
      setMessage('❌ 加载历史版本失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setVersionsLoading(false) }
  }

  // 点击「历史版本」按钮：展开则拉取列表，收起则仅关闭
  const handleToggleVersions = () => {
    const next = !showVersions
    setShowVersions(next)
    if (next) loadVersions()
  }

  // 回退到指定版本(二次确认; 回退本身可逆——后端会把当前内容另存为rollback版)
  const handleRollback = async (v: PageVersionEntry) => {
    if (!coursewareId || pageNum <= 0 || rollbackingId) return
    if (!confirm(
      '确定将第 ' + pageNum + ' 页回退到【' + v.source_label + ' · v' + v.version_no + '】这一版吗？\n\n'
      + '回退不会丢失当前内容：系统会先把"当前这一版"也存为一条历史，之后你随时可以再退回来。',
    )) return
    setRollbackingId(v.id)
    setMessage('↩️ 正在回退第 ' + pageNum + ' 页到 v' + v.version_no + '...')
    try {
      const result = await rollbackPage(coursewareId, pageNum, v.id)
      if (result.html_content) onPageUpdated(pageNum, result.html_content)
      setMessage('✅ ' + result.message)
      // 回退会新增一条 rollback 版本，刷新列表反映最新
      await loadVersions()
    } catch (e) {
      setMessage('❌ 回退失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setRollbackingId('') }
  }

  // 打开版本对比弹窗：并行拉取【历史版】与【当前版】两份完整 HTML。
  //   历史版走 getPageVersionDetail（按 versionId 取）；
  //   当前版走 getCoursewarePages 取当前页的 html_content（避免依赖父级传参，props 保持不变）。
  const handleOpenCompare = async (v: PageVersionEntry) => {
    if (!coursewareId || pageNum <= 0) return
    // 先打开弹窗显示加载态，避免用户点击后无反馈
    setCompare({
      ...emptyCompare, open: true, loading: true,
      versionNo: v.version_no, sourceLabel: v.source_label,
    })
    try {
      // 并行拉两份：历史版完整 HTML + 当前全部页面（从中取当前页）
      const [detail, pages] = await Promise.all([
        getPageVersionDetail(coursewareId, pageNum, v.id),
        getCoursewarePages(coursewareId),
      ])
      const curPage = (pages || []).find(p => p.page_number === pageNum)
      const curHtml = curPage?.html_content || ''
      setCompare(prev => ({
        ...prev,
        loading: false,
        error: '',
        versionNo: detail.version_no || v.version_no,
        sourceLabel: detail.source_label || v.source_label,
        historyHtml: detail.html_content || '',
        currentHtml: curHtml,
      }))
    } catch (e) {
      setCompare(prev => ({
        ...prev,
        loading: false,
        error: '加载对比内容失败: ' + (e instanceof Error ? e.message : '未知错误'),
      }))
    }
  }

  // 关闭对比弹窗
  const handleCloseCompare = () => setCompare(emptyCompare)

  // 就地编辑保存成功回调：回写该页 HTML（复用 onPageUpdated），并刷新可能展开的版本列表
  const handleInlineEditorSaved = (pn: number, html: string) => {
    onPageUpdated(pn, html)
    setMessage('✅ 第 ' + pn + ' 页文字修改已保存')
    // 就地编辑保存后后端新增了一个 manual 版本快照；若历史弹层正展开则刷新列表
    if (showVersions) loadVersions()
  }

  // 共用——将图片文件读为 dataURI 存入 refineImage(8MB上限)
  const loadRefineImageFile = (f: File, fromPaste = false) => {
    if (f.size > 8 * 1024 * 1024) { setMessage('❌ 截图不能超过8MB'); return }
    const reader = new FileReader()
    reader.onload = () => {
      setRefineImage(typeof reader.result === 'string' ? reader.result : '')
      if (fromPaste) setMessage('✅ 已从剪贴板粘贴截图，微调将参考该图')
    }
    reader.onerror = () => setMessage('❌ 截图读取失败')
    reader.readAsDataURL(f)
  }

  // 微调输入框 Ctrl+V 粘贴剪贴板图片(仅含图片时拦截; 纯文本粘贴走默认行为)
  // B-P1-11: 事件类型由 HTMLInputElement 改为 HTMLTextAreaElement（控件已换 textarea）
  const handleRefinePaste = (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
    const items = e.clipboardData?.items
    if (!items) return
    for (let i = 0; i < items.length; i++) {
      const it = items[i]
      if (it.type && it.type.startsWith('image/')) {
        const f = it.getAsFile()
        if (!f) continue
        e.preventDefault()
        loadRefineImageFile(f, true)
        return
      }
    }
  }

  // B-P1-11: textarea 键盘处理——回车提交微调，Shift+回车换行（与备课对话输入框同口径）
  const handleRefineKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()  // 阻止默认换行，改为提交
      if (!refineRunning && pageNum > 0 && refineInput.trim()) handleRefinePage()
    }
    // Shift+回车：不拦截，走 textarea 默认换行行为
  }

  // 格式化版本时间（ISO → 本地 月-日 时:分）
  const fmtVersionTime = (iso: string) => {
    if (!iso) return ''
    const d = new Date(iso)
    if (isNaN(d.getTime())) return iso
    const pad = (n: number) => (n < 10 ? '0' + n : '' + n)
    return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
  }

  return (
    <div style={{ marginTop: 16, padding: '16px', borderRadius: 10, border: `1px solid ${C.border}`, background: '#FAFAFA' }}>
      <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🎨 对某页不满意？在上方预览区选中该页，输入修改意见</div>
      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', alignItems: 'flex-start' }}>
        <span style={{ padding: '8px 12px', borderRadius: 8, background: C.primaryBg, color: C.primary, fontSize: 13, fontWeight: 600, whiteSpace: 'nowrap', marginTop: 2 }}>
          当前：第 {pageNum || '—'} 页
        </span>
        {/* B-P1-11: 单行input → 可拖高多行textarea；回车提交、Shift+回车换行 */}
        <textarea value={refineInput} onChange={e => updateRefineInput(e.target.value)}
          placeholder="例如：标题字号再大一些、增加图片占位...（回车提交，Shift+回车换行；可 Ctrl+V 粘贴截图，先在上方选要改的页）"
          onKeyDown={handleRefineKeyDown}
          onPaste={handleRefinePaste}
          rows={2}
          style={{ flex: 1, padding: '10px 14px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 14, outline: 'none', minWidth: 200, resize: 'vertical', fontFamily: 'inherit', lineHeight: 1.6, boxSizing: 'border-box' }}
          disabled={refineRunning} />
        <button onClick={handleRefinePage} disabled={refineRunning || pageNum <= 0 || !refineInput.trim()}
          style={{ padding: '10px 20px', borderRadius: 8, border: 'none', background: pageNum > 0 && refineInput.trim() && !refineRunning ? '#7C3AED' : '#E5E7EB', color: pageNum > 0 && refineInput.trim() && !refineRunning ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: pageNum > 0 && refineInput.trim() && !refineRunning ? 'pointer' : 'default', whiteSpace: 'nowrap', marginTop: 2 }}>
          {refineRunning ? '⏳ 微调中...' : '🎨 AI微调'}
        </button>
      </div>
      {/* 截图粘贴 + 就地改文字 + 历史版本 + 重生本页 */}
      <div style={{ display: 'flex', gap: 10, marginTop: 10, flexWrap: 'wrap', alignItems: 'center' }}>
        {refineImage ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <img src={refineImage} alt="参考截图" style={{ width: 40, height: 40, objectFit: 'cover', borderRadius: 6, border: '2px solid #7C3AED' }} />
            <span style={{ fontSize: 11, color: '#7C3AED' }}>已附截图(微调将参考)</span>
            <button onClick={() => setRefineImage('')} disabled={refineRunning || regenRunning} style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid #EF4444', background: 'transparent', color: '#EF4444', fontSize: 11, cursor: (refineRunning || regenRunning) ? 'default' : 'pointer' }}>移除</button>
          </div>
        ) : (
          <button onClick={() => {
            const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*'
            inp.onchange = (ev) => {
              const f = (ev.target as HTMLInputElement).files?.[0]
              if (!f) return
              loadRefineImageFile(f)
            }; inp.click()
          }} disabled={refineRunning || regenRunning} style={{ padding: '8px 14px', borderRadius: 8, border: '1px dashed #7C3AED', background: 'rgba(124,58,237,0.04)', color: '#7C3AED', fontSize: 13, cursor: (refineRunning || regenRunning) ? 'default' : 'pointer' }}>📷 附截图微调（或在输入框 Ctrl+V 粘贴）</button>
        )}
        <div style={{ flex: 1 }} />
        {/* 批次C: 注入参考代码选择器（从我的代码收藏选一条，微调时AI参照其布局/交互/视觉） */}
        <SnippetInjectPicker
          injected={injectedSnippet}
          onInject={setInjectedSnippet}
          onRemove={() => setInjectedSnippet(null)}
          disabled={refineRunning || regenRunning || pageNum <= 0}
        />
        {/* 就地改文字按钮：打开全屏浮层编辑器，点选文字改内容/字号/颜色（不调AI，轻量补充） */}
        <button onClick={() => setShowInlineEditor(true)} disabled={pageNum <= 0 || refineRunning || regenRunning}
          title={pageNum <= 0 ? '请先在上方预览区选中页' : '点选文字改内容/字号/颜色/加粗/字体，点选图片可替换（不调AI、不改版式）'}
          style={{ padding: '8px 16px', borderRadius: 8, border: `1px solid ${(pageNum > 0 && !refineRunning && !regenRunning) ? '#0EA5E9' : C.border}`, background: (pageNum > 0 && !refineRunning && !regenRunning) ? '#F0F9FF' : '#fff', color: (pageNum > 0 && !refineRunning && !regenRunning) ? '#0284C7' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (pageNum > 0 && !refineRunning && !regenRunning) ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
          ✏️ 就地编辑
        </button>
        {/* 历史版本按钮：展开/收起当前页版本列表 */}
        <button onClick={handleToggleVersions} disabled={pageNum <= 0}
          title={pageNum <= 0 ? '请先在上方预览区选中页' : '查看本页历次微调/重生前的版本，可一键回退（回退可逆）或与当前版对比'}
          style={{ padding: '8px 16px', borderRadius: 8, border: `1px solid ${showVersions ? '#2563EB' : C.border}`, background: showVersions ? '#EFF6FF' : '#fff', color: pageNum > 0 ? '#2563EB' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: pageNum > 0 ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
          📜 历史版本{showVersions ? ' ▲' : ' ▼'}
        </button>
        <button onClick={handleRegeneratePage} disabled={pageNum <= 0 || regenRunning || refineRunning}
          title={pageNum <= 0 ? '请先在上方预览区选中页' : '按方案从零重画本页(会清空本页已插入的图片)'}
          style={{ padding: '8px 18px', borderRadius: 8, border: 'none', background: (pageNum > 0 && !regenRunning && !refineRunning) ? 'linear-gradient(135deg, #F59E0B, #EF4444)' : '#E5E7EB', color: (pageNum > 0 && !regenRunning && !refineRunning) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (pageNum > 0 && !regenRunning && !refineRunning) ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
          {regenRunning ? '⏳ 重生中...' : '🔄 重生本页'}
        </button>
      </div>

      {/* 历史版本列表弹层 */}
      {showVersions && (
        <div style={{ marginTop: 12, padding: '12px 14px', borderRadius: 8, border: `1px solid ${C.border}`, background: '#fff' }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>
            📜 第 {pageNum} 页历史版本
            <span style={{ fontWeight: 400, color: '#9CA3AF', marginLeft: 8 }}>
              （仅记录"微调/重生/就地编辑前"的旧版，最多保留近20版；可「👁 对比」当前版或「↩️ 回退」，回退可逆）
            </span>
          </div>
          {versionsLoading ? (
            <div style={{ fontSize: 13, color: '#9CA3AF', padding: '12px 0' }}>⏳ 加载中...</div>
          ) : versions.length === 0 ? (
            <div style={{ fontSize: 13, color: '#9CA3AF', padding: '12px 0' }}>
              暂无历史版本，微调/重生后自动生成。
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8, maxHeight: 280, overflowY: 'auto' }}>
              {versions.map(v => (
                <div key={v.id} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 10px', borderRadius: 6, border: `1px solid ${C.border}`, background: '#FAFAFA' }}>
                  <span style={{ padding: '2px 8px', borderRadius: 4, background: '#EEF2FF', color: '#4F46E5', fontSize: 12, fontWeight: 600, whiteSpace: 'nowrap' }}>
                    v{v.version_no}
                  </span>
                  <span style={{ fontSize: 13, color: C.textPrimary, whiteSpace: 'nowrap' }}>{v.source_label}</span>
                  <span style={{ fontSize: 12, color: '#9CA3AF', whiteSpace: 'nowrap' }}>{fmtVersionTime(v.created_at)}</span>
                  {v.note ? (
                    <span style={{ fontSize: 12, color: '#6B7280', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={v.note}>
                      {v.note}
                    </span>
                  ) : null}
                  <div style={{ flex: 1 }} />
                  {/* 对比按钮：打开全屏对比弹窗（左=此历史版，右=当前版） */}
                  <button onClick={() => handleOpenCompare(v)} disabled={!!rollbackingId || compare.loading}
                    title="与当前版并排对比（渲染 + 源码），只读不修改"
                    style={{ padding: '5px 12px', borderRadius: 6, border: '1px solid #7C3AED', background: '#fff', color: '#7C3AED', fontSize: 12, fontWeight: 600, cursor: (rollbackingId || compare.loading) ? 'default' : 'pointer', whiteSpace: 'nowrap' }}>
                    👁 对比当前版
                  </button>
                  <button onClick={() => handleRollback(v)} disabled={!!rollbackingId}
                    style={{ padding: '5px 14px', borderRadius: 6, border: '1px solid #2563EB', background: rollbackingId === v.id ? '#DBEAFE' : '#fff', color: '#2563EB', fontSize: 12, fontWeight: 600, cursor: rollbackingId ? 'default' : 'pointer', whiteSpace: 'nowrap' }}>
                    {rollbackingId === v.id ? '↩️ 回退中...' : '↩️ 回退到此版'}
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <div style={{ marginTop: 6, fontSize: 11, color: '#9CA3AF' }}>💡 微调=AI在现有页面上增量修改、保留已插图片；重生=AI按方案从零重画、不保留已插图片；就地编辑=你自己点选文字改内容/字号/颜色/加粗/字体，或点图片替换（不调AI、最快）。页面变形/损坏时用重生补救。微调/重生/就地编辑前系统会自动存一版，可在「📜 历史版本」里「👁 对比当前版」看改了什么、或一键回退。输入框可向下拖高、Shift+回车换行写多条意见；截图除「附截图微调」选文件外，也可在输入框直接 Ctrl+V 粘贴。修改意见会自动暂存，刷新或切走再回来不丢。</div>
      {/* W2: 自带消息条(原Step5无展示位, 微调结果此前不可见) */}
      {message && (
        <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, fontSize: 13, background: message.startsWith('❌') ? '#FEE2E2' : message.startsWith('✅') ? '#D1FAE5' : '#EFF6FF', color: message.startsWith('❌') ? '#DC2626' : message.startsWith('✅') ? '#059669' : '#2563EB' }}>{message}</div>
      )}

      {/* ==================== 版本对比全屏弹窗 ==================== */}
      {compare.open && (
        <VersionCompareModal
          pageNum={pageNum}
          state={compare}
          onSwitchMode={(m) => setCompare(prev => ({ ...prev, mode: m }))}
          onClose={handleCloseCompare}
        />
      )}

      {/* ==================== 就地文字编辑全屏浮层 ==================== */}
      {showInlineEditor && (
        <InlineTextEditor
          coursewareId={coursewareId}
          pageNum={pageNum}
          onPageUpdated={handleInlineEditorSaved}
          onClose={() => setShowInlineEditor(false)}
        />
      )}
    </div>
  )
}

// ==================== 版本对比全屏弹窗子组件 ====================

interface CompareModalProps {
  pageNum: number
  state: CompareState
  onSwitchMode: (m: 'render' | 'code') => void
  onClose: () => void
}

/**
 * 全屏遮罩弹窗：左右并排对比【历史版】与【当前版】。
 *   - render 模式：两个 iframe，各自把 1920×1080 课件等比缩入所在半屏容器（复用 injectPreviewMode 降级注入）。
 *   - code 模式：两个只读 <pre>，并排展示两版 HTML 原文，便于精确核对差异。
 * 纯展示，只读，不触发任何写操作；关闭即销毁两个 iframe。
 */
function VersionCompareModal({ pageNum, state, onSwitchMode, onClose }: CompareModalProps) {
  // 每半屏可用宽度按视口一半减去内边距估算；iframe 缩放比 = 半屏宽 / 画布宽。
  // 用 CSS transform:scale 缩放，容器高度按缩放后画布高设定，保证 16:9 不变形。
  // 注：这里用固定估算比例，视口变化时靠 flex 布局自适应半屏宽，缩放层用 transformOrigin 顶部居中。
  const renderIframe = (html: string, label: string, accent: string) => {
    // 空 HTML（如某版内容缺失）给占位提示
    if (!html || !html.trim()) {
      return (
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#9CA3AF', fontSize: 14, background: '#F9FAFB', borderRadius: 8 }}>
          该版本无可渲染内容
        </div>
      )
    }
    return (
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <div style={{ fontSize: 13, fontWeight: 700, color: accent, marginBottom: 6, textAlign: 'center' }}>{label}</div>
        {/* 缩放容器：外层限定半屏宽高溢出隐藏，内层 iframe 固定 1920×1080 再 scale 缩入 */}
        <div style={{ flex: 1, position: 'relative', overflow: 'hidden', borderRadius: 8, border: `2px solid ${accent}`, background: '#fff' }}>
          <div style={{
            position: 'absolute', top: 0, left: '50%',
            width: CW_WIDTH, height: CW_HEIGHT,
            transform: 'translateX(-50%) scale(var(--cmp-scale))',
            transformOrigin: 'top center',
          }}
            ref={(el) => {
              // 挂载后按父容器实际宽度算缩放比，写入 CSS 变量（避免视口估算不准）
              if (!el) return
              const parent = el.parentElement
              if (!parent) return
              const setScale = () => {
                const availW = parent.clientWidth
                const availH = parent.clientHeight
                const s = Math.min(availW / CW_WIDTH, availH / CW_HEIGHT)
                el.style.setProperty('--cmp-scale', String(s > 0 ? s : 0.1))
              }
              setScale()
              // 视口变化时重算（弹窗生命周期内监听，随 el 卸载自动失效——用一次性 requestAnimationFrame 兜底）
              requestAnimationFrame(setScale)
            }}
          >
            <iframe
              title={label}
              srcDoc={injectPreviewMode(html)}
              sandbox="allow-scripts"
              style={{ width: CW_WIDTH, height: CW_HEIGHT, border: 'none', display: 'block' }}
            />
          </div>
        </div>
      </div>
    )
  }

  const renderCode = (html: string, label: string, accent: string) => (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
      <div style={{ fontSize: 13, fontWeight: 700, color: accent, marginBottom: 6, textAlign: 'center' }}>{label}</div>
      <pre style={{
        flex: 1, margin: 0, padding: '12px 14px', borderRadius: 8, border: `2px solid ${accent}`,
        background: '#1E1E1E', color: '#D4D4D4', fontSize: 12, lineHeight: 1.5,
        overflow: 'auto', whiteSpace: 'pre-wrap', wordBreak: 'break-all', fontFamily: 'Menlo, Consolas, monospace',
      }}>
        {html || '（该版本无内容）'}
      </pre>
    </div>
  )

  return (
    <div
      onClick={onClose}
      style={{
        position: 'fixed', inset: 0, zIndex: 99990,
        background: 'rgba(15,23,42,0.72)',
        display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24,
      }}
    >
      <div
        onClick={e => e.stopPropagation()}
        style={{
          width: '96vw', height: '92vh', background: '#fff', borderRadius: 14,
          display: 'flex', flexDirection: 'column', overflow: 'hidden', boxShadow: '0 20px 60px rgba(0,0,0,0.4)',
        }}
      >
        {/* 弹窗头部：标题 + 视图切换 + 关闭 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '14px 20px', borderBottom: `1px solid ${C.border}`, flexWrap: 'wrap' }}>
          <div style={{ fontSize: 15, fontWeight: 700, color: C.textPrimary }}>
            🆚 第 {pageNum} 页版本对比
            {state.versionNo > 0 && (
              <span style={{ marginLeft: 10, fontSize: 13, fontWeight: 500, color: '#7C3AED' }}>
                历史版：{state.sourceLabel} · v{state.versionNo}
              </span>
            )}
          </div>
          <div style={{ flex: 1 }} />
          {/* 视图切换 */}
          <div style={{ display: 'flex', gap: 6, background: '#F3F4F6', borderRadius: 8, padding: 3 }}>
            <button onClick={() => onSwitchMode('render')}
              style={{ padding: '6px 14px', borderRadius: 6, border: 'none', fontSize: 13, fontWeight: 600, cursor: 'pointer', background: state.mode === 'render' ? '#fff' : 'transparent', color: state.mode === 'render' ? '#7C3AED' : '#6B7280', boxShadow: state.mode === 'render' ? '0 1px 3px rgba(0,0,0,0.12)' : 'none' }}>
              🖼 渲染对比
            </button>
            <button onClick={() => onSwitchMode('code')}
              style={{ padding: '6px 14px', borderRadius: 6, border: 'none', fontSize: 13, fontWeight: 600, cursor: 'pointer', background: state.mode === 'code' ? '#fff' : 'transparent', color: state.mode === 'code' ? '#7C3AED' : '#6B7280', boxShadow: state.mode === 'code' ? '0 1px 3px rgba(0,0,0,0.12)' : 'none' }}>
              {'</> 源码对比'}
            </button>
          </div>
          <button onClick={onClose}
            style={{ padding: '6px 16px', borderRadius: 8, border: `1px solid ${C.border}`, background: '#fff', color: C.textSecondary, fontSize: 14, fontWeight: 600, cursor: 'pointer' }}>
            ✕ 关闭
          </button>
        </div>

        {/* 弹窗主体 */}
        <div style={{ flex: 1, padding: '16px 20px', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          {state.loading ? (
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#6B7280', fontSize: 15 }}>
              ⏳ 正在加载两版内容...
            </div>
          ) : state.error ? (
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#DC2626', fontSize: 14 }}>
              ❌ {state.error}
            </div>
          ) : (
            <div style={{ flex: 1, display: 'flex', gap: 16, minHeight: 0 }}>
              {state.mode === 'render' ? (
                <>
                  {renderIframe(state.historyHtml, `📜 历史版（v${state.versionNo} · ${state.sourceLabel}）`, '#7C3AED')}
                  {renderIframe(state.currentHtml, '✨ 当前版（现在显示的内容）', '#059669')}
                </>
              ) : (
                <>
                  {renderCode(state.historyHtml, `📜 历史版（v${state.versionNo} · ${state.sourceLabel}）`, '#7C3AED')}
                  {renderCode(state.currentHtml, '✨ 当前版（现在显示的内容）', '#059669')}
                </>
              )}
            </div>
          )}
        </div>

        {/* 底部提示 */}
        <div style={{ padding: '10px 20px', borderTop: `1px solid ${C.border}`, fontSize: 12, color: '#9CA3AF', textAlign: 'center' }}>
          左侧为历史版、右侧为当前版。对比为只读，不会修改任何内容；如需换回历史版，请关闭本窗后在版本列表点「↩️ 回退到此版」。
        </div>
      </div>
    </div>
  )
}

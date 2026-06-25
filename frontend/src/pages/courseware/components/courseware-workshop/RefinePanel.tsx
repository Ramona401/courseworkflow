/**
 * RefinePanel.tsx — 页面微调面板（批次W2从主页面抽出，工作台默认Tab）
 *
 * 内容：单页AI微调（支持附截图/Ctrl+V粘贴截图走多模态）+ 单页从零重生 + 页面级历史版本回退。
 * 选中页跟随上方大预览框（批次4b口径，pageNum=父级buildPreviewNum）。
 * W2改进：自带消息条——原Step5没有buildMessage的展示位，微调成功/失败提示此前不可见。
 *
 * 页面级版本与回退：
 *   每次"单页微调/单页重生"前，后端自动把旧 HTML 存为一个版本快照（最多保留近20版）。
 *   本面板提供「📜 历史版本」按钮 → 展开该页版本列表 → 任选一版「回退」（回退前二次确认）。
 *   回退本身可逆：回退时后端会把【当前】内容也另存为一版，故回退后还能再退回。
 *   列表为空表示该页还没产生过版本（首次生成不算"覆盖"，微调/重生后才开始累积）。
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
import { refinePage, regenerateCWPage, listPageVersions, rollbackPage } from '@/api/coursewares'
import type { PageVersionEntry } from '@/api/coursewares'
import { C } from './workshopConstants'

interface Props {
  coursewareId: string
  /** 当前选中页（父级 buildPreviewNum） */
  pageNum: number
  /** 微调/重生/回退成功后回写该页HTML（父级更新 generatedPages 刷新预览） */
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
    setRefineInput(readRefineDraft(coursewareId, pageNum))
    setRefineImage('')  // 切页清掉上一页的截图（截图与页强相关，不跨页保留）
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [coursewareId, pageNum])

  // 单页AI微调(批次4a: 支持随附截图走多模态; 微调=保留页内已插入图片)
  const handleRefinePage = async () => {
    if (!coursewareId || pageNum <= 0 || !refineInput.trim()) return
    setRefineRunning(true)
    try {
      const result = await refinePage(coursewareId, pageNum, refineInput.trim(), refineImage || undefined)
      if (result.html_content) onPageUpdated(pageNum, result.html_content)
      // P1-05: 微调成功 → 清空输入 + 清除该页草稿（已提交的意见无需再留）
      updateRefineInput('')
      setRefineImage('')
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
      {/* 截图粘贴 + 历史版本 + 重生本页 */}
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
        {/* 历史版本按钮：展开/收起当前页版本列表 */}
        <button onClick={handleToggleVersions} disabled={pageNum <= 0}
          title={pageNum <= 0 ? '请先在上方预览区选中页' : '查看本页历次微调/重生前的版本，可一键回退（回退可逆）'}
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
              （仅记录"微调/重生前"的旧版，最多保留近20版；回退可逆）
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

      <div style={{ marginTop: 6, fontSize: 11, color: '#9CA3AF' }}>💡 微调=在现有页面上增量修改、保留已插图片；重生=按方案从零重画、不保留已插图片。页面变形/损坏时用重生补救。微调/重生前系统会自动存一版，可在「📜 历史版本」里一键回退。输入框可向下拖高、Shift+回车换行写多条意见；截图除「附截图微调」选文件外，也可在输入框直接 Ctrl+V 粘贴。修改意见会自动暂存，刷新或切走再回来不丢。</div>
      {/* W2: 自带消息条(原Step5无展示位, 微调结果此前不可见) */}
      {message && (
        <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, fontSize: 13, background: message.startsWith('❌') ? '#FEE2E2' : message.startsWith('✅') ? '#D1FAE5' : '#EFF6FF', color: message.startsWith('❌') ? '#DC2626' : message.startsWith('✅') ? '#059669' : '#2563EB' }}>{message}</div>
      )}
    </div>
  )
}

/**
 * AddPageModal.tsx — 快速添加页面弹窗（批次B升级：双模式）
 *
 * 在 Step5 胶片条点 ＋ 后弹出，一站式完成加页，全程不离开 Step5 工作台。
 *
 * 模式一「🤖 AI生成」（原有流程，保持不变）：
 *   1. 填写页面方案（标题/目的/概要/丰富度）
 *   2. 调 addCWPage 创建页面
 *   3. 自动调 regenerateCWPage 触发 HTML 生成
 *   4. 生成完成后通知父级刷新，自动选中新页面
 *
 * 模式二「📋 粘贴HTML」（批次B新增）：
 *   看到别人做得好的课件页代码，整页粘贴进来直接成为新页：
 *   1. 填标题 + 粘贴完整HTML代码（提交前做 <div> 开闭标签数量自检，疑似残缺时二次确认）
 *   2. 调 addCWPage 创建页面（方案字段给通用默认值，概要注明"由粘贴HTML创建"）
 *   3. 调 importPageHtml 把粘贴内容导入该页——后端做画布契约归一（1920×1080）、
 *      导航栏替换重编号（仅当代码自带 NAV 标记，即从平台其它课件复制来的页）、
 *      背景幂等补注、覆盖前版本快照
 *   4. 完成后通知父级刷新，自动选中新页面
 *   注：外部来源HTML没有本平台导航栏属预期；后端会强制归一根容器为 1920×1080 画布，
 *       非该比例的外来页面导入后观感可能与原始来源不同（可再用微调/源码编辑修正）。
 */
import { useState } from 'react'
import { addCWPage, regenerateCWPage, importPageHtml } from '@/api/coursewares'
import { C } from './workshopConstants'

// ==================== 内容丰富度三档（与 IndexEditor 保持一致） ====================
const RICHNESS_OPTIONS = [
  { value: 2, emoji: '🌱', label: '精简', desc: '要点为主，简洁留白', color: '#059669', bg: '#D1FAE5' },
  { value: 3, emoji: '📖', label: '适中', desc: '标准图文讲解', color: '#0891B2', bg: '#CFFAFE' },
  { value: 5, emoji: '🎯', label: '充实', desc: '详尽展开，多举例', color: '#DC2626', bg: '#FEE2E2' },
]

/**
 * 批次B·轻量结构自检：比对 <div 开标签与 </div> 闭标签数量（与 PagePreviewBlock 源码编辑同口径）。
 * 只抓"最常见的粘贴残缺/漏尾"问题；数量不一致返回人话提示文案，一致返回空串。
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
  coursewareId: string
  /** 当前已有页数（用于默认标题和插入提示） */
  currentPageCount: number
  /** 操作完成后回调（刷新页面列表 + 选中新页面） */
  onDone: (newPageNumber: number) => void
  /** 关闭弹窗 */
  onClose: () => void
}

export default function AddPageModal({ coursewareId, currentPageCount, onDone, onClose }: Props) {
  // 模式：ai=AI生成（原有流程） / paste=粘贴HTML（批次B新增）
  const [mode, setMode] = useState<'ai' | 'paste'>('ai')

  // 表单字段（title 两模式共用）
  const [title, setTitle] = useState(`第 ${currentPageCount + 1} 页`)
  const [purpose, setPurpose] = useState('')
  const [contentSummary, setContentSummary] = useState('')
  const [richness, setRichness] = useState(3) // 默认适中
  const [pasteHtml, setPasteHtml] = useState('') // 批次B：粘贴的HTML代码

  // 操作状态
  const [phase, setPhase] = useState<'form' | 'creating' | 'generating' | 'done' | 'error'>('form')
  const [errorMsg, setErrorMsg] = useState('')
  const [newPageNum, setNewPageNum] = useState(0)

  /** 模式切换：清掉错误提示（各模式表单内容各自保留，来回切不丢） */
  const switchMode = (m: 'ai' | 'paste') => {
    setMode(m)
    setErrorMsg('')
  }

  /** 提交（AI生成模式）：创建页面 → 自动生成 HTML */
  const handleSubmit = async () => {
    if (!title.trim()) {
      setErrorMsg('请填写页面标题')
      return
    }

    setPhase('creating')
    setErrorMsg('')

    try {
      // 第1步：创建页面（方案数据）
      const newPage = await addCWPage(coursewareId, {
        title: title.trim(),
        purpose: purpose.trim() || undefined,
        content_summary: contentSummary.trim() || undefined,
        interaction_type: 'static',
        visual_format: 'text_heavy',
        estimated_complexity: richness,
      })

      const pageNum = newPage.page_number
      setNewPageNum(pageNum)
      setPhase('generating')

      // 第2步：自动触发 HTML 生成
      try {
        await regenerateCWPage(coursewareId, pageNum)
        setPhase('done')
        // 短暂展示成功后自动关闭
        setTimeout(() => onDone(pageNum), 800)
      } catch {
        // 生成失败但页面已创建成功——仍然通知父级刷新，老师可以手动重生成
        setPhase('done')
        setErrorMsg('页面已创建，但 HTML 生成失败，可稍后在微调面板手动重新生成')
        setTimeout(() => onDone(pageNum), 2000)
      }
    } catch (err: unknown) {
      setPhase('error')
      const msg = err instanceof Error ? err.message : '创建失败'
      setErrorMsg(msg)
    }
  }

  /** 提交（粘贴HTML模式，批次B）：结构自检 → 创建页面 → 导入粘贴的HTML */
  const handleSubmitPaste = async () => {
    if (!title.trim()) {
      setErrorMsg('请填写页面标题')
      return
    }
    if (!pasteHtml.trim()) {
      setErrorMsg('请粘贴页面HTML代码')
      return
    }
    // 轻量结构自检：疑似残缺时二次确认（不硬拦——外部代码写法多样，老师可自行决定）
    const warn = divBalanceCheck(pasteHtml)
    if (warn && !window.confirm(
      '⚠️ 结构自检提示：' + warn + '。\n\n仍要导入吗？\n（导入后如显示异常，可在源码编辑里修改，或删除该页重新粘贴）',
    )) return

    setPhase('creating')
    setErrorMsg('')

    try {
      // 第1步：创建页面（方案字段给通用默认值，概要注明来源便于日后识别）
      const newPage = await addCWPage(coursewareId, {
        title: title.trim(),
        content_summary: '（由粘贴的HTML代码创建）',
        interaction_type: 'static',
        visual_format: 'text_heavy',
        estimated_complexity: 3,
      })

      const pageNum = newPage.page_number
      setNewPageNum(pageNum)
      setPhase('generating')

      // 第2步：导入粘贴的HTML（后端做画布归一/导航重编号/背景补注/快照，不调AI，秒级完成）
      try {
        await importPageHtml(coursewareId, pageNum, pasteHtml)
        setPhase('done')
        setTimeout(() => onDone(pageNum), 800)
      } catch (err: unknown) {
        // 导入失败但页面已创建成功——仍然通知父级刷新，老师可在源码编辑里重试粘贴
        setPhase('done')
        setErrorMsg('页面已创建，但HTML导入失败：'
          + (err instanceof Error ? err.message : '未知错误')
          + '。可选中该页用「✏️ 编辑源码」重试粘贴')
        setTimeout(() => onDone(pageNum), 2500)
      }
    } catch (err: unknown) {
      setPhase('error')
      const msg = err instanceof Error ? err.message : '创建失败'
      setErrorMsg(msg)
    }
  }

  // ==================== 样式常量 ====================
  const overlay: React.CSSProperties = {
    position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
    background: 'rgba(0,0,0,0.4)', zIndex: 99990,
    display: 'flex', alignItems: 'center', justifyContent: 'center',
  }
  const panel: React.CSSProperties = {
    background: '#fff', borderRadius: 16, padding: '28px 32px',
    width: '100%', maxWidth: 520, maxHeight: '90vh', overflowY: 'auto',
    boxShadow: '0 20px 60px rgba(0,0,0,0.15)',
  }
  const labelStyle: React.CSSProperties = {
    display: 'block', fontSize: 13, color: '#6B7280', marginBottom: 4, fontWeight: 500,
  }
  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '8px 12px', borderRadius: 8,
    border: '1px solid #E5E7EB', fontSize: 14, outline: 'none',
    boxSizing: 'border-box',
  }
  const textareaStyle: React.CSSProperties = {
    ...inputStyle, resize: 'vertical' as const, minHeight: 60,
  }

  const isBusy = phase === 'creating' || phase === 'generating'

  return (
    <div style={overlay} onClick={isBusy ? undefined : onClose}>
      <div style={panel} onClick={e => e.stopPropagation()}>
        {/* 标题 */}
        <h3 style={{ margin: '0 0 16px', fontSize: 18, fontWeight: 600, color: '#1F2937' }}>
          ＋ 快速添加页面
        </h3>

        {/* 表单 */}
        {(phase === 'form' || phase === 'error') && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {/* 批次B：模式切换页签 */}
            <div style={{ display: 'flex', gap: 6, background: '#F3F4F6', borderRadius: 10, padding: 4 }}>
              <button
                type="button"
                onClick={() => switchMode('ai')}
                style={{
                  flex: 1, padding: '8px 10px', borderRadius: 8, border: 'none',
                  fontSize: 13, fontWeight: 600, cursor: 'pointer',
                  background: mode === 'ai' ? '#fff' : 'transparent',
                  color: mode === 'ai' ? C.primary : '#6B7280',
                  boxShadow: mode === 'ai' ? '0 1px 3px rgba(0,0,0,0.12)' : 'none',
                  transition: 'all 0.15s',
                }}
              >🤖 AI生成</button>
              <button
                type="button"
                onClick={() => switchMode('paste')}
                style={{
                  flex: 1, padding: '8px 10px', borderRadius: 8, border: 'none',
                  fontSize: 13, fontWeight: 600, cursor: 'pointer',
                  background: mode === 'paste' ? '#fff' : 'transparent',
                  color: mode === 'paste' ? '#059669' : '#6B7280',
                  boxShadow: mode === 'paste' ? '0 1px 3px rgba(0,0,0,0.12)' : 'none',
                  transition: 'all 0.15s',
                }}
              >📋 粘贴HTML</button>
            </div>

            {/* 标题（两模式共用） */}
            <div>
              <label style={labelStyle}>页面标题 *</label>
              <input
                value={title}
                onChange={e => setTitle(e.target.value)}
                placeholder="例如：实验操作步骤"
                style={inputStyle}
                autoFocus
              />
            </div>

            {/* ========== 模式一：AI生成（原有表单） ========== */}
            {mode === 'ai' && <>
              {/* 教学目的 */}
              <div>
                <label style={labelStyle}>
                  教学目的
                  <span style={{ color: '#9CA3AF', fontSize: 12, marginLeft: 4 }}>（可选，告诉 AI 这一页要达成什么）</span>
                </label>
                <input
                  value={purpose}
                  onChange={e => setPurpose(e.target.value)}
                  placeholder="例如：学生通过观察实验现象，理解光合作用原理"
                  style={inputStyle}
                />
              </div>

              {/* 内容概要 */}
              <div>
                <label style={labelStyle}>
                  内容概要
                  <span style={{ color: '#9CA3AF', fontSize: 12, marginLeft: 4 }}>（写得越详细，AI 越会逐点展开这一页）</span>
                </label>
                <textarea
                  value={contentSummary}
                  onChange={e => setContentSummary(e.target.value)}
                  placeholder="例如：1. 实验材料准备 2. 操作步骤（先...再...然后...） 3. 注意事项"
                  rows={3}
                  style={textareaStyle}
                />
              </div>

              {/* 内容丰富度 */}
              <div>
                <label style={labelStyle}>内容丰富度</label>
                <div style={{ display: 'flex', gap: 8 }}>
                  {RICHNESS_OPTIONS.map(opt => {
                    const selected = richness === opt.value
                    return (
                      <button
                        key={opt.value}
                        type="button"
                        onClick={() => setRichness(opt.value)}
                        style={{
                          flex: 1, padding: '10px 8px', borderRadius: 10, cursor: 'pointer',
                          border: `2px solid ${selected ? opt.color : '#E5E7EB'}`,
                          background: selected ? opt.bg : '#fff',
                          display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2,
                          transition: 'all 0.15s',
                        }}
                      >
                        <span style={{ fontSize: 18 }}>{opt.emoji}</span>
                        <span style={{ fontSize: 13, fontWeight: 700, color: selected ? opt.color : '#1F2937' }}>
                          {opt.label}
                        </span>
                        <span style={{ fontSize: 11, color: selected ? opt.color : '#9CA3AF', textAlign: 'center' }}>
                          {opt.desc}
                        </span>
                      </button>
                    )
                  })}
                </div>
              </div>
            </>}

            {/* ========== 模式二：粘贴HTML（批次B新增） ========== */}
            {mode === 'paste' && <>
              <div>
                <label style={labelStyle}>
                  页面HTML代码 *
                  <span style={{ color: '#9CA3AF', fontSize: 12, marginLeft: 4 }}>（整页粘贴，从最外层 &lt;div 到结尾）</span>
                </label>
                <textarea
                  value={pasteHtml}
                  onChange={e => setPasteHtml(e.target.value)}
                  placeholder={'把别人的完整页面HTML代码粘贴到这里…\n（例如从共享课件库「复制源码」得到的代码，或外部制作的 1920×1080 单页HTML）'}
                  rows={9}
                  spellCheck={false}
                  style={{
                    ...inputStyle, resize: 'vertical' as const, minHeight: 180,
                    fontFamily: 'Monaco, Consolas, "Courier New", monospace', fontSize: 12, lineHeight: 1.6,
                    background: '#1e1e1e', color: '#d4d4d4', border: '1px solid #374151',
                  }}
                />
              </div>
              <div style={{ padding: '10px 12px', borderRadius: 8, background: '#F0FDF4', color: '#166534', fontSize: 12, lineHeight: 1.7 }}>
                💡 导入时系统自动处理：① 统一为 1920×1080 画布（外来页面非该比例时观感可能变化）；
                ② 平台内复制来的页（自带导航标记）会换成<b>本课件</b>的导航栏并改对页码；
                ③ 补注本课件当前背景，与整套课件视觉统一。外部HTML没有导航栏属正常现象。
              </div>
            </>}

            {/* 错误提示 */}
            {errorMsg && (
              <div style={{ padding: '10px 14px', borderRadius: 8, background: '#FEE2E2', color: '#DC2626', fontSize: 13 }}>
                {errorMsg}
              </div>
            )}

            {/* 操作按钮 */}
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 6 }}>
              <button onClick={onClose} style={{
                padding: '8px 20px', borderRadius: 8, border: '1px solid #E5E7EB',
                background: 'transparent', color: '#6B7280', fontSize: 14, cursor: 'pointer',
              }}>取消</button>
              <button onClick={mode === 'ai' ? handleSubmit : handleSubmitPaste} style={{
                padding: '8px 24px', borderRadius: 8, border: 'none',
                background: mode === 'ai' ? C.primary : '#059669', color: '#fff', fontSize: 14, fontWeight: 600, cursor: 'pointer',
              }}>{mode === 'ai' ? '添加并生成' : '创建并导入'}</button>
            </div>
          </div>
        )}

        {/* 创建中 */}
        {phase === 'creating' && (
          <div style={{ textAlign: 'center', padding: '30px 0' }}>
            <div style={{ fontSize: 32, marginBottom: 12 }}>📝</div>
            <div style={{ fontSize: 15, color: '#4B5563' }}>正在创建页面...</div>
          </div>
        )}

        {/* 生成中/导入中 */}
        {phase === 'generating' && (
          <div style={{ textAlign: 'center', padding: '30px 0' }}>
            <div style={{ fontSize: 32, marginBottom: 12, animation: 'spin 2s linear infinite' }}>⚙️</div>
            <div style={{ fontSize: 15, color: '#4B5563', marginBottom: 6 }}>
              {mode === 'paste'
                ? `页面已创建（P${newPageNum}），正在导入HTML...`
                : `页面已创建（P${newPageNum}），正在 AI 生成 HTML...`}
            </div>
            <div style={{ fontSize: 13, color: '#9CA3AF' }}>
              {mode === 'paste' ? '导入不调AI，通常1-3秒完成' : '通常需要 15-40 秒，请耐心等待'}
            </div>
            {/* 旋转动画 */}
            <style>{`@keyframes spin { from { transform: rotate(0deg) } to { transform: rotate(360deg) } }`}</style>
          </div>
        )}

        {/* 完成 */}
        {phase === 'done' && (
          <div style={{ textAlign: 'center', padding: '30px 0' }}>
            <div style={{ fontSize: 32, marginBottom: 12 }}>✅</div>
            <div style={{ fontSize: 15, color: '#059669', marginBottom: 6 }}>
              {mode === 'paste' ? `P${newPageNum} 已导入完成！` : `P${newPageNum} 已生成完成！`}
            </div>
            {errorMsg && (
              <div style={{ fontSize: 13, color: '#D97706', marginTop: 8 }}>{errorMsg}</div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

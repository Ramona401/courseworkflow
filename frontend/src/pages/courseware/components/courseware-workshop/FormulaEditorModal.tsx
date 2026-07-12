/**
 * FormulaEditorModal.tsx — 课件公式编辑器弹窗
 *
 * 复用笔顺编辑器(StrokeOrderModal)的交互范式：
 * 老师输入/选模板 LaTeX 公式 → 实时预览渲染效果 → 调参(字号/颜色/显示模式/标注)
 * → 点"融入当前页" → 拼微调指令经 onInsert 回调交给父组件调 RefinePage。
 *
 * 技术方案：
 *   - 渲染预览用 KaTeX（CDN 动态加载，renderToString 输出到预览区）
 *   - 公式输入：手动输入 LaTeX + 常用模板点选（MathLive 所见即所得留迭代二）
 *   - 输出：自包含 HTML 代码片段（KaTeX CDN + 渲染脚本，离线可用）
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/FormulaEditorModal.tsx
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import {
  FORMULA_DEFAULTS,
  FORMULA_SIZE_PRESETS,
  FORMULA_COLOR_PRESETS,
  FORMULA_TEMPLATES,
  KATEX_CSS_CDN,
  KATEX_JS_CDN,
  buildFormulaRefineInstruction,
  type FormulaConfig,
} from './formulaEditorUtils'
import { C } from './workshopConstants'

// ==================== 类型 ====================

interface Props {
  /** 当前课件页号（显示用） */
  pageNum: number
  /** 融入回调：父组件收到微调指令后调 RefinePage */
  onInsert: (instruction: string) => void
  /** 关闭弹窗 */
  onClose: () => void
  /** 是否正在执行微调（禁用融入按钮） */
  inserting?: boolean
}

// ==================== KaTeX 全局类型 ====================

declare const katex: {
  renderToString: (latex: string, opts: { displayMode: boolean; throwOnError: boolean }) => string
}

// ==================== 组件 ====================

export default function FormulaEditorModal({ pageNum, onInsert, onClose, inserting }: Props) {
  // ---- 公式状态 ----
  const [latex, setLatex] = useState('')
  const [displayMode, setDisplayMode] = useState<'inline' | 'display'>('display')
  const [fontSize, setFontSize] = useState(FORMULA_DEFAULTS.fontSize)
  const [color, setColor] = useState(FORMULA_DEFAULTS.color)
  const [caption, setCaption] = useState('')
  const [layout, setLayout] = useState<'single' | 'side-by-side'>('single')
  const [positionHint, setPositionHint] = useState('')

  // ---- 多公式列表（可累积多个公式一起融入） ----
  const [formulaList, setFormulaList] = useState<FormulaConfig[]>([])

  // ---- KaTeX 加载与预览状态 ----
  const [katexLoaded, setKatexLoaded] = useState(false)
  const [previewHtml, setPreviewHtml] = useState('')
  const [previewError, setPreviewError] = useState('')
  const [toast, setToast] = useState('')

  // ---- 模板展开/收起 ----
  const [expandedGroup, setExpandedGroup] = useState<string | null>(null)

  const previewRef = useRef<HTMLDivElement>(null)

  // ---- 加载 KaTeX CDN ----
  useEffect(() => {
    // 先加载 CSS
    if (!document.querySelector('link[href*="katex.min.css"]')) {
      const link = document.createElement('link')
      link.rel = 'stylesheet'
      link.href = KATEX_CSS_CDN
      document.head.appendChild(link)
    }
    // 再加载 JS
    if (typeof (window as unknown as Record<string, unknown>).katex !== 'undefined') {
      setKatexLoaded(true)
      return
    }
    const existing = document.querySelector('script[src*="katex.min.js"]')
    if (existing) {
      const check = setInterval(() => {
        if (typeof (window as unknown as Record<string, unknown>).katex !== 'undefined') {
          setKatexLoaded(true)
          clearInterval(check)
        }
      }, 200)
      return () => clearInterval(check)
    }
    const s = document.createElement('script')
    s.src = KATEX_JS_CDN
    s.onload = () => setKatexLoaded(true)
    document.head.appendChild(s)
  }, [])

  // ---- 实时预览 ----
  const updatePreview = useCallback(() => {
    if (!katexLoaded || !latex.trim()) {
      setPreviewHtml('')
      setPreviewError('')
      return
    }
    try {
      const html = katex.renderToString(latex, {
        displayMode: displayMode === 'display',
        throwOnError: false,
      })
      setPreviewHtml(html)
      setPreviewError('')
    } catch (e) {
      setPreviewHtml('')
      setPreviewError(e instanceof Error ? e.message : '渲染失败')
    }
  }, [katexLoaded, latex, displayMode])

  useEffect(() => {
    const timer = setTimeout(updatePreview, 300)
    return () => clearTimeout(timer)
  }, [updatePreview])

  // ---- 模板点选 ----
  const handleTemplateClick = (tplLatex: string) => {
    setLatex(tplLatex)
  }

  // ---- 添加到列表 ----
  const handleAddToList = () => {
    if (!latex.trim()) { showToast('请先输入公式'); return }
    setFormulaList(prev => [...prev, { latex, displayMode, fontSize, color, caption: caption.trim() || undefined }])
    showToast('✅ 已添加到列表（共 ' + (formulaList.length + 1) + ' 个）')
    setLatex('')
    setCaption('')
  }

  // ---- 从列表移除 ----
  const handleRemoveFromList = (idx: number) => {
    setFormulaList(prev => prev.filter((_, i) => i !== idx))
  }

  // ---- 融入当前页 ----
  const handleInsert = () => {
    // 如果列表为空但当前输入框有公式，直接用当前这一个
    const allFormulas = formulaList.length > 0
      ? formulaList
      : latex.trim()
        ? [{ latex, displayMode, fontSize, color, caption: caption.trim() || undefined }]
        : []

    if (allFormulas.length === 0) { showToast('请先输入或添加公式'); return }

    const instruction = buildFormulaRefineInstruction({
      formulas: allFormulas,
      positionHint: positionHint.trim() || undefined,
      layout,
    })
    onInsert(instruction)
  }

  // ---- Toast ----
  const showToast = (msg: string) => {
    setToast(msg)
    setTimeout(() => setToast(''), 2500)
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 99995,
      background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}>
      <div style={{
        width: '92vw', maxWidth: 1040, maxHeight: '92vh', overflow: 'auto',
        background: '#fff', borderRadius: 16, boxShadow: '0 20px 60px rgba(0,0,0,0.3)',
        padding: 28, position: 'relative',
      }}>
        {/* 顶栏 */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
          <div>
            <div style={{ fontSize: 18, fontWeight: 700, color: C.textPrimary }}>📐 公式编辑器</div>
            <div style={{ fontSize: 13, color: C.textMuted, marginTop: 2 }}>编辑数学/物理/化学公式，融入第 {pageNum} 页课件</div>
          </div>
          <button onClick={onClose} style={{
            width: 36, height: 36, borderRadius: '50%', border: '1px solid ' + C.border,
            background: '#fff', fontSize: 18, cursor: 'pointer', display: 'flex',
            alignItems: 'center', justifyContent: 'center', color: C.textMuted,
          }}>✕</button>
        </div>

        {/* 主体：左编辑 + 右预览/参数 */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 340px', gap: 24 }}>
          {/* 左栏：输入 + 模板 */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {/* LaTeX 输入框 */}
            <div>
              <label style={labelStyle}>LaTeX 公式（手动输入或从下方模板点选）</label>
              <textarea value={latex} onChange={e => setLatex(e.target.value)}
                placeholder="例如: x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}"
                rows={3}
                style={{ width: '100%', padding: '10px 14px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 14, fontFamily: 'Menlo, Consolas, monospace', outline: 'none', resize: 'vertical', boxSizing: 'border-box', lineHeight: 1.6 }} />
            </div>

            {/* 标注文字 */}
            <div>
              <label style={labelStyle}>标注文字（可选，显示在公式下方）</label>
              <input type="text" value={caption} onChange={e => setCaption(e.target.value)}
                placeholder="如：勾股定理、牛顿第二定律"
                style={{ width: '100%', padding: '8px 12px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 13, outline: 'none', boxSizing: 'border-box' }} />
            </div>

            {/* 添加到列表按钮 */}
            <div style={{ display: 'flex', gap: 8 }}>
              <button onClick={handleAddToList} disabled={!latex.trim()}
                style={{ flex: 1, padding: '8px 16px', borderRadius: 8, border: '1px solid ' + (latex.trim() ? '#2563EB' : C.border), background: latex.trim() ? '#EFF6FF' : '#fff', color: latex.trim() ? '#2563EB' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: latex.trim() ? 'pointer' : 'default' }}>
                ＋ 添加到列表{formulaList.length > 0 ? '（已有 ' + formulaList.length + ' 个）' : ''}
              </button>
            </div>

            {/* 已添加的公式列表 */}
            {formulaList.length > 0 && (
              <div style={{ padding: '10px 12px', borderRadius: 8, border: '1px solid #BFDBFE', background: '#EFF6FF' }}>
                <div style={{ fontSize: 12, fontWeight: 600, color: '#1E40AF', marginBottom: 6 }}>📋 待融入公式列表</div>
                {formulaList.map((f, i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0', borderTop: i > 0 ? '1px solid #DBEAFE' : 'none' }}>
                    <span style={{ flex: 1, fontSize: 12, color: '#374151', fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {f.caption ? f.caption + ': ' : ''}{f.latex}
                    </span>
                    <button onClick={() => handleRemoveFromList(i)}
                      style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid #EF4444', background: 'transparent', color: '#EF4444', fontSize: 11, cursor: 'pointer' }}>✕</button>
                  </div>
                ))}
              </div>
            )}

            {/* 常用公式模板 */}
            <div style={{ background: '#FAFAFA', borderRadius: 10, padding: '12px 14px', border: '1px solid ' + C.border }}>
              <div style={{ fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 8 }}>📚 常用公式模板（点选即填入上方输入框）</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6, maxHeight: 300, overflowY: 'auto' }}>
                {FORMULA_TEMPLATES.map(group => (
                  <div key={group.group}>
                    <button onClick={() => setExpandedGroup(expandedGroup === group.group ? null : group.group)}
                      style={{ width: '100%', textAlign: 'left', padding: '6px 10px', borderRadius: 6, border: '1px solid ' + C.border, background: expandedGroup === group.group ? '#EFF6FF' : '#fff', color: C.textPrimary, fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>
                      {group.group} {expandedGroup === group.group ? '▲' : '▼'}
                    </button>
                    {expandedGroup === group.group && (
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, padding: '8px 4px' }}>
                        {group.items.map(item => (
                          <button key={item.label} onClick={() => handleTemplateClick(item.latex)}
                            title={item.latex}
                            style={{ padding: '4px 10px', borderRadius: 6, border: '1px solid ' + C.border, background: '#fff', color: '#374151', fontSize: 12, cursor: 'pointer', whiteSpace: 'nowrap' }}>
                            {item.label}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* 右栏：预览 + 参数 */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {/* 实时预览 */}
            <div style={{
              background: '#FAFAFA', border: '1px solid ' + C.border, borderRadius: 12,
              padding: 20, minHeight: 120, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
            }}>
              <div style={{ fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 8, alignSelf: 'flex-start' }}>实时预览</div>
              {!katexLoaded ? (
                <div style={{ color: C.textMuted, fontSize: 13 }}>⏳ 加载 KaTeX 中...</div>
              ) : !latex.trim() ? (
                <div style={{ color: C.textMuted, fontSize: 13 }}>输入公式后这里显示预览</div>
              ) : previewError ? (
                <div style={{ color: '#DC2626', fontSize: 13 }}>❌ {previewError}</div>
              ) : (
                <div ref={previewRef}
                  style={{ fontSize: fontSize, color: color, lineHeight: 1.6, textAlign: displayMode === 'display' ? 'center' : 'left', width: '100%' }}
                  dangerouslySetInnerHTML={{ __html: previewHtml }} />
              )}
            </div>

            {/* 显示模式 */}
            <ParamSection title="显示模式">
              <div style={{ display: 'flex', gap: 8 }}>
                {(['display', 'inline'] as const).map(m => (
                  <button key={m} onClick={() => setDisplayMode(m)} style={{
                    flex: 1, padding: '8px 6px', borderRadius: 8, fontSize: 13, cursor: 'pointer',
                    border: '1.5px solid ' + (displayMode === m ? '#1E40AF' : C.border),
                    background: displayMode === m ? '#EFF6FF' : '#fff',
                    color: displayMode === m ? '#1E40AF' : C.textSecondary,
                    fontWeight: displayMode === m ? 600 : 400,
                  }}>{m === 'display' ? '🖥 独立居中' : '📝 行内公式'}</button>
                ))}
              </div>
            </ParamSection>

            {/* 字号 */}
            <ParamSection title="公式字号">
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {FORMULA_SIZE_PRESETS.map(p => (
                  <button key={p.size} onClick={() => setFontSize(p.size)} style={{
                    padding: '5px 10px', borderRadius: 6, fontSize: 12, cursor: 'pointer',
                    border: '1px solid ' + (fontSize === p.size ? '#1E40AF' : C.border),
                    background: fontSize === p.size ? '#EFF6FF' : '#fff',
                    color: fontSize === p.size ? '#1E40AF' : C.textSecondary,
                    fontWeight: fontSize === p.size ? 600 : 400,
                  }}>{p.label}</button>
                ))}
              </div>
            </ParamSection>

            {/* 颜色 */}
            <ParamSection title="公式颜色">
              <div style={{ display: 'flex', gap: 8 }}>
                {FORMULA_COLOR_PRESETS.map(p => (
                  <div key={p.color} onClick={() => setColor(p.color)} title={p.label}
                    style={{
                      width: 32, height: 32, borderRadius: 8, cursor: 'pointer',
                      background: p.color, transition: 'all 0.15s',
                      border: color === p.color ? '3px solid #1E40AF' : '1px solid ' + C.border,
                      transform: color === p.color ? 'scale(1.15)' : 'scale(1)',
                    }} />
                ))}
              </div>
            </ParamSection>

            {/* 多公式布局 */}
            {formulaList.length > 1 && (
              <ParamSection title={'多公式布局（共 ' + formulaList.length + ' 个）'}>
                <div style={{ display: 'flex', gap: 8 }}>
                  {([['single', '↕️ 纵向'], ['side-by-side', '↔️ 并排']] as const).map(([k, label]) => (
                    <button key={k} onClick={() => setLayout(k as 'single' | 'side-by-side')} style={{
                      flex: 1, padding: '8px 6px', borderRadius: 8, fontSize: 13, cursor: 'pointer',
                      border: '1.5px solid ' + (layout === k ? '#1E40AF' : C.border),
                      background: layout === k ? '#EFF6FF' : '#fff',
                      color: layout === k ? '#1E40AF' : C.textSecondary,
                      fontWeight: layout === k ? 600 : 400,
                    }}>{label}</button>
                  ))}
                </div>
              </ParamSection>
            )}

            {/* 位置偏好 */}
            <ParamSection title="融入位置偏好（可选）">
              <input type="text" value={positionHint} onChange={e => setPositionHint(e.target.value)}
                placeholder="如：放在定理文字下方、替换现有公式区域"
                style={{ width: '100%', padding: '8px 12px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 13, outline: 'none', boxSizing: 'border-box' }} />
            </ParamSection>

            {/* 融入按钮 */}
            <button onClick={handleInsert}
              disabled={(formulaList.length === 0 && !latex.trim()) || inserting}
              style={{
                marginTop: 4, padding: '14px 24px', borderRadius: 10, border: 'none',
                background: (formulaList.length > 0 || latex.trim()) && !inserting
                  ? 'linear-gradient(135deg, #1E40AF, #2563EB)' : '#E5E7EB',
                color: (formulaList.length > 0 || latex.trim()) && !inserting ? '#fff' : '#9CA3AF',
                fontSize: 15, fontWeight: 700, cursor: (formulaList.length > 0 || latex.trim()) && !inserting ? 'pointer' : 'default',
                width: '100%', boxShadow: (formulaList.length > 0 || latex.trim()) && !inserting ? '0 4px 16px rgba(30,64,175,0.3)' : 'none',
              }}
            >
              {inserting ? '⏳ AI 正在融入当前页...' : '📐 融入第 ' + pageNum + ' 页课件'}
            </button>
            <div style={{ fontSize: 12, color: C.textMuted, lineHeight: 1.5, textAlign: 'center' }}>
              {formulaList.length > 0
                ? '将把列表中 ' + formulaList.length + ' 个公式一起融入'
                : '点击后 AI 会把公式自然融入当前页面的布局中'}
            </div>
          </div>
        </div>

        {/* Toast */}
        {toast && (
          <div style={{
            position: 'fixed', bottom: 30, left: '50%', transform: 'translateX(-50%)',
            background: '#1F2937', color: '#fff', padding: '8px 24px', borderRadius: 8,
            fontSize: 13, zIndex: 99999, boxShadow: '0 4px 12px rgba(0,0,0,0.2)',
          }}>{toast}</div>
        )}
      </div>
    </div>
  )
}

// ==================== 子组件 ====================

function ParamSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ background: '#FAFAFA', borderRadius: 10, padding: '12px 16px', border: '1px solid ' + C.border }}>
      <div style={{ fontSize: 12, fontWeight: 600, color: C.textSecondary, marginBottom: 8 }}>{title}</div>
      {children}
    </div>
  )
}

const labelStyle: React.CSSProperties = { fontSize: 12, fontWeight: 600, color: '#6B7280', display: 'block', marginBottom: 4 }

/**
 * FontPicker.tsx — 课件字体方案选择器（字体F2新建）
 *
 * 嵌入位置：课件工坊 Step3「确认导航栏」页 + Step5「确认提交」页，紧跟 BackgroundPicker 之后。
 * 交互（完全对齐 BackgroundPicker 的交互语言）：
 *   - 卡片画廊展示5套系统预设方案 + 「跟随模板」（=清除选择）。
 *   - 每张卡片用真实 woff2 渲染字样预览：标题行用标题字体、正文行用正文字体，
 *     英语方案可直接看到 Poppins 单层 a 的 "apple" 真实字形。
 *   - 点选即秒换全部已生成页（零token零等待），后续生成/重生/微调的页面自动带该字体。
 *   - disabled=true（批量生成进行中）时整体禁用，防在飞页落库带新旧混杂字体的竞态。
 * 选中判定：后端直接存方案code，以 font_scheme === scheme.code 判定（比背景的URL快照判定更直接）。
 * 字体加载：组件挂载后把全部 @font-face 注入一次 document.head（模块级标志防重复），
 *   font-display:swap 保证未下载完时先用系统字体占位不阻塞；woff2 走 Nginx 30天缓存。
 */
import { useState, useEffect } from 'react'
import { listFontSchemes, getCWFont, setCWFont, clearCWFont } from '@/api/coursewares'
import type { CWFontScheme } from '@/api/coursewares'
import { C } from './workshopConstants'

interface Props {
  coursewareId: string
  /** 选/清字体成功后回调（父组件重拉课件与页面，刷新预览） */
  onSwapped: () => void
  /** 批量生成进行中传true，整体禁用防竞态 */
  disabled?: boolean
}

// 系统级中文兜底字体栈（与后端注入CSS同口径，字体未到货时的过渡显示）
const CJK_FALLBACK = ",'PingFang SC','Microsoft YaHei',sans-serif"

// 模块级标志：@font-face 预览样式只注入 document.head 一次（跨组件实例/跨页面切换不重复）
let fontFacesInjected = false

/** 把全部方案的 @font-face 去重后注入 document.head，供卡片预览用真实字体渲染 */
const injectFontFaces = (baseUrl: string, schemes: CWFontScheme[]) => {
  if (fontFacesInjected || typeof document === 'undefined' || !baseUrl) return
  const seen = new Set<string>()
  let css = ''
  schemes.forEach(s => (s.faces || []).forEach(f => {
    const key = f.family + '|' + (f.weight || '400')
    if (seen.has(key)) return
    seen.add(key)
    css += "@font-face{font-family:'" + f.family + "';src:url('" + baseUrl + f.file +
      "') format('woff2');font-weight:" + (f.weight || '400') + ";font-style:normal;font-display:swap}"
  }))
  if (!css) return
  const el = document.createElement('style')
  el.id = 'tedna-cw-font-preview-faces'
  el.textContent = css
  document.head.appendChild(el)
  fontFacesInjected = true
}

export default function FontPicker({ coursewareId, onSwapped, disabled = false }: Props) {
  const [schemes, setSchemes] = useState<CWFontScheme[]>([])
  const [current, setCurrent] = useState('')     // 当前生效的方案code（''=未选，跟随模板）
  const [loading, setLoading] = useState(true)
  const [applying, setApplying] = useState('')   // 正在应用的方案code（'__clear__'=清除中, ''=空闲）
  const [message, setMessage] = useState('')

  const busy = !!applying

  // 进入即拉取方案列表 + 课件当前选择（防卸载后 setState 用 cancelled 标志）
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    Promise.all([listFontSchemes(), getCWFont(coursewareId)])
      .then(([ls, sel]) => {
        if (cancelled) return
        setSchemes(ls.schemes || [])
        setCurrent(sel.font_scheme || '')
        injectFontFaces(ls.base_url || '', ls.schemes || [])
      })
      .catch(() => { if (!cancelled) setMessage('❌ 字体方案加载失败，可刷新页面重试') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [coursewareId])

  // 应用某方案（scheme=null 表示清除，回退模板默认字体）
  const apply = async (scheme: CWFontScheme | null) => {
    if (busy || disabled) return
    setApplying(scheme ? scheme.code : '__clear__')
    setMessage(scheme ? '⏳ 正在应用字体并秒换已生成页…' : '⏳ 正在恢复模板默认字体…')
    try {
      const res = scheme ? await setCWFont(coursewareId, scheme.code) : await clearCWFont(coursewareId)
      setCurrent(res.font_scheme || '')
      if (scheme) {
        setMessage('✅ 已应用「' + scheme.name + '」' +
          (res.swapped_pages > 0 ? '，已秒换 ' + res.swapped_pages + ' 个已生成页' : '') +
          '，后续生成的页面将自动使用该字体')
      } else {
        setMessage('✅ 已恢复模板默认字体' + (res.swapped_pages > 0 ? '，已回退 ' + res.swapped_pages + ' 个已生成页' : ''))
      }
      onSwapped()
    } catch (e) {
      setMessage('❌ 操作失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setApplying('') }
  }

  const noneSelected = current === ''

  return (
    <div style={{ marginTop: 16, padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#FAFAFA', opacity: disabled ? 0.75 : 1 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
        <span style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>🔤 课件字体（可选）</span>
        {/* F2b: 常驻状态徽标——读服务器实时状态, 一眼确认当前生效的字体方案 */}
        {!loading && (
          <span style={{ padding: '1px 8px', borderRadius: 999, fontSize: 11, fontWeight: 600, color: current ? '#059669' : C.textMuted, background: current ? '#D1FAE5' : '#F3F4F6' }}>
            当前：{current ? (schemes.find(s => s.code === current)?.name || current) : '跟随模板'}
          </span>
        )}
      </div>
      <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 12 }}>
        一套字体 = 标题字体 + 正文字体的搭配，全部为开源免版权字体（可随离线包分发）。点选后已生成页立即换上新字体，后续生成的页面自动使用。
      </div>

      {/* 批量生成进行中禁用提示（口径与背景选择器一致） */}
      {disabled && (
        <div style={{ marginBottom: 12, padding: '10px 14px', borderRadius: 8, background: '#FEF3C7', color: '#92400E', fontSize: 13 }}>
          ⏳ 批量生成进行中，暂不能更换字体（避免正在生成的页面带上新旧混杂的字体）。生成完成后即可操作。
        </div>
      )}

      {loading && <div style={{ padding: '14px 0', fontSize: 13, color: C.textMuted, textAlign: 'center' }}>⏳ 字体方案加载中...</div>}

      {!loading && (
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          {/* 跟随模板（=清除字体选择） */}
          <div onClick={() => { if (!busy && !disabled && !noneSelected) apply(null) }}
            style={{
              width: 210, borderRadius: 10, overflow: 'hidden', cursor: (busy || disabled || noneSelected) ? 'default' : 'pointer',
              border: '2px ' + (noneSelected ? 'solid ' + C.primary : 'dashed ' + C.border),
              background: noneSelected ? C.primaryBg : '#fff', transition: 'all 200ms',
            }}>
            <div style={{ height: 72, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', background: '#F3F4F6', gap: 2 }}>
              <div style={{ fontSize: 22 }}>🎨</div>
              <div style={{ fontSize: 11, color: C.textMuted }}>由风格模板/AI决定字体</div>
            </div>
            <div style={{ padding: '8px 10px' }}>
              <div style={{ fontSize: 13, fontWeight: 600, color: noneSelected ? C.primary : C.textPrimary }}>
                跟随模板{noneSelected ? '（当前）' : ''}
              </div>
              <div style={{ fontSize: 11, color: C.textMuted, marginTop: 2 }}>
                {applying === '__clear__' ? '⏳ 恢复中...' : '不强制统一字体'}
              </div>
            </div>
          </div>

          {/* 5套方案卡片：用真实woff2渲染字样预览（标题行=标题字体 / 正文行=正文字体） */}
          {schemes.map(s => {
            const selected = current === s.code
            const cardBusy = applying === s.code
            const headingStack = s.heading_family + CJK_FALLBACK
            const bodyStack = s.body_family + CJK_FALLBACK
            return (
              <div key={s.code} onClick={() => { if (!busy && !disabled && !selected) apply(s) }}
                style={{
                  width: 210, borderRadius: 10, overflow: 'hidden',
                  cursor: (busy || disabled || selected) ? 'default' : 'pointer',
                  border: '2px solid ' + (selected ? C.primary : C.border),
                  background: selected ? C.primaryBg : '#fff', transition: 'all 200ms',
                }}>
                {/* 字样预览区：英语方案能直接看到 Poppins 单层 a 的真实字形 */}
                <div style={{ height: 72, padding: '8px 12px', background: '#fff', borderBottom: '1px solid ' + C.border, display: 'flex', flexDirection: 'column', justifyContent: 'center', gap: 2, overflow: 'hidden' }}>
                  <div style={{ fontFamily: headingStack, fontSize: 17, fontWeight: 700, color: '#1E293B', whiteSpace: 'nowrap', lineHeight: 1.3 }}>
                    {s.code === 'english' ? 'Reading Aa Gg' : '春风化雨 Aa'}
                  </div>
                  <div style={{ fontFamily: bodyStack, fontSize: 12, color: '#475569', whiteSpace: 'nowrap', lineHeight: 1.4 }}>
                    {s.code === 'english' ? 'an apple and a book 苹果' : '课件正文字样 apple 123'}
                  </div>
                </div>
                <div style={{ padding: '8px 10px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span style={{ flex: 1, fontSize: 13, fontWeight: 600, color: selected ? C.primary : C.textPrimary, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{s.name}</span>
                    <span style={{ flexShrink: 0, padding: '1px 6px', borderRadius: 4, fontSize: 10, fontWeight: 600, color: '#2563EB', background: '#EFF6FF' }}>系统</span>
                  </div>
                  <div style={{ fontSize: 11, color: C.textMuted, marginTop: 2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={s.description}>
                    {cardBusy ? '⏳ 应用中...' : selected ? '✅ 当前使用中' : (s.heading_label === s.body_label ? s.heading_label : s.heading_label + ' + ' + s.body_label)}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {message && (
        <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, fontSize: 13, background: message.startsWith('❌') ? '#FEE2E2' : message.startsWith('✅') ? '#D1FAE5' : '#EFF6FF', color: message.startsWith('❌') ? '#DC2626' : message.startsWith('✅') ? '#059669' : '#2563EB' }}>{message}</div>
      )}
    </div>
  )
}

/**
 * PagePreviewBlock.tsx — 课件页面胶片条+大预览块（批次5b-2从主页面 renderPagePreview 抽出）
 *
 * 拆出范围：W3单行横滚胶片条 + 第N页预览/源代码双视图 + 复制代码/全屏/放映按钮。
 * 确认导航栏(Step3)/批量生成(Step4)/确认提交(Step5)三步共享本组件。
 *
 * 状态归属：codeViewPageNum（源码视图开关）随组件内置——切步骤重挂载时复位为预览视图，
 * 属已知可接受的微变更（源码查看是低频动作，跨步骤保持无实际价值）。
 *
 * 另导出 MsgBar 共享消息条组件（原主页面 msgBar 辅助函数），供主页面与
 * SchemeSteps/NavConfirmStep 三处复用，按 ❌/✅/⚠️ 前缀自动配色。
 *
 * 预览刷新修复（页面级版本与回退轮）：
 *   预览用 iframe srcDoc 渲染。React 更新 iframe 的 srcDoc 属性时，浏览器不保证重新加载
 *   iframe 内容——当 activePage 不变（如原地微调/重生/回退同一页，页号没变），srcDoc 值变了
 *   但 iframe 仍显示旧内容，表现为"后端已改、前端预览不动"。
 *   修复：给 iframe 加 key=`p{页号}-{预览HTML长度}`，内容变化时 key 变 → React 卸载旧 iframe、
 *   挂载新 iframe，强制浏览器重新解析渲染最新 HTML。切页时页号变、同页改写时长度变，均能正确刷新。
 */
import { useState } from 'react'
import { C, CW_WIDTH, CW_HEIGHT } from './workshopConstants'
import { injectPreviewMode } from './previewInject'

/** 页面条目（主页面 previewPages/generatedPages 的统一元素类型） */
export interface PageItem {
  page_number: number
  title: string
  html_content: string
}

/** 共享消息条：按消息前缀(❌/✅/⚠️/其它)自动配色，空消息不渲染 */
export function MsgBar({ msg }: { msg: string }) {
  if (!msg) return null
  return (
    <div style={{ padding: '12px 16px', borderRadius: 8, marginBottom: 16, background: msg.startsWith('❌') ? '#FEE2E2' : msg.startsWith('✅') ? '#D1FAE5' : msg.startsWith('⚠️') ? '#FEF3C7' : '#EFF6FF', color: msg.startsWith('❌') ? '#DC2626' : msg.startsWith('✅') ? '#059669' : msg.startsWith('⚠️') ? '#D97706' : '#2563EB', fontSize: 14 }}>{msg}</div>
  )
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
  /** 放映回调（无参=从当前/默认页起；带参=从该页起） */
  onSlideshow: (pn?: number) => void
  /** 全屏预览回调（带工具栏的预览，非放映） */
  onFullscreen: (pn: number) => void
}

export default function PagePreviewBlock({ pages, currentNum, onSelectPage, showSlideshow, onSlideshow, onFullscreen }: Props) {
  // v137: 源代码查看状态（与预览互斥；批次5b-2迁入本组件）
  const [codeViewPageNum, setCodeViewPageNum] = useState(0)
  // 预览缩放：容器固定912宽，1920画布等比缩入
  const containerWidth = 912
  const previewScale = containerWidth / CW_WIDTH

  const activePage = currentNum > 0 ? currentNum : (pages[0]?.page_number || 0)
  const html = pages.find(p => p.page_number === activePage)?.html_content || ''
  const previewHtml = injectPreviewMode(html)

  return <>
    {pages.length > 0 && (
      <div style={{ marginBottom: 20 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: C.textPrimary }}>📄 已生成 {pages.length} 页</div>
          {showSlideshow && <button onClick={() => onSlideshow()} style={{ padding: '6px 14px', borderRadius: 8, border: `1px solid ${C.primary}`, background: C.primaryBg, color: C.primary, fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>🖥️ 全屏放映</button>}
        </div>
        {/* W3: 胶片条——单行横向滚动, 页数再多也只占一行高度(PPT/Canva同款导航模式) */}
        <div style={{ display: 'flex', gap: 6, flexWrap: 'nowrap', overflowX: 'auto', paddingBottom: 6 }}>
          {pages.map(gp => (
            <button key={gp.page_number} onClick={() => onSelectPage(gp.page_number)} title={'P' + gp.page_number + ' ' + gp.title} style={{
              padding: '6px 10px', borderRadius: 8, cursor: 'pointer', flexShrink: 0, whiteSpace: 'nowrap',
              border: `2px solid ${activePage === gp.page_number ? C.primary : C.border}`,
              background: activePage === gp.page_number ? C.primaryBg : C.white,
              color: activePage === gp.page_number ? C.primary : C.textPrimary,
              fontSize: 12, fontWeight: activePage === gp.page_number ? 600 : 400, transition: 'all 200ms',
            }}>
              <span style={{ fontWeight: 700 }}>P{gp.page_number}</span>
              <span style={{ marginLeft: 5, color: C.textSecondary, fontSize: 11 }}>{gp.title.length > 6 ? gp.title.slice(0, 6) + '…' : gp.title}</span>
            </button>
          ))}
        </div>
      </div>
    )}
    {html && (
      <div style={{ marginBottom: 20 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: C.textPrimary }}>{codeViewPageNum === activePage ? '💻' : '📺'} 第 {activePage} 页{codeViewPageNum === activePage ? '源代码' : '预览'}</div>
          <div style={{ display: 'flex', gap: 6 }}>
            <button onClick={() => { if (codeViewPageNum === activePage) setCodeViewPageNum(0); else setCodeViewPageNum(activePage) }} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${codeViewPageNum === activePage ? '#7C3AED' : C.border}`, background: codeViewPageNum === activePage ? 'rgba(124,58,237,0.06)' : 'transparent', color: codeViewPageNum === activePage ? '#7C3AED' : C.textSecondary, fontSize: 12, cursor: 'pointer' }}>{codeViewPageNum === activePage ? '📺 预览' : '💻 源代码'}</button>
            <button onClick={() => { navigator.clipboard.writeText(html).then(() => alert('源代码已复制到剪贴板')).catch(() => {}) }} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer' }}>📋 复制代码</button>
            <button onClick={() => onFullscreen(activePage)} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer' }}>🔍 全屏预览</button>
            <button onClick={() => onSlideshow(activePage)} style={{ padding: '4px 10px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: 12, cursor: 'pointer' }}>🖥️ 放映</button>
          </div>
        </div>
        {codeViewPageNum === activePage ? (
          <div style={{ width: '100%', maxHeight: 500, overflow: 'auto', borderRadius: 14, border: `1px solid ${C.border}`, background: '#1e1e1e', fontFamily: 'Monaco, Consolas, "Courier New", monospace', fontSize: 12, lineHeight: 1.7 }}>
            <table style={{ borderCollapse: 'collapse', width: '100%' }}><tbody>
              {html.split('\n').map((line: string, i: number) => (
                <tr key={i}>
                  <td style={{ width: 50, minWidth: 50, textAlign: 'right', padding: '0 10px 0 8px', color: '#858585', userSelect: 'none', verticalAlign: 'top', borderRight: '1px solid #333', whiteSpace: 'nowrap' }}>{i + 1}</td>
                  <td style={{ padding: '0 12px', color: '#d4d4d4', whiteSpace: 'pre', wordBreak: 'break-all' }}>{line || ' '}</td>
                </tr>
              ))}
            </tbody></table>
          </div>
        ) : (
          <div onClick={() => onSlideshow(activePage)} style={{
            width: '100%', height: Math.ceil(CW_HEIGHT * previewScale), position: 'relative', overflow: 'hidden',
            borderRadius: 14, border: `1px solid ${C.border}`, background: '#f8fafc', cursor: 'pointer',
          }}>
            {/* key 含预览HTML长度：原地微调/重生/回退后内容变化→key变→iframe重建→强制刷新最新HTML */}
            <iframe key={`p${activePage}-${previewHtml.length}`} srcDoc={previewHtml} scrolling="no" style={{ width: CW_WIDTH, height: CW_HEIGHT, border: 'none', pointerEvents: 'none', transform: `scale(${previewScale})`, transformOrigin: 'top left', position: 'absolute', top: 0, left: 0, overflow: 'hidden' }} sandbox="allow-scripts" title={`预览-P${activePage}`} />
          </div>
        )}
      </div>
    )}
  </>
}

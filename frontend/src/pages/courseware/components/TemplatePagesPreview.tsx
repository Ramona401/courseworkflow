/**
 * 课件模板多页预览组件 — TemplatePagesPreview v1.0 (2026-06-10)
 *
 * 背景：新一批"世界级"系统模板每套 5 页（封面/目标/内容/互动/作业），存于 sample_pages 数组。
 *       旧预览只渲染 sample_pages[0]（仅封面），看不到其余子页面。
 * 作用：把整套 sample_pages 逐页渲染（复用 TemplateThumbAuto 的等比缩放与沙箱），每页带页型标签。
 *
 * 渲染规则：
 *   1. 3D 模板（previewUrls 非空）：是完整 Three.js 交互文档，渲染单个交互预览，不拆页。
 *   2. 普通模板：逐个 sample_pages 渲染；恰好 5 页时用固定页型名（封面/目标/内容/互动/作业），
 *      其它数量回退"第 N 页"。
 *   3. 无样例：占位提示。
 *
 * 复用位置：风格选择器(StyleSelector)预览弹窗 + 模板管理页(CWTemplatesPage)预览弹窗。
 */
import { TemplateThumbAuto } from './TemplateThumb'

// 5 页标准模板的固定页型名（与课件五页固定页型 封面/目标/内容/互动/作业 对齐）
const FIVE_PAGE_LABELS = ['封面', '学习目标', '内容讲解', '互动练习', '课后作业']

interface TemplatePagesPreviewProps {
  /** 3D 模板预览 URL 数组（取第一个；非空即按 3D 单预览处理） */
  previewUrls?: string[]
  /** 普通模板样例页 HTML 数组（逐页渲染） */
  samplePages?: string[]
  /** 页序号徽章底色（沿用模板主色，纯装饰；默认琥珀色） */
  accentColor?: string
}

export default function TemplatePagesPreview({
  previewUrls = [],
  samplePages = [],
  accentColor = '#F59E0B',
}: TemplatePagesPreviewProps) {
  // 分支1：3D 模板 —— 完整交互文档，单预览不拆页
  if (previewUrls.length > 0 && previewUrls[0]) {
    return (
      <div>
        <div style={{ fontSize: '12px', color: '#9CA3AF', marginBottom: '8px' }}>
          🎮 3D 沉浸式课件（完整交互文档，下方为实时预览）
        </div>
        <TemplateThumbAuto previewUrl={previewUrls[0]} title="3D 模板预览" />
      </div>
    )
  }

  // 分支2：无样例页
  if (!samplePages || samplePages.length === 0) {
    return (
      <div style={{
        padding: '40px 0', textAlign: 'center', color: '#9CA3AF',
        border: '1px dashed #E5E7EB', borderRadius: '14px', fontSize: '14px',
      }}>暂无样例页面</div>
    )
  }

  // 分支3：普通模板 —— 逐页渲染（封面/目标/内容/互动/作业 或 第 N 页）
  const total = samplePages.length
  const labelOf = (idx: number): string => {
    if (total === 5) return FIVE_PAGE_LABELS[idx] || `第 ${idx + 1} 页`
    return `第 ${idx + 1} 页`
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '22px' }}>
      {samplePages.map((html, idx) => (
        <div key={idx}>
          {/* 页型标签条 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
            <span style={{
              display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
              minWidth: '22px', height: '22px', padding: '0 7px', borderRadius: '11px',
              background: accentColor, color: '#fff', fontSize: '12px', fontWeight: 700,
            }}>{idx + 1}</span>
            <span style={{ fontSize: '13px', fontWeight: 600, color: '#374151' }}>{labelOf(idx)}</span>
            <span style={{ fontSize: '11px', color: '#9CA3AF' }}>共 {total} 页</span>
          </div>
          {/* 单页等比缩放预览 */}
          <TemplateThumbAuto sampleHTML={html} title={`样例页 ${idx + 1}`} />
        </div>
      ))}
    </div>
  )
}

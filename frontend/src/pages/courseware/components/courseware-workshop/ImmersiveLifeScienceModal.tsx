/**
 * ImmersiveLifeScienceModal.tsx — 3D生命科学整页实验库
 *
 * 完整3D工作台以整页方式替换当前课件页，不经过AI改写。
 * 模板源码保存在静态资产库，前端只加载当前选中的一个WebGL预览。
 */
import { useMemo, useState } from 'react'
import { importPageHtml } from '@/api/coursewares'
import { C } from './workshopConstants'
import { IMMERSIVE_LIFE_SCIENCE_TEMPLATES } from './immersiveLifeScienceTemplates'
import type { ImmersiveLifeScienceTemplate } from './immersiveLifeScienceTemplates'
interface Props {
  coursewareId: string
  pageNum: number
  onPageUpdated?: (pageNum: number, html: string) => void
  onClose: () => void
}
const panel: React.CSSProperties = {
  borderRadius: 16, border: '1px solid #DDEBE4', background: '#FFFFFF',
}
function extractImportedHtml(value: unknown, fallback: string): string {
  if (typeof value === 'string' && value.trim()) return value
  if (!value || typeof value !== 'object') return fallback
  const records: Record<string, unknown>[] = [value as Record<string, unknown>]
  const nested = (value as Record<string, unknown>).data
  if (nested && typeof nested === 'object') {
    records.push(nested as Record<string, unknown>)
  }
  for (const record of records) {
    for (const key of ['html', 'html_content', 'content']) {
      const candidate = record[key]
      if (typeof candidate === 'string' && candidate.trim()) return candidate
    }
  }
  return fallback
}
function TemplateCard({
  template,
  active,
  onPick,
}: {
  template: ImmersiveLifeScienceTemplate
  active: boolean
  onPick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onPick}
      style={{
        ...panel,
        width: '100%', display: 'grid', gridTemplateColumns: '48px minmax(0,1fr)', gap: 12, alignItems: 'center', marginBottom: 10, padding: 13,
        textAlign: 'left', borderColor: active ? template.accent : '#DFE8E2', background: active ? template.softBackground : '#FFFFFF',
        boxShadow: active
          ? '0 10px 24px rgba(15,118,74,0.14)'
          : '0 3px 10px rgba(15,23,42,0.04)',
        cursor: 'pointer',
      }}
    >
      <span
        style={{
          width: 48, height: 48, display: 'grid', placeItems: 'center', borderRadius: 14, background: '#FFFFFF',
          boxShadow: '0 5px 15px rgba(15,23,42,0.08)', fontSize: 24,
        }}
      >
        {template.emoji}
      </span>
      <span style={{ minWidth: 0 }}>
        <span
          style={{
            display: 'block', color: active ? template.accent : C.textPrimary, fontSize: 14, fontWeight: 850,
          }}
        >
          {template.name}
        </span>
        <span
          style={{
            display: 'block', marginTop: 5, color: C.textMuted, fontSize: 11.5, lineHeight: 1.5,
          }}
        >
          {template.summary}
        </span>
      </span>
    </button>
  )
}
function Tag({ children }: { children: string }) {
  return (
    <span
      style={{
        padding: '5px 9px', borderRadius: 999, background: '#F0FDF4', color: '#166534', fontSize: 11.5, fontWeight: 700,
      }}
    >
      {children}
    </span>
  )
}
export default function ImmersiveLifeScienceModal({
  coursewareId,
  pageNum,
  onPageUpdated,
  onClose,
}: Props) {
  const [activeId, setActiveId] = useState(
    IMMERSIVE_LIFE_SCIENCE_TEMPLATES[0].id,
  )
  const [applying, setApplying] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const active = useMemo(
    () =>
      IMMERSIVE_LIFE_SCIENCE_TEMPLATES.find(item => item.id === activeId)
      || IMMERSIVE_LIFE_SCIENCE_TEMPLATES[0],
    [activeId],
  )
  const pickTemplate = (template: ImmersiveLifeScienceTemplate) => {
    if (applying) return
    setActiveId(template.id)
    setMessage('')
    setError('')
  }
  const applyCurrentPage = async () => {
    if (applying) return
    if (!coursewareId || pageNum <= 0) {
      setError('请先选择一张可以编辑的课件页面。')
      return
    }
    const confirmed = window.confirm(
      '将用“' + active.name + '”整体替换第 ' + pageNum
      + ' 页。\n\n原页面会自动保存为历史版本，可在版本历史中回退。是否继续？',
    )
    if (!confirmed) return
    setApplying(true)
    setMessage('')
    setError('')
    try {
      const response = await fetch(active.sourceUrl, {
        cache: 'no-store', credentials: 'same-origin',
      })
      if (!response.ok) {
        throw new Error('读取3D实验模板失败，HTTP ' + response.status)
      }
      const html = await response.text()
      if (!html.trim()) throw new Error('3D实验模板内容为空')
      if (html.length > 5 * 1024 * 1024) {
        throw new Error('3D实验模板超过5MB导入上限')
      }
      if (!html.includes('tedna-page-mode')) {
        throw new Error('3D实验模板缺少平台整页标记')
      }
      const result: unknown = await importPageHtml(
        coursewareId,
        pageNum,
        html,
      )
      onPageUpdated?.(pageNum, extractImportedHtml(result, html))
      setMessage(
        '✅ 已将“' + active.name + '”整页应用到第 ' + pageNum + ' 页。',
      )
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '整页应用失败')
    } finally {
      setApplying(false)
    }
  }
  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 99996, display: 'grid', placeItems: 'center', padding: '2vh 1vw', background: 'rgba(7,25,18,0.68)',
        backdropFilter: 'blur(7px)',
      }}
      onClick={() => {
        if (!applying) onClose()
      }}
    >
      <div
        style={{
          width: 'min(1600px,98vw)', height: 'min(940px,96vh)', display: 'flex', flexDirection: 'column', overflow: 'hidden', borderRadius: 24,
          background: '#FFFFFF', boxShadow: '0 34px 90px rgba(0,0,0,0.42)',
        }}
        onClick={event => event.stopPropagation()}
      >
        <header
          style={{
            minHeight: 72, display: 'flex', alignItems: 'center', gap: 14, padding: '12px 22px', color: '#FFFFFF',
            background:
              'linear-gradient(135deg,#0F766E,#059669 48%,#166534)',
          }}
        >
          <span
            style={{
              width: 48, height: 48, display: 'grid', placeItems: 'center', flexShrink: 0, borderRadius: 15, background: 'rgba(255,255,255,0.18)',
              fontSize: 25,
            }}
          >
            🌐
          </span>
          <div>
            <div style={{ fontSize: 20, fontWeight: 900 }}>
              3D生命科学实验室
            </div>
            <div
              style={{
                marginTop: 3, color: 'rgba(255,255,255,0.82)', fontSize: 12.5,
              }}
            >
              完整3D工作台 · 整页应用 · 确定性导入 · 自动保留历史版本
            </div>
          </div>
          <span
            style={{
              marginLeft: 8, padding: '6px 13px', border: '1px solid rgba(255,255,255,0.3)', borderRadius: 999, background: 'rgba(255,255,255,0.14)',
              fontSize: 12, fontWeight: 800,
            }}
          >
            当前目标：第 {pageNum} 页
          </span>
          <button
            type="button"
            onClick={() => {
              if (!applying) onClose()
            }}
            disabled={applying}
            title="关闭"
            style={{
              marginLeft: 'auto', width: 38, height: 38, border: 'none', borderRadius: 12, background: 'rgba(255,255,255,0.15)', color: '#FFFFFF',
              fontSize: 19, cursor: applying ? 'not-allowed' : 'pointer',
            }}
          >
            ✕
          </button>
        </header>
        <div
          style={{
            flex: 1, minHeight: 0, display: 'grid', gridTemplateColumns: '292px minmax(0,1fr) 330px',
          }}
        >
          <aside
            style={{
              minHeight: 0, overflowY: 'auto', padding: 16, borderRight: '1px solid #DDEBE4', background: 'linear-gradient(180deg,#F0FDF4,#F8FAFC)',
            }}
          >
            <div style={{ ...panel, marginBottom: 12, padding: '10px 11px' }}>
              <div style={{ color: '#166534', fontSize: 13, fontWeight: 850 }}>
                🧬 选择整页3D实验
              </div>
              <div
                style={{
                  marginTop: 5, color: C.textMuted, fontSize: 11.5, lineHeight: 1.55,
                }}
              >
                每次只运行当前选中的预览，避免多个WebGL场景同时占用显卡。
              </div>
            </div>
            {IMMERSIVE_LIFE_SCIENCE_TEMPLATES.map(template => (
              <TemplateCard
                key={template.id}
                template={template}
                active={template.id === active.id}
                onPick={() => pickTemplate(template)}
              />
            ))}
          </aside>
          <main
            style={{
              minWidth: 0, minHeight: 0, display: 'flex', flexDirection: 'column', padding: 16,
              background:
                'radial-gradient(circle at 50% 0%,#FFFFFF,#EDF8F2)',
            }}
          >
            <div
              style={{
                display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10,
              }}
            >
              <span style={{ fontSize: 21 }}>{active.emoji}</span>
              <span
                style={{
                  color: C.textPrimary, fontSize: 16, fontWeight: 900,
                }}
              >
                {active.name}
              </span>
              <span style={{ color: C.textMuted, fontSize: 12 }}>
                拖动、滚轮和点击均可直接测试
              </span>
              <a
                href={active.previewUrl}
                target="_blank"
                rel="noreferrer"
                style={{
                  marginLeft: 'auto', padding: '7px 12px', border: '1px solid #DDEBE4', borderRadius: 10, background: '#FFFFFF', color: active.accent,
                  fontSize: 12, fontWeight: 800, textDecoration: 'none',
                }}
              >
                ↗ 独立打开
              </a>
            </div>
            <div
              style={{
                flex: 1, minHeight: 0, overflow: 'hidden', border: '1px solid #CFE3D7', borderRadius: 17, background: '#FFFFFF',
                boxShadow: '0 18px 45px rgba(15,118,74,0.14)',
              }}
            >
              <iframe
                key={active.id}
                title={active.name}
                src={active.previewUrl}
                sandbox="allow-scripts allow-same-origin"
                allow="fullscreen"
                style={{
                  width: '100%', height: '100%', display: 'block', border: 'none', background: '#FFFFFF',
                }}
              />
            </div>
          </main>
          <aside
            style={{
              minHeight: 0, overflowY: 'auto', padding: 18, borderLeft: '1px solid #DDEBE4', background: '#FFFFFF',
            }}
          >
            <div
              style={{
                ...panel,
                padding: 15, background: active.softBackground,
              }}
            >
              <div
                style={{
                  color: active.accent, fontSize: 12, fontWeight: 800,
                }}
              >
                {active.category} · {active.stage}
              </div>
              <div
                style={{
                  marginTop: 7, color: C.textPrimary, fontSize: 18, fontWeight: 900,
                }}
              >
                {active.name}
              </div>
              <div
                style={{
                  marginTop: 9, color: C.textSecondary, fontSize: 12.5, lineHeight: 1.7,
                }}
              >
                {active.description}
              </div>
            </div>
            <section style={{ marginTop: 18 }}>
              <div
                style={{
                  color: C.textPrimary, fontSize: 13, fontWeight: 850,
                }}
              >
                知识点
              </div>
              <div
                style={{
                  display: 'flex', flexWrap: 'wrap', gap: 7, marginTop: 9,
                }}
              >
                {active.knowledgePoints.map(item => (
                  <Tag key={item}>{item}</Tag>
                ))}
              </div>
            </section>
            <section style={{ marginTop: 18 }}>
              <div
                style={{
                  color: C.textPrimary, fontSize: 13, fontWeight: 850,
                }}
              >
                互动能力
              </div>
              <div style={{ marginTop: 9 }}>
                {active.capabilities.map(item => (
                  <div
                    key={item}
                    style={{
                      display: 'flex', gap: 8, alignItems: 'flex-start', marginBottom: 8, color: C.textSecondary, fontSize: 12, lineHeight: 1.5,
                    }}
                  >
                    <span style={{ color: active.accent, fontWeight: 900 }}>
                      ✓
                    </span>
                    <span>{item}</span>
                  </div>
                ))}
              </div>
            </section>
            <div
              style={{
                marginTop: 18, padding: 12, border: '1px solid #F6D9A7', borderRadius: 13, background: '#FFFBEB', color: '#92400E', fontSize: 11.8,
                lineHeight: 1.65,
              }}
            >
              整页应用会替换当前页，而不是把工作台缩成小插件。原页会由后端自动保存为历史版本。
            </div>
            {message && (
              <div
                style={{
                  marginTop: 12, padding: 11, borderRadius: 12, background: '#ECFDF5', color: '#065F46', fontSize: 12, fontWeight: 700,
                  lineHeight: 1.5,
                }}
              >
                {message}
              </div>
            )}
            {error && (
              <div
                style={{
                  marginTop: 12, padding: 11, borderRadius: 12, background: '#FEF2F2', color: '#B91C1C', fontSize: 12, fontWeight: 700,
                  lineHeight: 1.5,
                }}
              >
                ❌ {error}
              </div>
            )}
          </aside>
        </div>
        <footer
          style={{
            minHeight: 66, display: 'flex', alignItems: 'center', gap: 12, padding: '10px 22px', borderTop: '1px solid #DDEBE4',
            background: '#F0FDF4',
          }}
        >
          <span style={{ color: C.textMuted, fontSize: 12.5 }}>
            模板从平台静态资产库读取，导入不调用AI、不消耗模型Token。
          </span>
          <button
            type="button"
            onClick={() => {
              if (!applying) onClose()
            }}
            disabled={applying}
            style={{
              marginLeft: 'auto', padding: '10px 22px', border: '1px solid #CFE3D7', borderRadius: 12, background: '#FFFFFF', color: C.textSecondary,
              fontSize: 13.5, fontWeight: 750, cursor: applying ? 'not-allowed' : 'pointer',
            }}
          >
            取消
          </button>
          <button
            type="button"
            onClick={applyCurrentPage}
            disabled={applying || pageNum <= 0}
            style={{
              padding: '10px 25px', border: 'none', borderRadius: 12,
              background:
                applying || pageNum <= 0
                  ? '#86D3B4'
                  : 'linear-gradient(135deg,#10B981,#047857)',
              boxShadow:
                applying || pageNum <= 0
                  ? 'none'
                  : '0 8px 22px rgba(5,150,105,0.3)',
              color: '#FFFFFF', fontSize: 13.5, fontWeight: 900,
              cursor:
                applying || pageNum <= 0
                  ? 'not-allowed'
                  : 'pointer',
            }}
          >
            {applying
              ? '⏳ 正在整页应用…'
              : '🌐 整页应用到第 ' + pageNum + ' 页'}
          </button>
        </footer>
      </div>
    </div>
  )
}

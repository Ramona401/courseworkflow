/**
 * TemplatePageReferencePicker.tsx
 *
 * 单页“全页重构”模式的具体模板页参考选择器。
 *
 * 流程：
 *   1. 加载当前用户可见的正式模板与本人草稿。
 *   2. 左侧选择模板，右侧选择模板中的具体一页。
 *   3. 页面预览使用TemplateThumbAuto，只做静态安全预览。
 *   4. 交给父组件的只有templateId和samplePageIndex，不传HTML。
 *   5. 提交时由RefinePanel生成内部引用标记。
 *   6. 后端重新读取模板并做权限校验，前端选择结果不作为权限依据。
 */

import { useState } from 'react'
import {
  getCWTemplatesWithUser,
  listMyDrafts,
} from '@/api/coursewares'
import type { CoursewareTemplate } from '@/api/coursewares'
import { TemplateThumbAuto } from '../TemplateThumb'
import { C } from './workshopConstants'

// ==================== 对外类型与协议 ====================

export interface TemplatePageReferenceSelection {
  templateId: string
  templateName: string
  templateScope: string
  samplePageIndex: number
  samplePageCount: number
}

/**
 * 构建后端可识别的内部模板页引用标记。
 *
 * 不包含HTML，后端会根据模板ID重新读取页面并验证权限。
 */
export function buildTemplatePageReferenceMarker(
  selection: TemplatePageReferenceSelection,
): string {
  return (
    '<!-- TEDNA_TEMPLATE_PAGE_REF '
    + JSON.stringify({
      template_id: selection.templateId,
      sample_page_index: selection.samplePageIndex,
    })
    + ' -->'
  )
}

interface Props {
  selected: TemplatePageReferenceSelection | null
  onSelect: (selection: TemplatePageReferenceSelection) => void
  onRemove: () => void
  disabled: boolean
}

// ==================== 内部类型 ====================

interface TemplateWithPages {
  template: CoursewareTemplate
  pages: string[]
}

// ==================== 辅助函数 ====================

function safeParsePages(raw: string): string[] {
  try {
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []

    return parsed.filter(
      page => typeof page === 'string' && page.trim().length > 0,
    )
  } catch {
    return []
  }
}

function scopeLabel(template: CoursewareTemplate): string {
  if (template.is_draft) return '我的草稿'

  switch (template.scope) {
    case 'personal':
      return '我的模板'
    case 'group':
      return '教研组模板'
    case 'school':
      return '学校模板'
    case 'system':
      return '系统模板'
    default:
      return '模板'
  }
}

function scopeColor(template: CoursewareTemplate): {
  color: string
  background: string
} {
  if (template.is_draft) {
    return {
      color: '#6D28D9',
      background: '#EDE9FE',
    }
  }

  switch (template.scope) {
    case 'personal':
      return {
        color: '#047857',
        background: '#D1FAE5',
      }
    case 'group':
      return {
        color: '#6D28D9',
        background: '#EDE9FE',
      }
    case 'school':
      return {
        color: '#1D4ED8',
        background: '#DBEAFE',
      }
    default:
      return {
        color: '#B45309',
        background: '#FEF3C7',
      }
  }
}

function mergeVisibleTemplates(
  published: CoursewareTemplate[],
  drafts: CoursewareTemplate[],
): TemplateWithPages[] {
  const seen = new Set<string>()
  const merged: TemplateWithPages[] = []

  // 本人草稿优先，方便老师刚提取完模板后立即引用。
  for (const template of [...drafts, ...published]) {
    if (!template?.id || seen.has(template.id)) continue

    const pages = safeParsePages(template.sample_pages)
    if (pages.length === 0) continue

    seen.add(template.id)
    merged.push({
      template,
      pages,
    })
  }

  return merged
}

// ==================== 主组件 ====================

export default function TemplatePageReferencePicker({
  selected,
  onSelect,
  onRemove,
  disabled,
}: Props) {
  const [open, setOpen] = useState(false)
  const [templates, setTemplates] = useState<TemplateWithPages[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [activeTemplateId, setActiveTemplateId] = useState('')
  const [activePageIndex, setActivePageIndex] = useState(0)

  const activeTemplate = templates.find(
    item => item.template.id === activeTemplateId,
  ) || null

  const activePageHTML = activeTemplate
    ? activeTemplate.pages[activePageIndex] || ''
    : ''

  const openPicker = async () => {
    if (disabled) return

    setOpen(true)
    setLoading(true)
    setError('')

    try {
      const [published, drafts] = await Promise.all([
        getCWTemplatesWithUser().catch(() => []),
        listMyDrafts().catch(() => []),
      ])

      const merged = mergeVisibleTemplates(
        published || [],
        drafts || [],
      )

      setTemplates(merged)

      const selectedTemplate = selected
        ? merged.find(
          item => item.template.id === selected.templateId,
        )
        : null

      const initialTemplate = selectedTemplate || merged[0] || null

      if (initialTemplate) {
        setActiveTemplateId(initialTemplate.template.id)

        const requestedIndex = selectedTemplate
          ? selected?.samplePageIndex || 0
          : 0

        setActivePageIndex(
          Math.min(
            requestedIndex,
            initialTemplate.pages.length - 1,
          ),
        )
      } else {
        setActiveTemplateId('')
        setActivePageIndex(0)
      }
    } catch (loadError) {
      setError(
        '加载模板失败：'
        + (
          loadError instanceof Error
            ? loadError.message
            : '未知错误'
        ),
      )
    } finally {
      setLoading(false)
    }
  }

  const chooseTemplate = (item: TemplateWithPages) => {
    setActiveTemplateId(item.template.id)
    setActivePageIndex(0)
  }

  const confirmSelection = () => {
    if (!activeTemplate) return

    if (
      activePageIndex < 0
      || activePageIndex >= activeTemplate.pages.length
    ) {
      setError('请选择有效的模板页面')
      return
    }

    onSelect({
      templateId: activeTemplate.template.id,
      templateName: activeTemplate.template.name,
      templateScope: activeTemplate.template.scope,
      samplePageIndex: activePageIndex,
      samplePageCount: activeTemplate.pages.length,
    })

    setOpen(false)
  }

  return (
    <>
      {/* ==================== 工具条按钮或已选芯片 ==================== */}
      {selected ? (
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            maxWidth: 360,
            padding: '8px 12px',
            borderRadius: 8,
            border: '1px solid #EA580C',
            background: '#FFF7ED',
            color: '#C2410C',
            fontSize: 13,
            fontWeight: 600,
            whiteSpace: 'nowrap',
          }}
          title={
            `已选择「${selected.templateName}」第`
            + `${selected.samplePageIndex + 1}页作为本次全页重构参考`
          }
        >
          <button
            type="button"
            onClick={openPicker}
            disabled={disabled}
            style={{
              minWidth: 0,
              padding: 0,
              border: 'none',
              background: 'transparent',
              color: '#C2410C',
              fontSize: 13,
              fontWeight: 700,
              cursor: disabled ? 'default' : 'pointer',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
            title="重新选择模板页"
          >
            🧩 {selected.templateName} · 第
            {selected.samplePageIndex + 1}页
          </button>

          <button
            type="button"
            onClick={onRemove}
            disabled={disabled}
            title="移除模板页参考"
            style={{
              padding: 0,
              border: 'none',
              background: 'transparent',
              color: '#C2410C',
              fontSize: 13,
              fontWeight: 800,
              cursor: disabled ? 'default' : 'pointer',
              lineHeight: 1,
              flexShrink: 0,
            }}
          >
            ✕
          </button>
        </span>
      ) : (
        <button
          type="button"
          onClick={openPicker}
          disabled={disabled}
          title="从可见模板中选择具体一页，作为本次全页重构的样式或交互逻辑参考"
          style={{
            padding: '8px 14px',
            borderRadius: 8,
            border: `1px dashed ${disabled ? C.border : '#EA580C'}`,
            background: '#FFF7ED',
            color: disabled ? '#9CA3AF' : '#C2410C',
            fontSize: 13,
            fontWeight: 700,
            cursor: disabled ? 'default' : 'pointer',
            whiteSpace: 'nowrap',
          }}
        >
          🧩 选择模板中的具体一页
        </button>
      )}

      {/* ==================== 全屏选择弹窗 ==================== */}
      {open && (
        <div
          onClick={() => setOpen(false)}
          style={{
            position: 'fixed',
            inset: 0,
            zIndex: 99980,
            padding: 24,
            background: 'rgba(15,23,42,0.68)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            backdropFilter: 'blur(4px)',
          }}
        >
          <div
            onClick={event => event.stopPropagation()}
            style={{
              width: 'min(1180px, 96vw)',
              height: 'min(780px, 92vh)',
              background: '#FFFFFF',
              borderRadius: 16,
              overflow: 'hidden',
              display: 'flex',
              flexDirection: 'column',
              boxShadow: '0 24px 80px rgba(0,0,0,0.34)',
            }}
          >
            {/* ---------- 头部 ---------- */}
            <div
              style={{
                padding: '16px 20px',
                borderBottom: `1px solid ${C.border}`,
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'flex-start',
                gap: 16,
                flexShrink: 0,
              }}
            >
              <div>
                <div
                  style={{
                    fontSize: 17,
                    fontWeight: 800,
                    color: C.textPrimary,
                  }}
                >
                  🧩 选择全页重构参考页
                </div>

                <div
                  style={{
                    marginTop: 4,
                    color: C.textSecondary,
                    fontSize: 12,
                    lineHeight: 1.6,
                  }}
                >
                  先选择模板，再选择其中具体一页。提交时只发送模板ID和页序号，
                  后端会重新读取并验证权限。
                </div>
              </div>

              <button
                type="button"
                onClick={() => setOpen(false)}
                style={{
                  width: 32,
                  height: 32,
                  borderRadius: 8,
                  border: 'none',
                  background: '#F3F4F6',
                  color: C.textSecondary,
                  fontSize: 18,
                  cursor: 'pointer',
                }}
              >
                ×
              </button>
            </div>

            {/* ---------- 主体 ---------- */}
            <div
              style={{
                flex: 1,
                minHeight: 0,
                display: 'flex',
                overflow: 'hidden',
              }}
            >
              {/* 左侧模板列表 */}
              <div
                style={{
                  width: 290,
                  flexShrink: 0,
                  borderRight: `1px solid ${C.border}`,
                  background: '#F9FAFB',
                  padding: 14,
                  overflowY: 'auto',
                }}
              >
                <div
                  style={{
                    padding: '2px 4px 10px',
                    color: C.textPrimary,
                    fontSize: 13,
                    fontWeight: 700,
                  }}
                >
                  可用模板
                </div>

                {loading ? (
                  <div
                    style={{
                      padding: '30px 8px',
                      color: C.textMuted,
                      fontSize: 13,
                      textAlign: 'center',
                    }}
                  >
                    ⏳ 正在加载模板...
                  </div>
                ) : templates.length === 0 ? (
                  <div
                    style={{
                      padding: '30px 8px',
                      color: C.textMuted,
                      fontSize: 12,
                      lineHeight: 1.7,
                    }}
                  >
                    暂无包含样例页的可用模板。可先在“风格模板”页面通过
                    AI提取或从课件保存模板。
                  </div>
                ) : (
                  <div
                    style={{
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 7,
                    }}
                  >
                    {templates.map(item => {
                      const template = item.template
                      const active = activeTemplateId === template.id
                      const badge = scopeColor(template)

                      return (
                        <button
                          key={template.id}
                          type="button"
                          onClick={() => chooseTemplate(item)}
                          style={{
                            width: '100%',
                            padding: '10px 11px',
                            borderRadius: 9,
                            border: `1.5px solid ${
                              active ? '#EA580C' : C.border
                            }`,
                            background: active ? '#FFF7ED' : '#FFFFFF',
                            textAlign: 'left',
                            cursor: 'pointer',
                          }}
                        >
                          <div
                            style={{
                              color: active ? '#C2410C' : C.textPrimary,
                              fontSize: 13,
                              fontWeight: 700,
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                            }}
                            title={template.name}
                          >
                            {template.name}
                          </div>

                          <div
                            style={{
                              marginTop: 6,
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'space-between',
                              gap: 8,
                            }}
                          >
                            <span
                              style={{
                                padding: '2px 7px',
                                borderRadius: 8,
                                color: badge.color,
                                background: badge.background,
                                fontSize: 10,
                                fontWeight: 700,
                              }}
                            >
                              {scopeLabel(template)}
                            </span>

                            <span
                              style={{
                                color: C.textMuted,
                                fontSize: 10,
                              }}
                            >
                              {item.pages.length}页
                            </span>
                          </div>
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>

              {/* 右侧页面选择和预览 */}
              <div
                style={{
                  flex: 1,
                  minWidth: 0,
                  padding: '16px 20px',
                  display: 'flex',
                  flexDirection: 'column',
                  overflow: 'hidden',
                }}
              >
                {!activeTemplate ? (
                  <div
                    style={{
                      flex: 1,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      color: C.textMuted,
                      fontSize: 14,
                    }}
                  >
                    请先在左侧选择一个模板
                  </div>
                ) : (
                  <>
                    <div
                      style={{
                        flexShrink: 0,
                        marginBottom: 12,
                      }}
                    >
                      <div
                        style={{
                          color: C.textPrimary,
                          fontSize: 14,
                          fontWeight: 800,
                        }}
                      >
                        {activeTemplate.template.name}
                      </div>

                      <div
                        style={{
                          marginTop: 3,
                          color: C.textSecondary,
                          fontSize: 11,
                        }}
                      >
                        请选择其中一页作为本次重构参考
                      </div>
                    </div>

                    <div
                      style={{
                        flexShrink: 0,
                        display: 'flex',
                        gap: 7,
                        flexWrap: 'wrap',
                        maxHeight: 104,
                        overflowY: 'auto',
                        paddingBottom: 10,
                      }}
                    >
                      {activeTemplate.pages.map((_, index) => (
                        <button
                          key={index}
                          type="button"
                          onClick={() => setActivePageIndex(index)}
                          style={{
                            padding: '6px 12px',
                            borderRadius: 14,
                            border: `1.5px solid ${
                              activePageIndex === index
                                ? '#EA580C'
                                : C.border
                            }`,
                            background: activePageIndex === index
                              ? '#FFF7ED'
                              : '#FFFFFF',
                            color: activePageIndex === index
                              ? '#C2410C'
                              : C.textSecondary,
                            fontSize: 12,
                            fontWeight: 700,
                            cursor: 'pointer',
                          }}
                        >
                          第 {index + 1} 页
                        </button>
                      ))}
                    </div>

                    <div
                      style={{
                        flex: 1,
                        minHeight: 0,
                        overflowY: 'auto',
                        paddingRight: 4,
                      }}
                    >
                      {activePageHTML ? (
                        <TemplateThumbAuto
                          sampleHTML={activePageHTML}
                          title={
                            `${activeTemplate.template.name}`
                            + `第${activePageIndex + 1}页`
                          }
                        />
                      ) : (
                        <div
                          style={{
                            height: '100%',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            color: C.textMuted,
                            fontSize: 13,
                          }}
                        >
                          当前页无可预览内容
                        </div>
                      )}
                    </div>

                    <div
                      style={{
                        flexShrink: 0,
                        marginTop: 10,
                        padding: '9px 12px',
                        borderRadius: 8,
                        background: '#FFF7ED',
                        color: '#9A3412',
                        fontSize: 11,
                        lineHeight: 1.65,
                      }}
                    >
                      提交重构前，请在修改要求中说明：
                      “参考这页的样式”“参考这页的交互逻辑”
                      或“样式和逻辑都参考”。当前页教学内容和导航栏不会从模板照抄。
                    </div>
                  </>
                )}
              </div>
            </div>

            {/* ---------- 底部 ---------- */}
            <div
              style={{
                padding: '13px 20px',
                borderTop: `1px solid ${C.border}`,
                display: 'flex',
                justifyContent: 'flex-end',
                gap: 10,
                flexShrink: 0,
              }}
            >
              <button
                type="button"
                onClick={() => setOpen(false)}
                style={{
                  padding: '8px 18px',
                  borderRadius: 8,
                  border: `1px solid ${C.border}`,
                  background: '#FFFFFF',
                  color: C.textSecondary,
                  fontSize: 13,
                  cursor: 'pointer',
                }}
              >
                取消
              </button>

              <button
                type="button"
                onClick={confirmSelection}
                disabled={!activeTemplate || !activePageHTML}
                style={{
                  padding: '8px 22px',
                  borderRadius: 8,
                  border: 'none',
                  background: activeTemplate && activePageHTML
                    ? 'linear-gradient(135deg, #EA580C, #F59E0B)'
                    : '#D1D5DB',
                  color: '#FFFFFF',
                  fontSize: 13,
                  fontWeight: 700,
                  cursor: activeTemplate && activePageHTML
                    ? 'pointer'
                    : 'not-allowed',
                }}
              >
                选用第 {activePageIndex + 1} 页
              </button>
            </div>

            {error && (
              <div
                style={{
                  position: 'absolute',
                  bottom: 72,
                  left: '50%',
                  transform: 'translateX(-50%)',
                  maxWidth: 620,
                  padding: '9px 14px',
                  borderRadius: 8,
                  background: '#FEE2E2',
                  color: '#B91C1C',
                  fontSize: 12,
                  boxShadow: '0 6px 24px rgba(0,0,0,0.16)',
                }}
              >
                ⚠️ {error}
              </div>
            )}
          </div>
        </div>
      )}
    </>
  )
}

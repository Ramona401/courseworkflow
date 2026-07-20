/**
 * TemplateExtractModal - AI提取风格模板弹窗
 *
 * 当前流程：
 *   1. 用户可粘贴1-20页完整HTML。
 *   2. 左侧管理页面，右侧只编辑当前选中页，避免大量文本框同时堆叠。
 *   3. 前端校验单页12万字符、全部页面60万字符。
 *   4. 前端先建立SSE连接，再触发异步提取。
 *   5. 后端完整保存全部原始页面，只抽取少量代表页交给AI分析视觉风格。
 *   6. AI不会重新生成或覆盖原始页面结构。
 */

import { useEffect, useRef, useState } from 'react'
import { useAuth } from '@/store/auth'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import {
  extractTemplateFromHTML,
  subscribeExtractSSE,
} from '@/api/coursewares'
import type { ExtractTemplateResponse } from '@/api/coursewares'

// ==================== 容量常量 ====================

const MAX_TEMPLATE_PAGES = 20
const MAX_SINGLE_PAGE_LENGTH = 120000
const MAX_TOTAL_HTML_LENGTH = 600000

// ==================== 颜色常量 ====================

const C = {
  primary: '#F59E0B',
  primaryDark: '#D97706',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  bgCard: '#FFFFFF',
  bgSoft: '#F9FAFB',
  danger: '#EF4444',
  success: '#10B981',
  refine: '#7C3AED',
}

// ==================== 组件Props ====================

interface Props {
  onClose: () => void
  onExtracted: (resp: ExtractTemplateResponse) => void
}

// ==================== 辅助函数 ====================

function formatCharacterCount(count: number): string {
  if (count < 1000) return `${count}`
  return `${(count / 1000).toFixed(1)}k`
}

function pageStatusLabel(content: string): {
  label: string
  color: string
  background: string
} {
  if (!content.trim()) {
    return {
      label: '空页',
      color: C.textMuted,
      background: '#F3F4F6',
    }
  }

  if (content.length > MAX_SINGLE_PAGE_LENGTH) {
    return {
      label: '超限',
      color: '#B91C1C',
      background: '#FEE2E2',
    }
  }

  return {
    label: '已填写',
    color: '#047857',
    background: '#D1FAE5',
  }
}


/**
 * 安全解析模板HTML页面草稿。
 *
 * 仅接受字符串数组，并再次限制为最多20页。
 * 存储损坏或版本不兼容时安全回退为空白单页。
 */
function parseTemplateHTMLPages(
  raw: string,
): string[] {
  if (!raw.trim()) {
    return ['']
  }

  try {
    const parsed = JSON.parse(raw)

    if (!Array.isArray(parsed)) {
      return ['']
    }

    const pages = parsed
      .filter(
        (page): page is string =>
          typeof page === 'string',
      )
      .slice(0, MAX_TEMPLATE_PAGES)

    return pages.length > 0
      ? pages
      : ['']
  } catch {
    return ['']
  }
}

// ==================== 主组件 ====================

export default function TemplateExtractModal({
  onClose,
  onExtracted,
}: Props) {
  const { user } = useAuth()

  /**
   * 模板HTML总长度最高可达60万字符。
   *
   * 提供刷新恢复和Ctrl/Command+Z能力，
   * 但只保留最多3份历史快照，
   * 避免大体积HTML产生几十份副本并占满sessionStorage。
   */
  const pagesDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'template-extract',
    resourceId: 'new-template',
    field: 'html-pages',
    initialValue: JSON.stringify(['']),
    maxHistory: 3,
    coalesceMs: 1200,
  })

  const pages = parseTemplateHTMLPages(
    pagesDraft.value,
  )

  const setPages = (
    nextPages: string[],
  ) => {
    pagesDraft.setValue(
      JSON.stringify(nextPages),
    )
  }

  /**
   * 当前编辑页序号单独保存。
   */
  const activePageDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'template-extract',
    resourceId: 'new-template',
    field: 'active-page-index',
    initialValue: '0',
    maxHistory: 12,
  })

  const parsedActivePageIndex =
    Number.parseInt(
      activePageDraft.value,
      10,
    )

  const activePageIndex =
    Number.isFinite(parsedActivePageIndex)
      ? Math.min(
          pages.length - 1,
          Math.max(
            0,
            parsedActivePageIndex,
          ),
        )
      : 0

  const setActivePageIndex = (
    index: number,
  ) => {
    activePageDraft.setValue(
      String(index),
    )
  }
  const [loading, setLoading] = useState(false)
  const [elapsedSec, setElapsedSec] = useState(0)
  const [progressMsg, setProgressMsg] = useState('')
  const [error, setError] = useState('')

  // SSE连接引用，组件卸载时安全关闭。
  const sseRef = useRef<{ close: () => void } | null>(null)

  const activePage = pages[activePageIndex] || ''
  const totalLen = pages.reduce((sum, page) => sum + page.length, 0)
  const validPages = pages.filter(page => page.trim().length > 0)
  const oversizedPageIndex = pages.findIndex(
    page => page.length > MAX_SINGLE_PAGE_LENGTH,
  )
  const totalOverLimit = totalLen > MAX_TOTAL_HTML_LENGTH

  // 加载期间每秒更新耗时。
  useEffect(() => {
    if (!loading) return

    setElapsedSec(0)
    const timer = setInterval(
      () => setElapsedSec(seconds => seconds + 1),
      1000,
    )

    return () => clearInterval(timer)
  }, [loading])

  // 组件卸载时关闭SSE连接。
  useEffect(() => {
    return () => {
      if (sseRef.current) {
        sseRef.current.close()
        sseRef.current = null
      }
    }
  }, [])

  // ==================== 页面管理 ====================

  const addPage = () => {
    if (loading || pages.length >= MAX_TEMPLATE_PAGES) return

    const nextPages = [...pages, '']
    setPages(nextPages)
    setActivePageIndex(nextPages.length - 1)
    setError('')
  }

  const removePage = (index: number) => {
    if (loading) return

    if (pages.length === 1) {
      setPages([''])
      setActivePageIndex(0)
      setError('')
      return
    }

    const nextPages = pages.filter((_, pageIndex) => pageIndex !== index)

    let nextActiveIndex = activePageIndex

    if (activePageIndex > index) {
      nextActiveIndex = activePageIndex - 1
    } else if (activePageIndex === index) {
      nextActiveIndex = Math.min(index, nextPages.length - 1)
    }

    setPages(nextPages)
    setActivePageIndex(nextActiveIndex)
    setError('')
  }

  const updateActivePage = (value: string) => {
    const nextPages = [...pages]
    nextPages[activePageIndex] = value
    setPages(nextPages)

    if (error) setError('')
  }

  // ==================== 提取任务 ====================

  const handleExtract = async () => {
    setError('')

    if (validPages.length === 0) {
      setError('请至少粘贴一页有效的HTML代码')
      return
    }

    if (pages.length > MAX_TEMPLATE_PAGES) {
      setError(`模板最多支持${MAX_TEMPLATE_PAGES}页`)
      return
    }

    if (oversizedPageIndex >= 0) {
      setActivePageIndex(oversizedPageIndex)
      setError(
        `第${oversizedPageIndex + 1}页共有` +
        `${pages[oversizedPageIndex].length}字符，` +
        `超过单页${MAX_SINGLE_PAGE_LENGTH}字符上限`,
      )
      return
    }

    if (totalOverLimit) {
      setError(
        `HTML总长度${totalLen}字符，` +
        `超过${MAX_TOTAL_HTML_LENGTH}字符上限`,
      )
      return
    }

    setLoading(true)
    setProgressMsg('正在建立SSE连接...')

    // 1. 先订阅SSE，避免任务过快启动时漏掉首个进度事件。
    const sse = subscribeExtractSSE({
      onStart: data => {
        setProgressMsg(data.message)
      },
      onProgress: data => {
        setProgressMsg(data.message)
      },
      onDone: data => {
        setLoading(false)
        sseRef.current = null

        // 后端已经完成分析与保存，清除已消费的大体积HTML草稿。
        pagesDraft.clear()
        activePageDraft.clear()

        onExtracted({
          template_id: data.template_id,
          suggested_name: data.suggested_name,
          suggested_desc: data.suggested_desc,
          suggested_category: data.suggested_category,
          extraction_notes: data.extraction_notes,
          message: data.message,
        })
      },
      onError: data => {
        setLoading(false)
        sseRef.current = null
        setError(data.message || 'AI提取失败，请稍后重试')
      },
    })

    sseRef.current = sse

    // 2. 再触发后端异步任务。
    try {
      await extractTemplateFromHTML(validPages, 'paste')
    } catch (requestError) {
      sse.close()
      sseRef.current = null
      setLoading(false)

      const message = (
        requestError as {
          response?: {
            data?: {
              message?: string
            }
          }
        }
      )?.response?.data?.message

      setError(message || 'AI提取请求发送失败')
    }
  }

  const canSubmit = (
    !loading
    && validPages.length > 0
    && oversizedPageIndex < 0
    && !totalOverLimit
  )

  // ==================== 渲染 ====================

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        width: '100vw',
        height: '100vh',
        background: 'rgba(0,0,0,0.62)',
        zIndex: 9999,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '24px',
        backdropFilter: 'blur(4px)',
      }}
      onClick={loading ? undefined : onClose}
    >
      <div
        style={{
          background: '#FFFFFF',
          borderRadius: '20px',
          width: '96%',
          maxWidth: '1180px',
          height: '92vh',
          minHeight: '650px',
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          boxShadow: '0 24px 80px rgba(0,0,0,0.28)',
        }}
        onClick={event => event.stopPropagation()}
      >
        {/* ==================== 头部 ==================== */}
        <div
          style={{
            padding: '19px 26px 16px',
            borderBottom: `1px solid ${C.border}`,
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-start',
            gap: '20px',
            flexShrink: 0,
          }}
        >
          <div>
            <div
              style={{
                fontSize: '20px',
                fontWeight: 700,
                color: C.textPrimary,
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
              }}
            >
              <span style={{ fontSize: '24px' }}>✨</span>
              AI提取风格模板
            </div>

            <div
              style={{
                fontSize: '13px',
                color: C.textSecondary,
                marginTop: '5px',
                lineHeight: 1.6,
              }}
            >
              可录入1-{MAX_TEMPLATE_PAGES}页完整HTML。系统会完整保存原始页面，
              AI只分析配色、字体、间距和视觉规律，不会重新生成页面结构。
            </div>
          </div>

          {!loading && (
            <button
              onClick={onClose}
              style={{
                background: 'none',
                border: 'none',
                fontSize: '27px',
                cursor: 'pointer',
                color: C.textMuted,
                lineHeight: 1,
                padding: '2px 6px',
              }}
            >
              ×
            </button>
          )}
        </div>

        {/* ==================== 主编辑区 ==================== */}
        <div
          style={{
            flex: 1,
            minHeight: 0,
            display: 'flex',
            overflow: 'hidden',
          }}
        >
          {/* ---------- 左侧页面列表 ---------- */}
          <div
            style={{
              width: '250px',
              flexShrink: 0,
              borderRight: `1px solid ${C.border}`,
              background: C.bgSoft,
              padding: '16px 14px',
              display: 'flex',
              flexDirection: 'column',
              minHeight: 0,
            }}
          >
            <div
              style={{
                padding: '0 4px 12px',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
            >
              <div
                style={{
                  fontSize: '13px',
                  fontWeight: 700,
                  color: C.textPrimary,
                }}
              >
                模板页面
              </div>

              <span
                style={{
                  fontSize: '11px',
                  color: C.textMuted,
                }}
              >
                {pages.length}/{MAX_TEMPLATE_PAGES}
              </span>
            </div>

            <div
              style={{
                flex: 1,
                minHeight: 0,
                overflowY: 'auto',
                display: 'flex',
                flexDirection: 'column',
                gap: '7px',
                paddingRight: '3px',
              }}
            >
              {pages.map((page, index) => {
                const selected = activePageIndex === index
                const status = pageStatusLabel(page)

                return (
                  <button
                    key={index}
                    type="button"
                    onClick={() => setActivePageIndex(index)}
                    disabled={loading}
                    style={{
                      width: '100%',
                      padding: '10px 11px',
                      borderRadius: '10px',
                      border: `1.5px solid ${
                        selected ? C.primary : C.border
                      }`,
                      background: selected ? '#FFF7ED' : '#FFFFFF',
                      cursor: loading ? 'not-allowed' : 'pointer',
                      textAlign: 'left',
                      display: 'flex',
                      flexDirection: 'column',
                      gap: '6px',
                      boxShadow: selected
                        ? '0 2px 8px rgba(245,158,11,0.12)'
                        : 'none',
                    }}
                  >
                    <div
                      style={{
                        width: '100%',
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        gap: '8px',
                      }}
                    >
                      <span
                        style={{
                          color: selected
                            ? C.primaryDark
                            : C.textPrimary,
                          fontSize: '13px',
                          fontWeight: 700,
                        }}
                      >
                        第 {index + 1} 页
                      </span>

                      <span
                        style={{
                          padding: '2px 7px',
                          borderRadius: '9px',
                          color: status.color,
                          background: status.background,
                          fontSize: '10px',
                          fontWeight: 700,
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {status.label}
                      </span>
                    </div>

                    <div
                      style={{
                        fontSize: '11px',
                        color: C.textMuted,
                      }}
                    >
                      {formatCharacterCount(page.length)} 字符
                    </div>
                  </button>
                )
              })}
            </div>

            <button
              type="button"
              onClick={addPage}
              disabled={loading || pages.length >= MAX_TEMPLATE_PAGES}
              style={{
                marginTop: '12px',
                padding: '9px 12px',
                borderRadius: '9px',
                border: `1px dashed ${
                  pages.length >= MAX_TEMPLATE_PAGES
                    ? C.border
                    : C.primary
                }`,
                background: 'transparent',
                color: pages.length >= MAX_TEMPLATE_PAGES
                  ? C.textMuted
                  : C.primaryDark,
                fontSize: '12px',
                fontWeight: 700,
                cursor: loading || pages.length >= MAX_TEMPLATE_PAGES
                  ? 'not-allowed'
                  : 'pointer',
              }}
            >
              {pages.length >= MAX_TEMPLATE_PAGES
                ? `已达到${MAX_TEMPLATE_PAGES}页上限`
                : '+ 增加一页'}
            </button>
          </div>

          {/* ---------- 右侧当前页编辑区 ---------- */}
          <div
            style={{
              flex: 1,
              minWidth: 0,
              padding: '18px 22px',
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                gap: '14px',
                marginBottom: '10px',
                flexShrink: 0,
              }}
            >
              <div>
                <div
                  style={{
                    fontSize: '14px',
                    fontWeight: 700,
                    color: C.textPrimary,
                  }}
                >
                  第 {activePageIndex + 1} 页HTML
                </div>

                <div
                  style={{
                    fontSize: '11px',
                    color: activePage.length > MAX_SINGLE_PAGE_LENGTH
                      ? C.danger
                      : C.textMuted,
                    marginTop: '3px',
                  }}
                >
                  当前 {activePage.length} / {MAX_SINGLE_PAGE_LENGTH} 字符
                </div>
              </div>

              <button
                type="button"
                onClick={() => removePage(activePageIndex)}
                disabled={loading}
                style={{
                  padding: '6px 12px',
                  borderRadius: '8px',
                  border: `1px solid ${C.border}`,
                  background: 'transparent',
                  color: C.danger,
                  fontSize: '12px',
                  fontWeight: 600,
                  cursor: loading ? 'not-allowed' : 'pointer',
                }}
              >
                {pages.length === 1 ? '清空本页' : '移除此页'}
              </button>
            </div>

            <textarea
              value={activePage}
              onChange={event => updateActivePage(event.target.value)}
              onKeyDown={event => {
                pagesDraft.handleKeyDown(event)
              }}
              disabled={loading}
              placeholder={
                activePageIndex === 0
                  ? '<!DOCTYPE html>\n<html>\n  <head>...</head>\n  <body>...</body>\n</html>'
                  : '在此粘贴第' +
                    (activePageIndex + 1) +
                    '页的完整HTML代码...'
              }
              style={{
                flex: 1,
                minHeight: '360px',
                width: '100%',
                padding: '14px 16px',
                borderRadius: '11px',
                border: `1.5px solid ${
                  activePage.length > MAX_SINGLE_PAGE_LENGTH
                    ? C.danger
                    : C.border
                }`,
                fontSize: '12px',
                fontFamily: 'Monaco, Consolas, monospace',
                outline: 'none',
                resize: 'none',
                lineHeight: 1.6,
                color: '#111827',
                background: loading ? C.bgSoft : '#FFFFFF',
                boxSizing: 'border-box',
              }}
            />

            {/* ---------- 统计与说明 ---------- */}
            <div
              style={{
                marginTop: '12px',
                display: 'grid',
                gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
                gap: '9px',
                flexShrink: 0,
              }}
            >
              <div
                style={{
                  padding: '9px 11px',
                  borderRadius: '9px',
                  background: '#F3F4F6',
                }}
              >
                <div
                  style={{
                    fontSize: '10px',
                    color: C.textMuted,
                  }}
                >
                  有效页面
                </div>
                <div
                  style={{
                    fontSize: '14px',
                    fontWeight: 700,
                    color: C.textPrimary,
                    marginTop: '2px',
                  }}
                >
                  {validPages.length} 页
                </div>
              </div>

              <div
                style={{
                  padding: '9px 11px',
                  borderRadius: '9px',
                  background: totalOverLimit ? '#FEE2E2' : '#F3F4F6',
                }}
              >
                <div
                  style={{
                    fontSize: '10px',
                    color: totalOverLimit ? '#B91C1C' : C.textMuted,
                  }}
                >
                  HTML总字符
                </div>
                <div
                  style={{
                    fontSize: '14px',
                    fontWeight: 700,
                    color: totalOverLimit ? '#B91C1C' : C.textPrimary,
                    marginTop: '2px',
                  }}
                >
                  {formatCharacterCount(totalLen)}
                  {' / '}
                  {formatCharacterCount(MAX_TOTAL_HTML_LENGTH)}
                </div>
              </div>

              <div
                style={{
                  padding: '9px 11px',
                  borderRadius: '9px',
                  background: '#EEF2FF',
                }}
              >
                <div
                  style={{
                    fontSize: '10px',
                    color: '#6366F1',
                  }}
                >
                  保存方式
                </div>
                <div
                  style={{
                    fontSize: '13px',
                    fontWeight: 700,
                    color: '#4F46E5',
                    marginTop: '2px',
                  }}
                >
                  原始页面保真
                </div>
              </div>
            </div>

            <div
              style={{
                marginTop: '10px',
                padding: '9px 12px',
                borderRadius: '9px',
                background: '#ECFDF5',
                color: '#047857',
                fontSize: '11px',
                lineHeight: 1.6,
                flexShrink: 0,
              }}
            >
              🔒 系统会把全部有效页面的原始HTML完整保存为模板母版。
              页面较多时仅抽取少量代表页供AI识别风格，不会用AI输出覆盖原始代码。
            </div>

            {/* ---------- SSE实时进度 ---------- */}
            {loading && progressMsg && (
              <div
                style={{
                  marginTop: '10px',
                  padding: '11px 14px',
                  borderRadius: '10px',
                  background: 'linear-gradient(135deg, #FEF3C7, #DBEAFE)',
                  border: `1px solid ${C.border}`,
                  display: 'flex',
                  alignItems: 'center',
                  gap: '12px',
                  flexShrink: 0,
                }}
              >
                <span
                  style={{
                    display: 'inline-block',
                    width: '17px',
                    height: '17px',
                    border: '2px solid rgba(124,58,237,0.24)',
                    borderTopColor: C.refine,
                    borderRadius: '50%',
                    animation: 'spin 0.8s linear infinite',
                    flexShrink: 0,
                  }}
                />

                <div>
                  <div
                    style={{
                      fontSize: '13px',
                      fontWeight: 700,
                      color: C.textPrimary,
                    }}
                  >
                    {progressMsg}
                  </div>

                  <div
                    style={{
                      fontSize: '11px',
                      color: C.textSecondary,
                      marginTop: '2px',
                    }}
                  >
                    已耗时 {Math.floor(elapsedSec / 60)} 分 {elapsedSec % 60} 秒
                  </div>
                </div>
              </div>
            )}

            {/* ---------- 错误提示 ---------- */}
            {error && (
              <div
                style={{
                  marginTop: '10px',
                  padding: '9px 13px',
                  borderRadius: '8px',
                  background: '#FEE2E2',
                  color: C.danger,
                  fontSize: '12px',
                  lineHeight: 1.6,
                  flexShrink: 0,
                }}
              >
                ⚠️ {error}
              </div>
            )}
          </div>
        </div>

        {/* ==================== 底部操作栏 ==================== */}
        <div
          style={{
            padding: '13px 24px',
            borderTop: `1px solid ${C.border}`,
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            gap: '12px',
            flexShrink: 0,
          }}
        >
          <div
            style={{
              color: C.textMuted,
              fontSize: '11px',
              lineHeight: 1.5,
            }}
          >
            HTML已自动保存，关闭或刷新后可恢复
            {' · '}
            最多保留3份大体积历史
            {' · '}
            Ctrl/Command+Z恢复误删
            <br />
            单页上限 {formatCharacterCount(MAX_SINGLE_PAGE_LENGTH)} 字符
            {' · '}
            总计上限 {formatCharacterCount(MAX_TOTAL_HTML_LENGTH)} 字符
            {' · '}
            最多 {MAX_TEMPLATE_PAGES} 页
          </div>

          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              gap: '10px',
            }}
          >
            <button
              onClick={onClose}
              disabled={loading}
              style={{
                padding: '9px 20px',
                borderRadius: '8px',
                border: `1px solid ${C.border}`,
                background: 'transparent',
                color: C.textSecondary,
                fontSize: '14px',
                cursor: loading ? 'not-allowed' : 'pointer',
              }}
            >
              取消
            </button>

            <button
              onClick={handleExtract}
              disabled={!canSubmit}
              style={{
                padding: '9px 24px',
                borderRadius: '8px',
                border: 'none',
                background: canSubmit
                  ? 'linear-gradient(135deg, #7C3AED, #F59E0B)'
                  : '#D1D5DB',
                color: '#FFFFFF',
                fontSize: '14px',
                fontWeight: 700,
                cursor: canSubmit ? 'pointer' : 'not-allowed',
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                boxShadow: canSubmit
                  ? '0 3px 12px rgba(124,58,237,0.24)'
                  : 'none',
              }}
            >
              {loading && (
                <span
                  style={{
                    display: 'inline-block',
                    width: '14px',
                    height: '14px',
                    border: '2px solid rgba(255,255,255,0.4)',
                    borderTopColor: '#FFFFFF',
                    borderRadius: '50%',
                    animation: 'spin 0.8s linear infinite',
                  }}
                />
              )}

              {loading
                ? `AI分析中 ${Math.floor(elapsedSec / 60)}分${elapsedSec % 60}秒`
                : `🚀 分析并保存${validPages.length}页模板`}
            </button>
          </div>
        </div>
      </div>

      <style>
        {`
          @keyframes spin {
            to {
              transform: rotate(360deg);
            }
          }
        `}
      </style>
    </div>
  )
}

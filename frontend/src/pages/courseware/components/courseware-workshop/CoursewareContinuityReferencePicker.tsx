/**
 * CoursewareContinuityReferencePicker.tsx
 *
 * 当前课件前序页面的连续性参考选择器。
 *
 * 安全协议：
 *   - 前端只提交页码数组，不提交页面HTML。
 *   - 后端重新从当前课件数据库读取最新页面。
 *   - 后端再次验证页码必须小于当前页，且一次最多5页。
 *
 * 使用目的：
 *   - 延续本课已形成的人物、叙事、卡片体系和交互步骤。
 *   - 当前页继续发展，不重复前页教学内容。
 *   - 可与一个模板具体页或一段代码收藏共同使用。
 */

import { useState } from 'react'
import { getCoursewarePages } from '@/api/coursewares'
import { TemplateThumbAuto } from '../TemplateThumb'
import { C } from './workshopConstants'

// ==================== 对外类型与协议 ====================

export interface CoursewareContinuityReferenceSelection {
  pageNumbers: number[]
  pageLabels: string[]
}

/**
 * 构建后端可识别的本课前页引用标记。
 *
 * 只包含页码，不包含HTML。
 */
export function buildCoursewareContinuityReferenceMarker(
  selection: CoursewareContinuityReferenceSelection,
): string {
  return (
    '<!-- TEDNA_COURSEWARE_PAGE_REFS '
    + JSON.stringify({
      page_numbers: [...selection.pageNumbers].sort(
        (left, right) => left - right,
      ),
    })
    + ' -->'
  )
}

interface Props {
  coursewareId: string
  currentPageNumber: number
  selected: CoursewareContinuityReferenceSelection | null
  onSelect: (
    selection: CoursewareContinuityReferenceSelection,
  ) => void
  onRemove: () => void
  disabled: boolean
}

// ==================== 内部类型 ====================

interface ContinuityPageOption {
  pageNumber: number
  title: string
  html: string
}

// ==================== 常量与辅助函数 ====================

const MAX_SELECTED_PAGES = 5

function formatPageNumbers(pageNumbers: number[]): string {
  return [...pageNumbers]
    .sort((left, right) => left - right)
    .map(pageNumber => `第${pageNumber}页`)
    .join('、')
}

function buildPageTitle(
  pageNumber: number,
  title: string,
): string {
  const cleanedTitle = title.trim()

  return cleanedTitle
    ? `第 ${pageNumber} 页 · ${cleanedTitle}`
    : `第 ${pageNumber} 页`
}

function normalizeLoadedPages(
  rawPages: unknown,
  currentPageNumber: number,
): ContinuityPageOption[] {
  if (!Array.isArray(rawPages)) return []

  const result: ContinuityPageOption[] = []

  for (const rawPage of rawPages) {
    if (
      typeof rawPage !== 'object'
      || rawPage === null
    ) {
      continue
    }

    const record = rawPage as Record<string, unknown>
    const pageNumber = Number(record.page_number)
    const html = typeof record.html_content === 'string'
      ? record.html_content.trim()
      : ''
    const title = typeof record.title === 'string'
      ? record.title.trim()
      : ''

    if (
      !Number.isInteger(pageNumber)
      || pageNumber <= 0
      || pageNumber >= currentPageNumber
      || !html
    ) {
      continue
    }

    result.push({
      pageNumber,
      title,
      html,
    })
  }

  return result.sort(
    (left, right) => left.pageNumber - right.pageNumber,
  )
}

// ==================== 主组件 ====================

export default function CoursewareContinuityReferencePicker({
  coursewareId,
  currentPageNumber,
  selected,
  onSelect,
  onRemove,
  disabled,
}: Props) {
  const [open, setOpen] = useState(false)
  const [pages, setPages] = useState<ContinuityPageOption[]>([])
  const [selectedNumbers, setSelectedNumbers] = useState<number[]>([])
  const [activePageNumber, setActivePageNumber] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const hasPreviousPages = currentPageNumber > 1

  const activePage = pages.find(
    page => page.pageNumber === activePageNumber,
  ) || null

  const openPicker = async () => {
    if (
      disabled
      || !coursewareId
      || !hasPreviousPages
    ) {
      return
    }

    setOpen(true)
    setLoading(true)
    setError('')

    try {
      const rawPages = await getCoursewarePages(coursewareId)
      const availablePages = normalizeLoadedPages(
        rawPages,
        currentPageNumber,
      )

      setPages(availablePages)

      const availablePageNumbers = new Set(
        availablePages.map(page => page.pageNumber),
      )

      const restoredSelection = (
        selected?.pageNumbers || []
      )
        .filter(pageNumber => availablePageNumbers.has(pageNumber))
        .slice(0, MAX_SELECTED_PAGES)
        .sort((left, right) => left - right)

      setSelectedNumbers(restoredSelection)

      const restoredActive = restoredSelection.length > 0
        ? restoredSelection[restoredSelection.length - 1]
        : availablePages.length > 0
          ? availablePages[availablePages.length - 1].pageNumber
          : 0

      setActivePageNumber(restoredActive)

      if (availablePages.length === 0) {
        setError(
          '当前页之前还没有可用的已生成页面。',
        )
      }
    } catch (loadError) {
      setPages([])
      setSelectedNumbers([])
      setActivePageNumber(0)
      setError(
        '加载本课前页失败：'
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

  const togglePage = (page: ContinuityPageOption) => {
    setActivePageNumber(page.pageNumber)
    setError('')

    if (selectedNumbers.includes(page.pageNumber)) {
      setSelectedNumbers(
        selectedNumbers.filter(
          pageNumber => pageNumber !== page.pageNumber,
        ),
      )
      return
    }

    if (selectedNumbers.length >= MAX_SELECTED_PAGES) {
      setError(
        `一次最多选择${MAX_SELECTED_PAGES}个前序页面。`,
      )
      return
    }

    setSelectedNumbers(
      [...selectedNumbers, page.pageNumber].sort(
        (left, right) => left - right,
      ),
    )
  }

  const selectRecentThree = () => {
    const recentPages = pages.slice(-3)
    const pageNumbers = recentPages.map(
      page => page.pageNumber,
    )

    setSelectedNumbers(pageNumbers)
    setError('')

    if (recentPages.length > 0) {
      setActivePageNumber(
        recentPages[recentPages.length - 1].pageNumber,
      )
    }
  }

  const clearSelection = () => {
    setSelectedNumbers([])
    setError('')
  }

  const confirmSelection = () => {
    if (selectedNumbers.length === 0) {
      setError('请至少选择一个本课前页。')
      return
    }

    const normalizedPageNumbers = [...selectedNumbers].sort(
      (left, right) => left - right,
    )

    const pageLabels = normalizedPageNumbers.map(pageNumber => {
      const page = pages.find(
        item => item.pageNumber === pageNumber,
      )

      return page
        ? buildPageTitle(page.pageNumber, page.title)
        : `第 ${pageNumber} 页`
    })

    onSelect({
      pageNumbers: normalizedPageNumbers,
      pageLabels,
    })

    setOpen(false)
  }

  const selectedSummary = selected
    ? formatPageNumbers(selected.pageNumbers)
    : ''

  return (
    <>
      {/* ==================== 工具条按钮或已选芯片 ==================== */}
      {selected && selected.pageNumbers.length > 0 ? (
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            maxWidth: 420,
            padding: '8px 12px',
            borderRadius: 8,
            border: '1px solid #0F766E',
            background: '#F0FDFA',
            color: '#0F766E',
            fontSize: 13,
            fontWeight: 600,
            whiteSpace: 'nowrap',
          }}
          title={
            `已选择${selectedSummary}作为本次全页重构的连续性参考`
          }
        >
          <button
            type="button"
            onClick={openPicker}
            disabled={disabled}
            title="重新选择本课前页"
            style={{
              minWidth: 0,
              padding: 0,
              border: 'none',
              background: 'transparent',
              color: '#0F766E',
              fontSize: 13,
              fontWeight: 700,
              cursor: disabled ? 'default' : 'pointer',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            🔗 本课前页：{selectedSummary}
          </button>

          <button
            type="button"
            onClick={onRemove}
            disabled={disabled}
            title="移除本课前页参考"
            style={{
              padding: 0,
              border: 'none',
              background: 'transparent',
              color: '#0F766E',
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
          disabled={
            disabled
            || !coursewareId
            || !hasPreviousPages
          }
          title={
            !hasPreviousPages
              ? '当前是第1页，没有可引用的前序页面'
              : '选择当前课件前面已完成的页面，延续人物、布局、叙事和交互步骤'
          }
          style={{
            padding: '8px 14px',
            borderRadius: 8,
            border: `1px dashed ${
              disabled || !hasPreviousPages
                ? C.border
                : '#0F766E'
            }`,
            background: '#F0FDFA',
            color: disabled || !hasPreviousPages
              ? '#9CA3AF'
              : '#0F766E',
            fontSize: 13,
            fontWeight: 700,
            cursor: disabled || !hasPreviousPages
              ? 'default'
              : 'pointer',
            whiteSpace: 'nowrap',
          }}
        >
          🔗 参考本课前页
        </button>
      )}

      {/* ==================== 全屏多选弹窗 ==================== */}
      {open && (
        <div
          onClick={() => setOpen(false)}
          style={{
            position: 'fixed',
            inset: 0,
            zIndex: 99982,
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
              position: 'relative',
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
                  🔗 选择本课前页作为连续性参考
                </div>

                <div
                  style={{
                    marginTop: 4,
                    color: C.textSecondary,
                    fontSize: 12,
                    lineHeight: 1.65,
                  }}
                >
                  可任意勾选当前第
                  {currentPageNumber}
                  页之前的1至5页。系统只提交页码，
                  后端会读取这些页面的最新版本。
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
              {/* 左侧页面列表 */}
              <div
                style={{
                  width: 340,
                  flexShrink: 0,
                  borderRight: `1px solid ${C.border}`,
                  background: '#F9FAFB',
                  padding: 14,
                  display: 'flex',
                  flexDirection: 'column',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    flexShrink: 0,
                    display: 'flex',
                    alignItems: 'center',
                    gap: 7,
                    padding: '2px 3px 10px',
                  }}
                >
                  <span
                    style={{
                      color: C.textPrimary,
                      fontSize: 13,
                      fontWeight: 800,
                    }}
                  >
                    可用前序页面
                  </span>

                  <span
                    style={{
                      padding: '2px 7px',
                      borderRadius: 10,
                      background: selectedNumbers.length > 0
                        ? '#CCFBF1'
                        : '#E5E7EB',
                      color: selectedNumbers.length > 0
                        ? '#0F766E'
                        : '#6B7280',
                      fontSize: 10,
                      fontWeight: 700,
                    }}
                  >
                    已选 {selectedNumbers.length}/{MAX_SELECTED_PAGES}
                  </span>
                </div>

                <div
                  style={{
                    flexShrink: 0,
                    display: 'flex',
                    gap: 7,
                    marginBottom: 10,
                  }}
                >
                  <button
                    type="button"
                    onClick={selectRecentThree}
                    disabled={pages.length === 0}
                    style={{
                      flex: 1,
                      padding: '6px 8px',
                      borderRadius: 7,
                      border: '1px solid #0F766E',
                      background: '#FFFFFF',
                      color: '#0F766E',
                      fontSize: 11,
                      fontWeight: 700,
                      cursor: pages.length > 0
                        ? 'pointer'
                        : 'default',
                    }}
                  >
                    最近3页
                  </button>

                  <button
                    type="button"
                    onClick={clearSelection}
                    disabled={selectedNumbers.length === 0}
                    style={{
                      flex: 1,
                      padding: '6px 8px',
                      borderRadius: 7,
                      border: `1px solid ${C.border}`,
                      background: '#FFFFFF',
                      color: selectedNumbers.length > 0
                        ? C.textSecondary
                        : '#9CA3AF',
                      fontSize: 11,
                      cursor: selectedNumbers.length > 0
                        ? 'pointer'
                        : 'default',
                    }}
                  >
                    清空选择
                  </button>
                </div>

                {loading ? (
                  <div
                    style={{
                      padding: '32px 8px',
                      color: C.textMuted,
                      fontSize: 13,
                      textAlign: 'center',
                    }}
                  >
                    ⏳ 正在读取本课页面...
                  </div>
                ) : pages.length === 0 ? (
                  <div
                    style={{
                      padding: '32px 8px',
                      color: C.textMuted,
                      fontSize: 12,
                      lineHeight: 1.75,
                    }}
                  >
                    当前页之前暂无已经生成HTML的页面。
                    请先完成前序页面，再进行连续性开发。
                  </div>
                ) : (
                  <div
                    style={{
                      flex: 1,
                      minHeight: 0,
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 7,
                      overflowY: 'auto',
                      paddingRight: 3,
                    }}
                  >
                    {pages.map(page => {
                      const checked = selectedNumbers.includes(
                        page.pageNumber,
                      )
                      const active = activePageNumber ===
                        page.pageNumber

                      return (
                        <button
                          key={page.pageNumber}
                          type="button"
                          onClick={() => togglePage(page)}
                          style={{
                            width: '100%',
                            padding: '10px 11px',
                            borderRadius: 9,
                            border: `1.5px solid ${
                              active
                                ? '#0F766E'
                                : checked
                                  ? '#5EEAD4'
                                  : C.border
                            }`,
                            background: checked
                              ? '#F0FDFA'
                              : '#FFFFFF',
                            textAlign: 'left',
                            cursor: 'pointer',
                          }}
                        >
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'flex-start',
                              gap: 9,
                            }}
                          >
                            <span
                              style={{
                                width: 19,
                                height: 19,
                                marginTop: 1,
                                borderRadius: 5,
                                border: `1.5px solid ${
                                  checked
                                    ? '#0F766E'
                                    : '#9CA3AF'
                                }`,
                                background: checked
                                  ? '#0F766E'
                                  : '#FFFFFF',
                                color: '#FFFFFF',
                                display: 'inline-flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                fontSize: 12,
                                fontWeight: 800,
                                flexShrink: 0,
                              }}
                            >
                              {checked ? '✓' : ''}
                            </span>

                            <div
                              style={{
                                minWidth: 0,
                                flex: 1,
                              }}
                            >
                              <div
                                style={{
                                  color: checked
                                    ? '#0F766E'
                                    : C.textPrimary,
                                  fontSize: 13,
                                  fontWeight: 700,
                                }}
                              >
                                第 {page.pageNumber} 页
                              </div>

                              <div
                                style={{
                                  marginTop: 3,
                                  color: C.textSecondary,
                                  fontSize: 11,
                                  overflow: 'hidden',
                                  textOverflow: 'ellipsis',
                                  whiteSpace: 'nowrap',
                                }}
                                title={page.title || '未命名页面'}
                              >
                                {page.title || '未命名页面'}
                              </div>
                            </div>

                            <span
                              style={{
                                padding: '2px 6px',
                                borderRadius: 8,
                                background: active
                                  ? '#CCFBF1'
                                  : '#F3F4F6',
                                color: active
                                  ? '#0F766E'
                                  : C.textMuted,
                                fontSize: 9,
                                fontWeight: 700,
                                flexShrink: 0,
                              }}
                            >
                              {active ? '预览中' : '查看'}
                            </span>
                          </div>
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>

              {/* 右侧预览 */}
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
                {!activePage ? (
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
                    请在左侧选择或查看一个前序页面
                  </div>
                ) : (
                  <>
                    <div
                      style={{
                        flexShrink: 0,
                        marginBottom: 10,
                      }}
                    >
                      <div
                        style={{
                          color: C.textPrimary,
                          fontSize: 14,
                          fontWeight: 800,
                        }}
                      >
                        {buildPageTitle(
                          activePage.pageNumber,
                          activePage.title,
                        )}
                      </div>

                      <div
                        style={{
                          marginTop: 3,
                          color: C.textSecondary,
                          fontSize: 11,
                        }}
                      >
                        此处只做安全预览；正式提交时后端读取数据库最新版本。
                      </div>
                    </div>

                    <div
                      style={{
                        flex: 1,
                        minHeight: 0,
                        overflowY: 'auto',
                        paddingRight: 4,
                      }}
                    >
                      <TemplateThumbAuto
                        sampleHTML={activePage.html}
                        title={
                          `本课第${activePage.pageNumber}页连续性参考`
                        }
                      />
                    </div>

                    <div
                      style={{
                        flexShrink: 0,
                        marginTop: 10,
                        padding: '9px 12px',
                        borderRadius: 8,
                        background: '#F0FDFA',
                        color: '#115E59',
                        fontSize: 11,
                        lineHeight: 1.7,
                      }}
                    >
                      建议在重构要求中说明需要延续的内容，例如：
                      “延续这些页面的人物、卡片体系和逐步点击逻辑，
                      把本页开发为下一阶段任务。”
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
                alignItems: 'center',
                gap: 10,
                flexShrink: 0,
              }}
            >
              <div
                style={{
                  flex: 1,
                  minWidth: 0,
                  color: selectedNumbers.length > 0
                    ? '#0F766E'
                    : C.textMuted,
                  fontSize: 11,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
                title={
                  selectedNumbers.length > 0
                    ? formatPageNumbers(selectedNumbers)
                    : ''
                }
              >
                {selectedNumbers.length > 0
                  ? `已选择：${formatPageNumbers(selectedNumbers)}`
                  : '尚未选择页面'}
              </div>

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
                disabled={selectedNumbers.length === 0}
                style={{
                  padding: '8px 22px',
                  borderRadius: 8,
                  border: 'none',
                  background: selectedNumbers.length > 0
                    ? 'linear-gradient(135deg, #0F766E, #14B8A6)'
                    : '#D1D5DB',
                  color: '#FFFFFF',
                  fontSize: 13,
                  fontWeight: 700,
                  cursor: selectedNumbers.length > 0
                    ? 'pointer'
                    : 'not-allowed',
                }}
              >
                使用所选 {selectedNumbers.length} 页
              </button>
            </div>

            {error && (
              <div
                style={{
                  position: 'absolute',
                  bottom: 72,
                  left: '50%',
                  transform: 'translateX(-50%)',
                  maxWidth: 680,
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

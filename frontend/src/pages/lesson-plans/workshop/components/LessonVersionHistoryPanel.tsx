/**
 * LessonVersionHistoryPanel.tsx — 教案正文版本历史共享面板
 *
 * 功能：
 *   1. 展示当前版本号和最近50份修改前快照。
 *   2. 按人工编辑、AI修改、导入、恢复和系统修改展示来源。
 *   3. 点击历史版本后加载完整正文。
 *   4. 历史正文与当前正文左右对比，并高亮不同文本行。
 *   5. 支持预览历史版本和一键恢复。
 *
 * 对话模式与专家模式通过LessonDocumentEditor共用本面板。
 */

import { useCallback, useEffect, useMemo, useState } from 'react'
import { renderMarkdown } from '@/pages/lesson-plans/plan-detail/components/planDetailConstants'
import {
  getLessonPlanContentVersion,
  getLessonPlanContentVersions,
  restoreLessonPlanContentVersion,
  type LessonPlanContentRestoreResponse,
  type LessonPlanContentVersion,
  type LessonPlanContentVersionListItem,
  type LessonPlanVersionSource,
} from '@/api/lesson-plan-versions'

interface LessonVersionHistoryPanelProps {
  open: boolean
  planID: string
  currentContent: string
  currentVersion: number
  restoreDisabled?: boolean
  restoreDisabledReason?: string
  onClose: () => void
  onRestored: (result: LessonPlanContentRestoreResponse) => void
}

interface MarkedLine {
  text: string
  changed: boolean
}

const SOURCE_META: Record<
  LessonPlanVersionSource,
  {
    label: string
    icon: string
    background: string
    color: string
  }
> = {
  manual: {
    label: '人工编辑',
    icon: '✏️',
    background: 'rgba(16,185,129,0.10)',
    color: '#047857',
  },
  ai: {
    label: 'AI修改',
    icon: '🤖',
    background: 'rgba(79,123,232,0.10)',
    color: '#365FB8',
  },
  import: {
    label: '导入教案',
    icon: '📥',
    background: 'rgba(139,92,246,0.10)',
    color: '#7C3AED',
  },
  restore: {
    label: '版本恢复',
    icon: '↩️',
    background: 'rgba(245,158,11,0.12)',
    color: '#B45309',
  },
  system: {
    label: '系统更新',
    icon: '⚙️',
    background: 'rgba(107,114,128,0.10)',
    color: '#4B5563',
  },
}

function formatDateTime(value: string): string {
  try {
    const date = new Date(value)
    return new Intl.DateTimeFormat('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }).format(date)
  } catch {
    return value
  }
}

/**
 * 使用LCS标记两份正文中未匹配的文本行。
 *
 * 正常教案通常只有几十至数百行，可以精确计算。
 * 极长正文超过计算预算时退化为同位置比较，避免浏览器卡顿。
 */
function markChangedLines(
  historicalText: string,
  currentText: string,
): {
  historical: MarkedLine[]
  current: MarkedLine[]
  changedCount: number
} {
  const historicalLines = historicalText.split('\n')
  const currentLines = currentText.split('\n')

  const historicalSame = new Set<number>()
  const currentSame = new Set<number>()

  const cellCount =
    historicalLines.length * currentLines.length

  if (cellCount <= 60000) {
    const matrix = Array.from(
      { length: historicalLines.length + 1 },
      () => new Uint16Array(currentLines.length + 1),
    )

    for (
      let historicalIndex = historicalLines.length - 1;
      historicalIndex >= 0;
      historicalIndex -= 1
    ) {
      for (
        let currentIndex = currentLines.length - 1;
        currentIndex >= 0;
        currentIndex -= 1
      ) {
        if (
          historicalLines[historicalIndex] ===
          currentLines[currentIndex]
        ) {
          matrix[historicalIndex][currentIndex] =
            matrix[historicalIndex + 1][currentIndex + 1] + 1
        } else {
          matrix[historicalIndex][currentIndex] = Math.max(
            matrix[historicalIndex + 1][currentIndex],
            matrix[historicalIndex][currentIndex + 1],
          )
        }
      }
    }

    let historicalIndex = 0
    let currentIndex = 0

    while (
      historicalIndex < historicalLines.length &&
      currentIndex < currentLines.length
    ) {
      if (
        historicalLines[historicalIndex] ===
        currentLines[currentIndex]
      ) {
        historicalSame.add(historicalIndex)
        currentSame.add(currentIndex)
        historicalIndex += 1
        currentIndex += 1
        continue
      }

      if (
        matrix[historicalIndex + 1][currentIndex] >=
        matrix[historicalIndex][currentIndex + 1]
      ) {
        historicalIndex += 1
      } else {
        currentIndex += 1
      }
    }
  } else {
    const maxLength = Math.max(
      historicalLines.length,
      currentLines.length,
    )

    for (let index = 0; index < maxLength; index += 1) {
      if (
        historicalLines[index] !== undefined &&
        historicalLines[index] === currentLines[index]
      ) {
        historicalSame.add(index)
        currentSame.add(index)
      }
    }
  }

  const historical = historicalLines.map(
    (text, index): MarkedLine => ({
      text,
      changed: !historicalSame.has(index),
    }),
  )

  const current = currentLines.map(
    (text, index): MarkedLine => ({
      text,
      changed: !currentSame.has(index),
    }),
  )

  return {
    historical,
    current,
    changedCount:
      historical.filter(line => line.changed).length +
      current.filter(line => line.changed).length,
  }
}

export default function LessonVersionHistoryPanel({
  open,
  planID,
  currentContent,
  currentVersion,
  restoreDisabled = false,
  restoreDisabledReason = '',
  onClose,
  onRestored,
}: LessonVersionHistoryPanelProps) {
  const [versions, setVersions] = useState<
    LessonPlanContentVersionListItem[]
  >([])
  const [total, setTotal] = useState(0)
  const [serverCurrentVersion, setServerCurrentVersion] =
    useState(currentVersion)

  const [loadingList, setLoadingList] = useState(false)
  const [listError, setListError] = useState('')

  const [selectedID, setSelectedID] = useState('')
  const [selectedVersion, setSelectedVersion] =
    useState<LessonPlanContentVersion | null>(null)
  const [loadingDetail, setLoadingDetail] = useState(false)
  const [detailError, setDetailError] = useState('')

  const [viewMode, setViewMode] =
    useState<'compare' | 'preview'>('compare')
  const [restoring, setRestoring] = useState(false)
  const [restoreMessage, setRestoreMessage] = useState('')

  const loadVersions = useCallback(async () => {
    if (!planID) return

    setLoadingList(true)
    setListError('')

    try {
      const response = await getLessonPlanContentVersions(
        planID,
        {
          limit: 50,
          offset: 0,
        },
      )

      setVersions(response.versions || [])
      setTotal(response.total || 0)
      setServerCurrentVersion(
        response.current_version || currentVersion,
      )
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : '版本记录加载失败'
      setListError(message)
      setVersions([])
    } finally {
      setLoadingList(false)
    }
  }, [planID, currentVersion])

  useEffect(() => {
    if (!open) return

    setSelectedID('')
    setSelectedVersion(null)
    setDetailError('')
    setRestoreMessage('')
    setViewMode('compare')

    void loadVersions()
  }, [open, loadVersions])

  useEffect(() => {
    setServerCurrentVersion(currentVersion)
  }, [currentVersion])

  const handleSelectVersion = async (
    item: LessonPlanContentVersionListItem,
  ) => {
    setSelectedID(item.id)
    setSelectedVersion(null)
    setDetailError('')
    setLoadingDetail(true)
    setRestoreMessage('')

    try {
      const detail = await getLessonPlanContentVersion(
        planID,
        item.id,
      )
      setSelectedVersion(detail)
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : '历史版本正文加载失败'
      setDetailError(message)
    } finally {
      setLoadingDetail(false)
    }
  }

  const diff = useMemo(() => {
    if (!selectedVersion) return null

    return markChangedLines(
      selectedVersion.content_markdown || '',
      currentContent || '',
    )
  }, [selectedVersion, currentContent])

  const handleRestore = async () => {
    if (!selectedVersion || restoring) return

    if (restoreDisabled) {
      setRestoreMessage(
        restoreDisabledReason ||
          '当前状态暂时不允许恢复历史版本',
      )
      return
    }

    const confirmed = window.confirm(
      `确定恢复历史版本 v${selectedVersion.version_number} 吗？\n\n` +
        `当前版本 v${serverCurrentVersion} 会先自动保存到版本记录中，` +
        '因此本次恢复可以再次撤回。\n\n' +
        '恢复只影响教案标题、正文和课时时长，不会回退审核或发布状态。',
    )

    if (!confirmed) return

    setRestoring(true)
    setRestoreMessage('')

    try {
      const result =
        await restoreLessonPlanContentVersion(
          planID,
          selectedVersion.id,
        )

      onRestored(result)
      setServerCurrentVersion(result.current_version)
      setRestoreMessage(
        `已恢复v${result.restored_from_version}，当前生成新版本v${result.current_version}`,
      )
      setSelectedID('')
      setSelectedVersion(null)
      await loadVersions()
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : '恢复历史版本失败'
      setRestoreMessage(`恢复失败：${message}`)
    } finally {
      setRestoring(false)
    }
  }

  if (!open) return null

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 12000,
        background: 'rgba(15,23,42,0.52)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '24px',
      }}
      onClick={event => {
        if (
          event.target === event.currentTarget &&
          !restoring
        ) {
          onClose()
        }
      }}
    >
      <div
        style={{
          width: 'min(1180px, 96vw)',
          height: 'min(820px, 90vh)',
          background: '#FFFFFF',
          borderRadius: '16px',
          boxShadow: '0 24px 80px rgba(15,23,42,0.28)',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
        onClick={event => event.stopPropagation()}
      >
        {/* 顶部标题 */}
        <div
          style={{
            padding: '16px 20px',
            borderBottom: '1px solid #E5E7EB',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '16px',
            flexShrink: 0,
          }}
        >
          <div>
            <div
              style={{
                fontSize: '16px',
                fontWeight: 700,
                color: '#1F2937',
              }}
            >
              🕘 教案版本记录
            </div>
            <div
              style={{
                marginTop: '4px',
                fontSize: '12px',
                color: '#6B7280',
              }}
            >
              当前版本 v{serverCurrentVersion} ·
              共 {total} 份历史快照 · 最多保留50份
            </div>
          </div>

          <button
            onClick={onClose}
            disabled={restoring}
            style={{
              width: '34px',
              height: '34px',
              borderRadius: '50%',
              border: '1px solid #E5E7EB',
              background: '#FFFFFF',
              color: '#6B7280',
              fontSize: '18px',
              cursor: restoring
                ? 'not-allowed'
                : 'pointer',
            }}
          >
            ×
          </button>
        </div>

        {restoreMessage && (
          <div
            style={{
              margin: '10px 16px 0',
              padding: '9px 12px',
              borderRadius: '8px',
              background: restoreMessage.startsWith('恢复失败')
                ? '#FEF2F2'
                : '#ECFDF5',
              border: restoreMessage.startsWith('恢复失败')
                ? '1px solid #FECACA'
                : '1px solid #A7F3D0',
              color: restoreMessage.startsWith('恢复失败')
                ? '#DC2626'
                : '#047857',
              fontSize: '12px',
              lineHeight: 1.6,
              flexShrink: 0,
            }}
          >
            {restoreMessage}
          </div>
        )}

        <div
          style={{
            flex: 1,
            minHeight: 0,
            display: 'flex',
          }}
        >
          {/* 左侧版本列表 */}
          <div
            style={{
              width: '300px',
              flexShrink: 0,
              borderRight: '1px solid #E5E7EB',
              display: 'flex',
              flexDirection: 'column',
              minHeight: 0,
              background: '#FAFBFC',
            }}
          >
            <div
              style={{
                padding: '12px 14px',
                borderBottom: '1px solid #E5E7EB',
                fontSize: '12px',
                fontWeight: 700,
                color: '#374151',
              }}
            >
              历史快照
            </div>

            <div
              style={{
                flex: 1,
                overflowY: 'auto',
                padding: '10px',
              }}
            >
              {loadingList && (
                <div
                  style={{
                    padding: '36px 12px',
                    textAlign: 'center',
                    color: '#9CA3AF',
                    fontSize: '13px',
                  }}
                >
                  正在加载版本记录…
                </div>
              )}

              {!loadingList && listError && (
                <div
                  style={{
                    padding: '18px 12px',
                    borderRadius: '8px',
                    background: '#FEF2F2',
                    color: '#DC2626',
                    fontSize: '12px',
                    lineHeight: 1.6,
                  }}
                >
                  ⚠️ {listError}
                  <button
                    onClick={() => void loadVersions()}
                    style={{
                      display: 'block',
                      marginTop: '8px',
                      padding: '5px 10px',
                      borderRadius: '6px',
                      border: '1px solid #FCA5A5',
                      background: '#FFFFFF',
                      color: '#DC2626',
                      cursor: 'pointer',
                    }}
                  >
                    重新加载
                  </button>
                </div>
              )}

              {!loadingList &&
                !listError &&
                versions.length === 0 && (
                  <div
                    style={{
                      padding: '48px 12px',
                      textAlign: 'center',
                      color: '#9CA3AF',
                      fontSize: '13px',
                      lineHeight: 1.8,
                    }}
                  >
                    <div
                      style={{
                        fontSize: '30px',
                        marginBottom: '8px',
                      }}
                    >
                      🕘
                    </div>
                    暂无历史版本
                    <br />
                    第一次修改正文后会自动记录
                  </div>
                )}

              {!loadingList &&
                versions.map(item => {
                  const selected = selectedID === item.id
                  const source =
                    SOURCE_META[item.change_source] ||
                    SOURCE_META.system

                  return (
                    <button
                      key={item.id}
                      onClick={() =>
                        void handleSelectVersion(item)
                      }
                      style={{
                        width: '100%',
                        display: 'block',
                        textAlign: 'left',
                        padding: '11px 12px',
                        marginBottom: '8px',
                        borderRadius: '9px',
                        border: selected
                          ? '1.5px solid #4F7BE8'
                          : '1px solid #E5E7EB',
                        background: selected
                          ? 'rgba(79,123,232,0.07)'
                          : '#FFFFFF',
                        cursor: 'pointer',
                      }}
                    >
                      <div
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                          gap: '8px',
                        }}
                      >
                        <span
                          style={{
                            fontSize: '13px',
                            fontWeight: 700,
                            color: selected
                              ? '#365FB8'
                              : '#1F2937',
                          }}
                        >
                          v{item.version_number}
                        </span>

                        <span
                          style={{
                            padding: '2px 7px',
                            borderRadius: '10px',
                            background: source.background,
                            color: source.color,
                            fontSize: '10px',
                            fontWeight: 600,
                          }}
                        >
                          {source.icon} {source.label}
                        </span>
                      </div>

                      <div
                        style={{
                          marginTop: '6px',
                          fontSize: '11px',
                          color: '#6B7280',
                        }}
                      >
                        {formatDateTime(item.created_at)}
                      </div>

                      <div
                        style={{
                          marginTop: '5px',
                          fontSize: '11px',
                          color: '#4B5563',
                          lineHeight: 1.5,
                          overflow: 'hidden',
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                        }}
                      >
                        {item.change_summary ||
                          item.content_preview ||
                          '正文历史快照'}
                      </div>

                      <div
                        style={{
                          marginTop: '6px',
                          fontSize: '10px',
                          color: '#9CA3AF',
                        }}
                      >
                        {item.character_count}字
                        {item.changed_by_name
                          ? ` · ${item.changed_by_name}`
                          : ''}
                      </div>
                    </button>
                  )
                })}
            </div>
          </div>

          {/* 右侧详情和对比 */}
          <div
            style={{
              flex: 1,
              minWidth: 0,
              display: 'flex',
              flexDirection: 'column',
              minHeight: 0,
            }}
          >
            {!selectedID && (
              <div
                style={{
                  flex: 1,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#9CA3AF',
                  textAlign: 'center',
                  lineHeight: 1.8,
                }}
              >
                <div
                  style={{
                    fontSize: '40px',
                    marginBottom: '12px',
                  }}
                >
                  📑
                </div>
                从左侧选择一个历史版本
                <br />
                查看与当前正文的差异
              </div>
            )}

            {selectedID && loadingDetail && (
              <div
                style={{
                  flex: 1,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#9CA3AF',
                  fontSize: '13px',
                }}
              >
                正在加载历史正文…
              </div>
            )}

            {selectedID && !loadingDetail && detailError && (
              <div
                style={{
                  margin: '20px',
                  padding: '14px 16px',
                  borderRadius: '8px',
                  background: '#FEF2F2',
                  border: '1px solid #FECACA',
                  color: '#DC2626',
                  fontSize: '13px',
                }}
              >
                ⚠️ {detailError}
              </div>
            )}

            {selectedVersion &&
              !loadingDetail &&
              !detailError && (
                <>
                  <div
                    style={{
                      padding: '11px 16px',
                      borderBottom: '1px solid #E5E7EB',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      gap: '12px',
                      flexShrink: 0,
                    }}
                  >
                    <div>
                      <div
                        style={{
                          fontSize: '13px',
                          fontWeight: 700,
                          color: '#1F2937',
                        }}
                      >
                        历史v{selectedVersion.version_number}
                        {' '}对比当前v{serverCurrentVersion}
                      </div>
                      <div
                        style={{
                          marginTop: '3px',
                          fontSize: '11px',
                          color: '#9CA3AF',
                        }}
                      >
                        {diff
                          ? `共标记${diff.changedCount}行差异`
                          : ''}
                      </div>
                    </div>

                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '7px',
                      }}
                    >
                      <button
                        onClick={() =>
                          setViewMode('compare')
                        }
                        style={{
                          padding: '5px 10px',
                          borderRadius: '7px',
                          border:
                            viewMode === 'compare'
                              ? '1px solid #4F7BE8'
                              : '1px solid #D1D5DB',
                          background:
                            viewMode === 'compare'
                              ? 'rgba(79,123,232,0.08)'
                              : '#FFFFFF',
                          color:
                            viewMode === 'compare'
                              ? '#365FB8'
                              : '#6B7280',
                          fontSize: '12px',
                          cursor: 'pointer',
                        }}
                      >
                        ↔ 差异对比
                      </button>

                      <button
                        onClick={() =>
                          setViewMode('preview')
                        }
                        style={{
                          padding: '5px 10px',
                          borderRadius: '7px',
                          border:
                            viewMode === 'preview'
                              ? '1px solid #4F7BE8'
                              : '1px solid #D1D5DB',
                          background:
                            viewMode === 'preview'
                              ? 'rgba(79,123,232,0.08)'
                              : '#FFFFFF',
                          color:
                            viewMode === 'preview'
                              ? '#365FB8'
                              : '#6B7280',
                          fontSize: '12px',
                          cursor: 'pointer',
                        }}
                      >
                        👁 历史预览
                      </button>
                    </div>
                  </div>

                  <div
                    style={{
                      flex: 1,
                      minHeight: 0,
                      overflow: 'hidden',
                    }}
                  >
                    {viewMode === 'compare' && diff && (
                      <div
                        style={{
                          height: '100%',
                          display: 'grid',
                          gridTemplateColumns: '1fr 1fr',
                          minHeight: 0,
                        }}
                      >
                        <DiffColumn
                          title={`历史 v${selectedVersion.version_number}`}
                          lines={diff.historical}
                          type="historical"
                        />
                        <DiffColumn
                          title={`当前 v${serverCurrentVersion}`}
                          lines={diff.current}
                          type="current"
                        />
                      </div>
                    )}

                    {viewMode === 'preview' && (
                      <div
                        style={{
                          height: '100%',
                          overflowY: 'auto',
                          padding: '20px 24px',
                          boxSizing: 'border-box',
                          fontSize: '14px',
                          lineHeight: 1.85,
                        }}
                      >
                        {renderMarkdown(
                          selectedVersion.content_markdown,
                        )}
                      </div>
                    )}
                  </div>

                  <div
                    style={{
                      padding: '12px 16px',
                      borderTop: '1px solid #E5E7EB',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      gap: '12px',
                      flexShrink: 0,
                    }}
                  >
                    <div
                      style={{
                        fontSize: '11px',
                        color: '#6B7280',
                        lineHeight: 1.5,
                      }}
                    >
                      恢复前会自动保存当前v
                      {serverCurrentVersion}，本操作可撤回
                    </div>

                    <button
                      onClick={() => void handleRestore()}
                      disabled={restoring || restoreDisabled}
                      title={
                        restoreDisabled
                          ? restoreDisabledReason
                          : `恢复历史v${selectedVersion.version_number}`
                      }
                      style={{
                        padding: '8px 18px',
                        borderRadius: '8px',
                        border: 'none',
                        background:
                          restoring || restoreDisabled
                            ? '#E5E7EB'
                            : 'linear-gradient(135deg, #F59E0B, #FBBF24)',
                        color:
                          restoring || restoreDisabled
                            ? '#9CA3AF'
                            : '#FFFFFF',
                        fontSize: '13px',
                        fontWeight: 700,
                        cursor:
                          restoring || restoreDisabled
                            ? 'not-allowed'
                            : 'pointer',
                      }}
                    >
                      {restoring
                        ? '恢复中…'
                        : `↩ 恢复v${selectedVersion.version_number}`}
                    </button>
                  </div>
                </>
              )}
          </div>
        </div>
      </div>
    </div>
  )
}

function DiffColumn({
  title,
  lines,
  type,
}: {
  title: string
  lines: MarkedLine[]
  type: 'historical' | 'current'
}) {
  const changedBackground =
    type === 'historical'
      ? 'rgba(239,68,68,0.10)'
      : 'rgba(16,185,129,0.11)'

  const changedBorder =
    type === 'historical'
      ? '#FCA5A5'
      : '#6EE7B7'

  const symbol = type === 'historical' ? '−' : '+'

  return (
    <div
      style={{
        minWidth: 0,
        minHeight: 0,
        display: 'flex',
        flexDirection: 'column',
        borderRight:
          type === 'historical'
            ? '1px solid #E5E7EB'
            : 'none',
      }}
    >
      <div
        style={{
          padding: '8px 12px',
          background:
            type === 'historical'
              ? '#FEF2F2'
              : '#ECFDF5',
          color:
            type === 'historical'
              ? '#B91C1C'
              : '#047857',
          fontSize: '12px',
          fontWeight: 700,
          borderBottom: '1px solid #E5E7EB',
          flexShrink: 0,
        }}
      >
        {title}
      </div>

      <div
        style={{
          flex: 1,
          overflow: 'auto',
          padding: '8px 0',
          background: '#FFFFFF',
          fontFamily:
            'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
          fontSize: '12px',
          lineHeight: 1.65,
        }}
      >
        {lines.map((line, index) => (
          <div
            key={`${index}-${line.text.slice(0, 20)}`}
            style={{
              display: 'grid',
              gridTemplateColumns: '44px 18px 1fr',
              minHeight: '21px',
              padding: '1px 8px 1px 0',
              background: line.changed
                ? changedBackground
                : 'transparent',
              borderLeft: line.changed
                ? `3px solid ${changedBorder}`
                : '3px solid transparent',
            }}
          >
            <span
              style={{
                paddingRight: '8px',
                textAlign: 'right',
                color: '#9CA3AF',
                userSelect: 'none',
              }}
            >
              {index + 1}
            </span>
            <span
              style={{
                color: line.changed
                  ? type === 'historical'
                    ? '#DC2626'
                    : '#059669'
                  : 'transparent',
                fontWeight: 700,
                userSelect: 'none',
              }}
            >
              {line.changed ? symbol : '·'}
            </span>
            <span
              style={{
                whiteSpace: 'pre-wrap',
                overflowWrap: 'anywhere',
                color: '#374151',
              }}
            >
              {line.text || ' '}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

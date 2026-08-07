/**
 * AddPageModal.tsx — 课件新增页面弹窗。
 *
 * 第一栏改为AI需求讨论：先把目标、内容和前后页衔接讨论清楚，
 * 老师点击独立按钮后才正式创建并生成页面。
 *
 * 第二栏保留粘贴HTML流程，只新增“插入为第几页”参数；
 * 导入、画布归一、导航替换和背景补注仍走原有后端链路。
 */
import { useState } from 'react'
import type {
  CSSProperties,
  ChangeEvent,
  MouseEvent,
} from 'react'
import { importPageHtml } from '@/api/coursewares'
import { addCWPageAtPosition } from '@/api/courseware-add-page-discussion'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import { useAuth } from '@/store/auth'
import AddPageDiscussionPanel from './AddPageDiscussionPanel'
import { C } from './workshopConstants'

interface Props {
  coursewareId: string
  currentPageCount: number
  onDone: (newPageNumber: number) => void
  onClose: () => void
}

interface AddPageDraftForm {
  mode: 'ai' | 'paste'
  insertAt: number
  pasteTitle: string
  pasteHtml: string
}

type PastePhase = 'form' | 'creating' | 'importing' | 'done' | 'error'

function divBalanceCheck(source: string): string {
  const open = (source.match(/<div\b/gi) || []).length
  const close = (source.match(/<\/div>/gi) || []).length
  return open === close
    ? ''
    : `<div> 开标签 ${open} 个、</div> 闭标签 ${close} 个，数量不一致，页面可能残缺或变形`
}

function createInitialForm(currentPageCount: number): AddPageDraftForm {
  const insertAt = currentPageCount + 1
  return {
    mode: 'ai',
    insertAt,
    pasteTitle: `第 ${insertAt} 页`,
    pasteHtml: '',
  }
}

function clampInsertAt(value: number, currentPageCount: number): number {
  return Math.min(
    currentPageCount + 1,
    Math.max(1, Math.round(value || currentPageCount + 1)),
  )
}

function parseAddPageDraftForm(
  raw: string,
  fallback: AddPageDraftForm,
  currentPageCount: number,
): AddPageDraftForm {
  if (!raw.trim()) return { ...fallback }

  try {
    const parsed = JSON.parse(raw) as Partial<AddPageDraftForm> & {
      title?: string
      pasteHtml?: string
    }
    const insertAt = clampInsertAt(
      typeof parsed.insertAt === 'number' ? parsed.insertAt : fallback.insertAt,
      currentPageCount,
    )

    return {
      mode: parsed.mode === 'paste' ? 'paste' : 'ai',
      insertAt,
      pasteTitle: typeof parsed.pasteTitle === 'string'
        ? parsed.pasteTitle
        : typeof parsed.title === 'string'
          ? parsed.title
          : `第 ${insertAt} 页`,
      pasteHtml: typeof parsed.pasteHtml === 'string' ? parsed.pasteHtml : '',
    }
  } catch {
    return { ...fallback }
  }
}

const labelStyle: CSSProperties = {
  display: 'block',
  marginBottom: 5,
  color: '#6B7280',
  fontSize: 13,
  fontWeight: 600,
}

const inputStyle: CSSProperties = {
  width: '100%',
  boxSizing: 'border-box',
  padding: '9px 12px',
  border: '1px solid #D1D5DB',
  borderRadius: 9,
  fontSize: 14,
  outline: 'none',
}

export default function AddPageModal({
  coursewareId,
  currentPageCount,
  onDone,
  onClose,
}: Props) {
  const { user } = useAuth()
  const initialForm = createInitialForm(currentPageCount)
  const formDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'courseware-add-page',
    resourceId: [
      coursewareId,
      `next-page-${currentPageCount + 1}`,
    ].join('|'),
    field: 'form',
    initialValue: JSON.stringify(initialForm),
    maxHistory: 8,
    coalesceMs: 1000,
  })
  const form = parseAddPageDraftForm(
    formDraft.value,
    initialForm,
    currentPageCount,
  )

  const [pastePhase, setPastePhase] = useState<PastePhase>('form')
  const [errorMessage, setErrorMessage] = useState('')
  const [createdPageNumber, setCreatedPageNumber] = useState(0)
  const [aiBusy, setAIBusy] = useState(false)

  const updateForm = (patch: Partial<AddPageDraftForm>) => {
    formDraft.setValue(previous => JSON.stringify({
      ...parseAddPageDraftForm(previous, initialForm, currentPageCount),
      ...patch,
    }))
  }

  const switchMode = (mode: 'ai' | 'paste') => {
    updateForm({ mode })
    setErrorMessage('')
    if (pastePhase === 'error') setPastePhase('form')
  }

  const updateInsertAt = (value: number) => {
    const next = clampInsertAt(value, currentPageCount)
    const defaultTitlePattern = /^第\s*\d+\s*页$/
    updateForm({
      insertAt: next,
      pasteTitle: defaultTitlePattern.test(form.pasteTitle.trim())
        ? `第 ${next} 页`
        : form.pasteTitle,
    })
    setErrorMessage('')
  }

  const finish = (pageNumber: number) => {
    formDraft.clear()
    onDone(pageNumber)
  }

  const submitPaste = async () => {
    if (!form.pasteTitle.trim()) {
      setErrorMessage('请填写页面标题')
      return
    }
    if (!form.pasteHtml.trim()) {
      setErrorMessage('请粘贴页面HTML代码')
      return
    }

    const warning = divBalanceCheck(form.pasteHtml)
    if (warning && !window.confirm(
      `⚠️ 结构自检提示：${warning}。\n\n仍要导入吗？\n（导入后如显示异常，可在源码编辑里修改，或删除该页重新粘贴）`,
    )) {
      return
    }

    setPastePhase('creating')
    setErrorMessage('')

    try {
      const page = await addCWPageAtPosition(coursewareId, {
        insert_at: form.insertAt,
        title: form.pasteTitle.trim(),
        content_summary: '（由粘贴的HTML代码创建）',
        interaction_type: 'static',
        visual_format: 'text_heavy',
        estimated_complexity: 3,
      })
      const pageNumber = page.page_number
      setCreatedPageNumber(pageNumber)
      setPastePhase('importing')

      try {
        await importPageHtml(coursewareId, pageNumber, form.pasteHtml)
        setPastePhase('done')
        window.setTimeout(() => finish(pageNumber), 700)
      } catch (error: unknown) {
        setPastePhase('done')
        setErrorMessage(
          '页面已创建，但HTML导入失败：'
          + (error instanceof Error ? error.message : '未知错误')
          + '。可选中该页用“编辑源码”重试粘贴。',
        )
        window.setTimeout(() => finish(pageNumber), 2200)
      }
    } catch (error: unknown) {
      setPastePhase('error')
      setErrorMessage(error instanceof Error ? error.message : '创建页面失败')
    }
  }

  const pasteBusy = pastePhase === 'creating' || pastePhase === 'importing'
  const isBusy = aiBusy || pasteBusy
  const overlay: CSSProperties = {
    position: 'fixed',
    inset: 0,
    zIndex: 99990,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 20,
    background: 'rgba(17,24,39,0.48)',
  }
  const panel: CSSProperties = {
    width: '100%',
    maxWidth: 1040,
    maxHeight: '92vh',
    overflowY: 'auto',
    padding: '22px 24px 24px',
    borderRadius: 18,
    background: '#fff',
    boxShadow: '0 24px 70px rgba(0,0,0,0.2)',
  }

  return (
    <div style={overlay} onClick={isBusy ? undefined : onClose}>
      <div style={panel} onClick={(event: MouseEvent<HTMLDivElement>) => event.stopPropagation()}>
        <div style={{
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent: 'space-between',
          gap: 16,
          marginBottom: 14,
        }}>
          <div>
            <h3 style={{ margin: 0, color: '#1F2937', fontSize: 20, fontWeight: 800 }}>
              ＋ 添加课件页面
            </h3>
            <div style={{ marginTop: 4, color: '#6B7280', fontSize: 12 }}>
              先确认插入位置，再选择AI讨论生成或粘贴HTML。
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={isBusy}
            aria-label="关闭"
            style={{
              width: 34,
              height: 34,
              border: '1px solid #E5E7EB',
              borderRadius: 9,
              background: '#fff',
              color: '#6B7280',
              fontSize: 18,
              cursor: isBusy ? 'not-allowed' : 'pointer',
            }}
          >
            ×
          </button>
        </div>

        <div style={{
          display: 'grid',
          gridTemplateColumns: '190px minmax(0, 1fr)',
          alignItems: 'center',
          gap: 12,
          marginBottom: 12,
          padding: '11px 13px',
          border: '1px solid #E5E7EB',
          borderRadius: 11,
          background: '#F9FAFB',
        }}>
          <label htmlFor="courseware-add-page-insert-at" style={{
            color: '#374151',
            fontSize: 13,
            fontWeight: 800,
          }}>
            插入为第几页
          </label>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <input
              id="courseware-add-page-insert-at"
              type="number"
              min={1}
              max={currentPageCount + 1}
              value={form.insertAt}
              disabled={isBusy}
              onChange={(event: ChangeEvent<HTMLInputElement>) => updateInsertAt(Number(event.target.value))}
              style={{ ...inputStyle, width: 120 }}
            />
            <span style={{ color: '#6B7280', fontSize: 12 }}>
              可选第 1 至第 {currentPageCount + 1} 页；原位置及之后页面会自动后移。
            </span>
          </div>
        </div>

        <div style={{
          display: 'flex',
          gap: 6,
          marginBottom: 14,
          padding: 4,
          borderRadius: 10,
          background: '#F3F4F6',
        }}>
          <button
            type="button"
            onClick={() => switchMode('ai')}
            disabled={isBusy}
            style={{
              flex: 1,
              padding: '9px 12px',
              border: 'none',
              borderRadius: 8,
              background: form.mode === 'ai' ? '#fff' : 'transparent',
              color: form.mode === 'ai' ? C.primary : '#6B7280',
              boxShadow: form.mode === 'ai' ? '0 1px 3px rgba(0,0,0,0.12)' : 'none',
              fontSize: 13,
              fontWeight: 800,
              cursor: isBusy ? 'not-allowed' : 'pointer',
            }}
          >
            🤖 AI讨论生成
          </button>
          <button
            type="button"
            onClick={() => switchMode('paste')}
            disabled={isBusy}
            style={{
              flex: 1,
              padding: '9px 12px',
              border: 'none',
              borderRadius: 8,
              background: form.mode === 'paste' ? '#fff' : 'transparent',
              color: form.mode === 'paste' ? '#059669' : '#6B7280',
              boxShadow: form.mode === 'paste' ? '0 1px 3px rgba(0,0,0,0.12)' : 'none',
              fontSize: 13,
              fontWeight: 800,
              cursor: isBusy ? 'not-allowed' : 'pointer',
            }}
          >
            📋 粘贴HTML
          </button>
        </div>

        {form.mode === 'ai' ? (
          <AddPageDiscussionPanel
            coursewareId={coursewareId}
            insertAt={form.insertAt}
            onBusyChange={setAIBusy}
            onDone={pageNumber => {
              formDraft.clear()
              onDone(pageNumber)
            }}
          />
        ) : (
          <div style={{
            minHeight: 470,
            display: 'flex',
            flexDirection: 'column',
            justifyContent: pastePhase === 'form' || pastePhase === 'error'
              ? 'flex-start'
              : 'center',
          }}>
            {(pastePhase === 'form' || pastePhase === 'error') && (
              <div style={{ display: 'grid', gap: 14 }}>
                <div>
                  <label style={labelStyle}>页面标题 *</label>
                  <input
                    value={form.pasteTitle}
                    onChange={(event: ChangeEvent<HTMLInputElement>) => updateForm({ pasteTitle: event.target.value })}
                    onKeyDown={formDraft.handleKeyDown}
                    placeholder="例如：实验操作步骤"
                    style={inputStyle}
                    autoFocus
                  />
                </div>

                <div>
                  <label style={labelStyle}>
                    页面HTML代码 *
                    <span style={{ marginLeft: 5, color: '#9CA3AF', fontSize: 12 }}>
                      整页粘贴，从最外层 &lt;div 到结尾
                    </span>
                  </label>
                  <textarea
                    value={form.pasteHtml}
                    onChange={(event: ChangeEvent<HTMLTextAreaElement>) => updateForm({ pasteHtml: event.target.value })}
                    onKeyDown={formDraft.handleKeyDown}
                    rows={12}
                    spellCheck={false}
                    placeholder={'把完整页面HTML代码粘贴到这里…\n支持平台内复制的页面源码或外部1920×1080单页HTML。'}
                    style={{
                      ...inputStyle,
                      minHeight: 260,
                      resize: 'vertical',
                      background: '#1E1E1E',
                      color: '#D4D4D4',
                      borderColor: '#374151',
                      fontFamily: 'Monaco, Consolas, "Courier New", monospace',
                      fontSize: 12,
                      lineHeight: 1.6,
                    }}
                  />
                </div>

                <div style={{
                  padding: '10px 12px',
                  borderRadius: 8,
                  background: '#F0FDF4',
                  color: '#166534',
                  fontSize: 12,
                  lineHeight: 1.7,
                }}>
                  导入时系统仍按原流程处理：统一1920×1080画布；平台内页面替换成本课件导航并重编号；
                  补注当前背景。外部HTML没有导航栏属于正常情况。
                </div>

                {errorMessage && (
                  <div style={{
                    padding: '10px 13px',
                    borderRadius: 8,
                    background: '#FEE2E2',
                    color: '#B91C1C',
                    fontSize: 13,
                  }}>
                    {errorMessage}
                  </div>
                )}

                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 12,
                }}>
                  <span style={{ color: '#9CA3AF', fontSize: 11 }}>
                    表单和HTML已自动保存 · 创建失败不会清除 · Ctrl/Command+Z可恢复误删
                  </span>
                  <div style={{ display: 'flex', gap: 9 }}>
                    <button
                      type="button"
                      onClick={onClose}
                      style={{
                        padding: '8px 18px',
                        border: '1px solid #E5E7EB',
                        borderRadius: 8,
                        background: '#fff',
                        color: '#6B7280',
                        cursor: 'pointer',
                      }}
                    >
                      取消
                    </button>
                    <button
                      type="button"
                      onClick={() => void submitPaste()}
                      style={{
                        padding: '8px 20px',
                        border: 'none',
                        borderRadius: 8,
                        background: '#059669',
                        color: '#fff',
                        fontSize: 14,
                        fontWeight: 700,
                        cursor: 'pointer',
                      }}
                    >
                      创建并导入第 {form.insertAt} 页
                    </button>
                  </div>
                </div>
              </div>
            )}

            {(pastePhase === 'creating' || pastePhase === 'importing' || pastePhase === 'done') && (
              <div style={{ textAlign: 'center', padding: '36px 0' }}>
                <div style={{ marginBottom: 12, fontSize: 36 }}>
                  {pastePhase === 'creating' ? '📝' : pastePhase === 'importing' ? '⚙️' : '✅'}
                </div>
                <div style={{
                  color: pastePhase === 'done' ? '#059669' : '#4B5563',
                  fontSize: 16,
                  fontWeight: 700,
                }}>
                  {pastePhase === 'creating'
                    ? `正在第 ${form.insertAt} 页位置创建页面…`
                    : pastePhase === 'importing'
                      ? `P${createdPageNumber} 已创建，正在导入HTML…`
                      : `P${createdPageNumber} 已导入完成`}
                </div>
                {errorMessage && (
                  <div style={{
                    maxWidth: 620,
                    margin: '12px auto 0',
                    padding: '10px 13px',
                    borderRadius: 8,
                    background: '#FFFBEB',
                    color: '#B45309',
                    fontSize: 13,
                    lineHeight: 1.65,
                  }}>
                    {errorMessage}
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        <style>{`
          @media (max-width: 820px) {
            .add-page-discussion-grid {
              grid-template-columns: 1fr !important;
            }
          }
        `}</style>
      </div>
    </div>
  )
}

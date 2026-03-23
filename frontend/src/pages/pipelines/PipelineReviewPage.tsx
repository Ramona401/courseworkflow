/**
 * Pipeline审核页面（P4.5-E 重构版 v3.2）
 *
 * 核心设计：
 * 1. 左侧列表按页码顺序排列（保持课程逻辑连贯性）
 * 2. 右侧预览根据操作类型自动适配：keep原版/modify左右对比/create生成版/merge标签切换/delete删除标记
 * 3. 每页显示修改理由面板（change_reason）
 * 4. 全屏预览支持左右翻页 + 原版/修改版切换按钮
 * 5. 所有页面（含keep）都逐个审核
 * 6. 全屏修改理由弹窗（P4.5-E-1）：点击按钮弹出模态框显示完整修改理由
 * 7. 全屏AI快修功能（P4.5-E-2）：审核员输入修改指令，AI基于当前HTML修复
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  getPipelineDetail,
  getGeneratedPages,
  updatePageDecision,
  finalizePipeline,
  type PipelineDetailResponse,
  type GeneratedPageFull,
  type UpdatePageDecisionRequest,
  aiFixPage,
} from '@/api/pipelines'
import {
  ArrowLeft, RefreshCw, Check, X, Edit3, CheckCircle, Send,
  Maximize2, Minimize2, ChevronLeft, ChevronRight, FileText, Columns, Wand2, Loader,
} from 'lucide-react'

// ==================== 常量 ====================

const OP_COLORS: Record<string, string> = {
  keep: '#34c759', modify: '#007aff', create: '#af52de',
  merge: '#ff9500', delete: '#ff3b30',
}
const OP_NAMES: Record<string, string> = {
  keep: '保留', modify: '修改', create: '新建', merge: '合并', delete: '删除',
}
const DECISION_COLORS: Record<string, string> = {
  approve: '#34c759', reject: '#ff3b30', edit: '#ff9500', pending: '#c7c7cc',
}
const DECISION_NAMES: Record<string, string> = {
  approve: '已采用', reject: '已拒绝', edit: '已编辑', pending: '待决策',
}

/** 解析merge_sources JSON字符串为数字数组 */
function parseMergeSources(ms: string): number[] {
  try {
    if (ms && ms !== 'null') return JSON.parse(ms)
  } catch { /* ignore */ }
  return []
}

// ==================== 主组件 ====================

export default function PipelineReviewPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [pipeline, setPipeline] = useState<PipelineDetailResponse | null>(null)
  const [pages, setPages] = useState<GeneratedPageFull[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [selectedIdx, setSelectedIdx] = useState(0)
  const [mergeSourceTab, setMergeSourceTab] = useState(0)
  const [finalizing, setFinalizing] = useState(false)
  const [deciding, setDeciding] = useState(false)
  const [editingHTML, setEditingHTML] = useState(false)
  const [editContent, setEditContent] = useState('')
  const [fullscreen, setFullscreen] = useState(false)
  const [reasonExpanded, setReasonExpanded] = useState(true)

  const currentPage = pages[selectedIdx] || null
  const totalPages = pages.length
  const decidedPages = pages.filter(p => p.decision !== 'pending').length
  const allDecided = totalPages > 0 && decidedPages === totalPages

  // 操作类型统计
  const opStats: Record<string, number> = {}
  for (const p of pages) {
    const op = p.operation || 'keep'
    opStats[op] = (opStats[op] || 0) + 1
  }

  /** 加载数据 */
  const loadData = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError('')
    try {
      const [pd, pg] = await Promise.all([getPipelineDetail(id), getGeneratedPages(id)])
      setPipeline(pd)
      setPages(pg || [])
    } catch (e: any) {
      setError(e.message || '加载失败')
    }
    setLoading(false)
  }, [id])

  useEffect(() => { loadData() }, [loadData])

  /** ESC退出全屏，全屏时←→翻页 */
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && fullscreen) { setFullscreen(false); return }
      if (fullscreen) {
        if (e.key === 'ArrowLeft' && selectedIdx > 0) { setSelectedIdx(i => i - 1); setMergeSourceTab(0) }
        if (e.key === 'ArrowRight' && selectedIdx < pages.length - 1) { setSelectedIdx(i => i + 1); setMergeSourceTab(0) }
      }
    }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [fullscreen, selectedIdx, pages.length])

  const selectPage = (idx: number) => { setSelectedIdx(idx); setEditingHTML(false); setMergeSourceTab(0) }

  const handleDecision = async (decision: 'approve' | 'reject' | 'edit', finalHTML?: string) => {
    if (!id || !currentPage) return
    setDeciding(true)
    try {
      const req: UpdatePageDecisionRequest = { decision }
      if (decision === 'edit' && finalHTML) req.final_html = finalHTML
      await updatePageDecision(id, currentPage.page_number, req)
      setPages(prev => prev.map(p =>
        p.page_number === currentPage.page_number ? { ...p, decision, final_html: finalHTML || p.final_html } : p
      ))
      setEditingHTML(false)
      const nextIdx = pages.findIndex((p, i) => i > selectedIdx && p.decision === 'pending')
      if (nextIdx >= 0) setSelectedIdx(nextIdx)
    } catch (e: any) { alert('决策失败: ' + (e.message || '未知错误')) }
    setDeciding(false)
  }

  const handleFinalize = async () => {
    if (!id || !allDecided) return
    if (!confirm('确认定稿归档？')) return
    setFinalizing(true)
    try { await finalizePipeline(id); alert('定稿成功！'); navigate('/pipelines') }
    catch (e: any) { alert('定稿失败: ' + (e.message || '未知错误')) }
    setFinalizing(false)
  }

  const startEdit = () => {
    if (!currentPage) return
    setEditContent(currentPage.final_html || currentPage.generated_html || '')
    setEditingHTML(true)
  }

  const btn: React.CSSProperties = {
    padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 6,
  }

  if (loading) return <div style={{ textAlign: 'center', padding: 60, color: '#8e8e93' }}>加载中...</div>
  if (error) return (
    <div style={{ textAlign: 'center', padding: 60 }}>
      <div style={{ color: '#ff3b30', fontSize: 14, marginBottom: 12 }}>{error}</div>
      <button style={btn} onClick={() => navigate(-1)}><ArrowLeft size={14} /> 返回</button>
    </div>
  )
  if (!pipeline) return null

  return (
    <div style={{ height: 'calc(100vh - 80px)', display: 'flex', flexDirection: 'column' }}>
      {/* ===== 顶部栏 ===== */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 12, padding: '12px 0',
        borderBottom: '1px solid rgba(0,0,0,0.06)', flexShrink: 0,
      }}>
        <button style={btn} onClick={() => navigate('/pipelines/' + id)}>
          <ArrowLeft size={14} /> 返回详情
        </button>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 16, fontWeight: 700, color: '#1c1c1e' }}>
            审核: {pipeline.course_code} — {pipeline.course_name}
          </div>
          <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 2 }}>
            {decidedPages}/{totalPages} 页已决策
            {allDecided && <span style={{ color: '#34c759', marginLeft: 8 }}>✓ 全部完成</span>}
          </div>
        </div>
        <button style={btn} onClick={loadData}><RefreshCw size={14} /> 刷新</button>
        <button
          style={{
            ...btn, background: allDecided ? '#34c759' : '#e5e5ea',
            color: allDecided ? '#fff' : '#aeaeb2', border: 'none',
            cursor: allDecided ? 'pointer' : 'not-allowed',
          }}
          onClick={handleFinalize} disabled={!allDecided || finalizing}
        >
          <Send size={14} /> {finalizing ? '定稿中...' : '定稿归档'}
        </button>
      </div>

      {/* ===== 主体 ===== */}
      <div style={{ flex: 1, display: 'flex', gap: 0, overflow: 'hidden', marginTop: 12 }}>

        {/* ===== 左侧列表（按页码顺序） ===== */}
        <div style={{
          width: 300, flexShrink: 0, background: 'rgba(255,255,255,0.72)',
          backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)',
          borderRadius: '14px 0 0 14px', overflow: 'auto',
        }}>
          {/* 头部统计 */}
          <div style={{
            padding: '14px 16px', fontSize: 13, fontWeight: 600, color: '#1c1c1e',
            borderBottom: '1px solid rgba(0,0,0,0.04)', position: 'sticky', top: 0,
            background: 'rgba(255,255,255,0.95)', backdropFilter: 'blur(20px)', zIndex: 2,
          }}>
            <div>课程页面 ({totalPages})</div>
            <div style={{ display: 'flex', gap: 6, marginTop: 6, flexWrap: 'wrap' }}>
              {Object.entries(opStats).map(([op, count]) => (
                <span key={op} style={{
                  fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px',
                  borderRadius: 4, background: OP_COLORS[op] || '#aeaeb2',
                }}>{OP_NAMES[op] || op} {count}</span>
              ))}
            </div>
          </div>

          {/* 页面列表 */}
          {pages.map((page, idx) => (
            <div
              key={page.page_number}
              onClick={() => selectPage(idx)}
              style={{
                padding: '10px 16px', cursor: 'pointer',
                background: idx === selectedIdx ? 'rgba(0,122,255,0.08)' : 'transparent',
                borderBottom: '1px solid rgba(0,0,0,0.03)',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{
                  fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 6px',
                  borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2', flexShrink: 0,
                }}>{OP_NAMES[page.operation] || page.operation}</span>
                <span style={{
                  fontSize: 13, fontWeight: 500, color: '#1c1c1e',
                  overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1,
                }}>P{page.page_number}. {page.page_title || '无标题'}</span>
              </div>
              {page.operation === 'merge' && page.merge_sources && page.merge_sources !== 'null' && (
                <div style={{ fontSize: 10, color: '#ff9500', marginTop: 2, paddingLeft: 42 }}>
                  合并自: {parseMergeSources(page.merge_sources).map(n => 'P' + n).join('+')}
                </div>
              )}
              <div style={{
                fontSize: 11, marginTop: 3, paddingLeft: 42,
                color: DECISION_COLORS[page.decision] || '#c7c7cc',
                fontWeight: page.decision !== 'pending' ? 600 : 400,
              }}>{DECISION_NAMES[page.decision] || page.decision}</div>
            </div>
          ))}
          {pages.length === 0 && (
            <div style={{ padding: 20, textAlign: 'center', color: '#aeaeb2', fontSize: 13 }}>暂无生成页面</div>
          )}
        </div>

        {/* ===== 右侧预览 ===== */}
        <div style={{
          flex: 1, display: 'flex', flexDirection: 'column',
          background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
          border: '1px solid rgba(0,0,0,0.06)', borderLeft: 'none',
          borderRadius: '0 14px 14px 0', overflow: 'hidden',
        }}>
          {currentPage ? (
            <>
              {/* 工具栏 */}
              <div style={{
                display: 'flex', alignItems: 'center', gap: 8, padding: '10px 16px',
                borderBottom: '1px solid rgba(0,0,0,0.04)', flexShrink: 0, flexWrap: 'wrap',
              }}>
                <span style={{
                  fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px',
                  borderRadius: 4, background: OP_COLORS[currentPage.operation] || '#aeaeb2',
                }}>{OP_NAMES[currentPage.operation] || currentPage.operation}</span>
                <span style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>
                  P{currentPage.page_number}. {currentPage.page_title || '无标题'}
                </span>
                {currentPage.operation === 'merge' && currentPage.merge_sources && currentPage.merge_sources !== 'null' && (
                  <span style={{ fontSize: 11, color: '#ff9500', fontWeight: 500 }}>
                    (合并自 {parseMergeSources(currentPage.merge_sources).map(n => 'P' + n).join(' + ')})
                  </span>
                )}
                <div style={{ flex: 1 }} />

                {!editingHTML && (
                  <button onClick={() => setFullscreen(true)} style={{ ...btn, padding: '6px 10px', fontSize: 12 }}
                    title="全屏预览（←→翻页，可切换原版/修改版）">
                    <Maximize2 size={13} /> 全屏
                  </button>
                )}
                {currentPage.change_reason && (
                  <button onClick={() => setReasonExpanded(!reasonExpanded)} style={{
                    ...btn, padding: '6px 10px', fontSize: 12,
                    background: reasonExpanded ? '#f0f7ff' : '#fff',
                    color: reasonExpanded ? '#007aff' : '#3c3c43',
                  }}><FileText size={13} /> {reasonExpanded ? '收起理由' : '修改理由'}</button>
                )}
                {!editingHTML && (
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }}
                      onClick={() => handleDecision('approve')} disabled={deciding}><Check size={14} /> 采用</button>
                    <button style={{ ...btn, background: '#ff3b30', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1, padding: '6px 14px' }}
                      onClick={() => handleDecision('reject')} disabled={deciding}><X size={14} /> 拒绝</button>
                    <button style={{ ...btn, background: '#ff9500', color: '#fff', border: 'none', padding: '6px 14px' }}
                      onClick={startEdit}><Edit3 size={14} /> 编辑</button>
                  </div>
                )}
                {editingHTML && (
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1 }}
                      onClick={() => handleDecision('edit', editContent)} disabled={deciding}><CheckCircle size={14} /> 保存编辑</button>
                    <button style={btn} onClick={() => setEditingHTML(false)}>取消</button>
                  </div>
                )}
              </div>

              {/* 修改理由 */}
              {currentPage.change_reason && reasonExpanded && (
                <div style={{
                  padding: '12px 16px', background: '#f8f9ff',
                  borderBottom: '1px solid rgba(0,122,255,0.1)', flexShrink: 0,
                  maxHeight: 200, overflow: 'auto',
                }}>
                  <div style={{ fontSize: 12, fontWeight: 600, color: '#007aff', marginBottom: 6 }}>
                    修改理由（Translator指令）
                  </div>
                  <div style={{
                    fontSize: 12, color: '#3c3c43', lineHeight: 1.7,
                    whiteSpace: 'pre-wrap', wordBreak: 'break-word',
                  }}>{currentPage.change_reason}</div>
                </div>
              )}

              {/* 预览区 */}
              <div style={{ flex: 1, overflow: 'hidden' }}>
                {editingHTML ? (
                  <textarea value={editContent} onChange={e => setEditContent(e.target.value)} style={{
                    width: '100%', height: '100%', border: 'none', outline: 'none',
                    fontFamily: 'Monaco, Consolas, "Courier New", monospace',
                    fontSize: 12, lineHeight: 1.6, padding: 16, resize: 'none',
                    background: '#fafafa', color: '#1c1c1e', boxSizing: 'border-box',
                  }} />
                ) : currentPage.operation === 'merge' ? (
                  <MergeCompareView page={currentPage} mergeSourceTab={mergeSourceTab} onTabChange={setMergeSourceTab} />
                ) : currentPage.operation === 'modify' ? (
                  <div style={{ display: 'flex', height: '100%' }}>
                    <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}>
                      <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0 }}>原版</div>
                      <HTMLPreview html={currentPage.original_html} />
                    </div>
                    <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
                      <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#007aff', background: '#f0f7ff', flexShrink: 0 }}>修改后</div>
                      <HTMLPreview html={currentPage.generated_html} />
                    </div>
                  </div>
                ) : currentPage.operation === 'create' ? (
                  <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
                    <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#af52de', background: '#faf5ff', flexShrink: 0 }}>新建页面（无原版）</div>
                    <HTMLPreview html={currentPage.generated_html} />
                  </div>
                ) : currentPage.operation === 'delete' ? (
                  <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
                    <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff3b30', background: '#fff5f5', flexShrink: 0 }}>此页面将被删除</div>
                    <HTMLPreview html={currentPage.original_html} />
                  </div>
                ) : (
                  <HTMLPreview html={currentPage.original_html || currentPage.final_html} />
                )}
              </div>
            </>
          ) : (
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 14 }}>
              请从左侧选择一个页面进行预览
            </div>
          )}
        </div>
      </div>

      {/* ===== 全屏预览 ===== */}
      {fullscreen && currentPage && (
        <FullscreenPreview
          page={currentPage} pages={pages} currentIdx={selectedIdx}
          pipelineId={id || ''}
          onNavigate={(idx) => { setSelectedIdx(idx); setMergeSourceTab(0) }}
          onClose={() => setFullscreen(false)}
          onPageUpdated={(pageNumber, newHTML) => {
            setPages(prev => prev.map(p =>
              p.page_number === pageNumber ? { ...p, generated_html: newHTML, final_html: newHTML } : p
            ))
          }}
        />
      )}
    </div>
  )
}

// ==================== 合并页对比 ====================

function MergeCompareView({ page, mergeSourceTab, onTabChange }: {
  page: GeneratedPageFull; mergeSourceTab: number; onTabChange: (idx: number) => void
}) {
  const sourceNums = parseMergeSources(page.merge_sources)

  if (sourceNums.length === 0) {
    return (
      <div style={{ display: 'flex', height: '100%' }}>
        <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}>
          <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0 }}>原版（第一个源页面）</div>
          <HTMLPreview html={page.original_html} />
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff9500', background: '#fff8f0', flexShrink: 0 }}>合并结果</div>
          <HTMLPreview html={page.generated_html} />
        </div>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', height: '100%' }}>
      <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}>
        <div style={{ display: 'flex', gap: 0, background: '#f9f9f9', borderBottom: '1px solid rgba(0,0,0,0.04)', flexShrink: 0 }}>
          {sourceNums.map((pn, idx) => (
            <button key={pn} onClick={() => onTabChange(idx)} style={{
              padding: '8px 16px', border: 'none', fontSize: 12, fontWeight: 600, cursor: 'pointer',
              background: mergeSourceTab === idx ? '#fff' : '#f5f5f5',
              color: mergeSourceTab === idx ? '#ff9500' : '#8e8e93',
              borderBottom: mergeSourceTab === idx ? '2px solid #ff9500' : '2px solid transparent',
            }}>源页面 P{pn}</button>
          ))}
          <div style={{ flex: 1, background: '#f5f5f5' }} />
        </div>
        {mergeSourceTab === 0 ? (
          <HTMLPreview html={page.original_html} />
        ) : (
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 13, padding: 20, textAlign: 'center' }}>
            源页面 P{sourceNums[mergeSourceTab]} 的原始HTML暂未单独存储。<br />目前仅第一个源页面(P{sourceNums[0]})可预览。
          </div>
        )}
      </div>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#ff9500', background: '#fff8f0', flexShrink: 0 }}>
          合并结果（{sourceNums.map(n => 'P' + n).join(' + ')} → P{page.page_number}）
        </div>
        <HTMLPreview html={page.generated_html} />
      </div>
    </div>
  )
}

// ==================== AI快修弹窗组件（P4.5-E-2新增） ====================

/**
 * 全屏模式下的AI快修模态框
 * 审核员输入修改指令，调用后端AI接口修复当前页面HTML
 */
function AIFixModal({ pipelineId, pageNumber, pageTitle, loading, instruction, onInstructionChange, onSubmit, onClose }: {
  pipelineId: string; pageNumber: number; pageTitle: string
  loading: boolean; instruction: string
  onInstructionChange: (v: string) => void
  onSubmit: () => void; onClose: () => void
}) {
  // 阻止ESC冒泡（同ReasonModal）
  useEffect(() => {
    const stopPropagation = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !loading) {
        e.stopPropagation()
        onClose()
      }
    }
    window.addEventListener('keydown', stopPropagation, true)
    return () => window.removeEventListener('keydown', stopPropagation, true)
  }, [onClose, loading])

  return (
    <div
      onClick={(e) => { if (e.target === e.currentTarget && !loading) onClose() }}
      style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        zIndex: 10001, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: 40,
      }}
    >
      <div style={{
        background: '#fff', borderRadius: 16, padding: 0,
        maxWidth: 640, width: '100%',
        display: 'flex', flexDirection: 'column',
        boxShadow: '0 20px 60px rgba(0,0,0,0.2), 0 0 0 1px rgba(0,0,0,0.05)',
      }}>
        {/* 标题栏 */}
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8, padding: '16px 20px',
          borderBottom: '1px solid rgba(0,0,0,0.06)', flexShrink: 0,
        }}>
          <Wand2 size={16} color="#e65100" />
          <span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e', flex: 1 }}>
            AI快修 — P{pageNumber}. {pageTitle || '无标题'}
          </span>
          <button
            onClick={onClose} disabled={loading}
            style={{
              padding: '4px 12px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)',
              background: '#f5f5f5', fontSize: 12, fontWeight: 500, color: '#3c3c43',
              cursor: loading ? 'not-allowed' : 'pointer', opacity: loading ? 0.5 : 1,
              display: 'inline-flex', alignItems: 'center', gap: 4,
            }}
          ><X size={12} /> 关闭</button>
        </div>

        {/* 说明 */}
        <div style={{
          padding: '12px 20px', background: '#fff8e1',
          borderBottom: '1px solid rgba(230,81,0,0.08)', fontSize: 12, color: '#e65100', lineHeight: 1.6,
        }}>
          输入修改指令，AI将基于当前页面HTML进行修复。修复完成后HTML会自动更新。
          <br />示例：&ldquo;修复第3个按钮点击无反应的问题&rdquo;、&ldquo;将标题字号从18px改为24px&rdquo;
        </div>

        {/* 输入区 */}
        <div style={{ padding: '16px 20px' }}>
          <textarea
            value={instruction}
            onChange={(e) => onInstructionChange(e.target.value)}
            placeholder="请输入修复指令，例如：修复互动按钮点击无反应的问题..."
            disabled={loading}
            style={{
              width: '100%', minHeight: 120, border: '1px solid rgba(0,0,0,0.1)',
              borderRadius: 10, padding: 12, fontSize: 13, lineHeight: 1.6,
              fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
              resize: 'vertical', outline: 'none', boxSizing: 'border-box',
              background: loading ? '#f9f9f9' : '#fff', color: '#1c1c1e',
            }}
            onFocus={(e) => { e.target.style.borderColor = 'rgba(0,122,255,0.4)' }}
            onBlur={(e) => { e.target.style.borderColor = 'rgba(0,0,0,0.1)' }}
          />
        </div>

        {/* 操作栏 */}
        <div style={{
          display: 'flex', justifyContent: 'flex-end', gap: 8, padding: '12px 20px',
          borderTop: '1px solid rgba(0,0,0,0.06)',
        }}>
          <button
            onClick={onClose} disabled={loading}
            style={{
              padding: '8px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
              background: '#fff', fontSize: 13, fontWeight: 500, color: '#3c3c43',
              cursor: loading ? 'not-allowed' : 'pointer', opacity: loading ? 0.5 : 1,
            }}
          >取消</button>
          <button
            onClick={onSubmit} disabled={loading || !instruction.trim()}
            style={{
              padding: '8px 24px', borderRadius: 10, border: 'none',
              background: loading || !instruction.trim() ? '#e5e5ea' : '#e65100',
              color: loading || !instruction.trim() ? '#aeaeb2' : '#fff',
              fontSize: 13, fontWeight: 600,
              cursor: loading || !instruction.trim() ? 'not-allowed' : 'pointer',
              display: 'inline-flex', alignItems: 'center', gap: 6,
            }}
          >
            {loading ? <><Loader size={14} style={{ animation: 'spin 1s linear infinite' }} /> AI修复中...</> : <><Wand2 size={14} /> 执行修复</>}
          </button>
        </div>
      </div>

      {/* loading动画CSS */}
      {loading && (
        <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`}</style>
      )}
    </div>
  )
}

// ==================== 修改理由弹窗组件（P4.5-E-1新增） ====================

/**
 * 全屏模式下的修改理由模态框
 * 点击后弹出完整修改理由文本，支持ESC或点击遮罩关闭
 */
function ReasonModal({ reason, onClose }: { reason: string; onClose: () => void }) {
  // 阻止键盘事件冒泡到全屏翻页（避免ESC关闭弹窗时同时关闭全屏）
  useEffect(() => {
    const stopPropagation = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation()
        onClose()
      }
    }
    // 用capture阶段拦截，优先于全屏的keydown处理
    window.addEventListener('keydown', stopPropagation, true)
    return () => window.removeEventListener('keydown', stopPropagation, true)
  }, [onClose])

  return (
    <div
      onClick={(e) => { if (e.target === e.currentTarget) onClose() }}
      style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        zIndex: 10001, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: 40,
      }}
    >
      <div style={{
        background: '#fff', borderRadius: 16, padding: 0,
        maxWidth: 680, width: '100%', maxHeight: '70vh',
        display: 'flex', flexDirection: 'column',
        boxShadow: '0 20px 60px rgba(0,0,0,0.2), 0 0 0 1px rgba(0,0,0,0.05)',
      }}>
        {/* 弹窗标题栏 */}
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8, padding: '16px 20px',
          borderBottom: '1px solid rgba(0,0,0,0.06)', flexShrink: 0,
        }}>
          <FileText size={16} color="#007aff" />
          <span style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e', flex: 1 }}>
            修改理由（Translator指令）
          </span>
          <button
            onClick={onClose}
            style={{
              padding: '4px 12px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)',
              background: '#f5f5f5', fontSize: 12, fontWeight: 500, color: '#3c3c43',
              cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 4,
            }}
          ><X size={12} /> 关闭</button>
        </div>
        {/* 弹窗内容区（可滚动） */}
        <div style={{
          padding: '20px 24px', overflow: 'auto', flex: 1,
          fontSize: 13, color: '#3c3c43', lineHeight: 1.8,
          whiteSpace: 'pre-wrap', wordBreak: 'break-word',
        }}>
          {reason}
        </div>
      </div>
    </div>
  )
}

// ==================== 全屏预览（支持翻页 + 原版/修改版切换 + 修改理由弹窗） ====================

function FullscreenPreview({ page, pages, currentIdx, pipelineId, onNavigate, onClose, onPageUpdated }: {
  page: GeneratedPageFull; pages: GeneratedPageFull[]; currentIdx: number
  pipelineId: string
  onNavigate: (idx: number) => void; onClose: () => void
  onPageUpdated: (pageNumber: number, newHTML: string) => void
}) {
  // 全屏视图模式：generated=修改后/生成版，original=原版，compare=左右对比
  const [fsMode, setFsMode] = useState<'generated' | 'original' | 'compare'>('generated')
  // P4.5-E-1：修改理由弹窗显示状态
  const [showReasonModal, setShowReasonModal] = useState(false)
  // P4.5-E-2：AI快修相关状态
  const [showAIFixModal, setShowAIFixModal] = useState(false)
  const [aiFixInstruction, setAIFixInstruction] = useState('')
  const [aiFixLoading, setAIFixLoading] = useState(false)

  const hasPrev = currentIdx > 0
  const hasNext = currentIdx < pages.length - 1

  // 判断当前页是否有原版和修改版（用于决定哪些切换按钮可用）
  const hasOriginal = !!(page.original_html && page.original_html.length > 10)
  const hasGenerated = !!(page.generated_html && page.generated_html.length > 10)
  const hasBoth = hasOriginal && hasGenerated

  // 根据操作类型设置默认模式
  useEffect(() => {
    if (page.operation === 'keep' || page.operation === 'delete') {
      setFsMode('original')
    } else if (page.operation === 'create') {
      setFsMode('generated')
    } else {
      setFsMode('generated')
    }
    // 翻页时关闭修改理由弹窗和AI快修弹窗
    setShowReasonModal(false)
    setShowAIFixModal(false)
    setAIFixInstruction('')
  }, [page.page_number, page.operation])

  const navBtn: React.CSSProperties = {
    padding: '6px 10px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.1)',
    background: '#fff', display: 'inline-flex', alignItems: 'center',
    fontSize: 13, fontWeight: 500, color: '#3c3c43', cursor: 'pointer',
  }

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
      zIndex: 9999, background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(8px)',
      display: 'flex', flexDirection: 'column',
    }} onClick={(e) => { if (e.target === e.currentTarget) onClose() }}>

      {/* 顶部工具栏 */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 10, padding: '10px 20px',
        background: 'rgba(255,255,255,0.95)', backdropFilter: 'blur(20px)',
        borderBottom: '1px solid rgba(0,0,0,0.08)', flexShrink: 0,
      }}>
        {/* 翻页：上一页 */}
        <button onClick={() => hasPrev && onNavigate(currentIdx - 1)}
          style={{ ...navBtn, opacity: hasPrev ? 1 : 0.3, cursor: hasPrev ? 'pointer' : 'not-allowed' }}
          disabled={!hasPrev} title="上一页 (←)"><ChevronLeft size={16} /></button>

        {/* 页面信息 */}
        <span style={{
          fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px',
          borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2',
        }}>{OP_NAMES[page.operation] || page.operation}</span>
        <span style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>
          P{page.page_number}. {page.page_title || '无标题'}
        </span>
        <span style={{ fontSize: 12, color: '#8e8e93' }}>{currentIdx + 1}/{pages.length}</span>

        {/* 翻页：下一页 */}
        <button onClick={() => hasNext && onNavigate(currentIdx + 1)}
          style={{ ...navBtn, opacity: hasNext ? 1 : 0.3, cursor: hasNext ? 'pointer' : 'not-allowed' }}
          disabled={!hasNext} title="下一页 (→)"><ChevronRight size={16} /></button>

        <div style={{ flex: 1 }} />

        {/* ===== 原版/修改版/对比 切换按钮组 ===== */}
        {hasBoth && (
          <div style={{ display: 'flex', gap: 0, borderRadius: 8, overflow: 'hidden', border: '1px solid rgba(0,0,0,0.1)' }}>
            {([
              { key: 'original' as const, label: '原版' },
              { key: 'generated' as const, label: '修改后' },
              { key: 'compare' as const, label: '对比' },
            ]).map(item => (
              <button key={item.key} onClick={() => setFsMode(item.key)} style={{
                padding: '6px 14px', border: 'none', fontSize: 12, fontWeight: 500, cursor: 'pointer',
                background: fsMode === item.key ? '#007aff' : '#fff',
                color: fsMode === item.key ? '#fff' : '#3c3c43',
              }}>{item.key === 'compare' && <Columns size={12} style={{ marginRight: 4, verticalAlign: -1 }} />}{item.label}</button>
            ))}
          </div>
        )}
        {/* 只有原版 */}
        {hasOriginal && !hasGenerated && (
          <span style={{ fontSize: 12, color: '#8e8e93', fontStyle: 'italic' }}>仅原版</span>
        )}
        {/* 只有生成版 */}
        {!hasOriginal && hasGenerated && (
          <span style={{ fontSize: 12, color: '#8e8e93', fontStyle: 'italic' }}>仅生成版（新建）</span>
        )}

        {/* P4.5-E-1：修改理由按钮（替换原来的缩略文本，改为点击弹窗） */}
        {page.change_reason && (
          <button
            onClick={() => setShowReasonModal(true)}
            style={{
              ...navBtn,
              padding: '6px 14px',
              background: '#f0f7ff',
              color: '#007aff',
              border: '1px solid rgba(0,122,255,0.15)',
            }}
            title="点击查看完整修改理由"
          >
            <FileText size={13} /> 修改理由
          </button>
        )}

        {/* P4.5-E-2：AI快修按钮 */}
        <button
          onClick={() => { setShowAIFixModal(true); setAIFixInstruction('') }}
          style={{
            ...navBtn,
            padding: '6px 14px',
            background: '#fff3e0',
            color: '#e65100',
            border: '1px solid rgba(230,81,0,0.15)',
          }}
          title="AI快修：输入修改指令让AI修复当前页面"
        >
          <Wand2 size={13} /> AI快修
        </button>

        {/* 关闭 */}
        <button onClick={onClose} style={{ ...navBtn, padding: '6px 14px' }} title="退出 (ESC)">
          <Minimize2 size={14} /> 退出
        </button>
      </div>

      {/* 全屏内容区 */}
      <div style={{ flex: 1, overflow: 'hidden', background: '#fff', position: 'relative' }}>
        {fsMode === 'compare' && hasBoth ? (
          /* 对比模式：左右分栏 */
          <div style={{ display: 'flex', height: '100%' }}>
            <div style={{ flex: 1, borderRight: '2px solid rgba(0,0,0,0.08)', display: 'flex', flexDirection: 'column' }}>
              <div style={{ padding: '8px 16px', fontSize: 13, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0, borderBottom: '1px solid rgba(0,0,0,0.04)' }}>原版</div>
              <HTMLPreview html={page.original_html} />
            </div>
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
              <div style={{ padding: '8px 16px', fontSize: 13, fontWeight: 600, color: '#007aff', background: '#f0f7ff', flexShrink: 0, borderBottom: '1px solid rgba(0,0,0,0.04)' }}>修改后</div>
              <HTMLPreview html={page.generated_html} />
            </div>
          </div>
        ) : fsMode === 'original' && hasOriginal ? (
          /* 原版 */
          <HTMLPreview html={page.original_html} />
        ) : hasGenerated ? (
          /* 修改后/生成版 */
          <HTMLPreview html={page.generated_html} />
        ) : hasOriginal ? (
          /* 回退到原版 */
          <HTMLPreview html={page.original_html} />
        ) : (
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2' }}>（无内容）</div>
        )}

        {/* 左右翻页热区 */}
        {hasPrev && (
          <div onClick={() => onNavigate(currentIdx - 1)} style={{
            position: 'absolute', left: 0, top: 0, bottom: 0, width: 60, cursor: 'pointer',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'linear-gradient(to right, rgba(0,0,0,0.05), transparent)',
          }}><ChevronLeft size={24} color="#8e8e93" /></div>
        )}
        {hasNext && (
          <div onClick={() => onNavigate(currentIdx + 1)} style={{
            position: 'absolute', right: 0, top: 0, bottom: 0, width: 60, cursor: 'pointer',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'linear-gradient(to left, rgba(0,0,0,0.05), transparent)',
          }}><ChevronRight size={24} color="#8e8e93" /></div>
        )}
      </div>

      {/* P4.5-E-1：修改理由弹窗 */}
      {showReasonModal && page.change_reason && (
        <ReasonModal reason={page.change_reason} onClose={() => setShowReasonModal(false)} />
      )}

      {/* P4.5-E-2：AI快修弹窗 */}
      {showAIFixModal && (
        <AIFixModal
          pipelineId={pipelineId}
          pageNumber={page.page_number}
          pageTitle={page.page_title}
          loading={aiFixLoading}
          instruction={aiFixInstruction}
          onInstructionChange={setAIFixInstruction}
          onSubmit={async () => {
            if (!aiFixInstruction.trim()) { alert('请输入修复指令'); return }
            setAIFixLoading(true)
            try {
              const resp = await aiFixPage(pipelineId, page.page_number, { fix_instruction: aiFixInstruction.trim() })
              onPageUpdated(page.page_number, resp.new_html)
              alert('AI快修完成！新HTML已更新（' + resp.html_length + '字符）')
              setShowAIFixModal(false)
              setAIFixInstruction('')
            } catch (e: any) {
              alert('AI快修失败: ' + (e?.response?.data?.message || e.message || '未知错误'))
            }
            setAIFixLoading(false)
          }}
          onClose={() => { if (!aiFixLoading) { setShowAIFixModal(false); setAIFixInstruction('') } }}
        />
      )}
    </div>
  )
}

// ==================== HTML预览 ====================

function HTMLPreview({ html }: { html: string }) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const doc = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<style>
body{margin:0;padding:16px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;font-size:14px;line-height:1.6;color:#1c1c1e;word-wrap:break-word}
img{max-width:100%;height:auto}table{border-collapse:collapse;width:100%}th,td{border:1px solid #e5e5ea;padding:8px 12px;text-align:left}th{background:#f5f5f7;font-weight:600}
</style></head><body>${html || '<p style="color:#aeaeb2;text-align:center;padding:40px 0">(空内容)</p>'}</body></html>`

  if (!html && html !== '') {
    return <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#aeaeb2', fontSize: 13 }}>（无HTML内容）</div>
  }
  return <iframe ref={iframeRef} srcDoc={doc} sandbox="allow-same-origin allow-scripts"
    style={{ flex: 1, width: '100%', height: '100%', border: 'none', background: '#fff' }} title="HTML预览" />
}

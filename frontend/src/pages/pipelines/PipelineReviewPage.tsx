/**
 * Pipeline审核页面
 * P4.5-C: 生成页面预览 + 原版vs修改版对比 + 逐页审核决策 + 定稿归档
 * P4.5-C增强: 全屏预览功能（支持单视图/对比模式全屏 + ESC退出）
 *
 * 功能：
 * 1. 页面列表侧边栏（含操作类型彩色标记 + 决策状态）
 * 2. HTML预览面板（iframe沙箱渲染）
 * 3. 原版 vs 生成版 对比视图
 * 4. 逐页决策（approve采用/reject拒绝/edit编辑）
 * 5. 全部决策后可定稿归档
 * 6. 全屏预览（单视图/对比模式，ESC或关闭按钮退出）
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
} from '@/api/pipelines'
import { ArrowLeft, RefreshCw, Check, X, Edit3, CheckCircle, Send, Maximize2, Minimize2 } from 'lucide-react'

// ==================== 常量定义 ====================

/** 操作类型 → 颜色映射 */
const OP_COLORS: Record<string, string> = {
  keep: '#34c759',
  modify: '#007aff',
  create: '#af52de',
  merge: '#ff9500',
  delete: '#ff3b30',
}

/** 操作类型 → 中文名映射 */
const OP_NAMES: Record<string, string> = {
  keep: '保留',
  modify: '修改',
  create: '新建',
  merge: '合并',
  delete: '删除',
}

/** 决策 → 颜色映射 */
const DECISION_COLORS: Record<string, string> = {
  approve: '#34c759',
  reject: '#ff3b30',
  edit: '#ff9500',
  pending: '#c7c7cc',
}

/** 决策 → 中文名映射 */
const DECISION_NAMES: Record<string, string> = {
  approve: '已采用',
  reject: '已拒绝',
  edit: '已编辑',
  pending: '待决策',
}

// ==================== 主组件 ====================

export default function PipelineReviewPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  // 状态
  const [pipeline, setPipeline] = useState<PipelineDetailResponse | null>(null)
  const [pages, setPages] = useState<GeneratedPageFull[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [selectedIdx, setSelectedIdx] = useState(0)
  const [viewMode, setViewMode] = useState<'generated' | 'original' | 'compare'>('generated')
  const [finalizing, setFinalizing] = useState(false)
  const [deciding, setDeciding] = useState(false)
  const [editingHTML, setEditingHTML] = useState(false)
  const [editContent, setEditContent] = useState('')
  // 全屏预览状态
  const [fullscreen, setFullscreen] = useState(false)
  const [fullscreenMode, setFullscreenMode] = useState<'generated' | 'original' | 'compare'>('generated')

  /** 加载数据 */
  const loadData = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError('')
    try {
      const [pipelineData, pagesData] = await Promise.all([
        getPipelineDetail(id),
        getGeneratedPages(id),
      ])
      setPipeline(pipelineData)
      setPages(pagesData || [])
    } catch (e: any) {
      setError(e.message || '加载失败')
    }
    setLoading(false)
  }, [id])

  useEffect(() => { loadData() }, [loadData])

  /** ESC键退出全屏 */
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && fullscreen) {
        setFullscreen(false)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [fullscreen])

  /** 当前选中的页面 */
  const currentPage = pages[selectedIdx] || null

  /** 统计决策情况 */
  const totalPages = pages.length
  const decidedPages = pages.filter(p => p.decision !== 'pending').length
  const allDecided = totalPages > 0 && decidedPages === totalPages

  /** 处理决策 */
  const handleDecision = async (decision: 'approve' | 'reject' | 'edit', finalHTML?: string) => {
    if (!id || !currentPage) return
    setDeciding(true)
    try {
      const req: UpdatePageDecisionRequest = { decision }
      if (decision === 'edit' && finalHTML) {
        req.final_html = finalHTML
      }
      await updatePageDecision(id, currentPage.page_number, req)
      // 更新本地状态
      setPages(prev => prev.map(p =>
        p.page_number === currentPage.page_number
          ? { ...p, decision, final_html: finalHTML || p.final_html }
          : p
      ))
      setEditingHTML(false)
      // 自动跳到下一个待决策页面
      const nextPendingIdx = pages.findIndex((p, i) => i > selectedIdx && p.decision === 'pending')
      if (nextPendingIdx >= 0) {
        setSelectedIdx(nextPendingIdx)
      }
    } catch (e: any) {
      alert('决策失败: ' + (e.message || '未知错误'))
    }
    setDeciding(false)
  }

  /** 处理定稿 */
  const handleFinalize = async () => {
    if (!id || !allDecided) return
    if (!confirm('确认定稿归档？定稿后Pipeline将标记为已完成。')) return
    setFinalizing(true)
    try {
      await finalizePipeline(id)
      alert('定稿成功！')
      navigate('/pipelines')
    } catch (e: any) {
      alert('定稿失败: ' + (e.message || '未知错误'))
    }
    setFinalizing(false)
  }

  /** 进入编辑模式 */
  const startEdit = () => {
    if (!currentPage) return
    // 优先使用已有的final_html，其次generated_html
    setEditContent(currentPage.final_html || currentPage.generated_html || '')
    setEditingHTML(true)
  }

  /** 提交编辑 */
  const submitEdit = () => {
    handleDecision('edit', editContent)
  }

  /** 打开全屏预览 */
  const openFullscreen = () => {
    setFullscreenMode(viewMode)
    setFullscreen(true)
  }

  // 通用按钮样式
  const btn: React.CSSProperties = {
    padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 6,
  }

  // 加载中
  if (loading) {
    return <div style={{ textAlign: 'center', padding: 60, color: '#8e8e93' }}>加载中...</div>
  }

  // 错误状态
  if (error) {
    return (
      <div style={{ textAlign: 'center', padding: 60 }}>
        <div style={{ color: '#ff3b30', fontSize: 14, marginBottom: 12 }}>{error}</div>
        <button style={btn} onClick={() => navigate(-1)}>
          <ArrowLeft size={14} /> 返回
        </button>
      </div>
    )
  }

  if (!pipeline) return null

  return (
    <div style={{ height: 'calc(100vh - 80px)', display: 'flex', flexDirection: 'column' }}>
      {/* 顶部导航栏 */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 12, padding: '12px 0',
        borderBottom: '1px solid rgba(0,0,0,0.06)', marginBottom: 0, flexShrink: 0,
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
        <button style={btn} onClick={loadData}>
          <RefreshCw size={14} /> 刷新
        </button>
        {/* 定稿按钮 */}
        <button
          style={{
            ...btn,
            background: allDecided ? '#34c759' : '#e5e5ea',
            color: allDecided ? '#fff' : '#aeaeb2',
            border: 'none',
            cursor: allDecided ? 'pointer' : 'not-allowed',
          }}
          onClick={handleFinalize}
          disabled={!allDecided || finalizing}
        >
          <Send size={14} /> {finalizing ? '定稿中...' : '定稿归档'}
        </button>
      </div>

      {/* 主体：左侧页面列表 + 右侧预览 */}
      <div style={{ flex: 1, display: 'flex', gap: 0, overflow: 'hidden', marginTop: 12 }}>
        {/* 左侧页面列表 */}
        <div style={{
          width: 280, flexShrink: 0, background: 'rgba(255,255,255,0.72)',
          backdropFilter: 'blur(20px)', border: '1px solid rgba(0,0,0,0.06)',
          borderRadius: '14px 0 0 14px', overflow: 'auto',
        }}>
          <div style={{
            padding: '14px 16px', fontSize: 13, fontWeight: 600, color: '#1c1c1e',
            borderBottom: '1px solid rgba(0,0,0,0.04)', position: 'sticky', top: 0,
            background: 'rgba(255,255,255,0.95)', backdropFilter: 'blur(20px)',
          }}>
            页面列表 ({totalPages})
          </div>
          {pages.map((page, idx) => (
            <div
              key={page.page_number}
              onClick={() => { setSelectedIdx(idx); setEditingHTML(false) }}
              style={{
                padding: '10px 16px', cursor: 'pointer',
                background: idx === selectedIdx ? 'rgba(0,122,255,0.08)' : 'transparent',
                borderBottom: '1px solid rgba(0,0,0,0.03)',
                transition: 'background 0.15s ease',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                {/* 操作类型标记 */}
                <span style={{
                  fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 6px',
                  borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2',
                  flexShrink: 0,
                }}>
                  {OP_NAMES[page.operation] || page.operation}
                </span>
                {/* 页码和标题 */}
                <span style={{
                  fontSize: 13, fontWeight: 500, color: '#1c1c1e',
                  overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1,
                }}>
                  P{page.page_number}. {page.page_title || '无标题'}
                </span>
              </div>
              {/* 决策状态 */}
              <div style={{
                fontSize: 11, marginTop: 4,
                color: DECISION_COLORS[page.decision] || '#c7c7cc',
                fontWeight: page.decision !== 'pending' ? 600 : 400,
              }}>
                {DECISION_NAMES[page.decision] || page.decision}
              </div>
            </div>
          ))}
          {pages.length === 0 && (
            <div style={{ padding: 20, textAlign: 'center', color: '#aeaeb2', fontSize: 13 }}>
              暂无生成页面
            </div>
          )}
        </div>

        {/* 右侧预览面板 */}
        <div style={{
          flex: 1, display: 'flex', flexDirection: 'column',
          background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
          border: '1px solid rgba(0,0,0,0.06)', borderLeft: 'none',
          borderRadius: '0 14px 14px 0', overflow: 'hidden',
        }}>
          {currentPage ? (
            <>
              {/* 预览工具栏 */}
              <div style={{
                display: 'flex', alignItems: 'center', gap: 8, padding: '10px 16px',
                borderBottom: '1px solid rgba(0,0,0,0.04)', flexShrink: 0,
                flexWrap: 'wrap',
              }}>
                {/* 视图切换 */}
                <div style={{ display: 'flex', gap: 0, borderRadius: 8, overflow: 'hidden', border: '1px solid rgba(0,0,0,0.08)' }}>
                  {(['generated', 'original', 'compare'] as const).map(mode => (
                    <button
                      key={mode}
                      onClick={() => { setViewMode(mode); setEditingHTML(false) }}
                      style={{
                        padding: '6px 14px', border: 'none', fontSize: 12, fontWeight: 500,
                        cursor: 'pointer',
                        background: viewMode === mode ? '#007aff' : '#fff',
                        color: viewMode === mode ? '#fff' : '#3c3c43',
                      }}
                    >
                      {mode === 'generated' ? '生成版' : mode === 'original' ? '原版' : '对比'}
                    </button>
                  ))}
                </div>

                {/* 全屏按钮 */}
                {!editingHTML && (
                  <button
                    onClick={openFullscreen}
                    style={{
                      padding: '6px 10px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)',
                      background: '#fff', cursor: 'pointer', display: 'inline-flex',
                      alignItems: 'center', gap: 4, fontSize: 12, fontWeight: 500, color: '#3c3c43',
                    }}
                    title="全屏预览"
                  >
                    <Maximize2 size={13} /> 全屏
                  </button>
                )}

                <div style={{ flex: 1 }} />

                {/* 页面信息 */}
                <span style={{ fontSize: 12, color: '#8e8e93' }}>
                  P{currentPage.page_number} · {OP_NAMES[currentPage.operation] || currentPage.operation}
                  {currentPage.lesson_id && ` · lesson_id: ${currentPage.lesson_id}`}
                </span>

                <div style={{ flex: 1 }} />

                {/* 决策按钮组 */}
                {!editingHTML && (
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button
                      style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1 }}
                      onClick={() => handleDecision('approve')}
                      disabled={deciding}
                      title="采用AI生成版本"
                    >
                      <Check size={14} /> 采用
                    </button>
                    <button
                      style={{ ...btn, background: '#ff3b30', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1 }}
                      onClick={() => handleDecision('reject')}
                      disabled={deciding}
                      title="拒绝，保留原版"
                    >
                      <X size={14} /> 拒绝
                    </button>
                    <button
                      style={{ ...btn, background: '#ff9500', color: '#fff', border: 'none' }}
                      onClick={startEdit}
                      title="手动编辑HTML"
                    >
                      <Edit3 size={14} /> 编辑
                    </button>
                  </div>
                )}
                {editingHTML && (
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button
                      style={{ ...btn, background: '#34c759', color: '#fff', border: 'none', opacity: deciding ? 0.6 : 1 }}
                      onClick={submitEdit}
                      disabled={deciding}
                    >
                      <CheckCircle size={14} /> 保存编辑
                    </button>
                    <button
                      style={btn}
                      onClick={() => setEditingHTML(false)}
                    >
                      取消
                    </button>
                  </div>
                )}
              </div>

              {/* 预览内容区域 */}
              <div style={{ flex: 1, overflow: 'auto' }}>
                {editingHTML ? (
                  /* 编辑模式：代码编辑器 */
                  <textarea
                    value={editContent}
                    onChange={e => setEditContent(e.target.value)}
                    style={{
                      width: '100%', height: '100%', border: 'none', outline: 'none',
                      fontFamily: 'Monaco, Consolas, "Courier New", monospace',
                      fontSize: 12, lineHeight: 1.6, padding: 16, resize: 'none',
                      background: '#fafafa', color: '#1c1c1e', boxSizing: 'border-box',
                    }}
                  />
                ) : viewMode === 'compare' ? (
                  /* 对比模式：左右分栏 */
                  <div style={{ display: 'flex', height: '100%' }}>
                    <div style={{ flex: 1, borderRight: '1px solid rgba(0,0,0,0.06)', display: 'flex', flexDirection: 'column' }}>
                      <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#8e8e93', background: '#f9f9f9', flexShrink: 0 }}>
                        原版 ({currentPage.original_html.length} 字符)
                      </div>
                      <HTMLPreview html={currentPage.original_html} />
                    </div>
                    <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
                      <div style={{ padding: '8px 12px', fontSize: 12, fontWeight: 600, color: '#007aff', background: '#f0f7ff', flexShrink: 0 }}>
                        生成版 ({currentPage.generated_html.length} 字符)
                      </div>
                      <HTMLPreview html={currentPage.generated_html} />
                    </div>
                  </div>
                ) : (
                  /* 单视图模式 */
                  <HTMLPreview
                    html={viewMode === 'original' ? currentPage.original_html : currentPage.generated_html}
                  />
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

      {/* 全屏预览遮罩层 */}
      {fullscreen && currentPage && (
        <FullscreenPreview
          page={currentPage}
          mode={fullscreenMode}
          onModeChange={setFullscreenMode}
          onClose={() => setFullscreen(false)}
        />
      )}
    </div>
  )
}

// ==================== 全屏预览组件 ====================

/**
 * FullscreenPreview — 全屏遮罩层预览HTML
 * 支持单视图（生成版/原版）和对比模式
 * ESC键或关闭按钮退出
 */
function FullscreenPreview({
  page,
  mode,
  onModeChange,
  onClose,
}: {
  page: GeneratedPageFull
  mode: 'generated' | 'original' | 'compare'
  onModeChange: (m: 'generated' | 'original' | 'compare') => void
  onClose: () => void
}) {
  return (
    <div
      style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        zIndex: 9999, background: 'rgba(0,0,0,0.6)',
        backdropFilter: 'blur(8px)',
        display: 'flex', flexDirection: 'column',
        animation: 'fadeIn 0.2s ease',
      }}
      onClick={(e) => {
        // 点击遮罩背景（非内容区域）关闭
        if (e.target === e.currentTarget) onClose()
      }}
    >
      {/* 全屏顶部工具栏 */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 12, padding: '12px 24px',
        background: 'rgba(255,255,255,0.95)', backdropFilter: 'blur(20px)',
        borderBottom: '1px solid rgba(0,0,0,0.08)', flexShrink: 0,
      }}>
        {/* 页面标题 */}
        <div style={{ fontSize: 15, fontWeight: 600, color: '#1c1c1e' }}>
          P{page.page_number}. {page.page_title || '无标题'}
        </div>
        <span style={{
          fontSize: 10, fontWeight: 600, color: '#fff', padding: '2px 8px',
          borderRadius: 4, background: OP_COLORS[page.operation] || '#aeaeb2',
        }}>
          {OP_NAMES[page.operation] || page.operation}
        </span>

        <div style={{ flex: 1 }} />

        {/* 视图切换 */}
        <div style={{ display: 'flex', gap: 0, borderRadius: 8, overflow: 'hidden', border: '1px solid rgba(0,0,0,0.1)' }}>
          {(['generated', 'original', 'compare'] as const).map(m => (
            <button
              key={m}
              onClick={() => onModeChange(m)}
              style={{
                padding: '6px 16px', border: 'none', fontSize: 13, fontWeight: 500,
                cursor: 'pointer',
                background: mode === m ? '#007aff' : '#fff',
                color: mode === m ? '#fff' : '#3c3c43',
              }}
            >
              {m === 'generated' ? '生成版' : m === 'original' ? '原版' : '对比'}
            </button>
          ))}
        </div>

        <div style={{ flex: 1 }} />

        {/* 关闭按钮 */}
        <button
          onClick={onClose}
          style={{
            padding: '6px 14px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.1)',
            background: '#fff', cursor: 'pointer', display: 'inline-flex',
            alignItems: 'center', gap: 5, fontSize: 13, fontWeight: 500, color: '#3c3c43',
          }}
          title="退出全屏 (ESC)"
        >
          <Minimize2 size={14} /> 退出全屏
        </button>
      </div>

      {/* 全屏预览内容 */}
      <div style={{ flex: 1, overflow: 'hidden', background: '#fff' }}>
        {mode === 'compare' ? (
          /* 对比模式 */
          <div style={{ display: 'flex', height: '100%' }}>
            <div style={{ flex: 1, borderRight: '2px solid rgba(0,0,0,0.08)', display: 'flex', flexDirection: 'column' }}>
              <div style={{
                padding: '10px 16px', fontSize: 13, fontWeight: 600, color: '#8e8e93',
                background: '#f9f9f9', flexShrink: 0, borderBottom: '1px solid rgba(0,0,0,0.04)',
              }}>
                原版 ({page.original_html.length.toLocaleString()} 字符)
              </div>
              <HTMLPreview html={page.original_html} />
            </div>
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
              <div style={{
                padding: '10px 16px', fontSize: 13, fontWeight: 600, color: '#007aff',
                background: '#f0f7ff', flexShrink: 0, borderBottom: '1px solid rgba(0,0,0,0.04)',
              }}>
                生成版 ({page.generated_html.length.toLocaleString()} 字符)
              </div>
              <HTMLPreview html={page.generated_html} />
            </div>
          </div>
        ) : (
          /* 单视图模式 */
          <HTMLPreview
            html={mode === 'original' ? page.original_html : page.generated_html}
          />
        )}
      </div>
    </div>
  )
}

// ==================== HTML预览组件 ====================

/**
 * HTMLPreview — 使用iframe沙箱安全渲染HTML内容
 * srcDoc方式，与主页面隔离
 */
function HTMLPreview({ html }: { html: string }) {
  const iframeRef = useRef<HTMLIFrameElement>(null)

  /** 构造完整的HTML文档（带基础样式重置） */
  const fullHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
    body {
      margin: 0;
      padding: 16px;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
      font-size: 14px;
      line-height: 1.6;
      color: #1c1c1e;
      word-wrap: break-word;
    }
    img { max-width: 100%; height: auto; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border: 1px solid #e5e5ea; padding: 8px 12px; text-align: left; }
    th { background: #f5f5f7; font-weight: 600; }
  </style>
</head>
<body>${html || '<p style="color:#aeaeb2;text-align:center;padding:40px 0;">（空内容）</p>'}</body>
</html>`

  if (!html && html !== '') {
    return (
      <div style={{
        flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
        color: '#aeaeb2', fontSize: 13,
      }}>
        （无HTML内容）
      </div>
    )
  }

  return (
    <iframe
      ref={iframeRef}
      srcDoc={fullHTML}
      sandbox="allow-same-origin"
      style={{
        flex: 1, width: '100%', height: '100%', border: 'none',
        background: '#fff',
      }}
      title="HTML预览"
    />
  )
}

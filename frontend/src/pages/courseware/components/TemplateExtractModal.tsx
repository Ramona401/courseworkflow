/**
 * TemplateExtractModal - AI 提取风格模板弹窗 (v145 SSE 异步版)
 *
 * v139 原版: 同步等待 10 分钟超时,只有秒表计时器
 * v145 改版: 异步 + SSE 推送实时进度,用户可看到每个阶段的状态
 *
 * 工作流程:
 *   1. 用户粘贴 1-6 页 HTML,点击「开始 AI 提取」
 *   2. 前端先订阅 SSE 连接(extract_start/progress/done/error 四种事件)
 *   3. 再触发 POST /extract 异步启动(后端 800ms 延迟等 SSE 建立)
 *   4. SSE 推送阶段进度文案,前端实时展示
 *   5. 收到 extract_done 后调用 onExtracted 回调,关闭弹窗
 */
import { useState, useEffect, useRef } from 'react'
import { extractTemplateFromHTML, subscribeExtractSSE } from '@/api/coursewares'
import type { ExtractTemplateResponse } from '@/api/coursewares'

// ==================== 颜色常量 ====================

const C = {
  primary: '#F59E0B', textPrimary: '#1F2937', textSecondary: '#6B7280',
  textMuted: '#9CA3AF', border: '#E5E7EB', bgCard: '#FFFFFF', danger: '#EF4444',
  success: '#10B981',
}

// ==================== 组件 Props ====================

interface Props {
  onClose: () => void
  onExtracted: (resp: ExtractTemplateResponse) => void
}

// ==================== 主组件 ====================

export default function TemplateExtractModal({ onClose, onExtracted }: Props) {
  const [pages, setPages] = useState<string[]>([''])
  const [loading, setLoading] = useState(false)
  const [elapsedSec, setElapsedSec] = useState(0)
  const [progressMsg, setProgressMsg] = useState('')
  const [error, setError] = useState('')

  // SSE 连接引用,组件卸载时安全关闭
  const sseRef = useRef<{ close: () => void } | null>(null)

  // 倒计时:loading 期间每秒 +1
  useEffect(() => {
    if (!loading) return
    setElapsedSec(0)
    const timer = setInterval(() => setElapsedSec(s => s + 1), 1000)
    return () => clearInterval(timer)
  }, [loading])

  // 组件卸载时关闭 SSE 连接,防止 Modal 关闭后继续烧 token
  useEffect(() => {
    return () => {
      if (sseRef.current) {
        sseRef.current.close()
        sseRef.current = null
      }
    }
  }, [])

  const totalLen = pages.reduce((s, p) => s + p.length, 0)
  const validPages = pages.filter(p => p.trim().length > 0)

  // ==================== 页面操作 ====================

  const addPage = () => {
    if (pages.length >= 6) return
    setPages([...pages, ''])
  }

  const removePage = (i: number) => {
    if (pages.length === 1) {
      setPages([''])
      return
    }
    setPages(pages.filter((_, idx) => idx !== i))
  }

  const updatePage = (i: number, val: string) => {
    const next = [...pages]
    next[i] = val
    setPages(next)
  }

  // ==================== 提取(SSE 驱动) ====================

  const handleExtract = async () => {
    setError('')
    if (validPages.length === 0) {
      setError('请至少粘贴一页 HTML 代码')
      return
    }
    if (totalLen > 200000) {
      setError(`HTML 总长度 ${totalLen} 字符,超出上限 200000`)
      return
    }
    setLoading(true)
    setProgressMsg('正在建立 SSE 连接...')

    // 1. 先订阅 SSE(前端先建连,后端 800ms 后开始广播)
    const sse = subscribeExtractSSE({
      onStart: (d) => {
        setProgressMsg(d.message)
      },
      onProgress: (d) => {
        setProgressMsg(d.message)
      },
      onDone: (d) => {
        setLoading(false)
        sseRef.current = null
        // 将 SSE 数据映射为 ExtractTemplateResponse 传给父组件
        onExtracted({
          template_id: d.template_id,
          suggested_name: d.suggested_name,
          suggested_desc: d.suggested_desc,
          suggested_category: d.suggested_category,
          extraction_notes: d.extraction_notes,
          message: d.message,
        })
      },
      onError: (d) => {
        setLoading(false)
        sseRef.current = null
        setError(d.message || 'AI 提取失败,请稍后重试')
      },
    })
    sseRef.current = sse

    // 2. 触发异步提取(后端立即返回,实际工作在 goroutine 中执行)
    try {
      await extractTemplateFromHTML(validPages, 'paste')
    } catch (e) {
      // POST 请求本身失败(网络错误/参数校验失败),关闭 SSE
      sse.close()
      sseRef.current = null
      setLoading(false)
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
      setError(msg || 'AI 提取请求发送失败')
    }
  }

  // ==================== 渲染 ====================

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh',
      background: 'rgba(0,0,0,0.6)', zIndex: 9999,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      backdropFilter: 'blur(4px)',
    }} onClick={loading ? undefined : onClose}>
      <div style={{
        background: '#fff', borderRadius: '20px',
        width: '92%', maxWidth: '900px', maxHeight: '92vh', overflow: 'hidden',
        display: 'flex', flexDirection: 'column',
      }} onClick={e => e.stopPropagation()}>

        {/* ---- 头部 ---- */}
        <div style={{
          padding: '20px 28px 16px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        }}>
          <div>
            <div style={{ fontSize: '20px', fontWeight: 700, color: C.textPrimary, display: 'flex', alignItems: 'center', gap: '10px' }}>
              <span style={{ fontSize: '24px' }}>✨</span>
              AI 提取风格模板
            </div>
            <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '4px' }}>
              粘贴 1-6 页 HTML,AI 抽象提取视觉风格并生成全新骨架样例页(约需 3-8 分钟)
            </div>
          </div>
          {!loading && (
            <button onClick={onClose} style={{ background: 'none', border: 'none', fontSize: '26px', cursor: 'pointer', color: C.textMuted, lineHeight: 1 }}>×</button>
          )}
        </div>

        {/* ---- 表单区 ---- */}
        <div style={{ flex: 1, padding: '20px 28px', overflow: 'auto' }}>
          {pages.map((p, i) => (
            <div key={i} style={{ marginBottom: '14px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary }}>
                  第 {i + 1} 页 HTML {p.trim() && <span style={{ fontWeight: 400, color: C.textMuted, fontSize: '11px' }}>· {p.length} 字符</span>}
                </label>
                {pages.length > 1 && (
                  <button onClick={() => removePage(i)} disabled={loading} style={{
                    padding: '2px 10px', borderRadius: '6px', border: `1px solid ${C.border}`,
                    background: 'transparent', color: C.danger, fontSize: '11px', cursor: loading ? 'not-allowed' : 'pointer',
                  }}>移除此页</button>
                )}
              </div>
              <textarea
                value={p}
                onChange={e => updatePage(i, e.target.value)}
                disabled={loading}
                placeholder={i === 0 ? '<!DOCTYPE html>\n<html>\n  <head>...</head>\n  <body>...</body>\n</html>' : '继续粘贴另一页 HTML(可选)...'}
                style={{
                  width: '100%', minHeight: '120px', padding: '10px 12px',
                  borderRadius: '8px', border: `1px solid ${C.border}`,
                  fontSize: '12px', fontFamily: 'Monaco, Consolas, monospace',
                  outline: 'none', resize: 'vertical', lineHeight: 1.5,
                  background: loading ? '#F9FAFB' : '#fff',
                }}
              />
            </div>
          ))}

          {pages.length < 6 && (
            <button onClick={addPage} disabled={loading} style={{
              padding: '8px 16px', borderRadius: '8px', border: `1px dashed ${C.primary}`,
              background: 'transparent', color: C.primary, fontSize: '13px',
              cursor: loading ? 'not-allowed' : 'pointer', width: '100%',
            }}>+ 增加一页(最多 6 页)</button>
          )}

          {/* ---- SSE 实时进度区 ---- */}
          {loading && progressMsg && (
            <div style={{
              marginTop: '16px', padding: '14px 16px', borderRadius: '10px',
              background: 'linear-gradient(135deg, #FEF3C7, #DBEAFE)',
              border: '1px solid #E5E7EB',
            }}>
              <div style={{ fontSize: '14px', fontWeight: 600, color: C.textPrimary, marginBottom: '6px' }}>
                {progressMsg}
              </div>
              <div style={{ fontSize: '12px', color: C.textSecondary }}>
                已耗时 {Math.floor(elapsedSec / 60)} 分 {elapsedSec % 60} 秒
              </div>
            </div>
          )}

          {/* ---- 错误提示 ---- */}
          {error && (
            <div style={{
              marginTop: '12px', padding: '10px 14px', borderRadius: '8px',
              background: '#FEE2E2', color: C.danger, fontSize: '13px',
            }}>⚠️ {error}</div>
          )}

          <div style={{ marginTop: '14px', fontSize: '12px', color: C.textMuted }}>
            总字符数: {totalLen} / 200000 · 有效页数: {validPages.length}
          </div>
        </div>

        {/* ---- 底部按钮 ---- */}
        <div style={{
          padding: '14px 28px', borderTop: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'flex-end', gap: '10px',
        }}>
          <button onClick={onClose} disabled={loading} style={{
            padding: '9px 20px', borderRadius: '8px', border: `1px solid ${C.border}`,
            background: 'transparent', color: C.textSecondary, fontSize: '14px',
            cursor: loading ? 'not-allowed' : 'pointer',
          }}>取消</button>
          <button onClick={handleExtract} disabled={loading || validPages.length === 0} style={{
            padding: '9px 24px', borderRadius: '8px', border: 'none',
            background: loading || validPages.length === 0
              ? '#D1D5DB'
              : 'linear-gradient(135deg, #F59E0B, #EF4444)',
            color: '#fff', fontSize: '14px', fontWeight: 600,
            cursor: loading || validPages.length === 0 ? 'not-allowed' : 'pointer',
            display: 'flex', alignItems: 'center', gap: '8px',
          }}>
            {loading && <span style={{ display: 'inline-block', width: '14px', height: '14px', border: '2px solid rgba(255,255,255,0.4)', borderTopColor: '#fff', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />}
            {loading
              ? `AI 分析中 ${Math.floor(elapsedSec / 60)}分${elapsedSec % 60}秒`
              : '🚀 开始 AI 提取'}
          </button>
        </div>
      </div>
      <style>{`@keyframes spin { to { transform: rotate(360deg) } }`}</style>
    </div>
  )
}

/**
 * 3D 互动单页课件视图 — ThreeDSingleView.tsx
 *
 * v0.42.11 新增模块。当课件 source_type === '3d_single' 时由 CoursewareWorkshopPage
 * 早返回到此组件，跳过标准六步流程，使用简化版"一键生成 → 预览 → 确认"工作流。
 *
 * 三种 UI 状态：
 *   1. 未生成（status=generating 且 html_content 为空）— 显示主题信息卡 + 大按钮
 *   2. 生成中（SSE 进行中）— 显示动画 emoji + 实时进度文案 + 计时器
 *   3. 已生成（status=preview 或 confirmed）— iframe 内嵌预览 + 工具栏 + 全屏放映
 *
 * 关键技术差异（vs 标准课件预览）：
 *   - iframe sandbox 含 allow-same-origin（Three.js ESM 模块需要 CDN 加载权限）
 *   - 不注入 injectPreviewMode 降级脚本（3D 不调用 edu 平台 API）
 *   - 全屏放映用浏览器原生 requestFullscreen，独立于 SlideshowPlayer
 *   - 确认完成用 apiClient.post 直调，不依赖 coursewares.ts 是否导出 confirmCourseware
 *
 * 后端依赖：
 *   - POST /api/v1/coursewares/{id}/generate-3d-page  触发异步生成
 *   - POST /api/v1/coursewares/{id}/confirm           确认提交 preview→confirmed
 *   - GET  /api/v1/sse/courseware/{id}                SSE 接收生成进度
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  getCourseware,
  generate3DPage,
  subscribeCWIndexSSE,
  CW_STATUS_CONFIG,
} from '@/api/coursewares'
import type { CoursewareDetail, CoursewarePage } from '@/api/coursewares'
import apiClient from '@/api/client'

// ==================== 局部常量 ====================
const C = {
  primary: '#F59E0B',
  primaryBg: 'rgba(245,158,11,0.08)',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  success: '#059669',
  white: '#fff',
  danger: '#EF4444',
  threeDColor: '#EF4444',   // 3D 主题色（红橙渐变锚点）
  threeDBg: 'rgba(239,68,68,0.08)',
}

const CW_WIDTH = 1920    // 课件画布宽度（与标准课件一致）
const CW_HEIGHT = 1080   // 课件画布高度
const PREVIEW_WIDTH = 1100  // 内嵌预览宽度，按 1100/1920 比例缩放高度

// ==================== 3D 全屏预览子组件 ====================
/**
 * ThreeDFullscreenViewer — 全屏放映 3D 课件
 *
 * 使用浏览器原生 requestFullscreen API 进入全屏，iframe 占满整个屏幕。
 * 区别于 SlideshowPlayer 的关键：
 *   1. sandbox 含 allow-same-origin（Three.js ESM 需要）
 *   2. 不需要多页导航/页码指示器（只有 1 页）
 *   3. 不注入预览降级脚本（3D 不依赖 edu API）
 */
function ThreeDFullscreenViewer({ html, title, onClose }: {
  html: string
  title: string
  onClose: () => void
}) {
  const boxRef = useRef<HTMLDivElement>(null)

  // 进入全屏 + 监听退出（ESC 键 + fullscreenchange 事件）
  useEffect(() => {
    const el = boxRef.current
    if (el?.requestFullscreen) el.requestFullscreen().catch(() => {})

    const onFs = () => { if (!document.fullscreenElement) onClose() }
    const onEsc = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }

    document.addEventListener('fullscreenchange', onFs)
    document.addEventListener('keydown', onEsc)
    return () => {
      document.removeEventListener('fullscreenchange', onFs)
      document.removeEventListener('keydown', onEsc)
      if (document.fullscreenElement) document.exitFullscreen().catch(() => {})
    }
  }, [onClose])

  return (
    <div ref={boxRef} style={{
      position: 'fixed', inset: 0, zIndex: 99999, background: '#000',
    }}>
      <iframe
        srcDoc={html}
        style={{ width: '100%', height: '100%', border: 'none', display: 'block' }}
        sandbox="allow-scripts allow-same-origin"
        title={title}
      />
      {/* 右上角退出按钮（半透明圆形） */}
      <button onClick={onClose}
        style={{
          position: 'absolute', top: 20, right: 20, zIndex: 100000,
          width: 44, height: 44, borderRadius: '50%', border: 'none',
          background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(12px)',
          color: '#fff', fontSize: 20, cursor: 'pointer',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          transition: 'all 200ms',
        }}
        onMouseEnter={e => { e.currentTarget.style.background = 'rgba(239,68,68,0.8)' }}
        onMouseLeave={e => { e.currentTarget.style.background = 'rgba(0,0,0,0.6)' }}
        title="退出 (ESC)">✕</button>
      {/* 底部提示条 */}
      <div style={{
        position: 'absolute', bottom: 20, left: '50%', transform: 'translateX(-50%)',
        padding: '8px 20px', borderRadius: 999, fontSize: 12,
        background: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(10px)',
        color: 'rgba(255,255,255,0.8)',
      }}>按 ESC 退出 · 鼠标拖拽旋转视角 · 滚轮缩放</div>
    </div>
  )
}

// ==================== 主视图组件 ====================
/**
 * ThreeDSingleView — 3D 单页课件主工作台
 *
 * Props:
 *   initialCourseware — 父组件 loadCourseware 后的初始课件数据，本组件接管后续刷新
 */
export default function ThreeDSingleView({ initialCourseware }: {
  initialCourseware: CoursewareDetail
}) {
  const navigate = useNavigate()
  const coursewareId = initialCourseware.id

  // ---- 课件主数据 ----
  const [courseware, setCourseware] = useState<CoursewareDetail>(initialCourseware)
  const [pages, setPages] = useState<CoursewarePage[]>(initialCourseware.pages || [])

  // ---- 生成相关状态 ----
  const [generating, setGenerating] = useState(false)
  const [genMessage, setGenMessage] = useState('')
  const [elapsed, setElapsed] = useState(0)  // 生成耗时秒数

  // ---- 操作相关状态 ----
  const [confirming, setConfirming] = useState(false)
  const [showFullscreen, setShowFullscreen] = useState(false)

  // ---- Refs ----
  const sseRef = useRef<{ close: () => void } | null>(null)
  const elapsedTimerRef = useRef<number | null>(null)

  // ---- 派生数据 ----
  // 3D 单页只有 page1，其他都是空
  const page1 = pages[0]
  const html = page1?.html_content || ''
  const hasGenerated = !!html
  const isPreview = courseware.status === 'preview'
  const isConfirmed = courseware.status === 'confirmed' || courseware.status === 'in_pipeline'

  // ==================== 工具函数 ====================
  // 重新加载课件数据（SSE 完成后 + 确认后调用）
  const reload = useCallback(async () => {
    try {
      const d = await getCourseware(coursewareId)
      setCourseware(d)
      setPages(d.pages || [])
    } catch { /* 忽略刷新失败，保持现有 state */ }
  }, [coursewareId])

  // ==================== 生成耗时计时器 ====================
  useEffect(() => {
    if (generating) {
      const t0 = Date.now()
      const timer = window.setInterval(() => {
        setElapsed(Math.floor((Date.now() - t0) / 1000))
      }, 1000)
      elapsedTimerRef.current = timer
      return () => {
        if (elapsedTimerRef.current) {
          clearInterval(elapsedTimerRef.current)
          elapsedTimerRef.current = null
        }
        setElapsed(0)
      }
    }
  }, [generating])

  // ==================== SSE 兜底轮询 ====================
  // 防止 SSE 中途断开导致用户不知道生成已完成
  // 每 15 秒轮询一次课件状态，发现 status 已变 preview 则自动收尾
  useEffect(() => {
    if (!generating) return
    const timer = window.setInterval(async () => {
      try {
        const d = await getCourseware(coursewareId)
        if (d.status === 'preview' || d.status === 'confirmed') {
          setGenerating(false)
          setCourseware(d)
          setPages(d.pages || [])
          setGenMessage('🎉 3D 课件已生成完成')
          sseRef.current?.close()
        }
      } catch { /* 轮询失败静默 */ }
    }, 15000)
    return () => clearInterval(timer)
  }, [generating, coursewareId])

  // ==================== 卸载清理 ====================
  useEffect(() => {
    return () => sseRef.current?.close()
  }, [])

  // ==================== 启动 3D 生成 ====================
  const handleStartGenerate = useCallback(async () => {
    if (generating) return
    setGenerating(true)
    setGenMessage('⏳ 正在启动 AI 生成任务...')
    try {
      // 1. 触发后端异步生成（后端 800ms 延迟启动等待 SSE 连接）
      await generate3DPage(coursewareId)

      // 2. 订阅 SSE 接收实时进度
      sseRef.current?.close()
      sseRef.current = subscribeCWIndexSSE(coursewareId, {
        onConnected: () => setGenMessage('✅ 已连接 AI 服务，开始构思 3D 场景...'),
        onGenStart: d => setGenMessage(d.message || '🎨 AI 正在生成 3D 单页课件...'),
        onGenProgress: d => setGenMessage(d.message || '🎨 AI 正在构建 3D 场景、粒子系统和交互逻辑...'),
        onGenPage: d => {
          // 单页 HTML 已就绪，立即更新本地预览（无需等 onGenDone）
          setPages(prev => prev.length > 0
            ? [{ ...prev[0], html_content: d.html_content, title: d.title || prev[0].title }]
            : prev)
          setGenMessage('✅ HTML 内容已就绪，正在保存到数据库...')
        },
        onGenDone: d => {
          setGenerating(false)
          setGenMessage('🎉 ' + (d.message || '3D 互动单页生成完成！'))
          reload()
        },
        onError: d => {
          setGenerating(false)
          setGenMessage('❌ ' + (d.message || '生成失败，请重试'))
        },
      })
    } catch (e) {
      setGenerating(false)
      setGenMessage('❌ 启动失败: ' + (e instanceof Error ? e.message : '未知错误'))
    }
  }, [generating, coursewareId, reload])

  // ==================== 确认完成（preview → confirmed） ====================
  // 直接调 apiClient.post，不依赖 coursewares.ts 是否导出 confirmCourseware 函数
  const handleConfirm = useCallback(async () => {
    if (confirming) return
    if (!window.confirm('确认提交此 3D 课件吗？\n提交后课件将进入已完成状态。')) return
    setConfirming(true)
    try {
      await apiClient.post(`/coursewares/${coursewareId}/confirm`, {})
      await reload()
    } catch (e) {
      alert('确认失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally {
      setConfirming(false)
    }
  }, [confirming, coursewareId, reload])

  // ==================== 渲染辅助 ====================
  const sc = CW_STATUS_CONFIG[courseware.status] || { label: courseware.status, color: '#6B7280', bg: '#F3F4F6' }
  const previewScale = PREVIEW_WIDTH / CW_WIDTH

  // 顶部消息条
  const msgBar = (msg: string) => msg ? (
    <div style={{
      padding: '12px 16px', borderRadius: 8, marginBottom: 16, fontSize: 14,
      background: msg.startsWith('❌') ? '#FEE2E2' : msg.startsWith('🎉') || msg.startsWith('✅') ? '#D1FAE5' : '#EFF6FF',
      color: msg.startsWith('❌') ? '#DC2626' : msg.startsWith('🎉') || msg.startsWith('✅') ? '#059669' : '#2563EB',
    }}>{msg}</div>
  ) : null

  return (
    <div style={{ maxWidth: 1280, margin: '0 auto' }}>
      {/* ============ 顶部信息栏（返回 + 状态徽章 + 3D 标识 + 标题） ============ */}
      <div style={{ marginBottom: 20 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 8, flexWrap: 'wrap' }}>
          <button onClick={() => navigate('/courseware')}
            style={{ background: 'none', border: 'none', fontSize: 14, color: C.textSecondary, cursor: 'pointer' }}>
            ← 返回列表
          </button>
          <span style={{
            padding: '2px 10px', borderRadius: 12, fontSize: 12, fontWeight: 500,
            color: sc.color, background: sc.bg,
          }}>{sc.label}</span>
          <span style={{
            padding: '2px 10px', borderRadius: 12, fontSize: 12, fontWeight: 600,
            color: C.threeDColor, background: C.threeDBg, border: `1px solid ${C.threeDColor}`,
          }}>🎮 3D 互动单页</span>
        </div>
        <h2 style={{ fontSize: 22, fontWeight: 700, color: C.textPrimary, margin: 0 }}>{courseware.title}</h2>
        <div style={{ fontSize: 13, color: C.textMuted, marginTop: 4 }}>
          📚 {courseware.subject} &nbsp;|&nbsp; 🎓 {courseware.grade}
        </div>
      </div>

      {/* ============ 课件主题信息卡片 ============ */}
      <div style={{
        background: C.white, borderRadius: 12, border: `1px solid ${C.border}`,
        padding: 20, marginBottom: 20,
      }}>
        <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 12, color: C.textPrimary }}>
          📋 课件主题信息
        </div>
        {page1?.purpose && (
          <div style={{ marginBottom: 10, fontSize: 13, color: C.textSecondary, lineHeight: 1.7 }}>
            <strong style={{ color: C.textPrimary }}>主题：</strong>{page1.purpose}
          </div>
        )}
        {page1?.content_summary && (
          <div style={{ fontSize: 13, color: C.textSecondary, lineHeight: 1.7 }}>
            <strong style={{ color: C.textPrimary }}>详细描述：</strong>{page1.content_summary}
          </div>
        )}
        {!page1?.purpose && !page1?.content_summary && (
          <div style={{ fontSize: 13, color: C.textMuted }}>暂无主题描述</div>
        )}
      </div>

      {/* ============ 主操作区（三状态分支渲染） ============ */}
      <div style={{
        background: C.white, borderRadius: 12, border: `1px solid ${C.border}`,
        padding: 24, minHeight: 400,
      }}>
        {/* ---- 分支1: 未生成（首次进入） ---- */}
        {!generating && !hasGenerated && (
          <div style={{ textAlign: 'center', padding: '40px 20px' }}>
            <div style={{ fontSize: 72, marginBottom: 16 }}>🎮</div>
            <div style={{ fontSize: 20, fontWeight: 600, marginBottom: 8, color: C.textPrimary }}>
              准备生成 3D 互动单页
            </div>
            <div style={{ fontSize: 14, color: C.textSecondary, marginBottom: 24, lineHeight: 1.7 }}>
              AI 将根据上述主题，一次性生成完整的 3D 交互式课件页面<br />
              包含 Three.js 3D 场景、粒子系统、相机过渡、步骤导航等丰富互动
            </div>
            <div style={{
              fontSize: 13, color: '#D97706', marginBottom: 24,
              padding: '12px 20px', borderRadius: 10, background: '#FEF3C7',
              display: 'inline-block',
            }}>
              ⏱️ 预计耗时 1-3 分钟（AI 需要生成 60KB+ 的完整 3D HTML）
            </div>
            <div>
              <button onClick={handleStartGenerate}
                style={{
                  padding: '16px 48px', borderRadius: 12, border: 'none',
                  background: 'linear-gradient(135deg, #EF4444, #DC2626)',
                  color: '#fff', fontSize: 17, fontWeight: 700, cursor: 'pointer',
                  boxShadow: '0 6px 20px rgba(239,68,68,0.4)',
                  transition: 'all 200ms',
                }}
                onMouseEnter={e => {
                  e.currentTarget.style.transform = 'translateY(-2px)'
                  e.currentTarget.style.boxShadow = '0 10px 28px rgba(239,68,68,0.5)'
                }}
                onMouseLeave={e => {
                  e.currentTarget.style.transform = 'translateY(0)'
                  e.currentTarget.style.boxShadow = '0 6px 20px rgba(239,68,68,0.4)'
                }}>
                🎮 开始生成 3D 互动单页
              </button>
            </div>
            {msgBar(genMessage)}
          </div>
        )}

        {/* ---- 分支2: 生成中（SSE 进行中） ---- */}
        {generating && (
          <div style={{ textAlign: 'center', padding: '40px 20px' }}>
            <div style={{
              fontSize: 72, marginBottom: 16, display: 'inline-block',
              animation: 'cw3d-spin 3s linear infinite',
            }}>🎨</div>
            <div style={{ fontSize: 18, fontWeight: 600, marginBottom: 8, color: C.textPrimary }}>
              AI 正在创作 3D 课件
            </div>
            <div style={{ fontSize: 14, color: C.textSecondary, marginBottom: 16, minHeight: 24 }}>
              {genMessage || '正在初始化...'}
            </div>
            <div style={{ fontSize: 13, color: C.textMuted, marginBottom: 8 }}>
              已耗时:&nbsp;
              <strong style={{ color: C.threeDColor, fontSize: 16, fontFamily: 'Monaco, monospace' }}>
                {Math.floor(elapsed / 60)}:{String(elapsed % 60).padStart(2, '0')}
              </strong>
            </div>
            {/* 不确定型进度条 — 流动光带动画 */}
            <div style={{
              maxWidth: 400, margin: '20px auto', height: 8,
              background: '#F3F4F6', borderRadius: 4, overflow: 'hidden', position: 'relative',
            }}>
              <div style={{
                position: 'absolute', top: 0, bottom: 0, width: '40%',
                background: 'linear-gradient(90deg, transparent, #EF4444, transparent)',
                animation: 'cw3d-wave 2s ease-in-out infinite',
              }} />
            </div>
            <div style={{
              fontSize: 12, color: C.textMuted, marginTop: 24,
              padding: '8px 16px', background: '#F9FAFB', borderRadius: 8, display: 'inline-block',
            }}>
              💡 生成期间请保持页面打开。完成后将自动切换到预览模式。
            </div>
            <style>{`
              @keyframes cw3d-spin {
                from { transform: rotate(0deg); }
                to { transform: rotate(360deg); }
              }
              @keyframes cw3d-wave {
                0% { left: -40%; }
                100% { left: 100%; }
              }
            `}</style>
          </div>
        )}

        {/* ---- 分支3: 已生成（预览 + 操作） ---- */}
        {!generating && hasGenerated && (
          <div>
            {msgBar(genMessage)}

            {/* 工具栏 */}
            <div style={{
              display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              marginBottom: 16, flexWrap: 'wrap', gap: 10,
            }}>
              <div style={{
                fontSize: 15, fontWeight: 600, color: C.textPrimary,
                display: 'flex', alignItems: 'center', gap: 8,
              }}>
                🎮 3D 课件预览
                {isConfirmed && (
                  <span style={{
                    padding: '2px 10px', borderRadius: 12, fontSize: 11, fontWeight: 600,
                    color: C.success, background: 'rgba(5,150,105,0.1)',
                  }}>✓ 已确认</span>
                )}
              </div>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {!isConfirmed && (
                  <button onClick={handleStartGenerate}
                    style={{
                      padding: '8px 16px', borderRadius: 8,
                      border: `1px solid ${C.threeDColor}`,
                      background: C.threeDBg, color: C.threeDColor,
                      fontSize: 13, fontWeight: 600, cursor: 'pointer',
                    }}>🔄 重新生成</button>
                )}
                <button onClick={() => setShowFullscreen(true)}
                  style={{
                    padding: '8px 16px', borderRadius: 8,
                    border: '1px solid #7C3AED',
                    background: 'rgba(124,58,237,0.06)', color: '#7C3AED',
                    fontSize: 13, fontWeight: 600, cursor: 'pointer',
                  }}>🖥️ 全屏放映</button>
                <button onClick={() => {
                  navigator.clipboard.writeText(html)
                    .then(() => alert('完整 HTML 已复制到剪贴板'))
                    .catch(() => alert('复制失败，请重试'))
                }}
                  style={{
                    padding: '8px 16px', borderRadius: 8,
                    border: `1px solid ${C.border}`,
                    background: 'transparent', color: C.textSecondary,
                    fontSize: 13, fontWeight: 600, cursor: 'pointer',
                  }}>📋 复制 HTML</button>
                {isPreview && !isConfirmed && (
                  <button onClick={handleConfirm} disabled={confirming}
                    style={{
                      padding: '8px 20px', borderRadius: 8, border: 'none',
                      background: confirming ? '#E5E7EB' : 'linear-gradient(135deg, #059669, #10B981)',
                      color: confirming ? '#9CA3AF' : '#fff',
                      fontSize: 13, fontWeight: 700,
                      cursor: confirming ? 'default' : 'pointer',
                      boxShadow: confirming ? 'none' : '0 2px 8px rgba(5,150,105,0.3)',
                    }}>
                    {confirming ? '⏳ 确认中...' : '✅ 确认完成'}
                  </button>
                )}
              </div>
            </div>

            {/* 内嵌预览容器 — 按宽度缩放 */}
            <div style={{
              width: PREVIEW_WIDTH, maxWidth: '100%',
              height: Math.ceil(CW_HEIGHT * previewScale),
              borderRadius: 12, border: `1px solid ${C.border}`,
              overflow: 'hidden', background: '#000',
              position: 'relative', margin: '0 auto',
            }}>
              <iframe
                srcDoc={html}
                style={{
                  width: CW_WIDTH, height: CW_HEIGHT,
                  border: 'none', display: 'block',
                  transform: `scale(${previewScale})`,
                  transformOrigin: 'top left',
                }}
                sandbox="allow-scripts allow-same-origin"
                title="3D 课件预览"
              />
            </div>

            {/* 操作提示 */}
            <div style={{
              marginTop: 16, padding: '12px 16px', borderRadius: 10,
              background: '#EFF6FF', color: '#2563EB', fontSize: 13, lineHeight: 1.7,
            }}>
              💡 3D 课件需要从 CDN 加载 Three.js 库，首次预览可能需要 3-5 秒初始化。<br />
              建议点击 <strong>「🖥️ 全屏放映」</strong> 体验完整的 3D 交互效果（鼠标拖拽旋转 · 滚轮缩放 · 按键导航）。
            </div>

            {/* HTML 元信息 */}
            <div style={{
              marginTop: 12, padding: '8px 16px', borderRadius: 8,
              background: '#F9FAFB', fontSize: 12, color: C.textMuted,
              display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              flexWrap: 'wrap', gap: 8,
            }}>
              <span>📊 HTML 大小: <strong>{(html.length / 1024).toFixed(1)} KB</strong>
                &nbsp;·&nbsp;行数: <strong>{html.split('\n').length}</strong></span>
              <span>当前状态: <strong style={{ color: sc.color }}>{sc.label}</strong></span>
            </div>
          </div>
        )}
      </div>

      {/* ============ 全屏放映弹层 ============ */}
      {showFullscreen && hasGenerated && (
        <ThreeDFullscreenViewer
          html={html}
          title={`3D 放映 - ${courseware.title}`}
          onClose={() => setShowFullscreen(false)}
        />
      )}
    </div>
  )
}

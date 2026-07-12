/**
 * 课件审核独立全屏工作台 — CWReviewWorkbenchPage.tsx（阶段3 · 弹窗改页面）
 *
 * 取代原 CWReviewDecisionModal 弹窗。痛点：弹窗里 iframe 未做等比缩放，课件 1920×1080
 * 被塞进小盒子只露左上角一小块，审核员看不全无法审核；底部胶片条又挤又难翻。
 *
 * 本页镜像教案审核 ReviewWorkbenchPage 的独立全屏范式（脱离 CWLayout，挂
 * /courseware/review/:id，注册在 /courseware 布局路由之前），并借鉴 CWFullscreenPreview
 * 的「等比 scale 缩放铺满容器」算法，让课件页完整呈现不截断。
 *
 * 布局（全屏三段）：
 *   顶栏（48px）：返回 + 课件标题 + 级别徽章
 *   主体：左栏大预览（flex:1，scale 等比缩入 + 上一页/下一页大按钮 + 键盘左右翻页
 *         + 可点页码块带批注角标 + 全屏放映按钮）；右栏（420px）批注/历史 Tab + 决策表单
 *
 * 数据：进页调 getCWReviewDetail 一次拉全（课件详情含 pages + 全部批注 + 审核历史）。
 * 提交：reviewCWL1 / reviewCWL2，成功后 navigate 回 /courseware/review。
 * URL：?level=1|2 决定本次审核级别（L1 教研组 / L2 学校）。
 *
 * 颜色用课件工坊暖色系（橙→红），与教案审核蓝紫系区分。本文件 < 600 行。
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import {
  getCWReviewDetail,
  reviewCWL1,
  reviewCWL2,
  type CWReviewDetailResponse,
  type CWReviewListItem,
  type CoursewareAnnotation,
} from '@/api/coursewares'
import { injectPreviewMode } from '../components/courseware-workshop/previewInject'
import { CW_WIDTH, CW_HEIGHT } from '../components/courseware-workshop/workshopConstants'
import CWFullscreenPreview from '../components/courseware-workshop/CWFullscreenPreview'

// ==================== 样式常量（暖色系） ====================
const C = {
  primary:   '#F59E0B',   // 橙（L1 / 主色）
  danger:    '#EF4444',   // 红（L2 / 退回）
  success:   '#10B981',   // 绿（通过）
  warning:   '#F59E0B',
  text:      '#1F2937',
  textSec:   '#6B7280',
  textMuted: '#9CA3AF',
  border:    '#F3F4F6',
  borderMid: '#E5E7EB',
  card:      '#FFFFFF',
  bg:        '#FAFBFC',
}

const DECISION_LABELS: Record<string, { label: string; color: string; icon: string }> = {
  approved: { label: '通过', color: '#10B981', icon: '✅' },
  revision: { label: '退回', color: '#F59E0B', icon: '↩️' },
  revoked:  { label: '撤回', color: '#EF4444', icon: '🚫' },
}

function formatDateTime(iso: string): string {
  try {
    const d = new Date(iso)
    const p = (n: number) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
  } catch { return iso }
}

type SideTab = 'annotations' | 'history'

// ==================== 主组件 ====================
export default function CWReviewWorkbenchPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  // URL ?level=1|2 决定本次审核级别，缺省按 1（L1）
  const level = searchParams.get('level') === '2' ? 2 : 1

  // —— 详情数据 ——
  const [detail, setDetail] = useState<CWReviewDetailResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadErr, setLoadErr] = useState('')

  // —— 左栏预览 ——
  const [activePage, setActivePage] = useState(1)
  const [showFullscreen, setShowFullscreen] = useState(false)
  // 预览容器实测尺寸（供 scale 计算等比缩入，避免 iframe 截断）
  const previewBoxRef = useRef<HTMLDivElement>(null)
  const [boxSize, setBoxSize] = useState({ w: 0, h: 0 })

  // —— 右栏 Tab + 决策表单 ——
  const [sideTab, setSideTab] = useState<SideTab>('annotations')
  const [decision, setDecision] = useState<'approved' | 'revision'>('approved')
  const [comment, setComment] = useState('')
  const [score, setScore] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const levelColor = level === 1 ? C.primary : C.danger
  const levelLabel = level === 1 ? '📋 L1 教研组审核' : '🏫 L2 学校审核'

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type }); setTimeout(() => setToast(null), 3000)
  }

  // —— 拉取审核详情（课件+批注+历史一次返回） ——
  const loadDetail = useCallback(async () => {
    if (!id) return
    setLoading(true); setLoadErr('')
    try {
      const d = await getCWReviewDetail(id)
      setDetail(d)
      const pages = d.courseware?.pages || []
      if (pages.length > 0) setActivePage(pages[0].page_number)
    } catch (e: unknown) {
      setLoadErr(e instanceof Error ? e.message : '加载课件审核详情失败')
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => { loadDetail() }, [loadDetail])

  // —— 监听预览容器尺寸（窗口 resize 时重算 scale） ——
  useEffect(() => {
    const measure = () => {
      const el = previewBoxRef.current
      if (el) setBoxSize({ w: el.clientWidth, h: el.clientHeight })
    }
    measure()
    window.addEventListener('resize', measure)
    return () => window.removeEventListener('resize', measure)
  }, [loading, detail])

  const pages = detail?.courseware?.pages || []
  const annotations: CoursewareAnnotation[] = detail?.annotations || []
  const reviews: CWReviewListItem[] = detail?.reviews || []
  const cwTitle = detail?.courseware?.title || '课件审核'
  const idx = pages.findIndex(p => p.page_number === activePage)
  const curPage = pages[idx] || pages[0]
  const hasPrev = idx > 0
  const hasNext = idx >= 0 && idx < pages.length - 1

  // 当前页的批注数量（给胶片条角标提示哪些页有批注）
  const annoCountByPage = annotations.reduce<Record<number, number>>((m, a) => {
    m[a.page_number] = (m[a.page_number] || 0) + 1; return m
  }, {})

  // —— 翻页（统一入口，键盘+按钮共用） ——
  const gotoPrev = useCallback(() => { if (hasPrev) setActivePage(pages[idx - 1].page_number) }, [hasPrev, pages, idx])
  const gotoNext = useCallback(() => { if (hasNext) setActivePage(pages[idx + 1].page_number) }, [hasNext, pages, idx])

  // 键盘左右翻页（全屏放映打开时不抢按键）
  useEffect(() => {
    if (showFullscreen) return
    const fn = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA') return  // 输入框内不翻页
      if (e.key === 'ArrowLeft') gotoPrev()
      if (e.key === 'ArrowRight') gotoNext()
    }
    window.addEventListener('keydown', fn)
    return () => window.removeEventListener('keydown', fn)
  }, [showFullscreen, gotoPrev, gotoNext])

  // —— scale 等比缩入：容器实测宽高 / 画布 1920×1080 取最小 ——
  const PAD = 24  // 容器内边距
  const availW = Math.max(0, boxSize.w - PAD * 2)
  const availH = Math.max(0, boxSize.h - PAD * 2)
  const scale = (availW > 0 && availH > 0)
    ? Math.min(availW / CW_WIDTH, availH / CW_HEIGHT)
    : 0
  const previewHtml = curPage?.html_content ? injectPreviewMode(curPage.html_content) : ''

  // —— 提交决策 ——
  const handleSubmit = async () => {
    if (!id) return
    if (!comment.trim()) { showToast('请填写审核意见', 'error'); return }
    setSubmitting(true)
    try {
      const req = { decision, comment: comment.trim(), score: score ? parseFloat(score) : undefined }
      if (level === 1) await reviewCWL1(id, req)
      else await reviewCWL2(id, req)
      showToast(decision === 'approved' ? '✅ 审核通过' : '↩️ 已退回修改')
      setTimeout(() => navigate('/courseware/review'), 1200)
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '审核失败', 'error')
    } finally {
      setSubmitting(false)
    }
  }

  // ==================== 加载/错误态 ====================
  if (loading) {
    return (
      <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: C.bg }}>
        <div style={{ textAlign: 'center', color: C.textMuted }}>
          <div style={{ width: '28px', height: '28px', border: `3px solid ${C.border}`, borderTopColor: C.primary, borderRadius: '50%', animation: 'spin 0.8s linear infinite', margin: '0 auto 12px' }} />
          <div>加载课件中...</div>
          <style>{`@keyframes spin{to{transform:rotate(360deg)}}`}</style>
        </div>
      </div>
    )
  }

  if (loadErr || !detail) {
    return (
      <div style={{ height: '100vh', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: '16px', background: C.bg }}>
        <div style={{ fontSize: '44px' }}>😵</div>
        <div style={{ fontSize: '15px', color: C.text }}>{loadErr || '课件不存在或无权限审核'}</div>
        <div style={{ display: 'flex', gap: '10px' }}>
          <button onClick={loadDetail} style={{ padding: '9px 20px', borderRadius: '8px', border: `1px solid ${C.borderMid}`, background: '#fff', fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>重试</button>
          <button onClick={() => navigate('/courseware/review')} style={{ padding: '9px 20px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>返回审核列表</button>
        </div>
      </div>
    )
  }

  // ==================== 主渲染 ====================
  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', background: C.bg }}>
      {/* 顶部导航条 */}
      <div style={{ height: '48px', background: C.card, borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', padding: '0 20px', gap: '12px', flexShrink: 0, boxShadow: '0 1px 4px rgba(0,0,0,0.06)' }}>
        <button onClick={() => navigate('/courseware/review')}
          style={{ display: 'flex', alignItems: 'center', gap: '5px', background: 'none', border: 'none', fontSize: '13px', color: C.textSec, cursor: 'pointer', padding: '4px 8px', borderRadius: '6px' }}>
          ← 返回列表
        </button>
        <div style={{ width: '1px', height: '16px', background: C.border }} />
        <span style={{ padding: '3px 10px', borderRadius: '8px', background: levelColor + '15', color: levelColor, fontSize: '12px', fontWeight: 600, flexShrink: 0 }}>{levelLabel}</span>
        <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {cwTitle}
        </div>
        <div style={{ fontSize: '12px', color: C.textSec, flexShrink: 0 }}>
          {detail.courseware?.subject} · {detail.courseware?.grade} · {pages.length} 页
        </div>
      </div>

      {/* 主体：左大预览 + 右批注/历史/决策 */}
      <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
        {/* ========== 左栏：课件大预览 ========== */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, borderRight: `1px solid ${C.border}`, background: C.bg }}>
          {pages.length === 0 ? (
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: C.textMuted, fontSize: '14px' }}>该课件暂无已生成页面</div>
          ) : (
            <>
              {/* 翻页工具条 */}
              <div style={{ flexShrink: 0, height: '46px', display: 'flex', alignItems: 'center', gap: '10px', padding: '0 16px', background: C.card, borderBottom: `1px solid ${C.border}` }}>
                <button onClick={gotoPrev} disabled={!hasPrev}
                  style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.borderMid}`, background: '#fff', color: hasPrev ? C.text : C.textMuted, cursor: hasPrev ? 'pointer' : 'not-allowed', fontSize: '14px', opacity: hasPrev ? 1 : 0.5 }}>‹ 上一页</button>
                <span style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>
                  P{curPage?.page_number}{curPage?.title ? ` — ${curPage.title}` : ''}
                </span>
                <span style={{ fontSize: '12px', color: C.textMuted }}>{(idx >= 0 ? idx : 0) + 1}/{pages.length}</span>
                <button onClick={gotoNext} disabled={!hasNext}
                  style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.borderMid}`, background: '#fff', color: hasNext ? C.text : C.textMuted, cursor: hasNext ? 'pointer' : 'not-allowed', fontSize: '14px', opacity: hasNext ? 1 : 0.5 }}>下一页 ›</button>
                <div style={{ flex: 1 }} />
                <span style={{ fontSize: '11px', color: C.textMuted }}>← → 键翻页</span>
                <button onClick={() => setShowFullscreen(true)}
                  style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.primary}`, background: C.primary + '0E', color: C.primary, cursor: 'pointer', fontSize: '13px', fontWeight: 600 }}>🔍 全屏放映</button>
              </div>

              {/* 大图预览：scale 等比缩入容器，完整呈现不截断 */}
              <div ref={previewBoxRef} style={{ flex: 1, position: 'relative', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: `${PAD}px` }}>
                {curPage && previewHtml && scale > 0 ? (
                  <div style={{ width: CW_WIDTH, height: CW_HEIGHT, flexShrink: 0, transform: `scale(${scale})`, transformOrigin: 'center center', borderRadius: '6px', overflow: 'hidden', boxShadow: '0 2px 16px rgba(0,0,0,0.12)', background: '#fff' }}>
                    <iframe
                      title={`cw-review-p${activePage}`}
                      srcDoc={previewHtml}
                      scrolling="no"
                      sandbox="allow-scripts"
                      style={{ width: CW_WIDTH, height: CW_HEIGHT, border: 'none', display: 'block', overflow: 'hidden' }}
                    />
                  </div>
                ) : (
                  <div style={{ color: C.textMuted, fontSize: '13px' }}>该页尚未生成 HTML 内容</div>
                )}
              </div>

              {/* 胶片条（页码块 + 批注角标） */}
              <div style={{ flexShrink: 0, borderTop: `1px solid ${C.border}`, padding: '10px 12px', display: 'flex', gap: '8px', overflowX: 'auto', background: C.card }}>
                {pages.map(p => {
                  const isCur = p.page_number === activePage
                  const ac = annoCountByPage[p.page_number] || 0
                  return (
                    <button key={p.page_number} onClick={() => setActivePage(p.page_number)}
                      style={{ position: 'relative', flexShrink: 0, padding: '6px 14px', borderRadius: '8px', border: isCur ? `2px solid ${C.primary}` : `1px solid ${C.borderMid}`, background: isCur ? C.primary + '12' : '#fff', cursor: 'pointer', fontSize: '12px', color: isCur ? C.primary : C.textSec, fontWeight: isCur ? 600 : 400, minWidth: '52px' }}>
                      P{p.page_number}
                      {ac > 0 && (
                        <span style={{ position: 'absolute', top: '-6px', right: '-6px', minWidth: '16px', height: '16px', padding: '0 4px', borderRadius: '8px', background: C.danger, color: '#fff', fontSize: '10px', fontWeight: 700, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>{ac}</span>
                      )}
                    </button>
                  )
                })}
              </div>
            </>
          )}
        </div>

        {/* ========== 右栏：批注/历史 + 决策表单 ========== */}
        <div style={{ width: '420px', flexShrink: 0, display: 'flex', flexDirection: 'column', minHeight: 0, background: C.card }}>
          {/* 右栏上半：Tab + 列表 */}
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, borderBottom: `1px solid ${C.border}` }}>
            <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, padding: '0 8px', flexShrink: 0 }}>
              {([
                { key: 'annotations' as SideTab, label: `💬 批注 ${annotations.length}` },
                { key: 'history' as SideTab, label: `📜 审核历史 ${reviews.length}` },
              ]).map(t => {
                const isAct = sideTab === t.key
                return (
                  <button key={t.key} onClick={() => setSideTab(t.key)}
                    style={{ padding: '12px 14px', border: 'none', background: 'transparent', fontSize: '13px', fontWeight: isAct ? 600 : 400, color: isAct ? C.primary : C.textSec, cursor: 'pointer', borderBottom: isAct ? `2px solid ${C.primary}` : '2px solid transparent', marginBottom: '-1px' }}>
                    {t.label}
                  </button>
                )
              })}
            </div>

            <div style={{ flex: 1, overflowY: 'auto', padding: '12px 16px', minHeight: 0 }}>
              {/* 批注列表（决策二：边看批注边决策，点批注跳到对应页） */}
              {sideTab === 'annotations' && (
                <>
                  {annotations.length === 0 && (
                    <div style={{ padding: '40px 0', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>
                      <div style={{ fontSize: '28px', marginBottom: '8px' }}>💬</div>
                      该课件暂无批注
                    </div>
                  )}
                  {annotations.map(a => {
                    const isCurPageAnno = a.page_number === activePage
                    const resolved = a.status === 'resolved'
                    return (
                      <div key={a.id} onClick={() => setActivePage(a.page_number)}
                        style={{ padding: '10px 12px', marginBottom: '8px', borderRadius: '10px', border: isCurPageAnno ? `1px solid ${C.primary}40` : `1px solid ${C.border}`, background: isCurPageAnno ? C.primary + '08' : '#fff', cursor: 'pointer' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '4px' }}>
                          <span style={{ padding: '1px 7px', borderRadius: '8px', background: C.primary + '15', color: C.primary, fontSize: '11px', fontWeight: 600 }}>P{a.page_number}</span>
                          <span style={{ fontSize: '12px', color: C.textSec, fontWeight: 500 }}>{a.reviewer_name || '匿名'}</span>
                          {resolved && <span style={{ fontSize: '11px', color: C.success }}>✓ 已处理</span>}
                          <span style={{ marginLeft: 'auto', fontSize: '11px', color: C.textMuted }}>{formatDateTime(a.created_at)}</span>
                        </div>
                        <div style={{ fontSize: '13px', color: resolved ? C.textMuted : C.text, lineHeight: 1.5, textDecoration: resolved ? 'line-through' : 'none', wordBreak: 'break-word' }}>
                          {a.content}
                        </div>
                      </div>
                    )
                  })}
                </>
              )}

              {/* 审核历史列表 */}
              {sideTab === 'history' && (
                <>
                  {reviews.length === 0 && (
                    <div style={{ padding: '40px 0', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>
                      <div style={{ fontSize: '28px', marginBottom: '8px' }}>📜</div>
                      暂无历史审核记录
                    </div>
                  )}
                  {reviews.map(rv => {
                    const dCfg = DECISION_LABELS[rv.decision] || DECISION_LABELS.approved
                    return (
                      <div key={rv.id} style={{ padding: '10px 12px', marginBottom: '8px', borderRadius: '10px', border: `1px solid ${C.border}`, background: '#fff' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '4px', flexWrap: 'wrap' }}>
                          <span style={{ padding: '1px 7px', borderRadius: '8px', background: (rv.review_level === 1 ? C.primary : C.danger) + '15', color: rv.review_level === 1 ? C.primary : C.danger, fontSize: '11px', fontWeight: 600 }}>{rv.level_name}</span>
                          <span style={{ padding: '1px 7px', borderRadius: '8px', background: dCfg.color + '15', color: dCfg.color, fontSize: '11px', fontWeight: 600 }}>{dCfg.icon} {dCfg.label}</span>
                          <span style={{ fontSize: '12px', color: C.textSec }}>{rv.reviewer_name}</span>
                          {rv.score != null && <span style={{ fontSize: '12px', color: C.primary, fontWeight: 600 }}>⭐ {rv.score.toFixed(1)}</span>}
                          <span style={{ marginLeft: 'auto', fontSize: '11px', color: C.textMuted }}>{formatDateTime(rv.created_at)}</span>
                        </div>
                        {rv.comment && (
                          <div style={{ fontSize: '13px', color: C.textSec, lineHeight: 1.5, wordBreak: 'break-word' }}>💬 {rv.comment}</div>
                        )}
                      </div>
                    )
                  })}
                </>
              )}
            </div>
          </div>

          {/* 右栏下半：决策表单 */}
          <div style={{ flexShrink: 0, padding: '16px', background: C.bg }}>
            {/* 决策选择 */}
            <div style={{ display: 'flex', gap: '10px', marginBottom: '12px' }}>
              {[
                { value: 'approved' as const, label: '✅ 通过', color: C.success },
                { value: 'revision' as const, label: '↩️ 退回修改', color: C.warning },
              ].map(opt => (
                <button key={opt.value} onClick={() => setDecision(opt.value)}
                  style={{ flex: 1, padding: '10px', borderRadius: '10px', border: decision === opt.value ? `2px solid ${opt.color}` : `1px solid ${C.borderMid}`, background: decision === opt.value ? opt.color + '10' : '#fff', cursor: 'pointer', fontSize: '14px', fontWeight: decision === opt.value ? 600 : 400, color: decision === opt.value ? opt.color : C.textSec, transition: 'all 150ms' }}>
                  {opt.label}
                </button>
              ))}
            </div>
            {/* 评分（可选） */}
            <div style={{ marginBottom: '12px' }}>
              <input type="number" min="1" max="10" step="0.5" value={score} onChange={e => setScore(e.target.value)}
                placeholder="评分（可选，1-10）"
                style={{ width: '100%', padding: '9px 12px', borderRadius: '10px', border: `1px solid ${C.borderMid}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
            </div>
            {/* 审核意见 */}
            <textarea value={comment} onChange={e => setComment(e.target.value)}
              placeholder={decision === 'approved' ? '课件整体质量良好，可通过…' : '请说明需要修改的地方…'}
              rows={3}
              style={{ width: '100%', padding: '10px 12px', borderRadius: '10px', border: `1px solid ${C.borderMid}`, fontSize: '14px', outline: 'none', resize: 'vertical', boxSizing: 'border-box', fontFamily: 'inherit', marginBottom: '12px' }} />
            {/* 提交按钮 */}
            <div style={{ display: 'flex', gap: '10px' }}>
              <button onClick={() => navigate('/courseware/review')} style={{ padding: '10px 18px', borderRadius: '10px', border: `1px solid ${C.borderMid}`, background: '#fff', cursor: 'pointer', fontSize: '14px', color: C.textSec, flexShrink: 0 }}>取消</button>
              <button onClick={handleSubmit} disabled={submitting}
                style={{ flex: 1, padding: '10px', borderRadius: '10px', border: 'none', background: decision === 'approved' ? C.success : C.warning, color: '#fff', cursor: submitting ? 'not-allowed' : 'pointer', fontSize: '14px', fontWeight: 600, opacity: submitting ? 0.6 : 1 }}>
                {submitting ? '提交中…' : (decision === 'approved' ? '✅ 确认通过' : '↩️ 确认退回')}
              </button>
            </div>
            <p style={{ margin: '8px 0 0', fontSize: '11px', color: C.textMuted, textAlign: 'center' }}>
              {level === 1 ? 'L1 通过后若学校开启 L2 将进入学校审核' : 'L2 通过后课件进入"待发布"，作者可共享'}
            </p>
          </div>
        </div>
      </div>

      {/* 全屏放映（复用工坊全屏预览组件，逐页放大审核） */}
      {showFullscreen && pages.length > 0 && (
        <CWFullscreenPreview
          pages={pages.map(p => ({ page_number: p.page_number, title: p.title || '', html_content: p.html_content || '' }))}
          initialPageNum={activePage}
          codeView={false}
          onToggleCode={() => {}}
          onClose={(finalPage) => { if (finalPage) setActivePage(finalPage); setShowFullscreen(false) }}
          onSlideshow={() => {}}
        />
      )}

      {/* Toast */}
      {toast && (
        <div style={{ position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)', padding: '12px 24px', borderRadius: '10px', background: toast.type === 'error' ? '#FEF2F2' : '#1F2937', color: toast.type === 'error' ? C.danger : '#fff', fontSize: '14px', fontWeight: 500, boxShadow: '0 8px 24px rgba(0,0,0,0.15)', zIndex: 99999, whiteSpace: 'nowrap', border: toast.type === 'error' ? '1px solid #FECACA' : 'none' }}>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}

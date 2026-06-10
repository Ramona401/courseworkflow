/**
 * KBCurriculumPage.tsx — 知识库课标压缩入库系统主页面（隐藏全屏）
 *
 * 路由：/kb-admin/curriculum（脱离任何 Layout，仅 AuthGuard；不进门户入口卡片）。
 * 真正的访问拦截靠后端 RequireKBAuthorized 白名单中间件——非白名单用户调任一 KB 接口返 403，
 * 本页捕获后显示「无权访问」友好提示。前端无守卫，仅体验优化非安全边界。
 *
 * 两视图：
 *   - manage（任务管理）：KBUploadForm 上传区 + KBJobList 任务列表（含 SSE 实时进度）。
 *   - review（审核）    ：KBReviewPanel（解码人话 + 三选一 + 入库 + 蓝绿切换）。
 *
 * SSE 时序编排（关键，PRD 强调）：
 *   后端 CreateJob 返回 job_id 后 go func + 800ms 延迟才真正跑压缩，需前端先连上 SSE。
 *   故本页：createKBJob 拿到 job_id 后立即 subscribeKBJobSSE，进度映射为 KBActiveProgress 传 KBJobList。
 *   事件到阶段的映射：extract 系列事件归 extract 阶段；item 系列与 progress 归 compress 阶段并更新计数；
 *   job_done 归 done 阶段并刷新列表；error 归 error 阶段。
 *   组件卸载时关闭 SSE 连接防泄漏。
 *
 * 字段名对齐后端：createKBJob 返回 {job_id}（非 id），listKBJobs 返回 {jobs}（非 items）。
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { C, Toast } from './components/kbConstants'
import { KBUploadForm } from './components/KBUploadForm'
import { KBJobList, type KBActiveProgress } from './components/KBJobList'
import { KBReviewPanel } from './KBReviewPanel'
import {
  createKBJob, listKBJobs, subscribeKBJobSSE,
  type KBCompressJob, type KBCreateJobRequest,
} from '@/api/kb'

type ViewMode =
  | { mode: 'manage' }
  | { mode: 'review'; job: KBCompressJob }

export default function KBCurriculumPage() {
  const navigate = useNavigate()

  const [view, setView] = useState<ViewMode>({ mode: 'manage' })

  // 任务列表
  const [jobs, setJobs] = useState<KBCompressJob[]>([])
  const [jobsLoading, setJobsLoading] = useState(false)
  const [forbidden, setForbidden] = useState(false) // 后端 403 兜底

  // 创建中 + SSE 实时进度
  const [creating, setCreating] = useState(false)
  const [activeProgress, setActiveProgress] = useState<KBActiveProgress | null>(null)
  const sseRef = useRef<{ close: () => void } | null>(null)

  // Toast
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)
  const showToast = useCallback((message: string, type: 'success' | 'error') => setToast({ message, type }), [])

  // ---- 加载任务列表（后端返回 {jobs, total}） ----
  const loadJobs = useCallback(async () => {
    try {
      setJobsLoading(true)
      const data = await listKBJobs({ kind: 'curriculum' })
      setJobs(data.jobs || [])
      setForbidden(false)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '加载任务失败'
      // 后端白名单拦截返 403（拦截器转成 message）；用关键词兜底识别
      if (/403|无权|权限|forbidden/i.test(msg)) {
        setForbidden(true)
      } else {
        showToast(msg, 'error')
      }
    } finally {
      setJobsLoading(false)
    }
  }, [showToast])

  useEffect(() => { loadJobs() }, [loadJobs])

  // ---- 组件卸载时关闭 SSE ----
  useEffect(() => {
    return () => { sseRef.current?.close() }
  }, [])

  // ---- 创建任务 + 订阅 SSE ----
  const handleCreate = useCallback(async (req: KBCreateJobRequest) => {
    // 关闭上一条可能残留的 SSE
    sseRef.current?.close()
    sseRef.current = null
    setActiveProgress(null)

    try {
      setCreating(true)
      // 后端返回 {job_id, message}，解构 job_id（之前误写 id 导致 SSE URL 变 undefined）
      const { job_id } = await createKBJob(req)

      // 立即订阅 SSE（后端 800ms 后才跑压缩，此处先连上）
      const initProgress: KBActiveProgress = {
        jobId: job_id, phase: 'extract', doneItems: 0, totalItems: 0,
        message: '任务已创建，正在准备压缩...',
      }
      setActiveProgress(initProgress)

      sseRef.current = subscribeKBJobSSE(job_id, {
        onConnected: () => {
          setActiveProgress(p => p && p.jobId === job_id ? { ...p, message: '已连接，等待开始...' } : p)
        },
        onExtractStart: () => {
          setActiveProgress(p => p && p.jobId === job_id ? { ...p, phase: 'extract', message: '正在通读原文、识别知识点...' } : p)
        },
        onExtractDone: (d) => {
          const total = d.total_items ?? 0
          setActiveProgress(p => p && p.jobId === job_id ? { ...p, phase: 'compress', totalItems: total, message: `识别到 ${total} 个知识点，开始逐个压缩` } : p)
        },
        onItemStart: (d) => {
          setActiveProgress(p => p && p.jobId === job_id ? {
            ...p, phase: 'compress',
            totalItems: d.total ?? p.totalItems,
            message: `正在压缩第 ${d.seq ?? '?'} 个知识点...`,
          } : p)
        },
        onItemDone: (d) => {
          setActiveProgress(p => p && p.jobId === job_id ? {
            ...p, phase: 'compress',
            doneItems: d.seq ?? p.doneItems,
            totalItems: d.total ?? p.totalItems,
            message: `第 ${d.seq ?? '?'} 个完成（${d.confidence === 'low' ? '低置信，待人工' : '高置信，自动通过'}）`,
          } : p)
        },
        onProgress: (d) => {
          setActiveProgress(p => p && p.jobId === job_id ? {
            ...p, phase: 'compress',
            doneItems: d.done_items ?? p.doneItems,
            totalItems: d.total_items ?? p.totalItems,
            message: d.message ?? p.message,
          } : p)
        },
        onJobDone: (d) => {
          setActiveProgress(p => p && p.jobId === job_id ? {
            ...p, phase: 'done',
            doneItems: p.totalItems || (d.total_items ?? 0),
            totalItems: d.total_items ?? p.totalItems,
            message: `压缩完成：共 ${d.total_items ?? '?'} 项，自动通过 ${d.auto_passed ?? '?'}，待人工 ${d.need_review ?? '?'}`,
          } : p)
          showToast('压缩完成，可进入审核', 'success')
          loadJobs() // 刷新列表拿到最新状态
        },
        onError: (d) => {
          setActiveProgress(p => p && p.jobId === job_id ? { ...p, phase: 'error', message: d.message || '压缩出错' } : p)
          showToast(d.message || '压缩过程出错', 'error')
        },
      })

      showToast('任务已创建，开始压缩', 'success')
      // 立即刷新一次列表，让新任务出现在列表里
      loadJobs()
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '创建任务失败'
      setActiveProgress(null)
      if (/403|无权|权限|forbidden/i.test(msg)) setForbidden(true)
      else showToast(msg, 'error')
    } finally {
      setCreating(false)
    }
  }, [loadJobs, showToast])

  // ---- 进入审核 ----
  const handleEnterReview = useCallback((job: KBCompressJob) => {
    setView({ mode: 'review', job })
  }, [])

  // ==================== 无权访问兜底 ====================
  if (forbidden) {
    return (
      <div style={{ minHeight: '100vh', background: C.bg, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div style={{ textAlign: 'center', maxWidth: '420px', padding: '0 20px' }}>
          <div style={{ fontSize: '46px', marginBottom: '14px' }}>🔒</div>
          <div style={{ fontSize: '18px', fontWeight: 700, color: C.text, marginBottom: '8px' }}>无权访问</div>
          <div style={{ fontSize: '13px', color: C.textSec, lineHeight: 1.7, marginBottom: '20px' }}>
            知识库压缩入库系统为受限功能，仅授权人员可用。如需访问，请联系系统管理员将你加入白名单。
          </div>
          <button onClick={() => navigate('/')}
            style={{ padding: '9px 24px', borderRadius: '9px', border: 'none', background: `linear-gradient(135deg,${C.primary},${C.purple})`, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>
            返回首页
          </button>
        </div>
      </div>
    )
  }

  // ==================== 正常渲染 ====================
  return (
    <div style={{ minHeight: '100vh', background: 'linear-gradient(135deg,#EEF2FF 0%,#FAFBFC 50%,#F0FDF4 100%)' }}>
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      {/* 顶部栏 */}
      <header style={{ height: '60px', position: 'sticky', top: 0, zIndex: 100, background: 'rgba(255,255,255,0.9)', backdropFilter: 'blur(20px)', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', padding: '0 28px', gap: '16px' }}>
        {view.mode === 'review' ? (
          <button onClick={() => setView({ mode: 'manage' })}
            style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>
            {'← 返回任务列表'}
          </button>
        ) : (
          <button onClick={() => navigate('/')}
            style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}>
            {'← 返回首页'}
          </button>
        )}
        <div style={{ flex: 1 }}>
          <h1 style={{ fontSize: '17px', fontWeight: 700, color: C.text, margin: 0 }}>
            📚 知识库课标压缩入库
            {view.mode === 'review' && (
              <span style={{ fontSize: '13px', fontWeight: 400, color: C.textMuted, marginLeft: '12px' }}>
                审核：{view.job.subject}{view.job.grade_num > 0 ? ` ${view.job.grade_num}年级` : ''} · 批次 {view.job.batch_tag}
              </span>
            )}
          </h1>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '1px' }}>
            课标原文经多轮压缩 + 语义一致性仲裁，人工审核后灌入知识库（隐藏功能）
          </div>
        </div>
      </header>

      <div style={{ maxWidth: '1280px', margin: '0 auto', padding: '24px' }}>
        {view.mode === 'manage' ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <KBUploadForm creating={creating} onCreate={handleCreate} onError={(m) => showToast(m, 'error')} />
            <KBJobList
              jobs={jobs}
              loading={jobsLoading}
              activeProgress={activeProgress}
              onRefresh={loadJobs}
              onEnterReview={handleEnterReview}
            />
          </div>
        ) : (
          <KBReviewPanel
            jobId={view.job.id}
            batchTag={view.job.batch_tag}
            onError={(m) => showToast(m, 'error')}
            onSuccess={(m) => showToast(m, 'success')}
          />
        )}
      </div>
    </div>
  )
}

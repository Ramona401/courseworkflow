/**
 * KBJobList.tsx — 课标压缩任务列表 + 实时进度
 *
 * 受控组件：任务数组 / 当前活跃任务的 SSE 实时进度 / loading 态均由父页面传入；
 * 自身只负责渲染，通过回调把「刷新」「进入审核」事件冒泡给父页面（KBCurriculumPage）。
 *
 * 进度展示（PRD：抽取中 → 逐 item 压缩 → 完成）：
 *   父页面把当前正在压缩的 jobId 与其阶段/计数通过 activeProgress 传入；
 *   命中的任务行下方渲染进度条 + 阶段文案。其余任务行只显示静态状态徽章。
 *
 * 进入审核：done / reviewing 态的任务可点「进入审核」，回调父页面切换到 KBReviewPanel。
 */
import { C, Spinner, StatusPill } from './kbConstants'
import { KB_JOB_STATUS_CONFIG, type KBCompressJob } from '@/api/kb'

/** 当前活跃任务的实时进度（来自父页面 SSE 订阅，仅一条活跃任务） */
export interface KBActiveProgress {
  jobId: string
  /** 阶段：extract=抽取知识点中 / compress=逐单元压缩仲裁中 / done=已完成 / error=出错 */
  phase: 'extract' | 'compress' | 'done' | 'error'
  doneItems: number
  totalItems: number
  message: string
}

interface KBJobListProps {
  jobs: KBCompressJob[]
  loading: boolean
  /** 当前正在压缩的任务进度（无则为 null） */
  activeProgress: KBActiveProgress | null
  onRefresh: () => void
  onEnterReview: (job: KBCompressJob) => void
}

export function KBJobList({ jobs, loading, activeProgress, onRefresh, onEnterReview }: KBJobListProps) {
  return (
    <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
      {/* 头部 */}
      <div style={{ padding: '14px 18px', borderBottom: `1px solid ${C.border}`, background: C.bg, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ fontSize: '15px', fontWeight: 700, color: C.text }}>
          📋 压缩任务列表
          <span style={{ fontSize: '12px', fontWeight: 400, color: C.textMuted, marginLeft: '8px' }}>共 {jobs.length} 个</span>
        </div>
        <button
          onClick={onRefresh}
          disabled={loading}
          style={{ padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: C.textSec, cursor: loading ? 'not-allowed' : 'pointer' }}
        >
          {loading ? '刷新中...' : '🔄 刷新'}
        </button>
      </div>

      {/* 列表 */}
      {loading && jobs.length === 0 ? (
        <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>加载中...</div>
      ) : jobs.length === 0 ? (
        <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>
          暂无任务。在上方新建一个课标压缩任务开始。
        </div>
      ) : (
        jobs.map((job, idx) => {
          const isActive = activeProgress && activeProgress.jobId === job.id
          const canReview = job.status === 'done' || job.status === 'reviewing'
          const pct = job.total_items > 0 ? Math.round((job.done_items / job.total_items) * 100) : 0
          return (
            <div key={job.id} style={{ borderBottom: idx < jobs.length - 1 ? `1px solid ${C.border}` : 'none' }}>
              {/* 任务主行 */}
              <div style={{ padding: '14px 18px', display: 'flex', alignItems: 'center', gap: '14px' }}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '4px' }}>
                    <StatusPill status={job.status} config={KB_JOB_STATUS_CONFIG} />
                    <span style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>
                      {job.subject || '—'}{job.grade_num > 0 ? ` · ${job.grade_num}年级` : ''}
                    </span>
                    <span style={{ fontSize: '12px', color: C.primary, background: C.primaryLight, padding: '1px 8px', borderRadius: '5px' }}>
                      批次 {job.batch_tag || '—'}
                    </span>
                  </div>
                  <div style={{ fontSize: '12px', color: C.textMuted }}>
                    已完成 {job.done_items}/{job.total_items} 个知识点
                    {' · '}创建于 {String(job.created_at).replace('T', ' ').substring(0, 16)}
                  </div>
                </div>

                {/* 静态进度（非活跃任务，已有 total 时显示静态条） */}
                {!isActive && job.total_items > 0 && (
                  <div style={{ width: '120px', flexShrink: 0 }}>
                    <div style={{ height: '6px', borderRadius: '3px', background: C.border, overflow: 'hidden' }}>
                      <div style={{ width: `${pct}%`, height: '100%', background: C.success }} />
                    </div>
                    <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '3px', textAlign: 'right' }}>{pct}%</div>
                  </div>
                )}

                {/* 进入审核按钮 */}
                <button
                  onClick={() => onEnterReview(job)}
                  disabled={!canReview}
                  style={{
                    padding: '7px 16px', borderRadius: '8px', border: 'none', flexShrink: 0,
                    background: canReview ? `linear-gradient(135deg,${C.cyan},${C.primary})` : C.bg,
                    color: canReview ? '#fff' : C.textMuted,
                    fontSize: '13px', fontWeight: 600, cursor: canReview ? 'pointer' : 'not-allowed',
                  }}
                  title={canReview ? '进入审核界面' : '任务完成后可进入审核'}
                >
                  进入审核 →
                </button>
              </div>

              {/* 活跃任务的实时进度（SSE 驱动） */}
              {isActive && (
                <div style={{ padding: '0 18px 16px' }}>
                  <div style={{
                    background: activeProgress!.phase === 'error' ? C.dangerLight : C.primaryLight,
                    borderRadius: '10px', padding: '12px 14px',
                  }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                      {activeProgress!.phase !== 'done' && activeProgress!.phase !== 'error' && <Spinner size={15} />}
                      <span style={{
                        fontSize: '13px', fontWeight: 600,
                        color: activeProgress!.phase === 'error' ? C.danger : C.primary,
                      }}>
                        {activeProgress!.phase === 'extract' && '🔍 正在通读原文、识别知识点...'}
                        {activeProgress!.phase === 'compress' && '⚙️ 正在逐个知识点多轮压缩 + 语义仲裁...'}
                        {activeProgress!.phase === 'done' && '✅ 压缩完成，可进入审核'}
                        {activeProgress!.phase === 'error' && '⚠️ 压缩出错'}
                      </span>
                    </div>
                    {/* 压缩阶段显示逐 item 进度条 */}
                    {activeProgress!.phase === 'compress' && activeProgress!.totalItems > 0 && (
                      <div style={{ marginBottom: '6px' }}>
                        <div style={{ height: '8px', borderRadius: '4px', background: C.white, overflow: 'hidden' }}>
                          <div style={{
                            width: `${Math.round((activeProgress!.doneItems / activeProgress!.totalItems) * 100)}%`,
                            height: '100%', background: `linear-gradient(90deg,${C.primary},${C.purple})`,
                            transition: 'width 300ms ease',
                          }} />
                        </div>
                        <div style={{ fontSize: '11px', color: C.textSec, marginTop: '4px' }}>
                          {activeProgress!.doneItems} / {activeProgress!.totalItems} 个知识点
                        </div>
                      </div>
                    )}
                    {activeProgress!.message && (
                      <div style={{ fontSize: '12px', color: C.textSec, lineHeight: 1.5 }}>
                        {activeProgress!.message}
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )
        })
      )}
    </div>
  )
}

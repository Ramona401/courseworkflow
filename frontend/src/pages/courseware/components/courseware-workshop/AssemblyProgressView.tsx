/**
 * AssemblyProgressView.tsx — 全自动装配「页面网格进度视图」（纯展示组件）
 *
 * 用途：全自动装配运行时，逐页展示三条流水线（HTML生成 / 配图 / 视频占位）的实时状态。
 * 为何用网格而非线性进度条：后端 HTML 流水线与配图流水线并行、重叠推进，且逐页 best-effort
 *   （某页失败不拖累其他页）。线性进度条无法表达"哪些页HTML好了、哪些正在配图、哪些失败"
 *   这种二维重叠状态；每页一张卡的网格能让老师一眼看清全局与逐页成败。
 *
 * 本组件是纯展示：不持有任何业务状态、不订阅 SSE、不发请求。
 *   所有数据由父组件 AutoAssemblyPanel 通过 props 喂入（pages 三态数组 + summary 汇总）。
 *   视觉全部复用项目既有 C 配色常量与"{color,bg}徽章"范式，不另起一套设计。
 */
import { C } from './workshopConstants'

// ==================== 类型 ====================

/** 单页某条流水线的阶段状态 */
export type AssemblyStageState =
  | 'pending'   // 待处理（灰）
  | 'running'   // 进行中（蓝，带动效点）
  | 'ok'        // 成功（绿✓）
  | 'skipped'   // 跳过/无需（灰，非失败）
  | 'failed'    // 失败（红）

/** 单页装配三态快照（父组件按 assembly_* 事件维护一份数组） */
export interface AssemblyPageState {
  page_number: number
  title: string
  html: AssemblyStageState    // HTML 生成链状态
  image: AssemblyStageState   // 配图链状态
  video: AssemblyStageState   // 视频首帧占位链状态
  /** 当前进行中阶段的文案（running 时显示，如"正在把配图融入版面…"） */
  note?: string
}

/** 装配汇总（assembly_start 建立、assembly_done 定格） */
export interface AssemblySummary {
  total_pages: number
  skip_video: boolean       // 交付模式：true=HTML+配图不做视频（中间档）
  running: boolean          // 是否装配进行中
  done: boolean             // 是否已收到 assembly_done
  message: string           // 顶部主文案（开场/进行中/完成）
  // 完成后的计数（done=true 时展示）
  html_success?: number
  html_fail?: number
  image_success?: number
  image_fail?: number
  image_skip?: number
  video_success?: number
  video_skip?: number
  elapsed_ms?: number
  errors?: string[]
}

interface Props {
  summary: AssemblySummary
  pages: AssemblyPageState[]   // 按 page_number 升序
}

// ==================== 阶段状态 → 徽章视觉 ====================

/** 单条流水线阶段的徽章配置（复用项目 {color,bg} 范式） */
const STAGE_BADGE: Record<AssemblyStageState, { label: string; color: string; bg: string; dot?: boolean }> = {
  pending: { label: '待处理', color: '#6B7280', bg: '#F3F4F6' },
  running: { label: '进行中', color: '#2563EB', bg: '#DBEAFE', dot: true },
  ok:      { label: '完成',   color: '#059669', bg: '#D1FAE5' },
  skipped: { label: '无需',   color: '#9CA3AF', bg: '#F3F4F6' },
  failed:  { label: '失败',   color: '#DC2626', bg: '#FEE2E2' },
}

/** 单条流水线的小徽章（HTML/配图/视频各一枚） */
function StageBadge({ icon, label, state }: { icon: string; label: string; state: AssemblyStageState }) {
  const b = STAGE_BADGE[state]
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 5, padding: '3px 8px', borderRadius: 6, background: b.bg }}>
      <span style={{ fontSize: 12 }}>{icon}</span>
      <span style={{ fontSize: 11, color: C.textSecondary }}>{label}</span>
      <span style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 11, fontWeight: 600, color: b.color }}>
        {b.dot && (
          <span style={{ display: 'inline-block', width: 6, height: 6, borderRadius: '50%', background: b.color, animation: 'cwAsmPulse 1s ease-in-out infinite' }} />
        )}
        {state === 'ok' ? '✓' : b.label}
      </span>
    </div>
  )
}

// ==================== 单页卡片 ====================

function PageCard({ p, skipVideo }: { p: AssemblyPageState; skipVideo: boolean }) {
  // 卡片左边框颜色：随该页整体推进状态变化（失败红 / 全好绿 / 进行中蓝 / 待处理灰）
  const anyFailed = p.html === 'failed' || p.image === 'failed'
  const htmlDone = p.html === 'ok'
  const imgSettled = p.image === 'ok' || p.image === 'skipped'
  const vidSettled = p.video === 'ok' || p.video === 'skipped'
  const allSettled = htmlDone && imgSettled && vidSettled
  const anyRunning = p.html === 'running' || p.image === 'running' || p.video === 'running'

  const edge = anyFailed ? '#EF4444' : allSettled ? '#10B981' : anyRunning ? '#3B82F6' : '#E5E7EB'

  return (
    <div style={{
      border: `1px solid ${C.border}`, borderLeft: `4px solid ${edge}`,
      borderRadius: 10, padding: '12px 14px', background: '#fff',
      display: 'flex', flexDirection: 'column', gap: 8, transition: 'border-color 300ms',
    }}>
      {/* 页头：页码 + 标题 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{
          flexShrink: 0, width: 26, height: 26, borderRadius: 7, display: 'flex', alignItems: 'center',
          justifyContent: 'center', fontSize: 12, fontWeight: 700, color: C.primary, background: C.primaryBg,
        }}>{p.page_number}</span>
        <span style={{
          fontSize: 13, fontWeight: 600, color: C.textPrimary, overflow: 'hidden',
          textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        }} title={p.title}>{p.title || `第 ${p.page_number} 页`}</span>
      </div>

      {/* 三条流水线徽章 */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
        <StageBadge icon="📝" label="HTML" state={p.html} />
        <StageBadge icon="🖼" label="配图" state={p.image} />
        {/* 中间档不做视频：视频徽章直接固定显"不做"，不占认知 */}
        {!skipVideo && <StageBadge icon="🎬" label="视频" state={p.video} />}
      </div>

      {/* 进行中文案（仅 running 有 note 时显示） */}
      {p.note && (
        <div style={{ fontSize: 11.5, color: C.textMuted, lineHeight: 1.5 }}>{p.note}</div>
      )}
    </div>
  )
}

// ==================== 主组件 ====================

export default function AssemblyProgressView({ summary, pages }: Props) {
  // 整体进度：以"三态都 settled（非 pending/running）的页数"为已完成口径
  const settledCount = pages.filter(p => {
    const htmlSettled = p.html === 'ok' || p.html === 'failed' || p.html === 'skipped'
    const imgSettled = p.image === 'ok' || p.image === 'skipped' || p.image === 'failed'
    const vidSettled = summary.skip_video || p.video === 'ok' || p.video === 'skipped' || p.video === 'failed'
    return htmlSettled && imgSettled && vidSettled
  }).length
  const total = summary.total_pages || pages.length
  const percent = total > 0 ? Math.round((settledCount / total) * 100) : 0

  // 完成后的耗时（秒，一位小数）
  const elapsedSec = summary.elapsed_ms != null ? (summary.elapsed_ms / 1000).toFixed(1) : null

  return (
    <div>
      {/* pulse 动效关键帧（进行中小圆点用；scoped 唯一名避免污染全局） */}
      <style>{`@keyframes cwAsmPulse{0%,100%{opacity:1}50%{opacity:0.3}}`}</style>

      {/* 顶部：模式标识 + 主文案 + 进度条 */}
      <div style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8, flexWrap: 'wrap' }}>
          <span style={{
            padding: '3px 10px', borderRadius: 12, fontSize: 12, fontWeight: 600,
            color: summary.skip_video ? '#0891B2' : '#7C3AED',
            background: summary.skip_video ? '#CFFAFE' : '#EDE9FE',
          }}>
            {summary.skip_video ? '🖼 HTML+配图（不做视频）' : '⚡ 全自动装配'}
          </span>
          {summary.running && (
            <span style={{ fontSize: 12, color: C.textMuted }}>正在装配，请勿关闭页面…</span>
          )}
          {summary.done && (
            <span style={{ fontSize: 12, fontWeight: 600, color: C.success }}>
              ✅ 装配完成{elapsedSec ? ` · 耗时 ${elapsedSec}s` : ''}
            </span>
          )}
        </div>

        {/* 主文案 */}
        {summary.message && (
          <div style={{ fontSize: 13, color: C.textSecondary, lineHeight: 1.6, marginBottom: 10 }}>{summary.message}</div>
        )}

        {/* 进度条 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, color: C.textSecondary, marginBottom: 6 }}>
          <span>装配进度（HTML与配图并行推进）</span>
          <span>共 {total} 页 · 已完成 <b style={{ color: C.success }}>{settledCount}</b> 页</span>
        </div>
        <div style={{ height: 8, borderRadius: 4, background: '#F3F4F6', overflow: 'hidden' }}>
          <div style={{
            height: '100%', borderRadius: 4, transition: 'width 500ms',
            width: `${percent}%`, background: 'linear-gradient(90deg, #7C3AED, #2563EB)',
          }} />
        </div>
      </div>

      {/* 页面网格 */}
      {pages.length > 0 ? (
        <div style={{
          display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 12,
        }}>
          {pages.map(p => <PageCard key={p.page_number} p={p} skipVideo={summary.skip_video} />)}
        </div>
      ) : (
        <div style={{ textAlign: 'center', padding: '32px 0', color: C.textMuted, fontSize: 13 }}>
          正在准备装配…
        </div>
      )}

      {/* 完成后：汇总统计 + 错误清单 */}
      {summary.done && (
        <div style={{ marginTop: 16, padding: '14px 16px', borderRadius: 10, background: '#F9FAFB', border: `1px solid ${C.border}` }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>装配汇总</div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px 18px', fontSize: 12.5, color: C.textSecondary, lineHeight: 1.8 }}>
            <span>📝 HTML 成功 <b style={{ color: C.success }}>{summary.html_success ?? 0}</b> 页{(summary.html_fail ?? 0) > 0 ? `，失败 ${summary.html_fail} 页` : ''}</span>
            <span>🖼 配图成功 <b style={{ color: C.success }}>{summary.image_success ?? 0}</b> 页{(summary.image_skip ?? 0) > 0 ? `，无需 ${summary.image_skip} 页` : ''}{(summary.image_fail ?? 0) > 0 ? `，失败 ${summary.image_fail} 页` : ''}</span>
            {!summary.skip_video && (
              <span>🎬 视频占位 <b style={{ color: C.success }}>{summary.video_success ?? 0}</b> 页{(summary.video_skip ?? 0) > 0 ? `，无需 ${summary.video_skip} 页` : ''}</span>
            )}
          </div>
          {/* 错误清单（有失败时展开，帮助老师定位哪几页要手动补） */}
          {summary.errors && summary.errors.length > 0 && (
            <details style={{ marginTop: 10 }}>
              <summary style={{ fontSize: 12, color: C.danger, cursor: 'pointer' }}>查看 {summary.errors.length} 条失败详情</summary>
              <div style={{ marginTop: 6, maxHeight: 160, overflow: 'auto', display: 'flex', flexDirection: 'column', gap: 4 }}>
                {summary.errors.map((e, i) => (
                  <div key={i} style={{ fontSize: 11.5, color: '#991B1B', background: '#FEF2F2', borderRadius: 6, padding: '5px 8px', lineHeight: 1.5 }}>{e}</div>
                ))}
              </div>
            </details>
          )}
        </div>
      )}
    </div>
  )
}

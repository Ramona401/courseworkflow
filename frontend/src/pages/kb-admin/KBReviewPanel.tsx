/**
 * KBReviewPanel.tsx — 课标压缩审核主面板
 *
 * 职责（PRD 6.2 审核界面）：
 *   - 拉取审核队列（getKBReviewQueue），渲染每个待审单元（KBReviewItemView）。
 *   - 多轮草稿并排展示：每轮是「解码后的中文卡片」（KBRoundView.decoded.fields），不是索引原文。
 *   - 高亮仲裁识别的冲突点（conflicts）。
 *   - 三选一操作：确认 / 选版 / 退回（不手写编辑——被采纳的永远是 AI 已生成的合法索引版本）。
 *   - 入库 + 蓝绿切换交给 KBCommitBar。
 *
 * 专利保护边界（本面板严格遵守）：
 *   只渲染后端解码后的人话（decoded.fields 的中文标签/内容 + kp_code 标识）。
 *   索引原文后端返回结构里就不含，前端也绝不自行拼接或显示。
 *
 * 字段名对齐后端（models/kb_compress.go）：
 *   - KBDecodedField 为 {label, tag, content}（之前误用 key/value）。
 *   - KBRoundView.decoded 为指针，出错轮次可能为 null，渲染前判空。
 *
 * 三选一语义：
 *   - confirm：采纳仲裁选中轮（chosen_round），意义正确且一致时用。
 *   - select ：审核员手动选中某一轮再确认（多轮有出入、仲裁选错时用）。
 *   - reject ：几版都不对，退回重压（该 item 标记 rejected）。
 */
import { useState, useEffect, useCallback } from 'react'
import { C, Spinner, StatusPill } from './components/kbConstants'
import { KBCommitBar } from './components/KBCommitBar'
import {
  getKBReviewQueue, reviewKBItem,
  KB_REVIEW_STATUS_CONFIG,
  type KBReviewItemView, type KBRoundView,
} from '@/api/kb'

interface KBReviewPanelProps {
  jobId: string
  batchTag: string
  onError: (msg: string) => void
  onSuccess: (msg: string) => void
}

export function KBReviewPanel({ jobId, batchTag, onError, onSuccess }: KBReviewPanelProps) {
  const [items, setItems] = useState<KBReviewItemView[]>([])
  const [loading, setLoading] = useState(false)
  // 每个 item 当前在界面上手动选中的轮次（用于 select 动作；默认取仲裁 chosen_round）
  const [selectedRound, setSelectedRound] = useState<Record<string, number>>({})
  // 每个 item 的退回意见输入
  const [rejectNote, setRejectNote] = useState<Record<string, string>>({})
  // 正在提交动作的 item（禁用按钮防重复）
  const [acting, setActing] = useState<string | null>(null)

  // ---- 加载审核队列（后端返回 {items, total}） ----
  const loadQueue = useCallback(async () => {
    try {
      setLoading(true)
      const data = await getKBReviewQueue(jobId)
      const list = data.items || []
      setItems(list)
      // 初始化每个 item 的选中轮为仲裁选中轮
      const initSel: Record<string, number> = {}
      for (const it of list) initSel[it.item_id] = it.chosen_round
      setSelectedRound(initSel)
    } catch (e: unknown) {
      onError(e instanceof Error ? e.message : '加载审核队列失败')
    } finally {
      setLoading(false)
    }
  }, [jobId, onError])

  useEffect(() => { loadQueue() }, [loadQueue])

  // ---- 三选一动作 ----
  const doAction = async (
    item: KBReviewItemView,
    action: 'confirm' | 'select' | 'reject',
  ) => {
    try {
      setActing(item.item_id)
      if (action === 'reject') {
        await reviewKBItem(item.item_id, { action: 'reject', review_note: rejectNote[item.item_id] || '' })
        onSuccess('已退回该知识点')
      } else if (action === 'select') {
        await reviewKBItem(item.item_id, { action: 'select', chosen_round: selectedRound[item.item_id] })
        onSuccess('已采纳所选版本')
      } else {
        await reviewKBItem(item.item_id, { action: 'confirm' })
        onSuccess('已确认该知识点')
      }
      await loadQueue() // 刷新队列（已处理项会改变状态）
    } catch (e: unknown) {
      onError(e instanceof Error ? e.message : '操作失败')
    } finally {
      setActing(null)
    }
  }

  // 统计
  const needReviewCount = items.filter(i => i.review_status === 'need_review').length
  const autoPassedCount = items.filter(i => i.review_status === 'auto_passed').length
  const handledCount = items.filter(i => i.review_status === 'approved' || i.review_status === 'rejected').length

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {/* 统计条 */}
      <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, padding: '16px 20px', display: 'flex', alignItems: 'center', gap: '28px' }}>
        <div style={{ fontSize: '15px', fontWeight: 700, color: C.text }}>🔎 审核队列</div>
        <div><span style={{ fontSize: '20px', fontWeight: 700, color: C.warning }}>{needReviewCount}</span><span style={{ fontSize: '13px', color: C.textSec, marginLeft: '6px' }}>待人工</span></div>
        <div><span style={{ fontSize: '20px', fontWeight: 700, color: C.success }}>{autoPassedCount}</span><span style={{ fontSize: '13px', color: C.textSec, marginLeft: '6px' }}>自动通过</span></div>
        <div><span style={{ fontSize: '20px', fontWeight: 700, color: C.primary }}>{handledCount}</span><span style={{ fontSize: '13px', color: C.textSec, marginLeft: '6px' }}>已处理</span></div>
        <button onClick={loadQueue} disabled={loading}
          style={{ marginLeft: 'auto', padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '13px', color: C.textSec, cursor: loading ? 'not-allowed' : 'pointer' }}>
          {loading ? '刷新中...' : '🔄 刷新'}
        </button>
      </div>

      {/* 队列内容 */}
      {loading && items.length === 0 ? (
        <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div>
      ) : items.length === 0 ? (
        <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, padding: '40px', textAlign: 'center', color: C.textMuted, fontSize: '13px' }}>
          暂无需审核的知识点（可能全部自动通过，或尚未压缩完成）。可直接到下方入库。
        </div>
      ) : (
        items.map(item => (
          <KBReviewItemCard
            key={item.item_id}
            item={item}
            selectedRound={selectedRound[item.item_id] ?? item.chosen_round}
            rejectNote={rejectNote[item.item_id] || ''}
            acting={acting === item.item_id}
            onSelectRound={(r) => setSelectedRound(prev => ({ ...prev, [item.item_id]: r }))}
            onRejectNoteChange={(v) => setRejectNote(prev => ({ ...prev, [item.item_id]: v }))}
            onAction={(a) => doAction(item, a)}
          />
        ))
      )}

      {/* 入库 + 蓝绿切换 */}
      <KBCommitBar
        jobId={jobId}
        batchTag={batchTag}
        pendingReviewCount={needReviewCount}
        onError={onError}
        onSuccess={onSuccess}
      />
    </div>
  )
}

// ==================== 单个待审单元卡片 ====================

function KBReviewItemCard({ item, selectedRound, rejectNote, acting, onSelectRound, onRejectNoteChange, onAction }: {
  item: KBReviewItemView
  selectedRound: number
  rejectNote: string
  acting: boolean
  onSelectRound: (round: number) => void
  onRejectNoteChange: (v: string) => void
  onAction: (action: 'confirm' | 'select' | 'reject') => void
}) {
  const [showReject, setShowReject] = useState(false)
  // 已处理（approved/rejected）的项只读展示，不再提供操作
  const handled = item.review_status === 'approved' || item.review_status === 'rejected'
  const isLowConf = item.confidence === 'low'

  return (
    <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${isLowConf ? C.warning + '55' : C.border}`, overflow: 'hidden' }}>
      {/* 卡片头 */}
      <div style={{ padding: '14px 18px', borderBottom: `1px solid ${C.border}`, background: C.bg, display: 'flex', alignItems: 'center', gap: '12px' }}>
        <span style={{ fontSize: '13px', fontWeight: 700, color: C.textSec }}>#{item.seq}</span>
        <StatusPill status={item.review_status} config={KB_REVIEW_STATUS_CONFIG} />
        <span style={{ fontSize: '12px', color: isLowConf ? C.warning : C.success, fontWeight: 600 }}>
          {isLowConf ? '低置信（存在分歧）' : '高置信（多轮一致）'}
        </span>
        {item.page_label && <span style={{ fontSize: '12px', color: C.textMuted }}>页 {item.page_label}</span>}
      </div>

      <div style={{ padding: '16px 18px' }}>
        {/* 原文片段（供对照判断，非索引） */}
        {item.source_excerpt && (
          <div style={{ marginBottom: '14px' }}>
            <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '5px' }}>📄 原文片段（供对照）</div>
            <div style={{ fontSize: '13px', color: C.text, background: C.bg, borderRadius: '8px', padding: '10px 12px', lineHeight: 1.6, whiteSpace: 'pre-wrap' }}>
              {item.source_excerpt}
            </div>
          </div>
        )}

        {/* 仲裁冲突高亮 */}
        {item.conflicts && item.conflicts.length > 0 && (
          <div style={{ marginBottom: '14px', background: C.dangerLight, borderRadius: '8px', padding: '10px 14px' }}>
            <div style={{ fontSize: '13px', fontWeight: 600, color: C.danger, marginBottom: '6px' }}>⚠️ 仲裁发现以下分歧，请重点核对：</div>
            <ul style={{ margin: 0, paddingLeft: '18px', fontSize: '12px', color: C.text, lineHeight: 1.7 }}>
              {item.conflicts.map((c, i) => <li key={i}>{c}</li>)}
            </ul>
          </div>
        )}

        {/* 多轮草稿并排（解码后的人话卡片） */}
        <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '8px' }}>
          各轮压缩结果（{item.rounds.length} 轮，已解码为中文）— {handled ? '只读' : '选中一版作为采纳'}
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: `repeat(${Math.min(item.rounds.length, 3)}, 1fr)`, gap: '12px', marginBottom: handled ? 0 : '16px' }}>
          {item.rounds.map(round => (
            <RoundCard
              key={round.round}
              round={round}
              chosen={round.round === item.chosen_round}
              selected={!handled && round.round === selectedRound}
              clickable={!handled && !acting}
              onClick={() => onSelectRound(round.round)}
            />
          ))}
        </div>

        {/* 三选一操作（仅未处理项显示） */}
        {!handled && (
          <div>
            {!showReject ? (
              <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap', alignItems: 'center' }}>
                <button
                  onClick={() => onAction('confirm')}
                  disabled={acting}
                  style={actionBtnStyle(C.success, acting)}
                  title="采纳仲裁选中的版本"
                >
                  {acting && <Spinner size={14} />}✓ 确认（采纳仲裁选中第 {item.chosen_round} 轮）
                </button>
                <button
                  onClick={() => onAction('select')}
                  disabled={acting}
                  style={actionBtnStyle(C.primary, acting)}
                  title="采纳我手动选中的版本"
                >
                  ◉ 采纳所选第 {selectedRound} 轮
                </button>
                <button
                  onClick={() => setShowReject(true)}
                  disabled={acting}
                  style={actionBtnStyle(C.danger, acting)}
                  title="几版都不对，退回重压"
                >
                  ✕ 退回重压
                </button>
              </div>
            ) : (
              <div style={{ background: C.dangerLight, borderRadius: '9px', padding: '12px 14px' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: C.danger, marginBottom: '8px' }}>退回重压（可填写原因，供后续修正参考）</div>
                <textarea
                  value={rejectNote}
                  onChange={e => onRejectNoteChange(e.target.value)}
                  placeholder="如：难度档判断有误 / 学业要求与原文不符 ..."
                  style={{ width: '100%', minHeight: '60px', padding: '8px 10px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', resize: 'vertical', boxSizing: 'border-box', fontFamily: 'inherit' }}
                />
                <div style={{ display: 'flex', gap: '10px', marginTop: '10px' }}>
                  <button onClick={() => onAction('reject')} disabled={acting} style={actionBtnStyle(C.danger, acting)}>
                    {acting && <Spinner size={14} />}确认退回
                  </button>
                  <button onClick={() => setShowReject(false)} disabled={acting}
                    style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, color: C.textSec, fontSize: '13px', cursor: acting ? 'not-allowed' : 'pointer' }}>
                    取消
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

// ==================== 单轮解码卡片 ====================

function RoundCard({ round, chosen, selected, clickable, onClick }: {
  round: KBRoundView
  chosen: boolean
  selected: boolean
  clickable: boolean
  onClick: () => void
}) {
  const decoded = round.decoded
  // 出错轮次 decoded 为 null，降级展示错误信息而非崩溃
  const hasError = !!round.error || !decoded

  return (
    <div
      onClick={clickable ? onClick : undefined}
      style={{
        border: `2px solid ${selected ? C.primary : C.border}`,
        borderRadius: '10px', padding: '12px',
        background: selected ? C.primaryLight : C.white,
        cursor: clickable ? 'pointer' : 'default',
        transition: 'all 150ms ease',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '8px' }}>
        <span style={{ fontSize: '12px', fontWeight: 700, color: C.text }}>第 {round.round} 轮</span>
        {chosen && <span style={{ fontSize: '10px', color: C.success, background: C.successLight, padding: '1px 6px', borderRadius: '4px', fontWeight: 600 }}>仲裁选中</span>}
        {selected && <span style={{ fontSize: '10px', color: C.primary, background: C.white, padding: '1px 6px', borderRadius: '4px', fontWeight: 600, border: `1px solid ${C.primary}` }}>已选</span>}
        {hasError && <span style={{ fontSize: '10px', color: C.danger, background: C.dangerLight, padding: '1px 6px', borderRadius: '4px', fontWeight: 600 }}>本轮异常</span>}
        {decoded?.decode_failed && <span style={{ fontSize: '10px', color: C.warning, background: C.warningLight, padding: '1px 6px', borderRadius: '4px', fontWeight: 600 }}>解码降级</span>}
      </div>

      {/* 出错轮次：只显示错误信息 */}
      {hasError ? (
        <div style={{ fontSize: '12px', color: C.danger, lineHeight: 1.5 }}>
          {round.error || '该轮无解码结果（压缩失败或返回为空）'}
        </div>
      ) : (
        <>
          {decoded!.kp_code && (
            <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '8px', fontFamily: 'monospace' }}>{decoded!.kp_code}</div>
          )}
          {/* 学科/学段/年级/深度档（解码后的定位信息） */}
          {(decoded!.subject_name || decoded!.grade_name || decoded!.depth_name) && (
            <div style={{ fontSize: '11px', color: C.textSec, marginBottom: '8px', display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
              {decoded!.subject_name && <span style={{ background: C.bg, padding: '1px 6px', borderRadius: '4px' }}>{decoded!.subject_name}</span>}
              {decoded!.stage_name && <span style={{ background: C.bg, padding: '1px 6px', borderRadius: '4px' }}>{decoded!.stage_name}</span>}
              {decoded!.grade_name && <span style={{ background: C.bg, padding: '1px 6px', borderRadius: '4px' }}>{decoded!.grade_name}</span>}
              {decoded!.depth_name && <span style={{ background: C.bg, padding: '1px 6px', borderRadius: '4px' }}>{decoded!.depth_name}</span>}
            </div>
          )}
          {/* 语义字段（人话标签 + 内容） */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            {decoded!.fields.map((f, i) => (
              <div key={i}>
                <div style={{ fontSize: '11px', color: C.textMuted, fontWeight: 600 }}>{f.label}</div>
                <div style={{ fontSize: '12px', color: C.text, lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>{f.content || '—'}</div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

function actionBtnStyle(color: string, disabled: boolean): React.CSSProperties {
  return {
    padding: '8px 16px', borderRadius: '8px', border: 'none',
    background: disabled ? C.textMuted : color,
    color: '#fff', fontSize: '13px', fontWeight: 600,
    cursor: disabled ? 'not-allowed' : 'pointer',
    display: 'flex', alignItems: 'center', gap: '6px',
  }
}

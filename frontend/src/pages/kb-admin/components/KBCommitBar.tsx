/**
 * KBCommitBar.tsx — 入库提交栏 + commit 结果回显 + 蓝绿切换弹窗
 *
 * 从 KBReviewPanel 拆出，专司「入库 + 切换」两个收口动作，使审核主面板保持 <600 行。
 *
 * PRD 关键产品细节（本组件重点落实）：
 *   commit 的部分失败必须明确回显。后端 CommitBatch 返回 {committed, skipped, errors}，
 *   单条解码缺字段会被 skip 并记进 errors。本组件不只显示"成功 N 条"，而是把 skipped 的
 *   条数与每条原因逐条列出，否则被跳过的知识点会悄悄消失，审核员无从察觉。
 *
 * 蓝绿切换：commit 后的候选数据（status='staged'）需经整批切换才真正 active（旧批转 archived）。
 *   切换是不可逆的上线动作，用确认弹窗二次确认，文案明确告知"旧批将被归档、消费端将读到本批"。
 */
import { useState } from 'react'
import { C, Spinner } from './kbConstants'
import { commitKBBatch, switchKBBatch, type KBCommitBatchResult } from '@/api/kb'

interface KBCommitBarProps {
  jobId: string
  batchTag: string
  /** 队列里仍处于 need_review（待人工）的条数，>0 时提示审核员尚有未处理项 */
  pendingReviewCount: number
  onError: (msg: string) => void
  onSuccess: (msg: string) => void
}

export function KBCommitBar({ jobId, batchTag, pendingReviewCount, onError, onSuccess }: KBCommitBarProps) {
  const [committing, setCommitting] = useState(false)
  const [commitResult, setCommitResult] = useState<KBCommitBatchResult | null>(null)

  const [showSwitchConfirm, setShowSwitchConfirm] = useState(false)
  const [switching, setSwitching] = useState(false)
  const [switched, setSwitched] = useState(false)

  // ---- commit 入库 ----
  const handleCommit = async () => {
    if (!batchTag) { onError('该任务缺少批次标识，无法入库'); return }
    try {
      setCommitting(true)
      const result = await commitKBBatch(jobId, batchTag)
      setCommitResult(result)
      if (result.skipped > 0) {
        // 有跳过项时不报喜，提示审核员去看明细
        onError(`入库完成：成功 ${result.committed} 条，跳过 ${result.skipped} 条（详见下方明细）`)
      } else {
        onSuccess(`入库成功：${result.committed} 条知识点已写入候选库`)
      }
    } catch (e: unknown) {
      onError(e instanceof Error ? e.message : '入库失败')
    } finally {
      setCommitting(false)
    }
  }

  // ---- 蓝绿切换 ----
  const handleSwitch = async () => {
    try {
      setSwitching(true)
      await switchKBBatch(batchTag)
      setSwitched(true)
      setShowSwitchConfirm(false)
      onSuccess(`已切换：批次「${batchTag}」现已上线，旧批已归档`)
    } catch (e: unknown) {
      onError(e instanceof Error ? e.message : '切换失败')
    } finally {
      setSwitching(false)
    }
  }

  return (
    <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, padding: '18px 20px' }}>
      <div style={{ fontSize: '15px', fontWeight: 700, color: C.text, marginBottom: '6px' }}>
        🗄️ 入库与上线
      </div>
      <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '14px', lineHeight: 1.6 }}>
        入库先把「已确认 + 自动通过」的知识点写入候选库（不影响线上）；确认无误后再整批蓝绿切换上线，
        旧批自动归档，消费端只读上线批，可回退。
      </div>

      {/* 待人工提示 */}
      {pendingReviewCount > 0 && (
        <div style={{ background: C.warningLight, borderRadius: '9px', padding: '10px 14px', marginBottom: '14px', fontSize: '13px', color: C.warning, fontWeight: 600 }}>
          ⚠️ 仍有 {pendingReviewCount} 个知识点待人工处理（确认/选版/退回）。这些项不会被入库——请先处理完再入库，以免遗漏。
        </div>
      )}

      {/* 操作按钮 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' }}>
        <button
          onClick={handleCommit}
          disabled={committing || switched}
          style={{
            padding: '10px 22px', borderRadius: '9px', border: 'none',
            background: (committing || switched) ? C.textMuted : `linear-gradient(135deg,${C.primary},${C.purple})`,
            color: '#fff', fontSize: '14px', fontWeight: 600,
            cursor: (committing || switched) ? 'not-allowed' : 'pointer',
            display: 'flex', alignItems: 'center', gap: '8px',
          }}
        >
          {committing && <Spinner size={15} />}
          {committing ? '入库中...' : '① 入库到候选库'}
        </button>

        <button
          onClick={() => setShowSwitchConfirm(true)}
          disabled={!commitResult || switched || (commitResult?.committed ?? 0) === 0}
          style={{
            padding: '10px 22px', borderRadius: '9px', border: 'none',
            background: (!commitResult || switched || (commitResult?.committed ?? 0) === 0) ? C.bg : `linear-gradient(135deg,${C.success},#059669)`,
            color: (!commitResult || switched || (commitResult?.committed ?? 0) === 0) ? C.textMuted : '#fff',
            fontSize: '14px', fontWeight: 600,
            cursor: (!commitResult || switched || (commitResult?.committed ?? 0) === 0) ? 'not-allowed' : 'pointer',
          }}
          title={!commitResult ? '请先入库' : switched ? '已切换' : '整批上线'}
        >
          {switched ? '✅ 已上线' : '② 蓝绿切换上线'}
        </button>

        <span style={{ fontSize: '12px', color: C.textMuted }}>
          批次：<b style={{ color: C.text }}>{batchTag || '—'}</b>
        </span>
      </div>

      {/* commit 结果回显（重点：skipped 必须明确显示，不能让知识点悄悄消失） */}
      {commitResult && (
        <div style={{ marginTop: '16px', borderTop: `1px solid ${C.border}`, paddingTop: '14px' }}>
          <div style={{ display: 'flex', gap: '20px', marginBottom: commitResult.skipped > 0 ? '12px' : 0 }}>
            <div>
              <span style={{ fontSize: '22px', fontWeight: 700, color: C.success }}>{commitResult.committed}</span>
              <span style={{ fontSize: '13px', color: C.textSec, marginLeft: '6px' }}>条已入库</span>
            </div>
            {commitResult.skipped > 0 && (
              <div>
                <span style={{ fontSize: '22px', fontWeight: 700, color: C.danger }}>{commitResult.skipped}</span>
                <span style={{ fontSize: '13px', color: C.textSec, marginLeft: '6px' }}>条被跳过</span>
              </div>
            )}
          </div>

          {/* 跳过明细：逐条列出原因，让审核员看清哪些知识点没进库 */}
          {commitResult.skipped > 0 && (
            <div style={{ background: C.dangerLight, borderRadius: '9px', padding: '12px 14px' }}>
              <div style={{ fontSize: '13px', fontWeight: 600, color: C.danger, marginBottom: '8px' }}>
                以下知识点被跳过，未写入库（请检查原因，必要时退回重压后重新入库）：
              </div>
              <ul style={{ margin: 0, paddingLeft: '18px', fontSize: '12px', color: C.text, lineHeight: 1.7 }}>
                {commitResult.errors.length > 0
                  ? commitResult.errors.map((err, i) => <li key={i}>{err}</li>)
                  : <li>共 {commitResult.skipped} 条被跳过（后端未返回具体原因）</li>}
              </ul>
            </div>
          )}
        </div>
      )}

      {/* 蓝绿切换确认弹窗 */}
      {showSwitchConfirm && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.45)', zIndex: 1500, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <div style={{ background: C.white, borderRadius: '16px', padding: '24px 26px', width: '440px', maxWidth: '90vw', boxShadow: '0 12px 40px rgba(0,0,0,0.25)' }}>
            <div style={{ fontSize: '17px', fontWeight: 700, color: C.text, marginBottom: '12px' }}>
              确认整批上线？
            </div>
            <div style={{ fontSize: '13px', color: C.textSec, lineHeight: 1.7, marginBottom: '20px' }}>
              此操作将把批次 <b style={{ color: C.primary }}>{batchTag}</b> 整批转为 <b>上线（active）</b>，
              同一学科年级范围的<b style={{ color: C.danger }}>旧批将被自动归档（archived）</b>。
              切换后课件工坊与教案撰写将读取本批知识点。此动作可通过重新切换旧批回退。
            </div>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
              <button
                onClick={() => setShowSwitchConfirm(false)}
                disabled={switching}
                style={{ padding: '9px 18px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, color: C.textSec, fontSize: '14px', cursor: switching ? 'not-allowed' : 'pointer' }}
              >
                取消
              </button>
              <button
                onClick={handleSwitch}
                disabled={switching}
                style={{
                  padding: '9px 20px', borderRadius: '8px', border: 'none',
                  background: switching ? C.textMuted : `linear-gradient(135deg,${C.success},#059669)`,
                  color: '#fff', fontSize: '14px', fontWeight: 600, cursor: switching ? 'not-allowed' : 'pointer',
                  display: 'flex', alignItems: 'center', gap: '8px',
                }}
              >
                {switching && <Spinner size={15} />}
                {switching ? '切换中...' : '确认上线'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

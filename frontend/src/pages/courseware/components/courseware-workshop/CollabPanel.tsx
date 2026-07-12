/**
 * 集体备课面板 — CollabPanel.tsx（阶段4新建）
 *
 * 线下集体备课的标记态 + 参与者管理 UI。挂在工坊页 collab tab。
 *
 * 设计（最小可用）：
 *   - 作者视角：发起/结束集体备课；进行中可加/移参与者（候选下拉=同校同组老师）。
 *   - 非作者视角：只读显示当前是否在集体备课、自己能否参与微调。
 *
 * 议课走页级批注（annotation tab），留痕走版本快照（RefinePanel 历史版本），本面板不涉及。
 *
 * 数据自包含：进面板自行拉 collab 状态 + 候选列表，增删后就地刷新并通知父组件（onChanged）。
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getCollabStatus, startCollab, endCollab,
  addCollabMember, removeCollabMember, listCollabCandidates,
  CW_COLLAB_IN_SESSION,
} from '@/api/coursewares'
import type { CollabStatusResponse, CollabCandidate } from '@/api/coursewares'

interface Props {
  coursewareId: string
  ownerId: string         // 课件作者ID（判断当前用户是不是作者）
  currentUserId: string   // 当前登录用户ID
  onChanged?: () => void  // 状态变化后通知父组件（刷新课件详情里的 collab_state 徽章）
}

export default function CollabPanel({ coursewareId, ownerId, currentUserId, onChanged }: Props) {
  const [status, setStatus] = useState<CollabStatusResponse | null>(null)
  const [candidates, setCandidates] = useState<CollabCandidate[]>([])
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [pickUserId, setPickUserId] = useState('')   // 下拉选中的待加入用户
  const [msg, setMsg] = useState('')

  const isOwner = currentUserId === ownerId
  const inSession = status?.collab_state === CW_COLLAB_IN_SESSION

  // 拉取集体备课状态
  const reload = useCallback(async () => {
    try {
      const s = await getCollabStatus(coursewareId)
      setStatus(s)
    } catch (e) {
      setMsg('❌ 加载集体备课状态失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally {
      setLoading(false)
    }
  }, [coursewareId])

  useEffect(() => { reload() }, [reload])

  // 作者才需要候选列表（用于加人下拉）
  useEffect(() => {
    if (!isOwner) return
    listCollabCandidates()
      .then(r => setCandidates(r.candidates || []))
      .catch(() => { /* 候选拉取失败不阻断面板，加人下拉为空即可 */ })
  }, [isOwner])

  // 已是参与者的 user_id 集合（候选下拉里过滤掉已加入的）
  const memberIdSet = new Set((status?.members || []).map(m => m.user_id))
  const selectable = candidates.filter(c => !memberIdSet.has(c.user_id))

  const doStart = async () => {
    if (busy) return
    setBusy(true); setMsg('')
    try {
      await startCollab(coursewareId)
      setMsg('✅ 已发起集体备课，现在可以添加参与的老师了')
      await reload()
      onChanged?.()
    } catch (e) {
      setMsg('❌ 发起失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setBusy(false) }
  }

  const doEnd = async () => {
    if (busy) return
    setBusy(true); setMsg('')
    try {
      await endCollab(coursewareId)
      setMsg('✅ 已结束集体备课')
      await reload()
      onChanged?.()
    } catch (e) {
      setMsg('❌ 结束失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setBusy(false) }
  }

  const doAdd = async () => {
    if (busy || !pickUserId) return
    setBusy(true); setMsg('')
    try {
      await addCollabMember(coursewareId, pickUserId)
      setPickUserId('')
      await reload()
      onChanged?.()
    } catch (e) {
      setMsg('❌ 添加失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setBusy(false) }
  }

  const doRemove = async (uid: string) => {
    if (busy) return
    setBusy(true); setMsg('')
    try {
      await removeCollabMember(coursewareId, uid)
      await reload()
      onChanged?.()
    } catch (e) {
      setMsg('❌ 移除失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setBusy(false) }
  }

  if (loading) {
    return <div style={{ padding: 16, color: '#888', fontSize: 13 }}>加载集体备课状态…</div>
  }

  return (
    <div style={{ padding: 16, fontSize: 13, color: '#333' }}>
      {/* 状态徽章 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
        <span style={{
          display: 'inline-flex', alignItems: 'center', gap: 4,
          padding: '3px 10px', borderRadius: 12, fontSize: 12, fontWeight: 600,
          background: inSession ? '#e6f7ed' : '#f0f0f0',
          color: inSession ? '#10893e' : '#888',
        }}>
          {inSession ? '🟢 集体备课中' : '⚪ 未集体备课'}
        </span>
        {inSession && (
          <span style={{ color: '#888', fontSize: 12 }}>
            共 {status?.total || 0} 位老师参与
          </span>
        )}
      </div>

      {/* 说明文案 */}
      <p style={{ color: '#888', fontSize: 12, lineHeight: 1.6, marginBottom: 14 }}>
        集体备课用于线下一群老师聚在一起、对着同一课件当场修改与讨论。发起后，被加入的老师可一起微调本课件；
        讨论意见请走「批注」，每次修改都会自动留版本可回退。
      </p>

      {/* 作者操作区 */}
      {isOwner ? (
        <>
          {!inSession ? (
            <button onClick={doStart} disabled={busy} style={btnPrimary(busy)}>
              发起集体备课
            </button>
          ) : (
            <>
              {/* 参与者列表 */}
              <div style={{ marginBottom: 12 }}>
                <div style={{ fontWeight: 600, marginBottom: 6, fontSize: 12, color: '#555' }}>参与的老师</div>
                {(!status?.members || status.members.length === 0) ? (
                  <div style={{ color: '#aaa', fontSize: 12 }}>还没有添加参与者，从下方选择老师加入</div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                    {status!.members!.map(m => (
                      <div key={m.id} style={memberRow}>
                        <span>{m.user_name}</span>
                        <button onClick={() => doRemove(m.user_id)} disabled={busy} style={btnRemove(busy)}>移除</button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* 添加参与者 */}
              <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
                <select
                  value={pickUserId}
                  onChange={e => setPickUserId(e.target.value)}
                  disabled={busy}
                  style={{ flex: 1, padding: '6px 8px', borderRadius: 6, border: '1px solid #ddd', fontSize: 13 }}
                >
                  <option value="">选择要加入的老师…</option>
                  {selectable.map(c => (
                    <option key={c.user_id} value={c.user_id}>
                      {c.display_name}（{c.username}）
                    </option>
                  ))}
                </select>
                <button onClick={doAdd} disabled={busy || !pickUserId} style={btnPrimary(busy || !pickUserId)}>添加</button>
              </div>

              <button onClick={doEnd} disabled={busy} style={btnDanger(busy)}>结束集体备课</button>
            </>
          )}
        </>
      ) : (
        /* 非作者视角：只读 */
        <div style={{ padding: 12, background: '#f9f9f9', borderRadius: 8, fontSize: 13 }}>
          {inSession ? (
            status?.can_edit
              ? '🟢 本课件正在集体备课，你已被加入，可在「预览/微调」中一起修改本课件。'
              : '🟢 本课件正在集体备课，但你尚未被加入参与者名单，暂不能微调。'
          ) : (
            '本课件当前未在集体备课。集体备课由课件作者发起。'
          )}
        </div>
      )}

      {msg && <div style={{ marginTop: 12, fontSize: 12, color: msg.startsWith('✅') ? '#10893e' : '#d4380d' }}>{msg}</div>}
    </div>
  )
}

// ---- 内联样式小工具 ----
function btnPrimary(disabled: boolean): React.CSSProperties {
  return {
    padding: '6px 14px', borderRadius: 6, border: 'none', fontSize: 13, fontWeight: 600,
    background: disabled ? '#bbb' : '#1677ff', color: '#fff',
    cursor: disabled ? 'not-allowed' : 'pointer',
  }
}
function btnDanger(disabled: boolean): React.CSSProperties {
  return {
    padding: '6px 14px', borderRadius: 6, border: '1px solid #ffccc7', fontSize: 13, fontWeight: 600,
    background: disabled ? '#f5f5f5' : '#fff1f0', color: disabled ? '#aaa' : '#d4380d',
    cursor: disabled ? 'not-allowed' : 'pointer',
  }
}
function btnRemove(disabled: boolean): React.CSSProperties {
  return {
    padding: '2px 10px', borderRadius: 4, border: '1px solid #eee', fontSize: 12,
    background: '#fff', color: disabled ? '#ccc' : '#d4380d',
    cursor: disabled ? 'not-allowed' : 'pointer',
  }
}
const memberRow: React.CSSProperties = {
  display: 'flex', justifyContent: 'space-between', alignItems: 'center',
  padding: '6px 10px', background: '#fafafa', borderRadius: 6, border: '1px solid #f0f0f0',
}

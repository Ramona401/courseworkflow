/**
 * KBAuthorizedPanel.tsx — 知识库压缩入库系统「访问白名单」管理面板
 *
 * 用途（KB 迭代一 · Phase 6）：
 *   admin 在 /admin 管理「谁能进知识库压缩系统」。白名单不绑角色——
 *   admin 恒通过 + 名单内用户通过 + 其余 403（后端 RequireKBAuthorized 中间件）。
 *   交互范式仿 OrgAdminsPanel：UserSearchPicker 选人 + 备注 + 列表 + 移除（二次确认）。
 *
 * 后端对接（api/kb.ts，路由 /api/v1/admin/kb-authorized，仅 admin）：
 *   - listKBAuthorized()                 列出白名单
 *   - addKBAuthorized({user_id, note?})   新增
 *   - removeKBAuthorized(userId)          移除
 *
 * 字段名对齐后端 KBAuthorizedUserItem：授权人字段为 granted_by（COALESCE 后的显示名，
 *   后端不单独返回 granted_by_name）。
 *
 * 放在 admin/components 下复用本目录的 C/fmt/UserSearchPicker/ConfirmDialog，
 * 由 AdminPage 概览 Tab 底部的一张卡片以展开式挂载（仅 admin 可见）。
 */
import { useState, useEffect, useCallback } from 'react'
import { listKBAuthorized, addKBAuthorized, removeKBAuthorized } from '@/api/kb'
import type { KBAuthorizedUserItem } from '@/api/kb'
import { C, fmt } from './adminConstants'
import { UserSearchPicker } from './UserSearchPicker'
import { ConfirmDialog } from './ConfirmDialog'

export function KBAuthorizedPanel() {
  const [members, setMembers] = useState<KBAuthorizedUserItem[]>([])
  const [loading, setLoading] = useState(true)
  const [addUserId, setAddUserId] = useState('')
  const [addUserName, setAddUserName] = useState('')
  const [addNote, setAddNote] = useState('')
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState('')
  const [confirmRemove, setConfirmRemove] = useState<{ open: boolean; userId: string; name: string }>({
    open: false, userId: '', name: '',
  })

  // ==================== 加载白名单 ====================
  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError('')
      const data = await listKBAuthorized()
      setMembers(data.items || [])
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载白名单失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  // ==================== 新增成员 ====================
  const handleAdd = useCallback(async () => {
    if (!addUserId) { setError('请先选择要加入白名单的用户'); return }
    try {
      setAdding(true)
      setError('')
      await addKBAuthorized({ user_id: addUserId, note: addNote.trim() || undefined })
      setAddUserId(''); setAddUserName(''); setAddNote('')
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '添加失败')
    } finally {
      setAdding(false)
    }
  }, [addUserId, addNote, load])

  // ==================== 移除成员（二次确认）====================
  const doRemove = useCallback(async (userId: string) => {
    try {
      setError('')
      await removeKBAuthorized(userId)
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '移除失败')
    } finally {
      setConfirmRemove({ open: false, userId: '', name: '' })
    }
  }, [load])

  return (
    <div style={{ padding: '16px', background: 'rgba(8,145,178,0.05)', borderTop: `1px dashed ${C.border}` }}>

      {/* 移除二次确认 */}
      {confirmRemove.open && (
        <ConfirmDialog
          title="移出白名单"
          message={`确认将「${confirmRemove.name}」移出知识库访问白名单？移出后该用户将无法再进入知识库压缩系统。`}
          onConfirm={() => doRemove(confirmRemove.userId)}
          onCancel={() => setConfirmRemove({ open: false, userId: '', name: '' })}
        />
      )}

      {/* 标题 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>🔐 知识库访问白名单</span>
          {members.length > 0 && (
            <span style={{ fontSize: '11px', padding: '1px 7px', borderRadius: '10px', background: 'rgba(8,145,178,0.1)', color: '#0891B2', fontWeight: 600 }}>
              共 {members.length} 人
            </span>
          )}
        </div>
      </div>

      {/* 说明 */}
      <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '12px', lineHeight: 1.6 }}>
        白名单成员（及系统管理员）可经隐藏链接 <code style={{ background: C.bg, padding: '1px 6px', borderRadius: '4px', color: C.text }}>/kb-admin/curriculum</code> 进入课标压缩入库系统。
        白名单不绑角色，仅认成员名单。
      </div>

      {/* 错误提示 */}
      {error && (
        <div style={{ fontSize: '12px', color: C.danger, marginBottom: '10px', padding: '8px 12px', background: C.dangerLight, borderRadius: '8px', lineHeight: 1.5 }}>
          {error}
        </div>
      )}

      {/* 成员列表 */}
      {loading ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>加载中...</div>
      ) : members.length === 0 ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '8px 0' }}>暂无授权成员（仅系统管理员可进）。可在下方添加。</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', marginBottom: '14px' }}>
          {members.map(m => (
            <div key={m.user_id} style={{
              display: 'flex', alignItems: 'center', gap: '8px',
              padding: '8px 12px', borderRadius: '8px',
              background: C.white, border: `1px solid ${C.border}`,
            }}>
              {/* 头像 */}
              <div style={{
                width: '28px', height: '28px', borderRadius: '50%', flexShrink: 0,
                background: 'linear-gradient(135deg,#0891B2,#4F7BE8)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                color: '#fff', fontSize: '11px', fontWeight: 700,
              }}>
                {(m.display_name || m.username).charAt(0).toUpperCase()}
              </div>
              {/* 姓名 + 备注 + 授权信息 */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>{m.display_name || m.username}</span>
                  {m.note && (
                    <span style={{ fontSize: '10px', padding: '1px 6px', borderRadius: '8px', background: C.bg, color: C.textSec, border: `1px solid ${C.border}` }}>
                      {m.note}
                    </span>
                  )}
                </div>
                <div style={{ fontSize: '11px', color: C.textMuted }}>
                  @{m.username}
                  {m.granted_by ? ` · 授权人 ${m.granted_by}` : ''}
                  {m.created_at ? ` · ${fmt(m.created_at)}` : ''}
                </div>
              </div>
              {/* 移除按钮 */}
              <button
                onClick={() => setConfirmRemove({ open: true, userId: m.user_id, name: m.display_name || m.username })}
                style={{
                  padding: '4px 10px', borderRadius: '6px',
                  border: '1px solid #FEE2E2', background: '#FEF2F2',
                  color: '#EF4444', fontSize: '11px', cursor: 'pointer',
                  fontWeight: 500, whiteSpace: 'nowrap',
                }}>
                移除
              </button>
            </div>
          ))}
        </div>
      )}

      {/* 添加区域 */}
      <div style={{ background: C.white, borderRadius: '10px', border: `1px solid ${C.border}`, padding: '12px' }}>
        <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '10px' }}>添加授权成员</div>
        <UserSearchPicker
          label=""
          value={addUserId} valueName={addUserName}
          onChange={(id, n) => { setAddUserId(id); setAddUserName(n) }}
          placeholder="输入用户名搜索要授权的用户..."
        />
        <input
          value={addNote}
          onChange={e => setAddNote(e.target.value)}
          placeholder="备注（可选，如：数学教研组录入员）"
          style={{ width: '100%', padding: '9px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none', boxSizing: 'border-box', marginBottom: '4px' }}
        />
        <button
          onClick={handleAdd} disabled={adding || !addUserId}
          style={{
            width: '100%', padding: '8px', borderRadius: '7px', border: 'none', marginTop: '6px',
            background: (!addUserId || adding) ? '#E5E7EB' : '#0891B2',
            color: (!addUserId || adding) ? '#9CA3AF' : '#fff',
            fontSize: '13px', fontWeight: 600,
            cursor: (!addUserId || adding) ? 'not-allowed' : 'pointer',
          }}>
          {adding ? '添加中...' : '+ 加入白名单'}
        </button>
      </div>
    </div>
  )
}

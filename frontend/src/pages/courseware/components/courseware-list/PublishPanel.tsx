/**
 * 发布面板弹窗 — PublishPanel.tsx
 *
 * 从 CoursewareListPage.tsx 拆出。集中处理一张"我的课件"的:
 *   ① 发布态切换(共享给同校同组 / 仅标记完成 / 撤回到私有)
 *   ② 源代码开放范围设置(下拉切换即时保存)
 *
 * 文案点明「可见范围(谁能看渲染效果)↔ 代码复制权(谁能复制源码)」双轨解耦,
 * 帮助老师理解产权设置:可以共享给全校看,但不开放源码。
 *
 * 弹窗内本地态(pubState/codeScope)操作后即时调接口并就地刷新徽章;
 * 点底部"完成"触发 onChanged,由父组件关闭弹窗 + 重新加载列表。
 */

import { useState } from 'react'
import { publishCourseware, setCodeShareScope, CW_PUBLISH_STATE_CONFIG, CW_CODE_SHARE_SCOPE_OPTIONS } from '@/api/coursewares'
import type { CoursewareListItem } from '@/api/coursewares'
import { C, btnBase } from './listConstants'

export default function PublishPanel({ item, onClose, onChanged }: {
  item: CoursewareListItem
  onClose: () => void
  onChanged: () => void
}) {
  // 当前发布态 / 代码范围(弹窗内本地态,操作后即时调接口并刷新)
  const [pubState, setPubState] = useState(item.publish_state || 'private')
  const [codeScope, setCodeScope] = useState(item.code_share_scope || 'none')
  const [saving, setSaving] = useState(false)
  const [msg, setMsg] = useState('')

  const curPub = CW_PUBLISH_STATE_CONFIG[pubState] || CW_PUBLISH_STATE_CONFIG.private

  // 发布态操作:published_personal / published_shared / private
  const doPublish = async (target: 'published_personal' | 'published_shared' | 'private') => {
    if (saving) return
    setSaving(true); setMsg('')
    try {
      await publishCourseware(item.id, target)
      setPubState(target)
      setMsg('✅ ' + (CW_PUBLISH_STATE_CONFIG[target]?.label || target) + ' 已生效')
    } catch (e) {
      setMsg('❌ 操作失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setSaving(false) }
  }

  // 代码范围操作:下拉切换即时保存
  const doSetCodeScope = async (scope: string) => {
    if (saving) return
    setSaving(true); setMsg('')
    try {
      await setCodeShareScope(item.id, scope as 'none' | 'group' | 'school' | 'region' | 'public')
      setCodeScope(scope)
      setMsg('✅ 源码开放范围已更新')
    } catch (e) {
      setMsg('❌ 设置失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setSaving(false) }
  }

  // 当前可执行的发布动作(按当前态智能给出)
  const isShared = pubState === 'published_shared'
  const isPersonal = pubState === 'published_personal'

  const actionBtn = (label: string, color: string, bg: string, border: string, onClick: () => void, disabled?: boolean): React.JSX.Element => (
    <button onClick={onClick} disabled={saving || disabled}
      style={{
        flex: 1, padding: '10px', borderRadius: '8px', border: `1px solid ${border}`,
        background: (saving || disabled) ? '#F3F4F6' : bg, color: (saving || disabled) ? '#9CA3AF' : color,
        fontSize: '13px', fontWeight: 600, cursor: (saving || disabled) ? 'default' : 'pointer',
      }}>{label}</button>
  )

  return (
    <div style={{ position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.5)', zIndex: 10000, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={onClose}>
      <div style={{ background: '#fff', borderRadius: '16px', width: '90%', maxWidth: '480px', maxHeight: '85vh', overflow: 'auto', padding: '28px' }}
        onClick={e => e.stopPropagation()}>

        {/* 标题 */}
        <div style={{ fontSize: '18px', fontWeight: 700, color: C.textPrimary, marginBottom: '4px' }}>🔗 发布与共享</div>
        <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '6px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{item.title}</div>
        {/* 当前发布态 */}
        <div style={{ display: 'inline-flex', alignItems: 'center', gap: '6px', marginBottom: '18px' }}>
          <span style={{ fontSize: '12px', color: C.textMuted }}>当前状态:</span>
          <span style={{ padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 600, color: curPub.color, background: curPub.bg }}>{curPub.label}</span>
        </div>

        {/* 双轨解耦说明 */}
        <div style={{
          padding: '12px 14px', borderRadius: '10px', marginBottom: '20px',
          background: 'rgba(245,158,11,0.06)', border: '1px solid rgba(245,158,11,0.2)',
          fontSize: '12px', color: '#92400E', lineHeight: 1.6,
        }}>
          💡 <b>可见范围</b>(谁能看课件效果)由「发布共享」决定;<b>源码复制权</b>(谁能复制源码二次创作)由下方「源码开放范围」单独决定。两者独立——你可以共享给全校看,但不开放源码。
        </div>

        {/* 一、发布动作 */}
        <div style={{ fontSize: '13px', fontWeight: 700, color: C.textPrimary, marginBottom: '10px' }}>① 发布共享</div>
        <div style={{ display: 'flex', gap: '8px', marginBottom: '10px' }}>
          {/* 共享发布 */}
          {actionBtn(
            isShared ? '✅ 已共享' : '🌐 共享给同校/同组',
            '#fff', 'linear-gradient(135deg, #059669, #10B981)', '#059669',
            () => doPublish('published_shared'),
            isShared,
          )}
        </div>
        <div style={{ display: 'flex', gap: '8px', marginBottom: '6px' }}>
          {/* 个人发布 */}
          {actionBtn(
            isPersonal ? '✅ 个人已发布' : '📌 仅标记完成(不共享)',
            '#0891B2', '#CFFAFE', '#A5F3FC',
            () => doPublish('published_personal'),
            isPersonal,
          )}
          {/* 撤回到私有 */}
          {actionBtn(
            '↩️ 撤回到私有',
            '#6B7280', '#F3F4F6', '#E5E7EB',
            () => doPublish('private'),
            pubState === 'private',
          )}
        </div>
        <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '20px', lineHeight: 1.5 }}>
          共享后,同校/同教研组的老师能在「共享课件库」看到并放映这套课件。
        </div>

        {/* 二、源码开放范围 */}
        <div style={{ fontSize: '13px', fontWeight: 700, color: C.textPrimary, marginBottom: '10px' }}>② 源码开放范围(产权保护)</div>
        <select value={codeScope} onChange={e => doSetCodeScope(e.target.value)} disabled={saving}
          style={{ width: '100%', padding: '10px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: '#fff', cursor: saving ? 'default' : 'pointer' }}>
          {CW_CODE_SHARE_SCOPE_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
        <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '8px', lineHeight: 1.5 }}>
          选择谁能「复制到我的」二次创作。默认不开放——别人只能看、能放映,但拿不走源码。
        </div>

        {/* 操作反馈 */}
        {msg && (
          <div style={{ marginTop: '16px', padding: '8px 12px', borderRadius: '8px', fontSize: '13px',
            background: msg.startsWith('✅') ? '#D1FAE5' : '#FEE2E2',
            color: msg.startsWith('✅') ? '#059669' : '#DC2626' }}>{msg}</div>
        )}

        {/* 底部完成 */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '20px' }}>
          <button onClick={onChanged} style={{
            ...btnBase, border: 'none', background: 'linear-gradient(135deg, #F59E0B, #EF4444)',
            color: '#fff', fontWeight: 600,
          }}>完成</button>
        </div>
      </div>
    </div>
  )
}

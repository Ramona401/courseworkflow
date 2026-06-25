/**
 * SchoolOverseasInline.tsx — 学校境外模型授权·单校就近开关（批二-B 新增）
 *
 * 用途：
 *   挂在 AdminPage「组织架构」Tab 的每张学校卡片展开区，让 admin 在浏览组织架构时
 *   就近查看/切换该校的境外模型授权（无需跳回概览 Tab 的全局授权卡片）。
 *   与概览 Tab 的 OverseasPolicyPanel 共用同一套后端 API，数据一致。
 *
 * 交互：
 *   - 挂载即查该校当前策略（getSchoolModelPolicy）：境外已授权 / 已关闭(境内) / 未登记(默认境内)。
 *   - 一个主开关：开启境外 ⇄ 切到境内（setSchoolModelPolicy，UPSERT）。
 *   - 可选备注（仅在“开启境外”时随手填，便于审计）。
 *   - 已登记的可“移除记录”（deleteSchoolModelPolicy，=回到默认境内）。
 *
 * 权限：仅 admin（调用方 AdminPage 用 isFullAdmin 门控按钮，不对 region_admin/senior 出现）。
 * 即时生效：分流模块对“学校是否授权”每次实时查库无缓存，保存即生效。
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getSchoolModelPolicy, setSchoolModelPolicy, deleteSchoolModelPolicy,
} from '@/api/admin'
import type { SchoolModelPolicyView } from '@/api/admin'
import { C } from './adminConstants'

interface SchoolOverseasInlineProps {
  /** 学校组织ID */
  schoolId: string
  /** 学校名称（仅用于文案展示） */
  schoolName: string
  /** 收起面板回调 */
  onClose: () => void
}

export function SchoolOverseasInline({ schoolId, schoolName, onClose }: SchoolOverseasInlineProps) {
  /** 当前策略（null=加载中） */
  const [policy, setPolicy] = useState<SchoolModelPolicyView | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  /** 备注输入（开启境外时可随手填） */
  const [note, setNote] = useState('')

  // ==================== 加载该校当前策略 ====================
  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError('')
      const p = await getSchoolModelPolicy(schoolId)
      setPolicy(p)
      setNote(p.note || '')
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载策略失败')
    } finally {
      setLoading(false)
    }
  }, [schoolId])

  useEffect(() => { load() }, [load])

  // ==================== 开关：境外⇄境内 ====================
  const handleToggle = useCallback(async (toOverseas: boolean) => {
    try {
      setSaving(true)
      setError('')
      const p = await setSchoolModelPolicy(schoolId, {
        overseas_enabled: toOverseas,
        note: note.trim(),
      })
      setPolicy(p)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '切换失败')
    } finally {
      setSaving(false)
    }
  }, [schoolId, note])

  // ==================== 移除记录（=回到默认境内）====================
  const handleRemove = useCallback(async () => {
    try {
      setSaving(true)
      setError('')
      await deleteSchoolModelPolicy(schoolId)
      // 移除后回到“未登记=默认境内”态
      setPolicy({ school_id: schoolId, overseas_enabled: false, note: '', has_record: false })
      setNote('')
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '移除失败')
    } finally {
      setSaving(false)
    }
  }, [schoolId])

  // 当前是否已授权境外
  const enabled = !!policy?.overseas_enabled
  // 状态文案
  const statusText = loading
    ? '加载中...'
    : enabled
      ? '🌐 境外已授权'
      : (policy?.has_record ? '🇨🇳 已关闭（境内）' : '🇨🇳 未登记（默认境内）')

  return (
    <div style={{ padding: '14px 18px', background: 'rgba(245,158,11,0.04)', borderTop: `1px dashed ${C.border}` }}>

      {/* 标题 + 收起 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '10px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>🌐 境外模型授权</span>
          <span style={{
            fontSize: '11px', padding: '2px 9px', borderRadius: '10px', fontWeight: 700,
            background: enabled ? C.successLight : C.bg,
            color: enabled ? C.success : C.textMuted,
            border: `1px solid ${enabled ? C.success + '44' : C.border}`,
          }}>
            {statusText}
          </span>
        </div>
        <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '12px', color: C.textMuted }}>
          收起 ▲
        </button>
      </div>

      {/* 错误提示 */}
      {error && (
        <div style={{ fontSize: '12px', color: C.danger, marginBottom: '10px', padding: '8px 12px', background: C.dangerLight, borderRadius: '8px', lineHeight: 1.5 }}>
          ⚠ {error}
        </div>
      )}

      {/* 说明 */}
      <div style={{ fontSize: '12px', color: C.textSec, marginBottom: '12px', lineHeight: 1.6 }}>
        默认「{schoolName}」走境内模型（qwen-max）。开启后该校老师的 AI 文本调用放行境外模型（claude/gemini 等），即时生效。
      </div>

      {loading ? (
        <div style={{ fontSize: '12px', color: C.textMuted, padding: '4px 0' }}>加载中...</div>
      ) : (
        <>
          {/* 备注（开启境外时可填） */}
          <input
            value={note}
            onChange={e => setNote(e.target.value)}
            placeholder="备注（可选，授权原因/用途）"
            style={{
              width: '100%', padding: '8px 12px', borderRadius: '8px', marginBottom: '10px',
              border: `1px solid ${C.border}`, fontSize: '13px', outline: 'none',
              background: C.white, color: C.text, boxSizing: 'border-box',
            }}
          />

          {/* 操作按钮区 */}
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
            {enabled ? (
              <button
                onClick={() => handleToggle(false)}
                disabled={saving}
                style={{
                  padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`,
                  background: saving ? '#E5E7EB' : C.white, color: saving ? '#9CA3AF' : C.textSec,
                  fontSize: '13px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer',
                }}>
                {saving ? '处理中...' : '切到境内'}
              </button>
            ) : (
              <button
                onClick={() => handleToggle(true)}
                disabled={saving}
                style={{
                  padding: '8px 16px', borderRadius: '8px', border: 'none',
                  background: saving ? '#E5E7EB' : C.warning, color: saving ? '#9CA3AF' : '#fff',
                  fontSize: '13px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer',
                }}>
                {saving ? '处理中...' : '🌐 开启境外'}
              </button>
            )}

            {/* 已登记的可移除记录 */}
            {policy?.has_record && (
              <button
                onClick={handleRemove}
                disabled={saving}
                style={{
                  padding: '8px 14px', borderRadius: '8px',
                  border: '1px solid #FEE2E2', background: '#FEF2F2',
                  color: '#EF4444', fontSize: '13px', cursor: saving ? 'not-allowed' : 'pointer', fontWeight: 500,
                }}>
                移除记录
              </button>
            )}
          </div>
        </>
      )}
    </div>
  )
}

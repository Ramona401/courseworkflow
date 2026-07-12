/**
 * MyOrganizationTab — 个人中心「我的组织」Tab
 *
 * 用途（测试反馈 7-1 #2）：
 *   让用户看清自己的组织归属和职位——所属区域、所属学校、在哪些教研组、
 *   在各组里是组长/骨干/普通成员，解决"看不到自己的角色、对权限层级茫然"。
 *
 * 数据来自 GET /api/v1/account/organization（只查自己）。
 * 展示逻辑：
 *   - 顶部：系统身份说明（区分"系统角色"门票 与 "教研职位"）
 *   - 学校归属卡片：逐校展示（区域 → 学校，是否本校管理员徽标）
 *   - 教研组卡片：逐组展示（学校 · 教研组 · 我的角色徽章 · 学科学段）
 *   - 空归属友好提示
 *
 * 纯展示组件，自拉数据，无需父组件传参。
 */
import { useState, useEffect, useCallback } from 'react'
import { getMyOrganization } from '@/api/account'
import type { UserOrganizationProfile } from '@/api/account'

const C = {
  primary:   '#4F7BE8',
  text:      '#1F2937',
  textSec:   '#6B7280',
  textMuted: '#9CA3AF',
  border:    '#E5E7EB',
  bg:        '#F9FAFB',
  white:     '#FFFFFF',
}

/** 教研职位角色 → 中文名 + 配色 */
const GROUP_ROLE_META: Record<string, { name: string; bg: string; color: string }> = {
  lead:     { name: '教研组长', bg: 'rgba(245,158,11,0.12)', color: '#D97706' },
  backbone: { name: '骨干教师', bg: 'rgba(79,123,232,0.12)', color: '#4F7BE8' },
  member:   { name: '普通成员', bg: 'rgba(107,114,128,0.12)', color: '#6B7280' },
}

/** 系统角色 → 中文名（与全站口径一致） */
const SYS_ROLE_NAMES: Record<string, string> = {
  admin:              '系统管理员',
  region_admin:       '区域管理员',
  district_inspector: '区域教研员',
  senior_operator:    '学校管理员',
  operator:           '骨干教师',
  viewer:             '普通教师',
}

interface Props {
  /** 当前用户的系统角色（用于顶部说明，来自 AccountPage 的 profile.role） */
  systemRole?: string
}

export default function MyOrganizationTab({ systemRole }: Props) {
  const [data, setData]       = useState<UserOrganizationProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState('')

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError('')
      const res = await getMyOrganization()
      setData(res)
    } catch {
      setError('获取组织归属信息失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  if (loading) {
    return (
      <div style={{ background: C.white, borderRadius: '16px', border: `1px solid ${C.border}`, padding: '48px', textAlign: 'center', color: C.textMuted, fontSize: '14px' }}>
        加载组织信息...
      </div>
    )
  }

  if (error) {
    return (
      <div style={{ background: C.white, borderRadius: '16px', border: `1px solid ${C.border}`, padding: '32px', textAlign: 'center' }}>
        <div style={{ color: '#EF4444', fontSize: '14px', marginBottom: '12px' }}>{error}</div>
        <button onClick={load} style={{ padding: '8px 20px', borderRadius: '8px', border: `1px solid ${C.primary}`, background: 'rgba(79,123,232,0.08)', color: C.primary, fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>重新加载</button>
      </div>
    )
  }

  const schools = data?.schools ?? []
  const groups  = data?.groups  ?? []
  const hasNothing = schools.length === 0 && groups.length === 0

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {/* 系统身份说明卡片 */}
      <div style={{ background: C.white, borderRadius: '16px', border: `1px solid ${C.border}`, padding: '20px 24px', boxShadow: '0 1px 4px rgba(0,0,0,0.04)' }}>
        <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '10px' }}>关于你的身份</div>
        <div style={{ fontSize: '13px', color: C.textSec, lineHeight: 1.7 }}>
          平台有两套并行的身份维度：
          <br />
          · <b style={{ color: C.text }}>系统角色</b>（{SYS_ROLE_NAMES[systemRole || ''] || systemRole || '—'}）决定你能进入哪些功能板块（如课件审核入口、用户管理入口）。
          <br />
          · <b style={{ color: C.text }}>教研职位</b>（组长/骨干/普通成员）决定你在具体教研组里的权限（如能否创建备课配方、发布模板、审核本组教案）。
          <br />
          下方是你当前所属的组织与在各教研组里的职位。
        </div>
      </div>

      {/* 空归属提示 */}
      {hasNothing && (
        <div style={{ background: C.white, borderRadius: '16px', border: `1px dashed ${C.border}`, padding: '40px 24px', textAlign: 'center' }}>
          <div style={{ fontSize: '32px', marginBottom: '12px' }}>🏫</div>
          <div style={{ fontSize: '15px', color: C.text, fontWeight: 600, marginBottom: '6px' }}>你还未加入任何学校或教研组</div>
          <div style={{ fontSize: '13px', color: C.textMuted, lineHeight: 1.6 }}>
            你目前可以独立使用平台的个人备课功能。<br />
            如需加入学校教研组，请联系你所在学校的管理员将你添加进对应教研组。
          </div>
        </div>
      )}

      {/* 学校归属 */}
      {schools.length > 0 && (
        <div style={{ background: C.white, borderRadius: '16px', border: `1px solid ${C.border}`, padding: '20px 24px', boxShadow: '0 1px 4px rgba(0,0,0,0.04)' }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '14px' }}>🏛️ 我所属的学校</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {schools.map(s => (
              <div key={s.school_id} style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '14px 16px', borderRadius: '12px', background: C.bg, border: `1px solid ${C.border}` }}>
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                    <span style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>{s.school_name}</span>
                    {s.is_school_admin && (
                      <span style={{ padding: '2px 8px', borderRadius: '10px', fontSize: '12px', fontWeight: 600, background: 'rgba(245,158,11,0.12)', color: '#D97706' }}>学校管理员</span>
                    )}
                  </div>
                  <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px' }}>
                    {s.region_name ? `所属区域：${s.region_name}` : '未归属区域'}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 教研组归属 */}
      {groups.length > 0 && (
        <div style={{ background: C.white, borderRadius: '16px', border: `1px solid ${C.border}`, padding: '20px 24px', boxShadow: '0 1px 4px rgba(0,0,0,0.04)' }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '14px' }}>👥 我所在的教研组</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {groups.map(g => {
              const roleMeta = GROUP_ROLE_META[g.my_role] || GROUP_ROLE_META.member
              return (
                <div key={g.group_id} style={{ padding: '14px 16px', borderRadius: '12px', background: C.bg, border: `1px solid ${C.border}` }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap', marginBottom: '6px' }}>
                    <span style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>{g.group_name}</span>
                    <span style={{ padding: '2px 10px', borderRadius: '10px', fontSize: '12px', fontWeight: 600, background: roleMeta.bg, color: roleMeta.color }}>{roleMeta.name}</span>
                  </div>
                  <div style={{ fontSize: '12px', color: C.textMuted, lineHeight: 1.6 }}>
                    {g.school_name && <span>所属学校：{g.school_name}　</span>}
                    {g.subject && <span>学科：{g.subject}　</span>}
                    {g.grade_range && <span>学段：{g.grade_range}</span>}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}

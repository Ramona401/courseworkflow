/**
 * UserSearchPicker.tsx — 通用用户搜索选择器
 * 输入关键词实时搜索，下拉选中后显示已选用户
 */
import { useState, useCallback, useRef } from 'react'
import { getAdminUsers } from '@/api/admin'
import type { AdminUserListItem } from '@/api/admin'
import { C } from './adminConstants'
import { RoleBadge } from './adminShared'

interface UserSearchPickerProps {
  label: string
  value: string
  valueName: string
  onChange: (id: string, name: string) => void
  placeholder?: string
}

export function UserSearchPicker({
  label, value, valueName, onChange, placeholder,
}: UserSearchPickerProps) {
  const [kw, setKw]               = useState('')
  const [results, setResults]     = useState<AdminUserListItem[]>([])
  const [searching, setSearching] = useState(false)
  const [open, setOpen]           = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // 防抖搜索
  const search = useCallback(async (q: string) => {
    if (!q.trim()) { setResults([]); return }
    try {
      setSearching(true)
      const data = await getAdminUsers({ keyword: q, page: 1, page_size: 8 })
      setResults(data.users)
    } catch { /* 忽略 */ } finally { setSearching(false) }
  }, [])

  const handleKwChange = (v: string) => {
    setKw(v)
    setOpen(true)
    if (timer.current) clearTimeout(timer.current)
    timer.current = setTimeout(() => search(v), 350)
  }

  const handleSelect = (u: AdminUserListItem) => {
    onChange(u.id, u.display_name || u.username)
    setKw(''); setResults([]); setOpen(false)
  }

  const handleClear = () => {
    onChange('', '')
    setKw(''); setResults([])
  }

  return (
    <div style={{ marginBottom: '14px' }}>
      {/* 标签 */}
      {label && (
        <label style={{
          display: 'block', fontSize: '13px', fontWeight: 600,
          color: C.text, marginBottom: '6px',
        }}>
          {label}
        </label>
      )}

      {/* 已选状态 */}
      {value ? (
        <div style={{
          display: 'flex', alignItems: 'center', gap: '8px',
          padding: '8px 12px', borderRadius: '8px',
          border: `1px solid ${C.primary}`, background: C.primaryLight,
        }}>
          <span style={{ flex: 1, fontSize: '13px', color: C.primary, fontWeight: 500 }}>
            ✓ {valueName}
          </span>
          <button
            onClick={handleClear}
            style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '14px', color: C.textMuted, padding: '0 4px' }}>
            ×
          </button>
        </div>
      ) : (
        // 搜索输入框
        <div style={{ position: 'relative' }}>
          <input
            value={kw}
            onChange={e => handleKwChange(e.target.value)}
            onFocus={e => { kw && setOpen(true); e.currentTarget.style.borderColor = C.primary }}
            onBlur={() => setTimeout(() => setOpen(false), 200)}
            placeholder={placeholder || '输入用户名或显示名搜索...'}
            style={{
              width: '100%', padding: '9px 14px', borderRadius: '8px',
              border: `1px solid ${C.border}`, fontSize: '14px',
              outline: 'none', boxSizing: 'border-box',
            }}
          />
          {/* 下拉结果列表 */}
          {open && (results.length > 0 || searching) && (
            <div style={{
              position: 'absolute', top: '100%', left: 0, right: 0, zIndex: 200,
              background: C.white, border: `1px solid ${C.border}`,
              borderRadius: '8px', boxShadow: '0 4px 16px rgba(0,0,0,0.12)',
              marginTop: '4px', overflow: 'hidden',
            }}>
              {searching ? (
                <div style={{ padding: '12px', textAlign: 'center', fontSize: '13px', color: C.textMuted }}>
                  搜索中...
                </div>
              ) : (
                results.map(u => (
                  <div
                    key={u.id}
                    onMouseDown={() => handleSelect(u)}
                    style={{
                      padding: '10px 14px', cursor: 'pointer',
                      borderBottom: `1px solid ${C.border}`,
                      display: 'flex', alignItems: 'center', gap: '10px',
                    }}
                    onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
                    onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.white }}>
                    {/* 头像 */}
                    <div style={{
                      width: '28px', height: '28px', borderRadius: '50%', flexShrink: 0,
                      background: `linear-gradient(135deg,${C.primary},#7C3AED)`,
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                      color: '#fff', fontSize: '11px', fontWeight: 700,
                    }}>
                      {(u.display_name || u.username).charAt(0).toUpperCase()}
                    </div>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontSize: '13px', fontWeight: 500, color: C.text }}>
                        {u.display_name}
                      </div>
                      <div style={{ fontSize: '11px', color: C.textMuted }}>@{u.username}</div>
                    </div>
                    <RoleBadge role={u.role} />
                  </div>
                ))
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

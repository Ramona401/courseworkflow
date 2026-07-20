/**
 * OrganizationEducationDomainsPanel — 组织教育域只读管理页
 *
 * 规则：
 *   - 区域创建时由后端和数据库强制写入mixed；
 *   - 学校创建时必须选择k12、vocational或adult；
 *   - 学校教育域创建后永久锁定；
 *   - 页面只提供查看、搜索和筛选；
 *   - 不提供下拉框、保存按钮、确认弹窗或换域请求。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react'
import {
  getOrganizationEducationDomains,
} from '@/api/organization-education-domains'
import type {
  OrganizationEducationDomainItem,
} from '@/api/organization-education-domains'
import type {
  EducationDomain,
} from '@/education-domain/types'
import {
  EDUCATION_DOMAIN_LABELS,
} from '@/education-domain/types'
import { C } from '@/pages/admin/components/adminConstants'
import {
  Toast,
} from '@/pages/admin/components/adminShared'

const DOMAIN_FILTER_OPTIONS: {
  value: EducationDomain
  label: string
}[] = [
  { value: 'k12', label: '中小学' },
  { value: 'vocational', label: '职业教育' },
  { value: 'adult', label: '成人教育' },
  { value: 'mixed', label: '跨域管理' },
]

function domainBadgeStyle(
  domain: EducationDomain,
): React.CSSProperties {
  const palette: Record<
    EducationDomain,
    { background: string; color: string }
  > = {
    k12: {
      background: 'rgba(79,123,232,0.10)',
      color: '#4F7BE8',
    },
    vocational: {
      background: 'rgba(16,185,129,0.10)',
      color: '#059669',
    },
    adult: {
      background: 'rgba(245,158,11,0.12)',
      color: '#D97706',
    },
    mixed: {
      background: 'rgba(124,58,237,0.10)',
      color: '#7C3AED',
    },
  }

  return {
    display: 'inline-flex',
    alignItems: 'center',
    padding: '3px 10px',
    borderRadius: '999px',
    fontSize: '11px',
    fontWeight: 700,
    ...palette[domain],
  }
}

export default function OrganizationEducationDomainsPanel() {
  const [items, setItems] = useState<
    OrganizationEducationDomainItem[]
  >([])

  const [loading, setLoading] = useState(true)
  const [keyword, setKeyword] = useState('')

  const [domainFilter, setDomainFilter] =
    useState<'all' | EducationDomain>('all')

  const [toast, setToast] = useState<{
    message: string
    type: 'success' | 'error'
  } | null>(null)

  const showToast = useCallback((
    message: string,
    type: 'success' | 'error',
  ) => {
    setToast({ message, type })
  }, [])

  const load = useCallback(async () => {
    try {
      setLoading(true)

      const result =
        await getOrganizationEducationDomains()

      setItems(result.organizations || [])
    } catch (error) {
      showToast(
        error instanceof Error
          ? error.message
          : '加载组织教育域失败',
        'error',
      )
    } finally {
      setLoading(false)
    }
  }, [showToast])

  useEffect(() => {
    load()
  }, [load])

  const filtered = useMemo(() => {
    const normalizedKeyword = keyword.trim().toLowerCase()

    return items.filter(item => {
      if (
        domainFilter !== 'all'
        && item.education_domain !== domainFilter
      ) {
        return false
      }

      if (!normalizedKeyword) return true

      return item.name.toLowerCase().includes(
        normalizedKeyword,
      ) || item.parent_name.toLowerCase().includes(
        normalizedKeyword,
      )
    })
  }, [
    items,
    keyword,
    domainFilter,
  ])

  const inputStyle: React.CSSProperties = {
    padding: '8px 11px',
    borderRadius: '8px',
    border: `1px solid ${C.border}`,
    background: C.white,
    color: C.text,
    fontSize: '13px',
    outline: 'none',
  }

  return (
    <div>
      {toast && (
        <Toast
          message={toast.message}
          type={toast.type}
          onClose={() => setToast(null)}
        />
      )}

      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'flex-start',
        gap: '20px',
        marginBottom: '16px',
      }}>
        <div>
          <div style={{
            fontSize: '16px',
            fontWeight: 700,
            color: C.text,
          }}>
            🏫 组织教育域
          </div>

          <div style={{
            fontSize: '12px',
            color: C.textMuted,
            marginTop: '5px',
            lineHeight: 1.7,
            maxWidth: '850px',
          }}>
            教育域是组织创建时确定的永久业务属性。
            区域固定使用跨域管理；学校创建时必须选择中小学、
            职业教育或成人教育，创建成功后不能通过普通业务修改。
          </div>

          <div style={{
            marginTop: '8px',
            display: 'inline-flex',
            alignItems: 'center',
            gap: '5px',
            padding: '5px 10px',
            borderRadius: '8px',
            background: C.bg,
            color: C.textSec,
            fontSize: '11px',
            border: `1px solid ${C.border}`,
          }}>
            <span aria-hidden="true">🔒</span>
            当前页面为只读视图
          </div>
        </div>

        <div style={{
          display: 'flex',
          gap: '8px',
          flexShrink: 0,
        }}>
          <input
            value={keyword}
            onChange={event => setKeyword(event.target.value)}
            placeholder="搜索组织或上级区域"
            style={{
              ...inputStyle,
              width: '190px',
            }}
          />

          <select
            value={domainFilter}
            onChange={event => {
              setDomainFilter(
                event.target.value as
                  | 'all'
                  | EducationDomain,
              )
            }}
            style={inputStyle}
          >
            <option value="all">全部教育域</option>

            {DOMAIN_FILTER_OPTIONS.map(option => (
              <option
                key={option.value}
                value={option.value}
              >
                {option.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div style={{
        background: C.white,
        borderRadius: '14px',
        border: `1px solid ${C.border}`,
        overflow: 'hidden',
      }}>
        <div style={{
          display: 'grid',
          gridTemplateColumns:
            '2.3fr 1.5fr 100px 150px 100px 100px 170px',
          gap: '12px',
          padding: '12px 18px',
          background: C.bg,
          borderBottom: `1px solid ${C.border}`,
          fontSize: '12px',
          fontWeight: 600,
          color: C.textSec,
        }}>
          <span>组织</span>
          <span>上级区域</span>
          <span>类型</span>
          <span>教育域</span>
          <span>教研组</span>
          <span>成员</span>
          <span>规则</span>
        </div>

        {loading ? (
          <div style={{
            padding: '44px',
            textAlign: 'center',
            color: C.textMuted,
          }}>
            加载中...
          </div>
        ) : filtered.length === 0 ? (
          <div style={{
            padding: '44px',
            textAlign: 'center',
            color: C.textMuted,
          }}>
            没有符合条件的组织
          </div>
        ) : (
          filtered.map((item, index) => (
            <div
              key={item.id}
              style={{
                display: 'grid',
                gridTemplateColumns:
                  '2.3fr 1.5fr 100px 150px 100px 100px 170px',
                gap: '12px',
                padding: '13px 18px',
                alignItems: 'center',
                borderBottom:
                  index < filtered.length - 1
                    ? `1px solid ${C.border}`
                    : 'none',
                fontSize: '13px',
              }}
            >
              <div>
                <div style={{
                  fontWeight: 600,
                  color: C.text,
                }}>
                  {item.name}
                </div>

                {item.status !== 'active' && (
                  <div style={{
                    fontSize: '10px',
                    color: C.danger,
                    marginTop: '2px',
                  }}>
                    已禁用
                  </div>
                )}
              </div>

              <span style={{
                color: item.parent_name
                  ? C.textSec
                  : C.textMuted,
              }}>
                {item.parent_name || '—'}
              </span>

              <span style={{ color: C.textSec }}>
                {item.type === 'region'
                  ? '区域'
                  : '学校'}
              </span>

              <span>
                <span style={
                  domainBadgeStyle(
                    item.education_domain,
                  )
                }>
                  {
                    EDUCATION_DOMAIN_LABELS[
                      item.education_domain
                    ]
                  }
                </span>
              </span>

              <span style={{ color: C.textSec }}>
                {item.group_count}
              </span>

              <span style={{ color: C.textSec }}>
                {item.member_count}
              </span>

              <span style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: '5px',
                color: C.textMuted,
                fontSize: '11px',
              }}>
                <span aria-hidden="true">🔒</span>

                {item.type === 'region'
                  ? '固定为跨域管理'
                  : '创建后不可修改'}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

/**
 * 课件组件库页面 — 教育域隔离主控制器。
 *
 * 本文件只负责：
 *   - 可信教育画像；
 *   - 列表筛选和请求；
 *   - 删除与K12种子动作；
 *   - 详情加载；
 *   - 组合列表和详情子模块。
 *
 * 展示实现拆到cw-components目录，避免单文件继续膨胀。
 */
import {
  useCallback,
  useEffect,
  useState,
} from 'react'
import {
  deleteCWComponent,
  getCWComponent,
  getCWComponents,
  seedCoursewareData,
} from '@/api/coursewares'
import type {
  CWComponentFull,
  CWComponentListItem,
  SeedResult,
} from '@/api/coursewares'
import {
  EDUCATION_DOMAIN_LABELS,
  type ResourceEducationDomain,
} from '@/education-domain/types'
import { useEducationProfile } from '@/hooks/useEducationProfile'
import { useAuth } from '@/store/auth'
import CWComponentDetailModal from './cw-components/CWComponentDetailModal'
import CWComponentGrid from './cw-components/CWComponentGrid'
import {
  CW_COMPONENT_COLORS as C,
  CW_COMPONENT_DOMAIN_OPTIONS,
  CW_COMPONENT_TYPE_FILTERS,
} from './cw-components/cwComponentUi'

export default function CWComponentsPage() {
  const { user } = useAuth()
  const {
    domain,
    isMixed,
    ready,
    error: educationError,
  } = useEducationProfile()

  const isAdmin = user?.role === 'admin'
  const canRunSeed = isAdmin && isMixed

  const [items, setItems] =
    useState<CWComponentListItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [domainFilter, setDomainFilter] =
    useState<ResourceEducationDomain | ''>('')
  const [detailComp, setDetailComp] =
    useState<CWComponentFull | null>(null)
  const [detailLoading, setDetailLoading] =
    useState(false)
  const [seeding, setSeeding] = useState(false)

  const loadData = useCallback(async () => {
    if (!ready) {
      setItems([])
      setTotal(0)
      setLoading(false)
      return
    }

    setLoading(true)
    setLoadError('')

    try {
      const response = await getCWComponents({
        component_type:
          typeFilter || undefined,
        education_domain:
          isMixed && domainFilter
            ? domainFilter
            : undefined,
        limit: 100,
      })

      setItems(response.components || [])
      setTotal(response.total || 0)
    } catch (error) {
      console.error(
        '加载课件组件失败',
        error,
      )
      setItems([])
      setTotal(0)
      setLoadError(
        error instanceof Error
          ? error.message
          : '组件列表加载失败',
      )
    } finally {
      setLoading(false)
    }
  }, [
    ready,
    typeFilter,
    isMixed,
    domainFilter,
  ])

  useEffect(() => {
    loadData()
  }, [loadData])

  useEffect(() => {
    if (!isMixed && domainFilter) {
      setDomainFilter('')
    }
  }, [
    isMixed,
    domainFilter,
  ])

  const openDetail = async (id: string) => {
    setDetailLoading(true)
    setDetailComp(null)

    try {
      setDetailComp(
        await getCWComponent(id),
      )
    } catch (error) {
      console.error(
        '加载课件组件详情失败',
        error,
      )
      window.alert('加载失败')
    } finally {
      setDetailLoading(false)
    }
  }

  const canMutateComponent = (
    component: CWComponentListItem,
  ) => {
    if (!isAdmin) {
      return false
    }

    if (isMixed) {
      return true
    }

    return (
      component.education_domain === domain
    )
  }

  const handleDelete = async (
    component: CWComponentListItem,
  ) => {
    if (!canMutateComponent(component)) {
      return
    }

    if (
      !window.confirm(
        `确定删除组件「${component.name}」？`,
      )
    ) {
      return
    }

    try {
      await deleteCWComponent(component.id)

      if (detailComp?.id === component.id) {
        setDetailComp(null)
      }

      await loadData()
    } catch (error) {
      console.error(
        '删除课件组件失败',
        error,
      )
      window.alert('删除失败')
    }
  }

  const handleSeed = async (force: boolean) => {
    if (!canRunSeed) {
      return
    }

    const confirmation = force
      ? '将仅清空并重建K12组件，不影响职教、成教和common组件。确定继续？'
      : '填充K12课件组件种子数据，确定继续？'

    if (!window.confirm(confirmation)) {
      return
    }

    setSeeding(true)

    try {
      const result: SeedResult =
        await seedCoursewareData(force)

      const messages = [
        `组件：${result.components_created}`,
        `模板：${result.templates_created}`,
      ]

      if (result.templates_skipped) {
        messages.push(
          `模板说明：${result.templates_skipped}`,
        )
      }

      if (result.errors?.length) {
        messages.push(
          `错误：${result.errors.join('\n')}`,
        )
      }

      window.alert(
        `完成！\n${messages.join('\n')}`,
      )

      await loadData()
    } catch (error) {
      console.error(
        '填充课件种子失败',
        error,
      )
      window.alert('填充失败')
    } finally {
      setSeeding(false)
    }
  }

  if (!ready) {
    return (
      <div style={{
        padding: '28px',
        borderRadius: '12px',
        border: '1px solid #FECACA',
        background: '#FEF2F2',
        color: '#991B1B',
        lineHeight: 1.7,
      }}>
        <div style={{
          fontSize: '16px',
          fontWeight: 700,
          marginBottom: '6px',
        }}>
          教育域不可用
        </div>

        <div style={{ fontSize: '13px' }}>
          {educationError}
        </div>
      </div>
    )
  }

  return (
    <div>
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: '18px',
        flexWrap: 'wrap',
        gap: '12px',
      }}>
        <div>
          <div style={{
            fontSize: '13px',
            color: C.textSecondary,
            marginBottom: '4px',
          }}>
            普通教学身份只能查看当前教育域及跨域通用组件。
          </div>

          {!isMixed && (
            <div style={{
              display: 'inline-flex',
              padding: '3px 10px',
              borderRadius: '12px',
              background: '#F3F4F6',
              color: C.textSecondary,
              fontSize: '11px',
              fontWeight: 600,
            }}>
              当前教育域：
              {EDUCATION_DOMAIN_LABELS[domain]}
            </div>
          )}
        </div>

        {canRunSeed && (
          <div style={{
            display: 'flex',
            gap: '8px',
          }}>
            <button
              onClick={() => handleSeed(false)}
              disabled={seeding}
              style={{
                padding: '7px 16px',
                borderRadius: '8px',
                border:
                  `1px solid ${C.primary}`,
                background: 'transparent',
                color: C.primary,
                fontSize: '13px',
                fontWeight: 600,
                cursor: seeding
                  ? 'wait'
                  : 'pointer',
              }}
            >
              {seeding
                ? '填充中...'
                : '🌱 填充K12种子'}
            </button>

            <button
              onClick={() => handleSeed(true)}
              disabled={seeding}
              style={{
                padding: '7px 16px',
                borderRadius: '8px',
                border:
                  `1px solid ${C.danger}`,
                background: 'transparent',
                color: C.danger,
                fontSize: '13px',
                fontWeight: 600,
                cursor: seeding
                  ? 'wait'
                  : 'pointer',
              }}
            >
              🔄 重建K12
            </button>
          </div>
        )}
      </div>

      <div style={{
        display: 'flex',
        gap: '8px',
        alignItems: 'center',
        marginBottom: '18px',
        flexWrap: 'wrap',
      }}>
        {CW_COMPONENT_TYPE_FILTERS.map(
          filter => (
            <button
              key={filter.value}
              onClick={() =>
                setTypeFilter(filter.value)
              }
              style={{
                padding: '6px 14px',
                borderRadius: '20px',
                border:
                  `1px solid ${
                    typeFilter === filter.value
                      ? C.primary
                      : C.border
                  }`,
                background:
                  typeFilter === filter.value
                    ? 'rgba(245,158,11,0.08)'
                    : 'transparent',
                color:
                  typeFilter === filter.value
                    ? C.primary
                    : C.textSecondary,
                fontSize: '13px',
                fontWeight:
                  typeFilter === filter.value
                    ? 600
                    : 400,
                cursor: 'pointer',
              }}
            >
              {filter.label}
            </button>
          ),
        )}

        {isMixed && (
          <select
            value={domainFilter}
            onChange={event =>
              setDomainFilter(
                event.target.value as
                  ResourceEducationDomain | '',
              )
            }
            style={{
              marginLeft: '4px',
              padding: '6px 12px',
              borderRadius: '8px',
              border:
                `1px solid ${C.border}`,
              background: '#fff',
              color: C.textSecondary,
              fontSize: '13px',
            }}
          >
            <option value="">
              全部资源域
            </option>

            {CW_COMPONENT_DOMAIN_OPTIONS.map(
              option => (
                <option
                  key={option.value}
                  value={option.value}
                >
                  {option.label}
                </option>
              ),
            )}
          </select>
        )}
      </div>

      <div style={{
        fontSize: '13px',
        color: C.textMuted,
        marginBottom: '16px',
      }}>
        共 {total} 个组件
      </div>

      {loadError && (
        <div style={{
          marginBottom: '16px',
          padding: '10px 14px',
          borderRadius: '8px',
          border: '1px solid #FECACA',
          background: '#FEF2F2',
          color: '#991B1B',
          fontSize: '13px',
        }}>
          {loadError}
        </div>
      )}

      <CWComponentGrid
        items={items}
        loading={loading}
        canRunSeed={canRunSeed}
        isMixed={isMixed}
        canMutate={canMutateComponent}
        onOpen={openDetail}
        onDelete={handleDelete}
      />

      <CWComponentDetailModal
        component={detailComp}
        loading={detailLoading}
        isMixed={isMixed}
        onClose={() => setDetailComp(null)}
      />
    </div>
  )
}

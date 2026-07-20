/**
 * SubjectCatalogTable.tsx — 后台课程定义与目录列表
 *
 * 职责：
 *   1. 展示统一课程定义；
 *   2. 展示课程所属教育域及适用学校；
 *   3. 提供课程定义启停、编辑和删除入口；
 *   4. 对没有任何目录归属的历史课程显示明确风险提示。
 *
 * 本组件只负责展示与触发回调，不直接请求接口，
 * 业务状态和保存逻辑继续由SubjectsPanel统一管理。
 */

import type {
  SubjectAdminItem,
} from '@/api/subjects'
import { C } from './adminConstants'

/* ==================== 展示辅助 ==================== */

const EDUCATION_DOMAIN_LABELS = {
  k12: '基础教育',
  vocational: '职业教育',
  adult: '成人教育',
} as const

function getCatalogEntryLabel(
  row: SubjectAdminItem['catalog_entries'][number],
): string {
  const domainLabel =
    EDUCATION_DOMAIN_LABELS[
      row.education_domain
    ]

  if (row.organization_id) {
    return [
      domainLabel,
      row.organization_name,
    ].join(' · ')
  }

  return [
    domainLabel,
    '域内公共',
  ].join(' · ')
}

/* ==================== 组件参数 ==================== */

interface SubjectCatalogTableProps {
  list: SubjectAdminItem[]
  loading: boolean

  onToggleActive:
    (row: SubjectAdminItem) => void

  onEdit:
    (row: SubjectAdminItem) => void

  onDelete:
    (row: SubjectAdminItem) => void
}

/* ==================== 主组件 ==================== */

export function SubjectCatalogTable({
  list,
  loading,
  onToggleActive,
  onEdit,
  onDelete,
}: SubjectCatalogTableProps) {
  return (
    <div
      style={{
        overflow: 'hidden',
        borderRadius: '14px',
        border:
          `1px solid ${C.border}`,
        background: C.white,
      }}
    >
      <div
        style={{
          display: 'grid',
          gridTemplateColumns:
            '70px 1.2fr 0.8fr 0.8fr 2.2fr 1.2fr 160px',
          gap: '10px',
          padding: '12px 20px',
          borderBottom:
            `1px solid ${C.border}`,
          background: C.bg,
          color: C.textSec,
          fontSize: '12px',
          fontWeight: 600,
        }}
      >
        <span>排序</span>
        <span>课程名称</span>
        <span>编码</span>
        <span>状态</span>
        <span>教育域与适用学校</span>
        <span>备注</span>
        <span>操作</span>
      </div>

      {loading ? (
        <div
          style={{
            padding: '40px',
            color: C.textMuted,
            textAlign: 'center',
          }}
        >
          加载中...
        </div>
      ) : list.length === 0 ? (
        <div
          style={{
            padding: '40px',
            color: C.textMuted,
            textAlign: 'center',
          }}
        >
          暂无课程，点击右上角“新增课程”添加
        </div>
      ) : (
        list.map(
          (
            row,
            index,
          ) => {
            const catalogEntries =
              row.catalog_entries || []

            return (
              <div
                key={row.id}
                style={{
                  display: 'grid',
                  gridTemplateColumns:
                    '70px 1.2fr 0.8fr 0.8fr 2.2fr 1.2fr 160px',
                  alignItems: 'center',
                  gap: '10px',
                  padding: '13px 20px',
                  borderBottom:
                    index < list.length - 1
                      ? `1px solid ${C.border}`
                      : 'none',
                  fontSize: '13px',
                }}
              >
                <span
                  style={{
                    color: C.textMuted,
                    fontFamily: 'monospace',
                  }}
                >
                  {row.sort_order}
                </span>

                <span
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                  }}
                >
                  <span
                    style={{
                      color: C.text,
                      fontWeight: 600,
                    }}
                  >
                    {row.name}
                  </span>

                  {row.is_system && (
                    <span
                      style={{
                        padding: '1px 7px',
                        borderRadius: '8px',
                        background:
                          C.primaryLight,
                        color: C.primary,
                        fontSize: '10px',
                        fontWeight: 600,
                      }}
                    >
                      内置
                    </span>
                  )}
                </span>

                <span>
                  {row.code ? (
                    <span
                      style={{
                        padding: '1px 8px',
                        borderRadius: '6px',
                        border:
                          `1px solid ${C.border}`,
                        background: C.bg,
                        color: C.textSec,
                        fontFamily: 'monospace',
                        fontSize: '11px',
                      }}
                    >
                      {row.code}
                    </span>
                  ) : (
                    <span
                      style={{
                        color: C.textMuted,
                      }}
                    >
                      —
                    </span>
                  )}
                </span>

                <span>
                  <button
                    type="button"
                    onClick={() => {
                      onToggleActive(row)
                    }}
                    title={
                      row.is_active
                        ? '点击停用'
                        : '点击启用'
                    }
                    style={{
                      padding: '3px 12px',
                      borderRadius: '20px',
                      border: 'none',
                      background:
                        row.is_active
                          ? C.successLight
                          : C.bg,
                      color:
                        row.is_active
                          ? C.success
                          : C.textMuted,
                      fontSize: '11px',
                      fontWeight: 600,
                      cursor: 'pointer',
                    }}
                  >
                    {row.is_active
                      ? '● 启用'
                      : '○ 停用'}
                  </button>
                </span>

                <span
                  style={{
                    display: 'flex',
                    flexWrap: 'wrap',
                    gap: '5px',
                  }}
                >
                  {catalogEntries.length === 0 ? (
                    <span
                      style={{
                        color: C.danger,
                        fontSize: '11px',
                      }}
                    >
                      未配置目录
                    </span>
                  ) : (
                    catalogEntries.map(
                      catalog => (
                        <span
                          key={catalog.id}
                          title={
                            catalog.display_name
                          }
                          style={{
                            padding: '2px 7px',
                            borderRadius: '8px',
                            border:
                              `1px solid ${C.border}`,
                            background:
                              catalog.is_active
                                ? C.primaryLight
                                : C.bg,
                            color:
                              catalog.is_active
                                ? C.primary
                                : C.textMuted,
                            fontSize: '10px',
                          }}
                        >
                          {getCatalogEntryLabel(
                            catalog,
                          )}

                          {!catalog.is_active &&
                            '（停用）'}
                        </span>
                      ),
                    )
                  )}
                </span>

                <span
                  title={row.note || ''}
                  style={{
                    overflow: 'hidden',
                    color: row.note
                      ? C.textSec
                      : C.textMuted,
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {row.note || '—'}
                </span>

                <span
                  style={{
                    display: 'flex',
                    gap: '6px',
                  }}
                >
                  <button
                    type="button"
                    onClick={() => {
                      onEdit(row)
                    }}
                    style={{
                      padding: '4px 12px',
                      borderRadius: '6px',
                      border:
                        `1px solid ${C.border}`,
                      background: C.bg,
                      color: C.primary,
                      fontSize: '12px',
                      fontWeight: 500,
                      cursor: 'pointer',
                    }}
                  >
                    编辑
                  </button>

                  {row.is_system ? (
                    <button
                      type="button"
                      disabled
                      title="内置课程不可删除，如需隐藏请停用"
                      style={{
                        padding: '4px 12px',
                        borderRadius: '6px',
                        border:
                          `1px solid ${C.border}`,
                        background: C.bg,
                        color: C.textMuted,
                        fontSize: '12px',
                        fontWeight: 500,
                        cursor: 'not-allowed',
                      }}
                    >
                      删除
                    </button>
                  ) : (
                    <button
                      type="button"
                      onClick={() => {
                        onDelete(row)
                      }}
                      style={{
                        padding: '4px 12px',
                        borderRadius: '6px',
                        border:
                          '1px solid #FEE2E2',
                        background: '#FEF2F2',
                        color: C.danger,
                        fontSize: '12px',
                        fontWeight: 500,
                        cursor: 'pointer',
                      }}
                    >
                      删除
                    </button>
                  )}
                </span>
              </div>
            )
          },
        )
      )}
    </div>
  )
}

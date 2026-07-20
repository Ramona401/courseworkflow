/**
 * SubjectCatalogEditor.tsx — 课程目录归属编辑器
 *
 * 职责：
 *   1. 编辑课程所属的具体教学教育域；
 *   2. 选择“教育域公共”或“指定学校”适用范围；
 *   3. 选择与教育域一致的启用学校；
 *   4. 编辑域内展示名、排序和启用状态；
 *   5. 支持一门课程配置多个教育域或多个学校。
 *
 * 数据安全说明：
 *   - mixed只用于跨域管理页面，不能成为课程目录教育域；
 *   - 指定学校列表严格按当前目录教育域过滤；
 *   - 教育域或范围发生变化时清空失效的学校ID；
 *   - 后端仍会再次校验学校类型、状态和教育域，前端不能替代后端防线。
 */

import type { CSSProperties } from 'react'
import type {
  SubjectCatalogEntry,
  TeachingEducationDomain,
} from '@/api/subjects'
import type {
  OrganizationEducationDomainItem,
} from '@/api/organization-education-domains'
import { C } from './adminConstants'

/* ==================== 表单数据结构 ==================== */

export type SubjectCatalogScope =
  | 'public'
  | 'organization'

/**
 * 弹窗内部使用的目录草稿。
 *
 * key只用于React列表稳定渲染，不提交给后端。
 * education_domain允许空字符串，便于新增时强制管理员明确选择。
 */
export interface SubjectCatalogDraft {
  key: string
  education_domain:
    | TeachingEducationDomain
    | ''
  scope: SubjectCatalogScope
  organization_id: string
  display_name: string
  sort_order: number
  is_active: boolean
}

let draftSequence = 0

function createDraftKey(): string {
  draftSequence += 1

  return [
    'subject-catalog',
    Date.now(),
    draftSequence,
  ].join('-')
}

/**
 * 创建一条空目录配置。
 */
export function createEmptyCatalogDraft(
  sortOrder = 100,
): SubjectCatalogDraft {
  return {
    key: createDraftKey(),
    education_domain: '',
    scope: 'organization',
    organization_id: '',
    display_name: '',
    sort_order: sortOrder,
    is_active: true,
  }
}

/**
 * 将后端目录记录转换为编辑草稿。
 */
export function catalogEntryToDraft(
  entry: SubjectCatalogEntry,
): SubjectCatalogDraft {
  return {
    key: createDraftKey(),
    education_domain:
      entry.education_domain,
    scope: entry.organization_id
      ? 'organization'
      : 'public',
    organization_id:
      entry.organization_id || '',
    display_name:
      entry.display_name || '',
    sort_order:
      entry.sort_order,
    is_active:
      entry.is_active,
  }
}

/* ==================== 展示配置 ==================== */

const EDUCATION_DOMAIN_OPTIONS: Array<{
  value: TeachingEducationDomain
  label: string
}> = [
  {
    value: 'k12',
    label: '基础教育',
  },
  {
    value: 'vocational',
    label: '职业教育',
  },
  {
    value: 'adult',
    label: '成人教育',
  },
]

const inputStyle: CSSProperties = {
  width: '100%',
  padding: '8px 10px',
  borderRadius: '7px',
  border: `1px solid ${C.border}`,
  background: C.white,
  color: C.text,
  fontSize: '12px',
  outline: 'none',
  boxSizing: 'border-box',
}

const labelStyle: CSSProperties = {
  display: 'block',
  marginBottom: '4px',
  color: C.textSec,
  fontSize: '11px',
  fontWeight: 600,
}

/* ==================== 组件 ==================== */

interface SubjectCatalogEditorProps {
  entries: SubjectCatalogDraft[]
  organizations:
    OrganizationEducationDomainItem[]
  subjectName: string
  subjectSortOrder: number
  disabled?: boolean
  onChange:
    (entries: SubjectCatalogDraft[]) => void
}

export function SubjectCatalogEditor({
  entries,
  organizations,
  subjectName,
  subjectSortOrder,
  disabled = false,
  onChange,
}: SubjectCatalogEditorProps) {
  /**
   * 只允许选择启用、school类型且教育域匹配的组织。
   */
  const getSchoolsForDomain = (
    domain:
      | TeachingEducationDomain
      | '',
  ) => organizations.filter(
    organization =>
      organization.type === 'school' &&
      organization.status === 'active' &&
      organization.education_domain ===
        domain,
  )

  const updateEntry = (
    index: number,
    patch: Partial<SubjectCatalogDraft>,
  ) => {
    onChange(
      entries.map(
        (entry, currentIndex) =>
          currentIndex === index
            ? {
                ...entry,
                ...patch,
              }
            : entry,
      ),
    )
  }

  const removeEntry = (
    index: number,
  ) => {
    onChange(
      entries.filter(
        (_, currentIndex) =>
          currentIndex !== index,
      ),
    )
  }

  const addEntry = () => {
    onChange([
      ...entries,
      createEmptyCatalogDraft(
        subjectSortOrder,
      ),
    ])
  }

  return (
    <div>
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent: 'space-between',
          gap: '16px',
          marginBottom: '10px',
        }}
      >
        <div>
          <div
            style={{
              color: C.text,
              fontSize: '13px',
              fontWeight: 700,
            }}
          >
            教育域与适用学校
            <span
              style={{
                color: C.danger,
                marginLeft: '3px',
              }}
            >
              *
            </span>
          </div>

          <div
            style={{
              marginTop: '3px',
              color: C.textMuted,
              fontSize: '11px',
              lineHeight: 1.5,
            }}
          >
            至少配置一项；教师下拉只显示其教育域公共课程和本校课程。
          </div>
        </div>

        <button
          type="button"
          disabled={disabled}
          onClick={addEntry}
          style={{
            padding: '6px 12px',
            borderRadius: '7px',
            border:
              `1px solid ${C.primary}`,
            background: C.primaryLight,
            color: C.primary,
            fontSize: '11px',
            fontWeight: 600,
            cursor: disabled
              ? 'not-allowed'
              : 'pointer',
            whiteSpace: 'nowrap',
          }}
        >
          + 添加目录
        </button>
      </div>

      {entries.length === 0 ? (
        <div
          style={{
            padding: '18px',
            borderRadius: '9px',
            border:
              `1px dashed ${C.border}`,
            background: C.bg,
            color: C.textMuted,
            fontSize: '12px',
            textAlign: 'center',
          }}
        >
          尚未配置课程目录。新增课程必须至少添加一项。
        </div>
      ) : (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: '10px',
          }}
        >
          {entries.map(
            (entry, index) => {
              const schools =
                getSchoolsForDomain(
                  entry.education_domain,
                )

              return (
                <div
                  key={entry.key}
                  style={{
                    padding: '12px',
                    borderRadius: '10px',
                    border:
                      `1px solid ${C.border}`,
                    background: C.bg,
                  }}
                >
                  <div
                    style={{
                      display: 'grid',
                      gridTemplateColumns:
                        '1fr 1fr 1.5fr',
                      gap: '8px',
                    }}
                  >
                    <div>
                      <label style={labelStyle}>
                        教育域
                      </label>
                      <select
                        value={
                          entry.education_domain
                        }
                        disabled={disabled}
                        onChange={event => {
                          updateEntry(index, {
                            education_domain:
                              event.target.value as
                                | TeachingEducationDomain
                                | '',
                            organization_id: '',
                          })
                        }}
                        style={inputStyle}
                      >
                        <option value="">
                          请选择教育域
                        </option>
                        {EDUCATION_DOMAIN_OPTIONS.map(
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
                    </div>

                    <div>
                      <label style={labelStyle}>
                        适用范围
                      </label>
                      <select
                        value={entry.scope}
                        disabled={disabled}
                        onChange={event => {
                          const scope =
                            event.target.value as
                              SubjectCatalogScope

                          updateEntry(index, {
                            scope,
                            organization_id:
                              scope === 'public'
                                ? ''
                                : entry
                                    .organization_id,
                          })
                        }}
                        style={inputStyle}
                      >
                        <option value="organization">
                          指定学校
                        </option>
                        <option value="public">
                          教育域公共
                        </option>
                      </select>
                    </div>

                    <div>
                      <label style={labelStyle}>
                        适用学校
                      </label>
                      {entry.scope === 'public' ? (
                        <div
                          style={{
                            ...inputStyle,
                            color: C.textSec,
                          }}
                        >
                          该教育域全部学校
                        </div>
                      ) : (
                        <select
                          value={
                            entry.organization_id
                          }
                          disabled={
                            disabled ||
                            !entry.education_domain
                          }
                          onChange={event => {
                            updateEntry(index, {
                              organization_id:
                                event.target.value,
                            })
                          }}
                          style={inputStyle}
                        >
                          <option value="">
                            {entry.education_domain
                              ? '请选择学校'
                              : '请先选择教育域'}
                          </option>

                          {schools.map(
                            school => (
                              <option
                                key={school.id}
                                value={school.id}
                              >
                                {school.name}
                              </option>
                            ),
                          )}
                        </select>
                      )}
                    </div>
                  </div>

                  {entry.scope ===
                    'organization' &&
                    entry.education_domain &&
                    schools.length === 0 && (
                      <div
                        style={{
                          marginTop: '6px',
                          color: C.danger,
                          fontSize: '11px',
                        }}
                      >
                        该教育域暂无启用学校，不能建立学校专属目录。
                      </div>
                    )}

                  <div
                    style={{
                      display: 'grid',
                      gridTemplateColumns:
                        '1.6fr 120px auto auto',
                      alignItems: 'end',
                      gap: '8px',
                      marginTop: '10px',
                    }}
                  >
                    <div>
                      <label style={labelStyle}>
                        域内展示名
                      </label>
                      <input
                        value={entry.display_name}
                        disabled={disabled}
                        maxLength={100}
                        placeholder={
                          subjectName.trim() ||
                          '留空则使用课程名'
                        }
                        onChange={event => {
                          updateEntry(index, {
                            display_name:
                              event.target.value,
                          })
                        }}
                        style={inputStyle}
                      />
                    </div>

                    <div>
                      <label style={labelStyle}>
                        域内排序
                      </label>
                      <input
                        type="number"
                        min={0}
                        max={9999}
                        value={entry.sort_order}
                        disabled={disabled}
                        onChange={event => {
                          updateEntry(index, {
                            sort_order:
                              Number(
                                event.target.value,
                              ) || 0,
                          })
                        }}
                        style={inputStyle}
                      />
                    </div>

                    <button
                      type="button"
                      disabled={disabled}
                      onClick={() => {
                        updateEntry(index, {
                          is_active:
                            !entry.is_active,
                        })
                      }}
                      style={{
                        padding: '8px 11px',
                        borderRadius: '7px',
                        border: 'none',
                        background:
                          entry.is_active
                            ? C.successLight
                            : C.white,
                        color:
                          entry.is_active
                            ? C.success
                            : C.textMuted,
                        fontSize: '11px',
                        fontWeight: 600,
                        cursor: disabled
                          ? 'not-allowed'
                          : 'pointer',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {entry.is_active
                        ? '● 已启用'
                        : '○ 已停用'}
                    </button>

                    <button
                      type="button"
                      disabled={disabled}
                      onClick={() => {
                        removeEntry(index)
                      }}
                      style={{
                        padding: '8px 11px',
                        borderRadius: '7px',
                        border:
                          '1px solid #FECACA',
                        background: '#FEF2F2',
                        color: C.danger,
                        fontSize: '11px',
                        cursor: disabled
                          ? 'not-allowed'
                          : 'pointer',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      移除
                    </button>
                  </div>
                </div>
              )
            },
          )}
        </div>
      )}
    </div>
  )
}

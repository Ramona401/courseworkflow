/**
 * SubjectCatalogFormModal.tsx — 课程定义与目录编辑弹窗
 *
 * 职责：
 *   1. 展示课程名称、编码、基础排序、备注和启停状态；
 *   2. 挂载SubjectCatalogEditor管理教育域与适用学校；
 *   3. 统一弹窗布局、字段更新和保存/取消按钮状态；
 *   4. 不直接调用接口，不承担业务校验和数据持久化。
 *
 * 业务边界：
 *   - SubjectsPanel负责加载数据、校验目录及调用后端事务接口；
 *   - SubjectCatalogEditor负责单条目录的教育域和学校编辑；
 *   - 本组件只负责表单展示，保持各模块职责清晰。
 */

import type {
  CSSProperties,
} from 'react'

import type {
  OrganizationEducationDomainItem,
} from '@/api/organization-education-domains'

import { C } from './adminConstants'
import {
  SubjectCatalogEditor,
} from './SubjectCatalogEditor'
import type {
  SubjectCatalogDraft,
} from './SubjectCatalogEditor'

/* ==================== 对外表单模型 ==================== */

/**
 * 课程管理弹窗的完整受控表单值。
 *
 * 由SubjectsPanel持有真实状态，本组件通过onChange提交部分字段更新。
 */
export interface SubjectCatalogFormValue {
  name: string
  code: string
  sort_order: number
  note: string
  is_active: boolean
  catalog_entries: SubjectCatalogDraft[]
}

/* ==================== 组件参数 ==================== */

interface SubjectCatalogFormModalProps {
  open: boolean
  editing: boolean

  value: SubjectCatalogFormValue

  organizations:
    OrganizationEducationDomainItem[]

  loading: boolean
  saving: boolean

  onChange: (
    patch: Partial<SubjectCatalogFormValue>,
  ) => void

  onCancel: () => void
  onSave: () => void
}

/* ==================== 通用样式 ==================== */

const inputStyle: CSSProperties = {
  width: '100%',
  padding: '9px 12px',
  borderRadius: '8px',
  border: `1px solid ${C.border}`,
  background: C.white,
  color: C.text,
  fontSize: '13px',
  outline: 'none',
  boxSizing: 'border-box',
}

const labelStyle: CSSProperties = {
  display: 'block',
  marginBottom: '5px',
  color: C.textSec,
  fontSize: '12px',
  fontWeight: 600,
}

/* ==================== 主组件 ==================== */

export function SubjectCatalogFormModal({
  open,
  editing,
  value,
  organizations,
  loading,
  saving,
  onChange,
  onCancel,
  onSave,
}: SubjectCatalogFormModalProps) {
  if (!open) {
    return null
  }

  const disabled =
    loading || saving

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 11000,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '24px',
        background: 'rgba(0,0,0,0.45)',
        backdropFilter: 'blur(4px)',
      }}
    >
      <div
        style={{
          width: '920px',
          maxWidth: '96vw',
          maxHeight: '92vh',
          overflowY: 'auto',
          padding: '26px',
          borderRadius: '16px',
          background: C.white,
          boxShadow:
            '0 20px 60px rgba(0,0,0,0.2)',
        }}
      >
        <div
          style={{
            marginBottom: '18px',
            color: C.text,
            fontSize: '17px',
            fontWeight: 700,
          }}
        >
          {editing
            ? '编辑课程及目录'
            : '新增课程及目录'}
        </div>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns:
              '1.5fr 1fr 140px',
            gap: '12px',
          }}
        >
          <div>
            <label style={labelStyle}>
              课程名称
              <span
                style={{
                  color: C.danger,
                }}
              >
                *
              </span>
            </label>

            <input
              value={value.name}
              maxLength={50}
              disabled={saving}
              placeholder="如：药学"
              onChange={event => {
                onChange({
                  name: event.target.value,
                })
              }}
              style={inputStyle}
            />
          </div>

          <div>
            <label style={labelStyle}>
              索引编码
            </label>

            <input
              value={value.code}
              maxLength={20}
              disabled={saving}
              placeholder="可留空"
              onChange={event => {
                onChange({
                  code: event.target.value,
                })
              }}
              style={inputStyle}
            />
          </div>

          <div>
            <label style={labelStyle}>
              基础排序
            </label>

            <input
              type="number"
              min={0}
              max={9999}
              value={value.sort_order}
              disabled={saving}
              onChange={event => {
                onChange({
                  sort_order:
                    Number(
                      event.target.value,
                    ) || 0,
                })
              }}
              style={inputStyle}
            />
          </div>
        </div>

        <div
          style={{
            marginTop: '14px',
          }}
        >
          <label style={labelStyle}>
            备注
          </label>

          <input
            value={value.note}
            maxLength={200}
            disabled={saving}
            placeholder="仅用于管理说明，不决定教育域或学校归属"
            onChange={event => {
              onChange({
                note: event.target.value,
              })
            }}
            style={inputStyle}
          />
        </div>

        {editing && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '10px',
              marginTop: '14px',
            }}
          >
            <span
              style={{
                ...labelStyle,
                marginBottom: 0,
              }}
            >
              课程定义状态
            </span>

            <button
              type="button"
              disabled={saving}
              onClick={() => {
                onChange({
                  is_active:
                    !value.is_active,
                })
              }}
              style={{
                padding: '5px 14px',
                borderRadius: '20px',
                border: 'none',
                background:
                  value.is_active
                    ? C.successLight
                    : C.bg,
                color:
                  value.is_active
                    ? C.success
                    : C.textMuted,
                fontSize: '12px',
                fontWeight: 600,
                cursor: saving
                  ? 'not-allowed'
                  : 'pointer',
              }}
            >
              {value.is_active
                ? '● 启用中'
                : '○ 已停用'}
            </button>
          </div>
        )}

        <div
          style={{
            height: '1px',
            margin: '20px 0',
            background: C.border,
          }}
        />

        <SubjectCatalogEditor
          entries={
            value.catalog_entries
          }
          organizations={
            organizations
          }
          subjectName={
            value.name
          }
          subjectSortOrder={
            value.sort_order
          }
          disabled={disabled}
          onChange={entries => {
            onChange({
              catalog_entries: entries,
            })
          }}
        />

        <div
          style={{
            display: 'flex',
            gap: '10px',
            marginTop: '22px',
          }}
        >
          <button
            type="button"
            disabled={saving}
            onClick={onCancel}
            style={{
              flex: 1,
              padding: '10px',
              borderRadius: '10px',
              border:
                `1px solid ${C.border}`,
              background: C.bg,
              color: C.textSec,
              fontSize: '14px',
              cursor: saving
                ? 'not-allowed'
                : 'pointer',
            }}
          >
            取消
          </button>

          <button
            type="button"
            disabled={disabled}
            onClick={onSave}
            style={{
              flex: 1,
              padding: '10px',
              borderRadius: '10px',
              border: 'none',
              background:
                disabled
                  ? '#9CA3AF'
                  : `linear-gradient(135deg,${C.primary},#7C3AED)`,
              color: '#fff',
              fontSize: '14px',
              fontWeight: 600,
              cursor: disabled
                ? 'not-allowed'
                : 'pointer',
            }}
          >
            {saving
              ? '保存中...'
              : '保存课程及目录'}
          </button>
        </div>
      </div>
    </div>
  )
}

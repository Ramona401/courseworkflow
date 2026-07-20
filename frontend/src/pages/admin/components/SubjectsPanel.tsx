/**
 * SubjectsPanel.tsx — 后台课程定义与课程目录管理主面板
 *
 * 职责：
 *   1. 加载课程定义、目录归属和学校教育域；
 *   2. 管理新增、编辑、启停和删除流程；
 *   3. 校验目录教育域、适用学校和重复配置；
 *   4. 调用后端事务接口同步保存subjects和subject_catalog_entries。
 *
 * 模块拆分：
 *   - SubjectCatalogFormModal：课程定义与目录编辑弹窗；
 *   - SubjectCatalogEditor：单条目录编辑；
 *   - SubjectCatalogTable：课程与目录列表；
 *   - 本文件只保留数据状态、校验和接口调用。
 */

import {
  useCallback,
  useEffect,
  useState,
} from 'react'

import {
  createSubject,
  deleteSubject,
  getAllSubjects,
  updateSubject,
} from '@/api/subjects'
import type {
  SubjectAdminItem,
  SubjectCatalogEntryRequest,
} from '@/api/subjects'

import {
  getOrganizationEducationDomains,
} from '@/api/organization-education-domains'
import type {
  OrganizationEducationDomainItem,
} from '@/api/organization-education-domains'

import {
  refreshSubjects,
} from '@/hooks/useSubjects'

import { C } from './adminConstants'
import { Toast } from './adminShared'
import { ConfirmDialog } from './ConfirmDialog'

import {
  catalogEntryToDraft,
  createEmptyCatalogDraft,
} from './SubjectCatalogEditor'
import type {
  SubjectCatalogDraft,
} from './SubjectCatalogEditor'

import {
  SubjectCatalogFormModal,
} from './SubjectCatalogFormModal'
import type {
  SubjectCatalogFormValue,
} from './SubjectCatalogFormModal'

import {
  SubjectCatalogTable,
} from './SubjectCatalogTable'

/* ==================== 表单模型 ==================== */

type SubjectForm =
  SubjectCatalogFormValue

/**
 * 创建新增课程表单。
 *
 * 默认生成一条空目录，提醒管理员必须明确选择
 * 教育域和适用学校。
 */
function createEmptyForm(
  sortOrder: number,
): SubjectForm {
  return {
    name: '',
    code: '',
    sort_order: sortOrder,
    note: '',
    is_active: true,
    catalog_entries: [
      createEmptyCatalogDraft(
        sortOrder,
      ),
    ],
  }
}

/* ==================== 目录请求转换 ==================== */

/**
 * 将目录编辑草稿转换成后端写入请求。
 *
 * 前端校验：
 *   - 至少存在一条目录；
 *   - 每条目录必须选择具体教育域；
 *   - 学校专属目录必须选择学校；
 *   - 同一教育域公共目录不能重复；
 *   - 同一学校目录不能重复。
 *
 * 后端仍会执行组织类型、状态和教育域一致性终校验。
 */
function buildCatalogRequests(
  entries: SubjectCatalogDraft[],
): {
  error: string
  requests: SubjectCatalogEntryRequest[]
} {
  if (entries.length === 0) {
    return {
      error:
        '请至少配置一个教育域或适用学校',
      requests: [],
    }
  }

  const seen =
    new Set<string>()

  const requests:
    SubjectCatalogEntryRequest[] = []

  for (
    let index = 0;
    index < entries.length;
    index += 1
  ) {
    const entry = entries[index]
    const position = index + 1

    if (!entry.education_domain) {
      return {
        error:
          `第${position}项课程目录尚未选择教育域`,
        requests: [],
      }
    }

    if (
      entry.scope ===
        'organization' &&
      !entry.organization_id.trim()
    ) {
      return {
        error:
          `第${position}项课程目录尚未选择适用学校`,
        requests: [],
      }
    }

    const organizationId =
      entry.scope === 'public'
        ? null
        : entry.organization_id.trim()

    const duplicateKey = [
      entry.education_domain,
      organizationId || 'public',
    ].join('::')

    if (seen.has(duplicateKey)) {
      return {
        error:
          `第${position}项与已有课程目录重复`,
        requests: [],
      }
    }

    seen.add(duplicateKey)

    requests.push({
      education_domain:
        entry.education_domain,
      organization_id:
        organizationId,
      display_name:
        entry.display_name.trim(),
      sort_order:
        entry.sort_order,
      is_active:
        entry.is_active,
    })
  }

  return {
    error: '',
    requests,
  }
}

/* ==================== 主组件 ==================== */

export function SubjectsPanel() {
  const [
    list,
    setList,
  ] = useState<
    SubjectAdminItem[]
  >([])

  const [
    organizations,
    setOrganizations,
  ] = useState<
    OrganizationEducationDomainItem[]
  >([])

  const [
    loading,
    setLoading,
  ] = useState(true)

  const [
    saving,
    setSaving,
  ] = useState(false)

  const [
    modalOpen,
    setModalOpen,
  ] = useState(false)

  const [
    editing,
    setEditing,
  ] = useState<
    SubjectAdminItem | null
  >(null)

  const [
    form,
    setForm,
  ] = useState<SubjectForm>(
    createEmptyForm(100),
  )

  const [
    toast,
    setToast,
  ] = useState<{
    message: string
    type:
      | 'success'
      | 'error'
  } | null>(null)

  const [
    confirmDel,
    setConfirmDel,
  ] = useState<{
    open: boolean
    id: string
    name: string
  }>({
    open: false,
    id: '',
    name: '',
  })

  const showToast =
    useCallback(
      (
        message: string,
        type:
          | 'success'
          | 'error',
      ) => {
        setToast({
          message,
          type,
        })
      },
      [],
    )

  /* ==================== 数据加载 ==================== */

  const load =
    useCallback(async () => {
      try {
        setLoading(true)

        const [
          subjectItems,
          organizationResult,
        ] = await Promise.all([
          getAllSubjects(),
          getOrganizationEducationDomains(),
        ])

        setList(subjectItems)

        setOrganizations(
          organizationResult
            .organizations || [],
        )
      } catch (
        error: unknown
      ) {
        showToast(
          error instanceof Error
            ? error.message
            : '加载课程管理数据失败',
          'error',
        )
      } finally {
        setLoading(false)
      }
    }, [
      showToast,
    ])

  useEffect(() => {
    load()
  }, [
    load,
  ])

  /* ==================== 打开弹窗 ==================== */

  const openCreate = () => {
    const maxSort =
      list.reduce(
        (
          currentMax,
          row,
        ) =>
          Math.max(
            currentMax,
            row.sort_order,
          ),
        0,
      )

    setEditing(null)

    setForm(
      createEmptyForm(
        maxSort + 10,
      ),
    )

    setModalOpen(true)
  }

  const openEdit = (
    row: SubjectAdminItem,
  ) => {
    setEditing(row)

    setForm({
      name: row.name,
      code: row.code || '',
      sort_order:
        row.sort_order,
      note: row.note || '',
      is_active:
        row.is_active,
      catalog_entries:
        (
          row.catalog_entries ||
          []
        ).map(
          catalogEntryToDraft,
        ),
    })

    setModalOpen(true)
  }

  /* ==================== 保存 ==================== */

  const handleSave =
    async () => {
      const name =
        form.name.trim()

      if (!name) {
        showToast(
          '请输入课程名称',
          'error',
        )
        return
      }

      const catalogResult =
        buildCatalogRequests(
          form.catalog_entries,
        )

      if (
        catalogResult.error
      ) {
        showToast(
          catalogResult.error,
          'error',
        )
        return
      }

      try {
        setSaving(true)

        if (editing) {
          await updateSubject(
            editing.id,
            {
              name,
              code:
                form.code.trim(),
              sort_order:
                form.sort_order,
              note:
                form.note.trim(),
              is_active:
                form.is_active,
              catalog_entries:
                catalogResult
                  .requests,
            },
          )

          showToast(
            '课程及目录配置已保存',
            'success',
          )
        } else {
          await createSubject({
            name,
            code:
              form.code.trim(),
            sort_order:
              form.sort_order,
            note:
              form.note.trim(),
            catalog_entries:
              catalogResult
                .requests,
          })

          showToast(
            '课程及目录配置已新增',
            'success',
          )
        }

        setModalOpen(false)

        await load()
        await refreshSubjects()
      } catch (
        error: unknown
      ) {
        showToast(
          error instanceof Error
            ? error.message
            : '保存失败',
          'error',
        )
      } finally {
        setSaving(false)
      }
    }

  /* ==================== 行内启停 ==================== */

  const handleToggleActive =
    async (
      row: SubjectAdminItem,
    ) => {
      try {
        await updateSubject(
          row.id,
          {
            is_active:
              !row.is_active,
          },
        )

        showToast(
          !row.is_active
            ? '课程已启用'
            : '课程已停用',
          'success',
        )

        await load()
        await refreshSubjects()
      } catch (
        error: unknown
      ) {
        showToast(
          error instanceof Error
            ? error.message
            : '操作失败',
          'error',
        )

        await load()
      }
    }

  /* ==================== 删除 ==================== */

  const doDelete =
    async (
      id: string,
    ) => {
      try {
        await deleteSubject(id)

        showToast(
          '课程及目录配置已删除',
          'success',
        )

        await load()
        await refreshSubjects()
      } catch (
        error: unknown
      ) {
        showToast(
          error instanceof Error
            ? error.message
            : '删除失败',
          'error',
        )
      } finally {
        setConfirmDel({
          open: false,
          id: '',
          name: '',
        })
      }
    }

  /* ==================== 渲染 ==================== */

  return (
    <div>
      {toast && (
        <Toast
          message={
            toast.message
          }
          type={
            toast.type
          }
          onClose={() => {
            setToast(null)
          }}
        />
      )}

      {confirmDel.open && (
        <ConfirmDialog
          title="删除课程"
          message={
            `确认删除课程「${confirmDel.name}」？` +
            '该课程的全部教育域和学校目录配置也会一并删除。' +
            '已有教案和课件的历史课程名称快照不受影响。'
          }
          onConfirm={() => {
            doDelete(
              confirmDel.id,
            )
          }}
          onCancel={() => {
            setConfirmDel({
              open: false,
              id: '',
              name: '',
            })
          }}
        />
      )}

      <SubjectCatalogFormModal
        open={modalOpen}
        editing={
          editing !== null
        }
        value={form}
        organizations={
          organizations
        }
        loading={loading}
        saving={saving}
        onChange={patch => {
          setForm(
            previous => ({
              ...previous,
              ...patch,
            }),
          )
        }}
        onCancel={() => {
          setModalOpen(false)
        }}
        onSave={
          handleSave
        }
      />

      <div
        style={{
          display: 'flex',
          alignItems:
            'flex-start',
          justifyContent:
            'space-between',
          gap: '20px',
          marginBottom: '16px',
        }}
      >
        <div>
          <div
            style={{
              display: 'flex',
              alignItems:
                'center',
              gap: '8px',
              color: C.text,
              fontSize: '16px',
              fontWeight: 700,
            }}
          >
            📚 课程定义与目录

            {list.length > 0 && (
              <span
                style={{
                  color:
                    C.textMuted,
                  fontSize:
                    '12px',
                  fontWeight:
                    400,
                }}
              >
                共 {list.length}
                门课程
              </span>
            )}
          </div>

          <div
            style={{
              maxWidth: '760px',
              marginTop: '4px',
              color: C.textMuted,
              fontSize: '12px',
              lineHeight: 1.6,
            }}
          >
            新增课程时必须配置教育域和适用学校。
            教师下拉只显示当前教育域公共课程及本校专属课程；
            备注字段不参与课程可见性判断。
          </div>
        </div>

        <button
          type="button"
          disabled={loading}
          onClick={openCreate}
          style={{
            padding:
              '8px 18px',
            borderRadius:
              '8px',
            border: 'none',
            background:
              loading
                ? '#9CA3AF'
                : `linear-gradient(135deg,${C.primary},#7C3AED)`,
            color: '#fff',
            fontSize: '13px',
            fontWeight: 600,
            cursor:
              loading
                ? 'not-allowed'
                : 'pointer',
            whiteSpace:
              'nowrap',
          }}
        >
          + 新增课程
        </button>
      </div>

      <SubjectCatalogTable
        list={list}
        loading={loading}
        onToggleActive={
          handleToggleActive
        }
        onEdit={openEdit}
        onDelete={row => {
          setConfirmDel({
            open: true,
            id: row.id,
            name: row.name,
          })
        }}
      />
    </div>
  )
}

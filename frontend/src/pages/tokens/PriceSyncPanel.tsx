/**
 * PriceSyncPanel.tsx — 文本及媒体价格自动同步管理主面板。
 *
 * 本组件只做前端业务编排：
 *   - 读取全局同步配置和全部同步目标；
 *   - 保存配置与单模型同步开关；
 *   - 拉取价格并展示预览；
 *   - 全部或选择性应用价格；
 *   - 查看最近同步历史。
 *
 * 图片、视频和TTS结算逻辑不在本组件范围内。
 */

import { useEffect, useState } from 'react'
import {
  applyAllPriceSyncChanges,
  applySelectedPriceSyncChanges,
  getPriceSyncManagementState,
  getPriceSyncRunDetail,
  getPriceSyncRuns,
  previewPriceSync,
  updatePriceSyncSettings,
  updatePriceSyncTarget,
  type PriceSyncManagementState,
  type PriceSyncPreviewResponse,
  type PriceSyncRun,
  type PriceSyncSettings,
  type PriceSyncTargetConfig,
  type UpdatePriceSyncSettingsRequest,
  type UpdatePriceSyncTargetRequest,
} from '@/api/priceSync'
import { C } from './tokenDashboardParts'
import { PriceSyncSettingsCard } from './PriceSyncSettingsCard'
import { PriceSyncTargetsTable } from './PriceSyncTargetsTable'
import { PriceSyncPreviewPanel } from './PriceSyncPreviewPanel'

interface PriceSyncPanelProps {
  onPricesChanged?: () => void | Promise<void>
}

function priceSyncErrorMessage(error: unknown): string {
  const candidate = error as {
    message?: string
    response?: {
      data?: {
        message?: string
      }
    }
  }

  return (
    candidate?.response?.data?.message ||
    candidate?.message ||
    '价格同步操作失败'
  )
}

export function PriceSyncPanel({
  onPricesChanged,
}: PriceSyncPanelProps) {
  const [management, setManagement] =
    useState<PriceSyncManagementState | null>(null)

  const [runs, setRuns] = useState<PriceSyncRun[]>([])
  const [preview, setPreview] =
    useState<PriceSyncPreviewResponse | null>(null)

  const [selectedItemIDs, setSelectedItemIDs] =
    useState<string[]>([])

  const [loading, setLoading] = useState(true)
  const [busyAction, setBusyAction] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const reloadManagement = async () => {
    const result = await getPriceSyncManagementState()
    setManagement(result)
  }

  const reloadRuns = async () => {
    const result = await getPriceSyncRuns(20)
    setRuns(result || [])
  }

  const reloadAll = async () => {
    setLoading(true)
    setError('')

    try {
      await Promise.all([
        reloadManagement(),
        reloadRuns(),
      ])
    } catch (requestError) {
      setError(
        priceSyncErrorMessage(requestError),
      )
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void reloadAll()
  }, [])

  const handleSaveSettings = async (
    request: UpdatePriceSyncSettingsRequest,
  ): Promise<PriceSyncSettings> => {
    setBusyAction('settings')
    setMessage('')
    setError('')

    try {
      const result =
        await updatePriceSyncSettings(request)

      setManagement(current => {
        if (!current) return current

        return {
          ...current,
          settings: result,
        }
      })

      setMessage('价格同步全局配置已保存。')
      return result
    } catch (requestError) {
      const reason =
        priceSyncErrorMessage(requestError)

      setError(reason)
      throw requestError
    } finally {
      setBusyAction('')
    }
  }

  const handleSaveTarget = async (
    target: PriceSyncTargetConfig,
    request: UpdatePriceSyncTargetRequest,
  ): Promise<PriceSyncTargetConfig> => {
    setBusyAction(`target:${target.id}`)
    setMessage('')
    setError('')

    try {
      const updated = await updatePriceSyncTarget(
        target.target_kind,
        target.id,
        request,
      )

      setManagement(current => {
        if (!current) return current

        const replace = (
          items: PriceSyncTargetConfig[],
        ) => items.map(item => (
          item.id === updated.id
            ? updated
            : item
        ))

        if (updated.target_kind === 'text') {
          return {
            ...current,
            text_targets: replace(
              current.text_targets,
            ),
          }
        }

        return {
          ...current,
          media_targets: replace(
            current.media_targets,
          ),
        }
      })

      setMessage(
        `「${updated.display_name || updated.model_name}」同步配置已保存。`,
      )

      return updated
    } catch (requestError) {
      setError(
        priceSyncErrorMessage(requestError),
      )
      throw requestError
    } finally {
      setBusyAction('')
    }
  }

  const handlePreview = async () => {
    if (!management) return

    setBusyAction('preview')
    setMessage('')
    setError('')
    setSelectedItemIDs([])

    try {
      const result = await previewPriceSync({
        group: management.settings.group,
        include_text: true,
        include_media: true,
        max_change_percent:
          management.settings.max_change_percent,
      })

      setPreview(result)
      setMessage(
        `价格检查完成：发现 ${result.summary.update_count} 条可更新价格。`,
      )

      await reloadRuns()
    } catch (requestError) {
      setError(
        priceSyncErrorMessage(requestError),
      )
    } finally {
      setBusyAction('')
    }
  }

  const refreshAfterApply = async () => {
    await Promise.all([
      reloadManagement(),
      reloadRuns(),
      onPricesChanged?.(),
    ])
  }

  const handleApplySelected = async () => {
    if (!preview || selectedItemIDs.length === 0) {
      setError('请先选择至少一条价格变化。')
      return
    }

    const confirmed = window.confirm(
      `确认应用已选择的 ${selectedItemIDs.length} 条价格变化？历史消费记录不会改变。`,
    )

    if (!confirmed) return

    setBusyAction('apply-selected')
    setMessage('')
    setError('')

    try {
      const result =
        await applySelectedPriceSyncChanges(
          preview.run.id,
          selectedItemIDs,
        )

      setPreview(result)
      setSelectedItemIDs([])
      setMessage(
        `价格应用完成：成功 ${result.summary.applied_count} 条，冲突跳过 ${result.summary.stale_count} 条。`,
      )

      await refreshAfterApply()
    } catch (requestError) {
      setError(
        priceSyncErrorMessage(requestError),
      )
    } finally {
      setBusyAction('')
    }
  }

  const handleApplyAll = async () => {
    if (!preview) return

    const updateCount =
      preview.items.filter(
        item => item.action === 'update',
      ).length

    if (updateCount === 0) {
      setError('当前预览没有可应用的价格变化。')
      return
    }

    const confirmed = window.confirm(
      `确认应用本批全部 ${updateCount} 条可信价格变化？历史消费记录不会改变。`,
    )

    if (!confirmed) return

    setBusyAction('apply-all')
    setMessage('')
    setError('')

    try {
      const result =
        await applyAllPriceSyncChanges(
          preview.run.id,
        )

      setPreview(result)
      setSelectedItemIDs([])
      setMessage(
        `全部价格应用完成：成功 ${result.summary.applied_count} 条，冲突跳过 ${result.summary.stale_count} 条。`,
      )

      await refreshAfterApply()
    } catch (requestError) {
      setError(
        priceSyncErrorMessage(requestError),
      )
    } finally {
      setBusyAction('')
    }
  }

  const handleOpenRun = async (
    runID: string,
  ) => {
    setBusyAction(`run:${runID}`)
    setMessage('')
    setError('')
    setSelectedItemIDs([])

    try {
      const result =
        await getPriceSyncRunDetail(runID)

      setPreview(result)
      setMessage(
        `已载入同步批次 ${runID}。`,
      )
    } catch (requestError) {
      setError(
        priceSyncErrorMessage(requestError),
      )
    } finally {
      setBusyAction('')
    }
  }

  if (loading) {
    return (
      <div
        style={{
          padding: '32px',
          textAlign: 'center',
          color: C.textMuted,
        }}
      >
        正在加载价格同步配置...
      </div>
    )
  }

  if (!management) {
    return (
      <div
        style={{
          padding: '20px',
          border: `1px solid ${C.red}`,
          borderRadius: '12px',
          color: C.red,
        }}
      >
        无法加载价格同步配置。
      </div>
    )
  }

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: '16px',
      }}
    >
      <div
        style={{
          padding: '16px 20px',
          borderRadius: '14px',
          border: `1px solid ${C.border}`,
          background: C.white,
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            flexWrap: 'wrap',
          }}
        >
          <span
            style={{
              fontSize: '17px',
              fontWeight: 700,
              color: C.text,
            }}
          >
            🔄 模型价格自动同步
          </span>

          <span
            style={{
              padding: '3px 10px',
              borderRadius: '999px',
              fontSize: '12px',
              background: management.settings.enabled
                ? 'rgba(16,185,129,0.10)'
                : C.bg,
              color: management.settings.enabled
                ? C.green
                : C.textMuted,
            }}
          >
            {management.settings.enabled
              ? '定时同步已开启'
              : '定时同步已关闭'}
          </span>

          <span
            style={{
              fontSize: '12px',
              color: C.textMuted,
            }}
          >
            文本 {management.text_targets.length} 个 ·
            图片/视频/TTS {management.media_targets.length} 个
          </span>
        </div>

        <div
          style={{
            marginTop: '8px',
            fontSize: '12px',
            lineHeight: 1.7,
            color: C.textSec,
          }}
        >
          价格检查只生成预览，不会直接修改正式价格。
          定时自动应用还需同时开启全局自动应用和单模型自动同步。
        </div>
      </div>

      {message && (
        <div
          style={{
            padding: '10px 14px',
            borderRadius: '10px',
            background: 'rgba(16,185,129,0.08)',
            border: `1px solid ${C.green}`,
            color: C.green,
            fontSize: '13px',
          }}
        >
          ✓ {message}
        </div>
      )}

      {error && (
        <div
          style={{
            padding: '10px 14px',
            borderRadius: '10px',
            background: 'rgba(239,68,68,0.06)',
            border: `1px solid ${C.red}`,
            color: C.red,
            fontSize: '13px',
          }}
        >
          ⚠ {error}
        </div>
      )}

      <PriceSyncSettingsCard
        settings={management.settings}
        saving={busyAction === 'settings'}
        onSave={handleSaveSettings}
      />

      <PriceSyncTargetsTable
        textTargets={management.text_targets}
        mediaTargets={management.media_targets}
        busyAction={busyAction}
        onSave={handleSaveTarget}
      />

      <PriceSyncPreviewPanel
        preview={preview}
        runs={runs}
        selectedItemIDs={selectedItemIDs}
        busyAction={busyAction}
        onSelectedItemIDsChange={
          setSelectedItemIDs
        }
        onPreview={handlePreview}
        onApplySelected={handleApplySelected}
        onApplyAll={handleApplyAll}
        onOpenRun={handleOpenRun}
      />
    </div>
  )
}

/**
 * CoursewareBatchIntegritySection.tsx — R-04 普通批量生成完整性入口
 *
 * 本组件负责把普通工作台与完整性Hook、教师视图卡片连接起来：
 *   - 只消费 run_kind=batch；
 *   - 补生与轮询状态由Hook维护；
 *   - 页面刷新通过 onPagesChanged 回传给现有工作台；
 *   - 不在浏览器自行判断 complete。
 */

import { useCallback, useState } from 'react'

import CoursewareGenerationIntegrityCard from './CoursewareGenerationIntegrityCard'
import { useCoursewareGenerationIntegrityState } from './useCoursewareGenerationIntegrityState'

interface Props {
  coursewareId?: string
  enabled?: boolean
  onPagesChanged?: () => void
}

export default function CoursewareBatchIntegritySection({
  coursewareId,
  enabled = true,
  onPagesChanged,
}: Props) {
  const [actionMessage, setActionMessage] = useState('')

  const {
    state,
    retrying,
    error,
    refresh,
    retryMissingPages,
  } = useCoursewareGenerationIntegrityState({
    coursewareId,
    enabled,
    onSettled: onPagesChanged,
  })

  const handleRetry = useCallback(async () => {
    setActionMessage('')

    try {
      await retryMissingPages()
      setActionMessage('✅ 已提交“只补生成缺失页”，系统会继续同步后台完整性状态。')
    } catch (retryError) {
      setActionMessage(
        `⚠️ ${
          retryError instanceof Error
            ? retryError.message
            : '补生成任务提交失败，请稍后重试。'
        }`,
      )
    }
  }, [retryMissingPages])

  const handleRefresh = useCallback(async () => {
    setActionMessage('')
    const next = await refresh()

    if (next) {
      onPagesChanged?.()
    }
  }, [onPagesChanged, refresh])

  const batchState = state?.run_kind === 'batch' ? state : null

  if (!batchState && !error && !actionMessage) {
    return null
  }

  return (
    <div style={{ marginBottom: 16 }}>
      {actionMessage && (
        <div
          style={{
            marginBottom: 10,
            padding: '10px 12px',
            borderRadius: 8,
            background: actionMessage.startsWith('✅') ? '#ECFDF5' : '#FFFBEB',
            border: `1px solid ${
              actionMessage.startsWith('✅') ? '#A7F3D0' : '#FDE68A'
            }`,
            color: actionMessage.startsWith('✅') ? '#047857' : '#92400E',
            fontSize: 12.5,
            lineHeight: 1.6,
          }}
        >
          {actionMessage}
        </div>
      )}

      {error && (
        <div
          style={{
            marginBottom: 10,
            padding: '10px 12px',
            borderRadius: 8,
            background: '#FFFBEB',
            border: '1px solid #FDE68A',
            color: '#92400E',
            fontSize: 12.5,
            lineHeight: 1.6,
          }}
        >
          ⚠️ {error}
        </div>
      )}

      <CoursewareGenerationIntegrityCard
        state={batchState}
        expectedRunKind="batch"
        busy={retrying}
        retryLabel="只补生成缺失页"
        onRetry={batchState ? handleRetry : undefined}
        onRefresh={batchState ? handleRefresh : undefined}
      />
    </div>
  )
}
